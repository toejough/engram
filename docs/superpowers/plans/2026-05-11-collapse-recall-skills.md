# Collapse Recall Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse three recall-related skills (`prepare`, `recall`, `recalling-from-vault`) into one repo-shipped `recall` skill, split the underlying CLI so vault recall and session-transcript reading are separate commands, and add a frontier-expansion mode that lets the skill drive iterative cascade traversal.

**Architecture:**
- `engram recall` becomes vault-only with three modes: no-arg (structural anchors), `--recent` (most-recent by date prefix), and `--follow X,Y --already-read A,B` (frontier expansion via outgoing links + backlinks, deduped, minus seen).
- `engram transcript --from DATE --to DATE` replaces the transcript-reading capability that previously lived inside `engram recall`. Output reuses the existing `internal/context/StripWithConfig` truncation as-is.
- `internal/recall/` splits into `internal/transcript/` (finders/readers/strip). `internal/vaultgraph/` gains `Recent` and `Follow` functions. `engram starting-points` command is deleted; its behavior is the no-arg `engram recall`.
- One `skills/recall/SKILL.md` replaces `skills/prepare/SKILL.md`, the existing `skills/recall/SKILL.md`, and the user-global `~/.claude/skills/recalling-from-vault/SKILL.md`. The skill drives the cascade as a loop calling `engram recall --follow` repeatedly until ≥100 memories or frontier empties.

**Tech Stack:** Go (CLI binary, internal packages), `targ` build system, imptest + rapid + gomega for tests, Claude Code skills (markdown).

---

## Pre-flight

Before starting, the engineer should read:

- `internal/recall/recall.go` — current session-finder + transcript-reader types
- `internal/recall/opencode.go` — opencode-specific transcript reader (also uses `sessionctx`)
- `internal/cli/cli.go:263` — current `runRecall` (will shrink dramatically)
- `internal/cli/cycle.go` — depends on `engram/internal/recall`; imports update only
- `internal/cli/startingpoints.go` — current vault-graph CLI surface; behavior moves into `runRecall`
- `internal/vaultgraph/vaultgraph.go` — `StartingPoints` already exists; we add `Recent` and `Follow`
- `internal/context/context.go` — `Strip` / `StripWithConfig`; unchanged
- `skills/prepare/SKILL.md` — pre-work framing to preserve
- `skills/recall/SKILL.md` — current minimal recall skill; replaced wholesale
- `~/.claude/skills/recalling-from-vault/SKILL.md` — the structural template we keep

**Build commands (use these — never `go test` / `go build` directly):**
- Tests: `targ test`
- Lint + coverage: `targ check-full`
- Build: `targ build`

**Test conventions:** imptest with impgen-generated mocks; rapid for property tests; gomega assertions. After `g.Expect(err).NotTo(HaveOccurred())`, always add `if err != nil { return }` before accessing values (nilaway). Every test and subtest gets `t.Parallel()`.

---

## File Structure

**Create:**
- `internal/transcript/transcript.go` — moved from `internal/recall/recall.go` (SessionFinder, TranscriptReader, composites)
- `internal/transcript/opencode.go` — moved from `internal/recall/opencode.go`
- `internal/transcript/transcript_test.go`, `internal/transcript/opencode_test.go` — moved
- `internal/transcript/export_test.go` — moved if it exists in `internal/recall/`
- `internal/vaultgraph/recent.go` — new `Recent` function (sort by filename date prefix, take N)
- `internal/vaultgraph/recent_test.go` — tests for `Recent`
- `internal/vaultgraph/follow.go` — new `Follow` function (outgoing links + backlinks, deduped, minus seen)
- `internal/vaultgraph/follow_test.go` — tests for `Follow`
- `internal/cli/transcript.go` — new `runTranscript` command + `TranscriptArgs`
- `internal/cli/transcript_test.go` — CLI-level tests
- `skills/recall/SKILL.md` — new merged skill (overwrites existing)
- `docs/superpowers/plans/2026-05-11-collapse-recall-skills.md` — this plan

**Modify:**
- `internal/recall/orchestrate.go`, `summarize.go`, `automemory_phase.go`, `claudemd_phase.go`, `skill_phase.go`, `cost_test.go` — stay in `internal/recall/` (still used by `cycle.go` for the eval pipeline); only `recall.go` + `opencode.go` move
- `internal/cli/cli.go` — `runRecall` rewritten to be vault-only with new flag set
- `internal/cli/cycle.go` — imports `engram/internal/transcript` for the moved types
- `internal/cli/targets.go` — `RecallArgs` updated, `StartingPointsArgs` deleted, `TranscriptArgs` added; target registrations updated
- `internal/cli/startingpoints.go` — deleted (behavior moved into `runRecall`)
- `internal/vaultgraph/vaultgraph.go` — unchanged (`StartingPoints` stays as a primitive used by no-arg recall)

**Delete:**
- `skills/prepare/SKILL.md`
- `internal/cli/startingpoints.go`, `internal/cli/startingpoints_test.go`
- `internal/recall/recall.go`, `internal/recall/recall_test.go`, `internal/recall/opencode.go`, `internal/recall/opencode_test.go`, `internal/recall/export_test.go` (all moved to `internal/transcript/`)

**Retire (out of scope, manual step by user):**
- `~/.claude/skills/recalling-from-vault/SKILL.md` — user replaces with symlink to repo's `skills/recall/SKILL.md` during dev

---

## Task 1: Move `internal/recall/recall.go` and `opencode.go` to `internal/transcript/`

The transcript-reading types (`SessionFinder`, `TranscriptReader`, `OpencodeSessionFinder`, `OpencodeTranscriptReader`, composites) are independently useful and need to be reachable from a new CLI command without going through the recall orchestrator. Move them to a new `internal/transcript/` package.

**Files:**
- Move: `internal/recall/recall.go` → `internal/transcript/transcript.go`
- Move: `internal/recall/recall_test.go` → `internal/transcript/transcript_test.go`
- Move: `internal/recall/opencode.go` → `internal/transcript/opencode.go`
- Move: `internal/recall/opencode_test.go` → `internal/transcript/opencode_test.go`
- Move: `internal/recall/export_test.go` → `internal/transcript/export_test.go` (if it exists)
- Modify: `internal/recall/orchestrate.go` — change types to reference `transcript` package
- Modify: `internal/cli/cycle.go` — change imports

