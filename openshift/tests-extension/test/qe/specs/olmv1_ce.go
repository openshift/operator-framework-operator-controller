package specs

import (
	"fmt"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
	olmv1util "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util/olmv1util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] clusterextension", g.Label("NonHyperShiftHOST"), func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("default")
	)

	g.BeforeEach(func() {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
	})

	g.It("PolarionID:83069-olmv1 static networkpolicy.", func() {
		policies := []olmv1util.NpExpecter{
			{
				Name:      "catalogd-controller-manager",
				Namespace: "openshift-catalogd",
				ExpectIngress: []olmv1util.IngressRule{
					{
						Ports: []olmv1util.Port{
							{Port: 7443, Protocol: "TCP"},
							{Port: 8443, Protocol: "TCP"},
							{Port: 9443, Protocol: "TCP"},
						},
						Selectors: nil,
					},
				},
				ExpectEgress: []olmv1util.EgressRule{
					{
						Ports:     []olmv1util.Port{{}}, // empty rule
						Selectors: nil,
					},
				},
				ExpectSelector:    map[string]string{"control-plane": "catalogd-controller-manager"},
				ExpectPolicyTypes: []string{"Ingress", "Egress"},
			},
			{
				Name:              "catalogd-default-deny-all-traffic",
				Namespace:         "openshift-catalogd",
				ExpectIngress:     nil,
				ExpectEgress:      nil,
				ExpectSelector:    map[string]string{},
				ExpectPolicyTypes: []string{"Ingress", "Egress"},
			},
			{
				Name:          "allow-egress-to-api-server",
				Namespace:     "openshift-cluster-olm-operator",
				ExpectIngress: nil,
				ExpectEgress: []olmv1util.EgressRule{
					{
						Ports:     []olmv1util.Port{{Port: 6443, Protocol: "TCP"}},
						Selectors: nil,
					},
				},
				ExpectSelector:    map[string]string{"name": "cluster-olm-operator"},
				ExpectPolicyTypes: []string{"Egress"},
			},
			{
				Name:          "allow-egress-to-openshift-dns",
				Namespace:     "openshift-cluster-olm-operator",
				ExpectIngress: nil,
				ExpectEgress: []olmv1util.EgressRule{
					{
						Ports: []olmv1util.Port{
							{Port: "dns-tcp", Protocol: "TCP"},
							{Port: "dns", Protocol: "UDP"},
						},
						Selectors: []olmv1util.Selector{
							{NamespaceLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-dns"}},
						},
					},
				},
				ExpectSelector:    map[string]string{"name": "cluster-olm-operator"},
				ExpectPolicyTypes: []string{"Egress"},
			},
			{
				Name:      "allow-metrics-traffic",
				Namespace: "openshift-cluster-olm-operator",
				ExpectIngress: []olmv1util.IngressRule{
					{
						Ports: []olmv1util.Port{{Port: 8443, Protocol: "TCP"}},
						Selectors: []olmv1util.Selector{
							{NamespaceLabels: map[string]string{"name": "openshift-monitoring"}},
						},
					},
				},
				ExpectEgress:      nil,
				ExpectSelector:    map[string]string{"name": "cluster-olm-operator"},
				ExpectPolicyTypes: []string{"Ingress"},
			},
			{
				Name:              "default-deny-all",
				Namespace:         "openshift-cluster-olm-operator",
				ExpectIngress:     nil,
				ExpectEgress:      nil,
				ExpectSelector:    map[string]string{},
				ExpectPolicyTypes: []string{"Ingress", "Egress"},
			},
			{
				Name:      "operator-controller-controller-manager",
				Namespace: "openshift-operator-controller",
				ExpectIngress: []olmv1util.IngressRule{
					{
						Ports:     []olmv1util.Port{{Port: 8443, Protocol: "TCP"}},
						Selectors: nil,
					},
				},
				ExpectEgress: []olmv1util.EgressRule{
					{
						Ports:     []olmv1util.Port{{}}, // empty rule
						Selectors: nil,
					},
				},
				ExpectSelector:    map[string]string{"control-plane": "operator-controller-controller-manager"},
				ExpectPolicyTypes: []string{"Ingress", "Egress"},
			},
			{
				Name:              "operator-controller-default-deny-all-traffic",
				Namespace:         "openshift-operator-controller",
				ExpectIngress:     nil,
				ExpectEgress:      nil,
				ExpectSelector:    map[string]string{},
				ExpectPolicyTypes: []string{"Ingress", "Egress"},
			},
		}

		for _, policy := range policies {

			g.By(fmt.Sprintf("Checking NP %s in %s", policy.Name, policy.Namespace))
			specs, err := oc.AsAdmin().WithoutNamespace().
				Run("get").Args("networkpolicy", policy.Name, "-n", policy.Namespace, "-o=jsonpath={.spec}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(specs).NotTo(o.BeEmpty())
			e2e.Logf("specs: %v", specs)

			olmv1util.VerifySelector(specs, policy.ExpectSelector, policy.Name)
			olmv1util.VerifyPolicyTypes(specs, policy.ExpectPolicyTypes, policy.Name)
			olmv1util.VerifyIngress(specs, policy.ExpectIngress, policy.Name)
			olmv1util.VerifyEgress(specs, policy.ExpectEgress, policy.Name)
		}

	})

})
