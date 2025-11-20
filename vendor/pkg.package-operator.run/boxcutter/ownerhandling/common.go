package ownerhandling

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type ownerStrategy interface {
	// SetOwnerReference adds owner as OwnerReference to obj, with Controller set to false.
	SetOwnerReference(owner, obj metav1.Object) error
	// SetControllerReference adds owner as OwnerReference to obj, with Controller set to true.
	SetControllerReference(owner, obj metav1.Object) error
	// GetController returns the OwnerReference with Controller==true, if one exist.
	GetController(obj metav1.Object) (metav1.OwnerReference, bool)
	// IsController returns true if the given owner is the controller of obj.
	IsController(owner, obj metav1.Object) bool
	// CopyOwnerReferences copies all OwnerReferences from objA to objB,
	// overriding any existing OwnerReferences on objB.
	CopyOwnerReferences(objA, objB metav1.Object)
	// EnqueueRequestForOwner returns a EventHandler to enqueue the owner.
	EnqueueRequestForOwner(ownerType client.Object, mapper meta.RESTMapper, isController bool) handler.EventHandler
	// ReleaseController sets all OwnerReferences Controller to false.
	ReleaseController(obj metav1.Object)
	// RemoveOwner removes the owner from objs OwnerReference list.
	RemoveOwner(owner, obj metav1.Object)
	// IsOwner returns true if owner is contained in object OwnerReference list.
	IsOwner(owner, obj metav1.Object) bool
}

// Removes the given index from the slice.
// does not perform an out-of-bounds check.
func remove[T any](s []T, i int) []T {
	s[i] = s[len(s)-1]

	return s[:len(s)-1]
}
