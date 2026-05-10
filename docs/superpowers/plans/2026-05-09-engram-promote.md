# `engram promote` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `engram promote {feedback|fact|moc}` CLI subcommand that writes a structured permanent (or MOC) note to `<vault>/{Permanent,MOCs}/<luhmann-id>.<YYYY-MM-DD>.<slug>.md`, with Luhmann ID computation under file lock so concurrent promotions cannot collide on ID assignment. Update `~/.claude/skills/promoting-to-permanent-notes/SKILL.md` to invoke the new command and to specify the structured-frontmatter + body-template format converged on in conversation.

**Architecture:**
- Pure-business-logic runner (`runPromote`) with all I/O behind injected function deps (`Now`, `Stdin`, `Getenv`, `StatDir`, `ListIDs`, `WriteNew`, `Lock`, `DeleteFleeting`).
- Luhmann logic (parse, sort, next-ID computation given target+relation) lives as pure functions in a separate file (`luhmann.go`) and tests independently.
- Frontmatter and body assembly are pure functions, one per type (`feedback`, `fact`, `moc`).
- Production adapter implements an exclusive file lock at `<vault>/.luhmann.lock` (Unix `flock(2)` via `golang.org/x/sys/unix`), held across read-existing-IDs → compute-next → write-permanent → optional-delete-fleeting → release. Lock is advisory but cooperative for any process using this binary.
- Subcommand registers as a `targ.Group("promote", …)` mirroring the existing `learn` group's shape.
- Skill update uses `superpowers:writing-skills` TDD to verify the new format propagates correctly.

**Tech Stack:** Go, `targ` build/CLI framework (`github.com/toejough/targ`), gomega for assertions (`github.com/onsi/gomega`), `golang.org/x/sys/unix` for file locking, standard library `os`/`time`/`io`/`path/filepath`/`regexp`.

**Inputs from brainstorming (this conversation):**
- Three memory types: `feedback` (SBIA), `fact` (SPO), `moc` (framing prose)
- Frontmatter shape per type (see Task 4)
- Body templates per type (see Task 5)
- No H1 title in body — filename is the display name
- MOCs have no constituent list — backlinks carry membership and rationale
- Source attribution lives in frontmatter only (no body source line)

---

## File Structure

**Engram repo (`/Users/joe/repos/personal/engram-worktrees/opencode-plugin/`):**

Create:
- `internal/cli/promote.go` — `runPromote` orchestrator, `PromoteDeps` struct, `PromoteArgs` field accessors, frontmatter/body assemblers, validation helpers
- `internal/cli/promote_test.go` — blackbox unit tests (`package cli_test`)
- `internal/cli/luhmann.go` — pure ID parser, sort comparator, next-ID computation
- `internal/cli/luhmann_test.go` — blackbox tests for `luhmann.go`

Modify:
- `internal/cli/targets.go` — add `PromoteFeedbackArgs`, `PromoteFactArgs`, `PromoteMOCArgs` arg structs and register the `promote` group in `Targets()`
- `internal/cli/cli.go` — add `osPromoteFS` production adapter (StatDir, ListIDs, WriteNew, Lock, DeleteFleeting)
- `internal/cli/export_test.go` — re-export the runner-with-deps and pure helpers for blackbox testing

**Skill repo (`/Users/joe/.claude/skills/`):**

Modify:
- `promoting-to-permanent-notes/SKILL.md` — replace the permanent-note format section with structured-frontmatter + body templates per type; replace the MOC format section with frontmatter + framing-paragraph (no constituent list); replace the workflow section's "Apply" step with `engram promote` invocation patterns; preserve the "information vs. knowledge" bar, the two-dispositions logic, and the contradiction-surfacing guidance

---

## Phase A: Pure Luhmann logic (no I/O)

### Task 1: Luhmann ID parsing

**Files:**
- Create: `internal/cli/luhmann.go`
- Create: `internal/cli/luhmann_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/luhmann_test.go`:

```go
package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestParseLuhmannID_TopLevelDigit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, err := cli.ExportParseLuhmannID("1")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal([]string{"1"}))
}

func TestParseLuhmannID_AlternatingSegments(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, err := cli.ExportParseLuhmannID("1a3b")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal([]string{"1", "a", "3", "b"}))
}

func TestParseLuhmannID_MultiCharSegments(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, err := cli.ExportParseLuhmannID("12ab3")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal([]string{"12", "ab", "3"}))
}

func TestParseLuhmannID_RejectsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, err := cli.ExportParseLuhmannID("")
	g.Expect(err).To(HaveOccurred())
}

func TestParseLuhmannID_RejectsLeadingLetter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, err := cli.ExportParseLuhmannID("a1")
	g.Expect(err).To(HaveOccurred())
}
```

Add to `internal/cli/export_test.go`:

```go
var ExportParseLuhmannID = parseLuhmannID
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error — `parseLuhmannID` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/luhmann.go`:

```go
package cli

import (
	"errors"
	"fmt"
	"unicode"
)

var errLuhmannEmpty = errors.New("luhmann: empty ID")
var errLuhmannLeadingLetter = errors.New("luhmann: ID must start with a digit")

// parseLuhmannID splits a Luhmann ID into alternating digit/letter segments.
// "1a3b" → ["1", "a", "3", "b"]. "12ab3" → ["12", "ab", "3"]. Top-level segment
// must be digits.
func parseLuhmannID(id string) ([]string, error) {
	if id == "" {
		return nil, errLuhmannEmpty
	}

	if !unicode.IsDigit(rune(id[0])) {
		return nil, fmt.Errorf("%w: %q", errLuhmannLeadingLetter, id)
	}

	segments := make([]string, 0, 4) //nolint:mnd // initial capacity hint
	current := []rune{rune(id[0])}
	currentIsDigit := unicode.IsDigit(rune(id[0]))

	for _, r := range id[1:] {
		isDigit := unicode.IsDigit(r)
		if isDigit == currentIsDigit {
			current = append(current, r)
			continue
		}

		segments = append(segments, string(current))
		current = []rune{r}
		currentIsDigit = isDigit
	}

	segments = append(segments, string(current))

	return segments, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS for all 5 tests in `TestParseLuhmannID_*`.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/luhmann.go internal/cli/luhmann_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): luhmann ID parser

Splits Luhmann IDs into alternating digit/letter segments. Pure function, no I/O.

AI-Used: [claude]
EOF
)"
```

