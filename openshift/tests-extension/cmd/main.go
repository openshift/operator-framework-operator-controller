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

	// TODO: Init test framework for cluster-aware cases
	// --------------------------------------------------
	// The external binary doesn't currently init the test framework (e.g., kubeconfig, REST client).
	// That's fine for now since our tests don't access the cluster.
	// However, any test that does will fail when run with this binary.
	//
	// openshift-tests handles this via:
	// - SuiteWithKubeTestInitializationPreSuite()
	//   -> calls DecodeProvider() and InitializeTestFramework()
	//
	// We'll need to add similar logic when we start add the tests here.
	//
	// References:
	// - https://github.com/openshift/origin/blob/main/pkg/cmd/openshift-tests/run/flags.go#L53
	// - https://github.com/openshift/origin/blob/main/pkg/clioptions/clusterdiscovery/provider.go#L100

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
