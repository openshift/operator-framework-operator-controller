package specs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWaitForRegistryInAuthfileReResolvesPodForReadinessErrors(t *testing.T) {
	var resolvedPods, checkedPods []string
	runner := func(_ context.Context, args ...string) ([]byte, []byte, int, error) {
		switch args[1] {
		case "get":
			pod := "catalogd-old"
			if len(resolvedPods) > 0 {
				pod = "catalogd-new"
			}
			resolvedPods = append(resolvedPods, pod)
			return []byte(pod), nil, 0, nil
		case "exec":
			pod := args[4]
			checkedPods = append(checkedPods, pod)
			if len(checkedPods) == 1 {
				return nil, []byte(`Error from server (NotFound): pods "catalogd-old" not found`), 1, errors.New("exit status 1")
			}
			return nil, nil, 0, nil
		default:
			t.Fatalf("unexpected command arguments: %v", args)
			return nil, nil, -1, errors.New("unexpected command")
		}
	}

	err := waitForRegistryInAuthfile(context.Background(), runner,
		"catalogd", "openshift-catalogd", "control-plane=catalogd-controller-manager",
		"/tmp/catalogd-global-pull-secret-*.json", time.Millisecond, time.Second)
	if err != nil {
		t.Fatalf("waitForRegistryInAuthfile() error = %v", err)
	}

	wantPods := []string{"catalogd-old", "catalogd-new"}
	if !reflect.DeepEqual(resolvedPods, wantPods) {
		t.Fatalf("resolved pods = %v, want %v", resolvedPods, wantPods)
	}
	if !reflect.DeepEqual(checkedPods, wantPods) {
		t.Fatalf("checked pods = %v, want %v", checkedPods, wantPods)
	}
}

func TestAuthfileProbe(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantStatus int
	}{
		{name: "registry present", content: `{"auths":{"fake.registry":{}}}`, wantStatus: 0},
		{name: "registry absent", content: `{"auths":{"another.registry":{}}}`, wantStatus: authfileNotReadyExitCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "authfile.json"), []byte(tt.content), 0o600); err != nil {
				t.Fatalf("write authfile: %v", err)
			}

			cmd := exec.Command("bash", "-c", authfileProbe, "--", filepath.Join(dir, "*.json"), authfileRegistry) //nolint:gosec // probe and arguments are controlled by the test
			output, err := cmd.CombinedOutput()
			status := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("run authfile probe: %v", err)
				}
				status = exitErr.ExitCode()
			}
			if status != tt.wantStatus {
				t.Fatalf("authfile probe exit status = %d, want %d", status, tt.wantStatus)
			}
			if len(output) != 0 {
				t.Fatalf("authfile probe output = %q, want empty output", output)
			}
		})
	}
}

func TestWaitForRegistryInAuthfileStopsOnUnclassifiedErrorWithoutOutput(t *testing.T) {
	const sensitiveOutput = `{"auths":{"fake.registry":{"auth":"must-not-appear"}}}`
	getCalls := 0
	runner := func(_ context.Context, args ...string) ([]byte, []byte, int, error) {
		switch args[1] {
		case "get":
			getCalls++
			return []byte("operator-controller-pod"), nil, 0, nil
		case "exec":
			return nil, []byte(sensitiveOutput), 7, errors.New("exit status 7")
		default:
			t.Fatalf("unexpected command arguments: %v", args)
			return nil, nil, -1, errors.New("unexpected command")
		}
	}

	err := waitForRegistryInAuthfile(context.Background(), runner,
		"operator-controller", "openshift-operator-controller", "control-plane=operator-controller-controller-manager",
		"/tmp/operator-controller-global-pull-secrets-*.json", time.Millisecond, time.Second)
	if err == nil {
		t.Fatal("waitForRegistryInAuthfile() error = nil, want terminal probe error")
	}
	if getCalls != 1 {
		t.Fatalf("pod resolution calls = %d, want 1 for an unclassified error", getCalls)
	}
	if !strings.Contains(err.Error(), "exit code 7") {
		t.Fatalf("error = %q, want exit code", err)
	}
	if strings.Contains(err.Error(), sensitiveOutput) || strings.Contains(err.Error(), "must-not-appear") {
		t.Fatalf("error exposed authentication data: %q", err)
	}
}

func TestWaitForRegistryInAuthfileReportsLastReadinessErrorAtTimeout(t *testing.T) {
	runner := func(_ context.Context, args ...string) ([]byte, []byte, int, error) {
		if args[1] == "get" {
			return []byte("catalogd-deadline"), nil, 0, nil
		}
		return nil, nil, authfileNotReadyExitCode, errors.New("exit status 42")
	}

	err := waitForRegistryInAuthfile(context.Background(), runner,
		"catalogd", "openshift-catalogd", "control-plane=catalogd-controller-manager",
		"/tmp/catalogd-global-pull-secret-*.json", time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatal("waitForRegistryInAuthfile() error = nil, want timeout")
	}
	if !strings.Contains(err.Error(), `catalogd pod "catalogd-deadline" authfile does not contain "fake.registry" yet`) {
		t.Fatalf("error = %q, want last classified readiness error", err)
	}
}

func TestIsAuthfilePodReadinessError(t *testing.T) {
	tests := []struct {
		name    string
		stderr  string
		expects bool
	}{
		{name: "pod replaced", stderr: `Error from server (NotFound): pods "catalogd-old" not found`, expects: true},
		{name: "container starting", stderr: `unable to upgrade connection: container not found`, expects: true},
		{name: "missing executable is terminal", stderr: `exec: "bash": executable file not found`, expects: false},
		{name: "authorization is terminal", stderr: `Error from server (Forbidden): pods "catalogd" is forbidden`, expects: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuthfilePodReadinessError([]byte(tt.stderr)); got != tt.expects {
				t.Fatalf("isAuthfilePodReadinessError(%q) = %t, want %t", tt.stderr, got, tt.expects)
			}
		})
	}
}
