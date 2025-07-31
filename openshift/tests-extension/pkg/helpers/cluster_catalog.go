package helpers

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
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
		_ = k8sClient.Delete(ctx, &olmv1.ClusterCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}, nil
}
