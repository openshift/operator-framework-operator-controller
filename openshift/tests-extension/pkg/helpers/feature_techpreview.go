package helpers

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/extlogs"
)

// RequireTechPreviewFeatureSetOnOpenshift checks whether the OpenShift cluster is configured
// with the TechPreviewNoUpgrade or DevPreviewNoUpgrade feature set. If not, it skips the test with a message.
func RequireTechPreviewFeatureSetOnOpenshift() {
	if !env.Get().IsOpenShift {
		extlogs.Warn("Skipping tech preview feature check: not OpenShift")
		return
	}

	ctx := context.TODO()
	clientset, err := configclient.NewForConfig(env.Get().RestCfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create config client")

	featureGate, err := clientset.ConfigV1().FeatureGates().Get(ctx, "cluster", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve cluster feature gate")

	if featureGate.Spec.FeatureSet != configv1.TechPreviewNoUpgrade || featureGate.Spec.FeatureSet != configv1.DevPreviewNoUpgrade {
		Skip("Test requires a TechPreviewNoUpgrade cluster")
	}
}
