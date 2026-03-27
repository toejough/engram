# Consolidation Apply Handler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire existing consolidation building blocks (LLMExtractor, TransferFields, FileArchiver) into the `apply-proposal` dispatcher so `engram apply-proposal --action consolidate` works end-to-end.

**Architecture:** Add a `consolidate` case to the `Applier.Apply()` dispatcher in `signal/apply.go`. The handler loads member memories as `MemoryRecord`, builds a `ConfirmedCluster`, calls `ExtractPrinciple` via LLM, transfers counters, writes the consolidated memory, and archives originals. CLI wiring in `signal.go` constructs the needed dependencies (extractor, archiver, record loader).

**Tech Stack:** Go, TOML, Anthropic API (via existing `makeAnthropicCaller`)

**Spec:** `docs/superpowers/specs/2026-03-25-consolidation-apply-design.md`

---

### Task 1: Store member paths in consolidation proposal details

**Files:**
- Modify: `internal/cli/cli.go:505-509` (consolidateDetails struct)
- Modify: `internal/cli/cli.go:297-325` (emit logic)
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Write the failing test**

Add a test that verifies consolidation proposals include member paths in details JSON.

```go
func TestRunMaintain_ConsolidationProposalIncludesMemberPaths(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Set up a maintain run that produces consolidation plans.
	// Verify the output JSON has details.members as objects with path+title fields.
	// Parse the proposal JSON and check details.members[0].path is set.
	var stdout bytes.Buffer
	// ... (use existing RunMaintain test patterns from cli_test.go)

	var proposals []maintain.Proposal
	err := json.Unmarshal(stdout.Bytes(), &proposals)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	// Find the consolidate proposal
	var found bool
	for _, p := range proposals {
		if p.Action == "consolidate" {
			found = true
			var details struct {
				Members []struct {
					Path  string `json:"path"`
					Title string `json:"title"`
				} `json:"members"`
			}
			unmarshalErr := json.Unmarshal(p.Details, &details)
			g.Expect(unmarshalErr).NotTo(gomega.HaveOccurred())
			if unmarshalErr != nil { return }
			g.Expect(details.Members).NotTo(gomega.BeEmpty())
			g.Expect(details.Members[0].Path).NotTo(gomega.BeEmpty())
		}
	}
	g.Expect(found).To(gomega.BeTrue())
}
```

Note: Look at existing `TestRunMaintain_*` tests in `cli_test.go` for the exact test setup pattern (temp dirs, stub memories, etc.). Match that pattern exactly.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunMaintain_ConsolidationProposalIncludesMemberPaths -v`
Expected: FAIL — current `members` field is `[]string`, not `[]object`

- [ ] **Step 3: Update consolidateDetails struct and emit logic**

In `internal/cli/cli.go`:

```go
type consolidateMember struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

//nolint:tagliatelle // DES-23 specifies snake_case JSON field names.
type consolidateDetails struct {
	Members        []consolidateMember `json:"members"`
	SharedKeywords []string            `json:"shared_keywords"`
	Confidence     float64             `json:"confidence"`
}
```

Update the emit loop (~line 298-324) to build `[]consolidateMember` instead of `[]string`:

```go
members := make([]consolidateMember, 0, len(plans[idx].Absorbed)+1)
members = append(members, consolidateMember{
	Path:  plans[idx].Survivor,
	Title: titleOrPath(memoryMap, plans[idx].Survivor),
})

