package olmv1util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

// ErrPodNotFoundDuringVerification indicates that the pod was not found during env var verification
// This typically happens when a pod is being deleted/recreated during a rolling update
// Tests can check for this error and skip instead of failing
var ErrPodNotFoundDuringVerification = errors.New("pod not found during verification")

// VerifyDeploymentEnvVars verifies that a deployment has the expected environment variables
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - deploymentName: name of the Deployment to verify
//   - namespace: namespace of the Deployment
//   - expectedEnvVars: map of expected env var name to value
//   - timeout: maximum time to wait for env vars to be set correctly
//
// Returns:
//   - error: error if verification fails, nil on success
func VerifyDeploymentEnvVars(oc *exutil.CLI, deploymentName, namespace string, expectedEnvVars map[string]string, timeout time.Duration) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if deploymentName == "" {
		return fmt.Errorf("deployment name cannot be empty")
	}
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	e2e.Logf("========= Verifying Deployment %s/%s has expected env vars (timeout: %v) =========", namespace, deploymentName, timeout)

	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		// Get deployment env vars
		jsonPath := `jsonpath={.spec.template.spec.containers[0].env[*].name}`
		envNames, err := GetNoEmpty(oc, "deployment", deploymentName, "-n", namespace, "-o", jsonPath)
		if err != nil {
			e2e.Logf("Failed to get deployment env var names: %v, retrying...", err)
			return false, nil
		}

		e2e.Logf("Deployment env var names: %s", envNames)

		// Check each expected env var
		for expectedName, expectedValue := range expectedEnvVars {
			if !strings.Contains(envNames, expectedName) {
				e2e.Logf("Deployment missing expected env var: %s, retrying...", expectedName)
				return false, nil
			}

			// Get the value for this specific env var
			valueJsonPath := fmt.Sprintf(`jsonpath={.spec.template.spec.containers[0].env[?(@.name=="%s")].value}`, expectedName)
			actualValue, err := GetNoEmpty(oc, "deployment", deploymentName, "-n", namespace, "-o", valueJsonPath)
			if err != nil {
				e2e.Logf("Failed to get env var %s value: %v, retrying...", expectedName, err)
				return false, nil
			}

			if actualValue != expectedValue {
				e2e.Logf("Env var %s has value '%s', expected '%s', retrying...", expectedName, actualValue, expectedValue)
				return false, nil
			}

			e2e.Logf("Env var %s=%s verified in deployment", expectedName, expectedValue)
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("failed to verify deployment env vars within timeout %v: %w", timeout, err)
	}

	e2e.Logf("All expected env vars verified in deployment %s/%s", namespace, deploymentName)
	return nil
}

