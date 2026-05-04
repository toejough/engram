# Phase 7: Companion-Emitted Queries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the companion's "filter memories" prompt with a "generate queries" prompt; have the plugin run each query through `engram recall --query <q>` and inject the concatenated per-query results into the primary's system prompt.

**Architecture:** Single-file edit to `opencode/plugins/engram.ts`. The `experimental.chat.system.transform` hook keeps its outer shape (recursion guard, project-scoped recall blob, companion call). The post-companion handling switches from "inject companion text directly" to "parse queries → parallel `engram recall --query` → concatenate per-query results → inject." Companion session persistence and the `ENGRAM_COMPANION_MODE` recursion guard are unchanged.

**Tech Stack:** TypeScript, Bun (`Bun.spawn`), opencode plugin API (`experimental.chat.system.transform`), engram CLI (`recall --query`).

**Spec:** `docs/companion-memory-steward.md` — Phase 7.

---

## File Structure

**Modified:**
- `opencode/plugins/engram.ts` — only code file. Replaces the `COMPANION_PROMPT_PREFIX` constant, adds `runEngramRecallWithQuery`, updates `logCompanionInjection` signature, replaces the post-companion block in the system.transform handler.
- `docs/companion-memory-steward.md` — Phase 7 validation findings appended after the Phase 7 spec section, status updated when complete.

No new files. No new dependencies.

---

## Task 1: Implement the new flow in engram.ts

**Files:**
- Modify: `opencode/plugins/engram.ts`

- [ ] **Step 1: Confirm baseline file shape**

Run: `wc -l opencode/plugins/engram.ts && grep -nE "COMPANION_PROMPT_PREFIX|runEngramRecall|logCompanionInjection|system.transform" opencode/plugins/engram.ts`

Expected: ~368 lines; line numbers for `COMPANION_PROMPT_PREFIX` (~59), `runEngramRecall` (~136), `logCompanionInjection` (~109), `experimental.chat.system.transform` (~208). Locks the targets for the next steps.

- [ ] **Step 2: Replace COMPANION_PROMPT_PREFIX**

In `opencode/plugins/engram.ts`, replace the entire `COMPANION_PROMPT_PREFIX` template literal (the constant declaration, including its trailing backtick + newline) with:

```typescript
const COMPANION_PROMPT_PREFIX = `You are a memory steward observing a primary AI agent's project session. Read the project history below and propose 3 to 5 targeted recall queries that would surface helpful past memories about what is currently happening.

Output the queries only, one per line, no numbering, no commentary, no other text. Each query should be 5 to 15 words capturing a specific facet you want to recall about.

If nothing in the history is worth recalling on, output exactly:
NO QUERIES

PROJECT HISTORY (most recent message at end):
`
```

- [ ] **Step 3: Add runEngramRecallWithQuery helper**

Insert this function immediately after `runEngramRecall` (the existing function around line 136):

```typescript
async function runEngramRecallWithQuery(query: string): Promise<string> {
  const proc = Bun.spawn(
    [ENGRAM_BIN, "recall", "--query", query, "--no-external-sources"],
    { stdout: "pipe", stderr: "pipe" },
  )
  await proc.exited
  return (await proc.stdout.text()).trim()
}
```

- [ ] **Step 4: Replace logCompanionInjection with the new shape**

Replace the entire existing `logCompanionInjection` function (the one that takes `injection: string`) with:

```typescript
function logCompanionInjection(
  sessionID: string,
  recallMs: number,
  companionMs: number,
  recallOutput: string,
  queries: string[],
  perQueryResults: { query: string; result: string }[],
): void {
  try {
    const dir = path.dirname(COMPANION_INJECTIONS)
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
    const ts = new Date().toISOString()
    const blocks = perQueryResults
      .map(({ query, result }) => `--- QUERY: ${query} ---\n${result}`)
      .join("\n\n")
    const entry = [
      `=== ${ts} primary=${sessionID} recall=${recallMs}ms companion=${companionMs}ms recallChars=${recallOutput.length} queries=${queries.length} ===`,
      `--- RECALL OUTPUT (input to companion) ---`,
      recallOutput,
      `--- COMPANION OUTPUT (queries emitted) ---`,
      queries.join("\n"),
      `--- SECONDARY RECALL RESULTS (memories injected) ---`,
      blocks,
      `=== END ===`,
      "",
      "",
    ].join("\n")
    fs.appendFileSync(COMPANION_INJECTIONS, entry, "utf8")
  } catch {
    // logging failure is non-fatal
  }
}
```

