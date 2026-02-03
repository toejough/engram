# Layer -1: Skill Unification Architecture

**Project:** Layer -1 Skill Unification
**Issue:** ISSUE-008
**Created:** 2026-02-02
**Status:** Draft

**Traces to:** DES-1, DES-2, DES-3, DES-4, DES-5, DES-6, DES-7

---

## Current State

```
skills/
├── shared/
│   ├── CONTEXT.md          # Context input format
│   ├── RESULT.md           # Result output format (old)
│   ├── SKILL-TEMPLATE-COMPRESSED.md
│   └── ownership-rules/    # Shared skill
├── pm-interview/
│   ├── SKILL.md            # Summary
│   └── SKILL-full.md       # Full docs
├── pm-infer/
├── pm-audit/               # To be deleted
├── ... (21 skill directories)
└── project/                # Orchestrator skill
```

---

## Architecture Decisions

### ARCH-1: Directory Structure

Maintain flat structure with new naming convention:

```
skills/
├── shared/
│   ├── CONTEXT.md          # Context input (unchanged)
│   ├── YIELD.md            # Yield protocol (replaces RESULT.md)
│   ├── PRODUCER-TEMPLATE.md
│   ├── QA-TEMPLATE.md
│   └── ownership-rules/
│
├── pm-interview-producer/
│   ├── SKILL.md
│   └── SKILL-full.md
├── pm-infer-producer/
├── pm-qa/
│
├── design-interview-producer/
├── design-infer-producer/
├── design-qa/
│
├── arch-interview-producer/
├── arch-infer-producer/
├── arch-qa/
│
├── breakdown-producer/
├── breakdown-qa/
│
├── doc-producer/
├── doc-qa/
│
├── tdd-red-producer/
├── tdd-red-infer-producer/
├── tdd-red-qa/
├── tdd-green-producer/
├── tdd-green-qa/
├── tdd-refactor-producer/
├── tdd-refactor-qa/
├── tdd-qa/                 # Overall TDD quality gate
│
├── alignment-producer/
├── alignment-qa/
├── retro-producer/
├── retro-qa/
├── summary-producer/
├── summary-qa/
│
├── context-explorer/       # Handles need-context queries
├── intake-evaluator/
├── next-steps/
├── commit/                 # Unchanged
│
└── project/                # Orchestrator (updated for new dispatch)
```

**Rationale:** Flat structure is simpler, skill names are self-documenting per DES-1.

**Traces to:** DES-1

---

### ARCH-2: Shared Content

Update shared directory for yield protocol:

```
shared/
├── CONTEXT.md              # Context input format (minor updates)
├── YIELD.md                # Yield protocol format (NEW, replaces RESULT.md)
├── PRODUCER-TEMPLATE.md    # Template for producer skills
├── QA-TEMPLATE.md          # Template for QA skills
├── EXPLORER-TEMPLATE.md    # Template for context-explorer
└── ownership-rules/        # Unchanged
```

**YIELD.md** documents:
- Yield types (complete, need-user-input, need-context, blocked, approved, improvement-request, escalate-phase)
- Payload formats per type
- Context serialization for resumption

**CONTEXT.md** updates:
- Add `output.yield_path` field
- Document query result injection for need-context resumption

**Traces to:** DES-2, DES-3

---

### ARCH-3: Skill File Structure

Each skill directory contains:

```
<skill-name>/
├── SKILL.md                # Summary (per DES-3 template)
└── SKILL-full.md           # Full documentation (optional)
```

**SKILL.md frontmatter:**
```yaml
---
name: pm-interview-producer
description: Gather requirements via user interview
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer                # NEW: producer | qa | standalone
phase: pm                     # NEW: pm | design | arch | breakdown | doc | tdd-red | etc.
variant: interview            # NEW: interview | infer (optional)
---
```

New frontmatter fields enable programmatic skill discovery and validation.

**Traces to:** DES-3, DES-5

---

### ARCH-4: Skill Dependencies

Skills reference shared content via relative paths:

```markdown
## Yield Format

See [shared/YIELD.md](../shared/YIELD.md) for full protocol.

This skill yields:
- `complete` with requirements artifact
- `need-user-input` for interview questions
- `need-context` for gathering existing docs
```

**Dependency graph:**
```
All skills ──► shared/YIELD.md
All skills ──► shared/CONTEXT.md
Producers ──► shared/PRODUCER-TEMPLATE.md (structure reference)
QA skills ──► shared/QA-TEMPLATE.md (structure reference)
All skills ──► shared/ownership-rules/ (skill dependency)
```

**Traces to:** DES-3, DES-4

---

### ARCH-5: Installation Process

Skills are installed via symlinks from `~/.claude/skills/`:

```bash
# Install script: scripts/install-skills.sh

SKILLS_DIR="$HOME/.claude/skills"
REPO_SKILLS="$(dirname $0)/../skills"

# Remove old symlinks
rm -f "$SKILLS_DIR/pm-interview" "$SKILLS_DIR/pm-audit" ...

# Create new symlinks
ln -sf "$REPO_SKILLS/pm-interview-producer" "$SKILLS_DIR/pm-interview-producer"
ln -sf "$REPO_SKILLS/pm-qa" "$SKILLS_DIR/pm-qa"
...

# Shared content
ln -sf "$REPO_SKILLS/shared" "$SKILLS_DIR/shared"
```