// VerifyPodEnvVars verifies that a pod has the expected environment variables at runtime
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - podName: name of the Pod to verify
//   - namespace: namespace of the Pod
//   - containerName: name of the container to check (empty string for first container)
//   - expectedEnvVars: map of expected env var name to value
//   - timeout: maximum time to wait for pod to be ready and env vars to be accessible
//
// Returns:
//   - error: error if verification fails, nil on success
func VerifyPodEnvVars(oc *exutil.CLI, podName, namespace, containerName string, expectedEnvVars map[string]string, timeout time.Duration) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if podName == "" {
		return fmt.Errorf("pod name cannot be empty")
	}
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	e2e.Logf("========= Verifying Pod %s/%s has expected env vars at runtime (timeout: %v) =========", namespace, podName, timeout)

	// Track if all errors are "pod not found" errors
	var lastError error
	allErrorsArePodNotFound := true
	hasAtLeastOneError := false

	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		// Check each expected env var in the running pod
		for expectedName, expectedValue := range expectedEnvVars {
			execArgs := []string{"exec", "-n", namespace, podName}
			if containerName != "" {
				execArgs = append(execArgs, "-c", containerName)
			}
			execArgs = append(execArgs, "--", "printenv", expectedName)

			actualValue, err := oc.WithoutNamespace().AsAdmin().Run(execArgs...).Args().Output()
			if err != nil {
				hasAtLeastOneError = true
				lastError = err
				errMsg := err.Error()

				// Check if this is a "pod not found" error
				isPodNotFound := strings.Contains(errMsg, "NotFound") || strings.Contains(errMsg, "not found")
				if !isPodNotFound {
					allErrorsArePodNotFound = false
				}

				e2e.Logf("Failed to get env var %s from pod: %v, retrying...", expectedName, err)
				return false, nil
			}

			// If we got here, we successfully executed command, so it's not a "pod not found" scenario
			allErrorsArePodNotFound = false

			// Trim whitespace
			actualValue = strings.TrimSpace(actualValue)

			if actualValue != expectedValue {
				e2e.Logf("Pod env var %s has value '%s', expected '%s', retrying...", expectedName, actualValue, expectedValue)
				return false, nil
			}

			e2e.Logf("Env var %s=%s verified in running pod", expectedName, expectedValue)
		}

		return true, nil
	})

	if err != nil {
		// If all errors were "pod not found" errors, return the special error
		if hasAtLeastOneError && allErrorsArePodNotFound {
			e2e.Logf("All verification attempts failed with 'pod not found' errors - pod may have been deleted during rolling update")
			return fmt.Errorf("%w: %v", ErrPodNotFoundDuringVerification, lastError)
		}
		return fmt.Errorf("failed to verify pod env vars within timeout %v: %w", timeout, err)
	}

	e2e.Logf("All expected env vars verified in running pod %s/%s", namespace, podName)
	return nil
}

// GetOperatorDeploymentName gets the deployment name created by a ClusterExtension
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - namespace: namespace where the operator is installed
//   - labelSelector: label selector to find the deployment (e.g., "app.kubernetes.io/name=operator")
//   - timeout: maximum time to wait for deployment to be created
//
// Returns:
//   - string: deployment name
//   - error: error if deployment not found or multiple deployments found
func GetOperatorDeploymentName(oc *exutil.CLI, namespace string, labelSelector string, timeout time.Duration) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	if namespace == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}

	e2e.Logf("========= Getting operator deployment in namespace %s with selector %s (timeout: %v) =========", namespace, labelSelector, timeout)

	var deploymentName string
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		args := []string{"get", "deployment", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}"}
		if labelSelector != "" {
			args = append(args, "-l", labelSelector)
		}

		deploymentNames, err := oc.WithoutNamespace().AsAdmin().Run(args...).Args().Output()
		if err != nil {
			e2e.Logf("Failed to get deployments: %v, retrying...", err)
			return false, nil
		}

		deploymentNames = strings.TrimSpace(deploymentNames)
		if deploymentNames == "" {
			e2e.Logf("No deployments found in namespace %s yet, retrying...", namespace)
			return false, nil
		}

		// Split by space in case there are multiple deployments
		names := strings.Fields(deploymentNames)
		if len(names) == 0 {
			e2e.Logf("No deployments found in namespace %s yet, retrying...", namespace)
			return false, nil
		}

		if len(names) > 1 {
			e2e.Logf("Multiple deployments found: %v, using first one: %s", names, names[0])
		}

		deploymentName = names[0]
		e2e.Logf("Found operator deployment: %s", deploymentName)
		return true, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to find deployment in namespace %s within timeout %v: %w", namespace, timeout, err)
	}

	return deploymentName, nil
}

