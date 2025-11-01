// Package boxcutter provides a object reconciliation library based on Package Operator.
package boxcutter

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/ownerhandling"
	"pkg.package-operator.run/boxcutter/validation"
)

// Revision represents multiple phases at a given point in time.
type Revision = types.Revision

// Phase represents a collection of objects lifecycled together.
type Phase = types.Phase

// ObjectReconcileOption is the common interface for object reconciliation options.
type ObjectReconcileOption = types.ObjectReconcileOption

// ObjectTeardownOption holds configuration options changing object teardown.
type ObjectTeardownOption = types.ObjectTeardownOption

// PhaseReconcileOption is the common interface for phase reconciliation options.
type PhaseReconcileOption = types.PhaseReconcileOption

// PhaseTeardownOption holds configuration options changing phase teardown.
type PhaseTeardownOption = types.PhaseTeardownOption

// RevisionReconcileOption is the common interface for revision reconciliation options.
type RevisionReconcileOption = types.RevisionReconcileOption

// RevisionTeardownOption holds configuration options changing revision teardown.
type RevisionTeardownOption = types.RevisionTeardownOption

// WithPreviousOwners is a list of known objects allowed to take ownership from.
// Objects from this list will not trigger collision detection and prevention.
type WithPreviousOwners = types.WithPreviousOwners

const (
	// CollisionProtectionPrevent prevents owner collisions entirely
	// by not allowing to work with objects already present on the cluster.
	CollisionProtectionPrevent = types.CollisionProtectionPrevent
	// CollisionProtectionIfNoController allows to patch and override
	// objects already present if they are not owned by another controller.
	CollisionProtectionIfNoController = types.CollisionProtectionIfNoController
	// CollisionProtectionNone allows to patch and override objects already
	// present and owned by other controllers.
	//
	// Be careful!
	// This setting may cause multiple controllers to fight over a resource,
	// causing load on the Kubernetes API server and etcd.
	CollisionProtectionNone = types.CollisionProtectionNone
)

// WithCollisionProtection instructs the given CollisionProtection setting to be used.
type WithCollisionProtection = types.WithCollisionProtection

// WithPaused skips reconciliation and just reports status information.
// Can also be described as dry-run, as no modification will occur.
type WithPaused = types.WithPaused

// Prober needs to be implemented by any probing implementation.
type Prober = types.Prober

// ProbeFunc wraps the given function to work with the Prober interface.
var ProbeFunc = types.ProbeFunc

// WithProbe registers the given probe to evaluate state of objects.
var WithProbe = types.WithProbe

// WithObjectReconcileOptions applies the given options only to the given object.
var WithObjectReconcileOptions = types.WithObjectReconcileOptions

// WithObjectTeardownOptions applies the given options only to the given object.
var WithObjectTeardownOptions = types.WithObjectTeardownOptions

// WithPhaseReconcileOptions applies the given options only to the given Phase.
var WithPhaseReconcileOptions = types.WithPhaseReconcileOptions

// WithPhaseTeardownOptions applies the given options only to the given Phase.
var WithPhaseTeardownOptions = types.WithPhaseTeardownOptions

// ProgressProbeType is a well-known probe type used to guard phase progression.
const ProgressProbeType = types.ProgressProbeType

// RevisionEngine manages rollout and teardown of multiple phases.
type RevisionEngine = machinery.RevisionEngine

// OwnerStrategy interface needed for RevisionEngine.
type OwnerStrategy interface {
	SetControllerReference(owner, obj metav1.Object) error
	GetController(obj metav1.Object) (metav1.OwnerReference, bool)
	IsController(owner, obj metav1.Object) bool
	CopyOwnerReferences(objA, objB metav1.Object)
	ReleaseController(obj metav1.Object)
	RemoveOwner(owner, obj metav1.Object)
}

// RevisionEngineOptions holds all configuration options for the RevisionEngine.
type RevisionEngineOptions struct {
	Scheme          *runtime.Scheme
	FieldOwner      string
	SystemPrefix    string
	DiscoveryClient discovery.DiscoveryInterface
	RestMapper      meta.RESTMapper
	Writer          client.Writer
	Reader          client.Reader

	// Optional

	OwnerStrategy OwnerStrategy
}

// NewRevisionEngine returns a new RevisionEngine instance.
func NewRevisionEngine(opts RevisionEngineOptions) (*RevisionEngine, error) {
	if err := validateRevisionEngineOpts(opts); err != nil {
		return nil, err
	}

	if opts.OwnerStrategy == nil {
		opts.OwnerStrategy = ownerhandling.NewNative(opts.Scheme)
	}

	pval := validation.NewNamespacedPhaseValidator(opts.RestMapper, opts.Writer)
	rval := validation.NewRevisionValidator()

	comp := machinery.NewComparator(
		opts.OwnerStrategy, opts.DiscoveryClient, opts.Scheme, opts.FieldOwner)

	oe := machinery.NewObjectEngine(
		opts.Scheme, opts.Reader, opts.Writer,
		opts.OwnerStrategy, comp, opts.FieldOwner, opts.SystemPrefix,
	)
	pe := machinery.NewPhaseEngine(oe, pval)

	return machinery.NewRevisionEngine(pe, rval, opts.Writer), nil
}

// RevisionEngineOptionsError is returned for errors with the RevisionEngineOptions.
type RevisionEngineOptionsError struct {
	msg string
}

func (e RevisionEngineOptionsError) Error() string {
	return e.msg
}

func validateRevisionEngineOpts(opts RevisionEngineOptions) error {
	if opts.Scheme == nil {
		return RevisionEngineOptionsError{msg: "scheme must be provided"}
	}

	if len(opts.FieldOwner) == 0 {
		return RevisionEngineOptionsError{msg: "fieldOwner must be provided"}
	}

	if len(opts.SystemPrefix) == 0 {
		return RevisionEngineOptionsError{msg: "systemPrefix must be provided"}
	}

	if opts.DiscoveryClient == nil {
		return RevisionEngineOptionsError{msg: "discoveryClient must be provided"}
	}

	if opts.RestMapper == nil {
		return RevisionEngineOptionsError{msg: "restMapper must be provided"}
	}

	if opts.Writer == nil {
		return RevisionEngineOptionsError{msg: "writer must be provided"}
	}

	if opts.Reader == nil {
		return RevisionEngineOptionsError{msg: "reader must be provided"}
	}

	return nil
}
