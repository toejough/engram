# Memory Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing `internal/signal.Consolidator` with BM25→Haiku semantic clustering to prevent principle loss when specific memories are decayed.

**Architecture:** Three intervention points (BeforeStore, OnIrrelevant, BeforeRemove) on the existing Consolidator, backed by a tiered similarity pipeline (BM25 candidate retrieval → Haiku cluster confirmation → Haiku principle extraction). Extends `internal/signal` with new interfaces and methods; wires into learn, correct, feedback, and maintain pipelines.

**Tech Stack:** Go, BM25 (`internal/bm25`), Claude Haiku API, TOML memory storage, `targ` build system.

**Spec:** `docs/superpowers/specs/2026-03-23-memory-consolidation-design.md`

---

## Key Type Decision

**Use `*memory.MemoryRecord` throughout the new consolidation pipeline**, not `*memory.Stored`. Per `record.go`: "ALL code that touches memory TOML must use this struct to prevent field loss" (#353). `Stored` is a lightweight type missing `Confidence`, `ProjectSlug`, `EnforcementLevel`, `ContentHash`, `Absorbed`, and other fields the consolidation pipeline needs. The existing `Consolidate()`/`Plan()` methods continue using `*memory.Stored`; the new semantic pipeline is a parallel path that operates on the canonical type.

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `internal/signal/consolidate_types.go` | Action, ScoredCandidate, ConfirmedCluster, OnIrrelevantInput, RefinementContext types |
| `internal/signal/consolidate_semantic.go` | Semantic clustering: BM25 → Haiku pipeline, BeforeStore/OnIrrelevant/BeforeRemove methods |
| `internal/signal/consolidate_semantic_test.go` | Tests for semantic clustering and intervention points |
| `internal/signal/consolidate_transfer.go` | Counter transfer and field construction for consolidated memories |
| `internal/signal/consolidate_transfer_test.go` | Tests for counter transfer logic |
| `internal/signal/consolidate_archive.go` | Archiver with injected filesystem operations |
| `internal/signal/consolidate_archive_test.go` | Tests for archive operations |
| `internal/signal/bm25_adapter.go` | `BM25ScorerAdapter` wrapping `bm25.Scorer` to satisfy `signal.Scorer` interface |
| `internal/signal/bm25_adapter_test.go` | Tests for BM25 adapter |
| `internal/signal/llm_confirm.go` | Haiku implementations of Confirmer and Extractor interfaces |
| `internal/signal/llm_confirm_test.go` | Tests for LLM prompt construction and response parsing |
| `internal/signal/consolidate_migrate.go` | MigrationRunner for `engram migrate-scores` subcommand |
| `internal/signal/consolidate_migrate_test.go` | Tests for migration pipeline |
| `internal/cli/migrate.go` | CLI wiring for `migrate-scores` subcommand |
| `internal/cli/migrate_test.go` | Tests for migrate CLI wiring |

### Modified Files

| File | Change |
|------|--------|
| `internal/signal/consolidate.go` | Add scorer/confirmer/extractor/archiver fields + With* options |
| `internal/learn/learn.go` | Add consolidator field + call BeforeStore before writeCandidate |
| `internal/learn/learn_test.go` | Test consolidation integration in learn pipeline |
| `internal/correct/correct.go` | Add consolidator field + call BeforeStore after writer.Write |
| `internal/correct/correct_test.go` | Test consolidation integration in correct pipeline |
| `internal/cli/feedback.go` | Add surfacing context flags + call OnIrrelevant on irrelevant feedback |
| `internal/cli/feedback_test.go` | Test consolidation integration in feedback pipeline |
| `internal/cli/targets.go` | Add FeedbackArgs surfacing fields + MigrateScoresArgs + register migrate-scores |
| `internal/cli/targets_test.go` | Test new args structs and flag builders |
| `internal/maintain/maintain.go` | Add consolidator field + call BeforeRemove before noise removal |
| `internal/maintain/maintain_test.go` | Test consolidation guard in maintain pipeline |

---

## Task 1: Type Definitions and Consolidator Options

**Files:**
- Create: `internal/signal/consolidate_types.go`
- Modify: `internal/signal/consolidate.go`

- [ ] **Step 1: Write the types file**

File: `internal/signal/consolidate_types.go`

```go
package signal

import (
	"context"

	"engram/internal/memory"
)

// ActionType describes the consolidator's decision at an intervention point.
type ActionType int

const (
	// StoreAsIs means no cluster was found; store the memory normally.
	StoreAsIs ActionType = iota
	// Consolidated means a cluster was found and merged into a generalized memory.
	Consolidated
	// RefineKeywords means no cluster was found; keyword refinement is suggested.
	RefineKeywords
	// ProceedWithRemoval means no cluster was found; removal can proceed.
	ProceedWithRemoval
)

// Action is the result of a consolidation intervention point.
type Action struct {
	Type             ActionType
	ConsolidatedMem  *memory.MemoryRecord
	Archived         []string
	RefinementContext *RefinementContext
}

// RefinementContext carries surfacing context for keyword refinement (#346).
type RefinementContext struct {
	Memory          *memory.MemoryRecord
	SurfacingQuery  string
	MatchedKeywords []string
	ToolName        string
	ToolInput       string
}

// OnIrrelevantInput carries the memory and surfacing context for irrelevant feedback.
type OnIrrelevantInput struct {
	Memory         *memory.MemoryRecord
	SurfacingQuery string
	ToolName       string
	ToolInput      string
}

// ScoredCandidate is a memory with its BM25 similarity score.
type ScoredCandidate struct {
	Memory *memory.MemoryRecord
	Score  float64
}

// ConfirmedCluster is a group of memories confirmed by LLM to share a principle.
type ConfirmedCluster struct {
	Members   []*memory.MemoryRecord
	Principle string
}

// Scorer retrieves candidate memories similar to a query memory.
type Scorer interface {
	FindSimilar(ctx context.Context, query *memory.MemoryRecord, exclude []string) ([]ScoredCandidate, error)
}

// Confirmer asks an LLM whether candidate memories share a principle.
type Confirmer interface {
	ConfirmClusters(ctx context.Context, query *memory.MemoryRecord, candidates []ScoredCandidate) ([]ConfirmedCluster, error)
}

// Extractor creates a generalized memory from a confirmed cluster.
type Extractor interface {
	ExtractPrinciple(ctx context.Context, cluster ConfirmedCluster) (*memory.MemoryRecord, error)
}

// Archiver moves memory files to an archive directory.
type Archiver interface {
	Archive(sourcePath string) error
}
```

- [ ] **Step 2: Verify it compiles**

Run: `targ build`
Expected: PASS

- [ ] **Step 3: Add new fields and options to existing Consolidator**

Modify `internal/signal/consolidate.go`. Add four fields to the `Consolidator` struct (after existing fields):

```go
scorer    Scorer
confirmer Confirmer
extractor Extractor
archiver  Archiver
```

Add four new option functions after existing `With*` functions:

```go
// WithScorer sets the BM25 candidate scorer for semantic clustering.
func WithScorer(s Scorer) ConsolidatorOption {
	return func(c *Consolidator) { c.scorer = s }
}

// WithConfirmer sets the LLM cluster confirmer for semantic clustering.
func WithConfirmer(cf Confirmer) ConsolidatorOption {
	return func(c *Consolidator) { c.confirmer = cf }
}

// WithExtractor sets the LLM principle extractor for semantic clustering.
func WithExtractor(e Extractor) ConsolidatorOption {
	return func(c *Consolidator) { c.extractor = e }
}

// WithArchiver sets the archiver for consolidated memory originals.
func WithArchiver(a Archiver) ConsolidatorOption {
	return func(c *Consolidator) { c.archiver = a }
}
```

- [ ] **Step 4: Verify it compiles**

Run: `targ build`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(signal): add semantic clustering types and consolidator options

New types (Action, ScoredCandidate, ConfirmedCluster, RefinementContext,
OnIrrelevantInput) and interfaces (Scorer, Confirmer, Extractor, Archiver)
for BM25→Haiku semantic clustering. Uses *memory.MemoryRecord throughout
per #353 canonical struct mandate.
```

---

## Task 2: Counter Transfer Logic

**Files:**
- Create: `internal/signal/consolidate_transfer.go`
- Create: `internal/signal/consolidate_transfer_test.go`

- [ ] **Step 1: Write failing tests for counter transfer**

File: `internal/signal/consolidate_transfer_test.go` (package `signal_test`)

Test cases using gomega assertions and `t.Parallel()`:
- `TestTransferFields_SumsFollowedCount`: 3 records with followed_count 2, 5, 3 → consolidated has 10
- `TestTransferFields_SumsContradictedCount`: similar for contradicted_count
- `TestTransferFields_ResetsIrrelevantCount`: originals have irrelevant_count > 0 → consolidated has 0
- `TestTransferFields_ResetsIgnoredCount`: → 0
- `TestTransferFields_ResetsSurfacedCount`: → 0
- `TestTransferFields_MaxEnforcementLevel`: originals at "advisory"/"emphasized_advisory"/"reminder" → consolidated gets "reminder"
- `TestTransferFields_ConfidenceAlwaysB`: regardless of originals, consolidated is "B"
- `TestTransferFields_EmptyProjectSlug`: consolidated always gets empty project_slug
- `TestTransferFields_AbsorbedRecords`: one `memory.AbsorbedRecord` per original with:
  - `From` = original's file path
  - `SurfacedCount` = original's surfaced_count
  - `Evaluations` = `memory.EvaluationCounters{Followed: N, Contradicted: N, Ignored: N}` (NOT a bare int)
  - `ContentHash` = original's content_hash
  - `MergedAt` = RFC3339 string (NOT `time.Time`)

Each test creates `[]*memory.MemoryRecord` originals and a `*memory.MemoryRecord` base (simulating Extractor output), calls `TransferFields(base, originals, now)`, and asserts the result.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — `TransferFields` undefined

- [ ] **Step 3: Implement TransferFields**

File: `internal/signal/consolidate_transfer.go`

```go
package signal

import (
	"time"

	"engram/internal/memory"
)

const consolidatedConfidence = "B"

// TransferFields applies counter transfer rules from originals onto a base
// consolidated memory. Mutates base in place. Per spec: sum followed/contradicted,
// reset irrelevant/ignored/surfaced, set confidence B, clear project_slug,
// take max enforcement level.
func TransferFields(base *memory.MemoryRecord, originals []*memory.MemoryRecord, now time.Time) {
	var totalFollowed, totalContradicted int

	maxEnforcement := base.EnforcementLevel

	absorbed := make([]memory.AbsorbedRecord, 0, len(originals))

	for _, orig := range originals {
		totalFollowed += orig.FollowedCount
		totalContradicted += orig.ContradictedCount

		if enforcementRank(orig.EnforcementLevel) > enforcementRank(maxEnforcement) {
			maxEnforcement = orig.EnforcementLevel
		}

		absorbed = append(absorbed, memory.AbsorbedRecord{
			From:          orig.SourcePath, // or FilePath equivalent
			SurfacedCount: orig.SurfacedCount,
			Evaluations: memory.EvaluationCounters{
				Followed:     orig.FollowedCount,
				Contradicted: orig.ContradictedCount,
				Ignored:      orig.IgnoredCount,
			},
			ContentHash: orig.ContentHash,
			MergedAt:    now.Format(time.RFC3339),
		})
	}

	base.FollowedCount = totalFollowed
	base.ContradictedCount = totalContradicted
	base.IrrelevantCount = 0
	base.IgnoredCount = 0
	base.SurfacedCount = 0
	base.Confidence = consolidatedConfidence
	base.ProjectSlug = ""
	base.EnforcementLevel = maxEnforcement
	base.Absorbed = append(base.Absorbed, absorbed...)
}

func enforcementRank(level string) int {
	switch level {
	case "reminder":
		return 2
	case "emphasized_advisory":
		return 1
	default:
		return 0
	}
}
```

Note: `MemoryRecord` doesn't have a `FilePath` field — it has `SourcePath`. Check the actual field name for the on-disk path. The caller may need to pass it separately or use a wrapper struct. Resolve during implementation.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(signal): add counter transfer logic for consolidated memories

TransferFields sums followed/contradicted counts, resets irrelevant/
ignored/surfaced, sets confidence B, clears project_slug, takes max
enforcement level. Uses EvaluationCounters and RFC3339 MergedAt per
actual AbsorbedRecord schema.
```

---

## Task 3: Archive Management

**Files:**
- Create: `internal/signal/consolidate_archive.go`
- Create: `internal/signal/consolidate_archive_test.go`

The Archiver implementation must use injected I/O operations per CLAUDE.md: "No function in `internal/` calls `os.*` directly."

- [ ] **Step 1: Write failing tests for FileArchiver**

File: `internal/signal/consolidate_archive_test.go` (package `signal_test`)

Use mock `Renamer` and `DirCreator` interfaces (NOT real filesystem). Test cases:
- `TestFileArchiver_Archive_CallsRenameWithCorrectPaths`: source `memories/foo.toml` → destination `memories/.archive/foo.toml`
- `TestFileArchiver_Archive_CreatesDirFirst`: DirCreator called before Renamer
- `TestFileArchiver_Archive_RenameError`: Renamer returns error → error propagated
- `TestFileArchiver_Archive_DirCreateError`: DirCreator returns error → error propagated

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement FileArchiver with DI**

File: `internal/signal/consolidate_archive.go`

```go
package signal

import (
	"fmt"
	"path/filepath"
)

// Renamer moves a file from one path to another.
type Renamer func(oldpath, newpath string) error

// DirCreator ensures a directory exists.
type DirCreator func(path string) error

// FileArchiver moves memory files to an archive directory using injected I/O.
type FileArchiver struct {
	archiveDir string
	rename     Renamer
	mkdirAll   DirCreator
}

// NewFileArchiver creates a FileArchiver. Wire os.Rename and
// os.MkdirAll (wrapped) at the CLI boundary.
func NewFileArchiver(archiveDir string, rename Renamer, mkdirAll DirCreator) *FileArchiver {
	return &FileArchiver{
		archiveDir: archiveDir,
		rename:     rename,
		mkdirAll:   mkdirAll,
	}
}

// Archive moves a memory file to the archive directory, preserving its name.
func (a *FileArchiver) Archive(sourcePath string) error {
	if err := a.mkdirAll(a.archiveDir); err != nil {
		return fmt.Errorf("creating archive dir: %w", err)
	}

	destPath := filepath.Join(a.archiveDir, filepath.Base(sourcePath))

	if err := a.rename(sourcePath, destPath); err != nil {
		return fmt.Errorf("archiving %s: %w", filepath.Base(sourcePath), err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(signal): add FileArchiver with injected I/O for consolidation

Moves original memories to memories/.archive/ during consolidation.
Uses injected Renamer and DirCreator per DI principle — wire os.*
at CLI boundary.
```

---

## Task 4: BM25 Scorer Adapter

**Files:**
- Create: `internal/signal/bm25_adapter.go`
- Create: `internal/signal/bm25_adapter_test.go`

The `signal.Scorer` interface takes `(ctx, *memory.MemoryRecord, []string)` but `bm25.Scorer.Score()` takes tokenized documents. This task creates an adapter that bridges the gap.

- [ ] **Step 1: Write failing tests**

File: `internal/signal/bm25_adapter_test.go` (package `signal_test`)

Test cases:
- `TestBM25Adapter_FindSimilar_ReturnsTopCandidates`: corpus of 5 memories, query matches 3, returns sorted by score
- `TestBM25Adapter_FindSimilar_ExcludesSlugs`: excluded slugs not in results
- `TestBM25Adapter_FindSimilar_RespectsThreshold`: only candidates above threshold returned
- `TestBM25Adapter_FindSimilar_CapsAtMax`: at most 10 results
- `TestBM25Adapter_FindSimilar_EmptyCorpus`: returns empty slice, no error
- `TestBM25Adapter_FindSimilar_BuildsQueryFromTitlePrincipleKeywords`: verify the query text is constructed from title + principle + keywords (not content)

Use a real `bm25.Scorer` (pure computation, no I/O) with test data.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement BM25ScorerAdapter**

File: `internal/signal/bm25_adapter.go`

```go
package signal

import (
	"context"
	"sort"
	"strings"

	"engram/internal/bm25"
	"engram/internal/memory"
)

const (
	defaultSimilarityThreshold = 0.3
	maxSimilarityCandidates    = 10
)

// MemoryRecordLister loads all MemoryRecords from the data directory.
type MemoryRecordLister interface {
	ListAllRecords(ctx context.Context) ([]*memory.MemoryRecord, error)
}

// BM25ScorerAdapter wraps bm25.Scorer to satisfy the signal.Scorer interface.
// Configured with threshold and max candidates at construction time.
type BM25ScorerAdapter struct {
	lister    MemoryRecordLister
	threshold float64
	maxCandidates int
}

// NewBM25ScorerAdapter creates an adapter. Threshold and maxCandidates are
// configurable for migration dry-run calibration.
func NewBM25ScorerAdapter(
	lister MemoryRecordLister,
	threshold float64,
	maxCandidates int,
) *BM25ScorerAdapter {
	return &BM25ScorerAdapter{
		lister:        lister,
		threshold:     threshold,
		maxCandidates: maxCandidates,
	}
}

// FindSimilar loads the corpus, scores against the query, and returns
// top candidates above threshold.
func (a *BM25ScorerAdapter) FindSimilar(
	ctx context.Context,
	query *memory.MemoryRecord,
	exclude []string,
) ([]ScoredCandidate, error) {
	records, err := a.lister.ListAllRecords(ctx)
	if err != nil {
		return nil, err
	}

	queryText := buildQueryText(query)
	excludeSet := toSet(exclude)

	// Build BM25 corpus from all non-excluded records
	var docs []bm25.Document
	var docRecords []*memory.MemoryRecord

	for _, rec := range records {
		slug := slugFromPath(rec) // extract slug from source_path or content_hash
		if _, excluded := excludeSet[slug]; excluded {
			continue
		}
		if isAlreadyAbsorbed(rec) {
			continue
		}

		docs = append(docs, bm25.Document{
			ID:   slug,
			Text: buildQueryText(rec),
		})
		docRecords = append(docRecords, rec)
	}

	scorer := bm25.NewScorer(docs)
	results := scorer.Score(queryText)

	// Filter and sort
	var candidates []ScoredCandidate
	for i, result := range results {
		if result.Score >= a.threshold && i < len(docRecords) {
			candidates = append(candidates, ScoredCandidate{
				Memory: docRecords[findDocIndex(docs, result.ID)],
				Score:  result.Score,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > a.maxCandidates {
		candidates = candidates[:a.maxCandidates]
	}

	return candidates, nil
}

func buildQueryText(rec *memory.MemoryRecord) string {
	parts := []string{rec.Title, rec.Principle}
	parts = append(parts, rec.Keywords...)
	return strings.Join(parts, " ")
}

func isAlreadyAbsorbed(rec *memory.MemoryRecord) bool {
	return len(rec.Absorbed) > 0
}

func toSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}
```

Note: `bm25.NewScorer` and `bm25.Document` names are approximate — read the actual `internal/bm25` package API before implementing. The adapter should match the real types. Also resolve `slugFromPath` and `findDocIndex` helper functions.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(signal): add BM25ScorerAdapter for memory similarity search

Wraps bm25.Scorer to satisfy signal.Scorer interface. Builds query
text from title + principle + keywords, filters by configurable
threshold, caps at max candidates. Excludes absorbed memories.
```

---

## Task 5: LLM Confirmer and Extractor

**Files:**
- Create: `internal/signal/llm_confirm.go`
- Create: `internal/signal/llm_confirm_test.go`

- [ ] **Step 1: Write failing tests**

File: `internal/signal/llm_confirm_test.go` (package `signal_test`)

Test cases:
- `TestLLMConfirmer_ConfirmsClusters`: mock LLM returns JSON grouping memories → parsed into `[]ConfirmedCluster`
- `TestLLMConfirmer_NoCluster`: mock LLM returns empty clusters → empty slice
- `TestLLMConfirmer_ExcludesContradictions`: mock LLM flags contradictions → those members excluded from cluster
- `TestLLMConfirmer_LLMError`: mock returns error → error propagated
- `TestLLMExtractor_ExtractsPrinciple`: mock LLM returns generalized memory JSON → parsed into `*memory.MemoryRecord`
- `TestLLMExtractor_SetsGeneralizability`: extracted memory has generalizability from LLM response
- `TestLLMExtractor_LLMError`: mock returns error → error propagated

Inject LLM caller as `func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)` — same pattern as `internal/extract/extract.go` and `internal/maintain/maintain.go`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement LLMConfirmer and LLMExtractor**

File: `internal/signal/llm_confirm.go`

Follow the patterns from `internal/extract/extract.go`:

**LLMConfirmer:**
- Struct with injected LLM caller function
- `ConfirmClusters()` builds a prompt with the query memory + candidates' titles/principles
- Prompt asks: group by shared principle, exclude contradictions, return JSON
- Parse response into `[]ConfirmedCluster`, matching members back to `*memory.MemoryRecord` by slug

**LLMExtractor:**
- Struct with injected LLM caller function
- `ExtractPrinciple()` builds a prompt with all cluster members' content
- Prompt asks for: title, principle, anti_pattern, content, keywords, concepts, generalizability
- Parse response JSON into `*memory.MemoryRecord`

Define response structs for reliable JSON parsing.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(signal): add LLM confirmer and extractor for semantic clustering

LLMConfirmer asks Haiku to confirm memory clusters share a principle
and detect contradictions. LLMExtractor creates generalized memories
from confirmed clusters. Both use injected LLM caller for testability.
```

