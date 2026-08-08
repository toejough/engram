## Context

`internal/update/update.go`'s first-sync migration path (`update-deploy-sync` capability) reports every real (non-symlink) entry in a harness surface directory that isn't part of the intended deploy set. This is populated into `HarnessReport.SurfaceUnattributable` and rendered by `internal/cli/update.go` as `stale artifact (not deleted): <path>`. The mechanism has no way to attribute ownership — it's a pure "not in our intended set" match — so it fires equally for genuine engram leftovers and for the user's own unrelated skills/commands/guidance files. The wording implies a judgment ("stale," "artifact") the code cannot actually make.

## Goals / Non-Goals

**Goals:**
- Rename the field, its populating local variable, and its doc comment so the code's own naming stops implying an ownership judgment it can't make.
- Reword the rendered report line to neutral, accurate language.
- Reword the corresponding spec scenario so the spec doesn't reintroduce the same judgmental framing for a future reader.

**Non-Goals:**
- No heuristic to actually distinguish engram-shaped strays from user files (considered and explicitly deferred during exploration — larger scope, separate issue if wanted later).
- No behavior change to what gets reported, when, or the never-delete guarantee.

## Decisions

- **Field rename**: `HarnessReport.SurfaceUnattributable` → `HarnessReport.SurfaceUnmanaged`. "Unmanaged" accurately describes the state (engram isn't managing this path) without claiming it's stale or foreign.
- **Rendered label**: `stale artifact (not deleted): %s` → `unmanaged (left alone): %s`. Keeps the parenthetical clarifying no deletion happened, drops "artifact" (implies engram provenance) and "stale" (implies it should be cleaned up).
- **Local variable**: rename `strays` (used near `internal/update/update.go:1234` when building the slice) to `unmanaged` for consistency with the field name.
- **Spec scenario**: rename "Unattributable stray is reported, not deleted" → "Unmanaged surface entry is reported, not deleted"; reword the THEN clause to drop "possible stale artifact."

Alternatives considered:
- *String-only reword, leave field name as-is*: rejected — the issue's chosen option (B) was picking this broader rename specifically so the judgmental language doesn't survive in code for a future call site to copy.
- *Heuristic engram-vs-user classification*: rejected as out of scope — no reliable ownership signal exists (SurfaceUnattributable is deliberately populated for anything outside the intended set, per its own doc comment), and building one is a materially larger, separate effort.

## Risks / Trade-offs

- [Risk] Renaming a report-facing string could be seen as a breaking change to anything scraping `engram update` output. → Mitigation: no known consumers parse this text (it's a human-readable notice); test updates confirm intended new wording.
- [Risk] Missing a call site during rename causes a compile error, not silent drift, given Go's static typing. → Mitigation: `grep -rn SurfaceUnattributable` before considering done; confirmed only two production call sites plus tests exist.

## Migration Plan

Single-PR rename + reword, no phased rollout needed:
1. Rename field + local var + doc comment in `internal/update/update.go`.
2. Reword rendered line in `internal/cli/update.go`.
3. Update test expectations in `internal/update/update_test.go` and `internal/cli/update_test.go`.
4. Update `openspec/specs/update-deploy-sync/spec.md` scenario via delta spec at archive time.

No user-facing migration or rollback concerns — pure rename/reword, no data format change.

## Open Questions

None.
