# `engram quick` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `engram quick` CLI subcommand that writes a fleeting note to `<vault>/Fleeting/<YYYY-MM-DD>.<slug>.md`, replacing the per-note `Write` tool roundtrip the `capturing-fleeting-notes` skill currently incurs.

**Architecture:** Pure-business-logic runner (`runQuick`) with all I/O behind injected function deps (`now`, `stdin`, `statDir`, `writeNew`). Slug validation, vault resolution, content source selection, filename derivation, and the no-overwrite write are each isolated helpers tested independently. Targ registers the subcommand alongside the existing `learn`/`recall`/`show`/etc.

**Tech Stack:** Go, `targ` build/CLI framework (`github.com/toejough/targ`), gomega for assertions (`github.com/onsi/gomega`), standard library `os`/`time`/`io`.

**Spec:** `docs/superpowers/specs/2026-05-09-engram-quick-fleeting-write-design.md`

---

## File Structure

**Create:**
- `internal/cli/quick.go` — runner + helpers + DI struct
- `internal/cli/quick_test.go` — unit tests (blackbox `package cli_test`)

**Modify:**
- `internal/cli/targets.go` — add `QuickArgs` struct + register the subcommand in `Targets()`
- `internal/cli/cli.go` — add the production `osQuickFS` adapter and wire DI in a small constructor
- `internal/cli/export_test.go` — re-export the helpers and the runner-with-deps for blackbox testing

Each helper lives as a small named function in `quick.go` so each task can add and test one in isolation.

---

## Task 1: Slug validation

**Files:**
- Create: `internal/cli/quick.go`
- Create: `internal/cli/quick_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/quick_test.go`:

```go
package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestValidateSlug_AcceptsKebabCase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateSlug("graph-connectedness-recall-axis")).To(Succeed())
	g.Expect(cli.ExportValidateSlug("a")).To(Succeed())
	g.Expect(cli.ExportValidateSlug("note-1")).To(Succeed())
}

func TestValidateSlug_RejectsInvalid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateSlug("")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("Has-Caps")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("has space")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("dot.in.it")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("under_score")).To(MatchError(ContainSubstring("slug")))
}
```

Add to `internal/cli/export_test.go`:

```go
ExportValidateSlug = validateSlug
```

(Add the line to the existing `var ( ... )` block.)

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `validateSlug` not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/quick.go`:

```go
// Package cli — quick subcommand: writes a fleeting note file.
package cli

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	errSlugEmpty   = errors.New("quick: slug is required")
	errSlugInvalid = errors.New("quick: slug must match [a-z0-9-]+")

	slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// validateSlug returns nil if slug is non-empty kebab-case lowercase.
func validateSlug(slug string) error {
	if slug == "" {
		return errSlugEmpty
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: got %q", errSlugInvalid, slug)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/quick.go internal/cli/quick_test.go internal/cli/export_test.go
git commit -m "feat(cli): add slug validation for engram quick"
```

---

## Task 2: Vault path resolution

**Files:**
- Modify: `internal/cli/quick.go`
- Modify: `internal/cli/quick_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/quick_test.go`:

```go
func TestResolveVault_FlagWins(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(string) string { return "/from/env" }
	got, err := cli.ExportResolveVault("/from/flag", getenv)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("/from/flag"))
}

func TestResolveVault_FallsBackToEnv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(name string) string {
		if name == "ENGRAM_VAULT_DIR" {
			return "/from/env"
		}
		return ""
	}
	got, err := cli.ExportResolveVault("", getenv)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("/from/env"))
}

func TestResolveVault_ErrorsWhenNeitherSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(string) string { return "" }
	_, err := cli.ExportResolveVault("", getenv)
	g.Expect(err).To(MatchError(ContainSubstring("vault")))
}
```

Append to `export_test.go`:

```go
ExportResolveVault = resolveVault
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `resolveVault` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/quick.go`:

```go
const envVaultDir = "ENGRAM_VAULT_DIR"

