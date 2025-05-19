package extract

// TODO: Create a utility lib with the code to pull and extract the images
// used in this file to be reused for this project and others.
// More info: https://issues.redhat.com/browse/OPRUN-3919

import (
	"context"
	"fmt"
	imgcopy "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/oci/layout"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"os"
	"path/filepath"
)

// ExtractedImage represents the extracted OCI content and its temporary directory.
type ExtractedImage struct {
	Store  oras.ReadOnlyTarget
	TmpDir string
	Tag    string
}

// Cleanup function to remove the temporary directory
func (r *ExtractedImage) Cleanup() {
	if r.TmpDir != "" {
		if err := os.RemoveAll(r.TmpDir); err != nil {
			fmt.Printf("failed to remove temp dir: %v\n", err)
		}
	}
}

// UnpackImage pulls the image, extracts it to disk, and opens it as an OCI store.
func UnpackImage(ctx context.Context, imageRef, name string) (res *ExtractedImage, err error) {
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("oci-%s-", name))
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	var digestTag string

	res, err = func() (*ExtractedImage, error) {
		srcRef, err := docker.ParseReference("//" + imageRef)
		if err != nil {
			return nil, fmt.Errorf("parse image ref: %w", err)
		}

		// Force image resolution to Linux to avoid OS mismatch errors on macOS,
		// like: "no image found for architecture 'arm64', OS 'darwin'".
		//
		// Setting OSChoice = "linux" ensures we always get a Linux image,
		// even when running on macOS.
		//
		// This skips the full multi-arch index and gives us just one manifest.
		// To check all supported architectures (e.g., amd64, arm64, ppc64le, s390x),
		// we’d need to avoid setting OSChoice and inspect the full index manually.
		//
		// TODO: Update this to support checking all architectures.
		// See: https://issues.redhat.com/browse/OPRUN-3793
		sysCtx := &types.SystemContext{
			OSChoice: "linux",
		}

		if authPath := os.Getenv("REGISTRY_AUTH_FILE"); authPath != "" {
			fmt.Println("Using registry auth file:", authPath)
			sysCtx.AuthFilePath = authPath
		}

		policyCtx, err := loadPolicyContext(sysCtx, imageRef)
		if err != nil {
			return nil, fmt.Errorf("create policy context: %w", err)
		}
		defer policyCtx.Destroy()

		canonicalRef, err := resolveCanonicalRef(ctx, srcRef, sysCtx)
		if err != nil {
			return nil, fmt.Errorf("resolve canonical ref: %w", err)
		}

		digestTag = canonicalRef.String()

		// Create subdirectory for the OCI layout
		ociDir := filepath.Join(tmpDir, "oci")
		if err := os.MkdirAll(ociDir, 0755); err != nil {
			return nil, fmt.Errorf("create oci layout dir: %w", err)
		}

		// Destination reference within OCI layout
		destRef, err := layout.ParseReference(fmt.Sprintf("%s:%s", ociDir, digestTag))
		if err != nil {
			return nil, fmt.Errorf("parse layout ref: %w", err)
		}

		// Pull and copy the image to the temporary directory
		if _, err := imgcopy.Image(ctx, policyCtx, destRef, srcRef, &imgcopy.Options{
			SourceCtx:      sysCtx,
			DestinationCtx: sysCtx,
		}); err != nil {
			return nil, fmt.Errorf("copy image: %w", err)
		}

		// Create and use a separate fs directory for unpacked layers
		fsDir := filepath.Join(tmpDir, "fs")
		if err := extractLayers(ctx, ociDir, fsDir, digestTag); err != nil {
			return nil, fmt.Errorf("extract filesystem: %w", err)
		}

		// Open the OCI layout from the correct layout directory
		store, err := oci.New(ociDir)
		if err != nil {
			return nil, fmt.Errorf("open OCI store: %w", err)
		}

		return &ExtractedImage{
			Store:  store,
			TmpDir: tmpDir,
			Tag:    digestTag,
		}, nil
	}()

	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	return res, nil
}

