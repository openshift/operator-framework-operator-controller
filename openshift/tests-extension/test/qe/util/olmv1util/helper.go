package olmv1util

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

// Get retrieves OLMv1 resource field values allowing empty results
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - parameters: variable arguments for oc command (add "-n" if namespace is needed)
//
// Returns:
//   - string: field value (can be empty)
//   - error: error if command execution fails, nil on success
func Get(oc *exutil.CLI, parameters ...string) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	return exutil.GetFieldWithJsonpath(oc, 3*time.Second, 150*time.Second, exutil.Immediately,
		exutil.AllowEmpty, exutil.AsAdmin, exutil.WithoutNamespace, parameters...)
}

// GetNoEmpty retrieves OLMv1 resource field values but does not allow empty results
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - parameters: variable arguments for oc command (add "-n" if namespace is needed)
//
// Returns:
//   - string: non-empty field value
//   - error: error if command execution fails or result is empty, nil on success
func GetNoEmpty(oc *exutil.CLI, parameters ...string) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	return exutil.GetFieldWithJsonpath(oc, 3*time.Second, 150*time.Second, exutil.Immediately,
		exutil.NotAllowEmpty, exutil.AsAdmin, exutil.WithoutNamespace, parameters...)
}

// Cleanup removes resources using admin privileges with predefined timeouts
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - parameters: variable arguments specifying the resource type and name to clean up
func Cleanup(oc *exutil.CLI, parameters ...string) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	exutil.CleanupResource(oc, 4*time.Second, 160*time.Second,
		exutil.AsAdmin, exutil.WithoutNamespace, parameters...)
}

// ApplyClusterResourceFromTemplate processes and applies a cluster-scoped resource template
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - parameters: template processing parameters including template file and variable assignments
//
// Returns:
//   - string: path to the generated configuration file
//   - error: error if template processing or resource application fails, nil on success
func ApplyClusterResourceFromTemplate(oc *exutil.CLI, parameters ...string) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	return resourceFromTemplate(oc, false, true, "", parameters...)
}

// ApplyNamespaceResourceFromTemplate processes and applies a namespace-scoped resource template
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - namespace: target namespace for the resource
//   - parameters: template processing parameters including template file and variable assignments
//
// Returns:
//   - string: path to the generated configuration file
//   - error: error if template processing or resource application fails, nil on success
func ApplyNamespaceResourceFromTemplate(oc *exutil.CLI, namespace string, parameters ...string) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	if strings.TrimSpace(namespace) == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}
	return resourceFromTemplate(oc, false, true, namespace, parameters...)
}

// ApplyNamepsaceResourceFromTemplate is a deprecated alias for ApplyNamespaceResourceFromTemplate
// Deprecated: Use ApplyNamespaceResourceFromTemplate instead
func ApplyNamepsaceResourceFromTemplate(oc *exutil.CLI, namespace string, parameters ...string) (string, error) {
	return ApplyNamespaceResourceFromTemplate(oc, namespace, parameters...)
}

// Appearance checks if a resource appears or disappears within a specified timeframe
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - appear: true to check for resource appearance, false to check for disappearance
//   - parameters: resource specification arguments for the check
//
// Returns:
//   - bool: true if the appearance/disappearance expectation is met, false otherwise
func Appearance(oc *exutil.CLI, appear bool, parameters ...string) bool {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	return exutil.CheckAppearance(oc, 4*time.Second, 200*time.Second, exutil.NotImmediately,
		exutil.AsAdmin, exutil.WithoutNamespace, appear, parameters...)
}

// IsFeaturegateEnabled checks if a specific feature gate is enabled in the cluster
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - featuregate: name of the feature gate to check
//
// Returns:
//   - bool: true if the feature gate is enabled, false otherwise
func IsFeaturegateEnabled(oc *exutil.CLI, featuregate string) bool {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(strings.TrimSpace(featuregate)).NotTo(o.BeEmpty(), "feature gate name cannot be empty")
	featureGate, err := oc.AdminConfigClient().ConfigV1().FeatureGates().Get(context.Background(), "cluster", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}
		o.Expect(err).NotTo(o.HaveOccurred(), "could not retrieve feature-gate: %v", err)
	}

	isEnabled := false
	for _, featureGate := range featureGate.Status.FeatureGates {
		for _, enabled := range featureGate.Enabled {
			if string(enabled.Name) == featuregate {
				isEnabled = true
				break
			}
		}
		if isEnabled {
			break
		}
	}
	return isEnabled
}

