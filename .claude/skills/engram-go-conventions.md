---
name: engram-go-conventions
description: |
  Use when writing or modifying any .go file inside engram/internal/ or engram/cmd/, before running targ check-full, when fixing repeated lint failures (gochecknoglobals, cyclop, wsl_v5, reorder-decls, coverage floor), or when a subagent is dispatched to implement a Go task in this repo.
  Triggers: implementing engram Go code, adding a new package, writing tests for engram internals, fixing engram lint errors, refactoring to satisfy targ check-full.
  Domains: go, engram, linting, code-style.
  Anti-patterns: NOT for editing markdown docs, NOT for skill files, NOT for non-engram repos.
context: inherit
model: inherit
user-invocable: false
---

# Engram Go Conventions

The engram repo runs strict lint config via `targ check-full`. Hit these patterns on the first pass and the build is green; miss them and you burn 30+ tool uses iterating.

**Read this BEFORE writing any .go file in this repo.**

## The Iron Loop

```
1. Write failing test (TDD red)
2. Write minimal implementation
3. Run `targ reorder-decls`     # auto-fix decl ordering — treat as a formatter
4. Run `targ check-full`        # ALL 8 gates at once; never use `targ check`
5. Fix every failure in one pass (don't whack-a-mole)
6. Re-run `targ check-full` → green
7. Commit immediately
```

Skip step 3 and you'll fail step 4 on `reorder-decls-check`. Use `targ check` instead of `check-full` and you'll only see the first failure, fix it, hit the next, and burn iterations.

## Mandatory Rules

### gochecknoglobals — no top-level `var` declarations

The lint rejects `var` at file scope **except** sentinel errors.

| Forbidden | Use instead |
|---|---|
| `var defaultFilenames = []string{...}` | Function-local literal: `filenames := []string{...}` |
| `var pattern = regexp.MustCompile(...)` | `const patternStr = "..."` and compile inside the function (cheap), OR `var compiled = sync.OnceValue(func() *regexp.Regexp { ... })` (function value is OK) |
| `var goosFn = func() string { return runtime.GOOS }` | Pass `goos` as a parameter to the public function |
| `var typeMap = map[string]Kind{...}` | Function-local literal, or build via switch |

**`var ErrFoo = errors.New("...")` is allowed and required** for sentinel errors (do not use inline `fmt.Errorf` for sentinels).

### cyclop max 10 — refactor complex functions

Cyclomatic complexity > 10 fails. Anticipate this for:
- Parsers with `for` + nested `switch` + multiple `if`s → extract `parserState` struct with small methods (`consumeLine`, `continueFolded`, `flushFolded`, `startKey`).
- Discovery loops covering multiple sources → extract per-source helpers (`walkAncestors`, `addUserScope`, `addManagedPolicy`).
- Pipeline loops with multiple short-circuits → extract `processOne(...)` returning early; loop body is one call.

If you write a function with 3+ nested control structures or 5+ `if`/`switch` branches, refactor before running the linter.

### wsl_v5 — blank lines around unrelated statements

Add a blank line before a `for`/`if`/`switch` that doesn't share variables with the immediately preceding statement:

```go
dir := start
                  // ← BLANK LINE REQUIRED
for {
    ...
}
```

Also: blank line between `var`/`const` blocks and following statements; blank line before `return` when separating logic from the return.

### reorder-decls-check — types after consumers

Engram convention: **declarations come AFTER the functions that use them.** Within a `.go` file, ordering is:

1. Public exported functions (alphabetical)
2. Unexported helper functions (alphabetical)
3. `// unexported constants.` const block
4. `// unexported variables.` var block (rare)
5. Unexported types

Don't fight it — write code in any order, then run `targ reorder-decls` before commit. Always.

Test files: tests alphabetized; helpers/fakes at end of file.

### check-coverage-for-fail — 80% function-coverage floor

Every function (public AND unexported helper) needs ≥80% statement coverage. If you write a `switch` with a `default` or a function with an "unrecognized input" branch, **add a test for it** with a sentinel out-of-range value:

```go
const unrecognizedKindValue = 99
g.Expect(externalsources.Kind(unrecognizedKindValue).String()).To(Equal("invalid"))
```

