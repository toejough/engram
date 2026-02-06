# Design: Clean Up Remaining Yield References

**Issue:** ISSUE-88
**Project:** Clean up remaining yield references in docs
**Date:** 2026-02-06

## Overview

This design documents the approach for systematically finding and removing all references to the deprecated yield infrastructure from documentation and code. The yield system (`.toml` files for skill communication) was replaced with direct teammate messaging (`SendMessage` tool) in the team-based orchestration model.

## Design Decisions

### DES-001: Search Strategy for Yield References

**Description:** Use multi-pattern grep approach to identify all yield-related content across documentation and code.

**Search Patterns:**
1. **Case-insensitive "yield" keyword**: `grep -i "yield"` to catch all variations
2. **Yield TOML files**: `find . -name "*yield*.toml"` to find actual yield artifacts
3. **Yield directories**: `find . -type d -name "yields"` to find yield storage directories
4. **Yield-specific terms**:
   - `yield_path` (context configuration)
   - `producer_yield_path` (QA context configuration)
   - `yield.type` (TOML access patterns)
   - `[yield]` (TOML section headers)

**File Scopes:**
- All `SKILL.md` files in `skills/*/SKILL.md`
- All `SKILL_test.sh` files in `skills/*/SKILL_test.sh`
- All markdown files in `docs/` directory
- All markdown files in `.claude/projects/*/` directories
- Root-level markdown files (`README.md`, etc.)
- Source code files in `internal/`, `cmd/` (for orphaned code references)
- Configuration files (`.toml`, `.json`) in project directories

**Exclusions:**
- This requirements document (`.claude/projects/issue-88/requirements.md`)
- This design document (`.claude/projects/issue-88/design.md`)
- Historical issue files that document the yield removal itself
- Test fixtures that intentionally preserve old formats for migration testing

**Traces to:** REQ-001, REQ-002, REQ-003, ISSUE-88

---

### DES-002: Replacement Strategy for Skill Documentation

**Description:** Map each yield-related concept to its current team-based equivalent in skill documentation.

**Replacement Mappings:**

| Old Yield Concept | New Team-Based Concept | Replacement Text |
|------------------|------------------------|------------------|
| Write yield TOML file | Send message to team lead | `SendMessage` tool with `type: "message"`, `recipient: "team-lead"` |
| `yield.type = "complete"` | Report completion | Send completion message with artifact paths |
| `yield.type = "blocked"` | Report blocker | Send blocker message explaining obstacle |
| `yield.type = "need-context"` | Request information | Send message requesting context or ask user via `AskUserQuestion` |
| `yield.type = "escalate-user"` | Ask user directly | Use `AskUserQuestion` tool |
| `yield.type = "approved"` (QA) | Report QA pass | Send approval message to team lead |
| `yield.type = "improvement-request"` (QA) | Report QA issues | Send improvement-request message with findings |
| `output.yield_path` in context | Direct messaging | No configuration needed; use `SendMessage` directly |
| Read yield file | Receive teammate message | Messages delivered automatically via system |

**Section Updates:**
- Replace "Yield Protocol" sections with "Communication Protocol" sections
- Update workflow diagrams showing yield file I/O to show direct messaging
- Change completion criteria from "write yield file" to "send completion message"
- Update examples showing TOML structure to show `SendMessage` tool usage

**Documentation Consistency:**
- Ensure all skills use the same messaging patterns
- Standardize on "send message to team-lead" language
- Remove references to file paths for communication artifacts

**Traces to:** REQ-002, REQ-004, ISSUE-88

---

### DES-003: Replacement Strategy for Architecture Documentation

**Description:** Update architecture documents to reflect the team-based communication model.

**Files to Update:**
- `docs/architecture.md` - Main architecture reference
- `docs/design.md` - Design decisions
- `docs/orchestration-system.md` - Orchestration flow
- Project-specific architecture files in `.claude/projects/*/architecture.md`

**Content Changes:**
1. **Contract sections**: Replace yield schema specifications with messaging protocol
2. **Flow diagrams**: Update to show `SendMessage` instead of file writes
3. **State machine descriptions**: Remove yield file reading steps, add message handling
4. **Protocol specifications**: Replace TOML schemas with message payload formats
5. **Resumption logic**: Replace yield-type-based branching with message-based continuation

**Conceptual Mappings:**
- "Orchestrator reads yield" → "Team lead receives message"
- "Yield type determines next action" → "Message content determines next action"
- "Yield schema" → "Message payload schema"
- "Context TOML with yield_path" → "No configuration needed; messaging is built-in"

**Traces to:** REQ-003, REQ-004, ISSUE-88

---

### DES-004: Strategy for Yield Artifact Files

**Description:** Handle actual yield TOML files and yield directories in project history.

**Categorization:**
1. **Active yield files** (`.claude/yield.toml`, `yield.toml` at root): Delete entirely
2. **Historical yield directories** (`.claude/projects/*/yields/`): Preserve as historical artifacts but document as deprecated
3. **Yield files in completed projects** (`.claude/projects/ISSUE-*/yields/`): Keep for historical reference
4. **Context files referencing yield paths**: Update to remove yield_path fields or mark deprecated

**Deletion Strategy:**
- Delete root-level `yield.toml` and `.claude/yield.toml` (active/stale files)
- Keep yield directories in closed issue projects (e.g., `.claude/projects/ISSUE-61/yields/`) as historical record
- Update any active context TOML files to remove `output.yield_path` fields

