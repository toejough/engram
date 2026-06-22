#!/usr/bin/env python3
"""One operation of the cumulative-accumulation eval (v3 — 2-regime modern design).

Two regimes only: `cold` (no memory) and `real.full` (complete modern memory system).
Modern engram has NO tiers/episodes/eager-L2. Memory = raw chunks (engram ingest) +
crystallized notes (engram learn fact|feedback). Recall = engram query → unified
clustering → /recall skill judges covered/near/ABSENT and crystallizes NEW notes lazily.

Each app is ONE one-session cell:
  build  [real.full: invoke /recall] -> build -> score -> feed back gaps -> loop ->
         [real.full: invoke /learn after convergence]
         Records: stated_conventions, convention_statements, arch_pass, recall_fired,
         escalation, timing (recall_s/build_s/learn_s), cost, notes_written,
         crystallizations_at_recall, chunks_ingested, learn_kind_breakdown.

Usage:
  harness.py build --app feeds --model sonnet --regime real.full --trial 1 \
      --vault-in <dir|none> --cfg <cfgdir> --workdir <dir> --spec <spec.json> \
      --out <build.json> [--max-rounds 8] [--stub good|naive]
  harness.py learn  (legacy only; real.full learns in-session — this mode remains for
      the cold/stub path which writes nothing)
"""
import argparse, glob as _glob, json, os, re, subprocess, sys, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import score as scoremod

# Single editable source of truth for the model registry — a new model is a one-line add (§1.5).
MODELS = {"haiku": "claude-haiku-4-5-20251001", "sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
ENGRAM_BIN_DIR = os.environ.get("ENGRAM_BIN_DIR", os.path.expanduser("~/go/bin"))
SCHEMA_VERSION = 4
CONVERGE_ARCH_BAR = 8  # arch_pass >= 8 (matches converged())
STALL_PATIENCE = 3  # halt the build loop if convergence score is flat this many consecutive rounds.
# Loosened 2→3 after haiku n=5: patience=2 cut 14/30 builds with round-budget to spare (only 1 genuine
# max-rounds hit), firing on builds still making slow monotone progress. Its original motivation —
# feeds-real.full plateauing at 9/18 rounds 3–8 — was the cmd/-layout build bug, since fixed; with that
# gone, 2 flat rounds is too aggressive a plateau call. 3 still caps truly-stuck tail spend.

# Two regimes — modern engram (v3, no tiers/episodes/eager-L2):
#   cold      = no memory at all; baseline
#   real.full = complete modern system: between apps the harness runs `engram ingest` on the
#               build transcript (chunks), and the agent invokes the real /learn skill
#               (writes fact/feedback notes); on each new app the agent invokes the real /recall
#               skill (unified clustering + lazy crystallization at recall time).
# read_mode: none | skill
REGIMES = {
    "cold":      {"write": "none",  "read_mode": "none"},
    "real.full": {"write": "skill", "read_mode": "skill"},
}


def engram_sha():
    try:
        here = os.path.dirname(os.path.abspath(__file__))
        r = subprocess.run(["git", "-C", here, "rev-parse", "HEAD"], capture_output=True, text=True, timeout=10)
        return r.stdout.strip()[:12] or "unknown"
    except Exception:
        return "unknown"


def loadj_str(txt):
    best = {}
    for line in txt.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            o = json.loads(line)
        except Exception:
            continue
        if isinstance(o, dict) and ("total_cost_usd" in o or o.get("type") == "result"):
            best = o
    return best


def claude(cfg, model, vault, cwd, prompt, resume_sid=None, chunks=None):
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "64000"
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    # `engram transcript` defaults to ~/.claude/projects/<slug> and IGNORES CLAUDE_CONFIG_DIR, so in a
    # headless cell it never finds the session and /learn falls back to hand-written --transcript-text
    # episodes (not real chunks). Point it at THIS cfg's session dir.
    if cwd:
        env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _project_slug(cwd))
    if vault and vault != "none":
        env["ENGRAM_VAULT_PATH"] = vault
    if chunks:
        env["ENGRAM_CHUNKS_DIR"] = chunks  # the /recall chunk-variant skill reads this
    args = ["claude", "-p", prompt, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    if resume_sid:
        args = ["claude", "--resume", resume_sid] + args[1:]
    r = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True)
    try:
        return json.loads(r.stdout)
    except Exception:
        return loadj_str(r.stdout)


def _project_slug(cwd):
    """Claude Code's project-dir name for a cwd: the realpath with every non-alphanumeric
    character mapped to '-' (verified empirically: '.' becomes '-' too, so a workdir named
    real.auto lands in ...-real-auto — a bare '/'-only replace MISSES the session dir)."""
    import re
    return re.sub(r"[^A-Za-z0-9-]", "-", os.path.realpath(cwd))


def _find_session_transcript(cfg, sid):
    """Locate <sid>.jsonl under cfg/projects regardless of slug scheme — robust against any
    future change in Claude's path sanitization."""
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        if f"{sid}.jsonl" in files:
            return os.path.join(root, f"{sid}.jsonl")
    return ""


def refresh_creds_path(cfg):
    # refresh the cfg's keychain cred between retries, in case a long run aged it out
    try:
        subprocess.run(["bash", "-c",
            f'security find-generic-password -s "Claude Code-credentials" -w '
            f'> {cfg}/.credentials.json && chmod 600 {cfg}/.credentials.json'],
            capture_output=True, timeout=15)
    except Exception:
        pass


