# Targ CLI Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace flag.FlagSet manual parsing with targ struct-based arg parsing across all engram CLI commands.

**Architecture:** Each command function accepts `(ctx context.Context, args XArgs, stdout io.Writer) error`. Targ dispatches subcommands via `targ.Main()`. The double-parse bridge (targ structs -> flag strings -> flag.FlagSet) is eliminated. Learn uses `targ.Group` for feedback/fact subcommands.

**Tech Stack:** Go, targ (github.com/toejough/targ), gomega, rapid

---

## File Structure

| File | Role | Action |
|------|------|--------|
| `cmd/engram/main.go` | Entry point | Modify: targ.Main() |
| `internal/cli/targets.go` | Arg structs + target wiring | Modify: add structs, rewrite Targets() |
| `internal/cli/cli.go` | Core commands + helpers | Modify: convert runRecall, delete Run/RunSafe/bridge |
| `internal/cli/show.go` | Show command | Modify: convert runShow, delete extractSlug/resolveSlug |
| `internal/cli/list.go` | List command | Modify: convert runList |
| `internal/cli/learn.go` | Learn commands | Modify: convert runLearnFeedback/runLearnFact, delete runLearn/learnCommonFlags |
| `internal/cli/update.go` | Update command | Modify: convert runUpdate, delete updateFlags |
| `internal/cli/signal.go` | Signal context | Delete: targ provides signal-aware context |
| `internal/cli/export_test.go` | Test exports | Modify: remove ExportRunLearn |
| `internal/cli/targets_test.go` | Target + integration tests | Modify: rewrite to use targ.Execute |
| `internal/cli/show_test.go` | Show tests | Modify: rewrite to use targ.Execute |
| `internal/cli/list_test.go` | List tests | Modify: rewrite to use targ.Execute |
| `internal/cli/learn_test.go` | Learn tests | Modify: rewrite to use targ.Execute |
| `internal/cli/update_test.go` | Update tests | Modify: rewrite to use targ.Execute |

---

### Task 1: Add new arg structs and bridge functions

**Files:**
- Modify: `internal/cli/targets.go`
- Test: `internal/cli/targets_test.go`

This task adds the missing arg structs for learn and update commands, plus temporary bridge functions that convert them to flag strings (matching the existing pattern for recall/show/list). This lets us wire all commands through Targets() in the next task.

- [ ] **Step 1: Write test for LearnFeedbackArgs wiring through targ.Execute**

In `internal/cli/targets_test.go`, add:

```go
func TestBuildTargets_LearnFeedbackWiring(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedSubcmd string
	var capturedFlags []string

	targets := cli.BuildTargets(func(subcmd string, flags []string) {
		capturedSubcmd = subcmd
		capturedFlags = flags
	})

	_, err := targ.Execute([]string{
		"engram", "learn", "feedback",
		"--situation", "test-sit",
		"--source", "human",
		"--behavior", "test-beh",
		"--impact", "test-imp",
		"--action", "test-act",
		"--no-dup-check",
		"--data-dir", "/tmp/test",
	}, targets...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedSubcmd).To(gomega.Equal("learn feedback"))
	g.Expect(capturedFlags).To(gomega.ContainElements(
		"--situation", "test-sit",
		"--source", "human",
		"--behavior", "test-beh",
		"--data-dir", "/tmp/test",
	))
	g.Expect(capturedFlags).To(gomega.ContainElement("--no-dup-check"))
}
```

- [ ] **Step 2: Write test for LearnFactArgs wiring**

```go
func TestBuildTargets_LearnFactWiring(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedSubcmd string
	var capturedFlags []string

	targets := cli.BuildTargets(func(subcmd string, flags []string) {
		capturedSubcmd = subcmd
		capturedFlags = flags
	})

	_, err := targ.Execute([]string{
		"engram", "learn", "fact",
		"--situation", "Go projects",
		"--source", "agent",
		"--subject", "engram",
		"--predicate", "uses",
		"--object", "targ",
		"--data-dir", "/tmp/test",
	}, targets...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedSubcmd).To(gomega.Equal("learn fact"))
	g.Expect(capturedFlags).To(gomega.ContainElements(
		"--subject", "engram",
		"--predicate", "uses",
		"--object", "targ",
	))
}
```

