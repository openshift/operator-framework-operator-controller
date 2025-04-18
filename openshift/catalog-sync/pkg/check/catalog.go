package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

func AllCatalogChecks() []CatalogCheck {
	return []CatalogCheck{
		BundleHasValidMetadataStructure(),
		ChannelHeadsHaveCSVMetadata(),
	}
}
func BundleHasValidMetadataStructure() CatalogCheck {
	return CatalogCheck{
		Name: "CheckBundleHasValidMetadataStructure",
		Fn: func(ctx context.Context, fbcFS fs.FS) error {
			const configsDir = "configs"

			entries, err := fs.ReadDir(fbcFS, configsDir)
			if err != nil {
				return fmt.Errorf("read configs dir: %w", err)
			}

			var errs []error

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				pkgPath := filepath.Join(configsDir, entry.Name())

				subFS, err := fs.Sub(fbcFS, pkgPath)
				if err != nil {
					errs = append(errs, fmt.Errorf("sub FS for %q: %w", pkgPath, err))
					continue
				}

				cfg, err := declcfg.LoadFS(ctx, subFS)
				if err != nil {
					errs = append(errs, fmt.Errorf("load declcfg from %q: %w", pkgPath, err))
					continue
				}

				for _, bundle := range cfg.Bundles {
					var hasCSV, hasObject bool
					for _, prop := range bundle.Properties {
						if prop.Type == "olm.csv.metadata" {
							hasCSV = true
						}
						if prop.Type == "olm.bundle.object" {
							hasObject = true
						}
					}

					if (hasCSV && hasObject) || (!hasCSV && !hasObject) {
						errs = append(errs, fmt.Errorf("bundle %q in package %q has both or neither of olm.bundle.object / olm.csv.metadata", bundle.Name, bundle.Package))
					}
				}
			}

			return errors.Join(errs...)
		},
	}
}

func ChannelHeadsHaveCSVMetadata() CatalogCheck {
	return CatalogCheck{
		Name: "CheckChannelHeadsHaveCSVMetadata",
		Fn: func(ctx context.Context, fbcFS fs.FS) error {
			const configsDir = "configs"

			entries, err := fs.ReadDir(fbcFS, configsDir)
			if err != nil {
				return fmt.Errorf("read configs dir: %w", err)
			}

			var errs []error

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				pkgPath := filepath.Join(configsDir, entry.Name())

				subFS, err := fs.Sub(fbcFS, pkgPath)
				if err != nil {
					errs = append(errs, fmt.Errorf("sub FS for %q: %w", pkgPath, err))
					continue
				}

				cfg, err := declcfg.LoadFS(ctx, subFS)
				if err != nil {
					errs = append(errs, fmt.Errorf("load declcfg from %q: %w", pkgPath, err))
					continue
				}

				models, err := declcfg.ConvertToModel(*cfg)
				if err != nil {
					errs = append(errs, fmt.Errorf("convert model for %q: %w", pkgPath, err))
					continue
				}

				// Get bundle names that are heads of any channel
				headBundleNames := map[string]bool{}
				for _, pkg := range models {
					for _, ch := range pkg.Channels {
						head, err := ch.Head()
						if err != nil {
							continue
						}
						headBundleNames[head.Name] = true
					}
				}

				// Check head bundles for olm.csv.metadata
				for _, bundle := range cfg.Bundles {
					if !headBundleNames[bundle.Name] {
						continue
					}

					hasCSV := false
					for _, prop := range bundle.Properties {
						if prop.Type == "olm.csv.metadata" {
							hasCSV = true
							break
						}
					}
					if !hasCSV {
						errs = append(errs, fmt.Errorf("head bundle %q in package %q is missing olm.csv.metadata", bundle.Name, bundle.Package))
					}
				}
			}

			return errors.Join(errs...)
		},
	}
}
