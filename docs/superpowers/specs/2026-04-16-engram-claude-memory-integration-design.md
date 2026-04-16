# Engram + Claude Code Memory Integration

**Date:** 2026-04-16
**Status:** Spec — awaiting plan
**Scope:** Make engram complementary to (not competitive with) Claude Code's built-in memory by cross-searching external sources on read and deduplicating against them on write.

---

## Context

Engram has shipped its files-and-search-and-synthesis memory model (situation-framed memories, three-phase recall pipeline, situation-quality remediation). Claude Code natively offers three documented memory mechanisms:

1. **CLAUDE.md hierarchy** — user-written prescriptive instructions in managed / project / user / local scopes, plus `@path` imports up to 5 hops.
2. **`.claude/rules/*.md`** — modular topic-scoped rules with optional `paths:` frontmatter.
3. **Auto memory** (Claude Code v2.1.59+) — Claude-written notes at `~/.claude/projects/<project>/memory/`, indexed by `MEMORY.md` (first 200 lines / 25 KB autoloaded), topic files loaded on demand.

Plus skills (`SKILL.md` files in project, user, and plugin-cache locations).

Engram and Claude Code's auto memory are on a collision course: both are "Claude-written per-project learnings." Without coordinated design, engram saves things that already live in CLAUDE.md / auto memory / skills, and engram's recall ignores curated content that lives in those sources.

## Goals

- **Read side**: engram `recall` and `prepare` synthesize content from CLAUDE.md hierarchy, rules, auto memory, and installed skills alongside engram's own memories and session transcripts.
- **Write side**: engram `learn` and `remember` deduplicate candidate memories against the same external sources before saving — same DUPLICATE flow as today (always allow override).
- Preserve engram's role as the place for **task-triggered, situation-framed, cross-session memories**. External sources own different content types (prescriptive rules, curated notes, packaged workflows).

## Non-goals

- Promoting/demoting memories between engram ↔ CLAUDE.md ↔ auto memory automatically. (Tier-shifting is a future evolution, not this spec.)
- Writing to external sources from engram (no auto-update of CLAUDE.md or auto memory files).
- Cross-machine sync of any external source. Auto memory is documented as machine-local; engram respects that.
- Replacing or shadowing Claude Code's own loading of CLAUDE.md / auto memory at session start. Engram is additive.

---

## Architecture

A new subsystem, **external sources**, lives at `internal/externalsources/`. It owns discovery, file reading, and the per-invocation file cache. The existing recall pipeline in `internal/recall/` gains new phase types that consume from external sources.

### Cross-search pipeline (six phases)

The existing three-phase pipeline becomes six phases. Each phase Haiku-extracts relevant content into a shared 10 KB buffer. Phases run in priority order; once the buffer fills, downstream phases are skipped.

| Phase | Source | Rationale |
|-------|--------|-----------|
| 1 | Engram memory search (existing) | Curated, task-framed, not surfaced anywhere else |
| 2 | Auto memory extraction (NEW) | Curated by Claude — high probability of true relevance |
| 3 | Session extraction (existing) | Unfiltered transcripts — not surfaced elsewhere |
| 4 | Skill extraction (NEW) | Specialized; frontmatter may not always trigger discovery |
| 5 | CLAUDE.md hierarchy + rules extraction (NEW) | Already in session context — reinforcement only |
| 6 | Final synthesis (existing) | Haiku produces polished output |

Priority reasoning: engram surfaces what is otherwise invisible first; reinforcement-only sources fall to the back of the line and are acceptably skipped when the buffer fills.

### Write path (learn / remember dedup)

Same six-phase pipeline runs with the candidate memory's `situation + subject/behavior` as the query. Phase 6 is replaced with a **dedup-judge** Haiku call:

> "Given this candidate memory and these matched snippets from various sources, which (if any) already convey the same actionable lesson? Return `{source_kind, path, snippet}` for each — empty if none."

Hits are returned as a structured DUPLICATE response. No automatic refusal — the existing engram DUPLICATE flow applies (`--force` overrides, see below).

### Shared in-memory file cache

`fileCache map[string]*cachedFile` keyed by absolute path. Populated lazily on first `Read(path)`. Cached errors prevent repeated permission-denied attempts. Lives for one engram process invocation only — no cross-invocation persistence (engram is invoked once per call, not as a daemon).

---

## File discovery

Discovery runs once per invocation, producing `[]ExternalFile{Kind, Path}` tuples consumed by extractors.

