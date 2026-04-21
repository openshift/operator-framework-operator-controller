//go:build dev

// Package output provides human-readable formatting utilities for local development.
// The Formatter type provides color-coded progress indicators and summaries that are
// used by CleanResultWriter (defined in writer.go), which implements the OTE ResultWriter interface.
package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	et "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[0;31m"
	ColorGreen  = "\033[0;32m"
	ColorYellow = "\033[1;33m"
	ColorBlue   = "\033[0;34m"
	ColorCyan   = "\033[0;36m"
	ColorGray   = "\033[0;90m"
	ColorBold   = "\033[1m"
)

// Formatter provides human-readable output for test results
type Formatter struct {
	writer       io.Writer
	totalTests   int
	currentTest  int
	passedCount  int
	failedCount  int
	skippedCount int
	pendingCount int
	failedTests  []FailedTest
}

// FailedTest holds information about a failed test
type FailedTest struct {
	Name  string
	Error string
}

// NewFormatter creates a new formatter instance
func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{
		writer:      w,
		failedTests: make([]FailedTest, 0),
	}
}

// PrintHeader prints the test suite header
func (f *Formatter) PrintHeader() {
	fmt.Fprintf(f.writer, "%s%s════════════════════════════════════════════════════════%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintf(f.writer, "%s%s  OLMv1 Test Suite - Starting%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintf(f.writer, "%s%s════════════════════════════════════════════════════════%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintln(f.writer)
}

func (f *Formatter) SetTotalTests(total int) {
	f.totalTests = total
	if total > 0 {
		fmt.Fprintf(f.writer, "%sTotal tests: %d%s\n", ColorBold, total, ColorReset)
		fmt.Fprintf(f.writer, "%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", ColorGray, ColorReset)
		fmt.Fprintln(f.writer)
	}
}

func (f *Formatter) PrintTestStart(name string) {
	f.currentTest++
	if f.totalTests > 0 {
		fmt.Fprintf(f.writer, "%s[%d/%d]%s %s▶ Running:%s %s\n",
			ColorBold, f.currentTest, f.totalTests, ColorReset,
			ColorBlue, ColorReset, name)
	} else {
		fmt.Fprintf(f.writer, "%s▶ Running:%s %s\n", ColorBlue, ColorReset, name)
	}
}

func (f *Formatter) PrintTestResult(result *et.ExtensionTestResult) {
	duration := formatDuration(result.Duration)

	switch result.Result {
	case et.ResultPassed:
		f.passedCount++
		fmt.Fprintf(f.writer, "%s  ✓ PASSED%s [%s] %s(Total: ✓%d ✗%d)%s\n",
			ColorGreen, ColorReset, duration,
			ColorGray, f.passedCount, f.failedCount, ColorReset)
	case et.ResultFailed:
		f.failedCount++
		f.failedTests = append(f.failedTests, FailedTest{
			Name:  result.Name,
			Error: result.Error,
		})
		fmt.Fprintf(f.writer, "%s  ✗ FAILED%s [%s] %s(Total: ✓%d ✗%d)%s\n",
			ColorRed, ColorReset, duration,
			ColorGray, f.passedCount, f.failedCount, ColorReset)
	case et.ResultSkipped:
		f.skippedCount++
		fmt.Fprintf(f.writer, "%s  ⊘ SKIPPED%s [%s] %s(Total: ✓%d ✗%d ⊘%d)%s\n",
			ColorYellow, ColorReset, duration,
			ColorGray, f.passedCount, f.failedCount, f.skippedCount, ColorReset)
	default:
		f.failedCount++
		errorMsg := fmt.Sprintf("unexpected result type: %s", result.Result)
		if result.Error != "" {
			errorMsg = fmt.Sprintf("%s; %s", errorMsg, result.Error)
		}
		f.failedTests = append(f.failedTests, FailedTest{
			Name:  result.Name,
			Error: errorMsg,
		})
		fmt.Fprintf(f.writer, "%s  ✗ UNKNOWN/FAILED%s [%s] %s(Total: ✓%d ✗%d)%s\n",
			ColorRed, ColorReset, duration,
			ColorGray, f.passedCount, f.failedCount, ColorReset)
	}
	fmt.Fprintln(f.writer)
}

func (f *Formatter) PrintSummary() {
	fmt.Fprintln(f.writer)
	fmt.Fprintf(f.writer, "%s%s════════════════════════════════════════════════════════%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintf(f.writer, "%s%s  Final Summary%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintf(f.writer, "%s%s════════════════════════════════════════════════════════%s\n", ColorCyan, ColorBold, ColorReset)

	fmt.Fprintf(f.writer, "%s✓ Passed:  %d%s\n", ColorGreen, f.passedCount, ColorReset)
	fmt.Fprintf(f.writer, "%s✗ Failed:  %d%s\n", ColorRed, f.failedCount, ColorReset)
	fmt.Fprintf(f.writer, "%s⊘ Skipped: %d%s\n", ColorYellow, f.skippedCount, ColorReset)

	fmt.Fprintln(f.writer)

	if f.failedCount == 0 {
		fmt.Fprintf(f.writer, "%s%s✓ ALL TESTS PASSED!%s\n", ColorGreen, ColorBold, ColorReset)
	} else {
		fmt.Fprintf(f.writer, "%s%s✗ %d TEST(S) FAILED%s\n", ColorRed, ColorBold, f.failedCount, ColorReset)
		f.printFailedTestDetails()
	}
}

func (f *Formatter) printFailedTestDetails() {
	if len(f.failedTests) == 0 {
		return
	}

	fmt.Fprintln(f.writer)
	fmt.Fprintf(f.writer, "%s%s════════════════════════════════════════════════════════%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintf(f.writer, "%s%s  Failed Test Details%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintf(f.writer, "%s%s════════════════════════════════════════════════════════%s\n", ColorCyan, ColorBold, ColorReset)
	fmt.Fprintln(f.writer)

	for _, ft := range f.failedTests {
		fmt.Fprintf(f.writer, "%s%sTest: %s%s\n", ColorRed, ColorBold, ft.Name, ColorReset)
		if ft.Error != "" {
			// Clean up the error message
			errorMsg := cleanErrorMessage(ft.Error)
			fmt.Fprintf(f.writer, "%s%s%s\n", ColorGray, errorMsg, ColorReset)
		}
		fmt.Fprintln(f.writer)
	}
}

func (f *Formatter) HasFailures() bool {
	return f.failedCount > 0
}

func formatDuration(durationMs int64) string {
	d := time.Duration(durationMs) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%.3f seconds", d.Seconds())
	}
	return fmt.Sprintf("%.1f seconds", d.Seconds())
}

func cleanErrorMessage(msg string) string {
	lines := strings.Split(msg, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, "\n")
}