def build_prompt(app, interface, read_mode):
    """Build prompt with read-mode-appropriate recall. Real-skill regimes only (recall-v2)."""
    if read_mode == "none":
        recall = ""
    elif read_mode == "skill":
        recall = (
            "\nBefore writing ANY code, consult your memory by INVOKING YOUR /recall SKILL — actually "
            "run the skill (it prints its Step 0 plan, queries the vault, and synthesizes impact). Do "
            "NOT hand-run `engram query` yourself in place of the skill. Frame the recall around "
            f"building a command-line {app} in Go and its architecture/conventions. Read every note the "
            "skill surfaces and APPLY every convention and decision — including any note the skill "
            "crystallizes on demand — as requirements for your build.\n")
    else:
        raise ValueError(f"Unknown read_mode {read_mode!r}; regimes use none|skill")
    return (f"Build a command-line {app} manager in Go, from scratch, in the current directory "
            f"(run `go mod init {app}` first).\n\nImplement these subcommands:\n{interface}\n{recall}\n"
            "Make `go test ./...` pass before you finish. Work fully autonomously: never stop to ask "
            "questions; keep going until it compiles and tests pass. Make changes by editing files "
            "directly with your tools; work across several steps; no need to reprint whole files. "
            "When done give a one-line summary.")


# Prescriptive, code-level fix per ARCH detector — used to ESCALATE feedback on a convention that
# a (usually weak) model keeps failing. It is fair to tell the model exactly what to write (§4);
# the say-once metric is round-1-based, so escalation in later rounds never changes it — it only
# drives the app to completion so the chain (learn → transfer) is valid.
ARCH_PRESCRIPTIONS = {
    "di": "Define a storage interface (e.g. `type Store interface { Load() ([]T, error); Save([]T) error }`) "
          "and pass it INTO your command logic as a parameter. Command handlers must call the interface, "
          "never os.ReadFile/os.WriteFile directly — so a test can pass an in-memory fake.",
    "sentinel": "Declare a package-level sentinel: `var ErrNotFound = errors.New(\"not found\")`. Return it "
                "wrapped (`fmt.Errorf(\"...: %w\", ErrNotFound)`) when an item isn't found, and detect it "
                "with `errors.Is(err, ErrNotFound)`.",
    "atomic": "Make saves crash-safe: write to a temp file (`os.CreateTemp(dir, ...)`), then `os.Rename` it "
              "over the real file. Never write the target file in place.",
    "stdlib": "Use only the Go standard library — remove every third-party `require` from go.mod and the imports.",
    "tests_fake_parallel": "Add `t.Parallel()` to every test, and write an in-memory implementation of your "
                           "storage interface (a struct holding a slice) to drive the tests — tests must not "
                           "touch real files.",
    "json": "Add a `--json` flag to list/get and encode the output with `json.NewEncoder(os.Stdout).Encode(v)`.",
    "nocolor": "Before emitting any ANSI color, check `os.Getenv(\"NO_COLOR\")` and whether stdout is a TTY; "
               "when NO_COLOR is set or stdout isn't a terminal, print with zero escape codes.",
    "wrapped_errors": "Wrap every returned error with context: `return fmt.Errorf(\"doing X: %w\", err)` — never "
                      "`return err` bare.",
    "named_perms": "Replace bare octal file-mode literals with named constants: "
                   "`const filePerm os.FileMode = 0o600` (and `dirPerm = 0o750`), and pass those.",
    "no_global_data": "Remove package-level mutable vars (e.g. `var items []T`). Hold that state in a struct you "
                      "construct and pass; no global mutable state.",
}


def _spec_check_detail(spec, name):
    """For a failed behavioral check, return a concrete 'run X, expect Y' instruction from the spec's
    own steps — the escalation for a feature gap (tell it the exact command + expected result)."""
    for c in spec.get("checks", []):
        if c["name"] != name:
            continue
        steps = c.get("steps") or (c.get("variants") or [[]])[0]
        seq = " ; ".join(f"`{c.get('app','app')} " + " ".join(s["argv"]) + "`" for s in steps if s.get("argv"))
        want = next((s["assert"] for s in steps if s.get("assert")), "")
        return f"Concretely: run {seq} and it must satisfy `{want}`."
    return ""


def feedback_prompt(failed, stated_counts=None, spec=None):
    """States ALL gaps (convention + feature; fair to tell the model what you want, §4), and
    ESCALATES granularity for any item that has already been stated before (stuck): first as a
    user-symptom, then with the literal code-level fix. stated_counts maps label -> #times already
    fed back; spec supplies the behavioral 'run X expect Y' detail. The harness counts conventions
    at round 1 only, so escalation never inflates the say-once metric — it just forces completion."""
    stated_counts = stated_counts or {}
    spec = spec or {}
    lines = []
    for label, sym in failed:
        stuck = stated_counts.get(label, 0)
        line = f"- {sym}"
        if stuck >= 1:  # already told once and still failing — escalate to the concrete fix
            if label.startswith("ARCH:"):
                presc = ARCH_PRESCRIPTIONS.get(label[len("ARCH:"):])
            else:
                presc = _spec_check_detail(spec, label.split(":", 1)[-1])
            if presc:
                line += f"\n  → STILL NOT DONE (told you {stuck}× already). Do exactly this: {presc}"
        lines.append(line)
    body = "\n".join(lines)
    return ("Thanks — it builds, but a few things aren't right yet. Here's what I'm seeing as a user:\n"
            f"{body}\n\nAddress every item. Keep `go test ./...` and `go vet ./...` passing. Edit the "
            "files directly; short summary when done.")


def is_convention(label):
    """The convention/feature split key: the scorer prefixes architecture detectors with
    ARCH: (name-agnostic, transferable conventions); behavioral checks carry an
    alpha:/beta:/native: bucket prefix (app-specific features). §4/§5."""
    return label.startswith("ARCH:")


