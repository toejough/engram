# Delete Dead Instruct Package and Review Command — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the unreachable `internal/instruct/` package and the dead `review` CLI subcommand.

**Architecture:** Pure deletion — no new code. Remove the package directory, its CLI wiring (case, flag parser, targ target, args/flags types), associated tests, and documentation references. Verify with `targ check-full` after each task.

**Tech Stack:** Go, targ build system

---

## Inventory of Dead Code

### `internal/instruct/` package (completely unreachable)
- `internal/instruct/scanner.go` — Scanner, InstructionItem, SourceType
- `internal/instruct/scanner_test.go` — scanner tests
- `internal/instruct/audit.go` — Auditor, AuditReport, all types, all unexported helpers
- `internal/instruct/audit_test.go` — audit tests

No code outside this package imports `engram/internal/instruct`.

### `instruct` CLI wiring (stub — prints a line and returns nil)
- `internal/cli/cli.go:75-76` — `case "instruct"` in Run switch
- `internal/cli/cli.go:578-604` — `runInstructAudit` function
- `internal/cli/cli.go:111` — `instruct` in errUsage string
- `internal/cli/targets.go:36-41` — `InstructArgs` struct
- `internal/cli/targets.go:149-150` — targ target registration
- `internal/cli/targets.go:198-201` — `InstructFlags` function
- `internal/cli/cli_test.go:163-228` — three instruct test functions
- `internal/cli/targets_test.go:136` — `"instruct"` in subcmd list

### `review` CLI wiring (no case in Run switch — completely dead)
- `internal/cli/targets.go:87-91` — `ReviewArgs` struct
- `internal/cli/targets.go:143-144` — targ target registration
- `internal/cli/targets.go:257-259` — `ReviewFlags` function
- `internal/cli/targets_test.go:135` — `"review"` in subcmd list

### Documentation references
- `docs/design/architecture.md:63` — review row in command table
- `docs/design/architecture.md:99` — instruct row in command table

---

### Task 1: Delete `internal/instruct/` package

**Files:**
- Delete: `internal/instruct/scanner.go`
- Delete: `internal/instruct/scanner_test.go`
- Delete: `internal/instruct/audit.go`
- Delete: `internal/instruct/audit_test.go`

- [ ] **Step 1: Delete the package directory**

```bash
rm -rf internal/instruct/
```

- [ ] **Step 2: Run `targ check-full` to confirm nothing breaks**

```bash
targ check-full
```

Expected: PASS — no code outside this package imports it.

- [ ] **Step 3: Commit**

```bash
git add -A internal/instruct/
git commit -m "refactor(instruct): delete unreachable instruct package

The internal/instruct/ package (scanner, auditor, all types) has no
importers outside its own tests. Remove entirely.

Closes part of #453.

AI-Used: [claude]"
```

---

### Task 2: Remove `instruct` CLI wiring

**Files:**
- Modify: `internal/cli/cli.go` (remove case, function, usage string reference)
- Modify: `internal/cli/targets.go` (remove InstructArgs, InstructFlags, targ target)
- Modify: `internal/cli/cli_test.go` (remove three instruct test functions)
- Modify: `internal/cli/targets_test.go` (remove "instruct" from subcmd list)

- [ ] **Step 1: Remove the `case "instruct"` from the Run switch in `internal/cli/cli.go:75-76`**

Delete these two lines:
```go
	case "instruct":
		return runInstructAudit(subArgs, stdout)
```

- [ ] **Step 2: Remove `runInstructAudit` function from `internal/cli/cli.go:578-604`**

Delete the entire function (lines 578–604):
```go
func runInstructAudit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("instruct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	projectDir := fs.String("project-dir", "", "path to project directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("instruct: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("instruct: %w", defaultErr)
	}

	if *projectDir == "" {
		*projectDir = "."
	}

	// Instruct audit uses its own scanner; it doesn't depend on deleted packages.
	_, _ = fmt.Fprintf(stdout, "[engram] instruct audit: data-dir=%s project-dir=%s\n",
		*dataDir, *projectDir)

	return nil
}
```

- [ ] **Step 3: Remove `|instruct` from errUsage string in `internal/cli/cli.go:110-111`**

Change:
```go
	errUsage = errors.New(
		"usage: engram <correct|surface|show|recall|maintain" +
			"|apply-proposal|reject-proposal|instruct|evaluate|refine|migrate-slugs> [flags]",
	)
```

To:
```go
	errUsage = errors.New(
		"usage: engram <correct|surface|show|recall|maintain" +
			"|apply-proposal|reject-proposal|evaluate|refine|migrate-slugs> [flags]",
	)
```

