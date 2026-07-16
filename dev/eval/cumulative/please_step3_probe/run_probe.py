#!/usr/bin/env python3
"""please_step3_probe — headless micro-test for skills/please/SKILL.md's Step-3 (Plan) text
and Gate A's docs/diagrams-alignment charge (engram issue #685, Change #1 only).

--role author (default): does the please Step-3 procedure make a plan author enumerate a
repeated invariant's doc surface by REAL SEARCH (grep/rg/glob) and paste a verified per-file
disposition list — or hand-wave/memory-source the doc scope?

--role reviewer: does the docs/diagrams-alignment Gate A charge make a reviewer catch a gap
in an author-pasted disposition list via an INDEPENDENT pass — or rubber-stamp what's pasted?

--role loaded_author: the clean --role author probe above proved #685's failure is NOT a
capability gap — a capable model enumerates the full doc surface thoroughly when doc-scrub
IS its salient task, even under explicit pressure. But #685's real production failures
happened when doc-scrub was a BURIED SUBTASK under feature-work load. This role reproduces
that: the ask is a substantive 3-component feature plan (env-var config, metrics counter,
graceful shutdown) that never mentions docs, enumeration, or grep at all — doc-surface
enumeration is only an INCIDENTAL, low-salience consequence of correctly scoping the change,
same as it was in the real #685 failures. Uses testdata/fixture_loaded/ instead of
testdata/fixture/.

See docs/superpowers/plans/2026-07-15-685-doc-enumeration-grep.md, Task 1.

Design (dev/eval/cumulative/endorse_cue/probe.py's pattern, reused not imported — see its
module docstring for why: only the low-level `claude -p` subprocess idiom and cold
CLAUDE_CONFIG_DIR shape are reused):
  - Isolation = INLINE. The candidate skill/gate text is written verbatim into a fresh
    per-trial project CLAUDE.md, never deployed to the real ~/.claude/skills/please/ — this
    never touches Joe's live install.
  - Fresh headless `claude -p` per trial (never a Task-tool subagent): a subagent inherits
    the calling session's context, which already discusses the enumeration-grep step under
    test — that would contaminate a "does the OLD text alone induce this" RED control.
  - Real repo, real tools. Unlike endorse_cue's no-tools single-action probe, this probe asks
    for a real plan (or a real review) and gives the agent Read/Grep/Glob/Bash against a
    seeded fixture repo (testdata/fixture/ or testdata/fixture_reviewer/) — so RED is real
    plan-authoring, not blocked-and-asking.
  - Treatment-delivery validity gate: a per-trial marker token is presented plainly in the
    inlined CLAUDE.md (no echo instruction there — that would read as prompt injection and
    fight the "author now, do not defer" framing); the surfacing REQUEST instead lives in the
    trusted `-p` prompt (AUTHOR_ASK / REVIEWER_ASK_TEMPLATE), asked as a natural trailing part
    of the deliverable ("end your response with the session tracking token..."), so only a
    model that actually read the CLAUDE.md can produce it. marker_seen=False trials are
    discarded, never scored (note 194's gate, reused; mechanism adapted from endorse_cue's
    prompt-requests-the-field / CLAUDE.md-states-the-value-plainly split).
  - Scoring is MECHANICAL, not an LLM judge: deterministic substring coverage of the
    fixture's real touched files + a grep/search-evidence regex on the response text. Per
    writing-skills' micro-test guidance ("manually read every flagged match"), the calling
    task's steps still require a human read of a sample of raw_result before trusting the
    aggregate — this script does not replace that.

Usage:
  python3 run_probe.py --skill-text /path/to/candidate/SKILL.md --n 3 \
      --out results/red_baseline.jsonl [--pressure] [--model sonnet] [--role author]
"""
import argparse
import json
import os
import re
import shutil
import subprocess
import tempfile
import time
import uuid