def split_failed(failed):
    conv = [(lbl, sym) for lbl, sym in failed if is_convention(lbl)]
    feat = [(lbl, sym) for lbl, sym in failed if not is_convention(lbl)]
    return conv, feat


def conv_labels(failed):
    return [lbl[len("ARCH:"):] for lbl, _ in failed if is_convention(lbl)]


# CONVENTION_FACTS templates drive ONLY the --stub deterministic learn (zero-cost pipeline
# validation). The REAL learn is agent-driven: the agent runs its /learn skill so the whole
# memory system (learn AND recall) is exercised. Each entry:
# (situation, subject, predicate, object).
CONVENTION_FACTS = {
    "di": ("When wiring dependencies in a Go CLI", "the storage, clock, and output layers",
           "should be", "injected interfaces (any name) so the core runs against in-memory fakes, not real files"),
    "sentinel": ("When signaling not-found or domain errors in a Go CLI", "error conditions", "should be",
           "package-level sentinel vars (var ErrX = errors.New(...)) wrapped with %w and matched via errors.Is"),
    "atomic": ("When persisting data to a file in a Go CLI", "file writes", "should be",
           "atomic — write a temp file then os.Rename over the target, so a crash mid-write cannot corrupt data"),
    "stdlib": ("When choosing dependencies for a Go CLI", "the implementation", "should",
           "use the Go standard library only — no third-party modules"),
    "tests_fake_parallel": ("When writing tests for a Go CLI package", "tests", "should",
           "call t.Parallel(), drive the core through an in-memory fake of the storage interface, and isolate state with t.TempDir()"),
    "json": ("When producing output from a Go CLI", "output", "should offer",
           "a machine-readable --json mode (encoding/json) alongside the human-readable format"),
    "nocolor": ("When producing terminal output from a Go CLI", "color output", "should",
           "honor NO_COLOR and a non-TTY stdout by emitting no ANSI escape codes"),
    "wrapped_errors": ("When returning errors from a Go CLI", "errors", "should be",
           "wrapped with context via fmt.Errorf(\"...: %w\", err), never returned bare"),
    "named_perms": ("When creating files or directories in a Go CLI", "file-mode permissions", "should be",
           "named constants (e.g. const filePerm os.FileMode = 0o600), not bare octal literals"),
    "no_global_data": ("When structuring state in a Go CLI", "application data", "should",
           "live in injected structs, never package-level mutable vars (globals)"),
}


def skill_learn_prompt():
    """Real-skill learn for the one-session cell: the BUILD agent invokes its /learn skill.
    The skill decides what to crystallize — facts for explicit conventions, feedback for
    corrections. NO episode/tier mandates: those are a proxy artifact from pre-v3 eval design
    that contradicted the shipped /learn SKILL.md (which writes fact|feedback, not episodes)."""
    return (
        "Now capture durable memory from the work you just did, by INVOKING YOUR /learn skill.\n\n"
        "Actually run the /learn skill — do NOT hand-run `engram learn` directly in place of the "
        "skill. Let the skill decide what to crystallize (fact notes for reusable conventions, "
        "feedback notes for corrections). Do NOT impose any episode, tier, or count constraints — "
        "the skill knows what to write.\n\n"
        "Work autonomously; end with a one-line summary of what was written to the vault."
    )


def eg_learn(vault, date, kind, slug, fields, relations):
    """Run one `engram learn <kind>` deterministically; return the created note's basename
    (parsed from the printed path), or None on failure. Used by the --stub learn only."""
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    slug = re.sub(r"[^a-z0-9-]+", "-", slug.lower()).strip("-") or "eval"  # engram slug: [a-z0-9-]+
    cmd = ["engram", "learn", kind, "--slug", slug, "--position", "top", "--source", f"eval harness {date}"]
    for key, val in fields.items():
        cmd += ["--" + key, val]
    for rel in relations:
        cmd += ["--relation", rel]
    res = subprocess.run(cmd, env=env, capture_output=True, text=True)
    for line in (res.stdout or "").strip().splitlines():
        line = line.strip()
        if line.endswith(".md"):
            return os.path.basename(line)[: -len(".md")]
    return None


def glob_notes(vault):
    return _glob.glob(os.path.join(vault, "**", "*.md"), recursive=True)


def converged(sc):
    # feature-complete (all behavioral buckets pass) + strong arch (arch_pass >= 8)
    beh_fail = [f for f in sc.get("failed", []) if not f[0].startswith("ARCH:")]
    return len(beh_fail) == 0 and sc.get("arch_pass", 0) >= CONVERGE_ARCH_BAR


def convergence_score(sc):
    """Build-loop convergence score (Bug 4): arch progress + feature progress = arch_pass minus the
    count of failing FEATURE buckets. Rises monotonically as the build improves (more arch detectors
    pass, fewer feature fails). Drives the convergence-stall early-stop."""
    feat_fails = len([f for f in sc.get("failed", []) if not f[0].startswith("ARCH:")])
    return sc.get("arch_pass", 0) - feat_fails


def run_stall_loop(scores, patience=None):
    """Pure simulation of the build loop's stall accounting over a sequence of per-round scores
    (already computed via convergence_score). Returns (stalled, halt_round): halt_round is 1-based
    (round 1 = initial build) and stalled is True iff the score failed to improve for `patience`
    consecutive rounds. Guards the early-stop without an LLM (Bug 4)."""
    if patience is None:
        patience = STALL_PATIENCE
    best = scores[0]
    no_improve = 0
    for i, s in enumerate(scores[1:], start=2):
        if s > best:
            best, no_improve = s, 0
        else:
            no_improve += 1
        if no_improve >= patience:
            return True, i
    return False, len(scores)


