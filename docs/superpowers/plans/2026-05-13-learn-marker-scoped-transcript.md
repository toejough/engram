# Learn-marker Scoped Transcript Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `engram transcript` callable with no flags by reading "since last `/learn`" using a per-project marker, capped at a byte budget so it never blows the context window. The `learn` skill calls it with `--mark` to advance the marker on each pass.

**Architecture:** New package `internal/learnmarker` reads/writes a single timestamp file at `${XDG_STATE_HOME:-$HOME/.local/state}/engram/projects/<slug>/last-learn-at`. `engram transcript` flags become optional: when `--from` is absent, derive it from the marker (or fall back to 24h ago if no marker exists); when `--to` is absent, default to now. Add `--mark` (advance marker to now after read) and `--max-bytes` (default 200_000) flags. Byte cap truncates oldest content first — most recent transcript wins when over budget. Learn skill's workflow step 1 calls `engram transcript --mark` to fetch and scope-advance in one shot.

**Tech Stack:** Go 1.25+, targ build system, imptest + gomega + rapid for tests, `time.Time` (RFC3339Nano on disk), filesystem-backed state with DI via small `FS` interface.

---

## File Structure

**New:**
- `internal/learnmarker/learnmarker.go` — `StateDirFromHome`, `MarkerPath`, `Read`, `Write`, `FS` interface
- `internal/learnmarker/learnmarker_test.go` — blackbox tests (`package learnmarker_test`)
- `internal/learnmarker/osfs.go` — `OSFS` adapter wrapping `os.ReadFile`/`os.WriteFile`/`os.MkdirAll` for production wiring

**Modify:**
- `internal/cli/transcript.go` — relax required validators, add `--mark` + `--max-bytes` flags, derive `From`/`To` defaults, wire marker advance, change byte budget plumbing in `emitTranscripts`
- `internal/cli/transcript_test.go` — extend to cover marker-derived defaults, byte cap, `--mark` write-back
- `skills/learn/SKILL.md` — workflow step 1 instructs `engram transcript --mark` for session-log retrieval

**Note:** `internal/transcript/transcript.go` already exposes `JSONLReader.Read(path string, budgetBytes int) (string, int, error)` — the second return is bytes consumed, which we route into a running tally in `emitTranscripts`. No changes needed inside `internal/transcript/`.

---

## Task 1: learnmarker package — paths and FS interface

**Files:**
- Create: `internal/learnmarker/learnmarker.go`
- Create: `internal/learnmarker/learnmarker_test.go`

- [ ] **Step 1: Write the failing test for StateDirFromHome**

Create `internal/learnmarker/learnmarker_test.go`:

```go
package learnmarker_test

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/engram/internal/learnmarker"
)

func TestStateDirFromHome_DefaultsToLocalState(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := learnmarker.StateDirFromHome("/Users/joe", func(string) string { return "" })

	g.Expect(dir).To(gomega.Equal("/Users/joe/.local/state/engram"))
}

func TestStateDirFromHome_RespectsXDGStateHome(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	getenv := func(key string) string {
		if key == "XDG_STATE_HOME" {
			return "/custom/state"
		}
		return ""
	}

	dir := learnmarker.StateDirFromHome("/Users/joe", getenv)

	g.Expect(dir).To(gomega.Equal("/custom/state/engram"))
}

func TestMarkerPath_JoinsStateDirAndSlug(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := learnmarker.MarkerPath("/state/engram", "Users-joe-repos-foo")

	g.Expect(path).To(gomega.Equal("/state/engram/projects/Users-joe-repos-foo/last-learn-at"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `learnmarker` package does not exist.

- [ ] **Step 3: Write the minimal implementation**

Create `internal/learnmarker/learnmarker.go`:

```go
// Package learnmarker tracks the per-project timestamp of the most recent
// successful transcript scope advance ("last /learn"). The marker is a
// single RFC3339Nano timestamp written to a file under the XDG state dir.
package learnmarker

import "path/filepath"

// StateDirFromHome returns the engram state directory.
// Respects $XDG_STATE_HOME if set, otherwise defaults to $HOME/.local/state/engram.
// getenv is injected so callers control environment access (pass os.Getenv in production).
func StateDirFromHome(home string, getenv func(string) string) string {
	if xdg := getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram")
	}

	return filepath.Join(home, ".local", "state", "engram")
}

