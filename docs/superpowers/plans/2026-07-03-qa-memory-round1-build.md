# Q&A Memory Round 1 — Capture Build

**Date:** 2026-07-03
**Decision source:** docs/design/2026-07-03-qa-memory-proposals.md (Joe, 2026-07-03), Joe green-lit 2026-07-03.
**Precedent plan:** docs/superpowers/plans/2026-07-03-vocab-lifecycle-o2-build.md (format + safety)
**Status:** ready for implementation

---

## Authority chain

1. `docs/design/2026-07-03-qa-memory-proposals.md` — decided design: D5′ asymmetric participation (A-notes compete; Q-notes excluded), round-1 scope, names `contributors` + `engram learn qa`, Arm V inside round 1.
2. `docs/superpowers/plans/2026-07-03-qa-memory-exploration.md` — Dim A templates (split Q/A notes, `qa.<date>.<slug>.q.md` / `.a.md`, machine body lines), Fact 4 (four seam points verified), Fact 3 (flag surface).
3. `docs/superpowers/plans/2026-07-03-vocab-lifecycle-o2-build.md` — format precedent: headless writing-skills TDD blocks, worktree rules, copy-vault guards.

Decisions D1–D5′ are settled. No task relitigates them.

---

## Global Constraints

All tasks must satisfy these non-negotiables before any task is marked done:

- **TDD red/green/refactor** per task: failing test first, then code, then refactor.
- **Test stack:** imptest (impgen mocks) + rapid (properties) + gomega assertions.
- **Nilaway/gomega:** after `g.Expect(err).NotTo(HaveOccurred())`, add `if err != nil { return }` before accessing values. Never use `err.Error()` — use `g.Expect(err).To(MatchError(...))`.
- **Build commands:** `targ test` / `targ check-full` only. Never `go test` or `go vet` directly. Install binary: `go install ./cmd/engram`.
- **DI everywhere:** no `os.*` / I/O in `internal/` logic bodies. Every I/O dep injected; nil deps are no-ops.
- **Named constants, no magic numbers.** `qaRound2MinPairs = 20`, not bare `20`.
- **Descriptive names:** `questionText`, `answerBody`, `contributorBasenames` — not `q`, `a`, `cs`.
- **Errors wrapped:** `fmt.Errorf("learn qa: writing Q note: %w", err)`.
- **`t.Parallel()` on every test and subtest.** Each subtest owns its own fixture.
- **Line length under 120 chars.**
- **Commits per task** with trailer `AI-Used: [claude]` (NEVER Co-Authored-By).
- **Live vault untouched** except by the shipped normal flow. Task 11 (validation) uses a copy vault — `set -u` + explicit `LIVE_VAULT` + `COPY_VAULT=$WORK_DIR/qa-r1-validation-vault` guard (Task-11 pattern from o2-build).
- **Skill edits cannot be marked complete** without RED and GREEN headless evidence from fresh `claude -p` processes (one per arm). Subagents are forbidden as skill-TDD TEST ARMS — a Task-tool subagent inherits session context and will act on the edited text even before the edit, poisoning the control; each RED/GREEN arm is a fresh headless `claude -p` process. This prohibition covers the arms ONLY: the `superpowers:writing-skills` skill named in Tasks 7–9 is the executor's DISCIPLINE SOURCE (process instructions), not an agent launcher — no contradiction.
- **D5′ is settled.** A-notes compete in the main set. Q-notes are excluded at all four seam points. No task relitigates this.
- **Arm V eval:** copy-vault only; no absolute paths to the real repo in eval payloads (note 160 — measured).

---

## Interfaces

All new and modified types, function signatures, and constants across the build, collected for cross-task reference.

### New exported constants: `internal/embed/hash.go` (Task 1)

```go
// ContributorsBodyMarker prefixes the machine-written `Contributors: [[…]], …`
// body line on QA answer notes. Excluded from BodyText/ContentHash so a
// contributors-only write leaves the body vector and hash unchanged.
ContributorsBodyMarker = "Contributors:"

// AnsweredByBodyMarker prefixes the machine-written `Answered by: [[…]]` body
// line on QA question notes. Same exclusion rationale.
AnsweredByBodyMarker = "Answered by:"

// AnswersBodyMarker prefixes the machine-written `Answers: [[…]]` body line on
// QA answer notes. Same exclusion rationale.
AnswersBodyMarker = "Answers:"
```

### New unexported constants: `internal/cli/qa.go` (new file, Task 3)

```go
// QA note type strings.
const (
    typeQAQuestion = "qa-question"
    typeQAAnswer   = "qa-answer"
)

// QA filename conventions.
const (
    qaNotePrefix      = "qa."
    qaQuestionSuffix  = ".q.md"
    qaAnswerSuffix    = ".a.md"
)

// Local aliases for machine-line markers (aliased from embed package to keep
// writer + exclusion in sync — the vocab/supersedes precedent).
const (
    contributorsBodyMarker = embed.ContributorsBodyMarker
    answeredByBodyMarker   = embed.AnsweredByBodyMarker
    answersBodyMarker      = embed.AnswersBodyMarker
)

// qa stats gate threshold.
const qaRound2MinPairs = 20

// Sentinel errors for engram learn qa validation.
var (
    errQAQuestionRequired      = errors.New("learn qa: --question is required")
    errQAAnswerSourceRequired  = errors.New("learn qa: exactly one of --answer or --answer-file is required")
    errQAContributorNotFound   = errors.New("learn qa: contributor not found in vault")
    errQACertaintyInvalid      = errors.New("learn qa: --certainty must be high, medium, or low")
)
```

### New type: `LearnQAArgs` (Task 3, `internal/cli/qa.go`)

```go
// LearnQAArgs holds parsed flags for the engram learn qa subcommand.
type LearnQAArgs struct {
    Vault        string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll
    Slug         string   `targ:"flag,name=slug,required,desc=kebab-case tag shared by the Q and A filenames (required)"`
    Question     string   `targ:"flag,name=question,desc=verbatim question text"`
    Answer       string   `targ:"flag,name=answer,desc=inline answer body (mutually exclusive with --answer-file)"`
    AnswerFile   string   `targ:"flag,name=answer-file,desc=path to a file whose content is the answer body"`
    Contributors []string `targ:"flag,name=contributors,desc=full note basenames (no .md) that contributed (repeatable; validated against vault)"`
    Certainty    string   `targ:"flag,name=certainty,desc=high|medium|low (default medium)"`
    Source       string   `targ:"flag,name=source,required,desc=provenance string for the source field (required)"`
}
```

### New type: `LearnQADeps` (Task 3, `internal/cli/qa.go`)

```go
// LearnQADeps holds injected dependencies for RunLearnQA.
// All fields are required except Embedder/WriteSidecar/LogWarning (embed pipeline)
// and LoadTermVectors/ReadSidecar/WriteNote/ListMD (vocab pipeline) — same
// optional-with-no-op contract as LearnDeps.
type LearnQADeps struct {
    Now             func() time.Time
    Getenv          func(string) string
    StatDir         func(string) error
    InitVault       func(string) error
    ListIDs         func(vault string) ([]string, error)
    ListMDFilenames func(vault string) ([]string, error) // full .md names for contributor validation + trigger
    Lock            func(vault string) (release func(), err error)
    WriteNew        func(path string, data []byte) error
    RemoveFile      func(path string) error              // Q-note cleanup on A-write failure
    ReadFile        func(path string) ([]byte, error)    // reading --answer-file
    Embedder        embed.Embedder
    WriteSidecar    func(path string, data []byte) error
    LogWarning      func(format string, args ...any)
    // Vocab assignment — optional; all four must be non-nil to activate.
    LoadTermVectors func(vault string) ([]TermWithVector, error)
    ReadSidecar     func(path string) ([]byte, error)
    WriteNote       func(path string, data []byte) error
}
```

### New functions: `internal/cli/qa.go` (Tasks 3–5)

```go
// RunLearnQA implements the engram learn qa subcommand.
func RunLearnQA(ctx context.Context, args LearnQAArgs, deps LearnQADeps, stdout io.Writer) error

// isQAQuestionKind reports whether a note's frontmatter type is qa-question.
func isQAQuestionKind(content string) bool

// isQAAnswerKind reports whether a note's frontmatter type is qa-answer.
func isQAAnswerKind(content string) bool

// isQAQuestionFilename reports whether a filename is a QA question note
// (prefix "qa." AND suffix ".q.md").
func isQAQuestionFilename(name string) bool

// isQAAnswerFilename reports whether a filename is a QA answer note
// (prefix "qa." AND suffix ".a.md").
func isQAAnswerFilename(name string) bool

// isQueryExcludedKind reports whether a note should be excluded from the
// query pipeline's main matched set. Excludes vocab kinds and qa-question.
// qa-answer COMPETES in the main set (D5′).
func isQueryExcludedKind(content string) bool

// countQAPairs counts vault files where both the .q.md and matching .a.md exist.
// Pure read-time scan; no new state.
func countQAPairs(names []string) int

// qaQuestionPath returns the vault-relative filename for a QA question note.
func qaQuestionPath(vault, slug string, when time.Time) string

// qaAnswerPath returns the vault-relative filename for a QA answer note.
func qaAnswerPath(vault, slug string, when time.Time) string

// renderQAQuestionNote assembles the full content of a QA question note.
func renderQAQuestionNote(questionText, slug, source string, when time.Time) string

// renderQAAnswerNote assembles the full content of a QA answer note.
func renderQAAnswerNote(answerBody, slug, source, certainty string,
    contributors []string, when time.Time) string

// validateContributors returns an error if any contributor basename is not
// present in the vault (same bar as supersedes targets: ERROR on unresolvable).
func validateContributors(contributors []string, vaultMDNames []string) error
```

### Modified: `internal/cli/vocab_commands.go` — scan loops + stats (Tasks 2, 6)

