# Plan: #733 — GH comment + OpenSpec proposal for recall's chunk-only-cluster read gap

Cycle: `/please`, 2026-08-26. **Two units**, separately committed.

## Verbatim ask

Joe, this session: *"Ok. Update the issue with the finding and propose a fix"* — following a
live investigation (same session) that traced #733's measured C5b honoring rate (~80%, not the
originally-verified 100%) to its actual root cause by reading the preserved trial transcripts.

## Measured starting state

Verified live against this working tree and the live GitHub issue before writing this plan:

- `gh issue view 733` — issue is OPEN, framed as an open statistical question ("was the original
  100% a small-n artifact, or did something regress?") with acceptance criteria calling for
  either (a) revising the verified bar, or (b) naming a genuine cause and fixing it.
- The 5 trial workdirs + full JSONL transcripts from the actual `gate.py --tier full` run that
  produced #733's numbers are still on disk: `$TMPDIR/gate-C5-6l_mjvl1/` (workdirs under
  `ws/warm-{0,1,2,3,4}-*`, transcripts under `warm-cfg/projects/*/*.jsonl`).
- Read directly off those transcripts this session: in all 4 PASSING trials (idx 0,1,2,4), the
  agent called `engram show-chunk` on the matched `R-decision.md` chunk before its synthesis and
  explicitly recognized it as a load-bearing, topic-independent convention. In the 1 FAILING
  trial (idx=3), the agent made **zero** `show-chunk` calls; its own synthesis text: *"Query
  surfaced 2 matched chunks (0 notes; the rest were recent-activity items)... The matched chunks
  — an 'inline annotation token decision,' a recruiting-status update — ... are all unrelated
  seed content. Memory holds no Go conventions, no project-specific standards."* — judged from
  the chunk's path/title alone (raw query payload: `budget.items_with_full_content: 0`, chunk
  items never carry content under `--lazy-chunks`). This directly matches `recall/SKILL.md` Step
  2.5A's own documented anti-pattern ("judged coverage before reading the candidate content"),
  occurring on a **zero-note cluster (chunks-only membership)** specifically.
- R matched via `provenances: [direct]` (cosine 0.348), not the recency channel the harness's own
  docstring assumes it's isolated to — confirmed against vault note 295, a previously-documented
  property (recall's own query angles can cosine-match a "topically distant" planted item), not a
  new anomaly.
- A subagent this session traced the cluster's `is_representative` assignment (the lower-scoring
  co-matched chunk was flagged representative, not R) to `internal/cli/query.go:1524-1540`
  (`pickRepresentative` — nearest to k-means centroid, independent of raw query score by design,
  confirmed against `docs/GLOSSARY.md:397-398`). Working as designed; the failing trial's own
  reasoning never referenced "representative" either. Not a contributing factor, not in scope.
- Already crystallized this session: vault note
  `794.2026-08-26.c5b-honoring-miss-is-skipped-show-chunk-not-competing-convention.md` (the
  root-cause finding, refuting an earlier working hypothesis about a competing Go-idiom
  trade-off).
- `openspec list --json` returned no active changes before this cycle.
- `openspec/specs/recall-glance-deep-dial/spec.md` establishes precedent: this repo already specs
  `agent-instructions/skills/recall/SKILL.md`'s own agent-facing procedure as capability
  requirements (not just binary/CLI payload shape) — e.g. "Glance SHALL execute Steps 0–3.5...".
  `openspec/specs/recall-payload-cuts/spec.md` already documents the `show-chunk` mechanism
  ("the agent fetches evidence on-demand via `engram show-chunk`") without yet specifying WHEN
  that fetch is mandatory.
