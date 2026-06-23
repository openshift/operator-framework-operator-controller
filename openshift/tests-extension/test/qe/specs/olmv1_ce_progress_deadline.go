package specs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"github.com/openshift/api/features"
	"github.com/openshift/origin/test/extended/util/image"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	catalogbindata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/catalog"
	operatorbindata "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/operator"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

var _ = g.Describe("[sig-olmv1][Jira:OLM] clusterextension progress deadline", g.Label("NonHyperShiftHOST"), func() {
	var oc = exutil.NewCLIWithoutNamespace("default")
	var fixture rolloutFailureFixture

	g.BeforeEach(func(ctx g.SpecContext) {
		exutil.SkipMicroshift(oc)
		exutil.SkipNoOLMv1Core(oc)
		helpers.RequireFeatureGateEnabled(features.FeatureGateNewOLMBoxCutterRuntime)
		helpers.RequireImageRegistry(ctx)
	})

	g.AfterEach(func() {
		if g.CurrentSpecReport().Failed() {
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), fixture.Namespace)
		}
	})

	g.It("PolarionID:88331-[OTP][Slow][OCPFeatureGate:NewOLMBoxCutterRuntime]A ClusterExtension is created and a persistent error prevents it from being fully rolled out", func(ctx g.SpecContext) {
		const caseID = "88331"
		fixture = newRolloutFailureFixture(ctx, caseID, []rolloutFailureBundle{
			{Version: "1.0.2", ControllerImage: "wrong/image"},
		})

		g.By("creating a ClusterExtension with a 10-minute progress deadline for the failing bundle")
		ceName, cleanup := helpers.CreateClusterExtension("test", "1.0.2", fixture.Namespace, caseID,
			helpers.WithCatalogNameSelector(fixture.CatalogName),
			helpers.WithProgressDeadlineMinutes(10),
		)
		g.DeferCleanup(cleanup)

		g.By("waiting for the ClusterExtension and ClusterObjectSet to report ProgressDeadlineExceeded")
		eventually(ctx, func(g o.Gomega) {
			ce := &olmv1.ClusterExtension{}
			err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: ceName}, ce)
			g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterExtension")
			expectCondition(g, ce.Status.Conditions, olmv1.TypeProgressing, metav1.ConditionFalse, olmv1.ReasonProgressDeadlineExceeded)
			g.Expect(ce.Status.ActiveRevisions).NotTo(o.BeEmpty(), "expected at least one active revision")

			cos := &olmv1.ClusterObjectSet{}
			err = env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: ce.Status.ActiveRevisions[0].Name}, cos)
			g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterObjectSet")
			expectCondition(g, cos.Status.Conditions, olmv1.TypeProgressing, metav1.ConditionFalse, olmv1.ReasonProgressDeadlineExceeded)
		}, 12*time.Minute) // 12 minutes for the minimum 10 minute timeout + padding
	})

	g.It("PolarionID:88332-[OTP][OCPFeatureGate:NewOLMBoxCutterRuntime]A ClusterExtension is being upgraded and a persistent error prevents it from being fully rolled out", func(ctx g.SpecContext) {
		const caseID = "88332"
		fixture = newRolloutFailureFixture(ctx, caseID, []rolloutFailureBundle{
			{Version: "1.0.0", ControllerImage: image.ShellImage()},
			{Version: "1.0.2", ControllerImage: "wrong/image", Replaces: "1.0.0"},
		})

		g.By("creating a ClusterExtension on a healthy initial version")
		ceName, cleanup := helpers.CreateClusterExtension("test", "1.0.0", fixture.Namespace, caseID,
			helpers.WithCatalogNameSelector(fixture.CatalogName),
			func(ce *olmv1.ClusterExtension) {
				ce.Spec.Source.Catalog.UpgradeConstraintPolicy = olmv1.UpgradeConstraintPolicySelfCertified
			},
		)
		g.DeferCleanup(cleanup)

		g.By("waiting for the initial installation to complete")
		eventually(ctx, func(g o.Gomega) {
			ce := &olmv1.ClusterExtension{}
			err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: ceName}, ce)
			g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterExtension")
			expectCondition(g, ce.Status.Conditions, olmv1.TypeInstalled, metav1.ConditionTrue, "")
			expectCondition(g, ce.Status.Conditions, olmv1.TypeProgressing, metav1.ConditionTrue, olmv1.ReasonSucceeded)
		}, 3*time.Minute)

		g.By("updating the ClusterExtension to a version with a persistent rollout error")
		patchTarget := &olmv1.ClusterExtension{ObjectMeta: metav1.ObjectMeta{Name: ceName}}
		patch := []byte(`{"spec":{"source":{"catalog":{"version":"1.0.2"}}}}`)
		o.Expect(env.Get().K8sClient.Patch(ctx, patchTarget, client.RawPatch(types.MergePatchType, patch))).To(o.Succeed(), "failed to patch ClusterExtension")

		g.By("verifying both revisions remain active and the new revision reports a probe failure")
		eventually(ctx, func(g o.Gomega) {
			ce := &olmv1.ClusterExtension{}
			err := env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: ceName}, ce)
			g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterExtension")
			g.Expect(len(ce.Status.ActiveRevisions)).To(o.BeNumerically(">", 1))

			cos := &olmv1.ClusterObjectSet{}
			err = env.Get().K8sClient.Get(ctx, client.ObjectKey{Name: ce.Status.ActiveRevisions[1].Name}, cos)
			g.Expect(err).NotTo(o.HaveOccurred(), "failed to get ClusterObjectSet")
			expectCondition(g, cos.Status.Conditions, olmv1.TypeProgressing, metav1.ConditionTrue, olmv1.ReasonRollingOut)
			expectCondition(g, cos.Status.Conditions, olmv1.ClusterObjectSetTypeAvailable, metav1.ConditionFalse, olmv1.ClusterObjectSetReasonProbeFailure)
		}, 3*time.Minute)
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

