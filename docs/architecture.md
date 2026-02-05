# projctl Architecture

Architecture decisions derived from [review-2025-01.md](./review-2025-01.md). Each ARCH item represents a phase of the implementation plan.

---

### ARCH-001: Structured Result Format

**Phase:** 0
**Priority:** High
**Timeline:** This Week

Foundation for skill communication. All skills return structured result.toml with status, outputs, decisions, and learnings.

**Traces to:** REQ-001

---

### ARCH-002: CLI Completeness

**Phase:** 1
**Priority:** High
**Timeline:** This Week

Complete CLI commands referenced by skills: context, escalation, conflict, integrate.

**Traces to:** REQ-001

---

### ARCH-003: Model Routing

**Phase:** 2
**Priority:** Medium
**Timeline:** This Month

Automatic model selection based on task complexity. Haiku for simple, Sonnet for medium, Opus for complex.

**Traces to:** REQ-001

---

### ARCH-004: Cost Visibility

**Phase:** 3
**Priority:** Medium
**Timeline:** Next Month

Token usage tracking and budget alerts for cost optimization.

**Traces to:** REQ-001, TASK-7

---

### ARCH-005: Learning Loop

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Correction tracking and pattern detection for automatic skill improvement proposals.

**Traces to:** REQ-001, TASK-8

---

### ARCH-006: Parallel Skill Dispatch

**Phase:** 5
**Priority:** Medium
**Timeline:** Next Month

Concurrent execution of independent skills for efficiency.

**Traces to:** REQ-001, TASK-9

---

### ARCH-007: Background Territory Mapping

**Phase:** 6
**Priority:** Medium
**Timeline:** Next Month

Pre-exploration of codebase structure to reduce repeated discovery.

**Traces to:** REQ-001, TASK-8

---

### ARCH-008: Graceful Degradation

**Phase:** 7
**Priority:** High
**Timeline:** This Month

Error recovery and continuation with unblocked tasks when failures occur.

**Traces to:** REQ-001

---

### ARCH-009: LSP Integration

**Phase:** 8
**Priority:** Lower
**Timeline:** Next Quarter

LSP-backed refactoring for deterministic symbol operations.

**Traces to:** REQ-001, TASK-10

---

### ARCH-010: CLAUDE.md Migration

**Phase:** 9
**Priority:** High
**Timeline:** This Month

Critical rules from skills moved to CLAUDE.md for passive context availability.

**Traces to:** REQ-001, TASK-9

---

### ARCH-011: Skill Compression

**Phase:** 10
**Priority:** Medium
**Timeline:** Next Month

Compress skills to < 500 tokens with full docs retrievable on demand.

**Traces to:** REQ-001

---

### ARCH-012: Deterministic Workflow Enforcement

**Phase:** 11
**Priority:** High
**Timeline:** This Week

State machine preconditions prevent skipping workflow steps.

**Traces to:** REQ-001

---

### ARCH-013: Relentless Continuation

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Orchestrator continues autonomously until all tasks complete or blocked.

**Traces to:** REQ-001

---

### ARCH-014: Cross-Project Memory System

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Persistent memory across projects and sessions for learnings and decisions.

**Traces to:** REQ-001, TASK-8

---

### ARCH-015: Visual Acceptance Criteria

**Phase:** 14
**Priority:** Medium
**Timeline:** This Month

UI verification through screenshots and visual regression detection.

**Traces to:** REQ-001, TASK-10

---

### ARCH-016: Skill Version Control

**Phase:** 15
**Priority:** High
**Timeline:** This Week

Skills versioned in projctl repo with install/status/uninstall commands.

**Traces to:** REQ-001, TASK-7

---

### ARCH-017: Code Cleanup

**Phase:** Housekeeping
**Priority:** Lower
**Timeline:** Next Quarter

Remove stub code and consolidate implementations.

**Traces to:** REQ-001

---

### ARCH-018: Orchestrator-Skill Contract

**Phase:** 0
**Priority:** High
**Timeline:** This Week

Defines the bidirectional communication protocol between the `/project` orchestrator and skills. All skills must read context TOML and write yield TOML using this contract.

#### Context Input Format

