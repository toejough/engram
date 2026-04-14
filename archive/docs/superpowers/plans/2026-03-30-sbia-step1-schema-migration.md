# SBIA Step 1: Schema + Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the keyword/principle memory schema with SBIA fields, migrate existing memories, and delete all packages being rewritten in later steps.

**Architecture:** Coordinated cutover — update core types, delete packages that depend on old fields and will be rewritten later, fix surviving packages, run migration. After this step: memories are SBIA-only, `engram show`/`engram recall` work, extraction/surfacing/maintain are temporarily removed (rebuilt in Steps 2-5).

**Tech Stack:** Go, TOML, Anthropic API (Sonnet for migration)

**Source spec:** `docs/superpowers/specs/2026-03-29-sbia-feedback-model-design.md` — see "Final SBIA Memory Schema" section for target schema, "Migration" section for tier handling.

---

### Task 1: Update Memory Structs

**Files:**
- Modify: `internal/memory/record.go`
- Modify: `internal/memory/memory.go`
- Modify: `internal/memory/record_test.go`
- Modify: `internal/memory/memory_test.go`
- Modify: `internal/memory/readmodifywrite_test.go`
- Modify: `internal/memory/maintenance_test.go`

- [ ] **Step 1: Write failing test for new MemoryRecord**

```go
func TestMemoryRecord_SBIAFields(t *testing.T) {
	t.Parallel()

	record := memory.MemoryRecord{
		Situation:    "When running tests in a targ project",
		Behavior:     "Invoking go test directly",
		Impact:       "Bypasses coverage thresholds",
		Action:       "Use targ test instead",
		ProjectScoped: true,
		ProjectSlug:  "engram",
		CreatedAt:    "2026-03-30T00:00:00Z",
		UpdatedAt:    "2026-03-30T00:00:00Z",
		SurfacedCount:    0,
		FollowedCount:    0,
		NotFollowedCount: 0,
		IrrelevantCount:  0,
	}

	g := NewGomegaWithT(t)
	g.Expect(record.Situation).To(Equal("When running tests in a targ project"))
	g.Expect(record.TotalEvaluations()).To(Equal(0))
}
```

Run: `targ test`
Expected: FAIL — fields don't exist yet

- [ ] **Step 2: Rewrite MemoryRecord in record.go**

Replace the entire MemoryRecord struct and supporting types with:

```go
// PendingEvaluation records a surfacing event awaiting evaluation at stop hook.
type PendingEvaluation struct {
	SurfacedAt  string `toml:"surfaced_at"`
	UserPrompt  string `toml:"user_prompt"`
	SessionID   string `toml:"session_id"`
	ProjectSlug string `toml:"project_slug"`
}

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
//
//nolint:revive // "memory.MemoryRecord" stutter is intentional for clarity. See #353.
// ALL code that touches memory TOML must use this struct to prevent field loss.
type MemoryRecord struct {
	// Content fields (SBIA).
	Situation string `toml:"situation"`
	Behavior  string `toml:"behavior"`
	Impact    string `toml:"impact"`
	Action    string `toml:"action"`

	// Scope.
	ProjectScoped bool   `toml:"project_scoped"`
	ProjectSlug   string `toml:"project_slug,omitempty"`

	// Timestamps.
	CreatedAt string `toml:"created_at"`
	UpdatedAt string `toml:"updated_at"`

	// Tracking — feedback counters.
	SurfacedCount    int `toml:"surfaced_count"`
	FollowedCount    int `toml:"followed_count"`
	NotFollowedCount int `toml:"not_followed_count"`
	IrrelevantCount  int `toml:"irrelevant_count"`

	// Pending evaluations (written at surface, consumed at stop).
	PendingEvaluations []PendingEvaluation `toml:"pending_evaluations,omitempty"`
}

// ToStored converts a MemoryRecord to a Stored for in-memory use.
func (r *MemoryRecord) ToStored(filePath string) *Stored {
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &Stored{
		Situation:        r.Situation,
		Behavior:         r.Behavior,
		Impact:           r.Impact,
		Action:           r.Action,
		ProjectScoped:    r.ProjectScoped,
		ProjectSlug:      r.ProjectSlug,
		SurfacedCount:    r.SurfacedCount,
		FollowedCount:    r.FollowedCount,
		NotFollowedCount: r.NotFollowedCount,
		IrrelevantCount:  r.IrrelevantCount,
		UpdatedAt:        updatedAt,
		FilePath:         filePath,
	}
}

// TotalEvaluations returns the sum of all evaluation counters.
func (r *MemoryRecord) TotalEvaluations() int {
	return r.FollowedCount + r.NotFollowedCount + r.IrrelevantCount
}
```

