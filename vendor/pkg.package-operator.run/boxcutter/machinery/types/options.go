package types

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RevisionReconcileOptions holds configuration options changing revision reconciliation.
type RevisionReconcileOptions struct {
	// DefaultObjectOptions applying to all phases in the revision.
	DefaultPhaseOptions []PhaseReconcileOption
	// PhaseOptions maps PhaseOptions for specific phases.
	PhaseOptions map[string][]PhaseReconcileOption
}

// ForPhase returns the options for a given phase.
func (rropts RevisionReconcileOptions) ForPhase(phaseName string) []PhaseReconcileOption {
	opts := make([]PhaseReconcileOption, 0, len(rropts.DefaultPhaseOptions)+len(rropts.PhaseOptions[phaseName]))
	opts = append(opts, rropts.DefaultPhaseOptions...)
	opts = append(opts, rropts.PhaseOptions[phaseName]...)

	return opts
}

// RevisionReconcileOption is the common interface for revision reconciliation options.
type RevisionReconcileOption interface {
	ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions)
}

// RevisionTeardownOptions holds configuration options changing revision teardown.
type RevisionTeardownOptions struct {
	// DefaultObjectOptions applying to all phases in the revision.
	DefaultPhaseOptions []PhaseTeardownOption
	// PhaseOptions maps PhaseOptions for specific phases.
	PhaseOptions map[string][]PhaseTeardownOption
}

// ForPhase returns the options for a given phase.
func (rtopts RevisionTeardownOptions) ForPhase(phaseName string) []PhaseTeardownOption {
	opts := make([]PhaseTeardownOption, 0, len(rtopts.DefaultPhaseOptions)+len(rtopts.PhaseOptions[phaseName]))
	opts = append(opts, rtopts.DefaultPhaseOptions...)
	opts = append(opts, rtopts.PhaseOptions[phaseName]...)

	return opts
}

// RevisionTeardownOption is the common interface for revision teardown options.
type RevisionTeardownOption interface {
	ApplyToRevisionTeardownOptions(opts *RevisionTeardownOptions)
}

// PhaseReconcileOptions holds configuration options changing phase reconciliation.
type PhaseReconcileOptions struct {
	// DefaultObjectOptions applying to all objects in the phase.
	DefaultObjectOptions []ObjectReconcileOption
	// ObjectOptions maps ObjectOptions for specific objects.
	ObjectOptions map[ObjectRef][]ObjectReconcileOption
}

// ForObject returns the options for the given object.
func (pro PhaseReconcileOptions) ForObject(obj client.Object) []ObjectReconcileOption {
	objRef := ToObjectRef(obj)

	opts := make([]ObjectReconcileOption, 0, len(pro.DefaultObjectOptions)+len(pro.ObjectOptions[objRef]))
	opts = append(opts, pro.DefaultObjectOptions...)
	opts = append(opts, pro.ObjectOptions[objRef]...)

	return opts
}

// PhaseReconcileOption is the common interface for phase reconciliation options.
type PhaseReconcileOption interface {
	ApplyToPhaseReconcileOptions(opts *PhaseReconcileOptions)
	RevisionReconcileOption
}

// PhaseTeardownOptions holds configuration options changing phase teardown.
type PhaseTeardownOptions struct {
	// DefaultObjectOptions applying to all objects in the phase.
	DefaultObjectOptions []ObjectTeardownOption
	// ObjectOptions maps ObjectOptions for specific objects.
	ObjectOptions map[ObjectRef][]ObjectTeardownOption
}

// ForObject returns the options for the given object.
func (pto PhaseTeardownOptions) ForObject(obj client.Object) []ObjectTeardownOption {
	objRef := ToObjectRef(obj)

	opts := make([]ObjectTeardownOption, 0, len(pto.DefaultObjectOptions)+len(pto.ObjectOptions[objRef]))
	opts = append(opts, pto.DefaultObjectOptions...)
	opts = append(opts, pto.ObjectOptions[objRef]...)

	return opts
}

// PhaseTeardownOption is the common interface for phase teardown options.
type PhaseTeardownOption interface {
	ApplyToPhaseTeardownOptions(opts *PhaseTeardownOptions)
	RevisionTeardownOption
}

// ObjectReconcileOptions holds configuration options changing object reconciliation.
type ObjectReconcileOptions struct {
	CollisionProtection CollisionProtection
	PreviousOwners      []client.Object
	Paused              bool
	Probes              map[string]Prober
}

// Default sets empty Option fields to their default value.
func (opts *ObjectReconcileOptions) Default() {
	if len(opts.CollisionProtection) == 0 {
		opts.CollisionProtection = CollisionProtectionPrevent
	}
}

// ObjectReconcileOption is the common interface for object reconciliation options.
type ObjectReconcileOption interface {
	ApplyToObjectReconcileOptions(opts *ObjectReconcileOptions)
	PhaseReconcileOption
}

