package commons

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/extlogs"
)

const (
	// GroupOLMv1 is the API group name used for OLMv1 custom resources.
	GroupOLMv1 = "olm.operatorframework.io"

	// CatalogAPIVersion is the version of API
	CatalogAPIVersion = "v1"

	// KindClusterExtension is the Kind name used for the ClusterExtension CRD in OLMv1.
	KindClusterExtension = "ClusterExtension"

	KindCatalog = "ClusterCatalog"

	// DefaultTimeout defines the maximum duration for Eventually assertions to succeed.
	DefaultTimeout = 5 * time.Minute

	// DefaultPollingInterval defines how frequently Eventually should poll the condition.
	DefaultPollingInterval = 1 * time.Second
)

// TestdataBaseDir is the absolute path to the testdata directory containing
// testdata for the tests in this package.
var TestdataBaseDir = resolveTestdataBaseDir()

func resolveTestdataBaseDir() string {
	dir, err := filepath.Abs("testdata")
	if err != nil {
		panic(fmt.Sprintf("failed to resolve absolute testdata base dir: %v", err))
	}
	return dir
}

// CheckFeatureCapability checks if the OpenShift cluster has the required
// OLMv1 capability enabled. If not, it skips the test with a message.
func CheckFeatureCapability() {
	if !env.Get().IsOpenShift {
		extlogs.Warn("Skipping feature capability check: not OpenShift")
		return
	}

	ctx := context.TODO()
	clientset, err := configclient.NewForConfig(env.Get().RestCfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create config client")

	cv, err := clientset.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve cluster version")

	for _, cap := range cv.Status.Capabilities.EnabledCapabilities {
		if cap == configv1.ClusterVersionCapabilityOperatorLifecycleManagerV1 {
			return
		}
	}
	Skip("Test requires OperatorLifecycleManagerV1 capability")
}

func ApplyResourceFile(packageName, version, unique, ceFile string) (func(), string) {
	k8sClient := env.Get().K8sClient
	ns := "default"

	if unique == "" {
		unique = rand.String(8)
	}

	By(fmt.Sprintf("applying cluster extension resources to namespace: %s", ns))
	raw, err := os.ReadFile(ceFile)
	Expect(err).NotTo(HaveOccurred())

	content := string(raw)
	replacements := map[string]string{
		"{NAMESPACE}":   ns,
		"{PACKAGENAME}": packageName,
		"{VERSION}":     version,
		"{UNIQUE}":      unique,
	}
	for k, v := range replacements {
		content = strings.ReplaceAll(content, k, v)
	}

	tmpDir := filepath.Join(TestdataBaseDir, "tmp")
	Expect(os.MkdirAll(tmpDir, 0755)).To(Succeed())
	outFile := filepath.Join(tmpDir, filepath.Base(ceFile)+"."+unique)
	Expect(os.WriteFile(outFile, []byte(content), 0600)).To(Succeed())

	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(content), 4096)
	var toDelete []client.Object

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err.Error() == "EOF" {
				break
			}
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(obj.GetKind()).NotTo(BeEmpty())
		Expect(obj.GetName()).NotTo(BeEmpty())

		toDelete = append(toDelete, obj.DeepCopy())
		err := k8sClient.Create(context.TODO(), obj)
		Expect(err == nil || apierrors.IsAlreadyExists(err)).To(BeTrue())
	}

	return func() {
		By("cleaning up applied resources")
		for _, obj := range toDelete {
			_ = k8sClient.Delete(context.TODO(), obj)
		}
		_ = os.Remove(outFile)
	}, unique
}