Remove: `AbsorbedRecord`, `EvaluationCounters`, `MaintenanceAction` types (no longer used in SBIA model).

- [ ] **Step 3: Rewrite Stored and SearchText in memory.go**

Replace the Stored struct, SearchText, and TotalFeedback:

```go
// Stored represents a memory read back from a TOML file on disk.
type Stored struct {
	Situation        string
	Behavior         string
	Impact           string
	Action           string
	ProjectScoped    bool
	ProjectSlug      string
	SurfacedCount    int
	FollowedCount    int
	NotFollowedCount int
	IrrelevantCount  int
	UpdatedAt        time.Time
	FilePath         string
}

// SearchText returns a concatenation of all SBIA fields for BM25 retrieval scoring.
func (s *Stored) SearchText() string {
	parts := make([]string, 0, searchTextCapacity)

	if s.Situation != "" {
		parts = append(parts, s.Situation)
	}

	if s.Behavior != "" {
		parts = append(parts, s.Behavior)
	}

	if s.Impact != "" {
		parts = append(parts, s.Impact)
	}

	if s.Action != "" {
		parts = append(parts, s.Action)
	}

	return strings.Join(parts, " ")
}

// TotalEvaluations returns the sum of all evaluation counters.
func (s *Stored) TotalEvaluations() int {
	return s.FollowedCount + s.NotFollowedCount + s.IrrelevantCount
}
```

Remove: `CandidateLearning`, `ClassifiedMemory`, `Enriched`, `PatternMatch`, `ToEnriched()` — these pipeline intermediaries are replaced in Step 2.

Keep `searchTextCapacity = 4` (was 5).

- [ ] **Step 4: Update memory package tests**

Update all tests in `memory/` to use SBIA fields. Key changes:
- `record_test.go`: Replace field references (`Title` → test new SBIA fields)
- `memory_test.go`: Update SearchText test to verify situation+behavior+impact+action concatenation
- `readmodifywrite_test.go`: Update TOML fixtures to SBIA format
- `maintenance_test.go`: Remove (MaintenanceAction type deleted)

Run: `targ test`
Expected: memory package tests pass, other packages fail to compile

- [ ] **Step 5: Commit memory package changes**

```bash
git add internal/memory/
git commit -m "refactor(memory): replace keyword/principle schema with SBIA fields

Situation, Behavior, Impact, Action replace title, content, principle,
anti_pattern, rationale, keywords, concepts. Tracking simplified to
four counters (surfaced, followed, not_followed, irrelevant). Added
PendingEvaluation for surface-time evaluation tracking.

Removes: AbsorbedRecord, EvaluationCounters, MaintenanceAction,
CandidateLearning, ClassifiedMemory, Enriched, PatternMatch types.

AI-Used: [claude]"
```

---

### Task 2: Delete Packages Being Rewritten

These packages depend on old fields and will be completely rewritten in Steps 2-5. Delete them now rather than fixing compilation errors in code we're about to replace.

**Packages to delete:**