### CLAUDE.md hierarchy
- Walk ancestors from `cwd` to `/`, collecting `CLAUDE.md` and `CLAUDE.local.md` at each level.
- Add `~/.claude/CLAUDE.md` (user scope).
- Add managed policy path:
  - macOS: `/Library/Application Support/ClaudeCode/CLAUDE.md`
  - Linux/WSL: `/etc/claude-code/CLAUDE.md`
  - Windows: `C:\Program Files\ClaudeCode\CLAUDE.md`
- Recursively expand `@path/to/file.md` imports relative to the file containing them, up to 5 hops (the documented cap). Detect cycles.

### Rules files
- `<project>/.claude/rules/**/*.md`
- `~/.claude/rules/**/*.md`
- Parse frontmatter `paths:` glob. Files with a glob are included only when at least one file under `cwd` matches; files without `paths:` are always included.
  - **Adaptation note:** Claude Code itself triggers path-scoped rules when it reads matching files; engram runs offline against a query and has no per-read context. The "matches at least one file under cwd" heuristic is engram-specific and intentionally broader.

### Auto memory
- Resolve directory in this order:
  1. `autoMemoryDirectory` from `./.claude/settings.local.json` or `~/.claude/settings.json`.
  2. `~/.claude/projects/<slug>/memory/` where `<slug>` is the slugified absolute path of `cwd`.
  3. If (2) does not exist and `cwd` is inside a worktree, slugify the main repo root and try again.
- Collect `MEMORY.md` plus all other `*.md` siblings.

### Skills
- `<project>/.claude/skills/*/SKILL.md`
- `~/.claude/skills/*/SKILL.md`
- `~/.claude/plugins/cache/*/*/skills/*/SKILL.md`
- Parse frontmatter (`name`, `description`) eagerly — drives the rank step. Body is read lazily during extraction.

### Error resilience

- Missing files / dirs: skip silently, log at debug.
- Permission errors: skip the file, log at warn.
- Malformed frontmatter: skip the file, log at warn.
- One source failing never blocks other phases.

---

## Recall / prepare flow — buffer-fill algorithm

**Core principle:** index-then-extract. One cheap "rank" Haiku call per source picks files; body extraction runs only on the ranked winners in order, stopping when the buffer fills.

### Phase 2 — Auto memory
1. Read `MEMORY.md` (cached after first read).
2. **One** Haiku call: "Given `MEMORY.md` and query `X`, return topic filenames ordered by relevance."
3. Iterate returned files: read body, Haiku-extract snippets, append to buffer. Break when `len(buffer) >= budget`.

### Phase 3 — Sessions (existing)
Unchanged. Newest-first, Haiku-extract per session, append until buffer fills.

### Phase 4 — Skills
1. Assemble `[(name, description, path), …]` from frontmatter discovered up front.
2. **One** Haiku call: "Given skill index and query `X`, return skill names ordered by relevance."
3. Iterate winners in order: read `SKILL.md` body, Haiku-extract, append. Break when buffer fills.

### Phase 5 — CLAUDE.md + rules
1. Concatenate all discovered `CLAUDE.md` / `CLAUDE.local.md` / rules files. (Each is small by design — under 200 lines.)
2. **One** Haiku call: extract snippets relevant to query. Append.

### Buffer budget

- Same 10 KB shared budget as today.
- Check `if len(buffer) >= budget: skip remaining phases` between phases and between extracts within a phase.
- Priority order ensures the most-relevant-likely sources win the budget.

### Haiku call count (worst case)

| Phase | Rank | Extract |
|-------|-----:|--------:|
| 1 (engram) | 1 | 0 |
| 2 (auto memory) | 1 | M (until buffer fills) |
| 3 (sessions) | 0 | N (until buffer fills) |
| 4 (skills) | 1 | K (until buffer fills) |
| 5 (CLAUDE.md) | 0 | 1 (combined) |
| 6 (synthesis) | 1 | — |

Total: same order of magnitude as today; the rank steps are cheap (frontmatter / index input only, tiny output).

### Status output

Existing `WithStatusWriter` hook gains new events for each phase boundary and each rank/extract call so users see progress during long recalls.

---

## Learn / remember dedup + DUPLICATE flow

### Dedup pipeline

Reuses phases 1–5 with the candidate's `situation + subject/behavior` as the query. Phase 6 is the **dedup-judge** Haiku call described in Architecture.

### DUPLICATE response shape

```
DUPLICATE
  - kind: engram
    name: <memory_name>
    situation: "<situation>"
  - kind: auto_memory
    path: ~/.claude/projects/<slug>/memory/debugging.md
    snippet: "…"
  - kind: claude_md
    path: ./CLAUDE.md
    snippet: "…"
```

All hits across all sources are reported. Caller (skill) decides what to do.

### `--force` flag

```bash
engram learn fact --situation "…" --subject "…" --predicate "…" --object "…" --source agent --force
```

