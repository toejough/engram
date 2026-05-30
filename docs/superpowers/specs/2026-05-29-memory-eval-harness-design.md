# Memory retrieval evaluation harness — design

Date: 2026-05-29. Brainstormed via `/please` against issue
[#637](https://github.com/toejough/engram/issues/637). Supersedes the
"build 637 first" reading of that issue: 637 (field-specific query) is
not the deliverable — it is a *substrate* a later milestone may need.
The deliverable is an **evaluation harness** that measures whether
memory retrieval actually helps an agent, so every retrieval design
choice becomes an experiment instead of a debate.

## Problem

Joe's suspicion: `engram` retrieval ranks poorly because `/recall`
forms field-shaped queries (situation / intended action) but matches
them against **whole-note** vectors, and we never actively pull on the
signals that actually co-vary with the work — recency, same-project,
same-area-of-codebase. We work sequentially in a project, in time, and
across code areas, but don't retrieve along those axes.

Empirical evidence found during orientation (real 526-note vault): the
current query-time connection machinery is **degenerate**. 3-hop
undirected wikilink BFS hits the 200-note cap immediately (the vault is
densely linked), so each per-phrase subgraph ≈ the whole vault, and
k-means returns giant clusters (130+ members) with silhouettes barely
above the 0.10 floor. The recall skill's Step 3a "link-to-bind"
(query-time connection *formation*) therefore can't fire — the
clustering it depends on never yields a coherent binding theme.

Three distinct things are half-conflated and must stay separated:

- **(A) direct-hit precision** — do the right notes surface at all.
- **(B) connection detection** — degenerate now (BFS explosion + coarse
  clustering).
- **(C) connection formation** — Step 3a link-to-bind; exists, blocked
  by (B).

Field queries (637) help (A); they do **not** fix the (B) BFS
explosion. We will not guess which matters — we will measure.

## Success criteria (the north star)

From Joe, verbatim intent:

- The **user** should not have to repeat lessons.
- The **LLM** should not waste cycles re-exploring known-bad patterns or
  re-searching for information it already learned.

The harness exists to detect movement on exactly these.

## Locked decisions

1. **Harness first, then evidence-driven redesign.** Build the
   measurement before changing retrieval. Measure the current system
   first to establish a baseline and test the salience hypothesis.
2. **End-to-end behavioral evaluation only.** No deterministic
   gold-set / labeled relevance. Score what the agent actually *does*.
3. **Differential across config arms**, capturing two metric layers
   (cost/efficiency + behavioral failure-modes) in the same runs.
4. **Real Claude Code headless agents** drive the tasks, with the
   actual shipping `recall`/`learn` skills (not a reimplemented loop),
   against a per-run sandbox vault via `ENGRAM_VAULT_PATH`.
5. **Real-vault + diverse-tasks** scenario model: copy the current real
   vault as-is; run a variety of realistic greenfield build tasks. No
   seeding, no historical snapshots, no synthetic vaults. If the
   baseline already retrieves well, that is a valid finding.
6. **Harness is build tooling in `dev/`**, invoked `targ eval <arm>
   [params…]`. The engram binary + skills are external artifacts under
   test and are **not** modified to host the harness.

## Scenario model

A **scenario** = a realistic greenfield build task run to a fixed,
checkable goal. Initial set (extensible):

- build a todo CLI app
- build a DORA-metrics dashboard
- build a SQLite explorer
- **calibration task** — chosen so a current-vault lesson is obviously
  on-point (e.g. a Go build where "use `targ`, not `go test`" must
  fire). Used to prove the harness can detect a *known* win before it
  is trusted on subtle deltas.

Tasks deliberately span domains/languages so retrieval is exercised on
both lexically-near and lexically-far-but-situationally-relevant
memories — directly probing the salience hypothesis (the relevant memory
is often relevant via a non-cosine signal, e.g. "this is a Go project /
an engram build", not via lexical overlap with "build a todo app").

**Vault:** a fresh copy-on-write clone of the current real vault per
(scenario × arm × trial), via `ENGRAM_VAULT_PATH`. `learn`-writes and
recall-time mutations during a run never leak across trials.

## Arms

An arm = (engram binary build + config) + (skill-bundle variant) +
(fresh vault clone). `targ eval` dispatches by arm name:

| Arm | Meaning |
| --- | --- |
| `nothing` | No engram skill/binary at all. **Floor** — how the agent does with no memory. |
| `skills-only` | Skills present, binary absent → recall's **degraded direct-read mode** (no BFS/cluster/hubs). Isolates the binary's contribution. |
| `current-state` | Full engram as-is (today's whole-note cosine + 3-hop BFS + clustering + Step 3a). **Baseline.** |
| `hops N` | Vary BFS depth (direct-only, 1, 3, 5…). |
| `subgraph CAP` | Vary subgraph cap. |
| `krange A..B` | Vary cluster k range. |
| `silhouette F` | Vary silhouette floor. |
| `fields …` | Field-specific / signal-based retrieval (milestone 3 / issue 637). |

**Arm constraint — recall-side vs learn-side.** Recall-only levers
(`hops`, `subgraph`, `krange`, `silhouette`, and the derived-note
synthesis toggle) run against the *same* vault clone, so they are
directly comparable to `current-state`. Any lever that changes
**learn-time** behavior (recording new signals, linking differently —
i.e. `fields` and signal-based learn) needs its vault **rebuilt by that
candidate's `learn` pipeline**, so those are milestone 3, after the
harness is proven.

**Derived-note synthesis** (Step 3a writing a new note from a cluster)
is a **skill-bundle variant** toggled on/off — answering Joe's direct
"is that helpful or harmful?" question.

## Metrics & detection

Two layers, scored per run from the agent's transcript + built
artifacts (reusing engram's transcript reader):

**Layer 1 — cost / efficiency (hard, no judgment):**

- task completed correctly (y/n) — per-scenario assertion (does the
  built app run / produce expected output)
- tool-call cycles to completion
- tokens consumed
- wall-time

**Layer 2 — behavioral failure-modes (the success criteria).** Hard
assertions wherever the scenario makes the signal detectable;
LLM-judge **only** for genuinely fuzzy signals (every judge call stacks
scoring-noise on run-noise — minimize it):

- **repeated a known lesson / violated a documented convention** —
  hard-detectable against conventions in the vault: used `go test`
  instead of `targ`, skipped TDD, wrong commit trailer, bypassed DI
  norms. Each is a concrete assertion over commands run / files
  produced.
- **re-searched / re-asked for known info** — agent queried (or asked
  the user) for a fact already in the vault.
- **took a known-bad path** — did the thing a vault memory warns
  against.
- **applied a relevant lesson** (positive) — judge-scored when not
  hard-assertable.

**Noise control.** A single run is meaningless with a real LLM. Run
**N trials per (scenario × arm)**; report rate distributions, not point
values. Budget arithmetic: 4 scenarios × 3 arms × 5 trials = **60
headless Claude Code sessions per pass**, each a full greenfield build.
This is the deliberate cost of rejecting a deterministic gold-set.
Dial trials/scenarios down for fast iteration, up for confidence.

**Calibration gate.** Before any subtle candidate-vs-baseline delta is
trusted, the harness must show it detects the obvious win on the
calibration scenario (`nothing` clearly worse than `current-state`). If
it can't detect a difference we *know* exists, it can't detect subtle
ones.

## Harness mechanics (dev tooling)

- All harness code under `dev/`. Honor engram's DI-everywhere split:
  pure logic (scenario specs, arm matrix, trial planning, scoring rules,
  metric aggregation, calibration gate) separated from I/O behind
  injected interfaces.
- DI'd adapters wired at the edge: `VaultCloner` (copy-on-write clone
  current vault → temp), `AgentRunner` (spawn a Claude Code headless
  session; return transcript path + workspace), `ScenarioScorer`
  (assertions + optional judge), `Clock`. Reuse engram's transcript
  reader as a library. Unit-test pure logic with imptest mocks;
  integration-test the thin adapters.
- `targ eval <arm> [params…]` runs the scenario set × N trials for that
  one arm, writes results JSONL, prints the distribution summary.
  Compare arms by diffing result files.

**Run loop**, per (scenario × arm × trial), isolated temp workspace:

1. Clone current vault → temp; set `ENGRAM_VAULT_PATH`.
2. Assemble agent workspace (skill bundle + engram build for the arm).
3. Spawn Claude Code headless with the fixed task prompt; capture
   transcript + built artifacts.
4. Score: Layer-1 cost + Layer-2 assertions (+ judge only where needed).
5. Record one result row keyed by (scenario, arm, trial).

Trials run concurrently within an API-rate budget. LLM nondeterminism
is absorbed by reporting rates.

**Lever mechanism (milestone 2 decision, recommended now):** the
`hops`/`subgraph`/`krange`/`silhouette` levers are currently hardcoded
in the binary (hop depth 3, k-range 2..7, 200-cap, 0.10 silhouette
floor per the F6/F9.1 spec). To keep engram's CLI surface clean and
avoid per-arm recompiles, expose them as **hidden env-var overrides**
read by the binary (`ENGRAM_HOPS`, `ENGRAM_SUBGRAPH_CAP`,
`ENGRAM_KRANGE`, `ENGRAM_SILHOUETTE`), defaulting to today's values when
unset — zero behavior change for normal use. The dev harness sets them
per arm. (Alternative considered: real `--hops`-style flags — more
discoverable, but expands the shipping surface the F6/F9.1 spec
deliberately kept closed.)

## Milestones

- **Milestone 1 (this round):** `dev/` harness + `targ eval
  nothing|skills-only|current-state`; the scenario set (todo CLI, DORA
  dashboard, sqlite explorer, calibration); Layer-1 + Layer-2 scoring;
  the calibration gate. **Deliverable:** a real
  floor-vs-skills-only-vs-baseline measurement — the salience
  hypothesis answered with numbers.
- **Milestone 2:** env-var lever overrides in the binary +
  `hops|subgraph|krange|silhouette` arms + derived-note-synthesis skill
  toggle; tune knobs against the harness.
- **Milestone 3:** `fields` arm + signal-based learn-time
  recording/linking (issue 637 substrate). Built **only** if M1/M2
  evidence shows whole-note + recall-side tuning is insufficient.

## Out of scope

- Changing the shipping `engram` CLI surface (levers go behind hidden
  env vars, not user-facing flags).
- Vault snapshotting / historical time-travel.
- Transcript-mined scenarios (the real vault + diverse tasks replaces
  the need; revisit only if synthetic-task realism proves inadequate).
- Auto-tuning / search over the knob space (the harness measures; a
  human reads the numbers and decides).
- Deterministic gold-set relevance scoring (explicitly rejected in
  favor of behavioral).

## Open questions (deferred, not blocking M1)

1. Exact trial count / scenario count for a "confidence" pass vs a
   "fast iteration" pass — tune once we see real per-run cost.
2. Which Layer-2 signals genuinely need the LLM-judge vs can be made
   hard-assertable by scenario design — settle per scenario as written.
3. Lever mechanism final form (hidden env vars vs flags) — recommended
   env vars; revisit at M2 if discoverability matters.

---

# Run 1 — empirical findings & redesign (2026-05-29/30)

The M1 harness was built (9 task commits + 1 fix on
`feat/eval-harness-m1`) and a first e2e calibration run executed
(`nothing` + `current-state`, 1 trial, 3 scenarios = 6 headless Go
builds). This section records what that run proved, the wall it hit,
and the redesign it forced.

## What Run 1 proved (validated)

- **Harness plumbing works end-to-end.** Clone vault → spawn real
  headless `claude` agent → parse result JSON → parse session JSONL →
  score → write results JSONL. All 6 builds completed; structured
  Layer-1 + Layer-2 metrics produced.
- **Arm isolation works.** `current-state` fired recall (20 `engram
  query` calls, recall's "Anchor concepts" preface present) and had the
  binary; `nothing` had zero `engram query` and no skills dir. Verified
  from the surviving `$TMPDIR/engram-eval-*` transcripts.
- **recall/learn skills load and recall *fires* in headless Claude
  Code** via per-arm `CLAUDE_CONFIG_DIR` (keychain creds + skills +
  binary-on-PATH). Non-trivial; this was a real unknown.

## The wall (calibration gate RED)

The calibration gate correctly returned `ErrCalibrationFlat` — **no
floor-vs-baseline delta**. Root causes, in order of depth:

1. **The `used-go-test-not-targ` check is non-discriminating by
   construction.** Every Go agent runs `go build`/`test`/`vet`, and in a
   bare `/tmp` module `go test` is *correct*. The check measures
   something memory cannot and should not change. (`occurred=true` in
   every arm × every scenario.)
2. **No clean greenfield *convention* discriminator exists.** The vault's
   codified, observable conventions (targ, AI-Used trailer,
   make-with-capacity, DI layout, t.Parallel, sentinel errors) all live
   in `~/.claude/CLAUDE.md` / `.claude/rules/` — loaded statically by
   *all* arms — so they cannot isolate the vault's contribution. The
   vault proper holds process / meta / engram-domain knowledge.
3. **The flagship vault gotcha (`229` yaml-v3 custom-scalar omitempty
   needs IsZero) does not reproduce.** Three reconstructions
   (string-backed plain, string-backed forced-quote `yaml.Node`,
   struct-backed forced-quote node) all omit the empty field correctly
   in yaml.v3 v3.0.1. A recalled memory can be *imprecise* — and a
   no-memory agent "burning cycles" can be doing the *correct* thing.
4. **Deeper conclusion:** the vault's discriminating value is
   **engram-domain-bound** — it was largely created *during the build of
   engram itself*. Measuring memory's behavioral value on portable
   greenfield tasks is a category mismatch.

## Redesign — self-seeding cold-vs-warm (agreed with Joe)

The vault must be **task-relevant by construction**. Instead of testing
a pre-existing vault against out-of-domain tasks, **let the agent
generate its own memories from its own mistakes**, then measure whether
replay benefits:

- **Cold arm (K trials):** fresh **empty** vault, recall+learn on. Agent
  works cold; learn writes its mistake-lessons into that trial's vault.
  → baseline cost distribution; produces candidate seed vaults.
- **Seed:** take a populated post-cold vault.
- **Warm arm (K trials):** vault = copy of the seed, recall+learn on.
  recall surfaces the prior lessons. → warm cost distribution.
- **Memory effect = cold − warm** on Layer-1 (turns/cost/failures).

Decisions: **identical replay task first** (a sanity floor — proves the
loop can regurgitate a just-written note; NOT proof memory is
*valuable*), **variant tasks later** (tests transfer/generalization).
Structure: **cold-vs-warm two groups** first; longitudinal accumulation
later. Most harness primitives transfer; only the orchestration changes
(variable = vault state, not skills present/absent).

## Blockers the redesign must solve FIRST (verify before building)

1. **learn does not autonomously fire in headless `claude -p`.** Run-1
   `current-state` clones still had 528 notes (= source); zero `engram
   learn` calls in any transcript. The cold→warm delta is impossible
   unless cold runs actually capture notes. Fix: explicitly trigger
   learn (append a `/learn` step to the task prompt, or run a second
   headless turn, or invoke `engram learn` directly) — and verify the
   captured notes are **mistake-specific**, not process-meta (the only
   notes learn wrote all session — `237`/`238` — were about its own dev
   process, which would not help a replay).
2. **learn's empty-vault first-run path is interactive.** On a vault with
   no progress marker, `engram transcript --mark` exits non-zero and the
   learn skill stops to ask the user via `AskUserQuestion` — which has no
   answerer in headless. Fix: pre-initialize the cold vault with a marker
   (or a minimal bootstrap) so the first-run prompt never fires.
3. **Correctness, not just turns.** "Fewer turns" can mean "gave up
   faster." Cold-vs-warm on cost is only interpretable if both completed
   *correctly*. Add a real per-task success check (the deferred
   `SuccessCmd`); `task_ok` from `IsError` is too weak.
4. **Persist evidence into results JSONL.** "Did memory fire / what did
   it do" required spelunking `$TMPDIR` transcripts. Persist the command
   list (or a transcript pointer) into the results rows, and don't reap
   the per-run vault/workspace before inspection.

**Gate for the redesign:** before writing cold/warm orchestration or
spending another multi-build run, run ONE probe — empty vault + headless
agent on a build task + explicit learn — and confirm learn writes
**replay-relevant mistake-notes** without choking on the first-run
marker. If it can't, that is the real bug and no orchestration fixes it.

## Status of the M1 harness code

The harness *machinery* (arms, scenarios, result/transcript parsing,
scoring, aggregation, calibration gate, DI adapters, `targ eval`
wiring) is built, unit-tested (`targ check-full` 8/8), and committed.
It is the reusable substrate for the redesign. The *scenario / check /
calibration* layer is superseded by the self-seeding design above and
will be reworked once the blockers are cleared.
