package specs

import (
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] cluster-olm-operator", g.Label("NonHyperShiftHOST"), func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("default")
	)

	g.BeforeEach(func() {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
	})

	g.It("PolarionID:80078-[OTP]Downstream feature gate promotion mechanics", g.Label("original-name:[sig-olmv1][Jira:OLM] cluster-olm-operator PolarionID:80078-Downstream feature gate promotion mechanics"), func() {
		args, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("deploy", "catalogd-controller-manager", "-o=jsonpath={.spec.template.spec.containers[0].args}", "-n", "openshift-catalogd").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if exutil.IsTechPreviewNoUpgrade(oc) {
			enabledFeatures, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("featuregate", "cluster", "-o=jsonpath={.status.featureGates[0].enabled}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			if !strings.Contains(enabledFeatures, "NewOLMCatalogdAPIV1Metas") {
				e2e.Failf("the NewOLMCatalogdAPIV1Metas feature wasn't enabled in the TP cluster: %v", enabledFeatures)
			}
			if !strings.Contains(args, "APIV1MetasHandler=true") {
				e2e.Failf("the APIV1MetasHandler argument wasn't enabled in the TP cluster: %v", args)
			}
		} else {
			if strings.Contains(args, "APIV1MetasHandler=true") {
				e2e.Failf("the APIV1MetasHandler argument enabled in the general cluster: %v", args)
			}
		}
	})

})
