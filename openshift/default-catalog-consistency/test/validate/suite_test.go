package validate

import (
	"context"
	"fmt"
	"github.com/containers/image/v5/types"
	"os"
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
	images, err := utils.ParseImageRefsFromCatalog(catalogsPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(images).ToNot(BeEmpty(), "no images found")
	authPath := os.Getenv("REGISTRY_AUTH_FILE")

	// Force image resolution to Linux to avoid OS mismatch errors on macOS,
	// like: "no image found for architecture 'arm64', OS 'darwin'".
	//
	// Setting OSChoice = "linux" ensures we always get a Linux image,
	// even when running on macOS.
	//
	// This skips the full multi-arch index and gives us just one manifest.
	// To check all supported architectures (e.g., amd64, arm64, ppc64le, s390x),
	// we’d need to avoid setting OSChoice and inspect the full index manually.
	//
	// TODO: Update this to support checking all architectures.
	// See: https://issues.redhat.com/browse/OPRUN-3793
	sysCtx := &types.SystemContext{
		OSChoice: "linux",
	}
	if authPath != "" {
		fmt.Println("Using registry auth file:", authPath)
		sysCtx.AuthFilePath = authPath
	}

	for _, url := range images {
		name := utils.ImageNameFromRef(url)

		It(fmt.Sprintf("validates image: %s", name), func() {
			ctx := context.Background()
			By(fmt.Sprintf("Validating image: %s", url))

			extractedImage, err := extract.UnpackImage(ctx, url, name, sysCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(check.Check(ctx, extractedImage, check.AllChecks())).To(Succeed())
			extractedImage.Cleanup()
		})
	}
})
