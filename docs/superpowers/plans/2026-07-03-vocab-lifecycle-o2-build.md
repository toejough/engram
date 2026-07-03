# O2-lite Vocab Lifecycle Build

**Date:** 2026-07-03  
**Decision source:** docs/design/2026-07-03-vocab-lifecycle-proposals.md (Joe, 2026-07-03)  
**Design decision:** O2-lite, autonomous refit, triggers 40/14 + untagged>8% + hub>25%, immediate riders 1–2  
**Status:** ready for implementation

---

## Global Constraints

All tasks must satisfy these non-negotiables before any task is marked done:

- **TDD red/green/refactor** per task: write a failing test, then the code to pass it, then refactor.
- **Test stack:** imptest (impgen mocks, interactive) + rapid (properties) + gomega assertions.
- **Nilaway/gomega:** after `g.Expect(err).NotTo(HaveOccurred())`, add `if err != nil { return }` before accessing values derived from err. Never use `err.Error()` — use `g.Expect(err).To(MatchError(...))`.
- **Build commands:** `targ test` / `targ check-full` only. Never `go test` or `go vet` directly. Install the binary with `go install ./cmd/engram`.
- **DI everywhere:** no `os.*`, `http.*`, `json.Marshal` I/O in `internal/` logic. Every I/O dep is injected. Nil deps are graceful no-ops.
- **Named constants, no magic numbers:** `refitGrowthMinNotes`, not bare `40`.
- **Descriptive names:** `noteCount`, `untaggedRate`, `lastRefitDoc` — not `n`, `r`, `d`.
- **Errors wrapped:** `fmt.Errorf("vocab trigger: reading centroids: %w", err)`.
- **`t.Parallel()` on every test and subtest.** Each subtest owns its own fixture data.
- **Line length under 120 chars.**
- **Commits per task** with trailer `AI-Used: [claude]` (NEVER Co-Authored-By).
- **Live vault untouched** except by the shipped normal flow. Validation (Task 11) uses a copy vault.
- **The learn SKILL.md task cannot be marked complete** without RED and GREEN headless evidence from fresh `claude -p` processes.

---

## Interfaces

Key structs and function signatures needed across tasks.

### New struct: `vocabLastRefitDoc` (Task 1)

```go
// vocabLastRefitDoc holds the vault state at the time of the last bootstrap or refit.
//
//nolint:tagliatelle // JSON keys follow snake_case sidecar spec contract
type vocabLastRefitDoc struct {
    NoteCount int    `json:"note_count"`
    Date      string `json:"date"` // YYYY-MM-DD
}
```

### Extended struct: `vocabCentroidsDoc` (Task 1)

Extend the existing struct in `internal/cli/vocab_centroids.go` (currently at line 41):

```go
type vocabCentroidsDoc struct {
    SchemaVersion    int                           `json:"schema_version"`
    EmbeddingModelID string                        `json:"embedding_model_id"`
    Dims             int                           `json:"dims"`
    Terms            map[string]vocabCentroidEntry `json:"terms"`
    // --- new fields ---
    RefitPending bool               `json:"refit_pending,omitempty"`
    RefitReason  string             `json:"refit_reason,omitempty"`
    LastRefit    *vocabLastRefitDoc `json:"last_refit,omitempty"`
}
```

### Extended struct: `aggregatedSummary` (Task 8)

Add one field to the existing `aggregatedSummary` in `internal/cli/query.go` (currently defined around line 175):

```go
refitPending bool
```

### New function signatures (Task 2, 3, 4)

```go
// evaluateVocabTriggers returns (fired, reason) for the in-process threshold checks.
// Returns ("", false) when lastRefit is nil (no baseline yet — seed it first).
func evaluateVocabTriggers(
    totalNotes, untaggedCount int,
    memberCounts map[string]int,
    lastRefit *vocabLastRefitDoc,
    now time.Time,
) (bool, string)

// collectTriggerVaultStats counts total non-vocab notes, untagged notes, and per-term
// member counts by reading frontmatter from vault filenames.
func collectTriggerVaultStats(
    vault    string,
    listMD   func(string) ([]string, error),
    readFile func(string) ([]byte, error),
) (totalNotes, untaggedCount int, memberCounts map[string]int)

// checkAndPersistVocabRefitTrigger evaluates vault trigger state and updates
// vocab.centroids.json in place. No-ops when any dep is nil.
func checkAndPersistVocabRefitTrigger(
    vault     string,
    listMD    func(string) ([]string, error),
    readFile  func(string) ([]byte, error),
    writeFile func(string, []byte) error,
    logWarn   func(string, ...any),
    now       time.Time,
)
```

### Modified function signatures (Task 4 — seeding)

```go
// retagAllNotesTwoPass — add lastRefit parameter; pass nil when called from tests that
// don't exercise the trigger fields.
func retagAllNotesTwoPass(
    deps       VocabDeps,
    vault      string,
    descTerms  []TermWithVector,
    floor      float32,
    lastRefit  *vocabLastRefitDoc, // NEW: seeded by bootstrap/refit, nil elsewhere
) map[string]int

// writeCentroidsFile — add lastRefit parameter (same rationale).
func writeCentroidsFile(
    deps      VocabDeps,
    vault     string,
    entries   map[string]vocabCentroidEntry,
    lastRefit *vocabLastRefitDoc, // NEW: when non-nil, resets refit_pending + writes last_refit
)
```

### New dep fields

**`LearnDeps` — ONE new field: `ListMD func(vault string) ([]string, error)`** (mirrors the
AmendDeps addition in Task 6; wire `ListMD: osVault.ListMD` in `newOsLearnDeps()` — `osVault`
is already in scope at learn.go:332). Do NOT use the existing `ListBasenames` for the trigger
scan: it strips `.md` and filters to Luhmann-ID notes (cli.go:39–47), so every
`readFile(vault, name)` would miss, the untagged rate would read 100%, and the trigger would
false-fire on every learn. The check also reuses `ReadSidecar` (read any vault file = `osVaultFS.ReadFile`),
`WriteNote` (write any vault file = `atomicWriteFile`). All three are already wired in
`newOsLearnDeps()`.

**`AmendDeps` — add one optional field** (Task 6):

```go
// ListMD lists .md filenames in the vault for the vocab trigger check.
// Optional: nil skips the trigger check (backward compat).
ListMD func(vault string) ([]string, error)
```

**`ResituateDeps` — add two optional fields** (Task 9):

```go
// LoadTermVectors returns vocab term+vector pairs from the vault.
// Optional: nil skips vocab assignment (backward compat).
LoadTermVectors func(vault string) ([]TermWithVector, error)
// ListMD lists .md filenames in the vault for the vocab trigger check.
// Optional: nil skips the trigger check.
ListMD func(vault string) ([]string, error)
```

### `queryPayload` extension (Task 8)

```go
type queryPayload struct {
    Version      int            `yaml:"version"`
    Phrases      []string       `yaml:"phrases"`
    Items        []queryItem    `yaml:"items"`
    Clusters     []queryCluster `yaml:"clusters"`
    Budget       queryBudget    `yaml:"budget"`
    RefitPending bool           `yaml:"refit_pending,omitempty"` // NEW
}
```

---

## Tasks

### Task 1 — Schema: extend `vocabCentroidsDoc`

**File:** `internal/cli/vocab_centroids.go`

**What:** Add `vocabLastRefitDoc` struct and three fields to `vocabCentroidsDoc`.

**RED test** (`internal/cli/vocab_commands_test.go`):

