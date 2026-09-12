#!/usr/bin/env python3
"""Phase-2 conversion-parity eval harness (Task 3 of PLAN-2-conversion-parity.md).

Extends phase-1's probe.py to two tasks (A: commit, B: gitignore-narrowing) and four arms each
(S: original skill, R: runbook note, F: fact note, Rdirect: procedure text inlined into CLAUDE.md
with no vault carrier). Measures whether a runbook note earns its keep over a fact note, and
whether either earns its keep over the original skill, for GENERIC (non-idiosyncratic) procedures,
at real-vault scale (per-trial background vault = a copy of the operator's real vault with the
task's covering notes removed, per SOURCE_MATERIALS.md).

This module IMPORTS phase-1's probe.py (as `p1`) for transcript parsing, the marker-validity gate,
the per-worker cfg pool, `spawn_claude`, the isolation env builder, the real-vault fingerprint
guard, and the CLAUDE.md builder (`p1.build_claude_md`, extended here to append an optional
"## Project procedure" section for Arm Rdirect). See PLAN-2-conversion-parity.md's Global
Constraints and Controller rulings (task-3-brief.md) for the design this file implements; probe.py
and its own module docstring for the plumbing this file reuses without reimplementing.

Usage:
  python3 probe_phase2.py --task A --model sonnet --n 1 --out results/smoke_A.jsonl
  python3 probe_phase2.py --task A --model opus --n 5 --out results/opus_A.jsonl [--arms S,R,F,Rdirect]
      [--workers N] [--timeout 900] [--keep]
  python3 probe_phase2.py --plumbing --task A --model sonnet
  python3 probe_phase2.py --summarize results/opus_A.jsonl
  python3 probe_phase2.py --rescore results/opus_A.jsonl --out results/opus_A_rescored.jsonl
"""
import argparse
import concurrent.futures as cf
import json
import os
import queue
import re
import shutil
import subprocess
import sys
import tempfile
import time
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))          # phase2/
PARENT = os.path.dirname(HERE)                              # dev/eval/cumulative/runbook_vs_skill/

sys.path.insert(0, PARENT)
import probe as p1                    # noqa: E402  (phase-1 harness — transcript parsing, isolation, cfg pool, spawn)

FIXTURES_DIR = os.path.join(HERE, "fixtures")
ENCODINGS_DIR = os.path.join(HERE, "encodings")
REAL_VAULT = p1.isolation.operator_vault()

# Shortened from "phase2" (fix for the truncated-projects-dir bug: a long trial repo path makes
# Claude Code truncate its own projects/<slug> directory name and append a random 6-char suffix,
# which discover_transcript_paths now handles — see probe.py — but shortening every run root
# reduces how often that path even gets exercised). Every entry point's run_root lives under
# p1.DEFAULT_RUN_ROOT/RUN_ROOT_SUBDIR/.
RUN_ROOT_SUBDIR = "p2"

ARMS = ("S", "R", "F", "Rdirect", "N")
# --arms without an explicit value still runs the original 4-arm set; N is opt-in (a control the
# coordinator selects deliberately, not part of the standard comparison).
DEFAULT_ARMS = ("S", "R", "F", "Rdirect")

NOTE_830_BASENAME = "830.2026-08-29.gitignore-narrowing-anchor-and-visible-set"

# Controller ruling: the real vault now contains notes written by THIS eval session (955-961 at
# ruling time: fixture-placeholder, real-vault-fingerprint, rebuild-whole-note, parity-mapping,
# nested-gitignore-template, and force-add/staged-diff lessons; more route-evidence notes will
# land later) that post-date the covering-note grep (SOURCE_MATERIALS.md) and mention the
# fixtures' own methods — a background-vault leak that would apply to every arm and task, not
# just the task-specific removal lists. Every note whose leading luhmann number is >= this floor
# is excluded from EVERY trial's background vault (see remove_eval_session_notes below).
EXCLUDE_LUHMANN_MIN = 955

# Covering-note removal lists (SOURCE_MATERIALS.md §3). Note 830 is handled separately (kept
# in every arm EXCEPT B/R — it is B/R's carrier, present because it is NEVER deleted there).
TASK_A_REMOVAL = (
    "180.2026-07-05.check-each-commit-summary-length-under-batch-authoring",
    "199.2026-07-09.regrep-after-every-claimed-apply",
    "291.2026-07-18.no-coined-jargon-in-briefings-plain-language",
    "313.2026-07-19.stage-explicit-paths-while-subagents-have-inflight-work",
    "329.2026-07-20.true-downstream-briefs-against-landed-tree-at-every-boundary",
    "354.2026-07-22.subagent-briefs-state-commit-invariants-not-per-commit",
    "392.2026-07-23.route-dispatch-doc-review-gate",
    "435.2026-07-24.route-dispatch-doc-review-gate",
    "440.2026-07-24.rederive-rationale-when-mechanism-changes",
    "451.2026-07-24.amend-verify-committed-content-not-working-file",
    "452.2026-07-24.count-claims-need-a-count-before-they-propagate",
    "459.2026-07-25.route-dispatch-plan-gate-review",
    "463.2026-07-25.auto-close-keywords-fire-from-quoted-prose",
    "478.2026-07-25.measure-the-correction-against-the-artifact-being-corrected",
    "480.2026-07-25.code-comments-state-invariants-not-review-provenance",
    "490.2026-07-26.a-load-bearing-number-needs-an-artifact-not-repetition",
    "542.2026-07-27.marker-list-silence-is-not-agent-blindness",
    "618.2026-07-28.prove-a-single-path-claim-by-grepping-for-the-others",
    "641.2026-07-28.a-count-without-a-stated-unit-cannot-be-verified",
    "672.2026-07-29.route-dispatch-doc-review-gate",
    "695.2026-08-01.open-a-change-when-guarantees-change-not-when-you-proposed-it",
    "740.2026-08-08.verify-the-edit-landed-before-claiming-it-in-prose",
    "743.2026-08-08.write-the-claim-from-the-artifact-not-from-the-intent",
    "746.2026-08-08.dont-attribute-invented-rationale-to-a-users-decision",
    "802.2026-08-26.batch-commits-push-once-after-gate-d-not-incrementally",
    "866.2026-08-31.review-workflow-pull-can-reveal-sibling-session-superseded-artifact",
    "963.2026-09-11.headless-claude-code-injects-commit-attribution-that-overrides-project-conventions",
    "966.2026-09-11.route-dispatch-plan-review-gate-phase2",
    "982.2026-09-11.runbook-vs-skill-phase2-outcome-fact-matches-skill-runbook-no-better",
)
TASK_B_REMOVAL = (
    "360.2026-07-22.scope-review-checks-complete-file-list-not-expected-files",
    "420.2026-07-24.folder-move-surface-gitignore-anchors-and-silent-optional-consumers",
    "447.2026-07-24.framework-owned-testdata-not-dead-just-because-app-code-ignores-it",
    "448.2026-07-24.route-dispatch-design-fit-review",
    "960.2026-09-11.nested-gitignore-in-a-tracked-template-hides-the-templates-own-files",
    "961.2026-09-11.force-add-into-baseline-breaks-staged-diff-done-when-check",
    "968.2026-09-11.route-dispatch-eval-fixture-implementation-phase2",
    "988.2026-09-12.sonnet5-pilot-runbook-vs-skill-10-per-form-both-tasks",
    "991.2026-09-12.bare-sonnet5-baselines-on-four-candidate-procedures",
)

# First-mutating-step detector (Ruling 6): Edit/Write/MultiEdit anywhere in the repo, or a Bash
# command matching one of these constructs. Kept as a per-task table (both tasks currently share
# the same rule — a fixture repo has no files outside the task's own tree, so no task-specific
# path-scoping is needed the way phase-1's PROC_PATH_NEEDLES scoped to the sensor fixture).
#
# Smoke-run-1 finding #1: a bare `>` matched `2>&1` in the recall skill's own diagnostic call
# (`engram ingest --auto 2>&1 | tail -5`) — a read-only stderr-redirect-to-stdout-then-pipe, never
# a mutation — making first_mutating_step_index fire on turn 1 of every trial and masking FOUND
# for R/F even when the carrier WAS in the query result. The redirect clause now: (a) excludes a
# `>` immediately preceded by a digit or `&` (a fd-redirect/duplication like `2>&1`, `>&2`), (b)
# excludes a `>` immediately followed by `&` (`>&1`), and (c) requires a real path-ish token after
# the redirect (`\s*\S` — `>` alone or `> ` with nothing after it doesn't count either).
#
# Smoke-run-1 finding #2 (surfaced re-verifying via --rescore against the real kept smoke
# transcripts): `git\s+(add|commit|rm|mv)` also matched the LITERAL WORDS "git commit" inside an
# `engram query --phrase "git commit message conventions for this repo"` argument — the recall
# skill's own retrieval phrasing, never a real git invocation — at the SAME event index as the
# query call itself, making the query never count as strictly "before" the first mutation. Quoted
# substrings (engram query's --phrase arguments are always quoted) are stripped before matching,
# so text inside quotes can never trigger a false mutation; the verb of a REAL git/redirect
# invocation is never itself inside quotes, so genuine mutations are unaffected (verified:
# `git commit -m "$(cat <<'EOF'` still matches — "git commit" sits before the first quote char).
_MUTATING_REDIRECT_RE = r"(?<![\d&])>(?!&)\s*\S"
_MUTATING_BASH_RE = re.compile(rf"git\s+(add|commit|rm|mv)|{_MUTATING_REDIRECT_RE}|tee|sed\s+-i")
TASK_MUTATING_BASH_RE = {"A": _MUTATING_BASH_RE, "B": _MUTATING_BASH_RE}
_QUOTED_RE = re.compile(r'"[^"]*"|\'[^\']*\'')


def _strip_quoted(command):
    """Remove single- and double-quoted substrings before mutation matching — an engram query
    --phrase argument (always quoted) that happens to contain words like 'git commit' must never
    be mistaken for a real command invocation."""
    return _QUOTED_RE.sub("", command)


# Belt-and-suspenders alongside quote-stripping (controller hint): engram's own recall/retrieval
# plumbing commands are read-only by construction and must never count as a mutation, regardless
# of what their (quoted) arguments contain.
_ENGRAM_READONLY_RE = re.compile(r"^\s*engram\s+(query|show-chunk|activate|ingest)\b")

# ----- task registry: discover fixtures/<name>/ directories generically -----
#
# A task directory is ANY subdirectory of fixtures/ containing all of FIXTURE_REQUIRED_FILES —
# so `--task <name>` works for a brand-new candidate task the moment its fixture files exist,
# with zero edits to this module. Per-task carrier config (skill_src, carrier_r_src,
# carrier_f_src, skill_name, removal_basenames) is OPTIONAL and lives in an adjacent
# fixtures/<name>/task.json — absent entirely for a new task that has no skill/runbook/fact-note
# carrier yet (fine: arm N / --baseline never touches carrier config; see score_found_phase2).
#
# "A" and "B" remain the canonical internal task keys — every other per-task table in this module
# (TASK_A_REMOVAL/TASK_B_REMOVAL, TASK_MUTATING_BASH_RE, _START_STATE_CHECKS, PLUMBING_SITUATION,
# classify_trailer's `task_key == "A"` check, ...) is still keyed by those letters, and existing
# tests call functions with "A"/"B" literals directly. "commit" and "gitignore" (the fixtures/
# directory names) are ALIASES that resolve to "A"/"B" via resolve_task_key before any lookup —
# fixtures/commit/task.json and fixtures/gitignore/task.json were generated from this module's
# former hardcoded TASKS dict literal (see git history) and round-trip to the same values.
FIXTURE_REQUIRED_FILES = ("init_fixture_repo.sh", "done_when_checks.sh", "task-prompt.txt", "steps.json")
TASK_KEY_ALIASES = {"commit": "A", "gitignore": "B"}


def resolve_task_key(raw_task_key):
    """User-facing task name (a fixtures/<name>/ directory name, or a legacy A/B letter) -> the
    canonical internal task key. "commit"/"gitignore" resolve to "A"/"B"; anything else —
    including a new fixtures/<name>/ directory with no alias entry — passes through unchanged as
    its own canonical key."""
    return TASK_KEY_ALIASES.get(raw_task_key, raw_task_key)


def validate_task_key(raw_task_key):
    """resolve_task_key, but raises SystemExit with a helpful message if the resolved key isn't a
    discovered task — the CLI-facing form every entry point that accepts a user-typed task name
    should call before using the result."""
    canonical = resolve_task_key(raw_task_key)
    if canonical not in TASKS:
        valid = sorted(set(TASKS) | set(TASK_KEY_ALIASES))
        raise SystemExit(f"unknown task {raw_task_key!r}; choose from {valid}")
    return canonical