- Vault note `495` (Joe, direct quote): the default remedy for an engram behavioral gap is a
  vault note (rung 1, user-actionable immediately); a skill/repo change (rung 2) is "a last
  resort." Checked whether rung 1 applies here and concluded no, structurally: recall's Step 2.5
  only consults notes matching the CURRENT task's own query phrases — it has no step that queries
  "am I executing Step 2.5 correctly," so a note about correctly running Step 2.5 has no reliable
  trigger surface inside Step 2.5's own execution. The gap is in the procedure's own text; the fix
  has to be too. (Same reasoning defeats reusing note 794 itself as the fix — its `situation:`
  field is phrased for a diagnostic moment, not the live Step 2.5 moment.)
- Vault note `26`: skill-text behavior changes require `superpowers:writing-skills` TDD (baseline
  RED, edit GREEN, pressure tests). The actual `SKILL.md` edit is therefore out of scope for this
  cycle — this plan produces the OpenSpec proposal/design/specs/tasks artifacts only.
- Vault note `198`: prose/wording mechanisms in skill text measured, in a prior unrelated case, to
  plateau **below ~95% per-trial adherence** even after 3 rounds of wording iteration — good for
  raising a compliance rate, not for guaranteeing every-trial (100%) compliance. `gate_verdict.py`
  requires every valid trial in a run to pass for GREEN. This bears directly on whether a
  wording-only Step 2.5A fix should be expected to make `gate.py --tier full`'s C5 axis reliably
  GREEN — named explicitly in design.md rather than assumed away.
- Vault note `755`: `openspec validate`'s target is a **positional** argument, not `--change`:
  `openspec validate <name> --strict`.
- `openspec new change recall-chunk-only-cluster-read-gate` already run this session — the change
  directory exists; `proposal.md`/`design.md`/`specs/`/`tasks.md` are all still to be written.

## Unit 1: OpenSpec change `recall-chunk-only-cluster-read-gate`

**Files:**
- Create: `openspec/changes/recall-chunk-only-cluster-read-gate/proposal.md`
- Create: `openspec/changes/recall-chunk-only-cluster-read-gate/design.md`
- Create: `openspec/changes/recall-chunk-only-cluster-read-gate/specs/recall-payload-cuts/spec.md`
- Create: `openspec/changes/recall-chunk-only-cluster-read-gate/tasks.md`

- [ ] **Step 1: Write `proposal.md`** (exact content below — per `openspec instructions proposal`'s
  Why / What Changes / Capabilities / Impact template)

```markdown
## Why

Issue #733 measured C5b (honoring a recency-channel standard) at ~80% (8/10) across two
independent full-tier runs, against an originally-verified 100% bar. Root-cause investigation
of the one preserved failing trial (session 2026-08-26, `gate-C5-6l_mjvl1/ws/warm-3-*`) found
the miss is not a capability ceiling: the agent's recall query matched the load-bearing chunk
into a cluster with zero note candidates, made zero `engram show-chunk` calls, and judged the
cluster's relevance from the chunk's title alone — the recall skill's own documented
"judged coverage before reading the candidate content" anti-pattern, occurring specifically
on a zero-note cluster (chunks-only membership). `recall/SKILL.md` Step 2.5A already requires reading
chunk content before judging, but the wording is easy to read as "nothing to write ⇒ nothing
to read" when a cluster carries no notes. This change specs the requirement precisely enough
to close that reading.

## What Changes

- Adds a requirement to the `recall-payload-cuts` capability: the agent SHALL fetch every
  chunk member's content via `engram show-chunk` before judging a cluster's relevance,
  including (and especially) clusters with zero note candidates.
- Does **not** modify `agent-instructions/skills/recall/SKILL.md` in this change — that edit
  requires `superpowers:writing-skills` TDD (baseline RED / edit GREEN / pressure tests, vault
  note 26) and is deferred to a follow-up implementation cycle, tracked in this change's
  `tasks.md`.
- Does **not** modify `dev/eval/traps/gate_verdict.py`'s C5 exact-bar pass criterion in this
  change — flagged as an explicit open question in `design.md`, to be decided from the
  post-fix measured rate, not guessed now.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `recall-payload-cuts`: adds a requirement that the agent must read (`engram show-chunk`)
  every chunk member of a cluster before judging its relevance/coverage, including clusters
  with zero note candidates — closing the gap root-caused in issue #733.

## Impact

- `agent-instructions/skills/recall/SKILL.md` (Step 2.5A) — future implementation target, not
  touched by this change.
- `openspec/specs/recall-payload-cuts/spec.md` — gains the new requirement once this change is
  archived.
- `dev/eval/traps/gate_verdict.py` (C5 axis verdict logic) — open question for the follow-up
  cycle; see `design.md`.
- GitHub issue #733 — this change is the "decision recorded" artifact its acceptance criteria
  call for.
```