```go
// isQAQuestionFilename — new (Task 2; also lives in qa.go — the two share the package)
// See Interfaces.

// collectVaultStats change (Task 2): after isVocabTermFilename skip, add:
//   if isQAQuestionFilename(name) { continue }

// assignTermsToAllNotes change (Task 2): after isVocabKindFilename skip, add:
//   if isQAQuestionFilename(name) { continue }

// printStatsReport — add qaPairs int parameter (Task 6)
func printStatsReport(
    stdout        io.Writer,
    termNames     []string,
    memberCounts  map[string]int,
    totalNotes, untaggedCount int,
    vocabVersion  string,
    refitPending  bool,
    refitReason   string,
    qaPairs       int, // NEW
)
```

### Modified: `internal/cli/vocab_trigger.go` — scan loops (Task 2)

```go
// scanNonVocabNotes change (Task 2): extend skip from
//   if isVocabKindFilename(name) { continue }
// to:
//   if isVocabKindFilename(name) || isQAQuestionFilename(name) { continue }

// countNonVocabNoteFiles change (Task 2): same extension.
```

### Modified: `internal/cli/query.go` + `query_nominations.go` — four seam points (Task 2)

```go
// Replace all four isVocabKind call sites with isQueryExcludedKind:
// query.go:435  — pre-clustering filter (Point A)
// query.go:846  — floor/cap guard (Point B)
// query.go:1084 — pre-clustering filter, second occurrence (Point A)
// query_nominations.go:95  — nomination gate (Point C)
// query_nominations.go:337 — TermIndex builder gate (Point D)
```

---

## Tasks

### Task 1 — New body-line marker constants in `hash.go`

**File:** `internal/embed/hash.go`

**What:** Add three exported constants for the new machine-written QA body lines. Extend `stripMachineLines` to strip them.

**RED test** (`internal/embed/hash_test.go` — file exists; add subtests):

```go
func TestStripMachineLines_QAMarkersRemoved(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name string
        in   string
        want string
    }{
        {
            name: "contributors_line_stripped",
            in:   "Body line.\n\nContributors: [[100.note]], [[101.note]]\n",
            want: "Body line.\n",
        },
        {
            name: "answered_by_line_stripped",
            in:   "What is the question?\n\nAnswered by: [[qa.2026-07-03.slug.a]]\n",
            want: "What is the question?\n",
        },
        {
            name: "answers_line_stripped",
            in:   "The answer body.\n\nAnswers: [[qa.2026-07-03.slug.q]]\n",
            want: "The answer body.\n",
        },
        {
            name: "all_three_stripped_together",
            in:   "Body.\n\nAnswered by: [[qa.2026-07-03.slug.a]]\nAnswers: [[qa.2026-07-03.slug.q]]\nContributors: [[100.note]]\n",
            want: "Body.\n",
        },
        {
            name: "no_markers_unchanged",
            in:   "Body without machine lines.\n",
            want: "Body without machine lines.\n",
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            g := NewWithT(t)
            // BodyText strips frontmatter first; wrap in frontmatter to exercise BodyText.
            raw := []byte("---\ntype: qa-answer\n---\n\n" + tc.in)
            got := string(BodyText(raw))
            g.Expect(got).To(Equal(tc.want))
        })
    }
}

func TestContributorsBodyMarker_IsExported(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    g.Expect(embed.ContributorsBodyMarker).To(Equal("Contributors:"))
    g.Expect(embed.AnsweredByBodyMarker).To(Equal("Answered by:"))
    g.Expect(embed.AnswersBodyMarker).To(Equal("Answers:"))
}
```

**GREEN:** Add the three exported constants to `hash.go`. Extend `stripMachineLines` to also skip lines with `ContributorsBodyMarker`, `AnsweredByBodyMarker`, and `AnswersBodyMarker` prefixes (same `bytes.HasPrefix` pattern as the existing two markers):

```go
if bytes.HasPrefix(trimmed, []byte(VocabBodyMarker)) ||
    bytes.HasPrefix(trimmed, []byte(SupersedesBodyMarker)) ||
    bytes.HasPrefix(trimmed, []byte(ContributorsBodyMarker)) ||
    bytes.HasPrefix(trimmed, []byte(AnsweredByBodyMarker)) ||
    bytes.HasPrefix(trimmed, []byte(AnswersBodyMarker)) {
    removed = true
    continue
}
```

**Verify:** `targ test` passes. `targ check-full` clean.

**Commit:** `feat(embed): QA machine-line markers + stripMachineLines extension`

---

### Task 2 — D5′ exclusion: QA filename helpers + query pipeline + scan loops

**Files:** `internal/cli/qa.go` (new), `internal/cli/query.go`, `internal/cli/query_nominations.go`, `internal/cli/vocab_commands.go`, `internal/cli/vocab_trigger.go`

**What:** Add QA type constants and filename helpers; extend the four query-pipeline seam points from `isVocabKind` to `isQueryExcludedKind`; extend scan loops to exclude Q-question filenames; extend `collectVaultStats` and `assignTermsToAllNotes`.

#### A. New file `internal/cli/qa.go`

Start with just the constants, filename helpers, and `isQueryExcludedKind`. (RunLearnQA comes in Task 3–5.)

```go
package cli

import (
    "errors"
    "strings"

    "github.com/toejough/engram/internal/embed"
)

const (
    typeQAQuestion = "qa-question"
    typeQAAnswer   = "qa-answer"

    qaNotePrefix     = "qa."
    qaQuestionSuffix = ".q.md"
    qaAnswerSuffix   = ".a.md"

    contributorsBodyMarker = embed.ContributorsBodyMarker
    answeredByBodyMarker   = embed.AnsweredByBodyMarker
    answersBodyMarker      = embed.AnswersBodyMarker

    qaRound2MinPairs = 20
)

var (
    errQAQuestionRequired     = errors.New("learn qa: --question is required")
    errQAAnswerSourceRequired = errors.New("learn qa: exactly one of --answer or --answer-file is required")
    errQAContributorNotFound  = errors.New("learn qa: contributor not found in vault")
    errQACertaintyInvalid     = errors.New("learn qa: --certainty must be high, medium, or low")
)

// isQAQuestionKind reports whether the note's frontmatter type is qa-question.
func isQAQuestionKind(content string) bool {
    return kindFromContent(content) == typeQAQuestion
}

// isQAAnswerKind reports whether the note's frontmatter type is qa-answer.
func isQAAnswerKind(content string) bool {
    return kindFromContent(content) == typeQAAnswer
}

// isQAQuestionFilename reports whether a filename is a QA question note.
func isQAQuestionFilename(name string) bool {
    return strings.HasPrefix(name, qaNotePrefix) && strings.HasSuffix(name, qaQuestionSuffix)
}

// isQAAnswerFilename reports whether a filename is a QA answer note.
func isQAAnswerFilename(name string) bool {
    return strings.HasPrefix(name, qaNotePrefix) && strings.HasSuffix(name, qaAnswerSuffix)
}

// isQueryExcludedKind reports whether a note should be excluded from the
// query pipeline's main matched set. Excludes vocab kinds AND qa-question.
// qa-answer COMPETES in the main set (D5′ — A notes are synthesis notes).
func isQueryExcludedKind(content string) bool {
    return isVocabKind(content) || isQAQuestionKind(content)
}

// countQAPairs counts vault files where both the .q.md and matching .a.md exist.
func countQAPairs(names []string) int {
    nameSet := make(map[string]struct{}, len(names))
    for _, name := range names {
        nameSet[name] = struct{}{}
    }

    count := 0
    for _, name := range names {
        if !isQAQuestionFilename(name) {
            continue
        }
        // Derive expected A filename: replace .q.md suffix with .a.md.
        base := strings.TrimSuffix(name, qaQuestionSuffix)
        aName := base + qaAnswerSuffix
        if _, ok := nameSet[aName]; ok {
            count++
        }
    }

    return count
}
```

**RED tests** (`internal/cli/qa_test.go`, new file):

```go
package cli_test

import (
    "testing"

    . "github.com/onsi/gomega"

    "github.com/toejough/engram/internal/cli"
)

func TestIsQAQuestionFilename(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name string
        in   string
        want bool
    }{
        {"question_file", "qa.2026-07-03.my-slug.q.md", true},
        {"answer_file", "qa.2026-07-03.my-slug.a.md", false},
        {"vocab_file", "vocab.agentic-recall-triggers.md", false},
        {"regular_fact", "100.2026-07-01.some-fact.md", false},
        {"bare_qa_prefix", "qa.md", false},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            g := NewWithT(t)
            g.Expect(cli.ExportIsQAQuestionFilename(tc.in)).To(Equal(tc.want))
        })
    }
}

func TestIsQueryExcludedKind(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name    string
        content string
        want    bool
    }{
        {"vocab_excluded", "---\ntype: vocab\n---\n", true},
        {"vocab_index_excluded", "---\ntype: vocab-index\n---\n", true},
        {"qa_question_excluded", "---\ntype: qa-question\n---\n", true},
        {"qa_answer_competes", "---\ntype: qa-answer\n---\n", false},
        {"fact_competes", "---\ntype: fact\n---\n", false},
        {"feedback_competes", "---\ntype: feedback\n---\n", false},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            g := NewWithT(t)
            g.Expect(cli.ExportIsQueryExcludedKind(tc.content)).To(Equal(tc.want))
        })
    }
}

func TestCountQAPairs(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    names := []string{
        "qa.2026-07-03.first.q.md",
        "qa.2026-07-03.first.a.md",
        "qa.2026-07-03.second.q.md",   // no matching .a.md — orphan Q
        "100.2026-07-01.some-fact.md",
        "vocab.agentic-recall-triggers.md",
    }
    g.Expect(cli.ExportCountQAPairs(names)).To(Equal(1))
}
```

Add to `export_test.go`:

