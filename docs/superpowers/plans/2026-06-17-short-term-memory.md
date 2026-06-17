# Short-Term Memory (recency recall) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `engram query` surface the agent's recent transcript events so a post-context-loss agent re-reads its own recent narration — provenance falling out of recency.

**Architecture:** Two cooperating mechanisms in the unified query's chunk path (`mergeChunkSpace`, `internal/cli/query.go`), leaving notes' scoring and the `query-chunks` experiment target untouched. (1) **Recency re-rank** — multiply each chunk's cosine by a time-decay factor (per-source mtime from the ingest `manifest.json`) and a small turn-tail factor (`turn-N` anchor) *before* chunk items are built, so recent chunks compete with their lifted scores. (2) **Adaptive recency band** — *after* the existing merge+cap, if fewer than `floor` recent chunk items survived, backfill the deficit with recent chunk items that were cut, displacing the lowest-ranked non-recent items. Pure recency math lives in `internal/cli/recency.go` (DI clock), validated by a deterministic retrieval eval. The recall skill is updated to present recent items as "what I recently did" context.

**Tech Stack:** Go 1.26 (no CGO), `internal/cli` query pipeline, `internal/chunk` records, `manifest.json` (source→mtime), gomega assertions + `package cli_test` blackbox tests backed by `export_test.go` aliases, `targ test` / `targ check-full`, `superpowers:writing-skills` for the SKILL.md change.

**v1 scope decisions (verified; pressure-tested at Gate A):**
- Recency applies to **chunk items only**; notes keep pure cosine. Out of scope: note recency.
- The binary does **not** exclude the "live" session (verified: `/clear`/`/compact`/auto-compaction all keep the same `.jsonl`, so there is no reliable query-time live-session signal). Instead **bias toward inclusion** — surface recent chunks; the recall skill dedups against the agent's visible context. `isCompactSummary`-precise boundary detection is a tracked follow-up (Task 12), not v1.
- The band runs **post-cap on the merged note+chunk `resolvedItems`** and may displace the lowest-ranked **non-recent** items (notes included) to guarantee `floor` recent chunks — the documented "guarantee" cost.
- All tunable values are **named constants** chosen by the deterministic eval (Task 8), never bare literals.
- Recency is **skipped entirely** when `deps.Now == nil` or the manifest is unreadable → behaviour is exactly today's pure cosine (safe fallback; existing tests stay green).

---

## File Structure

| File | Responsibility | Action |
|---|---|---|
| `internal/cli/recency.go` | Pure recency funcs + named constants: `parseTurnN`, `maxTurnBySource`, `recencyMultiplier`, `sourceAgeDays`, `applyChunkRecency`, `fillRecencyBand`, `sortScoredDesc`, `defaultRecencyParams`. No I/O. | Create |
| `internal/cli/export_test.go` | Add `Export*` aliases + test constructors for the unexported recency surface. | Modify |
| `internal/cli/recency_test.go` | `package cli_test` unit tests (gomega; `t.Parallel`). | Create |
| `internal/cli/recency_eval_test.go` | `package cli_test` deterministic retrieval eval: sweep (logged) + assert tuned defaults surface the planted chunk. | Create |
| `internal/cli/query.go` | Add `Now` to `QueryDeps`+`newOsQueryDeps`; wire recency re-rank + band into `mergeChunkSpace`. | Modify |
| `internal/cli/query_test.go` | Blackbox integration test via `RunQuery`: a planted recent low-cosine chunk surfaces where pure cosine buries it. | Modify |
| `skills/recall/SKILL.md` | Present recent chunk items as the agent's own recent activity. | Modify (writing-skills) |
| `docs/architecture/c1-system-context.md` | Update the recall flow narrative for recency + band. | Modify (Gate C) |

---

## Task 1: `export_test.go` scaffolding for the recency surface

**Files:** Modify `internal/cli/export_test.go`

Add the `Export*` aliases and constructors that the blackbox tests (Tasks 2–9) use. `ExportResolvedItem` already exists; reuse it. Constructors are needed because `recencyParams` and `scoredChunk` have unexported fields.

- [ ] **Step 1: Add to the `var (...)` block in `export_test.go`**

```go
	ExportParseTurnN          = parseTurnN
	ExportMaxTurnBySource     = maxTurnBySource
	ExportRecencyMultiplier   = recencyMultiplier
	ExportSourceAgeDays       = sourceAgeDays
	ExportApplyChunkRecency   = applyChunkRecency
	ExportFillRecencyBand     = fillRecencyBand
	ExportSortScoredDesc      = sortScoredDesc
	ExportDefaultRecencyParams = defaultRecencyParams
```

