"""Tier B: sites that set ENGRAM_VAULT_PATH but leave ENGRAM_CHUNKS_DIR unset.

An unset ENGRAM_CHUNKS_DIR silently resolves to the operator's global index. That is #642: a warm
arm read the operator's live sessions — the eval answer key — and scored round-1 8/8 off an
otherwise empty vault. Setting the vault alone is the half-fix that looks done.
"""
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))

TIER_B_PYTHON = [
    "traps/retrieval_probe.py",
    "traps/seed_c3.py",
    "traps/crowd.py",
    "traps/synth_fixtures.py",
    "traps/compound_fixtures.py",
    "traps/cake_fixtures.py",
    "traps/qanchor_retrieval_probe.py",
]

TIER_B_SHELL = [
    "run-chain-stage.sh",
    "run-layer-arm.sh",
    "run-layer-resume.sh",
]

SETS_VAULT_BY_HAND = re.compile(r'env\["ENGRAM_VAULT_PATH"\]\s*=')
SETS_CHUNKS = re.compile(r"ENGRAM_CHUNKS_DIR")


def _src(rel):
    with open(os.path.join(HERE, rel)) as f:
        return f.read()


def test_no_python_site_sets_a_vault_without_a_chunks_dir():
    """Direct `engram` CLI callers route through isolation.engram_env, which owns both paths.

    Asserted as "no hand-set vault survives" rather than "engram_env appears", because the
    failure mode is a site setting one path and forgetting the other — which is only possible
    when it sets them by hand.
    """
    offenders = [
        rel for rel in TIER_B_PYTHON
        if SETS_VAULT_BY_HAND.search(_src(rel)) and not SETS_CHUNKS.search(_src(rel))
    ]
    assert not offenders, (
        f"these set ENGRAM_VAULT_PATH by hand and never ENGRAM_CHUNKS_DIR: {offenders} — reads "
        "fall through to the operator's global chunk index (#642). Use isolation.engram_env()."
    )


def test_tier_b_python_sites_use_the_engram_env_helper():
    missing = [rel for rel in TIER_B_PYTHON if "isolation.engram_env(" not in _src(rel)]
    assert not missing, f"these build an engram env by hand: {missing}"


def test_no_shell_site_sets_a_vault_without_a_chunks_dir():
    offenders = [
        rel for rel in TIER_B_SHELL
        if "ENGRAM_VAULT_PATH" in _src(rel) and not SETS_CHUNKS.search(_src(rel))
    ]
    assert not offenders, (
        f"these shell harnesses set ENGRAM_VAULT_PATH but never ENGRAM_CHUNKS_DIR: {offenders}"
    )


def test_harness_claude_delegates_to_the_isolation_module():
    """One implementation of the isolation contract, not two."""
    src = _src("cumulative/harness.py")
    assert "isolation.isolated_env(" in src, (
        "harness.claude should build its env through isolation.isolated_env rather than "
        "hand-rolling the same four vars a second time"
    )
