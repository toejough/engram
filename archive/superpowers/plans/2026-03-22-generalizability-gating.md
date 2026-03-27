# Generalizability Gating Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce memory extraction noise by adding generalizability scoring, hard-gating low-value memories, tracking project provenance, and fixing the classify path's transcript stripping.

**Architecture:** Add `Generalizability int` to candidate/classified/enriched/record types. Sharpen both LLM prompts with a concrete litmus test and 1-5 scale. Filter candidates scoring <2 before writing. Thread `project_slug` from hooks through CLI to TOML. Fix the classify path to strip transcripts via DI.

**Tech Stack:** Go, TOML, Bash hooks, Anthropic API (Haiku)

**Build/test commands:** `targ test` (tests), `targ check-full` (lint + coverage)

---

### Task 1: Add Generalizability to memory types

**Files:**
- Modify: `internal/memory/memory.go:8-19` (CandidateLearning), `internal/memory/memory.go:23-36` (ClassifiedMemory), `internal/memory/memory.go:57-70` (Enriched), `internal/memory/memory.go:39-54` (ToEnriched)
- Modify: `internal/memory/record.go:32-66` (MemoryRecord)

- [ ] **Step 1: Add `Generalizability int` field to `CandidateLearning`**

In `internal/memory/memory.go`, add after `FilenameSummary` (line 18):

```go
Generalizability int
```

- [ ] **Step 2: Add `Generalizability int` field to `ClassifiedMemory`**

In `internal/memory/memory.go`, add after `FilenameSummary` (line 34):

```go
Generalizability int
```

- [ ] **Step 3: Add `Generalizability int` field to `Enriched`**

In `internal/memory/memory.go`, add after `FilenameSummary` (line 66):

```go
Generalizability int
```

- [ ] **Step 4: Add `Generalizability` to `ToEnriched()` mapping**

In `internal/memory/memory.go`, add inside the `ToEnriched()` struct literal (after `FilenameSummary` line ~49):

```go
Generalizability: cm.Generalizability,
```

- [ ] **Step 5: Add `ProjectSlug` and `Generalizability` to `MemoryRecord`**

In `internal/memory/record.go`, add after the `Rationale` field (around line 40):

```go
ProjectSlug      string `toml:"project_slug,omitempty"`
Generalizability int    `toml:"generalizability,omitempty"`
```

- [ ] **Step 6: Run tests to verify nothing breaks**

Run: `targ test`
Expected: All existing tests pass (new fields have zero values, TOML `omitempty` means no output change for existing records).

- [ ] **Step 7: Commit**

```bash
git add internal/memory/memory.go internal/memory/record.go
git commit -m "feat(memory): add Generalizability, ProjectSlug fields to memory types"
```

---

### Task 2: Add generalizability to extraction prompt and JSON parsing

**Files:**
- Modify: `internal/extract/extract.go:164-175` (llmCandidateLearningJSON), `internal/extract/extract.go:206-217` (parseLLMResponse mapping), `internal/extract/extract.go:249-298` (systemPrompt)
- Test: `internal/extract/extract_test.go`

- [ ] **Step 1: Write failing test — extraction response with generalizability field parses correctly**

In `internal/extract/extract_test.go`, add a test that provides a mock HTTP response with a JSON array containing `"generalizability": 4` and verifies the parsed `CandidateLearning` has `Generalizability: 4`.

Find an existing test that exercises `parseLLMResponse` or the `Extract` method with a mock HTTP client and follow that pattern. The test should assert `g.Expect(candidates[0].Generalizability).To(Equal(4))`.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `Generalizability` field not populated yet.

- [ ] **Step 3: Add `Generalizability` to `llmCandidateLearningJSON`**

In `internal/extract/extract.go`, add to the struct (after `FileSummary` at line ~174):

```go
Generalizability int `json:"generalizability"`
```

- [ ] **Step 4: Add `Generalizability` to the mapping in `parseLLMResponse`**

In `internal/extract/extract.go`, inside the loop at line ~206-217, add to the `CandidateLearning` struct literal:

