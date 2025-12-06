package machinery

import (
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"pkg.package-operator.run/boxcutter/machinery/types"
)

// ObjectResult is the common Result interface for multiple result types.
type ObjectResult interface {
	// Action taken by the reconcile engine.
	Action() Action
	// Object as last seen on the cluster after creation/update.
	Object() Object
	// Success returns true when the operation is considered successful.
	// Operations are considered a success, when the object reflects desired state,
	// is owned by the right controller and passes the given probe.
	Success() bool
	// Probes returns the results from the given object Probes.
	Probes() map[string]ObjectProbeResult
	// String returns a human readable description of the Result.
	String() string
}

// ObjectProbeResult records probe results for the object.
type ObjectProbeResult struct {
	Success  bool
	Messages []string
}

var (
	_ ObjectResult = (*ObjectResultCreated)(nil)
	_ ObjectResult = (*ObjectResultUpdated)(nil)
	_ ObjectResult = (*ObjectResultIdle)(nil)
	_ ObjectResult = (*ObjectResultProgressed)(nil)
	_ ObjectResult = (*ObjectResultRecovered)(nil)
	_ ObjectResult = (*ObjectResultCollision)(nil)
)

// ObjectResultCreated is returned when the Object was just created.
type ObjectResultCreated struct {
	obj          Object
	probeResults map[string]ObjectProbeResult
}

func newObjectResultCreated(
	obj Object,
	probes map[string]types.Prober,
) ObjectResult {
	return ObjectResultCreated{
		obj:          obj,
		probeResults: runProbes(obj, probes),
	}
}

// Action taken by the reconcile engine.
func (r ObjectResultCreated) Action() Action {
	return ActionCreated
}

// Object as last seen on the cluster after creation/update.
func (r ObjectResultCreated) Object() Object {
	return r.obj
}

// Success returns true when the operation is considered successful.
// Operations are considered a success, when the object reflects desired state,
// is owned by the right controller and passes the given probe.
func (r ObjectResultCreated) Success() bool {
	for _, res := range r.probeResults {
		if !res.Success {
			return false
		}
	}

	return true
}

// Probes returns the results from the given object Probe.
func (r ObjectResultCreated) Probes() map[string]ObjectProbeResult {
	return r.probeResults
}

// String returns a human readable description of the Result.
func (r ObjectResultCreated) String() string {
	return reportStart(r)
}

// ObjectResultUpdated is returned when the object is updated.
type ObjectResultUpdated struct {
	normalResult
}

func newObjectResultUpdated(
	obj Object,
	diverged CompareResult,
	probes map[string]types.Prober,
) ObjectResult {
	return ObjectResultUpdated{
		normalResult: newNormalObjectResult(ActionUpdated, obj, diverged, probes),
	}
}

// ObjectResultProgressed is returned when the object has been progressed to a newer revision.
type ObjectResultProgressed struct {
	normalResult
}

func newObjectResultProgressed(
	obj Object,
	diverged CompareResult,
	probes map[string]types.Prober,
) ObjectResult {
	return ObjectResultProgressed{
		normalResult: newNormalObjectResult(ActionProgressed, obj, diverged, probes),
	}
}

// ObjectResultIdle is returned when nothing was done.
type ObjectResultIdle struct {
	normalResult
}

func newObjectResultIdle(
	obj Object,
	diverged CompareResult,
	probes map[string]types.Prober,
) ObjectResult {
	return ObjectResultIdle{
		normalResult: newNormalObjectResult(ActionIdle, obj, diverged, probes),
	}
}

// ObjectResultRecovered is returned when the object had to be reset after conflicting with another actor.
type ObjectResultRecovered struct {
	normalResult
}

func newObjectResultRecovered(
	obj Object,
	diverged CompareResult,
	probes map[string]types.Prober,
) ObjectResult {
	return ObjectResultRecovered{
		normalResult: newNormalObjectResult(ActionRecovered, obj, diverged, probes),
	}
}

type normalResult struct {
	action        Action
	obj           Object
	probeResults  map[string]ObjectProbeResult
	compareResult CompareResult
}

func newNormalObjectResult(
	action Action,
	obj Object,
	compResult CompareResult,
	probes map[string]types.Prober,
) normalResult {
	return normalResult{
		obj:           obj,
		action:        action,
		probeResults:  runProbes(obj, probes),
		compareResult: compResult,
	}
}

// Action taken by the reconcile engine.
func (r normalResult) Action() Action {
	return r.action
}

