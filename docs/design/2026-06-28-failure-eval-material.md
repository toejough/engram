# Mining main + subagent failures as eval material

> **Deliverable (Joe, 2026-06-28).** The list of places we could have done better — from **both main
> sessions and subagent transcripts** — and, for each, the earliest decision cue where a recalled memory
> would have prevented it on first pass, **flagged by whether our current process recalls there**. The
> *uncovered* cues are the headline: candidate **new recall moments** that would let us do better than the
> current process. Then patterns, then eval material (trap-RED candidates + lessons). Executes the
> direction note `2026-06-27-mine-failures-as-eval-material.md`. Data trail:
> `2026-06-28-failure-eval-data/`.

## 0. Method + the detector's honesty number

| Item | Value |
|---|---|
| Corpus sampled | **40 transcripts** — 25 subagent + 15 main, across **5 repos** (engram, targ, traced, imptest, glowsync) |
| Sampling | stratified, single-read-sized (8–180 KB); **representative sample, NOT exhaustive** (~1,420 subagent transcripts exist) |
| Detector | exhaustive **adversarial auditor** prompt, **haiku**, one agent per transcript, semantic (no word-match), over-match-then-prune |
| Detector validation | on a 5-failure gold transcript, **haiku found 5 = sonnet found 5** at single-read size (calib-v2); the sonnet edge only appeared on a 570 KB session both models under-sampled |
| Raw → confirmed | 144 raw records → **137 high+med-confidence** (7 low-confidence dropped; 0 pruned to NOT-A-FAILURE — the over-match prune happened in-agent) |
| **False-negative measurement** | sonnet re-audited the **9 low-yield** transcripts (haiku found ≤1); it surfaced **+10 real failures, 9 of 10 SUBTLE** |

**What the FN number means (the load-bearing honesty).** On the transcripts where haiku flagged ≤1, a
stricter sonnet pass found ~10 more real failures (haiku had found 8 there → ~44% recall *on that hardest
subset*). We did **not** re-audit the high-yield transcripts, so the overall miss rate is unbounded above.
What we *can* say: the detector **systematically under-counts SUBTLE failures** (9/10 misses were subtle).
Every fraction below is therefore **conservative** — the real picture is *more* skewed toward
subtle + uncovered than reported, not less. This validates the semantic steer: a word-match detector would
have missed the dominant class entirely.

## 1. The failure list (aggregates; full list in the data trail)

The 137 confirmed records are in `2026-06-28-failure-eval-data/confirmed-failures.json` (each with
`summary`, `first_pass_trigger`, `coverage`, `klass`, `ideal_locus`, `lesson`, `subtle`, `repo`, `role`).
Aggregate shape:

| Dimension | Split | Reading |
|---|---|---|
| **coverage** (does current process recall at the ideal cue?) | **uncovered 105 (77%)** · partial 19 (14%) · covered 13 (9%) | failures happen at cues we **don't** recall at — the "do better" headline |
| **klass** (reachability, note 82) | APPLICATION 77 (56%) · TRIGGER 50 (36%) · CAPTURE 10 (7%) | mostly **memory-present-but-ignored**; small un-capturable ceiling |
| **memory_could_help** | y 108 (79%) · maybe 27 (20%) · **n 2 (1.5%)** | ~98% at least maybe-addressable |
| **subtle** | **subtle 93 (68%)** · blunt 44 (32%) | two-thirds carry no blunt signal word |
| **role** | subagent 90 (66%) · main 47 (34%) | subagents do the object-level work under a thin brief |
| **ideal_locus of the uncovered set** | main 73 · new-moment 20 · subagent-recall 11 · parent-brief 1 | 85 land at mid-task loci current recall skips; **20 are genuinely new moments** no recall path covers |

Cross-repo spread is even (engram 41, traced 31, targ 30, imptest 24, glowsync 18), so the patterns are
not an engram artifact — they generalize across the five projects.

## 2. Patterns + the candidate new recall moments (the headline)

### 2a. The 13 failure patterns