- [ ] **Step 3: Write test for UpdateArgs wiring**

```go
func TestBuildTargets_UpdateWiring(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedSubcmd string
	var capturedFlags []string

	targets := cli.BuildTargets(func(subcmd string, flags []string) {
		capturedSubcmd = subcmd
		capturedFlags = flags
	})

	_, err := targ.Execute([]string{
		"engram", "update",
		"--name", "test-mem",
		"--situation", "new-sit",
		"--data-dir", "/tmp/test",
	}, targets...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedSubcmd).To(gomega.Equal("update"))
	g.Expect(capturedFlags).To(gomega.ContainElements(
		"--name", "test-mem",
		"--situation", "new-sit",
	))
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `targ test -- -run 'TestBuildTargets_Learn|TestBuildTargets_Update' ./internal/cli/`
Expected: FAIL — structs and bridge functions don't exist yet, BuildTargets only returns 3 targets.

- [ ] **Step 5: Add arg structs to targets.go**

Add to `internal/cli/targets.go`:

```go
// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Situation  string `targ:"flag,name=situation,desc=context when this applies"`
	Source     string `targ:"flag,name=source,desc=human or agent"`
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	NoDupCheck bool   `targ:"flag,name=no-dup-check,desc=skip duplicate/contradiction detection"`
}

// LearnFeedbackArgs holds parsed flags for the learn feedback subcommand.
type LearnFeedbackArgs struct {
	CommonLearnArgs
	Behavior string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact   string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action   string `targ:"flag,name=action,desc=recommended action"`
}

// LearnFactArgs holds parsed flags for the learn fact subcommand.
type LearnFactArgs struct {
	CommonLearnArgs
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	Name      string `targ:"flag,name=name,required,desc=memory slug (required)"`
	DataDir   string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
	Source    string `targ:"flag,name=source,desc=human or agent"`
}
```

- [ ] **Step 6: Add bridge functions**

Add to `internal/cli/targets.go`:

```go
// LearnFeedbackFlags returns the CLI flag args for the learn feedback subcommand.
func LearnFeedbackFlags(a LearnFeedbackArgs) []string {
	flags := BuildFlags(
		"--situation", a.Situation,
		"--source", a.Source,
		"--data-dir", a.DataDir,
		"--behavior", a.Behavior,
		"--impact", a.Impact,
		"--action", a.Action,
	)
	flags = AddBoolFlag(flags, "--no-dup-check", a.NoDupCheck)

	return flags
}

// LearnFactFlags returns the CLI flag args for the learn fact subcommand.
func LearnFactFlags(a LearnFactArgs) []string {
	flags := BuildFlags(
		"--situation", a.Situation,
		"--source", a.Source,
		"--data-dir", a.DataDir,
		"--subject", a.Subject,
		"--predicate", a.Predicate,
		"--object", a.Object,
	)
	flags = AddBoolFlag(flags, "--no-dup-check", a.NoDupCheck)

	return flags
}