// Object as last seen on the cluster after creation/update.
func (r normalResult) Object() Object {
	return r.obj
}

// CompareResult returns the results from checking the
// actual object on the cluster against the desired spec.
// Contains informations about differences that had to be reconciled.
func (r normalResult) CompareResult() CompareResult {
	return r.compareResult
}

// Probe returns the results from the given object Probe.
func (r normalResult) Probes() map[string]ObjectProbeResult {
	return r.probeResults
}

// Success returns true when the operation is considered successful.
// Operations are considered a success, when the object reflects desired state,
// is owned by the right controller and passes the given probe.
func (r normalResult) Success() bool {
	for _, res := range r.probeResults {
		if !res.Success {
			return false
		}
	}

	return true
}

// String returns a human readable description of the Result.
func (r normalResult) String() string {
	msg := reportStart(r)

	return msg + r.compareResult.String()
}

// ObjectResultCollision is returned when conflicting with an existing object.
type ObjectResultCollision struct {
	normalResult
	// conflictingOwner is provided when Refusing due to Collision.
	conflictingOwner *metav1.OwnerReference
}

// ConflictingOwner Conflicting owner if Action == RefusingConflict.
func (r ObjectResultCollision) ConflictingOwner() (*metav1.OwnerReference, bool) {
	return r.conflictingOwner, r.conflictingOwner != nil
}

// Success returns true when the operation is considered successful.
// Operations are considered a success, when the object reflects desired state,
// is owned by the right controller and passes the given probe.
func (r ObjectResultCollision) Success() bool {
	return false
}

// String returns a human readable description of the Result.
func (r ObjectResultCollision) String() string {
	msg := r.normalResult.String()
	msg += fmt.Sprintf("Conflicting Owner: %s\n", r.conflictingOwner.String())

	return msg
}

func newObjectResultConflict(
	obj Object,
	diverged CompareResult,
	conflictingOwner *metav1.OwnerReference,
	probes map[string]types.Prober,
) ObjectResult {
	return ObjectResultCollision{
		normalResult: newNormalObjectResult(
			ActionCollision,
			obj, diverged, probes,
		),
		conflictingOwner: conflictingOwner,
	}
}

// Action describes the taken reconciliation action.
type Action string

const (
	// ActionCreated indicates that the object has been created to restore desired state.
	ActionCreated Action = "Created"
	// ActionUpdated indicates that the object has been updated to action on a change in desired state.
	ActionUpdated Action = "Updated"
	// ActionRecovered indicates that the object has been updated to recover values to
	// reflect desired state after interference from another actor of the system.
	ActionRecovered Action = "Recovered"
	// ActionProgressed indicates that the object progressed to newer revision.
	ActionProgressed Action = "Progressed"
	// ActionIdle indicates that no action was necessary. -> NoOp.
	ActionIdle Action = "Idle"
	// ActionCollision indicates aking actions was refused due to a collision with an existing object.
	ActionCollision Action = "Collision"
)

func reportStart(or ObjectResult) string {
	obj := or.Object()
	if err := ensureGVKIsSet(obj, scheme.Scheme); err != nil {
		panic(err)
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	msg := fmt.Sprintf(
		"Object %s.%s %s/%s\n"+
			`Action: %q`+"\n",
		gvk.Kind, gvk.GroupVersion().String(),
		obj.GetNamespace(), obj.GetName(),
		or.Action(),
	)

	probes := or.Probes()
	probeTypes := make([]string, 0, len(probes))

	for k := range probes {
		probeTypes = append(probeTypes, k)
	}

	sort.Strings(probeTypes)

	if len(probeTypes) > 0 {
		msg += "Probes:\n"
	}

	for _, probeType := range probeTypes {
		probeRes := probes[probeType]
		if probeRes.Success {
			msg += fmt.Sprintf("- %s: Succeeded\n", probeType)
		} else {
			msg += fmt.Sprintf("- %s: Failed\n", probeType)
			for _, m := range probeRes.Messages {
				msg += "  - " + m + "\n"
			}
		}
	}

	return msg
}

func runProbes(obj Object, probes map[string]types.Prober) map[string]ObjectProbeResult {
	results := map[string]ObjectProbeResult{}

	for t, probe := range probes {
		s, msgs := probe.Probe(obj)
		results[t] = ObjectProbeResult{
			Success:  s,
			Messages: msgs,
		}
	}

	return results
}