- [ ] **Step 1: Move the files (no content edits yet)**

```bash
mkdir internal/transcript
git mv internal/recall/recall.go internal/transcript/transcript.go
git mv internal/recall/recall_test.go internal/transcript/transcript_test.go
git mv internal/recall/opencode.go internal/transcript/opencode.go
git mv internal/recall/opencode_test.go internal/transcript/opencode_test.go
test -f internal/recall/export_test.go && git mv internal/recall/export_test.go internal/transcript/export_test.go
```

- [ ] **Step 2: Update package declarations**

In every moved file, change the package line:

```go
// Before:
package recall

// After:
package transcript
```

For test files using `_test` suffix:

```go
// Before:
package recall_test

// After:
package transcript_test
```

- [ ] **Step 3: Update internal references in moved code**

Inside `internal/transcript/transcript.go` and `opencode.go`, all symbol references already use bare names (e.g. `NewSessionFinder`, not `recall.NewSessionFinder`). No edits needed inside the moved files. In the test files, change any `recall.` prefixes to `transcript.`:

```bash
grep -n "recall\." internal/transcript/*_test.go
```

For each line found, replace `recall.` with `transcript.`.

- [ ] **Step 4: Update `internal/recall/orchestrate.go` to reference the new package**

```bash
grep -n "SessionFinder\|TranscriptReader\|CompositeSessionFinder\|CompositeTranscriptReader\|OpencodeSessionFinder\|OpencodeTranscriptReader\|FileEntry\|DefaultOpencodeDBPath" internal/recall/orchestrate.go
```

For each match, the type now lives in `transcript`. Add the import:

```go
import (
    // ... existing imports
    "engram/internal/transcript"
)
```

And prefix each referenced symbol with `transcript.` — for example `recall.SessionFinder` (already an internal bare reference like `*SessionFinder`) becomes `transcript.SessionFinder`. If the orchestrator defines interfaces named `Finder` / `Reader` that the moved types satisfied, those interfaces stay in `internal/recall/`; the orchestrator now consumes them via `transcript.NewSessionFinder` etc.

Do the same scan on the other remaining `internal/recall/` files (`automemory_phase.go`, `claudemd_phase.go`, `skill_phase.go`, `summarize.go`, `cost_test.go`):

```bash
grep -ln "SessionFinder\|TranscriptReader\|FileEntry" internal/recall/*.go
```

Update each to import `engram/internal/transcript` and prefix appropriately.

- [ ] **Step 5: Update `internal/cli/cycle.go` imports**

```bash
grep -n "recall\." internal/cli/cycle.go
```

For every `recall.NewSessionFinder`, `recall.NewTranscriptReader`, `recall.NewCompositeSessionFinder`, `recall.NewCompositeTranscriptReader`, `recall.NewOpencodeSessionFinder`, `recall.NewOpencodeTranscriptReader`, `recall.DefaultOpencodeDBPath`, `recall.Finder`, `recall.Reader` — replace `recall.` with `transcript.`. Keep `recall.NewOrchestrator`, `recall.SummarizerI`, and any orchestrator-related symbols as `recall.` because those stay in `internal/recall/`.

Add the import:

```go
import (
    // ... existing imports
    "engram/internal/transcript"
)
```

If `engram/internal/recall` is no longer referenced after edits, remove that import.

- [ ] **Step 6: Run tests to verify the move compiles and tests pass**

Run: `targ test`
Expected: all tests pass (no behavior change; this is a pure rename).

If errors mention missing symbols, scan with:

```bash
grep -rn "recall\.SessionFinder\|recall\.TranscriptReader\|recall\.FileEntry\|recall\.CompositeSessionFinder\|recall\.CompositeTranscriptReader\|recall\.OpencodeSessionFinder\|recall\.OpencodeTranscriptReader\|recall\.DefaultOpencodeDBPath\|recall\.Finder\|recall\.Reader" --include="*.go" .
```

Fix each by changing to `transcript.`.

- [ ] **Step 7: Run lint+coverage**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor(recall): split transcript reader into internal/transcript package

Moves SessionFinder, TranscriptReader, opencode composites, and FileEntry from
internal/recall to internal/transcript so they can be reused by a new
'engram transcript' CLI command without going through the recall orchestrator.
Pure rename — no behavior change.

AI-Used: [claude]"
```

---

## Task 2: Add `Recent` to `internal/vaultgraph/`

`Recent` returns the N most-recently-authored note basenames in the vault, sorted by the `YYYY-MM-DD` filename date prefix (descending). Used by `engram recall --recent`.

**Files:**
- Create: `internal/vaultgraph/recent.go`
- Create: `internal/vaultgraph/recent_test.go`

- [ ] **Step 1: Write the failing test**

```go
package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"engram/internal/vaultgraph"
)

func TestRecent_OrdersByDateDescending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "1.2026-01-15.alpha.md"},
		{Basename: "2.2026-03-20.bravo.md"},
		{Basename: "3.2026-02-10.charlie.md"},
	}

	got := vaultgraph.Recent(notes, 10)

	g.Expect(got).To(Equal([]string{
		"2.2026-03-20.bravo.md",
		"3.2026-02-10.charlie.md",
		"1.2026-01-15.alpha.md",
	}))
}

func TestRecent_RespectsLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "1.2026-01-15.a.md"},
		{Basename: "2.2026-02-15.b.md"},
		{Basename: "3.2026-03-15.c.md"},
	}

	got := vaultgraph.Recent(notes, 2)

	g.Expect(got).To(Equal([]string{
		"3.2026-03-15.c.md",
		"2.2026-02-15.b.md",
	}))
}

func TestRecent_DropsBasenamesWithoutDatePrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "MEMORY.md"},
		{Basename: "1.2026-03-15.b.md"},
		{Basename: "no-date.md"},
	}

	got := vaultgraph.Recent(notes, 10)

	g.Expect(got).To(Equal([]string{"1.2026-03-15.b.md"}))
}

func TestRecent_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(vaultgraph.Recent(nil, 10)).To(BeEmpty())
	g.Expect(vaultgraph.Recent([]vaultgraph.Note{}, 10)).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL with `undefined: vaultgraph.Recent`.

- [ ] **Step 3: Implement `Recent`**