// UpdateFlags returns the CLI flag args for the update subcommand.
func UpdateFlags(a UpdateArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--data-dir", a.DataDir,
		"--situation", a.Situation,
		"--behavior", a.Behavior,
		"--impact", a.Impact,
		"--action", a.Action,
		"--subject", a.Subject,
		"--predicate", a.Predicate,
		"--object", a.Object,
		"--source", a.Source,
	)
}
```

- [ ] **Step 7: Expand BuildTargets to include learn group and update**

Update `BuildTargets` in `internal/cli/targets.go`:

```go
func BuildTargets(run func(subcmd string, flags []string)) []any {
	return []any{
		targ.Targ(func(a RecallArgs) { run("recall", RecallFlags(a)) }).
			Name("recall").Description("Recall recent session context"),
		targ.Targ(func(a ShowArgs) { run("show", ShowFlags(a)) }).
			Name("show").Description("Display full memory details"),
		targ.Targ(func(a ListArgs) { run("list", ListFlags(a)) }).
			Name("list").Description("List all memories with type, name, and situation"),
		targ.Group("learn",
			targ.Targ(func(a LearnFeedbackArgs) { run("learn feedback", LearnFeedbackFlags(a)) }).
				Name("feedback").Description("Learn from behavioral feedback"),
			targ.Targ(func(a LearnFactArgs) { run("learn fact", LearnFactFlags(a)) }).
				Name("fact").Description("Learn a factual statement"),
		),
		targ.Targ(func(a UpdateArgs) { run("update", UpdateFlags(a)) }).
			Name("update").Description("Update an existing memory"),
	}
}
```

- [ ] **Step 8: Update Targets() to route learn subcommands correctly**

The `run` closure in `Targets()` currently builds args as `["engram", subcmd, ...flags]`. For learn subcommands, `subcmd` will be `"learn feedback"` or `"learn fact"`, which needs splitting:

```go
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	run := func(subcmd string, flags []string) {
		parts := strings.Fields(subcmd)
		args := append([]string{"engram"}, parts...)
		args = append(args, flags...)
		RunSafe(args, stdout, stderr, stdin)
	}

	return BuildTargets(run)
}
```

Note: `Targets()` needs `"strings"` import.

- [ ] **Step 9: Update TestBuildTargets existing test for new target count**

Update `TestBuildTargets/returns expected number of targets` in `targets_test.go`:

```go
// was: g.Expect(targets).To(gomega.HaveLen(3))
g.Expect(targets).To(gomega.HaveLen(5))
```

And update `TestBuildTargets/each subcommand wires to correct name`:

```go
t.Run("each subcommand wires to correct name", func(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	targets := cli.BuildTargets(func(subcmd string, _ []string) {
		calls = append(calls, subcmd)
	})

	subcmds := []string{"recall", "show", "list"}
	for _, sub := range subcmds {
		_, _ = targ.Execute([]string{"engram", sub}, targets...)
	}

	// Learn subcommands need the group prefix
	_, _ = targ.Execute([]string{
		"engram", "learn", "feedback",
		"--source", "human",
	}, targets...)
	_, _ = targ.Execute([]string{
		"engram", "learn", "fact",
		"--source", "agent",
	}, targets...)
	_, _ = targ.Execute([]string{
		"engram", "update",
		"--name", "test",
	}, targets...)

	g.Expect(calls).To(gomega.Equal([]string{
		"recall", "show", "list",
		"learn feedback", "learn fact", "update",
	}))
})
```

Also update `TestTargets/returns expected target count`:

```go
// was: g.Expect(targets).To(gomega.HaveLen(3))
g.Expect(targets).To(gomega.HaveLen(5))
```

- [ ] **Step 10: Run tests to verify they pass**

Run: `targ test -- -run 'TestBuildTargets|TestTargets' ./internal/cli/`
Expected: PASS

- [ ] **Step 11: Commit**

```
feat(cli): add learn/update arg structs and wire into Targets()
```

---

### Task 2: Update main.go to use targ.Main()

**Files:**
- Modify: `cmd/engram/main.go`

- [ ] **Step 1: Update main.go**

Replace `cmd/engram/main.go` with:

```go
// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"github.com/toejough/targ"

	"engram/internal/cli"
)

