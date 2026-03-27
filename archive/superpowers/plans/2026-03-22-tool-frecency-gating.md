# Tool-Call Frecency Gating Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce memory surfacing noise by skipping non-Bash tools and applying logarithmic frequency-based probability gating for Bash commands.

**Architecture:** Shell hooks gate non-Bash tools at the script level (zero overhead). For Bash, the Go binary extracts a command key (first two tokens), maintains a persistent counter file, and rolls a probability check before loading any memories. Random source is injected for testability.

**Tech Stack:** Go, bash, JSON state file

**Spec:** `docs/superpowers/specs/2026-03-21-tool-frecency-gating-design.md`

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/toolgate/toolgate.go` | Command key extraction, probability computation, Gate with CounterStore DI |
| Create | `internal/toolgate/toolgate_test.go` | Unit tests for all toolgate logic |
| Modify | `internal/surface/surface.go:695-743` | Call toolgate at top of `runTool()` to short-circuit |
| Modify | `internal/surface/surface_test.go` | Add test for toolgate integration in runTool |
| Modify | `hooks/pre-tool-use.sh:38-44` | Add Bash-only guard after engram plumbing filter |
| Modify | `hooks/post-tool-use.sh:15-21` | Add Bash-only guard after Write/Edit advisory block |
| Modify | `hooks/post-tool-use-failure.sh:42-48` | Wrap only memory surfacing block in Bash-only guard |

---

### Task 1: Command Key Extraction

**Files:**
- Create: `internal/toolgate/toolgate.go`
- Create: `internal/toolgate/toolgate_test.go`

- [ ] **Step 1: Write failing tests for CommandKey**

```go
// internal/toolgate/toolgate_test.go
package toolgate_test

import (
	"testing"

	"engram/internal/toolgate"

	. "github.com/onsi/gomega"
)

func TestCommandKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{name: "two tokens subcommand", cmd: "go test ./...", want: "go test"},
		{name: "targ subcommand", cmd: "targ check-full", want: "targ check-full"},
		{name: "flag second token dropped", cmd: "grep -r foo src/", want: "grep"},
		{name: "leading env var stripped", cmd: "FOO=bar git push origin main", want: "git push"},
		{name: "multiple env vars stripped", cmd: "A=1 B=2 npm install", want: "npm install"},
		{name: "single token command", cmd: "ls", want: "ls"},
		{name: "flag only second token", cmd: "ls -la", want: "ls"},
		{name: "empty string", cmd: "", want: ""},
		{name: "whitespace only", cmd: "   ", want: ""},
		{name: "env var only", cmd: "FOO=bar", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(toolgate.CommandKey(tt.cmd)).To(Equal(tt.want))
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestCommandKey ./internal/toolgate/...`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement CommandKey**

```go
// internal/toolgate/toolgate.go
package toolgate

import "strings"

// CommandKey extracts a stable identity from a bash command string.
// Strips leading VAR=val env assignments, takes first two tokens,
// drops the second token if it starts with "-" (flags aren't identity).
func CommandKey(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}

	// Strip leading env var assignments (contain '=' but not as first char).
	for len(fields) > 0 && strings.Contains(fields[0], "=") && !strings.HasPrefix(fields[0], "=") {
		fields = fields[1:]
	}

	if len(fields) == 0 {
		return ""
	}

	if len(fields) == 1 {
		return fields[0]
	}

	// Drop second token if it's a flag.
	if strings.HasPrefix(fields[1], "-") {
		return fields[0]
	}

	return fields[0] + " " + fields[1]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestCommandKey ./internal/toolgate/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/toolgate/toolgate.go internal/toolgate/toolgate_test.go
git commit -m "feat(toolgate): add command key extraction for bash frecency gating"
```

---

### Task 2: Probability Computation

**Files:**
- Modify: `internal/toolgate/toolgate.go`
- Modify: `internal/toolgate/toolgate_test.go`

- [ ] **Step 1: Write failing tests for SurfaceProbability**

```go
func TestSurfaceProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// count 0 → 1.0
	g.Expect(toolgate.SurfaceProbability(0)).To(BeNumerically("~", 1.0, 0.001))

	// count 1 → 1/(1+ln(2)) ≈ 0.59
	g.Expect(toolgate.SurfaceProbability(1)).To(BeNumerically("~", 0.59, 0.01))

	// count 10 → 1/(1+ln(11)) ≈ 0.29
	g.Expect(toolgate.SurfaceProbability(10)).To(BeNumerically("~", 0.29, 0.01))

	// count 100 → 1/(1+ln(101)) ≈ 0.18
	g.Expect(toolgate.SurfaceProbability(100)).To(BeNumerically("~", 0.18, 0.01))

	// monotonically decreasing
	prev := toolgate.SurfaceProbability(0)
	for _, c := range []int{1, 2, 5, 10, 50, 100, 1000} {
		p := toolgate.SurfaceProbability(c)
		g.Expect(p).To(BeNumerically("<", prev), "probability should decrease with count")
		prev = p
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestSurfaceProbability ./internal/toolgate/...`
Expected: FAIL — function not defined

- [ ] **Step 3: Implement SurfaceProbability**

```go
import "math"

// SurfaceProbability computes the probability of surfacing memories for a
// command that has been called count times. Uses smooth logarithmic decay:
// P = 1 / (1 + ln(1 + count)).
func SurfaceProbability(count int) float64 {
	return 1.0 / (1.0 + math.Log(1.0+float64(count)))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestSurfaceProbability ./internal/toolgate/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/toolgate/toolgate.go internal/toolgate/toolgate_test.go
git commit -m "feat(toolgate): add logarithmic surfacing probability function"
```

---

### Task 3: Persistent Counter (Read/Write with DI)

**Files:**
- Modify: `internal/toolgate/toolgate.go`
- Modify: `internal/toolgate/toolgate_test.go`

- [ ] **Step 1: Write failing tests for Gate (counter + probability roll)**

Tests use an in-memory `CounterStore` stub to satisfy DI — no `os.*` calls in `internal/`.

```go
// stubStore is an in-memory CounterStore for testing.
type stubStore struct {
	data map[string]toolgate.CounterEntry
}

func newStubStore() *stubStore {
	return &stubStore{data: make(map[string]toolgate.CounterEntry)}
}

func (s *stubStore) Load() (map[string]toolgate.CounterEntry, error) {
	// Return a copy to avoid aliasing.
	out := make(map[string]toolgate.CounterEntry, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out, nil
}

func (s *stubStore) Save(entries map[string]toolgate.CounterEntry) error {
	s.data = make(map[string]toolgate.CounterEntry, len(entries))
	for k, v := range entries {
		s.data[k] = v
	}
	return nil
}

func TestGate_FirstCall_AlwaysSurfaces(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	// Random source always returns 0.5 — should surface since P(0) = 1.0.
	gate := toolgate.NewGate(store, func() float64 { return 0.5 })

	shouldSurface, err := gate.Check("go test")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeTrue())
}

func TestGate_HighCount_SkipsWhenRollExceedsProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	gate := toolgate.NewGate(store, func() float64 { return 0.5 })

	// Call 100 times to build up the count.
	for range 100 {
		_, err := gate.Check("grep")
		g.Expect(err).NotTo(HaveOccurred())
	}

	// P(100) ≈ 0.18, roll of 0.5 > 0.18 → should skip.
	shouldSurface, err := gate.Check("grep")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeFalse())
}

func TestGate_HighCount_SurfacesWhenRollBelowProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	// Roll of 0.1 — always below any probability (min ~0.18).
	gate := toolgate.NewGate(store, func() float64 { return 0.1 })

	for range 100 {
		_, err := gate.Check("grep")
		g.Expect(err).NotTo(HaveOccurred())
	}

	shouldSurface, err := gate.Check("grep")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeTrue())
}

func TestGate_SeparateKeys_IndependentCounters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	gate := toolgate.NewGate(store, func() float64 { return 0.5 })

	// Build up "grep" count.
	for range 100 {
		_, err := gate.Check("grep")
		g.Expect(err).NotTo(HaveOccurred())
	}

	// "go test" is fresh — should surface.
	shouldSurface, err := gate.Check("go test")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeTrue())
}
```

Note: Persistence across instances is a property of the `FileCounterStore` (wired in Task 5), not the `Gate` itself. The Gate tests use the in-memory stub and verify the counter/probability logic.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestGate ./internal/toolgate/...`
Expected: FAIL — NewGate not defined

