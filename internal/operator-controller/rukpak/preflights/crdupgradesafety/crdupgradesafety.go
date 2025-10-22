package crdupgradesafety

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/release"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/operator-framework/operator-controller/internal/operator-controller/rukpak/util"
)

type Option func(p *Preflight)

func WithValidator(v *Validator) Option {
	return func(p *Preflight) {
		p.validator = v
	}
}

type Preflight struct {
	crdClient apiextensionsv1client.CustomResourceDefinitionInterface
	validator *Validator
}

func NewPreflight(crdCli apiextensionsv1client.CustomResourceDefinitionInterface, opts ...Option) *Preflight {
	changeValidations := []ChangeValidation{
		Description,
		Enum,
		Required,
		Maximum,
		MaxItems,
		MaxLength,
		MaxProperties,
		Minimum,
		MinItems,
		MinLength,
		MinProperties,
		Default,
		Type,
	}
	p := &Preflight{
		crdClient: crdCli,
		validator: &Validator{
			Validations: []Validation{
				NewValidationFunc("NoScopeChange", NoScopeChange),
				NewValidationFunc("NoStoredVersionRemoved", NoStoredVersionRemoved),
				NewValidationFunc("NoExistingFieldRemoved", NoExistingFieldRemoved),
				&ServedVersionValidator{Validations: changeValidations},
				&ChangeValidator{Validations: changeValidations},
			},
		},
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

func (p *Preflight) Install(ctx context.Context, rel *release.Release) error {
	return p.runPreflight(ctx, rel)
}

func (p *Preflight) Upgrade(ctx context.Context, rel *release.Release) error {
	return p.runPreflight(ctx, rel)
}

func (p *Preflight) runPreflight(ctx context.Context, rel *release.Release) error {
	if rel == nil {
		return nil
	}

	relObjects, err := util.ManifestObjects(strings.NewReader(rel.Manifest), fmt.Sprintf("%s-release-manifest", rel.Name))
	if err != nil {
		return fmt.Errorf("parsing release %q objects: %w", rel.Name, err)
	}

	validateErrors := make([]error, 0, len(relObjects))
	for _, obj := range relObjects {
		if obj.GetObjectKind().GroupVersionKind() != apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition") {
			continue
		}

		newCrd := &apiextensionsv1.CustomResourceDefinition{}
		uMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return fmt.Errorf("converting object %q to unstructured: %w", obj.GetName(), err)
		}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(uMap, newCrd); err != nil {
			return fmt.Errorf("converting unstructured to CRD object: %w", err)
		}

		oldCrd, err := p.crdClient.Get(ctx, newCrd.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("getting existing resource for CRD %q: %w", newCrd.Name, err)
		}

		if err = p.validator.Validate(*oldCrd, *newCrd); err != nil {
			validateErrors = append(validateErrors, fmt.Errorf("validating upgrade for CRD %q failed: %w", newCrd.Name, err))
		}
	}

	return errors.Join(validateErrors...)
}

const unhandledSummaryPrefix = "unhandled changes found"

func conciseUnhandledMessage(raw string) string {
	if !strings.Contains(raw, unhandledSummaryPrefix) {
		return raw
	}

	details := extractUnhandledDetails(raw)
	if len(details) == 0 {
		return unhandledSummaryPrefix
	}

	return fmt.Sprintf("%s (%s)", unhandledSummaryPrefix, strings.Join(details, "; "))
}

func extractUnhandledDetails(raw string) []string {
	type diffEntry struct {
		before    string
		after     string
		beforeRaw string
		afterRaw  string
	}

	entries := map[string]*diffEntry{}
	order := []string{}

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 2 {
			continue
		}

		sign := trimmed[0]
		if sign != '-' && sign != '+' {
			continue
		}

		field, value, rawValue := parseUnhandledDiffValue(trimmed[1:])
		if field == "" {
			continue
		}

		entry, ok := entries[field]
		if !ok {
			entry = &diffEntry{}
			entries[field] = entry
			order = append(order, field)
		}

		if sign == '-' {
			entry.before = value
			entry.beforeRaw = rawValue
		} else {
			entry.after = value
			entry.afterRaw = rawValue
		}
	}

	details := []string{}
	for _, field := range order {
		entry := entries[field]
		if entry.before == "" && entry.after == "" {
			continue
		}
		if entry.before == entry.after && entry.beforeRaw == entry.afterRaw {
			continue
		}

		before := entry.before
		if before == "" {
			before = "<empty>"
		}
		after := entry.after
		if after == "" {
			after = "<empty>"
		}
		if entry.before == entry.after && entry.beforeRaw != entry.afterRaw {
			after = after + " (changed)"
		}

		details = append(details, fmt.Sprintf("%s %s -> %s", field, before, after))
	}

	return details
}

func parseUnhandledDiffValue(fragment string) (string, string, string) {
	cleaned := strings.TrimSpace(fragment)
	cleaned = strings.TrimPrefix(cleaned, "\t")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimSuffix(cleaned, ",")

	parts := strings.SplitN(cleaned, ":", 2)
	if len(parts) != 2 {
		return "", "", ""
	}

	field := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	stripped := strings.Trim(value, `"`)
	return field, stripped, value
}
