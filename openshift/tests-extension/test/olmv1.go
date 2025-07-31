package test

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})
	It("should install a cluster extension", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP Catalogs: not OpenShift")
		}
		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension("quay-operator", "3.13.0")
		DeferCleanup(cleanup)

		By("waiting for the quay-operator ClusterExtension to be installed")
		Eventually(func(g Gomega) {
			k8sClient := env.Get().K8sClient
			ce := &olmv1.ClusterExtension{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
			g.Expect(err).ToNot(HaveOccurred())

			progressing := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeProgressing)
			g.Expect(progressing).ToNot(BeNil())
			g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue))

			installed := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeInstalled)
			g.Expect(installed).ToNot(BeNil())
			g.Expect(installed.Status).To(Equal(metav1.ConditionTrue))
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})

	It("should fail to install a non-existing cluster extension", func(ctx SpecContext) {
		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension("does-not-exist", "99.99.99")
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

	It("should block cluster upgrades if an incompatible operator is installed", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP Catalogs: not OpenShift")
		}

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension("cluster-logging", "6.2.2")
		DeferCleanup(cleanup)

		By("waiting for the function-mesh ClusterExtension to be installed")
		Eventually(func(g Gomega) {
			k8sClient := env.Get().K8sClient
			ce := &olmv1.ClusterExtension{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
			g.Expect(err).ToNot(HaveOccurred())

			progressing := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeProgressing)
			g.Expect(progressing).ToNot(BeNil())
			g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue))

			installed := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeInstalled)
			g.Expect(installed).ToNot(BeNil())
			g.Expect(installed.Status).To(Equal(metav1.ConditionTrue))
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())

		By("ensuring the cluster is not upgradeable when olm.maxopenshiftversion is specified")
		const typeUpgradeable = "Upgradeable"
		const reasonIncompatibleOperatorsInstalled = "InstalledOLMOperators_IncompatibleOperatorsInstalled"

		Eventually(func(g Gomega) {
			k8sClient := env.Get().K8sClient
			obj := &configv1.ClusterOperator{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "olm"}, obj)
			g.Expect(err).ToNot(HaveOccurred())

			var cond *configv1.ClusterOperatorStatusCondition
			for i, c := range obj.Status.Conditions {
				if c.Type == typeUpgradeable {
					cond = &obj.Status.Conditions[i]
					break
				}
			}

			g.Expect(cond).ToNot(BeNil(), "missing condition: %q", typeUpgradeable)
			g.Expect(cond.Status).To(Equal(configv1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(reasonIncompatibleOperatorsInstalled))
			g.Expect(cond.Message).To(ContainSubstring(name))
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})
})
