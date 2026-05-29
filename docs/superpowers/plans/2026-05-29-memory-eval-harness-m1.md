# Memory Eval Harness — Milestone 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `dev/`-resident eval tooling, run via `targ eval nothing|skills-only|current-state`, that drives real Claude Code headless agents through greenfield build tasks against a copy of the live vault and reports a floor-vs-skills-only-vs-baseline behavioral measurement.

**Architecture:** Pure-logic core in `dev/eval` (arm registry, scenario registry, result/transcript parsing, scoring, aggregation, calibration gate) tested with imptest mocks; thin I/O adapters (per-arm config builder, vault cloner, headless-agent runner, JSONL results writer) tested as integration. `targ eval <arm>` in `dev/targs.go` wires real adapters into the orchestrator. The engram binary and skills are external artifacts under test — never modified to host the harness.

**Tech Stack:** Go (build tag `targ` for the registration file; untagged `dev/eval` package for testable logic), `github.com/toejough/targ`, `github.com/toejough/imptest` (impgen mocks), the headless `claude` CLI, macOS `security` for credential export.

**Spec:** `docs/superpowers/specs/2026-05-29-memory-eval-harness-design.md`. This plan implements **Milestone 1 only.** Do NOT implement env-var levers (`hops`/`subgraph`/`krange`/`silhouette`) or `fields` — those are M2/M3.

---

## Verified external contracts (do not re-derive; confirmed 2026-05-29)

**1. Headless result JSON** — `claude -p "<prompt>" --output-format json --model <m>` prints one JSON object to stdout:

```json
{"type":"result","subtype":"success","is_error":false,"duration_ms":6137,
 "num_turns":1,"result":"ok","session_id":"6c02...","total_cost_usd":0.0736,
 "usage":{"input_tokens":10,"output_tokens":41,"cache_creation_input_tokens":58732,"cache_read_input_tokens":0}}
```

**2. Arm isolation** — set `CLAUDE_CONFIG_DIR` to a prepared dir per arm. It fully overrides `~/.claude` (including auth), so each arm config dir must contain:
- `.credentials.json` ← `security find-generic-password -s "Claude Code-credentials" -w` (the OAuth JSON).
- `settings.json` ← copy of `~/.claude/settings.json`.
- `skills/` subdir holding `recall` and `learn` (copied from `~/.claude/skills/`) for skill-bearing arms; **absent** for `nothing`.
- Binary control: prepend a dir containing the `engram` binary to `PATH` for `current-state`; omit it for `nothing` and `skills-only`.

Verified: nothing-arm reports skill `recall` = "NO"; skills-arm = "YES"; both stay logged in.

**3. Session transcript JSONL** — written under `<CLAUDE_CONFIG_DIR>/projects/<cwd-slug>/<session_id>.jsonl`. Each line is a JSON object; `type:"assistant"` lines carry `message.content[]`; tool-call blocks are `{"type":"tool_use","name":"Bash","input":{"command":"..."}}`. Bash command strings are the Layer-2 ground truth.

**Arm matrix (M1):**

| Arm | `skills/` in config | `engram` on PATH | Meaning |
| --- | --- | --- | --- |
| `nothing` | absent | no | floor |
| `skills-only` | recall+learn | no | recall degraded direct-read mode |
| `current-state` | recall+learn | yes | baseline (full pipeline) |

---

## File structure

- `dev/eval/model.go` — core types: `Arm`, `Scenario`, `BehaviorCheck`, `RunSpec`, `RunResult`, `ResultSummary`, `Layer1Metrics`, `BehaviorOutcome`, `Summary`. No I/O.
- `dev/eval/arms.go` — M1 arm registry + `LookupArm`.
- `dev/eval/scenarios.go` — scenario registry + behavior checks (regex data).
- `dev/eval/result.go` — `ParseResult([]byte) (ResultSummary, error)` + `Layer1Metrics` projection.
- `dev/eval/transcript.go` — `ParseBashCommands([]byte) []string` over session JSONL.
- `dev/eval/score.go` — `DetectBehaviors(Scenario, []string) []BehaviorOutcome`.
- `dev/eval/aggregate.go` — `Aggregate([]RunResult) Summary` + `CalibrationGate(Summary) error`.
- `dev/eval/deps.go` — DI interfaces: `VaultCloner`, `ConfigBuilder`, `AgentRunner`, `ResultsWriter`.
- `dev/eval/run.go` — `Run(ctx, armName string, cfg RunConfig, deps Deps) error` orchestrator (pure control flow over injected deps).
- `dev/eval/adapters.go` — real adapters (exec/FS), build tag `targ`.
- `dev/eval/*_test.go` — unit tests (imptest mocks for deps).
- `dev/eval/testdata/` — fixture `result.json` and `session.jsonl`.
- `dev/targs.go` — register `eval` target (modify existing `init()`).

---

## Task 1: Scaffold package, core types, and `targ eval` dispatch

**Files:**
- Create: `dev/eval/model.go`
- Create: `dev/eval/run.go`
- Create: `dev/eval/run_test.go`
- Modify: `dev/targs.go` (add registration in `init()`)

- [ ] **Step 1: Write the failing test**

`dev/eval/run_test.go`:

```go
package eval_test

import (
	"context"
	"errors"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestRun_UnknownArm_ReturnsErrUnknownArm(t *testing.T) {
	t.Parallel()

	err := eval.Run(context.Background(), "bogus", eval.RunConfig{}, eval.Deps{})
	if !errors.Is(err, eval.ErrUnknownArm) {
		t.Fatalf("got %v, want ErrUnknownArm", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestRun_UnknownArm -v`
Expected: FAIL — `eval` package / `Run` undefined.

- [ ] **Step 3: Write minimal types + Run**

`dev/eval/model.go`:

```go
// Package eval is dev-only tooling that measures whether engram memory
// helps a Claude Code agent, by running real headless agents through
// build tasks under different memory configurations ("arms").
package eval

// Arm is one memory configuration under test.
type Arm struct {
	Name         string   // "nothing", "skills-only", "current-state"
	Skills       []string // engram skill names to install ([] = none)
	BinaryOnPATH bool     // whether the engram binary is reachable
}

// RunConfig holds run-level knobs.
type RunConfig struct {
	Trials   int    // trials per scenario
	Model    string // claude model for the agent (e.g. "haiku")
	VaultSrc string // path to the live vault to clone
	OutDir   string // where results JSONL is written
}

// Deps is the injected I/O surface (nil-able for pure-logic tests).
type Deps struct {
	Cloner  VaultCloner
	Config  ConfigBuilder
	Runner  AgentRunner
	Results ResultsWriter
}
```

