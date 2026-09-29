"""Tests for parent_capture.py -- fixture transcripts and a fixture git vault only;
no API calls, no reads of the real vault or ~/.claude, writes only under tmp_path."""

import json
import os
import subprocess
import sys
from datetime import datetime, timezone

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import parent_capture as pc


# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------


def rec(kind, ts, content, **extra):
    r = {"type": kind, "timestamp": ts, "message": {"content": content}}
    r.update(extra)
    return r


def tool_use(tid, name, inp):
    return {"type": "tool_use", "id": tid, "name": name, "input": inp}


def tool_result(tid, text):
    return {"type": "tool_result", "tool_use_id": tid, "content": [{"type": "text", "text": text}]}


def notification(tool_use_id, result, summary='Agent "do x" finished'):
    body = (f"<task-notification>\n<task-id>a1</task-id>\n<tool-use-id>{tool_use_id}</tool-use-id>\n"
            f"<status>completed</status>\n<summary>{summary}</summary>\n")
    if result is not None:
        body += f"<result>{result}</result>"
    return body + "\n</task-notification>"


def write_jsonl(path, records):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("".join(json.dumps(r) + "\n" for r in records))
    return path


def dt(s):
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


# ---------------------------------------------------------------------------
# LESSONS block extraction
# ---------------------------------------------------------------------------


def test_lessons_block_takes_last_line_start_marker():
    payload = "Summary.\nLESSONS: quoted earlier\nMore text\nLESSONS: the real one\ntrailing line"
    assert pc.lessons_block(payload) == "the real one\ntrailing line"


@pytest.mark.parametrize("line", [
    "**LESSONS:** use exec in stubs",
    "**LESSONS**: use exec in stubs",
    "- LESSONS: use exec in stubs",
    "LESSONS:use exec in stubs",
])
def test_lessons_block_accepts_emphasis_and_marker_variants(line):
    assert pc.lessons_block("report\n" + line) == "use exec in stubs"


def test_lessons_block_ignores_mid_line_mentions():
    assert pc.lessons_block("The `LESSONS:` contract was never met.\nDone.") is None


def test_lessons_block_strips_usage_trailer():
    payload = "x\nLESSONS: a lesson\nagentId: abc (for resuming)\n<usage>tokens: 5</usage>"
    assert pc.lessons_block(payload) == "a lesson"


@pytest.mark.parametrize("block,expected", [
    ("none", True),
    ("**none**", True),
    ("None — read-only task", True),
    ("none new to capture — mechanical", True),
    ("nonetheless check X before Y", False),
    ("No new lesson crystallized", False),
    ("Use exec", False),
])
def test_is_none(block, expected):
    assert pc.is_none(block) is expected


# ---------------------------------------------------------------------------
# item splitting
# ---------------------------------------------------------------------------


def test_split_items_inline_block_is_one_item_even_with_commas():
    assert pc.split_items("Use exec, not sleep, in stubs") == ["Use exec, not sleep, in stubs"]


def test_split_items_bullets_with_continuations_and_header_preamble():
    block = "\n\n- first lesson\n  continues here\n- second lesson\n"
    assert pc.split_items(block) == ["first lesson continues here", "second lesson"]
    block2 = "From note 1001:\n1. **one** thing\n2) two thing"
    assert pc.split_items(block2) == ["**one** thing", "two thing"]


def test_split_items_keeps_non_header_preamble_as_item():
    assert pc.split_items("Overall lesson here.\n- detail a") == ["Overall lesson here.", "detail a"]


def test_split_items_paragraph_block_is_one_item():
    assert pc.split_items("\n\nMulti-trial plans must document fairness.\nAnd cost.") == [
        "Multi-trial plans must document fairness. And cost."
    ]


def test_normalize_collapses_markdown_case_and_space():
    assert pc.normalize("**Use**  `exec`\n in Stubs") == pc.normalize("use exec in stubs")


# ---------------------------------------------------------------------------
# receipts + items from a parent transcript
# ---------------------------------------------------------------------------


