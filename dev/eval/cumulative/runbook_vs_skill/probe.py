#!/usr/bin/env python3
"""runbook_vs_skill three-arm headless eval harness (Task 3).

Measures whether a runbook note (Arm R) or fact note (Arm F) in the engram vault can replace a
real skill (Arm S) in guiding a headless `claude -p` agent through a multi-step, idiosyncratic
task (registering a fictional sensor type in a fictional telemetry repo). See PLAN.md for the
full design and dev/eval/cumulative/runbook_vs_skill/README.md for the decision frame this
harness's --summarize mode applies.

Reused plumbing (never reinvented):
  * isolation.py — per-trial ENGRAM_VAULT_PATH/ENGRAM_CHUNKS_DIR/ENGRAM_TRANSCRIPT_DIR, the
    isolated_env()/assert_isolated() contract, and the real-vault fingerprint guard.
  * harness.py — MODELS registry and ENGRAM_BIN_DIR (so `engram` is reachable on PATH exactly
    the way every other dev/eval harness resolves it).
  * The underload_repro/endorse_cue family's marker-validity-gate pattern: a per-run UUID marker
    proves the fixture CLAUDE.md actually reached the trial's context. Verified directly against
    the plumbing trial's real transcript (see report): a fresh headless `claude -p` project cwd
    gets a `"type": "attachment"` record (`attachment.type == "instructions"`, one file entry per
    attached instructions file, `path` = the CLAUDE.md path, `content` = its full text) written
    into the session .jsonl BEFORE the first assistant turn — so a marker line in CLAUDE.md
    appears in the raw transcript text with NO prompt changes and no echo instruction required.
    (Sibling harnesses endorse_cue/underload_repro inline guidance into a fresh temp cwd's
    CLAUDE.md too, but ask the model to echo the marker back — belt-and-suspenders that turned
    out unnecessary here once this attachment record was found; the plumbing trial is the direct
    evidence.) The task prompt sent to `claude -p` is therefore fixtures/sensor-registration.txt's
    contents verbatim, unmodified, identical across arms, per Ruling 4.

Isolation note (see report): isolation.isolated_env(cfg, trial_dir, cwd=repo_path)'s
assert_isolated(env, cwd) walks cwd's ancestors (INCLUDING cwd itself) for a .git/.claude/.hg/.jj
marker and raises if found — correct for its designed purpose (never let a trial's cwd be nested
inside a REAL repo whose ancestor .git/.claude would get swept into the trial's own isolated chunk
index by `engram ingest --auto`), but every trial repo here IS its own freshly `git init`ed fixture
(required by the task: registry staging + commit-count checks are part of FOLLOWED/END-STATE), so
the check trips on every single trial before a single claude call ever runs. This harness calls
isolated_env(cfg, trial_dir, cwd=None) — skipping only the cwd-ancestor scan — then sets
ENGRAM_TRANSCRIPT_DIR itself via the same isolation.project_slug() helper, and still runs
isolation.assert_isolated(env) (the env-var / CLAUDE_CONFIG_DIR checks that actually guard the
operator's real vault/chunks/config) before every trial. isolation.py is out of scope to edit.

Usage:
  python3 probe.py --model sonnet --n 1 --out results/smoke_results.jsonl
  python3 probe.py --model opus --n 5 --out results/opus_results.jsonl [--arms S,R,F]
      [--workers N] [--timeout 900] [--keep]
  python3 probe.py --plumbing --model sonnet
  python3 probe.py --summarize results/smoke_results.jsonl
"""
import argparse
import concurrent.futures as cf
import glob
import json
import os
import re
import shutil
import subprocess
import sys
import time
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
CUM = os.path.dirname(HERE)                          # dev/eval/cumulative
DEV_EVAL = os.path.dirname(CUM)                      # dev/eval
REPO = os.path.dirname(os.path.dirname(DEV_EVAL))    # repo root

sys.path.insert(0, DEV_EVAL)
sys.path.insert(0, CUM)
import isolation                     # noqa: E402  (isolated_env, assert_isolated, project_slug, vault_fingerprint)
import harness as cum_harness        # noqa: E402  (MODELS, ENGRAM_BIN_DIR — single source of truth)
import matrix                        # noqa: E402  (refresh_creds — the keychain seam; NOT build_cfg_template, see below)