The orchestrator provides context to skills via TOML files at `.claude/context/<skill-name>-context.toml`:

```toml
[invocation]
skill = "pm-infer"           # Skill being invoked
mode = "infer"               # "interview", "infer", or "update"
task = "PHASE"               # Current task ID or "PHASE" for phase-level work
timestamp = 2024-01-15T10:30:00Z

[project]
name = "my-project"
dir = "/path/to/project"
phase = "adopt-infer-pm"     # Current state machine phase

[config]
# Resolved artifact paths from project-config.toml
docs_dir = "docs"
requirements_path = "docs/requirements.md"
design_path = "docs/design.md"
architecture_path = "docs/architecture.md"

[inputs]
# Curated summaries of relevant information

[inputs.readme]
exists = true
summary = "CLI tool for managing project documentation..."

[inputs.previous_phase]
skill = "coverage-analyze"
summary = "Coverage: 45%, recommendation: migrate"

[state]
tasks_complete = 0
tasks_total = 0
escalations_pending = 0

[output]
yield_path = ".claude/context/pm-infer-yield.toml"

[query_results]
# Injected when resuming after need-context yield
```

**Modes:**
- `interview` - Interactive Q&A with user
- `infer` - Analyze artifacts, generate content
- `update` - Lightweight mode for `/project align`

#### Yield Output Format

Skills write yield TOML to the path specified in `output.yield_path`:

```toml
[yield]
type = "<yield-type>"
timestamp = 2026-02-02T10:30:00Z

[payload]
# Type-specific fields

[context]
# State for resumption
phase = "pm"
iteration = 1
```

**Producer Yield Types:**

| Type | Meaning | Orchestrator Action |
|------|---------|---------------------|
| `complete` | Work finished | Advance to QA or next phase |
| `need-user-input` | Question for user | Prompt user, resume with answer |
| `need-context` | Need information | Run queries, resume with results |
| `need-decision` | Ambiguous choice | Present options, resume with choice |
| `need-agent` | Need another agent | Spawn agent, resume with result |
| `blocked` | Cannot proceed | Present blocker, await resolution |
| `error` | Something failed | Retry (max 3x) or escalate |

**QA Yield Types:**

| Type | Meaning | Orchestrator Action |
|------|---------|---------------------|
| `approved` | Work passes QA | Advance to next phase |
| `improvement-request` | Needs fixes | Resume producer with feedback |
| `escalate-phase` | Prior phase issue | Return to prior phase with proposed changes |
| `escalate-user` | Cannot resolve | Present to user |

#### Resumption Protocol

Each yield type triggers a specific orchestrator response:

1. **complete** - Orchestrator advances to QA skill or next phase. No skill resumption.

2. **need-user-input** - Orchestrator prompts user, captures response, writes to `[query_results]` section, re-invokes skill.

3. **need-context** - Orchestrator executes queries in parallel:
   - `file` queries: Read file contents
   - `memory` queries: ONNX semantic memory search
   - `territory` queries: Codebase structure map
   - `web` queries: URL fetch with prompt interpretation
   - `semantic` queries: LLM exploration via context-explorer agent

   Results injected into `[query_results.items]`, skill re-invoked.

4. **need-decision** - Orchestrator presents options to user, captures choice, writes to `[query_results]`, re-invokes skill.

5. **need-agent** - Orchestrator spawns specified agent with input, captures result, writes to `[query_results]`, re-invokes skill.

6. **blocked** - Orchestrator presents blocker to user, awaits resolution signal, re-invokes skill.

7. **error** - Orchestrator retries up to 3 times. If recoverable=false or retries exhausted, escalates to user.

8. **approved** (QA) - Orchestrator advances to next phase. No resumption.

9. **improvement-request** (QA) - Orchestrator re-invokes producer with feedback in `[inputs.qa_feedback]`.

10. **escalate-phase** (QA) - Orchestrator returns to prior phase producer with proposed changes in `[inputs.escalation]`.

11. **escalate-user** (QA) - Orchestrator presents to user, awaits resolution, resumes with decision.

#### Query Result Injection

When resuming after `need-context`, the orchestrator injects results:

