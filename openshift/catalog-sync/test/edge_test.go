package test

import (
	"fmt"
	"github/operator-framework-operator-controller/openshift/catalog-sync/test/common"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/catalog"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

// checkChannelEdgePatternUsage verifies if a package has `skipRange`, `replaces`, or `skips`
func checkChannelEdgePatternUsage(catalogName string, cfgs []*olmpackage.Data) {
	totalPackages := 0
	count := map[string]int{
		"skipRange": 0,
		"replaces":  0,
		"skips":     0,
	}

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}

		for _, pkg := range cfg.DeclarativeConfig.Packages {
			totalPackages++

			hasSkipRange, hasReplaces, hasSkips := false, false, false
			channels := common.GetChannelsForPackage(pkg.Name, &cfg.DeclarativeConfig)

			for _, channel := range channels {
				for _, entry := range channel.Entries {
					hasSkipRange = hasSkipRange || entry.SkipRange != ""
					hasReplaces = hasReplaces || entry.Replaces != ""
					hasSkips = hasSkips || len(entry.Skips) > 0
				}
			}

			if hasSkipRange {
				count["skipRange"]++
			}
			if hasReplaces {
				count["replaces"]++
			}
			if hasSkips {
				count["skips"]++
			}
		}
	}

	res := result.TestResult{
		TestID:           "edge_usage",
		TestContextTitle: "Channel Edge Graph Usage (skipRange, replaces, skips) by Package",
		CatalogName:      catalogName,
		TotalPackages:    totalPackages,
		OptionalColumns: map[string]interface{}{
			"Total With skipRange": count["skipRange"],
			"% of With skipRange":  pkg.FormatPercentage(count["skipRange"], totalPackages),
			"Total With replaces":  count["replaces"],
			"% of With replaces":   pkg.FormatPercentage(count["replaces"], totalPackages),
			"Total With skips":     count["skips"],
			"% of With skips":      pkg.FormatPercentage(count["skips"], totalPackages),
		},
	}

	testResults = append(testResults, res)
}

var _ = Describe("Channel Graph Edge Pattern Validation", func() {
	for image, catalogName := range catalog.CatalogImages {
		Context(fmt.Sprintf("Checking Graph Edge Patterns in Catalog %s (%s)", catalogName, image), func() {
			It(fmt.Sprintf("[%s] should validate edge pattern usage", catalogName), func() {
				cfgs, exists := catalogDataMap[catalogName]
				Expect(exists).To(BeTrue(), fmt.Sprintf("No catalog data found for %s", catalogName))
				Expect(len(cfgs)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", catalogName))

				checkChannelEdgePatternUsage(catalogName, cfgs)
			})
		})
	}
})
