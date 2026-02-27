# Data Model: Skill Generation Plugin Compliance

## Entities

### GeneratedSkill (existing — field semantics change)

| Field       | Type   | Change | Notes |
|-------------|--------|--------|-------|
| Slug        | string | No change | URL-safe slug (e.g., `error-handling`). DB stores slug without prefix. |
| Description | string | **Semantic change** | Must now contain "Use when..." trigger description, max 1024 chars. Previously held truncated content up to 1500 chars. |
| Content     | string | **Structural change** | Must now follow 4-section format (Overview, When to Use, Quick Reference, Common Mistakes). |
| All others  | -      | No change | Theme, Alpha, Beta, Utility, etc. unchanged. |

No schema/table changes required. The `description` and `content` columns in `generated_skills` table stay `TEXT`. Changes are purely in what gets stored.

### Naming Convention (changed)

| Element | Old | New |
|---------|-----|-----|
| Frontmatter `name` | `mem:{slug}` | `memory.{slug}` |
| Directory name | `mem-{slug}/` | `memory-{slug}/` |
| `generated` flag | `generated: true` | `generated: true` (unchanged) |

### SkillComplianceResult (new — validation output)

| Field           | Type     | Description |
|-----------------|----------|-------------|
| Slug            | string   | Skill identifier |
| DescriptionOK   | bool     | Description passes all checks (V1-V4) |
| BodyStructureOK | bool     | Body has required sections (V5) |
| BodyLengthOK    | bool     | Body under 500 lines (V6) |
| NamingOK        | bool     | Name has `memory.` prefix and `generated: true` (V7+V8) |
| Issues          | []string | Specific violation messages |

Runtime struct for validation reporting, not persisted.

## Dual Guard Model

A skill is considered "managed by the optimization pipeline" if and only if:
1. Its frontmatter contains `generated: true`
2. Its frontmatter `name` starts with `memory.`

Both conditions must be true. If either is missing, the pipeline MUST NOT modify the skill.

```
isGeneratedSkill(skill) = skill.Generated == true AND strings.HasPrefix(skill.Name, "memory.")
```

## State Transitions

Skills flow through this lifecycle:

```
Memory Cluster
  → [FR-011: sufficient content check — skip if too thin]
  → generateSkillContent() — produces 4-section body
  → generateTriggerDescription() — produces "Use when..." description
  → ValidateSkillCompliance() — blocking check
  → [PASS] → writeSkillFile() with memory. prefix + generated: true
  → [FAIL] → skip write, report violations in OptimizeResult
```

For existing skill updates, the dual guard is checked first:
```
Existing skill on disk
  → isGeneratedSkill() check — skip if either marker missing
  → regenerate content + description
  → ValidateSkillCompliance() — blocking check
  → [PASS] → overwrite file
  → [FAIL] → skip write, report violations
```

## Migration

Existing `mem-*` directories are renamed to `memory-*` via a migration function (same pattern as existing `migrateMemoryGenSkills`). Existing `mem:` names in SKILL.md files are updated to `memory.` during the rename.

## Validation Rules

| Rule | Field | Constraint |
|------|-------|------------|
| V1   | Description | Must start with "Use when" (case-sensitive) |
| V2   | Description | Must be <= 1024 characters |
| V3   | Description | Must be produced by `generateTriggerDescription()` (template path) or the LLM JSON `description` field (LLM path) — not derived from content truncation. Runtime check: verify description was not produced by `ExtractSkillDescription()` or equivalent substring extraction |
| V4   | Description | Must be third person (no "I", "you", "we" as subject) |
| V5   | Body (template) | Must contain `## Overview`, `## When to Use`, `## Quick Reference`, `## Common Mistakes` headers |
| V6   | Body | Must be <= 500 lines |
| V7   | Name | Must start with `memory.` |
| V8   | Frontmatter | Must contain `generated: true` |
