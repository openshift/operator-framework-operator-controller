# Long-Duration OLM v1 Test Framework

## Overview

A comprehensive testing framework for running OLM v1 tests continuously over extended periods (days/weeks) with intelligent anomaly detection and health monitoring.

## Key Features

✨ **Three-Layer Detection System**:
1. **Layer 1**: Overall pass rate monitoring (Bayesian anomaly detection)
2. **Layer 2**: Individual test regression (Fisher's Exact Test) - **ON by default**
3. **Layer 3**: Fast failure detection (Consecutive failures) - Optional

🔔 **Automatic Slack Notifications**:
- Overall quality degradation alerts
- Individual test regression alerts
- OLM v1 health check failures
- Test completion summaries

📊 **Google Sheets Integration**:
- Automatic result logging (summary statistics)
- Failed test tracking with detailed error messages
- Historical tracking
- Statistical summaries

🏥 **OLM v1 Health Monitoring**:
- Pre-test cluster validation
- Component health checks
- Automatic failure detection
- Must-gather on abnormal exit (optional)

## Quick Start

### TL;DR - Full Featured Example (Recommended)

```bash
python run_e2e.py -s -b ../bin/olmv1-tests-ext \
  --exclude-patterns "Disconnected" "Slow" \
  -m 15 \
  --kube-burner-binary /usr/local/bin/kube-burner-ocp \
  --check-olmv1-health \
  --google-credentials $GOOGLE_CREDENTIALS \
  -i 60 \
  --google-sheet-id $GOOGLE_SHEET_ID \
  --google-sheet-per-run \
  --min-pass-rate 80 \
  --slack-webhook $SLACK_WEBHOOK
```

### Basic Usage

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  -l run.log
```

This runs with **default configuration**:
- ✅ Overall pass rate monitoring (Bayesian)
- ✅ Individual test regression detection (Fisher's Test)
- ❌ Consecutive failure alerts (disabled)


## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Three-Layer Detection System                │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Layer 1: Overall Quality (Bayesian)                         │
│  ├─ Run #1: Gate check ≥ 85% (ONE TIME ONLY)                │
│  ├─ Run #4+: Z-score anomaly detection (statistical)        │
│  ├─ Status: Always ON with -s flag                          │
│  └─ Purpose: Detect overall test suite quality degradation  │
│     NOTE: 85% threshold only checked on Run #1 as a gate!   │
│                                                               │
│  Layer 2: Individual Tests (Fisher's Exact Test)            │
│  ├─ Activation: Run #30+                                     │
│  ├─ Status: ON by default (--disable-fishers-test to turn OFF)│
│  └─ Purpose: Statistical regression detection per test      │
│                                                               │
│  Layer 3: Individual Tests (Consecutive Failures)           │
│  ├─ Activation: Run #3+                                      │
│  ├─ Status: OFF by default (--enable-consecutive-failure-check)│
│  └─ Purpose: Fast alerts for obviously broken tests         │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Default Behavior

**When you use `-s` (stats mode)**:
- ✅ Layer 1: Bayesian detection - **ENABLED**
- ✅ Layer 2: Fisher's Exact Test - **ENABLED** (NEW: default ON)
- ❌ Layer 3: Consecutive failures - **DISABLED** (opt-in only)

This provides **statistical rigor** while minimizing **alert noise**.

## Command Line Parameters

### Core Parameters
```bash
-b, --binary PATH              # Path to test binary (required)
-s, --stats                    # Enable statistics and detection (recommended)
-d, --days DAYS               # Total duration in days
-i, --interval SECONDS        # Interval between runs (default: 600)
-l, --logfile PATH            # Log file path
-o, --once                    # Run once and exit
```

### Detection Control
```bash
--min-pass-rate PERCENT            # First run threshold (default: 85.0)
--anomaly-threshold FLOAT          # Bayesian threshold (default: 3.0)
--disable-fishers-test             # Disable Fisher's test (enabled by default)
--enable-consecutive-failure-check # Enable consecutive failure detection
```

### Test Filtering
```bash
--include-patterns PATTERNS   # Include tests matching patterns
--exclude-patterns PATTERNS   # Exclude tests matching patterns
```

### Health & Notifications
```bash
--check-olmv1-health          # Enable OLM v1 health checks
--slack-webhook URL           # Slack webhook for notifications
--must-gather-on-error        # Run must-gather on abnormal exit
--must-gather-dir DIR         # Must-gather output directory (default: ./must-gather)
```

### Google Sheets Integration
```bash
--google-sheet-id ID               # Google Spreadsheet ID
--google-sheet-name NAME           # Main summary sheet name (default: "OLM Overall Performance Test Report")
--google-sheet-start-cell CELL     # Start cell for writing data (e.g., 'B30'). Auto-detects if not specified.
--google-sheet-per-run             # Write per-run results after each test run
--google-sheet-per-run-name NAME   # Per-run sheet name (default: "Long-duration Test Results")
--google-credentials PATH          # Service account credentials JSON
```

## Common Usage Patterns

### 1. Production Long-Duration Test

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  --interval 600 \
  --check-olmv1-health \
  --must-gather-on-error \
  --slack-webhook $SLACK_WEBHOOK \
  --google-sheet-id YOUR_SHEET_ID \
  --google-sheet-per-run \
  --google-credentials ~/google-creds.json \
  -l run.log
```

**What this does**:
- Runs for 5 days, tests every 10 minutes
- Fisher's Test enabled by default (statistical regression detection)
- OLM v1 health check before each run
- **Automatically runs must-gather if test exits abnormally** (health check failure, first run below threshold, unexpected errors)
- Sends Slack alerts when issues detected
- Logs results to Google Sheets after EACH run (real-time tracking)

### 2. Fast Detection Mode

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 2 \
  --interval 300 \
  --enable-consecutive-failure-check \
  --consecutive-failure-threshold 2 \
  --slack-webhook $SLACK_WEBHOOK \
  -l run.log
```

**What this does**:
- Shorter intervals (5 minutes)
- Both Fisher's test and consecutive failure detection
- More aggressive consecutive threshold (2 instead of 3)
- Faster problem detection

### 3. Simple Monitoring (No Test-Case Level Detection)

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 7 \
  --disable-fishers-test \
  -l run.log
```

**What this does**:
- Only overall pass rate monitoring (Layer 1)
- No individual test case analysis
- Minimal overhead

### 4. Development/Testing

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --max-runs 10 \
  --enable-consecutive-failure-check \
  -l run.log
```

**What this does**:
- Run only 10 times
- Both detection layers active
- Quick validation

## Alert Examples

### Bayesian Anomaly (Overall Quality Drop)

```
:warning: Bayesian Anomaly Detection Alert

Run: #15
Current Pass Rate: 88.00%
Historical Mean: 95.20%
Z-Score: 8.57 (threshold: 3.0)
Deviation: 7.20%

This pass rate is statistically unusual compared to historical data.

Recent Pass Rates: 95.5%, 96.2%, 94.8%, 95.1%, 88.0%
```

### Fisher's Test Regression (Individual Test)

```
:warning: Test Regression Detected (Fisher's Exact Test)

Test Case: [sig-olm] OLM v1 ClusterExtension should support upgrades

Statistical Analysis:
• Historical Pass Rate: 90.0% (18/20 runs)
• Recent Pass Rate: 60.0% (6/10 runs)
• Fisher's p-value: 0.000892
• Significance Level: 0.001111 (Bonferroni corrected)
• Odds Ratio: 6.00x

Result: **Statistically Significant Regression**
Severity: MEDIUM
```

### Consecutive Failures (Fast Alert)

```
:x: Consecutive Test Failure Alert

Test Case: [sig-olm] OLM v1 should install operator

Consecutive Failures: 3 (threshold: 3)
Recent Results: ✅✅✅❌✅❌❌❌

Severity: HIGH - This test has failed 3 times in a row
```

## Dependencies

### Required
- Python 3.6+
- `olmv1-tests-ext` binary
- OpenShift cluster access (`oc` command)

### Optional (for Full Features)
```bash
# For Fisher's Exact Test
pip install scipy

# For Google Sheets integration
pip install google-auth google-auth-oauthlib google-auth-httplib2 google-api-python-client
```

## Files Generated

| File | Purpose | Size (typical) |
|------|---------|----------------|
| `pass_rates_history.json` | Overall pass rates | ~5KB per 100 runs |
| `test_case_history.json` | Individual test results | ~50KB for 45 tests × 100 runs |
| `junit_e2e_*.xml` | Test execution results | ~200KB per run |
| `run.log` | Execution log | Varies |
| `command.log` | Command history | ~1KB per run |

## Google Sheets Structure

When `--google-sheet-id` is configured, the framework creates and updates multiple sheets:

### 1. Main Summary Sheet (default: "OLM Overall Performance Test Report")
**Written**: Once at test completion
**Contains**: Overall test run statistics with PASS/FAIL status

| Column | Description |
|--------|-------------|
| **Status** | PASS or FAIL (based on avg_pass_rate vs min_pass_rate threshold) |
| Date | Test start date |
| OCP Version | OpenShift version |
| Test Duration | Total duration (e.g., "5d 2h") |
| Total Runs | Number of test runs completed |
| Avg Pass Rate | Average pass rate across all runs |
| Min Pass Rate | Minimum pass rate observed |
| Max Pass Rate | Maximum pass rate observed |
| Total Tests | Number of tests per run |
| Total Failures | Number of failures + errors in last run |
| **Skipped** | Number of skipped tests in last run |
| Failed Tests Summary | Semicolon-separated list of failed test names |
| Notes | Threshold and start time info |

**Features**:
- ✅ Auto-detects existing header location (searches for "Status" cell)
- ✅ Appends data below the header automatically
- ✅ One glance PASS/FAIL status for quick assessment

### 2. Per-Run Results Sheet (default: "Long-duration Test Results")
**Written**: After each test run (if `--google-sheet-per-run` is enabled)
**Contains**: Individual run statistics

| Column | Description |
|--------|-------------|
| Run # | Test run number (1, 2, 3...) |
| Date | Date of this run (YYYY-MM-DD) |
| OCP Version | OpenShift version |
| Pass Rate (%) | Pass rate for this specific run |
| Total Tests | Number of tests in this run |
| **Skipped** | Number of skipped tests |
| Failures | Number of failed tests |
| Errors | Number of error tests |
| Failed Test Names | Semicolon-separated list of failed tests |
| Started Time | When the test session started |

**Benefits**:
- ✅ Real-time progress tracking (see results while tests are still running)
- ✅ Identify trends across runs (visualize pass rate changes over time)
- ✅ Create charts and pivot tables in Google Sheets
- ✅ Share live results with team members

### 3. Failed Tests Sheet (auto-created: "Failed Tests")
**Written**: After each test run (always, if any failures occur)
**Contains**: Detailed failure information for each failed test
- **Run #**: Test run number
- **Timestamp**: When the test failed
- **Test Name**: Full test case name
- **Failure Type**: FAILURE or ERROR
- **Error Message**: Detailed error message (truncated to 500 chars)

**Benefits**:
- ✅ Track which tests fail over time
- ✅ Identify patterns in test failures
- ✅ Correlate failures with specific runs
- ✅ Export failure data for further analysis

## Performance

- **Overhead**: < 0.1% of test execution time
- **Typical test duration**: ~10 minutes
- **Detection overhead**: ~550ms per run
- **Memory usage**: < 100MB
- **Disk usage**: ~10MB per 100 runs

## Must-Gather on Abnormal Exit

The `--must-gather-on-error` feature automatically collects cluster diagnostic information when the test exits abnormally.

### When Must-Gather Runs

Must-gather is executed automatically when **any** of these occur:
- ❌ OLM v1 health check fails
- ❌ First run pass rate below minimum threshold
- ❌ Unexpected exceptions/errors

Must-gather is **NOT** executed for:
- ✅ Normal completion (test runs for specified duration)
- ✅ User interruption (Ctrl+C)
- ✅ Bayesian anomaly alerts (continues running)
- ✅ Fisher's test regression alerts (continues running)

### Output Location

```
./must-gather/
  └── must-gather-20251202-143025/   # Timestamped directory
      ├── event-filter.html
      ├── gather-debug.log
      ├── quay-io-*/                  # Collected data
      └── timestamp
```

### Usage Example

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --check-olmv1-health \
  --must-gather-on-error \
  --must-gather-dir /data/must-gather \
  -l run.log
```

**Benefits**:
- Automatic diagnostic collection without manual intervention
- Timestamped directories preserve multiple must-gather runs
- Contains complete cluster state at time of failure
- Useful for post-mortem analysis and bug reports

## Troubleshooting

### No Fisher's Test Alerts
- **Cause**: Need 30+ runs for baseline
- **Solution**: Wait for more runs or check `test_case_history.json`

### scipy Not Available
- **Impact**: Fisher's test skipped
- **Solution**: `pip install scipy`

### Too Many Alerts
- **Consecutive failures**: Disable with default settings (already off)
- **Fisher's test**: Increase `--fishers-alpha` or disable with `--disable-fishers-test`

### Tests Not Found
- **Check**: `--include-patterns` and `--exclude-patterns`
- **Debug**: Run with `--dry-run` first

## Best Practices

1. **Start with defaults**: Use `-s` flag without additional parameters
2. **Let Fisher's test build baseline**: Wait for 30+ runs before evaluating
3. **Monitor Slack alerts**: Set up proper webhook for notifications
4. **Review periodically**: Check `pass_rates_history.json` for trends
5. **Reset when needed**: Delete history files after major cluster changes

