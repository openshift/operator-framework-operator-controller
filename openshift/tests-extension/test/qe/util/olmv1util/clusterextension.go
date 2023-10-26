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

type ClusterExtensionDescription struct {
	Name                    string
	PackageName             string
	Channel                 string
	Version                 string
	InstallNamespace        string
	WatchNamespace          string
	SaName                  string
	UpgradeConstraintPolicy string
	LabelKey                string // default is olmv1-test
	LabelValue              string // suggest to use case id
	ExpressionsKey          string
	ExpressionsOperator     string
	ExpressionsValue1       string
	ExpressionsValue2       string
	ExpressionsValue3       string
	SourceType              string
	Template                string
	InstalledBundle         string
}

// Create creates a ClusterExtension and waits for successful installation and bundle resource retrieval
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clusterextension *ClusterExtensionDescription) Create(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(clusterextension.Name).NotTo(o.BeEmpty(), "ClusterExtension name cannot be empty")
	e2e.Logf("=========Create clusterextension %v=========", clusterextension.Name)
	err := clusterextension.CreateWithoutCheck(oc)
	o.Expect(err).NotTo(o.HaveOccurred())
	clusterextension.CheckClusterExtensionCondition(oc, "Progressing", "reason", "Succeeded", 3, 150, 0)
	clusterextension.WaitClusterExtensionCondition(oc, "Installed", "True", 0)
	clusterextension.GetBundleResource(oc)
}

// CreateWithoutCheck creates a ClusterExtension from template without waiting for installation status
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - error: error if template application fails, nil on success
func (clusterextension *ClusterExtensionDescription) CreateWithoutCheck(oc *exutil.CLI) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if clusterextension.Template == "" {
		return fmt.Errorf("template path cannot be empty")
	}
	e2e.Logf("=========CreateWithoutCheck clusterextension %v=========", clusterextension.Name)
	parameters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", clusterextension.Template, "-p"}
	if len(clusterextension.Name) > 0 {
		parameters = append(parameters, "NAME="+clusterextension.Name)
	}
	if len(clusterextension.PackageName) > 0 {
		parameters = append(parameters, "PACKAGE="+clusterextension.PackageName)
	}
	if len(clusterextension.Channel) > 0 {
		parameters = append(parameters, "CHANNEL="+clusterextension.Channel)
	}
	if len(clusterextension.Version) > 0 {
		parameters = append(parameters, "VERSION="+clusterextension.Version)
	}
	if len(clusterextension.InstallNamespace) > 0 {
		parameters = append(parameters, "INSTALLNAMESPACE="+clusterextension.InstallNamespace)
	}
	if len(clusterextension.WatchNamespace) > 0 {
		parameters = append(parameters, "WATCHNS="+clusterextension.WatchNamespace)
	}
	if len(clusterextension.SaName) > 0 {
		parameters = append(parameters, "SANAME="+clusterextension.SaName)
	}
	if len(clusterextension.UpgradeConstraintPolicy) > 0 {
		parameters = append(parameters, "POLICY="+clusterextension.UpgradeConstraintPolicy)
	}
	if len(clusterextension.LabelKey) > 0 {
		parameters = append(parameters, "LABELKEY="+clusterextension.LabelKey)
	}
	if len(clusterextension.LabelValue) > 0 {
		parameters = append(parameters, "LABELVALUE="+clusterextension.LabelValue)
	}
	if len(clusterextension.ExpressionsKey) > 0 {
		parameters = append(parameters, "EXPRESSIONSKEY="+clusterextension.ExpressionsKey)
	}
	if len(clusterextension.ExpressionsOperator) > 0 {
		parameters = append(parameters, "EXPRESSIONSOPERATOR="+clusterextension.ExpressionsOperator)
	}
	if len(clusterextension.ExpressionsValue1) > 0 {
		parameters = append(parameters, "EXPRESSIONSVALUE1="+clusterextension.ExpressionsValue1)
	}
	if len(clusterextension.ExpressionsValue2) > 0 {
		parameters = append(parameters, "EXPRESSIONSVALUE2="+clusterextension.ExpressionsValue2)
	}
	if len(clusterextension.ExpressionsValue3) > 0 {
		parameters = append(parameters, "EXPRESSIONSVALUE3="+clusterextension.ExpressionsValue3)
	}
	if len(clusterextension.SourceType) > 0 {
		parameters = append(parameters, "SOURCETYPE="+clusterextension.SourceType)
	}
	err := exutil.ApplyClusterResourceFromTemplateWithError(oc, parameters...)
	return err
}

