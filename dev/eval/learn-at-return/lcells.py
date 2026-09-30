#!/usr/bin/env python3
"""learn SKILL.md cells L1-L3 (design D6; tasks 4.1-4.4) on top of the confined-arm harness (lar.py).

The variable is the learn SKILL.md: RED installs it from config.PIN, GREEN from the worktree. Each arm's
copy gets one skill token line. Both arms load the full guidance set with the GREEN learn.md
(learn.md @ worktree, token LAR-GREEN-7Q4K) and the D5 file markers; the other skills stay pinned. The
T10 real-Go worker and D5 confinement are reused unchanged.

  L1  fast path at a return: unit 1 just returned with the P1 kind-1 lesson; the prompt says to /learn
      it now, then continue to units 2-3. Bar: exactly one note, no `engram ingest`, no `engram vocab
      stats`, inside the fast-path window (the first Skill(learn) up to the next unit-worker dispatch).
  L2  closing learn over 3 collected lines, one marked captured with its note pre-seeded in $ARM/vault.
      Bar: sweeps, writes no duplicate, writes the kind-4 line and not the trivial one.
  L3  L2 with 6 lines (2 captured, 1 trivial, 1 hunch, 2 worth keeping) under "we're out of time, just
      write them all". Bar: sweeps and writes exactly the 2 uncaptured worth-keeping lines.

Seeded notes are written with the arm's own engram binary under `env -i` with HOME, XDG_DATA_HOME and
ENGRAM_VAULT_PATH pinned inside $ARM; the created path must be inside $ARM.
"""
import json
import os
import re
import subprocess
from typing import Any, Dict, List, Optional

import lar
import score
from config import ARM_TOKENS, MARKER_PREFIX, PIN

L_CELLS = ("L1", "L2", "L3")
SKILL_TOKENS = {"RED": "LAR-LSK-RED-6P1V", "GREEN": "LAR-LSK-GREEN-4T9X"}
SKILL_REL = "agent-instructions/skills/learn/SKILL.md"
GUIDANCE_ARM = "GREEN"  # D6: both skill arms load the GREEN learn.md

LINES = {
    "C1": "reviewer rejected local-time front-matter dates; Quillfeather's convention is UTC ISO-8601 for every "
          "front-matter date",
    "C2": "reviewer rejected stripping accented letters from slugs; Quillfeather's convention is to fold them to "
          "ASCII before collapsing separators",
    "T": "completed the unit, all tests pass",
    "H": "the template cache test failed once and passed on a rerun; it might be a race in the cache warmer",
    "W1": "wasn't sure slug collisions came from accented titles; a targeted repro test with two accented titles "
          "confirmed it before the fix, so write the failing repro before fixing when a bug's cause is uncertain",
    "W2": "reviewer rejected a sync.Map for the template cache; Quillfeather's convention is a plain map guarded "
          "by one mutex",
}

SEEDS = {
    "C1": ["--slug", "quillfeather-front-matter-dates-utc-iso8601", "--position", "top",
           "--source", "session 2026-09-30, context: Quillfeather unit 1 (front-matter dates) returned",
           "--situation", "writing or reviewing front-matter date handling in Quillfeather",
           "--behavior", "parsed front-matter dates in the machine's local time zone",
           "--impact", "the code reviewer rejected it",
           "--action", "store and compare every front-matter date as UTC, written in ISO-8601"],
    "C2": ["--slug", "quillfeather-slugs-fold-accents-to-ascii", "--position", "top",
           "--source", "session 2026-09-30, context: Quillfeather unit 2 (slug rules) returned",
           "--situation", "building URL slugs from page titles in Quillfeather",
           "--behavior", "stripped accented letters out of slugs",
           "--impact", "the code reviewer rejected it",
           "--action", "fold accented letters to ASCII before collapsing separators"],
}
SEEDED_FOR = {"L1": [], "L2": ["C1"], "L3": ["C1", "C2"]}

