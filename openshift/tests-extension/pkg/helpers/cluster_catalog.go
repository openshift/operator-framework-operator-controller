package helpers

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// CreateClusterCatalog creates a ClusterCatalog with the specified name and image reference
// Returns a cleanup function to delete the catalog after use.
func CreateClusterCatalog(ctx context.Context, name, imageRef string) (func(), error) {
	k8sClient := env.Get().K8sClient

	catalog := &olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: olmv1.ClusterCatalogSpec{
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
				Image: &olmv1.ImageSource{
					Ref: imageRef,
				},
			},
		},
	}

	if err := k8sClient.Create(ctx, catalog); err != nil {
		return nil, fmt.Errorf("failed to create ClusterCatalog: %w", err)
	}

	return func() {
		ctx := context.TODO()
		k := env.Get().K8sClient
		obj := &olmv1.ClusterCatalog{ObjectMeta: metav1.ObjectMeta{Name: name}}

		_ = k.Delete(ctx, obj)
		EnsureCleanupClusterCatalog(ctx, name)
	}, nil
}

func EnsureCleanupClusterCatalog(ctx context.Context, name string) {
	k8s := env.Get().K8sClient
	cc := &olmv1.ClusterCatalog{}
	key := client.ObjectKey{Name: name}

	if err := k8s.Get(ctx, key, cc); err != nil {
		if !errors.IsNotFound(err) {
			fmt.Fprintf(GinkgoWriter, "Warning: failed to get ClusterCatalog %q during cleanup: %v\n", name, err)
		}
		return
	}

	By(fmt.Sprintf("deleting lingering ClusterCatalog %q", name))
	if err := k8s.Delete(ctx, cc); err != nil && !errors.IsNotFound(err) {
		fmt.Fprintf(GinkgoWriter, "Warning: failed to delete ClusterCatalog %q: %v\n", name, err)
	}

	Eventually(func() bool {
		err := k8s.Get(ctx, key, &olmv1.ClusterCatalog{})
		return errors.IsNotFound(err)
	}).WithTimeout(DefaultTimeout).WithPolling(DefaultPolling).
		Should(BeTrue(), "ClusterCatalog %q failed to delete", name)
}
