//go:build dev

package cmd

import (
	"github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	"github.com/spf13/cobra"
)

func RegisterLocalDevCommands(registry *extension.Registry) []*cobra.Command {
	return []*cobra.Command{
		NewRunSuiteDevCommand(registry),
		NewRunTestDevCommand(registry),
	}
}
