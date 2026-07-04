# Atomic skills redesign — options report

**Status:** exploration complete (research + census + sandbox smokes, ~$1 spent of the $8–18
envelope) — options for Joe's decision. NO production skill edits this round.
**Ask (Joe, 2026-07-04):** options for reworking the skills around atoms + SRP so a change
lives in one place; research good skill design; smoke test the options; behavior preservation
is the bar, readability/maintainability the goal.
**Evidence:** plan `docs/superpowers/plans/2026-07-04-atomic-skills-exploration.md`
(Gate-A-approved; settled constraints S1–S3); smokes committed at `dev/eval/atoms/`
(58ad18a4: results doc, 45-arm raw JSONL, the 8 sandbox texts actually tested); four research
beats summarized below. Numbers labeled **measured / analysis / null**.

## The verdict up front

**One option survived its pre-registered smoke: O-A — a `write-memory` atom skill holding the
mechanical write procedures, invoked by name from recall and learn, with all judgment staying
in the parents.** The theoretically-safer option (O-B, prose cross-references) FAILED its
smoke with a failure mode worse than the one we feared: arms following a pointer to text that
wasn't in front of them **confabulated plausible-but-invalid flags** (`--contributor`,
`--note`, and in one arm `--chunk-source` — none valid for `learn qa`, whose real flag is
`--contributors`) — silent, CLI-invalid output. The duplication you
named is real (measured census below), Rule of Three endorses extracting exactly the block
that bothers you, and the ecosystem/officials endorse the whole-named-skill sharing mechanism
O-A uses. Honest bounds and ship-gates below; your call at the end.

## Duplication census (measured, F2 of the plan)

| Procedure | Copies | Sites |
|---|---|---|
| `engram learn fact\|feedback` invocation + supersedes flag guidance | **3** | recall Step 2.5C, recall Step 4, learn Step 2 |
| `engram learn qa` invocation | **2** (+1 pointer) | recall Step 4, learn Step 2.5 (please step 7 points, correctly) |
| `engram ingest --auto` sweep | 2 (intentional — different roles) | recall Step 0.5, learn Step 1 |

Skill sizes: recall 291 / learn 127 / please 114 / route 77 lines (measured).

## Research consolidation (four beats, note-157 sweep)