def parent_fixture(tmp_path):
    records = [
        rec("user", "2026-09-10T10:00:00Z", "start"),
        rec("assistant", "2026-09-10T10:01:00Z", [tool_use("tu1", "Agent", {"description": "build x"})]),
        rec("user", "2026-09-10T10:01:01Z", [tool_result("tu1", "Async agent launched successfully.\nagentId: a1")]),
        rec("user", "2026-09-10T10:05:00Z", notification("tu1", "Done.\nLESSONS: use exec in stubs")),
        # same lesson again via TaskOutput later -> deduped, earliest kept
        rec("assistant", "2026-09-10T10:06:00Z", [tool_use("tu2", "TaskOutput", {"task_id": "a1"})]),
        rec("user", "2026-09-10T10:06:01Z", [tool_result(
            "tu2", "<status>completed</status>\n<output>\nDone.\n**LESSONS:** Use exec in stubs\n</output>")]),
        # a none receipt
        rec("assistant", "2026-09-10T11:00:00Z", [tool_use("tu3", "Agent", {"description": "review"})]),
        rec("user", "2026-09-10T11:00:05Z", [tool_result("tu3", "Review ok.\nLESSONS: none")]),
        # background Bash notification: not a subagent -> ignored
        rec("assistant", "2026-09-10T11:10:00Z", [tool_use("tu4", "Bash", {"command": "sleep 1"})]),
        rec("user", "2026-09-10T11:20:00Z", notification("tu4", "LESSONS: fake", summary="Background command done")),
        # a Read tool_result quoting a LESSONS line: ignored
        rec("assistant", "2026-09-10T11:30:00Z", [tool_use("tu5", "Read", {"file_path": "x"})]),
        rec("user", "2026-09-10T11:30:01Z", [tool_result("tu5", "LESSONS: from a file")]),
        # multi-item sync result
        rec("assistant", "2026-09-10T12:00:00Z", [tool_use("tu6", "Task", {"description": "fix y"})]),
        rec("user", "2026-09-10T12:00:09Z", [tool_result("tu6", "Fixed.\nLESSONS:\n- lesson A\n- lesson B")]),
        rec("assistant", "2026-09-11T09:00:00Z", [{"type": "text", "text": "bye"}]),
    ]
    return write_jsonl(tmp_path / "proj" / "sess1.jsonl", records)


def test_extract_items_from_parent(tmp_path):
    path = parent_fixture(tmp_path)
    out = pc.extract_parent(str(path))
    texts = [i["text"] for i in out["items"]]
    assert texts == ["use exec in stubs", "lesson A", "lesson B"]
    first = out["items"][0]
    assert first["t_ret"] == "2026-09-10T10:05:00Z"
    assert first["dispatch"] == "build x"
    assert out["receipts_with_lessons"] == 4  # tu1 notif, tu2 TaskOutput, tu3 none, tu6
    assert out["receipts_none"] == 1
    assert out["t_end"] == "2026-09-11T09:00:00Z"
    assert out["session_id"] == "sess1"


def test_extract_ignores_notification_without_result(tmp_path):
    records = [
        rec("assistant", "2026-09-10T10:01:00Z", [tool_use("tu1", "Agent", {"description": "d"})]),
        rec("user", "2026-09-10T10:05:00Z", notification("tu1", None)),
    ]
    path = write_jsonl(tmp_path / "p" / "s.jsonl", records)
    assert pc.extract_parent(str(path))["items"] == []


# ---------------------------------------------------------------------------
# learn invocations (C0)
# ---------------------------------------------------------------------------


def test_learn_invocations_skill_and_slash(tmp_path):
    records = [
        rec("assistant", "2026-09-10T10:00:00Z", [tool_use("s1", "Skill", {"skill": "learn", "args": "mid-cycle fast path"})]),
        rec("assistant", "2026-09-10T10:00:00Z", [tool_use("s2", "Skill", {"skill": "recall"})]),
        rec("user", "2026-09-11T10:00:00Z", "<command-name>/learn</command-name>\n<command-args></command-args>"),
    ]
    path = write_jsonl(tmp_path / "p" / "s.jsonl", records)
    inv = pc.learn_invocations(str(path))
    assert [(i["ts"], i["mid_cycle"]) for i in inv] == [
        ("2026-09-10T10:00:00Z", True), ("2026-09-11T10:00:00Z", False)]