FIXTURES_DIR = os.path.join(HERE, "fixtures")
FIXTURE_VAULTS_DIR = os.path.join(HERE, "fixture-vaults")
INIT_SCRIPT = os.path.join(FIXTURES_DIR, "init_fixture_repo.sh")
DONE_WHEN_SCRIPT = os.path.join(FIXTURES_DIR, "done_when_checks.sh")
TASK_PROMPT_PATH = os.path.join(FIXTURES_DIR, "sensor-registration.txt")
SKILL_SRC_DIR = os.path.join(FIXTURE_VAULTS_DIR, "skill", "sensors-add-skill")
RUNBOOK_VAULT_SRC = os.path.join(FIXTURE_VAULTS_DIR, "runbook", "vault")
FACT_VAULT_SRC = os.path.join(FIXTURE_VAULTS_DIR, "fact", "vault")
GUIDANCE_PATH = os.path.join(REPO, "agent-instructions", "guidance", "recall.md")

MODELS = cum_harness.MODELS
ENGRAM_BIN_DIR = cum_harness.ENGRAM_BIN_DIR

DEFAULT_RUN_ROOT = os.environ.get(
    "RUNBOOK_VS_SKILL_ROOT",
    "/private/tmp/claude-501/-Users-joe-repos-personal-engram/"
    "23a08637-d9a1-431d-9ac8-7770901edb97/scratchpad/runbook_vs_skill_runs",
)

ARMS = ("S", "R", "F")
DEFAULT_TIMEOUT_S = 900
TRANSIENT_BACKOFFS = (0, 15, 45, 120)
MIN_VALID_COST_USD = 0.02

# ----- fifth recall cue (PLAN.md "Fifth Recall Cue Draft") — inlined into fixture CLAUDE.md only;
# the repo's own agent-instructions/guidance/recall.md is never edited. -----
FIFTH_CUE = (
    "- **Before you start a multi-step task** — one you may have done before, in this or a "
    "similar situation (the usual test/tag/push dance, a dependency bump, a cleanup pass) — \n"
    "  **run `/recall glance` first** (the action is recalling the situation, not reasoning "
    "from scratch) — the vault may hold a runbook for this or a similar situation.\n"
)

PROC_PATH_NEEDLES = ("lib/sensors/registry.txt", "scripts/sensors.py", "migrations/", "TELEMETRY_CHANGELOG.log")
PROC_BASH_EXTRA_NEEDLES = ("git add", "make validate")


# ----- CLAUDE.md construction (Ruling 1) -----

def build_claude_md(marker):
    """agent-instructions/guidance/recall.md, verbatim (never @imported), with the fifth cue
    bullet appended to the cue list, plus the run's PROBE-TOKEN marker line. Identical across
    all three arms."""
    guidance = open(GUIDANCE_PATH).read()
    anchor = "\nEscalate to"
    idx = guidance.index(anchor)
    with_cue = guidance[:idx] + "\n" + FIFTH_CUE + guidance[idx:]
    return with_cue.rstrip() + f"\n\nPROBE-TOKEN: {marker}\n"


# ----- shared cfg (Ruling 3) -----

def build_cfg_template(dst):
    """Isolated CLAUDE_CONFIG_DIR carrying the repo's REAL recall+learn skills (warm), mirroring
    matrix.py::build_cfg_template(dst, warm=True) — but NOT calling it directly. Verified
    (see report): matrix.py's skill source is `REPO/skills/<skill>`, which does not exist in
    this repo (the skills live at `REPO/agent-instructions/skills/<skill>`, per this repo's own
    CLAUDE.md directory-structure section). Calling matrix.build_cfg_template as-is silently
    copies NOTHING (its `if os.path.isdir(src)` guard just skips), producing a cfg with no
    recall/learn skill — which would fail this eval's own pre-registered "recall delivery" smoke
    bar. matrix.py is out of scope to edit from this task, so this is a corrected LOCAL copy
    pointed at the real path; idempotent like the original (skips a rebuild once skills/ exists).
    """
    if os.path.exists(os.path.join(dst, ".claude.json")) and os.path.isdir(os.path.join(dst, "skills")):
        return
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
    for skill in ("recall", "learn"):
        src = os.path.join(REPO, "agent-instructions", "skills", skill)
        if os.path.isdir(src):
            shutil.copytree(src, os.path.join(dst, "skills", skill))