func newRolloutFailureFixture(ctx g.SpecContext, caseID string, bundles []rolloutFailureBundle) rolloutFailureFixture {
	namespace := "ns-" + caseID
	catalogName := "test-catalog-" + caseID

	g.By("creating a test namespace")
	helpers.CreateNamespace(namespace)
	helpers.ExpectServiceAccountExists(ctx, "builder", namespace)
	helpers.ExpectServiceAccountExists(ctx, "deployer", namespace)

	g.By("allowing catalogd and operator-controller to pull test images")
	helpers.CreateImagePullerRoleBinding("image-puller-"+caseID, namespace)

	bundleRefs := make(map[string]string, len(bundles))
	for _, bundle := range bundles {
		imageName := fmt.Sprintf("test-bundle-%s-%s", caseID, strings.ReplaceAll(bundle.Version, ".", "-"))
		bundleRefs[bundle.Version] = fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", namespace, imageName)
		g.By(fmt.Sprintf("building bundle image for version %s", bundle.Version))
		helpers.BuildImage(ctx, namespace, imageName, bundleImageFiles(bundle))
	}

	g.By("building the catalog image")
	helpers.BuildImage(ctx, namespace, catalogName, catalogImageFiles(caseID, bundles, bundleRefs))

	cleanup, err := helpers.CreateClusterCatalog(ctx, catalogName, fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", namespace, catalogName))
	g.DeferCleanup(cleanup)
	o.Expect(err).NotTo(o.HaveOccurred(), "failed to create ClusterCatalog")

	helpers.ExpectCatalogToBeServing(ctx, catalogName)

	return rolloutFailureFixture{
		Namespace:   namespace,
		CatalogName: catalogName,
	}
}

func bundleImageFiles(bundle rolloutFailureBundle) map[string][]byte {
	replacements := map[string]string{
		"{{ TEST-BUNDLE }}":     "test",
		"{{ TEST-CONTROLLER }}": bundle.ControllerImage,
		"{{ BUNDLE-VERSION }}":  bundle.Version,
	}
	names := operatorbindata.AssetNames()
	out := make(map[string][]byte, len(names))
	for _, name := range names {
		data := operatorbindata.MustAsset(name)
		for k, v := range replacements {
			data = bytes.ReplaceAll(data, []byte(k), []byte(v))
		}
		out[name] = data
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
		if bundle.Replaces != "" {
			entry["replaces"] = "test.v" + bundle.Replaces
		}
		entries = append(entries, entry)
	}
	writeCatalogEntry(map[string]any{
		"schema":  "olm.channel",
		"name":    "alpha",
		"package": "test",
		"entries": entries,
	})

	return map[string][]byte{
		"Dockerfile":                          catalogbindata.MustAsset("Dockerfile"),
		"configs/.indexignore":                catalogbindata.MustAsset("configs/.indexignore"),
		"configs/catalog-" + caseID + ".yaml": []byte(fbc.String()),
	}
}

func expectCondition(g o.Gomega, conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus, reason string) {
	condition := meta.FindStatusCondition(conditions, conditionType)
	g.Expect(condition).NotTo(o.BeNil(), "%s condition not found", conditionType)
	g.Expect(condition.Status).To(o.Equal(status), "%s status mismatch", conditionType)
	if reason != "" {
		g.Expect(condition.Reason).To(o.Equal(reason), "%s reason mismatch", conditionType)
	}
}

func eventually(ctx context.Context, callback func(o.Gomega), timeout time.Duration) {
	o.Eventually(callback).WithContext(ctx).WithTimeout(timeout).WithPolling(helpers.DefaultPolling).Should(o.Succeed())
}