- With `--force`: skip dedup entirely, save as CREATED.
- Without `--force`: run dedup; on any hit, return DUPLICATE without saving.

The default fails closed on likely duplicates. `--force` is the documented escape valve when the agent or user disagrees with Haiku's judgment.

### Skill updates

`skills/learn/SKILL.md` and `skills/remember/SKILL.md` Step 5 ("Handle results") expand to cover source kinds:

- **engram dup** — existing flow: diagnose `/recall` or `/prepare` query pattern; broaden situation with `engram update`.
- **auto_memory dup** — Claude's own notes already capture this. Examine query patterns rather than the auto memory file.
- **skill dup** — skill already covers this. If the skill's `description` frontmatter is not triggering reliably, the skill is the right home for an improvement.
- **claude_md / rules dup** — rule already loaded into context. Engram would be redundant; consider tightening the rule.
- **Override** — re-run with `--force` to save anyway.

Skill edits follow `superpowers:writing-skills` (baseline test → update → pressure test).

### Edge cases

| Case | Behavior |
|------|----------|
| No external sources discoverable | Phases 2/4/5 complete quickly with zero contribution. Behaves as today. |
| Haiku false positive | `--force` documented in DUPLICATE output as the escape. |
| Haiku false negative | Duplicate saves; caught later in `/learn` audit. No worse than today. |
| Dedup API error | Fail closed: return error, don't save. Caller retries or uses `--force`. |
| Large candidate fields | Situation + subject/behavior is small; no truncation. |

---

## Testing

Per engram's DI convention, business logic is unit-tested with mocks; thin I/O wrappers get integration tests with real dependencies.

### Unit tests

- `internal/externalsources/discovery_test.go` — mocked fs walks: ancestor chain, auto memory slug resolution (including worktree fallback), skill-dir enumeration, `paths:` glob filter on rules, `@path` import expansion (5-hop cap, cycle detection).
- `internal/externalsources/cache_test.go` — cache hit after first read, error caching, no cross-invocation persistence.
- `internal/recall/phases_test.go` — each new phase invokes mocked Haiku with expected input; buffer-budget gate stops extraction at the documented threshold; phase ordering skips downstream once buffer is full.
- `internal/recall/dedup_test.go` — dedup-judge prompt returns correct source attribution; empty findings → CREATED; `--force` bypasses dedup entirely.

### Integration tests

- Real filesystem fixtures at `testdata/fixture-project/` containing CLAUDE.md, rules, fake skills, and a fake auto-memory dir. Verify discovery collects the expected file set and reads pass through correctly.
- One end-to-end test per new phase that exercises `recall --query …` against the fixture, with mocked Haiku returning canned outputs, asserting full pipeline output.

### Cost regression guard

A single test invokes recall against a fixture with 50 fake skills and 20 fake auto-memory topic files. Asserts total Haiku call count stays bounded by `(phases × 1 rank) + (M extracts capped by buffer)`. Fails loudly if someone fans out per-file Haiku calls.

### Skill quality gate

`skills/learn/SKILL.md` and `skills/remember/SKILL.md` updates follow `superpowers:writing-skills`: baseline behavioral test demonstrating today's DUPLICATE handling, update the skill, pressure test that the new source-kind branches behave as designed.

---

## Rollout

Single ship. No feature flag, no env-var gate. All new phases on by default.

Backward compatibility: when no external sources are discoverable (fresh project, no auto memory yet, no installed skills), the new phases complete quickly with zero contribution and behave identically to today.

Manual smoke test on engram's own repo: verify engram's `CLAUDE.md`, `.claude/rules/go.md`, and installed plugin skills are discovered; run a realistic recall query and inspect output.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Rank step returns irrelevant files | Per-file extract still runs; irrelevant input yields empty snippet, buffer stays clean. |
| Skill corpus grows large (hundreds) | Rank input is frontmatter-only; scales linearly with small bytes per skill. If measured slow, add literal-keyword prefilter. Defer until measured. |
| `autoMemoryDirectory` in non-standard location | Read setting before falling back to slug-based default. |
| Dedup false positives frustrate users | `--force` is first-class, documented in DUPLICATE output. |
| Project slug computation diverges from Claude Code's | Settings override is checked first; worktree fallback covers the most common edge. If a slug mismatch is reported, document the override path. |

---

## Open implementation details (resolved during planning)

- Exact slugification algorithm for `cwd` → `~/.claude/projects/<slug>/`.
- Concrete prompt templates for each rank Haiku call and the dedup-judge call.
- Whether `Summarizer` interface gains a new method (`SummarizeExternalSource(kind, content, query)`) or the existing `SummarizeFindings` is reused with an enriched input shape.
- Status-writer event names and JSON shape.