- [ ] **Step 2: Add type aliases + constructors (after the existing `type Export... = ...` block)**

```go
type ExportRecencyParams = recencyParams
type ExportScoredChunk = scoredChunk

// ExportNewRecencyParams builds a recencyParams for tests.
func ExportNewRecencyParams(halfLifeDays, tailWeight float64, floor int, windowDays float64) recencyParams {
	return recencyParams{halfLifeDays: halfLifeDays, tailWeight: tailWeight, floor: floor, windowDays: windowDays}
}

// ExportNewScoredChunk builds a scoredChunk for tests.
func ExportNewScoredChunk(rec chunk.Record, score float32) scoredChunk {
	return scoredChunk{record: rec, score: score}
}

// ExportScoredChunkScore / Record expose the unexported fields for assertions.
func ExportScoredChunkScore(s scoredChunk) float32       { return s.score }
func ExportScoredChunkRecord(s scoredChunk) chunk.Record { return s.record }

// ExportNewChunkResolvedItem builds a chunk-kind resolvedItem for band tests.
// notePath mirrors mergeChunkSpace's "source#anchor" form.
func ExportNewChunkResolvedItem(notePath string, score float32) resolvedItem {
	return resolvedItem{notePath: notePath, score: score, kind: chunkItemKind}
}

// ExportResolvedItemPath / Score expose fields for assertions.
func ExportResolvedItemPath(r resolvedItem) string  { return r.notePath }
func ExportResolvedItemScore(r resolvedItem) float32 { return r.score }
```

Ensure `export_test.go` imports `"github.com/toejough/engram/internal/chunk"` (add if absent).

- [ ] **Step 3:** These reference symbols defined in later tasks; they will not compile until Task 2+ land. That is expected — `export_test.go` is edited incrementally alongside each task, OR add the aliases as each underlying symbol is created. **Practical order: add each alias in the same commit as the function it exports.** This task is the spec for the final state of those aliases.

- [ ] **Step 4: Commit** (with Task 2) — aliases land per-symbol.

---

## Task 2: `parseTurnN` — extract turn ordinal from a chunk anchor

**Files:** Create `internal/cli/recency.go`; Create `internal/cli/recency_test.go`; add the `ExportParseTurnN` alias.

- [ ] **Step 1: Write the failing test** (`internal/cli/recency_test.go`)

```go
package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestParseTurnN(t *testing.T) {
	t.Parallel()

	cases := []struct {
		anchor string
		wantN  int
		wantOK bool
	}{
		{"turn-0", 0, true},
		{"turn-42", 42, true},
		{"preamble", 0, false},
		{"Some Heading", 0, false},
		{"turn-", 0, false},
		{"turn-x", 0, false},
	}

	for _, tc := range cases {
		g := NewWithT(t)
		gotN, gotOK := cli.ExportParseTurnN(tc.anchor)
		g.Expect(gotOK).To(Equal(tc.wantOK), "ok for %q", tc.anchor)
		g.Expect(gotN).To(Equal(tc.wantN), "n for %q", tc.anchor)
	}
}
```

- [ ] **Step 2: Run** `targ test` → FAIL (`undefined: cli.ExportParseTurnN`).

- [ ] **Step 3: Implement** (`internal/cli/recency.go`)

```go
package cli

import (
	"strconv"
	"strings"
)

const turnAnchorPrefix = "turn-"

// parseTurnN extracts the turn ordinal from a "turn-N" anchor.
// Returns (0, false) for preamble/heading anchors that carry no ordinal.
func parseTurnN(anchor string) (int, bool) {
	rest, ok := strings.CutPrefix(anchor, turnAnchorPrefix)
	if !ok {
		return 0, false
	}

	n, err := strconv.Atoi(rest)
	if err != nil || n < 0 {
		return 0, false
	}

	return n, true
}
```

Add `ExportParseTurnN = parseTurnN` to `export_test.go`.

- [ ] **Step 4: Run** `targ test` → PASS.
- [ ] **Step 5: Commit** `feat(cli): parse turn-N ordinal from chunk anchors (STM)` (include recency.go, recency_test.go, export_test.go). Trailer `AI-Used: [claude]`.

---

## Task 3: `maxTurnBySource` — per-source max turn ordinal

