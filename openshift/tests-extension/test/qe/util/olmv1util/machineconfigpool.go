package olmv1util

import (
	"context"
	"fmt"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

// AssertMCPCondition checks a MachineConfigPool condition and ensures it remains consistent over time
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - name: name of the MachineConfigPool (e.g., "master", "worker")
//   - conditionType: type of condition to check (e.g., "Updating", "Updated", "Degraded")
//   - field: specific field within the condition (e.g., "status", "reason", "message")
//   - expect: expected value for the field
//   - checkInterval: interval between checks in seconds
//   - checkTimeout: maximum time to wait in seconds
//   - consistentTime: time in seconds to verify value remains consistent (0 to skip consistency check)
func AssertMCPCondition(oc *exutil.CLI, name, conditionType, field, expect string, checkInterval, checkTimeout, consistentTime int) {
	e2e.Logf("========= assert mcp %v %s %s expect is %s =========", name, conditionType, field, expect)
	err := CheckMCPCondition(oc, name, conditionType, field, expect, checkInterval, checkTimeout)
	o.Expect(err).NotTo(o.HaveOccurred())
	if consistentTime != 0 {
		e2e.Logf("make sure mcp %s expect is %s consistently for %ds", conditionType, expect, consistentTime)
		jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, conditionType, field)
		o.Consistently(func() string {
			output, _ := GetNoEmpty(oc, "mcp", name, "-o", jsonpath)
			return strings.ToLower(output)
		}, time.Duration(consistentTime)*time.Second, 5*time.Second).Should(o.ContainSubstring(strings.ToLower(expect)),
			"mcp %s expected is not %s", conditionType, expect)
	}
}

// CheckMCPCondition checks if a MachineConfigPool condition field matches the expected value within timeout
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - name: name of the MachineConfigPool (e.g., "master", "worker")
//   - conditionType: type of condition to check (e.g., "Updating", "Updated", "Degraded")
//   - field: specific field within the condition (e.g., "status", "reason", "message")
//   - expect: expected value for the field
//   - checkInterval: interval between checks in seconds
//   - checkTimeout: maximum time to wait in seconds
//
// Returns:
//   - error: error if condition is not met within timeout, nil if expectation is satisfied
func CheckMCPCondition(oc *exutil.CLI, name, conditionType, field, expect string, checkInterval, checkTimeout int) error {
	e2e.Logf("========= check mcp %v %s %s expect is %s =========", name, conditionType, field, expect)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, conditionType, field)
	errWait := wait.PollUntilContextTimeout(context.TODO(), time.Duration(checkInterval)*time.Second, time.Duration(checkTimeout)*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "mcp", name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("output is %v, error is %v, and try next", output, err)
			return false, nil
		}
		if !strings.Contains(strings.ToLower(output), strings.ToLower(expect)) {
			e2e.Logf("got is %v, not %v, and try next", output, expect)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := GetNoEmpty(oc, "mcp", name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get mcp debug info: %v", debugErr)
		} else {
			e2e.Logf("MCP debug status: %s", debugOutput)
		}
		errWait = fmt.Errorf("error happen: %v\n mcp %s expected is not %s in %v seconds", errWait, conditionType, expect, checkTimeout)
	}
	return errWait
}

// HealthyMCP4OLM checks if MachineConfigPools are healthy specifically for OLM operations
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - bool: true if all MCPs are healthy for OLM operations, false otherwise
func HealthyMCP4OLM(oc *exutil.CLI) bool {
	return HealthyMCP4Module(oc, "OLM")
}

// HealthyMCP4Module checks if MachineConfigPools are healthy for a specific module
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - module: module name to apply specific health checks (currently supports "OLM")
//
// Returns:
//   - bool: true if all MCPs are healthy for the specified module, false otherwise
func HealthyMCP4Module(oc *exutil.CLI, module string) bool {
	output, err := GetNoEmpty(oc, "mcp", "-ojsonpath={.items..metadata.name}")
	if err != nil {
		e2e.Logf("output is %v, error is %v, and try next", output, err)
		return false
	}
	// if your module has specific checking or not same checking with OLM. you could add your module branch
	// and please keep OLM logic
	if module == "OLM" {
		return validateOLMMachineConfigPools(oc, output)
	}

	return true
}

// validateOLMMachineConfigPools validates MCP status specifically for OLM module
func validateOLMMachineConfigPools(oc *exutil.CLI, output string) bool {
	mcpNames := strings.Fields(output)

	// Check if there are only expected MCPs
	if len(mcpNames) > 2 {
		e2e.Logf("there is unexpected mcp: %v", mcpNames)
		return false
	}

	for _, name := range mcpNames {
		if name != "worker" && name != "master" {
			e2e.Logf("there is mcp %v which is not expected", name)
			return false
		}
	}

	// Validate worker MCP status
	if !validateMCPStatus(oc, "worker") {
		return false
	}

	// Validate master MCP status
	return validateMCPStatus(oc, "master")
}

// validateMCPStatus validates the status of a specific MCP
func validateMCPStatus(oc *exutil.CLI, mcpName string) bool {
	status, err := GetMCPStatus(oc, mcpName)
	if err != nil {
		e2e.Logf("error getting %s status: %v", mcpName, err)
		return false
	}

	if !strings.Contains(status.UpdatingStatus, "False") ||
		status.MachineCount != status.ReadyMachineCount ||
		status.UnavailableMachineCount != status.DegradedMachineCount ||
		status.DegradedMachineCount != "0" {
		e2e.Logf("mcp %s's status is not correct: %v", mcpName, status)
		return false
	}

	return true
}

type McpStatus struct {
	MachineCount            string
	ReadyMachineCount       string
	UnavailableMachineCount string
	DegradedMachineCount    string
	UpdatingStatus          string
	UpdatedStatus           string
}

// GetMCPStatus retrieves comprehensive status information for a specific MachineConfigPool
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - name: name of the MachineConfigPool to get status for
//
// Returns:
//   - McpStatus: struct containing machine counts and condition status information
//   - error: error if any status field retrieval fails, nil on success
func GetMCPStatus(oc *exutil.CLI, name string) (McpStatus, error) {
	updatingStatus, err := GetNoEmpty(oc, "mcp", name, `-ojsonpath='{.status.conditions[?(@.type=="Updating")].status}'`)
	if err != nil {
		return McpStatus{}, err
	}
	updatedStatus, err := GetNoEmpty(oc, "mcp", name, `-ojsonpath='{.status.conditions[?(@.type=="Updated")].status}'`)
	if err != nil {
		return McpStatus{}, err
	}
	machineCount, err := GetNoEmpty(oc, "mcp", name, "-o=jsonpath={..status.machineCount}")
	if err != nil {
		return McpStatus{}, err
	}
	readyMachineCount, err := GetNoEmpty(oc, "mcp", name, "-o=jsonpath={..status.readyMachineCount}")
	if err != nil {
		return McpStatus{}, err
	}
	unavailableMachineCount, err := GetNoEmpty(oc, "mcp", name, "-o=jsonpath={..status.unavailableMachineCount}")
	if err != nil {
		return McpStatus{}, err
	}
	degradedMachineCount, err := GetNoEmpty(oc, "mcp", name, "-o=jsonpath={..status.degradedMachineCount}")
	if err != nil {
		return McpStatus{}, err
	}
	return McpStatus{
		MachineCount:            machineCount,
		ReadyMachineCount:       readyMachineCount,
		UnavailableMachineCount: unavailableMachineCount,
		DegradedMachineCount:    degradedMachineCount,
		UpdatingStatus:          updatingStatus,
		UpdatedStatus:           updatedStatus,
	}, nil
}