---

## Task 6: Semantic Clustering — Core Logic (findCluster)

**Files:**
- Create: `internal/signal/consolidate_semantic.go`
- Create: `internal/signal/consolidate_semantic_test.go`

- [ ] **Step 1: Write failing tests for findCluster**

File: `internal/signal/consolidate_semantic_test.go` (package `signal_test`)

Test cases:
- `TestFindCluster_NoScorer_ReturnsNil`: consolidator without scorer → nil
- `TestFindCluster_NoCandidates_ReturnsNil`: scorer returns empty → nil
- `TestFindCluster_BelowMinSize_ReturnsNil`: scorer returns 1 candidate (query+1=2, below min 3) → nil
- `TestFindCluster_ConfirmedCluster`: scorer returns 2+ candidates, confirmer groups them → returns cluster
- `TestFindCluster_ConfirmerRejectsAll`: confirmer returns empty → nil
- `TestFindCluster_MultipleClustersSortedSmallestFirst`: confirmer returns 2 clusters → smallest returned first
- `TestFindCluster_ConfirmerError_ReturnsNil`: confirmer fails → nil (graceful degradation, not error)

Use mock Scorer and Confirmer.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement findCluster**

File: `internal/signal/consolidate_semantic.go`

```go
package signal

import (
	"context"
	"sort"

	"engram/internal/memory"
)

const minSemanticClusterSize = 3

// findCluster attempts to find a semantic cluster for the given memory.
// Returns nil if no cluster is found or if the pipeline is unavailable.
func (c *Consolidator) findCluster(
	ctx context.Context,
	mem *memory.MemoryRecord,
	exclude []string,
) *ConfirmedCluster {
	if c.scorer == nil || c.confirmer == nil {
		return nil
	}

	candidates, err := c.scorer.FindSimilar(ctx, mem, exclude)
	if err != nil || len(candidates) < minSemanticClusterSize-1 {
		return nil
	}

	clusters, err := c.confirmer.ConfirmClusters(ctx, mem, candidates)
	if err != nil || len(clusters) == 0 {
		return nil
	}

	// Sort smallest first — protect fragile clusters (per spec)
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Members) < len(clusters[j].Members)
	})

	for i := range clusters {
		if len(clusters[i].Members) >= minSemanticClusterSize {
			return &clusters[i]
		}
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(signal): add findCluster for BM25→Haiku semantic clustering

Core clustering logic: BM25 candidate retrieval, Haiku confirmation,
smallest-first cluster selection. Degrades gracefully when scorer
or confirmer is unavailable.
```

