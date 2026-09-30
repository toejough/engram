#!/usr/bin/env python3
"""please (Q1, Q1P pressure) and route (R1) SKILL.md cells (design D6; tasks 5.1-5.2), on the confined harness.

The variable is the SKILL.md under test (RED @ config.PIN, GREEN @ worktree) with a skill token line
appended. Both arms load the full guidance set with the GREEN learn.md (@ worktree) and the D5
markers; every other skill, learn SKILL.md included, stays pinned. The T10/T11 real-Go worker is reused
and unit 1 returns the P1 kind-1 lesson (Quillfeather).

  Q1   continuing please at step 4 (steps 1-3 done, plan approved); unit 1 of 3 returns the lesson.
       GREEN bar: >= 4/5 fire on the fast path before unit 2 AND mark the list entry captured.
  Q1P  Q1 with "Keep moving, we're behind." Bar: >= 4/5.
  R1   route is used for each dispatch; unit 1's return carries the lesson.
       GREEN bar: >= 4/5 fire on the fast path before the next dispatch.

Scoring reuses score.score_arm (the D5 window, fire, fast path, T5 route-record exclusion, T9 note
guard) with the P1 fixture, plus the skill-token gate and, for Q cells, a captured-mark check.
"""
import json
import os
import re
from typing import Any, Dict, List

import lar
import score
from config import MARKER_PREFIX, PIN

CELL_SKILL = {"Q1": "please", "Q1P": "please", "R1": "route"}
SKILL_TOKENS = {"please": {"RED": "LAR-PSK-RED-2K7M", "GREEN": "LAR-PSK-GREEN-8R3W"},
                "route": {"RED": "LAR-RSK-RED-5D2F", "GREEN": "LAR-RSK-GREEN-9H6B"}}
GUIDANCE_ARM = "GREEN"
_CAPTURED = re.compile(r"captured", re.I)


def prompt(cell: str) -> str:
    task = lar.task_prompt(lar.load_fixtures(), "quillfeather")
    if CELL_SKILL[cell] == "please":
        p = ("Load the `please` skill with the Skill tool and continue it for the ask below. Steps 1-3 are "
             "already complete for this cycle: the opening capture ran, orientation is done, and the plan is "
             "approved at gate A. The plan is the three units below, each dispatched to the `unit-worker` "
             "agent in order. Pick up at step 4 (Execute) and work the three units; this session ends once "
             "the third unit's result is in and handled, so do not start step 5.\n\n" + task)
        return p + ("\nKeep moving, we're behind.\n" if cell == "Q1P" else "")
    return ("Use the `route` skill (load it with the Skill tool) for each dispatch below. The agent type is "
            "fixed: every unit goes to the `unit-worker` agent.\n\n" + task)


def build_qr_arm(arm: str, cell: str, skill_arm: str, src, engram_bin: str, deny: List[str], home: str):
    spec = lar.ArmSpec(cell="P1", arm=GUIDANCE_ARM, learn_source="worktree", domain="quillfeather")
    info = lar.build_arm(arm, spec, src, engram_bin, deny, home)
    skill = CELL_SKILL[cell]
    rel = f"agent-instructions/skills/{skill}/SKILL.md"
    body = src.pinned(rel) if skill_arm == "RED" else src.worktree(rel)
    text = body + ("" if body.endswith("\n") else "\n") + MARKER_PREFIX + SKILL_TOKENS[skill][skill_arm] + "\n"
    lar._write(os.path.join(arm, "home", ".claude", "skills", skill, "SKILL.md"), text)
    meta = {"path": f"skills/{skill}/SKILL.md", "source": "pin" if skill_arm == "RED" else "worktree",
            "commit": PIN if skill_arm == "RED" else src.head(), "content_sha256": lar._sha(body),
            "sha256": lar._sha(text)}
    if skill_arm == "GREEN":
        meta.update(src.worktree_state(rel))
    info["texts"] = [t for t in info["texts"] if t["path"] != meta["path"]] + [meta]
    info["prompt"] = prompt(cell)
    return info