def passed_of(sc):
    try:
        return int(sc.get("total", "0/18").split("/")[0])
    except Exception:
        return 0


def _skill_and_query_hits(cfg, sid, skill, need_query):
    """Count cell-transcript files for `sid` showing the named Skill tool firing (`"skill":"<skill>"`
    — SKILL.md actually loaded). For recall, also require an `engram query`: the old proxy ran
    queries WITHOUT the skill, so the skill marker is the faithful signal the real skill ran."""
    hits = 0
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        for fn in files:
            if sid and sid in fn:
                try:
                    txt = open(os.path.join(root, fn), errors="replace").read()
                except Exception:
                    continue
                fired = f'"skill":"{skill}"' in txt
                queried = ("engram query" in txt) if need_query else True
                if fired and queried:
                    hits += 1
    return hits


def recall_fired(cfg, sid):
    """Count turns that INVOKED the /recall skill (Skill tool fired AND a query ran). Grepping
    `engram query` alone is insufficient — the old proxy ran queries without the skill; the
    `"skill":"recall"` marker is the faithful signal a warm cell actually used memory (§4)."""
    return _skill_and_query_hits(cfg, sid, skill="recall", need_query=True)


def learn_fired(cfg, sid):
    """Whether the /learn skill was invoked in this session (Skill tool fired with skill=learn)."""
    return _skill_and_query_hits(cfg, sid, skill="learn", need_query=False) > 0


def snapshot_notes(vault):
    """Set of markdown note paths present in the vault — a START-of-op snapshot so notes_written
    can be reported as a SESSION DELTA (notes added by THIS op), not the cumulative vault total.
    A warm cell seeds the vault with prior apps' carry-forward notes; counting all of them would
    credit this op with notes it never wrote (retro Bug 1)."""
    return set(glob_notes(vault))


def count_notes_written(vault, baseline=None):
    """Markdown notes THIS op added to the vault. With a baseline (start-of-op snapshot) this is a
    session delta: notes present at end minus notes present at start. Without one, the vault total."""
    if baseline is None:
        return len(glob_notes(vault))
    return len(set(glob_notes(vault)) - baseline)


def count_learn_kind_breakdown(vault, baseline=None):
    """Count notes by frontmatter type (fact vs feedback) — replaces tier breakdown.
    With a baseline snapshot, count only notes THIS op added (session delta), not the vault total."""
    counts = {"fact": 0, "feedback": 0, "other": 0}
    paths = glob_notes(vault) if baseline is None else (set(glob_notes(vault)) - baseline)
    for path in paths:
        try:
            head = open(path, errors="replace").read(400)
        except Exception:
            continue
        kind = "other"
        for line in head.splitlines():
            stripped = line.strip()
            if stripped.startswith("type:"):
                kind = stripped.split(":", 1)[1].strip().strip('"').strip("'")
                break
        counts[kind if kind in counts else "other"] += 1
    return counts


def count_crystallizations_at_recall(cfg, sid):
    """Count engram learn / engram amend invocations the agent fired DURING a build session
    (i.e., at recall time, not at learn time). These are the lazy crystallizations the /recall
    skill writes when it finds cluster members evidence an absent or near-miss note."""
    hits = 0
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        for fn in files:
            if sid and sid in fn:
                try:
                    txt = open(os.path.join(root, fn), errors="replace").read()
                except Exception:
                    continue
                hits += txt.count("engram learn ") + txt.count("engram amend ")
    return hits


def link_followed(cfg, sid):
    """Whether the agent actually followed surfaced links — `engram show` calls or reads of
    Permanent/*.md beyond the surfaced set. Makes direct-vs-followed visible, not assumed (§1.4)."""
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        for fn in files:
            if sid and sid in fn:
                try:
                    txt = open(os.path.join(root, fn), errors="replace").read()
                except Exception:
                    continue
                if "engram show" in txt or "Permanent/" in txt:
                    return True
    return False


# ----- stub builders (no LLM; for the zero-cost dry-run, §7/§13) -----

def _stub_build(args):
    """Drop the chosen fixture Go app into the workdir (real, compilable Go the scorer
    builds and runs) and return a canned result. No LLM call. For real.full stub, also writes
    a couple of fact notes into the build vault so the persist-forward path is exercised."""
    import shutil
    fix = os.path.join(os.path.dirname(os.path.abspath(__file__)), "testdata", args.stub)
    for path in _glob.glob(os.path.join(fix, "*")):
        dst = os.path.join(args.workdir, os.path.basename(path))
        if os.path.isdir(path):
            shutil.copytree(path, dst, dirs_exist_ok=True)
        else:
            shutil.copy(path, dst)
    if args.regime == "real.full":
        _stub_learn_facts(args)
    return {"is_error": False, "total_cost_usd": 0.0, "num_turns": 1,
            "session_id": "stub-build", "result": "stubbed build"}


def _stub_learn_facts(args):
    """Fake the real.full learn: write two fact notes into the build vault so
    persist-forward and the notes_written metric path are exercised without an LLM."""
    build_vault = args.workdir + ".buildvault"
    if not os.path.isdir(build_vault):
        return
    date = args.date or "2026-06-06"
    eg_learn(build_vault, date, "fact", f"stub-{args.app}-di",
             {"situation": f"building a command-line {args.app} in Go",
              "subject": "dependencies", "predicate": "should be",
              "object": "injected interfaces so the core runs against in-memory fakes"}, [])
    eg_learn(build_vault, date, "feedback", f"stub-{args.app}-errors",
             {"situation": f"signaling errors in a command-line {args.app} in Go",
              "behavior": "returning bare errors",
              "impact": "loses context and is uncheckable with errors.Is",
              "action": "use package-level sentinel vars wrapped with %w"}, [])


