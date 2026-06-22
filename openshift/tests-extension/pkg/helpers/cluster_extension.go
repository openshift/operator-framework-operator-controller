package helpers

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

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

func WithProgressDeadlineMinutes(minutes int32) ClusterExtensionOption {
	return func(ext *olmv1.ClusterExtension) {
		if ext == nil {
			return
		}
		ext.Spec.ProgressDeadlineMinutes = minutes
	}
}

// CreateClusterExtension creates a ClusterExtension using typed APIs.
// It returns the unique suffix and a cleanup function.
func CreateClusterExtension(packageName, version, namespace, unique string, opts ...ClusterExtensionOption) (string, func()) {
	ctx := context.TODO()
	k8sClient := env.Get().K8sClient
	if unique == "" {
		unique = rand.String(4)
	}

	ceName := "install-test-ce-" + unique

	// 3. Create ClusterExtension
	ce := NewClusterExtensionObject(packageName, version, ceName, namespace, opts...)
	Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension")

	// Cleanup closure
	return ceName, func() {
		By("deleting CluserExtension")
		_ = k8sClient.Delete(ctx, ce)
	}
}

// NewClusterExtensionObject creates a new ClusterExtension object with the specified package, version, and name.
func NewClusterExtensionObject(pkg, version, ceName, namespace string, opts ...ClusterExtensionOption) *olmv1.ClusterExtension {
	ext := &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{Name: ceName},
		Spec: olmv1.ClusterExtensionSpec{
			Namespace: namespace,
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
// Uses InstallTimeout because the BoxcutterRuntime requires all availability probes to pass
// before marking the extension as installed.
func ExpectClusterExtensionToBeInstalled(ctx context.Context, name string) {
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		var ext olmv1.ClusterExtension
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
		g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get ClusterExtension %q", name))

		conditions := ext.Status.Conditions
		g.Expect(conditions).NotTo(BeEmpty(), fmt.Sprintf("ClusterExtension %q has empty status.conditions", name))

		progressing := meta.FindStatusCondition(conditions, string(olmv1.TypeProgressing))
		installed := meta.FindStatusCondition(conditions, string(olmv1.TypeInstalled))

		pStatus, pReason, pMsg := conditionSummary(progressing)
		iStatus, iReason, iMsg := conditionSummary(installed)
		fmt.Fprintf(GinkgoWriter, "CE %q: Progressing=%s/%s (%s), Installed=%s/%s (%s)\n",
			name, pStatus, pReason, pMsg, iStatus, iReason, iMsg)

		g.Expect(progressing).ToNot(BeNil(), "Progressing condition not found")
		g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue), "Progressing should be True")

		g.Expect(installed).ToNot(BeNil(), "Installed condition not found")
		g.Expect(installed.Status).To(Equal(metav1.ConditionTrue), "Installed should be True")
	}).WithTimeout(InstallTimeout).WithPolling(DefaultPolling).Should(Succeed())
}

func conditionSummary(cond *metav1.Condition) (string, string, string) {
	if cond == nil {
		return "<nil>", "", ""
	}
	msg := cond.Message
	if len(msg) > 120 {
		msg = msg[:120] + "..."
	}
	return string(cond.Status), cond.Reason, msg
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
				}).WithTimeout(DefaultTimeout).WithPolling(DefaultPolling).Should(BeTrue(), "Cleanup ClusterExtension %s failed to delete", ce.Name)
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
			}).WithTimeout(DefaultTimeout).WithPolling(DefaultPolling).Should(BeTrue(), "Lingering CRD %s failed to delete", crdName)
		} else if !errors.IsNotFound(err) {
			fmt.Fprintf(GinkgoWriter, "Warning: Failed to get CRD %s during cleanup: %v\n", crdName, err)
		}
	}
}
