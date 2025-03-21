package test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
	"github/operator-framework-operator-controller/openshift/catalog-sync/test/common"
)

var testResults []result.TestResult
var catalogDataMap = make(map[string][]*olmpackage.Data)
var tempDir string

// TestMetadataValidationSuite verifies the quality of the packages
func TestMetadataValidationSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog Metadata Validation Suite")
}

var _ = BeforeSuite(func() {
	catalogDataMap, tempDir = common.SetupTestSuite()
})

var _ = AfterSuite(func() {
	result.OutputSummaryWith(testResults)
	_ = os.RemoveAll(tempDir)
})
