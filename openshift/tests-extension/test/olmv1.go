package test

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/clusterinit"
)

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs", func() {
	It("should verify CRDs are served and stored", func(ctx context.Context) {
		dynClient, err := dynamic.NewForConfig(clusterinit.RESTConfig())
		Expect(err).NotTo(HaveOccurred())

		providedAPIs := []struct {
			group   string
			version string
			plural  string
		}{
			{"platform.operatorframework.io", "v1", "clusterextensions"},
			{"platform.operatorframework.io", "v1", "clustercatalogs"},
		}

		for _, api := range providedAPIs {
			name := fmt.Sprintf("%s.%s", api.plural, api.group)
			By(fmt.Sprintf("checking CRD %q is served/stored for version %q", name, api.version))

			crdGVR := schema.GroupVersionResource{
				Group:    "apiextensions.k8s.io",
				Version:  "v1",
				Resource: "customresourcedefinitions",
			}

			obj, err := dynClient.Resource(crdGVR).Get(ctx, name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			spec := obj.UnstructuredContent()["spec"].(map[string]interface{})
			versions := spec["versions"].([]interface{})
			var found bool
			for _, v := range versions {
				ver := v.(map[string]interface{})
				if ver["name"] == api.version &&
					ver["served"] == true &&
					ver["storage"] == true {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected CRD %s to have version %s served+stored", name, api.version)
		}
	})
})