- [ ] **Step 3: Implement Gate**

```go
import (
	"fmt"
	"time"
)

// CounterEntry tracks call frequency for a single command key.
type CounterEntry struct {
	Count int       `json:"count"`
	Last  time.Time `json:"last"`
}

// CounterStore abstracts persistent storage of tool call counters.
// Implementations handle serialization, file I/O, and atomicity.
type CounterStore interface {
	Load() (map[string]CounterEntry, error)
	Save(map[string]CounterEntry) error
}

// Gate decides whether to surface memories for a given command,
// based on persistent call frequency.
type Gate struct {
	store  CounterStore
	randFn func() float64 // injected: returns [0, 1)
}

// NewGate creates a Gate. store handles persistence, randFn provides randomness.
func NewGate(store CounterStore, randFn func() float64) *Gate {
	return &Gate{store: store, randFn: randFn}
}

// Check reads the current count for key, rolls the probability gate,
// then increments and persists the counter. Returns true if surfacing
// should proceed.
func (g *Gate) Check(key string) (bool, error) {
	counters, err := g.store.Load()
	if err != nil {
		return true, nil // fail-open: surface on read error
	}

	entry := counters[key]
	prob := SurfaceProbability(entry.Count)
	shouldSurface := g.randFn() < prob

	// Increment after roll.
	entry.Count++
	entry.Last = time.Now()
	counters[key] = entry

	if saveErr := g.store.Save(counters); saveErr != nil {
		return shouldSurface, fmt.Errorf("toolgate save: %w", saveErr)
	}

	return shouldSurface, nil
}
```

The concrete `FileCounterStore` implementation (using `os.ReadFile`, `os.WriteFile`, atomic rename) is wired in `internal/cli/cli.go` at the edges — not in `internal/toolgate/`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestGate ./internal/toolgate/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/toolgate/toolgate.go internal/toolgate/toolgate_test.go
git commit -m "feat(toolgate): add persistent counter with probability gating"
```

---

### Task 4: Integrate Gate into Surfacer.runTool()

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test for toolgate integration**

Add to `internal/surface/surface_test.go`. Tests use `fakeRetriever` (existing test helper) with a memory that would match the tool input, so the gate is the only thing preventing surfacing:

```go
// T-frecency-gate: runTool short-circuits when toolgate says skip.
func TestRunTool_ToolGateSkips(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A retriever that returns a memory matching "grep" keyword.
	retriever := &fakeRetriever{memories: []*memory.Stored{{
		FilePath: "grep-memory.toml",
		Record: memory.Record{
			Title:    "grep memory",
			Content:  "always use ripgrep",
			Keywords: []string{"grep"},
		},
	}}}

	// Gate that always says "skip".
	alwaysSkip := &stubToolGate{shouldSurface: false}
	surfacer := surface.New(
		retriever,
		surface.WithToolGate(alwaysSkip),
	)

	var buf bytes.Buffer
	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		ToolName:  "Bash",
		ToolInput: `{"command":"grep foo"}`,
		DataDir:   t.TempDir(),
		Format:    "json",
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Empty result — gate blocked surfacing despite matching memory.
	g.Expect(buf.String()).To(Equal(""))
}

// T-frecency-gate-2: runTool short-circuits for non-Bash tool names.
func TestRunTool_NonBashSkips(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Retriever with a matchable memory — proves the non-Bash guard is what blocks it.
	retriever := &fakeRetriever{memories: []*memory.Stored{{
		FilePath: "grep-memory.toml",
		Record: memory.Record{
			Title:    "grep memory",
			Content:  "always use ripgrep",
			Keywords: []string{"grep"},
		},
	}}}

	surfacer := surface.New(retriever)

	var buf bytes.Buffer
	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		ToolName:  "Grep",
		ToolInput: `{"pattern":"foo"}`,
		DataDir:   t.TempDir(),
		Format:    "json",
	})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(buf.String()).To(Equal(""))
}