```go
var ExportIsQAQuestionFilename  = isQAQuestionFilename
var ExportIsQueryExcludedKind   = isQueryExcludedKind
var ExportCountQAPairs          = countQAPairs
```

#### B. Query pipeline — replace `isVocabKind` with `isQueryExcludedKind`

Replace ALL FIVE occurrences (4 documented seam points, but point A appears at two line numbers):

- `internal/cli/query.go:435`: `if !item.isChunk && isVocabKind(item.note.content) {` → `isQueryExcludedKind`
- `internal/cli/query.go:846`: `return !item.isChunk && item.baseScore >= matchRelevanceFloor && !isVocabKind(item.note.content)` → `!isQueryExcludedKind`
- `internal/cli/query.go:1084`: `if !item.isChunk && isVocabKind(item.note.content) {` → `isQueryExcludedKind`
- `internal/cli/query_nominations.go:95`: `if isVocabKind(entry.Content) {` → `isQueryExcludedKind`
- `internal/cli/query_nominations.go:337`: `if !isVocabKind(content) && len(meta.Vocab) > 0 {` → `!isQueryExcludedKind`

**Re-grep before editing** (mandatory — line numbers may drift): `grep -n "isVocabKind" internal/cli/query.go internal/cli/query_nominations.go`

#### C. Scan loop exclusions

**`internal/cli/vocab_trigger.go` — `scanNonVocabNotes` (line ~198):**
```go
// Change:
if isVocabKindFilename(name) {
    continue
}
// To:
if isVocabKindFilename(name) || isQAQuestionFilename(name) {
    continue
}
```

**`internal/cli/vocab_trigger.go` — `countNonVocabNoteFiles` (line ~136):**
```go
// Change:
if !isVocabKindFilename(name) {
    count++
}
// To:
if !isVocabKindFilename(name) && !isQAQuestionFilename(name) {
    count++
}
```

**`internal/cli/vocab_commands.go` — `collectVaultStats` (after `isVocabTermFilename` skip, line ~668):**
```go
// Add after the isVocabTermFilename block:
if isQAQuestionFilename(name) {
    continue
}
```

**`internal/cli/vocab_commands.go` — `assignTermsToAllNotes` (after `isVocabKindFilename` skip, line ~445):**
```go
// Add after the isVocabKindFilename block:
if isQAQuestionFilename(name) {
    continue
}
```

**RED tests for scan loop exclusions** (add to `qa_test.go` or new `vocab_commands_exclusion_test.go`):

```go
func TestCollectVaultStats_QAQuestionExcluded_QAAnswerCounted(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    // qa.*.q.md file should NOT appear in totalNotes; qa.*.a.md should.
    names := []string{
        "qa.2026-07-03.slug.q.md",  // Q note — excluded
        "qa.2026-07-03.slug.a.md",  // A note — counted
        "vocab.agentic-recall-triggers.md",
        "vocab.index.md",
    }

    qaAContent := "---\ntype: qa-answer\ndate: \"2026-07-03\"\nvocab: [agentic-recall-triggers]\n---\n\nAnswer body.\n"

    deps := cli.VocabStatsDeps{
        ListMD: func(string) ([]string, error) { return names, nil },
        ReadFile: func(path string) ([]byte, error) {
            if strings.HasSuffix(path, "slug.a.md") {
                return []byte(qaAContent), nil
            }
            return nil, os.ErrNotExist
        },
    }

    _, _, totalNotes, _ := cli.ExportCollectVaultStats(names, deps, "/vault")
    g.Expect(totalNotes).To(Equal(1), "only A-note should count; Q-note excluded")
}

func TestScanNonVocabNotes_QAQuestionFilenameSkipped(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    names := []string{
        "qa.2026-07-03.slug.q.md",   // should be skipped
        "qa.2026-07-03.slug.a.md",   // should be visited
        "100.2026-07-01.note.md",    // should be visited
        "vocab.some-term.md",        // should be skipped
    }

    var visited []string
    cli.ExportScanNonVocabNotes("/vault", names,
        func(string) ([]byte, error) { return []byte("---\ntype: fact\n---\n"), nil },
        func(name string, _ []byte, _ error) { visited = append(visited, name) },
    )

    g.Expect(visited).To(ConsistOf("qa.2026-07-03.slug.a.md", "100.2026-07-01.note.md"))
}
```

Add to `export_test.go`:

```go
func ExportCollectVaultStats(names []string, deps VocabStatsDeps, vault string) ([]string, map[string]int, int, int) {
    return collectVaultStats(names, deps, vault)
}
var ExportScanNonVocabNotes = scanNonVocabNotes
```

**GREEN:** Apply all changes above. Re-grep line numbers before editing query.go and query_nominations.go.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(qa): D5' exclusion — isQueryExcludedKind + scan-loop Q-file exclusions`

---

### Task 3 — QA note renderers + validators (pure, no I/O)

**File:** `internal/cli/qa.go` (continue)

**What:** Add pure functions for rendering Q/A frontmatter + body and for validating args.

#### Frontmatter and body shapes

Q note (`qa.<date>.<slug>.q.md`):
```yaml
---
type: qa-question
date: "YYYY-MM-DD"
answered_by: qa.<date>.<slug>.a
source: "<source>"
---

<verbatim question text>

Answered by: [[qa.<date>.<slug>.a]]
```

A note (`qa.<date>.<slug>.a.md`):
```yaml
---
type: qa-answer
date: "YYYY-MM-DD"
answers: qa.<date>.<slug>.q
certainty: high|medium|low
contributors: [<full-basename-1>, <full-basename-2>]
source: "<source>"
vocab: [...]   # auto-assigned; not in the renderer
---

<answer body>

Answers: [[qa.<date>.<slug>.q]]
Contributors: [[<full-basename-1>]], [[<full-basename-2>]]
```

Note: `vocab:` is NOT in the renderer output — it is added later by `applyVocabAssignmentAfterLearn`, same as all other notes.

The frontmatter for both notes uses `marshalFrontmatter` (already in package, used by learn.go).

#### New YAML doc types for frontmatter rendering

```go
// (Plan note, Gate A: the templates here include the `source:` field the exploration doc's
// Dim A sketches omitted — `--source` is REQUIRED on every engram learn call (exploration
// Fact 3); the full shape HERE is the specification.)
// qaQuestionFrontmatterDoc is the YAML shape of a QA question note's frontmatter.
type qaQuestionFrontmatterDoc struct {
    Type       string `yaml:"type"`
    Date       string `yaml:"date"`
    AnsweredBy string `yaml:"answered_by"`
    Source     string `yaml:"source"`
}

// qaAnswerFrontmatterDoc is the YAML shape of a QA answer note's frontmatter.
type qaAnswerFrontmatterDoc struct {
    Type         string   `yaml:"type"`
    Date         string   `yaml:"date"`
    Answers      string   `yaml:"answers"`
    Certainty    string   `yaml:"certainty"`
    Contributors []string `yaml:"contributors,omitempty"`
    Source       string   `yaml:"source"`
}
```

#### Path helpers

```go
// qaSlug returns the shared date+slug prefix: "qa.<YYYY-MM-DD>.<slug>".
func qaSlug(slug string, when time.Time) string {
    return qaNotePrefix + when.Format(dateFormat) + "." + slug
}

// qaQuestionPath returns the full vault path for a QA question note.
func qaQuestionPath(vault, slug string, when time.Time) string {
    return filepath.Join(vault, qaSlug(slug, when)+qaQuestionSuffix)
}

// qaAnswerPath returns the full vault path for a QA answer note.
func qaAnswerPath(vault, slug string, when time.Time) string {
    return filepath.Join(vault, qaSlug(slug, when)+qaAnswerSuffix)
}
```

#### Body renderers

```go
// renderQAQuestionNote assembles the full content of a QA question note.
// The machine `Answered by:` line is appended after the question body.
func renderQAQuestionNote(questionText, slug, source string, when time.Time) string {
    sharedSlug := qaSlug(slug, when)
    // Full basename = filename minus .md (G0 constraint): the paired answer note.
    aBasename := sharedSlug + ".a"

    frontmatter := marshalFrontmatter(qaQuestionFrontmatterDoc{
        Type:       typeQAQuestion,
        Date:       when.Format(dateFormat),
        AnsweredBy: aBasename,
        Source:     source,
    })

    body := strings.TrimRight(questionText, "\n") + "\n"
    body += "\n" + answeredByBodyMarker + " [[" + aBasename + "]]\n"

    return frontmatter + "\n" + body
}

// renderQAAnswerNote assembles the full content of a QA answer note.
// The machine `Answers:` and `Contributors:` lines are appended.
func renderQAAnswerNote(answerBody, slug, source, certainty string,
    contributors []string, when time.Time) string {
    sharedSlug := qaSlug(slug, when)
    // Full basename of the paired question note (G0 constraint).
    qBasename := sharedSlug + ".q"

    frontmatter := marshalFrontmatter(qaAnswerFrontmatterDoc{
        Type:         typeQAAnswer,
        Date:         when.Format(dateFormat),
        Answers:      qBasename,
        Certainty:    certainty,
        Contributors: contributors,
        Source:       source,
    })

    body := strings.TrimRight(answerBody, "\n") + "\n"
    body += "\n" + answersBodyMarker + " [[" + qBasename + "]]\n"

    if len(contributors) > 0 {
        parts := make([]string, len(contributors))
        for i, c := range contributors {
            parts[i] = "[[" + c + "]]"
        }
        body += contributorsBodyMarker + " " + strings.Join(parts, ", ") + "\n"
    }

    return frontmatter + "\n" + body
}
```

#### Validators

