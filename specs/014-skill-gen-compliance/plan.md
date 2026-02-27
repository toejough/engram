# Implementation Plan: Skill Generation Plugin Compliance

**Branch**: `014-skill-gen-compliance` | **Date**: 2026-02-19 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-skill-gen-compliance/spec.md`

## Summary

Generated skills from projctl's memory compilation pipeline don't conform to Claude Code plugin conventions. The description field dumps content instead of providing "Use when..." trigger descriptions, template fallback produces flat content instead of structured sections, naming uses `mem:` instead of `memory.`, and there's no validation. This plan fixes all issues across 5 code paths in `internal/memory/`, adds blocking validation, implements dual-guard protection for user-edited skills, and migrates existing `mem-*` directories.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: database/sql (SQLite via go-sqlite3), encoding/json, strings, gomega (testing)
**Storage**: SQLite (embeddings.db) — no schema changes needed
**Testing**: `go test -tags sqlite_fts5` with gomega assertions, blackbox test pattern (`package memory_test`)
**Target Platform**: CLI tool (macOS/Linux)
**Project Type**: Single Go module
**Performance Goals**: N/A — batch optimization pipeline, not latency-sensitive
**Constraints**: LLM may be unavailable (template fallback must always work); blocking validation means no partial writes
**Scale/Scope**: 5 code paths, 2 templates, 2 LLM prompts, ~10 `mem-`/`mem:` reference sites, 1 migration function, 1 validation function, 1 dual-guard helper

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No project-specific constitution defined (constitution.md is a template). Standard engineering principles apply:
- TDD: Red/Green/Refactor cycle per CLAUDE.md
- Blackbox tests: `package memory_test`
- No breaking interface changes without justification

**Gate status**: PASS (no violations)
**Post-design re-check**: PASS — SkillCompiler interface unchanged, all changes internal

## Project Structure

### Documentation (this feature)

```text
specs/014-skill-gen-compliance/
├── plan.md              # This file
├── spec.md              # Feature specification (with clarifications)
├── research.md          # Phase 0 research findings (R1-R8)
├── data-model.md        # Entity changes, validation rules, dual guard model
├── quickstart.md        # Implementation guide
├── contracts/
│   └── skill-compliance.md  # Function contracts
├── checklists/
│   └── requirements.md      # Spec quality checklist
└── tasks.md                 # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/memory/
├── skill_gen.go                  # writeSkillFile(), generateTriggerDescription()
├── optimize.go                   # ValidateSkillCompliance(), isGeneratedSkill(), migrateMemSkills(),
│                                 #   template updates, call site replacements, blocking wire-up
├── llm.go                        # ClaudeCLIExtractor.CompileSkill() prompt update
├── llm_api.go                    # DirectAPIExtractor.CompileSkill() prompt update
├── embeddings_maintenance.go     # PromoteEmbeddingToSkill() updates
├── skills_maintenance.go         # Dual guard checks
├── optimize_test.go              # Validation, description, migration tests
├── skill_gen_test.go             # writeSkillFile tests, trigger description tests
├── skill_integration_test.go     # Updated prefix assertions
├── migrate_test.go               # Migration tests
└── llm_api_test.go               # CompileSkill JSON parsing tests
```

**Structure Decision**: All changes within existing `internal/memory/` package. No new packages.

## Implementation Tasks

### Task 1: Rename `mem-`/`mem:` to `memory-`/`memory.` and add migration (FR-012)

**Files**: `skill_gen.go`, `optimize.go`, `skills_maintenance.go`, `embeddings_maintenance.go`

Change all `"mem-"+slug` → `"memory-"+slug` (10 sites) and `"mem:"+slug` → `"memory."+slug` (1 site in `writeSkillFile`).

Add `migrateMemSkills()` function (same pattern as existing `migrateMemoryGenSkills()`) to rename `mem-*` directories to `memory-*` and update frontmatter `name:` in each SKILL.md. Wire into `optimizeCompileSkills()` alongside existing migration call.

Update test assertions in `skill_integration_test.go` and `migrate_test.go`.

**TDD approach**:
- RED: Test `migrateMemSkills()` renames `mem-foo/` to `memory-foo/` and updates `name: mem:foo` to `name: memory.foo` in SKILL.md.
- RED: Test `writeSkillFile()` creates `memory-{slug}/` directory with `name: memory.{slug}` frontmatter.
- GREEN: Implement migration function and update all prefix references.
- REFACTOR: Extract prefix constants if repeated string appears more than 3 times.

### Task 2: Add `isGeneratedSkill()` dual guard and wire into pipeline (FR-013)

**Files**: `optimize.go`, `skills_maintenance.go`

Create `isGeneratedSkill(name string, generated bool) bool` that returns true only when `generated == true` AND `strings.HasPrefix(name, "memory.")`.

Wire into every code path that modifies existing skills:
- `optimizeCompileSkills()` merge path
- `performSkillReorganization()` update path
- `optimizeSplitSkills()` delete path
- `optimizeMergeSkills()`
- `skills_maintenance.go` pruning/merging

**TDD approach**:
- RED: Test `isGeneratedSkill("memory.foo", true)` → true. Test all false cases: missing prefix, generated=false, both missing.
- RED: Integration test: create skill without `memory.` prefix, run optimize, verify it's untouched.
- GREEN: Implement helper and add guards at all sites.

### Task 3: Add `generateTriggerDescription()` function (FR-001, FR-002, FR-003, FR-004, FR-010)

**File**: `skill_gen.go` (or `optimize.go`)

Create function that generates a "Use when..." description from theme and content:
- Template path: `"Use when the user encounters {theme}-related patterns or needs guidance on {theme}."` Capped at 1024 chars. Third person.
- Must NOT be derived by truncating content.

**TDD approach**:
- RED: Test starts with "Use when", <= 1024 chars, third person, not a substring of content.
- GREEN: Implement.
- REFACTOR: Clean up.

### Task 4: Update template fallback body structure (FR-005, FR-011)

**Files**: `skill_gen.go` (`generateSkillContent` fallback), `optimize.go` (`generateSkillTemplate`)

Change both templates to produce 4-section structure:
```markdown
# {theme}

