"""Unit tests for lever_recheck/stub_engram.py's query ranking (offline — env + capsys, no LLM).

The load-bearing behavior: the stub reproduces MEASURED retrieval reality (its own docstring cites
it — note 80: rank-1 under lever phrasing, 0 hits under diagnostic phrasing). So a lever-keyed
query must return the buried note FIRST at the TOP score (the matched-note floor), and a
non-lever-keyed query must exclude it entirely. The pre-fix stub emitted the buried note LAST at
the LOWEST score on keyed queries (filename-ordered ladder), which made GREEN-run agents honestly
miss the bottom-ranked disproof — an instrument artifact, not a synthesis failure."""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "lever_recheck"))

import stub_engram  # noqa: E402

BURIED = "8.2026-06-20.closed-lever-note"
DISTRACTORS = ["1.2026-06-20.alpha-note", "2.2026-06-20.beta-note", "3.2026-06-20.gamma-note"]


def _make_vault(tmp_path):
    vault = tmp_path / "vault"
    vault.mkdir(exist_ok=True)  # some tests query the same vault twice
    for base in DISTRACTORS:
        (vault / f"{base}.md").write_text(f"# {base}\ndistractor body\n")
    (vault / f"{BURIED}.md").write_text("# closed lever\ntried; rolled back\n")
    return str(vault)


def _query(tmp_path, monkeypatch, capsys, phrase, lever_terms="closed,lever"):
    log = tmp_path / "stub_log.jsonl"
    monkeypatch.setenv("ENGRAM_VAULT_PATH", _make_vault(tmp_path))
    monkeypatch.setenv("STUB_ENGRAM_BURIED", BURIED)
    monkeypatch.setenv("STUB_ENGRAM_LEVER_TERMS", lever_terms)
    monkeypatch.setenv("STUB_ENGRAM_LOG", str(log))
    rc = stub_engram.main(["query", "--phrase", phrase])
    assert rc == 0
    payload = capsys.readouterr().out
    log_rows = [json.loads(line) for line in log.read_text().splitlines() if line.strip()]
    return _parse_items(payload), log_rows


def _parse_items(payload):
    """Parse the payload's top-level items (exact 2/4-space prefixes; stop at clusters, whose
    members use deeper indentation)."""
    items = []
    for line in payload.splitlines():
        if line == "clusters:":
            break
        if line.startswith("  - path: "):
            items.append({"path": line[len("  - path: "):]})
        elif line.startswith("    score: ") and items and "score" not in items[-1]:
            items[-1]["score"] = float(line[len("    score: "):])
    return items


def test_lever_keyed_query_returns_buried_note_first_at_top_score(tmp_path, monkeypatch, capsys):
    items, _ = _query(tmp_path, monkeypatch, capsys, "the closed lever we might retry")
    assert items[0]["path"].startswith(BURIED)                       # rank-1, per measured reality
    assert items[0]["score"] == max(it["score"] for it in items)     # top score, above the ladder
    assert len(items) == len(DISTRACTORS) + 1


def test_lever_keyed_query_keeps_distractor_order_after_the_buried_note(tmp_path, monkeypatch, capsys):
    items, _ = _query(tmp_path, monkeypatch, capsys, "the closed lever we might retry")
    tail_paths = [it["path"] for it in items[1:]]
    assert tail_paths == [f"{b}.md" for b in DISTRACTORS]            # filename order preserved
    tail_scores = [it["score"] for it in items[1:]]
    assert tail_scores == sorted(tail_scores, reverse=True)          # descending ladder preserved


def test_non_lever_keyed_query_excludes_buried_and_keeps_order(tmp_path, monkeypatch, capsys):
    items, _ = _query(tmp_path, monkeypatch, capsys, "general diagnostic ask")
    assert all(not it["path"].startswith(BURIED) for it in items)    # the measured retrieval miss
    assert [it["path"] for it in items] == [f"{b}.md" for b in DISTRACTORS]
    scores = [it["score"] for it in items]
    assert scores == sorted(scores, reverse=True)


def test_returned_buried_logging_unchanged(tmp_path, monkeypatch, capsys):
    _, log_keyed = _query(tmp_path, monkeypatch, capsys, "the closed lever we might retry")
    assert log_keyed[-1]["lever_keyed"] is True
    assert log_keyed[-1]["returned_buried"] is True
    _, log_plain = _query(tmp_path, monkeypatch, capsys, "general diagnostic ask")
    assert log_plain[-1]["lever_keyed"] is False
    assert log_plain[-1]["returned_buried"] is False
