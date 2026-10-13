# Parallel Memory Agent - Technical Spec

## 793d1 Orientation

The product scope is the intent of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec): the [System](./793a_product-spec.md#793a1-problem-statement) remembers for the [working agent](./793a_product-spec.md#793a1-problem-statement), beside the [intentional memory](./793a_product-spec.md#793a1-problem-statement) that stays as it is. This spec does not draft a design for it. It states the owner's two assumptions about the implementation as hypotheses, and gives the scenarios that confirm or disprove each ([793b240 The product spec is the owner's intent; the technical spec is his two hypotheses with the scenarios that test them](./793b_decision-log.md#793b240-the-product-spec-is-the-owners-intent-the-technical-spec-is-his-two-hypotheses-with-the-scenarios-that-test-them)). Every other implementation choice is an author choice, recorded in the decision log and listed in [793d19 Design as currently drafted](#793d19-design-as-currently-drafted); none is a requirement here, and they move to the implementation plan.

What exists, and the gap. Every [vault](./793a_product-spec.md#793a1-problem-statement) operation the [System](./793a_product-spec.md#793a1-problem-statement) needs exists: [Query Engine](../../specs/technical/query-engine.md), [Note Authoring](../../specs/technical/note-authoring.md), [Note Amendment](../../specs/technical/note-amendment.md), [Transcript Reader](../../specs/technical/transcript-reader.md), `engram ingest --auto` and [Harness Deploy Sync](../../specs/technical/harness-deploy-sync.md). Missing is whatever runs them without the [working agent](./793a_product-spec.md#793a1-problem-statement): observing the [moments](./793a_product-spec.md#793a1-problem-statement), adding text to the agent's [context](./793a_product-spec.md#793a1-problem-statement) without its own tool call, and judging what to deliver and what to save. The two hypotheses concern exactly that.

| Hypothesis | Statement | Scenarios |
| --- | --- | --- |
| [793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks](#793d17-h1-a-claude-code-mod-gives-a-smaller-implementation-surface-and-more-capability-than-settings-hooks) | A Claude Code [mod](#793d1b-terms) gives a smaller implementation [surface](#793d1b-terms) and more capability than [settings hooks](#793d1b-terms). | [793d17a1 Experiments](#793d17a1-experiments), [793d17b1 Experiments](#793d17b1-experiments), [793d17c1 Experiments](#793d17c1-experiments), [793d17d1 Experiments](#793d17d1-experiments), [793d17e1 Experiments](#793d17e1-experiments), [793d17f1 Experiments](#793d17f1-experiments) |
| [793d18 H2: the cheapest capable model is adequate for every System judgement, and is faster and cheaper](#793d18-h2-the-cheapest-capable-model-is-adequate-for-every-system-judgement-and-is-faster-and-cheaper) | The cheapest capable model is adequate for every [System](./793a_product-spec.md#793a1-problem-statement) [judgement](#793d1b-terms) and is faster and cheaper than the [session](../../GLOSSARY.md#session)'s model. | [793d18a1 Experiments](#793d18a1-experiments), [793d18b1 Experiments](#793d18b1-experiments), [793d18c1 Experiments](#793d18c1-experiments) |

How a hypothesis is settled. Every scenario below is a runnable experiment or a contract test, and its result is recorded in the decision log as the owner's evidence. A hypothesis is **confirmed** when every condition under "Confirmed if" holds, **disproved** when any condition under "Disproved if" holds (the owner then decides what to do next; no replacement is assumed here), and **inconclusive** when neither, because a result lay inside the [noise](#793d1b-terms) or a capability of the evidence ledger stayed "To verify". An inconclusive hypothesis is extended before anything is built on it.

### 793d1a What the hypotheses rely on in Claude Code mods

"Verified (docs)" is stated by the [Mods reference](https://code.claude.com/docs/en/plugins/mods/reference.md), [React to events](https://code.claude.com/docs/en/plugins/mods/events.md), [Use the mods API](https://code.claude.com/docs/en/plugins/mods/api.md) and [Test a mod](https://code.claude.com/docs/en/plugins/mods/test.md); "Verified (declarations)" by [claude-code.d.ts](https://github.com/anthropics/claude-code/blob/main/mods/types/claude-code.d.ts) (written by Claude Code 2.1.277); "Verified (tested)" was observed on Claude Code 2.1.296; "To verify" could not be confirmed, and the scenario in the last column settles it. The [mods](#793d1b-terms) runtime is early access.

| # | Capability relied on | Status | Settled by |
| --- | --- | --- | --- |
| 1 | A plugin whose `hooks/hooks.json` lists `modules` loads a module exporting `register(on, options)`; handlers are `($, e, next)`; Claude Code 2.1.287 or later | Verified (docs; tested, SP2 and every acceptance run: the debug log says `hooks module <name> loaded ... events: <the handlers>`) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 2 | A folder under `~/.claude/skills` holding `.claude-plugin/plugin.json` loads as a plugin in every [session](../../GLOSSARY.md#session), with no flag, install step or settings edit; its hooks module loads, also through a symlink into the engram root | Plugin load: Verified (docs); hooks module and symlink: Verified (tested). Also tested (SP2): the SDK `plugins` option loads it; a refused module is found in the debug log, not on `stderr`. | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 3 | `prompt.submit` fires for every origin (composer, task notification, scheduled trigger, peer [session](../../GLOSSARY.md#session)) and `next({ ...e, context })` adds text the model reads after the prompt as typed | Verified (tested, SP3 and SP8): `prompt.submit` fired for a typed prompt and for `/recall ...` (origin `sdk` under the driver) and the context reached the model; it also fired for a task notification, a scheduled trigger and a peer message, with their own origins | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 4 | `additionalContext` from `classic.PostToolUse`, `classic.PostToolUseFailure` and `classic.SubagentStart` reaches the model; [context](./793a_product-spec.md#793a1-problem-statement) added by a `tool.call` hook appears in the [transcript](../../GLOSSARY.md#transcript) as an `attachment` row; whether `prompt.submit` context is stored as a transcript row | Verified (tested, SP4): context is a separate `attachment` row, `attachment.type` `hook_additional_context`, in the transcript that received it; `prompt.submit` and `classic.PostToolUse` context in the main transcript, `classic.SubagentStart` context only in the subagent's | [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments) |
| 5 | `agent.spawn` carries the Agent call's `prompt`, `description` and `fork`; `classic.SubagentStop` carries `agent_transcript_path` and `last_assistant_message` | Verified (tested, SP5): the fields are as declared; the subagent transcript file is not complete at the stop (its final assistant row is written after), so `last_assistant_message` is used and the file polled | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 6 | `classic.SessionStart` carries the main agent's `transcript_path` | Verified (tested, SP6): `classic.SessionStart` `transcript_path` is the session's transcript file | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 7 | `session.compact` fires for the main agent and, with `agentId`, for each [subagent](../../GLOSSARY.md#subagent) or fork compaction; `turn.complete` also fires for an interrupted turn; module variables survive `/clear`, `/resume` and `/branch` | Verified (docs, declarations); tested (SP7: a `/compact` sent in the stream fired one `session.compact`; SP13: the module keeps running after `turn.complete`). Module variables across `/clear` and `/branch` stay docs-only until [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 8 | `$.session.messages()` returns the newest 4096 messages the next model request carries, main loop only; after a compaction or a resume, and whether a [context](./793a_product-spec.md#793a1-problem-statement) block added through row 3 or 4 appears in it | Shape and limit: Verified (declarations). Tested (SP7): it holds typed texts and replies, not a delivered block; a `/compact` in the stream does not shorten it; a resume restores it. Presence is read from the ledger and the transcript file instead | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments) |
| 10 | Time inside a [mods](#793d1b-terms) API call (`$.process.run`, `$.model.complete`) does not count against a hook's 10 s budget; `.catch` handlers have 1 s | Verified (docs, declarations; tested, SP12: a 25 s `$.process.run` in a `prompt.submit` hook was not cut off) | [793d17c1b Work longer than the hook budget is awaited without the handler being cut off](#793d17c1-experiments) |
| 11 | `$.model.complete({ model, prompt, system, maxTokens, timeoutMs })` returns `{ isAnswered, text, reason, usage }`; `model` is an alias (`haiku`, `sonnet`, `opus`) or a full id, billed to the [session](../../GLOSSARY.md#session)'s credentials. That the `haiku` alias resolves on this machine's subscription login | Verified (docs, declarations; tested, SP1: `haiku` answered on this login; the reply's `usage` holds token counts and does not name the model, which the debug log does: `$.model.complete (<mod>): <model id> answered in <n>ms`; the PoC's calls named `claude-haiku-5-5`) | [793d17c1a A model call is relayed on a named tier](#793d17c1-experiments) |
| 12 | `$.ui.status(text)` pins one line under the prompt; it is not a [transcript](../../GLOSSARY.md#transcript) row; `claude -p` draws no status line | Verified (declarations). Tested (SP9): in a stream `$.ui.status` arrives as a `system` message of subtype `ui_status`, and is not a transcript row | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) |
| 13 | `turn.step` fires once per model request of the main agent (with `agentId` for a subagent's), its result carries the step's `answer`, `toolUses`, `stopReason` and `usage`, a job started there with `$.clock.after(0, ...)` can be awaited from the next `classic.PostToolUse` and its `additionalContext` reaches the model before the next assistant row; a wait there is cut at about 10 s | Verified (tested, SP16, [793b320](./793b_decision-log.md#793b320-the-in-turn-recall-cue-is-built-as-793b238-decided)) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 14 | The environment of a settings hook command lacks the session's credential, so a hook script that calls a model needs one of its own; a nested `claude -p` with `--setting-sources ""`, `--tools ""`, a short `--system-prompt`, `--disable-slash-commands`, `--strict-mcp-config` and `--no-session-persistence` answers on `haiku` (its result names `claude-haiku-5-5`) with 438 tokens of usage, against 23 for the mod's `$.model.complete`, and loads none of the hooks | Verified (tested, 793e8i, [793b335 The settings-hook column is the recorder's counterpart plus one script, installed by a settings merge](./793b_decision-log.md#793b335-the-settings-hook-column-is-the-recorders-counterpart-plus-one-script-installed-by-a-settings-merge)): the hook's environment held the probe's variables and not `CLAUDE_CODE_OAUTH_TOKEN`; the call needed the token under another name. The default flags were measured once in a development run, about 17,000 tokens and the hooks loaded again in the nested session; that run is not recorded | [793d17c1a A model call is relayed on a named tier](#793d17c1-experiments) |
| 15 | A command hook's `timeout` defaults to 30 s on `UserPromptSubmit`; a longer job is cut off and the prompt goes on without its context; the `timeout` field of the entry avoids it | Verified (docs, [Hooks reference](https://code.claude.com/docs/en/hooks.md), Timeouts; tested, 793e8i): 25 s delivered at the default; cut off with `timeout` 10 (a `hook_cancelled` row) and at 35 s with the default; 35 s delivered with `timeout` 60 | [793d17c1b Work longer than the hook budget is awaited without the handler being cut off](#793d17c1-experiments) |
| 16 | A hook's `systemMessage` is shown to the user in the conversation and is stored in the transcript file as an attachment row of type `hook_system_message`; the mod's `$.ui.status` line is pinned under the prompt and is in no transcript row | Verified (tested, 793e8i, tmux): both on screen; the hook's text in the transcript file, the mod's not; the model quoted neither when asked | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) |
| 17 | A settings hook that exits 1 leaves an attachment row of type `hook_non_blocking_error` in the transcript, with its stderr; a hook that succeeds leaves `hook_success` attachment rows and each `Stop` a `stop_hook_summary` system row; the mod adds none of these | Verified (tested, 793e8i): 4 of 4 failures left the row; the agent, asked for any hook error in its context, quoted none (0 of 4 uncontained runs); the main tour's hook set added 4 `stop_hook_summary` and 1 `hook_success` row | [793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments), [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments) |
| 18 | Settings hooks placed in the `hooks.json` of a plugin linked into the skills directory fire with no settings edit | Verified (docs, [Mods reference](https://code.claude.com/docs/en/plugins/mods/reference.md), Files; tested, 793e8i): `SessionStart`, `UserPromptSubmit` and `Stop` fired, and `settings.json` held no hooks | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 19 | `UserPromptSubmit` fires for a task notification, a scheduled trigger and a peer message, and its input has no field that names the origin; the mod's `prompt.submit` carries `origin.kind` (`sdk`, `task-notification`, `scheduled-trigger`, `peer`) | Verified (tested, 793e8i): both surfaces saw all four prompts in the recorded run (and the hook column in three development reruns); the hook input holds `session_id`, `transcript_path`, `cwd`, `prompt_id`, `permission_mode`, `hook_event_name` and `prompt` | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 20 | A mod timer callback that runs while the handler that set it is still open puts its calls in that handler's scope, and the next event waits for them; set as the last action of the handler, the callback does not delay it | Verified (tested, 793e8i, [793b337 Defects of the probes found by running them, and the platform behaviours they exposed](./793b_decision-log.md#793b337-defects-of-the-probes-found-by-running-them-and-the-platform-behaviours-they-exposed)): a 20 s job held the next prompt 22.8 s in the first form and 2.7 s in the second | [793d17c1c Background work does not delay the next event](#793d17c1-experiments) |
| 21 | A settings hook registered with `async: true` does not delay the next prompt, and its `additionalContext` reaches the agent with the next conversation turn, not at a tool result | Verified (docs; tested, 793e8i): next prompt 3.2 s while a 20 s hook ran; the result arrived as an `async_hook_response` row before the next turn's first assistant row | [793d17c1c Background work does not delay the next event](#793d17c1-experiments) |

### 793d1b Terms

The terms **moment**, **delivery**, **lesson**, **working agent**, **context**, **vault** and **System** are as in [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement). This spec adds these:

- **Mod:** a Claude Code plugin whose `hooks/hooks.json` lists a module that Claude Code loads in process and calls with `register(on, options)`.
- **Settings hook:** a command that Claude Code runs on an event, as configured in a settings file, exchanging JSON through its standard input and output.
- **Surface:** the mod or the settings hooks, as a way to build the System's Claude Code side.
- **Hook budget:** the 10 seconds a hook handler's own time may take before Claude Code cuts it off.
- **Stand-in command:** a program that takes the place of the engram program in an experiment: it returns a fixed delivery after a set delay, so that a surface is measured apart from the real memory logic.
- **Probe:** a minimal mod module or settings hook that exercises one capability and writes what it observed to a log outside the transcript.
- **Judgement:** a model decision the System makes about memory. Its **kind** is one of the four the proof of concept has: `classify` (from the recent texts, which of the four decision moments of recall the agent is about to reach, or none), `relevance` (from a moment and candidate notes, which bear on the moment), `capture` (from a window of a session and the nearest notes, the lessons it contains, each marked covered by a nearest note or absent: extraction and reconciliation in one call) and `curation` (whether a pending offer is covered, near, absent or rejected). Extraction and reconcile are the two things `capture` is scored on, not kinds of their own.
- **Session model:** the model the working session is configured with; for H2 on this machine, the sonnet arm, because the acceptance sessions are pinned to it.
- **Cheapest tier:** the least capable and cheapest model tier available to the session, the `haiku` family, named by an alias and not by a dated id.
- **`judgeTiers`:** the table from a judgement kind to the model its requests name; every kind defaults to the cheapest tier.
- **Fixture set:** for one kind, the inputs drawn from real sessions and the vault, each with a gold label.
- **Gold label:** the correct answer for a fixture, fixed before either tier runs and not taken from the output of either tier.
- **Margin:** the largest shortfall of the cheapest tier against the session model that still counts as parity. A sample supports a margin when the one-sided 95% bound on its mean difference is no larger than it.
- **Noise:** the larger of the spread of the session model's own result over repeated runs of the same fixtures and the half-width of the 95% paired-bootstrap interval of a difference between two arms.

## 793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks

Hypothesis: 793d17 The Claude Code side of the [System](./793a_product-spec.md#793a1-problem-statement) is better built as a [mod](#793d1b-terms) than as [settings hooks](#793d1b-terms), because the mod has a smaller implementation [surface](#793d1b-terms) and does more.

Owner's reason: the owner assumes that [mods](#793d1b-terms) offer a smaller [surface](#793d1b-terms) and more functionality, and builds the design on "the novel capabilities mods provide" ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go)); for example "with mods, the system can just print stuff to the user" ([793c40 Does the user still see please's audit list?](./793c_open-questions.md#793c40-does-the-user-still-see-pleases-audit-list)). It is an assumption to test, not a settled fact.

Scope: this machine only, a macOS terminal and Claude Code sessions driven through the Agent SDK (the runtime of `claude -p`), signed in with a subscription, on the installed Claude Code version ([793b268 The PoC is minimal and targeted: environments the owner does not use and everything no scenario needs are cut](./793b_decision-log.md#793b268-the-poc-is-minimal-and-targeted-environments-the-owner-does-not-use-and-everything-no-scenario-needs-are-cut)). The verdict says nothing about other environments.

Every experiment builds the same [probe](#793d1b-terms) twice, as a [mod](#793d1b-terms) and as a [settings hook](#793d1b-terms), on the verified Claude Code version, and writes the observations to a log outside the [transcript](../../GLOSSARY.md#transcript). The memory logic is a [stand-in command](#793d1b-terms), so the [surfaces](#793d1b-terms) are compared with nothing else varying. "Smaller" and "more" are defined by [793d17f1a The same behavior is counted on both surfaces](#793d17f1-experiments) and [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments).

Confirmed if all hold:

- Capability: the [mod](#793d1b-terms) achieves every capability of [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), and every capability a [settings hook](#793d1b-terms) achieves.
- Channels: the mod adds [context](./793a_product-spec.md#793a1-problem-statement) at a prompt of every origin, at a tool result and at a [subagent](../../GLOSSARY.md#subagent)'s start, each attributable in the [transcript](../../GLOSSARY.md#transcript) ([793d17a1b Added context is attributable in the transcript](#793d17a1-experiments)), and shows the user text that is not in the transcript ([793d17b1a A message reaches the user and not the agent](#793d17b1-experiments)).
- Heavy work: the mod relays a model call on a named tier, awaits work longer than the [hook budget](#793d1b-terms) and runs work in the background without delaying the next event ([793d17c1a A model call is relayed on a named tier](#793d17c1-experiments), [793d17c1b Work longer than the hook budget is awaited without the handler being cut off](#793d17c1-experiments), [793d17c1c Background work does not delay the next event](#793d17c1-experiments)).
- Load: the mod loads in every [session](../../GLOSSARY.md#session) from the install path with no flag, install step or settings edit, on the installed Claude Code version ([793d17d1a The mod loads in every session from the install path](#793d17d1-experiments)).
- Restraint: a failure inside the mod leaves the session unaffected, and no handler denies, drops or answers an event ([793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments), [793d17e1b The mod changes nothing it must not](#793d17e1-experiments)).
- Smaller: the mod's [surface](#793d1b-terms) total is lower than the settings hooks' and none of its four counts is higher ([793d17f1a The same behavior is counted on both surfaces](#793d17f1-experiments)).

Disproved if any holds:

- A capability the product needs is achieved by the [settings hooks](#793d1b-terms) and not by the [mod](#793d1b-terms), or by neither [surface](#793d1b-terms).
- A "To verify" row on which a capability depends fails and no other mod capability replaces it.
- The mod is inactive in a [session](../../GLOSSARY.md#session) of this machine where the settings hooks are active ([793d17d1a The mod loads in every session from the install path](#793d17d1-experiments)).
- The settings hooks reach the same capability set with a lower surface total, or with any count lower than the mod's.

Result on this machine: DISPROVED ([793d17g4 Verdict](#793d17g4-verdict), [793b338 H1 is disproved by its own rule on this machine](./793b_decision-log.md#793b338-h1-is-disproved-by-its-own-rule-on-this-machine)).

### 793d17a Capability coverage

Settles rows 3 to 8 and 13 of [793d1a What the hypotheses rely on in Claude Code mods](#793d1a-what-the-hypotheses-rely-on-in-claude-code-mods). Run: live. The non-composer prompt origins are produced as the spikes found they can be (SP8): a background Bash task's completion notice, a one-shot `CronCreate` that fires about 60 s later, and a message sent by another session's module to the session's own id (the peer socket is shared with the machine's real sessions, so the message is addressed to that id and to no other); the human origin is `sdk` under the driver.

#### 793d17a1 Experiments

```gherkin
Scenario Outline: 793d17a1a Each surface is probed for each capability the product needs
  Given a probe for <capability> built once as a mod module and once as a settings hook, both writing what they observed to a log outside the transcript
  And a scripted session that triggers <capability>
  When the session runs on the verified Claude Code version
  Then each probe's log records whether <capability> was achieved
  And the experiment table gets a mod column and a hook column for it

  Examples:
    | capability                                                                                                                              |
    | a prompt from the composer, from a task notification, from a scheduled trigger and from another session is observed before the agent responds |
    | text is added after the prompt as typed, and the model reads it                                                                          |
    | text is added next to a tool result                                                                                                      |
    | text is added at a subagent's start, for that subagent only                                                                              |
    | a subagent's dispatch is observed with its task prompt                                                                                   |
    | a subagent's stop is observed with its transcript path and its last message; the file read at the stop may lack that message and holds it once polled |
    | the main agent's transcript path is known at the session's start                                                                         |
    | a compaction of the main agent is observed                                                                                               |
    | a step of the main agent is observed with its text and tool calls, and a job started there delivers its text at the next tool result       |

Scenario: 793d17a1b Added context is attributable in the transcript
  Given a mod that adds one block through each channel: after a prompt, next to a tool result and at a subagent's start
  When the session runs and its transcript file is read
  Then each block appears as its own attachment row of type hook_additional_context, in the transcript of the agent that received it (a subagent's start block only in that subagent's own file), separate from the prompt as typed and from the tool result, and its channel is told by where it sits and by any field of the row that names the hook event
  And the prompt text and the tool result are exactly as they were without the mod

Scenario: 793d17a1c The mod can read what the agent's context holds, through a compaction and a resume
  Given a mod that delivered block X to the main agent, and a probe that lists the delivered blocks found by reading what the surface can read: for the mod its own ledger and the hook_additional_context rows of the transcript file after the last compaction (the delivered blocks are in no message that session.messages returns, SP7), for a settings hook the transcript file named in its input
  When the session is compacted so that X no longer reaches the model, X is delivered again, and the session is then ended and resumed
  Then the probe lists X after its delivery, does not list X after the compaction removed it, and lists X again after the resume restored it
```

### 793d17b Output outside the agent's context

Settles row 12. Run: live.

#### 793d17b1 Experiments

```gherkin
Scenario: 793d17b1a A message reaches the user and not the agent
  Given a mod that shows the user one warning line, and a settings hook that returns the equivalent user-facing message, in an interactive session under tmux (a stream has no screen: it carries the mod's line as a ui_status message, SP9)
  When the session runs a turn and then the next prompt
  Then the user sees each message on screen
  And neither message appears in the transcript file or in the messages the model's next request carries
```

### 793d17c Model calls and long work

Settles rows 10 and 11. Run: live, on this machine's subscription login.

#### 793d17c1 Experiments

```gherkin
Scenario: 793d17c1a A model call is relayed on a named tier
  Given a mod that relays one short judgement request naming the model "haiku", and a settings hook script that must make the same call
  When both run
  Then the mod's call returns an answered reply with its text and token usage, on the session's credentials, and the debug log of the session names a model of the haiku family as the one that answered
  And the report counts what the hook script needed for the same call (credentials, subprocess and parsing)

Scenario: 793d17c1b Work longer than the hook budget is awaited without the handler being cut off
  Given a stand-in command that returns a delivery after 25 seconds, and a handler that awaits it through the mods API
  When a prompt is submitted
  Then the prompt continues with the delivery after about 25 seconds and the handler is not cut off
  And the handler's own time stays under the hook budget of 10 seconds
  And the report records, for a settings hook running the same stand-in command, whether it was cut off and what setting avoided that

Scenario: 793d17c1c Background work does not delay the next event
  Given a handler that starts a 20 second job in the background and continues its event at once, and a settings hook that starts the same job detached
  When a turn ends and the next prompt is submitted immediately
  Then the next prompt is processed without waiting for the job, under both surfaces
  And the job's result appears after about 20 seconds and, in the mod, reaches the agent at the first tool result that follows it (a wait for it inside the hook of a tool result is cut at about 10 s, SP16, so a job longer than that is not awaited there)
  And the report records how a settings hook's detached result would reach the agent
```

### 793d17d Install and load

Settles rows 1 and 2. Run: live.

#### 793d17d1 Experiments

```gherkin
Scenario: 793d17d1a The mod loads in every session from the install path
  Given the mod directory in the engram-owned root, a link to it in the skills directory, and no flag, install step or settings edit
  When a new session starts, is cleared (a /clear sent in the stream), is resumed (the SDK's resume option) and is branched (the SDK's fork option)
  Then the mod's probe log shows its register ran and a handler fired in each, and the debug log of each session says that the module loaded and lists its events
  And a settings hook probe built to the same events fires in the same sessions
  And claude plugin validate accepts the directory
  And the directory in the skills directory is a link whose target is in the engram-owned root
```

### 793d17e Failure containment and restraint

Settles the restraint conditions. Run: contract tests (`claude plugin test`) for the [mod](#793d1b-terms), live for the [settings hook](#793d1b-terms).

#### 793d17e1 Experiments

```gherkin
Scenario Outline: 793d17e1a A failure inside the surface leaves the session unaffected
  Given <cause> while the mod's handler, and the settings hook script, handle a prompt
  When the handler finishes
  Then the prompt continues with its text and no added context
  And no error text and no warning reach the agent's context, and no user input is lost
  And the user sees a warning outside the transcript saying what was skipped and why (in a stream, a ui_status message)
  And the report counts the code each surface needed to contain this failure

  Examples:
    | cause                                                |
    | the engram command cannot start                      |
    | the engram command's output cannot be parsed         |
    | the handler or script throws                         |
    | the work passes its deadline                         |

Scenario: 793d17e1b The mod changes nothing it must not
  Given the loaded mod and an agent that runs engram query and the learn skill in a turn
  When the debug log lists the events its handlers are registered for (claude plugin validate does not list them) and Claude Code fires the tool events of those commands
  Then no handler denies, drops, or answers an event in place of continuing it, and the handler of turn.step passes the step's input, chunks and result on unchanged
  And each memory command's input, permission decision and result are unchanged
  And the only effect of the mod on the agent's context is the blocks it added
```

### 793d17f Surface size

Run: build and count.

#### 793d17f1 Experiments

```gherkin
Scenario: 793d17f1a The same behavior is counted on both surfaces
  Given the probe pairs of 793d17a1a, 793d17c1a to 793d17c1c and 793d17e1a, each built once as a mod and once as a settings hook, calling the same stand-in command
  When the artifacts each surface needs are counted over the pairs
  Then the report gives, for each surface, the files it deploys outside the Go program, its registered handlers or hook entries, its edits to files the user owns, and its glue lines
  And the report gives how many capabilities of 793d17a1a each surface achieved
  And the mod's surface is smaller when its total of the four counts is lower and none of the four is higher than the settings hooks'
```

### 793d17g Results

Run on 2026-10-13 on this machine: Claude Code 2.1.296, a macOS terminal, Agent SDK sessions on the sonnet model signed in with the subscription token, `targ acceptance 793d17` and then `targ h1-report`. The final run was 22 scenario runs reading 55 sessions in 6 min 22 s, 1.99 USD by the SDK's client-side estimate, and about 6 USD more went to the development runs before it (a first full run, reruns and labs, by the same estimate). The observations are `dev/acceptance/h1/results/*.json`, the report `results/report.md`; how the run was built is [793b334 H1 is run on shared tours, and the Thens assert the mod and record the settings hook](./793b_decision-log.md#793b334-h1-is-run-on-shared-tours-and-the-thens-assert-the-mod-and-record-the-settings-hook), [793b335 The settings-hook column is the recorder's counterpart plus one script, installed by a settings merge](./793b_decision-log.md#793b335-the-settings-hook-column-is-the-recorders-counterpart-plus-one-script-installed-by-a-settings-merge) and [793b336 The surfaces are counted by rule, over the probe files as built](./793b_decision-log.md#793b336-the-surfaces-are-counted-by-rule-over-the-probe-files-as-built), and what the runs corrected in the probes is [793b337 Defects of the probes found by running them, and the platform behaviours they exposed](./793b_decision-log.md#793b337-defects-of-the-probes-found-by-running-them-and-the-platform-behaviours-they-exposed). Scope as [793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks](#793d17-h1-a-claude-code-mod-gives-a-smaller-implementation-surface-and-more-capability-than-settings-hooks): nothing is said of other environments.

#### 793d17g1 Scenario results

Before the steps were written the 11 scenarios (22 runs) had 110 undefined steps. After, 21 runs were Green and one Red (793d17e1b: the check counted the agent's own rows as rows the mod added, a fault of the check); with it corrected all 22 are Green. A Green run means the experiment ran and what the mod must do held; the hook column is in the observations.

| Scenario | Result | What the run recorded |
| --- | --- | --- |
| [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) (9 examples) | 9 Green | the mod achieved 9 of 9 capabilities, the settings hooks 9 of 9; the hook input names no origin; every origin arrived on the first try on both surfaces |
| [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments) | Green | each block is its own `hook_additional_context` row naming its hook event, on both surfaces; the typed prompts and the tool result equal the control run's; the hooks add 4 `stop_hook_summary` rows and 1 `hook_success` row, the mod none |
| [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments) | Green | blocks listed at the five prompts: [0, 1, 0, 1, 1] on both surfaces, read from the transcript file; the mod's own ledger alone: [no, yes, no, yes, no] (it is lost on the resume) |
| [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) | Green | both lines on screen under tmux; the mod's is in no transcript row, the hook's is in the transcript file as a `hook_system_message` row; the model quoted neither |
| [793d17c1a A model call is relayed on a named tier](#793d17c1-experiments) | Green | mod: answered, 23 tokens, `claude-haiku-5-5` named in the debug log, 1.3 s (0.9 s and 2.5 s in two earlier full runs); hook: answered through `claude -p`, 438 tokens, the model named in the result, 2.0 s (1.8 s in both earlier runs); the latencies were taken with up to six sessions at once and are indicative, the token counts are exact |
| [793d17c1b Work longer than the hook budget is awaited without the handler being cut off](#793d17c1-experiments) | Green | mod: 25 s awaited, own time 0 ms, turn 27.5 s; hook: delivered at the default 30 s, cut off with `timeout` 10, cut off at 35 s by default, delivered at 35 s with `timeout` 60 |
| [793d17c1c Background work does not delay the next event](#793d17c1-experiments) | Green | the next prompt took 2.7 s (mod), 3.4 s (detached hook process), 3.2 s (async hook) while the 20 s job ran; the mod's result, done at 20.1 s, reached the agent at the first tool result after it, the detached hook's at the next `PostToolUse`, the async hook's at the next turn; the mod with the timer set before an API call held the next prompt 22.8 s |
| [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) | Green | both surfaces saw startup, clear, resume and fork and handled a prompt in each; the debug log lists both modules and their events; `claude plugin validate` accepts both directories; hooks in a plugin's `hooks.json` fired with no settings edit |
| [793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments) (4 examples) | 4 Green | contained or not, on both surfaces: the prompt continued with its text and no block (8 of 8 per surface), the agent quoted no error (0 of 16); a contained failure warned the user 4 of 4 (mod `ui_status`, hook `systemMessage`); an uncontained hook left a `hook_non_blocking_error` row with its stderr 4 of 4, an uncontained mod nothing; containment code 3 lines on each surface |
| [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) | Green | the System's mod lists 9 events; its source holds no refusal object and its `turn.step` handler returns the step it was given; 138 contract tests pass; `claude plugin validate` passes with 5 gating-handler warnings (handlers that could deny and have no `.catch`; none denies); the same two commands in a live turn gave the same input, outcome and result with and without the mod, and the mod added no row |
| [793d17f1a The same behavior is counted on both surfaces](#793d17f1-experiments) | Green | the counts of 793d17g3 |

#### 793d17g2 Evidence table

Handler lines are the lines of the handler a capability uses in the probe (a handler shared by two capabilities carries the lines of both).

| Capability | Mod: possible | Hook: possible | Mod: handler lines (lines) | Hook: handler lines (lines) | Limit observed |
| --- | --- | --- | --- | --- | --- |
| prompts from every origin observed | yes | yes | 22 | 10 | the hook input names no origin; the notification and the peer message are told by their text |
| text after the prompt, read by the model | yes | yes | 22 | 10 | none |
| text next to a tool result | yes | yes | 9 | 4 | none |
| text at a subagent's start, for it only | yes | yes | 9 | 4 | none |
| a subagent's dispatch with its task prompt | yes | yes | 4 | 3 | the hook reads it from the `Agent` tool call (`PreToolUse`) |
| a subagent's stop with path and last message | yes | yes | 22 | 3 | the file at the stop lacked the message in 2 of 2 runs of the mod and in 1 of 2 of the hook: poll the file |
| the main transcript path at the start | yes | yes | 4 | 3 | none |
| a compaction observed | yes | yes | 4 | 3 | the hook event is `PostCompact` and carries the summary |
| a step with its text and tool calls, and a job's text at the next tool result | yes | yes | 21 | 17 | no step event for a hook: the text is read from the transcript file at the first `PostToolUse` after the step; the job starts one tool result later |
| context attributable in the transcript | yes | yes | (the handlers above) | (the handlers above) | the hooks add 4 `stop_hook_summary` and 1 `hook_success` row |
| what the context holds, through a compaction and a resume | yes | yes | 46 (4 + 4 + 38) | 24 | the mod's ledger alone is lost on a resume; both read the transcript file |
| a line for the user outside the agent's context | yes | partly | (in the prompt handler) | (in the prompt handler) | the hook's line is a transcript row |
| a model call on the cheapest tier | yes | yes | 6 | 6 | the hook has no credential of the session; 438 tokens against 23 and a subprocess |
| work longer than 10 s | yes | yes | (in the prompt handler) | (in the prompt handler) | the hook is cut off at its `timeout`: 30 s by default on `UserPromptSubmit` |
| background work that does not delay the next event | yes | yes | 17 (5 + 12) | 25 (8 + 17) | a mod timer must be set as the last action of its handler; a hook's job outlives it only if detached |
| loaded in every kind of session from the install path | yes | yes | (the install) | (the install) | the hooks need a settings edit, or the plugin form |
| a failure contained and shown | yes | yes | 3 | 3 | an uncontained hook leaves a `hook_non_blocking_error` row |
| restraint (no deny, drop or answer) | yes | yes | (the System's mod) | not run | not run for the hooks: no System hook set exists |

#### 793d17g3 Surface counts

| Surface | Files deployed (files + links) | Handlers or hook entries (count) | Edits to user files (files) | Glue lines (lines) | Total of the four (count) | Handler lines (lines) | Processes started by the surface in the main tour (count) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| mod, as built | 8 (6 + 2) | 13 | 0 | 86 | 107 | 146 | 0 |
| settings hooks, as built | 2 (2 + 0) | 10 | 1 | 81 | 94 | 79 | 27 |
| settings hooks packaged as a plugin | 5 (4 + 1) | 10 | 0 | 90 | 105 | 79 | 27 |

The mod has more files and links, more handlers and more glue than the hooks; the hooks have one more edit. A plugin's manifest, hooks file and link are 3 files wherever the mod is merged, so the file count of the mod does not fall to the hooks' under any packaging. The 27 processes are `node` commands (two scripts per event in this probe); the mod's handlers run in Claude Code's own worker.

#### 793d17g4 Verdict

| Condition of 793d17 | Holds | Evidence |
| --- | --- | --- |
| Confirmed if: capability | yes | 9 of 9 by the mod; 9 of 9 by the hooks, all of them also by the mod |
| Confirmed if: channels | yes | context at the prompt of all four origins, at a tool result and at a subagent's start, each its own attributable row; the line on screen and not in the transcript |
| Confirmed if: heavy work | yes | a haiku call answered, 25 s awaited with 0 ms of the handler's own time, a 20 s job without delaying the next prompt |
| Confirmed if: load | yes | startup, clear, resume and fork, from the link in the skills directory |
| Confirmed if: restraint | yes | four causes leave the session unaffected; no refusal object, `turn.step` passes through, 138 contract tests |
| Confirmed if: smaller | no | total 107 against 94; higher for the mod: files, handlers, glue lines |
| Disproved if: a capability the hooks achieve and the mod does not, or neither | no | none |
| Disproved if: a "To verify" row fails | no | no such row remains |
| Disproved if: the mod is inactive where the hooks are active | no | active in all four session kinds |
| Disproved if: the hooks reach the same capability set with a lower total or any lower count | yes | same set (9 of 9 each); total 94 against 107; lower for files (2 against 8), entries (10 against 13) and glue lines (81 against 86) |

Verdict: DISPROVED on this machine. The mod and the settings hooks both achieved every capability the product needs, so the clause of the disproof that applies is the counts: the hooks need fewer files, fewer entries, less glue and a lower total, and the mod wins only the edit to a user file (none against one), which the plugin form of the hooks also avoids at a total of 105. The first half of the hypothesis, a smaller surface, is therefore not supported; the second half, more capability, is not refuted but does not show up in the nine capabilities: the mod tells a prompt's origin, keeps its user line out of the transcript, makes a model call in 23 tokens without provisioning a credential or starting a process, and sees a step before its tool runs, while a hook set is smaller, runs in its own process (a crash is a notice row, a slow job cannot hold Claude Code's worker), works without the early-access mods runtime and, in its plugin form, installs the same way. What the hook surface can do that the mod cannot, as far as this run and the Hooks reference show: handler types that call a model or an HTTP endpoint without code (`prompt`, `agent`, `http`, `mcp_tool`), `async` with `asyncRewake`, `onFailure: "block"`, and hooks in managed settings that keep running when mods are restricted (documented, not run); none was needed for the nine. Whether the System's Claude Code side stays a mod is the owner's ([793c73](./793c_open-questions.md#793c73-does-the-systems-claude-code-side-stay-a-mod-now-that-h1-is-disproved-by-its-own-rule)).

## 793d18 H2: the cheapest capable model is adequate for every System judgement, and is faster and cheaper

Hypothesis: 793d18 Every [judgement](#793d1b-terms) the [System](./793a_product-spec.md#793a1-problem-statement) makes (`classify`, `relevance`, `capture` and `curation`: the kinds of the proof of concept, in which `capture` judges extraction and reconciliation in one call, [793b279 Judgement is one richer prompt per kind, and capture judges extraction and reconciliation in one call](./793b_decision-log.md#793b279-judgement-is-one-richer-prompt-per-kind-and-capture-judges-extraction-and-reconciliation-in-one-call)) can be made by the [cheapest tier](#793d1b-terms) as well as by the [session model](#793d1b-terms), within a [margin](#793d1b-terms), and the cheapest tier is faster and cheaper per call.

Scope: this machine only, Claude Code sessions (the Agent SDK) on a subscription login, priced at the published price of each model ([793b268 The PoC is minimal and targeted: environments the owner does not use and everything no scenario needs are cut](./793b_decision-log.md#793b268-the-poc-is-minimal-and-targeted-environments-the-owner-does-not-use-and-everything-no-scenario-needs-are-cut)).

Owner's reason: every [System](./793a_product-spec.md#793a1-problem-statement) task that needs model [judgement](#793d1b-terms) should go to the least capable tier that can handle that kind of judgement, not the [session](../../GLOSSARY.md#session)'s model, because the automatic System must be as fast and as cheap as possible ([793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier), which records the same for the [delivery](./793a_product-spec.md#793a1-problem-statement) side in [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)). Prior evidence, not a result for this System: the engram evaluation found a haiku-family tier at parity with a top tier on subtle lesson-sifting, with the framing of the request mattering more than the tier; a kind is raised only on measured evidence.

The comparison ([793b271 793c72 is answered: acceptance sessions on sonnet, and H2 compares three cost-scaled arms](./793b_decision-log.md#793b271-793c72-is-answered-acceptance-sessions-on-sonnet-and-h2-compares-three-cost-scaled-arms), [793b272 H2 arm sample sizes are inversely proportional to per-token price, anchored at the full set for sonnet](./793b_decision-log.md#793b272-h2-arm-sample-sizes-are-inversely-proportional-to-per-token-price-anchored-at-the-full-set-for-sonnet), sized again in [793b329](./793b_decision-log.md#793b329-h2-is-sized-again-for-the-kinds-the-poc-has-the-spread-is-stated-and-the-noise-floor-is-a-paired-bootstrap)). The **session model** of this machine is the sonnet arm, because the acceptance sessions are pinned to sonnet; the verdict of a kind is the cheapest tier against the sonnet arm. The prices are the published first-party list prices of 2026-10-06 (checked against the `claude-api` skill's cached table on 2026-10-13): US dollars per million input and output tokens, haiku (Claude Haiku 5.5) 0.10 and 0.50, sonnet (Claude Sonnet 5.5) 2 and 10, opus (Claude Opus 5.5) 4 and 20, fable (Claude Fable 5.1) 10 and 50; the run records the model id each alias resolves to and re-checks the prices that day.

| Arm | Model | Fixtures per kind | Margin the sample supports (one-sided 95%, spread below): over all fixtures, over the fixtures with a gold item | Reading |
| --- | --- | --- | --- | --- |
| cheapest tier | haiku | the full set | 0.047 at 200, 0.050 at 173 | the tier under test |
| session model | sonnet | the full set | 0.047 at 200, 0.050 at 173 | the verdict arm |
| opus | opus | half of the full set, stratified | 0.066 at 100, 0.071 at 87 | indicative |
| fable | fable | a fifth of the full set, stratified | 0.104 at 40, 0.111 at 35 | indicative |

The spread assumption: the per-fixture difference between two arms has a standard deviation of at most 0.4 (the arms disagree on at most 16 percent of the fixtures), so the one-sided 95% bound on a mean difference over n fixtures is 1.645 times 0.4 over the square root of n. For a guard measure over a subset the spread assumed is 0.3 and n is the size of the subset. A primary measure that is defined only over the fixtures that have a gold item (recall of the gold-relevant notes, lesson recall) has n equal to the number of those fixtures, so the full set of `relevance` and `capture` is 280 fixtures, at least 173 with a gold item and at least 100 with none, and the full set of `classify` and `curation` is 200 ([793b329](./793b_decision-log.md#793b329-h2-is-sized-again-for-the-kinds-the-poc-has-the-spread-is-stated-and-the-noise-floor-is-a-paired-bootstrap)). The run measures the real spread and reports it beside the assumed one; a sample whose real spread is larger supports a wider margin and says so.

Each kind is measured on its own [fixture set](#793d1b-terms) against its [gold labels](#793d1b-terms), with the request text the proof of concept sends (the same builders, copied byte for byte into the replay mod, and the same `maxTokens` and effort for every arm):

| Kind | Fixture input | [Gold label](#793d1b-terms) | Source of the gold | Primary measure | Guard measure |
| --- | --- | --- | --- | --- | --- |
| `classify` | the recent texts of a step of the main agent that wrote text and called a tool, as the cue reads them | which of the four [moments](./793a_product-spec.md#793a1-problem-statement) the agent is at, or none | drafted by fable from the step and what the agent did next, confirmed by the owner for every fixture | agreement: the moments named equal the gold moments | rate of naming a [moment](./793a_product-spec.md#793a1-problem-statement) where the gold is none |
| `relevance` | a lookup's moment text and its candidate [notes](./793a_product-spec.md#793a1-problem-statement) (the real lookup of the vault of that day) | which candidates bear on it | drafted by fable, confirmed by the owner for every fixture | recall of the gold-relevant [notes](./793a_product-spec.md#793a1-problem-statement) | rate of delivering a [note](./793a_product-spec.md#793a1-problem-statement) the gold excludes |
| `capture` | a window of a turn of a real session and the nearest notes of the vault of that day | the [lessons](./793a_product-spec.md#793a1-problem-statement) it contains, each with the terms a note stating it must hold, its stratum (explicit or subtle) and the nearest note that already states it, if any | the lessons the session's own `engram learn` recorded, plus drafted by fable and confirmed by the owner for every fixture | lesson recall (the share of gold lessons a written lesson states, by the terms) | the share of windows with no gold lesson in which the reply writes a lesson; and the share of covered gold lessons the reply writes as absent (a duplicate) |
| `curation` | one pending offer and its nearest notes | covered, near, absent or rejected, and the note it is covered by or folded into | constructed from real notes (the vault records no curation outcome that can be read, [793b330](./793b_decision-log.md#793b330-the-gold-of-curation-cannot-be-read-from-the-vaults-history-the-offers-are-made-from-real-notes)), the label by construction and drafted by fable, confirmed by the owner for every fixture | agreement: the outcome, and the note for covered and near | rate of destructive errors: a covered, near or rejected reply where the gold is another outcome |

Measurement points ([793b265 H2 is replayed by a small mod over fixtures, and gold comes from history or the owner](./793b_decision-log.md#793b265-h2-is-replayed-by-a-small-mod-over-fixtures-and-gold-comes-from-history-or-the-owner)): the replay mod loops `$.model.complete` over the fixtures with the proof of concept's request for the kind and the arm's model in place of the kind's. The wall time of a call is read around the call, in the mod (`$.clock.now()`); the cost of a call is the `usage` of its reply times the published price of the arm's model; each reply is read by the proof of concept's own strict parsers and a reply that is not read is a miss of the kind. A fixture's score is the mean of its 3 runs. The deadline of each kind in the proof of concept (3 s for `relevance`, 4 s for `classify`) is not applied to the replay (every call has 120 s); the share of calls over it is reported for every arm.

The noise floor of a comparison is the larger of two things: the spread of the sonnet arm's own result over its 3 runs (largest minus smallest of the three whole-set means) and the half-width of the 95% interval of the difference, from a paired bootstrap over fixtures (10,000 resamples with a fixed seed, resampling the fixtures and, inside a fixture, its 3 runs, the same resample for both arms).

Constants, initial values the owner can change: [margin](#793d1b-terms) 0.05, 3 runs per fixture per arm, a full set of 200 fixtures for `classify` and `curation` and 280 for `relevance` and `capture`, of which at least 100 have the gold "none" (`classify`), no relevant candidate (`relevance`), no lesson (`capture`) or, for `curation`, an outcome of each of the four (at least 40 each), and at least 173 have a gold item (`relevance`, `capture`); at least 40 fixtures per kind.

Confirmed if all hold, for each kind:

- Parity: the one-sided 95% lower bound of (the cheapest tier's primary measure minus the sonnet arm's) is at least minus the [margin](#793d1b-terms), and the one-sided 95% upper bound of (its guard measure minus the sonnet arm's) is at most the margin, on the whole set and on the subtle stratum of `capture` ([793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793d18a1c Subtle cases are not hidden by easy ones](#793d18a1-experiments)).
- Speed and cost: the cheapest tier's median latency per call and its cost per fixture are lower than the sonnet arm's ([793d18b1a The cheapest tier is faster per call](#793d18b1-experiments), [793d18b1b The cheapest tier is cheaper per fixture](#793d18b1-experiments)).
- Table: the kind can be raised alone, on evidence, and its request always names its own model ([793d18c1a Every request names the model of its kind](#793d18c1-experiments), [793d18c1b Raising one kind changes only that kind](#793d18c1-experiments)).

Disproved for a kind if the one-sided 95% upper bound of the primary difference is below minus the [margin](#793d1b-terms), or the one-sided lower bound of the guard difference is above the margin; that kind is then raised in [`judgeTiers`](#793d1b-terms) and measured again, and H2 holds for the other kinds. Disproved as a whole if the cheapest tier is not faster or not cheaper than the sonnet arm on the fixtures. Inconclusive for a kind if neither holds, that is when an interval reaches across the margin, which is when the gap lies inside the [noise](#793d1b-terms) of the sample ([793d18a1b A gap inside the noise is inconclusive, not a tie](#793d18a1-experiments)). The opus and fable arms are read as indicative ([793b272 H2 arm sample sizes are inversely proportional to per-token price, anchored at the full set for sonnet](./793b_decision-log.md#793b272-h2-arm-sample-sizes-are-inversely-proportional-to-per-token-price-anchored-at-the-full-set-for-sonnet)): a haiku shortfall beyond the margin their sample supports is flagged to the owner and is not a disproof, and they cannot show parity at 0.05.

### 793d18a Agreement with the gold labels

Run: replay of the [fixture sets](#793d1b-terms) through the four arms.

#### 793d18a1 Experiments

```gherkin
Scenario Outline: 793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin
  Given the fixture set of <kind> and the same request text for every arm
  When each arm judges its fixtures 3 times
  Then the cheapest tier's mean <primary> is at least the session model's mean minus 0.05, by the lower bound of the paired bootstrap
  And the cheapest tier's mean <guard> is no worse than the session model's plus 0.05, by the upper bound of the paired bootstrap

  Examples:
    | kind       | primary                              | guard                                                                                              |
    | classify   | agreement with the gold moments      | rate of naming a moment where the gold is none                                                     |
    | relevance  | recall of the gold-relevant notes    | rate of delivering a note the gold excludes                                                        |
    | capture    | lesson recall                        | rate of lessons written in a window with none, and rate of covered lessons written as absent       |
    | curation   | agreement                            | rate of destructive errors                                                                         |

Scenario: 793d18a1b A gap inside the noise is inconclusive, not a tie
  Given a kind whose interval of the cheapest tier's difference from the session model reaches across the margin, or whose gap is smaller than the noise floor while its bound does not clear the margin
  When the result is recorded
  Then the kind is marked inconclusive and not confirmed
  And the full set is the cap, so the report gives the interval and the fixtures that would bring it inside the margin at the real spread

Scenario: 793d18a1c Subtle cases are not hidden by easy ones
  Given the capture fixtures split into a stratum with an explicit correction in the user's words and a subtle stratum where the lesson is implied by what the user did or is the agent's own correction of itself
  When the arms are scored
  Then each stratum is reported on its own
  And H2 is confirmed for capture only if the subtle stratum also meets the parity of 793d18a1a, and is inconclusive for capture if the subtle stratum's bound does not clear the margin
```

### 793d18b Speed and cost

Run: the same replay, timing and pricing each call.

#### 793d18b1 Experiments

```gherkin
Scenario Outline: 793d18b1a The cheapest tier is faster per call
  Given the fixture set of <kind> and the same request text for every arm
  When each arm judges its fixtures 3 times and each call's wall time is recorded around the call
  Then the cheapest tier's median latency is below the session model's
  And the report gives the share of calls of each arm over the deadline the proof of concept gives the kind

  Examples:
    | kind       |
    | classify   |
    | relevance  |
    | capture    |
    | curation   |

Scenario: 793d18b1b The cheapest tier is cheaper per fixture
  Given the usage reported for each call and the published price of each model
  When the cost per fixture is computed for each kind and each arm
  Then the cheapest tier's cost is lower for each kind
  And the report gives the ratio of the two costs for each kind
```

### 793d18c The tier table

Run: contract tests over the request builders.

#### 793d18c1 Experiments

```gherkin
Scenario: 793d18c1a Every request names the model of its kind
  Given a job of each kind and the default judgeTiers
  When the requests are built
  Then each request carries the model judgeTiers gives for its kind, and none carries the session model

Scenario: 793d18c1b Raising one kind changes only that kind
  Given a kind that failed the parity of 793d18a1a
  When judgeTiers gives that kind the next tier
  Then the requests of that kind name the new model, the requests of the other three kinds are unchanged, and the raised kind is measured again with 793d18a1a
```

## 793d19 Design as currently drafted

These are the author's choices so far. They are not requirements of this spec and are not claimed to be right; each is recorded, with its rationale and the alternatives, in the decision log, and they move to the implementation plan once the hypotheses are settled.

- **The split between the [mod](#793d1b-terms) and the engram commands**, with [judgements](#793d1b-terms) relayed through the mod (the mod orchestrates over the existing commands and no Go runner is built, per [793b256 The PoC orchestrates in JavaScript inside the mod over the existing engram commands](./793b_decision-log.md#793b256-the-poc-orchestrates-in-javascript-inside-the-mod-over-the-existing-engram-commands)): [793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go), [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter).
- **Where the mod's source lives, and how `engram update` installs it:** [793b190 The mod's source lives in agent-instructions/mods and is written in JavaScript](./793b_decision-log.md#793b190-the-mods-source-lives-in-agent-instructionsmods-and-is-written-in-javascript), [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin).
- **[Delivery](./793a_product-spec.md#793a1-problem-statement) channels** (prompt [context](./793a_product-spec.md#793a1-problem-statement), tool-result context, subagent-start context): [793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context).
- **[Moments](./793a_product-spec.md#793a1-problem-statement):** the cues and the classification: [793b193 Moments are mechanical cues; the prompt lookup spans recall's four moments](./793b_decision-log.md#793b193-moments-are-mechanical-cues-the-prompt-lookup-spans-recalls-four-moments), [793b239 Author choices in applying 793b238, shown to the owner](./793b_decision-log.md#793b239-author-choices-in-applying-793b238-shown-to-the-owner); [subagents](../../GLOSSARY.md#subagent): [793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected); tier evidence at routing: [793b218 793c64 is answered (a): tier evidence is delivered when the orchestrator routes through route](./793b_decision-log.md#793b218-793c64-is-answered-a-tier-evidence-is-delivered-when-the-orchestrator-routes-through-route).
- **Presence and the delivery ledger:** [793b195 Presence is read from the live context and a delivery ledger](./793b_decision-log.md#793b195-presence-is-read-from-the-live-context-and-a-delivery-ledger).
- **Judgement as relayed completions, and the tier table:** [793b196 Judgement runs as relayed model completions, not sessions](./793b_decision-log.md#793b196-judgement-runs-as-relayed-model-completions-not-sessions), [793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier).
- **The round protocol, with a Go side that keeps no state:** dropped for the PoC by [793b256 The PoC orchestrates in JavaScript inside the mod over the existing engram commands](./793b_decision-log.md#793b256-the-poc-orchestrates-in-javascript-inside-the-mod-over-the-existing-engram-commands); it was [793b197 The Go side is stateless and advances a job one round per call](./793b_decision-log.md#793b197-the-go-side-is-stateless-and-advances-a-job-one-round-per-call).
- **Relevance, the item limit and the wording of the instruction:** [793b198 Relevance, item limit, chunk content and the wording of the instruction](./793b_decision-log.md#793b198-relevance-item-limit-chunk-content-and-the-wording-of-the-instruction).
- **Capture:** the window and the redaction (the [System](./793a_product-spec.md#793a1-problem-statement)'s own blocks are attachment rows and never in `$.session.messages()`, so nothing is stripped, [793b319 The PoC drops the code and the references of what does not exist](./793b_decision-log.md#793b319-the-poc-drops-the-code-and-the-references-of-what-does-not-exist)) [793b199 The capture window, redaction and the System's own blocks](./793b_decision-log.md#793b199-the-capture-window-redaction-and-the-systems-own-blocks); reconciliation [793b200 Reconciliation is covered or absent; dispatch outcomes are deduplicated mechanically](./793b_decision-log.md#793b200-reconciliation-is-covered-or-absent-dispatch-outcomes-are-deduplicated-mechanically); curation [793b201 Curation follows curate's table, including the skill-runbook case](./793b_decision-log.md#793b201-curation-follows-curates-table-including-the-skill-runbook-case); the index [793b202 The chunk index is brought up to date with engram ingest --auto after the turn's other work](./793b_decision-log.md#793b202-the-chunk-index-is-brought-up-to-date-with-engram-ingest---auto-after-the-turns-other-work).
- **Background work, the activity line and failures:** [793b203 Background work is one in-memory queue, and a final turn can be lost on exit](./793b_decision-log.md#793b203-background-work-is-one-in-memory-queue-and-a-final-turn-can-be-lost-on-exit), [793b204 The memory-activity line is the mod's status line](./793b_decision-log.md#793b204-the-memory-activity-line-is-the-mods-status-line), [793b205 Failures are classified by the mod and the runner, with named deadlines](./793b_decision-log.md#793b205-failures-are-classified-by-the-mod-and-the-runner-with-named-deadlines).
- **Backlog capture, the watermark and coexistence with intentional ingestion:** [793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](./793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact), [793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped), [793b233 Choices made in applying 793b232, shown to the owner](./793b_decision-log.md#793b233-choices-made-in-applying-793b232-shown-to-the-owner), [793b234 Intentional ingestion in other harnesses continues; the System coexists with it through the write lock](./793b_decision-log.md#793b234-intentional-ingestion-in-other-harnesses-continues-the-system-coexists-with-it-through-the-write-lock).
- **Validating the mod through the existing integration gate:** [793b236 The mod is validated and tested through the existing integration gate, not a new check target](./793b_decision-log.md#793b236-the-mod-is-validated-and-tested-through-the-existing-integration-gate-not-a-new-check-target).
- **The initial values** of deadlines, windows and limits: [793b239 Author choices in applying 793b238, shown to the owner](./793b_decision-log.md#793b239-author-choices-in-applying-793b238-shown-to-the-owner), [793b233 Choices made in applying 793b232, shown to the owner](./793b_decision-log.md#793b233-choices-made-in-applying-793b232-shown-to-the-owner).

### 793d19a Security

Ratified by the owner (Joe, 2026-10-13, [793b324 Owner answer 3: the residual-risk statement 793d19a is ratified as written, with the near outcome kept](./793b_decision-log.md#793b324-owner-answer-3-the-residual-risk-statement-793d19a-is-ratified-as-written-with-the-near-outcome-kept)). This is the residual-risk statement of the opus implementation and security review of the PoC ([793b315 An opus implementation and security review of the PoC returned "targeted fixes first"](./793b_decision-log.md#793b315-an-opus-implementation-and-security-review-of-the-poc-returned-targeted-fixes-first)), recorded here as the reviewer wrote it; the fixes it asked for are [793b318 The mod's lookups, offers and redaction are hardened against what an offer, a prompt or a transcript can say](./793b_decision-log.md#793b318-the-mods-lookups-offers-and-redaction-are-hardened-against-what-an-offer-a-prompt-or-a-transcript-can-say).

Automatic memory reads the user's and the agent's own messages, from live turns, subagent transcripts and session files on this machine that ended after the install time minus 7 days ([793b323 Owner answer 2: the backlog cut-off is one fixed point, install time minus 7 days](./793b_decision-log.md#793b323-owner-answer-2-the-backlog-cut-off-is-one-fixed-point-install-time-minus-7-days)), and pending offers from linked vaults. It sends them, after pattern-based redaction, to a cheap model that may write or amend notes. Tool inputs and results never reach the model. Speakers are taken from row structure. Replies are parsed strictly, and model output can touch only a note named by number from the nearest set, through fixed engram commands. Remaining risks: (1) an injection in user or agent text, or a planted session file, can become a note that is delivered as memory to later sessions; (2) a crafted offer can be folded into a trusted note's fields; (3) secrets of shapes the patterns do not cover can reach the model provider and the vault; (4) a wrongly judged `covered` discards an offer, recoverable only from vault history. Mitigations: provenance lines on every automatic note, aliases on folded offers, data framing on delivered notes, the user-visible activity line, and owner review of the vault. Model refusal to follow data is not a guarantee.

## 793d20 Traceability to the product spec

Every scenario of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is listed with the hypothesis scenarios that test the assumption it rests on, or the design [note](./793a_product-spec.md#793a1-problem-statement) that will realise it. The scenario text is not restated here ([793b188 Technical spec structure: flows and components, with the scenario mapping as a table](./793b_decision-log.md#793b188-technical-spec-structure-flows-and-components-with-the-scenario-mapping-as-a-table)).

| Product scenario | Tested or realised by |
| --- | --- |
| [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments), [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b193 Moments are mechanical cues; the prompt lookup spans recall's four moments](./793b_decision-log.md#793b193-moments-are-mechanical-cues-the-prompt-lookup-spans-recalls-four-moments), [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter) |
| [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b198 Relevance, item limit, chunk content and the wording of the instruction](./793b_decision-log.md#793b198-relevance-item-limit-chunk-content-and-the-wording-of-the-instruction) |
| [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments), [793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected) |
| [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context) |
| [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793b193 Moments are mechanical cues; the prompt lookup spans recall's four moments](./793b_decision-log.md#793b193-moments-are-mechanical-cues-the-prompt-lookup-spans-recalls-four-moments) |
| [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793b218 793c64 is answered (a): tier evidence is delivered when the orchestrator routes through route](./793b_decision-log.md#793b218-793c64-is-answered-a-tier-evidence-is-delivered-when-the-orchestrator-routes-through-route) |
| [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments), [793b195 Presence is read from the live context and a delivery ledger](./793b_decision-log.md#793b195-presence-is-read-from-the-live-context-and-a-delivery-ledger) |
| [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments), [793b195 Presence is read from the live context and a delivery ledger](./793b_decision-log.md#793b195-presence-is-read-from-the-live-context-and-a-delivery-ledger) |
| [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments), [793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected) |
| [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments) |
| [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios) | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments) |
| [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b199 The capture window, redaction and the System's own blocks](./793b_decision-log.md#793b199-the-capture-window-redaction-and-the-systems-own-blocks) |
| [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected) |
| [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b200 Reconciliation is covered or absent; dispatch outcomes are deduplicated mechanically](./793b_decision-log.md#793b200-reconciliation-is-covered-or-absent-dispatch-outcomes-are-deduplicated-mechanically) |
| [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b200 Reconciliation is covered or absent; dispatch outcomes are deduplicated mechanically](./793b_decision-log.md#793b200-reconciliation-is-covered-or-absent-dispatch-outcomes-are-deduplicated-mechanically), [793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped) |
| [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments), [793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](./793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact), [793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped) |
| [793a4j1a The index catches up after the turn](./793a_product-spec.md#793a4j1-uat-scenarios) | [793b202 The chunk index is brought up to date with engram ingest --auto after the turn's other work](./793b_decision-log.md#793b202-the-chunk-index-is-brought-up-to-date-with-engram-ingest---auto-after-the-turns-other-work) |
| [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios) | [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793b201 Curation follows curate's table, including the skill-runbook case](./793b_decision-log.md#793b201-curation-follows-curates-table-including-the-skill-runbook-case) |
| [793a4j1d The agent is not asked to curate](./793a_product-spec.md#793a4j1-uat-scenarios) | [793d17e1b The mod changes nothing it must not](#793d17e1-experiments), [793b201 Curation follows curate's table, including the skill-runbook case](./793b_decision-log.md#793b201-curation-follows-curates-table-including-the-skill-runbook-case) |
| [793a4j1e Slow housekeeping does not stall the session](./793a_product-spec.md#793a4j1-uat-scenarios) | [793d17c1c Background work does not delay the next event](#793d17c1-experiments), [793b203 Background work is one in-memory queue, and a final turn can be lost on exit](./793b_decision-log.md#793b203-background-work-is-one-in-memory-queue-and-a-final-turn-can-be-lost-on-exit) |
| [793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) |
| [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments) |
| [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments), [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) |
| [793a4c1f The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a4c1-uat-scenarios) | [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments), [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) |
| [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments), [793b205 Failures are classified by the mod and the runner, with named deadlines](./793b_decision-log.md#793b205-failures-are-classified-by-the-mod-and-the-runner-with-named-deadlines) |
| [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments), [793b205 Failures are classified by the mod and the runner, with named deadlines](./793b_decision-log.md#793b205-failures-are-classified-by-the-mod-and-the-runner-with-named-deadlines) |
| [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments), [793b205 Failures are classified by the mod and the runner, with named deadlines](./793b_decision-log.md#793b205-failures-are-classified-by-the-mod-and-the-runner-with-named-deadlines) |
| [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios) | [793d17e1a A failure inside the surface leaves the session unaffected](#793d17e1-experiments), [793b205 Failures are classified by the mod and the runner, with named deadlines](./793b_decision-log.md#793b205-failures-are-classified-by-the-mod-and-the-runner-with-named-deadlines) |
| [793a5j1a Update installs the System for the harness it updates](./793a_product-spec.md#793a5j1-uat-scenarios) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments), [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| [793a5j1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a5j1-uat-scenarios) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments), [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| [793a5j1c A preview installs none of the System](./793a_product-spec.md#793a5j1-uat-scenarios) | [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| [793a5j1d One update serves both harnesses differently](./793a_product-spec.md#793a5j1-uat-scenarios) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments), [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| [793a5j1l Running update twice changes nothing](./793a_product-spec.md#793a5j1-uat-scenarios) | [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments), [793b204 The memory-activity line is the mod's status line](./793b_decision-log.md#793b204-the-memory-activity-line-is-the-mods-status-line) |
| [793a5h1b Nothing is shown after a turn without events](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments), [793b204 The memory-activity line is the mod's status line](./793b_decision-log.md#793b204-the-memory-activity-line-is-the-mods-status-line) |
| [793a5h1c The summary is not in the agent's context](./793a_product-spec.md#793a5h1-uat-scenarios) | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) |
| [793a5h1e The counts cover the events since the previous summary](./793a_product-spec.md#793a5h1-uat-scenarios) | [793b204 The memory-activity line is the mod's status line](./793b_decision-log.md#793b204-the-memory-activity-line-is-the-mods-status-line) |
| [793a5k1c An intentional recall by the agent is allowed](./793a_product-spec.md#793a5k1-uat-scenarios) | [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) |
| [793a5k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a5k1-uat-scenarios) | [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) |
| [793a5k1e An intentional engram query by the agent is allowed](./793a_product-spec.md#793a5k1-uat-scenarios) | [793d17e1b The mod changes nothing it must not](#793d17e1-experiments) |
| [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios) | [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin) |
| [793a5c1b A Pi session on a machine whose Claude Code has the System](./793a_product-spec.md#793a5c1-uat-scenarios) | [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin), [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios) | [793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](./793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact) |