```toml
[query_results]
[[query_results.items]]
query_type = "file"
query_path = "docs/requirements.md"
result = "... file contents ..."

[[query_results.items]]
query_type = "semantic"
query_question = "How does authentication work?"
result = "Authentication uses JWT tokens..."
```

Skills check for `[query_results]` presence to detect resumption vs fresh invocation.

**Traces to:** REQ-001, ARCH-001, ARCH-013

---

## ISSUE-053: Universal QA Skill Architecture

Technical architecture for replacing 13 phase-specific QA skills with one universal QA skill.

---

### ARCH-019: Universal QA Skill Location

The universal QA skill lives at `skills/qa/SKILL.md`.

**Structure:**

```
skills/
  qa/
    SKILL.md           # Main skill definition with workflow
    SKILL-full.md      # Extended documentation (optional)
```

**Rationale:**
- Follows existing skill naming convention (`skills/<name>/SKILL.md`)
- Short name (`qa`) is sufficient - no phase prefix needed since it's universal
- Single skill replaces 13 phase-specific QA skills listed in REQ-009

**Frontmatter:**

```yaml
---
name: qa
description: Universal QA skill that validates any producer against its SKILL.md contract
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
role: qa
---
```

**Key attributes:**
- `model: haiku` - Fast and cheap enough for validation (per REQ-005)
- `role: qa` - Consistent with existing QA skill role
- No `phase` attribute - applies to all phases

**Traces to:** REQ-005, DES-013, ARCH-018, ARCH-011, TASK-2

---

### ARCH-020: Contract Standard Location

The contract format standard is documented at `skills/shared/CONTRACT.md`.

**Purpose:** Defines the YAML format that producers use in their Contract sections.

**Location rationale:**
- `skills/shared/` contains other cross-skill standards (YIELD.md, PRODUCER-TEMPLATE.md, QA-TEMPLATE.md)
- Single source of truth for contract format
- Producers reference it; QA implements it

**Contents outline:**
1. Contract YAML schema definition
2. Field descriptions and examples
3. Severity levels (error vs warning)
4. Evolution and versioning policy
5. Examples for each producer type

**Traces to:** REQ-006, DES-001, DES-002, ARCH-002, TASK-1

---

### ARCH-021: Contract Extraction from Producer SKILL.md

QA extracts contracts using markdown parsing, not complex YAML processing.

**Algorithm:**

1. Read producer SKILL.md as text
2. Search for `## Contract` heading (case-insensitive)
3. Find the next fenced code block with language `yaml`
4. Extract content between backticks
5. Parse as YAML

**Pseudocode:**

```
function extractContract(skillMdContent):
    lines = skillMdContent.split("\n")
    inContractSection = false
    inCodeBlock = false
    yamlLines = []

    for line in lines:
        if line.match(/^## Contract/i):
            inContractSection = true
            continue

        if inContractSection and line.startsWith("```yaml"):
            inCodeBlock = true
            continue

        if inCodeBlock and line.startsWith("```"):
            break  # End of YAML block

        if inCodeBlock:
            yamlLines.append(line)

    return parseYAML(yamlLines.join("\n"))
```

**Error cases:**
- No `## Contract` heading: Fall back to prose extraction (ARCH-024)
- No YAML code block: Fall back to prose extraction (ARCH-024)
- Invalid YAML: Yield `error` with parse details

**Why not a YAML parser for the whole file?**
- SKILL.md is markdown, not YAML
- Simple text search is sufficient and more robust
- Avoids dependency on markdown-to-AST libraries

**Traces to:** REQ-006, DES-002, DES-008, TASK-2

---

### ARCH-022: Orchestrator Dispatch to Universal QA

The orchestrator dispatches QA using a single invocation pattern regardless of phase.

**Current dispatch table (to be replaced):**

```markdown
| Phase         | QA             |
| ------------- | -------------- |
| PM            | `pm-qa`        |
| Design        | `design-qa`    |
| Architecture  | `arch-qa`      |
| ...           | ...            |
```

**New dispatch pattern:**

```markdown
| Phase         | QA   |
| ------------- | ---- |
| PM            | `qa` |
| Design        | `qa` |
| Architecture  | `qa` |
| Breakdown     | `qa` |
| TDD           | `qa` |
| Documentation | `qa` |
| Alignment     | `qa` |
| Retro         | `qa` |
| Summary       | `qa` |
```

