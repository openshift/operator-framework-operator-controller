package test

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"github.com/openshift/api/features"
	"github.com/openshift/origin/test/extended/util/image"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	singleownbundle "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/singleown/bundle"
	singleownindex "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/singleown/index"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

type preflightAuthTestScenario int

const (
	scenarioMissingServicePerms                            preflightAuthTestScenario = 0
	scenarioMissingCreateVerb                              preflightAuthTestScenario = 1
	scenarioMissingClusterRoleBindingsPerms                preflightAuthTestScenario = 2
	scenarioMissingNamedConfigMapPerms                     preflightAuthTestScenario = 3
	scenarioMissingClusterExtensionsFinalizerPerms         preflightAuthTestScenario = 4
	scenarioMissingEscalateAndBindPerms                    preflightAuthTestScenario = 5
	scenarioMissingClusterExtensionRevisionsFinalizerPerms preflightAuthTestScenario = 6
)

const preflightBundleVersion = "0.0.5"

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMPreflightPermissionChecks][Skipped:Disconnected] OLMv1 operator preflight checks", func() {
	var (
		namespace   string
		k8sClient   client.Client
		catalogName string
		packageName string
	)
	BeforeEach(func(ctx SpecContext) {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		helpers.RequireImageRegistry(ctx)
		k8sClient = env.Get().K8sClient
		namespace = "preflight-test-ns-" + rand.String(4)

		// Use an in-cluster catalog and bundle so tests do not depend on external indexes.
		crdSuffix := rand.String(4)
		packageName = fmt.Sprintf("preflight-operator-%s", crdSuffix)
		crdName := fmt.Sprintf("webhooktests-%s.webhook.operators.coreos.io", crdSuffix)
		helpers.EnsureCleanupClusterExtension(context.Background(), packageName, crdName)

		singleownImage := image.LocationFor("quay.io/olmtest/webhook-operator:v0.0.5")
		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ TEST-CONTROLLER }}": singleownImage,
			"{{ CRD-SUFFIX }}":      crdSuffix,
			"{{ PACKAGE-NAME }}":    packageName,
		}
		_, _, catalogName, _ = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			singleownindex.AssetNames, singleownindex.Asset,
			singleownbundle.AssetNames, singleownbundle.Asset,
		)
		By(fmt.Sprintf("catalog %q and package %q are ready", catalogName, packageName))

		By(fmt.Sprintf("creating namespace %s", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace")
		DeferCleanup(func() {
			_ = k8sClient.Delete(context.Background(), ns)
		})
	})

	It("should report error when {services} are not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, scenarioMissingServicePerms, namespace, packageName, catalogName)
	})

	It("should report error when {create} verb is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, scenarioMissingCreateVerb, namespace, packageName, catalogName)
	})

	It("should report error when {ClusterRoleBindings} are not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, scenarioMissingClusterRoleBindingsPerms, namespace, packageName, catalogName)
	})

	It("should report error when {ConfigMap:resourceNames} are not all specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, scenarioMissingNamedConfigMapPerms, namespace, packageName, catalogName)
	})

	It("should report error when {clusterextension/finalizer} is not specified", func(ctx SpecContext) {
		helpers.RequireFeatureGateDisabled(features.FeatureGateNewOLMBoxCutterRuntime)
		runNegativePreflightTest(ctx, scenarioMissingClusterExtensionsFinalizerPerms, namespace, packageName, catalogName)
	})

	It("should report error when {clusterextensionrevisions/finalizer} is not specified", func(ctx SpecContext) {
		helpers.RequireFeatureGateEnabled(features.FeatureGateNewOLMBoxCutterRuntime)
		runNegativePreflightTest(ctx, scenarioMissingClusterExtensionRevisionsFinalizerPerms, namespace, packageName, catalogName)
	})

	It("should report error when {escalate, bind} is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, scenarioMissingEscalateAndBindPerms, namespace, packageName, catalogName)
	})
})

