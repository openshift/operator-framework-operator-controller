package test

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"github.com/openshift/origin/test/extended/util/image"
	corev1 "k8s.io/api/core/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	catalogdata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/catalog"
	operatordata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/operator"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs", func() {
	const olmv1GroupName = "olm.operatorframework.io"
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})

	It("should be installed", func(ctx SpecContext) {
		cfg := env.Get().RestCfg
		crds := []struct {
			group   string
			version []string
			plural  string
		}{
			{olmv1GroupName, []string{"v1"}, "clusterextensions"},
			{olmv1GroupName, []string{"v1"}, "clustercatalogs"},
		}

		client, err := apiextclient.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		for _, crd := range crds {
			By(fmt.Sprintf("verifying CRD %s.%s", crd.plural, crd.group))
			crdObj, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, fmt.Sprintf("%s.%s",
				crd.plural, crd.group), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			for _, v := range crd.version {
				found := false
				for _, ver := range crdObj.Spec.Versions {
					if ver.Name == v {
						Expect(ver.Served).To(BeTrue(), "version %s not served", v)
						Expect(ver.Storage).To(BeTrue(), "version %s not used for storage", v)
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Expected version %q in CRD %s.%s", v, crd.plural, crd.group))
			}
		}
	})
})

// Keeping this test as skip:disconnected, so we can attempt to install a "real" operator, rather than a generated one
// There is an equivalent in-cluster positive test below
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation", func() {
	var (
		namespace string
		k8sClient client.Client
	)

	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		k8sClient = env.Get().K8sClient
		namespace = "install-test-ns-" + rand.String(4)
		By(fmt.Sprintf("creating namespace %s", namespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(context.Background(), ns)).To(Succeed(), "failed to create test namespace")
		DeferCleanup(func() {
			By(fmt.Sprintf("deleting namespace %s", namespace))
			_ = k8sClient.Delete(context.Background(), ns)
		})
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), namespace)
		}
	})

	It("should install an openshift catalog cluster extension", Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation should install an openshift catalog cluster extension"), func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP Catalogs: not OpenShift")
		}

		By("ensuring no ClusterExtension and CRD for quay-operator")
		helpers.EnsureCleanupClusterExtension(context.Background(), "quay-operator", "quayregistries.quay.redhat.com")

		By("applying the ClusterExtension resource")
		name, cleanup := helpers.CreateClusterExtension("quay-operator", "3.13.0", namespace, "")
		DeferCleanup(cleanup)

		By("waiting for the quay-operator ClusterExtension to be installed")
		helpers.ExpectClusterExtensionToBeInstalled(ctx, name)
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 operator installation", func() {
	var unique, nsName, ccName, rbName, opName string
	BeforeEach(func(ctx SpecContext) {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		unique = rand.String(8)
		nsName = "install-test-ns-" + unique
		ccName = "install-test-cc-" + unique
		rbName = "install-test-rb-" + unique
		opName = "install-test-op-" + unique

		By(fmt.Sprintf("setting a unique value: %q", unique))

		testVersion := env.Get().OpenShiftVersion
		replacements := map[string]string{
			"{{ TEST-BUNDLE }}": opName,
			"{{ NAMESPACE }}":   nsName,
			"{{ VERSION }}":     testVersion,

			// Using the shell image provided by origin as the controller image.
			// The image is mirrored into disconnected environments for testing.
			"{{ TEST-CONTROLLER }}": image.ShellImage(),
		}

		By("creating a new Namespace")
		nsCleanup := createNamespace(nsName)
		DeferCleanup(nsCleanup)

		// The builder (and deployer) service accounts are created by OpenShift itself which injects them in the NS.
		By(fmt.Sprintf("waiting for builder serviceaccount in %s", nsName))
		helpers.ExpectServiceAccountExists(ctx, "builder", nsName)

		By(fmt.Sprintf("waiting for deployer serviceaccount in %s", nsName))
		helpers.ExpectServiceAccountExists(ctx, "deployer", nsName)

		By("applying image-puller RoleBinding")
		rbCleanup := createImagePullerRoleBinding(rbName, nsName)
		DeferCleanup(rbCleanup)

		By("creating the operator BuildConfig")
		bcCleanup := createBuildConfig(opName, nsName)
		DeferCleanup(bcCleanup)

		By("creating the operator ImageStream")
		isCleanup := createImageStream(opName, nsName)
		DeferCleanup(isCleanup)

		By("creating the operator tarball")
		fileOperator, fileCleanup := createTempTarBall(replacements, operatordata.AssetNames, operatordata.Asset)
		DeferCleanup(fileCleanup)
		By(fmt.Sprintf("created operator tarball %q", fileOperator))

		By("starting the operator build via RAW URL")
		opArgs := []string{
			"create",
			"--raw",
			fmt.Sprintf(
				"/apis/build.openshift.io/v1/namespaces/%s/buildconfigs/%s/instantiatebinary?name=%s&namespace=%s",
				nsName, opName, opName, nsName,
			),
			"-f",
			fileOperator,
		}
		buildOperator := startBuild(opArgs...)

		By(fmt.Sprintf("waiting for the build %q to finish", buildOperator.Name))
		waitForBuildToFinish(ctx, buildOperator.Name, nsName)

		By("creating the catalog BuildConfig")
		bcCleanup = createBuildConfig(ccName, nsName)
		DeferCleanup(bcCleanup)

		By("creating the catalog ImageStream")
		isCleanup = createImageStream(ccName, nsName)
		DeferCleanup(isCleanup)

		By("creating the catalog tarball")
		fileCatalog, fileCleanup := createTempTarBall(replacements, catalogdata.AssetNames, catalogdata.Asset)
		DeferCleanup(fileCleanup)
		By(fmt.Sprintf("created catalog tarball %q", fileCatalog))

		By("starting the catalog build via RAW URL")
		catalogArgs := []string{
			"create",
			"--raw",
			fmt.Sprintf(
				"/apis/build.openshift.io/v1/namespaces/%s/buildconfigs/%s/instantiatebinary?name=%s&namespace=%s",
				nsName, ccName, ccName, nsName,
			),
			"-f",
			fileCatalog,
		}
		buildCatalog := startBuild(catalogArgs...)

		By(fmt.Sprintf("waiting for the build %q to finish", buildCatalog.Name))
		waitForBuildToFinish(ctx, buildCatalog.Name, nsName)

		By("creating the ClusterCatalog")
		ccCleanup := createClusterCatalog(ccName, nsName)
		DeferCleanup(ccCleanup)
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), nsName)
		}
	})

	It("should install a cluster extension",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation should install a cluster extension"), func(ctx SpecContext) {
			if !env.Get().IsOpenShift {
				Skip("Requires OCP Catalogs: not OpenShift")
			}

			By("ensuring no ClusterExtension and CRD for the operator")
			helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

			By("applying the ClusterExtension resource")
			name, cleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
			DeferCleanup(cleanup)

			By("waiting for the ClusterExtension to be installed")
			helpers.ExpectClusterExtensionToBeInstalled(ctx, name)
		})

	It("should fail to install a non-existing cluster extension",
		Label("original-name:[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation should fail to install a non-existing cluster extension"), func(ctx SpecContext) {
			if !env.Get().IsOpenShift {
				Skip("Requires OCP APIs: not OpenShift")
			}

			By("ensuring no ClusterExtension and CRD for non-existing operator")
			helpers.EnsureCleanupClusterExtension(context.Background(), "does-not-exist", "") // No CRD expected for non-existing operator

			By("applying the ClusterExtension resource")
			name, cleanup := helpers.CreateClusterExtension("does-not-exist", "99.99.99", nsName, unique, helpers.WithCatalogNameSelector(ccName))
			DeferCleanup(cleanup)

			By("waiting for the ClusterExtension to exist")
			ce := &olmv1.ClusterExtension{}
			Eventually(func() error {
				return env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
			}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("waiting up to 2 minutes for ClusterExtension to report failure")
			Eventually(func(g Gomega) {
				k8sClient := env.Get().K8sClient
				err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
				g.Expect(err).ToNot(HaveOccurred())

				progressing := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeProgressing)
				g.Expect(progressing).ToNot(BeNil())
				g.Expect(progressing.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(progressing.Reason).To(Equal("Retrying"))
				g.Expect(progressing.Message).To(ContainSubstring(`no bundles found`))

				installed := meta.FindStatusCondition(ce.Status.Conditions, olmv1.TypeInstalled)
				g.Expect(installed).ToNot(BeNil())
				g.Expect(installed.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(installed.Reason).To(Equal("Failed"))
				g.Expect(installed.Message).To(Equal("No bundle installed"))
			}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		})
})
