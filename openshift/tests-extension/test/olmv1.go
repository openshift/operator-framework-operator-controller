package test

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const olmv1GroupName = "olm.operatorframework.io"

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Serial] OLMv1 CRDs", func() {
	It("should be installed and versions served/stored", func(ctx SpecContext) {
		checkFeatureCapability(ctx)

		providedAPIs := []struct {
			group   string
			version []string
			plural  string
		}{
			{group: olmv1GroupName, version: []string{"v1"}, plural: "clusterextensions"},
			{group: olmv1GroupName, version: []string{"v1"}, plural: "clustercatalogs"},
		}

		for _, api := range providedAPIs {
			for _, ver := range api.version {
				crdName := fmt.Sprintf("%s.%s", api.plural, api.group)
				By(fmt.Sprintf("Validating CRD: %s version %s", crdName, ver))

				var crd apiextensionsv1.CustomResourceDefinition
				err := K8sClient.Get(ctx, client.ObjectKey{Name: crdName}, &crd)
				Expect(err).NotTo(HaveOccurred(), "expected CRD %s to exist", crdName)

				var found bool
				for _, v := range crd.Spec.Versions {
					if v.Name == ver {
						Expect(v.Served).To(BeTrue(), fmt.Sprintf("version %s not marked as served", ver))
						Expect(v.Storage).To(BeTrue(), fmt.Sprintf("version %s not marked as storage", ver))
						found = true
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("version %q not found in CRD %q", ver, crdName))
			}
		}
	})
})

func checkFeatureCapability(ctx context.Context) {
	clientset, err := configclient.NewForConfig(RestCfg)
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