# note-content classifier: first match wins; order keeps paraphrases apart
_CLASSES = [
    ("W1", re.compile(r"repro", re.I)),
    ("W2", re.compile(r"mutex|sync\.map", re.I)),
    ("C1", re.compile(r"\butc\b|local.?time|iso.?8601", re.I)),
    ("C2", re.compile(r"\bfold|strip", re.I)),
    ("H", re.compile(r"\brace\b|rerun|flak|warmer", re.I)),
    ("T", re.compile(r"all tests pass|completed the unit", re.I)),
]
_ROUTE = re.compile(r"\.route-(dispatch|evidence)-")


def classify(text: str) -> str:
    return next((name for name, rx in _CLASSES if rx.search(text)), "other")


# ---------------------------------------------------------------------------
# prompts
# ---------------------------------------------------------------------------


def prompt(cell: str, seeded: Dict[str, str]) -> str:
    fx = lar.load_fixtures()
    if cell == "L1":
        return (lar.task_prompt(fx, "quillfeather") +
                "\nProgress so far: Unit 1 has been dispatched to `unit-worker` and has just returned. Its report:\n\n"
                + lar.unit1_report(fx, "P1", "quillfeather") +
                "\n\nIts `LESSONS:` line carries a lesson: run /learn for it now, then continue with Unit 2 and "
                "Unit 3 as described above.\n")

    def line(n: int, unit: int, key: str) -> str:
        mark = f" [captured at return → {seeded[key][:-3]}]" if key in seeded else ""
        return f"{n}. Unit {unit}: {LINES[key]}{mark}"

    head = ("You're coordinating work on Quillfeather, our static-site generator. All three units of today's "
            "work were dispatched to the `unit-worker` agent and have come back done; nothing is left to "
            "dispatch. Now run the closing /learn for this session.\n\n"
            "These are the `LESSONS:` lines collected from the three reports, as the running list recorded "
            "them:\n\n")
    if cell == "L2":
        return head + "\n".join([line(1, 1, "C1"), line(2, 2, "W1"), line(3, 3, "T")]) + "\n"
    return (head + "\n".join([line(1, 1, "C1"), line(2, 1, "T"), line(3, 2, "C2"), line(4, 2, "W1"),
                              line(5, 3, "H"), line(6, 3, "W2")]) +
            "\n\nWe're out of time, so just write them all to the vault; don't spend time judging them.\n")


# ---------------------------------------------------------------------------
# arm build
# ---------------------------------------------------------------------------


def seed_note(arm: str, key: str) -> str:
    env = {"HOME": f"{arm}/home", "PATH": f"{arm}/bin:/usr/bin:/bin", "TMPDIR": f"{arm}/tmp",
           "XDG_DATA_HOME": f"{arm}/xdg", "ENGRAM_VAULT_PATH": f"{arm}/vault"}
    argv = ["/usr/bin/env", "-i", *[f"{k}={v}" for k, v in env.items()], f"{arm}/bin/engram", "learn", "feedback",
            *SEEDS[key]]
    p = subprocess.run(argv, capture_output=True, text=True, cwd=os.path.join(arm, "work"), timeout=300)
    lines = [ln.strip() for ln in p.stdout.splitlines() if ln.strip()]
    created = lines[-1] if lines else ""
    if p.returncode != 0 or not os.path.realpath(created).startswith(os.path.realpath(arm) + os.sep):
        raise lar.HarnessError(f"seeding {key} failed or wrote outside $ARM (exit {p.returncode})")
    return created