If a small helper has untested branches, add a one-liner test for the branch. Don't refactor to remove the helper — coverage of helpers is the right gate.

### Modernized Go — use new stdlib helpers

The linter rejects older patterns:

| Old | New |
|---|---|
| `bytes.Index(b, sep)` + slicing | `bytes.Cut(b, sep)` |
| `strings.HasPrefix(s, p)` + `strings.TrimPrefix(s, p)` | `strings.CutPrefix(s, p)` |
| `for _, v := range strings.Split(s, sep)` | `for v := range strings.SplitSeq(s, sep)` |
| `switch {}` with `case x == y:` cases | tagged `switch x {}` with `case y:` cases |
| `for i := 0; i < n; i++` | `for i := range n` |

### varnamelen — no short identifiers

Variable names ≥ 3 chars or descriptive: `frontmatter`, `discovered`, `entry`, `memory`, `pattern` — not `f`, `d`, `e`, `m`, `p`. Single-letter loop variables (`i`, `k`, `v`) are usually OK.

### Nilaway + gomega (from .claude/rules/go.md)

After `g.Expect(err).NotTo(HaveOccurred())`, add `if err != nil { return }` **only when accessing values** afterward. With `_, err := ...` (value discarded), the guard isn't required.

Use `g.Expect(err).To(MatchError(...))` not `err.Error()`.

### Nilaway + slice returns

**Functions returning slices should return `[]T{}` (empty slice), not `nil`.** Nilaway traces literal `nil` returns through callers and flags downstream slice indexing (e.g., `files[0].Path` after `g.Expect(files).To(HaveLen(1))`) as "potential nil panic," even when the test guarantees non-emptiness. Returning `[]T{}` from the empty branch eliminates the false positive at the source.

```go
// ❌ Triggers nilaway warnings on every test that indexes the result
if nothingFound {
    return nil
}

// ✅ Empty slice — same len/range behavior, no nilaway flags
if nothingFound {
    return []ExternalFile{}
}
```

### Test conventions

- `package foo_test` (blackbox) — never `package foo`
- `t.Parallel()` on every test AND every subtest
- Each subtest gets its own data — never share mutable state
- `g := NewWithT(t)` per test
- Inline fakes in `_test.go` (no separate testdata files for fakes)
- Use `t.TempDir()` for file fixtures

## Tactical Workflow