# ----- per-trial isolation env (Ruling 2) -----

def trial_env(cfg, trial_dir, repo_path):
    """isolation.isolated_env with the cwd-ancestor scan skipped (see module docstring) but the
    env-var isolation contract (assert_isolated) still enforced."""
    env = isolation.isolated_env(cfg, trial_dir, cwd=None, base=os.environ)
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", isolation.project_slug(repo_path))
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    isolation.assert_isolated(env)
    return env


def _real_vault_fingerprint():
    """(file count, newest mtime) for the operator's real vault dir — the abort-report guard
    (Ruling 2). Returns (0, None) if the vault doesn't exist (nothing to change)."""
    vault = isolation.operator_vault()
    try:
        names = os.listdir(vault)
    except FileNotFoundError:
        return 0, None
    mtimes = [os.path.getmtime(os.path.join(vault, n)) for n in names]
    return len(names), (max(mtimes) if mtimes else None)


# ----- trial cwd / vault setup -----

def setup_trial_repo(trial_dir, arm, marker):
    """init_fixture_repo.sh into trial_dir/repo, write CLAUDE.md (+ skill for Arm S), then
    `git add -A && git commit -m "add project config"` so the trial starts on a clean tree
    (Ruling 2)."""
    repo_path = os.path.join(trial_dir, "repo")
    subprocess.run(["bash", INIT_SCRIPT, repo_path], check=True, capture_output=True, text=True)
    with open(os.path.join(repo_path, "CLAUDE.md"), "w") as f:
        f.write(build_claude_md(marker))
    if arm == "S":
        dst = os.path.join(repo_path, ".claude", "skills", "sensors-add-skill")
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copytree(SKILL_SRC_DIR, dst)
    subprocess.run(["git", "-C", repo_path, "add", "-A"], check=True, capture_output=True)
    subprocess.run(["git", "-C", repo_path, "commit", "-m", "add project config"],
                    check=True, capture_output=True)
    return repo_path


def prepare_trial_vault(env, arm):
    """Arm R/F: replace the auto-created empty trial vault with a copy of the fixture vault.
    Arm S: leave the empty vault isolated_env already created (never the operator's real vault)."""
    if arm not in ("R", "F"):
        return
    vault = env["ENGRAM_VAULT_PATH"]
    shutil.rmtree(vault, ignore_errors=True)
    src = RUNBOOK_VAULT_SRC if arm == "R" else FACT_VAULT_SRC
    shutil.copytree(src, vault)


def fixture_note_basename(arm):
    """The single fixture note's basename (no .md), read from the vault dir at runtime, per
    Ruling 5 ('read it from the vault dir at runtime'). Only meaningful for Arm R/F."""
    if arm not in ("R", "F"):
        return None
    src = RUNBOOK_VAULT_SRC if arm == "R" else FACT_VAULT_SRC
    for name in os.listdir(src):
        if name.endswith(".md"):
            return name[: -len(".md")]
    return None


# ----- claude -p invocation -----

def _loadj_str(txt):
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


def _call_cost(out):
    return round(float((out.get("total_cost_usd") if isinstance(out, dict) else 0.0) or 0.0), 4)


def _degraded(out):
    return bool(out.get("is_error")) and _call_cost(out) < MIN_VALID_COST_USD


