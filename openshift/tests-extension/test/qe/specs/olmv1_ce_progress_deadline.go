package specs

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	buildv1 "github.com/openshift/api/build/v1"
	"github.com/openshift/api/features"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/openshift/origin/test/extended/util/image"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM][OCPFeatureGate:NewOLMBoxCutterRuntime] clusterextension progress deadline", g.Label("NonHyperShiftHOST", "ReleaseGate"), func() {
	defer g.GinkgoRecover()

	var oc = exutil.NewCLIWithoutNamespace("default")

	g.BeforeEach(func(ctx g.SpecContext) {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
		helpers.RequireFeatureGateEnabled(features.FeatureGateNewOLMBoxCutterRuntime)
		helpers.RequireImageRegistry(ctx)
	})

	g.AfterEach(func() {
		if g.CurrentSpecReport().Failed() {
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), "")
		}
	})

	g.It("PolarionID:88331-[OTP]A ClusterExtension is created and a persistent error prevents it from being fully rolled out", func(ctx g.SpecContext) {
		const caseID = "88331"
		fixture := newRolloutFailureFixture(ctx, oc, caseID, []rolloutFailureBundle{
			{Version: "1.0.2", ControllerImage: "wrong/image"},
		})

		g.By("creating a ClusterExtension with a 10-minute progress deadline for the failing bundle")
		ce := fixture.newClusterExtension("test-ce-install-timeout-"+caseID, "1.0.2", "olm-sa", ptr.To(int32(10)))
		o.Expect(env.Get().K8sClient.Create(ctx, ce)).To(o.Succeed(), "failed to create ClusterExtension")
		g.DeferCleanup(deleteObject, ce)

		g.By("waiting for the first ClusterObjectSet to report ProgressDeadlineExceeded")
		expectClusterObjectSetCondition(ctx, ce.Name+"-1", olmv1.TypeProgressing, metav1.ConditionFalse, olmv1.ReasonProgressDeadlineExceeded)

		g.By("verifying the ClusterExtension reports ProgressDeadlineExceeded")
		expectClusterExtensionCondition(ctx, ce.Name, olmv1.TypeProgressing, metav1.ConditionFalse, olmv1.ReasonProgressDeadlineExceeded, "Revision has not rolled out for 10 minute(s).")
	})

	g.It("PolarionID:88332-[OTP]A ClusterExtension is being upgraded and a persistent error prevents it from being fully rolled out", func(ctx g.SpecContext) {
		const caseID = "88332"
		fixture := newRolloutFailureFixture(ctx, oc, caseID, []rolloutFailureBundle{
			{Version: "1.0.0", ControllerImage: image.ShellImage()},
			{Version: "1.0.2", ControllerImage: "wrong/image", Replaces: "1.0.0"},
		})

		g.By("creating a ClusterExtension on a healthy initial version")
		ce := fixture.newClusterExtension("test-ce-upgrade-failure-"+caseID, "1.0.0", "olm-sa", nil)
		ce.Spec.Source.Catalog.UpgradeConstraintPolicy = olmv1.UpgradeConstraintPolicySelfCertified
		o.Expect(env.Get().K8sClient.Create(ctx, ce)).To(o.Succeed(), "failed to create ClusterExtension")
		g.DeferCleanup(deleteObject, ce)

		g.By("waiting for the initial installation to complete")
		expectClusterExtensionCondition(ctx, ce.Name, olmv1.TypeInstalled, metav1.ConditionTrue, "", "")
		expectClusterExtensionCondition(ctx, ce.Name, olmv1.TypeProgressing, metav1.ConditionTrue, olmv1.ReasonSucceeded, "")

		g.By("updating the ClusterExtension to a version with a persistent rollout error")
		patch := []byte(`{"spec":{"source":{"catalog":{"version":"1.0.2"}}}}`)
		o.Expect(env.Get().K8sClient.Patch(ctx, ce, client.RawPatch(types.MergePatchType, patch))).To(o.Succeed(), "failed to patch ClusterExtension")

		g.By("verifying both revisions remain active during the failed upgrade")
		expectActiveRevisions(ctx, ce.Name, ce.Name+"-1", ce.Name+"-2")

		g.By("verifying the new revision is still rolling out")
		expectClusterObjectSetCondition(ctx, ce.Name+"-2", olmv1.TypeProgressing, metav1.ConditionTrue, olmv1.ReasonRollingOut)

		g.By("verifying the new revision reports probe failure")
		expectClusterObjectSetCondition(ctx, ce.Name+"-2", olmv1.ClusterObjectSetTypeAvailable, metav1.ConditionFalse, olmv1.ClusterObjectSetReasonProbeFailure)
	})
})

