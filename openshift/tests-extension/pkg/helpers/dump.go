package helpers

import (
	"context"
	"fmt"
	"strings"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/ginkgo/v2"
)

func sectionHeader(format string, a ...any) {
	fmt.Fprintf(GinkgoWriter, "\n=== %s ===\n", fmt.Sprintf(format, a...))
}

func subHeader(format string, a ...any) {
	fmt.Fprintf(GinkgoWriter, "\n--- %s ---\n", fmt.Sprintf(format, a...))
}

// GetAllPodLogs prints logs for all containers in all pods in the given namespace.
func GetAllPodLogs(ctx context.Context, namespace string) {
	sectionHeader("[pod-logs] namespace=%s", namespace)

	By("listing pods in namespace " + namespace)
	namesOut, err := RunK8sCommand(ctx, "get", "pods", "-n", namespace, "-o", "name")
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list pods: %v\n%s\n", err, string(namesOut))
		return
	}
	lines := strings.Fields(strings.TrimSpace(string(namesOut)))
	if len(lines) == 0 {
		fmt.Fprintln(GinkgoWriter, "(no pods found)")
		return
	}

	for _, res := range lines {
		subHeader("logs for %s", res)
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
			fmt.Fprintf(GinkgoWriter, "error fetching logs for %s: %v\n%s\n", res, err, string(logsOut))
			continue
		}
		_, _ = GinkgoWriter.Write(logsOut) // ignore write error by design
	}
	fmt.Fprintln(GinkgoWriter)
}

// DescribePods prints the `kubectl/oc describe pods` output for all pods in a given namespace.
func DescribePods(ctx context.Context, namespace string) {
	sectionHeader("[describe pods] namespace=%s", namespace)
	RunAndPrint(ctx, "describe", "pods", "-n", namespace)
}

// DescribeAllClusterCatalogs lists all ClusterCatalogs and runs `describe` on each.
func DescribeAllClusterCatalogs(ctx context.Context) {
	sectionHeader("[cluster catalogs]")

	out, err := RunK8sCommand(ctx, "get", "clustercatalogs", "-o", "name")
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list clustercatalogs: %v\n", err)
		return
	}

	catalogs := strings.Fields(strings.TrimSpace(string(out)))
	if len(catalogs) == 0 {
		fmt.Fprintln(GinkgoWriter, "(no clustercatalogs found)")
		RunAndPrint(ctx, "get", "clustercatalogs")
		return
	}

	for _, catalog := range catalogs {
		subHeader("describe %s", catalog)
		RunAndPrint(ctx, "describe", catalog)
	}
	fmt.Fprintln(GinkgoWriter)
}

// DescribeAllClusterExtensions describes every ClusterExtension in the given namespace.
func DescribeAllClusterExtensions(ctx context.Context, namespace string) {
	if namespace == "" {
		return
	}
	sectionHeader("[clusterextensions] namespace=%s", namespace)

	args := []string{"get", "clusterextensions", "-n", namespace, "-o", "name"}
	out, err := RunK8sCommand(ctx, args...)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list clusterextensions: %v\n", err)
		RunAndPrint(ctx, args...)
		return
	}

	names := strings.Fields(strings.TrimSpace(string(out)))
	if len(names) == 0 {
		fmt.Fprintln(GinkgoWriter, "(no clusterextensions found)")
		RunAndPrint(ctx, args...)
		return
	}

	for _, n := range names {
		subHeader("describe %s", n)
		RunAndPrint(ctx, "describe", "clusterextension", strings.TrimPrefix(n, "clusterextension/"), "-n", namespace)
	}
	fmt.Fprintln(GinkgoWriter)
}