---

### Task 2: Sort Luhmann IDs in tree order

**Files:**
- Modify: `internal/cli/luhmann.go`
- Modify: `internal/cli/luhmann_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/luhmann_test.go`:

```go
func TestSortLuhmannIDs_TreeOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	in := []string{"2", "1b", "1a1", "1", "1a", "10", "1a2"}
	cli.ExportSortLuhmannIDs(in)
	g.Expect(in).To(Equal([]string{"1", "1a", "1a1", "1a2", "1b", "2", "10"}))
}

func TestSortLuhmannIDs_NumericNotLexical(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	in := []string{"10", "2", "1"}
	cli.ExportSortLuhmannIDs(in)
	g.Expect(in).To(Equal([]string{"1", "2", "10"}))
}
```

Add to `export_test.go`:

```go
var ExportSortLuhmannIDs = sortLuhmannIDs
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error.

- [ ] **Step 3: Implement sort**

Add to `internal/cli/luhmann.go`:

```go
import (
	"sort"
	"strconv"
	// ... existing imports ...
)

// sortLuhmannIDs sorts in tree order: parent before children, numeric segments
// compared numerically, alphabetic segments compared lexically. Mutates the input.
func sortLuhmannIDs(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		return luhmannLess(ids[i], ids[j])
	})
}

func luhmannLess(a, b string) bool {
	aSegs, _ := parseLuhmannID(a)
	bSegs, _ := parseLuhmannID(b)

	for idx := 0; idx < len(aSegs) && idx < len(bSegs); idx++ {
		if aSegs[idx] == bSegs[idx] {
			continue
		}

		aIsDigit := unicode.IsDigit(rune(aSegs[idx][0]))
		if aIsDigit {
			aNum, _ := strconv.Atoi(aSegs[idx])
			bNum, _ := strconv.Atoi(bSegs[idx])
			return aNum < bNum
		}

		return aSegs[idx] < bSegs[idx]
	}

	return len(aSegs) < len(bSegs)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/luhmann.go internal/cli/luhmann_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): luhmann ID tree-order sort

Numeric segments compared numerically (10 > 2), alphabetic lexically.

AI-Used: [claude]
EOF
)"
```

---

### Task 3: Compute next Luhmann ID

**Files:**
- Modify: `internal/cli/luhmann.go`
- Modify: `internal/cli/luhmann_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/luhmann_test.go`:

```go
func TestNextLuhmannID_NewTopLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := []string{"1", "1a", "2", "2a"}
	got, err := cli.ExportNextLuhmannID(existing, "", "top")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("3"))
}

func TestNextLuhmannID_NewTopLevel_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, err := cli.ExportNextLuhmannID(nil, "", "top")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("1"))
}

func TestNextLuhmannID_FirstChild(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := []string{"1", "2"}
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("1a"))
}

func TestNextLuhmannID_NextChild(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := []string{"1", "1a", "1b"}
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("1c"))
}

func TestNextLuhmannID_FirstGrandchild(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := []string{"1", "1a"}
	got, err := cli.ExportNextLuhmannID(existing, "1a", "continuation")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("1a1"))
}

func TestNextLuhmannID_Sibling(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := []string{"1", "1a"}
	got, err := cli.ExportNextLuhmannID(existing, "1a", "sibling")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("1b"))
}

func TestNextLuhmannID_SiblingOfTopLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := []string{"1", "2"}
	got, err := cli.ExportNextLuhmannID(existing, "1", "sibling")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("3"))
}

func TestNextLuhmannID_LetterRollover(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	existing := buildLetterChildren("1", 'a', 'z')
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("1aa"))
}

func buildLetterChildren(parent string, from, to rune) []string {
	out := []string{parent}
	for r := from; r <= to; r++ {
		out = append(out, parent+string(r))
	}
	return out
}
```

Add to `export_test.go`:

```go
var ExportNextLuhmannID = nextLuhmannID
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error.

- [ ] **Step 3: Implement next-ID computation**

Add to `internal/cli/luhmann.go`:

