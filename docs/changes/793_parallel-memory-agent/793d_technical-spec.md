# Parallel Memory Agent - Technical Spec

## 793d1 Orientation

The product scope is the 13 requirements of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec), approved in [793b187 The product spec is approved after the owner's walkthrough](./793b_decision-log.md#793b187-the-product-spec-is-approved-after-the-owners-walkthrough). The [System](./793a_product-spec.md#793a1-problem-statement) must consult the [vault](./793a_product-spec.md#793a1-problem-statement) and save [lessons](./793a_product-spec.md#793a1-problem-statement) at the right [moments](./793a_product-spec.md#793a1-problem-statement) without the [working agent](./793a_product-spec.md#793a1-problem-statement) running a [memory command](./793a_product-spec.md#793a1-problem-statement), show the user what it did, never stall or break the [session](../../GLOSSARY.md#session), and be installed by `engram update` for Claude Code.

**Premise (owner).** The [System](./793a_product-spec.md#793a1-problem-statement) is built on Claude Code mods, not settings hooks: "the whole design is based around the novel capabilities mods provide" ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go)), and "with mods, the system can just print stuff to the user" ([793c40 Does the user still see please's audit list?](./793c_open-questions.md#793c40-does-the-user-still-see-pleases-audit-list)). A mod is a plugin whose `hooks/hooks.json` lists `"modules": ["./register.js"]`; Claude Code loads the module in process and calls its `register(on, options)`; a handler `on(event, matcher?, async ($, e, next) => …)` can observe, rewrite or answer an event and uses the mods API (`$`) for everything else ([Mods reference](https://code.claude.com/docs/en/plugins/mods/reference.md), [React to events](https://code.claude.com/docs/en/plugins/mods/events.md), [Use the mods API](https://code.claude.com/docs/en/plugins/mods/api.md), [Test a mod](https://code.claude.com/docs/en/plugins/mods/test.md), [type declarations](https://github.com/anthropics/claude-code/blob/main/mods/types/claude-code.d.ts)). The runtime is early access; this spec marks every capability it could not confirm.

**What exists, and the gap.** Every [vault](./793a_product-spec.md#793a1-problem-statement) operation the [System](./793a_product-spec.md#793a1-problem-statement) needs exists: [Query Engine](../../specs/technical/query-engine.md), [Note Authoring](../../specs/technical/note-authoring.md), [Note Amendment](../../specs/technical/note-amendment.md), [Transcript Reader](../../specs/technical/transcript-reader.md), `engram ingest --auto` and [Harness Deploy Sync](../../specs/technical/harness-deploy-sync.md). Missing is everything that runs them without the agent: observing [moments](./793a_product-spec.md#793a1-problem-statement) and adding text to the agent's [context](./793a_product-spec.md#793a1-problem-statement) without its own tool call ([793a2n Relevant memory reaches the agent at the moments](./793a_product-spec.md#793a2n-relevant-memory-reaches-the-agent-at-the-moments), [793a2d Delivered memory is attributable](./793a_product-spec.md#793a2d-delivered-memory-is-attributable), [793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only)); reading what each agent's context holds ([793a2e Nothing already in context is delivered again](./793a_product-spec.md#793a2e-nothing-already-in-context-is-delivered-again)); judgement without an agent, over a [subagent](../../GLOSSARY.md#subagent)'s [transcript](../../GLOSSARY.md#transcript), with secrets removed and duplicates refused ([793a3a Lessons are captured without the agent](./793a_product-spec.md#793a3a-lessons-are-captured-without-the-agent), [793a3q Sound conclusions are persisted by the System](./793a_product-spec.md#793a3q-sound-conclusions-are-persisted-by-the-system), [793a2l Indexing and curation happen without the agent](./793a_product-spec.md#793a2l-indexing-and-curation-happen-without-the-agent), [793a4f No delivery or capture in its own sessions](./793a_product-spec.md#793a4f-no-delivery-or-capture-in-its-own-sessions)); finding the [backlog](./793a_product-spec.md#793a1-problem-statement) of other [sessions](../../GLOSSARY.md#session)' transcripts and tracking a [watermark](#793d1b-terms) for each [transcript source](./793a_product-spec.md#793a1-problem-statement), so that a session that ended before its [lessons](./793a_product-spec.md#793a1-problem-statement) were saved is captured once, after the fact ([793a3a Lessons are captured without the agent](./793a_product-spec.md#793a3a-lessons-are-captured-without-the-agent), [793a5c Other harnesses are unchanged](./793a_product-spec.md#793a5c-other-harnesses-are-unchanged)); bounding, queuing and containing the work, a warning surface and a summary outside the transcript ([793a4b The session is never broken or stalled](./793a_product-spec.md#793a4b-the-session-is-never-broken-or-stalled), [793a5h The user sees a summary of the memory activity](./793a_product-spec.md#793a5h-the-user-sees-a-summary-of-the-memory-activity)); and a deploy set for a Claude Code plugin ([793a4h Installed by engram update and active in Claude Code](./793a_product-spec.md#793a4h-installed-by-engram-update-and-active-in-claude-code), [793a5c Other harnesses are unchanged](./793a_product-spec.md#793a5c-other-harnesses-are-unchanged)). [793a2k Intentional memory stays available and is never blocked](./793a_product-spec.md#793a2k-intentional-memory-stays-available-and-is-never-blocked) is met by changing nothing the agent uses. Mods supply the observing, [delivery](./793a_product-spec.md#793a1-problem-statement), background execution and surface; judgement runs through the mod on the cheapest capable model tier ([793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier)), on the delivery side as well as the capture side ([793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)); the deploy extends the existing sync.

**Recommended solution.**

1. The **Memory Mod** ([793d3 Added: memory-mod](#793d3-added-memory-mod)), a thin adapter loaded by Claude Code: it forwards the events of [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) to the Go command, adds the text it returns through the documented [context](./793a_product-spec.md#793a1-problem-statement) channels, relays its model requests through `$.model.complete`, holds the [delivery](./793a_product-spec.md#793a1-problem-statement) of a step until the next tool result, runs the slow work in a background queue, shows one status line, and swallows every failure into a warning.
2. The **`engram auto run` command** ([793d4 Added: auto-job-runner](#793d4-added-auto-job-runner)) and the components behind it: [793d6 Added: memory-delivery](#793d6-added-memory-delivery) (decide the [moment](./793a_product-spec.md#793a1-problem-statement), by a judgement where the event does not name it, ask the [vault](./793a_product-spec.md#793a1-problem-statement), drop what the agent holds and what does not bear on the moment, write the [delivery](./793a_product-spec.md#793a1-problem-statement)), [793d7 Added: auto-capture](#793d7-added-auto-capture) (find the [backlog](./793a_product-spec.md#793a1-problem-statement), judge and write [lessons](./793a_product-spec.md#793a1-problem-statement) and conclusions) and [793d9 Added: auto-curation](#793d9-added-auto-curation) (settle pending [offers](../../GLOSSARY.md#offer)). Go orchestrates over the existing injected interfaces, and every judgement is relayed to the cheapest capable tier ([793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)).
3. An extension of Harness Deploy Sync ([793d10 Added: harness-deploy-sync (mod deploy)](#793d10-added-harness-deploy-sync-mod-deploy)) that deploys the mod for Claude Code only, driven by the update flow of [793d2f Mod install](#793d2f-mod-install). No existing canonical requirement is modified: the Memory Mod is validated and tested through the existing integration gate ([793d3g Validated and tested through the existing checks](#793d3g-validated-and-tested-through-the-existing-checks)).

The flows are [793d2 Added: flows](#793d2-added-flows); the option comparisons that bear on the contract are [793d13 Relative option impact](#793d13-relative-option-impact), and the other choices are in the decision log ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go) to [793b207 Late delivery at stop is a MAY and nothing else is late](./793b_decision-log.md#793b207-late-delivery-at-stop-is-a-may-and-nothing-else-is-late), [793b211 Option comparisons and rejected alternatives moved out of the technical spec](./793b_decision-log.md#793b211-option-comparisons-and-rejected-alternatives-moved-out-of-the-technical-spec)). There is no change diagram: each flow diagram shows the final system.

### 793d1a What the design relies on in Claude Code mods

"Verified (docs)" is stated by the pages cited above; "Verified (declarations)" by [claude-code.d.ts](https://github.com/anthropics/claude-code/blob/main/mods/types/claude-code.d.ts) (written by Claude Code 2.1.277); "Verified (tested)" was observed on Claude Code 2.1.296; "To verify" could not be confirmed, and the implementation checks it with `claude plugin validate` and `claude plugin test`, using the fallback shown.

| # | Capability relied on | Status | If it does not hold |
| --- | --- | --- | --- |
| 1 | A plugin whose `hooks/hooks.json` has `modules` loads a module exporting `register(on, options)`; handlers are `($, e, next)`; Claude Code ≥ 2.1.287 | Verified (docs) | None; the premise |
| 2 | A folder under `~/.claude/skills` with `.claude-plugin/plugin.json` loads as a plugin (`name@skills-dir`) in every [session](../../GLOSSARY.md#session), with no flag, install step or settings edit | Verified (docs) | A local-directory marketplace ([793d13h Install path](#793d13h-install-path)) |
| 3 | That plugin's hooks module (`hooks.json` with `modules`) loads | Verified (tested) | As row 2 |
| 4 | The load works through a symlink into the engram root | Verified (tested) | Manifest mode (real files and a record) |
| 5 | `prompt.submit` fires for every origin (composer, task notification, scheduled trigger, peer [session](../../GLOSSARY.md#session)) and `next({ ...e, context })` adds text the model reads after the prompt as typed | Verified (declarations: `PromptOrigin`, `PromptSubmitInput.context`); a typed slash command: To verify | `session.receive` for a missed origin; `classic.UserPromptExpansion` for a slash command |
| 6 | `additionalContext` from `classic.PostToolUse`, `classic.PostToolUseFailure` and `classic.SubagentStart` reaches the model next to the result or at the [subagent](../../GLOSSARY.md#subagent)'s start; [context](./793a_product-spec.md#793a1-problem-statement) added by a `tool.call` hook appears in the [transcript](../../GLOSSARY.md#transcript) as an `attachment` row of type `hook_additional_context`. Whether `prompt.submit` context is stored as a transcript row is **unverified**, and the transcript assertions of [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) and [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) depend on it | Verified (docs, declarations, a 2.1.296 [transcript](../../GLOSSARY.md#transcript)) for the classic events; To verify for `prompt.submit` | Deliver the prompt's [context](./793a_product-spec.md#793a1-problem-statement) through `classic.UserPromptSubmit` `additionalContext`, the settings hooks' channel |
| 7 | `agent.spawn` carries the Agent call's `prompt`, `description` and `fork`; `classic.SubagentStart` only `agent_id` and `agent_type`; `$.agent.list()` gives an agent's `description` | Verified (declarations) | Match the start to the spawn by order only |
| 8 | `classic.SubagentStop` carries `agent_transcript_path` and `last_assistant_message`; the file is complete at the stop | Fields: Verified (declarations); completeness: To verify | Retry the read once after one second |
| 9 | `session.compact` fires for the main agent and for each [subagent](../../GLOSSARY.md#subagent) or fork compaction, the latter with `agentId`; `turn.complete` also fires for an interrupted turn; module variables survive `/clear`, `/resume` and `/branch` (only `$.state` resets; `session.start` does not fire again) and are lost when the module reloads | Verified (docs, declarations) | None |
| 10 | `$.session.messages()` returns the newest 4096 messages the next model request carries (so compaction and a resume are reflected), main loop only | Verified (declarations); after a compaction or resume: To verify | The [ledger](#793d1b-terms) is lost and an item may be delivered again |
| 11 | A [context](./793a_product-spec.md#793a1-problem-statement) block added through row 5 or 6 appears in `$.session.messages()` | To verify | The [ledger](#793d1b-terms) covers it until a compaction or resume; [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios) then rests on this row |
| 12 | Time inside a mods API call (`$.process.run`, `$.model.complete`) does not count against a hook's 10 s budget, so a 25 s `momentDeadlineMs` is allowed; `.catch` handlers have 1 s | Verified (docs, declarations) | None |
| 13 | `$.model.complete({ model, prompt, system, maxTokens, timeoutMs })` returns `{ isAnswered, text, reason, usage }`; `$.session.model()` names the [session](../../GLOSSARY.md#session)'s model, which the judgements do not use | Verified (docs); use inside a timer callback: To verify | The mod enforces its own deadline |
| 14 | `$.ui.status(text)` pins one line under the prompt, drawn with the prefix "⚠ <mod name>:", replaced by the next call and cleared by `undefined`; it is not a [transcript](../../GLOSSARY.md#transcript) row | Pinned line: Verified (declarations); prefix and not a [transcript](../../GLOSSARY.md#transcript) row: To verify. `$.ui.log` adds a transcript row and is not used | `$.ui.toast`, documented to leave the [transcript](../../GLOSSARY.md#transcript) and the model untouched |
| 15 | Retired: the [System](./793a_product-spec.md#793a1-problem-statement) no longer relies on `$.store` ([793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped)); the row is kept so that the numbers of the rows after it stay | Not applicable | None |
| 16 | The mod is inactive under `--safe-mode`, `--bare`, `disableAllHooks`, managed policy, an untrusted folder, the remote rollout switch, WSL and Claude Code older than 2.1.287; `claude -p` and the VS Code panel do not draw the status line | Verified (docs) | The [System](./793a_product-spec.md#793a1-problem-statement) is silently inactive there ([793c66 Which platform cases where the System is silently inactive does the product promise cover?](./793c_open-questions.md#793c66-which-platform-cases-where-the-system-is-silently-inactive-does-the-product-promise-cover)) |
| 17 | The Claude Code version each event and field first appears in | To verify | The mod's README records the version verified against, and the integration test of [793d3g Validated and tested through the existing checks](#793d3g-validated-and-tested-through-the-existing-checks) runs against it ([793b236 The mod is validated and tested through the existing integration gate, not a new check target](./793b_decision-log.md#793b236-the-mod-is-validated-and-tested-through-the-existing-integration-gate-not-a-new-check-target)) |
| 18 | `$.model.complete` accepts a model: `model` is an alias (`haiku`, `sonnet`, `opus`) or a full id, resolved like a `--model` value, and the call bills to the [session](../../GLOSSARY.md#session)'s credentials (the docs' own example passes `model: 'haiku'`); the call rejects for a model the organization blocks | Verified (docs: mods API, "Send one prompt"; declarations: `model.complete` request) for the alias and full id; whether the `haiku` alias resolves on every provider (Bedrock, Vertex, Foundry, a gateway) and the rejection of a blocked model: To verify | The [job](#793d1b-terms) ends as an error with a warning naming the step and the model; the [session](../../GLOSSARY.md#session)'s model is not used instead, because it would silently raise the cost the owner chose to avoid ([793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier)); [`judgeTiers`](#793d1b-terms) can name a full id for such a provider |
| 19 | `classic.SessionStart` carries the main agent's `transcript_path`, the file its [session](../../GLOSSARY.md#session)'s [transcript](../../GLOSSARY.md#transcript) is written to (a common field of the classic events) | To verify | The mod sends no `transcriptPath`: the [notes](./793a_product-spec.md#793a1-problem-statement) of a turn name the [session](../../GLOSSARY.md#session) id in place of the [transcript](../../GLOSSARY.md#transcript), the session's own [watermark](#793d1b-terms) is not advanced, and its transcript may be captured again as [backlog](./793a_product-spec.md#793a1-problem-statement) after it ends, where the reconcile step finds its notes covered |
| 20 | `turn.step` is an async-generator hook that fires before each model request of the main agent (no `agentId`); `yield* next(e)` evaluates to a result whose `answer` is the response's visible text (`""` for a tool-only step) and whose `toolUses` lists its tool calls; the step input carries only `turnId`, `index`, `model`, `effort`, `messageCount` and `agentId`, no text; `$.session.messages()` entries are `{ role, text, toolUses }` | The event, the generator form, `agentId`, the result's `answer` and `toolUses`, and the message shape: Verified (docs: React to events, "Follow a turn"; declarations: `TurnStepInput`, `TurnStepResult`, `SessionMessage`). That a pass-through generator (every chunk forwarded, the result returned unchanged) adds no visible delay and drops no chunk, and that the step's own assistant message is in `$.session.messages()` when `next(e)` returns: To verify | Start the [step job](#793d1b-terms) from the next main-agent `classic.PostToolUse` of any tool instead, reading the newest assistant message with text from `$.session.messages()`; the [delivery](./793a_product-spec.md#793a1-problem-statement) then rides the result after that one |

### 793d1b Terms

The terms **moment**, **delivery**, **lesson**, **Integration**, **working agent**, **context**, **vault**, **transcript source** and **backlog** are as in [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement). This spec adds these:

- **Cue:** the event pattern at which [793d6a Moments and lookup](#793d6a-moments-and-lookup) looks for a moment: the trigger for a lookup. A cue either names its moment (a prompt, a `please` or `route` call, a dispatch, a subagent's start) or is a **judged cue** (the main agent's first edit and first failed tool result of a turn, and a step of the main agent that ended with text and a tool call), whose moment a `classify` request decides.
- **Job:** one unit of work that `engram auto run` advances in rounds: a moment job (one event, or one step; it runs one to three rounds, as its cue and its candidates need), a **step job** (a moment job started by a step of the main agent, run in the background), a capture job (one window; a **backlog capture job** is a capture job whose window is one transcript source's content after its watermark), a **scan job** (the backlog scan; it finishes in its one round) or a curation job (the pending offers).
- **Round:** one run of `engram auto run` that advances a job.
- **Judgement request:** a model request the command asks the mod to relay. Its kind is a key of `judgeTiers`.
- **Classify request:** the judgement request of kind `classify`: from the recent texts, which of the four decision moments of recall (endorse or rank an approach, declare work done, diagnose a failure the agent cannot explain, start building a new approach) the agent is about to reach, and on what.
- **Relevance request:** the judgement request of kind `relevance`: from a moment and the candidate items that remain after presence, which of them bear on that moment.
- **Recent texts:** the texts of the last `classifyWindowMessages` user and assistant messages of the main conversation, the oldest cut first to `classifyWindowChars` characters in all; for a step job the step's own response text is the newest.
- **Held delivery:** the delivery of a step job, which the mod keeps until the next tool result of the main agent that forms no job of its own.
- **Ledger:** the set of item ids delivered to one agent.
- **Window:** the messages one capture job judges: for a main-agent turn, the last assistant message before the turn's last human prompt, that prompt and everything after it; for a subagent, its whole transcript; for a backlog capture job, the entries of its transcript source after its watermark, up to `backlogWindowBytes` bytes.
- **Watermark:** the time up to which the System has captured one transcript source: its entries at or before it are covered, and its entries after it are the source's uncaptured content.
- **Live source:** a transcript source whose session may still be running: the transcript of the main agent's own session, or a file modified less than `backlogQuietMs` ago.
- **Task:** the text a moment job carries as what the receiving agent is working on: the latest human-origin prompt of at least `taskMinChars` characters, else the current prompt (for a starting subagent, its task prompt).
- **`judgeTiers`:** the configuration table from a request kind (`extraction`, `reconcile`, `curation`, `classify`, `relevance`) to the model its judgement requests name, an alias or a full id; every kind defaults to the cheapest tier.
- **`need`:** a job result that asks the mod to relay the judgement requests it lists; the next round carries their replies as `answers`.
- **`done`:** a job result that finishes the job, carrying its `events` and `notices` and, as the job's kind requires, a delivery or a warning.
- **`state`:** the opaque value a `need` returns, holding everything the next round needs, which the mod sends back unchanged with the `answers`.
- **`events`:** the integer counts (`delivered`, `added`, `amended`, `curated`) a job reports of the memory work it did.
- **`notices`:** the short texts a job reports, for the summary, about work that completed with a caveat (for example a note saved while its offer is queued).

## 793d2 Added: flows

Each of [793d2a Moment delivery](#793d2a-moment-delivery), [793d2b Turn memory work](#793d2b-turn-memory-work) and [793d2f Mod install](#793d2f-mod-install) is a flow spec (canonical destination `docs/specs/technical/<flow>-flow.md`); [793d2e Job order within a round](#793d2e-job-order-within-a-round) is the second interaction requirement of the flow of [793d2b Turn memory work](#793d2b-turn-memory-work). Every flow requirement names its one owner, and a flow whose steps belong to two owners has one requirement per owner. IDs appear on arrows only: the behavior requirement an arrow realizes and, where the flow's requirement constrains it, the flow's ID. Claude Code and the user are external actors, so their arrows carry the flow's ID.

### 793d2a Moment delivery

Canonical destination: `docs/specs/technical/moment-delivery-flow.md` (title `Moment Delivery Flow - Technical Spec`).

When Claude Code fires an event at which the [working agent](./793a_product-spec.md#793a1-problem-statement) may need memory, the Memory Mod hands it to the runner, which has Memory Delivery decide the [moment](./793a_product-spec.md#793a1-problem-statement), ask the [vault](./793a_product-spec.md#793a1-problem-statement) and drop what the receiving agent holds and what does not bear on the moment; the mod adds the returned text through the event's [context](./793a_product-spec.md#793a1-problem-statement) channel. Two model judgements take part, relayed by the mod to the cheapest capable tier ([793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)): a classification of the moment where the event does not name it, and a relevance filter on the candidates. A step of the main agent that ended with text and a tool call starts the same [job](#793d1b-terms) in the background, and its delivery waits for the next tool result, because the engine offers no channel inside a response ([793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context)). Whatever fails, the event continues as if the [System](./793a_product-spec.md#793a1-problem-statement) were absent and the user is told.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant Mod as Memory Mod
    participant Run as Auto Job Runner
    participant Del as Memory Delivery
    participant Qry as Query Engine
    participant User as User

    CC->>Mod: 793d2a, 793d3a event: prompt, tool result, failed tool, subagent start, or a step that ended with text and a tool call
    Note over Mod: a step's job runs in the background, and its delivery is held for the next tool result (793d3b)
    Mod->>Run: 793d2a, 793d3b, 793d4a moment job: event, recent texts, context, ledger, task
    alt program cannot start, vault unreadable, deadline passed, a judgement call failed, or a step throws
        Run-->>Mod: 793d4a a warning, or no result
        Mod-->>CC: 793d2a, 793d3a the event continues unchanged
        Mod->>User: 793d2a, 793d3f a warning naming the step and the reason
    else the job runs
        Run->>Del: 793d6a, 793d6b the event, the recent texts and the context
        opt a judged cue
            Del-->>Run: 793d6a the classify request
            Run-->>Mod: 793d2a, 793d4a need: the classify request
            Mod->>CC: 793d2a the model request, relayed
            CC-->>Mod: 793d2a the reply
            Mod->>Run: 793d2a, 793d4a the answers and the state
            Run->>Del: 793d6a the reply as moments with subjects, or none
        end
        alt the cue names its moment, or a moment is named
            Del->>Qry: 792b24c1 query with the moment's text and phrases
            Qry-->>Del: 792b24c1 the items
            opt items remain after presence
                Del-->>Run: 793d6b the relevance request
                Run-->>Mod: 793d2a, 793d4a need: the relevance request
                Mod->>CC: 793d2a the model request, relayed
                CC-->>Mod: 793d2a the reply
                Mod->>Run: 793d2a, 793d4a the answers and the state
                Run->>Del: 793d6b the reply as the items that bear on the moment
            end
        else no moment is named
            Note over Del: no query
        end
        Del-->>Run: 793d6a, 793d6b delivery text and ids, or empty
        Run-->>Mod: 793d4a done
        alt a delivery
            Mod-->>CC: 793d2a, 793d3a the text, in the event's context channel (a step's: the next tool result's)
        else no moment, or nothing remains
            Mod-->>CC: 793d2a, 793d3a the event unchanged
        end
    end
```

Requirement: 793d2a The Memory Mod MUST, for each event of [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), send the runner one [moment](./793a_product-spec.md#793a1-problem-statement) [job](#793d1b-terms) built from the event, the [recent texts](#793d1b-terms), the receiving agent's [context](./793a_product-spec.md#793a1-problem-statement), its [ledger](#793d1b-terms) and the [task](#793d1b-terms) ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)), run it with the relay loop of [793d2b Turn memory work](#793d2b-turn-memory-work) (the same loop: [`need`](#793d1b-terms) results, relayed calls, [`done`](#793d1b-terms)), and MUST wait for the result at most `momentDeadlineMs` (25 s, counted from the event; an initial value, tuned under #850) before letting the event continue. The deadline covers the lookup (about 11 s) and up to two relayed [judgement requests](#793d1b-terms), a `classify` and a `relevance` request, each bounded by `momentJudgeDeadlineMs` (5 s) in place of `judgeDeadlineMs`: 11 s and two calls of 5 s make 21 s, which leaves 4 s for the runner's process starts. When the result carries a [delivery](./793a_product-spec.md#793a1-problem-statement), the mod MUST add its text through the event's channel, record the delivered ids in the receiving agent's ledger and let the event continue otherwise unchanged. When the result is done without a delivery (the event is no moment, the classification named none, no candidate remained, the relevance reply named no item, or nothing remained after the receiving agent's context was subtracted) the mod MUST let the event continue unchanged and show nothing. It MUST NOT send a second job for one event (the [rounds](#793d1b-terms) of one job are not second jobs) or start a job for a delivery. A step of the main agent that ended with text and at least one tool call starts its job from a `$.clock.after` callback, in the background and one at a time (a step that ends while a step job runs starts none), and the mod MUST NOT wait for it, so that the step is not delayed. The delivery of a step job is held, and the mod MUST add its text as `additionalContext` to the first later `classic.PostToolUse` or `classic.PostToolUseFailure` of the main agent that forms no job of its own, the next channel the engine offers ([793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context)), and MUST drop it instead, adding nothing, when `turn.complete` fires first, when `momentDeadlineMs` passed before the job finished, or when any of its ids entered the ledger after the job began. When a step fails, because the `engram` program cannot start or the runner reports that the [vault](./793a_product-spec.md#793a1-problem-statement) cannot be read (`unavailable`), because `momentDeadlineMs` or `momentJudgeDeadlineMs` passed (`timeout`), or because the step threw, returned output the mod cannot read, relayed a call that failed or went unanswered, or ended the job as an error (`error`), the mod MUST let the event continue exactly as it would without the [System](./793a_product-spec.md#793a1-problem-statement), with no error, text or delay beyond the deadline reaching the agent and no user input lost, and MUST show the user one warning on the surface of [793d3f Memory-activity line](#793d3f-memory-activity-line) naming the step and the reason, the step being `classification` or `relevance` for a failed judgement call; a failed classification skips the moment (nothing is queried) and a failed relevance filter delivers nothing. A job that finds nothing to do MUST NOT produce a warning.

Owner: Memory Mod ([793d3 Added: memory-mod](#793d3-added-memory-mod)).

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios), [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios).

#### 793d2a1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2a1a A prompt that is a moment is delivered with the prompt
  Given the runner returns a delivery for a prompt
  When Claude Code fires prompt.submit for it
  Then the runner received exactly one moment job and none for the delivery
  And the mod continues the event with the delivery as one extra context block and the prompt text unchanged
  And the delivered ids are in the main agent's ledger

Scenario Outline: 793d2a1b Nothing delivered lets the event continue unchanged
  Given the runner returns done with no delivery because <reason>
  When Claude Code fires a tool-result event
  Then the mod continues the event unchanged, adds no context and shows no warning
  Examples:
    | reason                                             |
    | the event is no cue, or the classification named no moment |
    | every item is already in the agent's context       |
    | the relevance reply named no item                  |

Scenario: 793d2a1d The wait is bounded
  Given the runner takes longer than momentDeadlineMs
  When Claude Code fires a tool-result event
  Then the event continues unchanged after momentDeadlineMs and the warning gives timeout

Scenario Outline: 793d2a1g A failure continues the event and warns the user
  Given <cause> while the mod handles a prompt.submit event
  When the handler finishes
  Then the prompt continues with its text and no context entry
  And the status line warns that automatic memory was skipped, with the reason <class>
  And the agent's context holds no error text, no warning and no denied or dropped event
  Examples:
    | cause                                       | class       |
    | the engram program cannot start             | unavailable |
    | the runner reports the vault cannot be read | unavailable |
    | the runner's output is not a JSON document  | error       |
    | the handler throws                          | error       |

Scenario: 793d2a1h A moment named mid-turn is delivered at the next tool result
  Given the main agent's step ends with the text "the tests pass; I will declare the rename done next" and a tool call
  And the classification names declare-done with the subject "declaring the Go package rename done"
  And the lookup returns note X, the relevance reply keeps it, and the step job finishes while the agent works
  When the main agent's next tool result is a Read result
  Then the mod continues that event with note X as one additionalContext block and X is in the main agent's ledger
  And the step itself continued at once, unchanged, with nothing added to it

Scenario: 793d2a1i A step whose classification names no moment delivers nothing
  Given the main agent's step ends with text and a tool call and the classification names no moment
  When the step job finishes
  Then the runner made no query, no relevance request was relayed, no later event of the turn carries context from it and no warning is shown

Scenario Outline: 793d2a1j A failed judgement skips the moment and the event continues
  Given the first failed tool result of a turn, and the <request> call <cause>
  When the mod handles the event
  Then the event continues unchanged with no context entry and nothing is delivered for it
  And the status line warns that automatic memory was skipped at <step>, with the reason <class>
  And the agent's context holds no error text and no warning
  Examples:
    | request   | cause                              | step           | class   |
    | classify  | is unanswered (isAnswered false)   | classification | error   |
    | classify  | passes momentJudgeDeadlineMs       | classification | timeout |
    | relevance | is unanswered (isAnswered false)   | relevance      | error   |
    | relevance | passes momentJudgeDeadlineMs       | relevance      | timeout |

Scenario: 793d2a1k The three rounds of a judged cue fit in the deadline
  Given the first failed tool result of a turn, a classify call taking 4 s, a lookup taking 11 s and a relevance call taking 4 s
  When the mod handles the event
  Then the event continues with the delivery inside momentDeadlineMs and no warning is shown

Scenario: 793d2a1l A step job never delays its step
  Given a step that ended with text and a tool call, and a step job that takes 20 s
  When the mod handles the step
  Then the step's response and its tool call proceed at once, and a second step that ends while the job runs starts no second job
```

### 793d2b Turn memory work

Canonical destination: `docs/specs/technical/turn-memory-work-flow.md` (title `Turn Memory Work Flow - Technical Spec`).

When a top-level turn ends, or a [subagent](../../GLOSSARY.md#subagent) stops, the mod lets the event continue at once and queues the [System](./793a_product-spec.md#793a1-problem-statement)'s work: judging and writing [lessons](./793a_product-spec.md#793a1-problem-statement) and conclusions; capturing the [backlog](./793a_product-spec.md#793a1-problem-statement), the [transcripts](../../GLOSSARY.md#transcript) of other [sessions](../../GLOSSARY.md#session) that are no longer running; bringing the chunk index up to date; settling pending [offers](../../GLOSSARY.md#offer); showing the user what happened. The mod relays the model requests the runner asks for. A failed step is shown as a warning and the queue continues. Nothing is kept for a later session: work still queued when a session ends is lost, and what it would have captured is found again by the scan of a later turn, because its source's [watermark](#793d1b-terms) did not advance. The flow has two interaction requirements, one per owner: this requirement for the Memory Mod and [793d2e Job order within a round](#793d2e-job-order-within-a-round) for the Auto Job Runner.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant Mod as Memory Mod
    participant Run as Auto Job Runner
    participant Cap as Auto Capture
    participant Cur as Auto Curation
    participant Qry as Query Engine
    participant Aut as Note Authoring
    participant Amd as Note Amendment
    participant Ing as Ingestion Source Discovery
    participant Rdr as Transcript Reader
    participant Dur as Durable File Store
    participant User as User

    CC->>Mod: 793d2b, 793d3a turn ended, or subagent stopped
    Mod-->>CC: 793d2b, 793d3a the event continues at once
    Note over Mod: the work is queued and runs one item at a time, each from a timer
    Mod->>Run: 793d2b, 793d4a capture job: the window, or the subagent's transcript path
    Run->>Cap: 793d2e, 793d7a the window
    alt a model request is needed
        Cap-->>Run: 793d7a the extraction request
        Run-->>Mod: 793d2b, 793d4a need: the request
        Mod->>CC: 793d2b the model request, relayed
        CC-->>Mod: 793d2b the reply
        Mod->>Run: 793d2b, 793d4a the answers and the state
        Run->>Cap: 793d2e, 793d7a the reply as candidates
        Cap->>Qry: 792b24c1 query by each candidate's situation
        Qry-->>Cap: 792b24c1 the nearest notes
        Cap-->>Run: 793d7a one reconcile request per candidate
        Run-->>Mod: 793d2b, 793d4a need: the requests
        Mod->>CC: 793d2b the model requests, relayed
        CC-->>Mod: 793d2b the replies
        Mod->>Run: 793d2b, 793d4a the answers and the state
        Run->>Cap: 793d2e, 793d7c the reconcile replies
        alt a candidate is covered
            Cap->>Amd: 792b14c9 record a use of the existing note
            Amd-->>Cap: 792b14c9 done
        else a candidate is absent
            Cap->>Aut: 792b12c3 write the note
            Aut-->>Cap: 792b12c3 the path, or the parent-queued notice
        end
        Cap-->>Run: 793d7c the events and notices
    else the window is empty or holds no candidate
        Cap-->>Run: 793d7a no request
    end
    opt the job ended without a warning and its source is known
        Cap->>Dur: 792b3c1 advance the source's watermark
        Dur-->>Cap: 792b3c1 done
    end
    Run-->>Mod: 793d2b, 793d4a done with the events and notices
    Mod->>Run: 793d2b, 793d4a scan job: the session id and its transcript path
    Run->>Cap: 793d2e, 793d7e the session
    Cap->>Ing: 792b17c7 the swept files, with their origins
    Ing-->>Cap: 792b17c7 the swept files, with their origins
    loop each candidate source, most recently modified first, until backlogPerTurn hold content
        Cap->>Rdr: 793d16a the source's entries after its watermark
        Rdr-->>Cap: 793d16a the entries, the time of the last, and whether the budget stopped the read
        opt the source holds no content
            Cap->>Dur: 792b3c1 advance the watermark to the time of the scan
            Dur-->>Cap: 792b3c1 done
        end
    end
    Cap-->>Run: 793d7e the paths of the sources that hold content
    Run-->>Mod: 793d2b, 793d4a done with the sources
    loop each source returned
        Mod->>Run: 793d2b, 793d4a backlog capture job: the source's path
        Note over Run,Cap: the rounds of the capture job above, over the source's entries after its watermark
        Run-->>Mod: 793d2b, 793d4a done with the events and notices
    end
    Mod->>Ing: 792b17c3 engram ingest --auto
    Ing-->>Mod: 792b17c3 the index is up to date
    Mod->>Run: 793d2b, 793d4a curation job
    Run->>Cur: 793d2e, 793d9a the pending offers
    alt a pending offer exists
        Cur->>Amd: 792b14c14 settle by the judged outcome
        Amd-->>Cur: 792b14c14 done
    else none
        Note over Cur: finishes with no request
    end
    Cur-->>Run: 793d9a the curation events
    Run-->>Mod: 793d2b, 793d4a done with the events
    alt a step failed, timed out or was unavailable
        Mod->>User: 793d2b, 793d3f a warning naming the step and the reason
    else every step finished
        Mod->>User: 793d2b, 793d3f the memory-activity line, or nothing
    end
```

Requirement: 793d2b The Memory Mod MUST, on `turn.complete` of the main agent (an interrupted turn included) and on `classic.SubagentStop`, take what the capture needs (for a turn, the [window](#793d1b-terms) of [793d7a Judging a window](#793d7a-judging-a-window) formed from `$.session.messages()` at that [moment](./793a_product-spec.md#793a1-problem-statement), with the main agent's `transcriptPath` (from `classic.SessionStart`) and the time the turn ended, `turnEndedAt`; for a [subagent](../../GLOSSARY.md#subagent), the stop's `agent_transcript_path` and `last_assistant_message`), return from the handler without waiting, and queue the work in one in-memory first-in-first-out queue that runs one item at a time, each started from a `$.clock.after` callback: a subagent's capture [job](#793d1b-terms) at its stop; at a turn's end the turn's capture job, then one [scan job](#793d1b-terms) (carrying `$.session.id()` and the `transcriptPath`), then one [backlog capture job](#793d1b-terms) for each source the scan returns, in the order returned, then `engram ingest --auto` (`ingestDeadlineMs`), then one curation job, then the summary. It MUST run a capture, scan, curation or [moment](./793a_product-spec.md#793a1-problem-statement) job with one loop (a moment job under the deadlines of [793d2a Moment delivery](#793d2a-moment-delivery)): send the job to the runner (`roundDeadlineMs` per call); while the result is [`need`](#793d1b-terms), relay each request as one `$.model.complete` call in the model the request names (never the [session](../../GLOSSARY.md#session)'s model) with exactly the request's model, system, prompt and maxTokens, bounded by `judgeDeadlineMs`, and send the replies back with the [`state`](#793d1b-terms); stop at [`done`](#793d1b-terms). A relayed call MUST carry nothing but the request's fields, MUST pass the request's `model` through unchanged, and MUST end the job as an error when a request names none (it MUST NOT fall back to the session's model), and MUST NOT submit a prompt, start an agent or call a tool. A job that finishes without a request lets the queue continue. A job ends as an error when a reply is unanswered (`isAnswered` false), after `maxRounds` (4) [rounds](#793d1b-terms), or when a call fails or passes its deadline; a failed item MUST show a warning as in [793d2a Moment delivery](#793d2a-moment-delivery), naming the step (capture, backlog, index or curation), and MUST NOT remove the items after it; a failed scan job queues no backlog capture job. The mod MUST add the [`events`](#793d1b-terms) and [`notices`](#793d1b-terms) of every `done` to the counts since the previous summary, after the turn's items MUST show the line of [793d3f Memory-activity line](#793d3f-memory-activity-line) and nothing when the turn counted no `events` and produced no notice or warning; and it MUST NOT carry queued or running work into a later session (it registers no `session.end` handler and writes nothing to `$.store`). The mod's jobs and its `engram ingest --auto` run as any other engram command: they take the [vault write lock](../../specs/technical/vault-write-lock.md#792b8c1-exclusive-hold) and `.manifest.lock` as the command does and take no lock of their own, and a command that finds a lock held MUST wait for it ([792b8c1 Exclusive hold](../../specs/technical/vault-write-lock.md#792b8c1-exclusive-hold)); an `engram ingest --auto` that does not finish within `ingestDeadlineMs` because another process (an intentional ingest or learn in another [harness](../../GLOSSARY.md#harness), or another Claude Code session) holds `.manifest.lock` ends as a failed item with the warning of the index step, which leaves the index and the vault as they were (the ingest either completes or does not begin its reconcile), MUST NOT remove the items after it, and is retried by the next turn's `engram ingest --auto`.

Owner: Memory Mod ([793d3 Added: memory-mod](#793d3-added-memory-mod)).

Supports [793a5c1d A Pi agent's own ingest and learn run while the System ingests and captures](./793a_product-spec.md#793a5c1-uat-scenarios), [793a5c1e A lesson the Pi agent saved is not written again by the backlog](./793a_product-spec.md#793a5c1-uat-scenarios), [793a3a1p A note another harness's agent wrote is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1n A transcript is captured once](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1d Slow capture does not stall the session](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios), [793a4f1b A session of its own receives no delivery](./793a_product-spec.md#793a4f1-uat-scenarios), [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1i A backlog capture is counted](./793a_product-spec.md#793a5h1-uat-scenarios).

#### 793d2b1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2b1a The turn ends without waiting for the work
  Given the capture of a turn takes 20 seconds
  When Claude Code fires the end of the turn
  Then the handler returns before the capture is sent, and the capture starts from a $.clock.after callback
  And a prompt submitted immediately after is handled without waiting for it

Scenario: 793d2b1b A lesson is captured, then indexed, curated and summarized
  Given a turn whose capture writes one note and whose scan returns no source
  When the queue runs the turn's work
  Then the runner receives the capture job and the scan job, the mod runs engram ingest --auto, the runner receives the curation job, and the summary is shown
  And the summary counts one note added

Scenario: 793d2b1c Judgement requests are relayed until the job is done
  Given the runner answers the first round with a need and the second with another need
  When the mod runs the capture job
  Then it relays each request as one model completion in the model the request names, with the request's own fields only and no prompt submitted
  And it sends each reply back with the state it received, and the job ends with the runner's done

Scenario: 793d2b1k The mod passes the named model through
  Given the runner answers with a need whose requests name the models M1 and M2
  When the mod relays them
  Then the two completions are called with model M1 and model M2, and the session's model is not read

Scenario: 793d2b1m A request without a model ends the job
  Given the runner answers with a need whose request names no model
  When the mod relays it
  Then no completion is called, the job ends as an error and the warning names the step

Scenario: 793d2b1d A subagent's stop queues its capture
  Given a subagent stops
  When Claude Code fires its stop
  Then the handler returns at once and its capture job runs before the turn's items

Scenario Outline: 793d2b1f A job that needs no request lets the queue continue
  Given the runner answers the <job> job with done and no event because <reason>
  When the queue continues
  Then no model request is relayed, the next item runs, and a turn with no event, notice or warning shows no line
  Examples:
    | job      | reason                                |
    | capture  | the window is empty                   |
    | capture  | the reply holds no candidate          |
    | curation | no offer is pending                   |

Scenario Outline: 793d2b1h A failed step warns and the queue continues
  Given <failure> in the <step> of a turn
  When the queue continues
  Then the warning names the <step> and the reason, and the steps after it still run
  Examples:
    | step     | failure                                         |
    | capture  | a model reply is unanswered                     |
    | backlog  | the scan job throws                             |
    | backlog  | a backlog capture job's model reply is unanswered |
    | capture  | a need on every round up to maxRounds           |
    | capture  | no answer within roundDeadlineMs                |
    | index    | engram ingest --auto fails                      |
    | curation | the curation job throws                         |

Scenario: 793d2b1n The backlog follows the turn's own capture
  Given a turn whose capture writes one note, the main agent's transcript path from classic.SessionStart, and a scan that returns the source /pi/s1.jsonl
  When the queue runs the turn's work
  Then the runner receives, in order, the turn's capture job with the transcript path and the turn's end time, the scan job with the session id and the transcript path, a backlog capture job naming /pi/s1.jsonl, and the curation job
  And the mod runs engram ingest --auto after the backlog capture job and before the curation job
  And the summary counts the notes added by the turn's capture and by the backlog capture job together

Scenario: 793d2b1q A scan that returns no source queues no backlog capture job
  Given a scan job that finishes with no source
  When the queue continues
  Then no backlog capture job is sent and the next item runs

Scenario: 793d2b1r A failed scan queues no backlog capture job
  Given the scan job throws in a turn
  When the queue continues
  Then the warning names the backlog step and the reason, no backlog capture job is sent, and engram ingest --auto and the curation job still run

Scenario: 793d2b1s Ingest and capture coexist with an intentional ingest and learn in another harness
  Given a Pi agent running engram ingest --auto and the learn skill, and a turn whose work runs a capture job and engram ingest --auto
  When the two overlap
  Then each command waits for the lock the other holds and then proceeds
  And both complete, with every note readable and no note ID used twice
  And the mod took no lock of its own

Scenario: 793d2b1t An ingest that finds the manifest locked past its deadline is a warning
  Given another process holds .manifest.lock for longer than ingestDeadlineMs
  When the turn's engram ingest --auto runs
  Then the item ends with the warning naming the index step
  And the chunk index and the vault are unchanged by that attempt
  And the curation job and the summary still run
  And the next turn's engram ingest --auto runs and completes

Scenario: 793d2b1p The queue is not carried into a later session
  Given a capture is queued when the session ends
  When a new session starts
  Then the mod has registered no session.end handler and has written nothing to $.store
  And the new session's queue holds none of the old session's items
```

### 793d2e Job order within a round

Part of the flow spec of [793d2b Turn memory work](#793d2b-turn-memory-work) (`docs/specs/technical/turn-memory-work-flow.md`); its arrows are the runner's calls in that diagram.

Requirement: 793d2e The Auto Job Runner MUST run a capture [job](#793d1b-terms) in this order. In the first [round](#793d1b-terms) it calls Auto Capture on the [window](#793d1b-terms) and returns its extraction request as [`need`](#793d1b-terms). In the next round it calls Auto Capture with the extraction reply and returns the reconcile requests, one per candidate, as `need`. In the last round it calls Auto Capture with the reconcile replies, which makes the writes, and returns [`done`](#793d1b-terms) with the [`events`](#793d1b-terms) and [`notices`](#793d1b-terms) of every round. It MUST NOT call a component of a job before the component before it has returned. When the window is empty or holds no candidate it MUST return `done` with the `events` already counted and no `need`. It MUST run a curation job through Auto Curation alone, and return `done` with no `need` when no offer is pending. It MUST run a [scan job](#793d1b-terms) through Auto Capture alone and return `done` with the sources Auto Capture returns and no `need`, and it MUST run a [backlog capture job](#793d1b-terms) as it runs any capture job, over the window Auto Capture forms from the job's source.

Owner: Auto Job Runner ([793d4 Added: auto-job-runner](#793d4-added-auto-job-runner)).

Supports [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d2e1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2e1e Round one returns the extraction request
  Given a capture job whose window holds a candidate lesson
  When the runner runs round one
  Then Auto Capture is called on the window and no write is made
  And the output is a need holding the extraction request, with a state

Scenario: 793d2e1b The rounds follow the extraction, the reconcile requests and the writes
  Given an extraction reply with two candidates
  When the runner runs round two with the reply, and round three with two reconcile replies
  Then round two returns one need of two reconcile requests and calls no write
  And round three calls Auto Capture with both replies and returns a done with the events of all three rounds

Scenario Outline: 793d2e1c A job without work finishes without a request
  Given <job> whose <input>
  When the runner runs it
  Then it returns done with no need
  And no later component is called
  Examples:
    | job              | input                                            |
    | a capture job    | window is empty after the removals               |
    | a capture job    | extraction reply holds no candidate              |
    | a curation job   | vault holds no pending offer                     |
    | a scan job       | no source holds content beyond its watermark     |

Scenario: 793d2e1d A curation job calls Auto Curation alone
  Given a curation job and one pending offer
  When the runner runs it
  Then only Auto Curation is called and the output is a need holding the offer's request

Scenario: 793d2e1f A scan job calls Auto Capture alone
  Given a scan job and two sources that hold content beyond their watermarks
  When the runner runs it
  Then only Auto Capture is called and the output is a done holding the two paths and no need

Scenario: 793d2e1g A backlog capture job takes the rounds of a capture job
  Given a capture job naming a transcript source whose content after its watermark holds a candidate lesson
  When the runner runs round one, then round two with the extraction reply, then round three with the reconcile reply
  Then round one returns the extraction request, round two the reconcile request, and round three makes the write and returns a done with the events
```

### 793d2f Mod install

Canonical destination: `docs/specs/technical/mod-install-flow.md` (title `Mod Install Flow - Technical Spec`).

`engram update` deploys the mod with the other agent instructions, for the [harness](../../GLOSSARY.md#harness) that loads mods and for no other. As part of the Claude Code deploy the Claude Code version is checked, and when the mod cannot load on it the report says so. A preview writes nothing.

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant Upd as Self Installer
    participant Syn as Harness Deploy Sync
    participant Cli as Claude Code CLI

    Dev->>Upd: 793d2f engram update, optionally --dry-run
    Upd->>Syn: 792b29c1 detect the harnesses
    Syn-->>Upd: 792b29c1 the harnesses present
    Upd->>Syn: 793d2f, 793d10a the mod deploy for the harnesses that load mods
    opt Claude Code is present
        Syn->>Cli: 793d2f claude --version
        Cli-->>Syn: 793d2f the version, or that it cannot be run
    end
    alt a preview, or no harness that loads mods
        Syn-->>Upd: 793d10a the plan only, nothing written
    else the mod deploy fails
        Syn-->>Upd: 793d10a the failure
    else the mod is deployed
        alt claude is older than 2.1.287 or cannot be run
            Syn-->>Upd: 793d10a the mod deployed, adopted or left alone, with the row that it will not load
        else
            Syn-->>Upd: 793d10a the mod deployed, adopted or left alone
        end
    end
    Upd-->>Dev: 793d2f, 792b28c12 the report, with the mod's row where one applies
```

Requirement: 793d2f The Self Installer MUST, during an update, detect the [harnesses](../../GLOSSARY.md#harness), ask Harness Deploy Sync to deploy the mod after the [skills](../../GLOSSARY.md#skill) and guidance of the same harness and only for the harnesses that were detected, and include in the report the row Harness Deploy Sync returns for the mod, whether it says the mod was deployed, adopted, left alone or will not load. For a preview (`--dry-run`) it MUST ask for the plan and no write and MUST NOT change any other step of the update. When Harness Deploy Sync reports a failure of the mod deploy it MUST show the failure in the report and end the update with the update's failure status; otherwise the mod's deploy, including a row that says the mod will not load, MUST NOT change the exit status of the update.

Owner: Self Installer ([Self Installer](../../specs/technical/self-installer.md)).

Supports [793a4h1a Update installs the Integration for the harness it updates](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1c A preview installs none of the Integration](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1d One update serves both harnesses differently](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1l Running update twice changes nothing](./793a_product-spec.md#793a4h1-uat-scenarios), [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios), [793a5c1b A Pi session on a machine whose Claude Code has the Integration](./793a_product-spec.md#793a5c1-uat-scenarios).

#### 793d2f1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2f1a Claude Code gets the mod and Pi is not created
  Given ~/.claude exists and ~/.pi does not
  When the user runs engram update
  Then Harness Deploy Sync is asked to deploy the mod for Claude Code only
  And ~/.pi is not created
  And the report has a row for the mod

Scenario: 793d2f1b Both harnesses are updated and only Claude Code gets the mod
  Given ~/.claude and ~/.pi both exist
  When the user runs engram update
  Then Harness Deploy Sync is asked for the mod for Claude Code only
  And the report has no mod row for Pi

Scenario: 793d2f1c A harness that does not load mods gets no mod request
  Given ~/.pi exists and ~/.claude does not
  When the user runs engram update
  Then Harness Deploy Sync is asked for no mod deploy
  And the report has no mod row and the update's status is unchanged

Scenario: 793d2f1d A preview asks for the plan and writes nothing
  Given a machine with no mod
  When the user runs engram update --dry-run
  Then Harness Deploy Sync is asked for the plan only, and the other steps of the update are those of a preview
  And the report lists the mod as it would be deployed

Scenario: 793d2f1e A second update reports the mod unchanged
  Given an update has deployed the mod
  When the user runs engram update again
  Then the mod's row reports it unchanged and the update's status is unchanged

Scenario: 793d2f1f An existing install keeps what it had
  Given a Claude Code install with skills, guidance and shim, and an earlier copy of the mod
  When the user runs engram update
  Then the report names the mod as adopted or left alone
  And the rows of the skills, guidance and imports are the ones an update without the mod gives

Scenario: 793d2f1g A failed mod deploy is reported and fails the update
  Given Harness Deploy Sync reports a failure of the mod deploy
  When the update finishes
  Then the report shows the failure and the exit status is the update's failure status

Scenario Outline: 793d2f1h The report says when the mod will not load
  Given Harness Deploy Sync returns the mod's row for a Claude Code <claude>
  When the update finishes
  Then the report carries that row, which says the mod will not load
  And the exit status is the update's success status
  Examples:
    | claude                         |
    | older than 2.1.287             |
    | that cannot be run             |
```

## 793d3 Added: memory-mod

Canonical destination: `docs/specs/technical/memory-mod.md` (title `Memory Mod - Technical Spec`).

- Ownership: repository component, at the Claude Code mods boundary (the hooks module and the mods API it calls).
- Responsibility: turns Claude Code events into [System](./793a_product-spec.md#793a1-problem-statement) [jobs](#793d1b-terms) and job results into effects on Claude Code: [context](./793a_product-spec.md#793a1-problem-statement) for the agent, a status line for the user, background execution, the hold of a step's [delivery](./793a_product-spec.md#793a1-problem-statement) until the next tool result and the relay of model requests. It decides nothing about what a [moment](./793a_product-spec.md#793a1-problem-statement) is, what is delivered or what is written.
- Current realization (informational): `agent-instructions/mods/engram-memory/` (`.claude-plugin/plugin.json`, `hooks/hooks.json`, `hooks/register.js`, `tests/*.test.ts`).
- Qualification: stateable without naming a function; routing and gating are distinct from the Go components' decisions; `claude plugin test` tests it with the world beneath it stubbed; stable across Go refactors; the only way the System reaches a Claude Code [session](../../GLOSSARY.md#session).
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

Named constants (initial values, tuned under #850): `momentDeadlineMs` 25000, `momentJudgeDeadlineMs` 5000, `classifyWindowMessages` 6, `classifyWindowChars` 6000, `judgeDeadlineMs` 60000, `roundDeadlineMs` 120000, `ingestDeadlineMs` 300000, `maxRounds` 4.

### 793d3a Events, channels and pass-through

Requirement: 793d3a The Memory Mod MUST register handlers for the events of the table below and for no other event, MUST forward an event to the runner only under the condition shown, and MUST add a [delivery](./793a_product-spec.md#793a1-problem-statement) to Claude Code only through the channel shown, as one block separate from the prompt and from the tool result. Every handler MUST end by continuing the event it received (`next(e)`) with at most the channel's [context](./793a_product-spec.md#793a1-problem-statement) added, and every handler that can add context MUST have a `.catch` handler that records a failure after `next` as a warning. The `turn.step` handler is an async generator that MUST forward every chunk and return the result unchanged, rewrite neither `model` nor `effort`, read only the result's `answer` and `toolUses`, and return without waiting for any [job](#793d1b-terms). The mod MUST NOT register a handler that denies, drops or answers an event, and MUST NOT change a prompt's text, a tool call's input, a permission decision or a model, so that an [intentional memory](./793a_product-spec.md#793a1-problem-statement) command by the agent is neither touched nor undone.

| Event | Forwarded when | Channel |
| --- | --- | --- |
| `prompt.submit` | always, whatever the prompt's origin | an extra `context` entry after the prompt as typed |
| `classic.PostToolUse` | no `agent_id`, and the tool is `Skill`, `Agent`, `Edit`, `Write`, `MultiEdit` or `NotebookEdit` (a [job](#793d1b-terms) of its own), or a [held delivery](#793d1b-terms) waits (any tool) | `additionalContext` |
| `classic.PostToolUseFailure` | no `agent_id` | `additionalContext` |
| `turn.step` | no `agentId`; once its result is whole, when `answer` is not empty and `toolUses` is not | none: starts a [step job](#793d1b-terms) in the background ([793d2a Moment delivery](#793d2a-moment-delivery)); its delivery is held for the next tool result ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |
| `agent.spawn` | always | none: records the spawn ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |
| `classic.SubagentStart` | always | `additionalContext` |
| `classic.SubagentStop` | always | none: queues the [subagent](../../GLOSSARY.md#subagent)'s capture ([793d2b Turn memory work](#793d2b-turn-memory-work)) |
| `turn.start`, `turn.complete` | no `agentId` | none: reset the turn counters ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)); at `turn.complete`, drop a [held delivery](#793d1b-terms) and queue the turn's work ([793d2b Turn memory work](#793d2b-turn-memory-work)) |
| `session.compact` | always; acts on the main agent only | none: the [ledger](#793d1b-terms) rule of [793d3b State kept for the System](#793d3b-state-kept-for-the-system) |
| `classic.SessionStart` | always | none: records the main agent's `transcript_path` for the [jobs](#793d1b-terms) ([793d2b Turn memory work](#793d2b-turn-memory-work)); on `source` `resume` refills the [ledger](#793d1b-terms) ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2k1c An intentional recall by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2k1e An intentional engram query by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios).

#### 793d3a1 Contract Scenarios

```gherkin
Scenario: 793d3a1a The mod registers the listed events and none that can refuse
  Given the loaded mod and an agent that runs engram query and the learn skill in a turn
  When claude plugin validate lists its hooks and Claude Code fires the tool events of those commands
  Then the hooks are exactly the events of the table and no handler returns deny, drop or its own result in place of next(e)
  And each tool event of the agent's memory commands continues with its input, permission decision and result unchanged

Scenario Outline: 793d3a1b A delivery goes only through its channel
  Given the runner returns a delivery for a <event>, or a step delivery is held when it fires
  When the mod handles the event
  Then the text is added only as <channel>, and the prompt text, tool input and permission decision are the ones the event carried
  Examples:
    | event                      | channel                     |
    | prompt.submit              | one extra context entry     |
    | classic.PostToolUse        | one additionalContext entry |
    | classic.PostToolUseFailure | one additionalContext entry |
    | classic.SubagentStart      | one additionalContext entry |
    | classic.PostToolUse of a Read, a step delivery held | one additionalContext entry |

Scenario Outline: 793d3a1c Only the listed events are forwarded
  Given <event>
  When the mod handles it
  Then the runner <receives> a moment job for it
  Examples:
    | event                                      | receives         |
    | a Skill, Agent or Edit result, no agent_id | receives         |
    | a Read result, no agent_id                 | does not receive |
    | an Edit result with agent_id a1            | does not receive |
    | a prompt from the composer, a task notification, a scheduled trigger or a peer session | receives |
    | a step result with text and a tool call, no agentId | receives |
    | a step result with text and no tool call            | does not receive |
    | a step result with a tool call and no text          | does not receive |
    | a step result with agentId a1                       | does not receive |

Scenario: 793d3a1f A skipped handler is caught
  Given a handler that adds context fails after calling next
  When Claude Code runs its catch handler
  Then the event result stands and a warning is recorded

Scenario: 793d3a1g The step handler changes nothing about the step
  Given a main-agent step whose response streams text and a tool call
  When the mod's turn.step handler runs
  Then it forwards every chunk and returns the result unchanged, rewrites neither model nor effort, and returns without waiting for the step job
```

### 793d3b State kept for the System

Requirement: 793d3b The Memory Mod MUST keep, in module variables: a [ledger](#793d1b-terms) of the item ids delivered, for each agent and keyed by `$.session.id()`, so that `/clear`, `/resume` and `/branch` give a new [session](../../GLOSSARY.md#session) an empty ledger; for the main agent the number of edit-tool calls and of failure lookups in the current turn, zeroed at each `turn.start` (the edit count raised for every edit-tool result of the main agent after its [job](#793d1b-terms) is built, the failure count for every failure job sent); for the main agent, at most one step job in flight and at most one [held delivery](#793d1b-terms) (its text, its ids and the time its job began), both cleared at `turn.complete`; and for each `agent.spawn` its task prompt, description and fork flag, until the matching `classic.SubagentStart`, matched by the description `$.agent.list()` reports for the starting agent and otherwise by spawn order. A [moment](./793a_product-spec.md#793a1-problem-statement) job MUST carry the receiving agent's [context](./793a_product-spec.md#793a1-problem-statement) as a list of texts, its ledger, the [task](#793d1b-terms) and, for the main agent, the [recent texts](#793d1b-terms): the `text` of the last `classifyWindowMessages` messages of `$.session.messages()` that have one (user and assistant alike, no tool result), the oldest cut first to `classifyWindowChars` characters in all, and for a step job the step's `answer` as the newest when the newest assistant message does not already end with it. For the main agent the context is the text of every message of `$.session.messages()` and of every tool result in them, and the task is the latest human-origin prompt of at least `taskMinChars` characters, else the current prompt. For a starting [subagent](../../GLOSSARY.md#subagent) the context and the task are its task prompt, and a fork's context adds the main agent's texts. On `session.compact` without `agentId` the mod MUST empty the main ledger and refill it with the ids of the [delivery](./793a_product-spec.md#793a1-problem-statement) blocks (from the line `[engram memory]` to the line `[end engram memory]`) found in `$.session.messages()`; `session.compact` with an `agentId` MUST leave the main ledger untouched; and on `classic.SessionStart` with `source` `resume` the mod MUST refill the ledger the same way.

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios).

#### 793d3b1 Contract Scenarios

```gherkin
Scenario: 793d3b1a A delivered id is in the ledger of its agent and session
  Given a delivery of note X to the main agent of session S1
  When the next moment job for the main agent of S1 is built
  Then its ledger holds X, a subagent's ledger does not, and the first job after /clear, with a new session id, carries an empty ledger

Scenario: 793d3b1b The turn counters reset at the start of a turn
  Given the main agent made one edit and one failure lookup in a turn
  When Claude Code fires turn.start
  Then the next moment job carries zero edits and zero failure lookups

Scenario Outline: 793d3b1c A subagent start is matched to its spawn
  Given two agent.spawn events with descriptions "add tests" and "fix lint", and <listing>
  When classic.SubagentStart fires for the second agent started
  Then the moment job carries the task prompt of the <spawn> spawn
  Examples:
    | listing                                                    | spawn     |
    | $.agent.list() describes the starting agent as "fix lint"  | fix lint  |
    | $.agent.list() gives no description for it                 | add tests |

Scenario: 793d3b1d A subagent's context is its task, and a fork's is the parent's too
  Given a spawn with fork false and one with fork true
  When each subagent starts
  Then the first job's context is the task prompt alone and the second job's is the task prompt and the main agent's texts

Scenario Outline: 793d3b1e The ledger follows compaction and resume
  Given the main ledger holds notes X and Y, and only Y's delivery block remains in $.session.messages()
  When <event> fires
  Then the main ledger is <ledger>
  Examples:
    | event                                      | ledger  |
    | session.compact with no agentId            | Y alone |
    | session.compact with agentId a1            | X and Y |
    | classic.SessionStart, source resume, in a new process | Y alone |

Scenario Outline: 793d3b1f A held step delivery is added once or dropped
  Given a step job finished with a delivery of note X, held, and <situation>
  When <event>
  Then <outcome>
  Examples:
    | situation                                              | event                                  | outcome |
    | the next tool result forms no job of its own           | a Read result of the main agent fires  | the Read result continues with X as one additionalContext block, X is in the main ledger and nothing is held |
    | the next tool result forms a job of its own            | an Edit result with edits 0 fires      | the Edit's own job runs and X stays held for the first later result that forms no job of its own |
    | the turn ends first, or while the job still ran        | turn.complete fires                    | X is dropped, nothing is added to any event and the delivered count does not rise |
    | X entered the ledger after the step job began          | a Read result of the main agent fires  | X is dropped and nothing is added |
    | more than momentDeadlineMs passed before the job ended | the job finishes                       | X is dropped and nothing is added |

Scenario: 793d3b1g The recent texts are the newest user and assistant texts, bounded
  Given ten earlier user and assistant messages whose texts total more than classifyWindowChars, tool results among them, and a step whose answer is not yet in $.session.messages()
  When the mod builds the main agent's step job
  Then the recent texts hold at most classifyWindowMessages texts, no tool result, the oldest cut first to classifyWindowChars characters in all, and the step's answer as the newest
```

### 793d3f Memory-activity line

Requirement: 793d3f The Memory Mod MUST keep, since the previous summary, the sum over the main agent, all [subagents](../../GLOSSARY.md#subagent) and the [backlog capture jobs](#793d1b-terms) of the [`events`](#793d1b-terms) of every [job](#793d1b-terms) (`delivered`, `added`, `amended`, `curated`), the [`notices`](#793d1b-terms) and the warnings recorded under [793d2a Moment delivery](#793d2a-moment-delivery) and [793d2b Turn memory work](#793d2b-turn-memory-work), each work item's counts added once (a step job's counts are added when its held delivery is added, and not when it is dropped). After a top-level turn's work has finished it MUST set the status line, with `$.ui.status`, to one line: the non-zero counts in the order delivered, added, amended, curated as `engram memory: 3 items delivered, 1 note added, 1 note amended, 2 offers curated`, then each notice, then each warning as `engram memory skipped (<step>): <unavailable|timed out|error>`; or clear it with `undefined` when there is no count, notice or warning; and then MUST reset the sums. It MUST NOT write the line with `$.ui.log`, in a prompt, in a [context](./793a_product-spec.md#793a1-problem-statement) channel or in any result Claude reads.

Supports [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios), [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1b Nothing is shown after a turn without events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1c The summary is not in the agent's context](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1e The counts cover the events since the previous summary](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1i A backlog capture is counted](./793a_product-spec.md#793a5h1-uat-scenarios).

#### 793d3f1 Contract Scenarios

```gherkin
Scenario: 793d3f1a The line shows the counts of a turn with events
  Given jobs of a turn and its subagents reported 3 delivered, 1 added, 1 amended and 2 curated
  When the turn's work has finished
  Then the status line is "engram memory: 3 items delivered, 1 note added, 1 note amended, 2 offers curated"

Scenario: 793d3f1b The line is cleared without events and counts only what is new
  Given a summary was shown after one turn, and the next turn reports no event, notice or warning
  When that turn's work has finished
  Then the status line is cleared with undefined
  And after a later turn that reports one note added the line is "engram memory: 1 note added"

Scenario: 793d3f1c The line is not in the agent's context
  Given a line was shown after a turn
  When the next prompt is handled
  Then no context entry or tool result added by the mod contains the line, and $.ui.log is never called with it

Scenario: 793d3f1g A backlog capture's note is counted
  Given a turn whose backlog capture job reported 1 added and whose other jobs reported nothing
  When the turn's work has finished
  Then the status line is "engram memory: 1 note added"

Scenario Outline: 793d3f1f A notice or a warning is shown with the counts
  Given a turn with <counts> and <extra>
  When the turn's work has finished
  Then the line is "<line>"
  Examples:
    | counts         | extra                                                      | line                                                                       |
    | no count       | the notice "note saved, offer queued (parent unreachable)" | engram memory: note saved, offer queued (parent unreachable)               |
    | one note added | a timed-out curation                                       | engram memory: 1 note added; engram memory skipped (curation): timed out   |
```

### 793d3g Validated and tested through the existing checks

Requirement: 793d3g The Memory Mod MUST be validated and tested by one Go integration test, selected by the `integration` build tag so that it runs under the existing integration gates (`test-integration`, and `test-dev-integration` when the test lives in the dev tree) like the other tests that need an external tool. The test MUST run `claude plugin validate` and `claude plugin test` on the mod source (`agent-instructions/mods/engram-memory/`), the second running the mod's own `*.test.ts` files, and MUST fail, not skip, when any of them fails or when the Claude Code CLI is unavailable. It MUST NOT add a check target or change a dependency of any existing target.

Supports no product scenario: it is a development gate, and the product scenarios the mod realizes are supported by the requirements above that the `*.test.ts` files exercise. Decided in [793b236 The mod is validated and tested through the existing integration gate, not a new check target](./793b_decision-log.md#793b236-the-mod-is-validated-and-tested-through-the-existing-integration-gate-not-a-new-check-target).

#### 793d3g1 Contract Scenarios

```gherkin
Scenario: 793d3g1a Validation passes on the shipped mod
  Given the mod source in agent-instructions/mods/engram-memory/ and the Claude Code CLI installed
  When the integration test runs
  Then claude plugin validate and claude plugin test both succeed and the test passes

Scenario: 793d3g1b A failing mod test fails the integration test
  Given one of the mod's *.test.ts files fails under claude plugin test
  When the integration test runs
  Then the integration test fails and its output carries the mod test's failure

Scenario: 793d3g1c A missing CLI fails the integration test, not skips it
  Given the Claude Code CLI is not installed
  When the integration test runs
  Then it fails with a message naming the missing CLI and is not reported as skipped
```

## 793d4 Added: auto-job-runner

Canonical destination: `docs/specs/technical/auto-job-runner.md` (title `Auto Job Runner - Technical Spec`).

- Ownership: repository component at the command boundary (standard input and output of `engram auto run`); the caller of [793d6 Added: memory-delivery](#793d6-added-memory-delivery), [793d7 Added: auto-capture](#793d7-added-auto-capture) and [793d9 Added: auto-curation](#793d9-added-auto-curation).
- Responsibility: turns one invocation into one [round](#793d1b-terms) of one [job](#793d1b-terms): runs the components the job needs up to the next model request or to the end, and writes either the requests to relay with an opaque state, or the finished result. It keeps nothing between calls and reports a failure inside its result.
- Current realization (informational): `internal/auto/runner.go`; the command in `internal/cli/auto.go`, built by `cli.NewDeps` over `cli.Deps.Stdin`; no new `cli.Primitives` field.
- Qualification: stateable without naming a function; round sequencing, state and failure reporting are distinct from the components; tested over fakes of them and a fixed job document; stable across component changes; the only way the mod reaches the Go side.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d4a A job in rounds

Requirement: 793d4a The Auto Job Runner MUST read one JSON document from standard input holding `job` (`moment`, `capture`, `scan` or `curate`), the [job](#793d1b-terms)'s input and, from the second [round](#793d1b-terms), `answers` and [`state`](#793d1b-terms), run the job for one round, and write one JSON document and nothing else to standard output and exit 0. A `moment` job's input is the event, the receiving agent's [context](./793a_product-spec.md#793a1-problem-statement), its [ledger](#793d1b-terms), the [task](#793d1b-terms) and, for the main agent, the [recent texts](#793d1b-terms); a `capture` job's is the [window](#793d1b-terms) of messages (a turn) with the [session](../../GLOSSARY.md#session) id, the `transcriptPath` when the mod has it and `turnEndedAt`, or the [subagent](../../GLOSSARY.md#subagent)'s [transcript](../../GLOSSARY.md#transcript) path and last assistant message with the session id, or one [transcript source](./793a_product-spec.md#793a1-problem-statement)'s path (a [backlog capture job](#793d1b-terms)); a `scan` job's is the session id and the `transcriptPath` of the session it runs in. Every request in [`need`](#793d1b-terms) MUST name a `model`: the value [`judgeTiers`](#793d1b-terms) holds for its kind. `judgeTiers` is one configuration table from a request kind (`extraction`, which also judges the conclusions, `reconcile`, `curation`, which also authors a skill-runbook [note](./793a_product-spec.md#793a1-problem-statement), `classify`, which names the decision [moments](./793a_product-spec.md#793a1-problem-statement) the agent is about to reach, and `relevance`, which keeps the candidates that bear on a moment) to a model, an alias such as `haiku` or a full id, as `$.model.complete` accepts (row 18 of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods)); the default for every kind is the cheapest tier available to the session (the `haiku` family), and no dated id is written in this spec. A kind is raised only on measured evidence that the cheapest tier fails that kind (#850), and one kind's value changes only that kind's requests. The runner passes the table to the components that build requests. The document MUST be `{ "need": [ { id, model, prompt, system?, maxTokens } ], "state": "<opaque>" }` when the job needs model replies; `{ "done": { deliver, deliveredIds, moment, events, notices } }` when a `moment`, `capture` or `curate` job is finished (`moment` holding the kinds of the moments delivered for), or `{ "done": { "sources": [ <path> ], events, notices } }` when a [scan job](#793d1b-terms) is finished, with [`events`](#793d1b-terms) the integer counts `delivered`, `added`, `amended` and `curated`; or `{ "done": { "warning": { "why", "detail" }, "events", "notices" } }` when a step failed, with `why` `unavailable` when the [vault](./793a_product-spec.md#793a1-problem-statement) cannot be read and `error` otherwise, and the `events` and [`notices`](#793d1b-terms) of the steps that completed before the failure (a note written before a failure is still counted). A `scan` job MUST finish in one round. A `moment` job MUST take the rounds its [cue](#793d1b-terms) and its candidates need, and no others: for a [judged cue](#793d1b-terms) round one returns the `classify` request as `need`, and the round that reads its reply goes on to the lookup; for a cue that names its moment round one goes straight to the lookup; the lookup returns one `relevance` request as `need` when at least one candidate remains after the present items are removed, and otherwise finishes the job with no [delivery](./793a_product-spec.md#793a1-problem-statement); and the round that reads the relevance reply returns `done` with the delivery of the items it names. A classification that names no moment finishes the job in the round that reads its reply, with no query. A moment job therefore takes one to three rounds, within `maxRounds`. A reply the runner cannot read as the JSON its request asks for MUST end the job as a `done` with a warning `error`. `state` MUST hold everything the next round needs, the `events` already counted included; the runner MUST keep nothing between calls. It MUST exit non-zero, with the reason on standard error and nothing on standard output, only when its input is not a valid job document.

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d4a1 Contract Scenarios

```gherkin
Scenario Outline: 793d4a1a A moment job with nothing to judge finishes in one round
  Given a moment job whose event <moment>
  When the runner runs it
  Then the output is one JSON document with no need and a done with <result>, and the exit status is 0
  Examples:
    | moment                                                                              | result                     |
    | is no cue                                                                           | deliver "" and zero events |
    | is a cue that names its moment and whose lookup returns only items already present | deliver "" and zero events |

Scenario: 793d4a1h Every request names the model of its kind
  Given a capture job whose window holds a candidate and the default judgeTiers
  When the runner runs round one and round two
  Then each request in each need carries a model, and it is the model judgeTiers gives for that request's kind

Scenario: 793d4a1c A capture job resumes from its state alone and reports its events
  Given a capture job whose window holds a candidate lesson
  When the runner runs round one and then round two with the answers and the returned state
  Then round one returns a need with a state
  And a second process with no memory of round one gives the same round-two result, with events.added 1 and the other counts 0

Scenario Outline: 793d4a1f A failure is reported in the result
  Given <failure>
  When the runner runs the round
  Then the output is a done with warning why "<why>"<events> and exit status 0
  Examples:
    | failure                                                    | why         | events             |
    | a moment job and a vault that cannot be read               | unavailable |                    |
    | a capture job whose first note is written and second write fails | error | and events.added 1 |

Scenario: 793d4a1g An invalid job document is refused
  Given standard input that is not a job document
  When the runner runs
  Then it exits non-zero with the reason on standard error and writes nothing to standard output

Scenario: 793d4a1i A scan job finishes in one round and returns paths
  Given a scan job for session S with transcript path P, and Auto Capture returning the sources /pi/s1.jsonl and /pi/s2.jsonl
  When the runner runs it
  Then the output is one JSON document with a done holding the sources /pi/s1.jsonl and /pi/s2.jsonl and no need, and the exit status is 0

Scenario: 793d4a1j A backlog capture job names its source and resumes from its state alone
  Given a capture job whose input is the path of a transcript source holding a candidate lesson after its watermark
  When the runner runs round one and then round two with the answers and the returned state
  Then round one returns a need with a state
  And a second process with no memory of round one gives the same round-two result, with events.added 1

Scenario: 793d4a1k A cue that names its moment takes two rounds when a candidate remains
  Given a moment job for a prompt, and a lookup that returns note X, not present
  When the runner runs round one and then round two with the relevance reply naming X
  Then round one returns a need holding exactly one relevance request and a state
  And round two returns a done with the delivery text of X, the id of X and the moment prompt

Scenario: 793d4a1l A judged cue takes three rounds in order and resumes from its state alone
  Given a moment job for the first failed tool result of a turn, a classify reply naming declare-done and a relevance reply naming note X
  When the runner runs round one, round two with the classify reply and round three with the relevance reply
  Then round one returns a need holding exactly one classify request
  And round two returns a need holding exactly one relevance request, after one query
  And round three returns a done with X delivered
  And a second process with no memory of the earlier rounds gives the same round-two and round-three results from the state alone

Scenario: 793d4a1m A classification that names no moment ends the job without a query
  Given a moment job for a judged cue and a classify reply naming no moment
  When the runner runs the round that reads the reply
  Then the output is a done with deliver "" and zero events, no query was made and no relevance request was returned

Scenario: 793d4a1n Both delivery judgements name the cheapest tier by default
  Given a moment job and the default judgeTiers
  When the runner returns its classify request and its relevance request
  Then each carries a model, and it is the cheapest tier (the haiku family) that judgeTiers gives for its kind

Scenario: 793d4a1o A raised classify tier changes only classify requests
  Given judgeTiers sets classify to "sonnet" and leaves every other kind at its default
  When the runner returns a classify request, a relevance request, an extraction request, a reconcile request and a curation request
  Then the classify request names "sonnet" and each of the others names its default model

Scenario Outline: 793d4a1p An unreadable judgement reply is reported in the result
  Given a <kind> reply that is not the JSON its request asks for
  When the runner runs the round that reads it
  Then the output is a done with warning why "error", no delivery and exit status 0
  Examples:
    | kind      |
    | classify  |
    | relevance |
```

## 793d6 Added: memory-delivery

Canonical destination: `docs/specs/technical/memory-delivery.md` (title `Memory Delivery - Technical Spec`).

- Ownership: repository component; the caller of the [Query Engine](../../specs/technical/query-engine.md).
- Responsibility: decides from one event, with a classification relayed to a model where the event does not name its [moment](./793a_product-spec.md#793a1-problem-statement), whether it is a [moment](./793a_product-spec.md#793a1-problem-statement) and what to ask the [vault](./793a_product-spec.md#793a1-problem-statement), asks once, drops what the receiving agent holds, keeps the items a relevance judgement says bear on the moment, and writes the text that names each item's source and asks the agent to apply it.
- Current realization (informational): `internal/auto/moment.go` and `internal/auto/deliver.go`, over the query path `runLocalQueryPayload` and the payload type of `internal/cli/merged_query.go`.
- Qualification: stateable without naming a function; [cues](#793d1b-terms), relevance, presence and wording are distinct from the engine's matching and the runner's sequencing; tested as a pure decision over a fake query and scripted judgement replies; stable across engine payload changes; no [delivery](./793a_product-spec.md#793a1-problem-statement) without it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d6a Moments and lookup

Requirement: 793d6a The Memory Delivery MUST treat the events of the table below as [cues](#793d1b-terms), and no other event (an empty prompt, an edit when the turn's edit count is above 0, a failure when its failure-lookup count is above 0, a step of the main agent whose response has no text or no tool call, any event of a [subagent](../../GLOSSARY.md#subagent) other than its start, a `Skill` other than `please` and `route`, any other tool result) as one: a cue decides when to look, and the [moment](./793a_product-spec.md#793a1-problem-statement) is then decided as follows. A cue whose event names its moment (`please`, `prompt`, `dispatch`, `subagent`) gives that moment with no judgement. For a [judged cue](#793d1b-terms) (the `judged` rows) it MUST first build one `classify` [judgement request](#793d1b-terms) over the [recent texts](#793d1b-terms), with the event's own text for an edit or a failure, naming the model [`judgeTiers`](#793d1b-terms) gives for `classify`, and asking whether the agent is about to endorse or rank an approach (`endorse`), declare work done (`declare-done`), diagnose a failure it cannot explain (`diagnose`) or start building a new approach (`build`), the four decision moments of [792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments), and on what; it MUST read the reply as JSON naming zero or more of these moments, each with a one-line subject, and end the [job](#793d1b-terms) as an error when it cannot; and when the reply names no moment it MUST query nothing and deliver nothing. For a moment it MUST ask the Query Engine once, whatever the number of moments named, with the query text (cut to `textLimit`) as `--text`, one phrase per phrase (each cut to `phraseLimit`), the [recency channel](../../GLOSSARY.md#recency-channel) off and chunk content included. The phrases are the query text and, for the `prompt` moment, one phrase for each of recall's four decision moments built from the job's [task](#793d1b-terms) (the approximation, kept for the prompt alone because a prompt starts a turn and nothing has yet been said for a judgement to read); for `please`, `dispatch` and `subagent`, one phrase naming its situation; for a judged cue, one phrase for each named moment built from its kind and its subject, the query text being the named subjects joined (shapes in the implementation plan). It MUST keep, in the engine's order, the items whose provenances include `direct`, `cluster_rep` or `trigger` and whose score is at least `deliveryMinScore` (the score nominates items; it does not decide that they bear on the moment), remove the items present under [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text), keep at most `deliveryItemLimit` of what remains, pass what remains to the relevance filter of [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text), and write nothing to the [vault](./793a_product-spec.md#793a1-problem-statement).

| [Cue](#793d1b-terms) (event) | [Moment](./793a_product-spec.md#793a1-problem-statement) | Query text |
| --- | --- | --- |
| a prompt that, trimmed, starts with `/please` | `please` | the text after `/please` |
| a prompt of any other text or origin | `prompt` | the prompt |
| a `Skill` result for `please` | `please` | the Skill's args |
| a `Skill` result for `route`, or an `Agent` result, by the main agent | `dispatch` | the description and the first 500 characters of the prompt or args |
| a [subagent](../../GLOSSARY.md#subagent) start | `subagent` | the task prompt |
| the first edit-tool result of a turn by the main agent | judged | the subjects of the [moments](./793a_product-spec.md#793a1-problem-statement) the classification names; the file path and the first 500 characters of the new text go to the `classify` request |
| the first failed tool result of a turn by the main agent | judged | the same; the first 500 characters of the error go to the `classify` request |
| a step of the main agent whose response has text and at least one tool call | judged | the same; the step's text is the newest of the recent texts |

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios).

#### 793d6a1 Contract Scenarios

```gherkin
Scenario: 793d6a1a One query with the moment's text and phrases, and no write
  Given a moment with a query text and three phrases
  When the delivery looks up
  Then the Query Engine receives one query with that text, those phrases, the recency channel off and chunk content on, and no vault write is requested

Scenario: 793d6a1b Only nominated items are kept, and the limit applies after present items are removed
  Given the engine returns a direct note at 0.6, a recent-only chunk at 0.9, an explore note at 0.5, a direct note at 0.1 and twelve more nominated items of which three are present
  When the delivery filters
  Then the direct note at 0.6 and the nominated items remain, none of them present, at most deliveryItemLimit in all, and they pass to the relevance filter

Scenario Outline: 793d6a1e A cue gives its moment
  Given <event>
  When the delivery decides
  Then the moment is <moment> (judged: decided by the classification) and the query text is <text>
  Examples:
    | event                                                        | moment   | text                      |
    | the prompt "/please add a flag to the cli"                   | please   | "add a flag to the cli"   |
    | the prompt "rename package foo to bar" from a peer session   | prompt   | the prompt                |
    | a Skill result for route, args "add tests for internal/cli", by the main agent | dispatch | the description and the args |
    | a subagent start, task prompt "add tests for internal/cli"   | subagent | the task prompt           |
    | an edit result with edits 0                                  | judged   | the named subjects        |
    | a failure with failureLookups 0                              | judged   | the named subjects        |
    | a step with the text "tests pass, now the rename" and a tool call | judged | the named subjects      |

Scenario: 793d6a1f A prompt's phrases follow the task and long text is cut
  Given the prompt "ok, do that" after an earlier human prompt "rename package foo to bar" of at least taskMinChars characters, and another prompt of 5000 characters
  When the delivery decides for each
  Then the first prompt's phrases include one each for endorsing, declaring done, diagnosing a failure and starting to build, built from the earlier prompt
  And the long prompt's query text is textLimit characters and every phrase is at most phraseLimit

Scenario Outline: 793d6a1g Events outside the table are not moments
  Given <event>
  When the delivery decides
  Then there is no moment
  Examples:
    | event                                                  |
    | an empty prompt, or a Read result                      |
    | an edit result with edits 1, or a failure with failureLookups 1 |
    | a Skill result for another skill, or an Edit result by a subagent |
    | a step with text and no tool call, or a step with agentId a1   |

Scenario: 793d6a1h A judged cue builds one classify request over the recent texts
  Given the main agent's first failed tool result of a turn, and ten earlier user and assistant texts totalling more than classifyWindowChars
  When the delivery decides
  Then it builds one classify request naming the model judgeTiers gives for classify
  And the request holds the first 500 characters of the error and the newest recent texts, the oldest cut first to classifyWindowChars
  And it names the four moments endorse, declare-done, diagnose and build and asks on what

Scenario: 793d6a1i A named moment's subject forms the query
  Given the classification names declare-done with the subject "declaring the Go package rename done"
  When the delivery asks the Query Engine
  Then it receives one query with that subject as its text and one phrase built from declare-done and the subject, and not the four phrases of a prompt

Scenario: 793d6a1j No moment named, no query
  Given the classification names no moment
  When the delivery decides
  Then the Query Engine receives no query and the result is done with deliver "" and zero events

Scenario: 793d6a1k An unreadable classification ends the job as an error
  Given the classify reply is not a JSON document
  When the delivery reads it
  Then the job ends as an error with warning why "error", the Query Engine received no query and nothing is delivered

Scenario: 793d6a1l Two named moments make one query
  Given the classification names diagnose with the subject "go test fails on the nilaway check" and build with the subject "adding a flag to the cli"
  When the delivery asks the Query Engine
  Then it receives one query whose text joins both subjects and whose phrases include one for each moment
```

### 793d6b Presence, relevance and delivery text

Requirement: 793d6b The Memory Delivery MUST treat an item as present when its [note](./793a_product-spec.md#793a1-problem-statement) file name or its chunk id occurs in any text of the receiving agent's [context](./793a_product-spec.md#793a1-problem-statement) or equals an id in the [ledger](#793d1b-terms) it was given, and as not present when it occurs in neither (an item delivered, retrieved or restored whose text has left the context is not present, and a parent's item is not present for a [subagent](../../GLOSSARY.md#subagent) whose context is its task prompt). When at least one item remains after the present items are removed and the limit applied ([793d6a Moments and lookup](#793d6a-moments-and-lookup)) it MUST build one `relevance` [judgement request](#793d1b-terms), naming the model [`judgeTiers`](#793d1b-terms) gives for `relevance`, that gives the [moment](./793a_product-spec.md#793a1-problem-statement) (for a [judged cue](#793d1b-terms) the named moments with their subjects, otherwise the moment and its query text) and each remaining item (its id, its kind and its content cut to `relevanceItemChars`), and asks which items bear on the moment, judged from the content and not from the score, as [792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster) judges coverage; when no item remains it MUST build no request. It MUST read the reply as JSON naming item ids, end the [job](#793d1b-terms) as an error when it cannot, ignore a named id that is not among the candidates, and drop every item the reply does not name. When at least one item remains it MUST return a text whose first line is `[engram memory]` followed by the instruction `Engram surfaced the vault items below for this moment. Apply what bears on your next step; disregard the rest.`, then each item as a line `--- note <file name> (<kind>)` or `--- chunk <id> (score <score to two decimals>)` followed by its content, and a last line `[end engram memory]`, MUST return the items' ids and MUST set `events.delivered` to their number. The text MUST NOT ask the agent to run a command, to evaluate, approve or write a note, or to curate. When no item remains it MUST return the empty text, no ids and no instruction.

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2d1a Delivered content names its source](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1b A delivered chunk names its id](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios).

#### 793d6b1 Contract Scenarios

```gherkin
Scenario Outline: 793d6b1a An item is delivered only when it is not present
  Given <situation>
  When the item comes back from the query
  Then it is <outcome>
  Examples:
    | situation                                                                         | outcome         |
    | the ledger holds note X and the context texts do not mention it                   | not delivered   |
    | a tool result in the context shows note X by its file name                        | not delivered   |
    | the context shows the chunk id "session-ab12#turn-7"                              | not delivered   |
    | X is in the main agent's texts and the subagent's context is its task prompt      | delivered       |
    | X was delivered, the context was compacted, and neither ledger nor texts hold X   | delivered again |

Scenario Outline: 793d6b1f The text names each source, or there is none
  Given <items> remain after the present items are removed
  When the text is written
  Then the text is <text>, events.delivered is <count> and the ids are <ids>
  Examples:
    | items | text | count | ids |
    | the feedback note "474.2026-07-25.charge-one-reviewer-to-execute….md" and the chunk "session-ab12#turn-7" at score 0.614 | "[engram memory]", the instruction, "--- note 474.2026-07-25.charge-one-reviewer-to-execute….md (feedback)" and its content, "--- chunk session-ab12#turn-7 (score 0.61)" and its content, "[end engram memory]", and no request to run a command, evaluate, approve, write or curate | 2 | the file name and the chunk id |
    | no item | empty, with no instruction | 0 | none |

Scenario: 793d6b1g Relevance drops a cosine-high item that does not bear on the moment
  Given a lookup for the moment declare-done with the subject "declaring the Go package rename done" that returns note A (score 0.9, about naming Go test files) and note B (score 0.5, about declaring a rename done), neither present
  And the relevance reply names B only
  When the delivery writes the text
  Then the text holds B and not A, the ids are B's alone and events.delivered is 1

Scenario Outline: 793d6b1h Nothing to judge or nothing named delivers nothing
  Given <situation>
  When the delivery reaches the relevance filter
  Then <request> and the result is the empty text, no ids and events.delivered 0
  Examples:
    | situation                                                         | request                      |
    | every item the lookup returned is present                         | no relevance request is built |
    | the lookup returned no item                                       | no relevance request is built |
    | two candidates and a reply naming neither, or naming an id that is not a candidate | one relevance request was built |

Scenario: 793d6b1i A relevance request names its model, the moment and every candidate
  Given two candidates that are not present and the default judgeTiers
  When the delivery builds the relevance request
  Then it names the model judgeTiers gives for relevance, the moment with its subject, and both candidates' ids, kinds and content cut to relevanceItemChars

Scenario: 793d6b1j An unreadable relevance reply ends the job as an error
  Given the relevance reply is not a JSON document
  When the delivery reads it
  Then the job ends as an error with warning why "error" and nothing is delivered
```

## 793d7 Added: auto-capture

Canonical destination: `docs/specs/technical/auto-capture.md` (title `Auto Capture - Technical Spec`).

- Ownership: repository component; the caller of the [Query Engine](../../specs/technical/query-engine.md), [Note Authoring](../../specs/technical/note-authoring.md), [Note Amendment](../../specs/technical/note-amendment.md) and, for a [transcript](../../GLOSSARY.md#transcript), the [Transcript Reader](../../specs/technical/transcript-reader.md), the [Ingestion Source Discovery](../../specs/technical/ingestion-source-discovery.md) and the [Durable File Store](../../specs/technical/durable-file-store.md).
- Responsibility: finds the [lessons](./793a_product-spec.md#793a1-problem-statement) and sound conclusions in one turn, one [subagent](../../GLOSSARY.md#subagent) [session](../../GLOSSARY.md#session) or one other session's [transcript source](./793a_product-spec.md#793a1-problem-statement) and records those the [vault](./793a_product-spec.md#793a1-problem-statement) does not hold: finds the [transcript](../../GLOSSARY.md#transcript) sources that hold content beyond their [watermark](#793d1b-terms), forms the [window](#793d1b-terms), removes secrets and the [System](./793a_product-spec.md#793a1-problem-statement)'s own blocks, asks a model for candidates, reconciles each against the vault with a second request, writes the absent ones with their provenance, and keeps each transcript source's watermark.
- Current realization (informational): `internal/auto/capture.go`, `internal/auto/redact.go`, `internal/auto/backlog.go`; the criteria are the learn [skill](../../GLOSSARY.md#skill)'s Step 2, the recall skill's Step 4 and its derived-note rules, from `agent-instructions/skills/*/SKILL.md` embedded at build time.
- Qualification: stateable without naming a function; window, redaction, parsing, reconciliation and the write are distinct from the runner's [rounds](#793d1b-terms) and Note Authoring; tested over scripted replies, a fake query and a fake writer; stable across model or prompt changes; no lesson is captured without it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

Named constants (initial values, tuned under #850): `backlogPerTurn` 1, `backlogQuietMs` 900000, `backlogWindowBytes` 100000.

### 793d7a Judging a window

Requirement: 793d7a The Auto Capture MUST judge a [window](#793d1b-terms) with two kinds of model request, each naming the model that [793d4a A job in rounds](#793d4a-a-job-in-rounds) gives for its kind (`extraction`, then `reconcile`) and neither asking the model to run anything: one extraction request per window, then one reconcile request per candidate. A main-agent turn's window is the one the mod took from `$.session.messages()` at the turn's end (the last assistant message before the turn's last human prompt, that prompt, and everything after it); a [subagent](../../GLOSSARY.md#subagent)'s window is its whole [transcript](../../GLOSSARY.md#transcript) read with the Transcript Reader, with the stop's `last_assistant_message` appended when the transcript lacks it; a [backlog capture job](#793d1b-terms)'s window is the entries of its [transcript source](./793a_product-spec.md#793a1-problem-statement) after the source's [watermark](#793d1b-terms), read with the Transcript Reader from the watermark with the byte budget `backlogWindowBytes` ([793d16a Reading a transcript from a time](#793d16a-reading-a-transcript-from-a-time)), so that a source larger than the budget is captured over several turns. It MUST remove from the window every block from a line `[engram memory]` through the next line `[end engram memory]`, replace each credential, token and private key matched by the redactor's patterns with `[redacted]` in the window and in every later request and [note](./793a_product-spec.md#793a1-problem-statement), and build the extraction request from the criteria of the learn [skill](../../GLOSSARY.md#skill)'s Step 2 and the recall skill's Step 4, the redacted window and the file names of the notes present in the agent's [context](./793a_product-spec.md#793a1-problem-statement) (none for a [backlog](./793a_product-spec.md#793a1-problem-statement) window, because no agent holds it). It MUST read the reply as JSON with `lessons` and `conclusions` and end the [job](#793d1b-terms) as an error when it cannot; keep as candidates each [lesson](./793a_product-spec.md#793a1-problem-statement) of one of the four kinds (correction, explicit save-request, [reversal](../../GLOSSARY.md#reversal-capture-kind), [confirmed approach](../../GLOSSARY.md#confirmed-approach-capture-kind)) that the reply marks confirmed and that carries the fields its write needs, and each conclusion the reply marks sound and non-trivial with an inference of deduction, abduction or induction and a hedge of `none` or `probable`; drop a conclusion hedged below probable; and for each candidate query the [vault](./793a_product-spec.md#793a1-problem-statement) by its situation and build one reconcile request giving the candidate and the `candidateNotes` nearest notes and asking whether an existing note already states it (`covered`, naming the note) or not (`absent`). When the window is empty after the removals, or no candidate remains, it MUST finish the job with no further request.

Supports [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a3q1c A hunch is not persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a4c1e A secret in a correction is not persisted](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios), [793a4f1b A session of its own receives no delivery](./793a_product-spec.md#793a4f1-uat-scenarios), [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d7a1 Contract Scenarios

```gherkin
Scenario Outline: 793d7a1a A window carries the message it corrects
  Given <source>
  When the window is formed
  Then it holds <window>
  Examples:
    | source | window |
    | a turn whose prompt is "no, use the targ build system, not go test directly" after an assistant message that ran go test, with $.session.messages() ahead of the transcript file | the preceding assistant message, the prompt and the turn's messages, taken from the messages |
    | a subagent transcript in which the dispatching agent says "no, run targ test, not go test" | the whole transcript and the stop's last assistant message once |

Scenario Outline: 793d7a1c Secrets and the System's own blocks never reach a request
  Given a window with <content>
  When the request is built
  Then the request holds <result>
  Examples:
    | content                                                                        | result                                                |
    | the token "sk-ant-api03-AAAAAAAAAAAAAAAAAAAA"                                  | "[redacted]" and not the token                        |
    | a block from "[engram memory]" through "[end engram memory]" inside a longer message | none of that block's text and the rest of the message |

Scenario Outline: 793d7a1e A job without material asks for nothing
  Given <state>
  When the job runs
  Then it finishes with no further request and zero events
  Examples:
    | state                                                  |
    | a window that is empty after the removals              |
    | an extraction reply with no lessons and no conclusions |

Scenario: 793d7a1f The extraction request carries the criteria, the window and the notes present
  Given a window and a context that holds notes A and B by file name
  When the extraction request is built
  Then the system text holds the four kinds of lesson and the bar and tells the model to return JSON only and run nothing
  And the prompt holds the redacted window and the file names A and B, and the request names the model judgeTiers gives for extraction, which by default is the cheapest tier

Scenario Outline: 793d7a1g Candidates are the confirmed lessons and the sound conclusions
  Given an extraction reply with <reply>
  When the reply is read
  Then <result>
  Examples:
    | reply                                                                          | result                                                                     |
    | one correction lesson that is confirmed and complete                           | one candidate exists and one reconcile request lists it with the five nearest notes |
    | a conclusion hedged "hunch"                                                    | no candidate exists for it                                                 |
    | a conclusion marked sound and non-trivial, inference abduction, hedge probable | one candidate exists and its note text will mark the conclusion probable   |

Scenario: 793d7a1j A raised tier changes only its kind
  Given judgeTiers that raises reconcile above the cheapest tier and leaves extraction as is
  When a window with a candidate is judged
  Then the extraction request names the cheapest tier's model and each reconcile request names the raised model

Scenario: 793d7a1h An unreadable reply is an error
  Given an extraction reply that is not JSON
  When the reply is read
  Then the job ends as an error

Scenario: 793d7a1k A backlog window starts at the watermark
  Given a transcript source with a watermark W, entries before W and entries after W, and a context that holds notes A and B
  When the backlog capture job's window is formed
  Then it holds only the entries after W, as the Transcript Reader returns them with the byte budget backlogWindowBytes
  And the extraction request names no note as present
```

### 793d7c Reconciling and writing

Requirement: 793d7c The Auto Capture MUST read each reconcile reply. For a candidate judged covered by a named existing [note](./793a_product-spec.md#793a1-problem-statement) it MUST record a use of that note through Note Amendment and write nothing; for a candidate judged absent it MUST write one note through Note Authoring, of the kind the candidate names (feedback for a correction or a [reversal](../../GLOSSARY.md#reversal-capture-kind), fact for an explicit save-request, feedback or [runbook](../../GLOSSARY.md#runbook) for a [confirmed approach](../../GLOSSARY.md#confirmed-approach-capture-kind), fact or feedback for a conclusion, and also a question-and-answer note when a conclusion's body cites at least one `[[file name]]`), with the source the learn [skill](../../GLOSSARY.md#skill) defines for it followed by `; written automatically (source <source>, position <position>)`, where `<source>` is the path of the [transcript](../../GLOSSARY.md#transcript) the [window](#793d1b-terms) came from (the [job](#793d1b-terms)'s `transcriptPath`, a [subagent](../../GLOSSARY.md#subagent)'s transcript path or a [backlog capture job](#793d1b-terms)'s source path, else the [session](../../GLOSSARY.md#session) id) and `<position>` is the time of the window's last entry (for a turn, its `turnEndedAt`). It MUST write candidates one at a time in the reply's order, count `events.added` once for each note written, amend, discard or delete no note other than to record a use, and write no credential, token or private key. When a write reports that the parent is unreachable and an offer is queued (`engram: parent unreachable (retry after …); N offer(s) queued`), it MUST count the note as written, add the notice `note saved, offer queued (parent unreachable)` to [`notices`](#793d1b-terms) once for the [job](#793d1b-terms), and neither write the note again nor report the job as failed.

Supports [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1i A captured note records how it was written](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a4c1e A secret in a correction is not persisted](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios), [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios), [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d7c1 Contract Scenarios

```gherkin
Scenario: 793d7c1a An absent correction is written with its provenance
  Given a candidate correction "use the targ build system, not go test directly" judged absent in the window of transcript P ending at time T
  When the write runs
  Then Note Authoring receives one feedback write whose source is the learn skill's source followed by "; written automatically (source P, position T)"
  And events.added is 1

Scenario Outline: 793d7c1b A covered candidate writes nothing
  Given a candidate judged covered by note N because <why>
  When the write runs
  Then no note is written, Note Amendment receives a recorded use of N, and the vault holds one note for the rule
  Examples:
    | why                                                        |
    | N was already in the vault                                 |
    | the agent ran the learn skill in the turn and wrote N      |
    | an earlier capture job for the same lesson wrote N         |

Scenario: 793d7c1e A cited conclusion also gets a question-and-answer note
  Given a conclusion whose body cites [[a.md]] and [[b.md]], judged absent
  When the write runs
  Then one conclusion note and one question-and-answer note with contributors a.md and b.md are written

Scenario: 793d7c1g The System changes no other note and writes no credential
  Given any capture job whose text came from a redacted window
  When it finishes
  Then the only calls to Note Amendment are recordings of a use, and no written field holds a credential, token or private key

Scenario: 793d7c1h An unreachable parent is a notice and not a failure
  Given two writes in one job that both report "engram: parent unreachable (retry after 30s); 1 offer(s) queued"
  When the job finishes
  Then events.added is 2 and notices holds "note saved, offer queued (parent unreachable)" once
  And the job has no warning and each lesson is in the vault once

Scenario: 793d7c1j A job without a transcript path names its session
  Given a turn capture job for session S with no transcriptPath, ending at time T, and a candidate correction judged absent
  When the write runs
  Then Note Authoring receives one feedback write whose source is the learn skill's source followed by "; written automatically (source S, position T)"
```

### 793d7e The backlog scan and the watermark

Requirement: 793d7e The Auto Capture MUST keep a [watermark](#793d1b-terms) for each [transcript source](./793a_product-spec.md#793a1-problem-statement) in one state file beside the chunk index's manifest, replaced atomically through the Durable File Store ([792b3c1 Atomic replace](../../specs/technical/durable-file-store.md#792b3c1-atomic-replace)) with mode `0600`, and MUST record in it the time it was first written; a source with no recorded watermark has that time as its watermark, so that a [transcript](../../GLOSSARY.md#transcript) recorded before then is not part of the [backlog](./793a_product-spec.md#793a1-problem-statement). It MUST NOT move a watermark backwards, and MUST NOT write to, move or delete a transcript source.

On a [scan job](#793d1b-terms) it MUST take the [transcript sources](./793a_product-spec.md#793a1-problem-statement) from Ingestion Source Discovery: the files `engram ingest --auto` finds with the session-log or `.pi`-ancestor origin and a `.jsonl` name ([792b17c7 Walking a root](../../specs/technical/ingestion-source-discovery.md#792b17c7-walking-a-root), [792b17c8 Source files](../../specs/technical/ingestion-source-discovery.md#792b17c8-source-files)). It MUST leave out each [live source](#793d1b-terms) (the [job](#793d1b-terms)'s own [session](../../GLOSSARY.md#session) [transcript](../../GLOSSARY.md#transcript), being the file at its `transcriptPath` or else the file named for its session id, and any file modified less than `backlogQuietMs` ago) and each source not modified since its [watermark](#793d1b-terms). Taking the rest from the most recently modified, it MUST read each from its watermark with the byte budget `backlogWindowBytes` ([793d16a Reading a transcript from a time](#793d16a-reading-a-transcript-from-a-time)) until `backlogPerTurn` sources hold content, MUST advance the watermark of a source that holds none to the time of the scan, and MUST return the paths of the sources that hold content.

When a capture [job](#793d1b-terms) finishes without a warning, the Auto Capture MUST advance its source's [watermark](#793d1b-terms): for a [backlog capture job](#793d1b-terms), to the time of the last entry its [window](#793d1b-terms) held (so that a window the byte budget stopped is continued by the next read); for a [subagent](../../GLOSSARY.md#subagent)'s capture job, to the time of the last entry of its [transcript](../../GLOSSARY.md#transcript); and for a turn's capture job that carries a `transcriptPath`, to its `turnEndedAt`. A job that ends with a warning MUST leave the watermark unchanged. A failure to read or write the state file MUST end the job as an error. A backlog capture job writes its notes through the same commands as any other capture and so under the same locks ([793d2b Turn memory work](#793d2b-turn-memory-work)); the Auto Capture MUST NOT write a note for a lesson that a note already records, including one the agent of the source's harness wrote itself before the job ran ([793a3a Lessons are captured without the agent](./793a_product-spec.md#793a3a-lessons-are-captured-without-the-agent)).

Supports [793a3a1p A note another harness's agent wrote is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5c1e A lesson the Pi agent saved is not written again by the backlog](./793a_product-spec.md#793a5c1-uat-scenarios), [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1m A running session is not captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1n A transcript is captured once](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios).

#### 793d7e1 Contract Scenarios

```gherkin
Scenario Outline: 793d7e1a Only session transcripts of the swept origins are candidate sources
  Given Ingestion Source Discovery returns <file> with the <origin> origin
  When a scan job runs
  Then <result>
  Examples:
    | file     | origin      | result                                |
    | s1.jsonl | session-log | the file is a candidate source        |
    | p1.jsonl | pi-ancestor | the file is a candidate source        |
    | notes.md | repository  | the file is not a candidate source    |

Scenario Outline: 793d7e1b A live source is left out
  Given a candidate source modified <modified>, and a scan job whose <own>
  When the scan job runs
  Then the source is <result>
  Examples:
    | modified                                  | own                                         | result       |
    | 1 minute ago, less than backlogQuietMs    | transcriptPath names another file           | left out     |
    | 1 hour ago                                | transcriptPath names another file           | a candidate  |
    | 1 hour ago                                | transcriptPath names this file              | left out     |
    | 1 hour ago                                | session id is this file's name and there is no transcriptPath | left out |

Scenario: 793d7e1c The most recently modified sources come first and are bounded
  Given backlogPerTurn 1 and three quiet sources A (newest), B and C, each holding content after its watermark
  When a scan job runs
  Then it returns A alone
  And after A's backlog capture job finishes without a warning, the next scan job returns B alone

Scenario: 793d7e1d A source not modified since its watermark is not read
  Given a quiet source modified before its watermark
  When a scan job runs
  Then the Transcript Reader is not called for it and it is not returned

Scenario: 793d7e1e A source with no content is advanced and not read again
  Given a quiet source modified after its watermark whose entries after the watermark are all removed as noise
  When a scan job runs
  Then it is not returned and its watermark is the time of the scan
  And a second scan job does not read it

Scenario: 793d7e1f A source with no watermark starts at the first write of the state file
  Given a state file first written at time T0, a quiet source S1 last modified before T0 and a quiet source S2 modified after T0 with entries after T0, and no recorded watermark for either
  When a scan job runs
  Then S1 is not read and not returned, and S2 is returned

Scenario Outline: 793d7e1g A capture job that finishes advances its source's watermark
  Given <job> that finishes without a warning
  When it finishes
  Then <result>
  Examples:
    | job                                                          | result                                              |
    | a backlog capture job whose window ends at the entry at time T | the watermark of its source is T                   |
    | a subagent's capture job whose transcript ends at time T      | the watermark of the subagent's transcript is T     |
    | a turn's capture job with transcriptPath P and turnEndedAt T  | the watermark of P is T                             |
    | a turn's capture job with no transcriptPath                   | no watermark changes                                |

Scenario: 793d7e1h A window stopped by the byte budget is continued
  Given a quiet source whose entries after its watermark exceed backlogWindowBytes
  When its backlog capture job finishes without a warning
  Then its watermark is the time of the last entry the window held
  And the next scan job returns the source again

Scenario: 793d7e1i A job that ends with a warning leaves the watermark
  Given a backlog capture job whose second write fails
  When the job ends with a warning
  Then the watermark is unchanged and the next scan job returns the source again

Scenario: 793d7e1j A watermark never moves backwards and a source is never written
  Given a watermark at time T2 and a job that would advance it to the earlier time T1
  When the job finishes
  Then the watermark is T2
  And every transcript source file is identical to its content before the scan and the job

Scenario: 793d7e1k The state file is replaced atomically and records its first write
  Given no state file
  When the first watermark is advanced
  Then the Durable File Store receives one atomic replace of the state file beside the manifest, with mode 0600, holding the time of the first write and the watermark

Scenario Outline: 793d7e1l A state file that cannot be read or written ends the job as an error
  Given a state file that cannot be <failure>
  When <job> runs
  Then the job ends as an error and <result>
  Examples:
    | failure | job                                         | result                                      |
    | read    | a scan job                                  | returns no source                           |
    | written | a backlog capture job that would advance it | the watermark is unchanged and the notes it wrote are still counted |

Scenario: 793d7e1m A note the source's own agent wrote is not written again
  Given a quiet source holding a correction that the agent of the source's harness saved with its own learn, so that a note states it
  When a backlog capture job runs for the source
  Then no note is written for the correction
  And the vault holds one note for it
```

## 793d9 Added: auto-curation

Canonical destination: `docs/specs/technical/auto-curation.md` (title `Auto Curation - Technical Spec`).

- Ownership: repository component; the caller of the [Query Engine](../../specs/technical/query-engine.md) and [Note Amendment](../../specs/technical/note-amendment.md).
- Responsibility: settles the [vault](./793a_product-spec.md#793a1-problem-statement)'s pending [offers](../../GLOSSARY.md#offer) as the curate [skill](../../GLOSSARY.md#skill) does, with model replies for the judgements and Note Amendment for the changes.
- Current realization (informational): `internal/auto/curate.go`; the criteria are the curate skill's Step 2 table and its special case, embedded at build time.
- Qualification: stateable without naming a function; the [judgement requests](#793d1b-terms) and the choice of amend form are distinct from the amend and the runner's [rounds](#793d1b-terms); tested over scripted replies and a fake amender; stable across curate-skill wording changes; [793a2l Indexing and curation happen without the agent](./793a_product-spec.md#793a2l-indexing-and-curation-happen-without-the-agent) is delivered through it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d9a Settling offers

Requirement: 793d9a The Auto Curation MUST find the [vault](./793a_product-spec.md#793a1-problem-statement)'s pending [offers](../../GLOSSARY.md#offer) and, when there are none, finish with no request. For each offer that is not a skill-runbook registration it MUST build one request, naming the model [`judgeTiers`](#793d1b-terms) gives for `curation`, with the offer and its candidates (the [note](./793a_product-spec.md#793a1-problem-statement) its `offer.for` names, else the notes the query by its situation returns), and apply the reply with Note Amendment as the curate [skill](../../GLOSSARY.md#skill)'s table does: covered ([792a22f A covered offer is reinforced and folded away](../../specs/product/curate.md#792a22f-a-covered-offer-is-reinforced-and-folded-away)), near ([792a22g A near offer is folded away first, then its claim is folded in](../../specs/product/curate.md#792a22g-a-near-offer-is-folded-away-first-then-its-claim-is-folded-in)), absent ([792a22h An absent offer is accepted with its declared author](../../specs/product/curate.md#792a22h-an-absent-offer-is-accepted-with-its-declared-author)) and rejected ([792a22i A rejected offer is discarded outright](../../specs/product/curate.md#792a22i-a-rejected-offer-is-discarded-outright)). It MUST pass the offer's exchange hash when it has one and, when the hash check refuses it, leave the offer pending and continue. For a pending note that carries a `skill_hash` ([792a22k A pending skill runbook note is authored, not judged](../../specs/product/curate.md#792a22k-a-pending-skill-runbook-note-is-authored-not-judged)) it MUST build one request, naming the same model, from the skill's body asking for the note's `situation`, `done_when`, `triggers` and `red_flags`, amend the note with the reply and clear its pending marker, and MUST NOT judge it covered, near or rejected or discard it. It MUST count `events.curated` for each offer or note settled and `events.amended` for each existing note amended by a near, change no note that is neither pending nor named as an existing note by a reply, and not ask the agent to do any of it.

Supports [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios).

#### 793d9a1 Contract Scenarios

```gherkin
Scenario Outline: 793d9a1a An offer is settled by the judged outcome
  Given a pending offer judged <reply>
  When the curation runs
  Then <calls>, events.curated is 1 and events.amended is <amended>
  And no note that is neither pending nor named by the reply is changed
  Examples:
    | reply    | calls                                                                         | amended |
    | covered  | a use of the existing note is recorded and the offer is discarded into it     | 0       |
    | near     | the offer is discarded into the note and the note is amended with the claim   | 1       |
    | absent   | the offer's pending marker is cleared                                         | 0       |
    | rejected | the offer is discarded without --into                                         | 0       |

Scenario: 793d9a1e A refused hash leaves the offer pending
  Given an offer whose exchange hash check is refused, and other offers that are not
  When the curation runs
  Then the refused offer stays pending, the others are settled and events.curated counts the settled ones only

Scenario: 793d9a1j A curation request names its tier's model
  Given a pending offer and judgeTiers giving curation the cheapest tier
  When the curation builds its request
  Then the request names that model

Scenario: 793d9a1f No offer means no request
  Given a vault with no pending offer
  When the curation runs
  Then it finishes with no request and zero events

Scenario: 793d9a1h A registration offer is authored and never discarded
  Given a pending runbook note with a skill_hash and no situation
  When the curation runs with any reply
  Then the note is amended with the replied situation, done_when, triggers and red_flags and its pending marker is cleared
  And the note is not discarded or folded into another, and events.curated is 1
```

## 793d10 Added: harness-deploy-sync (mod deploy)

Canonical destination: `docs/specs/technical/harness-deploy-sync.md` (the existing component; the requirement below is Added to it). Its orientation is unchanged ([792b29a Orientation](../../specs/technical/harness-deploy-sync.md#792b29a-orientation)); responsibility added: deploying the mod directory for the harnesses that load mods.

### 793d10a Mod deploy

Requirement: 793d10a The Harness Deploy Sync MUST, for each detected [harness](../../GLOSSARY.md#harness) that has a mods surface, plan one copy of every file under `agent-instructions/mods/` of the source into the harness's engram-owned root under `mods/`, keeping paths relative to that directory, and make `<mods surface>/<top-level mod directory>` a link into the root (in manifest mode, a copy recorded in the manifest). Claude Code's mods surface is `~/.claude/skills`; Pi has none, so for Pi it MUST plan no mod file and MUST NOT create `~/.pi`. It MUST NOT remove, replace or alter any file that the update deploys for [skills](../../GLOSSARY.md#skill), guidance or imports, MUST apply to the mod the rules it applies to skills ([792b29c14 Dry run](../../specs/technical/harness-deploy-sync.md#792b29c14-dry-run), [792b29c16 Idempotence](../../specs/technical/harness-deploy-sync.md#792b29c16-idempotence), a mod file that left the source and a link that points at nothing are removed, and nothing it cannot prove it owns is deleted), and MUST report in the harness's section of the report each mod directory deployed, adopted or left alone. For Claude Code it MUST also run `claude --version` through the existing subprocess capability and, when `claude` cannot be run or reports a version older than 2.1.287, report with the mod's row that the mod will not load.

Supports [793a4h1a Update installs the Integration for the harness it updates](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1c A preview installs none of the Integration](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1d One update serves both harnesses differently](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1l Running update twice changes nothing](./793a_product-spec.md#793a4h1-uat-scenarios), [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios), [793a5c1b A Pi session on a machine whose Claude Code has the Integration](./793a_product-spec.md#793a5c1-uat-scenarios).

#### 793d10a1 Contract Scenarios

```gherkin
Scenario: 793d10a1a Claude Code gets the mod in its root and a link in its skills directory
  Given ~/.claude exists and the source holds agent-instructions/mods/engram-memory
  When the mod deploy is planned and applied for Claude Code
  Then ~/.claude/engram/mods/engram-memory holds every file of the source directory and ~/.claude/skills/engram-memory is a link into it

Scenario Outline: 793d10a1b Only Claude Code gets mod files
  Given ~/.claude exists and ~/.pi <pi>
  When the mod deploy is planned
  Then mod files are planned for Claude Code only and no path under ~/.pi is planned or created
  Examples:
    | pi             |
    | does not exist |
    | exists         |

Scenario: 793d10a1d An existing install keeps what it had, and the report names the mod
  Given a Claude Code install with skills, guidance and shim deployed, and an earlier copy of engram-memory in the skills directory
  When the mod deploy is applied
  Then every skill, guidance file and import is unchanged
  And Claude Code's section of the report names engram-memory as deployed, adopted or left alone

Scenario: 793d10a1h A preview writes no mod file
  Given a machine with no mod
  When the update runs with --dry-run
  Then the report lists the mod as it would be deployed and no file or link exists afterwards

Scenario Outline: 793d10a1i The sync rules hold for the mod
  Given <state>
  When the update runs
  Then <result>
  Examples:
    | state                                                                        | result                                        |
    | the mod deployed by an earlier run, same source                              | no file or link changes and the mod is reported unchanged |
    | a deployed mod file that the source no longer holds                          | the file is removed from the root             |
    | ~/.claude/skills holds a real directory my-own-plugin outside the deploy set | the directory is left untouched               |

Scenario Outline: 793d10a1l The report says when the mod will not load
  Given `claude --version` <claude>
  When the update runs
  Then the mod's row <row>
  Examples:
    | claude          | row                        |
    | reports 2.1.296 | carries no warning         |
    | reports 2.1.200 | says the mod will not load |
    | cannot be run   | says the mod will not load |
```

## 793d16 Added: transcript-reader (read from a time)

Canonical destination: `docs/specs/technical/transcript-reader.md` (the existing component). One requirement is added to it; the others are unchanged. The component's current realization already reads from a start time with a byte budget, which the canonical requirements do not yet state ([793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped)).

### 793d16a Reading a transcript from a time

Requirement: 793d16a The transcript-reader MUST, given a start time and a byte budget as well as a path, return the conversation of only the entries whose timestamp is after the start time (an entry without a timestamp taking the previous entry's, and an entry before any timestamped entry the zero time), add whole entries in order while their total length stays within the budget, always returning the first entry that qualifies, and report the timestamp of the last entry it returned and whether the budget stopped the read. With a zero start time and no budget it MUST return what [792b15c1 Reading a transcript](../../specs/technical/transcript-reader.md#792b15c1-reading-a-transcript) returns.

Supports [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1n A transcript is captured once](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d16a1 Contract Scenarios

```gherkin
Scenario: 793d16a1a Only entries after the start time are returned
  Given a transcript with entries at 10:00, 10:05 and 10:10 and the start time 10:00
  When it is read
  Then the conversation holds the 10:05 and 10:10 entries and the reported last time is 10:10

Scenario: 793d16a1b An entry without a timestamp takes the previous entry's
  Given a transcript whose entry at 10:05 is followed by an entry with no timestamp, and the start time 10:00
  When it is read
  Then both entries are returned

Scenario: 793d16a1c The budget stops the read and says so
  Given three qualifying entries and a budget that the third would take the total past
  When the transcript is read
  Then the conversation holds the first two, the reported last time is the second's, and the read is reported as stopped by the budget

Scenario: 793d16a1d The first qualifying entry is always returned
  Given a first qualifying entry longer than the budget
  When the transcript is read
  Then that entry is returned and the read is reported as stopped by the budget if another entry follows

Scenario: 793d16a1e A zero start time and no budget read the whole transcript
  Given a transcript and a zero start time with no budget
  When it is read
  Then the conversation is the one the whole-file read returns, and the read is not reported as stopped
```

## 793d11 Product-scenario mapping

Every scenario of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is listed once with the technical requirements that realize it; each requirement's `Supports` list is the same mapping read the other way; the scenario text is not restated here ([793b188 Technical spec structure: flows and components, with the scenario mapping as a table](./793b_decision-log.md#793b188-technical-spec-structure-flows-and-components-with-the-scenario-mapping-as-a-table)). A flow requirement ([793d2a Moment delivery](#793d2a-moment-delivery), [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d2f Mod install](#793d2f-mod-install)) realizes a scenario by its routing, ordering, continuation and gating; a component requirement realizes it by the behavior it owns.

| Product scenario | Realized by |
| --- | --- |
| [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Moments and lookup](#793d6a-moments-and-lookup), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Moments and lookup](#793d6a-moments-and-lookup), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Moments and lookup](#793d6a-moments-and-lookup), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Moments and lookup](#793d6a-moments-and-lookup), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Moments and lookup](#793d6a-moments-and-lookup), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Moments and lookup](#793d6a-moments-and-lookup), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2d1a Delivered content names its source](./793a_product-spec.md#793a2d1-uat-scenarios) | [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2d1b A delivered chunk names its id](./793a_product-spec.md#793a2d1-uat-scenarios) | [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a2k1c An intentional recall by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) |
| [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing), [793d9a Settling offers](#793d9a-settling-offers) |
| [793a2k1e An intentional engram query by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) |
| [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |
| [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d9a Settling offers](#793d9a-settling-offers) |
| [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text), [793d9a Settling offers](#793d9a-settling-offers) |
| [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1i A captured note records how it was written](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3a1k An interrupted Pi session is captured after the fact](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing), [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d16a Reading a transcript from a time](#793d16a-reading-a-transcript-from-a-time) |
| [793a3a1l A Claude Code session killed mid-turn is captured by a later session](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing), [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d16a Reading a transcript from a time](#793d16a-reading-a-transcript-from-a-time) |
| [793a3a1m A running session is not captured](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark) |
| [793a3a1n A transcript is captured once](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d16a Reading a transcript from a time](#793d16a-reading-a-transcript-from-a-time) |
| [793a3a1o A session is captured once it is quiet](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing), [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d16a Reading a transcript from a time](#793d16a-reading-a-transcript-from-a-time) |
| [793a3a1p A note another harness's agent wrote is not written again](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d7a Judging a window](#793d7a-judging-a-window) |
| [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a3q1c A hunch is not persisted](./793a_product-spec.md#793a3q1-uat-scenarios) | [793d7a Judging a window](#793d7a-judging-a-window) |
| [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |
| [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d2b Turn memory work](#793d2b-turn-memory-work), [793d2e Job order within a round](#793d2e-job-order-within-a-round), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |
| [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |
| [793a4b1d Slow capture does not stall the session](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work) |
| [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) |
| [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6b Presence, relevance and delivery text](#793d6b-presence-relevance-and-delivery-text) |
| [793a4c1e A secret in a correction is not persisted](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Judging a window](#793d7a-judging-a-window), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a4f1b A session of its own receives no delivery](./793a_product-spec.md#793a4f1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Judging a window](#793d7a-judging-a-window) |
| [793a4h1a Update installs the Integration for the harness it updates](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a4h1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a4h1c A preview installs none of the Integration](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a4h1d One update serves both harnesses differently](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a4h1l Running update twice changes nothing](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a5c1b A Pi session on a machine whose Claude Code has the Integration](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d2f Mod install](#793d2f-mod-install), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d10a Mod deploy](#793d10a-mod-deploy) |
| [793a5c1d A Pi agent's own ingest and learn run while the System ingests and captures](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work) |
| [793a5c1e A lesson the Pi agent saved is not written again by the backlog](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d7e The backlog scan and the watermark](#793d7e-the-backlog-scan-and-the-watermark), [793d2b Turn memory work](#793d2b-turn-memory-work) |
| [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |
| [793a5h1b Nothing is shown after a turn without events](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1c The summary is not in the agent's context](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1e The counts cover the events since the previous summary](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d7c Reconciling and writing](#793d7c-reconciling-and-writing) |
| [793a5h1i A backlog capture is counted](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |

## 793d12 Recommended impact

**Repository.** New: `agent-instructions/mods/engram-memory/` (the mod and its tests); `internal/auto/` (the runner and four components, the [backlog](./793a_product-spec.md#793a1-problem-statement) scan and the [watermark](#793d1b-terms) file among them); `internal/cli/auto.go` (`engram auto run`); a small Go package under `agent-instructions/` embedding the [skill](../../GLOSSARY.md#skill) files the judgement criteria come from. Changed: `internal/update` (mod deploy set, report row and the `claude --version` check, which needs a command-running seam in the update path: a `cli.Primitives` field if none exists), `cmd/engram/main.go` needs no change for the [job](#793d1b-terms), which arrives on `Deps.Stdin`. Docs at integration: the canonical files named in [793d2 Added: flows](#793d2-added-flows) to [793d10 Added: harness-deploy-sync (mod deploy)](#793d10-added-harness-deploy-sync-mod-deploy), the added requirement of the [Transcript Reader](../../specs/technical/transcript-reader.md) ([793d16 Added: transcript-reader (read from a time)](#793d16-added-transcript-reader-read-from-a-time)), `docs/specs/product/automatic-memory.md`, the glossary terms (mod, [moment](./793a_product-spec.md#793a1-problem-statement), [delivery](./793a_product-spec.md#793a1-problem-statement), job, [round](#793d1b-terms)), the `AGENTS.md` directory table, and `auto run` in the [Command Line Surface](../../specs/technical/command-line-surface.md).

**Dependencies and toolchain.** Runtime: Claude Code 2.1.287 or later; `engram` on the `PATH` of the Claude Code process. Development: the Claude Code CLI, which the mod's integration test needs ([793d3g Validated and tested through the existing checks](#793d3g-validated-and-tested-through-the-existing-checks), [793b236 The mod is validated and tested through the existing integration gate, not a new check target](./793b_decision-log.md#793b236-the-mod-is-validated-and-tested-through-the-existing-integration-gate-not-a-new-check-target)); no JavaScript package manager, bundler or build step; no new Go dependency.

**Tests.** The Go components are tested in the default suite over fakes; the scenarios of this spec use scripted model replies. One `integration`-tagged test starts the built `engram auto run` against a temporary [vault](./793a_product-spec.md#793a1-problem-statement); no `network` test is needed, because Go never calls a model. The mod's tests run under `claude plugin test` with the world beneath it stubbed, started by the `integration`-tagged test of [793d3g Validated and tested through the existing checks](#793d3g-validated-and-tested-through-the-existing-checks). Real-session checks of [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios) and the other capture scenarios depend on model judgement and are probabilistic; they are measured under #850, not asserted per run. [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios) and [793a4f1b A session of its own receives no delivery](./793a_product-spec.md#793a4f1-uat-scenarios) hold by construction, because `$.model.complete` starts no [session](../../GLOSSARY.md#session), so their scenarios pass trivially. Before the change is complete, the installed binary and deployed mod are run in a real Claude Code session, the unverified rows of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods) are decided, and the product scenarios' observable outputs ([transcript](../../GLOSSARY.md#transcript), vault, status line) are inspected.

**Operations.** Each prompt, `please`, dispatch and [subagent](../../GLOSSARY.md#subagent) start waits for one query before the agent's next step, about 11 seconds on the author's [vault](./793a_product-spec.md#793a1-problem-statement) (the shim's measured query, moved from the agent to the [System](./793a_product-spec.md#793a1-problem-statement)), and for one `relevance` call (bounded by `momentJudgeDeadlineMs`) when a candidate remains. The first edit and the first failure of a turn also wait for one `classify` call before the query, so a [judged cue](#793d1b-terms) waits for the query and up to two judgement calls, 21 s at most inside `momentDeadlineMs`. Each step of the main agent that ends with text and a tool call runs, in the background and one at a time, one `classify` call and, when it names a [moment](./793a_product-spec.md#793a1-problem-statement), one query and one `relevance` call when a candidate remains: one cheap call per such step, two more only when a moment is named, and no delay to the step. Each top-level turn runs in the background one extraction request plus one reconcile request per candidate, one scan of the [transcript sources](./793a_product-spec.md#793a1-problem-statement) for the [backlog](./793a_product-spec.md#793a1-problem-statement) (a local read that makes no model request when no source holds content) and, for each source it returns (at most `backlogPerTurn`), one more extraction request plus one reconcile request per candidate, one `engram ingest --auto` and one scan for pending [offers](../../GLOSSARY.md#offer) (one request per offer found), on the cheapest tier available to the [session](../../GLOSSARY.md#session) (the `haiku` family) and the session's credentials ([793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier)); #850 measures judgement quality per request kind and is the only path to raising a kind's tier. A process the mod starts runs outside any sandbox with the user's permissions, as every mod does.

**Risks.** The mods runtime is early access; the To-verify rows of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods) are the risk, each with its fallback. The two [delivery](./793a_product-spec.md#793a1-problem-statement) judgements are model judgements and can miss a [moment](./793a_product-spec.md#793a1-problem-statement) or drop a useful item; they are probabilistic, #850 measures them, and the prompt-time phrases of [793d6a Moments and lookup](#793d6a-moments-and-lookup) stay beside them. A hook that blocks the shared hooks worker can unload every installed mod, so the mod holds no synchronous loop beyond building one JSON string. The [System](./793a_product-spec.md#793a1-problem-statement) is silently inactive where mods do not load (row 16 of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods)), because a mod that does not load cannot warn.

### 793d12a Known limits and follow-ups

The first, second and fourth are the owner's answers of 2026-10-11 ([793b217 793c63 is answered (a): subagents get a dispatch delivery and a stop capture now](./793b_decision-log.md#793b217-793c63-is-answered-a-subagents-get-a-dispatch-delivery-and-a-stop-capture-now) to [793b220 793c66 is answered (a): the product promise covers the sessions in which mods run](./793b_decision-log.md#793b220-793c66-is-answered-a-the-product-promise-covers-the-sessions-in-which-mods-run)); the third replaces his answer to [793c65 Are captures still queued when a session ends kept for the next session?](./793c_open-questions.md#793c65-are-captures-still-queued-when-a-session-ends-kept-for-the-next-session) with his answer of 2026-10-12 ([793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](./793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact)).

- [Moments](./793a_product-spec.md#793a1-problem-statement) inside a [subagent](../../GLOSSARY.md#subagent)'s own loop are not detected; its dispatch and stop are. Follow-up #852 ([793c63 Must recall moments inside a subagent's own loop be delivered now?](./793c_open-questions.md#793c63-must-recall-moments-inside-a-subagents-own-loop-be-delivered-now)).
- Tier evidence is delivered when the orchestrator routes through the route [skill](../../GLOSSARY.md#skill); a direct `Agent` call has no pre-dispatch channel, so it reaches the subagent at its start and the orchestrator after the return ([793c64 When is tier evidence delivered relative to routing?](./793c_open-questions.md#793c64-when-is-tier-evidence-delivered-relative-to-routing)).
- The [backlog](./793a_product-spec.md#793a1-problem-statement) is captured only when a Claude Code [session](../../GLOSSARY.md#session) with the [Integration](./793a_product-spec.md#793a1-problem-statement) completes a turn, and at most `backlogPerTurn` sources per turn, so a large backlog takes several turns. Work still queued when a session ends, or lost to a crash or a module reload, is found again by a later turn's scan once the [transcript](../../GLOSSARY.md#transcript) has been quiet for `backlogQuietMs` ([793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](./793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact), [793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped)).
- Transcripts recorded before the [watermark](#793d1b-terms) file's first write are not captured: the [backlog](./793a_product-spec.md#793a1-problem-statement) starts there ([793b233 Choices made in applying 793b232, shown to the owner](./793b_decision-log.md#793b233-choices-made-in-applying-793b232-shown-to-the-owner)).
- A source whose [backlog capture job](#793d1b-terms) keeps ending in a warning is retried at every turn, with a warning each time.
- The [System](./793a_product-spec.md#793a1-problem-statement) is silently inactive, and cannot warn, where mods do not load or Claude Code is too old (row 16 of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods)); `engram update` reports an old version. `claude -p` and the VS Code panel do not draw the summary or warning ([793c66 Which platform cases where the System is silently inactive does the product promise cover?](./793c_open-questions.md#793c66-which-platform-cases-where-the-system-is-silently-inactive-does-the-product-promise-cover)).
- The done moment is delivered at the prompt, before the declaration, and by the classification of an earlier step that ended with a tool call; a stop continuation is a MAY ([793b207 Late delivery at stop is a MAY and nothing else is late](./793b_decision-log.md#793b207-late-delivery-at-stop-is-a-may-and-nothing-else-is-late)).
- A step's [delivery](./793a_product-spec.md#793a1-problem-statement) rides a later tool result ([793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context), [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)): a step with no tool call (a final answer) starts no [job](#793d1b-terms), and a [held delivery](#793d1b-terms) is dropped when the turn ends or its ids became present, so a delivery is never added at the declaration of a final answer.
- The classification and the relevance filter are model judgements on the cheapest tier: they can miss a [moment](./793a_product-spec.md#793a1-problem-statement) or drop a useful item. No scenario asserts a particular judgement per run; #850 measures them.
- The constants named here start at the stated values; tuning them needs #850. Redundant agent-side steps stay untouched until #851.
- Two Claude Code [sessions](../../GLOSSARY.md#session) capturing the same [lesson](./793a_product-spec.md#793a1-problem-statement) at the same moment can each write it: the queue serializes captures within one session only, and two sessions' scans can return the same [transcript source](./793a_product-spec.md#793a1-problem-statement) at the same moment.

Deferred to the implementation plan (not normative): `textLimit` 2000, `phraseLimit` 300, `deliveryMinScore` 0.25, `deliveryItemLimit` 10, `taskMinChars` 20, `extractMaxTokens` 2048, `candidateNotes` 5, `relevanceItemChars` 1500, the phrase shape of each [moment](./793a_product-spec.md#793a1-problem-statement) and the prompt text of each request.

## 793d13 Relative option impact

The options compare the same complete scope: the 13 requirements of the product spec, on Claude Code only, with [intentional memory](./793a_product-spec.md#793a1-problem-statement) untouched. Objectives: **O1 Verifiable**, the product scenarios' assertions can be met and checked; **O2 Safe**, the option cannot break or stall the [session](../../GLOSSARY.md#session) or change what the agent or user issued; **O3 Small**, the least repository-owned code and state; **O4 Grounded**, it relies on documented capabilities and the fewest unverified ones; **O5 Today's behavior**, it reuses existing commands and defaults. The comparisons of [moment](./793a_product-spec.md#793a1-problem-statement) detection, [subagent](../../GLOSSARY.md#subagent) coverage, late detection, the summary surface and the JavaScript/Go split are in [793b211 Option comparisons and rejected alternatives moved out of the technical spec](./793b_decision-log.md#793b211-option-comparisons-and-rejected-alternatives-moved-out-of-the-technical-spec); after the owner's decision in [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter), the choice between a judged and a deterministic moment and relevance is in [793d13j Moment classification and relevance](#793d13j-moment-classification-and-relevance).

### 793d13a Mid-turn delivery

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. [Context](./793a_product-spec.md#793a1-problem-statement) channels of the events: prompt context, tool-result context, subagent-start context | yes: separate blocks in the [transcript](../../GLOSSARY.md#transcript), before the next response or tool call (the prompt channel: row 6) | yes: the prompt and the tool call are untouched | yes | yes | Selected |
| B. Rewrite the prompt text, the Agent prompt or a tool result | partly: the [delivery](./793a_product-spec.md#793a1-problem-statement) is not separate content | no: changes what the user or agent issued ([793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only)) | yes | yes | Rejected |
| C. `$.prompt.submit` from a [job](#793d1b-terms) | partly: arrives as a new turn, after the agent's response | no: starts a turn nobody asked for | yes | yes | Rejected |
| D. Continue the turn at stop | partly: after the declaration | partly: an extra step the user sees | partly: needs a classifier | yes | A MAY ([793b207 Late delivery at stop is a MAY and nothing else is late](./793b_decision-log.md#793b207-late-delivery-at-stop-is-a-may-and-nothing-else-is-late)) |
| E. `$.session.send` to the agent | partly: queued, not guaranteed before the next step | yes | yes | partly | Rejected |

### 793d13c Presence source

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. `$.session.messages()` plus a [delivery](./793a_product-spec.md#793a1-problem-statement) [ledger](#793d1b-terms) | yes: reflects compaction and a resume by construction | yes | yes | partly: whether a delivered block is a message is unverified (row 11) | Selected |
| B. Scan the [transcript](../../GLOSSARY.md#transcript) file | partly: the file lags the conversation, so a just-delivered item is missed | yes | partly | partly: a running [subagent](../../GLOSSARY.md#subagent)'s path is undocumented | Rejected |
| C. The [ledger](#793d1b-terms) alone | no: misses what the agent retrieved itself ([793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios)) | yes | yes | yes | Rejected |
| D. Buffer every `session.append` row per agent | yes, and covers [subagents](../../GLOSSARY.md#subagent) | partly: a hook on every row | no | partly | Follow-up (#852) |

### 793d13d Judgement sessions

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. Relay `$.model.complete` (no [session](../../GLOSSARY.md#session)), in the model each request names (cheapest tier by default) | yes: any conversation, including a [subagent](../../GLOSSARY.md#subagent)'s, judged from the text Go builds | yes: no [transcript](../../GLOSSARY.md#transcript), no hooks, no recursion, bounded by deadline and token cap | yes: one fixed relay loop | partly: use inside a timer callback is unverified (row 13) | Selected |
| B. `$.model.fork` | partly: the main conversation only | yes | yes | partly: null on a cold snapshot | Rejected: cannot judge a [subagent](../../GLOSSARY.md#subagent) |
| C. `claude -p` started by the Go command | yes | partly: a real [session](../../GLOSSARY.md#session) with a [transcript](../../GLOSSARY.md#transcript), the user's global instructions and the mod loaded again, so it needs an environment guard and `--safe-mode` | no: a second process model, its output parsing and its leftovers in the [session](../../GLOSSARY.md#session) list | yes | Rejected |

The Go command cannot call a model itself (the repository has no LLM client and no credentials), which also rules out judging inside `engram auto run`.

### 793d13h Install path

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. A `skills-dir` plugin under `~/.claude/skills`, deployed by `engram update` | yes: active from the next [session](../../GLOSSARY.md#session) with no other step | yes: no edit of the user's settings | yes: the existing deploy sync | yes: rows 2 to 4 are verified | Selected |
| B. `CLAUDE_CODE_PLUGIN_DIRS` through `env` in `~/.claude/settings.json` | yes | partly: engram edits the user's settings file; managed `disableSideloadFlags` rejects it | partly: a settings merge | yes | Rejected |
| C. A local-directory marketplace with `extraKnownMarketplaces` and `enabledPlugins` | yes | partly: two edits of the user's settings | partly | yes | The fallback of row 2 |
| D. `claude --plugin-dir` by hand | no: the user has a step ([793a4h Installed by engram update and active in Claude Code](./793a_product-spec.md#793a4h-installed-by-engram-update-and-active-in-claude-code)) | yes | yes | yes | Rejected |

### 793d13j Moment classification and relevance

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. Two judgements on the cheapest tier: a `classify` request where the event does not name the [moment](./793a_product-spec.md#793a1-problem-statement), and a `relevance` request on the candidates; the mechanical [cues](#793d1b-terms) stay the trigger | yes: a moment is named from what the agent is saying, and an item that is near but does not bear on the moment is dropped on its content | partly: up to two bounded calls on the path of a judged cue and none on a step; a failed call skips the moment and the event continues | partly: two request kinds, a [held delivery](#793d1b-terms) and one more hook | partly: `$.model.complete` is relied on already (rows 13 and 18); the step result's fields are declared, and two behaviours are To verify (row 20) | Selected ([793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)) |
| B. Deterministic only: the mechanical [cues](#793d1b-terms), the prompt's four phrases and a score threshold (the design before [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)) | partly: the done moment reaches the agent only at the prompt, and an item is delivered on its score alone | yes: no model call on the path | yes | yes | Rejected by the owner: the judgements of the autonomous parts cannot all be deterministic; the standing finding is that detection is LLM reasoning, not embedding retrieval |
| C. A `classify` call on every tool call | yes | no: a model call before every tool result | partly | partly | Rejected: the cost [793b193 Moments are mechanical cues; the prompt lookup spans recall's four moments](./793b_decision-log.md#793b193-moments-are-mechanical-cues-the-prompt-lookup-spans-recalls-four-moments) named; the cues and the steps that carry text bound the calls |