func main() {
	// targ.Main handles dispatch, help, errors-to-stderr, and exit (ARCH-6).
	targ.Main(cli.Targets(os.Stdout, os.Stderr, os.Stdin)...)
}
```

- [ ] **Step 2: Build and smoke test**

Run: `targ build`
Then: `./engram recall --help` (or equivalent) — verify targ generates help text.
Then: `./engram list --data-dir /tmp/engram-test` — verify basic command routing works.

- [ ] **Step 3: Commit**

```
feat(cli): use targ.Main() as entry point
```

---

### Task 3: Migrate tests from cli.Run to targ.Execute

**Files:**
- Modify: `internal/cli/targets_test.go`
- Modify: `internal/cli/show_test.go`
- Modify: `internal/cli/list_test.go`
- Modify: `internal/cli/learn_test.go`
- Modify: `internal/cli/update_test.go`

All tests that call `cli.Run(args, &stdout, &stderr, stdin)` need to switch to `targ.Execute(args, cli.Targets(&stdout, &stderr, stdin)...)`. Since `targ.Execute` returns `(targ.ExecuteResult, error)` but the commands write errors through `RunSafe` (which always succeeds), we check stdout/stderr content rather than the execute error for command-level failures.

**Important:** After this task, `cli.Run` and `cli.RunSafe` are no longer called by any test. They become dead code, ready for deletion.

- [ ] **Step 1: Create test helper in targets_test.go**

Add at top of `internal/cli/targets_test.go`:

```go
// executeForTest runs an engram CLI command through targ, returning stdout content.
// Command-level errors are written to stderr (RunSafe contract), not returned as Go errors.
func executeForTest(t *testing.T, args []string) (stdoutStr, stderrStr string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, strings.NewReader(""))
	_, _ = targ.Execute(args, targets...)

	return stdout.String(), stderr.String()
}
```

- [ ] **Step 2: Migrate TestRunRecall tests in targets_test.go**

Replace the four `TestRunRecall` subtests. Each currently calls `cli.Run(...)` and checks `err`. Switch to `executeForTest` and check stderr for errors.

```go
func TestRunRecall(t *testing.T) {
	t.Parallel()

	t.Run("runs with empty data dir and no sessions", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()
		projectSlug := "test-recall-empty"

		home, homeErr := os.UserHomeDir()
		g.Expect(homeErr).NotTo(gomega.HaveOccurred())

		if homeErr != nil {
			return
		}

		projectDir := filepath.Join(home, ".claude", "projects", projectSlug)
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		stdout, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", projectSlug,
		})
		g.Expect(stderr).To(gomega.BeEmpty())
		_ = stdout
	})

	t.Run("defaults data dir and project slug when omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		home, homeErr := os.UserHomeDir()
		g.Expect(homeErr).NotTo(gomega.HaveOccurred())

		if homeErr != nil {
			return
		}

		cwd, cwdErr := os.Getwd()
		g.Expect(cwdErr).NotTo(gomega.HaveOccurred())

		if cwdErr != nil {
			return
		}

		defaultSlug := cli.ProjectSlugFromPath(cwd)
		projectDir := filepath.Join(home, ".claude", "projects", defaultSlug)
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		_, stderr := executeForTest(t, []string{"engram", "recall"})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("runs with query flag", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()
		projectSlug := "test-recall-query"

		home, homeErr := os.UserHomeDir()
		g.Expect(homeErr).NotTo(gomega.HaveOccurred())

		if homeErr != nil {
			return
		}

		projectDir := filepath.Join(home, ".claude", "projects", projectSlug)
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", projectSlug,
			"--query", "something",
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("returns error on invalid flag", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--invalid-flag",
		})
		g.Expect(stderr).NotTo(gomega.BeEmpty())
	})
}
```

- [ ] **Step 3: Replace TestRunSafe in targets_test.go**

```go
func TestRunSafe(t *testing.T) {
	t.Parallel()

	t.Run("writes error to stderr on failure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// Unknown subcommand triggers error path.
		_, stderr := executeForTest(t, []string{"engram", "nonexistent-subcommand"})
		g.Expect(stderr).NotTo(gomega.BeEmpty())
	})
}
```

- [ ] **Step 4: Migrate show_test.go tests**

Replace all `cli.Run(...)` calls with `executeForTest`. For error cases, check stderr. For success cases, check stdout.

Key changes for each test:
- `TestShow_FlagParseError_ReturnsError`: check `stderr` not empty
- `TestShow_HappyPath_PrintsSBIAFields`: check `stdout` content
- `TestShow_MemoryNotFound_ReturnsError`: check `stderr` contains "nonexistent-memory"
- `TestShow_MissingSlug_ReturnsError`: check `stderr` contains "slug"
- `TestShow_NameFlag_Works`: check `stdout` content
- `TestShow_OmitsEmptyFields`: check `stdout` content
- `TestShow_ReadsFromFactsDir`: check `stdout` content
- `TestShow_ReadsFromFeedbackDir`: check `stdout` content
- `TestShow_SlugAfterFlags_Works`: check `stdout` content

Pattern for success tests:
```go
// Before:
var stdout, stderr bytes.Buffer
err := cli.Run([]string{"engram", "show", "slug", "--data-dir", dataDir}, &stdout, &stderr, strings.NewReader(""))
g.Expect(err).NotTo(HaveOccurred())
output := stdout.String()

