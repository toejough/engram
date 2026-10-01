## MODIFIED Requirements

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
