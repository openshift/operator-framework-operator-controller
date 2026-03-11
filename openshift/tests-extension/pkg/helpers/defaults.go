package helpers

import "time"

// Default settings for tests that use Eventually.
//
// The timeout is long enough so slow CI systems don’t cause false failures.
// This helps prevent sending wrong signals to Sippy and stops blocking
// pull requests from merging in the whole OCP org.
//
// The polling interval controls how often we check again.
// It’s set to a reasonable value to avoid too many API calls.
const (
	// DefaultTimeout is how long we wait before giving up on an Eventually check.
	DefaultTimeout = 5 * time.Minute

	// InstallTimeout is how long we wait for an operator to be fully installed.
	// With the BoxcutterRuntime, Installed=True is only set after all availability
	// probes pass (Deployments available, CRDs established, etc.), which can take
	// longer than DefaultTimeout on resource-constrained or non-standard clusters.
	InstallTimeout = 10 * time.Minute

	// DefaultPolling is how often we check again during an Eventually test.
	DefaultPolling = 3 * time.Second
)