```go
// validateLearnQAArgs validates all LearnQAArgs fields before any I/O.
func validateLearnQAArgs(args LearnQAArgs) error {
    if err := validateSlug(args.Slug); err != nil {
        return fmt.Errorf("learn qa: %w", err)
    }
    if strings.TrimSpace(args.Question) == "" {
        return errQAQuestionRequired
    }
    if (args.Answer == "" && args.AnswerFile == "") || (args.Answer != "" && args.AnswerFile != "") {
        return errQAAnswerSourceRequired
    }
    if args.Source == "" {
        return errors.New("learn qa: --source is required")
    }
    certainty := args.Certainty
    if certainty == "" {
        certainty = "medium"
    }
    switch certainty {
    case "high", "medium", "low":
    default:
        return fmt.Errorf("%w: got %q", errQACertaintyInvalid, certainty)
    }
    return nil
}

// validateContributors checks that each full basename exists in the vault.
// vaultMDNames is the list of full .md filenames (from ListMDFilenames).
func validateContributors(contributors, vaultMDNames []string) error {
    nameSet := make(map[string]struct{}, len(vaultMDNames))
    for _, name := range vaultMDNames {
        nameSet[strings.TrimSuffix(name, ".md")] = struct{}{}
    }
    for _, contributor := range contributors {
        if _, ok := nameSet[contributor]; !ok {
            return fmt.Errorf("%w: %q", errQAContributorNotFound, contributor)
        }
    }
    return nil
}
```

**RED tests** (`internal/cli/qa_test.go`, add):

```go
func TestRenderQAQuestionNote_ContainsExpectedParts(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    when := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    got := cli.ExportRenderQAQuestionNote("What is the question?", "my-slug", "session 2026-07-03", when)

    g.Expect(got).To(ContainSubstring("type: qa-question"))
    g.Expect(got).To(ContainSubstring("answered_by: qa.2026-07-03.my-slug.a"))
    g.Expect(got).To(ContainSubstring("What is the question?"))
    g.Expect(got).To(ContainSubstring("Answered by: [[qa.2026-07-03.my-slug.a]]"))
    // Answered by: line must be in the BODY, not frontmatter
    body := strings.SplitN(got, "---\n\n", 2)
    g.Expect(len(body)).To(Equal(2))
    g.Expect(body[1]).To(ContainSubstring("Answered by:"))
}

func TestRenderQAAnswerNote_ContainsMachineLines(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    when := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    contributors := []string{"100.2026-06-01.note", "101.2026-06-02.other-note"}
    got := cli.ExportRenderQAAnswerNote("The answer.", "my-slug", "session 2026-07-03", "medium", contributors, when)

    g.Expect(got).To(ContainSubstring("type: qa-answer"))
    g.Expect(got).To(ContainSubstring("answers: qa.2026-07-03.my-slug.q"))
    g.Expect(got).To(ContainSubstring("certainty: medium"))
    g.Expect(got).To(ContainSubstring("The answer."))
    g.Expect(got).To(ContainSubstring("Answers: [[qa.2026-07-03.my-slug.q]]"))
    g.Expect(got).To(ContainSubstring("Contributors: [[100.2026-06-01.note]], [[101.2026-06-02.other-note]]"))
}

func TestRenderQAAnswerNote_NoContributors_NoContributorsLine(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    when := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    got := cli.ExportRenderQAAnswerNote("Answer.", "slug", "source", "high", nil, when)
    g.Expect(got).NotTo(ContainSubstring("Contributors:"))
}

func TestValidateLearnQAArgs_MissingQuestion_Error(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    args := cli.LearnQAArgs{Slug: "slug", Answer: "body", Source: "src"}
    g.Expect(cli.ExportValidateLearnQAArgs(args)).To(MatchError(cli.ErrQAQuestionRequired))
}

func TestValidateLearnQAArgs_BothAnswerAndFile_Error(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    args := cli.LearnQAArgs{Slug: "slug", Question: "Q?", Answer: "body", AnswerFile: "/tmp/f.md", Source: "src"}
    g.Expect(cli.ExportValidateLearnQAArgs(args)).To(MatchError(cli.ErrQAAnswerSourceRequired))
}

func TestValidateContributors_UnknownBasename_Error(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    vaultNames := []string{"100.2026-01-01.note.md"}
    err := cli.ExportValidateContributors([]string{"999.2026-01-01.ghost"}, vaultNames)
    g.Expect(err).To(MatchError(cli.ErrQAContributorNotFound))
}

func TestValidateContributors_KnownBasename_OK(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    vaultNames := []string{"100.2026-01-01.note.md"}
    g.Expect(cli.ExportValidateContributors([]string{"100.2026-01-01.note"}, vaultNames)).To(Succeed())
}
```

Add to `export_test.go`:

```go
var ExportRenderQAQuestionNote  = renderQAQuestionNote
var ExportRenderQAAnswerNote    = renderQAAnswerNote
var ExportValidateLearnQAArgs   = validateLearnQAArgs
var ExportValidateContributors  = validateContributors

var (
    ErrQAQuestionRequired     = errQAQuestionRequired
    ErrQAAnswerSourceRequired = errQAAnswerSourceRequired
    ErrQAContributorNotFound  = errQAContributorNotFound
)
```

**GREEN:** Write all pure functions above into `qa.go`.

Note on `renderQAQuestionNote` / `renderQAAnswerNote` implementation: the basename derivation is:
- `sharedSlug = "qa." + when.Format("2006-01-02") + "." + slug`
- Q basename (no .md): `sharedSlug + ".q"`
- A basename (no .md): `sharedSlug + ".a"`
- Q filename: `sharedSlug + ".q.md"` (for path helpers)
- A filename: `sharedSlug + ".a.md"` (for path helpers)

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(qa): QA note renderers, validators, path helpers (pure)`

---

### Task 4 — `RunLearnQA` — writes Q then A atomically-ish

**File:** `internal/cli/qa.go` (continue)

**What:** Implement `RunLearnQA` and `newOsLearnQADeps`. Writes Q note first, then A note. On A-write failure: attempt Q note removal (best-effort); wrap and return the A-write error with an explicit orphan warning if removal also fails.

```go
// RunLearnQA implements the engram learn qa subcommand.
// Writes Q then A atomically-ish: on A-write failure, removes Q (best-effort)
// and returns a descriptive error. The function calls the existing embed and
// vocab assignment pipeline for both notes, in order.
func RunLearnQA(ctx context.Context, args LearnQAArgs, deps LearnQADeps, stdout io.Writer) error {
    if err := validateLearnQAArgs(args); err != nil {
        return err
    }

    certainty := args.Certainty
    if certainty == "" {
        certainty = "medium"
    }

    vault := args.Vault

    // Ensure vault exists (same pattern as runLearn).
    if err := deps.StatDir(vault); err != nil {
        if !errors.Is(err, fs.ErrNotExist) {
            return fmt.Errorf("learn qa: vault %s: %w", vault, err)
        }
        if initErr := deps.InitVault(vault); initErr != nil {
            return fmt.Errorf("learn qa: %w", initErr)
        }
    }

    // Resolve answer body.
    answerBody := args.Answer
    if args.AnswerFile != "" {
        raw, readErr := deps.ReadFile(args.AnswerFile)
        if readErr != nil {
            return fmt.Errorf("learn qa: reading --answer-file %q: %w", args.AnswerFile, readErr)
        }
        answerBody = string(raw)
    }

    // Validate contributors before acquiring the lock.
    mdNames, listErr := deps.ListMDFilenames(vault)
    if listErr != nil {
        return fmt.Errorf("learn qa: listing vault: %w", listErr)
    }
    if err := validateContributors(args.Contributors, mdNames); err != nil {
        return err
    }

    when := deps.Now()
    qPath := qaQuestionPath(vault, args.Slug, when)
    aPath := qaAnswerPath(vault, args.Slug, when)

    qContent := renderQAQuestionNote(args.Question, args.Slug, args.Source, when)
    aContent  := renderQAAnswerNote(answerBody, args.Slug, args.Source, certainty,
        args.Contributors, when)

    // Write under lock (Q then A). Lock prevents Luhmann-ID collisions from
    // concurrent regular learn operations; QA notes use date+slug names, not
    // Luhmann IDs, so we only need to prevent partial-write races.
    release, lockErr := deps.Lock(vault)
    if lockErr != nil {
        return fmt.Errorf("learn qa: acquiring lock: %w", lockErr)
    }
    defer release()

    // Write Q note.
    if err := deps.WriteNew(qPath, []byte(qContent)); err != nil {
        return fmt.Errorf("learn qa: writing Q note: %w", err)
    }

    // Write A note — on failure, remove Q (best-effort) and return error.
    if err := deps.WriteNew(aPath, []byte(aContent)); err != nil {
        removeErr := deps.RemoveFile(qPath)
        if removeErr != nil {
            return fmt.Errorf("learn qa: writing A note: %w (also failed to remove orphan Q note %q: %v)",
                err, qPath, removeErr)
        }
        return fmt.Errorf("learn qa: writing A note (Q note removed): %w", err)
    }

    // Embed-on-write and vocab assignment for Q note (embed only; no vocab on Q).
    autoEmbedNote(ctx, asLearnDepsForEmbed(deps), qPath, qContent)
    // No vocab assignment for Q notes (D5′: Q notes carry no vocab).

    // Embed-on-write and vocab assignment for A note.
    autoEmbedNote(ctx, asLearnDepsForEmbed(deps), aPath, aContent)
    applyVocabAssignmentAfterLearn(asLearnDepsForVocab(deps), vault, aPath, aContent)

    _, _ = fmt.Fprintln(stdout, qPath)
    _, _ = fmt.Fprintln(stdout, aPath)

    return nil
}
```

`asLearnDepsForEmbed` and `asLearnDepsForVocab` are private adapters that project `LearnQADeps` into the subset of `LearnDeps` fields needed by `autoEmbedNote` and `applyVocabAssignmentAfterLearn`:

```go
// asLearnDepsForEmbed builds the minimal LearnDeps needed by autoEmbedNote.
func asLearnDepsForEmbed(d LearnQADeps) LearnDeps {
    return LearnDeps{
        Embedder:     d.Embedder,
        WriteSidecar: d.WriteSidecar,
        LogWarning:   d.LogWarning,
    }
}