| Package | Reason | Rebuilt in |
|---------|--------|-----------|
| `internal/extract/` | Old batch extraction → Sonnet SBIA extraction | Step 2 |
| `internal/learn/` | Old learn pipeline → eliminated | Step 2 |
| `internal/classify/` | Old Haiku classifier → new detect+extract | Step 2 |
| `internal/dedup/` | Old keyword overlap → Sonnet decision tree | Step 2 |
| `internal/signal/` | Old consolidation/apply → unified proposals | Step 5 |
| `internal/adapt/` | Old 5-dimension analysis → Sonnet adapt | Step 5 |
| `internal/contradict/` | Old contradiction detection → SBIA decision tree | Step 2 |
| `internal/crossref/` | Old cross-reference extraction → not in SBIA | Step 2 |
| `internal/surfacinglog/` | Eliminated — pending evaluations replace it | Step 4 |
| `internal/keyword/` | Keywords dropped | Step 2 |
| `internal/review/` | Old review pipeline → not in SBIA | — |
| `internal/policy/` | Old policy lifecycle → unified proposals + change_history | Step 5 |
| `internal/frecency/` | Old quality scoring → effectiveness-only | Step 3 |
| `internal/effectiveness/` | Old effectiveness calc → derived metrics | Step 3 |

- [ ] **Step 1: Delete packages**

```bash
rm -rf internal/extract internal/learn internal/classify internal/dedup \
       internal/signal internal/adapt internal/contradict internal/crossref \
       internal/surfacinglog internal/keyword internal/review internal/policy \
       internal/frecency internal/effectiveness
```

- [ ] **Step 2: Remove imports from CLI**

In `internal/cli/cli.go`, remove all imports for deleted packages. Remove or stub out CLI subcommands that depend on them:
- `learn`, `flush` commands → remove
- `signal` commands → remove
- `adapt` commands → remove
- `feedback` command → remove (LLM self-report dropped)
- `apply-proposal --action` old form → remove (new form added in Step 5)

Keep: `correct`, `surface`, `show`, `recall`, `maintain` (stubbed), `record`, `migrate-slugs`, `export`

For commands being stubbed, return an error message:
```go
fmt.Fprintln(os.Stderr, "engram maintain: temporarily disabled during SBIA migration (Step 5)")
```

- [ ] **Step 3: Fix CLI compilation errors**

Work through remaining compilation errors in `internal/cli/`:
- Remove references to deleted types (`policy.Policy`, `signal.Cluster`, etc.)
- Remove functions that only served deleted features
- Keep the CLI functional for `show`, `recall`, `surface`, `correct`

- [ ] **Step 4: Verify build**

Run: `targ build`
Expected: compiles (some commands stubbed)

- [ ] **Step 5: Commit deletions**

```bash
git add -A
git commit -m "refactor(engram): delete packages being rewritten in SBIA Steps 2-5

Removes extract, learn, classify, dedup, signal, adapt, contradict,
crossref, surfacinglog, keyword, review, policy, frecency, effectiveness.
These packages depend on old schema fields and will be rebuilt with SBIA
semantics in subsequent steps.

CLI commands learn, flush, signal, adapt, feedback removed. maintain
stubbed pending Step 5 rebuild.

AI-Used: [claude]"
```

---

### Task 3: Fix Surviving Packages

**Files to modify (field renames in surviving code):**

| File | Changes |
|------|---------|
| `internal/tomlwriter/tomlwriter.go` | Write takes `*MemoryRecord` directly (Enriched deleted). Remove slug generation from FilenameSummary. |
| `internal/tomlwriter/tomlwriter_test.go` | Update test fixtures to SBIA fields |
| `internal/surface/surface.go` | Update Stored field references. Remove old frecency/ranking imports. |
| `internal/surface/surface_test.go` | Update test fixtures |
| `internal/surface/suppress_p4f.go` | Update field references |
| `internal/surface/budget_test.go` | Update field references |
| `internal/surface/cold_start_budget_test.go` | Update field references |
| `internal/surface/p4e_test.go` | Update field references |
| `internal/surface/p4f_test.go` | Update field references |
| `internal/correct/correct.go` | Remove consolidation, simplify to write MemoryRecord. Stub — actual SBIA extraction added in Step 2. |
| `internal/correct/correct_test.go` | Update to new interface |
| `internal/retrieve/retrieve.go` | No changes needed (reads MemoryRecord from TOML, struct change propagates) |
| `internal/retrieve/retrieve_test.go` | Update test TOML fixtures |
| `internal/render/render.go` | Display SBIA fields instead of principle |
| `internal/render/render_test.go` | Update expected output |
| `internal/track/recorder.go` | Update Stored field references |
| `internal/track/recorder_test.go` | Update fixtures |
| `internal/maintain/` | Remove old maintain code. Stub `Run()` to return empty. Rebuilt in Step 5. |
| `internal/cli/show.go` | Display SBIA fields |
| `internal/cli/record.go` | Update field references |