def spawn_claude(env, model, cwd, prompt, timeout_s):
    """`claude -p` subprocess idiom (harness.py::claude's flags/PATH/permissions/output-format),
    reimplemented locally because harness.claude() does not expose a wall-clock timeout — Ruling 3
    requires one, with timed_out recorded and repo state still scored. Retries the transient
    degraded-call signature (harness.py's own do_build backoff pattern)."""
    args = ["claude", "-p", prompt, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out, timed_out = {}, False
    for backoff in TRANSIENT_BACKOFFS:
        if backoff:
            time.sleep(backoff)
        try:
            r = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True, timeout=timeout_s)
        except subprocess.TimeoutExpired:
            timed_out = True
            break
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = _loadj_str(r.stdout)
        if not _degraded(out):
            break
    return out, timed_out


# ----- transcript discovery + parsing -----

def discover_transcript_paths(cfg, repo_path):
    proj_dir = os.path.join(cfg, "projects", isolation.project_slug(repo_path))
    return sorted(glob.glob(os.path.join(proj_dir, "**", "*.jsonl"), recursive=True))


def transcript_raw_text(paths):
    text = []
    for path in paths:
        try:
            text.append(open(path, errors="replace").read())
        except OSError:
            continue
    return "\n".join(text)


def parse_transcript_events(paths):
    """Chronologically ordered tool_use/tool_result events across every transcript file for a
    trial, sorted by the record's `timestamp` field (falls back to file-then-line order when a
    timestamp is missing, which keeps single-file trials — the common case here — exactly in
    line order). Each event: idx (position in the returned order), kind, name (tool_use only),
    input (tool_use only), id (tool_use only, its tool_use_id), tool_use_id (tool_result only),
    content (tool_result only, stringified)."""
    raw = []
    for path in paths:
        try:
            lines = open(path, errors="replace").read().splitlines()
        except OSError:
            continue
        for line_no, line in enumerate(lines):
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except Exception:
                continue
            ts = obj.get("timestamp") or ""
            message = obj.get("message") or {}
            content = message.get("content")
            if not isinstance(content, list):
                continue
            for block in content:
                if not isinstance(block, dict):
                    continue
                btype = block.get("type")
                if btype == "tool_use":
                    raw.append((ts, path, line_no, {
                        "kind": "tool_use", "name": block.get("name"),
                        "input": block.get("input") or {}, "id": block.get("id"),
                    }))
                elif btype == "tool_result":
                    result_content = block.get("content")
                    text = result_content if isinstance(result_content, str) else json.dumps(result_content)
                    raw.append((ts, path, line_no, {
                        "kind": "tool_result", "tool_use_id": block.get("tool_use_id"),
                        "content": text or "",
                    }))
    raw.sort(key=lambda t: (t[0], t[1], t[2]))
    events = []
    for idx, (_, _, _, ev) in enumerate(raw):
        ev["idx"] = idx
        events.append(ev)
    return events


def is_marker_seen(raw_text, marker):
    return marker in raw_text


def _tool_result_text(events, tool_use_id):
    for ev in events:
        if ev["kind"] == "tool_result" and ev.get("tool_use_id") == tool_use_id:
            return ev.get("content", "")
    return ""


# ----- procedure-step detection (Ruling 5) -----

def is_procedure_step(event):
    if event["kind"] != "tool_use":
        return False
    name = event.get("name")
    inp = event.get("input") or {}
    if name in ("Edit", "Write", "MultiEdit"):
        file_path = inp.get("file_path", "") or ""
        return any(needle in file_path for needle in PROC_PATH_NEEDLES)
    if name == "Bash":
        command = inp.get("command", "") or ""
        return any(needle in command for needle in PROC_PATH_NEEDLES + PROC_BASH_EXTRA_NEEDLES)
    return False


def first_procedure_step_index(events):
    for ev in events:
        if is_procedure_step(ev):
            return ev["idx"]
    return None


# ----- FOUND (Ruling 5) -----

def score_found(arm, events, note_basename):
    first_step_idx = first_procedure_step_index(events)

    def _before_step(idx):
        return first_step_idx is None or idx < first_step_idx

    if arm == "S":
        for ev in events:
            if (ev["kind"] == "tool_use" and ev.get("name") == "Skill"
                    and (ev.get("input") or {}).get("skill") == "sensors-add-skill"
                    and _before_step(ev["idx"])):
                return True, ev["idx"]
        return False, None

    # Arm R / Arm F
    for ev in events:
        if not (ev["kind"] == "tool_use" and ev.get("name") == "Bash"
                and "engram query" in ((ev.get("input") or {}).get("command", "") or "")):
            continue
        if not _before_step(ev["idx"]):
            continue
        result_text = _tool_result_text(events, ev.get("id"))
        if note_basename and note_basename in result_text:
            return True, ev["idx"]
    return False, None


