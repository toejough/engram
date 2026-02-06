# Architecture: Clean Up Remaining Yield References

**Issue:** ISSUE-88
**Project:** Clean up remaining yield references in docs
**Date:** 2026-02-06

## Overview

This architecture documents the technical decisions for systematically finding and removing all yield infrastructure references from documentation and code. This is a find-and-replace cleanup task with no new systems or components - the decisions focus on search patterns, file modification strategy, and verification tooling.

## Architecture Decisions

### ARCH-001: File Discovery via Multi-Pattern Grep

**Description:** Use ripgrep (via Grep tool) with multiple case-insensitive patterns to identify all files containing yield references, then filter results by file type and location to build the work list.

**Technical Approach:**
- Primary pattern: `grep -i "yield"` across all markdown and source files
- Secondary patterns for specificity:
  - `yield_path` (context configuration fields)
  - `producer_yield_path` (QA-specific configuration)
  - `yield.type` (TOML access patterns)
  - `[yield]` (TOML section headers)
- File type filters: `*.md`, `*.sh`, `*.go`, `*.toml`, `*.json`
- Output mode: `files_with_matches` to build file list, then `content` mode with context for review

**Scope Filters:**
- Include: `skills/*/SKILL.md`, `skills/*/SKILL_test.sh`, `docs/**/*.md`, `.claude/projects/**/*.md`, root `*.md`, `internal/**/*.go`, `cmd/**/*.go`
- Exclude: `.claude/projects/issue-88/requirements.md`, `.claude/projects/issue-88/design.md`, `.claude/projects/issue-88/architecture.md`, historical issue retrospectives

**Rationale:**
- Grep tool is faster and more accurate than manual file traversal
- Multiple patterns catch different usage contexts (English vs technical terms)
- Case-insensitive search prevents misses from capitalization variations
- File type filters avoid binary files and irrelevant formats

**Traces to:** DES-001, DES-005, REQ-001, REQ-002, REQ-003, ISSUE-88

---

### ARCH-002: In-Place Edit Strategy via Edit Tool

**Description:** Use Edit tool for precise string replacements in documentation files, with Read-before-Edit pattern to verify context and avoid incorrect replacements.

**Edit Pattern:**
1. Read file with Read tool to review full context
2. Identify exact `old_string` containing yield reference with sufficient surrounding context for uniqueness
3. Construct `new_string` with appropriate replacement per DES-002/DES-007 mappings
4. Apply Edit tool with exact string match
5. Re-read modified section to verify correctness

**Replacement Strategy:**
- **Skill SKILL.md files**: Replace "Yield Protocol" sections with "Communication Protocol" sections showing `SendMessage` usage
- **Architecture docs**: Replace yield file I/O diagrams with messaging flow diagrams
- **Workflow docs**: Replace yield TOML examples with `SendMessage` tool call examples
- **Ambiguous English usage** (e.g., "skill yields a result"): Rephrase to avoid confusion (e.g., "skill returns a result")

**Section Rewrite Triggers:**
- When yield content is >50% of a section, rewrite entire section rather than piecemeal edits
- When yield references are interleaved with still-valid content, extract valid content and rebuild section
- When section is purely yield documentation with no current equivalent, delete section entirely

**Rationale:**
- Edit tool prevents accidental corruption vs manual sed/awk commands
- Read-before-Edit ensures context understanding and prevents wrong replacements
- Exact string matching avoids regex complexity and edge case bugs
- Section rewrite for heavily yield-dependent content is cleaner than many small edits

**Traces to:** DES-002, DES-003, DES-005, DES-007, REQ-004, ISSUE-88

---

### ARCH-003: Verification via Grep Post-Cleanup Validation

**Description:** Use Grep tool with identical patterns from ARCH-001 to verify complete removal after edits, with explicit allowlist for legitimate remaining references.

**Verification Commands:**
```bash
# Primary verification - should return only allowlisted files
grep -i "yield" --output_mode files_with_matches --glob "*.md"

# Context verification - review remaining matches
grep -i "yield" --output_mode content -C 3 --glob "*.md"

# Code verification - ensure no orphaned code references
grep -i "yield" --output_mode files_with_matches --glob "*.go"
```