Create `internal/vaultgraph/recent.go`:

```go
package vaultgraph

import (
	"regexp"
	"sort"
)

// datePrefixRE matches a leading "<luhmann>.YYYY-MM-DD." prefix and captures
// the date. The basename format is "<luhmann-id>.YYYY-MM-DD.<slug>.md".
var datePrefixRE = regexp.MustCompile(`^[^.]+\.(\d{4}-\d{2}-\d{2})\.`)

// Recent returns up to limit basenames sorted by their YYYY-MM-DD filename
// date prefix in descending order (newest first). Basenames lacking a valid
// date prefix are skipped — MEMORY.md and index files do not surface here.
// Ties on date break by basename ascending for determinism.
func Recent(notes []Note, limit int) []string {
	type dated struct {
		basename string
		date     string
	}

	candidates := make([]dated, 0, len(notes))

	for _, n := range notes {
		match := datePrefixRE.FindStringSubmatch(n.Basename)
		if match == nil {
			continue
		}

		candidates = append(candidates, dated{basename: n.Basename, date: match[1]})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].date == candidates[j].date {
			return candidates[i].basename < candidates[j].basename
		}

		return candidates[i].date > candidates[j].date
	})

	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.basename)
	}

	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Run lint+coverage**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/vaultgraph/recent.go internal/vaultgraph/recent_test.go
git commit -m "feat(vaultgraph): add Recent for most-recent-by-date filename selection

AI-Used: [claude]"
```

---

## Task 3: Add `Follow` to `internal/vaultgraph/`

`Follow` expands the cascade frontier: given a set of files to follow and a set of already-read files, return the deduplicated union of outgoing wikilinks AND backlinks from the follow set, minus the already-read set. Used by `engram recall --follow X,Y --already-read A,B`.

**Files:**
- Create: `internal/vaultgraph/follow.go`
- Create: `internal/vaultgraph/follow_test.go`

- [ ] **Step 1: Write the failing test**

```go
package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"engram/internal/vaultgraph"
)

func TestFollow_ReturnsOutgoingLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a.md", OutgoingLinks: []string{"b.md", "c.md"}},
		{Basename: "b.md"},
		{Basename: "c.md"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a.md"}, nil)

	g.Expect(got).To(ConsistOf("b.md", "c.md"))
}

func TestFollow_ReturnsBacklinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a.md"},
		{Basename: "b.md", OutgoingLinks: []string{"a.md"}},
		{Basename: "c.md", OutgoingLinks: []string{"a.md"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a.md"}, nil)

	g.Expect(got).To(ConsistOf("b.md", "c.md"))
}

func TestFollow_DedupesAcrossOutgoingAndBacklinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// b links to a AND a links to b — only one b should appear.
	notes := []vaultgraph.Note{
		{Basename: "a.md", OutgoingLinks: []string{"b.md"}},
		{Basename: "b.md", OutgoingLinks: []string{"a.md"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a.md"}, nil)

	g.Expect(got).To(Equal([]string{"b.md"}))
}

func TestFollow_SubtractsAlreadyRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a.md", OutgoingLinks: []string{"b.md", "c.md"}},
		{Basename: "b.md"},
		{Basename: "c.md"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a.md"}, []string{"b.md"})

	g.Expect(got).To(Equal([]string{"c.md"}))
}

func TestFollow_SubtractsFollowSetItself(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Following 'a' should never re-emit 'a' even if it appears in a self-link
	// or a partner's outgoing list.
	notes := []vaultgraph.Note{
		{Basename: "a.md", OutgoingLinks: []string{"b.md"}},
		{Basename: "b.md", OutgoingLinks: []string{"a.md"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a.md", "b.md"}, nil)

	g.Expect(got).To(BeEmpty())
}

func TestFollow_OutputIsDeterministicallySorted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a.md", OutgoingLinks: []string{"z.md", "m.md", "b.md"}},
		{Basename: "b.md"},
		{Basename: "m.md"},
		{Basename: "z.md"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a.md"}, nil)

	g.Expect(got).To(Equal([]string{"b.md", "m.md", "z.md"}))
}
```

> **Note on `Graph` shape:** `internal/vaultgraph/graph.go` defines `Graph` with both neighbor accessors and basename presence. Read that file first to confirm the API. If `Graph` exposes outgoing-only iteration, `Follow` may need to iterate every node's outgoing edges to compute backlinks for a given target — that is acceptable for vault sizes here. Adjust the implementation in Step 3 to match whatever accessors actually exist.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL with `undefined: vaultgraph.Follow`.

- [ ] **Step 3: Read the existing `Graph` API and implement `Follow`**

First inspect `internal/vaultgraph/graph.go` to see exactly what `Graph` exposes. Then create `internal/vaultgraph/follow.go`:

```go
package vaultgraph

import "sort"

// Follow expands the cascade frontier. Given a set of files to follow and a
// set of already-read files, returns the deduplicated union of outgoing
// wikilinks AND backlinks from the follow set, minus the follow set itself
// AND the already-read set. Output is sorted ascending for determinism.
//
// Both 'follow' and 'alreadyRead' are basename slices (no [[ ]] wrapping);
// the returned slice is also basenames. Unknown basenames in the input are
// ignored silently.
func Follow(graph Graph, follow, alreadyRead []string) []string {
	excluded := make(map[string]struct{}, len(follow)+len(alreadyRead))
	for _, name := range follow {
		excluded[name] = struct{}{}
	}
	for _, name := range alreadyRead {
		excluded[name] = struct{}{}
	}

	result := make(map[string]struct{})

	for _, source := range follow {
		for _, neighbor := range graph.Neighbors(source) {
			if _, skip := excluded[neighbor]; skip {
				continue
			}
			result[neighbor] = struct{}{}
		}
	}

	out := make([]string, 0, len(result))
	for name := range result {
		out = append(out, name)
	}

	sort.Strings(out)

	return out
}
```

> **If `Graph` does not have a `Neighbors(name) []string` method that returns the *undirected* neighborhood** (outgoing + incoming combined), implement that semantic here. Reading `internal/vaultgraph/graph_test.go:122` (`TestBuildGraph_UndirectedNeighbors`) confirms `BuildGraph` produces an undirected graph — so a `Neighbors` accessor likely already exists. Verify before assuming.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Run lint+coverage**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/vaultgraph/follow.go internal/vaultgraph/follow_test.go
git commit -m "feat(vaultgraph): add Follow for cascade frontier expansion