**Files:** Modify `internal/cli/recency.go`, `internal/cli/recency_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestMaxTurnBySource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	recs := []chunk.Record{
		{Source: "a.jsonl", Anchor: "turn-0"},
		{Source: "a.jsonl", Anchor: "turn-5"},
		{Source: "a.jsonl", Anchor: "preamble"},
		{Source: "b.jsonl", Anchor: "turn-2"},
		{Source: "c.md", Anchor: "Heading"},
	}

	got := cli.ExportMaxTurnBySource(recs)

	g.Expect(got["a.jsonl"]).To(Equal(5))
	g.Expect(got["b.jsonl"]).To(Equal(2))
	_, hasC := got["c.md"]
	g.Expect(hasC).To(BeFalse())
}
```

Add imports `"github.com/toejough/engram/internal/chunk"` to the test file.

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to `recency.go`; add `"github.com/toejough/engram/internal/chunk"` import)

```go
// maxTurnBySource returns the highest turn ordinal seen per source.
// Sources with no turn anchors are absent from the map.
func maxTurnBySource(records []chunk.Record) map[string]int {
	maxBySource := make(map[string]int, len(records))

	for _, r := range records {
		n, ok := parseTurnN(r.Anchor)
		if !ok {
			continue
		}

		if cur, seen := maxBySource[r.Source]; !seen || n > cur {
			maxBySource[r.Source] = n
		}
	}

	return maxBySource
}
```

Add `ExportMaxTurnBySource = maxTurnBySource`.

- [ ] **Step 4: Run** `targ test` → PASS.
- [ ] **Step 5: Commit** `feat(cli): per-source max turn ordinal (STM)`.

---

## Task 4: `recencyMultiplier` + named constants

**Files:** Modify `internal/cli/recency.go`, `internal/cli/recency_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRecencyMultiplier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	p := cli.ExportNewRecencyParams(3, 0.2, 0, 0) // halfLife=3, tail=0.2

	g.Expect(cli.ExportRecencyMultiplier(0, 0, p)).To(BeNumerically("~", 1.0, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(3, 0, p)).To(BeNumerically("~", 0.5, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(0, 1, p)).To(BeNumerically("~", 1.2, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(6, 0, p)).To(BeNumerically("<", cli.ExportRecencyMultiplier(3, 0, p)))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to `recency.go`; add `"math"` import)

```go
// recencyParams are the tunable knobs (defaults chosen by the eval in recency_eval_test.go).
type recencyParams struct {
	halfLifeDays float64 // age at which the decay factor is 0.5
	tailWeight   float64 // extra lift for the last turn of a session (turnFrac=1)
	floor        int     // min recent chunk items the band guarantees
	windowDays   float64 // age below which a chunk counts "recent"
}

// recencyMultiplier returns exp2(-ageDays/halfLife) * (1 + tailWeight*turnFrac).
// ageDays>=0; turnFrac in [0,1]. At age 0, turnFrac 0 it is exactly 1.0.
func recencyMultiplier(ageDays, turnFrac float64, p recencyParams) float64 {
	decay := math.Exp2(-ageDays / p.halfLifeDays)

	return decay * (1 + p.tailWeight*turnFrac)
}
```

Add `ExportRecencyMultiplier`, `type ExportRecencyParams`, `ExportNewRecencyParams` to `export_test.go`.

- [ ] **Step 4: Run** `targ test` → PASS.
- [ ] **Step 5: Commit** `feat(cli): recency multiplier (decay × turn-tail) (STM)`.

---

## Task 5: `sourceAgeDays` — manifest mtime → age in days (DI clock)

**Files:** Modify `internal/cli/recency.go`, `internal/cli/recency_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test** (add `"time"` import to the test file)

```go
func TestSourceAgeDays(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	mtimes := map[string]int64{
		"recent.jsonl": now.Add(-12 * time.Hour).UnixNano(),
		"old.jsonl":    now.Add(-72 * time.Hour).UnixNano(),
		"future.jsonl": now.Add(24 * time.Hour).UnixNano(), // clamp to 0
	}

	got := cli.ExportSourceAgeDays(mtimes, now)

	g.Expect(got["recent.jsonl"]).To(BeNumerically("~", 0.5, 1e-6))
	g.Expect(got["old.jsonl"]).To(BeNumerically("~", 3.0, 1e-6))
	g.Expect(got["future.jsonl"]).To(BeNumerically("~", 0.0, 1e-6))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to `recency.go`; add `"time"` import)

```go
const hoursPerDay = 24