- [ ] **Step 5: Replace the post-companion handling block in system.transform**

In the `experimental.chat.system.transform` handler, find the block that starts with:

```typescript
        if (companionOutput && !companionOutput.includes("NO RELEVANT MEMORIES")) {
```

Replace from that `if` through the matching closing `else { ... }` brace (i.e., the entire emit-or-skip section, but NOT the surrounding `try {` / `catch (err: any) {` block) with:

```typescript
        const SENTINEL = "NO QUERIES"
        const MAX_QUERIES = 5
        const allLines = companionOutput.split("\n").map((s) => s.trim()).filter(Boolean)
        const sentinelOnly = allLines.length === 1 && allLines[0] === SENTINEL
        const queries = allLines.filter((l) => l !== SENTINEL).slice(0, MAX_QUERIES)

        if (allLines.length === 0) {
          companionTrace("companion-skipped", { sessionID, reason: "empty-output" })
        } else if (sentinelOnly || queries.length === 0) {
          companionTrace("companion-skipped", { sessionID, reason: "no-queries" })
        } else {
          const queryStart = Date.now()
          const perQueryResults = await Promise.all(
            queries.map(async (query) => {
              const start = Date.now()
              const result = await runEngramRecallWithQuery(query)
              companionTrace("secondary-recall-complete", {
                sessionID, query, ms: Date.now() - start, resultLen: result.length,
              })
              return { query, result }
            }),
          )
          const totalQueryMs = Date.now() - queryStart

          let block = "## Recalled memories\n\n"
          for (const { query, result } of perQueryResults) {
            block += `### Query: ${query}\n${result}\n\n`
          }
          companionBlock = "\n\n" + block.trimEnd()

          companionTrace("companion-injected", {
            sessionID, blockLen: companionBlock.length, queryCount: queries.length, totalQueryMs,
          })
          logCompanionInjection(
            sessionID || "default", recallMs, companionMs, recallOutput, queries, perQueryResults,
          )
        }
```

The surrounding `try { ... } catch (err: any) { companionTrace("companion-error", ...) }` MUST be preserved unchanged — it covers thrown errors from `runEngramRecall`, `runCompanion`, or anything in this block.

- [ ] **Step 6: Verify no stale references to the old companion injection format**

Run: `grep -nE "NO RELEVANT MEMORIES|injection: string" opencode/plugins/engram.ts`

Expected: no matches. (The old sentinel and old logCompanionInjection signature are gone.)

- [ ] **Step 7: Bundle-check the plugin compiles**

Run: `cd opencode && bun build plugins/engram.ts --target=bun --outdir=/tmp/engram-build-check 2>&1 | tail -20`

Expected: `Bundled ... bytes` in output, no `error:` lines, no TypeScript diagnostics.

If any error: read the line, fix in `engram.ts`, re-run.

- [ ] **Step 8: Commit**

```bash
git add opencode/plugins/engram.ts
git commit -m "$(cat <<'EOF'
feat(opencode-plugin): wire companion-emitted queries through recall

Companion now emits 3-5 targeted queries (one per line, NO QUERIES sentinel
for skip). Plugin runs each through engram recall in parallel, injects
the concatenated per-query results as a ## Recalled memories block.
Replaces the Phase 5 filter-only companion pass.

Spec: docs/companion-memory-steward.md Phase 7.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: Smoke test — observe one full primary turn end-to-end

**Files:** none modified.

- [ ] **Step 1: Build engram if stale**

Run: `targ build && ls -la ~/.local/bin/engram`

Expected: build succeeds, binary mtime is recent.

- [ ] **Step 2: Reset companion session for the engram repo**

Find the primary session ID for any prior engram-repo session you'd be reusing — easier to just clear them all:

Run: `rm -f ~/.local/share/engram/companion-session/*.txt`

Expected: directory is empty.

- [ ] **Step 3: Tail trace + injection logs in a separate terminal**

Open a second terminal and run: `tail -F ~/.local/share/engram/companion-trace.jsonl ~/.local/share/engram/companion-injections.log`

Expected: empty initially; will stream as you trigger turns.

- [ ] **Step 4: Run a primary turn from the engram repo**

Run: `cd /Users/joe/repos/personal/engram-worktrees/opencode-plugin && opencode run -m opencode/qwen3.6-plus "I want to refactor the auth middleware to add token bucket rate limiting"`

