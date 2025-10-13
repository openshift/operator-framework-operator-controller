package validate

import (
	"context"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containers/image/v5/types"
	specsgov1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github/operator-framework-operator-controller/openshift/default-catalog-consistency/pkg/check"
	"github/operator-framework-operator-controller/openshift/default-catalog-consistency/pkg/extract"
	"github/operator-framework-operator-controller/openshift/default-catalog-consistency/test/utils"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OLM-Catalog-Validation")
}

var _ = Describe("OLM-Catalog-Validation", func() {
	catalogsPath := "../../../catalogd/manifests.yaml"
	images, err := utils.ParseImageRefsFromCatalog(catalogsPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(images).ToNot(BeEmpty(), "no images found")
	authPath := os.Getenv("REGISTRY_AUTH_FILE")

	sysCtx := &types.SystemContext{}
	if authPath != "" {
		fmt.Println("Using registry auth file:", authPath)
		sysCtx.AuthFilePath = authPath
	}

	for _, url := range images {
		name := utils.ImageNameFromRef(url)
		ctx := context.Background()

		It(fmt.Sprintf("validates multiarch support for image: %s", name), func() {
			By(fmt.Sprintf("Validating image: %s", url))
			err := check.ImageSupportsMultiArch(
				url,
				check.RequiredPlatforms,
				sysCtx,
			).Fn(ctx, specsgov1.Descriptor{}, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It(fmt.Sprintf("validates image: %s", name), func() {
			By(fmt.Sprintf("Validating image: %s", url))
			// Force image resolution to Linux to avoid OS mismatch errors on macOS,
			// like: "no image found for architecture 'arm64', OS 'darwin'".
			//
			// Setting OSChoice = "linux" ensures we always get a Linux image,
			// even when running on macOS.
			//
			// This skips the full multi-arch index and gives us just one manifest.
			sysCtx.OSChoice = "linux"

			extractedImage, err := extract.UnpackImage(ctx, url, name, sysCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(check.Check(ctx, extractedImage, check.AllChecks())).To(Succeed())
			extractedImage.Cleanup()
		})
	}
})
