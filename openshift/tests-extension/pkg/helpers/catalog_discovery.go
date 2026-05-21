package helpers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// preferredPackages is tried in order; the first one present in any serving catalog wins.
// Falls back to the first available package if none of these are found.
var preferredPackages = []string{
	"quay-operator",
	"cluster-logging",
	"serverless-operator",
	"cli-manager",
	"logic-operator",
}

const (
	// catalogReadyTimeout is how long we wait for a catalog's HTTP content to become
	// available after its Kubernetes Serving condition is True. The catalogd controller
	// may set Serving=True slightly before the HTTP server has indexed the content.
	catalogReadyTimeout = 2 * time.Minute

	// catalogRetryInterval is the pause between 404 retries when waiting for a catalog.
	catalogRetryInterval = 5 * time.Second
)

// FindInstallablePackage searches all serving ClusterCatalogs for an installable package,
// favouring preferredPackages in order. Returns the catalog name and package name,
// or fails the test if no packages are found in any serving catalog.
//
// It queries /api/v1/all (always available, no feature gate required) and reads the
// full catalog response to find the highest-priority preferred package.
func FindInstallablePackage(ctx context.Context) (string, string) {
	cfg := env.Get().RestCfg
	httpClient, err := rest.HTTPClientFor(cfg)
	Expect(err).ToNot(HaveOccurred(), "failed to build HTTP client from REST config")

	k8sClient := env.Get().K8sClient
	catalogList := &olmv1.ClusterCatalogList{}
	Expect(k8sClient.List(ctx, catalogList)).To(Succeed(), "failed to list ClusterCatalogs")

	// Build a rank map: package name → index in preferredPackages (lower = higher priority).
	wantedRank := make(map[string]int, len(preferredPackages))
	for i, p := range preferredPackages {
		wantedRank[p] = i
	}

	bestCatalog, bestPkg, bestRank := "", "", math.MaxInt

	for i := range catalogList.Items {
		cc := &catalogList.Items[i]
		if !meta.IsStatusConditionPresentAndEqual(cc.Status.Conditions, olmv1.TypeServing, metav1.ConditionTrue) {
			fmt.Fprintf(GinkgoWriter, "Catalog %q is not serving, skipping\n", cc.Name)
			continue
		}

		pkg, rank, qErr := findPackageInCatalog(ctx, httpClient, cfg.Host, cc.Name, wantedRank)
		if qErr != nil {
			fmt.Fprintf(GinkgoWriter, "Warning: could not query catalog %q: %v\n", cc.Name, qErr)
			continue
		}
		if pkg == "" {
			fmt.Fprintf(GinkgoWriter, "Catalog %q: no packages found\n", cc.Name)
			continue
		}

		fmt.Fprintf(GinkgoWriter, "Catalog %q: found %q (rank %d)\n", cc.Name, pkg, rank)
		if bestPkg == "" || rank < bestRank {
			bestCatalog, bestPkg, bestRank = cc.Name, pkg, rank
		}
	}

	if bestPkg != "" {
		fmt.Fprintf(GinkgoWriter, "Selected package %q from catalog %q\n", bestPkg, bestCatalog)
		return bestCatalog, bestPkg
	}

	Fail("no installable packages found in any serving catalog")
	return "", "" // unreachable
}

// findPackageInCatalog queries the catalog's /api/v1/all endpoint (always available,
// unlike /api/v1/metas which requires the NewOLMCatalogdAPIV1Metas feature gate),
// reads the full response, filters for olm.package objects, and returns the
// highest-ranked preferred package found across the whole catalog.
//
// A 404 response means the catalogd HTTP server has not yet indexed the catalog content
// even though the Kubernetes Serving condition is True; the call is retried with backoff
// up to catalogReadyTimeout before giving up.
//
// rank == math.MaxInt means the returned package is a fallback (not in wantedRank).
// Returns ("", 0, nil) when no packages are found.
func findPackageInCatalog(ctx context.Context, httpClient *http.Client, apiServerHost, catalogName string, wantedRank map[string]int) (string, int, error) {
	url := strings.TrimRight(apiServerHost, "/") +
		fmt.Sprintf("/api/v1/namespaces/openshift-catalogd/services/https:catalogd-service:443/proxy/catalogs/%s/api/v1/all",
			catalogName)

	deadline := time.Now().Add(catalogReadyTimeout)
	for {
		resp, err := doGet(ctx, httpClient, url)
		if err != nil {
			return "", 0, err
		}

		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return scanPackages(resp, wantedRank)
		}

		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound || time.Now().After(deadline) {
			return "", 0, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
		}

		// 404: catalogd has not yet indexed this catalog's content; wait and retry.
		fmt.Fprintf(GinkgoWriter, "Catalog %q not yet indexed by catalogd (404), retrying in %s\n",
			catalogName, catalogRetryInterval)
		select {
		case <-ctx.Done():
			return "", 0, ctx.Err()
		case <-time.After(catalogRetryInterval):
		}
	}
}

func doGet(ctx context.Context, httpClient *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET: %w", err)
	}
	return resp, nil
}

func scanPackages(resp *http.Response, wantedRank map[string]int) (string, int, error) {
	// Response is JSONL: one JSON object per line. Only olm.package lines carry
	// a "name" field we care about; bundle/channel lines are much larger and skipped.
	var (
		fallbackPkg string
		bestPkg     string
		bestRank    = math.MaxInt
	)

	const maxCatalogLineBytes = 16 * 1024 * 1024
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), maxCatalogLineBytes)
	for scanner.Scan() {
		var obj struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil || obj.Schema != "olm.package" || obj.Name == "" {
			continue
		}
		if fallbackPkg == "" {
			fallbackPkg = obj.Name
		}
		if rank, ok := wantedRank[obj.Name]; ok && rank < bestRank {
			bestRank = rank
			bestPkg = obj.Name
		}
	}
	if sErr := scanner.Err(); sErr != nil {
		return "", 0, sErr
	}

	if bestPkg != "" {
		return bestPkg, bestRank, nil
	}
	return fallbackPkg, math.MaxInt, nil
}