// sourceAgeDays converts per-source mtime (unix nanos) into age in days relative
// to now. Negative ages (clock skew / future mtime) clamp to 0.
func sourceAgeDays(mtimeBySource map[string]int64, now time.Time) map[string]float64 {
	ages := make(map[string]float64, len(mtimeBySource))

	for source, mtime := range mtimeBySource {
		age := now.Sub(time.Unix(0, mtime)).Hours() / hoursPerDay
		if age < 0 {
			age = 0
		}

		ages[source] = age
	}

	return ages
}
```

Add `ExportSourceAgeDays = sourceAgeDays`.

- [ ] **Step 4: Run** `targ test` → PASS.
- [ ] **Step 5: Commit** `feat(cli): source age in days from manifest mtime (STM)`.

---

## Task 6: `applyChunkRecency` — re-score chunks; `sortScoredDesc`

**Files:** Modify `internal/cli/recency.go`, `internal/cli/recency_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test** — a recent low-cosine chunk overtakes a stale high-cosine chunk.

```go
func TestApplyChunkRecencyLiftsRecentOverStaleHighCosine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunk(chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.80),
		cli.ExportNewScoredChunk(chunk.Record{Source: "recent.jsonl", Anchor: "turn-9"}, 0.45),
	}
	ages := map[string]float64{"old.jsonl": 90, "recent.jsonl": 0.01}
	maxTurn := map[string]int{"old.jsonl": 3, "recent.jsonl": 9}
	p := cli.ExportNewRecencyParams(3, 0.2, 0, 1)

	out := cli.ExportApplyChunkRecency(scored, ages, maxTurn, p)

	g.Expect(cli.ExportScoredChunkScore(out[1])).To(BeNumerically(">", cli.ExportScoredChunkScore(out[0])))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to `recency.go`; add `"sort"` import)

```go
// applyChunkRecency returns a copy of scored with each score multiplied by its
// recency factor. turnFrac = turnN / maxTurn(source); 0 when the source has no
// turn anchors. Sources absent from ages (e.g. never-swept) are treated as age 0
// (maximally recent) so a freshly written but not-yet-manifested source is not
// penalised.
func applyChunkRecency(
	scored []scoredChunk,
	ageDaysBySource map[string]float64,
	maxTurnBySource map[string]int,
	p recencyParams,
) []scoredChunk {
	out := make([]scoredChunk, len(scored))

	for i, s := range scored {
		age := ageDaysBySource[s.record.Source] // missing → 0.0

		turnFrac := 0.0
		if n, ok := parseTurnN(s.record.Anchor); ok {
			if maxN := maxTurnBySource[s.record.Source]; maxN > 0 {
				turnFrac = float64(n) / float64(maxN)
			}
		}

		out[i] = scoredChunk{
			record: s.record,
			score:  s.score * float32(recencyMultiplier(age, turnFrac, p)),
		}
	}

	return out
}

// sortScoredDesc sorts in place by descending score (stable).
func sortScoredDesc(scored []scoredChunk) {
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
}
```

Add `ExportApplyChunkRecency`, `ExportSortScoredDesc`, `ExportNewScoredChunk`, `ExportScoredChunkScore`, `ExportScoredChunkRecord`, `type ExportScoredChunk`.

- [ ] **Step 4: Run** `targ test` → PASS.
- [ ] **Step 5: Commit** `feat(cli): recency re-rank of chunk scores (STM)`.

---

## Task 7: `fillRecencyBand` — adaptive deficit backfill over resolved items

**Files:** Modify `internal/cli/recency.go`, `internal/cli/recency_test.go`, `export_test.go`

Operates on the **merged, capped** `[]resolvedItem`. `recentPool` is the recency-ordered chunk items (newest first) the caller built from recent chunks. Membership is by `notePath`. If fewer than `floor` recent items are present, append the missing ones (in pool order), displacing the lowest-ranked **non-recent** items, capped at `limit`.

- [ ] **Step 1: Write the failing tests**

```go
func TestFillRecencyBandBackfillsDeficit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// capped items: all stale notes/chunks; recentPool has 2 recent chunk items not present.
	items := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-1", 0.9),
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-2", 0.8),
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-3", 0.7),
	}
	recentPool := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-9", 0.30),
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-8", 0.20),
	}

	out := cli.ExportFillRecencyBand(items, recentPool, 2, len(items))

	g.Expect(out).To(HaveLen(len(items))) // budget preserved
	paths := map[string]bool{}
	for _, it := range out {
		paths[cli.ExportResolvedItemPath(it)] = true
	}
	g.Expect(paths["recent.jsonl#turn-9"]).To(BeTrue())
	g.Expect(paths["recent.jsonl#turn-8"]).To(BeTrue())
	g.Expect(paths["old.jsonl#turn-1"]).To(BeTrue()) // highest-ranked stale retained
}

func TestFillRecencyBandNoDeficitNoChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-9", 0.9),
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-8", 0.8),
	}
	recentPool := items // both already present and recent

	out := cli.ExportFillRecencyBand(items, recentPool, 2, len(items))
	g.Expect(out).To(Equal(items))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to `recency.go`)

