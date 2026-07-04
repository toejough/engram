# Lesson-Capture Blind Spot — Diagnosis and Guard Options

**Status:** analysis complete — options for Joe's decision. NO skill edits this cycle.
**Ask (Joe, 2026-07-04, verbatim):** "I don't love that I had to point those out. Is there a
category error or some blind spot we can guard against in the learn skill, or somewhere else?
/please think about this and present some options"
**The two pointed-out failures:** (1) building the reference-card atom against the design's own
research null; (2) reporting the unverified "0/27 dereference" zero-rate, which shaped a
decision. Both were crystallized only after Joe asked (now notes 166–168).
**Related:** `docs/design/2026-07-04-atomic-skills-options.md` (the cycle that produced both
failures), `docs/design/2026-07-01-memory-system-review.md` (system scorecard), ROADMAP "Deeper
arc — rebuild the skills from behavioral atoms" (where any picked guard lives — see Placement).

> **Since fixed (2026-07-04, same day):** the diagnosis below describes the taxonomy as it stood
> when written; G1+G2+G6 shipped hours later (see the Decision record at the end). The learn
> skill now scans three kinds; please step 7 audits; escalations carry provenance.

## The answer up front

**Yes — there is a category error, and it is in the learn skill's Step-2 taxonomy.** The scan
recognizes exactly two moment kinds — corrections *the user states* and explicit save-requests —
and its red-flags table forbids writing anything else ("things nobody asked you to remember").
A **self-discovered reversal** (a design I superseded, a finding I retro-invalidated) matches
neither kind: nobody *said* the correction, so the closing scan passes over precisely the
highest-value lessons — the ones that overturned my own conclusions. Three aggravators let the
misses compound, and one of them (ungated mid-cycle escalations) is where the faulty 0/27
reached you. This is a recurring family, not a lapse: note 154 records you correcting the same
pattern for parked observations on 2026-07-01; today was instance two.

All options below are **capture-side** — changes to how lessons get *written* (learn/please),
not to when recall *fires* (recall-side after-failure cues were rejected 2026-06-30, note 147,
and are not relitigated here).

## The pipeline trace — where each miss slipped through

