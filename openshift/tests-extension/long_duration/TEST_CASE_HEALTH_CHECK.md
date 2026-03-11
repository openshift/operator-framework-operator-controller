# Test Case Health Check System

## Overview

The long-duration test framework now implements a **three-layer detection system** to monitor both overall test quality and individual test case health:

1. **Layer 1**: Overall Pass Rate Bayesian Detection (existing) - Detects overall quality issues
2. **Layer 2**: Fisher's Exact Test (new) - Detects statistical regression in individual test cases
3. **Layer 3**: Consecutive Failure Detection (new) - Quick alerts for tests failing repeatedly

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Test Execution Completed                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │   Layer 1: Bayesian Detection    │
        │   (Overall Pass Rate)             │
        │   • Checks: Run #4+               │
        │   • Detects: Quality drops        │
        └──────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────────┐
        │  Enable Test Case Health Check?  │
        │  (--enable-test-case-health-check)│
        └──────────┬───────────────────────┘
                   │ YES
                   ▼
        ┌──────────────────────────────────┐
        │   Parse Individual Test Results   │
        │   Save to test_case_history.json  │
        └──────────┬───────────────────────┘
                   │
      ┌────────────┴────────────┐
      ▼                         ▼
┌─────────────────┐    ┌──────────────────────┐
│   Layer 3:      │    │   Layer 2:           │
│   Consecutive   │    │   Fisher's Exact     │
│   Failures      │    │   Test               │
│                 │    │                      │
│ • Always runs   │    │ • Needs 30+ runs     │
│ • Threshold: 3  │    │ • Statistical test   │
│ • Fast response │    │ • Less false alarms  │
└─────┬───────────┘    └──────┬───────────────┘
      │                       │
      │ Alert if failing      │ Alert if regression
      │                       │
      └───────────┬───────────┘
                  ▼
        ┌──────────────────────┐
        │  Send Slack Alerts   │
        │  (if configured)     │
        └──────────────────────┘
```

## Layer 1: Overall Pass Rate Bayesian Detection

**Already implemented** - See [BAYESIAN_ANOMALY_DETECTION.md](./BAYESIAN_ANOMALY_DETECTION.md)

- **What**: Monitors overall test suite pass rate
- **When**: After run #4 (needs 3 runs to build baseline)
- **How**: Z-score calculation (standard deviations from mean)
- **Threshold**: Default 2.0 σ (configurable via `--anomaly-threshold`)

**Example Alert**:
```
:warning: Bayesian Anomaly Detection Alert

Run: #8
Current Pass Rate: 88.00%
Historical Mean: 95.20%
Z-Score: 8.57 (threshold: 3.0)
```

## Layer 2: Fisher's Exact Test

**Newly implemented** - Statistical test for individual test case regression

### How It Works

Fisher's Exact Test compares two time periods:
- **Historical window**: Older runs (default: 20 runs)
- **Recent window**: Most recent runs (default: 10 runs)

It builds a 2x2 contingency table:

```
                  Pass    Fail
Historical (20)   18      2      → 90% pass rate
Recent (10)       6       4      → 60% pass rate
```

Then calculates: Is the recent failure rate **statistically significantly higher** than historical?

### Key Features

1. **Bonferroni Correction**: Adjusts for multiple testing
   - If testing 45 test cases
   - Alpha = 0.05 / 45 = 0.0011
   - Reduces false positives

2. **Requires Sufficient Data**:
   - Minimum: historical_window + recent_window runs
   - Default: 20 + 10 = 30 runs total

3. **Detects Gradual Degradation**:
   ```
   Historical: Pass, Pass, Pass, Pass, Fail, Pass, ...  (90% pass)
   Recent:     Pass, Fail, Pass, Fail, Fail, Fail      (40% pass)

   → Fisher's test detects this is statistically significant
   ```

### Configuration Parameters

```bash
# Fisher's Exact Test is ENABLED BY DEFAULT (no flag needed)
--disable-fishers-test            # Disable Fisher's test (optional)
--fishers-historical-window 20    # Historical baseline (default: 20)
--fishers-recent-window 10        # Recent comparison (default: 10)
--fishers-alpha 0.05              # Significance level (default: 0.05)
```

### Example Alert

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

Severity: MEDIUM - Test is failing more often than before

Run: #35
Timestamp: 2025-11-18 22:00:00
```

## Layer 3: Consecutive Failure Detection

**Newly implemented** - Fast, simple rule-based detection

### How It Works

Counts how many times a test has failed **in a row** (consecutively):

```
Test results: [Pass, Pass, Fail, Fail, Fail]
                             ↑    ↑    ↑
                       3 consecutive failures → ALERT!
```

### Key Features

1. **Immediate Response**: Alerts as soon as threshold is reached
2. **Simple Logic**: Easy to understand and debug
3. **No Data Requirements**: Works from run #3 onwards
4. **Reset on Pass**: Counter resets when test passes once

### Configuration Parameters

```bash
# Consecutive failure check is DISABLED BY DEFAULT
--enable-consecutive-failure-check  # Enable this layer (optional)
--consecutive-failure-threshold 3   # Default: 3 consecutive failures
```

