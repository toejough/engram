# Plan: Centralize policy.toml filename constant (#455)

**Goal:** Extract the `"policy.toml"` string literal to a single exported constant in `internal/policy/`, then replace all Go occurrences.

## Analysis

The string `"policy.toml"` appears as a literal in **34 locations** across Go files:
- 11 in source files (across 6 files)
- 23 in test files (across 5 files)

One additional occurrence (`policy_test.go:398`) embeds `"policy.toml"` at the tail of a full fixture path (`"/nonexistent/path/policy.toml"`) passed to `LoadFromPath`. This is left as-is — it's a path literal, not a standalone filename reference. Replacing it with string concatenation would hurt readability for no safety benefit.

It also appears in docs, plans, hooks, and README — those are prose/shell and stay as-is.

## Constant Location

`internal/policy/policy.go` — the package that owns policy concepts. The constant already exists as an unexported local `const policyPath = "policy.toml"` inside `Load()`. We promote it to a package-level exported constant.

```go
// Filename is the basename of the policy configuration file.
const Filename = "policy.toml"
```

## Task 1: Add exported constant and update internal/policy/

**Files:**
- `internal/policy/policy.go` — add `const Filename = "policy.toml"` after imports, replace local `const policyPath` in `Load()` with `Filename`
- `internal/policy/policy_test.go` — replace 10 occurrences of `"policy.toml"` with `policy.Filename`

**Verify:** `targ check-full`
**Commit:** `refactor(policy): export Filename constant for policy.toml (#455)`

## Task 2: Replace literals in internal/maintain/

**Files:**
- `internal/maintain/adapt.go:72` — replace `adaptTarget = "policy.toml"` with `adaptTarget = policy.Filename`
- `internal/maintain/gateaccuracy.go:28` — replace `Target: "policy.toml"` with `Target: policy.Filename`
- `internal/maintain/adapt_test.go` — replace 2 occurrences with `policy.Filename`
- `internal/maintain/gateaccuracy_test.go:33` — replace 1 occurrence with `policy.Filename`. **Must add `"engram/internal/policy"` import** (not currently imported).

Note: `adapt.go` already imports `policy` (uses `policy.AppendChangeHistory`). `gateaccuracy.go` already imports `policy` (uses `policy.Proposal`). `adapt_test.go` already imports `policy`. `gateaccuracy_test.go` does NOT — needs import added.

**Verify:** `targ check-full`
**Commit:** `refactor(maintain): use policy.Filename constant (#455)`

## Task 3: Replace literals in internal/cli/

**Files:**
- `internal/cli/cli.go` — 2 occurrences (lines 538, 663)
- `internal/cli/evaluate.go` — 1 occurrence (line 108)
- `internal/cli/maintain.go` — 4 occurrences (lines 52, 140, 195, 287)
- `internal/cli/refine.go` — 1 occurrence (line 172)
- `internal/cli/cli_test.go` — 1 occurrence (line 238)
- `internal/cli/maintain_test.go` — 9 occurrences (including line 211: `Target: filepath.Join(dataDir, "policy.toml")`)
- `internal/cli/refine_test.go` — 6 occurrences

All `internal/cli/` source files already import `policy` (for `policy.Load`, `policy.LoadFromPath`, etc.). Test files may need the import added — verify each.

**Verify:** `targ check-full`
**Commit:** `refactor(cli): use policy.Filename constant (#455)`

## Out of Scope

- `hooks/session-start.sh` — shell script, can't use Go constants
- `README.md`, `docs/`, `archive/`, `skills/` — prose references, not code
- Plan/spec markdown files — historical documentation

## Constraints

- No concurrent `targ` runs — coordinate via chat.toml
- TDD not needed — no new behavior, pure mechanical extraction
- All tests must pass with `targ check-full` after each task
- Commit trailer: `AI-Used: [claude]`
