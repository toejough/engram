# Glossary triage

Terminology inconsistencies found while assembling
[`GLOSSARY.md`](GLOSSARY.md). Each "Needs Review" item names *what*
disagrees, *where* it disagrees, and a proposed canonical form. Move items
to **Decided** once you've ruled, recording the decision and any
follow-up edits the codebase needs.

## Needs Review

### 4. Note-type names vs auto-opener labels are non-parallel
- Type names are **Feedback** and **Fact**.
- The auto-generated body opener says `Lesson learned: …` for Feedback and
  `Information learned: …` for Fact.
- Old notes in the vault occasionally collapse these — some Fact-style
  notes are tagged as Feedback because the distinguishing language
  ("information" vs "lesson") didn't match the type name.
- **Proposed canonical:** rename the openers to match the types — `Feedback
  noted: …` and `Fact noted: …` — or rename the types (less disruptive
  to keep the openers and rename types `Lesson` / `Information`).

### 9. "transcript" vs "session" used interchangeably
- Docs/skills use both "transcript" and "session", sometimes
  interchangeably (transcript ⊇ session content; session = unit of work).
- The skill uses "session" most of the time.
- **Proposed canonical:** *session* = the time-bound interaction; *
  transcript* = its serialized record. `engram ingest` reads serialized
  records through the `internal/transcript` package, not live sessions —
  so "transcript" is correct where the serialized record is meant.
  (The retired `engram transcript` subcommand previously anchored this
  distinction; ingest now does.) Update prose to keep the distinction
  sharp wherever it matters (mostly skill text and CLAUDE.md tree comment).

### 10. "tier" / L0–L3 vocabulary — Decided/Retired
- **Decided 2026-06-20 (deep clean):** L1/L3 tiers and the `--tier` flag
  are removed. The binary writes only L2 notes (fact/feedback); the tier
  frontmatter field is kept for backward compatibility with existing vault
  notes but is no longer a filter on `engram query`. Tier vocabulary is
  retired from user-facing docs; it appears only in ADR history.

### 11. "skill" vs "command" vs "slash command"
- `skills/` holds SKILL.md files. `commands/` holds OpenCode slash
  commands. Claude Code calls invocations like `/learn` "slash commands".
- README says: "engram update writes Claude Code skills to
  ~/.claude/skills/ and OpenCode skills + commands to …" — implying
  OpenCode has both. Claude Code's slash commands map to skills directly.
- **Proposed canonical:** *skill* = the SKILL.md behavior definition;
  *slash command* = the user-facing `/name` trigger that invokes a skill
  in a harness; *command* (without "slash") = an OpenCode-specific file
  in `commands/` that wraps a skill invocation. Audit README to make
  this distinction explicit.

### 13. `Path A` / `Path B` / `Path C` naming
- The learn skill uses "Path A" (current-locus, recall bracketed segment),
  "Path B" (current-locus, no recall bracketed segment), and "Path C"
  (retro-locus, reconstruct from injecting agent's situation). Letter
  labels carry no mnemonic.
- **Proposed canonical:** keep as-is unless a better mnemonic emerges
  (*post-recall* / *cold-start* / *retrojective* are candidates) —
  flagging in case you want to rename while the term is still narrowly
  known.

### 14. `slug` vs "kebab-case tag"
- Flag name: `--slug`.
- Skill prose: "kebab-case tag".
- **Proposed canonical:** **slug** everywhere. Update skill prose to
  match the flag name.

### 15. "engram" the project vs "engram" the binary
- Used interchangeably in README and CLAUDE.md.
- Sometimes ambiguous: "engram resolves the vault automatically" — is
  that the project's behavior or the binary's?
- **Proposed canonical:** *engram (project)*, *engram (binary)*,
  *`engram` (CLI command)* with the appropriate qualifier when the
  difference matters. In most prose context disambiguates and a bare
  *engram* is fine.

---

## Decided

### Retired (issue 649 — transcript/episode/marker surface removed)

The lazy-L2 retirement removed the `engram transcript` / `engram learn episode`
binary surface and the `internal/learnmarker` package (chunks are now the
episodic layer; `engram ingest` advances chunk provenance). The following items
named that surface and are no longer live inconsistencies:

- **1. "harness" (docs) vs "source" (code)** — the `transcript.go` `sources`
  fan-out is gone. The harness/source split now lives only in
  `internal/transcript` (kept for `engram ingest`); revisit there if it still
  matters, but the transcript-subcommand framing it described is retired.
- **2. `Package transcript` doc comment stale on OpenCode** — `internal/transcript`
  was retained for `engram ingest`; the OpenCode SQLite backend (`opencode.go`)
  was removed in the 2026-06-20 deep clean. Engram reads JSONL only. The package
  doc comment now correctly reflects this.
- **7. `--source` overloaded** — the `engram transcript` "source = harness"
  use is gone; `--source` is now unambiguously the `engram learn` provenance
  string.
- **8. "marker" naming sprawl** — the `learnmarker` package and the
  `last-learn-at-<harness>` progress markers are retired with the transcript
  subcommand. No marker vocabulary remains to canonicalize.

### Retired (2026-06-20 deep clean — flat vault + subgraph/hubs removal)

- **3. `anchors`/`StartingPoints`** — `StartingPoints` (`internal/vaultgraph`) was removed
  in the 2026-06-20 deep clean (A.1). The cascade/anchors concept is gone; neither
  term is live. The GLOSSARY `anchors` entry is deleted.
- **5. `Permanent/` folder naming** — the `Permanent/` folder was retired in the
  2026-06-12 flat-vault migration; notes now live at the vault root. The vault
  bootstrap no longer creates `Permanent/`. Naming inconsistency is moot.
- **6. `MOCs/` folder naming** — the `MOCs/` folder was retired in the same migration.
  No live folder means no canonical folder-name question. MOC/MOCs vocabulary is
  retained in the GLOSSARY for historical context only.
- **StartingPoints resolved** — deleted in Phase A.1 of the deep clean; the dead-code
  entry is removed from c3-components.md.