```go
Generalizability: raw.Generalizability,
```

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Sharpen the system prompt**

In `internal/extract/extract.go`, modify `systemPrompt()` (line 249-298):

Replace the ephemeral context bullet in the QUALITY GATE section:

```
- ephemeral context that does not generalize across sessions (e.g., current task status,
  one-off session state, transient observations that only apply to this specific moment)
```

With:

```
- ephemeral context: task/validation status updates (e.g., "S6 is validated," "step 3 is complete"),
  debugging observations about specific data or state (e.g., "pipeline produced flat faces,"
  "normals are inverted on mesh B"), project-specific variable/file names without a generalizable
  principle. Litmus test: would a developer on a different task in a different project, weeks from
  now, benefit from knowing this? If probably not, reject it or score it low.
```

Add `generalizability` to the JSON schema in the return format section, after `filename_summary`:

```
"generalizability": "Integer 1-5: 1=only this session, 2=this project/narrow, 3=across this project, 4=across similar projects, 5=universal"
```

- [ ] **Step 7: Run full test suite**

Run: `targ test`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/extract/extract.go internal/extract/extract_test.go
git commit -m "feat(extract): add generalizability scoring to extraction prompt and parsing"
```

---

### Task 3: Add generalizability to classify prompt and JSON parsing

**Files:**
- Modify: `internal/classify/classify.go:165-176` (llmClassifyJSON), `internal/classify/classify.go:286-299` (parseClassifyResponse mapping), `internal/classify/classify.go:330-375` (systemPrompt)
- Test: `internal/classify/classify_test.go`

- [ ] **Step 1: Write failing test — classify response with generalizability field parses correctly**

In `internal/classify/classify_test.go`, add a test that provides a mock HTTP response with `"generalizability": 3` and verifies the parsed `ClassifiedMemory` has `Generalizability: 3`.

Follow the existing test pattern for `Classify` with a mock HTTP client. Assert `g.Expect(result.Generalizability).To(Equal(3))`.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `Generalizability` field not populated.

- [ ] **Step 3: Add `Generalizability` to `llmClassifyJSON`**

In `internal/classify/classify.go`, add to the struct (after `FileSummary` at line ~175):

```go
Generalizability int `json:"generalizability"`
```

- [ ] **Step 4: Add `Generalizability` to the mapping in `parseClassifyResponse`**

In `internal/classify/classify.go`, inside the `ClassifiedMemory` struct literal (lines ~286-299), add:

```go
Generalizability: parsed.Generalizability,
```

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Sharpen the classify system prompt**

In `internal/classify/classify.go`, modify `systemPrompt()` (line 330-375):

In the `null` tier description, replace:

```
  (e.g., current task status, one-off session state, transient observations that apply only
  to this specific moment and would not be useful in a future session).
```

With:

```
  (e.g., current task status, one-off session state, transient observations that apply only
  to this specific moment). Includes: task/validation status updates ("S6 is validated"),
  debugging observations about specific data ("pipeline produced flat faces"), project-specific
  names without a generalizable principle. Litmus test: would a developer on a different task
  in a different project, weeks from now, benefit from this? If probably not, classify as null.
```

Add `generalizability` to the JSON return format, after `filename_summary`:

```
"generalizability": "Integer 1-5: 1=only this session, 2=this project/narrow, 3=across this project, 4=across similar projects, 5=universal. null-tier messages should not include this field."
```

- [ ] **Step 7: Run full test suite**

Run: `targ test`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/classify/classify.go internal/classify/classify_test.go
git commit -m "feat(classify): add generalizability scoring to classify prompt and parsing"
```

---

### Task 4: Hard gate in learn pipeline — filter candidates with generalizability < 2

**Files:**
- Modify: `internal/learn/learn.go:78-130` (Run method)
- Test: `internal/learn/learn_test.go`

- [ ] **Step 1: Write failing test — candidates with generalizability < 2 are dropped**