Returns outgoing + backlink expansion from the follow set, deduplicated and
minus the already-read set. Used by 'engram recall --follow --already-read'
to drive iterative skill-side cascade traversal.

AI-Used: [claude]"
```

---

## Task 4: Rewrite `engram recall` CLI to vault-only with new flag set

Replace the current transcript+memories-only behavior with three modes:
- No args: emit anchors (existing `StartingPoints` output)
- `--recent [--limit N]`: emit `N` most-recent notes (default N=20)
- `--follow X,Y[,...] [--already-read A,B,...]`: emit frontier expansion

Output format: one basename per line (no `[[ ]]` wrapping — change from the current `engram starting-points` which wrapped). The skill will format wikilinks at the consumption layer; emitting raw basenames is simpler and matches `--follow` / `--already-read` input format symmetrically.

**Files:**
- Modify: `internal/cli/targets.go` — `RecallArgs` rewritten, `StartingPointsArgs` removed
- Modify: `internal/cli/cli.go` — `runRecall` rewritten
- Delete: `internal/cli/startingpoints.go`, `internal/cli/startingpoints_test.go`
- Create: `internal/cli/recall_test.go` (if not already present alongside cli_test.go)

- [ ] **Step 1: Write the failing test for no-arg recall (anchors)**

In a new file `internal/cli/recall_test.go`:

```go
package cli_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"engram/internal/cli"
)

func TestRunRecall_NoArgsEmitsAnchorsAsBasenamesOneLinePerEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	mocsDir := filepath.Join(vault, "MOCs")
	g.Expect(os.MkdirAll(mocsDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(mocsDir, "1.2026-03-15.alpha.md"),
		[]byte("MOC content"),
		0o644,
	)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: vault,
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(Equal("1.2026-03-15.alpha.md\n"))
}
```

> **Note:** `cli.RunRecallForTest` is exposed via an `export_test.go` test helper if `runRecall` is unexported. Add such a helper in `internal/cli/export_test.go` if not already present:
>
> ```go
> package cli
>
> import (
> 	"context"
> 	"io"
> )
>
> func RunRecallForTest(ctx context.Context, args RecallArgs, stdout io.Writer) error {
> 	return runRecall(ctx, args, stdout)
> }
> ```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — either the test helper is missing, or `RecallArgs` no longer has `VaultPath`, or output format mismatches.

- [ ] **Step 3: Update `RecallArgs` and delete `StartingPointsArgs`**

In `internal/cli/targets.go`, replace the existing `RecallArgs` struct and remove `StartingPointsArgs`:

```go
// RecallArgs holds parsed flags for the recall subcommand. Recall is
// vault-only; three modes are mutually selected by flag:
//
//   - No args: emit structural anchors (MOCs + per-component winners).
//   - --recent: emit the most-recent notes by filename date prefix.
//   - --follow: emit cascade frontier expansion (outgoing + backlinks).
type RecallArgs struct {
	VaultPath   string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=path to vault root (required)"`
	Recent      bool     `targ:"flag,name=recent,desc=emit most-recent notes by filename date instead of anchors"`
	Limit       int      `targ:"flag,name=limit,desc=cap for --recent output (default 20)"`
	Follow      []string `targ:"flag,name=follow,desc=basenames to expand (outgoing + backlinks)"`
	AlreadyRead []string `targ:"flag,name=already-read,desc=basenames to subtract from output"`
}
```

Delete the `StartingPointsArgs` struct entirely.

In the `Targets` function, delete the `starting-points` registration block (the three lines containing `runStartingPoints`). The `recall` target stays, but its handler will be the rewritten `runRecall`.

> **`targ` slice-flag check:** verify the `targ` library accepts `[]string` flags. If `targ` does not natively support slice flags, fall back to comma-separated strings and split inside `runRecall`. Read `dev/targs.go` and any examples in `internal/cli/` (e.g. `LearnFeedbackArgs`, `LearnFactArgs`) to confirm. If comma-separated is required, change the types to `string` here and adjust the test accordingly.

- [ ] **Step 4: Rewrite `runRecall` in `internal/cli/cli.go`**

Replace the entire current `runRecall` (and the helper `runRecallMemoriesOnly`, `runRecallSessions`) with a vault-only implementation:

```go
func runRecall(ctx context.Context, args RecallArgs, stdout io.Writer) error {
	if args.VaultPath == "" {
		return errors.New("recall: --vault required (or set ENGRAM_VAULT_PATH)")
	}

	fs := &osVaultFS{}

	switch {
	case len(args.Follow) > 0:
		return runRecallFollow(fs, args, stdout)
	case args.Recent:
		return runRecallRecent(fs, args, stdout)
	default:
		return runRecallAnchors(fs, args, stdout)
	}
}

func runRecallAnchors(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	points, err := vaultgraph.StartingPoints(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return emitBasenames(stdout, points)
}

func runRecallRecent(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	notes, err := vaultgraph.ScanVault(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	limit := args.Limit
	if limit == 0 {
		limit = 20
	}

	return emitBasenames(stdout, vaultgraph.Recent(notes, limit))
}

func runRecallFollow(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	notes, err := vaultgraph.ScanVault(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	graph := vaultgraph.BuildGraph(notes)

	return emitBasenames(stdout, vaultgraph.Follow(graph, args.Follow, args.AlreadyRead))
}

func emitBasenames(stdout io.Writer, names []string) error {
	for _, name := range names {
		if _, err := fmt.Fprintln(stdout, name); err != nil {
			return fmt.Errorf("recall: writing output: %w", err)
		}
	}

	return nil
}
```

Move the `osVaultFS` type definition from `internal/cli/startingpoints.go` to `internal/cli/cli.go` (or a small new `internal/cli/vault_fs.go`) before deleting `startingpoints.go`. Add required imports to `cli.go`:

```go
import (
	// ... existing
	"engram/internal/vaultgraph"
)
```

Remove now-unused imports (`engram/internal/recall` if only used by the deleted helpers, `engram/internal/memory`, `os` if unused, `path/filepath` if unused, `engram/internal/llmcmd` if unused). Verify with `targ check-full` later.

- [ ] **Step 5: Delete `internal/cli/startingpoints.go` and `startingpoints_test.go`**

```bash
git rm internal/cli/startingpoints.go internal/cli/startingpoints_test.go
```

If `startingpoints_test.go` contained tests that exercise behavior the new `runRecall` also covers, port those assertions into `internal/cli/recall_test.go` before deletion.

- [ ] **Step 6: Run the test added in Step 1**

Run: `targ test`
Expected: PASS for `TestRunRecall_NoArgsEmitsAnchorsAsBasenamesOneLinePerEntry`. Other tests (cycle, opencode, etc.) should still pass — if any fail, it is likely an import that needs updating from `recall.` to `transcript.` (carried over from Task 1).

- [ ] **Step 7: Add tests for `--recent` and `--follow` modes**

Append to `internal/cli/recall_test.go`:

```go
func TestRunRecall_RecentReturnsMostRecentByDate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o755)).To(Succeed())

	for _, name := range []string{
		"1.2026-01-01.old.md",
		"2.2026-05-01.new.md",
		"3.2026-03-15.mid.md",
	} {
		g.Expect(os.WriteFile(filepath.Join(permDir, name), []byte("body"), 0o644)).To(Succeed())
	}

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: vault,
		Recent:    true,
		Limit:     2,
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(Equal("2.2026-05-01.new.md\n3.2026-03-15.mid.md\n"))
}