```go
func TestVocabCentroidsDoc_NewFieldsRoundTrip(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    doc := vocabCentroidsDoc{  // access via export shim below
        SchemaVersion: 1,
        RefitPending:  true,
        RefitReason:   "growth: 41 notes, 15 days",
        LastRefit:     &vocabLastRefitDoc{NoteCount: 100, Date: "2026-07-03"},
        Terms:         map[string]vocabCentroidEntry{"x": {MemberCount: 3}},
    }
    data, _ := json.Marshal(doc)
    var got vocabCentroidsDoc
    g.Expect(json.Unmarshal(data, &got)).NotTo(HaveOccurred())
    if err := json.Unmarshal(data, &got); err != nil { return }
    g.Expect(got.RefitPending).To(BeTrue())
    g.Expect(got.RefitReason).To(Equal("growth: 41 notes, 15 days"))
    g.Expect(got.LastRefit).NotTo(BeNil())
    if got.LastRefit == nil { return }
    g.Expect(got.LastRefit.NoteCount).To(Equal(100))
    g.Expect(got.LastRefit.Date).To(Equal("2026-07-03"))
}

func TestVocabCentroidsDoc_ZeroValueOmitted(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    doc := vocabCentroidsDoc{SchemaVersion: 1}
    data, _ := json.Marshal(doc)
    jsonStr := string(data)
    g.Expect(jsonStr).NotTo(ContainSubstring("refit_pending"))
    g.Expect(jsonStr).NotTo(ContainSubstring("refit_reason"))
    g.Expect(jsonStr).NotTo(ContainSubstring("last_refit"))
}
```

Add export shims to `export_test.go`:

```go
type ExportVocabCentroidsDoc     = vocabCentroidsDoc
type ExportVocabLastRefitDoc      = vocabLastRefitDoc
type ExportVocabCentroidEntry     = vocabCentroidEntry
```

**GREEN:** Add `vocabLastRefitDoc` struct and extend `vocabCentroidsDoc` per the Interfaces block.

**Verify:** `targ test` passes. `targ check-full` clean.

**Commit:** `feat(vocab): extend centroids schema — refit_pending, refit_reason, last_refit`

---

### Task 2 — Pure trigger evaluator

**File:** `internal/cli/vocab_trigger.go` (new file, `package cli`)

**What:** Named constants + pure `evaluateVocabTriggers` function.

```go
// Named constants for trigger thresholds.
const (
    // refitGrowthMinNotes is the minimum new-note growth since the last refit
    // to consider the growth trigger armed.
    refitGrowthMinNotes = 40
    // refitGrowthMinDays is the minimum days elapsed since the last refit
    // (conjunct with refitGrowthMinNotes) to fire the growth trigger.
    refitGrowthMinDays = 14
    // refitUntaggedRateMax is the vault-wide untagged rate above which the
    // untagged trigger fires (exclusive: >8%).
    refitUntaggedRateMax = 0.08
    // pctDivisor converts a fraction to a percentage-space value used in rate comparisons.
    // hubThreshold (0.25) is defined in vocab_commands.go and reused here.
)
```

**Note:** `hubThreshold = 0.25` is already defined in `vocab_commands.go:283` (`package cli`).
Do NOT redefine it. Reference it directly.

**Function:**

```go
func evaluateVocabTriggers(
    totalNotes, untaggedCount int,
    memberCounts map[string]int,
    lastRefit *vocabLastRefitDoc,
    now time.Time,
) (bool, string) {
    if lastRefit == nil {
        return false, "" // no baseline — caller seeds and returns
    }
    // (a) growth trigger
    lastRefitDate, parseErr := time.Parse(dateFormat, lastRefit.Date)
    if parseErr == nil {
        growth := totalNotes - lastRefit.NoteCount
        daysSince := int(now.Sub(lastRefitDate).Hours() / hoursPerDay)
        if growth >= refitGrowthMinNotes && daysSince >= refitGrowthMinDays {
            return true, fmt.Sprintf("growth: %d notes, %d days", growth, daysSince)
        }
    }
    // (b) untagged rate trigger
    if totalNotes > 0 {
        untaggedRate := float64(untaggedCount) / float64(totalNotes)
        if untaggedRate > refitUntaggedRateMax {
            return true, fmt.Sprintf("untagged: %.1f%%", untaggedRate*pctMultiplier)
        }
    }
    // (c) hub trigger
    for term, count := range memberCounts {
        if totalNotes > 0 && float64(count)/float64(totalNotes) > hubThreshold {
            return true, fmt.Sprintf("hub: %s (%.0f%%)",
                term, float64(count)/float64(totalNotes)*pctMultiplier)
        }
    }
    return false, ""
}
```

`hoursPerDay` ALREADY EXISTS in package cli (`recency.go:19`, untyped constant `= 24`) —
reference it directly; do NOT redeclare it in vocab_trigger.go (redeclaration breaks the build).

**RED tests** (`internal/cli/vocab_trigger_test.go`, new file):

```go
func TestEvaluateVocabTriggers_GrowthFires(t *testing.T) {
    t.Parallel()
    // growth >= 40 AND >= 14d: fires
    last := &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-15"}
    now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 18 days later
    fired, reason := cli.ExportEvaluateVocabTriggers(141, 5, nil, last, now)
    g.Expect(fired).To(BeTrue())
    g.Expect(reason).To(ContainSubstring("growth"))
}

func TestEvaluateVocabTriggers_GrowthBelowDaysFloor(t *testing.T) {
    t.Parallel()
    // growth >= 40 but only 5 days: no fire
    last := &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-29"}
    now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 4 days
    fired, _ := cli.ExportEvaluateVocabTriggers(141, 5, nil, last, now)
    g.Expect(fired).To(BeFalse())
}

func TestEvaluateVocabTriggers_UntaggedRateFires(t *testing.T) {
    t.Parallel()
    last := &cli.ExportVocabLastRefitDoc{NoteCount: 130, Date: "2026-06-01"}
    now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    // 10/100 = 10% > 8%
    fired, reason := cli.ExportEvaluateVocabTriggers(100, 10, nil, last, now)
    g.Expect(fired).To(BeTrue())
    g.Expect(reason).To(ContainSubstring("untagged"))
}

func TestEvaluateVocabTriggers_HubFires(t *testing.T) {
    t.Parallel()
    last := &cli.ExportVocabLastRefitDoc{NoteCount: 130, Date: "2026-06-01"}
    now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    // term "x" has 30/100 = 30% > 25%
    counts := map[string]int{"x": 30, "y": 5}
    fired, reason := cli.ExportEvaluateVocabTriggers(100, 0, counts, last, now)
    g.Expect(fired).To(BeTrue())
    g.Expect(reason).To(ContainSubstring("hub"))
}

func TestEvaluateVocabTriggers_NilLastRefit_NoFire(t *testing.T) {
    t.Parallel()
    now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    fired, _ := cli.ExportEvaluateVocabTriggers(100, 5, nil, nil, now)
    g.Expect(fired).To(BeFalse())
}
```

Add to `export_test.go`:

```go
var ExportEvaluateVocabTriggers = evaluateVocabTriggers
```

**GREEN:** Write the function and constants.

**Verify:** `targ test` passes.

**Commit:** `feat(vocab): pure trigger evaluator — growth/untagged/hub thresholds`

---

### Task 3 — Vault-stats helper + `checkAndPersistVocabRefitTrigger`

**File:** `internal/cli/vocab_trigger.go` (continue)

**What:** Side-effectful wrapper that reads vault state, evaluates triggers, persists the flag.

**`collectTriggerVaultStats`:**

```go
// countNonVocabNoteFiles counts basenames that are not vocab-kind files.
// A pure helper reused by bootstrap/refit seeding and the trigger check.
func countNonVocabNoteFiles(names []string) int {
    count := 0
    for _, name := range names {
        if !isVocabKindFilename(name) {
            count++
        }
    }
    return count
}

// collectTriggerVaultStats scans non-vocab note frontmatter for the trigger evaluation.
// Returns (totalNotes, untaggedCount, perTermMemberCounts).
// Unreadable or unparseable notes count as total but not tagged.
// DRY (Gate A W2): countMembersFromNotes (vocab_commands.go:697) walks the same
// read-unmarshal-filter loop for per-term counts. In the REFACTOR phase, extract a shared
// scanNonVocabNotes primitive both call (this function adds only untaggedCount) rather than
// keeping two copies of the loop.
func collectTriggerVaultStats(
    vault    string,
    listMD   func(string) ([]string, error),
    readFile func(string) ([]byte, error),
) (int, int, map[string]int) {
    names, listErr := listMD(vault)
    if listErr != nil {
        return 0, 0, nil
    }
    memberCounts := make(map[string]int)
    totalNotes, untaggedCount := 0, 0
    for _, name := range names {
        if isVocabKindFilename(name) {
            continue
        }
        totalNotes++
        raw, readErr := readFile(filepath.Join(vault, name))
        if readErr != nil {
            untaggedCount++
            continue
        }
        fm, ok := splitFrontmatter(raw)
        if !ok {
            untaggedCount++
            continue
        }
        var doc noteMiniDoc
        if yaml.Unmarshal(fm, &doc) != nil || len(doc.Vocab) == 0 {
            untaggedCount++
            continue
        }
        for _, term := range doc.Vocab {
            memberCounts[term]++
        }
    }
    return totalNotes, untaggedCount, memberCounts
}
```