**Context file format (written by orchestrator):**

```toml
[inputs]
producer_skill_path = "skills/design-interview-producer/SKILL.md"
producer_yield_path = ".projctl/yields/design-producer-yield.toml"
artifact_paths = ["docs/design.md"]

[inputs.producer]
name = "design-interview-producer"
phase = "design"

[output]
yield_path = ".projctl/yields/qa-yield.toml"
```

**Orchestrator workflow update:**

```
# In PAIR LOOP pattern (skills/project/SKILL.md)

1. Write context with producer info
2. Dispatch PRODUCER skill
3. Read yield
4. If yield.type = "complete":
   a. Write QA context with:
      - producer_skill_path = skills/{producer-name}/SKILL.md
      - producer_yield_path = from producer's yield
      - artifact_paths = from producer's yield payload
   b. Dispatch `qa` skill  # Changed from phase-specific
5. Handle QA yield (unchanged)
```

**Files to modify:**
- `skills/project/SKILL.md` - Update dispatch table
- `skills/project/SKILL-full.md` - Update phase details

**Traces to:** REQ-010, DES-004, DES-013, ARCH-012, ARCH-018, TASK-5

---

### ARCH-023: QA Workflow

Universal QA follows a three-phase workflow: LOAD -> VALIDATE -> RETURN.

**Phase 1: LOAD**

1. Read context file for inputs
2. Read producer SKILL.md from `producer_skill_path`
3. Extract contract (ARCH-021) or fall back to prose (ARCH-024)
4. Read producer yield from `producer_yield_path`
5. Read artifacts from `artifact_paths`

**Error handling in LOAD:**
- Producer SKILL.md not found: Yield `error` (cannot proceed)
- Producer yield invalid TOML: Yield `improvement-request` with parse details
- Artifacts missing: Yield `improvement-request` with file list

**Phase 2: VALIDATE**

1. For each check in contract:
   - Execute validation against artifacts
   - Record pass/fail with details
2. Aggregate results
3. Determine overall status

**Check execution:**
- ID format checks: Regex match on artifact content
- Trace checks: Parse `**Traces to:**` lines, verify referenced IDs exist
- Custom checks: Interpret description semantically (Haiku is capable enough)

**Phase 3: RETURN**

Based on validation results, yield one of:

| Condition | Yield |
|-----------|-------|
| All checks pass | `approved` |
| Producer-fixable issues | `improvement-request` |
| Upstream phase problem | `escalate-phase` |
| Cannot resolve | `escalate-user` |
| Max iterations reached | `escalate-user` |

**Traces to:** REQ-005, DES-003, DES-005, TASK-2

---

### ARCH-024: Prose Fallback for Missing Contracts

When a producer SKILL.md has no formal Contract section, QA extracts validation criteria from prose.

**Trigger conditions:**
- No `## Contract` heading found
- `## Contract` exists but no YAML code block inside

**Extraction heuristics:**

1. **Checklist patterns:** Lines starting with `- [ ]` or `- [x]`
   - Each checkbox item becomes an implicit check
   - Example: `- [ ] All entries have DES-N IDs` -> check for DES-N format

2. **Table patterns:** Tables with "Check" or "Criterion" columns
   - Each row becomes an implicit check
   - Example: `| Format | All entries use DES-NNN format |` -> format check

3. **"Must" statements:** Sentences containing "must", "shall", "required"
   - Example: "Every DES-N entry must trace to at least one REQ-N"

4. **ID format patterns:** References to ID formats like `REQ-N`, `DES-N`, `ARCH-N`
   - Derive expected ID format from mentions

**Fallback check structure:**

```yaml
# Implicit contract from prose extraction
contract:
  outputs:
    - path: "<inferred from context>"
      id_format: "<inferred from prose>"
  traces_to:
    - "<inferred from prose>"
  checks:
    - id: "IMPLICIT-001"
      description: "<extracted from prose>"
      severity: warning  # Implicit checks are warnings, not errors
```

**Warning in output:**

```
Warning: No contract section found in producer SKILL.md
Using prose extraction for validation criteria.
Recommend adding formal ## Contract section per skills/shared/CONTRACT.md
```