- [ ] **Step 1: Update tomlwriter**

The tomlwriter currently takes `*memory.Enriched` (deleted). Change `Write` to accept `*memory.MemoryRecord` directly. The slug comes from a new `slug` parameter since `FilenameSummary` no longer exists.

Update signature: `Write(record *memory.MemoryRecord, dataDir, slug string) (string, error)`

- [ ] **Step 2: Update tomlwriter tests**

Replace all test `Enriched` structs with `MemoryRecord` using SBIA fields.

- [ ] **Step 3: Update surface package**

Replace all `Stored` field references:
- `.Principle` → `.Action`
- `.Title` → remove or use `.Situation` for display
- `.Keywords` → remove
- `.Concepts` → remove
- `.Generalizability` → `.ProjectScoped` (bool, not int)
- `.Confidence` / `.Tier` → remove
- `.ContradictedCount` → remove
- `.IgnoredCount` → remove
- `.IrrelevantQueries` → remove
- `.LastSurfacedAt` → remove
- `.TotalFeedback()` → `.TotalEvaluations()`

Remove frecency scorer dependency (deleted). Use BM25 score directly for ranking (simplified until Step 3 rebuilds surfacing).

- [ ] **Step 4: Update surface tests**

Update all test Stored structs to use SBIA fields.

- [ ] **Step 5: Update correct package**

Simplify `Corrector` — remove consolidation flow, classifier dependency. Stub `Run()` to return empty string with a log message. Step 2 rebuilds this with Sonnet extraction.

- [ ] **Step 6: Update render package**

Change display format from `principle` to SBIA fields:
```go
fmt.Fprintf(w, "Situation: %s\n", mem.Situation)
fmt.Fprintf(w, "Behavior: %s\n", mem.Behavior)
fmt.Fprintf(w, "Impact: %s\n", mem.Impact)
fmt.Fprintf(w, "Action: %s\n", mem.Action)
```

- [ ] **Step 7: Update remaining packages (track, retrieve, maintain, cli)**

Fix all remaining compilation errors using the field mapping above. Maintain package gets stubbed (rebuilt in Step 5).

- [ ] **Step 8: Run full build and test**

Run: `targ check-full`
Expected: all tests pass, lint clean, build succeeds

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "refactor(engram): update surviving packages for SBIA schema

Field renames across surface, correct, render, track, retrieve,
tomlwriter, maintain, and CLI packages. tomlwriter takes MemoryRecord
directly. Surface uses BM25 on SBIA text. Render displays all four
SBIA fields. correct and maintain stubbed pending Steps 2 and 5.

