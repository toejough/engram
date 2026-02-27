# Contract: CLAUDE.md Quality Gate & Scoring

**Spec**: FR-010, FR-011, FR-013, FR-014, FR-015 | **Priority**: P4

## Interface

```go
// ProposeClaudeMDChange evaluates a memory for CLAUDE.md promotion and returns a proposal.
ProposeClaudeMDChange(db *sql.DB, memoryID int64, fs FileSystem, llm LLMExtractor) (*CLAUDEMDProposal, error)

// ScoreClaudeMD evaluates the quality of a CLAUDE.md file.
ScoreClaudeMD(claudeMDPath string, db *sql.DB, fs FileSystem, llm LLMExtractor) (*CLAUDEMDScore, error)

// EnforceClaudeMDBudget checks line count and proposes demotions if over budget.
EnforceClaudeMDBudget(claudeMDPath string, db *sql.DB, fs FileSystem) ([]CLAUDEMDProposal, error)

// ParseClaudeMDSections parses a CLAUDE.md file into typed sections.
ParseClaudeMDSections(content string) ([]CLAUDEMDSection, error)
```

## Data Types

### CLAUDEMDProposal

| Field | Type | Description |
|-------|------|-------------|
| Action | string | "add", "update", "remove" |
| Section | string | Suggested target section (e.g., "Commands", "Gotchas") |
| Content | string | Recommended content |
| SourceMemoryID | int64 | Memory that triggered this proposal |
| Reason | string | Why this change is proposed |
| QualityChecks | map[string]bool | Pass/fail for each gate |
| Recommendation | Recommendation | Describes what to do without naming specific tools |

### CLAUDEMDScore

| Field | Type | Description |
|-------|------|-------------|
| ContextPrecision | float64 | Are entries actionable? (0-100) |
| Faithfulness | float64 | Do entries change behavior? (0-100) |
| Currency | float64 | Do commands/paths still work? (0-100) |
| Conciseness | float64 | Line count, filler, redundancy (0-100) |
| Coverage | float64 | High-impact universal memories represented? (0-100) |
| OverallGrade | string | Letter grade A-F |
| OverallScore | float64 | Weighted composite (0-100) |
| Issues | []string | Specific problems found |

### CLAUDEMDSection

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Section heading |
| Type | string | "commands", "architecture", "gotchas", "code_style", "testing", "other" |
| Content | string | Section body text |
| LineCount | int | Lines in this section |

## Behavior

### ProposeClaudeMDChange

Quality gate checks (all must pass for "add" proposals):

1. **Working knowledge**: Memory quadrant must be "working".
2. **Universal**: Surfaced across 3+ projects (from pre-existing `projects_retrieved` column on embeddings table).
3. **Actionable**: Haiku evaluates "Can Claude follow this instruction?" (yes/no).
4. **Non-redundant**: No existing CLAUDE.md entry, hook, or skill covers this content (similarity check).
5. **Right tier**: Would a hook or skill be more appropriate? (Haiku evaluation).

If all pass:
- Classify content into section type (Haiku: "Is this a command, gotcha, architecture note, code style rule, or testing pattern?")
- Return proposal with quality check results, suggested section, and recommended content.
- Recommendation describes what to add, which section type it belongs in, and the formatted content. Does not name any specific tool.

If any fail: return nil proposal with reason.

### ScoreClaudeMD

Weights: Context Precision 20%, Faithfulness 25%, Currency 20%, Conciseness 15%, Coverage 20%.

| Dimension | Measurement |
|-----------|-------------|
| Context Precision | Haiku evaluates each entry: "Is this actionable guidance?" |
| Faithfulness | Average effectiveness score from surfacing_events for promoted memories |
| Currency | Check commands exist (`which`), file paths exist (`os.Stat`), package names valid |
| Conciseness | Line count vs budget, filler word detection, redundancy with skills/hooks |
| Coverage | % of "working knowledge" universal memories present in CLAUDE.md |

Grade scale: A (90+), B (80-89), C (70-79), D (60-69), F (0-59).

### EnforceClaudeMDBudget

1. Parse CLAUDE.md line count.
2. If over budget (default: 100 lines):
   - Score each entry using effectiveness from surfacing_events.
   - Sort by effectiveness ascending.
   - Propose removing lowest-effectiveness entries until under budget.
   - Each demotion proposal includes a Recommendation describing:
     - WHAT to remove from CLAUDE.md and WHY (low effectiveness data)
     - WHERE the content should go based on its characteristics:
       - Narrow/domain-specific content → "convert to a skill covering [topic]"
       - Low-impact content → "demote to memory tier" (projctl-internal, no recommendation needed)
       - Enforcement-pattern content → "convert to a deterministic hook that enforces [rule]"
     - No specific tool names — just the desired outcome.

### ParseClaudeMDSections

Recognizes markdown heading structure and classifies sections by content patterns:
- Tables with command columns → "commands"
- Directory tree diagrams → "architecture"
- Bullets with **NEVER**/**ALWAYS** → "gotchas"
- Short code-style rules → "code_style"
- Test-related commands/patterns → "testing"
- Everything else → "other"

## Error Handling

| Condition | Behavior |
|-----------|----------|
| CLAUDE.md doesn't exist | Return score with all zeros + "file not found" issue |
| LLM unavailable | Skip Haiku-dependent checks, score only filesystem-based dimensions |
| No surfacing data for promoted memories | Faithfulness = 0, flag as "insufficient data" |

## Recommendation Output

All non-memory proposals (promotions, demotions to hook/skill) produce Recommendations. The CLI MUST:

1. Print a summary of all Recommendations to stdout.
2. Offer to save full details to a timestamped markdown file (e.g., `memory-recommendations-2026-02-20.md`).
3. Never name specific external tools — describe the desired outcome so any capable tool or human can act on it.

Example Recommendation categories from this contract:
- `claude-md-promotion`: "Add this entry to the gotchas/warnings section of CLAUDE.md: [content]. Evidence: working knowledge across 4 projects, effectiveness 0.87."
- `claude-md-demotion-to-hook`: "Remove this entry from CLAUDE.md and create a deterministic hook that enforces [rule] on [event]. Evidence: low effectiveness (0.12), enforcement pattern detected."
- `claude-md-demotion-to-skill`: "Remove this entry from CLAUDE.md and create a skill covering [topic]. Evidence: narrow/domain-specific, only relevant to [project type]."
- `skill-merge`: "These N memories/entries cover overlapping patterns around [topic]. Consider consolidating into a single skill."

## Constraints

- projctl NEVER writes to CLAUDE.md, hook configs, or skill files. All changes are Recommendations.
- ScoreClaudeMD is called during `optimize --review`, not on every session.
- Line budget is configurable via metadata table (default: 100).
