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

	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	"github.com/openshift-eng/openshift-tests-extension/pkg/flags"

	"github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/localdevoutput/pkg/output"
)

// ErrTestsFailed is returned when one or more tests fail.
var ErrTestsFailed = errors.New("one or more tests failed")

func NewRunSuiteDevCommand(registry *extension.Registry) *cobra.Command {
	opts := struct {
		componentFlags   *flags.ComponentFlags
		concurrencyFlags *flags.ConcurrencyFlags
	}{
		componentFlags:   flags.NewComponentFlags(),
		concurrencyFlags: flags.NewConcurrencyFlags(),
	}

	cmd := &cobra.Command{
		Use:   "run-suite-dev NAME",
		Short: "Run a test suite with clean, human-readable output for local development",
		Long: `Run a test suite with clean, human-readable output.

This command provides a developer-friendly alternative to run-suite with:
  - Live test progress indicators
  - Color-coded pass/fail status
  - Running totals after each test
  - Clean summary with detailed failure information

Perfect for local development and debugging. For CI/CD integration, use run-suite instead.`,
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
			if len(args) != 1 {
				return fmt.Errorf("must specify one suite name")
			}
			suite, err := ext.GetSuite(args[0])
			if err != nil {
				return pkgerrors.Wrapf(err, "couldn't find suite: %s", args[0])
			}

			cleanWriter := output.NewCleanResultWriter(os.Stdout)

			specs, err := ext.GetSpecs().Filter(suite.Qualifiers)
			if err != nil {
				return pkgerrors.Wrap(err, "couldn't filter specs")
			}

			cleanWriter.SetTotalTests(len(specs))

			if err := specs.Run(ctx, cleanWriter, opts.concurrencyFlags.MaxConcurency); err != nil {
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
	opts.concurrencyFlags.BindFlags(cmd.Flags())

	return cmd
}
