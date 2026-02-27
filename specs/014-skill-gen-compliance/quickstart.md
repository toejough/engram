# Quickstart: Skill Generation Plugin Compliance

## What Changes

The skill generation pipeline in `internal/memory/` is updated so generated SKILL.md files conform to Claude Code plugin conventions:

1. **Naming**: `mem:` prefix → `memory.` prefix; `mem-` directories → `memory-` directories.
2. **Description field**: Generated as a "Use when..." trigger description (max 1024 chars), not by truncating content.
3. **Body structure**: Template fallback produces Overview / When to Use / Quick Reference / Common Mistakes sections. LLM prompt requests the same.
4. **Blocking validation**: New `ValidateSkillCompliance()` function blocks non-compliant skills from being written to disk.
5. **Dual guard**: Only skills with both `generated: true` and `memory.` name prefix are modified by the pipeline.
6. **Insufficient clusters**: Clusters that can't meaningfully populate all four sections are skipped entirely.
7. **Migration**: Existing `mem-*` skills renamed to `memory-*` with updated frontmatter.

## Files Modified

| File | Change |
|------|--------|
| `internal/memory/skill_gen.go` | `writeSkillFile()`: `memory-` dir, `memory.` name, 1024-char description cap. New `generateTriggerDescription()`. |
| `internal/memory/optimize.go` | Replace 5 `ExtractSkillDescription` calls. Update templates. Add `ValidateSkillCompliance()`, `isGeneratedSkill()`, `migrateMemSkills()`. Wire blocking validation. Add `SkillsBlocked` to OptimizeResult. |
| `internal/memory/llm.go` | `ClaudeCLIExtractor.CompileSkill()`: prompt → JSON output with description + body |
| `internal/memory/llm_api.go` | `DirectAPIExtractor.CompileSkill()`: prompt → JSON output with description + body |
| `internal/memory/embeddings_maintenance.go` | `PromoteEmbeddingToSkill()`: use `generateTriggerDescription()`, `memory-` prefix |
| `internal/memory/skills_maintenance.go` | Add dual guard checks before modifying skills |

## How to Test

```bash
# Run all memory package tests
go test -tags sqlite_fts5 ./internal/memory/...

# Run specific compliance tests
go test -tags sqlite_fts5 -run TestValidateSkillCompliance ./internal/memory/...
go test -tags sqlite_fts5 -run TestGenerateTriggerDescription ./internal/memory/...
go test -tags sqlite_fts5 -run TestMigrateMemSkills ./internal/memory/...
go test -tags sqlite_fts5 -run TestIsGeneratedSkill ./internal/memory/...
```

## Key Design Decisions

- **No interface change**: `SkillCompiler.CompileSkill()` signature stays `(ctx, theme, memories) → (string, error)`. LLM returns JSON; caller parses. Parse failures fall through to template.
- **Blocking validation**: Non-compliant skills are never written to disk. Violations reported in OptimizeResult.
- **Forward-only compliance**: No special retroactive migration for content/description. Existing skills become compliant during normal reorg cycles.
- **Dual guard**: Both `generated: true` AND `memory.` name prefix required before pipeline touches a skill.
- **Skip insufficient clusters**: If content can't meaningfully fill all four body sections, no skill is created. Better no skill than a hollow one.
- **Migration for naming only**: `mem-*` → `memory-*` rename is a one-time migration; content compliance is forward-only via reorg.