## Overview
{intro derived from memories/learning}

## When to Use
{trigger conditions from theme}

## Quick Reference
{numbered patterns from cluster entries}

## Common Mistakes
{anti-patterns or caveats}
```

If the input content is insufficient to meaningfully populate all sections, return an error (FR-011). Callers skip skill creation when this happens.

**TDD approach**:
- RED: Test template output contains all 4 headers in order.
- RED: Test with 1-entry cluster returns error (insufficient content).
- GREEN: Update both template functions.

### Task 5: Update LLM CompileSkill prompt for structured output (FR-006)

**Files**: `llm.go`, `llm_api.go`

Update prompt in both `CompileSkill` implementations to request JSON:
```json
{"description": "Use when...", "body": "## Overview\n..."}
```

Parse JSON in callers. On parse failure, fall through to template fallback.

**TDD approach**:
- RED: Test mock returning valid JSON → description and body extracted separately.
- RED: Test mock returning invalid JSON → falls through to template.
- GREEN: Update prompts and add JSON parsing.

### Task 6: Replace all `ExtractSkillDescription` call sites with `generateTriggerDescription` (FR-001, FR-010)

**Files**: `optimize.go` (3 sites), `embeddings_maintenance.go` (1 site)

Replace all 4 `ExtractSkillDescription(content, 1500)` calls with `generateTriggerDescription(theme, content)`. Update the `description := theme` path in `optimizeDemoteClaudeMD` (optimize.go:2209) similarly.

When LLM path returns valid JSON, use the parsed `description` field instead.

**TDD approach**:
- RED: Integration test: run optimize with mock compiler, verify all created skills have compliant descriptions.
- GREEN: Replace each call site.
- REFACTOR: Remove or deprecate `ExtractSkillDescription` if unused.

### Task 7: Add `ValidateSkillCompliance()` with blocking behavior (FR-007, FR-008, FR-009)

**File**: `optimize.go` (or new `skill_validation.go`)

Create `ValidateSkillCompliance(*GeneratedSkill) SkillComplianceResult` checking V1-V8 from data-model.md.

Wire into all skill creation/update paths **before** `writeSkillFile()` and DB insert/update. If validation fails:
- Skip the write entirely
- Increment `OptimizeResult.SkillsBlocked`
- Append violation details to `OptimizeResult.ValidationIssues`

Add `SkillsBlocked int` and `ValidationIssues []SkillComplianceResult` to `OptimizeResult`.

**TDD approach**:
- RED: Test each V1-V8 check individually (pass + fail).
- RED: Integration test: skill with bad description → not written to disk, blocked count incremented.
- GREEN: Implement validation and wire into pipeline.

### Task 8: Update `writeSkillFile()` description cap (FR-002)

**File**: `skill_gen.go`

Defensive cap: truncate description at 1024 chars in `writeSkillFile()` even though upstream should already enforce this. Ensure YAML-safe output (quote description if special chars present).

**TDD approach**:
- RED: Test writing skill with 2000-char description → file has description <= 1024 chars.
- GREEN: Add truncation.

## Task Dependencies

```
Task 1 (naming migration)              ──→  Task 2 (dual guard uses new prefix)
Task 1                                  ──→  Task 6 (call sites use new prefix)
Task 2 (dual guard)                     ──→  Task 6 (call sites check guard before updating)
Task 3 (generateTriggerDescription)     ──→  Task 6 (replace call sites)
Task 4 (template body structure)        ──→  Task 6 (templates used by call sites)
Task 5 (LLM prompt update)             ──→  Task 6 (LLM output used by call sites)
Task 3                                  ──→  Task 7 (validation uses same rules)
Task 3                                  ──→  Task 8 (writeSkillFile uses description)
Task 7 (validation)                     ──→  Task 6 (call sites wire in validation)
```

**Recommended execution order**: Task 1 → Task 2 → Task 3 → Task 4 → Task 5 → Task 8 → Task 7 → Task 6

**Parallelizable groups**:
- Group A (independent): Tasks 3, 4, 5 (description gen, template structure, LLM prompt)
- Group B (depends on 1): Task 2 (dual guard)
- Group C (depends on A+B): Tasks 6, 7, 8 (wiring, validation, writeSkillFile cap)

## Complexity Tracking

No constitution violations. All changes within existing package using existing patterns. The migration function follows the established `migrateMemoryGenSkills()` pattern.
