## MODIFIED Requirements

### Requirement: Orchestrator collects LESSONS lines from all dispatched work

As the please runbook set drives dispatched work steps, the orchestrator following it MUST collect the `LESSONS:` line from every subagent's completion report at the moment each dispatch resolves. Dispatches occur throughout the please runbook's seven-step workflow (chiefly step 4 "Execute" and the adversarial-review-gates sub-runbook); the runbook instructs the orchestrator to record each returned LESSONS line as part of handling every dispatch result, wherever in the workflow it lands. The orchestrator records the raw line exactly as written by the subagent, whether it says "none" or contains lessons, for every dispatch in the session. At the same moment, the orchestrator judges each lesson per the ambient learn guidance's return cue: a lesson that clears learn's bar is captured with `/learn` on the fast path before the next dispatch, and its entry in the running list is marked as captured. Judging never removes or rewrites a line; the list stays complete and verbatim, with captured marks added.

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
- **THEN** please records the line verbatim, the orchestrator captures it with a fast-path `/learn` before the next dispatch, and the running-list entry is marked as captured

### Requirement: Please passes collected LESSONS lines to closing learn

At workflow step 7 ("Capture (close) — `/learn`" in the please runbook, with its lessons-audit sub-runbook), the please runbook set MUST pass the collected LESSONS lines as input to the learn skill, alongside the existing lessons-audit output that step already hands over. This is explicit handoff — learn receives the full set of lines, with the entries captured at return marked as such, and judges the unmarked lines against the quality gate.

#### Scenario: Closing learn receives LESSONS collection

- **WHEN** the session's dispatched work is complete and please is preparing to run the closing learn
- **THEN** please includes the collected LESSONS lines, with their captured marks, in the learn's input context (e.g., passing them as a `lessons_from_session: [line1, line2, ...]` entry or inline text)

#### Scenario: No dispatches in session

- **WHEN** the session has no delegated work (e.g., user asked a question, coordinator answered directly)
- **THEN** please still runs the closing learn, passing an empty or nil LESSONS collection (learn handles this gracefully, no special-case code needed)

#### Scenario: Closing learn accesses collected lines

- **WHEN** the closing learn runs with a collection of [("LESSONS: none"), ("LESSONS: found bug in error handling"), ("LESSONS: review rejected X; the convention is Y" — captured at return), ("LESSONS: none")]
- **THEN** the learn skill judges each unmarked line independently: the first and fourth are empty/noop, the second is evaluated against the quality gate, and the captured third is skipped