// FilterPermissions extracts permission-related content between specified markers from a message
// Parameters:
//   - msg: source message string to filter
//   - startMarker: marker indicating the start of relevant content
//   - end1Marker: first possible end marker for the relevant section
//   - end2Marker: second possible end marker for the relevant section
//
// Returns:
//   - string: filtered content between start marker and either end marker
func FilterPermissions(msg, startMarker, end1Marker, end2Marker string) string {
	if msg == "" {
		return ""
	}
	if startMarker == "" {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(msg))
	var sb strings.Builder
	inSection := false

	for scanner.Scan() {
		line := scanner.Text()
		// Start when we see the start marker
		if !inSection && strings.Contains(line, startMarker) {
			inSection = true
			continue
		}
		if inSection {
			// Check if this line contains the end marker
			if idx := strings.Index(line, end1Marker); idx != -1 {
				// Include content before the marker and then stop
				part := strings.TrimSpace(line[:idx])
				sb.WriteString(part)
				sb.WriteString("\n")
				break
			}
			if idx := strings.Index(line, end2Marker); idx != -1 {
				// Include content before the marker and then stop
				part := strings.TrimSpace(line[:idx])
				sb.WriteString(part)
				sb.WriteString("\n")
				break
			}
			// Otherwise, include the full line
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// resourceFromTemplate processes a template and creates/applies the resulting resources
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - create: true to use 'create' command, false to use 'apply' command
//   - returnError: true to return errors instead of failing the test
//   - namespace: target namespace (empty string for cluster-scoped resources)
//   - parameters: template processing parameters
//
// Returns:
//   - string: path to the generated configuration file
//   - error: error if processing fails and returnError is true, nil on success
func resourceFromTemplate(oc *exutil.CLI, create bool, returnError bool, namespace string, parameters ...string) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	if len(parameters) == 0 {
		return "", fmt.Errorf("template parameters cannot be empty")
	}
	var configFile string
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 15*time.Second,
		false, func(ctx context.Context) (bool, error) {
			fileName := exutil.GetRandomString() + "config.json"
			stdout, _, err := oc.AsAdmin().Run("process").Args(parameters...).OutputsToFiles(fileName)
			if err != nil {
				e2e.Logf("the err:%v, and try next round", err)
				return false, nil
			}

			configFile = stdout
			return true, nil
		})
	if returnError && errWait != nil {
		e2e.Logf("fail to process %v", parameters)
		return "", errWait
	}
	exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("fail to process %v", parameters))

	e2e.Logf("the file of resource is %s", configFile)

	resourceErr := executeResourceOperation(oc, create, configFile, namespace)
	if returnError && resourceErr != nil {
		e2e.Logf("fail to create/apply resource %v", resourceErr)
		return "", resourceErr
	}
	exutil.AssertWaitPollNoErr(resourceErr, fmt.Sprintf("fail to create/apply resource %v", resourceErr))
	return configFile, nil
}

// executeResourceOperation executes create or apply operation with optional namespace
func executeResourceOperation(oc *exutil.CLI, create bool, configFile, namespace string) error {
	action := "apply"
	if create {
		action = "create"
	}

	args := []string{"-f", configFile}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	return oc.AsAdmin().WithoutNamespace().Run(action).Args(args...).Execute()
}

