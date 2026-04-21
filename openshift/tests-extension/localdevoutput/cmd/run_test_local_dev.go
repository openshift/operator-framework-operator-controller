//go:build dev

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	"github.com/openshift-eng/openshift-tests-extension/pkg/flags"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/localdevoutput/pkg/output"
)

func NewRunTestDevCommand(registry *extension.Registry) *cobra.Command {
	opts := struct {
		componentFlags *flags.ComponentFlags
		nameFlags      *flags.NamesFlags
	}{
		componentFlags: flags.NewComponentFlags(),
		nameFlags:      flags.NewNamesFlags(),
	}

	cmd := &cobra.Command{
		Use:   "run-test-dev [-n NAME...] [NAME]",
		Short: "Run individual tests with clean, human-readable output for local development",
		Long: `Run one or more tests with clean, human-readable output.

This command provides a developer-friendly alternative to run-test with:
  - Live test progress indicators
  - Color-coded pass/fail status
  - Running totals after each test
  - Clean summary with detailed failure information

Perfect for local development and debugging. For CI/CD integration, use run-test instead.

Examples:
  # Run a single test
  run-test-dev -n "[sig-olmv1] OLMv1 should pass a trivial sanity check"

  # Run multiple tests
  run-test-dev -n "test 1" -n "test 2"`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancelCause := context.WithCancelCause(context.Background())
			defer cancelCause(errors.New("exiting"))

			abortCh := make(chan os.Signal, 2)
			go func() {
				<-abortCh
				fmt.Fprintf(os.Stderr, "Interrupted, terminating tests")
				cancelCause(errors.New("interrupt received"))

				select {
				case sig := <-abortCh:
					fmt.Fprintf(os.Stderr, "Interrupted twice, exiting (%s)", sig)
					switch sig {
					case syscall.SIGINT:
						os.Exit(130)
					default:
						os.Exit(130)
					}

				case <-time.After(30 * time.Minute):
					fmt.Fprintf(os.Stderr, "Timed out during cleanup, exiting")
					os.Exit(130)
				}
			}()
			signal.Notify(abortCh, syscall.SIGINT, syscall.SIGTERM)

			ext := registry.Get(opts.componentFlags.Component)
			if ext == nil {
				return fmt.Errorf("component not found: %s", opts.componentFlags.Component)
			}

			names := opts.nameFlags.Names
			if len(args) > 0 {
				names = append(names, args...)
			}
			if len(names) == 0 {
				return fmt.Errorf("must specify at least one test name via -n flag or argument")
			}

			cleanWriter := output.NewCleanResultWriter(os.Stdout)

			specs, err := ext.FindSpecsByName(names...)
			if err != nil {
				return err
			}

			cleanWriter.SetTotalTests(len(specs))

			if err := specs.Run(ctx, cleanWriter, 1); err != nil {
				if flushErr := cleanWriter.Flush(); flushErr != nil {
					fmt.Fprintf(os.Stderr, "failed to write results: %v\n", flushErr)
				}
				return err
			}

			if err := cleanWriter.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write results: %v\n", err)
				return err
			}

			if cleanWriter.HasFailures() {
				return ErrTestsFailed
			}

			return nil
		},
	}

	opts.componentFlags.BindFlags(cmd.Flags())
	opts.nameFlags.BindFlags(cmd.Flags())

	return cmd
}
