package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

func AllFilesystemChecks() []FilesystemCheck {
	return []FilesystemCheck{
		FilesystemHasDirectories("configs", "tmp/cache"),
		FilesystemHasPogrebCache("tmp/cache"),
	}
}

func FilesystemHasDirectories(paths ...string) FilesystemCheck {
	return FilesystemCheck{
		Name: "FilesystemHasDirectories",
		Fn: func(ctx context.Context, imageFS fs.FS) error {
			var errs []error
			for _, path := range paths {
				stat, err := fs.Stat(imageFS, path)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				if !stat.IsDir() {
					errs = append(errs, fmt.Errorf("%q is not a directory", path))
				}
			}
			return errors.Join(errs...)
		},
	}
}

func FilesystemHasPogrebCache(cacheDir string) FilesystemCheck {
	return FilesystemCheck{
		Name: "FilesystemHasPogrebCache",
		Fn: func(ctx context.Context, imageFS fs.FS) error {
			pogrebCacheDir := filepath.Join(cacheDir, "pogreb.v1")
			stat, err := fs.Stat(imageFS, pogrebCacheDir)
			if err != nil {
				return err
			}
			if !stat.IsDir() {
				return fmt.Errorf("%q is not a directory", pogrebCacheDir)
			}
			return nil
		},
	}
}
