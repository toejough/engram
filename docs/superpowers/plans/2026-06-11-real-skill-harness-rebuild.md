# Real-Skill Harness Rebuild — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the cumulative eval so each warm cell runs the *real* `/recall` and `/learn` skills end-to-end in one session — no inlined proxy recall logic, no "exactly one episode" / tier-cap overrides — then re-run the lazy-vs-eager question on evidence that transfers to prod.

**Architecture:** A warm cell becomes a single headless `claude` session: `/recall` (real skill) → build (escalating-feedback rounds via `resume_sid`) → `/learn` (real skill) — the prod flow. `/learn` therefore captures its own real build transcript (episodes are genuine chunks, not summaries). Regimes collapse to `cold` / `real.lazy` / `real.eager`; the old tier ladder is dropped because the real `/recall` has one read pipeline and tier-capping only ever existed as the proxy override being removed. A cell is discarded unless the transcript proves the Skill tool fired.

**Tech Stack:** Python eval harness (`dev/eval/cumulative/{harness.py,matrix.py}`), headless `claude -p` with `CLAUDE_CONFIG_DIR` cfg pool (skills copied in), engram CLI, the migrated dual-vector prod-vault-derived seed vaults.

**Validation philosophy:** The harness is orchestration tooling, not product code — it has no unit-test suite. The two validation levers are (a) the existing `--stub` path (zero-LLM wiring/threading/schema check) and (b) **one real warm cell end-to-end** before the full matrix. Both are explicit gates below. Do NOT launch the full 3-model n=5 run until the one-cell gate passes.

