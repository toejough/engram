# Route Capability

## ADDED Requirements

### Requirement: Dispatch handoff checklist mandates LESSONS line in subagent completion report

When the route skill selects a subagent type and constructs the dispatch instructions, the completion-report section MUST include an explicit instruction that every completion report ends with a `LESSONS:` line. The instruction format is prescriptive: the subagent MUST include the line, not "should" or "may", and MUST follow the lessons-contract format (`LESSONS: none` or `LESSONS: <lesson1>, <lesson2>, <lesson3>`).

The mandate lives in `agent-instructions/skills/route/SKILL.md`, in the "The handoff is the unlock" section — the existing MUST-hand checklist every dispatch obeys (exact files, acceptance checks, do-NOT-touch bounds, recall-first instruction). The edit adds a new bullet to that checklist: the completion-report instruction, including the LESSONS line mandate. It applies to all subagent types (fork, fresh-context, workflow) because that checklist already governs every dispatch. No special cases.

#### Scenario: Route dispatches a fresh-context subagent
- **WHEN** route selects a subagent type (e.g., "general-purpose" agent) and builds the dispatch instructions
- **THEN** the completion-report section includes text like "Your final report MUST end with a LESSONS: line..."

#### Scenario: Route dispatches a fork subagent
- **WHEN** route dispatches a fork (subagent_type: "fork")
- **THEN** the fork's instructions still include the LESSONS mandate (even though the fork inherits session context, the mandate is explicit in writing)

#### Scenario: Dispatch with detailed task instruction
- **WHEN** route constructs instructions for a complex multi-step task (e.g., "design a feature")
- **THEN** the task description and acceptance criteria are followed by the completion-report instruction, including the LESSONS line mandate

### Requirement: Route enforces re-ask for missing LESSONS line

If a subagent's completion report does not include a `LESSONS:` line, the orchestrator (the agent executing route's dispatch loop — route is a skill the orchestrator follows, not a separate actor) MUST trigger exactly one re-ask, passing a targeted follow-up to the subagent with a reminder to include the LESSONS line. The re-ask is not a retry of the entire task — it is a targeted request to add the missing line. This is the same single-re-ask rule stated in the lessons-contract spec; route's SKILL.md carries the instruction text the orchestrator follows.

#### Scenario: Subagent returns report without LESSONS line
- **WHEN** the orchestrator receives a subagent completion report that ends without a LESSONS line
- **THEN** the orchestrator re-asks the subagent exactly once: "Your report is missing the LESSONS line — please add it in the format LESSONS: <lesson1>, <lesson2> or LESSONS: none"

#### Scenario: Subagent responds to re-ask
- **WHEN** the subagent provides a LESSONS line in response to the re-ask
- **THEN** the orchestrator treats this line the same as an on-time line (adds it to the session's collection)

#### Scenario: No infinite re-ask loop
- **WHEN** the subagent has been re-asked once and still does not provide a LESSONS line
- **THEN** the orchestrator does not re-ask again; it notes the line as missing and proceeds (collecting an empty slot for that subagent)
