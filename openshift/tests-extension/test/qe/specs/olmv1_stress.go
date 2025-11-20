package specs

import (
	"fmt"
	"path/filepath"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
	olmv1util "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util/olmv1util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] OLM v1 for stress", func() {

	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("default")
	)

	g.BeforeEach(func() {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
	})

	// author: kuiwang@redhat.com
	g.It("PolarionID:81509-[OTP][Skipped:Disconnected][OlmStress]olmv1 create mass operator to see if they all are installed successfully [Slow][Timeout:330m]", g.Label("StressTest"), g.Label("NonHyperShiftHOST"), func() {
		var (
			caseID                       = "81509"
			prefixCatalog                = "catalog-" + caseID
			prefixSa                     = "sa-" + caseID
			prefixCe                     = "ce-" + caseID
			prefixNs                     = "ns-" + caseID
			prefixPackage                = "stress-olmv1-c"
			prefixImage                  = "quay.io/olmqe/stress-index:vokv"
			nsOc                         = "openshift-operator-controller"
			nsCatalog                    = "openshift-catalogd"
			catalogLabel                 = "control-plane=catalogd-controller-manager"
			ocLabel                      = "control-plane=operator-controller-controller-manager"
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
		)

		if !olmv1util.IsPodReady(oc, nsCatalog, catalogLabel) {
			_, _ = olmv1util.Get(oc, "pod", "-n", nsCatalog, "-l", catalogLabel, "-o", "yaml")
			exutil.AssertWaitPollNoErr(fmt.Errorf("the pod with %s is not correct", catalogLabel), "the pod with app=catalog-operator is not correct")
		}
		if !olmv1util.IsPodReady(oc, nsOc, ocLabel) {
			_, _ = olmv1util.Get(oc, "pod", "-n", nsOc, "-l", ocLabel, "-o", "yaml")
			exutil.AssertWaitPollNoErr(fmt.Errorf("the pod with %s is not correct", ocLabel), "the pod with app=olm-operator is not correct")
		}

		startTime := time.Now().UTC()
		e2e.Logf("Start time: %s", startTime.Format(time.RFC3339))

		// for i := 0; i < 500; i++ {
		for i := 900; i < 969; i++ {
			// it is not enough with 330m for one case if we run 100 times
			e2e.Logf("=================it is round %v=================", i)
			ns := fmt.Sprintf("%s-%d", prefixNs, i)
			clustercatalog := olmv1util.ClusterCatalogDescription{
				Name:     fmt.Sprintf("%s-%d", prefixCatalog, i),
				Imageref: fmt.Sprintf("%s%d", prefixImage, i),
				Template: clustercatalogTemplate,
			}
			saCrb := olmv1util.SaCLusterRolebindingDescription{
				Name:      fmt.Sprintf("%s-%d", prefixSa, i),
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			ce := olmv1util.ClusterExtensionDescription{
				Name:             fmt.Sprintf("%s-%d", prefixCe, i),
				PackageName:      fmt.Sprintf("%s%d", prefixPackage, i),
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           fmt.Sprintf("%s-%d", prefixSa, i),
				Template:         clusterextensionTemplate,
			}
			g.By(fmt.Sprintf("Create namespace for %d", i))
			// defer oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
			// it take time delete ns which is not necessary. currently 5.5h is not enough to delete them.
			// so I prefer to keep ns to save case duration
			err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

			g.By(fmt.Sprintf("Create clustercatalog for %d", i))
			e2e.Logf("=========Create clustercatalog %v=========", clustercatalog.Name)
			defer clustercatalog.Delete(oc)
			err = clustercatalog.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			clustercatalog.WaitCatalogStatus(oc, "true", "Serving", 0)

			g.By(fmt.Sprintf("Create SA for clusterextension for %d", i))
			defer saCrb.Delete(oc)
			saCrb.Create(oc)

			g.By(fmt.Sprintf("check ce to be installed for %d", i))
			e2e.Logf("=========Create clusterextension %v=========", ce.Name)
			defer ce.Delete(oc)
			err = ce.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			ce.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 10, 600, 0)
			ce.WaitClusterExtensionCondition(oc, "Installed", "True", 0)
		}

		endTime := time.Now().UTC()
		e2e.Logf("End time:  %v", endTime.Format(time.RFC3339))

		duration := endTime.Sub(startTime)
		minutes := int(duration.Minutes())
		if minutes < 1 {
			minutes = 1
		}

		podName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", catalogLabel, "-o=jsonpath={.items[0].metadata.name}", "-n", nsCatalog).Output()
		if err == nil {
			if !olmv1util.WriteErrToArtifactDir(oc, nsCatalog, podName, "error", "Unhandled|Reconciler error|level=info", caseID, minutes) {
				e2e.Logf("no error log into artifact for pod %s in %s", podName, nsCatalog)
			}
		}
		podName, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", ocLabel, "-o=jsonpath={.items[0].metadata.name}", "-n", nsOc).Output()
		if err == nil {
			if !olmv1util.WriteErrToArtifactDir(oc, nsOc, podName, "error", "Unhandled|Reconciler error|level=info", caseID, minutes) {
				e2e.Logf("no error log into artifact for pod %s in %s", podName, nsOc)
			}
		}

		if !olmv1util.IsPodReady(oc, nsCatalog, catalogLabel) {
			_, _ = olmv1util.Get(oc, "pod", "-n", nsCatalog, "-l", catalogLabel, "-o", "yaml")
			exutil.AssertWaitPollNoErr(fmt.Errorf("the pod with %s is not correct", catalogLabel), "the pod with app=catalog-operator is not correct")
		}
		if !olmv1util.IsPodReady(oc, nsOc, ocLabel) {
			_, _ = olmv1util.Get(oc, "pod", "-n", nsOc, "-l", ocLabel, "-o", "yaml")
			exutil.AssertWaitPollNoErr(fmt.Errorf("the pod with %s is not correct", ocLabel), "the pod with app=olm-operator is not correct")
		}

	})

})
