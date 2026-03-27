# Remove Evaluate Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the `engram evaluate` CLI command and its underlying package, completing the migration to explicit feedback as the only outcome-recording mechanism.

**Architecture:** The flush pipeline already has evaluate removed (done in #348 wiring). What remains is deleting the now-dead `internal/evaluate` package, removing its CLI wiring from `cli.go`/`targets.go`, removing its tests, and pruning specs and README. `recordEvaluation` (the callback that updated TOML counters from evaluate outcomes) goes with it — feedback writes those same counters directly.

**Tech Stack:** Go, `targ test` / `targ check-full`, `gh issue close`

---

## File Map

| File | Change |
|------|--------|
| `internal/evaluate/evaluator.go` | **Delete** (400 lines — the Evaluator struct and all its logic) |
| `internal/evaluate/evaluator_test.go` | **Delete** (1078 lines) |
| `internal/cli/cli.go` | **Modify** — remove import, `RenderEvaluateResult`, `case "evaluate"` dispatch, `RunEvaluate`, `runEvaluate`, `recordEvaluation`, `evaluateMaxTokens` constant, `errEvaluateMissingFlags` error, update usage string |
| `internal/cli/cli_test.go` | **Modify** — remove `evaluate` import, T-117 tests, T-118 test, `TestRunEvaluateNoToken`, and any evaluations-fixture helpers only used by those tests |
| `internal/cli/adapters_test.go` | **Modify** — remove `TestRunEvaluate_WithDataDir` |
| `internal/cli/export_test.go` | **Modify** — remove `ExportRunEvaluate` |
| `internal/cli/targets.go` | **Modify** — remove `EvaluateArgs` struct, `EvaluateFlags` func, `targ.Targ` entry for evaluate |
| `internal/cli/targets_test.go` | **Modify** — remove `"evaluate"` from the subcommand list test, remove `TestEvaluateFlags` |
| `README.md` | **Modify** — remove evaluate from hooks table and data files table |
| `docs/specs/architecture.toml` | **Modify** — remove evaluate ARCH nodes |
| `docs/specs/design.toml` | **Modify** — remove evaluate DES entries |

---

### Task 1: Delete `internal/evaluate/` package and remove all Go code references

**Files:**
- Delete: `internal/evaluate/evaluator.go`
- Delete: `internal/evaluate/evaluator_test.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/adapters_test.go`
- Modify: `internal/cli/export_test.go`
- Modify: `internal/cli/targets.go`
- Modify: `internal/cli/targets_test.go`

This task must delete everything together — you cannot run `targ test` until all the references are removed, since the package no longer exists.

- [ ] **Step 1: Confirm starting state compiles**

```bash
targ test 2>&1 | tail -5
```
Expected: all packages pass.

- [ ] **Step 2: Delete the evaluate package**

```bash
rm internal/evaluate/evaluator.go internal/evaluate/evaluator_test.go
rmdir internal/evaluate/
```

At this point compilation fails because `cli.go` still imports `"engram/internal/evaluate"`.

- [ ] **Step 3: Remove evaluate from `internal/cli/cli.go`**

Read `internal/cli/cli.go` first to find the exact line ranges. Then remove:

1. **Import** — remove `"engram/internal/evaluate"` from the import block.

2. **`RenderEvaluateResult` function** (lines ~54–77) — delete the entire exported function.

3. **`case "evaluate"` dispatch** (lines ~128–129) — delete the two-line case block from the `Run` switch.

4. **`RunEvaluate` function** (lines ~155–208) — delete the entire exported function.

5. **`evaluateMaxTokens` constant** (line ~477) and its usage at line ~864 — the usage is inside `RunEvaluate` (which you are deleting in item 4), so delete the constant after deleting the function to avoid an "undefined" error mid-edit.

6. **`errEvaluateMissingFlags` error** (line ~497) — delete it from the vars block.

7. **Usage string** (line ~515) — remove `"evaluate"` from the usage string. The string looks like:
   ```
   "usage: engram <audit|correct|surface|learn|evaluate|..."
   ```
   Remove `evaluate|` from the pipe-separated list.

8. **`recordEvaluation` function** (lines ~996–1008) — delete the entire function.

9. **`runEvaluate` function** (lines ~1203–1219) — delete the entire function.

- [ ] **Step 4: Remove evaluate from `internal/cli/cli_test.go`**

Read the file first. Remove:

1. `"engram/internal/evaluate"` import.
2. `TestRunEvaluateNoToken` test function (~lines 847–865).
3. `TestT117_EvaluateRunsFullPipeline` test function (~lines 1025–1083).
4. `TestT117_RunEvaluateWritesEvaluationLog` test function (~lines 1086–1143).
5. `TestT118_EvaluateWithoutTokenEmitsError` test function (~lines 1172–1190).
6. `TestT119_EvaluateSummaryFormat` test function (~line 1200) — calls `cli.RenderEvaluateResult` and uses `evaluate.Outcome`.
7. `TestT161_EvaluateStripsTranscript` test function (~line 1456) — calls `cli.RunEvaluate` with `evaluate.WithLLMCaller` and `evaluate.WithStripFunc`.
8. `TestT322_BinarySmokeTest` (~line 1817) — builds the binary and runs `engram evaluate --data-dir`. Once the subcommand is removed this test would fail; delete it or update it to remove the evaluate invocation. Check what else this smoke test covers before deleting entirely — if it tests other subcommands too, remove only the evaluate assertion.
9. Any fixture data (e.g., `fmt.Sprintf` strings building evaluations JSONL) that was only used by the deleted tests. Search for references to `"followed"` or `"evaluations"` JSONL strings near the deleted tests.

- [ ] **Step 5: Remove `ExportRunEvaluate` from `internal/cli/export_test.go`**

Read the file. Find and delete:
```go
// ExportRunEvaluate exposes the unexported runEvaluate for testing.
func ExportRunEvaluate(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
    return runEvaluate(args, stdout, stderr, stdin)
}
```

- [ ] **Step 6: Remove `TestRunEvaluate_WithDataDir` from `internal/cli/adapters_test.go`**

Read the file. Find and delete the `TestRunEvaluate_WithDataDir` function (~lines 311–330).

- [ ] **Step 7: Remove evaluate from `internal/cli/targets.go`**

Read the file. Remove:
1. `EvaluateArgs` struct.
2. `EvaluateFlags` function.
3. The `targ.Targ(func(a EvaluateArgs) { run("evaluate", EvaluateFlags(a)) }).Name("evaluate")...` entry from the targ registration block.

- [ ] **Step 8: Remove evaluate from `internal/cli/targets_test.go`**

Read the file. Remove:
1. `"evaluate"` from the subcommand list test (line ~159 — it's in a string slice).
2. `TestEvaluateFlags` function (~lines 252–270).

- [ ] **Step 9: Run `targ test` and confirm all packages pass**

```bash
targ test 2>&1 | grep -E "FAIL|ok" | tail -20
```
Expected: all packages pass, no FAIL lines.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "chore(evaluate): delete evaluate command — feedback replaces it (#348)

AI-Used: [claude]"
```

---

### Task 2: Update README.md

**Files:**
- Modify: `README.md`

Read `README.md` first. There are five evaluate references to remove:

- [ ] **Step 1: Remove the pipeline diagram node**

Around lines 14–23 there is a pipeline diagram (`Learn ──→ Surface ──→ Evaluate ──→ Maintain`) and a numbered step 3 that says something like "**Evaluate** — After each session, an LLM reads the (stripped) transcript…"

Remove the `Evaluate` node from the diagram arrow chain so it reads `Learn ──→ Surface ──→ Maintain`.
Remove (or renumber) step 3 entirely. Renumber subsequent steps if needed.

- [ ] **Step 2: Update the hooks table**

Around lines 99–100, the hooks table has rows for Compact and End that say "Unified flush: learning extraction, evaluate memory effectiveness, save session context."

Remove "evaluate memory effectiveness," from both rows. The new description:
"Unified flush: learning extraction, save session context."

- [ ] **Step 3: Remove the evaluations/*.jsonl row**

In the data files table (~line 110), delete the row:
```
| `evaluations/*.jsonl` | Per-session evaluation results (followed/contradicted/ignored) |
```

- [ ] **Step 4: Remove the UC-25 use-case table row**

Around line 182 there is a use-case table row:
```
| UC-25 | Evaluate Strip Preprocessing |
```
Delete that row.

- [ ] **Step 5: Run `targ test` to confirm nothing broke**

```bash
targ test 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add README.md
git commit -m "docs: remove evaluate command from README (#348)

AI-Used: [claude]"
```

---

### Task 3: Remove evaluate from specs

**Files:**
- Modify: `docs/specs/architecture.toml`
- Modify: `docs/specs/design.toml`

The specs have several ARCH and DES nodes that describe evaluate. Remove the evaluate-specific nodes entirely. Leave nodes that describe things that still exist (surfacing log writer, effectiveness reading from TOML counters, etc.).

**Guidance for identifying what to remove:** Search for `evaluate` in each file. Remove entire `[[arch]]` or `[[des]]` blocks whose primary subject is evaluate. Be conservative — if a block is mostly about something else (like surfacing) but mentions evaluate in passing, update the description rather than deleting the block.

- [ ] **Step 1: Update `docs/specs/architecture.toml`**

Read the file. Find and remove these evaluate-specific ARCH nodes (identified by name/subject):

1. The ARCH block named **"Outcome Evaluation Pipeline"** (ARCH-23) — describes `Evaluator.Evaluate()`, surfacing log read-and-clear, evaluations JSONL output.

2. The ARCH block named **"Surfacing Log Isolation"** (ARCH-81) — describes atomic rename pattern for evaluate pipeline isolation.

3. The ARCH block named **"Hook Integration — evaluate Subcommand"** — describes `runEvaluate` CLI wiring.

4. The ARCH-P3-6 block **"EvalLinkUpdater interface in evaluate package"** — P3 feature for correlation links.

5. The ARCH block named **"Evaluate Strip Integration (UC-25)"** (line ~1704) — describes extending `Evaluator` with `WithStripFunc`.

6. Any ARCH blocks that describe the evaluate step as part of the flush pipeline — but note that the flush pipeline ARCH nodes already say learn → context-update, so just remove any remaining references to evaluate being in the pipeline.

7. In the surfacing ARCH nodes (lines ~1773–1792) that mention `"LogSurfacing (for evaluate pipeline)"` and `"so evaluate pipeline picks them up"` — update these comments to remove the evaluate reference. The surfacing log still exists and is still written; just remove evaluate as its consumer.

8. In the always-loaded-sources ARCH node (line ~1860) that ends with "The evaluate pipeline (Stop hook) requires no code changes..." — remove or rewrite that sentence.

**After editing:** Search for remaining `evaluate` occurrences and clean up stray references.

- [ ] **Step 2: Update `docs/specs/design.toml`**

Read the file. Find and remove/update:

1. The DES block for the surfacing log (line ~72) that says "read-and-cleared by `engram evaluate`" — update to say the log is cleared by `engram flush`.

2. The DES block describing `evaluations/*.jsonl` format (line ~111 — file path `<data-dir>/evaluations/...`, JSON schema) — delete the entire block.

3. The DES block named **"Hook wiring — evaluate in PreCompact and SessionEnd"** (line ~152) — describes the hook scripts invoking `engram evaluate`. Delete the entire block.

4. An AC (acceptance criteria) item (line ~330) that says something like "(3) Evaluate hook reads TOML, increments appropriate counter, writes back atomically." — remove this item from the AC list and renumber remaining items if needed.

5. Any DES entries for the evaluate subcommand wiring (DES-15 or similar).

**After editing:** Search for remaining `evaluate` occurrences and clean up stray references.

- [ ] **Step 3: Run `targ test` to confirm nothing broke**

```bash
targ test 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add docs/specs/architecture.toml docs/specs/design.toml
git commit -m "docs(specs): remove evaluate ARCH/DES nodes (#348)

AI-Used: [claude]"
```

---

### Task 4: Final check, delete data artifact, close issue

- [ ] **Step 1: Run full quality check**

```bash
targ check-full 2>&1 | tail -15
```
Expected: only `check-uncommitted` may fail (if any changes remain unstaged). All linters, nil checks, coverage pass.

- [ ] **Step 2: Fix any issues**

If `reorder-decls-check` fails: run `targ reorder-decls`, then commit:
```bash
targ reorder-decls
git add -A
git commit -m "chore: fix declaration ordering after evaluate removal

AI-Used: [claude]"
```

If lint fails: read the full output and fix before proceeding.

- [ ] **Step 3: Delete data artifacts**

```bash
rm -rf ~/.claude/engram/data/evaluations/
```

(No-op if the directory doesn't exist.)

- [ ] **Step 4: Close issue #348**

```bash
gh issue close 348 --comment "Deleted the evaluate command and its internal/evaluate package. The flush pipeline already ran learn → context-update (done in prior work). This PR removes all remaining evaluate code, tests, CLI wiring, README references, and spec nodes. Outcome counters (followed/contradicted/ignored) are now updated exclusively by the explicit feedback command."
```
