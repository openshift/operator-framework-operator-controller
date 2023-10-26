package olmv1util

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// PolicyRule represents a Kubernetes RBAC policy rule
type PolicyRule struct {
	APIGroups       []string `yaml:"apiGroups,omitempty"`
	Resources       []string `yaml:"resources,omitempty"`
	ResourceNames   []string `yaml:"resourceNames,omitempty"`
	NonResourceURLs []string `yaml:"nonResourceURLs,omitempty"`
	Verbs           []string `yaml:"verbs"`
}

// Metadata holds metadata for Role/ClusterRole
type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

// ClusterRole manifest definition
type ClusterRole struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   Metadata     `yaml:"metadata"`
	Rules      []PolicyRule `yaml:"rules"`
}

// Role manifest definition
type Role struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   Metadata     `yaml:"metadata"`
	Rules      []PolicyRule `yaml:"rules"`
}

// GenerateRBACFromMissingRules parses missing RBAC rules and generates ClusterRole and Role YAML files
// Parameters:
//   - missingrules: multi-line string containing RBAC rules in a specific format with APIGroups, Resources, Verbs, etc.
//   - cename: cluster extension name used as prefix for generated role names
//   - roleDir: directory path where the generated YAML files will be written
//
// Returns:
//   - error: error if parsing fails or file writing fails, nil on success
func GenerateRBACFromMissingRules(missingrules, cename, roleDir string) error {
	if strings.TrimSpace(missingrules) == "" {
		return fmt.Errorf("missing rules cannot be empty")
	}
	if strings.TrimSpace(cename) == "" {
		return fmt.Errorf("cluster extension name cannot be empty")
	}
	if strings.TrimSpace(roleDir) == "" {
		return fmt.Errorf("role directory cannot be empty")
	}
	lines := []string{}
	for _, ln := range strings.Split(missingrules, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			lines = append(lines, t)
		}
	}
	if len(lines) == 0 {
		return fmt.Errorf("no missing rules provided")
	}

	clusterRules := []PolicyRule{}
	namespaced := map[string][]PolicyRule{}
	for _, line := range lines {
		ns, rule := parseRule(line)
		if rule == nil {
			continue
		}
		if ns == "" {
			clusterRules = append(clusterRules, *rule)
		} else {
			namespaced[ns] = append(namespaced[ns], *rule)
		}
	}

	if len(clusterRules) == 0 && len(namespaced) == 0 {
		return fmt.Errorf("no valid rules parsed from input")
	}

	cr := ClusterRole{
		APIVersion: "rbac.authorization.k8s.io/v1",
		Kind:       "ClusterRole",
		Metadata:   Metadata{Name: cename + "-clusterrole"},
		Rules:      clusterRules,
	}
	if len(clusterRules) > 0 {
		out := fmt.Sprintf("%s/%s-clusterrole.yaml", roleDir, cename)
		if err := writeYAML(out, cr); err != nil {
			return fmt.Errorf("write clusterrole: %w", err)
		}
	}

	for ns, rules := range namespaced {
		if len(rules) == 0 {
			continue
		}
		role := Role{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
			Metadata:   Metadata{Name: fmt.Sprintf("%s-role-%s", cename, ns), Namespace: ns},
			Rules:      rules,
		}
		out := fmt.Sprintf("%s/%s-role-%s.yaml", roleDir, cename, ns)
		if err := writeYAML(out, role); err != nil {
			return fmt.Errorf("write role for ns %s: %w", ns, err)
		}
	}
	return nil
}

// parseRule parses a single line of RBAC rule text and extracts namespace and PolicyRule information
// Parameters:
//   - line: single line string containing RBAC rule information with APIGroups, Resources, Verbs, etc.
//
// Returns:
//   - string: namespace if the rule is namespace-scoped, empty string for cluster-scoped rules
//   - *PolicyRule: parsed PolicyRule struct containing the RBAC permissions, nil if parsing fails
func parseRule(line string) (string, *PolicyRule) {
	if strings.TrimSpace(line) == "" {
		return "", nil
	}
	ns := ""
	if i := strings.Index(line, `Namespace:"`); i >= 0 {
		rest := line[i+len(`Namespace:"`):]
		if j := strings.Index(rest, `"`); j >= 0 {
			ns = rest[:j]
		}
	}
	if idx := strings.Index(line, "NonResourceURLs:"); idx >= 0 {
		urls := extractList(line[idx:], "[")
		verbs := extractList(line, "Verbs:[")
		return ns, &PolicyRule{NonResourceURLs: urls, Verbs: verbs}
	}
	ag := extractList(line, "APIGroups:[")
	if len(ag) == 0 {
		ag = []string{""}
	}
	res := extractList(line, "Resources:[")
	rn := extractList(line, "ResourceNames:[")
	vs := extractList(line, "Verbs:[")
	if len(vs) == 0 {
		return "", nil // Invalid rule without verbs
	}
	return ns, &PolicyRule{APIGroups: ag, Resources: res, ResourceNames: rn, Verbs: vs}
}

// extractList extracts a comma-separated list of values between start and end delimiters from a string
// Parameters:
//   - s: source string to extract from
//   - start: starting delimiter to search for
//   - end: ending delimiter to search for
//
// Returns:
//   - []string: slice of trimmed, non-empty values found between delimiters, nil if delimiters not found
func extractList(s, start string) []string {
	if s == "" || start == "" {
		return nil
	}
	if i := strings.Index(s, start); i >= 0 {
		sub := s[i+len(start):]
		j := strings.Index(sub, "]")
		if j < 0 {
			j = len(sub)
		}
		parts := strings.Split(sub[:j], ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	return nil
}

// writeYAML writes an object to a YAML file with proper indentation
// Parameters:
//   - filename: path where the YAML file will be created
//   - obj: any object that can be marshaled to YAML
//
// Returns:
//   - error: error if file creation or YAML encoding fails, nil on success
func writeYAML(filename string, obj interface{}) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if obj == nil {
		return fmt.Errorf("object to write cannot be nil")
	}
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			// Log the error but don't override the primary error
			fmt.Printf("Warning: failed to close file %s: %v\n", filename, closeErr)
		}
	}()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(obj); err != nil {
		return fmt.Errorf("failed to encode YAML to file %s: %w", filename, err)
	}
	return nil
}
