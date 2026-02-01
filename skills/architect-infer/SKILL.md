---
name: architect-infer
description: Infer architecture decisions from code structure
user-invocable: false
---

# Architect Infer

Infer architecture decisions from an existing codebase by analyzing code structure, dependencies, and patterns. Used by `/project adopt` for codebase adoption.

## When Invoked

This skill is dispatched by `/project` orchestrator with a context file containing:
- Project directory and config paths
- Mode: "infer", "update", or "augment"
- Requirements and design summaries
- Existing artifact summaries

## Modes

- **infer**: Create architecture.md from scratch by analyzing code structure
- **update**: Add new architecture decisions to existing file, preserve existing IDs
- **augment**: Add ARCH-NNN IDs and traces_to fields to existing architecture content
- **normalize**: Convert existing file to standard format (table → header, fix header levels)

## Analysis Focus

Architecture focuses on **implementation structure**:
- How code is organized
- What dependencies are used
- What patterns are employed
- How components interact

## Analysis Order

1. **Existing architecture.md** - Preserve existing ARCH-NNN items
2. **go.mod/package.json** - Dependencies and module structure
3. **Directory structure** - Package/module organization
4. **Import graph** - Component dependencies
5. **Build tooling** - Development decisions (Makefile, mage, etc.)
6. **Configuration patterns** - How config is loaded/managed

## Workflow

### 1. Read Context

```bash
projctl context read --dir <project-dir> --skill architect-infer
```

Extract:
- `invocation.mode` - "infer" or "update"
- `inputs.requirements_summary` - From pm-infer
- `inputs.design_summary` - From design-infer

### 1a. Parse Upstream Artifacts for Tracing

Read requirements.md and design.md to extract all IDs with descriptions:

**Requirements (REQ-NNN) - both formats:**
- Header: `### REQ-001: Title`
- Table: `| REQ-001 | Capability | Description |`

**Design (DES-NNN):**
- Header: `### DES-001: Title`
- Look for `**Traces to:**` to understand requirement coverage

Build maps of ID → description for use when creating traces_to fields in architecture.

### 2. Preserve Existing Architecture

If existing architecture.md:
- Parse existing ARCH-NNN items
- Keep these as-is
- Only add new decisions with higher IDs

### 3. Analyze Dependencies

For Go projects, read `go.mod`:
- Direct dependencies → ARCH decisions
- Major frameworks → architectural choices

```
ARCH-001: Uses BurntSushi/toml for configuration
ARCH-002: Uses onsi/gomega for test assertions
ARCH-003: Uses pgregory.net/rapid for property testing
```

### 4. Analyze Directory Structure

Map package organization:
```
cmd/           → CLI entry points
internal/      → Non-exported implementation
  config/      → Configuration management
  parser/      → Parsing logic
  trace/       → Traceability features
```

```
ARCH-004: Entry points in cmd/, implementation in internal/
ARCH-005: Single responsibility packages (config, parser, trace)
```

### 5. Analyze Import Graph

Look at internal package dependencies:
- Which packages import which
- Layered architecture patterns
- Circular dependency avoidance

```
ARCH-006: Parser depends on trace for TraceItem type
ARCH-007: Config has no internal dependencies (leaf package)
```

### 6. Analyze Build Tooling

Examine Makefile, magefile, or scripts:
- Build commands
- Test commands
- Lint configuration
- CI integration

```
ARCH-008: Uses mage for build automation
ARCH-009: golangci-lint for code quality
ARCH-010: No vendoring, relies on module cache
```

### 7. Analyze Configuration Patterns

How is configuration managed:
- Environment variables
- Config files
- Command-line flags
- Defaults

```
ARCH-011: Config files use TOML format
ARCH-012: Layered config: defaults → global → project
ARCH-013: CLI flags override config file values
```

### 8. Analyze Code Patterns

Common patterns observed:
- Dependency injection
- Interface-based design
- Error handling patterns
- Testing patterns

