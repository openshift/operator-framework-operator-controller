package specs

import (
	"context"
	"fmt"
	"path/filepath"
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
		clustercatalog.CheckClusterCatalogCondition(oc, "Progressing", "message", "manifest unknown", 5, 90, 0)
	})

})