In `internal/learn/learn_test.go`, add a test that:
- Creates a `Learner` with a fake extractor returning two candidates: one with `Generalizability: 1` and one with `Generalizability: 3`
- Uses a fake retriever returning no existing memories
- Uses a fake deduplicator that passes all candidates through as `Surviving`
- Calls `Run` and asserts only one memory was created (the one with generalizability 3)
- Asserts `result.SkippedCount` accounts for the dropped candidate

Follow the existing test patterns in `learn_test.go` for fakes/stubs.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — both candidates are written (no filter exists yet).

- [ ] **Step 3: Write failing test — candidates with generalizability == 0 are kept (backward compat)**

Add a test where the extractor returns a candidate with `Generalizability: 0` (default/unset). Assert it is NOT dropped — zero means "pre-existing or LLM didn't return the field," not "low quality."

- [ ] **Step 4: Run test to verify it fails (or passes if 0 is already kept)**

Run: `targ test`
Expected: This may already pass since 0 < 2 is true but we want 0 to be kept. If it passes now, it will fail after we add the filter.

- [ ] **Step 5: Implement the generalizability filter in `Run`**

In `internal/learn/learn.go`, add filtering after extraction (line ~83) and before dedup (line ~94):

```go
const minGeneralizability = 2

// Filter low-generalizability candidates (generalizability gating).
filtered := make([]memory.CandidateLearning, 0, len(candidates))
droppedCount := 0

for _, c := range candidates {
    if c.Generalizability >= minGeneralizability || c.Generalizability == 0 {
        filtered = append(filtered, c)
    } else {
        droppedCount++

        if l.stderr != nil {
            _, _ = fmt.Fprintf(l.stderr,
                "[engram] dropped (generalizability=%d): %q\n",
                c.Generalizability, c.Title)
        }
    }
}

candidates = filtered
```

Update `skippedCount` calculation (line ~97) to include `droppedCount`:

```go
skippedCount := len(candidates) - len(surviving) - len(mergePairs) + droppedCount
```

Wait — `candidates` has already been reassigned to `filtered` at this point, so the existing calculation won't double-count. But we need `droppedCount` in the `Result`. The cleanest approach: track it separately and add it to `SkippedCount` at the end. Adjust:

```go
skippedCount := droppedCount + len(candidates) - len(surviving) - len(mergePairs)
```

Note: `len(candidates)` here is the length of `filtered` (post-filter), so this correctly counts: dropped by filter + dropped by dedup.

**Important:** This replaces the existing `skippedCount` assignment at line ~97 — delete the old line and use this formula instead.

- [ ] **Step 6: Run tests to verify both pass**

Run: `targ test`
Expected: PASS — generalizability 1 is dropped, generalizability 0 is kept, generalizability 3 is kept.

- [ ] **Step 7: Commit**

```bash
git add internal/learn/learn.go internal/learn/learn_test.go
git commit -m "feat(learn): hard-gate candidates with generalizability < 2"
```

---

### Task 5: Hard gate in correct pipeline — filter ClassifiedMemory with generalizability < 2

**Files:**
- Modify: `internal/correct/correct.go:45-68` (Run method)
- Test: `internal/correct/correct_test.go`

- [ ] **Step 1: Write failing test — classified memory with generalizability 1 is not written**

In `internal/correct/correct_test.go`, add a test that:
- Creates a `Corrector` with a fake classifier returning a `ClassifiedMemory` with `Generalizability: 1`
- Calls `Run` and asserts the writer was NOT called
- Asserts the return string is empty (no reminder generated for dropped memories)

Follow existing test patterns for fakes.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — the writer is called regardless of generalizability.

- [ ] **Step 3: Write failing test — classified memory with generalizability 0 is kept (backward compat)**

Add a test where classifier returns `Generalizability: 0`. Assert the writer IS called.

- [ ] **Step 4: Implement the generalizability filter in `Run`**

In `internal/correct/correct.go`, add after the nil check (line ~56) and before `ToEnriched()` (line ~58):

```go
const minGeneralizability = 2

if classified.Generalizability > 0 && classified.Generalizability < minGeneralizability {
    return "", nil
}
```