// GetOperatorPodName gets a pod name created by an operator deployment
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - namespace: namespace where the operator is installed
//   - deploymentName: name of the deployment
//   - timeout: maximum time to wait for pod to be created
//
// Returns:
//   - string: pod name
//   - error: error if pod not found
func GetOperatorPodName(oc *exutil.CLI, namespace string, deploymentName string, timeout time.Duration) (string, error) {
	if oc == nil {
		return "", fmt.Errorf("CLI client cannot be nil")
	}
	if namespace == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}
	if deploymentName == "" {
		return "", fmt.Errorf("deployment name cannot be empty")
	}

	e2e.Logf("========= Getting pod for deployment %s in namespace %s (timeout: %v) =========", deploymentName, namespace, timeout)

	var podName string
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		// Method 1: Get the deployment's selector and use it to find pods
		selectorJSON, err := oc.WithoutNamespace().AsAdmin().Run("get", "deployment", deploymentName,
			"-n", namespace,
			"-o", "jsonpath={.spec.selector.matchLabels}").Args().Output()

		if err == nil && selectorJSON != "" {
			e2e.Logf("Deployment selector: %s", selectorJSON)

			// Parse selector JSON to build label selector string
			// Example: {"app":"name","version":"v1"} -> "app=name,version=v1"
			var selector map[string]string
			if err := json.Unmarshal([]byte(selectorJSON), &selector); err == nil && len(selector) > 0 {
				var labelPairs []string
				for k, v := range selector {
					labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
				}
				labelSelector := strings.Join(labelPairs, ",")

				e2e.Logf("Using deployment selector: %s", labelSelector)
				foundPodName, err := oc.WithoutNamespace().AsAdmin().Run("get", "pod", "-n", namespace,
					"-l", labelSelector,
					"-o", "jsonpath={.items[0].metadata.name}").Args().Output()

				if err == nil && strings.TrimSpace(foundPodName) != "" {
					podName = strings.TrimSpace(foundPodName)
					e2e.Logf("Found pod using deployment selector: %s", podName)
					return true, nil
				}
			}
		}

		// Method 2: Get pods with deployment name prefix (fallback)
		e2e.Logf("Trying fallback method: finding pods with name prefix %s", deploymentName)
		allPods, err := oc.WithoutNamespace().AsAdmin().Run("get", "pod", "-n", namespace,
			"-o", "jsonpath={.items[*].metadata.name}").Args().Output()

		if err == nil && allPods != "" {
			pods := strings.Fields(allPods)
			for _, pod := range pods {
				if strings.HasPrefix(pod, deploymentName) {
					podName = pod
					e2e.Logf("Found pod using name prefix: %s", podName)
					return true, nil
				}
			}
		}

		// If all methods fail on this iteration, log and retry
		e2e.Logf("No pod found yet for deployment %s, retrying...", deploymentName)
		return false, nil
	})

	if err != nil {
		// If all methods fail after timeout, list all pods for debugging
		e2e.Logf("Failed to find pod within timeout, listing all pods in namespace %s for debugging:", namespace)
		podList, _ := oc.WithoutNamespace().AsAdmin().Run("get", "pods", "-n", namespace, "-o", "wide").Args().Output()
		e2e.Logf("All pods:\n%s", podList)
		return "", fmt.Errorf("no pods found for deployment %s in namespace %s within timeout %v: %w", deploymentName, namespace, timeout, err)
	}

	return podName, nil
}

