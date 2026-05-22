# Glossary triage

Terminology inconsistencies found while assembling
[`GLOSSARY.md`](GLOSSARY.md). Each "Needs Review" item names *what*
disagrees, *where* it disagrees, and a proposed canonical form. Move items
to **Decided** once you've ruled, recording the decision and any
follow-up edits the codebase needs.

## Needs Review

### 1. "harness" (docs) vs "source" (code)
- **Docs/skills:** README, CLAUDE.md, and both SKILL.md files call Claude
  Code and OpenCode *harnesses* ("per-harness progress marker", "every
  detected harness's user directory").
- **Code:** `internal/cli/transcript.go:488` literally `sources :=
  []string{"claude", "opencode"}`; the map key for marker bookkeeping is
  `source`; the SessionFinder fan-out treats them as sources.
- **Marker filenames** split the difference: `last-learn-at-claude`,
  `last-learn-at-opencode` (neither word appears).
- **Proposed canonical:** *harness* in user-facing prose; rename the
  code-internal `sources` map/variable to `harnesses` to match.

### 2. `Package transcript` doc comment is stale on OpenCode
- `internal/transcript/transcript.go:1` says "Package transcript finds and
  reads **Claude Code** session transcripts." `SessionFinder` doc at
  line 95 says the same.
- The package now also reads OpenCode (`opencode.go` defines
  `OpencodeSessionFinder`, `OpencodeTranscriptReader`, etc.) and the
  README correctly describes both.
- **Proposed canonical:** update both doc comments to "Claude Code JSONL +
  OpenCode SQLite", matching the README.

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

### 7. `--source` is overloaded
- In `engram learn`, `--source` is a **provenance string** ("session log
  engram, 2026-05-15, context: …").
- In `engram transcript`, the same word names a **harness** ("claude",
  "opencode").
- **Proposed canonical:** keep `--source` for provenance (it's a known
  zettelkasten term); rename the code-level transcript "source" concept
  to "harness" per item 1 above. No flag conflict, but the dual use
  confuses prose.

### 8. "marker" naming sprawl
- Package: `learnmarker`.
- Files on disk: `last-learn-at-<harness>`.
- README prose: "progress marker", "per-harness progress marker".
- Skill prose: "marker".
- CLAUDE.md: "Per-harness progress marker (read/write/FS interface)".
- **Proposed canonical:** **per-harness progress marker** on first use;
  **marker** thereafter. Keep `learnmarker` as the package name — it's
  already shipped — but document the connection in the package doc
  comment.

### 9. "transcript" vs "session" used interchangeably
- README uses both: "Read session transcripts since last /learn"
  (transcript ⊇ session content) and "sessions from one harness don't
  skip" (sessions = unit of work).
- The skill uses "session" most of the time.
- **Proposed canonical:** *session* = the time-bound interaction; *
  transcript* = its serialized record. The binary subcommand is
  `engram transcript` because the binary reads serialized records, not
  live sessions. Update prose to keep the distinction sharp wherever it
  matters (mostly skill text and CLAUDE.md tree comment).

### 10. "tier" / L0–L3 vocabulary present in design doc but unimplemented
- `docs/superpowers/specs/2026-05-14-tiered-memory-design.md` and
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

### 12. `engram recall` "no-arg" behavior overlap with `--recent`
- The skill says the no-arg recap "Anchors (`engram recall`) seeded with
  `engram recall --recent`."
- That's two separate invocations the skill loops together — but the
  prose reads like there's a single mode named "no-arg recap" that
  bundles them.
- **Proposed canonical:** name the two streams explicitly — *anchors
  stream* (`engram recall` with no flags) and *recent stream*
  (`engram recall --recent`) — and call their union the *initial
  frontier*. The glossary uses this framing; the skill should too.

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

*(empty — pending your review of the items above)*