// After:
stdout, stderr := executeForTest(t, []string{"engram", "show", "slug", "--data-dir", dataDir})
g.Expect(stderr).To(BeEmpty())
output := stdout
```

Pattern for error tests:
```go
// Before:
err := cli.Run([]string{"engram", "show", "--bogus-flag"}, &stdout, &stderr, strings.NewReader(""))
g.Expect(err).To(HaveOccurred())
g.Expect(err.Error()).To(ContainSubstring("show"))

// After:
_, stderr := executeForTest(t, []string{"engram", "show", "--bogus-flag"})
g.Expect(stderr).To(ContainSubstring("show"))
```

Apply this pattern to all tests in `show_test.go`. Remove `var stdout, stderr bytes.Buffer` declarations and `strings.NewReader("")` args. Remove `if err != nil { return }` guards (no error to check).

- [ ] **Step 5: Migrate list_test.go tests**

Same pattern. 4 tests to convert:
- `TestList_EmptyDataDir_ReturnsEmptyOutput`
- `TestList_FeedbackMemory_OutputsTypeNameSituation`
- `TestList_FlagParseError_ReturnsError`
- `TestList_MultipleMixedMemories_OutputsBoth`

- [ ] **Step 6: Migrate learn_test.go integration tests**

Tests that call `cli.Run` (NOT the unit tests that use Export* helpers — those stay unchanged):
- `TestLearnFact_FlagParseError_ReturnsError`
- `TestLearnFact_InvalidSource_ReturnsError`
- `TestLearnFact_NoDupCheck_WritesToFactsDir`
- `TestLearnFeedback_AgentSource_Accepted`
- `TestLearnFeedback_FlagParseError_ReturnsError`
- `TestLearnFeedback_InvalidSource_ReturnsError`
- `TestLearnFeedback_NoDupCheck_WritesToFeedbackDir`
- `TestLearnFeedback_OutputFormatIncludesCreatedName`
- `TestLearn_NoSubcommand_ReturnsError`
- `TestLearn_UnknownSubcommand_ReturnsError`

Same conversion pattern. For `TestLearn_NoSubcommand_ReturnsError` — with targ, `engram learn` with no subcommand shows help text (not an error). This test should verify targ handles it (stderr may show help or be empty). Adjust expectation: the test becomes a no-op or is deleted since targ handles missing subcommands.

For `TestLearn_UnknownSubcommand_ReturnsError` — targ will produce an error for unknown subcommand. Check stderr.

- [ ] **Step 7: Migrate update_test.go tests**

All 7 tests use `cli.Run`. Same conversion pattern.

- [ ] **Step 8: Run all tests**

Run: `targ test -- ./internal/cli/`
Expected: PASS

- [ ] **Step 9: Commit**

```
refactor(cli): migrate tests from cli.Run to targ.Execute
```

---

### Task 4: Convert command functions to typed args and delete old dispatch

**Files:**
- Modify: `internal/cli/cli.go` (recall + delete Run/RunSafe/newFlagSet/constants)
- Modify: `internal/cli/show.go` (show)
- Modify: `internal/cli/list.go` (list)
- Modify: `internal/cli/learn.go` (learn feedback, learn fact)
- Modify: `internal/cli/update.go` (update)
- Modify: `internal/cli/targets.go` (Targets closures + delete bridge functions)
- Modify: `internal/cli/export_test.go` (delete ExportRunLearn)
- Delete: `internal/cli/signal.go`

Convert each command function from `func(args []string, stdout io.Writer) error` to `func(ctx context.Context, args XArgs, stdout io.Writer) error`, removing flag.FlagSet parsing. Then update `Targets()` to call directly instead of through the bridge.

- [ ] **Step 1: Convert runRecall**

Change signature and body in `cli.go`:

```go
func runRecall(ctx context.Context, args RecallArgs, stdout io.Writer) error {
	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("recall: %w", defaultErr)
	}

	token := resolveToken(ctx)
	summarizer := newSummarizer(token)
	memLister := memory.NewLister()

	if args.MemoriesOnly {
		limit := args.Limit
		if limit == 0 {
			limit = recall.DefaultMemoryLimit
		}

		return runRecallMemoriesOnly(ctx, stdout, summarizer, memLister, dataDir, args.Query, limit)
	}

	projectSlug := args.ProjectSlug

	return runRecallSessions(ctx, stdout, &projectSlug, summarizer, memLister, dataDir, args.Query)
}
```

Remove the old `runRecall(args []string, stdout io.Writer) error` function.

- [ ] **Step 2: Convert runShow**

Change signature and body in `show.go`:

```go
func runShow(ctx context.Context, args ShowArgs, stdout io.Writer) error {
	_ = ctx // available for future use

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("show: %w", defaultErr)
	}

	if args.Name == "" {
		return errShowMissingSlug
	}

	memPath := memory.ResolveMemoryPath(dataDir, args.Name, fileExists)

	mem, err := loadMemoryTOML(memPath)
	if err != nil {
		return fmt.Errorf("show: loading %s: %w", args.Name, err)
	}

	renderMemory(stdout, mem)

	return nil
}
```

Delete `extractSlug()` and `resolveSlug()` functions. Remove `"flag"` from imports if no longer used. Remove the `nameFlag` field handling — the slug now comes directly from `args.Name`.

**Note on ShowArgs:** The design says Name should be a positional arg. However, existing tests use both `--name` flag and positional. To minimize test churn in this step, keep Name as a flag for now. We can convert to positional in a follow-up if desired. The `--data-dir` flag on `ShowArgs` stays as-is.

- [ ] **Step 3: Convert runList**

Change signature and body in `list.go`:

```go
func runList(ctx context.Context, args ListArgs, stdout io.Writer) error {
	_ = ctx

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("list: %w", defaultErr)
	}

	lister := memory.NewLister()

	memories, err := lister.ListAllMemories(dataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("list: %w", err)
	}

	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)

		_, writeErr := fmt.Fprintf(stdout, "%s | %s | %s\n", mem.Type, name, mem.Situation)
		if writeErr != nil {
			return fmt.Errorf("list: %w", writeErr)
		}
	}

	return nil
}
```

- [ ] **Step 4: Convert runLearnFeedback and runLearnFact**

In `learn.go`, replace `runLearnFeedback` and `runLearnFact`:

```go
func runLearnFeedback(ctx context.Context, args LearnFeedbackArgs, stdout io.Writer) error {
	_ = ctx

	srcErr := validateSource(args.Source)
	if srcErr != nil {
		return fmt.Errorf("learn feedback: %w", srcErr)
	}

	record := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        args.Source,
		Situation:     args.Situation,
		Type:          typeFeedback,
		Content: memory.ContentFields{
			Behavior: args.Behavior,
			Impact:   args.Impact,
			Action:   args.Action,
		},
	}

	dataDir := args.DataDir

	return writeMemory(record, args.Situation, &dataDir, args.NoDupCheck, stdout, "learn feedback")
}

