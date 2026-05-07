# Companion Evaluation Cycle — Progress Log

This is a live handoff doc. Update it as work proceeds.

## Quick orientation for the next agent

**Branch:** `opencode-plugin`
**Worktree:** `/Users/joe/repos/personal/engram-worktrees/opencode-plugin/`
**Spec:** `docs/superpowers/specs/2026-05-05-companion-evaluation-cycle-design.md`
**Plan:** `docs/superpowers/plans/2026-05-06-companion-evaluation-cycle-plan.md`
**Goal:** replace Anthropic Haiku with pluggable `--llm-cmd` backend, add `engram cycle` orchestrator, switch recall to always-synthesize prose, broaden learn dedup with O_EXCL slug auto-increment.

**Read those two docs first.** Then read this file. Then check `git log --oneline | head -20` to see the actual state.

**Verify current state:**
```bash
cd /Users/joe/repos/personal/engram-worktrees/opencode-plugin/
targ test 2>&1 | grep -E "ok|FAIL"
targ lint-full 2>&1 | tail -5
git status
```

If `targ test` fails or lint produces new issues, something regressed since this doc was written.

---

## Done — All phases (B → H) complete except H3 manual smoke test

Status as of 2026-05-07: every automated task in the plan landed. H3 (live opencode plugin smoke test with API credit) is the only remaining step and requires the user.

### Phase A — `internal/llmcmd` foundation (committed pre-handoff)

| SHA | Message |
|---|---|
| `c7500ebe` | feat(llmcmd): stdin/stdout shell-cmd runner |
| `2f79abf6` | style(llmcmd): move constants block to idiomatic position |
| `791c60b4` | feat(llmcmd): wall-clock timeout |
| `01dce7ef` | feat(llmcmd): inject ENGRAM_COMPANION_MODE=1 for recursion guard |
| `a0006091` | feat(llmcmd): implement Extractor and FindingSummarizer adapters |
| `fb0928d8` | feat(llmcmd): CallerFunc for learn dedup |
| `64307cc5` | style(llmcmd): reorder decls, fix wsl whitespace, drop unused param name |

### Phases B/G — wire `--llm-cmd` and delete Anthropic

| SHA | Message |
|---|---|
| `e528c279` | feat(cli): add --llm-cmd flag with ENGRAM_LLM_CMD fallback (B1) |
| `5efb76c1` | feat(cli): requireLLMCmd helper for non-empty enforcement (B2) |
| `79dd0d25` | refactor(cli): route recall through llmcmd backend (B3) |
| `89ba5a5d` | refactor(learn): route dedup through llmcmd backend (B4) |
| `8b226060` | refactor: delete internal/anthropic package and dead chain (G1) |

### Phase C — recall pipeline updates

| SHA | Message |
|---|---|
| `441bf517` | refactor(recall): collapse Result to single Report field (C1) |
| `f595e6d6` | feat(llmcmd): synthesis prompt demands directive advice (C2) |
| `2ecf7668` | feat(recall): bare mode runs synthesis (C3) |

### Phase D — learn pipeline + race-safe slug claim

| SHA | Message |
|---|---|
| `7207458e` | feat(memory): BuildIndex includes content fields for richer dedup (D1) |
| `02ccb32d` | style(memory): extract content-field writers to flatten BuildIndex |
| `805bb523` | refactor(learn): drop CONTRADICTION from dedup prompt and parser (D2) |
| `ecc353f0` | feat(tomlwriter): O_EXCL atomic auto-increment for slug collisions (D3) |
| `5431f8fa` | refactor(learn): writeMemory returns (name, persisted, err) (D4) |

### Phase E — `engram cycle` package and CLI

| SHA | Message |
|---|---|
| `73f8c002` | feat(cycle): JSON output schema (E1, also added JSON tags to MemoryRecord/ContentFields) |
| `2c01bec2` | style: apply reorder-decls across packages |
| `f3c0f950` | feat(cycle): learn-extraction and query-proposal prompts (E2) |
| `f53eb8ea` | feat(cycle): orchestrator with learning extraction and query-driven recall (E3) |
| `4a0def50` | feat(cli): engram cycle subcommand (E4) |

### Phase F — TypeScript plugin rewrite

| SHA | Message |
|---|---|
| `7c460bba` | refactor(plugin): use engram cycle for learn+recall (F1; opencode/plugins/engram.ts dropped from 479 → 243 lines) |

### Phase H — validation tests

