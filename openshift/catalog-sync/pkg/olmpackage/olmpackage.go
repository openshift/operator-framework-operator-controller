package olmpackage

import (
	"context"
	"fmt"
	"os"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

// Data load from the image (Package).
type Data struct {
	// DeclarativeConfig stored in the catalog for each package
	DeclarativeConfig declcfg.DeclarativeConfig
	// ChannelHeads obtained from the data
	ChannelHeads []ChannelHead
	// Image labels store the labels
	ImageLabels map[string]string
}

// ChannelHead represents the latest bundle version for a given channel.
type ChannelHead struct {
	ChannelName string
	Version     string
}

// NewDataFrom loads catalog data from a path
func NewDataFrom(path string, labels map[string]string) (*Data, error) {
	cfg, err := loadDeclarativeConfig(path)
	if err != nil {
		// If loading the catalog failed, return an error
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}

	return &Data{
		DeclarativeConfig: *cfg,
		ChannelHeads:      featchChannelHeads(cfg),
		ImageLabels:       labels,
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