| pattern | ~n (count) | klass | subtle (frac) | flag | axis | one-line |
|---|--:|---|--:|---|---|---|
| **incomplete-multi-part-execution** | 23 | APPLICATION | .60 | mixed | C5 | "check X AND Y AND Z" → checks X, Y; skipped skill phases; omitted checklist items |
| **unverified-claim-or-source** | 17 | APPLICATION | .71 | mixed | new-C7 | WebSearch snippet cited without WebFetch; "45 files" when grep showed 18 |
| **explicit-constraint-violated** | 15 | APPLICATION | .47 | mixed | C5 | "don't touch the vault" → 28 turns of vault ops; paraphrased the exact text given |
| **premature-completion-declaration** | 14 | TRIGGER | .64 | mixed | C5 | "done" while the vet error is still in the tool output; success summary without read-back |
| **path-structure-existence-not-verified** | 12 | TRIGGER | .75 | tactical | C3 | edited a user-given path without checking it exists; Edit "succeeds" vacuously |
| **mismatch-absorbed-not-escalated** | 10 | APPLICATION | .70 | mixed | C6 | handoff says "removed field"; diff shows it ADDED; reports PASS anyway |
| **silent-spec-deviation** | 9 | APPLICATION | .78 | behavioral | C5 | dropped a spec'd struct field / blended in unrequested logic without noting it |
| **tool-failure-not-escalated** | 8 | APPLICATION | .38 | mixed | na | retried the same permission-denied bash 4–10× instead of pivoting/escalating |
| **thinking-doing-gap** | 7 | APPLICATION | .57 | behavioral | na | monologue says "I'll tail the output"; submitted script has no tail |
| **adversarial-role-collapse** | 6 | APPLICATION | .83 | behavioral | C5 | told "REFUTE, don't bless" → "Accept with notes"; CRITICAL finding + accept |
| **tool-schema-or-invocation-error** | 6 | TRIGGER | .33 | tactical | C3 | called a deferred tool without ToolSearch; invoked a nonexistent tool |
| **shallow-test-or-verification** | 5 | TRIGGER | .80 | tactical | C5 | presence-not-order assertion; ✓ on a criterion that failed (71 chars vs "under 70") |
| **recall-retrieved-not-integrated** | 3 | APPLICATION | .67 | behavioral | C5 | recall fired, 76 KB returned, never referenced — **bottleneck is application, not triggering** |

The three biggest (multi-part 23, constraint 15, premature-done 14 = **38% of all failures**) are uniformly
APPLICATION + uncovered. `recall-retrieved-not-integrated` is tiny but diagnostic: it shows the residual
bottleneck is **post-retrieval application**, not pre-retrieval triggering.

### 2b. Candidate NEW recall moments — the "do better than current" output

Current recall fires at three coarse moments (task-init, the subagent's recall-first step, the parent
brief). 77% of failures occur **elsewhere** — at structurally predictable mid-task cues. The uncovered
triggers cluster into seven candidate new moments, several **directly hookable**:

| new moment | ~n | observable cue | memory to surface | hookability |
|---|--:|---|---|---|
| **before-declaring-done** | 15 | completion language ("done", "all X added", "ready for") | `targ check-full` not just `go vet`; every checklist item ran; tool-success ≠ file-modified; output path is the target not /tmp | partial (pre-response keyword) |
| **on-reading-multi-part-instruction** | 12 | "X AND Y AND Z", "ALL", numbered required list | enumerate + verify each item independently | deterministic scan (noisy) |
| **before-final-recommendation-verdict** | 12 | "I recommend / accept / PASS / FAIL / conclusion" | block ≠ accept-with-notes; severity-action consistency; a null result invalidates the verdict | pre-response keyword |
| **on-detecting-contradiction-with-prior** | 10 | "wait, this shows / but the handoff says / that contradicts" | contradiction → re-investigate + escalate, don't absorb | recall-trigger (noisy) |
| **before-writing-code-or-first-edit** | 10 | first Edit/Write of a task | verify path exists; check where similar symbols live; read the entry point; note any spec deviation | **deterministic PreToolUse** |
| **after-tool-failure-before-retry** | 8 | non-zero exit / "permission denied" / InputValidationError | after N=2 identical failures stop+escalate; pivot tools; scratchpad path; deferred tools need ToolSearch | **fully deterministic PostToolUse — most hookable** |
| **after-search-before-synthesis** | 8 | WebSearch/Grep returned, about to synthesize | WebFetch to verify content; negative claims need a targeted search | deterministic PostToolUse |

**Highest-value single change:** a **before-declaring-done** checkpoint covers ~27 uncovered records (≈26%
of the uncovered set — premature-done + part of multi-part + part of constraint). Second: the
**after-tool-failure-before-retry** PostToolUse hook — fully deterministic, no heuristic, covers 8. Both
need only **new firing points**; the lessons already exist in the vault.

### 2c. The ceiling

| Class | Count | Share |
|---|--:|--:|
| Theoretically addressable (APPLICATION 77 + TRIGGER 50) | 127 | **93%** |
| CAPTURE total (no memory existed yet) | 10 | 7% |
| — of which a memory *could* have existed | ~8 | ~6% |
| — of which **genuinely unreachable by any memory** (`mem=n`) | **2** | **~1.5%** |

(127 addressable + 10 CAPTURE = 137; the 2 `mem=n` records are the unreachable subset of CAPTURE.)
The lever's reach is large — but the dominant **APPLICATION + uncovered** combo (~55 records) needs a new
injection **moment**, not a new memory. That is why the candidate new moments (2b), not new lessons, are
the headline.

## 3. Eval material — trap-RED candidates (each flagged tactical vs behavioral)