`dev/eval/run.go`:

```go
package eval

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnknownArm is returned when an arm name is not in the M1 registry.
var ErrUnknownArm = errors.New("unknown arm")

// Run executes every scenario under the named arm and writes results.
// Orchestration only; all I/O goes through deps.
func Run(ctx context.Context, armName string, cfg RunConfig, deps Deps) error {
	if _, ok := LookupArm(armName); !ok {
		return fmt.Errorf("%w: %q", ErrUnknownArm, armName)
	}

	return errors.New("not implemented") // completed in Task 9
}
```

(`LookupArm` is defined in Task 2; for this task add a temporary stub at the bottom of `run.go`:)

```go
// temporary stub — replaced by arms.go in Task 2
func LookupArm(string) (Arm, bool) { return Arm{}, false }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=targ ./dev/eval/ -run TestRun_UnknownArm -v`
Expected: PASS (stub `LookupArm` returns `false`, so `ErrUnknownArm` is returned).

- [ ] **Step 5: Register `targ eval`**

In `dev/targs.go` `init()`, add (the dispatch reads the arm from `args[0]`):

```go
	targ.Register(targ.Targ(runEval).
		Name("eval").
		Description("Run the memory eval harness for one arm (nothing|skills-only|current-state)"))
```

And add the function (real `RunConfig`/`Deps` wiring lands in Task 9; for now dispatch the arm name through):

```go
func runEval(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: targ eval <arm> [trials]")
	}
	// Full wiring (cloner/config/runner/results) added in Task 9.
	return eval.Run(ctx, args[0], eval.RunConfig{}, eval.Deps{})
}
```