type rolloutFailureBundle struct {
	Version         string
	ControllerImage string
	Replaces        string
}

type rolloutFailureFixture struct {
	Namespace   string
	CatalogName string
}

func newRolloutFailureFixture(ctx g.SpecContext, oc *exutil.CLI, caseID string, bundles []rolloutFailureBundle) rolloutFailureFixture {
	k8sClient := env.Get().K8sClient
	namespace := "ns-" + caseID
	catalogName := "test-catalog-" + caseID

	g.By("creating a test namespace")
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	o.Expect(k8sClient.Create(ctx, ns)).To(o.Succeed(), "failed to create Namespace")
	g.DeferCleanup(deleteObject, ns)
	helpers.ExpectServiceAccountExists(ctx, "builder", namespace)
	helpers.ExpectServiceAccountExists(ctx, "deployer", namespace)

	g.By("creating the ServiceAccount and permissions for the ClusterExtension")
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "olm-sa", Namespace: namespace}}
	o.Expect(k8sClient.Create(ctx, sa)).To(o.Succeed(), "failed to create ServiceAccount")
	g.DeferCleanup(deleteObject, sa)

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "olm-sa-" + caseID + "-cluster-admin"},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      sa.Name,
			Namespace: namespace,
		}},
	}
	o.Expect(k8sClient.Create(ctx, crb)).To(o.Succeed(), "failed to create ClusterRoleBinding")
	g.DeferCleanup(deleteObject, crb)

	g.By("allowing catalogd and operator-controller to pull test images")
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "image-puller-" + caseID, Namespace: namespace},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:image-puller",
		},
		Subjects: []rbacv1.Subject{
			{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "system:serviceaccounts:openshift-catalogd"},
			{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "system:serviceaccounts:openshift-operator-controller"},
		},
	}
	o.Expect(k8sClient.Create(ctx, rb)).To(o.Succeed(), "failed to create image-puller RoleBinding")
	g.DeferCleanup(deleteObject, rb)

	bundleRefs := make(map[string]string, len(bundles))
	for _, bundle := range bundles {
		imageName := fmt.Sprintf("test-bundle-%s-%s", caseID, strings.ReplaceAll(bundle.Version, ".", "-"))
		bundleRefs[bundle.Version] = fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", namespace, imageName)
		buildImage(ctx, oc, namespace, imageName, bundleImageFiles(caseID, bundle))
	}

	g.By("building the catalog image")
	buildImage(ctx, oc, namespace, catalogName, catalogImageFiles(caseID, bundles, bundleRefs))

	g.By("creating the ClusterCatalog")
	catalog := &olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: catalogName},
		Spec: olmv1.ClusterCatalogSpec{
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
				Image: &olmv1.ImageSource{
					PollIntervalMinutes: ptr.To(600),
					Ref:                 fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", namespace, catalogName),
				},
			},
		},
	}
	o.Expect(k8sClient.Create(ctx, catalog)).To(o.Succeed(), "failed to create ClusterCatalog")
	g.DeferCleanup(deleteObject, catalog)
	expectClusterCatalogServing(ctx, catalogName)

	return rolloutFailureFixture{
		Namespace:   namespace,
		CatalogName: catalogName,
	}
}

