# Strip Keywords Blobs from Situation Fields Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `strip-keywords` cleanup command that removes `\nKeywords: ...` suffixes from `situation` fields in all memory TOML files under `memory/feedback/` and `memory/facts/`.

**Architecture:** A pure string-transform function `StripKeywordsSuffix` lives in a new `internal/stripkeywords` package; a new `cmd/strip-keywords` binary wires it to the filesystem via the same injected-deps pattern used by `cmd/migrate-v2`. The transform is property-tested with `pgregory.net/rapid` before the implementation exists.

**Tech Stack:** Go, `pgregory.net/rapid` (property-based testing), `github.com/onsi/gomega`, `github.com/BurntSushi/toml`, `targ test`.

---

## File Map

| File | Change |
|------|--------|
| `internal/stripkeywords/strip.go` | Create: pure `StripKeywordsSuffix(string) string` function |
| `internal/stripkeywords/strip_test.go` | Create: rapid property test + example table tests |
| `internal/stripkeywords/run.go` | Create: `Run(dataDir string, deps Deps) error` and `Deps` struct |
| `internal/stripkeywords/run_test.go` | Create: integration tests for `Run` |
| `cmd/strip-keywords/main.go` | Create: CLI entry point |

---

## Task 1: Pure transform function with property-based tests

**Files:**
- Create: `internal/stripkeywords/strip_test.go`
- Create: `internal/stripkeywords/strip.go`

- [ ] **Step 1: Write the failing test**

Create `internal/stripkeywords/strip_test.go`:

```go
package stripkeywords_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/stripkeywords"
)

func TestStripKeywordsSuffix_NoSuffix(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("when running tests")).
		To(Equal("when running tests"))
}

func TestStripKeywordsSuffix_StripsSuffix(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("when running tests\nKeywords: go, test, targ")).
		To(Equal("when running tests"))
}

func TestStripKeywordsSuffix_MultipleNewlines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("line one\nline two\nKeywords: a, b")).
		To(Equal("line one\nline two"))
}

func TestStripKeywordsSuffix_KeywordsWithNoLeadingNewline(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	// "Keywords: ..." without a preceding \n must NOT be stripped (not the suffix pattern)
	g.Expect(stripkeywords.StripKeywordsSuffix("Keywords: foo, bar")).
		To(Equal("Keywords: foo, bar"))
}

func TestStripKeywordsSuffix_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("")).To(Equal(""))
}

func TestStripKeywordsSuffix_Idempotent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		base := rapid.StringMatching(`[A-Za-z0-9 .,!?;:'\-]{1,80}`).Draw(rt, "base")

		withSuffix := base + "\nKeywords: " +
			rapid.StringMatching(`[A-Za-z0-9 ,]{1,40}`).Draw(rt, "keywords")

		once := stripkeywords.StripKeywordsSuffix(withSuffix)
		twice := stripkeywords.StripKeywordsSuffix(once)

		g.Expect(twice).To(Equal(once))
		g.Expect(once).NotTo(ContainSubstring("\nKeywords:"))
	})
}

func TestStripKeywordsSuffix_NeverCorruptsContent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		base := rapid.StringMatching(`[A-Za-z0-9 .,!?;'\-]{1,80}`).Draw(rt, "base")
		input := base + "\nKeywords: " +
			rapid.StringMatching(`[A-Za-z0-9 ,]{1,40}`).Draw(rt, "kw")

		result := stripkeywords.StripKeywordsSuffix(input)

		g.Expect(result).To(Equal(base))
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `package engram/internal/stripkeywords: cannot find package`

- [ ] **Step 3: Write minimal implementation**

Create `internal/stripkeywords/strip.go`:

```go
// Package stripkeywords removes legacy Keywords suffixes from memory situation fields.
package stripkeywords

import "strings"

// StripKeywordsSuffix removes the first occurrence of "\nKeywords: ..." from s,
// where "..." extends to the end of the string. If no such suffix exists, s is
// returned unchanged. The operation is idempotent.
func StripKeywordsSuffix(s string) string {
	idx := strings.Index(s, "\nKeywords:")
	if idx == -1 {
		return s
	}

	return s[:idx]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stripkeywords/strip.go internal/stripkeywords/strip_test.go
git commit -m "$(cat <<'EOF'
feat(stripkeywords): add StripKeywordsSuffix with property-based tests

Pure transform, no file I/O yet. Part of #459.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: `Run` function that walks memory dirs and patches files

**Files:**
- Create: `internal/stripkeywords/run_test.go`
- Create: `internal/stripkeywords/run.go`

**Note:** Before writing these files, check what helper functions and types exist in `internal/memory/` for getting directory paths and the `MemoryRecord` type. Use `grep -r "FeedbackDir\|FactsDir\|MemoryRecord" internal/memory/` to find them. If `FeedbackDir`/`FactsDir` don't exist as functions, use `filepath.Join(dataDir, "memory", "feedback")` and `filepath.Join(dataDir, "memory", "facts")` directly in `run.go` instead.

- [ ] **Step 1: Write the failing test**

Create `internal/stripkeywords/run_test.go`:

```go
package stripkeywords_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/stripkeywords"
)

