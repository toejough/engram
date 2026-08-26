"""TDD for wrun.py build_warm_cfg: verify that skills are structurally present in WARM trials.

Requirement (spec.md): A WARM trial config SHALL be verified to contain the real recall/learn
skill sources it claims to grant. Missing or empty skill sources SHALL raise, never silently
degrade to a cold config.

Run: python3 test_wrun.py  (or python3 -m pytest test_wrun.py -q)
"""
import os
import shutil
import sys
import tempfile
import unittest.mock as mock

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import wrun


def test_successful_skill_installation_verified():
    """Scenario: Successful skill installation is verified.

    WHEN build_warm_cfg is called with the REAL agent-instructions/skills sources
    THEN both recall and learn SKILL.md files exist at the destination
    """
    with tempfile.TemporaryDirectory() as tmp_dst:
        wrun.build_warm_cfg(tmp_dst)

        recall_skill = os.path.join(tmp_dst, "skills", "recall", "SKILL.md")
        learn_skill = os.path.join(tmp_dst, "skills", "learn", "SKILL.md")

        assert os.path.exists(recall_skill), f"recall SKILL.md not found at {recall_skill}"
        assert os.path.exists(learn_skill), f"learn SKILL.md not found at {learn_skill}"
        assert os.path.getsize(recall_skill) > 0, f"recall SKILL.md is empty"
        assert os.path.getsize(learn_skill) > 0, f"learn SKILL.md is empty"


def test_missing_skill_source_raises():
    """Scenario: Missing skill source raises.

    WHEN build_warm_cfg is called and a skill source directory does not exist
    THEN the call raises RuntimeError naming the missing skill and path
    """
    with tempfile.TemporaryDirectory() as tmp_dst:
        # Monkeypatch REPO to point to a directory where skills/ subdirectory doesn't exist
        # (but we'll make sure agent-instructions/skills doesn't resolve correctly)
        fake_repo = tempfile.mkdtemp()
        try:
            with mock.patch.object(wrun, 'REPO', fake_repo):
                try:
                    wrun.build_warm_cfg(tmp_dst)
                    assert False, "Expected RuntimeError to be raised for missing skill source"
                except RuntimeError as e:
                    error_msg = str(e)
                    # Should mention either 'recall' or 'learn' and include 'agent-instructions/skills'
                    assert 'recall' in error_msg or 'learn' in error_msg, \
                        f"Error message should mention skill name: {error_msg}"
                    assert 'agent-instructions' in error_msg or 'skills' in error_msg, \
                        f"Error message should mention path: {error_msg}"
        finally:
            shutil.rmtree(fake_repo, ignore_errors=True)


def test_empty_skill_source_raises():
    """Scenario: Empty skill source raises.

    WHEN build_warm_cfg is called and a skill source directory exists but is empty (no SKILL.md)
    THEN the call raises RuntimeError identifying the empty skill source
    """
    with tempfile.TemporaryDirectory() as tmp_dst:
        fake_repo = tempfile.mkdtemp()
        try:
            # Create empty skill directories (exist but no SKILL.md)
            os.makedirs(os.path.join(fake_repo, "agent-instructions", "skills", "recall"))
            os.makedirs(os.path.join(fake_repo, "agent-instructions", "skills", "learn"))

            with mock.patch.object(wrun, 'REPO', fake_repo):
                try:
                    wrun.build_warm_cfg(tmp_dst)
                    assert False, "Expected RuntimeError to be raised for empty skill source"
                except RuntimeError as e:
                    error_msg = str(e)
                    # Should mention the skill name and empty/missing SKILL.md
                    assert 'recall' in error_msg or 'learn' in error_msg, \
                        f"Error message should mention skill name: {error_msg}"
                    assert 'agent-instructions' in error_msg or 'SKILL.md' in error_msg, \
                        f"Error message should indicate invalid/empty source: {error_msg}"
        finally:
            shutil.rmtree(fake_repo, ignore_errors=True)


if __name__ == "__main__":
    tests = [
        test_successful_skill_installation_verified,
        test_missing_skill_source_raises,
        test_empty_skill_source_raises,
    ]
    for test_fn in tests:
        try:
            test_fn()
            print(f"PASS {test_fn.__name__}")
        except AssertionError as e:
            print(f"FAIL {test_fn.__name__}: {e}")
            sys.exit(1)
    print(f"\n{len(tests)} passed")