// runNegativePreflightTest creates a ClusterRole that is missing one required permission,
// a ClusterExtension that uses it (via the in-cluster catalog), then waits for the preflight failure.
func runNegativePreflightTest(ctx context.Context, scenario preflightAuthTestScenario, namespace, packageName, catalogName string) {
	k8sClient := env.Get().K8sClient
	unique := rand.String(8)

	// Define names
	crName := fmt.Sprintf("install-test-cr-%s", unique)
	saName := fmt.Sprintf("install-test-sa-%s", unique)
	crbName := fmt.Sprintf("install-test-crb-%s", unique)
	ceName := fmt.Sprintf("install-test-ce-%s", unique)

	// Step 1: Create deficient ClusterRole
	defCR := createDeficientClusterRole(scenario, crName, ceName)
	Expect(k8sClient.Create(ctx, defCR)).To(Succeed(), "failed to create ClusterRole")
	DeferCleanup(func(ctx SpecContext) {
		_ = k8sClient.Delete(ctx, defCR)
	})

	// Step 2: Create matching ServiceAccount
	sa := helpers.NewServiceAccount(saName, namespace)
	Expect(k8sClient.Create(ctx, sa)).To(Succeed(), "failed to create ServiceAccount")
	DeferCleanup(func(ctx SpecContext) {
		_ = k8sClient.Delete(ctx, sa)
	})

	// Step 3: Bind SA to the deficient ClusterRole
	crb := helpers.NewClusterRoleBinding(crbName, crName, saName, namespace)
	Expect(k8sClient.Create(ctx, crb)).To(Succeed(), "failed to create ClusterRoleBinding")
	DeferCleanup(func(ctx SpecContext) {
		_ = k8sClient.Delete(ctx, crb)
	})

	// Step 4: Create ClusterExtension for that SA using the in-cluster catalog.
	// Set watchNamespace in config so the controller can run preflight; otherwise it fails on config validation first.
	ce := helpers.NewClusterExtensionObject(packageName, preflightBundleVersion, ceName, saName, namespace, helpers.WithCatalogNameSelector(catalogName))
	ce.Spec.Config = &olmv1.ClusterExtensionConfig{
		ConfigType: "Inline",
		Inline: &apiextensionsv1.JSON{
			Raw: []byte(fmt.Sprintf(`{"watchNamespace": "%s"}`, namespace)),
		},
	}
	Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension")
	DeferCleanup(func(ctx SpecContext) {
		_ = k8sClient.Delete(ctx, ce)
	})

	// Step 5: Wait for the controller to report preflight failure.
	// The error is in the Progressing condition. We only check the message, not True/False, so the test stays stable.
	By("waiting for ClusterExtension to report preflight failure")
	Eventually(func(g Gomega) {
		latest := &olmv1.ClusterExtension{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: ce.Name}, latest)
		g.Expect(err).NotTo(HaveOccurred())

		c := meta.FindStatusCondition(latest.Status.Conditions, olmv1.TypeProgressing)
		g.Expect(c).NotTo(BeNil(), "Progressing condition should be set")
		g.Expect(c.Message).To(ContainSubstring("pre-authorization failed"), "message should report pre-authorization failure")
	}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
}

