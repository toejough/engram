# Learn Capability

## ADDED Requirements

### Requirement: Closing learn curates collected LESSONS lines

At the closing learn step (the final write-to-vault step in a session), the learn skill MUST scan the collected `LESSONS:` lines from all dispatched work and judge each line against the skill's existing Step-2 bar (defined in `agent-instructions/skills/learn/SKILL.md` Step 2; the edit adds LESSONS lines as an explicit scan input to that step, it does not change the bar):
1. The existing four-kind moment taxonomy — kind 1: corrections (a review or the user rejected an approach), kind 2: explicit save-requests, kind 3: reversals (a presented conclusion later overturned), kind 4: confirmed approaches (a validated bet or explicitly praised specific behavior)
2. The existing quality bar — the lesson is confirmed rather than hypothesized (kind 4 requires resolved uncertainty or explicit specific confirmation, never bare success) and states a general reusable principle with a retrieval-shaped situation, not a session-specific narrative

Lines that pass both crystallize into vault notes via write-memory (the standard flow). Lines that do not pass are discarded silently (no logging, no storage, no offer archive). This is the "curate-at-write" approach — capture-then-filter with zero new plumbing.

#### Scenario: LESSONS line clears quality gate
- **WHEN** a LESSONS line reads "review rejected bare error returns; the convention is `fmt.Errorf` with `%w`" and matches kind 1 (a confirmed correction)
- **THEN** the closing learn passes it to write-memory, which writes it as a new note to the vault

#### Scenario: LESSONS line fails quality gate
- **WHEN** a LESSONS line reads "we did the work" (bare success, no resolution of uncertainty) and does not match any kind
- **THEN** the closing learn discards it silently without attempting to write it

#### Scenario: Mixed batch of LESSONS lines
- **WHEN** the session collected four LESSONS lines: one kind-1 correction, one kind-4 confirmed approach, one bare success, and one generic ("remember to test")
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
