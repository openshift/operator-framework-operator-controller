package util

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestConfigureKubernetesTestContextAfterSuiteBuilt(t *testing.T) {
	if !ginkgo.GetSuite().InPhaseBuildTree() {
		if err := ginkgo.GetSuite().BuildTree(); err != nil {
			t.Fatal(err)
		}
	}

	providerName := TestContext.Provider
	provider := TestContext.CloudConfig.Provider
	t.Cleanup(func() {
		TestContext.Provider = providerName
		TestContext.CloudConfig.Provider = provider
	})

	configureKubernetesTestContext()
	if TestContext.CloudConfig.Provider == nil {
		t.Fatal("Kubernetes test provider was not initialized")
	}
}
