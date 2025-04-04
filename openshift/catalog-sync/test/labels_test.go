package test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/catalog"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

const requiredLabelKey = "operators.operatorframework.io.index.configs.v1"
const expectedValue = "/configs"

func checkRequiredLabel(catalogName string, cfgs []*olmpackage.Data) {
	if len(cfgs) == 0 || cfgs[0] == nil {
		Fail(fmt.Sprintf("No catalog data available for %s", catalogName))
	}

	labels := cfgs[0].ImageLabels
	labelResult := "Yes"
	failureReason := "-"

	actual, ok := labels[requiredLabelKey]
	if !ok {
		labelResult = "No"
		failureReason = "missing label"
	} else if actual != expectedValue {
		labelResult = "No"
		failureReason = "label has incorrect value"
	}

	res := result.TestResult{
		TestID:           "required_label_check",
		TestContextTitle: fmt.Sprintf("Catalog image must contain label %q with value %q", requiredLabelKey, expectedValue),
		CatalogName:      catalogName,
		TotalPackages:    len(cfgs),
		OptionalColumns: map[string]interface{}{
			"Contains": labelResult,
			"Failure":  failureReason,
		},
	}

	testResults = append(testResults, res)

	Expect(labelResult).To(Equal("Yes"),
		"Image for catalog %s is not labeled correctly: %s", catalogName, failureReason)
}

var _ = Describe("Image Label Validation", func() {
	for image, catalogName := range catalog.CatalogImages {
		Context(fmt.Sprintf("Catalog %s (%s)", catalogName, image), func() {
			It(fmt.Sprintf("should have the required label (%s) with correct value (%s)", requiredLabelKey, expectedValue), func() {
				cfgs, exists := catalogDataMap[catalogName]
				Expect(exists).To(BeTrue(), "No catalog data found for %s", catalogName)
				Expect(len(cfgs)).ToNot(BeZero(), "No packages found for %s", catalogName)
				checkRequiredLabel(catalogName, cfgs)
			})
		})
	}
})
