# #650 — Clean up flat-layout migration vestiges in internal/cli/cli.go

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans. Checkbox steps.

**Goal:** Remove three flat-layout-migration vestiges in `internal/cli/cli.go` — behavior-preserving — so the vault-traversal helpers share one flat-root idiom and `pathOf` takes only `basename`.

**Architecture:** Pure cleanup, no behavior change. `ListIDs` keeps a single-element `for…range []string{"."}` loop and a stale doc comment from the retired Permanent/+MOCs/ layout; `pathOf` carries a vestigial ignored `isMOC bool` param passed by 9 call-sites. Flatten `ListIDs` to mirror its sibling `ListBasenames` (reads the vault root directly), and drop `pathOf`'s param.

**Tech stack:** Go; `targ` for all build/test/check; tests `gomega`. Behavior guarded by existing `ListIDs` tests + compiler-enforced call-site updates.

---

## Verified facts (against current source)

- `ListIDs` doc (`cli.go:108`): "from filenames in vault/Permanent and vault/MOCs" — STALE.
- `ListIDs` body (`cli.go:110-136`): `out := []string{}` then `for _, sub := range []string{"."}` reading `filepath.Join(vault, sub)` with `continue` on IsNotExist and `fmt.Errorf("read %s: %w", sub, err)`.
- Sibling `ListBasenames` (`cli.go:79-106`): `out := []string{}` then a bare `{ … }` block reading `os.ReadDir(vault)` directly, `return out, nil` on IsNotExist, `fmt.Errorf("read vault root: %w", err)`. **This is the idiom to mirror.**
- `pathOf` (`cli.go:261-264`): `func pathOf(basename string, _ bool) string { return basename + ".md" }`; doc says the `isMOC` param is vestigial.
- **9** `pathOf` call-sites (issue says 8): `embed.go:67,268`, `migrate.go:46`, `query.go:661,1016,1073,1277,1824`, `resituate.go:93`, `show.go:49`. All pass `note.IsMOC`/`hit.note.IsMOC`.
- `IsMOC` field (`vaultgraph/scanner.go:15`) stays CONSUMED by `check.go:123,175` — dropping `pathOf`'s param does NOT orphan it. No cascade. (The deeper question of whether the whole `IsMOC` field is post-flat-layout vestigial is OUT OF SCOPE for #650.)
- Existing `ListIDs` tests fully guard the flatten: `TestOsLearnFS_ListIDs_ReturnsRootNotesOnly` (adapters_test.go:85, creates a MOCs/ subdir to prove root-only), `_BadVaultReturnsError` (learn_adapters_test.go:93, file-as-vault → error; asserts only `HaveOccurred`, not the message), `_MissingSubdirsTolerated` (learn_adapters_test.go:106, empty vault → empty+nil). None assert the error string, so the message change is safe.
- Two test comments become factually wrong after item 2 (they describe the old `vault/Permanent` path): `_BadVaultReturnsError:97` and `_MissingSubdirsTolerated:110-111`.

## TDD note (behavior-preserving refactor)

No NEW failing test: the three `ListIDs` tests are the regression guard (they would fail if the flatten changed behavior), and the `pathOf` signature change is compiler-enforced across all 9 call-sites. The discipline is: confirm the suite is green BEFORE, make the change, confirm green AFTER + `targ check-full`. (`pathOf` is `return basename + ".md"` — a dedicated test is YAGNI; call-site tests + compiler cover it.)

---

### Task 1: The three cleanups (one atomic change)

**Files:** Modify `internal/cli/cli.go` (ListIDs doc + body, pathOf doc + signature); 9 call-sites in `embed.go`, `migrate.go`, `query.go`, `resituate.go`, `show.go`; 2 stale test comments in `learn_adapters_test.go`.

- [ ] **Step 1: Confirm the suite is green before any change**

Run: `targ test`
Expected: PASS (establishes the behavior baseline the refactor must preserve).

- [ ] **Step 2: Item 1 — fix the ListIDs doc comment**

`cli.go:108`, replace:
```go
// ListIDs returns Luhmann IDs from filenames in vault/Permanent and vault/MOCs.
```
with:
```go
// ListIDs returns Luhmann IDs from .md filenames at the vault root (flat layout).
```

- [ ] **Step 3: Item 2 — flatten the ListIDs loop to mirror ListBasenames**

