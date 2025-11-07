package specs

import (
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] clustercatalog", g.Label("NonHyperShiftHOST"), func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLI("olmv1-opeco"+exutil.GetRandomString(), exutil.KubeConfigPath())
	)

	g.BeforeEach(func() {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
	})

	g.It("PolarionID:69242-[OTP][Skipped:Disconnected]Catalogd deprecated package bundlemetadata catalogmetadata from clustercatalog CR", g.Label("original-name:[sig-olmv1][Jira:OLM] clustercatalog PolarionID:69242-[Skipped:Disconnected]Catalogd deprecated package bundlemetadata catalogmetadata from clustercatalog CR"), func() {
		g.By("get the old related crd package/bundlemetadata/bundledeployment")
		crds, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("crd").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if strings.Contains(crds, "bundlemetadata") || strings.Contains(crds, "catalogmetadata") {
			o.Expect(strings.Contains(crds, "bundlemetadata")).NotTo(o.BeTrue())
			o.Expect(strings.Contains(crds, "catalogmetadata")).NotTo(o.BeTrue())
		} else {
			e2e.Logf("old related crd bundlemetadata/bundledeployment has been delete")
		}

	})

})
