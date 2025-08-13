package test

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	catalogdata "github/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/catalog"
	operatordata "github/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/operator"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 operator installation", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})
	unique := rand.String(8)
	nsName := "install-test-ns-" + unique
	ccName := "install-test-cc-" + unique
	rbName := "install-test-rb-" + unique
	opName := "install-test-op-" + unique
	It("should block cluster upgrades if an incompatible operator is installed", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Requires OCP APIs: not OpenShift")
		}

		By(fmt.Sprintf("setting a unique value: %q", unique))

		testVersion := env.Get().OpenShiftVersion
		replacements := map[string]string{
			"TEST-BUNDLE": opName,
			"NAMESPACE":   nsName,
			"VERSION":     testVersion,
		}
		By(fmt.Sprintf("testing against OCP %s", testVersion))

		By("creating a new Namespace")
		nsCleanup := createNamespace(nsName)
		DeferCleanup(nsCleanup)

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

		By("waiting for InstalledOLMOperatorUpgradable to be true")
		waitForOlmUpgradeStatus(ctx, operatorv1.ConditionTrue, "")

		By("creating the ClusterExtension")
		ceName, ceCleanup := helpers.CreateClusterExtension(opName, "", nsName, unique)
		DeferCleanup(ceCleanup)
		helpers.ExpectClusterExtensionToBeInstalled(ctx, ceName)

		By("waiting for InstalledOLMOperatorUpgradable to be false")
		waitForOlmUpgradeStatus(ctx, operatorv1.ConditionFalse, ceName)

		By("waiting for ClusterOperator Upgradeable to be false")
		waitForClusterOperatorUpgradable(ctx, ceName)
	})
})

func createClusterCatalog(name, namespace string) func() {
	ctx := context.Background()
	k8sClient := env.Get().K8sClient

	cc := &olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: olmv1.ClusterCatalogSpec{
			AvailabilityMode: olmv1.AvailabilityModeAvailable,
			Priority:         0,
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
				Image: &olmv1.ImageSource{
					PollIntervalMinutes: ptr.To(int(600)),
					Ref:                 fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", namespace, name),
				},
			},
		},
	}

	Expect(k8sClient.Create(ctx, cc)).To(Succeed(), "failed to create ClusterCatalog")
	waitForClusterCatalogServing(ctx, cc.Name)
	return func() {
		By(fmt.Sprintf("deleting ClusterCatalog %q", name))
		Expect(k8sClient.Delete(context.Background(), cc)).To(Succeed())
	}
}

func createImagePullerRoleBinding(name, namespace string) func() {
	ctx := context.Background()
	k8sClient := env.Get().K8sClient

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:image-puller",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:serviceaccounts:openshift-catalogd",
			},
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:serviceaccounts:openshift-operator-controller",
			},
		},
	}
	Expect(k8sClient.Create(ctx, rb)).To(Succeed(), "failed to create image-puller RoleBinding")
	return func() {
		By(fmt.Sprintf("deleting image-puller RoleBinding %q", name))
		_ = k8sClient.Delete(ctx, rb)
	}
}

func createNamespace(namespace string) func() {
	ctx := context.Background()
	k8sClient := env.Get().K8sClient

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create Namespace: %q", namespace)
	return func() {
		By(fmt.Sprintf("deleting Namespace %q", namespace))
		_ = k8sClient.Delete(context.Background(), ns)
	}
}

func createImageStream(name, namespace string) func() {
	ctx := context.Background()
	k8sClient := env.Get().K8sClient

	is := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"name": name,
			},
		},
	}

	Expect(k8sClient.Create(ctx, is)).To(Succeed(), "failed to create ImageStream: %q", name)
	return func() {
		By(fmt.Sprintf("deleting ImageStream %q", name))
		_ = k8sClient.Delete(context.Background(), is)
	}
}

func createBuildConfig(name, namespace string) func() {
	ctx := context.Background()
	k8sClient := env.Get().K8sClient

	bc := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"name": name,
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceBinary,
				},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.DockerBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						ForcePull: true,
						From: &corev1.ObjectReference{
							Kind: "DockerImage",
							Name: "scratch",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "BUILD_LOGLEVEL",
								Value: "5",
							},
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: name + ":latest",
					},
				},
			},
		},
	}

	Expect(k8sClient.Create(ctx, bc)).To(Succeed(), "failed to create BuildConfig: %q", name)
	return func() {
		By(fmt.Sprintf("deleting BuildConfig %q", name))
		_ = k8sClient.Delete(context.Background(), bc)
	}
}

func waitForBuildToFinish(ctx SpecContext, name, namespace string) {
	const typeBuildConditionComplete = "Complete"
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		b := &buildv1.Build{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, b)
		g.Expect(err).ToNot(HaveOccurred())

		conditions := b.Status.Conditions
		var cond *buildv1.BuildCondition
		for i := range conditions {
			if conditions[i].Type == typeBuildConditionComplete {
				cond = &conditions[i]
				break
			}
		}
		g.Expect(cond).ToNot(BeNil())
		g.Expect(cond.Status).To(Equal(corev1.ConditionTrue))
	}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
}