`cli.go:110-136`, replace the whole body between the signature line and the final `return out, nil` (the `out := []string{}` … closing `}` of the `for _, sub` loop) with the ListBasenames-mirroring form:
```go
	out := []string{}

	{
		entries, err := os.ReadDir(vault)
		if err != nil {
			if os.IsNotExist(err) {
				return out, nil
			}

			return nil, fmt.Errorf("read vault root: %w", err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			id, ok := extractLuhmannFromFilename(e.Name())
			if !ok {
				continue
			}

			out = append(out, id)
		}
	}

	return out, nil
```
Behavior identical: `filepath.Join(vault, ".")` == `vault`; `continue`-then-exit and `return out, nil` both yield empty+nil on a missing vault; MOCs/ (a dir) is skipped by `e.IsDir()` exactly as before (the old code never descended into it either). Only the error string changes ("read .:" → "read vault root:"), which no test asserts.

- [ ] **Step 4: Item 2 (cont.) — fix the two now-false test comments**

`learn_adapters_test.go:97`, replace `// vault is a file, not a dir; ReadDir on vault/Permanent → ENOTDIR (not IsNotExist).` with `// vault is a file, not a dir; ReadDir on the vault root → ENOTDIR (not IsNotExist).`

`learn_adapters_test.go:110`, replace `// vault exists but neither Permanent nor MOCs subdirs.` with `// vault exists but is empty (flat layout — no notes).`

(Leave the test FUNCTION names unchanged — renaming is churn beyond #650's scope; the comments were the factually-wrong part.)

- [ ] **Step 5: Item 3 — drop pathOf's vestigial param + fix its doc**

`cli.go:261-264`, replace:
```go
// pathOf returns the vault-relative path for a note, e.g. "foo.md". The vault
// is flat — notes live at the root (Permanent/ and MOCs/ are retired); the
// isMOC parameter is vestigial and ignored.
func pathOf(basename string, _ bool) string {
	return basename + ".md"
}
```
with:
```go
// pathOf returns the vault-relative path for a note, e.g. "foo.md". The vault
// is flat — notes live at the root (Permanent/ and MOCs/ are retired).
func pathOf(basename string) string {
	return basename + ".md"
}
```

- [ ] **Step 6: Item 3 (cont.) — update all 9 call-sites**

Drop the second argument at each. Exact edits:
- `embed.go:67`  `pathOf(note.Basename, note.IsMOC)` → `pathOf(note.Basename)`
- `embed.go:268` `pathOf(note.Basename, note.IsMOC)` → `pathOf(note.Basename)`
- `migrate.go:46` `pathOf(note.Basename, note.IsMOC)` → `pathOf(note.Basename)`
- `query.go:661` `pathOf(name, hit.note.IsMOC)` → `pathOf(name)`
- `query.go:1016` `pathOf(hit.note.Basename, hit.note.IsMOC)` → `pathOf(hit.note.Basename)`
- `query.go:1073` `pathOf(hit.note.Basename, hit.note.IsMOC)` → `pathOf(hit.note.Basename)`
- `query.go:1277` `pathOf(note.Basename, note.IsMOC)` → `pathOf(note.Basename)`
- `query.go:1824` `pathOf(hit.note.Basename, hit.note.IsMOC)` → `pathOf(hit.note.Basename)`
- `resituate.go:93` `pathOf(note.Basename, note.IsMOC)` → `pathOf(note.Basename)`
- `show.go:49` `pathOf(note.Basename, note.IsMOC)` → `pathOf(note.Basename)`

- [ ] **Step 7: Verify tests + full check**

Run: `targ test` → PASS (existing ListIDs tests still green = behavior preserved; call-sites compile).
Run: `targ check-full` → PASS:8 (lint-full, deadcode, nilaway, coverage, check-uncommitted-aware). Confirm `IsMOC` is NOT reported dead (check.go still uses it).

- [ ] **Step 8: Real-binary verification (note 54 — verify with the real installed binary)**

`pathOf` resolves note paths; exercise it via the real binary, not just tests:
Run: `go install ./cmd/engram` then from a non-vault cwd: `engram query --phrase "memory" --limit 3` (a path-resolving read) and confirm it returns note items with resolvable `path:` values (i.e. pathOf still maps basename→`.md`). No panic, exit 0.

- [ ] **Step 9: Commit** (message drafted in Step 6/Gate D of the please workflow; trailer `AI-Used: [claude]`).

## Self-Review

1. **Coverage:** Goal's 3 items → Steps 2 (item 1), 3-4 (item 2 + fallout), 5-6 (item 3). All covered.
2. **Placeholders:** none — every edit shown verbatim with exact locations.
3. **Type consistency:** `pathOf(basename string) string` used at all 9 call-sites; `IsMOC` field untouched (still read by check.go); `extractLuhmannFromFilename`, `os.ReadDir`, `fmt.Errorf` already imported/used in cli.go.
