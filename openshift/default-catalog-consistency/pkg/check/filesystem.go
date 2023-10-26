package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// AllFilesystemChecks returns a list of filesystem checks to be performed on the image filesystem.
func AllFilesystemChecks() []FilesystemCheck {
	return []FilesystemCheck{
		FilesystemHasGoExecutables("bin/opm"),
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
		Fn: func(ctx context.Context, imageFS fs.FS, _ string) error {
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

// FilesystemHasGoExecutables checks that the given paths exist inside the extracted image filesystem.
// It supports regular files and symlinks:
//   - If it's a regular file, we just check that it exists.
//   - If it's a symlink, we check that the link target exists.
//
// This is useful for verifying container images where binaries may be symlinks,
// In our image builds (see:
// https://github.com/openshift/operator-framework-olm/blob/082d59a819afc43b24e9ca23c531bdfc35418722/operator-registry.Dockerfile#L16-L19),
// the actual binaries are in /bin/registry/*, and /bin/* just has symlinks that point to them.
func FilesystemHasGoExecutables(paths ...string) FilesystemCheck {
	return FilesystemCheck{
		Name: "FilesystemHasGoExecutables" + fmt.Sprintf(":(%q)", paths),

		// We use the `os` package instead of fs.FS because fs.FS does not support Lstat or Readlink,
		// which are needed to detect and resolve symlinks correctly.
		// Therefore, we are looking to real file paths under the unpacked image (in tmpDir/fs).
		Fn: func(_ context.Context, _ fs.FS, tmpDir string) error {
			var errs []error
			root := filepath.Join(tmpDir, "fs")

			for _, rel := range paths {
				fullPath := filepath.Join(root, rel)

				binaryPath, err := resolveImageSymlink(root, fullPath)
				if err != nil {
					errs = append(errs, fmt.Errorf("path %q: %w", rel, err))
					continue
				}

				out, err := exec.Command("go", "version", "-m", binaryPath).CombinedOutput()
				if err != nil {
					errs = append(errs, fmt.Errorf("not a Go binary or unreadable metadata for %q: %v\n%s", rel, err, out))
					continue
				}
			}

			return errors.Join(errs...)
		},
	}
}

// resolveImageSymlink resolve the symlink target.
// Example 1: if the target is absolute like "/bin/registry/opm",
// that means it should exist under the image root at tmpDir/fs/bin/registry/opm.
// Example 2: if the target is relative like "registry/opm" or "../registry/opm",
// resolve it relative to the symlink’s own directory.
func resolveImageSymlink(root, fullPath string) (string, error) {
	fi, err := os.Lstat(fullPath)
	if err != nil {
		return "", fmt.Errorf("error stating file: %w", err)
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return fullPath, nil
	}

	target, err := os.Readlink(fullPath)
	if err != nil {
		return "", fmt.Errorf("unreadable symlink: %w", err)
	}

	var resolved string
	if filepath.IsAbs(target) {
		// remove leading "/" and resolve from root
		resolved = filepath.Join(root, target[1:])
	} else {
		// If it's not a symlink, we just check that the file exists.
		resolved = filepath.Join(filepath.Dir(fullPath), target)
	}

	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("symlink points to missing target %q: %w", resolved, err)
	}

	return resolved, nil
}
