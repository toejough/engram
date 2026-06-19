# Baseline scenario — single unified query + inline Step 2.5

A subagent is given a /recall task and asked to follow the current SKILL.md. Under the current
model the agent must:

1. Generate 5–15 Step 1 phrases without pre-filtering them.
2. Run **one** `engram query --synthesize-l2` with all phrases as repeatable `--phrase` flags
   (NOT one call per phrase, NOT multiple parallel calls).
3. After the query, activate any `activated: true` items in one batched `engram activate` call.
4. For each cluster in the payload, run Step 2.5 **inline**: read `candidate_l2s` notes via
   `engram show`, judge coverage (covered/near/absent), and issue the appropriate write.
5. Wait for each write before moving to the next cluster.

## Scenario prompt (verbatim, give to subagent)

You are working on the `engram` repo at `/Users/joe/repos/personal/engram`. The user has asked:

> "I need to wire OpenCode session transcript reads alongside the existing Claude Code reader,
> behind a shared interface, so the marker advance works for both harnesses."

You have not done any work yet. You have the `engram` binary on PATH and a populated vault at
the default location.

Read `/Users/joe/repos/personal/engram/skills/recall/SKILL.md` and follow its instructions
exactly to run recall against this ask. Do NOT actually run `engram query` — instead:
- Print your Step 0 judgement (ask / situation / plan).
- Print the exact `engram query --synthesize-l2 ...` invocation you would issue (one call,
  all phrases as `--phrase` flags).
- Describe what you would do if the query returned clusters with `candidate_l2s`.

Constraints:
- Do not consult any other skill file.
- Use only what `skills/recall/SKILL.md` tells you.

## What we are measuring

A pass requires ALL of:

1. **Step 0 printed upfront.** Ask / situation / plan stated before any retrieval action.
2. **One `engram query --synthesize-l2` call** with all phrases. NOT one call per phrase;
   NOT multiple parallel calls. The `--synthesize-l2` flag is present.
3. **No collapse to fewer phrases before querying.** The agent generates the full 5–15 and
   passes them all.
4. **No `--tier`, `--vault`, or `--chunks-dir` flags** on the query.
5. **Step 2.5 described as inline.** The agent would read `candidate_l2s` notes via
   `engram show` (within this same agent's tool calls), judge coverage, and write — NOT
   dispatch a synthesis subagent, NOT skip crystallization.
6. **One write per cluster.** The agent describes issuing one `engram learn` or `engram amend`
   per cluster (or explaining why it is no-op), not batching clusters or writing multiple notes.

## Failure modes to watch for

- Agent issues one `engram query` call per phrase (the old per-phrase model).
- Agent omits `--synthesize-l2` from the query.
- Agent adds `--tier L2` or `--tier L3` to the query.
- Agent describes dispatching a synthesis subagent for Step 2.5.
- Agent skips Step 2.5 ("the chunks are enough") or skips clusters because "nothing conflicts."
- Agent describes writing multiple L2 notes for a single cluster.
- Agent invokes `engram recall` anywhere (the legacy command is gone).
