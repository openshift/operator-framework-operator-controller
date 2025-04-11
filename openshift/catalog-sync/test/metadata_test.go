package test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-registry/alpha/declcfg"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/catalog"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

// checkHasBothOrNeither returns results to check if catalog has any bundle with both or neither
// (olm.bundle.object or olm.csv.metadata)
func checkHasBothOrNeither(catalog string, cfgs []*olmpackage.Data) {
	totalPackages := 0
	totalFailedPackages := 0

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}
		totalPackages++

		packageFailed := false
		for _, bundle := range cfg.DeclarativeConfig.Bundles {
			if hasBothOrNeither(bundle) {
				packageFailed = true
				break
			}
		}

		if packageFailed {
			totalFailedPackages++
		}
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
	}

	testResults = append(testResults, res)

	Expect(totalFailedPackages).To(BeZero(),
		"found failing packages for the catalog %s", catalog)
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

func checkHasCsvMetadata(catalog string, cfgs []*olmpackage.Data) {
	totalPackages := 0
	totalFailedPackages := 0
	headBundlesFailed := 0

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}
		totalPackages++

		packageFailed := false
		for _, bundle := range cfg.DeclarativeConfig.Bundles {
			if !hasCsvMetadata(bundle) {
				packageFailed = true

				if olmpackage.IsHeadOfChannel(bundle, cfg.ChannelHeads) {
					headBundlesFailed++
				}
			}
		}

		if packageFailed {
			totalFailedPackages++
		}
	}

	onlyHeads := "No"
	if totalFailedPackages > 0 && totalFailedPackages == headBundlesFailed {
		onlyHeads = "Yes"
	}

	failurePercentage := 0.0
	if totalPackages > 0 {
		failurePercentage = (float64(totalFailedPackages) / float64(totalPackages)) * 100
	}

	res := result.TestResult{
		TestID:               "must_have_csv_metadata",
		TestContextTitle:     "If any bundle does not have olm.csv.metadata, the entire package fails",
		CatalogName:          catalog,
		TotalPackages:        totalPackages,
		FailedPackages:       totalFailedPackages,
		FailurePercentage:    failurePercentage,
		IsPackageFailureType: true,
		OptionalColumns: map[string]interface{}{
			"Only Heads (y/n)": onlyHeads,
		},
	}

	testResults = append(testResults, res)
	Expect(totalFailedPackages).To(BeZero(),
		"found failing packages for the catalog %s", catalog)
}

// hasCsvMetadata returns true when a bundle does not have olm.csv.metadata
func hasCsvMetadata(bundle declcfg.Bundle) bool {
	var hasMetadata = false
	for _, prop := range bundle.Properties {
		if prop.Type == "olm.csv.metadata" {
			hasMetadata = true
			break
		}
	}
	return hasMetadata
}

var _ = Describe("If any bundle has both or neither (olm.bundle.object or olm.csv.metadata)", func() {
	for image, catalogName := range catalog.CatalogImages {
		Context(fmt.Sprintf("Check Catalog %s (%s)", catalogName, image), func() {
			It(fmt.Sprintf("[%s] should not have bundle(s) which have neither or both: (olm.bundle.object or olm.csv.metadata)", catalogName), func() {
				cfgs, exists := catalogDataMap[catalogName]
				Expect(exists).To(BeTrue(), fmt.Sprintf("No catalog data found for %s", catalogName))
				Expect(len(cfgs)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", catalogName))

				checkHasBothOrNeither(catalogName, cfgs)
			})

			It(fmt.Sprintf("[%s] should not have bundle(s) without olm.csv.metadata", catalogName), func() {
				cfgs, exists := catalogDataMap[catalogName]
				Expect(exists).To(BeTrue(), fmt.Sprintf("No catalog data found for %s", catalogName))
				Expect(len(cfgs)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", catalogName))

				checkHasCsvMetadata(catalogName, cfgs)
			})
		})
	}
})