type stubToolGate struct {
	shouldSurface bool
}

func (s *stubToolGate) Check(_ string) (bool, error) {
	return s.shouldSurface, nil
}
```

Note: Check that `fakeRetriever` (existing in surface_test.go) supports a `memories` field or adapt accordingly.

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run "TestRunTool_ToolGate|TestRunTool_NonBash" ./internal/surface/...`
Expected: FAIL — WithToolGate not defined

- [ ] **Step 3: Implement integration**

Add to `internal/surface/surface.go`:

1. Add `ToolGater` interface and field on Surfacer:

```go
// ToolGater decides whether to surface memories for a tool call.
type ToolGater interface {
	Check(commandKey string) (bool, error)
}
```

Add `toolGate ToolGater` field to `Surfacer` struct.

2. Add option constructor:

```go
// WithToolGate sets the tool frecency gate.
func WithToolGate(gate ToolGater) SurfacerOption {
	return func(s *Surfacer) { s.toolGate = gate }
}
```

3. Add `ExtractBashCommand` to `internal/toolgate/toolgate.go` (keeps feature self-contained):

```go
import "encoding/json"

// ExtractBashCommand extracts the .command field from Bash tool input JSON.
func ExtractBashCommand(toolInput string) string {
	var input struct {
		Command string `json:"command"`
	}
	if jsonErr := json.Unmarshal([]byte(toolInput), &input); jsonErr != nil {
		return ""
	}
	return input.Command
}
```

4. Add non-Bash guard and gate check at top of `runTool()`:

```go
func (s *Surfacer) runTool(...) (Result, []*memory.Stored, []SuppressionEvent, error) {
	// Defense-in-depth: non-Bash tools should not reach here (shell filters first).
	if opts.ToolName != "Bash" {
		return Result{}, nil, nil, nil
	}

	// Frecency gate: extract command key, check counter, maybe skip.
	if s.toolGate != nil {
		key := toolgate.CommandKey(toolgate.ExtractBashCommand(opts.ToolInput))
		if key != "" {
			shouldSurface, _ := s.toolGate.Check(key)
			if !shouldSurface {
				return Result{}, nil, nil, nil
			}
		}
	}

	// ... existing code continues ...
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run "TestRunTool_ToolGate|TestRunTool_NonBash" ./internal/surface/...`
Expected: PASS

- [ ] **Step 5: Run full surface test suite**

Run: `targ test -- ./internal/surface/...`
Expected: PASS (no regressions)

- [ ] **Step 6: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): integrate toolgate into runTool for frecency gating"
```

---

### Task 5: Wire Gate in CLI

**Files:**
- Modify: `internal/cli/cli.go` (in `runSurface` function where Surfacer is constructed)

- [ ] **Step 1: Read `runSurface` to find where Surfacer is wired**

Read `internal/cli/cli.go` and find the `surface.New(...)` call to understand where options are passed.

- [ ] **Step 2: Add `FileCounterStore` and wire the gate**

Add `FileCounterStore` to `internal/cli/cli.go` (concrete I/O lives at the edges, not in `internal/toolgate/`):

```go
import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"

	"engram/internal/toolgate"
)

// fileCounterStore implements toolgate.CounterStore with atomic file I/O.
type fileCounterStore struct {
	path string
}

func newFileCounterStore(dataDir string) *fileCounterStore {
	return &fileCounterStore{path: filepath.Join(dataDir, "tool-frecency.json")}
}

func (s *fileCounterStore) Load() (map[string]toolgate.CounterEntry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]toolgate.CounterEntry), nil
		}
		return nil, fmt.Errorf("toolgate load: %w", err)
	}

	var counters map[string]toolgate.CounterEntry
	if jsonErr := json.Unmarshal(data, &counters); jsonErr != nil {
		return make(map[string]toolgate.CounterEntry), nil
	}
	return counters, nil
}

