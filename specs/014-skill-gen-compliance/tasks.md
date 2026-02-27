# Tasks: Skill Generation Plugin Compliance

**Input**: Design documents from `/specs/014-skill-gen-compliance/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: TDD required per CLAUDE.md — RED/GREEN/REFACTOR cycle for all tasks.

**Organization**: Tasks grouped by user story. All code in `internal/memory/` package. Tests use `package memory_test` (blackbox), `go test -tags sqlite_fts5`, gomega assertions.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Paths relative to repository root

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Naming migration (`mem-`→`memory-`) and dual guard — MUST complete before any user story work

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests

- [x] T001 [P] Write RED tests for `migrateMemSkills()`: verify `mem-foo/` renamed to `memory-foo/`, `name: mem:foo` updated to `name: memory.foo` in SKILL.md, skip when `memory-foo/` already exists (log warning) in `internal/memory/migrate_test.go`
- [x] T002 [P] Write RED tests for `writeSkillFile()` memory- prefix: verify creates `memory-{slug}/` directory with `name: memory.{slug}` frontmatter in `internal/memory/skill_gen_test.go`
- [x] T003 Write RED tests for `isGeneratedSkill()`: verify `("memory.foo", true)` → true, `("memory.foo", false)` → false, `("custom.foo", true)` → false, `("", false)` → false in `internal/memory/optimize_test.go`

### Implementation

- [x] T004 GREEN — Implement `migrateMemSkills()` in `internal/memory/optimize.go` (pattern: existing `migrateMemoryGenSkills()`). Update all `"mem-"+slug` → `"memory-"+slug` (10 sites across `skill_gen.go`, `optimize.go`, `skills_maintenance.go`, `embeddings_maintenance.go`) and `"mem:"+slug` → `"memory."+slug` (1 site in `skill_gen.go:427`). Wire migration call into `optimizeCompileSkills()` alongside existing migration. Also update existing test assertions for `memory-`/`memory.` prefixes in `internal/memory/skill_integration_test.go` (lines ~116, ~118) and `internal/memory/migrate_test.go` (lines ~122-179) — all tests must pass (GREEN = entire suite green, not just new tests)
- [x] T005 GREEN — Implement `isGeneratedSkill(name string, generated bool) bool` in `internal/memory/optimize.go`. Wire guard checks into all existing skill modification paths: `optimizeCompileSkills()` merge, `performSkillReorganization()` update, `optimizeSplitSkills()` delete, `optimizeMergeSkills()`, and `skills_maintenance.go` pruning/merging

**Checkpoint**: All `mem-*` references replaced with `memory-*`. Dual guard protects non-generated skills. Existing tests updated for new prefixes.

---

## Phase 2: User Story 1 — Generated Skills Trigger Correctly (Priority: P1) 🎯 MVP

**Goal**: Generated SKILL.md files have `description` fields that start with "Use when" followed by triggering conditions, under 1024 chars, in third person — enabling Claude Code to correctly discover and invoke skills.

**Independent Test**: Unit tests verify `generateTriggerDescription()` output starts with "Use when", is ≤1024 chars, uses third person, and is not a substring of the content body. Unit tests verify `writeSkillFile()` caps description at 1024 chars and YAML-quotes special characters.

### Tests

- [x] T006 [US1] Write RED tests for `generateTriggerDescription(theme, content)`: verify output starts with "Use when", ≤1024 chars (Go string length), third person (no "I"/"you"/"we" subjects), not a substring of content, handles long themes gracefully in `internal/memory/skill_gen_test.go`
- [x] T007 [US1] Write RED tests for `writeSkillFile()` description cap: verify 2000-char description truncated to ≤1024 chars on disk, description with colons/quotes/newlines is YAML-quoted in frontmatter in `internal/memory/skill_gen_test.go`

### Implementation

- [x] T008 [US1] GREEN — Implement `generateTriggerDescription(theme string, content string) string` in `internal/memory/skill_gen.go`. Template: `"Use when the user encounters {theme}-related patterns or needs guidance on {theme}."` Cap at 1024 chars. Third person only. Must not truncate content
- [x] T009 [US1] GREEN — Add defensive 1024-char description cap in `writeSkillFile()` in `internal/memory/skill_gen.go`. YAML-quote description when it contains special characters (colons, quotes, newlines)

**Checkpoint**: `generateTriggerDescription()` and `writeSkillFile()` cap are unit-tested. Not yet wired into pipeline (Phase 5).

---

## Phase 3: User Story 2 — Body Structure Matches Expected Format (Priority: P2)

**Goal**: Template fallback produces Overview / When to Use / Quick Reference / Common Mistakes sections. LLM prompt requests JSON with separate `description` and `body` fields.

**Independent Test**: Unit tests verify template output contains all 4 section headers in order. Unit tests verify LLM JSON parsing extracts description and body separately, with fallback on parse failure.

### Tests

- [x] T010 [P] [US2] Write RED tests for template 4-section body structure: verify `generateSkillContent()` and `generateSkillTemplate()` output contains `## Overview`, `## When to Use`, `## Quick Reference`, `## Common Mistakes` in order. Test insufficient content (1-entry cluster) returns error (FR-011) in `internal/memory/skill_gen_test.go` and `internal/memory/optimize_test.go`
- [x] T011 [P] [US2] Write RED tests for LLM CompileSkill JSON parsing: verify valid JSON `{"description":"Use when...","body":"## Overview\n..."}` → separate fields extracted. Verify invalid JSON (plain markdown) → falls through to template. Verify valid JSON with empty/null description → fallback to `generateTriggerDescription()` in `internal/memory/llm_api_test.go`