# ----- token I/O capture + cost audit (§6 / note-17: decompose tokens; reconstruct cost ~1.00x) -----

# Price sheet ($/token), verified 2026-06-02 (carried forward; see results doc). Opus 4.5–4.8 are
# $5/$25, NOT the $15/$75 of Opus 4/4.1 — a real trap when costing from an old table.
PRICES = {
    "claude-opus-4-8":           {"i": 5e-6, "o": 25e-6, "cw": 6.25e-6, "cr": 0.5e-6},
    "claude-sonnet-4-6":         {"i": 3e-6, "o": 15e-6, "cw": 3.75e-6, "cr": 0.30e-6},
    "claude-haiku-4-5-20251001": {"i": 1e-6, "o": 5e-6,  "cw": 1.25e-6, "cr": 0.10e-6},
}

EMPTY_TOKENS = {"input": 0, "output": 0, "cache_write": 0, "cache_read": 0}


def token_usage_for_session(search_root, sid):
    """Sum token I/O for one claude session — its main transcript AND its subagents (recall/learn
    dispatch Task subagents whose tokens are billed into the parent total) — deduped by message id,
    from the recorded JSONL. This is authoritative and CUMULATIVE across resume rounds (the result
    object's `usage` is only the last turn), and it's stored in the result JSON so cost provenance
    survives even if transcripts are later pruned (the 2026-06-02 run lost some to pool re-creation)."""
    if not sid or not search_root:
        return dict(EMPTY_TOKENS)
    paths = (_glob.glob(f"{search_root}/**/{sid}.jsonl", recursive=True)
             + _glob.glob(f"{search_root}/**/{sid}/subagents/*.jsonl", recursive=True))
    return _sum_usage(paths)


def _sum_usage(paths):
    """Sum input/output/cache token usage across jsonl transcripts, deduped by message id
    (duplicate-id messages carry identical usage). Shared by the sid- and workdir-keyed lookups."""
    tot = dict(EMPTY_TOKENS)
    seen = set()
    for path in set(paths):
        try:
            lines = open(path, errors="replace").read().splitlines()
        except Exception:
            continue
        for line in lines:
            try:
                entry = json.loads(line)
            except Exception:
                continue
            msg = entry.get("message") or {}
            usage = msg.get("usage")
            if not usage:
                continue
            mid = msg.get("id")
            if mid:
                if mid in seen:
                    continue
                seen.add(mid)
            tot["input"] += usage.get("input_tokens", 0) or 0
            tot["output"] += usage.get("output_tokens", 0) or 0
            tot["cache_write"] += usage.get("cache_creation_input_tokens", 0) or 0
            tot["cache_read"] += usage.get("cache_read_input_tokens", 0) or 0
    return tot


def tokens_for_workdir(cfgpool_root, workdir):
    """Sum a session's tokens by its WORKDIR's claude project dir, when the session id is unknown
    (a headless `/learn` whose top-level result JSON came back malformed — no sid/cost — even
    though the agent ran and wrote notes). Claude encodes the cwd as a project-dir name: `/private`
    prefix (macOS /tmp symlink) then every '/' and '.' → '-'. Sums across all cfgpool copies (the
    pool reuses cfgs) and subagents; dedup-by-id makes the union safe."""
    if not (cfgpool_root and workdir):
        return dict(EMPTY_TOKENS)
    slug = re.sub(r"[/.]", "-", "/private" + workdir if workdir.startswith("/tmp") else workdir)
    paths = (_glob.glob(f"{cfgpool_root}/*/projects/{slug}/*.jsonl")
             + _glob.glob(f"{cfgpool_root}/*/projects/{slug}/**/*.jsonl", recursive=True))
    return _sum_usage(paths)


def recompute_cost(tokens, model_id):
    """Reconstruct $ from token counts × the price sheet — the 1.00x cost audit. None if the model
    isn't in the sheet."""
    price = PRICES.get(model_id)
    if not price:
        return None
    return round(tokens["input"] * price["i"] + tokens["output"] * price["o"]
                 + tokens["cache_write"] * price["cw"] + tokens["cache_read"] * price["cr"], 4)


def token_audit(search_root, sid, model_id, reported_cost):
    """Bundle the captured tokens, recomputed cost, and reported/recomputed ratio (the audit)."""
    tokens = token_usage_for_session(search_root, sid)
    recomputed = recompute_cost(tokens, model_id)
    ratio = round(recomputed / reported_cost, 3) if (recomputed and reported_cost) else None
    return {"tokens": tokens, "recomputed_cost": recomputed, "cost_ratio": ratio}


# ----- build mode -----

def _seed_build_vault(workdir, vault_in):
    """Per-cell isolated copy of vault_in so in-loop recall synthesis writes land in a
    throwaway, never the shared snapshot other cells read. Returns (build_vault, vault_in)."""
    import shutil
    build_vault = workdir + ".buildvault"
    shutil.rmtree(build_vault, ignore_errors=True)
    if vault_in != "none" and os.path.isdir(vault_in):
        shutil.copytree(vault_in, build_vault)
    else:
        if vault_in != "none":
            print(f"WARN: vault_in {vault_in} missing — building COLD", file=sys.stderr)
            vault_in = "none"
        os.makedirs(os.path.join(build_vault, "Permanent"), exist_ok=True)
    return build_vault, vault_in


