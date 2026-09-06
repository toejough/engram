"""
Unit tests for build_scorecard.py's bootstrap confidence-interval helpers.

Covers:
1. bootstrap_ci determinism under the fixed seed
2. Degenerate all-true list -> tight interval at 100
3. 50/50 list of n=100 -> interval containing 50, roughly +/-10 wide
4. rate_block passthrough: flags -> ci95 next to rate_pct; no flags -> no ci95
5. rate_block consistency assertions between flags and stated n's
"""

import pytest

from build_scorecard import bootstrap_ci, rate_block


# ---------------------------------------------------------------------------
# bootstrap_ci
# ---------------------------------------------------------------------------

def test_bootstrap_ci_deterministic_under_fixed_seed():
    flags = [True] * 13 + [False] * 37
    first = bootstrap_ci(flags)
    second = bootstrap_ci(flags)
    assert first == second
    # explicit-arg call matches the defaults it documents
    assert bootstrap_ci(flags, n_boot=2000, alpha=0.05, seed=739) == first


def test_bootstrap_ci_all_true_is_tight_at_100():
    ci = bootstrap_ci([True] * 30)
    assert ci == {"lo_pct": 100.0, "hi_pct": 100.0}


def test_bootstrap_ci_all_false_is_tight_at_0():
    ci = bootstrap_ci([False] * 30)
    assert ci == {"lo_pct": 0.0, "hi_pct": 0.0}


def test_bootstrap_ci_half_and_half_n100_contains_50_roughly_pm10():
    flags = [True] * 50 + [False] * 50
    ci = bootstrap_ci(flags)
    assert ci["lo_pct"] < 50.0 < ci["hi_pct"]
    # binomial 95% half-width at p=.5, n=100 is ~9.8; allow slack around that
    assert 35.0 <= ci["lo_pct"] <= 45.0
    assert 55.0 <= ci["hi_pct"] <= 65.0


def test_bootstrap_ci_empty_list_returns_none_bounds():
    assert bootstrap_ci([]) == {"lo_pct": None, "hi_pct": None}


# ---------------------------------------------------------------------------
# rate_block passthrough
# ---------------------------------------------------------------------------

def test_rate_block_with_flags_adds_ci95_next_to_rate_pct():
    flags = [True] * 30 + [False] * 70
    block = rate_block(
        question="q", counting_unit="per-moment", label="DERIVED",
        denom_n=100, denom_def="d", num_n=30, num_def="n", flags=flags,
    )
    assert block["rate_pct"] == 30.0
    ci = block["ci95"]
    assert set(ci) == {"lo_pct", "hi_pct"}
    assert ci["lo_pct"] < 30.0 < ci["hi_pct"]
    # deterministic: identical to calling the helper directly
    assert ci == bootstrap_ci(flags)
    # "next to rate_pct": ci95 immediately follows rate_pct in key order
    keys = list(block)
    assert keys.index("ci95") == keys.index("rate_pct") + 1


def test_rate_block_without_flags_has_no_ci95():
    block = rate_block(
        question="q", counting_unit="per-moment", label="DERIVED",
        denom_n=10, denom_def="d", num_n=4, num_def="n",
    )
    assert "ci95" not in block
    assert block["rate_pct"] == 40.0


def test_rate_block_flags_sum_mismatch_raises():
    with pytest.raises(AssertionError):
        rate_block(
            question="q", counting_unit="per-moment", label="DERIVED",
            denom_n=4, denom_def="d", num_n=3, num_def="n",
            flags=[True, False, False, False],  # sum 1 != num_n 3
        )


def test_rate_block_flags_length_mismatch_raises():
    with pytest.raises(AssertionError):
        rate_block(
            question="q", counting_unit="per-moment", label="DERIVED",
            denom_n=5, denom_def="d", num_n=2, num_def="n",
            flags=[True, True, False],  # len 3 != denom_n 5
        )
