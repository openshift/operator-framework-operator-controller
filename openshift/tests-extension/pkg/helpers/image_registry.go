package helpers

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/extlogs"
)

// RequireImageRegistry checks if the OpenShift image-registry is available in the cluster.
// If the image-registry is not available (either not installed or no pods running), it skips the test.
// This is necessary for tests that depend on the internal image registry service at
// image-registry.openshift-image-registry.svc:5000
func RequireImageRegistry(ctx context.Context) {
	if !env.Get().IsOpenShift {
		extlogs.Warn("Skipping image-registry check: not OpenShift")
		return
	}

	clientset, err := kubernetes.NewForConfig(env.Get().RestCfg)
	if err != nil {
		extlogs.WarnContextf("Failed to create kubernetes client for image-registry check: %v", err)
		Skip("Cannot verify image-registry availability: failed to create kubernetes client")
		return
	}

	// Check if there are any running image-registry pods
	pods, err := clientset.CoreV1().Pods("openshift-image-registry").List(ctx, metav1.ListOptions{
		LabelSelector: "docker-registry=default",
	})

	if err != nil {
		extlogs.WarnContextf("Failed to list image-registry pods: %v", err)
		Skip("Cannot verify image-registry availability: failed to list pods")
		return
	}

	if len(pods.Items) == 0 {
		Skip("Test requires image-registry to be available, but no image-registry pods found in openshift-image-registry namespace")
	}

	extlogs.Infof("Image-registry is available with %d pod(s) running", len(pods.Items))
}
