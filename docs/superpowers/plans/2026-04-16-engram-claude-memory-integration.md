# Engram + Claude Code Memory Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend engram's recall pipeline to cross-search content from Claude Code's auto memory, installed skills, and CLAUDE.md hierarchy in addition to engram's own memories and session transcripts. Write path (`learn` / `remember`) is unchanged.

**Architecture:** New `internal/externalsources/` package owns discovery and per-invocation file caching. `internal/recall/` gains three new phase functions (auto memory, skills, CLAUDE.md+rules) inserted between the existing engram-memory and session phases. Each new phase uses index-then-extract: one cheap Haiku rank call selects relevant files, then per-file Haiku extract calls fill the shared 10 KB buffer in priority order. Buffer-full short-circuits downstream phases.

**Tech Stack:** Go 1.x (no CGO), engram's existing DI patterns (interfaces wired at the edges), gomega for assertions, `targ` for build/test. Spec: `docs/superpowers/specs/2026-04-16-engram-claude-memory-integration-design.md`.

---

## File Structure

**New package: `internal/externalsources/`**

| File | Responsibility |
|---|---|
| `externalsources.go` | Public API: `ExternalFile`, `Kind`, `Discover` orchestrator |
| `cache.go` | In-memory `FileCache` with lazy read + error caching |
| `claudemd.go` | CLAUDE.md ancestor walk + managed-policy paths |
| `imports.go` | `@path` import expansion (5-hop cap, cycle detection) |
| `rules.go` | `.claude/rules/*.md` discovery + `paths:` frontmatter filter |
| `frontmatter.go` | YAML frontmatter parser (shared by rules + skills) |
| `automemory.go` | Auto memory directory resolution (settings → slug → worktree fallback) + file enumeration |
| `slug.go` | Project slug computation matching Claude Code's algorithm |
| `skills.go` | Skill discovery (project + user + plugin cache) |
| Each above gets a `*_test.go` sibling | Unit tests with gomega + DI fakes |

**Modified files:**

| File | Change |
|---|---|
| `internal/recall/orchestrate.go` | Add `AutoMemorySource`, `SkillSource`, `ClaudeMdSource` interfaces + matching `OrchestratorOption`s; insert new phases into `recallModeB` |
| `internal/recall/phases.go` (NEW) | New file holding the three new phase functions to keep `orchestrate.go` lean |
| `internal/recall/phases_test.go` (NEW) | Phase-level unit tests with mocks |
| `internal/cli/cli.go` | Wire `externalsources.Discover` + cache into the orchestrator construction |
| `internal/cli/cli.go` adapter section | Add `osStatReader` if needed for testing externalsources I/O |

**New test fixture directory: `internal/externalsources/testdata/fixture-project/`** — used by the integration test.

---

## Build Sequence

The plan is grouped into six phases. Each task within a phase is independently committable. Phases mostly build on each other but later tasks within a phase usually do not depend on earlier ones in the same phase.

### Phase 0: Package skeleton + cache + frontmatter (Tasks 1-3)

Foundational types and the cache. Nothing depends on other engram packages yet.

### Phase 1: Per-source discovery (Tasks 4-9)

Each source type gets its own discovery function. Independent of recall.

### Phase 2: Discovery aggregator (Task 10)

One entry point that produces the full `[]ExternalFile` for a given cwd.

### Phase 3: Recall phases (Tasks 11-14)

New phase functions in `internal/recall/`. Pipeline integration last.

### Phase 4: CLI wiring + status output (Tasks 15-16)

Real `Discover` + `FileCache` constructed in CLI; status events for new phases.

### Phase 5: Integration test + cost regression guard (Tasks 17-18)

End-to-end fixture test and Haiku-call-count assertion.

### Phase 6: Smoke test (Task 19)

Manual verification on engram's own repo.

---

## Conventions for All Tasks

- **TDD strictly.** Red → Green → Commit. Refactor only when there's a real reason (duplication, unclear naming).
- **Use `targ`, not `go`.** Tests: `targ test`. Lint+coverage: `targ check-full`. Never `go test` directly.
- **Test package suffix.** All test files use `package foo_test` (blackbox). Project convention.
- **`t.Parallel()` everywhere.** Every test and subtest. Each subtest gets its own data — no shared mutable state.
- **gomega assertions.** `g := NewWithT(t)`. Use `g.Expect(err).NotTo(HaveOccurred())` followed by `if err != nil { return }` (nilaway compatibility — see `.claude/rules/go.md`).
- **Sentinel errors as package vars.** `var ErrXyz = errors.New("…")` — never inline `fmt.Errorf("…")` for sentinels.
- **Wrap errors with context.** `fmt.Errorf("doing X: %w", err)`.
- **Named constants over magic numbers.** `const maxImportHops = 5` not bare `5`.
- **Descriptive identifiers.** `frontmatter`, `discoveredFiles` — not `f`, `fs`.
- **Line length under 120 chars.**
- **Use `make([]T, 0, capacity)` when size is known.**
- **Commits use `AI-Used: [claude]` trailer** (not Co-Authored-By). Conventional Commits format. Specific files in `git add` (no `-A`/`.`).

---

## Phase 0 — Package skeleton + cache + frontmatter

### Task 1: Create externalsources package with core types

**Files:**
- Create: `internal/externalsources/externalsources.go`
- Create: `internal/externalsources/externalsources_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/externalsources_test.go`:
```go
// Package externalsources_test verifies the public types of the externalsources package.
package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestKindString(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.KindClaudeMd.String()).To(Equal("claude_md"))
	g.Expect(externalsources.KindRules.String()).To(Equal("rules"))
	g.Expect(externalsources.KindAutoMemory.String()).To(Equal("auto_memory"))
	g.Expect(externalsources.KindSkill.String()).To(Equal("skill"))
}

func TestExternalFileFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	file := externalsources.ExternalFile{
		Kind: externalsources.KindClaudeMd,
		Path: "/some/abs/path/CLAUDE.md",
	}

	g.Expect(file.Kind).To(Equal(externalsources.KindClaudeMd))
	g.Expect(file.Path).To(Equal("/some/abs/path/CLAUDE.md"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "package engram/internal/externalsources is not in std" or "undefined: externalsources.KindClaudeMd".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/externalsources.go`:
```go
// Package externalsources discovers and reads files outside engram's own
// memory store that recall/prepare may want to cross-search:
// CLAUDE.md hierarchy, .claude/rules/, Claude Code auto memory, and
// installed skills.
package externalsources

// Kind identifies which kind of external source an ExternalFile came from.
type Kind int

// Exported Kind constants.
const (
	KindUnknown Kind = iota
	KindClaudeMd
	KindRules
	KindAutoMemory
	KindSkill
)

// String returns the canonical lowercase identifier for a Kind, used in
// status output and DUPLICATE response shapes.
func (k Kind) String() string {
	switch k {
	case KindClaudeMd:
		return "claude_md"
	case KindRules:
		return "rules"
	case KindAutoMemory:
		return "auto_memory"
	case KindSkill:
		return "skill"
	case KindUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// ExternalFile names one discovered file along with its source kind.
// Discovery produces a slice of these; phase extractors consume them.
type ExternalFile struct {
	Kind Kind
	Path string // absolute path
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS (no lint, no nilaway, no coverage drops in the new package — empty package may need a coverage exclusion if check-full enforces a per-package floor; if it complains, follow the existing convention used in another small package like `tomlwriter` or `tokenresolver`).

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/externalsources.go internal/externalsources/externalsources_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): add package skeleton with Kind and ExternalFile types

AI-Used: [claude]
EOF
)"
```

---

### Task 2: Add per-invocation FileCache

**Files:**
- Create: `internal/externalsources/cache.go`
- Create: `internal/externalsources/cache_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/cache_test.go`:
```go
package externalsources_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestFileCache_FirstReadReadsThrough(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	calls := 0
	reader := func(path string) ([]byte, error) {
		calls++
		return []byte("content of " + path), nil
	}

	cache := externalsources.NewFileCache(reader)

	content, err := cache.Read("/abs/path.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(content)).To(Equal("content of /abs/path.md"))
	g.Expect(calls).To(Equal(1))
}

func TestFileCache_RepeatReadsHitCache(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	calls := 0
	reader := func(_ string) ([]byte, error) {
		calls++
		return []byte("body"), nil
	}

	cache := externalsources.NewFileCache(reader)

	for range 3 {
		_, err := cache.Read("/p.md")
		g.Expect(err).NotTo(HaveOccurred())
	}

	g.Expect(calls).To(Equal(1))
}

func TestFileCache_ErrorIsCached(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	calls := 0
	wantErr := errors.New("permission denied")
	reader := func(_ string) ([]byte, error) {
		calls++
		return nil, wantErr
	}

	cache := externalsources.NewFileCache(reader)

	_, err1 := cache.Read("/no.md")
	_, err2 := cache.Read("/no.md")

	g.Expect(err1).To(MatchError(wantErr))
	g.Expect(err2).To(MatchError(wantErr))
	g.Expect(calls).To(Equal(1))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.NewFileCache".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/cache.go`:
```go
package externalsources

// ReaderFunc reads a file's bytes given an absolute path. Wired at the edge
// to os.ReadFile in production; replaced by a fake in tests.
type ReaderFunc func(path string) ([]byte, error)

// FileCache memoizes file contents (and errors) for the duration of a single
// engram process invocation. There is no cross-invocation persistence — the
// cache is dropped when the process exits.
type FileCache struct {
	reader ReaderFunc
	cache  map[string]cachedRead
}

type cachedRead struct {
	content []byte
	err     error
}

// NewFileCache creates a FileCache backed by the given ReaderFunc.
func NewFileCache(reader ReaderFunc) *FileCache {
	return &FileCache{
		reader: reader,
		cache:  make(map[string]cachedRead),
	}
}

// Read returns the file's bytes, reading through to the underlying reader
// only on first access for a given path. Errors are cached too — repeated
// reads of an unreadable file do not retry.
func (c *FileCache) Read(path string) ([]byte, error) {
	if entry, ok := c.cache[path]; ok {
		return entry.content, entry.err
	}

	content, err := c.reader(path)
	c.cache[path] = cachedRead{content: content, err: err}

	return content, err
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/cache.go internal/externalsources/cache_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): per-invocation FileCache with cached errors

AI-Used: [claude]
EOF
)"
```