// extractLayers extracts the filesystem layers from the OCI image layout under the given digest tag.
func extractLayers(ctx context.Context, layoutPath, fsPath, tag string) error {
	ref, err := layout.ParseReference(fmt.Sprintf("%s:%s", layoutPath, tag))
	if err != nil {
		return fmt.Errorf("parse layout: %w", err)
	}
	src, err := ref.NewImageSource(ctx, nil)
	if err != nil {
		return fmt.Errorf("open image source: %w", err)
	}
	defer src.Close()

	manifests, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		return fmt.Errorf("get manifest: %w", err)
	}

	mf, err := manifest.FromBlob(manifests, manifest.GuessMIMEType(manifests))
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	if err := os.MkdirAll(fsPath, 0755); err != nil {
		return fmt.Errorf("make fs dir to unpack: %w", err)
	}

	for i, layer := range mf.LayerInfos() {
		rc, _, err := src.GetBlob(ctx, layer.BlobInfo, nil)
		if err != nil {
			return fmt.Errorf("get blob %d: %w", i, err)
		}

		decompress, _, err := compression.AutoDecompress(rc)
		if err != nil {
			rc.Close()
			return fmt.Errorf("decompress blob %d: %w", i, err)
		}

		// To avoid permission errors faced when extracting the layers
		mask := os.FileMode(0755)
		opts := &archive.TarOptions{
			ForceMask: &mask,
			// Required to avoid permission errors when extracting the layers in the OCP CI environment.
			// extract filesystem: apply layer 0: lchown /tmp/.../afs: operation not permitted
			NoLchown: true,
		}

		_, err = archive.ApplyUncompressedLayer(fsPath, decompress, opts)
		decompress.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("apply layer %d: %w", i, err)
		}
	}
	return nil
}

// resolveCanonicalRef resolves the canonical reference for the given image reference.
// If the image reference is already a canonical reference, it returns it directly.
// Otherwise, it retrieves the manifest from the image source and creates a canonical reference
// using the digest of the manifest.
// Same code implementation from: operator-framework-operator-controller/internal/shared/util/image/pull.go
func resolveCanonicalRef(ctx context.Context, imgRef types.ImageReference, srcCtx *types.SystemContext) (reference.Canonical, error) {
	if canonicalRef, ok := imgRef.DockerReference().(reference.Canonical); ok {
		return canonicalRef, nil
	}

	imgSrc, err := imgRef.NewImageSource(ctx, srcCtx)
	if err != nil {
		return nil, fmt.Errorf("error creating image source: %w", err)
	}
	defer imgSrc.Close()

	manifestBlob, _, err := imgSrc.GetManifest(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting manifest: %w", err)
	}
	imgDigest, err := manifest.Digest(manifestBlob)
	if err != nil {
		return nil, fmt.Errorf("error getting digest of manifest: %w", err)
	}
	canonicalRef, err := reference.WithDigest(reference.TrimNamed(imgRef.DockerReference()), imgDigest)
	if err != nil {
		return nil, fmt.Errorf("error creating canonical reference: %w", err)
	}
	return canonicalRef, nil
}

// loadPolicyContext loads the signature verification policy for pulling the image.
func loadPolicyContext(sourceContext *types.SystemContext, imageRef string) (*signature.PolicyContext, error) {
	policy, err := signature.DefaultPolicy(sourceContext)

	// Allow we run without a policy file configured
	// if we need to validate the image signature then we will need to
	// change it.
	if err != nil {
		fmt.Println(fmt.Sprintf("no default policy found for (%s), using insecure policy", imageRef))
		insecurePolicy := []byte(`{
			"default": [{"type": "insecureAcceptAnything"}]
		}`)
		policy, err = signature.NewPolicyFromBytes(insecurePolicy)
	}

	if err != nil {
		return nil, fmt.Errorf("error loading signature policy: %w", err)
	}

	return signature.NewPolicyContext(policy)
}
