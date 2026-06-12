#!/usr/bin/env python3
"""One operation of the cumulative-accumulation eval (v2).

Two modes, so build and learn decouple (app1 is built cold ONCE, then fanned out
to 4 write-tier learns — §1.3):

  build  recall (read-subset) -> build -> score -> feed back ALL gaps -> resume ->
         loop to the bar.  Records per-round convention/feature intervention counts
         (split on the scorer's ARCH: prefix), the round-1 per-detector ARCH
         snapshot (the say-once discriminator), stated_conventions (for the learn
         prompt), rounds_to_converge, recall_fired (+ recall_ok flag), link-following,
         per-round cost/turns, wall_min.  NO learn.

  learn  over an already-built workdir, write at the regime's write-tier
         (cold=nothing; L1=episode; L2=episode+facts; L3=episode+facts+§6b synthesis),
         capturing the STATED conventions the build fed back (so "say it once"
         persists), per the tiered-capture-discipline ADR.  Verifies output populated.

Recall encoding (read-subset, §1.4):
  none           -> no recall (cold)
  blended        -> engram query (no --tier): full vault returned
  tier [T ...]   -> engram query --tier T [--tier T2 ...]; surfaced notes carry
                    outbound_links and the build is told to follow them with
                    `engram show <basename>` (direct-vs-follow-on-demand, not a blinding)

Usage:
  harness.py build --app feeds --model sonnet --regime l2.l2 --trial 1 \
      --vault-in <dir|none> --cfg <cfgdir> --workdir <dir> --spec <spec.json> \
      --out <build.json> [--max-rounds 6] [--stub good|naive]
  harness.py learn --app notes --model sonnet --regime l2.l2 --trial 1 \
      --write-tier L2 --workdir <built-dir> --vault-in <dir|none> --vault-out <dir> \
      --build-result <build.json> --cfg <cfgdir> --out <learn.json> [--stub good|naive]
"""
import argparse, glob as _glob, json, os, re, subprocess, sys, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import score as scoremod