// MarkerPath returns the full path to the last-learn-at file for a given project slug.
func MarkerPath(stateDir, projectSlug string) string {
	return filepath.Join(stateDir, "projects", projectSlug, "last-learn-at")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS — `internal/learnmarker` shows ok.

- [ ] **Step 5: Commit**

```bash
git add internal/learnmarker/learnmarker.go internal/learnmarker/learnmarker_test.go
git commit -m "feat(learnmarker): state dir + marker path helpers"
```

---

## Task 2: learnmarker package — Read and Write

**Files:**
- Modify: `internal/learnmarker/learnmarker.go`
- Modify: `internal/learnmarker/learnmarker_test.go`
- Create: `internal/learnmarker/osfs.go`

- [ ] **Step 1: Write the failing test for FS interface + Read missing-file behavior**

Append to `internal/learnmarker/learnmarker_test.go`:

```go
import (
	"errors"
	"os"
	"testing"
	"time"
	// ... existing imports
)

type fakeFS struct {
	files     map[string][]byte
	mkdirCall string
	writeErr  error
	readErr   error
}

func newFakeFS() *fakeFS { return &fakeFS{files: map[string][]byte{}} }

func (f *fakeFS) ReadFile(path string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	b, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

func (f *fakeFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.files[path] = data
	return nil
}

func (f *fakeFS) MkdirAll(path string, _ os.FileMode) error {
	f.mkdirCall = path
	return nil
}

func TestRead_ReturnsNotFoundWhenMarkerMissing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()

	_, found, err := learnmarker.Read(fs, "/state/engram/projects/foo/last-learn-at")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(found).To(gomega.BeFalse())
}

func TestRead_ReturnsTimestampWhenMarkerPresent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()
	want := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)
	fs.files["/state/engram/projects/foo/last-learn-at"] = []byte(want.Format(time.RFC3339Nano))

	got, found, err := learnmarker.Read(fs, "/state/engram/projects/foo/last-learn-at")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got.Equal(want)).To(gomega.BeTrue())
}

func TestRead_WrapsCorruptTimestampError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()
	fs.files["/p"] = []byte("not-a-timestamp")

	_, _, err := learnmarker.Read(fs, "/p")

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("learnmarker: parsing")))
}

func TestWrite_CreatesParentDirAndWritesRFC3339Nano(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()
	when := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	err := learnmarker.Write(fs, "/state/engram/projects/foo/last-learn-at", when)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(fs.mkdirCall).To(gomega.Equal("/state/engram/projects/foo"))
	g.Expect(string(fs.files["/state/engram/projects/foo/last-learn-at"])).
		To(gomega.Equal(when.Format(time.RFC3339Nano)))
}

func TestWrite_PropagatesMkdirError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := &fakeFS{files: map[string][]byte{}, writeErr: errors.New("disk full")}

	err := learnmarker.Write(fs, "/p", time.Now())

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("learnmarker: writing")))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — `learnmarker.FS`, `Read`, `Write` undefined.

- [ ] **Step 3: Implement FS interface + Read + Write**

Append to `internal/learnmarker/learnmarker.go`:

```go
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FS is the minimal filesystem surface learnmarker needs.
// OSFS in osfs.go wraps os.* for production; tests inject fakes.
type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// Read returns the marker timestamp at path. The bool return is true when the
// marker file existed; false (with nil error) when it did not — callers handle
// the absent case (first-run) without treating it as an error.
func Read(fs FS, path string) (time.Time, bool, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("learnmarker: reading %s: %w", path, err)
	}

	t, parseErr := time.Parse(time.RFC3339Nano, string(data))
	if parseErr != nil {
		return time.Time{}, false, fmt.Errorf("learnmarker: parsing %s: %w", path, parseErr)
	}

	return t, true, nil
}

// Write replaces the marker file at path with the RFC3339Nano-formatted timestamp.
// Creates parent directories as needed (0o755 perms).
func Write(fs FS, path string, when time.Time) error {
	if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("learnmarker: writing %s: %w", path, err)
	}

	if err := fs.WriteFile(path, []byte(when.Format(time.RFC3339Nano)), 0o644); err != nil {
		return fmt.Errorf("learnmarker: writing %s: %w", path, err)
	}

	return nil
}
```

- [ ] **Step 4: Implement OSFS adapter**

Create `internal/learnmarker/osfs.go`:

```go
package learnmarker

