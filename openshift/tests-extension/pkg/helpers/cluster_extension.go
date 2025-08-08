package helpers

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// CreateClusterExtension creates a ServiceAccount, ClusterRoleBinding, and ClusterExtension using typed APIs.
// It returns the unique suffix and a cleanup function.
func CreateClusterExtension(packageName, version, namespace string) (string, func()) {
	ctx := context.TODO()
	k8sClient := env.Get().K8sClient
	unique := rand.String(8)

	saName := "install-test-sa-" + unique
	crbName := "install-test-crb-" + unique
	ceName := "install-test-ce-" + unique

	// 1. Create ServiceAccount
	sa := NewServiceAccount(saName, namespace)
	Expect(k8sClient.Create(ctx, sa)).To(Succeed(),
		"failed to create ServiceAccount")

	// 2. Create ClusterRoleBinding
	crb := NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
	Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding")

	// 3. Create ClusterExtension
	ce := NewClusterExtensionObject(packageName, version, ceName, saName, namespace)
	Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension")

	// Cleanup closure
	return ceName, func() {
		_ = k8sClient.Delete(ctx, ce)
		_ = k8sClient.Delete(ctx, crb)
		_ = k8sClient.Delete(ctx, sa)
	}
}

// NewServiceAccount creates a new ServiceAccount object in the openshift-operators namespace.
func NewServiceAccount(name, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// NewClusterRoleBinding creates a new ClusterRoleBinding object that binds a ClusterRole to a ServiceAccount.
func NewClusterRoleBinding(name, roleName, saName, namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      saName,
			Namespace: namespace,
		}},
	}
}

// NewClusterExtensionObject creates a new ClusterExtension object with the specified package, version, name, and ServiceAccount.
func NewClusterExtensionObject(pkg, version, ceName, saName, namespace string) *olmv1.ClusterExtension {
	return &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{Name: ceName},
		Spec: olmv1.ClusterExtensionSpec{
			Namespace: namespace,
			ServiceAccount: olmv1.ServiceAccountReference{
				Name: saName,
			},
			Source: olmv1.SourceConfig{
				SourceType: olmv1.SourceTypeCatalog,
				Catalog: &olmv1.CatalogFilter{
					PackageName:             pkg,
					Version:                 version,
					Selector:                &metav1.LabelSelector{},
					UpgradeConstraintPolicy: olmv1.UpgradeConstraintPolicyCatalogProvided,
				},
			},
		},
	}
}

// ExpectClusterExtensionToBeInstalled checks that the ClusterExtension has both Progressing=True and Installed=True.
func ExpectClusterExtensionToBeInstalled(ctx context.Context, name string) {
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		var ext olmv1.ClusterExtension
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get ClusterExtension %q", name))

		conditions := ext.Status.Conditions
		g.Expect(conditions).NotTo(BeEmpty(), fmt.Sprintf("ClusterExtension %q has empty status.conditions", name))

		progressing := meta.FindStatusCondition(conditions, olmv1.TypeProgressing)
		g.Expect(progressing).ToNot(BeNil(), "Progressing condition not found")
		g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue), "Progressing should be True")

		installed := meta.FindStatusCondition(conditions, olmv1.TypeInstalled)
		g.Expect(installed).ToNot(BeNil(), "Installed condition not found")
		g.Expect(installed.Status).To(Equal(metav1.ConditionTrue), "Installed should be True")
	}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
}