```go
import (
	"strings"
	// ... existing ...
)

const (
	relationTop          = "top"
	relationContinuation = "continuation"
	relationSibling      = "sibling"
)

var errLuhmannRelation = errors.New("luhmann: relation must be top, continuation, or sibling")
var errLuhmannTargetEmpty = errors.New("luhmann: target required for continuation/sibling")
var errLuhmannSiblingTopLevelMustBeTop = errors.New(
	"luhmann: sibling of top-level requires relation=top",
)

// nextLuhmannID computes the next available Luhmann ID given existing IDs and a (target, relation).
// relation=top  → next available top-level (ignores target)
// relation=continuation → next child of target
// relation=sibling → next sibling of target (target must have a parent; for top-level use relation=top)
func nextLuhmannID(existing []string, target, relation string) (string, error) {
	switch relation {
	case relationTop:
		return nextTopLevel(existing), nil
	case relationContinuation:
		if target == "" {
			return "", errLuhmannTargetEmpty
		}
		return nextChild(existing, target)
	case relationSibling:
		if target == "" {
			return "", errLuhmannTargetEmpty
		}
		return nextSibling(existing, target)
	default:
		return "", fmt.Errorf("%w: got %q", errLuhmannRelation, relation)
	}
}

func nextTopLevel(existing []string) string {
	max := 0
	for _, id := range existing {
		segs, err := parseLuhmannID(id)
		if err != nil || len(segs) != 1 {
			continue
		}
		n, err := strconv.Atoi(segs[0])
		if err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

func nextChild(existing []string, parent string) (string, error) {
	parentSegs, err := parseLuhmannID(parent)
	if err != nil {
		return "", err
	}
	depth := len(parentSegs)
	childSegmentIsDigit := depth%2 == 1 // depth 1 (top) → letters at depth 2 → digits at depth 3 ...
	maxDigit := 0
	maxLetter := ""
	for _, id := range existing {
		if !strings.HasPrefix(id, parent) || id == parent {
			continue
		}
		segs, parseErr := parseLuhmannID(id)
		if parseErr != nil || len(segs) != depth+1 {
			continue
		}
		seg := segs[depth]
		if childSegmentIsDigit {
			n, atoiErr := strconv.Atoi(seg)
			if atoiErr == nil && n > maxDigit {
				maxDigit = n
			}
		} else if seg > maxLetter {
			maxLetter = seg
		}
	}
	if childSegmentIsDigit {
		return parent + strconv.Itoa(maxDigit+1), nil
	}
	return parent + nextLetter(maxLetter), nil
}

func nextSibling(existing []string, target string) (string, error) {
	targetSegs, err := parseLuhmannID(target)
	if err != nil {
		return "", err
	}
	if len(targetSegs) == 1 {
		return "", fmt.Errorf("%w: %q", errLuhmannSiblingTopLevelMustBeTop, target)
	}
	parent := strings.Join(targetSegs[:len(targetSegs)-1], "")
	return nextChild(existing, parent)
}

// nextLetter returns "a" if cur is empty, else the next letter ("a"→"b", "z"→"aa", "az"→"ba").
func nextLetter(cur string) string {
	if cur == "" {
		return "a"
	}
	runes := []rune(cur)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < 'z' {
			runes[i]++
			return string(runes)
		}
		runes[i] = 'a'
	}
	return "a" + string(runes)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS for all `TestNextLuhmannID_*`.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/luhmann.go internal/cli/luhmann_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): next-luhmann-ID computation

Pure function: given existing IDs and a (target, relation), returns the next ID.
Supports top-level, continuation (child), and sibling relations.

AI-Used: [claude]
EOF
)"
```

---

## Phase B: Frontmatter and body assembly

### Task 4: Frontmatter assembly per type

**Files:**
- Create: `internal/cli/promote.go`
- Create: `internal/cli/promote_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/promote_test.go`:

```go
package cli_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRenderFrontmatter_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFeedbackFrontmatter(cli.ExportFeedbackFields{
		Situation: "writing concurrent Go code with context",
		Behavior:  "ignoring context cancellation",
		Impact:    "leaks goroutines on shutdown",
		Action:    "always check ctx.Done() in select loops",
		Luhmann:   "9z",
		Source:    "session log foo, 2026-05-09 12:00 UTC",
	}, when)
	g.Expect(got).To(Equal(strings.Join([]string{
		"---",
		"type: feedback",
		"situation: writing concurrent Go code with context",
		"behavior: ignoring context cancellation",
		"impact: leaks goroutines on shutdown",
		"action: always check ctx.Done() in select loops",
		`luhmann: "9z"`,
		"created: 2026-05-09",
		"source: session log foo, 2026-05-09 12:00 UTC",
		"---",
		"",
	}, "\n")))
}

func TestRenderFrontmatter_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFactFrontmatter(cli.ExportFactFields{
		Situation: "reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem dressed as coordination",
		Luhmann:   "11",
		Source:    "session log bar, 2026-05-09 13:00 UTC",
	}, when)
	g.Expect(got).To(ContainSubstring("type: fact"))
	g.Expect(got).To(ContainSubstring("subject: subagent dispatch"))
}

func TestRenderFrontmatter_MOC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderMOCFrontmatter(cli.ExportMOCFields{
		Topic:   "llm rationalization patterns under pressure",
		Luhmann: "5",
		Source:  "constructed from cluster analysis, 2026-05-09",
	}, when)
	g.Expect(got).To(ContainSubstring("type: moc"))
	g.Expect(got).To(ContainSubstring("topic: llm rationalization patterns under pressure"))
}
```

Add to `export_test.go`:

```go
type ExportFeedbackFields = feedbackFields
type ExportFactFields = factFields
type ExportMOCFields = mocFields

var ExportRenderFeedbackFrontmatter = renderFeedbackFrontmatter
var ExportRenderFactFrontmatter = renderFactFrontmatter
var ExportRenderMOCFrontmatter = renderMOCFrontmatter
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation errors — types and functions undefined.

- [ ] **Step 3: Implement frontmatter assembly**

Create `internal/cli/promote.go`:

```go
package cli

import (
	"fmt"
	"strings"
	"time"
)

type feedbackFields struct {
	Situation string
	Behavior  string
	Impact    string
	Action    string
	Luhmann   string
	Source    string
}

type factFields struct {
	Situation string
	Subject   string
	Predicate string
	Object    string
	Luhmann   string
	Source    string
}

type mocFields struct {
	Topic   string
	Luhmann string
	Source  string
}

func renderFeedbackFrontmatter(f feedbackFields, when time.Time) string {
	return strings.Join([]string{
		"---",
		"type: feedback",
		"situation: " + f.Situation,
		"behavior: " + f.Behavior,
		"impact: " + f.Impact,
		"action: " + f.Action,
		fmt.Sprintf("luhmann: %q", f.Luhmann),
		"created: " + when.Format(dateFormat),
		"source: " + f.Source,
		"---",
		"",
	}, "\n")
}

func renderFactFrontmatter(f factFields, when time.Time) string {
	return strings.Join([]string{
		"---",
		"type: fact",
		"situation: " + f.Situation,
		"subject: " + f.Subject,
		"predicate: " + f.Predicate,
		"object: " + f.Object,
		fmt.Sprintf("luhmann: %q", f.Luhmann),
		"created: " + when.Format(dateFormat),
		"source: " + f.Source,
		"---",
		"",
	}, "\n")
}