---

### Task 3: YAML frontmatter parser

**Files:**
- Create: `internal/externalsources/frontmatter.go`
- Create: `internal/externalsources/frontmatter_test.go`

The parser is intentionally minimal — it only needs the fields engram cares about (`name`, `description`, `paths`). Pulling in `gopkg.in/yaml.v3` is overkill for that surface; use a small string-based parser.

- [ ] **Step 1: Write the failing test**

`internal/externalsources/frontmatter_test.go`:
```go
package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte("# Just a markdown file\n\nNo frontmatter here.\n")

	fm, rest := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(BeEmpty())
	g.Expect(fm.Description).To(BeEmpty())
	g.Expect(fm.Paths).To(BeEmpty())
	g.Expect(string(rest)).To(Equal(string(body)))
}

func TestParseFrontmatter_NameAndDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
name: prepare
description: Use before starting new work to load context
---

# Skill body
`)

	fm, rest := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(Equal("prepare"))
	g.Expect(fm.Description).To(Equal("Use before starting new work to load context"))
	g.Expect(string(rest)).To(ContainSubstring("# Skill body"))
	g.Expect(string(rest)).NotTo(ContainSubstring("---"))
}

func TestParseFrontmatter_PathsList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
paths:
  - "src/api/**/*.ts"
  - "lib/**/*.ts"
---

content
`)

	fm, _ := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Paths).To(ConsistOf("src/api/**/*.ts", "lib/**/*.ts"))
}

func TestParseFrontmatter_FoldedDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
name: learn
description: >
  Use after completing a task,
  finishing work, or changing direction.
---

body
`)

	fm, _ := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(Equal("learn"))
	g.Expect(fm.Description).To(ContainSubstring("Use after completing a task"))
	g.Expect(fm.Description).To(ContainSubstring("changing direction."))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.ParseFrontmatter".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/frontmatter.go`:
```go
package externalsources

import (
	"bytes"
	"strings"
)

// Frontmatter holds the YAML frontmatter fields engram extracts from rules
// and skill markdown files. Only fields engram actually uses are populated.
type Frontmatter struct {
	Name        string
	Description string
	Paths       []string
}

// ParseFrontmatter splits a markdown file into its frontmatter (if any) and
// the remaining body bytes. Files without a leading "---" line return an
// empty Frontmatter and the original body unchanged.
//
// This is intentionally a small, hand-rolled parser covering only the fields
// engram cares about (name, description, paths). It supports literal-scalar
// strings, the ">" folded-block style for description, and the "- item"
// list style for paths.
func ParseFrontmatter(body []byte) (Frontmatter, []byte) {
	const fenceMarker = "---"

	if !bytes.HasPrefix(body, []byte(fenceMarker+"\n")) {
		return Frontmatter{}, body
	}

	rest := body[len(fenceMarker)+1:]

	endIdx := bytes.Index(rest, []byte("\n"+fenceMarker+"\n"))
	if endIdx < 0 {
		return Frontmatter{}, body
	}

	yamlBlock := string(rest[:endIdx])
	remainder := rest[endIdx+len("\n"+fenceMarker+"\n"):]

	return parseYAMLBlock(yamlBlock), remainder
}

func parseYAMLBlock(yamlBlock string) Frontmatter {
	var (
		fm                  Frontmatter
		inFoldedDescription bool
		inPathsList         bool
		foldedLines         []string
	)

	for _, raw := range strings.Split(yamlBlock, "\n") {
		trimmed := strings.TrimSpace(raw)

		if inFoldedDescription {
			if strings.HasPrefix(raw, "  ") || strings.HasPrefix(raw, "\t") {
				foldedLines = append(foldedLines, trimmed)

				continue
			}

			fm.Description = strings.Join(foldedLines, " ")
			inFoldedDescription = false
			foldedLines = nil
		}

		if inPathsList {
			if strings.HasPrefix(trimmed, "- ") {
				fm.Paths = append(fm.Paths, strings.Trim(strings.TrimPrefix(trimmed, "- "), `"'`))

				continue
			}

			inPathsList = false
		}

		switch {
		case strings.HasPrefix(trimmed, "name:"):
			fm.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		case trimmed == "description: >":
			inFoldedDescription = true
		case strings.HasPrefix(trimmed, "description:"):
			fm.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
		case trimmed == "paths:":
			inPathsList = true
		}
	}

	if inFoldedDescription {
		fm.Description = strings.Join(foldedLines, " ")
	}

	return fm
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/frontmatter.go internal/externalsources/frontmatter_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): minimal YAML frontmatter parser for skills + rules

AI-Used: [claude]
EOF
)"
```

---

## Phase 1 — Per-source discovery

### Task 4: CLAUDE.md ancestor walk

**Files:**
- Create: `internal/externalsources/claudemd.go`
- Create: `internal/externalsources/claudemd_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/claudemd_test.go`:
```go
package externalsources_test

import (
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverClaudeMd_AncestorWalk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(path string) (bool, error) {
		// Pretend CLAUDE.md exists at /a/b and /a, plus a CLAUDE.local.md at /a/b.
		switch path {
		case "/a/b/CLAUDE.md", "/a/CLAUDE.md", "/a/b/CLAUDE.local.md":
			return true, nil
		default:
			return false, nil
		}
	}

	files := externalsources.DiscoverClaudeMd("/a/b", "/home/user", statFn)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		g.Expect(f.Kind).To(Equal(externalsources.KindClaudeMd))
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElement("/a/b/CLAUDE.md"))
	g.Expect(paths).To(ContainElement("/a/b/CLAUDE.local.md"))
	g.Expect(paths).To(ContainElement("/a/CLAUDE.md"))
}

func TestDiscoverClaudeMd_IncludesUserScope(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(path string) (bool, error) {
		return path == "/home/user/.claude/CLAUDE.md", nil
	}

	files := externalsources.DiscoverClaudeMd("/somewhere", "/home/user", statFn)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElement("/home/user/.claude/CLAUDE.md"))
}

func TestDiscoverClaudeMd_IncludesManagedPolicy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	wantPath := externalsources.ManagedPolicyPath(runtime.GOOS)
	g.Expect(wantPath).NotTo(BeEmpty())

	statFn := func(path string) (bool, error) {
		return path == wantPath, nil
	}

	files := externalsources.DiscoverClaudeMd("/somewhere", "/home/user", statFn)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElement(wantPath))
}

func TestDiscoverClaudeMd_NoFilesPresent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(_ string) (bool, error) { return false, nil }

	files := externalsources.DiscoverClaudeMd("/x/y", "/home/user", statFn)
	g.Expect(files).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.DiscoverClaudeMd" and "undefined: externalsources.ManagedPolicyPath".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/claudemd.go`:
```go
package externalsources

import (
	"path/filepath"
)

// StatFunc reports whether a file exists at the given absolute path.
// Wired at the edge to a thin os.Stat wrapper.
type StatFunc func(path string) (exists bool, err error)

// claudeMdFilenames is the set of CLAUDE.md variants to look for at each
// ancestor directory. Within a directory, CLAUDE.local.md is appended after
// CLAUDE.md so that conflicting personal notes win, matching Claude Code.
var claudeMdFilenames = []string{"CLAUDE.md", "CLAUDE.local.md"}

// ManagedPolicyPath returns the documented system-wide CLAUDE.md location for
// the given GOOS, or empty string for an unrecognized OS.
func ManagedPolicyPath(goos string) string {
	switch goos {
	case "darwin":
		return "/Library/Application Support/ClaudeCode/CLAUDE.md"
	case "linux":
		return "/etc/claude-code/CLAUDE.md"
	case "windows":
		return `C:\Program Files\ClaudeCode\CLAUDE.md`
	default:
		return ""
	}
}

// DiscoverClaudeMd walks ancestors from cwd to "/", collecting any CLAUDE.md
// or CLAUDE.local.md it finds. It also adds the user-scope CLAUDE.md
// (~/.claude/CLAUDE.md) and the managed-policy CLAUDE.md if present.
//
// Files that do not exist are silently skipped. Stat errors are also skipped
// (logged at the caller's discretion).
func DiscoverClaudeMd(cwd, home string, statFn StatFunc) []ExternalFile {
	files := make([]ExternalFile, 0, defaultClaudeMdCapacity)

	// Walk ancestors from cwd up to root.
	dir := cwd
	for {
		for _, name := range claudeMdFilenames {
			candidate := filepath.Join(dir, name)
			if exists, _ := statFn(candidate); exists {
				files = append(files, ExternalFile{Kind: KindClaudeMd, Path: candidate})
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	// User scope.
	if home != "" {
		userMd := filepath.Join(home, ".claude", "CLAUDE.md")
		if exists, _ := statFn(userMd); exists {
			files = append(files, ExternalFile{Kind: KindClaudeMd, Path: userMd})
		}
	}

	// Managed policy.
	if managed := ManagedPolicyPath(runtimeGOOS()); managed != "" {
		if exists, _ := statFn(managed); exists {
			files = append(files, ExternalFile{Kind: KindClaudeMd, Path: managed})
		}
	}

	return files
}

// defaultClaudeMdCapacity is a heuristic preallocation hint — most projects
// produce 3-6 CLAUDE.md files across ancestors + user + managed.
const defaultClaudeMdCapacity = 8
```

- [ ] **Step 4: Add a tiny `runtimeGOOS()` indirection for testability**

We could pass GOOS through the API, but production callers always want runtime.GOOS. Keep the indirection internal:

