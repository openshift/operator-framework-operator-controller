package crdupgradesafety

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	versionhelper "k8s.io/apimachinery/pkg/version"
)

type ServedVersionValidator struct {
	Validations []ChangeValidation
}

func (c *ServedVersionValidator) Validate(old, new apiextensionsv1.CustomResourceDefinition) error {
	// If conversion webhook is specified, pass check
	if new.Spec.Conversion != nil && new.Spec.Conversion.Strategy == apiextensionsv1.WebhookConverter {
		return nil
	}

	newErrs := c.collectServedVersionErrors(new)
	if len(newErrs) == 0 {
		return nil
	}

	existingErrCounts := map[string]int{}
	if len(old.Spec.Versions) > 0 {
		for _, err := range c.collectServedVersionErrors(old) {
			existingErrCounts[err.Error()]++
		}
	}

	filteredErrs := make([]error, 0, len(newErrs))
	for _, err := range newErrs {
		msg := err.Error()
		if count, ok := existingErrCounts[msg]; ok && count > 0 {
			existingErrCounts[msg]--
			continue
		}
		filteredErrs = append(filteredErrs, err)
	}

	if len(filteredErrs) > 0 {
		return errors.Join(filteredErrs...)
	}
	return nil
}

func (c *ServedVersionValidator) collectServedVersionErrors(crd apiextensionsv1.CustomResourceDefinition) []error {
	errs := []error{}
	servedVersions := []apiextensionsv1.CustomResourceDefinitionVersion{}
	for _, version := range crd.Spec.Versions {
		if version.Served {
			servedVersions = append(servedVersions, version)
		}
	}

	if len(servedVersions) < 2 {
		return nil
	}

	slices.SortFunc(servedVersions, func(a, b apiextensionsv1.CustomResourceDefinitionVersion) int {
		return versionhelper.CompareKubeAwareVersionStrings(a.Name, b.Name)
	})

	for i := 0; i < len(servedVersions)-1; i++ {
		oldVersion := servedVersions[i]
		for j := i + 1; j < len(servedVersions); j++ {
			newVersion := servedVersions[j]
			flatOld := FlattenSchema(oldVersion.Schema.OpenAPIV3Schema)
			flatNew := FlattenSchema(newVersion.Schema.OpenAPIV3Schema)
			diffs, err := CalculateFlatSchemaDiff(flatOld, flatNew)
			if err != nil {
				errs = append(errs, fmt.Errorf("calculating schema diff between CRD versions %q and %q", oldVersion.Name, newVersion.Name))
				continue
			}

			for _, field := range slices.Sorted(maps.Keys(diffs)) {
				diff := diffs[field]

				handled := false
				for _, validation := range c.Validations {
					ok, err := validation(diff)
					if err != nil {
						errs = append(errs, fmt.Errorf("%s -> %s: %s: %w", oldVersion.Name, newVersion.Name, field, err))
					}
					if ok {
						handled = true
						break
					}
				}

				if !handled {
					errs = append(errs, fmt.Errorf("%s -> %s: %s: has unknown change, refusing to determine that change is safe", oldVersion.Name, newVersion.Name, field))
				}
			}
		}
	}
	return errs
}

func (c *ServedVersionValidator) Name() string {
	return "ServedVersionValidator"
}
