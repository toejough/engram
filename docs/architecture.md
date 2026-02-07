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

### ARCH-018: Orchestrator-Teammate Contract

**Phase:** 0
**Priority:** High
**Timeline:** This Week

Defines the communication protocol between the `/project` team lead and skill teammates. The lead spawns teammates with context in spawn prompts; teammates send results via SendMessage.

#### Context Input via Spawn Prompt

The team lead provides context to teammates via the Task tool spawn prompt:

```
Invoke the /pm-interview-producer skill for this project.

Project: my-project
Issue: ISSUE-NNN
Phase: pm
Docs dir: docs/
Requirements path: docs/requirements.md

Prior context:
<territory map summary>
<memory query results>
<issue description>

When complete, send me a message with:
- Artifact path
- IDs created (REQ-NNN list)
- Files modified
- Key decisions made
```

**Execution Modes:**
- **Interview** - Teammate uses AskUserQuestion directly for user interaction
- **Infer** - Teammate analyzes artifacts and generates content
- **Update** - Lightweight mode for `/project align`

#### Result Communication via SendMessage

Teammates send results to the team lead via SendMessage:

**Producer Message Types:**

| Pattern | Meaning | Lead Action |
|---------|---------|-------------|
| `complete: <summary>` | Work finished | Spawn QA teammate |
| AskUserQuestion | Question for user | User responds directly |
| `need-context: <query>` | **ELIMINATED** - teammates read files directly | N/A |
| `blocked: <reason>` | Cannot proceed | Present to user |
| `error: <details>` | Something failed | Retry or escalate |

**QA Message Types:**

| Pattern | Meaning | Lead Action |
|---------|---------|-------------|
| `approved` | Work passes QA | Advance to next phase |
| `improvement-request: <issues>` | Needs fixes | Spawn new producer with feedback |
| `escalate-phase: <reason>` | Prior phase issue | Return to prior phase |
| `escalate-user: <reason>` | Cannot resolve | Present to user |

#### Team Coordination Protocol

The team lead handles teammate messages as follows:

1. **complete: <summary>** - Lead spawns QA teammate with producer's artifact paths and SKILL.md location.

2. **AskUserQuestion** - Teammate asks user directly; no lead relay needed.

3. **blocked: <reason>** - Lead presents blocker to user, spawns replacement teammate after resolution.

4. **error: <details>** - Lead retries by spawning new teammate with same context (max 3x) or escalates to user.

5. **approved** (QA) - Lead advances to next phase via `projctl state transition`.

6. **improvement-request: <issues>** (QA) - Lead spawns new producer teammate with QA feedback in context.

7. **escalate-phase: <reason>** (QA) - Lead returns to prior phase, spawns that phase's producer with escalation context.

8. **escalate-user: <reason>** (QA) - Lead presents to user, awaits resolution, spawns appropriate teammate.

#### Context Injection in Spawn Prompts

The team lead injects context upfront when spawning teammates:

```
Invoke the /design-interview-producer skill for this project.

Project: my-project
Phase: design
Prior phase artifacts:
- docs/requirements.md (REQ-001 to REQ-015)

Relevant memory:
- Previous project: Material Design components worked well
- Learned: Keep mobile-first for responsive design

Territory context:
- Frontend: React with TypeScript
- UI components in src/components/

When complete, send me a message with artifact paths and IDs created.
```

Teammates read files directly using Read tool; no query-resume cycle needed.

**Traces to:** REQ-001, ARCH-001, ARCH-013

---

## ISSUE-53: Universal QA Skill Architecture

Technical architecture for replacing 13 phase-specific QA skills with one universal QA skill using team messaging.

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
- `skills/shared/` contains other cross-skill standards (PRODUCER-TEMPLATE.md, INTERVIEW-PATTERN.md)
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
- Invalid YAML: Send `error: contract parse failed: <details>` message

**Why not a YAML parser for the whole file?**
- SKILL.md is markdown, not YAML
- Simple text search is sufficient and more robust
- Avoids dependency on markdown-to-AST libraries

**Traces to:** REQ-006, DES-002, DES-008, TASK-2

---

### ARCH-022: Team Lead Dispatch to Universal QA

The team lead spawns the universal QA teammate for all phases.

**Dispatch pattern:**

| Phase         | QA Teammate |
| ------------- | ----------- |
| PM            | `qa`        |
| Design        | `qa`        |
| Architecture  | `qa`        |
| Breakdown     | `qa`        |
| TDD           | `qa`        |
| Documentation | `qa`        |
| Alignment     | `qa`        |
| Retro         | `qa`        |
| Summary       | `qa`        |

**QA teammate spawn prompt format:**

```
Invoke the /qa skill to validate the producer's output.

Producer SKILL.md: skills/design-interview-producer/SKILL.md
Artifact paths: docs/design.md
Iteration: 1

Context:
The design-interview-producer completed and reported:
- Created DES-001 through DES-012
- Modified: docs/design.md

Extract the contract from the producer's SKILL.md and validate the artifacts.

When complete, send me a message with:
- Verdict: approved | improvement-request
- If improvement-request: list of issues for the producer to fix
```

**Team Lead PAIR LOOP workflow:**

1. Spawn producer teammate with phase context
2. Receive completion message from producer
3. Spawn QA teammate with:
   - Producer SKILL.md path
   - Artifact paths from producer message
   - Iteration count