- [ ] **Step 2: Write `design.md`** (exact content below — per `openspec instructions design`'s
  Context / Goals-Non-Goals / Decisions / Risks-Trade-offs / Migration Plan / Open Questions
  template)

```markdown
## Context

Issue #733: C5b (honoring a recency-channel standard under conflict) measured ~80% (8/10)
across two independent full-tier runs, against an originally-verified 100%. Root-cause
investigation (session 2026-08-26) read the preserved trial transcripts from the actual run
(`gate-C5-6l_mjvl1/`) and found the one failing trial (idx=3) never called `engram show-chunk`
on the load-bearing matched chunk, unlike all 4 passing trials — a recall-procedure compliance
gap, not a model capability ceiling. Full evidence: vault note 794 and this session's #733 GH
comment.

This repo already specs `agent-instructions/skills/recall/SKILL.md`'s own agent-facing
procedure as capability requirements — see `openspec/specs/recall-glance-deep-dial/spec.md`
("Glance SHALL execute Steps 0–3.5..."). That precedent makes a capability spec the right
mechanism for capturing this fix's target behavior, not just a SKILL.md diff description.

## Goals / Non-Goals

**Goals:**
- Specify the exact agent-behavior requirement precisely enough that a future
  `writing-skills` TDD cycle can implement it against a concrete RED baseline (the failing
  trial's own transcript).
- Name explicitly why this is a repo/skill change (rung 2, vault note 495's ladder) and not a
  vault-note fix (rung 1) — not skip that reasoning silently.
- Name the ceiling on what a wording-only fix can guarantee (vault note 198), so the eventual
  `gate_verdict.py` decision is made with that constraint in view, not blind to it.

**Non-Goals:**
- Editing `agent-instructions/skills/recall/SKILL.md` — deferred to a follow-up cycle (vault
  note 26: skill-text behavior changes require `writing-skills` TDD).
- Changing `gate_verdict.py`'s C5 pass criterion — deferred; needs its own decision informed by
  a real post-fix measured rate (see Open Questions).
- Investigating or fixing cluster representative selection (`internal/cli/query.go`,
  `pickRepresentative`) — verified this session (subagent trace to
  `internal/cli/query.go:1524-1540`) as working as designed: representative = nearest to the
  cluster's k-means centroid, independent of raw query-match score by design, confirmed
  against `docs/GLOSSARY.md:397-398`. Not implicated in the miss (the failing trial's own
  reasoning never referenced "representative"). No action needed.

## Decisions

### Decision: rung-2 (repo/skill change), not rung-1 (vault note)

Vault note 495 (Joe's own design direction) sets a two-rung default: prefer a vault note
(immediately user-actionable) over a repo/skill change ("a last resort"). Considered: could a
vault note stating "read every chunk before judging, even at 0 notes" close this gap on its
own? No, structurally — recall's Step 2.5 only consults notes/chunks matching the CURRENT
task's own query phrases; it has no step that queries "am I executing Step 2.5 correctly." A
note about correctly running Step 2.5 has no reliable trigger surface inside Step 2.5's own
execution; it would only resurface by coincidence (explore-sampling near some unrelated future
query), not reliably at the moment it's needed. The gap is in the procedure's own text, so the
fix has to be too.

Alternative considered: rely on vault note 794 alone (already crystallized this session), no
repo change. Rejected for the identical structural reason — 794's `situation:` field is
phrased for a diagnostic moment ("diagnosing why a recall-based agent failed to honor..."), not
for the live Step 2.5 moment, so it will not surface as a Step 2.5 candidate during a future
live recall run either.

### Decision: spec the requirement now; implement the SKILL.md text in a follow-up cycle

`writing-skills` TDD (note 26) requires a baseline (RED) run showing the failure mode on the
OLD text before editing. That baseline already exists, and is stronger than a synthetic one:
the actual failing trial's transcript (`gate-C5-6l_mjvl1/warm-cfg/projects/.../warm-3-*/*.jsonl`)
is a real RED artifact — cite its exact turns rather than re-deriving a synthetic repro. This
change's job is to (a) spec-record the target behavior and (b) hand the follow-up cycle a
concrete `tasks.md` plus a pointer to the RED evidence — not to run that TDD cycle itself,
since a SKILL.md edit is its own gated, test-driven unit.

### Decision: name the ~95% ceiling explicitly; do not silently assume the wording fix "solves" C5

Vault note 198 measured (a different skill-text mechanism, 3 wording iterations): prose/wording
mechanisms in skill text plateau below ~95% per-trial adherence — good for raising a rate,
wrong tool for guaranteeing every-trial compliance. `gate_verdict.py`'s current C5 verdict
(`axis_verdict`) requires every valid trial in a run to pass for GREEN — at a generous 95% true
compliance rate, a 5-trial `--tier full` run has only ~77% (0.95^5) chance of showing all-GREEN;
at 90% only ~59%. A wording-only fix, even one that genuinely raises true compliance well above
today's measured 80%, should be EXPECTED to still occasionally show C5 as RED with zero real
regression. Naming this now so it isn't rediscovered as a fresh mystery later; the actual
disposition (accept the residual flakiness under the existing same-tree-recheck discipline —
notes 209/793 — vs. move `gate_verdict.py`'s C5 criterion to a threshold test) is left as an
Open Question, decided from the ACTUAL post-fix rate once measured.

## Risks / Trade-offs

- [Risk] The eventual SKILL.md wording fix may not fully close the gap (per the ~95% ceiling
  above) → [Mitigation] The spec requirement + RED transcript give the follow-up cycle a
  concrete target and baseline; measure the post-fix rate with `c5.py --arms warm` before
  declaring the fix sufficient, and revisit `gate_verdict.py`'s C5 threshold based on that
  measurement rather than an assumption that 100% is achievable.
- [Risk] Strengthening Step 2.5A's wording could add verbosity/cost to every recall
  invocation, not only chunks-only-cluster ones → [Mitigation] Scope the wording addition to
  the chunks-only-cluster branch specifically (already a distinct sub-case in the existing
  text), not a blanket rewrite of Step 2.5A.
- [Risk] This requirement could go stale if a future, unrelated recall refactor removes the
  `show-chunk` mechanism (`--lazy-chunks`) entirely → [Mitigation] The requirement is written
  against `recall-payload-cuts`, the capability that already owns `show-chunk`'s existence; if
  that capability is retired, this requirement retires with it via the normal spec-sync
  process.

## Migration Plan

N/A — this change is spec-only; no code or skill edits ship in this cycle, so there is nothing
to deploy or roll back.

## Open Questions

1. Should `gate_verdict.py`'s C5 axis verdict move from an exact-bar
   (`passed == len(valid)`) to a threshold-based test (e.g. a confidence bound against a
   target rate) once the post-fix true compliance rate is measured? Decide in the follow-up
   implementation cycle, with real post-fix data — not now, on a guess.
2. Does the eventual SKILL.md wording fix belong only in Step 2.5A's existing chunk-reading
   sentence, or does the zero-note branch need its own more visually distinct callout? (The
   failing trial's exact words — "0 notes; crystallized 0 lessons" — suggest the current
   text's flow makes "no notes" read as a stopping point.) Leave to the `writing-skills` TDD
   cycle to decide by pressure-testing candidate wordings.
```

- [ ] **Step 3: Write the specs delta** at
  `openspec/changes/recall-chunk-only-cluster-read-gate/specs/recall-payload-cuts/spec.md`
  (exact content below — `## ADDED Requirements`, per `openspec instructions specs`)

```markdown
## ADDED Requirements

### Requirement: Agent reads chunk content before judging cluster relevance, even in zero-note clusters

When Step 2.5 of the recall procedure (`agent-instructions/skills/recall/SKILL.md`) processes
a cluster, the agent SHALL fetch every chunk member's content via `engram show-chunk` before
stating any relevance or coverage judgment about that cluster — including clusters whose
`candidate_l2s` list is empty (no note candidates, chunks-only membership). The absence of a
note candidate to write does not exempt a cluster's chunk members from the read-before-judge
requirement.

#### Scenario: Zero-note cluster still gets its chunks read
- **WHEN** Step 2.5 processes a cluster whose `candidate_l2s` list is empty
- **THEN** the agent invokes `engram show-chunk` on every chunk member of that cluster before
  stating any relevance or coverage judgment about it

#### Scenario: Metadata-only judgment is a violation
- **WHEN** a chunk member's `path` or anchor text (title) alone — without its fetched content
  — is used as the basis for judging it "unrelated," "not applicable," or otherwise irrelevant
- **THEN** the judgment fails this requirement, regardless of how many other members of the
  same cluster were genuinely read and correctly judged

#### Scenario: A load-bearing chunk surfaces in a zero-note cluster
- **WHEN** a chunk carrying a hard-requirement convention (e.g. a recency-channel standard
  planted topically distant from the task) matches into a cluster with no note candidates,
  alongside an unrelated distractor chunk
- **THEN** reading both chunks' content (not just their titles) is required before either is
  judged relevant or irrelevant — this is the exact configuration in which #733's C5b honoring
  miss occurred (`dev/eval/traps/c5.py`, trial idx=3, `gate-C5-6l_mjvl1`)
```

- [ ] **Step 4: Write `tasks.md`** (exact content below — follow-up implementation work; NOT
  executed by this cycle)

```markdown
## 1. Recall skill wording fix (writing-skills TDD — required, no exceptions per vault note 26)

- [ ] 1.1 Baseline (RED): cite the preserved failing-trial transcript
      (`gate-C5-6l_mjvl1/warm-cfg/projects/.../warm-3-*/*.jsonl`, or re-run
      `dev/eval/traps/seed_c5.py && dev/eval/traps/c5.py --arms warm --n 5` fresh if the
      original artifacts have been cleaned up) as RED evidence — the agent's own synthesis text
      ("0 notes; crystallized 0 lessons... unrelated seed content") demonstrates the failure
      mode on the CURRENT Step 2.5A wording.
- [ ] 1.2 Invoke `superpowers:writing-skills` to edit `agent-instructions/skills/recall/SKILL.md`
      Step 2.5A: strengthen the read-before-judge instruction specifically for the zero-note,
      chunks-only cluster case (see this change's `design.md` Open Question 2).
- [ ] 1.3 GREEN: re-run `dev/eval/traps/seed_c5.py && dev/eval/traps/c5.py --arms warm --n 5`
      against the new wording; confirm the failure mode from 1.1 no longer reproduces.
- [ ] 1.4 Pressure-test the new wording against fresh rationalization loopholes (writing-skills
      TDD requirement) — e.g. a cluster with 0 notes AND 0 load-bearing chunks, to confirm the
      agent doesn't over-correct into reading chunks it has no real need to fetch.
- [ ] 1.5 Sync the fixed skill text to deployable form (`engram update --with-guidance` or this
      repo's current equivalent); confirm via a diff against
      `agent-instructions/skills/recall/SKILL.md`.

## 2. Re-measure and decide the gate threshold

- [ ] 2.1 Run a larger-n measurement (`dev/eval/traps/c5.py --arms warm --n 15` or more) against
      the FIXED wording for a real post-fix compliance rate with a narrower confidence interval
      than n=5.
- [ ] 2.2 Using the rate from 2.1, decide `design.md` Open Question 1: keep
      `dev/eval/traps/gate_verdict.py`'s exact-bar C5 verdict (if the fix reliably clears it),
      or move to a threshold-based test (if it doesn't) — record the decision and rationale in
      `dev/eval/LEDGER.md`.
- [ ] 2.3 If the gate criterion changes, update `dev/eval/traps/gate_verdict.py`
      (`_norm_c5` / `axis_verdict`) and its unit tests in `test_gate.py` accordingly.
- [ ] 2.4 Using the rate from 2.1, update `docs/GLOSSARY.md`'s lazy-chunks entry (lines 372-379)
      and both architecture diagram comments (`docs/architecture/c2-containers.md:126`,
      `docs/architecture/c1-system-context.md:165-166`) to add the zero-note-cluster
      mandatory-read case explicitly and re-state the fetch-frequency figure against the
      post-fix measurement rather than the pre-fix "0/13".

## 3. Close out

- [ ] 3.1 Sync specs (`/opsx:sync` or `openspec` equivalent) to merge the ADDED requirement into
      `openspec/specs/recall-payload-cuts/spec.md`.
- [ ] 3.2 Update `dev/eval/traps/README.md`'s C5 bar description and `dev/eval/LEDGER.md`'s
      `733-c5b-honoring-rate` row with the resolution (per #733's own acceptance criteria).
- [ ] 3.3 Close GitHub issue #733, referencing the merged spec, the SKILL.md commit, and the
      final measured rate.
- [ ] 3.4 Archive this change (`openspec archive recall-chunk-only-cluster-read-gate`).
```

