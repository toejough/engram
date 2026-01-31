---
name: pm-infer
description: Infer requirements from codebase analysis
user-invocable: false
---

# PM Infer

Infer requirements from an existing codebase by analyzing documentation, public interfaces, and tests. Used by `/project adopt` for codebase adoption.

## When Invoked

This skill is dispatched by `/project` orchestrator with a context file containing:
- Project directory and config paths
- Mode: "infer", "update", or "normalize"
- Existing artifact summaries

## Modes

- **infer**: Create requirements.md from scratch by analyzing codebase
- **update**: Add new requirements to existing file, preserve existing IDs
- **normalize**: Convert existing file to standard format (table → header)

## Analysis Order

Analyze in this order (broader context first):

1. **README.md** - Project purpose, features, usage examples
2. **Existing requirements.md** - Preserve existing REQ-NNN items
3. **CLI --help** - Commands, flags, options
4. **Public API exports** - Functions, types, interfaces
5. **Test names/descriptions** - Implied requirements from test coverage

## Output Location

All artifacts go in `docs/` at the repo root (not `.project/` or other locations):
- `docs/requirements.md` - Generated requirements document
- `docs/context/pm-infer-result.toml` - Result file for orchestrator

## Workflow

### 1. Read Context

```bash
projctl context read --dir <project-dir> --skill pm-infer
```

Extract:
- `invocation.mode` - "infer" or "update"
- `config.*_path` - Artifact locations
- `inputs.*` - Summaries from previous analysis

### 2. Analyze README

Read the project's README.md and extract:
- **Project purpose**: What problem does it solve?
- **Key features**: Bullet points, feature lists
- **Usage examples**: What can users do with it?
- **Installation/setup**: Prerequisites, requirements

Convert each feature/capability into a potential requirement.

### 3. Preserve Existing Requirements

If `inputs.existing_docs.requirements_exists = true`:
- Read existing requirements.md
- Parse existing REQ-NNN items (header format AND table format)
- Keep these as-is (preserve mode)
- Only add new requirements with higher IDs

**ID extraction patterns:**
- Header format: `### REQ-001: Title` or `## REQ-001: Title`
- Table format: `| REQ-001 |` or `| ID |...| REQ-001 |`
- The format doesn't matter - extract all REQ-NNN patterns

### 3a. Identify Prose Requirements

Look for requirement-like content without IDs:
- Bullet points under headings like "Requirements:", "Must:", "Shall:"
- Numbered lists describing capabilities
- Prose paragraphs that describe what the system must do

**For each prose requirement found:**
1. **If clear and well-defined**: Assign the next available REQ-NNN ID
2. **If ambiguous or overlapping**: Add to escalation list

**Prose requirement indicators:**
- "must", "shall", "should", "will" language
- "The system...", "Users can...", "Supports..."
- Bullet lists under requirements-related headers

**Example transformation:**
```markdown
**Requirements:**
- Start as simply as possible
- Add capabilities incrementally as needed
```

Becomes:
```markdown
### REQ-070: Simple Starting Point
System must support starting with minimal configuration.
**Source:** Inferred from prose requirement
**Confidence:** medium

### REQ-071: Incremental Capability Addition
System must support adding capabilities incrementally.
**Source:** Inferred from prose requirement
**Confidence:** medium
```

**Important:** Don't duplicate - if a prose requirement is already covered by an existing REQ-NNN, skip it.

### 4. Analyze CLI Help

If the project has a CLI:
```bash
<binary> --help
<binary> <command> --help
```

Extract:
- Available commands → functional requirements
- Global flags → configuration requirements
- Per-command flags → feature requirements

### 5. Analyze Public Interfaces

For Go projects, examine:
- Exported functions in root package
- Exported types and interfaces
- Package-level documentation

Each significant export may imply a requirement:
- `func Parse(args []string) → REQ: Support argument parsing`
- `type Config struct {...} → REQ: Support configuration`

### 6. Analyze Test Names

Test function names often describe requirements:
- `TestParsePositionalArgs` → REQ: Support positional arguments
- `TestValidateRequired` → REQ: Validate required fields
- `TestHelpOutput` → REQ: Generate help text

Group related tests to infer broader requirements.

### 7. Generate Requirements

For each identified requirement:

```markdown
## REQ-NNN: <Title>

<Brief description of what the system must do>

**Source:** <Where this was inferred from>
**Confidence:** <high/medium/low>
**Traces to:** ISSUE-NNN (if originating from an issue)
```

### ID Assignment

- Start from REQ-001 (or next available if preserving)
- Group related requirements together
- Number sequentially within groups

### 8. Create Traceability Links (if ISSUE context provided)

If the orchestrator provides ISSUE context (e.g., requirements are being inferred for a specific issue), add `**Traces to:** ISSUE-NNN` to each relevant REQ section.

All traceability is inline in artifacts - no separate traceability.toml. This enables the full chain: ISSUE → REQ → DES → ARCH → TASK → TEST

