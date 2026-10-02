"""TDD for matrix.py's build_cfg_template(dst, warm=True): verify that skills are structurally
present in a WARM trial config.

Requirement (openspec/specs/eval-warm-config-fidelity/spec.md, extended by fix-eval-tooling-defects
#749): a WARM trial config SHALL be verified to carry the real recall/learn skill sources it
claims to grant. A missing or empty skill source SHALL raise, never silently proceed with fewer
skills than requested. matrix.py::build_cfg_template is a second, independent implementation of
this same contract as wrun.py::build_warm_cfg (dev/eval/traps/test_wrun.py) -- its own source path
(`REPO/skills/<skill>`) went stale at the 2026-07-24 agent-instructions/ move (#697) and its
`if os.path.isdir(src):` guard silently no-op'd on the missing source instead of raising.

Run: python3 -m pytest dev/eval/cumulative/test_matrix.py -v
"""
import os
import shutil
import tempfile
import unittest.mock as mock

import matrix


def test_build_cfg_template_warm_installs_real_recall_and_learn_skills():
    """Scenario: Successful skill installation is verified.

    WHEN build_cfg_template(dst, warm=True) is called with the REAL agent-instructions/skills
    sources THEN both recall and learn SKILL.md files exist under the destination's skills/ dir.
    """
    with tempfile.TemporaryDirectory() as tmp_dst:
        dst = os.path.join(tmp_dst, "cfg")
        matrix.build_cfg_template(dst, warm=True)

        recall_skill = os.path.join(dst, "skills", "recall", "SKILL.md")
        learn_skill = os.path.join(dst, "skills", "learn", "SKILL.md")

        assert os.path.exists(recall_skill), f"recall SKILL.md not found at {recall_skill}"
        assert os.path.exists(learn_skill), f"learn SKILL.md not found at {learn_skill}"
        assert os.path.getsize(recall_skill) > 0, "recall SKILL.md is empty"
        assert os.path.getsize(learn_skill) > 0, "learn SKILL.md is empty"


def test_build_cfg_template_warm_raises_on_missing_skill_source():
    """Scenario: Missing skill source raises.

    WHEN build_cfg_template(dst, warm=True) is called and a named skill's source directory does
    not exist at the expected path THEN the call raises, naming the specific missing skill and
    the path that was checked -- instead of silently skipping the copy as it did before #749.
    """
    with tempfile.TemporaryDirectory() as tmp_dst:
        dst = os.path.join(tmp_dst, "cfg")
        fake_repo = tempfile.mkdtemp()
        try:
            with mock.patch.object(matrix, "REPO", fake_repo):
                try:
                    matrix.build_cfg_template(dst, warm=True)
                    assert False, "expected an error for the missing skill source"
                except Exception as e:
                    error_msg = str(e)
                    assert "recall" in error_msg or "learn" in error_msg, (
                        f"error message should name the missing skill: {error_msg}"
                    )
                    assert "agent-instructions" in error_msg or "skills" in error_msg, (
                        f"error message should name the path checked: {error_msg}"
                    )
        finally:
            shutil.rmtree(fake_repo, ignore_errors=True)


def test_make_pools_warm_cfg_still_succeeds_end_to_end(tmp_path, monkeypatch):
    """Task 3.5: matrix.py's own CLI entry point (make_pools) must still succeed end-to-end
    against the real repo's real agent-instructions/skills/{recall,learn} sources, exercised
    against a throwaway temp CFGPOOL rather than the real /tmp/cummatrix."""
    monkeypatch.setattr(matrix, "CFGPOOL", str(tmp_path / "cfgpool"))
    pools = matrix.make_pools(1, 1)
    warm_dir = pools["warm"].get()
    cold_dir = pools["cold"].get()
    assert os.path.exists(os.path.join(warm_dir, "skills", "recall", "SKILL.md"))
    assert os.path.exists(os.path.join(warm_dir, "skills", "learn", "SKILL.md"))
    assert not os.path.isdir(os.path.join(cold_dir, "skills"))


def test_build_cfg_template_warm_raises_on_empty_skill_source():
    """Scenario: Empty skill source raises.

    WHEN build_cfg_template(dst, warm=True) is called and a named skill's source directory
    exists but contains no SKILL.md THEN the call raises, identifying the empty/invalid source.
    """
    with tempfile.TemporaryDirectory() as tmp_dst:
        dst = os.path.join(tmp_dst, "cfg")
        fake_repo = tempfile.mkdtemp()
        try:
            os.makedirs(os.path.join(fake_repo, "agent-instructions", "skills", "recall"))
            os.makedirs(os.path.join(fake_repo, "agent-instructions", "skills", "learn"))

            with mock.patch.object(matrix, "REPO", fake_repo):
                try:
                    matrix.build_cfg_template(dst, warm=True)
                    assert False, "expected an error for the empty skill source"
                except Exception as e:
                    error_msg = str(e)
                    assert "recall" in error_msg or "learn" in error_msg, (
                        f"error message should name the empty skill: {error_msg}"
                    )
                    assert "SKILL.md" in error_msg or "agent-instructions" in error_msg, (
                        f"error message should indicate an invalid/empty source: {error_msg}"
                    )
        finally:
            shutil.rmtree(fake_repo, ignore_errors=True)
