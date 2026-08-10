## Why

Issue #724 surfaced that `--reparent-luhmann` apply leaves a stale chunk-manifest entry behind,
needing a manual `engram prune`. The originally-planned fix (`reparent-luhmann-prune-hint`, not
yet implemented — superseded by this change) was to print a hint naming the follow-up command.
Joe's direction is stronger: **the only step that should ever require a human or agent judgment
call is the disposition decision itself** (is note B a sub-point of note A?). Everything
mechanical — renaming, rewriting backlinks, detaching the stale manifest entry, re-indexing —
should happen automatically inside `apply`, not be left as a separate command the operator has
to know to run. And every phase that DOES require a next action (derive → judge → apply, or
apply → judge again if more candidates remain) should print the literal next command to run, so
an agent can drive the whole loop from `engram update`'s first notice to a fully re-evaluated
vault without the user typing anything after the initial ask.

## What Changes

- **`RunReparentLuhmann`'s apply phase becomes the full mechanical pipeline**: after renaming
  notes and rewriting backlinks (`RenameAndRewriteReferences`, unchanged), apply now also calls
  `RunPrune` (stale-source detach, in-process — `internal/cli/prune.go`'s `RunPrune` is already a
  plain Go function taking `PruneArgs`/`PruneDeps`, not a subprocess) and `RunIngest` (re-index
  the renamed notes, in-process — `RunIngest` similarly) as part of the same command invocation.
  The operator/agent never runs `engram prune` or `engram ingest --auto` separately for this
  purpose.
- **Every phase prints the literal next command, not just a description.** The flat-vault notice
  (`engram update`) already names `engram update --reparent-luhmann` — kept. Derive's output
  gains an explicit "next: write the answers file, then run `engram update --reparent-luhmann
  --answers <path>`" line. Apply's output, after the mechanical pipeline completes, reports
  whether further candidates remain (some top-level notes may only surface as candidates once
  earlier ones are placed, or the top-K cap may have left some out) and if so prints "next: run
  `engram update --reparent-luhmann` again"; if none remain, it reports done.
- **`--dry-run` becomes optional inspection, not a required step in the loop.** Apply's own
  fingerprint-gating (unchanged) is the safety net; an agent driving the autonomous loop is not
  required to preview before applying. `--dry-run` still exists for an operator who wants to
  look before committing.
- **BREAKING (behavior change on an unreleased-to-users feature)**: `--reparent-luhmann --answers
  <file>` (no `--dry-run`) now mutates the chunk index (via `RunPrune`/`RunIngest`) in addition to
  the vault, where it previously only touched vault notes. Since `--reparent-luhmann` shipped in
  the same work cycle and #724 was filed before any real-world use, this is not expected to break
  an existing workflow — flagged for completeness per repo convention.

## Capabilities

### Modified Capabilities
- `update-reparent-luhmann-batch`: apply phase requirement changes from "renames notes and
  rewrites backlinks, leaving the chunk index for a separate manual step" to "renames, rewrites,
  detaches the stale chunk-manifest entry, and re-indexes — a single mechanical pipeline"; derive
  and apply phase requirements both gain an explicit next-command output; the standalone
  `engram prune`-hint requirement from the (unimplemented, now-superseded) prior proposal is
  replaced by this automatic pipeline.

### New Capabilities
(none)

## Impact

- `internal/cli/luhmann_reparent_apply.go` — apply phase gains calls to `RunPrune`/`RunIngest`
  after `RenameAndRewriteReferences` succeeds; derive and apply output gain explicit
  next-command text; apply gains a post-pipeline "candidates remain?" check to decide its final
  message
- `internal/cli/prune.go` / `internal/cli/ingest.go` — reused as-is via their existing
  `RunPrune`/`RunIngest` entry points and dep-composer patterns (`newPruneDeps`, and whatever
  `RunIngest`'s deps composer is called — confirm exact name during design/implementation); no
  behavior change to either function itself
- `docs/GLOSSARY.md` — the `--reparent-luhmann` entry's "known gap (#724)" language is replaced
  with a description of the automatic mechanical pipeline
- Issue #724 closes via this change instead of the narrower hint-only fix
