package test

import (
	"context"
	"encoding/json"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/commons"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/mocks"
)

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 Catalogs", func() {
	It("should be installed", func(ctx SpecContext) {
		if !env.Get().IsOpenShift {
			Skip("Skipping test because requires OCP Catalogs: not OpenShift")
		}
		catalogs := []string{
			"openshift-certified-operators",
			"openshift-community-operators",
			"openshift-redhat-marketplace",
			"openshift-redhat-operators",
		}

		for _, name := range catalogs {
			By(fmt.Sprintf("checking that %q exists and is serving", name))
			conditions := getCatalogConditions(ctx, name)
			Expect(meta.IsStatusConditionPresentAndEqual(conditions, "Serving", metav1.ConditionTrue)).
				To(BeTrue(), fmt.Sprintf("catalog %q not serving", name))
		}
	})
})

var _ = Describe("[sig-olmv1][OCPFeatureGate:NewOLM][Skipped:Disconnected] OLMv1 New Catalog Install", func() {
	It("should fail to install if it has an invalid reference", func(ctx SpecContext) {
		By("creating the malformed catalog with an invalid image ref")
		var catName = "bad-catalog"
		cleanup, err := mocks.CreateBrokenClusterCatalog(catName, "example.com/does-not-exist:latest")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(cleanup)

		By("waiting for the catalog to report failure")
		Eventually(func() {
			conditions := getCatalogConditions(ctx, catName)
			c := meta.FindStatusCondition(conditions, "Progressing")
			Expect(c).ToNot(BeNil(), "condition 'Progressing' should be present")
			Expect(c.Status).To(Equal(metav1.ConditionTrue), "expected 'Progressing' to be True")
			Expect(c.Reason).To(Equal("Retrying"), "expected reason 'Retrying'")
			Expect(c.Message).To(ContainSubstring("error creating image source"), "expected error message about image source")
		}).WithTimeout(commons.DefaultTimeout).WithPolling(commons.DefaultPollingInterval).Should(Succeed())
	})
})

func getCatalogConditions(ctx context.Context, name string) []metav1.Condition {
	k8sClient := env.Get().K8sClient
	cat := &unstructured.Unstructured{}
	cat.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   commons.GroupOLMv1,
		Version: commons.CatalogAPIVersion,
		Kind:    commons.KindCatalog,
	})
	Expect(k8sClient.Get(ctx, client.ObjectKey{Name: name}, cat)).To(Succeed())
	raw, found, err := unstructured.NestedSlice(cat.Object, "status", "conditions")
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())
	data, err := json.Marshal(raw)
	Expect(err).NotTo(HaveOccurred())

	var conditions []metav1.Condition
	Expect(json.Unmarshal(data, &conditions)).To(Succeed())
	return conditions
}
