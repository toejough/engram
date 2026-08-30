"""TDD for runbook probe: signal-fidelity, batched probing, and escalation."""
import json
import os
import sys
from unittest import mock

sys.path.insert(0, os.path.dirname(__file__))
import retrieval_probe as rp


# ============================================================================
# Signal-Fidelity Tests (Task 2.3)
# ============================================================================

def test_rank_in_payload_reads_items_not_candidate_l2s():
    """Core requirement: rank_in_payload MUST read items[] as the surfacing signal,
    never candidate_l2s (which is capped top-5-per-cluster L2-synthesis candidates).

    Scenario: target is present in items[] but absent from every cluster's candidate_l2s.
    The probe must still report surfaced=True, because items[] is the actual signal."""

    # Synthetic payload: target present in items[], missing from all candidate_l2s lists
    payload = {
        "items": [
            {"path": "v/other.md", "kind": "fact"},
            {"path": "123.target-slug.md", "kind": "fact"},  # Target at rank 2
            {"path": "v/third.md", "kind": "fact"},
        ],
        "clusters": [
            {
                # This cluster has members but candidate_l2s lists OTHER notes (no target)
                "members": [{"path": "123.target-slug.md"}, {"path": "v/other.md"}],
                "candidate_l2s": [
                    {"path": "v/other.md", "cosine": 0.95},
                    {"path": "v/third.md", "cosine": 0.90},
                    # target-slug deliberately absent — it's a cluster member but not top-5 by centroid
                ]
            }
        ]
    }

    # The new code MUST return surfaced=True because target is in items[]
    result = rp.rank_in_payload(payload, "target-slug")
    assert result["surfaced"] is True, (
        f"Expected surfaced=True (target in items[]), got {result}. "
        "rank_in_payload must read items[], not candidate_l2s."
    )
    assert result["rank"] == 2, f"Expected rank 2, got {result['rank']}"


def test_rank_in_payload_old_logic_control_arm():
    """Discriminating control: reproduce the OLD, wrong way of checking surfacing
    (searching candidate_l2s instead of items[]) to prove the fixture actually
    distinguishes old from new behavior."""

    # Same fixture as above
    payload = {
        "items": [
            {"path": "v/other.md", "kind": "fact"},
            {"path": "123.target-slug.md", "kind": "fact"},  # Target at rank 2 in items[]
            {"path": "v/third.md", "kind": "fact"},
        ],
        "clusters": [
            {
                "members": [{"path": "123.target-slug.md"}, {"path": "v/other.md"}],
                "candidate_l2s": [
                    {"path": "v/other.md", "cosine": 0.95},
                    {"path": "v/third.md", "cosine": 0.90},
                    # target deliberately missing from candidate_l2s
                ]
            }
        ]
    }

    # OLD (wrong) logic: search candidate_l2s membership
    def old_rank_logic(payload, target_basename):
        """Reproduction of the WRONG approach: checking candidate_l2s instead of items[]."""
        for cluster in payload.get("clusters", []):
            for candidate in cluster.get("candidate_l2s", []):
                path = candidate.get("path", "")
                basename = os.path.basename(path.split("#", 1)[0])
                if target_basename in basename:
                    return {"surfaced": True, "rank": None}
        return {"surfaced": False, "rank": None}

    # The old logic MUST report not surfaced (proving the fixture discriminates)
    old_result = old_rank_logic(payload, "target-slug")
    assert old_result["surfaced"] is False, (
        f"Old logic should report surfaced=False (target absent from candidate_l2s), "
        f"got {old_result}. Fixture does not discriminate."
    )

    # The new logic MUST report surfaced=True
    new_result = rp.rank_in_payload(payload, "target-slug")
    assert new_result["surfaced"] is True, (
        f"New logic should report surfaced=True (target in items[]), got {new_result}."
    )

    # This proves the fixture actually tests the difference
    assert old_result["surfaced"] != new_result["surfaced"], (
        "Fixture must discriminate between old (wrong) and new (correct) logic"
    )


# ============================================================================
# Batched Probe Tests (Task 2.1/2.2)
# ============================================================================

def test_probe_runbook_batched_success():
    """Batched probe: run one multi-phrase query, return rank result."""
    target = {
        "luhmann_id": "820",
        "slug": "skill-edits-validated",
        "phrasings": ["editing SKILL.md", "writing skill tests"]
    }

    mock_payload = {
        "items": [
            {"path": "v/other.md"},
            {"path": "820.skill-edits-validated.md"},  # Target at rank 2
            {"path": "v/third.md"},
        ]
    }

    with mock.patch("subprocess.run") as mock_run:
        mock_run.return_value = mock.Mock(
            returncode=0,
            stdout=json.dumps(mock_payload),
            stderr=""
        )
        with mock.patch("isolation.engram_env") as mock_env:
            mock_env.return_value = {}

            result = rp.probe_runbook_batched("/fake/vault", target)

            # Verify the command structure
            call_args = mock_run.call_args
            cmd = call_args[0][0]
            assert cmd[0] == "engram"
            assert cmd[1] == "query"
            assert "--phrase" in cmd
            assert "editing SKILL.md" in cmd
            assert "writing skill tests" in cmd

            # Verify result structure
            assert result["luhmann_id"] == "820"
            assert result["slug"] == "skill-edits-validated"
            assert result["surfaced"] is True
            assert result["rank"] == 2