```
ARCH-014: File system operations via interface for testability
ARCH-015: Errors wrapped with context using fmt.Errorf
ARCH-016: Table-driven tests with gomega matchers
```

### 9. Generate Architecture Document

```markdown
---
project: <project-name>
generated: <timestamp>
source: architect-infer
---

# Architecture

## Architecture Overview

<High-level description of system structure>

## Dependencies

### ARCH-001: TOML Configuration Library

Uses `github.com/BurntSushi/toml` for parsing configuration files.

**Rationale:** Standard, well-maintained, good error messages
**Source:** go.mod analysis

## Package Structure

### ARCH-004: Entry Point Convention

CLI entry points in `cmd/`, implementation in `internal/`.

**Rationale:** Standard Go project layout
**Source:** Directory structure analysis

## Build & Development

### ARCH-008: Mage Build System

Uses mage for build automation instead of Makefile.

**Rationale:** Go-native, type-safe build scripts
**Source:** magefile.go presence

## Code Patterns

### ARCH-014: Dependency Injection via Interfaces

File system operations abstracted via interfaces for testability.

**Rationale:** Enables unit testing without real file I/O
**Source:** FileSystem interface in parser package
**Traces to:** DES-003, REQ-010
```

### 10. Traceability (Inline Only)

Traceability is embedded directly in the architecture.md file via `**Traces to:**` fields. There is NO separate traceability.toml file.

When generating architecture.md, ensure each ARCH-NNN section includes:

```markdown
### ARCH-014: Dependency Injection

File system operations via interfaces.

**Traces to:** DES-003, REQ-010
```

The `**Traces to:**` field links this decision to its upstream requirements/design decisions.

**Important:** Links exist only in the artifact files. Run `projctl trace validate` to verify all links are valid.

### 11. Identify Escalations

Escalate for:
- Unusual patterns that may be intentional or accidental
- Dependency choices that seem outdated
- Architecture decisions that conflict with design

```toml
[[escalations]]
id = "ESC-004"
category = "architecture"
context = "Analyzing dependencies"
question = "Uses deprecated library X. Should this be updated?"
suggested_answer = "Consider migrating to library Y which is actively maintained"
```

### 12. Write Result

```toml
[result]
skill = "architect-infer"
status = "success"
timestamp = 2024-01-15T13:00:00Z

[result.summary]
text = "Inferred 16 architecture decisions. Created 20 traceability links."

[outputs.architecture]
path = "docs/architecture.md"
action = "created"
items_count = 16
id_range = "ARCH-001 to ARCH-016"

[outputs.traceability]
links_created = 20

[context_for_next]
[context_for_next.artifacts]
architecture_summary = "16 decisions covering deps, structure, patterns..."
```

## Mode: Update

When `invocation.mode = "update"`:
- Only add new architecture decisions
- Flag potentially stale decisions
- Never modify existing ARCH-NNN

## Mode: Augment

When `invocation.mode = "augment"` (from `/project align` backfill):

Used when architecture.md exists with good content but lacks ARCH-NNN IDs.

### Add ARCH-NNN IDs

For each major section that represents an architectural decision:

1. **Identify decision sections:**
   - `## Arguments`, `## Execution`, `## Hierarchy`, etc.
   - Any section describing "how" something is implemented

2. **Add ARCH-NNN header after section title:**

**From:**
```markdown
## Arguments

What CLI inputs the target accepts...
```

**To:**
```markdown
## Arguments

### ARCH-001: Struct-Based Argument Definition

**Traces to:** REQ-008, REQ-009, REQ-010, REQ-011

What CLI inputs the target accepts...
```

3. **Determine traces_to by matching to requirements:**
   - Read requirements.md to understand what each section implements
   - Add `**Traces to:** REQ-NNN, REQ-NNN` listing all requirements the decision supports

### Preserve Content

- Never modify the actual architecture content
- Only add ARCH-NNN headers and traces_to fields
- Keep existing section structure intact