func (s *fileCounterStore) Save(counters map[string]toolgate.CounterEntry) error {
	data, err := json.Marshal(counters)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := s.path + ".tmp"
	if writeErr := os.WriteFile(tmp, data, 0o644); writeErr != nil {
		return fmt.Errorf("write tmp: %w", writeErr)
	}
	return os.Rename(tmp, s.path)
}
```

Then in `runSurface`, after building the surfacer options list:

```go
// Tool frecency gate: inject with crypto/rand source and file-backed store.
store := newFileCounterStore(dataDir)
gate := toolgate.NewGate(store, cryptoRandFloat)
surfacerOpts = append(surfacerOpts, surface.WithToolGate(gate))
```

Add the random float helper at package level:

```go
// cryptoRandFloat returns a cryptographically random float64 in [0, 1).
func cryptoRandFloat() float64 {
	const maxMantissa = 1 << 53
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand never errors on supported platforms
	val := binary.BigEndian.Uint64(b) >> 11 // top 53 bits
	return float64(val) / float64(maxMantissa)
}
```

- [ ] **Step 3: Run `targ check-full` to verify everything compiles and passes**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/cli.go
git commit -m "feat(cli): wire toolgate into surface command with crypto/rand"
```

---

### Task 6: Shell Hook Guards

**Files:**
- Modify: `hooks/pre-tool-use.sh`
- Modify: `hooks/post-tool-use.sh`
- Modify: `hooks/post-tool-use-failure.sh`

- [ ] **Step 1: Add Bash-only guard to `pre-tool-use.sh`**

After the existing engram plumbing filter (line 44), before the surface call (line 47), add:

```bash
# Only surface memories for Bash tool calls — non-Bash tools produce near-random BM25 matches
if [[ "$TOOL_NAME" != "Bash" ]]; then
    exit 0
fi
```

- [ ] **Step 2: Add Bash-only guard to `post-tool-use.sh`**

After the Write/Edit advisory block (line 35), before the memory surfacing block (line 37), add:

```bash
# Only surface memories for Bash tool calls
if [[ "$TOOL_NAME" != "Bash" ]]; then
    exit 0
fi
```

This preserves the Write/Edit skill-file advisory for all tools.

- [ ] **Step 3: Add Bash-only guard to `post-tool-use-failure.sh`**

Wrap only the memory surfacing block (lines 42-51) in a Bash check. The static failure advice (lines 20-40) remains for all tools:

```bash
# Only surface memories for Bash failures — static advice still applies to all tools
MEMORY_CONTEXT=""
MEMORY_SUMMARY=""
if [[ "$TOOL_NAME" == "Bash" && -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$ERROR" --tool-errored \
        --data-dir "$DATA_DIR" --format json 2>/dev/null) || SURFACE_OUT=""
    MEMORY_CONTEXT="$(echo "$SURFACE_OUT" | jq -r '.context // empty' 2>/dev/null)" || MEMORY_CONTEXT=""
    MEMORY_SUMMARY="$(echo "$SURFACE_OUT" | jq -r '.summary // empty' 2>/dev/null)" || MEMORY_SUMMARY=""
fi
```

- [ ] **Step 4: Test hooks manually**

```bash
# pre-tool-use: Grep should produce no output
echo '{"tool_name":"Grep","tool_input":{"pattern":"foo"}}' | bash hooks/pre-tool-use.sh

# pre-tool-use: Bash should still produce output (if binary is built)
echo '{"tool_name":"Bash","tool_input":{"command":"go test"}}' | bash hooks/pre-tool-use.sh
```

- [ ] **Step 5: Commit**

```bash
git add hooks/pre-tool-use.sh hooks/post-tool-use.sh hooks/post-tool-use-failure.sh
git commit -m "feat(hooks): skip memory surfacing for non-Bash tool calls"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 2: Manual smoke test**

Start a new Claude Code session in the engram project. Verify:
1. Grep/Read/Edit/Glob calls produce NO memory advisories
2. Bash calls still produce memory advisories (with decreasing frequency)
3. `post-tool-use-failure.sh` still shows static failure advice for Read/Edit failures
4. `post-tool-use.sh` still shows skill-file advisory for Write/Edit of skill files

- [ ] **Step 3: Final commit (if any fixups needed)**
