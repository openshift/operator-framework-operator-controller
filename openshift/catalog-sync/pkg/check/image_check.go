package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"oras.land/oras-go/v2"
	"os"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Checks struct {
	ImageChecks      []ImageCheck
	FilesystemChecks []FilesystemCheck
	CatalogChecks    []CatalogCheck
}

type ImageCheckFunc func(ctx context.Context, root ocispec.Descriptor, target oras.ReadOnlyTarget) error
type FilesystemCheckFunc func(ctx context.Context, imageFS fs.FS) error
type CatalogCheckFunc func(ctx context.Context, fbcFS fs.FS) error

type ImageCheck struct {
	Name string
	Fn   ImageCheckFunc
}

type FilesystemCheck struct {
	Name string
	Fn   FilesystemCheckFunc
}

type CatalogCheck struct {
	Name string
	Fn   CatalogCheckFunc
}

type Error struct {
	CheckName string
	Err       error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.CheckName, e.Err)
}

func Check(ctx context.Context, tag string, store oras.ReadOnlyTarget, ociDir string, checks Checks) error {
	desc, err := store.Resolve(ctx, tag)
	if err != nil {
		return fmt.Errorf("resolve descriptor: %w", err)
	}

	unpackPath := filepath.Join(ociDir, "fs")
	if _, err := os.Stat(unpackPath); err != nil {
		return fmt.Errorf("expected extracted filesystem at %q: %w", unpackPath, err)
	}

	var checkErrors []error

	for _, check := range checks.ImageChecks {
		if err := check.Fn(ctx, desc, store); err != nil {
			checkErrors = append(checkErrors, Error{CheckName: fmt.Sprintf("Image:%s", check.Name), Err: err})
		}
	}
	for _, check := range checks.FilesystemChecks {
		if err := check.Fn(ctx, os.DirFS(unpackPath)); err != nil {
			checkErrors = append(checkErrors, Error{CheckName: fmt.Sprintf("FS:%s", check.Name), Err: err})
		}
	}
	for _, check := range checks.CatalogChecks {
		if err := check.Fn(ctx, os.DirFS(unpackPath)); err != nil {
			checkErrors = append(checkErrors, Error{CheckName: fmt.Sprintf("FBC:%s", check.Name), Err: err})
		}
	}

	return errors.Join(checkErrors...)
}
