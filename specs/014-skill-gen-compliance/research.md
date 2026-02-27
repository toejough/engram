# Research: Skill Generation Plugin Compliance

## R1: How is the `description` field currently populated?

**Decision**: The description is derived from content body via `ExtractSkillDescription()`, not generated independently.

**Findings**:
- `ExtractSkillDescription(content, 1500)` is called in 4 locations: `optimizeCompileSkills` (line 1496), `performSkillReorganization` (line 1656), `optimizeSplitSkills` (line 2675), and `embeddings_maintenance.go` (line 448).
- One additional path in `optimizeDemoteClaudeMD` (line 2209) uses `description := theme` truncated to 200 chars.
- The function either extracts structured markers (Core:/Triggers:/Domains:/Anti-patterns:/Related:) or concatenates non-empty non-header lines up to `maxLen` (1500 chars).
- The 1500-char limit exceeds the plugin's 1024-char max.
- The content is never in "Use when..." trigger format.

**Alternatives considered**: None — factual finding.

## R2: What does the LLM CompileSkill prompt produce?

**Decision**: The LLM prompt asks for general skill content with no instruction to produce a trigger description separately.

**Findings**:
- Both `ClaudeCLIExtractor.CompileSkill()` (llm.go:207) and `DirectAPIExtractor.CompileSkill()` (llm_api.go:407) use identical prompt structure.
- Prompt says: "Create a comprehensive SKILL.md content that: Explains the concept clearly, Provides actionable guidance, Includes specific examples, Uses markdown formatting."
- Returns a single string — no structured output separating description from body.
- The description is then extracted by `ExtractSkillDescription()` from this combined output.

**Alternatives considered**: Return JSON with separate `description` and `body` fields — this is the chosen approach.

## R3: What does the template fallback produce?

**Decision**: Template fallback produces a flat "Related Memories" list, not the expected 4-section structure.

**Findings**:
- `generateSkillContent()` fallback (skill_gen.go:316-327): `# {theme}` → intro line → `## Related Memories` → numbered list → `## Application` → generic line.
- `generateSkillTemplate()` (optimize.go:2252-2261): `# {theme}` → intro line → `## Context` → learning → `## Application` → generic line.
- Neither template matches the expected sections: Overview, When to Use, Quick Reference, Common Mistakes.

**Alternatives considered**: None — factual finding.

## R4: What are all the code paths that set `Description` on a GeneratedSkill?

**Decision**: There are 5 distinct code paths that need to be updated.

**Findings**:
1. `optimizeCompileSkills()` at optimize.go:1496 — main cluster compilation
2. `performSkillReorganization()` at optimize.go:1656-1667 — periodic reorg (both new + update paths)
3. `optimizeSplitSkills()` at optimize.go:2675 — skill splitting
4. `optimizeDemoteClaudeMD()` at optimize.go:2209 — CLAUDE.md demotion to skill (uses `theme` directly, not `ExtractSkillDescription`)
5. `PromoteEmbeddingToSkill()` at embeddings_maintenance.go:448 — single embedding promotion

All 5 paths must generate compliant descriptions.

## R5: What is the SkillCompiler interface contract?

**Decision**: `SkillCompiler` has 2 methods: `CompileSkill(ctx, theme, memories) (string, error)` and `Synthesize(ctx, memories) (string, error)`.

**Findings**:
- Defined in llm.go:72-75.
- `CompileSkill` returns a single string. It would need to return a description too.
- Two implementations: `ClaudeCLIExtractor` (llm.go:207) and `DirectAPIExtractor` (llm_api.go:407).
- Both use identical prompt structure.
- Tests mock via `mockSkillCompiler` in test_helpers_integration_test.go.

**Chosen approach**: Keep `CompileSkill` signature unchanged. Update the LLM prompt to return JSON with `description` + `body` fields. Caller parses JSON; on parse failure, falls through to template fallback. For template path, `generateTriggerDescription()` creates the description deterministically from theme.

## R6: How are generated skills tested before insertion?

**Decision**: `TestAndCompileSkill()` validates skill candidates, but has no compliance checks for description format.

**Findings**:
- `TestAndCompileSkill()` at optimize.go:166 tests skill candidates via the test harness.
- No validation of description format, length, or body structure currently exists.
- This is the natural place to add blocking validation per clarification (non-compliant skills must not be written to disk).

## R7: Where are `mem-` and `mem:` prefixes used? (new — from clarification)

**Decision**: All naming must change from `mem-`/`mem:` to `memory-`/`memory.`.

**Findings — `mem-` directory prefix (10 sites)**:
- `skill_gen.go:416` — `writeSkillFile()`: `"mem-"+skill.Slug`
- `skills_maintenance.go:435,543` — merge/split maintenance
- `optimize.go:1339` — legacy migration target path
- `optimize.go:1746` — merge skill cleanup
- `optimize.go:1933` — prune skill cleanup
- `optimize.go:2569` — merge delete
- `optimize.go:2705` — split delete

**Findings — `mem:` frontmatter name prefix (1 site)**:
- `skill_gen.go:427` — `writeSkillFile()`: `"name: mem:%s\n"`

**Findings — `generated: true` frontmatter (1 site)**:
- `skill_gen.go:431` — `writeSkillFile()`

**Test references**:
- `skill_integration_test.go:116,118` — asserts `"mem:"` and `"generated: true"`
- `migrate_test.go:122-179` — legacy migration from `memory-gen/` to `mem-`

**Migration needed**: Existing `mem-*` directories on disk will need a migration function (similar to existing `migrateMemoryGenSkills`) to rename to `memory-*`. Existing DB records store slugs without prefix, so no DB schema change needed.

## R8: Dual guard — where must the `generated: true` + `memory.` name check be enforced? (new)

**Decision**: Every code path that reads or modifies an existing skill file must check both markers before proceeding.

**Findings**:
- `performSkillReorganization()` updates existing skills by slug lookup — must verify markers
- `optimizeCompileSkills()` merges into existing skills — must verify markers
- `optimizeSplitSkills()` soft-deletes and rewrites — must verify markers
- `optimizeMergeSkills()` — must verify markers
- Skills maintenance (`skills_maintenance.go`) — pruning, merging — must verify markers

A helper function `isGeneratedSkill(skill) bool` checking both `generated: true` in DB and `memory.` name prefix provides the dual guard at all call sites.
