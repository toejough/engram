---
name: arch-infer-producer
description: Infer architecture decisions from existing code structure
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: arch
variant: infer
---

# Architecture Infer Producer

Analyze existing code to infer architecture decisions and produce architecture.md.

**Protocol:** [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md) | [YIELD.md](../shared/YIELD.md)

## Purpose

This skill examines code structure, dependencies, and patterns to reverse-engineer architecture decisions. Used for:
- Adopting existing codebases into the workflow
- Documenting undocumented architecture
- Creating ARCH-N IDs from implementation evidence

## Workflow

### 1. GATHER

Collect information about code structure:

1. Read context from `[inputs]` section
2. Check for `[query_results]` (resuming after need-context)
3. If code analysis needed, yield `need-context` with queries:
   - `territory` queries for directory structure
   - `file` queries for go.mod, package.json, Makefile, etc.
   - `semantic` queries for pattern understanding
4. Proceed to SYNTHESIZE when sufficient information gathered

**Typical need-context queries:**

```toml
[[payload.queries]]
type = "territory"
scope = "all"

[[payload.queries]]
type = "file"
path = "go.mod"

[[payload.queries]]
type = "file"
path = "package.json"

[[payload.queries]]
type = "semantic"
question = "How is dependency injection handled in this codebase?"
```

### 2. SYNTHESIZE

Analyze gathered code structure:

1. Identify package/module organization patterns
2. Extract technology choices and dependencies
3. Infer component boundaries from imports
4. Detect configuration and build patterns
5. Map to upstream requirements (REQ-N) and design (DES-N) where traceable

### 3. PRODUCE

Create architecture.md artifact:

1. Generate ARCH-N IDs for each decision
2. Include `**Traces to:**` links to REQ/DES IDs
3. Write to configured output path
4. Yield `complete` with artifact details

## Input (Context TOML)

```toml
[inputs]
project_dir = "/path/to/project"
requirements = "docs/requirements.md"  # Optional, for tracing
design = "docs/design.md"              # Optional, for tracing

[config]
output_path = "docs/architecture.md"
mode = "infer"  # infer | update | augment | normalize
```

## Output (architecture.md)

```markdown
# Architecture

## ARCH-001: Module Organization

Project uses Go module structure with internal/ for private packages.

**Traces to:** REQ-001

## ARCH-002: Build System

Uses Mage for build automation with targets defined in dev/.

**Traces to:** DES-003
```

## Yield Types

| Yield | When | Payload |
|-------|------|---------|
| `need-context` | Need code structure analysis | `queries[]` with territory/file/semantic |
| `need-decision` | Ambiguous architecture choice | Options with recommendation |
| `complete` | architecture.md created | `artifact`, `ids_created[]` |
| `blocked` | Cannot analyze (e.g., binary-only) | Blocker details |
| `error` | Parse/access failure | Error details, recoverable flag |

## ARCH ID Format

```markdown
### ARCH-NNN: Decision Title

Description of the architecture decision with rationale.

**Traces to:** REQ-XXX, DES-YYY
```

## Analysis Sources

| Source | What to Extract |
|--------|-----------------|
| go.mod / package.json | Dependencies, module structure |
| Directory structure | Package organization, boundaries |
| Import graph | Component dependencies |
| Build tooling | Makefile, mage, scripts |
| Config patterns | How configuration is loaded |
| Test structure | Testing strategy and boundaries |

## Modes

| Mode | Behavior |
|------|----------|
| `infer` | Create architecture.md from scratch |
| `update` | Add new decisions, preserve existing ARCH IDs |
| `augment` | Add ARCH-N IDs to existing content |
| `normalize` | Fix format, header levels, traceability |

## Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "docs/architecture.md"
ids_created = ["ARCH-001", "ARCH-002", "ARCH-003"]
files_modified = ["docs/architecture.md"]

[[payload.decisions]]
context = "Module organization"
choice = "internal/ for private packages"
reason = "Go standard layout for encapsulation"

[context]
phase = "arch"
subphase = "complete"
```

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/architecture.md"
      id_format: "ARCH-N"

  traces_to:
    - "docs/design.md"
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every architecture decision has ARCH-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every ARCH-N traces to at least one DES-N or REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "All technical implications from requirements addressed (completeness)"
      severity: error

    - id: "CHECK-004"
      description: "All technology decisions from design covered"
      severity: error

    - id: "CHECK-005"
      description: "No conflicts with requirements or design"
      severity: error

    - id: "CHECK-006"
      description: "No orphan references (mentions IDs that don't exist)"
      severity: error

    - id: "CHECK-007"
      description: "Technical decisions include rationale"
      severity: warning

    - id: "CHECK-008"
      description: "Inferred architecture traces to observable code patterns"
      severity: warning
```
