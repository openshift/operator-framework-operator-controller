package steps

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPSACheck(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		required   bool
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "successful check writes JSON artifact",
			script:     "printf '%s' '{\"items\":[]}'",
			wantOutput: `{"items":[]}`,
		},
		{
			name:       "violations fail the check and preserve output",
			script:     "printf '%s' '{\"items\":[{\"namespace\":\"ns-test\"}]}' ; exit 1",
			wantErr:    true,
			wantOutput: `{"items":[{"namespace":"ns-test"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "psa-check")
			if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+tt.script+"\n"), 0o700); err != nil {
				t.Fatal(err)
			}

			artifactPath := t.TempDir()
			t.Setenv("ARTIFACT_PATH", artifactPath)
			oldBin, oldRequired, oldKubeconfig := psaCheckBin, psaCheckRequired, kubeconfigPath
			psaCheckBin, psaCheckRequired, kubeconfigPath = bin, tt.required, "/tmp/test.kubeconfig"
			t.Cleanup(func() {
				psaCheckBin, psaCheckRequired, kubeconfigPath = oldBin, oldRequired, oldKubeconfig
			})

			sc := &scenarioContext{id: "scenario-1", featureName: "install", scenarioName: tt.name, namespace: "ns-test"}
			err := runPSACheck(context.Background(), sc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("runPSACheck() error = %v, want error: %t", err, tt.wantErr)
			}

			result, err := os.ReadFile(filepath.Join(artifactPath, "psa", "install", "scenario-1", "psa.json"))
			if err != nil {
				t.Fatal(err)
			}
			if string(result) != tt.wantOutput {
				t.Fatalf("PSA artifact = %q, want %q", result, tt.wantOutput)
			}
		})
	}
}

func TestRunPSACheckConfiguration(t *testing.T) {
	t.Run("optional checker can be disabled", func(t *testing.T) {
		oldBin, oldRequired := psaCheckBin, psaCheckRequired
		psaCheckBin, psaCheckRequired = "", false
		t.Cleanup(func() { psaCheckBin, psaCheckRequired = oldBin, oldRequired })

		err := runPSACheck(context.Background(), &scenarioContext{namespace: "ns-test"})
		if err != nil {
			t.Fatalf("runPSACheck() error = %v", err)
		}
	})

	t.Run("required checker must be configured", func(t *testing.T) {
		oldBin, oldRequired := psaCheckBin, psaCheckRequired
		psaCheckBin, psaCheckRequired = "", true
		t.Cleanup(func() { psaCheckBin, psaCheckRequired = oldBin, oldRequired })

		err := runPSACheck(context.Background(), &scenarioContext{namespace: "ns-test"})
		if err == nil || !strings.Contains(err.Error(), "checker is required") {
			t.Fatalf("runPSACheck() error = %v, want required-checker error", err)
		}
	})
}

func TestSanitizePSAArtifactPart(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "normal value", input: "feature/name with spaces", want: "feature-name-with-spaces"},
		{name: "dot", input: ".", want: "unknown"},
		{name: "dot dot", input: "..", want: "unknown"},
		{name: "separators trim to dot dot", input: "../", want: "unknown"},
		{name: "empty after trimming", input: "---", want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizePSAArtifactPart(tt.input); got != tt.want {
				t.Fatalf("sanitizePSAArtifactPart(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsWithinArtifactPath(t *testing.T) {
	basePath := filepath.Join(t.TempDir(), "artifacts")
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "descendant", path: filepath.Join(basePath, "psa", "feature", "scenario"), want: true},
		{name: "parent", path: filepath.Dir(basePath), want: false},
		{name: "sibling with common prefix", path: basePath + "-outside", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWithinArtifactPath(basePath, tt.path); got != tt.want {
				t.Fatalf("isWithinArtifactPath(%q, %q) = %t, want %t", basePath, tt.path, got, tt.want)
			}
		})
	}
}

func TestPSAArtifactPathRejectsTraversalComponents(t *testing.T) {
	artifactPath := t.TempDir()
	t.Setenv("ARTIFACT_PATH", artifactPath)

	path, err := psaArtifactPath(&scenarioContext{featureName: "..", id: ".."})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(artifactPath, "psa", "unknown", "unknown", "psa.json")
	if path != want {
		t.Fatalf("psaArtifactPath() = %q, want %q", path, want)
	}
	if !isWithinArtifactPath(artifactPath, path) {
		t.Fatalf("psaArtifactPath() = %q, want path inside %q", path, artifactPath)
	}
}
