package test

import (
	"fmt"
	"github/operator-framework-operator-controller/openshift/catalog-sync/test/common"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/catalog"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"
	"github/operator-framework-operator-controller/openshift/catalog-sync/pkg/result"
)

// channelNameUsage verifies how many packages have channels that follow `candidate/fast/stable` naming.
func channelNameUsage(catalogName string, cfgs []*olmpackage.Data) {
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
func checkSemverChannelUsage(catalogName string, cfgs []*olmpackage.Data) {
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

var _ = Describe("Channel Conventional Naming Validation", func() {
	for image, catalogName := range catalog.CatalogImages {
		Context(fmt.Sprintf("Checking conventional channel name usage for Catalog %s (%s)", catalogName, image), func() {
			It(fmt.Sprintf("[%s] should check packages following `candidate/fast/stable` naming", catalogName), func() {
				cfgs, exists := catalogDataMap[catalogName]
				Expect(exists).To(BeTrue(), fmt.Sprintf("No catalog data found for %s", catalogName))
				Expect(len(cfgs)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", catalogName))

				channelNameUsage(catalogName, cfgs)
			})

			It(fmt.Sprintf("[%s] should check packages with SemVer-based channel naming", catalogName), func() {
				cfgs, exists := catalogDataMap[catalogName]
				Expect(exists).To(BeTrue(), fmt.Sprintf("No catalog data found for %s", catalogName))
				Expect(len(cfgs)).ToNot(BeZero(), fmt.Sprintf("No packages found for %s", catalogName))

				checkSemverChannelUsage(catalogName, cfgs)
			})
		})
	}
})