- [ ] **Step 5: Validate**

```bash
openspec validate recall-chunk-only-cluster-read-gate --strict
```

Expected: PASS (positional target argument — vault note 755; NOT `--change`).

- [ ] **Step 6: Commit**

```bash
git add openspec/changes/recall-chunk-only-cluster-read-gate/ docs/superpowers/plans/2026-08-26-733-recall-chunk-only-cluster-read-gate.md
git commit -m "docs(openspec): propose recall-chunk-only-cluster-read-gate for #733

AI-Used: [claude]"
```

## Unit 2: GH comment on #733

**Files:** none (GitHub, not repo state) — depends on Unit 1 being committed (and, per the
orchestrator's plan, pushed) so the comment's reference to the OpenSpec change resolves to real
content.

- [ ] **Step 1: Draft the comment body** (exact content below)

```markdown
## Root cause found: recall-procedure compliance gap, not a capability ceiling or measurement artifact

Traced this to the actual trial data. The 5 workdirs + full JSONL transcripts from the
`gate.py --tier full` run that produced this issue's numbers were still on disk
(`$TMPDIR/gate-C5-6l_mjvl1/`), so this isn't inference from aggregate stats — it's read
directly off the failing trial's own transcript.

**C5a (retrieval) is clean in all 5 trials, including the miss.** `R-decision.md` (the
recency-channel decision this trap plants — a team convention that every new inline code
comment must open with the marker token `ZÖRBAX`) matched Channel 1 (recall's relevance-ranked
match channel, tagged `provenances: [direct]`) in every trial — this is the harness's own
`surfaced=true` result for all 5, not just the miss — ruling out a retrieval problem and
confirming this issue's own framing that C5a and C5b are separate mechanisms. (One correction
to the harness's own docstring, freshly confirmed this session on the failing trial's raw
query payload: R matched via **direct/cosine** (score 0.348), not the recency channel it's
designed to be isolated to — the agent's own "comment style above Go function" query phrase
naturally cosine-matches R's content. The general mechanism behind this — recall's query
angles can cosine-match a "topically distant" planted item — is previously documented (engram
vault note 295), so this isn't a new anomaly; doesn't change the diagnosis below.)

**The miss (trial idx=3) is a located, mechanical compliance gap:**

| | `show-chunk` calls on R | synthesis | comment emitted |
|---|---|---|---|
| warm-0,1,2,4 (honored) | 1 each | recognized R as a load-bearing, topic-independent convention | `// ZÖRBAX Timestamp returns...` |
| warm-3 (miss) | **0** | *"Query surfaced 2 matched chunks (0 notes...)... unrelated seed content... Memory holds no Go conventions"* | `// Timestamp returns...` (idiomatic Go, no marker) |

R matched into a 2-member cluster with zero note candidates. The pattern above is exactly the
recall skill's own documented anti-pattern ("judged coverage before reading the candidate
content"), occurring specifically on a cluster with no note candidates — the failing trial's
own synthesis text reads as if it treated "0 notes to write" as license to skip reading the
chunks at all.

This means:
- On this issue's own dichotomy (small-n artifact of a real ~80% rate, vs. something that
  regressed since 2026-06-26/06-30): this finding is a **mechanistic explanation consistent
  with the first branch**, not a third option. Nothing here identifies a change over time — no
  wording/model/payload diff between the original verification and now — so there's no positive
  evidence for "something regressed." What it does show is *why* a true rate near 80% is
  plausible: a specific, reproducible procedural miss (below) that fires on a specific
  cluster-composition pattern. Treat this as a strong lean toward "small-n artifact," not a
  closed case — this session didn't bisect history to rule out a regression outright.