**Migration approach:**
1. Create new skill directories alongside old ones
2. Test new skills independently
3. Run install script to switch symlinks atomically
4. Delete old skill directories after validation

**Traces to:** REQ-7

---

### ARCH-6: /project Skill Updates

The `/project` orchestrator skill needs updates:

```markdown
## Changes Required

1. **Dispatch to new skill names**
   - pm-interview → pm-interview-producer
   - pm-audit → pm-qa
   - etc.

2. **Parse yield protocol**
   - Read .claude/context/<skill>-yield.toml
   - Handle all yield types
   - Implement pair loop logic

3. **Handle need-context yields**
   - Dispatch to context-explorer
   - Aggregate results
   - Resume producer with enriched context

4. **Provide unique yield paths**
   - Include session ID in yield_path
   - Track which yield corresponds to which invocation
```

**Traces to:** DES-6, REQ-8

---

### ARCH-7: Context Explorer Architecture

Single agent handling all query types (B1 approach):

```
context-explorer/
├── SKILL.md
└── SKILL-full.md
```

**Input:** List of queries from need-context yield

**Process:**
1. Parse query list
2. Execute each query (may parallelize internally via Task tool)
3. Aggregate results
4. Return unified context

**Query handlers:**
| Type | Implementation |
|------|----------------|
| file | Read tool |
| memory | projctl memory query (or direct ONNX if available) |
| territory | projctl territory map |
| web | WebFetch tool |
| semantic | Explore agent (Task tool) |

**Output:** Aggregated context for producer resumption

**Traces to:** DES-7, REQ-10

---

## File Changes Summary

| Action | Path | Notes |
|--------|------|-------|
| CREATE | skills/pm-interview-producer/ | New producer |
| CREATE | skills/pm-infer-producer/ | New producer |
| CREATE | skills/pm-qa/ | New QA |
| CREATE | skills/design-interview-producer/ | New producer |
| CREATE | skills/design-infer-producer/ | New producer |
| CREATE | skills/design-qa/ | New QA |
| CREATE | skills/arch-interview-producer/ | New producer |
| CREATE | skills/arch-infer-producer/ | New producer |
| CREATE | skills/arch-qa/ | New QA |
| CREATE | skills/breakdown-producer/ | Rename from task-breakdown |
| CREATE | skills/breakdown-qa/ | New QA |
| CREATE | skills/doc-producer/ | New producer |
| CREATE | skills/doc-qa/ | New QA |
| CREATE | skills/tdd-red-producer/ | Rename from tdd-red |
| CREATE | skills/tdd-red-infer-producer/ | New producer |
| CREATE | skills/tdd-red-qa/ | New QA |
| CREATE | skills/tdd-green-producer/ | Rename from tdd-green |
| CREATE | skills/tdd-green-qa/ | New QA |
| CREATE | skills/tdd-refactor-producer/ | Rename from tdd-refactor |
| CREATE | skills/tdd-refactor-qa/ | New QA |
| CREATE | skills/tdd-qa/ | New overall TDD QA |
| CREATE | skills/alignment-producer/ | Rename from alignment-check |
| CREATE | skills/alignment-qa/ | New QA |
| CREATE | skills/retro-producer/ | New producer |
| CREATE | skills/retro-qa/ | New QA |
| CREATE | skills/summary-producer/ | New producer |
| CREATE | skills/summary-qa/ | New QA |
| CREATE | skills/context-explorer/ | New explorer |
| CREATE | skills/intake-evaluator/ | New standalone |
| CREATE | skills/next-steps/ | New standalone |
| UPDATE | skills/commit/ | Minor updates for yield protocol |
| UPDATE | skills/project/ | Major updates for new dispatch |
| UPDATE | skills/shared/CONTEXT.md | Add yield_path |
| CREATE | skills/shared/YIELD.md | New yield protocol doc |
| CREATE | skills/shared/PRODUCER-TEMPLATE.md | New template |
| CREATE | skills/shared/QA-TEMPLATE.md | New template |
| DELETE | skills/pm-interview/ | After migration |
| DELETE | skills/pm-infer/ | After migration |
| DELETE | skills/pm-audit/ | Merged into pm-qa |
| DELETE | skills/design-interview/ | After migration |
| DELETE | skills/design-infer/ | After migration |
| DELETE | skills/design-audit/ | Merged into design-qa |
| DELETE | skills/architect-interview/ | After migration |
| DELETE | skills/architect-infer/ | After migration |
| DELETE | skills/architect-audit/ | Merged into arch-qa |
| DELETE | skills/task-breakdown/ | Renamed |
| DELETE | skills/task-audit/ | Merged into tdd-qa |
| DELETE | skills/tdd-red/ | Renamed |
| DELETE | skills/tdd-green/ | Renamed |
| DELETE | skills/tdd-refactor/ | Renamed |
| DELETE | skills/alignment-check/ | Renamed |
| DELETE | skills/negotiate/ | Merged into QA escalate |
| DELETE | skills/meta-audit/ | Merged into retro |
| DELETE | skills/test-mapper/ | Obsolete |
| DELETE | skills/shared/RESULT.md | Replaced by YIELD.md |

---

## Open Questions

None - architecture follows directly from design decisions.
