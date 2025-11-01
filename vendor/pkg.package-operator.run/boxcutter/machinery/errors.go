package machinery

// CreateCollisionError is returned when boxcutter tries to create an object,
// but it already exists. \
// This happens when another actor has created the object and caches are slow,
// or the colliding object is excluded via cache selectors.
type CreateCollisionError struct {
	msg string
}

// Error implements golangs error interface.
func (e CreateCollisionError) Error() string {
	return e.msg
}