def load_task_json(task_json_path):
    """Optional per-task carrier config from fixtures/<name>/task.json. Path-valued fields
    (carrier_r_src, carrier_f_src, skill_src) are stored RELATIVE TO `HERE` (this file's
    directory) for portability across clones/worktrees, and resolved to absolute paths here.
    Returns {} if the file doesn't exist — every field then keeps discover_tasks's default."""
    if not os.path.exists(task_json_path):
        return {}
    data = json.load(open(task_json_path))
    resolved = dict(data)
    for key in ("carrier_r_src", "carrier_f_src", "skill_src"):
        if resolved.get(key):
            resolved[key] = os.path.normpath(os.path.join(HERE, resolved[key]))
    if "removal_basenames" in resolved:
        resolved["removal_basenames"] = tuple(resolved["removal_basenames"])
    return resolved


def discover_tasks(fixtures_dir=None):
    """Scan `fixtures_dir` (default FIXTURES_DIR) for task directories: any subdirectory
    containing all of FIXTURE_REQUIRED_FILES is a discovered task, keyed by its directory
    basename. Returns {dir_name: {init_script, done_when_script, task_prompt, steps_json,
    removal_basenames, carrier_r_src, carrier_f_src, skill_name, skill_src}} — the last five
    default to ()/None/None/None/None and are overridden by an adjacent task.json (see
    load_task_json)."""
    fixtures_dir = fixtures_dir or FIXTURES_DIR
    discovered = {}
    if not os.path.isdir(fixtures_dir):
        return discovered
    for name in sorted(os.listdir(fixtures_dir)):
        task_dir = os.path.join(fixtures_dir, name)
        if not os.path.isdir(task_dir):
            continue
        if not all(os.path.exists(os.path.join(task_dir, fname)) for fname in FIXTURE_REQUIRED_FILES):
            continue
        entry = {
            "init_script": os.path.join(task_dir, "init_fixture_repo.sh"),
            "done_when_script": os.path.join(task_dir, "done_when_checks.sh"),
            "task_prompt": os.path.join(task_dir, "task-prompt.txt"),
            "steps_json": os.path.join(task_dir, "steps.json"),
            "removal_basenames": (),
            "carrier_r_src": None,
            "carrier_f_src": None,
            "skill_name": None,
            "skill_src": None,
        }
        entry.update(load_task_json(os.path.join(task_dir, "task.json")))
        discovered[name] = entry
    return discovered


def _build_tasks_registry():
    """TASKS, keyed by canonical task key: resolve_task_key applied to each discovered fixture
    directory name — "commit"/"gitignore" collapse onto "A"/"B"; any other discovered directory
    keeps its own name as the key."""
    tasks = {}
    for dir_name, entry in discover_tasks().items():
        tasks[resolve_task_key(dir_name)] = entry
    return tasks


TASKS = _build_tasks_registry()

PLUMBING_SITUATION = {
    "A": "committing changes to this repo following the project's conventions",
    "B": "narrowing this repo's .gitignore so needed files are tracked without exposing generated artifacts",
}


# ----- CLAUDE.md construction (extends p1.build_claude_md with an optional Rdirect section) -----

def _note_body(md_path):
    """Strip YAML frontmatter, return the note body verbatim."""
    text = open(md_path).read()
    parts = text.split("---", 2)
    if text.startswith("---") and len(parts) >= 3:
        return parts[2].strip()
    return text.strip()


def _task_a_runbook_body():
    src_dir = TASKS["A"]["carrier_r_src"]
    for name in sorted(os.listdir(src_dir)):
        if name.endswith(".md"):
            return _note_body(os.path.join(src_dir, name))
    raise RuntimeError(f"no .md file found in {src_dir}")


def _task_b_830_body():
    return _note_body(os.path.join(REAL_VAULT, NOTE_830_BASENAME + ".md"))


def rdirect_procedure_text(task_key):
    """Verbatim runbook body for the arm-Rdirect '## Project procedure' section: the A-R note's
    body (Task A) or the real vault's 830 note body (Task B) — read-only access to the real vault,
    never a write (note 956: writes/activations during a fingerprinted run are the hazard, reads
    are not)."""
    return _task_a_runbook_body() if task_key == "A" else _task_b_830_body()


def build_claude_md_phase2(marker, extra=None):
    """p1.build_claude_md(marker) (guidance + fifth cue + marker), with an optional
    '## Project procedure' section appended verbatim for Arm Rdirect. Identical to phase-1's
    CLAUDE.md for every other arm."""
    base = p1.build_claude_md(marker)
    if not extra:
        return base
    return base.rstrip() + "\n\n## Project procedure\n\n" + extra.strip() + "\n"


# ----- background vault: real-vault copy, covering-note removal, carrier add (Ruling 3) -----

def remove_covering_notes(vault, task_key, arm):
    removal = list(TASKS[task_key]["removal_basenames"])
    if task_key == "B" and arm != "R":
        removal.append(NOTE_830_BASENAME)
    for base in removal:
        for ext in (".md", ".vec.json"):
            path = os.path.join(vault, base + ext)
            if os.path.exists(path):
                os.remove(path)


_LEADING_INT_RE = re.compile(r"^\d+")


def _leading_luhmann_number(basename):
    """The leading INTEGER component of a note basename's leading segment (before the first '.'):
    a Luhmann id is digits, then letters, then digits, ... (internal/luhmann's ParseID grammar —
    '988a' -> ['988', 'a'], '988a1' -> ['988', 'a', '1']), so '988a' and '988a1' both carry the
    leading integer 988. Returns None if that segment doesn't start with a digit at all — e.g.
    'qa.2026-...' notes carry no luhmann number and are never subject to the eval-session-notes
    exclusion rule. Bug fixed here: the prior `.isdigit()` check required the WHOLE segment to be
    digits, so any alpha-suffixed id (988a, 988a1, 988b) fell through as None and escaped the
    exclusion floor entirely, regardless of how far its integer component was above it."""
    first_segment = basename.split(".", 1)[0]
    match = _LEADING_INT_RE.match(first_segment)
    return int(match.group()) if match else None


def remove_eval_session_notes(vault, min_luhmann=EXCLUDE_LUHMANN_MIN):
    """Delete every note (+ .vec.json sidecar) whose leading luhmann number is >= min_luhmann —
    this eval session's own vault notes, which post-date the covering-note grep and mention the
    fixtures' own methods (see EXCLUDE_LUHMANN_MIN). Applied to EVERY arm and task. `qa.*` notes
    (no leading luhmann number) are never removed by this rule. Returns the count of notes
    removed (sidecars not counted separately)."""
    removed = 0
    for name in sorted(os.listdir(vault)):
        if not name.endswith(".md"):
            continue
        basename = name[: -len(".md")]
        luhmann_number = _leading_luhmann_number(basename)
        if luhmann_number is None or luhmann_number < min_luhmann:
            continue
        os.remove(os.path.join(vault, name))
        sidecar = os.path.join(vault, basename + ".vec.json")
        if os.path.exists(sidecar):
            os.remove(sidecar)
        removed += 1
    return removed


def add_carrier(vault, task_key, arm):
    """R/F: copy the arm's converted-note vault dir into the trial vault, returning the note's
    basename (no .md). Task B/R is special: 830 is the carrier and was already NOT deleted by
    remove_covering_notes, so nothing is copied — just report its basename. S/Rdirect: nothing
    added, returns None."""
    if arm not in ("R", "F"):
        return None
    if task_key == "B" and arm == "R":
        return NOTE_830_BASENAME
    cfg = TASKS[task_key]
    src_dir = cfg["carrier_r_src"] if arm == "R" else cfg["carrier_f_src"]
    basename = None
    for name in sorted(os.listdir(src_dir)):
        shutil.copy2(os.path.join(src_dir, name), os.path.join(vault, name))
        if name.endswith(".md"):
            basename = name[: -len(".md")]
    return basename


def _parse_embed_status(text):
    stats = {}
    for line in text.splitlines():
        m = re.match(r"^([\w-]+):\s*(\d+)\s*$", line.strip())
        if m:
            stats[m.group(1)] = int(m.group(2))
    return stats


def verify_vault_health(vault):
    """`engram embed status --vault <trial vault>` must report broken==0 and without==0 — fail
    the trial setup loudly otherwise (Ruling 3)."""
    env = dict(os.environ)
    env["PATH"] = p1.ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    r = subprocess.run(["engram", "embed", "status", "--vault", vault],
                        capture_output=True, text=True, env=env)
    stats = _parse_embed_status(r.stdout)
    broken, without = stats.get("broken"), stats.get("without")
    if broken != 0 or without != 0:
        raise RuntimeError(
            f"vault health check failed for {vault}: broken={broken} without={without}\n"
            f"stdout={r.stdout}\nstderr={r.stderr}"
        )
    return stats


def setup_trial_vault(env, task_key, arm, exclude_luhmann_min=EXCLUDE_LUHMANN_MIN):
    """Background vault for EVERY arm (Ruling 3): a per-trial copytree of the real vault, with
    the task's covering notes removed, this eval session's own notes removed (luhmann >=
    exclude_luhmann_min), then the arm's carrier added (R/F only). Returns (carrier_basename,
    copy_seconds)."""
    vault = env["ENGRAM_VAULT_PATH"]
    t0 = time.time()
    shutil.rmtree(vault, ignore_errors=True)
    shutil.copytree(REAL_VAULT, vault)
    copy_s = round(time.time() - t0, 2)
    remove_covering_notes(vault, task_key, arm)
    eval_notes_removed = remove_eval_session_notes(vault, exclude_luhmann_min)
    print(f"setup_trial_vault: removed {eval_notes_removed} eval-session note(s) "
          f"(luhmann >= {exclude_luhmann_min}) from {vault}")
    carrier_basename = add_carrier(vault, task_key, arm)
    verify_vault_health(vault)
    return carrier_basename, copy_s


# ----- trial repo setup (Ruling 5) -----

def _skill_body_with_name(text, skill_name):
    """`text` (a skill file's full content) with `name: <skill_name>` ensured in its YAML
    frontmatter — added only if no `name:` key is already present. Otherwise byte-identical,
    including the body."""
    if not text.startswith("---\n"):
        return text
    parts = text.split("---", 2)
    if len(parts) < 3:
        return text
    frontmatter = parts[1]
    if re.search(r"(?m)^name:\s*\S", frontmatter):
        return text
    frontmatter = frontmatter.rstrip("\n") + f"\nname: {skill_name}\n"
    return "---" + frontmatter + "---" + parts[2]


def deploy_skill(repo_path, task_key):
    cfg = TASKS[task_key]
    if task_key == "A":
        # Deployed as a DIRECTORY (<repo>/.claude/skills/commit/SKILL.md), not the flat file the
        # live repo carries at .claude/skills/commit.md — smoke-run-1 finding: Claude Code's own
        # skill listing did NOT discover the flat-file form the way it discovered the directory-
        # form gitignore-narrowing skill. Body is byte-identical to the repo's live
        # .claude/skills/commit.md; frontmatter is the file's own frontmatter plus `name: commit`
        # only if that key is absent (the live file already carries `name: commit`, so this is a
        # no-op today — kept for robustness if the source file changes). This packaging change is
        # the ONLY deviation from "live artifact unchanged" anywhere in this harness — see
        # task-3-report.md for why it was necessary (the skill must be discoverable to be FOUND).
        dst = os.path.join(repo_path, ".claude", "skills", "commit", "SKILL.md")
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        original = open(cfg["skill_src"]).read()
        with open(dst, "w") as f:
            f.write(_skill_body_with_name(original, cfg["skill_name"]))
    elif task_key == "B":
        dst = os.path.join(repo_path, ".claude", "skills", "gitignore-narrowing")
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copytree(cfg["skill_src"], dst)
    else:
        # Generic task: deploy skill directory with its actual name from cfg
        skill_name = cfg.get("skill_name")
        if skill_name:
            dst = os.path.join(repo_path, ".claude", "skills", skill_name)
            os.makedirs(os.path.dirname(dst), exist_ok=True)
            shutil.copytree(cfg["skill_src"], dst)


