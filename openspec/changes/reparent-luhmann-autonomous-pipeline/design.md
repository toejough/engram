## Context

Confirmed by reading the code: `RunPrune` (`internal/cli/prune.go`) and `RunIngest`
(`internal/cli/ingest.go`) are both plain in-process Go functions (`func RunPrune(ctx,
PruneArgs, PruneDeps, io.Writer) error`, `func RunIngest(ctx, IngestArgs, IngestDeps,
io.Writer) error`) — not subprocess-shelled CLI invocations. Each independently acquires and
releases the same manifest lock (`acquireOptionalLock(deps.Lock, chunksDir)`) around its own
read-modify-write, per #660's concurrent-write-safety design. Called sequentially (not nested)
from apply, there is no lock contention: each call acquires, does its work, releases, before the
next call acquires. This makes "apply also runs ingest then prune" a straightforward composition
of existing, already-tested primitives — not new manifest/locking logic.

`newIngestDeps`/`newPruneDeps` (`internal/cli/ingest.go`/`prune.go`) already show the pattern for
composing each function's deps from the shared `cli.Deps` — `RunReparentLuhmann`'s composition
needs to build both `IngestDeps` and `PruneDeps` alongside its existing `RenameRewriteDeps`.

## Goals / Non-Goals

**Goals:**
- Apply (`--reparent-luhmann --answers <file>`, no `--dry-run`) performs the ENTIRE mechanical
  pipeline: rename + rewrite backlinks (unchanged) → re-index renamed notes (`RunIngest`) →
  detach the old paths' stale manifest entries (`RunPrune`, plain mode, not `--duplicates`) — in
  one command invocation, no follow-up command required.
- Derive's output explicitly states the next command to run (with the answers-file path as a
  placeholder the agent fills in), not just an implicit expectation the agent infers from
  `learn/SKILL.md`.
- Apply's output, after the mechanical pipeline, explicitly states whether to loop (run derive
  again — more candidates may exist among notes that weren't candidates this round, e.g. a note
  that stayed `top` this round could gain new above-floor neighbors after other notes moved) or
  that the vault is done.
- `--dry-run` remains available but optional — not a required step in the minimal loop.

**Non-Goals:**
- Determining WHEN to stop looping via anything other than "derive finds zero candidates" — no
  new heuristic for "vault is sufficiently branched"; the existing candidate-derivation logic
  (unchanged: top-3 neighbors above the 0.75 floor) is the sole termination signal.
- Removing `--dry-run` — an operator who wants a preview before an automatic pipeline run still
  gets one; it just isn't mandatory.
- Changing `RunPrune`/`RunIngest` themselves — reused via their existing public signatures,
  unmodified.
- Making `engram update` itself (the routine, no-flag path) run the reparent pipeline
  automatically — the notice still requires the explicit opt-in flag; only what happens WITHIN
  an already-invoked `--reparent-luhmann --answers` apply changes.

## Decisions

**Decision 1 — Call order within apply's mechanical pipeline: rename+rewrite → `RunIngest` →
`RunPrune`.** Index the new paths' content before detaching the old paths' manifest entries, so
there's never a window where a renamed note's content is unindexed anywhere. Alternative
considered: prune-then-ingest — rejected, no benefit, and briefly leaves the note's content
entirely unindexed between steps.

**Decision 2 — A failure in `RunIngest` or `RunPrune` does not roll back the rename.** The vault
mutation (rename + rewrite) already succeeded and is the authoritative state; the chunk index is
a derived cache that can always be rebuilt by a later `engram ingest --auto`/`engram prune`.
Apply reports the pipeline failure clearly (which stage failed, exact error) but does not attempt
transactional rollback of the rename — matches the existing precedent that `engram update`
notices already treat the chunk index as separately, safely reconcilable state.

**Decision 3 — "More candidates remain?" check reuses the existing derive candidate-generation
function directly (in-process), not a shelled-out re-invocation.** After the pipeline completes,
apply calls the same top-K/similarity-floor candidate function again against the now-current
vault state and reports its count in the final message ("N further candidates found — run
`engram update --reparent-luhmann` again" / "no further candidates — vault fully evaluated").
This is a read-only check (never emits or requires new answers) — it does not print the
candidate payload itself, just the count and the next-step instruction; the agent gets the full
payload by actually re-running derive.

**Decision 4 — Next-command text is literal and copy-pasteable**, including the actual
`--answers <path>` the agent should use (apply may suggest a default path, e.g. alongside the
derive output or a fixed scratch location — decide during implementation; the point is the agent
never has to invent the flag shape from memory of `learn/SKILL.md`, `engram`'s own output is
sufficient).

## Risks / Trade-offs

- **[Risk] Folding three operations (rename/rewrite, ingest, prune) into one apply call means a
  partial failure leaves the vault renamed but the chunk index only partly reconciled.** →
  Mitigation: per Decision 2, this is an accepted, recoverable state — `engram ingest --auto` and
  `engram prune` remain safe to run manually at any time and will finish reconciling; apply's
  error output should say this explicitly so an agent hitting a mid-pipeline failure knows the
  vault itself is fine and what manual fallback exists.
- **[Risk] Auto-running `RunIngest` inside `--reparent-luhmann` couples two previously-independent
  commands more tightly — a future change to `RunIngest`'s behavior/signature now has a second
  caller to consider.** → Mitigation: both are stable, already-public, already-tested entry
  points; this is normal in-process composition, not a new coupling risk beyond what
  `newIngestDeps`/`newPruneDeps` already represent as shared infrastructure.
- **[Risk] The "loop again?" check (Decision 3) could recommend re-running derive forever if the
  similarity floor keeps surfacing marginal candidates from notes intentionally left at `top`.**
  → Mitigation: out of scope per Non-Goals — the agent (or user) can stop the loop at their own
  judgment; `engram` only reports the candidate count, it never forces another round.

## Migration Plan

No data migration. Purely additive behavior inside an already-unreleased-to-real-users command
(`--reparent-luhmann` shipped this same work cycle, #724 was filed from its own verification
step, not a field report). No backward-compatibility shim needed.
