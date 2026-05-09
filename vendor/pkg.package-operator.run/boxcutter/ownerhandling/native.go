package ownerhandling

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

var _ ownerStrategy = (*OwnerStrategyNative)(nil)

// OwnerStrategyNative handling strategy uses .metadata.ownerReferences.
type OwnerStrategyNative struct {
	scheme *runtime.Scheme
}

// NewNative returns a new OwnerStrategyNative instance.
func NewNative(scheme *runtime.Scheme) *OwnerStrategyNative {
	return &OwnerStrategyNative{
		scheme: scheme,
	}
}

// GetController returns the OwnerReference with Controller==true, if one exist.
func (s *OwnerStrategyNative) GetController(obj metav1.Object) (
	metav1.OwnerReference, bool,
) {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			return ref, true
		}
	}

	return metav1.OwnerReference{}, false
}

// CopyOwnerReferences copies all OwnerReferences from objA to objB,
// overriding any existing OwnerReferences on objB.
func (s *OwnerStrategyNative) CopyOwnerReferences(objA, objB metav1.Object) {
	objB.SetOwnerReferences(objA.GetOwnerReferences())
}

// IsOwner returns true if owner is contained in object OwnerReference list.
func (s *OwnerStrategyNative) IsOwner(owner, obj metav1.Object) bool {
	ownerRefComp := s.ownerRefForCompare(owner)
	for _, ownerRef := range obj.GetOwnerReferences() {
		if s.referSameObject(ownerRefComp, ownerRef) {
			return true
		}
	}

	return false
}

// IsController returns true if the given owner is the controller of obj.
func (s *OwnerStrategyNative) IsController(
	owner, obj metav1.Object,
) bool {
	ownerRefComp := s.ownerRefForCompare(owner)
	for _, ownerRef := range obj.GetOwnerReferences() {
		if s.referSameObject(ownerRefComp, ownerRef) &&
			ownerRef.Controller != nil &&
			*ownerRef.Controller {
			return true
		}
	}

	return false
}

// RemoveOwner removes the owner from objs OwnerReference list.
func (s *OwnerStrategyNative) RemoveOwner(owner, obj metav1.Object) {
	ownerRefComp := s.ownerRefForCompare(owner)
	ownerRefs := obj.GetOwnerReferences()
	foundIndex := -1

	for i, ownerRef := range ownerRefs {
		if s.referSameObject(ownerRefComp, ownerRef) {
			foundIndex = i

			break
		}
	}

	if foundIndex != -1 {
		obj.SetOwnerReferences(remove(ownerRefs, foundIndex))
	}
}

// ReleaseController sets all OwnerReferences Controller to false.
func (s *OwnerStrategyNative) ReleaseController(obj metav1.Object) {
	ownerRefs := obj.GetOwnerReferences()
	for i := range ownerRefs {
		ownerRefs[i].Controller = ptr.To(false)
	}

	obj.SetOwnerReferences(ownerRefs)
}

// SetOwnerReference adds owner as OwnerReference to obj, with Controller set to false.
func (s *OwnerStrategyNative) SetOwnerReference(owner, obj metav1.Object) error {
	return controllerutil.SetOwnerReference(owner, obj, s.scheme)
}

// SetControllerReference adds owner as OwnerReference to obj, with Controller set to true.
func (s *OwnerStrategyNative) SetControllerReference(owner, obj metav1.Object) error {
	return controllerutil.SetControllerReference(owner, obj, s.scheme)
}

// EnqueueRequestForOwner returns a EventHandler to enqueue the owner.
func (s *OwnerStrategyNative) EnqueueRequestForOwner(
	ownerType client.Object, mapper meta.RESTMapper, isController bool,
) handler.EventHandler {
	if isController {
		return handler.EnqueueRequestForOwner(s.scheme, mapper, ownerType, handler.OnlyControllerOwner())
	}

	return handler.EnqueueRequestForOwner(s.scheme, mapper, ownerType)
}

func (s *OwnerStrategyNative) ownerRefForCompare(owner metav1.Object) metav1.OwnerReference {
	// Validate the owner.
	ro, ok := owner.(runtime.Object)
	if !ok {
		panic(fmt.Sprintf("%T is not a runtime.Object, cannot call SetOwnerReference", owner))
	}

	// Create a new owner ref.
	gvk, err := apiutil.GVKForObject(ro, s.scheme)
	if err != nil {
		panic(err)
	}

	ref := metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		UID:        owner.GetUID(),
		Name:       owner.GetName(),
	}

	return ref
}

// Returns true if a and b point to the same object.
func (s *OwnerStrategyNative) referSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		panic(err)
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		panic(err)
	}

	return aGV.Group == bGV.Group && a.Kind == b.Kind && a.Name == b.Name && a.UID == b.UID
}