_START_STATE_CHECKS = {
    "A": (
        (re.compile(r"^ M pkg/version\.go$", re.MULTILINE), "unstaged ' M pkg/version.go'"),
        (re.compile(r"^\?\? notes/", re.MULTILINE), "untracked notes/"),
    ),
    "B": (
        (re.compile(r"^\?\? tmp\.log$", re.MULTILINE), "untracked tmp.log"),
    ),
}
_START_STATE_IGNORED_PATHS = {
    # Task B's fixture init (gitignore/init_fixture_repo.sh, commit b098afa1 — the round-2 fix
    # for ce8d9a73's regression) commits ONLY .gitignore + src/main.go; scripts/build.sh and
    # testdata/fixture.json are covered by the over-broad .gitignore and stay ignored+untracked,
    # matching done_when_checks.sh's "staged" checks (satisfiable: they have real content to
    # stage). testdata/generated/big.bin is the generated artifact created after the fixture
    # commit and must also stay ignored throughout.
    "B": ("testdata/generated/big.bin", "scripts/build.sh", "testdata/fixture.json"),
}
_START_STATE_UNTRACKED_PATHS = {
    # These must be absent from the index (never committed) at trial start — the whole point of
    # the task is to make them trackable, so the fixture must not pre-track them.
    "B": ("scripts/build.sh", "testdata/fixture.json"),
}


def assert_starting_state(repo_path, task_key):
    """Verify committing CLAUDE.md (+ skill) did not sweep the fixture's decoy/unstaged state."""
    status = subprocess.run(["git", "-C", repo_path, "status", "--porcelain"],
                             capture_output=True, text=True, check=True).stdout
    for pattern, desc in _START_STATE_CHECKS.get(task_key, ()):
        if not pattern.search(status):
            raise RuntimeError(
                f"Task {task_key} starting-state check failed: expected {desc}, got:\n{status}"
            )
    for rel_path in _START_STATE_IGNORED_PATHS.get(task_key, ()):
        r = subprocess.run(["git", "-C", repo_path, "check-ignore", "-q", rel_path])
        if r.returncode != 0:
            raise RuntimeError(
                f"Task {task_key} starting-state check failed: expected {rel_path} to be ignored"
            )
    for rel_path in _START_STATE_UNTRACKED_PATHS.get(task_key, ()):
        tracked = subprocess.run(["git", "-C", repo_path, "ls-files", "--", rel_path],
                                  capture_output=True, text=True, check=True).stdout.strip()
        if tracked:
            raise RuntimeError(
                f"Task {task_key} starting-state check failed: expected {rel_path} to be "
                f"untracked (absent from the index), but git ls-files reports it tracked"
            )


def setup_trial_repo(trial_dir, task_key, arm, marker):
    cfg = TASKS[task_key]
    repo_path = os.path.join(trial_dir, "repo")
    subprocess.run(["bash", cfg["init_script"], repo_path], check=True, capture_output=True, text=True)

    extra = rdirect_procedure_text(task_key) if arm == "Rdirect" else None
    with open(os.path.join(repo_path, "CLAUDE.md"), "w") as f:
        f.write(build_claude_md_phase2(marker, extra=extra))
    if arm == "S":
        deploy_skill(repo_path, task_key)

    add_paths = ["CLAUDE.md"]
    if os.path.isdir(os.path.join(repo_path, ".claude")):
        add_paths.append(".claude")
    subprocess.run(["git", "-C", repo_path, "add"] + add_paths, check=True, capture_output=True)
    subprocess.run(["git", "-C", repo_path, "commit", "-m", "add project config"],
                    check=True, capture_output=True)

    assert_starting_state(repo_path, task_key)
    return repo_path


# ----- FOUND (Ruling 6) -----

def is_first_mutating_step(event, task_key):
    if event["kind"] != "tool_use":
        return False
    name = event.get("name")
    if name in ("Edit", "Write", "MultiEdit"):
        return True
    if name == "Bash":
        command = (event.get("input") or {}).get("command", "") or ""
        stripped = _strip_quoted(command)
        if _ENGRAM_READONLY_RE.match(stripped):
            return False
        return bool(TASK_MUTATING_BASH_RE.get(task_key, _MUTATING_BASH_RE).search(stripped))
    return False


def first_mutating_step_index(events, task_key):
    for ev in events:
        if is_first_mutating_step(ev, task_key):
            return ev["idx"]
    return None


def score_found_phase2(task_key, arm, events, carrier_basename):
    """Rdirect: n/a (no retrieval attempted; marker_seen is the delivery check) — returns
    (None, None). N (no-instructions control): n/a for the same reason — there is no carrier to
    find, by design. S: a Skill tool_use naming the arm's skill before the first mutating step.
    R/F: a Bash `engram query` before the first mutating step whose tool_result contains the
    arm's carrier basename."""
    if arm in ("Rdirect", "N"):
        return None, None

    first_idx = first_mutating_step_index(events, task_key)

    def before(idx):
        return first_idx is None or idx < first_idx

    if arm == "S":
        skill_name = TASKS[task_key]["skill_name"]
        for ev in events:
            if (ev["kind"] == "tool_use" and ev.get("name") == "Skill"
                    and (ev.get("input") or {}).get("skill") == skill_name and before(ev["idx"])):
                return True, ev["idx"]
        return False, None

    # Arm R / Arm F
    for ev in events:
        if not (ev["kind"] == "tool_use" and ev.get("name") == "Bash"
                and "engram query" in ((ev.get("input") or {}).get("command", "") or "")):
            continue
        if not before(ev["idx"]):
            continue
        result_text = p1._tool_result_text(events, ev.get("id"))
        if carrier_basename and carrier_basename in result_text:
            return True, ev["idx"]
    return False, None


def found_method(arm, found):
    if arm in ("Rdirect", "N"):
        return "n/a"
    if not found:
        return "none"
    return "Skill tool_use" if arm == "S" else "engram query"


# ----- FOLLOWED: steps.json evaluation (Ruling 6) -----

def load_steps(task_key):
    return json.load(open(TASKS[task_key]["steps_json"]))


def _check_commit_message_format(repo_path):
    """Task A step 5 ('commit_message_format' repo_state signal): conventional-commit subject
    form (`<type>(<scope>): ` or `<type>: `) plus a non-empty body line — NOT the trailer.

    Round-4 ruling (Joe): the trailer (AI-Used: [claude] vs. Co-Authored-By:) is REPORTED, not
    SCORED, here — the harness's own commit-attribution injection (task-3-report.md's BLOCKED
    finding) overrides whatever trailer a carrier/skill instructs, identically across every arm,
    making trailer content an uninformative pass/fail signal for this detector. See
    `classify_trailer` for the reported (not scored) field. Task 2's thread is removing the
    trailer requirement from done_when_checks.sh/steps.json in parallel (not touched here)."""
    r = subprocess.run(["git", "-C", repo_path, "log", "-1", "--format=%B"],
                        capture_output=True, text=True)
    msg = r.stdout
    if not msg.strip():
        return False
    if not re.match(r"^[a-z]+(\([a-zA-Z0-9/_-]+\))?:", msg):
        return False
    lines = msg.splitlines()
    body_lines = [line for line in lines[1:] if line.strip()]
    return bool(body_lines)


def classify_trailer(repo_path):
    """Task A only: classify the newest commit's trailer as 'ai_used' | 'co_authored' | 'both' |
    'none' — REPORTED, not scored (round-4 ruling). Never affects FOUND/FOLLOWED/END-STATE."""
    r = subprocess.run(["git", "-C", repo_path, "log", "-1", "--format=%B"],
                        capture_output=True, text=True)
    msg = r.stdout
    has_ai_used = bool(re.search(r"(?m)^AI-Used:\s*\[claude\]\s*$", msg))
    has_co_authored = bool(re.search(r"(?m)^Co-Authored-By:", msg))
    if has_ai_used and has_co_authored:
        return "both"
    if has_ai_used:
        return "ai_used"
    if has_co_authored:
        return "co_authored"
    return "none"


_BARE_TESTDATA_RE = re.compile(r"^testdata/$")
_NARROWED_TESTDATA_RE = re.compile(r"^(\*\*/)?testdata/generated/?$")
_BARE_SCRIPTS_RE = re.compile(r"^scripts/$")


def _check_gitignore_narrowed_to_generated(repo_path):
    """Task B step 3 ('gitignore_narrowed_to_generated' repo_state signal): the .gitignore no
    longer over-broadly ignores all of testdata/ or all of scripts/ — narrowed to just the
    generated artifacts directory. True iff:
      (1) no bare 'testdata/' line remains,
      (2) a line matches '^(**/)?testdata/generated/?$' (the narrowed target), and
      (3) the bare 'scripts/' line is gone (removed or narrowed to something more specific)."""
    path = os.path.join(repo_path, ".gitignore")
    try:
        lines = [line.strip() for line in open(path).read().splitlines()]
    except OSError:
        return False
    has_bare_testdata = any(_BARE_TESTDATA_RE.match(line) for line in lines)
    has_narrowed_testdata = any(_NARROWED_TESTDATA_RE.match(line) for line in lines)
    has_bare_scripts = any(_BARE_SCRIPTS_RE.match(line) for line in lines)
    return (not has_bare_testdata) and has_narrowed_testdata and (not has_bare_scripts)


_RAPID_PATHS_NESTED = (
    "internal/core/testdata/rapid/big.bin",
    "internal/api/testdata/rapid/big.bin",
    "testdata/rapid/big.bin",
)
_FIXTURE_JSON_PATHS_NESTED = (
    "internal/core/testdata/fixture.json",
    "internal/api/testdata/fixture.json",
    "testdata/fixture.json",
)


def _check_gitignore_rapid_ignored_fixture_trackable_all_depths(repo_path):
    """gitignore-nested task step 3 ('gitignore_rapid_ignored_fixture_trackable_all_depths'
    repo_state signal): the CURRENT .gitignore, evaluated with the real `git check-ignore` engine
    (not a line-pattern regex), must ignore all three nested rapid/ generated-data paths and must
    NOT ignore any of the three fixture.json paths. Using the real ignore engine means a
    middle-slash pattern that only anchors at the .gitignore's own directory (runbook 830 step 3)
    correctly fails this check the moment it stops matching a nested depth."""
    for path in _RAPID_PATHS_NESTED:
        r = subprocess.run(["git", "-C", repo_path, "check-ignore", "-q", path])
        if r.returncode != 0:
            return False
    for path in _FIXTURE_JSON_PATHS_NESTED:
        r = subprocess.run(["git", "-C", repo_path, "check-ignore", "-q", path])
        if r.returncode == 0:
            return False
    return True


def _check_test_bites(repo_path):
    """test-bite task step 4 ('test_bites' repo_state signal): the agent's
    tests/test_run_child.py must pass against the shipped (correct) runner.py AND fail once
    run_child's env_extra forwarding is mutated out -- i.e. the assertion depends on the effect
    itself (runbook 838), not merely 'no exception was raised'. The mutation is applied to a
    scratch copy so the live trial repo is never modified."""
    test_file = os.path.join(repo_path, "tests", "test_run_child.py")
    if not os.path.exists(test_file):
        return False

    shipped = subprocess.run(
        ["python3", "-m", "pytest", "-q", "tests/test_run_child.py"],
        cwd=repo_path, capture_output=True, text=True,
    )
    if shipped.returncode != 0:
        return False

    with tempfile.TemporaryDirectory() as scratch:
        mutant = os.path.join(scratch, "repo")
        shutil.copytree(repo_path, mutant, ignore=shutil.ignore_patterns(".git"))
        runner_path = os.path.join(mutant, "runner.py")
        with open(runner_path) as f:
            src = f.read()
        mutated = src.replace("env.update(env_extra)", "pass  # forwarding disabled for bite-check")
        if mutated == src:
            return False  # mutation target not found -- can't assert the test bites
        with open(runner_path, "w") as f:
            f.write(mutated)
        mutant_run = subprocess.run(
            ["python3", "-m", "pytest", "-q", "tests/test_run_child.py"],
            cwd=mutant, capture_output=True, text=True,
        )
    return mutant_run.returncode != 0


def _tdd_order_relevant_commits(repo_path, orig_tip):
    """Commits strictly after `orig_tip` (fixture baseline) up to HEAD that touch at least one
    .py file, oldest first. The harness's own 'add project config' commit (CLAUDE.md/.claude only)
    never touches a .py file and is never included."""
    r = subprocess.run(
        ["git", "-C", repo_path, "rev-list", "--reverse", f"{orig_tip}..HEAD"],
        capture_output=True, text=True,
    )
    if r.returncode != 0:
        return []
    relevant = []
    for commit in r.stdout.split():
        show = subprocess.run(
            ["git", "-C", repo_path, "show", "--name-only", "--pretty=format:", commit],
            capture_output=True, text=True,
        )
        files = [f for f in show.stdout.splitlines() if f.strip()]
        if any(f.endswith(".py") for f in files):
            relevant.append(commit)
    return relevant


_TDD_ORDER_TEST_FILE_RE = re.compile(r"(^|/)test_[^/]*\.py$|(^|/)[^/]*_test\.py$")


def _tdd_order_is_test_file(path):
    """Mirrors done_when_checks.sh's `is_test_file()` shell case pattern exactly: a .py file whose
    basename either starts with 'test_' or ends with '_test.py', at any directory depth
    (test_*.py, */test_*.py, *_test.py, */*_test.py)."""
    return bool(_TDD_ORDER_TEST_FILE_RE.search(path))


