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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	catalogdata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/catalog"
	operatordata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/operator"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

// These tests verify the DeploymentConfig feature: operator deployments are
// customised via spec.config.inline.deploymentConfig in the ClusterExtension.
// The whole suite is gated on [OCPFeatureGate:NewOLMConfigAPI] so it is
// skipped automatically when that OCP feature gate is not enabled.
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support", Ordered, Serial, func() {
	var (
		k8sClient client.Client
		nsName    string
		ccName    string
		opName    string
		unique    string
	)

	BeforeAll(func(ctx SpecContext) {
		By("checking prerequisites")
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for the tests")
		}
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient

		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ VERSION }}":         env.Get().OpenShiftVersion,
			"{{ TEST-CONTROLLER }}": image.ShellImage(),
		}
		unique, nsName, ccName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			catalogdata.AssetNames, catalogdata.Asset,
			operatordata.AssetNames, operatordata.Asset,
		)
		By(fmt.Sprintf("catalog %q and operator bundle %q ready in namespace %q", ccName, opName, nsName))
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping diagnostics")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), nsName)
		}
	})

	// installAndVerify is a helper that creates an install namespace, ServiceAccount,
	// ClusterRoleBinding and ClusterExtension, waits for successful installation, and
	// then calls verify against the resulting DeploymentList.  All resources are
	// cleaned up via DeferCleanup.
	installAndVerify := func(
		ctx SpecContext,
		namePrefix string,
		inlineConfig string,
		verify func(Gomega, []appsv1.Deployment),
	) {
		suffix := rand.String(4)
		installNamespace := fmt.Sprintf("olmv1-%s-%s-%s", namePrefix, unique, suffix)

		By(fmt.Sprintf("creating install namespace %s", installNamespace))
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: installNamespace}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create namespace %q", installNamespace)
		DeferCleanup(func() {
			_ = k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})

		saName := fmt.Sprintf("dc-%s-sa-%s", namePrefix, suffix)
		crbName := fmt.Sprintf("dc-%s-crb-%s", namePrefix, suffix)
		ceName := fmt.Sprintf("dc-%s-ce-%s", namePrefix, suffix)

		sa := helpers.NewServiceAccount(saName, installNamespace)
		Expect(k8sClient.Create(ctx, sa)).To(Succeed())
		helpers.ExpectServiceAccountExists(ctx, saName, installNamespace)
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), sa) })

		crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, installNamespace)
		Expect(k8sClient.Create(ctx, crb)).To(Succeed())
		helpers.ExpectClusterRoleBindingExists(ctx, crbName)
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), crb) })

		ce := helpers.NewClusterExtensionObject(opName, "", ceName, saName, installNamespace,
			helpers.WithCatalogNameSelector(ccName))
		ce.Spec.Config = &olmv1.ClusterExtensionConfig{
			ConfigType: olmv1.ClusterExtensionConfigTypeInline,
			Inline:     &apiextensionsv1.JSON{Raw: []byte(inlineConfig)},
		}
		Expect(k8sClient.Create(ctx, ce)).To(Succeed())
		DeferCleanup(func() {
			_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))
			helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")
		})

		By("waiting for ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)

		By("verifying deployment customisations")
		Eventually(func(g Gomega) {
			deployments := &appsv1.DeploymentList{}
			g.Expect(k8sClient.List(ctx, deployments, client.InNamespace(installNamespace))).To(Succeed())
			g.Expect(deployments.Items).NotTo(BeEmpty(), "expected at least one deployment in %q", installNamespace)
			verify(g, deployments.Items)
		}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
	}

	It("should apply environment variables from deploymentConfig to operator deployment containers",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should apply environment variables from deploymentConfig to operator deployment containers"),
		func(ctx SpecContext) {
			installAndVerify(ctx, "env",
				`{"deploymentConfig":{"env":[{"name":"TEST_DEPLOY_CFG_ENV","value":"test-value-from-deploymentconfig"}]}}`,
				func(g Gomega, deps []appsv1.Deployment) {
					found := false
					for _, dep := range deps {
						for _, c := range dep.Spec.Template.Spec.Containers {
							for _, e := range c.Env {
								if e.Name == "TEST_DEPLOY_CFG_ENV" && e.Value == "test-value-from-deploymentconfig" {
									found = true
								}
							}
						}
					}
					g.Expect(found).To(BeTrue(), "env var TEST_DEPLOY_CFG_ENV=test-value-from-deploymentconfig not found in any container")
				},
			)
		})

	It("should apply resource requirements from deploymentConfig to operator deployment containers",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should apply resource requirements from deploymentConfig to operator deployment containers"),
		func(ctx SpecContext) {
			installAndVerify(ctx, "res",
				`{"deploymentConfig":{"resources":{"requests":{"cpu":"50m","memory":"64Mi"},"limits":{"cpu":"200m","memory":"128Mi"}}}}`,
				func(g Gomega, deps []appsv1.Deployment) {
					found := false
					for _, dep := range deps {
						for _, c := range dep.Spec.Template.Spec.Containers {
							if c.Resources.Requests != nil {
								if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok && cpu.String() == "50m" {
									found = true
								}
							}
						}
					}
					g.Expect(found).To(BeTrue(), "resource request cpu=50m not found in any deployment container")
				},
			)
		})

	It("should apply tolerations from deploymentConfig to operator deployment pods",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should apply tolerations from deploymentConfig to operator deployment pods"),
		func(ctx SpecContext) {
			// The toleration key used here does not exist as a taint on any node, so
			// it is purely additive and does not affect scheduling.
			installAndVerify(ctx, "tol",
				`{"deploymentConfig":{"tolerations":[{"key":"dc-test-taint","operator":"Exists","effect":"NoSchedule"}]}}`,
				func(g Gomega, deps []appsv1.Deployment) {
					found := false
					for _, dep := range deps {
						for _, t := range dep.Spec.Template.Spec.Tolerations {
							if t.Key == "dc-test-taint" &&
								t.Operator == corev1.TolerationOpExists &&
								t.Effect == corev1.TaintEffectNoSchedule {
								found = true
							}
						}
					}
					g.Expect(found).To(BeTrue(), "toleration key=dc-test-taint,operator=Exists,effect=NoSchedule not found")
				},
			)
		})

	It("should apply node selector from deploymentConfig to operator deployment pods",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should apply node selector from deploymentConfig to operator deployment pods"),
		func(ctx SpecContext) {
			// kubernetes.io/os=linux is present on every node so the deployment remains schedulable.
			installAndVerify(ctx, "nsel",
				`{"deploymentConfig":{"nodeSelector":{"kubernetes.io/os":"linux"}}}`,
				func(g Gomega, deps []appsv1.Deployment) {
					found := false
					for _, dep := range deps {
						if val, ok := dep.Spec.Template.Spec.NodeSelector["kubernetes.io/os"]; ok && val == "linux" {
							found = true
						}
					}
					g.Expect(found).To(BeTrue(), "nodeSelector kubernetes.io/os=linux not found in any deployment pod spec")
				},
			)
		})

	// installAndExpectBlocked is a helper for negative tests: it creates the same set of
	// resources as installAndVerify but expects the ClusterExtension to reach a terminal
	// failure (Progressing=False, Reason=Blocked) whose message contains all of msgSubstrings.
	installAndExpectBlocked := func(
		ctx SpecContext,
		namePrefix string,
		inlineConfig string,
		msgSubstrings ...string,
	) {
		suffix := rand.String(4)
		installNamespace := fmt.Sprintf("olmv1-%s-%s-%s", namePrefix, unique, suffix)

		By(fmt.Sprintf("creating install namespace %s", installNamespace))
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: installNamespace}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create namespace %q", installNamespace)
		DeferCleanup(func() {
			_ = k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		})

		saName := fmt.Sprintf("dc-%s-sa-%s", namePrefix, suffix)
		crbName := fmt.Sprintf("dc-%s-crb-%s", namePrefix, suffix)
		ceName := fmt.Sprintf("dc-%s-ce-%s", namePrefix, suffix)

		sa := helpers.NewServiceAccount(saName, installNamespace)
		Expect(k8sClient.Create(ctx, sa)).To(Succeed())
		helpers.ExpectServiceAccountExists(ctx, saName, installNamespace)
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), sa) })

		crb := helpers.NewClusterRoleBinding(crbName, "cluster-admin", saName, installNamespace)
		Expect(k8sClient.Create(ctx, crb)).To(Succeed())
		helpers.ExpectClusterRoleBindingExists(ctx, crbName)
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), crb) })

		ce := helpers.NewClusterExtensionObject(opName, "", ceName, saName, installNamespace,
			helpers.WithCatalogNameSelector(ccName))
		ce.Spec.Config = &olmv1.ClusterExtensionConfig{
			ConfigType: olmv1.ClusterExtensionConfigTypeInline,
			Inline:     &apiextensionsv1.JSON{Raw: []byte(inlineConfig)},
		}
		Expect(k8sClient.Create(ctx, ce)).To(Succeed())
		DeferCleanup(func() {
			_ = k8sClient.Delete(context.Background(), ce, client.PropagationPolicy(metav1.DeletePropagationForeground))
			helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")
		})

		By("waiting for ClusterExtension to reach a terminal blocked state")
		Eventually(func(g Gomega) {
			var ext olmv1.ClusterExtension
			g.Expect(k8sClient.Get(ctx, client.ObjectKey{Name: ceName}, &ext)).To(Succeed())
			progressing := apimeta.FindStatusCondition(ext.Status.Conditions, olmv1.TypeProgressing)
			g.Expect(progressing).NotTo(BeNil(), "Progressing condition not found")
			g.Expect(progressing.Status).To(Equal(metav1.ConditionFalse), "expected Progressing=False for terminal error")
			g.Expect(progressing.Reason).To(Equal(olmv1.ReasonInvalidConfiguration), "expected Reason=InvalidConfiguration for config validation error")
			for _, sub := range msgSubstrings {
				g.Expect(progressing.Message).To(ContainSubstring(sub),
					"expected message to contain %q, got: %s", sub, progressing.Message)
			}
		}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
	}

	It("should reach a terminal blocked state when deploymentConfig.env has an invalid type",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should reach a terminal blocked state when deploymentConfig.env has an invalid type"),
		func(ctx SpecContext) {
			// Schema validation requires env to be an array; passing a string causes a terminal error.
			installAndExpectBlocked(ctx, "inv-env",
				`{"deploymentConfig":{"env":"not-an-array"}}`,
				"invalid ClusterExtension configuration",
				"deploymentConfig.env",
			)
		})

	It("should reach a terminal blocked state when deploymentConfig contains an unknown field",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should reach a terminal blocked state when deploymentConfig contains an unknown field"),
		func(ctx SpecContext) {
			// The JSON schema for deploymentConfig has additionalProperties:false, so
			// unknown fields produce a terminal validation error.
			installAndExpectBlocked(ctx, "unk-field",
				`{"deploymentConfig":{"bogusUnknownField":"some-value"}}`,
				"invalid ClusterExtension configuration",
				"bogusUnknownField",
			)
		})

	It("should apply annotations from deploymentConfig to operator deployment and its pod template",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLMConfigAPI][Skipped:Disconnected] OLMv1 DeploymentConfig support should apply annotations from deploymentConfig to operator deployment and its pod template"),
		func(ctx SpecContext) {
			installAndVerify(ctx, "ann",
				`{"deploymentConfig":{"annotations":{"test-dc-annotation":"test-dc-annotation-value"}}}`,
				func(g Gomega, deps []appsv1.Deployment) {
					foundOnDeployment := false
					foundOnPodTemplate := false
					for _, dep := range deps {
						if val, ok := dep.Annotations["test-dc-annotation"]; ok && val == "test-dc-annotation-value" {
							foundOnDeployment = true
						}
						if val, ok := dep.Spec.Template.Annotations["test-dc-annotation"]; ok && val == "test-dc-annotation-value" {
							foundOnPodTemplate = true
						}
					}
					g.Expect(foundOnDeployment).To(BeTrue(), "annotation test-dc-annotation not found on any deployment")
					g.Expect(foundOnPodTemplate).To(BeTrue(), "annotation test-dc-annotation not found on any deployment pod template")
				},
			)
		})
})
