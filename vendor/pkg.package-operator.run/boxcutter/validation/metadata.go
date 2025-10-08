package validation

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateObjectMetadata(obj *unstructured.Unstructured) []error {
	var errs []error

	// Type Meta
	if len(obj.GetAPIVersion()) == 0 {
		errs = append(errs,
			field.Required(
				field.NewPath("apiVersion"),
				"must not be empty",
			))
	}

	if len(obj.GetKind()) == 0 {
		errs = append(errs,
			field.Required(
				field.NewPath("kind"),
				"must not be empty",
			))
	}

	metadataPath := field.NewPath("metadata")
	if len(obj.GetUID()) > 0 {
		errs = append(errs,
			field.Forbidden(
				metadataPath.Child("uid"),
				"must be empty",
			))
	}

	// Metadata
	if obj.GetGeneration() > 0 {
		errs = append(errs,
			field.Forbidden(
				metadataPath.Child("generation"),
				"must be empty",
			))
	}

	if len(obj.GetGenerateName()) > 0 {
		errs = append(errs,
			field.Forbidden(
				metadataPath.Child("generateName"),
				"must be empty",
			))
	}

	if len(obj.GetFinalizers()) > 0 {
		errs = append(errs,
			field.Forbidden(
				metadataPath.Child("finalizers"),
				"must be empty",
			))
	}

	if len(obj.GetOwnerReferences()) > 0 {
		errs = append(errs,
			field.Forbidden(
				metadataPath.Child("ownerReferences"),
				"must be empty",
			))
	}

	if len(obj.GetResourceVersion()) > 0 {
		errs = append(errs,
			field.Forbidden(
				metadataPath.Child("resourceVersion"),
				"must be empty",
			))
	}

	return errs
}