- [ ] **Step 5: Run tests to verify both pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/correct/correct.go internal/correct/correct_test.go
git commit -m "feat(correct): hard-gate classified memories with generalizability < 2"
```

---

### Task 6: Thread project_slug through tomlwriter

**Files:**
- Modify: `internal/memory/memory.go:57-70` (Enriched struct)
- Modify: `internal/tomlwriter/tomlwriter.go:76-88` (Write method, record literal)
- Test: `internal/tomlwriter/tomlwriter_test.go`

- [ ] **Step 1: Write failing test — Enriched with ProjectSlug writes to TOML**

In `internal/tomlwriter/tomlwriter_test.go`, add a test that:
- Creates an `Enriched` with `ProjectSlug: "-Users-joe-repos-foo"` and `Generalizability: 4`
- Calls `Write`
- Reads the written TOML file and asserts it contains `project_slug = "-Users-joe-repos-foo"` and `generalizability = 4`

Follow existing `Write` test patterns.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: Won't compile — `Enriched` doesn't have `ProjectSlug` field yet. This is the expected red state.

- [ ] **Step 3: Add `ProjectSlug` to `Enriched` struct**

In `internal/memory/memory.go`, add to `Enriched` (after `Generalizability`):

```go
ProjectSlug string
```

- [ ] **Step 4: Add `ProjectSlug` and `Generalizability` to `ToEnriched()` mapping**

In `internal/memory/memory.go`, add inside `ToEnriched()` (should already have `Generalizability` from Task 1). Verify `ProjectSlug` doesn't need mapping here — `ClassifiedMemory` doesn't have `ProjectSlug` since project slug comes from CLI flags, not the LLM. So `ToEnriched()` doesn't map it.

Skip — `ProjectSlug` is set at the CLI wiring layer on `Enriched` directly, not derived from `ClassifiedMemory`.

- [ ] **Step 5: Set `ProjectSlug` and `Generalizability` on the record in `tomlwriter.Write`**

In `internal/tomlwriter/tomlwriter.go`, add to the `record` struct literal (after `UpdatedAt` at line ~87):

```go
ProjectSlug:      mem.ProjectSlug,
Generalizability: mem.Generalizability,
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/memory/memory.go internal/tomlwriter/tomlwriter.go internal/tomlwriter/tomlwriter_test.go
git commit -m "feat(tomlwriter): write project_slug and generalizability to TOML"
```

---

### Task 7: Thread project_slug through learn and correct CLI wiring

**Files:**
- Modify: `internal/learn/learn.go:315-332` (writeCandidate, Enriched literal)
- Modify: `internal/learn/learn.go:43-56` (Learner struct), `internal/learn/learn.go:59-74` (New func)
- Modify: `internal/correct/correct.go:45-68` (Run method), `internal/correct/correct.go:21-26` (Corrector struct)
- Modify: `internal/cli/cli.go:136-219` (RunLearn), `internal/cli/cli.go:1057-1104` (runCorrect)
- Test: `internal/learn/learn_test.go`, `internal/correct/correct_test.go`

- [ ] **Step 1: Write failing test — Learner passes projectSlug to written Enriched**

In `internal/learn/learn_test.go`, add a test that:
- Creates a `Learner` with `projectSlug` set (new field on Learner or New param)
- Runs extraction with a candidate
- Asserts the `Enriched` passed to `writer.Write` has `ProjectSlug` set

Use a spy writer that captures the `*memory.Enriched` argument.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `Learner` doesn't accept or pass `projectSlug`.

- [ ] **Step 3: Add `projectSlug` to `Learner` and wire through `writeCandidate`**

In `internal/learn/learn.go`:

Add field to `Learner` struct (line ~56):
```go
projectSlug string
```

Add setter method (after existing setters):
```go
// SetProjectSlug sets the originating project slug for new memories.
func (l *Learner) SetProjectSlug(slug string) {
    l.projectSlug = slug
}
```

In `writeCandidate` (line ~319-332), add to the `Enriched` literal:
```go
ProjectSlug:      l.projectSlug,
Generalizability: candidate.Generalizability,
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write failing test — Corrector passes projectSlug to written Enriched**