Append to `internal/externalsources/claudemd.go`:
```go
import "runtime"

// runtimeGOOS is a package-level alias for runtime.GOOS. Defined as a var so
// it can be overridden in tests via export_test.go if needed.
var runtimeGOOSFn = func() string { return runtime.GOOS }

func runtimeGOOS() string {
	return runtimeGOOSFn()
}
```

(Update the import section to combine both `path/filepath` and `runtime`.)

- [ ] **Step 5: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 6: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/externalsources/claudemd.go internal/externalsources/claudemd_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): walk ancestors + user + managed-policy CLAUDE.md

AI-Used: [claude]
EOF
)"
```

---

### Task 5: `@path` import expansion with cycle detection

**Files:**
- Create: `internal/externalsources/imports.go`
- Create: `internal/externalsources/imports_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/imports_test.go`:
```go
package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestExpandImports_NoImports(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a/CLAUDE.md": []byte("Just text, no imports.\n"),
	}

	imports := externalsources.ExpandImports("/a/CLAUDE.md", fakeReader(contents))
	g.Expect(imports).To(BeEmpty())
}

func TestExpandImports_RelativePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a/CLAUDE.md":           []byte("See @docs/git.md for details.\n"),
		"/a/docs/git.md":         []byte("Git workflow.\n"),
	}

	imports := externalsources.ExpandImports("/a/CLAUDE.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, f := range imports {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ConsistOf("/a/docs/git.md"))
}

func TestExpandImports_RecursiveAndCycleSafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a.md": []byte("@b.md\n"),
		"/b.md": []byte("@c.md\n"),
		"/c.md": []byte("@a.md\n"), // cycle back to a.md
	}

	imports := externalsources.ExpandImports("/a.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, f := range imports {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ConsistOf("/b.md", "/c.md"))
}

func TestExpandImports_DepthCappedAt5Hops(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/0.md": []byte("@1.md\n"),
		"/1.md": []byte("@2.md\n"),
		"/2.md": []byte("@3.md\n"),
		"/3.md": []byte("@4.md\n"),
		"/4.md": []byte("@5.md\n"),
		"/5.md": []byte("@6.md\n"), // 6th hop should NOT be followed
		"/6.md": []byte("never reached\n"),
	}

	imports := externalsources.ExpandImports("/0.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, f := range imports {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElements("/1.md", "/2.md", "/3.md", "/4.md", "/5.md"))
	g.Expect(paths).NotTo(ContainElement("/6.md"))
}

// fakeReader returns a ReaderFunc backed by an in-memory map.
func fakeReader(contents map[string][]byte) externalsources.ReaderFunc {
	return func(path string) ([]byte, error) {
		body, ok := contents[path]
		if !ok {
			return nil, nil
		}

		return body, nil
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.ExpandImports".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/imports.go`:
```go
package externalsources

import (
	"path/filepath"
	"regexp"
	"strings"
)

// maxImportHops mirrors the documented Claude Code 5-hop import depth cap.
const maxImportHops = 5

// importPattern matches `@path/to/file.md` references.
//
// Constraints intentionally narrow:
//   - must be preceded by start-of-line, whitespace, or "(" so we don't pick
//     up email addresses
//   - path segments may not contain whitespace or "@"
var importPattern = regexp.MustCompile(`(?:^|[\s(])@([^\s@()]+)`)

// ExpandImports walks @path imports starting from the given file, returning
// every distinct file reached (excluding the starting file itself). Cycles
// are detected and broken; depth is capped at maxImportHops.
//
// The reader is consulted to read each imported file's contents so further
// imports can be discovered. The same FileCache the rest of discovery uses
// should back this — repeated reads of the same file are free.
func ExpandImports(startPath string, reader ReaderFunc) []ExternalFile {
	visited := map[string]bool{startPath: true}

	var (
		discovered []ExternalFile
		queue      = []importNode{{path: startPath, depth: 0}}
	)

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.depth >= maxImportHops {
			continue
		}

		body, err := reader(node.path)
		if err != nil || body == nil {
			continue
		}

		for _, match := range importPattern.FindAllStringSubmatch(string(body), -1) {
			rawPath := match[1]
			absPath := resolveImportPath(node.path, rawPath)

			if visited[absPath] {
				continue
			}

			visited[absPath] = true

			discovered = append(discovered, ExternalFile{Kind: KindClaudeMd, Path: absPath})
			queue = append(queue, importNode{path: absPath, depth: node.depth + 1})
		}
	}

	return discovered
}

type importNode struct {
	path  string
	depth int
}

// resolveImportPath returns the absolute path for a @import target relative
// to the file that contains the import. Absolute paths are returned as-is.
// "~" is expanded against the user's home directory if the home is known
// (callers can pre-substitute or accept "~/..." stays literal — we keep this
// minimal and treat "~" paths as absolute strings for now).
func resolveImportPath(containingFile, importTarget string) string {
	if strings.HasPrefix(importTarget, "/") || strings.HasPrefix(importTarget, "~") {
		return importTarget
	}

	return filepath.Join(filepath.Dir(containingFile), importTarget)
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/imports.go internal/externalsources/imports_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): expand @path imports with 5-hop cap and cycle detection

AI-Used: [claude]
EOF
)"
```

---

### Task 6: Rules discovery with `paths:` glob filter

**Files:**
- Create: `internal/externalsources/rules.go`
- Create: `internal/externalsources/rules_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/rules_test.go`:
```go
package externalsources_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverRules_FilesWithoutPathsAlwaysIncluded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mdContent := []byte("# Always-on rule\nNo frontmatter.\n")

	contents := map[string][]byte{
		"/proj/.claude/rules/always.md": mdContent,
	}

	walker := func(root string) []string {
		if root == "/proj/.claude/rules" {
			return []string{"/proj/.claude/rules/always.md"}
		}

		return nil
	}

	matchAny := func(_ []string) bool { return false }

	files := externalsources.DiscoverRules("/proj", "/home/user", walker, fakeReader(contents), matchAny)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		g.Expect(f.Kind).To(Equal(externalsources.KindRules))
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ConsistOf("/proj/.claude/rules/always.md"))
}

func TestDiscoverRules_PathsGlobIncludedWhenAtLeastOneFileMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/proj/.claude/rules/api.md": []byte(`---
paths:
  - "src/api/**/*.ts"
---
API rule
`),
	}

	walker := func(root string) []string {
		if root == "/proj/.claude/rules" {
			return []string{"/proj/.claude/rules/api.md"}
		}

		return nil
	}

	matchAny := func(globs []string) bool {
		// Pretend at least one file under cwd matches "src/api/**/*.ts".
		return len(globs) > 0
	}

	files := externalsources.DiscoverRules("/proj", "/home/user", walker, fakeReader(contents), matchAny)
	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.FromSlash("/proj/.claude/rules/api.md")))
}

func TestDiscoverRules_PathsGlobExcludedWhenNoFilesMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/proj/.claude/rules/api.md": []byte(`---
paths:
  - "src/api/**/*.ts"
---
API rule
`),
	}

	walker := func(_ string) []string { return []string{"/proj/.claude/rules/api.md"} }
	matchAny := func(_ []string) bool { return false }

	files := externalsources.DiscoverRules("/proj", "/home/user", walker, fakeReader(contents), matchAny)
	g.Expect(files).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.DiscoverRules".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/rules.go`:
```go
package externalsources

import (
	"path/filepath"
)

// MdWalker returns absolute paths to all *.md files found recursively under
// root. Wired at the edge to filepath.WalkDir; replaced by a fake in tests.
type MdWalker func(root string) []string

// GlobMatcher reports whether at least one path under cwd matches any of the
// given globs. Wired at the edge to a small filepath.Glob loop; replaced by
// a fake in tests.
type GlobMatcher func(globs []string) (anyMatch bool)

// DiscoverRules walks <project>/.claude/rules and ~/.claude/rules collecting
// markdown rule files. Files with a `paths:` frontmatter are included only
// when at least one file under cwd matches the glob (per the engram-specific
// adaptation noted in the spec).
func DiscoverRules(
	cwd, home string,
	walker MdWalker,
	reader ReaderFunc,
	matchAny GlobMatcher,
) []ExternalFile {
	files := make([]ExternalFile, 0, defaultRulesCapacity)

	roots := []string{filepath.Join(cwd, ".claude", "rules")}
	if home != "" {
		roots = append(roots, filepath.Join(home, ".claude", "rules"))
	}

	for _, root := range roots {
		for _, mdPath := range walker(root) {
			if shouldIncludeRule(mdPath, reader, matchAny) {
				files = append(files, ExternalFile{Kind: KindRules, Path: mdPath})
			}
		}
	}

	return files
}

func shouldIncludeRule(path string, reader ReaderFunc, matchAny GlobMatcher) bool {
	body, err := reader(path)
	if err != nil || body == nil {
		return false
	}

	fm, _ := ParseFrontmatter(body)
	if len(fm.Paths) == 0 {
		return true
	}

	return matchAny(fm.Paths)
}

const defaultRulesCapacity = 8
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/rules.go internal/externalsources/rules_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): discover .claude/rules with paths-glob filter

AI-Used: [claude]
EOF
)"
```

---

### Task 7: Project slug computation

**Files:**
- Create: `internal/externalsources/slug.go`
- Create: `internal/externalsources/slug_test.go`

The slug must match Claude Code's algorithm. Based on observation, Claude Code replaces `/` with `-` in absolute paths (e.g., `/Users/joe/repos/engram` → `-Users-joe-repos-engram`). Verify by inspecting `~/.claude/projects/` on the dev machine — the existing project subdirectories show the live encoding.

- [ ] **Step 1: Write the failing test**

`internal/externalsources/slug_test.go`:
```go
package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestProjectSlug_DashSubstitution(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.ProjectSlug("/Users/joe/repos/engram")).
		To(Equal("-Users-joe-repos-engram"))
	g.Expect(externalsources.ProjectSlug("/home/alice/work")).
		To(Equal("-home-alice-work"))
}

func TestProjectSlug_RootDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.ProjectSlug("/")).To(Equal("-"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.ProjectSlug".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/slug.go`:
```go
package externalsources

import "strings"

// ProjectSlug returns the slugified form of an absolute path that Claude Code
// uses when constructing the auto-memory directory path
// (~/.claude/projects/<slug>/memory/).
//
// Claude Code's algorithm: replace every "/" in the absolute path with "-".
// "/Users/joe/repos/engram" → "-Users-joe-repos-engram".
//
// Verified against the actual ~/.claude/projects/ subdirectories on the dev
// machine. If Claude Code changes this algorithm, this function (and the
// auto memory resolver) must be updated together.
func ProjectSlug(absPath string) string {
	return strings.ReplaceAll(absPath, "/", "-")
}
```

- [ ] **Step 4: Verify against real `~/.claude/projects/` layout**

Run `ls ~/.claude/projects/ | head -5` on the dev machine. Confirm the output matches `ProjectSlug` applied to known project paths. If it does not, update the algorithm and add a regression test for the corrected case.

- [ ] **Step 5: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 6: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/externalsources/slug.go internal/externalsources/slug_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): project slug for Claude Code auto-memory paths

AI-Used: [claude]
EOF
)"
```

---

### Task 8: Auto memory directory resolution + file enumeration

**Files:**
- Create: `internal/externalsources/automemory.go`
- Create: `internal/externalsources/automemory_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/automemory_test.go`:
```go
package externalsources_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverAutoMemory_HonorsAutoMemoryDirectorySetting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "/custom/memdir", true }

	dirLister := func(dir string) ([]string, error) {
		if dir == "/custom/memdir" {
			return []string{
				"/custom/memdir/MEMORY.md",
				"/custom/memdir/debugging.md",
			}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(
		"/some/cwd", "/home/user", "", settings, dirLister,
	)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		g.Expect(f.Kind).To(Equal(externalsources.KindAutoMemory))
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ConsistOf(
		"/custom/memdir/MEMORY.md",
		"/custom/memdir/debugging.md",
	))
}

