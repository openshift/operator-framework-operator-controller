package common

import (
	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

// GetChannelsForPackage returns all channels associated with a specific package
func GetChannelsForPackage(pkgName string, cfg *declcfg.DeclarativeConfig) []declcfg.Channel {
	var channels []declcfg.Channel
	for _, channel := range cfg.Channels {
		if channel.Package == pkgName {
			channels = append(channels, channel)
		}
	}
	return channels
}
