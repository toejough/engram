# Baseline scenario — upfront judgement, per-round progress, closing synthesis

A subagent is given a realistic non-trivial request, loaded with the current `skills/recall/SKILL.md`, and asked to perform recall against the agent-memory vault at `/Users/joe/repos/personal/agent-memory`.

## Scenario prompt (verbatim, give to subagent)

You are working on the `engram` repo at `/Users/joe/repos/personal/engram-worktrees/opencode-plugin`. The user has asked you to wire OpenCode session transcripts into the `engram transcript` pipeline alongside the existing Claude Code JSONL reader, including per-harness progress markers. Before you start, the user wants you to run the `recall` skill against the agent-memory vault.

Read `skills/recall/SKILL.md` from the working directory and follow its instructions exactly. The vault is at `/Users/joe/repos/personal/agent-memory`. The `engram` binary is on `PATH`. Produce the user-facing reply the skill prescribes.

## What we are measuring

For the **GREEN** version of the skill we want all five of these behaviors to appear in the subagent's output without prompting:

1. **Upfront judgement print** — before any retrieval, the agent states: what the user asked, the situation, and the plan it would take absent any recalled guidance.
2. **Initial frontier query** — runs `engram recall` (anchors) and `engram recall --recent` once.
3. **Cascade with follow-up** — for every note that scored relevant, the agent makes a follow-up `engram recall --follow ... --already-read ...` call, and keeps going until the budget is exhausted or the frontier is empty.
4. **Per-round progress line** — after each cascade round, one line of the form `N relevant / M irrelevant / P budget left / Q next links to follow`.
5. **Closing synthesis** — a short paragraph that names, plainly, how the surfaced memories did or did not change the plan.

For the **RED** baseline we expect the current skill to miss at least items 1, 4, and 5. Item 3 is the one we already know fails in practice — agents read the initial frontier and stop.

## Capture format

Save observations in `baseline-RED-results.md` (current skill) and `baseline-GREEN-results.md` (updated skill). For each of the five behaviors above, record: present / partial / absent, with a one-line excerpt from the subagent's reply as evidence.