func TestDiscoverAutoMemory_DefaultSlugPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }

	wantDir := filepath.Join("/home/user", ".claude", "projects", "-proj-cwd", "memory")
	dirLister := func(dir string) ([]string, error) {
		if dir == wantDir {
			return []string{filepath.Join(wantDir, "MEMORY.md")}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(
		"/proj/cwd", "/home/user", "", settings, dirLister,
	)

	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.Join(wantDir, "MEMORY.md")))
}

func TestDiscoverAutoMemory_WorktreeFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }

	cwdSlugDir := filepath.Join("/home/user", ".claude", "projects", "-proj-wt", "memory")
	mainSlugDir := filepath.Join("/home/user", ".claude", "projects", "-proj", "memory")

	dirLister := func(dir string) ([]string, error) {
		switch dir {
		case cwdSlugDir:
			return nil, nil // worktree slug has no dir
		case mainSlugDir:
			return []string{filepath.Join(mainSlugDir, "MEMORY.md")}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(
		"/proj/wt", "/home/user", "/proj", settings, dirLister,
	)

	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.Join(mainSlugDir, "MEMORY.md")))
}

func TestDiscoverAutoMemory_NoDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }
	dirLister := func(_ string) ([]string, error) { return nil, nil }

	files := externalsources.DiscoverAutoMemory(
		"/no/such/proj", "/home/user", "", settings, dirLister,
	)

	g.Expect(files).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.DiscoverAutoMemory".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/automemory.go`:
```go
package externalsources

import (
	"path/filepath"
)

// AutoMemorySettingsFunc returns the configured `autoMemoryDirectory` setting,
// if any, with a found flag. Wired at the edge to a settings.json reader.
type AutoMemorySettingsFunc func() (dir string, found bool)

// DirListerFunc lists files in a directory, returning absolute paths. An
// error or non-existent directory should return (nil, nil) so callers can
// treat both as "no contents". Wired at the edge to os.ReadDir.
type DirListerFunc func(dir string) ([]string, error)

// DiscoverAutoMemory resolves the auto-memory directory in this order:
//
//  1. autoMemoryDirectory setting (if found and the dir has files);
//  2. ~/.claude/projects/<slug(cwd)>/memory/ (default per-project location);
//  3. ~/.claude/projects/<slug(mainRepoRoot)>/memory/ — worktree fallback,
//     used only when (2) yields no files and mainRepoRoot != "".
//
// Returns the *.md files in the resolved directory as ExternalFile entries.
// An empty result is normal — auto memory is opt-in and may not exist yet.
func DiscoverAutoMemory(
	cwd, home, mainRepoRoot string,
	settings AutoMemorySettingsFunc,
	dirLister DirListerFunc,
) []ExternalFile {
	if dir, ok := settings(); ok && dir != "" {
		if files := listAutoMemoryDir(dir, dirLister); len(files) > 0 {
			return files
		}
	}

	if home == "" {
		return nil
	}

	cwdDir := filepath.Join(home, ".claude", "projects", ProjectSlug(cwd), "memory")
	if files := listAutoMemoryDir(cwdDir, dirLister); len(files) > 0 {
		return files
	}

	if mainRepoRoot == "" || mainRepoRoot == cwd {
		return nil
	}

	mainDir := filepath.Join(home, ".claude", "projects", ProjectSlug(mainRepoRoot), "memory")

	return listAutoMemoryDir(mainDir, dirLister)
}

func listAutoMemoryDir(dir string, dirLister DirListerFunc) []ExternalFile {
	paths, err := dirLister(dir)
	if err != nil || len(paths) == 0 {
		return nil
	}

	files := make([]ExternalFile, 0, len(paths))

	for _, path := range paths {
		if filepath.Ext(path) == ".md" {
			files = append(files, ExternalFile{Kind: KindAutoMemory, Path: path})
		}
	}

	return files
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/automemory.go internal/externalsources/automemory_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): resolve auto memory dir and enumerate topic files

AI-Used: [claude]
EOF
)"
```

---

### Task 9: Skill discovery (project + user + plugin cache)

**Files:**
- Create: `internal/externalsources/skills.go`
- Create: `internal/externalsources/skills_test.go`

Plugin cache layout per Claude Code convention: `~/.claude/plugins/cache/<plugin>/<version>/skills/<skill-name>/SKILL.md`. Project skills: `<cwd>/.claude/skills/<name>/SKILL.md`. User skills: `~/.claude/skills/<name>/SKILL.md`.

- [ ] **Step 1: Write the failing test**

`internal/externalsources/skills_test.go`:
```go
package externalsources_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverSkills_AllThreeRoots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cwdSkill := filepath.Join("/proj", ".claude", "skills", "projlocal", "SKILL.md")
	userSkill := filepath.Join("/home/user", ".claude", "skills", "personal", "SKILL.md")
	pluginSkill := filepath.Join(
		"/home/user", ".claude", "plugins", "cache", "core", "1.0.0",
		"skills", "core-skill", "SKILL.md",
	)

	skillFinder := func(root string) []string {
		switch root {
		case filepath.Join("/proj", ".claude", "skills"):
			return []string{cwdSkill}
		case filepath.Join("/home/user", ".claude", "skills"):
			return []string{userSkill}
		case filepath.Join("/home/user", ".claude", "plugins", "cache"):
			return []string{pluginSkill}
		}

		return nil
	}

	files := externalsources.DiscoverSkills("/proj", "/home/user", skillFinder)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		g.Expect(f.Kind).To(Equal(externalsources.KindSkill))
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ConsistOf(cwdSkill, userSkill, pluginSkill))
}

func TestDiscoverSkills_NoneFoundReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := externalsources.DiscoverSkills("/proj", "/home/user", func(_ string) []string { return nil })
	g.Expect(files).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.DiscoverSkills".

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/skills.go`:
```go
package externalsources

import (
	"path/filepath"
)

// SkillFinder returns absolute paths to every SKILL.md found anywhere under
// root (recursive). Wired at the edge to filepath.WalkDir; replaced by a
// fake in tests.
type SkillFinder func(root string) []string

// DiscoverSkills returns SKILL.md files from the three documented locations:
// project, user, and plugin cache. Each root is walked independently; an
// empty root simply contributes nothing.
func DiscoverSkills(cwd, home string, finder SkillFinder) []ExternalFile {
	roots := []string{filepath.Join(cwd, ".claude", "skills")}

	if home != "" {
		roots = append(roots,
			filepath.Join(home, ".claude", "skills"),
			filepath.Join(home, ".claude", "plugins", "cache"),
		)
	}

	files := make([]ExternalFile, 0, defaultSkillsCapacity)

	for _, root := range roots {
		for _, path := range finder(root) {
			files = append(files, ExternalFile{Kind: KindSkill, Path: path})
		}
	}

	return files
}

const defaultSkillsCapacity = 32
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/skills.go internal/externalsources/skills_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): discover skills from project, user, and plugin cache

AI-Used: [claude]
EOF
)"
```

---

## Phase 2 — Discovery aggregator

### Task 10: `Discover` aggregator function

**Files:**
- Create: `internal/externalsources/discover.go`
- Create: `internal/externalsources/discover_test.go`

- [ ] **Step 1: Write the failing test**

`internal/externalsources/discover_test.go`:
```go
package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscover_CombinesAllSourceKinds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := externalsources.DiscoverDeps{
		CWD:          "/proj",
		Home:         "/home/user",
		MainRepoRoot: "",
		StatFn:       func(_ string) (bool, error) { return true },
		Reader:       fakeReader(map[string][]byte{}),
		MdWalker:     func(_ string) []string { return nil },
		MatchAny:     func(_ []string) bool { return false },
		Settings:     func() (string, bool) { return "", false },
		DirLister:    func(_ string) ([]string, error) { return nil, nil },
		SkillFinder:  func(_ string) []string { return nil },
	}

	// With every source returning empty, Discover should still succeed and
	// return at least the user-scope CLAUDE.md (StatFn returns true for all).
	files := externalsources.Discover(deps)
	g.Expect(files).NotTo(BeNil())
}
```

