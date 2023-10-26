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

// RequireOLMv1CapabilityOnOpenshift checks if the OpenShift cluster has
// OLMv1 capability enabled. If not, it skips the test with a message.
func RequireOLMv1CapabilityOnOpenshift() {
	if !env.Get().IsOpenShift {
		extlogs.Warn("Skipping feature capability check: not OpenShift")
		return
	}

	ctx := context.TODO()
	clientset, err := configclient.NewForConfig(env.Get().RestCfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create config client")

	cv, err := clientset.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve cluster version")

	for _, cap := range cv.Status.Capabilities.EnabledCapabilities {
		if cap == configv1.ClusterVersionCapabilityOperatorLifecycleManagerV1 {
			return
		}
	}
	Skip("Test requires OperatorLifecycleManagerV1 capability")
}