func runLearnFact(ctx context.Context, args LearnFactArgs, stdout io.Writer) error {
	_ = ctx

	srcErr := validateSource(args.Source)
	if srcErr != nil {
		return fmt.Errorf("learn fact: %w", srcErr)
	}

	record := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        args.Source,
		Situation:     args.Situation,
		Type:          typeFact,
		Content: memory.ContentFields{
			Subject:   args.Subject,
			Predicate: args.Predicate,
			Object:    args.Object,
		},
	}

	dataDir := args.DataDir

	return writeMemory(record, args.Situation, &dataDir, args.NoDupCheck, stdout, "learn fact")
}
```

Delete:
- `runLearn()` function
- `learnCommonFlags` struct
- `registerCommonFlags()` function
- `parseAndValidate()` function
- `errLearnUsage`, `errUnknownLearnSubcmd` variables

- [ ] **Step 5: Convert runUpdate**

In `update.go`:

```go
func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error {
	_ = ctx

	if args.Source != "" {
		srcErr := validateSource(args.Source)
		if srcErr != nil {
			return fmt.Errorf("update: %w", srcErr)
		}
	}

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("update: %w", defaultErr)
	}

	memPath := memory.ResolveMemoryPath(dataDir, args.Name, fileExists)

	record, loadErr := loadMemoryTOML(memPath)
	if loadErr != nil {
		return fmt.Errorf("update: loading %s: %w", args.Name, loadErr)
	}

	applyUpdateArgs(record, args)

	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	writer := tomlwriter.New()

	writeErr := writer.AtomicWrite(memPath, record)
	if writeErr != nil {
		return fmt.Errorf("update: writing %s: %w", args.Name, writeErr)
	}

	name := memory.NameFromPath(memPath)

	_, printErr := fmt.Fprintf(stdout, "UPDATED: %s\n", name)
	if printErr != nil {
		return fmt.Errorf("update: %w", printErr)
	}

	return nil
}