Expected: opencode completes, returns a response. In the tail terminal you should see, in this order: `system.transform-start`, `recall-complete`, `companion-complete`, multiple `secondary-recall-complete` events (one per query, all close together because parallel), then `companion-injected`. The injection log should append a new `=== ... queries=N ===` block.

- [ ] **Step 5: Verify injection log shape**

Run: `grep -c "=== END ===" ~/.local/share/engram/companion-injections.log` to find the latest entry, then inspect the bottom of the file. The latest block must contain (in order): a header with `queries=N` matching the trace, a `--- COMPANION OUTPUT (queries emitted) ---` section listing N queries one-per-line, and a `--- SECONDARY RECALL RESULTS ---` section with N `--- QUERY: ... ---` sub-blocks.

If `queries=0` or the section is empty, the smoke failed — re-check Task 1 step 5.

- [ ] **Step 6: Verify the system prompt actually got the injection**

Run: `awk '/=== SYSTEM TRANSFORM ===/{flag=NR} END{print flag}' ~/.local/share/engram/debug-system-transform.log` to find the latest entry's start, then read from there to end of file. Look inside the `--- AFTER ---` section for `## Recalled memories` followed by `### Query:` headers and recall summaries.

Expected: present and well-formed.

- [ ] **Step 7: If anything failed — diagnose**

Common failure modes and fixes:
- No `secondary-recall-complete` events but companion-skipped fired → companion emitted sentinel or empty. Inspect `--- COMPANION OUTPUT ---` section in injection log.
- `secondary-recall-complete` events fire but `companion-injected` doesn't → exception in concatenation; check `companion-error` trace event for stack.
- `engram recall --query "<q>"` fails directly when run by hand → engram CLI bug, not plugin bug; investigate engram, not engram.ts.

If a fix is needed in `engram.ts`, edit, re-run from Step 4, and amend the previous commit if the fix is small (`git commit --amend --no-edit`). If substantive, new commit.

- [ ] **Step 8: No commit needed if smoke passed cleanly.** If a fix was applied, ensure it's committed.

---

## Task 3: Validation 1 — Phase 4 scenario replay

Method: re-run Phase 4's three scenarios with the new prompt. Bypass the full pipeline (avoid DB inserts); construct each scenario as a synthetic recall-blob string and feed directly to the companion via `opencode run`.

**Files:**
- Create: `/tmp/engram-phase7-scen-pivot.txt`, `/tmp/engram-phase7-scen-sparse.txt`, `/tmp/engram-phase7-scen-tool-result.txt` (scenario inputs; not committed)
- Modify: `docs/companion-memory-steward.md` (append findings)

- [ ] **Step 1: Construct scenario A (pivot) blob**

Write to `/tmp/engram-phase7-scen-pivot.txt`:

```
PROJECT SESSION HISTORY (most recent message at end):

USER: I want to add token-bucket rate limiting to our API. Where should I start?
ASSISTANT: For token-bucket rate limiting, you'd typically add middleware that tracks request counts per client. Look at internal/middleware/ for similar patterns. We'd need a Redis-backed counter or in-memory store depending on scale.
USER: Hmm — actually wait. I just hit a 401 again on a request that should've passed. The auth middleware is throwing intermittent token validation errors. Let's fix that first before the rate-limiting work.
```

- [ ] **Step 2: Construct scenario B (sparse) blob**

Write to `/tmp/engram-phase7-scen-sparse.txt`:

```
PROJECT SESSION HISTORY (most recent message at end):

USER: I'm thinking about refactoring the auth + rate-limit modules together. Should I create a worktree for this?
ASSISTANT: Yes — for any multi-file refactor that touches multiple subsystems, a worktree keeps the change isolated and lets you abandon the branch if the approach doesn't pan out. Want me to set one up?
USER: yes do it
```

- [ ] **Step 3: Construct scenario C (tool-result) blob**

Write to `/tmp/engram-phase7-scen-tool-result.txt`:

```
PROJECT SESSION HISTORY (most recent message at end):

USER: ls internal/recall/
ASSISTANT: ran `ls internal/recall/`. Output:
  automemory_phase.go     claudemd_phase.go    cost_test.go
  debug-trace.log         orchestrate.go       skill_phase.go
  summarize.go            summarize_test.go

Note: `debug-trace.log` is 47MB — that looks like it shouldn't be in the package directory.
USER: wait what — yeah that should not be there.
```