// IsPodReady checks if all pods matching a label selector in a namespace are ready and running
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - ns: namespace to check pods in
//   - label: label selector to filter pods
//
// Returns:
//   - bool: true if all matching pods are ready and running, false otherwise
func IsPodReady(oc *exutil.CLI, ns, label string) bool {
	if oc == nil {
		return false
	}
	if strings.TrimSpace(ns) == "" {
		return false
	}
	if strings.TrimSpace(label) == "" {
		return false
	}
	pods, err := oc.AdminKubeClient().CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		e2e.Logf("Failed to list pods in namespace %s with label %s: %v", ns, label, err)
		return false
	}
	if len(pods.Items) == 0 {
		return true
	}
	isReady := true
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			isReady = false
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
				isReady = false
			}
		}
	}
	return isReady
}

// makeArtifactDir creates a subdirectory within the artifact directory for storing test artifacts
// Parameters:
//   - subdir: name of the subdirectory to create
//
// Returns:
//   - string: full path to the created subdirectory, empty string if creation fails
func makeArtifactDir(subdir string) string {
	if strings.TrimSpace(subdir) == "" {
		e2e.Logf("subdirectory name cannot be empty")
		return ""
	}
	dirPath := os.Getenv("ARTIFACT_DIR")
	if dirPath == "" {
		dirPath = "/tmp"
	}
	e2e.Logf("the log dir path: %s", dirPath)
	logSubDir := dirPath + "/" + subdir
	err := os.MkdirAll(logSubDir, 0750)
	if err != nil {
		e2e.Logf("failed to create %s", logSubDir)
		return ""
	}
	return logSubDir
}

// WriteErrToArtifactDir extracts error logs from a pod and writes them to the artifact directory
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - ns: namespace containing the pod
//   - podName: name of the pod to extract logs from
//   - pattern: regex pattern to match error lines
//   - expattern: regex pattern to exclude from matches
//   - caseid: test case identifier for the output filename
//   - minutes: number of minutes of recent logs to retrieve
//
// Returns:
//   - bool: true if error logs were successfully written to artifact directory, false otherwise
func WriteErrToArtifactDir(oc *exutil.CLI, ns, podName, pattern, expattern, caseid string, minutes int) bool {
	if oc == nil {
		e2e.Logf("CLI client cannot be nil")
		return false
	}
	if strings.TrimSpace(ns) == "" || strings.TrimSpace(podName) == "" {
		e2e.Logf("namespace and pod name cannot be empty")
		return false
	}
	if minutes <= 0 {
		e2e.Logf("minutes must be positive")
		return false
	}
	logFile, errLog := oc.AsAdmin().WithoutNamespace().Run("logs").Args("-n", ns, podName, "--since", fmt.Sprintf("%dm", minutes)).OutputToFile(podName + ".log")
	if errLog != nil {
		e2e.Logf("can not get log of pod %s in %s", podName, ns)
		return false
	}
	cmd := fmt.Sprintf(`grep -iE '%s' %s | grep -vE '%s' || true`, pattern, logFile, expattern)
	errLogs, errExec := exec.Command("bash", "-c", cmd).Output()
	if errExec != nil {
		e2e.Logf("can not cat error log of pod %s in %s", podName, ns)
		return false
	}

	if len(errLogs) == 0 {
		e2e.Logf("no error log of pod %s in %s", podName, ns)
		return false
	}

	subdir := makeArtifactDir("podLog")
	if len(subdir) == 0 {
		e2e.Logf("can not make sub dir for log of pod %s in %s", podName, ns)
		return false
	}
	errLogFile := subdir + "/" + caseid + "-" + podName + "-errors.log"
	if writeErr := os.WriteFile(errLogFile, errLogs, 0600); writeErr != nil {
		e2e.Logf("failed to write error logs to %s: %v\n", errLogFile, writeErr)
		return false
	}

	return true
}

type CheckDescription struct {
	Method          string
	Executor        bool
	InlineNamespace bool
	ExpectAction    bool
	ExpectContent   string
	Expect          bool
	Resource        []string
}