```go
// fillRecencyBand guarantees at least floor of recentPool's items appear in the
// returned slice of length <= limit. recentPool is the recency-ordered (newest
// first) chunk items the caller deemed "recent". Items already present count
// toward the floor; the deficit is filled from recentPool (skipping those
// already present), displacing the lowest-ranked items NOT in recentPool. No-op
// when the floor is already met or recentPool is empty.
func fillRecencyBand(items, recentPool []resolvedItem, floor, limit int) []resolvedItem {
	recentKey := make(map[string]bool, len(recentPool))
	for _, r := range recentPool {
		recentKey[r.notePath] = true
	}

	present := make(map[string]bool, len(items))
	have := 0

	for _, it := range items {
		present[it.notePath] = true
		if recentKey[it.notePath] {
			have++
		}
	}

	deficit := floor - have
	if deficit <= 0 {
		return items
	}

	missing := make([]resolvedItem, 0, deficit)

	for _, r := range recentPool {
		if len(missing) >= deficit {
			break
		}

		if !present[r.notePath] {
			missing = append(missing, r)
		}
	}

	if len(missing) == 0 {
		return items
	}

	return spliceRecent(items, missing, recentKey, limit)
}

// spliceRecent prepends the missing recent items, then refills from the original
// items dropping the lowest-ranked NON-recent ones first, capped at limit.
func spliceRecent(items, missing []resolvedItem, recentKey map[string]bool, limit int) []resolvedItem {
	out := make([]resolvedItem, 0, limit)
	out = append(out, missing...)

	// keep recent items from the original first, then non-recent, in original order.
	for _, it := range items {
		if len(out) >= limit {
			break
		}

		if recentKey[it.notePath] {
			out = append(out, it)
		}
	}

	for _, it := range items {
		if len(out) >= limit {
			break
		}

		if !recentKey[it.notePath] {
			out = append(out, it)
		}
	}

	return out
}
```

Add `ExportFillRecencyBand`, `ExportNewChunkResolvedItem`, `ExportResolvedItemPath`, `ExportResolvedItemScore`.

> Note: prepending `missing` then re-appending the original recent items could duplicate if a recent item were both missing and present — impossible here because `missing` only holds `!present` items. The two original-item loops preserve "recent kept over non-recent" displacement; dedup is guaranteed by `present`/`missing` disjointness.

- [ ] **Step 4: Run** `targ test` → PASS (both tests).
- [ ] **Step 5: Commit** `feat(cli): adaptive recency band over resolved items (STM)`.

---

## Task 8: Deterministic retrieval eval — TUNE and VALIDATE (enforced)

**Files:** Create `internal/cli/recency_eval_test.go`; add `defaultRecencyParams` + named constants to `recency.go`.

The tuning is **enforced**, not advisory: the test sweeps params, logs each cell's planted rank, and asserts the chosen `defaultRecencyParams()` puts the planted chunk in the **top 5**. The chosen sweep cell is recorded in a comment in `recency.go`.

- [ ] **Step 1: Add `defaultRecencyParams` with NAMED constants** (append to `recency.go`)

```go
// Tuned recency defaults — chosen from the sweep in recency_eval_test.go.
// Chosen cell (recorded after Step 4): halfLife=3d, floor=3 (planted rank reaches
// the top-5 while keeping topical hits). Revisit via the eval, never by feel.
const (
	defaultHalfLifeDays     = 3.0
	defaultTailWeight       = 0.2
	defaultRecencyFloor     = 3
	defaultRecentWindowDays = 1.0
)

func defaultRecencyParams() recencyParams {
	return recencyParams{
		halfLifeDays: defaultHalfLifeDays,
		tailWeight:   defaultTailWeight,
		floor:        defaultRecencyFloor,
		windowDays:   defaultRecentWindowDays,
	}
}
```

Add `ExportDefaultRecencyParams = defaultRecencyParams`.

- [ ] **Step 2: Write the eval** (`internal/cli/recency_eval_test.go`, `package cli_test`)