**Traces to:** REQ-011, DES-008, TASK-2

---

### ARCH-025: Gap Analysis Workflow

Before deleting any QA skill, perform gap analysis comparing its checks to the corresponding producer's contract.

**Workflow:**

```
For each QA skill in [pm-qa, design-qa, arch-qa, ...]:
  1. Read QA skill's SKILL.md
  2. Extract its validation checklist (from tables, checklists, prose)
  3. Read corresponding producer(s) SKILL.md
  4. Extract producer's contract
  5. Compare: QA checks vs producer contract checks
  6. Report gaps
```

**Gap detection:**

| Situation | Classification |
|-----------|----------------|
| QA check has equivalent in producer contract | Covered |
| QA check has no equivalent in producer contract | Gap |
| Producer contract check has no equivalent in QA | New (expected) |

**Gap report format:**

```markdown
## Gap Analysis: design-qa vs design-interview-producer

### Covered Checks
- [x] ID format (DES-NNN)
- [x] Traces to REQ-N
- [x] Coverage of requirements

### Gaps (QA checks not in producer contract)
- [ ] "Content describes visual/interaction, not implementation"
- [ ] "No conflicting design decisions"

### Decision Required
For each gap, choose:
1. Add to producer contract (preserves validation)
2. Drop (explicitly decide not to validate)
```

**QA skills to analyze (13 total):**
1. pm-qa
2. design-qa
3. arch-qa
4. breakdown-qa
5. tdd-qa
6. tdd-red-qa
7. tdd-green-qa
8. tdd-refactor-qa
9. doc-qa
10. context-qa
11. alignment-qa
12. retro-qa
13. summary-qa

**Corresponding producers:**

| QA Skill | Producer(s) |
|----------|-------------|
| pm-qa | pm-interview-producer, pm-infer-producer |
| design-qa | design-interview-producer, design-infer-producer |
| arch-qa | arch-interview-producer, arch-infer-producer |
| breakdown-qa | breakdown-producer |
| tdd-qa | tdd-producer |
| tdd-red-qa | tdd-red-producer, tdd-red-infer-producer |
| tdd-green-qa | tdd-green-producer |
| tdd-refactor-qa | tdd-refactor-producer |
| doc-qa | doc-producer |
| context-qa | context-explorer |
| alignment-qa | alignment-producer |
| retro-qa | retro-producer |
| summary-qa | summary-producer |

**Traces to:** REQ-008, ARCH-017, TASK-3

---

### ARCH-026: QA Context File Schema

Formal schema for the context file passed to universal QA.

**Schema:**

```toml
# Required fields
[inputs]
# Path to producer's SKILL.md for contract extraction
producer_skill_path = "skills/<producer-name>/SKILL.md"

# Path to producer's yield file
producer_yield_path = ".projctl/yields/<producer>-yield.toml"

# Paths to artifacts to validate
artifact_paths = ["docs/<artifact>.md"]

# Optional fields
[inputs.producer]
# Producer name (for logging/routing)
name = "<producer-name>"

# Phase this producer belongs to
phase = "<pm|design|arch|breakdown|tdd|doc|alignment|retro|summary>"

# Variant if applicable
variant = "<interview|infer>"  # optional

[inputs.iteration]
# Current iteration in producer-QA loop
current = 1
max = 3

# Previous QA feedback (if this is a retry)
[inputs.previous_feedback]
issues = ["issue 1", "issue 2"]

# Output configuration
[output]
yield_path = ".projctl/yields/qa-yield.toml"
```

**Validation rules:**
- `producer_skill_path`: Must exist and be readable
- `producer_yield_path`: Must exist and be valid TOML
- `artifact_paths`: At least one path required; all must exist
- `inputs.iteration.current`: Must not exceed `max`

**Traces to:** REQ-010, DES-004, TASK-5

---

### ARCH-027: QA Yield Schema

Formal schema for yields produced by universal QA.

**approved:**

