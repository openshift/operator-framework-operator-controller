package test

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs", func() {
	const olmv1GroupName = "olm.operatorframework.io"
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})

	It("should be installed", func(ctx SpecContext) {
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
				Expect(found).To(BeTrue(), fmt.Sprintf("Expected version %q in CRD %s.%s", v, crd.plural, crd.group))
			}
		}
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation", func() {
	var (
		namespace string
		k8sClient client.Client
	)

	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient
		namespace = "install-test-ns-" + rand.String(4)
		By(fmt.Sprintf("creating namespace %s", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace")
		DeferCleanup(func() {
			By(fmt.Sprintf("deleting namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns)
		})
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), namespace)
		}
	})

	It("should install a cluster extension", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP Catalogs: not OpenShift")
		}

		By("ensuring no ClusterExtension and CRD for quay-operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension("quay-operator", "3.13.0", namespace, "")
		DeferCleanup(cleanup)

		By("waiting for the quay-operator ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, name)
	})

	It("should fail to install a non-existing cluster extension", func(ctx SpecContext) {
		By("ensuring no ClusterExtension and CRD for non-existing operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), "does-not-exist", "") // No CRD expected for non-existing operator

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension("does-not-exist", "99.99.99", namespace, "")
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to exist")
		ce := &olmv1.ClusterExtension{}
		Eventually(func() error {
			return env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())

		By("waiting up to 2 minutes for ClusterExtension to report failure")
		Eventually(func(g Gomega) {
			k8sClient := env.Get().K8sClient
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
			g.Expect(err).ToNot(HaveOccurred())

			progressing := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeProgressing)
			g.Expect(progressing).ToNot(BeNil())
			g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(progressing.Reason).To(Equal("Retrying"))
			g.Expect(progressing.Message).To(ContainSubstring(`no bundles found`))

			installed := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeInstalled)
			g.Expect(installed).ToNot(BeNil())
			g.Expect(installed.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(installed.Reason).To(Equal("Failed"))
			g.Expect(installed.Message).To(Equal("No bundle installed"))
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})
})
