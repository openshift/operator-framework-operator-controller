# Local Dev Output Commands

Developer-friendly test commands with clean, human-readable output for local execution.

## Structure

```text
localdevoutput/
├── cmd/                          # Cobra commands for local development
│   ├── register.go               # Returns nil (production build)
│   ├── register_local_dev.go     # Registers local dev commands (with -tags dev)
│   ├── run_suite_local_dev.go    # run-suite-dev command
│   └── run_test_local_dev.go     # run-test-dev command
└── pkg/output/                   # Output formatting
    ├── formatter.go              # ANSI colors, progress, summaries
    └── writer.go                 # OTE ResultWriter implementation
```

## Purpose

These commands are **for local execution only**. They provide human-readable output when testing against OCP clusters locally.

## Build Separation

- **Production build**: `make build` → Excludes local dev commands (ships to OCP payload)
- **Local dev build**: `make build-local-dev` → Includes local dev commands (for developers)

Build tags (`//go:build dev`) ensure dev commands are only compiled when explicitly requested with `-tags dev`.

### Output Flow

```text
Test Run → ResultWriter.Write(result) → Formatter → Colored Terminal Output
```

The `CleanResultWriter` implements OTE's `ResultWriter` interface to intercept test results and format them with colors and progress indicators.

## Usage

```bash
# Run test suite with clean output
make test-local SUITE=olmv1/all

# Run single test
make test-local-single TEST="[sig-olmv1] test name"

# Direct binary usage
./bin/olmv1-tests-ext run-suite-dev olmv1/all
./bin/olmv1-tests-ext run-test-dev -n "test name"
```

## Why Separate Directory?

Isolated in `localdevoutput/` to keep local development tools separate from the core test framework that ships to production.