def _seed_build_chunks(workdir, chunks_in):
    """Per-cell isolated copy of the accumulated chunk index (auto regime's memory). Mirrors
    _seed_build_vault: the cell reads/writes a throwaway, never the shared snapshot."""
    import shutil
    build_chunks = workdir + ".buildchunks"
    shutil.rmtree(build_chunks, ignore_errors=True)
    if chunks_in and chunks_in != "none" and os.path.isdir(chunks_in):
        shutil.copytree(chunks_in, build_chunks)
    else:
        os.makedirs(build_chunks, exist_ok=True)
    return build_chunks


def _count_chunks(chunks_dir):
    """Total chunk records across the index .jsonl files."""
    total = 0
    try:
        for name in os.listdir(chunks_dir):
            if name.endswith(".jsonl"):
                with open(os.path.join(chunks_dir, name), errors="replace") as f:
                    total += sum(1 for line in f if line.strip())
    except OSError:
        pass
    return total


def _round_rec(rnd, sc, res, conv, feat):
    return {"round": rnd, "score": sc.get("total"), "feat_buckets": sc.get("feat_buckets"),
            "arch": sc.get("arch"), "convention_fails": len(conv), "feature_fails": len(feat),
            "cost": res.get("total_cost_usd", 0) or 0, "turns": res.get("num_turns", 0) or 0,
            "is_error": bool(res.get("is_error"))}


def _arch_detector_names():
    import archscore
    return [n for n, _ in archscore.DETECTORS]