func TestRunRecall_FollowReturnsExpansionMinusAlreadyRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o755)).To(Succeed())

	// a links to b and c; b is already-read.
	g.Expect(os.WriteFile(
		filepath.Join(permDir, "1.2026-01-01.a.md"),
		[]byte("body with [[2.2026-01-02.b]] and [[3.2026-01-03.c]]"),
		0o644,
	)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(permDir, "2.2026-01-02.b.md"), []byte("b"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(permDir, "3.2026-01-03.c.md"), []byte("c"), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath:   vault,
		Follow:      []string{"1.2026-01-01.a.md"},
		AlreadyRead: []string{"2.2026-01-02.b.md"},
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(Equal("3.2026-01-03.c.md\n"))
}
```

> **Wikilink parsing nuance:** the test assumes `[[2.2026-01-02.b]]` resolves to basename `2.2026-01-02.b.md`. Read `internal/vaultgraph/parser.go` to confirm how wikilinks-to-basenames normalization works. If the parser requires `[[2.2026-01-02.b.md]]` or some other form, adjust the test content accordingly. The test should reflect the parser's actual behavior, not assume.

- [ ] **Step 8: Run new tests**

Run: `targ test`
Expected: all pass.

- [ ] **Step 9: Run lint+coverage**

Run: `targ check-full`
Expected: clean. Address any unused-import errors by removing the imports.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "refactor(cli): make 'engram recall' vault-only with anchors/recent/follow modes

- Drops --memories-only and transcript-reading from recall.
- Adds --recent (most-recent by filename date) and --follow/--already-read
  (cascade frontier expansion via vaultgraph.Follow).
- Folds 'engram starting-points' behavior into no-arg 'engram recall'.
- Legacy ~/.local/share/engram store remains on disk for later mining.

AI-Used: [claude]"
```

---

## Task 5: Add `engram transcript --from --to`

New CLI command that reads session transcripts in the given date range and emits them with existing `internal/context/StripWithConfig` truncation applied. No LLM summarization — raw-ish stripped output.

**Files:**
- Modify: `internal/cli/targets.go` — add `TranscriptArgs`, register `transcript` target
- Create: `internal/cli/transcript.go` — `runTranscript`
- Create: `internal/cli/transcript_test.go`

- [ ] **Step 1: Add `TranscriptArgs` and register target**

In `internal/cli/targets.go`, add:

```go
// TranscriptArgs holds parsed flags for the transcript subcommand.
type TranscriptArgs struct {
	From          string `targ:"flag,name=from,required,desc=start date inclusive (YYYY-MM-DD)"`
	To            string `targ:"flag,name=to,required,desc=end date inclusive (YYYY-MM-DD)"`
	TranscriptDir string `targ:"flag,name=transcript-dir,env=ENGRAM_TRANSCRIPT_DIR,desc=override transcript directory"`
	ProjectSlug   string `targ:"flag,name=project-slug,desc=project directory slug (defaults to cwd)"`
}
```

In the `Targets` function, after the `recall` registration, add:

```go
targ.Targ(func(ctx context.Context, a TranscriptArgs) {
    errHandler(runTranscript(withLog(ctx), a, stdout))
}).Name("transcript").Description("Read session transcripts in a date range"),
```

- [ ] **Step 2: Write the failing test**

Create `internal/cli/transcript_test.go`:

```go
package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"engram/internal/cli"
)

func TestRunTranscript_EmitsStrippedContentForFilesInDateRange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	// Claude transcript filenames use a session ID; the date filter applies to
	// the file mtime or a known date field — check internal/transcript for the
	// authoritative criterion and align this test to that.
	transcriptPath := filepath.Join(dir, "2026-05-11.jsonl")
	body := `{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"content":"hi back"}}
`
	g.Expect(os.WriteFile(transcriptPath, []byte(body), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		From:          "2026-05-11",
		To:            "2026-05-11",
		TranscriptDir: dir,
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	// Stripped output should contain the user and assistant messages verbatim.
	g.Expect(stdout.String()).To(ContainSubstring("hello"))
	g.Expect(stdout.String()).To(ContainSubstring("hi back"))
}
```

Add the test export to `internal/cli/export_test.go`:

```go
func RunTranscriptForTest(ctx context.Context, args TranscriptArgs, stdout io.Writer) error {
    return runTranscript(ctx, args, stdout)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `targ test`
Expected: FAIL with `undefined: runTranscript`.

- [ ] **Step 4: Implement `runTranscript`**

Create `internal/cli/transcript.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	sessionctx "engram/internal/context"
	"engram/internal/transcript"
)

const dateLayout = "2006-01-02"