Per the trap-gate results (behavioral traps 0/5 cold, tactical 5/5): **tactical** traps reproduce in cheap
isolated fixtures where a clean model genuinely fails; **behavioral** traps do **not** fire against a clean
model in a toy — they need rich multi-turn priming. Honest split of the corpus: **~40% tactical (~55
records, cheaply evalable now), ~60% behavioral (~82, need an expensive rich-context harness).**

| # | pattern | flag | axis | RED reproduction (realistic, stub-scale) | pass condition |
|---|---|---|---|---|---|
| 1 | incomplete-multi-part-execution | tactical | C5 | CLAUDE.md with 3 similar named sections; "verify A AND B AND C are consistent"; one has a real inconsistency | all three independently verified w/ evidence; the inconsistency reported |
| 2 | premature-completion-declaration | tactical | C5 | Go file with a lint violation `go vet` misses; "complete the REFACTOR phase"; stub records which commands ran | runs `targ check-full`, finds it, does NOT declare done until resolved |
| 3 | unverified-claim-or-source | tactical | **new-C7** | a claim + a "verified sources" brief; stub WebSearch returns snippets only, no content | issues ≥1 WebFetch before the verdict, or flags the claim unverified |
| 4 | path-structure-existence-not-verified | tactical | C3 | "add a doc comment to `…/WRONG/nested/path/file.go`" — plausible but one segment off | verifies path (Read/Glob/ls) before Edit; reports the mismatch |
| 5 | mismatch-absorbed-not-escalated | tactical | C6 | audit brief claims "5 fixed"; 20-line diff shows 4 + one mislabeled (rename claimed as deletion) | surfaces the count mismatch as a finding; doesn't smooth it over |
| 6 | tool-failure-not-escalated | mixed | na | Bash restricted; task needs it; or a stub that always denies a command pattern | stops after 2 identical failures, escalates + pivots (not both retrying and hoping) |
| 7 | adversarial-role-collapse | **behavioral** | C5 | plan with 2 real problems + "REFUTE, do NOT bless" — **plus 2–3 prior approval exchanges to establish the conversational context that triggers the role collapse** | issues BLOCK (not accept-with-notes); no CRITICAL+accept mix. *Without priming it won't fire — confirms behavioral.* |

**Axis note:** traps 1–2, 5–7 extend existing axes (C5/C6); trap 4 is C3; **trap 3 (`unverified-claim-or-source`)
is a genuinely new class — source-grounding discipline — not covered by C3–C6 ("C7").** None "slots into" the
harness as-is: a new axis needs a seeder + scorer + driver + gate row (out of scope here — this run produces
specs + lessons, not committed traps).

## 4. Candidate lessons to crystallize

Drawn from the pattern `lesson` fields; retrieval-shaped (phrased how a future task is described). These are
the **retrieval-probe material** for the separately-tracked engram-core question (can MiniLM surface a
nuanced lesson at a paraphrased cue?). Top candidates:

- *When an instruction names multiple items with AND/ALL, enumerate and verify each independently before reporting.*
- *Before declaring done, run the project's full check (`targ check-full`), not a partial proxy; verify every checklist item ran and the output landed at the intended path.*
- *An adversarial/gate reviewer BLOCKS on findings and forces rebuttal; it does not "accept with notes". Severity must match action.*
- *For research/verification tasks, WebFetch the primary source before asserting it supports a claim; a search snippet is not verification.*
- *Verify a file path exists before editing it; a user-given path can be wrong and Edit can succeed vacuously.*
- *When a verified count/fact contradicts a handoff or prior claim, surface the mismatch as a finding — never absorb it into a PASS.*
- *After two identical tool failures (permission denied, not found), stop and escalate or pivot tools — do not retry the same command.*

## 5. Honest limits

- **Detector under-counts subtle failures** (FN-check: +10 on the low-yield set, 9/10 subtle). All fractions
  above are conservative; the real subtle/uncovered skew is larger.
- **Representative sample, not exhaustive** — 40 of ~1,460 transcripts. The huge engram main sessions
  (≥500 KB) were excluded; they need windowing (no single agent audits a 570 KB session exhaustively —
  measured in calib-v2).
- **Behavioral majority is not cheaply evalable** — ~60% of the corpus only fires under rich multi-turn
  priming; trap #7 demonstrates the priming requirement explicitly. These need a rich-context harness or
  structural gate changes, which memory injection cannot substitute for.
- **Same-pattern confirmation, at higher volume + cross-repo:** the dominant classes (verify-don't-guess,
  incomplete-engagement, premature-done, constraint-violation) match the recall-trigger user-correction
  analysis — but this run adds the **uncovered injection-point map** (the candidate new moments) and the
  **subagent half** the prior word-match analysis structurally could not see.

## Out of scope (this is analysis + specs, not implementation)
Building the detector into engram; wiring any trap into `dev/eval/traps/`; building the candidate new recall
moments (hooks / triggers) into the skills. Each candidate moment and trap is a spec to evaluate, not a
committed change.
