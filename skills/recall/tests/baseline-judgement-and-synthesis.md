# Baseline scenario — upfront judgement, single query, closing synthesis

A subagent is given a realistic non-trivial request, loaded with the current `skills/recall/SKILL.md`, and asked to perform recall against the agent-memory vault.

## Scenario prompt (verbatim, give to subagent)

You are working on the `engram` repo at `/Users/joe/repos/personal/engram`. The user has asked you to wire OpenCode session transcripts into the `engram ingest` pipeline alongside the existing Claude Code JSONL reader, including per-harness progress markers. Before you start, the user wants you to run the `recall` skill.

Read `skills/recall/SKILL.md` from the working directory and follow its instructions exactly. The vault is at the default location resolved by the binary. The `engram` binary is on `PATH`. Produce the user-facing reply the skill prescribes.

Do NOT actually run `engram ingest` or `engram query` — instead:
- Print your Step 0 judgement (ask / situation / plan).
- Print the exact `engram query ...` invocation you would issue.
- Describe what you would do in Step 2.5 if the query returned clusters with `candidate_l2s`.
- Print the Step 3 closing synthesis structure (confirming/adjusting/contradicting each plan step).

## What we are measuring

For the **GREEN** version of the skill we want ALL of these behaviors to appear without prompting:

1. **Step 0 printed upfront** — before any retrieval call, the agent states: what the user asked, the situation, and the plan it would take absent any recalled guidance. Three labeled blocks (Ask / Situation / Plan). This must appear BEFORE the sweep or query.
2. **Step 0.5 sweep** — `engram ingest --auto` called first.
3. **Exactly ONE `engram query` call** with all ~10 phrases as repeatable `--phrase` flags. No `--synthesize-l2`, no `--tier`, no `--vault`, no `--chunks-dir`. Not one call per phrase.
4. **No cascade / no `--follow` calls.** There is no `engram recall --follow`. No per-round progress lines. One query, done.
5. **Step 2.5 described as inline** — agent reads `candidate_l2s` notes via `engram show`, judges covered/near/absent, issues one write per cluster (or states why it is no-op).
6. **Step 3 closing synthesis** — walks the Step 0 plan in order; for each action says confirmed / adjusted / contradicted / silent, with one-line reason. Opens with the count ("Query surfaced N items (K chunks, M notes); crystallized J lessons.").

## Failure modes to watch for

- Agent skips Step 0 and goes straight to retrieval.
- Agent runs `engram recall` (the legacy command is gone).
- Agent runs `engram recall --follow` (cascade is gone).
- Agent runs one `engram query` call per phrase (the old per-phrase model).
- Agent adds `--vault`, `--chunks-dir`, `--synthesize-l2`, or `--tier` to the query.
- Agent emits per-round progress lines ("N relevant / M irrelevant / P budget left").
- Agent dispatches a subagent for Step 2.5 synthesis.
- Agent ends with a bullet recap of surfaced notes instead of walking the Step 0 plan.

## Capture format

Save observations in `baseline-RED-results.md` (current skill) and `baseline-GREEN-results.md` (updated skill). For each of the six behaviors above, record: present / partial / absent, with a one-line excerpt from the subagent's reply as evidence.