func renderMOCFrontmatter(f mocFields, when time.Time) string {
	return strings.Join([]string{
		"---",
		"type: moc",
		"topic: " + f.Topic,
		fmt.Sprintf("luhmann: %q", f.Luhmann),
		"created: " + when.Format(dateFormat),
		"source: " + f.Source,
		"---",
		"",
	}, "\n")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/promote.go internal/cli/promote_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): frontmatter assemblers for feedback/fact/moc

Pure render functions producing YAML frontmatter blocks per memory type.

AI-Used: [claude]
EOF
)"
```

---

### Task 5: Body assembly per type

**Files:**
- Modify: `internal/cli/promote.go`
- Modify: `internal/cli/promote_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/promote_test.go`:

```go
func TestRenderBody_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFeedbackBody(cli.ExportFeedbackFields{
		Situation: "orchestrating multi-step work as the main LLM under context pressure",
		Action:    "set up a task list with self-contained briefs and dispatch; if a small model cannot finish a subtask, shrink the task",
	}, "Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n")
	g.Expect(got).To(Equal(
		"Lesson learned: when orchestrating multi-step work as the main LLM under context pressure, " +
			"set up a task list with self-contained briefs and dispatch; if a small model cannot finish a subtask, shrink the task.\n" +
			"\n" +
			"Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n"))
}

func TestRenderBody_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFactBody(cli.ExportFactFields{
		Situation: "reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem dressed as coordination",
	}, "Related to:\n- [[X]] — adjacent.\n")
	g.Expect(got).To(Equal(
		"Information learned: when in reasoning about agent coordination, " +
			"subagent dispatch is fundamentally a verification problem dressed as coordination.\n" +
			"\n" +
			"Related to:\n- [[X]] — adjacent.\n"))
}

func TestRenderBody_MOC_PassesThrough(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderMOCBody("This cluster names a recurring pattern of LLM rationalization under pressure.\n")
	g.Expect(got).To(Equal("This cluster names a recurring pattern of LLM rationalization under pressure.\n"))
}
```

Add to `export_test.go`:

```go
var ExportRenderFeedbackBody = renderFeedbackBody
var ExportRenderFactBody = renderFactBody
var ExportRenderMOCBody = renderMOCBody
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation errors.

- [ ] **Step 3: Implement body assemblers**

Add to `internal/cli/promote.go`:

```go
func renderFeedbackBody(f feedbackFields, relatedSection string) string {
	formula := fmt.Sprintf("Lesson learned: when %s, %s.\n", f.Situation, f.Action)
	return formula + "\n" + relatedSection
}

func renderFactBody(f factFields, relatedSection string) string {
	formula := fmt.Sprintf(
		"Information learned: when in %s, %s %s %s.\n",
		f.Situation, f.Subject, f.Predicate, f.Object,
	)
	return formula + "\n" + relatedSection
}

func renderMOCBody(framing string) string {
	return framing
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/promote.go internal/cli/promote_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): body assemblers for feedback/fact/moc

Feedback/fact get a formulaic restatement line; MOC body is the framing prose passed through.

AI-Used: [claude]
EOF
)"
```

---

## Phase C: Filename derivation and validation

### Task 6: Permanent/MOC path derivation

**Files:**
- Modify: `internal/cli/promote.go`
- Modify: `internal/cli/promote_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/promote_test.go`:

```go
func TestPromotePath_Permanent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportPromotePath("/vault", "feedback", "1a3", "subagent-driven-recovery", when)
	g.Expect(got).To(Equal("/vault/Permanent/1a3.2026-05-09.subagent-driven-recovery.md"))
}

func TestPromotePath_MOC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportPromotePath("/vault", "moc", "5", "llm-rationalization-patterns", when)
	g.Expect(got).To(Equal("/vault/MOCs/5.2026-05-09.llm-rationalization-patterns.md"))
}
```

Add to `export_test.go`:

```go
var ExportPromotePath = promotePath
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error.

- [ ] **Step 3: Implement path derivation**

Add to `internal/cli/promote.go`:

```go
import "path/filepath"

const (
	permanentSubdir = "Permanent"
	mocSubdir       = "MOCs"
	typeFeedback    = "feedback"
	typeFact        = "fact"
	typeMOC         = "moc"
)

func promotePath(vault, memType, luhmann, slug string, when time.Time) string {
	subdir := permanentSubdir
	if memType == typeMOC {
		subdir = mocSubdir
	}
	filename := fmt.Sprintf("%s.%s.%s.md", luhmann, when.Format(dateFormat), slug)
	return filepath.Join(vault, subdir, filename)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/promote.go internal/cli/promote_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): permanent/moc path derivation for engram promote

AI-Used: [claude]
EOF
)"
```

---

### Task 7: ID extraction from filenames

**Files:**
- Modify: `internal/cli/promote.go`
- Modify: `internal/cli/promote_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/promote_test.go`:

```go
func TestExtractLuhmannFromFilename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, ok := cli.ExportExtractLuhmannFromFilename("1a3.2026-05-09.subagent-recovery.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("1a3"))
}

func TestExtractLuhmannFromFilename_RejectsBadFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, ok := cli.ExportExtractLuhmannFromFilename("README.md")
	g.Expect(ok).To(BeFalse())
}
```

Add to `export_test.go`:

```go
var ExportExtractLuhmannFromFilename = extractLuhmannFromFilename
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error.

- [ ] **Step 3: Implement extraction**

Add to `internal/cli/promote.go`:

```go
var luhmannFilenamePattern = regexp.MustCompile(
	`^([0-9][0-9a-z]*)\.\d{4}-\d{2}-\d{2}\..+\.md$`,
)

func extractLuhmannFromFilename(name string) (string, bool) {
	m := luhmannFilenamePattern.FindStringSubmatch(name)
	if m == nil {
		return "", false
	}
	return m[1], true
}
```

