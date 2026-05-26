package test

import (
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

// olmv1Component groups a deployment and its namespace for topology checks.
type olmv1Component struct {
	namespace      string
	deploymentName string
}

// olmv1Components lists the OLMv1 operand deployments that cluster-olm-operator manages.
var olmv1Components = []olmv1Component{
	{"openshift-operator-controller", "operator-controller-controller-manager"},
	{"openshift-catalogd", "catalogd-controller-manager"},
}

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 topology-based deployment scaling", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})

	// This test verifies the cluster-olm-operator behaviour described in
	// pkg/controller/helm.go: HighlyAvailable / HighlyAvailableArbiter /
	// DualReplica topologies get replicas=2 and a PodDisruptionBudget, while
	// SingleReplica (SNO) and External topologies keep the chart default of
	// replicas=1 with no PDB.
	It("should configure replicas and PodDisruptionBudgets to match the cluster control plane topology", func(ctx SpecContext) {
		Skip("TODO OCPBUGS-94187: OLMv1 test extension does not yet support the arguments required to run this test")
		k8sClient := env.Get().K8sClient

		infra := &configv1.Infrastructure{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "cluster"}, infra)).To(Succeed())

		topology := infra.Status.ControlPlaneTopology
		isHA := topology == configv1.HighlyAvailableTopologyMode ||
			topology == configv1.HighlyAvailableArbiterMode ||
			topology == configv1.DualReplicaTopologyMode

		By(fmt.Sprintf("detected control plane topology: %q (HA=%v)", topology, isHA))

		for _, c := range olmv1Components {
			By(fmt.Sprintf("checking deployment %s/%s", c.namespace, c.deploymentName))

			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Namespace: c.namespace,
				Name:      c.deploymentName,
			}, dep)).To(Succeed(), "deployment %s/%s should exist", c.namespace, c.deploymentName)

			if isHA {
				Expect(dep.Spec.Replicas).NotTo(BeNil(),
					"topology %q: deployment %s should have replicas explicitly set", topology, c.deploymentName)
				Expect(*dep.Spec.Replicas).To(BeNumerically("==", 2),
					"topology %q: deployment %s should have 2 replicas", topology, c.deploymentName)
			} else {
				// Kubernetes treats nil replicas as 1; both nil and the explicit value 1 are acceptable.
				if dep.Spec.Replicas != nil {
					Expect(*dep.Spec.Replicas).To(BeNumerically("==", 1),
						"topology %q: deployment %s should have 1 replica", topology, c.deploymentName)
				}
			}

			By(fmt.Sprintf("checking PodDisruptionBudgets for deployment %s/%s", c.namespace, c.deploymentName))

			pdbList := &policyv1.PodDisruptionBudgetList{}
			Expect(k8sClient.List(ctx, pdbList, client.InNamespace(c.namespace))).To(Succeed())

			// Filter to PDBs whose selector actually targets this deployment's pods.
			podLabels := labels.Set(dep.Spec.Template.Labels)
			var matchingPDBs []policyv1.PodDisruptionBudget
			for _, pdb := range pdbList.Items {
				if pdb.Spec.Selector == nil {
					continue
				}
				sel, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
				Expect(err).NotTo(HaveOccurred(), "PDB %s has an invalid selector", pdb.Name)
				if sel.Matches(podLabels) {
					matchingPDBs = append(matchingPDBs, pdb)
				}
			}

			if isHA {
				Expect(matchingPDBs).NotTo(BeEmpty(),
					"topology %q: deployment %s should have a PodDisruptionBudget", topology, c.deploymentName)
			} else {
				Expect(matchingPDBs).To(BeEmpty(),
					"topology %q: deployment %s should have no PodDisruptionBudgets", topology, c.deploymentName)
			}
		}
	})
})