```toml
[yield]
type = "approved"
timestamp = <datetime>

[payload]
reviewed_artifact = "<primary artifact path>"
contract_source = "<producer SKILL.md path>"

[[payload.checklist]]
id = "CHECK-001"
description = "<check description>"
passed = true

[[payload.checklist]]
id = "CHECK-002"
description = "<check description>"
passed = true
note = "<optional note>"

[context]
phase = "<phase>"
role = "qa"
iteration = <n>
```

**improvement-request:**

```toml
[yield]
type = "improvement-request"
timestamp = <datetime>

[payload]
from_agent = "qa"
to_agent = "<producer-name>"
iteration = <n>

# Specific issues to fix
issues = [
    "CHECK-001: <specific problem>",
    "CHECK-002: <specific problem>"
]

# Full checklist for reference
[[payload.checklist]]
id = "CHECK-001"
description = "<check description>"
passed = false
details = "<what's wrong>"

[context]
phase = "<phase>"
role = "qa"
iteration = <n>
max_iterations = 3
```

**escalate-phase:**

```toml
[yield]
type = "escalate-phase"
timestamp = <datetime>

[payload.escalation]
from_phase = "<current phase>"
to_phase = "<upstream phase>"
reason = "<error|gap|conflict>"

[payload.issue]
summary = "<brief description>"
context = "<detailed context>"

[[payload.proposed_changes.<artifact>]]
action = "<add|modify|remove>"
id = "<ID>"
title = "<title>"
content = "<content>"

[context]
phase = "<phase>"
role = "qa"
escalating = true
```

**escalate-user:**

```toml
[yield]
type = "escalate-user"
timestamp = <datetime>

[payload]
reason = "<why escalating>"
context = "<detailed context>"
question = "<question for user>"
options = ["option 1", "option 2"]

[context]
phase = "<phase>"
role = "qa"
escalating = true
iteration = <n>
max_iterations_reached = <true|false>
```

**error:**

```toml
[yield]
type = "error"
timestamp = <datetime>

[payload]
error = "<error message>"
details = "<detailed description>"
recoverable = <true|false>

[context]
phase = "<phase>"
role = "qa"
```

**Traces to:** REQ-005, DES-005, DES-006, DES-007, DES-009, DES-010, DES-011, TASK-2

---

### ARCH-028: Iteration Tracking

QA tracks iterations to prevent infinite producer-QA loops.

**Mechanism:**

1. Orchestrator increments iteration count before each QA dispatch
2. QA receives iteration info in context
3. QA includes iteration in yield
4. After max iterations (3), QA yields `escalate-user` instead of `improvement-request`

**Context flow:**

```
Iteration 1:
  Orchestrator -> QA: inputs.iteration.current = 1
  QA -> Orchestrator: context.iteration = 1, type = "improvement-request"

Iteration 2:
  Orchestrator -> QA: inputs.iteration.current = 2
  QA -> Orchestrator: context.iteration = 2, type = "improvement-request"

Iteration 3:
  Orchestrator -> QA: inputs.iteration.current = 3
  QA -> Orchestrator: context.iteration = 3, type = "escalate-user"
  (Regardless of whether issues remain)
```

**Iteration state in yield:**

```toml
[context]
iteration = 3
max_iterations = 3
max_iterations_reached = true
```

**Traces to:** REQ-005, DES-012, ARCH-008, TASK-2

---

### ARCH-029: QA Model Selection

Universal QA uses Haiku model for validation.

**Rationale:**
- Contract extraction is mostly text matching
- Check validation is pattern matching against parsed artifacts
- Haiku is sufficient for these tasks and much cheaper/faster than Sonnet
- If validation requires deep semantic understanding, the contract is underspecified

**Frontmatter:**

```yaml
model: haiku
```

**When to escalate to Sonnet:**
- Never automatically - QA should work with Haiku
- If Haiku consistently fails on certain checks, the checks need rewriting, not a larger model

**Traces to:** REQ-005, ARCH-003, TASK-2

---

### ARCH-030: Files to Modify

Summary of all files requiring modification for this change.

**New files:**
- `skills/qa/SKILL.md` - Universal QA skill
- `skills/shared/CONTRACT.md` - Contract format standard

**Modified files:**
- `skills/project/SKILL.md` - Update dispatch table to use single `qa` skill
- `skills/project/SKILL-full.md` - Update phase details and resume map