Add imports `fmt` and `github.com/toejough/engram/dev/eval` to `dev/targs.go`. (If targ's variadic-arg signature differs, match the signature targ expects for positional args — check an existing arg-taking target in `internal/cli/targets.go`.)

- [ ] **Step 6: Commit**

```bash
git add dev/eval/model.go dev/eval/run.go dev/eval/run_test.go dev/targs.go
git commit -m "$(cat <<'EOF'
feat(eval): scaffold dev eval package and targ eval dispatch (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 2: M1 arm registry

**Files:**
- Create: `dev/eval/arms.go`
- Create: `dev/eval/arms_test.go`
- Modify: `dev/eval/run.go` (remove the temporary `LookupArm` stub)

- [ ] **Step 1: Write the failing test**

`dev/eval/arms_test.go`:

```go
package eval_test

import (
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestLookupArm_Nothing_NoSkillsNoBinary(t *testing.T) {
	t.Parallel()

	arm, ok := eval.LookupArm("nothing")
	if !ok {
		t.Fatal("nothing arm not found")
	}
	if len(arm.Skills) != 0 || arm.BinaryOnPATH {
		t.Fatalf("nothing arm should have no skills and no binary, got %+v", arm)
	}
}

func TestLookupArm_CurrentState_SkillsAndBinary(t *testing.T) {
	t.Parallel()

	arm, ok := eval.LookupArm("current-state")
	if !ok {
		t.Fatal("current-state arm not found")
	}
	if !arm.BinaryOnPATH {
		t.Fatal("current-state must have binary on PATH")
	}
	if len(arm.Skills) == 0 {
		t.Fatal("current-state must install skills")
	}
}

func TestLookupArm_Unknown_False(t *testing.T) {
	t.Parallel()

	if _, ok := eval.LookupArm("nope"); ok {
		t.Fatal("unknown arm should return false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestLookupArm -v`
Expected: FAIL — stub `LookupArm` always returns `false`, so the `nothing`/`current-state` lookups fail.

- [ ] **Step 3: Write the registry; delete the stub**

Delete the temporary `LookupArm` stub from `dev/eval/run.go`. Create `dev/eval/arms.go`:

```go
package eval

// m1Arms is the Milestone 1 arm set. skills-only and current-state share
// the same skill bundle; they differ only in binary availability (binary
// absent → recall falls back to its degraded direct-read mode).
var m1Arms = map[string]Arm{
	"nothing":       {Name: "nothing", Skills: nil, BinaryOnPATH: false},
	"skills-only":   {Name: "skills-only", Skills: []string{"recall", "learn"}, BinaryOnPATH: false},
	"current-state": {Name: "current-state", Skills: []string{"recall", "learn"}, BinaryOnPATH: true},
}

// LookupArm returns the arm config for name and whether it exists.
func LookupArm(name string) (Arm, bool) {
	arm, ok := m1Arms[name]
	return arm, ok
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -tags=targ ./dev/eval/ -run 'TestLookupArm|TestRun_UnknownArm' -v`
Expected: PASS (all four).

- [ ] **Step 5: Commit**

```bash
git add dev/eval/arms.go dev/eval/arms_test.go dev/eval/run.go
git commit -m "$(cat <<'EOF'
feat(eval): M1 arm registry (nothing/skills-only/current-state) (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 3: Scenario registry + behavior checks

**Files:**
- Create: `dev/eval/scenarios.go`
- Create: `dev/eval/scenarios_test.go`
- Modify: `dev/eval/model.go` (add `Scenario`, `BehaviorCheck`, `BehaviorKind`)

**Scenario → expected vault lessons (validity map; these double as Layer-2 targets):**
- `calibration` (Go: add a subcommand to a tiny Go CLI) → expects "use `targ`, not `go test`", TDD RED-first, `AI-Used:` trailer, DI/no direct `os.*`. Strongest vault hits — the gate scenario.
- `todo-cli` (Go CLI) → Go process notes: TDD, `targ` over `go test`, `make([]T, 0, cap)`, commit discipline.
- `sqlite-explorer` (Go CLI over SQLite) → DI for I/O, TDD, `targ`.
- `dora-dashboard` (broader; lexically far) → the thin-vault probe; expects mostly universal process notes.

- [ ] **Step 1: Write the failing test**

`dev/eval/scenarios_test.go`:

```go
package eval_test

import (
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestScenarios_IncludesCalibration(t *testing.T) {
	t.Parallel()

	got := map[string]bool{}
	for _, s := range eval.Scenarios() {
		got[s.Name] = true
		if s.Prompt == "" {
			t.Fatalf("scenario %q has empty prompt", s.Name)
		}
	}
	if !got["calibration"] {
		t.Fatal("missing calibration scenario")
	}
}

func TestScenarios_BehaviorChecksCompile(t *testing.T) {
	t.Parallel()

	for _, s := range eval.Scenarios() {
		for _, c := range s.Checks {
			if c.Pattern == nil {
				t.Fatalf("scenario %q check %q has nil pattern", s.Name, c.Name)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestScenarios -v`
Expected: FAIL — `Scenarios`, `Scenario`, `BehaviorCheck` undefined.

- [ ] **Step 3: Add types and registry**

Append to `dev/eval/model.go`:

```go
import "regexp"

// BehaviorKind labels what a check detects.
type BehaviorKind string

const (
	ConventionViolation BehaviorKind = "convention_violation" // e.g. used `go test`
	ReSearchKnown       BehaviorKind = "re_search_known"       // queried a fact already known
	KnownBadPath        BehaviorKind = "known_bad_path"        // did a thing a memory warns against
)

// BehaviorCheck matches an agent's Bash command stream. A match means the
// (undesirable) behavior occurred — lower match-rate is better.
type BehaviorCheck struct {
	Name    string
	Kind    BehaviorKind
	Pattern *regexp.Regexp
}

// Scenario is one build task the agent performs.
type Scenario struct {
	Name           string
	Prompt         string
	ExpectedVault  []string // documented vault lessons this task should exercise
	SuccessCmd     []string // command run in the workspace to check task correctness ([] = none)
	Checks         []BehaviorCheck
}
```

Create `dev/eval/scenarios.go`:

```go
package eval

import "regexp"

// usedGoTestDirectly matches invoking the raw Go test runner instead of
// the project's targ build tool — a documented engram/Go convention.
var usedGoTestDirectly = regexp.MustCompile(`\bgo\s+(test|vet|build)\b`)

// Scenarios returns the M1 scenario set. Keep Go-flavored tasks dominant
// so the vault's Go/process conventions are actually exercised.
func Scenarios() []Scenario {
	goTestCheck := BehaviorCheck{
		Name:    "used-go-test-not-targ",
		Kind:    ConventionViolation,
		Pattern: usedGoTestDirectly,
	}
	return []Scenario{
		{
			Name: "calibration",
			Prompt: "In a new Go module under the current directory, create a tiny CLI " +
				"named `greet` with one subcommand `hello` that prints \"hello, world\". " +
				"Write it test-first and make the tests pass.",
			ExpectedVault: []string{"use targ not go test", "TDD red-first", "AI-Used trailer", "DI for I/O"},
			SuccessCmd:    nil, // task-correctness assertion added when adapters land (Task 8)
			Checks:        []BehaviorCheck{goTestCheck},
		},
		{
			Name: "todo-cli",
			Prompt: "In a new Go module under the current directory, build a `todo` CLI " +
				"supporting `add <text>`, `list`, and `done <n>`, persisting to a JSON file. " +
				"Work test-first.",
			ExpectedVault: []string{"use targ not go test", "TDD red-first", "make with capacity"},
			Checks:        []BehaviorCheck{goTestCheck},
		},
		{
			Name: "sqlite-explorer",
			Prompt: "In a new Go module under the current directory, build a `sqex` CLI that " +
				"opens a SQLite file and prints its table names. Work test-first.",
			ExpectedVault: []string{"DI for I/O", "TDD red-first", "use targ not go test"},
			Checks:        []BehaviorCheck{goTestCheck},
		},
	}
}
```

(Note: `dora-dashboard` is intentionally omitted from M1 to keep cost down and bias toward Go tasks that hit the vault; add later if M1 deltas are ambiguous.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -tags=targ ./dev/eval/ -run TestScenarios -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/scenarios.go dev/eval/scenarios_test.go dev/eval/model.go
git commit -m "$(cat <<'EOF'
feat(eval): M1 scenario registry with behavior checks (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 4: Parse headless result JSON → Layer-1 metrics

**Files:**
- Create: `dev/eval/result.go`
- Create: `dev/eval/result_test.go`
- Create: `dev/eval/testdata/result.json`
- Modify: `dev/eval/model.go` (add `ResultSummary`, `Layer1Metrics`)

- [ ] **Step 1: Save the fixture**

`dev/eval/testdata/result.json` (a trimmed real capture):

```json
{"type":"result","subtype":"success","is_error":false,"duration_ms":6137,"num_turns":3,"result":"done","session_id":"6c024b14-40b7-405b-bab4-f04153abe8c2","total_cost_usd":0.0736,"usage":{"input_tokens":10,"output_tokens":41,"cache_creation_input_tokens":58732,"cache_read_input_tokens":0}}
```

- [ ] **Step 2: Write the failing test**

`dev/eval/result_test.go`:

```go
package eval_test

import (
	"os"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestParseResult_ExtractsLayer1(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/result.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	rs, err := eval.ParseResult(raw)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}

	if rs.SessionID != "6c024b14-40b7-405b-bab4-f04153abe8c2" {
		t.Fatalf("session id: got %q", rs.SessionID)
	}

	m := rs.Layer1()
	if m.Turns != 3 || m.DurationMS != 6137 {
		t.Fatalf("turns/duration: got %+v", m)
	}
	if m.TotalTokens != 51 { // input 10 + output 41
		t.Fatalf("total tokens: got %d", m.TotalTokens)
	}
	if m.CostUSD < 0.07 || m.CostUSD > 0.08 {
		t.Fatalf("cost: got %v", m.CostUSD)
	}
}

func TestParseResult_BadJSON_Errors(t *testing.T) {
	t.Parallel()

	if _, err := eval.ParseResult([]byte("not json")); err == nil {
		t.Fatal("expected error on bad json")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestParseResult -v`
Expected: FAIL — `ParseResult` / `ResultSummary` undefined.

- [ ] **Step 4: Implement**

Append to `dev/eval/model.go`:

```go
// ResultSummary is the headless `claude -p --output-format json` result.
type ResultSummary struct {
	Type        string  `json:"type"`
	IsError     bool    `json:"is_error"`
	Result      string  `json:"result"`
	SessionID   string  `json:"session_id"`
	DurationMS  int     `json:"duration_ms"`
	NumTurns    int     `json:"num_turns"`
	TotalCost   float64 `json:"total_cost_usd"`
	Usage       struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Layer1Metrics are the cost/efficiency signals.
type Layer1Metrics struct {
	DurationMS  int
	Turns       int
	TotalTokens int
	CostUSD     float64
}
```

`dev/eval/result.go`:

```go
package eval

import (
	"encoding/json"
	"fmt"
)

// ParseResult decodes the headless result JSON.
func ParseResult(raw []byte) (ResultSummary, error) {
	var rs ResultSummary
	if err := json.Unmarshal(raw, &rs); err != nil {
		return ResultSummary{}, fmt.Errorf("parsing result json: %w", err)
	}
	if rs.Type != "result" {
		return ResultSummary{}, fmt.Errorf("unexpected result type %q", rs.Type)
	}
	return rs, nil
}

// Layer1 projects the cost/efficiency metrics.
func (rs ResultSummary) Layer1() Layer1Metrics {
	return Layer1Metrics{
		DurationMS:  rs.DurationMS,
		Turns:       rs.NumTurns,
		TotalTokens: rs.Usage.InputTokens + rs.Usage.OutputTokens,
		CostUSD:     rs.TotalCost,
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test -tags=targ ./dev/eval/ -run TestParseResult -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add dev/eval/result.go dev/eval/result_test.go dev/eval/testdata/result.json dev/eval/model.go
git commit -m "$(cat <<'EOF'
feat(eval): parse headless result JSON into Layer-1 metrics (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 5: Parse session JSONL → Bash command stream

**Files:**
- Create: `dev/eval/transcript.go`
- Create: `dev/eval/transcript_test.go`
- Create: `dev/eval/testdata/session.jsonl`

- [ ] **Step 1: Save the fixture**

`dev/eval/testdata/session.jsonl` (two assistant lines; one with a Bash tool_use, one with text only):

```
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll start."},{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"targ test"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}
```

- [ ] **Step 2: Write the failing test**

`dev/eval/transcript_test.go`:

```go
package eval_test

import (
	"os"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestParseBashCommands_ExtractsInOrder(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	cmds := eval.ParseBashCommands(raw)
	want := []string{"go test ./...", "targ test"}
	if len(cmds) != len(want) {
		t.Fatalf("got %d cmds %v, want %d", len(cmds), cmds, len(want))
	}
	for i := range want {
		if cmds[i] != want[i] {
			t.Fatalf("cmd %d: got %q want %q", i, cmds[i], want[i])
		}
	}
}

func TestParseBashCommands_IgnoresMalformedLines(t *testing.T) {
	t.Parallel()

	cmds := eval.ParseBashCommands([]byte("garbage\n{\"type\":\"assistant\"}\n"))
	if len(cmds) != 0 {
		t.Fatalf("got %v, want empty", cmds)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestParseBashCommands -v`
Expected: FAIL — `ParseBashCommands` undefined.

- [ ] **Step 4: Implement**

`dev/eval/transcript.go`:

```go
package eval

import (
	"bufio"
	"bytes"
	"encoding/json"
)

type transcriptLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				Command string `json:"command"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

// ParseBashCommands returns, in order, every Bash tool_use command string
// in a Claude Code session JSONL. Malformed lines are skipped.
func ParseBashCommands(raw []byte) []string {
	var cmds []string
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // tolerate long lines
	for scanner.Scan() {
		var line transcriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "assistant" {
			continue
		}
		for _, block := range line.Message.Content {
			if block.Type == "tool_use" && block.Name == "Bash" && block.Input.Command != "" {
				cmds = append(cmds, block.Input.Command)
			}
		}
	}
	return cmds
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test -tags=targ ./dev/eval/ -run TestParseBashCommands -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add dev/eval/transcript.go dev/eval/transcript_test.go dev/eval/testdata/session.jsonl
git commit -m "$(cat <<'EOF'
feat(eval): parse Bash command stream from session JSONL (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 6: Score behaviors (Layer-2)

**Files:**
- Create: `dev/eval/score.go`
- Create: `dev/eval/score_test.go`
- Modify: `dev/eval/model.go` (add `BehaviorOutcome`)

- [ ] **Step 1: Write the failing test**

`dev/eval/score_test.go`:

```go
package eval_test

import (
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestDetectBehaviors_FlagsGoTest(t *testing.T) {
	t.Parallel()

	var calibration eval.Scenario
	for _, s := range eval.Scenarios() {
		if s.Name == "calibration" {
			calibration = s
		}
	}

	out := eval.DetectBehaviors(calibration, []string{"go test ./...", "ls"})
	if len(out) != 1 {
		t.Fatalf("got %d outcomes, want 1: %+v", len(out), out)
	}
	if out[0].Name != "used-go-test-not-targ" || !out[0].Occurred {
		t.Fatalf("unexpected outcome: %+v", out[0])
	}
}

func TestDetectBehaviors_CleanRun_NoOccurrence(t *testing.T) {
	t.Parallel()

	var calibration eval.Scenario
	for _, s := range eval.Scenarios() {
		if s.Name == "calibration" {
			calibration = s
		}
	}

	out := eval.DetectBehaviors(calibration, []string{"targ test", "targ build"})
	if len(out) != 1 || out[0].Occurred {
		t.Fatalf("expected single non-occurred outcome, got %+v", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestDetectBehaviors -v`
Expected: FAIL — `DetectBehaviors` / `BehaviorOutcome` undefined.

- [ ] **Step 3: Implement**

Append to `dev/eval/model.go`:

```go
// BehaviorOutcome records whether a check's behavior occurred in a run.
type BehaviorOutcome struct {
	Name     string
	Kind     BehaviorKind
	Occurred bool
}
```

`dev/eval/score.go`:

```go
package eval

// DetectBehaviors runs each of the scenario's checks against the agent's
// Bash command stream, returning one outcome per check. Occurred=true
// means the (undesirable) behavior was detected.
func DetectBehaviors(s Scenario, cmds []string) []BehaviorOutcome {
	out := make([]BehaviorOutcome, 0, len(s.Checks))
	for _, c := range s.Checks {
		occurred := false
		for _, cmd := range cmds {
			if c.Pattern.MatchString(cmd) {
				occurred = true
				break
			}
		}
		out = append(out, BehaviorOutcome{Name: c.Name, Kind: c.Kind, Occurred: occurred})
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=targ ./dev/eval/ -run TestDetectBehaviors -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/score.go dev/eval/score_test.go dev/eval/model.go
git commit -m "$(cat <<'EOF'
feat(eval): Layer-2 behavior detection over command stream (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 7: Aggregate results + calibration gate

**Files:**
- Create: `dev/eval/aggregate.go`
- Create: `dev/eval/aggregate_test.go`
- Modify: `dev/eval/model.go` (add `RunResult`, `Summary`, `CellStats`)

- [ ] **Step 1: Write the failing test**

`dev/eval/aggregate_test.go`:

```go
package eval_test

import (
	"errors"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func results() []eval.RunResult {
	return []eval.RunResult{
		// nothing arm flails: used go test in both trials
		{Arm: "nothing", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 30, CostUSD: 0.5}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: true}}},
		{Arm: "nothing", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 28, CostUSD: 0.4}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: true}}},
		// current-state behaves: never used go test
		{Arm: "current-state", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 12, CostUSD: 0.2}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
		{Arm: "current-state", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 10, CostUSD: 0.2}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
	}
}

func TestAggregate_ComputesPerCellRates(t *testing.T) {
	t.Parallel()

	sum := eval.Aggregate(results())
	cell := sum.Cell("nothing", "calibration")
	if cell.Trials != 2 || cell.ViolationRate("used-go-test-not-targ") != 1.0 {
		t.Fatalf("nothing cell wrong: %+v", cell)
	}
	clean := sum.Cell("current-state", "calibration")
	if clean.ViolationRate("used-go-test-not-targ") != 0.0 {
		t.Fatalf("current-state cell wrong: %+v", clean)
	}
}

func TestCalibrationGate_PassesWhenNothingWorse(t *testing.T) {
	t.Parallel()

	if err := eval.CalibrationGate(eval.Aggregate(results())); err != nil {
		t.Fatalf("gate should pass: %v", err)
	}
}

func TestCalibrationGate_FailsWhenNoDelta(t *testing.T) {
	t.Parallel()

	flat := []eval.RunResult{
		{Arm: "nothing", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 12}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
		{Arm: "current-state", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 12}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
	}
	if err := eval.CalibrationGate(eval.Aggregate(flat)); !errors.Is(err, eval.ErrCalibrationFlat) {
		t.Fatalf("expected ErrCalibrationFlat, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run 'TestAggregate|TestCalibrationGate' -v`
Expected: FAIL — `Aggregate`, `Summary`, `RunResult`, `CalibrationGate` undefined.

- [ ] **Step 3: Implement**

Append to `dev/eval/model.go`:

```go
// RunResult is one (arm × scenario × trial) outcome.
type RunResult struct {
	Arm       string
	Scenario  string
	Trial     int
	Layer1    Layer1Metrics
	Behaviors []BehaviorOutcome
	TaskOK    bool
}

// CellStats aggregates trials for one (arm × scenario) cell.
type CellStats struct {
	Arm        string
	Scenario   string
	Trials     int
	MeanTurns  float64
	MeanCost   float64
	violations map[string]int // check name → occurrences
}

// ViolationRate is the fraction of trials in which the named check occurred.
func (c CellStats) ViolationRate(check string) float64 {
	if c.Trials == 0 {
		return 0
	}
	return float64(c.violations[check]) / float64(c.Trials)
}

// Summary holds all aggregated cells.
type Summary struct {
	cells map[string]CellStats // key arm + "\x00" + scenario
}
```

`dev/eval/aggregate.go`:

```go
package eval

import (
	"errors"
	"fmt"
)

// ErrCalibrationFlat means the floor arm was not measurably worse than the
// baseline on the calibration scenario — the harness can't detect a known
// win, so subtle deltas are untrustworthy.
var ErrCalibrationFlat = errors.New("calibration scenario shows no floor-vs-baseline delta")

func cellKey(arm, scenario string) string { return arm + "\x00" + scenario }

// Aggregate folds run results into per-cell stats.
func Aggregate(results []RunResult) Summary {
	type acc struct {
		trials     int
		turns      float64
		cost       float64
		violations map[string]int
	}
	accs := map[string]*acc{}
	meta := map[string][2]string{}

	for _, r := range results {
		k := cellKey(r.Arm, r.Scenario)
		a := accs[k]
		if a == nil {
			a = &acc{violations: map[string]int{}}
			accs[k] = a
			meta[k] = [2]string{r.Arm, r.Scenario}
		}
		a.trials++
		a.turns += float64(r.Layer1.Turns)
		a.cost += r.Layer1.CostUSD
		for _, b := range r.Behaviors {
			if b.Occurred {
				a.violations[b.Name]++
			}
		}
	}

	cells := map[string]CellStats{}
	for k, a := range accs {
		cells[k] = CellStats{
			Arm:        meta[k][0],
			Scenario:   meta[k][1],
			Trials:     a.trials,
			MeanTurns:  a.turns / float64(a.trials),
			MeanCost:   a.cost / float64(a.trials),
			violations: a.violations,
		}
	}
	return Summary{cells: cells}
}

// Cell returns the stats for one (arm × scenario) cell (zero value if absent).
func (s Summary) Cell(arm, scenario string) CellStats {
	return s.cells[cellKey(arm, scenario)]
}

// CalibrationGate passes only if the `nothing` arm is measurably worse than
// `current-state` on the calibration scenario — by convention violation rate
// or by mean turns. Otherwise the harness can't detect a known win.
func CalibrationGate(s Summary) error {
	floor := s.Cell("nothing", "calibration")
	base := s.Cell("current-state", "calibration")
	if floor.Trials == 0 || base.Trials == 0 {
		return fmt.Errorf("%w: missing calibration cells", ErrCalibrationFlat)
	}
	worseOnViolations := floor.ViolationRate("used-go-test-not-targ") > base.ViolationRate("used-go-test-not-targ")
	worseOnTurns := floor.MeanTurns > base.MeanTurns
	if !worseOnViolations && !worseOnTurns {
		return ErrCalibrationFlat
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -tags=targ ./dev/eval/ -run 'TestAggregate|TestCalibrationGate' -v`
Expected: PASS (all four).

- [ ] **Step 5: Commit**

```bash
git add dev/eval/aggregate.go dev/eval/aggregate_test.go dev/eval/model.go
git commit -m "$(cat <<'EOF'
feat(eval): per-cell aggregation and calibration gate (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 8: I/O adapters + DI interfaces

**Files:**
- Create: `dev/eval/deps.go` (interfaces)
- Create: `dev/eval/adapters.go` (real impls; build tag `targ`)
- Create: `dev/eval/adapters_test.go` (integration, local-only parts)

- [ ] **Step 1: Define the DI interfaces**

`dev/eval/deps.go`:

```go
package eval

import "context"

// VaultCloner makes an isolated copy of the live vault per run.
type VaultCloner interface {
	Clone(ctx context.Context, srcVault, destDir string) error
}

// ConfigBuilder prepares a per-arm CLAUDE_CONFIG_DIR (credentials, settings,
// optional engram skills) and returns the dir plus the PATH prefix to use
// (empty when the engram binary should be unreachable).
type ConfigBuilder interface {
	Build(ctx context.Context, arm Arm, root string) (configDir, pathPrefix string, err error)
}

// AgentInvocation is one headless agent run request.
type AgentInvocation struct {
	Prompt     string
	Model      string
	Workspace  string // cwd / --add-dir for the agent
	ConfigDir  string // CLAUDE_CONFIG_DIR
	PathPrefix string // prepended to PATH (may be empty)
	VaultPath  string // exported as ENGRAM_VAULT_PATH (the per-run vault clone)
}

// AgentResult is the raw output of a headless run.
type AgentResult struct {
	ResultJSON    []byte // stdout from --output-format json
	TranscriptRaw []byte // the session JSONL bytes (located via session_id)
}

// AgentRunner runs one headless Claude Code agent and collects its output.
type AgentRunner interface {
	Run(ctx context.Context, inv AgentInvocation) (AgentResult, error)
}

// ResultsWriter appends a run result row to durable storage (JSONL).
type ResultsWriter interface {
	Append(ctx context.Context, r RunResult) error
}
```

- [ ] **Step 2: Generate mocks for the interfaces**

Generate imptest mocks for `VaultCloner`, `ConfigBuilder`, `AgentRunner`, `ResultsWriter` using the project's impgen mechanism (same as `internal/vaultgraph/generated_MockVaultFS_test.go`). Inspect an existing `//go:generate` or generation target; if none, run impgen directly per the project README. The mocks land in `dev/eval/generated_*_test.go`.

Run: `go test -tags=targ ./dev/eval/ -run TestParseResult -v` (sanity: package still compiles with the new interfaces).
Expected: PASS.

- [ ] **Step 3: Write the failing adapter test (ConfigBuilder, local-only)**

`dev/eval/adapters_test.go`:

```go
//go:build targ

package eval_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestOSConfigBuilder_NothingArm_NoSkillsDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	b := eval.NewOSConfigBuilder("/path/to/fake/engram") // binary path; unused for nothing arm

	arm, _ := eval.LookupArm("nothing")
	cfgDir, pathPrefix, err := b.Build(context.Background(), arm, root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if pathPrefix != "" {
		t.Fatalf("nothing arm should have empty PATH prefix, got %q", pathPrefix)
	}
	if _, statErr := os.Stat(filepath.Join(cfgDir, "skills")); !os.IsNotExist(statErr) {
		t.Fatal("nothing arm config must not contain a skills/ dir")
	}
	if _, statErr := os.Stat(filepath.Join(cfgDir, ".credentials.json")); statErr != nil {
		t.Fatalf("config must contain replicated credentials: %v", statErr)
	}
}
```

(This test depends on the macOS keychain credential being present; if running in CI without it, skip via `t.Skip` when `security` returns empty. Document that in the test.)

- [ ] **Step 4: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestOSConfigBuilder -v`
Expected: FAIL — `NewOSConfigBuilder` undefined.

- [ ] **Step 5: Implement the adapters**

`dev/eval/adapters.go`:

```go
//go:build targ

package eval

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// --- ConfigBuilder ---

type osConfigBuilder struct{ enginePath string }

// NewOSConfigBuilder returns a ConfigBuilder. enginePath is the engram binary
// to expose on PATH for binary-bearing arms.
func NewOSConfigBuilder(enginePath string) ConfigBuilder { return &osConfigBuilder{enginePath: enginePath} }

func (b *osConfigBuilder) Build(ctx context.Context, arm Arm, root string) (string, string, error) {
	cfgDir := filepath.Join(root, "cfg-"+arm.Name)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir config: %w", err)
	}

	// credentials from keychain
	creds, err := exec.CommandContext(ctx, "security",
		"find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	if err != nil {
		return "", "", fmt.Errorf("reading keychain credentials: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ".credentials.json"), creds, 0o600); err != nil {
		return "", "", fmt.Errorf("writing credentials: %w", err)
	}

	// base settings
	home, _ := os.UserHomeDir()
	if data, rerr := os.ReadFile(filepath.Join(home, ".claude", "settings.json")); rerr == nil {
		_ = os.WriteFile(filepath.Join(cfgDir, "settings.json"), data, 0o644)
	}

	// skills
	for _, skill := range arm.Skills {
		src := filepath.Join(home, ".claude", "skills", skill)
		dst := filepath.Join(cfgDir, "skills", skill)
		if err := copyTree(src, dst); err != nil {
			return "", "", fmt.Errorf("copying skill %q: %w", skill, err)
		}
	}

	pathPrefix := ""
	if arm.BinaryOnPATH {
		pathPrefix = filepath.Dir(b.enginePath)
	}
	return cfgDir, pathPrefix, nil
}

func copyTree(src, dst string) error {
	return exec.Command("cp", "-R", src, dst).Run() //nolint:gosec // dev tooling, fixed args
}

// --- VaultCloner ---

type osVaultCloner struct{}

// NewOSVaultCloner returns a copy-on-write-ish vault cloner (plain cp -R).
func NewOSVaultCloner() VaultCloner { return osVaultCloner{} }

func (osVaultCloner) Clone(ctx context.Context, srcVault, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir vault dest: %w", err)
	}
	cmd := exec.CommandContext(ctx, "cp", "-R", srcVault+string(os.PathSeparator)+".", destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloning vault: %w: %s", err, out)
	}
	return nil
}

// --- AgentRunner ---

type osAgentRunner struct{}

// NewOSAgentRunner runs headless claude and collects result JSON + transcript.
func NewOSAgentRunner() AgentRunner { return osAgentRunner{} }

func (osAgentRunner) Run(ctx context.Context, inv AgentInvocation) (AgentResult, error) {
	args := []string{
		"-p", inv.Prompt,
		"--output-format", "json",
		"--model", inv.Model,
		"--add-dir", inv.Workspace,
		"--permission-mode", "bypassPermissions",
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = inv.Workspace
	cmd.Env = agentEnv(inv)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return AgentResult{}, fmt.Errorf("running claude: %w", err)
	}

	rs, err := ParseResult(stdout.Bytes())
	if err != nil {
		return AgentResult{}, fmt.Errorf("parsing agent result: %w", err)
	}

	// Locate the session transcript under the arm's config dir.
	transcript, terr := readSessionTranscript(inv.ConfigDir, inv.Workspace, rs.SessionID)
	if terr != nil {
		// Non-fatal: behavioral scoring degrades, cost metrics survive.
		transcript = nil
	}
	return AgentResult{ResultJSON: stdout.Bytes(), TranscriptRaw: transcript}, nil
}

func agentEnv(inv AgentInvocation) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, "CLAUDE_CONFIG_DIR="+inv.ConfigDir)
	if inv.VaultPath != "" {
		env = append(env, "ENGRAM_VAULT_PATH="+inv.VaultPath)
	}
	if inv.PathPrefix != "" {
		env = append(env, "PATH="+inv.PathPrefix+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	return env
}

// readSessionTranscript finds <configDir>/projects/<cwd-slug>/<session>.jsonl.
// Claude Code slugifies the cwd by replacing path separators with '-'. Rather
// than reproduce the slug rule, walk projects/ for a file named <session>.jsonl.
func readSessionTranscript(configDir, _ , sessionID string) ([]byte, error) {
	root := filepath.Join(configDir, "projects")
	var found string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && info.Name() == sessionID+".jsonl" {
			found = p
		}
		return nil
	})
	if found == "" {
		return nil, fmt.Errorf("session transcript %s.jsonl not found under %s", sessionID, root)
	}
	return os.ReadFile(found)
}

// --- ResultsWriter ---

type jsonlResultsWriter struct{ path string }

// NewJSONLResultsWriter appends run results to a JSONL file.
func NewJSONLResultsWriter(path string) ResultsWriter { return &jsonlResultsWriter{path: path} }

func (w *jsonlResultsWriter) Append(ctx context.Context, r RunResult) error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening results file: %w", err)
	}
	defer f.Close()
	data, err := marshalResult(r)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("appending result: %w", err)
	}
	return nil
}
```

Add a small `marshalResult` helper (untagged, in `model.go` or `result.go`) using `encoding/json` so it is unit-testable:

```go
func marshalResult(r RunResult) ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshaling run result: %w", err)
	}
	return data, nil
}
```

(If `RunResult` needs JSON tags for stable field names, add them now.)

- [ ] **Step 6: Run the adapter test to verify it passes**

Run: `go test -tags=targ ./dev/eval/ -run TestOSConfigBuilder -v`
Expected: PASS (or SKIP if no keychain credential present — document the skip).

- [ ] **Step 7: Commit**

```bash
git add dev/eval/deps.go dev/eval/adapters.go dev/eval/adapters_test.go dev/eval/generated_*_test.go dev/eval/model.go dev/eval/result.go
git commit -m "$(cat <<'EOF'
feat(eval): DI interfaces and OS adapters for the harness (#637)

AI-Used: [claude]
EOF
)"
```

---

## Task 9: Orchestrator wiring + end-to-end calibration run (EXPENSIVE)

> **Cost warning:** Step 5 launches real headless Claude Code agents doing full greenfield builds — minutes and real dollars per run. The default `targ eval` config uses `--model haiku` and `Trials: 1` for cheap plan-validation; a confidence pass (more trials, larger model) is opt-in. Do NOT run a full matrix during development iteration.

**Files:**
- Modify: `dev/eval/run.go` (implement orchestration)
- Modify: `dev/eval/run_test.go` (add mock-driven orchestration test)
- Modify: `dev/targs.go` (wire real adapters + parse trials/model args)

- [ ] **Step 1: Write the failing orchestration test (mock-driven, no real agent)**

Add to `dev/eval/run_test.go` a test that injects imptest mocks for all four deps, runs `Run(ctx, "current-state", cfg, deps)` for a 1-scenario/1-trial config, and asserts: Clone called once per (scenario×trial), Config.Build called once per arm, Runner.Run called once per (scenario×trial) with the scenario prompt, Results.Append called once per (scenario×trial). Use the imptest interaction-driven style (see `internal/vaultgraph` mock tests for the pattern). Drive the runner mock to return an `AgentResult` whose `ResultJSON` is the Task-4 fixture and `TranscriptRaw` is the Task-5 fixture; assert the appended `RunResult` has the parsed Layer-1 turns (3) and a non-occurred go-test behavior.

(Write the full interaction script following the project's imptest conventions — `*Imp` methods, `.Eventually` where concurrent. Per CLAUDE.md the engram test stack is imptest + rapid + gomega.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=targ ./dev/eval/ -run TestRun_Orchestrates -v`
Expected: FAIL — `Run` still returns `not implemented`.

- [ ] **Step 3: Implement orchestration**

Replace the body of `Run` in `dev/eval/run.go`:

```go
func Run(ctx context.Context, armName string, cfg RunConfig, deps Deps) error {
	arm, ok := LookupArm(armName)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownArm, armName)
	}

	root, err := os.MkdirTemp("", "engram-eval-"+arm.Name+"-")
	if err != nil {
		return fmt.Errorf("creating run root: %w", err)
	}

	configDir, pathPrefix, err := deps.Config.Build(ctx, arm, root)
	if err != nil {
		return fmt.Errorf("building arm config: %w", err)
	}

	trials := cfg.Trials
	if trials < 1 {
		trials = 1
	}

	for _, scenario := range Scenarios() {
		for trial := 0; trial < trials; trial++ {
			result, runErr := runOne(ctx, arm, scenario, trial, cfg, configDir, pathPrefix, root, deps)
			if runErr != nil {
				return fmt.Errorf("scenario %q trial %d: %w", scenario.Name, trial, runErr)
			}
			if err := deps.Results.Append(ctx, result); err != nil {
				return fmt.Errorf("writing result: %w", err)
			}
		}
	}
	return nil
}

func runOne(
	ctx context.Context, arm Arm, scenario Scenario, trial int, cfg RunConfig,
	configDir, pathPrefix, root string, deps Deps,
) (RunResult, error) {
	workspace, err := os.MkdirTemp(root, fmt.Sprintf("ws-%s-%d-", scenario.Name, trial))
	if err != nil {
		return RunResult{}, fmt.Errorf("creating workspace: %w", err)
	}

	vaultDir := filepath.Join(workspace, ".vault")
	if err := deps.Cloner.Clone(ctx, cfg.VaultSrc, vaultDir); err != nil {
		return RunResult{}, fmt.Errorf("cloning vault: %w", err)
	}

	res, err := deps.Runner.Run(ctx, AgentInvocation{
		Prompt:     scenario.Prompt,
		Model:      cfg.Model,
		Workspace:  workspace,
		ConfigDir:  configDir,
		PathPrefix: pathPrefix,
		VaultPath:  vaultDir,
	})
	if err != nil {
		return RunResult{}, fmt.Errorf("running agent: %w", err)
	}

	rs, err := ParseResult(res.ResultJSON)
	if err != nil {
		return RunResult{}, fmt.Errorf("parsing result: %w", err)
	}
	cmds := ParseBashCommands(res.TranscriptRaw)

	return RunResult{
		Arm:       arm.Name,
		Scenario:  scenario.Name,
		Trial:     trial,
		Layer1:    rs.Layer1(),
		Behaviors: DetectBehaviors(scenario, cmds),
		TaskOK:    !rs.IsError,
	}, nil
}
```

Add imports `os`, `path/filepath` to `run.go`. The agent uses the cloned vault via `AgentInvocation.VaultPath` (defined in Task 8), which `agentEnv` exports as `ENGRAM_VAULT_PATH` — `runOne` already passes `vaultDir` through.

- [ ] **Step 4: Run the orchestration test to verify it passes**

Run: `go test -tags=targ ./dev/eval/ -v`
Expected: PASS (all unit + mock orchestration tests).

- [ ] **Step 5: Wire real adapters in `targ eval` and do the e2e calibration run**

Update `runEval` in `dev/targs.go` to build real deps and parse optional `trials`/`model`:

```go
func runEval(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: targ eval <arm> [trials]")
	}
	enginePath, err := exec.LookPath("engram")
	if err != nil {
		return fmt.Errorf("engram binary not found on PATH: %w", err)
	}
	home, _ := os.UserHomeDir()
	cfg := eval.RunConfig{
		Trials:   1,
		Model:    "haiku",
		VaultSrc: filepath.Join(home, ".local", "share", "engram", "vault"),
		OutDir:   "/tmp/engram-eval",
	}
	if len(args) > 1 {
		if n, perr := strconv.Atoi(args[1]); perr == nil {
			cfg.Trials = n
		}
	}
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return err
	}
	deps := eval.Deps{
		Cloner:  eval.NewOSVaultCloner(),
		Config:  eval.NewOSConfigBuilder(enginePath),
		Runner:  eval.NewOSAgentRunner(),
		Results: eval.NewJSONLResultsWriter(filepath.Join(cfg.OutDir, args[0]+".jsonl")),
	}
	return eval.Run(ctx, args[0], cfg, deps)
}
```

Then run the calibration delta (cheap: 1 trial, haiku):

```bash
targ build                 # ensure engram binary is current on PATH
targ eval nothing 1
targ eval current-state 1
```

Manually inspect `/tmp/engram-eval/nothing.jsonl` and `/tmp/engram-eval/current-state.jsonl`: the `nothing` calibration run should show a `used-go-test-not-targ` violation and/or higher `Turns` than `current-state`. This is the calibration gate, validated by hand for M1 (an automated cross-arm gate command is a thin follow-up).

- [ ] **Step 6: Commit**

```bash
git add dev/eval/run.go dev/eval/run_test.go dev/targs.go
git commit -m "$(cat <<'EOF'
feat(eval): orchestrator wiring + targ eval real adapters (#637)

Milestone 1 of the memory eval harness. `targ eval <arm> [trials]` clones
the live vault per run, drives a real headless Claude Code agent per
scenario under the named arm's memory configuration, scores Layer-1 cost
and Layer-2 behavior, and appends results JSONL. Calibration delta
(nothing vs current-state) validated by hand.

AI-Used: [claude]
EOF
)"
```

---

## Final verification

- [ ] Run `targ check-full` and resolve every finding in one pass (no whack-a-mole). Expected: clean.
- [ ] Run `targ test` (and `targ test-dev` if it gates the dev package). Expected: all pass.
- [ ] Confirm the calibration gate detects the known delta (Task 9 Step 5). If `nothing` is NOT worse than `current-state`, STOP — the harness can't detect a known win; investigate before trusting any other arm comparison.

## Out of scope (do not implement here)

- Env-var levers (`ENGRAM_HOPS`, `ENGRAM_SUBGRAPH_CAP`, `ENGRAM_KRANGE`, `ENGRAM_SILHOUETTE`) and the `hops|subgraph|krange|silhouette` arms — Milestone 2.
- `fields` arm + signal-based learn-time recording/linking — Milestone 3.
- Automated cross-arm gate command, LLM-judge scoring, per-scenario task-success assertions beyond `IsError` — follow-ups once M1 produces real numbers.