// applyUpdateArgs sets only non-empty field values on the record.
func applyUpdateArgs(record *memory.MemoryRecord, args UpdateArgs) {
	if args.Situation != "" {
		record.Situation = args.Situation
	}

	if args.Behavior != "" {
		record.Content.Behavior = args.Behavior
	}

	if args.Impact != "" {
		record.Content.Impact = args.Impact
	}

	if args.Action != "" {
		record.Content.Action = args.Action
	}

	if args.Subject != "" {
		record.Content.Subject = args.Subject
	}

	if args.Predicate != "" {
		record.Content.Predicate = args.Predicate
	}

	if args.Object != "" {
		record.Content.Object = args.Object
	}

	if args.Source != "" {
		record.Source = args.Source
	}
}
```

Delete:
- Old `runUpdate` function
- `updateFlags` struct
- `registerUpdateFlags()` function
- `applyUpdateFields()` function (replaced by `applyUpdateArgs`)

- [ ] **Step 6: Rewrite Targets() to call command functions directly**

Replace `Targets()` and `BuildTargets()` in `targets.go`:

```go
// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, stdin io.Reader) []any {
	errHandler := func(err error) {
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
		}
	}

	return []any{
		targ.Targ(func(ctx context.Context, a RecallArgs) {
			errHandler(runRecall(ctx, a, stdout))
		}).Name("recall").Description("Recall recent session context"),
		targ.Targ(func(ctx context.Context, a ShowArgs) {
			errHandler(runShow(ctx, a, stdout))
		}).Name("show").Description("Display full memory details"),
		targ.Targ(func(ctx context.Context, a ListArgs) {
			errHandler(runList(ctx, a, stdout))
		}).Name("list").Description("List all memories with type, name, and situation"),
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				errHandler(runLearnFeedback(ctx, a, stdout))
			}).Name("feedback").Description("Learn from behavioral feedback"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				errHandler(runLearnFact(ctx, a, stdout))
			}).Name("fact").Description("Learn a factual statement"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(ctx, a, stdout))
		}).Name("update").Description("Update an existing memory"),
	}
}
```

- [ ] **Step 7: Delete Run(), RunSafe(), old dispatch, and bridge functions**

Now that `Targets()` calls command functions directly:

Delete from `cli.go`:
- `Run()` function
- `errUnknownCommand`, `errUsage` variables
- `minArgs` constant
- `newFlagSet()` function

Delete from `export_test.go`:
- `ExportRunLearn` function

Delete from `targets.go`:
- `RunSafe()` function
- `BuildTargets()` function
- `BuildFlags()` function
- `AddBoolFlag()` function
- `AddIntFlag()` function
- `RecallFlags()` function
- `ShowFlags()` function
- `ListFlags()` function
- `LearnFeedbackFlags()` function
- `LearnFactFlags()` function
- `UpdateFlags()` function

Remove `"strconv"` and `"strings"` from imports if no longer used.

Delete from `cli.go`:
- `newFlagSet()` function (if not already deleted in Task 4)

- [ ] **Step 8: Delete signal.go**

Delete `internal/cli/signal.go` entirely. Targ provides signal-aware context.

- [ ] **Step 9: Clean up imports**

In each modified file, remove unused imports:
- `cli.go`: remove `"flag"` if unused
- `show.go`: remove `"flag"`, `"strings"` if unused
- `list.go`: remove `"flag"` if unused
- `learn.go`: remove `"flag"` if unused
- `update.go`: remove `"flag"` if unused
- `targets.go`: remove `"strconv"`, `"strings"` if unused

- [ ] **Step 10: Run tests**

Run: `targ test -- ./internal/cli/`
Expected: PASS (or some tests fail due to bridge test removals — address in next step)

- [ ] **Step 11: Commit**

```
refactor(cli): convert all commands to targ typed args
```

---

### Task 5: Clean up tests and dead code

**Files:**
- Modify: `internal/cli/targets_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Delete bridge tests from targets_test.go**

