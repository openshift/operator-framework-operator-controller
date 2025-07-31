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

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected]", func() {
	var (
		ceFile        string
		expectFailure bool
	)

	BeforeEach(func() {
		if !env.Get().IsOpenShift {
			Skip("Skipping test because requires OCP Catalogs: not OpenShift")
		}
	})

	It("should install a cluster extension successfully", func(ctx SpecContext) {
		ceFile = filepath.Join(commons.TestdataBaseDir, "install-quay-operator-singlens.yaml")
		expectFailure = false
		runSingleOwnNamespaceTest(ceFile, expectFailure)
	})

	It("should install a cluster extension successfully", func(ctx SpecContext) {
		ceFile = filepath.Join(commons.TestdataBaseDir, "install-quay-operator-ownns.yaml")
		expectFailure = false
		runSingleOwnNamespaceTest(ceFile, expectFailure)
	})

	It("should fail to install a cluster extension successfully", func(ctx SpecContext) {
		ceFile = filepath.Join(commons.TestdataBaseDir, "install-openshift-pipelines-operator-ownns.yaml")
		expectFailure = true
		runSingleOwnNamespaceTest(ceFile, expectFailure)
	})
})

func runSingleOwnNamespaceTest(ceFile string, expectFailure bool) {
	if !env.Get().IsOpenShift {
		Skip("Skipping test because requires OCP Catalogs: not OpenShift")
	}
	cleanup, unique := commons.ApplyResourceFile("quay-operator", "3.13.0", "", ceFile)
	DeferCleanup(cleanup)

	ceName := "install-test-ce-" + unique
	var lastReason string

	Eventually(func() error {
		obj := commons.FetchUnstructured(commons.GroupOLMv1, "v1", commons.KindClusterExtension, ceName)
		conds := commons.ExtractConditions(obj)
		for _, cond := range conds {
			if cond.Type == commons.TypeInstalled {
				lastReason = cond.Reason
				if cond.Status == metav1.ConditionTrue && !expectFailure {
					return nil
				}
				if cond.Status == metav1.ConditionFalse && expectFailure {
					return nil
				}
			}
		}
		return fmt.Errorf("waiting for expected Installed condition")
	}).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())

	if expectFailure {
		Expect(lastReason).NotTo(BeEmpty())
	} else {
		Expect(lastReason).To(BeEmpty())
	}
}
