package helpers

import (
	"context"
	"fmt"
	"strings"
	"time"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
)

// GetAllPodLogs prints logs for all containers in all pods in the given namespace.
func GetAllPodLogs(ctx context.Context, namespace string) {
	fmt.Fprintf(GinkgoWriter, "\n[pod-logs] namespace=%s\n", namespace)

	By("Getting all pods in the namespace")
	namesOut, err := RunK8sCommand(ctx, "get", "pods", "-n", namespace, "-o", "name")
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list pods: %v\n%s\n", err, string(namesOut))
		return
	}
	lines := strings.Split(strings.TrimSpace(string(namesOut)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		fmt.Fprintln(GinkgoWriter, "no pods found")
		return
	}

	By(fmt.Sprintf("[pod-logs] namespace=%s\n", namespace))
	for _, res := range lines {
		res = strings.TrimSpace(res)
		if res == "" {
			continue
		}
		fmt.Fprintf(GinkgoWriter, "\n--- logs: %s @ %s ---\n", res, time.Now().Format(time.RFC3339))
		logsOut, err := RunK8sCommand(
			ctx,
			"logs",
			"-n", namespace,
			"--all-containers",
			"--prefix",
			"--timestamps",
			res,
		)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "error fetching logs: %v\n%s\n", err, string(logsOut))
			continue
		}
		_, _ = GinkgoWriter.Write(logsOut) // ignore write error by design
	}
}

// DescribePods prints the `kubectl describe pods` output for all pods in a given namespace.
func DescribePods(ctx context.Context, namespace string) {
	fmt.Fprintf(GinkgoWriter, "\n[diag] === describe pods in namespace %q ===\n", namespace)
	RunAndPrint(ctx, "describe", "pods", "-n", namespace)
}
