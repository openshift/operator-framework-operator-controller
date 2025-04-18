package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/containers/image/v5/manifest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

func AllImageChecks() []ImageCheck {
	return []ImageCheck{
		ImageIsManifest(),
		ImageHasLabels(map[string]string{
			"operators.operatorframework.io.index.configs.v1": "/configs",
		}),
	}
}

func ImageIsManifest() ImageCheck {
	return ImageCheck{
		Name: "IsImageManifest",
		Fn: func(ctx context.Context, root ocispec.Descriptor, target oras.ReadOnlyTarget) error {
			manifestReader, err := target.Fetch(ctx, root)
			if err != nil {
				return err
			}
			defer manifestReader.Close()
			var imgManifest ocispec.Manifest
			if err := json.NewDecoder(manifestReader).Decode(&imgManifest); err != nil {
				return err
			}
			switch imgManifest.MediaType {
			case ocispec.MediaTypeImageManifest, manifest.DockerV2Schema2MediaType:
			default:
				return fmt.Errorf("unrecognized manifest type: %s", imgManifest.MediaType)
			}
			return nil
		},
	}
}

func ImageHasLabels(expectedLabels map[string]string) ImageCheck {
	return ImageCheck{
		Name: "HasExpectedImageLabels",
		Fn: func(ctx context.Context, root ocispec.Descriptor, target oras.ReadOnlyTarget) error {
			manifestReader, err := target.Fetch(ctx, root)
			if err != nil {
				return err
			}
			defer manifestReader.Close()
			var imgManifest ocispec.Manifest
			if err := json.NewDecoder(manifestReader).Decode(&imgManifest); err != nil {
				return err
			}
			switch imgManifest.MediaType {
			case ocispec.MediaTypeImageManifest, manifest.DockerV2Schema2MediaType:
			default:
				return fmt.Errorf("unrecognized manifest type: %s", imgManifest.MediaType)
			}

			imageReader, err := target.Fetch(ctx, imgManifest.Config)
			if err != nil {
				return err
			}
			var img ocispec.Image
			if err := json.NewDecoder(imageReader).Decode(&img); err != nil {
				return err
			}

			var errs []error
			for expectedLabel, expectedValue := range expectedLabels {
				actualValue, ok := img.Config.Labels[expectedLabel]
				if !ok {
					errs = append(errs, fmt.Errorf("missing label: %s", expectedLabel))
					continue
				}
				if actualValue != expectedValue {
					errs = append(errs, fmt.Errorf("label %q: test expected %q, but image has %q", expectedLabel, expectedValue, actualValue))
				}
			}
			return errors.Join(errs...)
		},
	}
}