// asLearnDepsForVocab builds the minimal LearnDeps needed by applyVocabAssignmentAfterLearn.
func asLearnDepsForVocab(d LearnQADeps) LearnDeps {
    return LearnDeps{
        Now:             d.Now,
        LoadTermVectors: d.LoadTermVectors,
        ReadSidecar:     d.ReadSidecar,
        WriteNote:       d.WriteNote,
        LogWarning:      d.LogWarning,
        ListMD:          d.ListMDFilenames,
    }
}
```

**`newOsLearnQADeps`:**

```go
func newOsLearnQADeps() LearnQADeps {
    osVault := &osVaultFS{}
    vaultFS := &osLearnFS{}

    return LearnQADeps{
        Now:             time.Now,
        Getenv:          os.Getenv,
        StatDir:         vaultFS.StatDir,
        InitVault:       func(path string) error { return initializeVault(vaultFS, path) },
        ListIDs:         vaultFS.ListIDs,
        ListMDFilenames: osVault.ListMD,
        Lock:            vaultFS.Lock,
        WriteNew:        vaultFS.WriteNew,
        RemoveFile:      os.Remove,
        ReadFile:        osVault.ReadFile,
        Embedder:        sharedEmbedder,
        WriteSidecar:    vaultFS.WriteSidecar,
        LogWarning:      logWarningToStderrf,
        LoadTermVectors: func(vault string) ([]TermWithVector, error) {
            return loadAssignmentTermVectors(vault, osVault.ListMD, osVault.ReadFile)
        },
        ReadSidecar: osVault.ReadFile,
        WriteNote: func(path string, data []byte) error {
            return atomicWriteFile(path, data, vocabNotePerm)
        },
    }
}
```

**RED tests** (`internal/cli/qa_test.go`, add):

```go
func TestRunLearnQA_WritesQAndAFiles(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    var written []string
    deps := cli.LearnQADeps{
        Now:             func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
        Getenv:          func(string) string { return "" },
        StatDir:         func(string) error { return nil },
        InitVault:       func(string) error { return nil },
        ListIDs:         func(string) ([]string, error) { return nil, nil },
        ListMDFilenames: func(string) ([]string, error) { return []string{"100.note.md"}, nil },
        Lock:            func(string) (func(), error) { return func() {}, nil },
        WriteNew:        func(path string, _ []byte) error { written = append(written, path); return nil },
        RemoveFile:      func(string) error { return nil },
        ReadFile:        func(string) ([]byte, error) { return nil, nil },
    }

    var buf strings.Builder
    err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
        Slug:     "test-qa",
        Question: "What is X?",
        Answer:   "X is Y.",
        Source:   "test",
    }, deps, &buf)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }
    g.Expect(written).To(HaveLen(2))
    g.Expect(written[0]).To(ContainSubstring(".q.md"))
    g.Expect(written[1]).To(ContainSubstring(".a.md"))
}

func TestRunLearnQA_AWriteFailure_RemovesQAndErrors(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    var removed []string
    var writeCount int
    deps := cli.LearnQADeps{
        Now:             func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
        Getenv:          func(string) string { return "" },
        StatDir:         func(string) error { return nil },
        InitVault:       func(string) error { return nil },
        ListIDs:         func(string) ([]string, error) { return nil, nil },
        ListMDFilenames: func(string) ([]string, error) { return nil, nil },
        Lock:            func(string) (func(), error) { return func() {}, nil },
        WriteNew: func(path string, _ []byte) error {
            writeCount++
            if writeCount == 2 {
                return errors.New("disk full")
            }
            return nil
        },
        RemoveFile: func(path string) error { removed = append(removed, path); return nil },
        ReadFile:   func(string) ([]byte, error) { return nil, nil },
    }

    var buf strings.Builder
    err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
        Slug: "slug", Question: "Q?", Answer: "A.", Source: "src",
    }, deps, &buf)
    g.Expect(err).To(HaveOccurred())
    g.Expect(removed).To(HaveLen(1), "Q note must be removed on A-write failure")
    g.Expect(removed[0]).To(ContainSubstring(".q.md"))
}

func TestRunLearnQA_UnknownContributor_ErrorBeforeWrite(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    var writeCallCount int
    deps := cli.LearnQADeps{
        Now:             func() time.Time { return time.Now() },
        Getenv:          func(string) string { return "" },
        StatDir:         func(string) error { return nil },
        InitVault:       func(string) error { return nil },
        ListIDs:         func(string) ([]string, error) { return nil, nil },
        ListMDFilenames: func(string) ([]string, error) { return []string{"100.note.md"}, nil },
        Lock:            func(string) (func(), error) { return func() {}, nil },
        WriteNew:        func(string, []byte) error { writeCallCount++; return nil },
        RemoveFile:      func(string) error { return nil },
        ReadFile:        func(string) ([]byte, error) { return nil, nil },
    }

    var buf strings.Builder
    err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
        Slug: "slug", Question: "Q?", Answer: "A.", Source: "src",
        Contributors: []string{"999.ghost"},
    }, deps, &buf)
    g.Expect(err).To(MatchError(ContainSubstring("contributor not found")))
    g.Expect(writeCallCount).To(Equal(0), "no writes before validation error")
}
```

Add to `export_test.go`:

```go
// No extra export shims needed: RunLearnQA is already exported.
// LearnQADeps is exported (struct).
```

**GREEN:** Write `RunLearnQA`, `newOsLearnQADeps`, `asLearnDepsForEmbed`, `asLearnDepsForVocab`.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(qa): RunLearnQA — write Q+A pair, embed-on-write, vocab on A only`

---

### Task 5 — Wire `learn qa` in `targets.go`

**File:** `internal/cli/targets.go`

**What:** Add `"qa"` sub-target under the existing `targ.Group("learn", ...)` in `learnUpdateTargets`.

**Change** (inside the `targ.Group("learn", ...)` call, after the `"fact"` entry):

```go
targ.Group("learn",
    targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
        a.Vault = resolveVault(a.Vault, home, os.Getenv)
        errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, stdout))
    }).Name("feedback").Description("Write a feedback note to the vault"),
    targ.Targ(func(ctx context.Context, a LearnFactArgs) {
        a.Vault = resolveVault(a.Vault, home, os.Getenv)
        errHandler(runLearnFromFactArgs(withLog(ctx), a, stdout))
    }).Name("fact").Description("Write a fact note to the vault"),
    targ.Targ(func(ctx context.Context, a LearnQAArgs) {  // NEW
        a.Vault = resolveVault(a.Vault, home, os.Getenv)
        errHandler(RunLearnQA(withLog(ctx), a, newOsLearnQADeps(), stdout))
    }).Name("qa").Description("Write a QA pair (Q+A notes) to the vault"),
),
```

**RED test** (`internal/cli/targets_test.go` — if it exists — or new `targets_qa_test.go`):

```go
func TestLearnQASubcommandExists(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    // Verify the CLI exposes "engram learn qa" by inspecting Targets().
    // A smoke test: Targets must not panic and must include a "learn qa" path.
    var allNames []string
    for _, target := range cli.Targets(
        io.Discard, io.Discard, func(int) {}, debuglog.NewNullLogger(),
    ) {
        collectTargetNames(target, "", &allNames)
    }
    g.Expect(allNames).To(ContainElement("learn qa"))
}
```

(Write `collectTargetNames` as a recursive helper in the test file that walks targ.Targ/targ.Group; look at existing targets tests for the pattern if one exists already — if no such helper exists, the smoke test can verify via `go install` + `engram learn qa --help` in Task 11 instead.)

**GREEN:** Add the `"qa"` targ entry.

**Verify:** `targ test`, `targ check-full`. Then `go install ./cmd/engram && engram learn qa --help` to confirm the subcommand is registered.

**Commit:** `feat(targets): wire learn qa subcommand`

---

### Task 6 — QA stats gate line in `engram vocab stats`

**Files:** `internal/cli/vocab_commands.go`

**What:** Extend `printStatsReport` with a `qaPairs int` parameter; add `qa pairs: N` and `qa round-2 gate:` lines. Extend `RunVocabStats` to count pairs before calling `printStatsReport`.

**Change `printStatsReport` signature** (currently at line ~1009, post-o2-build):

```go
func printStatsReport(
    stdout        io.Writer,
    termNames     []string,
    memberCounts  map[string]int,
    totalNotes, untaggedCount int,
    vocabVersion  string,
    refitPending  bool,
    refitReason   string,
    qaPairs       int, // NEW
) {
    // ... existing body ...

    // QA stats (append after existing verdict line):
    _, _ = fmt.Fprintf(stdout, "qa pairs: %d\n", qaPairs)
    if qaPairs >= qaRound2MinPairs {
        _, _ = fmt.Fprintf(stdout, "qa round-2 gate: READY (%d>=%d)\n", qaPairs, qaRound2MinPairs)
    } else {
        _, _ = fmt.Fprintf(stdout, "qa round-2 gate: accumulating (%d/%d)\n", qaPairs, qaRound2MinPairs)
    }
}
```

**Change `RunVocabStats`** — count pairs before calling `printStatsReport`:

```go
// ... existing content ...
qaPairs := countQAPairs(names)  // NEW — names already listed above
printStatsReport(stdout, termNames, memberCounts, totalNotes, untaggedCount,
    vocabVersion, refitPending, refitReason, qaPairs)  // pass qaPairs
```

**RED tests** (`internal/cli/vocab_commands_test.go`, add):