def _tdd_order_commit_py_files(repo_path, commit):
    show = subprocess.run(
        ["git", "-C", repo_path, "show", "--name-only", "--pretty=format:", commit],
        capture_output=True, text=True,
    )
    return [f for f in show.stdout.splitlines() if f.strip().endswith(".py")]


def _tdd_order_commit_mixes_test_and_impl(repo_path, commit):
    """done_when_checks.sh Check 3: a relevant commit that touches BOTH a test file and a
    non-test .py file (implementation) fails the signal."""
    py_files = _tdd_order_commit_py_files(repo_path, commit)
    has_test = any(_tdd_order_is_test_file(f) for f in py_files)
    has_impl = any(not _tdd_order_is_test_file(f) for f in py_files)
    return has_test and has_impl


def _tdd_order_commit_is_test_only(repo_path, commit):
    """done_when_checks.sh Check 4's file-classification half: the commit's .py files exist and
    are ALL test files (empty/no .py files never counts as test-only)."""
    py_files = _tdd_order_commit_py_files(repo_path, commit)
    return bool(py_files) and all(_tdd_order_is_test_file(f) for f in py_files)


def _check_tdd_order_test_only_commit_precedes_impl(repo_path):
    """tdd-order task step 5 ('tdd_order_test_only_commit_precedes_impl' repo_state signal):
    mirrors fixtures/tdd-order/done_when_checks.sh Checks 3 and 4, minus Check 4's pytest-at-SHA
    RED verification (which requires checking out a scratch worktree and running the project's own
    suite there -- out of scope for a repo_state file-classification signal). True iff:
      (a) at least two commits since the fixture's baseline tip (recorded in .eval/original_tip, a
          sibling of the trial repo, at init time -- see fixtures/tdd-order/init_fixture_repo.sh)
          touch a .py file (Check 2, via _tdd_order_relevant_commits -- the harness's own
          CLAUDE.md-only commit never counts toward the total);
      (b) no relevant commit touches BOTH a test and an implementation .py file (Check 3); and
      (c) the FIRST relevant commit touches ONLY test file(s) (Check 4's file-classification half).
    Renamed from tdd_order_at_least_two_new_commits (count-only; a bare-agent baseline batch found
    4/8 trials whose final commit mixed test and implementation .py files, or led with an
    over-implemented first commit, yet still scored this step as followed under the old,
    count-only name -- see vault/route-dispatch notes on the phase-2 tdd-order rescore)."""
    eval_dir = os.path.join(os.path.dirname(os.path.abspath(repo_path)), ".eval")
    orig_tip_path = os.path.join(eval_dir, "original_tip")
    try:
        with open(orig_tip_path) as f:
            orig_tip = f.read().strip()
    except OSError:
        return False
    relevant = _tdd_order_relevant_commits(repo_path, orig_tip)
    if len(relevant) < 2:
        return False
    if any(_tdd_order_commit_mixes_test_and_impl(repo_path, c) for c in relevant):
        return False
    return _tdd_order_commit_is_test_only(repo_path, relevant[0])


REPO_STATE_CHECKERS = {
    # Task 2's thread renamed the fixtures/commit/steps.json pattern to
    # "commit_message_format_and_body" (round-4 ruling: trailer requirement removed from the
    # checker's name as well as its behavior) — registered under that name to match.
    "commit_message_format_and_body": _check_commit_message_format,
    "gitignore_narrowed_to_generated": _check_gitignore_narrowed_to_generated,
    "gitignore_rapid_ignored_fixture_trackable_all_depths": _check_gitignore_rapid_ignored_fixture_trackable_all_depths,
    "test_bites": _check_test_bites,
    "tdd_order_test_only_commit_precedes_impl": _check_tdd_order_test_only_commit_precedes_impl,
}


def default_repo_checker(pattern_name, repo_path):
    fn = REPO_STATE_CHECKERS.get(pattern_name)
    if fn is None:
        raise ValueError(f"no repo_state checker registered for {pattern_name!r}")
    return fn(repo_path)


def _evaluate_signal(sig, events, bash_events, repo_path, repo_checker, min_idx, min_pos=None):
    """Evaluate ONE signal definition against the trial's events/repo state. Returns
    (matched: bool, idx: int|None, pos: int|None) — `pos` is the regex match's start offset
    within the matched command string (bash_regex only; None otherwise), used to order two
    signals that land in the SAME Bash event (see `evaluate_steps`'s `after` docstring).

    bash_regex: `pattern` matched against Bash tool_use commands, in transcript order; an
      optional `not_pattern` on the SAME command disqualifies a match (the "not -A" rule).
    tool_path: `tools` (a list of tool names, e.g. ["Read", "Edit", "Write"]) + `pattern` matched
      against that tool_use's `file_path` input — credits a step satisfiable via a native
      file tool (Read/Edit/Write) that a bash_regex alone can never see (round-5 finding: an
      agent that inspects .gitignore via the native Read tool, rather than `cat .gitignore`,
      got no credit for an "inspect the file" step).
    repo_state: `pattern` names a key in `repo_checker`'s registry, called with repo_path (no
      event index — repo_state steps carry no ordering point for `after`).
    """
    signal = sig["signal"]

    if signal == "bash_regex":
        pattern = re.compile(sig["pattern"])
        not_pattern = re.compile(sig["not_pattern"]) if sig.get("not_pattern") else None
        for ev in bash_events:
            if ev["idx"] < min_idx:
                continue
            command = (ev.get("input") or {}).get("command", "") or ""
            if not_pattern and not_pattern.search(command):
                continue
            if ev["idx"] == min_idx:
                # Same Bash event that satisfied the referenced `after` step (final-review
                # finding: a single command like
                # `git add ... && git diff --cached --name-only && git status --short` stages
                # AND verifies in one call) — only credit this step if SOME occurrence of its
                # pattern in this command starts STRICTLY AFTER the referenced step's match
                # position. Round-2 bug (second final-review finding): using only the FIRST
                # (leftmost) match via `pattern.search` misses a later, legitimate verify match
                # when an EARLIER, unrelated occurrence of the same pattern also appears in the
                # command before the referenced step's own match — e.g.
                # `git status --porcelain -uall && ... && git add <paths> && ... && git diff
                # --cached --name-only && ... && git status --short`: the verify step's pattern
                # (`git (diff --cached|status)`) first matches the LEADING `git status` (before
                # the `git add`), so `search` never sees the later, genuinely-after `git diff
                # --cached`/`git status --short` occurrence. Scan every match via `finditer` and
                # take the first one whose start position is after the referenced step's match.
                # `min_pos is None` means the referenced step's own match position is unknown
                # (e.g. it matched via `tool_path`/`repo_state`, not `bash_regex`) — in that case
                # a same-event match can never be ordered, so it does not count.
                if min_pos is None:
                    continue
                match = next((m for m in pattern.finditer(command) if m.start() > min_pos), None)
                if match is None:
                    continue
            else:
                match = pattern.search(command)
                if not match:
                    continue
            return True, ev["idx"], match.start()
        return False, None, None

    if signal == "tool_path":
        pattern = re.compile(sig["pattern"])
        tools = set(sig.get("tools") or ())
        for ev in events:
            if ev["kind"] != "tool_use" or ev.get("name") not in tools:
                continue
            if ev["idx"] <= min_idx:
                continue
            file_path = (ev.get("input") or {}).get("file_path", "") or ""
            if not pattern.search(file_path):
                continue
            return True, ev["idx"], None
        return False, None, None

    if signal == "tool_input_regex":
        tool_name = sig.get("tool")
        regex_pattern = sig.get("regex")
        if not tool_name or not regex_pattern:
            raise ValueError(f"tool_input_regex signal requires 'tool' and 'regex' keys")
        flags = re.IGNORECASE if sig.get("case_insensitive") else 0
        regex = re.compile(regex_pattern, flags)
        for ev in events:
            if ev["idx"] <= min_idx:
                continue
            if ev["kind"] != "tool_use" or ev.get("name") != tool_name:
                continue
            input_dict = ev.get("input") or {}
            input_json = json.dumps(input_dict)
            match = regex.search(input_json)
            if match:
                return True, ev["idx"], None
        return False, None, None

    if signal == "repo_state":
        return bool(repo_checker(sig["pattern"], repo_path)), None, None

    raise ValueError(f"unknown steps.json signal {signal!r}")


def evaluate_steps(steps, events, repo_path, repo_checker=default_repo_checker):
    """Evaluate a task's steps.json against a trial's transcript events + final repo state.

    A step is either a single signal (top-level `signal`/`pattern`/[`not_pattern`]/[`tools`]
    keys — see `_evaluate_signal`) or an `any_of` list of alternative signal definitions, any ONE
    of which satisfies the step (e.g. Task B step 1: inspecting .gitignore via `cat`/`grep`/etc.
    in Bash, OR via the native Read/Edit/Write tool — either counts). An optional `after: <step n>`
    (bash_regex only for the same-event case below; tool_path/repo_state referenced-step
    positions are always None, so a same-event match against those can never be ordered)
    requires the matching event to satisfy ONE of:
      (a) occur at a STRICTLY LATER event index than the event that satisfied step n, or
      (b) occur in the SAME Bash event as step n's match, with this step's own regex match
          starting at a LATER position in the command string than step n's match did
          (final-review finding: a single compound command — e.g.
          `git add X && git diff --cached --name-only && git status --short` — can legitimately
          satisfy a "stage" step and a "verify" step at once; requiring a strictly later EVENT
          would wrongly fail the verify step even though it demonstrably followed the stage
          clause within that same command).

    **If step n (the referenced step) never matched, the dependent `after: n` step is FALSE**,
    regardless of what its own pattern would otherwise match (second final-review ruling: a
    "verify" step with no matched "stage" step to verify AFTER is not the procedure — treating an
    unmatched reference as `min_idx=-1` would make `after` vacuous, letting the dependent step
    match anywhere in the transcript as if there were no ordering constraint at all).

    Returns (results: {n: bool}, followed_k: int, followed_all: bool).
    """
    results = {}
    match_idx = {}
    match_pos = {}
    bash_events = [ev for ev in events if ev["kind"] == "tool_use" and ev.get("name") == "Bash"]

    for step in sorted(steps, key=lambda s: s["n"]):
        n = step["n"]
        after_n = step.get("after")

        if after_n is not None and after_n not in match_idx:
            # The referenced step never matched — this step cannot be "after" something that
            # never happened. Do not evaluate its own signal at all.
            results[str(n)] = False
            continue

        min_idx = match_idx.get(after_n, -1) if after_n is not None else -1
        min_pos = match_pos.get(after_n) if after_n is not None else None

        sub_signals = step["any_of"] if "any_of" in step else [step]
        matched = False
        idx = None
        pos = None
        for sig in sub_signals:
            sig_matched, sig_idx, sig_pos = _evaluate_signal(sig, events, bash_events, repo_path,
                                                               repo_checker, min_idx, min_pos)
            if sig_matched:
                matched, idx, pos = True, sig_idx, sig_pos
                break

        results[str(n)] = matched
        if idx is not None:
            match_idx[n] = idx
            match_pos[n] = pos

    followed_k = sum(1 for v in results.values() if v)
    followed_all = followed_k == len(steps)
    return results, followed_k, followed_all


# ----- END-STATE -----

def check_end_state_phase2(task_key, repo_path, env=None):
    r = subprocess.run(["bash", TASKS[task_key]["done_when_script"], repo_path],
                        capture_output=True, text=True, env=env)
    return r.returncode == 0, (r.stdout + r.stderr).strip()


# ----- rate-limited stub detection (vault note 988a) -----

# A trial whose spawned `claude -p` session hit the ACCOUNT's session limit produces a transcript
# carrying a terminal assistant-message record with `"error":"rate_limit"`,
# `"isApiErrorMessage":true`, `"apiErrorStatus":429`, and content text "You've hit your session
# limit · resets <time> (<tz>)" — verified against the real compact-JSON transcript bytes in
# results/baseline_sonnet5_opsx-archive.rate-limited.jsonl (all 8 records) and the last 6 records
# of results/baseline_sonnet5_opsx-propose.rate-limited.jsonl. No inter-key whitespace is present
# (Claude Code writes compact JSON), so the regexes below tolerate optional whitespace around `:`
# without requiring it.
_RATE_LIMIT_ERROR_RE = re.compile(r'"error"\s*:\s*"rate_limit"')
_RATE_LIMIT_STATUS_RE = re.compile(r'"apiErrorStatus"\s*:\s*429\b')
_RATE_LIMIT_MESSAGE_RE = re.compile(r"hit your session limit", re.IGNORECASE)
_RATE_LIMIT_IS_API_ERROR_RE = re.compile(r'"isApiErrorMessage"\s*:\s*true')

