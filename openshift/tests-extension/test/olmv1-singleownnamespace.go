package test

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
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

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation support for ownNamespace and single namespace watch mode with quay-operator", Ordered, Serial, func() {
	var (
		k8sClient        client.Client
		activeNamespaces map[string]struct{}
	)

	BeforeEach(func() {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient
		activeNamespaces = map[string]struct{}{}
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			for ns := range activeNamespaces {
				helpers.DescribeAllClusterExtensions(ctx, ns)
			}
		}
	})

	It("should install cluster extensions successfully in both watch modes",
		func(ctx SpecContext) {
			scenarios := []struct {
				id     string
				label  string
				watchN func(string) string
			}{
				{
					id:    "singlens",
					label: "singleNamespace watch mode",
					watchN: func(installNamespace string) string {
						return fmt.Sprintf("%s-watch", installNamespace)
					},
				},
				{
					id:    "ownns",
					label: "ownNamespace watch mode",
					watchN: func(installNamespace string) string {
						return installNamespace
					},
				},
			}

			for _, scenario := range scenarios {
				sc := scenario
				suffix := rand.String(4)
				installNamespace := fmt.Sprintf("olmv1-quay-bothns-%s-%s", sc.id, suffix)
				watchNamespace := sc.watchN(installNamespace)

				activeNamespaces[installNamespace] = struct{}{}
				if watchNamespace != installNamespace {
					activeNamespaces[watchNamespace] = struct{}{}
				}

				By(fmt.Sprintf("ensuring no ClusterExtension and CRD for quay-operator before %s scenario", sc.label))
				helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

				By(fmt.Sprintf("creating namespace %s for %s tests", installNamespace, sc.label))
				installNS := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: installNamespace,
					},
				}
				Expect(k8sClient.Create(ctx, installNS)).To(Succeed(), "failed to create install namespace %q", installNamespace)
				installNamespaceCopy := installNamespace
				DeferCleanup(func() {
					By(fmt.Sprintf("cleanup: deleting install namespace %s", installNamespaceCopy))
					_ = k8sClient.Delete(context.Background(), &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: installNamespaceCopy},
					}, client.PropagationPolicy(metav1.DeletePropagationForeground))
				})

				var watchNSObj *corev1.Namespace
				if watchNamespace != installNamespace {
					By(fmt.Sprintf("creating namespace %s for watch namespace in %s scenario", watchNamespace, sc.label))
					watchNSObj = &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: watchNamespace},
					}
					Expect(k8sClient.Create(ctx, watchNSObj)).To(Succeed(), "failed to create watch namespace %q", watchNamespace)
					watchNamespaceCopy := watchNamespace
					DeferCleanup(func() {
						By(fmt.Sprintf("cleanup: deleting watch namespace %s", watchNamespaceCopy))
						_ = k8sClient.Delete(context.Background(), &corev1.Namespace{
							ObjectMeta: metav1.ObjectMeta{Name: watchNamespaceCopy},
						}, client.PropagationPolicy(metav1.DeletePropagationForeground))
					})
				}

				saName := fmt.Sprintf("install-quay-bothns-%s-sa-%s", sc.id, suffix)
				By(fmt.Sprintf("creating ServiceAccount %s for %s scenario", saName, sc.label))
				sa := helpers.NewServiceAccount(saName, installNamespace)
				Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
				helpers.ExpectServiceAccountExists(ctx, saName, installNamespace)
				DeferCleanup(func() {
					By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
					_ = k8sClient.Delete(context.Background(), sa, client.PropagationPolicy(metav1.DeletePropagationForeground))
				})

				crbName := fmt.Sprintf("install-quay-bothns-%s-crb-%s", sc.id, suffix)
				By(fmt.Sprintf("creating ClusterRoleBinding %s for %s scenario", crbName, sc.label))
				crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, installNamespace)
				Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
				helpers.ExpectClusterRoleBindingExists(ctx, crbName)
				DeferCleanup(func() {
					By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", crb.Name))
					_ = k8sClient.Delete(context.Background(), crb, client.PropagationPolicy(metav1.DeletePropagationForeground))
				})

				ceName := fmt.Sprintf("install-quay-bothns-%s-ce-%s", sc.id, suffix)
				By(fmt.Sprintf("creating ClusterExtension %s for %s scenario", ceName, sc.label))
				ce := helpers.NewClusterExtensionObject("quay-operator", "3.14.2", ceName, saName, installNamespace)
				ce.Spec.Config = &olmv1.ClusterExtensionConfig{
					ConfigType: olmv1.ClusterExtensionConfigTypeInline,
					Inline: &apiextensionsv1.JSON{
						Raw: []byte(fmt.Sprintf(`{"watchNamespace": "%s"}`, watchNamespace)),
					},
				}
				Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
				DeferCleanup(func() {
					By(fmt.Sprintf("cleanup: deleting ClusterExtension %s", ce.Name))
					_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))
				})

				By(fmt.Sprintf("waiting for the ClusterExtension %s to be installed for %s scenario", ceName, sc.label))
				helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)

				By(fmt.Sprintf("verifying the operator deployment watch scope annotation for %s scenario", sc.label))
				Eventually(func(g Gomega) {
					deployments := &appsv1.DeploymentList{}
					err := k8sClient.List(ctx, deployments, client.InNamespace(installNamespace))
					g.Expect(err).ToNot(HaveOccurred(), "failed to list deployments in namespace %s", installNamespace)
					g.Expect(deployments.Items).ToNot(BeEmpty(), "expected at least one deployment in namespace %s", installNamespace)

					found := false
					for i := range deployments.Items {
						annotations := deployments.Items[i].Spec.Template.Annotations
						if annotations == nil {
							continue
						}
						if val, ok := annotations["olm.targetNamespaces"]; ok {
							g.Expect(val).To(Equal(watchNamespace), "unexpected watch scope annotation value")
							found = true
							break
						}
					}
					g.Expect(found).To(BeTrue(), "failed to find deployment with olm.targetNamespaces annotation")
				}).WithTimeout(5 * time.Minute).WithPolling(3 * time.Second).Should(Succeed())

				By(fmt.Sprintf("cleaning up resources created for %s scenario to allow next scenario", sc.label))
				deletePolicy := metav1.DeletePropagationForeground
				Expect(k8sClient.Delete(ctx, ce, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete ClusterExtension %q", ceName)
				helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

				Expect(k8sClient.Delete(ctx, crb, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete ClusterRoleBinding %q", crbName)
				Expect(k8sClient.Delete(ctx, sa, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete ServiceAccount %q", saName)

				if watchNSObj != nil {
					Expect(k8sClient.Delete(ctx, watchNSObj, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete watch namespace %q", watchNamespace)
				}
				Expect(k8sClient.Delete(ctx, installNS, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete install namespace %q", installNamespace)
			}
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

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation should reject invalid watch namespace configuration and update the status conditions accordingly", Ordered, Serial, func() {
	var (
		k8sClient  client.Client
		namespace  string
		testPrefix = "invalidwatch"
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

		By("ensuring no lingering ClusterExtensions or CRDs for quay-operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

		By(fmt.Sprintf("creating namespace %s for invalid watch namespace tests", namespace))
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

	// The controller validates the inline watchNamespace using the same DNS-1123 rules that gate namespace names.
	// Setting a trailing '-' produces an invalid identifier that cannot exist in the cluster, so the install should
	// fail fast and surface a failure through the Installed condition.
	It("should fail to install the ClusterExtension when watch namespace is invalid",
		func(ctx SpecContext) {
			By("creating ServiceAccount")
			sa := helpers.NewServiceAccount(saName, namespace)
			Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
			By("ensuring ServiceAccount is available before proceeding")
			helpers.ExpectServiceAccountExists(ctx, saName, namespace)
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
				_ = k8sClient.Delete(context.Background(), sa, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			By("creating ClusterRoleBinding")
			crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, namespace)
			Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
			By("ensuring ClusterRoleBinding is available before proceeding")
			helpers.ExpectClusterRoleBindingExists(ctx, crbName)
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", crb.Name))
				_ = k8sClient.Delete(context.Background(), crb, client.PropagationPolicy(metav1.DeletePropagationForeground))
			})

			invalidWatchNamespace := fmt.Sprintf("%s-", namespace)

			By("creating ClusterExtension with an invalid watch namespace configured")
			ce := helpers.NewClusterExtensionObject("quay-operator", "3.14.2", ceName, saName, namespace)
			ce.Spec.Config = &olmv1.ClusterExtensionConfig{
				ConfigType: olmv1.ClusterExtensionConfigTypeInline,
				Inline: &apiextensionsv1.JSON{
					Raw: []byte(fmt.Sprintf(`{"watchNamespace": "%s"}`, invalidWatchNamespace)),
				},
			}
			Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension %q", ceName)
			DeferCleanup(func() {
				By(fmt.Sprintf("cleanup: deleting ClusterExtension %s", ce.Name))
				_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))

				By("ensuring ClusterExtension is deleted")
				helpers.EnsureCleanupClusterExtension(context.Background(), ceName, namespace)
			})

			By("waiting for the ClusterExtension installation to fail due to invalid watch namespace")
			Eventually(func(g Gomega) {
				var ext olmv1.ClusterExtension
				err := k8sClient.Get(ctx, client.ObjectKey{Name: ceName}, &ext)
				g.Expect(err).ToNot(HaveOccurred(), "failed to get ClusterExtension %q", ceName)

				conditions := ext.Status.Conditions
				g.Expect(conditions).ToNot(BeEmpty(), "ClusterExtension %q has empty status.conditions", ceName)

				installed := meta.FindStatusCondition(conditions, olmv1.TypeInstalled)
				g.Expect(installed).ToNot(BeNil(), "Installed condition not found")
				g.Expect(installed.Status).To(Equal(metav1.ConditionFalse), "Installed should be False")
				g.Expect(installed.Reason).To(Equal(olmv1.ReasonFailed))
				g.Expect(installed.Message).ToNot(BeEmpty())
			}).WithTimeout(5 * time.Minute).WithPolling(3 * time.Second).Should(Succeed())
		})
})
