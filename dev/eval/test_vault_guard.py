"""Every harness that spawns trials brackets its run with the real-vault fingerprint.

This is the backstop for a path the env vars do not cover — a subcommand resolving its own way, or
a future spawn site that skips isolated_env. It catches contamination at exit instead of days later
as unexplained real notes (#708, where seven notes were found and hand-deleted after the fact).
"""
import os

HERE = os.path.dirname(os.path.abspath(__file__))

HARNESSES = [
    "cumulative/please_step7_probe/run_probe.py",
    "cumulative/please_step3_probe/run_probe.py",
    "cumulative/endorse_cue/probe.py",
    "traps/brun.py",
    "traps/run.py",
    "traps/qanchor_eval.py",
    "traps/reasoning_eval.py",
]


def _src(rel):
    with open(os.path.join(HERE, rel)) as f:
        return f.read()


def test_every_harness_snapshots_the_real_vault():
    missing = [rel for rel in HARNESSES if "vault_fingerprint(" not in _src(rel)]
    assert not missing, f"these never snapshot the real vault before running trials: {missing}"


def test_every_harness_asserts_the_vault_is_unchanged():
    missing = [rel for rel in HARNESSES if "assert_vault_unchanged(" not in _src(rel)]
    assert not missing, f"these never assert the real vault survived the run: {missing}"