// NewCheck creates a CheckDescription object for resource validation
// Parameters:
//   - method: "expect" for content checking or "present" for existence checking
//   - executor: true for admin privileges, false for user privileges
//   - inlineNamespace: true for WithoutNamespace(), false for WithNamespace()
//   - expectAction: true for exact comparison, false for substring containment (only for "expect" method)
//   - expectContent: expected content string for "expect" method
//   - expect: true for positive expectation, false for negative expectation
//   - resource: slice of resource specification arguments
//
// Returns:
//   - CheckDescription: configured check object for validation
func NewCheck(method string, executor bool, inlineNamespace bool, expectAction bool,
	expectContent string, expect bool, resource []string) CheckDescription {
	o.Expect(method == "expect" || method == "present").To(o.BeTrue(), "invalid method: %s, must be 'expect' or 'present'", method)
	o.Expect(len(resource)).To(o.BeNumerically(">", 0), "resource specification cannot be empty")
	return CheckDescription{
		Method:          method,
		Executor:        executor,
		InlineNamespace: inlineNamespace,
		ExpectAction:    expectAction,
		ExpectContent:   expectContent,
		Expect:          expect,
		Resource:        resource,
	}
}

// Check executes the validation defined in the CheckDescription object
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (ck CheckDescription) Check(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	switch ck.Method {
	case "present":
		ok := checkPresent(oc, 3, 70, ck.Executor, ck.InlineNamespace, ck.ExpectAction, ck.Resource...)
		o.Expect(ok).To(o.BeTrue())
	case "expect":
		err := expectedResource(oc, ck.Executor, ck.InlineNamespace, ck.ExpectAction, ck.ExpectContent, ck.Expect, ck.Resource...)
		exutil.AssertWaitPollNoErr(err, fmt.Sprintf("expected content %s not found by %v", ck.ExpectContent, ck.Resource))
	default:
		err := fmt.Errorf("unknown method")
		o.Expect(err).NotTo(o.HaveOccurred())
	}
}

// checkPresent checks if a resource exists or does not exist within a specified timeframe
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - intervalSec: polling interval in seconds
//   - durationSec: maximum duration to wait in seconds
//   - asAdmin: true to use admin privileges, false for user privileges
//   - withoutNamespace: true to use WithoutNamespace(), false to use WithNamespace()
//   - present: true to expect resource presence, false to expect absence
//   - parameters: resource specification arguments
//
// Returns:
//   - bool: true if presence expectation is met within timeout, false otherwise
func checkPresent(oc *exutil.CLI, intervalSec int, durationSec int, asAdmin bool, withoutNamespace bool, present bool, parameters ...string) bool {
	if oc == nil {
		return false
	}
	if intervalSec <= 0 || durationSec <= 0 {
		e2e.Logf("intervalSec and durationSec must be positive")
		return false
	}
	if len(parameters) == 0 {
		e2e.Logf("parameters cannot be empty")
		return false
	}
	parameters = append(parameters, "--ignore-not-found")
	err := wait.PollUntilContextTimeout(context.TODO(), time.Duration(intervalSec)*time.Second, time.Duration(durationSec)*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := exutil.OcAction(oc, "get", asAdmin, withoutNamespace, parameters...)
		if err != nil {
			e2e.Logf("the get error is %v, and try next", err)
			return false, nil
		}
		if !present && output == "" {
			return true, nil
		}
		if present && output != "" {
			return true, nil
		}
		return false, nil
	})
	return err == nil
}

