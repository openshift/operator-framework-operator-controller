package test

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected] OLMv1 operator installation support for singleNamespace watch mode with quay-operator", Serial, func() {
	var (
		k8sClient client.Client
		namespace string
	)

	BeforeEach(func() {
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}

		k8sClient = env.Get().K8sClient
		namespace = "olmv1-single-own-ns-" + rand.String(4)

		By(fmt.Sprintf("creating namespace %s for single-namespace tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace %q", namespace)
		DeferCleanup(func() {
			By(fmt.Sprintf("cleaning up namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns)
		})
	})

	It("should install a cluster extension successfully", func(ctx SpecContext) {
		unique := rand.String(4)
		saName := "install-test-sa-" + unique
		crbName := "install-test-crb-" + unique
		ceName := "install-test-ce-" + unique

		By("creating ServiceAccount, ClusterRoleBinding, and ClusterExtension with the watch-namespace annotation")
		sa := helpers.NewServiceAccount(saName, namespace)
		Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, sa) })

		crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
		Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, crb) })

		ce := helpers.NewClusterExtensionObject("quay-operator", "3.14.2", ceName, saName, namespace)
		ce.Annotations = map[string]string{
			"olm.operatorframework.io/watch-namespace": namespace,
		}
		Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, ce) })

		By("waiting for the ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected] OLMv1 operator installation support for ownNamespace watch mode with quay-operator", Serial, func() {
	var (
		k8sClient client.Client
		namespace string
	)

	BeforeEach(func() {
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}

		k8sClient = env.Get().K8sClient
		namespace = "olmv1-own-ns-" + rand.String(4)

		By(fmt.Sprintf("creating namespace %s for own-namespace tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace %q", namespace)
		DeferCleanup(func() {
			By(fmt.Sprintf("cleaning up namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns)
		})
	})

	It("should install a cluster extension successfully", func(ctx SpecContext) {
		unique := rand.String(4)
		saName := "install-test-sa-" + unique
		crbName := "install-test-crb-" + unique
		ceName := "install-test-ce-" + unique

		By("creating ServiceAccount, ClusterRoleBinding, and ClusterExtension with the watch-namespace annotation")
		sa := helpers.NewServiceAccount(saName, namespace)
		Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, sa) })

		crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
		Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, crb) })

		ce := helpers.NewClusterExtensionObject("quay-operator", "3.14.2", ceName, saName, namespace)
		ce.Annotations = map[string]string{
			"olm.operatorframework.io/watch-namespace": namespace,
		}
		Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, ce) })

		By("waiting for the ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected] OLMv1 operator installation support for ownNamespace watch mode with an operator that does not support ownNamespace installation mode", func() {
	var (
		k8sClient client.Client
		namespace string
	)

	BeforeEach(func() {
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}

		k8sClient = env.Get().K8sClient
		namespace = "olmv1-failing-own-ns-" + rand.String(4)

		By(fmt.Sprintf("creating namespace %s for failing tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace %q", namespace)
		DeferCleanup(func() {
			By(fmt.Sprintf("cleaning up namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns)
		})
	})

	It("should fail to install a cluster extension successfully", func(ctx SpecContext) {
		unique := rand.String(4)
		saName := "install-test-sa-" + unique
		crbName := "install-test-crb-" + unique
		ceName := "install-test-ce-" + unique

		By("creating ServiceAccount, ClusterRoleBinding, and ClusterExtension with the watch-namespace annotation")
		sa := helpers.NewServiceAccount(saName, namespace)
		Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, sa) })

		crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
		Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, crb) })

		ce := helpers.NewClusterExtensionObject("openshift-pipelines-operator-rh", "1.17.1", ceName, saName, namespace)
		ce.Annotations = map[string]string{
			"olm.operatorframework.io/watch-namespace": namespace,
		}
		Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, ce) })

		By("waiting for the ClusterExtension to fail installation")
		Eventually(func(g Gomega) {
			var ext olmv1.ClusterExtension
			err := k8sClient.Get(ctx, client.ObjectKey{Name: ceName}, &ext)
			g.Expect(err).ToNot(HaveOccurred(), "failed to get ClusterExtension %q", ceName)

			conditions := ext.Status.Conditions
			g.Expect(conditions).ToNot(BeEmpty(), "ClusterExtension %q has empty status.conditions", ceName)

			installed := meta.FindStatusCondition(conditions, olmv1.TypeInstalled)
			g.Expect(installed).ToNot(BeNil(), "Installed condition not found")
			g.Expect(installed.Status).To(Equal(metav1.ConditionFalse), "Installed should be False")
			g.Expect(installed.Reason).To(Equal("Failed"))
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})
})
