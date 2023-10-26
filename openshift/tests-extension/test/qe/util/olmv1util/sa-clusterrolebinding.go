package olmv1util

import (
	"fmt"
	"strings"

	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

type ChildResource struct {
	Kind  string
	Ns    string
	Names []string
}

type SaCLusterRolebindingDescription struct {
	Name      string
	Namespace string
	// if it take admin permission, no need to setup RBACObjects and take default value
	RBACObjects []ChildResource
	Kinds       string
	Template    string
}

// Create creates ServiceAccount, ClusterRole, and ClusterRoleBinding resources and waits for their appearance
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (sacrb *SaCLusterRolebindingDescription) Create(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(sacrb.Name).NotTo(o.BeEmpty(), "ServiceAccount ClusterRoleBinding name cannot be empty")
	e2e.Logf("=========Create sacrb %v=========", sacrb.Name)
	err := sacrb.CreateWithoutCheck(oc)
	o.Expect(err).NotTo(o.HaveOccurred())

	if len(sacrb.RBACObjects) != 0 {
		sacrb.validateCustomRBACObjects(oc)
	} else {
		sacrb.validateDefaultRBACObjects(oc)
	}
}

// CreateWithoutCheck creates RBAC resources from template without waiting for appearance verification
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - error: error if template application fails, nil on success
func (sacrb *SaCLusterRolebindingDescription) CreateWithoutCheck(oc *exutil.CLI) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if sacrb.Template == "" {
		return fmt.Errorf("template path cannot be empty")
	}
	e2e.Logf("=========CreateWithoutCheck sacrb %v=========", sacrb.Name)
	parameters := []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", sacrb.Template, "-p"}
	if len(sacrb.Name) > 0 {
		parameters = append(parameters, "NAME="+sacrb.Name)
	}
	if len(sacrb.Namespace) > 0 {
		parameters = append(parameters, "NAMESPACE="+sacrb.Namespace)
	}
	if len(sacrb.Kinds) > 0 {
		parameters = append(parameters, "KINDS="+sacrb.Kinds)
	}
	err := exutil.ApplyClusterResourceFromTemplateWithError(oc, parameters...)
	return err
}

// Delete removes ServiceAccount, ClusterRole, and ClusterRoleBinding resources
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (sacrb *SaCLusterRolebindingDescription) Delete(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	e2e.Logf("=========Delete sacrb %v=========", sacrb.Name)
	if len(sacrb.RBACObjects) != 0 {
		sacrb.cleanupCustomRBACObjects(oc)
	} else {
		sacrb.cleanupDefaultRBACObjects(oc)
	}
}

// validateCustomRBACObjects validates custom RBAC objects appearance
func (sacrb *SaCLusterRolebindingDescription) validateCustomRBACObjects(oc *exutil.CLI) {
	for _, object := range sacrb.RBACObjects {
		if strings.TrimSpace(object.Kind) == "" {
			e2e.Logf("Warning: empty Kind found in RBACObjects, skipping")
			continue
		}
		sacrb.validateObjectNames(oc, object)
	}
}

// validateDefaultRBACObjects validates default RBAC objects appearance
func (sacrb *SaCLusterRolebindingDescription) validateDefaultRBACObjects(oc *exutil.CLI) {
	o.Expect(Appearance(oc, exutil.Appear, "ServiceAccount", sacrb.Name, "-n", sacrb.Namespace)).To(o.BeTrue())
	o.Expect(Appearance(oc, exutil.Appear, "ClusterRole", fmt.Sprintf("%s-installer-admin-clusterrole", sacrb.Name))).To(o.BeTrue())
	o.Expect(Appearance(oc, exutil.Appear, "ClusterRoleBinding", fmt.Sprintf("%s-installer-admin-clusterrole-binding", sacrb.Name))).To(o.BeTrue())
}

// validateObjectNames validates each name in an RBAC object
func (sacrb *SaCLusterRolebindingDescription) validateObjectNames(oc *exutil.CLI, object ChildResource) {
	for _, name := range object.Names {
		if strings.TrimSpace(name) == "" {
			e2e.Logf("Warning: empty name found in RBACObjects for kind %s, skipping", object.Kind)
			continue
		}

		if object.Ns == "" {
			o.Expect(Appearance(oc, exutil.Appear, object.Kind, name)).To(o.BeTrue())
		} else {
			o.Expect(Appearance(oc, exutil.Appear, object.Kind, name, "-n", object.Ns)).To(o.BeTrue())
		}
	}
}

// cleanupCustomRBACObjects cleans up custom RBAC objects
func (sacrb *SaCLusterRolebindingDescription) cleanupCustomRBACObjects(oc *exutil.CLI) {
	for _, object := range sacrb.RBACObjects {
		if strings.TrimSpace(object.Kind) == "" {
			e2e.Logf("Warning: empty Kind found in RBACObjects, skipping cleanup")
			continue
		}
		sacrb.cleanupObjectNames(oc, object)
	}
}

// cleanupDefaultRBACObjects cleans up default RBAC objects
func (sacrb *SaCLusterRolebindingDescription) cleanupDefaultRBACObjects(oc *exutil.CLI) {
	Cleanup(oc, "ClusterRoleBinding", fmt.Sprintf("%s-installer-admin-clusterrole-binding", sacrb.Name))
	Cleanup(oc, "ClusterRole", fmt.Sprintf("%s-installer-admin-clusterrole", sacrb.Name))
	Cleanup(oc, "ServiceAccount", sacrb.Name, "-n", sacrb.Namespace)
}

// cleanupObjectNames cleans up each name in an RBAC object
func (sacrb *SaCLusterRolebindingDescription) cleanupObjectNames(oc *exutil.CLI, object ChildResource) {
	for _, name := range object.Names {
		if strings.TrimSpace(name) == "" {
			e2e.Logf("Warning: empty name found in RBACObjects for kind %s, skipping cleanup", object.Kind)
			continue
		}

		if object.Ns == "" {
			Cleanup(oc, object.Kind, name)
		} else {
			Cleanup(oc, object.Kind, name, "-n", object.Ns)
		}
	}
}
