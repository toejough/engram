# Generalizability Gating for Memory Extraction

## Problem

Engram has accumulated 3,140 memories with no quality filter beyond the extraction prompt's abstract instruction to "reject ephemeral context." Many memories are project-specific debugging observations ("pipeline produced flat faces," "verify medial dot connections," "S6 is validated") that will never be useful again — especially across projects, since memories are global.

This erodes user trust. When users see junk memories being recorded, they lose confidence in the system regardless of retrieval quality. Trust is the product.

### Root causes

1. **The extraction prompt lacks a concrete litmus test.** It says "reject ephemeral context that does not generalize across sessions" but gives no specific evaluation criteria. Meanwhile, the "EXTRACT" section lists attractive categories ("architectural decisions!", "working solutions!") that encourage Haiku to find *something*.

2. **No generalizability scoring.** The extractor makes a binary keep/reject decision. Haiku is better at ranking than binary classification — a structured 1-5 scale with anchored examples would let it express uncertainty.

3. **No project provenance.** Memories don't record which project they came from. This prevents future surfacing logic from penalizing cross-project matches for narrow memories.

4. **The classify path sends raw JSONL as transcript context.** `transcript.ReadRecent` reads raw file bytes without stripping. The extract path uses `sessionctx.Strip` for clean text, but the classify path bypasses it. This is both a quality bug and a duplication risk — two paths should share one cleaning function.

## Design

### 1. Add `project_slug` and `generalizability` to MemoryRecord

**File:** `internal/memory/record.go`

Add two fields to `MemoryRecord`:

```go
ProjectSlug      string `toml:"project_slug,omitempty"`
Generalizability int    `toml:"generalizability,omitempty"`
```

- `project_slug`: the originating project, derived from `$PWD` in hooks (same slug format as recall: path with `/` replaced by `-`)
- `generalizability`: Haiku's self-rated 1-5 score (0 for pre-existing memories without a score)

### 2. Sharpen extraction prompts

**Files:** `internal/extract/extract.go`, `internal/classify/classify.go`

Add to both system prompts:

**Concrete litmus test** (replaces the vague "reject ephemeral context" line):
> Before extracting any learning, ask: "Would a developer working on a different task in a different project, weeks from now, benefit from knowing this?" If the answer is "probably not," either reject it or score it low.

**Concrete reject-examples** (added to the quality gate):
> - debugging observations about specific data/geometry/state (e.g., "pipeline produced flat faces," "normals are inverted on mesh B")
> - task/validation status updates (e.g., "S6 is validated," "step 3 is complete")
> - project-specific variable/file names without a generalizable principle

**Generalizability field** added to the output JSON schema:

```
"generalizability": "Integer 1-5:
  1 = only meaningful in this exact session (e.g., 'S6 is validated')
  2 = useful in this project but narrow scope (e.g., 'this specific pipeline config causes flat faces')
  3 = useful across this project (e.g., 'use targ for builds in this repo')
  4 = useful across similar projects (e.g., 'wrap errors with context in Go')
  5 = universal principle (e.g., 'never amend pushed commits')"
```

### 3. Hard gate at extraction time

**Files:** `internal/learn/learn.go`, `internal/correct/correct.go`

After the LLM returns candidates, filter: `generalizability < 2` → drop, never store. This catches the obvious junk ("S6 is validated" → generalizability 1) without being aggressive enough to lose useful-but-narrow content.

The gate applies in both extraction paths:
- `learn.go`: filter the `[]CandidateLearning` slice before dedup/write
- `correct.go`: if the single `ClassifiedMemory` scores < 2, return early without writing

Log dropped candidates to stderr for observability: `[engram] dropped (generalizability=1): "title here"`

### 4. Hooks pass project slug

**Files:** `hooks/stop.sh`, `hooks/user-prompt-submit.sh`

Both hooks derive the project slug from `$PWD`:

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
```

Note: this produces a leading `-` (e.g., `-Users-joe-repos-foo`). This matches the existing slug format used by `recall` — confirm consistency before implementing.

- `stop.sh`: pass `--project-slug "$PROJECT_SLUG"` to `engram flush`
- `user-prompt-submit.sh`: pass `--project-slug "$PROJECT_SLUG"` to `engram correct`

**CLI wiring:**
- `RunLearn` (`cli.go`): accept `--project-slug` flag, thread through to `Learner` and `tomlwriter`
- `RunCorrect` (`cli.go`): accept `--project-slug` flag, thread through to `Corrector` and `tomlwriter`
- `tomlwriter`: set `ProjectSlug` on `MemoryRecord` when writing

### 5. Classify path strips transcript context

**File:** `internal/transcript/transcript.go`

`ReadRecent` currently returns raw file bytes. It needs to strip through `sessionctx.Strip` so the classify path sends clean text to Haiku, matching what the extract path does.

**Approach:** Add a `StripFunc` field to `transcript.Reader` (DI, consistent with the pattern in `learn.IncrementalLearner`). When set, `ReadRecent` splits the raw content into lines (newline-delimited JSONL), passes the full `[]string` to the strip function (which returns cleaned lines), and rejoins with newlines. When nil, behavior is unchanged (backward compatible for tests). Note: `sessionctx.Strip` operates on the full `[]string` slice, not line-by-line — it expects JSONL input and returns `"ROLE: text"` lines.

Wire in `cli.go` at the `RunCorrect` callsite:

```go
reader := transcript.New(os.ReadFile)
reader.SetStrip(sessionctx.Strip)
```

### 6. CandidateLearning and ClassifiedMemory types

**Files:** `internal/memory/memory.go` (or wherever these types live)

Add `Generalizability int` field to both `CandidateLearning` and `ClassifiedMemory` structs so the LLM response parsing populates the score.

## What this does NOT cover (see filed issues)

- **Project-scoped surfacing**: penalizing low-generalizability memories when surfaced in a different project than they originated from. The data (`project_slug` + `generalizability`) will be stored and ready for this.
- **Time-based decay**: adding a time-since-last-relevance factor so memories decay and eventually get pruned if never confirmed useful.
- **Retroactive cleanup**: reviewing/scoring/pruning the existing 3,140 memories.

## Testing approach

- Unit tests for the generalizability filter in `learn` and `correct`
- Unit tests for the prompt changes (verify the output schema includes `generalizability`)
- Unit tests for `transcript.Reader` with `StripFunc` set
- Integration-style tests for the CLI flag threading (project-slug flows through to written TOML)
- Existing tests must continue passing (backward compat for memories without `generalizability` field)