### Example Alert

```
:x: Consecutive Test Failure Alert

Test Case: [sig-olm] OLM v1 should install operator from catalog

Consecutive Failures: 3 (threshold: 3)
Total Runs: 15
Recent Results: ✅✅✅❌✅❌❌❌

Severity: HIGH - This test has failed 3 times in a row

Run: #15
Timestamp: 2025-11-18 22:00:00
```

## Default Behavior

**By default when using `-s` (stats mode)**:
- ✅ Layer 1: Bayesian detection - **ENABLED** (always on with `-s`)
- ✅ Layer 2: Fisher's Exact Test - **ENABLED** (automatically runs)
- ❌ Layer 3: Consecutive failures - **DISABLED** (opt-in only)

This means **Fisher's Exact Test runs by default** for statistical rigor while avoiding noise from simple consecutive failure alerts.

## Usage Examples

### Basic Usage (Default: Fisher's Test Enabled)

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5
```

This enables:
- ✅ Layer 1: Bayesian detection (always on with `-s`)
- ✅ Layer 2: Fisher's Exact Test (**enabled by default**)
- ❌ Layer 3: Consecutive failures (disabled)

### Enable Consecutive Failure Detection

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  --enable-consecutive-failure-check
```

This enables:
- ✅ Layer 1: Bayesian detection
- ✅ Layer 2: Fisher's Exact Test (default)
- ✅ Layer 3: Consecutive failures (**now enabled**)

### Disable Fisher's Test (Use Only Consecutive Failures)

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  --disable-fishers-test \
  --enable-consecutive-failure-check
```

This configuration:
- ✅ Layer 1: Bayesian detection
- ❌ Layer 2: Fisher's Exact Test (**disabled**)
- ✅ Layer 3: Consecutive failures (enabled)

### Custom Thresholds

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  --enable-consecutive-failure-check \
  --consecutive-failure-threshold 2 \
  --fishers-historical-window 30 \
  --fishers-recent-window 10 \
  --fishers-alpha 0.01
```

This configuration:
- More sensitive consecutive failures (2 instead of 3)
- Longer historical baseline (30 runs instead of 20)
- More conservative Fisher's test (alpha 0.01 instead of 0.05)
- Both Fisher's test and consecutive failure detection enabled

### Recommended Configuration (Production)

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  --interval 600 \
  --slack-webhook https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -l run.log
```

**This is the recommended setup for most users**:
- Fisher's Exact Test runs by default (statistical rigor)
- Consecutive failure check disabled (avoids noise)
- Clean, reliable alerts with low false positives

### Recommended Configuration (With Consecutive Failures for Fast Detection)

```bash
python run_e2e.py \
  -b /path/to/olmv1-tests-ext \
  -s \
  --days 5 \
  --interval 600 \
  --enable-consecutive-failure-check \
  --consecutive-failure-threshold 3 \
  --slack-webhook https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -l run.log
```

**Use this if you want both statistical detection AND fast alerts**:
- Fisher's Exact Test (default)
- Consecutive failure detection (enabled)
- More comprehensive but may produce more alerts

## Data Storage

### test_case_history.json

Stores individual test case results across runs:

```json
{
  "timestamp": "2025-11-18 22:00:00",
  "test_cases": {
    "[sig-olm] OLM v1 should install operator": [
      true, true, true, false, true, false, false, false
    ],
    "[sig-olm] OLM v1 ClusterExtension should support upgrades": [
      true, true, true, true, true, true, true, true
    ]
  }
}
```

- `true` = Test passed
- `false` = Test failed (or skipped)
- Array is ordered chronologically (oldest → newest)

## Comparison: When Each Layer Triggers

### Scenario 1: Sudden Complete Failure

```
Overall pass rate: 100% → 100% → 95% → 93% → 90%
Test "upgrade":    Pass → Pass → Fail → Fail → Fail
```

| Layer | Detection Time | Reason |
|-------|----------------|--------|
| Layer 3 (Consecutive) | Run #5 | 3 consecutive failures ⚡ FASTEST |
| Layer 2 (Fisher's) | Not yet | Need 30+ total runs |
| Layer 1 (Bayesian) | Run #5 or #6 | Overall rate dropped significantly |

### Scenario 2: Gradual Degradation

```
Overall pass rate: 95% → 94% → 93% → 92% → 91%
Test "upgrade":    90% → 85% → 80% → 70% → 60%
                  (9/10) (8.5/10 avg) ...
```

| Layer | Detection Time | Reason |
|-------|----------------|--------|
| Layer 3 (Consecutive) | Never | Failures not consecutive ❌ |
| Layer 2 (Fisher's) | Run #30+ | Detects trend statistically ✅ |
| Layer 1 (Bayesian) | Run #6+ | Overall rate drifting down ✅ |

### Scenario 3: Intermittent (Flaky) Failures

```
Overall pass rate: 95% stable
Test "upgrade":    Pass, Fail, Pass, Fail, Pass, Fail, Fail, Fail
                  (50% recent failure rate)
