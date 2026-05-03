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

## Phase 4 — Companion prompt design (next)

**Goal:** find a prompt for the companion that reliably emits the right action (recall query, learn payload, or "no action") given a primary transcript snippet + the latest user message.

**Status:** not started.