def score_recall_fired(events):
    return any(ev["kind"] == "tool_use" and ev.get("name") == "Skill"
               and (ev.get("input") or {}).get("skill") == "recall" for ev in events)


# ----- FOLLOWED per-step checks (Ruling 5) -----

_REGISTRY_LINE_RE = re.compile(r"^pressure_v2:[0-9]+(?:\.[0-9]+)*\t[^\t\n]+\t[0-9]{4}-", re.MULTILINE)
_CHANGELOG_LINE_RE = re.compile(r"\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] \S+ Added sensor pressure_v2:[0-9.]+")


def step1_registry(repo_path):
    path = os.path.join(repo_path, "lib", "sensors", "registry.txt")
    try:
        text = open(path).read()
    except OSError:
        return False
    return bool(_REGISTRY_LINE_RE.search(text))


def step2_codegen(repo_path, events):
    for ev in events:
        if ev["kind"] == "tool_use" and ev.get("name") == "Bash":
            if "sensors.py" in ((ev.get("input") or {}).get("command", "") or ""):
                return True
    types_go = os.path.join(repo_path, "lib", "sensors", "types.go")
    try:
        return "pressure_v2" in open(types_go).read()
    except OSError:
        return False


def step3_migration(repo_path):
    matches = glob.glob(os.path.join(repo_path, "migrations", "*_sensor_pressure_v2.go"))
    if not matches:
        return False
    try:
        text = open(matches[0]).read()
    except OSError:
        return False
    return "func init()" in text and "registerSensor" in text


def step4_changelog(repo_path):
    path = os.path.join(repo_path, "TELEMETRY_CHANGELOG.log")
    try:
        text = open(path).read()
    except OSError:
        return False
    return bool(_CHANGELOG_LINE_RE.search(text))


def step5_staged_no_commit(repo_path):
    staged = subprocess.run(["git", "-C", repo_path, "diff", "--cached", "--name-only"],
                             capture_output=True, text=True)
    if "lib/sensors/registry.txt" not in staged.stdout:
        return False
    log = subprocess.run(["git", "-C", repo_path, "log", "--oneline"], capture_output=True, text=True)
    commit_count = len([line for line in log.stdout.splitlines() if line.strip()])
    return commit_count == 2  # "Initial commit..." + "add project config"; no new commit


def step6_validate(events):
    for ev in events:
        if ev["kind"] == "tool_use" and ev.get("name") == "Bash":
            if "make validate" in ((ev.get("input") or {}).get("command", "") or ""):
                return True
    return False


def score_followed(repo_path, events):
    steps = {
        "1": step1_registry(repo_path),
        "2": step2_codegen(repo_path, events),
        "3": step3_migration(repo_path),
        "4": step4_changelog(repo_path),
        "5": step5_staged_no_commit(repo_path),
        "6": step6_validate(events),
    }
    return steps, sum(1 for v in steps.values() if v)


# ----- END-STATE (Ruling 5) -----

def check_end_state(repo_path):
    r = subprocess.run(["bash", DONE_WHEN_SCRIPT, repo_path], capture_output=True, text=True)
    return r.returncode == 0, (r.stdout + r.stderr).strip()


# ----- one trial -----

