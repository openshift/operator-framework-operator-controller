package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	specsgov1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

// AllImageChecks returns a list of image checks to be performed on the image.
func AllImageChecks() []ImageCheck {
	return []ImageCheck{
		ImageIsValidManifest(),
		ImageHasLabels(map[string]string{
			"operators.operatorframework.io.index.configs.v1": "/configs",
		}),
	}
}

// ImageIsValidManifest checks if the image is a valid manifest.
func ImageIsValidManifest() ImageCheck {
	return ImageCheck{
		Name: "ImageIsValidManifest",
		Fn: func(ctx context.Context, root specsgov1.Descriptor, target oras.ReadOnlyTarget) error {
			manifestReader, err := target.Fetch(ctx, root)
			if err != nil {
				return err
			}
			defer manifestReader.Close()
			var img specsgov1.Manifest
			if err := json.NewDecoder(manifestReader).Decode(&img); err != nil {
				return err
			}
			if err := isValidMediaType(img); err != nil {
				return err
			}
			return nil
		},
	}
}

// ImageHasLabels checks if the image has the expected labels.
func ImageHasLabels(expectedLabels map[string]string) ImageCheck {
	if len(expectedLabels) == 0 {
		return ImageCheck{
			Name: "ImageHasExpectedLabels[error: no labels provided]",
			Fn: func(ctx context.Context, root specsgov1.Descriptor, target oras.ReadOnlyTarget) error {
				return errors.New("ImageHasLabels: expectedLabels must not be empty")
			},
		}
	}

	pairs := make([]string, 0, len(expectedLabels))
	for k, v := range expectedLabels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(pairs)
	name := fmt.Sprintf("ImageHasExpectedLabels[%s]", strings.Join(pairs, ","))

	return ImageCheck{
		Name: name,
		Fn: func(ctx context.Context, root specsgov1.Descriptor, target oras.ReadOnlyTarget) error {
			fetch, err := target.Fetch(ctx, root)
			if err != nil {
				return err
			}
			defer fetch.Close()
			var m specsgov1.Manifest
			if err := json.NewDecoder(fetch).Decode(&m); err != nil {
				return err
			}
			if err := isValidMediaType(m); err != nil {
				return err
			}
			reader, err := target.Fetch(ctx, m.Config)
			if err != nil {
				return err
			}
			defer reader.Close()

			var img specsgov1.Image
			if err := json.NewDecoder(reader).Decode(&img); err != nil {
				return err
			}

			if len(img.Config.Labels) == 0 {
				return errors.New("image has no labels at all")
			}

			var errs []error
			for expectedLabel, expectedValue := range expectedLabels {
				actualValue, ok := img.Config.Labels[expectedLabel]
				if !ok {
					errs = append(errs, fmt.Errorf("missing label: %s", expectedLabel))
					continue
				}
				if actualValue != expectedValue {
					errs = append(errs,
						fmt.Errorf("label %q: expected %q, got %q",
							expectedLabel, expectedValue, actualValue))
				}
			}

			return errors.Join(errs...)
		},
	}
}

// RequiredPlatforms is a list of platforms that are required for the image to be considered valid.
var RequiredPlatforms = []specsgov1.Platform{
	{OS: "linux", Architecture: "amd64"},
	{OS: "linux", Architecture: "arm64"},
	{OS: "linux", Architecture: "ppc64le"},
	{OS: "linux", Architecture: "s390x"},
}

// ImageSupportsMultiArch verifies multi-arch support by inspecting the remote image manifest
// list directly. We don’t use extract.UnpackImage and the check interfaces here because it picks
// one platform based on OSChoice. On macOS, this fails if the image doesn’t support darwin/arm64
// or darwin/amd64 and to make easier develop and test on macOS we keep the check independent of
// the unpacking logic where we set OSChoice = "linux" to avoid OS mismatch errors.
func ImageSupportsMultiArch(imageRef string, expected []specsgov1.Platform, sysCtx *types.SystemContext) ImageCheck {
	return ImageCheck{
		Name: "ImageSupportsMultiArch",
		// Do not use the unpacked image, but pull the manifest list directly from the remote.
		Fn: func(ctx context.Context, _ specsgov1.Descriptor, _ oras.ReadOnlyTarget) error {
			ref, err := docker.ParseReference("//" + imageRef)
			if err != nil {
				return fmt.Errorf("parse image ref: %w", err)
			}

			src, err := ref.NewImageSource(ctx, sysCtx)
			if err != nil {
				return fmt.Errorf("new image source: %w", err)
			}
			defer src.Close()

			manifestBytes, _, err := src.GetManifest(ctx, nil)
			if err != nil {
				return fmt.Errorf("get manifest: %w", err)
			}

			mf, err := manifest.Schema2ListFromManifest(manifestBytes)
			if err != nil {
				return fmt.Errorf("parse multiarch list: %w", err)
			}

			found := map[string]struct{}{}
			for _, desc := range mf.Manifests {
				key := fmt.Sprintf("%s/%s", desc.Platform.OS, desc.Platform.Architecture)
				found[key] = struct{}{}
			}

			var missing []string
			for _, p := range expected {
				key := fmt.Sprintf("%s/%s", p.OS, p.Architecture)
				if _, ok := found[key]; !ok {
					missing = append(missing, key)
				}
			}

			if len(missing) > 0 {
				return fmt.Errorf("missing required platforms: %v", missing)
			}
			return nil
		},
	}
}

func isValidMediaType(m specsgov1.Manifest) error {
	switch m.MediaType {
	case specsgov1.MediaTypeImageManifest, manifest.DockerV2Schema2MediaType:
	default:
		return fmt.Errorf("unrecognized manifest type: %s", m.MediaType)
	}
	return nil
}