// WaitClusterExtensionCondition waits for a specific condition status on the ClusterExtension
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - conditionType: type of condition to check (e.g., "Installed", "Progressing")
//   - status: expected status value (e.g., "True", "False")
//   - consistentTime: time in seconds to verify status remains consistent (0 to skip consistency check)
func (clusterextension *ClusterExtensionDescription) WaitClusterExtensionCondition(oc *exutil.CLI, conditionType string, status string, consistentTime int) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("========= wait clusterextension %v %s status is %s =========", clusterextension.Name, conditionType, status)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].status}`, conditionType)
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("output is %v, error is %v, and try next", output, err)
			return false, nil
		}
		if !strings.Contains(strings.ToLower(output), strings.ToLower(status)) {
			e2e.Logf("status is %v, not %v, and try next", output, status)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clusterextension %s status is not %s", conditionType, status))
	}
	if consistentTime != 0 {
		e2e.Logf("make sure clusterextension %s status is %s consistently for %ds", conditionType, status, consistentTime)
		o.Consistently(func() string {
			output, _ := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
			return strings.ToLower(output)
		}, time.Duration(consistentTime)*time.Second, 5*time.Second).Should(o.ContainSubstring(strings.ToLower(status)),
			"clusterextension %s status is not %s", conditionType, status)
	}
}

// CheckClusterExtensionCondition checks a specific field within a condition type of the ClusterExtension
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - conditionType: type of condition to check (e.g., "Progressing", "Installed")
//   - field: specific field within the condition (e.g., "status", "reason", "message")
//   - expect: expected value for the field
//   - checkInterval: interval between checks in seconds
//   - checkTimeout: maximum time to wait in seconds
//   - consistentTime: time in seconds to verify value remains consistent (0 to skip consistency check)
func (clusterextension *ClusterExtensionDescription) CheckClusterExtensionCondition(oc *exutil.CLI, conditionType, field, expect string, checkInterval, checkTimeout, consistentTime int) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("========= check clusterextension %v %s %s expect is %s =========", clusterextension.Name, conditionType, field, expect)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, conditionType, field)
	errWait := wait.PollUntilContextTimeout(context.TODO(), time.Duration(checkInterval)*time.Second, time.Duration(checkTimeout)*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
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
		if debugOutput, debugErr := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clusterextension %s expected is not %s in %v seconds", conditionType, expect, checkTimeout))
	}
	if consistentTime != 0 {
		e2e.Logf("make sure clusterextension %s expect is %s consistently for %ds", conditionType, expect, consistentTime)
		o.Consistently(func() string {
			output, _ := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
			return strings.ToLower(output)
		}, time.Duration(consistentTime)*time.Second, 5*time.Second).Should(o.ContainSubstring(strings.ToLower(expect)),
			"clusterextension %s expected is not %s", conditionType, expect)
	}
}

// CheckClusterExtensionNotCondition verifies a specific field within a condition does NOT contain the expected value
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - conditionType: type of condition to check (e.g., "Progressing", "Installed")
//   - field: specific field within the condition (e.g., "status", "reason", "message")
//   - expect: value that should NOT be present in the field
//   - checkInterval: interval between checks in seconds
//   - checkTimeout: maximum time to wait in seconds
//   - consistentTime: time in seconds to verify value remains absent (0 to skip consistency check)
func (clusterextension *ClusterExtensionDescription) CheckClusterExtensionNotCondition(oc *exutil.CLI, conditionType, field, expect string, checkInterval, checkTimeout, consistentTime int) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("========= check clusterextension %v %s %s expect is %s =========", clusterextension.Name, conditionType, field, expect)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, conditionType, field)
	errWait := wait.PollUntilContextTimeout(context.TODO(), time.Duration(checkInterval)*time.Second, time.Duration(checkTimeout)*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("output is %v, error is %v, and try next", output, err)
			return false, nil
		}
		if strings.Contains(strings.ToLower(output), strings.ToLower(expect)) {
			e2e.Logf("got is %v, still %v, and try next", output, expect)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clusterextension %s expected is still %s in %v seconds", conditionType, expect, checkTimeout))
	}
	if consistentTime != 0 {
		e2e.Logf("make sure clusterextension %s expect is %s consistently for %ds", conditionType, expect, consistentTime)
		o.Consistently(func() string {
			output, err := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
			o.Expect(err).NotTo(o.HaveOccurred())
			return strings.ToLower(output)
		}, time.Duration(consistentTime)*time.Second, 5*time.Second).ShouldNot(o.ContainSubstring(strings.ToLower(expect)),
			"clusterextension %s expected is still %s", conditionType, expect)
	}
}

// GetClusterExtensionMessage retrieves the message field from a specific condition type
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - conditionType: type of condition to get message from (e.g., "Progressing", "Installed")
//
// Returns:
//   - string: message content from the specified condition
func (clusterextension *ClusterExtensionDescription) GetClusterExtensionMessage(oc *exutil.CLI, conditionType string) string {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")

	var message string
	e2e.Logf("========= return clusterextension %v %s message =========", clusterextension.Name, conditionType)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].message}`, conditionType)
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
		var err error
		message, err = Get(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("message is %v, error is %v, and try next", message, err)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := Get(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("can't get clusterextension %s message", conditionType))
	}
	return message
}