**`checkAndPersistVocabRefitTrigger`:**

```go
// checkAndPersistVocabRefitTrigger evaluates vault trigger state and updates
// vocab.centroids.json. On first call (no last_refit): seeds the baseline and
// returns without firing. On subsequent calls: evaluates and persists when triggered.
// Silent no-op when any dep is nil or when centroids file is absent.
func checkAndPersistVocabRefitTrigger(
    vault     string,
    listMD    func(string) ([]string, error),
    readFile  func(string) ([]byte, error),
    writeFile func(string, []byte) error,
    logWarn   func(string, ...any),
    now       time.Time,
) {
    if listMD == nil || readFile == nil || writeFile == nil {
        return
    }
    doc, _ := readCentroidsDoc(vault, readFile) // zero-value on missing file

    totalNotes, untaggedCount, memberCounts := collectTriggerVaultStats(vault, listMD, readFile)

    if doc.LastRefit == nil {
        // Seed baseline — no trigger fires this call.
        doc.LastRefit = &vocabLastRefitDoc{
            NoteCount: totalNotes,
            Date:      now.Format(dateFormat),
        }
        if err := writeCentroidsDocRaw(vault, doc, writeFile); err != nil && logWarn != nil {
            logWarn("vocab trigger: seeding last_refit: %v", err)
        }
        return
    }

    if doc.RefitPending {
        return // already flagged — idempotent
    }

    fired, reason := evaluateVocabTriggers(totalNotes, untaggedCount, memberCounts, doc.LastRefit, now)
    if !fired {
        return
    }

    doc.RefitPending = true
    doc.RefitReason = reason
    if err := writeCentroidsDocRaw(vault, doc, writeFile); err != nil && logWarn != nil {
        logWarn("vocab trigger: persisting refit_pending: %v", err)
    }
}

// writeCentroidsDocRaw marshals doc and writes it to vocab.centroids.json.
// Preserves all existing fields (terms, trigger state) in a single write.
func writeCentroidsDocRaw(vault string, doc vocabCentroidsDoc, writeFile func(string, []byte) error) error {
    data, marshalErr := json.Marshal(doc)
    if marshalErr != nil {
        return fmt.Errorf("marshaling centroids: %w", marshalErr)
    }
    return writeFile(filepath.Join(vault, vocabCentroidsFilename), data)
}
```

**RED tests** (expand `vocab_trigger_test.go`):

```go
func TestCheckAndPersistVocabRefitTrigger_NilDeps_NoOp(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    // nil listMD → no panic, no write
    var written bool
    cli.ExportCheckAndPersistVocabRefitTrigger(
        "/vault", nil, func(string) ([]byte, error) { return nil, nil },
        func(string, []byte) error { written = true; return nil },
        nil, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
    )
    g.Expect(written).To(BeFalse())
}

func TestCheckAndPersistVocabRefitTrigger_MissingCentroids_SeedsBaseline(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    // No centroids file → seeds last_refit, no trigger fires.
    names := []string{"1.2026-01-01.note.md", "vocab.x.md"}
    noteContent := "---\ntype: fact\ntieriL2\nsituation: x\n---\n"
    var writtenData []byte
    cli.ExportCheckAndPersistVocabRefitTrigger(
        "/vault",
        func(string) ([]string, error) { return names, nil },
        func(path string) ([]byte, error) {
            if path == "/vault/1.2026-01-01.note.md" {
                return []byte(noteContent), nil
            }
            return nil, os.ErrNotExist
        },
        func(_ string, data []byte) error { writtenData = data; return nil },
        nil,
        time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
    )
    g.Expect(writtenData).NotTo(BeNil())
    var doc cli.ExportVocabCentroidsDoc
    g.Expect(json.Unmarshal(writtenData, &doc)).NotTo(HaveOccurred())
    if err := json.Unmarshal(writtenData, &doc); err != nil { return }
    g.Expect(doc.RefitPending).To(BeFalse(), "no trigger should fire on first seed")
    g.Expect(doc.LastRefit).NotTo(BeNil())
    if doc.LastRefit == nil { return }
    g.Expect(doc.LastRefit.NoteCount).To(Equal(1)) // only the non-vocab note
}

func TestCheckAndPersistVocabRefitTrigger_GrowthTrigger_SetsPending(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    // Vault has 150 notes; last_refit was at 100, 20 days ago → growth trigger fires.
    names := make([]string, 150)
    for i := range names {
        names[i] = fmt.Sprintf("%d.2026-01-01.note.md", i+1)
    }
    centroids := cli.ExportVocabCentroidsDoc{
        SchemaVersion: 1,
        LastRefit:     &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-13"},
    }
    centroidsData, _ := json.Marshal(centroids)
    var writtenData []byte
    now := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
    cli.ExportCheckAndPersistVocabRefitTrigger(
        "/vault",
        func(string) ([]string, error) { return names, nil },
        func(path string) ([]byte, error) {
            if strings.HasSuffix(path, "vocab.centroids.json") {
                return centroidsData, nil
            }
            // note content: untagged (no vocab frontmatter key)
            return []byte("---\ntype: fact\n---\n"), nil
        },
        func(_ string, data []byte) error { writtenData = data; return nil },
        nil, now,
    )
    g.Expect(writtenData).NotTo(BeNil())
    var got cli.ExportVocabCentroidsDoc
    g.Expect(json.Unmarshal(writtenData, &got)).NotTo(HaveOccurred())
    if err := json.Unmarshal(writtenData, &got); err != nil { return }
    g.Expect(got.RefitPending).To(BeTrue())
    g.Expect(got.RefitReason).To(ContainSubstring("growth"))
}

func TestCheckAndPersistVocabRefitTrigger_AlreadyPending_Idempotent(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    centroids := cli.ExportVocabCentroidsDoc{
        RefitPending: true,
        RefitReason:  "growth: 40 notes, 15 days",
        LastRefit:    &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-15"},
    }
    centroidsData, _ := json.Marshal(centroids)
    var writeCount int
    cli.ExportCheckAndPersistVocabRefitTrigger(
        "/vault",
        func(string) ([]string, error) { return []string{"1.note.md"}, nil },
        func(path string) ([]byte, error) {
            if strings.HasSuffix(path, "vocab.centroids.json") {
                return centroidsData, nil
            }
            return []byte("---\ntype: fact\n---\n"), nil
        },
        func(string, []byte) error { writeCount++; return nil },
        nil, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
    )
    g.Expect(writeCount).To(Equal(0), "already-pending should not write again")
}
```

Add to `export_test.go`:

```go
var ExportCheckAndPersistVocabRefitTrigger = checkAndPersistVocabRefitTrigger
var ExportCountNonVocabNoteFiles           = countNonVocabNoteFiles
```