var (
	_ ObjectReconcileOption = (WithCollisionProtection)("")
	_ ObjectReconcileOption = (WithPaused{})
	_ ObjectReconcileOption = (WithPreviousOwners{})
	_ ObjectReconcileOption = (WithProbe("", nil))
)

// ObjectTeardownOptions holds configuration options changing object teardown.
type ObjectTeardownOptions struct{}

// Default sets empty Option fields to their default value.
func (opts *ObjectTeardownOptions) Default() {}

// ObjectTeardownOption is the common interface for object teardown options.
type ObjectTeardownOption interface {
	ApplyToObjectTeardownOptions(opts *ObjectTeardownOptions)
	PhaseTeardownOption
}

// CollisionProtection specifies how collision with existing objects and
// other controllers should be handled.
type CollisionProtection string

const (
	// CollisionProtectionPrevent prevents owner collisions entirely
	// by not allowing to work with objects already present on the cluster.
	CollisionProtectionPrevent CollisionProtection = "Prevent"
	// CollisionProtectionIfNoController allows to patch and override
	// objects already present if they are not owned by another controller.
	CollisionProtectionIfNoController CollisionProtection = "IfNoController"
	// CollisionProtectionNone allows to patch and override objects already
	// present and owned by other controllers.
	//
	// Be careful!
	// This setting may cause multiple controllers to fight over a resource,
	// causing load on the Kubernetes API server and etcd.
	CollisionProtectionNone CollisionProtection = "None"
)

// WithCollisionProtection instructs the given CollisionProtection setting to be used.
type WithCollisionProtection CollisionProtection

// ApplyToObjectReconcileOptions implements ObjectReconcileOption.
func (p WithCollisionProtection) ApplyToObjectReconcileOptions(opts *ObjectReconcileOptions) {
	opts.CollisionProtection = CollisionProtection(p)
}

// ApplyToPhaseReconcileOptions implements PhaseOption.
func (p WithCollisionProtection) ApplyToPhaseReconcileOptions(opts *PhaseReconcileOptions) {
	opts.DefaultObjectOptions = append(opts.DefaultObjectOptions, p)
}

// ApplyToRevisionReconcileOptions implements RevisionReconcileOptions.
func (p WithCollisionProtection) ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions) {
	opts.DefaultPhaseOptions = append(opts.DefaultPhaseOptions, p)
}

// WithPreviousOwners is a list of known objects allowed to take ownership from.
// Objects from this list will not trigger collision detection and prevention.
type WithPreviousOwners []client.Object

// ApplyToObjectReconcileOptions implements ObjectReconcileOption.
func (p WithPreviousOwners) ApplyToObjectReconcileOptions(opts *ObjectReconcileOptions) {
	opts.PreviousOwners = p
}

// ApplyToPhaseReconcileOptions implements PhaseOption.
func (p WithPreviousOwners) ApplyToPhaseReconcileOptions(opts *PhaseReconcileOptions) {
	opts.DefaultObjectOptions = append(opts.DefaultObjectOptions, p)
}

// ApplyToRevisionReconcileOptions implements RevisionReconcileOptions.
func (p WithPreviousOwners) ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions) {
	opts.DefaultPhaseOptions = append(opts.DefaultPhaseOptions, p)
}

// WithPaused skips reconciliation and just reports status information.
// Can also be described as dry-run, as no modification will occur.
type WithPaused struct{}

// ApplyToObjectReconcileOptions implements ObjectReconcileOption.
func (p WithPaused) ApplyToObjectReconcileOptions(opts *ObjectReconcileOptions) {
	opts.Paused = true
}

// ApplyToPhaseReconcileOptions implements PhaseOption.
func (p WithPaused) ApplyToPhaseReconcileOptions(opts *PhaseReconcileOptions) {
	opts.DefaultObjectOptions = append(opts.DefaultObjectOptions, p)
}

// ApplyToRevisionReconcileOptions implements RevisionReconcileOptions.
func (p WithPaused) ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions) {
	opts.DefaultPhaseOptions = append(opts.DefaultPhaseOptions, p)
}

// ProgressProbeType is a well-known probe type used to guard phase progression.
const ProgressProbeType = "Progress"

// Prober needs to be implemented by any probing implementation.
type Prober interface {
	Probe(obj client.Object) (success bool, messages []string)
}

// ProbeFunc wraps the given function to work with the Prober interface.
func ProbeFunc(fn func(obj client.Object) (success bool, messages []string)) Prober {
	return &probeFn{Fn: fn}
}

type probeFn struct {
	Fn func(obj client.Object) (success bool, messages []string)
}

func (p *probeFn) Probe(obj client.Object) (success bool, messages []string) {
	return p.Fn(obj)
}

