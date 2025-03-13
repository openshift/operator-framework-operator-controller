package olmpackage

import (
	"context"
	"fmt"
	"os"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

// Package holds parsed catalog data per package.
type Package struct {
	DeclarativeConfig declcfg.DeclarativeConfig
	HeadChannels      []HeadChannels
}

// HeadChannels holds a single head-of-channel record.
type HeadChannels struct {
	ChannelName string
	BundleName  string
}

// NewPackageDataFrom loads catalog data from a path and computes head bundles.
func NewPackageDataFrom(path string) (*Package, error) {
	cfg, err := loadDeclarativeConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}

	models, err := declcfg.ConvertToModel(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to model: %w", err)
	}

	var heads []HeadChannels
	for _, pkg := range models {
		for _, ch := range pkg.Channels {
			head, err := ch.Head()
			if err != nil {
				continue
			}
			heads = append(heads, HeadChannels{
				ChannelName: ch.Name,
				BundleName:  head.Name,
			})
		}
	}

	return &Package{
		DeclarativeConfig: *cfg,
		HeadChannels:      heads,
	}, nil
}

// loadDeclarativeConfig reads and parses catalog data from the filesystem.
func loadDeclarativeConfig(path string) (*declcfg.DeclarativeConfig, error) {
	fileSystem := os.DirFS(path)
	cfg, err := declcfg.LoadFS(context.TODO(), fileSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}
	return cfg, nil
}

// IsHeadOfChannel checks if a given bundle is a head of any of its package's channels.
func IsHeadOfChannel(bundle declcfg.Bundle, heads []HeadChannels) bool {
	for _, head := range heads {
		if head.BundleName == bundle.Name {
			return true
		}
	}
	return false
}