### Traceability (Inline Only)

After adding ARCH-NNN IDs, ensure each section has a `**Traces to:**` field linking to upstream REQ/DES items. All traceability is inline in the artifact - no separate traceability.toml.

### Result

```toml
[result]
skill = "architect-infer"
status = "success"
mode = "augment"

[result.summary]
text = "Added 11 ARCH-NNN IDs to existing architecture. Added **Traces to:** fields."

[outputs.architecture]
path = "docs/architecture.md"
action = "augmented"
ids_added = 11
id_range = "ARCH-001 to ARCH-011"

[outputs.traceability]
links_created = 45
```

## Architecture Categories

### Dependencies
- External libraries
- Frameworks
- Tools

### Package Structure
- Module organization
- Layer boundaries
- Component isolation

### Build & Development
- Build tooling
- Test infrastructure
- CI/CD patterns

### Code Patterns
- Error handling
- Dependency injection
- Testing patterns

### Data Flow
- Request handling
- State management
- Persistence patterns

## Traceability

Link architecture to design and requirements:
```markdown
### ARCH-014: Dependency Injection

File system operations via interfaces.

**Traces to:** DES-003 (Testability), REQ-010 (Unit test coverage)
```

## Language-Specific Analysis

### Go Projects
- go.mod for dependencies
- internal/ convention
- interface-based DI

### Node.js Projects
- package.json for dependencies
- src/ structure
- middleware patterns

### Python Projects
- requirements.txt/pyproject.toml
- package structure
- decorators and mixins

## Mode: Normalize

When `invocation.mode = "normalize"` (from `/project align` Phase 2):

Convert non-standard architecture.md formats to standard header format while preserving all content.

### Input Detection

Detect non-standard formats:
- Table format: `| ARCH-001 | ... |`
- Wrong header levels: `## ARCH-001` instead of `### ARCH-001`
- Non-standard field names: `Traces:` instead of `**Traces to:**`
- Content without ARCH-NNN IDs (use augment logic to add)

### Conversion Process

For each architecture decision:

1. **Convert tables to headers:**

**From:**
```markdown
| ARCH-001 | TOML Config | Uses BurntSushi/toml |
```

**To:**
```markdown
### ARCH-001: TOML Config

Uses BurntSushi/toml for configuration parsing.
```

2. **Normalize header levels:**
   - Convert `## ARCH-001` or `# ARCH-001` to `### ARCH-001`
   - Keep section headers (`## Dependencies`) at `##` level

3. **Normalize field names:**
   - `Traces:` → `**Traces to:**`
   - `Rationale:` → `**Rationale:**`
   - `Source:` → `**Source:**`
   - Ensure all metadata fields are bold

4. **Add missing IDs (augment behavior):**
   - Sections describing architectural decisions without ARCH-NNN get assigned next available ID
   - Use the next number after highest existing ARCH-NNN

### Traceability (Inline Only)

After normalization, verify all `**Traces to:**` fields are properly formatted and reference valid upstream IDs. All traceability is inline - no separate traceability.toml.

Run `projctl trace validate` to verify links.

### Result

```toml
[result]
skill = "architect-infer"
status = "success"
mode = "normalize"

[result.summary]
text = "Normalized 15 architecture decisions. Added 3 IDs to prose content."

[outputs.architecture]
path = "docs/architecture.md"
action = "normalized"
format_before = "mixed"
format_after = "header"
items_count = 18
```

## Error Handling

- **No build files**: Document manual build process
- **Mixed patterns**: Document as "varied" and escalate
- **Unclear rationale**: Mark as "inferred" with lower confidence

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = ["docs/architecture.md"]

[[decisions]]
context = "Module structure"
choice = "Use internal/ for implementation"
reason = "Go convention for non-public code"
alternatives = ["Flat package structure"]

[[learnings]]
content = "Codebase uses dependency injection pattern"
```