### Implementation

- [x] T012 [US2] GREEN — Update `generateSkillContent()` in `internal/memory/skill_gen.go` and `generateSkillTemplate()` in `internal/memory/optimize.go` to produce 4-section structure: `## Overview` (intro from memories), `## When to Use` (trigger conditions from theme), `## Quick Reference` (numbered patterns from cluster entries), `## Common Mistakes` (anti-patterns). Return error when cluster content is insufficient to populate all sections (FR-011). Also implement `parseCompileSkillJSON(output string) (description, body string, err error)` helper in `internal/memory/optimize.go` for LLM JSON response parsing — this is the GREEN target for T011's RED tests
- [x] T013 [P] [US2] GREEN — Update `ClaudeCLIExtractor.CompileSkill()` prompt in `internal/memory/llm.go` to request JSON output: `{"description": "Use when...", "body": "## Overview\n..."}`
- [x] T014 [P] [US2] GREEN — Update `DirectAPIExtractor.CompileSkill()` prompt in `internal/memory/llm_api.go` to request JSON output (same format as T013)

**Checkpoint**: Templates produce correct 4-section structure. LLM prompts request structured JSON. JSON parsing tested. Not yet wired into pipeline (Phase 5).

---

## Phase 4: User Story 3 — Validation Catches Non-Compliant Skills (Priority: P3)

**Goal**: `ValidateSkillCompliance()` checks all 8 rules (V1-V8) and blocks non-compliant skills from being written to disk. Results surfaced via `OptimizeResult`.

**Independent Test**: Unit tests verify each V1-V8 check individually (pass + fail). Unit tests verify `SkillComplianceResult` aggregates issues correctly.

### Tests

- [x] T015 [US3] Write RED tests for `ValidateSkillCompliance()` checking V1-V8: V1 (starts with "Use when"), V2 (≤1024 chars), V3 (not truncated from content), V4 (third person), V5 (required sections present), V6 (≤500 lines), V7 (`memory.` prefix), V8 (`generated: true`). Test each individually with pass and fail cases in `internal/memory/optimize_test.go`

### Implementation

- [x] T016 [US3] GREEN — Implement `SkillComplianceResult` struct (Slug, DescriptionOK, BodyStructureOK, BodyLengthOK, NamingOK, Issues []string) and `ValidateSkillCompliance(*GeneratedSkill) SkillComplianceResult` in `internal/memory/optimize.go`
- [x] T017 [US3] GREEN — Add `SkillsBlocked int`, `SkillsSkipped int`, and `ValidationIssues []SkillComplianceResult` fields to `OptimizeResult` in `internal/memory/optimize.go`

**Checkpoint**: Validation function tested for all 8 rules. OptimizeResult has reporting fields. Not yet wired into pipeline (Phase 5).

---

## Phase 5: Integration & Wiring

**Purpose**: Wire all new functions into the pipeline's 5 code paths. This is where generated skills on disk actually become compliant.

### Tests

- [x] T018 Write RED integration test: run optimize with mock compiler, verify ALL created skills have compliant descriptions (start with "Use when", ≤1024 chars), compliant body structure (4 sections), and pass `ValidateSkillCompliance()` in `internal/memory/optimize_test.go`

### Implementation

- [x] T019 [P] Replace `ExtractSkillDescription(content, 1500)` calls with `generateTriggerDescription(theme, content)` at 3 sites in `internal/memory/optimize.go`: `optimizeCompileSkills` (line ~1496), `performSkillReorganization` (line ~1656), `optimizeSplitSkills` (line ~2675)
- [x] T020 [P] Replace `ExtractSkillDescription(content, 1500)` call with `generateTriggerDescription(theme, content)` in `PromoteEmbeddingToSkill()` in `internal/memory/embeddings_maintenance.go` (line ~448)
- [x] T021 Update `optimizeDemoteClaudeMD` description path (`description := theme` at line ~2209) to use `generateTriggerDescription(theme, content)` in `internal/memory/optimize.go`
- [x] T022 Wire LLM JSON parsing into `optimizeCompileSkills()` in `internal/memory/optimize.go`: parse CompileSkill return as JSON, extract `description` and `body` fields separately. On parse failure, use body as content and `generateTriggerDescription()` for description. On empty/null description, use `generateTriggerDescription()` fallback
- [x] T023 Wire `ValidateSkillCompliance()` blocking before `writeSkillFile()` and DB insert/update in all creation/update paths in `internal/memory/optimize.go`. On validation failure: skip write, increment `OptimizeResult.SkillsBlocked`, append to `OptimizeResult.ValidationIssues`
- [x] T024 Wire `SkillsSkipped` reporting: when `generateSkillContent()` or `generateSkillTemplate()` returns insufficient-content error (FR-011), increment `OptimizeResult.SkillsSkipped` and log skip reason in `internal/memory/optimize.go`

