package common

import (
	"fmt"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
	"time"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/catalog"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
)

// SetupTestSuite initializes the test environment for a suite.
func SetupTestSuite() ([]*catalog.Image, string) {
	tempDir, _ := os.MkdirTemp(".", "extract")
	var images []*catalog.Image

	for image, catalogName := range catalog.CatalogImages {
		fmt.Printf("Processing image: %s (%s)\n", image, catalogName)

		ociPath := filepath.Join(tempDir, catalogName)
		Expect(retry(3, 5*time.Second, func() error {
			return catalog.PullImageToOCI(image, ociPath)
		})).To(Succeed(), fmt.Sprintf("failed to pull image %s", image))

		labels, err := catalog.GetImageLabelsFromOCI(ociPath)
		Expect(err).ToNot(HaveOccurred(), "failed to get image labels")

		fsPath, err := catalog.ExtractFileSystemFromOCI(ociPath)
		Expect(err).ToNot(HaveOccurred(), "failed to extract image filesystem")

		configsPath := filepath.Join(fsPath, "configs")
		content, err := os.ReadDir(configsPath)
		Expect(err).ToNot(HaveOccurred(), "failed to read configs directory")

		var packages []*olmpackage.Package
		for _, entry := range content {
			if entry.IsDir() {
				pkgPath := filepath.Join(configsPath, entry.Name())
				pkg, err := olmpackage.NewPackageDataFrom(pkgPath)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to load package data from %s", pkgPath))
				packages = append(packages, pkg)
			}
		}

		image := &catalog.Image{
			Name:     catalogName,
			Labels:   labels,
			Packages: packages,
		}
		images = append(images, image)
	}

	return images, tempDir
}

// Retry retries the given function for `attempts` times with `delay` between tries.
func retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}
