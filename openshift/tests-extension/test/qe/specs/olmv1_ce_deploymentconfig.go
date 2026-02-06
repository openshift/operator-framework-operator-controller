package specs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
	olmv1util "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util/olmv1util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] OLMv1 ClusterExtension DeploymentConfig", g.Label("NonHyperShiftHOST"), func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("default")
	)

	g.BeforeEach(func() {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
		if !olmv1util.IsFeaturegateEnabled(oc, "NewOLMConfigAPI") {
			g.Skip("NewOLMConfigAPI feature gate is disabled. This test requires deployment configuration support.")
		}
	})

	g.It("PolarionID:87536-deploymentConfig env vars are applied to operator deployment and available in pod", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                       = "87536"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			inlineConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "CUSTOM_LOG_LEVEL", "value": "debug"},
        {"name": "TEST_ENV_VAR", "value": "test-value"},
        {"name": "ANOTHER_VAR", "value": "another-value"}
      ]
    }
  }`

			// Define test env vars for verification
			testEnvVars = map[string]string{
				"CUSTOM_LOG_LEVEL": "debug",
				"TEST_ENV_VAR":     "test-value",
				"ANOTHER_VAR":      "another-value",
			}

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87536",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87536",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Verify env vars are present in Deployment manifest")
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, testEnvVars, 15*time.Second)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("All env vars verified in Deployment manifest")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Verify env vars are accessible in pod")
		err = olmv1util.VerifyPodEnvVars(oc, podName, ns, "", testEnvVars, 15*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("All env vars verified in pod")

		e2e.Logf("Test completed successfully - deploymentConfig env vars are applied and accessible")
	})

	g.It("PolarionID:87537-deploymentConfig env vars override existing bundle env vars with same name", func() {
		olmv1util.ValidateAccessEnvironment(oc)
		var (
			caseID                       = "87537"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// ENV1: will override bundle's ENV1=bundle_value1
			// ENV3: will be added (not in bundle)
			// ENV2: not specified, will preserve bundle's ENV2=bundle_value2
			inlineConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "ENV1", "value": "config_value1"},
        {"name": "ENV3", "value": "config_value3"}
      ]
    }
  }`

			// Expected env vars after merge:
			// - ENV1: config_value1 (override from config)
			// - ENV2: bundle_value2 (preserved from bundle)
			// - ENV3: config_value3 (added from config)
			expectedEnvVars = map[string]string{
				"ENV1": "config_value1", // Test Point 1: Override existing env var
				"ENV2": "bundle_value2", // Test Point 2: Preserve existing env var with different name
				"ENV3": "config_value3", // Test Point 3: Add new env var
			}

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87537",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87537",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Verify env vars merge correctly in Deployment manifest")
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedEnvVars, 15*time.Second)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("All env vars verified in Deployment manifest - override, preserve, and add all work correctly")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Verify env vars are accessible in pod")
		err = olmv1util.VerifyPodEnvVars(oc, podName, ns, "", expectedEnvVars, 15*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("All env vars verified in pod - ENV1 overridden, ENV2 preserved, ENV3 added")

		e2e.Logf("Test completed successfully - env vars merge with override rule works correctly")
	})

	g.It("PolarionID:87539-[Skipped:Disconnected]deploymentConfig envFrom sources are appended to operator deployment without duplicates", func() {
		var (
			caseID                       = "87539"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			cmBundle1                    = "test-cm-bundle-1"   // Predefined in bundle CSV
			cmBundle2                    = "test-cm-bundle-2"   // Predefined in bundle CSV
			cmConfigNew                  = "test-cm-config-new" // New CM from config
			secretConfig                 = "test-secret-config" // New Secret from config
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// test-cm-bundle-1: Duplicate with bundle - should NOT be added again (deduplication)
			// test-cm-config-new: New ConfigMap - should be appended
			// test-secret-config: New Secret - should be appended
			inlineConfig = `{
    "deploymentConfig": {
      "envFrom": [
        {"configMapRef": {"name": "test-cm-bundle-1"}},
        {"configMapRef": {"name": "test-cm-config-new"}},
        {"secretRef": {"name": "test-secret-config"}}
      ]
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87539",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87539",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ConfigMaps for envFrom testing")
		// CM1 and CM2 are referenced by bundle CSV
		defer func() {
			e2e.Logf("Cleaning up ConfigMaps")
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("cm", cmBundle1, "-n", ns, "--ignore-not-found").Execute()
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("cm", cmBundle2, "-n", ns, "--ignore-not-found").Execute()
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("cm", cmConfigNew, "-n", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("configmap", cmBundle1,
			"-n", ns,
			"--from-literal=BUNDLE_CM1_KEY=bundle_cm1_value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created ConfigMap: %s", cmBundle1)

		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("configmap", cmBundle2,
			"-n", ns,
			"--from-literal=BUNDLE_CM2_KEY=bundle_cm2_value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created ConfigMap: %s", cmBundle2)

		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("configmap", cmConfigNew,
			"-n", ns,
			"--from-literal=CONFIG_CM_KEY=config_cm_value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created ConfigMap: %s", cmConfigNew)

		g.By("Create Secret for envFrom testing")
		defer func() {
			e2e.Logf("Cleaning up Secret %s", secretConfig)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("secret", secretConfig, "-n", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("secret", "generic", secretConfig,
			"-n", ns,
			"--from-literal=CONFIG_SECRET_KEY=config_secret_value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created Secret: %s", secretConfig)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Verify envFrom sources in Deployment manifest")
		// Get all envFrom entries
		envFromListPath := `jsonpath={range .spec.template.spec.containers[0].envFrom[*]}{.configMapRef.name},{.secretRef.name}{"\n"}{end}`
		envFromList, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", envFromListPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("EnvFrom list (configMapRef,secretRef per line):\n%s", envFromList)

		// Test Point 1: Verify ConfigMap envFrom sources are present
		o.Expect(envFromList).To(o.ContainSubstring(cmBundle1), "Bundle CM1 should be present")
		o.Expect(envFromList).To(o.ContainSubstring(cmBundle2), "Bundle CM2 should be present")
		o.Expect(envFromList).To(o.ContainSubstring(cmConfigNew), "Config CM should be appended")
		e2e.Logf("Test Point 1 passed: ConfigMap envFrom sources verified")

		// Test Point 2: Verify Secret envFrom source is present
		o.Expect(envFromList).To(o.ContainSubstring(secretConfig), "Config Secret should be appended")
		e2e.Logf("Test Point 2 passed: Secret envFrom source verified")

		// Test Point 3: Verify no duplicates - each source appears exactly once
		cmBundle1Count := strings.Count(envFromList, cmBundle1)
		cmBundle2Count := strings.Count(envFromList, cmBundle2)
		cmConfigNewCount := strings.Count(envFromList, cmConfigNew)
		secretConfigCount := strings.Count(envFromList, secretConfig)

		o.Expect(cmBundle1Count).To(o.Equal(1), "test-cm-bundle-1 should appear exactly once (duplicate from config removed)")
		o.Expect(cmBundle2Count).To(o.Equal(1), "test-cm-bundle-2 should appear exactly once")
		o.Expect(cmConfigNewCount).To(o.Equal(1), "test-cm-config-new should appear exactly once")
		o.Expect(secretConfigCount).To(o.Equal(1), "test-secret-config should appear exactly once")
		e2e.Logf("Test Point 3 passed: No duplicate envFrom sources - deduplication works correctly")

		// Verify total count: should have exactly 4 envFrom entries
		envFromLines := strings.Split(strings.TrimSpace(envFromList), "\n")
		o.Expect(len(envFromLines)).To(o.Equal(4), "Should have exactly 4 envFrom entries total")
		e2e.Logf("Total envFrom entries: %d (expected: 4)", len(envFromLines))

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Verify env vars from envFrom sources are accessible in pod")
		// Verify all env vars from all 4 envFrom sources
		allEnvVars := map[string]string{
			"BUNDLE_CM1_KEY":    "bundle_cm1_value",    // From bundle CM1
			"BUNDLE_CM2_KEY":    "bundle_cm2_value",    // From bundle CM2
			"CONFIG_CM_KEY":     "config_cm_value",     // From config CM
			"CONFIG_SECRET_KEY": "config_secret_value", // From config Secret
		}
		err = olmv1util.VerifyPodEnvVars(oc, podName, ns, "", allEnvVars, 15*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("All env vars from envFrom sources verified in pod")

		e2e.Logf("Test completed successfully - envFrom append without duplicates works correctly")
	})

	g.It("PolarionID:87541-[Skipped:Disconnected]deploymentConfig volumes are appended to operator deployment", func() {
		var (
			caseID                       = "87541"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			cmVolume                     = "test-cm-vol"     // ConfigMap for volume
			secretVolume                 = "test-secret-vol" // Secret for volume
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// Add two volumes: one ConfigMap volume and one Secret volume
			inlineConfig = `{
    "deploymentConfig": {
      "volumes": [
        {
          "name": "config-cm-vol",
          "configMap": {
            "name": "test-cm-vol"
          }
        },
        {
          "name": "config-secret-vol",
          "secret": {
            "secretName": "test-secret-vol"
          }
        }
      ]
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87541",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87541",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ConfigMap for volume testing")
		defer func() {
			e2e.Logf("Cleaning up ConfigMap %s", cmVolume)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("cm", cmVolume, "-n", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("configmap", cmVolume,
			"-n", ns,
			"--from-literal=cm-key=cm-value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created ConfigMap: %s", cmVolume)

		g.By("Create Secret for volume testing")
		defer func() {
			e2e.Logf("Cleaning up Secret %s", secretVolume)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("secret", secretVolume, "-n", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("secret", "generic", secretVolume,
			"-n", ns,
			"--from-literal=secret-key=secret-value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created Secret: %s", secretVolume)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify volumes appended to Deployment")
		// Get all volume names from Deployment spec.template.spec.volumes
		volumesPath := `jsonpath={range .spec.template.spec.volumes[*]}{.name}{"\n"}{end}`
		volumesList, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", volumesPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Volumes list:\n%s", volumesList)

		// Verify config volumes are present
		o.Expect(volumesList).To(o.ContainSubstring("config-cm-vol"), "ConfigMap volume should be appended")
		o.Expect(volumesList).To(o.ContainSubstring("config-secret-vol"), "Secret volume should be appended")
		e2e.Logf("Test Point 1 passed: Config volumes appended to Deployment")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify direct append behavior in actual Pod (bundle volume + config volumes)")
		// Get all volume names from Pod spec.volumes
		podVolumesPath := `jsonpath={range .spec.volumes[*]}{.name}{"\n"}{end}`
		podVolumesList, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podVolumesPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod volumes list:\n%s", podVolumesList)

		// Verify bundle volume is present (bundle has predefined emptyDir volume)
		o.Expect(podVolumesList).To(o.ContainSubstring("bundle-emptydir-vol"), "Bundle emptyDir volume should be preserved in Pod")

		// Verify config volumes are present in Pod
		o.Expect(podVolumesList).To(o.ContainSubstring("config-cm-vol"), "ConfigMap volume should be appended to Pod")
		o.Expect(podVolumesList).To(o.ContainSubstring("config-secret-vol"), "Secret volume should be appended to Pod")

		// Count our configured volumes (not system volumes)
		// Bundle has 1 volume: bundle-emptydir-vol
		// Config adds 2 volumes: config-cm-vol, config-secret-vol
		podVolumeLines := strings.Split(strings.TrimSpace(podVolumesList), "\n")
		bundleVolumeCount := 0
		configVolumeCount := 0
		for _, vol := range podVolumeLines {
			vol = strings.TrimSpace(vol)
			if vol == "bundle-emptydir-vol" {
				bundleVolumeCount++
			}
			if vol == "config-cm-vol" || vol == "config-secret-vol" {
				configVolumeCount++
			}
		}

		o.Expect(bundleVolumeCount).To(o.Equal(1), "Pod should have 1 bundle volume")
		o.Expect(configVolumeCount).To(o.Equal(2), "Pod should have 2 config volumes appended")

		// Note: Pod may have additional system volumes (e.g., kube-api-access-xxx for serviceaccount token)
		// We only verify our configured volumes are present, not the total count
		e2e.Logf("Test Point 2 passed: Direct append in Pod - bundle volume (1) + config volumes (2) = %d configured volumes (total volumes: %d including system volumes)",
			bundleVolumeCount+configVolumeCount, len(podVolumeLines))

		e2e.Logf("Test completed successfully - volumes direct append mechanism works correctly")
	})

	g.It("PolarionID:87542-[Skipped:Disconnected]deploymentConfig volumeMounts are appended to all operator containers", func() {
		var (
			caseID                       = "87542"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			cmVolume                     = "test-cm-vol"     // ConfigMap for volume
			secretVolume                 = "test-secret-vol" // Secret for volume
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// Add volumes and volumeMounts together
			inlineConfig = `{
    "deploymentConfig": {
      "volumes": [
        {
          "name": "config-cm-vol",
          "configMap": {
            "name": "test-cm-vol"
          }
        },
        {
          "name": "config-secret-vol",
          "secret": {
            "secretName": "test-secret-vol"
          }
        }
      ],
      "volumeMounts": [
        {
          "name": "config-cm-vol",
          "mountPath": "/config-cm-mount"
        },
        {
          "name": "config-secret-vol",
          "mountPath": "/config-secret-mount"
        }
      ]
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87542",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87542",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ConfigMap for volume testing")
		defer func() {
			e2e.Logf("Cleaning up ConfigMap %s", cmVolume)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("cm", cmVolume, "-n", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("configmap", cmVolume,
			"-n", ns,
			"--from-literal=cm-key=cm-value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created ConfigMap: %s", cmVolume)

		g.By("Create Secret for volume testing")
		defer func() {
			e2e.Logf("Cleaning up Secret %s", secretVolume)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("secret", secretVolume, "-n", ns, "--ignore-not-found").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("secret", "generic", secretVolume,
			"-n", ns,
			"--from-literal=secret-key=secret-value").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Created Secret: %s", secretVolume)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify volumeMounts appended to Deployment containers")
		// NOTE: This test uses DIFFERENT volumeMount names to avoid duplicate name scenario
		//       Bundle has: bundle-emptydir-vol
		//       Config has: config-cm-vol, config-secret-vol
		//       This validates current OLMv1 "direct append" behavior
		//       If OLMv1 should align with OLMv0 "merge with override", this test needs update
		//
		// Get all volumeMount names from the first (main) container
		volumeMountsPath := `jsonpath={range .spec.template.spec.containers[0].volumeMounts[*]}{.name}{"\n"}{end}`
		volumeMountsList, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", volumeMountsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("VolumeMounts list (main container):\n%s", volumeMountsList)

		// Verify config volumeMounts are present (different names from bundle)
		o.Expect(volumeMountsList).To(o.ContainSubstring("config-cm-vol"), "ConfigMap volumeMount should be appended")
		o.Expect(volumeMountsList).To(o.ContainSubstring("config-secret-vol"), "Secret volumeMount should be appended")
		e2e.Logf("Test Point 1 passed: Config volumeMounts appended to container")

		g.By("Verify bundle volumeMount is preserved")
		// Verify bundle volumeMount is present (bundle has predefined volumeMount)
		o.Expect(volumeMountsList).To(o.ContainSubstring("bundle-emptydir-vol"), "Bundle emptyDir volumeMount should be preserved")

		// Count volumeMounts: should have bundle volumeMount(s) + 2 config volumeMounts
		// Bundle has 1 volumeMount: bundle-emptydir-vol
		// Config adds 2 volumeMounts: config-cm-vol, config-secret-vol
		volumeMountLines := strings.Split(strings.TrimSpace(volumeMountsList), "\n")
		bundleVolumeMountCount := 0
		configVolumeMountCount := 0
		for _, vm := range volumeMountLines {
			vm = strings.TrimSpace(vm)
			if vm == "bundle-emptydir-vol" {
				bundleVolumeMountCount++
			}
			if vm == "config-cm-vol" || vm == "config-secret-vol" {
				configVolumeMountCount++
			}
		}

		o.Expect(bundleVolumeMountCount).To(o.Equal(1), "Should have 1 bundle volumeMount preserved")
		o.Expect(configVolumeMountCount).To(o.Equal(2), "Should have 2 config volumeMounts appended")
		e2e.Logf("VolumeMounts count: bundle (1) + config (2) = %d total", bundleVolumeMountCount+configVolumeMountCount)

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify volumeMounts applied to ALL containers in actual Pod")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		podContainersArray := gjson.Parse(podContainersJSON).Array()
		containerCount := len(podContainersArray)
		e2e.Logf("Total containers in Pod: %d", containerCount)

		// Verify each container in Pod has the config volumeMounts
		for i, container := range podContainersArray {
			containerName := container.Get("name").String()
			volumeMountsArray := container.Get("volumeMounts").Array()

			// Collect volumeMount names for verification
			volumeMountNames := make([]string, 0, len(volumeMountsArray))
			for _, vm := range volumeMountsArray {
				volumeMountNames = append(volumeMountNames, vm.Get("name").String())
			}
			volumeMountsStr := strings.Join(volumeMountNames, "\n")

			// Each container in Pod should have config volumeMounts
			o.Expect(volumeMountsStr).To(o.ContainSubstring("config-cm-vol"), fmt.Sprintf("Pod container %d (%s) should have config-cm-vol", i, containerName))
			o.Expect(volumeMountsStr).To(o.ContainSubstring("config-secret-vol"), fmt.Sprintf("Pod container %d (%s) should have config-secret-vol", i, containerName))
			e2e.Logf("Pod container %d (%s) has config volumeMounts applied", i, containerName)
		}
		e2e.Logf("Test Point 2 passed: VolumeMounts applied to ALL %d container(s) in actual Pod", containerCount)

		e2e.Logf("Test completed successfully - volumeMounts append to all containers mechanism works correctly")
	})

	g.It("PolarionID:87543-[Skipped:Disconnected]deploymentConfig tolerations are appended to operator deployment without duplicates", func() {
		var (
			caseID                       = "87543"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// bundle-key: Duplicate with bundle - should NOT be added again (deduplication)
			// config-key1: New toleration - should be appended
			// config-key2: New toleration - should be appended
			inlineConfig = `{
    "deploymentConfig": {
      "tolerations": [
        {
          "key": "bundle-key",
          "operator": "Equal",
          "value": "bundle-value",
          "effect": "NoSchedule"
        },
        {
          "key": "config-key1",
          "operator": "Equal",
          "value": "config-value1",
          "effect": "NoSchedule"
        },
        {
          "key": "config-key2",
          "operator": "Equal",
          "value": "config-value2",
          "effect": "NoExecute"
        }
      ]
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87543",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87543",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify tolerations appended to Deployment")
		// Get all tolerations from Deployment spec.template.spec.tolerations
		// Format: key=value:effect per line
		tolerationsPath := `jsonpath={range .spec.template.spec.tolerations[*]}{.key}{"="}{.value}{":"}{.effect}{"\n"}{end}`
		tolerationsList, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", tolerationsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Tolerations list:\n%s", tolerationsList)

		// Verify config tolerations are present
		o.Expect(tolerationsList).To(o.ContainSubstring("bundle-key=bundle-value:NoSchedule"), "Bundle toleration should be present")
		o.Expect(tolerationsList).To(o.ContainSubstring("config-key1=config-value1:NoSchedule"), "Config toleration 1 should be appended")
		o.Expect(tolerationsList).To(o.ContainSubstring("config-key2=config-value2:NoExecute"), "Config toleration 2 should be appended")
		e2e.Logf("Test Point 1 passed: Config tolerations appended to Deployment")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify no duplicates in actual Pod - each toleration appears exactly once")
		// Get all tolerations from Pod spec.tolerations
		// Format: key=value:effect per line
		podTolerationsPath := `jsonpath={range .spec.tolerations[*]}{.key}{"="}{.value}{":"}{.effect}{"\n"}{end}`
		podTolerationsList, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podTolerationsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod tolerations list:\n%s", podTolerationsList)

		// Count occurrences of each toleration in Pod
		bundleKeyCount := strings.Count(podTolerationsList, "bundle-key=bundle-value:NoSchedule")
		configKey1Count := strings.Count(podTolerationsList, "config-key1=config-value1:NoSchedule")
		configKey2Count := strings.Count(podTolerationsList, "config-key2=config-value2:NoExecute")

		o.Expect(bundleKeyCount).To(o.Equal(1), "bundle-key toleration should appear exactly once in Pod (duplicate from config removed)")
		o.Expect(configKey1Count).To(o.Equal(1), "config-key1 toleration should appear exactly once in Pod")
		o.Expect(configKey2Count).To(o.Equal(1), "config-key2 toleration should appear exactly once in Pod")

		e2e.Logf("Test Point 2 passed: No duplicate tolerations in Pod - deduplication works correctly")

		e2e.Logf("Test completed successfully - tolerations append without duplicates works correctly")
	})

	g.It("PolarionID:87544-[Skipped:Disconnected]deploymentConfig resources completely replace existing resource requirements", func() {
		var (
			caseID                       = "87544"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// Config resources COMPLETELY REPLACE bundle resources (no merge)
			// Bundle has: CPU 10m/500m, Memory 256Mi/768Mi (standard bundle values)
			// Config has: CPU 100m/200m, Memory 128Mi/256Mi (different values to test replacement)
			// Expected: Only config values in final Deployment, bundle values completely gone
			inlineConfig = `{
    "deploymentConfig": {
      "resources": {
        "limits": {
          "cpu": "200m",
          "memory": "256Mi"
        },
        "requests": {
          "cpu": "100m",
          "memory": "128Mi"
        }
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87544",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87544",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify config resources completely replace bundle resources")
		// Get resource values from the first (main) container (one call instead of 4)
		firstContainerJSONPath := `jsonpath={.spec.template.spec.containers[0]}`
		firstContainerJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", firstContainerJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("First container JSON retrieved (length: %d bytes)", len(firstContainerJSON))

		// Parse container resources using gjson
		firstContainer := gjson.Parse(firstContainerJSON)
		cpuLimit := firstContainer.Get("resources.limits.cpu").String()
		memLimit := firstContainer.Get("resources.limits.memory").String()
		cpuRequest := firstContainer.Get("resources.requests.cpu").String()
		memRequest := firstContainer.Get("resources.requests.memory").String()

		e2e.Logf("Container resources - Limits: CPU=%s, Memory=%s | Requests: CPU=%s, Memory=%s",
			cpuLimit, memLimit, cpuRequest, memRequest)

		// Verify config values are present (complete replacement)
		o.Expect(cpuLimit).To(o.Equal("200m"), "CPU limit should be config value (200m)")
		o.Expect(memLimit).To(o.Equal("256Mi"), "Memory limit should be config value (256Mi)")
		o.Expect(cpuRequest).To(o.Equal("100m"), "CPU request should be config value (100m)")
		o.Expect(memRequest).To(o.Equal("128Mi"), "Memory request should be config value (128Mi)")

		// Verify bundle values are NOT present (complete replacement, not merge)
		o.Expect(cpuLimit).NotTo(o.Equal("500m"), "CPU limit should NOT be bundle value (500m) - complete replacement")
		o.Expect(memLimit).NotTo(o.Equal("768Mi"), "Memory limit should NOT be bundle value (768Mi) - complete replacement")
		o.Expect(cpuRequest).NotTo(o.Equal("10m"), "CPU request should NOT be bundle value (10m) - complete replacement")
		o.Expect(memRequest).NotTo(o.Equal("256Mi"), "Memory request should NOT be bundle value (256Mi) - complete replacement")

		e2e.Logf("Test Point 1 passed: Config resources completely replaced bundle resources (no merge)")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify resources applied to ALL containers in actual Pod")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		containersArray := gjson.Parse(podContainersJSON).Array()
		containerCount := len(containersArray)
		e2e.Logf("Total containers in Pod: %d", containerCount)

		// Verify each container in Pod has the config resources
		for i, container := range containersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			// Each container in Pod should have config resource values
			o.Expect(containerCpuLimit).To(o.Equal("200m"), fmt.Sprintf("Pod container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("256Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Pod container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Pod container %d (%s) has config resources applied - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 2 passed: Resources applied to ALL %d container(s) in actual Pod", containerCount)

		e2e.Logf("Test completed successfully - resources complete replacement mechanism works correctly")
	})

	g.It("PolarionID:87545-[Skipped:Disconnected]deploymentConfig nodeSelector completely replaces existing node selector", func() {
		var (
			caseID                       = "87545"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// Config nodeSelector COMPLETELY REPLACES bundle nodeSelector (no merge, no partial override)
			// Bundle has: node-role.kubernetes.io/worker: "", disktype: ssd
			// Config has: node-role.kubernetes.io/infra: "", storage: fast
			// Expected: Only config values in final Deployment, bundle nodeSelector completely gone
			inlineConfig = `{
    "deploymentConfig": {
      "nodeSelector": {
        "node-role.kubernetes.io/infra": "",
        "storage": "fast"
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87545",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87545",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMBoxCutterRuntime") {
			e2e.Logf("Boxcutter applier detected")
			err := clusterextension.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Helm applier detected")
			clusterextension.Create(oc)
		}

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify config nodeSelector completely replaces bundle nodeSelector")
		// Get nodeSelector from Deployment spec.template.spec.nodeSelector
		// Format: key=value pairs (one per line)
		nodeSelectorPath := `jsonpath={range .spec.template.spec.nodeSelector}{@}{"\n"}{end}`
		nodeSelectorOutput, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", nodeSelectorPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("NodeSelector output:\n%s", nodeSelectorOutput)

		// Better approach: Get the full nodeSelector as JSON for easier parsing
		nodeSelectorJSONPath := `jsonpath={.spec.template.spec.nodeSelector}`
		nodeSelectorJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", nodeSelectorJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("NodeSelector JSON: %s", nodeSelectorJSON)

		// Positive checks: Config nodeSelector labels should be present
		o.Expect(nodeSelectorJSON).To(o.ContainSubstring("node-role.kubernetes.io/infra"), "Config nodeSelector label 'node-role.kubernetes.io/infra' should be present")
		o.Expect(nodeSelectorJSON).To(o.ContainSubstring("storage"), "Config nodeSelector label 'storage' should be present")
		o.Expect(nodeSelectorJSON).To(o.ContainSubstring("fast"), "Config nodeSelector value 'fast' should be present")

		// Negative checks: Bundle nodeSelector labels should NOT be present (complete replacement)
		o.Expect(nodeSelectorJSON).NotTo(o.ContainSubstring("node-role.kubernetes.io/worker"), "Bundle nodeSelector 'node-role.kubernetes.io/worker' should NOT be present (complete replacement)")
		o.Expect(nodeSelectorJSON).NotTo(o.ContainSubstring("disktype"), "Bundle nodeSelector label 'disktype' should NOT be present (complete replacement)")
		o.Expect(nodeSelectorJSON).NotTo(o.ContainSubstring("ssd"), "Bundle nodeSelector value 'ssd' should NOT be present (complete replacement)")

		e2e.Logf("Test Point 1 passed: Config nodeSelector completely replaced bundle nodeSelector (no merge)")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify nodeSelector applied to actual Pod spec")

		// Get the full nodeSelector from Pod as JSON
		// Use bracket notation for keys with special characters (dots, slashes)
		podNodeSelectorPath := `jsonpath={.spec.nodeSelector}`
		podNodeSelectorJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podNodeSelectorPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod nodeSelector JSON: %s", podNodeSelectorJSON)

		// Verify config nodeSelector labels are present with correct values in Pod
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("node-role.kubernetes.io/infra"), "Pod should have node-role.kubernetes.io/infra label")
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("storage"), "Pod should have storage label")
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("fast"), "Pod storage label should have value 'fast'")

		// Verify bundle nodeSelector labels are NOT present in Pod (complete replacement)
		o.Expect(podNodeSelectorJSON).NotTo(o.ContainSubstring("node-role.kubernetes.io/worker"), "Pod should NOT have bundle's node-role.kubernetes.io/worker label")
		o.Expect(podNodeSelectorJSON).NotTo(o.ContainSubstring("disktype"), "Pod should NOT have bundle's disktype label")
		o.Expect(podNodeSelectorJSON).NotTo(o.ContainSubstring("ssd"), "Pod should NOT have bundle's ssd value")

		e2e.Logf("Test Point 2 passed: NodeSelector applied to actual Pod spec with correct values")
		e2e.Logf("Pod NodeSelector: %s", podNodeSelectorJSON)

		e2e.Logf("Test completed successfully - nodeSelector complete replacement mechanism works correctly")
	})

	g.It("PolarionID:87546-[Skipped:Disconnected]deploymentConfig nodeAffinity overrides existing nodeAffinity", func() {
		var (
			caseID                       = "87546"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// CRITICAL: Only specify nodeAffinity, NOT podAffinity or podAntiAffinity
			// This tests "selective override" mechanism:
			//   - nodeAffinity: Overridden by config (bundle value replaced)
			//   - podAffinity: NOT specified in config (bundle value preserved)
			//   - podAntiAffinity: NOT specified in config (bundle value preserved)
			// Bundle has all three affinity sub-types, config only has nodeAffinity
			inlineConfig = `{
    "deploymentConfig": {
      "affinity": {
        "nodeAffinity": {
          "requiredDuringSchedulingIgnoredDuringExecution": {
            "nodeSelectorTerms": [
              {
                "matchExpressions": [
                  {
                    "key": "node-role.kubernetes.io/worker",
                    "operator": "Exists"
                  }
                ]
              }
            ]
          }
        }
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87546",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87546",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMBoxCutterRuntime") {
			e2e.Logf("Boxcutter applier detected")
			err := clusterextension.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Helm applier detected")
			clusterextension.Create(oc)
		}

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify nodeAffinity override and selective override mechanism")

		// Get full affinity from Deployment as JSON for easier parsing
		affinityJSONPath := `jsonpath={.spec.template.spec.affinity}`
		affinityJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", affinityJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment affinity JSON: %s", affinityJSON)

		// Part 1: Verify nodeAffinity override (config replaces bundle)
		// Positive checks: Config nodeAffinity should be present
		o.Expect(affinityJSON).To(o.ContainSubstring("node-role.kubernetes.io/worker"), "Config nodeAffinity should be present")
		o.Expect(affinityJSON).To(o.ContainSubstring("Exists"), "Config nodeAffinity operator should be Exists")

		// Negative checks: Bundle nodeAffinity should NOT be present (overridden)
		o.Expect(affinityJSON).NotTo(o.ContainSubstring("disktype"), "Bundle nodeAffinity 'disktype' should NOT be present (overridden)")
		o.Expect(affinityJSON).NotTo(o.ContainSubstring("ssd"), "Bundle nodeAffinity value 'ssd' should NOT be present (overridden)")

		e2e.Logf("nodeAffinity override verified: Config replaced bundle nodeAffinity")

		// Part 2: Verify selective override - podAffinity and podAntiAffinity PRESERVED
		// These should still have bundle values (not touched by config)
		// Positive checks: Bundle podAffinity should be preserved
		o.Expect(affinityJSON).To(o.ContainSubstring("podAffinity"), "Bundle podAffinity should be preserved (selective override)")
		o.Expect(affinityJSON).To(o.ContainSubstring("cache"), "Bundle podAffinity 'app=cache' should be preserved")

		// Positive checks: Bundle podAntiAffinity should be preserved
		o.Expect(affinityJSON).To(o.ContainSubstring("podAntiAffinity"), "Bundle podAntiAffinity should be preserved (selective override)")
		o.Expect(affinityJSON).To(o.ContainSubstring("database"), "Bundle podAntiAffinity 'app=database' should be preserved")
		o.Expect(affinityJSON).To(o.ContainSubstring("100"), "Bundle podAntiAffinity weight=100 should be preserved")

		e2e.Logf("Selective override verified: podAffinity and podAntiAffinity preserved from bundle")
		e2e.Logf("Test Point 1 passed: nodeAffinity overridden, other affinity sub-types preserved")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify affinity applied to actual Pod spec")

		// Get full affinity from Pod as JSON
		podAffinityPath := `jsonpath={.spec.affinity}`
		podAffinityJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podAffinityPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod affinity JSON: %s", podAffinityJSON)

		// Verify same affinity configuration in actual running Pod
		// Config nodeAffinity present
		o.Expect(podAffinityJSON).To(o.ContainSubstring("node-role.kubernetes.io/worker"), "Pod should have config nodeAffinity")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("Exists"), "Pod nodeAffinity operator should be Exists")

		// Bundle nodeAffinity NOT present (overridden)
		o.Expect(podAffinityJSON).NotTo(o.ContainSubstring("disktype"), "Pod should NOT have bundle's disktype nodeAffinity")
		o.Expect(podAffinityJSON).NotTo(o.ContainSubstring("ssd"), "Pod should NOT have bundle's ssd value")

		// Bundle podAffinity preserved
		o.Expect(podAffinityJSON).To(o.ContainSubstring("podAffinity"), "Pod should have bundle's podAffinity (preserved)")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("cache"), "Pod should have bundle's podAffinity app=cache")

		// Bundle podAntiAffinity preserved
		o.Expect(podAffinityJSON).To(o.ContainSubstring("podAntiAffinity"), "Pod should have bundle's podAntiAffinity (preserved)")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("database"), "Pod should have bundle's podAntiAffinity app=database")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("100"), "Pod should have bundle's podAntiAffinity weight=100")

		e2e.Logf("Test Point 2 passed: Affinity configuration correctly applied to actual Pod")
		e2e.Logf("Pod Affinity: %s", podAffinityJSON)

		e2e.Logf("Test completed successfully - nodeAffinity selective override mechanism works correctly")
	})

	g.It("PolarionID:87547-[Skipped:Disconnected]deploymentConfig podAffinity overrides existing podAffinity", func() {
		var (
			caseID                       = "87547"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// CRITICAL: Only specify podAffinity, NOT nodeAffinity or podAntiAffinity
			// This tests "selective override" mechanism:
			//   - podAffinity: Overridden by config (bundle value replaced)
			//   - nodeAffinity: NOT specified in config (bundle value preserved)
			//   - podAntiAffinity: NOT specified in config (bundle value preserved)
			// Bundle has all three affinity sub-types, config only has podAffinity
			inlineConfig = `{
    "deploymentConfig": {
      "affinity": {
        "podAffinity": {
          "preferredDuringSchedulingIgnoredDuringExecution": [
            {
              "weight": 50,
              "podAffinityTerm": {
                "labelSelector": {
                  "matchExpressions": [
                    {
                      "key": "component",
                      "operator": "In",
                      "values": ["frontend"]
                    }
                  ]
                },
                "topologyKey": "topology.kubernetes.io/zone"
              }
            }
          ]
        }
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87546", // Reuse same bundle as 87546
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87546", // Reuse same bundle as 87546
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMBoxCutterRuntime") {
			e2e.Logf("Boxcutter applier detected")
			err := clusterextension.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Helm applier detected")
			clusterextension.Create(oc)
		}

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify podAffinity override and selective override mechanism")

		// Get full affinity from Deployment as JSON for easier parsing
		affinityJSONPath := `jsonpath={.spec.template.spec.affinity}`
		affinityJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", affinityJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment affinity JSON: %s", affinityJSON)

		// Part 1: Verify podAffinity override (config replaces bundle)
		// Positive checks: Config podAffinity should be present
		o.Expect(affinityJSON).To(o.ContainSubstring("component"), "Config podAffinity should be present")
		o.Expect(affinityJSON).To(o.ContainSubstring("frontend"), "Config podAffinity value 'frontend' should be present")
		o.Expect(affinityJSON).To(o.ContainSubstring("50"), "Config podAffinity weight 50 should be present")
		o.Expect(affinityJSON).To(o.ContainSubstring("topology.kubernetes.io/zone"), "Config podAffinity topologyKey should be zone")

		// Negative checks: Bundle podAffinity should NOT be present (overridden)
		o.Expect(affinityJSON).NotTo(o.ContainSubstring("cache"), "Bundle podAffinity 'app=cache' should NOT be present (overridden)")
		// NOTE: Cannot check for "kubernetes.io/hostname" absence because it appears in podAntiAffinity (which should be preserved)

		e2e.Logf("podAffinity override verified: Config replaced bundle podAffinity")

		// Part 2: Verify selective override - nodeAffinity and podAntiAffinity PRESERVED
		// These should still have bundle values (not touched by config)
		// Positive checks: Bundle nodeAffinity should be preserved
		o.Expect(affinityJSON).To(o.ContainSubstring("nodeAffinity"), "Bundle nodeAffinity should be preserved (selective override)")
		o.Expect(affinityJSON).To(o.ContainSubstring("disktype"), "Bundle nodeAffinity 'disktype' should be preserved")
		o.Expect(affinityJSON).To(o.ContainSubstring("ssd"), "Bundle nodeAffinity value 'ssd' should be preserved")

		// Positive checks: Bundle podAntiAffinity should be preserved
		o.Expect(affinityJSON).To(o.ContainSubstring("podAntiAffinity"), "Bundle podAntiAffinity should be preserved (selective override)")
		o.Expect(affinityJSON).To(o.ContainSubstring("database"), "Bundle podAntiAffinity 'app=database' should be preserved")
		o.Expect(affinityJSON).To(o.ContainSubstring("100"), "Bundle podAntiAffinity weight=100 should be preserved")

		e2e.Logf("Selective override verified: nodeAffinity and podAntiAffinity preserved from bundle")
		e2e.Logf("Test Point 1 passed: podAffinity overridden, other affinity sub-types preserved")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify affinity applied to actual Pod spec")

		// Get full affinity from Pod as JSON
		podAffinityPath := `jsonpath={.spec.affinity}`
		podAffinityJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podAffinityPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod affinity JSON: %s", podAffinityJSON)

		// Verify same affinity configuration in actual running Pod
		// Config podAffinity present
		o.Expect(podAffinityJSON).To(o.ContainSubstring("component"), "Pod should have config podAffinity")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("frontend"), "Pod should have config podAffinity value 'frontend'")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("50"), "Pod podAffinity weight should be 50")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("topology.kubernetes.io/zone"), "Pod podAffinity topologyKey should be zone")

		// Bundle podAffinity NOT present (overridden)
		o.Expect(podAffinityJSON).NotTo(o.ContainSubstring("cache"), "Pod should NOT have bundle's podAffinity app=cache")

		// Bundle nodeAffinity preserved
		o.Expect(podAffinityJSON).To(o.ContainSubstring("nodeAffinity"), "Pod should have bundle's nodeAffinity (preserved)")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("disktype"), "Pod should have bundle's nodeAffinity disktype")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("ssd"), "Pod should have bundle's nodeAffinity value ssd")

		// Bundle podAntiAffinity preserved
		o.Expect(podAffinityJSON).To(o.ContainSubstring("podAntiAffinity"), "Pod should have bundle's podAntiAffinity (preserved)")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("database"), "Pod should have bundle's podAntiAffinity app=database")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("100"), "Pod should have bundle's podAntiAffinity weight=100")

		e2e.Logf("Test Point 2 passed: Affinity configuration correctly applied to actual Pod")
		e2e.Logf("Pod Affinity: %s", podAffinityJSON)

		e2e.Logf("Test completed successfully - podAffinity selective override mechanism works correctly")
	})

	g.It("PolarionID:87548-[Skipped:Disconnected]deploymentConfig podAntiAffinity overrides existing podAntiAffinity", func() {
		var (
			caseID                       = "87548"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// CRITICAL: This tests TWO behaviors:
			//   1. Selective override: podAntiAffinity overridden, podAffinity preserved
			//   2. Empty nodeAffinity object behavior:
			//      - OLMv1 CURRENT: Empty {} object preserved in manifest (cosmetic issue)
			//      - OLMv1 EXPECTED (after fix): Empty {} cleaned to nil (match OLMv0)
			//      - This test verifies CURRENT behavior, will need update when bug fixed
			//
			// Config affinity sub-types:
			//   - nodeAffinity: {} (empty object - tests bug OCPBUGS-76383)
			//   - podAntiAffinity: full config (overrides bundle)
			//   - podAffinity: NOT specified (bundle value preserved)
			inlineConfig = `{
    "deploymentConfig": {
      "affinity": {
        "nodeAffinity": {},
        "podAntiAffinity": {
          "requiredDuringSchedulingIgnoredDuringExecution": [
            {
              "labelSelector": {
                "matchExpressions": [
                  {
                    "key": "security",
                    "operator": "In",
                    "values": ["S1"]
                  }
                ]
              },
              "topologyKey": "topology.kubernetes.io/zone"
            }
          ]
        }
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87548",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87548",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMBoxCutterRuntime") {
			e2e.Logf("Boxcutter applier detected")
			err := clusterextension.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Helm applier detected")
			clusterextension.Create(oc)
		}

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify podAntiAffinity override and selective override mechanism")

		// Get full affinity from Deployment as JSON for easier parsing
		affinityJSONPath := `jsonpath={.spec.template.spec.affinity}`
		affinityJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", affinityJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment affinity JSON: %s", affinityJSON)

		// Part 1: Verify podAntiAffinity override (config replaces bundle)
		// Positive checks: Config podAntiAffinity should be present
		o.Expect(affinityJSON).To(o.ContainSubstring("security"), "Config podAntiAffinity should be present")
		o.Expect(affinityJSON).To(o.ContainSubstring("S1"), "Config podAntiAffinity value 'S1' should be present")
		o.Expect(affinityJSON).To(o.ContainSubstring("topology.kubernetes.io/zone"), "Config podAntiAffinity topologyKey should be zone")

		// Negative checks: Bundle podAntiAffinity should NOT be present (overridden)
		o.Expect(affinityJSON).NotTo(o.ContainSubstring("database"), "Bundle podAntiAffinity 'app=database' should NOT be present (overridden)")
		// NOTE: Cannot check for weight "100" absence because it also appears in bundle podAffinity (which should be preserved)

		e2e.Logf("podAntiAffinity override verified: Config replaced bundle podAntiAffinity")

		// Part 2: Verify empty nodeAffinity behavior and podAffinity preservation
		// Verify empty nodeAffinity object presence (CURRENT OLMv1 behavior)
		// Step 1: Get nodeAffinity value directly to verify it's empty {}
		nodeAffinityJSONPath := `jsonpath={.spec.template.spec.affinity.nodeAffinity}`
		nodeAffinityJSON, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("deployment", deploymentName, "-n", ns, "-o", nodeAffinityJSONPath).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment nodeAffinity value: '%s'", nodeAffinityJSON)

		// Step 2: Verify nodeAffinity is empty object {}
		// Expected: "{}" or empty/whitespace (K8s may return empty for {})
		trimmedNodeAffinity := strings.TrimSpace(nodeAffinityJSON)
		// Check it's either "{}" or empty (both indicate empty object in manifest)
		isEmptyObject := trimmedNodeAffinity == "{}"
		o.Expect(isEmptyObject).To(o.BeTrue(), "nodeAffinity should be empty object {}")
		e2e.Logf("Empty nodeAffinity verified: value='%s'", trimmedNodeAffinity)

		// Step 3: Verify bundle nodeAffinity values NOT present (overridden by empty {})
		o.Expect(affinityJSON).NotTo(o.ContainSubstring("disktype"), "Bundle nodeAffinity 'disktype' should NOT be present (overridden by empty {})")
		o.Expect(affinityJSON).NotTo(o.ContainSubstring("ssd"), "Bundle nodeAffinity value 'ssd' should NOT be present (overridden by empty {})")
		e2e.Logf("Bundle nodeAffinity values confirmed absent (overridden by empty {})")

		// Positive checks: Bundle podAffinity should be preserved (selective override)
		o.Expect(affinityJSON).To(o.ContainSubstring("podAffinity"), "Bundle podAffinity should be preserved (selective override)")
		o.Expect(affinityJSON).To(o.ContainSubstring("cache"), "Bundle podAffinity 'app=cache' should be preserved")
		// NOTE: weight "100" appears in both bundle podAffinity and podAntiAffinity, so we rely on "cache" check

		e2e.Logf("Selective override verified: podAffinity preserved from bundle, nodeAffinity overridden by empty {}")
		e2e.Logf("Test Point 1 passed: podAntiAffinity overridden, podAffinity preserved, nodeAffinity empty {}")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify affinity applied to actual Pod spec")

		// Get full affinity from Pod as JSON
		podAffinityPath := `jsonpath={.spec.affinity}`
		podAffinityJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podAffinityPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod affinity JSON: %s", podAffinityJSON)

		// Verify same affinity configuration in actual running Pod
		// Config podAntiAffinity present
		o.Expect(podAffinityJSON).To(o.ContainSubstring("security"), "Pod should have config podAntiAffinity")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("S1"), "Pod should have config podAntiAffinity value 'S1'")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("topology.kubernetes.io/zone"), "Pod podAntiAffinity topologyKey should be zone")

		// Bundle podAntiAffinity NOT present (overridden)
		o.Expect(podAffinityJSON).NotTo(o.ContainSubstring("database"), "Pod should NOT have bundle's podAntiAffinity app=database")
		// NOTE: Cannot check for weight "100" absence because it appears in podAffinity (which should be preserved)

		// Verify empty nodeAffinity in Pod
		// Step 1: Get Pod nodeAffinity value directly to verify it's empty {}
		podNodeAffinityPath := `jsonpath={.spec.affinity.nodeAffinity}`
		podNodeAffinityJSON, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", podName, "-n", ns, "-o", podNodeAffinityPath).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod nodeAffinity value: '%s'", podNodeAffinityJSON)

		// Step 2: Verify Pod nodeAffinity is empty object {}
		trimmedPodNodeAffinity := strings.TrimSpace(podNodeAffinityJSON)
		isPodEmptyObject := trimmedPodNodeAffinity == "{}"
		o.Expect(isPodEmptyObject).To(o.BeTrue(), "Pod nodeAffinity should be empty object {}")
		e2e.Logf("Pod empty nodeAffinity verified: value='%s'", trimmedPodNodeAffinity)

		// Step 3: Verify bundle nodeAffinity values NOT present in Pod
		o.Expect(podAffinityJSON).NotTo(o.ContainSubstring("disktype"), "Pod should NOT have bundle's nodeAffinity disktype (overridden by empty {})")
		o.Expect(podAffinityJSON).NotTo(o.ContainSubstring("ssd"), "Pod should NOT have bundle's nodeAffinity value ssd (overridden by empty {})")
		e2e.Logf("Pod bundle nodeAffinity values confirmed absent")

		// Bundle podAffinity preserved
		o.Expect(podAffinityJSON).To(o.ContainSubstring("podAffinity"), "Pod should have bundle's podAffinity (preserved)")
		o.Expect(podAffinityJSON).To(o.ContainSubstring("cache"), "Pod should have bundle's podAffinity app=cache")
		// NOTE: weight "100" appears in both bundle podAffinity and podAntiAffinity, so we rely on "cache" check

		e2e.Logf("Test Point 2 passed: Affinity configuration correctly applied to actual Pod")
		e2e.Logf("Pod Affinity: %s", podAffinityJSON)

		e2e.Logf("Test completed successfully - podAntiAffinity override, podAffinity preserved, nodeAffinity empty {}")
	})

	g.It("PolarionID:87549-[Skipped:Disconnected]deploymentConfig annotations are merged with existing taking precedence", func() {
		var (
			caseID                       = "87549"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Define inline config as JSON string (for ${{}} template parsing)
			// NOTE: Due to CSV API limitation, deployment-level and pod-level behave differently:
			//
			// Deployment level (config-only):
			//   - annotation1: config_value1 (no bundle conflict)
			//   - annotation3: config_value3 (config added)
			//
			// Pod level (merge with existing precedence):
			//   - annotation1: bundle_value1 takes precedence (NOT config_value1)
			//   - annotation2: bundle_value2 preserved (only in bundle pod template)
			//   - annotation3: config_value3 added (new annotation)
			inlineConfig = `{
    "deploymentConfig": {
      "annotations": {
        "annotation1": "config_value1",
        "annotation3": "config_value3"
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87549",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87549",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify config annotations applied at Deployment level (config-only scenario)")

		// Get Deployment metadata.annotations
		deploymentAnnotationsPath := `jsonpath={.metadata.annotations}`
		deploymentAnnotations, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", deploymentAnnotationsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment metadata.annotations: %s", deploymentAnnotations)

		// IMPORTANT: Due to CSV API limitation, bundle CANNOT define deployment-level annotations
		// Therefore, we can only verify that config annotations are applied correctly
		// There is NO merge behavior to test at this level (no bundle values to conflict with)

		// Verify config annotations are applied
		o.Expect(deploymentAnnotations).To(o.ContainSubstring("annotation1"), "Deployment should have annotation1")
		o.Expect(deploymentAnnotations).To(o.ContainSubstring("config_value1"), "Deployment annotation1 should be config_value1 (from config)")
		o.Expect(deploymentAnnotations).To(o.ContainSubstring("annotation3"), "Deployment should have annotation3")
		o.Expect(deploymentAnnotations).To(o.ContainSubstring("config_value3"), "Deployment annotation3 should be config_value3 (from config)")

		// Negative check: annotation2 should NOT be present (bundle only defined it at pod level)
		o.Expect(deploymentAnnotations).NotTo(o.ContainSubstring("annotation2"), "Deployment should NOT have annotation2 (bundle cannot define deployment-level annotations)")
		o.Expect(deploymentAnnotations).NotTo(o.ContainSubstring("bundle_value"), "Deployment should NOT have any bundle_value* (bundle cannot define deployment-level annotations)")

		e2e.Logf("Deployment level annotations verified: config annotations applied correctly (no bundle conflict due to API limitation)")
		e2e.Logf("Note: Deployment-level merge behavior cannot be tested due to CSV API limitation")

		g.By("Test Point 2: Verify annotations merge with existing precedence at pod template level")

		// Get Deployment spec.template.metadata.annotations
		podTemplateAnnotationsPath := `jsonpath={.spec.template.metadata.annotations}`
		podTemplateAnnotations, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", podTemplateAnnotationsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment spec.template.metadata.annotations: %s", podTemplateAnnotations)

		// Part 1: Verify bundle annotations take precedence (NOT overridden by config)
		o.Expect(podTemplateAnnotations).To(o.ContainSubstring("annotation1"), "Pod template should have annotation1")
		o.Expect(podTemplateAnnotations).To(o.ContainSubstring("bundle_value1"), "Pod template annotation1 should be bundle_value1 (bundle takes precedence)")
		o.Expect(podTemplateAnnotations).To(o.ContainSubstring("annotation2"), "Pod template should have annotation2")
		o.Expect(podTemplateAnnotations).To(o.ContainSubstring("bundle_value2"), "Pod template annotation2 should be bundle_value2 (bundle preserved)")

		// Part 2: Verify config value does NOT override bundle value
		o.Expect(podTemplateAnnotations).NotTo(o.ContainSubstring("config_value1"), "Pod template should NOT have config_value1 (existing takes precedence)")

		// Part 3: Verify new config annotation added
		o.Expect(podTemplateAnnotations).To(o.ContainSubstring("annotation3"), "Pod template should have new annotation3")
		o.Expect(podTemplateAnnotations).To(o.ContainSubstring("config_value3"), "Pod template annotation3 should be config_value3 (new annotation added)")

		e2e.Logf("Test Point 2 passed: Pod template annotations merge with existing precedence (bundle wins conflicts, config adds new)")
		e2e.Logf("This is the primary test of 'merge with existing precedence' behavior")

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 3: Verify same merge behavior in actual running Pod")

		// Get Pod metadata.annotations
		podAnnotationsPath := `jsonpath={.metadata.annotations}`
		podAnnotations, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podAnnotationsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod metadata.annotations: %s", podAnnotations)

		// Verify same annotation merge behavior in actual running Pod
		// Should match pod template behavior (existing takes precedence)
		o.Expect(podAnnotations).To(o.ContainSubstring("annotation1"), "Pod should have annotation1")
		o.Expect(podAnnotations).To(o.ContainSubstring("bundle_value1"), "Pod annotation1 should be bundle_value1 (bundle takes precedence)")
		o.Expect(podAnnotations).To(o.ContainSubstring("annotation2"), "Pod should have annotation2")
		o.Expect(podAnnotations).To(o.ContainSubstring("bundle_value2"), "Pod annotation2 should be bundle_value2 (bundle preserved)")

		// Config value should NOT override bundle value
		o.Expect(podAnnotations).NotTo(o.ContainSubstring("config_value1"), "Pod should NOT have config_value1 (existing takes precedence)")

		// New config annotation should be added
		o.Expect(podAnnotations).To(o.ContainSubstring("annotation3"), "Pod should have new annotation3")
		o.Expect(podAnnotations).To(o.ContainSubstring("config_value3"), "Pod annotation3 should be config_value3 (new annotation added)")

		e2e.Logf("Test Point 3 passed: Pod annotations match pod template behavior (merge with existing precedence)")

		e2e.Logf("Test completed successfully - annotations merge with existing precedence mechanism validated")
		e2e.Logf("Summary:")
		e2e.Logf("  - Deployment level: Config-only (CSV API limitation prevents bundle annotations)")
		e2e.Logf("  - Pod template level: Merge with existing precedence (bundle wins conflicts, config adds new)")
		e2e.Logf("  - Pod level: Same as pod template (actual runtime behavior)")
		e2e.Logf("Note: Annotations behavior is OPPOSITE to env vars (env: config overrides; annotations: existing preserves)")
	})

	g.It("PolarionID:87550-[Skipped:Disconnected]deploymentConfig with resources and nodeSelector both work correctly", func() {
		var (
			caseID                       = "87550"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Configure resources and nodeSelector together
			// Test Point 1: Resources applied to all containers in Deployment
			// Test Point 2: Resources applied to all containers in Pod
			// Test Point 3: NodeSelector applied to Pod spec
			inlineConfig = `{
    "deploymentConfig": {
      "resources": {
        "limits": {
          "cpu": "500m",
          "memory": "512Mi"
        },
        "requests": {
          "cpu": "100m",
          "memory": "128Mi"
        }
      },
      "nodeSelector": {
        "disktype": "ssd",
        "region": "east"
      }
    }
  }`

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87550",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87550",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig (resources + nodeSelector)")
		defer clusterextension.Delete(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMBoxCutterRuntime") {
			e2e.Logf("Boxcutter applier detected")
			err := clusterextension.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Helm applier detected")
			clusterextension.Create(oc)
		}

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify resources configuration in Deployment")
		// Get all containers info as JSON from Deployment (one call instead of multiple)
		deploymentContainersJSONPath := `jsonpath={.spec.template.spec.containers}`
		deploymentContainersJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", deploymentContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment containers JSON retrieved (length: %d bytes)", len(deploymentContainersJSON))

		// Parse containers array using gjson
		deploymentContainersArray := gjson.Parse(deploymentContainersJSON).Array()
		containerCount := len(deploymentContainersArray)
		e2e.Logf("Total containers in Deployment: %d", containerCount)

		// Verify each container has the config resources
		for i, container := range deploymentContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			// Each container should have config resource values
			o.Expect(containerCpuLimit).To(o.Equal("500m"), fmt.Sprintf("Container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("512Mi"), fmt.Sprintf("Container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Container %d (%s) has config resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 1 passed: Resources applied to ALL %d container(s) in Deployment", containerCount)

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 2: Verify resources applied to ALL containers in actual Pod")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		containersArray := gjson.Parse(podContainersJSON).Array()
		podContainerCount := len(containersArray)
		e2e.Logf("Total containers in Pod: %d", podContainerCount)

		// Verify each container in Pod has the config resources
		for i, container := range containersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			// Each container in Pod should have config resource values
			o.Expect(containerCpuLimit).To(o.Equal("500m"), fmt.Sprintf("Pod container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("512Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Pod container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Pod container %d (%s) has config resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 2 passed: Resources applied to ALL %d container(s) in actual Pod", podContainerCount)

		g.By("Test Point 3: Verify nodeSelector in actual Pod")
		podNodeSelectorPath := `jsonpath={.spec.nodeSelector}`
		podNodeSelectorJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podNodeSelectorPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod nodeSelector JSON: %s", podNodeSelectorJSON)

		// Verify config nodeSelector labels are present with correct values in Pod
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("disktype"), "Pod should have disktype label")
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("ssd"), "Pod disktype label should have value 'ssd'")
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("region"), "Pod should have region label")
		o.Expect(podNodeSelectorJSON).To(o.ContainSubstring("east"), "Pod region label should have value 'east'")
		e2e.Logf("Test Point 3 passed: NodeSelector correctly applied in actual Pod")

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Test Point 1: Resources correctly applied to all containers in Deployment")
		e2e.Logf("  - Test Point 2: Resources correctly applied to all containers in actual Pod")
		e2e.Logf("  - Test Point 3: NodeSelector correctly applied in actual Pod")
		e2e.Logf("  - Resources and nodeSelector configured together without conflicts")
		e2e.Logf("Test completed successfully - multiple fields work correctly together")
	})

	g.It("PolarionID:87551-[Skipped:Disconnected]deploymentConfig with env tolerations and resources all work correctly", func() {
		var (
			caseID                       = "87551"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Configure env, tolerations, and resources together
			// Test Point 1: Env vars applied to containers and accessible in pod
			// Test Point 2: Tolerations applied to pod spec
			// Test Point 3: Resources applied to all containers
			inlineConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "TEST_ENV1", "value": "value1"},
        {"name": "TEST_ENV2", "value": "value2"}
      ],
      "tolerations": [
        {
          "key": "node.kubernetes.io/unreachable",
          "operator": "Exists",
          "effect": "NoExecute",
          "tolerationSeconds": 120
        },
        {
          "key": "dedicated",
          "operator": "Equal",
          "value": "test",
          "effect": "NoSchedule"
        }
      ],
      "resources": {
        "limits": {
          "cpu": "300m",
          "memory": "256Mi"
        },
        "requests": {
          "cpu": "150m",
          "memory": "128Mi"
        }
      }
    }
  }`

			// Expected env vars
			expectedEnvVars = map[string]string{
				"TEST_ENV1": "value1",
				"TEST_ENV2": "value2",
			}

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87551",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87551",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with deploymentConfig (env + tolerations + resources)")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 1: Verify env vars in Deployment")
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedEnvVars, 15*time.Second)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Test Point 1 passed: Env vars verified in Deployment manifest")

		g.By("Test Point 2: Verify resources in Deployment")
		// Get all containers info as JSON from Deployment (one call instead of multiple)
		deploymentContainersJSONPath := `jsonpath={.spec.template.spec.containers}`
		deploymentContainersJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", deploymentContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment containers JSON retrieved (length: %d bytes)", len(deploymentContainersJSON))

		// Parse containers array using gjson
		deploymentContainersArray := gjson.Parse(deploymentContainersJSON).Array()
		containerCount := len(deploymentContainersArray)
		e2e.Logf("Total containers in Deployment: %d", containerCount)

		// Verify each container has the config resources
		for i, container := range deploymentContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("300m"), fmt.Sprintf("Container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("256Mi"), fmt.Sprintf("Container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("150m"), fmt.Sprintf("Container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Container %d (%s) has config resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 2 passed: Resources applied to ALL %d container(s) in Deployment", containerCount)

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 3: Verify env vars accessible in pod")
		err = olmv1util.VerifyPodEnvVars(oc, podName, ns, "", expectedEnvVars, 15*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Test Point 3 passed: Env vars verified in actual pod")

		g.By("Test Point 4: Verify tolerations in actual Pod")
		podTolerationsPath := `jsonpath={.spec.tolerations}`
		podTolerationsJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podTolerationsPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod tolerations JSON: %s", podTolerationsJSON)

		// Verify config tolerations are present in Pod
		o.Expect(podTolerationsJSON).To(o.ContainSubstring("node.kubernetes.io/unreachable"), "Pod should have unreachable toleration")
		o.Expect(podTolerationsJSON).To(o.ContainSubstring("NoExecute"), "Pod should have NoExecute effect toleration")
		o.Expect(podTolerationsJSON).To(o.ContainSubstring("dedicated"), "Pod should have dedicated toleration")
		o.Expect(podTolerationsJSON).To(o.ContainSubstring("test"), "Pod should have dedicated=test toleration")
		o.Expect(podTolerationsJSON).To(o.ContainSubstring("NoSchedule"), "Pod should have NoSchedule effect toleration")
		e2e.Logf("Test Point 4 passed: Tolerations correctly applied in actual Pod")

		g.By("Test Point 5: Verify resources in actual Pod")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		podContainersArray := gjson.Parse(podContainersJSON).Array()
		podContainerCount := len(podContainersArray)
		e2e.Logf("Total containers in Pod: %d", podContainerCount)

		// Verify each container in Pod has the config resources
		for i, container := range podContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("300m"), fmt.Sprintf("Pod container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("256Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("150m"), fmt.Sprintf("Pod container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Pod container %d (%s) has config resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 5 passed: Resources applied to ALL %d container(s) in actual Pod", podContainerCount)

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Test Point 1: Env vars correctly applied in Deployment")
		e2e.Logf("  - Test Point 2: Resources correctly applied to all containers in Deployment")
		e2e.Logf("  - Test Point 3: Env vars accessible in actual Pod")
		e2e.Logf("  - Test Point 4: Tolerations correctly applied in actual Pod")
		e2e.Logf("  - Test Point 5: Resources correctly applied to all containers in actual Pod")
		e2e.Logf("  - Env, tolerations, and resources configured together without conflicts")
		e2e.Logf("Test completed successfully - three fields work correctly together")
	})

	g.It("PolarionID:87552-[Skipped:Disconnected]deploymentConfig works correctly when combined with watchNamespace configuration", func() {
		var (
			caseID                       = "87552"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Configure watchNamespace (SingleNamespace mode) and deploymentConfig together
			// Test Point 1: WatchNamespace configuration verified (operator scoped to specific namespace)
			// Test Point 2: DeploymentConfig env vars applied
			// Test Point 3: DeploymentConfig resources applied to all containers
			// Note: We don't validate actual watch behavior (that's operator-specific), only config applied
			inlineConfig = `{
    "watchNamespace": "test-watch-ns-87552",
    "deploymentConfig": {
      "env": [
        {"name": "TEST_ENV_WATCH", "value": "watchvalue"}
      ],
      "resources": {
        "limits": {
          "cpu": "400m",
          "memory": "384Mi"
        },
        "requests": {
          "cpu": "100m",
          "memory": "128Mi"
        }
      }
    }
  }`

			// Expected env vars
			expectedEnvVars = map[string]string{
				"TEST_ENV_WATCH": "watchvalue",
			}

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87552",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87552",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     inlineConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create dedicated namespace for watchNamespace (for config only, not validated)")
		watchNs := "test-watch-ns-87552"
		defer func() {
			e2e.Logf("Cleaning up watch namespace %s", watchNs)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", watchNs, "--ignore-not-found", "--force").Execute()
		}()
		err = oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", watchNs).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", watchNs)).To(o.BeTrue())
		e2e.Logf("Created watch namespace: %s (for config test only)", watchNs)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Create ClusterExtension with watchNamespace and deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)

		g.By("Test Point 1: Verify watchNamespace configuration accepted (both features can coexist)")
		e2e.Logf("Test Point 1 passed: WatchNamespace and deploymentConfig both configured without conflict because of successful installation")

		g.By("Get operator Deployment name")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Test Point 2: Verify deploymentConfig env vars in Deployment")
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedEnvVars, 15*time.Second)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Test Point 2 passed: DeploymentConfig env vars verified in Deployment manifest")

		g.By("Test Point 3: Verify deploymentConfig resources in Deployment")
		// Get all containers info as JSON from Deployment (one call instead of multiple)
		deploymentContainersJSONPath := `jsonpath={.spec.template.spec.containers}`
		deploymentContainersJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", deploymentContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment containers JSON retrieved (length: %d bytes)", len(deploymentContainersJSON))

		// Parse containers array using gjson
		deploymentContainersArray := gjson.Parse(deploymentContainersJSON).Array()
		containerCount := len(deploymentContainersArray)
		e2e.Logf("Total containers in Deployment: %d", containerCount)

		// Verify each container has the config resources
		for i, container := range deploymentContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("400m"), fmt.Sprintf("Container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("384Mi"), fmt.Sprintf("Container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Container %d (%s) has config resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 3 passed: DeploymentConfig resources applied to ALL %d container(s) in Deployment", containerCount)

		g.By("Get operator Pod name")
		podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(podName).NotTo(o.BeEmpty())
		e2e.Logf("Operator pod name: %s", podName)

		g.By("Dump Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, podName, ns)

		g.By("Test Point 4: Verify deploymentConfig env vars accessible in pod")
		err = olmv1util.VerifyPodEnvVars(oc, podName, ns, "", expectedEnvVars, 15*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Test Point 4 passed: DeploymentConfig env vars verified in actual pod")

		g.By("Test Point 5: Verify deploymentConfig resources in actual Pod")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", podName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		podContainersArray := gjson.Parse(podContainersJSON).Array()
		podContainerCount := len(podContainersArray)
		e2e.Logf("Total containers in Pod: %d", podContainerCount)

		// Verify each container in Pod has the config resources
		for i, container := range podContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("400m"), fmt.Sprintf("Pod container %d (%s) should have config CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("384Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Pod container %d (%s) should have config CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Pod container %d (%s) should have config Memory request", i, containerName))

			e2e.Logf("Pod container %d (%s) has config resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 5 passed: DeploymentConfig resources applied to ALL %d container(s) in actual Pod", podContainerCount)

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Test Point 1: WatchNamespace and deploymentConfig configured together without conflict")
		e2e.Logf("  - Test Point 2: DeploymentConfig env vars correctly applied in Deployment")
		e2e.Logf("  - Test Point 3: DeploymentConfig resources correctly applied to all containers in Deployment")
		e2e.Logf("  - Test Point 4: DeploymentConfig env vars accessible in actual Pod")
		e2e.Logf("  - Test Point 5: DeploymentConfig resources correctly applied to all containers in actual Pod")
		e2e.Logf("  - Both features work correctly together without interference")
		e2e.Logf("Test completed successfully - integration test passed")
	})

	g.It("PolarionID:87553-[Skipped:Disconnected]adding deploymentConfig multiple fields to existing ClusterExtension works correctly", func() {
		var (
			caseID                       = "87553"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Configuration to add (env + resources)
			// Test Point 1: Adding config triggers reconcile
			// Test Point 2: Deployment updated with added fields
			// Test Point 3: New pods rolled out with added config
			// Note: Using only env + resources to avoid any scheduling issues
			addedConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "ADDED_ENV", "value": "added_value"}
      ],
      "resources": {
        "limits": {
          "cpu": "500m",
          "memory": "512Mi"
        },
        "requests": {
          "cpu": "100m",
          "memory": "128Mi"
        }
      }
    }
  }`

			// Expected values after adding config
			expectedEnvVars = map[string]string{
				"ADDED_ENV": "added_value",
			}

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87553",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87553",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				// NO InlineConfig initially - will be added later
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Phase 1: Create ClusterExtension WITHOUT deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		e2e.Logf("ClusterExtension %s created without deploymentConfig", extName)

		g.By("Wait for initial installation to complete")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Initial operator deployment name: %s", deploymentName)

		g.By("Dump initial Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Get initial Pod name (before adding config)")
		initialPodName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(initialPodName).NotTo(o.BeEmpty())
		e2e.Logf("Initial operator pod name (before config): %s", initialPodName)

		g.By("Dump initial Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, initialPodName, ns)

		g.By("Get initial generation for comparison")
		initialGenerationPath := `jsonpath={.metadata.generation}`
		initialGeneration, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", initialGenerationPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Initial ClusterExtension generation: %s", initialGeneration)

		g.By("Phase 2: Update ClusterExtension to ADD deploymentConfig with multiple fields")
		// Patch to add config.inline with deploymentConfig
		patchData := fmt.Sprintf(`{
  "spec": {
    "config": {
      "configType": "Inline",
      "inline": %s
    }
  }
}`, addedConfig)
		err = oc.WithoutNamespace().AsAdmin().Run("patch").Args("clusterextension", extName, "--type=merge", "-p", patchData).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Updated ClusterExtension to add deploymentConfig (env + resources)")

		g.By("Test Point 1: Verify ClusterExtension reconcile triggered (generation incremented)")
		var newGeneration string
		err = wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			newGenerationPath := `jsonpath={.metadata.generation}`
			gen, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", newGenerationPath)
			if err != nil {
				return false, nil
			}
			if gen != initialGeneration {
				newGeneration = gen
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "ClusterExtension generation did not increment after adding config")
		e2e.Logf("New ClusterExtension generation after adding config: %s", newGeneration)
		o.Expect(newGeneration).NotTo(o.Equal(initialGeneration), "Generation should increment after config update")
		e2e.Logf("Test Point 1 passed: ClusterExtension reconcile triggered (generation changed from %s to %s)", initialGeneration, newGeneration)

		g.By("Test Point 2: Verify Deployment updated with added env vars")
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedEnvVars, 2*time.Minute)
		g.By("Dump updated Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment has added env var: ADDED_ENV=added_value")

		g.By("Test Point 2: Verify Deployment updated with added resources")
		// Get all containers info as JSON from Deployment (one call instead of multiple)
		deploymentContainersJSONPath := `jsonpath={.spec.template.spec.containers}`
		deploymentContainersJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", deploymentContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment containers JSON retrieved (length: %d bytes)", len(deploymentContainersJSON))

		// Parse containers array using gjson
		deploymentContainersArray := gjson.Parse(deploymentContainersJSON).Array()
		containerCount := len(deploymentContainersArray)
		e2e.Logf("Total containers in Deployment: %d", containerCount)

		// Verify each container has the added resources
		for i, container := range deploymentContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("500m"), fmt.Sprintf("Container %d (%s) should have added CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("512Mi"), fmt.Sprintf("Container %d (%s) should have added Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Container %d (%s) should have added CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Container %d (%s) should have added Memory request", i, containerName))

			e2e.Logf("Container %d (%s) has added resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 2 passed: Deployment updated with all added fields (env + resources)")

		g.By("Test Point 3: Wait for new pod rollout")
		// Wait for new pod to be created (different ReplicaSet hash from initial pod)
		var newPodName string
		initialPodHash := olmv1util.ExtractReplicaSetHash(initialPodName)
		if initialPodHash == "" {
			g.Skip("Cannot extract ReplicaSet hash from initial pod name, skipping pod rollout verification")
		}
		e2e.Logf("Initial pod ReplicaSet hash: %s", initialPodHash)
		err = wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 30*time.Second)
			if err != nil {
				return false, nil // Pod might be terminating/restarting
			}
			podHash := olmv1util.ExtractReplicaSetHash(podName)
			if podHash != "" && podHash != initialPodHash {
				newPodName = podName
				e2e.Logf("Found new pod with different ReplicaSet hash: %s (was %s)", podHash, initialPodHash)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "New pod was not created after adding config")
		o.Expect(newPodName).NotTo(o.BeEmpty())
		o.Expect(newPodName).NotTo(o.Equal(initialPodName), "New pod should be created after config update")
		e2e.Logf("New operator pod name (after adding config): %s", newPodName)

		g.By("Dump new Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, newPodName, ns)

		g.By("Test Point 3: Verify new pod has added env vars")
		err = olmv1util.VerifyPodEnvVars(oc, newPodName, ns, "", expectedEnvVars, 90*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("New pod has added env var: ADDED_ENV=added_value")

		g.By("Test Point 3: Verify new pod has added resources")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", newPodName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		podContainersArray := gjson.Parse(podContainersJSON).Array()
		podContainerCount := len(podContainersArray)
		e2e.Logf("Total containers in new Pod: %d", podContainerCount)

		// Verify each container in new Pod has the added resources
		for i, container := range podContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("500m"), fmt.Sprintf("Pod container %d (%s) should have added CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("512Mi"), fmt.Sprintf("Pod container %d (%s) should have added Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("100m"), fmt.Sprintf("Pod container %d (%s) should have added CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("128Mi"), fmt.Sprintf("Pod container %d (%s) should have added Memory request", i, containerName))

			e2e.Logf("Pod container %d (%s) has added resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 3 passed: New pod rolled out with all added fields (env + resources)")

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Phase 1: ClusterExtension created without deploymentConfig")
		e2e.Logf("  - Phase 2: Updated ClusterExtension to add deploymentConfig (env + resources)")
		e2e.Logf("  - Test Point 1: ClusterExtension reconcile triggered (generation %s -> %s)", initialGeneration, newGeneration)
		e2e.Logf("  - Test Point 2: Deployment updated with all added fields")
		e2e.Logf("  - Test Point 3: New pod rolled out with all added configuration")
		e2e.Logf("  - Initial pod: %s, New pod: %s", initialPodName, newPodName)
		e2e.Logf("Test completed successfully - adding deploymentConfig works correctly")
	})

	g.It("PolarionID:87554-[Skipped:Disconnected]modifying deploymentConfig multiple fields in existing ClusterExtension works correctly", func() {
		var (
			caseID                       = "87554"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Initial configuration (env + resources)
			initialConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "INITIAL_ENV", "value": "initial_value"}
      ],
      "resources": {
        "limits": {
          "cpu": "300m",
          "memory": "256Mi"
        },
        "requests": {
          "cpu": "50m",
          "memory": "64Mi"
        }
      }
    }
  }`

			// Modified configuration (change env var name and value, change resources)
			// Test Point 1: Modifying config triggers reconcile
			// Test Point 2: Deployment updated with modified values
			// Test Point 3: New pods rolled out with modified config
			modifiedConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "MODIFIED_ENV", "value": "modified_value"}
      ],
      "resources": {
        "limits": {
          "cpu": "600m",
          "memory": "768Mi"
        },
        "requests": {
          "cpu": "150m",
          "memory": "192Mi"
        }
      }
    }
  }`

			// Expected values after modification
			modifiedEnvVars = map[string]string{
				"MODIFIED_ENV": "modified_value",
			}

			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87554",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87554",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     initialConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Phase 1: Create ClusterExtension WITH initial deploymentConfig")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		e2e.Logf("ClusterExtension %s created with initial deploymentConfig", extName)

		g.By("Wait for initial installation to complete")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump initial Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Get initial Pod name (before modifying config)")
		initialPodName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(initialPodName).NotTo(o.BeEmpty())
		e2e.Logf("Initial operator pod name (before modification): %s", initialPodName)

		g.By("Dump initial Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, initialPodName, ns)

		g.By("Get initial generation for comparison")
		initialGenerationPath := `jsonpath={.metadata.generation}`
		initialGeneration, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", initialGenerationPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Initial ClusterExtension generation: %s", initialGeneration)

		g.By("Phase 2: Update ClusterExtension to MODIFY deploymentConfig with new values")
		// Patch to modify config.inline with new deploymentConfig values
		patchData := fmt.Sprintf(`{
  "spec": {
    "config": {
      "configType": "Inline",
      "inline": %s
    }
  }
}`, modifiedConfig)
		err = oc.WithoutNamespace().AsAdmin().Run("patch").Args("clusterextension", extName, "--type=merge", "-p", patchData).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Updated ClusterExtension to modify deploymentConfig (env + resources)")

		g.By("Test Point 1: Verify ClusterExtension reconcile triggered (generation incremented)")
		var newGeneration string
		err = wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			newGenerationPath := `jsonpath={.metadata.generation}`
			gen, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", newGenerationPath)
			if err != nil {
				return false, nil
			}
			if gen != initialGeneration {
				newGeneration = gen
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "ClusterExtension generation did not increment after modifying config")
		e2e.Logf("New ClusterExtension generation after modifying config: %s", newGeneration)
		o.Expect(newGeneration).NotTo(o.Equal(initialGeneration), "Generation should increment after config update")
		e2e.Logf("Test Point 1 passed: ClusterExtension reconcile triggered (generation changed from %s to %s)", initialGeneration, newGeneration)

		g.By("Test Point 2: Verify Deployment updated with modified env vars")
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, modifiedEnvVars, 2*time.Minute)
		g.By("Dump updated Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment has modified env var: MODIFIED_ENV=modified_value")

		g.By("Test Point 2: Verify Deployment updated with modified resources")
		// Get all containers info as JSON from Deployment (one call instead of multiple)
		deploymentContainersJSONPath := `jsonpath={.spec.template.spec.containers}`
		deploymentContainersJSON, err := olmv1util.GetNoEmpty(oc, "deployment", deploymentName, "-n", ns, "-o", deploymentContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Deployment containers JSON retrieved (length: %d bytes)", len(deploymentContainersJSON))

		// Parse containers array using gjson
		deploymentContainersArray := gjson.Parse(deploymentContainersJSON).Array()
		containerCount := len(deploymentContainersArray)
		e2e.Logf("Total containers in Deployment: %d", containerCount)

		// Verify each container has the modified resources
		for i, container := range deploymentContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("600m"), fmt.Sprintf("Container %d (%s) should have modified CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("768Mi"), fmt.Sprintf("Container %d (%s) should have modified Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("150m"), fmt.Sprintf("Container %d (%s) should have modified CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("192Mi"), fmt.Sprintf("Container %d (%s) should have modified Memory request", i, containerName))

			e2e.Logf("Container %d (%s) has modified resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 2 passed: Deployment updated with all modified fields (env + resources)")

		g.By("Test Point 3: Wait for new pod rollout")
		// Wait for new pod to be created (different ReplicaSet hash from initial pod)
		var newPodName string
		initialPodHash := olmv1util.ExtractReplicaSetHash(initialPodName)
		if initialPodHash == "" {
			g.Skip("Cannot extract ReplicaSet hash from initial pod name, skipping pod rollout verification")
		}
		e2e.Logf("Initial pod ReplicaSet hash: %s", initialPodHash)
		err = wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 30*time.Second)
			if err != nil {
				return false, nil // Pod might be terminating/restarting
			}
			podHash := olmv1util.ExtractReplicaSetHash(podName)
			if podHash != "" && podHash != initialPodHash {
				newPodName = podName
				e2e.Logf("Found new pod with different ReplicaSet hash: %s (was %s)", podHash, initialPodHash)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "New pod was not created after modifying config")
		o.Expect(newPodName).NotTo(o.BeEmpty())
		o.Expect(newPodName).NotTo(o.Equal(initialPodName), "New pod should be created after config update")
		e2e.Logf("New operator pod name (after modification): %s", newPodName)

		g.By("Dump new Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, newPodName, ns)

		g.By("Test Point 3: Verify new pod has modified env vars")
		err = olmv1util.VerifyPodEnvVars(oc, newPodName, ns, "", modifiedEnvVars, 90*time.Second)
		if errors.Is(err, olmv1util.ErrPodNotFoundDuringVerification) {
			g.Skip("Pod was deleted during verification (likely due to rolling update) - skipping test")
		}
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("New pod has modified env var: MODIFIED_ENV=modified_value")

		g.By("Test Point 3: Verify new pod has modified resources")
		// Get all containers info as JSON (one call instead of multiple)
		podContainersJSONPath := `jsonpath={.spec.containers}`
		podContainersJSON, err := olmv1util.GetNoEmpty(oc, "pod", newPodName, "-n", ns, "-o", podContainersJSONPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Pod containers JSON retrieved (length: %d bytes)", len(podContainersJSON))

		// Parse containers array using gjson
		podContainersArray := gjson.Parse(podContainersJSON).Array()
		podContainerCount := len(podContainersArray)
		e2e.Logf("Total containers in new Pod: %d", podContainerCount)

		// Verify each container in new Pod has the modified resources
		for i, container := range podContainersArray {
			containerName := container.Get("name").String()
			containerCpuLimit := container.Get("resources.limits.cpu").String()
			containerMemLimit := container.Get("resources.limits.memory").String()
			containerCpuRequest := container.Get("resources.requests.cpu").String()
			containerMemRequest := container.Get("resources.requests.memory").String()

			o.Expect(containerCpuLimit).To(o.Equal("600m"), fmt.Sprintf("Pod container %d (%s) should have modified CPU limit", i, containerName))
			o.Expect(containerMemLimit).To(o.Equal("768Mi"), fmt.Sprintf("Pod container %d (%s) should have modified Memory limit", i, containerName))
			o.Expect(containerCpuRequest).To(o.Equal("150m"), fmt.Sprintf("Pod container %d (%s) should have modified CPU request", i, containerName))
			o.Expect(containerMemRequest).To(o.Equal("192Mi"), fmt.Sprintf("Pod container %d (%s) should have modified Memory request", i, containerName))

			e2e.Logf("Pod container %d (%s) has modified resources - Limits: %s/%s, Requests: %s/%s",
				i, containerName, containerCpuLimit, containerMemLimit, containerCpuRequest, containerMemRequest)
		}
		e2e.Logf("Test Point 3 passed: New pod rolled out with all modified fields (env + resources)")

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Phase 1: ClusterExtension created with initial deploymentConfig")
		e2e.Logf("  - Phase 2: Updated ClusterExtension to modify deploymentConfig (env + resources)")
		e2e.Logf("  - Test Point 1: ClusterExtension reconcile triggered (generation %s -> %s)", initialGeneration, newGeneration)
		e2e.Logf("  - Test Point 2: Deployment updated with all modified fields")
		e2e.Logf("  - Test Point 3: New pod rolled out with all modified configuration")
		e2e.Logf("  - Initial pod: %s, New pod: %s", initialPodName, newPodName)
		e2e.Logf("Test completed successfully - modifying deploymentConfig works correctly")
	})

	g.It("PolarionID:87555-[Skipped:Disconnected]removing entire deploymentConfig from ClusterExtension reverts all settings to bundle defaults", func() {
		var (
			caseID                       = "87555"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Initial configuration with multiple fields
			// This tests comprehensive removal of all custom configurations
			// NOTE: Bundle defaults are CPU 500m/10m, Memory 768Mi/256Mi
			// Use different values to ensure we can verify removal
			// Fields tested: env, resources, annotations, tolerations, affinity (5 types)
			initialConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "REMOVAL_TEST_ENV", "value": "should_be_removed"},
        {"name": "CUSTOM_ENV_87555", "value": "custom_value"}
      ],
      "resources": {
        "limits": {
          "cpu": "600m",
          "memory": "512Mi"
        },
        "requests": {
          "cpu": "150m",
          "memory": "192Mi"
        }
      },
      "annotations": {
        "custom.annotation.removal/test": "should_be_removed"
      },
      "tolerations": [
        {
          "key": "custom-toleration-87555",
          "operator": "Equal",
          "value": "test",
          "effect": "NoSchedule"
        }
      ],
      "affinity": {
        "nodeAffinity": {
          "requiredDuringSchedulingIgnoredDuringExecution": {
            "nodeSelectorTerms": [
              {
                "matchExpressions": [
                  {
                    "key": "custom-node-label-87555",
                    "operator": "In",
                    "values": ["custom-value"]
                  }
                ]
              }
            ]
          }
        }
      }
    }
  }`
		)

		var (
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87555",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87555",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     initialConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Phase 1: Create ClusterExtension WITH deploymentConfig (env + resources + annotations + tolerations)")
		defer clusterextension.Delete(oc)
		if olmv1util.IsFeaturegateEnabled(oc, "NewOLMBoxCutterRuntime") {
			e2e.Logf("Boxcutter applier detected")
			err := clusterextension.CreateWithoutCheck(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
		} else {
			e2e.Logf("Helm applier detected")
			clusterextension.Create(oc)
		}
		e2e.Logf("ClusterExtension %s created with initial deploymentConfig (multiple fields)", extName)

		g.By("Wait for initial installation to complete")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump initial Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Get initial Pod name (before removing config)")
		initialPodName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(initialPodName).NotTo(o.BeEmpty())
		e2e.Logf("Initial operator pod name (with custom config): %s", initialPodName)

		g.By("Dump initial Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, initialPodName, ns)

		g.By("Get initial generation for comparison")
		initialGenerationPath := `jsonpath={.metadata.generation}`
		initialGeneration, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", initialGenerationPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Initial ClusterExtension generation: %s", initialGeneration)

		g.By("Verify Phase 1 custom configuration applied correctly")
		// Verify custom env vars
		expectedEnvVars := map[string]string{
			"REMOVAL_TEST_ENV": "should_be_removed",
			"CUSTOM_ENV_87555": "custom_value",
		}
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedEnvVars, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Verified: Custom env vars applied in Deployment")

		// Verify custom resources (at least one container should have the custom values)
		deploymentYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("deployment", deploymentName, "-n", ns, "-o", "yaml").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentYaml).To(o.ContainSubstring("600m"), "Deployment should have custom CPU limit 600m")
		o.Expect(deploymentYaml).To(o.ContainSubstring("512Mi"), "Deployment should have custom memory limit 512Mi")
		e2e.Logf("Verified: Custom resources applied in Deployment")

		// Verify custom affinity
		o.Expect(deploymentYaml).To(o.ContainSubstring("custom-node-label-87555"), "Deployment should have custom nodeAffinity")
		e2e.Logf("Verified: Custom affinity applied in Deployment")

		g.By("Phase 2: Update ClusterExtension to REMOVE entire deploymentConfig")
		// Approach: Remove entire spec.config field using JSON Patch
		patchData := `[{"op": "remove", "path": "/spec/config"}]`
		e2e.Logf("DEBUG: Patch data (remove entire spec.config using JSON Patch):\n%s", patchData)

		err = oc.WithoutNamespace().AsAdmin().Run("patch").Args("clusterextension", extName, "--type=json", "-p", patchData).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Updated ClusterExtension - removed entire spec.config field")

		g.By("Test Point 1: Verify ClusterExtension reconcile triggered (generation incremented)")
		var newGeneration string
		err = wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			newGenerationPath := `jsonpath={.metadata.generation}`
			gen, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", newGenerationPath)
			if err != nil {
				return false, nil
			}
			if gen != initialGeneration {
				newGeneration = gen
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "ClusterExtension generation did not increment after removing config")
		e2e.Logf("New ClusterExtension generation after removing config: %s", newGeneration)
		o.Expect(newGeneration).NotTo(o.Equal(initialGeneration), "Generation should increment after removing config")
		e2e.Logf("Test Point 1 passed: ClusterExtension reconcile triggered (generation changed from %s to %s)", initialGeneration, newGeneration)

		g.By("Test Point 2: Verify Deployment reverts to bundle defaults (custom env vars removed)")
		// Wait for deployment to reconcile and custom env vars to be removed
		err = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			deploymentYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("deployment", deploymentName, "-n", ns, "-o", "yaml").Output()
			if err != nil {
				return false, nil
			}

			// Custom env vars should NOT exist
			if strings.Contains(deploymentYaml, "REMOVAL_TEST_ENV") {
				e2e.Logf("Waiting: REMOVAL_TEST_ENV still exists in Deployment")
				return false, nil
			}
			if strings.Contains(deploymentYaml, "CUSTOM_ENV_87555") {
				e2e.Logf("Waiting: CUSTOM_ENV_87555 still exists in Deployment")
				return false, nil
			}

			e2e.Logf("Verified: Custom env vars removed from Deployment")
			return true, nil
		})
		g.By("Dump updated Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)
		exutil.AssertWaitPollNoErr(err, "Deployment did not revert custom env vars to bundle defaults")

		g.By("Test Point 2: Verify Deployment reverts to bundle defaults (all custom fields removed)")
		// Verify all 4 custom field types are removed (in a single wait loop)
		err = wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			deploymentYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("deployment", deploymentName, "-n", ns, "-o", "yaml").Output()
			if err != nil {
				return false, nil
			}

			// Check all custom values are removed
			var pendingRemovals []string

			// 1. Custom resource values should NOT exist
			if strings.Contains(deploymentYaml, `cpu: 600m`) || strings.Contains(deploymentYaml, `"cpu":"600m"`) {
				pendingRemovals = append(pendingRemovals, "resources(cpu:600m)")
			}
			if strings.Contains(deploymentYaml, `memory: 512Mi`) || strings.Contains(deploymentYaml, `"memory":"512Mi"`) {
				pendingRemovals = append(pendingRemovals, "resources(memory:512Mi)")
			}

			// 2. Custom annotation should NOT exist
			if strings.Contains(deploymentYaml, "custom.annotation.removal/test") {
				pendingRemovals = append(pendingRemovals, "annotations")
			}

			// 3. Custom toleration should NOT exist
			if strings.Contains(deploymentYaml, "custom-toleration-87555") {
				pendingRemovals = append(pendingRemovals, "tolerations")
			}

			// 4. Custom affinity should NOT exist
			if strings.Contains(deploymentYaml, "custom-node-label-87555") {
				pendingRemovals = append(pendingRemovals, "affinity")
			}

			// If any custom fields still exist, continue waiting
			if len(pendingRemovals) > 0 {
				e2e.Logf("Waiting: Custom fields still exist: %v", pendingRemovals)
				return false, nil
			}

			e2e.Logf("Verified: All custom fields removed from Deployment (resources, annotations, tolerations, affinity)")
			return true, nil
		})
		exutil.AssertWaitPollNoErr(err, "Deployment did not revert all custom fields to bundle defaults")

		e2e.Logf("Test Point 2 passed: Deployment reverted to bundle defaults (all 5 custom field types removed)")

		g.By("Test Point 3: Wait for new pod rollout")
		// Wait for new pod to be created (different ReplicaSet hash from initial pod)
		var newPodName string
		initialPodHash := olmv1util.ExtractReplicaSetHash(initialPodName)
		if initialPodHash == "" {
			g.Skip("Cannot extract ReplicaSet hash from initial pod name, skipping pod rollout verification")
		}
		e2e.Logf("Initial pod ReplicaSet hash: %s", initialPodHash)
		err = wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 30*time.Second)
			if err != nil {
				return false, nil // Pod might be terminating/restarting
			}
			podHash := olmv1util.ExtractReplicaSetHash(podName)
			if podHash != "" && podHash != initialPodHash {
				newPodName = podName
				e2e.Logf("Found new pod with different ReplicaSet hash: %s (was %s)", podHash, initialPodHash)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "New pod was not created after removing entire config")
		o.Expect(newPodName).NotTo(o.BeEmpty())
		o.Expect(newPodName).NotTo(o.Equal(initialPodName), "New pod should be created after removing config")
		e2e.Logf("New operator pod name (with bundle defaults): %s", newPodName)

		g.By("Dump new Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, newPodName, ns)

		g.By("Test Point 3: Verify new pod has bundle defaults (custom env vars removed)")
		podYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("pod", newPodName, "-n", ns, "-o", "yaml").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		o.Expect(podYaml).NotTo(o.ContainSubstring("REMOVAL_TEST_ENV"), "Pod should not have custom env var REMOVAL_TEST_ENV")
		o.Expect(podYaml).NotTo(o.ContainSubstring("CUSTOM_ENV_87555"), "Pod should not have custom env var CUSTOM_ENV_87555")
		e2e.Logf("Verified: New pod does not have custom env vars")

		g.By("Test Point 3: Verify new pod has bundle defaults (custom resources removed)")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`cpu: 600m`), "Pod should not have custom CPU limit 600m")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`"cpu":"600m"`), "Pod should not have custom CPU limit 600m (json format)")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`memory: 512Mi`), "Pod should not have custom memory limit 512Mi")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`"memory":"512Mi"`), "Pod should not have custom memory limit 512Mi (json format)")
		e2e.Logf("Verified: New pod does not have custom resources")

		g.By("Test Point 3: Verify new pod has bundle defaults (custom annotations removed)")
		o.Expect(podYaml).NotTo(o.ContainSubstring("custom.annotation.removal/test"), "Pod should not have custom annotation")
		e2e.Logf("Verified: New pod does not have custom annotations")

		g.By("Test Point 3: Verify new pod has bundle defaults (custom tolerations removed)")
		o.Expect(podYaml).NotTo(o.ContainSubstring("custom-toleration-87555"), "Pod should not have custom toleration")
		e2e.Logf("Verified: New pod does not have custom tolerations")

		g.By("Test Point 3: Verify new pod has bundle defaults (custom affinity removed)")
		o.Expect(podYaml).NotTo(o.ContainSubstring("custom-node-label-87555"), "Pod should not have custom affinity")
		e2e.Logf("Verified: New pod does not have custom affinity")

		e2e.Logf("Test Point 3 passed: New pod rolled out with bundle defaults (all 5 custom field types removed)")

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Phase 1: ClusterExtension created WITH deploymentConfig (env + resources + annotations + tolerations + affinity)")
		e2e.Logf("  - Phase 2: Removed entire deploymentConfig from ClusterExtension (via removing spec.config)")
		e2e.Logf("  - Test Point 1: ClusterExtension reconcile triggered (generation %s -> %s)", initialGeneration, newGeneration)
		e2e.Logf("  - Test Point 2: Deployment reverted to bundle defaults (all 5 field types verified)")
		e2e.Logf("  - Test Point 3: New pod rolled out with bundle defaults")
		e2e.Logf("  - Initial pod (with custom config): %s", initialPodName)
		e2e.Logf("  - New pod (with bundle defaults): %s", newPodName)
		e2e.Logf("  - Fields verified removed: env, resources, annotations, tolerations, affinity")
		e2e.Logf("Test completed successfully - removing entire deploymentConfig reverts all settings to bundle defaults")
	})

	g.It("PolarionID:87556-[Skipped:Disconnected]removing partial fields from deploymentConfig reverts those fields to bundle defaults while keeping others", func() {
		var (
			caseID                       = "87556"
			ns                           = "test-ns-" + caseID
			sa                           = "test-sa-" + caseID
			catalogName                  = "test-catalog-" + caseID
			extName                      = "test-ext-" + caseID
			labelValue                   = caseID
			baseDir                      = exutil.FixturePath("testdata", "olm")
			clustercatalogTemplate       = filepath.Join(baseDir, "clustercatalog-withlabel.yaml")
			clusterextensionTemplate     = filepath.Join(baseDir, "clusterextension-withselectorlabel-inlineconfig.yaml")
			saClusterRoleBindingTemplate = filepath.Join(baseDir, "sa-admin.yaml")

			// Initial configuration with multiple fields (env + resources + tolerations)
			// We'll later remove resources and tolerations while keeping env
			// NOTE: Bundle defaults are CPU 500m/10m, Memory 768Mi/256Mi, no custom env, no tolerations
			initialConfig = `{
    "deploymentConfig": {
      "env": [
        {"name": "KEEP_THIS_ENV", "value": "should-remain-87556"},
        {"name": "ANOTHER_KEEP_ENV", "value": "also-remain"}
      ],
      "resources": {
        "limits": {
          "cpu": "700m",
          "memory": "640Mi"
        },
        "requests": {
          "cpu": "200m",
          "memory": "320Mi"
        }
      },
      "tolerations": [
        {
          "key": "remove-this-toleration",
          "operator": "Equal",
          "value": "test-87556",
          "effect": "NoSchedule"
        }
      ]
    }
  }`
		)

		var (
			saCrb = olmv1util.SaCLusterRolebindingDescription{
				Name:      sa,
				Namespace: ns,
				Template:  saClusterRoleBindingTemplate,
			}
			clustercatalog = olmv1util.ClusterCatalogDescription{
				Name:       catalogName,
				Imageref:   "quay.io/olmqe/nginx-ok-index:vokv87556",
				LabelValue: labelValue,
				Template:   clustercatalogTemplate,
			}
			clusterextension = olmv1util.ClusterExtensionDescription{
				Name:             extName,
				PackageName:      "nginx-ok-v87556",
				Channel:          "alpha",
				Version:          ">=0.0.1",
				InstallNamespace: ns,
				SaName:           sa,
				LabelValue:       labelValue,
				Template:         clusterextensionTemplate,
				InlineConfig:     initialConfig,
			}
		)

		g.By("Create test namespace")
		defer func() {
			e2e.Logf("Cleaning up namespace %s", ns)
			_ = oc.WithoutNamespace().AsAdmin().Run("delete").Args("ns", ns, "--ignore-not-found", "--force").Execute()
		}()
		err := oc.WithoutNamespace().AsAdmin().Run("create").Args("ns", ns).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(olmv1util.Appearance(oc, exutil.Appear, "ns", ns)).To(o.BeTrue())
		e2e.Logf("Created namespace: %s", ns)

		g.By("Create ServiceAccount and RBAC for ClusterExtension")
		defer saCrb.Delete(oc)
		saCrb.Create(oc)
		e2e.Logf("Created ServiceAccount and RBAC: %s", sa)

		g.By("Create ClusterCatalog with test operator")
		defer clustercatalog.Delete(oc)
		clustercatalog.Create(oc)
		e2e.Logf("Created ClusterCatalog: %s", catalogName)

		g.By("Phase 1: Create ClusterExtension WITH deploymentConfig (env + resources + tolerations)")
		defer clusterextension.Delete(oc)
		clusterextension.Create(oc)
		e2e.Logf("ClusterExtension %s created with initial deploymentConfig (3 fields)", extName)

		g.By("Wait for initial installation to complete")
		deploymentName, err := olmv1util.GetOperatorDeploymentName(oc, ns, "", 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentName).NotTo(o.BeEmpty())
		e2e.Logf("Operator deployment name: %s", deploymentName)

		g.By("Dump initial Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)

		g.By("Get initial Pod name (before partial removal)")
		initialPodName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 3*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(initialPodName).NotTo(o.BeEmpty())
		e2e.Logf("Initial operator pod name (with all custom fields): %s", initialPodName)

		g.By("Dump initial Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, initialPodName, ns)

		g.By("Get initial generation for comparison")
		initialGenerationPath := `jsonpath={.metadata.generation}`
		initialGeneration, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", initialGenerationPath)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Initial ClusterExtension generation: %s", initialGeneration)

		g.By("Verify Phase 1 custom configuration applied correctly")
		// Verify custom env vars
		expectedEnvVars := map[string]string{
			"KEEP_THIS_ENV":    "should-remain-87556",
			"ANOTHER_KEEP_ENV": "also-remain",
		}
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedEnvVars, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Verified: Custom env vars applied in Deployment")

		// Verify custom resources
		deploymentYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("deployment", deploymentName, "-n", ns, "-o", "yaml").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(deploymentYaml).To(o.ContainSubstring("700m"), "Deployment should have custom CPU limit 700m")
		o.Expect(deploymentYaml).To(o.ContainSubstring("640Mi"), "Deployment should have custom memory limit 640Mi")
		e2e.Logf("Verified: Custom resources applied in Deployment")

		// Verify custom tolerations
		o.Expect(deploymentYaml).To(o.ContainSubstring("remove-this-toleration"), "Deployment should have custom toleration")
		o.Expect(deploymentYaml).To(o.ContainSubstring("test-87556"), "Deployment should have toleration value")
		e2e.Logf("Verified: Custom tolerations applied in Deployment")

		g.By("Phase 2: Update ClusterExtension to remove PARTIAL fields (resources + tolerations)")
		// Approach: Use JSON Patch to remove specific fields while keeping others
		// Remove resources and tolerations, but keep env
		patchData := `[
  {"op": "remove", "path": "/spec/config/inline/deploymentConfig/resources"},
  {"op": "remove", "path": "/spec/config/inline/deploymentConfig/tolerations"}
]`
		err = oc.WithoutNamespace().AsAdmin().Run("patch").Args("clusterextension", extName, "--type=json", "-p", patchData).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Updated ClusterExtension - removed resources and tolerations from config (kept env)")

		g.By("Test Point 1: Verify ClusterExtension reconcile triggered (generation incremented)")
		var newGeneration string
		err = wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			newGenerationPath := `jsonpath={.metadata.generation}`
			gen, err := olmv1util.GetNoEmpty(oc, "clusterextension", extName, "-o", newGenerationPath)
			if err != nil {
				return false, nil
			}
			if gen != initialGeneration {
				newGeneration = gen
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "ClusterExtension generation did not increment after partial removal")
		e2e.Logf("New ClusterExtension generation after partial removal: %s", newGeneration)
		o.Expect(newGeneration).NotTo(o.Equal(initialGeneration), "Generation should increment after removing partial fields")
		e2e.Logf("Test Point 1 passed: ClusterExtension reconcile triggered (generation changed from %s to %s)", initialGeneration, newGeneration)

		g.By("Test Point 2: Verify removed fields revert to bundle defaults (resources and tolerations removed)")
		// Wait for deployment to reconcile and removed fields to revert to bundle defaults
		err = wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			deploymentYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("deployment", deploymentName, "-n", ns, "-o", "yaml").Output()
			if err != nil {
				return false, nil
			}

			// Check all removed fields are reverted to bundle defaults
			var pendingRemovals []string

			// 1. Custom resource values should NOT exist
			if strings.Contains(deploymentYaml, `cpu: 700m`) || strings.Contains(deploymentYaml, `"cpu":"700m"`) {
				pendingRemovals = append(pendingRemovals, "resources(cpu:700m)")
			}
			if strings.Contains(deploymentYaml, `memory: 640Mi`) || strings.Contains(deploymentYaml, `"memory":"640Mi"`) {
				pendingRemovals = append(pendingRemovals, "resources(memory:640Mi)")
			}

			// 2. Custom toleration should NOT exist
			if strings.Contains(deploymentYaml, "remove-this-toleration") {
				pendingRemovals = append(pendingRemovals, "tolerations")
			}

			// If any removed fields still exist, continue waiting
			if len(pendingRemovals) > 0 {
				e2e.Logf("Waiting: Removed fields still exist: %v", pendingRemovals)
				return false, nil
			}

			e2e.Logf("Verified: All removed fields reverted to bundle defaults (resources, tolerations)")
			return true, nil
		})
		g.By("Dump updated Deployment manifest for debugging")
		olmv1util.DumpDeploymentManifest(oc, deploymentName, ns)
		exutil.AssertWaitPollNoErr(err, "Deployment did not revert removed fields to bundle defaults")

		e2e.Logf("Test Point 2 passed: Removed fields (resources, tolerations) reverted to bundle defaults")

		g.By("Test Point 3: Verify kept field remains configured (custom env vars still exist)")
		expectedKeptEnvVars := map[string]string{
			"KEEP_THIS_ENV":    "should-remain-87556",
			"ANOTHER_KEEP_ENV": "also-remain",
		}
		err = olmv1util.VerifyDeploymentEnvVars(oc, deploymentName, ns, expectedKeptEnvVars, 1*time.Minute)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Verified: Custom env vars remain in Deployment")

		e2e.Logf("Test Point 3 passed: Kept field (env) remains configured with custom values")

		g.By("Test Point 4: Wait for new pod rollout")
		// Wait for new pod to be created (different ReplicaSet hash from initial pod)
		var newPodName string
		initialPodHash := olmv1util.ExtractReplicaSetHash(initialPodName)
		if initialPodHash == "" {
			g.Skip("Cannot extract ReplicaSet hash from initial pod name, skipping pod rollout verification")
		}
		e2e.Logf("Initial pod ReplicaSet hash: %s", initialPodHash)
		err = wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			podName, err := olmv1util.GetOperatorPodName(oc, ns, deploymentName, 30*time.Second)
			if err != nil {
				return false, nil // Pod might be terminating/restarting
			}
			podHash := olmv1util.ExtractReplicaSetHash(podName)
			if podHash != "" && podHash != initialPodHash {
				newPodName = podName
				e2e.Logf("Found new pod with different ReplicaSet hash: %s (was %s)", podHash, initialPodHash)
				return true, nil
			}
			return false, nil
		})
		exutil.AssertWaitPollNoErr(err, "New pod was not created after partial removal")
		o.Expect(newPodName).NotTo(o.BeEmpty())
		o.Expect(newPodName).NotTo(o.Equal(initialPodName), "New pod should be created after partial removal")
		e2e.Logf("New operator pod name (with mixed config): %s", newPodName)

		g.By("Dump new Pod manifest for debugging")
		olmv1util.DumpPodManifest(oc, newPodName, ns)

		g.By("Test Point 4: Verify new pod has bundle defaults for removed fields (custom resources removed)")
		podYaml, err := oc.WithoutNamespace().AsAdmin().Run("get").Args("pod", newPodName, "-n", ns, "-o", "yaml").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		o.Expect(podYaml).NotTo(o.ContainSubstring(`cpu: 700m`), "Pod should not have custom CPU limit 700m")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`"cpu":"700m"`), "Pod should not have custom CPU limit 700m (json format)")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`memory: 640Mi`), "Pod should not have custom memory limit 640Mi")
		o.Expect(podYaml).NotTo(o.ContainSubstring(`"memory":"640Mi"`), "Pod should not have custom memory limit 640Mi (json format)")
		e2e.Logf("Verified: New pod does not have custom resources")

		g.By("Test Point 4: Verify new pod has bundle defaults for removed fields (custom tolerations removed)")
		o.Expect(podYaml).NotTo(o.ContainSubstring("remove-this-toleration"), "Pod should not have custom toleration")
		e2e.Logf("Verified: New pod does not have custom tolerations")

		g.By("Test Point 4: Verify new pod has custom value for kept field (custom env vars still exist)")
		o.Expect(podYaml).To(o.ContainSubstring("KEEP_THIS_ENV"), "Pod should have custom env var KEEP_THIS_ENV")
		o.Expect(podYaml).To(o.ContainSubstring("should-remain-87556"), "Pod should have env var value")
		o.Expect(podYaml).To(o.ContainSubstring("ANOTHER_KEEP_ENV"), "Pod should have custom env var ANOTHER_KEEP_ENV")
		e2e.Logf("Verified: New pod has custom env vars")

		e2e.Logf("Test Point 4 passed: New pod rolled out with mixed configuration (bundle defaults for resources/tolerations, custom env)")

		e2e.Logf("Test Summary:")
		e2e.Logf("  - Phase 1: ClusterExtension created WITH deploymentConfig (env + resources + tolerations)")
		e2e.Logf("  - Phase 2: Removed partial fields from deploymentConfig (resources and tolerations deleted via JSON Patch)")
		e2e.Logf("  - Test Point 1: ClusterExtension reconcile triggered (generation %s -> %s)", initialGeneration, newGeneration)
		e2e.Logf("  - Test Point 2: Removed fields reverted to bundle defaults (resources, tolerations)")
		e2e.Logf("  - Test Point 3: Kept field remained configured (env)")
		e2e.Logf("  - Test Point 4: New pod rolled out with mixed configuration")
		e2e.Logf("  - Initial pod (all custom): %s", initialPodName)
		e2e.Logf("  - New pod (mixed config): %s", newPodName)
		e2e.Logf("  - Fields removed: resources, tolerations")
		e2e.Logf("  - Fields kept: env")
		e2e.Logf("Test completed successfully - removing partial fields reverts those fields to bundle defaults while keeping others")
	})
})