// expectedResource checks if a resource's attribute matches expected content using contain or exact comparison
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - asAdmin: true to use admin privileges, false for user privileges
//   - withoutNamespace: true to use WithoutNamespace(), false to use WithNamespace()
//   - isCompare: true for exact comparison, false for substring containment
//   - content: expected content string (supports "+2+" separator for multiple values and "-TIME-WAIT-" for custom timeout)
//   - expect: true for positive expectation, false for negative expectation
//   - parameters: resource specification arguments
//
// Returns:
//   - error: error if expectation is not met within timeout, nil if expectation is satisfied
func expectedResource(oc *exutil.CLI, asAdmin bool, withoutNamespace bool, isCompare bool, content string, expect bool, parameters ...string) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if len(parameters) == 0 {
		return fmt.Errorf("parameters cannot be empty")
	}
	expectMap := map[bool]string{
		true:  "do",
		false: "do not",
	}

	cc := func(a, b string, ic bool) bool {
		bs := strings.Split(b, "+2+")
		for _, s := range bs {
			if (ic && a == s) || (!ic && strings.Contains(a, s)) {
				return true
			}
		}
		return false
	}
	e2e.Logf("Running: oc get asAdmin(%t) withoutNamespace(%t) %s", asAdmin, withoutNamespace, strings.Join(parameters, " "))

	// The default timeout
	timeString := "300s"
	// extract the custom timeout
	if strings.Contains(content, "-TIME-WAIT-") {
		parts := strings.Split(content, "-TIME-WAIT-")
		if len(parts) >= 2 {
			timeString = parts[1]
			content = parts[0]
			e2e.Logf("! reset the timeout to %s", timeString)
		}
	}
	timeout, err := time.ParseDuration(timeString)
	if err != nil {
		e2e.Failf("! Fail to parse the timeout value:%s, err:%v", content, err)
	}

	return wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		output, err := exutil.OcAction(oc, "get", asAdmin, withoutNamespace, parameters...)
		if err != nil {
			e2e.Logf("the get error is %v, and try next", err)
			return false, nil
		}
		e2e.Logf("---> we %v expect value: %s, in returned value: %s", expectMap[expect], content, output)
		if isCompare && expect && cc(output, content, isCompare) {
			e2e.Logf("the output %s matches one of the content %s, expected", output, content)
			return true, nil
		}
		if isCompare && !expect && !cc(output, content, isCompare) {
			e2e.Logf("the output %s does not matche the content %s, expected", output, content)
			return true, nil
		}
		if !isCompare && expect && cc(output, content, isCompare) {
			e2e.Logf("the output %s contains one of the content %s, expected", output, content)
			return true, nil
		}
		if !isCompare && !expect && !cc(output, content, isCompare) {
			e2e.Logf("the output %s does not contain the content %s, expected", output, content)
			return true, nil
		}
		e2e.Logf("---> Not as expected! Return false")
		return false, nil
	})
}

// HasExternalNetworkAccess tests network connectivity from a cluster master node
// by attempting to access an external container registry (quay.io).
// This method uses DebugNodeWithChroot to avoid creating pods and pulling images,
// which would fail in disconnected environments.
//
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - bool: true if external network access is available, false otherwise
func HasExternalNetworkAccess(oc *exutil.CLI) bool {
	if oc == nil {
		e2e.Logf("CLI client is nil, assuming connected environment")
		return true
	}

	e2e.Logf("Testing external network connectivity from master node using DebugNodeWithChroot")

	masterNode, masterErr := exutil.GetFirstMasterNode(oc)
	if masterErr != nil {
		e2e.Logf("Failed to get master node: %v", masterErr)
		g.Skip(fmt.Sprintf("Cannot determine network connectivity: %v", masterErr))
	}

	// Test connectivity to quay.io (container registry)
	// Use timeout to avoid hanging, and redirect output to check connection status
	// Note: In disconnected environments, curl will fail and bash will return non-zero exit code,
	// causing DebugNodeWithChroot to return an error. We ignore this error and rely on output checking.
	cmd := `timeout 10 curl -k https://quay.io > /dev/null 2>&1; [ $? -eq 0 ] && echo "connected"`
	output, _ := exutil.DebugNodeWithChroot(oc, masterNode, "bash", "-c", cmd)

	// Check if the output contains "connected"
	// - Connected environment: curl succeeds -> echo "connected" -> output contains "connected"
	// - Disconnected environment: curl fails -> no echo -> output empty or only debug messages
	if strings.Contains(output, "connected") {
		e2e.Logf("External network connectivity test succeeded (output: %s), cluster can access quay.io", strings.TrimSpace(output))
		return true
	}

	e2e.Logf("External network connectivity test failed (output: %s), cluster cannot access quay.io", strings.TrimSpace(output))
	return false
}

