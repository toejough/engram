"""TDD for the recency-value scorers (#646 Task 1, Gate B fixes) + runner helpers.

score.py's pure functions gate the two-phase context-loss scenario. Surfacing is a DUAL metric
(Gate B FIX 2): surfaced_any serves the P2 vacuous-contrast gate (catches the re-rank bias
leaking the chunk as `direct`), surfaced_via_recency is the note-83 diagnostic (did the RECENCY
CHANNEL specifically deliver it). Both are validated against REAL payload shapes — the block-
scalar content real `engram query` emits, and the real golden dev/eval/cumulative/testdata/
recency_with_R.yaml — not the plan's synthetic (mis-indented) fixture.

build_trial_env / _build_ws2 / _ingest_padding are runner helpers; per-trial chunk-index
isolation is mandatory (LEDGER harder-regime-op-cost-unmeasurable, the contamination bug), and
WS2 is a strict whitelist (Gate B FIX 1) so no phase-1-authored source leaks the milli-dollar
formula into phase 2.
"""
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
import score
import recency_value as rv

GOLDEN = os.path.abspath(os.path.join(
    os.path.dirname(__file__), "..", "..", "cumulative", "testdata", "recency_with_R.yaml"))

# Real `engram query` shape: chunk content is a YAML block scalar (content: |-), items nested
# under items: at indent 2, provenances list entries at indent 6.
RECENT_HIT = (
    "version: 1\n"
    "items:\n"
    "  - path: /t/session.jsonl#Money unit discovery\n"
    "    kind: chunk\n"
    "    score: 0.11\n"
    "    provenances:\n"
    "      - recent\n"
    "    content: |-\n"
    "      While implementing import I found a milli-dollar unit: amt = round(dollars*1000).\n"
    "clusters: []\n"
)
# The re-rank leak: the milli-dollar chunk is boosted into the matched set as `direct`, NOT via
# the recency channel. surfaced_any catches it (P2 gate); surfaced_via_recency does not (note-83).
DIRECT_ONLY = (
    "version: 1\n"
    "items:\n"
    "  - path: /t/session.jsonl#Money unit discovery\n"
    "    kind: chunk\n"
    "    score: 0.42\n"
    "    provenances:\n"
    "      - direct\n"
    "    content: |-\n"
    "      While implementing import I found a milli-dollar unit: amt = round(dollars*1000).\n"
    "clusters: []\n"
)
EMPTY_PAYLOAD = "version: 1\nitems: []\n"


# --- report_revenue_ok -------------------------------------------------------------------

def test_report_revenue_ok_accepts_milli_dollar_math():
    assert score.report_revenue_ok("total revenue: $24.99\n", 24.99) is True


def test_report_revenue_ok_rejects_cents_misread():
    assert score.report_revenue_ok("total revenue: $2499.00\n", 24.99) is False   # amt read as cents => 100x


def test_report_revenue_ok_no_match_is_false():
    assert score.report_revenue_ok("build succeeded\n", 24.99) is False


# --- surfaced_any (P2 vacuous-contrast gate) ---------------------------------------------

def test_surfaced_any_true_for_recent_channel_hit():
    assert score.surfaced_any(RECENT_HIT) is True


def test_surfaced_any_true_for_direct_leak():
    # The re-rank bias can leak the chunk into the matched set as `direct` — the P2 gate must
    # still see it, else a vacuous contrast passes.
    assert score.surfaced_any(DIRECT_ONLY) is True


def test_surfaced_any_false_when_absent():
    assert score.surfaced_any(EMPTY_PAYLOAD) is False


# --- surfaced_via_recency (note-83 diagnostic) -------------------------------------------

def test_surfaced_via_recency_true_when_recent_chunk_mentions_unit():
    assert score.surfaced_via_recency(RECENT_HIT) is True


def test_surfaced_via_recency_false_for_direct_leak():
    # delivered via cosine (`direct`), not the recency channel — the diagnostic must NOT credit
    # the recency channel for it.
    assert score.surfaced_via_recency(DIRECT_ONLY) is False


def test_surfaced_via_recency_false_when_absent():
    assert score.surfaced_via_recency(EMPTY_PAYLOAD) is False


# --- provenance filter validated against the REAL golden (Gate B FIX 2) ------------------

def test_recency_filter_matches_recent_item_on_real_golden_shape():
    payload = open(GOLDEN).read()
    # The golden's recent-provenance item content mentions "handling errors"; the direct item's
    # content mentions "direct similarity". The provenance filter must credit the former (recent
    # channel) and not the latter (direct only).
    assert score._recent_block_has(payload, ["handling errors"]) is True
    assert score._recent_block_has(payload, ["direct similarity"]) is False


# --- recall_fired -------------------------------------------------------------------------

