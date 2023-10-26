// Package extlogs provides helper functions for printing warnings
// during Ginkgo test runs.
//
// Why we use GinkgoWriter:
// -------------------------
// In Ginkgo, using fmt.Println() or log.Print() may not show messages.
// That's why we use fmt.Fprintf(ginkgo.GinkgoWriter, ...) instead of fmt.Println().
package extlogs

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
)

// Warn prints a simple warning message to the Ginkgo output.
// Example: Warn("something might be wrong")
func Warn(msg string) {
	fmt.Fprintf(ginkgo.GinkgoWriter, "[WARN] %s", msg)
}

// WarnError prints an error as a warning, if the error is not nil.
// Example: WarnError(err) will print "[WARN] error: ..." if err != nil
func WarnError(err error) {
	if err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "[WARN] error: %v", err)
	}
}

// WarnContext prints a warning with context + error, if the error is not nil.
// Example: WarnContext("failed to load config", err)
func WarnContext(context string, err error) {
	if err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "[WARN] %s: %v", context, err)
	}
}

// WarnContextf prints a formatted warning message with context.
// This is like printf + context-aware logging.
// Example: WarnContextf("unexpected value: %d", value)
func WarnContextf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(ginkgo.GinkgoWriter, "[WARN] %s", msg)
}

// Info prints an informational message to the Ginkgo output.
// Example: Info("starting test suite")
func Info(msg string) {
	fmt.Fprintf(ginkgo.GinkgoWriter, "[INFO] %s", msg)
}

// Infof prints a formatted informational message (like fmt.Printf).
// Example: Infof("using config: %s", path)
func Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(ginkgo.GinkgoWriter, "[INFO] %s", msg)
}