---

## Task 7: consolidateCluster (Shared Execution Logic)

**Files:**
- Modify: `internal/signal/consolidate_semantic.go`
- Modify: `internal/signal/consolidate_semantic_test.go`

This implements the shared consolidation execution that all three intervention points call.

- [ ] **Step 1: Write failing tests for consolidateCluster**

Test cases:
- `TestConsolidateCluster_ExtractsAndTransfers`: cluster with 3 members → extractor called, TransferFields applied, Action{Type: Consolidated} returned
- `TestConsolidateCluster_ArchivesAllMembers`: all cluster members archived
- `TestConsolidateCluster_RewritesGraphLinks`: linkRecomputer called for each archived member
- `TestConsolidateCluster_ExtractorError_ReturnsError`: extractor fails → error returned
- `TestConsolidateCluster_UpdatesExistingConsolidated`: one cluster member has non-empty `Absorbed` → that member is updated (not replaced), other members archived into it
- `TestConsolidateCluster_IDFFiltersKeywords`: after extraction, keywords are IDF-filtered using `keyword.FilterByDocFrequency` on post-archival corpus

Use mock Extractor, Archiver, LinkRecomputer, and keyword filter.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement consolidateCluster**

```go
// consolidateCluster executes consolidation for a confirmed cluster.
// Shared by BeforeStore, OnIrrelevant, and BeforeRemove.
func (c *Consolidator) consolidateCluster(
	ctx context.Context,
	cluster *ConfirmedCluster,
) (Action, error) {
	// Check if any member is already consolidated (has Absorbed records)
	existing := findExistingConsolidated(cluster.Members)

	var consolidated *memory.MemoryRecord
	var err error

	if existing != nil {
		// Update existing consolidated memory
		consolidated, err = c.updateExistingConsolidated(ctx, existing, cluster)
	} else {
		// Create new consolidated memory
		consolidated, err = c.extractor.ExtractPrinciple(ctx, *cluster)
	}

	if err != nil {
		return Action{}, fmt.Errorf("consolidating cluster: %w", err)
	}

	// Apply counter transfer from originals
	originals := excludeExisting(cluster.Members, existing)
	now := time.Now()
	TransferFields(consolidated, originals, now)

	// IDF-filter keywords on post-archival corpus
	// (pass through keyword.FilterByDocFrequency if available)

	// Archive originals (skip the existing consolidated if updating)
	archived := make([]string, 0, len(originals))
	for _, orig := range originals {
		if c.archiver != nil {
			if archErr := c.archiver.Archive(orig.SourcePath); archErr != nil {
				c.logStderrf("[engram] archive failed for %q: %v\n", orig.Title, archErr)
			}
		}

		archived = append(archived, slugFromRecord(orig))

		// Rewrite graph links
		if c.linkRecomputer != nil {
			_ = c.linkRecomputer.RecomputeAfterMerge(consolidated.SourcePath, orig.SourcePath)
		}
	}

	return Action{
		Type:            Consolidated,
		ConsolidatedMem: consolidated,
		Archived:        archived,
	}, nil
}

// findExistingConsolidated returns the first member that is already
// a consolidated memory (non-empty Absorbed), or nil.
func findExistingConsolidated(members []*memory.MemoryRecord) *memory.MemoryRecord {
	for _, mem := range members {
		if len(mem.Absorbed) > 0 {
			return mem
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(signal): add consolidateCluster shared execution logic

Handles both create-new and update-existing paths. Extracts principle
via LLM, transfers counters, archives originals, rewrites graph links.
Used by all three intervention points.
```