def test_closing_learn_flags():
    inv = [{"ts": "2026-09-10T10:00:00Z", "mid_cycle": True}, {"ts": "2026-09-12T10:00:00Z", "mid_cycle": True}]
    assert pc.closing_learn(inv, "2026-09-11T00:00:00Z") == {"any": True, "non_mid_cycle": False}
    assert pc.closing_learn(inv, "2026-09-13T00:00:00Z") == {"any": False, "non_mid_cycle": False}


# ---------------------------------------------------------------------------
# learn-call index (exact write times)
# ---------------------------------------------------------------------------


def test_learn_call_index_collects_vault_paths(tmp_path):
    vault = "/v/vault"
    records = [
        rec("assistant", "2026-09-10T10:00:00Z", [tool_use("b1", "Bash", {"command": "engram learn feedback --slug x"})]),
        rec("user", "2026-09-10T10:00:03Z", [tool_result("b1", f"{vault}/955.2026-09-10.x.md\n{vault}/955.2026-09-10.x.vec.json")]),
        rec("assistant", "2026-09-10T10:01:00Z", [tool_use("b2", "Bash", {"command": "cat notes | grep 'engram learn'"})]),
        rec("user", "2026-09-10T10:01:03Z", [tool_result("b2", f"{vault}/1.2026-01-01.old.md")]),
        rec("assistant", "2026-09-10T10:02:00Z", [tool_use("b3", "Bash", {"command": "cd /x && engram amend 12 --action y"})]),
        rec("user", "2026-09-10T10:02:03Z", [tool_result("b3", f"{vault}/12.2026-09-01.y.md")]),
        rec("assistant", "2026-09-10T10:03:00Z", [tool_use("b4", "Bash", {"command": "engram learn fact"})]),
        rec("user", "2026-09-10T10:03:03Z", [tool_result("b4", "/other/vault/9.2026-09-10.z.md")]),
    ]
    path = write_jsonl(tmp_path / "p" / "s.jsonl", records)
    idx = pc.learn_call_index([str(path)], vault)
    assert set(idx) == {"955.2026-09-10.x.md", "12.2026-09-01.y.md"}
    assert idx["955.2026-09-10.x.md"]["t_w"] == "2026-09-10T10:00:03Z"
    assert idx["955.2026-09-10.x.md"]["transcript"] == str(path)
    assert "engram learn feedback" in idx["955.2026-09-10.x.md"]["command"]


def test_learn_call_index_keeps_earliest(tmp_path):
    vault = "/v/vault"
    a = write_jsonl(tmp_path / "a.jsonl", [
        rec("assistant", "2026-09-10T12:00:00Z", [tool_use("b1", "Bash", {"command": "engram learn fact"})]),
        rec("user", "2026-09-10T12:00:01Z", [tool_result("b1", f"{vault}/5.2026-09-10.x.md")])])
    b = write_jsonl(tmp_path / "b.jsonl", [
        rec("assistant", "2026-09-10T09:00:00Z", [tool_use("b1", "Bash", {"command": "engram learn fact"})]),
        rec("user", "2026-09-10T09:00:01Z", [tool_result("b1", f"{vault}/5.2026-09-10.x.md")])])
    idx = pc.learn_call_index([str(a), str(b)], vault)
    assert idx["5.2026-09-10.x.md"]["t_w"] == "2026-09-10T09:00:01Z"


# ---------------------------------------------------------------------------
# vault note set (fixture git repo)
# ---------------------------------------------------------------------------


NOTE = """---
type: feedback
situation: {sit}
action: {act}
luhmann: "{lid}"
created: "{created}"
---

Body of {lid}.
"""


def git(repo, *args):
    subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True,
                   env={**os.environ, "GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t",
                        "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t"})


