package catalog

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/oci/layout"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
)

// PullImageToOCI will pull the indexes images configured in the images.go
func PullImageToOCI(imageRef, ociDir string) error {
	ctx := context.Background()

	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return fmt.Errorf("parse image ref: %w", err)
	}

	destRef, err := layout.ParseReference(ociDir)
	if err != nil {
		return fmt.Errorf("parse destination OCI ref: %w", err)
	}

	// Required to allow run the tests on mac
	sysCtx := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "amd64",
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

	_, err = copy.Image(ctx, policyCtx, destRef, srcRef, &copy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
	})
	if err != nil {
		return fmt.Errorf("copy image to OCI layout: %w", err)
	}

	return nil
}

// GetImageLabelsFromOCI will get the labels from images
func GetImageLabelsFromOCI(ociDir string) (map[string]string, error) {
	ctx := context.Background()

	ref, err := layout.ParseReference(ociDir)
	if err != nil {
		return nil, fmt.Errorf("parse oci layout: %w", err)
	}

	src, err := ref.NewImageSource(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("create image source: %w", err)
	}
	defer src.Close()

	manifestBytes, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get manifest: %w", err)
	}

	mf, err := manifest.FromBlob(manifestBytes, manifest.GuessMIMEType(manifestBytes))
	if err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	configBlob, _, err := src.GetBlob(ctx, mf.ConfigInfo(), nil)
	if err != nil {
		return nil, fmt.Errorf("get config blob: %w", err)
	}
	defer configBlob.Close()

	var parsed struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"config"`
	}

	if err := json.NewDecoder(configBlob).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode config blob: %w", err)
	}

	return parsed.Config.Labels, nil
}

// ExtractFileSystemFromOCI extracts the full filesystem from all image layers.
func ExtractFileSystemFromOCI(ociDir string) (string, error) {
	ctx := context.Background()

	ref, err := layout.ParseReference(ociDir)
	if err != nil {
		return "", fmt.Errorf("parse oci layout: %w", err)
	}

	src, err := ref.NewImageSource(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("create image source: %w", err)
	}
	defer src.Close()

	manifestBytes, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("get manifest: %w", err)
	}

	mf, err := manifest.FromBlob(manifestBytes, manifest.GuessMIMEType(manifestBytes))
	if err != nil {
		return "", fmt.Errorf("decode manifest: %w", err)
	}

	outDir := filepath.Join(ociDir, "fs")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	for _, layer := range mf.LayerInfos() {
		rc, _, err := src.GetBlob(ctx, layer.BlobInfo, nil)
		if err != nil {
			return "", fmt.Errorf("get layer blob: %w", err)
		}

		if err := extractTarball(rc, outDir); err != nil {
			rc.Close()
			return "", fmt.Errorf("extract from layer: %w", err)
		}
		rc.Close()
	}

	return outDir, nil
}

// extractTarball extracts everything from a compressed tarball to dest.
func extractTarball(r io.Reader, dest string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}

	return nil
}
