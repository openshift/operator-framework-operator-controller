package olmv1util

import (
	"fmt"
	"time"

	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

type CipDescription struct {
	Name     string
	Repo1    string
	Repo2    string
	Repo3    string
	Repo4    string
	Policy   string
	Template string
}

// Create creates a ClusterImagePolicy and waits for Machine Config Pool updates to complete
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (cip *CipDescription) Create(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(cip.Name).NotTo(o.BeEmpty(), "ClusterImagePolicy name cannot be empty")
	e2e.Logf("=========Create cip %v=========", cip.Name)
	err := cip.CreateWithoutCheck(oc)
	o.Expect(err).NotTo(o.HaveOccurred())
	// start to update it
	AssertMCPCondition(oc, "master", "Updating", "status", "True", 3, 120, 5)
	AssertMCPCondition(oc, "worker", "Updating", "status", "True", 3, 120, 5)
	// AssertMCPCondition(oc, "master", "Updated", "status", "False", 3, 90)
	// AssertMCPCondition(oc, "worker", "Updated", "status", "False", 3, 90)
	// finish to update it
	AssertMCPCondition(oc, "master", "Updating", "status", "False", 30, 900, 10)
	AssertMCPCondition(oc, "worker", "Updating", "status", "False", 30, 900, 10)
	o.Expect(HealthyMCP4OLM(oc)).To(o.BeTrue())
	// AssertMCPCondition(oc, "master", "Updated", "status", "True", 5, 30)
	// AssertMCPCondition(oc, "worker", "Updated", "status", "True", 5, 30)
}

// CreateWithoutCheck creates a ClusterImagePolicy from template without waiting for MCP updates
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - error: error if template application fails, nil on success
func (cip *CipDescription) CreateWithoutCheck(oc *exutil.CLI) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if cip.Template == "" {
		return fmt.Errorf("template path cannot be empty")
	}
	e2e.Logf("=========CreateWithoutCheck cip %v=========", cip.Name)
	parameters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", cip.Template, "-p"}
	if len(cip.Name) > 0 {
		parameters = append(parameters, "NAME="+cip.Name)
	}
	if len(cip.Repo1) > 0 {
		parameters = append(parameters, "REPO1="+cip.Repo1)
	}
	if len(cip.Repo2) > 0 {
		parameters = append(parameters, "REPO2="+cip.Repo2)
	}
	if len(cip.Repo3) > 0 {
		parameters = append(parameters, "REPO3="+cip.Repo3)
	}
	if len(cip.Repo4) > 0 {
		parameters = append(parameters, "REPO4="+cip.Repo4)
	}
	if len(cip.Policy) > 0 {
		parameters = append(parameters, "POLICY="+cip.Policy)
	}
	err := exutil.ApplyClusterResourceFromTemplateWithError(oc, parameters...)
	return err
}

// DeleteWithoutCheck removes the ClusterImagePolicy resource without waiting for MCP recovery
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (cip *CipDescription) DeleteWithoutCheck(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========DeleteWithoutCheck cip %v=========", cip.Name)
	exutil.CleanupResource(oc, 4*time.Second, 160*time.Second, exutil.AsAdmin,
		exutil.WithoutNamespace, "ClusterImagePolicy", cip.Name)
}

// Delete removes the ClusterImagePolicy and waits for Machine Config Pool recovery
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (cip *CipDescription) Delete(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========Delete cip %v=========", cip.Name)
	cip.DeleteWithoutCheck(oc)
	// start to update it
	// AssertMCPCondition(oc, "master", "Updating", "status", "True", 3, 90, 5)
	// AssertMCPCondition(oc, "worker", "Updating", "status", "True", 3, 90, 5)
	// AssertMCPCondition(oc, "master", "Updated", "status", "False", 3, 90, 5)
	// AssertMCPCondition(oc, "worker", "Updated", "status", "False", 3, 90, 5)
	// finish to update it
	AssertMCPCondition(oc, "master", "Updating", "status", "False", 90, 900, 30)
	AssertMCPCondition(oc, "worker", "Updating", "status", "False", 30, 900, 10)
	// AssertMCPCondition(oc, "master", "Updated", "status", "True", 5, 30, 5)
	// AssertMCPCondition(oc, "worker", "Updated", "status", "True", 5, 30, 5)
	o.Eventually(func() bool {
		return HealthyMCP4OLM(oc)
	}, 600*time.Second, 30*time.Second).Should(o.BeTrue(), "mcp is not recovered after delete cip")
}