Delete these test functions (they test deleted bridge code):
- `TestAddBoolFlag`
- `TestAddIntFlag`
- `TestBuildFlags`
- `TestBuildTargets` (entire function — replaced by tests using Targets() directly)
- `TestBuildTargets_LearnFeedbackWiring` (bridge test from Task 1)
- `TestBuildTargets_LearnFactWiring` (bridge test from Task 1)
- `TestBuildTargets_UpdateWiring` (bridge test from Task 1)
- `TestListFlags`
- `TestRecallFlags`
- `TestShowFlags`

- [ ] **Step 2: Update TestTargets**

The `TestTargets/returns expected target count` test should now expect the full count. The `TestTargets/closure wiring` test should exercise a command through the direct path:

```go
func TestTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected target count", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(targets).To(gomega.HaveLen(6))
	})

	t.Run("show command writes to injected stdout", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{"engram", "show", "--name", "nonexistent", "--data-dir", t.TempDir()}, targets...)

		// show with nonexistent memory produces error (written to stderr), stdout is empty.
		g.Expect(stdout.String()).To(gomega.BeEmpty())
	})
}
```

Note: target count is 6 because `targ.Group("learn", ...)` counts as 1 top-level entry, plus recall, show, list, update = 5 top-level targets. Verify the actual count at runtime and adjust.

- [ ] **Step 3: Clean up export_test.go**

Remove any exports that reference deleted types or functions. Keep exports for business logic helpers that are still tested (buildMemoryIndex, describeNewMemory, parseConflictResponse, renderConflictContent, renderFactContent, renderMemoryContent, validateSource, writeMemory, etc.).

- [ ] **Step 4: Run full test suite**

Run: `targ check-full`
Expected: PASS with no lint errors

- [ ] **Step 5: Commit**

```
refactor(cli): delete bridge tests and clean up exports
```

---

### Task 6: Final verification

**Files:** None (read-only verification)

- [ ] **Step 1: Build binary**

Run: `targ build`
Expected: Clean build, no errors.

- [ ] **Step 2: Run full quality check**

Run: `targ check-full`
Expected: All checks pass (tests, lint, coverage).

- [ ] **Step 3: Manual smoke test**

Test each command through the real binary:

```bash
# List (empty dir)
./engram list --data-dir /tmp/engram-smoke-test

# Learn feedback
./engram learn feedback \
  --situation "smoke test" \
  --behavior "manual verification" \
  --impact "confirms wiring" \
  --action "run after refactors" \
  --source human \
  --no-dup-check \
  --data-dir /tmp/engram-smoke-test

# List (should show the memory)
./engram list --data-dir /tmp/engram-smoke-test

# Show
./engram show --name <slug-from-list> --data-dir /tmp/engram-smoke-test

# Update
./engram update --name <slug-from-list> --action "updated action" --data-dir /tmp/engram-smoke-test

# Learn fact
./engram learn fact \
  --situation "Go projects" \
  --subject "engram" \
  --predicate "uses" \
  --object "targ for CLI parsing" \
  --source agent \
  --no-dup-check \
  --data-dir /tmp/engram-smoke-test

# Recall (memories only)
./engram recall --data-dir /tmp/engram-smoke-test --memories-only --query "targ"

# Help
./engram --help
./engram learn --help

# Clean up
rm -rf /tmp/engram-smoke-test
```

Every command should produce expected output with no empty/nil fields.

- [ ] **Step 4: Final commit**

If any fixes were needed during verification, commit them:

```
fix(cli): address issues found during smoke test
```

- [ ] **Step 5: Summary commit (if all clean)**

```
docs: update targ CLI migration plan as complete
```