- [ ] **Step 4: Run companion against scenario A**

The companion prompt is `COMPANION_PROMPT_PREFIX` + scenario blob. Construct it:

```bash
cat <(printf 'You are a memory steward observing a primary AI agent'\''s project session. Read the project history below and propose 3 to 5 targeted recall queries that would surface helpful past memories about what is currently happening.\n\nOutput the queries only, one per line, no numbering, no commentary, no other text. Each query should be 5 to 15 words capturing a specific facet you want to recall about.\n\nIf nothing in the history is worth recalling on, output exactly:\nNO QUERIES\n\nPROJECT HISTORY (most recent message at end):\n') /tmp/engram-phase7-scen-pivot.txt > /tmp/engram-phase7-prompt-pivot.txt

opencode run -m opencode/qwen3.6-plus --format json "$(cat /tmp/engram-phase7-prompt-pivot.txt)" 2>/dev/null \
  | jq -r 'select(.type == "text") | .part.text' > /tmp/engram-phase7-output-pivot.txt

cat /tmp/engram-phase7-output-pivot.txt
```

Expected: 3-5 query lines, no numbering, no commentary.

- [ ] **Step 5: Run companion against scenario B**

Same as step 4, swapping `pivot` → `sparse` in all paths.

- [ ] **Step 6: Run companion against scenario C**

Same, `pivot` → `tool-result`.

- [ ] **Step 7: Append findings to docs/companion-memory-steward.md**

Open `docs/companion-memory-steward.md` and append immediately after the Phase 7 spec's "Cost note" section:

````markdown
### Phase 7 — Validation findings (2026-05-03)

#### Validation 1 — scenario replay

Re-ran Phase 4's three scenarios against the new prompt, using `opencode/qwen3.6-plus`.

| Scenario | Companion's queries |
|---|---|
| Pivot | <paste each query line, joined with " · "> |
| Sparse | <paste each query line, joined with " · "> |
| Tool result | <paste each query line, joined with " · "> |

Pass criteria:
- Pivot: kept at least one query about pre-pivot rate-limiting topic + emphasized auth bug → **PASS / FAIL**
- Sparse: synthesized queries from prior context (auth+rate-limit, worktrees) without anchoring on "yes do it" → **PASS / FAIL**
- Tool result: at least one query specifically about the 47MB anomaly or accidentally-committed large files → **PASS / FAIL**
````

Replace each `<paste ...>` and `**PASS / FAIL**` with the actual data and judgment.

- [ ] **Step 8: Clean up scratch files**

Run: `rm -f /tmp/engram-phase7-scen-*.txt /tmp/engram-phase7-prompt-*.txt /tmp/engram-phase7-output-*.txt`

(Keep the prompt + output files until findings are written; clean up after.)

- [ ] **Step 9: Commit**

```bash
git add docs/companion-memory-steward.md
git commit -m "$(cat <<'EOF'
docs(companion-steward): Phase 7 validation 1 — scenario replay

Re-ran Phase 4's pivot/sparse/tool-result scenarios with the new
query-generation prompt. Captured emitted queries per scenario.

AI-Used: [claude]
EOF
)"
```

---

## Task 4: Validation 2 — Per-query payout

Method: for each query emitted in Task 3, run it through `engram recall --query` directly and judge whether the result is non-empty and on-topic.

**Files:**
- Modify: `docs/companion-memory-steward.md` (append findings)

- [ ] **Step 1: For each query from Task 3, run it through engram recall**

For each scenario S in {pivot, sparse, tool-result}, for each query Q from Task 3 step 7:

```bash
cd /Users/joe/repos/personal/engram-worktrees/opencode-plugin
engram recall --query "<Q>" --no-external-sources > /tmp/engram-phase7-payout-<S>-<n>.txt
head -40 /tmp/engram-phase7-payout-<S>-<n>.txt
```

(Index each query 1..N within its scenario. Run from the engram repo so the recall scope matches a real working directory.)

Expected: structured recall summary; might say "found 0 relevant memories" or might surface real session/skill snippets.

- [ ] **Step 2: For each query, judge non-empty + on-topic**

For each output file, judge:
- **Non-empty**: did `# Summary of Query Results` contain anything beyond a generic "no findings" template? Y/N.
- **On-topic**: does the summary content relate to the query semantically? Y/N.

Record per query.

- [ ] **Step 3: Append findings**