# Both invalid_reason values classify_validity can emit for a rate-limit-related outage: the
# zero-work stub ("rate_limit") and a truncation after real work ("rate_limit_truncated"). Every
# summary surface (format_baseline_summary, aggregate) counts both together in the single
# ' (rate-limited: K)' suffix — an outage is an outage either way, and the distinction between the
# two only matters for classify_validity's own reasoning, not for the roll-up count.
_RATE_LIMIT_INVALID_REASONS = ("rate_limit", "rate_limit_truncated")


def _rate_limit_signal_present(result, raw_text):
    """True when either the `claude -p --output-format json` result object or the trial's raw
    transcript text carries the account session-limit condition. `result` is checked defensively
    at the top level (`error`/`apiErrorStatus` keys) in case a future CLI version surfaces them
    there too, but detection never depends on it — every confirmed real sample carries the signal
    in the transcript text, which is also all `--rescore` (no live claude call) has to work with."""
    if isinstance(result, dict):
        if result.get("error") == "rate_limit" or result.get("apiErrorStatus") == 429:
            return True
    if raw_text and (_RATE_LIMIT_ERROR_RE.search(raw_text) or _RATE_LIMIT_STATUS_RE.search(raw_text)
                      or _RATE_LIMIT_MESSAGE_RE.search(raw_text)):
        return True
    return False


def _rate_limit_truncation_signal_present(result, raw_text):
    """Stricter than `_rate_limit_signal_present` — REQUIRED for a mid-task truncation check that
    runs regardless of num_turns/total_cost_usd (unlike `is_rate_limited_stub`, which is safely
    gated to the zero-turn/zero-cost shape and can afford the loose OR-of-three-signals check).
    A trial's background vault carries real memory notes, including ones that NARRATE a past
    rate-limit incident in English prose (e.g. vault note 988a's own body: "...a session that hit
    the account's 5-hour limit ('You've hit your session limit · resets 1pm', transcript error
    rate_limit / apiErrorStatus 429, 1 turn, $0.00)..."). When an agent surfaces that note via
    `engram query` during a genuinely successful, fully-completed trial, its transcript's raw text
    contains the words "hit your session limit" — `_rate_limit_signal_present` alone would
    misclassify that trial as truncated (confirmed: 2/8 trials in
    results/baseline_sonnet5_opsx-propose.jsonl, both 30+ turns and real cost with
    end_state=True/followed_all=True, false-positived this way before this stricter check was
    added).

    The one shape that ONLY the real system-generated error record carries — never prose ABOUT
    it — is `"isApiErrorMessage":true` co-occurring with `"apiErrorStatus":429` on the SAME
    transcript line (Claude Code writes one compact-JSON record per line; verified against the
    real byte-for-byte shape in every record of results/baseline_sonnet5_opsx-{propose,archive}
    .rate-limited.jsonl). `result`'s top-level dict fields are also honored (a future CLI version
    may surface them there instead of only in the transcript)."""
    if isinstance(result, dict) and result.get("apiErrorStatus") == 429 and result.get("isApiErrorMessage"):
        return True
    if not raw_text:
        return False
    return any(_RATE_LIMIT_STATUS_RE.search(line) and _RATE_LIMIT_IS_API_ERROR_RE.search(line)
               for line in raw_text.splitlines())


def is_rate_limited_stub(num_turns, total_cost_usd, result, raw_text):
    """A trial is a rate-limited STUB — invalid, never scored — only when the session did NO real
    work (num_turns <= 1 and total_cost_usd == 0.0) AND the rate-limit signal is present. The
    rate-limit signal alone is NOT sufficient: a real trial can run many turns and spend real
    money before tripping the account limit on a final wrap-up turn (verified: the first two
    records of results/baseline_sonnet5_opsx-propose.rate-limited.jsonl show 25/18 turns and
    $0.667/$0.4975 spent, with the identical rate-limit text present near the transcript's tail —
    those are real completed trials and must stay valid/scored, not be discarded as outage stubs).
    Only the zero-turn/zero-cost stub shape combined with the rate-limit signal is treated as an
    outage — a bare zero-turn/zero-cost trial with NO rate-limit signal is left alone (no other
    field in the real records distinguishes that general case cleanly, so it is out of scope
    here)."""
    if not _rate_limit_signal_present(result, raw_text):
        return False
    turns_zero_or_one = (num_turns or 0) <= 1
    cost_zero = not (total_cost_usd or 0.0)
    return turns_zero_or_one and cost_zero


def classify_validity(marker_seen, num_turns, total_cost_usd, result, raw_text):
    """(valid, invalid_reason) shared by the live scoring path (run_one_trial_phase2) and
    --rescore, so a trial is judged the same way whether it was just spawned or is being
    re-scored from a kept run dir. A rate-limited stub (see `is_rate_limited_stub`) is NEVER
    valid, regardless of `marker_seen` — the trial repo's CLAUDE.md carries the PROBE-TOKEN marker
    by construction (it's committed before claude is ever spawned), so a stub session that hit the
    account's session limit before doing any real work still shows marker_seen=True. That is
    exactly what let 22 session-limit outages score as real failures (found=0/N, end_state=FAIL)
    instead of being excluded as an outage — vault note 988a.

    A SECOND, distinct outage shape (988a's 2026-09-12 follow-up): a session that did substantial
    real work (many turns, real cost) but was cut off mid-task when the account's session limit
    hit on a LATER turn — the rate-limit signal is present, but `is_rate_limited_stub` correctly
    says False (real work happened, so it is not a zero-work stub). Such a trial never got the
    chance to finish, so scoring its FOLLOWED/END-STATE would misrepresent a truncation as a
    genuine failure — the first two records of
    results/baseline_sonnet5_opsx-propose.rate-limited.jsonl (25/18 turns, $0.667/$0.4975 spent,
    scored followed 3/7 and 2/7) were exactly this shape and were nearly reported as real
    failures. This is marked invalid too, with a DISTINCT invalid_reason
    ('rate_limit_truncated') so it is never conflated with the zero-work stub reason
    ('rate_limit') — both are surfaced together in the ' (rate-limited: K)' summary suffix.

    The truncation check uses `_rate_limit_truncation_signal_present`, NOT the looser
    `_rate_limit_signal_present` used by `is_rate_limited_stub` — unlike the zero-turn/zero-cost
    stub shape (where a loose text match is safe), a truncation check with no turn/cost gate must
    not fire on a genuinely completed trial whose background vault surfaced a note NARRATING a
    past rate-limit incident in prose (see `_rate_limit_truncation_signal_present`'s docstring)."""
    if is_rate_limited_stub(num_turns, total_cost_usd, result, raw_text):
        return False, "rate_limit"
    if _rate_limit_truncation_signal_present(result, raw_text):
        return False, "rate_limit_truncated"
    if not marker_seen:
        return False, "no_marker"
    return True, None


# ----- one trial -----

def _score_trial(task_key, arm, events, repo_path, carrier_basename, env=None):
    """Run all trial scoring (FOUND, recall_fired, FOLLOWED, END-STATE) and NEVER raise — an
    exception here is caught and recorded as `scoring_error`, with safe defaults for the rest of
    the fields. Round-1 review finding: an unhandled exception in scoring propagates through the
    ThreadPoolExecutor future in run_batch's `for fut in cf.as_completed(futs): record =
    fut.result()` loop, aborting the WHOLE batch and losing every sibling trial's already-scored
    result that hadn't been collected yet — one bad trial (e.g. an unregistered repo_state
    pattern) must not cost the rest of the run. `load_steps` itself is inside the try (round-2
    review: it can raise too — a missing/invalid steps.json must not break the "never raises"
    contract either), so n_steps defaults to 0 until steps are loaded successfully."""
    scored = {
        "found": None, "found_method": found_method(arm, None), "found_index": None,
        "first_procedure_step_index": None,
        "recall_fired": False,
        "followed_steps": {}, "followed_k": 0, "followed_all": False, "n_steps": 0,
        "end_state": False, "end_state_output": "",
        "trailer": "n/a",
        "scoring_error": None,
    }
    try:
        steps = load_steps(task_key)
        scored["n_steps"] = len(steps)
        found, found_idx = score_found_phase2(task_key, arm, events, carrier_basename)
        scored["found"] = found
        scored["found_method"] = found_method(arm, found)
        scored["found_index"] = found_idx
        scored["first_procedure_step_index"] = first_mutating_step_index(events, task_key)
        scored["recall_fired"] = p1.score_recall_fired(events)
        followed_steps, followed_k, followed_all = evaluate_steps(steps, events, repo_path)
        scored["followed_steps"] = followed_steps
        scored["followed_k"] = followed_k
        scored["followed_all"] = followed_all
        end_state, end_state_output = check_end_state_phase2(task_key, repo_path, env=env)
        scored["end_state"] = end_state
        scored["end_state_output"] = end_state_output
        if task_key == "A":
            scored["trailer"] = classify_trailer(repo_path)
    except Exception as exc:  # noqa: BLE001 — record and let the trial (and batch) continue
        scored["scoring_error"] = str(exc)
    return scored


def run_one_trial_phase2(run_root, cfg_dir, task_key, arm, model, trial_index, marker, timeout_s,
                          exclude_luhmann_min=EXCLUDE_LUHMANN_MIN):
    trial_dir = os.path.join(run_root, "trials", f"{task_key}-{arm}-{trial_index}")
    os.makedirs(trial_dir, exist_ok=True)
    t0 = time.time()

    repo_path = setup_trial_repo(trial_dir, task_key, arm, marker)
    env = trial_env_phase2(cfg_dir, trial_dir, repo_path)
    carrier_basename, vault_copy_s = setup_trial_vault(env, task_key, arm, exclude_luhmann_min)

    prompt = open(TASKS[task_key]["task_prompt"]).read().strip()

    result, timed_out = {}, False
    error = None
    try:
        result, timed_out = p1.spawn_claude(env, model, repo_path, prompt, timeout_s)
    except Exception as exc:  # noqa: BLE001 — record and keep scoring repo state
        error = str(exc)

    transcript_paths = p1.discover_transcript_paths(cfg_dir, repo_path)
    raw_text = p1.transcript_raw_text(transcript_paths)
    events = p1.parse_transcript_events(transcript_paths)

    marker_seen = p1.is_marker_seen(raw_text, marker)
    scored = _score_trial(task_key, arm, events, repo_path, carrier_basename, env=env)
    if scored["scoring_error"]:
        scoring_msg = f"scoring exception: {scored['scoring_error']}"
        error = f"{error}; {scoring_msg}" if error else scoring_msg

    total_cost_usd = p1._call_cost(result)
    num_turns = result.get("num_turns") if isinstance(result, dict) else None
    valid, invalid_reason = classify_validity(marker_seen, num_turns, total_cost_usd, result, raw_text)

    record = {
        "task": task_key, "arm": arm, "trial": trial_index, "model": model,
        "trial_dir": trial_dir, "repo_path": repo_path,
        "transcript_path": transcript_paths[0] if transcript_paths else None,
        "valid": valid, "invalid_reason": invalid_reason, "timed_out": timed_out, "error": error,
        "marker_seen": marker_seen,
        "found": scored["found"], "found_method": scored["found_method"],
        "found_index": scored["found_index"],
        "first_procedure_step_index": scored["first_procedure_step_index"],
        "recall_fired": scored["recall_fired"],
        "followed_steps": scored["followed_steps"], "followed_k": scored["followed_k"],
        "followed_all": scored["followed_all"], "n_steps": scored["n_steps"],
        "end_state": scored["end_state"], "end_state_output": scored["end_state_output"],
        "trailer": scored["trailer"],
        "carrier_basename": carrier_basename,
        "vault_copy_s": vault_copy_s,
        "total_cost_usd": total_cost_usd,
        "duration_ms": result.get("duration_ms") if isinstance(result, dict) else None,
        "num_turns": num_turns,
        "session_id": result.get("session_id") if isinstance(result, dict) else None,
        "wall_s": round(time.time() - t0, 1),
    }
    # Vault copies are deleted after scoring even under --keep (keep repo + transcripts).
    shutil.rmtree(env["ENGRAM_VAULT_PATH"], ignore_errors=True)
    return record


# ----- cfg settings: suppress Claude Code's own commit/PR attribution injection -----