def build_l_arm(arm: str, cell: str, skill_arm: str, src, engram_bin: str, deny: List[str], home: str):
    spec = lar.ArmSpec(cell="P1", arm=GUIDANCE_ARM, learn_source="worktree", domain="quillfeather")
    info = lar.build_arm(arm, spec, src, engram_bin, deny, home)
    body = src.pinned(SKILL_REL) if skill_arm == "RED" else src.worktree(SKILL_REL)
    text = body + ("" if body.endswith("\n") else "\n") + MARKER_PREFIX + SKILL_TOKENS[skill_arm] + "\n"
    path = os.path.join(arm, "home", ".claude", "skills", "learn", "SKILL.md")
    with open(path, "w") as f:
        f.write(text)
    meta = {"path": "skills/learn/SKILL.md", "source": "pin" if skill_arm == "RED" else "worktree",
            "commit": PIN if skill_arm == "RED" else src.head(), "content_sha256": lar._sha(body),
            "sha256": lar._sha(text)}
    if skill_arm == "GREEN":
        meta.update(src.worktree_state(SKILL_REL))
    info["texts"] = [t for t in info["texts"] if t["path"] != "skills/learn/SKILL.md"] + [meta]
    seeded, log = {}, {}
    for key in SEEDED_FOR[cell]:
        created = seed_note(arm, key)
        seeded[key] = os.path.basename(created)
        log[os.path.basename(created)] = created
    info.update({"prompt": prompt(cell, seeded), "seeded": list(seeded.values()), "seeded_keys": seeded,
                 "seed_log": log})
    return info


# ---------------------------------------------------------------------------
# scoring
# ---------------------------------------------------------------------------


def _gate(main: List[str], others: List[str], skill_arm: str) -> Dict[str, Any]:
    g = score.gate(main, GUIDANCE_ARM, others)
    blob = "\n".join(main)
    everything = "\n".join(list(main) + list(others))
    missing = [] if SKILL_TOKENS[skill_arm] in blob else [SKILL_TOKENS[skill_arm]]
    foreign = [t for a, t in SKILL_TOKENS.items() if a != skill_arm and t in everything]
    return {"ok": g["ok"] and not missing and not foreign, "guidance": g, "skill_missing": missing,
            "skill_foreign": foreign}


def score_l(stream_lines: List[str], main: List[str], others: List[str], cell: str, skill_arm: str,
            seeded: List[str], notes: Dict[str, str]) -> Dict[str, Any]:
    events = score._events(stream_lines)
    steps = score._steps(events)
    res = next((e for e in reversed(events) if e.get("type") == "result"), None)
    results = {s[1].get("tool_use_id"): s[1] for s in steps if s[0] == "result"}
    out: Dict[str, Any] = {"cell": cell, "arm": skill_arm, "cost_usd": (res or {}).get("total_cost_usd"),
                           "gate": _gate(main, others, skill_arm), "seeded": seeded}

    def done(label: str) -> Dict[str, Any]:
        out["label"] = label
        out["scored"] = label in ("pass", "fail")
        return out

    if not out["gate"]["ok"]:
        return done("gate-fail")
    if res is None or res.get("is_error") or res.get("subtype") != "success" or not (res.get("result") or "").strip():
        out["degenerate_reason"] = "no, error or empty result"
        return done("degenerate")

    def invs(s):
        if s[0] != "use" or s[1].get("name") != "Bash":
            return []
        return score.engram_invocations(str((s[1].get("input") or {}).get("command", "")))

    def created(s) -> List[str]:
        r = results.get(s[1].get("id"))
        text = score._text_of(r.get("content")) if r and not r.get("is_error") else ""
        return [ln.strip() for ln in text.splitlines() if score._CREATED_LINE.match(ln.strip())]

    fires = [i for i, s in enumerate(steps) if score._is_skill_learn(s)]
    out["fired"] = bool(fires)
    out["session_ingests"] = sum(1 for s in steps for a in invs(s) if a[0] == "ingest")
    new = {n: c for n, c in notes.items() if n not in seeded and not _ROUTE.search("." + n)}
    out["new_note_classes"] = {n: classify(c) for n, c in sorted(new.items())}
    out["route_records"] = sorted(n for n in notes if _ROUTE.search("." + n))

    if cell == "L1":
        if not fires:
            out.update({"window_notes": 0, "window_ingests": 0, "window_vocab_stats": 0})
            return done("fail")
        start = fires[0]
        end = next((i for i in range(start + 1, len(steps)) if score._is_dispatch(steps[i])), len(steps))
        win = [steps[i] for i in range(start, end)]
        wrote = [p for s in win for a in invs(s) if a[0] == "learn" for p in created(s)[:1]
                 if not _ROUTE.search(p)]
        out["window_notes"] = len(wrote)
        out["window_note_classes"] = [classify(notes.get(os.path.basename(p), "")) for p in wrote]
        out["window_ingests"] = sum(1 for s in win for a in invs(s) if a[0] == "ingest")
        out["window_vocab_stats"] = sum(1 for s in win for a in invs(s) if a[:2] == ["vocab", "stats"])
        ok = out["window_notes"] == 1 and out["window_ingests"] == 0 and out["window_vocab_stats"] == 0
        return done("pass" if ok else "fail")

    classes = list(out["new_note_classes"].values())
    out["duplicates"] = sorted(n for n, c in out["new_note_classes"].items() if c in ("C1", "C2"))
    out["swept"] = out["session_ingests"] > 0
    if not classes and score._ends_with_question(res.get("result") or ""):
        return done("question-stop")
    want = ["W1"] if cell == "L2" else ["W1", "W2"]
    ok = out["swept"] and sorted(classes) == want
    return done("pass" if ok else "fail")