var errVaultUnset = errors.New("quick: vault path is required (--vault flag or ENGRAM_VAULT_DIR env)")

// resolveVault returns the vault path: flag wins, env is fallback, error if neither set.
func resolveVault(flagValue string, getenv func(string) string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := getenv(envVaultDir); env != "" {
		return env, nil
	}
	return "", errVaultUnset
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/quick.go internal/cli/quick_test.go internal/cli/export_test.go
git commit -m "feat(cli): vault path resolution for engram quick"
```

---

## Task 3: Content source resolution (flag XOR stdin)

**Files:**
- Modify: `internal/cli/quick.go`
- Modify: `internal/cli/quick_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/quick_test.go`:

```go
import (
	// ... existing imports
	"strings"
)

func TestResolveContent_FlagOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, err := cli.ExportResolveContent("hello body", strings.NewReader(""))
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("hello body"))
}

func TestResolveContent_StdinOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, err := cli.ExportResolveContent("", strings.NewReader("hello stdin"))
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("hello stdin"))
}

func TestResolveContent_ErrorsWhenBoth(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, err := cli.ExportResolveContent("flag body", strings.NewReader("stdin body"))
	g.Expect(err).To(MatchError(ContainSubstring("content")))
}

func TestResolveContent_ErrorsWhenNeither(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, err := cli.ExportResolveContent("", strings.NewReader(""))
	g.Expect(err).To(MatchError(ContainSubstring("content")))
}
```

Append to `export_test.go`:

```go
ExportResolveContent = resolveContent
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `resolveContent` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/quick.go`:

```go
import (
	// ... existing imports
	"io"
)

var (
	errContentBoth    = errors.New("quick: provide --content OR stdin, not both")
	errContentNeither = errors.New("quick: --content flag or stdin required")
)

// resolveContent picks content from flag XOR stdin. Errors on both or neither.
func resolveContent(flagValue string, stdin io.Reader) (string, error) {
	stdinBytes, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("quick: reading stdin: %w", err)
	}
	hasFlag := flagValue != ""
	hasStdin := len(stdinBytes) > 0
	if hasFlag && hasStdin {
		return "", errContentBoth
	}
	if !hasFlag && !hasStdin {
		return "", errContentNeither
	}
	if hasFlag {
		return flagValue, nil
	}
	return string(stdinBytes), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/quick.go internal/cli/quick_test.go internal/cli/export_test.go
git commit -m "feat(cli): content source resolution (flag XOR stdin) for engram quick"
```

---

## Task 4: Filename derivation

**Files:**
- Modify: `internal/cli/quick.go`
- Modify: `internal/cli/quick_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/quick_test.go`:

```go
import (
	// ... existing imports
	"time"
)

func TestFleetingPath_BuildsExpectedPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC)
	got := cli.ExportFleetingPath("/vault", "my-tag", when)
	g.Expect(got).To(Equal("/vault/Fleeting/2026-05-09.my-tag.md"))
}
```

Append to `export_test.go`:

```go
ExportFleetingPath = fleetingPath
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `fleetingPath` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/quick.go`:

```go
import (
	// ... existing imports
	"path/filepath"
	"time"
)

const (
	fleetingSubdir = "Fleeting"
	dateFormat     = "2006-01-02"
)

// fleetingPath builds the absolute path for a fleeting note file.
func fleetingPath(vault, slug string, when time.Time) string {
	filename := fmt.Sprintf("%s.%s.md", when.Format(dateFormat), slug)
	return filepath.Join(vault, fleetingSubdir, filename)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/quick.go internal/cli/quick_test.go internal/cli/export_test.go
git commit -m "feat(cli): fleeting filename derivation for engram quick"
```

---

## Task 5: Vault directory existence check

**Files:**
- Modify: `internal/cli/quick.go`
- Modify: `internal/cli/quick_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/quick_test.go`:

```go
func TestRequireFleetingDir_PassesWhenStatSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	statOK := func(string) error { return nil }
	g.Expect(cli.ExportRequireFleetingDir("/vault", statOK)).To(Succeed())
}