4. Receive verdict from QA teammate
5. If improvement-request: spawn new producer with feedback (max 3 iterations)
6. If approved: advance via `projctl state transition`

**Traces to:** REQ-010, DES-004, DES-013, ARCH-012, ARCH-018, TASK-5

---

### ARCH-023: QA Workflow

Universal QA follows a three-phase workflow: LOAD -> VALIDATE -> RETURN.

**Phase 1: LOAD**

1. Parse spawn prompt for producer SKILL.md path and artifact paths
2. Read producer SKILL.md using Read tool
3. Extract contract (ARCH-021) or fall back to prose (ARCH-024)
4. Read artifacts using Read tool

**Error handling in LOAD:**
- Producer SKILL.md not found: Send `error: cannot read producer SKILL.md` message
- Artifacts missing: Send `improvement-request: missing artifacts <list>` message

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

Based on validation results, send message to team lead:

| Condition | Message Pattern |
|-----------|-----------------|
| All checks pass | `approved` |
| Producer-fixable issues | `improvement-request: <issues>` |
| Upstream phase problem | `escalate-phase: <reason>` |
| Cannot resolve | `escalate-user: <reason>` |
| Max iterations reached | `escalate-user: max iterations reached` |

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

### ARCH-028: Iteration Tracking

QA tracks iterations to prevent infinite producer-QA loops.

**Mechanism:**

1. Team lead includes iteration count in QA spawn prompt
2. QA teammate includes iteration in response message
3. After max iterations (3), QA sends `escalate-user: max iterations reached` instead of `improvement-request`

**Message flow:**

```
Iteration 1:
  Lead -> QA: "Iteration: 1" in spawn prompt
  QA -> Lead: "improvement-request: <issues>"

Iteration 2:
  Lead -> QA: "Iteration: 2" in spawn prompt
  QA -> Lead: "improvement-request: <issues>"

Iteration 3:
  Lead -> QA: "Iteration: 3" in spawn prompt
  QA -> Lead: "escalate-user: max iterations reached with remaining issues: <list>"
  (Regardless of whether issues remain)
```

**Iteration tracking in lead:**

The team lead maintains PairState to track iteration count between producer-QA cycles.

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

### ISSUE-53 Architecture Summary

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
| ARCH-028 | REQ-005, DES-012 |
| ARCH-029 | REQ-005 |
| ARCH-030 | REQ-005, REQ-007, REQ-009 |

---

## ISSUE-56: Inferred Specification Warning Architecture

Technical architecture for how producers detect and send inferred specifications via AskUserQuestion, and how the team lead handles them.

---

### ARCH-031: Inferred Message Extension

Extend the existing user input message pattern with an optional `inferred` flag rather than introducing a new message type.

**Message pattern:**

Producer sends a message via SendMessage or AskUserQuestion with:
- `inferred = true` flag
- `items` array with each item containing `specification`, `reasoning`, and `source`
- `source` enum: `best-practice`, `edge-case`, `implicit-need`, `professional-judgment`

**Example:**
```
Producer uses AskUserQuestion with:
- question: "The following specifications were inferred. Accept or reject each."
- answers: List of inferred items with reasoning
```

**Backward compatibility:** Existing messages without `inferred` flag continue working unchanged. The orchestrator checks for `inferred` to determine handling.

**Traces to:** REQ-012, DES-014, ARCH-018, TASK-11

---

### ARCH-032: Orchestrator Inferred Message Handling

The orchestrator detects and presents inferred specifications from producer messages.

**Message handling:**

```
When orchestrator receives message from producer:
   a. If message contains inferred = true:
      - Format items as numbered accept/reject list
      - Present to user with reasoning for each
      - Capture per-item accept/reject decisions
      - Send decisions back to producer via SendMessage
   b. Else (regular message):
      - Handle normally (existing behavior)
```

**Response sent back to producer:**

Producer receives message with accepted/rejected decisions for each inferred item.

**File to modify:** `skills/project/SKILL.md` (PAIR LOOP pattern), `skills/project/SKILL-full.md`

**Traces to:** REQ-014, DES-015, ARCH-018, TASK-13

---

### ARCH-033: Producer Inference Detection Pattern

Producers add an inference classification step between SYNTHESIZE and PRODUCE phases.

**Modified producer workflow:**

```
GATHER -> SYNTHESIZE -> CLASSIFY -> [SEND INFERRED MESSAGE] -> PRODUCE
```

**CLASSIFY step:**

1. After SYNTHESIZE, producer reviews each specification it plans to create
2. For each specification, determine if it is:
   - **Explicit**: Directly traceable to user input, issue description, interview response, or gathered context document
   - **Inferred**: Added by the producer based on best practices, edge cases, implicit needs, or professional judgment
3. If any inferred items exist, send message to orchestrator with `inferred = true` before proceeding to PRODUCE
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

**Traces to:** REQ-013, REQ-015, DES-016, TASK-12, TASK-14

---

### ISSUE-56 Architecture Summary

| Decision | Choice |
|----------|--------|
| Communication mechanism | AskUserQuestion with `inferred = true` flag |
| Team lead handling | Format and present inferred items; relay user decisions to producer |
| Producer workflow | New CLASSIFY step between SYNTHESIZE and PRODUCE |
| Classification default | When in doubt, classify as inferred (conservative) |
| Source categories | best-practice, edge-case, implicit-need, professional-judgment |
| Affected files | PRODUCER-TEMPLATE.md, project SKILL.md, 6 producer SKILL.md files |

