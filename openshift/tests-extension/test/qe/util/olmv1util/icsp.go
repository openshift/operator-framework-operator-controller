package olmv1util

import (
	"fmt"
	"time"

	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

type IcspDescription struct {
	Name     string
	Mirror   string
	Source   string
	Template string
}

// Create creates an ImageContentSourcePolicy and waits for Machine Config Pool updates to complete
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (icsp *IcspDescription) Create(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(icsp.Name).NotTo(o.BeEmpty(), "ImageContentSourcePolicy name cannot be empty")
	e2e.Logf("=========Create icsp %v=========", icsp.Name)
	err := icsp.CreateWithoutCheck(oc)
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

// CreateWithoutCheck creates an ImageContentSourcePolicy from template without waiting for MCP updates
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - error: error if template application fails, nil on success
func (icsp *IcspDescription) CreateWithoutCheck(oc *exutil.CLI) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if icsp.Template == "" {
		return fmt.Errorf("template path cannot be empty")
	}
	e2e.Logf("=========CreateWithoutCheck icsp %v=========", icsp.Name)
	parameters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", icsp.Template, "-p"}
	if len(icsp.Name) > 0 {
		parameters = append(parameters, "NAME="+icsp.Name)
	}
	if len(icsp.Mirror) > 0 {
		parameters = append(parameters, "MIRROR="+icsp.Mirror)
	}
	if len(icsp.Source) > 0 {
		parameters = append(parameters, "SOURCE="+icsp.Source)
	}
	err := exutil.ApplyClusterResourceFromTemplateWithError(oc, parameters...)
	return err
}

// DeleteWithoutCheck removes the ImageContentSourcePolicy resource without waiting for MCP recovery
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (icsp *IcspDescription) DeleteWithoutCheck(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========DeleteWithoutCheck icsp %v=========", icsp.Name)
	exutil.CleanupResource(oc, 4*time.Second, 160*time.Second, exutil.AsAdmin,
		exutil.WithoutNamespace, "ImageContentSourcePolicy", icsp.Name)
}

// Delete removes the ImageContentSourcePolicy and waits for Machine Config Pool recovery
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (icsp *IcspDescription) Delete(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========Delete icsp %v=========", icsp.Name)
	icsp.DeleteWithoutCheck(oc)
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
	}, 600*time.Second, 30*time.Second).Should(o.BeTrue(), "mcp is not recovered after delete icsp")
}
