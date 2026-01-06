package specs

import (
	"path/filepath"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
	olmv1util "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util/olmv1util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] clusterextension watchNamespace configuration", g.Label("NonHyperShiftHOST"), func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("default")
	)

	g.BeforeEach(func() {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMOwnSingleNamespace") {
			g.Skip("NewOLMOwnSingleNamespace feature gate is disabled. This test requires watchNamespace configuration support.")
		}
	})

	g.It("PolarionID:85510-watchNamespace configuration with AllNamespaces InstallMode", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                         = "85510"
			ns                             = "ns-" + caseID
			nsTarget                       = "ns-" + caseID + "-target"
			sa                             = "sa" + caseID
			labelValue                     = caseID
			catalogName                    = "clustercatalog-" + caseID
			baseDir                        = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate         = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate       = filepath.Join(baseDir, "clusterextension.yaml")
			clusterextensionConfigTemplate = filepath.Join(baseDir, "clusterextension-watchns-config.yaml")
			saClusterRoleBindingTemplate   = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                          = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv85510",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
		)

		g.By("Create install namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create target namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsTarget, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsTarget).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsTarget)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("=== ServiceAccount resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ServiceAccount", sa, "-n", ns, "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRole resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRole", sa+"-installer-admin-clusterrole", "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRoleBinding resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRoleBinding", sa+"-installer-admin-clusterrole-binding", "-o", "yaml").Execute()

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("=== ClusterCatalog resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clustercatalog", catalogName, "-o", "yaml").Execute()

		g.By("Scenario 1-1: No config - expect AllNamespaces mode success")
		e2e.Logf("Testing ClusterExtension with {AllNamespaces} InstallMode and no watchNamespace config")
		ceNoConfig := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-noconfig",
			PackageName:      "nginx-ok-v85510",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			Template:         clusterextensionTemplate,
		}
		defer ceNoConfig.Delete(oc)
		ceNoConfig.Create(oc)
		e2e.Logf("=== ClusterExtension NoConfig resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNoConfig.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 1-1: ClusterExtension installed successfully in AllNamespaces mode without config")
		ceNoConfig.Delete(oc)
		e2e.Logf("Scenario 1-1 cleanup: ClusterExtension deleted")

		g.By("Scenario 1-2: Empty string watchNamespace")
		ceEmptyString := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-empty",
			PackageName:      "nginx-ok-v85510",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   "",
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEmptyString.Delete(oc)
		_ = ceEmptyString.CreateWithoutCheck(oc)
		ceEmptyString.CheckClusterExtensionCondition(oc, "Progressing", "message", "unknown field", 3, 60, 0)
		e2e.Logf("=== ClusterExtension EmptyString resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEmptyString.Name, "-o", "yaml").Execute()
		ceEmptyString.Delete(oc)
		e2e.Logf("Scenario 1-2 cleanup: ClusterExtension deleted")

		g.By("Scenario 1-3: watchNamespace equals install namespace")
		ceEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-equalinstall",
			PackageName:      "nginx-ok-v85510",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   ns,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEqualsInstall.Delete(oc)
		_ = ceEqualsInstall.CreateWithoutCheck(oc)
		ceEqualsInstall.CheckClusterExtensionCondition(oc, "Progressing", "message", "unknown field", 3, 60, 0)
		e2e.Logf("=== ClusterExtension EqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEqualsInstall.Name, "-o", "yaml").Execute()
		ceEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 1-3 cleanup: ClusterExtension deleted")

		g.By("Scenario 1-4: watchNamespace not equals install namespace")
		ceNotEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-notequalinstall",
			PackageName:      "nginx-ok-v85510",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   nsTarget,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceNotEqualsInstall.Delete(oc)
		_ = ceNotEqualsInstall.CreateWithoutCheck(oc)
		ceNotEqualsInstall.CheckClusterExtensionCondition(oc, "Progressing", "message", "unknown field", 3, 60, 0)
		e2e.Logf("=== ClusterExtension NotEqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNotEqualsInstall.Name, "-o", "yaml").Execute()
		ceNotEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 1-4 cleanup: ClusterExtension deleted")

		e2e.Logf("PASS: All scenarios for PolarionID:85510 completed successfully")
	})

	g.It("PolarionID:85543-watchNamespace configuration with AllNamespaces+OwnNamespace+SingleNamespace InstallModes", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                         = "85543"
			ns                             = "ns-" + caseID
			nsTarget                       = "ns-" + caseID + "-target"
			sa                             = "sa" + caseID
			labelValue                     = caseID
			catalogName                    = "clustercatalog-" + caseID
			baseDir                        = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate         = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate       = filepath.Join(baseDir, "clusterextension.yaml")
			clusterextensionConfigTemplate = filepath.Join(baseDir, "clusterextension-watchns-config.yaml")
			saClusterRoleBindingTemplate   = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                          = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv85543",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
		)

		g.By("Create install namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create target namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsTarget, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsTarget).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsTarget)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		// Print full RBAC resources for manual test documentation
		e2e.Logf("=== ServiceAccount resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ServiceAccount", sa, "-n", ns, "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRole resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRole", sa+"-installer-admin-clusterrole", "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRoleBinding resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRoleBinding", sa+"-installer-admin-clusterrole-binding", "-o", "yaml").Execute()

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		// Print full ClusterCatalog resource for manual test documentation
		e2e.Logf("=== ClusterCatalog resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clustercatalog", catalogName, "-o", "yaml").Execute()

		g.By("Scenario 2-1: No config - expect AllNamespaces mode success")
		e2e.Logf("Testing ClusterExtension with {All,Own,Single} InstallModes and no watchNamespace config")
		ceNoConfig := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-noconfig",
			PackageName:      "nginx-ok-v85543",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			Template:         clusterextensionTemplate,
		}
		defer ceNoConfig.Delete(oc)
		ceNoConfig.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NoConfig resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNoConfig.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 2-1 passed: ClusterExtension installed successfully in AllNamespaces mode without config")
		ceNoConfig.Delete(oc)
		e2e.Logf("Scenario 2-1 cleanup: ClusterExtension deleted")

		g.By("Scenario 2-2: Empty string watchNamespace - expect AllNamespaces mode success")
		e2e.Logf("Testing ClusterExtension with {All,Own,Single} InstallModes and empty watchNamespace")
		e2e.Logf("Expected behavior: Empty watchNamespace defaults to AllNamespaces mode")
		ceEmptyString := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-empty",
			PackageName:      "nginx-ok-v85543",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   "",
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEmptyString.Delete(oc)
		// For {All,Own,Single} InstallModes, empty string defaults to AllNamespaces mode
		ceEmptyString.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EmptyString resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEmptyString.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 2-2 passed: ClusterExtension with empty watchNamespace installed successfully in AllNamespaces mode")
		ceEmptyString.Delete(oc)
		e2e.Logf("Scenario 2-2 cleanup: ClusterExtension deleted")

		g.By("Scenario 2-3: watchNamespace equals install namespace - expect OwnNamespace mode success")
		e2e.Logf("Testing ClusterExtension with {All,Own,Single} InstallModes and watchNamespace=install namespace")
		ceEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-equalinstall",
			PackageName:      "nginx-ok-v85543",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   ns,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEqualsInstall.Delete(oc)
		ceEqualsInstall.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 2-3 passed: ClusterExtension installed successfully in OwnNamespace mode")
		ceEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 2-3 cleanup: ClusterExtension deleted")

		g.By("Scenario 2-4: watchNamespace not equals install namespace - expect SingleNamespace mode success")
		e2e.Logf("Testing ClusterExtension with {All,Own,Single} InstallModes and watchNamespace!=install namespace")
		ceNotEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-notequalinstall",
			PackageName:      "nginx-ok-v85543",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   nsTarget,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceNotEqualsInstall.Delete(oc)
		ceNotEqualsInstall.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NotEqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNotEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 2-4 passed: ClusterExtension installed successfully in SingleNamespace mode")
		ceNotEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 2-4 cleanup: ClusterExtension deleted")

		e2e.Logf("PASS: All scenarios for PolarionID:85543 completed successfully")
	})

	g.It("PolarionID:85546-watchNamespace configuration with OwnNamespace+SingleNamespace InstallModes", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                         = "85546"
			ns                             = "ns-" + caseID
			nsTarget                       = "ns-" + caseID + "-target"
			sa                             = "sa" + caseID
			labelValue                     = caseID
			catalogName                    = "clustercatalog-" + caseID
			baseDir                        = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate         = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate       = filepath.Join(baseDir, "clusterextension.yaml")
			clusterextensionConfigTemplate = filepath.Join(baseDir, "clusterextension-watchns-config.yaml")
			saClusterRoleBindingTemplate   = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                          = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv85546",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
		)

		g.By("Create install namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create target namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsTarget, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsTarget).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsTarget)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		// Print full RBAC resources for manual test documentation
		e2e.Logf("=== ServiceAccount resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ServiceAccount", sa, "-n", ns, "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRole resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRole", sa+"-installer-admin-clusterrole", "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRoleBinding resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRoleBinding", sa+"-installer-admin-clusterrole-binding", "-o", "yaml").Execute()

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		// Print full ClusterCatalog resource for manual test documentation
		e2e.Logf("=== ClusterCatalog resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clustercatalog", catalogName, "-o", "yaml").Execute()

		g.By("Scenario 3-1: No config - expect failure with required field error")
		e2e.Logf("Testing ClusterExtension with {Own,Single} InstallModes and no watchNamespace config")
		e2e.Logf("Bundle InstallModes verified: OwnNamespace=true, SingleNamespace=true, AllNamespaces=false")
		e2e.Logf("Expected behavior: Should fail with 'required field watchNamespace is missing' error")
		ceNoConfig := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-noconfig",
			PackageName:      "nginx-ok-v85546",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			Template:         clusterextensionTemplate,
		}
		defer ceNoConfig.Delete(oc)
		_ = ceNoConfig.CreateWithoutCheck(oc)
		// For {Own,Single} InstallModes, watchNamespace is REQUIRED
		// Should fail with error indicating required field is missing
		ceNoConfig.CheckClusterExtensionCondition(oc, "Progressing", "message", "required", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NoConfig resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNoConfig.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 3-1 passed: ClusterExtension correctly rejected - watchNamespace is required for {Own,Single} bundles")
		ceNoConfig.Delete(oc)
		e2e.Logf("Scenario 3-1 cleanup: ClusterExtension deleted")

		g.By("Scenario 3-2: Empty string watchNamespace - expect validation error")
		e2e.Logf("Testing ClusterExtension with {Own,Single} InstallModes and empty watchNamespace")
		ceEmptyString := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-empty",
			PackageName:      "nginx-ok-v85546",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   "",
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEmptyString.Delete(oc)
		_ = ceEmptyString.CreateWithoutCheck(oc)
		// Empty string is a supported field but invalid value (DNS format validation)
		ceEmptyString.CheckClusterExtensionCondition(oc, "Progressing", "message", "invalid", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EmptyString resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEmptyString.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 3-2 passed: ClusterExtension with empty watchNamespace correctly rejected with validation error")
		ceEmptyString.Delete(oc)
		e2e.Logf("Scenario 3-2 cleanup: ClusterExtension deleted")

		g.By("Scenario 3-3: watchNamespace equals install namespace - expect OwnNamespace mode success")
		e2e.Logf("Testing ClusterExtension with {Own,Single} InstallModes and watchNamespace=install namespace")
		ceEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-equalinstall",
			PackageName:      "nginx-ok-v85546",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   ns,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEqualsInstall.Delete(oc)
		ceEqualsInstall.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 3-3 passed: ClusterExtension installed successfully in OwnNamespace mode")
		ceEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 3-3 cleanup: ClusterExtension deleted")

		g.By("Scenario 3-4: watchNamespace not equals install namespace - expect SingleNamespace mode success")
		e2e.Logf("Testing ClusterExtension with {Own,Single} InstallModes and watchNamespace!=install namespace")
		ceNotEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-notequalinstall",
			PackageName:      "nginx-ok-v85546",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   nsTarget,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceNotEqualsInstall.Delete(oc)
		ceNotEqualsInstall.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NotEqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNotEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 3-4 passed: ClusterExtension installed successfully in SingleNamespace mode")
		ceNotEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 3-4 cleanup: ClusterExtension deleted")

		e2e.Logf("PASS: All scenarios for PolarionID:85546 completed successfully")
	})

	g.It("PolarionID:85547-watchNamespace configuration with SingleNamespace InstallMode", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                         = "85547"
			ns                             = "ns-" + caseID
			nsTarget                       = "ns-" + caseID + "-target"
			sa                             = "sa" + caseID
			labelValue                     = caseID
			catalogName                    = "clustercatalog-" + caseID
			baseDir                        = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate         = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate       = filepath.Join(baseDir, "clusterextension.yaml")
			clusterextensionConfigTemplate = filepath.Join(baseDir, "clusterextension-watchns-config.yaml")
			saClusterRoleBindingTemplate   = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                          = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv85547",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
		)

		g.By("Creating namespaces for install and target")

		g.By("Create install namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create target namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsTarget, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsTarget).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsTarget)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		// Print full RBAC resources for manual test documentation
		e2e.Logf("=== ServiceAccount resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ServiceAccount", sa, "-n", ns, "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRole resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRole", sa+"-installer-admin-clusterrole", "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRoleBinding resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRoleBinding", sa+"-installer-admin-clusterrole-binding", "-o", "yaml").Execute()

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		// Print full ClusterCatalog resource for manual test documentation
		e2e.Logf("=== ClusterCatalog resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clustercatalog", catalogName, "-o", "yaml").Execute()

		e2e.Logf("Testing Group 5: {SingleNamespace} InstallMode only")
		e2e.Logf("Expected behavior: watchNamespace is REQUIRED (no AllNamespaces support)")

		g.By("Scenario 5-1: No config - expect required field error")
		e2e.Logf("Testing ClusterExtension with {Single} InstallMode and no config")
		e2e.Logf("Bundle InstallModes: SingleNamespace=true only (AllNamespaces=false, OwnNamespace=false)")
		e2e.Logf("Expected behavior: Should fail with 'exactly one target namespace must be specified' error")
		ceNoConfig := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-noconfig",
			PackageName:      "nginx-ok-v85547",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			Template:         clusterextensionTemplate,
		}
		defer ceNoConfig.Delete(oc)
		_ = ceNoConfig.CreateWithoutCheck(oc)
		ceNoConfig.CheckClusterExtensionCondition(oc, "Progressing", "message", "required field", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NoConfig resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNoConfig.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 5-1 passed: Got expected required field error")
		ceNoConfig.Delete(oc)
		e2e.Logf("Scenario 5-1 cleanup: ClusterExtension deleted")

		g.By("Scenario 5-2: watchNamespace is empty string - expect DNS validation error")
		e2e.Logf("Testing ClusterExtension with {Single} InstallMode and watchNamespace=''")
		ceEmptyString := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-empty",
			PackageName:      "nginx-ok-v85547",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   "",
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEmptyString.Delete(oc)
		_ = ceEmptyString.CreateWithoutCheck(oc)
		ceEmptyString.CheckClusterExtensionCondition(oc, "Progressing", "message", "invalid", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EmptyString resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEmptyString.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 5-2 passed: Got expected DNS validation error for empty string")
		ceEmptyString.Delete(oc)
		e2e.Logf("Scenario 5-2 cleanup: ClusterExtension deleted")

		g.By("Scenario 5-3: watchNamespace equals install namespace - expect failure (OwnNamespace mode not supported)")
		e2e.Logf("Testing ClusterExtension with {Single} InstallMode and watchNamespace=install namespace")
		e2e.Logf("Bundle only supports SingleNamespace, not OwnNamespace mode")
		e2e.Logf("Expected: Should fail because watchNamespace==installNamespace requires OwnNamespace support")
		ceEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-equalinstall",
			PackageName:      "nginx-ok-v85547",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   ns,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEqualsInstall.Delete(oc)
		_ = ceEqualsInstall.CreateWithoutCheck(oc)
		ceEqualsInstall.CheckClusterExtensionCondition(oc, "Progressing", "message", "is not valid singleNamespaceInstallMode", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 5-3 passed: Got expected OwnNamespace mode not supported error")
		ceEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 5-3 cleanup: ClusterExtension deleted")

		g.By("Scenario 5-4: watchNamespace not equals install namespace - expect SingleNamespace mode success")
		e2e.Logf("Testing ClusterExtension with {Single} InstallMode and watchNamespace!=install namespace")
		ceNotEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-notequalinstall",
			PackageName:      "nginx-ok-v85547",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   nsTarget,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceNotEqualsInstall.Delete(oc)
		ceNotEqualsInstall.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NotEqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNotEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 5-4 passed: ClusterExtension installed successfully in SingleNamespace mode (watching target-ns)")
		ceNotEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 5-4 cleanup: ClusterExtension deleted")

		e2e.Logf("PASS: All scenarios for PolarionID:85547 completed successfully")
	})

	g.It("PolarionID:85549-watchNamespace configuration with OwnNamespace InstallMode", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                         = "85549"
			ns                             = "ns-" + caseID
			nsTarget                       = "ns-" + caseID + "-target"
			sa                             = "sa" + caseID
			labelValue                     = caseID
			catalogName                    = "clustercatalog-" + caseID
			baseDir                        = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate         = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate       = filepath.Join(baseDir, "clusterextension.yaml")
			clusterextensionConfigTemplate = filepath.Join(baseDir, "clusterextension-watchns-config.yaml")
			saClusterRoleBindingTemplate   = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                          = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv85549",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
		)

		e2e.Logf("Starting Group 4 tests: {OwnNamespace} only InstallMode with special constraint")

		g.By("Create install namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create target namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsTarget, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsTarget).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsTarget)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		// Print full RBAC resources for manual test documentation
		e2e.Logf("=== ServiceAccount resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ServiceAccount", sa, "-n", ns, "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRole resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRole", sa+"-installer-admin-clusterrole", "-o", "yaml").Execute()
		e2e.Logf("=== ClusterRoleBinding resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("ClusterRoleBinding", sa+"-installer-admin-clusterrole-binding", "-o", "yaml").Execute()

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		// Print full ClusterCatalog resource for manual test documentation
		e2e.Logf("=== ClusterCatalog resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clustercatalog", catalogName, "-o", "yaml").Execute()

		e2e.Logf("Testing Group 4: {OwnNamespace} InstallMode only")
		e2e.Logf("Special constraint: watchNamespace MUST equal install namespace")

		g.By("Scenario 4-1: No config - expect failure with required field error")
		e2e.Logf("Testing ClusterExtension with {Own} InstallMode only and no watchNamespace config")
		e2e.Logf("Bundle InstallModes: OwnNamespace=true only (AllNamespaces=false, SingleNamespace=false)")
		e2e.Logf("Expected behavior: Should fail with 'required field watchNamespace is missing' error")
		ceNoConfig := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-noconfig",
			PackageName:      "nginx-ok-v85549",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			Template:         clusterextensionTemplate,
		}
		defer ceNoConfig.Delete(oc)
		_ = ceNoConfig.CreateWithoutCheck(oc)
		// For {OwnNamespace} only InstallMode, watchNamespace is REQUIRED
		// Should fail with error indicating required field is missing
		ceNoConfig.CheckClusterExtensionCondition(oc, "Progressing", "message", "required field", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NoConfig resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNoConfig.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 4-1 passed: ClusterExtension correctly rejected - watchNamespace is required for {Own} bundles")
		ceNoConfig.Delete(oc)
		e2e.Logf("Scenario 4-1 cleanup: ClusterExtension deleted")

		g.By("Scenario 4-2: watchNamespace is empty string - expect DNS validation error")
		e2e.Logf("Testing ClusterExtension with {Own} InstallMode and watchNamespace=''")
		e2e.Logf("Expected behavior: watchNamespace field should be supported with DNS validation")
		ceEmptyString := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-empty",
			PackageName:      "nginx-ok-v85549",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   "",
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEmptyString.Delete(oc)
		_ = ceEmptyString.CreateWithoutCheck(oc)
		// Empty string should fail with DNS validation error
		ceEmptyString.CheckClusterExtensionCondition(oc, "Progressing", "message", "invalid", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EmptyString resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEmptyString.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 4-2 passed: ClusterExtension with empty watchNamespace correctly rejected with validation error")
		ceEmptyString.Delete(oc)
		e2e.Logf("Scenario 4-2 cleanup: ClusterExtension deleted")

		g.By("Scenario 4-3: watchNamespace equals install namespace - expect OwnNamespace mode success")
		e2e.Logf("Testing ClusterExtension with {Own} InstallMode and watchNamespace=install namespace")
		e2e.Logf("Expected behavior: Should succeed in OwnNamespace mode")
		ceEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-equalinstall",
			PackageName:      "nginx-ok-v85549",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   ns,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceEqualsInstall.Delete(oc)
		// When watchNamespace equals install namespace, should succeed in OwnNamespace mode
		ceEqualsInstall.Create(oc)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension EqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 4-3 passed: ClusterExtension installed successfully in OwnNamespace mode")
		ceEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 4-3 cleanup: ClusterExtension deleted")

		g.By("Scenario 4-4: watchNamespace not equals install namespace - expect failure")
		e2e.Logf("Testing ClusterExtension with {Own} InstallMode and watchNamespace≠install namespace")
		e2e.Logf("Expected behavior: Should fail with namespace mismatch error")
		e2e.Logf("Bundle only supports OwnNamespace mode, so watchNamespace must equal install namespace")
		ceNotEqualsInstall := olmv1util.ClusterExtensionDescription{
			Name:             "ce-" + caseID + "-notequalinstall",
			PackageName:      "nginx-ok-v85549",
			Channel:          "alpha",
			Version:          ">=0.0.1",
			InstallNamespace: ns,
			SaName:           sa,
			LabelValue:       labelValue,
			WatchNamespace:   nsTarget,
			Template:         clusterextensionConfigTemplate,
		}
		defer ceNotEqualsInstall.Delete(oc)
		_ = ceNotEqualsInstall.CreateWithoutCheck(oc)
		// For {OwnNamespace} only mode, watchNamespace must equal install namespace
		// Should fail with error indicating namespace mismatch
		ceNotEqualsInstall.CheckClusterExtensionCondition(oc, "Progressing", "message", "is not valid ownNamespaceInstallMode", 3, 60, 0)
		// Print full ClusterExtension resource for manual test documentation
		e2e.Logf("=== ClusterExtension NotEqualsInstall resource ===")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterextension", ceNotEqualsInstall.Name, "-o", "yaml").Execute()
		e2e.Logf("PASS: Scenario 4-4 passed: ClusterExtension correctly rejected - watchNamespace must equal install namespace for {Own} bundles")
		ceNotEqualsInstall.Delete(oc)
		e2e.Logf("Scenario 4-4 cleanup: ClusterExtension deleted")

		e2e.Logf("PASS: All scenarios for PolarionID:85549 completed successfully")
	})

	g.It("PolarionID:85650-[Skipped:Disconnected]API-level error validation for watchNamespace configuration", func() {
		var (
			caseID                       = "85650"
			ns                           = "ns-" + caseID
			nsTarget                     = "ns-" + caseID + "-target"
			sa                           = "sa" + caseID
			labelValue                   = caseID
			catalogName                  = "clustercatalog-" + caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")
			saCrb                        = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv85650",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
		)

		g.By("Create install namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())

		g.By("Create target namespace")
		defer func() {
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", nsTarget, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", nsTarget).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", nsTarget)).To(o.BeTrue())

		g.By("Create SA for clusterextension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)

		g.By("Create clustercatalog")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)

		g.By("Scenario 6-1: Empty config object (missing configType)")
		e2e.Logf("Testing ClusterExtension with spec.config={} (empty object)")
		e2e.Logf("Expected behavior: Should fail with API validation error")
		ceName1 := "ce-" + caseID + "-emptyconfig"
		ceYaml1 := `apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: ` + ceName1 + `
spec:
  packageName: nginx-ok-v85650
  channel: alpha
  version: ">=0.0.1"
  installNamespace: ` + ns + `
  serviceAccount:
    name: ` + sa + `
  selector:
    matchLabels:
      test: ` + labelValue + `
  config: {}`
		result1 := oc.AsAdmin().Run("apply").Args("-f", "-").InputString(ceYaml1).Execute()
		o.Expect(result1).Should(o.HaveOccurred(), "Should fail with 'configType is required' error")
		e2e.Logf("PASS: Scenario 6-1 passed: ClusterExtension creation failed as expected (configType validation)")
		_ = oc.AsAdmin().Run("delete").Args("clusterextension", ceName1, "--ignore-not-found=true").Execute()
		e2e.Logf("Scenario 6-1 cleanup: Attempted cleanup")

		g.By("Scenario 6-2: Invalid configType value")
		e2e.Logf("Testing ClusterExtension with configType='Invalid'")
		e2e.Logf("Expected behavior: Should fail with API validation error")
		ceName2 := "ce-" + caseID + "-invalidtype"
		ceYaml2 := `apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: ` + ceName2 + `
spec:
  packageName: nginx-ok-v85650
  channel: alpha
  version: ">=0.0.1"
  installNamespace: ` + ns + `
  serviceAccount:
    name: ` + sa + `
  selector:
    matchLabels:
      test: ` + labelValue + `
  config:
    configType: Invalid`
		result2 := oc.AsAdmin().Run("apply").Args("-f", "-").InputString(ceYaml2).Execute()
		o.Expect(result2).Should(o.HaveOccurred(), "Should fail with invalid configType error")
		e2e.Logf("PASS: Scenario 6-2 passed: ClusterExtension creation failed as expected (invalid configType)")
		_ = oc.AsAdmin().Run("delete").Args("clusterextension", ceName2, "--ignore-not-found=true").Execute()
		e2e.Logf("Scenario 6-2 cleanup: Attempted cleanup")

		g.By("Scenario 6-3: configType='Inline' without inline field")
		e2e.Logf("Testing ClusterExtension with configType='Inline' but no inline field")
		e2e.Logf("Expected behavior: Should fail with API validation error")
		ceName3 := "ce-" + caseID + "-noinline"
		ceYaml3 := `apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: ` + ceName3 + `
spec:
  packageName: nginx-ok-v85650
  channel: alpha
  version: ">=0.0.1"
  installNamespace: ` + ns + `
  serviceAccount:
    name: ` + sa + `
  selector:
    matchLabels:
      test: ` + labelValue + `
  config:
    configType: Inline`
		result3 := oc.AsAdmin().Run("apply").Args("-f", "-").InputString(ceYaml3).Execute()
		o.Expect(result3).Should(o.HaveOccurred(), "Should fail with inline required error")
		e2e.Logf("PASS: Scenario 6-3 passed: ClusterExtension creation failed as expected (inline required)")
		_ = oc.AsAdmin().Run("delete").Args("clusterextension", ceName3, "--ignore-not-found=true").Execute()
		e2e.Logf("Scenario 6-3 cleanup: Attempted cleanup")

		g.By("Scenario 6-4: Invalid JSON in inline field")
		e2e.Logf("Testing ClusterExtension with malformed JSON in spec.config.inline")
		e2e.Logf("Expected behavior: Should fail with API validation error")
		ceName4 := "ce-" + caseID + "-invalidjson"
		ceYaml4 := `apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: ` + ceName4 + `
spec:
  packageName: nginx-ok-v85650
  channel: alpha
  version: ">=0.0.1"
  installNamespace: ` + ns + `
  serviceAccount:
    name: ` + sa + `
  selector:
    matchLabels:
      test: ` + labelValue + `
  config:
    configType: Inline
    inline:
      this is not valid json`
		result4 := oc.AsAdmin().Run("apply").Args("-f", "-").InputString(ceYaml4).Execute()
		o.Expect(result4).Should(o.HaveOccurred(), "Should fail with JSON validation error")
		e2e.Logf("PASS: Scenario 6-4 passed: ClusterExtension creation failed as expected (JSON validation)")
		_ = oc.AsAdmin().Run("delete").Args("clusterextension", ceName4, "--ignore-not-found=true").Execute()
		e2e.Logf("Scenario 6-4 cleanup: Attempted cleanup")

		e2e.Logf("PASS: All scenarios for PolarionID:85650 completed successfully")
	})
})