Add `import "regexp"` if not already present.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/promote.go internal/cli/promote_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): luhmann ID extraction from permanent/moc filenames

AI-Used: [claude]
EOF
)"
```

---

## Phase D: runPromote orchestrator

### Task 8: PromoteDeps + runPromote

**Files:**
- Modify: `internal/cli/promote.go`
- Modify: `internal/cli/promote_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/cli/promote_test.go`:

```go
func TestRunPromote_Feedback_WritesExpectedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var lockAcquired, lockReleased bool
	var writtenPath string
	var writtenContent []byte
	deletedFleetings := []string{}

	deps := cli.PromoteDeps{
		Now:    func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Stdin:  strings.NewReader("Related to:\n- [[X]] — adjacent.\n"),
		Getenv: func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(vault string) ([]string, error) {
			return []string{"1", "2"}, nil
		},
		Lock: func(string) (func(), error) {
			lockAcquired = true
			return func() { lockReleased = true }, nil
		},
		WriteNew: func(path string, data []byte) error {
			writtenPath = path
			writtenContent = data
			return nil
		},
		DeleteFleeting: func(path string) error {
			deletedFleetings = append(deletedFleetings, path)
			return nil
		},
	}

	args := cli.PromoteArgs{
		Type:           "feedback",
		Slug:           "ctx-cancellation-rule",
		Vault:          "/vault",
		Target:         "",
		Relation:       "top",
		Source:         "session log foo, 2026-05-09 12:00 UTC",
		Situation:      "writing concurrent Go code",
		Behavior:       "ignoring ctx.Done()",
		Impact:         "leaks goroutines",
		Action:         "always check ctx.Done() in select",
		DeleteFleeting: "",
	}

	var stdout strings.Builder
	err := cli.ExportRunPromote(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(lockAcquired).To(BeTrue())
	g.Expect(lockReleased).To(BeTrue())
	g.Expect(writtenPath).To(Equal("/vault/Permanent/3.2026-05-09.ctx-cancellation-rule.md"))
	g.Expect(string(writtenContent)).To(ContainSubstring("type: feedback"))
	g.Expect(string(writtenContent)).To(ContainSubstring("Lesson learned: when writing concurrent Go code"))
	g.Expect(deletedFleetings).To(BeEmpty())
}

func TestRunPromote_DeletesFleetingWhenAsked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deletedFleetings := []string{}

	deps := cli.PromoteDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Stdin:   strings.NewReader("Related to:\n- [[X]] — adjacent.\n"),
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
		DeleteFleeting: func(path string) error {
			deletedFleetings = append(deletedFleetings, path)
			return nil
		},
	}

	args := cli.PromoteArgs{
		Type:           "feedback",
		Slug:           "rule",
		Vault:          "/vault",
		Relation:       "top",
		Situation:      "x",
		Behavior:       "y",
		Impact:         "z",
		Action:         "w",
		DeleteFleeting: "/vault/Fleeting/2026-05-08.note.md",
	}

	var stdout strings.Builder
	err := cli.ExportRunPromote(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(deletedFleetings).To(Equal([]string{"/vault/Fleeting/2026-05-08.note.md"}))
}

func TestRunPromote_RejectsUnknownType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := cli.PromoteDeps{
		Now: func() time.Time { return time.Now() },
		Stdin: strings.NewReader(""),
		Getenv: func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock: func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
		DeleteFleeting: func(string) error { return nil },
	}
	args := cli.PromoteArgs{Type: "principle", Slug: "x", Vault: "/v", Relation: "top"}
	var stdout strings.Builder
	err := cli.ExportRunPromote(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}
```

Add to `export_test.go`:

```go
var ExportRunPromote = runPromote
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation errors — `PromoteDeps`, `PromoteArgs`, `runPromote` undefined.

- [ ] **Step 3: Implement runPromote**

Add to `internal/cli/promote.go`:

```go
import (
	"context"
	"errors"
	"io"
)

// PromoteDeps holds injected dependencies for runPromote. All fields required.
type PromoteDeps struct {
	Now            func() time.Time
	Stdin          io.Reader
	Getenv         func(string) string
	StatDir        func(string) error
	ListIDs        func(vault string) ([]string, error)
	Lock           func(vault string) (release func(), err error)
	WriteNew       func(path string, data []byte) error
	DeleteFleeting func(path string) error
}

// PromoteArgs holds the parsed flags for the promote subcommand.
type PromoteArgs struct {
	Type           string
	Slug           string
	Vault          string
	Target         string
	Relation       string
	Source         string
	DeleteFleeting string

	// feedback / fact share these
	Situation string
	// feedback only
	Behavior string
	Impact   string
	Action   string
	// fact only
	Subject   string
	Predicate string
	Object    string
	// moc only
	Topic string
}

var errPromoteUnknownType = errors.New("promote: type must be feedback, fact, or moc")

func runPromote(_ context.Context, args PromoteArgs, deps PromoteDeps, stdout io.Writer) error {
	slugErr := validateSlug(args.Slug)
	if slugErr != nil {
		return fmt.Errorf("promote: %w", slugErr)
	}

	vault, err := resolveVault(args.Vault, deps.Getenv)
	if err != nil {
		return fmt.Errorf("promote: %w", err)
	}

	if dirErr := deps.StatDir(vault); dirErr != nil {
		return fmt.Errorf("promote: vault %s: %w", vault, dirErr)
	}

	body, bodyErr := io.ReadAll(deps.Stdin)
	if bodyErr != nil {
		return fmt.Errorf("promote: reading stdin: %w", bodyErr)
	}

	release, lockErr := deps.Lock(vault)
	if lockErr != nil {
		return fmt.Errorf("promote: acquiring lock: %w", lockErr)
	}
	defer release()

	existing, listErr := deps.ListIDs(vault)
	if listErr != nil {
		return fmt.Errorf("promote: listing existing IDs: %w", listErr)
	}

	luhmann, idErr := nextLuhmannID(existing, args.Target, args.Relation)
	if idErr != nil {
		return fmt.Errorf("promote: %w", idErr)
	}

	when := deps.Now()
	path := promotePath(vault, args.Type, luhmann, args.Slug, when)

	content, contentErr := assemblePromoteContent(args, luhmann, when, string(body))
	if contentErr != nil {
		return fmt.Errorf("promote: %w", contentErr)
	}

	if writeErr := deps.WriteNew(path, []byte(content)); writeErr != nil {
		return fmt.Errorf("promote: writing %s: %w", path, writeErr)
	}

	if args.DeleteFleeting != "" {
		if delErr := deps.DeleteFleeting(args.DeleteFleeting); delErr != nil {
			return fmt.Errorf("promote: deleting fleeting %s: %w", args.DeleteFleeting, delErr)
		}
	}

	_, _ = fmt.Fprintln(stdout, path)
	return nil
}

func assemblePromoteContent(args PromoteArgs, luhmann string, when time.Time, body string) (string, error) {
	switch args.Type {
	case typeFeedback:
		f := feedbackFields{
			Situation: args.Situation, Behavior: args.Behavior, Impact: args.Impact,
			Action: args.Action, Luhmann: luhmann, Source: args.Source,
		}
		return renderFeedbackFrontmatter(f, when) + renderFeedbackBody(f, body), nil
	case typeFact:
		f := factFields{
			Situation: args.Situation, Subject: args.Subject, Predicate: args.Predicate,
			Object: args.Object, Luhmann: luhmann, Source: args.Source,
		}
		return renderFactFrontmatter(f, when) + renderFactBody(f, body), nil
	case typeMOC:
		f := mocFields{Topic: args.Topic, Luhmann: luhmann, Source: args.Source}
		return renderMOCFrontmatter(f, when) + renderMOCBody(body), nil
	default:
		return "", fmt.Errorf("%w: got %q", errPromoteUnknownType, args.Type)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS for all three tests.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/promote.go internal/cli/promote_test.go internal/cli/export_test.go
git commit -m "$(cat <<'EOF'
feat(cli): runPromote orchestrator with DI'd lock/list/write/delete

Acquires lock, lists existing IDs, computes next Luhmann ID, writes file, optionally deletes the originating fleeting.

AI-Used: [claude]
EOF
)"
```

---

## Phase E: Production adapters

### Task 9: osPromoteFS adapter (StatDir, ListIDs, Lock, WriteNew, DeleteFleeting)

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/adapters_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/adapters_test.go`:

```go
func TestOsPromoteFS_ListIDs_ReturnsBothPermanentAndMOC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "1.2026-05-09.foo.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "1a.2026-05-09.bar.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "MOCs", "5.2026-05-09.moc.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "README.md"), nil, 0o600)).To(Succeed())

	fs := cli.ExportNewOsPromoteFS()
	got, err := fs.ListIDs(vault)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(ConsistOf("1", "1a", "5"))
}

func TestOsPromoteFS_Lock_ExclusiveAcrossSecondAcquisition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()

	fs := cli.ExportNewOsPromoteFS()
	release1, err := fs.Lock(vault)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	// Second acquisition in the same process should block; we test with a non-blocking variant.
	// Use a goroutine with a deadline to confirm second is blocked.
	done := make(chan struct{})
	go func() {
		release2, err2 := fs.Lock(vault)
		g.Expect(err2).NotTo(HaveOccurred())
		if release2 != nil {
			release2()
		}
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("second Lock should not have succeeded while first holds")
	case <-time.After(100 * time.Millisecond):
		// expected: blocked
	}

	release1()
	<-done // now second should succeed
}

func TestOsPromoteFS_DeleteFleeting_RemovesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "fleeting.md")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.DeleteFleeting(path)).To(Succeed())
	_, err := os.Stat(path)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}
```

Add to `export_test.go`:

```go
func ExportNewOsPromoteFS() *osPromoteFS { return &osPromoteFS{} }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error.

- [ ] **Step 3: Implement osPromoteFS**

Add to `internal/cli/cli.go`:

```go
import (
	"golang.org/x/sys/unix"
)

const luhmannLockFile = ".luhmann.lock"

type osPromoteFS struct{}

func (*osPromoteFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", errNotADirectory, path)
	}
	return nil
}

func (*osPromoteFS) ListIDs(vault string) ([]string, error) {
	out := []string{}
	for _, sub := range []string{"Permanent", "MOCs"} {
		entries, err := os.ReadDir(filepath.Join(vault, sub))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", sub, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			id, ok := extractLuhmannFromFilename(e.Name())
			if !ok {
				continue
			}
			out = append(out, id)
		}
	}
	return out, nil
}

func (*osPromoteFS) Lock(vault string) (func(), error) {
	path := filepath.Join(vault, luhmannLockFile)
	const perm = 0o600
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}
	if flockErr := unix.Flock(int(f.Fd()), unix.LOCK_EX); flockErr != nil {
		_ = f.Close()
		return nil, fmt.Errorf("flock: %w", flockErr)
	}
	release := func() {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
	}
	return release, nil
}

func (*osPromoteFS) WriteNew(path string, data []byte) error {
	const perm = 0o600
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // path from caller
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, writeErr := f.Write(data); writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
	}
	return nil
}

func (*osPromoteFS) DeleteFleeting(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}
```

Run `go mod tidy` if needed to pull `golang.org/x/sys/unix`.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS for all three.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli.go internal/cli/adapters_test.go internal/cli/export_test.go go.mod go.sum
git commit -m "$(cat <<'EOF'
feat(cli): osPromoteFS adapter — stat/list-ids/flock/write/delete

Production filesystem adapter for engram promote. Uses unix.Flock for serializing
concurrent Luhmann ID assignment.

