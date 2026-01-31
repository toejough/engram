---
name: design-infer
description: Infer design decisions from interface analysis
user-invocable: false
---

# Design Infer

Infer design decisions from an existing codebase by analyzing user-facing interfaces, interaction patterns, and output formatting. Used by `/project adopt` for codebase adoption.

## When Invoked

This skill is dispatched by `/project` orchestrator with a context file containing:
- Project directory and config paths
- Mode: "infer", "update", or "create"
- Requirements summary from pm-infer
- Existing artifact summaries

## Modes

- **infer**: Create design.md from scratch by analyzing interfaces
- **update**: Add new design decisions to existing file, preserve existing IDs
- **create**: Create project-level design.md when only feature-specific ones exist
- **normalize**: Convert existing file to standard format (table → header)

## Analysis Focus

Design focuses on **user interaction**, not implementation:
- How users invoke commands
- What users see in output
- How errors are communicated
- Information hierarchy and formatting

## Analysis Order

1. **Existing design.md** - Preserve existing DES-NNN items
2. **CLI --help formatting** - Help text layout, structure
3. **Error message patterns** - How errors are presented
4. **Output formatting** - Tables, JSON, colors, progress
5. **Interactive prompts** - Questions, confirmations

## Workflow

### 1. Read Context

```bash
projctl context read --dir <project-dir> --skill design-infer
```

Extract:
- `invocation.mode` - "infer" or "update"
- `config.*_path` - Artifact locations
- `inputs.requirements_summary` - From pm-infer

### 1a. Parse Requirements for Tracing

Read requirements.md and extract all REQ-NNN IDs with their descriptions:

**Header format:**
```markdown
### REQ-001: Title
Description text...
```

**Table format:**
```markdown
| REQ-001 | Capability Name | Description |
```

Build a map of REQ-ID → description for use when creating traces_to fields.

### 2. Preserve Existing Design

If existing design.md:
- Parse existing DES-NNN items
- Keep these as-is
- Only add new decisions with higher IDs

### 3. Analyze CLI Help Format

Examine `--help` output structure:
- Header/banner style
- Command grouping
- Flag documentation format
- Examples section format

```
DES-001: Help text uses two-column layout for flags
DES-002: Commands grouped by category with headers
DES-003: Examples section at bottom of help
```

### 4. Analyze Error Patterns

Look at how errors are communicated:
- Error message format
- Exit codes
- Suggestions/hints
- Stack trace handling

```
DES-004: Errors prefixed with "Error: " in red
DES-005: Suggestions shown after error message
DES-006: No stack traces in production mode
```

### 5. Analyze Output Formatting

Examine command output:
- Table formatting
- JSON output option
- Color usage
- Progress indicators

```
DES-007: Tables use ASCII box drawing
DES-008: --json flag for machine-readable output
DES-009: Progress shown as spinner with status text
```

### 6. Analyze Interaction Patterns

Look for interactive elements:
- Confirmation prompts
- Input validation feedback
- Multi-step wizards

```
DES-010: Destructive actions require confirmation
DES-011: Invalid input shows inline error and re-prompts
```

### 7. Generate Design Document

```markdown
---
project: <project-name>
generated: <timestamp>
source: design-infer
---

# Design

## Design Principles

### DES-PRIN-001: <Principle Name>

<Description of guiding principle>

## CLI Interface

### DES-001: Help Text Layout

<Description of help formatting>

**Source:** `--help` output analysis
**Traces to:** REQ-NNN (if applicable)

## Output Formatting

### DES-005: Table Display

<Description of table formatting>

## Error Handling

### DES-008: Error Message Format

<Description of error presentation>

## Interactive Elements

### DES-010: Confirmation Prompts

<Description of confirmation pattern>
```

### 8. Create Traceability Links

Traceability is embedded directly in the design.md file via `**Traces to:**` fields. There is NO separate traceability.toml file.

When generating design.md, ensure each DES-NNN section includes:

```markdown
### DES-003: Error Message Format

Description...

**Traces to:** REQ-001, REQ-002
```

**Important:** Links exist only in the artifact files. Run `projctl trace validate` to verify all links are valid.

### 9. Identify Escalations

Escalate for:
- Inconsistent patterns (different error formats in different commands)
- Missing user feedback (silent operations)
- Unclear interaction flow

```toml
[[escalations]]
id = "ESC-003"
category = "design"
context = "Analyzing error output"
question = "Some errors use stderr, others stdout. Is this intentional?"
suggested_answer = "Likely inconsistency - recommend standardizing to stderr"
```

### 10. Write Result