func (f rolloutFailureFixture) newClusterExtension(name, version, serviceAccount string, progressDeadlineMinutes *int32) *olmv1.ClusterExtension {
	ce := &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: olmv1.ClusterExtensionSpec{
			Namespace: f.Namespace,
			ServiceAccount: olmv1.ServiceAccountReference{
				Name: serviceAccount,
			},
			Source: olmv1.SourceConfig{
				SourceType: olmv1.SourceTypeCatalog,
				Catalog: &olmv1.CatalogFilter{
					PackageName: "test",
					Version:     version,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"olm.operatorframework.io/metadata.name": f.CatalogName,
						},
					},
				},
			},
		},
	}
	if progressDeadlineMinutes != nil {
		ce.Spec.ProgressDeadlineMinutes = *progressDeadlineMinutes
	}
	return ce
}

func buildImage(ctx g.SpecContext, oc *exutil.CLI, namespace, name string, files map[string][]byte) {
	k8sClient := env.Get().K8sClient

	imageStream := &imagev1.ImageStream{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	o.Expect(k8sClient.Create(ctx, imageStream)).To(o.Succeed(), "failed to create ImageStream %s", name)
	g.DeferCleanup(deleteObject, imageStream)

	buildConfig := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{Type: buildv1.BuildSourceBinary},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.DockerBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						ForcePull: true,
						From: &corev1.ObjectReference{
							Kind: "DockerImage",
							Name: "scratch",
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
	o.Expect(k8sClient.Create(ctx, buildConfig)).To(o.Succeed(), "failed to create BuildConfig %s", name)
	g.DeferCleanup(deleteObject, buildConfig)

	archive := createBuildArchive(files)
	g.DeferCleanup(func() {
		o.Expect(os.Remove(archive)).To(o.Succeed(), "failed to delete build archive %s", archive)
	})

	output, err := oc.AsAdmin().WithoutNamespace().Run("start-build").Args(name, "-n", namespace, "--from-archive="+archive, "--wait").Output()
	o.Expect(err).NotTo(o.HaveOccurred(), "failed to build image %s: %s", name, output)
}

func createBuildArchive(files map[string][]byte) string {
	file, err := os.CreateTemp("", "rollout-failure-build-*.tar")
	o.Expect(err).NotTo(o.HaveOccurred(), "failed to create build archive")
	defer func() {
		o.Expect(file.Close()).To(o.Succeed(), "failed to close build archive")
	}()

	tw := tar.NewWriter(file)
	defer func() {
		o.Expect(tw.Close()).To(o.Succeed(), "failed to close tar writer")
	}()

	for name, data := range files {
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(data)),
			Mode: 0o644,
		}
		o.Expect(tw.WriteHeader(hdr)).To(o.Succeed(), "failed to write tar header for %s", name)
		_, err := tw.Write(data)
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to write tar content for %s", name)
	}
	return file.Name()
}

func bundleImageFiles(caseID string, bundle rolloutFailureBundle) map[string][]byte {
	replacements := map[string]string{
		"{{ CASE_ID }}":          caseID,
		"{{ VERSION }}":          bundle.Version,
		"{{ CONTROLLER_IMAGE }}": bundle.ControllerImage,
	}
	files := map[string]string{
		"Dockerfile":                                bundleDockerfile,
		"metadata/annotations.yaml":                 bundleAnnotations,
		"metadata/properties.yaml":                  bundleProperties,
		"manifests/test.clusterserviceversion.yaml": bundleCSV,
		"manifests/test-script.configmap.yaml":      bundleScriptConfigMap,
	}
	out := make(map[string][]byte, len(files))
	for name, content := range files {
		out[name] = []byte(replaceAll(content, replacements))
	}
	return out
}

