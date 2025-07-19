package test

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/test/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/test/extlogs"
)

const (
	olmv1GroupName = "olm.operatorframework.io"
)

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))

	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs", func() {
	It("should be installed", func(ctx SpecContext) {
		checkFeatureCapability(ctx)
		cfg := env.Get().RestCfg
		crds := []struct {
			group   string
			version []string
			plural  string
		}{
			{olmv1GroupName, []string{"v1"}, "clusterextensions"},
			{olmv1GroupName, []string{"v1"}, "clustercatalogs"},
		}

		client, err := apiextclient.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		for _, crd := range crds {
			By(fmt.Sprintf("verifying CRD %s.%s", crd.plural, crd.group))
			crdObj, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, fmt.Sprintf("%s.%s",
				crd.plural, crd.group), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			for _, v := range crd.version {
				found := false
				for _, ver := range crdObj.Spec.Versions {
					if ver.Name == v {
						Expect(ver.Served).To(BeTrue(), "version %s not served", v)
						Expect(ver.Storage).To(BeTrue(), "version %s not used for storage", v)
						found = true
						break
					}
				}
				if !found {
					extlogs.WarnContextf("Expected version not found for CRD %s.%s. Available: %#v",
						crd.plural,
						crd.group,
						crdObj.Spec.Versions)
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Expected version %q in CRD %s.%s", v,
					crd.plural, crd.group))
			}
		}
	})
})

func checkFeatureCapability(ctx context.Context) {
	if !env.Get().IsOpenShift {
		extlogs.Warn("Skipping feature capability check: not OpenShift")
		return
	}

	clientset, err := configclient.NewForConfig(env.Get().RestCfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create config client")

	cv, err := clientset.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve cluster version")

	for _, cap := range cv.Status.Capabilities.EnabledCapabilities {
		if cap == configv1.ClusterVersionCapabilityOperatorLifecycleManagerV1 {
			return
		}
	}

	extlogs.Warn("Skipping test: OperatorLifecycleManagerV1 capability not enabled")
	Skip("Test requires OperatorLifecycleManagerV1 capability")
}
