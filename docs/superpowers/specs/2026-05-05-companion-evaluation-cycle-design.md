# Companion Evaluation Cycle

## Problem

Engram's evaluation pipeline is hard-wired to the Anthropic Haiku API:

- `internal/anthropic/anthropic.go` is the only LLM backend.
- `internal/recall/orchestrate.go` calls Haiku for memory ranking, per-source extraction, and final summarization.
- `internal/cli/learn.go` calls Haiku for duplicate/contradiction detection at write time.

This bakes in two constraints we want to lift:

1. **Anthropic API key required.** OpenCode users (and any host without an Anthropic relationship) cannot use engram's intelligence layer.
2. **Plugin-side LLM orchestration in TypeScript.** Phase 7 of the OpenCode-plugin work introduced a "companion" — a sidecar OpenCode session that emits recall queries — and the plugin runs each query through `engram recall --query`, which still hits Haiku internally. Two LLM tiers, two languages, two sets of prompts.

We also want a richer per-turn loop than today's "emit recall queries" companion: every turn, the system should evaluate whether something has been learned and whether new memories should be injected, then act accordingly.

This spec consolidates both concerns: engram becomes the orchestrator, the LLM backend becomes a pluggable command, Anthropic is removed entirely, and a new top-level command (`engram cycle`) drives a per-turn learn-and-recall loop on behalf of any host that calls it.

## Design

### High-level shape

```
host (e.g. opencode plugin) — system.transform fires
        │
        ▼
host invokes:
  engram cycle --llm-cmd "<cmd>" --project-dir <abs-path>
        │
        ▼
engram orchestrates:
  1. Read transcript window for the project.
  2. LLM Call A — extract any learnings from the transcript.
  3. Persist each surviving learning (dedup, auto-rename slug).
  4. LLM Call B — propose recall queries (or none).
  5. For each query: run the recall pipeline (which itself uses --llm-cmd).
  6. Return JSON: { learned: [...], recalled: [...] }.
        │
        ▼
host reads JSON, formats `recalled` reports into the next system prompt.
```

The host (plugin or skill) does no LLM orchestration of its own. Engram owns the prompts, the parsing, the dedup, the ranking, and the synthesis.

### New command: `engram cycle`

```
engram cycle --llm-cmd <cmd> --project-dir <abs-path> [--transcript-budget <bytes>]
```

Flags:

- `--llm-cmd` (required, or fall back to `ENGRAM_LLM_CMD`): shell command that invokes an LLM. Contract below.
- `--project-dir` (required): absolute path to the host's project working directory. Engram passes it to the existing session-finder logic; reuses today's transcript discovery (no new directory mapping).
- `--transcript-budget` (optional, default `15360` = 15KB): max bytes of transcript content sent to LLM Calls A and B. Reuses `DefaultModeABudget`.

Output (stdout): a single JSON object.

```json
{
  "learned": [
    {
      "type": "feedback",
      "name": "skill-edits-require-tdd",
      "situation": "editing a SKILL.md file",
      "behavior": "Wrote skill changes without running pressure tests",
      "impact": "Behavioral regression slipped through",
      "action": "Always invoke superpowers:writing-skills first"
    }
  ],
  "recalled": [
    {
      "query": "engram cycle command JSON contract",
      "report": "Project memories show the cycle command was added in this design …\n\nKeep in mind: …"
    }
  ]
}
```

`learned` is an array of memory objects for memories actually persisted this cycle. Each object contains all `MemoryRecord` fields plus a `name` field derived from the on-disk path (basename minus `.toml`, post-auto-increment if a slug collision occurred). True duplicates are silently absent.

`recalled` is an array of `{query, report}` objects, one per query that produced a non-empty result. `report` is the LLM-synthesized prose described under "Recall pipeline changes" below.

Both arrays may be empty. Empty `learned` and empty `recalled` mean "this cycle had nothing to do" — host treats as no-op.

### Cycle internal flow

1. **Load transcript window.** Reuse `internal/recall.Reader` + finder, scoped to `--project-dir`. Cap at `--transcript-budget`. No memory listing yet.