func catalogImageFiles(caseID string, bundles []rolloutFailureBundle, bundleRefs map[string]string) map[string][]byte {
	var fbc strings.Builder
	writeCatalogEntry := func(entry any) {
		data, err := json.Marshal(entry)
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to marshal catalog metadata")
		_, err = fbc.Write(data)
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to write catalog metadata")
		o.Expect(fbc.WriteByte('\n')).To(o.Succeed(), "failed to terminate catalog metadata entry")
	}

	writeCatalogEntry(map[string]any{
		"schema":         "olm.package",
		"name":           "test",
		"defaultChannel": "alpha",
	})

	for _, bundle := range bundles {
		writeCatalogEntry(map[string]any{
			"schema":  "olm.bundle",
			"name":    "test.v" + bundle.Version,
			"package": "test",
			"image":   bundleRefs[bundle.Version],
			"properties": []map[string]any{{
				"type": "olm.package",
				"value": map[string]string{
					"packageName": "test",
					"version":     bundle.Version,
				},
			}},
		})
	}

	entries := make([]map[string]string, 0, len(bundles))
	for _, bundle := range bundles {
		entry := map[string]string{"name": "test.v" + bundle.Version}
		if bundle.Replaces == "" {
			entries = append(entries, entry)
			continue
		}
		entry["replaces"] = "test.v" + bundle.Replaces
		entries = append(entries, entry)
	}
	writeCatalogEntry(map[string]any{
		"schema":  "olm.channel",
		"name":    "alpha",
		"package": "test",
		"entries": entries,
	})

	return map[string][]byte{
		"Dockerfile":                          []byte(catalogDockerfile),
		"configs/.indexignore":                []byte("..*\n"),
		"configs/catalog-" + caseID + ".yaml": []byte(fbc.String()),
	}
}

func replaceAll(input string, replacements map[string]string) string {
	for old, newValue := range replacements {
		input = strings.ReplaceAll(input, old, newValue)
	}
	return input
}

func expectClusterCatalogServing(ctx context.Context, name string) {
	eventually(func(g o.Gomega) {
		catalog := &olmv1.ClusterCatalog{}
		err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: name}, catalog)
		g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterCatalog")

		serving := meta.FindStatusCondition(catalog.Status.Conditions, olmv1.TypeServing)
		g.Expect(serving).NotTo(o.BeNil(), "Serving condition not found")
		g.Expect(serving.Status).To(o.Equal(metav1.ConditionTrue), "Serving condition should be True")
	}, 5*time.Minute)
}

func expectClusterExtensionCondition(ctx context.Context, name, conditionType string, status metav1.ConditionStatus, reason, messageSubstring string) {
	eventually(func(g o.Gomega) {
		ce := &olmv1.ClusterExtension{}
		err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
		g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterExtension")

		condition := meta.FindStatusCondition(ce.Status.Conditions, conditionType)
		g.Expect(condition).NotTo(o.BeNil(), "%s condition not found", conditionType)
		g.Expect(condition.Status).To(o.Equal(status), "%s status mismatch", conditionType)
		if reason != "" {
			g.Expect(condition.Reason).To(o.Equal(reason), "%s reason mismatch", conditionType)
		}
		if messageSubstring != "" {
			g.Expect(condition.Message).To(o.ContainSubstring(messageSubstring), "%s message mismatch", conditionType)
		}
	}, 3*time.Minute)
}

func expectClusterObjectSetCondition(ctx context.Context, name, conditionType string, status metav1.ConditionStatus, reason string) {
	eventually(func(g o.Gomega) {
		cos := &olmv1.ClusterObjectSet{}
		err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: name}, cos)
		g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterObjectSet")

		condition := meta.FindStatusCondition(cos.Status.Conditions, conditionType)
		g.Expect(condition).NotTo(o.BeNil(), "%s condition not found", conditionType)
		g.Expect(condition.Status).To(o.Equal(status), "%s status mismatch", conditionType)
		g.Expect(condition.Reason).To(o.Equal(reason), "%s reason mismatch", conditionType)
	}, 12*time.Minute)
}

