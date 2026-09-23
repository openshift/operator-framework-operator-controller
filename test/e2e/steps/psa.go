package steps

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var (
	psaCheckBin      string
	psaCheckRequired bool
)

func init() {
	flagSet := pflag.CommandLine
	flagSet.StringVar(&psaCheckBin, "psa.check-bin", os.Getenv("PSA_CHECK_BIN"), "Path to the cluster-debug-tools PSA checker; empty disables automatic PSA checks")
	flagSet.BoolVar(&psaCheckRequired, "psa.check-required", psaCheckRequiredFromEnv(), "Fail scenarios when the PSA checker is unavailable")
}

func psaCheckRequiredFromEnv() bool {
	required, err := strconv.ParseBool(os.Getenv("PSA_CHECK_REQUIRED"))
	if err != nil {
		return false
	}
	return required
}

// runPSACheck evaluates the current scenario namespace before cleanup removes it.
// The checker is intentionally external because psa-check is distributed as a
// kubectl-dev_tool plugin rather than a package consumed by this repository.
func runPSACheck(ctx context.Context, sc *scenarioContext) error {
	bin := strings.TrimSpace(psaCheckBin)
	if bin == "" {
		if psaCheckRequired {
			return fmt.Errorf("PSA checker is required but --psa.check-bin or PSA_CHECK_BIN is unset")
		}
		return nil
	}

	if _, err := exec.LookPath(bin); err != nil {
		if psaCheckRequired {
			return fmt.Errorf("PSA checker %q is unavailable: %w", bin, err)
		}
		logger.Info("Skipping PSA check because checker is unavailable", "binary", bin, "error", err)
		return nil
	}

	cmd := exec.CommandContext(ctx, bin,
		"psa-check",
		"--namespace", sc.namespace,
		"--level", "restricted",
		"--output", "json",
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	artifactPath, artifactErr := psaArtifactPath(sc)
	if artifactErr != nil {
		return artifactErr
	}
	if artifactPath != "" {
		if writeErr := os.WriteFile(artifactPath, stdout.Bytes(), 0o600); writeErr != nil {
			return fmt.Errorf("write PSA result for scenario %q: %w", sc.scenarioName, writeErr)
		}
		if stderr.Len() > 0 {
			stderrPath := strings.TrimSuffix(artifactPath, filepath.Ext(artifactPath)) + ".stderr"
			if writeErr := os.WriteFile(stderrPath, stderr.Bytes(), 0o600); writeErr != nil {
				return fmt.Errorf("write PSA stderr for scenario %q: %w", sc.scenarioName, writeErr)
			}
		}
	}

	if err == nil {
		return nil
	}

	message := strings.TrimSpace(stdout.String())
	if stderrMessage := strings.TrimSpace(stderr.String()); stderrMessage != "" {
		if message != "" {
			message += "; "
		}
		message += stderrMessage
	}
	if message == "" {
		message = "no diagnostic output"
	}
	return fmt.Errorf("PSA check failed for scenario %q in namespace %q: %w: %s", sc.scenarioName, sc.namespace, err, message)
}

func psaArtifactPath(sc *scenarioContext) (string, error) {
	basePath := strings.TrimSpace(os.Getenv("ARTIFACT_PATH"))
	if basePath == "" {
		return "", nil
	}
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("resolve PSA artifact base path: %w", err)
	}

	path := filepath.Join(basePath, "psa", sanitizePSAArtifactPart(sc.featureName), sanitizePSAArtifactPart(sc.id))
	if !isWithinArtifactPath(basePath, path) {
		return "", fmt.Errorf("PSA artifact directory %q escapes artifact path %q", path, basePath)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create PSA artifact directory %q: %w", path, err)
	}
	return filepath.Join(path, "psa.json"), nil
}

// isWithinArtifactPath reports whether path is lexically contained by basePath.
func isWithinArtifactPath(basePath, path string) bool {
	relative, err := filepath.Rel(basePath, path)
	return err == nil && filepath.IsLocal(relative)
}

func sanitizePSAArtifactPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	part := strings.Trim(b.String(), "-")
	if part == "" || part == "." || part == ".." {
		return "unknown"
	}
	return part
}
