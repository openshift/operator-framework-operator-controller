package olmv1util

import (
	"context"
	"fmt"
	"time"

	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

type ItdmsDescription struct {
	Name            string
	MirrorSite      string
	SourceSite      string
	MirrorNamespace string
	SourceNamespace string
	Template        string
}

// Create creates an ImageTagDigestMirrorSet and waits for Machine Config Pool updates to complete
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (itdms *ItdmsDescription) Create(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(itdms.Name).NotTo(o.BeEmpty(), "ImageTagDigestMirrorSet name cannot be empty")
	e2e.Logf("=========Create itdms %v=========", itdms.Name)
	err := itdms.CreateWithoutCheck(oc)
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

// CreateWithoutCheck creates an ImageTagDigestMirrorSet from template without waiting for MCP updates
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - error: error if template application fails, nil on success
func (itdms *ItdmsDescription) CreateWithoutCheck(oc *exutil.CLI) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if itdms.Template == "" {
		return fmt.Errorf("template path cannot be empty")
	}
	e2e.Logf("=========CreateWithoutCheck itdms %v=========", itdms.Name)
	parameters := getParameters(itdms)
	err := exutil.ApplyClusterResourceFromTemplateWithError(oc, parameters...)
	return err
}

// DeleteWithoutCheck removes the ImageTagDigestMirrorSet resource without waiting for MCP recovery
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (itdms *ItdmsDescription) DeleteWithoutCheck(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========DeleteWithoutCheck itdms %v=========", itdms.Name)
	parameters := getParameters(itdms)

	var configFile string
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 9*time.Second, false, func(ctx context.Context) (bool, error) {
		stdout, _, err := oc.AsAdmin().Run("process").Args(parameters...).OutputsToFiles(exutil.GetRandomString() + "config.json")
		if err != nil {
			e2e.Logf("the err:%v, and try next round", err)
			return false, nil
		}

		configFile = stdout
		return true, nil
	})
	exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("fail to process %v", parameters))

	err := oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFile).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Delete removes the ImageTagDigestMirrorSet and waits for Machine Config Pool recovery
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (itdms *ItdmsDescription) Delete(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========Delete itdms %v=========", itdms.Name)
	itdms.DeleteWithoutCheck(oc)
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
	}, 600*time.Second, 30*time.Second).Should(o.BeTrue(), "mcp is not recovered after delete itdms")
}

// getParameters builds template processing parameters from ItdmsDescription fields
// Parameters:
//   - itdms: ItdmsDescription containing configuration values for template processing
//
// Returns:
//   - []string: slice of template parameters including template file and variable assignments
func getParameters(itdms *ItdmsDescription) []string {
	if itdms == nil {
		return nil
	}
	parameters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", itdms.Template, "-p"}
	if len(itdms.Name) > 0 {
		parameters = append(parameters, "NAME="+itdms.Name)
	}
	if len(itdms.MirrorSite) > 0 {
		parameters = append(parameters, "MIRRORSITE="+itdms.MirrorSite)
	}
	if len(itdms.SourceSite) > 0 {
		parameters = append(parameters, "SOURCESITE="+itdms.SourceSite)
	}
	if len(itdms.MirrorNamespace) > 0 {
		parameters = append(parameters, "MIRRORNAMESPACE="+itdms.MirrorNamespace)
	}
	if len(itdms.SourceNamespace) > 0 {
		parameters = append(parameters, "SOURCENAMESPACE="+itdms.SourceNamespace)
	}
	return parameters
}