**Producer skills to add Contract sections:**
(Listed in REQ-007)
- `skills/pm-interview-producer/SKILL.md`
- `skills/pm-infer-producer/SKILL.md`
- `skills/design-interview-producer/SKILL.md`
- `skills/design-infer-producer/SKILL.md`
- `skills/arch-interview-producer/SKILL.md`
- `skills/arch-infer-producer/SKILL.md`
- `skills/breakdown-producer/SKILL.md`
- `skills/tdd-red-producer/SKILL.md`
- `skills/tdd-red-infer-producer/SKILL.md`
- `skills/tdd-green-producer/SKILL.md`
- `skills/tdd-refactor-producer/SKILL.md`
- `skills/doc-producer/SKILL.md`
- `skills/alignment-producer/SKILL.md`
- `skills/retro-producer/SKILL.md`
- `skills/summary-producer/SKILL.md`

**Files to delete (after gap analysis per REQ-008):**
- `skills/pm-qa/` directory
- `skills/design-qa/` directory
- `skills/arch-qa/` directory
- `skills/breakdown-qa/` directory
- `skills/tdd-qa/` directory
- `skills/tdd-red-qa/` directory
- `skills/tdd-green-qa/` directory
- `skills/tdd-refactor-qa/` directory
- `skills/doc-qa/` directory
- `skills/context-qa/` directory
- `skills/alignment-qa/` directory
- `skills/retro-qa/` directory
- `skills/summary-qa/` directory

**Templates to update or remove:**
- `skills/shared/QA-TEMPLATE.md` - Update to reference universal QA, or mark as deprecated

**Traces to:** REQ-005, REQ-007, REQ-009, TASK-1, TASK-4, TASK-6

---

### ISSUE-053 Architecture Summary

| Decision | Choice |
|----------|--------|
| QA skill location | `skills/qa/SKILL.md` |
| Contract standard | `skills/shared/CONTRACT.md` |
| Contract extraction | Markdown text search for `## Contract` + YAML block |
| Orchestrator dispatch | Single `/qa` for all phases |
| Prose fallback | Extract from checklists, tables, "must" statements |
| Gap analysis | Compare QA checklists to producer contracts before deletion |
| Model | Haiku |
| Iteration limit | 3 |

**Traceability Matrix:**

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-019 | REQ-005, DES-013 |
| ARCH-020 | REQ-006, DES-001, DES-002 |
| ARCH-021 | REQ-006, DES-002, DES-008 |
| ARCH-022 | REQ-010, DES-004, DES-013 |
| ARCH-023 | REQ-005, DES-003, DES-005 |
| ARCH-024 | REQ-011, DES-008 |
| ARCH-025 | REQ-008 |
| ARCH-026 | REQ-010, DES-004 |
| ARCH-027 | REQ-005, DES-005, DES-006, DES-007, DES-009 |
| ARCH-028 | REQ-005, DES-012 |
| ARCH-029 | REQ-005 |
| ARCH-030 | REQ-005, REQ-007, REQ-009 |

---

## ISSUE-056: Inferred Specification Warning Architecture

Technical architecture for how producers detect and yield inferred specifications, and how the orchestrator handles them.

---

### ARCH-031: Inferred Yield Extension to YIELD.md

Extend the existing `need-user-input` yield type with an optional `inferred` flag rather than introducing a new yield type.

**Extension to YIELD.md:**

```toml
[yield]
type = "need-user-input"
timestamp = 2026-02-05T12:00:00Z

[payload]
inferred = true
question = "The following specifications were inferred and not explicitly requested. Accept or reject each."

[[payload.items]]
specification = "REQ-X: Input validation for empty strings"
reasoning = "Edge case: empty input could cause downstream errors"
source = "edge-case"

[[payload.items]]
specification = "REQ-Y: Rate limiting on API calls"
reasoning = "Implicit need: without rate limiting, external API costs could spike"
source = "implicit-need"

[context]
phase = "pm"
subphase = "SYNTHESIZE"
awaiting = "user-response"
```

**New payload fields when `inferred = true`:**
- `payload.inferred` (bool): Distinguishes inference confirmations from regular questions
- `payload.items` (array): Each item has `specification`, `reasoning`, and `source`
- `source` enum: `best-practice`, `edge-case`, `implicit-need`, `professional-judgment`

