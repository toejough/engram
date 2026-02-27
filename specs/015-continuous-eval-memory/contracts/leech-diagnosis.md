# Contract: Leech Diagnosis Engine

**Spec**: FR-007, FR-008 | **Priority**: P3

## Interface

```go
// DiagnoseLeech analyzes a leech memory using surfacing_events data only (no API calls).
// Returns root cause analysis with proposed action. SuggestedContent is empty for
// content_quality diagnoses — use PreviewLeechRewrite to generate a preview.
DiagnoseLeech(db *sql.DB, memoryID int64) (*LeechDiagnosis, error)

// PreviewLeechRewrite generates a rewritten version of a leech memory's content.
// Called by the CLI after DiagnoseLeech, only for content_quality diagnoses.
// The preview is shown to the user alongside the diagnosis before they approve.
PreviewLeechRewrite(db *sql.DB, diagnosis LeechDiagnosis, llm LLMExtractor) (string, error)

// GetLeeches returns all memories currently classified as "leech" with leech_count >= threshold.
GetLeeches(db *sql.DB) ([]LeechCandidate, error)

// ApplyLeechAction executes a user-approved leech remediation action.
ApplyLeechAction(db *sql.DB, diagnosis LeechDiagnosis, fs FileSystem) error
```

## Data Types

### LeechCandidate

| Field | Type | Description |
|-------|------|-------------|
| MemoryID | int64 | Embedding ID |
| Content | string | Memory content |
| LeechCount | int | Consecutive low-impact surfacings |
| ImportanceScore | float64 | How often it's surfaced |
| ImpactScore | float64 | How effective when surfaced |
| SurfacingCount | int | Total times surfaced |

### LeechDiagnosis

| Field | Type | Description |
|-------|------|-------------|
| MemoryID | int64 | Embedding ID |
| Content | string | Memory content |
| DiagnosisType | string | Root cause category |
| Signal | string | Evidence description |
| ProposedAction | string | Recommended remediation |
| SuggestedContent | string | Rewritten content (for `rewrite` action only) |
| Recommendation | *Recommendation | Non-nil for non-memory actions (promote, convert_to_hook) |

## Behavior

### DiagnoseLeech

Analyzes surfacing_events history for the memory to determine root cause:

1. **Content quality** (`content_quality`):
   - Signal: High surfacing count + faithfulness consistently < 0.3 + agent response doesn't reference memory content.
   - Proposed action: `rewrite`. SuggestedContent left empty by DiagnoseLeech; populated by PreviewLeechRewrite before presentation to user.

2. **Wrong tier** (`wrong_tier`):
   - Signal: Surfacing timestamps consistently AFTER a user correction in the same session (memory surfaced too late to prevent the mistake).
   - Proposed action: `promote_to_claude_md`. Recommendation populated with: category "claude-md-promotion", description of what content to add and which section type it belongs in, evidence from surfacing history.

3. **Enforcement gap** (`enforcement_gap`):
   - Signal: Agent response references memory content (substring match) but user still corrects (agent understood the guidance but didn't follow it).
   - Proposed action: `convert_to_hook`. Recommendation populated with: category "hook-conversion", description of the rule to enforce, the trigger conditions (tool event, pattern), and enforcement behavior (warn vs block).

4. **Retrieval mismatch** (`retrieval_mismatch`):
   - Signal: Haiku `haiku_relevant=false` on >50% of surfacings (E5 matches broadly but content isn't relevant).
   - Proposed action: `narrow_scope` (re-embed with more specific content or restrict to certain contexts).

Priority order: wrong_tier > enforcement_gap > content_quality > retrieval_mismatch.
First matching diagnosis wins (most actionable first).

### PreviewLeechRewrite

Generates a rewritten version of the memory content for user review:

1. Only called for `content_quality` diagnoses (caller checks DiagnosisType).
2. Sends the original content + diagnosis signal to an LLM (Haiku) asking for a clearer, more actionable rewrite.
3. Returns the rewritten string. Caller populates `diagnosis.SuggestedContent` before presenting to the user.
4. If LLM is unavailable, returns empty string — diagnosis is still presentable without a preview.

### GetLeeches

Returns memories where `quadrant='leech'` AND `leech_count >= leech_threshold` (from metadata).

### ApplyLeechAction

Executes memory-internal actions directly. Non-memory actions are not executed — they produce Recommendations.

| Action | Type | Implementation |
|--------|------|---------------|
| `rewrite` | Memory-internal | Update memory content and re-embed |
| `narrow_scope` | Memory-internal | Re-embed with narrowed content |
| `promote_to_claude_md` | Recommendation | Return Recommendation; mark leech as "action_recommended" |
| `convert_to_hook` | Recommendation | Return Recommendation; mark leech as "action_recommended" |

For non-memory actions, ApplyLeechAction returns the Recommendation from the diagnosis. The CLI collects all Recommendations and offers to save them to a markdown file (see Recommendation Output below).

## Error Handling

| Condition | Behavior |
|-----------|----------|
| Memory has no surfacing events | Return diagnosis with "insufficient_data" type |
| LLM unavailable for preview rewrite | PreviewLeechRewrite returns empty string; diagnosis still valid |
| Memory doesn't exist | Return error |

## Recommendation Output

Non-memory recommendations are potentially disruptive (they suggest changes to CLAUDE.md, hooks, skills, etc.). The CLI MUST:

1. Print a summary of all Recommendations to stdout.
2. Offer to save the full details to a timestamped markdown file (e.g., `memory-recommendations-2026-02-20.md`).
3. Never assume any specific external tool exists — describe WHAT should happen, not which tool to use.

Recommendations do not reference specific tool names (no hookify, no claude-md-management). They describe the desired outcome in enough detail that any capable tool or human can execute them.

## Constraints

- DiagnoseLeech is pure data analysis — no API calls, no LLM dependency.
- PreviewLeechRewrite is the only LLM-dependent function; it's optional (diagnosis is complete without it).
- projctl NEVER writes to CLAUDE.md, hook configs, or skill files.
- Each leech diagnosis is presented as a proposal; no automatic remediation.
- Memory-internal actions (rewrite, narrow_scope) MUST execute within a single database transaction. If re-embedding fails, the transaction rolls back and the error is returned. The memory's original content and embedding are preserved.