// memRecord is a minimal struct for reading/writing test memory TOML files.
// (Avoids importing internal/memory which may pull in heavy deps.)
type memRecord struct {
	Type      string `toml:"type"`
	Situation string `toml:"situation"`
	UpdatedAt string `toml:"updated_at"`
	CreatedAt string `toml:"created_at"`
}

func TestRun_StripsKeywordsFromFeedbackAndFacts(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	factsDir := filepath.Join(dataDir, "memory", "facts")
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(factsDir, 0o750)).To(Succeed())

	writeMem := func(dir, filename, situation string) {
		rec := memRecord{
			Type:      "feedback",
			Situation: situation,
			UpdatedAt: "2026-01-01T00:00:00Z",
			CreatedAt: "2026-01-01T00:00:00Z",
		}
		var buf bytes.Buffer
		g.Expect(toml.NewEncoder(&buf).Encode(rec)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(dir, filename), buf.Bytes(), 0o640)).To(Succeed())
	}

	writeMem(feedbackDir, "fb.toml", "when running tests\nKeywords: go, targ")
	writeMem(factsDir, "fact.toml", "context for project\nKeywords: project, context")
	writeMem(feedbackDir, "clean.toml", "when deploying")

	stdout := &bytes.Buffer{}
	deps := stripkeywords.DefaultDeps()
	deps.Stdout = stdout

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	readSituation := func(path string) string {
		data, readErr := os.ReadFile(path)
		g.Expect(readErr).NotTo(HaveOccurred())
		if readErr != nil {
			return ""
		}
		var rec memRecord
		_, decErr := toml.Decode(string(data), &rec)
		g.Expect(decErr).NotTo(HaveOccurred())
		return rec.Situation
	}

	g.Expect(readSituation(filepath.Join(feedbackDir, "fb.toml"))).
		To(Equal("when running tests"))
	g.Expect(readSituation(filepath.Join(factsDir, "fact.toml"))).
		To(Equal("context for project"))
	g.Expect(readSituation(filepath.Join(feedbackDir, "clean.toml"))).
		To(Equal("when deploying"))

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("STRIPPED: fb.toml"))
	g.Expect(out).To(ContainSubstring("STRIPPED: fact.toml"))
	g.Expect(out).To(ContainSubstring("OK (no change): clean.toml"))
	g.Expect(out).To(ContainSubstring("Stripped: 2, Unchanged: 1"))
}