// createDeficientClusterRole returns a ClusterRole that is missing one permission needed by the test scenario.
func createDeficientClusterRole(scenario preflightAuthTestScenario, name, ceName string) *rbacv1.ClusterRole {
	var baseRules []rbacv1.PolicyRule
	if helpers.IsFeatureGateEnabled(features.FeatureGateNewOLMBoxCutterRuntime) {
		baseRules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"olm.operatorframework.io"},
				Resources:     []string{"clusterextensionrevisions/finalizers"},
				Verbs:         []string{"update"},
				ResourceNames: []string{ceName},
			},
		}
	} else {
		baseRules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"olm.operatorframework.io"},
				Resources:     []string{"clusterextensions/finalizers"},
				Verbs:         []string{"update"},
				ResourceNames: []string{ceName},
			},
		}
	}

	baseRules = append(baseRules, []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "pods/finalizers", "services", "services/finalizers", "endpoints", "endpoints/finalizers", "persistentvolumeclaims", "persistentvolumeclaims/finalizers", "events", "events/finalizers", "configmaps", "configmaps/finalizers", "secrets", "secrets/finalizers", "pods/log", "limitranges", "limitranges/finalizers", "namespaces", "namespaces/finalizers", "serviceaccounts", "serviceaccounts/finalizers"},
			Verbs:     []string{"delete", "deletecollection", "create", "patch", "get", "list", "update", "watch"},
		},
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"clusterroles", "clusterroles/finalizers", "roles", "roles/finalizers", "clusterrolebindings", "clusterrolebindings/finalizers", "rolebindings", "rolebindings/finalizers"},
			Verbs:     []string{"delete", "deletecollection", "create", "patch", "get", "list", "update", "watch", "bind", "escalate"},
		},
	}...)

	// Copy rules to avoid mutation
	rules := make([]rbacv1.PolicyRule, len(baseRules))
	copy(rules, baseRules)

	switch scenario {
	case scenarioMissingServicePerms:
		// Remove services and services/finalizers so preflight fails.
		for i, r := range rules {
			if r.APIGroups[0] == "" {
				filtered := []string{}
				for _, res := range r.Resources {
					if res != "services" && res != "services/finalizers" {
						filtered = append(filtered, res)
					}
				}
				rules[i].Resources = filtered
			}
		}
	case scenarioMissingCreateVerb:
		// Remove the create verb so preflight fails.
		for i, r := range rules {
			if r.APIGroups[0] == "" {
				filtered := []string{}
				for _, v := range r.Verbs {
					if v != "create" {
						filtered = append(filtered, v)
					}
				}
				rules[i].Verbs = filtered
			}
		}
	case scenarioMissingClusterRoleBindingsPerms:
		// Remove clusterrolebindings so preflight fails.
		for i, r := range rules {
			if r.APIGroups[0] == "rbac.authorization.k8s.io" {
				filtered := []string{}
				for _, res := range r.Resources {
					if res != "clusterrolebindings" && res != "clusterrolebindings/finalizers" {
						filtered = append(filtered, res)
					}
				}
				rules[i].Resources = filtered
			}
		}
	case scenarioMissingNamedConfigMapPerms:
		// Allow only one ClusterRole by name so the SA cannot manage the rest; preflight then fails.
		// The singleown bundle uses ClusterRoles like webhook-operator-metrics-reader.
		for i := range rules {
			if rules[i].APIGroups[0] == "rbac.authorization.k8s.io" {
				filtered := []string{}
				for _, res := range rules[i].Resources {
					if res != "clusterroles" && res != "clusterroles/finalizers" {
						filtered = append(filtered, res)
					}
				}
				rules[i].Resources = filtered
				rules = append(rules, rbacv1.PolicyRule{
					APIGroups:     []string{"rbac.authorization.k8s.io"},
					Resources:     []string{"clusterroles", "clusterroles/finalizers"},
					Verbs:         []string{"delete", "deletecollection", "create", "patch", "get", "list", "update", "watch"},
					ResourceNames: []string{"webhook-operator-metrics-reader"},
				})
				break
			}
		}
	case scenarioMissingClusterExtensionsFinalizerPerms:
		// Remove permission for clusterextensions/finalizers so preflight fails.
		filtered := []rbacv1.PolicyRule{}
		for _, r := range rules {
			if len(r.APIGroups) != 1 || r.APIGroups[0] != "olm.operatorframework.io" {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	case scenarioMissingClusterExtensionRevisionsFinalizerPerms:
		// Remove permission for clusterextensionrevisions/finalizers so preflight fails.
		filtered := []rbacv1.PolicyRule{}
		for _, r := range rules {
			if len(r.APIGroups) != 1 || r.APIGroups[0] != "olm.operatorframework.io" {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	case scenarioMissingEscalateAndBindPerms:
		// Remove bind and escalate verbs so preflight fails.
		for i, r := range rules {
			if r.APIGroups[0] == "rbac.authorization.k8s.io" {
				filtered := []string{}
				for _, v := range r.Verbs {
					if v != "bind" && v != "escalate" {
						filtered = append(filtered, v)
					}
				}
				rules[i].Verbs = filtered
			}
		}
	}

	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Rules:      rules,
	}
}
