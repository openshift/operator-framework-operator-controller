package olmpackage

import (
	"encoding/json"
	"sort"

	"github.com/blang/semver/v4"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

// IsHeadOfChannel checks if a bundle is a head version in its channel.
func IsHeadOfChannel(bundle declcfg.Bundle, channelHeads []ChannelHead) bool {
	bundleChannel, bundleVersion, ok := parseChannelAndVersion(bundle)
	if !ok {
		return false
	}

	for _, head := range channelHeads {
		if head.ChannelName == bundleChannel && head.Version == bundleVersion {
			return true
		}
	}

	return false
}

// featchChannelHeads determines the latest bundle version for each channel.
func featchChannelHeads(cfg *declcfg.DeclarativeConfig) []ChannelHead {
	channelVersions := extractChannelVersions(cfg)

	var heads []ChannelHead
	for channelName, versions := range channelVersions {
		if latestVersion := getLatestVersion(versions); latestVersion != "" {
			heads = append(heads, ChannelHead{ChannelName: channelName, Version: latestVersion})
		}
	}

	return heads
}

// extractChannelVersions groups bundles by channel and collects versions.
func extractChannelVersions(cfg *declcfg.DeclarativeConfig) map[string][]semver.Version {
	channelVersions := make(map[string][]semver.Version)

	for _, bundle := range cfg.Bundles {
		channelName, bundleVersion, ok := parseChannelAndVersion(bundle)
		if !ok {
			continue
		}

		parsedVersion, err := semver.Parse(bundleVersion)
		if err == nil {
			channelVersions[channelName] = append(channelVersions[channelName], parsedVersion)
		}
	}

	return channelVersions
}

// getLatestVersion finds the highest semver version in a slice.
func getLatestVersion(versions []semver.Version) string {
	if len(versions) == 0 {
		return ""
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LT(versions[j])
	})

	return versions[len(versions)-1].String()
}

// parseChannelAndVersion extracts the channel name and version from a bundle.
func parseChannelAndVersion(bundle declcfg.Bundle) (string, string, bool) {
	var channelName, bundleVersion string

	for _, prop := range bundle.Properties {
		switch prop.Type {
		case "olm.channel":
			var value struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(prop.Value, &value); err == nil {
				channelName = value.Name
			}
		case "olm.bundle.version":
			if err := json.Unmarshal(prop.Value, &bundleVersion); err != nil {
				return "", "", false
			}
		}
	}

	if channelName == "" || bundleVersion == "" {
		return "", "", false
	}

	return channelName, bundleVersion, true
}
