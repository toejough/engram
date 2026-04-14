# Hold Flag Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Validate required flags in `runHoldAcquire` and `runHoldRelease` so missing flags exit non-zero with a descriptive error instead of silently succeeding.

**Architecture:** Inline validation added immediately after flag parsing in each function. No new abstractions. Three new tests follow the established `TestRun_HoldAcquire_ParseError_ReturnsError` pattern.

**Tech Stack:** Go, `flag` stdlib, gomega for tests, `targ` for build/test/check.

---

## Files

- Modify: `internal/cli/cli_test.go` — add 3 failing tests
- Modify: `internal/cli/cli.go:717–724` (runHoldAcquire), `internal/cli/cli.go:892–898` (runHoldRelease) — add validation

---

### Task 1: Write failing tests for missing required flags

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add three failing tests after `TestRun_HoldAcquire_ParseError_ReturnsError` (line ~1021)**

Open `internal/cli/cli_test.go`. After `TestRun_HoldAcquire_ParseError_ReturnsError` (around line 1021), insert:

```go
func TestRun_HoldAcquire_EmptyHolder_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run([]string{"engram", "hold", "acquire", "--chat-file", chatFile}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--holder is required"))
}

func TestRun_HoldAcquire_EmptyTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run([]string{"engram", "hold", "acquire", "--chat-file", chatFile, "--holder", "lead"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--target is required"))
}

func TestRun_HoldRelease_EmptyHoldID_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run([]string{"engram", "hold", "release", "--chat-file", chatFile}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--hold-id is required"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test
```

Expected: the three new tests fail. `TestRun_HoldAcquire_EmptyHolder_ReturnsError` and `TestRun_HoldAcquire_EmptyTarget_ReturnsError` and `TestRun_HoldRelease_EmptyHoldID_ReturnsError` each expect an error but get nil.

---

### Task 2: Add validation to `runHoldAcquire`

**Files:**
- Modify: `internal/cli/cli.go` — `runHoldAcquire` function

- [ ] **Step 1: Add holder and target validation after the `parseErr != nil` block**

In `internal/cli/cli.go`, locate `runHoldAcquire` (line ~708). After this existing block (lines ~722–724):

```go
	if parseErr != nil {
		return fmt.Errorf("hold acquire: %w", parseErr)
	}
```

Insert:

```go
	if *holder == "" {
		return fmt.Errorf("hold acquire: --holder is required")
	}

	if *target == "" {
		return fmt.Errorf("hold acquire: --target is required")
	}
```

The function body should now read (lines ~717 onwards):

```go
	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold acquire: %w", parseErr)
	}

	if *holder == "" {
		return fmt.Errorf("hold acquire: --holder is required")
	}

	if *target == "" {
		return fmt.Errorf("hold acquire: --target is required")
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold acquire", os.UserHomeDir, os.Getwd)
```

- [ ] **Step 2: Run the two acquire tests to verify they now pass**

```bash
targ test
```

Expected: `TestRun_HoldAcquire_EmptyHolder_ReturnsError` and `TestRun_HoldAcquire_EmptyTarget_ReturnsError` pass. `TestRun_HoldRelease_EmptyHoldID_ReturnsError` still fails.

---

### Task 3: Add validation to `runHoldRelease`

**Files:**
- Modify: `internal/cli/cli.go` — `runHoldRelease` function

- [ ] **Step 1: Add hold-id validation after the `parseErr != nil` block**

In `internal/cli/cli.go`, locate `runHoldRelease` (line ~886). After this existing block (lines ~892–898):

```go
	if parseErr != nil {
		return fmt.Errorf("hold release: %w", parseErr)
	}
```

Insert:

```go
	if *holdID == "" {
		return fmt.Errorf("hold release: --hold-id is required")
	}
```

The function body should now read (lines ~892 onwards):

```go
	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold release: %w", parseErr)
	}

	if *holdID == "" {
		return fmt.Errorf("hold release: --hold-id is required")
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold release", os.UserHomeDir, os.Getwd)
```

- [ ] **Step 2: Run all tests to verify all three pass**

```bash
targ test
```

Expected: all three new tests pass. All pre-existing tests continue to pass.

---

### Task 4: Full quality check and commit

**Files:**
- No new files

- [ ] **Step 1: Run full lint + coverage check**

```bash
targ check-full
```

Expected: no lint errors, coverage thresholds pass. If coverage drops, the three new tests cover the new branches — this should be net positive.

- [ ] **Step 2: Commit**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "fix(cli): validate required flags in hold acquire/release

Fixes #518. runHoldAcquire now errors when --holder or --target is
empty. runHoldRelease now errors when --hold-id is empty. Previously
both commands exited 0 with no error, silently polluting the hold
registry with invalid entries.

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage:**
- ✓ `runHoldAcquire` validates `--holder` → Task 2
- ✓ `runHoldAcquire` validates `--target` → Task 2
- ✓ `runHoldRelease` validates `--hold-id` → Task 3
- ✓ Non-zero exit with descriptive error → returning `error` from `Run` causes non-zero via `os.Exit(1)` in main
- ✓ TDD: tests written before implementation → Tasks 1 then 2+3
- ✓ Tests use `t.Parallel()` → included in all three test functions

**Placeholder scan:** None found.

**Type consistency:** No new types introduced. `cli.Run` signature unchanged. `err.Error()` assertions match `fmt.Errorf(...)` strings exactly.