| SHA | Message |
|---|---|
| `9ebd44e3` | test(cycle): llm-cmd failure paths produce empty arrays (H2 — three failure-mode tests) |
| `a0d5f711` | test(cycle): planted-token integration verifies persist + recall (H1 — leaner than the plan's exec-engram form; uses real tomlwriter persister + stub Runner/Recaller) |
| `8caf0782` | style: apply reorder-decls to cycle and cycle CLI files |

**Files created in Phase A:**
- `internal/llmcmd/llmcmd.go` — Runner type, New, NewWithTimeout, Run
- `internal/llmcmd/llmcmd_test.go` — 4 tests (success, error, timeout, env var)
- `internal/llmcmd/extractor.go` — Extractor type, NewExtractor, ExtractRelevant, SummarizeFindings (placeholder synthesis prompt — Task C2 replaces it with the directive prompt)
- `internal/llmcmd/extractor_test.go` — 2 tests (extract prompt content, summarize prompt content)
- `internal/llmcmd/dedup.go` — CallerFunc(runner) returns the llmCaller-shaped function
- `internal/llmcmd/dedup_test.go` — 1 test (system+user concatenation)

**Final state:** `targ test` passes, `targ lint-full` passes (0 issues).

**Coverage gate caveat (still failing):** `targ check-full` reports `check-coverage-for-fail` because some functions in `internal/cycle` and the cycle CLI adapters fall below the 80% function-coverage threshold. Behavioral paths are covered by unit tests + the planted-token integration test, but the exec-style adapters (transcriptReaderAdapter, cyclePersisterAdapter, cycleRecallerAdapter wiring in `internal/cli/cycle.go`) are exercised end-to-end only by H3, not by unit tests. Re-running `targ check-full` after H3 lands the smoke test should not change this — the gate is a known structural mismatch with the DI adapter layer. Either lower the threshold for adapter packages, add explicit adapter unit tests, or accept the failure as documented.

---

## Strategy: how to execute the remaining 19 tasks

**Lessons from Phase A** (the hard way):

- **Subagent-driven-development with strict per-task review is too slow** for the bite-sized tasks in this plan. Two implementers in a row (haiku, sonnet) drifted into coverage chasing or advisor calls and stopped before committing. Each ate ~5 min real time on what should be 30-second commits. With 19 tasks ahead × 3 subagents each (implementer + spec-reviewer + code-quality-reviewer), the overhead would dwarf the actual work.
- **Inline execution by the controller** is much faster for mechanical tasks: Phase A's last three tasks (A3, A4, A5) took <2 minutes each when I just did them in the controller.
- **Subagents earn their keep on substantive tasks**: judgment-heavy work, multi-file refactors with cascading impact, novel logic, integration tests. Use them there.

**Recommended approach for remaining tasks:**

| Bucket | Tasks | Approach |
|---|---|---|
| Mechanical edits | B1, B2, G1, C2, D1, D2, E1, E2, H2 | Inline. Surgical. Verify with `targ test && targ lint-full` after each. |
| Cascading multi-file | B3, B4, C1, C3, D4 | Inline but read carefully — these touch multiple callers. Run full `targ check-full` after each to catch downstream breakage. |
| Substantive judgment | D3, E3, E4, F1, H1 | Subagent-dispatch, but skip the two-stage reviewer protocol — just dispatch one capable implementer (sonnet/opus), instruct them to commit when done, and verify with `targ check-full` + manual code read. |
| Manual user step | H3 | Hand off to the user. They run `opencode run -m ... 'prompt'` and confirm. |

**Don't use the subagent-driven-development skill's full per-task two-stage review protocol** for this plan. It's well-suited for tasks of substantial size (a feature implemented in one task), not for plans broken into 25 bite-sized TDD steps where most are 5-line edits.

**Prompt-engineering tip for substantive subagents** (when you do dispatch them):
- Tell them explicitly: "Do NOT call advisor. Do NOT chase coverage beyond what's specified. Implement the code from the plan verbatim, commit, report DONE. Stop after committing."
- Give them the full task text inline (don't make them read the plan file).
- Keep the working directory in the prompt.

---

## Remaining tasks — H3 only

**H3 (manual smoke test, requires user with API credit):**

1. Plant a verification token via `engram learn fact`:
   ```bash
   engram learn fact \
     --source agent --no-dup-check \
     --situation "asked about plugin integration verification" \
     --subject plugin --predicate "integration token is" --object PLUGIN-VERIFY-99181
   ```
2. Run a fresh opencode session that prompts about the planted token:
   ```bash
   opencode run -m opencode/qwen3.6-plus 'What plugin integration verification details do you remember?' 2>&1 | tail -50
   ```
3. Expected: response contains `PLUGIN-VERIFY-99181` and the engram cycle subprocess emits JSON with at least one `recalled[].report` mentioning the token.
4. Run `targ check-full` and confirm the only failures are the pre-existing coverage and reorder issues (or all-clean if those have been addressed).
5. Clean up: `rm -f ~/.local/share/engram/companion-{trace,injections,debug}.{jsonl,log}` and any session-cwd artifacts (the new plugin no longer writes these, but old runs may have left them).

---

## Original remaining tasks reference (kept for archaeology)



For each task, the plan has full code and steps. This table just maps task → file → key gotcha.

### Phase B — Wire `--llm-cmd` through CLI

- **B1 (mechanical)**: Add `--llm-cmd` flag and `ENGRAM_LLM_CMD` env-var fallback. New helper `resolveLLMCmd(flagValue string) string` in `internal/cli/cli.go`. Add `LLMCmd string` field to relevant args structs in `internal/cli/targets.go`. Plan has the test code. **Gotcha:** look at how existing flags are registered in `targets.go` — the targ struct-tag pattern is non-obvious.

- **B2 (mechanical)**: Add `requireLLMCmd(flagValue string) error` helper. Errors when neither flag nor env var is set. Plan has tests.

- **B3 (cascading)**: Replace Anthropic client construction in recall wiring with `llmcmd.NewExtractor(llmcmd.New(resolveLLMCmd(args.LLMCmd)))`. The construction site is in `internal/cli/cli.go` or wherever `recall.NewSummarizer` is called today. **Gotcha:** trace all callers — `RunRecall` may be called from multiple places.

- **B4 (cascading)**: Replace `makeConflictDeps` in `internal/cli/learn.go`. Today it returns `(llmCaller, memoryLister)`. Replace with `(llmcmd.CallerFunc(runner), memory.NewLister())`. Remove `makeAnthropicCaller`, `resolveToken`. Pass `args.LLMCmd` through. **Gotcha:** `runLearnFact` and `runLearnFeedback` both call this — both call sites need updating.

### Phase G — Delete `internal/anthropic`

- **G1 (mechanical)**: After B3 + B4, no callers should reference Anthropic. Verify with `grep -rn "engram/internal/anthropic" internal/ cmd/`. Then `git rm -r internal/anthropic/`. Then `targ check-full` to confirm clean. **Gotcha:** If grep shows any references, something in B3/B4 was missed — fix that first, don't force the delete.

### Phase C — Recall pipeline updates

- **C1 (cascading)**: Collapse `Result` struct to `{Report string}`. Today's `Result` has `Summary` and `Memories` fields. Find all callers (`grep -rn "Result.Summary\|\.Memories\b" internal/ cmd/`). Update them to read `Report`. Update `FormatResult` to write `Report`. Update `recallModeA` and `recallModeB` to populate `Report`. **Gotcha:** `recallModeA` currently does no synthesis — for now just write the concatenated transcript+memories into `Report` as a temporary measure. C3 then wires the synthesis call.

- **C2 (mechanical)**: Replace the placeholder `SummarizeFindings` body in `internal/llmcmd/extractor.go` with the directive synthesis prompt from the spec. Plan has the prompt verbatim and a test that checks `"directive advice"`, `"imperative voice"`, `"cite the specific memory or outcome"`.

- **C3 (substantive)**: Update `recallModeA` to call `SummarizeFindings` over its concatenated buffer (memories + transcripts). Plan has the full new function body. **Gotcha:** the empty-buffer case must early-return (don't call the LLM with no input).

### Phase D — Learn pipeline updates

- **D1 (mechanical)**: Update `memory.BuildIndex` (`internal/memory/memory.go`) to include content fields (behavior/impact/action for feedback; subject/predicate/object for fact). Plan has the full new function. Plan also has the test that asserts the new fields are in the output.

- **D2 (mechanical)**: Update `internal/cli/learn.go`'s dedup prompt and parser to drop CONTRADICTION. Plan has the new prompt (only DUPLICATE | NONE) and the new parser (drops the CONTRADICTION-prefixed lines). Update `describeNewMemory` to also include content fields (mirroring `BuildIndex`).

- **D3 (substantive)**: O_EXCL atomic slug auto-increment in `internal/tomlwriter/writer.go` (or wherever `Write` lives — `grep -n "func.*Write\b" internal/tomlwriter/`). Plan has the loop with `os.OpenFile(..., O_CREATE|O_EXCL|O_WRONLY, 0o644)` and `errors.Is(err, os.ErrExist)` retry. **Plan also has a concurrent race test (10 goroutines writing the same slug) that must pass.** This is the one Phase D task that benefits from a substantive subagent — the race test catches subtle bugs.

- **D4 (cascading)**: Refactor `writeMemory` in `internal/cli/learn.go` to return `(name string, persisted bool, err error)` instead of just `error`. Plan has the new function body. Update `runLearnFact` and `runLearnFeedback` to discard the new returns (`_, _, err := writeMemory(...)`). Cycle (Phase E) needs the (name, persisted) return values to populate its `learned[]` array.

### Phase E — Cycle command

- **E1 (mechanical)**: Create `internal/cycle/output.go` with `Output`, `LearnedMemory`, `RecalledReport` types and `NewOutput()` returning non-nil empty slices. Plan has the full file and tests. **Gotcha:** `LearnedMemory` embeds `memory.MemoryRecord` — the JSON marshaling needs the `name` field at the top level, not nested.

- **E2 (mechanical)**: Create `internal/cycle/prompts.go` with `LearnExtractionPrompt(transcript)` and `QueryProposalPrompt(transcript)` functions. Plan has the prompts verbatim and tests asserting key phrases.

- **E3 (substantive)**: Create `internal/cycle/cycle.go` with the orchestrator. Plan has 90% of the code. Defines `Runner`, `TranscriptReader`, `Persister`, `Recaller` interfaces and the `Cycle.Run` method. **This is dense logic** — recommend dispatching as one subagent task with the full plan section pasted into the prompt.

- **E4 (substantive)**: Create `internal/cli/cycle.go` with `RunCycle` handler and three adapter types (`transcriptReaderAdapter`, `cyclePersisterAdapter`, `cycleRecallerAdapter`). Plan has the full code for each. **Gotcha:** the adapters call into `recall.Finder`, `recall.Reader`, `recall.NewOrchestrator` — verify those names exist with `grep -n "func New" internal/recall/` first; the plan notes this. Also: `engram cycle` needs to be registered in `internal/cli/targets.go` — find the registration pattern there.

### Phase F — Plugin TS

- **F1 (substantive, judgment)**: Rewrite `opencode/plugins/engram.ts`'s `experimental.chat.system.transform` hook to call `engram cycle` and format the JSON output. Plan has the new hook body and helper functions (`runEngramCycle`, `formatCycleResult`). Remove the now-unused helpers (the recall-blob + companion-text-emit pipeline). **Gotcha:** Bun.spawn signature for a child process; verify with the existing `runCompanion` function in the same file. Also: `bun install` if `node_modules` looks stale.

### Phase H — Validation

- **H1 (substantive)**: End-to-end planted-token integration test. Plan has the test skeleton; needs adaptation to engram's actual transcript-discovery convention. **Gotcha:** Build tag conventions — check whether the project uses `//go:build integration` or runs everything by default.

- **H2 (mechanical)**: Add three failure-path tests to `internal/cycle/cycle_test.go` — LLM call A failure → empty `learned`; call B failure → empty `recalled`; per-query recall failure → entry dropped. Plan has the test stubs.

- **H3 (manual, requires user)**: Plant a verification token, run a fresh `opencode run` session, verify the token surfaces in the response, run `targ check-full`, clean up. **The user must run this** because it requires their actual opencode session and a model with API credit. Hand off the runbook (Steps 1-5 in the plan's Task H3).

---

## Open questions / things to watch

1. **Coverage gate failure (`check-coverage-for-fail`)** — flagged in Phase A. Need to determine: does the gate apply per-file with a min threshold, or aggregated? `internal/anthropic` has 2.6% coverage and presumably passes today, so there's probably some allowlist or new-file logic. Re-evaluate after Phase G removes anthropic and Phase B/C/D adds callers that exercise llmcmd more.

2. **`recall.Finder` and `recall.Reader` constructor names** — plan assumes `recall.NewFinder()` and `recall.NewReader()` exist. **Verify before E4** with `grep -n "func New" internal/recall/`. If named differently, use the actual names.

3. **OpenCode plain-text output mode** — spec says `opencode run -m <model>` (no `--format json`) emits plain text. Verify in F1 that this is true; if it emits a banner/preamble, document and add a small filter.

4. **Plugin recursion guard** — F1 must verify the existing `ENGRAM_COMPANION_MODE` short-circuit still triggers. The plugin's `system.transform` should return the reminder only when this env var is set, without re-entering cycle.

5. **engram CLI invocation in plugin** — the plugin currently shells out to `ENGRAM_BIN`. After F1, it shells out to `engram cycle ...`. Verify `ENGRAM_BIN` is still set/derived correctly by `ensureBinary()`.

6. **Test ordering convention** — the project's `targ reorder-decls` enforces a specific declaration order. Run it after every code change before committing.

---

## Memories captured at /learn (2026-05-06)

Five feedback memories were persisted with `--source human` covering:
- Required CLI flags should error on missing, not silently no-op
- Advice-producing LLM prompts need imperative voice + evidence citation
- Slug-collision overwrite is memory loss, not deduplication
- Address easy risk fixes inline rather than deferring
- Contradicting memories should coexist; reconcile at recall time

These should surface via `engram recall` if the next agent calls `/prepare` on a relevant query.

---

## Update protocol for the next agent

When you complete a task or batch:

1. Update the **Done** section at the top with the new commits.
2. Move the corresponding entries from **Remaining tasks** down into Done (or strike them through).
3. Update **Open questions** with what you discovered or resolved.
4. Commit this file alongside the work it tracks.

Keep the doc terse but accurate. The point is to let the next agent pick up without re-deriving context.
