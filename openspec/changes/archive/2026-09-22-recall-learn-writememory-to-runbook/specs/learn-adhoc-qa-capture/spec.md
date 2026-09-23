# Learn Ad-hoc QA Capture Gate

## Purpose

Learn's Step 2.5 ("Ad-hoc QA capture") decides whether a question substantively answered during
the session gets its own `engram learn qa` pair, on top of whatever Step 2 notes were already
crystallized. This capability specifies the gate that stops Step 2.5 from re-capturing content a
Step 2 note already captured. Why: confirmed production bug found via the write-memory eval (task
3.4b, trial R-0 — `dev/eval/cumulative/runbook_vs_skill/phase2/results/3.4b_write_memory_shim_only_retry2_sonnet5.md`):
the old Step 2.5 wording's "OR if you crystallized a new vault note (Step 2) as the answer"
disjunct treated the agent's own just-written Step 2 note as a TRIGGER for a separate QA-capture
write, rather than as evidence the answer was already captured — producing an unrequested
duplicate `engram learn qa` write (two extra files) whenever the agent posed and answered a
diagnostic question via its own Step 2 note in the same turn.

## ADDED Requirements

### Requirement: Learn SHALL treat a self-answered Step 2 note as already-captured, not as an additional QA-capture trigger

When a question is posed and answered within the learn skill's own Step 2 processing this
session, and the answer's traceability is provided by a Step 2 note the agent wrote this same
turn, that note SHALL be treated as the capture for that question. Learn SHALL NOT additionally
write a separate QA pair (Step 2.5) for the same question, even though the answer would otherwise
satisfy the substantive-answer bar (a `[[wikilink]]` or "crystallized a new vault note" disjunct).

#### Scenario: Question answered by the agent's own just-written Step 2 note

- **WHEN** a question is posed and answered within this session, and the answer's traceability is
  provided by a Step 2 note the agent wrote this same turn (regardless of whether that note also
  cites a `[[wikilink]]`)
- **THEN** learn SHALL skip Step 2.5 QA capture for that question — the Step 2 note is the capture

### Requirement: A question answered by citing an EXISTING prior note SHALL still get a QA pair

The wikilink-based substantive-answer trigger remains valid on its own: when a question is
answered by citing a `[[wikilink]]` to a vault note that existed BEFORE this turn (not one the
agent just crystallized in Step 2 this turn), learn SHALL still write the QA pair per Step 2.5,
subject to the existing duplicate-write gate (e.g. a pair already written by recall's Step 4).

#### Scenario: Question answered by citing a pre-existing note, no new Step 2 note written this turn

- **WHEN** a question is answered this session by citing an existing prior vault note's
  `[[wikilink]]`, and no new Step 2 note was crystallized for it this turn
- **THEN** learn SHALL write the QA pair per Step 2.5
