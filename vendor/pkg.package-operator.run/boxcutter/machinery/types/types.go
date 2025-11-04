// Package types contains common type definitions for boxcutter machinery.
package types

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectRef holds information to identify an object.
type ObjectRef struct {
	schema.GroupVersionKind
	client.ObjectKey
}

// ToObjectRef returns an ObjectRef object from unstructured.
func ToObjectRef(obj client.Object) ObjectRef {
	return ObjectRef{
		GroupVersionKind: obj.GetObjectKind().GroupVersionKind(),
		ObjectKey:        client.ObjectKeyFromObject(obj),
	}
}

// String returns a string representation.
func (oid ObjectRef) String() string {
	return fmt.Sprintf("%s %s", oid.GroupVersionKind, oid.ObjectKey)
}

// Phase represents a named collection of objects.
type Phase struct {
	// Name of the Phase.
	Name string
	// Objects contained in the phase.
	Objects []unstructured.Unstructured
}

// GetName returns the name of the phase.
func (p *Phase) GetName() string {
	return p.Name
}

// GetObjects returns the objects contained in the phase.
func (p *Phase) GetObjects() []unstructured.Unstructured {
	return p.Objects
}

// Revision represents the version of a content collection consisting of phases.
type Revision struct {
	// Name of the Revision.
	Name string
	// Owner object will be added as OwnerReference
	// to all objects managed by this revision.
	Owner client.Object
	// Revision number.
	Revision int64
	// Ordered list of phases.
	Phases []Phase
}

// GetName returns the name of the revision.
func (r *Revision) GetName() string {
	return r.Name
}

// GetOwner returns the owning object.
func (r *Revision) GetOwner() client.Object {
	return r.Owner
}

// GetRevisionNumber returns the current revision number.
func (r *Revision) GetRevisionNumber() int64 {
	return r.Revision
}

// GetPhases returns the phases a revision is made up of.
func (r *Revision) GetPhases() []Phase {
	return r.Phases
}