---

## Task 8: BeforeStore Intervention Point

**Files:**
- Modify: `internal/signal/consolidate_semantic.go`
- Modify: `internal/signal/consolidate_semantic_test.go`

- [ ] **Step 1: Write failing tests for BeforeStore**

Test cases:
- `TestBeforeStore_NoCluster_ReturnsStoreAsIs`: no cluster found → `Action{Type: StoreAsIs}`
- `TestBeforeStore_ClusterFound_ReturnsConsolidated`: cluster found → `Action{Type: Consolidated}`
- `TestBeforeStore_ConsolidateError_ReturnsStoreAsIs`: consolidateCluster fails → graceful degradation to StoreAsIs

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement BeforeStore**

```go
// BeforeStore checks if a candidate memory belongs to an existing cluster.
// Called by extract pipeline before writing a new memory to disk.
func (c *Consolidator) BeforeStore(ctx context.Context, candidate *memory.MemoryRecord) (Action, error) {
	cluster := c.findCluster(ctx, candidate, nil)
	if cluster == nil {
		return Action{Type: StoreAsIs}, nil
	}

	action, err := c.consolidateCluster(ctx, cluster)
	if err != nil {
		c.logStderrf("[engram] consolidation failed, storing as-is: %v\n", err)
		return Action{Type: StoreAsIs}, nil
	}

	return action, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(signal): add BeforeStore intervention point

Checks if a new candidate belongs to an existing semantic cluster
before storing. Degrades to StoreAsIs on any error.
```

