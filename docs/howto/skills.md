# Engram Skills Reference

## /recall — Load Previous Session Context

Resume work by loading context from previous sessions in the same project.

**Two modes**:
- `/recall` (no args) — returns raw transcript content from recent sessions. The agent reads it and presents a concise summary: what was decided, what got done, what's outstanding, what state things were left in.
- `/recall <query>` — uses Haiku to extract content relevant to the query from each session transcript.

**Example interaction**:

```
User: /recall
Agent: Here's what happened last session:
  - Implemented consolidation confidence threshold (#403)
  - Fixed duplicate proposal bug (#404)
  - Triage: removed 9 stale memories, narrowed 8 keyword sets
  - State: clean tree, all tests passing, pushed to main
```

```
User: /recall adaptation pipeline
Agent: From last 3 sessions on adaptation:
  - Session 12: wired adaptation config from policy.toml into analysis pipeline
  - Session 11: added functional options for threshold overrides
  - Session 10: initial adaptation proposal generation from feedback patterns
  - Outstanding: auto-apply after consecutive approvals not yet implemented
```

## /adapt — Manage Adaptation Policies

Review system-level adaptation proposals that engram generates from feedback patterns. Proposals suggest changes like de-prioritizing noisy keywords during extraction.

**Workflow**:
1. Run `/adapt` to see pending proposals grouped by dimension (extraction, surfacing, maintenance)
2. Each proposal shows what it would change and the evidence (e.g., "de-prioritize keyword 'toml': 93% irrelevance rate")
3. Approve, reject, or retire individual proposals
4. After 3+ consecutive approvals in a dimension, option to enable auto-apply

**Example interaction**:

```
User: /adapt
Agent: 22 pending proposals:
  - pol-003 [extraction] de-prioritize "toml": 93% irrelevance across 14 memories
  - pol-021 [extraction] de-prioritize "readme": 100% irrelevance across 16 memories
  ...
  Approve all? Or review individually?

User: approve the 100% ones, reject the rest
Agent: Approved pol-021 (readme), pol-011 (recommendation), pol-017 (branch)...
       Rejected pol-003 (toml), pol-009 (jsonl)...
```

```
User: /adapt
Agent: 3 pending proposals:
  - pol-025 [surfacing] boost keyword weight for "parallel-test": surfaced 2x but
    effectiveness 94%
  - pol-026 [extraction] de-prioritize "error": appears in 40% of memories, low signal
  - pol-027 [maintenance] lower consolidation threshold for "git-workflow" cluster:
    5 near-duplicates detected

  You've approved 4 consecutive extraction proposals. Enable auto-apply for extraction?

User: yes, and approve pol-025
Agent: Auto-apply enabled for extraction dimension.
       Approved pol-025 (parallel-test keyword boost).
       pol-027 still pending.
```

## /memory-triage — Interactive Memory Maintenance

Review and act on maintenance signals — memories that need attention based on effectiveness data.

**Three categories surfaced**:
1. **Noise removal** — rarely surfaced + low effectiveness. Recommended for deletion.
2. **Hidden gem broadening** — high effectiveness but rarely surfaced. Keywords need expanding.
3. **Graduation candidates** — memories ready to promote to skills, CLAUDE.md, or rules.

Categories are presented one at a time. For each, the agent groups by theme, shows counts and examples, and recommends a bulk action. User approves, modifies, or skips.

**Example interaction**:

```
User: /memory-triage
Agent: ## Noise Removal (6 candidates)
  Stale project-specific memories (4):
    - "traced-spec-migration-cleanup" — 0% effectiveness
    - "registry-commands-stale" — completed work
    - "old-ci-workaround" — CI config since replaced
    - "temp-debug-flag" — debugging artifact
  Duplicate of CLAUDE.md rules (2):
    - "always-use-targ" — already in CLAUDE.md
    - "no-direct-go-test" — redundant with targ rule
  Remove all 6?

User: yes
Agent: Removed 6 memories.

  ## Hidden Gems (3 candidates)
  High-impact but under-surfaced:
    - "parallel-test-fixture-isolation" — 100% effectiveness, surfaced 1x
      Current keywords: [parallel, fixture]
      Suggested additions: [t.Parallel, shared state, test isolation]
    - "error-wrap-context" — 90% effectiveness, surfaced 2x
      Current keywords: [error, wrap]
      Suggested additions: [fmt.Errorf, sentinel, context]
    - "di-interface-boundaries" — 95% effectiveness, surfaced 1x
      Current keywords: [dependency injection]
      Suggested additions: [interface, mock, wire, boundary]
  Broaden all 3?

User: broaden the first two, skip di one
Agent: Broadened keywords for "parallel-test-fixture-isolation" and "error-wrap-context".
       Skipped "di-interface-boundaries".

  ## Graduation Candidates (1 candidate)
    - "always-check-vcs-type" — 100% effectiveness, surfaced 8x, applies across all projects
      Recommend: promote to CLAUDE.md
  Promote?

User: already there actually, just remove the memory
Agent: Removed "always-check-vcs-type".
       Triage complete. 8 removed, 2 broadened, 0 promoted.
```

**Available actions** (all via CLI, never direct file edits):
- Remove: `engram apply-proposal --action remove --memory <path>`
- Broaden keywords: `engram apply-proposal --action broaden_keywords --memory <path> --keywords "kw1,kw2"`
- Rewrite: `engram apply-proposal --action rewrite --memory <path> --fields '{"title":"...","principle":"..."}'`
- Consolidate: `engram apply-proposal --action consolidate --memory <survivor-path> --fields '{"members":[...]}'`
