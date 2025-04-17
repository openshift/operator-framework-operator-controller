package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
)

// AllFilesystemChecks returns a list of filesystem checks to be performed on the image filesystem.
func AllFilesystemChecks() []FilesystemCheck {
	return []FilesystemCheck{
		FilesystemHasDirectories(
			"configs",
			"tmp/cache",
			"tmp/cache/pogreb.v1"),
	}
}

// FilesystemHasDirectories checks if the specified paths exist and are directories in the image filesystem.
func FilesystemHasDirectories(paths ...string) FilesystemCheck {
	return FilesystemCheck{
		Name: "FilesystemHasDirectories" + fmt.Sprintf(":(%q) ", paths),
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