// WaitProgressingMessage waits for the Progressing condition message to contain a specific substring
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - expect: substring that should be present in the Progressing message
func (clusterextension *ClusterExtensionDescription) WaitProgressingMessage(oc *exutil.CLI, expect string) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("========= wait clusterextension %v Progressing message includes %s =========", clusterextension.Name, expect)
	jsonpath := `jsonpath={.status.conditions[?(@.type=="Progressing")].message}`
	errWait := wait.PollUntilContextTimeout(context.TODO(), 6*time.Second, 180*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("output is %v, error is %v, and try next", output, err)
			return false, nil
		}
		if !strings.Contains(output, expect) {
			e2e.Logf("message is %v, not include %v, and try next", output, expect)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clusterextension progressing message does not include %s", expect))
	}
}

// GetClusterExtensionField retrieves a specific field value from a condition type
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - conditionType: type of condition to get field from (e.g., "Progressing", "Installed")
//   - field: specific field to retrieve (e.g., "status", "reason", "message")
//
// Returns:
//   - string: value of the specified field from the condition
func (clusterextension *ClusterExtensionDescription) GetClusterExtensionField(oc *exutil.CLI, conditionType, field string) string {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")

	var content string
	e2e.Logf("========= return clusterextension %v %s %s =========", clusterextension.Name, conditionType, field)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, conditionType, field)
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
		var err error
		content, err = Get(oc, "clusterextension", clusterextension.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("content is %v, error is %v, and try next", content, err)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := Get(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("can't get clusterextension %s %s", conditionType, field))
	}
	return content
}

// GetBundleResource retrieves and stores the installed bundle name for the ClusterExtension
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clusterextension *ClusterExtensionDescription) GetBundleResource(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========Get clusterextension %v BundleResource =========", clusterextension.Name)

	installedBundle, err := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.install.bundle.name}")
	if err != nil {
		if debugOutput, debugErr := Get(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
	}
	o.Expect(err).NotTo(o.HaveOccurred())
	clusterextension.InstalledBundle = installedBundle
}

// Patch applies a merge patch to the ClusterExtension resource
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - patch: JSON patch string to apply to the resource
func (clusterextension *ClusterExtensionDescription) Patch(oc *exutil.CLI, patch string) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	_, err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("clusterextension", clusterextension.Name, "--type", "merge", "-p", patch).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// DeleteWithoutCheck removes the ClusterExtension resource without verification
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clusterextension *ClusterExtensionDescription) DeleteWithoutCheck(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========DeleteWithoutCheck clusterextension %v=========", clusterextension.Name)
	exutil.CleanupResource(oc, 4*time.Second, 160*time.Second, exutil.AsAdmin, exutil.WithoutNamespace, "clusterextension", clusterextension.Name)
}

// Delete removes the ClusterExtension and performs cleanup operations
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clusterextension *ClusterExtensionDescription) Delete(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========Delete clusterextension %v=========", clusterextension.Name)
	clusterextension.DeleteWithoutCheck(oc)
	//add check later
}

// WaitClusterExtensionVersion waits for the ClusterExtension to reach a specific version
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - version: expected version string that should be present in the installed bundle name
func (clusterextension *ClusterExtensionDescription) WaitClusterExtensionVersion(oc *exutil.CLI, version string) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("========= wait clusterextension %v version is %s =========", clusterextension.Name, version)
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 120*time.Second, false, func(ctx context.Context) (bool, error) {
		installedBundle, _ := GetNoEmpty(oc, "clusterextension", clusterextension.Name, "-o", "jsonpath={.status.install.bundle.name}")
		if strings.Contains(installedBundle, version) {
			e2e.Logf("version is %v", installedBundle)
			return true, nil
		}
		e2e.Logf("version is %v, not %s, and try next", installedBundle, version)
		return false, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := Get(oc, "clusterextension", clusterextension.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster extension debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster extension debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clusterextension version is not %s", version))
	}
}
