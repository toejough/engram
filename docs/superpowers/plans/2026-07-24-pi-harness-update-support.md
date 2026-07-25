# Plan: pi harness update-support — completion (rev 2, post-Gate-A)

Date: 2026-07-24. Ask (verbatim): finish the pi-agents-integration branch — fix the
AGENTS.md import-prefix bug (align on guidance/), extract the duplicated inline parsing
loop into a helper, decide/implement per-harness guidance-import attribution, add
pi-harness tests (spec probing, deploy targets, AGENTS.md detection), run the real
`engram update --dry-run` functional verification including the dangling
write-memory.md import question, then merge to main, push, and remove the worktree.

## Decisions (settled; reopen only on a genuinely new fact)

- **D1 — canonical pi guidance dir: `~/.pi/agent/guidance/`, per the ask.** Rev 1
  reversed this to `engram/`; Gate A ask-alignment refuted every evidence leg of that
  reversal (Claude's live `~/.claude/engram/` holds 3 files, not 4; `guidance/` is
  newer on disk than `engram/`; the deploy source `agent-instructions/guidance/` ships
  exactly 3 files; the WIP deploy spec already targets `guidance/`). The retraction is
  recorded here. Consequence: the code fix is the detection prefix (engram/ →
  guidance/), exactly as the ask said, and `~/.pi/agent/engram/` is the stale orphan
  (manually removed in U4 migration).
- **D2 — write-memory.md is a skill, not guidance, in every harness.** Claude's
  CLAUDE.md imports 3 guidance files; the guidance deploy source has 3. The dangling
  `@~/.pi/agent/guidance/write-memory.md` import in AGENTS.md is erroneous and is
  REMOVED during U4's manual migration. (This answers the ask's "dangling import
  question" with a decision, not a coincidence of stray files.)