```

| Layer | Detection Time | Reason |
|-------|----------------|--------|
| Layer 3 (Consecutive) | Run #8 | Last 3 are failures ⚡ |
| Layer 2 (Fisher's) | Run #30+ | High failure rate vs historical ✅ |
| Layer 1 (Bayesian) | Maybe not | Only 1 test failing, overall OK |

## When to Use Each Layer

### Use Layer 1 (Bayesian) for:
- ✅ Detecting overall test suite quality issues
- ✅ Finding systemic problems (cluster instability, infrastructure issues)
- ✅ Quick detection with minimal data (4 runs)

### Use Layer 2 (Fisher's Test) for:
- ✅ Statistical confidence in individual test regression
- ✅ Detecting gradual degradation over time
- ✅ Reducing false positives with Bonferroni correction
- ❌ Requires 30+ runs (slow to activate)

### Use Layer 3 (Consecutive) for:
- ✅ Fast alerts for broken tests
- ✅ Detecting tests that suddenly start failing
- ✅ Simple, easy to understand
- ❌ Misses intermittent failures

## Best Practices

### 1. Start with All Three Layers

```bash
--enable-test-case-health-check  # Enables Layer 2 & 3
-s                               # Enables Layer 1
```

### 2. Adjust Thresholds Based on Experience

**Too many alerts?**
- Increase `--consecutive-failure-threshold` (3 → 4 or 5)
- Decrease `--fishers-alpha` (0.05 → 0.01)

**Missing real problems?**
- Decrease `--consecutive-failure-threshold` (3 → 2)
- Increase `--fishers-alpha` (0.05 → 0.10)

### 3. Monitor Slack Alerts

First few days: Expect some noise as system learns baseline
After 30+ runs: Fisher's test activates, more accurate detection

### 4. Historical Data Management

**Reset when needed**:
```bash
# If you make major cluster changes, reset history
rm test_case_history.json
rm pass_rates_history.json
```

### 5. Interpretation Guide

| Alert Type | Severity | Action |
|------------|----------|--------|
| Consecutive failures | HIGH | Investigate immediately - test is broken |
| Fisher's regression | MEDIUM | Review test - degrading over time |
| Bayesian anomaly | MEDIUM | Check cluster health - overall quality drop |

## Troubleshooting

### "scipy not available" Warning

**Cause**: Fisher's Exact Test requires scipy
**Solution**:
```bash
pip install scipy
```

### No Fisher's Test Alerts

**Cause 1**: Not enough historical data
**Solution**: Wait for 30+ runs (historical_window + recent_window)

**Cause 2**: Fisher's test was disabled
**Solution**: Remove `--disable-fishers-test` flag (Fisher's test is enabled by default)

### Too Many Consecutive Failure Alerts

**Cause**: Consecutive failure check is enabled and generating noise
**Solution 1**: Disable it (it's optional)
```bash
# Simply don't use --enable-consecutive-failure-check
```

**Solution 2**: If you want to keep it, increase threshold
```bash
--enable-consecutive-failure-check --consecutive-failure-threshold 4  # Increase from 3 to 4
```

### Fisher's Test Not Detecting Known Problem

**Cause**: Windows too large or too small
**Solution**: Adjust window sizes
```bash
# For faster detection (less data needed):
--fishers-historical-window 15 --fishers-recent-window 5

# For more stable detection:
--fishers-historical-window 30 --fishers-recent-window 15
```

## Dependencies

- **Required**: Python 3.6+
- **Optional (for Fisher's Test)**: scipy
  ```bash
  pip install scipy
  ```

## Files Created

| File | Purpose | Example Size |
|------|---------|--------------|
| `test_case_history.json` | Individual test results | ~50KB for 45 tests × 100 runs |
| `pass_rates_history.json` | Overall pass rates | ~5KB for 100 runs |

## Performance Impact

| Layer | Overhead per Run | Data Requirements |
|-------|-----------------|-------------------|
| Layer 1 (Bayesian) | < 1ms | 4+ runs |
| Layer 2 (Fisher's) | ~10ms per test × 45 tests = ~450ms | 30+ runs |
| Layer 3 (Consecutive) | < 1ms per test × 45 tests = ~45ms | 3+ runs |

**Total overhead**: < 500ms per run (negligible compared to test execution time)

## Summary

The three-layer detection system provides comprehensive monitoring:

1. **Layer 1 (Bayesian)**: Catches overall quality drops quickly (4 runs) - **Always ON with `-s`**
2. **Layer 2 (Fisher's Test)**: Provides statistical confidence for individual test issues (30+ runs) - **ON by default**
3. **Layer 3 (Consecutive)**: Fast alerts for obviously broken tests (3 runs) - **OFF by default (opt-in)**

### Default Configuration (Recommended)

**By default**, the framework uses:
- ✅ **Layer 1**: Bayesian detection (always on with `-s`)
- ✅ **Layer 2**: Fisher's Exact Test (**enabled by default**)
- ❌ **Layer 3**: Consecutive failures (disabled, opt-in with `--enable-consecutive-failure-check`)

This provides **statistical rigor** (Fisher's test) while avoiding **noise** (consecutive failures), giving you the best balance of accuracy and actionable alerts.
