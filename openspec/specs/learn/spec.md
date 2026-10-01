# learn Specification

## Purpose

How the learn skill treats the `LESSONS:` lines returned by a session's dispatched subagents — judged at each return on the mid-cycle fast path, with the closing capture as a backstop that skips lines a note written earlier this session already covers: each line is judged against Step 2's existing four-kind bar and silently discarded if it fails, and the kind-4 (confirmed-approach) scan is anchored to the moment a unit's outcome is confirmed, with audit-derived exemplars. Why: the memory-loop audit (`dev/eval/audit/REPORT-2026-09-02.md`) found learn fired for 5.6% of worth-learning moments, with missed success-reinforcements the largest lost bucket; this is the curation half of capture-then-curate (change `learn-rate-skill-only`).
## Requirements
### Requirement: Closing learn curates collected LESSONS lines

The learn skill MUST apply its LESSONS rule to each returned subagent result, on the mid-cycle fast path, and MUST keep the closing learn as a backstop.

**Per return (fast path).** When the orchestrator fires `/learn` at a subagent return (the cue in the ambient learn guidance), the learn skill MUST skip Step 1 (sweep) and Step 1.5 (vocab), go straight to Step 2, and judge each lesson in that one returned `LESSONS:` line against the skill's existing Step-2 bar (defined in `agent-instructions/skills/learn/SKILL.md` Step 2; this change moves where the bar is applied, it does not change the bar):
1. The existing four-kind moment taxonomy — kind 1: corrections (a review or the user rejected an approach), kind 2: explicit save-requests, kind 3: reversals (a presented conclusion later overturned), kind 4: confirmed approaches (a validated bet or explicitly praised specific behavior)
2. The existing quality bar — the lesson is confirmed rather than hypothesized (kind 4 requires resolved uncertainty or explicit specific confirmation, never bare success) and states a general reusable principle with a retrieval-shaped situation, not a session-specific narrative

Lessons that pass both crystallize into vault notes via write-memory (the standard flow), one note per distinct principle. Lessons that do not pass are discarded silently (no logging, no storage, no offer archive).

**Closing backstop.** At the closing learn step (the final write-to-vault step in a session), the learn skill MUST still run its sweep, and MUST scan the collected `LESSONS:` lines from all dispatched work for lines not yet captured at return. A line counts as captured when a note written earlier in this session covers it. Uncaptured lines are judged against the same bar and crystallized or silently discarded as above; captured lines are skipped, never written a second time.

#### Scenario: LESSONS line clears quality gate

- **WHEN** a LESSONS line reads "review rejected bare error returns; the convention is `fmt.Errorf` with `%w`" and matches kind 1 (a confirmed correction)
- **THEN** the learn skill passes it to write-memory, which writes it as a new note to the vault

#### Scenario: LESSONS line fails quality gate

- **WHEN** a LESSONS line reads "we did the work" (bare success, no resolution of uncertainty) and does not match any kind
- **THEN** the learn skill discards it silently without attempting to write it

#### Scenario: Fast-path fire at a subagent return

- **WHEN** the orchestrator invokes `/learn` mid-cycle for one returned LESSONS line that clears the bar
- **THEN** the learn skill runs no `engram ingest --auto` and no vocab check, and writes one note for that lesson via write-memory

#### Scenario: Closing learn skips lines captured at return

- **WHEN** the session collected three LESSONS lines, one of which was captured by a fast-path `/learn` at its return
- **THEN** the closing learn sweeps, does not write that line again, and judges only the other two

#### Scenario: Closing learn catches a line missed at return

- **WHEN** a LESSONS line that clears the bar was not captured at its return
- **THEN** the closing learn crystallizes it via write-memory

#### Scenario: Mixed batch of LESSONS lines

- **WHEN** the session collected four uncaptured LESSONS lines: one kind-1 correction, one kind-4 confirmed approach, one bare success, and one generic ("remember to test")
- **THEN** the closing learn writes the kind-1 and kind-4 lines via write-memory, discards the other two

### Requirement: Kind-4 cue gains completion-moment anchor and audit-derived exemplars

The kind-4 (confirmed-approach) detection in the learn skill's Step 2 MUST include:
1. An explicit completion-moment anchor (per design D-C): the kind-4 scan fires when a unit's outcome is confirmed — a review verdict lands, a check/test passes, or the user explicitly confirms — and, at the closing learn, over the session's completion reports and collected LESSONS lines
2. The 2–3 concrete audit-derived exemplars of real missed success moments listed in `tasks.md` task 1.3 (to improve recognizability of the pattern)

The bar itself (resolved uncertainty or explicit specific confirmation, never bare success) is unchanged. Only the recognizability of kind-4 moments improves.

#### Scenario: Kind-4 moment at completion
- **WHEN** the session ends with the agent delivering a specific decision (e.g., "validated the X pattern for use in Y; future sessions do the same here") and the LESSONS line captures this
- **THEN** the closing learn's kind-4 scan anchors to this completion moment and recognizes it as a learnable success

#### Scenario: Kind-4 moment with exemplar match
- **WHEN** a LESSONS line from a subagent reads "targeted reproduction test confirmed the suspected defect — stray route persists when the tunnel setup fails" (the same shape as the skill text's audit-derived exemplar "defect confirmed by targeted reproduction test")
- **THEN** the closing learn recognizes it as kind-4 (the exemplar in the skill text makes the shape recognizable) and crystallizes it

#### Scenario: Bare success not kind-4
- **WHEN** a LESSONS line reads "the task is done" (success but no resolved uncertainty or specific confirmation)
- **THEN** the closing learn does not classify it as kind-4 (bare success still fails the bar, exemplars and anchor do not lower it)