- **Beat 1 — Official guidance (nulls labeled):** skill scope is defined ONLY by token budget
  (<500 lines / <5k tokens) — no official "one skill per function" concept exists. The ONE
  endorsed cross-skill mechanism is the prose name-pointer (`REQUIRED SUB-SKILL: X`), with
  explicit "don't repeat what's in cross-referenced skills." A NON-TRIGGERING description
  (O-A's key device) is addressed nowhere — not endorsed, not prohibited: smoke-only evidence.
- **Beat 2 — Shipped ecosystems (measured observations):** across superpowers' 14 skills,
  1,000+ community skills, and the major agent frameworks, **no shipped system shares fragment
  FILES between skills — the sharing unit is always the whole named skill.** Superpowers
  knowingly duplicates discipline scaffolding (its "Iron Law" no-X-without-Y-first rule blocks, copy-pasted across 4 skills) so each skill stands alone, and
  extracts only mechanical template content — within the owning skill. Engram's 3-copy
  fact/feedback block is template-class (extract-candidate); our duplicated discipline framing
  is duplicate-class (leave alone). O-A's shape would be a genuine first only in its
  non-triggering description; "instruction bleed" literature warns against ACTIVE-description
  atoms (O-C — parked).
- **Beat 3 — SE theory (analysis + cited literature):** extraction pays iff the indirection is
  reliable AND the abstraction is context-free. Current skills already sit at good cohesion
  seams (recall = sequential-cohesive pipeline; learn/please/route = functional). Rule of
  Three triggers for exactly one duplication — the 3-copy fact/feedback block — and the AHA
  principle ("avoid hasty abstraction", Dodds) plus Metz's wrong-abstraction test
  constrain the extraction to flag mechanics only (an atom that carries covered/near/absent
  branching is the wrong abstraction). Five testable predictions were pre-stated; see smoke
  outcomes.
- **Beat 4 — Failure modes (vault-grounded):** seven modes cataloged (under-fire silent step
  loss; over-fire/competing descriptions — the 147× history; judgment-seam fracture — note 78;
  please's non-thinnable anti-amnesia core — note 100; dispatch-overhead-not-worth-it — the
  −14% rollback, note 80; cross-reference drift; context-coupled duplication). Each mapped to
  options and mitigations in the plan.

**S0: no amendments to options catalog.** The four beats informed emphasis — the atom's
judgment-free scope pin (beat 3), O-C's park rationale (beats 2/4), and the non-fire ship-gate
(beats 1/2) — but warranted no additions, removals, or reshaping of the pre-registered
O-A..O-D set. Full beat consolidation: `docs/design/2026-07-04-atomic-skills-research.md`.

## Options (full set, honest ratings)

| Option | Shape | Smoke | Rating |
|---|---|---|---|
| **O-A — write-memory atom skill** | new `skills/write-memory/` holding the MECHANICAL write procedures (fact/feedback + qa flag blocks + supersedes syntax); recall/learn invoke it by name; ALL judgment (coverage verdicts, D2 bars, when-to-fire) stays in parents; deliberately non-triggering description; deploys automatically via `engram update`'s recursive walker (code-verified) | **PASSED all scenarios** (tables below) | **CONTENDER — recommended, with ship-gates** |
| **O-B — prose cross-references** | single-owner sections + pointers ("apply learn Step 2.5 verbatim"); no new skills | **FAILED Scenario 2**: 0/3 pointer arms produced valid flags (confabulated `--contributor`/`--note`/`--chunk-source`) vs 3/3 old | **ELIMINATED by pre-registered branch** ("a single scenario failure = the option FAILS, period") |
| O-C — active-description atoms | atoms that fire autonomously | not run (parked at plan stage) | PARK — the 147× over-fire history + "instruction bleed" literature |
| O-D — per-skill references/ copies | intra-skill file moves only | not run (parked at plan stage) | PARK — N copies remain N maintenance surfaces; zero cross-skill SRP (SE analysis: dominated) |

### Worked example under O-A (your one-place-to-update test)

A `--source` → `--source-ref` flag rename today: **3 edits** across recall 2.5C, recall Step 4,
learn Step 2 (plus 2 more for qa). Under O-A: **1 edit** in `skills/write-memory/SKILL.md`;
recall and learn pick it up at next invocation, their text untouched. The qa-capture change
scenario: likewise 1 edit.

## Smoke evidence (measured; haiku-4-5 arms, n=3 per text per scenario, 45 arms, ~$1)

| Scenario | Checkpoint | O-A arms hitting checkpoint, of 3 (old→new) | O-B arms hitting checkpoint, of 3 (old→new) | Fired branch |
|---|---|---|---|---|
| S1 recall NEAR coverage | NEAR verdict + `engram amend --target` + content flags | 3/3 → 3/3 | 3/3 → 3/3 | PASS both |
| S2 recall Step-4 qa capture | `learn qa` with `--contributors 159....` wikilink-derived | 3/3 → 3/3 behavioral | 3/3 → **0/3** (confabulated flags) | O-A PASS; **O-B FAIL** |
| S3 learn correction | one `learn feedback` w/ behavior/impact/action | 3/3 → 3/3 | 3/3 → 3/3 (identical text) | PASS both |
| S4 CONTROL (please unchanged) | {learn, recall, plan-not-skipped} in order | 2.5/3 = 2.5/3 (identical both groups; two arms listed 2 of 3 actions but affirmed plan-not-skipped; 0 disqualifiers) | same arms | harness valid |

- **Beat-3 prediction outcomes:** the predicted O-A under-fire (atom invocation silently
  skipped) was **not observed** — 0 skip incidents across all O-A arms; arms invoked the atom
  and produced correct flags 6/6 behaviorally on S2. The Metz wrong-abstraction signal was
  also absent (no branching errors — the atom carried no judgment by construction).
- **The O-B failure, precisely:** pointer arms had file-tool access and COULD have fetched the
  referenced learn text; none did — they guessed. Plausible-looking, CLI-invalid commands are
  a **silent** failure (nothing errors until the command runs). This is beat 3's "unreliable
  dereferencing" cost, measured: confabulation, not omission.
- **Harness incidents (disclosed):** 4 O-A S2 arms failed the sentinel check by echoing the
  ATOM's marker instead of the parent's — a structural quirk (the arm echoes the last marker
  it saw, i.e., after invoking the atom), itself evidence the atom text loaded; behavioral
  proof inline (representative of all 6 O-A S2 arms, from raw-arms.jsonl): "1. Identified that
  Step 4 requires invoking write-memory's QA capture procedure. 2. Produced `engram learn qa
  --contributors 159.2026-07-02.eval-runs-checkpoint-per-trial`" — correct plural flag,
  wikilink-derived, every arm — outputs uniformly correct; reruns confirmed the cause. Documented in the results doc.

## Honest bounds and ship-gates (what the smokes did NOT establish)

1. **Model tier + n:** arms were haiku-4-5 at n=3 — direction, not magnitude. The production
   readers are stronger models; O-B's confabulation would plausibly shrink on them, but
   "plausibly" is not evidence, and O-B is eliminated by its pre-registered branch regardless.
2. **The non-firing hypothesis was NOT tested.** The inline-text fixtures never exposed the
   atom's DESCRIPTION to an autonomous-firing decision (structurally untestable in this
   harness). If O-A ships, a deployed-sandbox negative test — "does anything fire write-memory
   without a parent instructing it?" — is a SHIP-GATE, not optional. Official guidance is
   silent on the pattern (beat 1); nothing in the ecosystem has shipped it (beat 2).
3. **The atom's judgment-free scope is load-bearing** (beat 3, confirmed by S2's clean run):
   any drift toward covered/near/absent branching inside the atom re-opens the
   wrong-abstraction failure. The build plan must pin the scope the way D5′ pinned polarity.
4. **please and route are untouched** in every option (please's anti-amnesia weight is
   measured value — note 100; route already maps 1:1 to its atom). The charter's read-memory
   atom is NOT extracted in any contender: recall's read+judge+write is sequential cohesion
   worth keeping (beat 3) — the atoms vision lands as "one write-memory atom + three skills
   already at their seams," which honors your "no N look-alike skills" constraint.
5. **Behavior preservation:** the bar Joe set held in every surviving cell (no regression
   anywhere in O-A's battery; S1/S3/S4 clean for both).

## Recommended path (if you pick O-A)

Round 1: build `skills/write-memory/` + the two parent edits via writing-skills TDD (headless
arms, sandboxed per the now-standard discipline), with the non-fire negative test as the
ship-gate; deploy via `engram update`; GLOSSARY entries for "atom" and "non-triggering
description" (docs-gate deliverable). Estimated 3 skill-TDD cycles, ~$5–10, one session.
Rollback is trivial (revert two parent texts + delete one dir).

## Ship-gate checklist (if O-A is picked — all three gate the deploy)

- [ ] **Non-fire negative test on a DEPLOYED sandbox**: with write-memory deployed, verify
      NOTHING invokes it without a parent skill's explicit instruction (the smokes could not
      test this — inline fixtures never exposed the description to autonomous firing).
- [ ] **Atom-scope pin honored**: the shipped atom text contains ONLY mechanical flag
      procedures — zero coverage/D2/when-to-fire judgment (audit against the smoke-validated
      sandbox text in dev/eval/atoms/sandbox-texts/).
- [ ] **writing-skills TDD per parent edit** (headless sandboxed arms, standing discipline) +
      GLOSSARY entries for "atom" and "non-triggering description".

## Decisions needed from Joe

1. **Pick:** O-A as scoped (recommended) / stay with the status quo (legitimate — the census
   pain is real but bounded: 3 copies, ~1 flag change per month at current cadence) / something
   else from the table.
2. If O-A: bless the ship-gates above (esp. the deployed-sandbox non-fire test) as build
   requirements.
3. The O-B confabulation finding stands on its own: pointer-style "apply X verbatim"
   references to out-of-context text are now a measured anti-pattern for these skills — worth
   remembering wherever we write skill prose, regardless of your pick.

---

## Postscript (2026-07-04, post-pick)

Joe picked O-A; the build then superseded this report's recommended FORM twice, honestly
recorded here:

1. **The reference-card atom was redrawn as a WORKER skill** on Joe's boundary correction
   ("you just drew the boundaries between skills at the wrong points"): parents keep every
   judgment and hand off at the write seams; write-memory composes, EXECUTES, retries on CLI
   errors, and reports — the whole-skill-as-next-action pattern please/superpowers already use,
   not a mid-procedure reference fetch. Shipped 2026-07-04 (`skills/write-memory/SKILL.md`).
2. **A deployed-round instrument caveat now attaches to this report's smoke table:** the inline
   fixtures above remain valid, but every LATER deployed-context arm round (project-level
   `.claude/skills` fixtures) was invalidated by a skill-shadowing artifact — same-named global
   skills load instead of fixture copies, so candidate texts were never read. The widely-cited
   interim "0/27 mid-procedure dereference" figure is instrument-invalid; the reference-card
   form was never actually tested deployed. See `dev/eval/atoms-build/results-2026-07-04.md`
   CORRECTION. The worker form was validated under a fixed, marker-gated harness: handoff fired
   W1 3/3, W2 3/3, W3 2/3 (haiku and sonnet), boundary violations 0, non-fire 0/6.
