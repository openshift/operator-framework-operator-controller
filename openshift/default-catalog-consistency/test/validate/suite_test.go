package validate

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-controller/openshift/default-catalog-consistency/pkg/check"
	"github.com/operator-framework/operator-controller/openshift/default-catalog-consistency/pkg/extract"
	"github.com/operator-framework/operator-controller/openshift/default-catalog-consistency/test/utils"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validate Catalog Test Suite")
}

// images is a list of image references to be validated.
var images = []string{
	"registry.redhat.io/redhat/community-operator-index:v4.17",
	"registry.redhat.io/redhat/redhat-marketplace-index:v4.17",
	"registry.redhat.io/redhat/certified-operator-index:v4.17",
	"registry.redhat.io/redhat/redhat-operator-index:v4.17",
}

var _ = Describe("Check Catalog Consistency", func() {
	for _, url := range images {
		name := utils.ImageNameFromRef(url)

		It(fmt.Sprintf("validates image: %s", name), func() {
			ctx := context.Background()
			By(fmt.Sprintf("Validating image: %s", url))

			extractedImage, err := extract.UnpackImage(ctx, url, name)
			Expect(err).ToNot(HaveOccurred())
			Expect(check.Check(ctx, extractedImage, check.AllChecks())).To(Succeed())
			extractedImage.Cleanup()
		})
	}
})