(More extensive Discover-aggregator tests come from the integration test in Task 17.)

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: externalsources.Discover" or similar.

- [ ] **Step 3: Write minimal implementation**

`internal/externalsources/discover.go`:
```go
package externalsources

// DiscoverDeps bundles the function-typed dependencies Discover needs.
// Wiring this as a struct makes the public API stable as new optional
// dependencies are added.
type DiscoverDeps struct {
	CWD          string
	Home         string
	MainRepoRoot string // empty if not in a worktree (or main repo == cwd)
	StatFn       StatFunc
	Reader       ReaderFunc
	MdWalker     MdWalker
	MatchAny     GlobMatcher
	Settings     AutoMemorySettingsFunc
	DirLister    DirListerFunc
	SkillFinder  SkillFinder
}

// Discover runs each per-source discovery and concatenates the results.
// Within the result, ordering is: CLAUDE.md (ancestors → user → managed) →
// imports → rules → auto memory → skills.
//
// The returned slice is the input to the recall pipeline phases; ordering
// here does NOT determine phase priority — that is set in
// internal/recall/orchestrate.go.
func Discover(deps DiscoverDeps) []ExternalFile {
	files := DiscoverClaudeMd(deps.CWD, deps.Home, deps.StatFn)

	for _, base := range files {
		files = append(files, ExpandImports(base.Path, deps.Reader)...)
	}

	files = append(files, DiscoverRules(deps.CWD, deps.Home, deps.MdWalker, deps.Reader, deps.MatchAny)...)
	files = append(files, DiscoverAutoMemory(
		deps.CWD, deps.Home, deps.MainRepoRoot, deps.Settings, deps.DirLister,
	)...)
	files = append(files, DiscoverSkills(deps.CWD, deps.Home, deps.SkillFinder)...)

	return files
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/discover.go internal/externalsources/discover_test.go
git commit -m "$(cat <<'EOF'
feat(externalsources): Discover aggregator combining all source kinds

AI-Used: [claude]
EOF
)"
```

---

## Phase 3 — Recall pipeline phases

### Task 11: Auto memory phase (rank + extract)

**Files:**
- Create: `internal/recall/phases.go`
- Create: `internal/recall/phases_test.go`
- Modify: `internal/recall/orchestrate.go` (add new interfaces only — no pipeline integration yet)

- [ ] **Step 1: Write the failing test**

`internal/recall/phases_test.go`:
```go
package recall_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
	"engram/internal/recall"
)

func TestExtractFromAutoMemory_RanksAndExtractsUntilBufferFills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/debugging.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/architecture.md"},
	}

	contents := map[string][]byte{
		"/m/MEMORY.md":     []byte("Index: debugging.md, architecture.md"),
		"/m/debugging.md":  []byte("debugging body"),
		"/m/architecture.md": []byte("architecture body"),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	// fakeRanker returns "debugging.md" first then "architecture.md".
	// fakeExtractor echoes the body.
	summarizer := &phasePipelineSummarizer{
		rankResponse: "debugging.md\narchitecture.md",
		extractMap: map[string]string{
			"debugging body":    "debugging snippet",
			"architecture body": "architecture snippet",
		},
	}

	var buffer strings.Builder

	const cap1 = 1024

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, cap1,
	)

	g.Expect(bytesUsed).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(ContainSubstring("debugging snippet"))
}

func TestExtractFromAutoMemory_NoMemoryIndexFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/debugging.md"},
	}

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("body"), nil
	})

	summarizer := &phasePipelineSummarizer{}

	var buffer strings.Builder

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(bytesUsed).To(Equal(0))
	g.Expect(buffer.String()).To(BeEmpty())
}

// phasePipelineSummarizer satisfies recall.SummarizerI for phase-level tests.
type phasePipelineSummarizer struct {
	rankResponse string
	extractMap   map[string]string
}

func (s *phasePipelineSummarizer) ExtractRelevant(_ context.Context, content, query string) (string, error) {
	if strings.HasPrefix(query, "Rank") {
		return s.rankResponse, nil
	}

	return s.extractMap[content], nil
}

func (s *phasePipelineSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	return content, nil
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: recall.ExtractFromAutoMemory".

- [ ] **Step 3: Write minimal implementation**

`internal/recall/phases.go`:
```go
package recall

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"engram/internal/externalsources"
)

// ExtractFromAutoMemory runs phase 2 of the recall pipeline against Claude
// Code's auto memory directory.
//
// It uses MEMORY.md as the index for one Haiku rank call, then iterates the
// returned topic files in rank order, Haiku-extracting each into the buffer
// until cap is reached. Returns total bytes appended this phase.
func ExtractFromAutoMemory(
	ctx context.Context,
	files []externalsources.ExternalFile,
	query string,
	cache *externalsources.FileCache,
	summarizer SummarizerI,
	buffer *strings.Builder,
	bytesUsed, cap int,
) int {
	if summarizer == nil || cache == nil || bytesUsed >= cap {
		return 0
	}

	indexBody, indexFound := readMemoryIndex(files, cache)
	if !indexFound {
		return 0
	}

	rankPrompt := "Rank topic files by relevance to the query, one filename per line. Query: " + query
	rankResponse, rankErr := summarizer.ExtractRelevant(ctx, string(indexBody), rankPrompt)
	if rankErr != nil {
		return 0
	}

	topicByName := indexAutoMemoryFiles(files)
	added := 0

	for _, name := range parseRankedNames(rankResponse) {
		if ctx.Err() != nil {
			break
		}

		if bytesUsed+added >= cap {
			break
		}

		path, ok := topicByName[name]
		if !ok {
			continue
		}

		body, readErr := cache.Read(path)
		if readErr != nil {
			continue
		}

		snippet, extractErr := summarizer.ExtractRelevant(ctx, string(body), query)
		if extractErr != nil || snippet == "" {
			continue
		}

		buffer.WriteString(snippet)

		added += len(snippet)
	}

	return added
}

func readMemoryIndex(files []externalsources.ExternalFile, cache *externalsources.FileCache) ([]byte, bool) {
	for _, f := range files {
		if f.Kind == externalsources.KindAutoMemory && filepath.Base(f.Path) == "MEMORY.md" {
			body, err := cache.Read(f.Path)
			if err != nil {
				return nil, false
			}

			return body, true
		}
	}

	return nil, false
}

func indexAutoMemoryFiles(files []externalsources.ExternalFile) map[string]string {
	index := make(map[string]string, len(files))

	for _, f := range files {
		if f.Kind == externalsources.KindAutoMemory {
			index[filepath.Base(f.Path)] = f.Path
		}
	}

	return index
}

func parseRankedNames(response string) []string {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	names := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}

	return names
}