AI-Used: [claude]
EOF
)"
```

---

## Phase F: Subcommand wiring

### Task 10: PromoteFeedbackArgs / PromoteFactArgs / PromoteMOCArgs + targets

**Files:**
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Add the three arg structs**

Add to `internal/cli/targets.go` (after existing arg structs):

```go
// CommonPromoteArgs holds shared flags for promote subcommands.
type CommonPromoteArgs struct {
	Slug           string `targ:"flag,name=slug,desc=kebab-case tag for the filename"`
	Vault          string `targ:"flag,name=vault,env=ENGRAM_VAULT_DIR,desc=vault root directory"`
	Target         string `targ:"flag,name=target,desc=Luhmann ID this note relates to (empty for top-level)"`
	Relation       string `targ:"flag,name=relation,desc=top|continuation|sibling"`
	Source         string `targ:"flag,name=source,desc=provenance string for the source field"`
	DeleteFleeting string `targ:"flag,name=delete-fleeting,desc=path to fleeting note to delete after success"`
}

// PromoteFeedbackArgs holds parsed flags for the promote feedback subcommand.
type PromoteFeedbackArgs struct {
	CommonPromoteArgs
	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
}

// PromoteFactArgs holds parsed flags for the promote fact subcommand.
type PromoteFactArgs struct {
	CommonPromoteArgs
	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// PromoteMOCArgs holds parsed flags for the promote moc subcommand.
type PromoteMOCArgs struct {
	CommonPromoteArgs
	Topic string `targ:"flag,name=topic,desc=cluster topic name"`
}
```

- [ ] **Step 2: Register the promote group in `Targets()`**

Add to the `Targets()` return slice in `targets.go`, after the existing `learn` group:

```go
targ.Group("promote",
	targ.Targ(func(ctx context.Context, a PromoteFeedbackArgs) {
		errHandler(runPromoteFromFeedbackArgs(ctx, a, stdout))
	}).Name("feedback").Description("Promote a feedback note to Permanent/"),
	targ.Targ(func(ctx context.Context, a PromoteFactArgs) {
		errHandler(runPromoteFromFactArgs(ctx, a, stdout))
	}).Name("fact").Description("Promote a fact note to Permanent/"),
	targ.Targ(func(ctx context.Context, a PromoteMOCArgs) {
		errHandler(runPromoteFromMOCArgs(ctx, a, stdout))
	}).Name("moc").Description("Promote a MOC note to MOCs/"),
),
```

- [ ] **Step 3: Add the three thin wrappers**

Add to `internal/cli/promote.go`:

```go
func runPromoteFromFeedbackArgs(ctx context.Context, a PromoteFeedbackArgs, stdout io.Writer) error {
	deps := newOsPromoteDeps()
	return runPromote(ctx, PromoteArgs{
		Type:           typeFeedback,
		Slug:           a.Slug,
		Vault:          a.Vault,
		Target:         a.Target,
		Relation:       a.Relation,
		Source:         a.Source,
		DeleteFleeting: a.DeleteFleeting,
		Situation:      a.Situation,
		Behavior:       a.Behavior,
		Impact:         a.Impact,
		Action:         a.Action,
	}, deps, stdout)
}

func runPromoteFromFactArgs(ctx context.Context, a PromoteFactArgs, stdout io.Writer) error {
	deps := newOsPromoteDeps()
	return runPromote(ctx, PromoteArgs{
		Type:           typeFact,
		Slug:           a.Slug,
		Vault:          a.Vault,
		Target:         a.Target,
		Relation:       a.Relation,
		Source:         a.Source,
		DeleteFleeting: a.DeleteFleeting,
		Situation:      a.Situation,
		Subject:        a.Subject,
		Predicate:      a.Predicate,
		Object:         a.Object,
	}, deps, stdout)
}

func runPromoteFromMOCArgs(ctx context.Context, a PromoteMOCArgs, stdout io.Writer) error {
	deps := newOsPromoteDeps()
	return runPromote(ctx, PromoteArgs{
		Type:           typeMOC,
		Slug:           a.Slug,
		Vault:          a.Vault,
		Target:         a.Target,
		Relation:       a.Relation,
		Source:         a.Source,
		DeleteFleeting: a.DeleteFleeting,
		Topic:          a.Topic,
	}, deps, stdout)
}

func newOsPromoteDeps() PromoteDeps {
	fs := &osPromoteFS{}
	return PromoteDeps{
		Now:            time.Now,
		Stdin:          os.Stdin,
		Getenv:         os.Getenv,
		StatDir:        fs.StatDir,
		ListIDs:        fs.ListIDs,
		Lock:           fs.Lock,
		WriteNew:       fs.WriteNew,
		DeleteFleeting: fs.DeleteFleeting,
	}
}
```

Add `import "os"` to `promote.go` if not already present.

- [ ] **Step 4: Build to verify wiring**

Run: `targ build`
Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/targets.go internal/cli/promote.go
git commit -m "$(cat <<'EOF'
feat(cli): wire engram promote {feedback|fact|moc} subcommands

Three sibling subcommands under the promote group, each producing a permanent
note (feedback/fact) or MOC with locked Luhmann ID assignment.

AI-Used: [claude]
EOF
)"
```

---

### Task 11: End-to-end smoke test

**Files:**
- Modify: `internal/cli/cli_test.go` (or new `promote_smoke_test.go`)

- [ ] **Step 1: Write the smoke test**

Add to `internal/cli/cli_test.go`:

```go
func TestEngramPromote_Feedback_EndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())

	binPath := filepath.Join(t.TempDir(), "engram")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	cmd.Dir = projectRoot(t)
	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), "build failed: %s", out)
	if err != nil {
		return
	}

	run := exec.Command(binPath, "promote", "feedback",
		"--slug", "ctx-rule",
		"--vault", vault,
		"--relation", "top",
		"--source", "smoke test",
		"--situation", "writing concurrent Go code",
		"--behavior", "ignoring ctx",
		"--impact", "leaks goroutines",
		"--action", "check ctx.Done()",
	)
	run.Stdin = strings.NewReader("Related to:\n- [[X]] — adjacent.\n")
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)
	if runErr != nil {
		return
	}

	expectedPath := filepath.Join(vault, "Permanent")
	entries, err := os.ReadDir(expectedPath)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(entries).To(HaveLen(1))
	name := entries[0].Name()
	g.Expect(name).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-rule\.md$`))

	body, err := os.ReadFile(filepath.Join(expectedPath, name))
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(string(body)).To(ContainSubstring("type: feedback"))
	g.Expect(string(body)).To(ContainSubstring("Lesson learned: when writing concurrent Go code, check ctx.Done()."))
	g.Expect(string(body)).To(ContainSubstring("Related to:"))
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/cli → ../..
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
```

Add the `os/exec` import to the imports of `cli_test.go` if not already present.

- [ ] **Step 2: Run the smoke test**

Run: `targ test`
Expected: PASS — file written under `Permanent/` with expected name and content.

- [ ] **Step 3: Manually drive the binary**

```bash
targ build
mkdir -p /tmp/promote-smoke/{Permanent,MOCs}
echo "Related to:\n- [[X]] — test\n" | ./bin/engram promote moc \
    --slug test-cluster \
    --vault /tmp/promote-smoke \
    --relation top \
    --source "smoke test" \
    --topic "a smoke test cluster"
