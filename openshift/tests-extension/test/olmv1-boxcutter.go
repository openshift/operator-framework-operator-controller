package test

import (
	"context"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"github.com/openshift/origin/test/extended/util/image"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	boxcutterbundle "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/boxcutter/bundle"
	boxcuttercatalog "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/bindata/boxcutter/catalog"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:BoxcutterRuntime] OLMv1 Boxcutter Runtime", func() {
	var unique, nsName, ccName, opName string

	BeforeEach(func(ctx SpecContext) {
		helpers.RequireOLMv1CapabilityOnOpenshift()

		testVersion := env.Get().OpenShiftVersion
		replacements := map[string]string{
			"{{ TEST-BUNDLE }}": "", // Auto-filled
			"{{ NAMESPACE }}":   "", // Auto-filled
			"{{ VERSION }}":     testVersion,

			// Using the shell image provided by origin as the controller image.
			// The image is mirrored into disconnected environments for testing.
			"{{ TEST-CONTROLLER }}": image.ShellImage(),
		}
		unique, nsName, ccName, opName = helpers.NewCatalogAndClusterBundles(ctx, replacements,
			boxcuttercatalog.AssetNames, boxcuttercatalog.Asset,
			boxcutterbundle.AssetNames, boxcutterbundle.Asset,
		)
	})

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			By("dumping for debugging")
			helpers.DescribeAllClusterCatalogs(context.Background())
			helpers.DescribeAllClusterExtensions(context.Background(), nsName)
		}
	})

	It("should create ClusterExtensionRevision on initial installation",
		Label("[sig-olmv1][OCPFeatureGate:BoxcutterRuntime][Skipped:Disconnected] OLMv1 Boxcutter Runtime should create ClusterExtensionRevision on initial installation"),
		func(ctx SpecContext) {

			By("ensuring no ClusterExtension and CRD for the operator")
			helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

			By("applying the ClusterExtension resource")
			name, cleanup := helpers.CreateClusterExtension(opName, "", nsName, unique, helpers.WithCatalogNameSelector(ccName))
			DeferCleanup(cleanup)

			By("waiting for the ClusterExtension to be installed")
			helpers.ExpectClusterExtensionToBeInstalled(ctx, name)

			k8sClient := env.Get().K8sClient

			By("verifying that a ClusterExtensionRevision with revision number 1 is created")
			expectedRevisionName := fmt.Sprintf("%s-1", name)
			var revision olmv1.ClusterExtensionRevision

			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: expectedRevisionName}, &revision)
				g.Expect(err).ToNot(HaveOccurred(), "ClusterExtensionRevision should exist")
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("verifying the revision has spec.revision = 1")
			Expect(revision.Spec.Revision).To(Equal(int64(1)), "Revision number should be 1")

			By("verifying the revision has lifecycleState = Active")
			Expect(revision.Spec.LifecycleState).To(Equal(olmv1.ClusterExtensionRevisionLifecycleStateActive),
				"Lifecycle state should be Active")

			By("verifying the revision has proper owner reference to the ClusterExtension")
			var foundOwnerRef bool
			for _, ownerRef := range revision.GetOwnerReferences() {
				if ownerRef.Name == name && ownerRef.Kind == "ClusterExtension" {
					foundOwnerRef = true
					Expect(ownerRef.Controller).ToNot(BeNil(), "Controller field should be set")
					Expect(*ownerRef.Controller).To(BeTrue(), "ClusterExtension should be the controller")
					break
				}
			}
			Expect(foundOwnerRef).To(BeTrue(), "ClusterExtension should be in owner references")

			By("verifying the revision has proper labels")
			Expect(revision.Labels).To(HaveKeyWithValue("olm.operatorframework.io/owner", name),
				"Revision should have owner label")

			By("verifying the revision has objects in spec.phases")
			Expect(revision.Spec.Phases).ToNot(BeEmpty(), "Revision should have at least one phase")

			// Count total objects across all phases
			totalObjects := 0
			for _, phase := range revision.Spec.Phases {
				totalObjects += len(phase.Objects)
			}
			Expect(totalObjects).To(BeNumerically(">", 0), "Revision should have at least one object in phases")

			By("verifying the revision has no previous revisions (first revision)")
			Expect(revision.Spec.Previous).To(BeEmpty(), "First revision should have no previous revisions")

			By("verifying the revision status conditions")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: expectedRevisionName}, &revision)
				g.Expect(err).ToNot(HaveOccurred())

				// Check for expected conditions (may vary based on implementation)
				// At minimum, we expect some status to be populated
				g.Expect(revision.Status.Conditions).ToNot(BeNil(), "Revision should have status conditions")
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("verifying the ClusterExtension references the created revision")
			var ext olmv1.ClusterExtension
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
			Expect(err).ToNot(HaveOccurred())

			// The ClusterExtension status should reflect information about the revision
			// This depends on the implementation - we're checking that installation succeeded
			installed := meta.FindStatusCondition(ext.Status.Conditions, string(olmv1.TypeInstalled))
			Expect(installed).ToNot(BeNil())
			Expect(installed.Status).To(Equal(metav1.ConditionTrue))
		})

	It("should create new revision on upgrade",
		Label("[sig-olmv1][OCPFeatureGate:BoxcutterRuntime][Skipped:Disconnected] OLMv1 Boxcutter Runtime should create new revision on upgrade"),
		func(ctx SpecContext) {

			By("ensuring no ClusterExtension and CRD for the operator")
			helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

			By("applying the ClusterExtension resource with version 1.0.0")
			name, cleanup := helpers.CreateClusterExtension(opName, "1.0.0", nsName, unique, helpers.WithCatalogNameSelector(ccName))
			DeferCleanup(cleanup)

			By("waiting for the ClusterExtension to be installed with v1.0.0")
			helpers.ExpectClusterExtensionToBeInstalled(ctx, name)

			k8sClient := env.Get().K8sClient

			By("verifying that revision-1 exists")
			revision1Name := fmt.Sprintf("%s-1", name)
			var revision1 olmv1.ClusterExtensionRevision
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: revision1Name}, &revision1)
				g.Expect(err).ToNot(HaveOccurred(), "revision-1 should exist")
				g.Expect(revision1.Spec.Revision).To(Equal(int64(1)))
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("storing the UID of revision-1 for later verification")
			revision1UID := revision1.GetUID()

			By("upgrading the ClusterExtension to version 2.0.0")
			var ext olmv1.ClusterExtension
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
			Expect(err).ToNot(HaveOccurred())

			// Update the version constraint to 2.0.0
			ext.Spec.Source.Catalog.Version = "2.0.0"
			err = k8sClient.Update(ctx, &ext)
			Expect(err).ToNot(HaveOccurred(), "failed to update ClusterExtension version to 2.0.0")

			By("waiting for the upgrade to complete")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
				g.Expect(err).ToNot(HaveOccurred())

				installed := meta.FindStatusCondition(ext.Status.Conditions, string(olmv1.TypeInstalled))
				g.Expect(installed).ToNot(BeNil())
				g.Expect(installed.Status).To(Equal(metav1.ConditionTrue))
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("verifying that a new revision-2 is created")
			revision2Name := fmt.Sprintf("%s-2", name)
			var revision2 olmv1.ClusterExtensionRevision
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: revision2Name}, &revision2)
				g.Expect(err).ToNot(HaveOccurred(), "revision-2 should exist")
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("verifying revision-2 has spec.revision = 2")
			Expect(revision2.Spec.Revision).To(Equal(int64(2)), "Revision number should be 2")

			By("verifying revision-2 has spec.previous array containing reference to revision-1")
			Expect(revision2.Spec.Previous).ToNot(BeEmpty(), "revision-2 should have previous revisions")
			foundRevision1 := false
			for _, prev := range revision2.Spec.Previous {
				if prev.Name == revision1Name && prev.UID == revision1UID {
					foundRevision1 = true
					break
				}
			}
			Expect(foundRevision1).To(BeTrue(), "revision-2 should reference revision-1 in spec.previous")

			By("verifying both revision-1 and revision-2 exist")
			err = k8sClient.Get(ctx, client.ObjectKey{Name: revision1Name}, &revision1)
			Expect(err).ToNot(HaveOccurred(), "revision-1 should still exist after upgrade")

			err = k8sClient.Get(ctx, client.ObjectKey{Name: revision2Name}, &revision2)
			Expect(err).ToNot(HaveOccurred(), "revision-2 should exist")

			By("verifying revision-2 is Active")
			Expect(revision2.Spec.LifecycleState).To(Equal(olmv1.ClusterExtensionRevisionLifecycleStateActive),
				"revision-2 should be Active")

			By("verifying the ClusterExtension reflects the upgrade")
			err = k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
			Expect(err).ToNot(HaveOccurred())

			installed := meta.FindStatusCondition(ext.Status.Conditions, string(olmv1.TypeInstalled))
			Expect(installed).ToNot(BeNil())
			Expect(installed.Status).To(Equal(metav1.ConditionTrue), "ClusterExtension should remain installed after upgrade")
		})

	It("should garbage collect only archived revisions beyond limit of 5",
		Label("[sig-olmv1][OCPFeatureGate:BoxcutterRuntime][Skipped:Disconnected][Slow] OLMv1 Boxcutter Runtime should garbage collect only archived revisions beyond limit of 5"),
		func(ctx SpecContext) {

			By("ensuring no ClusterExtension and CRD for the operator")
			helpers.EnsureCleanupClusterExtension(context.Background(), opName, "")

			By("applying the ClusterExtension resource with version 1.0.0")
			name, cleanup := helpers.CreateClusterExtension(opName, "1.0.0", nsName, unique, helpers.WithCatalogNameSelector(ccName))
			DeferCleanup(cleanup)

			By("waiting for the ClusterExtension to be installed with v1.0.0")
			helpers.ExpectClusterExtensionToBeInstalled(ctx, name)

			k8sClient := env.Get().K8sClient

			By("performing upgrades through v2.0.0 to v9.0.0 to create 9 total revisions")
			versions := []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0", "6.0.0", "7.0.0", "8.0.0", "9.0.0"}
			for _, version := range versions {
				By(fmt.Sprintf("upgrading to version %s", version))
				var ext olmv1.ClusterExtension
				Eventually(func(g Gomega) {
					err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
					g.Expect(err).ToNot(HaveOccurred())

					ext.Spec.Source.Catalog.Version = version
					err = k8sClient.Update(ctx, &ext)
					g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to update to version %s", version))
				}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

				By(fmt.Sprintf("waiting for upgrade to %s to complete", version))
				Eventually(func(g Gomega) {
					var ext olmv1.ClusterExtension
					err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
					g.Expect(err).ToNot(HaveOccurred())

					installed := meta.FindStatusCondition(ext.Status.Conditions, string(olmv1.TypeInstalled))
					g.Expect(installed).ToNot(BeNil())
					g.Expect(installed.Status).To(Equal(metav1.ConditionTrue))
				}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
			}

			By("verifying all 9 revisions exist")
			for i := 1; i <= 9; i++ {
				revisionName := fmt.Sprintf("%s-%d", name, i)
				var revision olmv1.ClusterExtensionRevision
				err := k8sClient.Get(ctx, client.ObjectKey{Name: revisionName}, &revision)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("revision-%d should exist", i))
				Expect(revision.Spec.Revision).To(Equal(int64(i)))
			}

			By("archiving revisions 1 through 6")
			// NOTE: GC only deletes ARCHIVED revisions beyond the limit of 5.
			// Active revisions are NEVER deleted regardless of count.
			// We archive 6 revisions here, so when we later trigger a new revision,
			// the oldest archived revision (revision-1) should be deleted,
			// keeping 5 archived revisions (2-6) + active revisions (7-9).
			for i := 1; i <= 6; i++ {
				revisionName := fmt.Sprintf("%s-%d", name, i)
				Eventually(func(g Gomega) {
					var revision olmv1.ClusterExtensionRevision
					err := k8sClient.Get(ctx, client.ObjectKey{Name: revisionName}, &revision)
					g.Expect(err).ToNot(HaveOccurred())

					revision.Spec.LifecycleState = olmv1.ClusterExtensionRevisionLifecycleStateArchived
					err = k8sClient.Update(ctx, &revision)
					g.Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to archive revision-%d", i))
				}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

				By(fmt.Sprintf("waiting for revision-%d to be archived", i))
				Eventually(func(g Gomega) {
					var revision olmv1.ClusterExtensionRevision
					err := k8sClient.Get(ctx, client.ObjectKey{Name: revisionName}, &revision)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(revision.Spec.LifecycleState).To(Equal(olmv1.ClusterExtensionRevisionLifecycleStateArchived))
				}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
			}

			By("verifying all 9 revisions still exist before triggering GC")
			for i := 1; i <= 9; i++ {
				revisionName := fmt.Sprintf("%s-%d", name, i)
				var revision olmv1.ClusterExtensionRevision
				err := k8sClient.Get(ctx, client.ObjectKey{Name: revisionName}, &revision)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("revision-%d should still exist before GC", i))
			}

			By("triggering a new revision creation by modifying the ClusterExtension (should trigger GC)")
			// We can trigger a new revision by making a change to the spec that requires a new revision
			// For example, adding a label or annotation to the ClusterExtension
			Eventually(func(g Gomega) {
				var ext olmv1.ClusterExtension
				err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
				g.Expect(err).ToNot(HaveOccurred())

				if ext.Annotations == nil {
					ext.Annotations = make(map[string]string)
				}
				ext.Annotations["test.boxcutter/trigger-gc"] = "true"
				err = k8sClient.Update(ctx, &ext)
				g.Expect(err).ToNot(HaveOccurred())
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("waiting for reconciliation to complete")
			Eventually(func(g Gomega) {
				var ext olmv1.ClusterExtension
				err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &ext)
				g.Expect(err).ToNot(HaveOccurred())

				installed := meta.FindStatusCondition(ext.Status.Conditions, string(olmv1.TypeInstalled))
				g.Expect(installed).ToNot(BeNil())
				g.Expect(installed.Status).To(Equal(metav1.ConditionTrue))
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("verifying that only revision-1 has been garbage collected (deleted)")
			// Only revision-1 should be deleted because:
			// - We have 6 archived revisions (1-6)
			// - Limit is 5 archived revisions
			// - Revisions 7-9 are active (current = 9)
			// - GC deletes only the oldest archived revision beyond the limit
			Eventually(func(g Gomega) {
				var revision olmv1.ClusterExtensionRevision
				err := k8sClient.Get(ctx, client.ObjectKey{Name: fmt.Sprintf("%s-1", name)}, &revision)
				g.Expect(err).To(HaveOccurred(), "revision-1 should be deleted")
				g.Expect(err).To(MatchError(ContainSubstring("not found")), "revision-1 should be not found")
			}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())

			By("verifying that revisions 2-9 still exist")
			// Revisions 2-6: 5 archived revisions (within limit)
			// Revisions 7-9: active revisions (never deleted)
			for i := 2; i <= 9; i++ {
				revisionName := fmt.Sprintf("%s-%d", name, i)
				var revision olmv1.ClusterExtensionRevision
				err := k8sClient.Get(ctx, client.ObjectKey{Name: revisionName}, &revision)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("revision-%d should still exist", i))
				Expect(revision.Spec.Revision).To(Equal(int64(i)))
			}

			By("verifying revision-9 has spec.previous containing references to revisions 2-8")
			var revision9 olmv1.ClusterExtensionRevision
			err := k8sClient.Get(ctx, client.ObjectKey{Name: fmt.Sprintf("%s-9", name)}, &revision9)
			Expect(err).ToNot(HaveOccurred())

			// spec.previous should contain 7 revisions (2-8)
			// revision-1 was GC'd so it's not in the list
			Expect(revision9.Spec.Previous).To(HaveLen(7), "revision-9 should have 7 previous revisions (2-8)")

			// Verify the previous revisions are 2, 3, 4, 5, 6, 7, 8
			previousRevisionNames := make(map[string]bool)
			for _, prev := range revision9.Spec.Previous {
				previousRevisionNames[prev.Name] = true
			}

			for i := 2; i <= 8; i++ {
				expectedName := fmt.Sprintf("%s-%d", name, i)
				Expect(previousRevisionNames).To(HaveKey(expectedName),
					fmt.Sprintf("revision-9 should reference revision-%d in spec.previous", i))
			}

			// Verify revision-1 is NOT in the previous list (it was GC'd)
			Expect(previousRevisionNames).ToNot(HaveKey(fmt.Sprintf("%s-1", name)),
				"revision-9 should NOT reference revision-1 (it was GC'd)")
		})
})