def test_probe_runbook_batched_miss():
    """Batched probe: target missing from items[] reports surfaced=False."""
    target = {
        "luhmann_id": "821",
        "slug": "target-not-here",
        "phrasings": ["phrase 1", "phrase 2"]
    }

    mock_payload = {
        "items": [
            {"path": "v/other.md"},
            {"path": "v/different.md"},
        ]
    }

    with mock.patch("subprocess.run") as mock_run:
        mock_run.return_value = mock.Mock(
            returncode=0,
            stdout=json.dumps(mock_payload),
            stderr=""
        )
        with mock.patch("isolation.engram_env") as mock_env:
            mock_env.return_value = {}

            result = rp.probe_runbook_batched("/fake/vault", target)

            assert result["luhmann_id"] == "821"
            assert result["slug"] == "target-not-here"
            assert result["surfaced"] is False
            assert result["rank"] is None


def test_probe_runbook_batched_subprocess_fails():
    """Batched probe: fail loud on non-zero exit."""
    target = {
        "luhmann_id": "820",
        "slug": "skill-edits",
        "phrasings": ["editing"]
    }

    with mock.patch("subprocess.run") as mock_run:
        mock_run.return_value = mock.Mock(
            returncode=1,
            stdout="",
            stderr="query failed"
        )
        with mock.patch("isolation.engram_env") as mock_env:
            mock_env.return_value = {}

            try:
                rp.probe_runbook_batched("/fake/vault", target)
                assert False, "Should have raised RuntimeError"
            except RuntimeError as e:
                assert "engram query failed" in str(e)
                assert "820" in str(e)


# ============================================================================
# Escalation Tests (Task 2.4)
# ============================================================================

def test_probe_runbook_escalation_per_phrase():
    """Escalation: run isolated per-phrase queries for a target that missed batched."""
    target = {
        "luhmann_id": "820",
        "slug": "skill-edits",
        "phrasings": ["phrase 1", "phrase 2", "phrase 3"]
    }

    # Each phrasing gets its own payload
    payloads = [
        {"items": [{"path": "v/other.md"}]},  # phrase 1: miss
        {"items": [{"path": "820.skill-edits.md"}]},  # phrase 2: hit at rank 1
        {"items": [{"path": "v/unrelated.md"}]},  # phrase 3: miss
    ]

    call_count = [0]
    def mock_run_side_effect(cmd, **kwargs):
        result = call_count[0]
        call_count[0] += 1
        return mock.Mock(
            returncode=0,
            stdout=json.dumps(payloads[result]),
            stderr=""
        )

    with mock.patch("subprocess.run") as mock_run:
        mock_run.side_effect = mock_run_side_effect
        with mock.patch("isolation.engram_env") as mock_env:
            mock_env.return_value = {}

            result = rp.probe_runbook_escalation("/fake/vault", target)

            # Verify structure
            assert result["luhmann_id"] == "820"
            assert result["slug"] == "skill-edits"
            assert "per_phrasing" in result
            assert len(result["per_phrasing"]) == 3

            # Verify per-phrasing results
            assert result["per_phrasing"][0]["phrasing"] == "phrase 1"
            assert result["per_phrasing"][0]["surfaced"] is False

            assert result["per_phrasing"][1]["phrasing"] == "phrase 2"
            assert result["per_phrasing"][1]["surfaced"] is True
            assert result["per_phrasing"][1]["rank"] == 1

            assert result["per_phrasing"][2]["phrasing"] == "phrase 3"
            assert result["per_phrasing"][2]["surfaced"] is False

            # Verify 3 separate subprocess calls (one per phrasing)
            assert mock_run.call_count == 3


def test_probe_all_runbooks_loads_corpus():
    """Load corpus and probe all entries."""
    # Mock load_runbook_corpus to return a small test corpus
    test_corpus = [
        {"luhmann_id": "1", "slug": "note-1", "phrasings": ["p1"]},
        {"luhmann_id": "2", "slug": "note-2", "phrasings": ["p2"]},
    ]

    with mock.patch("retrieval_probe.load_runbook_corpus") as mock_load:
        mock_load.return_value = test_corpus
        with mock.patch("retrieval_probe.probe_runbook_batched") as mock_probe:
            mock_probe.side_effect = [
                {"luhmann_id": "1", "slug": "note-1", "surfaced": True, "rank": 1},
                {"luhmann_id": "2", "slug": "note-2", "surfaced": False, "rank": None},
            ]

            result = rp.probe_all_runbooks("/fake/vault")

            assert len(result) == 2
            assert result[0]["luhmann_id"] == "1"
            assert result[1]["luhmann_id"] == "2"
            mock_load.assert_called_once()
            assert mock_probe.call_count == 2
