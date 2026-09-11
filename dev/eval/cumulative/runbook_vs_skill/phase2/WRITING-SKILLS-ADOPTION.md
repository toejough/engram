# What `superpowers:writing-skills` Provides That the Runbook Path Lacks

Scope: compares the `superpowers:writing-skills` skill-authoring process/format against engram's
`learn`/`write-memory` runbook-authoring path, using the phase-2 conversion-parity encodings
(Task A: commit skill → A-R runbook / A-F fact; Task B: vault note 830 → B-F fact / B-S skill) as
concrete evidence.

## 1. What writing-skills prescribes

**STRUCTURE.** SKILL.md is a fixed template
(`/Users/joe/.claude/plugins/cache/claude-plugins-official/superpowers/6.3.0/skills/writing-skills/SKILL.md:105-137`):

```
---
name: Skill-Name-With-Hyphens
description: Use when [specific triggering conditions and symptoms]
---
# Skill Name
## Overview / ## When to Use / ## Core Pattern / ## Quick Reference
## Implementation / ## Common Mistakes / ## Real-World Impact (optional)
```

- **Frontmatter description is a trigger key, not a summary.** "The description should ONLY
  describe triggering conditions. Do NOT summarize the skill's process or workflow" (SKILL.md:150-152)
  — a workflow-summarizing description measurably caused an agent to skip the body and follow the
  summary instead (SKILL.md:154-158, the two-review vs one-review finding).
- **Numbered procedure**, plus a **Common Mistakes / rationalization table** and a **Red Flags**
  list for discipline-enforcing skills: "Capture rationalizations from baseline testing... Every
  excuse agents make goes in the table" (SKILL.md:516-526); "Make it easy for agents to self-check
  when rationalizing" (SKILL.md:528-542).