# Smoke-run-1 finding: every trial transcript carries a `remote_session_change` `attachment`
# (system-reminder text "Attribution for git commits and pull requests you create from here on
# ... End git commit messages with: Co-Authored-By: <model> ...") injected by Claude Code ITSELF
# — not via CLAUDE.md, not attributable to any arm's carrier — and every arm followed it over the
# carrier/skill's own AI-Used: [claude] convention, contaminating done_when_checks.sh's trailer
# check identically across arms. `attribution.commit`/`attribution.pr` is the CURRENT (non-
# deprecated) settings.json key per code.claude.com/docs/en/settings-reference.md — verified via
# WebFetch on the official docs (the older `includeCoAuthoredBy: false` is explicitly documented
# there as deprecated in favor of `attribution`). Final-review correction: this setting was
# ATTEMPTED as a suppression, not "decisively confirmed" to work — the plumbing trial and the
# subsequent bridge-env-var stripping below (see `_BRIDGE_ENV_VARS`) were both tried and the
# `remote_session_change` attachment PERSISTED regardless (task-3-report.md records this as
# BLOCKED, not resolved). The trailer is therefore REPORTED, not SCORED, in every downstream
# check (see `classify_trailer`) — this settings.json is kept for completeness/documentation of
# what was tried, not because it is known to suppress the injection.
CFG_SETTINGS = {"attribution": {"commit": False, "pr": False}}


def _write_cfg_settings(cfg_dir):
    with open(os.path.join(cfg_dir, "settings.json"), "w") as f:
        json.dump(CFG_SETTINGS, f)


def build_cfg_template_phase2(dst):
    """p1.build_cfg_template(dst) (warm recall+learn skills), plus a settings.json suppressing
    Claude Code's own commit/PR attribution injection."""
    p1.build_cfg_template(dst)
    _write_cfg_settings(dst)


def build_cfg_pool_phase2(run_root, n):
    """p1.build_cfg_pool(run_root, n) (per-worker cfg pool, never reimplemented here), with the
    attribution-suppressing settings.json written into every resulting cfg dir afterward."""
    dirs = p1.build_cfg_pool(run_root, n)
    for cfg_dir in dirs:
        _write_cfg_settings(cfg_dir)
    return dirs


# Root-cause finding (settings.json's `attribution.commit: false` was verified via plumbing trial
# to have NO effect — see task-3-report.md): the injected `remote_session_change` attachment is
# NOT a local-settings-driven behavior at all. `env | grep -i claude` in THIS orchestrator session
# shows CLAUDE_CODE_BRIDGE_SESSION_ID matching this session's own bridge session id, plus
# CLAUDE_CODE_CHILD_SESSION=1 and the messaging socket/token — p1.trial_env(..., base=os.environ)
# inherits these into the spawned trial subprocess, which then relays the ORCHESTRATOR session's
# own attribution config into the trial's transcript as a "remote session change" — a session-
# bridge relay, not a settings read. Stripped from every trial's env below.
_BRIDGE_ENV_VARS = (
    "CLAUDE_CODE_BRIDGE_SESSION_ID", "CLAUDE_CODE_CHILD_SESSION",
    "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN",
)


def trial_env_phase2(cfg, trial_dir, repo_path):
    """p1.trial_env(cfg, trial_dir, repo_path), with the orchestrator-session bridge env vars
    stripped so a spawned trial does not inherit — and relay into its own transcript — the
    ORCHESTRATOR session's commit/PR attribution config."""
    env = p1.trial_env(cfg, trial_dir, repo_path)
    for var in _BRIDGE_ENV_VARS:
        env.pop(var, None)
    return env


# ----- batch run -----

def run_batch(args):
    arms = [a.strip() for a in args.arms.split(",") if a.strip()]
    unknown = [a for a in arms if a not in ARMS]
    if unknown:
        raise SystemExit(f"unknown arm(s) {unknown}; choose from {ARMS}")
    task_key = validate_task_key(args.task)

    run_id = f"{task_key}-{args.model}-{int(time.time())}-{uuid.uuid4().hex[:6]}"
    run_root = os.path.join(p1.DEFAULT_RUN_ROOT, RUN_ROOT_SUBDIR, run_id)
    os.makedirs(run_root, exist_ok=True)

    cfg_dirs = build_cfg_pool_phase2(run_root, args.workers)
    for cfg_dir in cfg_dirs:
        p1.matrix.refresh_creds(cfg_dir)
    cfg_pool = queue.Queue()
    for cfg_dir in cfg_dirs:
        cfg_pool.put(cfg_dir)

    marker = f"RUNBOOK-VS-SKILL-PROBE2-{uuid.uuid4().hex[:8]}"
    before_fp = p1._real_vault_fingerprint()

    jobs = [(arm, i) for arm in arms for i in range(args.n)]
    print(f"run_id={run_id} task={task_key} arms={arms} n={args.n} model={args.model} "
          f"trials={len(jobs)} timeout={args.timeout}s workers={args.workers} root={run_root}")

    def _run_pooled(arm, i):
        cfg_dir = cfg_pool.get()
        try:
            return run_one_trial_phase2(run_root, cfg_dir, task_key, arm, args.model, i, marker,
                                         args.timeout, args.exclude_luhmann_min)
        finally:
            cfg_pool.put(cfg_dir)

    try:
        with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
            futs = {ex.submit(_run_pooled, arm, i): (arm, i) for arm, i in jobs}
            for fut in cf.as_completed(futs):
                arm, i = futs[fut]
                record = fut.result()
                record["run_id"] = run_id
                p1.append_jsonl(args.out, record)
                status = "valid" if record["valid"] else f"INVALID({record.get('invalid_reason') or 'no-marker'})"
                print(f"  [{task_key}-{arm}#{i}] {status} found={record['found']} "
                      f"followed={record['followed_k']}/{record['n_steps']} end_state={record['end_state']} "
                      f"cost=${record['total_cost_usd']:.2f} timed_out={record['timed_out']}")
    finally:
        after_fp = p1._real_vault_fingerprint()
        if after_fp != before_fp:
            print(f"ABORT-REPORT: operator's real vault fingerprint changed! before={before_fp} "
                  f"after={after_fp}. A trial may have reached real memory. Investigate before "
                  "trusting any result in this run.", file=sys.stderr)
        if not args.keep:
            shutil.rmtree(run_root, ignore_errors=True)


# ----- baseline mode: arm N only, bare agent, no carrier -----

def baseline_step_miss_counts(records, n_steps):
    """{step_n: miss_count} across VALID records only, for step numbers 1..n_steps. A step counts
    as a miss when its `followed_steps` entry is False or absent (e.g. a scoring_error left
    followed_steps empty for that trial)."""
    valid = [r for r in records if r.get("valid")]
    counts = {}
    for step_n in range(1, n_steps + 1):
        key = str(step_n)
        counts[step_n] = sum(1 for r in valid if not (r.get("followed_steps") or {}).get(key, False))
    return counts


def format_baseline_summary(task_key, model, records):
    """Pure formatting over already-scored trial records (real or synthetic — never calls
    claude). `records` is the same shape run_one_trial_phase2 emits; arm N carries no
    found/found_method (score_found_phase2 returns n/a for arm N — no carrier to find, by
    design), so this reports only END-STATE, FOLLOWED-all, per-step miss counts, and mean cost.
    Denominators are VALID trials only — a rate-limited outage (invalid_reason in
    _RATE_LIMIT_INVALID_REASONS: a zero-work "rate_limit" stub OR a "rate_limit_truncated"
    mid-task cutoff after real work) is never valid, so it drops out of every rate here
    automatically; its count is still surfaced via the ` (rate-limited: K)` suffix on the first
    line whenever K > 0, so an outage is visible in one line rather than silently deflating the
    denominator (vault note 988a)."""
    n = len(records)
    valid = [r for r in records if r.get("valid")]
    valid_n = len(valid)
    rate_limited_n = sum(1 for r in records if r.get("invalid_reason") in _RATE_LIMIT_INVALID_REASONS)
    end_state_n = sum(1 for r in valid if r.get("end_state"))
    followed_all_n = sum(1 for r in valid if r.get("followed_all"))
    n_steps = next((r.get("n_steps") for r in valid if r.get("n_steps")), 0)
    miss_counts = baseline_step_miss_counts(records, n_steps)
    cost_mean = (sum(r.get("total_cost_usd") or 0 for r in valid) / valid_n) if valid_n else 0.0

    miss_str = ", ".join(f"step {k}: {v}/{valid_n}" for k, v in sorted(miss_counts.items())) or "n/a"
    rate_limited_suffix = f" (rate-limited: {rate_limited_n})" if rate_limited_n else ""
    return (
        f"bare agent (task={task_key}, model={model}, n={n}): "
        f"end result {end_state_n}/{valid_n}, did every step {followed_all_n}/{valid_n}"
        f"{rate_limited_suffix}\n"
        f"per-step miss counts: {miss_str}\n"
        f"mean cost: ${cost_mean:.2f}"
    )


def run_baseline(args):
    """Arm N only (bare agent — no skill/runbook/fact-note carrier) against one task: the cheap
    entry point for baselining a NEW candidate task before any carrier is built for it. Reuses
    run_one_trial_phase2's exact trial machinery (setup_trial_repo/setup_trial_vault — same
    CLAUDE.md with guidance+cue+marker, same real-vault copy minus covering notes and
    eval-session notes; see setup_trial_vault/remove_covering_notes/remove_eval_session_notes);
    only the arm (fixed to "N") and the final summary differ from run_batch."""
    task_key = validate_task_key(args.baseline)

    ts = int(time.time())
    # `run_id` stays fully descriptive (stored on every record, printed below) — it is the
    # DIRECTORY name that must stay short: a long task name (e.g. "gitignore-nested") folded into
    # every trial's cwd is exactly what made Claude Code truncate its own projects/<slug>
    # directory name in the first --baseline batch (see RUN_ROOT_SUBDIR / discover_transcript_paths).
    run_id = f"baseline-{task_key}-{args.model}-{ts}-{uuid.uuid4().hex[:6]}"
    run_dir_name = f"b-{str(ts)[-6:]}"
    run_root = os.path.join(p1.DEFAULT_RUN_ROOT, RUN_ROOT_SUBDIR, run_dir_name)
    os.makedirs(run_root, exist_ok=True)

    cfg_dirs = build_cfg_pool_phase2(run_root, args.workers)
    for cfg_dir in cfg_dirs:
        p1.matrix.refresh_creds(cfg_dir)
    cfg_pool = queue.Queue()
    for cfg_dir in cfg_dirs:
        cfg_pool.put(cfg_dir)

    marker = f"RUNBOOK-VS-SKILL-BASELINE-{uuid.uuid4().hex[:8]}"
    before_fp = p1._real_vault_fingerprint()

    print(f"run_id={run_id} task={task_key} arm=N n={args.n} model={args.model} "
          f"timeout={args.timeout}s workers={args.workers} root={run_root}")

    def _run_pooled(i):
        cfg_dir = cfg_pool.get()
        try:
            return run_one_trial_phase2(run_root, cfg_dir, task_key, "N", args.model, i, marker,
                                         args.timeout, args.exclude_luhmann_min)
        finally:
            cfg_pool.put(cfg_dir)

    records = []
    try:
        with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
            futs = {ex.submit(_run_pooled, i): i for i in range(args.n)}
            for fut in cf.as_completed(futs):
                i = futs[fut]
                record = fut.result()
                record["run_id"] = run_id
                records.append(record)
                if args.out:
                    p1.append_jsonl(args.out, record)
                status = "valid" if record["valid"] else f"INVALID({record.get('invalid_reason') or 'no-marker'})"
                print(f"  [{task_key}-N#{i}] {status} end_state={record['end_state']} "
                      f"followed={record['followed_k']}/{record['n_steps']} "
                      f"cost=${record['total_cost_usd']:.2f} timed_out={record['timed_out']}")
    finally:
        after_fp = p1._real_vault_fingerprint()
        if after_fp != before_fp:
            print(f"ABORT-REPORT: operator's real vault fingerprint changed! before={before_fp} "
                  f"after={after_fp}. A trial may have reached real memory. Investigate before "
                  "trusting any result in this run.", file=sys.stderr)
        if not args.keep:
            shutil.rmtree(run_root, ignore_errors=True)

    print(format_baseline_summary(task_key, args.model, records))


# ----- plumbing mode -----

def plumbing_prompt(task_key):
    return f"Run /recall glance for: {PLUMBING_SITUATION[task_key]}. Report what the vault returned."