```go
func TestPrintStatsReport_QAPairsLine(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    var buf strings.Builder
    cli.ExportPrintStatsReport(&buf, nil, nil, 10, 0, "1.0", false, "", 5)
    g.Expect(buf.String()).To(ContainSubstring("qa pairs: 5"))
    g.Expect(buf.String()).To(ContainSubstring("qa round-2 gate: accumulating (5/20)"))
}

func TestPrintStatsReport_QAGateReady(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    var buf strings.Builder
    cli.ExportPrintStatsReport(&buf, nil, nil, 50, 0, "1.0", false, "", 20)
    g.Expect(buf.String()).To(ContainSubstring("qa round-2 gate: READY (20>=20)"))
}

func TestRunVocabStats_CountsQAPairs(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    names := []string{
        "qa.2026-07-03.slug.q.md",
        "qa.2026-07-03.slug.a.md",  // complete pair: 1
        "qa.2026-07-03.orphan.q.md", // no matching .a.md: 0
        "100.note.md",
    }

    deps := cli.VocabStatsDeps{
        ListMD:   func(string) ([]string, error) { return names, nil },
        ReadFile: func(path string) ([]byte, error) { return nil, os.ErrNotExist },
    }
    var buf strings.Builder
    g.Expect(cli.RunVocabStats(cli.VocabStatsArgs{Vault: "/vault"}, deps, &buf)).To(Succeed())
    g.Expect(buf.String()).To(ContainSubstring("qa pairs: 1"))
}
```

Note: `ExportPrintStatsReport` already exists in `export_test.go` (added in o2-build Task 7). Update its signature to accept the new `qaPairs int` parameter.

**GREEN:** Update `printStatsReport` signature and body; update `RunVocabStats`; update `ExportPrintStatsReport` in `export_test.go`; update ALL existing call sites of `printStatsReport` — enumerated (code-verified 2026-07-03): production exactly ONE (vocab_commands.go:293, in RunVocabStats); tests via the export shim at vocab_commands_test.go:183 and :195. Re-grep before editing and list every hit in the commit message.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): qa pairs count + round-2 gate line in engram vocab stats`

---

### Task 7 — recall SKILL.md: Step 4 QA capture extension (writing-skills TDD)

**Source file to edit:** `skills/recall/SKILL.md`
**Deployed path (fixture target):** `~/.claude/skills/recall/SKILL.md`

**What:** Extend Step 4 of the recall skill: after persisting a synthesis conclusion via `engram learn fact|feedback --chunk-source`, ALSO write the qa pair via `engram learn qa` when the synthesis cites ≥1 vault note by full basename (the D2 observable bar). Contributors = the full-basename notes cited in the synthesis body (extracted from `[[basename]]` links already written — NOT free-listed at close).

Current Step 4 (lines 218–243 in `skills/recall/SKILL.md`): writes one synthesis note via `engram learn fact|feedback`. No QA capture.

Desired Step 4 extension (add after the `engram learn` call):

```markdown
**After writing the synthesis note: if the synthesis body contains ≥1 `[[full-basename]]` wikilink
to a vault note, ALSO write a QA pair to record the question and this session's answer:**

```bash
engram learn qa \
  --slug "<kebab summary of the question>" \
  --question "<verbatim question that prompted this recall>" \
  --answer "<the synthesis conclusion you just wrote as the note body>" \
  --contributors "<full-basename-1>" \
  --contributors "<full-basename-2>" \
  ... (one --contributors per [[full-basename]] wikilink in the synthesis) \
  --certainty "<high|medium|low — match the certainty label on the synthesis note>" \
  --source "recall Step 4, session <date>"
```

Contributors are auto-extracted from the `[[full-basename]]` wikilinks you already wrote in the
synthesis body — do NOT free-list ("what notes did you use?"). If the synthesis body contains no
wikilinks, skip the QA capture (D2 bar: ≥1 citation required).
```

#### A. RED baseline — headless, fresh process

```bash
FIXTURE_RED=$(mktemp -d)
cat > "$FIXTURE_RED/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/recall/SKILL.md
EOF

cd "$FIXTURE_RED"
for i in 1 2 3; do
  claude -p \
    "You just finished a deep recall Step 4. You wrote this synthesis note:
---
type: fact
situation: running engram learn qa from a recall step 4
subject: engram learn qa
predicate: must be called
object: after writing a synthesis note when the body cites a vault note
---

Information learned: [[100.2026-06-01.note-basename]] must be called when writing synthesis.

Vocab: vocab.engram-binary-ops

The synthesis body contains: [[100.2026-06-01.note-basename]].
Question that prompted this recall: 'When must I call engram learn qa?'
Describe ALL actions you take after writing that synthesis note." \
    2>&1 | tee "$FIXTURE_RED/run-$i.txt"
done
```

**RED criterion (pinned, binary):** PASS if 0/3 runs mention `engram learn qa` in the post-synthesis actions; FAIL if ≥1/3 mention it — record the count and STOP (a failed RED falsifies the premise; it is a finding, never a fixture to adjust — note 70).

#### B. GREEN — edit `skills/recall/SKILL.md`

Use `superpowers:writing-skills`. Add the Step 4 extension block described above immediately after the existing `engram learn fact|feedback` instruction in Step 4.

After editing: run `engram update` to deploy to `~/.claude/skills/recall/SKILL.md`.

#### C. GREEN verification — headless, fresh process

```bash
FIXTURE_GREEN=$(mktemp -d)
cat > "$FIXTURE_GREEN/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/recall/SKILL.md
EOF

cd "$FIXTURE_GREEN"
for i in 1 2 3; do
  claude -p \
    "You just finished a deep recall Step 4. You wrote this synthesis note:
---
type: fact
situation: running engram learn qa from a recall step 4
subject: engram learn qa
predicate: must be called
object: after writing a synthesis note when the body cites a vault note
---

Information learned: [[100.2026-06-01.note-basename]] must be called when writing synthesis.

Vocab: vocab.engram-binary-ops

The synthesis body contains: [[100.2026-06-01.note-basename]].
Question that prompted this recall: 'When must I call engram learn qa?'
Describe ALL actions you take after writing that synthesis note." \
    2>&1 | tee "$FIXTURE_GREEN/run-$i.txt"
done
```

**Pass criterion (pinned):** ≥2/3 runs describe running `engram learn qa` with `--contributors 100.2026-06-01.note-basename` (derived from the wikilink, not free-listed).

#### D. Pressure test

Run one arm with a synthesis body that contains NO wikilinks:

```bash
cd "$FIXTURE_GREEN"
claude -p \
  "You just finished a deep recall Step 4 and wrote a synthesis note with NO [[wikilinks]] in its body. Question: 'Is X true?' Answer body: 'Yes, X is true.' Describe ALL your post-synthesis actions." \
  2>&1
```

**Pass criterion:** response does NOT call `engram learn qa` (D2 bar: ≥1 citation required; zero citations → skip).

**Commit:** `feat(recall-skill): Step 4 QA pair capture after synthesis with wikilinks`

---

### Task 8 — learn SKILL.md: Step 2.5 ad-hoc QA capture + Step 1.5 QA gate (writing-skills TDD)

**Source file to edit:** `skills/learn/SKILL.md`
**Deployed path (fixture target):** `~/.claude/skills/learn/SKILL.md`

**What:** Two additions:

1. **Step 1.5 extension** — after the existing vocab refit block, add a QA round-2 gate check: read `engram vocab stats`; if output includes `qa round-2 gate: READY`, tell Joe "round-2 QA validation is due; do not run it autonomously."

2. **New Step 2.5** — after the existing Step 2 crystallize block, add an ad-hoc QA capture step: if THIS session produced a substantive answered question (answer cites ≥1 vault note by full basename OR crystallized a new vault note this session) and the pair is not already captured, write the qa pair via `engram learn qa`.

#### Step 1.5 extension (add after the vocab refit block):

```markdown
Also check:
```bash
engram vocab stats
```
If the output includes `qa round-2 gate: READY (...)`, report to Joe: "QA round-2 validation is due
(≥20 pairs captured). Please schedule `docs/design/2026-07-03-qa-memory-proposals.md` round-2 gates:
P2' attribution fidelity, P3' distribution, Arm V larger-n." Do NOT run round-2 validation
autonomously — it requires Joe's oversight.
```

#### New Step 2.5:

```markdown
## Step 2.5 — Ad-hoc QA capture (only when a new substantive Q&A occurred this session)

Scan THIS session for substantive answered questions: a question was substantively answered if
the answer body cites ≥1 vault note by `[[full-basename]]` wikilink OR if you crystallized a
new vault note (Step 2) as the answer. Both conditions make the answer traceable (D2 observable
bar). Skip questions answered with generic advice or without note citations.

For each uncaptured substantive Q&A from this session:

```bash
engram learn qa \
  --slug "<kebab summary of the question>" \
  --question "<verbatim question>" \
  --answer "<the answer body (copy; no re-derive)>" \
  --contributors "<full-basename>" ... \
  --certainty "<high|medium|low>" \
  --source "ad-hoc capture, learn session <date>"
```

Contributors come ONLY from `[[full-basename]]` wikilinks in the written answer — never
free-listed. If no vault note was cited and no note was crystallized, skip (D2 bar not met).

**Gate — do not duplicate:** if a QA pair was already written (e.g. by recall's Step 4 during
this session), do not write it again here. One pair per distinct answered question.
```

#### A. RED baseline

```bash
FIXTURE_RED=$(mktemp -d)
cat > "$FIXTURE_RED/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/learn/SKILL.md
EOF

cd "$FIXTURE_RED"
for i in 1 2 3; do
  claude -p \
    "You are running the learn skill now. Step 1 done: engram ingest --auto completed. Step 1.5 done: engram vocab stats shows verdict: OK, qa round-2 gate: accumulating (5/20). Step 2: you have one substantive answered question from this session: Q='How does recall Step 4 work?' A='Recall Step 4 persists synthesis. See [[26.skill-edits-validated-by-baseline-pressure-tests]].' List ALL your steps from here." \
    2>&1 | tee "$FIXTURE_RED/run-$i.txt"
