package machinery

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateCollisionError is returned when boxcutter tries to create an object,
// but it already exists. \
// This happens when another actor has created the object and caches are slow,
// or the colliding object is excluded via cache selectors.
type CreateCollisionError struct {
	object client.Object
	msg    string
}

// Error implements golangs error interface.
func (e CreateCollisionError) Error() string {
	return fmt.Sprintf("%s: %s", e.object, e.msg)
}
