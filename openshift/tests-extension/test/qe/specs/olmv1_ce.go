package specs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
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

	g.It("PolarionID:68936-[OTP]cluster extension can not be installed with insufficient permission sa for operand", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:68936-[Skipped:Disconnected]cluster extension can not be installed with insufficient permission sa for operand"), func() {
		e2e.Logf("Testing ClusterExtension installation failure when ServiceAccount lacks sufficient permissions for operand resources. Originally case 75492, using 68936 for faster execution.")
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:68937-[OTP]cluster extension can not be installed with insufficient permission sa for operand rbac object", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:68937-[Skipped:Disconnected]cluster extension can not be installed with insufficient permission sa for operand rbac object"), func() {
		e2e.Logf("Testing ClusterExtension installation failure when ServiceAccount lacks sufficient permissions for operand RBAC objects. Originally case 75492, using 68937 for faster execution.")
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:70723-[OTP][Skipped:Disconnected]olmv1 downgrade version", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			ns                           = "ns-70723"
			sa                           = "sa70723"
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
				Name:     "clustercatalog-70723",
				Imageref: "quay.io/openshifttest/nginxolm-operator-index:nginxolm70723",
				Template: clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-70723",
				InstallNamespace: ns,
				PackageName:      "nginx70723",
				Channel:          "candidate-v2",
				Version:          "2.2.1",
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

		g.By("Install version 2.2.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("2.2.1"))

		g.By("Attempt to downgrade to version 2.0.0 with CatalogProvided policy and expect failure")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version": "2.0.0"}}}}`)
		clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "message", "error upgrading", 3, 150, 0)

		g.By("Change UpgradeConstraintPolicy to SelfCertified and allow downgrade")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"upgradeConstraintPolicy": "SelfCertified"}}}}`)
		clusterextension.WaitClusterExtensionVersion(oc, "2.0.0")
	})

	g.It("PolarionID:75492-[OTP][Level0]cluster extension can not be installed with wrong sa or insufficient permission sa", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:75492-[Skipped:Disconnected]cluster extension can not be installed with wrong sa or insufficient permission sa"), func() {
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:75493-[OTP][Level0]cluster extension can be installed with enough permission sa", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:75493-[Skipped:Disconnected]cluster extension can be installed with enough permission sa"), func() {
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:81538-[OTP]preflight check on permission on allns mode", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:81538-[Skipped:Disconnected]preflight check on permission on allns mode"), func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") {
			g.Skip("NewOLMPreflightPermissionChecks feature gate is disabled. This test requires preflight permission validation to be enabled.")
		}
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:81664-[OTP]preflight check on permission on own ns mode", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:81664-[Skipped:Disconnected]preflight check on permission on own ns mode"), func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") ||
			!olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("Required feature gates are disabled: NewOLMPreflightPermissionChecks and NewOLMOwnSingleNamespace must both be enabled for this test.")
		}
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:81696-[OTP]preflight check on permission on single ns mode", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:81696-[Skipped:Disconnected]preflight check on permission on single ns mode"), func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMPreflightPermissionChecks") ||
			!olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("Required feature gates are disabled: NewOLMPreflightPermissionChecks and NewOLMOwnSingleNamespace must both be enabled for this test.")
		}
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:74618-[OTP]ClusterExtension supports simple registry vzero bundles only", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:74618-[Skipped:Disconnected]ClusterExtension supports simple registry vzero bundles only"), func() {
		exutil.SkipForSNOCluster(oc)
		olmv1util.ValidateAccessEnvironment(oc)
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

	g.It("PolarionID:83026-[OTP][Skipped:Disconnected]clusterextension updates sometimes failed with the following error from the CRDUpgradeCheck resource unknown change and refusing to determine that change is safe", g.Label("original-name:[sig-olmv1][Jira:OLM] clusterextension PolarionID:83026-[Skipped:Disconnected]clusterextension updates sometimes failed with the following error from the CRDUpgradeCheck resource unknown change and refusing to determine that change is safe"), func() {
		baseDir := exutil.FixturePath("testdata", "olm")
		clusterextensionTemplate := filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
		saAdminTemplate := filepath.Join(baseDir, "sa-admin.yaml")
		g.By("1)install Argocd operator v0.4.0 in a random namespace")
		sa := "argocd-83026"
		oc.SetupProject()

		saCrb := olmv1util.SaCLusterRolebindingDescription{
			Name:      sa,
			Namespace: oc.Namespace(),
			Template:  saAdminTemplate,
		}
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		ceArgocd := olmv1util.ClusterExtensionDescription{
			Name:             "extension-argocd-83026",
			PackageName:      "argocd-operator",
			Channel:          "alpha",
			Version:          "v0.4.0",
			InstallNamespace: oc.Namespace(),
			SaName:           sa,
			LabelKey:         "olm.operatorframework.io/metadata.name",
			LabelValue:       "openshift-community-operators",
			Template:         clusterextensionTemplate,
		}
		defer ceArgocd.Delete(oc)
		ceArgocd.Create(oc)

		g.By("2)upgrade it to v0.5.0")
		if err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("clusterextension", "extension-argocd-83026", "-p", "{\"spec\": {\"source\": {\"catalog\": {\"version\": \"v0.5.0\"}}}}", "--type=merge").Execute(); err != nil {
			e2e.Failf("patch clusterextension failed:%v", err)
		}
		ceArgocd.WaitProgressingMessage(oc, "Desired state reached")
		g.By("3)upgrade it to v0.7.0")
		if err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("clusterextension", "extension-argocd-83026", "-p", "{\"spec\": {\"source\": {\"catalog\": {\"version\": \"v0.7.0\"}}}}", "--type=merge").Execute(); err != nil {
			e2e.Failf("patch clusterextension failed:%v", err)
		}
		ceArgocd.WaitProgressingMessage(oc, "Desired state reached")
	})

	g.It("PolarionID:69196-[OTP][Level0][Skipped:Disconnected]Supports Version Ranges during clusterextension upgrade", func() {
		var (
			caseID                       = "69196"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-69196"
			sa                           = "sa69196"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-69196",
				LabelValue: labelValue,
				Imageref:   "quay.io/olmqe/olmtest-operator-index:nginxolm69196",
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-69196",
				InstallNamespace: ns,
				PackageName:      "nginx69196",
				Channel:          "candidate-v1.0",
				Version:          "1.0.1",
				SaName:           sa,
				LabelValue:       labelValue,
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

		g.By("Create clusterextension with channel candidate-v1.0, version 1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("update version to be 1.0.3")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version": "1.0.3"}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			conditions, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.conditions}")
			if !strings.Contains(conditions, "error upgrading") {
				e2e.Logf("error message is not raised")
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "error message is not raised")
		}

		g.By("update version to be >=1.0.1")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version": ">=1.0.1"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			resolvedBundle, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.install.bundle.name}")
			if !strings.Contains(resolvedBundle, "v1.0.2") {
				e2e.Logf("clusterextension.resolvedBundle is %s, not v1.0.2, and try next", resolvedBundle)
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "clusterextension resolvedBundle is not v1.0.2")
		}
		conditions, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.conditions}")
		o.Expect(strings.ToLower(conditions)).To(o.ContainSubstring("desired state reached"))
		o.Expect(conditions).NotTo(o.ContainSubstring("error"))

		g.By("update channel to be candidate-v1.1")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v1.1"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			resolvedBundle, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.install.bundle.name}")
			if !strings.Contains(resolvedBundle, "v1.1.0") {
				e2e.Logf("clusterextension.resolvedBundle is %s, not v1.1.0, and try next", resolvedBundle)
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "clusterextension resolvedBundle is not v1.1.0")
		}
	})

	g.It("PolarionID:68821-[OTP][Skipped:Disconnected]Supports Version Ranges during Installation", func() {
		var (
			caseID                                        = "68821"
			labelValue                                    = caseID
			baseDir                                       = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate                        = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate                      = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			clusterextensionWithoutChannelTemplate        = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannel.yaml")
			clusterextensionWithoutChannelVersionTemplate = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannelVersion.yaml")
			saClusterRoleBindingTemplate                  = filepath.Join(baseDir, "sa-admin.yaml")
			ns                                            = "ns-68821"
			sa                                            = "sa68821"
			saCrb                                         = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-68821",
				LabelValue: labelValue,
				Imageref:   "quay.io/olmqe/olmtest-operator-index:nginxolm68821",
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-68821",
				PackageName:      "nginx68821",
				Channel:          "candidate-v0.0",
				Version:          ">=0.0.1",
				LabelValue:       labelValue,
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

		g.By("Create clusterextension with channel candidate-v0.0, version >=0.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v0.0.3"))
		clusterextension.Delete(oc)

		g.By("Create clusterextension with channel candidate-v1.0, version 1.0.x")
		clusterextension.Channel = "candidate-v1.0"
		clusterextension.Version = "1.0.x"
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.2"))
		clusterextension.Delete(oc)

		g.By("Create clusterextension with channel empty, version >=0.0.1 !=1.1.0 <1.1.2")
		clusterextension.Channel = ""
		clusterextension.Version = ">=0.0.1 !=1.1.0 <1.1.2"
		clusterextension.Template = clusterextensionWithoutChannelTemplate
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.2"))
		clusterextension.Delete(oc)

		g.By("Create clusterextension with channel empty, version empty")
		clusterextension.Channel = ""
		clusterextension.Version = ""
		clusterextension.Template = clusterextensionWithoutChannelVersionTemplate
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.1.0"))
		clusterextension.Delete(oc)

		g.By("Create clusterextension with invalid version")
		clusterextension.Version = "!1.0.1"
		clusterextension.Template = clusterextensionTemplate
		err = clusterextension.CreateWithoutCheck(oc)
		o.Expect(err).To(o.HaveOccurred())

	})

	g.It("PolarionID:74108-[OTP][Skipped:Disconnected][Slow]olm v1 supports legacy upgrade edges", func() {
		var (
			caseID                       = "74108"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutVersion.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-74108"
			sa                           = "sa74108"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-74108",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm74108",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-74108",
				InstallNamespace: ns,
				PackageName:      "nginx74108",
				Channel:          "candidate-v0.0",
				LabelValue:       labelValue,
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

		g.By("1) Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2) Install clusterextension with channel candidate-v0.0")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("0.0.2"))

		g.By("3) Attempt to update to channel candidate-v2.1 with CatalogProvided policy, that should fail")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v2.1"]}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")]}`)
			if strings.Contains(message, "error upgrading") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
		}
		exutil.AssertWaitPollNoErr(errWait, "no error message raised")

		g.By("4) Attempt to update to channel candidate-v0.1 with CatalogProvided policy, that should success")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v0.1"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "0.1.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx74108 0.1.0 is not installed")

		g.By("5) Attempt to update to channel candidate-v1.0 with CatalogProvided policy, that should fail")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v1.0"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")]}`)
			if strings.Contains(message, "error upgrading") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "no error message raised")

		g.By("6) update policy to SelfCertified, upgrade should success")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"upgradeConstraintPolicy": "SelfCertified"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.0.2") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx74108 1.0.2 is not installed")

		g.By("7) Attempt to update to channel candidate-v1.1 with CatalogProvided policy, that should success")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"upgradeConstraintPolicy": "CatalogProvided"}}}}`)
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v1.1"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.1.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx74108 0.1.0 is not installed")

		g.By("8) Attempt to update to channel candidate-v1.2 with CatalogProvided policy, that should fail")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v1.2"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")]}`)
			if strings.Contains(message, "error upgrading") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "no error message raised")

		g.By("9) update policy to SelfCertified, upgrade should success")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"upgradeConstraintPolicy": "SelfCertified"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.2.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx74108 1.2.0 is not installed")

		g.By("10) Attempt to update to channel candidate-v2.0 with CatalogProvided policy, that should fail")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"upgradeConstraintPolicy": "CatalogProvided"}}}}`)
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v2.0"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")]}`)
			if strings.Contains(message, "error upgrading") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "no error message raised")

		g.By("11) Attempt to update to channel candidate-v2.1 with CatalogProvided policy, that should success")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v2.1"]}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "2.1.1") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx74108 2.1.1 is not installed")

		g.By("8) downgrade to version 1.0.1 with SelfCertified policy, that should work")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"upgradeConstraintPolicy": "SelfCertified"}}}}`)
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"channels": ["candidate-v1.0"],"version":"1.0.1"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.0.1") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
		}
		exutil.AssertWaitPollNoErr(errWait, "nginx74108 1.0.1 is not installed")

	})

	g.It("PolarionID:74923-[OTP][Skipped:Disconnected]no two ClusterExtensions can manage the same underlying object", func() {
		var (
			caseID                       = "74923"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannelVersion.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns1                          = "ns-74923-1"
			ns2                          = "ns-74923-2"
			sa1                          = "sa74923-1"
			sa2                          = "sa74923-2"
			saCrb1                       = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa1,
				Namespace: ns1,
				Template:  saClusterRoleBindingTemplate,
			}
			saCrb2 = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa2,
				Namespace: ns2,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-74923-1",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm74923",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension1 = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-74923-1",
				PackageName:      "nginx74923",
				InstallNamespace: ns1,
				SaName:           sa1,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
			clusterextension2 = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-74923-2",
				PackageName:      "nginx74923",
				InstallNamespace: ns2,
				SaName:           sa2,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("1. Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2. Create clusterextension1")
		g.By("2.1 Create namespace 1")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns1, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns1).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns1)).To(o.BeTrue())

		g.By("2.2 Create SA for clusterextension1")
		defer saCrb1.Delete(oc)
		saCrb1.Create(oc)

		g.By("2.3 Create clusterextension1")
		defer clusterextension1.Delete(oc)
		clusterextension1.Create(oc)
		o.Expect(clusterextension1.InstalledBundle).To(o.ContainSubstring("v1.0.2"))

		g.By("3 Create clusterextension2")
		g.By("3.1 Create namespace 2")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns2, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns2).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns2)).To(o.BeTrue())

		g.By("3.2 Create SA for clusterextension2")
		defer saCrb2.Delete(oc)
		saCrb2.Create(oc)

		g.By("3.3 Create clusterextension2")
		defer clusterextension2.Delete(oc)
		_ = clusterextension2.CreateWithoutCheck(oc)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension2.Name, "-o", "jsonpath={.status.conditions[*].message}")
			if !strings.Contains(message, "already exists in namespace") {
				e2e.Logf("status is %s", message)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "clusterextension2 should not be installed")
		clusterextension2.Delete(oc)
		clusterextension1.Delete(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			status, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("crd", "nginxolm74923s.cache.example.com").Output()
			if !strings.Contains(status, "NotFound") {
				e2e.Logf("crd status: %s", status)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "crd nginxolm74923s.cache.example.com is not deleted")

		g.By("4 Create crd")
		crdFilePath := filepath.Join(baseDir, "crd-nginxolm74923.yaml")
		defer func() {
			_, _ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("crd", "nginxolm74923s.cache.example.com").Output()
		}()
		_, _ = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", crdFilePath).Output()
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			status, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("crd", "nginxolm74923s.cache.example.com").Output()
			if strings.Contains(status, "NotFound") {
				e2e.Logf("crd status: %s", status)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "crd nginxolm74923s.cache.example.com is not deleted")

		_ = clusterextension1.CreateWithoutCheck(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension1.Name, "-o", "jsonpath={.status.conditions[*].message}")
			if !strings.Contains(message, "already exists in namespace") {
				e2e.Logf("status is %s", message)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "clusterextension1 should not be installed")

	})

	g.It("PolarionID:75501-[OTP][Skipped:Disconnected]the updates of various status fields is orthogonal", func() {
		var (
			caseID                       = "75501"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-75501"
			sa                           = "sa75501"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       "clustercatalog-75501",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm75501",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-75501",
				InstallNamespace: ns,
				PackageName:      "nginx75501",
				Channel:          "candidate-v2.1",
				Version:          "2.1.0",
				SaName:           sa,
				LabelValue:       labelValue,
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

		g.By("Create clusterextension with channel candidate-v2.1, version 2.1.0")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
		reason, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].reason}`)
		o.Expect(reason).To(o.ContainSubstring("Succeeded"))
		status, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Installed")].status}`)
		o.Expect(status).To(o.ContainSubstring("True"))
		reason, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Installed")].reason}`)
		o.Expect(reason).To(o.ContainSubstring("Succeeded"))
		installedBundleVersion, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.install.bundle.version}`)
		o.Expect(installedBundleVersion).To(o.ContainSubstring("2.1.0"))
		installedBundleName, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.install.bundle.name}`)
		o.Expect(installedBundleName).To(o.ContainSubstring("nginx75501.v2.1.0"))
		resolvedBundleVersion, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.install.bundle.version}`)
		o.Expect(resolvedBundleVersion).To(o.ContainSubstring("2.1.0"))
		resolvedBundleName, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.install.bundle.name}`)
		o.Expect(resolvedBundleName).To(o.ContainSubstring("nginx75501.v2.1.0"))

		clusterextension.Delete(oc)

		g.By("Test UnpackFailed, bundle image cannot be pulled successfully")
		clusterextension.Channel = "candidate-v2.0"
		clusterextension.Version = "2.0.0"
		_ = clusterextension.CreateWithoutCheck(oc)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			unpackedReason, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].reason}`)
			unpackedMessage, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].message}`)
			if !strings.Contains(unpackedReason, "Retrying") || !strings.Contains(unpackedMessage, "manifest unknown") {
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "clusterextension status is not correct")
		}
		clusterextension.Delete(oc)

		g.By("Test ResolutionFailed, wrong version")
		clusterextension.Version = "3.0.0"
		_ = clusterextension.CreateWithoutCheck(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			resolvedReason, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].reason}`)
			resolvedMessage, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].message}`)
			if !strings.Contains(resolvedReason, "Retrying") || !strings.Contains(resolvedMessage, "no bundles found for package") {
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "clusterextension status is not correct")
		}
		clusterextension.Delete(oc)

		g.By("Test ResolutionFailed, no package")
		clusterextension.PackageName = "nginxfake"
		_ = clusterextension.CreateWithoutCheck(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			resolvedReason, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].reason}`)
			resolvedMessage, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")].message}`)
			if !strings.Contains(resolvedReason, "Retrying") || !strings.Contains(resolvedMessage, "no bundles found for package") {
				return false, nil
			}
			return true, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
			exutil.AssertWaitPollNoErr(errWait, "clusterextension status is not correct")
		}

	})

	g.It("PolarionID:76685-[OTP][Skipped:Disconnected]olm v1 supports selecting catalogs [Serial]", func() {
		var (
			baseDir                                  = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate                   = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate                 = filepath.Join(baseDir, "clusterextensionWithoutChannelVersion.yaml")
			clusterextensionLabelTemplate            = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannelVersion.yaml")
			clusterextensionExpressionsTemplate      = filepath.Join(baseDir, "clusterextension-withselectorExpressions-WithoutChannelVersion.yaml")
			clusterextensionLableExpressionsTemplate = filepath.Join(baseDir, "clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml")

			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-76685"
			sa                           = "sa76685"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog1 = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: "ocp-76685-1",
				Name:       "clustercatalog-76685-1",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginx76685v1",
				Template:   clustercatalogTemplate,
			}
			clustercatalog2 = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: "ocp-76685-2",
				Name:       "clustercatalog-76685-2",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginx76685v2",
				Template:   clustercatalogTemplate,
			}
			clustercatalog3 = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: "ocp-76685-3",
				Name:       "clustercatalog-76685-3",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginx76685v3",
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-76685",
				InstallNamespace: ns,
				PackageName:      "nginx76685",
				SaName:           sa,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("1) Create namespace, sa, clustercatalog1 and clustercatalog2")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		defer clustercatalog1.Delete(oc)
		clustercatalog1.Create(oc)
		defer clustercatalog2.Delete(oc)
		clustercatalog2.Create(oc)

		g.By("2) 2 clustercatalogs with same priority, install clusterextension, selector of clusterextension is empty")
		defer clusterextension.Delete(oc)
		_ = clusterextension.CreateWithoutCheck(oc)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", `jsonpath={.status.conditions[?(@.type=="Progressing")]}`)
			if strings.Contains(message, "multiple catalogs with the same priority") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}")
		}
		exutil.AssertWaitPollNoErr(errWait, "no error message raised")
		clusterextension.Delete(oc)

		g.By("3) 2 clustercatalogs with same priority, install clusterextension, selector of clusterextension is not empty")
		clusterextension.Template = clusterextensionLabelTemplate
		clusterextension.LabelKey = "olm.operatorframework.io/metadata.name"
		clusterextension.LabelValue = clustercatalog1.Name
		clusterextension.Create(oc)
		clusterextension.WaitClusterExtensionVersion(oc, "v1.0.1")
		clusterextension.Delete(oc)

		g.By("4) Install 2 clustercatalogs with different priorities, and the selector of  clusterextension is empty")
		clustercatalog1.Patch(oc, `{"spec":{"priority": 100}}`)
		clustercatalog2.Patch(oc, `{"spec":{"priority": 1000}}`)
		clusterextension.Template = clusterextensionTemplate
		clusterextension.Create(oc)
		clusterextension.WaitClusterExtensionVersion(oc, "v2.0.0")
		clusterextension.Delete(oc)

		g.By("5) Install 2 clustercatalogs with different priorities, and the selector of clusterextension is not empty")
		clusterextension.Template = clusterextensionLabelTemplate
		clusterextension.LabelKey = "olm.operatorframework.io/metadata.name"
		clusterextension.LabelValue = clustercatalog1.Name
		clusterextension.Create(oc)
		clusterextension.WaitClusterExtensionVersion(oc, "v1.0.1")

		g.By("6) add ClusterCatalog 3, and modify the selector of clusterextension to use ClusterCatalog 3")
		defer clustercatalog3.Delete(oc)
		clustercatalog3.Create(oc)
		clusterextension.LabelKey = clustercatalog3.LabelKey
		clusterextension.LabelValue = clustercatalog3.LabelValue
		clusterextension.Create(oc)
		clusterextension.WaitClusterExtensionVersion(oc, "v3.0.0")
		clusterextension.Delete(oc)

		g.By("7) matchExpressions")
		clusterextension.Template = clusterextensionExpressionsTemplate
		clusterextension.ExpressionsKey = clustercatalog3.LabelKey
		clusterextension.ExpressionsOperator = "NotIn"
		clusterextension.ExpressionsValue1 = clustercatalog3.LabelValue
		clusterextension.Create(oc)
		clusterextension.WaitClusterExtensionVersion(oc, "v2.0.0")

		g.By("8) test both matchLabels and matchExpressions")
		clusterextension.Template = clusterextensionLableExpressionsTemplate
		clusterextension.LabelKey = "olm.operatorframework.io/metadata.name"
		clusterextension.LabelValue = clustercatalog3.Name
		clusterextension.ExpressionsKey = clustercatalog3.LabelKey
		clusterextension.ExpressionsOperator = "In"
		clusterextension.ExpressionsValue1 = clustercatalog1.LabelValue
		clusterextension.ExpressionsValue2 = clustercatalog2.LabelValue
		clusterextension.ExpressionsValue3 = clustercatalog3.LabelValue
		clusterextension.Create(oc)
		clusterextension.WaitClusterExtensionVersion(oc, "v3.0.0")

	})

	g.It("PolarionID:77972-[OTP][Skipped:Disconnected]olm v1 Supports MaxOCPVersion in properties file", func() {
		var (
			caseID                       = "77972"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-77972"
			sa                           = "sa77972"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: labelValue,
				Name:       "clustercatalog-77972",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm77972",
				Template:   clustercatalogTemplate,
			}

			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-77972",
				InstallNamespace: ns,
				PackageName:      "nginx77972",
				SaName:           sa,
				Version:          "0.0.1",
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("1) Create namespace, sa, clustercatalog1 and clustercatalog2")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2) install clusterextension, version 0.0.1, without setting olm.maxOpenShiftVersion")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v0.0.1"))
		status, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].status}`)
		o.Expect(status).To(o.ContainSubstring("True"))
		message, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].message}`)
		o.Expect(message).To(o.ContainSubstring("All is well"))

		g.By("3) upgrade clusterextension to 1.1.0, olm.maxOpenShiftVersion is 4.19")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.1.0"}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 60*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].message}`)
			if strings.Contains(message, "InstalledOLMOperatorsUpgradeable") && strings.Contains(message, "nginx77972.v1.1.0") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		status, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].status}`)
		o.Expect(status).To(o.ContainSubstring("False"))
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o=jsonpath-as-json={.status.conditions}")
		}
		exutil.AssertWaitPollNoErr(errWait, "Upgradeable message is not correct")

		g.By("4) upgrade clusterextension to 1.2.0, olm.maxOpenShiftVersion is 4.20")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.2.0"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 60*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].message}`)
			if strings.Contains(message, "InstalledOLMOperatorsUpgradeable") && strings.Contains(message, "nginx77972.v1.2.0") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		status, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].status}`)
		o.Expect(status).To(o.ContainSubstring("False"))
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o=jsonpath-as-json={.status.conditions}")
		}
		exutil.AssertWaitPollNoErr(errWait, "Upgradeable message is not correct")
	})

	g.It("PolarionID:82249-[OTP][Skipped:Disconnected]Verify olmv1 support for float type maxOCPVersion in properties file", func() {
		var (
			caseID                       = "82249"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-82249"
			sa                           = "sa82249"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				Name:       "clustercatalog-82249",
				LabelValue: labelValue,
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm82249",
				Template:   clustercatalogTemplate,
			}

			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-82249",
				InstallNamespace: ns,
				PackageName:      "nginx82249",
				SaName:           sa,
				Version:          "0.0.1",
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("1) Create namespace, sa, clustercatalog")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2) install clusterextension, version 0.0.1, without setting olm.maxOpenShiftVersion")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v0.0.1"))
		status, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].status}`)
		o.Expect(status).To(o.ContainSubstring("True"))
		message, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].message}`)
		o.Expect(message).To(o.ContainSubstring("All is well"))

		g.By("3) upgrade clusterextension to 1.2.0, olm.maxOpenShiftVersion is 4.20")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.2.0"}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 60*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].message}`)
			if strings.Contains(message, "InstalledOLMOperatorsUpgradeable") && strings.Contains(message, "nginx82249.v1.2.0") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		status, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].status}`)
		o.Expect(status).To(o.ContainSubstring("False"))
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o=jsonpath-as-json={.status.conditions}")
		}
		exutil.AssertWaitPollNoErr(errWait, "Upgradeable message is not correct")

		g.By("4) upgrade clusterextension to 1.3.0, olm.maxOpenShiftVersion is 4.21")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.3.0"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 60*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].message}`)
			if strings.Contains(message, "InstalledOLMOperatorsUpgradeable") && strings.Contains(message, "nginx82249.v1.3.0") {
				e2e.Logf("status is %s", message)
				return true, nil
			}
			return false, nil
		})
		status, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o", `jsonpath={.status.conditions[?(@.type=="Upgradeable")].status}`)
		o.Expect(status).To(o.ContainSubstring("False"))
		if errWait != nil {
			_, _ = olmv1util.GetNoEmpty(oc, "co", "olm", "-o=jsonpath-as-json={.status.conditions}")
		}
		exutil.AssertWaitPollNoErr(errWait, "Upgradeable message is not correct")

		g.By("5) Test PASS")

	})

	g.It("PolarionID:80117-[OTP][Skipped:Disconnected] Single Namespace Install Mode should be supported", func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("SingleOwnNamespaceInstallSupport is not enable, so skip it")
		}
		var (
			caseID                            = "80117"
			labelValue                        = caseID
			baseDir                           = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate            = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionOwnSingleTemplate = filepath.Join(baseDir, "clusterextension-withselectorlabel-withoutChannel-OwnSingle.yaml")
			clusterextensionTemplate          = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannel.yaml")

			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-80117"
			nsWatch                      = "ns-80117-watch"
			sa                           = "sa80117"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: labelValue,
				Name:       "clustercatalog-80117",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm80117",
				Template:   clustercatalogTemplate,
			}

			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-80117",
				InstallNamespace: ns,
				PackageName:      "nginx80117",
				SaName:           sa,
				Version:          "1.0.1",
				WatchNamespace:   nsWatch,
				LabelValue:       labelValue,
				Template:         clusterextensionOwnSingleTemplate,
			}
			clusterextensionAllNs = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-80117",
				InstallNamespace: ns,
				PackageName:      "nginx80117",
				SaName:           sa,
				Version:          "1.1.0",
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("1) Create namespace, sa, clustercatalog")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2) install clusterextension, version 1.0.1, without creating watch ns")
		defer clusterextension.Delete(oc)
		_ = clusterextension.CreateWithoutCheck(oc)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.conditions[*].message}")
			if !strings.Contains(message, "failed to create resource") {
				e2e.Logf("status is %s", message)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "status is not correct")
		clusterextension.Delete(oc)

		g.By("3) create watch ns")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsWatch, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsWatch).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsWatch)).To(o.BeTrue())

		g.By("4) create clusterextension, version 1.0.1")
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))

		g.By("4.1) check deployment")
		deploymentTargetNS, _ := olmv1util.GetNoEmpty(oc, "deployment", "nginx80117-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`)
		o.Expect(deploymentTargetNS).To(o.ContainSubstring(nsWatch))
		g.By("4.2) check rolebinding")
		rdNS, _ := olmv1util.GetNoEmpty(oc, "rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", nsWatch, "-o", `jsonpath={..subjects[].namespace}`)
		o.Expect(rdNS).To(o.ContainSubstring(ns))

		g.By("5) upgrade clusterextension to 1.0.2, v1.0.2 only support singleNamespace")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.0.2"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.0.2") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80117 1.0.2 is not installed")
		g.By("5.1) check deployment")
		deploymentTargetNS, _ = olmv1util.GetNoEmpty(oc, "deployment", "nginx80117-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`)
		o.Expect(deploymentTargetNS).To(o.ContainSubstring(nsWatch))
		g.By("5.2) check rolebinding")
		rdNS, _ = olmv1util.GetNoEmpty(oc, "rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", nsWatch, "-o", `jsonpath={..subjects[].namespace}`)
		o.Expect(rdNS).To(o.ContainSubstring(ns))

		g.By("6) upgrade clusterextension to 1.1.0, support allnamespace")
		clusterextensionAllNs.Create(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.1.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80117 1.1.0 is not installed")
		g.By("6.1) check deployment")
		deploymentTargetNS, _ = oc.AsAdmin().WithoutNamespace().Run("get").Args("deployment", "nginx80117-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`).Output()
		o.Expect(deploymentTargetNS).To(o.BeEmpty())
		g.By("6.2) check rolebinding")
		rdNS, _ = oc.AsAdmin().WithoutNamespace().Run("get").Args("rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", nsWatch).Output()
		o.Expect(rdNS).To(o.ContainSubstring("No resources found"))

		g.By("7) upgrade clusterextension to 2.0.0, support singleNamespace")
		clusterextension.Version = "2.0.0"
		clusterextension.Create(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "2.0.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80117 2.0.0 is not installed")
		g.By("7.1) check deployment")
		deploymentTargetNS, _ = olmv1util.GetNoEmpty(oc, "deployment", "nginx80117-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`)
		o.Expect(deploymentTargetNS).To(o.ContainSubstring(nsWatch))
		g.By("7.2) check rolebinding")
		rdNS, _ = olmv1util.GetNoEmpty(oc, "rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", nsWatch, "-o", `jsonpath={..subjects[].namespace}`)
		o.Expect(rdNS).To(o.ContainSubstring(ns))

		g.By("8) check not support install two same clusterextensions")
		ns2 := ns + "-2"
		nsWatch2 := nsWatch + "-2"
		sa2 := "sa80117-2"
		saCrb2 := olmv1util.SaCLusterRolebindingDescription{
			Name:      sa2,
			Namespace: ns2,
			Template:  saClusterRoleBindingTemplate,
		}

		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns2, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns2).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns2)).To(o.BeTrue())
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsWatch2, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsWatch2).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsWatch2)).To(o.BeTrue())

		defer saCrb2.Delete(oc)
		saCrb2.Create(oc)
		clusterextension2 := olmv1util.ClusterExtensionDescription{
			Name:             "clusterextension-80117-2",
			InstallNamespace: ns2,
			PackageName:      "nginx80117",
			SaName:           sa2,
			Version:          "1.0.1",
			WatchNamespace:   nsWatch2,
			LabelKey:         "olmv1-test",
			LabelValue:       labelValue,
			Template:         clusterextensionOwnSingleTemplate,
		}
		defer clusterextension2.Delete(oc)
		_ = clusterextension2.CreateWithoutCheck(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension2.Name, "-o", "jsonpath={.status.conditions[*].message}")
			if !strings.Contains(message, "already exists") {
				e2e.Logf("status is %s", message)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "status is not correct")

		g.By("9) Test SUCCESS")

	})

	g.It("PolarionID:80120-[OTP][Skipped:Disconnected] Own Namespace Install Mode should be supported", func() {
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("SingleOwnNamespaceInstallSupport is not enable, so skip it")
		}
		var (
			caseID                            = "80120"
			labelValue                        = caseID
			baseDir                           = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate            = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionOwnSingleTemplate = filepath.Join(baseDir, "clusterextension-withselectorlabel-withoutChannel-OwnSingle.yaml")
			clusterextensionTemplate          = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannel.yaml")

			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-80120"
			sa                           = "sa80120"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: labelValue,
				Name:       "clustercatalog-80120",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm80120",
				Template:   clustercatalogTemplate,
			}

			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-80120",
				InstallNamespace: ns,
				PackageName:      "nginx80120",
				SaName:           sa,
				Version:          "1.0.1",
				LabelKey:         "olmv1-test",
				LabelValue:       labelValue,
				WatchNamespace:   ns,
				Template:         clusterextensionOwnSingleTemplate,
			}
			clusterextensionAllNs = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-80120",
				InstallNamespace: ns,
				PackageName:      "nginx80120",
				SaName:           sa,
				Version:          "3.0.0",
				LabelKey:         "olmv1-test",
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
			}
		)

		g.By("1) Create namespace, sa, clustercatalog")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2) install clusterextension, version 1.0.1")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("v1.0.1"))
		g.By("2.1) check deployment")
		deploymentTargetNS, _ := olmv1util.GetNoEmpty(oc, "deployment", "nginx80120-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`)
		o.Expect(deploymentTargetNS).To(o.ContainSubstring(ns))
		g.By("2.2) check rolebinding")
		rdNS, _ := olmv1util.GetNoEmpty(oc, "rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", ns, "-o", `jsonpath={..subjects[].namespace}`)
		o.Expect(rdNS).To(o.ContainSubstring(ns))

		g.By("3) upgrade clusterextension to 1.0.2, v1.0.2 only support OwnNamespace")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.0.2"}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.0.2") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80120 1.0.2 is not installed")

		g.By("4) upgrade clusterextension to 3.0.0, support allnamespace")
		clusterextensionAllNs.Create(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "3.0.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80120 3.0.0 is not installed")
		g.By("4.1) check deployment")
		deploymentTargetNS, _ = oc.AsAdmin().WithoutNamespace().Run("get").Args("deployment", "nginx80120-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`).Output()
		o.Expect(deploymentTargetNS).To(o.BeEmpty())
		g.By("4.2) check rolebinding")
		rdNS, _ = oc.AsAdmin().WithoutNamespace().Run("get").Args("rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", ns).Output()
		o.Expect(rdNS).To(o.ContainSubstring("No resources found"))

		g.By("5) upgrade clusterextension to 4.0.0, support OwnNamespace")
		clusterextension.Version = "4.0.0"
		clusterextension.Create(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "4.0.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80120 4.0.0 is not installed")
		g.By("5.1) check deployment")
		deploymentTargetNS, _ = olmv1util.GetNoEmpty(oc, "deployment", "nginx80120-controller-manager", "-n", ns, "-o", `jsonpath={.spec.template.metadata.annotations.olm\.targetNamespaces}`)
		o.Expect(deploymentTargetNS).To(o.ContainSubstring(ns))
		g.By("5.2) check rolebinding")
		rdNS, _ = olmv1util.GetNoEmpty(oc, "rolebinding", "-l", "olm.operatorframework.io/owner-name="+clusterextension.Name, "-n", ns, "-o", `jsonpath={..subjects[].namespace}`)
		o.Expect(rdNS).To(o.ContainSubstring(ns))

		g.By("6) if the annotations is not correct, error should be raised")
		clusterextension.Delete(oc)
		clusterextension = olmv1util.ClusterExtensionDescription{
			Name:             "clusterextension-80120",
			InstallNamespace: ns,
			PackageName:      "nginx80120",
			SaName:           sa,
			Version:          "1.0.1",
			WatchNamespace:   ns + "flake",
			LabelKey:         "olmv1-test",
			LabelValue:       labelValue,
			Template:         clusterextensionOwnSingleTemplate,
		}
		_ = clusterextension.CreateWithoutCheck(oc)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.conditions[*].message}")
			if !strings.Contains(message, "invalid configuration") {
				e2e.Logf("status is %s", message)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx80120 status is not correct")

		g.By("7) Test SUCCESS")

	})

	g.It("PolarionID:82136-[OTP][Skipped:Disconnected]olm v1 supports NetworkPolicy resources", func() {
		var (
			caseID                       = "82136"
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-WithoutChannel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			ns                           = "ns-82136"
			sa                           = "sa82136"
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				LabelKey:   "olmv1-test",
				LabelValue: labelValue,
				Name:       "clustercatalog-82136",
				Imageref:   "quay.io/openshifttest/nginxolm-operator-index:nginxolm82136",
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             "clusterextension-82136",
				InstallNamespace: ns,
				PackageName:      "nginx82136",
				Version:          "1.0.1",
				SaName:           sa,
				LabelKey:         "olmv1-test",
				LabelValue:       labelValue,
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

		g.By("1) Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("2) Installnginx82136.v1.0.1, no networkpolicy")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		o.Expect(clusterextension.InstalledBundle).To(o.ContainSubstring("1.0.1"))
		networkpolicies, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("networkpolicy", "-n", ns).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(networkpolicies).To(o.ContainSubstring("No resources found"))

		g.By("3) upgrade to nginx82136.v1.1.0, 1 networkpolicy, allow all ingress and all egress traffic")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"1.1.0"}}}}`)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "1.1.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx82136 1.1.0 is not installed")
		networkpolicies, err = oc.WithoutNamespace().AsAdmin().Run("get").Args("networkpolicy", "-n", ns).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(networkpolicies).To(o.ContainSubstring("nginx82136-controller-acceptall"))

		g.By("4) upgrade to nginx82136.v2.0.0, 2 networkpolicy, one default deny all traffic, one for controller-manager")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"2.0.0"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "2.0.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx82136 2.2.0 is not installed")
		networkpolicies, err = oc.WithoutNamespace().AsAdmin().Run("get").Args("networkpolicy", "-n", ns).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(networkpolicies).To(o.ContainSubstring("default-deny-all"))
		o.Expect(networkpolicies).To(o.ContainSubstring("nginx82136-controller"))
		o.Expect(networkpolicies).NotTo(o.ContainSubstring("nginx82136-controller-acceptall"))

		g.By("5) upgrade to nginx82136.v2.1.0, wrong networkpolicy")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"2.1.0"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			message, _ := olmv1util.GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.conditions[*].message}")
			if !strings.Contains(message, "Unsupported value") {
				e2e.Logf("status is %s", message)
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx82136.v2.1.0 should not be installed, wrong error message")

		g.By("6) upgrade to nginx82136.v2.2.0, no networkpolicy")
		clusterextension.Patch(oc, `{"spec":{"source":{"catalog":{"version":"2.2.0"}}}}`)
		errWait = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
			clusterextension.GetBundleResource(oc)
			if strings.Contains(clusterextension.InstalledBundle, "2.2.0") {
				e2e.Logf("InstalledBundle is %s", clusterextension.InstalledBundle)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "nginx82136 2.2.0 is not installed")
		networkpolicies, err = oc.WithoutNamespace().AsAdmin().Run("get").Args("networkpolicy", "-n", ns).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(networkpolicies).To(o.ContainSubstring("No resources found"))

		g.By("7) Test SUCCESS")
	})

})