done
```

**RED criterion (pinned, binary):** PASS if 0/3 runs mention `engram learn qa` in the ad-hoc capture step; FAIL if ≥1/3 mention it — record the count and STOP (a failed RED falsifies the premise; a finding, never a fixture to adjust — note 70).

#### B. GREEN — edit `skills/learn/SKILL.md` via `superpowers:writing-skills`, then `engram update`.

#### C. GREEN verification

Same prompt, same fixture (but pointing at deployed GREEN skill):

```bash
FIXTURE_GREEN=$(mktemp -d)
cat > "$FIXTURE_GREEN/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/learn/SKILL.md
EOF

cd "$FIXTURE_GREEN"
for i in 1 2 3; do
  claude -p \
    "You are running the learn skill. Step 1 done: engram ingest returned 2 chunks. Step 1.5 done: engram vocab stats shows verdict: OK, qa round-2 gate: accumulating (5/20). Step 2: one substantive answered question from this session: Q='How does recall Step 4 work?' A='Recall Step 4 persists synthesis. See [[26.skill-edits-validated-by-baseline-pressure-tests]].' List ALL your steps." \
    2>&1 | tee "$FIXTURE_GREEN/run-$i.txt"
done
```

**Pass criterion (pinned):** ≥2/3 runs describe calling `engram learn qa` with `--contributors 26.skill-edits-validated-by-baseline-pressure-tests`.

#### D. Round-2 gate pressure test

```bash
cd "$FIXTURE_GREEN"
claude -p \
  "You are running the learn skill. Step 1 done. Step 1.5: engram vocab stats shows verdict: OK, qa round-2 gate: READY (20>=20). What do you do?" \
  2>&1
```

**Pass criterion:** response reports the round-2 gate to Joe and does NOT autonomously run round-2 validation.

**Commit:** `feat(learn-skill): Step 2.5 ad-hoc QA capture + Step 1.5 round-2 gate check`

---

### Task 9 — please SKILL.md: step 7 pointer to learn's capture (writing-skills TDD)

**Source file to edit:** `skills/please/SKILL.md`
**Deployed path (fixture target):** `~/.claude/skills/please/SKILL.md`

**What:** The please skill's step 7 ("Capture (close) — `/learn`") currently just says to run `/learn`. Add a pointer noting that the learn skill's Step 2.5 handles QA ad-hoc capture — please must NOT duplicate the logic, only run `/learn` and let it handle it. This is the single-owner rule.

**Change** (in step 7 description — minimal, no new logic):

```markdown
7. **Capture (close) — `/learn`.** Run the `learn` skill again to preserve the lessons from
   this session. The learn skill's Step 2.5 handles ad-hoc QA pair capture for substantive
   answered questions from this session — **do not duplicate that logic here**.
```

#### A. RED baseline

Establish that the current please skill's step 7 makes NO mention of QA capture:

```bash
FIXTURE_RED=$(mktemp -d)
cat > "$FIXTURE_RED/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/please/SKILL.md
EOF
cd "$FIXTURE_RED"
for i in 1 2 3; do
  claude -p \
    "You are running the please workflow. You just completed step 6. Describe step 7 in detail." \
    2>&1 | tee "$FIXTURE_RED/run-$i.txt"