def run_one_trial(run_root, cfg, arm, model, trial_index, marker, timeout_s):
    trial_dir = os.path.join(run_root, "trials", f"{arm}-{trial_index}")
    os.makedirs(trial_dir, exist_ok=True)
    t0 = time.time()

    repo_path = setup_trial_repo(trial_dir, arm, marker)
    env = trial_env(cfg, trial_dir, repo_path)
    prepare_trial_vault(env, arm)
    note_basename = fixture_note_basename(arm) if arm in ("R", "F") else None

    # Ruling 4: the task prompt is fixtures/sensor-registration.txt's contents verbatim,
    # identical across arms. No marker-echo suffix needed — see module docstring: the fixture
    # CLAUDE.md's marker line reaches the transcript via Claude Code's own "attachment" record.
    prompt = open(TASK_PROMPT_PATH).read().strip()

    result, timed_out = {}, False
    error = None
    try:
        result, timed_out = spawn_claude(env, model, repo_path, prompt, timeout_s)
    except Exception as exc:  # noqa: BLE001 — record and keep scoring repo state
        error = str(exc)

    transcript_paths = discover_transcript_paths(cfg, repo_path)
    raw_text = transcript_raw_text(transcript_paths)
    events = parse_transcript_events(transcript_paths)

    marker_seen = is_marker_seen(raw_text, marker)
    found, found_idx = score_found(arm, events, note_basename)
    first_step_idx = first_procedure_step_index(events)
    recall_fired = score_recall_fired(events)
    followed_steps, followed_k = score_followed(repo_path, events)
    end_state, end_state_output = check_end_state(repo_path)

    record = {
        "arm": arm, "trial_index": trial_index, "model": model,
        "trial_dir": trial_dir, "repo_path": repo_path,
        "transcript_path": transcript_paths[0] if transcript_paths else None,
        "valid": marker_seen, "timed_out": timed_out, "error": error,
        "marker_seen": marker_seen,
        "found": found, "found_index": found_idx,
        "recall_fired": recall_fired,
        "first_procedure_step_index": first_step_idx,
        "followed_steps": followed_steps, "followed_k": followed_k,
        "end_state": end_state, "end_state_output": end_state_output,
        "total_cost_usd": _call_cost(result),
        "duration_ms": result.get("duration_ms") if isinstance(result, dict) else None,
        "num_turns": result.get("num_turns") if isinstance(result, dict) else None,
        "session_id": result.get("session_id") if isinstance(result, dict) else None,
        "wall_s": round(time.time() - t0, 1),
    }
    return record


# ----- batch run -----

def append_jsonl(path, record):
    out_dir = os.path.dirname(os.path.abspath(path))
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    with open(path, "a") as f:
        f.write(json.dumps(record) + "\n")
        f.flush()
        os.fsync(f.fileno())


def run_batch(args):
    arms = [a.strip() for a in args.arms.split(",") if a.strip()]
    unknown = [a for a in arms if a not in ARMS]
    if unknown:
        raise SystemExit(f"unknown arm(s) {unknown}; choose from {ARMS}")

    run_id = f"{args.model}-{int(time.time())}-{uuid.uuid4().hex[:6]}"
    run_root = os.path.join(DEFAULT_RUN_ROOT, run_id)
    os.makedirs(run_root, exist_ok=True)
    cfg = os.path.join(run_root, "cfg")
    build_cfg_template(cfg)
    matrix.refresh_creds(cfg)

    marker = f"RUNBOOK-VS-SKILL-PROBE-{uuid.uuid4().hex[:8]}"
    before_fp = _real_vault_fingerprint()

    jobs = [(arm, i) for arm in arms for i in range(args.n)]
    print(f"run_id={run_id} arms={arms} n={args.n} model={args.model} "
          f"trials={len(jobs)} timeout={args.timeout}s root={run_root}")

    try:
        with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
            futs = {ex.submit(run_one_trial, run_root, cfg, arm, args.model, i, marker, args.timeout): (arm, i)
                    for arm, i in jobs}
            for fut in cf.as_completed(futs):
                arm, i = futs[fut]
                record = fut.result()
                record["run_id"] = run_id
                append_jsonl(args.out, record)
                status = "valid" if record["valid"] else "INVALID(no-marker)"
                print(f"  [{arm}#{i}] {status} found={record['found']} followed={record['followed_k']}/6 "
                      f"end_state={record['end_state']} cost=${record['total_cost_usd']:.2f} "
                      f"timed_out={record['timed_out']}")
    finally:
        after_fp = _real_vault_fingerprint()
        if after_fp != before_fp:
            print(f"ABORT-REPORT: operator's real vault fingerprint changed! before={before_fp} "
                  f"after={after_fp}. A trial may have reached real memory. Investigate before "
                  "trusting any result in this run.", file=sys.stderr)
        if not args.keep:
            shutil.rmtree(run_root, ignore_errors=True)