---

## Task 9: OnIrrelevant Intervention Point

**Files:**
- Modify: `internal/signal/consolidate_semantic.go`
- Modify: `internal/signal/consolidate_semantic_test.go`

- [ ] **Step 1: Write failing tests for OnIrrelevant**

Test cases:
- `TestOnIrrelevant_ClusterFound_ReturnsConsolidated`: cluster exists → `Consolidated`
- `TestOnIrrelevant_NoCluster_ReturnsRefineKeywords`: no cluster → `RefineKeywords` with populated `RefinementContext`
- `TestOnIrrelevant_RefinementContextPopulated`: verify SurfacingQuery, ToolName, ToolInput carried through
- `TestOnIrrelevant_PartialContext`: some surfacing fields empty → still returns RefineKeywords with empty strings

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement OnIrrelevant**

```go
// OnIrrelevant checks if an irrelevantly-surfaced memory belongs to a cluster.
// Called by feedback pipeline after recording irrelevant feedback.
func (c *Consolidator) OnIrrelevant(ctx context.Context, input OnIrrelevantInput) (Action, error) {
	cluster := c.findCluster(ctx, input.Memory, nil)
	if cluster == nil {
		return Action{
			Type: RefineKeywords,
			RefinementContext: &RefinementContext{
				Memory:         input.Memory,
				SurfacingQuery: input.SurfacingQuery,
				ToolName:       input.ToolName,
				ToolInput:      input.ToolInput,
			},
		}, nil
	}

	action, err := c.consolidateCluster(ctx, cluster)
	if err != nil {
		c.logStderrf("[engram] consolidation failed on irrelevant: %v\n", err)
		return Action{
			Type: RefineKeywords,
			RefinementContext: &RefinementContext{
				Memory:         input.Memory,
				SurfacingQuery: input.SurfacingQuery,
				ToolName:       input.ToolName,
				ToolInput:      input.ToolInput,
			},
		}, nil
	}

	return action, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(signal): add OnIrrelevant intervention point

On irrelevant feedback: check for cluster first (consolidate), fall
back to RefineKeywords with populated RefinementContext for #346.
```