// statusf is a tiny helper to keep phases free of nil-writer guards.
func statusf(buf *strings.Builder, format string, args ...any) {
	if buf == nil {
		return
	}

	fmt.Fprintf(buf, format+"\n", args...)
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/recall/phases.go internal/recall/phases_test.go
git commit -m "$(cat <<'EOF'
feat(recall): auto memory phase with index-then-extract via Haiku rank

AI-Used: [claude]
EOF
)"
```

---

### Task 12: Skill phase (rank frontmatter + extract bodies)

**Files:**
- Modify: `internal/recall/phases.go`
- Modify: `internal/recall/phases_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/recall/phases_test.go`:
```go
func TestExtractFromSkills_RanksFrontmatterAndExtractsBodies(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindSkill, Path: "/skills/prepare/SKILL.md"},
		{Kind: externalsources.KindSkill, Path: "/skills/learn/SKILL.md"},
	}

	contents := map[string][]byte{
		"/skills/prepare/SKILL.md": []byte(`---
name: prepare
description: Use before starting new work to load context
---

# Prepare body
`),
		"/skills/learn/SKILL.md": []byte(`---
name: learn
description: Use after completing work to capture learnings
---

# Learn body
`),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	summarizer := &phasePipelineSummarizer{
		rankResponse: "prepare",
		extractMap: map[string]string{
			contentBodyAfterFrontmatter("/skills/prepare/SKILL.md", cache): "prepare-snippet",
		},
	}

	var buffer strings.Builder

	added := recall.ExtractFromSkills(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(ContainSubstring("prepare-snippet"))
}

// contentBodyAfterFrontmatter is a tiny helper that strips frontmatter so the
// fake summarizer can match on body content.
func contentBodyAfterFrontmatter(path string, cache *externalsources.FileCache) string {
	body, _ := cache.Read(path)
	_, rest := externalsources.ParseFrontmatter(body)

	return string(rest)
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: recall.ExtractFromSkills".

- [ ] **Step 3: Add the implementation**

Append to `internal/recall/phases.go`:
```go
// ExtractFromSkills runs phase 4 of the recall pipeline against installed
// SKILL.md files. It first builds a name+description index from each skill's
// frontmatter (one Haiku rank call), then iterates ranked-winning skills in
// order, Haiku-extracting from each body into the buffer until cap is hit.
func ExtractFromSkills(
	ctx context.Context,
	files []externalsources.ExternalFile,
	query string,
	cache *externalsources.FileCache,
	summarizer SummarizerI,
	buffer *strings.Builder,
	bytesUsed, cap int,
) int {
	if summarizer == nil || cache == nil || bytesUsed >= cap {
		return 0
	}

	frontmatterByName := loadSkillFrontmatter(files, cache)
	if len(frontmatterByName) == 0 {
		return 0
	}

	index := buildSkillIndex(frontmatterByName)

	rankPrompt := "Rank skills by relevance to the query, one skill name per line. Query: " + query
	rankResponse, rankErr := summarizer.ExtractRelevant(ctx, index, rankPrompt)
	if rankErr != nil {
		return 0
	}

	pathByName := skillPathByName(files, cache)
	added := 0

	for _, name := range parseRankedNames(rankResponse) {
		if ctx.Err() != nil {
			break
		}

		if bytesUsed+added >= cap {
			break
		}

		path, ok := pathByName[name]
		if !ok {
			continue
		}

		body, readErr := cache.Read(path)
		if readErr != nil {
			continue
		}

		_, rest := externalsources.ParseFrontmatter(body)

		snippet, extractErr := summarizer.ExtractRelevant(ctx, string(rest), query)
		if extractErr != nil || snippet == "" {
			continue
		}

		buffer.WriteString(snippet)

		added += len(snippet)
	}

	return added
}

func loadSkillFrontmatter(
	files []externalsources.ExternalFile,
	cache *externalsources.FileCache,
) map[string]externalsources.Frontmatter {
	out := make(map[string]externalsources.Frontmatter)

	for _, f := range files {
		if f.Kind != externalsources.KindSkill {
			continue
		}

		body, err := cache.Read(f.Path)
		if err != nil {
			continue
		}

		fm, _ := externalsources.ParseFrontmatter(body)
		if fm.Name != "" {
			out[fm.Name] = fm
		}
	}

	return out
}

func buildSkillIndex(frontmatterByName map[string]externalsources.Frontmatter) string {
	var builder strings.Builder

	for name, fm := range frontmatterByName {
		fmt.Fprintf(&builder, "%s | %s\n", name, fm.Description)
	}

	return builder.String()
}

func skillPathByName(
	files []externalsources.ExternalFile,
	cache *externalsources.FileCache,
) map[string]string {
	out := make(map[string]string)

	for _, f := range files {
		if f.Kind != externalsources.KindSkill {
			continue
		}

		body, err := cache.Read(f.Path)
		if err != nil {
			continue
		}

		fm, _ := externalsources.ParseFrontmatter(body)
		if fm.Name != "" {
			out[fm.Name] = f.Path
		}
	}

	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/recall/phases.go internal/recall/phases_test.go
git commit -m "$(cat <<'EOF'
feat(recall): skill phase ranks frontmatter and extracts winning bodies

AI-Used: [claude]
EOF
)"
```

---

### Task 13: CLAUDE.md + rules phase (concat + single extract)

**Files:**
- Modify: `internal/recall/phases.go`
- Modify: `internal/recall/phases_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/recall/phases_test.go`:
```go
func TestExtractFromClaudeMd_ConcatenatesAndExtractsOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindClaudeMd, Path: "/proj/CLAUDE.md"},
		{Kind: externalsources.KindRules, Path: "/proj/.claude/rules/code.md"},
	}

	contents := map[string][]byte{
		"/proj/CLAUDE.md":              []byte("Project-level rules.\n"),
		"/proj/.claude/rules/code.md":  []byte("Code style rules.\n"),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	summarizer := &phasePipelineSummarizer{
		extractMap: map[string]string{
			"Project-level rules.\nCode style rules.\n": "combined-snippet",
		},
	}

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(ContainSubstring("combined-snippet"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: recall.ExtractFromClaudeMd".

- [ ] **Step 3: Add the implementation**

Append to `internal/recall/phases.go`:
```go
// ExtractFromClaudeMd runs phase 5: concatenate every discovered CLAUDE.md
// and rules file, then make a single Haiku extract call against the result.
// CLAUDE.md files are designed to be small (<200 lines each), so we don't
// need an index step here.
func ExtractFromClaudeMd(
	ctx context.Context,
	files []externalsources.ExternalFile,
	query string,
	cache *externalsources.FileCache,
	summarizer SummarizerI,
	buffer *strings.Builder,
	bytesUsed, cap int,
) int {
	if summarizer == nil || cache == nil || bytesUsed >= cap {
		return 0
	}

	combined := concatRulesAndClaudeMd(files, cache)
	if combined == "" {
		return 0
	}

	snippet, err := summarizer.ExtractRelevant(ctx, combined, query)
	if err != nil || snippet == "" {
		return 0
	}

	buffer.WriteString(snippet)

	return len(snippet)
}

func concatRulesAndClaudeMd(
	files []externalsources.ExternalFile,
	cache *externalsources.FileCache,
) string {
	var builder strings.Builder

	for _, f := range files {
		if f.Kind != externalsources.KindClaudeMd && f.Kind != externalsources.KindRules {
			continue
		}

		body, err := cache.Read(f.Path)
		if err != nil {
			continue
		}

		builder.Write(body)
	}

	return builder.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/recall/phases.go internal/recall/phases_test.go
git commit -m "$(cat <<'EOF'
feat(recall): CLAUDE.md + rules phase concatenates and extracts once

AI-Used: [claude]
EOF
)"
```

---

### Task 14: Wire new phases into `recallModeB`

**Files:**
- Modify: `internal/recall/orchestrate.go` (extend Orchestrator + add WithExternalSources option + update recallModeB)
- Modify: `internal/recall/orchestrate_test.go` (new test for full pipeline ordering)

- [ ] **Step 1: Write the failing test**

Append to `internal/recall/orchestrate_test.go`:
```go
func TestRecallModeB_FullPipelinePriorityOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/topic.md"},
		{Kind: externalsources.KindSkill, Path: "/s/skill/SKILL.md"},
		{Kind: externalsources.KindClaudeMd, Path: "/proj/CLAUDE.md"},
	}

	contents := map[string][]byte{
		"/m/MEMORY.md":          []byte("Index: topic.md"),
		"/m/topic.md":           []byte("auto memory body"),
		"/s/skill/SKILL.md":     []byte("---\nname: foo\ndescription: a skill\n---\nskill body"),
		"/proj/CLAUDE.md":       []byte("CLAUDE.md content"),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	finder := &fakeFinder{entries: []recall.FileEntry{
		{Path: "/sessions/now.jsonl", Mtime: time.Now()},
	}}

	reader := &fakeReader{contents: map[string]string{
		"/sessions/now.jsonl": "session content",
	}}

	summarizer := &orderTrackingSummarizer{}

	orch := recall.NewOrchestrator(finder, reader, summarizer, nil, "",
		recall.WithExternalSources(files, cache),
	)

	_, err := orch.Recall(context.Background(), "/anywhere", "what's relevant?")
	g.Expect(err).NotTo(HaveOccurred())

	// Verify the order of source kinds extracted: auto memory first (after engram),
	// then sessions, then skills, then CLAUDE.md.
	g.Expect(summarizer.callOrder).To(ContainElement("auto_memory"))
	g.Expect(summarizer.callOrder).To(ContainElement("session"))
	g.Expect(summarizer.callOrder).To(ContainElement("skill"))
	g.Expect(summarizer.callOrder).To(ContainElement("claude_md"))

	autoIdx := indexOf(summarizer.callOrder, "auto_memory")
	sessionIdx := indexOf(summarizer.callOrder, "session")
	skillIdx := indexOf(summarizer.callOrder, "skill")
	claudeIdx := indexOf(summarizer.callOrder, "claude_md")

	g.Expect(autoIdx).To(BeNumerically("<", sessionIdx))
	g.Expect(sessionIdx).To(BeNumerically("<", skillIdx))
	g.Expect(skillIdx).To(BeNumerically("<", claudeIdx))
}

// orderTrackingSummarizer records the order of extract calls by source kind,
// inferred from the prompt content.
type orderTrackingSummarizer struct {
	callOrder []string
}

func (s *orderTrackingSummarizer) ExtractRelevant(_ context.Context, content, query string) (string, error) {
	switch {
	case strings.Contains(query, "Rank topic files"):
		return "topic.md", nil
	case strings.Contains(query, "Rank skills"):
		return "foo", nil
	case strings.Contains(content, "auto memory body"):
		s.callOrder = append(s.callOrder, "auto_memory")
		return "auto-snippet", nil
	case strings.Contains(content, "session content"):
		s.callOrder = append(s.callOrder, "session")
		return "session-snippet", nil
	case strings.Contains(content, "skill body"):
		s.callOrder = append(s.callOrder, "skill")
		return "skill-snippet", nil
	case strings.Contains(content, "CLAUDE.md content"):
		s.callOrder = append(s.callOrder, "claude_md")
		return "claude-snippet", nil
	default:
		return "", nil
	}
}

func (s *orderTrackingSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	return content, nil
}

func indexOf(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}

	return -1
}
```

(Add `"engram/internal/externalsources"` to the test imports.)

- [ ] **Step 2: Run test to verify it fails**

```
targ test
```
Expected: FAIL with "undefined: recall.WithExternalSources".

- [ ] **Step 3: Add the option + integrate into recallModeB**

In `internal/recall/orchestrate.go`:

Add to the `Orchestrator` struct:
```go
type Orchestrator struct {
	// ... existing fields ...
	externalFiles []externalsources.ExternalFile
	fileCache     *externalsources.FileCache
}
```

Add a new option:
```go
// WithExternalSources configures the orchestrator to cross-search external
// sources (CLAUDE.md hierarchy, rules, auto memory, skills) discovered by
// the externalsources package. The cache is shared across phases so the
// same file is read at most once per invocation.
func WithExternalSources(files []externalsources.ExternalFile, cache *externalsources.FileCache) OrchestratorOption {
	return func(o *Orchestrator) {
		o.externalFiles = files
		o.fileCache = cache
	}
}
```

Update `recallModeB` to insert the new phases (auto memory between phase 1 and old phase 2; skills + claude.md between sessions and synthesis):
```go
func (o *Orchestrator) recallModeB(
	ctx context.Context,
	sessions []FileEntry,
	query string,
) (*Result, error) {
	if o.summarizer == nil {
		return &Result{}, nil
	}

	var buffer strings.Builder

	// Phase 1: Engram memory search.
	memoriesLen := o.searchMemories(ctx, query, &buffer)
	bytesUsed := memoriesLen

	// Phase 2: Auto memory extraction.
	autoLen := ExtractFromAutoMemory(
		ctx, o.externalFiles, query, o.fileCache, o.summarizer,
		&buffer, bytesUsed, DefaultExtractCap,
	)

	bytesUsed += autoLen

	o.writeStatusf("auto memory contributed %d bytes", autoLen)

	// Phase 3: Per-session extraction.
	preSessionLen := bytesUsed
	o.extractFromSessions(ctx, sessions, query, &buffer, bytesUsed)

	bytesUsed += buffer.Len() - preSessionLen

	// Phase 4: Skill extraction.
	skillLen := ExtractFromSkills(
		ctx, o.externalFiles, query, o.fileCache, o.summarizer,
		&buffer, bytesUsed, DefaultExtractCap,
	)

	bytesUsed += skillLen

	o.writeStatusf("skills contributed %d bytes", skillLen)

	// Phase 5: CLAUDE.md + rules extraction.
	claudeLen := ExtractFromClaudeMd(
		ctx, o.externalFiles, query, o.fileCache, o.summarizer,
		&buffer, bytesUsed, DefaultExtractCap,
	)

	o.writeStatusf("claude.md contributed %d bytes", claudeLen)

	if buffer.Len() == 0 {
		return &Result{}, nil
	}

	// Phase 6: Structured summary.
	o.writeStatusf("summarizing %d bytes of findings", buffer.Len())

	summary, err := o.summarizer.SummarizeFindings(ctx, buffer.String(), query)
	if err != nil {
		return nil, fmt.Errorf("summarizing recall: %w", err)
	}

	return &Result{Summary: summary}, nil
}
```

(Update imports to include `engram/internal/externalsources`.)

- [ ] **Step 4: Run test to verify it passes**

```
targ test
```
Expected: PASS.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/recall/orchestrate.go internal/recall/orchestrate_test.go
git commit -m "$(cat <<'EOF'
feat(recall): wire auto memory, skills, claude.md phases into recallModeB

AI-Used: [claude]
EOF
)"
```