**Checkpoint**: All 5 code paths produce compliant skills. Validation blocks non-compliant output. Integration tests pass.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Cleanup, assertion updates, and final validation

- [x] T025 Remove or deprecate `ExtractSkillDescription()` from `internal/memory/optimize.go` if no remaining callers
- [x] T026 Verify no remaining `mem-`/`mem:` string literals in source or test files via grep across `internal/memory/` — confirm T004 prefix migration is complete
- [x] T027 Run full test suite validation: `go test -tags sqlite_fts5 ./internal/memory/...` — all tests must pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — start immediately. BLOCKS all user stories.
- **US1 (Phase 2)**: Depends on Phase 1 completion (needs `memory.` prefix in place)
- **US2 (Phase 3)**: Depends on Phase 1 completion (needs `memory.` prefix in place)
- **US3 (Phase 4)**: Depends on Phase 1 completion. Also benefits from Phase 2 (validation rules align with description rules)
- **Integration (Phase 5)**: Depends on Phases 1-4 ALL complete (wires everything together)
- **Polish (Phase 6)**: Depends on Phase 5 completion

### User Story Dependencies

- **US1 (P1)**: Independent after Phase 1
- **US2 (P2)**: Independent after Phase 1
- **US3 (P3)**: Independent after Phase 1. Uses same rule definitions as US1 (FR-001-FR-004)
- **US1, US2, US3 can run in parallel** after Phase 1 completes

### Within Each Phase

- RED tests MUST be written and FAIL before GREEN implementation
- Within a phase, tasks without [P] marker are sequential
- Tasks with [P] marker can run in parallel with other [P] tasks in the same phase

### Parallel Opportunities

**Phase 1**: T001 ‖ T002 ‖ T003 → T004 → T005 (tests parallel, impl sequential)

**Phases 2-4 (after Phase 1)**: All three user stories can proceed in parallel:
```
Phase 2 (US1): T006 → T007 → T008 → T009
Phase 3 (US2): T010 ‖ T011 → T012 → T013 ‖ T014
Phase 4 (US3): T015 → T016 → T017
```

**Phase 5**: T018 → T019 ‖ T020 → T021 → T022 → T023 → T024

---

## Parallel Example: Phases 2-4 Concurrent

```text
# After Phase 1 completes, launch all three in parallel:

Agent A (US1): T006 → T007 → T008 → T009
  Files: internal/memory/skill_gen.go, internal/memory/skill_gen_test.go

Agent B (US2): T010 ‖ T011 → T012 → T013 ‖ T014
  Files: internal/memory/skill_gen.go, internal/memory/optimize.go,
         internal/memory/llm.go, internal/memory/llm_api.go,
         internal/memory/optimize_test.go, internal/memory/llm_api_test.go

Agent C (US3): T015 → T016 → T017
  Files: internal/memory/optimize.go, internal/memory/optimize_test.go
```

**⚠️ File conflict**: Agents A+B both touch `skill_gen.go`. Agents B+C both touch `optimize.go`. If running in parallel, coordinate merges. Alternatively, run US1 first (MVP), then US2+US3 in parallel.

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Foundational (naming + dual guard)
2. Complete Phase 2: US1 (description triggering)
3. Complete Phase 5 tasks T018-T021 (description wiring only)
4. **STOP and VALIDATE**: Skills on disk have compliant descriptions
5. This alone fixes the critical gap — Claude can discover and invoke skills correctly

### Incremental Delivery

1. Phase 1 → Foundation ready
2. Phase 2 + partial Phase 5 → US1 done (MVP — skills trigger correctly)
3. Phase 3 + partial Phase 5 → US2 done (body structure correct)
4. Phase 4 + remaining Phase 5 → US3 done (validation safety net)
5. Phase 6 → Polish complete

### Full Sequential

1. Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6

---

## Traceability

| Task | Plan Task | FRs | User Story |
|------|-----------|-----|------------|
| T001-T002, T004 | Task 1 | FR-012 | Foundation |
| T003, T005 | Task 2 | FR-013 | Foundation |
| T006, T008 | Task 3 | FR-001, FR-002, FR-003, FR-004, FR-010 | US1 |
| T007, T009 | Task 8 | FR-002 | US1 |
| T010, T012 | Task 4+5 | FR-005, FR-006, FR-011 | US2 |
| T011, T013-T014 | Task 5 | FR-006 | US2 |
| T015-T017 | Task 7 | FR-007, FR-008, FR-009, FR-014 | US3 |
| T018-T024 | Task 6 | FR-001, FR-009, FR-010, FR-011, FR-014 | Integration |
| T025-T027 | — | — | Polish |

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks in same phase
- All code in `internal/memory/` package — no new packages
- Blackbox tests use `package memory_test`
- Run tests: `go test -tags sqlite_fts5 ./internal/memory/...`
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