### 9. Identify Escalations

Create escalations for:
- **Ambiguous scope**: Feature mentioned but unclear boundaries
- **Conflicting signals**: README says one thing, tests suggest another
- **Missing context**: Can't determine if something is a requirement

Format:
```toml
[[escalations]]
id = "ESC-001"
category = "requirement"
context = "Analyzing README features"
question = "Is 'plugin support' a core requirement or future roadmap?"
suggested_answer = "Appears to be mentioned as future work, not current feature"
```

### 9. Write Outputs

1. **Generate requirements.md**:
   - YAML frontmatter with metadata
   - Requirements grouped by category
   - Each requirement has ID, description, source

2. **Write result file**:
   ```bash
   projctl context write --dir <project-dir> --skill pm-infer --result --file <result.toml>
   ```

## Output Format

### requirements.md

```markdown
---
project: <project-name>
generated: <timestamp>
source: pm-infer
---

# Requirements

## Core Functionality

### REQ-001: <Title>

<Description>

**Source:** README.md, "Key Features" section
**Confidence:** high

### REQ-002: <Title>

...

## Configuration

### REQ-005: <Title>

...

## Error Handling

### REQ-008: <Title>

...
```

### Result File

```toml
[result]
skill = "pm-infer"
status = "needs_escalation"  # or "success"
timestamp = 2024-01-15T11:00:00Z

[result.summary]
text = "Inferred 12 requirements from codebase. 2 escalations pending."

[outputs.requirements]
path = "docs/requirements.md"
action = "created"
items_count = 12
id_range = "REQ-001 to REQ-012"

[[escalations]]
id = "ESC-001"
category = "requirement"
context = "README features"
question = "Is plugin support a core requirement?"

[context_for_next]
[context_for_next.artifacts]
requirements_summary = "12 requirements across 3 categories..."

[context_for_next.decisions]
decisions = ["Grouped by functionality", "CLI commands = functional reqs"]
```

## Mode: Update

When `invocation.mode = "update"` (from `/project align`):

1. Read existing requirements.md
2. Parse all existing REQ-NNN items
3. Re-analyze codebase
4. **Only add new** requirements (higher IDs)
5. **Never modify** existing requirements
6. Flag potentially stale requirements in result

```toml
[result.drift]
new_items = ["REQ-015", "REQ-016"]
potentially_stale = ["REQ-003"]  # Code for this seems removed
```

## Mode: Normalize

When `invocation.mode = "normalize"` (from `/project align` Phase 2):

Convert table-format requirements to header format while preserving all content.

### Input Detection

Detect table format by looking for patterns like:
```markdown
| ID | ... |
| -- | --- |
| REQ-001 | ... |
```

Or:
```markdown
| REQ-001 | Capability | Description |
```

### Conversion Process

For each table row with a REQ-NNN:

1. Extract the ID, title/capability, and description columns
2. Convert to header format:

**From:**
```markdown
| REQ-001 | Basic | Executable behavior |
```

**To:**
```markdown
### REQ-001: Basic

Executable behavior.
```

3. Preserve section structure (keep `## Model`, `## Operations`, etc. headers)
4. Preserve any Implementation Status sections unchanged
5. Preserve any non-table prose as context above the requirements

### Prose Requirements

Also convert prose requirements (bullet lists under "Requirements:" headers):

**From:**
```markdown
**Requirements:**
- Start as simply as possible
- Add capabilities incrementally
```

**To:**
```markdown
### REQ-070: Simplicity First

Start as simply as possible.

**Source:** Inferred from prose requirement
**Confidence:** medium

### REQ-071: Incremental Capabilities

Add capabilities incrementally as needed.

**Source:** Inferred from prose requirement
**Confidence:** medium
```

Use the next available REQ-NNN after the highest existing ID.

### Result

```toml
[result]
skill = "pm-infer"
status = "success"
mode = "normalize"

[result.summary]
text = "Normalized 69 requirements from table format. Added 3 IDs to prose requirements."

[outputs.requirements]
path = "docs/requirements.md"
action = "normalized"
format_before = "table"
format_after = "header"
items_count = 72
```

## Confidence Levels

- **High**: Explicitly stated in README, has tests
- **Medium**: Inferred from multiple sources
- **Low**: Single source, may be implementation detail

Low-confidence items should be escalated for user confirmation.

## Common Patterns

### CLI Tool Requirements

```
README "features" → REQ: Core capabilities
--help commands → REQ: Functional requirements
--help flags → REQ: Configuration requirements
Error messages → REQ: Error handling requirements
```

### Library Requirements

```
README "API" → REQ: Public interface requirements
Exported types → REQ: Data structure requirements
Examples → REQ: Usage pattern requirements
```

## Error Handling

- **README not found**: Log warning, continue with other sources
- **Parse error**: Log error, skip that source, continue
- **No requirements found**: Report as failure, suggest manual creation
