package mocks

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// CreateBrokenClusterCatalog creates a ClusterCatalog with the specified name and image reference using the strongly typed API.
// It returns a cleanup function to delete the catalog after use.
func CreateBrokenClusterCatalog(name, imageRef string) (func(), error) {
	ctx := context.TODO()
	k8sClient := env.Get().K8sClient

	catalog := &ocv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: ocv1.ClusterCatalogSpec{
			Source: ocv1.CatalogSource{
				Type: ocv1.SourceTypeImage,
				Image: &ocv1.ImageSource{
					Ref: imageRef,
				},
			},
		},
	}

	if err := k8sClient.Create(ctx, catalog); err != nil {
		return nil, fmt.Errorf("failed to create ClusterCatalog: %w", err)
	}

	// Cleanup function to delete the catalog when done
	return func() {
		_ = k8sClient.Delete(context.Background(), &ocv1.ClusterCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}, nil
}
