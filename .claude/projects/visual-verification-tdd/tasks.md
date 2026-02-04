# Visual Verification TDD Tasks

Implementation tasks for integrating visual verification into TDD workflow.

**Traces to:** requirements.md, design.md, architecture.md

---

## Dependency Graph

```
TASK-1 (tdd-red-producer: unified interface testing)
    |
TASK-2 (breakdown-producer: visual task detection)
    |
    +----TASK-3 (tdd-green-producer: visual verification step)
    |         |
    |    TASK-4 (tdd-qa: visual evidence requirement)
    |
TASK-5 (CLAUDE.md: lesson updates)
```

**Notes:**
- TASK-1 is foundational - defines the interface testing model referenced by other skills
- TASK-2 depends on TASK-1 (needs to understand what makes a task "visual")
- TASK-3 and TASK-4 can proceed in parallel after TASK-2
- TASK-5 is standalone but should come last to incorporate any refinements

---

### TASK-1: Update tdd-red-producer with unified interface testing model

**Description:** Add "Testing User Interfaces" section to tdd-red-producer SKILL.md documenting that UI, CLI, and API all follow the same testing model (structure + behavior + properties). Include concrete test examples for each interface type.

**Status:** Ready

**Acceptance Criteria:**
- [ ] New section "## Testing User Interfaces" added after "Test Philosophy" section
- [ ] Section includes table showing Structure/Behavior/Properties layers
- [ ] Structure testing documented for UI (element existence, properties)
- [ ] Structure testing documented for CLI (command parsing, arguments)
- [ ] Structure testing documented for API (endpoint existence, request/response shape)
- [ ] Behavior testing documented for UI (full event chain: interaction -> handler -> state -> UI)
- [ ] Behavior testing documented for CLI (command -> processing -> output)
- [ ] Behavior testing documented for API (request -> processing -> response)
- [ ] Property testing approach documented (invariants across all screens/commands/endpoints)
- [ ] UI testing examples subsection with structure, behavior, and property test code
- [ ] CLI testing examples subsection with structure and behavior test code
- [ ] API testing examples subsection with contract test code

**Files:** `~/.claude/skills/tdd-red-producer/SKILL.md`

**Dependencies:** None

**Traces to:** DES-1, DES-2, REQ-1, REQ-2, REQ-3, REQ-4, REQ-9, REQ-10

---

### TASK-2: Update breakdown-producer with visual task detection

**Description:** Add "Visual Task Detection" section to breakdown-producer SKILL.md explaining when to apply the `[visual]` marker to tasks based on files affected, description keywords, and acceptance criteria content.

**Status:** Ready

**Acceptance Criteria:**
- [ ] New section "## Visual Task Detection" added after "Task Format" section
- [ ] Heuristics documented: files created/modified include UI components, CSS, CLI output, templates
- [ ] Heuristics documented: description mentions display, render, button, dialog, output format
- [ ] Heuristics documented: acceptance criteria reference visual properties or design spec
- [ ] Example showing task with `[visual]` marker in title
- [ ] Task format example updated to show `### TASK-N: [visual] Title` syntax

**Files:** `~/.claude/skills/breakdown-producer/SKILL.md`

**Dependencies:** TASK-1

**Traces to:** DES-3, DES-4, REQ-7, REQ-8

---

### TASK-3: [visual] Update tdd-green-producer with visual verification step

**Description:** Add "Visual Verification" section to tdd-green-producer SKILL.md documenting the visual verification step for tasks with `[visual]` marker, including capture mechanisms, comparison approaches, and yield payload format with visual evidence fields.

**Status:** Ready

**Acceptance Criteria:**
- [ ] New section "## Visual Verification" added after "PRODUCE Phase" section
- [ ] Detection rule: check if task title contains `[visual]`
- [ ] Web UI capture mechanism: Chrome DevTools MCP `take_screenshot`
- [ ] CLI capture mechanism: output redirection and `script` command
- [ ] Comparison approach: `projctl screenshot diff` for baselines, manual review otherwise
- [ ] Yield payload example includes `visual_verified` and `visual_evidence` fields
- [ ] "No Screenshot Capture Tool" subsection documenting workarounds
- [ ] Capture approaches table: Interface -> Capture Method -> Tool

**Files:** `~/.claude/skills/tdd-green-producer/SKILL.md`

**Dependencies:** TASK-2

**Traces to:** DES-5, DES-7, REQ-5, REQ-6, REQ-8

---

### TASK-4: Update tdd-qa with visual evidence requirement

**Description:** Add "Visual Task Validation" subsection to tdd-qa SKILL.md under REVIEW Phase, requiring visual evidence for tasks with `[visual]` marker and documenting the waiver/escalation process.

**Status:** Ready

**Acceptance Criteria:**
- [ ] New subsection "### Visual Task Validation" under "1. REVIEW Phase"
- [ ] Check for `visual_verified = true` and `visual_evidence` path in producer yield
- [ ] `improvement-request` yield example for missing visual evidence
- [ ] Waiver process documented: producer must explain, QA escalates to user
- [ ] Visual evidence items added to acceptance criteria checklist
- [ ] Checklist includes: screenshot provided, visual matches AC, no obvious defects

**Files:** `~/.claude/skills/tdd-qa/SKILL.md`

**Dependencies:** TASK-2

**Traces to:** DES-6, REQ-5, REQ-8

---

### TASK-5: Update CLAUDE.md with interface testing lessons

**Description:** Expand existing CLAUDE.md lessons in "Code & Debugging" section to cover all user interfaces (UI, CLI, API), make visual verification non-optional, and add property-based interaction testing as standard practice.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Lesson "UI testing verifies visual correctness..." expanded to "Interface testing verifies correctness..." covering UI, CLI, and API
- [ ] Lesson "UI validation is critical..." generalized to "Interface validation is critical..." for all user-facing output
- [ ] New lesson added for property-based testing of interfaces (invariants across screens/commands/endpoints)
- [ ] Visual verification documented as non-optional for user-facing changes
- [ ] Examples cover all three interface types, not just UI

**Files:** `~/.claude/CLAUDE.md`

**Dependencies:** None

**Traces to:** DES-8, REQ-11

---

## Summary

| Task | Skill File | Design Sections | Estimated Lines |
|------|------------|-----------------|-----------------|
| TASK-1 | tdd-red-producer | DES-1, DES-2 | ~80 |
| TASK-2 | breakdown-producer | DES-3, DES-4 | ~25 |
| TASK-3 | tdd-green-producer | DES-5, DES-7 | ~50 |
| TASK-4 | tdd-qa | DES-6 | ~30 |
| TASK-5 | CLAUDE.md | DES-8 | ~20 |

**Total estimated changes:** ~205 lines across 5 files
