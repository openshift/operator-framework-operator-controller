package helpers

import (
	"context"
	"fmt"

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
	clientset := configv1Client()
	cv, err := clientset.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve cluster version")

	for _, capability := range cv.Status.Capabilities.EnabledCapabilities {
		if capability == configv1.ClusterVersionCapabilityOperatorLifecycleManagerV1 {
			return
		}
	}
	Skip("Test requires OperatorLifecycleManagerV1 capability")
}

// RequireFeatureGateEnabled requires featureGate to be in the list of enabled feature gates
// for the OCP version in env.Get().OpenShiftVersion in the status of the
// featuregates.config.openshift.io cluster resource
func RequireFeatureGateEnabled(featureGate configv1.FeatureGateName) {
	if !IsFeatureGateEnabled(featureGate) {
		Skip(fmt.Sprintf("Test requires %q feature gate to be enabled", featureGate))
	}
}

// RequireFeatureGateDisabled requires featureGate to be in the list of disabled feature gates
// for the OCP version in env.Get().OpenShiftVersion in the status of the
// featuregates.config.openshift.io cluster resource
func RequireFeatureGateDisabled(featureGate configv1.FeatureGateName) {
	if IsFeatureGateEnabled(featureGate) {
		Skip(fmt.Sprintf("Test requires %q feature gate to be disabled", featureGate))
	}
}

func IsFeatureGateEnabled(featureGate configv1.FeatureGateName) bool {
	featureGates := getFeatureGateDetails(configv1Client())
	for _, fg := range featureGates.Enabled {
		if fg.Name == featureGate {
			return true
		}
	}
	return false
}

func getFeatureGateDetails(clientset *configclient.Clientset) configv1.FeatureGateDetails {
	ctx := context.TODO()
	featureGates, err := clientset.ConfigV1().FeatureGates().Get(ctx, "cluster", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve feature gates")

	openshiftVersion := getOpenShiftVersion(clientset)
	for _, fgs := range featureGates.Status.FeatureGates {
		if fgs.Version == openshiftVersion {
			return fgs
		}
	}
	Fail(fmt.Sprintf("No feature gates found for version %s", openshiftVersion))
	return configv1.FeatureGateDetails{}
}

func getOpenShiftVersion(clientset *configclient.Clientset) string {
	// Fetch the single global 'version' resource
	cv, err := clientset.ConfigV1().ClusterVersions().Get(context.TODO(), "version", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to retrieve cluster version")

	// The status.history is a slice of updates.
	// The first element (index 0) is the current or most recent version.
	if len(cv.Status.History) > 0 {
		return cv.Status.History[0].Version
	}

	Fail("No history found in cluster version status")
	return ""
}

func configv1Client() *configclient.Clientset {
	clientset, err := configclient.NewForConfig(env.Get().RestCfg)
	Expect(err).NotTo(HaveOccurred(), "failed to create config client")
	return clientset
}
