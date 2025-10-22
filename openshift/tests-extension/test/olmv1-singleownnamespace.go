package test

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"github.com/openshift/origin/test/extended/util/image"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	singleownbundle "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/singleown/bundle"
	singleownindex "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/singleown/index"
	webhookbundle "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/webhook/bundle"
	webhookindex "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/webhook/index"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace] OLMv1 operator installation support for singleNamespace watch mode with operator", func() {
	var (
		k8sClient   client.Client
		namespace   string
		testPrefix  = "singlens"
		catalogName string
		packageName string
		crdSuffix   string
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func(ctx SpecContext) {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient

		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		crdSuffix = unique

		// Build in-cluster bundle and catalog
		singleownImage := image.LocationFor("quay.io/olmtest/webhook-operator:v0.0.5")
		packageName = fmt.Sprintf("singleown-operator-single-%s", crdSuffix)
		By(fmt.Sprintf("using singleown operator image: %s, CRD suffix: %s, package: %s", singleownImage, crdSuffix, packageName))
		crdName := fmt.Sprintf("webhooktests-%s.webhook.operators.coreos.io", crdSuffix)
		helpers.EnsureCleanupClusterExtension(context.Background(), packageName, crdName)

		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ TEST-CONTROLLER }}": singleownImage,
			"{{ CRD-SUFFIX }}":      crdSuffix,
			"{{ PACKAGE-NAME }}":    packageName,
		}

		var nsName, opName string
		_, nsName, catalogName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			singleownindex.AssetNames, singleownindex.Asset,
			singleownbundle.AssetNames, singleownbundle.Asset,
		)
		By(fmt.Sprintf("singleown bundle %q and catalog %q built successfully in namespace %q", opName, catalogName, nsName))

		By(fmt.Sprintf("creating namespace %s for single-namespace tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create test namespace %q", namespace)
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
			ce := helpers.NewClusterExtensionObject(packageName, "0.0.5", ceName, saName, namespace)
			ce.Spec.Source.Catalog.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"olm.operatorframework.io/metadata.name": catalogName,
				},
			}
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

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace] OLMv1 operator installation support for ownNamespace watch mode with operator", func() {
	var (
		k8sClient   client.Client
		namespace   string
		testPrefix  = "ownns"
		catalogName string
		packageName string
		crdSuffix   string
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func(ctx SpecContext) {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient
		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		crdSuffix = unique

		// Build in-cluster bundle and catalog
		singleownImage := image.LocationFor("quay.io/olmtest/webhook-operator:v0.0.5")
		packageName = fmt.Sprintf("singleown-operator-own-%s", crdSuffix)
		By(fmt.Sprintf("using singleown operator image: %s, CRD suffix: %s, package: %s", singleownImage, crdSuffix, packageName))
		crdName := fmt.Sprintf("webhooktests-%s.webhook.operators.coreos.io", crdSuffix)
		helpers.EnsureCleanupClusterExtension(context.Background(), packageName, crdName)

		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ TEST-CONTROLLER }}": singleownImage,
			"{{ CRD-SUFFIX }}":      crdSuffix,
			"{{ PACKAGE-NAME }}":    packageName,
		}

		var nsName, opName string
		_, nsName, catalogName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			singleownindex.AssetNames, singleownindex.Asset,
			singleownbundle.AssetNames, singleownbundle.Asset,
		)
		By(fmt.Sprintf("singleown bundle %q and catalog %q built successfully in namespace %q", opName, catalogName, nsName))

		By(fmt.Sprintf("creating namespace %s for own-namespace tests", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create test namespace %q", namespace)
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
			ce := helpers.NewClusterExtensionObject(packageName, "0.0.5", ceName, saName, namespace)
			ce.Spec.Source.Catalog.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"olm.operatorframework.io/metadata.name": catalogName,
				},
			}
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

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace] OLMv1 operator installation support for ownNamespace and single namespace watch mode with operator", func() {
	var (
		k8sClient        client.Client
		activeNamespaces map[string]struct{}
		singleownImage   string
	)

	BeforeEach(func(ctx SpecContext) {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient
		activeNamespaces = map[string]struct{}{}
		singleownImage = image.LocationFor("quay.io/olmtest/webhook-operator:v0.0.5")
		By(fmt.Sprintf("using singleown operator image: %s", singleownImage))
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			for ns := range activeNamespaces {
				helpers.DescribeAllClusterExtensions(ctx, ns)
			}
		}
	})

	It("should install cluster extensions successfully in both watch modes",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation support for ownNamespace and single namespace watch mode with quay-operator should install cluster extensions successfully in both watch modes"),
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
				installNamespace := fmt.Sprintf("olmv1-webhook-bothns-%s-%s", sc.id, suffix)
				watchNamespace := sc.watchN(installNamespace)

				activeNamespaces[installNamespace] = struct{}{}
				if watchNamespace != installNamespace {
					activeNamespaces[watchNamespace] = struct{}{}
				}

				// Ensure unique names per scenario
				crdSuffix := rand.String(4)
				packageName := fmt.Sprintf("singleown-operator-both-%s", crdSuffix)
				By(fmt.Sprintf("building singleown operator assets for %s scenario: image=%s, CRD suffix=%s, package=%s", sc.label, singleownImage, crdSuffix, packageName))

				replacements := map[string]string{
					"{{ TEST-BUNDLE }}":                        "",
					"{{ NAMESPACE }}":                          "",
					"{{ TEST-CONTROLLER }}":                    singleownImage,
					"{{ CRD-SUFFIX }}":                         crdSuffix, // Unique CRD suffix per scenario
					"{{ PACKAGE-NAME }}":                       packageName,
					"webhook-operator-webhooktest-admin-role":  fmt.Sprintf("webhook-operator-webhooktest-admin-role-%s", crdSuffix),
					"webhook-operator-webhooktest-editor-role": fmt.Sprintf("webhook-operator-webhooktest-editor-role-%s", crdSuffix),
					"webhook-operator-webhooktest-viewer-role": fmt.Sprintf("webhook-operator-webhooktest-viewer-role-%s", crdSuffix),
					"webhook-operator-metrics-reader":          fmt.Sprintf("webhook-operator-metrics-reader-%s", crdSuffix),
				}

				_, nsName, catalogName, opName := helpers.NewCatalogAndClusterBundles(ctx, replacements,
					singleownindex.AssetNames, singleownindex.Asset,
					singleownbundle.AssetNames, singleownbundle.Asset,
				)
				By(fmt.Sprintf("singleown bundle %q and catalog %q built successfully in namespace %q for %s scenario", opName, catalogName, nsName, sc.label))

				By(fmt.Sprintf("ensuring no ClusterExtension for %s before %s scenario", packageName, sc.label))
				crdName := fmt.Sprintf("webhooktests-%s.webhook.operators.coreos.io", crdSuffix)
				helpers.EnsureCleanupClusterExtension(context.Background(), packageName, crdName)

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

				saName := fmt.Sprintf("install-webhook-bothns-%s-sa-%s", sc.id, suffix)
				By(fmt.Sprintf("creating ServiceAccount %s for %s scenario", saName, sc.label))
				sa := helpers.NewServiceAccount(saName, installNamespace)
				Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount %q", saName)
				helpers.ExpectServiceAccountExists(ctx, saName, installNamespace)
				DeferCleanup(func() {
					By(fmt.Sprintf("cleanup: deleting ServiceAccount %s in namespace %s", sa.Name, sa.Namespace))
					_ = k8sClient.Delete(context.Background(), sa, client.PropagationPolicy(metav1.DeletePropagationForeground))
				})

				crbName := fmt.Sprintf("install-webhook-bothns-%s-crb-%s", sc.id, suffix)
				By(fmt.Sprintf("creating ClusterRoleBinding %s for %s scenario", crbName, sc.label))
				crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, installNamespace)
				Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding %q", crbName)
				helpers.ExpectClusterRoleBindingExists(ctx, crbName)
				DeferCleanup(func() {
					By(fmt.Sprintf("cleanup: deleting ClusterRoleBinding %s", crb.Name))
					_ = k8sClient.Delete(context.Background(), crb, client.PropagationPolicy(metav1.DeletePropagationForeground))
				})

				ceName := fmt.Sprintf("install-webhook-bothns-%s-ce-%s", sc.id, suffix)
				By(fmt.Sprintf("creating ClusterExtension %s for %s scenario", ceName, sc.label))
				ce := helpers.NewClusterExtensionObject(packageName, "0.0.5", ceName, saName, installNamespace)
				ce.Spec.Source.Catalog.Selector = &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"olm.operatorframework.io/metadata.name": catalogName,
					},
				}
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
					if crdName != "" {
						crd := &apiextensionsv1.CustomResourceDefinition{}
						if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: crdName}, crd); err == nil {
							By(fmt.Sprintf("cleanup: deleting CRD %s", crdName))
							_ = k8sClient.Delete(context.Background(), crd)
						}
					}
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
				}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

				// Ginkgo never invokes those deferred cleanups until we exit the whole spec, so the first scenario’s
				// cluster resources survive long enough to collide with the second scenario.
				By(fmt.Sprintf("cleaning up resources created for %s scenario to allow next scenario", sc.label))
				deletePolicy := metav1.DeletePropagationForeground
				Expect(k8sClient.Delete(ctx, ce, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete ClusterExtension %q", ceName)
				helpers.EnsureCleanupClusterExtension(context.Background(), packageName, crdName)

				Expect(k8sClient.Delete(ctx, crb, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete ClusterRoleBinding %q", crbName)
				Expect(k8sClient.Delete(ctx, sa, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete ServiceAccount %q", saName)

				if watchNSObj != nil {
					Expect(k8sClient.Delete(ctx, watchNSObj, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete watch namespace %q", watchNamespace)
				}
				Expect(k8sClient.Delete(ctx, installNS, client.PropagationPolicy(deletePolicy))).To(Succeed(), "failed to delete install namespace %q", installNamespace)

				By(fmt.Sprintf("waiting for namespace %s to be fully deleted before next scenario", installNamespace))
				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKey{Name: installNamespace}, &corev1.Namespace{})
					return errors.IsNotFound(err)
				}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(BeTrue(), "expected namespace %s to be deleted", installNamespace)
				if watchNSObj != nil {
					By(fmt.Sprintf("waiting for namespace %s to be fully deleted before next scenario", watchNamespace))
					Eventually(func() bool {
						err := k8sClient.Get(ctx, client.ObjectKey{Name: watchNamespace}, &corev1.Namespace{})
						return errors.IsNotFound(err)
					}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(BeTrue(), "expected namespace %s to be deleted", watchNamespace)
				}
			}
		})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Serial] OLMv1 operator installation support for ownNamespace watch mode with an operator that does not support ownNamespace installation mode", Serial, func() {
	var (
		k8sClient   client.Client
		namespace   string
		testPrefix  = "pipelines"
		catalogName string
		crdSuffix   string
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func(ctx SpecContext) {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient
		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		// Build in-cluster bundle and catalog using webhook testdata (supports AllNamespaces mode only)
		webhookImage := image.LocationFor("quay.io/olmtest/webhook-operator:v0.0.5")
		By(fmt.Sprintf("using webhook operator image: %s, CRD suffix: %s", webhookImage, crdSuffix))

		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ TEST-CONTROLLER }}": webhookImage,
		}

		var nsName, opName string
		_, nsName, catalogName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			webhookindex.AssetNames, webhookindex.Asset,
			webhookbundle.AssetNames, webhookbundle.Asset,
		)
		By(fmt.Sprintf("webhook bundle %q and catalog %q built successfully in namespace %q", opName, catalogName, nsName))

		By("ensuring no ClusterExtension for webhook-operator")
		crdName := "webhooktests.webhook.operators.coreos.io"
		helpers.EnsureCleanupClusterExtension(context.Background(), "webhook-operator", crdName)

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

			By("creating ClusterExtension with the watch-namespace configured using webhook operator that only supports AllNamespaces mode")
			ce := helpers.NewClusterExtensionObject("webhook-operator", "0.0.5", ceName, saName, namespace)
			ce.Spec.Source.Catalog.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"olm.operatorframework.io/metadata.name": catalogName,
				},
			}
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
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
		})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace] OLMv1 operator installation should reject invalid watch namespace configuration and update the status conditions accordingly", func() {
	var (
		k8sClient   client.Client
		namespace   string
		testPrefix  = "invalidwatch"
		catalogName string
		packageName string
		crdSuffix   string
	)

	var unique, saName, crbName, ceName string
	BeforeEach(func(ctx SpecContext) {
		By("checking if OpenShift is available for tests")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient
		unique = rand.String(4)
		namespace = fmt.Sprintf("olmv1-%s-ns-%s", testPrefix, unique)
		saName = fmt.Sprintf("install-%s-sa-%s", testPrefix, unique)
		crbName = fmt.Sprintf("install-%s-crb-%s", testPrefix, unique)
		ceName = fmt.Sprintf("install-%s-ce-%s", testPrefix, unique)

		// Generate unique CRD suffix for parallel execution
		crdSuffix = unique

		// Build in-cluster bundle and catalog
		singleownImage := image.LocationFor("quay.io/olmtest/webhook-operator:v0.0.5")
		packageName = fmt.Sprintf("singleown-operator-%s", crdSuffix)
		By(fmt.Sprintf("using singleown operator image: %s, CRD suffix: %s, package: %s", singleownImage, crdSuffix, packageName))

		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ TEST-CONTROLLER }}": singleownImage,
			"{{ CRD-SUFFIX }}":      crdSuffix,
			"{{ PACKAGE-NAME }}":    packageName,
		}

		var nsName, opName string
		_, nsName, catalogName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			singleownindex.AssetNames, singleownindex.Asset,
			singleownbundle.AssetNames, singleownbundle.Asset,
		)
		By(fmt.Sprintf("singleown bundle %q and catalog %q built successfully in namespace %q", opName, catalogName, nsName))

		By(fmt.Sprintf("ensuring no lingering ClusterExtensions for %s", packageName))
		crdName := fmt.Sprintf("webhooktests-%s.webhook.operators.coreos.io", crdSuffix)
		helpers.EnsureCleanupClusterExtension(context.Background(), packageName, crdName)

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
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMOwnSingleNamespace][Skipped:Disconnected][Serial] OLMv1 operator installation should reject invalid watch namespace configuration and update the status conditions accordingly should fail to install the ClusterExtension when watch namespace is invalid"),
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
			ce := helpers.NewClusterExtensionObject(packageName, "0.0.5", ceName, saName, namespace)
			ce.Spec.Source.Catalog.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"olm.operatorframework.io/metadata.name": catalogName,
				},
			}
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
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
		})
})
