/*
This command is used to run the OLMv1 tests extension for OpenShift.
It registers the OLMv1 tests with the OpenShift Tests Extension framework
and provides a command-line interface to execute them.

For further information, please refer to the documentation at:
https://github.com/openshift-eng/openshift-tests-extension/blob/main/cmd/example-tests/main.go
*/
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	e "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	et "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
	g "github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"
	"github.com/spf13/cobra"

	// The import below is necessary to ensure that the OLMv1 tests are registered with the extension.
	_ "github/operator-framework-operator-controller/openshift/tests-extension/test"
	"github/operator-framework-operator-controller/openshift/tests-extension/test/env"
)

func main() {
	registry := e.NewRegistry()
	ext := e.NewExtension("openshift", "payload", "olmv1")

	// Register the OLMv1 test suites with OpenShift Tests Extension.
	// These suites determine how test cases are grouped and executed in various jobs.
	//
	// Definitions of labels:
	// - [Serial]: test must run in isolation, one at a time. Typically used for disruptive cases (e.g., kill nodes).
	// - [Slow]: test takes a long time to execute (i.e. >5 min.). Cannot be included in fast/parallel suites.
	//
	// IMPORTANT:
	// Even though a suite is marked "parallel", all tests run serially when using the *external binary*
	// (`run-suite`, `run-test`) because it executes within a single process and Ginkgo
	// cannot parallelize within a single process.
	// See: https://github.com/openshift-eng/openshift-tests-extension/blob/main/pkg/ginkgo/util.go#L50
	//
	// For actual parallel test execution (e.g., in CI), use `openshift-tests`, which launches one process per test:
	// https://github.com/openshift/origin/blob/main/pkg/test/ginkgo/test_runner.go#L294

	// Suite: olmv1/parallel
	// ---------------------------------------------------------------
	// Contains fast, parallel-safe test cases only.
	// Excludes any tests labeled [Serial] or [Slow].
	// Note: Tests with [Serial] and [Slow] cannot run with openshift/conformance/parallel.
	ext.AddSuite(e.Suite{
		Name:    "olmv1/parallel",
		Parents: []string{"openshift/conformance/parallel"},
		Qualifiers: []string{
			`!(name.contains("[Serial]") || name.contains("[Slow]"))`,
		},
	})

	// Suite: olmv1/serial
	// ---------------------------------------------------------------
	// Contains tests explicitly labeled [Serial].
	// These tests are typically disruptive and must run one at a time.
	ext.AddSuite(e.Suite{
		Name:    "olmv1/serial",
		Parents: []string{"openshift/conformance/serial"},
		Qualifiers: []string{
			`name.contains("[Serial]")`,
		},
	})

	// Suite: olmv1/slow
	// 	// ---------------------------------------------------------------
	// Contains tests labeled [Slow], which take significant time to run.
	// These are not allowed in fast/parallel suites, and should run in optional/slow jobs.
	ext.AddSuite(e.Suite{
		Name:    "olmv1/slow",
		Parents: []string{"openshift/optional/slow"},
		Qualifiers: []string{
			`name.contains("[Slow]")`,
		},
	})

	// Suite: olmv1/all
	// ---------------------------------------------------------------
	// All tests in one suite: includes [Serial], [Slow], [Disruptive], etc.
	ext.AddSuite(e.Suite{
		Name: "olmv1/all",
	})

	specs, err := g.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite()
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ginkgo: %+v", err.Error()))
	}

	// Ensure `[Disruptive]` tests are always also marked `[Serial]`.
	// This prevents them from running in parallel suites, which could cause flaky failures
	// due to disruptive behavior.
	specs = specs.Walk(func(spec *et.ExtensionTestSpec) {
		if strings.Contains(spec.Name, "[Disruptive]") && !strings.Contains(spec.Name, "[Serial]") {
			spec.Name = strings.ReplaceAll(
				spec.Name,
				"[Disruptive]",
				"[Serial][Disruptive]",
			)
		}
	})

	// To handle renames and preserve test ID by setting the original-name.
	// This logic looks for a custom Ginkgo label in the format:
	//   Label("original-name:<full old test name>")
	// When found, it sets spec.OriginalName = <old name>.
	// **Example**
	// It("should pass a renamed sanity check",
	//		Label("original-name:[sig-olmv1] OLMv1 should pass a trivial sanity check"),
	//  	func(ctx context.Context) {
	//  		Expect(len("test")).To(BeNumerically(">", 0))
	// 	    })
	specs = specs.Walk(func(spec *et.ExtensionTestSpec) {
		for label := range spec.Labels {
			if strings.HasPrefix(label, "original-name:") {
				parts := strings.SplitN(label, "original-name:", 2)
				if len(parts) > 1 {
					spec.OriginalName = parts[1]
				}
			}
		}
	})

	// To delete tests you must mark them as obsolete.
	// These tests will be excluded from metadata validation during OTE update.
	// 1 - To get the full name of the test you want to remove run: make list-test-names
	// 2 - Add the test name here to avoid validation errors
	// 3 - Remove the test in your test file.
	// 4 - Run make build-update
	ext.IgnoreObsoleteTests(
	// "[sig-olmv1] OLMv1 should pass a trivial sanity check",
	// Add more removed test names below
	)

	// Initialize the environment before running any tests.
	specs.AddBeforeAll(func() {
		env.Init()
	})

	ext.AddSpecs(specs)
	registry.Register(ext)

	root := &cobra.Command{
		Long: "OLMv1 Tests Extension",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	if err := func() error {
		return root.Execute()
	}(); err != nil {
		os.Exit(1)
	}
}
