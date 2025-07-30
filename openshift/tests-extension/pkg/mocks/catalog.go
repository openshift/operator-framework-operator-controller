package mocks

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/commons"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// CreateBrokenClusterCatalog creates a ClusterCatalog with the specified name and image reference.
// It returns a cleanup function to delete the catalog after use.
func CreateBrokenClusterCatalog(name, imageRef string) (func(), error) {
	ctx := context.TODO()
	k8sClient := env.Get().K8sClient

	cat := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", commons.GroupOLMv1, commons.CatalogAPIVersion),
			"kind":       commons.KindCatalog,
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"source": map[string]interface{}{
					"type": "Image",
					"image": map[string]interface{}{
						"ref": imageRef,
					},
				},
			},
		},
	}
	cat.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   commons.GroupOLMv1,
		Version: commons.CatalogAPIVersion,
		Kind:    commons.KindCatalog,
	})

	if err := k8sClient.Create(ctx, cat); err != nil {
		return nil, fmt.Errorf("failed to create catalog: %w", err)
	}

	return func() {
		_ = k8sClient.Delete(context.Background(), cat)
	}, nil
}
