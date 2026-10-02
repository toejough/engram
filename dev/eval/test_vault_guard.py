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


def test_every_harness_asserts_no_trial_leak():
    missing = [rel for rel in HARNESSES if "assert_no_trial_leak(" not in _src(rel)]
    assert not missing, f"these never assert no trial leaked into the real vault: {missing}"


# #750 R2/round-3: allowlist-scoped guard for the now-fully-migrated `_real_vault_fingerprint`
# removal — an EXPLICIT, named list of exactly the two files that used to reference it, never a
# dev/eval/-wide walk. A tree-wide "no file contains X" version could never pass: it would also
# match this guard test's own source (which must contain the literal string to check for it) and
# the tracked historical artifacts `audit/results-parent-capture-2026-09-29/{items.jsonl,
# items-judged.jsonl}` (quote `probe.py::_real_vault_fingerprint()` inside a historical transcript
# excerpt) — both correctly left untouched per this change's Doc-Surface Disposition convention.
# (`dev/eval/cumulative/runbook_vs_skill/test_probe.py` also now mentions the literal string, in
# its own mutation-sensitive guard tests for the two call sites that used to call it — the same
# self-reference class this guard's own source has, and likewise out of scope for this allowlist.)
REAL_VAULT_FINGERPRINT_MIGRATED_FILES = [
    "cumulative/runbook_vs_skill/probe.py",
    "cumulative/runbook_vs_skill/phase2/probe_phase2.py",
]


def test_real_vault_fingerprint_is_gone_from_its_two_former_call_sites():
    present = [rel for rel in REAL_VAULT_FINGERPRINT_MIGRATED_FILES if "_real_vault_fingerprint" in _src(rel)]
    assert not present, f"these still reference the retired _real_vault_fingerprint: {present}"