**Traceability Matrix:**

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-031 | REQ-012, DES-014, TASK-11 |
| ARCH-032 | REQ-014, DES-015, TASK-13 |
| ARCH-033 | REQ-013, REQ-015, DES-016, TASK-12, TASK-14 |

---

## ISSUE-92: Per-Phase QA in TDD Loop Architecture

Technical architecture for restructuring the TDD loop so each sub-phase (red, green, refactor) has its own producer/QA pair instead of deferring all QA to the end.

---

### ARCH-034: TDD Sub-Phase QA Phases

Add QA phases for each TDD sub-phase in the state machine.

**New state machine phases:**
- `tdd-red-qa`: QA validation after tdd-red-producer completes
- `tdd-green-qa`: QA validation after tdd-green-producer completes
- `tdd-refactor-qa`: QA validation after tdd-refactor-producer completes

**Phase registry entries:**

All three TDD sub-phases already have PhaseInfo entries in `internal/step/registry.go` with `QA: "qa"` and `QAPath: "skills/qa/SKILL.md"`.

**State transitions:**
```
tdd-red → tdd-red-qa → commit-red →
tdd-green → tdd-green-qa → commit-green →
tdd-refactor → tdd-refactor-qa → commit-refactor →
task-audit
```

**Rationale:** Immediate QA feedback after each TDD phase catches issues early. A failing test in red phase is caught before green/refactor happen. Smaller QA scope per phase makes validation focused and fast.

**Alternatives considered:**
- Keep QA at end (current): Issues compound, late detection
- QA between phases only (no commit QA): Misses commit-specific issues

**Traces to:** ISSUE-92

---

### ARCH-035: Commit-Phase Pair Loops

Add commit producer and QA phases for each TDD commit.

**New state machine phases:**
- `commit-red`: Producer creates commit after tdd-red-qa approval
- `commit-red-qa`: QA validates commit correctness
- `commit-green`: Producer creates commit after tdd-green-qa approval
- `commit-green-qa`: QA validates commit correctness
- `commit-refactor`: Producer creates commit after tdd-refactor-qa approval
- `commit-refactor-qa`: QA validates commit correctness

**Commit-producer responsibilities:**
1. Stage appropriate files (no secrets, no unrelated changes)
2. Generate conventional commit message
3. Create commit
4. Report commit hash and files staged

**Commit-QA validation criteria:**
1. Right files staged (matches phase scope)
2. No secrets committed (.env, credentials)
3. Commit message follows convention
4. Commit message describes change accurately
5. No blanket lint suppressions added

**Phase registry:**

Commit phases are not producer/QA pairs in the traditional sense. They use existing tools:
- Producer: Likely `/commit` skill or orchestrator-driven git commands
- QA: Validate via git status/log analysis

**Rationale:** Commit errors are expensive if caught late. Staging wrong files, committing secrets, or malformed messages break CI/CD. QA before pushing ensures commits are clean.

**Alternatives considered:**
- No commit QA: Relies on pre-commit hooks (unreliable, inconsistent)
- Single commit at end: Loses per-phase commit discipline

**Traces to:** ISSUE-92

---

### ARCH-036: Transition Enforcement

Update state machine transitions to enforce QA phases between producer and commit.

**Illegal transitions to prevent:**
- `tdd-red → commit-red` (must go through tdd-red-qa)
- `tdd-green → commit-green` (must go through tdd-green-qa)
- `tdd-refactor → commit-refactor` (must go through tdd-refactor-qa)
- `commit-red → tdd-green` (must go through commit-red-qa)
- `commit-green → tdd-refactor` (must go through commit-green-qa)
- `commit-refactor → task-audit` (must go through commit-refactor-qa)

**Implementation:**

Legal targets in `internal/state/transitions.go`:
```
"tdd-red": []string{"tdd-red-qa"},
"tdd-red-qa": []string{"commit-red"},
"commit-red": []string{"commit-red-qa"},
"commit-red-qa": []string{"tdd-green"},
...
```

**Verification:**

Tests in `internal/state/tdd_qa_phases_test.go` validate:
- Legal transitions succeed
- Illegal transitions fail with "illegal transition" error
- Full chain from tdd-red to task-audit works

**Rationale:** Enforcing transitions programmatically prevents skipping QA. State machine guarantees no shortcuts.

**Alternatives considered:**
- Soft enforcement (warnings): Unreliable, agents could skip
- Manual orchestrator logic: Error-prone, duplicates state machine knowledge

**Traces to:** ISSUE-92

---

### ARCH-037: projctl step next Integration

Update `projctl step next` to return QA actions between producer and commit.

**Current behavior:**

`projctl step next` reads the current phase from state and looks up the PhaseInfo in the registry. It returns the action for that phase (spawn producer or spawn QA).

**Expected behavior with ISSUE-92:**

When phase is `tdd-red`, `step next` returns:
```json
{
  "action": "spawn-producer",
  "skill": "tdd-red-producer",
  "skill_path": "skills/tdd-red-producer/SKILL.md",
  "model": "sonnet"
}
```

After producer completes and state transitions to `tdd-red-qa`, `step next` returns:
```json
{
  "action": "spawn-qa",
  "skill": "qa",
  "skill_path": "skills/qa/SKILL.md",
  "model": "haiku",
  "producer_skill_path": "skills/tdd-red-producer/SKILL.md"
}
```