**Allowlist (Expected Matches):**
- `.claude/projects/issue-88/requirements.md` (this project's requirements)
- `.claude/projects/issue-88/design.md` (this project's design)
- `.claude/projects/issue-88/architecture.md` (this architecture doc)
- Historical retrospective files explicitly documenting the migration from yield to messaging
- Test fixtures intentionally preserving old formats for migration validation (if any exist)

**Failure Criteria:**
- ANY match outside allowlist = cleanup incomplete
- Broken markdown links detected by manual spot-check = incomplete replacement
- Files referencing removed yield sections without update = incomplete

**Rationale:**
- Same patterns as discovery ensures consistent coverage
- Allowlist makes verification deterministic (pass/fail, not subjective review)
- Context mode (-C 3) allows quick review of any unexpected matches
- Code grep catches orphaned imports/references that tests might not exercise

**Traces to:** DES-001, DES-006, REQ-001, REQ-005, ISSUE-88

---

### ARCH-004: Yield Artifact File Handling via Bash Tool

**Description:** Use Bash tool to locate and delete active yield TOML files, preserve historical yield directories in closed projects.

**File Categorization:**
1. **Active/stale root-level files** (delete): `yield.toml`, `.claude/yield.toml`
2. **Historical project yields** (preserve): `.claude/projects/ISSUE-*/yields/*.toml` where ISSUE is closed
3. **Context configuration files** (edit): Any `.toml` files with `output.yield_path` fields

**Deletion Commands:**
```bash
# Find and delete root-level yield files (if exist)
find . -maxdepth 2 -name "yield.toml" -type f -delete

# Verify deletion
ls yield.toml .claude/yield.toml 2>&1 | grep "No such file"
```

**Preservation Logic:**
- Keep `.claude/projects/ISSUE-*/yields/` directories as historical artifacts
- Add deprecation notice to project README files if not already present
- Do NOT delete historical yields - they document the evolution of the system

**Context File Updates:**
- Use Grep to find `.toml` files with `yield_path`
- Use Edit to remove `output.yield_path` lines or entire `[output]` sections if empty after removal
- Verify TOML syntax remains valid after edits

**Rationale:**
- Bash tool is appropriate for file system operations (delete, find)
- Preserving historical yields maintains project archaeology value
- Deleting active yields prevents confusion and re-introduction of deprecated patterns
- Context file updates ensure no broken references to removed infrastructure

**Traces to:** DES-004, REQ-001, REQ-003, ISSUE-88

---

### ARCH-005: Build/Test Validation via Bash Tool

**Description:** Run project build and test commands after all edits to ensure no runtime dependencies on yield infrastructure remain.

**Validation Commands:**
```bash
# Run full project validation suite
mage check

# If mage check not available, run individual checks
go test ./...
go build ./cmd/...
```

**Expected Results:**
- All tests pass (exit code 0)
- All builds succeed (exit code 0)
- No runtime errors about missing yield files or functions
- No import errors for removed yield packages

**Failure Handling:**
- Test failures indicating yield dependency = incomplete code cleanup, requires grep for orphaned code
- Build failures = incorrect edits corrupted syntax, requires review of recent edits
- Import errors = missed code references, requires broader code search

**Rationale:**
- Build/test validation catches runtime dependencies that static grep might miss
- Running after all edits (not per-file) is more efficient
- Success criteria is binary (pass/fail), not subjective
- mage check is project standard per CLAUDE.md

**Traces to:** DES-006, REQ-005, ISSUE-88

---

## Implementation Notes

**Execution Order:**
1. ARCH-001: Run grep to generate file list with yield references
2. ARCH-002: Edit files in priority order (skills → architecture → project docs)
3. ARCH-004: Delete active yield files and update context configurations
4. ARCH-003: Run grep verification to confirm complete removal
5. ARCH-005: Run build/test validation to confirm no runtime breakage

**Risk Mitigation:**
- Commit after each file type batch (all skills, all architecture docs, etc.)
- Run verification grep after each batch to catch issues early
- Defer yield file deletion until doc updates complete to prevent confusion during review

**Tools Used:**
- Grep: File discovery and post-cleanup verification
- Read: Context review before edits
- Edit: Precise string replacements
- Bash: File deletion and build/test execution

**Traces to:** ISSUE-88
