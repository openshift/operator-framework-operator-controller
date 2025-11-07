package specs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util/architecture"
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

	g.It("PolarionID:83069-[OTP]olmv1 static networkpolicy.", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:83069-olmv1 static networkpolicy."), g.Label("ReleaseGate"), func() {
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

	g.It("PolarionID:68936-[OTP][Skipped:Disconnected]cluster extension can not be installed with insufficient permission sa for operand", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:68936-[Skipped:Disconnected]cluster extension can not be installed with insufficient permission sa for operand"), func() {
		e2e.Logf("Testing ClusterExtension installation failure when ServiceAccount lacks sufficient permissions for operand resources. Originally case 75492, using 68936 for faster execution.")
		exutil.SkipForSNOCluster(oc)
		var (
			ns                                  = "ns-68936"
			sa                                  = "68936"
			baseDir                             = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate              = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate            = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingOperandTemplate = filepath.Join(baseDir, "sa-nginx-insufficient-operand-clusterrole.yaml")
			saCrb                               = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				RBACObjects: []olmv1util.ChildResource{
					{Kind: "RoleBinding", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role-binding", sa)}},
					{Kind: "Role", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role", sa)}},
					{Kind: "ClusterRoleBinding", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole-binding", sa),
						fmt.Sprintf("%s-installer-clusterrole-binding", sa)}},
					{Kind: "ClusterRole", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole", sa),
						fmt.Sprintf("%s-installer-clusterrole", sa)}},
					{Kind: "ServiceAccount", Ns: ns, Names: []string{sa}},
				},
				Kinds:    "okv68936s",
				Template: saClusterRoleBindingOperandTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-68936",
				Imageref: "quay.io/olmqe/nginx-ok-index:vokv68936",
				Template: clustercatalogTemplate,
			}
			ceInsufficient = olmv1util.ClusterExtensionDescription{
				Name:             "insufficient-68936",
				PackageName:      "nginx-ok-v68936",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("check Insufficient sa from operand")
		defer ceInsufficient.Delete(oc)
		_ = ceInsufficient.CreateWithoutCheck(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") {
			ceInsufficient.CheckClusterExtensionCondition(oc, "Progressing", "message", "pre-authorization failed", 10, 60, 0)
		} else {
			ceInsufficient.CheckClusterExtensionCondition(oc, "Progressing", "message", "cannot set blockOwnerDeletion", 10, 60, 0)
		}

	})

	g.It("PolarionID:68937-[OTP][Skipped:Disconnected]cluster extension can not be installed with insufficient permission sa for operand rbac object", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:68937-[Skipped:Disconnected]cluster extension can not be installed with insufficient permission sa for operand rbac object"), func() {
		e2e.Logf("Testing ClusterExtension installation failure when ServiceAccount lacks sufficient permissions for operand RBAC objects. Originally case 75492, using 68937 for faster execution.")
		exutil.SkipForSNOCluster(oc)
		var (
			ns                                  = "ns-68937"
			sa                                  = "68937"
			baseDir                             = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate              = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate            = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingOperandTemplate = filepath.Join(baseDir, "sa-nginx-insufficient-operand-rbac.yaml")
			saCrb                               = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				RBACObjects: []olmv1util.ChildResource{
					{Kind: "RoleBinding", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role-binding", sa)}},
					{Kind: "Role", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role", sa)}},
					{Kind: "ClusterRoleBinding", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole-binding", sa),
						fmt.Sprintf("%s-installer-clusterrole-binding", sa)}},
					{Kind: "ClusterRole", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole", sa),
						fmt.Sprintf("%s-installer-clusterrole", sa)}},
					{Kind: "ServiceAccount", Ns: ns, Names: []string{sa}},
				},
				Kinds:    "okv68937s",
				Template: saClusterRoleBindingOperandTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-68937",
				Imageref: "quay.io/olmqe/nginx-ok-index:vokv68937",
				Template: clustercatalogTemplate,
			}
			ceInsufficient = olmv1util.ClusterExtensionDescription{
				Name:             "insufficient-68937",
				PackageName:      "nginx-ok-v68937",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("check Insufficient sa from operand rbac")
		defer ceInsufficient.Delete(oc)
		_ = ceInsufficient.CreateWithoutCheck(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") {
			ceInsufficient.CheckClusterExtensionCondition(oc, "Progressing", "message", "pre-authorization failed", 10, 60, 0)
		} else {
			ceInsufficient.CheckClusterExtensionCondition(oc, "Progressing", "message", "permissions not currently held", 10, 60, 0)
		}

	})

	g.It("PolarionID:75492-[OTP][Level0][Skipped:Disconnected]cluster extension can not be installed with wrong sa or insufficient permission sa", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:75492-[Skipped:Disconnected]cluster extension can not be installed with wrong sa or insufficient permission sa"), func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75492"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			ceInsufficientName           = "ce-insufficient-" + caseID
			ceWrongSaName                = "ce-wrongsa-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-nginx-insufficient-bundle.yaml")
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				RBACObjects: []olmv1util.ChildResource{
					{Kind: "RoleBinding", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role-binding", sa)}},
					{Kind: "Role", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role", sa)}},
					{Kind: "ClusterRoleBinding", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole-binding", sa),
						fmt.Sprintf("%s-installer-clusterrole-binding", sa)}},
					{Kind: "ClusterRole", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole", sa),
						fmt.Sprintf("%s-installer-clusterrole", sa)}},
					{Kind: "ServiceAccount", Ns: ns, Names: []string{sa}},
				},
				Kinds:    "okv3277775492s",
				Template: saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv3283",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			ce75492Insufficient = olmv1util.ClusterExtensionDescription{
				Name:             ceInsufficientName,
				PackageName:      "nginx-ok-v3277775492",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
			ce75492WrongSa = olmv1util.ClusterExtensionDescription{
				Name:             ceWrongSaName,
				PackageName:      "nginx-ok-v3277775492",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa + "1",
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("check Insufficient sa from bundle")
		defer ce75492Insufficient.Delete(oc)
		_ = ce75492Insufficient.CreateWithoutCheck(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") {
			ce75492Insufficient.CheckClusterExtensionCondition(oc, "Progressing", "message", "pre-authorization failed", 10, 60, 0)
		} else {
			ce75492Insufficient.CheckClusterExtensionCondition(oc, "Progressing", "message", "could not get information about the resource CustomResourceDefinition", 10, 60, 0)
		}
		g.By("check wrong sa")
		defer ce75492WrongSa.Delete(oc)
		_ = ce75492WrongSa.CreateWithoutCheck(oc)
		ce75492WrongSa.CheckClusterExtensionCondition(oc, "Installed", "message", "not found", 10, 60, 0)
	})

	g.It("PolarionID:75493-[OTP][Level0][Skipped:Disconnected]cluster extension can be installed with enough permission sa", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:75493-[Skipped:Disconnected]cluster extension can be installed with enough permission sa"), func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "75493"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			ceSufficientName             = "ce-sufficient" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-nginx-limited.yaml")
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				RBACObjects: []olmv1util.ChildResource{
					{Kind: "RoleBinding", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role-binding", sa)}},
					{Kind: "Role", Ns: ns, Names: []string{fmt.Sprintf("%s-installer-role", sa)}},
					{Kind: "ClusterRoleBinding", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole-binding", sa),
						fmt.Sprintf("%s-installer-clusterrole-binding", sa)}},
					{Kind: "ClusterRole", Ns: "", Names: []string{fmt.Sprintf("%s-installer-rbac-clusterrole", sa),
						fmt.Sprintf("%s-installer-clusterrole", sa)}},
					{Kind: "ServiceAccount", Ns: ns, Names: []string{sa}},
				},
				Kinds:    "okv3277775493s",
				Template: saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv3283",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			ce75493 = olmv1util.ClusterExtensionDescription{
				Name:             ceSufficientName,
				PackageName:      "nginx-ok-v3277775493",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("check if ce is installed with limited permission")
		defer ce75493.Delete(oc)
		ce75493.Create(oc)
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "customresourcedefinitions.apiextensions.k8s.io", "okv3277775493s.cache.example.com")).To(o.BeTrue())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "services", "nginx-ok-v3283-75493-controller-manager-metrics-service", "-n", ns)).To(o.BeTrue())
		ce75493.Delete(oc)
		o.Expect(olmv1util.Appearance(oc, exutil.Disappear, "customresourcedefinitions.apiextensions.k8s.io", "okv3277775493s.cache.example.com")).To(o.BeTrue())
		o.Expect(olmv1util.Appearance(oc, exutil.Disappear, "services", "nginx-ok-v3283-75493-controller-manager-metrics-service", "-n", ns)).To(o.BeTrue())
	})

	g.It("PolarionID:81538-[OTP][Skipped:Disconnected]preflight check on permission on allns mode", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:81538-[Skipped:Disconnected]preflight check on permission on allns mode"), func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") {
			g.Skip("NewOLMPreflightPermissionChecks feature gate is disabled. This test requires preflight permission validation to be enabled.")
		}
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                   = "81538"
			ns                       = "ns-" + caseID
			sa                       = "sa" + caseID
			labelValue               = caseID
			catalogName              = "clustercatalog-" + caseID
			ceName                   = "ce-" + caseID
			clusterroleName          = ceName + "-clusterrole"
			roleName                 = ceName + "-role" + "-" + ns
			baseDir                  = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate   = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saTemplate               = filepath.Join(baseDir, "sa.yaml")
			bindingTemplate          = filepath.Join(baseDir, "binding-prefligth.yaml")
			clusterroleTemplate      = filepath.Join(baseDir, "prefligth-clusterrole.yaml")
			clustercatalog           = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv81538",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			ce = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v81538",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("create sa")
		paremeters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", saTemplate, "-p",
			"NAME=" + sa, "NAMESPACE=" + ns}
		configFileSa, errApplySa := olmv1util.ApplyNamepsaceResourceFromTemplate(oc, ns, paremeters...)
		o.Expect(errApplySa).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileSa).Execute() }()

		g.By("create clusterrole with wrong rule")
		paremeters = []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", clusterroleTemplate, "-p",
			"NAME=" + clusterroleName}
		configFileCLusterroe, errApplyCLusterrole := olmv1util.ApplyClusterResourceFromTemplate(oc, paremeters...)
		o.Expect(errApplyCLusterrole).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileCLusterroe).Execute() }()

		g.By("create binding")
		paremeters = []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", bindingTemplate, "-p",
			"SANAME=" + sa, "NAMESPACE=" + ns, "ROLENAME=" + roleName, "CLUSTERROLESANAME=" + clusterroleName}
		configFileBinding, errApplyBinding := olmv1util.ApplyClusterResourceFromTemplate(oc, paremeters...)
		o.Expect(errApplyBinding).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileBinding).Execute() }()

		g.By("check missing rule")
		defer ce.Delete(oc)
		_ = ce.CreateWithoutCheck(oc)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"" Verbs:[get] NonResourceURLs:[/metrics]`, 3, 150, 0)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"ns-81538" APIGroups:[] Resources:[services] ResourceNames:[nginx-ok-v81538-controller-manager-metrics-service] Verbs:[delete,get,patch,update]`, 3, 150, 0)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"" APIGroups:[olm.operatorframework.io] Resources:[clusterextensions/finalizers] ResourceNames:[ce-81538] Verbs:[update]`, 3, 150, 0)

		g.By("generate rbac per missing rule and delete ce")
		jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, "Progressing", "message")
		output, errGet := olmv1util.GetNoEmpty(oc, "clusterextension", ce.Name, "-o", jsonpath)
		o.Expect(errGet).NotTo(o.HaveOccurred())
		e2e.Logf("====%v====", output)

		start := "permissions to manage cluster extension:"
		end1 := "authorization evaluation error:"
		end2 := "for resolved bundle"
		filtered := olmv1util.FilterPermissions(output, start, end1, end2)
		e2e.Logf("===============================================================================")
		e2e.Logf("%v", filtered)
		e2e.Logf("===============================================================================")
		rabcDir := e2e.TestContext.OutputDir
		clusterroleFile := filepath.Join(rabcDir, fmt.Sprintf("%s.yaml", clusterroleName))
		roleFile := filepath.Join(rabcDir, fmt.Sprintf("%s.yaml", roleName))
		errGen := olmv1util.GenerateRBACFromMissingRules(filtered, ceName, rabcDir)
		o.Expect(errGen).NotTo(o.HaveOccurred())

		g.By("create clusterrole")
		err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", clusterroleFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("create role")
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", roleFile).Execute() }()
		err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", roleFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("check ce again afrer applying correct rules")
		ce.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 10, 600, 0)
	})

	g.It("PolarionID:81664-[OTP][Skipped:Disconnected]preflight check on permission on own ns mode", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:81664-[Skipped:Disconnected]preflight check on permission on own ns mode"), func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") ||
			!olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("Required feature gates are disabled: NewOLMPreflightPermissionChecks and NewOLMOwnSingleNamespace must both be enabled for this test.")
		}
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                   = "81664"
			ns                       = "ns-" + caseID
			sa                       = "sa" + caseID
			labelValue               = caseID
			catalogName              = "clustercatalog-" + caseID
			ceName                   = "ce-" + caseID
			clusterroleName          = ceName + "-clusterrole"
			roleName                 = ceName + "-role" + "-" + ns
			baseDir                  = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate   = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate = filepath.Join(baseDir, "clusterextension-withselectorlabel-OwnSingle.yaml")
			saTemplate               = filepath.Join(baseDir, "sa.yaml")
			bindingTemplate          = filepath.Join(baseDir, "binding-prefligth.yaml")
			clustercatalog           = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv81664",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			ce = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v81664",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				WatchNamespace:   ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("create sa")
		paremeters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", saTemplate, "-p",
			"NAME=" + sa, "NAMESPACE=" + ns}
		configFileSa, errApplySa := olmv1util.ApplyNamepsaceResourceFromTemplate(oc, ns, paremeters...)
		o.Expect(errApplySa).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileSa).Execute() }()

		g.By("check missing rule")
		defer ce.Delete(oc)
		_ = ce.CreateWithoutCheck(oc)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"" Verbs:[get] NonResourceURLs:[/metrics]`, 3, 150, 0)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"ns-81664" APIGroups:[] Resources:[services] ResourceNames:[nginx-ok-v81664-controller-manager-metrics-service] Verbs:[delete,get,patch,update]`, 3, 150, 0)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"" APIGroups:[olm.operatorframework.io] Resources:[clusterextensions/finalizers] ResourceNames:[ce-81664] Verbs:[update]`, 3, 150, 0)

		g.By("generate rbac per missing rule and delete ce")
		jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, "Progressing", "message")
		output, errGet := olmv1util.GetNoEmpty(oc, "clusterextension", ce.Name, "-o", jsonpath)
		o.Expect(errGet).NotTo(o.HaveOccurred())
		ce.Delete(oc)
		e2e.Logf("====%v====", output)

		start := "permissions to manage cluster extension:"
		end1 := "authorization evaluation error:"
		end2 := "for resolved bundle"
		filtered := olmv1util.FilterPermissions(output, start, end1, end2)
		e2e.Logf("===============================================================================")
		e2e.Logf("%v", filtered)
		e2e.Logf("===============================================================================")
		rabcDir := e2e.TestContext.OutputDir
		clusterroleFile := filepath.Join(rabcDir, fmt.Sprintf("%s.yaml", clusterroleName))
		roleFile := filepath.Join(rabcDir, fmt.Sprintf("%s.yaml", roleName))
		errGen := olmv1util.GenerateRBACFromMissingRules(filtered, ceName, rabcDir)
		o.Expect(errGen).NotTo(o.HaveOccurred())

		g.By("create clusterrole")
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", clusterroleFile).Execute() }()
		err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", clusterroleFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("create role")
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", roleFile).Execute() }()
		err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", roleFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("create binding")
		paremeters = []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", bindingTemplate, "-p",
			"SANAME=" + sa, "NAMESPACE=" + ns, "ROLENAME=" + roleName, "CLUSTERROLESANAME=" + clusterroleName}
		configFileBinding, errApplyBinding := olmv1util.ApplyClusterResourceFromTemplate(oc, paremeters...)
		o.Expect(errApplyBinding).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileBinding).Execute() }()

		g.By("check ce again afrer applying correct rules")
		ce.Create(oc)
	})

	g.It("PolarionID:81696-[OTP][Skipped:Disconnected]preflight check on permission on single ns mode", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:81696-[Skipped:Disconnected]preflight check on permission on single ns mode"), func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") ||
			!olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("Required feature gates are disabled: NewOLMPreflightPermissionChecks and NewOLMOwnSingleNamespace must both be enabled for this test.")
		}
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                   = "81696"
			ns                       = "ns-" + caseID
			nsWatch                  = "ns-" + caseID + "-watch"
			sa                       = "sa" + caseID
			labelValue               = caseID
			catalogName              = "clustercatalog-" + caseID
			ceName                   = "ce-" + caseID
			clusterroleName          = ceName + "-clusterrole"
			roleNsName               = ceName + "-role" + "-" + ns
			roleNsWatchName          = ceName + "-role" + "-" + nsWatch
			baseDir                  = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate   = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate = filepath.Join(baseDir, "clusterextension-withselectorlabel-OwnSingle.yaml")
			saTemplate               = filepath.Join(baseDir, "sa.yaml")
			bindingTemplate          = filepath.Join(baseDir, "binding-prefligth_multirole.yaml")
			clustercatalog           = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv81696",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			ce = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v81696",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				WatchNamespace:   nsWatch,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create watch namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsWatch, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsWatch).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsWatch)).To(o.BeTrue())

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("create sa")
		paremeters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", saTemplate, "-p",
			"NAME=" + sa, "NAMESPACE=" + ns}
		configFileSa, errApplySa := olmv1util.ApplyNamepsaceResourceFromTemplate(oc, ns, paremeters...)
		o.Expect(errApplySa).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileSa).Execute() }()

		g.By("check missing rule")
		defer ce.Delete(oc)
		_ = ce.CreateWithoutCheck(oc)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"" Verbs:[get] NonResourceURLs:[/metrics]`, 3, 150, 0)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"ns-81696" APIGroups:[] Resources:[services] ResourceNames:[nginx-ok-v81696-controller-manager-metrics-service] Verbs:[delete,get,patch,update]`, 3, 150, 0)
		ce.CheckClusterExtensionCondition(oc, "Progressing", "message",
			`Namespace:"" APIGroups:[olm.operatorframework.io] Resources:[clusterextensions/finalizers] ResourceNames:[ce-81696] Verbs:[update]`, 3, 150, 0)

		g.By("generate rbac per missing rule and delete ce")
		jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, "Progressing", "message")
		output, errGet := olmv1util.GetNoEmpty(oc, "clusterextension", ce.Name, "-o", jsonpath)
		o.Expect(errGet).NotTo(o.HaveOccurred())
		ce.Delete(oc)
		e2e.Logf("====%v====", output)

		start := "permissions to manage cluster extension:"
		end1 := "authorization evaluation error:"
		end2 := "for resolved bundle"
		filtered := olmv1util.FilterPermissions(output, start, end1, end2)
		e2e.Logf("===============================================================================")
		e2e.Logf("%v", filtered)
		e2e.Logf("===============================================================================")
		rbacDir := e2e.TestContext.OutputDir
		clusterroleFile := filepath.Join(rbacDir, fmt.Sprintf("%s.yaml", clusterroleName))
		roleNsFile := filepath.Join(rbacDir, fmt.Sprintf("%s.yaml", roleNsName))
		roleNsWatchFile := filepath.Join(rbacDir, fmt.Sprintf("%s.yaml", roleNsWatchName))

		errGen := olmv1util.GenerateRBACFromMissingRules(filtered, ceName, rbacDir)
		o.Expect(errGen).NotTo(o.HaveOccurred())

		g.By("create clusterrole")
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", clusterroleFile).Execute() }()
		err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", clusterroleFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("create role for ns")
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", roleNsFile).Execute() }()
		err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", roleNsFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("create role for ns watch")
		// Check if the watch namespace role file exists before trying to apply it
		// The file may not exist if no permissions are needed for the watch namespace
		if _, err := os.Stat(roleNsWatchFile); err == nil {
			defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", roleNsWatchFile).Execute() }()
			err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", roleNsWatchFile).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Watch namespace role file %s does not exist, skipping creation", roleNsWatchFile)
		}

		g.By("create binding")
		paremeters = []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", bindingTemplate, "-p",
			"SANAME=" + sa, "NAMESPACE=" + ns, "ROLENAME=" + roleNsName, "CLUSTERROLESANAME=" + clusterroleName,
			"WATCHNAMESPACE=" + nsWatch, "WATCHROLENAME=" + roleNsWatchName}
		configFileBinding, errApplyBinding := olmv1util.ApplyClusterResourceFromTemplate(oc, paremeters...)
		o.Expect(errApplyBinding).NotTo(o.HaveOccurred())
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFileBinding).Execute() }()

		g.By("check ce again afrer applying correct rules")
		ce.Create(oc)
	})

	g.It("PolarionID:74618-[OTP][Skipped:Disconnected]ClusterExtension supports simple registry vzero bundles only", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:74618-[Skipped:Disconnected]ClusterExtension supports simple registry vzero bundles only"), func() {
		exutil.SkipForSNOCluster(oc)
		var (
			ns                           = "ns-74618"
			sa                           = "sa74618"
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     "clustercatalog-74618",
				Imageref: "quay.io/olmqe/nginx-ok-index:vokv32777",
				Template: clustercatalogTemplate,
			}
			ceGVK = olmv1util.ClusterExtensionDescription{
				Name:             "dep-gvk-32777",
				PackageName:      "nginx-ok-v32777gvk",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
			cePKG = olmv1util.ClusterExtensionDescription{
				Name:                    "dep-pkg-32777",
				PackageName:             "nginx-ok-v32777pkg",
				Channel:                 "alpha",
				Version:                 ">=0.0.1",
				InstallNamespace:        ns,
				UpgradeConstraintPolicy: "SelfCertified",
				SaName:                  sa,
				Template:                clusterextensionTemplate,
			}
			ceCST = olmv1util.ClusterExtensionDescription{
				Name:             "dep-cst-32777",
				PackageName:      "nginx-ok-v32777cst",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
			ceWBH = olmv1util.ClusterExtensionDescription{
				Name:                    "wbh-32777",
				PackageName:             "nginx-ok-v32777wbh",
				Channel:                 "alpha",
				Version:                 ">=0.0.1",
				InstallNamespace:        ns,
				UpgradeConstraintPolicy: "SelfCertified",
				SaName:                  sa,
				Template:                clusterextensionTemplate,
			}
			ceNAN = olmv1util.ClusterExtensionDescription{
				Name:             "nan-32777",
				PackageName:      "nginx-ok-v32777nan",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("check gvk dependency fails to be installed")
		defer ceGVK.Delete(oc)
		_ = ceGVK.CreateWithoutCheck(oc)
		ceGVK.CheckClusterExtensionCondition(oc, "Progressing", "message", "has a dependency declared via property \"olm.gvk.required\" which is currently not supported", 10, 180, 0)
		ceGVK.Delete(oc)

		g.By("check pkg dependency fails to be installed")
		defer cePKG.Delete(oc)
		_ = cePKG.CreateWithoutCheck(oc)
		cePKG.CheckClusterExtensionCondition(oc, "Progressing", "message", "has a dependency declared via property \"olm.package.required\" which is currently not supported", 10, 180, 0)
		cePKG.Delete(oc)

		g.By("check cst dependency fails to be installed")
		defer ceCST.Delete(oc)
		_ = ceCST.CreateWithoutCheck(oc)
		ceCST.CheckClusterExtensionCondition(oc, "Progressing", "message", "has a dependency declared via property \"olm.constraint\" which is currently not supported", 10, 180, 0)
		ceCST.Delete(oc)

		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMWebhookProviderOpenshiftServiceCA") {
			g.By("check webhook fails to be installed")
			defer ceWBH.Delete(oc)
			_ = ceWBH.CreateWithoutCheck(oc)
			ceWBH.CheckClusterExtensionCondition(oc, "Progressing", "message", "webhookDefinitions are not supported", 10, 180, 0)
			ceWBH.CheckClusterExtensionCondition(oc, "Installed", "reason", "Failed", 10, 180, 0)
			ceWBH.Delete(oc)
		}

		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.By("check non all ns mode fails to be installed.")
			defer ceNAN.Delete(oc)
			_ = ceNAN.CreateWithoutCheck(oc)
			ceNAN.CheckClusterExtensionCondition(oc, "Progressing", "message", "bundle does not support AllNamespaces install mode", 10, 180, 0)
			ceNAN.CheckClusterExtensionCondition(oc, "Installed", "reason", "Failed", 10, 180, 0)
			ceNAN.Delete(oc)
		}

	})

	g.It("PolarionID:76843-[OTP][Skipped:Disconnected]support disc with icsp[Timeout:30m] [Disruptive][Slow]", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:76843-[Skipped:Disconnected]support disc with icsp[Timeout:30m] [Serial][Disruptive][Slow]"), func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "76843"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			ceName                       = "ce-" + caseID
			iscpName                     = "icsp-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			icspTemplate                 = filepath.Join(baseDir, "icsp-single-mirror.yaml")
			icsp                         = olmv1util.IcspDescription{
				Name:     iscpName,
				Mirror:   "quay.io/olmqe",
				Source:   "qe76843.myregistry.io/olmqe",
				Template: icspTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     catalogName,
				Imageref: "qe76843.myregistry.io/olmqe/nginx-ok-index@sha256:c613ddd68b74575d823c6f370c0941b051ea500aa4449224489f7f2cc716e712",
				Template: clustercatalogTemplate,
			}
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			ce76843 = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v76843",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("check if there is idms or itms")
		if exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
			exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "ImageDigestMirrorSet") ||
			exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
				exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "ImageTagMirrorSet") {
			g.Skip("ImageTagMirrorSet or ImageDigestMirrorSet already exists in cluster. This test requires ICSP-only environment.")
		}

		g.By("check if current mcp is healthy")
		if !olmv1util.HealthyMCP4OLM(oc) {
			g.Skip("MachineConfigPool is not in healthy state. Cannot proceed with disruptive test that modifies machine configuration.")
		}

		g.By("create icsp")
		defer icsp.Delete(oc)
		icsp.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("check ce to be installed")
		defer ce76843.Delete(oc)
		ce76843.Create(oc)

	})

	g.It("PolarionID:76844-[OTP][Skipped:Disconnected]support disc with itms and idms[Timeout:30m] [Disruptive][Slow]", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:76844-[Skipped:Disconnected]support disc with itms and idms[Timeout:30m] [Serial][Disruptive][Slow]"), func() {
		exutil.SkipOnProxyCluster(oc)
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "76844"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			ceName                       = "ce-" + caseID
			itdmsName                    = "itdms-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			itdmsTemplate                = filepath.Join(baseDir, "itdms-full-mirror.yaml")
			itdms                        = olmv1util.ItdmsDescription{
				Name:            itdmsName,
				MirrorSite:      "quay.io",
				SourceSite:      "qe76844.myregistry.io",
				MirrorNamespace: "quay.io/olmqe",
				SourceNamespace: "qe76844.myregistry.io/olmqe",
				Template:        itdmsTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     catalogName,
				Imageref: "qe76844.myregistry.io/olmqe/nginx-ok-index:vokv76844",
				Template: clustercatalogTemplate,
			}
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			ce76844 = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v76844",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("check if there is icsp")
		if exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
			exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "ImageContentSourcePolicy") {
			g.Skip("ImageContentSourcePolicy already exists in cluster. This test requires ITMS/IDMS-only environment.")
		}

		g.By("check if current mcp is healthy")
		if !olmv1util.HealthyMCP4OLM(oc) {
			g.Skip("MachineConfigPool is not in healthy state. Cannot proceed with disruptive test that modifies machine configuration.")
		}

		g.By("create itdms")
		defer itdms.Delete(oc)
		itdms.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("check ce to be installed")
		defer ce76844.Delete(oc)
		ce76844.Create(oc)

	})

	g.It("PolarionID:78193-[OTP][Skipped:Disconnected]Runtime validation of container images using sigstore signatures [Disruptive][Slow]", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:78193-[Skipped:Disconnected]Runtime validation of container images using sigstore signatures [Serial][Disruptive][Slow]"), func() {
		if !exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
			exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "crd", "clusterimagepolicies.config.openshift.io") {
			g.Skip("ClusterImagePolicy CRD not found. This test requires sigstore signature validation capabilities.")
		}
		architecture.SkipNonAmd64SingleArch(oc)
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "78193"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			catalog1Name                 = "clustercatalog-" + caseID + "1"
			ceName                       = "ce-" + caseID
			cipName                      = "cip-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			cipTemplate                  = filepath.Join(baseDir, "cip.yaml")
			cip                          = olmv1util.CipDescription{
				Name:     cipName,
				Repo1:    "quay.io/olmqe/nginx-ok-bundle-sigstore",
				Repo2:    "quay.io/olmqe/nginx-ok-bundle-sigstore1",
				Repo3:    "quay.io/olmqe/nginx-ok-index-sigstore",
				Repo4:    "quay.io/olmqe/nginx-ok-index-sigstore1",
				Template: cipTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     catalogName,
				Imageref: "quay.io/olmqe/nginx-ok-index-sigstore:vokv78193",
				Template: clustercatalogTemplate,
			}
			clustercatalog1 = olmv1util.ClusterCatalogDescription{
				Name:     catalog1Name,
				Imageref: "quay.io/olmqe/nginx-ok-index-sigstore1:vokv781931",
				Template: clustercatalogTemplate,
			}
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			ce = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v78193",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("check if current mcp is healthy")
		if !olmv1util.HealthyMCP4OLM(oc) {
			g.Skip("MachineConfigPool is not in healthy state. Cannot proceed with disruptive test that modifies machine configuration.")
		}

		g.By("create cip")
		defer cip.Delete(oc)
		cip.Create(oc)

		g.By("Create clustercatalog with olmsigkey signed successfully")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension with olmsigkey signed successfully")
		defer ce.Delete(oc)
		ce.Create(oc)

		g.By("Create clustercatalog with olmsigkey1 signed failed")
		defer clustercatalog1.Delete(oc)
		_ = clustercatalog1.CreateWithoutCheck(oc)
		clustercatalog1.CheckClusterCatalogCondition(oc, "Progressing", "message", "signature verification failed: invalid signature", 10, 90, 0)

	})

	g.It("PolarionID:78300-[OTP][Skipped:Disconnected]validation of container images using sigstore signatures with different policy [Disruptive][Slow]", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:78300-[Skipped:Disconnected]validation of container images using sigstore signatures with different policy [Serial][Disruptive][Slow]"), func() {
		if !exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
			exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "crd", "clusterimagepolicies.config.openshift.io") {
			g.Skip("ClusterImagePolicy CRD not found. This test requires sigstore signature validation capabilities.")
		}
		exutil.SkipForSNOCluster(oc)
		var (
			caseID                       = "781932"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			imageRef                     = "quay.io/olmqe/nginx-ok-index-sigstore:vokv" + caseID
			packageName                  = "nginx-ok-v" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			ceName                       = "ce-" + caseID
			cipName                      = "cip-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			cipTemplate                  = filepath.Join(baseDir, "cip.yaml")
			cip                          = olmv1util.CipDescription{
				Name:     cipName,
				Repo1:    "quay.io/olmqe/nginx-ok-bundle-sigstore",
				Repo2:    "quay.io/olmqe/nginx-ok-bundle-sigstore1",
				Repo3:    "quay.io/olmqe/nginx-ok-index-sigstore",
				Repo4:    "quay.io/olmqe/nginx-ok-index-sigstore1",
				Policy:   "MatchRepository",
				Template: cipTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:     catalogName,
				Imageref: imageRef,
				Template: clustercatalogTemplate,
			}
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			ce = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      packageName,
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)
		g.By("check if current mcp is healthy")
		if !olmv1util.HealthyMCP4OLM(oc) {
			g.Skip("MachineConfigPool is not in healthy state. Cannot proceed with disruptive test that modifies machine configuration.")
		}

		g.By("create cip")
		defer cip.Delete(oc)
		cip.Create(oc)

		g.By("Create clustercatalog with olmsigkey signed successfully")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clusterextension with olmsigkey signed successfully")
		defer ce.Delete(oc)
		ce.Create(oc)

	})

	g.It("PolarionID:76983-[OTP][Skipped:Disconnected]install index and bundle from private image[Slow]", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:76983-[Skipped:Disconnected]install index and bundle from private image[Slow]"), func() {
		exutil.SkipForSNOCluster(oc)
		// This test validates installation from private container images and depends on cluster-wide pull secrets
		var (
			caseID                       = "76983"
			ns                           = "ns-" + caseID
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			ceName                       = "ce-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			clustercatalog               = olmv1util.ClusterCatalogDescription{
				Name:     catalogName,
				Imageref: "quay.io/olmqe/nginx-ok-index-private:vokv76983",
				Template: clustercatalogTemplate,
			}
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			ce = olmv1util.ClusterExtensionDescription{
				Name:             ceName,
				PackageName:      "nginx-ok-v76983",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("check if there is global secret and it includes token to access quay.io")
		if !exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
			exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "secret/pull-secret", "-n", "openshift-config") {
			g.Skip("Global pull secret not found in openshift-config namespace. This test requires cluster-wide image pull authentication.")
		}

		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("secret/pull-secret", "-n", "openshift-config",
			`--template={{index .data ".dockerconfigjson" | base64decode}}`).Output()
		if err != nil {
			e2e.Failf("Failed to extract dockerconfigjson data from global pull secret: %v", err)
		}
		if !strings.Contains(output, "quay.io/olmqe") {
			g.Skip("Global pull secret does not contain credentials for quay.io/olmqe. This test requires access to private test images.")
		}

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Create namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("check ce to be installed")
		defer ce.Delete(oc)
		ce.Create(oc)

	})

	g.It("PolarionID:76985-[OTP][Skipped:Disconnected]authfile is updated automatically[Timeout:30m] [Disruptive][Slow]", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:76985-[Skipped:Disconnected]authfile is updated automatically[Timeout:30m] [Serial][Disruptive][Slow]"), func() {
		exutil.SkipForSNOCluster(oc)
		var (
			caseID = "76985"
		)

		g.By("check if there is global secret")
		if !exutil.CheckAppearance(oc, 1*time.Second, 1*time.Second, exutil.Immediately,
			exutil.AsAdmin, exutil.WithoutNamespace, exutil.Appear, "secret/pull-secret", "-n", "openshift-config") {
			g.Skip("Global pull secret not found in openshift-config namespace. This test requires cluster-wide image pull authentication.")
		}

		g.By("check if current mcp is healthy")
		if !olmv1util.HealthyMCP4OLM(oc) {
			g.Skip("MachineConfigPool is not in healthy state. Cannot proceed with disruptive test that modifies machine configuration.")
		}

		g.By("set gobal secret")
		dirname := "/tmp/" + caseID + "-globalsecretdir"
		err := os.MkdirAll(dirname, 0777)
		o.Expect(err).NotTo(o.HaveOccurred())
		defer os.RemoveAll(dirname)

		err = oc.AsAdmin().WithoutNamespace().Run("extract").Args("secret/pull-secret", "-n", "openshift-config", "--to="+dirname, "--confirm").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		newAuthCmd := fmt.Sprintf(`cat %s/.dockerconfigjson | jq '.auths["fake.registry"] |= . + {"auth":"faketoken=="}' > %s/newdockerconfigjson`, dirname, dirname)
		_, err = exec.Command("bash", "-c", newAuthCmd).Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		err = oc.AsAdmin().WithoutNamespace().Run("set").Args("data", "secret/pull-secret", "-n", "openshift-config",
			"--from-file=.dockerconfigjson="+dirname+"/newdockerconfigjson").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		defer func() {
			err = oc.AsAdmin().WithoutNamespace().Run("set").Args("data", "secret/pull-secret", "-n", "openshift-config",
				"--from-file=.dockerconfigjson="+dirname+"/.dockerconfigjson").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			olmv1util.AssertMCPCondition(oc, "master", "Updating", "status", "True", 3, 120, 1)
			olmv1util.AssertMCPCondition(oc, "worker", "Updating", "status", "True", 3, 120, 1)
			olmv1util.AssertMCPCondition(oc, "master", "Updating", "status", "False", 30, 600, 20)
			olmv1util.AssertMCPCondition(oc, "worker", "Updating", "status", "False", 30, 600, 20)
			o.Expect(olmv1util.HealthyMCP4OLM(oc)).To(o.BeTrue())
		}()

		olmv1util.AssertMCPCondition(oc, "master", "Updating", "status", "True", 3, 120, 1)
		olmv1util.AssertMCPCondition(oc, "worker", "Updating", "status", "True", 3, 120, 1)
		olmv1util.AssertMCPCondition(oc, "master", "Updating", "status", "False", 30, 600, 20)
		olmv1util.AssertMCPCondition(oc, "worker", "Updating", "status", "False", 30, 600, 20)
		o.Expect(olmv1util.HealthyMCP4OLM(oc)).To(o.BeTrue())

		g.By("check if auth is updated for catalogd")
		catalogDPod, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "control-plane=catalogd-controller-manager",
			"-o=jsonpath={.items[0].metadata.name}", "-n", "openshift-catalogd").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(catalogDPod).NotTo(o.BeEmpty())

		// Use exec command directly to avoid logging sensitive authentication data
		checkAuthCmdCatalogd := `grep -q "fake.registry" /tmp/catalogd-global-pull-secret-*.json`
		finalArgsCatalogd := []string{
			"--kubeconfig=" + exutil.KubeConfigPath(),
			"exec",
			"-n",
			"openshift-catalogd",
			catalogDPod,
			"--",
			"bash",
			"-c",
			checkAuthCmdCatalogd,
		}

		e2e.Logf("cmdCatalogd: %v", "oc"+" "+strings.Join(finalArgsCatalogd, " "))
		cmdCatalogd := exec.Command("oc", finalArgsCatalogd...)
		_, err = cmdCatalogd.CombinedOutput()
		// Output not logged to prevent leaking authentication credentials
		o.Expect(err).NotTo(o.HaveOccurred(), "auth for catalogd is not updated")

		g.By("check if auth is updated for operator-controller")
		operatorControlerPod, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "control-plane=operator-controller-controller-manager",
			"-o=jsonpath={.items[0].metadata.name}", "-n", "openshift-operator-controller").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(operatorControlerPod).NotTo(o.BeEmpty())

		// Use exec command directly to avoid logging sensitive authentication data
		checkAuthCmdOperatorController := `grep -q "registry" /tmp/operator-controller-global-pull-secrets-*.json`
		finalArgsOperatorController := []string{
			"--kubeconfig=" + exutil.KubeConfigPath(),
			"exec",
			"-n",
			"openshift-operator-controller",
			operatorControlerPod,
			"--",
			"bash",
			"-c",
			checkAuthCmdOperatorController,
		}

		e2e.Logf("cmdOperatorController: %v", "oc"+" "+strings.Join(finalArgsOperatorController, " "))
		cmdOperatorController := exec.Command("oc", finalArgsOperatorController...)
		_, err = cmdOperatorController.CombinedOutput()
		// Output not logged to prevent leaking authentication credentials
		o.Expect(err).NotTo(o.HaveOccurred(), "auth for operator-controller is not updated")

	})

})
