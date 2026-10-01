# please Specification

## Purpose

What the please workflow does with the `LESSONS:` lines its dispatched units return: it records each line verbatim as every dispatch resolves, judges each lesson at that moment per the ambient learn guidance's return cue (a fast-path `/learn` for each that clears learn's bar), hands the full set to the closing `/learn` at step 7 alongside the lessons-audit output, and never lets its gates filter that collection. Distinct from `lessons-contract`, which defines the line itself; this is where please carries it. Why: `learn-rate-skill-only` routed subagent lessons to the closing learn; `learn-at-subagent-return` moved the judgment to the return and kept the closing learn as backstop.
## Requirements
### Requirement: Orchestrator collects LESSONS lines from all dispatched work

As the please skill drives dispatched work steps, the orchestrator following it MUST collect the `LESSONS:` line from every subagent's completion report at the moment each dispatch resolves. Dispatches occur throughout the please skill's seven-step workflow (chiefly step 4 "Execute" and the adversarial review gates); the skill instructs the orchestrator to record each returned LESSONS line as part of handling every dispatch result, wherever in the workflow it lands. The orchestrator records the raw line exactly as written by the subagent, whether it says "none" or contains lessons, for every dispatch in the session. At the same moment, the orchestrator judges each lesson per the ambient learn guidance's return cue: a lesson that clears learn's bar is captured with `/learn` on the fast path before the next dispatch. Judging never removes or rewrites a line; the list stays complete and verbatim.

#### Scenario: Single dispatch in session

- **WHEN** please dispatches one subagent for a task and it returns a LESSONS line
- **THEN** please records the line for later hand-off (e.g., "LESSONS: fixed typo in docs, added unit test scenario")

#### Scenario: Multiple dispatches in session

- **WHEN** please dispatches three subagents in sequence (e.g., brainstorming, planning, implementation)
- **THEN** please collects all three LESSONS lines, preserving the order and exact text of each

#### Scenario: Subagent returns LESSONS: none

- **WHEN** a dispatched subagent's completion report includes "LESSONS: none"
- **THEN** please still collects the line (it is a valid entry in the collected set, not a blank), and no `/learn` fires for it

#### Scenario: Worth-keeping line captured at return

- **WHEN** a dispatched subagent returns a LESSONS line carrying a confirmed correction
- **THEN** please records the line verbatim, and the orchestrator captures it with a fast-path `/learn` before the next dispatch

### Requirement: Please passes collected LESSONS lines to closing learn

At workflow step 7 ("Capture (close) — `/learn`" in the please skill, with its lessons audit), the please skill MUST pass the collected LESSONS lines as input to the learn skill, alongside the existing lessons-audit output that step already hands over. This is explicit handoff — learn receives the full set of lines and judges them against the quality gate, skipping any lesson that a vault note written earlier in this session already covers (such as a lesson captured with a fast-path `/learn` at its return). The list carries no captured marks; the closing learn dedupes by vault coverage.

#### Scenario: Closing learn receives LESSONS collection

- **WHEN** the session's dispatched work is complete and please is preparing to run the closing learn
- **THEN** please includes the collected LESSONS lines in the learn's input context (e.g., passing them as a `lessons_from_session: [line1, line2, ...]` entry or inline text)

#### Scenario: No dispatches in session

- **WHEN** the session has no delegated work (e.g., user asked a question, coordinator answered directly)
- **THEN** please still runs the closing learn, passing an empty or nil LESSONS collection (learn handles this gracefully, no special-case code needed)

#### Scenario: Closing learn accesses collected lines

- **WHEN** the closing learn runs with a collection of [("LESSONS: none"), ("LESSONS: found bug in error handling"), ("LESSONS: review rejected X; the convention is Y" — a vault note written at its return already covers it), ("LESSONS: none")]
- **THEN** the learn skill judges each line independently: the first and fourth are empty/noop, the second is evaluated against the quality gate, and the third is skipped because a vault note written this session already covers it

### Requirement: Please gates do not block LESSONS collection

The gates in the please skill (documentation gate, review gate, final gate before closing learn) MUST NOT block or filter the LESSONS collection. Collection is unconditional; gates may filter the work being gated, but they do not filter what gets recorded.

#### Scenario: Gate rejects a subagent's output
- **WHEN** please's gate rejects a subagent's work (e.g., "documentation is missing")
- **THEN** the LESSONS line from that subagent is still collected (even though the work itself may be re-done or failed)

#### Scenario: Re-executed work after gate rejection
- **WHEN** a subagent re-executes work after a gate rejection and returns a new LESSONS line
- **THEN** please collects the new line; old lines from the rejected attempt are not carried forward (collect once per final dispatch, not per attempt)

