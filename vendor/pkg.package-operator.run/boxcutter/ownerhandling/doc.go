// Package ownerhandling provides an interface and strategies to handle OwnerReferences.
// Native Kubernetes OwnerReferences should be used where possible,
// but in cross-namespace or cross-cluster scenarios,
// controllers have to fall back to use custom annotations.
package ownerhandling
