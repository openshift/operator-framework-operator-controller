package mocks

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// CreateClusterExtension creates a ServiceAccount, ClusterRoleBinding, and ClusterExtension with unique names.
// It returns the unique suffix and a cleanup function to delete all created resources.
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

	// 2. Create ClusterRoleBinding to grant permissions
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

	// 3. Create ClusterExtension as unstructured.Unstructured
	ce := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "olm.operatorframework.io/v1",
			"kind":       "ClusterExtension",
			"metadata": map[string]interface{}{
				"name": ceName,
			},
			"spec": map[string]interface{}{
				"namespace": ns,
				"serviceAccount": map[string]interface{}{
					"name": saName,
				},
				"source": map[string]interface{}{
					"sourceType": "Catalog",
					"catalog": map[string]interface{}{
						"packageName":             packageName,
						"version":                 version,
						"selector":                map[string]interface{}{},
						"upgradeConstraintPolicy": "CatalogProvided",
					},
				},
			},
		},
	}
	ce.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "olm.operatorframework.io",
		Version: "v1",
		Kind:    "ClusterExtension",
	})
	Expect(k8sClient.Create(ctx, ce)).To(Succeed())

	return unique, func() {
		_ = k8sClient.Delete(ctx, ce)
		_ = k8sClient.Delete(ctx, crb)
		_ = k8sClient.Delete(ctx, sa)
	}
}
