# Engram Show Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `engram show <slug>` CLI command that displays full memory details (principle, anti-pattern, content, keywords, effectiveness) so the agent can pull deep context for any surfaced memory.

**Architecture:** New `show` case in CLI dispatcher. Resolves slug to `<data-dir>/memories/<slug>.toml`, loads the memory via existing `memory.Stored` parser, formats and prints all fields. Pure CLI wiring — no new packages needed.

**Tech Stack:** Go, gomega, targ

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/cli/show.go` | `runShow` function: parse flags, resolve slug to file path, load memory, format output |
| `internal/cli/show_test.go` | Tests for show command via `cli.Run` |
| `internal/cli/cli.go` | Add `case "show"` to dispatcher |

---

### Task 1: Add show command — resolve slug, load memory, print details

**Files:**
- Create: `internal/cli/show.go`
- Create: `internal/cli/show_test.go`
- Modify: `internal/cli/cli.go` (add `case "show"`)

- [ ] **Step 1: Write the failing test — show prints memory details**

In `internal/cli/show_test.go`:

```go
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestShow_PrintsMemoryDetails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o750)).To(Succeed())

	memContent := `title = "Use targ check-full"
content = "Always use targ check-full for comprehensive validation"
keywords = ["targ", "check", "full"]
principle = "Run targ check-full before declaring done"
anti_pattern = "Running targ check which stops at first error"
surfaced_count = 10
followed_count = 8
contradicted_count = 1
ignored_count = 1
`
	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "use-targ-check-full.toml"),
		[]byte(memContent), 0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "use-targ-check-full", "--data-dir", dataDir},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Use targ check-full"))
	g.Expect(output).To(ContainSubstring("Run targ check-full before declaring done"))
	g.Expect(output).To(ContainSubstring("Running targ check which stops at first error"))
	g.Expect(output).To(ContainSubstring("targ, check, full"))
	g.Expect(output).To(ContainSubstring("80%"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestShow_PrintsMemoryDetails -v`
Expected: FAIL — `show` command not recognized.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/cli/cli.go` dispatcher (around line 144, after `apply-proposal`):

```go
case "show":
    return runShow(subArgs, stdout)
```

Create `internal/cli/show.go`:

```go
package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"engram/internal/memory"
)

func runShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("show: %w", parseErr)
	}

	if *dataDir == "" {
		return errShowMissingDataDir
	}

	slug := fs.Arg(0)
	if slug == "" {
		return errShowMissingSlug
	}

	memPath := filepath.Join(*dataDir, "memories", slug+".toml")

	mem, loadErr := memory.LoadFile(memPath)
	if loadErr != nil {
		return fmt.Errorf("show: loading %s: %w", slug, loadErr)
	}

	printMemory(stdout, mem)

	return nil
}

func printMemory(w io.Writer, mem *memory.Stored) {
	_, _ = fmt.Fprintf(w, "Title: %s\n", mem.Title)
	_, _ = fmt.Fprintf(w, "Principle: %s\n", mem.Principle)

	if mem.AntiPattern != "" {
		_, _ = fmt.Fprintf(w, "Anti-pattern: %s\n", mem.AntiPattern)
	}

	if mem.Content != "" {
		_, _ = fmt.Fprintf(w, "Content: %s\n", mem.Content)
	}

	if len(mem.Keywords) > 0 {
		_, _ = fmt.Fprintf(w, "Keywords: %s\n", strings.Join(mem.Keywords, ", "))
	}

	total := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
	if total > 0 {
		score := mem.FollowedCount * 100 / total
		_, _ = fmt.Fprintf(w, "Effectiveness: %d%% (%d followed, %d contradicted, %d ignored)\n",
			score, mem.FollowedCount, mem.ContradictedCount, mem.IgnoredCount)
	}
}
```

Add sentinel errors:

```go
var (
	errShowMissingDataDir = errors.New("show: --data-dir is required")
	errShowMissingSlug    = errors.New("show: memory slug argument is required")
)
```

**Important:** Check if `memory.LoadFile` exists. If not, check for `memory.Load`, `memory.ParseFile`, or similar. The memory package must have a function that reads a TOML file and returns `*memory.Stored`. Search with: `grep -n "func Load\|func Parse" internal/memory/memory.go`

If no such function exists, use the retrieve package's `ListMemories` or parse the TOML directly with the existing TOML library. The slug resolves to `<data-dir>/memories/<slug>.toml`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestShow_PrintsMemoryDetails -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/show.go internal/cli/show_test.go internal/cli/cli.go
git commit -m "feat(cli): add engram show command for full memory details (#342)"
```

---

### Task 2: Error path tests

**Files:**
- Modify: `internal/cli/show_test.go`

- [ ] **Step 1: Add error path tests**

```go
func TestShow_MissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "some-slug"},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
}

func TestShow_MissingSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "--data-dir", t.TempDir()},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("slug"))
	}
}

func TestShow_MemoryNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "nonexistent-memory", "--data-dir", t.TempDir()},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/cli/ -run "TestShow" -v`
Expected: all PASS

- [ ] **Step 3: Run targ check-full**

Run: `targ check-full`
Expected: all checks pass (except check-uncommitted)

- [ ] **Step 4: Commit**

```bash
git add internal/cli/show_test.go
git commit -m "test(cli): add show command error path tests (#342)"
```

---

### Task 3: Wire slug argument before flags

**Note:** `flag.Parse` in Go stops at the first non-flag argument. If the user calls `engram show use-targ-check-full --data-dir /path`, the slug comes before the flag. The `fs.Arg(0)` approach in Task 1 only works if the slug comes AFTER flags: `engram show --data-dir /path use-targ-check-full`.

Check both orderings in the test. If slug-first doesn't work, extract the slug before flag parsing:

```go
// Extract slug from args before flag parsing.
// Slug is the first arg that doesn't start with "--".
var slug string
var flagArgs []string
for _, arg := range args {
    if slug == "" && !strings.HasPrefix(arg, "--") {
        slug = arg
    } else {
        flagArgs = append(flagArgs, arg)
    }
}
```

- [ ] **Step 1: Add test for slug-first ordering**

```go
func TestShow_SlugBeforeFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o750)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "my-mem.toml"),
		[]byte("title = \"My Memory\"\nprinciple = \"Do the thing\"\n"), 0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	// Slug BEFORE --data-dir flag.
	err := cli.Run(
		[]string{"engram", "show", "my-mem", "--data-dir", dataDir},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("My Memory"))
}
```

- [ ] **Step 2: Fix implementation if needed**

If the test fails, update `runShow` to extract slug before flag parsing as shown above.

- [ ] **Step 3: Run all show tests + check-full**

Run: `targ check-full`
Expected: all checks pass

- [ ] **Step 4: Commit if changes made**

```bash
git add internal/cli/show.go internal/cli/show_test.go
git commit -m "fix(cli): show command handles slug before flags (#342)"
```