**GREEN:** Write the functions per the signatures above.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): trigger wrapper — checkAndPersistVocabRefitTrigger`

---

### Task 4 — Extend `writeCentroidsFile`/`retagAllNotesTwoPass` for bootstrap/refit seeding

**Files:** `internal/cli/vocab_centroids.go`, `internal/cli/vocab_commands.go`

**What:** Thread `lastRefit *vocabLastRefitDoc` through `writeCentroidsFile` and
`retagAllNotesTwoPass` so bootstrap and refit seed `last_refit` in a single write.
Add `countNonVocabNoteFiles` (pure helper, already written in Task 3) here if not extracted.

**Change `writeCentroidsFile`** (currently at line 260 in `vocab_centroids.go`):

```go
func writeCentroidsFile(
    deps      VocabDeps,
    vault     string,
    entries   map[string]vocabCentroidEntry,
    lastRefit *vocabLastRefitDoc,
) {
    names, _ := deps.ListMD(vault)
    modelID, dims := firstTermSidecarMeta(vault, names, deps.ReadFile)
    doc := vocabCentroidsDoc{
        SchemaVersion:    vocabCentroidsSchemaVersion,
        EmbeddingModelID: modelID,
        Dims:             dims,
        Terms:            entries,
        LastRefit:        lastRefit, // NEW — nil for non-seeding calls
        // RefitPending/RefitReason are intentionally zero: bootstrap/refit is a fresh start.
    }
    data, _ := json.Marshal(doc)
    writeErr := deps.WriteFile(filepath.Join(vault, vocabCentroidsFilename), data)
    if writeErr != nil && deps.LogWarning != nil {
        deps.LogWarning("vocab: writing %s: %v", vocabCentroidsFilename, writeErr)
    }
}
```

**Change `retagAllNotesTwoPass`** (currently at line 211 in `vocab_centroids.go`):

```go
func retagAllNotesTwoPass(
    deps      VocabDeps,
    vault     string,
    descTerms []TermWithVector,
    floor     float32,
    lastRefit *vocabLastRefitDoc, // NEW
) map[string]int {
    noteVecs := loadMemberNoteVectors(deps, vault)
    // ... (existing logic unchanged) ...
    writeCentroidsFile(deps, vault, entries, lastRefit) // pass through
    return memberCounts
}
```

**Update call sites in `RunVocabBootstrap`** (around line 151 in `vocab_commands.go`):

```go
// After loadTermVectors, before retagAllNotesTwoPass:
names, _ := deps.ListMD(args.Vault)
noteCount := countNonVocabNoteFiles(names)
lastRefit := &vocabLastRefitDoc{
    NoteCount: noteCount,
    Date:      when.Format(dateFormat),
}
if len(terms) > 0 {
    memberCounts = retagAllNotesTwoPass(deps, args.Vault, terms, floor, lastRefit)
}
```

**Update call site in `RunVocabRefit`** (around line 248 in `vocab_commands.go`):

```go
// After loadTermVectors, before retagAllNotesTwoPass:
names, _ := deps.ListMD(args.Vault)
noteCount := countNonVocabNoteFiles(names)
lastRefit := &vocabLastRefitDoc{
    NoteCount: noteCount,
    Date:      when.Format(dateFormat),
}
if len(terms) > 0 {
    _ = retagAllNotesTwoPass(deps, args.Vault, terms, DefaultVocabFloor, lastRefit)
}
```

Also update the test call site in `vocab_commands_test.go` where `retagAllNotesTwoPass` is called — pass `nil` for `lastRefit` to keep existing tests unchanged.

**RED test** (`vocab_commands_test.go` or new `vocab_centroids_test.go`):

```go
func TestRetagAllNotesTwoPass_SeedsLastRefit(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    lastRefit := &cli.ExportVocabLastRefitDoc{NoteCount: 50, Date: "2026-07-03"}
    var written []byte
    deps := cli.VocabDeps{
        ListMD:     func(string) ([]string, error) { return nil, nil },
        ReadFile:   func(string) ([]byte, error) { return nil, os.ErrNotExist },
        WriteFile:  func(_ string, data []byte) error { written = data; return nil },
        LogWarning: nil,
    }
    cli.ExportRetagAllNotesTwoPass(deps, "/vault", nil, 0.35, lastRefit)
    g.Expect(written).NotTo(BeNil())
    var doc cli.ExportVocabCentroidsDoc
    g.Expect(json.Unmarshal(written, &doc)).NotTo(HaveOccurred())
    if err := json.Unmarshal(written, &doc); err != nil { return }
    g.Expect(doc.LastRefit).NotTo(BeNil())
    if doc.LastRefit == nil { return }
    g.Expect(doc.LastRefit.NoteCount).To(Equal(50))
}
```

Add to `export_test.go`:

```go
var ExportRetagAllNotesTwoPass = retagAllNotesTwoPass
```

**GREEN:** Apply the signature changes and update all call sites.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): bootstrap/refit seed last_refit in centroids file`

---

### Task 5 — Hook into `applyVocabAssignmentAfterLearn`

**File:** `internal/cli/learn.go`

**What:** Call `checkAndPersistVocabRefitTrigger` after the vocab assignment in
`applyVocabAssignmentAfterLearn`. ONE new dep: `ListMD func(vault string) ([]string, error)`
on `LearnDeps` (full `.md` filenames, unfiltered — Gate A F2: `ListBasenames` strips `.md`
and Luhmann-filters, which would 100%-false-fire the untagged trigger). Wire in
`newOsLearnDeps()`: `ListMD: osVault.ListMD` (osVault already in scope at learn.go:332).
Reuses `ReadSidecar`, `WriteNote`, `LogWarning`, `Now` from `LearnDeps`.

**Add to `LearnDeps`** (learn.go:64–86 block):

```go
ListMD func(vault string) ([]string, error) // full .md filenames, for the trigger vault scan
```

**Change `applyVocabAssignmentAfterLearn`** (line 199 in `learn.go`) — add at the end:

```go
// Trigger check: evaluate vocab refit thresholds after every note write.
// Uses existing deps; all must be non-nil (gated inside the callee).
checkAndPersistVocabRefitTrigger(
    vault,
    deps.ListMD,
    deps.ReadSidecar,
    deps.WriteNote,
    deps.LogWarning,
    deps.Now(),
)
```

**RED test** (`internal/cli/learn_test.go` or existing `cli_test.go`):

```go
// TestApplyVocabAssignment_Learn_SetsTriggerFlag drives applyVocabAssignmentAfterLearn
// with a vault at the growth threshold and asserts the trigger flag is persisted.
func TestApplyVocabAssignmentAfterLearn_TriggerFires(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    // 150 non-vocab notes, last_refit at 100 notes 20 days ago → growth trigger
    names := make([]string, 150)
    for i := range names {
        names[i] = fmt.Sprintf("%d.2026-01-01.note.md", i+1)
    }
    centroidsDoc := cli.ExportVocabCentroidsDoc{
        SchemaVersion: 1,
        LastRefit:     &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-13"},
    }
    centroidsData, _ := json.Marshal(centroidsDoc)
    var centroidsWritten []byte

    deps := cli.LearnDeps{
        Now:    func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
        ListMD: func(string) ([]string, error) { return names, nil },
        ReadSidecar: func(path string) ([]byte, error) {
            if strings.HasSuffix(path, "vocab.centroids.json") {
                return centroidsData, nil
            }
            // Note sidecar (body vector) — needed by applyVocabAssignmentAfterLearn:
            // return a minimal valid sidecar so term loading returns empty.
            return nil, os.ErrNotExist
        },
        WriteNote: func(path string, data []byte) error {
            if strings.HasSuffix(path, "vocab.centroids.json") { // check the REAL path (Gate A W1)
                centroidsWritten = data
            }
            return nil
        },
        LogWarning: nil,
        // Vocab assignment deps: all nil → assignment skips (no term notes in this fixture)
    }

    cli.ExportApplyVocabAssignmentAfterLearn(deps, "/vault", "/vault/150.note.md", "---\ntype: fact\n---\n")

    g.Expect(centroidsWritten).NotTo(BeNil(), "trigger check must write centroids")
    var got cli.ExportVocabCentroidsDoc
    g.Expect(json.Unmarshal(centroidsWritten, &got)).NotTo(HaveOccurred())
    if err := json.Unmarshal(centroidsWritten, &got); err != nil { return }
    g.Expect(got.RefitPending).To(BeTrue())
}
```

Add to `export_test.go`:

```go
var ExportApplyVocabAssignmentAfterLearn = applyVocabAssignmentAfterLearn
```

