package helpers

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// ClusterExtensionOption mutates ext
type ClusterExtensionOption func(ext *olmv1.ClusterExtension)

// WithCatalogSelector sets .spec.Source.Catalog.Selector to selector if ext.Spec.Source.Catalog is defined
func WithCatalogSelector(selector metav1.LabelSelector) ClusterExtensionOption {
	return func(ext *olmv1.ClusterExtension) {
		if ext == nil || ext.Spec.Source.Catalog == nil {
			return
		}
		ext.Spec.Source.Catalog.Selector = &selector
	}
}

// WithCatalogNameSelector adds a selector to the ClusterExtension's catalog filter to restrict package resolution a ClusterCatalog
// called catalogName
func WithCatalogNameSelector(catalogName string) ClusterExtensionOption {
	return WithCatalogSelector(metav1.LabelSelector{
		MatchLabels: map[string]string{
			"olm.operatorframework.io/metadata.name": catalogName,
		},
	})
}

// CreateClusterExtension creates a ServiceAccount, ClusterRoleBinding, and ClusterExtension using typed APIs.
// It returns the unique suffix and a cleanup function.
func CreateClusterExtension(packageName, version, namespace, unique string, opts ...ClusterExtensionOption) (string, func()) {
	ctx := context.TODO()
	k8sClient := env.Get().K8sClient
	if unique == "" {
		unique = rand.String(4)
	}

	saName := "install-test-sa-" + unique
	crbName := "install-test-crb-" + unique
	ceName := "install-test-ce-" + unique

	// 1. Create ServiceAccount
	sa := NewServiceAccount(saName, namespace)
	Expect(k8sClient.Create(ctx, sa)).To(Succeed(),
		"failed to create ServiceAccount")
	By("ensuring ServiceAccount is available before proceeding")
	ExpectServiceAccountExists(ctx, saName, namespace)

	// 2. Create ClusterRoleBinding
	crb := NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
	Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding")
	By("ensuring ClusterRoleBinding is available before proceeding")
	ExpectClusterRoleBindingExists(ctx, crbName)

	// 3. Create ClusterExtension
	ce := NewClusterExtensionObject(packageName, version, ceName, saName, namespace, opts...)
	Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension")

	// Cleanup closure
	return ceName, func() {
		By("deleting CluserExtension, ClusterRoleBinding and ServiceAccount")
		_ = k8sClient.Delete(ctx, ce)
		_ = k8sClient.Delete(ctx, crb)
		_ = k8sClient.Delete(ctx, sa)
	}
}

// NewServiceAccount creates a new ServiceAccount.
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
func NewClusterExtensionObject(pkg, version, ceName, saName, namespace string, opts ...ClusterExtensionOption) *olmv1.ClusterExtension {
	ext := &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{Name: ceName},
		Spec: olmv1.ClusterExtensionSpec{
			Namespace: namespace,
			ServiceAccount: olmv1.ServiceAccountReference{
				Name: saName,
			},
			Source: olmv1.SourceConfig{
				SourceType: olmv1.SourceTypeCatalog,
				Catalog: &olmv1.CatalogFilter{
					PackageName: pkg,
					Version:     version,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{},
					},
					UpgradeConstraintPolicy: olmv1.UpgradeConstraintPolicyCatalogProvided,
				},
			},
		},
	}
	for _, applyOpt := range opts {
		applyOpt(ext)
	}
	return ext
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

		progressing := meta.FindStatusCondition(conditions, string(olmv1.TypeProgressing))
		g.Expect(progressing).ToNot(BeNil(), "Progressing condition not found")
		g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue), "Progressing should be True")

		installed := meta.FindStatusCondition(conditions, string(olmv1.TypeInstalled))
		g.Expect(installed).ToNot(BeNil(), "Installed condition not found")
		g.Expect(installed.Status).To(Equal(metav1.ConditionTrue), "Installed should be True")
	}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
}

// EnsureCleanupClusterExtension attempts to delete any ClusterExtension and a specified CRD
// that might be left over from previous test runs. This helps prevent conflicts in serial tests.
func EnsureCleanupClusterExtension(ctx context.Context, packageName, crdName string) {
	k8sClient := env.Get().K8sClient

	// 1. Clean up any ClusterExtensions related to this test/package
	ceList := &olmv1.ClusterExtensionList{}
	// List all ClusterExtensions, then filter in code by packageName
	if err := k8sClient.List(ctx, ceList); err == nil {
		for _, ce := range ceList.Items {
			if ce.Spec.Source.Catalog.PackageName == packageName {
				By(fmt.Sprintf("deleting ClusterExtension %s (package: %s)", ce.Name, packageName))
				propagationPolicy := metav1.DeletePropagationForeground
				deleteOpts := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
				if err := k8sClient.Delete(ctx, &ce, deleteOpts); err != nil && !errors.IsNotFound(err) {
					fmt.Fprintf(GinkgoWriter, "Warning: Failed to delete remaning ClusterExtension %s: %v\n", ce.Name, err)
				}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKey{Name: ce.Name}, &olmv1.ClusterExtension{})
					return errors.IsNotFound(err)
				}).WithTimeout(1*time.Minute).WithPolling(2*time.Second).Should(BeTrue(), "Cleanup ClusterExtension %s failed to delete", ce.Name)
			}
		}
	} else if !errors.IsNotFound(err) {
		fmt.Fprintf(GinkgoWriter, "Warning: Failed to list ClusterExtensions during cleanup: %v\n", err)
	}

	// 2. Clean up specific operator-created CRD if it exists
	if crdName != "" {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Name: crdName}, crd); err == nil {
			By(fmt.Sprintf("deleting CRD %s", crdName))
			if err := k8sClient.Delete(ctx, crd); err != nil && !errors.IsNotFound(err) {
				fmt.Fprintf(GinkgoWriter, "Warning: Failed to delete lingering CRD %s: %v\n", crdName, err)
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: crdName}, &apiextensionsv1.CustomResourceDefinition{})
				return errors.IsNotFound(err)
			}).WithTimeout(1*time.Minute).WithPolling(2*time.Second).Should(BeTrue(), "Lingering CRD %s failed to delete", crdName)
		} else if !errors.IsNotFound(err) {
			fmt.Fprintf(GinkgoWriter, "Warning: Failed to get CRD %s during cleanup: %v\n", crdName, err)
		}
	}
}

// ExpectServiceAccountExists waits for a ServiceAccount to be available and visible to the client.
func ExpectServiceAccountExists(ctx context.Context, name, namespace string) {
	k8sClient := env.Get().K8sClient
	sa := &corev1.ServiceAccount{}
	Eventually(func(g Gomega) {
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, sa)
		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get ServiceAccount %q/%q: %v", namespace, name, err))
	}).WithTimeout(5*time.Minute).WithPolling(3*time.Second).Should(Succeed(), "ServiceAccount %q/%q did not become visible within timeout", namespace, name)
}

// ExpectClusterRoleBindingExists waits for a ClusterRoleBinding to be available and visible to the client.
func ExpectClusterRoleBindingExists(ctx context.Context, name string) {
	k8sClient := env.Get().K8sClient
	crb := &rbacv1.ClusterRoleBinding{}
	Eventually(func(g Gomega) {
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, crb)
		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get ClusterRoleBinding %q: %v", name, err))
	}).WithTimeout(10*time.Second).WithPolling(1*time.Second).Should(Succeed(), "ClusterRoleBinding %q did not become visible within timeout", name)
}