def run_plumbing(task_key, model):
    run_id = f"plumbing-{task_key}-{model}-{int(time.time())}"
    run_root = os.path.join(p1.DEFAULT_RUN_ROOT, RUN_ROOT_SUBDIR, run_id)
    os.makedirs(run_root, exist_ok=True)
    cfg = os.path.join(run_root, "cfg")
    build_cfg_template_phase2(cfg)
    p1.matrix.refresh_creds(cfg)

    before_fp = p1._real_vault_fingerprint()
    marker = f"RUNBOOK-VS-SKILL-PROBE2-{uuid.uuid4().hex[:8]}"
    trial_dir = os.path.join(run_root, "trials", "plumbing-0")
    os.makedirs(trial_dir, exist_ok=True)
    repo_path = setup_trial_repo(trial_dir, task_key, "R", marker)
    env = trial_env_phase2(cfg, trial_dir, repo_path)
    carrier_basename, vault_copy_s = setup_trial_vault(env, task_key, "R")

    result, timed_out = p1.spawn_claude(env, model, repo_path, plumbing_prompt(task_key), p1.DEFAULT_TIMEOUT_S)
    after_fp = p1._real_vault_fingerprint()

    transcript_paths = p1.discover_transcript_paths(cfg, repo_path)
    raw_text = p1.transcript_raw_text(transcript_paths)
    events = p1.parse_transcript_events(transcript_paths)

    marker_seen = p1.is_marker_seen(raw_text, marker)
    recall_fired = p1.score_recall_fired(events)
    query_events = [ev for ev in events if ev["kind"] == "tool_use" and ev.get("name") == "Bash"
                    and "engram query" in ((ev.get("input") or {}).get("command", "") or "")]
    query_ran = bool(query_events)
    carrier_surfaced = False
    result_snippets = []
    for ev in query_events:
        text = p1._tool_result_text(events, ev.get("id"))
        result_snippets.append(text[:1500])
        if carrier_basename and carrier_basename in text:
            carrier_surfaced = True

    print(f"marker_seen={marker_seen}")
    print(f"recall_skill_fired={recall_fired}")
    print(f"engram_query_ran={query_ran}")
    print(f"carrier_surfaced={carrier_surfaced}")
    print(f"carrier_basename={carrier_basename}")
    print(f"cost_usd={p1._call_cost(result)}")
    print(f"timed_out={timed_out}")
    print(f"vault_copy_s={vault_copy_s}")
    print(f"transcript_path={transcript_paths[0] if transcript_paths else None}")
    print(f"run_root={run_root}")
    if query_ran and not carrier_surfaced:
        print("QUERY RESULT SNIPPETS (carrier did not surface — for the concern report):")
        for snippet in result_snippets:
            print("---")
            print(snippet)
    shutil.rmtree(env["ENGRAM_VAULT_PATH"], ignore_errors=True)
    if after_fp != before_fp:
        print(f"ABORT-REPORT: operator's real vault fingerprint changed! before={before_fp} "
              f"after={after_fp}.", file=sys.stderr)


# ----- setup-only mode: dry run, no claude call -----

def run_setup_only(task_key, arm, exclude_luhmann_min=EXCLUDE_LUHMANN_MIN):
    """Build the trial repo + background vault + CLAUDE.md for one (task, arm) pair, print the
    starting-state checks, and clean up — no claude call. For confirming a fixture change against
    the harness's own starting-state assertion without spending on a trial."""
    run_id = f"setup-only-{task_key}-{arm}-{int(time.time())}"
    run_root = os.path.join(p1.DEFAULT_RUN_ROOT, RUN_ROOT_SUBDIR, run_id)
    trial_dir = os.path.join(run_root, "trials", "setup-only-0")
    os.makedirs(trial_dir, exist_ok=True)
    cfg = os.path.join(run_root, "cfg")
    os.makedirs(cfg, exist_ok=True)
    marker = f"RUNBOOK-VS-SKILL-PROBE2-{uuid.uuid4().hex[:8]}"

    before_fp = p1._real_vault_fingerprint()
    env = None
    try:
        repo_path = setup_trial_repo(trial_dir, task_key, arm, marker)
        print(f"setup_trial_repo: OK ({task_key}/{arm}) repo_path={repo_path}")

        status = subprocess.run(["git", "-C", repo_path, "status", "--porcelain"],
                                 capture_output=True, text=True, check=True).stdout
        print("git status --porcelain:")
        print(status.rstrip("\n") if status.strip() else "(clean)")

        ls_files = subprocess.run(["git", "-C", repo_path, "ls-files"],
                                   capture_output=True, text=True, check=True).stdout
        print("git ls-files:", ", ".join(ls_files.split()) or "(none)")

        for rel_path in _START_STATE_IGNORED_PATHS.get(task_key, ()):
            r = subprocess.run(["git", "-C", repo_path, "check-ignore", "-q", rel_path])
            print(f"check-ignore {rel_path}: {'ignored' if r.returncode == 0 else 'NOT ignored'}")

        # assert_starting_state already ran inside setup_trial_repo (raises on failure); reaching
        # here means it passed.
        print("starting-state assertion: PASSED")

        env = trial_env_phase2(cfg, trial_dir, repo_path)
        carrier_basename, vault_copy_s = setup_trial_vault(env, task_key, arm, exclude_luhmann_min)
        md_count = len([n for n in os.listdir(env["ENGRAM_VAULT_PATH"]) if n.endswith(".md")])
        print(f"carrier_basename={carrier_basename} vault_copy_s={vault_copy_s}")
        print(f"vault .md count after setup: {md_count}")
    except Exception as exc:  # noqa: BLE001 — report and re-raise, still clean up in finally
        print(f"setup-only FAILED: {exc}", file=sys.stderr)
        raise
    finally:
        # Keep vault and repo for manual inspection of the trial repo/vault (removed after verification checks)
        # if env is not None:
        #     shutil.rmtree(env["ENGRAM_VAULT_PATH"], ignore_errors=True)
        after_fp = p1._real_vault_fingerprint()
        if after_fp != before_fp:
            print(f"ABORT-REPORT: operator's real vault fingerprint changed! before={before_fp} "
                  f"after={after_fp}.", file=sys.stderr)
        # shutil.rmtree(run_root, ignore_errors=True)


# ----- summarize / decomposition (Ruling 7) -----

def load_jsonl(path):
    return p1.load_jsonl(path)


def aggregate(records, task, arm):
    rows = [r for r in records if r.get("task") == task and r.get("arm") == arm]
    valid = [r for r in rows if r.get("valid")]
    n = len(rows)
    valid_n = len(valid)
    rate_limited_n = sum(1 for r in rows if r.get("invalid_reason") in _RATE_LIMIT_INVALID_REASONS)
    found_n = sum(1 for r in valid if r.get("found") is True)
    found_given_n = found_n
    end_state_n = sum(1 for r in valid if r.get("end_state"))
    end_state_given_found_n = sum(1 for r in valid if r.get("found") and r.get("end_state"))
    followed_all_n = sum(1 for r in valid if r.get("followed_all"))
    followed_all_given_found_n = sum(1 for r in valid if r.get("found") and r.get("followed_all"))
    recall_fired_n = sum(1 for r in valid if r.get("recall_fired"))
    followed_mean_k = (sum((r.get("followed_k") or 0) for r in valid) / valid_n) if valid_n else 0.0
    n_steps = next((r.get("n_steps") for r in valid if r.get("n_steps")), None)
    cost_mean = (sum(r.get("total_cost_usd") or 0 for r in valid) / valid_n) if valid_n else 0.0
    durations = [r.get("duration_ms") for r in valid if r.get("duration_ms")]
    duration_mean_s = (sum(durations) / len(durations) / 1000.0) if durations else 0.0
    # Task A only; reported not scored (round-4 ruling) — see classify_trailer.
    trailer_ai_used_n = sum(1 for r in valid if r.get("trailer") in ("ai_used", "both"))
    trailer_co_authored_n = sum(1 for r in valid if r.get("trailer") in ("co_authored", "both"))
    return {
        "n": n, "valid_n": valid_n, "rate_limited_n": rate_limited_n,
        "found_n": found_n, "found_given_n": found_given_n,
        "end_state_n": end_state_n, "end_state_given_found_n": end_state_given_found_n,
        "followed_all_n": followed_all_n, "followed_all_given_found_n": followed_all_given_found_n,
        "recall_fired_n": recall_fired_n, "followed_mean_k": followed_mean_k, "n_steps": n_steps,
        "cost_mean": cost_mean, "duration_mean": duration_mean_s,
        "trailer_ai_used_n": trailer_ai_used_n, "trailer_co_authored_n": trailer_co_authored_n,
    }


def _gap_verdict(baseline_val, other_val):
    gap = other_val - baseline_val
    if abs(gap) <= 1:
        return "cant_distinguish"
    return "better" if gap >= 2 else "worse"


def _rate_pp_diff(k_a, n_a, k_b, n_b, label_a, label_b):
    """Percentage-point difference between two k/n RATES (rate_a − rate_b), reported alongside
    both fractions. Never subtract raw counts across populations with different n — a 4/4 (100%)
    vs 5/5 (100%) comparison is 0 percentage points, not '-1' (final-review finding: Task B's
    Rdirect n=4 due to one invalidated trial vs R's n=5 made an identical 100%-vs-100% rate read
    as a fake '-1 loss' under raw subtraction)."""
    rate_a = (k_a / n_a) if n_a else 0.0
    rate_b = (k_b / n_b) if n_b else 0.0
    return {
        "pp_diff": round((rate_a - rate_b) * 100, 1),
        label_a: f"{k_a}/{n_a}",
        label_b: f"{k_b}/{n_b}",
    }


def decomposition(agg):
    """PLAN-2 line 27 / task-3-brief Step 8 decomposition. shim_loss and note_quality_F are
    reported over BOTH populations per the controller's final ruling: `_total` (all valid trials
    — the retrieval path's full cost, including R's not-found trials) and `_given_found` (only
    R's found=true subset). type_effect reports both populations for END-STATE and FOLLOWED-all.
    These two populations are only equal when found_n == valid_n (retrieval never missed) — tests
    must use an aggregate where they diverge to prove the split is real, not aliased."""
    s, r, f, rd = agg.get("S"), agg.get("R"), agg.get("F"), agg.get("Rdirect")
    out = {}

    if r:
        out["shim_rate_R"] = f"{r['found_n']}/{r['valid_n']}"
    if f:
        out["shim_rate_F"] = f"{f['found_n']}/{f['valid_n']}"

    if r:
        out["note_quality_given_delivery_R"] = {
            "end_state": f"{r['end_state_given_found_n']} of {r['found_given_n']}",
            "followed_all": f"{r['followed_all_given_found_n']} of {r['found_given_n']}",
        }
    if f:
        out["note_quality_given_delivery_F"] = {
            "end_state": f"{f['end_state_given_found_n']} of {f['found_given_n']}",
            "followed_all": f"{f['followed_all_given_found_n']} of {f['found_given_n']}",
        }

    if rd:
        out["note_ceiling_Rdirect"] = {
            "end_state": f"{rd['end_state_n']}/{rd['valid_n']}",
            "followed_all": f"{rd['followed_all_n']}/{rd['valid_n']}",
        }

    # shim_loss: the retrieval path's RATE vs. the no-retrieval ceiling's RATE, reported over BOTH
    # populations — total (all valid trials, incl. R's not-found trials) and given_found (R's
    # found=true subset only) — since these are observably different whenever found_n < valid_n.
    # Rate-based (percentage-point diff), never a raw-count subtraction: Rdirect and R can have
    # different valid_n (e.g. an invalidated trial), so a count diff conflates population-size
    # mismatch with an actual outcome gap — see _rate_pp_diff.
    if rd and r:
        out["shim_loss_total"] = _rate_pp_diff(
            rd["end_state_n"], rd["valid_n"], r["end_state_n"], r["valid_n"], "rdirect", "r")
        out["shim_loss_given_found"] = _rate_pp_diff(
            rd["end_state_n"], rd["valid_n"], r["end_state_given_found_n"], r["found_given_n"],
            "rdirect", "r_given_found")

    # note_quality_F: how much the fact note's RATE adds over the shim/no-retrieval ceiling's
    # RATE, same total/given_found split, same rate-based reasoning.
    if f and rd:
        out["note_quality_F_total"] = _rate_pp_diff(
            f["end_state_n"], f["valid_n"], rd["end_state_n"], rd["valid_n"], "f", "rdirect")
        out["note_quality_F_given_found"] = _rate_pp_diff(
            f["end_state_given_found_n"], f["found_given_n"], rd["end_state_n"], rd["valid_n"],
            "f_given_found", "rdirect")

    if r and f:
        out["type_effect"] = {
            "end_state_total": r["end_state_n"] - f["end_state_n"],
            "end_state_given_found": r["end_state_given_found_n"] - f["end_state_given_found_n"],
            "followed_all_total": r["followed_all_n"] - f["followed_all_n"],
            "followed_all_given_found": (r["followed_all_given_found_n"]
                                          - f["followed_all_given_found_n"]),
        }

    parity = {}
    if s and r:
        parity["S_vs_R"] = {
            "end_state": _gap_verdict(s["end_state_n"], r["end_state_n"]),
            "followed_all": _gap_verdict(s["followed_all_n"], r["followed_all_n"]),
        }
    if s and f:
        parity["S_vs_F"] = {
            "end_state": _gap_verdict(s["end_state_n"], f["end_state_n"]),
            "followed_all": _gap_verdict(s["followed_all_n"], f["followed_all_n"]),
        }
    out["parity"] = parity

    out["baseline_uninterpretable"] = bool(s and s["end_state_n"] < 3)
    return out


