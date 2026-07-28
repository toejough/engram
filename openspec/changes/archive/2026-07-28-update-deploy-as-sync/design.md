# Design: update-deploy-as-sync

## Context

`engram update` (internal/update/update.go, ~1,030 lines) plans per-file `CopyOp`s from `agent-instructions/{skills,commands,guidance}` into each detected harness's directories and applies them copy-only. Deletion exists solely *within* an artifact that still exists in the source (`clearSkillDirOnce` wipes a surviving skill's target dir before re-copy; `applyCmdOne` does `RemoveAll` before write). An artifact that leaves the source generates no CopyOp and is never visited — whole-artifact removal never propagates (#706).

Three harnesses today (`supportedHarnesses`): Claude Code (`~/.claude/skills`, guidance at `~/.claude/engram`, imports in `~/.claude/CLAUDE.md`), OpenCode (`~/.config/opencode/{skills,commands}`, guidance unverified/skipped), Pi (`~/.pi/agent/{skills,guidance}`, imports in `~/.pi/agent/AGENTS.md`). Harness skill/command dirs are **shared spaces** — they also hold artifacts the user installed by other means. `engram update` keeps no record of what it wrote, so it cannot today distinguish "engram-owned, since removed" from "user's own file".

Constraints: DI-everywhere (no direct `os.*` in internal/; all I/O through the injected `Filesystem` interface, wired via `cli.Primitives`/`cli.NewDeps` and `cmd/engram`'s checker-thin functions); pure Go, no CGO; guidance deployment is opt-in (`--with-guidance` flag or pre-existing imports in the harness config file).

## Goals / Non-Goals

**Goals:**

- Removals propagate: an artifact deleted/renamed in `agent-instructions/` disappears from every install on the next `engram update`.
- Deletion is ownership-bounded by construction: engram deletes only inside roots it provably owns, plus symlinks that provably point into them.
- Harness-visible surfaces (skill dirs, command files) become symlinks into an engram-owned root wherever the harness's discovery resolves symlinks; a harness that fails that verification gets a recorded-manifest fallback instead.
- One-time migration converts previously-copied engram artifacts to symlinks without touching user-owned files.
- Preserve: the deployed artifact sets, `--with-guidance` opt-in semantics, harness detection, source layout.

**Non-Goals:**

- Windows/symlink-permission support (no Windows harness exists).
- New harnesses (Shelley #707) or OpenCode guidance enablement (#667) — this change reshapes the mechanism they will later plug into, nothing more.
- Changing which artifacts are deployed or their source-side layout.

## Decisions

### D1 — One engram-owned artifacts root per harness, beside the harness's existing dirs

Each harness gets a single engram-owned root: `~/.claude/engram/` (Claude Code), `~/.config/opencode/engram/` (OpenCode), `~/.pi/agent/engram/` (Pi), holding subtrees `skills/`, `commands/`, `guidance/` as applicable. Real files live only here; everything harness-visible outside it is a symlink in.

*Why not a global root under `$XDG_DATA_HOME/engram/`?* Keeping the root inside each harness's config tree keeps links short, survives harness-tree relocation as a unit, and for Claude Code reuses a directory engram already owns. *Claude guidance caveat:* `~/.claude/engram/*.md` is today both the guidance deploy target and referenced verbatim by user-authored `@import` lines in `CLAUDE.md`. Guidance moves to `~/.claude/engram/guidance/*.md` for layout uniformity, and migration leaves **compat symlinks** at the old flat paths (`~/.claude/engram/recall.md` → `guidance/recall.md`) so user import lines keep working; the update report tells the user the new canonical paths. Pi guidance (`~/.pi/agent/guidance/`) is a shared-space dir like skills dirs: its files become symlinks into the Pi root's `guidance/`.

### D2 — Ownership is explicit: a marker file stamps every engram-owned root

The root carries a marker (`.engram-owned`, written at creation/migration). Sync-deletion runs only inside marker-stamped roots. A root-candidate dir that exists but lacks the marker gets a first-sync adoption pass (D6); unexpected files found in a stamped root during sync are deleted (that is the point of ownership), but the update report lists them.

*Why a marker over path convention alone?* The path name (`engram/`) is suggestive but not proof; the marker makes the ownership claim inspectable and survives the "user coincidentally has a dir named engram" edge.

### D3 — Symlink granularity: per-skill-dir for skills, per-file for commands and guidance

One symlink per skill directory (`~/.claude/skills/recall` → `~/.claude/engram/skills/recall`) — matches the discovery unit (a dir containing SKILL.md) and keeps link counts low. Commands and guidance link per-file (their discovery units are files).

### D4 — Sync target is the *intended deploy set*, not the raw source tree

The existing planners (`planSkillCopies`, `planCommandCopies`, `planGuidanceCopies`) already compute the intended set per harness, including the `--with-guidance` opt-in gate. The sync engine diffs the engram-owned root against that intended set: missing → create, different → overwrite, present-but-unintended → delete. Guidance non-opt-in means **unmanaged, not removed**: when guidance is not in the intended set (no flag, no detected imports), the root's `guidance/` subtree is left untouched, preserving today's semantics.

### D5 — Dangling-link cleanup identifies engram links by logical target prefix

On every update, scan each harness surface dir (skills root, commands root, Pi guidance dir) one level deep: for each symlink, resolve its literal target against the link's parent dir (lexical join + `filepath.Clean` — no `EvalSymlinks` on the result) and prefix-match against the harness's engram-owned root path. Match + target missing → delete the link. Match + target present → healthy engram link, leave. No match → foreign symlink, never touched. Real files/dirs → never touched by cleanup.

*Why lexical, not resolved, comparison?* Resolution is for reaching bytes, never for naming them (vault note 475): on macOS, `EvalSymlinks` rewrites `/var` → `/private/var` and breaks prefix identity for trees containing no user symlinks at all. The owned root's path and the link targets we wrote are both logical paths; comparing them lexically is the identity transform.

### D6 — First-sync migration: adopt what matches the intended set, report the rest

Per harness, on the first sync (no marker present): (1) create the root + marker; (2) for every artifact in the intended set whose harness path holds a **real** file/dir (a pre-sync copy), write the artifact into the root and replace the harness path with a symlink — the repo is the source of truth, so no content comparison is needed; (3) real files at harness paths **not** in the intended set are left in place and listed in the update report as possible stale engram artifacts for manual review — engram cannot prove ownership of what it has no record of writing, so it does not delete them.

### D7 — Per-harness deploy mode, gated by symlink-discovery verification; manifest fallback

`HarnessSpec` gains a deploy mode: `symlink` (default) or `manifest`. Before a harness ships in symlink mode, its skill/command discovery through symlinks is verified against the real harness (implementation task, per harness). A harness that fails verification runs in manifest mode: real files are copied as today, and a manifest inside the engram-owned root records every written path; sync deletes recorded paths whose sources left. The sync engine, planners, and report are shared; only the materialization step (link vs copy+record) differs.

### D8 — Filesystem interface grows symlink primitives; wired per DI-everywhere

`Filesystem` gains `Symlink(target, link string) error`, `ReadLink(path string) (string, error)`, and `Lstat`-shaped type probing (enough to distinguish symlink / real file / dir without following). Adapters land in `internal/cli` (composition root) and `cmd/engram`'s checker-thin per-group functions; `targ check-thin-api` keeps them thin.

### D9 — Every sync operation flows through the dry-run prefix discipline

The reshaped apply path emits one op-classified line per planned action (`create-link`, `sync-write`, `sync-delete`, `cleanup-link`, `migrate`, …), uniformly prefixed in dry-run mode. #709 (guidance-refreshed lines print unprefixed, reading as writes that didn't happen) is a defect of the old path; the new path's output contract subsumes it and #709 is verified fixed-or-moot during implementation.

## Risks / Trade-offs

- [Harness discovery may not resolve symlinked skill dirs] → D7's verification gate runs before symlink mode ships for each harness; failure flips that harness to manifest mode, which preserves all sync semantics with weaker (recorded, not structural) ownership proof.
- [Path-identity leakage on macOS (`/var` → `/private/var`) breaking link classification or tests] → D5 compares logical paths lexically; tests include a symlink-free tree asserting recorded paths are unchanged (note 475's regression check).
- [Sync deletes something the user cared about inside a stamped root] → deletion is confined to marker-stamped roots engram created or adopted; the update report lists every deletion; migration never stamps a dir while leaving unknown files inside it unlisted (D6 reports them).
- [Compat symlinks for Claude guidance linger indefinitely] → they live inside the engram-owned root and are part of the intended set, so they remain managed; a later change can retire them once import lines are migrated.
- [`--dry-run` fidelity: preview must not create the root or marker] → dry-run renders the full op list including first-sync migration without writing; covered by scenario-level tests.
- [Marker file is user-deletable] → deleting `.engram-owned` degrades engram to refusing sync-deletion in that root (fails safe: report + re-adoption path), never to deleting more.

## Migration Plan

1. Ship the sync engine dark: first `engram update` after upgrade performs D6 migration per harness (root + marker, adopt intended-set copies to symlinks, report unknowns), then the normal sync.
2. Rollback: the engram-owned root plus symlinks are equivalent to the old copies from the harness's point of view; reverting the binary and running the old `update` re-copies real files over the symlinks (RemoveAll-then-write already in the old path), restoring the status quo ante.
3. No data migration: the vault, chunk index, and sidecars are untouched.

## Open Questions

- None blocking. Per-harness symlink-discovery verification (Claude Code, OpenCode, Pi) is deliberately an implementation task with a decision point (D7), not a design unknown — the design is valid under either verdict.