```go
package cli_test

import (
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

const evalPlantedHash = "sha256:planted"

func buildSyntheticPool(n int) ([]cli.ExportScoredChunk, map[string]float64, map[string]int) {
	planted := chunk.Record{
		Source: "recent.jsonl", Anchor: "turn-40",
		Text: "ASSISTANT: I'll file issue #644 for the recall flakiness", ContentHash: evalPlantedHash,
	}
	pool := []cli.ExportScoredChunk{cli.ExportNewScoredChunk(planted, 0.42)}
	ages := map[string]float64{"recent.jsonl": 0.01}
	maxTurn := map[string]int{"recent.jsonl": 40}

	for i := range n {
		rec := chunk.Record{Source: "old.jsonl", Anchor: "turn-" + strconv.Itoa(i), ContentHash: "sha256:old" + strconv.Itoa(i)}
		pool = append(pool, cli.ExportNewScoredChunk(rec, 0.55+0.003*float32(i))) // all beat planted cosine
		ages["old.jsonl"] = 90
		maxTurn["old.jsonl"] = 200
	}

	return pool, ages, maxTurn
}

// plantedRank returns the 0-based rank of the planted chunk after recency re-rank
// + cap + band, or -1. Mirrors the mergeChunkSpace ordering at the chunk level.
func plantedRank(pool []cli.ExportScoredChunk, ages map[string]float64, maxTurn map[string]int, p cli.ExportRecencyParams, limit int) int {
	scored := cli.ExportApplyChunkRecency(pool, ages, maxTurn, p)
	cli.ExportSortScoredDesc(scored)

	// to resolvedItems (chunk kind), cap, then band over the recent pool.
	items := make([]cli.ExportResolvedItem, 0, len(scored))
	var recentPool []cli.ExportResolvedItem
	for _, s := range scored {
		rec := cli.ExportScoredChunkRecord(s)
		it := cli.ExportNewChunkResolvedItem(rec.Source+"#"+rec.Anchor, cli.ExportScoredChunkScore(s))
		items = append(items, it)
		if ages[rec.Source] <= cli.ExportRecencyWindowDays(p) {
			recentPool = append(recentPool, it)
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	items = cli.ExportFillRecencyBand(items, recentPool, cli.ExportRecencyFloor(p), limit)

	for i, it := range items {
		if cli.ExportResolvedItemPath(it) == "recent.jsonl#turn-40" {
			return i
		}
	}
	return -1
}

func TestRecencyEvalSweepAndValidateDefaults(t *testing.T) {
	t.Parallel()
	pool, ages, maxTurn := buildSyntheticPool(40)
	const limit = 20

	for _, hl := range []float64{1, 3, 7, 14} {
		for _, fl := range []int{0, 1, 3} {
			p := cli.ExportNewRecencyParams(hl, 0.2, fl, 1)
			t.Logf("halfLife=%4.0f floor=%d -> plantedRank=%d", hl, fl, plantedRank(pool, ages, maxTurn, p, limit))
		}
	}

	g := NewWithT(t)
	rank := plantedRank(pool, ages, maxTurn, cli.ExportDefaultRecencyParams(), limit)
	g.Expect(rank).To(BeNumerically(">=", 0), "planted chunk must surface")
	g.Expect(rank).To(BeNumerically("<=", 5), "tuned defaults must put planted narration in the top 6 (got rank %d)", rank)
}
```

This needs two tiny exported accessors for `recencyParams.windowDays`/`floor` (the eval reads them to mirror the band). They MUST be exported — `recency_eval_test.go` is `package cli_test` and cannot see unexported helpers in `export_test.go` (`package cli`). Add to `export_test.go`:

```go
func ExportRecencyWindowDays(p recencyParams) float64 { return p.windowDays }
func ExportRecencyFloor(p recencyParams) int          { return p.floor }
```

- [ ] **Step 3: Run** `targ test` → FAIL until defaults are confirmed.

- [ ] **Step 4: Run the sweep, READ the log, set the winning cell** (MANDATORY — not advisory)

Run: `targ test -- -run TestRecencyEvalSweepAndValidateDefaults -v`
Read the logged `plantedRank` per cell. **Update the `const` block in `recency.go` to the cell with the lowest planted rank that still respects topical hits, and update the "Chosen cell" comment to quote the actual log line.** Re-run until the `<= 5` assertion passes with the committed defaults. Do not commit advisory/guessed defaults.

- [ ] **Step 5: Commit** `test(cli): deterministic recency eval + tuned defaults (STM)`.

---

## Task 9: Wire recency into the unified query (`mergeChunkSpace`)

**Files:** Modify `internal/cli/query.go`, `internal/cli/query_test.go`

- [ ] **Step 1: Write the failing blackbox integration test** (`query_test.go`, `package cli_test`)

Construct `cli.QueryDeps` (exported) with: `Scan` → returns no notes; `ListChunkIndexes` → returns one index path; `Read` → serves the index `.jsonl` (one planted recent low-cosine chunk + several stale higher-cosine chunks) and a `manifest.json` mapping `recent.jsonl` mtime=now, `old.jsonl` mtime=120d ago; `Embedder` → the deterministic stub used by existing `query_test.go` chunk tests (reuse that fixture; it returns a fixed vector per phrase so cosine is controllable); `Now` → fixed time. Call `cli.RunQuery(ctx, args, deps, &buf)`. Assert `buf.String()` contains `recent.jsonl#turn-40` (the planted narration surfaced) — which it would NOT under pure cosine.