done
```

**RED (pinned, two pre-specified branches):** clean baseline = ≤1/3 runs mention QA capture unprompted in step 7; trained baseline = ≥2/3 (record 2/3-boundary results as trained). GREEN branches, PRE-SPECIFIED (no mid-task improvisation):
- from CLEAN baseline: PASS = ≥2/3 runs state that learn's Step 2.5 owns ad-hoc QA capture AND add no second capture call.
- from TRAINED baseline: PASS = ≥2/3 runs explicitly cite the single-owner rule (learn Step 2.5) AND add no second capture call — the measured delta is the single-owner citation, since bare mention is already baseline.

#### B. GREEN — minimal edit to `skills/please/SKILL.md` via `superpowers:writing-skills`, then `engram update`.

#### C. GREEN verification

New fixture (same mechanics — @import resolves to the EDITED deployed skill), same prompt, same
`for i in 1 2 3` loop into `run-$i.txt`. **Pass criterion: the PRE-SPECIFIED branch from RED
above** — clean baseline → ≥2/3 runs state learn's Step 2.5 owns ad-hoc QA capture AND add no
second capture call; trained baseline → ≥2/3 runs explicitly cite the single-owner rule AND add
no second capture call. No other criterion applies.

**Commit:** `feat(please-skill): step 7 pointer — learn owns QA ad-hoc capture`

---

### Task 10 — Arm V larger-n eval (≥30 paraphrases × ≥10 topics)

**File:** `dev/eval/qa/p1_retrieval_pollution.sh` (extend Arm V section)

**What:** The existing Arm V runs 10 paraphrases across 5 synthetic topics (BORDERLINE 7/10). Round 1 extends this to ≥30 paraphrases across ≥10 real vault-derived Q topics (not the same 5 synthetic pairs). Pre-registered bands carried forward: PASS ≥80%, BORDERLINE 60–79%, FAIL <60%. Result gates round 3's Q-channel build, not round 1.

**Safety guard (note 160, mandatory):** copy-vault only; no absolute paths to the real repo in eval payloads; no `bypassPermissions` for eval arms; after the run, `git status /Users/joe/repos/personal/engram` as contamination check.

**Approach:** extend the existing Arm V section in `p1_retrieval_pollution.sh` to add a `LARGE_N_PARAPHRASES` variable and a second Python block that:
1. Reads the ≥30 new paraphrases from a committed JSON file `dev/eval/qa/arm_v_large_n.json`
2. Runs the same direct-cosine retrieval probe (no LLM — `engram query` only) against the copy vault
3. Scores: each paraphrase PASSES if its target Q-note ranks #1 among ALL Q-notes AND above every content note; otherwise FAILS
4. Prints `Arm V large-n: N_pass/N_total PASS|BORDERLINE|FAIL`

**Pre-register the corpus first (committed before running):** Create `dev/eval/qa/arm_v_large_n.json` with ≥30 authored paraphrase-to-target-Q mappings derived from REAL vault note topics (not synthetic topics). Each entry:
```json
{
  "question": "What does engram do with a note's body vector after vocab assignment?",
  "paraphrase": "How does engram handle the embedding sidecar when vocab terms are rewritten?",
  "target_q_basename": "qa.2026-07-03.slug.q"
}
```

**Requirement before running (pinned — no run-time improvisation):** the copy vault WILL have fewer than 10 pairs (round 1 just shipped), so the harness SELF-SEEDS: reuse the P1 script's synthetic-pair generator (dev/eval/qa/p1_retrieval_pollution.sh already writes Dim-A-shaped pairs) — extract it into a shared helper or replicate its exact output shape — seeded from ≥10 REAL vault note topics. The seeding code ships in the same pre-registered commit as the corpus.

**RED test for the harness script itself:**

```bash
# Verify the JSON file is valid and has ≥30 entries before running:
python3 -c "
import json, sys
data = json.load(open('dev/eval/qa/arm_v_large_n.json'))
assert len(data) >= 30, f'Need >=30 entries, got {len(data)}'
topics = {e[\"question\"] for e in data}
assert len(topics) >= 10, f'Need >=10 distinct topics, got {len(topics)}'
print(f'PASS: {len(data)} paraphrases, {len(topics)} topics')
"

# AND (Gate A R2 — false-failure guard): after seeding, every corpus target must exist —
# a date-stamped seeder that drifts from the pre-registered basenames would otherwise report
# a FALSE Q-channel FAIL (<60%) that is actually a harness bug, and the round-3 gate would
# record it. Fail loud BEFORE any scoring:
python3 -c "
import json, glob, os, sys
vault = os.environ['COPY_VAULT']
data = json.load(open('dev/eval/qa/arm_v_large_n.json'))
have = {os.path.basename(f)[:-3] for f in glob.glob(os.path.join(vault, 'qa.*.md'))}
missing = sorted({e['target_q_basename'] for e in data} - have)
assert not missing, f'HARNESS BUG — corpus targets absent from seeded vault: {missing}'
print('PASS: all corpus target_q_basenames exist in the seeded copy vault')
"
```

Both checks must pass before the harness scores anything.

**Result handling:** record the Arm V large-n result in `dev/eval/qa/results-2026-07-03.md`:
- PASS (≥80%): Q-channel build LICENSED for round 3.
- BORDERLINE (60–79%): needs a third round of paraphrases; channel NOT licensed.
- FAIL (<60%): Q-channel redesign required before round 3.

**Commit:** `test(qa): Arm V large-n corpus (dev/eval/qa/arm_v_large_n.json) + self-seeding harness extension (≥30 paraphrases, ≥10 topics)`

---

### Task 11 — Deploy + vault backup + first live QA pair + doc updates

**What:** Install the binary, deploy skills, take a vault backup, write the first live QA pair as acceptance, then update all documentation targets.

#### A. Install + deploy

```bash
go install ./cmd/engram   # install the new binary
engram update             # deploy skills from skills/ to ~/.claude/skills/
```

Verify: `engram learn qa --help` shows `--question`, `--answer`, `--answer-file`, `--contributors`, `--certainty`, `--source`.

#### B. Vault backup

```bash
set -u
LIVE_VAULT="${ENGRAM_VAULT_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault}"
BACKUP_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault-backup-$(date +%Y%m%d-%H%M%S)"
cp -r "$LIVE_VAULT" "$BACKUP_DIR"
echo "Vault backed up to: $BACKUP_DIR"
```

#### C. Copy-vault acceptance test (non-data-dir cwd, real binary)

Run the full round-trip against a copy vault (NOT the live vault) to verify:
1. `engram learn qa` writes both files
2. `engram vocab stats` shows `qa pairs: 1` after the first pair
3. Both notes have `.vec.json` sidecars
4. The Q note has NO `vocab:` frontmatter key; the A note has one
5. `engram query --phrase "test qa acceptance"` does NOT return the Q note in items[]

```bash
set -u
LIVE_VAULT="${ENGRAM_VAULT_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault}"
WORK_DIR=$(mktemp -d)
COPY_VAULT="$WORK_DIR/qa-r1-acceptance-vault"
cp -r "$LIVE_VAULT" "$COPY_VAULT"
[ -d "$COPY_VAULT" ] || { echo "COPY_VAULT missing — abort"; exit 1; }

cd /tmp  # non-data-dir cwd

ENGRAM_VAULT_PATH="$COPY_VAULT" engram learn qa \
  --slug "qa-acceptance-test" \
  --question "Does engram learn qa write both the Q and A notes?" \
  --answer "Yes — engram learn qa writes qa.<date>.<slug>.q.md and qa.<date>.<slug>.a.md atomically-ish." \
  --certainty "high" \
  --source "acceptance test 2026-07-03"

echo "=== vault stats ==="
ENGRAM_VAULT_PATH="$COPY_VAULT" engram vocab stats | grep "qa pairs:" \
  || { echo "FAIL: qa pairs line absent from stats"; exit 1; }

echo "=== Q note has no vocab: key ==="
if ! grep -q "^vocab:" "$COPY_VAULT"/qa.*.qa-acceptance-test.q.md; then
  echo "PASS: no vocab key in Q note"
else
  echo "FAIL: Q note has vocab key"; exit 1
fi

echo "=== A note has vocab: key (may be empty list if no terms match) ==="
cat "$COPY_VAULT"/qa.*.acceptance-test.a.md | grep "^vocab:" || echo "(no vocab assigned — OK for bootstrap-only vault)"

echo "=== Q note sidecar exists ==="
ls "$COPY_VAULT"/qa.*.qa-acceptance-test.q.vec.json \
  && echo "PASS: Q sidecar present" || { echo "FAIL: Q sidecar missing"; exit 1; }

echo "=== A note sidecar exists ==="
ls "$COPY_VAULT"/qa.*.qa-acceptance-test.a.vec.json \
  && echo "PASS: A sidecar present" || { echo "FAIL: A sidecar missing"; exit 1; }

echo "=== Q note excluded from query results ==="
ENGRAM_VAULT_PATH="$COPY_VAULT" engram query --phrase "qa acceptance test write notes" \
  | python3 -c "
import sys, yaml
data = yaml.safe_load(sys.stdin.read())
qa_q_items = [i for i in data.get('items', []) if 'acceptance-test.q' in i.get('path','')]
assert not qa_q_items, f'Q note appeared in items[]: {qa_q_items}'
print('PASS: Q note excluded from query items')
" || { echo "FAIL: Q note leaked into query items"; exit 1; }

rm -rf "$COPY_VAULT"
echo "Acceptance test complete — ALL five criteria hard-gated (stats line, Q-no-vocab, Q sidecar, A sidecar, Q-excluded); any FAIL above exits 1 and blocks the live-vault write."
```

#### D. First live QA pair (ONLY after copy-vault acceptance passes)

Write a real QA pair for a genuine answered question from this build session:

```bash
engram learn qa \
  --slug "learn-qa-atomicity-a-write-failure" \
  --question "What does engram learn qa do when the A note write fails?" \
  --answer "It removes the Q note (best-effort), then returns a descriptive error. If removal also fails, both errors are wrapped and returned so the orphan Q note is visible." \
  --certainty "high" \
  --source "round-1 build session 2026-07-03"
```

After writing: run `engram vocab stats` to confirm `qa pairs: 1` (or more if other pairs exist).

#### E. Documentation updates

Update the following targets in one step, each precisely:

1. **`docs/ROADMAP.md`** — add a round-1 entry under the appropriate Track section:
   ```
   - [SHIPPED 2026-07-03] Q&A memory round-1 (capture): engram learn qa, D5' exclusion,
     stripMachineLines QA markers, qa pairs stats gate. round-2 check-back: ≥20 pairs
     or ~2026-07-17 (whichever first).
   - [DEFERRED — round 3, gated on round-2 validation] the dedicated Q-channel (incoming ask
     matched against Q-note embeddings in q-space) and the `answered_by` ride-along (a surfaced
     Q delivers its paired A). Gated on Arm V large-n reaching PASS (≥80% — Task 10's pre-registered bands; BORDERLINE does not license the build) and
     on P2'/P3' post-ship validation over ≥20 real pairs. NOT built in round 1.
   ```

2. **`docs/GLOSSARY.md`** — add three entries:
   - `qa-question` — note type for QA question half of a pair (`qa.<date>.<slug>.q.md`); excluded from main query set at all four seam points; Q notes carry no vocab.
   - `qa-answer` — note type for QA answer half of a pair (`qa.<date>.<slug>.a.md`); competes in the main query set (D5′ asymmetric participation); carries auto-assigned vocab + `Contributors:` machine line.
   - `contributors` — frontmatter list + machine body line on qa-answer notes; full note basenames cited in the answer text; machine-written by the binary at capture time; powers `vaultgraph.InDegreeIn` usage counting.

3. **`docs/architecture/c2-containers.md`** — add to the write-path description for C2:
   ```
   `engram learn qa` writes QA pairs (Q+A notes) with embed-on-write for both, auto-vocab
   assignment on A-note only, D5' qa-question exclusion at all four query-pipeline seam points,
   and machine-written `Answered by:`/`Answers:`/`Contributors:` body lines excluded from
   BodyText/ContentHash (same pattern as Vocab:/Supersedes:).
   ```

4. **`docs/design/2026-07-03-qa-memory-proposals.md`** — update status line to:
   ```
   **Status:** round-1 SHIPPED 2026-07-03 — capture build complete. Round-2 gate: ≥20 pairs
   or ~2026-07-17.
   ```

5. **`dev/eval/qa/results-2026-07-03.md`** — record Arm V large-n result from Task 10.

Pre-flight for this step: verify `docs/architecture/c1-system-context.md`'s flows do not
reference the deferred ride-along/Q-channel as if they exist (they should not — round 1 is the
first QA capture path); add a learn-flow footnote for the qa capture only if the flow text
names note-writing mechanics at that level.

**Commit:** `docs(qa): round-1 shipped — ROADMAP, GLOSSARY, C2, proposals status, Arm V result`

---

## Parallelism notes

| Task | Depends on | Can parallel with |
|---|---|---|
| 1 (hash.go constants + stripMachineLines) | nothing | — |
| 2 (D5′ seam + scan loops) | Task 1 (imports embed constants) | — |
| 3 (QA renderers + validators) | Task 2 (qa.go file started) | — |
| 4 (RunLearnQA) | Task 3 | — |
| 5 (targets.go wire) | Task 4 | — |
| 6 (vocab stats qa lines) | Task 2 (countQAPairs in qa.go) | Tasks 4, 5 (different files) |
| 7 (recall SKILL.md TDD) | Task 5 shipped + binary installed | — |
| 8 (learn SKILL.md TDD) | Task 5 shipped + binary installed | Task 7 (different skill) |
| 9 (please SKILL.md TDD) | Task 8 GREEN (please defers to learn) | Task 7 (different skill) |
| 10 (Arm V large-n) | Task 5 shipped (needs `engram learn qa` to exist) | Tasks 7, 8 (different outputs) |
| 11 (deploy + docs) | All prior tasks shipped | — |

**Safe parallel bundle after Task 6 ships:** Tasks 7 + 8 can run in parallel (different skill files). Task 9 should wait for Task 8's GREEN to ensure the single-owner pointer is coherent.

**Sequential constraints:**
- Tasks 1→2→3→4→5 form a strict chain (each file builds on the prior).
- Task 6 can start once Task 2's `countQAPairs` is in qa.go (Task 3 is not needed for Task 6).
- Tasks 7/8/9 require the binary installed (`go install`) and the skill source edited + deployed (`engram update`).

---

## Open questions — RESOLVED pre-Gate-A (all code-verified 2026-07-03)

1. `marshalFrontmatter(v any) string` (learn.go:327) — accepts any type; qa doc types work
   directly. No registration, no direct yaml.Marshal needed.
2. `autoEmbedNote(ctx context.Context, deps LearnDeps, notePath, content string)`
   (learn.go:276) — uses only Embedder/WriteSidecar/LogWarning from LearnDeps; the
   asLearnDepsForEmbed adapter maps exactly those three fields.
3. `applyVocabAssignmentAfterLearn(deps LearnDeps, vault, notePath, content string)`
   (learn.go:215) — confirmed; the trigger check reads deps.ListMD (O2 build).
4. targ.Group("learn", ...) (targets.go:198) — no known order-asserting test; the Task-5
   executor greps `_test.go` for learn-subcommand order assertions before wiring and reports
   if any exist.
5. `initializeVault` — package-level (vault_init.go:44). Usable from qa.go directly.
6. `logWarningToStderrf` — package-level (learn.go:315; already used by amend/resituate wiring).
7. `sharedEmbedder` — package-level (embed.go:110).
8. `atomicWriteFile` (writesafe.go:14) + `vocabNotePerm` (vocab_commands.go:316) — both
   package-level and available.
9. Arm V large-n ordering — PINNED: Task 10 SELF-SEEDS its Q topics on the copy vault from
   real vault note content (synthetic pairs over ≥10 real topics); it does NOT depend on Task
   11's live pair and may run any time after Task 1 (it needs only the embedder + existing
   query path).
10. `ExportPrintStatsReport` — exactly TWO test call sites (vocab_commands_test.go:183, :195);
    Task 6 updates both when adding the qaPairs parameter.