// IsProxyCluster checks whether the cluster is configured with HTTP/HTTPS proxy.
// Proxy clusters are treated as connected environments since they can access external networks through the proxy.
//
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - true if cluster has HTTP or HTTPS proxy configured in status
//   - false if no proxy is configured
//
// Behavior:
//   - Skips the test if oc is nil or if error occurs while checking proxy configuration
func IsProxyCluster(oc *exutil.CLI) bool {
	if oc == nil {
		e2e.Logf("CLI client is nil, cannot check proxy configuration")
		g.Skip("CLI client is nil, cannot check proxy configuration")
	}

	// Get proxy status in one call to check both httpProxy and httpsProxy
	// Format: {"httpProxy":"<value>","httpsProxy":"<value>"}
	proxyStatus, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("proxy", "cluster", "-o=jsonpath={.status}").Output()
	if err != nil {
		e2e.Logf("Failed to get proxy status: %v", err)
		g.Skip(fmt.Sprintf("cannot get proxy status: %v", err))
	}

	// If either httpProxy or httpsProxy is configured, the status will contain http
	// Connected cluster status is empty "{}"
	// Proxy cluster status contains "httpProxy" or "httpsProxy" fields with non-empty values
	if strings.Contains(proxyStatus, "httpProxy") || strings.Contains(proxyStatus, "httpsProxy") {
		e2e.Logf("Proxy cluster detected")
		return true
	}

	e2e.Logf("No proxy configuration detected in cluster (status=%s)", proxyStatus)
	return false
}

// ValidateAccessEnvironment checks if the cluster is in a disconnected environment
// and validates that required mirror configurations (ImageTagMirrorSet) are present.
// This should be called at the beginning of test cases that support disconnected environments.
//
// The function recognizes three types of cluster network access:
//  1. Connected: Direct access to external networks (no proxy, no disconnected)
//  2. Proxy: Access through HTTP/HTTPS proxy (treated as connected)
//  3. Disconnected: No external access, requires ImageTagMirrorSet for image mirroring
//
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Behavior:
//   - Skips the test if master node cannot be accessed (cannot determine environment)
//   - Returns immediately if proxy cluster detected (no mirror validation needed)
//   - Skips the test if in disconnected environment but ImageTagMirrorSet is not configured
//   - Continues normally if in connected environment or disconnected with proper configuration
//
// Usage:
//
//	g.It("test case supporting disconnected", func() {
//	    olmv1util.ValidateAccessEnvironment(oc)
//	    // rest of test code
//	})
func ValidateAccessEnvironment(oc *exutil.CLI) {
	// First check if this is a proxy cluster
	// Proxy clusters can access external networks through proxy, so they don't need mirror validation
	if IsProxyCluster(oc) {
		e2e.Logf("Proxy cluster detected, treating as connected environment (no mirror validation needed)")
		return
	}

	// Check if we can access external network directly
	hasNetwork := HasExternalNetworkAccess(oc)

	// If connected (and not proxy, already checked above), no validation needed
	if hasNetwork {
		e2e.Logf("Cluster has external network access (connected environment), no mirror validation needed")
		return
	}

	// In disconnected environment (not proxy, no external access), check for required ImageTagMirrorSet
	e2e.Logf("Cluster is in disconnected environment, validating ImageTagMirrorSet configuration")

	// Check if ImageTagMirrorSet "image-policy-aosqe" exists
	itmsOutput, itmsErr := oc.AsAdmin().WithoutNamespace().Run("get").Args("imagetagmirrorset", "image-policy-aosqe", "--ignore-not-found").Output()
	if itmsErr != nil || !strings.Contains(itmsOutput, "image-policy-aosqe") {
		g.Skip(fmt.Sprintf("Disconnected environment detected but ImageTagMirrorSet 'image-policy-aosqe' is not configured. "+
			"This test requires proper mirror configuration to run in disconnected clusters. "+
			"ITMS check result: output=%q, error=%v", itmsOutput, itmsErr))
	}

	e2e.Logf("Disconnected environment validation passed: ImageTagMirrorSet 'image-policy-aosqe' is configured")
}
