# Contract: Skill Compliance Validation

This feature operates entirely within the Go memory package. There are no REST APIs or external contracts. The "contracts" are the function signatures and behavioral guarantees.

## Function Contracts

### generateTriggerDescription(theme string, content string) string

Generates a "Use when..." description from a skill's theme and content body.

- **Input**: theme (skill topic), content (markdown body)
- **Output**: String starting with "Use when", max 1024 chars, third person
- **Invariant**: Output must not be a substring of `content`
- **Template path**: Constructs deterministically from theme string
- **LLM path**: Parsed from JSON output of CompileSkill (see below)

### ValidateSkillCompliance(skill *GeneratedSkill) SkillComplianceResult

Validates a generated skill against all plugin criteria. **Blocking** — non-compliant skills must not be written to disk.

- **Input**: GeneratedSkill with Description and Content populated
- **Output**: SkillComplianceResult with pass/fail per check and issue list
- **Checks**: V1-V8 from data-model.md (description format, length, body structure, body length, naming, generated flag)
- **Behavior**: Called before `writeSkillFile()` and before DB insert/update. If any check fails, caller skips the write and appends violations to OptimizeResult.

### isGeneratedSkill(name string, generated bool) bool

Dual guard check for whether a skill is managed by the optimization pipeline.

- **Input**: skill frontmatter `name` and `generated` flag
- **Output**: true only if `generated == true` AND `name` starts with `memory.`
- **Usage**: Called before any update/delete/regeneration of an existing skill. If false, the skill is not touched.

### Updated template fallback (generateSkillContent, generateSkillTemplate)

Template output must contain these sections in order:
1. `## Overview`
2. `## When to Use`
3. `## Quick Reference`
4. `## Common Mistakes`

If the memory cluster or learning content is insufficient to meaningfully populate all four sections (FR-011), the function returns an error and the caller skips skill creation entirely.

### Updated LLM prompt (CompileSkill)

LLM prompt instructs the model to produce JSON output:
```json
{
  "description": "Use when...",
  "body": "## Overview\n..."
}
```

The `CompileSkill` method signature stays unchanged: `(ctx, theme, memories) → (string, error)`. The returned string is now JSON. Callers parse it. On parse failure, fall through to template fallback (same as current LLM-error behavior).

### writeSkillFile (updated)

- Directory: `memory-{slug}/` (was `mem-{slug}/`)
- Frontmatter `name`: `memory.{slug}` (was `mem:{slug}`)
- Frontmatter `description`: capped at 1024 chars (was 1500)
- Frontmatter `generated: true` (unchanged)

### migrateMemSkills (new — one-time migration)

Renames existing `mem-*` directories to `memory-*` and updates frontmatter `name:` from `mem:` to `memory.` in each SKILL.md. Same pattern as existing `migrateMemoryGenSkills()`.

## Interface Changes

### SkillCompiler (no breaking change)

The `CompileSkill` method signature stays `(ctx, theme, memories) → (string, error)`. The LLM now returns JSON. The caller parses it. Parse failures fall through to template fallback.

### OptimizeResult (extended)

New field: `SkillsBlocked int` — count of skills that failed validation and were not written.
New field: `ValidationIssues []SkillComplianceResult` — detailed per-skill violations.