def run_build(args):
    regime = REGIMES[args.regime]
    os.makedirs(args.workdir, exist_ok=True)
    t0 = time.time()
    build_vault, vault_in = _seed_build_vault(args.workdir, args.vault_in)
    build_chunks = os.path.join(args.workdir + ".buildchunks")
    os.makedirs(build_chunks, exist_ok=True)
    # START-of-op snapshot of the seeded vault — warm cells carry prior apps' notes forward, so
    # notes_written/learn_kind_breakdown must be reported as a delta against this, not the total.
    notes_baseline = snapshot_notes(build_vault)

    prompt = build_prompt(args.app, json.load(open(args.spec))["interface"],
                          regime["read_mode"])

    def do_build(msg, resume_sid=None):
        if args.stub:
            return _stub_build(args)
        res = claude(args.cfg, args.model, build_vault, args.workdir, msg, resume_sid=resume_sid)
        # Transient rate-limit/overload retry on BOTH the initial build and resumes (a $0-ish,
        # 1-turn error is the 429 signature). Sustained quota exhaustion outlasts these backoffs
        # and is handled downstream (a never-built round writes no success result; a rate-limited
        # resume is flagged), so a re-run when quota resets fills the gap cleanly.
        for backoff in (15, 45, 120):
            if not (res.get("is_error") and (res.get("total_cost_usd", 0) or 0) < 0.02):
                break
            refresh_creds_path(args.cfg)
            time.sleep(backoff)
            res = claude(args.cfg, args.model, build_vault, args.workdir, msg, resume_sid=resume_sid)
        return res

    t_recall_start = time.time()
    res = do_build(prompt)
    t_recall_end = time.time()
    sid = res.get("session_id")
    sc = scoremod.score(args.workdir, args.spec)
    conv, feat = split_failed(sc.get("failed", []))
    rounds = [_round_rec(1, sc, res, conv, feat)]

    # A build that never produced a working build at round 1 (the agent errored out — almost always
    # a sustained rate-limit / infra failure, not a real attempt) is NOT a result. Exit without
    # writing one, so a resume re-runs this cell when quota is back (don't poison the dataset with
    # a final:None cell that op_done() would treat as complete).
    if sc.get("build") != "ok" and bool(res.get("is_error")):
        print(f"build FAILED at round 1 ({args.app} {args.regime}) — likely rate_limit; "
              f"no result written so resume re-runs it.", file=sys.stderr)
        sys.exit(1)

    rate_limited = bool(res.get("is_error"))

    # Round-1 per-detector ARCH snapshot — the say-once discriminator (advisor flag 3).
    arch_fail1 = set(sc.get("arch_fail", []))
    round1_arch = {name: name not in arch_fail1 for name in _arch_detector_names()}
    round1_conv_fails = len(conv)
    round1_feat_fails = len(feat)

    stated = list(conv_labels(sc.get("failed", [])))  # cumulative, for the learn prompt
    spec = json.load(open(args.spec))

    # DRIVE TO COMPLETION. No stale-break / round-cap give-up — the chain's premise is that each
    # app is built to completion before its memory is learned and the next app recalls it (a
    # half-built prior teaches half a lesson). When an item stays stuck, ESCALATE the feedback's
    # granularity (symptom → literal code-level fix); a real reviewer does this, and it is fair to
    # tell the model exactly what to write. stated_counts drives the escalation; it does NOT touch
    # the say-once metric (round-1 based). args.max_rounds is now a high safety cap, not a target.
    from collections import defaultdict
    stated_counts = defaultdict(int)
    rnd = 1
    # stub builds are deterministic (re-copy the same fixture), so the feedback loop is a no-op —
    # one round suffices to validate wiring/threading/schema without burning the time budget.
    t_build_start = time.time()
    # Convergence-stall early-stop (retro Bug 4): the convergence score is arch progress + feature
    # progress = arch_pass minus the count of failing feature buckets. It rises monotonically as the
    # build improves (more arch detectors pass, fewer feature fails). If it does NOT improve for
    # STALL_PATIENCE consecutive rounds, HALT — escalated feedback is no longer making progress and
    # the tail rounds just burn spend (feeds-real.full sat at 9/18 rounds 3–8 for ~$0.78 of zero
    # progress). Escalation still runs every round up to the stall; this only caps wasted tail spend.
    stalled = False
    conv_score = convergence_score(sc)
    no_improve = 0
    while not args.stub and not converged(sc) and rnd < args.max_rounds and sc.get("build") == "ok":
        rnd += 1
        fb = feedback_prompt(sc["failed"], stated_counts, spec)
        for lbl, _ in sc["failed"]:
            stated_counts[lbl] += 1   # this convention/feature has now been stated once more
        res = do_build(fb, resume_sid=sid) if not args.stub else _stub_build(args)
        errored = bool(res.get("is_error"))
        sc = scoremod.score(args.workdir, args.spec)
        conv, feat = split_failed(sc.get("failed", []))
        rounds.append(_round_rec(rnd, sc, res, conv, feat))
        for lbl in conv_labels(sc.get("failed", [])):
            if lbl not in stated:
                stated.append(lbl)
        new_score = convergence_score(sc)
        if new_score > conv_score:
            conv_score = new_score
            no_improve = 0
        else:
            no_improve += 1
        if errored:
            rate_limited = True  # built at round 1 but a resume hit the limit — result kept, flagged
            break
        if no_improve >= STALL_PATIENCE and not converged(sc):
            stalled = True
            print(f"STALL: {args.app} {args.regime} convergence score flat for {STALL_PATIENCE} "
                  f"rounds (score={conv_score}) — halting at round {rnd} to save tail spend.",
                  file=sys.stderr)
            break
    t_build_end = time.time()

    completed = converged(sc)
    if not completed and not rate_limited and sc.get("build") == "ok":
        print(f"WARN: {args.app} {args.regime} did NOT converge in {rnd} rounds "
              f"(escalated feedback exhausted the safety cap) — flagged did_not_complete.", file=sys.stderr)

    # Recall-skill assertion (real.full / warm): the faithful signal is the Skill tool firing, not
    # a bare `engram query`. A warm cell where /recall did not fire never had the memory condition —
    # discard it (exit without a result so a resume re-runs it), BEFORE spending a learn on it.
    rf = 0 if (regime["read_mode"] == "none" or args.stub) else recall_fired(args.cfg, sid)
    if regime["read_mode"] == "skill" and not args.stub and rf == 0:
        print(f"recall SKILL did not fire ({args.app} {args.regime}) — invalid warm cell; "
              f"no result written so resume re-runs it.", file=sys.stderr)
        sys.exit(1)

    # One-session learn: real.full — the BUILD agent runs /learn on its OWN transcript at the end.
    # The /learn skill ingests the session as chunks AND crystallizes explicit fact/feedback notes.
    # resume_sid=sid keeps it the same session. cold skips this entirely.
    learn_meta = {"ran": False, "cost": 0.0, "turns": 0, "fired": None}
    t_learn_start = time.time()
    if not args.stub and regime["write"] == "skill" and sc.get("build") == "ok" and completed:
        lr = do_build(skill_learn_prompt(), resume_sid=sid)  # same session
        learn_meta["ran"] = True
        learn_meta["cost"] = round(lr.get("total_cost_usd", 0) or 0, 4)
        learn_meta["turns"] = lr.get("num_turns", 0) or 0
        learn_meta["fired"] = learn_fired(args.cfg, sid)
        subprocess.run(["engram", "embed", "apply", "--all"],
                       env={**os.environ, "ENGRAM_VAULT_PATH": build_vault,
                            "PATH": ENGRAM_BIN_DIR + ":" + os.environ.get("PATH", "")},
                       capture_output=True, text=True)
    t_learn_end = time.time()

    # Escalation depth — how granular the human feedback had to get before convergence (§5 signal).
    # stated_counts[label] = #rounds an item was fed back; feedback escalates to the literal
    # code-level fix once an item has been stated >=1 time already (so depth>=2 = needed
    # prescriptive hand-holding; depth 1 = fixed on the symptom alone; depth 0 = right in round 1).
    # Split convention vs feature, since "how hard to teach the conventions" is the load-bearing one.
    esc = dict(stated_counts)
    conv_esc = {k[len("ARCH:"):]: v for k, v in esc.items() if k.startswith("ARCH:")}
    feat_esc = {k: v for k, v in esc.items() if not k.startswith("ARCH:")}
    escalation = {
        "by_label": esc,                                        # full per-item statement counts
        "max_depth": max(esc.values(), default=0),              # deepest hand-holding any item needed
        "max_convention_depth": max(conv_esc.values(), default=0),
        "items_escalated": sum(1 for v in esc.values() if v >= 2),       # needed the prescriptive fix
        "conventions_escalated": sum(1 for v in conv_esc.values() if v >= 2),
        "features_escalated": sum(1 for v in feat_esc.values() if v >= 2),
    }

    recall_ok = regime["read_mode"] == "none" or bool(args.stub) or rf > 0
    followed = False if args.stub else link_followed(args.cfg, sid)

    build_cost = round(sum(r["cost"] for r in rounds), 4)
    audit = ({"tokens": dict(EMPTY_TOKENS), "recomputed_cost": 0.0, "cost_ratio": None} if args.stub
             else token_audit(args.cfg, sid, MODELS[args.model], build_cost))

    # Modern vault metrics: count notes written by /learn + lazy crystallizations fired by /recall.
    # No tier breakdown — notes are flat (fact | feedback).
    notes_written = count_notes_written(build_vault, notes_baseline) if not args.stub else 0
    learn_kind = (count_learn_kind_breakdown(build_vault, notes_baseline) if not args.stub
                  else {"fact": 0, "feedback": 0, "other": 0})
    crystallizations = (0 if args.stub or regime["read_mode"] == "none"
                        else count_crystallizations_at_recall(args.cfg, sid))
    chunks_ingested = _count_chunks(build_chunks) if not args.stub else 0

    out = {
        "schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "build",
        "app": args.app, "model": args.model, "model_id": MODELS[args.model],
        "regime": args.regime, "trial": args.trial, "date": args.date,
        "read_mode": regime["read_mode"],
        "vault_in": args.vault_in,
        "stub": args.stub or None,
        "converged": converged(sc), "rounds_to_converge": rnd if converged(sc) else None,
        "max_rounds": args.max_rounds,
        "round1_score": rounds[0]["score"], "round1_arch_detectors": round1_arch,
        "round1_convention_fails": round1_conv_fails, "round1_feature_fails": round1_feat_fails,
        "final_score": sc.get("total"), "final_buckets": sc.get("feat_buckets"),
        "final_arch": sc.get("arch"), "arch_pass": sc.get("arch_pass", 0),
        "stated_conventions": stated, "convention_statements": round1_conv_fails,
        "feature_statements": round1_feat_fails,
        "recall_fired": rf, "recall_ok": recall_ok, "link_followed": followed,
        "rate_limited": rate_limited, "completed": completed,
        "did_not_complete": (not completed and not rate_limited and sc.get("build") == "ok"),
        "hit_max_rounds": len(rounds) >= args.max_rounds, "total_rounds": len(rounds),
        "stalled": stalled,
        "escalation": escalation,
        "build_cost": build_cost,
        "build_turns": sum(r["turns"] for r in rounds),
        "tokens": audit["tokens"], "recomputed_cost": audit["recomputed_cost"],
        "cost_ratio": audit["cost_ratio"],
        "wall_min": round((time.time() - t0) / 60.0, 1),
        "recall_s": round(t_recall_end - t_recall_start, 3),
        "build_s": round(t_build_end - t_build_start, 3),
        "learn_s": round(t_learn_end - t_learn_start, 3),
        "axis_c1_recall_s": round(t_recall_end - t_recall_start, 3),
        "axis_c1_build_s": round(t_build_end - t_build_start, 3),
        "axis_c1_learn_s": round(t_learn_end - t_learn_start, 3),
        "axis_c2_cost_usd": round(sum(r["cost"] for r in rounds), 4),
        "axis_c3_interventions": round1_conv_fails,
        # Modern memory metrics (v3 — no tiers/episodes)
        "notes_written": notes_written,
        "learn_kind_breakdown": learn_kind,
        "crystallizations_at_recall": crystallizations,
        "chunks_ingested": chunks_ingested,
        "learn": learn_meta,
        "rounds": rounds, "session_id": sid, "workdir": args.workdir,
    }

    # Promote the accumulated build vault to vault_out for real.full so the next app recalls it.
    if regime["write"] in ("skill",) and getattr(args, "vault_out", ""):
        import shutil as _shutil
        _shutil.rmtree(args.vault_out, ignore_errors=True)
        _shutil.copytree(build_vault, args.vault_out)

    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({k: out[k] for k in ["app", "model", "regime", "converged",
          "rounds_to_converge", "round1_score", "convention_statements", "feature_statements",
          "recall_fired", "recall_ok", "notes_written", "crystallizations_at_recall",
          "chunks_ingested", "build_cost", "wall_min"]}, indent=2))


