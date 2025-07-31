package test

import (
	"fmt"
	"path/filepath"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/commons"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMPreflightPermissionChecks][Skipped:Disconnected] OLMv1 operator preflight checks", func() {
	It("should report error when {services} are not specified", func(ctx SpecContext) {
		runNegativePreflightTest(1)
	})

	It("should report error when {create} verb is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(2)
	})

	It("should report error when {ClusterRoleBindings} are not specified", func(ctx SpecContext) {
		runNegativePreflightTest(3)
	})

	It("should report error when {ConfigMap:resourceNames} are not all specified", func(ctx SpecContext) {
		runNegativePreflightTest(4)
	})

	It("should report error when {clusterextension/finalizer} is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(5)
	})

	It("should report error when {escalate, bind} is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(6)
	})
})

func runNegativePreflightTest(iteration int) {
	if !env.Get().IsOpenShift {
		Skip("Skipping test because requires OCP Catalogs: not OpenShift")
	}

	crFile := filepath.Join(commons.TestdataBaseDir, fmt.Sprintf("install-pipeline-operator-%d.yaml", iteration))
	ceFile := filepath.Join(commons.TestdataBaseDir, "install-pipeline-operator-base.yaml")

	cleanupCr, unique := commons.ApplyResourceFile("pipeline", "latest", "", crFile)
	DeferCleanup(cleanupCr)

	cleanupCe, _ := commons.ApplyResourceFile("pipeline", "latest", unique, ceFile)
	DeferCleanup(cleanupCe)

	ceName := "install-test-ce-" + unique

	By("checking for progressing=true with preflight failure messages")
	Eventually(commons.ExpectedConditionsWithMessages(
		commons.GroupOLMv1, "v1", commons.KindClusterExtension, ceName,
		[]metav1.Condition{
			{Type: commons.TypeProgressing, Status: metav1.ConditionTrue},
			{Type: commons.TypeInstalled, Status: metav1.ConditionFalse},
		},
		map[string]string{
			commons.TypeProgressing: "pre-authorization failed:",
		},
	)).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())

	By("checking for progressing=true with service account message")
	Eventually(commons.ExpectedConditionsWithMessages(
		commons.GroupOLMv1, "v1", commons.KindClusterExtension, ceName,
		[]metav1.Condition{
			{Type: commons.TypeProgressing, Status: metav1.ConditionTrue},
		},
		map[string]string{
			commons.TypeProgressing: "service account requires the following permissions",
		},
	)).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())
}