**GREEN:** Add the `checkAndPersistVocabRefitTrigger` call at the end of
`applyVocabAssignmentAfterLearn`.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): hook trigger check into applyVocabAssignmentAfterLearn`

---

### Task 6 — Hook into `applyVocabAssignmentAfterAmend`

**File:** `internal/cli/amend.go`

**What:** Add `ListMD func(vault string) ([]string, error)` to `AmendDeps`. Call the trigger
check at the end of `applyVocabAssignmentAfterAmend` using `deps.ListMD`, `deps.Read`,
`deps.Write`, `deps.LogWarning`, `deps.Now()`.

**Add field to `AmendDeps`** (at line ~44, after existing fields):

```go
// ListMD lists .md filenames in the vault for the vocab trigger check.
// Optional: nil skips the trigger check (backward compat).
ListMD func(vault string) ([]string, error)
```

**Wire in `newOsAmendDeps()`** (currently at line ~335 — reuse the `osVault := &osVaultFS{}`
instance already in scope at amend.go:340; do not construct a second one, Gate A W5):

```go
ListMD: osVault.ListMD,
```

**Extend `applyVocabAssignmentAfterAmend`** (line 285 in `amend.go`) — add at the end:

```go
// Trigger check — reuses Read/Write deps already wired for vocab assignment.
checkAndPersistVocabRefitTrigger(
    vault,
    deps.ListMD,
    deps.Read,
    deps.Write,
    deps.LogWarning,
    deps.Now(),
)
```

**RED test** (in `amend_test.go` or `vocab_test.go`):

```go
func TestApplyVocabAssignmentAfterAmend_TriggerFires(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    names := make([]string, 150)
    for i := range names {
        names[i] = fmt.Sprintf("%d.2026-01-01.note.md", i+1)
    }
    centroidsDoc := cli.ExportVocabCentroidsDoc{
        SchemaVersion: 1,
        LastRefit:     &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-06-13"},
    }
    centroidsData, _ := json.Marshal(centroidsDoc)
    var centroidsWritten []byte

    deps := cli.AmendDeps{
        Now:    func() time.Time { return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) },
        ListMD: func(string) ([]string, error) { return names, nil },
        Read: func(path string) ([]byte, error) {
            if strings.HasSuffix(path, "vocab.centroids.json") {
                return centroidsData, nil
            }
            return nil, os.ErrNotExist
        },
        Write: func(_ string, data []byte) error {
            centroidsWritten = data
            return nil
        },
        LogWarning: nil,
    }

    cli.ExportApplyVocabAssignmentAfterAmend(deps, "/vault", "/vault/1.note.md", "---\ntype: fact\n---\n")

    g.Expect(centroidsWritten).NotTo(BeNil())
    var got cli.ExportVocabCentroidsDoc
    g.Expect(json.Unmarshal(centroidsWritten, &got)).NotTo(HaveOccurred())
    if err := json.Unmarshal(centroidsWritten, &got); err != nil { return }
    g.Expect(got.RefitPending).To(BeTrue())
}
```

Add to `export_test.go`:

```go
var ExportApplyVocabAssignmentAfterAmend = applyVocabAssignmentAfterAmend
```

**GREEN:** Add the `ListMD` field, wire it, and add the trigger call.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): hook trigger check into applyVocabAssignmentAfterAmend`

---

### Task 7 — Verdict line in `engram vocab stats`

**File:** `internal/cli/vocab_commands.go`

**What:** Append a `verdict:` line to `printStatsReport`. Read `refit_pending` from
`vocab.centroids.json` in `RunVocabStats`. Migration: absent `last_refit` → `verdict: OK`.

**Change `printStatsReport` signature** (currently at line 1019):

```go
func printStatsReport(
    stdout        io.Writer,
    termNames     []string,
    memberCounts  map[string]int,
    totalNotes, untaggedCount int,
    vocabVersion  string,
    refitPending  bool,   // NEW
    refitReason   string, // NEW
) {
    // ... existing body unchanged ...

    // Verdict line — single source of truth is the persisted flag.
    if refitPending {
        _, _ = fmt.Fprintf(stdout, "verdict: REFIT_PENDING (%s)\n", refitReason)
    } else {
        _, _ = fmt.Fprintln(stdout, "verdict: OK")
    }
}
```

**Change `RunVocabStats`** (line 267 in `vocab_commands.go`):

```go
func RunVocabStats(args VocabStatsArgs, deps VocabStatsDeps, stdout io.Writer) error {
    names, listErr := deps.ListMD(args.Vault)
    if listErr != nil {
        return fmt.Errorf("vocab stats: listing vault: %w", listErr)
    }
    termNames, memberCounts, totalNotes, untaggedCount := collectVaultStats(names, deps, args.Vault)
    vocabVersion := loadCurrentVocabVersion(args.Vault, deps.ReadFile)

    // Read refit_pending from centroids (migration: absent = OK, no false fire).
    refitPending := false
    refitReason := ""
    centroidsDoc, centroidsOK := readCentroidsDoc(args.Vault, deps.ReadFile)
    if centroidsOK {
        refitPending = centroidsDoc.RefitPending
        refitReason = centroidsDoc.RefitReason
    }

    sort.Strings(termNames)
    printStatsReport(stdout, termNames, memberCounts, totalNotes, untaggedCount,
        vocabVersion, refitPending, refitReason)
    return nil
}
```

**RED test** (`vocab_commands_test.go`):

```go
func TestPrintStatsReport_VerdictOK(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    var buf strings.Builder
    cli.ExportPrintStatsReport(&buf, nil, nil, 10, 0, "1.0", false, "")
    g.Expect(buf.String()).To(ContainSubstring("verdict: OK"))
}

func TestPrintStatsReport_VerdictRefitPending(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    var buf strings.Builder
    cli.ExportPrintStatsReport(&buf, nil, nil, 10, 0, "1.0", true, "growth: 41 notes, 15 days")
    g.Expect(buf.String()).To(ContainSubstring("verdict: REFIT_PENDING (growth: 41 notes, 15 days)"))
}

func TestRunVocabStats_ReadsRefitPendingFromCentroids(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    centroidsDoc := cli.ExportVocabCentroidsDoc{
        RefitPending: true,
        RefitReason:  "hub: agentic-recall-triggers (30%)",
    }
    centroidsData, _ := json.Marshal(centroidsDoc)
    deps := cli.VocabStatsDeps{
        ListMD:   func(string) ([]string, error) { return nil, nil },
        ReadFile: func(path string) ([]byte, error) {
            if strings.HasSuffix(path, "vocab.centroids.json") {
                return centroidsData, nil
            }
            return nil, os.ErrNotExist
        },
    }
    var buf strings.Builder
    g.Expect(cli.RunVocabStats(cli.VocabStatsArgs{Vault: "/vault"}, deps, &buf)).NotTo(HaveOccurred())
    g.Expect(buf.String()).To(ContainSubstring("REFIT_PENDING"))
}
```

Add to `export_test.go`:

```go
var ExportPrintStatsReport = printStatsReport
```

**GREEN:** Change the signature of `printStatsReport`, update its call site in `RunVocabStats`,
and update any other test call sites that call `printStatsReport` via the export shim.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): verdict line in engram vocab stats`

---

### Task 8 — Query payload `refit_pending` field

**File:** `internal/cli/query.go`

**What:** Add `RefitPending bool yaml:"refit_pending,omitempty"` to `queryPayload`. Read from
`vocab.centroids.json` in `RunQuery` using the existing `deps.Read` and `args.VaultPath`.
Pass through `aggregatedSummary.refitPending`.

**Add field to `aggregatedSummary`** (around line 175):

```go
refitPending bool
```

**Add field to `queryPayload`** (line 323):

```go
RefitPending bool `yaml:"refit_pending,omitempty"` // set only when vocab refit is pending
```

**Populate in `RunQuery`** (after the vault scan, before `renderQueryPayload`):

```go
// Read refit_pending from vocab.centroids.json (read-only: query never writes).
// readCentroidsDoc is defined in vocab_centroids.go and degrades cleanly on missing file.
if centroidsDoc, ok := readCentroidsDoc(args.VaultPath, deps.Read); ok {
    merged.refitPending = centroidsDoc.RefitPending
}
```

**In `renderQueryPayload`** (line ~1431), set the new field in the payload literal:

```go
payload := queryPayload{
    Version:      1,
    Phrases:      merged.phrases,
    Items:        items,
    Clusters:     clusters,
    Budget:       queryBudget{...},
    RefitPending: merged.refitPending, // NEW
}
```

**RED test** (`cli_test.go` or new `query_refit_test.go`):

```go
func TestQueryPayload_RefitPendingOmittedWhenFalse(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    out, err := cli.ExportRenderQueryPayloadRefitPending(false)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }
    g.Expect(out).NotTo(ContainSubstring("refit_pending"))
}

