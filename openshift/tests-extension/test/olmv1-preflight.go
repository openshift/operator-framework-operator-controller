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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMPreflightPermissionChecks][Skipped:Disconnected] OLMv1 operator preflight checks", func() {
	var (
		namespace string
		k8sClient client.Client
	)
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient
		namespace = "preflight-test-ns-" + rand.String(4)

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
		runNegativePreflightTest(ctx, 0, namespace)
	})

	It("should report error when {create} verb is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, 1, namespace)
	})

	It("should report error when {ClusterRoleBindings} are not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, 2, namespace)
	})

	It("should report error when {ConfigMap:resourceNames} are not all specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, 3, namespace)
	})

	It("should report error when {clusterextension/finalizer} is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, 4, namespace)
	})

	It("should report error when {escalate, bind} is not specified", func(ctx SpecContext) {
		runNegativePreflightTest(ctx, 5, namespace)
	})
})

// runNegativePreflightTest creates a deficient ClusterRole and a ClusterExtension that
// relies on it, then waits for the expected preflight failure.
func runNegativePreflightTest(ctx context.Context, scenario int, namespace string) {
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

	// Step 4: Create ClusterExtension referencing that SA
	ce := helpers.NewClusterExtensionObject("openshift-pipelines-operator-rh", "1.15.0", ceName, saName, namespace)
	Expect(k8sClient.Create(ctx, ce)).To(Succeed(), "failed to create ClusterExtension")
	DeferCleanup(func(ctx SpecContext) {
		_ = k8sClient.Delete(ctx, ce)
	})

	// Step 5: Wait for failure
	By("waiting for ClusterExtension to report preflight failure")
	Eventually(func(g Gomega) {
		latest := &olmv1.ClusterExtension{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: ce.Name}, latest)
		g.Expect(err).NotTo(HaveOccurred())

		c := meta.FindStatusCondition(latest.Status.Conditions, "Progressing")
		g.Expect(c).NotTo(BeNil())
		g.Expect(c.Status).To(Equal(metav1.ConditionTrue))
		g.Expect(c.Message).To(ContainSubstring("pre-authorization failed"))

		c = meta.FindStatusCondition(latest.Status.Conditions, "Installed")
		g.Expect(c).NotTo(BeNil())
		g.Expect(c.Status).To(Equal(metav1.ConditionFalse))
	}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}

// createDeficientClusterRole returns a modified ClusterRole according to the test scenario.
func createDeficientClusterRole(scenario int, name, ceName string) *rbacv1.ClusterRole {
	baseRules := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"olm.operatorframework.io"},
			Resources:     []string{"clusterextensions/finalizers"},
			Verbs:         []string{"update"},
			ResourceNames: []string{ceName},
		},
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
	}

	// Copy rules to avoid mutation
	rules := make([]rbacv1.PolicyRule, len(baseRules))
	copy(rules, baseRules)

	switch scenario {
	case 0:
		// Remove 'services' and 'services/finalizers'
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
	case 1:
		// Remove 'create' verb
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
	case 2:
		// Remove 'clusterrolebindings'
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
	case 3:
		// Restrict configmaps to named subset (resourceNames)
		for i, r := range rules {
			if r.APIGroups[0] == "" {
				filtered := []string{}
				for _, res := range r.Resources {
					if res != "configmaps" && res != "configmaps/finalizers" {
						filtered = append(filtered, res)
					}
				}
				rules[i].Resources = filtered
				rules = append(rules, rbacv1.PolicyRule{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					Verbs:         r.Verbs,
					ResourceNames: []string{"config-logging", "tekton-config-defaults", "tekton-config-observability"},
				})
			}
		}
	case 4:
		// Remove olm.operatorframework.io permission for finalizers
		filtered := []rbacv1.PolicyRule{}
		for _, r := range rules {
			if len(r.APIGroups) != 1 || r.APIGroups[0] != "olm.operatorframework.io" {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	case 5:
		// Remove 'bind' and 'escalate' verbs
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