# ----- plumbing mode (Ruling 8) -----

PLUMBING_PROMPT = ("Run /recall glance for: registering a sensor type in this telemetry repo. "
                    "Report what the vault returned.")


def run_plumbing(model):
    run_id = f"plumbing-{model}-{int(time.time())}"
    run_root = os.path.join(DEFAULT_RUN_ROOT, run_id)
    os.makedirs(run_root, exist_ok=True)
    cfg = os.path.join(run_root, "cfg")
    build_cfg_template(cfg)
    matrix.refresh_creds(cfg)

    before_fp = _real_vault_fingerprint()
    marker = f"RUNBOOK-VS-SKILL-PROBE-{uuid.uuid4().hex[:8]}"
    trial_dir = os.path.join(run_root, "trials", "plumbing-0")
    os.makedirs(trial_dir, exist_ok=True)
    repo_path = setup_trial_repo(trial_dir, "R", marker)
    env = trial_env(cfg, trial_dir, repo_path)
    prepare_trial_vault(env, "R")
    note_basename = fixture_note_basename("R")

    result, timed_out = spawn_claude(env, model, repo_path, PLUMBING_PROMPT, DEFAULT_TIMEOUT_S)
    after_fp = _real_vault_fingerprint()

    transcript_paths = discover_transcript_paths(cfg, repo_path)
    raw_text = transcript_raw_text(transcript_paths)
    events = parse_transcript_events(transcript_paths)

    marker_seen = is_marker_seen(raw_text, marker)
    recall_fired = score_recall_fired(events)
    query_events = [ev for ev in events if ev["kind"] == "tool_use" and ev.get("name") == "Bash"
                    and "engram query" in ((ev.get("input") or {}).get("command", "") or "")]
    query_ran = bool(query_events)
    note_surfaced = False
    for ev in query_events:
        if note_basename and note_basename in _tool_result_text(events, ev.get("id")):
            note_surfaced = True
            break

    print(f"marker_seen={marker_seen}")
    print(f"recall_skill_fired={recall_fired}")
    print(f"engram_query_ran={query_ran}")
    print(f"note_surfaced={note_surfaced}")
    print(f"cost_usd={_call_cost(result)}")
    print(f"timed_out={timed_out}")
    print(f"transcript_path={transcript_paths[0] if transcript_paths else None}")
    print(f"run_root={run_root}")
    if after_fp != before_fp:
        print(f"ABORT-REPORT: operator's real vault fingerprint changed! before={before_fp} "
              f"after={after_fp}.", file=sys.stderr)


# ----- summarize / decision frame (Ruling 9, PLAN.md lines 518-537) -----

def load_jsonl(path):
    rows = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def aggregate(records, arm):
    arm_records = [r for r in records if r.get("arm") == arm]
    valid = [r for r in arm_records if r.get("valid")]
    n = len(arm_records)
    valid_n = len(valid)
    found_n = sum(1 for r in valid if r.get("found"))
    end_state_n = sum(1 for r in valid if r.get("end_state"))
    recall_fired_n = sum(1 for r in valid if r.get("recall_fired"))
    followed_trial_equiv = sum((r.get("followed_k") or 0) / 6.0 for r in valid)
    followed_mean = (followed_trial_equiv / valid_n) if valid_n else 0.0
    cost_mean = (sum(r.get("total_cost_usd") or 0 for r in valid) / valid_n) if valid_n else 0.0
    durations = [r.get("duration_ms") for r in valid if r.get("duration_ms")]
    duration_mean_s = (sum(durations) / len(durations) / 1000.0) if durations else 0.0
    return {
        "n": n, "valid_n": valid_n, "found_n": found_n, "end_state_n": end_state_n,
        "recall_fired_n": recall_fired_n, "followed_trial_equiv": followed_trial_equiv,
        "followed_mean": followed_mean, "cost_mean": cost_mean, "duration_mean": duration_mean_s,
    }