import "os"

// OSFS is the production FS, backed by package os.
type OSFS struct{}

func (OSFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (OSFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (OSFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS — `internal/learnmarker` shows ok with all four tests passing.

- [ ] **Step 6: Commit**

```bash
git add internal/learnmarker/learnmarker.go internal/learnmarker/learnmarker_test.go internal/learnmarker/osfs.go
git commit -m "feat(learnmarker): read/write marker file with FS DI"
```

---

## Task 3: Wire optional From/To + marker default into transcript CLI

**Files:**
- Modify: `internal/cli/transcript.go`
- Modify: `internal/cli/transcript_test.go`

- [ ] **Step 1: Write the failing test for From defaulting to marker**

Append to `internal/cli/transcript_test.go`:

```go
func TestResolveTimeWindow_UsesMarkerWhenFromMissing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, to, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "", To: "", Marker: markerTime, MarkerFound: true, Now: now},
	)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(from.Equal(markerTime)).To(gomega.BeTrue())
	g.Expect(to.Equal(now)).To(gomega.BeTrue())
}

func TestResolveTimeWindow_FallsBackTo24hWhenNoMarker(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, to, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "", To: "", MarkerFound: false, Now: now},
	)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(from.Equal(now.Add(-24 * time.Hour))).To(gomega.BeTrue())
	g.Expect(to.Equal(now)).To(gomega.BeTrue())
}

func TestResolveTimeWindow_ExplicitFromOverridesMarker(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	explicit := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, _, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "2026-05-10", Marker: markerTime, MarkerFound: true, Now: now},
	)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(from.Equal(explicit)).To(gomega.BeTrue())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — `cli.ResolveTimeWindow`, `cli.TimeWindowInputs` undefined.

- [ ] **Step 3: Implement TimeWindowInputs + ResolveTimeWindow**

In `internal/cli/transcript.go`, add (above `runTranscript`):

```go
// TimeWindowInputs is the resolution input for ResolveTimeWindow. From/To are
// raw CLI strings (may be empty); Marker is the marker timestamp; MarkerFound
// distinguishes a missing-marker first-run from a zero-time marker. Now is the
// current time, injected for testability.
type TimeWindowInputs struct {
	From, To           string
	Marker             time.Time
	MarkerFound        bool
	Now                time.Time
}

const defaultLookback = 24 * time.Hour

// ResolveTimeWindow returns the effective (from, to) time range for a
// transcript scan. Precedence: explicit --from > marker > now - 24h.
// Explicit --to > now. Date-only forms ("YYYY-MM-DD") have inclusive
// end-of-day semantics applied to To.
func ResolveTimeWindow(in TimeWindowInputs) (time.Time, time.Time, error) {
	from := in.Now.Add(-defaultLookback)
	if in.MarkerFound {
		from = in.Marker
	}
	if in.From != "" {
		parsed, err := parseDate(in.From)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = parsed
	}

	to := in.Now
	if in.To != "" {
		parsed, err := parseDate(in.To)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		if len(in.To) == len("2006-01-02") {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		to = parsed
	}

	return from, to, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS — three new tests in `cli` package pass.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/transcript.go internal/cli/transcript_test.go
git commit -m "feat(cli): ResolveTimeWindow with marker + 24h-fallback precedence"
```

---

## Task 4: Add --mark and --max-bytes flags; relax From/To required

**Files:**
- Modify: `internal/cli/transcript.go`
- Modify: `internal/cli/transcript_test.go`

- [ ] **Step 1: Write the failing test for runTranscript using marker default**

Append to `internal/cli/transcript_test.go`:

```go
func TestRunTranscript_NoFlagsUsesMarkerAndDoesNotErrorOnMissingFrom(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Set up an in-memory marker for the test project slug.
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPath(stateDir, slug)
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(gomega.Succeed())
	markerTime := time.Now().Add(-2 * time.Hour).UTC()
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(gomega.Succeed())

	// Run transcript with no --from/--to; expect no missing-flag error.
	var stdout bytes.Buffer
	err := cli.RunTranscriptForTest(cli.TranscriptArgs{
		ProjectSlug: slug,
		StateDir:    stateDir,
		TranscriptDir: t.TempDir(), // empty dir; we only care that flags resolved
	}, &stdout)

	g.Expect(err).NotTo(gomega.HaveOccurred())
}

func TestRunTranscript_MarkFlagAdvancesMarkerToNow(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	var stdout bytes.Buffer
	before := time.Now().UTC()
	err := cli.RunTranscriptForTest(cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, &stdout)
	after := time.Now().UTC()

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	got, _ := os.ReadFile(learnmarker.MarkerPath(stateDir, slug))
	parsed, parseErr := time.Parse(time.RFC3339Nano, string(got))
	g.Expect(parseErr).NotTo(gomega.HaveOccurred())
	g.Expect(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second))).
		To(gomega.BeTrue())
}
```

Top of `transcript_test.go`, ensure imports include `learnmarker` and `bytes` and `path/filepath` and `os`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — `cli.RunTranscriptForTest`, `TranscriptArgs.Mark`, `TranscriptArgs.StateDir` undefined.

- [ ] **Step 3: Add flags to TranscriptArgs and refactor runTranscript**

Replace `TranscriptArgs` and `runTranscript` in `internal/cli/transcript.go`:

```go
type TranscriptArgs struct {
	From          string `targ:"flag,name=from,desc=start date (YYYY-MM-DD or RFC3339); defaults to marker or 24h ago"`
	To            string `targ:"flag,name=to,desc=end date (YYYY-MM-DD or RFC3339); defaults to now"`
	TranscriptDir string `targ:"flag,name=transcript-dir,env=ENGRAM_TRANSCRIPT_DIR,desc=path to transcript directory"`
	ProjectSlug   string `targ:"flag,name=project-slug,desc=project slug for transcript-dir and marker derivation"`
	StateDir      string `targ:"flag,name=state-dir,env=ENGRAM_STATE_DIR,desc=state directory (defaults to XDG_STATE_HOME/engram)"`
	Mark          bool   `targ:"flag,name=mark,desc=advance the last-learn marker to now after reading"`
	MaxBytes      int    `targ:"flag,name=max-bytes,desc=byte cap for transcript output (default 200000)"`
}

const defaultMaxBytes = 200_000

// RunTranscriptForTest is an exported entry point for the cli package's tests.
// Production callers go through runTranscript via the targ Target wiring.
func RunTranscriptForTest(args TranscriptArgs, stdout io.Writer) error {
	return runTranscript(context.Background(), args, stdout)
}

func runTranscript(_ context.Context, args TranscriptArgs, stdout io.Writer) error {
	transcriptDir := args.TranscriptDir
	if err := applyTranscriptDirDefault(&transcriptDir, args.ProjectSlug, os.Getwd); err != nil {
		return err
	}

	stateDir := args.StateDir
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("transcript: resolving home dir: %w", err)
		}
		stateDir = learnmarker.StateDirFromHome(home, os.Getenv)
	}

	slug := args.ProjectSlug
	if slug == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("transcript: resolving working directory: %w", err)
		}
		slug = ProjectSlugFromPath(cwd)
	}

	markerPath := learnmarker.MarkerPath(stateDir, slug)
	markerTime, markerFound, err := learnmarker.Read(learnmarker.OSFS{}, markerPath)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	fromTime, toTime, err := ResolveTimeWindow(TimeWindowInputs{
		From: args.From, To: args.To,
		Marker: markerTime, MarkerFound: markerFound, Now: now,
	})
	if err != nil {
		return err
	}

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	lister := &osDirLister{}
	finder := transcript.NewCompositeSessionFinder(transcript.NewSessionFinder(lister))
	fileReader := &osFileReader{}
	reader := transcript.NewCompositeTranscriptReader(transcript.NewJSONLReader(fileReader))

	entries, findErr := finder.Find(transcriptDir)
	if findErr != nil {
		return fmt.Errorf("transcript: finding sessions: %w", findErr)
	}

	filtered := filterByDateRange(entries, fromTime, toTime)
	slices.Reverse(filtered) // chronological for output

	if err := emitTranscripts(reader, filtered, maxBytes, stdout); err != nil {
		return err
	}

	if args.Mark {
		if err := learnmarker.Write(learnmarker.OSFS{}, markerPath, now); err != nil {
			return err
		}
	}

	return nil
}
```

Also delete `errTranscriptFromRequired` and `errTranscriptToRequired` declarations, plus any unused imports they leave behind. Add `"github.com/toejough/engram/internal/learnmarker"` and `"context"` to imports if not already there.

- [ ] **Step 4: Modify emitTranscripts to honor the byte cap**

Replace `emitTranscripts` in `internal/cli/transcript.go`:

```go
// emitTranscripts writes entries (oldest first) to stdout, capped at maxBytes
// of total content. When the cap is reached, oldest content is dropped —
// the most recent transcript wins. A one-line truncation notice is emitted
// when content was dropped.
func emitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	stdout io.Writer,
) error {
	// Read all content into a slice of strings, then drop from the front until under cap.
	contents := make([]string, 0, len(entries))
	total := 0
	for _, entry := range entries {
		content, _, readErr := reader.Read(entry.Path, math.MaxInt32)
		if readErr != nil {
			return fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
		}
		contents = append(contents, content)
		total += len(content)
	}

	dropped := 0
	for total > maxBytes && len(contents) > 1 {
		total -= len(contents[0])
		dropped++
		contents = contents[1:]
	}

	if dropped > 0 {
		notice := fmt.Sprintf("[engram transcript: dropped %d oldest session(s) to fit %d-byte cap]\n", dropped, maxBytes)
		if _, err := io.WriteString(stdout, notice); err != nil {
			return fmt.Errorf("transcript: writing output: %w", err)
		}
	}

	for _, content := range contents {
		if _, err := io.WriteString(stdout, content); err != nil {
			return fmt.Errorf("transcript: writing output: %w", err)
		}
	}

	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS — all existing transcript tests still pass; new no-flag and --mark tests pass.

If existing tests fail because they assumed required From/To validators: update each failing test to either pass explicit From/To (preserving original intent) or use the new no-flag behavior. Do not delete the validators' coverage — convert them into "explicit values still work" tests.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/transcript.go internal/cli/transcript_test.go
git commit -m "feat(cli): transcript no-flag scan via marker + byte cap + --mark advance"
```

---

## Task 5: Byte-cap behavior — explicit test for truncation order

**Files:**
- Modify: `internal/cli/transcript_test.go`

- [ ] **Step 1: Write the failing test for byte-cap truncation order**

Append to `internal/cli/transcript_test.go`:

```go
func TestEmitTranscripts_DropsOldestWhenOverCap(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Three entries, each 100 bytes; cap at 150 should keep the newest only.
	mkContent := func(prefix string) string { return prefix + strings.Repeat("x", 99) }
	reader := &fakeReader{contents: map[string]string{
		"/a": mkContent("A"),
		"/b": mkContent("B"),
		"/c": mkContent("C"),
	}}
	entries := []transcript.FileEntry{
		{Path: "/a", Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		{Path: "/b", Time: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
		{Path: "/c", Time: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer
	err := cli.EmitTranscriptsForTest(reader, entries, 150, &buf)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	out := buf.String()
	g.Expect(out).To(gomega.ContainSubstring("dropped 2 oldest"))
	g.Expect(out).To(gomega.ContainSubstring("C"))
	g.Expect(out).NotTo(gomega.ContainSubstring("A"))
	g.Expect(out).NotTo(gomega.ContainSubstring("B"))
}
```

Add `fakeReader` test helper if absent:

```go
type fakeReader struct{ contents map[string]string }

func (f *fakeReader) Read(path string, _ int) (string, int, error) {
	c, ok := f.contents[path]
	if !ok {
		return "", 0, fmt.Errorf("fakeReader: no content for %s", path)
	}
	return c, len(c), nil
}
```

- [ ] **Step 2: Add the EmitTranscriptsForTest export**

Append to `internal/cli/transcript.go`:

```go
// EmitTranscriptsForTest is an exported entry point so the cli_test package can
// exercise emitTranscripts directly without going through the full runTranscript
// flow. Production code does not call this.
func EmitTranscriptsForTest(reader transcript.Reader, entries []transcript.FileEntry, maxBytes int, stdout io.Writer) error {
	return emitTranscripts(reader, entries, maxBytes, stdout)
}
```

- [ ] **Step 3: Run tests to verify pass**

Run: `targ test`
Expected: PASS — truncation order test green.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/transcript.go internal/cli/transcript_test.go
git commit -m "test(cli): pin transcript byte-cap truncation order (drop oldest first)"
```

---

## Task 6: Update learn skill to use `engram transcript --mark`

**Files:**
- Modify: `skills/learn/SKILL.md`

- [ ] **Step 1: Update workflow §1 to point at the new mechanism**

In `skills/learn/SKILL.md`, replace the §"1. Identify candidates" block. Currently it reads:

```
### 1. Identify candidates

Scan the in-context conversation (default) or session logs (when source isn't loaded) for:

- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about tools, idioms, conventions, gotchas
- **Recurring patterns** — behaviors that should be codified
```

Replace with:

```
### 1. Identify candidates

Source of candidates depends on what's already loaded:

- **In-context conversation** (default when this skill fires mid-session): scan the recent turns directly. No CLI call needed.
- **Session logs** (when this skill fires fresh, or the recent in-context window doesn't cover the just-completed work): run `engram transcript --mark` to fetch transcripts since the last `/learn` for this project. The `--mark` flag advances the per-project marker after read so the next pass starts from the right place. Output is capped at ~200KB by default; if `engram` reports oldest sessions dropped, that's expected — most-recent content wins.

In either source, look for:

- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about tools, idioms, conventions, gotchas
- **Recurring patterns** — behaviors that should be codified
```

- [ ] **Step 2: Quick TDD check via baseline pressure test**

Per CLAUDE.md, all SKILL.md edits go through `superpowers:writing-skills` (TDD). Before declaring the edit complete:

Run: dispatch one subagent with the scenario "long session, user invoked /learn after substantial work, in-context conversation no longer contains the early turns" and verify the subagent reaches for `engram transcript --mark` rather than `engram transcript --from ... --to ...`.

Document the result inline (one or two lines noting which sub-step the subagent picked). If the subagent reaches for the old `--from`/`--to` form, sharpen the §1 wording to mention `--mark` more prominently.

- [ ] **Step 3: Commit**

```bash
git add skills/learn/SKILL.md
git commit -m "feat(learn): use engram transcript --mark for session-log retrieval"
```

---

## Task 7: Manual end-to-end smoke

**Files:** none — this is a smoke test.

- [ ] **Step 1: Build and verify the CLI accepts no-flag invocation**

```bash
targ build
engram transcript --max-bytes 50000
```

Expected: either prints recent transcripts (≤50KB worth, oldest dropped) or prints nothing with no error if no transcripts exist for the cwd's slug. **Must not error** with "from required" or "to required."

- [ ] **Step 2: Verify --mark writes the marker file**

```bash
rm -f ~/.local/state/engram/projects/Users-joe-repos-personal-engram-worktrees-opencode-plugin/last-learn-at
engram transcript --mark --max-bytes 1000
cat ~/.local/state/engram/projects/Users-joe-repos-personal-engram-worktrees-opencode-plugin/last-learn-at
```

Expected: file contains an RFC3339Nano timestamp from within the last few seconds.

- [ ] **Step 3: Verify subsequent no-flag call uses the marker (returns less content)**

```bash
engram transcript --max-bytes 1000  # should now scan only since the marker; likely returns very little
```

Expected: noticeably less output than Step 1 (since the marker is fresh).

- [ ] **Step 4: Run targ check-full to catch any lint regressions**

```bash
targ check-full
```

Expected: PASS. Fix any issues before commit.

- [ ] **Step 5: Commit any check-full fixes**

```bash
git add -A
git commit -m "chore: lint fixes for learn-marker scoped transcript"
```

Skip if there are no fixes.

---

## Acceptance criteria

- `engram transcript` with no flags exits 0 and produces output (or empty) without complaining about missing `--from`/`--to`.
- `engram transcript --mark` writes/updates the marker file at the XDG state path.
- Subsequent no-flag invocations after `--mark` scan only since the marker.
- Byte cap defaults to 200_000 and truncates oldest content first, with a `[engram transcript: dropped N oldest session(s) ...]` notice line.
- Explicit `--from`/`--to` flags still work and override the marker default.
- `skills/learn/SKILL.md` step §1 references `engram transcript --mark` for the session-logs branch.
- `targ test` and `targ check-full` both green.

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-13-learn-marker-scoped-transcript.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