**Implementation:**

The registry already has QA configured for tdd-red, tdd-green, tdd-refactor. `step next` logic in `internal/step/next.go` checks if the phase is in the registry and returns the appropriate action.

For commit phases (`commit-red`, `commit-green`, `commit-refactor`), the orchestrator handles these differently (not via step registry, but via state transitions).

**Rationale:** `step next` already drives per-phase QA for other phases (pm, design, arch). Extending to TDD sub-phases is consistent with existing pattern.

**Alternatives considered:**
- Orchestrator hardcodes TDD flow: Duplicates state machine knowledge
- No step next support: Forces manual orchestration

**Traces to:** ISSUE-92

---

### ARCH-038: TDD-QA Scope Reduction

Reduce `tdd-qa` (final QA after refactor) to a meta-check instead of full validation.

**Current tdd-qa responsibilities:**
- Verify all tests pass
- Verify implementation is complete
- Verify refactoring is done
- Verify AC coverage
- Verify no deferrals

**New tdd-qa responsibilities (meta-check only):**
- Did tdd-red complete with passing QA?
- Did tdd-green complete with passing QA?
- Did tdd-refactor complete with passing QA?
- Did all three commits happen?
- Are we ready to transition to task-audit?

**Rationale:** Per-phase QA already validates each step. tdd-qa becomes a lightweight sanity check that the full RED/GREEN/REFACTOR cycle completed properly.

**Alternatives considered:**
- Remove tdd-qa entirely: Loses explicit cycle completion verification
- Keep full tdd-qa validation: Duplicates per-phase QA work

**Traces to:** ISSUE-92

---

### ARCH-039: Commit Producer Skill Requirements

Define requirements for commit-producer skill behavior.

**Skill responsibilities:**

1. **Read current phase** - Determine scope (red/green/refactor)
2. **Stage appropriate files** - Use `git add <files>` for changed files in scope
3. **Validate no secrets** - Check staged files for .env, credentials, API keys
4. **Generate commit message** - Follow conventional commits format
5. **Create commit** - Execute `git commit` with generated message
6. **Report result** - Send completion message with commit hash, files staged

**Commit message format:**

```
<type>(<scope>): <description>

<optional body>

AI-Used: [claude]
```

Where `<type>` is one of: `feat`, `fix`, `test`, `refactor`, `docs`, `chore`.

**Staging rules by phase:**

| Phase | Files to Stage |
|-------|----------------|
| commit-red | Test files only (new tests from tdd-red) |
| commit-green | Test files + implementation files (from tdd-green) |
| commit-refactor | Implementation files only (refactored code from tdd-refactor) |

**Secret detection patterns:**

- `.env`, `.env.*` files
- `credentials.json`, `secrets.yaml`
- Files containing `API_KEY=`, `SECRET=`, `PASSWORD=`
- Private key patterns (`-----BEGIN PRIVATE KEY-----`)

**Rationale:** Explicit staging rules per phase prevent cross-phase contamination. Red commits should not include implementation code. Green commits should not include refactoring.

**Alternatives considered:**
- `/commit` skill handles all commits: Needs phase-aware logic
- Orchestrator creates commits directly: Loses commit skill reusability

**Traces to:** ISSUE-92

---

### ARCH-040: Commit-QA Validation Contract

Define validation criteria for commit-QA phases.

**Validation checks:**

| Check ID | Description | Severity |
|----------|-------------|----------|
| CHECK-COMMIT-001 | Files staged match phase scope | error |
| CHECK-COMMIT-002 | No secrets in staged files | error |
| CHECK-COMMIT-003 | Commit message follows conventional format | error |
| CHECK-COMMIT-004 | Commit message describes change accurately | warning |
| CHECK-COMMIT-005 | No blanket lint suppressions added | error |
| CHECK-COMMIT-006 | Commit created successfully | error |

**Phase-specific scope validation:**

| Phase | Expected Files |
|-------|----------------|
| commit-red-qa | Only test files (no implementation) |
| commit-green-qa | Test files + implementation (no refactoring-only changes) |
| commit-refactor-qa | Implementation files (behavior unchanged) |

**QA actions on failure:**

| Failure Type | QA Response |
|--------------|-------------|
| Wrong files staged | `improvement-request: unstage <files>, stage <correct-files>` |
| Secrets detected | `improvement-request: remove <files> from staging, add to .gitignore` |
| Bad commit message | `improvement-request: amend commit message to: <suggestion>` |
| Commit failed | `error: commit creation failed: <details>` |

**Rationale:** Automated commit validation catches common mistakes before they reach CI. Secret detection prevents credential leaks.

**Alternatives considered:**
- Pre-commit hooks only: Not enforced in orchestrator, inconsistent
- Manual review: Too slow, error-prone

**Traces to:** ISSUE-92

---

### ARCH-041: State Machine Changes Summary

Summary of all state machine modifications for ISSUE-92.

**New phases added (10 total):**
- `tdd-red-qa`
- `tdd-green-qa`
- `tdd-refactor-qa`
- `commit-red`
- `commit-red-qa`
- `commit-green`
- `commit-green-qa`
- `commit-refactor`
- `commit-refactor-qa`
- (tdd-qa scope changed, not new)

**Transition updates:**