- **D3 — per-harness attribution, derived from the spec.** `Report.GuidanceImports`
  becomes per-harness. Detection prefixes are DERIVED from each `HarnessSpec`:
  tilde prefix `"@~/" + GuidanceTargetRel + "/"` (plus the expanded-home form), with a
  new `ImportsFileRel` field naming the harness config file (`.claude/CLAUDE.md` for
  Claude, `.pi/agent/AGENTS.md` for Pi; empty for OpenCode = detection skipped). One
  source of truth makes prefix-vs-target drift (this branch's original bug)
  structurally impossible.
- **D4 — sync semantics** (vault note 442, inlined): deploys sync engram-owned dirs —
  removals propagate; user content outside engram-owned names is never touched.
  Verified in code: `clearSkillDirOnce` (update.go:426-451) RemoveAlls only per-skill
  dirs named by the source repo walk (update.go:846-872) — Joe's non-engram `ping/`
  skill survives live runs. Live-run guard: RESOLVED SAFE, live run may proceed.
- **D5 — the tool never edits user config files.** Parity with the CLAUDE.md flow:
  guidance imports are a one-time user opt-in; `engram update` prints hints. AGENTS.md
  edits and stale-dir deletion on this machine are explicit MANUAL migration steps in
  U4, not tool behavior.

## Units (TDD; RED stated up front; Gate B after each refactor)

- **U1 — detection correctness + helper extraction.**
  - Baseline disclosure: the tree is ALREADY RED —
    `TestGuidanceImportDetection/inside-code-fence-ignored` fails because the WIP's
    inline loops dropped the fence-state tracking that (now-dead) `detectGuidanceImports`
    had. U1 fixes this, it does not merely refactor around it.
  - RED-1: the existing failing fence test (kept as-is).
  - RED-2: table test — per-harness attribution over CLAUDE.md + AGENTS.md fixtures
    (pi `guidance/` prefix per D1; imports inside fences ignored; pi import never
    attributed to Claude and vice versa).
  - RED-3: `rapid` property test (package convention, runner_test.go precedent):
    for generated harness specs + generated import/noise/fence lines, (a) an import
    line inside an unclosed fence is never counted, (b) every counted import
    attributes to exactly the harness whose derived prefix matched.
  - GREEN: `collectGuidanceImports(data, tildePrefix, expandedPrefix)` with fence
    state, called once per spec with derived prefixes; per-harness map; delete dead
    `detectGuidanceImports`.
  - REFACTOR: black-box testing via `Updater.Run` per existing convention (no new
    export hooks unless a test cannot reach a seam).
- **U2 — spec: `ImportsFileRel` + derivation.** `GuidanceTargetRel` stays
  `.pi/agent/guidance` (unchanged — D1). RED: spec test asserting pi's probe root,
  skills target, guidance target, and `ImportsFileRel: .pi/agent/AGENTS.md`; plus a
  derivation test (spec → expected prefixes) for all three harnesses.
- **U3 — per-harness report rendering.** Real design work, not a tweak: current
  `writeGuidanceHints` + `claudeGuidanceFiles` (internal/cli/update.go:303-322,
  149-157) hardcode Claude paths and filter to HarnessClaude. GREEN target: hints and
  wiring status rendered per harness from spec-derived paths (Claude keeps current
  wording; pi gains the AGENTS.md equivalent; OpenCode: no detection, no hint). RED:
  cli tests for pi hint/wired lines + existing Claude assertions updated to the
  per-harness shape (consumers at internal/cli/update.go:308,324 and both test files).
- **U4 — functional verification + this-machine migration.** Build the branch binary.
  `--dry-run`: pi listed with correct targets; per-harness import status matches live
  AGENTS.md. Live run: `ping/` survives; pi skills synced; `~/.pi/agent/guidance/`
  synced to exactly the 3 source files. MANUAL migration (this machine, D5): remove
  the dangling write-memory import line from AGENTS.md; delete stale
  `~/.pi/agent/engram/`; re-run `engram update` → pi reports fully wired, no hints.
  Subprocess invocations budgeted ≥120s each (note 364: subprocess cost counts
  against test/step timeouts).
- **U5 — integrate.** Rebase onto main if moved; full suite + vet; merge --ff-only;
  push; verify `git log origin/main` matches local and `git worktree list` no longer
  shows engram-pi-worktree; delete branch.

## Doc-surface dispositions (rev 2 — verified by Gate A docs reviewer + author grep)

| File | Disposition | Exact change |
| --- | --- | --- |
| README.md :10 | UPDATE | harness sentence → "Claude Code, OpenCode, and Pi agents" |
| README.md :12 | UPDATE | asymmetry paragraph: pi ingest shipped 2026-07-24 (pi-sessions-support) and pi update-support ships here; OpenCode SQLite gap (#644) remains. Scope note: this restores truth about shipped reality — without it the README misdescribes the feature this branch ships. |
| README.md :32, :40, :108 | UPDATE | add pi install targets (`~/.pi/agent/skills/`, `--with-guidance` → `~/.pi/agent/guidance/` + AGENTS.md import wording) |
| docs/GLOSSARY.md :108-111 | UPDATE | "supports two: Claude Code and OpenCode" → three, adding Pi |
| agent-instructions/skills/route/SKILL.md :96-98 | UPDATE | "Current harness: Claude Code exposes…" → harness-plural wording |
| docs/ROADMAP.md | UPDATE | pi-integration row: mark update-support shipped (this branch) alongside the pi-sessions ingest row (shipped 2026-07-24) |
| docs/architecture/c1-system-context.md :17 (mermaid), :47 | UPDATE | add Pi to harness enumeration + `~/.pi/agent/` install path |
| docs/architecture/c2-containers.md, c3-components.md | UPDATE | harness-deployment references gain Pi (hand-authored mermaid — note 171) |
| docs/FEATURES.md | KEEP | verified: no harness enumeration |
| docs/README.md | KEEP | verified: index page, no enumeration |
| docs/GLOSSARY other entries, memory-invariants.md, memory-system-rigor.md | KEEP | generic harness language |
| docs/design/2026-07-01-…subprocess-design.md | KEEP | historical record |
| docs/architecture/adr.md | KEEP | ADR-line already contemplates "adding a third harness… same seam"; no new decision class introduced (D3 is implementation, not architecture change) |
| agent-instructions/guidance/*.md | KEEP | harness-generic source content |

## Gate pass criteria

- Gate A: all four reviewers ACK this rev.
- Gate B (per refactored unit): design-fit reviewer finds no DRY/SRP/YAGNI violation;
  baseline suite green including the previously-failing fence test.
- Gate C: every UPDATE row above reviewed (relevance + clarity); KEEP rows spot-checked.
- Gate D: commit messages follow conventional commits + repo trailer (`AI-Used: [claude]`).
- U4 completion bar: all listed live-run outcomes observed on this machine, including
  post-migration re-run showing pi wired with zero hints.

## Out of scope

- OpenCode SQLite ingest (#644).
- Tool-driven editing of user config files (D5).
- Promoting write-memory.md to guidance in any harness (D2 decides the opposite).
