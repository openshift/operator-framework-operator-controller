package specs

import (
	"context"
	"fmt"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/util/wait"
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

	g.It("PolarionID:78393-[OTP][Skipped:Disconnected]support metrics", func() {
		exutil.SkipOnProxyCluster(oc)

		var metricsMsg string
		g.By("get catalogd metrics")
		promeEp, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("service", "-n", "openshift-catalogd", "catalogd-service", "-o=jsonpath={.spec.clusterIP}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(promeEp).NotTo(o.BeEmpty())
		if strings.Count(promeEp, ":") >= 2 {
			g.Skip("Skip for IPv6.")
		}
		queryContent := "https://" + promeEp + ":7443/metrics"

		g.By("Get token")
		metricsToken, err := exutil.GetSAToken(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(metricsToken).NotTo(o.BeEmpty())

		wrongToken, err := oc.AsAdmin().WithoutNamespace().Run("create").Args("token", "openshift-state-metrics", "-n", "openshift-monitoring").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(wrongToken).NotTo(o.BeEmpty())

		g.By("Get metrics")
		podnameStr, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", "openshift-monitoring", "-l", "prometheus==k8s", "-o=jsonpath='{..metadata.name}'").Output()
		o.Expect(podnameStr).NotTo(o.BeEmpty())
		prometheusPodname := strings.Split(strings.Trim(podnameStr, "'"), " ")[0]

		errWait := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			metricsMsg, err := oc.AsAdmin().NotShowInfo().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", prometheusPodname, "-i", "--", "curl", "-k", "-H", fmt.Sprintf("Authorization: Bearer %v", metricsToken), queryContent).Output()
			e2e.Logf("err:%v", err)
			if strings.Contains(metricsMsg, "catalogd_http_request_duration_seconds_bucket{code=\"200\"") {
				e2e.Logf("found catalogd_http_request_duration_seconds_bucket{code=\"200\"")
				return true, nil
			}
			return false, nil
		})
		if errWait != nil {
			e2e.Logf("metricsMsg:%v", metricsMsg)
			exutil.AssertWaitPollNoErr(errWait, "catalogd_http_request_duration_seconds_bucket{code=\"200\" not found.")
		}

		g.By("ClusterRole/openshift-state-metrics has no rule to get the catalogd metrics")
		metricsMsg, _ = oc.AsAdmin().NotShowInfo().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", prometheusPodname, "-i", "--", "curl", "-k", "-H", fmt.Sprintf("Authorization: Bearer %v", wrongToken), queryContent).Output()
		o.Expect(metricsMsg).To(o.ContainSubstring("Authorization denied"))

		g.By("get operator-controller metrics")
		promeEp, err = oc.WithoutNamespace().AsAdmin().Run("get").Args("service", "-n", "openshift-operator-controller", "operator-controller-service", "-o=jsonpath={.spec.clusterIP}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(promeEp).NotTo(o.BeEmpty())
		queryContent = "https://" + promeEp + ":8443/metrics"

		errWait = wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			metricsMsg, err := oc.AsAdmin().NotShowInfo().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", prometheusPodname, "-i", "--", "curl", "-k", "-H", fmt.Sprintf("Authorization: Bearer %v", metricsToken), queryContent).Output()
			e2e.Logf("err:%v", err)
			if strings.Contains(metricsMsg, "controller_runtime_active_workers") {
				e2e.Logf("found controller_runtime_active_workers")
				return true, nil
			}
			return false, nil
		})
		if errWait != nil {
			e2e.Logf("metricsMsg:%v", metricsMsg)
			exutil.AssertWaitPollNoErr(errWait, "controller_runtime_active_workers not found.")
		}

		g.By("ClusterRole/openshift-state-metrics has no rule to get the operator-controller metrics")
		metricsMsg, _ = oc.AsAdmin().NotShowInfo().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", prometheusPodname, "-i", "--", "curl", "-k", "-H", fmt.Sprintf("Authorization: Bearer %v", wrongToken), queryContent).Output()
		o.Expect(metricsMsg).To(o.ContainSubstring("Authorization denied"))

	})

	g.It("PolarionID:79770-[OTP][Level0]metrics are collected by default", func() {
		podnameStr, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", "openshift-monitoring", "-l", "prometheus==k8s", "-o=jsonpath='{..metadata.name}'").Output()
		o.Expect(podnameStr).NotTo(o.BeEmpty())
		k8sPodname := strings.Split(strings.Trim(podnameStr, "'"), " ")[0]

		g.By("1) check status of Metrics targets is up")
		targetsUrl := "http://localhost:9090/api/v1/targets"
		targetsContent, _ := oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", k8sPodname, "--", "curl", "-s", targetsUrl).Output()
		status := gjson.Get(targetsContent, `data.activeTargets.#(labels.namespace=="openshift-catalogd").health`).String()
		if strings.Compare(status, "up") != 0 {
			statusAll := gjson.Get(targetsContent, `data.activeTargets.#(labels.namespace=="openshift-catalogd")`).String()
			e2e.Logf("catalogd target status: %s", statusAll)
			o.Expect(status).To(o.Equal("up"))
		}
		status = gjson.Get(targetsContent, `data.activeTargets.#(labels.namespace=="openshift-operator-controller").health`).String()
		if strings.Compare(status, "up") != 0 {
			statusAll := gjson.Get(targetsContent, `data.activeTargets.#(labels.namespace=="openshift-operator-controller")`).String()
			e2e.Logf("operator-controller target status: %s", statusAll)
			o.Expect(status).To(o.Equal("up"))
		}

		g.By("2) check metrics are collected")
		queryUrl := "http://localhost:9090/api/v1/query"
		query1 := `query=rest_client_requests_total{namespace="openshift-catalogd"}`
		queryResult1, _ := oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", k8sPodname, "--", "curl", "-G", "--data-urlencode", query1, queryUrl).Output()
		e2e.Logf("catalogd rest_client_requests_total query result: %s", queryResult1)
		o.Expect(queryResult1).To(o.ContainSubstring("value"))

		query2 := `query=rest_client_requests_total{namespace="openshift-operator-controller"}`
		queryResult2, _ := oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", k8sPodname, "--", "curl", "-G", "--data-urlencode", query2, queryUrl).Output()
		e2e.Logf("operator-controller rest_client_requests_total query result: %s", queryResult2)
		o.Expect(queryResult2).To(o.ContainSubstring("value"))

		g.By("3) test SUCCESS")
	})

	g.It("PolarionID:75877-[OTP]Make sure that rukpak is removed from payload", func() {
		g.By("1) Ensure bundledeployments.core.rukpak.io CRD is not installed")
		_, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("crd", "bundledeployments.core.rukpak.io").Output()
		o.Expect(err).To(o.HaveOccurred())

		g.By("2) Ensure openshift-rukpak namespace is not created")
		_, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("ns", "openshift-rukpak").Output()
		o.Expect(err).To(o.HaveOccurred())
	})

})