def test_note_set_includes_head_deleted_and_transcript_only(tmp_path):
    repo = tmp_path / "vault"
    repo.mkdir()
    git(repo, "init", "-q")
    (repo / "1.2026-09-01.kept.md").write_text(NOTE.format(sit="s1", act="a1", lid="1", created="2026-09-01"))
    (repo / "2.2026-09-02.gone.md").write_text(NOTE.format(sit="s2", act="a2", lid="2", created="2026-09-02"))
    (repo / "2.2026-09-02.gone.vec.json").write_text("{}")
    git(repo, "add", "-A")
    git(repo, "commit", "-q", "-m", "one")
    (repo / "2.2026-09-02.gone.md").unlink()
    git(repo, "add", "-A")
    git(repo, "commit", "-q", "-m", "two")
    idx = {"3.2026-09-10.lost.md": {"t_w": "2026-09-10T10:00:00Z", "transcript": "/t.jsonl",
                                     "command": "engram learn feedback --situation 'when x' --action 'do y'"},
           "1.2026-09-01.kept.md": {"t_w": "2026-09-01T10:00:00Z", "transcript": "/t.jsonl", "command": "c"}}
    notes = {n["file"]: n for n in pc.note_set(str(repo), idx)}
    assert set(notes) == {"1.2026-09-01.kept.md", "2.2026-09-02.gone.md", "3.2026-09-10.lost.md"}
    assert notes["1.2026-09-01.kept.md"]["source"] == "head"
    assert notes["1.2026-09-01.kept.md"]["situation"] == "s1"
    assert notes["1.2026-09-01.kept.md"]["t_w_exact"] == "2026-09-01T10:00:00Z"
    assert notes["2.2026-09-02.gone.md"]["source"] == "deleted"
    assert notes["2.2026-09-02.gone.md"]["action"] == "a2"
    assert notes["2.2026-09-02.gone.md"]["t_w_exact"] is None
    assert notes["2.2026-09-02.gone.md"]["created"] == "2026-09-02"
    assert notes["3.2026-09-10.lost.md"]["source"] == "transcript_only"
    assert "when x" in notes["3.2026-09-10.lost.md"]["body"]
    assert notes["3.2026-09-10.lost.md"]["note_id"] == "3"


def test_note_set_dedupes_renamed_note_by_slug(tmp_path):
    repo = tmp_path / "vault"
    repo.mkdir()
    git(repo, "init", "-q")
    (repo / "7.2026-09-05.same-slug.md").write_text(NOTE.format(sit="old", act="a", lid="7", created="2026-09-05"))
    git(repo, "add", "-A")
    git(repo, "commit", "-q", "-m", "one")
    git(repo, "mv", "7.2026-09-05.same-slug.md", "7a.2026-09-05.same-slug.md")
    git(repo, "commit", "-q", "-m", "rename")
    files = [n["file"] for n in pc.note_set(str(repo), {})]
    assert files == ["7a.2026-09-05.same-slug.md"]


# ---------------------------------------------------------------------------
# candidate windows
# ---------------------------------------------------------------------------


def note(file, created, t_w=None):
    return {"file": file, "note_id": file.split(".")[0], "created": created, "t_w_exact": t_w,
            "slug": file.split(".", 2)[2], "situation": "", "action": "", "body": ""}


def test_classify_note_timing():
    t_ret, t_end = "2026-09-10T15:00:00Z", "2026-09-12T15:00:00Z"  # local 11:00 on 09-10 / 09-12
    c = lambda n: pc.classify_timing(n, t_ret, t_end)
    assert c(note("a.2026-09-10.a.md", "2026-09-10", "2026-09-10T16:00:00Z")) == "post"
    assert c(note("b.2026-09-10.b.md", "2026-09-10", "2026-09-10T14:00:00Z")) == "pre"
    assert c(note("c.2026-09-13.c.md", "2026-09-13", "2026-09-13T14:59:00Z")) == "post"
    assert c(note("d.2026-09-13.d.md", "2026-09-13", "2026-09-13T15:01:00Z")) is None
    assert c(note("e.2026-09-10.e.md", "2026-09-10")) == "post_same_day_ambiguous"
    assert c(note("f.2026-09-13.f.md", "2026-09-13")) == "post"
    assert c(note("g.2026-09-14.g.md", "2026-09-14")) is None
    assert c(note("h.2026-09-09.h.md", "2026-09-09")) == "pre"