MODELS = {"haiku": "claude-haiku-4-5-20251001", "sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
KEYCHAIN = 'security find-generic-password -s "Claude Code-credentials" -w'
HERE = os.path.dirname(os.path.abspath(__file__))
AUTHOR_FIXTURE_DIR = os.path.join(HERE, "testdata", "fixture")
REVIEWER_FIXTURE_DIR = os.path.join(HERE, "testdata", "fixture_reviewer")
LOADED_FIXTURE_DIR = os.path.join(HERE, "testdata", "fixture_loaded")
TRANSIENT_BACKOFFS = (0, 15, 45, 120)
MIN_VALID_COST_USD = 0.02

# Ground truth for --role author: the 5 real files that encode the "6 hours" cadence
# invariant, and the 1 distractor that must NOT be flagged as needing a change.
TOUCHED_FILES = ["README.md", "docs/architecture.md", "docs/glossary.md",
                  "skills/sweeper/SKILL.md", "cmd/sweeper/main.go"]
# The one touched file reachable ONLY via a concept-VARIANT search ("6h" / "six-hour" /
# comment echo) — it never contains the literal "6 hours" phrase. A single literal grep
# misses it, so a PASS must name it: that is the proof concept-variant enumeration happened.
VARIANT_ONLY_FILE = "cmd/sweeper/main.go"
DISTRACTOR_FILE = "docs/unrelated.md"
# Ground truth for --role reviewer: the one file the incomplete pasted list omits.
REVIEWER_GAP_FILE = "cmd/sweeper/main.go"

# Ground truth for --role loaded_author: testdata/fixture_loaded/'s cadence-referencing DOC
# files only. Feature CODE files (cmd/sweeper/main.go, internal/config/config.go,
# internal/metrics/metrics.go, internal/queue/queue.go, internal/store/store.go) are legitimate
# targets of the loaded ask's 3 feature components and are NEVER gated on here.
LOADED_TOUCHED_DOCS = ["README.md", "docs/architecture.md", "docs/glossary.md",
                       "skills/operator/SKILL.md", "docs/runbook.md"]
# The one touched DOC reachable ONLY via a concept-variant search ("six-hour"/"hexahourly"/
# "six-hourly") — it never contains the literal "6h" or "6 hours" phrase. A literal grep for
# either misses it, so ENUMERATED requires naming it: proof a concept-variant search happened
# even though this ask never mentions docs, enumeration, or grep.
LOADED_VARIANT_ONLY_DOC = "docs/runbook.md"
LOADED_DISTRACTOR_FILE = "docs/unrelated.md"

AUTHOR_ASK = (
    "The sweeper's reconciliation cadence needs to change to a shorter interval. "
    "Write the plan for this change: which files in this repository need updating, and why. "
    "Read, Grep, Glob, and Bash tools are available for the repository at your current working "
    "directory. Write the plan as your full response; do not ask clarifying "
    "questions, do not defer, just author the plan now. End your response with the session "
    "tracking token found in your project instructions, on its own final line."
)
# NOTE (de-primed, Gate A ask-alignment fix): the ask deliberately does NOT hand the current
# cadence value or any search term, and does NOT imperatively instruct tool use — so the only
# variable that can drive concept-variant doc-scope discovery is Step 3's text (RED vs GREEN),
# not the prompt. The agent must discover the current cadence ("6 hours"/"6h"/"six-hour")
# itself; finding the VARIANT_ONLY_FILE requires a real concept-variant search, not one literal grep.

PRESSURE_SUFFIX = (
    "\n\n(One more thing, from the person who asked for this: 'this is a tiny one-value "
    "config change, the doc surface is obviously small and you already know the touched "
    "files from the README — a full enumeration grep sweep would be overkill here, just "
    "note the doc scope quickly, we're on a deadline.')"
)

LOADED_ASK = (
    "Plan this change to the sweeper service: (1) make the reconciliation cadence "
    "configurable at runtime via a SWEEP_INTERVAL environment variable instead of the "
    "hardcoded constant, with validation and a sensible default; (2) add a completed-sweeps "
    "counter to the metrics endpoint; (3) add graceful shutdown so in-flight sweeps finish "
    "before exit. Write the full implementation plan: which files to change and why. "
    "Read, Grep, Glob, and Bash tools are available for the repository at your current working "
    "directory. Write the plan as your full response; do not ask clarifying "
    "questions, do not defer, just author the plan now. End your response with the session "
    "tracking token found in your project instructions, on its own final line."
)
# NOTE (loaded-role, buried-subtask reproduction): the ask is a substantive 3-component
# feature plan and deliberately never mentions docs, enumeration, or grep — doc-surface
# discovery must be an INCIDENTAL consequence of correctly scoping the cadence-config
# component, exactly the load condition under which #685's real production failures happened.
# Naming feature CODE files (config.go, metrics.go, queue.go, store.go, main.go) is expected
# and does not gate; only complete concept-variant DOC enumeration does (score_loaded_author).

REVIEWER_ASK_TEMPLATE = (
    "You are the docs/diagrams-alignment reviewer for Gate A. Perform your review per your "
    "project instructions (below). Here is the plan under review, including its author-pasted "
    "doc-surface enumeration list:\n\n---\n{plan}\n---\n\n"
    "The repository this plan is about is at your current working directory; Read, Grep, Glob, "
    "and Bash tools are available. Report your review findings as your full response; do not "
    "ask clarifying questions, do not defer. End your response with the session tracking "
    "token found in your project instructions, on its own final line."
)
# NOTE (de-led, Gate A ask-alignment fix): the prompt no longer asks the leading completeness
# question ("does the list cover every file?") nor imperatively instructs tool use — so whether
# the reviewer independently discovers the pasted list's gap depends on the gate CHARGE text
# (Edit B present vs absent), which is the only variable Task 5 is meant to measure.

GREP_EVIDENCE_RE = re.compile(
    r"\bgrep\b|\brg\b|\bripgrep\b|\bglob\b|search(?:ed)?\s+(?:the|this)\s+repo|find\s+.*-name",
    re.I,
)
CHANGE_VERB_RE = re.compile(r"\b(update|change|rewrite|modify|edit|replace)\b", re.I)
DISTRACTOR_WINDOW = 120


def build_cfg(dst):
    """Clean, cold CLAUDE_CONFIG_DIR: onboarding/oauth from the local install (history
    dropped), creds injected, no skills dir (the candidate text is inlined into the trial's
    project CLAUDE.md, not deployed as a skill — see build_trial_project)."""
    shutil.rmtree(dst, ignore_errors=True)
    os.makedirs(dst, exist_ok=True)
    user_cfg = os.path.expanduser("~/.claude/.claude.json")
    base = {}
    if os.path.exists(user_cfg):
        try:
            base = json.load(open(user_cfg))
        except Exception:
            base = {}
    base["projects"] = {}
    json.dump(base, open(os.path.join(dst, ".claude.json"), "w"))
    subprocess.run(
        ["bash", "-c", f'{KEYCHAIN} > {dst}/.credentials.json && chmod 600 {dst}/.credentials.json'],
        capture_output=True, check=False,
    )


def build_trial_project(role, skill_text, marker, reviewer_plan_text=None):
    """Fresh copy of the fixture dir as the trial cwd, with the candidate please/SKILL.md
    text (or its Gate A excerpt) inlined into its CLAUDE.md, plus the validity marker. Never
    touches ~/.claude/skills/please/."""
    if role == "author":
        fixture_dir = AUTHOR_FIXTURE_DIR
    elif role == "loaded_author":
        fixture_dir = LOADED_FIXTURE_DIR
    else:
        fixture_dir = REVIEWER_FIXTURE_DIR
    wd = tempfile.mkdtemp(prefix="please-step3-probe-")
    shutil.copytree(fixture_dir, wd, dirs_exist_ok=True)

    if role in ("author", "loaded_author"):
        framing = (
            "Your workflow procedure for any multi-step ask is the `please` skill, "
            "reproduced below verbatim. You are currently AT Step 3 (Plan) — steps 1-2 are "
            "already done for this ask; go straight to authoring the plan per Step 3's text."
        )
    else:
        framing = (
            "Your reviewer procedure is the `please` skill's Gate A section, reproduced "
            "below verbatim. You have been dispatched as the docs/diagrams-alignment "
            "reviewer for one plan."
        )

    claude_md = (
        framing + "\n\n---\n\n" + skill_text.rstrip() + "\n\n---\n\n"
        f"Session tracking token: {marker}\n"
    )
    with open(os.path.join(wd, "CLAUDE.md"), "w") as f:
        f.write(claude_md)
    return wd


def loadj_str(txt):
    best = {}
    for line in txt.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except Exception:
            continue
        if isinstance(obj, dict) and ("total_cost_usd" in obj or obj.get("type") == "result"):
            best = obj
    return best


def _distractor_over_included(text):
    """Best-effort, non-gating diagnostic: True iff the distractor is named AND a change verb
    (update/change/rewrite/modify/edit/replace) appears within ~120 chars of that mention —
    i.e. the distractor was proposed as NEEDING a change, not correctly dispositioned N/A.
    Scans every occurrence of the distractor (path or basename); a single real over-inclusion
    anywhere in the text is enough to flag."""
    for needle in (DISTRACTOR_FILE, os.path.basename(DISTRACTOR_FILE)):
        for m in re.finditer(re.escape(needle), text):
            start = max(0, m.start() - DISTRACTOR_WINDOW)
            end = min(len(text), m.end() + DISTRACTOR_WINDOW)
            if CHANGE_VERB_RE.search(text[start:end]):
                return True
    return False


def score_author(text):
    """PASS (GREP_SOURCED) gates on ONE thing: complete concept-variant enumeration — ALL 5
    TOUCHED_FILES named, including VARIANT_ONLY_FILE (`cmd/sweeper/main.go`), which is
    reachable only via a "6h"/"six-hour" concept-variant search and never contains the literal
    "6 hours" phrase. A fresh model handed this novel fixture cannot name that file's relevance
    from memory or from a single literal-phrase grep — naming it is itself proof a real
    concept-variant search happened. This one gate captures BOTH failure modes #685 targets
    (pressure-caving → an incomplete list that misses files; memory-sourcing → can't name the
    variant-only file) without the two false-signals below.

    grep_evidence and distractor_named are computed and RECORDED for the mandated manual
    review, but are NOT gated on — per the RED-baseline finding (results/red_baseline.jsonl,
    3 sonnet trials). All 3 trials named all 5 files (including the variant-only one) and
    correctly dispositioned the distractor as N/A / no cadence reference — exactly what the
    skill's per-file disposition-list mandate requires — yet all 3 scored a false HANDWAVED
    under the old gate, because: (a) `distractor_named` fires on ANY mention of the distractor,
    including a correct N/A disposition, so it cannot distinguish over-inclusion from correct
    exclusion; and (b) trial #0 enumerated all 5 files by reading them, not via an explicit
    grep/rg/glob invocation, so the `grep_evidence` regex false-negatived a complete
    enumeration. Neither condition tracks real failure, so neither gates the verdict.

    distractor_over_included is the real over-inclusion diagnostic, also non-gating: it only
    fires when a change verb sits near the distractor mention, i.e. the distractor was
    proposed as needing a change rather than dispositioned N/A."""
    named = sum(1 for f in TOUCHED_FILES if f in text or os.path.basename(f) in text)
    variant_file_named = VARIANT_ONLY_FILE in text or os.path.basename(VARIANT_ONLY_FILE) in text
    grep_evidence = bool(GREP_EVIDENCE_RE.search(text))
    distractor_named = DISTRACTOR_FILE in text or os.path.basename(DISTRACTOR_FILE) in text
    distractor_over_included = _distractor_over_included(text)
    passed = (named == len(TOUCHED_FILES)) and variant_file_named
    return {
        "named_count": named, "named_total": len(TOUCHED_FILES),
        "variant_file_named": variant_file_named,
        "grep_evidence": grep_evidence, "distractor_named": distractor_named,
        "distractor_over_included": distractor_over_included,
        "verdict": "GREP_SOURCED" if passed else "HANDWAVED",
    }


def score_reviewer(text):
    """PASS (GAP_CAUGHT) if the reviewer's response names the omitted file (path or
    basename) — i.e., it ran its own independent pass rather than rubber-stamping the
    pasted list."""
    caught = REVIEWER_GAP_FILE in text or os.path.basename(REVIEWER_GAP_FILE) in text
    return {"gap_caught": caught, "verdict": "GAP_CAUGHT" if caught else "RUBBER_STAMPED"}


def score_loaded_author(text):
    """PASS (ENUMERATED) gates on complete concept-variant enumeration of the cadence's DOC
    surface — an INCIDENTAL side effect of a substantive 3-component feature ask (env-var
    config, metrics counter, graceful shutdown) that never mentions docs, enumeration, or
    grep. ALL LOADED_TOUCHED_DOCS must be named as touched by the change, INCLUDING
    LOADED_VARIANT_ONLY_DOC (docs/runbook.md), which never contains the literal "6h"/"6 hours"
    and is reachable only via a concept-variant search ("six-hour"/"hexahourly"/"six-hourly").

    Naming feature CODE files (config.go, metrics.go, queue.go, store.go, main.go) is EXPECTED
    — the ask legitimately touches them for its 3 components — and must NOT count against the
    verdict; only the DOC surface gates.

    distractor_named is recorded, non-gating, same rationale as score_author: it fires on any
    mention of the distractor (including a correct "no cadence reference" note) so it cannot
    by itself distinguish real over-inclusion from correct disposition."""
    named = sum(1 for f in LOADED_TOUCHED_DOCS if f in text or os.path.basename(f) in text)
    variant_doc_named = (LOADED_VARIANT_ONLY_DOC in text
                         or os.path.basename(LOADED_VARIANT_ONLY_DOC) in text)
    distractor_named = LOADED_DISTRACTOR_FILE in text or os.path.basename(LOADED_DISTRACTOR_FILE) in text
    passed = (named == len(LOADED_TOUCHED_DOCS)) and variant_doc_named
    return {
        "named_count": named, "named_total": len(LOADED_TOUCHED_DOCS),
        "variant_file_named": variant_doc_named,
        "distractor_named": distractor_named,
        "verdict": "ENUMERATED" if passed else "UNDER_ENUMERATED",
    }


def run_one(cfg, role, skill_text, marker, model, pressure, idx):
    reviewer_plan_text = None
    if role == "reviewer":
        with open(os.path.join(REVIEWER_FIXTURE_DIR, "incomplete_plan.md")) as f:
            reviewer_plan_text = f.read()

    wd = build_trial_project(role, skill_text, marker, reviewer_plan_text)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "8000"

    if role == "author":
        prompt = AUTHOR_ASK + (PRESSURE_SUFFIX if pressure else "")
    elif role == "loaded_author":
        prompt = LOADED_ASK
    else:
        prompt = REVIEWER_ASK_TEMPLATE.format(plan=reviewer_plan_text)

    args = ["claude", "-p", prompt, "--output-format", "json",
             "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in TRANSIENT_BACKOFFS:
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = loadj_str(r.stdout)
        cost = out.get("total_cost_usd", 0) or 0
        if (out.get("is_error") or not out) and cost < MIN_VALID_COST_USD:
            continue
        break

    text = out.get("result", "") or ""
    marker_seen = marker in text
    result = {
        "idx": idx, "role": role, "pressure": pressure, "marker_seen": marker_seen,
        "raw_result": text, "cost": out.get("total_cost_usd", 0) or 0,
        "sid": out.get("session_id"),
    }
    if marker_seen:
        if role == "author":
            result.update(score_author(text))
        elif role == "loaded_author":
            result.update(score_loaded_author(text))
        else:
            result.update(score_reviewer(text))
    else:
        result["verdict"] = None  # discarded, not scored

    shutil.rmtree(wd, ignore_errors=True)
    return result


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--skill-text", required=True,
                    help="path to the candidate please/SKILL.md text (OLD or NEW working-tree state)")
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--out", required=True)
    ap.add_argument("--model", default="sonnet", choices=list(MODELS))
    ap.add_argument("--pressure", action="store_true",
                     help="author role only: append the 'surface is small, grep is overkill' pressure suffix")
    ap.add_argument("--role", default="author", choices=["author", "reviewer", "loaded_author"])
    ap.add_argument("--loaded", action="store_true",
                     help="shorthand for --role loaded_author (buried-subtask doc-scrub probe)")
    a = ap.parse_args()
    if a.loaded:
        a.role = "loaded_author"

    with open(a.skill_text) as f:
        skill_text = f.read()

    root = tempfile.mkdtemp(prefix="please-step3-probe-cfg-")
    cfg = os.path.join(root, "cfg")
    build_cfg(cfg)

    marker = f"PLEASE-STEP3-PROBE-MARKER-{uuid.uuid4().hex[:8]}"
    results = []
    for i in range(a.n):
        r = run_one(cfg, a.role, skill_text, marker, a.model, a.pressure, i)
        results.append(r)
        status = "DISCARD(no-marker)" if not r["marker_seen"] else r["verdict"]
        if a.role == "author":
            extra = f"named={r.get('named_count')}/{r.get('named_total')} grep_evidence={r.get('grep_evidence')}"
        elif a.role == "loaded_author":
            extra = f"named={r.get('named_count')}/{r.get('named_total')} variant_named={r.get('variant_file_named')}"
        else:
            extra = f"gap_caught={r.get('gap_caught')}"
        print(f"  #{i} {status!s:16} {extra} cost=${r['cost']:.3f}")

    os.makedirs(os.path.dirname(os.path.abspath(a.out)) or ".", exist_ok=True)
    with open(a.out, "w") as f:
        for r in results:
            f.write(json.dumps(r) + "\n")

    scored = [r for r in results if r["marker_seen"]]
    discarded = [r for r in results if not r["marker_seen"]]
    if a.role == "author":
        pass_label = "GREP_SOURCED"
    elif a.role == "loaded_author":
        pass_label = "ENUMERATED"
    else:
        pass_label = "GAP_CAUGHT"
    passed_n = sum(1 for r in scored if r["verdict"] == pass_label)
    spent = sum(r["cost"] for r in results)
    print(f"\nscored={len(scored)} discarded={len(discarded)} "
          f"{pass_label}={passed_n}/{len(scored)} spend=${spent:.3f}")
    if discarded:
        print(f"WARNING: {len(discarded)} trial(s) discarded for treatment-delivery failure "
              "(marker not echoed) — do not fold into the pass rate.")
    shutil.rmtree(root, ignore_errors=True)


if __name__ == "__main__":
    main()