```
OLD: tdd-red → commit-red → tdd-green → commit-green → tdd-refactor → commit-refactor → tdd-qa

NEW: tdd-red → tdd-red-qa → commit-red → commit-red-qa →
     tdd-green → tdd-green-qa → commit-green → commit-green-qa →
     tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa →
     task-audit
```

**Files to modify:**
- `internal/state/transitions.go` - Add new phases to legal targets
- `internal/step/registry.go` - Add commit-phase entries (if using registry for commits)
- `internal/state/state.go` - No changes needed (generic transition logic)

**Tests:**
- `internal/state/tdd_qa_phases_test.go` - Already exists with full test coverage

**Rationale:** Clean separation of QA and commit responsibilities. Each phase has a single concern.

**Traces to:** ISSUE-92

---

## ISSUE-92 Architecture Summary

| Decision | Choice |
|----------|--------|
| TDD sub-phase QA | tdd-red-qa, tdd-green-qa, tdd-refactor-qa phases |
| Commit pattern | commit-producer → commit-qa for each TDD phase |
| Transition enforcement | State machine prevents skipping QA |
| projctl step next | Returns QA actions between producer and commit |
| tdd-qa scope | Meta-check only (did the right steps happen?) |
| Commit staging rules | Phase-specific file scope (red=tests, green=tests+impl, refactor=impl) |
| Secret detection | Pre-commit validation for .env, credentials, API keys |
| Commit message format | Conventional commits with AI-Used trailer |

**Traceability Matrix:**

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-034 | ISSUE-92 |
| ARCH-035 | ISSUE-92 |
| ARCH-036 | ISSUE-92 |
| ARCH-037 | ISSUE-92 |
| ARCH-038 | ISSUE-92 |
| ARCH-039 | ISSUE-92 |
| ARCH-040 | ISSUE-92 |
| ARCH-041 | ISSUE-92 |

---

## ISSUE-104: Orchestrator as Haiku Teammate Architecture

Technical architecture for splitting the orchestrator into a team lead (opus) and orchestrator teammate (haiku) to reduce cost by using the cheapest model for mechanical step loop work.

---

### ARCH-042: Two-Role Split

Split the orchestrator into two distinct roles with clear separation of responsibilities.

**Role 1: Team Lead (Opus)**
- Owns the team via TeamCreate/TeamDelete
- Spawns teammates using Task tool
- Receives and relays messages between orchestrator and user
- Performs model handshake validation after spawning
- Runs end-of-command sequence after project completion
- **Never edits files or produces artifacts directly**

**Role 2: Orchestrator Teammate (Haiku)**
- Runs the `projctl step next` → dispatch → `projctl step complete` loop
- Manages project state via `projctl state` commands
- Sends spawn requests to team lead (cannot spawn directly)
- Sends shutdown requests to team lead when complete
- Tracks iteration counts and pair loop state
- Handles error recovery with retry-backoff logic

**Handoff protocol:**
```
User invokes /project
→ Team lead: TeamCreate, spawn orchestrator teammate with project name/issue
→ Orchestrator: Takes over, runs step loop until completion
→ Orchestrator: Sends "all-complete" message to team lead
→ Team lead: Runs end-of-command sequence, TeamDelete
```

**Rationale:**
- Haiku ($0.001/1K tokens input) is 30x cheaper than Opus ($0.03/1K tokens input) for mechanical work
- Step loop is deterministic JSON parsing and routing - doesn't require Opus reasoning
- Opus context preserved for user interaction and high-level decisions
- Clear responsibility boundaries prevent role confusion

**Alternatives considered:**
- Keep orchestrator in main conversation: Wastes Opus on mechanical work, current state
- Use Sonnet for orchestrator: Still 10x more expensive than Haiku for no added value
- Non-LLM orchestrator (ISSUE-1): Long-term goal, but requires external API integration

**Traces to:** REQ-016, ISSUE-104

---

### ARCH-043: Spawn Request Protocol

Orchestrator sends structured spawn requests to team lead when `projctl step next` returns `spawn-producer` or `spawn-qa`.

**Message format (via SendMessage):**

```json
{
  "type": "message",
  "recipient": "team-lead",
  "content": "spawn-request: pm-interview-producer",
  "summary": "Spawn teammate",
  "spawn_request": {
    "task_params": {
      "subagent_type": "code",
      "name": "pm-interview-producer",
      "model": "sonnet",
      "prompt": "First, respond with your model name...\n\nThen invoke /pm-interview-producer.\n\nIssue: ISSUE-104"
    },
    "expected_model": "sonnet",
    "action": "spawn-producer",
    "phase": "pm"
  }
}
```

**Team lead processing:**

1. Receives spawn request message
2. Extracts `task_params` from message
3. Calls `Task(subagent_type, name, model, prompt)` with extracted params
4. Validates model handshake (first teammate message contains expected_model substring, case-insensitive)
5. On handshake success: Sends confirmation to orchestrator
6. On handshake failure: Calls `projctl step complete --status failed --reported-model "<model>"`, sends failure message to orchestrator

**Confirmation message format:**

```
spawn-confirmed: pm-interview-producer

Teammate spawned successfully with correct model (sonnet).
Ready to receive work.
```