for _, absorbed := range plans[idx].Absorbed {
	members = append(members, consolidateMember{
		Path:  absorbed,
		Title: titleOrPath(memoryMap, absorbed),
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestRunMaintain_ConsolidationProposalIncludesMemberPaths -v`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: All passing. Fix any compilation errors from the struct change in other tests.

- [ ] **Step 6: Commit**

```
git add internal/cli/cli.go internal/cli/cli_test.go
```
Message: `feat: store member paths in consolidation proposal details (#373)`

---

### Task 2: Add readRecord helper

**Files:**
- Create: `internal/cli/record.go`
- Test: `internal/cli/record_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestReadRecord_LoadsMemoryRecord(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	content := `title = "Test Memory"
principle = "Test principle"
keywords = ["kw1", "kw2"]
surfaced_count = 5
source_path = "stale/path.toml"
`
	err := os.WriteFile(path, []byte(content), 0o600)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	record, readErr := readRecord(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())
	if readErr != nil { return }

	g.Expect(record).NotTo(gomega.BeNil())
	if record == nil { return }

	g.Expect(record.Title).To(gomega.Equal("Test Memory"))
	g.Expect(record.Principle).To(gomega.Equal("Test principle"))
	g.Expect(record.Keywords).To(gomega.Equal([]string{"kw1", "kw2"}))
	g.Expect(record.SurfacedCount).To(gomega.Equal(5))
	// SourcePath must be overwritten with actual path, not stale on-disk value
	g.Expect(record.SourcePath).To(gomega.Equal(path))
}

func TestReadRecord_NonexistentFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	_, err := readRecord("/nonexistent/test.toml")
	g.Expect(err).To(gomega.HaveOccurred())
}
```

Note: This test is in `package cli` (whitebox) since `readRecord` is unexported. Use the test package pattern matching `signal.go` tests.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestReadRecord -v`
Expected: FAIL — `readRecord` undefined

- [ ] **Step 3: Implement readRecord**

Create `internal/cli/record.go`:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// readRecord reads a memory TOML file into a *memory.MemoryRecord.
// Always overwrites SourcePath with the given path (on-disk value may be stale).
func readRecord(path string) (*memory.MemoryRecord, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from trusted flag/internal source
	if err != nil {
		return nil, fmt.Errorf("reading record %s: %w", path, err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return nil, fmt.Errorf("decoding record TOML: %w", decodeErr)
	}

	record.SourcePath = path

	return &record, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestReadRecord -v`
Expected: PASS

- [ ] **Step 5: Commit**

```
git add internal/cli/record.go internal/cli/record_test.go
```
Message: `feat: add readRecord helper for loading MemoryRecord from TOML (#373)`

---

### Task 3: Add applyConsolidate handler to Applier

**Files:**
- Modify: `internal/signal/apply.go` (new deps, new handler, new constant)
- Test: `internal/signal/apply_test.go`

- [ ] **Step 1: Write the failing test**

Follow existing test patterns from `apply_test.go`. Tests use stubs for all injected deps.

```go
func TestApply_Consolidate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Stub extractor that returns a known consolidated memory
	extractedMem := &memory.MemoryRecord{
		Title:     "Consolidated principle",
		Principle: "Generalized principle",
		Keywords:  []string{"general"},
	}
	extractor := &stubExtractor{result: extractedMem}

	// Stub archiver
	archiver := &stubArchiver{}

	// Stub record loader — returns different records for each path
	records := map[string]*memory.MemoryRecord{
		"/mem/a.toml": {Title: "Memory A", SourcePath: "/mem/a.toml", FollowedCount: 2},
		"/mem/b.toml": {Title: "Memory B", SourcePath: "/mem/b.toml", FollowedCount: 1},
		"/mem/c.toml": {Title: "Memory C", SourcePath: "/mem/c.toml", FollowedCount: 3},
	}

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithWriteMemory(writer),
		signal.WithExtractor(extractor),
		signal.WithArchiver(archiver),
		signal.WithLoadRecord(func(path string) (*memory.MemoryRecord, error) {
			rec, ok := records[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}
			return rec, nil
		}),
	)

	membersJSON, _ := json.Marshal([]map[string]string{
		{"path": "/mem/a.toml", "title": "Memory A"},
		{"path": "/mem/b.toml", "title": "Memory B"},
		{"path": "/mem/c.toml", "title": "Memory C"},
	})

	action := signal.ApplyAction{
		Action: "consolidate",
		Memory: "/mem/a.toml",
		Fields: map[string]any{
			"members": json.RawMessage(membersJSON),
		},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	g.Expect(result.Success).To(gomega.BeTrue())

	// Extractor was called with all 3 members
	g.Expect(extractor.calledWith).NotTo(gomega.BeNil())
	if extractor.calledWith == nil { return }
	g.Expect(extractor.calledWith.Members).To(gomega.HaveLen(3))

	// Archived non-survivor members (b and c, not a since it's the survivor)
	g.Expect(archiver.archived).To(gomega.ConsistOf("/mem/b.toml", "/mem/c.toml"))
}

func TestApply_ConsolidateNilExtractor(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier()

	action := signal.ApplyAction{
		Action: "consolidate",
		Memory: "/mem/a.toml",
		Fields: map[string]any{"members": json.RawMessage(`[{"path":"/mem/a.toml"}]`)},
	}

	_, err := applier.Apply(context.Background(), action)
	g.Expect(err).To(gomega.HaveOccurred())
}
```

Stubs needed (add to test file):

```go
type stubExtractor struct {
	result     *memory.MemoryRecord
	calledWith *signal.ConfirmedCluster
}

func (s *stubExtractor) ExtractPrinciple(_ context.Context, cluster signal.ConfirmedCluster) (*memory.MemoryRecord, error) {
	s.calledWith = &cluster
	return s.result, nil
}

type stubArchiver struct {
	archived []string
}

func (s *stubArchiver) Archive(path string) error {
	s.archived = append(s.archived, path)
	return nil
}
```

Note: Check if `stubArchiver` already exists in the test file — it may be defined in `consolidate_semantic_test.go`. If so, it's unexported and in a different test file, so you'll need your own or move to a shared test helper.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestApply_Consolidate -v`
Expected: FAIL — `WithExtractor`, `WithArchiver`, `WithLoadRecord`, and `"consolidate"` action not recognized

- [ ] **Step 3: Implement the handler**

In `internal/signal/apply.go`:

Add constant:
```go
actionConsolidate = "consolidate"
```

Add fields to `Applier`:
```go
extractor  Extractor
archiver   Archiver
loadRecord func(string) (*memory.MemoryRecord, error)
```

Add `With*` options:
```go
func WithExtractor(e Extractor) ApplierOption {
	return func(a *Applier) { a.extractor = e }
}

func WithArchiver(arch Archiver) ApplierOption {
	return func(a *Applier) { a.archiver = arch }
}

func WithLoadRecord(fn func(string) (*memory.MemoryRecord, error)) ApplierOption {
	return func(a *Applier) { a.loadRecord = fn }
}
```

Update `Apply` — rename `_` to `ctx`:
```go
func (a *Applier) Apply(ctx context.Context, action ApplyAction) (ApplyResult, error) {
```

Add case in switch:
```go
case actionConsolidate:
	err = a.applyConsolidate(ctx, action)
```

Add handler:
```go
func (a *Applier) applyConsolidate(ctx context.Context, action ApplyAction) error {
	if a.extractor == nil {
		return fmt.Errorf("consolidate: extractor is nil")
	}

	if a.loadRecord == nil {
		return fmt.Errorf("consolidate: loadRecord is nil")
	}

	if a.writeMem == nil {
		return fmt.Errorf("consolidate: memory writer is nil")
	}

	memberPaths, parseErr := parseMemberPaths(action.Fields)
	if parseErr != nil {
		return fmt.Errorf("consolidate: %w", parseErr)
	}

	members := make([]*memory.MemoryRecord, 0, len(memberPaths))
	for _, path := range memberPaths {
		rec, loadErr := a.loadRecord(path)
		if loadErr != nil {
			return fmt.Errorf("consolidate: loading %s: %w", path, loadErr)
		}
		members = append(members, rec)
	}

	cluster := ConfirmedCluster{Members: members}

	consolidated, extractErr := a.extractor.ExtractPrinciple(ctx, cluster)
	if extractErr != nil {
		return fmt.Errorf("consolidate: extracting principle: %w", extractErr)
	}

	TransferFields(consolidated, members, time.Now())

	// Write consolidated memory via MemoryWriter.
	// Convert MemoryRecord to Stored for the existing writer interface.
	stored := recordToStored(consolidated)
	writeErr := a.writeMem.Write(action.Memory, stored)
	if writeErr != nil {
		return fmt.Errorf("consolidate: writing: %w", writeErr)
	}

	// Archive non-survivor originals.
	if a.archiver != nil {
		for _, path := range memberPaths {
			if path == action.Memory {
				continue // survivor gets overwritten, not archived
			}
			if archErr := a.archiver.Archive(path); archErr != nil {
				return fmt.Errorf("consolidate: archiving %s: %w", path, archErr)
			}
		}
	}

	return nil
}
```

Add helpers:
```go
func parseMemberPaths(fields map[string]any) ([]string, error) {
	raw, ok := fields["members"]
	if !ok {
		return nil, fmt.Errorf("missing members field")
	}

	// Handle json.RawMessage or []any
	var membersList []struct {
		Path string `json:"path"`
	}

	switch v := raw.(type) {
	case json.RawMessage:
		if unmarshalErr := json.Unmarshal(v, &membersList); unmarshalErr != nil {
			return nil, fmt.Errorf("parsing members: %w", unmarshalErr)
		}
	case []any:
		for _, item := range v {
			if m, isMap := item.(map[string]any); isMap {
				if p, hasPath := m["path"].(string); hasPath {
					membersList = append(membersList, struct{ Path string `json:"path"` }{Path: p})
				}
			}
		}
	default:
		return nil, fmt.Errorf("unexpected members type: %T", raw)
	}

	paths := make([]string, 0, len(membersList))
	for _, m := range membersList {
		paths = append(paths, m.Path)
	}
	return paths, nil
}

func recordToStored(rec *memory.MemoryRecord) *memory.Stored {
	updatedAt, _ := time.Parse(time.RFC3339, rec.UpdatedAt)
	lastSurfacedAt, _ := time.Parse(time.RFC3339, rec.LastSurfacedAt)
	return &memory.Stored{
		Title:             rec.Title,
		Content:           rec.Content,
		Concepts:          rec.Concepts,
		Keywords:          rec.Keywords,
		AntiPattern:       rec.AntiPattern,
		Principle:         rec.Principle,
		SurfacedCount:     rec.SurfacedCount,
		FollowedCount:     rec.FollowedCount,
		ContradictedCount: rec.ContradictedCount,
		IgnoredCount:      rec.IgnoredCount,
		IrrelevantCount:   rec.IrrelevantCount,
		IrrelevantQueries: rec.IrrelevantQueries,
		UpdatedAt:         updatedAt,
		LastSurfacedAt:    lastSurfacedAt,
		FilePath:          rec.SourcePath,
		Generalizability:  rec.Generalizability,
		ProjectSlug:       rec.ProjectSlug,
	}
}
```

Note: `recordToStored` is needed because the existing `MemoryWriter` interface operates on `*memory.Stored`. Check if the `Stored` struct has additional fields beyond what's listed — compare with `readStoredMemory` in `signal.go:249-264` for the canonical field mapping.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestApply_Consolidate -v`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: All passing

- [ ] **Step 6: Commit**

```
git add internal/signal/apply.go internal/signal/apply_test.go
```
Message: `feat: add applyConsolidate handler to Applier (#373)`

---

### Task 4: Wire consolidation dependencies in CLI

**Files:**
- Modify: `internal/cli/signal.go:267-341` (runApplyProposal)
- Test: `internal/cli/signal_test.go`

- [ ] **Step 1: Write the failing test**

Test that `runApplyProposal` with `--action consolidate` calls through to the apply pipeline. This needs real TOML files on disk and a way to stub the LLM. Check existing `signal_test.go` patterns for how `runApplyProposal` is tested.

Note: The LLM call is hard to stub at the CLI level since `runApplyProposal` constructs the `Applier` internally. Two options:
1. Integration test with a mock HTTP server for the Anthropic API
2. Test at the `Applier` level (already done in Task 3) and only test CLI flag parsing here

Recommend option 2 — test that `--action consolidate` reaches the dispatcher without error when given valid inputs, and that missing dependencies produce clear errors.

```go
func TestRunApplyProposal_ConsolidateRequiresToken(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	// Create a minimal memory TOML
	memPath := filepath.Join(dir, "memories", "test.toml")
	_ = os.MkdirAll(filepath.Dir(memPath), 0o755)
	_ = os.WriteFile(memPath, []byte(`title = "test"`), 0o600)

	membersJSON := fmt.Sprintf(`{"members":[{"path":"%s","title":"test"}]}`, memPath)
	var stdout bytes.Buffer
	err := runApplyProposal([]string{
		"--data-dir", dir,
		"--action", "consolidate",
		"--memory", memPath,
		"--fields", membersJSON,
	}, &stdout)

	// Should fail because no API token is available in test env
	// (extractor will be nil)
	g.Expect(err).To(gomega.HaveOccurred())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunApplyProposal_ConsolidateRequiresToken -v`
Expected: FAIL — `"consolidate"` hits `unsupported action` in the current switch

- [ ] **Step 3: Wire dependencies in runApplyProposal**

In `internal/cli/signal.go`, inside `runApplyProposal`, after constructing the base `Applier`, add consolidation-specific wiring:

```go
// Consolidation-specific dependencies.
var applierOpts []signal.ApplierOption

applierOpts = append(applierOpts,
	signal.WithReadMemory(readStoredMemory),
	signal.WithWriteMemory(newStoredMemoryWriter()),
	signal.WithEnforcementApplier(&funcEnforcementApplier{fn: enforcementFunc}),
)

if *action == "consolidate" {
	token := resolveToken(context.Background())
	if token != "" {
		caller := makeAnthropicCaller(token)
		extractor := signal.NewLLMExtractor(caller)
		applierOpts = append(applierOpts, signal.WithExtractor(extractor))
	}

	archiveDir := filepath.Join(*dataDir, "archive")
	archiver := signal.NewFileArchiver(archiveDir, os.Rename, func(path string) error {
		const dirPerms = 0o750
		return os.MkdirAll(path, dirPerms)
	})
	applierOpts = append(applierOpts, signal.WithArchiver(archiver))
	applierOpts = append(applierOpts, signal.WithLoadRecord(readRecord))
}

applier := signal.NewApplier(applierOpts...)
```

This replaces the current inline `signal.NewApplier(...)` call. The existing options become part of the `applierOpts` slice.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestRunApplyProposal_ConsolidateRequiresToken -v`
Expected: PASS — now reaches consolidate handler and fails on nil extractor (no token)

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: All passing

- [ ] **Step 6: Commit**

```
git add internal/cli/signal.go internal/cli/signal_test.go
```
Message: `feat: wire consolidation dependencies in runApplyProposal (#373)`

---

### Task 5: Update memory-triage skill docs

**Files:**
- Modify: `skills/memory-triage/SKILL.md`

- [ ] **Step 1: Add consolidate command to Executing Decisions section**

In `skills/memory-triage/SKILL.md`, add after the existing commands:

```markdown
- **Consolidate**: `engram apply-proposal --action consolidate --memory <survivor-path> --fields '{"members":[{"path":"path1","title":"title1"},{"path":"path2","title":"title2"}]}'`
  - Requires API token (LLM synthesizes generalized principle). Survivor path gets overwritten with consolidated memory; other members are archived.
```

- [ ] **Step 2: Commit**

```
git add skills/memory-triage/SKILL.md
```
Message: `docs: add consolidate command to memory-triage skill (#373)`

---

### Task 6: End-to-end smoke test

- [ ] **Step 1: Run full test suite**

Run: `targ check-full`
Expected: All passing

- [ ] **Step 2: Manual smoke test**

Run `engram maintain` and check that consolidation proposals (if any) include member paths in the JSON output:

```bash
engram maintain 2>&1 | jq '[.[] | select(.action == "consolidate") | .details.members[0].path]'
```

Expected: Paths (not bare titles) for any consolidation proposals, or empty array if no clusters found.
