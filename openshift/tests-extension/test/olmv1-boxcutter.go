package test

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"github.com/openshift/api/features"
	"github.com/openshift/origin/test/extended/util/image"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	catalogdata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/catalog"
	operatordata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/operator"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMBoxCutterRuntime] OLMv1 Boxcutter runtime", func() {
	var unique, nsName, ccName, opName string

	BeforeEach(func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OpenShift for Boxcutter runtime tests")
		}
		helpers.RequireFeatureGateEnabled(features.FeatureGateNewOLMBoxCutterRuntime)
		helpers.RequireImageRegistry(ctx)

		testVersion := env.Get().OpenShiftVersion
		replacements := map[string]string{
			"{{ TEST-BUNDLE }}":     "",
			"{{ NAMESPACE }}":       "",
			"{{ VERSION }}":         testVersion,
			"{{ TEST-CONTROLLER }}": image.ShellImage(),
		}
		unique, nsName, ccName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			catalogdata.AssetNames, catalogdata.Asset,
			operatordata.AssetNames, operatordata.Asset,
		)
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), nsName)
		}
	})

	It("should install a cluster extension via the Boxcutter runtime", func(ctx SpecContext) {
		By("ensuring no ClusterExtension for the operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to be installed with availability probes passing")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, name)
	})

	It("should report active revisions in the ClusterExtension status after installation", func(ctx SpecContext) {
		By("ensuring no ClusterExtension for the operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, name)

		By("verifying active revisions are reported in the ClusterExtension status")
		k8sClient := env.Get().K8sClient
		Eventually(func(g Gomega) {
			var ext olmv1.ClusterExtension
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(ext.Status.ActiveRevisions).ToNot(BeEmpty(),
				"expected at least one active revision after installation")
			g.Expect(ext.Status.ActiveRevisions[0].Name).ToNot(BeEmpty(),
				"expected active revision name to be non-empty")
		}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
	})

	It("should label managed resources with OLM ownership metadata", func(ctx SpecContext) {
		By("ensuring no ClusterExtension for the operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, name)

		By("verifying the operator Deployment carries OLM ownership labels")
		k8sClient := env.Get().K8sClient
		Eventually(func(g Gomega) {
			deployments := &appsv1.DeploymentList{}
			err := k8sClient.List(ctx, deployments, client.InNamespace(nsName))
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(deployments.Items).ToNot(BeEmpty(),
				"expected at least one Deployment in namespace %s", nsName)

			found := false
			for _, d := range deployments.Items {
				labels := d.GetLabels()
				if labels["olm.operatorframework.io/owner-kind"] == "ClusterExtension" &&
					labels["olm.operatorframework.io/owner-name"] == name {
					found = true
					break
				}
			}
			g.Expect(found).To(BeTrue(),
				"expected a Deployment labeled with OLM ownership for ClusterExtension %q", name)
		}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
	})

	It("should clean up managed resources when a cluster extension is deleted", func(ctx SpecContext) {
		By("ensuring no ClusterExtension for the operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
		DeferCleanup(cleanup)

		By("waiting for the ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, name)

		k8sClient := env.Get().K8sClient

		By("deleting the ClusterExtension")
		ce := &olmv1.ClusterExtension{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)).To(Succeed())
		Expect(k8sClient.Delete(ctx, ce)).To(Succeed())

		By("waiting for the ClusterExtension to be fully deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &olmv1.ClusterExtension{})
			return errors.IsNotFound(err)
		}).WithTimeout(helpers.InstallTimeout).WithPolling(helpers.DefaultPolling).Should(BeTrue(),
			"ClusterExtension %q was not deleted within timeout", name)

		By("verifying managed Deployments are removed from the namespace")
		Eventually(func(g Gomega) {
			deployments := &appsv1.DeploymentList{}
			err := k8sClient.List(ctx, deployments, client.InNamespace(nsName),
				client.MatchingLabels{"olm.operatorframework.io/owner-name": name})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(deployments.Items).To(BeEmpty(),
				"expected no Deployments owned by ClusterExtension %q after deletion", name)
		}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
	})

	It("should successfully reinstall a cluster extension after deletion", func(ctx SpecContext) {
		By("ensuring no ClusterExtension for the operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

		By("applying the ClusterExtension resource for the first time")
		firstName, firstCleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))

		By("waiting for the first installation to complete")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, firstName)

		k8sClient := env.Get().K8sClient

		By("deleting the first ClusterExtension and associated resources")
		firstCleanup()

		By("waiting for the first ClusterExtension to be fully deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: firstName}, &olmv1.ClusterExtension{})
			return errors.IsNotFound(err)
		}).WithTimeout(helpers.InstallTimeout).WithPolling(helpers.DefaultPolling).Should(BeTrue(),
			"first ClusterExtension %q was not deleted within timeout", firstName)

		By("reinstalling the ClusterExtension with a fresh identity")
		reinstallUnique := rand.String(4)
		secondName, secondCleanup := helpers.CreateClusterExtension(opName, "", nsName, reinstallUnique, helpers.WithCatalogNameSelector(ccName))
		DeferCleanup(secondCleanup)

		By("waiting for the second installation to complete")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, secondName)
	})
})