```toml
[result]
skill = "design-infer"
status = "needs_escalation"
timestamp = 2024-01-15T12:00:00Z

[result.summary]
text = "Inferred 8 design decisions. Created 12 traceability links. 1 escalation pending."

[outputs.design]
path = "docs/design.md"
action = "created"
items_count = 8
id_range = "DES-001 to DES-008"

[outputs.traceability]
links_created = 12

[[escalations]]
id = "ESC-003"
category = "design"
context = "Error output"
question = "Inconsistent error handling - intentional?"

[context_for_next]
[context_for_next.artifacts]
design_summary = "8 design decisions covering CLI, output, errors..."
```

## Mode: Update

When `invocation.mode = "update"`:
- Only add new design decisions
- Flag potentially stale patterns
- Never modify existing DES-NNN

## Mode: Create

When `invocation.mode = "create"` (from `/project align` backfill):

Used when design.md is missing or only feature-specific design docs exist.

### Check for Feature-Specific Docs

If design.md exists, check the header:
- If it mentions a specific feature (e.g., "Help System Design"), it's feature-specific
- Rename to `design-<feature>.md`
- Proceed to create project-level design.md

### Create Project-Level Design

Analyze the full codebase for design patterns:

1. **CLI Interface patterns:**
   - Help text structure and styling
   - Command organization
   - Flag conventions

2. **Output patterns:**
   - Default output format
   - Structured output options (--json, --yaml)
   - Table/list formatting

3. **Error patterns:**
   - Error message format
   - Exit codes
   - User guidance/suggestions

4. **Interactive patterns:**
   - Prompts and confirmations
   - Progress indicators

Generate DES-NNN IDs with traces to REQ-NNN based on which requirements each design decision supports.

### Result

```toml
[result]
skill = "design-infer"
status = "success"
mode = "create"

[result.summary]
text = "Created project-level design.md with 12 design decisions."

[outputs.design]
path = "docs/design.md"
action = "created"
items_count = 12

[outputs.renamed]
from = "docs/design.md"
to = "docs/design-help-system.md"
reason = "Feature-specific design moved to allow project-level doc"
```

## Design Categories

### CLI Interface
- Help text layout
- Command structure
- Flag naming conventions

### Output Formatting
- Default output format
- Table styling
- JSON/structured output

### Error Handling
- Error message format
- Exit codes
- User guidance

### Interactive Elements
- Prompts and confirmations
- Progress indicators
- Validation feedback

### Visual Design (if applicable)
- Color scheme
- Typography
- Layout patterns

## Traceability

Link design decisions to requirements:
```markdown
### DES-005: JSON Output Option

Provide `--json` flag for machine-readable output.

**Traces to:** REQ-008 (Support automation)
```

## Mode: Normalize

When `invocation.mode = "normalize"` (from `/project align` Phase 2):

Convert non-standard design.md formats to standard header format while preserving all content.

### Input Detection

Detect non-standard formats:
- Table format: `| DES-001 | ... |`
- Wrong header levels: `## DES-001` instead of `### DES-001`
- Non-standard field names: `Traces:` instead of `**Traces to:**`
- Missing structure: design content without DES-NNN IDs

### Conversion Process

For each design decision:

1. **Convert tables to headers:**

**From:**
```markdown
| DES-001 | Help Layout | Two-column format for flags |
```

**To:**
```markdown
### DES-001: Help Layout

Two-column format for flags.
```

2. **Normalize header levels:**
   - Convert `## DES-001` or `# DES-001` to `### DES-001`
   - Keep section headers (`## CLI Interface`) at `##` level

3. **Normalize field names:**
   - `Traces:` → `**Traces to:**`
   - `Source:` → `**Source:**`
   - Ensure all metadata fields are bold

4. **Add missing IDs:**
   - Prose design content without DES-NNN gets assigned next available ID
   - Use the next number after highest existing DES-NNN

### Traceability (Inline Only)

After normalization, verify all `**Traces to:**` fields are properly formatted. All traceability is inline - no separate traceability.toml. Run `projctl trace validate` to verify.

### Result

```toml
[result]
skill = "design-infer"
status = "success"
mode = "normalize"

[result.summary]
text = "Normalized 8 design decisions. Added 2 IDs to prose content."

[outputs.design]
path = "docs/design.md"
action = "normalized"
format_before = "mixed"
format_after = "header"
items_count = 10
```

## Error Handling

- **No CLI found**: Focus on API/library interface patterns
- **No consistent patterns**: Document as "varied" and escalate
- **UI project**: Use screenshots for visual design inference