---

## Phase 4 — CLI wiring

### Task 15: Wire `Discover` + `FileCache` into the CLI

**Files:**
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Inspect current CLI orchestrator construction**

Re-read `internal/cli/cli.go:216-249` (the `runRecallSessions` function) to see how `recall.NewOrchestrator` is currently wired. The new option goes alongside `WithStatusWriter`.

- [ ] **Step 2: Add CLI adapters for the new edge dependencies**

Append to `internal/cli/cli.go` (or a new `internal/cli/externalsources_adapters.go` if you prefer to keep cli.go lean — match the existing convention):

```go
// osStatExists reports whether a file exists at path. Adapter for
// externalsources.StatFunc.
func osStatExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, fmt.Errorf("stat %s: %w", path, err)
}

// osWalkMd returns absolute paths to all *.md files under root, recursively.
// Errors and missing directories are silently treated as empty.
func osWalkMd(root string) []string {
	var found []string

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // skip unreadable subtrees, continue walk
		}

		if filepath.Ext(d.Name()) == ".md" {
			found = append(found, path)
		}

		return nil
	})

	return found
}

// osWalkSkills returns absolute paths to all SKILL.md files under root.
func osWalkSkills(root string) []string {
	var found []string

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // skip unreadable subtrees, continue walk
		}

		if d.Name() == "SKILL.md" {
			found = append(found, path)
		}

		return nil
	})

	return found
}

// osMatchAny reports whether any file under cwd matches any of the globs.
func osMatchAny(cwd string) externalsources.GlobMatcher {
	return func(globs []string) bool {
		for _, g := range globs {
			matches, err := filepath.Glob(filepath.Join(cwd, g))
			if err == nil && len(matches) > 0 {
				return true
			}
		}

		return false
	}
}

// osDirListMd returns absolute paths to *.md files in dir (non-recursive).
// Returns nil for missing dirs or read errors so DiscoverAutoMemory treats it
// as "no files".
func osDirListMd(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil //nolint:nilerr // missing dir is normal for opt-in auto memory
	}

	out := make([]string, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}

	return out, nil
}

// readAutoMemoryDirectorySetting reads the user/local settings.json for the
// `autoMemoryDirectory` field. Returns ("", false) on any error or when the
// field is absent.
func readAutoMemoryDirectorySetting(home string) externalsources.AutoMemorySettingsFunc {
	return func() (string, bool) {
		paths := []string{
			filepath.Join(".", ".claude", "settings.local.json"),
			filepath.Join(home, ".claude", "settings.json"),
		}

		for _, p := range paths {
			body, err := os.ReadFile(p) //nolint:gosec // user-controlled path inside .claude
			if err != nil {
				continue
			}

			var settings struct {
				AutoMemoryDirectory string `json:"autoMemoryDirectory"`
			}

			if json.Unmarshal(body, &settings) == nil && settings.AutoMemoryDirectory != "" {
				return settings.AutoMemoryDirectory, true
			}
		}

		return "", false
	}
}

// detectMainRepoRoot returns the main repo root if cwd is inside a git
// worktree distinct from the main checkout. Returns "" on any error or
// non-worktree case (callers treat "" as "no fallback needed").
func detectMainRepoRoot(cwd string) string {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-common-dir")

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	commonDir := strings.TrimSpace(string(out))
	if commonDir == "" {
		return ""
	}

	// common-dir is "<main>/.git"; main repo root is its parent.
	return filepath.Dir(commonDir)
}
```

(Add `"encoding/json"`, `"io/fs"`, `"os/exec"`, and `"engram/internal/externalsources"` to the import block as needed.)

- [ ] **Step 3: Construct discovery + cache and pass to orchestrator**

In `runRecallSessions` (around `internal/cli/cli.go:236-243`), insert before the `orch := recall.NewOrchestrator(...)` line:

```go
externalFiles, externalCache := discoverExternalSources(home, *projectSlug)

orch := recall.NewOrchestrator(finder, reader, summarizer, memLister, dataDir,
	recall.WithStatusWriter(os.Stderr),
	recall.WithExternalSources(externalFiles, externalCache),
)
```

Add a helper above:
```go
func discoverExternalSources(home, projectSlug string) ([]externalsources.ExternalFile, *externalsources.FileCache) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	cache := externalsources.NewFileCache(func(path string) ([]byte, error) {
		return os.ReadFile(path) //nolint:gosec // platform-internal path
	})

	deps := externalsources.DiscoverDeps{
		CWD:          cwd,
		Home:         home,
		MainRepoRoot: detectMainRepoRoot(cwd),
		StatFn:       osStatExists,
		Reader:       cache.Read,
		MdWalker:     osWalkMd,
		MatchAny:     osMatchAny(cwd),
		Settings:     readAutoMemoryDirectorySetting(home),
		DirLister:    osDirListMd,
		SkillFinder:  osWalkSkills,
	}
	_ = projectSlug // reserved for future per-slug overrides

	return externalsources.Discover(deps), cache
}
```

- [ ] **Step 4: Run tests to verify nothing regresses**

```
targ test
```
Expected: PASS (no new tests needed for this task — coverage by integration test in Phase 5).

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS. If new lint warnings appear (large file, too many deps in one function), refactor minimally.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go
git commit -m "$(cat <<'EOF'
feat(cli): wire externalsources discovery and cache into recall orchestrator

AI-Used: [claude]
EOF
)"
```

---

### Task 16: Status output events for new phases

**Files:**
- Modify: `internal/recall/orchestrate.go`

The `writeStatusf` calls were added in Task 14. This task verifies they are useful in practice and adjusts wording if needed.

- [ ] **Step 1: Inspect current status output**

Run `engram recall --query "test"` against engram's own repo (Task 19 will smoke-test more thoroughly). Confirm status messages are emitted for each new phase. If any look confusing, edit the format strings.

- [ ] **Step 2: Add a phase-start event for skills (the slowest phase)**

In `internal/recall/orchestrate.go` `recallModeB`, before the skill phase runs:
```go
o.writeStatusf("ranking %d skills", countByKind(o.externalFiles, externalsources.KindSkill))
```

Add helper:
```go
func countByKind(files []externalsources.ExternalFile, kind externalsources.Kind) int {
	count := 0

	for _, f := range files {
		if f.Kind == kind {
			count++
		}
	}

	return count
}
```

- [ ] **Step 3: Run tests**

```
targ test
```
Expected: PASS.

- [ ] **Step 4: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/recall/orchestrate.go
git commit -m "$(cat <<'EOF'
feat(recall): status output for new pipeline phases

AI-Used: [claude]
EOF
)"
```

---

## Phase 5 — Integration test + cost regression guard

### Task 17: End-to-end integration test against fixture project

**Files:**
- Create: `internal/externalsources/testdata/fixture-project/CLAUDE.md`
- Create: `internal/externalsources/testdata/fixture-project/.claude/rules/code.md`
- Create: `internal/externalsources/testdata/fixture-project/.claude/skills/test-skill/SKILL.md`
- Create: `internal/externalsources/testdata/fixture-project/auto-memory/MEMORY.md`
- Create: `internal/externalsources/testdata/fixture-project/auto-memory/topic.md`
- Create: `internal/externalsources/integration_test.go`

- [ ] **Step 1: Create fixture files**

```bash
mkdir -p internal/externalsources/testdata/fixture-project/.claude/rules
mkdir -p internal/externalsources/testdata/fixture-project/.claude/skills/test-skill
mkdir -p internal/externalsources/testdata/fixture-project/auto-memory
```

`internal/externalsources/testdata/fixture-project/CLAUDE.md`:
```markdown
# Fixture project

Project-level rules go here.
```

`internal/externalsources/testdata/fixture-project/.claude/rules/code.md`:
```markdown
# Code style

- 2-space indent
```

`internal/externalsources/testdata/fixture-project/.claude/skills/test-skill/SKILL.md`:
```markdown
---
name: test-skill
description: A skill for the integration test
---

# Test skill body
```

`internal/externalsources/testdata/fixture-project/auto-memory/MEMORY.md`:
```markdown
# Index

- topic.md — example topic
```