(Model the deps/stub-embedder exactly on the existing chunk-merge test in `internal/cli/query_test.go`; add the `manifest.json` response to that test's `Read` switch and set `Now`.)

- [ ] **Step 2: Run** `targ test` → FAIL (planted chunk absent; `QueryDeps` has no `Now`).

- [ ] **Step 3: Add `Now` to `QueryDeps`** (query.go ~line 39) and wire it in `newOsQueryDeps` (~line 1532):

```go
	// Now supplies the query-time clock for recency (DI; production = time.Now).
	Now func() time.Time
```
In `newOsQueryDeps`: add `Now: time.Now,`. **Add `"time"` to query.go's import block — it is NOT currently imported** (verified: `grep -c '"time"' internal/cli/query.go` = 0). Without it both `Now func() time.Time` and `time.Now` fail to compile.

- [ ] **Step 4: Replace the body of `mergeChunkSpace` (query.go:1245-1261) — exact diff**

CURRENT (lines 1245-1261):
```go
	for _, s := range scored {
		merged.resolvedItems = append(merged.resolvedItems, resolvedItem{
			notePath:    s.record.Source + "#" + s.record.Anchor,
			content:     s.record.Text,
			score:       s.score,
			provenances: []string{provenanceDirect},
			kind:        chunkItemKind,
		})
	}

	sort.SliceStable(merged.resolvedItems, func(i, j int) bool {
		return merged.resolvedItems[i].score > merged.resolvedItems[j].score
	})

	if len(merged.resolvedItems) > limit {
		merged.resolvedItems = merged.resolvedItems[:limit]
	}
```

REPLACE WITH:
```go
	// Recency re-rank (chunk-only): lift recent chunks before they compete with notes.
	var recentPool []resolvedItem

	if deps.Now != nil {
		ages := chunkSourceAges(args.ChunksDir, deps)
		if ages != nil {
			params := defaultRecencyParams()
			scored = applyChunkRecency(scored, ages, maxTurnBySource(records), params)
			sortScoredDesc(scored)
			recentPool = recentChunkItems(scored, ages, params.windowDays)
		}
	}

	for _, s := range scored {
		merged.resolvedItems = append(merged.resolvedItems, resolvedItem{
			notePath:    s.record.Source + "#" + s.record.Anchor,
			content:     s.record.Text,
			score:       s.score,
			provenances: []string{provenanceDirect},
			kind:        chunkItemKind,
		})
	}

	sort.SliceStable(merged.resolvedItems, func(i, j int) bool {
		return merged.resolvedItems[i].score > merged.resolvedItems[j].score
	})

	if len(merged.resolvedItems) > limit {
		merged.resolvedItems = merged.resolvedItems[:limit]
	}

	// Adaptive band: guarantee a floor of recent chunk items survived the cap.
	if recentPool != nil {
		merged.resolvedItems = fillRecencyBand(merged.resolvedItems, recentPool, defaultRecencyFloor, limit)
	}
```

- [ ] **Step 5: Add the two small helpers** (in `recency.go` or near `mergeChunkSpace` in query.go)

```go
// chunkSourceAges reads the chunks-dir manifest and returns source→ageDays,
// or nil when the manifest is unreadable (→ recency skipped, pure cosine).
func chunkSourceAges(chunksDir string, deps QueryDeps) map[string]float64 {
	manifest, err := readManifest(chunksDir, IngestDeps{ReadFile: deps.Read})
	if err != nil {
		return nil
	}

	mtimes := make(map[string]int64, len(manifest))
	for src, entry := range manifest {
		mtimes[src] = entry.MtimeUnixNano
	}

	return sourceAgeDays(mtimes, deps.Now())
}

// recentChunkItems builds the recency-ordered (newest first) chunk resolvedItems
// whose source age is within windowDays — the band's backfill pool.
func recentChunkItems(scored []scoredChunk, ages map[string]float64, windowDays float64) []resolvedItem {
	var pool []resolvedItem

	for _, s := range scored {
		if age, ok := ages[s.record.Source]; ok && age <= windowDays {
			pool = append(pool, resolvedItem{
				notePath:    s.record.Source + "#" + s.record.Anchor,
				content:     s.record.Text,
				score:       s.score,
				provenances: []string{provenanceDirect},
				kind:        chunkItemKind,
			})
		}
	}

	return pool
}
```

(`scored` is already sorted newest-relevant first by `sortScoredDesc`; the pool inherits that order, which is "newest/highest-recency-score first" — acceptable for backfill.)

- [ ] **Step 6: Run** `targ test` → PASS (new integration test + all existing query tests, which have `Now == nil` → pure cosine, unchanged).
- [ ] **Step 7: Run** `targ check-full` → no lint/coverage regressions. Fix any.
- [ ] **Step 8: Commit** `feat(cli): recency-aware chunk ranking + band in unified query (STM)`.

---

## Task 10: Recall skill consumption (REQUIRED SUB-SKILL: superpowers:writing-skills)

**Files:** Modify `skills/recall/SKILL.md`

- [ ] **Step 1:** Invoke `superpowers:writing-skills`. Write the RED baseline scenario: a recall payload whose top item is a first-person chunk (`ASSISTANT: I'll file issue #644…`, anchor `turn-N`, recent source). Baseline behaviour: the agent treats it as a stranger's note / expresses surprise. Capture that as the failing behaviour.
- [ ] **Step 2:** Edit a short subsection into the recall synthesis step: "**Recent items are your own recent activity.** Chunk items from recent transcript turns (first-person `ASSISTANT:` narration, `turn-N` anchors) are work *you* did in a just-prior or pre-clear session. Treat them as your own actions — do not re-derive them or express surprise — and dedup against what is already in your context."
- [ ] **Step 3:** GREEN — verify the behavioural change against the baseline; run the skill's pressure tests.
- [ ] **Step 4:** Propagate via `engram update` if that is the repo's deploy step for skills. **Commit** `feat(recall): consume recency band as own recent activity (STM)`.

---

## Task 11: Docs (Gate C)

**Files:** `docs/architecture/c1-system-context.md` (the "Flow: recall" narrative, ~lines 70-86 — grep the `### Flow: recall` heading, don't hardcode line numbers).

- [ ] Update the recall flow prose: after "top-k cosine", note "recency re-rank of chunk hits (manifest mtime + turn-N decay) and an adaptive recency band guaranteeing a floor of recent chunk items." Keep it source-accurate; cite `mergeChunkSpace`. Gate C reviews this and any other doc the change touched. (Linking the brainstorm doc to the chosen build is optional polish — include only if Gate C's relevance reviewer wants it.)

---

## Task 12: Tracked follow-up issue (value proof)

**Files:** none (gh)

- [ ] Confirm the dependency issues exist: `gh issue view 642` and `gh issue view 643` (both OPEN as of this session).
- [ ] `gh issue create` — title "e2e value-proof for short-term memory (recency recall)"; body: acceptance = measure agent cost + correctness on a **context-loss continuity task with a real dead-end** (recalling recent work prevents a costly re-do), with vs without recency recall, multiple trials; depends on #642 (cold-warm orchestration + `SuccessCmd` correctness gate) and #643 (headless-learn first-run marker); references this plan and `docs/superpowers/research/2026-06-16-short-term-memory-brainstorm.md`.

---

## Self-Review

- **Spec coverage:** #1 recency re-rank → Tasks 4,6,9. #2 adaptive band → Tasks 7,9. Signals (turn-N, mtime) → Tasks 2,3,5. Tune+validate (enforced) → Task 8. Skill → Task 10. Docs → Task 11. Value-proof tracked → Task 12. ✓
- **Blocker fixes confirmed:** band operates on `[]resolvedItem` post-cap (Task 7/9), not on mistyped `top` — type-correct. Tests are `package cli_test` + `export_test.go` aliases (Task 1). `mergeChunkSpace` change is an exact before/after diff against the real body (Task 9). ✓
- **Standards:** named constants `defaultHalfLifeDays`… (no bare literals); `targ test` everywhere (no `go test`); `maxBySource` (typo fixed); `t.Parallel` on every test; DI clock injected, manifest via injected `deps.Read`; SKILL.md via writing-skills (Task 10). ✓
- **Tuning enforced:** Task 8 asserts planted rank ≤ 5 with the committed defaults and mandates recording the winning sweep cell — not advisory. ✓
- **Fallback:** `deps.Now == nil` or unreadable manifest → recency skipped → existing behaviour + tests preserved. ✓
- **Type consistency:** `recencyParams{halfLifeDays,tailWeight,floor,windowDays}`, `scoredChunk{record,score}`, `resolvedItem{notePath,content,score,provenances,kind}`, `chunkItemKind`, `readManifest`/`IngestDeps`/`manifestEntry.MtimeUnixNano`, and all `Export*` aliases used consistently. ✓
