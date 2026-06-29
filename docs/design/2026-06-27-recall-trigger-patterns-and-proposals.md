# Recall-trigger patterns + 10 proposals

> **CORPUS CORRECTION — READ FIRST (the original corpus was unrepresentative).**
> The first version of this doc mined **one corpus that is 100% engram eval / memory-system work** and
> treated its taxonomy as Joe's recall-gap profile. **That corpus was not representative.** A later audit
> of `~/.claude/history.jsonl` (**15,022 prompts**) showed Claude was used across **~29 repos**, and engram
> was only about **a third** of that activity. The engram-only corpus survived because engram is the
> active project whose **full session transcripts** are still on disk; the **non-engram CLI transcripts
> aged out under retention** (only the desktop sandbox store remains, and that ~10GB is not real-repo work).
> So the original taxonomy was over-fit to eval-design vocabulary and engram code loci, and **only ~8% of
> real corrections resemble that corpus's content** (`engram-like` = 27/350 cross-repo).
>
> **There are now two corpora, and the gap between them IS the headline finding:**
>
> | corpus | what it is | n | depth | over-fire denominators? |
> |---|---|---|---|---|
> | **ENGRAM** | full session transcripts for the engram project | 59 TRIGGER + 23 CAPTURE | deep (full transcript: tool calls, turns, cues) | **YES** — the only place we can count over-fire |
> | **CROSS-REPO** | genuine corrections recovered from **user PROMPTS** in `history.jsonl`, across 24 repos | 350 corrections | shallow (**prompt text only** — surrounding transcript is pruned) | **NO** — transcripts are gone, so no denominators |
>
> The engram corpus keeps **depth** (and is the *only* source of real over-fire math); the cross-repo
> corpus supplies **breadth** (what real corrections actually look like) but is **prompt-only**, so it has
> **no over-fire denominators**. Every claim below is tagged to its corpus. Where the original doc said
> "this is Joe's recall-gap profile," the honest statement is: **the engram taxonomy is a map of one
> project's eval-design gaps; the cross-repo taxonomy is the real distribution, and it nearly inverts it.**

---

## 0. Headline verdict

**The near-inversion (the finding).** The two patterns that were **71%** of the engram corpus are **~4%**
of real corrections; the two patterns that are **76%** of real corrections were **essentially absent** from
the engram corpus.

| pattern | ENGRAM share (of 59 TRIGGER) | CROSS-REPO share (of 350) | verdict |
|---|---|---|---|
| eval-design (task-type) | **46%** | **2%** | **engram artifact** — eval-vocab over-fit |
| self-recommendation | **25%** | **2%** | **engram artifact** — late-task lever re-proposal, rare elsewhere |
| **verify-don't-guess** | **absent as a moment** | **41%** | **real dominant pattern** — was invisible in engram |
| **design-direction** | **not a category** | **35%** | **real dominant pattern** — was invisible in engram |
| step-boundary | **7%** | **15%** | real, and **larger** than engram showed |
| tool-imminent | 14% | 2% | engram-inflated (go-lint loci) |
| phrase-intent | 7% | 2% | engram-inflated |
| ask-keyed / other | 2% | ~3% | tail |

**The deeper meta-finding (the part that changes the recommendation).** The three dominant cross-repo
patterns — **verify-don't-guess (41%), design-direction (35%), step-boundary (15%) = 90% of all
corrections** — are overwhelmingly **discipline / application failures, not retrieval misses.** The rule
usually **already exists**:

- **verify-don't-guess** is literally a standing rule in engram's `CLAUDE.md` ("Verify, don't guess") and
  Joe's global memory — yet it is the #1 correction (about to use an API/path/format/command not yet
  checked).