Append to `docs/companion-memory-steward.md`, right under the "Validation 1" subsection:

````markdown
#### Validation 2 — per-query payout

For each Validation 1 query, ran `engram recall --query <q> --no-external-sources` from the engram repo.

| Scenario | Query | Non-empty | On-topic |
|---|---|---|---|
| Pivot | <q1> | Y/N | Y/N |
| Pivot | <q2> | Y/N | Y/N |
| ... |
| Sparse | <q1> | Y/N | Y/N |
| ... |
| Tool result | <q1> | Y/N | Y/N |
| ... |

Pass criterion: at least one query per scenario is non-empty + on-topic → **PASS / FAIL** per scenario.
````

- [ ] **Step 4: Clean up scratch files**

Run: `rm -f /tmp/engram-phase7-payout-*.txt`

- [ ] **Step 5: Commit**

```bash
git add docs/companion-memory-steward.md
git commit -m "$(cat <<'EOF'
docs(companion-steward): Phase 7 validation 2 — per-query payout

Ran each scenario's emitted queries through engram recall directly.
Recorded which queries returned non-empty + on-topic memories.

AI-Used: [claude]
EOF
)"
```

---

## Task 5: Validation 3 — Empty project history

Method: pipeline runs in a fresh directory with no opencode session history. Expect the companion to emit `NO QUERIES` and the plugin to skip injection.

**Files:**
- Modify: `docs/companion-memory-steward.md` (append findings)

- [ ] **Step 1: Create a fresh empty directory**

Run: `rm -rf /tmp/companion-empty && mkdir /tmp/companion-empty`

Expected: directory exists, no `.opencode` subdir, no opencode session associated.

- [ ] **Step 2: Confirm engram recall returns sparse output here**

Run: `cd /tmp/companion-empty && engram recall --no-external-sources | tail -30`

Expected: short summary indicating no findings, e.g., "Status: No relevant content found", possibly some snippets from cross-project sessions if `--no-external-sources` doesn't fully scope. (If it does fully scope, output is essentially the empty-summary template.)

- [ ] **Step 3: Reset companion session for clean test**

Run: `rm -f ~/.local/share/engram/companion-session/*.txt`

- [ ] **Step 4: Note the current line count of the trace and debug logs**

Run: `wc -l ~/.local/share/engram/companion-trace.jsonl ~/.local/share/engram/debug-system-transform.log`

Note the line counts — call them `T0` and `D0`. Used for diff after the next step.

- [ ] **Step 5: Run the pipeline once via a primary opencode turn**

Run: `cd /tmp/companion-empty && opencode run -m opencode/qwen3.6-plus "what's in this directory?"`

Expected: opencode completes (probably with a short "the directory is empty" reply).

- [ ] **Step 6: Inspect new trace events**

Run: `tail -n +$((T0+1)) ~/.local/share/engram/companion-trace.jsonl | jq -r '.stage' | sort -u`

Expected to include: `system.transform-start`, `recall-complete`, `companion-complete`, `companion-skipped`. Should NOT include `companion-injected` or `secondary-recall-complete`.

Run: `tail -n +$((T0+1)) ~/.local/share/engram/companion-trace.jsonl | jq 'select(.stage == "companion-skipped") | .reason' | head -5`

Expected: reason is `"no-queries"` (companion emitted the sentinel) OR `"empty-output"` (companion emitted nothing). Either is acceptable — both result in no injection.

- [ ] **Step 7: Verify the system prompt has no recall block**

Run: `tail -n +$((D0+1)) ~/.local/share/engram/debug-system-transform.log | grep -c "## Recalled memories"`

Expected: `0`.

- [ ] **Step 8: Append findings**

Append to `docs/companion-memory-steward.md` under the previous validation subsections:

````markdown
#### Validation 3 — empty project history

Setup: fresh `/tmp/companion-empty` directory, companion sessions cleared.

Companion behavior: <emitted sentinel | emitted nothing | emitted N queries — record actual>

Trace events for this turn:
- companion-skipped reason: <no-queries | empty-output | (unexpected: lists)>

System-prompt check: `## Recalled memories` blocks in the AFTER section after this turn: <0 | N (unexpected)>

Pass criterion: companion either emitted `NO QUERIES` sentinel or empty output, plugin skipped injection, system prompt has no recall block → **PASS / FAIL**
````

