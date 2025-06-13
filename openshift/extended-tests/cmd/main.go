package main

import (
	"fmt"
	"os"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	e "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	et "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
	g "github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"
	// If using ginkgo, import your tests here
	_ "github.com/openshift-eng/openshift-tests-extension/test/example"
	"github.com/spf13/cobra"
)

func main() {
	registry := e.NewRegistry()
	ext := e.NewExtension("openshift", "payload", "olmv1-origin-tests-extension")

	ext.AddSuite(
		e.Suite{
			Name:    "example/tests",
			Parents: []string{"openshift/conformance/parallel"},
		})

	// The tests that a suite is composed of can be filtered by CEL expressions. By
	// default, the qualifiers only apply to tests from this extension.
	ext.AddSuite(e.Suite{
		Name: "example/fast",
		Qualifiers: []string{
			`!labels.exists(l, l=="SLOW")`,
		},
	})

	// Global suites' qualifiers will apply to all tests available, even
	// those outside of this extension (when invoked by origin).
	ext.AddGlobalSuite(e.Suite{
		Name: "example/slow",
		Qualifiers: []string{
			`labels.exists(l, l=="SLOW")`,
		},
	})

	// If using Ginkgo, build test specs automatically
	specs, err := g.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite()
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ginkgo: %+v", err.Error()))
	}

	// Environment selector information can be added to test specs to support filtering by environment
	specs.Select(et.NameContains("[sig-testing] openshift-tests-extension should support test-skips via environment flags")).
		Include(et.PlatformEquals("aws"))

	// You can add hooks to run before/after tests. There are BeforeEach, BeforeAll, AfterEach,
	// and AfterAll. "Each" functions must be thread safe.
	//
	// specs.AddBeforeAll(func() {
	// 	initializeTestFramework()
	// })
	//
	// specs.AddBeforeEach(func(spec ExtensionTestSpec) {
	//	if spec.Name == "my test" {
	//		// do stuff
	//	}
	// })
	//
	// specs.AddAfterEach(func(res *ExtensionTestResult) {
	// 	if res.Result == ResultFailed && apiTimeoutRegexp.Matches(res.Output) {
	// 		res.AddDetails("api-timeout", collectDiagnosticInfo())
	// 	}
	// })

	// You can also manually build a test specs list from other testing tooling
	// TODO: example

	// Modify specs, such as adding a label to all specs
	// 	specs = specs.AddLabel("SLOW")

	// Specs can be globally filtered...
	// specs = specs.MustFilter([]string{`name.contains("filter")`})

	// Or walked...
	// specs = specs.Walk(func(spec *extensiontests.ExtensionTestSpec) {
	//	if strings.Contains(e.Name, "scale up") {
	//		e.Labels.Insert("SLOW")
	//	}
	//
	// Specs can also be selected...
	// specs = specs.Select(et.NameContains("slow test")).AddLabel("SLOW")
	//
	// Or with "any" (or) matching selections
	// specs = specs.SelectAny(et.NameContains("slow test"), et.HasLabel("SLOW"))
	//
	// Or with "all" (and) matching selections
	// specs = specs.SelectAll(et.NameContains("slow test"), et.HasTagWithValue("speed", "slow"))
	//
	// There are also Must* functions for any of the above flavors of selection
	// which will return an error if nothing is found
	// specs, err = specs.MustSelect(et.NameContains("slow test")).AddLabel("SLOW")
	// if err != nil {
	//    logrus.Warn("no specs found: %w", err)
	// }
	// Test renames
	//	if spec.Name == "[sig-testing] openshift-tests-extension has a test with a typo" {
	//		spec.OriginalName = `[sig-testing] openshift-tests-extension has a test with a tpyo`
	//	}
	//
	// Filter by environment flags
	// if spec.Name == "[sig-testing] openshift-tests-extension should support defining the platform for tests" {
	//		spec.Include(et.PlatformEquals("aws"))
	//		spec.Exclude(et.And(et.NetworkEquals("ovn"), et.TopologyEquals("ha")))
	//	}
	// })

	ext.AddSpecs(specs)
	registry.Register(ext)

	root := &cobra.Command{
		Long: "OLMv1 Origin Tests Extension",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	if err := func() error {
		return root.Execute()
	}(); err != nil {
		os.Exit(1)
	}
}