**Already done (uncommitted, fold into Task 1's commit):**
- `REGIMES` gains `real.lazy` (`write: skill`, `read_mode: skill`) and `real.eager` (`write: skill-eager`, `read_mode: skill`).
- `build_prompt` gains a `read_mode == "skill"` branch instructing the agent to INVOKE `/recall` (not hand-run `engram query`).
- Recall mechanism PROVEN: a headless `claude -p` told "use your /recall skill" fires the Skill tool (`"skill":"recall"`), loads `SKILL.md`, prints Step 0, runs `--synthesize-l2`, synthesizes (verified 2026-06-11, sid `5018b5dd`).

---

## File Structure

- `dev/eval/cumulative/harness.py` — the cell engine. Changes: `learn_prompt` skill modes; merge `/learn` into the build session (`run_build`); skill-fired assertions (`recall_fired` sharpen + new `learn_fired`); `vault_out` = build vault for real regimes.
- `dev/eval/cumulative/matrix.py` — the DAG/orchestrator. Changes: a `real.*` warm cell is ONE op (build+learn), not two; `--regimes cold,real.lazy,real.eager` selection; `op_done`/cells_for adjust.
- `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md` — context only (no edit).

---

## Task 1: Lock in the regime + recall-branch edits (already made)

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (REGIMES ~line 59; `build_prompt` `read_mode=="skill"` branch ~line 148)

- [ ] **Step 1: Verify the edits are present and syntactically valid**

Run: `cd /Users/joe/repos/personal/engram && python3 -c "import ast; ast.parse(open('dev/eval/cumulative/harness.py').read()); print('ok')"`
Expected: `ok`

- [ ] **Step 2: Confirm regimes load**

Run: `cd dev/eval/cumulative && python3 -c "import harness; print(sorted(harness.REGIMES))" ` (expect `real.eager`, `real.lazy` present alongside legacy keys)
Expected: list includes `real.eager`, `real.lazy`.

- [ ] **Step 3: Commit (after Task 2-6 land too — see note)**

This task's edits commit together with Tasks 2–6 as one coherent "real-skill cell" change, since they are interdependent (a regime with no learn-prompt mode or no cell wiring is half-built). Do NOT commit Task 1 alone.

---

## Task 2: `learn_prompt` skill modes — invoke the real `/learn`, drop the overrides

**Why:** The current `learn_prompt` forces "exactly ONE episode" (`LEARN_PROMPT_INTRO`) + feeds the agent the exact convention labels (`LEARN_STATED` — the closed-loop scorer confound) + a tier guide. For `real.*` regimes we want the *real* skill: episodes per-arc (the one-session model gives it a real transcript), and no label spoon-feeding.

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (`learn_prompt` ~line 355; add `skill_learn_prompt`)

- [ ] **Step 1: Add `skill_learn_prompt`**

```python
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
```

- [ ] **Step 2: Route skill tiers through it in `learn_prompt`**

At the top of `learn_prompt(write_tier, stated)`:

```python
def learn_prompt(write_tier, stated):
    if write_tier in ("skill", "skill-eager"):
        return skill_learn_prompt(write_tier)
    # ---- legacy tier-guide path (cold/l1/l2/l3 proxy regimes) unchanged below ----
    parts = [LEARN_PROMPT_INTRO]
    ...
```

- [ ] **Step 3: Verify syntactically + the two modes differ**

Run: `cd dev/eval/cumulative && python3 -c "import harness; a=harness.learn_prompt('skill',[]); b=harness.learn_prompt('skill-eager',[]); print('lazy has no-facts:', 'do NOT distill' in a); print('eager has facts:', 'EAGER L2' in b); print('no episode cap:', 'exactly ONE' not in a and 'exactly ONE' not in b)"`
Expected: all three `True`.

---

## Task 3: Merge `/learn` into the build session (the one-session cell)

**Why:** Episodes must be real transcript chunks → the agent that built must run `/learn` in the SAME session. `run_build` already threads the session via `resume_sid` for feedback rounds; add a terminal learn step for `real.*` regimes.

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (`run_build`, after convergence ~line 790; before assembling `out`)

- [ ] **Step 1: After the build loop, run `/learn` in-session for real regimes**

Insert after `completed = converged(sc)` block (~line 768), before the escalation/metrics assembly:

```python
    # One-session learn: for real-skill regimes the BUILD agent runs /learn on its own transcript
    # (episodes are genuine chunks). resume_sid = sid keeps it the same session. cold/legacy regimes
    # skip this (their learn is a separate op via run_learn).
    learn_meta = {"ran": False, "cost": 0.0, "turns": 0, "fired": None, "notes_by_tier": {}}
    if not args.stub and regime["write"] in ("skill", "skill-eager") and sc.get("build") == "ok" and completed:
        lr = do_build(skill_learn_prompt(regime["write"]), resume_sid=sid)  # same session
        learn_meta["ran"] = True
        learn_meta["cost"] = round(lr.get("total_cost_usd", 0) or 0, 4)
        learn_meta["turns"] = lr.get("num_turns", 0) or 0
        learn_meta["fired"] = learn_fired(args.cfg, sid)            # Task 4
        subprocess.run(["engram", "embed", "apply", "--all"],       # embed the new episodes/L2
                       env={**os.environ, "ENGRAM_VAULT_PATH": build_vault,
                            "PATH": ENGRAM_BIN_DIR + ":" + os.environ.get("PATH","")},
                       capture_output=True, text=True)
        learn_meta["notes_by_tier"] = notes_by_tier(build_vault)    # existing helper or inline count
```

- [ ] **Step 2: Promote the build vault to `vault_out` for real regimes**

The build vault already holds vault_in + recall-crystallized L2 (lazy) + now the learn episodes. For real regimes, `vault_out` IS the build vault (no separate staging). Add near the end of `run_build`:

```python
    if regime["write"] in ("skill", "skill-eager") and args.vault_out:
        import shutil
        shutil.rmtree(args.vault_out, ignore_errors=True)
        shutil.copytree(build_vault, args.vault_out)
```

- [ ] **Step 3: Record learn_meta in the build `out` dict**

Add to the `out` dict: `"learn": learn_meta,`. (Keeps build+learn metrics in one result for real cells.)

- [ ] **Step 4: Syntactic check**

Run: `cd dev/eval/cumulative && python3 -c "import ast; ast.parse(open('harness.py').read()); print('ok')"`
Expected: `ok`

---

## Task 4: Skill-fired assertions (recall AND learn)

**Why:** `recall_fired` greps for `engram query` — necessary but NOT sufficient (the proxy ran queries too). The faithful signal is the Skill tool: `"skill":"recall"` / `"skill":"learn"` in the transcript. Discard cells where the skill didn't fire.

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (`recall_fired` ~line 477; add `learn_fired`; tighten `recall_ok` ~line 788)

- [ ] **Step 1: Sharpen `recall_fired` to require the Skill tool**

```python
def recall_fired(cfg, sid):
    """Count transcript turns that INVOKED the /recall skill (the faithful signal — the Skill tool
    fired and SKILL.md loaded). Grepping `engram query` alone is insufficient: the old proxy ran
    queries without the skill. Require BOTH the skill invocation and a query."""
    return _skill_and_query_hits(cfg, sid, skill="recall", need_query=True)

def learn_fired(cfg, sid):
    """Whether the /learn skill was invoked in this session (Skill tool fired with skill=learn)."""
    return _skill_and_query_hits(cfg, sid, skill="learn", need_query=False) > 0
```

- [ ] **Step 2: Add the shared transcript scanner**

```python
def _skill_and_query_hits(cfg, sid, skill, need_query):
    hits = 0
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        for fn in files:
            if sid and sid in fn:
                try:
                    txt = open(os.path.join(root, fn)).read()
                except Exception:
                    continue
                fired = f'"skill":"{skill}"' in txt
                queried = ("engram query" in txt) if (skill == "recall" and need_query) else True
                if fired and queried:
                    hits += 1
    return hits
```

- [ ] **Step 3: Discard cells where recall didn't fire**

In `run_build`, after computing `rf`/`recall_ok` (~line 787): for `real.*` regimes, a cell with `recall_ok == False` is invalid (the warm condition never happened). Mark it and `sys.exit(1)` WITHOUT writing a result (mirrors the rate-limit no-result pattern at line 721) so a resume re-runs it:

```python
    if regime["read_mode"] == "skill" and not args.stub and rf == 0:
        print(f"recall SKILL did not fire ({args.app} {args.regime}) — invalid warm cell; "
              f"no result written so resume re-runs it.", file=sys.stderr)
        sys.exit(1)
```

- [ ] **Step 4: Verify with the mechanism-test transcript (already on disk)**

Run: `cd dev/eval/cumulative && python3 -c "import harness; print('recall fired:', harness.recall_fired('/tmp/mechtest-cfg','5018b5dd-53bd-4db8-b306-c53b49e26f87'))"`
Expected: `recall fired: 1` (≥1). (If `/tmp/mechtest-cfg` was cleaned, re-run a one-cell test first.)

---

## Task 5: Persist-forward reconciliation

**Why:** The old `run_learn` staged a `learn_vault` and, for `l2.lazy`, seeded it from the build vault to persist crystallized L2 forward. In the one-session model the build vault already accumulates everything (seeded vault_in + recall-crystallized L2 + learn episodes), so `vault_out = build_vault` (Task 3 Step 2) IS the persist-forward. No separate staging for real regimes.

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (`run_learn` — guard so it is a no-op / not invoked for real regimes)

- [ ] **Step 1: Make `run_learn` reject real regimes (they learn in-session)**

At the top of `run_learn`:

```python
    if REGIMES[args.regime]["write"] in ("skill", "skill-eager"):
        raise SystemExit("real.* regimes learn in-session (run_build); run_learn is not used for them")
```

- [ ] **Step 2: Confirm the chain seed wiring** — verify (read-only) that matrix passes the prior app's `vault_out` (= build vault) as the next app's `--vault-in`. Document the exact matrix line in Task 6.

---

## Task 6: Matrix DAG — one warm cell per (model, regime, app, trial)

**Why:** Build and learn were two ops; now a real warm cell is ONE op (build does it all). cold is unchanged (no learn). The matrix must (a) not emit a separate learn op for real regimes, (b) thread vault_in←prior vault_out, (c) select `cold,real.lazy,real.eager` via `--regimes`.

**Files:**
- Modify: `dev/eval/cumulative/matrix.py` (`cells_for` / op generation; `op_done`; `--regimes` default for this run)

- [ ] **Step 1: Read `cells_for` and the op DAG** — map exactly where build and learn ops are emitted per regime (the audit located `matrix.py` orchestration ~line 220). Identify the learn-op emission and gate it on `write not in ("skill","skill-eager")`.

- [ ] **Step 2: Gate out the separate learn op for real regimes**

In the op generator, where a `learn` op is appended for a regime, wrap:

```python
if REGIMES[regime]["write"] not in ("skill", "skill-eager"):
    ops.append(learn_op(...))   # legacy regimes only; real regimes learn in-session
```

- [ ] **Step 3: Thread vault_in ← prior app's vault_out** — confirm the existing chain wiring already passes `vault_out` forward (real regimes write `vault_out` in run_build Task 3 Step 2), so app2 recalls app1's accumulated vault. Adjust only if the learn-op removal broke the vault_out producer for the chain.

- [ ] **Step 4: Verify op generation under `--regimes cold,real.lazy,real.eager`**

Run (dry, no LLM): `cd dev/eval/cumulative && python3 matrix.py --regimes cold,real.lazy,real.eager --models opus --trials 1 --dry-run 2>&1 | head -40` (use the existing dry-run/plan flag; if none, add a `--plan` that prints ops without running).
Expected: per app, ONE build op for each real regime (no separate learn op); cold has build only.

---

## Task 7: Scorer de-confound (scope decision)

**Decision:** The closed-loop `score_learn_capture` confound has TWO halves. (a) **Label-feed** — fixed in Task 2 (skill learn prompt does not pass `stated` labels). (b) **Keyword-grep** (`CONVENTION_KEYWORDS` substring match) — KEEP for now but DEMOTE: it is not the lazy-vs-eager discriminator. The discriminators are the build outcomes (`archscore` say-once, convergence, completion) + whether L2 actually got created (`l2_generated` lazy vs `notes_by_tier` eager). Flag the keyword scorer's limitation in the results doc; do not block this rebuild on rewriting it.

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (comment on `score_learn_capture` ~line 300 noting it is a coarse coverage proxy, not the primary signal)

- [ ] **Step 1: Add the caveat comment** (one line) so a future reader does not over-trust the metric.
- [ ] **Step 2: (Deferred follow-up — file a GitHub issue)** rewrite `score_learn_capture` to judge capture semantically (name-agnostic), not by keyword grep. Out of scope for this rebuild.

---

## Task 8: GATES — validate one cell, then the full run

**Files:** none (operational)

- [ ] **Step 1: `--stub` wiring check** — run one stubbed warm cell to confirm threading/schema/DAG with zero LLM cost.

Run: `cd dev/eval/cumulative && python3 matrix.py --regimes cold,real.lazy,real.eager --models opus --trials 1 --stub 2>&1 | tail -20`
Expected: cells complete, results written, no exceptions, vault_out produced for real regimes.

- [ ] **Step 2: ONE real warm cell end-to-end (the gate)** — opus, `real.lazy`, app1 (notes), one trial, REAL skills. Then inspect the build vault:
  - the `/recall` skill fired (`recall_fired` ≥ 1) and `/learn` fired (`learn_fired` True);
  - episodes are REAL transcript chunks (`## Transcript` sections with actual session content, not a one-line summary);
  - the lazy-vs-eager learn difference manifests: run the SAME cell as `real.eager` and confirm `real.eager` wrote fact/feedback (L2) at learn while `real.lazy` wrote episodes only.

Run: the single-cell harness invocation (`harness.py build --app notes --model opus --regime real.lazy --trial 1 ...`) then `engram show` / `find` on the build vault. **Do NOT proceed if episodes are summaries or the skills didn't fire.**

- [ ] **Step 3: Auth + cost pre-flight for the full run** — confirm stable credentials (keychain creds were valid 2026-06-11; the long-lived token may be revoked by then) and surface the estimated cost/runtime of 3-model × {cold,real.lazy,real.eager} × 3-app × n=5 to the user before launching (a recall cell alone was ~$1.36 on sonnet; the full matrix is materially larger than the prior $150–200 estimate because build+learn now run real skills in-session).

- [ ] **Step 4: Launch the full matrix** (only after Steps 1–3 pass):

Run: `cd dev/eval/cumulative && python3 matrix.py --regimes cold,real.lazy,real.eager --models haiku,sonnet,opus --trials 1,2,3,4,5 --out runs/2026-06-11-real-skill-rebuild ...` (match the prior run's archival flags).

- [ ] **Step 5: Aggregate + honest write-up** — regenerate the results doc to a NEW file (never overwrite a baseline), and write the verdict with the scorer caveat (Task 7) and the cold/lazy/eager framing (no tier ladder).

---

## Self-review notes

- **Spec coverage:** (1) one-session cell → Task 3; (2) regime collapse → Task 1 + Task 6; (3) skill-fired assertion → Task 4; (4) persist-forward reconciliation → Task 5; (5) DAG → Task 6; (6) scorer scope → Task 7. All covered.
- **The riskiest unknowns are operational, not code:** does `/learn` in-session produce real episodes (Task 8 Step 2 gate), and does the agent reliably invoke BOTH skills headlessly (recall proven; learn unproven — Task 8 Step 2 also gates this; if `/learn` does not fire reliably, add an explicit "invoke /learn" retry like the existing note-14 retry).
- **No baseline overwrite:** results go to a new run dir + new results file.
- **Reversibility:** all changes are in `dev/eval/` tooling; nothing touches the shipped engram binary, skills, or prod vault.
