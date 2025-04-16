package test

import (
	"fmt"
	"github/operator-framework-operator-controller/openshift/catalog-sync/test/common"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

// checkChannelEdgePatternUsage verifies if a package has `skipRange`, `replaces`, or `skips`
func checkChannelEdgePatternUsage(catalogName string, cfgs []*olmpackage.Package) {
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

// channelNameUsage verifies how many packages have channels that follow `candidate/fast/stable` naming.
func channelNameUsage(catalogName string, cfgs []*olmpackage.Package) {
	totalPackages := 0
	packagesFollowingStandard := 0

	validPrefixes := map[string]struct{}{
		"candidate": {},
		"fast":      {},
		"stable":    {},
	}

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}

		for _, pkg := range cfg.DeclarativeConfig.Packages {
			totalPackages++
			channels := common.GetChannelsForPackage(pkg.Name, &cfg.DeclarativeConfig)

			hasStandardChannel := false
			for _, channel := range channels {
				channelParts := strings.Split(channel.Name, "-")
				if _, exists := validPrefixes[channelParts[0]]; exists {
					hasStandardChannel = true
					break
				}
			}

			if hasStandardChannel {
				packagesFollowingStandard++
			}
		}
	}

	res := result.TestResult{
		TestID:           "channel_naming_convention",
		TestContextTitle: "Packages with Channels Following `candidate/fast/stable` Naming",
		CatalogName:      catalogName,
		TotalPackages:    totalPackages,
		OptionalColumns: map[string]interface{}{
			"Total Following Conventional": packagesFollowingStandard,
			"% Following Conventional":     pkg.FormatPercentage(packagesFollowingStandard, totalPackages),
		},
	}

	testResults = append(testResults, res)
}

// checkSemverChannelUsage verifies how many packages have at least one channel with a SemVer name.
func checkSemverChannelUsage(catalogName string, cfgs []*olmpackage.Package) {
	totalPackages := 0
	packagesWithSemver := 0

	// Looking for: (MAJOR.MINOR.PATCH) and partial (MAJOR.MINOR)
	semverPattern := regexp.MustCompile(`v?\d+\.\d+(\.\d+)?(-[\w\d]+)?`)

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}

		for _, pkg := range cfg.DeclarativeConfig.Packages {
			totalPackages++
			channels := common.GetChannelsForPackage(pkg.Name, &cfg.DeclarativeConfig)

			hasSemverChannel := false
			for _, channel := range channels {
				if semverPattern.MatchString(channel.Name) {
					hasSemverChannel = true
					break
				}
			}

			if hasSemverChannel {
				packagesWithSemver++
			}
		}
	}

	res := result.TestResult{
		TestID:           "semver_channel_naming",
		TestContextTitle: "Packages with SemVer-Based Channel Naming",
		CatalogName:      catalogName,
		TotalPackages:    totalPackages,
		OptionalColumns: map[string]interface{}{
			"Total Using SemVer": packagesWithSemver,
			"% Using SemVer":     pkg.FormatPercentage(packagesWithSemver, totalPackages),
		},
	}

	testResults = append(testResults, res)
}

var _ = Describe("Pattern Inspection", func() {
	Context("Checking Graph Edge Patterns", func() {
		It("should validate edge pattern usage", func() {
			for _, image := range catalogImages {
				Expect(image).ToNot(BeNil(), fmt.Sprintf("No catalog data found for %s", image.Name))
				Expect(len(image.Packages)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", image.Name))
				checkChannelEdgePatternUsage(image.Name, image.Packages)
			}
		})
	})
	
	Context("Checking Channel Naming Patterns", func() {
		It("should check packages following `candidate/fast/stable` naming", func() {
			for _, image := range catalogImages {
				Expect(len(image.Packages)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", image.Name))
				channelNameUsage(image.Name, image.Packages)
			}
		})

		It("should check packages with SemVer-based channel naming", func() {
			for _, image := range catalogImages {
				Expect(len(image.Packages)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", image.Name))
				checkSemverChannelUsage(image.Name, image.Packages)
			}
		})
	})
})