def _gap_verdict(skill_val, other_val):
    gap = other_val - skill_val
    if abs(gap) <= 1:
        return "indistinguishable"
    return "better" if gap >= 2 else "worse"


def decision_frame(agg):
    """PLAN.md lines 518-537's parity + functionality decision frame. FOLLOWED's per-trial mean
    (k/6) is converted to a 'trial-equivalent' scalar (sum of k_i/6 across trials, same 0..n scale
    as FOUND/END-STATE's trial counts) so the '1 trial' / '2+ trials' gap language applies
    uniformly across all three metrics — PLAN.md states the frame for count metrics; this is the
    documented generalization for the continuous FOLLOWED mean (see README)."""
    s, r, f = agg.get("S"), agg.get("R"), agg.get("F")
    parity = {}
    if s and r:
        parity["found"] = _gap_verdict(s["found_n"], r["found_n"])
        parity["followed"] = _gap_verdict(s["followed_trial_equiv"], r["followed_trial_equiv"])
        parity["end_state"] = _gap_verdict(s["end_state_n"], r["end_state_n"])
    baseline_uninterpretable = bool(s and s["valid_n"] and s["end_state_n"] < 0.6 * s["valid_n"])
    fact = {"verdict": "cant_distinguish"}
    if r and f:
        followed_gap = r["followed_trial_equiv"] - f["followed_trial_equiv"]
        end_state_gap = r["end_state_n"] - f["end_state_n"]
        if followed_gap >= 2 or end_state_gap >= 2:
            fact["verdict"] = "runbook_exceeds_fact"
        fact["followed_gap"] = followed_gap
        fact["end_state_gap"] = end_state_gap
    return {"parity": parity, "baseline_uninterpretable": baseline_uninterpretable, "fact": fact}


def format_table(agg):
    arms = [a for a in ARMS if a in agg]
    lines = []
    header = "metric".ljust(20) + "".join(a.ljust(16) for a in arms)
    lines.append(header)
    lines.append("-" * len(header))

    def row(label, fmt):
        cells = [fmt(agg[a]) for a in arms]
        lines.append(label.ljust(20) + "".join(c.ljust(16) for c in cells))

    row("FOUND (k/n)", lambda a: f"{a['found_n']}/{a['valid_n']}")
    row("FOLLOWED (mean k/6)", lambda a: f"{a['followed_mean']*6:.2f}/6")
    row("END-STATE (k/n)", lambda a: f"{a['end_state_n']}/{a['valid_n']}")
    row("recall_fired (k/n)", lambda a: f"{a['recall_fired_n']}/{a['valid_n']}")
    row("cost (mean USD)", lambda a: f"${a['cost_mean']:.2f}")
    row("duration (mean s)", lambda a: f"{a['duration_mean']:.0f}")
    row("valid (n)", lambda a: f"{a['valid_n']}/{a['n']}")
    return "\n".join(lines)


def summarize_file(path):
    records = load_jsonl(path)
    agg = {arm: aggregate(records, arm) for arm in ARMS if any(r.get("arm") == arm for r in records)}
    print(format_table(agg))
    frame = decision_frame(agg)
    print()
    print("Decision frame:")
    print(json.dumps(frame, indent=2))
    return agg, frame


# ----- CLI -----

def build_argparser():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--model", choices=list(MODELS))
    ap.add_argument("--n", type=int)
    ap.add_argument("--out")
    ap.add_argument("--arms", default=",".join(ARMS))
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT_S)
    ap.add_argument("--keep", action="store_true", help="keep the run root instead of deleting it on exit")
    ap.add_argument("--plumbing", action="store_true")
    ap.add_argument("--summarize")
    return ap


def main(argv=None):
    args = build_argparser().parse_args(argv)
    if args.summarize:
        summarize_file(args.summarize)
        return
    if args.plumbing:
        run_plumbing(args.model or "sonnet")
        return
    if not (args.model and args.n and args.out):
        build_argparser().error("--model, --n, and --out are required for a normal run")
    run_batch(args)


if __name__ == "__main__":
    main()