def test_local_date_uses_new_york():
    assert pc.local_date("2026-09-10T02:00:00Z") == "2026-09-09"
    assert pc.local_date("2026-09-10T05:00:00Z") == "2026-09-10"


def test_tfidf_top_k_ranks_lexical_match_first():
    notes = [
        {"file": "1.x.a.md", "slug": "stub-exec", "situation": "writing stub binaries in shell tests",
         "action": "use exec so the stub is killed with its pipes", "body": ""},
        {"file": "2.x.b.md", "slug": "vault", "situation": "curating vault notes", "action": "merge", "body": ""},
        {"file": "3.x.c.md", "slug": "timeouts", "situation": "budgeting timeouts", "action": "table", "body": ""},
    ]
    ranked = pc.top_k_lexical("stub binaries in shell tests should use exec", notes, k=2)
    assert [n["file"] for n in ranked] == ["1.x.a.md", ranked[1]["file"]]
    assert len(ranked) == 2


# ---------------------------------------------------------------------------
# judge parsing, batching, outcomes, metrics, cost
# ---------------------------------------------------------------------------


def test_parse_json_reply_handles_fences_and_prose():
    assert pc.parse_json_reply('Here:\n```json\n[{"id": "1"}]\n```') == [{"id": "1"}]
    assert pc.parse_json_reply('{"a": []}') == {"a": []}
    with pytest.raises(ValueError):
        pc.parse_json_reply("no json here")


def test_batches_respect_size_and_grouping():
    items = [{"id": f"i{n}", "session_id": "s" if n < 12 else "t"} for n in range(14)]
    batches = pc.batch(items, 10, key=lambda i: i["session_id"])
    assert [len(b) for b in batches] == [10, 2, 2]


def test_outcome_precedence():
    assert pc.outcome(matched_pre=True, matched_post=True) == "covered"
    assert pc.outcome(matched_pre=False, matched_post=True) == "captured"
    assert pc.outcome(matched_pre=False, matched_post=False) == "lost"


def test_metrics_compute_c1_variants():
    items = (
        [{"worth": True, "outcome": "covered", "ambiguous_only": False}] * 2
        + [{"worth": True, "outcome": "captured", "ambiguous_only": False}] * 3
        + [{"worth": True, "outcome": "captured", "ambiguous_only": True}] * 1
        + [{"worth": True, "outcome": "lost", "ambiguous_only": False}] * 4
        + [{"worth": False, "outcome": None, "ambiguous_only": False}] * 5
    )
    m = pc.compute_metrics(items)
    assert m["C1"]["numerator"] == 6 and m["C1"]["denominator"] == 10 and m["C1"]["rate_pct"] == 60.0
    assert m["C1"]["ci95"]["lo_pct"] is not None
    assert m["C1_new"]["numerator"] == 4 and m["C1_new"]["denominator"] == 8
    assert m["C1_strict"]["numerator"] == 5
    assert m["worth"]["numerator"] == 10 and m["worth"]["denominator"] == 15


def test_project_cost_uses_fitted_model_and_margin():
    est = pc.project_cost([1000, 3000])
    base = 2 * 0.0464 + 0.00205 * 4
    assert est == pytest.approx(base * 1.25)


def test_refuses_frozen_results_dir(tmp_path):
    with pytest.raises(SystemExit):
        pc.check_output_dir(pc.run_audit.FROZEN_RESULTS_DIR / "x")