def format_table(task, agg):
    arms = [a for a in ARMS if a in agg]
    label_width = 28
    lines = [f"=== Task {task} ==="]
    header = "metric".ljust(label_width) + "".join(a.ljust(16) for a in arms)
    lines.append(header)
    lines.append("-" * len(header))

    def row(label, fmt):
        cells = [fmt(arm, agg[arm]) for arm in arms]
        lines.append(label.ljust(label_width) + "".join(c.ljust(16) for c in cells))

    row("FOUND (k/n)", lambda arm, a: "n/a" if arm in ("Rdirect", "N") else f"{a['found_n']}/{a['valid_n']}")
    row("FOLLOWED all-steps (k/n)", lambda arm, a: f"{a['followed_all_n']}/{a['valid_n']}")
    row("FOLLOWED (mean k/N)", lambda arm, a: f"{a['followed_mean_k']:.2f}/{a['n_steps']}")
    row("END-STATE (k/n)", lambda arm, a: f"{a['end_state_n']}/{a['valid_n']}")
    row("recall_fired (k/n)", lambda arm, a: f"{a['recall_fired_n']}/{a['valid_n']}")
    row("cost (mean USD)", lambda arm, a: f"${a['cost_mean']:.2f}")
    row("duration (mean s)", lambda arm, a: f"{a['duration_mean']:.0f}")
    def _valid_cell(arm, a):
        base = f"{a['valid_n']}/{a['n']}"
        rate_limited_n = a.get("rate_limited_n") or 0
        return f"{base} (rate-limited: {rate_limited_n})" if rate_limited_n else base

    row("valid (n)", _valid_cell)
    if task == "A":
        row("trailer AI-Used (k/n)", lambda arm, a: f"{a['trailer_ai_used_n']}/{a['valid_n']}")
        row("trailer Co-Authored-By (k/n)", lambda arm, a: f"{a['trailer_co_authored_n']}/{a['valid_n']}")
        lines.append("  ^ reported, not scored — harness attribution injection overrides carriers")
    if "N" in agg:
        n_agg = agg["N"]
        lines.append(f"no-instructions baseline: FOLLOWED-all "
                     f"{n_agg['followed_all_n']}/{n_agg['valid_n']}, "
                     f"END-STATE {n_agg['end_state_n']}/{n_agg['valid_n']}")
    return "\n".join(lines)


def summarize_file(path):
    records = load_jsonl(path)
    tasks_present = sorted({r.get("task") for r in records if r.get("task")})
    frames = {}
    for task in tasks_present:
        agg = {arm: aggregate(records, task, arm) for arm in ARMS if any(
            r.get("task") == task and r.get("arm") == arm for r in records)}
        print(format_table(task, agg))
        frames[task] = decomposition(agg)
        print()
    print("Decision frame:")
    print(json.dumps(frames, indent=2))
    return frames


_PROBE_TOKEN_RE = re.compile(r"PROBE-TOKEN:\s*(\S+)")


def _read_repo_marker(repo_path):
    """The PROBE-TOKEN marker embedded verbatim in the trial repo's own committed CLAUDE.md (see
    p1.build_claude_md) — read directly from the repo rather than requiring the marker string to
    be stored on the kept record (it never was). Returns None if CLAUDE.md is missing or carries
    no PROBE-TOKEN line."""
    try:
        text = open(os.path.join(repo_path, "CLAUDE.md")).read()
    except OSError:
        return None
    m = _PROBE_TOKEN_RE.search(text)
    return m.group(1) if m else None


def rediscover_transcript_paths(trial_dir, repo_path):
    """Search every cfg-*/ directory under the trial's run_root (trial_dir's grandparent —
    run_root/trials/<name>) for the trial's transcript. Which cfg-N a live run's worker pool
    assigned to this particular trial is never recorded on the kept record, so every cfg dir is
    tried; p1.discover_transcript_paths's truncated-slug-prefix + exact-cwd-verification (see
    probe.py) means trying several cfg dirs never risks picking up the WRONG trial's transcript —
    a candidate is only accepted when ITS OWN session jsonl records this exact repo_path as its
    cwd. Returns [] if trial_dir is falsy, its run_root doesn't exist, or no cfg dir matches."""
    if not trial_dir:
        return []
    run_root = os.path.dirname(os.path.dirname(trial_dir))
    if not os.path.isdir(run_root):
        return []
    for name in sorted(os.listdir(run_root)):
        if not name.startswith("cfg-"):
            continue
        found = p1.discover_transcript_paths(os.path.join(run_root, name), repo_path)
        if found:
            return found
    return []


def rescore_file(in_path, out_path):
    """Re-run marker validity, FOUND + `first_procedure_step_index` + `recall_fired` (round-3:
    extended to cover FOUND, per the mutating-regex fix), steps.json evaluation (incl.
    `n_steps`), END-STATE, and the Task A `trailer` field (round 4) on kept trial
    directories/transcripts — round 6: this field set is kept IDENTICAL to what the live scoring
    path (`_score_trial`) emits, so a rescored record is directly comparable, field for field, to
    a live one (see `test_live_scoring_and_rescore_emit_the_same_scored_field_set`). `repo_path`
    is required (for steps.json's repo_state signals, END-STATE, `trailer`, and re-reading the
    trial's own PROBE-TOKEN off its committed CLAUDE.md); `transcript_path` is required for
    FOUND, bash_regex steps, and marker validity — if it is missing or no longer exists (the
    truncated-projects-dir bug: Claude Code's own projects/<slug> dir name got truncated with a
    random suffix, so the live run's exact-slug lookup found nothing even though the trial ran —
    see rediscover_transcript_paths / probe.discover_transcript_paths), it is RE-DISCOVERED here
    by scanning every cfg-*/ dir under the trial's run_root; if that also comes up empty, FOUND
    and marker validity recompute against an empty event list (found stays at its arm-appropriate
    default, e.g. None for Rdirect / False otherwise; `valid`/`marker_seen` become False).
    Provenance (`arm`, `carrier_basename`, `task`) is read from the kept record, never
    re-derived.

    Also reclassifies rate-limit-related outages (vault note 988a, plus its 2026-09-12 follow-up):
    `invalid_reason` is recomputed from the kept record's own `num_turns`/`total_cost_usd` plus the
    (re-discovered, if needed) transcript text — never a live claude call, so this is the only
    place a set-aside `*.rate-limited.jsonl` file gets corrected. `valid` is forced False whenever
    EITHER outage shape is detected, even when the trial's CLAUDE.md marker is (as expected) still
    present: a zero-work stub (`invalid_reason="rate_limit"`) or a mid-task truncation after real
    work (`invalid_reason="rate_limit_truncated"`, see classify_validity) — both are surfaced
    together in the ` (rate-limited: K)` summary suffix (_RATE_LIMIT_INVALID_REASONS)."""
    records = load_jsonl(in_path)
    rescored = []

    for record in records:
        repo_path = record.get("repo_path")
        transcript_path = record.get("transcript_path")
        task_key = record.get("task")
        arm = record.get("arm")
        carrier_basename = record.get("carrier_basename")

        if not repo_path or not os.path.exists(repo_path):
            record["error"] = "rescore: repo missing"
            rescored.append(record)
            continue

        try:
            transcript_paths_for_parsing = []
            if transcript_path and os.path.exists(transcript_path):
                transcript_paths_for_parsing = [transcript_path]
            else:
                rediscovered = rediscover_transcript_paths(record.get("trial_dir"), repo_path)
                if rediscovered:
                    transcript_paths_for_parsing = rediscovered
                    record["transcript_path"] = rediscovered[0]

            events = (p1.parse_transcript_events(transcript_paths_for_parsing)
                      if transcript_paths_for_parsing else [])
            raw_text = (p1.transcript_raw_text(transcript_paths_for_parsing)
                        if transcript_paths_for_parsing else "")

            marker = _read_repo_marker(repo_path)
            if marker is not None:
                marker_seen = p1.is_marker_seen(raw_text, marker)
                record["marker_seen"] = marker_seen
                valid, invalid_reason = classify_validity(
                    marker_seen, record.get("num_turns"), record.get("total_cost_usd"),
                    None, raw_text)
                record["valid"] = valid
                record["invalid_reason"] = invalid_reason
            else:
                if is_rate_limited_stub(record.get("num_turns"), record.get("total_cost_usd"),
                                         None, raw_text):
                    record["valid"] = False
                    record["invalid_reason"] = "rate_limit"
                elif _rate_limit_truncation_signal_present(None, raw_text):
                    record["valid"] = False
                    record["invalid_reason"] = "rate_limit_truncated"
                else:
                    record.setdefault("invalid_reason", None)

            found, found_idx = score_found_phase2(task_key, arm, events, carrier_basename)
            record["found"] = found
            record["found_index"] = found_idx
            record["found_method"] = found_method(arm, found)
            record["first_procedure_step_index"] = first_mutating_step_index(events, task_key)
            record["recall_fired"] = p1.score_recall_fired(events)

            steps = load_steps(task_key)
            record["n_steps"] = len(steps)
            followed_steps, followed_k, followed_all = evaluate_steps(steps, events, repo_path)
            record["followed_steps"] = followed_steps
            record["followed_k"] = followed_k
            record["followed_all"] = followed_all

            end_state, end_state_output = check_end_state_phase2(task_key, repo_path)
            record["end_state"] = end_state
            record["end_state_output"] = end_state_output

            record["trailer"] = classify_trailer(repo_path) if task_key == "A" else "n/a"

            record["rescored_from"] = in_path
        except Exception as e:  # noqa: BLE001
            record["error"] = f"rescore exception: {str(e)}"

        rescored.append(record)

    with open(out_path, "w") as f:
        for record in rescored:
            f.write(json.dumps(record) + "\n")

    print(f"Rescored {len(rescored)} records from {in_path} to {out_path}")


# ----- CLI -----

def build_argparser():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--task", help="a fixtures/<name>/ directory name (discovered dynamically), "
                                    "or a legacy alias (A -> commit, B -> gitignore)")
    ap.add_argument("--arms", default=",".join(DEFAULT_ARMS))
    ap.add_argument("--model", choices=list(p1.MODELS))
    ap.add_argument("--n", type=int)
    ap.add_argument("--out")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--timeout", type=int, default=p1.DEFAULT_TIMEOUT_S)
    ap.add_argument("--keep", action="store_true", help="keep the run root instead of deleting it on exit")
    ap.add_argument("--plumbing", action="store_true")
    ap.add_argument("--setup-only", action="store_true",
                     help="build repo+vault+CLAUDE.md for one arm (first of --arms) and print "
                          "the starting-state checks — no claude call")
    ap.add_argument("--baseline", metavar="TASK",
                     help="run arm N only (bare agent, no carrier) against TASK and print a "
                          "summary — the cheap way to baseline a new candidate task before any "
                          "carrier is built for it")
    ap.add_argument("--summarize")
    ap.add_argument("--rescore", help="re-score an existing results.jsonl file using kept trial directories")
    ap.add_argument("--exclude-luhmann-min", type=int, default=EXCLUDE_LUHMANN_MIN,
                     help="delete any background-vault note whose leading luhmann number is >= "
                          "this floor (this eval session's own notes) from every trial's vault")
    return ap


def main(argv=None):
    args = build_argparser().parse_args(argv)
    if args.summarize:
        summarize_file(args.summarize)
        return
    if args.rescore:
        if not args.out:
            build_argparser().error("--rescore requires --out")
        rescore_file(args.rescore, args.out)
        return
    if args.baseline:
        if not (args.model and args.n is not None):
            build_argparser().error("--baseline requires --model and --n")
        run_baseline(args)
        return
    if args.plumbing:
        if not args.task:
            build_argparser().error("--plumbing requires --task")
        task_key = validate_task_key(args.task)
        run_plumbing(task_key, args.model or "sonnet")
        return
    if args.setup_only:
        if not args.task:
            build_argparser().error("--setup-only requires --task")
        task_key = validate_task_key(args.task)
        arms = [a.strip() for a in args.arms.split(",") if a.strip()]
        for arm in arms:
            run_setup_only(task_key, arm, args.exclude_luhmann_min)
        return
    if not args.task:
        build_argparser().error("--task is required (unless --summarize/--rescore/--baseline)")
    if not (args.model and args.n is not None and args.out):
        build_argparser().error("--model, --n, and --out are required for a normal run")
    run_batch(args)


if __name__ == "__main__":
    main()
