package test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-registry/alpha/declcfg"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

// hasInvalidBundleObjectMetadata returns results to check if catalog has any bundle with both or neither
// (olm.bundle.object or olm.csv.metadata)
func hasInvalidBundleObjectMetadata(catalog string, cfgs []*olmpackage.Package) bool {
	totalPackages := 0
	totalFailedPackages := 0
	uniqueFailedPackages := map[string]bool{}

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}
		totalPackages++

		packageFailed := false
		for _, bundle := range cfg.DeclarativeConfig.Bundles {
			if hasBothOrNeither(bundle) {
				packageFailed = true
				uniqueFailedPackages[bundle.Package] = true
				fmt.Println(fmt.Sprintf("[LOG] - [%s] - Bundle (%s) with both or neither "+
					"(olm.bundle.object or olm.csv.metadata) found in package %s",
					catalog,
					bundle.Name,
					bundle.Package))
				break
			}
		}

		if packageFailed {
			totalFailedPackages++
		}
	}

	var failedPackagesNames []string
	for name := range uniqueFailedPackages {
		failedPackagesNames = append(failedPackagesNames, name)
	}

	failurePercentage := 0.0
	if totalPackages > 0 {
		failurePercentage = (float64(totalFailedPackages) / float64(totalPackages)) * 100
	}

	res := result.TestResult{
		TestID:               "both_or_neither",
		TestContextTitle:     "If any bundle has both or neither (olm.bundle.object or olm.csv.metadata), the entire package fails",
		CatalogName:          catalog,
		TotalPackages:        totalPackages,
		FailedPackages:       totalFailedPackages,
		FailurePercentage:    failurePercentage,
		IsPackageFailureType: true,
		FailedPackageNames:   failedPackagesNames,
	}

	testResults = append(testResults, res)
	if totalFailedPackages > 0 {
		return true
	}
	return false
}

// hasBothOrNeither returns true when a bundle has both or neither
// (olm.bundle.object or olm.csv.metadata)
func hasBothOrNeither(bundle declcfg.Bundle) bool {
	var hasObject, hasMetadata bool

	for _, prop := range bundle.Properties {
		if prop.Type == "olm.bundle.object" {
			hasObject = true
		}
		if prop.Type == "olm.csv.metadata" {
			hasMetadata = true
		}
	}
	return (hasObject && hasMetadata) || (!hasObject && !hasMetadata)
}

func hasHeadOfChannelWithoutCsvMetadadata(catalog string, packages []*olmpackage.Package) bool {
	uniqueFailedPackages := map[string]bool{}
	totalPackages := 0
	totalFailedPackages := 0

	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		totalPackages++
		packageFailed := false
		for _, bundle := range pkg.DeclarativeConfig.Bundles {
			if !olmpackage.IsHeadOfChannel(bundle, pkg.HeadChannels) {
				continue
			}
			if !hasCsvMetadata(bundle) {
				uniqueFailedPackages[bundle.Package] = true
				packageFailed = true
				fmt.Println(fmt.Sprintf("[LOG] - [%s] - Head of Channel (%s) "+
					" and is missing \"olm.csv.metadata\" for the package %s",
					catalog,
					bundle.Name,
					bundle.Package))
			}
		}
		if packageFailed {
			totalFailedPackages++
		}
	}

	var failedPackagesNames []string
	for name := range uniqueFailedPackages {
		failedPackagesNames = append(failedPackagesNames, name)
	}

	failurePercentage := 0.0
	if totalPackages > 0 {
		failurePercentage = (float64(totalFailedPackages) / float64(totalPackages)) * 100
	}

	testResults = append(testResults, result.TestResult{
		TestID:               "must_have_csv_metadata",
		TestContextTitle:     "If any bundle which is head of channel does not have olm.csv.metadata, the entire package fails",
		CatalogName:          catalog,
		TotalPackages:        len(packages),
		FailedPackages:       totalFailedPackages,
		FailurePercentage:    failurePercentage,
		IsPackageFailureType: true,
		FailedPackageNames:   failedPackagesNames,
	})

	if totalFailedPackages > 0 {
		return true
	}
	return false
}

// hasCsvMetadata returns true when a bundle has olm.csv.metadata
func hasCsvMetadata(bundle declcfg.Bundle) bool {
	for _, prop := range bundle.Properties {
		if prop.Type == "olm.csv.metadata" {
			return true
		}
	}
	return false
}

var _ = Describe("Metadata Bundle Validation", func() {
	Context("Check olm.bundle.object and olm.csv.metadata settings", func() {
		It("should not have bundle(s) which have neither or both: (olm.bundle.object or olm.csv.metadata)", func() {
			hasFailures := false
			for _, image := range catalogImages {
				Expect(len(image.Packages)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", image.Name))
				if hasInvalidBundleObjectMetadata(image.Name, image.Packages) {
					hasFailures = true
				}
			}
			Expect(hasFailures).To(BeFalse(), "One or more catalogs failed because a bundle has both or neither (olm.bundle.object or olm.csv.metadata)")
		})
		It("should not have head of channel bundle(s) without olm.csv.metadata", func() {
			hasFailures := false
			for _, image := range catalogImages {
				Expect(len(image.Packages)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", image.Name))
				if hasHeadOfChannelWithoutCsvMetadadata(image.Name, image.Packages) {
					hasFailures = true
				}
			}
			Expect(hasFailures).To(BeFalse(), "One or more catalogs failed due be missing olm.csv.metadata in head of channels")
		})
	})
})