// runTranscript emits stripped session-transcript content for files whose
// authored-date falls within [from, to] inclusive. Output reuses
// internal/context/StripWithConfig — full user/llm messages, tool calls
// truncated per the existing config.
func runTranscript(ctx context.Context, args TranscriptArgs, stdout io.Writer) error {
	from, err := time.Parse(dateLayout, args.From)
	if err != nil {
		return fmt.Errorf("transcript: parsing --from: %w", err)
	}

	to, err := time.Parse(dateLayout, args.To)
	if err != nil {
		return fmt.Errorf("transcript: parsing --to: %w", err)
	}

	dirs, err := resolveTranscriptDirs(args)
	if err != nil {
		return fmt.Errorf("transcript: %w", err)
	}

	finder := transcript.NewCompositeSessionFinder(
		transcript.NewSessionFinder(&osDirLister{}),
		transcript.NewOpencodeSessionFinder(transcript.DefaultOpencodeDBPath(), ""),
	)

	entries, err := finder.Find(dirs...)
	if err != nil {
		return fmt.Errorf("transcript: finding sessions: %w", err)
	}

	reader := transcript.NewCompositeTranscriptReader(
		transcript.NewTranscriptReader(&osFileReader{}),
		transcript.NewOpencodeTranscriptReader(transcript.DefaultOpencodeDBPath()),
	)

	for _, entry := range entries {
		if !dateInRange(entry.ModTime, from, to) {
			continue
		}

		body, _, readErr := reader.Read(entry.Path, 0)
		if readErr != nil {
			return fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
		}

		if _, writeErr := fmt.Fprintln(stdout, body); writeErr != nil {
			return fmt.Errorf("transcript: writing output: %w", writeErr)
		}
	}

	return nil
}

func dateInRange(t, from, to time.Time) bool {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return !day.Before(from) && !day.After(to)
}

func resolveTranscriptDirs(args TranscriptArgs) ([]string, error) {
	if args.TranscriptDir != "" {
		return []string{args.TranscriptDir}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	slug := args.ProjectSlug
	if slug == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return nil, fmt.Errorf("cwd: %w", cwdErr)
		}
		slug = ProjectSlugFromPath(cwd)
	}

	return []string{filepath.Join(home, ".claude", "projects", slug)}, nil
}

// silence the unused-import linter if sessionctx is only referenced inside
// the transcript package; keep this comment if no direct call site exists.
var _ = sessionctx.Strip
```

> **Reality-check the transcript reader's API** by reading `internal/transcript/transcript.go` (the moved file). Confirm `FileEntry`'s field is `ModTime` (or whatever it is). Confirm `Reader.Read` signature returns `(string, int, error)` — the second int is bytes consumed and can be ignored here. Adjust the code above to match the actual signatures. The dummy `var _ = sessionctx.Strip` line can be removed if `sessionctx` is not imported (it isn't directly here since the reader already applies `StripWithConfig`).

> **Date-filtering criterion:** the test in Step 2 assumes filename-prefix date filtering, but `FileEntry.ModTime` filters on filesystem mtime. Pick one consistently. Recommended: filter on the transcript's *actual authored-date*, which for Claude transcripts is encoded in the JSONL message timestamps. The simplest first cut is filtering on `ModTime`; revisit if needed. Update the test to write the file then `os.Chtimes` it to the desired date if filtering on mtime.

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test`
Expected: PASS. If the date-range logic doesn't match the test, adjust per the note above (probably an `os.Chtimes` call in the test to set mtime).

- [ ] **Step 6: Run lint+coverage**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat(cli): add 'engram transcript --from --to' command

Reads session transcripts within a date range and emits stripped content
(internal/context/StripWithConfig unchanged — full user/LLM messages, tool
calls truncated). Replaces the transcript-reading capability removed from
'engram recall'.