def test_recall_fired_detects_engram_query(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... engram query --phrase ..."}')
    assert score.recall_fired(str(p)) is True


def test_recall_fired_false_when_absent(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... wrote the report command ..."}')
    assert score.recall_fired(str(p)) is False


# --- phase1_used_learn (Gate B FIX 3 P1 check) -------------------------------------------

def test_phase1_used_learn_detects_engram_learn(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... engram learn --subject orders ..."}')
    assert score.phase1_used_learn(str(p)) is True


def test_phase1_used_learn_false_when_absent(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... built the importer, no memory writes ..."}')
    assert score.phase1_used_learn(str(p)) is False


# --- import_ok ------------------------------------------------------------------------------

def test_import_ok_true_for_correct_milli_dollar_conversion(tmp_path):
    csv_path = tmp_path / "orders.csv"
    csv_path.write_text("id,customer,dollars\n1,Acme,19.99\n2,Blue Sky,45.50\n")
    db_path = tmp_path / "orders.db.json"
    db_path.write_text(json.dumps([
        {"id": "1", "customer": "Acme", "amt": 19990},
        {"id": "2", "customer": "Blue Sky", "amt": 45500},
    ]))
    assert score.import_ok(str(db_path), str(csv_path)) is True


def test_import_ok_false_when_amt_left_in_raw_dollars(tmp_path):
    csv_path = tmp_path / "orders.csv"
    csv_path.write_text("id,customer,dollars\n1,Acme,19.99\n")
    db_path = tmp_path / "orders.db.json"
    db_path.write_text(json.dumps([{"id": "1", "customer": "Acme", "amt": 19.99}]))  # naive: no conversion
    assert score.import_ok(str(db_path), str(csv_path)) is False


def test_import_ok_false_when_amt_misread_as_cents(tmp_path):
    csv_path = tmp_path / "orders.csv"
    csv_path.write_text("id,customer,dollars\n1,Acme,19.99\n")
    db_path = tmp_path / "orders.db.json"
    db_path.write_text(json.dumps([{"id": "1", "customer": "Acme", "amt": 1999}]))   # cents, not milli-dollars
    assert score.import_ok(str(db_path), str(csv_path)) is False


# --- build_trial_env (Task 2 Step 1) -------------------------------------------------------

def test_build_trial_env_per_trial_unique_chunks_dir(tmp_path):
    env_a = rv.build_trial_env("on", str(tmp_path / "trial-a"))
    env_b = rv.build_trial_env("on", str(tmp_path / "trial-b"))
    assert env_a["ENGRAM_CHUNKS_DIR"] and env_b["ENGRAM_CHUNKS_DIR"]
    assert env_a["ENGRAM_CHUNKS_DIR"] != env_b["ENGRAM_CHUNKS_DIR"]
    assert env_a["ENGRAM_VAULT_PATH"]


def test_build_trial_env_recent_fill_absent_for_arm_on(tmp_path):
    env = rv.build_trial_env("on", str(tmp_path / "trial"))
    assert "ENGRAM_RECENT_FILL" not in env


def test_build_trial_env_recent_fill_off_for_arm_off(tmp_path):
    env = rv.build_trial_env("off", str(tmp_path / "trial"))
    assert env["ENGRAM_RECENT_FILL"] == "-1"


# --- _build_ws2 whitelist (Gate B FIX 1) -------------------------------------------------

def test_build_ws2_is_a_strict_whitelist(tmp_path):
    ws1 = tmp_path / "ws1"
    ws1.mkdir()
    # Phase-1 output + phase-1-authored source that MUST NOT leak (its importer contains the
    # literal round(dollars*1000) formula) + the withheld CSV.
    (ws1 / "orders.db.json").write_text(json.dumps([{"id": "1", "customer": "Acme", "amt": 19990}]))
    (ws1 / "orders.csv").write_text("id,customer,dollars\n1,Acme,19.99\n")
    (ws1 / "orders-cli").write_text("#!/usr/bin/env python3\namt = round(dollars * 1000)\n")
    (ws1 / "importer.py").write_text("def convert(d): return round(d * 1000)\n")

    ws2 = str(tmp_path / "ws2")
    rv._build_ws2(str(ws1), ws2)

    assert sorted(os.listdir(ws2)) == ["SPEC.md", "orders.db.json"]
    # orders.db.json is phase-1's actual output (opaque integer amt).
    assert json.load(open(os.path.join(ws2, "orders.db.json")))[0]["amt"] == 19990
    # SPEC.md is the fixture spec — it must not name the money unit anywhere.
    spec = open(os.path.join(ws2, "SPEC.md")).read().lower()
    assert "milli" not in spec and "1000" not in spec and "tenths" not in spec


def test_build_ws2_raises_if_unexpected_file_present(tmp_path):
    ws1 = tmp_path / "ws1"
    ws1.mkdir()
    (ws1 / "orders.db.json").write_text("[]")
    ws2 = tmp_path / "ws2"
    ws2.mkdir()
    # An extraneous file already present in ws2 must trip the defense-in-depth assertion.
    (ws2 / "leaked_importer.py").write_text("round(d*1000)")
    import pytest
    with pytest.raises(RuntimeError):
        rv._build_ws2(str(ws1), str(ws2))


# --- _ingest_padding (Gate B FIX 4) ------------------------------------------------------

def test_ingest_padding_builds_markdown_ingest_command(tmp_path, monkeypatch):
    calls = []

    class _FakeResult:
        returncode = 0
        stdout = ""
        stderr = ""

    def fake_run(cmd, **kwargs):
        calls.append(cmd)
        return _FakeResult()

    monkeypatch.setattr(rv.subprocess, "run", fake_run)

    chunks_dir = str(tmp_path / "chunks")
    os.makedirs(chunks_dir)
    rv._ingest_padding(chunks_dir, {"ENGRAM_CHUNKS_DIR": chunks_dir}, 3)

    assert len(calls) == 1
    cmd = calls[0]
    assert cmd[:2] == ["engram", "ingest"]
    assert "--chunks-dir" in cmd and chunks_dir in cmd
    assert cmd.count("--markdown") == 3
    # the three padding markdown files were actually written to disk before ingest
    md_paths = [cmd[i + 1] for i, tok in enumerate(cmd) if tok == "--markdown"]
    assert len(md_paths) == 3
    for path in md_paths:
        assert os.path.exists(path)
