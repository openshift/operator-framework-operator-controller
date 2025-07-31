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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/operator-framework/operator-controller/api/v1"

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
		catName := "bad-catalog"
		cleanup, err := mocks.CreateBrokenClusterCatalog(catName, "example.com/does-not-exist:latest")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(cleanup)

		By("waiting for the catalog to report failure")
		Eventually(func() error {
			conditions := getCatalogConditions(ctx, catName)
			c := meta.FindStatusCondition(conditions, "Progressing")
			if c == nil {
				return fmt.Errorf("condition 'Progressing' not present")
			}
			if c.Status != metav1.ConditionTrue {
				return fmt.Errorf("expected status 'True', got '%s'", c.Status)
			}
			if c.Reason != "Retrying" {
				return fmt.Errorf("expected reason 'Retrying', got '%s'", c.Reason)
			}
			if !strings.Contains(c.Message, "error creating image source") {
				return fmt.Errorf("expected error message about image source, got '%s'", c.Message)
			}
			return nil
		}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
	})
})

func getCatalogConditions(ctx context.Context, name string) []metav1.Condition {
	k8sClient := env.Get().K8sClient
	catalog := &v1.ClusterCatalog{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, catalog)
	Expect(err).To(Succeed())
	return catalog.Status.Conditions
}
