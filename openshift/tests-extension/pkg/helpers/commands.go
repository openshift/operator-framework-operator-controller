package helpers

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
)

// findK8sTool returns "oc" if available, otherwise "kubectl".
// If we are running locally we either prefer to use oc since some tests
// require it, or fallback to kubectl if oc is not available.
func findK8sTool() (string, error) {
	tools := []string{"oc", "kubectl"}
	for _, t := range tools {
		// First check if the tool is available in the PATH.
		if _, err := exec.LookPath(t); err != nil {
			continue
		}
		// Verify that the tool is working by checking its version.
		if err := exec.Command(t, "version", "--client").Run(); err == nil {
			return t, nil
		}
	}
	return "", fmt.Errorf("no Kubernetes CLI client found (tried %s)",
		strings.Join(tools, ", "))
}

// RunK8sCommand runs a Kubernetes CLI command and returns ONLY stdout.
// If the command fails, stderr is included in the returned error (not mixed with stdout).
func RunK8sCommand(ctx context.Context, args ...string) ([]byte, error) {
	tool, err := findK8sTool()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, tool, args...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr := strings.TrimSpace(string(ee.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("%s %s failed: %w\nstderr:\n%s",
					tool, strings.Join(args, " "), err, stderr)
			}
		}
		return nil, fmt.Errorf("%s %s failed: %w",
			tool, strings.Join(args, " "), err)
	}
	return out, nil
}

// RunAndPrint runs a `kubectl/oc` command via RunK8sCommand and writes both stdout and stderr
// to the GinkgoWriter. It also prints the exact command being run.
func RunAndPrint(ctx context.Context, args ...string) {
	fmt.Fprintf(GinkgoWriter, "\n[diag] running: oc %s\n", strings.Join(args, " "))
	out, err := RunK8sCommand(ctx, args...)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "[diag] command failed: %v\n", err)
	}
	if len(out) > 0 {
		fmt.Fprintf(GinkgoWriter, "%s\n", string(out))
	}
}
