package test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/check"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/extract"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog Validation Suite")
}

var _ = Describe("Catalog Image Validation Suite", func() {
	for i := range catalogTestCases {
		tc := catalogTestCases[i]

		if !hasAnyChecks(tc.Checks) {
			continue
		}

		It(fmt.Sprintf("validates image: %s", tc.Name), func() {
			ctx := context.Background()

			res, err := extract.PrepareOCIImage(ctx, tc.ImageRef, tc.Name)
			Expect(err).ToNot(HaveOccurred(), "failed to prepare image")
			defer res.Cleanup()

			err = check.Check(ctx, "v1", res.Store, res.TmpDir, tc.Checks)
			Expect(err).ToNot(HaveOccurred(), "validation failed")
		})
	}
})

func hasAnyChecks(c check.Checks) bool {
	return len(c.ImageChecks) > 0 ||
		len(c.FilesystemChecks) > 0 ||
		len(c.CatalogChecks) > 0
}

type CatalogTestCase struct {
	Name     string
	ImageRef string
	Checks   check.Checks
}

var catalogTestCases = []CatalogTestCase{
	{
		Name:     "community",
		ImageRef: "registry.redhat.io/redhat/community-operator-index:v4.18",
		Checks: check.Checks{
			ImageChecks:      check.AllImageChecks(),
			FilesystemChecks: check.AllFilesystemChecks(),
			CatalogChecks:    check.AllCatalogChecks(),
		},
	},
	{
		Name:     "marketplace",
		ImageRef: "registry.redhat.io/redhat/redhat-marketplace-index:v4.18",
		Checks: check.Checks{
			ImageChecks:      check.AllImageChecks(),
			FilesystemChecks: check.AllFilesystemChecks(),
			CatalogChecks:    check.AllCatalogChecks(),
		},
	},
	{
		Name:     "certified",
		ImageRef: "registry.redhat.io/redhat/certified-operator-index:v4.18",
		Checks: check.Checks{
			ImageChecks:      check.AllImageChecks(),
			FilesystemChecks: check.AllFilesystemChecks(),
			CatalogChecks:    check.AllCatalogChecks(),
		},
	},
	{
		Name:     "redhat",
		ImageRef: "registry.redhat.io/redhat/redhat-operator-index:v4.18",
		Checks: check.Checks{
			ImageChecks:      check.AllImageChecks(),
			FilesystemChecks: check.AllFilesystemChecks(),
			CatalogChecks:    check.AllCatalogChecks(),
		},
	},
}
