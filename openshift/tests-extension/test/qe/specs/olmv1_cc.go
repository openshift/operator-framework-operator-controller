package specs

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
	olmv1util "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util/olmv1util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] clustercatalog", g.Label("NonHyperShiftHOST", "ClusterCatalog"), func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("default")
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

	g.It("PolarionID:80458-[OTP][Level0][Skipped:Disconnected]clustercatalog get x509 error since it cannot get the custom CA automatically [Serial]", g.Label("original-name:[sig-olmv1][Jira:OLM] clustercatalog PolarionID:80458-[Skipped:Disconnected]clustercatalog get x509 error since it cannot get the custom CA automatically [Serial]"), func() {
		g.By("1) create a random namespace")
		oc.SetupProject()
		g.By("2) create an image registry")
		err := oc.WithoutNamespace().Run("new-app").Args("--image", "quay.io/openshifttest/registry@sha256:1106aedc1b2e386520bc2fb797d9a7af47d651db31d8e7ab472f2352da37d1b3", "-n", oc.Namespace(), "REGISTRY_STORAGE_DELETE_ENABLED=true", "--import-mode=PreserveOriginal").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		registryPodsList, err := exutil.WaitForPods(
			oc.AdminKubeClient().CoreV1().Pods(oc.Namespace()),
			exutil.ParseLabelsOrDie("deployment=registry"),
			exutil.CheckPodIsReady, 1, 180000000000)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Get registry pods: %v", registryPodsList)

		_, err = oc.WithoutNamespace().Run("create").Args("route", "edge", "my-route", "--service=registry", "-n", oc.Namespace()).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		hostName, err := oc.WithoutNamespace().Run("get").Args("route", "my-route", "-o=jsonpath={.spec.host}", "-n", oc.Namespace()).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		_, err = oc.WithoutNamespace().Run("set").Args("volume", "deploy", "registry", "--add", "-t", "pvc", "--claim-size=30G", "-m", "/var/lib/registry", "--overwrite", "-n", oc.Namespace()).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		_, err = oc.AsAdmin().WithoutNamespace().Run("extract").Args("secret/router-ca", "-n", "openshift-ingress-operator", "--to=/tmp", "--confirm").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		defer func() {
			err = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-n", "openshift-config", "configmap", "trusted-ca-80458").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
		}()
		_, err = oc.AsAdmin().WithoutNamespace().Run("create").Args("-n", "openshift-config", "configmap", "trusted-ca-80458", fmt.Sprintf("--from-file=%s=/tmp/tls.crt", hostName)).Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		defer func() {
			if err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("image.config.openshift.io/cluster", "-p", fmt.Sprintf("{\"spec\": {\"additionalTrustedCA\": {\"name\": \"%s\"}}}", ""), "--type=merge").Execute(); err != nil {
				e2e.Failf("unpatch image.config.openshift.io/cluster failed:%v", err)
			}
		}()
		if err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("image.config.openshift.io/cluster", "-p", fmt.Sprintf("{\"spec\": {\"additionalTrustedCA\": {\"name\": \"%s\"}}}", "trusted-ca-80458"), "--type=merge").Execute(); err != nil {
			e2e.Failf("patch image.config.openshift.io/cluster failed:%v", err)
		}

		g.By("2.1) Wait for CA bundle to propagate to catalogd")
		// Wait for cluster-network-operator to sync CA from trusted-ca-80458 to catalogd-trusted-ca-bundle
		// This happens automatically without pod restart
		configTime := time.Now()
		errWait := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 3*time.Minute, false, func(ctx context.Context) (bool, error) {
			cm, err := oc.AdminKubeClient().CoreV1().ConfigMaps("openshift-catalogd").Get(ctx, "catalogd-trusted-ca-bundle", metav1.GetOptions{})
			if err != nil {
				e2e.Logf("Failed to get catalogd-trusted-ca-bundle ConfigMap: %v, retrying...", err)
				return false, nil
			}
			// Check if ConfigMap was updated after we configured additionalTrustedCA
			if cm.CreationTimestamp.After(configTime) {
				e2e.Logf("catalogd-trusted-ca-bundle ConfigMap was created at %v (after CA config)", cm.CreationTimestamp)
				return true, nil
			}
			// For existing ConfigMap, check if it has data (indicating it was updated)
			if len(cm.Data) > 0 {
				e2e.Logf("catalogd-trusted-ca-bundle ConfigMap has CA data, CA bundle synced")
				return true, nil
			}
			e2e.Logf("Waiting for catalogd-trusted-ca-bundle ConfigMap to be synced...")
			return false, nil
		})
		if errWait != nil {
			e2e.Failf("Timeout waiting for CA bundle to propagate to catalogd ConfigMap: %v", errWait)
		}

		g.By("3) create a ClusterCatalog")
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")

			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-80458",
				Imageref: fmt.Sprintf("%s/redhat/redhat-operator-index:v4.17", hostName),
				Template: clustercatalogTemplate,
			}
		)
		defer clustercatalog.Delete(oc)
		_ = clustercatalog.CreateWithoutCheck(oc)
		// it should retrun error message: source catalog content: error creating image source:
		// reading manifest v4.17 in my-route-e2e-test-default-gt5wh.apps.xiyuan-19b.qe.devcluster.openshift.com/redhat/redhat-operator-index: manifest unknown
		clustercatalog.CheckClusterCatalogCondition(oc, "Progressing", "message", "manifest unknown", 5, 180, 0)
	})

	g.It("PolarionID:77413-[OTP][Level0][Skipped:Disconnected]Check if ClusterCatalog is in Serving properly", func() {
		g.By("Verify built-in ClusterCatalogs report Serving=True")
		checks := []olmv1util.CheckDescription{
			olmv1util.NewCheck("expect", exutil.AsAdmin, exutil.WithoutNamespace, exutil.Contain, "True", exutil.Ok,
				[]string{"clustercatalog", "openshift-certified-operators", `-o=jsonpath={.status.conditions[?(@.type=="Serving")].status}`}),
			olmv1util.NewCheck("expect", exutil.AsAdmin, exutil.WithoutNamespace, exutil.Contain, "True", exutil.Ok,
				[]string{"clustercatalog", "openshift-community-operators", `-o=jsonpath={.status.conditions[?(@.type=="Serving")].status}`}),
			olmv1util.NewCheck("expect", exutil.AsAdmin, exutil.WithoutNamespace, exutil.Contain, "True", exutil.Ok,
				[]string{"clustercatalog", "openshift-redhat-operators", `-o=jsonpath={.status.conditions[?(@.type=="Serving")].status}`}),
		}
		for _, check := range checks {
			check.Check(oc)
		}
	})

	g.It("PolarionID:69123-[OTP][Skipped:Disconnected]Catalogd clustercatalog offer the operator content through http server", func() {
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")
			clustercatalog         = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-69123",
				Imageref: "quay.io/olmqe/olmtest-operator-index:nginxolm69123",
				Template: clustercatalogTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("get the index content through http service on cluster")
		unmarshalContent, err := clustercatalog.UnmarshalContent(oc, "all")
		o.Expect(err).NotTo(o.HaveOccurred())

		allPackageName := olmv1util.ListPackagesName(unmarshalContent.Packages)
		o.Expect(allPackageName[0]).To(o.ContainSubstring("nginx69123"))

		channelData := olmv1util.GetChannelByPakcage(unmarshalContent.Channels, "nginx69123")
		o.Expect(channelData[0].Name).To(o.ContainSubstring("candidate-v0.0"))

		bundlesName := olmv1util.GetBundlesNameByPakcage(unmarshalContent.Bundles, "nginx69123")
		o.Expect(bundlesName[0]).To(o.ContainSubstring("nginx69123.v0.0.1"))

	})

	g.It("PolarionID:69069-[OTP][Skipped:Disconnected]Replace pod-based image unpacker with an image registry client", func() {
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")
			clustercatalog         = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-69069",
				Imageref: "quay.io/olmqe/olmtest-operator-index:nginxolm69069",
				Template: clustercatalogTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		initresolvedRef, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("clustercatalog", clustercatalog.Name, "-o=jsonpath={.status.resolvedSource.image.ref}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Update the index image with different tag , but the same digestID")
		err = oc.AsAdmin().Run("patch").Args("clustercatalog", clustercatalog.Name, "-p", `{"spec":{"source":{"image":{"ref":"quay.io/olmqe/olmtest-operator-index:nginxolm69069v1"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Check the image is updated without wait but the resolvedSource is still the same and won't unpack again")
		jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].status}`, "Serving")
		statusOutput, err := olmv1util.GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o", jsonpath)
		o.Expect(err).NotTo(o.HaveOccurred())
		if !strings.Contains(statusOutput, "True") {
			e2e.Failf("status is %v, not Serving", statusOutput)
		}
		errWait := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			img, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("clustercatalog", clustercatalog.Name, "-o=jsonpath={.status.resolvedSource.image.ref}").Output()
			if err != nil {
				return false, err
			}
			if strings.Contains(img, initresolvedRef) {
				return true, nil
			}
			e2e.Logf("diff image1: %v, but expect same", img)
			return false, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o=jsonpath-as-json={.status}")
		}
		exutil.AssertWaitPollNoErr(errWait, "disgest is not same, but should be same")

		g.By("Update the index image with different tag and digestID")
		err = oc.AsAdmin().Run("patch").Args("clustercatalog", clustercatalog.Name, "-p", `{"spec":{"source":{"image":{"ref":"quay.io/olmqe/olmtest-operator-index:nginxolm69069v2"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		errWait = wait.PollUntilContextTimeout(context.TODO(), 30*time.Second, 90*time.Second, false, func(ctx context.Context) (bool, error) {
			img, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("clustercatalog", clustercatalog.Name, "-o=jsonpath={.status.resolvedSource.image.ref}").Output()
			if err != nil {
				return false, err
			}
			if strings.Contains(img, initresolvedRef) {
				e2e.Logf("same image, but expect not same")
				return false, nil
			}
			e2e.Logf("image2: %v", img)
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o=jsonpath-as-json={.status}")
		}
		exutil.AssertWaitPollNoErr(errWait, "digest is same, but should be not same")

	})

	g.It("PolarionID:69869-[OTP][Skipped:Disconnected]Catalogd Add metrics to the Storage implementation", func() {
		exutil.SkipOnProxyCluster(oc)
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")
			clustercatalog         = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-69869",
				Imageref: "quay.io/olmqe/olmtest-operator-index:nginxolm69869",
				Template: clustercatalogTemplate,
			}
			metricsMsg string
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Get http content")
		packageDataOut, err := clustercatalog.UnmarshalContent(oc, "package")
		o.Expect(err).NotTo(o.HaveOccurred())
		packageName := olmv1util.ListPackagesName(packageDataOut.Packages)
		o.Expect(packageName[0]).To(o.ContainSubstring("nginx69869"))

		g.By("Get token and clusterIP")
		promeEp, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("service", "-n", "openshift-catalogd", "catalogd-service", "-o=jsonpath={.spec.clusterIP}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(promeEp).NotTo(o.BeEmpty())

		metricsToken, err := exutil.GetSAToken(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(metricsToken).NotTo(o.BeEmpty())

		podnameStr, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", "openshift-monitoring", "-l", "prometheus==k8s", "-o=jsonpath='{..metadata.name}'").Output()
		o.Expect(podnameStr).NotTo(o.BeEmpty())
		prometheusPodname := strings.Split(strings.Trim(podnameStr, "'"), " ")[0]

		errWait := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			queryContent := "https://" + promeEp + ":7443/metrics"
			metricsMsg, err = oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", prometheusPodname, "-i", "--", "curl", "-k", "-H", fmt.Sprintf("Authorization: Bearer %v", metricsToken), queryContent).Output()
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

	})

	g.It("PolarionID:69202-[OTP][Skipped:Disconnected][Skipped:Proxy]Catalogd clustercatalog offer the operator content through http server off cluster", func() {
		exutil.SkipOnProxyCluster(oc)
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")
			clustercatalog         = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-69202",
				Imageref: "quay.io/olmqe/olmtest-operator-index:nginxolm69202",
				Template: clustercatalogTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("get the index content through http service off cluster")
		errWait := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, 100*time.Second, false, func(ctx context.Context) (bool, error) {
			// #nosec G204 -- ContentURL is obtained from trusted ClusterCatalog Kubernetes resource
			checkOutput, err := exec.Command("bash", "-c", "curl -k "+clustercatalog.ContentURL).Output()
			if err != nil {
				e2e.Logf("failed to execute the curl: %s. Trying again", err)
				return false, nil
			}
			if matched, _ := regexp.MatchString("nginx69202", string(checkOutput)); matched {
				e2e.Logf("Check the content off cluster success\n")
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "Cannot get the result")
	})

	g.It("PolarionID:73219-[OTP][Skipped:Disconnected]Fetch deprecation data from the catalogd http server", func() {
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")
			clustercatalog         = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-73219",
				Imageref: "quay.io/olmqe/olmtest-operator-index:nginxolm73219",
				Template: clustercatalogTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("get the deprecation content through http service on cluster")
		unmarshalContent, err := clustercatalog.UnmarshalContent(oc, "deprecations")
		o.Expect(err).NotTo(o.HaveOccurred())

		deprecatedChannel := olmv1util.GetDeprecatedChannelNameByPakcage(unmarshalContent.Deprecations, "nginx73219")
		o.Expect(deprecatedChannel[0]).To(o.ContainSubstring("candidate-v0.0"))

		deprecatedBundle := olmv1util.GetDeprecatedBundlesNameByPakcage(unmarshalContent.Deprecations, "nginx73219")
		o.Expect(deprecatedBundle[0]).To(o.ContainSubstring("nginx73219.v0.0.1"))

	})

	g.It("PolarionID:75441-[OTP][Skipped:Disconnected]Catalogd supports compression and jsonlines format", func() {
		var (
			baseDir                = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate = filepath.Join(baseDir, "clustercatalog.yaml")
			clustercatalog         = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-75441",
				Imageref: "quay.io/openshifttest/nginxolm-operator-index:nginxolm75441",
				Template: clustercatalogTemplate,
			}
			clustercatalog1 = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-75441v2",
				Imageref: "quay.io/openshifttest/nginxolm-operator-index:nginxolm75441v2",
				Template: clustercatalogTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		defer clustercatalog1.Delete(oc)
		clustercatalog1.Create(oc)

		g.By("Get the gzip response")
		url1 := clustercatalog.ContentURL

		g.By("Check the url response of clustercatalog-75441")
		getCmd := fmt.Sprintf("curl -ki %s -H \"Accept-Encoding: gzip\" --output -", url1)
		// #nosec G204 -- url1 (ContentURL) is obtained from trusted ClusterCatalog Kubernetes resource
		stringMessage, err := exec.Command("bash", "-c", getCmd).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if !strings.Contains(strings.ToLower(string(stringMessage)), "content-encoding: gzip") {
			e2e.Logf("response is %s", string(stringMessage))
			e2e.Failf("string Content-Encoding: gzip not in the output")
		}
		if !strings.Contains(strings.ToLower(string(stringMessage)), "content-type: application/jsonl") {
			e2e.Logf("response is %s", string(stringMessage))
			e2e.Failf("string Content-Type: application/jsonl not in the output")
		}
		g.By("Check the url response of clustercatalog-75441v2")
		url2 := clustercatalog1.ContentURL
		getCmd2 := fmt.Sprintf("curl -ki %s -H \"Accept-Encoding: gzip\"", url2)
		// #nosec G204 -- url2 (ContentURL) is obtained from trusted ClusterCatalog Kubernetes resource
		stringMessage2, err := exec.Command("bash", "-c", getCmd2).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(strings.ToLower(string(stringMessage2))).NotTo(o.ContainSubstring("content-encoding: gzip"))
		o.Expect(strings.ToLower(string(stringMessage2))).To(o.ContainSubstring("content-type: application/jsonl"))

	})

	g.It("PolarionID:73289-[OTP][Skipped:Disconnected]Check the deprecation conditions and messages", func() {
		var (
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-73289"
			sa                           = "sa73289"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-73289",
				Imageref: "quay.io/olmqe/olmtest-operator-index:nginxolm73289",
				Template: clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-73289",
				InstallNamespace: ns,
				PackageName:      "nginx73289v1",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create clusterextension with channel candidate-v1.0, version 1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		// Test BundleDeprecated
		g.By("Check BundleDeprecated status")
		clusterextension.WaitClusterExtensionCondition(oc, "Deprecated", "True", 0)
		clusterextension.WaitClusterExtensionCondition(oc, "BundleDeprecated", "True", 0)

		g.By("Check BundleDeprecated message info")
		message := clusterextension.GetClusterExtensionMessage(oc, "BundleDeprecated")
		if !strings.Contains(message, "nginx73289v1.v1.0.1 is deprecated. Uninstall and install v1.0.3 for support.") {
			e2e.Failf("Info does not meet expectations, message :%v", message)
		}

		g.By("update version to be >=1.0.2")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version": ">=1.0.2"}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			installedBundle, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.install.bundle.name}")
			if !strings.Contains(installedBundle, "v1.0.3") {
				e2e.Logf("clusterextension.InstalledBundle is %s, not v1.0.3, and try next", installedBundle)
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "clusterextension resolvedBundle is not v1.0.3")
		}

		g.By("Check if BundleDeprecated status and messages still exist")
		clusterextension.WaitClusterExtensionCondition(oc, "Deprecated", "False", 0)
		clusterextension.WaitClusterExtensionCondition(oc, "BundleDeprecated", "False", 0)
		message = clusterextension.GetClusterExtensionMessage(oc, "BundleDeprecated")
		if strings.Contains(message, "nginx73289v1.v1.0.1 is deprecated. Uninstall and install v1.0.3 for support.") {
			e2e.Failf("BundleDeprecated message still exists :%v", message)
		}
		clusterextension.Delete(oc)
		g.By("BundleDeprecated test done")

		// Test ChannelDeprecated
		g.By("update channel to candidate-v3.0")
		clusterextension.PackageName = "nginx73289v2"
		clusterextension.Channel = "candidate-v3.0"
		clusterextension.Version = ">=1.0.0"
		clusterextension.Template = clusterextensionTemplate

		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v3.0.1"))

		g.By("Check ChannelDeprecated status and message")
		clusterextension.WaitClusterExtensionCondition(oc, "Deprecated", "True", 0)
		clusterextension.WaitClusterExtensionCondition(oc, "ChannelDeprecated", "True", 0)
		message = clusterextension.GetClusterExtensionMessage(oc, "ChannelDeprecated")
		if !strings.Contains(message, "The 'candidate-v3.0' channel is no longer supported. Please switch to the 'candidate-v3.1' channel.") {
			e2e.Failf("Info does not meet expectations, message :%v", message)
		}

		g.By("update channel to candidate-v3.1")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v3.1"]}}}}`)

		g.By("Check if ChannelDeprecated status and messages still exist")
		clusterextension.WaitClusterExtensionCondition(oc, "Deprecated", "False", 0)
		clusterextension.WaitClusterExtensionCondition(oc, "ChannelDeprecated", "False", 0)
		message = clusterextension.GetClusterExtensionMessage(oc, "ChannelDeprecated")
		if strings.Contains(message, "The 'candidate-v3.0' channel is no longer supported. Please switch to the 'candidate-v3.1' channel.") {
			e2e.Failf("ChannelDeprecated message still exists :%v", message)
		}
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 3, 150, 0)
		clusterextension.WaitClusterExtensionCondition(oc, "Installed", "True", 0)
		clusterextension.Delete(oc)
		g.By("ChannelDeprecated test done")

		// Test PackageDeprecated
		g.By("update Package to 73289v3")
		clusterextension.PackageName = "nginx73289v3"
		clusterextension.Channel = "candidate-v1.0"
		clusterextension.Version = ">=1.0.0"
		clusterextension.Template = clusterextensionTemplate
		clusterextension.Create(oc)

		g.By("Check PackageDeprecated status and message")
		clusterextension.WaitClusterExtensionCondition(oc, "Deprecated", "True", 0)
		clusterextension.WaitClusterExtensionCondition(oc, "PackageDeprecated", "True", 0)
		message = clusterextension.GetClusterExtensionMessage(oc, "PackageDeprecated")
		if !strings.Contains(message, "The nginx73289v3 package is end of life. Please use the another package for support.") {
			e2e.Failf("Info does not meet expectations, message :%v", message)
		}
		g.By("PackageDeprecated test done")

	})

	g.It("PolarionID:74948-[OTP][Skipped:Disconnected]catalog offer the operator content through https server", func() {
		var (
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-74948"
			sa                           = "sa74948"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-74948",
				Imageref: "quay.io/openshifttest/nginxolm-operator-index:nginxolm74948",
				Template: clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-74948",
				InstallNamespace: ns,
				PackageName:      "nginx74948",
				Channel:          "candidate-v1.0",
				Version:          "1.0.3",
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Examine the service to confirm that the annotations are present")
		describe, err := oc.WithoutNamespace().AsAdmin().Run("describe").Args("service", "catalogd-service", "-n", "openshift-catalogd").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(describe).To(o.ContainSubstring("service.beta.openshift.io/serving-cert-secret-name: catalogserver-cert"))

		g.By("Ensure that the service CA bundle has been injected")
		crt, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("configmap", "openshift-service-ca.crt", "-n", "openshift-catalogd", "-o", "jsonpath={.metadata.annotations}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(crt).To(o.ContainSubstring("{\"service.beta.openshift.io/inject-cabundle\":\"true\"}"))

		g.By("Check secret data tls.crt tls.key")
		secretData, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("secret", "catalogserver-cert", "-n", "openshift-catalogd", "-o", "jsonpath={.data}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if !strings.Contains(secretData, "tls.crt") || !strings.Contains(secretData, "tls.key") {
			e2e.Failf("secret data not found")
		}

		g.By("Get the index content through https service on cluster")
		unmarshalContent, err := clustercatalog.UnmarshalContent(oc, "all")
		o.Expect(err).NotTo(o.HaveOccurred())

		allPackageName := olmv1util.ListPackagesName(unmarshalContent.Packages)
		o.Expect(allPackageName[0]).To(o.ContainSubstring("nginx74948"))

		g.By("Create clusterextension to verify operator-controller has been started, appropriately loaded the CA certs")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.3"))

	})

	g.It("PolarionID:74978-[OTP][Level0][Skipped:Disconnected]CRD upgrade will be prevented if the Scope is switched between Namespaced and Cluster", func() {
		var (
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-74978"
			sa                           = "sa74978"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-74978",
				Imageref: "quay.io/openshifttest/nginxolm-operator-index:nginxolm74978",
				Template: clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-74978",
				InstallNamespace: ns,
				PackageName:      "nginx74978",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("Update the version to 1.0.2, check changed from Namespaced to Cluster")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `scope changed`, 10, 60, 0)
		clusterextension.Delete(oc)

		g.By("Create clusterextension v1.0.2")
		clusterextension.Version = "1.0.2"
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.2"))

		g.By("Update the version to 1.0.3, check changed from Cluster to Namespaced")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.3","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.2"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `scope changed`, 10, 60, 0)

	})

	g.It("PolarionID:75218-[OTP][Skipped:Disconnected]Disabling the CRD Upgrade Safety preflight checks", func() {
		var (
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75218"
			sa                           = "sa75218"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-75218",
				Imageref: "quay.io/openshifttest/nginxolm-operator-index:nginxolm75218",
				Template: clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75218",
				InstallNamespace: ns,
				PackageName:      "nginx75218",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("update the version to 1.0.2, report messages and upgrade safety fail")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `scope changed`, 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `type changed`, 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `existingFieldRemoval`, 10, 60, 0)

		g.By("disabled crd upgrade safety check, it will not affect spec.scope: Invalid value: Cluster")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}, "install":{"preflight":{"crdUpgradeSafety":{"enforcement":"None"}}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		var message string
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 18*time.Second, false, func(ctx context.Context) (bool, error) {
			message = clusterextension.GetClusterExtensionMessage(oc, "Progressing")
			if !strings.Contains(message, `spec.scope: Invalid value: "Cluster": field is immutable`) {
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o=jsonpath-as-json={.status}")
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("Unexpected results message: %v", message))
		clusterextension.CheckClusterExtensionNotCondition(oc, "Progressing", "message", `type changed`, 10, 60, 0)
		clusterextension.CheckClusterExtensionNotCondition(oc, "Progressing", "message", `existingFieldRemoval`, 10, 60, 0)

		g.By("disabled crd upgrade safety check An existing stored version of the CRD is removed")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.3","upgradeConstraintPolicy":"SelfCertified"}}, "install":{"preflight":{"crdUpgradeSafety":{"enforcement":"None"}}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `must remain in spec.versions`, 10, 60, 0)

		g.By("disabled crd upgrade safety successfully")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.5","upgradeConstraintPolicy":"SelfCertified"}}, "install":{"preflight":{"crdUpgradeSafety":{"enforcement":"None"}}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if !strings.Contains(clusterextension.InstalledBundle, "1.0.5") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx75218 1.0.5 is not installed")

	})

	g.It("PolarionID:75122-[OTP][Skipped:Disconnected]CRD upgrade check Removing an existing stored version and add a new CRD with no modifications to existing versions", func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75122"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75122"
			sa                           = "sa75122"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-75122",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm75122",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75122",
				InstallNamespace: ns,
				PackageName:      "nginx75122",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("upgrade will be prevented if An existing stored version of the CRD is removed")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `storedVersionRemoval`, 10, 60, 0)

		g.By("upgrade will be allowed if A new version of the CRD is added with no modifications to existing versions")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.3","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 3, 150, 0)
		clusterextension.WaitClusterExtensionCondition(oc, "Installed", "True", 0)
		clusterextension.GetBundleResource(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.3"))

		clusterextension.CheckClusterExtensionCondition(oc, "Installed", "message",
			"Installed bundle quay.io/openshifttest/nginxolm-operator-bundle:v1.0.3-nginxolm75122 successfully", 10, 60, 0)

		g.By("upgrade will be prevented if An existing served version of the CRD is removed")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.6","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		clusterextension.CheckClusterExtensionCondition(oc, "Installed", "message",
			"Installed bundle quay.io/openshifttest/nginxolm-operator-bundle:v1.0.6-nginxolm75122 successfully", 10, 60, 0)

	})

	g.It("PolarionID:75123-[OTP][Skipped:Disconnected]CRD upgrade checks for changes in required field and field type", func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75123"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75123"
			sa                           = "sa75123"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-75123",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm75123",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75123",
				InstallNamespace: ns,
				PackageName:      "nginx75123",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("upgrade will be prevented if A new required field is added to an existing version of the CRD")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		// Cover test case: OCP-75217 - [olmv1] Override the unsafe upgrades with the warning message
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `new required fields`, 10, 60, 0)

		g.By("upgrade will be prevented if An existing field is removed from an existing version of the CRD")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.3","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `existingFieldRemoval`, 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `type changed`, 10, 60, 0)

		g.By("upgrade will be prevented if An existing field type is changed in an existing version of the CRD")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.6","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `type changed`, 10, 60, 0)

		g.By("upgrade will be allowed if An existing required field is changed to optional in an existing version")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.8","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 3, 150, 0)
		clusterextension.WaitClusterExtensionCondition(oc, "Installed", "True", 0)
		clusterextension.GetBundleResource(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.8"))

		clusterextension.CheckClusterExtensionCondition(oc, "Installed", "message",
			"Installed bundle quay.io/openshifttest/nginxolm-operator-bundle:v1.0.8-nginxolm75123 successfully", 10, 60, 0)

	})

	g.It("PolarionID:75124-[OTP][Skipped:Disconnected]CRD upgrade checks for changes in default values", func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75124"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75124"
			sa                           = "sa75124"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-75124",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm75124",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75124",
				InstallNamespace: ns,
				PackageName:      "nginx75124",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("upgrade will be prevented if A new default value is added to a field that did not previously have a default value")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `default added when there was none previously`, 10, 60, 0)

		g.By("upgrade will be prevented if The default value of a field is changed")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.3","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `default value changed`, 10, 60, 0)

		g.By("upgrade will be prevented if An existing default value of a field is removed")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.6","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `default value removed`, 10, 60, 0)

	})

	g.It("PolarionID:75515-[OTP][Skipped:Disconnected]CRD upgrade checks for changes in enumeration values", func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75515"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75515"
			sa                           = "sa75515"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-75515",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm75515",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75515",
				InstallNamespace: ns,
				PackageName:      "nginx75515",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("upgrade will be prevented if New enum restrictions are added to an existing field which did not previously have enum restrictions")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", `enum constraint added when there was none previously`, 10, 60, 0)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))
		clusterextension.Delete(oc)

		g.By("Create clusterextension v1.0.3")
		clusterextension.Version = "1.0.3"
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.3"))

		g.By("upgrade will be prevented if Existing enum values from an existing field are removed")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.5","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", "allowed enum values removed", 10, 60, 0)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.3"))

		g.By("upgrade will be allowed if Adding new enum values to the list of allowed enum values in a field")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.6","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterextension.WaitClusterExtensionVersion(oc, "v1.0.6")
	})

	g.It("PolarionID:75516-[OTP][Skipped:Disconnected]CRD upgrade checks for the field maximum minimum changes", func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75516"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75516"
			sa                           = "sa75516"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-75516",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm75516",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75516",
				InstallNamespace: ns,
				PackageName:      "nginx75516",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension v1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("upgrade will be prevented if The minimum value of an existing field is increased in an existing version and The maximum value of an existing field is decreased in an existing version")
		g.By("Check minimum & maximum")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.2","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", "minimum: minimum increased", 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", "maximum: maximum decreased", 10, 60, 0)

		g.By("Check minLength & maxLength")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.3","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			"maxLength: maximum decreased", 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			"minLength: minimum increased", 10, 60, 0)

		g.By("Check minProperties & maxProperties")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.4","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			"maxProperties: maximum decreased", 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			"minProperties: minimum increased", 10, 60, 0)

		g.By("Check minItems & maxItems")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.5","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			"maxItems: maximum decreased", 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			"minItems: minimum increased", 10, 60, 0)

		g.By("upgrade will be prevented if Minimum or maximum field constraints are added to a field that did not previously have constraints")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.6","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`maximum: maximum constraint added when there was none previously`, 10, 60, 0)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`minimum: minimum constraint added when there was none previously`, 10, 60, 0)

		g.By("upgrade will be Allowed if The minimum value of an existing field is decreased in an existing version & The maximum value of an existing field is increased in an existing version")
		err = oc.AsAdmin().Run("patch").Args("clusterextension", clusterextension.Name, "-p", `{"spec":{"source":{"catalog":{"version":"1.0.7","upgradeConstraintPolicy":"SelfCertified"}}}}`, "--type=merge").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 3, 150, 0)
		clusterextension.WaitClusterExtensionCondition(oc, "Installed", "True", 0)
		clusterextension.GetBundleResource(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.7"))

		clusterextension.CheckClusterExtensionCondition(oc, "Installed", "message",
			"Installed bundle quay.io/openshifttest/nginxolm-operator-bundle:v1.0.7-nginxolm75516 successfully", 10, 60, 0)

	})

})
