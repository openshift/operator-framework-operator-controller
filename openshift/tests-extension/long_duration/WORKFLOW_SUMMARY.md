# Three-Layer Detection System - Quick Reference

## High-Level Workflow

```
START
  │
  ├─> [Optional] OLM v1 Health Check ──FAIL──> Slack Alert + EXIT
  │                                   └─PASS─┐
  │                                           │
  ├─> Run Test Suite ─────────────────────────┤
  │                                           │
  ├─> Parse JUnit Results ────────────────────┤
  │                                           │
  ├─> LAYER 1: Bayesian (Overall) ────────────┤
  │   • Run #1 ONLY: pass_rate ≥ 85%? ──NO──> EXIT
  │   • Run #2-3: Build baseline (no gate check)
  │   • Run #4+: Z-score anomaly? ──YES──> Slack Alert
  │                                           │
  ├─> LAYER 2: Fisher's Test (DEFAULT ON) ────┤
  │   • Run #30+: Statistical regression? ──YES──> Slack Alert
  │                                           │
  ├─> LAYER 3: Consecutive (OPTIONAL) ────────┤
  │   • Run #3+: 3+ consecutive fails? ──YES──> Slack Alert
  │                                           │
  └─> Loop until exit condition met
```

## Detection Activation Timeline

| Run # | Layer 1 (Bayesian) | Layer 2 (Fisher's) | Layer 3 (Consecutive) |
|-------|-------------------|-------------------|-----------------------|
| 1 | Gate check ≥ 85% (ONE TIME ONLY) | ❌ Need data | ❌ Need data |
| 2-3 | Build baseline (NO gate check) | ❌ Need data | ✅ If enabled |
| 4-29 | ✅ Z-score detection (NO gate check) | ❌ Need data | ✅ If enabled |
| 30+ | ✅ Z-score detection (NO gate check) | ✅ **Active** | ✅ If enabled |

**Note**: The 85% threshold is ONLY checked on Run #1 as a gate to ensure cluster health. After that, Layer 1 uses statistical Z-score detection based on historical data, NOT a fixed 85% threshold.

## Default Configuration

```bash
python run_e2e.py -b /path/to/binary -s --days 5
```

Enables:
- ✅ Layer 1: Bayesian (always on with `-s`)
- ✅ Layer 2: Fisher's Exact Test (**default ON**)
- ❌ Layer 3: Consecutive failures (off by default)

## Alert Types

| Alert | Severity | Layer | Action |
|-------|----------|-------|--------|
| OLM v1 unhealthy | CRITICAL | - | EXIT |
| First run < 85% | CRITICAL | L1 | EXIT |
| Bayesian anomaly | MEDIUM | L1 | Continue |
| Fisher's regression | MEDIUM | L2 | Continue |
| Consecutive failures | HIGH | L3 | Continue |

## Key Files

| File | Purpose | Used By |
|------|---------|---------|
| `pass_rates_history.json` | Overall pass rates | Layer 1 |
| `test_case_history.json` | Individual test results | Layer 2, 3 |
| `junit_e2e_*.xml` | Test execution results | All |

## Quick Command Reference

```bash
# Default (recommended)
-s

# Disable Fisher's test
-s --disable-fishers-test

# Enable consecutive failures
-s --enable-consecutive-failure-check

# Both Fisher's and consecutive
-s --enable-consecutive-failure-check

# Only consecutive (no Fisher's)
-s --disable-fishers-test --enable-consecutive-failure-check
```

## Data Requirements

| Layer | Minimum Runs | Typical Activation |
|-------|-------------|-------------------|
| Layer 1 (Bayesian) | 4 runs | 40 minutes (10-min interval) |
| Layer 2 (Fisher's) | 30 runs | 5 hours (10-min interval) |
| Layer 3 (Consecutive) | 3 runs | 30 minutes (10-min interval) |

## Example: 5-Day Test Execution

```
Day 1:
  0:00  → Run #1  → Layer 1: Check ≥ 85% ✓
  0:30  → Run #3  → Layer 3: Can detect (if enabled)
  0:40  → Run #4  → Layer 1: Bayesian active ✨
  5:00  → Run #30 → Layer 2: Fisher's active ✨

Day 2-5:
  All layers active
  Continuous monitoring
  Alerts sent to Slack as detected

End:
  → Final statistics
  → Write to Google Sheets
  → Send completion notification
```

## Performance Impact

- **Overhead per run**: ~550ms
- **Test execution**: ~10 minutes
- **Impact**: < 0.1% (negligible)