func TestRequireFleetingDir_ErrorsWhenStatFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	statFail := func(string) error { return errors.New("not found") }
	err := cli.ExportRequireFleetingDir("/vault", statFail)
	g.Expect(err).To(MatchError(ContainSubstring("Fleeting")))
}
```

(Also add `"errors"` to the test file's imports if not already present.)

Append to `export_test.go`:

```go
ExportRequireFleetingDir = requireFleetingDir
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `requireFleetingDir` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/quick.go`:

```go
// requireFleetingDir checks that <vault>/Fleeting exists, via the injected stat function.
func requireFleetingDir(vault string, statDir func(string) error) error {
	dir := filepath.Join(vault, fleetingSubdir)
	if err := statDir(dir); err != nil {
		return fmt.Errorf("quick: vault Fleeting directory not accessible at %s: %w", dir, err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/quick.go internal/cli/quick_test.go internal/cli/export_test.go
git commit -m "feat(cli): vault Fleeting directory existence check for engram quick"
```

---

## Task 6: `runQuick` orchestrator

**Files:**
- Modify: `internal/cli/quick.go`
- Modify: `internal/cli/quick_test.go`
- Modify: `internal/cli/export_test.go`

This wires the helpers together and exercises the no-overwrite write path via the injected `writeNew` dep.

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/quick_test.go`:

```go
func TestRunQuick_HappyPath_WritesExpectedFileAndPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		gotPath string
		gotData []byte
	)
	deps := cli.QuickDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC) },
		Stdin:   strings.NewReader(""),
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		WriteNew: func(path string, data []byte) error {
			gotPath = path
			gotData = data
			return nil
		},
	}
	args := cli.QuickArgs{
		Slug:    "test-tag",
		Content: "# tag\n\nbody.\n",
		Vault:   "/vault",
	}
	var stdout strings.Builder
	err := cli.ExportRunQuick(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(gotPath).To(Equal("/vault/Fleeting/2026-05-09.test-tag.md"))
	g.Expect(string(gotData)).To(Equal("# tag\n\nbody.\n"))
	g.Expect(stdout.String()).To(ContainSubstring("/vault/Fleeting/2026-05-09.test-tag.md"))
}

func TestRunQuick_ErrorsWhenWriteNewReportsExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.QuickDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC) },
		Stdin:   strings.NewReader(""),
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		WriteNew: func(string, []byte) error {
			return fs.ErrExist
		},
	}
	args := cli.QuickArgs{Slug: "tag", Content: "body", Vault: "/vault"}
	err := cli.ExportRunQuick(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).To(MatchError(ContainSubstring("exists")))
}

func TestRunQuick_PropagatesSlugValidationError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := cli.QuickDeps{
		Now:      func() time.Time { return time.Now() },
		Stdin:    strings.NewReader(""),
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.QuickArgs{Slug: "Bad Slug", Content: "body", Vault: "/vault"}
	err := cli.ExportRunQuick(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).To(MatchError(ContainSubstring("slug")))
}
```

Add `"io/fs"` to test imports.

Append to `export_test.go`:

```go
ExportRunQuick = runQuick
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `runQuick`, `QuickDeps` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/quick.go`:

```go
import (
	// ... existing imports
	"context"
	"io/fs"
)

// QuickDeps holds injected dependencies for runQuick. All fields required.
type QuickDeps struct {
	Now      func() time.Time
	Stdin    io.Reader
	Getenv   func(string) string
	StatDir  func(string) error
	WriteNew func(path string, data []byte) error // must error with fs.ErrExist if the file exists
}

// runQuick orchestrates the quick subcommand: validates inputs, derives the path, writes the file.
func runQuick(_ context.Context, args QuickArgs, deps QuickDeps, stdout io.Writer) error {
	if err := validateSlug(args.Slug); err != nil {
		return err
	}
	vault, err := resolveVault(args.Vault, deps.Getenv)
	if err != nil {
		return err
	}
	if err := requireFleetingDir(vault, deps.StatDir); err != nil {
		return err
	}
	content, err := resolveContent(args.Content, deps.Stdin)
	if err != nil {
		return err
	}
	path := fleetingPath(vault, args.Slug, deps.Now())
	if err := deps.WriteNew(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("quick: target file already exists: %s", path)
		}
		return fmt.Errorf("quick: writing %s: %w", path, err)
	}
	_, _ = fmt.Fprintln(stdout, path)
	return nil
}
```

Note: `QuickArgs` is defined in `targets.go` in Task 7. For now this won't compile — that's expected; Task 7 closes the gap. If you want to keep each task's tests green in isolation, define a minimal `QuickArgs` here temporarily and remove it in Task 7. **Recommended:** define it in `targets.go` *now* (one-line struct addition) so this task compiles.

To keep this task self-contained, add to `internal/cli/targets.go` (above `Targets()`):

```go
// QuickArgs holds parsed flags for the quick subcommand.
type QuickArgs struct {
	Slug    string `targ:"flag,name=slug,desc=kebab-case tag for the fleeting note"`
	Content string `targ:"flag,name=content,desc=full body markdown (or pipe via stdin)"`
	Vault   string `targ:"flag,name=vault,env=ENGRAM_VAULT_DIR,desc=vault root directory"`
}
```

(Targ registration in `Targets()` is still in Task 7 — adding the struct now lets the runner compile.)

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/quick.go internal/cli/quick_test.go internal/cli/targets.go internal/cli/export_test.go
git commit -m "feat(cli): runQuick orchestrator for engram quick"
```

---

## Task 7: Production wiring (osQuickFS adapter + targ registration)

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/quick_test.go`:

```go
func TestOsQuickFS_WriteNew_ErrorsOnExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "exists.md")
	g.Expect(os.WriteFile(path, []byte("old"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsQuickFS()
	err := fs.WriteNew(path, []byte("new"))
	g.Expect(err).To(HaveOccurred())
	if err == nil {
		return
	}
	g.Expect(errors.Is(err, ioFs.ErrExist)).To(BeTrue())
}

func TestOsQuickFS_WriteNew_CreatesNewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")

	fs := cli.ExportNewOsQuickFS()
	g.Expect(fs.WriteNew(path, []byte("hello"))).To(Succeed())

	got, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(string(got)).To(Equal("hello"))
}

func TestOsQuickFS_StatDir_ErrorsOnMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := cli.ExportNewOsQuickFS()
	err := fs.StatDir(filepath.Join(t.TempDir(), "missing"))
	g.Expect(err).To(HaveOccurred())
}

func TestOsQuickFS_StatDir_PassesOnExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := cli.ExportNewOsQuickFS()
	g.Expect(fs.StatDir(t.TempDir())).To(Succeed())
}
```

Add imports to test file: `"os"`, `"path/filepath"`, and alias `ioFs "io/fs"` (rename if `fs` is already imported as the package — adjust to whatever non-conflicting name keeps compile clean).

Add to `export_test.go`:

```go
// ExportNewOsQuickFS returns the production osQuickFS adapter for testing.
func ExportNewOsQuickFS() interface {
	StatDir(path string) error
	WriteNew(path string, data []byte) error
} {
	return &osQuickFS{}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `osQuickFS` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/cli.go`:

```go
// osQuickFS is the production filesystem adapter for the quick subcommand.
type osQuickFS struct{}

// StatDir returns an error if the directory does not exist or isn't accessible.
func (*osQuickFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}

// WriteNew creates the file with O_EXCL — errors with fs.ErrExist if it already exists.
func (*osQuickFS) WriteNew(path string, data []byte) error {
	const perm = 0o600
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}
```

Now register the targ entry. Modify `internal/cli/targets.go` `Targets()` — add this entry inside the returned `[]any{...}` after the `learn` group (or in a reasonable spot near other top-level commands):

```go
		targ.Targ(func(ctx context.Context, a QuickArgs) {
			fsAdapter := &osQuickFS{}
			deps := QuickDeps{
				Now:      time.Now,
				Stdin:    os.Stdin,
				Getenv:   os.Getenv,
				StatDir:  fsAdapter.StatDir,
				WriteNew: fsAdapter.WriteNew,
			}
			errHandler(runQuick(ctx, a, deps, stdout))
		}).Name("quick").Description("Write a fleeting note to the agent-memory vault"),
```

Add imports to `targets.go` if not already present: `"os"`, `"time"`.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: PASS — no lint errors, coverage thresholds met.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go internal/cli/targets.go internal/cli/quick_test.go internal/cli/export_test.go
git commit -m "feat(cli): wire engram quick subcommand with O_EXCL filesystem adapter"
```

---

## Task 8: Smoke test the binary end-to-end

**Files:**
- (no source changes; verification only)

- [ ] **Step 1: Build the binary**

Run: `targ build`
Expected: PASS — produces `engram` binary.

- [ ] **Step 2: Smoke test happy path**

```bash
TMPVAULT=$(mktemp -d)
mkdir -p "$TMPVAULT/Fleeting"
./engram quick --vault "$TMPVAULT" --slug "smoke-test" --content "# smoke

body.

**Source:** test."
ls "$TMPVAULT/Fleeting/"
cat "$TMPVAULT/Fleeting/"*.md
```

Expected: prints the written path on stdout; file `<date>.smoke-test.md` exists in `$TMPVAULT/Fleeting/` with the content above. Exit code 0.

- [ ] **Step 3: Smoke test stdin path**

```bash
TMPVAULT=$(mktemp -d)
mkdir -p "$TMPVAULT/Fleeting"
echo "# from stdin" | ./engram quick --vault "$TMPVAULT" --slug "stdin-smoke"
cat "$TMPVAULT/Fleeting/"*.md
```

Expected: file contains `# from stdin`. Exit 0.

- [ ] **Step 4: Smoke test no-overwrite error**

```bash
TMPVAULT=$(mktemp -d)
mkdir -p "$TMPVAULT/Fleeting"
./engram quick --vault "$TMPVAULT" --slug "dupe" --content "first"
./engram quick --vault "$TMPVAULT" --slug "dupe" --content "second"; echo "exit=$?"
```

Expected: second invocation prints an error mentioning the existing path; exit code non-zero. File still contains `first`.

- [ ] **Step 5: Smoke test missing vault flag and env**

```bash
unset ENGRAM_VAULT_DIR
./engram quick --slug "no-vault" --content "x"; echo "exit=$?"
```

Expected: error mentions `vault` and `ENGRAM_VAULT_DIR`; exit code non-zero.

- [ ] **Step 6: Cleanup any stray test files**

```bash
rm -rf /tmp/tmp.*  # or whatever your $TMPVAULT prefix was
```

(`mktemp -d` puts dirs under `$TMPDIR`; remove only the directories you created.)

- [ ] **Step 7: Commit (only if anything changed)**

If smoke tests revealed nothing to fix, no commit needed. If a fix was required, commit it under `fix(cli): ...`.

---

## Self-Review Checklist (post-write)

- [ ] **Spec coverage** — every spec section has a task: slug validation (1), vault resolution (2), content source (3), filename derivation (4), Fleeting/ check (5), runQuick orchestrator + no-overwrite + write (6), targ registration + production adapter (7), end-to-end smoke (8).
- [ ] **Type consistency** — `QuickArgs` defined once in `targets.go`; `QuickDeps` defined once in `quick.go`; all helper signatures referenced consistently across tasks.
- [ ] **No placeholders** — every step shows actual code or actual command + expected output. No "TBD" / "etc." / "similar to above".

## Out of plan (intentional)

- Skill update for `capturing-fleeting-notes` step 5 (Write tool → `engram quick`) — separate change, must follow `superpowers:writing-skills` TDD discipline. Track as a follow-up.
- `--batch` mode, body templating, file-locking, promotion integration — explicitly out per spec.
