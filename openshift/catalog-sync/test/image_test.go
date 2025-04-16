package test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

const requiredLabelKey = "operators.operatorframework.io.index.configs.v1"
const expectedValue = "/configs"

func hasRequiredLabel(catalogName string, labels map[string]string) bool {
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
		OptionalColumns: map[string]interface{}{
			"Contains": labelResult,
			"Failure":  failureReason,
		},
	}
	testResults = append(testResults, res)
	if labelResult == "Yes" {
		return true
	}
	return false
}

func hasTmpCachePogrebFile(catalogName string) bool {
	filePath := filepath.Join(tempDir, catalogName, "fs", "tmp", "cache", "pogreb.v1")
	_, err := os.Stat(filePath)
	res := "No"
	if err == nil {
		res = "Yes"
	}
	resultRow := result.TestResult{
		TestID:           "pogreb_cache_check",
		TestContextTitle: "Catalog image must contain must contain /tmp/cache/pogreb.v1",
		CatalogName:      catalogName,
		OptionalColumns: map[string]interface{}{
			"Found": res,
		},
	}

	testResults = append(testResults, resultRow)
	return true
}

var _ = Describe("Image Validation", func() {
	Context("Verify Labels", func() {
		It(fmt.Sprintf("should have the required label (%s) with correct value (%s)", requiredLabelKey, expectedValue), func() {
			failed := false
			for _, image := range catalogImages {
				if !hasRequiredLabel(image.Name, image.Labels) {
					failed = true
				}
			}
			Expect(failed).To(BeFalse(), "One or more catalogs does not have the required label")
		})
	})

	Context("Verify FileSystem", func() {
		It("should contain /tmp/cache/pogreb.v1", func() {
			failed := false
			for _, image := range catalogImages {
				if !hasTmpCachePogrebFile(image.Name) {
					failed = true
				}
			}
			Expect(failed).To(BeFalse(), "One or more catalogs are missing /tmp/cache/pogreb.v1")
		})
	})
})
