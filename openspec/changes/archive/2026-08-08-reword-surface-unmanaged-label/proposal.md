## Why

`engram update`'s first-sync migration report labels every real (non-symlink) entry found in a harness surface directory that doesn't match the intended deploy set as `stale artifact (not deleted): <path>`. This field (`SurfaceUnattributable`) is populated purely by "matches no intended-set basename" — it cannot distinguish an actual engram-stray from the user's own unrelated skills, commands, or guidance files. On a live first-sync report (2026-07-28), this surfaced the user's own skills (`c4`, `dev`, `mycelium`, `property-rigor`, `.obsidian`) labeled as if they were engram cruft needing cleanup, which is misleading and mildly alarming. See issue #716.

## What Changes

- Reword the rendered report line from `stale artifact (not deleted): <path>` to neutral wording that doesn't imply the entry is engram debris (e.g. `unmanaged (left alone): <path>`).
- Rename the `HarnessReport.SurfaceUnattributable` field (and its populating "strays" local-variable naming in `internal/update/update.go`) to a neutral name (e.g. `SurfaceUnmanaged`), and rewrite its doc comment to describe what the field actually is — an unmanaged surface entry the sync engine has no ownership signal for — rather than implying it's a stray requiring judgment.
- Update the `update-deploy-sync` spec's "Unattributable stray is reported, not deleted" scenario wording to match the neutral framing.
- No behavior change: the set of paths reported and the "never deleted, left in place" semantics are unchanged. This is a wording/naming-only change.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `update-deploy-sync`: the scenario describing surface entries outside the intended deploy set (currently "Unattributable stray is reported, not deleted") is reworded to remove judgmental "stale artifact"/"stray" language, since the mechanism cannot actually attribute ownership.

## Impact

- `internal/update/update.go`: rename `HarnessReport.SurfaceUnattributable` field and doc comment; rename local `strays` variable near its population site (~line 1234).
- `internal/cli/update.go`: reword the rendered report line (~line 568-570).
- `internal/update/update_test.go` and `internal/cli/update_test.go`: update test expectations referencing the old field name and string (~line 641, ~779).
- `openspec/specs/update-deploy-sync/spec.md`: reword the affected scenario via delta spec.
- No other call sites reference this field or string.
