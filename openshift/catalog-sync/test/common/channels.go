package common

import (
	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

// GetChannelsForPackage returns all channels associated with a specific package
func GetChannelsForPackage(pkgName string, cfg *declcfg.DeclarativeConfig) []declcfg.Channel {
	var relevantChannels []declcfg.Channel
	for _, channel := range cfg.Channels {
		if channel.Package == pkgName {
			relevantChannels = append(relevantChannels, channel)
		}
	}
	return relevantChannels
}