# Single editable source of truth for the model registry — a new model is a one-line add (§1.5).
MODELS = {"haiku": "claude-haiku-4-5-20251001", "sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
ENGRAM_BIN_DIR = os.environ.get("ENGRAM_BIN_DIR", os.path.expanduser("~/go/bin"))
SCHEMA_VERSION = 2
CONVERGE_ARCH_BAR = 8  # arch_pass >= 8 (matches converged())

# The 7 regimes: write-tier (learn ceiling) x read-subset (recall surface). §1.2.
# read_mode: none | blended | tier ; read_tiers used only when read_mode == tier.
REGIMES = {
    "cold":      {"write": "none", "read_mode": "none",    "read_tiers": []},
    "l1":        {"write": "L1",   "read_mode": "tier",    "read_tiers": ["L1"]},
    "l2.l1l2":   {"write": "L2",   "read_mode": "blended", "read_tiers": []},
    "l2.l2":     {"write": "L2",   "read_mode": "tier",    "read_tiers": ["L2"]},
    # Lazy arm (B): write L1 at learn; crystallize L2s on demand at recall (synthesize_l2),
    # then persist them forward (see run_learn). Both arms end with L1+L2 readable — A writes
    # L2 eagerly at learn, B crystallizes lazily at recall.
    "l2.lazy":   {"write": "L1",   "read_mode": "synthesize_l2", "read_tiers": []},
    "l3.l1l2l3": {"write": "L3",   "read_mode": "blended", "read_tiers": []},
    "l3.l2l3":   {"write": "L3",   "read_mode": "tier",    "read_tiers": ["L2", "L3"]},
    "l3.l3":     {"write": "L3",   "read_mode": "tier",    "read_tiers": ["L3"]},
    # Real-skill rebuild (2026-06-11; eval-validity audit). The agent INVOKES the actual
    # /recall and /learn skills — no inlined proxy logic, no "exactly one episode" / tier
    # overrides. These are the only regimes that exercise the SHIPPED skills end to end.
    # read_mode "skill"  => prompt the agent to invoke /recall (assert the Skill tool fired).
    # write "skill"       => invoke /learn at its lazy default (episodes per-arc; defers L2 to recall).
    # write "skill-eager" => invoke /learn and explicitly request eager learn-time L2 (documented mode).
    "real.lazy":  {"write": "skill",       "read_mode": "skill", "read_tiers": []},
    "real.eager": {"write": "skill-eager", "read_mode": "skill", "read_tiers": []},
    # Auto-chunk experiment (2026-06-11): NO agent /learn at all. After the build session the
    # HARNESS runs `engram ingest` over the cell's own transcript (binary chunks+embeds, zero
    # LLM); recall is the /recall chunk-variant skill querying `engram query-chunks`. Memory
    # persists forward as the chunk index, not a vault.
    "real.auto":  {"write": "auto",        "read_mode": "skill-chunks", "read_tiers": []},
    # Chunks + vault-backed L2: same zero-LLM transcript ingest, but recall runs the UNIFIED
    # `engram query` (chunks + vault in one top-N ranking) and crystallizes lessons as REAL
    # vault notes via `engram learn fact|feedback` when near-match chunk groups bind a principle.
    # Both the chunk index AND the vault persist forward. Tests whether recall-time L2 stays
    # cheap and helps (or harms) vs pure chunks.
    "real.autol2": {"write": "auto-l2",    "read_mode": "skill-chunks", "read_tiers": []},
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


def build_prompt(app, interface, read_mode, read_tiers):
    """Build prompt with read-subset-appropriate recall. Tier-capped reads are told
    EXPLICITLY they can follow each surfaced note's outbound_links with `engram show
    <basename>` — this is what makes a tier-read regime a test of
    direct-provision-vs-follow-on-demand rather than a blinding (§1.4)."""
    phrases = (f'--phrase "building a command-line {app} in Go" '
               f'--phrase "architecture and conventions for a Go CLI tool" '
               f'--phrase "{app} storage, data handling, and features"')
    if read_mode == "none":
        recall = ""
    elif read_mode == "blended":
        recall = ("\nBefore writing any code, consult your memory — run exactly this, read every "
                  "surfaced note, and APPLY every convention and decision it surfaces:\n"
                  f"  engram query {phrases}\n")
    elif read_mode == "synthesize_l2":
        recall = (
            "\nBefore writing any code, consult your memory. Run exactly this, read every surfaced "
            "note, and APPLY every convention and decision it surfaces:\n"
            f"  engram query --synthesize-l2 {phrases}\n"
            "This is LAZY L2 synthesis. Each cluster in the payload may carry `nearest_l2: {path, cosine}` "
            "— the closest existing L2 to that cluster. For each cluster (any size — there is no minimum "
            "cluster size), apply the bands and WAIT for any writes before continuing (the new L2s are "
            "for THIS build):\n"
            "  - no `nearest_l2` field (no existing L2 to compare against) -> `engram learn fact|feedback` "
            "creating a new L2 synthesizing the cluster (--position top, --relation to each member; NO "
            "--tier). This is the bootstrap path that crystallizes the first L2s.\n"
            "  - cosine >= 0.95 -> do nothing (an L2 already covers it).\n"
            "  - 0.80 <= cosine < 0.95 -> `engram learn fact|feedback` updating the nearest L2 "
            "(--target <luhmann-id from nearest_l2.path> --position continuation; NO --tier).\n"
            "  - cosine < 0.80 -> `engram learn fact|feedback` creating a new L2 synthesizing the "
            "cluster (--position top, --relation to each member; NO --tier).\n"
            "Prefer the more-recently-created member where members diverge. Then APPLY the surfaced "
            "and freshly-written L2 conventions to your build.\n")
    elif read_mode == "skill":
        recall = (
            "\nBefore writing ANY code, consult your memory by INVOKING YOUR /recall SKILL — actually "
            "run the skill (it prints its Step 0 plan, queries the vault, and synthesizes impact). Do "
            "NOT hand-run `engram query` yourself in place of the skill. Frame the recall around "
            f"building a command-line {app} in Go and its architecture/conventions. Read every note the "
            "skill surfaces and APPLY every convention and decision — including any L2 the skill "
            "crystallizes on demand — as requirements for your build.\n")
    elif read_mode == "skill-chunks":
        recall = (
            "\nBefore writing ANY code, consult your memory by INVOKING YOUR /recall SKILL — actually "
            "run the skill (it prints its plan, queries the chunk index, and synthesizes impact). Do "
            "NOT hand-run engram commands yourself in place of the skill. Frame the recall around "
            f"building a command-line {app} in Go and its architecture/conventions. Read every chunk "
            "the skill surfaces and APPLY every convention and decision they reveal as requirements "
            "for your build.\n")
    else:  # tier
        tier_flags = " ".join(f"--tier {t}" for t in read_tiers)
        recall = (
            "\nBefore writing any code, consult your memory. Run exactly this, read every surfaced "
            "note, and APPLY every convention and decision it surfaces:\n"
            f"  engram query {tier_flags} {phrases}\n"
            "This recall is TIER-CAPPED: engram returns only the listed tier(s). Each surfaced item's "
            "YAML carries `outbound_links` — the basenames of related notes one hop away (often the "
            "lower-tier notes a distilled standard was built from). When a surfaced note cites a "
            "constituent whose content you need, FETCH IT on demand with:\n"
            "  engram show <basename>\n"
            "(`engram show` prints the note's full content plus its own outbound_links, so one fetch "
            "reveals the next hop.) Follow whatever links you need to apply the conventions fully — the "
            "tier cap limits only what engram volunteers, never what you can open.\n")
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
# memory system (learn AND recall) is exercised, and learn-quality — whether the agent captured
# what matters per tier — is itself a measured output (score_learn_capture). Each entry:
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


# Name-agnostic detection of whether a learn CAPTURED each convention: substring match on note
# content (lowercased). Scores learn quality — did the agent persist what we know matters per tier.
CONVENTION_KEYWORDS = {
    "di": ["inject", "dependenc", "interface"],
    "sentinel": ["sentinel", "errors.is", "%w", "errnotfound", "error var"],
    "atomic": ["atomic", "rename", "createtemp", "temp file"],
    "stdlib": ["standard library", "stdlib", "third-party", "no external", "no dependenc"],
    "tests_fake_parallel": ["parallel", "fake", "in-memory", "tempdir"],
    "json": ["json", "machine-readable"],
    "nocolor": ["no_color", "nocolor", "ansi", "color"],
    "wrapped_errors": ["%w", "wrap", "fmt.errorf"],
    "named_perms": ["permission", "filemode", "named constant", "perm "],
    "no_global_data": ["global", "package-level", "mutable"],
}


# CAVEAT (2026-06-11 eval-validity audit): this is a COARSE coverage proxy — substring keyword
# matching (CONVENTION_KEYWORDS), NOT the semantic/name-agnostic judgement the wording implies. It
# is NOT the lazy-vs-eager discriminator (that is build outcomes + whether L2 was actually created).
# For real.* regimes the learn prompt no longer feeds the agent the labels (closed-loop half fixed);
# a semantic rescore of this metric is a deferred follow-up (plan Task 7 Step 2). Do not over-trust it.
def score_learn_capture(vault, stated, write_tier):
    """Did the agent's learn capture the conventions we expect for this tier? Name-agnostic: for
    each STATED convention, check whether any vault note's content covers it. Also tracks whether
    an L1 EPISODE was extracted — an episode is the foundation of every tier (facts/ADRs link down
    to it), so a tiered learn that produced no episode is a real failure, not a nuance."""
    blobs = []
    episodes = 0
    for path in glob_notes(vault):
        try:
            blobs.append(open(path, errors="replace").read().lower())
        except Exception:
            pass
        if note_tier(path) == "L1":
            episodes += 1
    corpus = "\n".join(blobs)
    captured = [c for c in stated if any(kw in corpus for kw in CONVENTION_KEYWORDS.get(c, [c]))]
    return {
        "engaged": len(blobs) > 0,
        "write_tier": write_tier,
        "episodes": episodes,
        "episode_extracted": episodes >= 1,  # an L1 episode must ALWAYS be extracted
        "captured": captured,
        "missed": [c for c in stated if c not in captured],
        "stated_count": len(stated),
        "captured_count": len(captured),
        "coverage": round(len(captured) / len(stated), 3) if stated else 1.0,
    }


# Agent-driven learn prompt: the agent runs its /learn skill (testing the whole memory system).
LEARN_PROMPT_INTRO = (
    "Use your engram /learn skill to capture durable memory from the build in THIS directory into "
    "the engram vault (the one `engram learn` manages). Derive the lessons from the code here — skip "
    "`engram transcript --mark`. Frame every note so a future agent building a DIFFERENT Go CLI "
    "surfaces and applies it. Capture via the /learn skill / `engram learn` — do NOT hand-write .md "
    "files or a MEMORY.md index; this is the engram vault, not a personal-memory store.\n"
    "ALWAYS begin by writing exactly ONE L1 episode of this build — it is REQUIRED for every tier "
    "(the foundation that facts and ADRs link down to), even when the tier's emphasis is facts. "
    "Skipping the episode is a failure.")

LEARN_STATED = (
    "\nThe reviewer stated these architecture conventions during this build — your capture MUST "
    "cover each one so a later app's recall surfaces it:\n{stated}\n")

LEARN_TIER_GUIDE = {
    "L1": "Write exactly ONE episode of this build (recording what you built and the conventions you "
          "applied). Episode only — no fact notes, no L3 synthesis.",
    "L2": "Write ONE episode, then one FACT per architecture convention, each relation-linked to the "
          "episode. Do NOT run L3 synthesis.",
    "L3": "Write ONE episode, then FACTS (one per convention, relation-linked to the episode), then "
          "run the §6b L3 synthesis (`engram query --synthesis`) and write the resulting ADR(s) "
          "(tier L3) linked down to their L2 facts.",
}


def skill_learn_prompt(write_tier):
    """Real-skill learn for the one-session cell: the BUILD agent (which has the real build
    transcript) invokes its /learn skill. No 'exactly one episode' cap, no convention label-feed
    (the agent derives lessons from its own session — dropping the closed-loop-scorer confound)."""
    eager = (write_tier == "skill-eager")
    parts = [
        "Now capture durable memory from the work you just did, by INVOKING YOUR /learn skill. "
        "Actually run the skill (it scans this session's transcript and writes episodes per work-arc). "
        "Do NOT hand-run `engram learn` in place of the skill, and do NOT cap the episode count — let "
        "the skill write one episode per arc as it sees fit.",
        "HEADLESS first-run: if `engram transcript --mark` reports no progress marker, do NOT stop to "
        "ask — run `engram transcript --mark --from all` to scan from the start, then proceed. Write "
        "episode bodies from the REAL transcript via `--from-transcript-range <session-id>:<start>.."
        "<end>` (the session-id is your current session); do NOT hand-write them with --transcript-text.",
    ]
    if eager:
        parts.append(
            "EAGER L2: in addition to episodes, explicitly request the skill's eager learn-time L2 — "
            "distill the recurring conventions you applied into fact/feedback notes NOW, rather than "
            "deferring them to a future recall. (This is the skill's documented eager mode.)")
    else:
        parts.append(
            "Run /learn at its DEFAULT (lazy): episodes only; do NOT distill facts/feedback at learn "
            "time — those are crystallized later at recall. Just capture the episodes.")
    parts.append("Work autonomously; end with a one-line summary of what you wrote.")
    return "\n\n".join(parts)


def learn_prompt(write_tier, stated):
    if write_tier in ("skill", "skill-eager"):
        return skill_learn_prompt(write_tier)
    parts = [LEARN_PROMPT_INTRO]
    if stated:
        parts.append(LEARN_STATED.format(stated="\n".join(f"  - {s}" for s in stated)))
    parts.append(LEARN_TIER_GUIDE[write_tier])
    parts.append("Work autonomously; end with a one-line summary of how many notes of each tier you wrote.")
    return "\n\n".join(parts)


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


def count_notes_by_tier(vault):
    """Verify the learn actually populated each tier — a tested mechanism produces nothing until
    run on real data (note-18). Reads the frontmatter tier: of every note."""
    counts = {"L1": 0, "L2": 0, "L3": 0, "other": 0}
    for path in glob_notes(vault):
        try:
            head = open(path, errors="replace").read(600)
        except Exception:
            continue
        tier = "other"
        for line in head.splitlines():
            s = line.strip()
            if s.startswith("tier:"):
                tier = s.split(":", 1)[1].strip().strip('"').strip("'")
                break
        counts[tier if tier in counts else "other"] += 1
    return counts


TIER_CEILING = {"L1": 1, "L2": 2, "L3": 3}


def note_tier(path):
    """Read a note's frontmatter tier (L1/L2/L3), or None."""
    try:
        head = open(path, errors="replace").read(600)
    except Exception:
        return None
    for line in head.splitlines():
        stripped = line.strip()
        if stripped.startswith("tier:"):
            return stripped.split(":", 1)[1].strip().strip('"').strip("'")
    return None


def _links_to_l2(note_path, vault):
    """Whether a note's wikilinks ([[basename]], e.g. in its `Related to:` block) point at
    ANOTHER note whose tier is L2 — the composition signal for lazy L2 synthesis (a crystallized
    L2 that builds on an existing L2). Resolves each target basename to its note file under the
    build vault and reads that target's tier."""
    try:
        body = open(note_path, errors="replace").read()
    except Exception:
        return False
    by_base = {os.path.basename(p)[: -len(".md")]: p for p in glob_notes(vault)}
    self_base = os.path.basename(note_path)[: -len(".md")]
    for target in re.findall(r"\[\[([^\]]+)\]\]", body):
        target = target.strip()
        if target == self_base:
            continue
        tgt_path = by_base.get(target)
        if tgt_path and note_tier(tgt_path) == "L2":
            return True
    return False


def prune_to_ceiling(vault, write_tier):
    """Deterministically enforce the write-tier ceiling: drop any note whose frontmatter
    tier exceeds write_tier (and its .vec.json sidecar). Higher tiers link DOWN to lower
    ones (ADR->facts->episode), so removing higher tiers never dangles a lower-tier link.
    This guards against the /learn skill writing above the requested ceiling — e.g. emitting
    facts during an L1 episode-only capture — which would make v1[L1] == v1[L2] and confound
    the write-tier axis."""
    ceil = TIER_CEILING.get(write_tier, max(TIER_CEILING.values()))
    removed = 0
    for path in glob_notes(vault):
        tier = note_tier(path)
        if tier in TIER_CEILING and TIER_CEILING[tier] > ceil:
            os.remove(path)
            sidecar = path[: -len(".md")] + ".vec.json"
            if os.path.exists(sidecar):
                os.remove(sidecar)
            removed += 1
    return removed


def converged(sc):
    # feature-complete (all behavioral buckets pass) + strong arch (arch_pass >= 8)
    beh_fail = [f for f in sc.get("failed", []) if not f[0].startswith("ARCH:")]
    return len(beh_fail) == 0 and sc.get("arch_pass", 0) >= CONVERGE_ARCH_BAR


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
    builds and runs) and return a canned result. No LLM call. For the lazy arm, also fakes
    a synthesize-l2 recall that crystallizes L2(s) into the build vault — so persist-forward
    and the vault-metrics path are validated without an LLM."""
    import shutil
    fix = os.path.join(os.path.dirname(os.path.abspath(__file__)), "testdata", args.stub)
    for path in _glob.glob(os.path.join(fix, "*")):
        dst = os.path.join(args.workdir, os.path.basename(path))
        if os.path.isdir(path):
            shutil.copytree(path, dst, dirs_exist_ok=True)
        else:
            shutil.copy(path, dst)
    if args.regime == "l2.lazy":
        _stub_crystallize_l2(args)
    return {"is_error": False, "total_cost_usd": 0.0, "num_turns": 1,
            "session_id": "stub-build", "result": "stubbed build"}


def _stub_crystallize_l2(args):
    """Fake the l2.lazy recall's crystallize-on-demand: write two L2 facts into the build vault
    (one composing on the other via a relation) so persist-forward (run_learn) and the build
    vault metrics (l2_generated/l2_composed) are exercised with no LLM. The deterministic learn
    for l2.lazy writes L1 only; this is the recall-side write the real run gets from /recall."""
    build_vault = args.workdir + ".buildvault"
    if not os.path.isdir(build_vault):
        return
    date = args.date or "2026-06-06"
    base1 = eg_learn(build_vault, date, "fact", f"stub-lazy-{args.app}-di",
                     {"situation": f"building a command-line {args.app} in Go",
                      "subject": "dependencies", "predicate": "should be",
                      "object": "injected interfaces so the core runs against in-memory fakes"}, [])
    rel = [f"{base1}|builds on"] if base1 else []
    eg_learn(build_vault, date, "fact", f"stub-lazy-{args.app}-errors",
             {"situation": f"signaling errors in a command-line {args.app} in Go",
              "subject": "error conditions", "predicate": "should be",
              "object": "package-level sentinel vars wrapped with %w and matched via errors.Is"}, rel)


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
    """Total chunk records across the index .jsonl files (the auto arm's notes_by_tier analog)."""
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
    build_chunks = _seed_build_chunks(args.workdir, getattr(args, "chunks_in", "none"))

    prompt = build_prompt(args.app, json.load(open(args.spec))["interface"],
                          regime["read_mode"], regime["read_tiers"])

    def do_build(msg, resume_sid=None):
        if args.stub:
            return _stub_build(args)
        res = claude(args.cfg, args.model, build_vault, args.workdir, msg, resume_sid=resume_sid,
                     chunks=build_chunks if regime["read_mode"] == "skill-chunks" else None)
        # Transient rate-limit/overload retry on BOTH the initial build and resumes (a $0-ish,
        # 1-turn error is the 429 signature). Sustained quota exhaustion outlasts these backoffs
        # and is handled downstream (a never-built round writes no success result; a rate-limited
        # resume is flagged), so a re-run when quota resets fills the gap cleanly.
        for backoff in (15, 45, 120):
            if not (res.get("is_error") and (res.get("total_cost_usd", 0) or 0) < 0.02):
                break
            refresh_creds_path(args.cfg)
            time.sleep(backoff)
            res = claude(args.cfg, args.model, build_vault, args.workdir, msg, resume_sid=resume_sid,
                         chunks=build_chunks if regime["read_mode"] == "skill-chunks" else None)
        return res

    res = do_build(prompt)
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
        if errored:
            rate_limited = True  # built at round 1 but a resume hit the limit — result kept, flagged
            break

    completed = converged(sc)
    if not completed and not rate_limited and sc.get("build") == "ok":
        print(f"WARN: {args.app} {args.regime} did NOT converge in {rnd} rounds "
              f"(escalated feedback exhausted the safety cap) — flagged did_not_complete.", file=sys.stderr)

    # Recall-skill assertion (real.* / warm): the faithful signal is the Skill tool firing, not a
    # bare `engram query`. A warm cell where /recall did not fire never had the memory condition —
    # discard it (exit without a result so a resume re-runs it), BEFORE spending a learn on it.
    rf = 0 if (regime["read_mode"] == "none" or args.stub) else recall_fired(args.cfg, sid)
    if regime["read_mode"] in ("skill", "skill-chunks") and not args.stub and rf == 0:
        print(f"recall SKILL did not fire ({args.app} {args.regime}) — invalid warm cell; "
              f"no result written so resume re-runs it.", file=sys.stderr)
        sys.exit(1)

    # One-session learn: for real-skill regimes the BUILD agent runs /learn on its OWN transcript
    # (episodes are genuine chunks, not summaries). resume_sid=sid keeps it the same session. cold
    # and legacy regimes skip this (cold has no learn; legacy regimes learn via a separate run_learn).
    learn_meta = {"ran": False, "cost": 0.0, "turns": 0, "fired": None, "notes_by_tier": {}}
    if not args.stub and regime["write"] in ("skill", "skill-eager") and sc.get("build") == "ok" and completed:
        lr = do_build(skill_learn_prompt(regime["write"]), resume_sid=sid)  # same session
        learn_meta["ran"] = True
        learn_meta["cost"] = round(lr.get("total_cost_usd", 0) or 0, 4)
        learn_meta["turns"] = lr.get("num_turns", 0) or 0
        learn_meta["fired"] = learn_fired(args.cfg, sid)
        subprocess.run(["engram", "embed", "apply", "--all"],
                       env={**os.environ, "ENGRAM_VAULT_PATH": build_vault,
                            "PATH": ENGRAM_BIN_DIR + ":" + os.environ.get("PATH", "")},
                       capture_output=True, text=True)
        learn_meta["notes_by_tier"] = count_notes_by_tier(build_vault)
    elif not args.stub and regime["write"] in ("auto", "auto-l2") and sc.get("build") == "ok" and completed:
        # Auto-chunk write path: ZERO-LLM. The harness ingests the cell's own session transcript
        # into the chunk staging dir; the next app in the chain recalls from it. The agent never
        # runs /learn — that absence IS the experimental condition.
        learn_meta["ran"] = True
        learn_meta["cost"] = 0.0
        _tpath = _find_session_transcript(args.cfg, sid)
        if not _tpath:
            print(f"session transcript {sid}.jsonl not found under {args.cfg}/projects "
                  f"({args.app} {args.regime}) — no result written so resume re-runs it.", file=sys.stderr)
            sys.exit(1)
        ir = subprocess.run(["engram", "ingest", "--transcript", _tpath, "--chunks-dir", build_chunks],
                            env={**os.environ, "PATH": ENGRAM_BIN_DIR + ":" + os.environ.get("PATH", "")},
                            capture_output=True, text=True)
        learn_meta["ingest_ok"] = ir.returncode == 0
        learn_meta["ingest_out"] = (ir.stdout or ir.stderr or "").strip()[-300:]
        learn_meta["chunks_total"] = _count_chunks(build_chunks)
        if ir.returncode != 0:
            print(f"engram ingest FAILED ({args.app} {args.regime}): {ir.stderr.strip()[:400]} — "
                  f"no result written so resume re-runs it.", file=sys.stderr)
            sys.exit(1)
        if regime["write"] == "auto-l2":
            # Recall-time L2s were written into build_vault by `engram learn` during the build
            # session (auto_embed gives sidecars); count them so the L2 volume is a measured output.
            learn_meta["notes_by_tier"] = count_notes_by_tier(build_vault)

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
    followed = False if args.stub else (regime["read_mode"] == "tier" and link_followed(args.cfg, sid))

    build_cost = round(sum(r["cost"] for r in rounds), 4)
    audit = ({"tokens": dict(EMPTY_TOKENS), "recomputed_cost": 0.0, "cost_ratio": None} if args.stub
             else token_audit(args.cfg, sid, MODELS[args.model], build_cost))

    # Post-op vault metrics: for the lazy arm, recall crystallizes L2s INTO the build vault during
    # the build session. Count those newly-crystallized L2s (vs. the seeded vault_in note set) and
    # how many compose on an existing L2. Arm A reports 0 here (it does not write during build);
    # the aggregate sources arm A's L2 count from learn notes_by_tier instead (see Task 4.4 note).
    seeded = (set(os.path.basename(p) for p in glob_notes(args.vault_in))
              if args.vault_in != "none" and os.path.isdir(args.vault_in) else set())
    new_l2 = [p for p in glob_notes(build_vault)
              if note_tier(p) == "L2" and os.path.basename(p) not in seeded]
    l2_composed = sum(1 for p in new_l2 if _links_to_l2(p, build_vault))

    out = {
        "schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "build",
        "app": args.app, "model": args.model, "model_id": MODELS[args.model],
        "regime": args.regime, "trial": args.trial, "date": args.date,
        "read_mode": regime["read_mode"], "read_tiers": regime["read_tiers"],
        "vault_in": args.vault_in, "chunks_in": getattr(args, "chunks_in", "none"),
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
        "escalation": escalation,
        "build_cost": build_cost,
        "build_turns": sum(r["turns"] for r in rounds),
        "tokens": audit["tokens"], "recomputed_cost": audit["recomputed_cost"],
        "cost_ratio": audit["cost_ratio"],
        "wall_min": round((time.time() - t0) / 60.0, 1),
        "l2_generated": len(new_l2), "l2_composed": l2_composed,
        "vault_notes_total": len(glob_notes(build_vault)),
        "learn": learn_meta,
        "rounds": rounds, "session_id": sid, "workdir": args.workdir,
    }

    # Promote the accumulated build vault to vault_out for real regimes so the next app recalls it
    # (build vault = seeded vault_in + recall-crystallized L2 + /learn episodes). This IS the
    # persist-forward for the one-session model — no separate learn-stage vault.
    if regime["write"] in ("skill", "skill-eager", "auto-l2") and getattr(args, "vault_out", ""):
        import shutil as _shutil
        _shutil.rmtree(args.vault_out, ignore_errors=True)
        _shutil.copytree(build_vault, args.vault_out)

    # Auto-chunk persist-forward: the chunk index (seeded chunks_in + this cell's ingested
    # transcript) is the accumulated memory the next app recalls.
    if regime["write"] in ("auto", "auto-l2") and getattr(args, "chunks_out", ""):
        import shutil as _shutil
        _shutil.rmtree(args.chunks_out, ignore_errors=True)
        if os.path.isdir(build_chunks):
            _shutil.copytree(build_chunks, args.chunks_out)
        else:
            os.makedirs(args.chunks_out, exist_ok=True)

    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({k: out[k] for k in ["app", "model", "regime", "converged",
          "rounds_to_converge", "round1_score", "convention_statements", "feature_statements",
          "recall_fired", "recall_ok", "link_followed", "build_cost", "wall_min"]}, indent=2))


# ----- learn mode -----

def run_learn(args):
    import shutil
    t0 = time.time()
    if REGIMES[args.regime]["write"] in ("skill", "skill-eager"):
        raise SystemExit("real.* regimes learn in-session (run_build); run_learn is not used for them")
    if args.write_tier == "none":
        out = {"schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "learn",
               "app": args.app, "model": args.model, "regime": args.regime, "trial": args.trial,
               "date": args.date, "write_tier": "none", "vault_in": args.vault_in,
               "vault_out": args.vault_out, "learned": False, "notes_by_tier": {},
               "learn_cost": 0.0, "learn_turns": 0, "wall_min": 0.0}
        json.dump(out, open(args.out, "w"), indent=2)
        print(json.dumps({"app": args.app, "regime": args.regime, "write_tier": "none", "learned": False}))
        return

    # Learn into a fresh copy of vault_in (accumulate on prior memory), then promote to vault_out.
    # Stage off vault_out (unique per op), NOT workdir: app1's 4 write-tier learns share one
    # build workdir and run in parallel, so a workdir-derived stage path would race and
    # cross-contaminate the seed vaults.
    learn_vault = args.vault_out + ".staging"
    shutil.rmtree(learn_vault, ignore_errors=True)

    # Lazy arm (l2.lazy): recall crystallized L2s INTO the build vault during
    # the build session. Seed the learn stage from THAT (it holds vault_in +
    # the new L2s) so crystallized L2s persist forward across the app chain.
    # A missing build vault here is a hard failure — NEVER fall back to
    # vault_in silently (that would measure the strawman, invisibly).
    build_vault = args.workdir + ".buildvault"
    if args.regime == "l2.lazy":
        if not os.path.isdir(build_vault):
            raise RuntimeError(
                f"l2.lazy: build vault {build_vault} missing — cannot persist "
                f"crystallized L2s forward; refusing to silently seed from vault_in")
        seed_src = build_vault
    else:
        seed_src = args.vault_in

    if seed_src != "none" and os.path.isdir(seed_src):
        shutil.copytree(seed_src, learn_vault)
    else:
        os.makedirs(os.path.join(learn_vault, "Permanent"), exist_ok=True)

    stated = []
    if args.build_result and os.path.exists(args.build_result):
        try:
            stated = json.load(open(args.build_result)).get("stated_conventions", [])
        except Exception:
            stated = []

    date = args.date or "2026-06-06"
    learn_sid = None
    if args.stub:
        # --stub: deterministic writer for zero-cost pipeline validation (NOT the real learn).
        learn_cost, learn_turns = 0.0, 0
        _deterministic_learn(learn_vault, args.app, args.regime, args.write_tier, stated, date)
    else:
        # Real learn: the AGENT runs its /learn skill, exercising the whole memory system. One
        # retry if it wrote nothing to the vault (a fair shot under note-14 skill-self-fire).
        lr = claude(args.cfg, args.model, learn_vault, args.workdir, learn_prompt(args.write_tier, stated))
        if len(glob_notes(learn_vault)) == 0:
            time.sleep(5)
            lr = claude(args.cfg, args.model, learn_vault, args.workdir, learn_prompt(args.write_tier, stated))
        learn_cost = round(lr.get("total_cost_usd", 0) or 0, 4)
        learn_turns = lr.get("num_turns", 0) or 0
        learn_sid = lr.get("session_id")

    # Enforce the write-tier ceiling (the experimental condition), embed, then SCORE capture
    # quality — did the agent persist the conventions we expect for this tier? A poor or empty
    # capture is RECORDED (a measured output), not engineered away.
    # For l2.lazy the crystallized L2s are the experiment's OUTPUT and must survive the
    # ceiling prune (write_tier is L1 for this regime, which would otherwise drop them).
    prune_tier = "L2" if args.regime == "l2.lazy" else args.write_tier
    pruned = prune_to_ceiling(learn_vault, prune_tier)
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = learn_vault
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    subprocess.run(["engram", "embed", "apply", "--all"], env=env, capture_output=True, text=True)
    quality = score_learn_capture(learn_vault, stated, args.write_tier)

    shutil.rmtree(args.vault_out, ignore_errors=True)
    shutil.copytree(learn_vault, args.vault_out)
    by_tier = count_notes_by_tier(args.vault_out)

    # Token capture. Prefer the sid-keyed lookup; but a headless /learn sometimes returns a
    # malformed top-level result (no sid, no total_cost_usd) even though the agent ran and wrote
    # notes — fall back to the WORKDIR's project dir so we never lose a real learn's tokens.
    if args.stub:
        audit = {"tokens": dict(EMPTY_TOKENS), "recomputed_cost": 0.0, "cost_ratio": None}
    else:
        audit = token_audit(args.cfg, learn_sid, MODELS[args.model], learn_cost)
        if sum(audit["tokens"].values()) == 0 and quality["engaged"]:
            tok = tokens_for_workdir(os.path.dirname(args.cfg), args.workdir)
            rec = recompute_cost(tok, MODELS[args.model])
            audit = {"tokens": tok, "recomputed_cost": rec,
                     "cost_ratio": (round(rec / learn_cost, 3) if (rec and learn_cost) else None)}
            if not learn_cost and rec:
                learn_cost = rec   # the result JSON dropped the cost too — use the recomputed one

    out = {
        "schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "learn",
        "app": args.app, "model": args.model, "model_id": MODELS[args.model],
        "regime": args.regime, "trial": args.trial, "date": args.date,
        "write_tier": args.write_tier, "vault_in": args.vault_in, "vault_out": args.vault_out,
        "stub": args.stub or None, "stated_conventions_input": stated, "workdir": args.workdir,
        "learned": quality["engaged"], "learn_quality": quality, "pruned_above_ceiling": pruned,
        "notes_total": len(glob_notes(args.vault_out)), "notes_by_tier": by_tier,
        "learn_cost": learn_cost, "learn_turns": learn_turns,
        "tokens": audit["tokens"], "recomputed_cost": audit["recomputed_cost"],
        "cost_ratio": audit["cost_ratio"],
        "wall_min": round((time.time() - t0) / 60.0, 1),
    }
    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({"app": args.app, "regime": args.regime, "write_tier": args.write_tier,
          "notes_by_tier": by_tier, "learn_coverage": quality["coverage"],
          "captured": f"{quality['captured_count']}/{quality['stated_count']}", "learn_cost": learn_cost}))