- **references/** for heavy material — kept out of the always-loaded body (SKILL.md:84-91,
  347-372).
- **Match the Form to the Failure** (SKILL.md:459-474): a discipline failure (skips a rule under
  pressure) gets prohibition + rationalization table + red flags; a shaping failure (wrong output
  shape) gets a positive recipe instead — prohibitions on a shaping problem "trended worse than even
  the no-guidance control" in the skill's own micro-tests.

**PROCESS.** The Iron Law: "NO SKILL WITHOUT A FAILING TEST FIRST" (SKILL.md:374-393), applied as
RED-GREEN-REFACTOR (SKILL.md:552-591):

- **RED** — run the pressure scenario with a subagent *without* the skill; record verbatim what
  choices and rationalizations it produces. "If you didn't watch an agent fail without the skill,
  you don't know if the skill teaches the right thing" (SKILL.md:16).
- **GREEN** — write the skill targeting exactly those observed rationalizations; re-run; verify
  compliance.
- **REFACTOR** — find new rationalizations, add explicit counters, re-test until bulletproof.
- **Pressure tests** stack multiple pressures (time, sunk cost, authority, exhaustion) for
  discipline skills (SKILL.md:395-410); **description/trigger-keyword tests** are folded into SDO
  (Skill Discovery Optimization, SKILL.md:140-278) — rich description + keyword coverage so a
  *future* agent finds the skill at all, independent of whether it complies once found.

## 2. What the runbook path prescribes

The schema is three required fields, per
`openspec/specs/learn-runbook-capture/spec.md:22-24`: `situation` (retrieval handle, embedded as a
situation vector), `done_when` (ending expectation), and a `body` of numbered steps (may
`[[wikilink]]` fact/feedback notes). "There SHALL be no `inputs`/argument-signature field and no
`task_type` field" — the schema is deliberately minimal.

Authoring, per `agent-instructions/skills/learn/SKILL.md:182-197`: a runbook is captured at a
single **confirmation moment** — kind-4 (confirmed approach), fired when "a specific,
generalizable approach was validated as good" — picked over `feedback` by shape ("a reusable,
ordered, multi-step procedure for a recurring task"). The write-memory handoff
(`agent-instructions/skills/write-memory/SKILL.md:62-73`) composes one `engram learn runbook`
CLI call: `--slug --position --source --situation --done-when --body`. There is:
- no baseline run of the task *without* the note to confirm a failure exists,
- no pressure test of the note's discipline content under stacked pressure,
- no separate trigger-keyword test of `situation` against a natural task prompt,
- no rationalization-table or red-flags field in the schema — anything of that shape has to be
  hand-folded into the `body` prose.

What the binary adds mechanically, per the spec: embed-on-write (dual-vector sidecar — situation
vector + body vector, `learn-runbook-capture/spec.md:60-67`), vocab term assignment
(same-kind as fact/feedback), and Luhmann disposition (`--position`/`--target`,
`learn-runbook-capture/spec.md:37-49`). None of this is process — it's binary-side indexing that
happens regardless of authoring rigor.

## 3. What the B-S conversion actually did

Per `encodings/taskB/B-S/evidence/EVIDENCE.md` and `conversion-fidelity-report.md:96-146`:

- **RED**: 4 escalating-pressure headless runs (`claude -p`, same scratch-repo fixture, no skill
  installed) — red1 (explicit), red2 (terse+time), red3 (authority: wrong pattern handed down),
  red4 (deadline + authority + false-confidence + explicit "skip verification" + explicit
  `git add -A`). **3 of 4 passed** without any skill; **red4 failed**: the model staged
  `scratch/wip.txt`, complying with the literal `git add -A` instruction under stacked pressure,
  violating the runbook's own step 5 / `done_when`.
- **GREEN**: skill installed, identical red4 prompt — passed. Staged exactly the 4 correct files,
  explicitly excluded `scratch/wip.txt` and `.claude/`, and stated in its own transcript "I staged
  an explicit list instead of `git add -A`."
- **PRESSURE**: a second, different combined-pressure prompt (exhaustion/sunk-cost + false-
  confidence callback + skip-instruction + `git add -A`) — passed, held under the new combination.
- **Cost**: RED $3.27 + GREEN $0.79 + PRESSURE $1.86 = **$5.92** total, no refactor iteration needed
  (GREEN and PRESSURE both passed on the first-written skill content).
- The `## Common Mistakes` and `## Red Flags` sections were written **specifically to counter the
  observed red4 failure** — three Common Mistakes rows target exactly the three rationalizations
  in the red4 prompt ("proven pattern, no need to re-verify," "out of time, `git add -A`," "skip
  checks, ship it"); the Red Flags list restates S3/S5/S6 as pre-action stop triggers
  (`conversion-fidelity-report.md:124-127`).

From the fidelity report (`conversion-fidelity-report.md:96-146`):
- **B-S is the most faithful of the three encodings** of note 830 — all 6 steps plus `situation`
  and `done_when` reproduced **word-for-word**, in a genuine numbered markdown list, "nothing
  reworded, weakened, or dropped" (line 111).
- **Fact encodings (A-F, B-F) flatten the ordered list into one scalar field** (no markdown list —
  "the fact schema has no notion of an ordered list," line 62) and then **duplicate that entire
  content a second time in the body** as a restated "Information learned: ..." sentence — "this
  roughly doubles byte count relative to an equivalent runbook encoding of the same content (A-R
  2513B vs A-F 4113B for the identical source)" (line 91). B-F is 3603B vs. source B's 1733B.
- **A-F's `situation` key is weak and self-contradicting**: "committing changes in a git
  (**non-jj**) repo..." excludes jj even though the object's own step 1 branches on jj — "an
  internal contradiction" (line 60), and it's phrased as a retrieval request rather than a task
  situation, producing weaker lexical overlap against a natural prompt than A-R's situation.

## 4. Gap table

| Dimension | skill (writing-skills) | runbook (learn path) | fact (learn path) |
|---|---|---|---|
| Retrieval/trigger key quality | Frontmatter `description` is an explicit trigger-keyword field, separately optimized (SDO, symptom/keyword coverage) and format-tested against the "summarizes workflow → agent shortcuts past body" failure (`SKILL.md:140-197`). B-S's description lists explicit symptom keywords ("gitignore anchoring, git check-ignore, middle-slash patterns... git add -A, git add .") | `situation` is a single embedded-vector field, verbatim-reusable from a source note (B-F/B-S both reused note 830's situation verbatim — `conversion-fidelity-report.md:70,102`) but with no keyword/symptom design step and no test against a natural prompt | Same `situation` field, same lack of design/test step; A-F's instance was measurably weaker — narrowed, self-contradicting (`report.md:60`) |
| Ordered-steps preservation | Numbered `## Procedure` list, native to the template | Native numbered list in `body` — schema supports it and A-R/B-S both keep it as a real list | **Flattened** — no list construct in the schema; both A-F and B-F collapse steps into one run-on `object`/`predicate` sentence (`report.md:62,85`) |
| Evidence-backed failure targeting | `## Common Mistakes` + `## Red Flags` are *required outputs of RED* — every row traces to an observed rationalization (`SKILL.md:516-542`); B-S's 3 Common Mistakes rows map 1:1 to the red4 pressure prompt's 3 rationalizations | No such field; a runbook's `body` prose can *state* a rule ("never `git add -A`") but nothing prescribes capturing the rationalization that defeats it, and nothing was captured for note 830's B-R form (only B-S added it) | Same absence as runbook — no structural field, no process step generating one |
| Validation (baseline / pressure) | Mandatory: RED baseline without the skill, GREEN with it, pressure tests stacking 2-3+ pressure types (`SKILL.md:395-410,552-591`) — B-S ran 4 RED + 1 GREEN + 1 PRESSURE scenario before shipping | None — a runbook is captured once at confirmation and shipped; no baseline run without it, no pressure test of its discipline content | None — same as runbook |
| Size / duplication | Body holds content once; heavy reference material lives in separate files outside the always-loaded skill (`SKILL.md:84-91`) | Content held once, in a genuine list — A-R (2513B) ~ source A (2475B) | **Content duplicated** — frontmatter field + body "Information learned" restatement double the bytes for identical content (A-F 4113B vs A-R 2513B; B-F 3603B vs source B 1733B) |
| Authoring cost | High and explicit: 4 RED runs + GREEN + PRESSURE = $5.92 in this eval, plus iteration/refactor cycles when loopholes are found | Low: one `engram learn runbook` CLI call composed by write-memory from an in-session confirmation moment — no paid validation run | Same low cost as runbook — one CLI call, no validation |

## 5. What to adopt, and where

**STRUCTURE adoptions**

- Add a `## Common Mistakes` / rationalization-table convention to the runbook `body` (or a new
  optional field) for runbooks that encode a discipline rule (a "never do X" under pressure) —
  populate it only from an observed failure, not speculatively. *Applies to any note* — a fact
  note's `object` prose could carry the same table shape if the fact encodes a discipline rule;
  it's not runbook-specific in principle, but runbooks are the shape most likely to carry
  step-adjacent discipline rules (e.g. B's step 5 "never `git add -A`"). — **applies to any note**
- Add a trigger-keyword clause to `situation` phrasing guidance: today `situation` is a single
  retrieval-shaped sentence; writing-skills' SDO practice (symptom/error-message/tool-name keyword
  coverage, third-person, "Use when...") is a cheap wording discipline that both fact and runbook
  situations lack today, and A-F's self-contradicting/narrowed situation shows the cost of skipping
  it. — **applies to any note**
- Keep the numbered list in the `body` for runbooks (already schema-supported and already the
  differentiator that keeps A-R/B-S faithful while A-F/B-F flatten and duplicate) — this is *already
  correct behavior*, not a gap, but it argues for **never allowing a runbook body to collapse into
  prose** the way A-R's `## Rules` section did (`report.md:34,36`: the 6 rules were flattened by
  authoring choice, not schema force) — worth a write-memory reminder to preserve list structure
  for any embedded rule set, not just the numbered steps. — **applies to any note** (structural
  discipline, not exclusive to runbooks)
- A `red_flags`-style self-check list distinct from Common Mistakes (pre-action stop triggers vs.
  after-the-fact rationalization rebuttals) is a genuinely separate structural element B-S added
  beyond note 830 (`report.md:125`) — **argues for a distinct runbook type**, since a fact's
  subject/predicate/object shape has no natural slot for a "stop and check" list the way an
  ordered-steps body does.

**PROCESS adoptions**

- A lightweight RED/GREEN for runbooks captured at a correction moment: replay the correction
  scenario headless once *without* the note and once *with* it, to confirm the note actually
  changes behavior (mirrors writing-skills' Iron Law, scaled down — not full pressure-scenario
  design). — **process only**, and cheap enough to consider for any note type that encodes a
  behavioral rule, not just runbooks.
- Reserve full pressure-testing (stacked time/authority/exhaustion pressure, the $5.92-per-skill
  cost seen here) for runbooks that encode a **discipline rule** (a "never do X even under
  pressure" clause) rather than every runbook — B's `git add -A` prohibition warranted it; a purely
  sequential technique runbook with no prohibition clause likely doesn't. — **process only**,
  runbook-shaped work but gated on content, not kind.
- Do not adopt full writing-skills SDO/description-shortcut testing wholesale — engram's
  `situation` is matched by embedding similarity, not literal LLM-read description text, so the
  specific "workflow-summary description causes shortcut-past-the-body" failure mode
  (`SKILL.md:154-158`) doesn't transfer as-is; the applicable subset is keyword/symptom richness in
  the situation wording, not the trigger-vs-summary distinction itself. — **process only** (a
  caveat on what *not* to port)

**The distinct-runbook-type question:** the adoptions that argue for a distinct runbook type are
narrow — a `red_flags`/self-check-list slot (no natural analog in fact's flat triple) and gating
full pressure-testing on discipline content specifically found in ordered-step bodies. The
Common-Mistakes-table convention, situation/trigger-keyword wording discipline, and list-
preservation discipline all apply to facts too and don't by themselves justify a special build —
they're authoring-quality fixes available to any note kind. If runbook survives only because it
needs special build to match a skill's behavior-shaping power, the load-bearing adoption is the
red-flags slot plus discipline-gated pressure-testing; the rest is general authoring hygiene.

## 6. What the B-S vs B-R result will tell us

B-S carries `## Common Mistakes` and `## Red Flags` sections that the original vault note 830
(B-R) lacks entirely — both were written specifically to counter the empirically observed red4
failure (staging an unrelated file under `git add -A` compliance pressure), not spliced in
speculatively. If B-S beats B-R by 2+ trials on FOLLOWED-all or END-STATE in the phase-2 harness,
that is direct evidence the rationalization-table/red-flags additions change agent behavior beyond
what the plain numbered steps already convey — validating the "argues for a distinct runbook type"
adoptions above. If the two arms land at parity, the finding is narrower: for *this* task at n=5,
the runbook's plain ordered steps were already sufficient to drive correct behavior, and the
skill-format additions were unneeded scaffolding for this particular discipline rule — which would
weigh toward the "applies to any note, general hygiene only" reading of section 5 rather than a
distinct runbook type.

RESULTS: <to be filled after the opus run>