- It's **not** the model trading off the marker against Go's idiomatic doc-comment convention
  — the failing trial never engaged with R's content to make that trade-off at all.
- **Acceptance criterion 1** (a larger-n, n=15-20 measurement to narrow the confidence interval
  on the *current, unfixed* rate) is not run this cycle. With the mechanism now identified,
  spending more n against the *old* wording measures a number this fix intends to change —
  lower value than fixing first. Repurposed as `tasks.md` task 2.1: measure the *post-fix* rate
  at n=15-20 once the follow-up cycle lands the wording change, which is what actually resolves
  ACs 2-4.

**Side question, checked and cleared:** the failing trial's cluster marked the *lower-scoring*
co-matched chunk (`hiring-update.md`, 0.273) as `is_representative`, not R (0.348). Traced to
`internal/cli/query.go:1524-1540` — representative selection is by **centroid proximity**, not
query-match score, confirmed working as designed against `docs/GLOSSARY.md`. Not a bug, and
the failing trial's own reasoning never referenced "representative" anyway — not a
contributing factor.

**Fix proposed:** [`openspec/changes/recall-chunk-only-cluster-read-gate`](https://github.com/toejough/engram/tree/main/openspec/changes/recall-chunk-only-cluster-read-gate) specs an explicit
requirement that the agent must read every chunk member via `show-chunk` before judging a
cluster's relevance, even (especially) when the cluster carries zero note candidates. The
actual `SKILL.md` wording edit is deferred to a follow-up implementation cycle (requires
`writing-skills` TDD against this trial's transcript as the RED baseline — see the change's
`tasks.md`). The design doc also flags that a wording-only fix is expected to *raise*, not
*guarantee*, compliance (engram vault note 198: a prior, unrelated skill-text mechanism
measured to plateau below ~95% per-trial adherence after 3 rounds of wording iteration), which
has a direct implication for `gate_verdict.py`'s exact-bar C5
verdict — left as an explicit open question for the follow-up cycle, to be decided from the
actual post-fix measured rate.

Crystallized as vault note 794 for future diagnosis of similar C5-style misses.

Updating, not closing — the acceptance criteria's fix isn't implemented yet.
```

- [ ] **Step 2: Fact-check the draft** — before posting, re-verify every quoted line and
  file:line citation above against the raw evidence already gathered this session (the
  `gate-C5-6l_mjvl1` transcripts, the `internal/cli/query.go` citations, vault note 794's text).
  This is the non-code analogue of "run the tests before declaring done" for this unit.

- [ ] **Step 3: Post**

```bash
gh issue comment 733 --body-file <tmpfile with Step 1's content>
```

Do NOT close the issue.

## Self-Review

**Spec coverage.** The ask has two halves: *"update the issue"* → Unit 2. *"propose a fix"* →
Unit 1. Both covered, nothing beyond them.

**Placeholder scan.** None — `proposal.md`, `design.md`, the specs delta, `tasks.md`, and the GH
comment are all drafted in full above, not deferred to execution-time invention.

**Type/name consistency.** Change name `recall-chunk-only-cluster-read-gate` used identically
across `openspec new change`, the plan, `validate`, and the commit message. Capability name
`recall-payload-cuts` matches the existing spec folder exactly (`openspec/specs/recall-payload-cuts/`).

**Scope check.** `agent-instructions/skills/recall/SKILL.md` is NOT edited this cycle (vault
note 26 — requires `writing-skills` TDD, correctly deferred to `tasks.md`). Cluster
representative selection is resolved as working-as-designed and excluded from scope (subagent
finding this session). `gate_verdict.py` is NOT edited now — named as an explicit open question
in `design.md` and a follow-up task, not silently assumed solved by the wording fix (vault note
198). Nothing here reaches into #732 or any other issue's scope.

**Push note.** This repo's `main` tracks `origin/main` with no divergence as of this session.
Unit 1's commit will be pushed before Unit 2's GH comment is posted, so the comment's reference
to the OpenSpec change resolves to real, visible content rather than a dead link.

**Doc-surface enumeration grep (please Step 3, non-waivable) — RUN, not N/A.** Gate A's
docs/diagrams-alignment reviewer correctly rejected an initial "N/A" judgement on this section;
the actual grep (`grep -rn "show-chunk" docs/ openspec/specs/`) surfaces three locations that
describe the `show-chunk` fetch mechanism's frequency, which this change's new requirement
elaborates on:

| location | current text | disposition |
|---|---|---|
| `docs/GLOSSARY.md:372-379` (lazy-chunks entry) | "chunks are supplementary, notes load-bearing — measured 0 chunk fetches in 13/13 realistic recalls, with on-target fetch when a chunk is the sole carrier of a needed fact" | UPDATE once the fix ships — add the zero-note-cluster mandatory-read sub-case explicitly; re-check whether the 0/13 figure still holds once real-world zero-note-cluster rates are known |
| `docs/architecture/c2-containers.md:126` (sequence diagram comment) | `opt a chunk is the sole carrier of a needed fact (rare — notes are load-bearing; measured 0 fetches in 13/13 realistic recalls, on-target fetch when sole-source)` | UPDATE — same reason as above |
| `docs/architecture/c1-system-context.md:165-166` (sequence diagram comment) | `opt a needed fact lives only in a chunk (rare — notes are load-bearing)` | UPDATE — same reason as above |

None of these three are logically contradicted by the new requirement — "sole carrier of a needed
fact" is exactly the judgment a zero-note cluster's chunk requires, and the new requirement only
makes explicit that this judgment cannot be made from a title alone. But all three currently omit
the zero-note-cluster case and cite a pre-fix "0/13" figure that this change's own fix could move.
Tracked as `tasks.md` task 2.4 (added below) rather than done in this cycle, since it depends on
the post-fix measured rate from task 2.1 — updating the docs before that data exists would be
guessing the number, not reporting it. All other `show-chunk` matches (subcommand-reference
mentions, `--parent` routing in `vault-merged-recall`, `serve` API exposure) describe unrelated
mechanics and need no change.