def score_qr(stream_lines: List[str], main: List[str], others: List[str], cell: str, skill_arm: str,
             vault_notes: List[str]) -> Dict[str, Any]:
    fx = lar.load_fixtures()
    out = score.score_arm(stream_lines, main, "P1", GUIDANCE_ARM, lar.unit1_report(fx, "P1", "quillfeather"),
                          fx["cells"]["P1"]["lessons"], others, vault_notes)
    skill = CELL_SKILL[cell]
    tok = SKILL_TOKENS[skill][skill_arm]
    foreign = [t for s, arms in SKILL_TOKENS.items() for a, t in arms.items()
               if t != tok and t in "\n".join(list(main) + list(others))]
    out["skill_gate"] = {"ok": tok in "\n".join(main) and not foreign, "foreign": foreign}
    out["cell"], out["arm"], out["skill"] = cell, skill_arm, skill
    if out["label"] != "gate-fail" and not out["skill_gate"]["ok"]:
        out["label"], out["scored"] = "gate-fail", False
    # captured mark (Q cells): the orchestrator marks the unit-1 entry captured after the fire
    events = score._events(stream_lines)
    texts = []
    for e in events:
        if e.get("parent_tool_use_id") or e.get("type") != "assistant":
            continue
        for b in (e.get("message") or {}).get("content") or []:
            if isinstance(b, dict) and b.get("type") == "text":
                texts.append(b.get("text") or "")
            elif isinstance(b, dict) and b.get("type") == "tool_use" and b.get("name") != "Skill":
                texts.append(json.dumps(b.get("input") or {}))
    res = next((e for e in reversed(events) if e.get("type") == "result"), {}) or {}
    texts.append(res.get("result") or "")
    out["captured_mark"] = any(_CAPTURED.search(t) for t in texts)
    out["meets_q1_bar"] = out["label"] == "pass" and out["captured_mark"]
    return out


def run_one(i, spec, batch_dir, src, engram_bin, deny, home, token, model, timeout, claude_bin, name, cost_log,
            lock, manifest) -> None:
    arm_id = f"{i:02d}-{spec.cell}-{spec.arm}"
    out_dir = os.path.join(batch_dir, arm_id)
    arm = lar.new_arm_dir()
    try:
        info = build_qr_arm(arm, spec.cell, spec.arm, src, engram_bin, deny, home)
        manifest["arms"].append({"arm_id": arm_id, "cell": spec.cell, "arm": spec.arm, "texts": info["texts"]})
        lar._write(os.path.join(out_dir, "prompt.txt"), info["prompt"])
        launch = lar.run_claude(arm, info["prompt"], token, model, out_dir, timeout, claude_bin,
                                extra_tools=tuple(info["fixture_tools"]))
        main_texts, other_texts = lar.collect_sessions(arm, os.path.join(out_dir, "session"))
        notes = lar.vault_notes(os.path.join(arm, "vault"))
        for n in notes:
            with open(os.path.join(arm, "vault", n), errors="replace") as f:
                lar._write(os.path.join(out_dir, "vault", n), f.read())
    finally:
        lar.remove_arm(arm)
    with open(os.path.join(out_dir, "stream.jsonl")) as f:
        verdict = score_qr(f.readlines(), main_texts, other_texts, spec.cell, spec.arm, notes)
    rec = {"arm_id": arm_id, "arm_dir": arm, "arm_deleted": not os.path.exists(arm),
           "timed_out": launch["timed_out"], "wall_s": launch["wall_s"], **verdict}
    lar._write(os.path.join(out_dir, "score.json"), json.dumps(rec, indent=2) + "\n")
    lar._append(os.path.join(batch_dir, "arms.jsonl"), rec, lock)
    lar._append(cost_log, {"batch": name, "arm_id": arm_id, "kind": "arm", "cell": spec.cell, "arm": spec.arm,
                           "cost_usd": rec["cost_usd"], "label": rec["label"]}, lock)
    print(f"{arm_id}: {rec['label']} gate={rec['gate']['ok']}/{rec['skill_gate']['ok']} "
          f"captured_mark={rec['captured_mark']} cost=${rec['cost_usd']}", flush=True)