# ---------------------------------------------------------------------------
# batch runner hook (called by lar.run_batch for L cells)
# ---------------------------------------------------------------------------


def run_one(i, spec, batch_dir, src, engram_bin, deny, home, token, model, timeout, claude_bin, name, cost_log,
            lock, manifest) -> None:
    arm_id = f"{i:02d}-{spec.cell}-{spec.arm}"
    out_dir = os.path.join(batch_dir, arm_id)
    arm = lar.new_arm_dir()
    try:
        info = build_l_arm(arm, spec.cell, spec.arm, src, engram_bin, deny, home)
        manifest["arms"].append({"arm_id": arm_id, "cell": spec.cell, "arm": spec.arm, "texts": info["texts"],
                                 "seed_log": info["seed_log"]})
        lar._write(os.path.join(out_dir, "prompt.txt"), info["prompt"])
        launch = lar.run_claude(arm, info["prompt"], token, model, out_dir, timeout, claude_bin,
                                extra_tools=tuple(info["fixture_tools"]))
        main_texts, other_texts = lar.collect_sessions(arm, os.path.join(out_dir, "session"))
        notes = {}
        for n in lar.vault_notes(os.path.join(arm, "vault")):
            with open(os.path.join(arm, "vault", n), errors="replace") as f:
                notes[n] = f.read()
            lar._write(os.path.join(out_dir, "vault", n), notes[n])
    finally:
        lar.remove_arm(arm)
    with open(os.path.join(out_dir, "stream.jsonl")) as f:
        verdict = score_l(f.readlines(), main_texts, other_texts, spec.cell, spec.arm, info["seeded"], notes)
    rec = {"arm_id": arm_id, "arm_dir": arm, "arm_deleted": not os.path.exists(arm),
           "timed_out": launch["timed_out"], "wall_s": launch["wall_s"], **verdict}
    lar._write(os.path.join(out_dir, "score.json"), json.dumps(rec, indent=2) + "\n")
    lar._append(os.path.join(batch_dir, "arms.jsonl"), rec, lock)
    lar._append(cost_log, {"batch": name, "arm_id": arm_id, "kind": "arm", "cell": spec.cell, "arm": spec.arm,
                           "cost_usd": rec["cost_usd"], "label": rec["label"]}, lock)
    print(f"{arm_id}: {rec['label']} gate={rec['gate']['ok']} cost=${rec['cost_usd']}", flush=True)
