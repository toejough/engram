# `engram amend --target` accepts the `.md` basename (#664) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (or executing-plans).
> Steps use `- [ ]`. Gate B/C/D markers are run by the `/please` orchestrator. Single-file Go fix + test.

**Goal:** Make `engram amend --target` resolve the full `.md` basename (and `[[wikilink]]` forms) the same way
`engram show` does, so the recall skill's Step-2.5C/2.6 amends — which pass the query payload's `.md` path as
`--target` — stop silently failing with "amend: note not found" (issue #664).

**Architecture:** `RunAmend` (`internal/cli/amend.go:73`) resolves `--target` via `findNote(notes, args.Target)`
with **no normalization**, so a `.md`-suffixed target never matches the `.md`-stripped basenames. `RunShow`
already normalizes first via `normalizeShowRef` (`show.go:79-95`: strips `[[ ]]`, `|display`, and a trailing
`.md`). The fix reuses that normalization in amend. Since it's now shared by `show` + `amend`, **rename
`normalizeShowRef` → `normalizeNoteRef`** (it's a general note-ref normalizer, not show-specific) and call it in
both.

**Tech Stack:** Go (`internal/cli/`); imptest + gomega unit tests (`internal/cli/amend_test.go`); `targ` build
system; verify with the real binary via `go install ./cmd/engram` (note 106 — there is **no** `targ build`).

## Global Constraints
- `targ` for all Go test/lint (`targ test`, `targ check-full`) — NEVER `go test`/`go vet`. Binary install is the
  one exception: `go install ./cmd/engram` (note 106).
- nilaway/gomega guards per `.claude/rules/go.md` (nil-check after `Expect(err).NotTo(HaveOccurred())`); line
  length < 120; descriptive names; `t.Parallel()` with no shared mutable state.
- No SKILL.md edit — the resolver fix makes the recall skill's existing "use the payload basename" instruction
  correct (writing-skills Iron Law: don't author against a passing baseline).
- Commit trailer `AI-Used: [claude]`.

---

## Task 1: Normalize `amend --target` like `show` (rename to a shared helper)

**Files:** Modify `internal/cli/amend.go`, `internal/cli/show.go`; Test `internal/cli/amend_test.go`.

- [ ] **Step 1 — RED test.** In `amend_test.go`, add `TestRunAmend_ResolvesTargetWithMdSuffix` (follow the
  existing amend-test pattern — imptest mocks for `Scan`/`Read`/`Write`, gomega): seed a note whose basename is
  e.g. `1.linting` (Scan returns it); call `RunAmend` with `Target: "1.linting.md"` and a no-op-ish amend (e.g.
  one `--relation` to an existing note, or a `--subject/predicate/object`); assert **no error** and that
  `deps.Write` was called for the resolved note. (Currently this errors `amend: note not found`.) Add a sibling
  assertion or sub-test that `Target: "1.linting"` (bare basename) and `Target: "<luhmann id>"` resolve the same
  note — pin the equivalence.
- [ ] **Step 2 — Run RED, expect FAIL.** `targ test` → the `.md`-suffix case fails with `amend: note not found`.
- [ ] **Step 3 — GREEN.** (a) In `show.go`, rename `normalizeShowRef` → `normalizeNoteRef` (doc comment: "a
  general note-ref normalizer used by `show` and `amend`") and update its caller in `RunShow`. (b) In
  `amend.go:73`, change `findNote(notes, args.Target)` → `findNote(notes, normalizeNoteRef(args.Target))`.
  (c) Update the `--target` flag desc (`amend.go:24`) to mention the accepted forms, mirroring show's
  (`"note ref: full basename | [[wikilink]] | trailing .md | or bare Luhmann id"`).
- [ ] **Step 4 — Run GREEN, expect PASS.** `targ test` → the new test + the full suite pass.
- [ ] **Step 5 — REFACTOR + Gate B.** Confirm the result is DRY (one `normalizeNoteRef`, two callers; no
  duplicated `.md`-stripping — note the existing dedup `TrimSuffix(".md")` at `amend.go:296/307` is a *different*
  concern (relation-bullet dedup) and stays). Reads as written-from-the-start. Hand the diff to Gate B.
- [ ] **Step 6 — Verify with the real binary.** `go install ./cmd/engram` (NOT `targ build`); then on a real
  seeded vault note, `engram amend --target "<full basename>.md" --relation "<existing>|test"` → succeeds (no
  "note not found"); `engram amend --target "<luhmann id>" ...` → same note. Revert/clean the test amend if it
  polluted a real note (use a throwaway `ENGRAM_VAULT_PATH` temp vault to avoid touching the live vault).
- [ ] **Step 7 — `targ check-full`** green (lint + coverage + nils + uncommitted).
- [ ] **Step 8 — Commit** `internal/cli/amend.go internal/cli/show.go internal/cli/amend_test.go`.

## Task 2: Doc check + close-out

- [ ] **Step 1 — Doc sweep (Gate C).** Check whether any doc pins the `--target` form as "bare id only" or
  "no `.md`": grep `docs/GLOSSARY.md`, `docs/architecture/*`, `README.md`, and the recall `SKILL.md` for
  `--target`/`amend` form claims. The recall SKILL.md Step-2.6 line ("use each note's basename exactly as in the
  payload (Luhmann-prefixed)") becomes *correct* with this fix (the payload `.md` form now resolves) — no edit
  needed, but confirm it isn't contradicted. Update only docs that are now stale.
- [ ] **Step 2 — Commit** any doc changes (Gate C over them).
- [ ] **Step 3 — Comment + close #664** (Gate D over the prose): root cause (amend didn't normalize `--target`
  like show), the fix (shared `normalizeNoteRef`), and the real impact (recall's own amends could silently fail).
  `AI-Used: [claude]` on the commit.
- [ ] **Step 4 — Push** `git push origin main` (main is synced; keep it so).
- [ ] **Step 5 — Delete** the plan + any temp vault/files.

## Self-review (writing-plans checklist)
- **Coverage:** the fix (resolver normalization) = Task 1; doc check + close = Task 2. The ask (amend accepts
  `.md`) maps to Task 1 Steps 1-4; the real-binary verify (note 106) is Step 6.
- **Scope honesty:** Go-only; no SKILL.md edit (the fix makes the existing instruction correct). The `.md`-dedup
  at amend.go:296/307 is a separate concern, untouched.
- **DRY:** one shared `normalizeNoteRef` for show + amend, not an ad-hoc re-strip.
