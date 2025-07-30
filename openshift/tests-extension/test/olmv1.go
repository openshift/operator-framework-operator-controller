package test

import (
	"context"
	"fmt"
	"strings"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/commons"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/mocks"
)

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect("test").ToNot(BeEmpty())
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs", func() {
	It("should be installed", func(ctx SpecContext) {
		cfg := env.Get().RestCfg
		crds := []struct {
			group   string
			version []string
			plural  string
		}{
			{commons.GroupOLMv1, []string{"v1"}, "clusterextensions"},
			{commons.GroupOLMv1, []string{"v1"}, "clustercatalogs"},
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
				Expect(found).To(BeTrue(), fmt.Sprintf("Expected version %q in CRD %s.%s", v,
					crd.plural, crd.group))
			}
		}
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation", func() {
	It("should install a cluster extension", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP Catalogs: not OpenShift")
		}

		By("applying the ClusterExtension resource")
		name, cleanup := mocks.CreateClusterExtension("quay-operator", "3.13.0")
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to be installed")
		Eventually(commons.ExpectedConditionsMatch(
			commons.GroupOLMv1, "v1", commons.KindClusterExtension, name,
			[]metav1.Condition{
				{Type: commons.TypeProgressing, Status: metav1.ConditionTrue},
				{Type: commons.TypeInstalled, Status: metav1.ConditionTrue},
			},
		)).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())
	})

	It("should fail to install a non-existing cluster extension", func(ctx SpecContext) {
		By("applying the ClusterExtension resource")
		name, cleanup := mocks.CreateClusterExtension("does-not-exist", "99.99.99")
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to report failure")
		Eventually(commons.ExpectedConditionsMatch(
			commons.GroupOLMv1, "v1", commons.KindClusterExtension, name,
			[]metav1.Condition{
				{Type: commons.TypeProgressing, Status: metav1.ConditionTrue},
				{Type: commons.TypeInstalled, Status: metav1.ConditionFalse},
			},
		)).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())
	})

	It("should block cluster upgrades if an incompatible operator is installed", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP Catalogs: not OpenShift")
		}

		By("applying the ClusterExtension resource")
		name, cleanup := mocks.CreateClusterExtension("cluster-logging", "6.2.2")
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to be installed")
		Eventually(commons.ExpectedConditionsMatch(
			commons.GroupOLMv1, "v1", commons.KindClusterExtension, name,
			[]metav1.Condition{
				{Type: commons.TypeProgressing, Status: metav1.ConditionTrue},
				{Type: commons.TypeInstalled, Status: metav1.ConditionTrue},
			},
		)).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())

		By("ensuring the cluster is not upgradeable when olm.maxopenshiftversion is specified")
		const typeIncompatibleOperatorsUpgradeable = "InstalledOLMOperatorsUpgradeable"
		const reasonIncompatibleOperatorsInstalled = "IncompatibleOperatorsInstalled"
		const groupOpenShiftOperators = "operator.openshift.io"
		Eventually(func() error {
			obj := commons.FetchUnstructured(groupOpenShiftOperators, "v1", "OLM", "cluster")
			conds := commons.ExtractConditions(obj)

			c := meta.FindStatusCondition(conds, typeIncompatibleOperatorsUpgradeable)
			if c == nil {
				return fmt.Errorf("missing condition: %q", typeIncompatibleOperatorsUpgradeable)
			}
			if c.Status != metav1.ConditionFalse {
				return fmt.Errorf("expected status to be False, got %q", c.Status)
			}
			if c.Reason != reasonIncompatibleOperatorsInstalled {
				return fmt.Errorf("expected reason to be %q, got %q", reasonIncompatibleOperatorsInstalled, c.Reason)
			}
			if !strings.Contains(c.Message, name) {
				return fmt.Errorf("expected message to contain %q, got %q", name, c.Message)
			}
			return nil
		}).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())
	})
})