func TestQueryPayload_RefitPendingPresentWhenTrue(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)
    out, err := cli.ExportRenderQueryPayloadRefitPending(true)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }
    g.Expect(out).To(ContainSubstring("refit_pending: true"))
}
```

Add to `export_test.go`:

```go
// ExportRenderQueryPayloadRefitPending renders a minimal payload with refitPending set,
// so tests can assert the refit_pending field's presence/omission.
func ExportRenderQueryPayloadRefitPending(pending bool) (string, error) {
    var buf bytes.Buffer
    err := renderQueryPayload(&buf, aggregatedSummary{refitPending: pending})
    return buf.String(), err
}
```

**GREEN:** Add the field, populate it in `RunQuery`, render it in `renderQueryPayload`.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): refit_pending flag in query payload (top-level, omitempty)`

---

### Task 9 — Resituate rider: vocab assignment + trigger check

**File:** `internal/cli/resituate.go`

**What:** Add `LoadTermVectors` and `ListMD` to `ResituateDeps`. Add
`applyVocabAssignmentAfterResituate` (mirrors `applyVocabAssignmentAfterAmend`). Call it in
`RunResituate` after `writeResituatedSidecar` succeeds.

**Verify first (code-confirmed):** `resituate.go` currently has ZERO vocab assignment calls.
`RunResituate` calls `resituate.go:writeResituatedSidecar` then returns. No assignment, no trigger.

**Extend `ResituateDeps`** (line ~25 in `resituate.go`):

```go
type ResituateDeps struct {
    Lock     func(vault string) (func(), error)
    Scan     func(vault string) ([]vaultgraph.Note, error)
    Read     func(path string) ([]byte, error)
    Write    func(path string, data []byte) error
    Embedder embed.Embedder
    // NEW optional fields — nil skips vocab assignment and trigger check.
    LoadTermVectors func(vault string) ([]TermWithVector, error)
    ListMD          func(vault string) ([]string, error)
    LogWarning      func(format string, args ...any)
    Now             func() time.Time
}
```

**Wire in `newOsResituateDeps()`** (line ~122; single shared `osVault := &osVaultFS{}` — do
not construct multiple instances, Gate A W5):

```go
osVault := &osVaultFS{}
// ... existing fields ...
LoadTermVectors: func(vault string) ([]TermWithVector, error) {
    return loadAssignmentTermVectors(vault, osVault.ListMD, osVault.ReadFile)
},
ListMD:     osVault.ListMD,
LogWarning: logWarningToStderrf, // package-level helper, learn.go:315 (same wiring as amend.go:364)
Now:        time.Now,            // DI at the edge — time.Now only in wiring, never in the function body
```

**New function** (in `resituate.go`):

```go
// applyVocabAssignmentAfterResituate assigns vocab terms and checks the refit
// trigger for a resituated note. Mirrors applyVocabAssignmentAfterAmend.
// Requires LoadTermVectors + Read + Write to be non-nil for assignment.
// Requires ListMD + Read + Write + Now to be non-nil for the trigger check.
func applyVocabAssignmentAfterResituate(deps ResituateDeps, vault, notePath, content string) {
    if deps.LoadTermVectors == nil || deps.Read == nil || deps.Write == nil {
        return
    }
    terms, termsErr := deps.LoadTermVectors(vault)
    if termsErr != nil || len(terms) == 0 {
        return
    }
    bodyVec, ok := loadBodyVectorForNote(deps.Read, notePath)
    if !ok {
        return
    }
    assigned := AssignVocabTerms(bodyVec, terms, DefaultVocabFloor)
    updated := WriteVocabAssignment(content, assigned)
    if updated != content {
        if writeErr := deps.Write(notePath, []byte(updated)); writeErr != nil && deps.LogWarning != nil {
            deps.LogWarning("resituate: vocab assignment write failed for %s: %v", notePath, writeErr)
        }
    }
    if deps.Now == nil {
        return // trigger check needs an injected clock; wiring provides time.Now
    }
    // Trigger check — uses ListMD dep (optional, gated inside the callee).
    checkAndPersistVocabRefitTrigger(
        vault, deps.ListMD, deps.Read, deps.Write, deps.LogWarning, deps.Now(),
    )
}
```

**Call in `RunResituate`** (after `writeResituatedSidecar`, line ~87 in `resituate.go`):

```go
if embedErr := writeResituatedSidecar(ctx, deps, full, content); embedErr != nil {
    return embedErr
}

applyVocabAssignmentAfterResituate(deps, args.Vault, full, content) // NEW
```

**DI note (Gate A W3/W4):** `Now` and `LogWarning` are injected via `ResituateDeps` and wired
to `time.Now`/`logWarningToStderrf` at the edge — no direct `time.Now()`/stderr calls in the
function body (repo DI principle; consistent with LearnDeps/AmendDeps).

**RED test** (`resituate_test.go` or `vocab_test.go`):

```go
func TestRunResituate_CallsVocabAssignment(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    const noteContent = "---\ntype: fact\ntier: L2\nsituation: old\nsubject: A\npredicate: has\nobject: B\n" +
        "luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\nInformation learned: when in old, A has B.\n"

    var writtenPaths []string
    fakeVec := make([]float32, 4) // zero vec — below floor → no assignment
    fakeSidecar := embed.MarshalSidecar(embed.Sidecar{
        SchemaVersion: 1, EmbeddingModelID: "test", Dims: 4,
        BodyVector: fakeVec, SituationVector: fakeVec,
    })

    deps := cli.ResituateDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: "1aa.2026-01-01.note.md", LuhmannID: "1aa"}}, nil
        },
        Read: func(path string) ([]byte, error) {
            switch {
            case strings.HasSuffix(path, ".md"):
                return []byte(noteContent), nil
            case strings.HasSuffix(path, ".vec.json"):
                return fakeSidecar, nil
            }
            return nil, os.ErrNotExist
        },
        Write: func(path string, _ []byte) error {
            writtenPaths = append(writtenPaths, path)
            return nil
        },
        Embedder: fakeEmbedder{}, // must provide; write tests use it
        LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
            return nil, nil // no terms → no assignment, but call path exercised
        },
    }

    var buf strings.Builder
    err := cli.RunResituate(t.Context(), cli.ResituateArgs{
        Vault:     "/vault",
        Note:      "1aa",
        Situation: "new situation",
    }, deps, &buf)
    g.Expect(err).NotTo(HaveOccurred())
    // Verify note + sidecar were written (existing behavior).
    g.Expect(writtenPaths).To(ContainElement(ContainSubstring(".md")))
}
```

The test above is a RED for the assignment wiring (since `RunResituate` currently fails to call
`applyVocabAssignmentAfterResituate`). For a sharper RED, also verify that a note with a term
in a non-nil `LoadTermVectors` result gets vocab tags rewritten:

```go
func TestApplyVocabAssignmentAfterResituate_TagsNote(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    termVec := []float32{1, 0, 0, 0}
    noteVec  := []float32{0.9, 0.1, 0, 0} // cosine ≈ 0.99 > floor 0.35
    fakeSidecar := embed.MarshalSidecar(embed.Sidecar{
        SchemaVersion: 1, EmbeddingModelID: "test", Dims: 4,
        BodyVector: noteVec, SituationVector: make([]float32, 4),
    })

    const rawNote = "---\ntype: fact\n---\n\nBody.\n"
    var written []byte
    deps := cli.ResituateDeps{
        Read: func(path string) ([]byte, error) {
            if strings.HasSuffix(path, ".vec.json") { return fakeSidecar, nil }
            return nil, os.ErrNotExist
        },
        Write: func(_ string, data []byte) error { written = data; return nil },
        LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
            return []cli.TermWithVector{{Term: "agentic-recall-triggers", Vector: termVec}}, nil
        },
    }

    cli.ExportApplyVocabAssignmentAfterResituate(deps, "/vault", "/vault/1.note.md", rawNote)
    g.Expect(written).NotTo(BeNil())
    g.Expect(string(written)).To(ContainSubstring("agentic-recall-triggers"))
}
```

Add to `export_test.go`:

```go
var ExportApplyVocabAssignmentAfterResituate = applyVocabAssignmentAfterResituate
```