- [ ] **Step 4: Remove `InstructArgs` struct from `internal/cli/targets.go:36-41`**

Delete:
```go
// InstructArgs holds parsed flags for the instruct subcommand.
type InstructArgs struct {
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectDir string `targ:"flag,name=project-dir,desc=path to project directory"`
	APIToken   string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
}
```

- [ ] **Step 5: Remove instruct targ target from `BuildTargets` in `internal/cli/targets.go:149-150`**

Delete:
```go
		targ.Targ(func(a InstructArgs) { run("instruct", InstructFlags(a)) }).
			Name("instruct").Description("Audit instruction quality"),
```

- [ ] **Step 6: Remove `InstructFlags` function from `internal/cli/targets.go:198-201`**

Delete:
```go
// InstructFlags returns the CLI flag args for the instruct subcommand.
func InstructFlags(a InstructArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--project-dir", a.ProjectDir)
}
```

- [ ] **Step 7: Remove three instruct test functions from `internal/cli/cli_test.go`**

Delete `TestRun_Instruct_DefaultProjectDir` (lines ~163-188), `TestRun_Instruct_ParseError` (lines ~190-206), and `TestRun_Instruct_PrintsAuditInfo` (lines ~208-228).

- [ ] **Step 8: Remove `"instruct"` from subcmd list in `internal/cli/targets_test.go:136`**

Change:
```go
		subcmds := []string{
			"correct", "review",
			"maintain", "surface", "instruct",
```

To:
```go
		subcmds := []string{
			"correct", "review",
			"maintain", "surface",
```

(Note: `"review"` is removed in Task 3 — leave it here for now so this task's commit is self-contained.)

- [ ] **Step 9: Run `targ check-full`**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/cli/cli.go internal/cli/targets.go internal/cli/cli_test.go internal/cli/targets_test.go
git commit -m "refactor(cli): remove instruct subcommand wiring

runInstructAudit was a stub (print and return nil). Remove the case,
function, args struct, flags builder, targ target, and all three tests.

Closes part of #453.

AI-Used: [claude]"
```

---

### Task 3: Remove `review` CLI wiring

**Files:**
- Modify: `internal/cli/targets.go` (remove ReviewArgs, ReviewFlags, targ target)
- Modify: `internal/cli/targets_test.go` (remove "review" from subcmd list)

- [ ] **Step 1: Remove `ReviewArgs` struct from `internal/cli/targets.go:87-91`**

Delete:
```go
// ReviewArgs holds parsed flags for the review subcommand.
type ReviewArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Format  string `targ:"flag,name=format,default=table,desc=output format: json or table"`
}
```

- [ ] **Step 2: Remove review targ target from `BuildTargets` in `internal/cli/targets.go:143-144`**

Delete:
```go
		targ.Targ(func(a ReviewArgs) { run("review", ReviewFlags(a)) }).
			Name("review").Description("Review instruction registry"),
```

- [ ] **Step 3: Remove `ReviewFlags` function from `internal/cli/targets.go:257-259`**

Delete:
```go
// ReviewFlags returns the CLI flag args for the review subcommand.
func ReviewFlags(a ReviewArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--format", a.Format)
}
```

- [ ] **Step 4: Remove `"review"` from subcmd list in `internal/cli/targets_test.go:135`**

Change:
```go
		subcmds := []string{
			"correct", "review",
			"maintain", "surface",
```

To:
```go
		subcmds := []string{
			"correct",
			"maintain", "surface",
```

- [ ] **Step 5: Run `targ check-full`**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/targets.go internal/cli/targets_test.go
git commit -m "refactor(cli): remove dead review subcommand

ReviewArgs/ReviewFlags were registered as a targ target but Run() had no
case for 'review' — the command was completely unreachable.

Closes part of #453.

AI-Used: [claude]"
```

---

### Task 4: Update documentation

**Files:**
- Modify: `docs/design/architecture.md`

- [ ] **Step 1: Remove the `review` row from the command table at line 63**

Delete:
```
| `review` | Effectiveness review with budget tracking and threshold analysis |
```

- [ ] **Step 2: Remove the `instruct` row from the command table at line 99**

Delete:
```
| `instruct` | Instruction quality audit |
```

- [ ] **Step 3: Run `targ check-full`**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add docs/design/architecture.md
git commit -m "docs(architecture): remove instruct and review from command table

Both commands were deleted in #453.

AI-Used: [claude]"
```