ls /tmp/promote-smoke/MOCs/
cat /tmp/promote-smoke/MOCs/*.md
```

Expected: a `<id>.<date>.test-cluster.md` file with `type: moc`, `topic: a smoke test cluster`, and the related-to bullets in the body. Verify `.luhmann.lock` was cleaned up.

- [ ] **Step 4: Run lint + coverage**

Run: `targ check-full`
Expected: no warnings; coverage on new code ≥ existing thresholds.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(cli): end-to-end smoke test for engram promote

Builds the binary, invokes it against a temp vault, and verifies the file shape.

AI-Used: [claude]
EOF
)"
```

---

## Phase G: Update the promotion skill

### Task 12: Update `promoting-to-permanent-notes` skill via writing-skills TDD

**Files:**
- Modify: `~/.claude/skills/promoting-to-permanent-notes/SKILL.md`

This task MUST use `superpowers:writing-skills` per the project rule "ALWAYS use the superpowers:writing-skills skill when editing any SKILL.md file. No exceptions."

The behavioral changes the skill must produce:

1. **Permanent notes have YAML frontmatter** with type-specific fields:
   - `feedback`: type, situation, behavior, impact, action, luhmann, created, source
   - `fact`: type, situation, subject, predicate, object, luhmann, created, source
2. **MOCs have YAML frontmatter** with: type, topic, luhmann, created, source
3. **Permanent body is**:
   - For `feedback`: `Lesson learned: when [situation], [action].\n\nRelated to:\n- [[X]] — [rationale]\n…`
   - For `fact`: `Information learned: when in [situation], [subject] [predicate] [object].\n\nRelated to:\n- [[X]] — [rationale]\n…`
4. **MOC body is** the framing paragraph(s) only — no constituent list, no bulleted members (Obsidian backlinks carry membership and per-link rationale).
5. **No H1 title in body** — filename serves as the display name.
6. **Source attribution lives in frontmatter only** — no body source line.
7. **The Apply step invokes `engram promote {feedback|fact|moc}`** instead of constructing markdown directly. The skill author specifies `--target` and `--relation` for ID assignment; the binary handles locking.
8. **Discard fleetings via the `--delete-fleeting` flag**, not a separate filesystem step.

- [ ] **Step 1: Invoke the writing-skills skill**

Use the `Skill` tool: `superpowers:writing-skills` with arguments describing the edit goal.

- [ ] **Step 2: Follow that skill's TDD process**

The writing-skills skill enforces RED→GREEN→REFACTOR for skill edits. Follow it. Specifically:
- Write a baseline behavioral test against the current skill (the test should pass against current behavior or fail against the change you want — the skill tells you which).
- Update the skill content per the eight bullets above.
- Verify the behavioral change.
- Run pressure tests per writing-skills guidance.

- [ ] **Step 3: Verify the skill file passes the writing-skills checks**

The writing-skills skill exits with the verification step. Mark this task complete only when that exits clean.

- [ ] **Step 4: Commit**

```bash
cd ~/.claude
git add skills/promoting-to-permanent-notes/SKILL.md
git commit -m "$(cat <<'EOF'
feat(skills): structured frontmatter + body templates for permanent notes

Adopt feedback/fact/moc types with YAML frontmatter for query/auto-surface.
Bodies use formulaic restatement lines (feedback/fact) or framing prose (moc).
MOCs drop the constituent list — backlinks carry membership and rationale.
Apply step invokes engram promote with --target/--relation/--delete-fleeting.

AI-Used: [claude]
EOF
)"
```

(Note: the user's `~/.claude` may or may not be a git repo. If not, skip the commit and just save the file.)

---

## Self-Review Checklist

Run this after the plan is fully executed.

- [ ] Every task in this plan has a passing test.
- [ ] `targ check-full` is clean — zero warnings.
- [ ] The smoke test in Task 11 passes against a fresh temp vault.
- [ ] Concurrent invocation test: run two `engram promote` invocations in parallel against the same vault; verify both succeed with distinct Luhmann IDs.
- [ ] The skill file in Task 12 produces the expected output format for an example feedback, fact, and MOC promotion.
- [ ] No placeholder TODO comments remain in any file.
- [ ] The `1.2026-05-07.subagent-driven-recovery.md` permanent already has the new format from this conversation — confirm `engram promote` would have produced an equivalent file.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-09-engram-promote.md`. Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task with two-stage review between tasks. Best for this plan: 12 tasks, mostly independent, with a clear TDD cycle.
2. **Inline Execution** — execute tasks in this session via `superpowers:executing-plans`, batched with checkpoints.

Which approach?
