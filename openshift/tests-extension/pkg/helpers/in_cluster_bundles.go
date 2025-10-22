package helpers

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

// NewCatalogAndClusterBundles creates bundle and catalog images in-cluster.
//
// The replacements parameter allows callers to control which template variables
// should be replaced. To have this function automatically fill in a value,
// add the key with an empty string value. For example:
//
//	replacements := map[string]string{
//	    "{{ NAMESPACE }}":       "",  // Will be auto-filled with the generated namespace name
//	    "{{ TEST-BUNDLE }}":     "",  // Will be auto-filled with the generated bundle/operator name
//	    "{{ TEST-CONTROLLER }}": "my-controller:latest",
//	}
//
// Supported auto-fill keys:
//   - "{{ NAMESPACE }}" - will be filled with the generated namespace name
//   - "{{ TEST-BUNDLE }}" - will be filled with the generated bundle/operator name
func NewCatalogAndClusterBundles(ctx SpecContext, replacements map[string]string,
	getAssetNamesCatalog func() []string, getAssetCatalog func(string) ([]byte, error),
	getAssetNamesBundle func() []string, getAssetBundle func(string) ([]byte, error),
) (string, string, string, string) {
	RequireOLMv1CapabilityOnOpenshift()
	unique := rand.String(8)
	nsName := "install-test-ns-" + unique
	ccName := "install-test-cc-" + unique
	opName := "install-test-op-" + unique
	rbName := "install-test-rb-" + unique

	By(fmt.Sprintf("setting a unique value: %q", unique))

	// Auto-fill empty values in replacements map based on key patterns
	// This allows callers to control which variables they want to use
	for key, value := range replacements {
		if value == "" {
			// Check common patterns for namespace
			if key == "{{ NAMESPACE }}" {
				replacements[key] = nsName
			}
			// Check for bundle/operator name
			if key == "{{ TEST-BUNDLE }}" {
				replacements[key] = opName
			}
			// Future: could add more auto-fill patterns here
		}
	}

	By("creating a new Namespace")
	createNamespace(nsName)

	// The builder (and deployer) service accounts are created by OpenShift itself which injects them in the NS.
	By(fmt.Sprintf("waiting for builder serviceaccount in %s", nsName))
	ExpectServiceAccountExists(ctx, "builder", nsName)

	By(fmt.Sprintf("waiting for deployer serviceaccount in %s", nsName))
	ExpectServiceAccountExists(ctx, "deployer", nsName)

	By("applying image-puller RoleBinding")
	createImagePullerRoleBinding(rbName, nsName)

	By("creating the operator BuildConfig")
	createBuildConfig(opName, nsName)

	By("creating the operator ImageStream")
	createImageStream(opName, nsName)

	By("creating the operator tarball")
	fileOperator := createTempTarBall(replacements, getAssetNamesBundle, getAssetBundle)
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
	createBuildConfig(ccName, nsName)

	By("creating the catalog ImageStream")
	createImageStream(ccName, nsName)

	By("creating the catalog tarball")
	fileCatalog := createTempTarBall(replacements, getAssetNamesCatalog, getAssetCatalog)
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
	createClusterCatalog(ccName, nsName)

	// using named returns
	return unique, nsName, ccName, opName
}

func createClusterCatalog(name, namespace string) {
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
	DeferCleanup(func() {
		By(fmt.Sprintf("deleting ClusterCatalog %q", name))
		Expect(k8sClient.Delete(context.Background(), cc)).To(Succeed())
	})
	waitForClusterCatalogServing(ctx, cc.Name)
}

func createImagePullerRoleBinding(name, namespace string) {
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
	DeferCleanup(func() {
		By(fmt.Sprintf("deleting image-puller RoleBinding %q", name))
		_ = k8sClient.Delete(ctx, rb)
	})
}

func createNamespace(namespace string) {
	ctx := context.Background()
	k8sClient := env.Get().K8sClient

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create Namespace: %q", namespace)
	DeferCleanup(func() {
		By(fmt.Sprintf("deleting Namespace %q", namespace))
		_ = k8sClient.Delete(context.Background(), ns)
	})
}

func createImageStream(name, namespace string) {
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
	DeferCleanup(func() {
		By(fmt.Sprintf("deleting ImageStream %q", name))
		_ = k8sClient.Delete(context.Background(), is)
	})
}

func createBuildConfig(name, namespace string) {
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
	DeferCleanup(func() {
		By(fmt.Sprintf("deleting BuildConfig %q", name))
		_ = k8sClient.Delete(context.Background(), bc)
	})
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
	}).WithTimeout(DefaultTimeout).WithPolling(DefaultPolling).Should(Succeed())

	DeferCleanup(func() {
		if CurrentSpecReport().Failed() {
			if CurrentSpecReport().Failed() {
				RunAndPrint(context.Background(), "get", "build", name, "-n", namespace, "-oyaml")
				RunAndPrint(context.Background(), "logs", fmt.Sprintf("build/%s", name), "-n", namespace, "--tail=200")
			}
		}
	})
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
	}).WithTimeout(DefaultTimeout).WithPolling(DefaultPolling).Should(Succeed())
}

func startBuild(args ...string) *buildv1.Build {
	output, err := RunK8sCommand(context.Background(), args...)
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

func createTempTarBall(replacements map[string]string, getAssetNames func() []string, getAsset func(string) ([]byte, error)) string {
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

	DeferCleanup(func() {
		By(fmt.Sprintf("deleting file %q", filename))
		Expect(os.Remove(filename)).To(Succeed())
	})
	return filename
}
