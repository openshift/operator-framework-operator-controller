package test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/catalog"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
	"github/operator-framework-operator-controller/openshift/catalog-sync/test/common"
)

var testResults []result.TestResult
var catalogImages []*catalog.Image
var tempDir string

// TestSuite verifies the quality of the packages
func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validate Catalogs")
}

var _ = BeforeSuite(func() {
	catalogImages, tempDir = common.SetupTestSuite()
})

var _ = AfterSuite(func() {
	result.OutputSummaryWith(testResults)
	_ = os.RemoveAll(tempDir)
})
