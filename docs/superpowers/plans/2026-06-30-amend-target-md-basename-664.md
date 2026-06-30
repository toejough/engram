# `engram amend`/`resituate` `--target` accepts the `.md` basename (#664) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (or executing-plans).
> Steps use `- [ ]`. Gate B/C/D markers are run by the `/please` orchestrator. Single shared-resolver Go fix + tests.

**Goal:** Make `engram amend --target` resolve the full `.md` basename (and `[[wikilink]]` forms) the same way
`engram show` does, so the recall skill's Step-2.5C/2.6 amends — which pass the query payload's `.md` path as
`--target` — stop silently failing with "amend: note not found" (issue #664).

**Architecture (revised per Gate A — fix the shared resolver, not one call site):** `findNote`
(`internal/cli/resituate.go:90`) is the **shared** note-ref resolver used by both `RunAmend` (`amend.go:73`,
`args.Target`) and `RunResituate` (`resituate.go:48`, `args.Note`). It matches `note.LuhmannID == target ||
note.Basename == target` with **no normalization**, and `Basename` is `.md`-stripped (`vaultgraph` `ParseBasename`),
so any `.md`-suffixed or `[[wikilink]]` target never matches → "note not found". `RunShow` already normalizes via
`normalizeShowRef` (`show.go:82-94`: strips `[[ ]]`, `|display`, trailing `.md`). **The fix normalizes inside
`findNote`**, using a renamed-shared `normalizeNoteRef` (was `normalizeShowRef`). This (a) fixes `amend` *and*
`resituate` in one place — no per-caller patch, no inconsistency; (b) makes the rename genuinely justified (the
normalizer is now shared by `show`'s path and `findNote`); (c) leaves both call sites unchanged.

**Tech Stack:** Go (`internal/cli/`); gomega unit tests with the **plain inline-closure** `…Deps` pattern the
existing `amend_test.go` uses (NOT imptest — that file has no imptest); `targ`; verify the real binary via
`go install ./cmd/engram` (there is **no** `targ build`).

## Global Constraints
- `targ` for all Go test/lint (`targ test`, `targ check-full`) — NEVER `go test`/`go vet`. The one exception is
  binary install: `go install ./cmd/engram` (targ has no build/install target).
- nilaway/gomega guards per `.claude/rules/go.md` (nil-check after `Expect(err).NotTo(HaveOccurred())`); line
  length < 120; descriptive names; `t.Parallel()` with no shared mutable state.
- No SKILL.md edit — the resolver fix makes the recall skill's existing "use the payload basename" instruction
  correct (Iron Law: don't author against a passing baseline). Confirmed by Gate-A docs review.
- Commit trailer `AI-Used: [claude]`.

---

## Task 1: Normalize inside the shared `findNote` resolver (fixes amend + resituate)

**Files:** Modify `internal/cli/resituate.go` (findNote + the `--note` flag desc), `internal/cli/show.go`
(rename), `internal/cli/amend.go` (the `--target` flag desc only); Test `internal/cli/amend_test.go` +
`internal/cli/resituate_test.go` (whichever holds the findNote-level tests; else add to `amend_test.go`).

- [ ] **Step 1 — RED test.** Add `TestFindNote_ResolvesMdAndWikilinkRefs` (place it where `findNote` is tested;
  it's package-internal so a `package cli` test can call it directly). Seed
  `notes := []vaultgraph.Note{{Basename: "1.linting", LuhmannID: "1"}}` and assert all three resolve to the same
  note: `findNote(notes, "1.linting.md")`, `findNote(notes, "1.linting")`, `findNote(notes, "1")` — and a
  `[[1.linting]]` form. (Today `"1.linting.md"` and the wikilink form error.) ALSO add a user-facing
  `TestRunAmend_ResolvesTargetWithMdSuffix` in `amend_test.go` following the existing **plain-closure** idiom
  (seed the `Scan` closure to return that note, `Read`/`Write` closures as the other tests do; call `RunAmend`
  with `Target: "1.linting.md"` and one `--relation` to an existing seeded note; assert no error + `Write` was
  called for the resolved note).
- [ ] **Step 2 — Run RED, expect FAIL.** `targ test` → the `.md`/wikilink cases fail (`findNote` returns
  not-found; `RunAmend` returns `amend: note not found`).
- [ ] **Step 3 — GREEN.**
  (a) In `show.go`, rename `normalizeShowRef` → `normalizeNoteRef` (update the doc comment to "a general note-ref
  normalizer used by `show` and `findNote`"); update its single caller at `show.go:34`.
  (b) In `findNote` (`resituate.go:90`), normalize the incoming `target` as the first line:
  `target = normalizeNoteRef(target)` (then the existing `LuhmannID`/`Basename` match runs against the
  normalized ref). Both callers (`amend.go:73`, `resituate.go:48`) are unchanged.
  (c) Update the flag descriptions to list the accepted forms, mirroring `show.go:17`
  (`"note ref: full basename | [[wikilink]] | trailing .md | or bare Luhmann id"`): `--target` at `amend.go:24`
  and the `--note` flag in `resituate.go`.
- [ ] **Step 4 — Run GREEN, expect PASS.** `targ test` → new tests + full suite pass.
- [ ] **Step 5 — REFACTOR + Gate B.** Confirm DRY: ONE `normalizeNoteRef` (callers: `show.go:34` and inside
  `findNote`); the `.md`-dedup `TrimSuffix(".md")` calls at `amend.go` **lines 296 and 307** are a *separate*
  concern (relation-bullet dedup in `mergeRelatedSection`) and stay untouched. Verify `findNote` has no other
  callers beyond `amend.go:73` + `resituate.go:48` (grep) so the normalization is safe for all. Hand the diff to Gate B.
- [ ] **Step 6 — Verify with the real binary (throwaway vault — do NOT touch the live vault).**
```bash
go install ./cmd/engram   # NOT 'targ build' (no such target)
V="$(mktemp -d)/vault"; mkdir -p "$V"
printf -- '---\ntype: fact\n---\nlint before commit\n' > "$V/1.linting.md"
printf -- '---\ntype: fact\n---\nfoo\n'                > "$V/2.foo.md"
ENGRAM_VAULT_PATH="$V" engram amend --target "1.linting.md" --relation "2.foo|test"   # expect: success, no "note not found"
ENGRAM_VAULT_PATH="$V" engram amend --target "1"          --relation "2.foo|test"     # expect: same note resolves
```
- [ ] **Step 7 — `targ check-full`** green (lint + coverage + nils + uncommitted).
- [ ] **Step 8 — Commit:**
```bash
git add internal/cli/resituate.go internal/cli/show.go internal/cli/amend.go internal/cli/*_test.go
git commit -m "$(printf 'fix(cli): resolve note refs with .md / wikilink forms in findNote (#664)\n\nfindNote (shared by amend --target and resituate --note) matched\nLuhmannID/Basename with no normalization, so a .md-suffixed or\n[[wikilink]] ref never matched the .md-stripped basenames. Normalize\ninside findNote via the renamed-shared normalizeNoteRef (was\nnormalizeShowRef), fixing amend + resituate consistently with show.\n\nCloses #664.\n\nAI-Used: [claude]')"
```

## Task 2: Doc tighten + close-out

- [ ] **Step 1 — Doc sweep (Gate C).** Per Gate-A docs review: no doc claims "bare id only", so nothing is
  *stale*, but two are under-specified now that the forms are accepted — tighten both to match `show`'s form list:
  - `README.md` (~line 87): `--target <id|basename>` → list `full basename | [[wikilink]] | trailing .md | bare Luhmann id`.
  - `docs/GLOSSARY.md` (~lines 250-254): tighten the `--target` mention similarly if it pins a form.
  Confirm the recall `SKILL.md` Step-2.6 instruction ("use each note's basename exactly as in the payload") is
  now *correct* (not contradicted) — no edit. Update only what's genuinely improved.
- [ ] **Step 2 — Commit** the doc changes (Gate C over them).
- [ ] **Step 3 — Comment + close #664** (Gate D over the prose): root cause (`findNote` didn't normalize like
  `show`), the fix (normalize in the shared resolver via `normalizeNoteRef`), and that it covers **both** `amend`
  and `resituate` (no separate resituate issue needed). `AI-Used: [claude]`.
- [ ] **Step 4 — Push** `git push origin main` (main is synced; keep it so).
- [ ] **Step 5 — Delete** the plan + any temp vault/files.

## Self-review (writing-plans checklist)
- **Coverage:** the shared-resolver fix = Task 1; doc tighten + close = Task 2. Ask (`amend --target` accepts
  `.md`) = Task 1 Steps 1-4; real-binary verify = Step 6.
- **Scope (Gate-A-adjusted):** fixing `findNote` is the *root* fix (minimal, one place) and covers `resituate`
  for free — recorded as the deliberate choice over patching amend's call site + a follow-up issue.
- **DRY:** one `normalizeNoteRef`, used by `show` + `findNote`; the amend.go:296/307 dedup is a different concern.
- **Test idiom:** plain inline closures (the existing `amend_test.go` pattern), not imptest.
