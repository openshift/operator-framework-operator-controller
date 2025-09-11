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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation support for singleNamespace watch mode with quay-operator", Ordered, Serial, func() {
	var (
		k8sClient  client.Client
		namespace  string
		testPrefix = "quay-singlens"
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func() {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient

		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		By("ensuring no ClusterExtension and CRD for quay-operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

		By(fmt.Sprintf("creating namespace %s for single-namespace tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace %q", namespace)
		DeferCleanup(func() {
			By(fmt.Sprintf("cleaning up namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterExtensions(ctx, namespace)
		}
	})

	It("should install a cluster extension successfully",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected] OLMv1 operator installation support for singleNamespace watch mode with quay-operator should install a cluster extension successfully"),
		func(ctx SpecContext) {
			By("creating ServiceAccount")
			sa := helpers.NewServiceAccount(saName, namespace)
			Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
			By("ensuring ServiceAccount is available before proceeding")
			helpers.ExpectServiceAccountExists(ctx, saName, namespace)
			By("registering cleanup for ServiceAccount")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
				_ = k8sClient.Delete(context.Background(), sa, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterRoleBinding")
			crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
			Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
			By("ensuring ClusterRoleBinding is available before proceeding")
			helpers.ExpectClusterRoleBindingExists(ctx, crbName)
			By("registering cleanup for ClusterRoleBinding")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", crb.Name))
				_ = k8sClient.Delete(context.Background(), crb, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterExtension with the watch-namespace configured")
			ce := helpers.NewClusterExtensionObject("quay-operator", "3.14.2", ceName, saName, namespace)
			ce.Spec.Config = &olmv1.ClusterExtensionConfig{
				ConfigType: "Inline",
				Inline: &apiextensionsv1.JSON{
					Raw: []byte(fmt.Sprintf(`{"watchNamespace": "%s"}`, namespace)),
				},
			}
			Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
			By("registering cleanup for ClusterExtension")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterExtension %s", ce.Name))
				_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))

				By("ensuring ClusterExtension is deleted")
				helpers.EnsureCleanupClusterExtension(context.Background(), ceName, namespace)
			})

			By("waiting for the ClusterExtension to be installed")
			helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)
		})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation support for ownNamespace watch mode with quay-operator", Ordered, Serial, func() {
	var (
		k8sClient  client.Client
		namespace  string
		testPrefix = "quay-ownns"
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func() {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient
		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		By("ensuring no ClusterExtension and CRD for quay-operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

		By(fmt.Sprintf("creating namespace %s for own-namespace tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace %q", namespace)
		DeferCleanup(func() {
			By(fmt.Sprintf("cleaning up namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterExtensions(ctx, namespace)
		}
	})

	It("should install a cluster extension successfully",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected] OLMv1 operator installation support for ownNamespace watch mode with quay-operator should install a cluster extension successfully"),
		func(ctx SpecContext) {
			By("creating ServiceAccount")
			sa := helpers.NewServiceAccount(saName, namespace)
			Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
			By("ensuring ServiceAccount is available before proceeding")
			helpers.ExpectServiceAccountExists(ctx, saName, namespace)
			By("registering cleanup for ServiceAccount")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
				_ = k8sClient.Delete(context.Background(), sa, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterRoleBinding")
			crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
			Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
			By("ensuring ClusterRoleBinding is available before proceeding")
			helpers.ExpectClusterRoleBindingExists(ctx, crbName)
			By("registering cleanup for ClusterRoleBinding")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", crb.Name))
				_ = k8sClient.Delete(context.Background(), crb, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterExtension with the watch-namespace configured")
			ce := helpers.NewClusterExtensionObject("quay-operator", "3.14.2", ceName, saName, namespace)
			ce.Spec.Config = &olmv1.ClusterExtensionConfig{
				ConfigType: "Inline",
				Inline: &apiextensionsv1.JSON{
					Raw: []byte(fmt.Sprintf(`{"watchNamespace": "%s"}`, namespace)),
				},
			}
			Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
			By("registering cleanup for ClusterExtension")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterExtension %s", ce.Name))
				_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))

				By("ensuring ClusterExtension is deleted")
				helpers.EnsureCleanupClusterExtension(context.Background(), ceName, namespace)
			})

			By("waiting for the ClusterExtension to be installed")
			helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)
		})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation support for ownNamespace watch mode with an operator that does not support ownNamespace installation mode", Ordered, Serial, func() {
	var (
		k8sClient  client.Client
		namespace  string
		testPrefix = "pipelines"
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func() {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient
		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		By("ensuring no ClusterExtension and CRD for openshift-pipelines-operator-rh")
		helpers.EnsureCleanupClusterExtension(context.Background(), "openshift-pipelines-operator-rh", "clustertasks.tekton.dev")

		By(fmt.Sprintf("creating namespace %s for failing tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace %q", namespace)
		DeferCleanup(func() {
			By(fmt.Sprintf("cleaning up namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterExtensions(ctx, namespace)
		}
	})

	It("should fail to install a cluster extension successfully",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected] OLMv1 operator installation support for ownNamespace watch mode with an operator that does not support ownNamespace installation mode should fail to install a cluster extension successfully"),
		func(ctx SpecContext) {
			By("creating ServiceAccount")
			sa := helpers.NewServiceAccount(saName, namespace)
			Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
			By("ensuring ServiceAccount is available before proceeding")
			helpers.ExpectServiceAccountExists(ctx, saName, namespace)
			By("registering cleanup for ServiceAccount")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
				_ = k8sClient.Delete(context.Background(), sa, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterRoleBinding")
			crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
			Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
			By("ensuring ClusterRoleBinding is available before proceeding")
			helpers.ExpectClusterRoleBindingExists(ctx, crbName)
			By("registering cleanup for ClusterRoleBinding")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", crb.Name))
				_ = k8sClient.Delete(context.Background(), crb, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterExtension with the watch-namespace configured")
			ce := helpers.NewClusterExtensionObject("openshift-pipelines-operator-rh", "1.17.1", ceName, saName, namespace)
			ce.Spec.Config = &olmv1.ClusterExtensionConfig{
				ConfigType: "Inline",
				Inline: &apiextensionsv1.JSON{
					Raw: []byte(fmt.Sprintf(`{"watchNamespace": "%s"}`, namespace)),
				},
			}
			Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
			By("registering cleanup for ClusterExtension")
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterExtension %s", ce.Name))
				_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))

				By("ensuring ClusterExtension is deleted")
				helpers.EnsureCleanupClusterExtension(context.Background(), ceName, namespace)
			})

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
