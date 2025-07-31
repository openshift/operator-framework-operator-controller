package test

import (
	"context"
	"fmt"
	"strings"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 Catalogs", func() {
	It("should be installed", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Skipping test because it requires OpenShift Catalogs")
		}

		k8sClient := env.Get().K8sClient

		catalogs := []string{
			"openshift-certified-operators",
			"openshift-community-operators",
			"openshift-redhat-marketplace",
			"openshift-redhat-operators",
		}

		for _, name := range catalogs {
			By(fmt.Sprintf("checking that %q exists", name))
			catalog := &olmv1.ClusterCatalog{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, catalog)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to get catalog %q", name))

			conditions := catalog.Status.Conditions
			Expect(conditions).NotTo(BeEmpty(), fmt.Sprintf("catalog %q has empty status.conditions", name))

			By(fmt.Sprintf("checking that %q is serving", name))

			Expect(meta.IsStatusConditionPresentAndEqual(conditions, "Serving", metav1.ConditionTrue)).
				To(BeTrue(), fmt.Sprintf("expected catalog %q to have condition {type: Serving, status: True},"+
					" but it did not", name))
		}
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-community-operators Catalog", testCatalogAllEndpoint("openshift-community-operators"))
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-certified-operators Catalog", testCatalogAllEndpoint("openshift-certified-operators"))
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-redhat-marketplace Catalog", testCatalogAllEndpoint("openshift-redhat-marketplace"))
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-redhat-operators Catalog", testCatalogAllEndpoint("openshift-redhat-operators"))

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-community-operators Catalog", testCatalogMetasEndpoint("openshift-community-operators"))
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-certified-operators Catalog", testCatalogMetasEndpoint("openshift-certified-operators"))
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-redhat-marketplace Catalog", testCatalogMetasEndpoint("openshift-redhat-marketplace"))
var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-redhat-operators Catalog", testCatalogMetasEndpoint("openshift-redhat-operators"))

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 New Catalog Install", func() {
	It("should fail to install if it has an invalid reference", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift")
		}

		catName := "bad-catalog"
		imageRef := "example.com/does-not-exist:latest"

		By("creating the malformed catalog with an invalid image ref")
		cleanup, err := createClusterCatalog(ctx, catName, imageRef)
		Expect(err).NotTo(HaveOccurred(), "failed to create ClusterCatalog")
		DeferCleanup(cleanup)

		k8sClient := env.Get().K8sClient

		By("waiting for the catalog to report failure via Progressing=True and reason=Retrying")
		Eventually(func(g Gomega) {
			catalog := &olmv1.ClusterCatalog{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: catName}, catalog)
			g.Expect(err).NotTo(HaveOccurred(), "failed to get catalog")
			conditions := catalog.Status.Conditions
			c := meta.FindStatusCondition(conditions, "Progressing")
			g.Expect(c).NotTo(BeNil(), "expected 'Progressing' condition to be present")
			g.Expect(c.Status).To(Equal(metav1.ConditionTrue), "expected Progressing=True")
			g.Expect(c.Reason).To(Equal("Retrying"), "expected reason to be 'Retrying'")
			g.Expect(c.Message).To(ContainSubstring("error creating image source"), "expected image source error")
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})
})

// createClusterCatalog creates a ClusterCatalog with the specified name and image reference using the strongly typed API.
// It returns a cleanup function to delete the catalog after use.
func createClusterCatalog(ctx context.Context, name, imageRef string) (func(), error) {
	k8sClient := env.Get().K8sClient

	catalog := &olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: olmv1.ClusterCatalogSpec{
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
				Image: &olmv1.ImageSource{
					Ref: imageRef,
				},
			},
		},
	}

	if err := k8sClient.Create(ctx, catalog); err != nil {
		return nil, fmt.Errorf("failed to create ClusterCatalog: %w", err)
	}

	// Cleanup function to delete the catalog when done
	return func() {
		_ = k8sClient.Delete(ctx, &olmv1.ClusterCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}, nil
}

func testCatalogAllEndpoint(catalog string) func() {
	return func() {
		It("should serve FBC via the /v1/api/all endpoint", func(ctx SpecContext) {
			if !env.Get().IsOpenShift {
				Skip("This test requires OpenShift")
			}

			k8sClient := env.Get().K8sClient

			By(fmt.Sprintf("Retrieving base URL from ClusterCatalog %q", catalog))
			cc := &olmv1.ClusterCatalog{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: catalog}, cc)
			Expect(err).NotTo(HaveOccurred(), "failed to get ClusterCatalog")

			Expect(cc.Status.URLs.Base).NotTo(BeEmpty(), fmt.Sprintf("catalog %q has empty base URL", catalog))
			serviceURL := fmt.Sprintf("%s/api/v1/all", cc.Status.URLs.Base)

			By(fmt.Sprintf("Creating curl Job to hit: %s", serviceURL))

			job := buildCurlJob(fmt.Sprintf("verify-all-%s", strings.ReplaceAll(catalog, "-", "")), "default", serviceURL)
			err = k8sClient.Create(ctx, job)
			Expect(err).NotTo(HaveOccurred(), "failed to create Job")

			By("Waiting for Job to succeed")
			Eventually(func(g Gomega) {
				recheck := &batchv1.Job{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(job), recheck)
				Expect(err).NotTo(HaveOccurred(), "failed to get Job")

				for _, c := range recheck.Status.Conditions {
					if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
						return
					}
					if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
						Fail(fmt.Sprintf("Job failed: %s", c.Message))
					}
				}
				Fail("Job has not completed yet")
			}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		})
	}
}

func buildCurlJob(name, namespace, url string) *batchv1.Job {
	backoff := int32(1)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoff,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "curl",
						Image:   "registry.redhat.io/rhel8/httpd-24:latest",
						Command: []string{"/bin/bash", "-c", fmt.Sprintf("curl -vk %q", url)},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("50Mi"),
							},
						},
					}},
				},
			},
		},
	}
}

func testCatalogMetasEndpoint(catalog string) func() {
	return func() {
		It("should serve FBC via the /v1/api/metas endpoint", func(ctx SpecContext) {
			if !env.Get().IsOpenShift {
				Skip("This test requires OpenShift catalogs")
			}

			k8sClient := env.Get().K8sClient

			By(fmt.Sprintf("Retrieving base URL from ClusterCatalog %q", catalog))
			cc := &olmv1.ClusterCatalog{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: catalog}, cc)
			Expect(err).NotTo(HaveOccurred(), "failed to get ClusterCatalog")

			Expect(cc.Status.URLs.Base).NotTo(BeEmpty(), fmt.Sprintf("catalog %q has empty base URL", catalog))
			serviceURL := fmt.Sprintf("%s/api/v1/metas?schema=olm.package", cc.Status.URLs.Base)

			By(fmt.Sprintf("Creating curl Job to hit: %s", serviceURL))

			job := buildCurlJob(fmt.Sprintf("verify-metas-%s", strings.ReplaceAll(catalog, "-", "")), "default", serviceURL)
			err = k8sClient.Create(ctx, job)
			Expect(err).NotTo(HaveOccurred(), "failed to create Job")

			By("Waiting for Job to succeed")
			Eventually(func(g Gomega) {
				recheck := &batchv1.Job{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(job), recheck)
				g.Expect(err).NotTo(HaveOccurred(), "failed to get Job")

				for _, c := range recheck.Status.Conditions {
					if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
						return
					}
					if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
						Fail(fmt.Sprintf("Job failed: %s", c.Message))
					}
				}
				Fail("Job has not completed yet")
			}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		})
	}
}
