package helpers

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// NewClusterCatalog returns a new ClusterCatalog object.
// It sets the image reference as source.
func NewClusterCatalog(name, imageRef string) *olmv1.ClusterCatalog {
	return &olmv1.ClusterCatalog{
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
}

// ExpectCatalogToBeServing checks that the catalog with the given name is installed
func ExpectCatalogToBeServing(ctx context.Context, name string) {
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		var catalog olmv1.ClusterCatalog
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &catalog)
		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get catalog %q", name))

		conditions := catalog.Status.Conditions
		g.Expect(conditions).NotTo(BeEmpty(), fmt.Sprintf("catalog %q has empty status.conditions", name))

		g.Expect(meta.IsStatusConditionPresentAndEqual(conditions, olmv1.TypeServing, metav1.ConditionTrue)).
			To(BeTrue(), fmt.Sprintf("catalog %q is not serving", name))
	}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}