func TestRun_Idempotent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	rec := memRecord{
		Type:      "feedback",
		Situation: "when running tests\nKeywords: go, targ",
		UpdatedAt: "2026-01-01T00:00:00Z",
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	var buf bytes.Buffer
	g.Expect(toml.NewEncoder(&buf).Encode(rec)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), buf.Bytes(), 0o640)).To(Succeed())

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}

	g.Expect(stripkeywords.Run(dataDir, deps)).To(Succeed())

	secondOut := &bytes.Buffer{}
	deps.Stdout = secondOut
	g.Expect(stripkeywords.Run(dataDir, deps)).To(Succeed())

	data, readErr := os.ReadFile(filepath.Join(feedbackDir, "mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())
	if readErr != nil {
		return
	}
	var result memRecord
	_, decErr := toml.Decode(string(data), &result)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(result.Situation).To(Equal("when running tests"))
	g.Expect(secondOut.String()).To(ContainSubstring("Stripped: 0"))
}

func TestRun_MissingFeedbackDir_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	// No memory/feedback directory created

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_InvalidTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(feedbackDir, "bad.toml"),
		[]byte("not = [valid toml"),
		0o640,
	)).To(Succeed())

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("bad.toml"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — missing `Run`, `DefaultDeps`, `Deps`

- [ ] **Step 3: Write minimal implementation**

Create `internal/stripkeywords/run.go`:

```go
package stripkeywords

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// memRecord is the minimal TOML shape needed to read/write situation fields.
type memRecord struct {
	Type      string `toml:"type"`
	Situation string `toml:"situation"`
	UpdatedAt string `toml:"updated_at"`
	CreatedAt string `toml:"created_at"`
}

// Deps holds injected I/O dependencies for the cleanup run.
type Deps struct {
	ReadDir    func(string) ([]os.DirEntry, error)
	ReadFile   func(string) ([]byte, error)
	CreateTemp func(string, string) (*os.File, error)
	Rename     func(string, string) error
	Remove     func(string) error
	Stdout     io.Writer
	Stderr     io.Writer
}

// DefaultDeps returns Deps wired to real filesystem operations.
func DefaultDeps() Deps {
	return Deps{
		ReadDir:    os.ReadDir,
		ReadFile:   os.ReadFile,
		CreateTemp: os.CreateTemp,
		Rename:     os.Rename,
		Remove:     os.Remove,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
}

// Run walks memory/feedback/ and memory/facts/ under dataDir, stripping
// "\nKeywords: ..." suffixes from situation fields. It is idempotent.
func Run(dataDir string, deps Deps) error {
	dirs := []string{
		filepath.Join(dataDir, "memory", "feedback"),
		filepath.Join(dataDir, "memory", "facts"),
	}

	totalStripped := 0
	totalUnchanged := 0

	for _, dir := range dirs {
		stripped, unchanged, err := processDir(dir, deps)
		if err != nil {
			return err
		}

		totalStripped += stripped
		totalUnchanged += unchanged
	}

	_, _ = fmt.Fprintf(deps.Stdout, "\nStripped: %d, Unchanged: %d\n", totalStripped, totalUnchanged)

	return nil
}

func processDir(dir string, deps Deps) (stripped, unchanged int, err error) {
	entries, err := deps.ReadDir(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("reading %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		changed, processErr := processFile(path, entry.Name(), deps)
		if processErr != nil {
			return 0, 0, processErr
		}

		if changed {
			stripped++
		} else {
			unchanged++
		}
	}

	return stripped, unchanged, nil
}

func processFile(path, name string, deps Deps) (bool, error) {
	data, err := deps.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", name, err)
	}

	var rec memRecord

	if _, decErr := toml.Decode(string(data), &rec); decErr != nil {
		return false, fmt.Errorf("%s: decoding TOML: %w", name, decErr)
	}

	stripped := StripKeywordsSuffix(rec.Situation)
	if stripped == rec.Situation {
		_, _ = fmt.Fprintf(deps.Stdout, "OK (no change): %s\n", name)

		return false, nil
	}

	rec.Situation = stripped

	if writeErr := atomicWrite(path, rec, deps); writeErr != nil {
		return false, fmt.Errorf("%s: writing: %w", name, writeErr)
	}

	_, _ = fmt.Fprintf(deps.Stdout, "STRIPPED: %s\n", name)

	return true, nil
}

func atomicWrite(path string, rec memRecord, deps Deps) error {
	dir := filepath.Dir(path)

	tmpFile, err := deps.CreateTemp(dir, ".tmp-stripkw-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	cleanup := func() { _ = deps.Remove(tmpPath) }

	encErr := toml.NewEncoder(tmpFile).Encode(rec)
	closeErr := tmpFile.Close()

	if encErr != nil {
		cleanup()
		return fmt.Errorf("encoding TOML: %w", encErr)
	}

	if closeErr != nil {
		cleanup()
		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	if renameErr := deps.Rename(tmpPath, path); renameErr != nil {
		cleanup()
		return fmt.Errorf("renaming temp to destination: %w", renameErr)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stripkeywords/run.go internal/stripkeywords/run_test.go
git commit -m "$(cat <<'EOF'
feat(stripkeywords): add Run function that walks memory dirs and patches files

Idempotent. Covers memory/feedback/ and memory/facts/. Part of #459.

AI-Used: [claude]
EOF
)"
```

---

## Task 3: CLI entry point `cmd/strip-keywords`

**Files:**
- Create: `cmd/strip-keywords/main.go`

- [ ] **Step 1: Verify the package doesn't exist yet**

Run: `go build ./cmd/strip-keywords/...`
Expected: FAIL — `no Go files in .../cmd/strip-keywords`

- [ ] **Step 2: Create the CLI entry point**

Create `cmd/strip-keywords/main.go`:

```go
// Package main provides the strip-keywords CLI entry point.
// It removes legacy "\nKeywords: ..." suffixes from situation fields
// in all memory TOML files under the data directory.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"engram/internal/stripkeywords"
)

func main() {
	flags := flag.NewFlagSet("strip-keywords", flag.ContinueOnError)
	dataDir := flags.String(
		"data-dir",
		filepath.Join(os.Getenv("HOME"), ".claude", "engram", "data"),
		"path to data directory",
	)

	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if err := stripkeywords.Run(*dataDir, stripkeywords.DefaultDeps()); err != nil {
		fmt.Fprintf(os.Stderr, "strip-keywords: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./cmd/strip-keywords/...`
Expected: SUCCESS (no output)

- [ ] **Step 4: Run full test suite**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/strip-keywords/main.go
git commit -m "$(cat <<'EOF'
feat: add strip-keywords CLI command

Fixes #459. Strips \nKeywords: ... suffix from situation fields
in memory/feedback/ and memory/facts/. Idempotent.

AI-Used: [claude]
EOF
)"
```