def test_run_judges_end_to_end_with_stub(tmp_path):
    parent = str(tmp_path / "proj" / "s1.jsonl")
    notes = [
        {**note("10.2026-09-09.old.md", "2026-09-09"), "note_id": "10"},
        {**note("20.2026-09-10.new.md", "2026-09-10", "2026-09-10T16:00:00Z"), "note_id": "20",
         "writer_transcript": str(tmp_path / "proj" / "s1" / "subagents" / "agent-x.jsonl")},
    ]
    base = {"session_id": "s1", "transcript": parent, "t_ret": "2026-09-10T15:00:00Z",
            "t_end": "2026-09-10T20:00:00Z", "dispatch": "d",
            "post_candidates": ["20.2026-09-10.new.md"], "coverage_candidates": ["10.2026-09-09.old.md"]}
    items = [{**base, "id": f"s1-00{n}", "text": f"lesson {n}", "norm": f"lesson {n}"} for n in (1, 2, 3, 4)]
    prep = {"items": items, "notes": notes, "pop": {"parents": []}}

    def stub(prompt, model):
        assert model == "sonnet"
        if prompt.startswith("You are applying"):
            return {"result": json.dumps([
                {"id": "s1-001", "worth_capturing": True, "kind": 1, "reason": "r"},
                {"id": "s1-002", "worth_capturing": True, "kind": 4, "reason": "r"},
                {"id": "s1-003", "worth_capturing": True, "kind": 3, "reason": "r"},
                {"id": "s1-004", "worth_capturing": False, "kind": None, "reason": "r"}]),
                "total_cost_usd": 0.1}
        return {"result": "```json\n" + json.dumps({
            "s1-001": [{"note_id": "10", "reason": "x"}, {"note_id": "20", "reason": "y"}],
            "s1-002": [{"note_id": "20", "reason": "y"}],
            "s1-003": []}) + "\n```", "total_cost_usd": 0.2}

    out = pc.run_judges(prep, tmp_path, stub)
    by = {i["id"]: i for i in out}
    assert by["s1-001"]["outcome"] == "covered"
    assert by["s1-002"]["outcome"] == "captured"
    assert by["s1-002"]["attribution"] == "parent_tree"
    assert by["s1-003"]["outcome"] == "lost"
    assert by["s1-004"]["outcome"] is None
    costs = [json.loads(l)["cost_usd"] for l in open(tmp_path / "cost-log.jsonl")]
    assert costs == [0.1, 0.2]
    # resume: a second pass makes no new calls
    pc.run_judges(prep, tmp_path, lambda p, m: pytest.fail("re-called a done batch"))


def test_run_judges_unparseable_reply_retries_once_then_judge_error(tmp_path):
    items = [{"id": "s-001", "session_id": "s", "transcript": "/p/s.jsonl", "text": "t", "norm": "t",
              "t_ret": "2026-09-10T15:00:00Z", "t_end": "2026-09-10T20:00:00Z",
              "post_candidates": [], "coverage_candidates": []}]
    calls = []

    def stub(prompt, model):
        calls.append(1)
        return {"result": "sorry", "total_cost_usd": 0.01}

    out = pc.run_judges({"items": items, "notes": [], "pop": {"parents": []}}, tmp_path, stub)
    assert len(calls) == 2
    assert out[0]["worth"] is None and out[0]["outcome"] is None


def test_note_deleted_before_return_is_not_coverage():
    t_ret, t_end = "2026-09-10T15:00:00Z", "2026-09-12T15:00:00Z"
    gone = {**note("h.2026-06-01.h.md", "2026-06-01"), "deleted_at": "2026-06-03T21:23:28Z"}
    later = {**note("i.2026-06-01.i.md", "2026-06-01"), "deleted_at": "2026-09-26T12:29:30Z"}
    assert pc.classify_timing(gone, t_ret, t_end) is None
    assert pc.classify_timing(later, t_ret, t_end) == "pre"


def test_note_set_records_deletion_time(tmp_path):
    repo = tmp_path / "vault"
    repo.mkdir()
    git(repo, "init", "-q")
    (repo / "2.2026-09-02.gone.md").write_text(NOTE.format(sit="s2", act="a2", lid="2", created="2026-09-02"))
    git(repo, "add", "-A")
    git(repo, "commit", "-q", "-m", "one")
    (repo / "2.2026-09-02.gone.md").unlink()
    git(repo, "add", "-A")
    git(repo, "commit", "-q", "-m", "two")
    (n,) = pc.note_set(str(repo), {})
    assert n["deleted_at"] is not None and "T" in n["deleted_at"]