**Rationale:**
- Orchestrator cannot call Task tool directly (doesn't own the team)
- Team lead already has Task tool and team ownership
- Structured message format ensures all params are transmitted correctly
- Model handshake catches spawn failures early (before work begins)

**Alternatives considered:**
- Orchestrator spawns directly: Violates team ownership model
- Plain text spawn requests: Requires parsing, error-prone
- Team lead reads state file for spawn params: Couples implementation to filesystem

**Traces to:** REQ-017, REQ-021, ISSUE-104

---

### ARCH-044: Shutdown Request Protocol

Orchestrator sends shutdown request when `projctl step next` returns `all-complete`.

**Message format (via SendMessage):**

```
all-complete

Project completed successfully. Ready to shut down.

Summary:
- Requirements: REQ-001 through REQ-008 created
- Architecture: ARCH-001 through ARCH-004 created
- Design: DES-001 through DES-012 created
- Tasks: TASK-001 through TASK-015 completed
- All commits pushed
```

**Team lead processing:**

1. Receives all-complete message
2. Runs end-of-command sequence:
   - Display summary to user
   - Offer next steps (retro, summary, issue updates)
3. Sends `shutdown_request` to all active teammates (including orchestrator)
4. Waits for shutdown confirmations
5. Calls `TeamDelete()`
6. Reports completion to user

**Graceful shutdown flow:**
```
Orchestrator: "all-complete" →
Team lead: shutdown_request to all teammates →
Teammates: shutdown_response(approve=true) →
Team lead: TeamDelete() →
User sees: "Project complete"
```

**Rationale:**
- Team lead owns team lifecycle, must trigger TeamDelete
- Orchestrator reports completion but doesn't shut down unilaterally
- End-of-command sequence (retro prompt, summary offer) requires user interaction, belongs in team lead
- Graceful shutdown ensures no orphaned teammates

**Alternatives considered:**
- Orchestrator calls TeamDelete directly: Violates ownership (orchestrator doesn't own team)
- Auto-shutdown without confirmation: Could terminate active work
- No end-of-command sequence: Misses opportunity for retro/summary

**Traces to:** REQ-018, ISSUE-104

---

### ARCH-045: State Persistence Ownership

Orchestrator teammate owns state persistence; team lead never touches state files.

**Orchestrator responsibilities:**
- Call `projctl state init` on first run
- Call `projctl state set --workflow <type>` after workflow classification
- Call `projctl state set` after each `projctl step complete` to persist progress
- State includes: current phase, sub-phase, workflow type, active issue, pair loop iteration

**Team lead responsibilities:**
- None - team lead does not read or write state files
- Team lead receives state updates implicitly through orchestrator messages

**State file location:** `.claude/projects/<project-name>/state.toml`

**Resumption support:**
- If orchestrator crashes/terminates mid-session, team lead can respawn orchestrator
- Respawned orchestrator reads state via `projctl state get` and resumes from last saved phase
- No state is lost as long as `projctl step complete` was called before termination

**Rationale:**
- Orchestrator runs the step loop, so it owns state transitions
- Keeping state management in one place (orchestrator) eliminates sync issues
- Team lead doesn't need state file access - it coordinates via messages
- Aligns with REQ-022 (state persisted after each step)

**Alternatives considered:**
- Team lead manages state: Adds coordination overhead, orchestrator must send updates
- Shared state management: Risk of conflicting writes
- No state persistence: Cannot resume after crashes

**Traces to:** REQ-016, REQ-020, REQ-022, ISSUE-104

---

### ARCH-046: Error Handling with Retry-Backoff

Orchestrator implements automatic retry with exponential backoff before escalating errors to team lead.

**Retry logic:**

```
max_retries = 3
backoff_delays = [1s, 2s, 4s]

function executeStepWithRetry(action):
  for attempt in 1..max_retries:
    result = executeStep(action)
    if result.success:
      return result
    else:
      log("Attempt {attempt} failed: {result.error}")
      if attempt < max_retries:
        sleep(backoff_delays[attempt - 1])
      else:
        escalateToTeamLead(result.error)
```

**Errors that trigger retry:**
- `projctl step next` command fails (exit code != 0)
- `projctl step complete` command fails
- Spawn confirmation timeout (teammate doesn't respond within reasonable time)
- JSON parse errors from step next output

**Errors that skip retry (immediate escalation):**
- User cancellation signals
- State file corruption (invalid TOML)
- Team lead shutdown request

**Escalation message format:**

```
error: step execution failed after 3 attempts

Action: spawn-producer
Phase: pm
Error: projctl step next exited with code 1

Output:
<command output>

Please investigate and provide guidance.
```

**Rationale:**
- Transient errors (network hiccups, filesystem delays) often resolve on retry
- Exponential backoff prevents hammering failing resources
- Max 3 retries balances recovery chance vs. time wasted
- Escalation to team lead (and thus user) for persistent errors

**Alternatives considered:**
- No retry: Fails immediately on transient errors, requires manual restart
- Infinite retry: Could loop forever on persistent errors
- Team lead retries: Pushes retry logic to wrong layer

**Traces to:** REQ-019, REQ-023, ISSUE-104

---

### ARCH-047: Orchestrator Model Selection

Orchestrator teammate always uses Haiku model for step loop execution.

**Frontmatter in project SKILL.md:**

```yaml
---
name: project
description: State-machine-driven project orchestrator (team lead)
model: haiku
user-invocable: true
---
```

**Note:** The `model: haiku` metadata in SKILL.md frontmatter does NOT change the model when the skill is loaded in the main conversation. This metadata is advisory only. To actually run haiku, the orchestrator must be spawned as a teammate via Task tool.

**Team lead spawn call:**

```
Task(
  subagent_type: "code",
  name: "orchestrator",
  model: "haiku",
  prompt: "You are the orchestrator teammate...",
  team_name: "<project-name>"
)
```

**Model handshake validation:**
After spawning orchestrator, team lead reads first message and verifies it contains "haiku" (case-insensitive substring match).

**Cost comparison per 10K tokens:**
- Opus: $0.30 input
- Sonnet: $0.10 input
- Haiku: $0.01 input

**Orchestrator workload estimate:**
- Typical project: 50-100 step loop iterations
- Each iteration: ~500 tokens (read JSON, parse, route, call tools)
- Total: 25K-50K tokens per project

**Cost savings:**
- Opus: $0.75-$1.50 per project
- Haiku: $0.025-$0.05 per project
- **Savings: 30x reduction (96.7% cheaper)**

**Rationale:**
- Step loop is mechanical: JSON parsing, routing, tool calls - no complex reasoning needed
- Haiku is sufficient for deterministic workflows
- Opus reserved for user-facing decisions and complex problem solving
- Cost optimization aligns with REQ-001 ("cheapest agents + smallest context possible")

**Alternatives considered:**
- Sonnet for orchestrator: Still 10x more expensive, no added value
- Opus for orchestrator: Current wasteful state, 30x too expensive

**Traces to:** REQ-016, ARCH-003, ISSUE-104

---

### ARCH-048: Team Lead Spawn Orchestrator on /project

Team lead spawns orchestrator teammate immediately after TeamCreate on `/project` invocation.

**Startup sequence:**

```
User: /project <project-name> ISSUE-NNN

Team lead (opus):
  1. TeamCreate(team_name: "<project-name>", description: "...")
  2. Task(
       subagent_type: "code",
       name: "orchestrator",
       model: "haiku",
       prompt: "You are the orchestrator teammate for project <project-name>.\n\n
                Run the step-driven control loop as documented in project SKILL.md.\n\n
                Project: <project-name>\n
                Issue: ISSUE-NNN\n\n
                Start by calling `projctl state init` and entering the step loop.",
       team_name: "<project-name>"
     )
  3. Validate model handshake (orchestrator's first message contains "haiku")
  4. Wait for messages from orchestrator (spawn requests, completion, errors)
```

**Orchestrator startup actions:**

```
Orchestrator (haiku):
  1. projctl state init --name "<project-name>" --issue ISSUE-NNN
  2. projctl state set --workflow <new|task|adopt|align>
  3. Enter step loop:
     loop:
       result = projctl step next
       handle(result.action)
       projctl step complete
```

**Team lead idle state:**
After spawning orchestrator, team lead enters idle state waiting for messages. Team lead does NOT poll or check in - orchestrator drives all work.

**Rationale:**
- Clean separation: team lead spawns, orchestrator executes
- Orchestrator takes over immediately after spawn confirmation
- Team lead remains responsive to user questions while orchestrator works
- Aligns with REQ-016 (split roles from startup)

**Alternatives considered:**
- Team lead runs first few steps: Blurs role boundaries
- Orchestrator waits for explicit "start" signal: Adds unnecessary coordination round-trip
- No model handshake: Risk of wrong model running orchestrator

**Traces to:** REQ-016, REQ-017, ISSUE-104

---

### ARCH-049: Resumption After Orchestrator Termination

Team lead can respawn orchestrator teammate after unexpected termination without losing progress.

**Resumption trigger scenarios:**
- Orchestrator crashed (unhandled exception, OOM)
- User manually terminated orchestrator agent
- Network/API timeout killed orchestrator session
- Context limit hit (unlikely with haiku + short step loop)

**Resumption flow:**

```
Team lead detects orchestrator gone:
  1. Check if project state exists (.claude/projects/<name>/state.toml)
  2. If state exists:
     - Respawn orchestrator with same spawn params
     - Orchestrator reads state via `projctl state get`
     - Orchestrator resumes from last saved phase
  3. If state missing:
     - Report to user: "Cannot resume - no state file"
     - Offer to start new project
```

**Orchestrator resumption logic:**

```
On startup:
  state = projctl state get --format json
  if state.phase != "":
    log("Resuming from phase: {state.phase}")
    # Skip init, go straight to step loop
    enterStepLoop()
  else:
    log("No prior state, starting fresh")
    projctl state init
    projctl state set --workflow <type>
    enterStepLoop()
```

**State persistence guarantees:**
- State saved after every `projctl step complete` call
- Atomic write (temp file + rename)
- No partial state (write succeeds or fails completely)

**Work not lost:**
- Completed phases and their artifacts (docs/requirements.md, etc.)
- Completed commits (git history persists)
- Pair loop iteration counts and QA feedback

**Work that may repeat:**
- Current in-progress phase (if orchestrator crashed mid-step)
- Latest step loop iteration (since last `step complete`)

**Rationale:**
- State file is single source of truth for progress
- Orchestrator crash doesn't require restarting entire project
- User doesn't lose multi-hour project work due to transient failure
- Aligns with REQ-020 (resumption support) and REQ-022 (state persistence)

**Alternatives considered:**
- No resumption: User must restart project from scratch
- Team lead tracks state: Duplicates state management, sync issues
- Checkpoint files: Overcomplicates state management

**Traces to:** REQ-020, REQ-022, ARCH-045, ISSUE-104

---

### ARCH-050: Team Lead Delegation-Only Mode

Team lead never edits files or produces artifacts directly; always delegates to spawned teammates.

**Team lead allowed actions:**
- TeamCreate / TeamDelete
- Task (spawn teammates)
- SendMessage (communicate with teammates)
- AskUserQuestion (user interaction)
- Read (read files for context to pass to teammates)

**Team lead prohibited actions:**
- Write (create files)
- Edit (modify files)
- NotebookEdit (modify notebooks)
- Bash (run git commit, build commands, tests)

**Enforcement:**
- SKILL.md documents prohibition prominently in "DO NOT" column
- Team lead self-monitors during execution
- If team lead catches itself about to Write/Edit, stops and spawns appropriate teammate instead

**Example violation prevention:**

```
User: "Update requirements.md with REQ-009"

Team lead thinks: "I should edit requirements.md"
Team lead catches: "Wait, I'm in delegate mode"
Team lead instead: Spawn pm-interview-producer with context "Add REQ-009 to requirements.md"
```

**Rationale:**
- Clear separation of concerns: team lead coordinates, teammates do work
- Prevents context pollution in team lead (opus) from file content
- Maintains team lead's availability for user interaction
- Enforces delegation discipline consistently

**Alternatives considered:**
- Allow team lead to edit in emergencies: Slippery slope, breaks discipline
- Soft guideline instead of prohibition: Easy to violate, inconsistent

**Traces to:** REQ-016, ARCH-042, ISSUE-104

---

### ARCH-051: SKILL.md Documentation Updates

Update project SKILL.md to document the two-role split and orchestrator spawn pattern.

**Files to modify:**

1. **skills/project/SKILL.md**
   - Add "Team Lead Mode" section documenting delegation-only behavior
   - Add orchestrator spawn sequence to "Startup" section
   - Update step loop description to clarify team lead receives messages, doesn't run loop
   - Add spawn request/confirmation protocol documentation

2. **skills/project/SKILL-full.md**
   - Add detailed orchestrator teammate behavior section
   - Document state persistence ownership (orchestrator)
   - Add resumption flow documentation
   - Add error handling and retry-backoff details

**New sections to add:**

```markdown
## Two-Role Architecture

The `/project` skill operates in a two-role architecture:

1. **Team Lead (Opus)** - You are here
   - Owns the team (TeamCreate, TeamDelete)
   - Spawns teammates including orchestrator
   - Receives messages from orchestrator
   - Relays spawn/shutdown requests
   - Never edits files directly

2. **Orchestrator Teammate (Haiku)** - Spawned by you
   - Runs the `projctl step next` loop
   - Manages project state
   - Sends spawn/shutdown requests to you
   - Handles retries and error recovery

## Orchestrator Spawn

On `/project` invocation:

1. TeamCreate(...)
2. Spawn orchestrator teammate:
   ```
   Task(subagent_type: "code",
        name: "orchestrator",
        model: "haiku",
        prompt: "Run the step loop for project <name>...",
        team_name: "<project-name>")
   ```
3. Validate model handshake (first message contains "haiku")
4. Enter idle state, wait for orchestrator messages
```

**Rationale:**
- Documentation in SKILL.md serves as prompt for both team lead and orchestrator
- Clear role boundaries prevent confusion
- Spawn pattern documented for consistency
- Aligns with REQ-016 (document two-role split)

**Traces to:** REQ-016, ARCH-042, ARCH-048, ISSUE-104

---

## ISSUE-104 Architecture Summary

| Decision | Choice |
|----------|--------|
| Role split | Team lead (opus) + orchestrator teammate (haiku) |
| Orchestrator model | Haiku (30x cheaper for mechanical work) |
| Spawn protocol | SendMessage with structured spawn_request JSON |
| Shutdown protocol | Orchestrator sends "all-complete", team lead handles TeamDelete |
| State persistence | Orchestrator owns state via projctl commands |
| Error handling | Retry with exponential backoff (1s, 2s, 4s) before escalation |
| Resumption | State-file-based, orchestrator reads state on respawn |
| Team lead mode | Delegation-only, never edits files directly |
| Spawn timing | Immediate after TeamCreate on /project invocation |
| Model handshake | Team lead validates first message contains expected model |

**Cost savings:**
- Current: Opus for entire orchestration loop (~$0.75-$1.50 per project)
- New: Haiku for step loop (~$0.025-$0.05 per project)
- **Reduction: 30x (96.7% cost savings)**

**Traceability Matrix:**

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-042 | REQ-016, ISSUE-104 |
| ARCH-043 | REQ-017, REQ-021, ISSUE-104 |
| ARCH-044 | REQ-018, ISSUE-104 |
| ARCH-045 | REQ-016, REQ-020, REQ-022, ISSUE-104 |
| ARCH-046 | REQ-019, REQ-023, ISSUE-104 |
| ARCH-047 | REQ-016, ARCH-003, ISSUE-104 |
| ARCH-048 | REQ-016, REQ-017, ISSUE-104 |
| ARCH-049 | REQ-020, REQ-022, ARCH-045, ISSUE-104 |
| ARCH-050 | REQ-016, ARCH-042, ISSUE-104 |
| ARCH-051 | REQ-016, ARCH-042, ARCH-048, ISSUE-104 |

---

