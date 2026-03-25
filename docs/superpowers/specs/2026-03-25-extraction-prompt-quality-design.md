# Extraction Prompt Quality Improvements (#379, #380, #381)

## Problem

The extraction prompt in `internal/extract/extract.go:systemPrompt()` produces low-quality memories in three ways:

1. **Tasks stored as principles (#379)** — Completed one-time actions ("remove --data-dir flag from hooks") and common knowledge ("test both boolean branches") pass the quality gate.
2. **Specific instances not generalized (#380)** — "Persist surfacing queries in irrelevant_queries field" stored instead of the transferable principle "capture diagnostic context at the point of observation."
3. **Generic keywords (#381)** — Keywords like `git log`, `boolean flags`, `TOML` match nearly any session, causing memories to surface where they're irrelevant.

## Design

All three fixes are additions to the `systemPrompt()` text. No pipeline code changes needed — the quality gate is enforced by the LLM at extraction time.

### 1. Task vs principle filter (#379)

Add to QUALITY GATE section:

```
- one-time tasks or completed actions (e.g., "remove the --data-dir flag," "file an issue about X,"
  "clean up the hooks"). If the user said "do X" and X has a completion state, it is a task, not a
  reusable principle. Do not extract it.
- common knowledge any competent developer already knows (e.g., "test both branches of a boolean,"
  "handle errors," "use descriptive names"). If the principle would appear in an introductory
  course or tutorial, the model already knows it — skip it.
```

### 2. Generalize before storing (#380)

Add new section after EXTRACT:

```
GENERALIZE — before storing, restate each learning at its most transferable level:
- Strip project-specific details (file names, variable names, tool names) unless they ARE the point.
- Ask: "What is the underlying principle that makes this correct?" State that, not the specific instance.
- Example: "persist surfacing queries in irrelevant_queries field" → "capture diagnostic context at
  the point of observation for later analysis, not after the fact."
- If the generalized form is identical to an existing well-known principle, score generalizability
  lower or reject entirely.
```

No pipeline-level dedup change — the existing keyword-overlap dedup (>50%) catches exact duplicates. The prompt-level generalization reduces the rate of near-duplicates that slip through with different keywords.

### 3. Context-specific keywords (#381)

Add keyword guidance to the JSON schema section, replacing the bare example:

```
"keywords": ["context-specific", "trigger", "phrases"],
  // Keywords should match the SITUATION where this principle is needed, not just the subject area.
  // BAD: "git log", "boolean", "testing", "UI" — these are domain terms that match too broadly.
  // GOOD: "post-migration verification", "parallel-agent id collision", "algorithm-exposed controls"
  //   — these describe the specific context where the principle helps.
  // Ask: "What would someone be doing when they need this memory?" Use those activity-level terms.
```

Note: JSON doesn't support comments, so this guidance goes in a paragraph before the JSON example, not inline.

## Files changed

| File | Change |
|------|--------|
| `internal/extract/extract.go` | Add 3 blocks to `systemPrompt()` return string |
| `internal/extract/extract_test.go` | Add/update tests verifying prompt contains new guidance |

## Testing

The prompt is a string constant. Tests verify the prompt text contains the expected guidance strings. No behavioral tests needed — the LLM's interpretation of the prompt is validated by the memory quality observed over time (which is what surfaced these issues).

## Not in scope

- Pipeline-level semantic dedup (#380 mentions it but the prompt-level fix is sufficient for now)
- Changing the generalizability scoring scale
- Modifying the keyword filtering pipeline in `keyword/filter.go`
