package test

import (
	"fmt"
	"strings"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/pkg/helpers"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 Catalogs", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})

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

func verifyCatalogEndpoint(ctx SpecContext, catalog, endpoint, query string) {
	k8sClient := env.Get().K8sClient

	By(fmt.Sprintf("Retrieving base URL from ClusterCatalog %q", catalog))
	cc := &olmv1.ClusterCatalog{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: catalog}, cc)
	Expect(err).NotTo(HaveOccurred(), "failed to get ClusterCatalog")

	Expect(cc.Status.URLs.Base).NotTo(BeEmpty(), fmt.Sprintf("catalog %q has empty base URL", catalog))
	serviceURL := fmt.Sprintf("%s/api/v1/%s%s", cc.Status.URLs.Base, endpoint, query)

	By(fmt.Sprintf("Creating curl Job to hit: %s", serviceURL))

	jobNamePrefix := fmt.Sprintf("verify-%s-%s",
		strings.ReplaceAll(endpoint, "?", ""),
		strings.ReplaceAll(catalog, "-", ""))

	job := buildCurlJob(jobNamePrefix, "default", serviceURL)
	err = k8sClient.Create(ctx, job)
	Expect(err).NotTo(HaveOccurred(), "failed to create Job")

	DeferCleanup(func(ctx SpecContext) {
		_ = k8sClient.Delete(ctx, job)
	})

	By("Waiting for Job to succeed")
	Eventually(func(g Gomega) {
		recheck := &batchv1.Job{}
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(job), recheck)).NotTo(HaveOccurred())

		for _, c := range recheck.Status.Conditions {
			if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
				return
			}
			if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
				Fail(fmt.Sprintf("Job failed: %s", c.Message))
			}
		}
	}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
}

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-community-operators Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/all endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-community-operators", "all", "")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-certified-operators Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/all endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-certified-operators", "all", "")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-redhat-marketplace Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/all endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-redhat-marketplace", "all", "")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 openshift-redhat-operators Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/all endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-redhat-operators", "all", "")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-community-operators Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/metas endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-community-operators", "metas", "?schema=olm.package")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-certified-operators Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/metas endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-certified-operators", "metas", "?schema=olm.package")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-redhat-marketplace Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/metas endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-redhat-marketplace", "metas", "?schema=olm.package")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLMCatalogdAPIV1Metas][Skipped:Disconnected] OLMv1 openshift-redhat-operators Catalog", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
		if !env.Get().IsOpenShift {
			Skip("This test requires OpenShift Catalogs")
		}
	})
	It("should serve FBC via the /v1/api/metas endpoint", func(ctx SpecContext) {
		verifyCatalogEndpoint(ctx, "openshift-redhat-operators", "metas", "?schema=olm.package")
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 New Catalog Install", func() {
	BeforeEach(func() {
		helpers.RequireOLMv1CapabilityOnOpenshift()
	})
	It("should fail to install if it has an invalid reference", func(ctx SpecContext) {
		unique := rand.String(4)
		catName := "bad-catalog-" + unique
		imageRef := "example.com/does-not-exist:latest"

		By("creating the malformed catalog with an invalid image ref")
		cleanup, err := helpers.CreateClusterCatalog(ctx, catName, imageRef)
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
		}).WithTimeout(helpers.DefaultTimeout).WithPolling(helpers.DefaultPolling).Should(Succeed())
	})
})

func buildCurlJob(prefix, namespace, url string) *batchv1.Job {
	backoff := int32(1)
	// This means the k8s garbage collector will automatically delete the job 5 minutes
	// after it has completed or failed.
	// However, this automatic process is subordinate to a manual deletion
	// When we call `k8sClient.Delete(ctx, job)` k8s will delete it immediately,
	// overriding the TTL setting
	ttl := int32(300)

	// The command string with the URL placeholder
	commandString := fmt.Sprintf(`set -ex;
                            curl -v -k %q;
                            if [ $? -ne 0 ]; then
                                echo "Failed to access endpoint";
                                exit 1;
                            fi;
                            echo "Successfully verified API endpoint";
                            exit 0;`, url)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix + "-",
			Namespace:    namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttl,
			BackoffLimit:            &backoff,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "api-tester",
						Image:   "registry.redhat.io/rhel8/httpd-24:latest",
						Command: []string{"/bin/bash", "-c", commandString},
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