**GREEN:** Add the fields, wire them, write `applyVocabAssignmentAfterResituate`, call it.

**Verify:** `targ test`, `targ check-full`.

**Commit:** `feat(vocab): resituate rider — vocab assignment + trigger check`

---

### Task 10 — learn SKILL.md conditional (writing-skills TDD)

**File:** `/Users/joe/.claude/skills/learn/SKILL.md`

**What:** Add a Step 1.5 conditional to the learn skill: after `engram ingest --auto`, run
`engram vocab stats`. If the verdict reads `REFIT_PENDING`, run the vocab refit flow
autonomously and report the result loudly.

**Important:** this task MUST follow the `superpowers:writing-skills` discipline in full.
Red/green headless evidence is a hard gate. Subagents MUST NOT be used for the control/treatment
arms — they inherit session context and will recall the REFIT_PENDING behavior even before the
edit. Each arm is a fresh `claude -p` process.

#### A. Verify the exact refit flow invocation

The `engram vocab refit` binary has two sub-flows (verified via `engram vocab --help` and
`vocab_commands.go:RunVocabRefit`):

1. `engram vocab refit --emit-request` → prints JSON payload (current_terms, stats, instruction)
2. LLM derives a YAML plan `{new_terms, renames, removals}`
3. `engram vocab refit --plan <file>` → applies the plan (renames, removals, additions, re-tag)

No existing skill references this flow. The learn skill conditional must describe all three steps.

#### B. RED baseline — headless, fresh process

**Prerequisite:** the current learn skill must already be deployed at
`/Users/joe/.claude/skills/learn/SKILL.md` (verified present 2026-07-03) — the RED baseline
captures the behavior of the DEPLOYED text.

Establish that the CURRENT learn/SKILL.md does NOT act on a REFIT_PENDING verdict:

```bash
# Create isolated fixture dir with only the current learn skill as context
FIXTURE_RED=$(mktemp -d)
cat > "$FIXTURE_RED/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/learn/SKILL.md
EOF

# Run headless FROM the fixture dir (claude -p reads CLAUDE.md from cwd); N=3, independent
cd "$FIXTURE_RED"
for i in 1 2 3; do
  claude -p \
    "You are running the learn skill right now. Step 1 — you ran engram ingest --auto and got: 3 chunks embedded. The output of engram vocab stats is:\n\nvocab stats (version: 2.0)\nterms: 12  member-notes: 163  untagged: 4\nuntagged-rate: 2.5%\n  agentic-recall-triggers: 18 members [hub]\n  cost-optimization: 14 members\n  go-code-conventions: 22 members\nverdict: REFIT_PENDING (growth: 41 notes, 15 days)\n\nDescribe all the actions you now take. Be specific about what commands you run." \
    2>&1 | tee "$FIXTURE_RED/run-$i.txt"
done
```

(The prompt is describe-only — the arm needs no tool permissions, so no permission flags.)

**Expected RED (pinned criterion):** a run FAILS-to-act when its response never names
`engram vocab refit`. RED passes when 0/3 runs name it. The agent should stop after the sweep
(Step 1) and proceed to Step 2 (crystallize lessons).

Record the RED score (0/3, 1/3, etc.) in the plan results doc.

#### C. GREEN — edit the SKILL.md

Add a **Step 1.5** section immediately after Step 1's `engram ingest --auto` block:

```markdown
## Step 1.5 — Vocab liveness check

Run `engram vocab stats`.

If the output includes a line matching `verdict: REFIT_PENDING (<reason>)`, run the vocab
refit flow autonomously — do not defer to the user:

1. Run `engram vocab refit --emit-request`. Save its JSON output.
2. Derive a YAML refit plan from the JSON (review terms, propose merges/splits/removals for
   orphans < 2 members and hubs > 25%). Write the plan to `/tmp/vocab-refit-plan.yaml`.
3. Run `engram vocab refit --plan /tmp/vocab-refit-plan.yaml` to apply the plan.
4. **Report loudly:** "Vocab refit applied: <version bump>. Triggered by: <reason>."

If the verdict is `verdict: OK`, continue to Step 2 with no further vocab action.
```

**IMPORTANT: use `superpowers:writing-skills` for the actual edit.** Do not edit the SKILL.md
file directly in this task without first invoking that skill. It enforces the full RED/GREEN/
REFACTOR cycle and pressure tests.

#### D. GREEN verification — headless, fresh process

Same fixture mechanics as RED — the `@import` now resolves to the EDITED deployed skill:

```bash
FIXTURE_GREEN=$(mktemp -d)
cat > "$FIXTURE_GREEN/CLAUDE.md" <<'EOF'
@/Users/joe/.claude/skills/learn/SKILL.md
EOF

cd "$FIXTURE_GREEN"
for i in 1 2 3; do
  claude -p \
    "You are running the learn skill right now. Step 1 — you ran engram ingest --auto and got: 3 chunks embedded. The output of engram vocab stats is:\n\nvocab stats (version: 2.0)\nterms: 12  member-notes: 163  untagged: 4\nuntagged-rate: 2.5%\n  agentic-recall-triggers: 18 members [hub]\n  cost-optimization: 14 members\n  go-code-conventions: 22 members\nverdict: REFIT_PENDING (growth: 41 notes, 15 days)\n\nDescribe all the actions you now take. Be specific about what commands you run." \
    2>&1 | tee "$FIXTURE_GREEN/run-$i.txt"
done
```

**Pass criterion (pinned):** ≥ 2/3 runs name BOTH `engram vocab refit --emit-request` AND
`engram vocab refit --plan` as actions they take (not merely quote the skill text).

#### E. Refactor — pressure test

Run one more arm with an OK verdict to verify the conditional does NOT fire. Run this in the
SAME shell session as D (`$FIXTURE_GREEN` is only set there); if the session was lost, re-set
`FIXTURE_GREEN` to D's mktemp path first — `cd ""` on an unset var exits 1:

```bash
cd "$FIXTURE_GREEN"
claude -p \
  "You are running the learn skill. engram ingest --auto returned 2 chunks embedded. engram vocab stats shows: verdict: OK. What do you do next?" \
  2>&1
```

**Pass criterion:** response does not mention vocab refit.

**Commit (for the skill edit, via `/commit`):**
`feat(learn-skill): step 1.5 vocab liveness check — REFIT_PENDING triggers autonomous refit`

---

### Task 11 — End-to-end validation harness (metered refit rider)

**What:** Copy-vault harness verifying:
1. After a simulated growth trigger condition, `engram learn` sets `refit_pending: true`.
2. `engram vocab stats` shows `verdict: REFIT_PENDING`.
3. The refit flow runs once against the copy vault with token/$ metering recorded.
4. After refit, `refit_pending: false` and `last_refit` is updated.

**Steps:**

