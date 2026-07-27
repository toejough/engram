"""Every tier-A spawn site must build its env through isolation.isolated_env.

These seven sites hand-rolled `env = dict(os.environ)` and set only CLAUDE_CONFIG_DIR, which is
how #687's measurement wrote seven real notes into the operator's vault. A regression to a
hand-rolled env reopens the leak, so this asserts on the source text — the sites are in three
directories with no shared entry point to instrument, so there is nothing to monkeypatch that
covers them all. A text assertion is a weak test of behavior but a strong test of the property
that actually regressed: a site quietly building its own env.
"""
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))

TIER_A = [
    "cumulative/please_step7_probe/run_probe.py",
    "cumulative/please_step3_probe/run_probe.py",
    "cumulative/endorse_cue/probe.py",
    "traps/brun.py",
    "traps/run.py",
    "traps/qanchor_eval.py",
    "traps/reasoning_eval.py",
]

HANDROLLED = re.compile(r"^\s*env\s*=\s*dict\(os\.environ\)", re.M)


def _src(rel):
    with open(os.path.join(HERE, rel)) as f:
        return f.read()


def test_tier_a_sites_import_isolation():
    missing = [rel for rel in TIER_A if "import isolation" not in _src(rel)]
    assert not missing, f"these do not import the isolation module: {missing}"


def test_tier_a_sites_call_isolated_env():
    missing = [rel for rel in TIER_A if "isolation.isolated_env(" not in _src(rel)]
    assert not missing, f"these do not call isolated_env: {missing}"


def test_tier_a_sites_have_no_handrolled_env():
    offenders = [rel for rel in TIER_A if HANDROLLED.search(_src(rel))]
    assert not offenders, (
        f"these still build a subprocess env by hand: {offenders} — route them through "
        "isolation.isolated_env so no engram state var can be forgotten"
    )