- **step-boundary** is mostly **TDD-RED enforcement** and **commit-between-steps** (e.g. "wrote a test that
  passes in the failure state, violating the TDD RED requirement"; "agent continued adding changes without
  the user having committed the previous work") — TDD and brainstorm-first are already mandated.
- **design-direction** is "agent built approach X, user wanted Y," and **56% of it (68/121) is
  `memory_could_help = n`** — no general memory would have helped; it needs a **propose-before-build gate**,
  not a recall.

So **"fire recall more" is an even smaller lever than the engram analysis implied.** Across the whole
cross-repo corpus **37% of corrections (128/350) are `memory_could_help = n`**, and the dominant patterns
are failures to **apply an existing rule**. The high-leverage fix is **discipline ENFORCEMENT — skills,
gates, hooks — not recall-trigger timing.** Triggering recall faster cannot apply a rule the agent already
ignored.

**The ceiling still holds (ENGRAM-derived).** Even within the part triggering *can* reach: in the engram
corpus **28% of corrections (23/23 CAPTURE) had no memory at all** — a write-side capture problem (P5), not
a trigger problem. The cross-repo corpus corroborates a large unreachable share (37% `memory_could_help =
n`). And the correction signal itself is noisy: the cross-repo correction-detector ran at a **~29%
false-positive rate** (350 genuine out of a 494 stratified sample), so any "fire on a correction" lever
inherits that noise.

**Headline (one sentence):** The real correction distribution nearly **inverts** the engram-only one —
the dominant patterns are **verify-don't-guess (41%) + design-direction (35%) + step-boundary (15%)**, and
they are mostly **discipline/application failures against rules that already exist**, so the high-value
moves are **enforcement gates and free hooks**, not faster recall; the engram-only over-fire math still
proves the "fire more" triggers are fatal, and a ~28–37% slice has **no memory to recall at all**.

### 0.1 The cost model (unchanged — still valid, applied uniformly below)

> **memory-tax(trigger) = over-fire-ratio × per-fire-cost.** Two regimes, and which one a proposal is in
> decides whether the ratio matters:
>
> - **Recall-firing triggers** — per-fire-cost ≈ **190s** (the full recall body; per
>   `recall-cost-isolation.md`, 2026-06-25). At 190s/fire, Joe's **~10× over-fire bound binds hard**:
>   11×-on-user-turn and 147×-on-tool-call are *fatal*; only ≤ ~10× is viable.
> - **Free-fire triggers** — a PreToolUse/PostToolUse **hook**, an always-loaded **static doc / gate**, or
>   a **binary re-rank** costs ≈ **0s** per fire. The ratio does **not** bind: a **43× hook is fine**, a
>   380× static doc is fine. The only cost is attention/noise, not wall-time.
>
> This single model explains every rating: it is why **11×-recall is fatal but 43×-hook is fine** — and it
> is why the three new cross-repo proposals are framed as **free gates/hooks**, since they have no
> over-fire denominator to bound (see §0.2) and a free surface does not need one.

### 0.2 The fire-unit, and the cross-repo denominator gap (stated once)

> A recall trigger **fires at TASK INITIATION** — a user turn / new task context where skill selection
> happens — **not at every assistant turn or every tool call.** All over-fire ratios below are computed at
> this unit, **except** hook proposals (fire at a tool event) and static-doc/gate proposals (always-present;
> "fire" once per context). Every per-proposal number is tagged **DERIVED** (computed from a measured
> denominator) or **ESTIMATE** (behavioral signal, no clean denominator → wide range, confidence lowered).
>
> **Critical limitation:** over-fire denominators exist **ONLY in the engram corpus**, because computing a
> ratio requires the full transcript (how many tool calls / turns / correct uses occurred). The cross-repo
> corpus is **prompt-only** — the surrounding transcripts were pruned — so **cross-repo proposals carry NO
> over-fire denominator and are marked `NO-DENOMINATOR (qualitative)`.** This is stated, not hidden: a
> cross-repo proposal's coverage is real (we counted the corrections), but its over-fire is **unmeasurable
> from this data.** That is the main reason the new top proposals are scoped to **free surfaces**, where the
> missing denominator does not matter.

---

## 1. The two corpora + provenance

> **Provenance / data trail.** Committed at
> [`2026-06-27-recall-trigger-data/`](2026-06-27-recall-trigger-data/):
>
> - **`extract_moments.py`** — deterministic extractor over `~/.claude/projects/*engram*/*.jsonl`.
>   Reproduces the **engram** denominators (8674 tool calls, 635/586 user turns, the 551→78 candidate
>   filter, tool/git/skill counts).
> - **`moments_all.json`** (83) / **`moments_trigger.json`** (59) — the classified **engram** moments
>   (`summary`/`source`/`signal_category`/`klass`/`preceding_cue`/`lesson`): the audit trail for the
>   59 TRIGGER / 23 CAPTURE split and the per-cluster source IDs.
> - **`xrepo_corrections_classified.json`** (350) — the **cross-repo** corpus: genuine corrections
>   classified from the user PROMPTS in `history.jsonl`, each with
>   `repo`/`is_correction`/`signal_category`/`what_corrected`/`memory_could_help`/`generic_vs_engramish`.
>   **Prompt-only** (transcripts pruned), so no over-fire denominators.
> - **`~/.claude/history.jsonl`** (**15,022 prompts**, not committed — Joe's machine) — the source that
>   revealed engram was ~⅓ of activity and the basis for the 494-prompt stratified sample across ~29 repos.
>
> Engram denominators are deterministic; all classifications are LLM judgment, adversarially critiqued
> (see the doc's git history).

**Why the original corpus was unrepresentative (the pruning mechanism).** Claude Code retains full
transcripts for **active** projects and ages out the rest. Engram is Joe's current focus, so its
transcripts survived intact — which is exactly why it was *available* to mine, and exactly why mining only
it was a **survivorship-biased** sample. The non-engram CLI transcripts (the other ~⅔ of work, ~29 repos)
were pruned; only their **user prompts** persist in `history.jsonl`. (The ~10GB desktop sandbox store is
not real-repo work and was excluded.) The cross-repo corpus is therefore **prompt-only by necessity**: it
recovers *what was corrected* from the prompt text, but cannot recover the surrounding transcript, so it
cannot supply over-fire denominators.

**Stratified sample.** 494 prompts were drawn stratified across the ~29 repos; **350 were genuine
corrections** (the rest were non-corrections → a **~29% false-positive rate** for the correction
detector). The 350 genuine corrections fall across **24 repos** (top contributors: `traced` 45,
`projctl` 41, `letter_blocks_project` 29, `specwalk` 28, `5m-increments` 23, `glowsync` 22,
`claude-pencil` 21, `wasm` 20, `imptest` 18, …).

---

## 2. Taxonomy — both distributions, side by side

**Lead table — signal split, ENGRAM (59 TRIGGER, DERIVED from `moments_trigger.json`) vs CROSS-REPO
(350, from `xrepo_corrections_classified.json`):**

| signal_category | ENGRAM count | ENGRAM share | CROSS-REPO count | CROSS-REPO share | reading |
|---|---|---|---|---|---|
| **verify-don't-guess** | 0 (absent) | 0% | **143** | **41%** | **real #1** — invisible in engram |
| **design-direction** | 0 (no category) | 0% | **121** | **35%** | **real #2** — invisible in engram |
| **step-boundary** | 4 | 7% | **51** | **15%** | real #3 — engram under-counted it |
| task-type (eval-design) | **27** | **46%** | 7 | 2% | **engram artifact** |
| self-recommendation | **15** | **25%** | 7 | 2% | **engram artifact** |
| tool-imminent | 8 | 14% | 7 | 2% | engram-inflated |
| phrase-intent | 4 | 7% | 6 | 2% | engram-inflated |
| ask-keyed / other / file-type | 1 | 2% | 8 | ~2% | tail |
| **total** | **59** | 100% | **350** | 100% | — |

**The inversion, stated plainly:** eval-design + self-recommendation = **71% of the engram corpus** but
**~4% of real corrections**. verify-don't-guess + design-direction = **76% of real corrections** but **0%
of the engram corpus** (they did not even appear as failure moments — the original doc's caveat already
flagged that verify-don't-guess was "absent as a failure moment here"). The engram taxonomy measured the
*shape of engram eval work*, not the *shape of Joe's corrections*.

**Cross-repo content split (`generic_vs_engramish`, 350):**

| bucket | count | share | meaning |
|---|---|---|---|
| generic-coding | 193 | 55% | ordinary coding corrections — would recur in any repo |
| project-specific | 130 | 37% | tied to a project's conventions (the per-project static-rule surface) |
| **engram-like** | **27** | **8%** | resembles the original corpus — **only 8% of reality looked like the whole original doc** |

**Cross-repo memory-addressability (`memory_could_help`, 350):**

| value | count | share | implication |
|---|---|---|---|
| y | 161 | 46% | a memory plausibly helps — the addressable trigger/capture zone |
| maybe | 61 | 17% | marginal |
| **n** | **128** | **37%** | **no memory would help** — gate/workflow/write-side, not recall |

**The discipline cross-tabs (why "fire recall more" is a small lever):**

| pattern | n | `memory_could_help` | what it really is |
|---|---|---|---|
| design-direction | 121 | y 29 / maybe 24 / **n 68 (56%)** | mostly **not memory-addressable** → propose-before-build **gate** |
| step-boundary | 51 | **y 43 (84%)** / maybe 3 / n 5 | TDD-RED + commit-between-steps — rule **already exists**, needs **enforcement** |
| verify-don't-guess | 143 | y 63 / maybe 32 / n 48 | rule **already exists** in CLAUDE.md — a **discipline/gate** failure, not a recall miss |

Read together: the top-3 patterns are **90% of corrections (315/350)**, they are dominated by failures to
**apply rules that already exist**, and a third of all corrections are not memory-addressable at all. The
lever is **enforcement**, not retrieval timing.

---

## 3. Cost / over-fire — ENGRAM-ONLY math (clearly labeled), cross-repo has no denominators

**Lead table — over-fire at the fire-unit (§0.2). DERIVED rows are ENGRAM-ONLY; cross-repo rows carry NO
denominator:**

| signal / candidate trigger | over-fire | unit | corpus / D-E | per-fire cost | bound? |
|---|---|---|---|---|---|
| every tool call | **147×** (8674/59) | tool call | ENGRAM **DERIVED** | 190s | **FAIL** |
| every user turn | **11×** (635/59) | user turn | ENGRAM **DERIVED** | 190s | **FAIL** |
| eval-design vocab → recall | **~7×** (85/12) | task init | ENGRAM **DERIVED** | 190s | borderline — **engram-only value** |
| done-claim → verify-recall | **~380×** (1141/3) | asst turn | ENGRAM **DERIVED** | 190s | **FAIL → free hook only** |
| destructive-git → hook | **43×** (43/1) | tool event | ENGRAM **DERIVED** | ~0s | **PASS (free)** |
| verify-don't-guess gate (P1) | **—** | gate/tool event | CROSS-REPO **NO-DENOMINATOR** | ~0s (gate) | n/a — free surface |
| design-direction gate (P2) | **—** | task-init gate | CROSS-REPO **NO-DENOMINATOR** | « 190s (gate) | n/a — free surface |
| step-boundary enforcement (P3) | **—** | step boundary | CROSS-REPO **NO-DENOMINATOR** | ~0s (hook/gate) | n/a — free surface |

**Engram denominators used** (DERIVED, engram corpus only): 8674 tool calls; 635 user turns (586 genuine);
85 eval-vocab turns; ~1141 done-claim assistant turns; .md edits 944; SKILL.md 105; git commit 369; git
destructive 43; git push 43; Agent dispatch 567; Skill 145.

**Why the cross-repo proposals are scoped to free surfaces.** Because the transcripts are pruned, we
**cannot** compute how often a verify/design/step pattern *fires harmlessly* — there is no denominator. A
recall-firing trigger with an unknown over-fire ratio at 190s/fire is unshippable (it could be the fatal
147× case). The escape is the §0.1 cost model: put these on **free surfaces (gates, hooks, static rules)**
where the over-fire ratio **does not bind**, so the missing denominator does not block the proposal. This
is a deliberate, stated consequence of the pruning — not a hidden assumption.

**Engram application note (still valid):** the engram self-recommendation cluster (25% *there*, 2%
cross-repo) is **APPLICATION**, not retrieval — the relevant note is *already in context* but **outranked
by chunks at synthesis** (note 82). Its fix is a **re-rank (#655) + reconcile gate (#656)**, not "fire
recall more." This mechanism is preserved as a free binary re-rank (P6) even though the cluster is small
cross-repo, because the **ranking bug it fixes is corpus-independent**.

---

## 4. The 10 proposals (re-ranked to cross-repo reality)

**Lead table — the set, re-ranked. New top-3 are the cross-repo-dominant patterns; the demoted
eval-design trigger drops to P10.** (`old-id` traces to the first version.)

| id | name | old-id | solution-surface | coverage | over-fire | per-fire cost | rating |
|---|---|---|---|---|---|---|---|
| **P1** | verify-before-asserting-external-contracts gate | new | gate/PreToolUse hook | **41% cross-repo (143)** | **NO-DENOMINATOR** | ~0s | **CONTENDER (top)** |
| **P2** | propose-approach-before-building gate | new (≈#656) | please/brainstorming gate | **35% cross-repo (121)** | **NO-DENOMINATOR** | « 190s | **CONTENDER (top)** |
| **P3** | step-boundary enforcement (TDD-RED + commit-between) | new / old P9 | skill/gate/Stop-hook | **15% cross-repo (51)** | NO-DENOM (xrepo); 380× (engram, **D**) | ~0s (free hook) | **CONTENDER** |
| **P4** | destructive-git PreToolUse hook | P4 | binary/hook | 1 engram; class-wide | **43×** (tool event, **D**) | ~0s | **CONTENDER (KEEP)** |
| **P5** | capture-on-correction (write-side) | P6 | learn / write-side | the **28–37% no-memory** slice | ~13% base-rate (**E**) | learn-side | **CONTENDER** |
| **P6** | note-negation re-rank (#655) | P5 | re-rank (binary) | ranking bug (corpus-independent) | n/a (no new fires) | ~0s | **CONTENDER** |
| **P7** | two-speed recall quick-probe (#657) | P3 | split-skill / two-speed | enabler for any recall trigger | inherits caller's | decide cheap / exec deferred | **CONTENDER** |
| **P8** | binary-staleness PostToolUse hook | P10 | binary/hook | 1 engram (note 106) | rare (tool event, **E**) | ~0s | **CONTENDER (free, engram-specific)** |
| **P9** | per-project static rules → `go.md`/CLAUDE.md | P7+P8 | static-doc | 5+3 engram; 37% project-specific xrepo | always-loaded (**D** denom) | ~0s | **PARK (right win, static surface)** |
| **P10** | eval-design recall trigger | **P1 (demoted)** | skill-body / frontmatter | ~12 engram (20% of 59) | ~7× (task init, **D**) | 190s→probe | **PARK (engram-specific, low cross-repo value)** |

Surfaces spanned: gate (P1, P2), skill/Stop-hook (P3), binary/hook (P4, P8), write-side (P5), re-rank (P6),
two-speed (P7), static-doc (P9), skill-body/frontmatter (P10). The cheapest regimes (gates, hooks, static
docs) dominate the top of the list — exactly because they escape the §0.1 over-fire bound and the
cross-repo denominator gap.

---

### P1 — Verify-before-asserting-external-contracts gate (NEW — the 41% pattern)

| field | value |
|---|---|
| problem-pattern | **verify-don't-guess is the #1 real correction (143/350, 41%).** The agent is about to **use an external contract it has not checked** — an API signature, a file path, a data format, a CLI command/flag, an import path — and asserts/acts on a guess. Examples: "missed a hardcoded import path another LLM caught"; "wrong path to the impgen binary"; "wrote tests without verifying they actually catch the failure case." The rule **already exists** (engram CLAUDE.md "Verify, don't guess"; Joe's global memory) — this is an **application/discipline** failure. |
| trigger-signal | the agent is **about to assert or act on an unverified external contract**: emitting a path/flag/field/signature it has not read, or claiming a command/target exists. Detectable at a tool boundary (about-to-Edit-with-a-guessed-path, about-to-Bash-an-unverified-command) or as a pre-assert gate. |
| solution-surface | **free gate / PreToolUse hook** — interpose before the unverified use: "this path/flag/contract hasn't been read this session — verify (Read/grep/`--help`) before asserting." Not a recall fire; a discipline checkpoint. Pairs with the existing CLAUDE.md rule (enforce what's already written). |
| coverage | **41% cross-repo (143)** — the single largest correction class. (generic-coding 81/143, project-specific 56 — broadly cross-cutting.) |
| over-fire | **NO-DENOMINATOR (qualitative).** Prompt-only corpus → we cannot count harmless verified uses. Mitigated by the **free** surface: even a high over-fire ratio costs ~0s on a gate/hook. |
| trigger-cost | **decide** = cheap heuristic at the tool boundary (was this path/command read/checked this session?); **execute** = ~0s prompt, **not** a recall fire. |
| test-sketch | Replay a prompt where the agent emits an unread import path / non-existent flag; assert the gate interposes and forces a verify step before the assertion ships. Precision check: a path the agent *did* read must not trip it. |
| prior-art | engram CLAUDE.md "Verify, don't guess"; Joe's `feedback_verify_*` memory cluster; note 94 (verify-metric); P4/P8 hook mechanics. |
| **rating** | **CONTENDER (top).** Highest coverage of any proposal and squarely cross-cutting. NO-DENOMINATOR is acceptable **because the surface is free** — the §0.1 bound does not apply. The hard part is **detection precision** (knowing "unverified"), which the test-sketch must pin down. |

---

### P2 — Propose-approach-before-building gate (NEW — the 35% pattern)

| field | value |
|---|---|
| problem-pattern | **design-direction is the #2 real correction (121/350, 35%):** the agent **builds the wrong approach** and the user redirects ("chose to error on generics instead of supporting type params"; "created a new `realMain` instead of using the existing `Run`"; "complex `packages.Load` when a deep-equal suffices"). Crucially, **56% (68/121) is `memory_could_help = n`** — no general memory would have prevented it. This is a **workflow-gate** problem: surface the approach for a yes **before** building, not retrieve a note. |
| trigger-signal | a **non-trivial implementation about to start** without an approach having been proposed/approved — the brainstorm-first / please-gate boundary. |
| solution-surface | **free gate (please / brainstorming)** — a propose-approach-then-confirm step before multi-file building. This is exactly what `superpowers:brainstorming` and the `please` plan-gate already encode; the proposal is to **enforce** that gate where corrections show it was skipped, and (engram-side) wire it as please-gate **#656**. |
| coverage | **35% cross-repo (121)** — second largest. Generic 59 / project-specific 59 (evenly split — applies everywhere). |
| over-fire | **NO-DENOMINATOR (qualitative).** Prompt-only → cannot count tasks where building-without-proposing was fine. The gate is **« 190s** (a short propose/confirm, no full recall), so the bound is slack. |
| trigger-cost | **decide** = "is this a multi-file/architectural change with no approved approach?" (cheap); **execute** = a short propose-and-confirm pass, no recall fire. |
| test-sketch | A task where the obvious-but-wrong approach diverges from the user's intent; assert the gate forces an approach proposal that surfaces the fork **before** code is written. (Current baseline: agent builds straight through.) |
| prior-art | `superpowers:brainstorming`; the `please` plan-gate; **#656** (gate analytical recommendations as artifacts); the 68 `n` design-direction cases that no memory addresses. |
| **rating** | **CONTENDER (top).** Targets the 35% class **and** the part of it (56%) that is provably not a recall problem. Risk: a propose-gate that fires on trivial edits becomes friction — scope it to multi-file/architectural changes (test-sketch precision arm). |

---

### P3 — Step-boundary enforcement: TDD-RED + commit-between-steps (NEW / absorbs old P9)

| field | value |
|---|---|
| problem-pattern | **step-boundary is the #3 real correction (51/350, 15%),** and **84% (43/51) is `memory_could_help = y`** — yet the rules **already exist** (TDD-RED, commit-between-steps, status check-ins). Examples: "wrote a test that passes in the failure state, violating TDD RED"; "fixed code before verifying the test was failing first"; "continued adding changes without the user having committed the previous work"; "proceeded without a status check-in at a natural stopping point." This is **enforcement**, not retrieval. The engram instance (done-claim before verifying, 4/59) is the same pattern. |
| trigger-signal | a **step boundary**: about to write implementation before a failing test exists (TDD-RED); about to start the next step before the prior was committed; a natural stopping point passed without a check-in. |
| solution-surface | **free skill/gate/Stop-hook** — a RED-gate (block impl edits until a failing test is observed) and a commit-boundary check between plan steps. **Never a recall fire:** the engram done-claim denominator is **~380× at 190s = catastrophic** as a recall trigger; it is only viable as a **free** Stop-hook/checklist. |
| coverage | **15% cross-repo (51).** Engram done-claim subset: 4/59 (**DERIVED**). |
| over-fire | **NO-DENOMINATOR cross-repo (qualitative).** ENGRAM done-claim: **~380×** (1141/3, asst turn, **DERIVED**) — proof it is fatal as a recall fire, fine as a free hook. |
| trigger-cost | **decide** = detect the boundary (cheap: "impl edit with no failing test observed"; "next step, prior uncommitted"); **execute** = ~0s gate/hook, no recall. |
| test-sketch | RED-gate: attempt an implementation edit with no failing test recorded; assert the gate blocks until RED is observed. Commit-gate: start step N+1 with step N uncommitted; assert the check interposes. |
| prior-art | `superpowers:test-driven-development`, `verification-before-completion`; engram notes 54/64/66/90; old P9 (the 380× cautionary case). |
| **rating** | **CONTENDER.** Big cross-repo class, rules already mandated → pure enforcement. The 380× engram number is the textbook §0.1 case: real pattern, **fatal as recall, fine as a free hook.** |

---

### P4 — Destructive-git PreToolUse hook (KEEP — genuinely cross-cutting)

| field | value |
|---|---|
| problem-pattern | engram note 25: `git restore`/`checkout` run on user-deleted files, treating deliberate deletion as data loss. Maps directly to Joe's **global** rule "never run destructive git without status/diff first" — cross-cutting, not engram-specific. |
| trigger-signal | a `git restore` / `git checkout -- ` / `clean -fd` about to run (PreToolUse on Bash). |
| solution-surface | **binary/hook** — PreToolUse hook that pauses: "these deletions weren't yours — intentional?". |
| coverage | **1** named engram moment (note 25); class of destructive-git in general (applies in every repo). |
| over-fire | **43×** (43 destructive-git events / 1 genuine, tool event, **ENGRAM DERIVED**). Free surface → fine. |
| trigger-cost | **decide** = regex on the command (sub-ms); **execute** = ~0s prompt, not a recall fire. |
| test-sketch | Fire `git restore` on a path the agent did not delete; assert the hook interposes before execution. |
| prior-art | note 25; Joe's global "never run destructive git without status/diff first." |
| **rating** | **CONTENDER (KEEP).** 43× is fatal for a recall trigger but **fine on a free hook** — the clearest cheap win, and cross-cutting (Joe's global rule, every repo). |

---

### P5 — Capture-on-correction (write-side — reaches the no-memory slice)

| field | value |
|---|---|
| problem-pattern | A large slice of corrections have **no memory to recall**: engram **28% CAPTURE (23/23)**; cross-repo **37% `memory_could_help = n` (128/350)**. Triggering ships nothing for these — it is a **write-side** gap. The cross-repo data makes this *more* important than the engram-only view implied. |
| trigger-signal | a **user correction** event ("why are you…", "that won't scale", "stop", "use the existing X"); the write side, not the read side. |
| solution-surface | **write-side / learn** — capture each correction as a reproducible guard at the moment it happens (note 67: turn a correction into a non-recurring guard). |
| coverage | reaches the **28% (engram) / 37% (cross-repo)** that triggering cannot — **this is the learn side, stated plainly: not a recall trigger.** |
| over-fire | **ESTIMATE.** Engram ~82 corrections / 635 turns ≈ **~13% base rate**; precision bounded by the **~29% correction-detector false-positive rate** (§1) — so capture must be confirmable, not auto-committed. |
| trigger-cost | learn-side write; no recall fire. |
| test-sketch | Replay a session with a known novel correction; assert a guard is captured that would TRIGGER next time — and that a false-positive "correction" does not spawn a junk note. |
| prior-art | note 67 (corrections→reproducible tests); **#635** (learn explicit-request triggers); the 23 CAPTURE + 128 `n` cases. |
| **rating** | **CONTENDER.** The **only** lever that moves the no-memory ceiling, and the cross-repo data enlarges that ceiling to ~37%. Belongs to **learn**, not triggering — included because the ceiling is now the dominant unreachable mass. Gate writes behind correction-confidence to survive the 29% FP rate. |

---

### P6 — Note-negation re-rank (#655) — corpus-independent ranking fix

| field | value |
|---|---|
| problem-pattern | When a directly-relevant note (especially a negation/"already-tried" note) **is in context but outranked by chunks at synthesis** (engram note 82), recall surfaces the wrong thing. The self-recommendation cluster that exposed this is small cross-repo (2%), but the **ranking bug is corpus-independent** — any retrieval can bury a negation under co-topical chunks. |
| trigger-signal | none — runs **inside every existing recall**; re-ranks so a directly-relevant note can **override** a chunk flood. |
| solution-surface | **re-rank (binary)** — note-negation override in `engram query` candidate ranking. |
| coverage | the ranking bug itself (engram ~13/15 of self-recommendation *there*); cross-repo value is the **mechanism**, not the cluster size. |
| over-fire | **n/a** — adds **no new fires**; modifies the output of recalls already happening. |
| trigger-cost | **~0s** — a ranking tweak in the binary; no extra LLM round-trip. |
| test-sketch | Vault with a negation note + many co-topical chunks; assert the negation note ranks above the flood (RED: note buried). |
| prior-art | **#655**, **#652** (recency-weighted centroid); notes 81, 82. |
| **rating** | **CONTENDER.** Free, no over-fire, fixes the actual ranking mechanism. Kept despite the small cross-repo cluster because the **bug is general** — but de-emphasized vs the original (it is no longer near the headline). |

---

### P7 — Two-speed recall quick-probe (#657) — the enabler

> **Refined + revived 2026-06-29 → `docs/design/2026-06-29-recall-depth-dial-design.md`.** That design generalizes
> P7 into a 2-rung glance/deep dial and **refines the cut line**: the cut is **read-vs-write** (keep Step-2
> matched-note retrieval + 2.5B recency-resolution — the win-nucleus — and drop the *write* side), **not**
> "skip Step-2 paging" as stated below. Glance reduces Step-2 cost via *fewer phrases*, not by skipping it.

| field | value |
|---|---|
| problem-pattern | Recall has **no two-speed path** — any fire runs the full **190s** body. That is what makes every recall-firing trigger borderline-to-fatal (the §0.1 bound). A "cheap to decide" trigger still pays 190s to execute. |
| trigger-signal | not a standalone trigger — a **mode** any caller (P10, a recall-probe) selects: "fast surfaced+rank check, not full synthesis." |
| solution-surface | **split-skill / two-speed** — quick-probe path (Step-0/1 surface + rank; skip Step-2 paging / Step-2.5 synthesis) vs the full body. |
| coverage | **enabler** — covers no moments itself; **lowers per-fire-cost**, which is the only thing that could make a recall-firing trigger (P10) honest. |
| over-fire | inherits the caller's ratio; the point is the **per-fire-cost** term, not the ratio. |
| trigger-cost | **decide** = classifier match (cheap); **execute** = quick-probe (Step-2 paging is the ~43–63% dominant cost per `recall-cost-isolation.md`, so deferring it is the win). |
| test-sketch | Measure quick-probe wall-time vs full body on the same query; assert it surfaces the right note's rank without the Step-2 page-in. |
| prior-art | **#657** (recall: safe procedure-time cuts, C7-gated); `recall-cost-isolation.md`; note 78. |
| **rating** | **CONTENDER.** Precondition for any honest recall-firing trigger. Lower priority now that the headline lever is enforcement, not recall — but still the right fix for recall's cost. |

---

### P8 — Binary-staleness PostToolUse hook (free, engram-specific)

| field | value |
|---|---|
| problem-pattern | engram note 106: ran `targ build` (no such target); a stale binary on PATH made measurements read as unchanged — a near-mis-conclusion. |
| trigger-signal | a `targ build` (non-existent target) invocation, or a stale-mtime engram binary — a deterministic tool event. |
| solution-surface | **binary/hook** — PostToolUse hook on Bash matching `targ build`, emitting "no `targ build` target; install with `go install ./cmd/engram`." |
| coverage | **1** engram moment (note 106); class of stale-binary measurement bugs. **Engram-specific** (the `targ build` non-target is an engram fact). |
| over-fire | rare (only on the matched command). Tool event. **ESTIMATE** (no `targ build` denominator measured). |
| trigger-cost | ~0s — a hook print; no recall fire. |
| test-sketch | Run `targ build`; assert the hook prints the `go install ./cmd/engram` correction. |
| prior-art | note 106; engram CLAUDE.md ("there is no `targ build` target"). |
| **rating** | **CONTENDER (free, engram-specific).** Cheap and deterministic; prevents a silent measurement-invalidation bug. Lower cross-repo value than P1–P5 (the fact is engram-local), but ~0s to keep. |

---

### P9 — Per-project static rules → `go.md` / CLAUDE.md (PARK: right win, static surface)

| field | value |
|---|---|
| problem-pattern | Two recurring static-rule clusters. (a) engram Go-lint conventions (modernize/`slices.Backward`, funlen/wsl scope-lift, `bytes.Split` trailing-newline + nil-guard, unused-field) — notes 37/43/57/58/59. (b) behavioral defaults (cadence assumptions, results-as-a-labeled-table) — notes 45/`c1195a11#1272`/`a19e7b75#5589`. Cross-repo, **37% of corrections are project-specific** — the general form of this surface. |
| trigger-signal | editing in-scope files (the glob scope of `.claude/rules/go.md`) / about to set a default or present results. |
| solution-surface | **static-doc** — extend `.claude/rules/go.md` and the project CLAUDE.md with the recurring conventions; per project, the project-specific 37% lives here. |
| coverage | **5 + 3 engram**; the **37% project-specific** cross-repo slice is the general target (per-repo static rules, not a global trigger). |
| over-fire | **always-loaded** for the glob; the "fire" is free; the denominator does not gate it. **ENGRAM DERIVED denom.** |
| trigger-cost | ~0s — a few always-present lines; no recall fire. |
| test-sketch | Write an in-scope `*.go` file with a backward for-loop; assert the static rule steers to `slices.Backward` with no recall. |
| prior-art | notes 37/43/57/58/59 (lint), 45/`a19e7b75#5589` (cadence/table); existing `.claude/rules/go.md`; Joe's MEMORY.md. |
| **rating** | **PARK (real win, wrong surface for *triggering*).** Genuine recurring failures, but the fix is **static per-project rules**, not a recall trigger. Merges old P7+P8. |

---

### P10 — Eval-design recall trigger (DEMOTED — engram-specific, low cross-repo value)

| field | value |
|---|---|
| problem-pattern | In the **engram** corpus, eval-design is the largest cluster (27/59; 37/59 eval-vocab): the agent designs an eval (distractor ages, terminal literalness, retrieval-vs-synthesis metric) without recalling the note that warns against the trap. **In the cross-repo corpus this collapses to 2% (7/350)** and is `engram-like` content — so it is **specific to engram eval work**, not a general lever. |
| trigger-signal | eval-design vocabulary at task initiation (eval, harness, fixture, probe, A/B, cold/warm, distractor, RED baseline). |
| solution-surface | **frontmatter-description (primary)** — recall's `description:` governs auto-invoke (note 100); add eval-design phrasings. **skill-body (secondary)** — an eval-design task-type routing line. |
| coverage | **~12/59 engram (20%)** — the genuine eval-design recall-fixable slice. **~2% cross-repo.** |
| over-fire | **~7×** (85 eval-vocab turns / ~12, task init, **ENGRAM DERIVED**). Under the ~10× bound, but **only paired with P7** (else 190s/fire makes 7× borderline-fatal). |
| trigger-cost | **decide** = cheap vocab match; **execute** = 190s recall (until P7 makes it a quick-probe). |
| test-sketch | Cold pilot: an eval-design task-init turn omitting the obvious vocab; assert the trigger fires and a non-eval turn does not (precision both ways). |
| prior-art | notes 42/44/75/83/86 (eval traps), note 104 (free retrieval-probe gate); #654 (C7 harness); requires #657 (P7). |
| **rating** | **PARK (engram-specific, low cross-repo value).** This was **P1 in the original doc** — the demotion *is* the correction: it addresses ~2% of real corrections / 8%-resembling content, and is viable only after P7. Worth keeping for engram's own eval workflow, but it is **not** a general recall-trigger win. |

---

## 5. Recommendation (within the set — no member deleted)

**Ship order, re-prioritized to cross-repo reality (enforcement first, recall last):**

1. **Enforcement gates for the dominant 90% — P1, P2, P3.** These hit the real top-3 patterns
   (verify-don't-guess 41%, design-direction 35%, step-boundary 15%) on **free gate/hook surfaces**. They
   carry **NO over-fire denominator** (prompt-only corpus) — which is exactly why they must live on free
   surfaces, where the missing denominator does not block them. The hard engineering is **detection
   precision** ("unverified," "no approved approach," "no RED test yet"), pinned by each test-sketch. These
   are **enforcement of rules that already exist**, the single biggest lever the cross-repo data revealed.
2. **Free deterministic hooks — P4, P8.** Zero per-fire cost, the bound does not apply; P4 is cross-cutting
   (Joe's global destructive-git rule), P8 is a cheap engram-local guard.
3. **Write-side to lift the now-larger ceiling — P5.** ~28% (engram) / ~37% (cross-repo) of corrections have
   **no memory to recall**; capture-on-correction is the only lever that moves it. Gate writes behind
   correction-confidence to survive the **29% false-positive** rate.
4. **Free binary re-rank — P6.** No new fires, fixes a corpus-independent ranking bug (#655); de-emphasized
   vs the original since the cluster that exposed it is small cross-repo, but still free and correct.
5. **The recall-cost enabler, then the one recall trigger — P7 then P10.** P7 (#657 two-speed) must precede
   P10; P10 (eval-design) is now **PARK / engram-specific** — the original headline trigger, demoted to ~2%
   cross-repo value.
6. **P9 stays PARK** — per-project static rules are the right home for the project-specific 37%, but they
   are not recall triggers.

**Bottom line (the correction):** the original doc concluded "the high-leverage moves are free surfaces +
re-rank + a narrow recall trigger." The cross-repo data sharpens and re-points that: **the high-leverage
moves are ENFORCEMENT GATES for verify-don't-guess, propose-before-build, and step-boundary discipline —
rules that already exist and are simply not applied.** "Fire recall more" is an **even smaller** lever than
the engram analysis implied (its headline trigger addresses ~2% of real corrections), and a ~28–37% slice
has **no memory to recall at all**. Recall timing was never the dominant problem; **discipline enforcement
is.**
