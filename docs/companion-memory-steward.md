# Companion Memory Steward — Design & Findings Log

> A live working log. Each section is a finding from incremental experiments. Chronological. Update as we learn.

## Problem

Non-Claude models in OpenCode (gpt-mini, qwen, etc.) don't reliably engage with engram's memory operations (`/prepare`, `/learn`) when nudged via system prompt or hook-injected reminders, regardless of phrasing tone (calm, urgent, imperative, behavioral-cycle framing — all tried, none stuck).

Engram also currently uses Haiku internally for memory classification, which assumes Anthropic API availability — not a given for OpenCode users.

## Hypothesis

Move memory engagement off the primary model entirely. A separate **companion** (its own persistent OpenCode session, runs as a sidecar to each primary session) inspects the primary's transcript and decides:

- Could the latest user message be considered the start of a new task? → call `engram_recall`, return memories to inject.
- Could the latest assistant turn be considered task completion? → call `engram_learn_feedback` / `engram_learn_fact`.

The plugin's hook injects the companion's output into the primary's next system prompt. The primary never has to remember to call /prepare or /learn — the companion does it.

## Architecture (target)

```
                         primary OpenCode session
                                   │
                                   ▼
                    experimental.chat.system.transform
                                   │
                ┌──────────────────┼──────────────────┐
                ▼                                     ▼
   read companion sidecar file              spawn companion
   inject memories into prompt              (sync, then async later)
                                                     │
                                                     ▼
                                       opencode run -s <companionID>
                                       "look at primary transcript,
                                        recall or learn as needed,
                                        emit memories to inject"
                                                     │
                                                     ▼
                                       writes sidecar file for next turn
```

Sync first (simpler to debug). Move to async after the loop is proven.

## Open Questions

| # | Question | Default | Resolved? |
|---|---|---|---|
| Q1 | Can OpenCode resume a session via CLI with a chosen model? | yes — `opencode run -s <id> -m provider/model` | being validated, Phase 1 |
| Q2 | How does the companion access the primary's transcript? | `opencode export <primaryID>` | not yet validated |
| Q3 | Where does the companion write its output? | `~/.local/share/engram/companion/<primarySessionID>.md` | TBD |
| Q4 | Companion model? | configurable; default to whatever primary uses; option to pin | TBD |
| Q5 | Trigger cadence? | every turn (sync) at first | TBD |
| Q6 | Companion's tool surface? | engram_* tools only; no file/Bash | TBD |
| Q7 | Where does the companion live as code? | external steward process invoked from plugin hook | TBD |

## Phase 1 — Validate OpenCode CLI session persistence + model selection

**Goal:** confirm a multi-turn conversation can be held with OpenCode via `opencode run`, with state preserved across invocations and a chosen model (qwen).

**Test:** three-turn conversation. Turn 1 plants a fact, turn 2 is unrelated, turn 3 asks the model to recall the fact from turn 1. If turn 3 recalls correctly, session persistence works.

**Status:** ✅ complete (2026-05-03).

### Findings

**Q1 resolved: yes, OpenCode supports CLI session persistence + model selection.**

Three-turn experiment with `opencode/qwen3.6-plus`:

| Turn | Command | Response | Cost | Cache |
|---|---|---|---|---|
| 1 | `opencode run -m opencode/qwen3.6-plus --format json "Remember this number: 4271..."` | `Got it.` | $0.0143 | write 22772 |
| 2 | `opencode run -s ses_210d78933ffe9icIPeSLkfKkfc -m opencode/qwen3.6-plus --format json "What's the capital of France?"` | `Paris` | $0.0012 | read 22772, write 20 |
| 3 | `opencode run -s ses_210d78933ffe9icIPeSLkfKkfc -m opencode/qwen3.6-plus --format json "What number did I tell you to remember?"` | `4271` ✅ | $0.0013 | read 22792, write 20 |

Total: ~$0.017 for the 3-turn validation.

Mechanics learned:

- **Session ID extraction**: `--format json` emits NDJSON events; the first `step_start` event contains `sessionID` at the top level. Easy to capture: `opencode run ... --format json | jq -r 'select(.type == "step_start") | .sessionID' | head -1`.
- **Resume**: `-s <sessionID>` resumes; `-m provider/model` re-asserts the model on each invocation (otherwise it falls back to the agent's default).
- **Cache hits across invocations** — subsequent turns re-use the cached system prompt across `opencode run` invocations (cache.read = 22772 on turn 2). This is huge: a long-lived companion conversation amortizes the system-prompt cost.
- **Model spec**: the user's config exposes `opencode/qwen3.5-plus` and `opencode/qwen3.6-plus`. Latest works.
- **Plugin overhead**: each turn fires the engram plugin (system.transform appended a 22KB cached system prompt). Plugin sees the companion sessions just like any other session — no special handling.

**Q2 resolved: yes, `opencode export <sessionID>` works.** Output is JSON with `info` (metadata: id, projectID, directory, title, time) + `messages[]` (each with role, agent, model, parts). 11KB for the 3-turn session — small enough to feed straight into a companion prompt without summarization for short conversations. For long primary sessions we'll need to truncate / summarize.

### Implications for the architecture

- Spawning the companion is straightforward: `opencode run -s <companionSessionID> -m <companionModel> --format json "<prompt with primary transcript>"`.
- The companion's session ID is stable across invocations and can be persisted in the sidecar file (`~/.local/share/engram/companion/<primarySessionID>.json` mapping primary → companion + last output).
- The primary's transcript can be passed into the companion either by paste-into-prompt (small sessions) or via file path (the companion uses a `read_file` tool — but that adds tool surface; see Q6).
- Cache reuse across `opencode run` invocations means the recurring system-prompt cost is paid once per companion session, not per turn. Latency and cost both stay bounded.

### Caveats / risks surfaced

- The companion will inherit the same engram plugin (system.transform) — meaning the companion sees engram-reminders too. Probably fine (it's still a reasoning task), but watch for recursion if the companion ever decides to call /prepare or /learn on itself. Guard: the companion's prompt should explicitly say "you are reviewing the *primary's* session, not your own."
- A companion session accumulates indefinitely if reused for many primary sessions. May want one companion session per primary session (1:1) and `opencode session delete` when the primary closes.

## Phase 2 — Hook trigger inventory: when can we call out, and can we tell sender?

**Goal:** find which OpenCode plugin hooks fire on which conversational events, and whether the input/output gives enough info to distinguish user vs assistant turns.

**Status:** ✅ complete (2026-05-03).

### Method

Added a `trace()` helper to `opencode/plugins/engram.ts` that JSON-dumps every hook invocation (input + output, with string fields truncated at 500 chars) to `~/.local/share/engram/hook-trace.jsonl`. Wired tracing into both existing hooks and the ones not previously used. Ran a 3-turn conversation that mixed plain replies with a tool call (`bash`) to trigger as many code paths as possible. Then `jq`'d the trace.

### Snag found and fixed first

**The plugin wasn't loading at all** in the cloned-repo install. `opencode run --print-logs --log-level DEBUG` revealed: `Cannot find module '@opencode-ai/plugin'`. The plugin's `package.json` declares the dep but no `npm install` had been run in the `opencode/` dir, so `node_modules/` didn't exist. Node module resolution traverses up from `opencode/plugins/engram.ts` looking for `node_modules`, didn't find one until somewhere irrelevant.

**Fix**: `cd opencode && npm install` (or `bun install`). The `node_modules/@opencode-ai/plugin` directory is required for the plugin to resolve its imports.

**Implication**: today's earlier qwen 3-turn validation (Phase 1) ran with the plugin **silently disabled**. The ~22KB cache writes we observed were OpenCode's base system prompt, not engram's injection. After `npm install`, cache writes grew by ~1KB (engram's System reminder).

**README change**: install instructions need `cd opencode && npm install` as a step. (Fixed in same commit.)

### Hook firing counts (3-turn conversation: plain → bash tool → plain)

| Hook | Calls | Notes |
|---|---|---|
| `event` | 122 | Every internal session/message/diff event. `event.type` distinguishes them; many are noisy (deltas, updates), but `session.idle` is gold (fires once per turn at end-of-response — 3 calls for 3 turns). |
| `experimental.chat.system.transform` | 5 | Once per LLM request. 5, not 3, because turn 2 had a tool call → model invoked twice (decide tool, then summarize result). Currently used for system-prompt injection. |
| `chat.params` | 5 | Paired 1:1 with system.transform. Lets us mutate temperature/topK/etc. before the LLM call. |
| `chat.headers` | 5 | Paired 1:1 with system.transform. Custom HTTP headers per request. |
| `experimental.chat.messages.transform` | 4 | Fires per LLM request (slightly different cadence than system.transform). Lets us mutate the message list. |
| `experimental.text.complete` | 3 | **Once per assistant text part** — input has `messageID`/`partID`, output is the assistant's actual text. |
| `chat.message` | 3 | **Once per USER message** (role=user confirmed empirically — see below). |
| `tool.execute.before` | 1 | Fires once for the bash call. |
| `tool.execute.after` | 1 | Fires once for the bash call. |

### Sender disambiguation: confirmed

The big surprise: **`chat.message` only fires for user messages**, not "all messages" as commit `0888479f` claimed when removing it. Empirical data — 3 user messages, 3 `chat.message` calls. The `output.message.role` field is `"user"` in every invocation, and `output.parts[0].text` contains the user's prompt verbatim.

Either OpenCode v1.14.30 changed behavior, or the original commit message was wrong. Either way: `chat.message` is now safe and useful as a "user just sent a message" signal — re-wirable for companion-trigger purposes.

For assistant-side: `experimental.text.complete` fires once per assistant text part, with the actual text in `output.text`. Combined with the `session.idle` event, we have two candidate signals for "turn finished, model is done."

### Implications for the architecture

| Companion responsibility | Best hook trigger | Why |
|---|---|---|
| Decide "could this be a new task?" → `engram_recall` | `chat.message` | Fires exactly once per user message, with full text in `output.parts[0].text`. Earliest possible point. |
| Decide "could this be the end of a task?" → `engram_learn_*` | `event` filtered to `event.type === "session.idle"` | Fires once at end-of-turn after model is fully done responding (including post-tool followups). Cleaner than `experimental.text.complete` which fires per-part and could fire mid-turn. |
| Inject recalled memories into next system prompt | `experimental.chat.system.transform` (already wired) | Fires per LLM request — a recall result written between turns is picked up here. |

The async companion model is straightforward to wire:

1. `chat.message` → spawn companion with the user message text and primary session ID; companion decides + writes recall result to sidecar file.
2. `experimental.chat.system.transform` → read sidecar; append to system prompt; clear it (so the recall isn't injected on every subsequent LLM call within the same turn).
3. `event` (session.idle) → spawn companion with primary transcript; companion decides + calls engram_learn if appropriate.

A note on the `event` hook: 122 invocations in a 3-turn conversation. Most are noisy (`message.part.delta` x41 — token-level streaming events). Filter aggressively: `if (event.type !== "session.idle") return`.

### Caveats

- `session.idle` fires after the model fully stops generating, including after tool followups within the same user turn. So it's a clean once-per-turn signal — but it doesn't distinguish "task done" from "task still in progress, just waiting for the next user input." That distinction is the companion's job (the LLM analyzing transcript content, not just timing).
- The trace file grew to 149 entries / 164KB for 3 short turns. We don't want this in production. Plan: strip the `trace()` helper after Phase 2; keep a thin `engram debug-trace` toggle behind an env var if we ever need it again.
- `chat.message` fired *before* any of the LLM-request-time hooks (system.transform, params, headers). So a synchronous companion call from `chat.message` will block the primary's turn. Expected for sync mode; document.

## Phase 3 — Validate sync engram callout + dynamic injection

**Goal:** confirm the plugin can synchronously call out to engram from `experimental.chat.system.transform`, fetch fresh state per invocation (not cached at plugin load), and inject it into the system prompt — and that the LLM picks it up.

**Status:** ✅ complete (2026-05-03).

### Method

Planted a fact memory `engram-favorite-color-test-memory` with `object=blue`. Temporarily modified `experimental.chat.system.transform` to also spawn `engram show --name engram-favorite-color-test-memory` and append its output to the system prompt. Ran `opencode run -m opencode/qwen3.6-plus "What is engram's favorite color? Answer with just the color name."` in 5 fresh sessions, updating the memory to a different color (`engram update --name ... --object <color>`) between each. Reverted the hook modification and deleted the test memory after.

### Findings

| Memory state | Model said | Elapsed |
|---|---|---|
| blue | "Blue" | 7316ms |
| red | "Red" | 6409ms |
| yellow | "Yellow" | 4609ms |
| magenta | "Magenta" | 4121ms |
| cyan | "cyan" | 3412ms |

5/5 correct. Each fresh `opencode run` read the **current** memory state (no plugin-load-time caching). The sync shell-out → injection → LLM consumption loop works end-to-end.

**Latency**: cold start ~7s, warmed ~3.4s. The decrease is OpenCode's system-prompt cache becoming warm across invocations of the same model. Per-turn budget for the companion's sync work: roughly **3–7 seconds without bursting OpenCode's tolerance**, with a downward trend as the session warms.

**Q3 resolved: yes, sync injection of fresh state works.** Confirms hook can `await` an arbitrarily-paced shell command and inject its output without breaking.

### Implications for the architecture

- The sync companion model is feasible. A `chat.message` hook can spawn `opencode run -s <companion>` synchronously, wait for the companion's recall result, and the next `system.transform` (which fires very shortly after) can inject the result. Total added latency per primary turn ≈ companion turn cost (~3–7s for qwen-class).
- The cold-start of ~7s is paid once per companion session; long-lived companion sessions (one per primary, persisted) amortize that. Subsequent turns are ~3–4s.
- We don't need anything fancier than `engram show` / `engram recall` returning text, and a string append. No streaming, no protocol — file IO + stdout.

## Phase 4 — Companion prompt design

**Goal:** find a prompt for the companion that reliably emits sensible targeted recall queries given a primary transcript snippet + recent project history. Validate across realistic conversational twists.

**Status:** ✅ v1 works (2026-05-03).

### Snag found and fixed first

The OpenCode SQLite recall hardcoded `role: "assistant"` for every text part — so transcripts read back with `ASSISTANT:` prefixed on user messages too. The role actually lives on `message.data.role`. Fixed `queryParts` to JOIN the `message` table and pass role through to `buildJSONLLine`. Companion now sees correct USER/ASSISTANT alternation.

### Method

Three fake sessions inserted directly into `~/.local/share/opencode/opencode.db`, each in its own `/tmp/companion-scen-*` directory, with proper user/assistant roles via the `message` table:

1. **Pivot**: user wants rate-limiting; mid-conversation pivots to "actually fix the auth bug first."
2. **Sparse**: user asks "should I create a worktree?", assistant explains, user says "yes do it."
3. **Tool result**: user asks for `ls`, assistant lists files including a 47MB `debug-trace.log` that shouldn't be there; user reacts to the anomaly.

For each scenario, ran `engram recall --no-external-sources` from the scenario's directory, captured the project history portion (USER/ASSISTANT transcript), prepended a companion-role prompt, and sent to `opencode run -m opencode/qwen3.6-plus`.

### v1 prompt

```
You are a memory steward observing a primary AI agent's project session.
Your job: read the recent conversation in the project history below, and
propose 2 to 4 targeted recall queries that would surface helpful past
memories about what is currently happening.

Output your queries as a numbered list, one per line, with no other text
or commentary. Each query should be 5 to 15 words capturing a specific
facet you would want to recall about. If nothing in the history seems
memory worthy, output exactly: no action

PROJECT HISTORY (most recent message at end):
<recall output>
```

### Findings — qwen3.6-plus output per scenario

| Scenario | Companion's queries |
|---|---|
| Pivot | (1) auth middleware 401 intermittent token validation bug · (2) token bucket rate limiting implementation patterns · (3) internal/auth/middleware.go previous modifications or issues · (4) API authentication token validation logic history |
| Sparse | (1) Previous auth module refactoring patterns and design decisions · (2) Rate limiting implementation history and architectural choices · (3) Git worktree setup preferences and past usage patterns · (4) Multi-file refactor approaches and lessons learned |
| Tool result | (1) internal/recall package structure and purpose in this project · (2) past incidents with large files accidentally committed to repository · (3) debug tracing log file handling and cleanup procedures · (4) git history conventions for log files in this codebase |

All three scenarios produced sensible, on-topic queries. Notable:

- **Pivot**: the companion correctly weighted toward the new direction (auth bug) but kept one fallback query about rate-limiting (the pre-pivot topic) — exactly the right judgment, since a pivot can revert.
- **Sparse "yes do it"**: the companion synthesized queries from prior context (auth+rate-limit refactor, worktrees) without anchoring on the literal "yes do it." Sparse-turn handling works without further engineering.
- **Tool result anomaly**: the companion caught the 47MB-file outlier specifically and proposed queries about accidentally-committed large files + repo conventions — reading the *anomaly*, not just the topic.

### Implications for the architecture

- v1 prompt is good enough to wire into a real companion. No further iteration needed before plumbing.
- A non-frontier model (qwen3.6-plus) is sufficient. Latency/cost stays in the budget from Phase 3 (~3–5s warm).
- The two-stage flow (bare recall → companion proposes targeted queries → companion calls engram_recall on each → aggregate → inject) is validated end-to-end on the prompt side. The plumbing side is what's next.

### Caveats / open items

- **Companion was given the recall output as input text**; in the real companion it would have `engram_recall` as a tool and call it itself. Need to verify tool-use works the same way (the model's tool-call decision should be similar but worth a check).
- Companion didn't actually CALL engram_recall on its proposed queries — that's a separate validation. Step is: feed each proposed query back through `engram_recall --query <q>` and verify hits.
- Output formatting (numbered list, single-line vs newline-separated) varies; downstream consumer needs to be tolerant or the prompt should pin a JSON-array shape.

## Phase 5 — Companion plumbing end-to-end

**Goal:** wire the companion into the OpenCode plugin and validate end-to-end with a planted-token verification.

**Status:** ✅ working (2026-05-03), with cost concerns flagged for the next pass.

### Architecture (revised mid-phase)

Initial split was `chat.message` → companion → write sidecar; `system.transform` → read sidecar → inject. Two problems surfaced:

1. **Recursion**: the companion's `opencode run` ALSO loads this plugin in its own process, which fires `chat.message`, which spawns *another* companion. Infinite recursion.
2. **Two hooks for one job** added complexity for no benefit.

**Revised**: do everything in `experimental.chat.system.transform`. Recursion is broken by passing `ENGRAM_COMPANION_MODE=1` in the spawned companion's env; the hook checks the env var at the top and short-circuits to the plain reminder injection if set.

### Implementation

`opencode/plugins/engram.ts`:

- `system.transform` (top): if `ENGRAM_COMPANION_MODE === "1"` → inject reminder only, return.
- Else: run `engram recall --no-external-sources` (project-scoped via Phase 2's filter); compose the companion prompt (Phase 4 v1 wording, asking for a memory-injection block); run `opencode run -m opencode/qwen3.6-plus -s <companion> --format json <prompt>` synchronously; capture the companion's text; append to `output.system[0]`.
- Companion session ID is stored at `~/.local/share/engram/companion-session/<primarySessionID>.txt` for reuse across primary turns.
- Trace events go to `~/.local/share/engram/companion-trace.jsonl` at every stage: `system.transform-start`, `recall-complete`, `companion-session-created`, `companion-complete`, `companion-injected` / `companion-skipped`, `system.transform-skipped-companion`.

### End-to-end validation

Planted a verifiable fact memory: `engram` → `secret verification token is` → `MAGENTA-VERIFICATION-PHRASE-7392`. Then asked a fresh primary OpenCode session: *"Tell me any unusual facts you remember about engram from our previous work together. Quote them exactly if you can."*

**Result**: primary's response included verbatim:

> "engram's secret verification phrase for end-to-end companion tests is `MAGENTA-VERIFICATION-PHRASE-7392` (planted today, 2026-05-03)"

Plus 6 other real project memories surfaced (ESC-interrupts-stuck-agents, FD-exhaustion bug, zombie-tasks bug, stale `coverage.out` after rebase, worktree chat path bug, "keywords blob suffixes" cleanup) — all entries from prior project session memories.

The trace shows the full pipeline firing: recall (~280–350ms), companion qwen call (~14–17s), inject. Recursion guard fires correctly: `system.transform-skipped-companion` events are logged for the inner opencode-run invocations the companion spawns.

### ⚠️ Cost issue — needs addressing before this is usable

`system.transform` fires **multiple times per primary turn** (once per LLM step — and a single user message can trigger several internal LLM steps even without explicit tool use). In the test, **4 companion runs fired for one user message**, each ~17s. Total ~68s of companion overhead for a single primary turn that would otherwise take ~5s. **~13× slowdown.**

The companion's actual work (qwen extraction over ~26KB of recall output) is ~17s, much higher than my Phase 3 estimate (~3–5s) because Phase 3 was just `engram show` of a tiny memory. With a full recall + extraction, qwen is doing substantive reasoning per call.

Caching paths to consider in Phase 6:

- **Per-turn cache keyed by latest user message timestamp/ID.** Compute companion once per turn (first system.transform fire); reuse for subsequent fires within the same turn. Invalidate when user sends new message.
- **Trim the companion's input.** Phase 4 used a synthetic, short transcript; Phase 5 hands it the full recall output (~26KB). The latency scales with input size. Truncating to last N user/assistant pairs would speed it up.
- **Skip the companion entirely on tool-followup turns** (no new user message → no new context → reuse prior result).
- **Use a faster companion model** (haiku-class) for the extraction step — qwen3.6-plus is overkill for "summarize this transcript."

### Implications for the architecture

- Architecture works end-to-end. Memory recall → companion synthesis → primary injection is fully validated.
- **Not yet shippable**: the fan-out cost makes it impractical for real use. Phase 6 = caching + cost reduction.
- The recursion guard pattern (env-var-as-mode-flag) is a generalizable trick for any plugin that recursively invokes its own host.

## Phase 6 — Cost reduction (next)

**Goal:** cut companion overhead from ~13× primary-turn cost to <2×, primarily through per-turn caching of the companion's output.

**Status:** not started.

## Phase 7 — Wire companion-emitted queries through recall

**Goal:** close the Phase 4 open item — instead of asking the companion to filter the bare recall output, ask it to emit targeted recall queries; have the plugin run each query through `engram recall` and inject the concatenated per-query results.

**Status:** in design (2026-05-03), ahead of Phase 6.

### Motivation

Phase 5 wired in a companion prompt that asks the companion to *filter* the bare project-history blob into a "memories worth injecting" list. That works, but it bypasses engram's actual relevance retrieval (memory ranking, two-phase Haiku extract). The companion is being asked to pick from a fixed input rather than to ask better questions.

Phase 4 already validated that a companion reliably emits on-topic queries from project context across three scenarios (pivot, sparse, tool-result). Phase 7 closes that loop: companion proposes queries → plugin runs them → primary sees query-targeted memories.

Sequencing ahead of Phase 6 (caching) is intentional — Phase 7 changes the shape of what is cached, so building the cache against the wrong shape is wasted work.

### Design

**Companion prompt (replaces the Phase 5 filter prompt):**

```
You are a memory steward observing a primary AI agent's project session.
Read the project history below and propose 3 to 5 targeted recall queries
that would surface helpful past memories about what is currently happening.

Output the queries only, one per line, no numbering, no commentary, no
other text. Each query should be 5 to 15 words capturing a specific facet
you want to recall about.

If nothing in the history is worth recalling on, output exactly:
NO QUERIES

PROJECT HISTORY (most recent message at end):
<recall output>
```

Adapted from Phase 4 v1: drops the numbered-list framing so a plain `split("\n")` is unambiguous; renames the empty sentinel from `no action` to `NO QUERIES` for log clarity.

**Plugin flow** (in `experimental.chat.system.transform`, replacing the current filter pass):

1. Run `engram recall --no-external-sources` (unchanged) to get the project-history blob.
2. Send the blob to the companion via `opencode run` with the new prompt. Companion session persistence and `ENGRAM_COMPANION_MODE=1` recursion guard are unchanged from Phase 5.
3. Parse companion output:
   - Split on `\n`, trim each line, drop empties → `lines: string[]`.
   - If `lines.length === 1 && lines[0] === "NO QUERIES"` → skip injection (trace `companion-skipped` with reason `no-queries`).
   - If the companion returned nothing at all (`lines.length === 0`, or the spawn produced empty output / non-zero exit / threw) → log and skip injection (trace `companion-skipped` with reason `empty-output`, or `companion-error` for thrown errors — both already exist in the current implementation).
4. For the remaining query lines, run `engram recall --query <q> --no-external-sources` for each in parallel via `Promise.all`.
5. Concatenate the results into a single block:

   ```
   ## Recalled memories

   ### Query: <query 1>
   <recall summary 1>

   ### Query: <query 2>
   <recall summary 2>
   ```

6. Append the block to the system prompt at the same point the current implementation injects.

The plugin makes no judgment about per-query result usefulness. Whatever each `engram recall --query <q>` returns is passed through verbatim. Filtering would require an extra LLM evaluation that this design intentionally avoids.

**Logging:**
- `companion-trace.jsonl`: existing stages plus one `secondary-recall-complete` event per query (query text, ms, output length).
- `companion-injections.log`: new shape per entry — the companion's emitted query lines (its output, fed into the secondary recalls) and the concatenated per-query recall outputs (what is actually injected).

### Validation

End-to-end against example project repos, mirroring the Phase 4/5 harnesses.

| # | Test | Pass criterion |
|---|---|---|
| 1 | **Phase 4 scenario replay** — re-run the pivot, sparse, and tool-result scenarios with the new prompt | All three scenarios produce 3–5 on-topic queries. Pivot weights toward the new direction (auth bug) while keeping at least one fallback to the pre-pivot topic. Tool-result scenario produces at least one query about the anomaly itself, not just the surrounding topic. |
| 2 | **Per-query payout** — for each scenario, run each emitted query through `engram recall --query <q> --no-external-sources` directly | For each scenario, at least one of the emitted queries returns a non-empty recall summary that is on-topic for that query. |
| 3 | **Empty project history** — fresh repo with no session history; run the new pipeline once | Companion emits exactly `NO QUERIES`. Plugin skips injection. Resulting system prompt contains no `## Recalled memories` block. `companion-trace.jsonl` shows a `companion-skipped` event with a `no-queries` reason. |
| 4 | **Nil/failed companion response** — point `opencode` at a shim that exits non-zero, run the pipeline once | Plugin logs `companion-error` (or `companion-run-failed`), skips injection, primary turn completes without crash. System prompt has no recall block. |
| 5 | **Phase 5 planted-token replay** — plant a fact memory with a unique token (e.g., `MAGENTA-VERIFICATION-PHRASE-N`) whose `situation` field is something like "asked about unusual facts about engram"; user message in the fresh primary `opencode run` is "Tell me any unusual facts you remember about engram from our previous work together. Quote them exactly if you can." | Token appears verbatim in the primary's response. Trace shows at least one companion-emitted query that overlaps the planted memory's situation, and the per-query recall result for that query contains the token. |

### Cost note

Each `system.transform` fire now does 1 companion call plus N (3–5) parallel `engram recall --query` calls. Each per-query recall internally does its own Haiku two-phase extraction, so this multiplies LLM work per fire compared to Phase 5. With `system.transform` firing 4–5× per primary turn, the unmitigated cost is materially higher than Phase 5. Phase 6 (per-turn caching keyed by user-message ID) becomes more important after Phase 7 lands. Cost measurements collected during the validation tests above feed directly into the Phase 6 cache design.

### Phase 7 — Validation findings (2026-05-04)

#### Validation 1 — scenario replay

Re-ran Phase 4's three scenarios against the new prompt, using `opencode/qwen3.6-plus`.

| Scenario | Companion's queries |
|---|---|
| Pivot | auth middleware intermittent 401 token validation errors debugging · rate limiting middleware implementation patterns internal codebase · authentication token validation previous fixes and troubleshooting · middleware error handling patterns in internal middleware directory · previous work on API authentication and token validation |
| Sparse | auth rate-limit module refactoring architecture changes · worktree branch setup and isolation strategy · multi-file refactoring subsystem coupling patterns |
| Tool result | debug trace log misplaced internal recall package cleanup procedure · proper storage location debug trace logs engram project · gitignore patterns prevent log files committed source tree · engram logging configuration log file paths setup · previous debug log cleanup incidents engram repository |

Pass criteria:
- Pivot: kept at least one query about pre-pivot rate-limiting topic + emphasized auth bug → **PASS** (query 2 covers rate-limiting; queries 1, 3, 5 cover auth bug)
- Sparse: synthesized queries from prior context (auth+rate-limit, worktrees) without anchoring on "yes do it" → **PASS** (all 3 queries draw from substantive content, none anchor on "yes do it")
- Tool result: at least one query specifically about the 47MB anomaly or accidentally-committed large files → **PASS** (query 1 directly addresses misplaced log; query 3 addresses accidentally-committed files)

#### Validation 2 — per-query payout

For each Validation 1 query, ran `engram recall --query <q> --no-external-sources` from the engram repo.

| Scenario | # | Query | Non-empty | On-topic |
|---|---|---|---|---|
| Pivot | 1 | auth middleware intermittent 401 token validation errors debugging | N | N |
| Pivot | 2 | rate limiting middleware implementation patterns internal codebase | N | N |
| Pivot | 3 | authentication token validation previous fixes and troubleshooting | Y | Y |
| Pivot | 4 | middleware error handling patterns in internal middleware directory | Y | Y |
| Pivot | 5 | previous work on API authentication and token validation | N | N |
| Sparse | 1 | auth rate-limit module refactoring architecture changes | N | N |
| Sparse | 2 | worktree branch setup and isolation strategy | Y | Y |
| Sparse | 3 | multi-file refactoring subsystem coupling patterns | Y | Y |
| Tool result | 1 | debug trace log misplaced internal recall package cleanup procedure | Y | Y |
| Tool result | 2 | proper storage location debug trace logs engram project | Y | Y |
| Tool result | 3 | gitignore patterns prevent log files committed source tree | Y | Y |
| Tool result | 4 | engram logging configuration log file paths setup | Y | Y |
| Tool result | 5 | previous debug log cleanup incidents engram repository | Y | Y |

Pass criterion: at least one query per scenario is non-empty + on-topic.

- Pivot: **PASS** (queries 3 and 4 both Y/Y — recalled engram project memories about token validation pipeline and middleware error handling patterns)
- Sparse: **PASS** (queries 2 and 3 both Y/Y — recalled worktree setup bugs/workflow and single-responsibility hook refactoring pattern)
- Tool result: **PASS** (all 5 queries Y/Y — recalled the 47MB debug-trace.log incident, proper log storage paths, gitignore patterns, and cleanup procedures)

Notable: the "pivot" scenario's auth-specific queries (1, 2, 5) returned empty because the engram repo has no auth middleware memories — the companion generated sensible queries but the underlying memory store doesn't contain that domain. The queries that hit (3, 4) did so because engram *does* have memories about token validation (the planted MAGENTA phrase and pipeline validation) and middleware patterns (hook refactoring). The tool-result scenario performed best (5/5) because the 47MB log incident was a real engram event that generated real memories.

#### Validation 3 — empty project history

Setup: fresh `/tmp/companion-empty` directory (only `.` and `..`), companion sessions cleared.

`engram recall --no-external-sources` from the empty dir produced: empty output (no bytes, no summary, no status message).

Trace events from this turn (19 total new events):

| Stage | Count |
|---|---|
| `system.transform-start` | 3 |
| `recall-complete` | 3 |
| `system.transform-skipped-companion` | 5 |
| `companion-session-created` | 2 |
| `companion-complete` | 3 |
| `companion-skipped` | 3 |

Companion-skipped events: 3, all with reason `no-queries` (sessionID `ses_20f3b468fffeLh4Dcv1iojNuHu`).

System-prompt check across new fires: 8 fires, 0 has-recall-block, 8 no-recall-block.

Pass criterion: at least one fire produced `companion-skipped` with a recognized reason and no `## Recalled memories` block in that fire's AFTER section → **PASS**

All 3 companion-complete events produced `no-queries` — qwen3.6-plus correctly emitted the sentinel on empty/sparse input rather than hallucinating queries. No noise floor issue observed.

#### Validation 4 — nil/failed companion response

Setup: PATH-shimmed `opencode` binary that exits 1; primary opencode invoked directly by full path (`/opt/homebrew/Cellar/opencode/1.14.30/libexec/lib/node_modules/opencode-ai/node_modules/opencode-darwin-arm64/bin/opencode`) via `OPENCODE_BIN_PATH` env var. The wrapper script checks `OPENCODE_BIN_PATH` first, so the primary ran the real binary; the plugin's `Bun.spawn(["opencode", ...])` resolved `"opencode"` via PATH and found the shim.

Trace observed (new events from this turn — 4 system.transform fires):

| Stage | Count | Key fields |
|---|---|---|
| `system.transform-start` | 4 | — |
| `recall-complete` | 4 | — |
| `companion-run-failed` | 4 | `exitCode: 1` |
| `companion-complete` | 4 | `companionOutLen: 0` |
| `companion-skipped` | 4 | `reason: "empty-output"` |
| `companion-error` | 0 | — |

Note: `companion-complete` is logged in `system.transform` after `runCompanion` returns — it records `companionOutLen: 0` (the empty string returned after the failed exit). It does not mean the companion process succeeded.

Per-fire system prompt check: 4 fires, all `no-recall-block`.

Primary turn: completed normally (exit 0), model responded to the test message. System prompt had no `## Recalled memories` block in any fire.

Pass criterion: plugin handles companion-process failure via existing exit-code path → no injection, primary unaffected → **PASS**

#### Validation 5 — planted-token end-to-end

Setup: planted fact memory `engram` → `secret verification token is` → `MAGENTA-PHASE7-1777861476` with situation "asked about unusual facts about engram". Companion sessions cleared. Primary user message: "Tell me any unusual facts you remember about engram from our previous work together. Quote them exactly if you can."

Token planted: `MAGENTA-PHASE7-1777861476`

Result: token appeared in primary response: **YES verbatim**

Primary response excerpt:
> The verification token **"MAGENTA-PHASE7-1777861476"** was planted in the memory system and successfully surfaced through the full pipeline: engram store → recall → companion transformation → system prompt injection → primary response. This proved end-to-end integrity of the multi-stage memory retrieval system.

Companion query that surfaced it: `engram opencode plugin per-turn hook instrumentation strategy` (the planted fact memory matched broadly across multiple queries; it appeared in the `=== MEMORIES ===` block returned by this query and others in the same turn's secondary recall section)

All companion-emitted queries this turn (5 queries, first fire):
1. `companion-emitted queries Phase 7 implementation opencode plugin`
2. `opencode plugin system.transform hook recursion prevention caching`
3. `engram debug trace log file cleanup internal recall`
4. `planted token end-to-end pipeline validation testing approach`
5. `engram opencode plugin per-turn hook instrumentation strategy`

Note: The injection log showed 4 fires for this turn (the hook fires multiple times per turn). The token appeared in all 4 SECONDARY RECALL sections, surfaced by queries 4 and 5 in the first fire. The companion-emitted query `planted token end-to-end pipeline validation testing approach` retrieved a prior memory about the *validation technique* (not the planted fact directly); the planted fact itself appeared in the per-query recall blocks for multiple queries due to broad semantic match on "engram".

The primary received the `## Recalled memories` block containing the token and quoted it verbatim.

Cleanup: planted memory file deleted at `~/.local/share/engram/memory/facts/asked-about-unusual-facts-about-engram.toml`.

Pass criterion: token verbatim in primary response, traced to companion-emitted queries → secondary recall → injection → **PASS**