def _deterministic_learn(learn_vault, app, regime, write_tier, stated, date):
    """The --stub learn: write tier-correct notes deterministically via `engram learn` (no LLM)
    for zero-cost pipeline validation. The REAL learn is agent-driven (learn_prompt)."""
    conv_list = ", ".join(stated) if stated else "the reviewed architecture conventions"
    episode = eg_learn(learn_vault, date, "episode", f"eval-{app}-{regime}", {
        "situation": f"building a command-line {app} in Go",
        "summary": f"Built a command-line {app} in Go, applying: {conv_list}.",
        "boundary-rationale": "single eval build arc",
        "session": "eval-harness", "transcript-range": f"{date}T00:00:00Z..{date}T00:01:00Z",
        "transcript-text": f"Eval build of {app}. Conventions: {conv_list}.",
    }, [])
    fact_bases = []
    if write_tier in ("L2", "L3"):
        ep_rel = [f"{episode}|extracted from this build"] if episode else []
        for conv in stated:
            tmpl = CONVENTION_FACTS.get(conv)
            if tmpl is None:
                continue
            sit, subj, pred, obj = tmpl
            fact = eg_learn(learn_vault, date, "fact", f"eval-{app}-{regime}-{conv}",
                            {"situation": sit, "subject": subj, "predicate": pred, "object": obj}, ep_rel)
            if fact:
                fact_bases.append(fact)
    if write_tier == "L3":
        adr_rel = [f"{fb}|synthesized into this standard" for fb in fact_bases] or \
                  ([f"{episode}|distilled from this build"] if episode else [])
        eg_learn(learn_vault, date, "fact", f"eval-{app}-{regime}-adr", {
            "tier": "L3", "situation": f"designing the architecture of a Go CLI such as {app}",
            "subject": "Go CLI architecture", "predicate": "means",
            "object": "DI + atomic storage + sentinel errors + fake-driven parallel tests + output "
                      "discipline + no global state — the transferable conventions for a Go CLI",
        }, adr_rel)


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="mode", required=True)

    b = sub.add_parser("build")
    for a in ["app", "model", "regime", "vault-in", "cfg", "workdir", "spec", "out"]:
        b.add_argument("--" + a, required=True)
    # real.* regimes learn in-session; --vault-out is where the build agent's accumulated vault
    # (seeded vault_in + recall-crystallized L2 + /learn episodes) is promoted for the next app.
    b.add_argument("--vault-out", default="")
    b.add_argument("--chunks-in", default="none")
    b.add_argument("--chunks-out", default="")
    b.add_argument("--trial", type=int, default=1)
    b.add_argument("--date", default="")
    b.add_argument("--max-rounds", type=int, default=15)  # high safety cap; escalation drives completion
    b.add_argument("--stub", default="", choices=["", "good", "naive"])

    le = sub.add_parser("learn")
    for a in ["app", "model", "regime", "write-tier", "workdir", "vault-in", "vault-out", "out"]:
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
        args.write_tier = getattr(args, "write_tier")
        run_build_result = getattr(args, "build_result", "")
        args.build_result = run_build_result
        run_learn(args)


if __name__ == "__main__":
    main()
