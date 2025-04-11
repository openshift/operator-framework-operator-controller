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
func SetupTestSuite() (map[string][]*olmpackage.Data, string) {
	tempDir, _ := os.MkdirTemp(".", "extract")
	catalogData := make(map[string][]*olmpackage.Data)

	for image, catalogName := range catalog.CatalogImages {
		fmt.Printf("Processing image: %s (%s)\n", image, catalogName)

		// Retry logic for pulling images
		ociPath := filepath.Join(tempDir, catalogName)
		Expect(retry(3, 5*time.Second, func() error {
			return catalog.PullImageToOCI(image, ociPath)
		})).To(Succeed(), fmt.Sprintf("failed to pull image %s", image))

		labels, err := catalog.GetImageLabelsFromOCI(ociPath)
		Expect(err).ToNot(HaveOccurred(), "failed to get image labels")

		configsPath, err := catalog.ExtractConfigsFromOCI(ociPath)
		Expect(err).ToNot(HaveOccurred(), "failed to extract configs")

		content, err := os.ReadDir(configsPath)
		Expect(err).ToNot(HaveOccurred(), "failed to read config dir")

		for _, entry := range content {
			if entry.IsDir() {
				fullPath := filepath.Join(configsPath, entry.Name())
				data, err := olmpackage.NewDataFrom(fullPath, labels)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to load package data from %s", fullPath))
				catalogData[catalogName] = append(catalogData[catalogName], data)
			}
		}
	}

	return catalogData, tempDir
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