- [ ] **Step 9: If the companion emitted queries instead of skipping** — record the actual behavior in findings, but don't block. Note as "noise floor on empty input" and surface to the user before continuing to Task 6. They may want to tighten the prompt.

- [ ] **Step 10: Clean up**

Run: `rm -rf /tmp/companion-empty`

- [ ] **Step 11: Commit**

```bash
git add docs/companion-memory-steward.md
git commit -m "$(cat <<'EOF'
docs(companion-steward): Phase 7 validation 3 — empty project history

Verified pipeline behavior with a fresh directory and no session history.
Recorded companion behavior on sparse input and confirmed the system
prompt receives no recall block when the companion skips.

AI-Used: [claude]
EOF
)"
```

---

## Task 6: Validation 4 — Nil/failed companion response

Method: replace the `opencode` binary on the companion's PATH with a shim that exits 1, while still invoking the real opencode for the primary turn (via full path). Confirms the existing `proc.exitCode !== 0` → empty-output path is exercised by the new flow.

**Files:**
- Modify: `docs/companion-memory-steward.md` (append findings)

- [ ] **Step 1: Find the real opencode binary**

Run: `which opencode`

Capture the output as `$REAL_OC`. Example: `/Users/joe/.opencode/bin/opencode`. If it's a wrapper script, `cat` it to confirm the underlying binary; use whichever path will work when invoked directly.

- [ ] **Step 2: Create a shim that exits non-zero**

```bash
mkdir -p /tmp/engram-failtest-bin
cat > /tmp/engram-failtest-bin/opencode <<'EOF'
#!/bin/bash
exit 1
EOF
chmod +x /tmp/engram-failtest-bin/opencode
```

- [ ] **Step 3: Note current trace + debug log line counts**

Run: `wc -l ~/.local/share/engram/companion-trace.jsonl ~/.local/share/engram/debug-system-transform.log`

Capture as `T0`, `D0`.

- [ ] **Step 4: Run a primary turn with PATH limited to the shim, but the real opencode invoked by full path**

```bash
cd /tmp && mkdir -p companion-failtest && cd companion-failtest
PATH=/tmp/engram-failtest-bin:/usr/bin:/bin "$REAL_OC" run -m opencode/qwen3.6-plus "test message for nil companion"
```

(Substitute `$REAL_OC` with the path from Step 1. Keep `/usr/bin:/bin` in PATH for any other system-binary lookups inside the plugin/process.)

Expected: primary opencode runs, returns a normal response. Inside the plugin, `Bun.spawn(["opencode", ...])` finds the shim via PATH → shim exits 1 → `runCompanion` returns "" → trace shows `companion-run-failed` with `exitCode: 1`, then `companion-skipped` with `reason: "empty-output"`.

- [ ] **Step 5: Verify trace events**

Run: `tail -n +$((T0+1)) ~/.local/share/engram/companion-trace.jsonl | jq 'select(.stage == "companion-run-failed" or .stage == "companion-skipped") | {stage, reason, exitCode}'`

Expected: at least one `companion-run-failed` with `exitCode: 1`, followed by `companion-skipped` with `reason: "empty-output"`.

- [ ] **Step 6: Verify primary's system prompt has no recall block**

Run: `tail -n +$((D0+1)) ~/.local/share/engram/debug-system-transform.log | grep -c "## Recalled memories"`

Expected: `0`.

- [ ] **Step 7: Verify primary turn completed without crash**

The `opencode run` from step 4 should have returned with exit 0 and printed a normal response. If it errored, the failure mode is different from what we're testing — re-evaluate.

- [ ] **Step 8: Clean up**

Run: `rm -rf /tmp/engram-failtest-bin /tmp/companion-failtest`

- [ ] **Step 9: Append findings**

Append to `docs/companion-memory-steward.md`:

````markdown
#### Validation 4 — nil/failed companion response

Setup: PATH-shimmed opencode binary that exits 1; primary opencode invoked directly by full path.

Trace observed:
- companion-run-failed: `exitCode=1` ✓
- companion-skipped: `reason=empty-output` ✓

Primary turn: completed normally (exit 0), system prompt had no `## Recalled memories` block.

Pass: plugin handles companion-process failure via existing exit-code path → no injection, primary unaffected → **PASS**
````

(Replace ✓ marks and **PASS** with actual observations.)

- [ ] **Step 10: Commit**