In `internal/correct/correct_test.go`, add a test that:
- Creates a `Corrector` with `projectSlug` set
- Runs with a classified memory
- Asserts the `Enriched` passed to `writer.Write` has `ProjectSlug` set

- [ ] **Step 6: Run test to verify it fails**

Run: `targ test`
Expected: FAIL

- [ ] **Step 7: Add `projectSlug` to `Corrector` and wire through `Run`**

In `internal/correct/correct.go`:

Add field to `Corrector` struct:
```go
projectSlug string
```

Add setter:
```go
// SetProjectSlug sets the originating project slug for new memories.
func (c *Corrector) SetProjectSlug(slug string) {
    c.projectSlug = slug
}
```

In `Run` (line ~58), after `enriched := classified.ToEnriched()`, add:
```go
enriched.ProjectSlug = c.projectSlug
enriched.Generalizability = classified.Generalizability
```

- [ ] **Step 8: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 9: Wire `--project-slug` flag in CLI**

In `internal/cli/cli.go`, `RunLearn` (line ~146):
Add flag:
```go
projectSlug := fs.String("project-slug", "", "originating project slug")
```

After `learner` construction (line ~182), add:
```go
if *projectSlug != "" {
    learner.SetProjectSlug(*projectSlug)
}
```

In `runCorrect` (line ~1064):
Add flag:
```go
projectSlug := fs.String("project-slug", "", "originating project slug")
```

After `corrector` construction (line ~1091), add:
```go
if *projectSlug != "" {
    corrector.SetProjectSlug(*projectSlug)
}
```

- [ ] **Step 10: Run full test suite**

Run: `targ test`
Expected: All tests pass.

- [ ] **Step 11: Commit**

```bash
git add internal/learn/learn.go internal/learn/learn_test.go \
       internal/correct/correct.go internal/correct/correct_test.go \
       internal/cli/cli.go
git commit -m "feat(cli): thread project-slug flag through learn and correct pipelines"
```

---

### Task 8: Hooks pass project slug

**Files:**
- Modify: `hooks/stop.sh`
- Modify: `hooks/user-prompt-submit.sh`

- [ ] **Step 1: Add project slug to `stop.sh`**

In `hooks/stop.sh`, before the `FLUSH_ARGS` line (line ~42), add:

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
```

Add to the `FLUSH_ARGS` array (after `--data-dir`, line ~42):

```bash
FLUSH_ARGS=(--data-dir "$ENGRAM_DATA" --project-slug "$PROJECT_SLUG")
```

- [ ] **Step 2: Add project slug to `user-prompt-submit.sh`**

In `hooks/user-prompt-submit.sh`, before the `CORRECT_ARGS` line (line ~44), add:

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
```

Add to the `CORRECT_ARGS` array:

```bash
CORRECT_ARGS=(correct --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA" --project-slug "$PROJECT_SLUG")
```

- [ ] **Step 3: Verify hooks are syntactically valid**