AI-Used: [claude]"
```

---

## Task 6: Write the merged `skills/recall/SKILL.md`

Replace the current minimal `skills/recall/SKILL.md` with the full merged skill: structure + cascade from `recalling-from-vault`, pre-work trigger framing from `prepare`, "query by task, not by fear" anti-pattern note, and the new frontier-expansion loop driving `engram recall --follow` iteratively until ≥100 memories surfaced or frontier empties.

**Files:**
- Modify (overwrite): `skills/recall/SKILL.md`
- Delete: `skills/prepare/SKILL.md`

> **Skill-editing discipline:** per CLAUDE.md, **always use the `superpowers:writing-skills` skill** when editing any SKILL.md. That skill enforces baseline behavior test (RED) → update skill (GREEN) → behavioral verification. Invoke it before doing the steps below; the steps describe the *content* that should land, not a license to skip the workflow.

- [ ] **Step 1: Invoke `superpowers:writing-skills` and run its baseline behavior test**

Follow the skill's instructions. The baseline behavior test should capture: given a user message like "/recall vault session work", what does Claude currently do under the existing skills? Document the baseline before editing.

- [ ] **Step 2: Delete `skills/prepare/SKILL.md`**

```bash
git rm skills/prepare/SKILL.md
```

- [ ] **Step 3: Overwrite `skills/recall/SKILL.md` with the merged content**

Replace the file's entire contents with:

```markdown
---
name: recall
description: >
  Use when the user invokes recall ("/recall", "recall about X", "what do we
  know about Y"), references prior work ("like we did before", "the auth
  refactor", "didn't we already build"), or before starting new work,
  switching tasks, beginning a feature, changing direction, tackling an issue,
  or any other non-trivial action where prior vault guidance may apply.
  Retrieves relevant notes from the agent-memory vault and structures them for
  the LLM caller.
---

# Recall from the Agent-Memory Vault

Surface relevant notes from the agent-memory vault and inject them into the parent agent's context. Output is structured for an LLM reader; format is human-readable too.

## Overview

The vault is a Zettelkasten — past-you's notes for future-you. Two queries always run in parallel:

1. **Explicit query** — the topic the user named (or recent/anchors for no-arg).
2. **Situational baseline** — features of where you are right now, treated as continuous queries against memory.

Most of this skill's value lives in the situational baseline. Your instinct is to retrieve topically; the situational baseline is how you surface things you didn't know to ask about — tooling gotchas, language conventions, role reminders.

## Vault structure

```
/Users/joe/repos/personal/agent-memory/
  Fleeting/    raw captures, transient
  Permanent/   atomic principle-stated notes; <Luhmann-ID>.YYYY-MM-DD.<slug>.md
  MOCs/        Maps of Content with framing prose and in-prose wikilinks; same filename format
  MEMORY.md    index — names notes; substance is in the notes themselves
```

Permanents are higher-quality than fleetings (promoted, principle-stated, wikilinks established). Fleetings are recent raw signal; missing them means missing the most recent material. Surface fleetings as raw observation, not polished claims — promotion does the principle-stating; recall just preserves the shape.

Notes are LLM-voiced. Wikilinks appear in prose with surrounding context — that context is the relevance signal. Luhmann IDs (`1`, `1a`, `1a1`) signal lineage; following wikilinks reaches Luhmann-adjacent notes by construction.

## Modes

| Mode | Trigger | Explicit query |
|------|---------|----------------|
| **No-arg recap** | User said `/recall` with no topic, or you self-invoke for orientation | `engram recall` (anchors) seeded with `engram recall --recent` (latest activity) |
| **Topic query** | User named a topic, or you formed one from context | The topic, phrased as the user gave it |
| **Self-invoked** | You decided recall applies | Phrase your own seed; treated as a topic query |

If the user's invocation is ambiguous ("recall that thing"), make a best-effort phrasing rather than asking. If results are off-target, the user can refine.

## The retrieval pipeline

### Step 1 — Enumerate your current situation

Before retrieving, list features of your current situation — each becomes a query against memory.

**Apply this test to each candidate:**

> If a future-me on a fresh context were dropped into roughly this same situation, and there were one memory about *this feature alone*, would it be worth surfacing?

If yes, list it. If you can't imagine what that memory would even be about, the feature is either too generic ("coding") or too specific to this exact moment ("a bug at line 47").

**Cast wide.** Your situation includes everything continuously true — tooling, language, platform, project conventions, the kind of operation underway, the user's role and goal, what's loaded into your context. There are usually features you'll only notice on a second pass.

**Produce a list** — 5–15 short queryable phrases. Internal scratch, not part of the final output.

### Step 2 — Form the explicit query

For no-arg: combine anchors and recent activity (see Step 3 cascade).

For topic / self-invoked: phrase as the user gave it, or as you'd phrase a search.

**Query by task, not by fear.** What are you trying to do? Not what might go wrong. "implementing Claude Code hooks" — not "common mistakes when writing hooks." Memories are written to match task descriptions, so query the same way.

### Step 3 — Cascade retrieval via frontier expansion

The skill drives the cascade as a loop calling `engram recall`. The binary is a thin graph primitive; relevance evaluation lives here.

**Initial frontier:**

```bash
engram recall --vault /Users/joe/repos/personal/agent-memory
engram recall --vault /Users/joe/repos/personal/agent-memory --recent --limit 20
```

Union the outputs. These are the initial files to evaluate.

**Loop (until ≥100 surfaced memories OR frontier empties):**

1. **Evaluate the current frontier in parallel via subagents.** For each file in the frontier, dispatch a subagent (or batch a few per subagent) that reads the file and scores relevance against (a) the explicit query and (b) every situational feature from Step 1. Surface notes are those scoring above the relevance threshold.
2. **Track read files.** Maintain a cumulative `--already-read` set that includes every file read so far, whether surfaced or not.
3. **Expand the frontier** by calling:

   ```bash
   engram recall --vault <path> \
     --follow A.md,B.md,C.md \
     --already-read X.md,Y.md,Z.md,...
   ```

   `--follow` = files that scored above threshold *and* whose surrounding prose signaled there is more worth chasing. `--already-read` = the cumulative set.

4. **Repeat** from step 1 with the new frontier.

**Termination:**
- ≥100 surfaced notes → stop and synthesize.
- Empty frontier → stop and synthesize.

**Contradictions.** If two surfaced notes make incompatible claims about the same thing, mark them. The vault preserves contradictions; recall surfaces both, never picks a side.

### Step 4 — Synthesize for context injection

Format for an LLM reader (the parent agent). Wikilinks are required — the parent may re-read source notes for depth. Stay human-readable too.

**If a note matches both the explicit query and a situational feature**, surface it once under the more specifically relevant section and list both signals (e.g., add a `Also matches: <feature>` line). Don't duplicate the same note under two sections.

**Fleetings get their own section.** Fleetings are raw observation, not principle-stated; surface them under `### From recent fleetings` with the observation as-written. Don't translate to a principle.

**Output template** (use the structure; phrase content naturally):

```
## Recall — <mode>

### Vault state
(omit unless something structurally surprising about the vault is worth flagging — layout drift, unusual sparsity. Brief.)

### From your query: <explicit query phrasing>
- [[<wikilink>]] — <one-line claim or principle>
  Context: <1–2 sentence excerpt of in-prose framing>
- ...

### From your situation
- [[<wikilink>]] — <one-line claim>
  Matches: <situational feature(s) this applies to>
  Context: <1–2 sentence excerpt>
- ...

### From recent fleetings (pre-promotion, raw)
(omit section if no fleeting matches; if it has matches, list them here regardless of which seed they matched)
- [[<fleeting-wikilink>]] (fleeting) — <raw observation, as-written, 1–2 lines>
  Matches: <signal it matched — query / situation feature>
- ...

### Contradictions in the vault
(omit section if none)
- [[<note A>]] vs [[<note B>]] on <topic>
  <one-line summary of the disagreement>
```

Empty section — write `(no matches)` rather than omitting. Exception: if a section is empty *because* its matches were consolidated under another section per the dedup rule above, write `(matches consolidated above)`.

## Failure modes

| Situation | Behavior |
|-----------|----------|
| `--vault` not provided and `ENGRAM_VAULT_PATH` unset | `engram recall` errors; report "vault path required" and stop. |
| Vault directory does not exist | Report "vault not found" and stop. Do not create. |
| Vault exists but is empty | Report "vault is empty; no recall produced." Do not fabricate. |
| `engram recall` command not found | Fall back: read every `.md` under `MOCs/` and `Permanent/` and `Fleeting/` directly, scoring as in Step 3. Note the missing binary in *Vault state*. |
| No matches for explicit query | `(no matches)` for that section. Situational baseline may still produce. |
| No matches anywhere | State plainly. Normal early in a vault's life. |
| A note read fails | Log which note, continue with the rest. One bad note ≠ abort. |

## What this skill is not for

- Reading session transcripts. Use `engram transcript --from --to` if you need past-session activity.
- Writing to the vault. Capture and promotion are separate skills (`learn`).
- Inventing memories. If recall would surface nothing, surface nothing.
- Inventing classifications (confidence tiers, freshness scores, priority ranks) the upstream skills don't produce.
- Deduplicating against your prior context. The parent agent handles that; this skill returns full surfaced content.

## Discovery and trigger ceiling

This skill fires when the model recognizes the situation as recall-relevant from the description. Some genuinely relevant moments will be missed because the model didn't realize recall applied. That ceiling is accepted; proactive triggering (hooks) is a separate concern and out of scope for the skill itself.
```

- [ ] **Step 4: Run the skill's pressure tests per `superpowers:writing-skills`**

Verify the behavioral change vs. baseline. The expected change: when the user invokes recall (or starts new work), Claude follows the new loop calling `engram recall` iteratively, instead of the old `engram starting-points` single-shot or the old `engram recall --query` transcript pipeline.

- [ ] **Step 5: Commit**

```bash
git add skills/recall/SKILL.md
git rm skills/prepare/SKILL.md
git commit -m "feat(skills): collapse prepare/recall/recalling-from-vault into one recall skill

- Vault-only retrieval driven by 'engram recall' frontier-expansion loop.
- Absorbs prepare's pre-work trigger framing into the description.
- Adds 'query by task, not by fear' anti-pattern note.
- Termination: >=100 surfaced notes OR frontier empty (replaces hard 3-hop cap).
- Output: LLM-first structured format, human-readable.
- 'engram transcript --from --to' is the replacement for session-transcript reads.

AI-Used: [claude]"
```

---

## Task 7: End-to-end smoke test

Verify the new commands work from the built binary against a small fixture vault.

**Files:**
- Modify: existing CLI smoke-test file (the last commit added one for learn — `b6e54260 test(cli): add end-to-end smoke tests for learn fact and moc`). Find that file and add new test cases.

- [ ] **Step 1: Locate the existing smoke test**

```bash
grep -rln "TestE2E\|smoke\|targ build" --include="*.go" .
```

Read the existing smoke test to understand the harness pattern.

- [ ] **Step 2: Write smoke tests for `engram recall` (anchors / recent / follow) and `engram transcript`**

Add tests that:
1. Build the binary via `targ build` (or the smoke harness's existing build path).
2. Create a temp vault with 2 MOCs, 3 permanents (with wikilinks), 1 fleeting.
3. Run `engram recall --vault <tmp>` and assert it lists the two MOC basenames.
4. Run `engram recall --vault <tmp> --recent --limit 2` and assert it lists the two most-recently-dated files.
5. Run `engram recall --vault <tmp> --follow <a.md> --already-read <b.md>` and assert it lists the expected expansion.
6. Run `engram transcript --from 2026-05-11 --to 2026-05-11 --transcript-dir <tmp-transcripts>` with a fixture JSONL and assert the output contains the user message verbatim.

Use the existing smoke harness conventions (likely `exec.Command` against a built binary). Match the test style of the learn smoke tests.

- [ ] **Step 3: Run the smoke tests**

Run: `targ test`
Expected: all smoke tests pass.

- [ ] **Step 4: Run full lint+coverage**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "test(cli): add end-to-end smoke tests for recall and transcript commands

AI-Used: [claude]"
```

---

## Task 8: Final cleanup

Verify nothing dangling: dead imports, leftover references to the deleted `starting-points` subcommand in docs, stale README content.

**Files:**
- Search-only; modify only as findings dictate.

- [ ] **Step 1: Search for leftover references**

```bash
grep -rn "engram starting-points\|starting-points\|engram recall --query\|engram recall --memories-only" \
  --include="*.go" --include="*.md" --include="*.json" .
```

For each hit:
- If in code: update to the new command form.
- If in docs/comments: update or delete.
- If in tests asserting old behavior: update.

- [ ] **Step 2: Search for orphaned imports**

```bash
targ check-full
```

Address any unused-import or unused-symbol warnings.

- [ ] **Step 3: Verify the skill files**

```bash
ls skills/
# Expected: c4, learn, migrate, recall (NOT prepare).
cat skills/recall/SKILL.md | head -10
# Expected: frontmatter shows merged description.
```

- [ ] **Step 4: Final commit (if cleanup made changes)**

```bash
git add -A
git commit -m "chore: remove leftover references to deprecated recall flags

AI-Used: [claude]"
```

- [ ] **Step 5: Verify the worktree is clean**

```bash
git status
# Expected: clean tree.
git log --oneline main..HEAD
# Expected: ~6-8 focused commits matching the tasks above.
```

---

## Out-of-scope follow-ups

These are not part of this plan; surface them to the user when this work merges:

1. **Mine `~/.local/share/engram/`** — the legacy memory store. Promote any worth-keeping notes into the vault as fleetings.
2. **Retire `~/.claude/skills/recalling-from-vault/`** — replace with a symlink to the repo's `skills/recall/SKILL.md` during dev (manual step).
3. **`engram recall --query`** — currently not on the binary (eval lives in the skill). If TF-IDF or grep-based ranking proves useful later, that's a future addition.
4. **Date-source for `engram transcript` filtering** — currently filters on file mtime (or whichever criterion was chosen in Task 5). Revisit if transcript-internal timestamps prove more accurate.

---

## Self-review notes

- **Spec coverage:** all five decisions from the conversation are reflected — (1) `engram transcript --from --to` with ISO dates (Task 5), (2) frontier-expander `engram recall` with `--follow`/`--already-read` and termination at ≥100 notes or empty frontier (Tasks 2/3/4/Skill Step 3), (3) drop `--memories-only` but leave the legacy store untouched on disk (Task 4 Step 4; out-of-scope follow-up #1), (4) merged skill in repo at `skills/recall/` (Task 6), (5) `engram starting-points` folded into `engram recall` (Task 4 Steps 3/5).
- **Placeholders:** none. Every step shows the code or the command. Where the engineer must read a real file before adjusting (e.g. `Graph` API, transcript reader signatures, wikilink parser format), the note is explicit and bounded.
- **Type consistency:** `RecallArgs.Follow` and `RecallArgs.AlreadyRead` are `[]string` throughout (with a fallback-to-comma-separated note in Task 4 Step 3 if `targ` doesn't support slices). `Recent` and `Follow` consistently take and return basenames (no `[[ ]]` wrapping). `FileEntry`/`ModTime` flagged for verification in Task 5.