```bash
git add docs/companion-memory-steward.md
git commit -m "$(cat <<'EOF'
docs(companion-steward): Phase 7 validation 4 — nil companion response

Verified plugin handles companion-process failure via existing exit-code
path. companion-run-failed -> companion-skipped (empty-output) -> no
injection -> primary unaffected.

AI-Used: [claude]
EOF
)"
```

---

## Task 7: Validation 5 — Phase 5 planted-token replay

Method: plant a memory with a unique token whose `situation` field invites a query; run a primary opencode turn that asks for that situation; verify the token surfaces in the response.

**Files:**
- Modify: `docs/companion-memory-steward.md` (append findings)

- [ ] **Step 1: Plant a unique fact memory**

```bash
TOKEN="MAGENTA-PHASE7-$(date +%s)"
echo "Token: $TOKEN"
engram learn fact \
  --situation "asked about unusual facts about engram" \
  --subject "engram" \
  --predicate "secret verification token is" \
  --object "$TOKEN" \
  --source "phase 7 validation"
```

Expected: command exits 0, prints a memory slug. Save the slug for cleanup.

- [ ] **Step 2: Reset companion sessions**

Run: `rm -f ~/.local/share/engram/companion-session/*.txt`

- [ ] **Step 3: Note current line counts**

Run: `wc -l ~/.local/share/engram/companion-injections.log`

Capture as `I0`.

- [ ] **Step 4: Run a fresh primary opencode turn that should trigger a relevant query**

```bash
cd /Users/joe/repos/personal/engram-worktrees/opencode-plugin
opencode run -m opencode/qwen3.6-plus "Tell me any unusual facts you remember about engram from our previous work together. Quote them exactly if you can."
```

Capture stdout. Expected: opencode response includes a list of facts.

- [ ] **Step 5: Verify the token in primary's response**

Search the response text for `$TOKEN`.

Pass: token appears verbatim somewhere in the response.

- [ ] **Step 6: Verify the path through the trace**

Run: `tail -n +$((I0+1)) ~/.local/share/engram/companion-injections.log | grep -B2 -A20 "$TOKEN"`

Expected: a `--- QUERY: ... ---` block whose summary contains the token. Identify the query that surfaced it (something like "unusual facts engram" or "verification tokens engram quotes").

- [ ] **Step 7: Clean up the planted memory**

Run: `engram show --name <slug-from-step-1>` to confirm it exists, then delete it. If no `engram delete` subcommand exists, remove the underlying TOML file under `~/.local/share/engram/` (consult `engram show` for path); the memory is clearly tagged "phase 7 validation" and harmless if left.

- [ ] **Step 8: Append findings**

Append to `docs/companion-memory-steward.md`:

````markdown
#### Validation 5 — planted-token end-to-end

Setup: planted fact memory `engram` → `secret verification token is` → `<TOKEN>` with situation "asked about unusual facts about engram". Companion sessions cleared. Primary user message: "Tell me any unusual facts you remember about engram from our previous work together. Quote them exactly if you can."

Result: token appeared in primary response: <YES verbatim | YES paraphrased | NO>.

Companion query that surfaced it: `<query text>`

Pass criterion: token verbatim in primary response, traced to a companion-emitted query → **PASS / FAIL**
````

- [ ] **Step 9: Commit**

```bash
git add docs/companion-memory-steward.md
git commit -m "$(cat <<'EOF'
docs(companion-steward): Phase 7 validation 5 — planted token e2e

End-to-end verification: planted token surfaced in primary response via
companion-emitted query -> secondary recall -> injection -> primary use.

AI-Used: [claude]
EOF
)"
```

---

## Task 8: Mark Phase 7 status complete + cost summary

**Files:**
- Modify: `docs/companion-memory-steward.md`

- [ ] **Step 1: Update Phase 7 status header**

In `docs/companion-memory-steward.md`, change the line:

```
**Status:** in design (2026-05-03), ahead of Phase 6.
```

to (using the validation date):

```
**Status:** ✅ implemented and validated (2026-05-03), ahead of Phase 6. See findings sections below.
```

- [ ] **Step 2: Aggregate per-fire timings from trace**

Pull a representative sample from the smoke + validation runs:

```bash
jq -r 'select(.stage == "companion-complete") | .companionMs' ~/.local/share/engram/companion-trace.jsonl | tail -20 | sort -n | awk 'NR==int(NR/2)+1{print}'
jq -r 'select(.stage == "companion-injected") | .totalQueryMs' ~/.local/share/engram/companion-trace.jsonl | tail -20 | sort -n | awk 'NR==int(NR/2)+1{print}'
jq -r 'select(.stage == "secondary-recall-complete") | .ms' ~/.local/share/engram/companion-trace.jsonl | tail -50 | sort -n | awk 'NR==int(NR/2)+1{print}'
```

