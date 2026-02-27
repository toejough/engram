# Feature Specification: Skill Generation Plugin Compliance

**Feature Branch**: `014-skill-gen-compliance`
**Created**: 2026-02-18
**Status**: Draft
**Input**: ISSUE-234 - Generated skills don't meet plugin skill criteria

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Generated Skills Trigger Correctly in Claude Code (Priority: P1)

A developer runs `projctl optimize` and skills are generated from memory clusters. When Claude Code loads the generated skill files, it correctly identifies when to invoke each skill based on its description field. Currently, descriptions contain raw content dumps that cause Claude to follow the description text instead of loading the skill body, defeating the purpose of skill generation.

**Why this priority**: This is the critical gap. If skills don't trigger at the right time, the entire skill generation pipeline produces unusable output. No other fix matters if Claude can't discover and correctly invoke generated skills.

**Independent Test**: Can be fully tested by running skill generation, then verifying that the output SKILL.md files have description fields that start with "Use when" and are under 1024 characters. Delivers skills that Claude Code can correctly select from its skill index.

**Acceptance Scenarios**:

1. **Given** a memory cluster about error handling patterns, **When** skill generation runs, **Then** the generated SKILL.md has a `description` field that starts with "Use when" followed by triggering conditions (not a content summary), and the description is under 1024 characters.
2. **Given** a skill generated without the LLM compiler (template fallback mode), **When** the SKILL.md is written, **Then** the `description` field still contains a proper trigger description derived from the skill's theme, not truncated content.
3. **Given** a previously generated skill with a content-dump description, **When** `projctl optimize` regenerates that skill during a normal reorg cycle, **Then** the regenerated skill has a compliant trigger description. No special retroactive migration is performed.

---

### User Story 2 - Generated Skill Body Structure Matches Expected Format (Priority: P2)

A developer opens a generated SKILL.md file and finds it organized into the standard sections: Overview, When to Use, Quick Reference, and Common Mistakes. This structure allows Claude to parse the skill content predictably and present it correctly when invoked.

**Why this priority**: Even with correct triggering, poorly structured content degrades skill usefulness. Standard structure ensures skills are readable and parseable.

**Independent Test**: Can be tested by generating a skill and verifying the output body contains the expected section headers in the correct order.

**Acceptance Scenarios**:

1. **Given** a memory cluster with sufficient content, **When** skill generation runs in template fallback mode (no LLM), **Then** the skill body contains sections "Overview", "When to Use", "Quick Reference", and "Common Mistakes" in that order.
2. **Given** a memory cluster, **When** skill generation runs with the LLM compiler, **Then** the LLM output contains both a trigger description and a structured content body as separate outputs.
3. **Given** a generated skill, **When** the body exceeds 500 lines, **Then** the skill fails validation and is not written to disk. The violation is reported in the optimize output with the actual line count.

---

### User Story 3 - Post-Generation Validation Catches Non-Compliant Skills (Priority: P3)

After skill generation completes, a validation step checks every generated skill file for compliance: description format, description length, body structure, and body length. Non-compliant skills are flagged in the optimize output so the developer knows which skills need attention.

**Why this priority**: Validation provides a safety net that catches regressions and edge cases that the generation logic might miss. Without it, non-compliant skills silently enter the system.

**Independent Test**: Can be tested by generating a skill with a deliberately non-compliant description, then running validation and confirming the violation is reported.

**Acceptance Scenarios**:

1. **Given** a generated skill with a description that doesn't start with "Use when", **When** post-generation validation runs, **Then** the skill is flagged as non-compliant with a specific reason.
2. **Given** a generated skill with a description exceeding 1024 characters, **When** validation runs, **Then** it is flagged with the actual character count.
3. **Given** a generated skill in template fallback mode missing required body sections, **When** validation runs, **Then** each missing section is listed in the validation output.

---

### Edge Cases