// WaitForDeploymentRollout waits for a deployment to complete rollout with updated configuration
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - deploymentName: name of the Deployment
//   - namespace: namespace of the Deployment
//   - timeout: maximum time to wait
//
// Returns:
//   - error: error if rollout doesn't complete in time, nil on success
func WaitForDeploymentRollout(oc *exutil.CLI, deploymentName, namespace string, timeout time.Duration) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if deploymentName == "" {
		return fmt.Errorf("deployment name cannot be empty")
	}
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	e2e.Logf("========= Waiting for deployment %s/%s rollout to complete (timeout: %v) =========", namespace, deploymentName, timeout)

	var lastStatus string
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		// Check if deployment is available
		availableReplicas, err := GetNoEmpty(oc, "deployment", deploymentName, "-n", namespace,
			"-o", "jsonpath={.status.availableReplicas}")
		if err != nil {
			e2e.Logf("Failed to get available replicas: %v, retrying...", err)
			return false, nil
		}

		desiredReplicas, err := GetNoEmpty(oc, "deployment", deploymentName, "-n", namespace,
			"-o", "jsonpath={.spec.replicas}")
		if err != nil {
			e2e.Logf("Failed to get desired replicas: %v, retrying...", err)
			return false, nil
		}

		updatedReplicas, err := GetNoEmpty(oc, "deployment", deploymentName, "-n", namespace,
			"-o", "jsonpath={.status.updatedReplicas}")
		if err != nil {
			e2e.Logf("Failed to get updated replicas: %v, retrying...", err)
			return false, nil
		}

		readyReplicas, _ := GetNoEmpty(oc, "deployment", deploymentName, "-n", namespace,
			"-o", "jsonpath={.status.readyReplicas}")

		currentStatus := fmt.Sprintf("desired=%s, updated=%s, ready=%s, available=%s",
			desiredReplicas, updatedReplicas, readyReplicas, availableReplicas)

		if currentStatus != lastStatus {
			e2e.Logf("Deployment %s status: %s", deploymentName, currentStatus)
			lastStatus = currentStatus
		}

		// Check if all replicas are updated and available
		if availableReplicas == desiredReplicas && updatedReplicas == desiredReplicas && desiredReplicas != "" && desiredReplicas != "0" {
			e2e.Logf("Deployment %s rollout complete!", deploymentName)
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		// Get detailed status for debugging
		e2e.Logf("Deployment rollout failed, getting detailed status...")
		status, _ := oc.WithoutNamespace().AsAdmin().Run("get", "deployment", deploymentName, "-n", namespace, "-o", "yaml").Args().Output()
		e2e.Logf("Deployment status:\n%s", status)

		pods, _ := oc.WithoutNamespace().AsAdmin().Run("get", "pods", "-n", namespace, "-l", fmt.Sprintf("app=%s", deploymentName), "-o", "wide").Args().Output()
		e2e.Logf("Pods:\n%s", pods)

		return fmt.Errorf("deployment %s rollout did not complete: %w", deploymentName, err)
	}

	e2e.Logf("Deployment %s/%s rollout completed successfully", namespace, deploymentName)
	return nil
}

// DumpDeploymentManifest logs the complete deployment manifest for debugging
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - deploymentName: name of the Deployment
//   - namespace: namespace of the Deployment
func DumpDeploymentManifest(oc *exutil.CLI, deploymentName, namespace string) {
	if oc == nil || deploymentName == "" || namespace == "" {
		e2e.Logf("Invalid parameters for DumpDeploymentManifest")
		return
	}

	e2e.Logf("========= Dumping Deployment Manifest: %s/%s =========", namespace, deploymentName)

	// Get full deployment YAML
	manifest, err := oc.WithoutNamespace().AsAdmin().Run("get", "deployment", deploymentName,
		"-n", namespace,
		"-o", "yaml").Args().Output()
	if err != nil {
		e2e.Logf("Failed to get deployment manifest: %v", err)
		return
	}

	e2e.Logf("Deployment Manifest:\n%s", manifest)
	e2e.Logf("========= End of Deployment Manifest =========")
}

// DumpPodManifest logs the complete pod manifest for debugging
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - podName: name of the Pod
//   - namespace: namespace of the Pod
func DumpPodManifest(oc *exutil.CLI, podName, namespace string) {
	if oc == nil || podName == "" || namespace == "" {
		e2e.Logf("Invalid parameters for DumpPodManifest")
		return
	}

	e2e.Logf("========= Dumping Pod Manifest: %s/%s =========", namespace, podName)

	// Get full pod YAML
	manifest, err := oc.WithoutNamespace().AsAdmin().Run("get", "pod", podName,
		"-n", namespace,
		"-o", "yaml").Args().Output()
	if err != nil {
		e2e.Logf("Failed to get pod manifest: %v", err)
		return
	}

	e2e.Logf("Pod Manifest:\n%s", manifest)
	e2e.Logf("========= End of Pod Manifest =========")
}

// ExtractReplicaSetHash extracts the ReplicaSet hash from a pod name.
// Pod name format: <deployment-name>-<replicaset-hash>-<random-suffix>
// Example: nginx-ok-v87554-controller-manager-5497779f6b-4l2hv -> 5497779f6b
//
// Parameters:
//   - podName: the name of the pod
//
// Returns:
//   - string: the ReplicaSet hash, or empty string if extraction fails
func ExtractReplicaSetHash(podName string) string {
	parts := strings.Split(podName, "-")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}