| Stage | Failure 1 (ecosystem-null violated) | Failure 2 (0/27 misreported) |
|---|---|---|
| At the moment of the error | Beat-2's "nobody ships this" logged as an honest bound + ship-gate; design proceeded | 0/27 reported to Joe via a mid-cycle escalation with no evidence-pointer or validity check — no gate reviews escalation payloads (Gate D fires at step 6, over commit prose) |
| At discovery of the error | Joe's boundary critique → design redrawn; supersession recorded in plan docs | Shadowing artifact found → CORRECTION section written into the results doc |
| Mid-cycle capture hook | none exists — learn fires only at cycle open/close | none exists |
| Cycle-close learn scan | not a USER correction at scan time (Joe's critique preceded the previous close; the root-cause lesson was self-derived) → no category → skipped | self-discovered; record already corrected → felt "handled" → skipped |
| Result | lesson evaporated until Joe asked | lesson evaporated until Joe asked |

The "felt handled" step is load-bearing: correcting the repo record satisfies the integrity
instinct, but record-correction is not lesson-capture — repo docs are chunk-memory (raw,
searchable), not recall-ranked lessons, and note 76's rule applies: you can only fix (or reuse)
a conclusion that exists as a note.

## Guard options (full set, honest ratings)

Glosses used below: **capture-side** = edits to the learn/please write path, never recall
firing. **Pressure tests** = note-28's fresh-agent probes that try to rationalize around new
skill text. **AskUserQuestion** = the harness's mid-cycle option-question escalation to the
user (how the 0/27 reached Joe). **Doc scrub** = note-64's rule that every doc naming a changed
path updates in the same effort — every build cost below includes it: G1's scrub targets are
called out in its cell (docs naming the two-kind scan); for G2–G6 the scrub target is the
please skill's own step descriptions plus any doc naming the step-7 close or the escalation
flow (c1's please sequence notes, GLOSSARY if a new term ships). **Instrument-invalid** = a
measurement later found not to have tested what it claimed (e.g., the 0/27 count: the arms never
loaded the texts under test), so its verdict is void — a commit-message marker for reversals.

| # | Guard | Mechanism | Would have caught | Cost (build, incl. doc scrub) | Weakness | Rating |
|---|---|---|---|---|---|---|
| **G1** | learn taxonomy third kind: REVERSALS | Step-2 scan adds: any conclusion/design/verdict PRESENTED (to user, gate, or plan) later overturned — by you, a reviewer, or an instrument — crystallizes the ROOT CAUSE of the original error. Red-flags row: "you corrected a repo doc and skipped the note — record-correction is not capture." (The "exactly two kinds" header and line-110 red-flag are change targets; a new numbered sub-step per the 2.5 precedent is the alternative shape.) | Both failures, at cycle close | one learn-skill TDD cycle, ~$0.5–1 (+ scrub of docs naming the two-kind scan) | still relies on the closing agent's honest scan — the same bias window it guards | **CONTENDER (core)** |
| **G2** | please step-7 lessons audit over the cycle's mechanical corpus | Before the cycle report: enumerate (a) pre-registered STOPs fired, (b) gate FAILs, (c) commits containing CORRECTION/supersede/instrument-invalid/redraw, (d) escalations — each maps to a vault note or an explicit "no lesson because X" line | Both (both failures left commit-message fingerprints) | please-skill edit: writing-skills TDD + pressure tests, ~$1–2 | heavier step 7; the mapping judgment is still self-performed | **CONTENDER (core)** — the corpus is observable, so the scan can't rely on memory |
| G3 | fresh-context lessons reviewer at cycle close | A reviewer reads the cycle's commits + report draft and REFUTES: "which failure modes occurred; is each crystallized?" — a subagent plays Joe's role before Joe has to | Both — and removes self-serving bias entirely | please-skill edit (TDD + pressure tests, ~$1–2) + reviewer prompt; ~$0.05–0.15/cycle runtime | needs G2's corpus definition anyway; adds a reviewer round per cycle | CONTENDER (stronger, heavier alternative to G2) |
| G4 | crystallize-on-discovery coupling | Writing a CORRECTION/supersession/instrument-invalid label into any repo doc REQUIRES the paired vault write in the same action (write-memory handoff), scoped to reversals of already-presented conclusions | Failure 2 at the discovery moment; failure 1 at supersession time | please-skill edit (TDD + pressure tests, ~$1–2) | rule-in-prose enforcement — the under-fire family this repo has measured repeatedly; needs scope discipline against note-spam | PARK unless paired with G2/G3 (the audit catches what the rule under-fires) |
| G5 | gate mid-cycle escalations | Any AskUserQuestion/STOP report carrying MEASURED claims gets a fast fresh-context ground-truth check first (re-derive the load-bearing number from raw artifacts — note 162's results-doc rule extended to escalations) | Failure 2 BEFORE Joe decided on it | please-skill edit (TDD + pressure tests, ~$1–2) | latency on escalations; needs a proportionality rule (load-bearing measured claims only) | CONTENDER (reporting side, enforced) |
| **G6** | escalation provenance rule (lightweight G5) | Any measured claim in a mid-cycle escalation must carry its evidence pointer (file/command) + a one-line validity statement ("verified how?") — forcing at authoring time exactly the question the 0/27 report skipped | Failure 2, plausibly (the honest answer to "verified how?" was "not verified — no delivery check") | please-skill edit (TDD + pressure tests, ~$1–2) | self-enforced prose rule (under-fire family); G5 is its enforced upgrade | **CONTENDER (reporting side, cheap)** |

## Recommendation (Joe decides)

**G1 + G2 as the core, G6 for the reporting side.** G1 fixes the category error where it lives;
G2 makes the close-scan run over an *observable* corpus (commits, gate verdicts, STOPs) instead
of trusting the same memory-and-bias window that missed twice; G6 forces "verified how?" into
every measured escalation at near-zero cost. Upgrade paths, explicitly staged: if G6 proves
leaky (a future escalation ships an unverified number despite the rule), G5 is its enforced
form; if G2's self-performed mapping proves leaky, G3 externalizes it. G4 is parked as a
complement — its discovery-moment timing is attractive, but it is pure rule-in-prose, the
enforcement class this repo has measured under-firing, and G2/G3 catch the same content later
with better enforcement.

**What this costs if you take the core + G6:** one learn-skill TDD cycle plus one please-skill
edit (both under writing-skills TDD; the please edit also takes note-28 pressure tests), ~$2–3
total, one session. Everything is text — trivially reversible.

## Placement

A picked guard lives in the ROADMAP as a small item under the atoms arc's discipline thread
(the same seam family as the write-memory worker: capture behavior at the step boundaries the
workflow already has). No new track needed.

## Decisions needed from Joe

1. **Pick guards:** G1+G2+G6 (recommended) / any other combination from the table / none
   (accept the recurrence risk; note 169 alone raises the odds the close-scan self-catches, but
   it is exactly the kind of prose-only guard this analysis says under-fires).
2. If picking: bless the build order (G1 first — smallest, fixes the category error; then G2+G6
   in one please-skill edit).

---

**Decision (Joe, 2026-07-04): "run through your recommendation" — G1+G2+G6 picked and SHIPPED
the same day** (plan `docs/superpowers/plans/2026-07-04-capture-guards-build.md`; batteries
all-PASS at e13c3c9f under marker validity gates; production commits 65327115 + bdc3f3dc).
Staged upgrades G6→G5 and G2→G3 pre-registered in the ROADMAP atoms-arc status block; G4 parked.
