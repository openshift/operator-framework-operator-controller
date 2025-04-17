package validate

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/default-catalog-consistency/pkg/check"
	"github/operator-framework-operator-controller/openshift/default-catalog-consistency/pkg/extract"
	"github/operator-framework-operator-controller/openshift/default-catalog-consistency/test/utils"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validate Catalog Test Suite")
}

var _ = Describe("Check Catalog Consistency", func() {
	catalogsPath := "../../../catalogd/kustomize/overlays/openshift/catalogs"
	images, err := utils.LoadImagesFromCatalogsFrom(catalogsPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(images).ToNot(BeEmpty(), "no images found")

	for _, url := range images {
		name := utils.GetImageNameFrom(url)

		It(fmt.Sprintf("validates image: %s", name), func() {
			ctx := context.Background()
			By(fmt.Sprintf("Validating image: %s", url))

			res, err := extract.UnpackImage(ctx, url, name)
			Expect(err).ToNot(HaveOccurred())
			Expect(check.Check(ctx, res.Store, res.TmpDir, check.AllChecks())).To(Succeed())
			res.Cleanup()
		})
	}
})