2. **LLM Call A — learning extraction.**

   Prompt:

   > You are reviewing a project session transcript to identify learnings worth preserving.
   >
   > Examine the transcript and propose any new learnings: corrections you observe, completed work that taught a lesson, decisions made, or facts established.
   >
   > Output a JSON array of objects, each with:
   > - `type`: `"feedback"` or `"fact"`
   > - `situation`: short context phrase identifying when this applies
   > - For feedback: `behavior`, `impact`, `action`
   > - For fact: `subject`, `predicate`, `object`
   >
   > Return `[]` if there is nothing learnable.
   >
   > Transcript:
   > <transcript>

   Engram parses the JSON. For each candidate, calls `writeMemory` (see "Learn pipeline changes" below). The result of each `writeMemory` is appended to `learned[]` if it actually wrote.

3. **LLM Call B — query proposal.**

   Prompt (close to today's Phase 7 prompt, integrated into engram):

   > You are reviewing a project session transcript to decide if memories should be recalled.
   >
   > If the project is starting new research, taking new action, shifting approach, or otherwise embarking on something where prior memories could help, propose 1-5 targeted recall queries. Each query is 5-15 words capturing a specific facet to recall about.
   >
   > Output one query per line, no numbering, no commentary.
   >
   > If nothing in the transcript warrants recall, output exactly `NO QUERIES`.
   >
   > Transcript:
   > <transcript>

   Engram parses lines. `NO QUERIES`, empty output, or zero parsed queries → empty `recalled` array. Otherwise, up to 5 queries.

4. **Per-query recall.** For each query, invoke the recall pipeline (next section). Collect non-empty reports as `{query, report}` entries.

5. **Emit JSON** to stdout.

### Recall pipeline changes

**`engram recall` always synthesizes a prose report**, in both bare mode and query mode.

The query-mode pipeline retains its existing phase shape but replaces the final output:

| Phase | Today | After |
|-------|-------|-------|
| Phase 1: Memory rank | LLM picks names from index | Same, via `--llm-cmd` |
| Phase 2: Auto memory extract | LLM extracts snippets | Same, via `--llm-cmd` |
| Phase 3: Per-session extract | LLM extracts snippets | Same, via `--llm-cmd` |
| Phase 4: Skill extract | LLM extracts snippets | Same, via `--llm-cmd` |
| Phase 5: CLAUDE.md extract | LLM extracts snippets | Same, via `--llm-cmd` |
| Phase 6: Synthesis | `SummarizeFindings` (query-only) | New synthesis prompt, via `--llm-cmd` |

**Bare mode (no `--query`)** was previously a raw concatenation of recent transcripts plus time-window-matched memories. After this change, bare mode collects the same inputs (time-window-matched memories + recent transcripts under budget) and feeds them to the new Phase 6 synthesis prompt. No per-source extraction (Phases 2, 4, 5) runs in bare mode — those are query-anchored. The output of `engram recall` (with or without `--query`) is always the prose report on stdout.

**Phase 6 synthesis prompt:**

> You are synthesizing engram memory sources into a coherent report for an AI agent.
>
> The sources include facts, behavioral feedback, action records, and outcomes drawn from prior project work. Weave them into a narrative that captures what has been learned and tried, then end with explicit advice on what the reader should keep in mind moving forward.
>
> [If query: "Focus on material relevant to: <query>"]
>
> Sources:
> <buffer>
>
> Output the report only — no preamble, no list of sources, no JSON.

The `Result` struct collapses:

```go
type Result struct {
    Report string `json:"report"`
}
```

`FormatResult` writes the report verbatim. `Result.Memories` is removed; structured memories are no longer carried at this layer (the synthesis weaves them into prose, and the cycle command builds its own structured `learned`/`recalled` from the writeMemory and recall paths).

### Learn pipeline changes

**Three changes to `internal/cli/learn.go`:**

1. **Broaden the dedup index.** `memory.BuildIndex` (and the dedup prompt) include the full content fields (behavior/impact/action for feedback; subject/predicate/object for fact), not just type+name+situation. The detector's prompt is updated to explicitly compare full content.

2. **Drop CONTRADICTION.** Detector returns only `DUPLICATE: <name>` or `NONE`. Contradicting memories coexist — recall surfaces both, the consuming LLM reconciles.

3. **Slug-collision auto-increment.** When `writeMemory` decides to write (no DUPLICATE returned by the detector), it computes `slug := Slugify(situation)`, then:

   - If `<dataDir>/<type>/<slug>.toml` does not exist → write there.
   - If it exists, scan for the smallest unused `<slug>-N` (N starting at 1) and write there.

   No `--force` flag. No silent overwrite. The detector + auto-increment together guarantee: identical content is dropped, distinct content always lands at a fresh path.

   Race window (two concurrent writes choosing the same suffix) is not addressed in this design — `engram learn` is invoked from a single host hook serially per primary session, and we accept the tiny race risk for the simplicity. A one-line fix (`O_EXCL` on create) is a follow-up if it ever bites.

The dedup detector itself uses `--llm-cmd` like everything else (see next section).

### LLM backend: `--llm-cmd` contract

A single command-line abstraction replaces the Anthropic client.

**Invocation contract:**

- Engram spawns the command via the user's shell with the prompt streamed to **stdin**.
- The command must emit the LLM's response on **stdout**.
- Exit code 0 = success. Non-zero = failure.
- stderr is captured and logged to engram's status writer (stderr) for debugging.
- A 60s wall-clock timeout is enforced (configurable later if needed). On timeout, engram kills the process and treats the call as a failure.

**Why stdin/stdout:** keeps the command a pure filter. No argument-template gymnastics, no prompt-quoting concerns, no length limits from argv.

**Why no model parameter:** the command itself encodes the model. Different invocations of cycle can use different models by configuring different commands.

**Failure mode:** on any failure (non-zero exit, timeout, empty stdout), engram treats the call as if it returned nothing actionable. Specifically:

- LLM Call A failure → no learnings, empty `learned[]`.
- LLM Call B failure → no queries, empty `recalled[]`.
- Recall Phase 1 (rank) failure → no memories matched, continue to phase 2.
- Recall Phase 2-5 (extract) failures → skip that source, continue.
- Recall Phase 6 (synthesis) failure → empty `report` for that query (cycle drops the entry).
- Dedup failure during writeMemory → write the memory anyway (consistent with today's behavior at `internal/cli/learn.go:99`).

A trace of the failure is logged to engram's stderr in every case so hosts/users can debug without `--llm-cmd` becoming a black hole.

**Default plumbing:** `ENGRAM_LLM_CMD` environment variable provides the default. `--llm-cmd` overrides. If neither is set:

- `engram cycle`: returns `{"learned": [], "recalled": []}`, logs a notice to stderr.
- `engram recall`: returns an error. Synthesis cannot run without an LLM.
- `engram learn`: writes the memory (skips dedup).

The plugin sets `ENGRAM_LLM_CMD` in the OpenCode session environment when the plugin loads, so skill bash blocks (e.g. `/prepare`) inherit the same configuration.

### Anthropic removal

`internal/anthropic/` is deleted. Its callers are rewired:

- `internal/recall/summarize.go` — replace `anthropic.Client` with a new `internal/llmcmd` package that implements the same `Extractor` / `FindingSummarizer` interfaces by spawning `--llm-cmd`.
- `internal/cli/learn.go` — remove `makeAnthropicCaller`, `resolveToken`. Replace with the `llmcmd` package.
- `internal/cli/cli.go` — drop Anthropic wiring from command setup. Add `--llm-cmd` flag plumbing and `ENGRAM_LLM_CMD` env-var resolution.

The `Extractor` and `FindingSummarizer` interfaces in `internal/recall/orchestrate.go` are unchanged — the swap is purely at the implementation layer.

### Plugin changes

`opencode/plugins/engram.ts` simplifies dramatically. The current `system.transform` hook becomes:

```typescript
"experimental.chat.system.transform": async (input, output) => {
  const before = output.system[0]
  const reminder = await getReminder("system")
  const sessionID = input?.sessionID
  const projectDir = input?.directory ?? process.cwd()

  if (process.env.ENGRAM_COMPANION_MODE === "1") {
    output.system[0] = before + reminder
    return
  }

  const llmCmd = "opencode-companion-shim"  // see below
  const cycleResult = await runEngramCycle(projectDir, llmCmd)

  const block = formatCycleResult(cycleResult)
  output.system[0] = before + reminder + (block ? "\n\n" + block : "")
}
```

`runEngramCycle` shells out to `engram cycle --llm-cmd <cmd> --project-dir <projectDir>`, parses JSON, returns `{learned, recalled}`. `formatCycleResult` renders the `recalled[]` array as the `## Recalled memories` block (one report per query, headed by the query text).

**Companion shim:** because `opencode run --format json` emits NDJSON streaming events rather than plain text, the plugin ships a tiny shim script (`opencode/bin/engram-llm-shim`) that:

1. Reads the prompt from stdin.
2. Runs `opencode run -m opencode/qwen3.6-plus --format json` with the prompt as the message argument.
3. Parses NDJSON, concatenates `text` events, emits the result on stdout.
4. Exits 0 on success, non-zero on subprocess failure.

The shim is the only place that knows about opencode's wire format. Engram remains opencode-agnostic. The plugin sets `ENGRAM_LLM_CMD=<absolute path to shim>` in the session environment so skill bash blocks reach the same backend.

The recursion guard (`ENGRAM_COMPANION_MODE=1`) is still set on the companion's environment by the shim, so when the companion's own opencode process loads the plugin, the plugin's `system.transform` returns the reminder only and does not re-enter cycle.

### Files modified

**New:**

- `internal/cli/cycle.go` — cycle command entry point + flag parsing.
- `internal/cycle/` (new package) — orchestration, prompt templates, JSON output.
- `internal/llmcmd/` (new package) — `--llm-cmd` execution, stdin/stdout protocol, timeout, failure handling. Implements `Extractor`, `FindingSummarizer`, and the dedup `llmCaller` signature.
- `opencode/bin/engram-llm-shim` — shim that bridges opencode's NDJSON to plain stdin/stdout.

**Modified:**

- `internal/recall/orchestrate.go` — `Result` collapses to `{Report string}`; bare mode runs the synthesis prompt.
- `internal/recall/summarize.go` — `SummarizeFindings` prompt updated to the new synthesis prompt; backend swap to `llmcmd`.
- `internal/cli/learn.go` — broadened dedup index, drop CONTRADICTION, slug auto-increment, backend swap.
- `internal/cli/cli.go` — register `cycle`, `--llm-cmd` flag, env-var fallback, drop Anthropic wiring.
- `internal/memory/memory.go` — `BuildIndex` includes content fields.
- `opencode/plugins/engram.ts` — replace inline companion logic with a single `engram cycle` call + JSON formatter.

**Deleted:**

- `internal/anthropic/anthropic.go`
- `internal/anthropic/anthropic_test.go`

Tests for every modified package are updated; new packages get their own tests using imptest mocks for the LLM and filesystem boundaries.

### Out of scope

- **Cost optimization.** Today's plugin fires `system.transform` 4-5x per primary turn; each fire now drives a full cycle (2 + N×6 LLM calls). At ~7s cold start per `--llm-cmd` invocation, this is materially expensive. The user has flagged this as a deliberate later-pass concern. Caching the cycle result per primary turn (keyed by the latest user message ID) is the obvious follow-up.
- **Concurrent slug-collision races.** Single-host serial invocation only; no `O_EXCL` or lockfile.
- **MCP wrapper around engram.** Discussed and intentionally deferred. Strategic value (cross-host portability) is independent of this redesign.
- **Companion session reuse.** Each `--llm-cmd` invocation is independent. Future optimization could share an opencode session across the cycle's internal LLM calls.
- **`--force` flag on `learn`.** Not needed under the new write logic. If a use case appears, it can be added.

## Validation plan

1. **End-to-end planted-token replay (Phase 5/7 harness style):** plant a fact memory with a unique token, run `engram cycle` on a synthetic transcript that asks about the topic. Verify the token surfaces in a `recalled[].report`.

2. **Empty-transcript no-op:** call cycle on a fresh project directory with no transcripts. Verify `{"learned": [], "recalled": []}` and zero LLM calls beyond Calls A and B (or Call B only if A correctly emits empty).

3. **Dedup correctness:** plant a feedback memory; run cycle on a transcript that would extract the same content. Verify `learned[]` is empty (true duplicate skipped).

4. **Slug auto-increment:** plant `feedback/foo.toml`; run cycle that extracts a feedback memory whose slug also resolves to `foo` but with different content. Verify a `feedback/foo-1.toml` exists after the run and shows up in `learned[]`.

5. **`--llm-cmd` failure paths:** point `--llm-cmd` at a script that exits 1, then a script that times out, then a script that emits empty stdout. Verify cycle returns the expected empty/skip JSON in each case and logs the failure to stderr.

6. **Plugin integration:** with the shim wired into the engram plugin, run a multi-turn opencode session. Verify the plugin's `system.transform` produces the expected `## Recalled memories` block from cycle's `recalled[]` and that `learned[]` memories appear under `~/.local/share/engram/memory/`.