`internal/externalsources/testdata/fixture-project/auto-memory/topic.md`:
```markdown
# Topic

Some auto-memory body content.
```

- [ ] **Step 2: Write the failing test**

`internal/externalsources/integration_test.go`:
```go
package externalsources_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscover_AgainstFixtureProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixtureRoot, err := filepath.Abs("testdata/fixture-project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cache := externalsources.NewFileCache(func(path string) ([]byte, error) {
		return os.ReadFile(path) //nolint:gosec // test-controlled path
	})

	deps := externalsources.DiscoverDeps{
		CWD:          fixtureRoot,
		Home:         "/nonexistent",
		MainRepoRoot: "",
		StatFn: func(p string) (bool, error) {
			_, statErr := os.Stat(p)
			return statErr == nil, nil
		},
		Reader: cache.Read,
		MdWalker: func(root string) []string {
			var out []string

			_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() && filepath.Ext(d.Name()) == ".md" {
					out = append(out, path)
				}

				return nil
			})

			return out
		},
		MatchAny: func(_ []string) bool { return false },
		Settings: func() (string, bool) {
			return filepath.Join(fixtureRoot, "auto-memory"), true
		},
		DirLister: func(dir string) ([]string, error) {
			entries, _ := os.ReadDir(dir)
			out := make([]string, 0, len(entries))

			for _, e := range entries {
				out = append(out, filepath.Join(dir, e.Name()))
			}

			return out, nil
		},
		SkillFinder: func(root string) []string {
			var out []string

			_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() && d.Name() == "SKILL.md" {
					out = append(out, path)
				}

				return nil
			})

			return out
		},
	}

	files := externalsources.Discover(deps)

	kinds := make(map[externalsources.Kind]int)
	for _, f := range files {
		kinds[f.Kind]++
	}

	g.Expect(kinds[externalsources.KindClaudeMd]).To(BeNumerically(">=", 1))
	g.Expect(kinds[externalsources.KindRules]).To(BeNumerically(">=", 1))
	g.Expect(kinds[externalsources.KindAutoMemory]).To(BeNumerically(">=", 2)) // MEMORY.md + topic.md
	g.Expect(kinds[externalsources.KindSkill]).To(BeNumerically(">=", 1))
}
```

- [ ] **Step 3: Run test to verify it fails**

```
targ test
```
Expected: FAIL with assertion mismatches if fixture is misnamed; otherwise PASS once everything is wired.

- [ ] **Step 4: Iterate until green**

If the test fails because of off-by-one fixture issues (e.g., user-scope CLAUDE.md being absent makes the count off), adjust the assertions or the fixture so the test is deterministic.

- [ ] **Step 5: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/externalsources/testdata/ internal/externalsources/integration_test.go
git commit -m "$(cat <<'EOF'
test(externalsources): integration test against testdata fixture project

AI-Used: [claude]
EOF
)"
```

---

### Task 18: Cost regression guard

**Files:**
- Create: `internal/recall/cost_test.go`

- [ ] **Step 1: Write the failing test**

`internal/recall/cost_test.go`:
```go
package recall_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
	"engram/internal/recall"
)

func TestRecall_HaikuCallCountStaysBounded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const numSkills = 50

	const numTopics = 20

	files := make([]externalsources.ExternalFile, 0, numSkills+numTopics+1)
	contents := make(map[string][]byte, numSkills+numTopics+1)

	files = append(files,
		externalsources.ExternalFile{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
	)
	contents["/m/MEMORY.md"] = []byte("Index of topics")

	for i := range numTopics {
		path := fmt.Sprintf("/m/topic%d.md", i)
		files = append(files, externalsources.ExternalFile{Kind: externalsources.KindAutoMemory, Path: path})
		contents[path] = []byte(fmt.Sprintf("body of topic %d", i))
	}

	for i := range numSkills {
		path := fmt.Sprintf("/s/skill%d/SKILL.md", i)
		files = append(files, externalsources.ExternalFile{Kind: externalsources.KindSkill, Path: path})
		contents[path] = []byte(fmt.Sprintf("---\nname: skill%d\ndescription: a skill\n---\nbody %d", i, i))
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	finder := &fakeFinder{entries: []recall.FileEntry{
		{Path: "/sessions/now.jsonl", Mtime: time.Now()},
	}}
	reader := &fakeReader{contents: map[string]string{
		"/sessions/now.jsonl": "session content",
	}}

	counter := &countingSummarizer{returnSnippet: strings.Repeat("x", 1000)}

	orch := recall.NewOrchestrator(finder, reader, counter, nil, "",
		recall.WithExternalSources(files, cache),
	)

	_, err := orch.Recall(context.Background(), "/anywhere", "query")
	g.Expect(err).NotTo(HaveOccurred())

	// Bound: 1 (engram rank) + 1 (auto memory rank) + N (auto extracts capped by buffer) +
	//        N (session extracts capped) + 1 (skill rank) + N (skill extracts capped) +
	//        1 (claude.md combined) + 1 (synthesis).
	// With a 10KB buffer and 1KB per snippet, no phase should exceed ~10 extracts.
	const maxAcceptableCalls = 50

	g.Expect(int(counter.calls.Load())).To(BeNumerically("<=", maxAcceptableCalls))
}

type countingSummarizer struct {
	calls         atomic.Int64
	returnSnippet string
}

func (c *countingSummarizer) ExtractRelevant(_ context.Context, _, query string) (string, error) {
	c.calls.Add(1)

	if strings.Contains(query, "Rank topic files") {
		return "topic0.md\ntopic1.md\ntopic2.md", nil
	}

	if strings.Contains(query, "Rank skills") {
		return "skill0\nskill1\nskill2", nil
	}

	return c.returnSnippet, nil
}

func (c *countingSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	c.calls.Add(1)
	return content, nil
}
```

- [ ] **Step 2: Run test**

```
targ test
```
Expected: PASS (assertion holds because buffer fills early).

- [ ] **Step 3: Run check-full**

```
targ check-full
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/recall/cost_test.go
git commit -m "$(cat <<'EOF'
test(recall): cost regression guard caps Haiku calls under buffer pressure

AI-Used: [claude]
EOF
)"
```

---

## Phase 6 — Smoke test

### Task 19: Manual smoke test on engram repo

**Files:** none (operational only).

- [ ] **Step 1: Build engram**

```
targ build
```
Expected: success, binary updated.

- [ ] **Step 2: Run a real recall query in the engram repo**

```
ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" engram recall --query "implementing concurrent Go code with context"
```
Expected: output mentions content from at least one of: engram memories, sessions, auto memory, skills, CLAUDE.md / `.claude/rules/go.md`. Status messages on stderr show each phase running.

- [ ] **Step 3: Diagnose any unexpected output**

If a phase doesn't surface content you'd expect:
- Re-read the phase's status line — was the rank step skipped because the source had no files?
- Re-run with a more targeted query (e.g., `--query "nilaway gomega compatibility"` should pull from `.claude/rules/go.md`).

If any phase silently does nothing despite having files (e.g., skill phase yields zero bytes despite installed skills), file an issue with reproduction steps. Don't paper over the issue.

- [ ] **Step 4: Commit any documentation update if behavior diverges from the spec**

If the smoke test reveals a spec inaccuracy, update the spec, then re-commit:
```bash
git add docs/superpowers/specs/2026-04-16-engram-claude-memory-integration-design.md
git commit -m "$(cat <<'EOF'
docs(specs): note <specific behavior> per smoke test

AI-Used: [claude]
EOF
)"
```

Otherwise no commit needed for this task.

---

## Self-Review Checklist (run after writing the plan)

- ✅ **Spec coverage**: every spec section has a task that implements it.
  - Architecture (subsystem `internal/externalsources/`, write path untouched) → Tasks 1, 14.
  - File discovery (CLAUDE.md, rules, auto memory, skills, error resilience) → Tasks 4, 5, 6, 7, 8, 9, 10.
  - Recall buffer-fill algorithm (per-phase rank + extract, buffer cap) → Tasks 11, 12, 13, 14.
  - Edge cases (no external sources discoverable, malformed rank, Haiku error, file mtime mid-invocation) → covered by per-task tests; missing files handled by discovery returning empty.
  - Testing (per-package unit tests, fixture integration, cost regression guard) → Tasks 17, 18, plus per-task tests in Tasks 1-13.
  - Rollout (single ship, no flag) → no plan task needed; just don't add toggles.
  - Risks (rank irrelevance, large skill corpus, autoMemoryDirectory non-standard, slug divergence) → mitigated by Tasks 7, 11, 15.
- ✅ **Placeholder scan**: no TBD/TODO/"add appropriate", every code block has full content. The `_ = projectSlug` line in Task 15 is intentional with explanatory comment.
- ✅ **Type consistency**: `ReaderFunc`, `StatFunc`, `MdWalker`, `GlobMatcher`, `AutoMemorySettingsFunc`, `DirListerFunc`, `SkillFinder`, `Frontmatter`, `Kind`, `ExternalFile`, `FileCache`, `DiscoverDeps`, `Discover`, `WithExternalSources`, `ExtractFromAutoMemory`, `ExtractFromSkills`, `ExtractFromClaudeMd`, `ProjectSlug`, `ManagedPolicyPath` — all referenced consistently across tasks.

---

## Open implementation questions for the executing engineer

These are deliberately left for the executor to resolve during implementation, since the right answer depends on what's actually in the codebase / runtime when the task is reached:

1. **Project slug verification** (Task 7): if Claude Code's slug algorithm differs from `strings.ReplaceAll(absPath, "/", "-")`, update the function and add a regression test for the corrected case before continuing.
2. **Status writer plumbing** (Task 16): the new phase events should land cleanly in the existing `WithStatusWriter` output. If they look spammy, prune.
3. **Coverage thresholds**: if `targ check-full` enforces a per-package coverage floor for `internal/externalsources`, mirror whatever the existing thin-package convention is (e.g., `tomlwriter`).