- What happens when a memory cluster has only 1-2 short memories, making it impossible to produce all four body sections meaningfully? → Skill generation is skipped for that cluster. No skill with placeholder content is created.
- What happens when the LLM compiler produces a description that doesn't start with "Use when" despite being prompted to? → The blocking validation (FR-008/FR-009) catches it. The non-compliant skill is not written to disk. The LLM output still provides the body content; the template-path `generateTriggerDescription()` produces a compliant description as fallback.
- What happens when the skill theme contains special characters that break YAML frontmatter formatting? → The `description` field in frontmatter is YAML-quoted when it contains special characters (colons, quotes, newlines). The existing `slugify()` function already strips special chars from slugs/directory names.
- What happens when an existing skill on disk was manually edited by the user and optimization re-generates it? → Only skills with both `generated: true` frontmatter and `memory.` name prefix are touched. Users protect edits by removing either marker.
- What happens when the LLM returns valid JSON but the `description` field is empty or null? → The LLM body content is kept; `generateTriggerDescription()` produces a compliant description as fallback. The same fallback applies when the LLM returns valid markdown but not valid JSON (entire response falls through to template).
- What happens when an existing `memory-{slug}/` directory already exists during migration from `mem-{slug}/`? → The migration skips that skill and logs a warning. The existing `memory-{slug}/` content is preserved.
- What happens when a user manually creates a skill with the `memory.` name prefix? → The `memory.` prefix is reserved for pipeline-managed skills. If the user does not set `generated: true`, the dual guard (FR-013) prevents the pipeline from touching it. If the user sets both `memory.` prefix AND `generated: true`, they are opting into pipeline management.
- What happens when `ExtractSkillDescription()` is replaced? → It is deprecated and removed if no non-skill callers remain. All description generation uses `generateTriggerDescription()` (template path) or the LLM JSON `description` field.

## Clarifications

### Session 2026-02-18

- Q: Should validation be blocking (prevent non-compliant skills from being written) or advisory (write anyway, report warnings)? → A: Blocking — non-compliant skills are not written to disk; violations are reported in optimize output.
- Q: Should compliance changes apply retroactively to all existing skills or only to newly generated ones? → A: Forward-only — new and regenerated skills must comply; existing skills become compliant during normal reorg cycles.
- Q: For small clusters, should all four body sections still be required or should requirements be relaxed? → A: If a cluster doesn't have enough information to meaningfully populate all four sections, no skill should be generated. Don't create skills with placeholder content.
- Q: Should manually edited skills be protected from regeneration? → A: Dual guard — only touch skills that have BOTH `generated: true` in frontmatter AND a `memory.` name prefix (e.g., `memory.error-handling`). Skills missing either marker are never overwritten. The `mem-` directory prefix also changes to `memory-` for clarity. This makes it obvious to users which skills are auto-generated from their memories.

### Session 2026-02-19

