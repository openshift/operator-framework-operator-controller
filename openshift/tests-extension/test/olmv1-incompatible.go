package test

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/origin/test/extended/util/image"
	"sigs.k8s.io/controller-runtime/pkg/client"

	catalogdata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/catalog"
	operatordata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/operator"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 operator installation", func() {
	var unique, nsName, ccName, opName string
	BeforeEach(func(ctx SpecContext) {
		testVersion := env.Get().OpenShiftVersion
		replacements := map[string]string{
			"{{ TEST-BUNDLE }}": "", // Auto-filled
			"{{ NAMESPACE }}":   "", // Auto-filled
			"{{ VERSION }}":     testVersion,

			// Using the shell image provided by origin as the controller image.
			// The image is mirrored into disconnected environments for testing.
			"{{ TEST-CONTROLLER }}": image.ShellImage(),
		}
		unique, nsName, ccName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			catalogdata.AssetNames, catalogdata.Asset,
			operatordata.AssetNames, operatordata.Asset,
		)
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), nsName)
		}
	})
	It("should block cluster upgrades if an incompatible operator is installed",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation should block cluster upgrades if an incompatible operator is installed"), func(ctx SpecContext) {
			if !env.Get().IsOpenShift {
				Skip("Requires OCP APIs: not OpenShift")
			}

			By("waiting for InstalledOLMOperatorUpgradable to be true")
			waitForOlmUpgradeStatus(ctx, operatorv1.ConditionTrue, "")

			By("creating the ClusterExtension")
			ceName, ceCleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
			DeferCleanup(ceCleanup)
			helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)

			By("waiting for InstalledOLMOperatorUpgradable to be false")
			waitForOlmUpgradeStatus(ctx, operatorv1.ConditionFalse, ceName)

			By("waiting for ClusterOperator Upgradeable to be false")
			waitForClusterOperatorUpgradable(ctx, ceName)
		})
})

func waitForOlmUpgradeStatus(ctx SpecContext, status operatorv1.ConditionStatus, name string) {
	const reasonIncompatibleOperatorsInstalled = "IncompatibleOperatorsInstalled"
	const typeInstalledOLMOperatorsUpgradeable = "InstalledOLMOperatorsUpgradeable"
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		olm := &operatorv1.OLM{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: "cluster"}, olm)
		g.Expect(err).ToNot(HaveOccurred())

		conditions := olm.Status.Conditions
		var cond *operatorv1.OperatorCondition
		for i := range conditions {
			if conditions[i].Type == typeInstalledOLMOperatorsUpgradeable {
				cond = &conditions[i]
				break
			}
		}
		g.Expect(cond).ToNot(BeNil(), "missing condition: %q", typeInstalledOLMOperatorsUpgradeable)
		g.Expect(cond.Status).To(Equal(status))
		if status == operatorv1.ConditionFalse {
			g.Expect(name).ToNot(BeEmpty())
			g.Expect(cond.Reason).To(Equal(reasonIncompatibleOperatorsInstalled))
			g.Expect(cond.Message).To(ContainSubstring(name))
		}
	}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
}

func waitForClusterOperatorUpgradable(ctx SpecContext, name string) {
	const reasonIncompatibleOperatorsInstalled = "InstalledOLMOperators_IncompatibleOperatorsInstalled"

	Eventually(func(g Gomega) {
		k8sClient := env.Get().K8sClient
		obj := &configv1.ClusterOperator{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: "olm"}, obj)
		g.Expect(err).ToNot(HaveOccurred())

		var cond *configv1.ClusterOperatorStatusCondition
		for i, c := range obj.Status.Conditions {
			if c.Type == configv1.OperatorUpgradeable {
				cond = &obj.Status.Conditions[i]
				break
			}
		}

		g.Expect(cond).ToNot(BeNil(), "missing condition: %q", configv1.OperatorUpgradeable)
		g.Expect(cond.Status).To(Equal(configv1.ConditionFalse))
		g.Expect(cond.Reason).To(Equal(reasonIncompatibleOperatorsInstalled))
		g.Expect(cond.Message).To(ContainSubstring(name))
	}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
}