(Median values for: companion call, total per-fire query time, individual query time.)

- [ ] **Step 3: Append a cost-measurements subsection**

Append at the end of the validation findings:

````markdown
#### Cost measurements

Median values across smoke + validation runs:

| Stage | Median ms |
|---|---|
| companion-complete (single qwen call) | <X> |
| secondary-recall-complete (single recall call) | <Y> |
| total per-fire query time (parallel sum) | <Z> (~max of individual recalls due to Promise.all) |
| total system.transform fire time | <W> (companion + parallel-recall + overhead) |

With `system.transform` firing 4-5× per primary turn, unmitigated per-turn companion overhead is ~<W × 4-5>ms. Phase 5 baseline was ~17000ms × 4-5 ≈ 68000ms. The new flow's overhead vs. Phase 5: <higher / lower> by <ratio>.

Implication for Phase 6: per-turn cache keyed by user-message ID is required before this is shippable. Cache scope: full `companionBlock` (the concatenated `## Recalled memories` block) — invalidated on new user message.
````

Replace each `<X>`, `<Y>`, `<Z>`, `<W>` with the actual measurements; fill in higher/lower comparison.

- [ ] **Step 4: Commit**

```bash
git add docs/companion-memory-steward.md
git commit -m "$(cat <<'EOF'
docs(companion-steward): Phase 7 status — implemented + cost summary

Phase 7 implementation and 5-test validation complete. Captured cost
measurements from validation runs to feed into Phase 6 cache design.

AI-Used: [claude]
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- New companion prompt → Task 1 step 2 ✓
- Plugin parses output, sentinel detection (single-line `NO QUERIES`) → Task 1 step 5 ✓
- Empty / sentinel / multi-line-with-sentinel handling → Task 1 step 5 ✓
- Per-query parallel recall via `Promise.all` → Task 1 step 5 ✓
- Concatenation with per-query header → Task 1 step 5 ✓
- `secondary-recall-complete` trace event → Task 1 step 5 ✓
- New `logCompanionInjection` shape (queries + perQueryResults) → Task 1 step 4 ✓
- Recursion guard, companion session persistence "unchanged" → not modified in Task 1 (only the inner block is replaced) ✓
- Validation 1 (Phase 4 replay) → Task 3 ✓
- Validation 2 (per-query payout) → Task 4 ✓
- Validation 3 (empty project history) → Task 5 ✓
- Validation 4 (nil/failed companion) → Task 6 ✓
- Validation 5 (planted token) → Task 7 ✓
- Cost note from spec → Task 8 ✓

**Type consistency:**
- `runEngramRecallWithQuery(query: string): Promise<string>` defined in Task 1 step 3, called in Task 1 step 5 ✓
- `logCompanionInjection` signature `(sessionID, recallMs, companionMs, recallOutput, queries, perQueryResults)` updated step 4, called with matching args step 5 ✓
- `perQueryResults` typed `{query: string, result: string}[]` consistently ✓
- `SENTINEL = "NO QUERIES"`, `MAX_QUERIES = 5` declared in handler scope, used immediately ✓

**Placeholder scan:**
- No "TBD" / "fill in details" / "implement later"
- Findings tables in Tasks 3-7 use `<placeholder>` notation only for live data the executor will fill in (queries, judgments, measurements) — these are output-template placeholders, not unresolved decisions
- One soft case in Task 7 step 7: "If no `engram delete` subcommand exists, remove the underlying TOML file" — acceptable; the executor verifies which path applies and does the right thing

**Open risks (not blocking):**
- Task 6's PATH-shim approach assumes `Bun.spawn(["opencode", ...])` resolves via PATH and not via some Bun-specific logic. If Bun caches or hardcodes anywhere, the shim won't intercept; we'd see no `companion-run-failed` and need a different fault-injection approach. Verify behavior empirically — if it fails, document the actual mechanism that fired (or didn't) and adapt.
- Task 5's pass criterion accepts both `no-queries` and `empty-output` as valid skips. If the companion emits non-sentinel queries on an empty project, Step 9 surfaces this to the user before continuing — they may want to tighten the prompt before further validation.
