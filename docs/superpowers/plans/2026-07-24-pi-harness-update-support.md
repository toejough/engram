# Plan — finish pi-agents-integration: pi as a first-class `engram update` harness

Date: 2026-07-24. Ask (Joe, verbatim intent): finish the branch per briefing — fix the
AGENTS.md import-prefix mismatch, extract the duplicated parsing loop, decide/implement
per-harness guidance-import attribution, add pi tests, verify with the real binary
(including the dangling write-memory.md import), then merge, push, remove the worktree.

## Decisions locked at orientation

- **Canonical pi guidance dir: `~/.pi/agent/engram/`** — REVERSAL of the briefing's
  "align on guidance/". New fact: both dirs exist on disk; `engram/` holds all 4 files
  and mirrors Claude's `~/.claude/engram/` layout; `guidance/` holds 3 and is stale.
  Recorded per anti-sycophancy resolution rule: considered guidance/, chose engram/
  because symmetry + completeness + note 442's engram-owned-folder sync doctrine.
  Consequence: the WIP's detection prefix (`@~/.pi/agent/engram/`) was CORRECT; the
  deploy spec (`GuidanceTargetRel`) and Joe's live AGENTS.md imports move to engram/.
- **Attribution semantics: per-harness.** `Report.GuidanceImports` becomes per-harness
  (keyed by harness name) so the CLI can say "wired in Claude Code, not in Pi".
  Merged-set behavior would mask a harness with zero imports.
- **Sync semantics (vault note 442, requirement):** deploys are syncs — removals
  propagate. Live migration also deletes the stale `~/.pi/agent/guidance/` dir and
  rewrites AGENTS.md imports guidance/ → engram/.
- **Live-run guard:** before any non-dry-run against `~/.pi/agent/skills`, verify
  `clearSkillDirOnce` scope — Joe's dir contains NON-engram skills (`ping/`). If sync
  clears the whole target dir rather than only engram-deployed skill dirs, that is a
  STOP: report before running live.

## Units (TDD each: RED → GREEN → REFACTOR + Gate B)

1. **U1 — per-harness imports + helper extraction.** Extract the duplicated inline
   loop into a `collectGuidanceImports(data, tildePrefix, expandedPrefix)` helper
   (wrapping existing `guidanceImportBase`); detection reads CLAUDE.md and AGENTS.md
   into `GuidanceImports map[Harness]map[string]bool` (or equivalent). RED: table test
   over both files incl. pi prefix `@~/.pi/agent/engram/`, code-fenced-line exclusion,
   per-harness attribution.
2. **U2 — pi harness spec correction.** `GuidanceTargetRel: .pi/agent/engram`
   (currently `.pi/agent/guidance`). RED: spec test asserting pi probe root, skills
   target `.pi/agent/skills`, guidance target `.pi/agent/engram`.
3. **U3 — report rendering.** Find the CLI consumer of `GuidanceImports`; render
   per-harness wiring status (and the existing "hint" line per harness). Tests to match.
4. **U4 — functional verification (real binary, real machine).**
   `engram update --dry-run` → pi harness listed with correct targets; then (guard
   permitting) live run; verify `~/.pi/agent/engram/` synced (4 files incl.
   write-memory.md — which resolves the dangling AGENTS.md import once imports move),
   AGENTS.md rewritten, stale `guidance/` removed, `ping/` skill SURVIVES, and pi's
   skills refreshed. Subprocess timeouts budgeted generously (note 364).

## Doc-surface enumeration grep — disposition list

Grep: `opencode|harness|guidance.?import|\.claude/engram` over README.md, docs/, agent-instructions/, cmd/.

| File | Disposition | Reason |
| --- | --- | --- |
| README.md (:10, :12, :32, :40, :108) | UPDATE | harness lists say "Claude Code and OpenCode"; :12 asymmetry paragraph is stale twice over (pi ingest shipped to main 2026-07-24 via pi-sessions-support; pi update-support ships here); :40/:108 --with-guidance targets gain `~/.pi/agent/engram/` + AGENTS.md import wording |
| docs/README.md | VERIFY→likely N/A | index page; update only if it enumerates harnesses |
| docs/FEATURES.md | UPDATE if lists harness support | verify at execution |
| docs/GLOSSARY.md | VERIFY→likely KEEP | "harness" entry is generic; add Pi only if entries enumerate |
| docs/ROADMAP.md | UPDATE | pi-integration row(s) exist → mark shipped/reference commits |
| docs/design/2026-07-01-engram-recall-subprocess-design.md | KEEP | historical design record |
| docs/architecture/c1-system-context.md, c2-containers.md, c3-components.md | UPDATE if harnesses enumerated in diagrams | hand-authored mermaid (note 171); verify labels |
| docs/architecture/adr.md | KEEP + possible ADR addendum | historical decisions; ADR notes "adding a third harness is an extension of the same seam" — pi is that third harness; add one line only if ADR practice requires |
| docs/architecture/memory-invariants.md, memory-system-rigor.md | VERIFY→likely KEEP | harness mentions are generic |
| agent-instructions/guidance/*.md | KEEP | source content, harness-generic |

## Out of scope

- OpenCode SQLite ingest (#644) — unrelated gap, stays open.
- Any change to pi-sessions ingest (shipped this morning).

## Verification gates

Gate A on this plan; Gate B per refactor; Gate C on docs; Gate D on commit prose.
Functional pass criteria in U4 are the completion bar (CLAUDE.md: passing tests ≠ done).
