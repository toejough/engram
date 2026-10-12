# Parallel Memory Agent - Technical Spec

## 793d1 Orientation

The product scope is the 14 requirements of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec), approved in [793b187 The product spec is approved after the owner's walkthrough](./793b_decision-log.md#793b187-the-product-spec-is-approved-after-the-owners-walkthrough). The System they describe must consult the vault and save lessons at the right moments without the working agent running a memory command, show the user what it did, never stall or break the session, and be installed by `engram update` for Claude Code.

**Premise (owner).** The System is built on Claude Code mods, not on settings hooks: "the whole design is based around the novel capabilities mods provide" ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go)), and "with mods, the system can just print stuff to the user" ([793c40 Does the user still see please's audit list?](./793c_open-questions.md#793c40-does-the-user-still-see-pleases-audit-list)). A mod is a plugin whose `hooks/hooks.json` lists `"modules": ["./register.js"]`; Claude Code loads that JavaScript or TypeScript module in process and calls its `register(on, options)`; each handler `on(event, matcher?, async ($, e, next) => …)` can observe, rewrite or answer an event, and calls the mods API (`$`) for everything outside itself ([Mods overview](https://code.claude.com/docs/en/plugins/mods/overview.md), [Create a mod](https://code.claude.com/docs/en/plugins/mods/create.md), [Mods reference](https://code.claude.com/docs/en/plugins/mods/reference.md), [React to events](https://code.claude.com/docs/en/plugins/mods/events.md), [Use the mods API](https://code.claude.com/docs/en/plugins/mods/api.md), [Test a mod](https://code.claude.com/docs/en/plugins/mods/test.md), [Manage mods](https://code.claude.com/docs/en/plugins/mods/admin.md), [the type declarations](https://github.com/anthropics/claude-code/blob/main/mods/types/claude-code.d.ts), [the built-in mods](https://github.com/anthropics/claude-code/tree/main/mods)). The mods runtime is early access: events and methods may change between releases, and this spec marks every capability it could not confirm.

**What exists today.** Every memory operation the System needs already exists as a command or component: [Query Engine](../../specs/technical/query-engine.md) (lookup), [Note Authoring](../../specs/technical/note-authoring.md) (`learn`), [Note Amendment](../../specs/technical/note-amendment.md) (`amend`, activation, settling an offer), [Transcript Reader](../../specs/technical/transcript-reader.md), `engram ingest --auto` ([Ingestion Source Discovery](../../specs/technical/ingestion-source-discovery.md)) and the deploy of agent instructions ([Harness Deploy Sync](../../specs/technical/harness-deploy-sync.md), [Self Installer](../../specs/technical/self-installer.md)). What is missing is everything that runs them without the agent:

| Product requirement | Existing components that do the work | Confirmed gap |
| --- | --- | --- |
| [793a2n Relevant memory reaches the agent at the moments](./793a_product-spec.md#793a2n-relevant-memory-reaches-the-agent-at-the-moments) | Query Engine; recall and the shim, run by the agent | Nothing observes the session's moments, and nothing can put text in front of the agent except the agent's own tool calls |
| [793a2d Delivered memory is attributable](./793a_product-spec.md#793a2d-delivered-memory-is-attributable) | the query payload carries each item's path, kind, id and score | A delivery text that names the source |
| [793a2e Nothing already in context is delivered again](./793a_product-spec.md#793a2e-nothing-already-in-context-is-delivered-again) | none: the agent decides what it already holds | Reading what an agent's context holds, per agent |
| [793a2k Intentional memory stays available and is never blocked](./793a_product-spec.md#793a2k-intentional-memory-stays-available-and-is-never-blocked) | the skills and guidance, unchanged | Doing all of the above without touching the agent's prompt, tool calls or permissions |
| [793a2l Indexing and curation happen without the agent](./793a_product-spec.md#793a2l-indexing-and-curation-happen-without-the-agent) | `engram ingest --auto`; the curate skill (LLM judgement) | Running both after a turn, and the judgement without an agent |
| [793a3a Lessons are captured without the agent](./793a_product-spec.md#793a3a-lessons-are-captured-without-the-agent) | the learn skill (LLM judgement); Note Authoring | Judgement without an agent, over a subagent's transcript, with secrets removed and duplicates refused |
| [793a3p Dispatch outcomes are recorded by the System](./793a_product-spec.md#793a3p-dispatch-outcomes-are-recorded-by-the-system) | the route skill's report row and evidence note, written by the orchestrator | Reading the row and writing the note |
| [793a3q Sound conclusions are persisted by the System](./793a_product-spec.md#793a3q-sound-conclusions-are-persisted-by-the-system) | the recall skill's rules for a derived note | Judgement without an agent |
| [793a4b The session is never broken or stalled](./793a_product-spec.md#793a4b-the-session-is-never-broken-or-stalled) | none | Bounding, queuing and containing the System's work, and a surface for a warning |
| [793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only) | none | A channel that adds text without rewriting the prompt or a tool call |
| [793a4f No delivery or capture in its own sessions](./793a_product-spec.md#793a4f-no-delivery-or-capture-in-its-own-sessions) | none | Judgement that runs where no hook sees it |
| [793a4h Installed by engram update and active in Claude Code](./793a_product-spec.md#793a4h-installed-by-engram-update-and-active-in-claude-code) | Harness Deploy Sync; Self Installer | A deploy set for something Claude Code loads as a plugin |
| [793a5c Other harnesses are unchanged](./793a_product-spec.md#793a5c-other-harnesses-are-unchanged) | harness isolation of the deploy | The mod reaching Claude Code only |
| [793a5h The user sees a summary of the memory activity](./793a_product-spec.md#793a5h-the-user-sees-a-summary-of-the-memory-activity) | none | Counting events and showing them outside the transcript |

The existing system is therefore sufficient for every vault operation and insufficient for event observation, delivery, judgement without an agent, background execution, the user surface and the mod's deploy. Mods supply the first four and the surface; the deploy is an addition to the existing deploy sync.

**Recommended solution.** Two new things and one extension:

1. The **Memory Mod** ([793d3 Added: memory-mod](#793d3-added-memory-mod)), a thin adapter loaded by Claude Code. It forwards the events listed in [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) to the Go command, adds the text the command returns through the documented context channels, relays the model requests the command asks for through `$.model.complete`, runs the slow work in a background queue, shows one status line, and swallows every failure into a warning.
2. The **`engram auto run` command** ([793d4 Added: auto-job-runner](#793d4-added-auto-job-runner)) and the components behind it: [793d5 Added: moment-detector](#793d5-added-moment-detector) (is this a moment, and what to ask the vault), [793d6 Added: memory-delivery](#793d6-added-memory-delivery) (ask, drop what the agent holds, write the delivery), [793d7 Added: auto-capture](#793d7-added-auto-capture) (judge and write lessons and conclusions), [793d8 Added: dispatch-outcome-recorder](#793d8-added-dispatch-outcome-recorder) (write dispatch outcome notes from report rows) and [793d9 Added: auto-curation](#793d9-added-auto-curation) (settle pending offers). All of the System's decisions are made here, in Go, over the existing injected interfaces.
3. An extension of **Harness Deploy Sync** ([793d10 Added: harness-deploy-sync (mod deploy)](#793d10-added-harness-deploy-sync-mod-deploy)) that deploys the mod, for Claude Code only, as a plugin Claude Code loads from `~/.claude/skills`.

The flows are in [793d2 Added: flows](#793d2-added-flows). The choices that shaped them are summarized in [793d1b Key choices](#793d1b-key-choices) and compared in [793d13 Relative option impact](#793d13-relative-option-impact).

Section map. The flows are [793d2 Added: flows](#793d2-added-flows), the components [793d3 Added: memory-mod](#793d3-added-memory-mod) to [793d10 Added: harness-deploy-sync (mod deploy)](#793d10-added-harness-deploy-sync-mod-deploy), the scenario mapping [793d11 Product-scenario mapping](#793d11-product-scenario-mapping), the impact [793d12 Recommended impact](#793d12-recommended-impact), the option comparison [793d13 Relative option impact](#793d13-relative-option-impact) and the rejected alternatives [793d14 Alternative options](#793d14-alternative-options). The system is new, so there is no change diagram: each flow diagram shows the final system.

Terms used below. **Moment**, **delivery**, **lesson**, **Integration**, **working agent**, **context** and **vault** are as in [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement). A **cue** is the event pattern from which [793d5 Added: moment-detector](#793d5-added-moment-detector) recognizes a moment. A **job** is one unit of the System's work (a moment, a capture, a curation) run by `engram auto run` in rounds; a **round** is one call of the command; a **judgement request** is a model request the command asks the mod to relay. The **ledger** is the set of item ids the mod has delivered to one agent.

### 793d1a What the design relies on in Claude Code mods

"Verified (docs)" means stated by the pages cited above; "Verified (declarations)" means stated by the type declarations at [claude-code.d.ts](https://github.com/anthropics/claude-code/blob/main/mods/types/claude-code.d.ts) (written by Claude Code 2.1.277, older than the documentation, which cites features up to 2.1.296); "To verify" means this draft could not confirm it, and the implementation checks it with `claude plugin validate` and `claude plugin test` against the Claude Code version it targets, using the fallback shown. No item below changes a product promise.

| # | Capability relied on | Status | If it does not hold |
| --- | --- | --- | --- |
| 1 | A plugin whose `hooks/hooks.json` has `modules` loads a hooks module exporting `register(on, options)`; handlers are `($, e, next)`; Claude Code ≥ 2.1.287 | Verified (docs) | None; the premise |
| 2 | A folder under `~/.claude/skills` holding `.claude-plugin/plugin.json` loads as a plugin (`name@skills-dir`) in every session, with no flag, install step or settings edit | Verified (docs: Create a plugin, Plugin loading reference) | Fall back to a local-directory marketplace ([793d13h Install path](#793d13h-install-path)) |
| 3 | The `skills-dir` plugin's hooks module loads (the mods administration page names `skills-dir` as an origin mods load from) | To verify | As row 2 |
| 4 | That load works through a symlink into the engram root | To verify | Manifest mode (real files and a record of what was written) for that surface, [793d10a Mod deploy set](#793d10a-mod-deploy-set) |
| 5 | `prompt.submit` fires for every origin (composer, task notification, scheduled trigger, peer session), with a typed slash command's text as typed, and `next({ ...e, context })` adds text the model reads after the prompt as typed | Verified (docs, declarations: `PromptOrigin`, `PromptSubmitInput.context`); that it fires for each origin and for a typed slash command: To verify | Add `session.receive` for the origin that is missed; for a slash command, the `classic.UserPromptExpansion` event |
| 6 | `additionalContext` returned from `classic.PostToolUse`, `classic.PostToolUseFailure` and `classic.SubagentStart` reaches the model next to the tool result or at the subagent's start | Verified (docs: Add context for Claude; declarations: `ClassicResultFields`); a mod handler returning it: Verified (declarations: `ClassicResult`) | The `context` field of a `tool.call` result (Verified (declarations)) |
| 7 | `agent.spawn` carries the Agent call's `prompt`, `description` and `fork`; `classic.SubagentStart` carries only `agent_id` and `agent_type`; `$.agent.list()` gives an agent's `description` | Verified (declarations) | The spawn is matched to the start by order only ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |
| 8 | `classic.SubagentStop` carries `agent_transcript_path` and `last_assistant_message`; the file is complete when the subagent has stopped | Verified (declarations: fields); completeness: To verify | Retry the read once after one second |
| 9 | `turn.start` and `turn.complete` fire for the main loop without `agentId` | Verified (docs, declarations) | None |
| 10 | `$.session.messages()` returns the messages the next model request carries (so compaction and a resume are reflected), main loop only | Verified (declarations: `TurnStepInput`); after compaction and after a resume: To verify | After a compaction or a resume the ledger is refilled from the delivered blocks (from the line `[engram memory]` to the line `[end engram memory]`) in the transcript file named by `transcript_path` |
| 11 | A context block added through row 5 or 6 appears in `$.session.messages()` | To verify | The ledger covers it until a compaction or a resume; then as row 10 |
| 12 | `$.process.run(argv, { stdin, timeoutMs })` runs without a shell, 30 s by default and 10 minutes at most; time inside a mods API call does not count against a hook's 10 s | Verified (docs, declarations) | None |
| 13 | `$.model.complete({ model, prompt, system, maxTokens })` works inside a timer callback; its result is a string in the declarations and an object `{ isAnswered, text, usage }` in the documentation | Verified (docs, declarations); the shape and a `timeoutMs` option: To verify | The mod accepts either shape and enforces its own deadline ([793d3c Judgement relay and deadlines](#793d3c-judgement-relay-and-deadlines)) |
| 14 | `$.clock.after` callbacks run outside any event and outlive the hook that started them; they do not start a turn | Verified (docs) | None |
| 15 | `$.ui.status(text)` pins one line under the prompt, replaced by the next call and cleared by `undefined`; the line is not a transcript row | Verified (declarations: pinned line); not a transcript row: To verify. `$.ui.log` is documented to add a transcript row and is not used | `$.ui.toast`, documented to leave the transcript and the model untouched |
| 16 | A hook may exceed 10 s only inside mods API calls; `.catch` handlers have 1 s; `session.end` hooks share 1.5 s | Verified (docs) | None |
| 17 | Installed mods share one worker thread; a hook that blocks it is unloaded, and three unexplained crashes unload every non-built-in mod | Verified (docs: Troubleshoot) | The mod does no synchronous loop longer than building one JSON string |
| 18 | The mod is inactive under `--safe-mode`, `--bare`, `disableAllHooks`, managed policy (`allowManagedModsOnly`, `allowManagedHooksOnly`), an untrusted folder, the remote rollout switch, WSL sessions and Claude Code older than 2.1.287 | Verified (docs) | The System is then inactive and cannot warn ([793d12a Known limits and follow-ups](#793d12a-known-limits-and-follow-ups)) |
| 19 | `claude plugin validate` lists the mod's hooks and calls; `claude plugin test` runs `*.test.ts` without a session, sign-in or network | Verified (docs) | None |
| 20 | The Claude Code version each event and field used here first appears in | To verify (`reference.md` gives a version only for features this design does not use) | The mod records the version it was verified against in its README, and `check-mod` runs against that version ([793b206 The mod is validated and tested by claude plugin validate and claude plugin test in check-full](./793b_decision-log.md#793b206-the-mod-is-validated-and-tested-by-claude-plugin-validate-and-claude-plugin-test-in-check-full)) |

### 793d1b Key choices

| Choice | Selected | Compared in | Recorded in |
| --- | --- | --- | --- |
| Mid-turn delivery | prompt context, tool-result context, subagent-start context | [793d13a Mid-turn delivery](#793d13a-mid-turn-delivery) | [793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context) |
| Moment detection | mechanical cues in Go; the prompt lookup spans recall's four moments | [793d13b Moment detection](#793d13b-moment-detection) | [793b193 Moments are mechanical cues; the prompt lookup spans recall's four moments](./793b_decision-log.md#793b193-moments-are-mechanical-cues-the-prompt-lookup-spans-recalls-four-moments) |
| Presence in context | `$.session.messages()` plus a delivery ledger | [793d13c Presence source](#793d13c-presence-source) | [793b195 Presence is read from the live context and a delivery ledger](./793b_decision-log.md#793b195-presence-is-read-from-the-live-context-and-a-delivery-ledger) |
| Judgement | relayed `$.model.complete`, not a session | [793d13d Judgement sessions](#793d13d-judgement-sessions) | [793b196 Judgement runs as relayed model completions, not sessions](./793b_decision-log.md#793b196-judgement-runs-as-relayed-model-completions-not-sessions) |
| Subagent coverage | delivery at start, capture at stop | [793d13e Subagent coverage](#793d13e-subagent-coverage) | [793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected) |
| Late detection | none required; a stop continuation is a MAY | [793d13f Late detection](#793d13f-late-detection) | [793b207 Late delivery at stop is a MAY and nothing else is late](./793b_decision-log.md#793b207-late-delivery-at-stop-is-a-may-and-nothing-else-is-late) |
| Summary and warning surface | `$.ui.status` | [793d13g Summary and warning surface](#793d13g-summary-and-warning-surface) | [793b204 The memory-activity line is the mod's status line](./793b_decision-log.md#793b204-the-memory-activity-line-is-the-mods-status-line) |
| Install path | a `skills-dir` plugin deployed by `engram update` | [793d13h Install path](#793d13h-install-path) | [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| JavaScript and Go | the mod only adapts; Go decides | [793d13i What runs in JavaScript and what in Go](#793d13i-what-runs-in-javascript-and-what-in-go) | [793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go) |

## 793d2 Added: flows

Each section is a flow spec: its canonical destination is `docs/specs/technical/<flow>-flow.md`. A diagram shows the current system as it will be after this change. IDs appear on arrows only: each arrow carries the ID of the behavior requirement it realizes and, where the flow's interaction requirement constrains it, the flow's ID. The called component's behavior is not repeated in a flow's requirement. Claude Code and the user are external actors, so an arrow to or from them carries the flow's ID.

### 793d2a Moment delivery

Canonical destination: `docs/specs/technical/moment-delivery-flow.md` (title `Moment Delivery Flow - Technical Spec`).

When Claude Code fires an event at which the working agent may need memory, the Memory Mod hands the event to the runner. The runner asks the detector whether the event is a moment; if it is, the delivery component asks the vault, drops what the receiving agent holds, and returns text, which the mod adds to the agent's context through the channel the event offers.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant Mod as Memory Mod
    participant Run as Auto Job Runner
    participant Det as Moment Detector
    participant Del as Memory Delivery
    participant Qry as Query Engine

    CC->>Mod: 793d2a, 793d3a event: prompt, tool result, failed tool, subagent start
    Mod->>Run: 793d2a, 793d3b, 793d4a moment job with the event, the receiving agent's context and its ledger
    Run->>Det: 793d5a the event
    Det-->>Run: 793d5a a moment with its text and phrases, or none
    alt a moment
        Run->>Del: 793d6a, 793d6b the moment and the agent's context
        Del->>Qry: 792b24c1 query with the moment's text and phrases
        Qry-->>Del: 792b24c1 the items
        Del-->>Run: 793d6a, 793d6c delivery text and item ids, empty when nothing remains
        Run-->>Mod: 793d4a done with the delivery
        alt a delivery
            Mod-->>CC: 793d2a, 793d3a the text, in the event's context channel
        else nothing remains
            Mod-->>CC: 793d2a, 793d3a the event unchanged
        end
    else no moment
        Run-->>Mod: 793d4a done with no delivery
        Mod-->>CC: 793d2a, 793d3a the event unchanged
    end
```

Requirement: 793d2a The Memory Mod MUST, for each event of [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), build the moment job from the event and the receiving agent's context and send it to the runner, and MUST wait for the result for at most `momentDeadlineMs` before letting the event continue. When the result is `done` with no delivery, because the detector found no moment or because nothing remained after the receiving agent's context was subtracted, the mod MUST let the event continue unchanged. When the result carries a delivery, the mod MUST add its text through the channel of the event ([793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through)), MUST record the delivered ids in the receiving agent's ledger, and MUST let the event continue otherwise unchanged. The mod MUST NOT ask the runner twice for one event, and MUST NOT start another job for a delivery. A failure on any step is the flow of [793d2c Automatic memory failure](#793d2c-automatic-memory-failure).

Owner: Memory Mod ([793d3 Added: memory-mod](#793d3-added-memory-mod)).

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios).

#### 793d2a1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2a1a A prompt that is a moment is delivered with the prompt
  Given the runner returns a delivery for a prompt
  When Claude Code fires prompt.submit for that prompt
  Then the mod continues the event with the delivery text as one extra context block
  And the prompt text is the text it received
  And the delivered ids are in the main agent's ledger

Scenario: 793d2a1b No moment lets the event continue unchanged
  Given the runner returns done with no delivery because the detector found no moment
  When Claude Code fires a tool-result event
  Then the mod continues the event unchanged
  And the main agent's ledger is unchanged

Scenario: 793d2a1c Nothing remaining lets the event continue unchanged
  Given the runner returns done with no delivery because every item is already in the agent's context
  When Claude Code fires prompt.submit
  Then the mod continues the event unchanged and adds no context

Scenario: 793d2a1d The wait is bounded
  Given the runner takes longer than momentDeadlineMs
  When Claude Code fires prompt.submit
  Then the event continues after momentDeadlineMs with no delivery
  And the flow of 793d2c runs

Scenario: 793d2a1e One event is one job
  Given any event of 793d3a
  When the mod handles it
  Then the runner receives exactly one moment job for it
  And a delivery starts no further job

Scenario: 793d2a1f A tool result carries its delivery next to the result
  Given the runner returns a delivery for a Skill call of route
  When Claude Code fires the tool-result event
  Then the mod returns the delivery as additional context of that event
  And the tool result is unchanged
```

### 793d2b Turn memory work

Canonical destination: `docs/specs/technical/turn-memory-work-flow.md` (title `Turn Memory Work Flow - Technical Spec`).

When a top-level turn ends, or a subagent stops, the mod lets the event continue at once and queues the System's work for it: judging and writing lessons, conclusions and dispatch outcomes; bringing the chunk index up to date; settling pending offers; and showing the user what happened. The model requests the runner asks for are relayed by the mod.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant Mod as Memory Mod
    participant Run as Auto Job Runner
    participant Cap as Auto Capture
    participant Out as Dispatch Outcome Recorder
    participant Cur as Auto Curation
    participant Qry as Query Engine
    participant Aut as Note Authoring
    participant Amd as Note Amendment
    participant Ing as Ingestion Source Discovery
    participant User as User

    CC->>Mod: 793d2b, 793d3d turn ended, or subagent stopped
    Mod-->>CC: 793d2b, 793d3d the event continues at once
    Note over Mod: the work is queued and runs one item at a time
    Mod->>Run: 793d2b, 793d4a capture job for the turn or the subagent
    Run->>Out: 793d8a, 793d8b the report rows of the window
    Out->>Aut: 792b12c3 write one note per new row
    Aut-->>Out: 792b12c3 the path
    Out-->>Run: 793d8a the outcome events
    Run->>Cap: 793d7a the window, redacted, as an extraction request
    alt a request is needed
        Cap-->>Run: 793d7a the extraction request
        Run-->>Mod: 793d4a need: the request
        Mod->>CC: 793d2b, 793d3c the model request, relayed
        CC-->>Mod: 793d2b, 793d3c the reply
        Mod->>Run: 793d4a the answers and the state
        Run->>Cap: 793d7b the reply as candidates
        Cap->>Qry: 792b24c1 query by each candidate's situation
        Qry-->>Cap: 792b24c1 the nearest notes
        Cap-->>Run: 793d7b the reconcile request
        Run-->>Mod: 793d4a need: the request
        Mod->>CC: 793d2b, 793d3c the model request, relayed
        CC-->>Mod: 793d2b, 793d3c the reply
        Mod->>Run: 793d4a the answers and the state
        Run->>Cap: 793d7c the reconcile replies
        alt a candidate is covered
            Cap->>Amd: 792b14c9 record a use of the existing note
            Amd-->>Cap: 792b14c9 done
        else a candidate is absent
            Cap->>Aut: 792b12c3 write the note
            Aut-->>Cap: 792b12c3 the path, or the parent-queued notice
        end
        Cap-->>Run: 793d7c, 793d7d the events and notices
    else the window is empty or holds no candidate
        Cap-->>Run: 793d7a no request
    end
    Run-->>Mod: 793d4a done with the events and notices
    Mod->>Ing: 792b17c3 engram ingest --auto
    Ing-->>Mod: 792b17c3 the index is up to date
    Mod->>Run: 793d4a curation job
    Run->>Cur: 793d9a, 793d9b settle the pending offers
    alt a pending offer exists
        Cur->>Amd: 792b14c14 settle by the judged outcome
        Amd-->>Cur: 792b14c14 done
    else none
        Note over Cur: finishes with no request
    end
    Cur-->>Run: 793d9a the curation events
    Run-->>Mod: 793d4a done with the events
    Mod->>User: 793d2b, 793d3f the memory-activity line, or nothing
```

Requirement: 793d2b The Memory Mod MUST, on the end of a top-level turn and on the stop of a subagent, return from the event handler without waiting for any of the work below, and MUST queue the work in the order: the capture job for the event; then, once per top-level turn after its capture, the index update, one curation job and the summary. The mod MUST run one queued item at a time. A capture job MUST be run to completion by relaying every judgement request the runner returns, up to `maxRounds` rounds; a capture that needs no request (an empty window, or no candidate) finishes at once and the queue continues with the next item. The mod MUST add the `events` and `notices` of every `done` to the counts since the previous summary, MUST run `engram ingest --auto` once per top-level turn after that turn's captures, and MUST run the curation job after the index update whether or not the vault holds a pending offer (a curation job with none finishes with no request). After the work of a top-level turn has finished, the mod MUST show the memory-activity line of [793d3f Memory-activity line](#793d3f-memory-activity-line) and MUST show nothing when the turn's work produced no event, notice or warning. A step that fails is the flow of [793d2c Automatic memory failure](#793d2c-automatic-memory-failure), and the queue then continues with the next step.

Owner: Memory Mod ([793d3 Added: memory-mod](#793d3-added-memory-mod)).

Supports [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3p1a A dispatch's outcome note is written from its report row](./793a_product-spec.md#793a3p1-uat-scenarios), [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a4b1d Slow capture does not stall the session](./793a_product-spec.md#793a4b1-uat-scenarios), [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios).

#### 793d2b1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2b1a The turn ends without waiting for the work
  Given the capture of a turn takes 20 seconds
  When Claude Code fires the end of the turn
  Then the mod's handler returns before the capture starts
  And a prompt submitted immediately after is handled without waiting for the capture

Scenario: 793d2b1b A lesson is captured, then indexed, then summarized
  Given a turn whose capture writes one note
  When the queue runs the turn's work
  Then the runner receives the capture job, then the mod runs engram ingest --auto, then the summary is shown
  And the summary counts one note added

Scenario: 793d2b1c Judgement requests are relayed until the job is done
  Given the runner answers the first round with a need and the second with another need
  When the mod runs the capture job
  Then the mod relays each request as one model completion and sends its reply back with the state
  And the job ends with the runner's done

Scenario: 793d2b1d A subagent's stop queues its capture
  Given a subagent stops
  When Claude Code fires the subagent's stop
  Then the handler returns at once and a capture job for that subagent is queued before the turn's job

Scenario: 793d2b1e A curation job runs after the index update
  Given a turn whose capture and index update have finished
  When the queue continues
  Then a curation job is sent, in rounds as above, before the summary

Scenario: 793d2b1f A curation job with nothing pending finishes without a request
  Given the runner answers the curation job with done and no event
  When the queue continues
  Then the mod relays no model request and continues to the summary

Scenario: 793d2b1i A capture that needs no request lets the queue continue
  Given the runner answers the capture job with done and no event because the window is empty
  When the queue continues
  Then no model request is relayed and the index update runs next

Scenario: 793d2b1g A turn without events shows nothing
  Given a turn whose work delivered, wrote, recorded and curated nothing
  When the turn's work has finished
  Then no memory-activity line is shown

Scenario: 793d2b1h The round limit ends a job
  Given the runner returns a need on every round
  When the mod has relayed maxRounds rounds
  Then the mod ends the job as an error and the flow of 793d2c runs
```

### 793d2c Automatic memory failure

Canonical destination: `docs/specs/technical/automatic-memory-failure-flow.md` (title `Automatic Memory Failure Flow - Technical Spec`).

Whatever step of the System fails, times out or is unavailable, the working agent's event continues as if the System were absent, and the user, not the agent, is told.

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant Mod as Memory Mod
    participant Run as Auto Job Runner
    participant User as User

    CC->>Mod: 793d2c, 793d3a an event of the System
    Mod->>Run: 793d4a the job
    alt the vault cannot be read, the program cannot start, a deadline passes, or any step throws
        Run-->>Mod: 793d4b a failure with its reason, or no result
        Mod-->>CC: 793d2c, 793d3e the event continues unchanged
        Mod->>User: 793d2c, 793d3f a warning with where and why
    else the job finds nothing to do
        Run-->>Mod: 793d4a done with no event
        Mod-->>CC: 793d3a the event continues unchanged
    end
```

Requirement: 793d2c The Memory Mod MUST, when a step of the System fails because the vault cannot be read or the `engram` program cannot start (`unavailable`), because a deadline of [793d3c Judgement relay and deadlines](#793d3c-judgement-relay-and-deadlines) passed (`timeout`), or because the step threw, returned output the mod cannot read, or ended a job as an error (`error`), let the working agent's event continue exactly as it would without the System, with no error, text or delay beyond the deadline reaching the agent, and with no user input lost. The mod MUST then show the user one warning on the surface of [793d3f Memory-activity line](#793d3f-memory-activity-line) naming the step (the moment's event, or capture, index or curation) and the reason, and MUST continue the queue. The mod MUST NOT show a warning when a job finds nothing to do.

Owner: Memory Mod ([793d3 Added: memory-mod](#793d3-added-memory-mod)).

Supports [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios).

#### 793d2c1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2c1a An unreadable vault is a skipped moment with a warning
  Given the runner reports that the vault cannot be read
  When Claude Code fires prompt.submit
  Then the prompt continues with its text and no added context
  And the status line shows a warning that automatic memory was skipped because it is unavailable
  And the warning is in no context block added to the agent

Scenario: 793d2c1b A missing program is unavailable
  Given the engram program cannot start
  When any event of 793d3a is handled
  Then the event continues unchanged and the warning gives unavailable

Scenario: 793d2c1c A deadline is a timeout
  Given the runner does not answer within momentDeadlineMs
  When Claude Code fires a tool-result event
  Then the event continues unchanged after momentDeadlineMs
  And the warning gives timeout

Scenario: 793d2c1d A throw is an error and the next prompt is normal
  Given the capture of a turn throws
  When the turn ends and the user submits the next prompt
  Then the prompt is handled normally
  And the warning gives error

Scenario: 793d2c1e A background failure does not stop the queue
  Given the index update of a turn fails
  When the queue continues
  Then the curation and the summary of the turn still run
  And the warning names the index and the reason

Scenario: 793d2c1f Nothing matched is not a failure
  Given the runner returns done with no delivery
  When the event continues
  Then no warning is shown
```

### 793d2d Mod install

Canonical destination: `docs/specs/technical/mod-install-flow.md` (title `Mod Install Flow - Technical Spec`).

`engram update` deploys the mod with the other agent instructions, for the harness that loads mods and for no other, and does nothing for a preview.

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant Upd as Self Installer
    participant Syn as Harness Deploy Sync

    Dev->>Upd: 793d2d engram update, optionally --dry-run
    Upd->>Syn: 792b29c1 detect the harnesses
    Syn-->>Upd: 792b29c1 the harnesses present
    Upd->>Syn: 793d10a the mod deploy for the harnesses that load mods
    alt Claude Code is present and the run is not a preview
        Syn-->>Upd: 793d10a the mod deployed or unchanged
    else a preview, or no harness that loads mods
        Syn-->>Upd: 793d10b nothing written
    end
    Upd-->>Dev: 793d2d, 792b28c12 the report, with the mod's row where one applies
```

Requirement: 793d2d The Self Installer MUST, during an update, ask Harness Deploy Sync to deploy the mod after the skills and guidance of the same harness, MUST ask it only for the harnesses that were detected, and MUST include in the report the row Harness Deploy Sync returns for the mod. For a preview (`--dry-run`) it MUST ask for the plan and no write, and MUST NOT change any other step of the update. The mod's deploy MUST NOT change the exit status of an update unless Harness Deploy Sync reports a failure.

Owner: Self Installer ([Self Installer](../../specs/technical/self-installer.md)).

Supports [793a4h1a Update installs the Integration for the harness it updates](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1c A preview installs none of the Integration](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1d One update serves both harnesses differently](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1l Running update twice changes nothing](./793a_product-spec.md#793a4h1-uat-scenarios), [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios), [793a5c1b A Pi session on a machine whose Claude Code has the Integration](./793a_product-spec.md#793a5c1-uat-scenarios).

#### 793d2d1 Aggregate-flow Scenarios

```gherkin
Scenario: 793d2d1a Claude Code gets the mod and Pi is not created
  Given ~/.claude exists and ~/.pi does not
  When the user runs engram update
  Then Harness Deploy Sync is asked to deploy the mod for Claude Code only
  And ~/.pi is not created
  And the report has a row for the mod

Scenario: 793d2d1b Both harnesses are updated and only Claude Code gets the mod
  Given ~/.claude and ~/.pi both exist
  When the user runs engram update
  Then the mod is deployed for Claude Code
  And no mod file is written under ~/.pi

Scenario: 793d2d1c A preview writes nothing
  Given a machine with no mod
  When the user runs engram update --dry-run
  Then Harness Deploy Sync is asked for the plan only
  And no mod file exists afterwards

Scenario: 793d2d1d A second update changes nothing
  Given an update has deployed the mod
  When the user runs engram update again
  Then the mod's row reports it unchanged

Scenario: 793d2d1e A failed mod deploy is reported
  Given Harness Deploy Sync reports a failure for the mod
  When the update finishes
  Then the report shows the failure and the exit status is the update's failure status
```

## 793d3 Added: memory-mod

Canonical destination: `docs/specs/technical/memory-mod.md` (title `Memory Mod - Technical Spec`).

- Ownership: repository component, at the Claude Code mods boundary (the hooks module and the mods API it calls).
- Responsibility: turns Claude Code events into System jobs and job results into effects on Claude Code: context for the agent, a status line for the user, background execution and the relay of model requests. It owns routing, ordering, gating, continuation and value propagation for every flow of [793d2 Added: flows](#793d2-added-flows); it decides nothing about what a moment is, what is delivered or what is written.
- Current realization (informational): `agent-instructions/mods/engram-memory/` (`.claude-plugin/plugin.json`, `hooks/hooks.json`, `hooks/register.js`, `tests/*.test.ts`).
- Qualification: responsibility, adapting events to jobs and results to effects is stated without naming a function; ownership, its routing, gating and containment are distinct from the Go components' decisions; independent verification, `claude plugin test` fires each event at the loaded mod with the world beneath it stubbed (`process.run`, `model.complete`, `ui.status`, the clock) and `claude plugin validate` lists its hooks and calls; stability, the contract survives any change to the Go command's internals; necessity, it is the only way the System sees or reaches a Claude Code session.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

Named constants of the mod (initial values, tuned under #850): `momentDeadlineMs` 25000, `judgeDeadlineMs` 60000, `roundDeadlineMs` 120000, `ingestDeadlineMs` 300000, `maxRounds` 4.

### 793d3a Events, channels and pass-through

Requirement: 793d3a The Memory Mod MUST register handlers for the events of the table below and for no other event, MUST forward an event to the runner only under the condition shown, and MUST add a delivery to Claude Code only through the channel shown. Every handler MUST end by continuing the event it received (`next(e)`) with, at most, the channel's context added: the mod MUST NOT register a handler that denies, drops or answers an event, and MUST NOT change a prompt's text, a tool call's input, a permission decision or a model. A delivery MUST be added as one block that is separate from the prompt and from the tool result.

| Event | Forwarded when | Channel |
| --- | --- | --- |
| `prompt.submit` | always, whatever the prompt's origin | an extra `context` entry after the prompt as typed |
| `classic.PostToolUse` | the event has no `agent_id` and the tool is `Skill`, `Agent`, `Edit`, `Write`, `MultiEdit` or `NotebookEdit` | `additionalContext` |
| `classic.PostToolUseFailure` | the event has no `agent_id` | `additionalContext` |
| `agent.spawn` | always | none: records the spawn ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |
| `classic.SubagentStart` | always | `additionalContext` |
| `classic.SubagentStop` | always | none: queues the subagent's capture ([793d2b Turn memory work](#793d2b-turn-memory-work)) |
| `turn.start` | no `agentId` | none: resets the turn counters ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |
| `turn.complete` | no `agentId` | none: queues the turn's work ([793d2b Turn memory work](#793d2b-turn-memory-work)) |
| `session.compact` | always | none: after the compaction, empties the main ledger ([793d3b State kept for the System](#793d3b-state-kept-for-the-system)) |

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2k1c An intentional recall by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2k1e An intentional engram query by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios), [793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d3a1 Contract Scenarios

```gherkin
Scenario: 793d3a1a The mod registers the listed events and none that can refuse
  Given the loaded mod
  When claude plugin validate lists its hooks
  Then the hooks are exactly the events of the table
  And no handler returns deny, drop or its own result in place of next(e)

Scenario Outline: 793d3a1b A delivery goes only through its channel
  Given the runner returns a delivery for a <event>
  When the mod handles the event
  Then the delivery text is added only as <channel>
  And the prompt text, the tool input and the permission decision are the ones the event carried
  Examples:
    | event                   | channel                                  |
    | prompt.submit           | one extra context entry                  |
    | classic.PostToolUse     | one additionalContext entry              |
    | classic.PostToolUseFailure | one additionalContext entry           |
    | classic.SubagentStart   | one additionalContext entry              |

Scenario Outline: 793d3a1c Only the listed tool calls and agents are forwarded
  Given a classic.PostToolUse event for <tool> with agent_id <agent>
  When the mod handles it
  Then the runner <receives> a moment job for it
  Examples:
    | tool  | agent   | receives          |
    | Skill | absent  | receives          |
    | Agent | absent  | receives          |
    | Edit  | absent  | receives          |
    | Read  | absent  | does not receive  |
    | Edit  | a1      | does not receive  |

Scenario: 793d3a1d Every origin of a prompt is forwarded
  Given prompts from the composer, a task notification, a scheduled trigger and a peer session
  When Claude Code fires prompt.submit for each
  Then the runner receives a moment job for each

Scenario: 793d3a1e An intentional memory command is not touched
  Given the agent runs engram query or the learn skill in a turn
  When Claude Code fires the tool events
  Then the mod continues each tool event with no change to its input, its permission decision or its result
```

### 793d3b State kept for the System

Requirement: 793d3b The Memory Mod MUST keep, in memory: for each agent a ledger of the item ids it delivered to that agent; for the main agent the number of edit-tool calls and the number of failure lookups in the current turn, set to zero at each `turn.start` (the edit count is raised by one for every edit-tool result of the main agent, after that result's job is built, and the failure-lookup count by one for every failure job sent); and for each `agent.spawn` the task prompt, the description and the fork flag, until the matching `classic.SubagentStart`, matched by the description `$.agent.list()` reports for the starting agent and otherwise by spawn order. A moment job MUST carry the receiving agent's context as a list of texts and its ledger: for the main agent the text of every message of `$.session.messages()` and of every tool result in them; for a starting subagent its task prompt, and for a fork the main agent's texts as well. The mod MUST add the delivered ids of a delivery to the receiving agent's ledger, and MUST empty the main ledger after the main agent is compacted.

Supports [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios).

#### 793d3b1 Contract Scenarios

```gherkin
Scenario: 793d3b1a A delivered id is in the ledger of the agent it was delivered to
  Given a delivery of note X to the main agent
  When the next moment job for the main agent is built
  Then its ledger holds X
  And the ledger of a subagent does not

Scenario: 793d3b1b The turn counters reset at the start of a turn
  Given the main agent made one edit and one failure lookup in a turn
  When Claude Code fires turn.start
  Then the next moment job carries zero edits and zero failure lookups

Scenario: 793d3b1c A subagent start is matched to its spawn
  Given two agent.spawn events with descriptions "add tests" and "fix lint"
  When classic.SubagentStart fires for the agent that $.agent.list() describes as "fix lint"
  Then the moment job carries the task prompt of the "fix lint" spawn

Scenario: 793d3b1d A subagent's context is its task, and a fork's is the parent's too
  Given a spawn with fork false and one with fork true
  When each subagent starts
  Then the first job's context is the task prompt alone
  And the second job's context is the task prompt and the main agent's texts

Scenario: 793d3b1e Compaction empties the main ledger
  Given the main ledger holds note X
  When the main agent is compacted
  Then the next moment job for the main agent carries an empty ledger
```

### 793d3c Judgement relay and deadlines

Requirement: 793d3c The Memory Mod MUST run every job with one loop: send the job to the runner; while the result is `need`, relay each request as one `$.model.complete` call with exactly the request's model, system, prompt and maxTokens, bounded by `judgeDeadlineMs`, and send the replies back with the result's `state`; and stop at `done`, after `maxRounds` rounds, or at a failure. It MUST accept a reply given as a string or as an object with `isAnswered` and `text`, and MUST treat an unanswered reply as a failure. It MUST bound a moment job's runner call by `momentDeadlineMs`, a capture or curation round's by `roundDeadlineMs` and the index update by `ingestDeadlineMs`. A relayed call MUST carry nothing but the request's own fields, and MUST NOT be submitted as a prompt, start an agent or call a tool.

Supports [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios), [793a4f1b A session of its own receives no delivery](./793a_product-spec.md#793a4f1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d3c1 Contract Scenarios

```gherkin
Scenario: 793d3c1a A request is relayed with its own fields only
  Given the runner returns a need with one request
  When the mod relays it
  Then $.model.complete is called once with that request's model, system, prompt and maxTokens
  And no prompt is submitted and no agent is started

Scenario: 793d3c1b Either reply shape is accepted
  Given a model reply that is the string "ok" and another that is an object with isAnswered true and text "ok"
  When the mod sends each back
  Then the runner receives the answer text "ok" both times

Scenario: 793d3c1c An unanswered reply is a failure
  Given a model reply with isAnswered false
  When the mod relays it
  Then the job ends as an error and no further round is sent

Scenario: 793d3c1d Each call has its deadline
  Given a moment job, a capture round, a curation round and the index update
  When each is run
  Then it is bounded by momentDeadlineMs, roundDeadlineMs, roundDeadlineMs and ingestDeadlineMs respectively

Scenario: 793d3c1e The state is returned with the answers
  Given a need that carries state S
  When the mod sends the answers
  Then the next call carries the same state S and the answers by request id
```

### 793d3d Background queue

Requirement: 793d3d The Memory Mod MUST hold the System's background work in one in-memory first-in-first-out queue, run one item at a time, and start each item from a timer callback and not from the handler that queued it. The handlers of `classic.SubagentStop` and `turn.complete` MUST queue their work and return without waiting for it. At a top-level turn's end the mod MUST queue, in this order, the turn's capture job, the index update, one curation job and the summary; a subagent's capture job is queued at its stop, so it runs before the turn's items. A failed item MUST NOT remove the items after it.

Supports [793a4b1d Slow capture does not stall the session](./793a_product-spec.md#793a4b1-uat-scenarios), [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios).

#### 793d3d1 Contract Scenarios

```gherkin
Scenario: 793d3d1a The handler does not wait
  Given a capture that takes 20 seconds
  When the end of a turn is handled
  Then the handler returns before the capture job is sent

Scenario: 793d3d1b Items run one at a time in order
  Given a subagent's capture and a turn's work are queued
  When the queue runs
  Then the subagent's capture finishes before the turn's capture starts
  And the turn's items run in the order capture, index, curation, summary

Scenario: 793d3d1c A failed item does not stop the queue
  Given the turn's capture fails
  When the queue continues
  Then the index update, the curation job and the summary still run

Scenario: 793d3d1d Work starts from a timer
  Given an item is queued
  When it starts
  Then it starts from a $.clock.after callback and not inside an event handler
```

### 793d3e Containment

Requirement: 793d3e The Memory Mod MUST catch every failure of its own work (a rejection, a throw, a result it cannot read, a deadline) so that the event it is handling continues unchanged, MUST register a `.catch` handler on every handler that can add context so that a skipped handler is also caught, MUST classify the failure as `unavailable` (the `engram` program cannot start, or the runner reports the vault cannot be read), `timeout` (a deadline passed) or `error` (anything else), and MUST record a warning naming the step and the reason for [793d3f Memory-activity line](#793d3f-memory-activity-line). A failure MUST NOT reach the agent as an error, a denied tool call, a dropped prompt or text in any context block, and MUST NOT lose the user's input.

Supports [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios).

#### 793d3e1 Contract Scenarios

```gherkin
Scenario Outline: 793d3e1a A failure continues the event and is classified
  Given <cause> while the mod handles a prompt.submit event
  When the handler finishes
  Then the prompt continues with its text and no context entry
  And the recorded warning is <class> for the moment
  Examples:
    | cause                                             | class       |
    | the engram program cannot start                   | unavailable |
    | the runner reports the vault cannot be read       | unavailable |
    | the runner does not answer within momentDeadlineMs | timeout    |
    | the runner's output is not a JSON document        | error       |
    | the handler throws                                | error       |

Scenario: 793d3e1b A skipped handler is caught
  Given a handler that adds context fails after calling next
  When Claude Code runs its catch handler
  Then the event result stands and a warning is recorded

Scenario: 793d3e1c A failure never reaches the agent
  Given any failure of the System
  When the agent's context is inspected
  Then it holds no error text, no warning and no denied or dropped event
```

### 793d3f Memory-activity line

Requirement: 793d3f The Memory Mod MUST keep, since the previous summary, the sum over the main agent and all subagents of the `events` of every job (`delivered`, `added`, `amended`, `outcomes`, `curated`), the `notices` and the warnings recorded under [793d3e Containment](#793d3e-containment). After a top-level turn's work has finished it MUST set the status line, with `$.ui.status`, to one line: the non-zero counts in the order delivered, added, amended, outcomes, curated as `engram memory: 3 items delivered, 1 note added, 1 note amended, 1 dispatch outcome recorded, 2 offers curated`, then each notice, then each warning as `engram memory skipped (<step>): <unavailable|timed out|error>`; or clear it with `undefined` when there is no count, notice or warning; and then MUST reset the sums. The mod MUST NOT write this line with `$.ui.log`, in a prompt, in a context channel or in any result Claude reads. A work item's counts MUST be added to the sums once.

Supports [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1b Nothing is shown after a turn without events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1c The summary is not in the agent's context](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1e The counts cover the events since the previous summary](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios), [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios).

#### 793d3f1 Contract Scenarios

```gherkin
Scenario: 793d3f1a The line shows the counts of a turn with events
  Given jobs of a turn and its subagents reported 3 delivered, 1 added, 1 amended, 1 outcome and 2 curated
  When the turn's work has finished
  Then the status line is "engram memory: 3 items delivered, 1 note added, 1 note amended, 1 dispatch outcome recorded, 2 offers curated"

Scenario: 793d3f1b A turn without events clears the line
  Given the jobs of a turn reported no event, notice or warning
  When the turn's work has finished
  Then the status line is cleared with undefined

Scenario: 793d3f1c The line is not in the agent's context
  Given a line was shown after a turn
  When the next prompt is handled
  Then no context entry and no tool result added by the mod contains the line
  And $.ui.log is never called with it

Scenario: 793d3f1d The counts cover the events since the previous summary
  Given a summary was shown after one turn
  When the next turn reports one note added and nothing else
  Then the line is "engram memory: 1 note added"

Scenario: 793d3f1e The line waits for the work
  Given a turn whose capture adds a note after 20 seconds
  When the turn ends
  Then the line is set only after the capture has finished and it counts that note

Scenario: 793d3f1f A parent-queued notice is shown
  Given a job returned the notice "note saved, offer queued (parent unreachable)" and no counts
  When the turn's work has finished
  Then the line is "engram memory: note saved, offer queued (parent unreachable)"

Scenario: 793d3f1g A warning is shown with the counts
  Given a turn with one note added and a timed-out curation
  When the turn's work has finished
  Then the line is "engram memory: 1 note added; engram memory skipped (curation): timed out"
```

## 793d4 Added: auto-job-runner

Canonical destination: `docs/specs/technical/auto-job-runner.md` (title `Auto Job Runner - Technical Spec`).

- Ownership: repository component at the command boundary (standard input and output of `engram auto run`); the caller of [793d5 Added: moment-detector](#793d5-added-moment-detector), [793d6 Added: memory-delivery](#793d6-added-memory-delivery), [793d7 Added: auto-capture](#793d7-added-auto-capture), [793d8 Added: dispatch-outcome-recorder](#793d8-added-dispatch-outcome-recorder) and [793d9 Added: auto-curation](#793d9-added-auto-curation).
- Responsibility: turns one invocation into one round of one job. It reads the job, runs the components the job needs up to the next model request or to the end, and writes either the requests to relay with an opaque state, or the finished result. It keeps nothing between calls and reports a failure inside its result.
- Current realization (informational): `internal/auto/runner.go`; the command in `internal/cli/auto.go`, built by `cli.NewDeps` over `cli.Deps.Stdin`; no new `cli.Primitives` field.
- Qualification: responsibility, advancing a job one round is stated without naming a function; ownership, round sequencing, state and failure reporting are distinct from the components it calls; independent verification, it is tested over fakes of those components and a fixed job document; stability, the contract survives any change to a component; necessity, the mod reaches the Go side only through it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d4a A job in rounds

Requirement: 793d4a The Auto Job Runner MUST read one JSON document from standard input holding `job` (`moment`, `capture` or `curate`), `event` and, from the second round of a job, `answers` and `state`, MUST run the job for one round, and MUST write one JSON document to standard output and exit 0. The document MUST be `{ "need": [ { id, model, prompt, system?, maxTokens } ], "state": "<opaque>" }` when the job needs model replies, or `{ "done": { deliver, deliveredIds, moment, events, notices } }` when it is finished, with `events` the integer counts `delivered`, `added`, `amended`, `outcomes` and `curated`. A `moment` job MUST finish in its one round and MUST NOT return `need`. A `capture` job MUST call the Dispatch Outcome Recorder in its first round, before Auto Capture, so that a failed model request does not lose the outcomes the report rows already hold; a `curate` job MUST call Auto Curation. `state` MUST hold everything the next round needs, the events already counted included, and the runner MUST keep nothing between calls. It MUST write nothing else to standard output.

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3p1a A dispatch's outcome note is written from its report row](./793a_product-spec.md#793a3p1-uat-scenarios), [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios).

#### 793d4a1 Contract Scenarios

```gherkin
Scenario: 793d4a1a A moment job finishes in one round
  Given a moment job whose event is a moment with items
  When the runner runs it
  Then the output is a done with the delivery text, the delivered ids and the moment's kind
  And the output has no need

Scenario: 793d4a1b A moment job with nothing to deliver is a done with an empty delivery
  Given a moment job whose event is not a moment
  When the runner runs it
  Then the output is a done with deliver "" and zero events

Scenario: 793d4a1c A capture job asks for model replies and resumes from its state
  Given a capture job whose window holds a candidate lesson
  When the runner runs round one and then round two with the answers and the returned state
  Then round one returns a need with a state and round two continues from that state alone
  And a second process with no memory of round one gives the same round-two result

Scenario: 793d4a1d A finished job reports its events
  Given a capture job that writes one note and records one outcome
  When the job finishes
  Then events.added is 1, events.outcomes is 1 and the other counts are 0

Scenario: 793d4a1e The output is one document
  Given any job
  When the runner runs
  Then standard output holds exactly one JSON document and the exit status is 0
```

### 793d4b Failures

Requirement: 793d4b The Auto Job Runner MUST, when the vault cannot be read, write `{ "done": { "warning": { "why": "unavailable", "detail": … } } }`, and when any other step of the job fails, write `{ "done": { "warning": { "why": "error", "detail": … }, "events": …, "notices": … } }` with the events and notices of the steps that completed before the failure, and exit 0 in both cases. It MUST exit non-zero, writing the reason to standard error, only when its input is not a valid job document. A note written before a failure MUST still be counted in `events`.

Supports [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios).

#### 793d4b1 Contract Scenarios

```gherkin
Scenario: 793d4b1a An unreadable vault is unavailable
  Given a moment job and a vault that cannot be read
  When the runner runs
  Then the output is a done with warning why "unavailable" and exit status 0

Scenario: 793d4b1b A failing step is an error with the completed steps' events
  Given a capture job whose first note is written and whose second write fails
  When the runner runs the round
  Then the output is a done with warning why "error" and events.added 1

Scenario: 793d4b1c An invalid job document is refused
  Given standard input that is not a job document
  When the runner runs
  Then it exits non-zero with the reason on standard error and writes nothing to standard output
```

## 793d5 Added: moment-detector

Canonical destination: `docs/specs/technical/moment-detector.md` (title `Moment Detector - Technical Spec`).

- Ownership: repository component; a pure decision with no dependency.
- Responsibility: decides, from one event and the counters the mod keeps, whether the event is a moment, and if so what to ask the vault: a kind, a query text and phrases.
- Current realization (informational): `internal/auto/detect.go`.
- Qualification: responsibility, recognizing a moment from an event is stated without naming a function; ownership, the decision and the wording of the phrases are distinct from the runner's sequencing and the delivery's lookup; independent verification, it is a pure function tested on event values and by property tests; stability, the contract survives any change to the cues; necessity, every delivery starts from its decision.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d5a Cues

Requirement: 793d5a The Moment Detector MUST map an event to a moment by the table below, and to no moment for any event not in it. `task` is the text of the latest human-origin prompt in the event's context (for a prompt, the prompt itself). The query text MUST be cut to `textLimit` (2000) characters and each phrase to `phraseLimit` (300). The decision MUST depend on the event alone: the same event gives the same moment.

| Cue (event) | Moment | Query text | Phrases |
| --- | --- | --- | --- |
| a prompt that, trimmed, starts with `/please` | `please` | the text after `/please` | the ask; `orienting an ask given to please: <ask>` |
| a prompt of any other text or origin | `prompt` | the prompt | the prompt; `evaluating or ranking a proposed approach: <task>`; `declaring work done: <task>`; `diagnosing a failure I cannot immediately explain: <task>`; `starting to build a new approach: <task>` |
| `Skill` result with skill `please` | `please` | the Skill's args | the args; `orienting an ask given to please: <args>` |
| `Skill` result with skill `route`, or an `Agent` result, by the main agent | `dispatch` | the description and the first 500 characters of the prompt or args | the description; `routing a subagent dispatch and deciding its tier for <description>` |
| a subagent start | `subagent` | the task prompt | the task prompt; `doing a delegated unit of work: <description>` |
| an edit-tool result by the main agent when `edits` is 0 | `edit` | the file path and the first 500 characters of the new text | `<tool> <path>: <first line of the change>`; `starting to build: <task>` |
| a failed tool result by the main agent when `failureLookups` is 0 | `failure` | the first 500 characters of the error | the error's first line; `diagnosing an unexplained failure: <task>` |

A prompt whose trimmed text is empty, an edit when `edits` is above 0, a failure when `failureLookups` is above 0, an event of a subagent other than its start, a `Skill` other than `please` and `route`, and any other tool result MUST give no moment.

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios).

#### 793d5a1 Contract Scenarios

```gherkin
Scenario: 793d5a1a An ordinary prompt carries the four recall moments
  Given a prompt "rename package foo to bar" from the composer
  When the detector decides
  Then the moment is prompt
  And the phrases include the prompt and one phrase for each of endorsing, declaring done, diagnosing a failure and starting to build

Scenario Outline: 793d5a1b A prompt from any source is a moment
  Given a prompt "check the build" with origin <origin>
  When the detector decides
  Then the moment is prompt
  Examples:
    | origin            |
    | composer          |
    | task-notification |
    | scheduled-trigger |
    | peer              |

Scenario: 793d5a1c A please prompt is a please moment
  Given a prompt "/please add a flag to the cli"
  When the detector decides
  Then the moment is please
  And the query text is "add a flag to the cli"

Scenario: 793d5a1d A route skill result or an Agent call is a dispatch moment
  Given a Skill result for route with args "add tests for internal/cli" by the main agent
  When the detector decides
  Then the moment is dispatch
  And a phrase asks about routing a subagent dispatch and deciding its tier

Scenario: 793d5a1e A subagent's start is a subagent moment
  Given a subagent start with task prompt "add tests for internal/cli"
  When the detector decides
  Then the moment is subagent and the query text is the task prompt

Scenario: 793d5a1f Only the first edit and the first failure of a turn are moments
  Given an edit result with edits 0, an edit result with edits 1, a failure with failureLookups 0 and a failure with failureLookups 1
  When the detector decides for each
  Then the first and third are moments of kind edit and failure and the second and fourth are not

Scenario: 793d5a1g Events outside the table are not moments
  Given an empty prompt, a Read result, a Skill result for another skill and an Edit result by a subagent
  When the detector decides for each
  Then none is a moment

Scenario: 793d5a1h Long text is cut
  Given a prompt of 5000 characters
  When the detector decides
  Then the query text is 2000 characters and every phrase is at most 300

Scenario: 793d5a1i The decision depends on the event alone
  Given any event
  When the detector decides twice
  Then both decisions are equal
```

## 793d6 Added: memory-delivery

Canonical destination: `docs/specs/technical/memory-delivery.md` (title `Memory Delivery - Technical Spec`).

- Ownership: repository component; the caller of the [Query Engine](../../specs/technical/query-engine.md).
- Responsibility: turns a moment and the receiving agent's context into the delivery: it asks the vault once, keeps the relevant items, drops what the agent already holds, and writes the text that names each item's source and asks the agent to apply it.
- Current realization (informational): `internal/auto/deliver.go`, over the query path `runLocalQueryPayload` and the payload type of `internal/cli/merged_query.go`.
- Qualification: responsibility, producing a delivery from a moment and a context is stated without naming a function; ownership, relevance, presence and wording are distinct from the engine's matching and from the runner's sequencing; independent verification, it is tested over a fake query and context texts; stability, the contract survives any change to the engine's payload shape; necessity, no delivery is produced without it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

Constants: `deliveryMinScore` 0.25 (the engine's relevance floor), `deliveryItemLimit` 10.

### 793d6a Lookup and relevance

Requirement: 793d6a The Memory Delivery MUST ask the Query Engine once per moment with the moment's query text as `--text`, one phrase per moment phrase, the recency channel off and chunk content included, and MUST keep, in the engine's order, the items whose provenances include `direct`, `cluster_rep` or `trigger` and whose score is at least `deliveryMinScore`; MUST then remove the items present under [793d6b Presence](#793d6b-presence); and MUST keep at most `deliveryItemLimit` of what remains. It MUST write nothing to the vault.

Supports [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios).

#### 793d6a1 Contract Scenarios

```gherkin
Scenario: 793d6a1a One query with the moment's text and phrases
  Given a moment with a query text and three phrases
  When the delivery looks up
  Then the Query Engine receives one query with that text, those phrases, the recency channel off and chunk content on

Scenario: 793d6a1b Only relevant items are kept
  Given the engine returns a direct note at 0.6, a recent-only chunk at 0.9, an explore note at 0.5 and a direct note at 0.1
  When the delivery filters
  Then only the direct note at 0.6 remains

Scenario: 793d6a1c The limit applies after the present items are removed
  Given twelve relevant items of which three are present
  When the delivery filters
  Then ten items remain and none is a present one

Scenario: 793d6a1d A lookup writes nothing
  Given any moment
  When the delivery looks up
  Then no vault write is requested
```

### 793d6b Presence

Requirement: 793d6b The Memory Delivery MUST treat an item as present when its note file name or its chunk id occurs in any text of the receiving agent's context, or equals an id in the ledger it was given, and MUST NOT treat an item as present when it occurs in neither. An item that was delivered, retrieved or restored but whose text is no longer in the context MUST NOT be present.

Supports [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios).

#### 793d6b1 Contract Scenarios

```gherkin
Scenario: 793d6b1a A delivered item is present through the ledger
  Given the ledger holds note X and the context texts do not mention it
  When X comes back from the query
  Then X is not delivered

Scenario: 793d6b1b An item the agent retrieved is present through the context
  Given a tool result in the context shows note X by its file name
  When X comes back from the query
  Then X is not delivered

Scenario: 793d6b1c A parent's item is not present for a subagent
  Given note X is in the main agent's texts and the subagent's context is its task prompt
  When X comes back for the subagent
  Then X is delivered

Scenario: 793d6b1d An item that left the context is not present
  Given note X was delivered earlier, the context was compacted, and the ledger and the texts no longer hold X
  When X comes back from the query
  Then X is delivered again

Scenario: 793d6b1e A chunk is present by its id
  Given the context shows the chunk id "session-ab12#turn-7"
  When that chunk comes back from the query
  Then it is not delivered
```

### 793d6c Delivery text

Requirement: 793d6c The Memory Delivery MUST, when at least one item remains, return a text whose first line is `[engram memory]` followed by the instruction `Engram surfaced the vault items below for this moment. Apply what bears on your next step; disregard the rest.`, then each item as a line `--- note <file name> (<kind>)` or `--- chunk <id> (score <score to two decimals>)` followed by the item's content, and a last line `[end engram memory]`, and MUST return the ids of the items. The text MUST NOT ask the agent to run a command, to evaluate, approve or write a note, or to curate. When no item remains it MUST return the empty text, no ids and no instruction, and `events.delivered` MUST be the number of items in the text.

Supports [793a2d1a Delivered content names its source](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1b A delivered chunk names its id](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios), [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios).

#### 793d6c1 Contract Scenarios

```gherkin
Scenario: 793d6c1a A note names its file name and kind
  Given the item "474.2026-07-25.charge-one-reviewer-to-execute….md" of kind feedback
  When the text is written
  Then it holds "--- note 474.2026-07-25.charge-one-reviewer-to-execute….md (feedback)" followed by the note's content

Scenario: 793d6c1b A chunk names its id and score
  Given the chunk "session-ab12#turn-7" at score 0.614
  When the text is written
  Then it holds "--- chunk session-ab12#turn-7 (score 0.61)" followed by the chunk's content

Scenario: 793d6c1c The instruction asks only to apply
  Given any delivery
  When its text is read
  Then it starts with "[engram memory]" and the instruction, ends with "[end engram memory]", and holds no request to run a command, to evaluate, approve or write a note, or to curate

Scenario: 793d6c1d Nothing remaining gives nothing
  Given no item remains
  When the text is written
  Then the text is empty, there are no ids and the instruction is absent

Scenario: 793d6c1e The delivered count is the items
  Given a delivery of three items
  When the result is read
  Then events.delivered is 3 and the ids are the three items' file names or chunk ids
```

## 793d7 Added: auto-capture

Canonical destination: `docs/specs/technical/auto-capture.md` (title `Auto Capture - Technical Spec`).

- Ownership: repository component; the caller of the [Query Engine](../../specs/technical/query-engine.md), [Note Authoring](../../specs/technical/note-authoring.md) and [Note Amendment](../../specs/technical/note-amendment.md) and of the [Transcript Reader](../../specs/technical/transcript-reader.md) for a subagent's transcript.
- Responsibility: finds the lessons and sound conclusions in one turn or one subagent session and records those the vault does not hold: it forms the window, removes secrets and the System's own blocks, asks a model for candidates, reconciles each against the vault with a second request, and writes the absent ones with their provenance.
- Current realization (informational): `internal/auto/capture.go`, `internal/auto/redact.go`; the criteria are the learn skill's Step 2, the recall skill's Step 4 and its derived-note rules, taken from `agent-instructions/skills/*/SKILL.md` embedded in the binary at build time.
- Qualification: responsibility, recording the unrecorded lessons of a conversation is stated without naming a function; ownership, window, redaction, parsing, reconciliation and the write are distinct from the runner's rounds and from Note Authoring's write; independent verification, it is tested over fake replies, a fake query and a fake writer; stability, the contract survives a change of model or prompt wording; necessity, no lesson is captured without it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

Constants: `judgeModel` `haiku`, `extractMaxTokens` 2048, `candidateNotes` 5.

### 793d7a Extraction request

Requirement: 793d7a The Auto Capture MUST form the window of a main-agent turn as the last assistant message before the turn's last human prompt, that prompt, and everything after it, and the window of a subagent as its whole transcript read with the Transcript Reader, with the event's `last_assistant_message` appended when the transcript lacks it. It MUST remove from the window every block from a line `[engram memory]` through the next line `[end engram memory]`, MUST replace each credential, token and private key that the redactor's patterns match with `[redacted]` in the window and in every later request and note, and MUST build one request for `judgeModel` whose system text is the criteria named above with the instruction to return only JSON and run nothing, and whose prompt is the redacted window and the file names of the notes present in the agent's context. When the window is empty after the removals it MUST finish the job with no request.

Supports [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4c1e A secret in a correction is not persisted](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios).

#### 793d7a1 Contract Scenarios

```gherkin
Scenario: 793d7a1a A correction needs the message it corrects
  Given a turn whose prompt is "no, use the targ build system, not go test directly" and a preceding assistant message that ran go test
  When the window is formed
  Then it holds the preceding assistant message, the prompt and the turn's messages

Scenario: 793d7a1b A subagent's window is its transcript
  Given a subagent transcript in which the dispatching agent says "no, run targ test, not go test"
  When the window is formed
  Then it holds the whole transcript and the stop's last assistant message once

Scenario: 793d7a1c A secret never reaches a request
  Given a prompt containing the token "sk-ant-api03-AAAAAAAAAAAAAAAAAAAA"
  When the request is built
  Then the request holds "[redacted]" and not the token

Scenario: 793d7a1d The System's own blocks are removed
  Given a window that holds a block from "[engram memory]" through "[end engram memory]" inside a longer message
  When the request is built
  Then the request holds none of that block's text and keeps the rest of the message

Scenario: 793d7a1e An empty window asks for nothing
  Given a window that is empty after the removals
  When the job runs
  Then it finishes with no request and zero events

Scenario: 793d7a1f The request carries the criteria and no instruction to run anything
  Given any window
  When the request is built
  Then the system text holds the four kinds of lesson and the bar, and tells the model to return JSON only and run nothing
```

### 793d7b Candidates and the reconcile request

Requirement: 793d7b The Auto Capture MUST read the extraction reply as JSON with `lessons` and `conclusions`, and MUST end the job as an error when it cannot. It MUST keep as candidates each lesson of one of the four kinds (correction, explicit save-request, reversal, confirmed approach) that carries the fields its write needs, and each conclusion the reply marks sound and non-trivial with an inference of deduction, abduction or induction and a hedge of `none` or `probable`; it MUST drop a conclusion hedged below probable, and any lesson the reply does not mark confirmed. For each candidate it MUST query the vault by the candidate's situation and build one request, for `judgeModel`, that gives the candidate and the `candidateNotes` nearest notes and asks whether an existing note already states it (`covered`, naming the note) or not (`absent`). When no candidate remains it MUST finish the job with no further request.

Supports [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a3q1c A hunch is not persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d7b1 Contract Scenarios

```gherkin
Scenario: 793d7b1a A confirmed correction becomes a candidate
  Given a reply with one correction lesson that is confirmed and complete
  When the reply is read
  Then one candidate exists and one reconcile request lists it with the five nearest notes

Scenario: 793d7b1b A hunch is dropped
  Given a reply with a conclusion hedged "hunch"
  When the reply is read
  Then no candidate exists for it

Scenario: 793d7b1c A sound abduction is kept as probable
  Given a reply with a conclusion marked sound and non-trivial, inference abduction, hedge probable
  When the reply is read
  Then one candidate exists and its note text will mark the conclusion probable

Scenario: 793d7b1d An unreadable reply is an error
  Given a reply that is not JSON
  When the reply is read
  Then the job ends as an error

Scenario: 793d7b1e No candidate ends the job
  Given a reply with no lessons and no conclusions
  When the reply is read
  Then the job finishes with no further request and zero events
```

### 793d7c Reconcile and write

Requirement: 793d7c The Auto Capture MUST read each reconcile reply, and for a candidate judged covered by a named existing note MUST record a use of that note through Note Amendment and write nothing; for a candidate judged absent it MUST write one note through Note Authoring, of the kind the candidate names (feedback for a correction or a reversal, fact for an explicit save-request, feedback or runbook for a confirmed approach, fact or feedback for a conclusion, and also a question-and-answer note when a conclusion's body cites at least one `[[file name]]`), with the source that the learn skill defines for it followed by `; written automatically (session <session id>, turn <turn id>)`. It MUST write the candidates one at a time in the reply's order and MUST count `events.added` once for each note written. It MUST NOT write a note for a candidate it judged covered, MUST NOT amend, discard or delete a note other than to record a use, and MUST NOT write a credential, token or private key.

Supports [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1i A captured note records how it was written](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios), [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios), [793a4c1e A secret in a correction is not persisted](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios).

#### 793d7c1 Contract Scenarios

```gherkin
Scenario: 793d7c1a An absent correction is written with its provenance
  Given a candidate correction "use the targ build system, not go test directly" judged absent in session S, turn T
  When the write runs
  Then Note Authoring receives one feedback write whose source is the learn skill's source followed by "; written automatically (session S, turn T)"
  And events.added is 1

Scenario: 793d7c1b A covered candidate writes nothing
  Given a candidate judged covered by note N
  When the write runs
  Then no note is written and Note Amendment receives a recorded use of N

Scenario: 793d7c1c An earlier note by the agent covers the candidate
  Given the agent ran the learn skill in the turn and wrote a feedback note stating the same rule
  When the candidate is reconciled
  Then the vault holds one note for the rule after the job

Scenario: 793d7c1d Two captures of the same lesson give one note
  Given two capture jobs for the same lesson run one after the other
  When both have finished
  Then the vault holds exactly one note stating the lesson

Scenario: 793d7c1e A cited conclusion also gets a question-and-answer note
  Given a conclusion whose body cites [[a.md]] and [[b.md]], judged absent
  When the write runs
  Then one conclusion note and one question-and-answer note with contributors a.md and b.md are written

Scenario: 793d7c1f No credential is written
  Given a candidate whose text came from a redacted window
  When the write runs
  Then no written field holds a credential, token or private key

Scenario: 793d7c1g The System changes no other note
  Given any capture job
  When it finishes
  Then the only calls to Note Amendment are recordings of a use
```

### 793d7d Parent-queued notice

Requirement: 793d7d The Auto Capture MUST, when a write reports that the parent is unreachable and an offer is queued (`engram: parent unreachable (retry after …); N offer(s) queued`), count the note as written, add the notice `note saved, offer queued (parent unreachable)` to the result's `notices` once for the job, and MUST NOT write the note again or report the job as failed.

Supports [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios).

#### 793d7d1 Contract Scenarios

```gherkin
Scenario: 793d7d1a An unreachable parent is a notice and not a failure
  Given a write that reports "engram: parent unreachable (retry after 30s); 1 offer(s) queued"
  When the job finishes
  Then events.added is 1 and notices holds "note saved, offer queued (parent unreachable)"
  And the job has no warning and the vault holds one note for the lesson

Scenario: 793d7d1b The notice is given once per job
  Given two writes in one job that both report an unreachable parent
  When the job finishes
  Then notices holds the notice once
```

## 793d8 Added: dispatch-outcome-recorder

Canonical destination: `docs/specs/technical/dispatch-outcome-recorder.md` (title `Dispatch Outcome Recorder - Technical Spec`).

- Ownership: repository component; the caller of [Note Authoring](../../specs/technical/note-authoring.md).
- Responsibility: turns the rows of route's mini-report in a window into the evidence notes route defines ([792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note)), without a model.
- Current realization (informational): `internal/auto/outcome.go`.
- Qualification: responsibility, recording a dispatch from its report row is stated without naming a function; ownership, row parsing and the duplicate rule are distinct from the capture's judgement and from the write; independent verification, it is a table parser over fixed text plus a fake writer; stability, the contract survives a change to the capture; necessity, [793a3p Dispatch outcomes are recorded by the System](./793a_product-spec.md#793a3p-dispatch-outcomes-are-recorded-by-the-system) is delivered through it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d8a Rows to notes

Requirement: 793d8a The Dispatch Outcome Recorder MUST find in the assistant text of the window each row of route's mini-report table (columns work-kind, tier, model, why, outcome, escalation, duration, cost) and, for each row whose outcome is PASS or FAIL and whose work-kind, tier and outcome are valid (kebab-case work-kind; tier `cheap`, `mid` or `deep`; outcome `pass` or `fail`), MUST write one fact through Note Authoring with the slug `route-dispatch-<work-kind>`, the tags `work-kind/<k>`, `tier/<t>` and `outcome/<o>`, the situation `routing <work-kind> work`, the subject `<work-kind> dispatch at <tier> (<model>)`, the predicate `resolved as`, an object stating the verdict, why, escalation, duration and cost with the row's units, and the source `route dispatch record, <project>, <date>` followed by `; written automatically (session <session id>, turn <turn id>)`, where `<project>` is the base name of the session's working directory. It MUST count `events.outcomes` for each note written, and MUST write nothing for a dispatch whose row is not in the window.

Supports [793a3p1a A dispatch's outcome note is written from its report row](./793a_product-spec.md#793a3p1-uat-scenarios), [793a3p1c A missing row writes nothing](./793a_product-spec.md#793a3p1-uat-scenarios), [793a3a1i A captured note records how it was written](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d8a1 Contract Scenarios

```gherkin
Scenario: 793d8a1a A passing row becomes a tagged fact
  Given a row for work-kind go-package-implementation, tier mid, model sonnet, outcome PASS, duration 4 min, cost 45,231 tok
  When the window is read
  Then one fact is written with slug route-dispatch-go-package-implementation and tags work-kind/go-package-implementation, tier/mid and outcome/pass
  And its object states the verdict, the reason, the escalation, the duration and the cost
  And events.outcomes is 1

Scenario: 793d8a1b No row writes nothing
  Given a window that dispatched a subagent but holds no mini-report row
  When the window is read
  Then no note is written and events.outcomes is 0

Scenario: 793d8a1c An invalid row is skipped
  Given a row whose tier is "huge"
  When the window is read
  Then no note is written for it

Scenario: 793d8a1d A note records how it was written
  Given a written outcome note
  When its source is read
  Then it holds the route record source followed by "; written automatically (session S, turn T)"
```

### 793d8b No duplicate outcome

Requirement: 793d8b The Dispatch Outcome Recorder MUST NOT write a note for a row when the vault already holds a note with the same slug and the same three tags whose object holds the row's duration text and cost text, whoever wrote it.

Supports [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d8b1 Contract Scenarios

```gherkin
Scenario: 793d8b1a The orchestrator's own note is not duplicated
  Given the orchestrator already wrote the note for a dispatch of work-kind K whose duration and cost are in its object
  When the row of that dispatch is read
  Then no second note is written and events.outcomes is 0

Scenario: 793d8b1b A different dispatch of the same kind is recorded
  Given a note for work-kind K with another duration and cost
  When a row of K with this dispatch's duration and cost is read
  Then one note is written
```

## 793d9 Added: auto-curation

Canonical destination: `docs/specs/technical/auto-curation.md` (title `Auto Curation - Technical Spec`).

- Ownership: repository component; the caller of the [Query Engine](../../specs/technical/query-engine.md) and [Note Amendment](../../specs/technical/note-amendment.md).
- Responsibility: settles the vault's pending offers as the curate skill does, with model replies for the judgements and Note Amendment for the changes.
- Current realization (informational): `internal/auto/curate.go`; criteria are the curate skill's Step 2 table and its special case, embedded at build time.
- Qualification: responsibility, settling pending offers is stated without naming a function; ownership, the judgement requests and the choice of amend form are distinct from the amend itself and the runner's rounds; independent verification, it is tested over fake replies and a fake amender; stability, the contract survives a change to the curate skill's wording; necessity, [793a2l Indexing and curation happen without the agent](./793a_product-spec.md#793a2l-indexing-and-curation-happen-without-the-agent) is delivered through it.
- Supported capabilities: [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec).

### 793d9a Settling offers

Requirement: 793d9a The Auto Curation MUST find the vault's pending offers and, when there are none, finish with no request. For each offer that is not a skill-runbook registration it MUST build one request for `judgeModel` with the offer and its candidates (the note its `offer.for` names, else the notes the query by its situation returns), and MUST apply the reply with Note Amendment by the curate skill's table: `covered` records a use of the existing note and discards the offer into it; `near` discards the offer into the existing note and then amends that note with the offer's new claim; `absent` clears the offer's pending marker; `rejected` discards the offer without `--into`. It MUST pass the offer's exchange hash when it has one, and when the hash check refuses it MUST leave the offer pending and continue. It MUST count `events.curated` for each offer settled and `events.amended` for each existing note amended by a `near`, and MUST NOT ask the agent to do any of it.

Supports [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios), [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios).

#### 793d9a1 Contract Scenarios

```gherkin
Scenario: 793d9a1a A covered offer is folded into its note
  Given a pending offer that an existing note already covers
  When the curation runs
  Then a use of the existing note is recorded, the offer is discarded into it and no longer pending
  And events.curated is 1 and events.amended is 0

Scenario: 793d9a1b An absent offer is accepted
  Given a pending offer no existing note addresses
  When the curation runs
  Then its pending marker is cleared and events.curated is 1

Scenario: 793d9a1c A near offer is folded and its claim added
  Given a pending offer that overlaps a note and adds a claim
  When the curation runs
  Then the offer is discarded into the note and the note is amended with the claim
  And events.curated is 1 and events.amended is 1

Scenario: 793d9a1d A rejected offer is discarded without into
  Given a pending offer a note contradicts
  When the curation runs
  Then the offer is discarded without --into and events.curated is 1

Scenario: 793d9a1e A refused hash leaves the offer pending
  Given an offer whose exchange hash check is refused
  When the curation runs
  Then the offer stays pending, the other offers are settled and events.curated counts the settled ones only

Scenario: 793d9a1f No offer means no request
  Given a vault with no pending offer
  When the curation runs
  Then it finishes with no request and zero events

Scenario: 793d9a1g Only pending notes are touched
  Given a vault with a pending offer and notes the agent wrote with the learn skill
  When the curation runs
  Then no note that is not pending or named as an existing note by a reply is changed
```

### 793d9b Skill-runbook registration offers

Requirement: 793d9b The Auto Curation MUST settle a pending note that carries a `skill_hash` by building one request for `judgeModel` from the skill's body that asks for the note's `situation`, `done_when`, `triggers` and `red_flags`, amending the note with the reply through Note Amendment and clearing its pending marker. It MUST NOT judge such a note covered, near or rejected and MUST NOT discard it. It MUST count `events.curated` for each such note settled.

Supports [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios).

#### 793d9b1 Contract Scenarios

```gherkin
Scenario: 793d9b1a A registration offer is authored and accepted
  Given a pending runbook note with a skill_hash and no situation
  When the curation runs
  Then the note is amended with the replied situation, done_when, triggers and red_flags, and its pending marker is cleared
  And events.curated is 1

Scenario: 793d9b1b A registration offer is never discarded
  Given a pending runbook note with a skill_hash
  When the curation runs with any reply
  Then the note is not discarded or folded into another note
```

## 793d10 Added: harness-deploy-sync (mod deploy)

Canonical destination: `docs/specs/technical/harness-deploy-sync.md` (the existing component; the two requirements below are Added to it).

The component's orientation is unchanged ([792b29a Orientation](../../specs/technical/harness-deploy-sync.md#792b29a-orientation)). Responsibility added: deploying the mod directory for the harnesses that load mods.

### 793d10a Mod deploy set

Requirement: 793d10a The Harness Deploy Sync MUST, for each detected harness that has a mods surface, plan one copy of every file under `agent-instructions/mods/` of the source into the harness's engram-owned root under `mods/`, keeping paths relative to that directory, and MUST make `<mods surface>/<top-level mod directory>` a link into the root (or, in manifest mode, a copy recorded in the manifest). Claude Code's mods surface is `~/.claude/skills`; Pi has none, so for Pi it MUST plan no mod file and MUST NOT create `~/.pi`. When the source has no `agent-instructions/mods/` it MUST plan nothing and not fail. It MUST NOT remove, replace or alter any file that the update deploys for skills, guidance or imports, and MUST report in the harness's section of the report each mod directory deployed, adopted or left alone.

Supports [793a4h1a Update installs the Integration for the harness it updates](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1d One update serves both harnesses differently](./793a_product-spec.md#793a4h1-uat-scenarios), [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios), [793a5c1b A Pi session on a machine whose Claude Code has the Integration](./793a_product-spec.md#793a5c1-uat-scenarios).

#### 793d10a1 Contract Scenarios

```gherkin
Scenario: 793d10a1a Claude Code gets the mod in its root and a link in its skills directory
  Given ~/.claude exists and the source holds agent-instructions/mods/engram-memory
  When the mod deploy is planned and applied for Claude Code
  Then ~/.claude/engram/mods/engram-memory holds every file of the source directory
  And ~/.claude/skills/engram-memory is a link into it

Scenario: 793d10a1b Pi gets nothing and ~/.pi is not created
  Given only ~/.claude exists
  When the mod deploy is planned
  Then no path under ~/.pi is planned or created

Scenario: 793d10a1c Both harnesses are present
  Given ~/.claude and ~/.pi both exist
  When the mod deploy is planned
  Then mod files are planned for Claude Code only

Scenario: 793d10a1d An existing install keeps what it had
  Given a Claude Code install with the skills, guidance and shim deployed
  When the mod deploy is applied
  Then every skill, guidance file and import is unchanged

Scenario: 793d10a1e The report names the mod
  Given a mod directory deployed
  When the report is written
  Then Claude Code's section names engram-memory as deployed

Scenario: 793d10a1f A source without mods plans nothing
  Given a source with no agent-instructions/mods
  When the mod deploy is planned
  Then nothing is planned and no error is returned

Scenario: 793d10a1g Manifest mode records the copy
  Given the surface runs in manifest mode
  When the mod deploy is applied
  Then the mod's files are real files under the surface and each written path is in the manifest
```

### 793d10b Sync rules apply to the mod

Requirement: 793d10b The Harness Deploy Sync MUST apply to the mod deploy the rules it applies to the skills deploy: a preview (`--dry-run`) MUST report the plan and write nothing; a second run over an unchanged source MUST change nothing and report the mod unchanged; a mod file that has left the source MUST be removed from the root and a link that has come to point at nothing MUST be removed; and nothing the sync cannot prove it owns MUST be deleted.

Supports [793a4h1c A preview installs none of the Integration](./793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1l Running update twice changes nothing](./793a_product-spec.md#793a4h1-uat-scenarios).

#### 793d10b1 Contract Scenarios

```gherkin
Scenario: 793d10b1a A preview writes no mod file
  Given a machine with no mod
  When the update runs with --dry-run
  Then the report lists the mod as it would be deployed and no file or link exists afterwards

Scenario: 793d10b1b A second run changes nothing
  Given the mod deployed by an earlier run
  When the update runs again over the same source
  Then no file or link changes and the report says the mod is unchanged

Scenario: 793d10b1c A removed mod file leaves the root
  Given a deployed mod file that the source no longer holds
  When the update runs
  Then the file is removed from the root

Scenario: 793d10b1d A user's own plugin directory is never deleted
  Given ~/.claude/skills holds a real directory my-own-plugin that is not in the deploy set
  When the update runs
  Then the directory is left untouched
```

## 793d11 Product-scenario mapping

Every scenario of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is listed once, with the technical requirements that realize it. The scenario text is not restated here (see [793b188 Technical spec structure: flows and components, with the scenario mapping as a table](./793b_decision-log.md#793b188-technical-spec-structure-flows-and-components-with-the-scenario-mapping-as-a-table)). A flow requirement ([793d2a Moment delivery](#793d2a-moment-delivery) to [793d2d Mod install](#793d2d-mod-install)) realizes the scenario by its routing, ordering, continuation and gating; a component requirement realizes it by the behavior it owns.

| Product scenario | Realized by |
| --- | --- |
| [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d5a Cues](#793d5a-cues), [793d6a Lookup and relevance](#793d6a-lookup-and-relevance), [793d6c Delivery text](#793d6c-delivery-text) |
| [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d6a Lookup and relevance](#793d6a-lookup-and-relevance), [793d6c Delivery text](#793d6c-delivery-text) |
| [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d5a Cues](#793d5a-cues), [793d6a Lookup and relevance](#793d6a-lookup-and-relevance) |
| [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d5a Cues](#793d5a-cues), [793d6a Lookup and relevance](#793d6a-lookup-and-relevance) |
| [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d5a Cues](#793d5a-cues), [793d6a Lookup and relevance](#793d6a-lookup-and-relevance) |
| [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d5a Cues](#793d5a-cues), [793d6a Lookup and relevance](#793d6a-lookup-and-relevance) |
| [793a2d1a Delivered content names its source](./793a_product-spec.md#793a2d1-uat-scenarios) | [793d6c Delivery text](#793d6c-delivery-text) |
| [793a2d1b A delivered chunk names its id](./793a_product-spec.md#793a2d1-uat-scenarios) | [793d6c Delivery text](#793d6c-delivery-text) |
| [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence](#793d6b-presence) |
| [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence](#793d6b-presence) |
| [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence](#793d6b-presence) |
| [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence](#793d6b-presence) |
| [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d3b State kept for the System](#793d3b-state-kept-for-the-system), [793d6b Presence](#793d6b-presence) |
| [793a2k1a The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6c Delivery text](#793d6c-delivery-text) |
| [793a2k1c An intentional recall by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) |
| [793a2k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d7c Reconcile and write](#793d7c-reconcile-and-write), [793d9a Settling offers](#793d9a-settling-offers) |
| [793a2k1e An intentional engram query by the agent is allowed](./793a_product-spec.md#793a2k1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) |
| [793a2l1a The index catches up after the turn](./793a_product-spec.md#793a2l1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3d Background queue](#793d3d-background-queue), [793d4a A job in rounds](#793d4a-a-job-in-rounds) |
| [793a2l1c Offers are resolved by the System](./793a_product-spec.md#793a2l1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d9a Settling offers](#793d9a-settling-offers), [793d9b Skill-runbook registration offers](#793d9b-skill-runbook-registration-offers) |
| [793a2l1d The agent is not asked to curate](./793a_product-spec.md#793a2l1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6c Delivery text](#793d6c-delivery-text), [793d9a Settling offers](#793d9a-settling-offers) |
| [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3c Judgement relay and deadlines](#793d3c-judgement-relay-and-deadlines), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d7a Extraction request](#793d7a-extraction-request), [793d7b Candidates and the reconcile request](#793d7b-candidates-and-the-reconcile-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Extraction request](#793d7a-extraction-request), [793d7b Candidates and the reconcile request](#793d7b-candidates-and-the-reconcile-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a3a1f The agent's own mistake overturned by evidence is captured](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7a Extraction request](#793d7a-extraction-request), [793d7b Candidates and the reconcile request](#793d7b-candidates-and-the-reconcile-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d7a Extraction request](#793d7a-extraction-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3d Background queue](#793d3d-background-queue), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a3a1i A captured note records how it was written](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d7c Reconcile and write](#793d7c-reconcile-and-write), [793d8a Rows to notes](#793d8a-rows-to-notes) |
| [793a3a1j A record that already exists is not written again](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d7b Candidates and the reconcile request](#793d7b-candidates-and-the-reconcile-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write), [793d8b No duplicate outcome](#793d8b-no-duplicate-outcome) |
| [793a3p1a A dispatch's outcome note is written from its report row](./793a_product-spec.md#793a3p1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d8a Rows to notes](#793d8a-rows-to-notes) |
| [793a3p1c A missing row writes nothing](./793a_product-spec.md#793a3p1-uat-scenarios) | [793d8a Rows to notes](#793d8a-rows-to-notes) |
| [793a3q1a A sound conclusion in the response is persisted](./793a_product-spec.md#793a3q1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d7b Candidates and the reconcile request](#793d7b-candidates-and-the-reconcile-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a3q1c A hunch is not persisted](./793a_product-spec.md#793a3q1-uat-scenarios) | [793d7b Candidates and the reconcile request](#793d7b-candidates-and-the-reconcile-request) |
| [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2c Automatic memory failure](#793d2c-automatic-memory-failure), [793d3e Containment](#793d3e-containment), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4b Failures](#793d4b-failures) |
| [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2c Automatic memory failure](#793d2c-automatic-memory-failure), [793d3e Containment](#793d3e-containment), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d4b Failures](#793d4b-failures) |
| [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2c Automatic memory failure](#793d2c-automatic-memory-failure), [793d3c Judgement relay and deadlines](#793d3c-judgement-relay-and-deadlines), [793d3e Containment](#793d3e-containment), [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a4b1d Slow capture does not stall the session](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3d Background queue](#793d3d-background-queue) |
| [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d2c Automatic memory failure](#793d2c-automatic-memory-failure), [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through) |
| [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6c Delivery text](#793d6c-delivery-text) |
| [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d2a Moment delivery](#793d2a-moment-delivery), [793d3a Events, channels and pass-through](#793d3a-events-channels-and-pass-through), [793d6c Delivery text](#793d6c-delivery-text) |
| [793a4c1e A secret in a correction is not persisted](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d7a Extraction request](#793d7a-extraction-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a4f1a No note records the System's own work](./793a_product-spec.md#793a4f1-uat-scenarios) | [793d3c Judgement relay and deadlines](#793d3c-judgement-relay-and-deadlines), [793d7a Extraction request](#793d7a-extraction-request), [793d7c Reconcile and write](#793d7c-reconcile-and-write) |
| [793a4f1b A session of its own receives no delivery](./793a_product-spec.md#793a4f1-uat-scenarios) | [793d3c Judgement relay and deadlines](#793d3c-judgement-relay-and-deadlines) |
| [793a4h1a Update installs the Integration for the harness it updates](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10a Mod deploy set](#793d10a-mod-deploy-set) |
| [793a4h1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10a Mod deploy set](#793d10a-mod-deploy-set) |
| [793a4h1c A preview installs none of the Integration](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10b Sync rules apply to the mod](#793d10b-sync-rules-apply-to-the-mod) |
| [793a4h1d One update serves both harnesses differently](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10a Mod deploy set](#793d10a-mod-deploy-set) |
| [793a4h1l Running update twice changes nothing](./793a_product-spec.md#793a4h1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10b Sync rules apply to the mod](#793d10b-sync-rules-apply-to-the-mod) |
| [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10a Mod deploy set](#793d10a-mod-deploy-set) |
| [793a5c1b A Pi session on a machine whose Claude Code has the Integration](./793a_product-spec.md#793a5c1-uat-scenarios) | [793d2d Mod install](#793d2d-mod-install), [793d10a Mod deploy set](#793d10a-mod-deploy-set) |
| [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d4a A job in rounds](#793d4a-a-job-in-rounds), [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1b Nothing is shown after a turn without events](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1c The summary is not in the agent's context](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1e The counts cover the events since the previous summary](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1g The summary follows the capture](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3d Background queue](#793d3d-background-queue), [793d3f Memory-activity line](#793d3f-memory-activity-line) |
| [793a5h1h An unreachable parent is not reported as a failure](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d2b Turn memory work](#793d2b-turn-memory-work), [793d3f Memory-activity line](#793d3f-memory-activity-line), [793d7d Parent-queued notice](#793d7d-parent-queued-notice) |

## 793d12 Recommended impact

**Repository.** New: `agent-instructions/mods/engram-memory/` (the mod, about one module of adapter code and its tests); `internal/auto/` (the runner and the five components); `internal/cli/auto.go` (the `engram auto run` command); a small Go package under `agent-instructions/` that embeds the skill files the judgement criteria come from. Changed: `internal/update` (the mod deploy set and the report row), `dev/targs.go` (`check-mod`), `cmd/engram/main.go` (no change expected: no new `cli.Primitives` field is needed, the job arrives on `Deps.Stdin`). Docs at integration: the new canonical files named in [793d2 Added: flows](#793d2-added-flows) to [793d10 Added: harness-deploy-sync (mod deploy)](#793d10-added-harness-deploy-sync-mod-deploy), `docs/specs/product/automatic-memory.md` from [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec), the glossary terms (mod, moment, delivery, job, round), and the directory table of `AGENTS.md`. The command list in the [Command Line Surface](../../specs/technical/command-line-surface.md) orientation gains `auto run`; its routing requirements apply to the new command unchanged and are reviewed at implementation.

**Dependencies and toolchain.** Runtime: Claude Code 2.1.287 or later for mods (the version of each event used is a check at implementation, [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods)); the `engram` program on the `PATH` of the Claude Code process, as the skills already assume. Development: the Claude Code CLI, for `claude plugin validate` and `claude plugin test`, behind a `check-mod` target that `check-full` and `check-for-fail` depend on ([793b206 The mod is validated and tested by claude plugin validate and claude plugin test in check-full](./793b_decision-log.md#793b206-the-mod-is-validated-and-tested-by-claude-plugin-validate-and-claude-plugin-test-in-check-full)); no JavaScript package manager, bundler or build step (the mod is one ES module Claude Code loads directly). No Go dependency is added.

**Tests.** The Go components are tested in the default suite over fakes of the query, the note writer, the amender, the transcript reader and the clock; one `integration`-tagged test starts the built `engram auto run` with a job on standard input against a temporary vault. No `network` test is needed, because Go never calls a model. The mod's tests run under `claude plugin test` with the world beneath the mod stubbed. Before the change is called complete, the installed binary and the deployed mod are run in a real Claude Code session, the unverified items of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods) are decided, and the observable outputs of the product scenarios (transcript, vault, status line) are inspected.

**Operations.** Each prompt waits for one query before the agent's first step, about 11 seconds on the author's vault (the measured cost of the shim's query today, moved from the agent to the System); each first edit, failure, dispatch and subagent start costs one more. Each top-level turn runs, in the background, two model requests for the capture (plus one per candidate), one `engram ingest --auto` and one scan for pending offers; the model is `haiku`, so the cost is bounded and measured by #850. The mod uses the session's own credentials for model requests, and a process the mod starts runs outside any sandbox with the user's permissions, as every mod does.

**Risks.** The mods runtime is early access; the unverified items of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods) are the risk, each with its fallback. A hook that blocks the shared hooks worker can unload every installed mod, so the mod holds no synchronous loop beyond building one JSON string. The System is silently inactive wherever mods do not load (row 18 of [793d1a What the design relies on in Claude Code mods](#793d1a-what-the-design-relies-on-in-claude-code-mods)), because a mod that does not load cannot warn.

### 793d12a Known limits and follow-ups

- Work still queued or running when the session ends is lost, so the lessons of a final turn can be missed if the user exits within seconds of it. Follow-up: a durable queue kept in `$.store` and resumed by the next session ([793b203 Background work is one in-memory queue, and a final turn can be lost on exit](./793b_decision-log.md#793b203-background-work-is-one-in-memory-queue-and-a-final-turn-can-be-lost-on-exit)).
- Moments inside a subagent's own loop are not detected; its dispatch and its stop are ([793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected)). Follow-up: presence from the subagent's transcript.
- The done moment is delivered at the prompt, before the declaration, and not at the declaration; a stop continuation is a MAY ([793b207 Late delivery at stop is a MAY and nothing else is late](./793b_decision-log.md#793b207-late-delivery-at-stop-is-a-may-and-nothing-else-is-late)).
- `deliveryMinScore`, `deliveryItemLimit`, `judgeModel` and the deadlines start at the values above; tuning them needs the measurement of #850. Redundant agent-side steps stay untouched until #851.
- The summary and warning are not shown where nothing draws (`claude -p`, the VS Code chat panel); delivery and capture still run there.
- Two Claude Code sessions capturing the same lesson at the same moment can each write it: the queue serializes captures within one session only.

## 793d13 Relative option impact

The options below are the holistic alternatives for each key choice, compared on the same complete scope: the 14 requirements of 793a, on Claude Code only, with intentional memory untouched. The agreed objectives are: **O1 Verifiable**, the product scenarios' assertions (transcripts, the vault, the screen) can be met and checked; **O2 Safe**, the option cannot break or stall the session or change what the agent or the user issued; **O3 Small**, the least repository-owned code and state; **O4 Grounded**, it relies on documented capabilities and the fewest unverified ones; **O5 Today's behavior**, it reuses the existing commands and defaults. Counts of units are not used, because the options differ in kind and not in count.

### 793d13a Mid-turn delivery

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. Context channels of the events: prompt context, tool-result context, subagent-start context | yes: separate blocks in the transcript, before the next response or tool call | yes: the prompt and the tool call are untouched | yes | yes (docs and declarations; the hook worker returns it) | Selected |
| B. Rewrite the prompt text, the Agent prompt or a tool result | partly: the delivery is not separate content | no: changes what the user or agent issued ([793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only)) | yes | yes | Rejected |
| C. `$.prompt.submit` from a job | partly: arrives as a new turn, after the agent's response | no: starts a turn nobody asked for | yes | yes | Rejected |
| D. Continue the turn at stop | partly: after the declaration | partly: an extra step the user sees | partly: needs a classifier | yes | A MAY ([793b207 Late delivery at stop is a MAY and nothing else is late](./793b_decision-log.md#793b207-late-delivery-at-stop-is-a-may-and-nothing-else-is-late)) |
| E. `$.session.send` to the agent | partly: queued, not guaranteed before the next step | yes | yes | partly | Rejected |

### 793d13b Moment detection

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. Mechanical cues in Go; the prompt lookup spans recall's four moments | yes for every scenario: a note about declaring work done reaches the agent at the prompt | yes: no model call on the path | yes | yes | Selected |
| B. A classifier call (`$.model.classify`) on each model step | yes, and finds semantic moments | partly: about a second per step, on the path | partly | partly | Rejected for now; the cues can grow into it |
| C. A periodic scan of the transcript with delivery at the next channel | partly: late and unpredictable | partly | no: a timer, a cursor and a second delivery path | partly | Rejected |
| D. The prompt cue alone | partly: no mid-turn moment | yes | yes | yes | Rejected: misses failures and dispatches |

### 793d13c Presence source

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. `$.session.messages()` plus a delivery ledger | yes: reflects compaction and a resume by construction | yes | yes | partly: whether a delivered block is a message is unverified (row 11) | Selected |
| B. Scan the transcript file | partly: the file lags the conversation, so a just-delivered item is missed | yes | partly | partly: a running subagent's path is undocumented | Rejected |
| C. The ledger alone | no: misses what the agent retrieved itself ([793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios)) | yes | yes | yes | Rejected |
| D. Buffer every `session.append` row per agent | yes, and covers subagents | partly: a hook on every row | no | partly | Follow-up, for moments inside subagents |

### 793d13d Judgement sessions

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. Relay `$.model.complete` (no session) | yes: any conversation, including a subagent's, judged from the text Go builds; the model is a constant | yes: no transcript, no hooks, no recursion, bounded by deadline and token cap | yes: one fixed relay loop | partly: result shape and a timeout option differ between docs and declarations (row 13) | Selected |
| B. `$.model.fork` | partly: the main conversation only; the session's model | yes | yes | partly: null on a cold snapshot | Rejected: cannot judge a subagent |
| C. `claude -p` started by the Go command | yes | partly: a real session with a transcript, the user's global instructions and the mod loaded again, so it needs an environment guard and `--safe-mode` to avoid delivering into itself | no: a second process model, its output parsing and its leftovers in the session list | yes | Rejected |

The Go command cannot call a model itself (the repository has no LLM client and no credentials), which also rules out judging inside `engram auto run`.

### 793d13e Subagent coverage

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. Record the spawn, deliver at the subagent's start, capture at its stop from its transcript | yes: both subagent scenarios | yes | yes | partly: matching a start to its spawn, and the transcript's completeness (rows 7 and 8) | Selected |
| B. Rewrite the `agent.spawn` prompt | partly | no: changes the Agent call's arguments ([793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios)) | yes | yes | Rejected |
| C. Rewrite the subagent's first row at `session.append` | yes | partly: holds the row while the lookup runs | partly | partly | Rejected |
| D. Also detect moments inside the subagent | yes | partly | no: needs the subagent's context | partly | Follow-up |

### 793d13f Late detection

A moment can be recognized only after the agent acted (a dispatch decided without the route skill, the declaration of done). Option A, selected: the dispatch cue delivers with the result of the call that revealed it, which the agent reads before its next response or tool call, and the done moment is covered early by the prompt's done-shaped phrase; nothing further is required. Option B: make the stop continuation required, with a classifier (rejected: [793d13a Mid-turn delivery](#793d13a-mid-turn-delivery) option D). Option C: start a turn with `$.prompt.submit` (rejected: [793d13a Mid-turn delivery](#793d13a-mid-turn-delivery) option C).

### 793d13g Summary and warning surface

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. `$.ui.status`, the mod's pinned line | yes: outside the agent's context; not a transcript row (row 15, to verify) | yes | yes | partly | Selected |
| B. `$.ui.log` | no: documented as a transcript row, so the warning would appear in the transcript ([793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios)) | yes | yes | yes | Rejected |
| C. `$.ui.toast` | partly: leaves the transcript untouched, but vanishes after seconds | yes | yes | yes | The fallback of row 15 |
| D. The line under the answer from `turn.complete` | partly: tied to the turn's rendering and to main turns | yes | yes | partly | Rejected |
| E. A pane or the band above the prompt | yes | yes | no: a drawing, its state and its keys | partly | Rejected |

### 793d13h Install path

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. A `skills-dir` plugin under `~/.claude/skills`, deployed by `engram update` | yes: active from the next session with no other step | yes: no edit of the user's settings | yes: the existing deploy sync | partly: hooks module from that origin and through a link (rows 3 and 4) | Selected |
| B. `CLAUDE_CODE_PLUGIN_DIRS` through `env` in `~/.claude/settings.json` | yes | partly: engram edits the user's settings file; managed `disableSideloadFlags` rejects it | partly: a settings merge | yes | Rejected |
| C. A local-directory marketplace with `extraKnownMarketplaces` and `enabledPlugins` | yes | partly: two edits of the user's settings | partly | yes | The fallback of row 2 |
| D. `claude --plugin-dir` by hand | no: the user has a step ([793a4h Installed by engram update and active in Claude Code](./793a_product-spec.md#793a4h-installed-by-engram-update-and-active-in-claude-code)) | yes | yes | yes | Rejected |

### 793d13i What runs in JavaScript and what in Go

| Option | O1 | O2 | O3 | O4 | Verdict |
| --- | --- | --- | --- | --- | --- |
| A. The mod adapts events, relays model requests, queues, shows the line; Go decides everything else | yes | yes: the JS holds no judgement or vault logic | yes | yes | Selected |
| B. Logic in the mod | yes | partly | no: duplicates vault logic outside the Go gates | yes | Rejected |
| C. A long-lived Go server the mod talks to | yes | partly: a lifecycle to start, restart and stop | no | partly | Rejected |

## 793d14 Alternative options

### 793d14a Rejected alternatives

Non-normative. Each is rejected for the reason given; the comparison tables of [793d13 Relative option impact](#793d13-relative-option-impact) give the objectives.

- **Settings hooks** (`UserPromptSubmit`, `PostToolUse`, `SubagentStart`, `Stop` in a settings file): rejected by the owner's premise ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go)); they cannot draw a line outside the transcript or relay a model request in process.
- **An MCP server or a skill that asks the agent to call it**: puts the work back on the agent's initiative, the failure [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement) measures.
- **A model call to judge relevance at each lookup**: one more round trip before every prompt for a judgement the engine's relevance floor already makes ([793b198 Relevance, item limit, chunk content and the wording of the instruction](./793b_decision-log.md#793b198-relevance-item-limit-chunk-content-and-the-wording-of-the-instruction)).
- **Go-owned state files for markers, ledgers or queues**: a new I/O capability and a state directory to specify; the window needs no marker and the ledger and queue live as long as the session ([793b203 Background work is one in-memory queue, and a final turn can be lost on exit](./793b_decision-log.md#793b203-background-work-is-one-in-memory-queue-and-a-final-turn-can-be-lost-on-exit)).
- **Reading the deployed skill files at run time for the judgement criteria**: depends on harness paths; the files embedded in the binary are the same text at the same version ([793b196 Judgement runs as relayed model completions, not sessions](./793b_decision-log.md#793b196-judgement-runs-as-relayed-model-completions-not-sessions)).
- **A durable background queue now**: see [793d12a Known limits and follow-ups](#793d12a-known-limits-and-follow-ups).
- **A mod for Pi**: out of scope; Pi is unchanged ([793a5c Other harnesses are unchanged](./793a_product-spec.md#793a5c-other-harnesses-are-unchanged)).