---

## Task 10: BeforeRemove Intervention Point

**Files:**
- Modify: `internal/signal/consolidate_semantic.go`
- Modify: `internal/signal/consolidate_semantic_test.go`

- [ ] **Step 1: Write failing tests for BeforeRemove**

Test cases:
- `TestBeforeRemove_ClusterFound_ReturnsConsolidated`: cluster exists → `Consolidated`
- `TestBeforeRemove_NoCluster_ReturnsProceedWithRemoval`: no cluster → `ProceedWithRemoval`
- `TestBeforeRemove_ScorerUnavailable_ReturnsProceedWithRemoval`: no scorer → proceed (with warning logged)

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement BeforeRemove**

```go
// BeforeRemove checks if a memory slated for removal belongs to a cluster.
// Called by maintain pipeline before generating a removal proposal.
func (c *Consolidator) BeforeRemove(ctx context.Context, mem *memory.MemoryRecord) (Action, error) {
	cluster := c.findCluster(ctx, mem, nil)
	if cluster == nil {
		if c.scorer == nil {
			c.logStderrf("[engram] warning: BeforeRemove called without scorer, proceeding with removal\n")
		}
		return Action{Type: ProceedWithRemoval}, nil
	}

	action, err := c.consolidateCluster(ctx, cluster)
	if err != nil {
		c.logStderrf("[engram] consolidation failed before remove: %v\n", err)
		return Action{Type: ProceedWithRemoval}, nil
	}

	return action, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(signal): add BeforeRemove intervention point

Safety net: no memory is removed if it has unconsolidated siblings.
Degrades to ProceedWithRemoval on error.
```

---

## Task 11: Wire BeforeStore into Learn Pipeline

**Files:**
- Modify: `internal/learn/learn.go`
- Modify: `internal/learn/learn_test.go` or appropriate export_test.go

- [ ] **Step 1: Write failing test**

Test that when a consolidator is set and BeforeStore returns Consolidated:
- The candidate is NOT written to disk (writeCandidate skipped)
- The consolidated memory IS written

Test that when BeforeStore returns StoreAsIs:
- The candidate IS written to disk normally

Note: The existing `Learner` uses a positional constructor `New()`. Add `SetConsolidator` as a setter (same pattern as `SetProjectSlug` on `Corrector`).

The test needs a `*signal.Consolidator` configured with mock Scorer/Confirmer/Extractor. Construct it with `signal.NewConsolidator(signal.WithScorer(...), ...)`.

To convert between `extract.CandidateLearning` and `*memory.MemoryRecord`, a conversion helper will be needed. Check what conversion functions already exist in the codebase (e.g., `ToEnriched()` on CandidateLearning).

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement wiring**

Add to `Learner` struct:
```go
consolidator interface {
	BeforeStore(ctx context.Context, candidate *memory.MemoryRecord) (signal.Action, error)
}
```

Note: Use a narrow interface (just `BeforeStore`) rather than depending on `*signal.Consolidator` directly. This follows the existing codebase pattern of depending on behavior, not concrete types.

Add setter:
```go
func (l *Learner) SetConsolidator(c interface{ BeforeStore(context.Context, *memory.MemoryRecord) (signal.Action, error) }) {
	l.consolidator = c
}
```

Modify the loop at `learn.go:110`:
```go
for _, candidate := range surviving {
	if l.consolidator != nil {
		record := candidateToMemoryRecord(candidate)
		action, err := l.consolidator.BeforeStore(ctx, record)
		if err == nil && action.Type == signal.Consolidated {
			// Write consolidated memory via fileWriter
			// Skip writing individual candidate
			continue
		}
	}

	filePath, err := l.writeCandidate(candidate, now)
	// ... existing code
}
```

Implement `candidateToMemoryRecord` conversion helper.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(learn): wire consolidation BeforeStore into learn pipeline

