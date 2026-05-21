package helpers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	bsemver "github.com/blang/semver/v4"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"github.com/operator-framework/operator-registry/alpha/property"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// preferredPackages is tried in order across all serving catalogs; the first one
// found that satisfies the OLMv1 GA install requirements wins.
// Falls back to the first catalog package that satisfies the requirements.
//
// OLMv1 GA install requirements:
//   - AllNamespaces install mode is supported
//   - No dependencies (olm.gvk.required / olm.package.required)
//
// Verified against v4.22 redhat-operator-index:
//
//	quay-operator       AllNamespaces=true  deps=0
//	cluster-logging     AllNamespaces=true  deps=0
//	serverless-operator AllNamespaces=true  deps=0
//	logic-operator      AllNamespaces=true  deps=0
var preferredPackages = []string{
	"quay-operator",
	"cluster-logging",
	"serverless-operator",
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
// full catalog response to find the highest-priority preferred package that satisfies
// the OLMv1 GA install requirements (AllNamespaces=true, no dependencies).
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

	// Wait until at least one catalog is Serving before attempting package discovery.
	// Catalogs may briefly lag behind at cluster startup even after the Serving condition
	// has been written, so a short Eventually avoids a startup race.
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.List(ctx, catalogList)).To(Succeed(), "failed to list ClusterCatalogs")
		serving := 0
		for i := range catalogList.Items {
			if meta.IsStatusConditionPresentAndEqual(catalogList.Items[i].Status.Conditions, olmv1.TypeServing, metav1.ConditionTrue) {
				serving++
			}
		}
		g.Expect(serving).To(BeNumerically(">", 0), "no ClusterCatalogs are Serving yet")
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(DefaultPolling).Should(Succeed())

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
			fmt.Fprintf(GinkgoWriter, "Catalog %q: no suitable packages found\n", cc.Name)
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
// reads the full response, and returns the highest-ranked preferred package that
// satisfies the OLMv1 GA install requirements (AllNamespaces=true, no dependencies).
//
// A 404 response means the catalogd HTTP server has not yet indexed the catalog content
// even though the Kubernetes Serving condition is True; the call is retried with backoff
// up to catalogReadyTimeout before giving up.
//
// rank == math.MaxInt means the returned package is a fallback (not in wantedRank).
// Returns ("", 0, nil) when no suitable packages are found.
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

// scanPackages reads the full /api/v1/all JSONL response in a single pass.
// For each package it finds the highest-semver bundle across all channels
// (default channel is an OLMv0 concept that OLMv1 ignores) and validates
// it against the OLMv1 GA install requirements (AllNamespaces=true, no dependencies).
//
// Returns the highest-ranked preferred package that passes, or the first valid
// fallback (alphabetical), or ("", math.MaxInt, nil) when nothing suitable is found.
func scanPackages(resp *http.Response, wantedRank map[string]int) (string, int, error) {
	const maxCatalogLineBytes = 16 * 1024 * 1024
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), maxCatalogLineBytes)

	// Track the highest-semver bundle seen for each package across all channels.
	// property.Parse is called once per bundle here and the result stored so that
	// bundleVersion and checkBundleProps do not each call it again.
	type bundleRecord struct {
		version bsemver.Version
		parsed  *property.Properties
	}
	best := map[string]bundleRecord{} // pkg name → best bundle seen so far

	for scanner.Scan() {
		var obj struct {
			Schema     string              `json:"schema"`
			Name       string              `json:"name"`
			Package    string              `json:"package"`
			Properties []property.Property `json:"properties"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil || obj.Name == "" {
			continue
		}
		if obj.Schema != "olm.bundle" || obj.Package == "" {
			continue
		}
		parsed, err := property.Parse(obj.Properties)
		if err != nil {
			continue
		}
		ver, ok := bundleVersion(parsed)
		if !ok {
			continue
		}
		if existing, exists := best[obj.Package]; !exists || ver.GT(existing.version) {
			best[obj.Package] = bundleRecord{version: ver, parsed: parsed}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	// Sort package names so fallback selection is deterministic across runs.
	pkgNames := make([]string, 0, len(best))
	for pkg := range best {
		pkgNames = append(pkgNames, pkg)
	}
	sort.Strings(pkgNames)

	bestPkg, bestRank, fallback := "", math.MaxInt, ""
	for _, pkgName := range pkgNames {
		allNS, hasDeps := checkBundleProps(best[pkgName].parsed)
		if !allNS || hasDeps {
			continue // does not satisfy GA requirements
		}
		if fallback == "" {
			fallback = pkgName
		}
		if rank, ok := wantedRank[pkgName]; ok && (bestPkg == "" || rank < bestRank) {
			bestPkg, bestRank = pkgName, rank
		}
	}

	if bestPkg != "" {
		return bestPkg, bestRank, nil
	}
	return fallback, math.MaxInt, nil
}

// bundleVersion extracts the semver version from a pre-parsed bundle's olm.package property.
func bundleVersion(parsed *property.Properties) (bsemver.Version, bool) {
	if len(parsed.Packages) == 0 {
		return bsemver.Version{}, false
	}
	v, err := bsemver.Parse(parsed.Packages[0].Version)
	if err != nil {
		return bsemver.Version{}, false
	}
	return v, true
}

// checkBundleProps inspects a pre-parsed bundle's properties and returns whether
// AllNamespaces is supported and whether any dependencies are declared.
func checkBundleProps(parsed *property.Properties) (bool, bool) {
	hasDeps := len(parsed.GVKsRequired) > 0 || len(parsed.PackagesRequired) > 0
	allNamespaces := false
	for _, csvMeta := range parsed.CSVMetadatas {
		for _, mode := range csvMeta.InstallModes {
			if mode.Type == operatorsv1alpha1.InstallModeTypeAllNamespaces {
				allNamespaces = mode.Supported
			}
		}
	}
	return allNamespaces, hasDeps
}