// WithProbe registers the given probe to evaluate state of objects.
func WithProbe(t string, probe Prober) ObjectReconcileOption {
	return &optionFn{
		fn: func(opts *ObjectReconcileOptions) {
			if opts.Probes == nil {
				opts.Probes = map[string]Prober{}
			}
			opts.Probes[t] = probe
		},
	}
}

type withObjectReconcileOptions struct {
	obj  ObjectRef
	opts []ObjectReconcileOption
}

// WithObjectReconcileOptions applies the given options only to the given object.
func WithObjectReconcileOptions(obj client.Object, opts ...ObjectReconcileOption) PhaseReconcileOption {
	return &withObjectReconcileOptions{
		obj:  ToObjectRef(obj),
		opts: opts,
	}
}

// ApplyToPhaseReconcileOptions implements PhaseOption.
func (p *withObjectReconcileOptions) ApplyToPhaseReconcileOptions(opts *PhaseReconcileOptions) {
	if opts.ObjectOptions == nil {
		opts.ObjectOptions = map[ObjectRef][]ObjectReconcileOption{}
	}

	opts.ObjectOptions[p.obj] = p.opts
}

// ApplyToRevisionReconcileOptions implements RevisionReconcileOptions.
func (p *withObjectReconcileOptions) ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions) {
	opts.DefaultPhaseOptions = append(opts.DefaultPhaseOptions, p)
}

type withObjectTeardownOptions struct {
	obj  ObjectRef
	opts []ObjectTeardownOption
}

// WithObjectTeardownOptions applies the given options only to the given object.
func WithObjectTeardownOptions(obj client.Object, opts ...ObjectTeardownOption) PhaseTeardownOption {
	return &withObjectTeardownOptions{
		obj:  ToObjectRef(obj),
		opts: opts,
	}
}

// ApplyToPhaseTeardownOptions implements PhaseOption.
func (p *withObjectTeardownOptions) ApplyToPhaseTeardownOptions(opts *PhaseTeardownOptions) {
	if opts.ObjectOptions == nil {
		opts.ObjectOptions = map[ObjectRef][]ObjectTeardownOption{}
	}

	opts.ObjectOptions[p.obj] = p.opts
}

// ApplyToRevisionTeardownOptions implements RevisionTeardownOptions.
func (p *withObjectTeardownOptions) ApplyToRevisionTeardownOptions(opts *RevisionTeardownOptions) {
	opts.DefaultPhaseOptions = append(opts.DefaultPhaseOptions, p)
}

type withPhaseReconcileOptions struct {
	phaseName string
	opts      []PhaseReconcileOption
}

// WithPhaseReconcileOptions applies the given options only to the given phase.
func WithPhaseReconcileOptions(phaseName string, opts ...PhaseReconcileOption) RevisionReconcileOption {
	return &withPhaseReconcileOptions{
		phaseName: phaseName,
		opts:      opts,
	}
}

// ApplyToRevisionReconcileOptions implements RevisionReconcileOptions.
func (p *withPhaseReconcileOptions) ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions) {
	if opts.PhaseOptions == nil {
		opts.PhaseOptions = map[string][]PhaseReconcileOption{}
	}

	opts.PhaseOptions[p.phaseName] = p.opts
}

type withPhaseTeardownOptions struct {
	phaseName string
	opts      []PhaseTeardownOption
}

// WithPhaseTeardownOptions applies the given options only to the given phase.
func WithPhaseTeardownOptions(phaseName string, opts ...PhaseTeardownOption) RevisionTeardownOption {
	return &withPhaseTeardownOptions{
		phaseName: phaseName,
		opts:      opts,
	}
}

// ApplyToRevisionTeardownOptions implements RevisionTeardownOptions.
func (p *withPhaseTeardownOptions) ApplyToRevisionTeardownOptions(opts *RevisionTeardownOptions) {
	if opts.PhaseOptions == nil {
		opts.PhaseOptions = map[string][]PhaseTeardownOption{}
	}

	opts.PhaseOptions[p.phaseName] = p.opts
}

type optionFn struct {
	fn func(opts *ObjectReconcileOptions)
}

// ApplyToObjectReconcileOptions implements ObjectReconcileOption.
func (p *optionFn) ApplyToObjectReconcileOptions(opts *ObjectReconcileOptions) {
	p.fn(opts)
}

// ApplyToPhaseReconcileOptions implements PhaseOption.
func (p *optionFn) ApplyToPhaseReconcileOptions(opts *PhaseReconcileOptions) {
	opts.DefaultObjectOptions = append(opts.DefaultObjectOptions, p)
}

// ApplyToRevisionReconcileOptions implements RevisionReconcileOptions.
func (p *optionFn) ApplyToRevisionReconcileOptions(opts *RevisionReconcileOptions) {
	opts.DefaultPhaseOptions = append(opts.DefaultPhaseOptions, p)
}
