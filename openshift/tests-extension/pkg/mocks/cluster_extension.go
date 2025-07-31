package mocks

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// CreateClusterExtension creates a ServiceAccount, ClusterRoleBinding, and ClusterExtension using typed APIs.
// It returns a unique suffix and a cleanup function.
func CreateClusterExtension(packageName, version string) (string, func()) {
	ctx := context.TODO()
	k8sClient := env.Get().K8sClient
	ns := "default"
	unique := rand.String(8)

	saName := "install-test-sa-" + unique
	crbName := "install-test-crb-" + unique
	ceName := "install-test-ce-" + unique

	// 1. Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: ns,
		},
	}
	Expect(k8sClient.Create(ctx, sa)).To(Succeed())

	// 2. Create ClusterRoleBinding
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: crbName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      saName,
			Namespace: ns,
		}},
	}
	Expect(k8sClient.Create(ctx, crb)).To(Succeed())

	// 3. Create ClusterExtension
	ce := &ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: ceName,
		},
		Spec: ocv1.ClusterExtensionSpec{
			Namespace: ns,
			ServiceAccount: ocv1.ServiceAccountReference{
				Name: saName,
			},
			Source: ocv1.SourceConfig{
				SourceType: ocv1.SourceTypeCatalog,
				Catalog: &ocv1.CatalogFilter{
					PackageName:             packageName,
					Version:                 version,
					Selector:                &metav1.LabelSelector{},
					UpgradeConstraintPolicy: ocv1.UpgradeConstraintPolicyCatalogProvided,
				},
			},
		},
	}
	Expect(k8sClient.Create(ctx, ce)).To(Succeed())

	// Cleanup closure
	return unique, func() {
		_ = k8sClient.Delete(ctx, ce)
		_ = k8sClient.Delete(ctx, crb)
		_ = k8sClient.Delete(ctx, sa)
	}
}