Run: `bash -n hooks/stop.sh && bash -n hooks/user-prompt-submit.sh && echo "OK"`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add hooks/stop.sh hooks/user-prompt-submit.sh
git commit -m "feat(hooks): pass project-slug to flush and correct commands"
```

---

### Task 9: Classify path strips transcript context via DI

**Files:**
- Modify: `internal/transcript/transcript.go`
- Test: `internal/transcript/transcript_test.go`
- Modify: `internal/cli/cli.go:1057-1104` (runCorrect wiring)

- [ ] **Step 1: Write failing test — ReadRecent applies StripFunc when set**

In `internal/transcript/transcript_test.go`, add a test that:
- Creates a `Reader` with a `fakeReadFile` returning multi-line content (simulated JSONL)
- Sets a `StripFunc` that transforms the lines (e.g., a simple fake that uppercases or filters)
- Calls `ReadRecent` and asserts the output was transformed by the strip function

```go
func TestReadRecent_AppliesStripFunc(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    content := "line1\nline2\nline3"
    reader := transcript.New(fakeReadFile([]byte(content), nil))
    reader.SetStrip(func(lines []string) []string {
        return lines[:1] // keep only first line
    })

    result, err := reader.ReadRecent("/some/path", 2000)
    g.Expect(err).NotTo(HaveOccurred())

    if err != nil {
        return
    }

    g.Expect(result).To(Equal("line1"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `SetStrip` method doesn't exist.

- [ ] **Step 3: Write failing test — ReadRecent works without StripFunc (backward compat)**

Add a test that creates a `Reader` WITHOUT calling `SetStrip` and verifies `ReadRecent` returns raw content as before.

This test should already exist (existing tests don't set a strip func). Verify by running existing tests — they should still pass after implementation.

- [ ] **Step 4: Implement StripFunc on Reader**

In `internal/transcript/transcript.go`:

Add `StripFunc` type and field:

```go
// StripFunc transforms raw transcript lines into cleaned conversation text.
type StripFunc func(lines []string) []string
```

Add field to `Reader` struct:
```go
type Reader struct {
    readFile FileReader
    strip    StripFunc
}
```

Add setter:
```go
// SetStrip sets an optional function to clean transcript lines.
func (r *Reader) SetStrip(fn StripFunc) {
    r.strip = fn
}
```

Modify `ReadRecent` — after reading content and before the tail trim, apply strip if set:

```go
func (r *Reader) ReadRecent(transcriptPath string, maxTokens int) (string, error) {
    if transcriptPath == "" {
        return "", nil
    }

    content, err := r.readFile(transcriptPath)
    if err != nil {
        return "", nil //nolint:nilerr // non-fatal: transcript context is advisory
    }

    text := string(content)

    if r.strip != nil {
        lines := strings.Split(text, "\n")
        stripped := r.strip(lines)
        text = strings.Join(stripped, "\n")
    }

    if len(text) <= maxTokens {
        return text, nil
    }

    // Take the tail (most recent portion)
    return text[len(text)-maxTokens:], nil
}
```

Add `"strings"` to imports if not already present.

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS — both new test and existing tests.

- [ ] **Step 6: Wire strip func in CLI**

In `internal/cli/cli.go`, `runCorrect` (line ~1080), change:

```go
reader := transcript.New(os.ReadFile)
```

To:

```go
reader := transcript.New(os.ReadFile)
reader.SetStrip(sessionctx.Strip)
```

Add import for `sessionctx` if not already present. The package is `internal/context` — check the existing import alias used in `cli.go` (likely imported as `sessionctx` based on the `runIncrementalLearn` function at line 1119).

- [ ] **Step 7: Run full test suite**

Run: `targ check-full`
Expected: All tests pass, lint clean.

- [ ] **Step 8: Commit**

```bash
git add internal/transcript/transcript.go internal/transcript/transcript_test.go internal/cli/cli.go
git commit -m "feat(transcript): strip transcript context via DI in classify path"
```

---

### Task 10: Final integration verification

**Files:** None (verification only)

- [ ] **Step 1: Run full quality checks**

Run: `targ check-full`
Expected: All tests pass, lint clean, coverage thresholds met.

- [ ] **Step 2: Verify end-to-end data flow manually**

Check that the binary builds and the new flags are accepted:

```bash
targ build
~/.claude/engram/bin/engram flush --data-dir /tmp/test-data --project-slug test-proj --help 2>&1 || true
~/.claude/engram/bin/engram correct --message "test" --data-dir /tmp/test-data --project-slug test-proj --help 2>&1 || true
```

Expected: No "unknown flag" errors.

- [ ] **Step 3: Verify backward compatibility with existing TOML files**

```bash
head -5 ~/.claude/engram/data/memories/tdd-test-first-discipline.toml
```

Expected: Existing files without `project_slug` or `generalizability` fields are still readable (TOML `omitempty` means missing fields default to zero values).