AI-Used: [claude]"
```

---

### Task 4: Write Migration Command

**Files:**
- Create: `internal/cli/migrate_sbia.go`
- Create: `internal/cli/migrate_sbia_test.go`

The migration command reads existing TOML files, uses Sonnet to convert tier A memories to SBIA fields, and archives tier B/C memories.

- [ ] **Step 1: Write failing test for migration**

```go
func TestMigrateSBIA_TierA_ConvertedToSBIA(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Create a tier A memory in old format
	oldTOML := `title = "Use targ"
content = "Always use targ build system"
principle = "Use targ test, targ check-full"
anti_pattern = "Running go test directly"
rationale = "targ wraps build/test"
keywords = ["targ", "test"]
confidence = "A"
created_at = "2026-03-01T00:00:00Z"
updated_at = "2026-03-01T00:00:00Z"
surfaced_count = 5
followed_count = 3
contradicted_count = 0
ignored_count = 1
irrelevant_count = 1
last_surfaced_at = ""
`
	// ... test that migration produces SBIA fields
}
```

- [ ] **Step 2: Implement migration command**

`engram migrate-sbia` does:
1. List all `.toml` files in `{dataDir}/memories/`
2. Parse each as old-format struct (need a temporary `LegacyMemoryRecord` for parsing)
3. For tier A: call Sonnet to convert principle/anti_pattern/rationale → situation/behavior/impact/action. Carry forward `project_slug`, map `generalizability ≤ 2` → `project_scoped = true`. Carry forward `surfaced_count`, `followed_count`. Map `contradicted_count + ignored_count` → `not_followed_count`. Keep `irrelevant_count`.
4. For tier B/C: move file to `{dataDir}/archive/`
5. For conversion failures: move to `{dataDir}/archive/`
6. Write converted memories as new-format TOML

The `LegacyMemoryRecord` struct is defined locally in the migration file — it only exists for parsing old TOML, not used elsewhere.

- [ ] **Step 3: Write Sonnet conversion prompt**

The prompt should extract SBIA dimensions from old fields:
- `situation` from: context in `content` + `title` + `keywords`
- `behavior` from: `anti_pattern` (what not to do)
- `impact` from: `rationale` (why it matters)
- `action` from: `principle` (what to do instead)

- [ ] **Step 4: Test migration end-to-end**

Run: `targ test`
Expected: migration tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/cli/migrate_sbia.go internal/cli/migrate_sbia_test.go
git commit -m "feat(engram): add migrate-sbia command for one-time SBIA conversion

Tier A memories converted via Sonnet to SBIA fields. Tier B/C archived.
Carries forward surfaced_count, followed_count, maps contradicted+ignored
to not_followed_count. Maps generalizability ≤ 2 to project_scoped.

AI-Used: [claude]"
```

---

### Task 5: Run Migration and Verify End-to-End

- [ ] **Step 1: Run migration on live data**

```bash
engram migrate-sbia --data-dir ~/.claude/engram
```

Verify: check a few converted TOML files manually. Verify B/C memories are in archive.

- [ ] **Step 2: Verify engram show works**

```bash
engram show --name <converted-memory-name>
```

Expected: displays Situation, Behavior, Impact, Action fields.

- [ ] **Step 3: Verify engram recall works**

```bash
engram recall --query "targ"
```

Expected: finds converted memories via BM25 on SBIA text.

- [ ] **Step 4: Verify engram surface works (basic)**

Expected: surfaces memories with SBIA fields displayed. Ranking may be simplified (no frecency) but functional.

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "feat(engram): complete SBIA schema migration (Step 1 of 5)

All memories converted to SBIA format. Tier A converted via Sonnet,
tier B/C archived. Old packages deleted. show, recall, surface functional
with SBIA fields. correct and maintain stubbed pending Steps 2 and 5.

AI-Used: [claude]"
```

---

## File Map Summary

| Action | Files | Count |
|--------|-------|-------|
| **Modify** | `internal/memory/record.go`, `memory.go`, tests | 6 |
| **Delete** | 14 packages (~50 files) | ~50 |
| **Modify** | Surviving packages (surface, correct, render, track, retrieve, tomlwriter, maintain, cli) | ~25 |
| **Create** | `internal/cli/migrate_sbia.go`, test | 2 |

## What Works After Step 1

| Command | Status |
|---------|--------|
| `engram show` | Works — displays SBIA fields |
| `engram recall` | Works — BM25 on SBIA text |
| `engram surface` | Works (basic) — BM25 ranking, no Haiku gate yet |
| `engram correct` | Stubbed — rebuilt in Step 2 |
| `engram evaluate` | Not yet created — added in Step 4 |
| `engram maintain` | Stubbed — rebuilt in Step 5 |
| `engram learn/flush` | Removed — replaced by correct in Step 2 |
| `engram feedback` | Removed — replaced by evaluate in Step 4 |

## What's Temporarily Broken

- **Creating new memories** — `engram correct` is stubbed. No new memories until Step 2.
- **Maintenance** — `engram maintain` is stubbed. No health analysis until Step 5.
- **Haiku gate** — Not yet added to surface. Added in Step 3.
- **Evaluation** — Not yet created. Added in Step 4.