New memories are checked against existing clusters before storage.
If cluster found: generalized principle created instead of specific
candidate. Uses narrow interface dependency.
```

---

## Task 12: Wire BeforeStore into Correct Pipeline

**Files:**
- Modify: `internal/correct/correct.go`
- Modify: `internal/correct/correct_test.go`

- [ ] **Step 1: Write failing test**

Same pattern as Task 11 but for the correct pipeline. Note: `Corrector` already has a `SetProjectSlug` setter, so `SetConsolidator` follows the same pattern.

The insertion point is before `c.writer.Write()` at `correct.go:68`. If BeforeStore returns Consolidated, write the consolidated memory instead and return its path.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement wiring**

Add narrow-interface consolidator field and setter. Before `writer.Write`:
```go
if c.consolidator != nil {
	record := enrichedToMemoryRecord(enriched)
	action, err := c.consolidator.BeforeStore(ctx, record)
	if err == nil && action.Type == signal.Consolidated {
		// Write consolidated memory, return its path
		filePath, writeErr := c.writer.Write(consolidatedToEnriched(action.ConsolidatedMem), c.dataDir)
		if writeErr != nil {
			return "", fmt.Errorf("correct: write consolidated: %w", writeErr)
		}
		return c.renderer.Render(classified, filePath), nil
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(correct): wire consolidation BeforeStore into correct pipeline

User-corrected memories checked against existing clusters before
storage, same as learn pipeline.
```

---

## Task 13: Wire OnIrrelevant into Feedback Pipeline

**Files:**
- Modify: `internal/cli/targets.go`
- Modify: `internal/cli/targets_test.go`
- Modify: `internal/cli/feedback.go`
- Modify: `internal/cli/feedback_test.go`

- [ ] **Step 1: Add surfacing context flags to FeedbackArgs**

In `internal/cli/targets.go`, add to `FeedbackArgs`:

```go
SurfacingQuery string `targ:"flag,name=surfacing-query,desc=query that caused memory to surface"`
ToolName       string `targ:"flag,name=tool-name,desc=tool name if surfaced during tool use"`
ToolInput      string `targ:"flag,name=tool-input,desc=tool input if surfaced during tool use"`
```

Update `FeedbackFlags()` to include these new flags.

Note: `feedback.go` uses `flag.NewFlagSet` internally. The new flags must be added to BOTH the `FeedbackArgs` struct (for targ dispatch) AND the internal `FlagSet` in `runFeedback`.

- [ ] **Step 2: Write failing test for feedback consolidation**

Test that when `--irrelevant` is passed and a consolidator is provided:
- Consolidator returns `Consolidated` → "Consolidated cluster" message to stdout
- Consolidator returns `RefineKeywords` → warning logged to stderr (waiting for #346)
- Consolidator is nil → existing behavior unchanged (no error)

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 4: Implement OnIrrelevant call**

After `writeFeedbackTOML` in `feedback.go`, when `*irrelevant` is true:

```go
if *irrelevant && consolidator != nil {
	record, loadErr := readFeedbackTOML(memPath, slug) // already loaded above
	input := signal.OnIrrelevantInput{
		Memory:         record,
		SurfacingQuery: *surfacingQuery,
		ToolName:       *toolName,
		ToolInput:      *toolInput,
	}
	action, consErr := consolidator.OnIrrelevant(ctx, input)
	if consErr == nil && action.Type == signal.Consolidated {
		fmt.Fprintf(stdout, "[engram] Consolidated cluster into %q\n", action.ConsolidatedMem.Title)
	} else if consErr == nil && action.Type == signal.RefineKeywords {
		fmt.Fprintf(stderr, "[engram] No cluster found; keyword refinement pending (#346)\n")
	}
}
```

Note: The consolidator needs to be injected into `runFeedback`. Check how other CLI commands receive dependencies (likely via function parameters or a context struct). The `RunFeedback` function in the CLI layer constructs the consolidator from API token and data dir, then passes it in. When no API token is available, consolidator is nil and the check is skipped.

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 7: Commit**

```
feat(cli): wire consolidation OnIrrelevant into feedback pipeline

Irrelevant feedback now triggers consolidation check. If cluster
found: consolidate. If not: populate RefinementContext (logged as
warning until #346 ships). Surfacing context passed via new
--surfacing-query, --tool-name, --tool-input flags.
```

---

## Task 14: Wire BeforeRemove into Maintain Pipeline

**Files:**
- Modify: `internal/maintain/maintain.go`
- Modify: `internal/maintain/maintain_test.go`

- [ ] **Step 1: Write failing test**

Test that when the Generator has a consolidator and a Noise-quadrant memory:
- Consolidator returns `Consolidated` → no removal proposal generated
- Consolidator returns `ProceedWithRemoval` → normal removal proposal generated
- Consolidator is nil → normal removal proposal (existing behavior)

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement wiring**

The `handleNoise` method currently takes only `classifiedMem` and returns `(Proposal, bool)`. It has no `ctx` or access to `*memory.MemoryRecord`. The consolidation check should be inserted in the **caller** of `handleNoise` — the `generateOne` or `Generate` method — which has access to `ctx` and can load the full `*memory.MemoryRecord` from disk.

Add to `Generator`:
```go
consolidator interface {
	BeforeRemove(ctx context.Context, mem *memory.MemoryRecord) (signal.Action, error)
}
memLoader func(path string) (*memory.MemoryRecord, error)
```

Add option:
```go
func WithConsolidator(c interface{ ... }, loader func(string) (*memory.MemoryRecord, error)) Option {
	...
}
```

In `generateOne` (or wherever `handleNoise` is called), before the `Noise` case:
```go
case review.Noise:
	if g.consolidator != nil && g.memLoader != nil {
		mem, err := g.memLoader(classifiedMem.Name)
		if err == nil {
			action, consErr := g.consolidator.BeforeRemove(ctx, mem)
			if consErr == nil && action.Type == signal.Consolidated {
				// Skip removal — cluster was consolidated
				return Proposal{}, false
			}
		}
	}
	return g.handleNoise(classifiedMem)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(maintain): wire consolidation BeforeRemove into maintain pipeline

No memory is removed if it has unconsolidated siblings. BeforeRemove
check inserted before Noise-quadrant removal proposals.
```

---

## Task 15: Migration Command — Scoring

**Files:**
- Create: `internal/signal/consolidate_migrate.go`
- Create: `internal/signal/consolidate_migrate_test.go`

- [ ] **Step 1: Write failing tests for batch scoring**

Test cases:
- `TestMigrationScorer_ScoresUnscoredMemories`: memories with generalizability=0 get scored
- `TestMigrationScorer_SkipsAlreadyScored`: memories with generalizability>0 untouched
- `TestMigrationScorer_BatchesCorrectly`: 25 memories batched into groups of ~20
- `TestMigrationScorer_WritesScoresBack`: scored memories written back via MemoryWriter

Use mock LLM caller and mock MemoryRecordLister/MemoryWriter.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement MigrationRunner scoring**

```go
type MigrationRunner struct {
	lister       MemoryRecordLister
	scorer       func(ctx context.Context, memories []*memory.MemoryRecord) ([]int, error)
	writer       MemoryWriter
	consolidator *Consolidator
	stderr       io.Writer
}
```

`ScoreUnscored(ctx) (int, error)` — loads all, filters generalizability==0, batch scores via Haiku, writes scores back.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(signal): add migration scoring for unscored memories

MigrationRunner batch-scores existing memories with generalizability=0
via Haiku, writing scores back to TOML files.
```

---

## Task 16: Migration Command — Batch Consolidation

**Files:**
- Modify: `internal/signal/consolidate_migrate.go`
- Modify: `internal/signal/consolidate_migrate_test.go`

- [ ] **Step 1: Write failing tests for batch consolidation**

Test cases:
- `TestMigrationConsolidate_DryRun`: finds clusters, outputs proposals, writes nothing
- `TestMigrationConsolidate_Apply`: creates consolidated memories, archives originals
- `TestMigrationConsolidate_Idempotent`: second run finds no new clusters (consolidated memories have non-empty Absorbed, excluded from clustering)
- `TestMigrationConsolidate_ReportsStats`: outputs scored count, cluster count, unclusterable count
- `TestMigrationConsolidate_DeduplicatesClusters`: memory in 2 clusters → assigned to smallest, larger cluster still valid without it

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL

- [ ] **Step 3: Implement batch consolidation**

`ConsolidateBatch(ctx, dryRun bool) (*MigrationResult, error)`:
1. Load all memories
2. Track assigned slugs in a `map[string]bool`
3. For each unassigned memory: run `findCluster(ctx, mem, assigned)`
4. If cluster found: add all member slugs to assigned set
5. Sort clusters smallest-first for execution order
6. If dry-run: format and output proposals
7. If apply: execute `consolidateCluster` for each

This ensures no memory appears in multiple clusters (smallest-first deduplication).

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(signal): add batch consolidation for migration command

Scans full corpus for semantic clusters. Deduplicates overlapping
clusters (smallest-first). Supports dry-run and apply modes.
Idempotent via absorbed-record exclusion.
```

---

## Task 17: CLI Wiring for migrate-scores

**Files:**
- Create: `internal/cli/migrate.go`
- Create: `internal/cli/migrate_test.go`
- Modify: `internal/cli/targets.go`
- Modify: `internal/cli/targets_test.go`

- [ ] **Step 1: Add MigrateScoresArgs to targets.go**

```go
type MigrateScoresArgs struct {
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Apply       bool   `targ:"flag,name=apply,desc=apply consolidations instead of dry-run"`
	APIToken    string `targ:"flag,name=api-token,env=ENGRAM_API_TOKEN,desc=Anthropic API token"`
	ProjectSlug string `targ:"flag,name=project-slug,desc=originating project slug"`
}
```

Register in `BuildTargets()`.

- [ ] **Step 2: Write failing test for RunMigrateScores**

Test that the CLI function:
- Constructs MigrationRunner with correct dependencies
- Calls ScoreUnscored then ConsolidateBatch
- Respects --apply flag (dry-run by default)
- Outputs results to stdout

- [ ] **Step 3: Run test to verify it fails**

Run: `targ test`
Expected: FAIL

- [ ] **Step 4: Implement RunMigrateScores**

Follow the pattern of `RunLearn` / `RunFeedback`:
- Parse flags + apply defaults (`applyDataDirDefault`, etc.)
- Construct HTTP client, LLM caller
- Construct MigrationRunner with dependencies
- Call ScoreUnscored + ConsolidateBatch
- Output results

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 7: Commit**

```
feat(cli): add migrate-scores subcommand

Scores unscored memories and consolidates clusters. Dry-run by
default, --apply to execute. Closes #360.
```

---

## Task 18: Update Hooks for Surfacing Context

**Files:**
- Modify: `hooks/post-tool-use.sh`
- Modify: `hooks/stop.sh`

The surfacing context for `engram feedback` calls originates from the LLM agent (Claude), not from hooks. Hooks call `engram surface` — the LLM agent reads the surfaced memories and later calls `engram feedback` as a Bash command. The hooks need to **store** the surfacing context so the agent can include it in feedback calls.

Two approaches:
1. **Surface command outputs context**: `engram surface` already outputs which memories were surfaced. The agent has access to the tool name/input from its own context. The `--surfacing-query` can be passed by the agent from its current context.
2. **Hooks write context to a temp file**: The hook writes `$TOOL_NAME`, `$TOOL_INPUT` to a known location that the feedback skill can read.

- [ ] **Step 1: Check how the agent currently calls feedback**

Read the hooks and skills that invoke `engram feedback` to understand the current data flow. The agent (via Bash tool) calls `engram feedback --name <slug> --relevant|--irrelevant`. The tool name and input are available in the hook environment but not passed through to the agent's feedback call.

- [ ] **Step 2: Update the memory-triage skill to pass context**

The `skills/memory-triage/SKILL.md` guides the agent on feedback commands. Update it to include `--surfacing-query`, `--tool-name`, `--tool-input` when available. The agent has access to tool context from its conversation — it knows which tool was being used when a memory surfaced.

- [ ] **Step 3: Update hook feedback invocations**

If hooks call `engram feedback` directly (check each hook), add the surfacing context flags from the hook's environment variables.

- [ ] **Step 4: Test manually**

Run feedback with new flags: `engram feedback --name test --irrelevant --surfacing-query "test query"`
Expected: No error, feedback recorded.

- [ ] **Step 5: Commit**

```
feat(hooks): pass surfacing context to feedback for consolidation

Updated feedback invocations to include --surfacing-query,
--tool-name, --tool-input when available, enabling the
consolidation pipeline to populate RefinementContext.
```

---

## Task 19: End-to-End Verification

- [ ] **Step 1: Rebuild binary**

Run: `targ build`
Expected: PASS

- [ ] **Step 2: Run full test suite**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 3: Manual smoke test — migration dry-run**

Run: `~/.claude/engram/bin/engram migrate-scores`
Expected: Outputs scored count + proposed clusters without writing.
Use this output to calibrate the BM25 threshold (spec says 0.3 is starting point).

- [ ] **Step 4: Manual smoke test — feedback consolidation**

Run: `~/.claude/engram/bin/engram feedback --name <test-memory> --irrelevant`
Expected: Either consolidation message or "keyword refinement pending" warning.

- [ ] **Step 5: Final commit if any cleanup needed**

```
chore: final cleanup for memory consolidation (#368, #360, #346)
```
