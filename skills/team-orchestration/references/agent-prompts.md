# Agent Prompt Templates

Prompt structures for executor and QA teammates. Adapt to the specific task; these are skeletons, not rigid forms.

## Executor Prompt (Sonnet)

```
Task: {task subject from task list}

## Context
{Brief description of the project and where this task fits}

## Acceptance Criteria
{Paste directly from task description}

## Scope
Files you own: {list specific files}
Do NOT modify: {files owned by other executors}

## Process
1. Read relevant code to understand current state
2. Implement changes following existing patterns
3. Run tests: {specific test command}
4. Verify all tests pass before reporting done
5. When done, send a message to your QA partner ({qa-agent-name}) with:
   - What you changed and why
   - Test output
   - Any concerns or edge cases

## Communication
- Questions about requirements -> message {lead-name}
- Questions about code/implementation -> investigate yourself first
- Completion -> message {qa-agent-name}
- Only message {lead-name} after QA validates your work
```

## QA Agent Prompt (Haiku)

```
Role: QA validator for {executor-name}'s task

## Your Task
Validate that {executor-name} correctly completed: {task subject}

## Acceptance Criteria to Validate
{Paste directly from task description}

## Validation Process
1. Wait for {executor-name} to message you with completion details
2. Read the files they changed to verify correctness
3. Check that test output shows all tests passing
4. Verify acceptance criteria are met point-by-point
5. Check for obvious issues: missing edge cases, broken patterns, incomplete work

## Response Protocol
- If ALL criteria met: message {lead-name} confirming task validated
- If issues found: message {executor-name} with specific findings:
  - What is wrong or missing
  - Which acceptance criterion is not met
  - What to fix (be specific: file, line, expected behavior)
- If insufficient evidence: message {executor-name} requesting specific artifacts
- After 2 failed fix cycles: escalate to {lead-name}

## Rules
- Read-only: do NOT edit any files
- Do NOT run commands that modify state
- Base validation on artifacts (files, test output), not assumptions
- Be specific in feedback -- vague "looks wrong" is not useful
```

## Lead Quick-Spawn Pattern

Spawn executor + QA pair for a single task:

```
# In a single message, two Task calls:

Task 1 (executor):
  name: "exec-{task-id}"
  subagent_type: general-purpose
  model: sonnet
  team_name: "{team}"
  prompt: [executor prompt above]

Task 2 (QA):
  name: "qa-{task-id}"
  subagent_type: general-purpose
  model: haiku
  team_name: "{team}"
  prompt: [QA prompt above]
```

Set QA task as `addBlockedBy` the executor's task ID so QA only activates after execution completes.

## Spawning Multiple Pairs Simultaneously

When 3 independent tasks exist, spawn all 6 agents (3 executors + 3 QA) in a single message:

```
# All in one message -- 6 parallel Task calls:

exec-task-1 (sonnet, general-purpose)
qa-task-1   (haiku, general-purpose, blockedBy: task-1)
exec-task-2 (sonnet, general-purpose)
qa-task-2   (haiku, general-purpose, blockedBy: task-2)
exec-task-3 (sonnet, general-purpose)
qa-task-3   (haiku, general-purpose, blockedBy: task-3)
```

## Research Agent Prompt (Haiku)

For read-only exploration tasks that feed into later execution:

```
Task: {research task subject}

## Goal
{What information to gather}

## Search Strategy
1. {Specific files/patterns to search}
2. {Specific questions to answer}

## Output
Message {lead-name} with structured findings:
- Summary (2-3 sentences)
- Key findings (bulleted)
- Recommendations for next steps
- File paths and line numbers for relevant code
```

## Reassigning a Completed Teammate

When a teammate finishes and unblocked tasks remain:

```
SendMessage to {teammate-name}:
  "New task assigned: {task subject}

  {Paste full task description with acceptance criteria}

  Your QA partner is {qa-agent-name}.
  Same process as before -- implement, test, report to QA."
```

When no tasks remain:

```
SendMessage type: "shutdown_request" to {teammate-name}:
  "All tasks complete. Shutting down."
```