func waitForClusterCatalogServing(ctx context.Context, name string) {
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		cc := &olmv1.ClusterCatalog{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, cc)
		g.Expect(err).ToNot(HaveOccurred())

		serving := meta.FindStatusCondition(cc.Status.Conditions, olmv1.TypeServing)
		g.Expect(serving).ToNot(BeNil())
		g.Expect(serving.Status).To(Equal(metav1.ConditionTrue))
	}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
}

func waitForOlmUpgradeStatus(ctx SpecContext, status operatorv1.ConditionStatus, name string) {
	const reasonIncompatibleOperatorsInstalled = "IncompatibleOperatorsInstalled"
	const typeInstalledOLMOperatorsUpgradeable = "InstalledOLMOperatorsUpgradeable"
	k8sClient := env.Get().K8sClient
	Eventually(func(g Gomega) {
		olm := &operatorv1.OLM{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: "cluster"}, olm)
		g.Expect(err).ToNot(HaveOccurred())

		conditions := olm.Status.Conditions
		var cond *operatorv1.OperatorCondition
		for i := range conditions {
			if conditions[i].Type == typeInstalledOLMOperatorsUpgradeable {
				cond = &conditions[i]
				break
			}
		}
		g.Expect(cond).ToNot(BeNil(), "missing condition: %q", typeInstalledOLMOperatorsUpgradeable)
		g.Expect(cond.Status).To(Equal(status))
		if status == operatorv1.ConditionFalse {
			g.Expect(name).ToNot(BeEmpty())
			g.Expect(cond.Reason).To(Equal(reasonIncompatibleOperatorsInstalled))
			g.Expect(cond.Message).To(ContainSubstring(name))
		}
	}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
}

func waitForClusterOperatorUpgradable(ctx SpecContext, name string) {
	const reasonIncompatibleOperatorsInstalled = "InstalledOLMOperators_IncompatibleOperatorsInstalled"

	Eventually(func(g Gomega) {
		k8sClient := env.Get().K8sClient
		obj := &configv1.ClusterOperator{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: "olm"}, obj)
		g.Expect(err).ToNot(HaveOccurred())

		var cond *configv1.ClusterOperatorStatusCondition
		for i, c := range obj.Status.Conditions {
			if c.Type == configv1.OperatorUpgradeable {
				cond = &obj.Status.Conditions[i]
				break
			}
		}

		g.Expect(cond).ToNot(BeNil(), "missing condition: %q", configv1.OperatorUpgradeable)
		g.Expect(cond.Status).To(Equal(configv1.ConditionFalse))
		g.Expect(cond.Reason).To(Equal(reasonIncompatibleOperatorsInstalled))
		g.Expect(cond.Message).To(ContainSubstring(name))
	}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
}

func startBuild(args ...string) *buildv1.Build {
	output, err := helpers.RunK8sCommand(context.Background(), args...)
	Expect(err).To(Succeed(), printExitError(err))

	/* The output is JSON of a build.build.openshift.io resource */
	build := &buildv1.Build{}
	Expect(json.Unmarshal(output, build)).To(Succeed(), "failed to unmarshal build")
	return build
}

func printExitError(err error) string {
	if err == nil {
		return ""
	}
	exiterr := &exec.ExitError{}
	if errors.As(err, &exiterr) {
		return fmt.Sprintf("ExitError.Stderr: %q", string(exiterr.Stderr))
	}
	return err.Error()
}

func createTempTarBall(replacements map[string]string, getAssetNames func() []string, getAsset func(string) ([]byte, error)) (string, func()) {
	file, err := os.CreateTemp("", "bundle-*.tar")
	Expect(err).To(Succeed())
	filename := file.Name()

	namesCatalog := getAssetNames()
	twCatalog := tar.NewWriter(file)
	for _, name := range namesCatalog {
		data, err := getAsset(name)
		Expect(err).To(Succeed())
		for k, v := range replacements {
			data = bytes.ReplaceAll(data, []byte(k), []byte(v))
		}
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(data)),
			Mode: 0o644,
		}
		err = twCatalog.WriteHeader(hdr)
		Expect(err).To(Succeed())
		_, err = twCatalog.Write(data)
		Expect(err).To(Succeed())
	}
	Expect(twCatalog.Close()).To(Succeed(), "failed to close tar writer for file %q", filename)
	Expect(file.Close()).To(Succeed(), "failed to close tar file %q", filename)

	return filename, func() {
		By(fmt.Sprintf("deleting file %q", filename))
		Expect(os.Remove(filename)).To(Succeed())
	}
}