# ----- learn mode -----

def run_learn(args):
    """Legacy learn mode — only used for cold (write_mode=none). real.full learns in-session."""
    t0 = time.time()
    if REGIMES.get(args.regime, {}).get("write") in ("skill",):
        raise SystemExit("real.full learns in-session (run_build); run_learn is not used for it")
    # cold regime: write nothing
    out = {"schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "learn",
           "app": args.app, "model": args.model, "regime": args.regime, "trial": args.trial,
           "date": args.date, "write_mode": "none", "vault_in": args.vault_in,
           "vault_out": args.vault_out, "learned": False,
           "learn_cost": 0.0, "learn_turns": 0, "wall_min": 0.0}
    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({"app": args.app, "regime": args.regime, "write_mode": "none", "learned": False}))


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="mode", required=True)

    b = sub.add_parser("build")
    for a in ["app", "model", "regime", "vault-in", "cfg", "workdir", "spec", "out"]:
        b.add_argument("--" + a, required=True)
    # real.full learns in-session; --vault-out is where the accumulated vault is promoted for the next app.
    b.add_argument("--vault-out", default="")
    b.add_argument("--trial", type=int, default=1)
    b.add_argument("--date", default="")
    b.add_argument("--max-rounds", type=int, default=8)  # lowered from 15; escalation drives completion
    b.add_argument("--stub", default="", choices=["", "good", "naive"])

    le = sub.add_parser("learn")
    for a in ["app", "model", "regime", "write-mode", "workdir", "vault-in", "vault-out", "out"]:
        le.add_argument("--" + a, required=True)
    le.add_argument("--trial", type=int, default=1)
    le.add_argument("--date", default="")
    le.add_argument("--cfg", default="")
    le.add_argument("--build-result", default="")
    le.add_argument("--stub", default="", choices=["", "good", "naive"])

    args = ap.parse_args()
    # argparse stores hyphenated flags with underscores; normalize the few we read by attr.
    args.vault_in = getattr(args, "vault_in")
    if args.mode == "build":
        args.vault_out = getattr(args, "vault_out", "")
        run_build(args)
    else:
        args.vault_out = getattr(args, "vault_out")
        args.write_mode = getattr(args, "write_mode")
        run_build_result = getattr(args, "build_result", "")
        args.build_result = run_build_result
        run_learn(args)


if __name__ == "__main__":
    main()