- Q: Is "forward-only" compliance consistent with US1-AS3 (skills updated during reorg)? → A: Yes. "Forward-only" means no special retroactive migration for content/descriptions. Reorg cycles are forward-only — they apply new compliance rules when skills are naturally regenerated. US1-AS3 describes this natural regeneration path.
- Q: How do blocking validation (FR-009) and existing `TestAndCompileSkill()` coordinate? → A: They are sequential gates. First compile (LLM or template), then `ValidateSkillCompliance()`, then `writeSkillFile()`. Validation is an additional gate after compilation; they do not conflict.
- Q: Is the 1024-character limit bytes or Unicode characters? → A: Measured as Go string length (`len(string)`), which is UTF-8 byte count. For ASCII descriptions (the expected case), bytes equals characters.
- Q: What constitutes "sufficient information" in FR-011? → A: Coordinates with existing `minClusterSize` (default 3 embeddings). Clusters below this threshold are filtered before generation reaches the template.
- Q: Should SC-001 be measurable automatically? → A: Yes, via unit tests that run skill generation and inspect output files programmatically.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Generated skill `description` fields MUST start with "Use when" followed by triggering conditions that describe when Claude should load the skill. This applies to all 5 generation code paths: `optimizeCompileSkills`, `performSkillReorganization`, `optimizeSplitSkills`, `optimizeDemoteClaudeMD`, and `PromoteEmbeddingToSkill`.
- **FR-002**: Generated skill `description` fields MUST be under 1024 characters (measured as Go string length, i.e., UTF-8 byte count).
- **FR-003**: Generated skill `description` fields MUST be written in third person. Prohibited subject pronouns: "I", "my", "me", "we", "our", "us" (first person) and "you", "your" (second person). Required style: "the user", "the developer", "they" or passive voice.
- **FR-004**: Generated skill `description` fields MUST NOT contain content summaries or workflow descriptions. They must describe triggering conditions only. Enforced at generation time by `generateTriggerDescription()` (template path) and the LLM prompt (LLM path), not by runtime validation. Examples — Pass: `"Use when the user encounters error handling patterns or needs guidance on retry strategies."` Fail (content summary): `"This skill covers error handling including try/catch blocks, retry logic, and timeout configuration."` Fail (workflow): `"Step 1: Check for errors. Step 2: Retry the operation."`
- **FR-005**: When generating skills in template fallback mode (no LLM compiler), the body MUST contain sections: "Overview", "When to Use", "Quick Reference", and "Common Mistakes".
- **FR-006**: When generating skills with the LLM compiler, the system MUST produce a trigger description and structured body as separate outputs (not derived by truncating content).
- **FR-007**: Generated skill bodies MUST NOT exceed 500 lines.
- **FR-008**: After skill generation, a validation step MUST check each generated skill for: description format (V1: starts with "Use when"), description length (V2: ≤1024 chars), description independence (V3: not derived from content truncation), description person (V4: third person), body structure presence (V5: required sections), body length (V6: ≤500 lines), naming convention (V7: `memory.` prefix), and generated flag (V8: `generated: true`).
- **FR-009**: Validation failures MUST be reported in the optimize output with specific reasons per skill. Non-compliant skills MUST NOT be written to disk.
- **FR-010**: The `description` field MUST be generated as a distinct output, never by truncating the skill content body.
- **FR-011**: If a memory cluster lacks sufficient information to meaningfully populate all four required body sections, skill generation MUST be skipped for that cluster. No skill with placeholder or generic content is created. The threshold coordinates with the existing `minClusterSize` (default 3 embeddings): clusters below this size are filtered before generation. The skip is reported in the optimize output via `OptimizeResult.SkillsSkipped`.
- **FR-012**: Generated skill names MUST use the `memory.` prefix with dot separator (e.g., `memory.error-handling`) in frontmatter, replacing the current `mem:` prefix. Skill directories MUST use the `memory-` prefix with hyphen separator (e.g., `memory-error-handling/`). The naming change is filesystem and frontmatter only — the DB `generated_skills.slug` column stores slugs without any prefix.
- **FR-013**: The optimization pipeline MUST only modify skills that have BOTH `generated: true` in frontmatter AND a `memory.` name prefix. Skills missing either marker MUST NOT be overwritten or regenerated.
- **FR-014**: When skill generation is skipped due to insufficient content (FR-011) or blocked by validation (FR-009), the optimize output MUST report the count and reasons. `OptimizeResult` includes `SkillsSkipped int` (insufficient content) and `SkillsBlocked int` (validation failures) with `ValidationIssues []SkillComplianceResult` for detailed per-skill violations.

### Key Entities

- **Generated Skill**: A SKILL.md file produced by the memory compilation pipeline, identified by the dual markers: `generated: true` frontmatter and `memory.` name prefix (e.g., `memory.error-handling`). Stored in `memory-{slug}/SKILL.md` directories. Consists of YAML frontmatter (name, description, metadata) and a markdown body (structured content from memory clusters).
- **Skill Description**: The frontmatter `description` field used by Claude Code's skill index to determine when to load and invoke the skill. Distinct from the skill body content.
- **Memory Cluster**: A group of related memories (embeddings) that are compiled together into a single skill. Minimum cluster size for skill generation is `minClusterSize` (default 3 embeddings).
- **DB `generated_skills` table**: Stores slugs WITHOUT any prefix in the `slug` column. The naming convention change (`mem:` → `memory.`) is purely a filesystem and frontmatter concern — no DB schema changes required.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of generated skills have `description` fields that start with "Use when" and are under 1024 characters. (MVP milestone — subset of SC-005 V1+V2.)
- **SC-002**: 100% of template-fallback generated skills contain all four required body sections.
- **SC-003**: 0 generated skills have description fields derived from content truncation. Verified by confirming that all template-path descriptions are produced by `generateTriggerDescription()` and all LLM-path descriptions come from the separate JSON `description` field, not by substring extraction from the body.
- **SC-004**: Post-generation validation blocks 100% of non-compliant skills from being written to disk. Blocked count is surfaced via `OptimizeResult.SkillsBlocked` and per-skill violations via `OptimizeResult.ValidationIssues`. Measurable via unit tests that inject non-compliant skills and assert blocked count > 0 and no files written.
- **SC-005**: Generated skills pass all 8 validation checks (V1-V8 from data-model.md): description starts with "Use when" (V1), description ≤1024 chars (V2), description not truncated from content (V3), description in third person (V4), body contains required sections (V5), body ≤500 lines (V6), name has `memory.` prefix (V7), frontmatter has `generated: true` (V8). Measurable via `ValidateSkillCompliance()` unit tests.
