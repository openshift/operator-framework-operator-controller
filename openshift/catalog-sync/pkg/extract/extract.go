package extract

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	imgcopy "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/oci/layout"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
)

// Result represents the extracted OCI content and related context
type Result struct {
	Store   oras.ReadOnlyTarget
	TmpDir  string
	Cleanup func()
}

// PrepareOCIImage pulls the image, extracts it to disk, and opens it as an OCI store.
func PrepareOCIImage(ctx context.Context, imageRef, name string) (res *Result, err error) {
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("oci-%s-", name))
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() { os.RemoveAll(tmpDir) }

	if err := pullImageToLayout(ctx, imageRef, tmpDir); err != nil {
		cleanup()
		return nil, fmt.Errorf("pull image: %w", err)
	}

	if err := extractFilesystem(ctx, tmpDir); err != nil {
		cleanup()
		return nil, fmt.Errorf("extract filesystem: %w", err)
	}

	store, err := oci.New(tmpDir)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("open OCI store: %w", err)
	}

	return &Result{
		Store:   store,
		TmpDir:  tmpDir,
		Cleanup: cleanup,
	}, nil
}

// pullImageToLayout pulls an image and writes it to OCI layout at the specified directory
func pullImageToLayout(ctx context.Context, imageRef, tmpDir string) error {
	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return fmt.Errorf("parse image ref: %w", err)
	}

	destRef, err := layout.ParseReference(fmt.Sprintf("%s:%s", tmpDir, "v1"))
	if err != nil {
		return fmt.Errorf("parse layout ref: %w", err)
	}

	policy := &signature.Policy{
		Default: []signature.PolicyRequirement{
			signature.NewPRInsecureAcceptAnything(),
		},
	}
	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("create policy context: %w", err)
	}
	defer policyCtx.Destroy()

	// Required to allow run the tests on Mac
	sysCtx := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "amd64",
	}

	if _, err := imgcopy.Image(ctx, policyCtx, destRef, srcRef, &imgcopy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
	}); err != nil {
		return fmt.Errorf("copy image to layout: %w", err)
	}

	return nil
}

// extractFilesystem extracts the filesystem layers from the OCI image into <tmpDir>/fs
func extractFilesystem(ctx context.Context, ociDir string) error {
	ref, err := layout.ParseReference(fmt.Sprintf("%s:%s", ociDir, "v1"))
	if err != nil {
		return fmt.Errorf("parse layout: %w", err)
	}
	src, err := ref.NewImageSource(ctx, nil)
	if err != nil {
		return fmt.Errorf("open image source: %w", err)
	}
	defer src.Close()

	manifestBytes, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		return fmt.Errorf("get manifest: %w", err)
	}

	mf, err := manifest.FromBlob(manifestBytes, manifest.GuessMIMEType(manifestBytes))
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	fsDir := filepath.Join(ociDir, "fs")
	if err := os.MkdirAll(fsDir, 0755); err != nil {
		return fmt.Errorf("make fs dir: %w", err)
	}

	for _, layer := range mf.LayerInfos() {
		rc, _, err := src.GetBlob(ctx, layer.BlobInfo, nil)
		if err != nil {
			return fmt.Errorf("get layer: %w", err)
		}
		if err := untarLayer(rc, fsDir); err != nil {
			rc.Close()
			return fmt.Errorf("extract layer: %w", err)
		}
		rc.Close()
	}

	return nil
}

// untarLayer decompresses and unpacks a gzip'd tarball into the given directory
func untarLayer(r io.Reader, dest string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		target := filepath.Join(dest, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", target, err)
			}
			out, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("copy data to %s: %w", target, err)
			}
			out.Close()
		}
	}

	return nil
}