```bash
# 1. Install the real binary
go install ./cmd/engram

# 2. Copy the live vault to a temp dir (read-only from here)
set -u   # unset-var expansion fails loud — never silently falls back to the live vault
LIVE_VAULT="${ENGRAM_VAULT_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault}"  # env var usually UNSET; mirrors the binary's resolution (vocab_commands.go:48 + DataDirFromHome honoring XDG_DATA_HOME)
WORK_DIR=$(mktemp -d)   # per-run scratch (refit request/plan files); avoids /tmp collisions
COPY_VAULT=$WORK_DIR/vocab-trigger-validation-vault
export WORK_DIR COPY_VAULT   # the python heredocs below read these via os.environ
cp -r "$LIVE_VAULT" "$COPY_VAULT"

# 3. Rewrite last_refit in the copy to simulate growth threshold:
#    note_count = (current_count - 41), date = (today - 15d)
python3 - <<'EOF'
import json, os, datetime
vault = os.environ['COPY_VAULT']
path = os.path.join(vault, 'vocab.centroids.json')
with open(path) as f:
    doc = json.load(f)
# Count non-vocab notes
import glob
total = len([n for n in glob.glob(os.path.join(vault, '*.md'))
             if not os.path.basename(n).startswith('vocab.')])
doc['last_refit'] = {
    'note_count': total - 41,
    'date': (datetime.date.today() - datetime.timedelta(days=15)).strftime('%Y-%m-%d')
}
doc['refit_pending'] = False
with open(path, 'w') as f:
    json.dump(doc, f)
print(f"Seeded last_refit: note_count={total - 41}, date set -15d, total_now={total}")
EOF

# 4. Perform one real write via the installed binary against the copy vault
ENGRAM_VAULT_PATH="$COPY_VAULT" engram learn fact \
  --slug "trigger-validation-probe" \
  --situation "validating the vocab trigger check" \
  --subject "trigger" --predicate "fires" --object "correctly" \
  --source "validation harness 2026-07-03"

# 5. Assert refit_pending is now true
python3 - <<'EOF'
import json, os
vault = os.environ['COPY_VAULT']
with open(os.path.join(vault, 'vocab.centroids.json')) as f:
    doc = json.load(f)
assert doc.get('refit_pending') == True, f"Expected refit_pending=true, got: {doc}"
print("PASS: refit_pending is true after growth trigger")
EOF

# 6. Verify stats verdict
ENGRAM_VAULT_PATH="$COPY_VAULT" engram vocab stats | grep "verdict: REFIT_PENDING"
echo "PASS: stats shows REFIT_PENDING"

# 7. Run the metered refit flow once (this is the metered-refit rider)
echo "--- METERED REFIT: PHASE A (script) ---"
ENGRAM_VAULT_PATH="$COPY_VAULT" engram vocab refit --emit-request | tee "$WORK_DIR/refit-request.json"
echo "PHASE A done — script STOPS here. Phase B is executor work, not script."
echo "Carry into Phases B/C: WORK_DIR=$WORK_DIR COPY_VAULT=$COPY_VAULT"
```

**PHASE B (the LLM half of the two-phase flow, and the metered work):** the harness is
deliberately two script blocks with the derivation between them; it cannot run unattended
end-to-end. Shell state does NOT persist across tool invocations — re-set and re-export
`WORK_DIR`/`COPY_VAULT` at the top of every Phase B/C block from Phase A's "Carry into" echo
line. To get a REAL measured dollar figure (Rider 2's whole point — an executor cannot
isolate one step's tokens from its session total), run the derivation as a dedicated fresh
headless process and use its whole-session cost:

```bash
set -u
export WORK_DIR=<value from Phase A's carry echo>
claude -p \
  "Derive an engram vocab refit plan. Below is the refit request JSON (current terms, member counts, stats, and instructions). Output ONLY the YAML plan ({new_terms, renames, removals} — merge/split/rename per the request's instructions; orphans <2 members, hubs >25%). No prose.

$(cat "$WORK_DIR/refit-request.json")" \
  --output-format json > "$WORK_DIR/refit-derivation.json"

# Extract the plan and the MEASURED cost of the derivation:
python3 - <<'EOF'
import json, os
work = os.environ['WORK_DIR']
with open(os.path.join(work, 'refit-derivation.json')) as f:
    result = json.load(f)
with open(os.path.join(work, 'refit-plan.yaml'), 'w') as f:
    f.write(result['result'])
print(f"MEASURED derivation cost: ${result['total_cost_usd']:.4f}")
print(f"(field name may vary by CLI version — inspect refit-derivation.json if absent; "
      f"the binary calls are pure Go, ~$0)")
EOF
```

Sanity-check the extracted YAML before Phase C (it must parse and reference only existing
terms). The printed `total_cost_usd` IS the per-refit LLM cost — record it. Also record
wall-clock for phases A–C together.

```bash
set -u
export WORK_DIR=<value from Phase A's carry echo>
export COPY_VAULT=<value from Phase A's carry echo>
# GUARD (Gate A B3): an empty ENGRAM_VAULT_PATH falls back to the LIVE vault — abort instead.
[ -d "$COPY_VAULT" ] || { echo "COPY_VAULT missing — abort (live-vault fallback hazard)"; exit 1; }
echo "--- METERED REFIT: PHASE C (script) ---"
ENGRAM_VAULT_PATH="$COPY_VAULT" engram vocab refit --plan "$WORK_DIR/refit-plan.yaml"
echo "--- METERED REFIT END ---"

# 8. Assert refit_pending is now cleared and last_refit is updated
python3 - <<'EOF'
import json, os
vault = os.environ['COPY_VAULT']
with open(os.path.join(vault, 'vocab.centroids.json')) as f:
    doc = json.load(f)
assert doc.get('refit_pending') == False or 'refit_pending' not in doc, \
    f"Expected refit_pending cleared, got: {doc}"
assert doc.get('last_refit') is not None, "Expected last_refit populated after refit"
print(f"PASS: refit cleared. last_refit = {doc['last_refit']}")
EOF

# 9. Clean up copy vault
rm -rf "$COPY_VAULT"
echo "Validation complete."
```

**Record the metered refit cost** ($ from Anthropic API) in:
`docs/design/2026-07-03-vocab-lifecycle-proposals.md` under a new section
"Metered refit result (2026-07-03)" — one line: `per-refit cost: $X.XX (measured YYYY-MM-DD)`.

**Commit:** `docs(vocab): record metered refit cost from validation harness`

---

## Parallelism notes (for a single executor)

| Task | Depends on | Can parallelize with |
|---|---|---|
| 1 (schema) | nothing | — |
| 2 (pure eval) | Task 1 | — |
| 3 (wrapper) | Task 2 | — |
| 4 (seeding) | Task 1 | — (modifies vocab_commands.go) |
| 5 (learn hook) | Task 3 | Tasks 6, 8, 9 (different files) |
| 6 (amend hook) | Task 3 | Tasks 5, 8, 9 (different files) |
| 7 (stats verdict) | Task 1 | — (modifies vocab_commands.go, conflicts with Task 4) |
| 8 (query payload) | Task 1 | Tasks 5, 6, 9 (different files) |
| 9 (resituate rider) | Task 3 | Tasks 5, 6, 8 (different files) |
| 10 (skill TDD) | Tasks 5–9 shipped | — |
| 11 (validation) | Tasks 1–9 installed | — |

**Sequential constraint:** Tasks 4 and 7 both modify `vocab_commands.go` — run them
sequentially (4 then 7) or use a git worktree to isolate and merge with `--ff-only`.

**Safe parallel bundle:** Tasks 5 + 6 + 8 + 9 can run in parallel worktrees after Task 3
ships (each touches a distinct file).

---

## Open questions

None that could not be resolved from code and docs. All paths, function names, struct fields,
and line references are verified against the working tree (2026-07-03, main branch).

**Not in this plan — executed by the /please Document step (Step 5) immediately after these
tasks ship (this build IS "recalibration lands"; named targets so nothing goes silently stale):**
- `docs/ROADMAP.md:163–172` (Track A "WRITE-SIDE SHIPPED" block) — add the integration line:
  refit lifecycle live as of this build (write-time trigger check → refit_pending →
  verdict-line/payload surfacing → autonomous learn-skill refit), recalibrated triggers named
  (growth ≥40 notes AND ≥14d / vault-wide untagged >8% / hub >25%). ROADMAP contains no trigger
  numbers to replace — this is an addition, not an edit.
- `docs/design/2026-07-03-vocab-notes-build-results.md:96–98` "What remains" — documents the
  OLD trigger set verbatim (untagged >10% of last 25 writes / term >25% / vault +30%); replace
  with the shipped set above.
- `docs/architecture/c1-system-context.md` learn flow — add a footnote: the learn/amend/resituate
  write path now runs an in-process vocab trigger check that persists `refit_pending` in
  `vocab.centroids.json` (side-effect only; surfaced via the stats verdict line + query payload
  field). No redraw — no new participant or edge.
- `docs/GLOSSARY.md` — add the `vocab.centroids.json` schema fields: `refit_pending` (bool,
  omitted when false), `refit_reason` (string), `last_refit` ({note_count int, date YYYY-MM-DD}
  baseline set at bootstrap/refit).
- `docs/design/2026-07-03-vocab-lifecycle-proposals.md` — Task 11 writes its one-line
  "Metered refit result (2026-07-03)" section into this doc (the ONLY post-landing update it
  gets).

**Deferred beyond this effort (design-doc rider 4):**
- Re-replay after 30+ days of history (30d interval column unanswerable today).