**Backward compatibility:** Existing `need-user-input` yields without `inferred` field continue working unchanged. The orchestrator checks for `payload.inferred` to determine handling.

**File to modify:** `skills/shared/YIELD.md`

**Traces to:** REQ-012, DES-014, ARCH-018, TASK-7

---

### ARCH-032: Orchestrator Inferred Yield Handling

The orchestrator adds a new branch in its `need-user-input` handler to detect and present inferred specifications.

**Modified resumption protocol (ARCH-018, step 2):**

```
2. **need-user-input** - Orchestrator checks payload:
   a. If payload.inferred = true:
      - Format items as numbered accept/reject list
      - Present to user with reasoning for each
      - Capture per-item accept/reject decisions
      - Write decisions to [query_results.inferred_decisions]
      - Re-invoke skill
   b. Else (regular question):
      - Prompt user, capture response (existing behavior)
      - Write to [query_results]
      - Re-invoke skill
```

**Response format written back to producer:**

```toml
[query_results]
[[query_results.inferred_decisions]]
specification = "REQ-X: Input validation for empty strings"
accepted = true

[[query_results.inferred_decisions]]
specification = "REQ-Y: Rate limiting on API calls"
accepted = false
```

**File to modify:** `skills/project/SKILL.md` (PAIR LOOP pattern), `skills/project/SKILL-full.md`

**Traces to:** REQ-014, DES-015, ARCH-018, TASK-8

---

### ARCH-033: Producer Inference Detection Pattern

Producers add an inference classification step between SYNTHESIZE and PRODUCE phases.

**Modified producer workflow:**

```
GATHER -> SYNTHESIZE -> CLASSIFY -> [YIELD INFERRED] -> PRODUCE
```

**CLASSIFY step:**

1. After SYNTHESIZE, producer reviews each specification it plans to create
2. For each specification, determine if it is:
   - **Explicit**: Directly traceable to user input, issue description, interview response, or gathered context document
   - **Inferred**: Added by the producer based on best practices, edge cases, implicit needs, or professional judgment
3. If any inferred items exist, yield `need-user-input` with `inferred = true` before proceeding to PRODUCE
4. After receiving user decisions, drop rejected items and proceed to PRODUCE with only explicit + accepted items

**Classification heuristic:**
- If a specification can be traced to a specific user statement, issue field, or context document passage: **explicit**
- If a specification requires the producer to make a judgment call or assumption: **inferred**
- When in doubt: classify as **inferred** (conservative approach)

**Shared documentation:** Add classification guidelines to `skills/shared/PRODUCER-TEMPLATE.md`

**Files to modify:**
- `skills/shared/PRODUCER-TEMPLATE.md` - Add CLASSIFY step and guidelines
- `skills/pm-interview-producer/SKILL.md` - Reference shared guidelines
- `skills/pm-infer-producer/SKILL.md` - Reference shared guidelines
- `skills/design-interview-producer/SKILL.md` - Reference shared guidelines
- `skills/design-infer-producer/SKILL.md` - Reference shared guidelines
- `skills/arch-interview-producer/SKILL.md` - Reference shared guidelines
- `skills/arch-infer-producer/SKILL.md` - Reference shared guidelines

**Traces to:** REQ-013, REQ-015, DES-016, TASK-9

---

### ISSUE-056 Architecture Summary

| Decision | Choice |
|----------|--------|
| Yield mechanism | `need-user-input` with `inferred = true` flag (not new type) |
| Orchestrator handling | New branch in need-user-input handler |
| Producer workflow | New CLASSIFY step between SYNTHESIZE and PRODUCE |
| Classification default | When in doubt, classify as inferred (conservative) |
| Source categories | best-practice, edge-case, implicit-need, professional-judgment |
| Affected files | YIELD.md, PRODUCER-TEMPLATE.md, project SKILL.md, 6 producer SKILL.md files |

**Traceability Matrix:**

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-031 | REQ-012, DES-014 |
| ARCH-032 | REQ-014, DES-015 |
| ARCH-033 | REQ-013, REQ-015, DES-016 |