func expectActiveRevisions(ctx context.Context, name string, expected ...string) {
	eventually(func(g o.Gomega) {
		ce := &olmv1.ClusterExtension{}
		err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: name}, ce)
		g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterExtension")

		actual := make([]string, 0, len(ce.Status.ActiveRevisions))
		for _, revision := range ce.Status.ActiveRevisions {
			actual = append(actual, revision.Name)
		}
		g.Expect(actual).To(o.ConsistOf(expected), "active revisions mismatch")
	}, 3*time.Minute)
}

func eventually(callback func(o.Gomega), timeout time.Duration) {
	o.Eventually(callback).WithTimeout(timeout).WithPolling(helpers.DefaultPolling).Should(o.Succeed())
}

func deleteObject(obj client.Object) {
	err := env.Get().K8sClient.Delete(context.Background(), obj)
	if err != nil && !errors.IsNotFound(err) {
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to delete %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}
}

const bundleDockerfile = `FROM scratch

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=test
LABEL operators.operatorframework.io.bundle.channels.v1=alpha

COPY manifests /manifests/
COPY metadata /metadata/
`

const bundleAnnotations = `annotations:
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.manifests.v1: manifests/
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  operators.operatorframework.io.bundle.package.v1: test
  operators.operatorframework.io.bundle.channels.v1: alpha
`

const bundleProperties = `properties: []`

const bundleScriptConfigMap = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-script-{{ CASE_ID }}
data:
  httpd.sh: |-
    #!/bin/sh
    mkdir -p /tmp/www
    echo true > /tmp/www/started
    echo true > /tmp/www/ready
    echo true > /tmp/www/live
    python3 -m http.server 8081 --bind :: --directory /tmp/www
`

const bundleCSV = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: test.v{{ VERSION }}
  annotations:
    capabilities: Basic Install
spec:
  displayName: test
  description: test
  installModes:
  - supported: true
    type: AllNamespaces
  - supported: true
    type: OwnNamespace
  - supported: true
    type: SingleNamespace
  - supported: false
    type: MultiNamespace
  install:
    strategy: deployment
    spec:
      deployments:
      - name: test-controller-manager-{{ CASE_ID }}
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: test-controller-manager-{{ CASE_ID }}
          template:
            metadata:
              labels:
                app: test-controller-manager-{{ CASE_ID }}
            spec:
              serviceAccountName: test-controller-manager-{{ CASE_ID }}
              securityContext:
                runAsNonRoot: true
              volumes:
              - name: scripts
                configMap:
                  name: test-script-{{ CASE_ID }}
                  defaultMode: 0755
              containers:
              - name: manager
                image: "{{ CONTROLLER_IMAGE }}"
                command:
                - /scripts/httpd.sh
                ports:
                - containerPort: 8080
                  name: http
                volumeMounts:
                - name: scripts
                  mountPath: /scripts
                  readOnly: true
                startupProbe:
                  httpGet:
                    path: /started
                    port: 8081
                  failureThreshold: 30
                  periodSeconds: 10
                readinessProbe:
                  httpGet:
                    path: /ready
                    port: 8081
                  initialDelaySeconds: 5
                  periodSeconds: 10
                livenessProbe:
                  httpGet:
                    path: /live
                    port: 8081
                  initialDelaySeconds: 15
                  periodSeconds: 20
                securityContext:
                  allowPrivilegeEscalation: false
                  capabilities:
                    drop:
                    - ALL
      permissions:
      - serviceAccountName: test-controller-manager-{{ CASE_ID }}
        rules:
        - apiGroups:
          - ""
          resources:
          - configmaps
          verbs:
          - get
          - list
          - watch
          - create
          - update
          - patch
          - delete
  version: {{ VERSION }}
`

const catalogDockerfile = `FROM scratch
ADD configs /configs
LABEL operators.operatorframework.io.index.configs.v1=/configs
`