**Documentation Updates:**
- Add note to project READMEs explaining yield directories are historical
- Document migration from yield to messaging in appropriate architecture sections

**Traces to:** REQ-001, REQ-003, ISSUE-88

---

### DES-005: Edge Case Handling

**Description:** Define how to handle ambiguous or partial yield references.

**Edge Cases:**

1. **Historical context mentions**: When documentation describes "how we used to do X with yields", keep if explaining migration path, remove if purely historical
   - **Action**: Preserve if in retrospective/learning docs, remove if in active workflow docs

2. **Partial references** (e.g., "the skill yields a result"): Common English word usage vs technical term
   - **Action**: Evaluate context; if clearly non-technical English usage, keep; if ambiguous or technical, rephrase to avoid confusion

3. **Code comments**: References in source code comments explaining old behavior
   - **Action**: Remove if explaining yield protocol; keep if documenting historical decisions with clear "deprecated" marker

4. **Test fixtures**: Old yield TOML files used as test data
   - **Action**: Remove tests for yield validation; update tests to use new messaging patterns

5. **Broken internal links**: References to removed yield documentation sections
   - **Action**: Remove links; replace with links to new messaging documentation if equivalent exists

6. **Migration instructions**: Step-by-step guides showing "old way" vs "new way"
   - **Action**: Keep if in dedicated migration guide; remove if in primary workflow documentation

**Default Rule**: When uncertain whether a reference should be kept, err on the side of removal. The yield system is fully deprecated; lingering references cause confusion.

**Traces to:** REQ-001, REQ-004, ISSUE-88

---

### DES-006: Verification Strategy

**Description:** Define how to verify complete removal of yield references.

**Verification Steps:**

1. **Grep validation**: Run final case-insensitive grep for "yield" across all documented scopes
   - Expected result: Only matches in this design doc, requirements doc, and explicitly marked historical sections

2. **File cleanup verification**: Confirm deletion of active yield files
   - Check: `yield.toml` and `.claude/yield.toml` do not exist
   - Check: No new yield TOML files created in active projects

3. **Link validation**: Verify no broken links to removed yield documentation
   - Tool: Use markdown link checker or manual verification
   - Scope: All SKILL.md files, all docs/*.md files

4. **Build/test verification**: Ensure removal doesn't break tests
   - Run: `mage check` or equivalent project validation
   - Expected: All tests pass; no runtime errors from missing yield infrastructure

5. **Documentation consistency check**: All skills use consistent messaging patterns
   - Check: Search for "SendMessage" in all SKILL.md files
   - Verify: Completion reporting follows standard pattern

6. **Code reference check**: Verify no orphaned code still imports/uses yield infrastructure
   - Grep: Search source files for yield-related function names
   - Expected: No matches in active code paths

**Success Criteria:**
- Zero grep matches for yield in active workflow documentation (excluding explicit exceptions)
- All tests pass
- All documentation links functional
- All skills document messaging-based communication

**Traces to:** REQ-001, REQ-005, ISSUE-88

---

### DES-007: Current Workflow Replacement Reference

**Description:** Document the complete mapping from yield workflow to messaging workflow for implementers.

**Old Workflow (Yield-based):**
```
1. Skill receives context TOML with output.yield_path
2. Skill performs work
3. Skill writes yield TOML to output.yield_path
4. Orchestrator reads yield file
5. Orchestrator branches on yield.type
```

**New Workflow (Messaging-based):**
```
1. Teammate receives task assignment or context message
2. Teammate performs work
3. Teammate sends message to team-lead via SendMessage
4. Team lead receives message automatically
5. Team lead processes message content and continues
```

**Tool Usage Pattern:**
```markdown
## Communication Protocol

When work is complete, send a message to team-lead:

```toml
SendMessage:
  type: "message"
  recipient: "team-lead"
  summary: "PM requirements complete"
  content: "Created requirements.md with REQ-001 through REQ-005. All requirements trace to ISSUE-88."
```

When blocked, send a blocker message:

```toml
SendMessage:
  type: "message"
  recipient: "team-lead"
  summary: "Blocked on missing context"
  content: "Cannot proceed: Need clarification on X. Should I Y or Z?"
```
```

**References for Implementers:**
- See any current SKILL.md file for messaging examples (e.g., `skills/pm-interview-producer/SKILL.md`)
- Team communication protocol documented in `skills/project/SKILL.md`
- `SendMessage` tool usage in system prompts

**Traces to:** REQ-004, ISSUE-88

---

## Implementation Notes

**Execution Order:**
1. Start with grep to generate comprehensive list of files with yield references
2. Process skill documentation first (highest impact, most references)
3. Process architecture documentation second
4. Process project-specific documentation third
5. Clean up yield artifact files last (after doc updates prevent re-creation)
6. Run verification steps

**Risk Mitigation:**
- Take git snapshot before bulk deletions
- Process files in small batches with commits between batches
- Test documentation rendering after each major change
- Verify links after removing referenced sections

**Estimated Scope:**
- ~40-50 SKILL.md files with yield references
- ~10-15 architecture/design documents
- ~100+ project-specific files in .claude/projects/
- Dozens of historical yield TOML files

**Traces to:** ISSUE-88
