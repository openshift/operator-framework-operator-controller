package test

import (
	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/ginkgo/v2"

	"github/operator-framework-operator-controller/openshift/tests-extension/test/env"
)

// -----------------------------------------------------------------------------
// Global test setup (works for both single and parallel test runs)
// -----------------------------------------------------------------------------
//
// Ginkgo can run tests in parallel using multiple processes, called "nodes".
// Each node runs part of the test suite in its own separate process.
//
// This will run the tests using 4 separate processes (nodes).
// Each node has it owns env setup before running tests to avoid issues with shared state.
//
// `SynchronizedBeforeSuite` helps us do two things:
//   - Run some setup code only once (on Node 1)
//   - Run other setup code on *all* nodes (including Node 1)
//
// We use it here to initialize the test environment for our cluster.
//
// Why this is needed:
//   - Each parallel node needs to create its own test client
//   - `env.Get()` sets up the config and test client
//   - By calling `env.Get()` in both parts of SynchronizedBeforeSuite,
//     we make sure every node has a working client
var _ = SynchronizedBeforeSuite(func() []byte {
	// Runs only on Node 1
	_ = env.Get()
	return nil
}, func([]byte) {
	// Runs on *every* node (including Node 1)
	_ = env.Get()
})
