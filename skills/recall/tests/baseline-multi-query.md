# Baseline scenario — multi-query + dispatched-subagent-reads-members

A subagent is given a /recall task that, under the new SKILL.md, should:

1. Generate 5–15 Step 1 phrases without pre-filtering them.
2. Run one `engram query` invocation per Step 1 phrase (no collapse-to-one).
3. Union the items across query results (dedup by path, keep max score, union provenances). Clusters listed per-query, not merged. Hubs union-and-dedup.
4. For each cluster, decide gate-pass purely from cluster size (≥3) plus the rep's hint of coherence — and if pass, dispatch a synthesis subagent that reads ALL members from disk (the parent only read the rep).
5. When a synthesis subagent finds the binding principle is already stated in one member, write `--relation` bullets linking the other members to that anchor (link-to-bind), not skip the cluster.
6. NOT apply the recall-mirror test or "principle is generalizable" criterion at the recall layer — those are handled inside the dispatched subagent's /learn invocation.

## Scenario prompt (verbatim, give to subagent)

You are an agent working in a vault-backed memory system. The user has just asked:

> "I need to wire OpenCode session transcript reads alongside the existing Claude Code reader, behind a shared interface, so the marker advance works for both harnesses."

You have not done any work yet. You have the engram binary on PATH and a populated vault at the default location.

Read `/Users/joe/repos/personal/engram/skills/recall/SKILL.md` and follow its instructions exactly to run /recall against this ask. Do NOT actually run `engram query` — instead, print the exact `engram query` invocations you would issue (in a parallel tool-use block if the skill mandates parallelism), show your full thought process inline, and describe what you would do if the queries returned clusters.

Constraints for this run:
- Do not consult any other skill file.
- Use only what `skills/recall/SKILL.md` tells you.

## What we are measuring

A pass requires ALL of:

1. **Step 1 phrases are not pre-filtered.** No "would this be worth surfacing" gate; the agent just generates 5–15 phrases and moves on.
2. **One `engram query` per Step 1 phrase.** If the agent generated 8 phrases, they issue 8 `engram query` invocations (in a single parallel tool-use block).
3. **No collapse to "one or two phrases" before querying.** The skill must explicitly say multi-query is the default.
4. **Per-cluster gate uses size ≥3 + rep coherence**, not a recall-mirror test or "generalizable" test.
5. **Synthesis subagents are described as reading all members from disk.** The parent /recall does not synthesize inline.
6. **When binding principle is already stated in a member: link-to-bind via `--relation`**, not skip.

## Failure modes to watch for

- Agent issues only one `engram query` call (the old collapsed-phrase behavior).
- Agent applies the recall-mirror test to clusters at recall time.
- Agent applies "generalizable, not project-specific" criterion.
- Agent inline-synthesizes from cluster rep content without dispatching a subagent.
- Agent skips clusters when the binding principle is already stated (instead of link-to-bind).
- Agent invokes `engram recall` anywhere (the legacy command is gone).
