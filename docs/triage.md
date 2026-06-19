# Glossary triage

Terminology inconsistencies found while assembling
[`GLOSSARY.md`](GLOSSARY.md). Each "Needs Review" item names *what*
disagrees, *where* it disagrees, and a proposed canonical form. Move items
to **Decided** once you've ruled, recording the decision and any
follow-up edits the codebase needs.

## Needs Review

### 3. `anchors` (docs/skill) vs `StartingPoints` (code)
- `skills/recall/SKILL.md` and README both call the cascade entry set
  *anchors*.
- `internal/vaultgraph/parser.go:1` says "Public entry point:
  **StartingPoints** — emits one canonical wikilink per starting point".
- **Proposed canonical:** *anchors* in prose and code. Either rename the
  Go function to `Anchors`, or update the doc comment to "Anchors (aka
  starting points)" if the function name has to stay for compat.

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

### 5. "Permanent" vs "Permanents" vs "permanent note"
- Folder: `Permanent/` (singular, capital).
- README §"Vault layout": `Permanent/   atomic principle-stated notes`.
- Skills mostly say "permanent note(s)" lowercase.
- Some prose uses bare "Permanent" as a noun ("write a Permanent for
  …").
- **Proposed canonical:** *permanent note* (lowercase) for the concept;
  *Permanents* (capital, plural) when referring to the collection;
  *Permanent/* (with slash) only for the folder. Bare capital "Permanent"
  as a noun should be retired.

### 6. "MOC" / "MOCs" / "Map of Content" / "Maps of Content"
- Folder: `MOCs/`.
- README: "Maps of Content" and "MOCs".
- Skills: mostly "MOC" / "MOCs".
- **Proposed canonical:** **MOC** (singular) and **MOCs** (plural) in
  running prose; spell out **Map of Content** on first use in a doc;
  folder `MOCs/`. Avoid "Maps of Content" — the acronym pluralizes with
  a lowercase "s", not by re-pluralizing the expansion.

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

### 10. "tier" / L0–L3 vocabulary present in design doc but unimplemented
- The legacy tiered-memory design (now in `docs/DESIGN-HISTORY.md` §1) and
  `MOCs/65.memory-system-design` use "L0/L1/L2/L3 tiers".
- The current vault has only Permanent + MOCs (effectively L2 + L3 in the
  design doc's terms, with no L0 or L1).
- The user-facing README and skills don't mention tiers at all.
- **Proposed canonical:** keep tier vocabulary scoped to the design doc
  until implementation lands; do **not** introduce L0/L1/L2/L3 into the
  glossary, skills, or README until they're real. Flag this here so
  future readers of the design doc don't think the project ships with
  tiers.

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
  is retained for `engram ingest` and still reads both Claude Code JSONL +
  OpenCode SQLite; the doc-comment freshen is a minor follow-up, not a
  glossary inconsistency anymore.
- **7. `--source` overloaded** — the `engram transcript` "source = harness"
  use is gone; `--source` is now unambiguously the `engram learn` provenance
  string.
- **8. "marker" naming sprawl** — the `learnmarker` package and the
  `last-learn-at-<harness>` progress markers are retired with the transcript
  subcommand. No marker vocabulary remains to canonicalize.