1. **Write the failing test first** (TDD red).
2. **Write minimal implementation** to make it green.
3. **Run `targ reorder-decls`** — fix declaration ordering proactively. Treat as a formatter.
4. **Run `targ check-full`** — read EVERY failure (don't stop at the first). Group fixes:
   - All `gochecknoglobals` together (delete top-level vars)
   - All `wsl_v5` together (add blank lines)
   - All coverage gaps together (add tests for default branches)
   - All `cyclop` together (refactor into helpers)
5. **Fix all in one pass.** One lint failure usually means more nearby — find them with `targ check-full`, not `targ check`.
6. **Re-run `targ check-full`.** When green, commit immediately.
7. **Commit** with `feat|fix|refactor|test(scope): description` + `AI-Used: [claude]` trailer (NOT `Co-Authored-By`). Specific files in `git add` (no `-A`/`.`). Pre-commit hook failure → fix and create a NEW commit (never `--amend`).

## Common Refactor Patterns

### Parser → state object

When a parser function has nested switches and shared mutable state:

```go
// Before: one function over cyclop limit
func parseYAMLBlock(yaml string) Frontmatter {
    var fm Frontmatter
    var inFolded bool
    var foldedLines []string
    for _, line := range strings.Split(yaml, "\n") {
        // ... 12+ branches ...
    }
    return fm
}

// After: state object with small methods
type parserState struct {
    matter      Frontmatter
    inFolded    bool
    foldedLines []string
}

func parseYAMLBlock(yaml string) Frontmatter {
    state := newParserState()
    for line := range strings.SplitSeq(yaml, "\n") {
        state.consumeLine(line)
    }
    state.flushFolded()
    return state.matter
}

func (s *parserState) consumeLine(line string) { /* ... */ }
func (s *parserState) continueFolded(line string) { /* ... */ }
func (s *parserState) flushFolded() { /* ... */ }
func (s *parserState) startKey(line string) { /* ... */ }
```

### Discovery → per-source helpers

```go
// Before: one big function
func DiscoverClaudeMd(...) []ExternalFile {
    files := make([]ExternalFile, 0, 8)
    // ... walk ancestors (loop)
    // ... add user scope (if/append)
    // ... add managed policy (switch + if/append)
    return files
}

// After: thin orchestrator + helpers
func DiscoverClaudeMd(...) []ExternalFile {
    files := make([]ExternalFile, 0, 8)
    files = walkAncestors(files, cwd, statFn)
    files = addUserScope(files, home, statFn)
    files = addManagedPolicy(files, goos, statFn)
    return files
}

func walkAncestors(files []ExternalFile, start string, statFn StatFunc) []ExternalFile { /* ... */ }
func addUserScope(files []ExternalFile, home string, statFn StatFunc) []ExternalFile { /* ... */ }
func addManagedPolicy(files []ExternalFile, goos string, statFn StatFunc) []ExternalFile { /* ... */ }
```

### Loops with short-circuits → guarded helper

```go
// Before: interleaved short-circuits
for _, entry := range items {
    if ctx.Err() != nil { break }
    if buf.Len() >= cap { break }
    if read fails { continue }
    if extract fails { continue }
    buf.WriteString(snippet)
}

// After: extract one-iteration helper
for _, entry := range items {
    if ctx.Err() != nil || buf.Len() >= cap { break }
    processOne(ctx, entry, buf, ...)
}

func processOne(ctx context.Context, entry Entry, buf *strings.Builder, ...) {
    body, err := read(entry)
    if err != nil { return }
    snippet, err := extract(ctx, body, query)
    if err != nil || snippet == "" { return }
    buf.WriteString(snippet)
}
```

## Anti-Patterns We Keep Hitting

| Anti-pattern | Symptom | Fix |
|---|---|---|
| `var pkgFilenames = []string{...}` at file scope | gochecknoglobals fails | Inline literal inside the function that uses it |
| `var runtimeGOOSFn = func() string { return runtime.GOOS }` | gochecknoglobals fails | Pass `goos` as a parameter to the public API |
| `dir := start` immediately followed by `for {` | wsl_v5 rejects | Insert a blank line between them |
| Commit before running `targ check-full` | Pre-commit hook fails, half-staged mess | Always `targ check-full` → green → commit |
| `--amend` after pre-commit hook failure | "Trailer is `AI-Used: [claude]`... NEVER amend" warning | Create a NEW commit (`git commit`, not `git commit --amend`) |
| Switch-only test for `String()`, no default branch coverage | Coverage falls below 80% | Add an out-of-range sentinel test (e.g., `Kind(99)`) |
| Stop at first lint failure | Whack-a-mole | Always `targ check-full` (not `targ check`) — read every failure |
| Mid-investigation stall on a global | 15+ min, 30+ tool uses, no commit | Pick a reasonable approach (parameter, sync.OnceValue, function-local), try it, iterate, commit |

## Red Flags — Stop and Apply This Skill

If you find yourself doing any of these, you are off-track:

- About to write `var foo = ...` at file scope (and it's not a sentinel error)
- Function body has 5+ `if`/`switch` branches and you haven't extracted helpers
- About to commit before running `targ check-full`
- Pre-commit hook failed and you're typing `git commit --amend`
- About to use `bytes.Index` + slicing instead of `bytes.Cut`
- Test file uses `package foo` instead of `package foo_test`
- Skipping `t.Parallel()` because "this test is small"
- Commit message has `Co-Authored-By:` instead of `AI-Used: [claude]`

## When to Ask

- A function genuinely needs cyclop > 10 because the domain is irreducibly complex (rare). → Ask before disabling cyclop with `//nolint:cyclop` and document why inline.
- A `var` at file scope is genuinely required (e.g., a sync primitive). → Ask before disabling gochecknoglobals; document the reason inline.
- A coverage gap is intentional because the branch is unreachable defensive code. → Comment with reason; reviewer may accept or push back.
