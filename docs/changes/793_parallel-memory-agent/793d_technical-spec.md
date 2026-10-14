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

Run on 2026-10-13 on this machine: Claude Code 2.1.296, a macOS terminal, Agent SDK sessions on the sonnet model signed in with the subscription token, `targ acceptance 793d17` (the steps loaded from `ENGRAM_EVAL_DIR`) and then `node h1/report.js --write` there. The final run was 22 scenario runs reading 55 sessions in 6 min 22 s, 1.99 USD by the SDK's client-side estimate, and about 6 USD more went to the development runs before it (a first full run, reruns and labs, by the same estimate). The observations were `dev/acceptance/h1/results/*.json` and the report `results/report.md`; they are not in the repository (ephemeral, under `ENGRAM_EVAL_DIR`: [793b343 The hypothesis-eval artifacts are ephemeral and kept out of the repository](./793b_decision-log.md#793b343-the-hypothesis-eval-artifacts-are-ephemeral-and-kept-out-of-the-repository)), and the tables below are the durable record; how the run was built is [793b334 H1 is run on shared tours, and the Thens assert the mod and record the settings hook](./793b_decision-log.md#793b334-h1-is-run-on-shared-tours-and-the-thens-assert-the-mod-and-record-the-settings-hook), [793b335 The settings-hook column is the recorder's counterpart plus one script, installed by a settings merge](./793b_decision-log.md#793b335-the-settings-hook-column-is-the-recorders-counterpart-plus-one-script-installed-by-a-settings-merge) and [793b336 The surfaces are counted by rule, over the probe files as built](./793b_decision-log.md#793b336-the-surfaces-are-counted-by-rule-over-the-probe-files-as-built), and what the runs corrected in the probes is [793b337 Defects of the probes found by running them, and the platform behaviours they exposed](./793b_decision-log.md#793b337-defects-of-the-probes-found-by-running-them-and-the-platform-behaviours-they-exposed). Scope as [793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks](#793d17-h1-a-claude-code-mod-gives-a-smaller-implementation-surface-and-more-capability-than-settings-hooks): nothing is said of other environments.

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

Verdict: DISPROVED on this machine. The mod and the settings hooks both achieved every capability the product needs, so the clause of the disproof that applies is the counts: the hooks need fewer files, fewer entries, less glue and a lower total, and the mod wins only the edit to a user file (none against one), which the plugin form of the hooks also avoids at a total of 105. The first half of the hypothesis, a smaller surface, is therefore not supported; the second half, more capability, is not refuted but does not show up in the nine capabilities: the mod tells a prompt's origin, keeps its user line out of the transcript, makes a model call in 23 tokens without provisioning a credential or starting a process, and sees a step before its tool runs, while a hook set is smaller, runs in its own process (a crash is a notice row, a slow job cannot hold Claude Code's worker), works without the early-access mods runtime and, in its plugin form, installs the same way. What the hook surface can do that the mod cannot, as far as this run and the Hooks reference show: handler types that call a model or an HTTP endpoint without code (`prompt`, `agent`, `http`, `mcp_tool`), `async` with `asyncRewake`, `onFailure: "block"`, and hooks in managed settings that keep running when mods are restricted (documented, not run); none was needed for the nine. Decision (Joe, 2026-10-14): H1 is disproved on the surface clause and its capability half is confirmed; the Claude Code side stays a mod ([793c73](./793c_open-questions.md#793c73-does-the-systems-claude-code-side-stay-a-mod-now-that-h1-is-disproved-by-its-own-rule), [793b352](./793b_decision-log.md#793b352-owner-answer-793c73-the-systems-claude-code-side-stays-a-mod)).

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

Result of the run on this machine: [793d18d Results](#793d18d-results).

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

### 793d18d Results

Run on 2026-10-13 on this machine: Claude Code 2.1.296, Agent SDK sessions on the subscription login, a replay mod of 40 lines that relays prepared requests to `$.model.complete` and writes one file per call (wall time around the call by `$.clock.now()`, the whole result with its `usage`). The requests are the proof of concept's own: `classifyRequest`, `relevanceRequest`, `captureRequest` and `curationRequest` of `judge.js`, called unchanged on the fixtures, with only `model` replaced by the arm's alias, the same `maxTokens` and effort, and a 120 s timeout; no fixture has two different requests across arms and runs (checked by hash over the 610 fixtures). The sources it built the requests from have these SHA-256 prefixes: `judge.js` 751f4b8fe729, `moment.js` 7b5958cbcf46, `capture.js` 8580c8bd4e07, `housekeeping.js` 17f4260042a8. The debug log names the model each alias resolved to: `claude-haiku-5-5` (1,830 calls), `claude-sonnet-5-5` (1,830), `claude-opus-5-5` (921) and `claude-fable-5-1` (375, and 199 of the adjudication). Six sessions ran at once in each arm's phase, in the order haiku, sonnet, opus, fable, each session working through its calls one after another; the sessions themselves ran on haiku and made no judgement. 4,956 calls were made in all: none failed, none was cut by the 120 s timeout, no rate limit was reached, and 2 opus `classify` calls returned an empty reply. The fixture sets are the confirmed first tranche ([793b346 The confirmed gold: final fixture counts and the share of drafts the owner changed](./793b_decision-log.md#793b346-the-confirmed-gold-final-fixture-counts-and-the-share-of-drafts-the-owner-changed)): 200 `classify`, 130 `relevance`, 130 `capture` and 150 `curation` fixtures, so the samples support wider margins than the full sets of the plan, which the tables state. Opus judged a stratified half (nested in the full set) and fable a fifth. The replay cost 32.49 USD at the list prices of 2026-10-06 (haiku 0.79, sonnet 9.69, opus 10.40, fable 11.62), the adjudication 2.43 USD and the sessions' own turns under 0.1 USD, about 35 USD in all against the 43 USD the owner confirmed. The observations (`run/out/*.json`, `run/report*.json`, `run/report*.md`) are ephemeral under `ENGRAM_EVAL_DIR` ([793b343 The hypothesis-eval artifacts are ephemeral and kept out of the repository](./793b_decision-log.md#793b343-the-hypothesis-eval-artifacts-are-ephemeral-and-kept-out-of-the-repository)); the tables below are the durable record. How the run was built and scored: [793b347 The H2 replay is a mod that relays prepared requests, run as ephemeral scripts](./793b_decision-log.md#793b347-the-h2-replay-is-a-mod-that-relays-prepared-requests-run-as-ephemeral-scripts) and [793b348 How the four kinds are scored, and how the verdict rule is applied](./793b_decision-log.md#793b348-how-the-four-kinds-are-scored-and-how-the-verdict-rule-is-applied).

The finding that governs how the quality results are read: the PoC's prompts and strict parsers are tuned to the cheapest tier, and sonnet, given the same request, often does not reply in the form the parsers read. By the rule of 793d18 a reply that is not read is a miss, and that is the reading of the verdict; 793d18d3 shows how large the effect is and what the other readings give.

#### 793d18d1 Scenario results

| Scenario | Result | What the run recorded (the reading of the rule: strict parsers) |
| --- | --- | --- |
| [793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), `classify` | Inconclusive | primary clears (haiku 0.612 against 0.450, lower bound +9.8 points); guard does not (named a moment where the gold is none 0.310 against 0.337, upper bound +5.3 points against 5.0) |
| the same, `relevance` | Does not hold | primary clears (recall 0.858 against 0.380); the guard fails the other way: haiku delivers a note the gold excludes in 0.538 of the fixtures against 0.131, lower bound +33.1 points. Sonnet's reply is not read in 62.6% of its calls, which lowers its recall and flatters its guard (793d18d3) |
| the same, `capture` | Inconclusive | lesson recall 0.289 against 0.390, difference -10.1 points, bounds -17.8 to -2.6: neither parity nor disproof; the window guard clears (0.048 against 0.060, upper bound +1.6); the covered-as-absent guard has 1 lesson and cannot be assessed |
| the same, `curation` | Inconclusive | agreement 0.889 against 0.736 (lower bound +9.3); destructive errors 0.049 against 0.013, difference +3.6 points, bounds +0.9 to +6.4: neither clears nor disproves |
| [793d18a1b A gap inside the noise is inconclusive, not a tie](#793d18a1-experiments) | Applied | three kinds are marked inconclusive on a bound that straddles the margin, with the fixtures that would bring it inside in the table of 793d18d2 (105 for `classify`'s guard, 555 for `curation`'s guard); `capture`'s recall gap, -10.1 points, is larger than its noise floor, 9.0, and no number of fixtures changes that at the observed difference |
| [793d18a1c Subtle cases are not hidden by easy ones](#793d18a1-experiments) | Inconclusive | subtle-stratum lesson recall 0.250 against 0.339 (36 fixtures), difference -8.9 points, bounds -17.0 to -0.7: not parity, not disproof |
| [793d18b1a The cheapest tier is faster per call](#793d18b1-experiments), `classify` | Not faster | median 1.70 s against 1.66 s (haiku minus sonnet +0.04 s, 95% interval -0.04 to +0.13 s: a tie); calls over the 4 s deadline 0.7% against 1.2% |
| the same, `relevance` | Faster | median 2.47 s against 2.75 s (-0.28 s, 95% interval -0.42 to +0.12 s: inside the noise); calls over the 3 s deadline 25.9% against 35.9% |
| the same, `capture` | Not faster | median 2.47 s against 1.31 s (+1.16 s, 95% interval +0.56 to +1.74 s); none over the 40 s deadline for either |
| the same, `curation` | Not faster | median 2.40 s against 1.28 s (+1.11 s, 95% interval +0.87 to +1.66 s); none over the 60 s deadline for haiku, 0.2% for sonnet |
| [793d18b1b The cheapest tier is cheaper per fixture](#793d18b1-experiments) | Holds | cost per call 0.00025, 0.00039, 0.00077 and 0.00041 USD against 0.00340, 0.00582, 0.00899 and 0.00416 USD: sonnet costs 13.8, 14.9, 11.7 and 10.1 times as much for `classify`, `relevance`, `capture` and `curation` |
| [793d18c1a Every request names the model of its kind](#793d18c1-experiments) | Holds | `tests/tiers.test.ts` builds a request of each kind: each names `judgeTiers`' model for its kind and none names the session model |
| [793d18c1b Raising one kind changes only that kind](#793d18c1-experiments) | Holds | the same file raises each kind in turn to `sonnet`: that kind's request changes in its model alone and the other three are equal to their earlier requests; a kind without a row throws; 141 contract tests pass under `claude plugin test`, and 2 of the file's 3 tests fail when the table is bypassed |

#### 793d18d2 Criteria tables

Arms side by side, per kind. The primary measure is agreement (`classify`, `curation`) or recall (`relevance`, lesson recall of `capture`, by the terms); the guard is the rate named in the cell. The sonnet arm's run-to-run spread of its primary measure (largest minus smallest of its three whole-set means) is 4.0, 2.8, 3.0 and 2.0 score points for `classify`, `relevance`, `capture` and `curation`. Wall time is read around each call in the mod, with six sessions at once; the interleaved control below has the sequential figures.

| Kind | Arm | Fixtures (n) | Calls (n) | Replies the PoC parsers do not read (% of calls) | Primary: agreement or recall (share) | Guard (share) | Median latency (s) | p95 latency (s) | Calls over the PoC deadline (%) | Cost (USD per call) | Cost of the arm (USD) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| classify | haiku | 200 | 600 | 0.0 | 0.612 (n=200) | moment named where the gold is none 0.310 (n=100) | 1.70 | 2.99 | 0.7 | 0.00025 | 0.15 |
| classify | sonnet | 200 | 600 | 26.8 | 0.450 (n=200) | moment named where the gold is none 0.337 (n=100) | 1.66 | 2.49 | 1.2 | 0.00340 | 2.04 |
| classify | opus (indicative) | 100 | 300 | 1.7 | 0.693 (n=100) | moment named where the gold is none 0.387 (n=50) | 1.90 | 3.17 | 0.0 | 0.00725 | 2.18 |
| classify | fable (indicative) | 40 | 120 | 10.0 | 0.567 (n=40) | moment named where the gold is none 0.350 (n=20) | 1.32 | 3.57 | 3.3 | 0.01647 | 1.98 |
| relevance | haiku | 130 | 390 | 0.0 | 0.858 (n=54) | a gold-excluded note delivered 0.538 (n=130); any note delivered where the gold is none 0.583 (n=76) | 2.47 | 4.22 | 25.9 | 0.00039 | 0.15 |
| relevance | sonnet | 130 | 390 | 62.6 | 0.380 (n=54) | a gold-excluded note delivered 0.131 (n=130); any note delivered where the gold is none 0.118 (n=76) | 2.75 | 3.82 | 35.9 | 0.00582 | 2.27 |
| relevance | opus (indicative) | 65 | 195 | 0.5 | 0.944 (n=27) | a gold-excluded note delivered 0.569 (n=65); any note delivered where the gold is none 0.588 (n=38) | 2.14 | 3.23 | 8.7 | 0.01026 | 2.00 |
| relevance | fable (indicative) | 27 | 81 | 8.6 | 0.955 (n=11) | a gold-excluded note delivered 0.444 (n=27); any note delivered where the gold is none 0.375 (n=16) | 1.52 | 8.60 | 40.7 | 0.02414 | 1.95 |
| capture | haiku | 130 | 390 | 3.3 | 0.289 (n=46) | lesson written in a window with none 0.048 (n=83); subtle recall 0.250 (n=36) | 2.47 | 12.09 | 0.0 | 0.00077 | 0.30 |
| capture | sonnet | 130 | 390 | 3.1 | 0.390 (n=46) | lesson written in a window with none 0.060 (n=83); subtle recall 0.339 (n=36) | 1.31 | 5.75 | 0.0 | 0.00899 | 3.51 |
| capture | opus (indicative) | 66 | 198 | 0.0 | 0.562 (n=24) | lesson written in a window with none 0.040 (n=42); subtle recall 0.534 (n=19) | 2.01 | 10.92 | 0.0 | 0.02048 | 4.06 |
| capture | fable (indicative) | 27 | 81 | 11.1 | 0.450 (n=10) | lesson written in a window with none 0.059 (n=17); subtle recall 0.424 (n=8) | 5.11 | 24.75 | 0.0 | 0.06500 | 5.26 |
| curation | haiku | 150 | 450 | 2.0 | 0.889 (n=150) | destructive error 0.049 (n=150) | 2.40 | 5.68 | 0.0 | 0.00041 | 0.19 |
| curation | sonnet | 150 | 450 | 11.8 | 0.736 (n=150) | destructive error 0.013 (n=150) | 1.28 | 3.66 | 0.2 | 0.00416 | 1.87 |
| curation | opus (indicative) | 76 | 228 | 0.0 | 0.860 (n=76) | destructive error 0.048 (n=76) | 1.89 | 5.98 | 0.0 | 0.00948 | 2.16 |
| curation | fable (indicative) | 31 | 93 | 2.2 | 0.914 (n=31) | destructive error 0.032 (n=31) | 3.40 | 10.02 | 0.0 | 0.02604 | 2.42 |

Differences from sonnet on the fixtures an arm shares with it (arm minus sonnet; paired two-stage bootstrap of 10,000 resamples, seed 793023; the noise floor is the larger of the sonnet arm's run spread and the half-width of the 95% interval; the supported margin is 1.645 times the real spread over the square root of n). The real spread of the primary measures is 0.43 to 0.52 for `classify`, `relevance` and `curation`, above the assumed 0.4, so their supported margins are 5.8 to 10.1 points and not the 4.7 to 5.0 of the plan; for the `capture` measures and the `curation` guard it is 0.16 to 0.26, below it.

| Kind | Arm | Measure | Fixtures (n) | Difference from sonnet (score points) | One-sided 95% lower bound (score points) | One-sided 95% upper bound (score points) | Noise floor (score points) | Real spread (SD of fixture differences) | Margin the sample supports (score points) | Fixtures that would bring the bound inside the margin (n) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| classify | haiku | agreement with the gold moments | 200 | +16.2 | +9.8 | +22.7 | 7.6 | 0.517 | 6.0 | clear |
| classify | haiku | moment named where the gold is none | 100 | -2.7 | -10.3 | +5.3 | 9.2 | 0.451 | 7.4 | 105 |
| classify | opus (indicative) | agreement with the gold moments | 100 | +20.3 | +13.0 | +27.7 | 8.7 | 0.415 | 6.8 | clear |
| classify | opus (indicative) | moment named where the gold is none | 50 | -0.7 | -12.0 | +10.7 | 13.3 | 0.454 | 10.6 | 200 |
| classify | fable (indicative) | agreement with the gold moments | 40 | +8.3 | -4.2 | +21.7 | 15.0 | 0.476 | 12.4 | clear |
| classify | fable (indicative) | moment named where the gold is none | 20 | -8.3 | -20.0 | +5.0 | 15.0 | 0.303 | 11.2 | clear |
| relevance | haiku | recall of the gold-relevant notes | 54 | +47.8 | +37.3 | +58.6 | 12.7 | 0.452 | 10.1 | clear |
| relevance | haiku | a gold-excluded note delivered | 130 | +40.8 | +33.1 | +48.5 | 9.4 | 0.522 | 7.5 | never |
| relevance | haiku | a note delivered where the gold is none | 76 | +46.5 | +36.8 | +56.1 | 11.4 | 0.496 | 9.4 | never |
| relevance | opus (indicative) | recall of the gold-relevant notes | 27 | +57.4 | +42.6 | +71.6 | 17.3 | 0.465 | 14.7 | clear |
| relevance | opus (indicative) | a gold-excluded note delivered | 65 | +43.1 | +32.8 | +53.3 | 12.3 | 0.496 | 10.1 | never |
| relevance | opus (indicative) | a note delivered where the gold is none | 38 | +44.7 | +29.8 | +58.8 | 17.1 | 0.527 | 14.1 | never |
| relevance | fable (indicative) | recall of the gold-relevant notes | 11 | +51.5 | +28.8 | +72.7 | 27.3 | 0.456 | 22.6 | clear |
| relevance | fable (indicative) | a gold-excluded note delivered | 27 | +38.3 | +24.7 | +51.9 | 16.0 | 0.410 | 13.0 | never |
| relevance | fable (indicative) | a note delivered where the gold is none | 16 | +29.2 | +12.5 | +45.8 | 18.8 | 0.382 | 15.7 | never |
| capture | haiku | lesson recall (terms) | 46 | -10.1 | -17.8 | -2.6 | 9.0 | 0.262 | 6.4 | never |
| capture | haiku | lesson written in a window with none | 83 | -1.2 | -4.4 | +1.6 | 3.8 | 0.160 | 2.9 | clear |
| capture | haiku | covered gold lesson written as absent | 1 | not assessable (1 fixture) | | | | | | |
| capture | haiku | lesson recall, subtle stratum | 36 | -8.9 | -17.0 | -0.7 | 9.7 | 0.237 | 6.5 | never |
| capture | opus (indicative) | lesson recall (terms) | 24 | +13.0 | +3.5 | +22.6 | 11.4 | 0.243 | 8.2 | clear |
| capture | opus (indicative) | lesson written in a window with none | 42 | -0.8 | -4.8 | +2.4 | 4.8 | 0.116 | 2.9 | clear |
| capture | opus | covered gold lesson written as absent | 0 | not assessable (0 fixture) | | | | | | |
| capture | opus (indicative) | lesson recall, subtle stratum | 19 | +9.8 | -0.4 | +20.6 | 12.6 | 0.257 | 9.7 | clear |
| capture | fable (indicative) | lesson recall (terms) | 10 | +20.0 | +2.8 | +38.1 | 21.2 | 0.286 | 14.9 | clear |
| capture | fable (indicative) | lesson written in a window with none | 17 | +0.0 | +0.0 | +0.0 | 0.0 | 0.000 | 0.0 |  |
| capture | fable | covered gold lesson written as absent | 0 | not assessable (0 fixture) | | | | | | |
| capture | fable (indicative) | lesson recall, subtle stratum | 8 | +13.9 | -3.5 | +31.9 | 20.8 | 0.264 | 15.4 | clear |
| curation | haiku | agreement (outcome and note) | 150 | +15.3 | +9.3 | +21.3 | 7.1 | 0.433 | 5.8 | clear |
| curation | haiku | destructive error | 150 | +3.6 | +0.9 | +6.4 | 3.2 | 0.190 | 2.5 | 555 |
| curation | opus (indicative) | agreement (outcome and note) | 76 | +11.0 | +3.5 | +18.4 | 9.0 | 0.394 | 7.4 | clear |
| curation | opus (indicative) | destructive error | 76 | +3.5 | +0.0 | +7.5 | 4.6 | 0.193 | 3.6 | 475 |
| curation | fable (indicative) | agreement (outcome and note) | 31 | +15.1 | +2.2 | +29.0 | 15.6 | 0.446 | 13.2 | clear |
| curation | fable (indicative) | destructive error | 31 | +1.1 | -5.4 | +6.5 | 7.5 | 0.182 | 5.4 | 71 |

Speed control. To rule out that the order of the arm phases (haiku first) or the six sessions at once explain the speed result, one session ran 20 fixtures of each kind sequentially, haiku and sonnet alternating on the same fixture: median latency in seconds, haiku against sonnet, `classify` 1.76 against 1.59, `relevance` 2.61 against 2.59, `capture` 3.76 against 1.69, `curation` 2.87 against 1.27 (p95 3.42 against 2.23, 9.51 against 3.96, 13.37 against 7.31, 6.20 against 3.78). The pattern is the main run's. Haiku's output tokens per call (201, 374, 802 and 508 for the four kinds, against 50, 173, 164 and 98 for sonnet) exceed what its reply text needs, as the reply is one short JSON document; that points to thinking tokens counted in `usage`, which the run did not read, and they would set its latency.

Effort note (added 2026-10-14, after the owner asked whether the run used the lowest effort). The replay relayed the PoC's own requests, which set `effort: 'low'` for `classify` and `relevance` and `'medium'` for `capture` and `curation`; so the recorded run used the floor (`low`, the lowest value this build accepts) for two kinds and one step above it for the other two. A latency-only control of 960 calls on 40 fixtures a kind, haiku and sonnet, run one call at a time, with `capture` and `curation` at `low`, found: haiku's output falls by 26 percent (`capture`, 778 to 577 tokens per call) and 11 percent (`curation`, 467 to 417), its median moves by -0.18 and +0.23 s (inside the noise), and it is still slower than sonnet at the floor (`capture` median 2.40 against 1.74 s, `curation` 2.35 against 1.63 s; paired, haiku minus sonnet +0.88 s and +0.56 s); agreement with the gold moves by +0.054 (`capture`, ±0.139) and -0.063 (`curation`, ±0.072) for haiku, intervals that include zero. `usage` does not separate thinking tokens (they are inside `output_tokens`); haiku's estimated thinking is 160 to 660 tokens a call at every setting, sonnet's under 30. The verdict above is not changed by this note. The numbers and the decision are in [793b351 The H2 replay ran at the PoC's effort (low and medium), not at the floor; a latency control at the floor does not flip the speed clause](./793b_decision-log.md#793b351-the-h2-replay-ran-at-the-pocs-effort-low-and-medium-not-at-the-floor-a-latency-control-at-the-floor-does-not-flip-the-speed-clause).

#### 793d18d3 Replies the parsers do not read, and the other readings

The PoC reads a reply with strict parsers (one JSON document, optionally in a fence that wraps the whole reply) and does nothing with a reply it cannot read. The share of calls not read: classify 0.0% (haiku), 26.8% (sonnet), 1.7% (opus), 10.0% (fable), relevance 0.0% (haiku), 62.6% (sonnet), 0.5% (opus), 8.6% (fable), capture 3.3% (haiku), 3.1% (sonnet), 0.0% (opus), 11.1% (fable), curation 2.0% (haiku), 11.8% (sonnet), 0.0% (opus), 2.2% (fable). Sonnet and fable write a prose analysis before the JSON document: 161 of sonnet's 600 `classify` replies and 243 of its 390 `relevance` replies end in a valid JSON document after prose; haiku's unread replies are 13 of its 390 `capture` replies (4 end before the document is complete) and 9 of its 450 `curation` replies. Scored as a miss (the rule), this lowers sonnet's agreement and recall; and since an unread reply delivers and writes nothing, it also lowers sonnet's guard rates. The same differences under the two other readings (lenient: a reader takes the last JSON document of a reply, which the PoC does not; read-only: the unread replies are left out of the means instead of scored as misses; the speed and cost are the same under each):

| Kind | Reading | Haiku primary | Sonnet primary | Primary difference (score points) | Primary lower bound (score points) | Guard difference (score points) | Guard upper bound (score points) | Guard lower bound (score points) | Parity reading |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| classify | strict | 0.612 | 0.450 | +16.2 | +9.8 | -2.7 | +5.3 | -10.3 | inconclusive |
| classify | lenient | 0.612 | 0.608 | +0.3 | -6.0 | -23.7 | -16.0 | -31.0 | inconclusive |
| classify | readonly | 0.612 | 0.610 | +0.6 | -6.1 | -21.2 | -13.7 | -29.2 | inconclusive |
| relevance | strict | 0.858 | 0.380 | +47.8 | +37.3 | +40.8 | +48.5 | +33.1 | disproved |
| relevance | lenient | 0.858 | 0.830 | +2.8 | -4.3 | +1.3 | +7.9 | -5.4 | inconclusive |
| relevance | readonly | 0.858 | 0.907 | +1.2 | -3.7 | +5.2 | +16.1 | -5.7 | inconclusive |
| capture | strict | 0.289 | 0.390 | -10.1 | -17.8 | -1.2 | +1.6 | -4.4 | inconclusive |
| capture | lenient | 0.289 | 0.390 | -10.1 | -17.8 | -1.2 | +1.6 | -4.4 | inconclusive |
| capture | readonly | 0.325 | 0.428 | -11.3 | -19.5 | -1.2 | +1.6 | -4.4 | inconclusive |
| curation | strict | 0.889 | 0.736 | +15.3 | +9.3 | +3.6 | +6.4 | +0.9 | inconclusive |
| curation | lenient | 0.889 | 0.811 | +7.8 | +2.7 | +1.6 | +4.7 | -1.6 | parity |
| curation | readonly | 0.906 | 0.831 | +5.9 | +1.0 | +3.2 | +6.2 | +0.2 | inconclusive |

Under either other reading haiku and sonnet agree with the gold to within a few points on `classify` and `relevance` (0.612 against 0.608 and 0.858 against 0.830 under the lenient reading), the `relevance` guard difference falls from +40.8 to +1.3 points, and `curation` is at parity when read leniently. The result for the System is not that sonnet judges worse: it is that the PoC's request, as written for the cheapest tier, does not get the form it reads from sonnet, so raising a kind to sonnet needs a request sonnet obeys (and the table of `judgeTiers` alone is not enough). `capture` is the same under all three readings.

`capture` is scored by the terms a note must hold ([793b331 H2's capture is scored by the terms a note must hold](./793b_decision-log.md#793b331-h2s-capture-is-scored-by-the-terms-a-note-must-hold)). To measure the scoring error, fable judged blind (no arm named, replies shuffled) whether the written lessons of each reply state each gold lesson, for the 27 fixtures with a gold lesson on which haiku and sonnet differ in the term match, all 3 runs of both arms (162 replies, 1 not answered in the format after a retry).

| Arm | Reply and gold lesson pairs (n) | Term match and judgement agree (share) | Term match says stated, judgement says not (share) | Term match says not stated, judgement says stated (share) | Recall by the term match (share) | Recall by the fable judgement (share) |
| --- | --- | --- | --- | --- | --- | --- |
| haiku | 177 | 0.599 | 0.034 | 0.367 | 0.373 | 0.665 |
| sonnet | 172 | 0.738 | 0.052 | 0.209 | 0.553 | 0.693 |
| both | 349 | 0.668 | | | | |

The term match misses a stated lesson in a third of the pairs, and misses more of haiku's (0.367) than of sonnet's (0.209): haiku words the same lesson differently. On these 27 fixtures the recall gap between the arms is 0.180 by the terms and 0.028 by the judgement. The fixtures were chosen because the arms differ in the term match, so the gap is not a whole-set estimate; if the 19 other fixtures with a gold lesson, where the arms tie by the terms, tie by the judgement too, the whole-set gap would be about 0.02 and not 0.10. The verdict of `capture` is by the pre-registered terms (inconclusive); the scoring error is large enough, and one-sided enough, that the terms are not a trustworthy score of haiku against sonnet. The adjudication is a single fable reading of each reply, not a second gold.

Flags from the indicative arms (haiku minus the arm on the arm's fixtures; a shortfall beyond the margin the arm's sample supports is flagged and is not a disproof): `capture` lesson recall, haiku below opus by 18.9 points (supported margin 7.1) and below fable by 23.9 points (11.2); the subtle stratum likewise (opus 20.0, fable 19.4). No other kind or guard is flagged. The larger tiers read better than sonnet on the PoC's request (opus is +20.3 points above sonnet in `classify` agreement on its fixtures and +57.4 points in `relevance` recall), which is largely the unread replies again.

#### 793d18d4 Verdict

By the rule as written (strict parsers; unread is a miss):

| Condition of 793d18 | `classify` | `relevance` | `capture` | `curation` |
| --- | --- | --- | --- | --- |
| Parity, primary (lower bound at least -5.0 points) | yes (+9.8) | yes (+37.3) | no (-17.8; upper bound -2.6, so not disproved) | yes (+9.3) |
| Parity, guard (upper bound at most +5.0 points) | no (+5.3; not disproved) | no, disproved (lower bound +33.1) | yes for the window guard (+1.6); the covered guard has 1 lesson | no (+6.4; lower bound +0.9, not disproved) |
| Parity, subtle stratum (`capture`) | | | no (-17.0; upper bound -0.7, not disproved) | |
| Median latency below sonnet's | no (1.70 against 1.66 s, a tie) | yes (2.47 against 2.75 s, inside the noise) | no (2.47 against 1.31 s) | no (2.40 against 1.28 s) |
| Cost per fixture below sonnet's | yes (13.8 times) | yes (14.9 times) | yes (11.7 times) | yes (10.1 times) |
| Table (793d18c1a, 793d18c1b) | yes | yes | yes | yes |
| Kind | Inconclusive | Disproved (guard) | Inconclusive | Inconclusive |
| Kind, under the lenient and read-only readings | Inconclusive (primary lower bound -6.0 and -6.1) | Inconclusive (primary -4.3 and -3.7; guard upper bound +7.9 and +16.1) | Inconclusive | Parity read leniently (+2.7; +4.7), inconclusive read-only |

Verdict: H2 is DISPROVED as a whole on this machine by its own rule, which disproves H2 when the cheapest tier is not faster or not cheaper than the sonnet arm on the fixtures: haiku is cheaper in all four kinds (10.1 to 14.9 times), but its median call is slower than sonnet's in three kinds (`classify` by a tie, `capture` by 1.16 s, `curation` by 1.11 s; 2.10 s against 1.63 s over all the calls) and faster only in `relevance`, inside the noise. The speed result does not depend on the reading of the replies, and the sequential control repeats it. Per kind:

- `classify`: inconclusive under every reading. By the rule haiku agrees with the gold moments more than sonnet (0.612 against 0.450, lower bound +9.8 points) and its guard bound, +5.3 points, is 0.3 point beyond the margin (105 fixtures instead of 100 would bring it inside); read leniently the agreement is equal (0.612 against 0.608) and the primary bound, -6.0 points, is 1.0 point beyond. Not faster (a tie), 13.8 times cheaper.
- `relevance`: disproved by the rule, on its guard: haiku delivers a gold-excluded note in 0.538 of the fixtures against 0.131 (lower bound +33.1 points above the margin). The disproof is an effect of sonnet's unread replies (62.6% of its calls deliver nothing); read leniently the kinds are at 0.858 and 0.830 and the guard difference is +1.3 points (upper bound +7.9), which is inconclusive. The rule's consequence, raising `relevance` in `judgeTiers` and measuring again, would raise it to a tier whose request the PoC reads in 37% of its calls. 25.9% of haiku's calls and 35.9% of sonnet's took longer than the 3 s the PoC gives the judgement.
- `capture`: inconclusive under every reading. Haiku's lesson recall by the terms is below sonnet's by 10.1 points (bounds -17.8 to -2.6) and below opus and fable beyond their supported margins (flagged); the term match, though, misses a stated lesson in 0.367 of haiku's pairs and 0.209 of sonnet's, and by the blind judgement the gap on the fixtures where they differ shrinks from 0.180 to 0.028. The window guard clears; the covered guard cannot be assessed (1 gold lesson). Haiku is 1.16 s slower per call; the call is background work with a 40 s deadline, which no call approached.
- `curation`: inconclusive by the rule. Haiku agrees with the gold more than sonnet (0.889 against 0.736, lower bound +9.3 points); its destructive errors are 0.049 against 0.013 (difference +3.6, bounds +0.9 to +6.4), which neither clears the margin nor disproves it, and 555 fixtures at the observed spread would settle it; read leniently parity holds. It is 1.11 s slower per call (a background job with a 60 s deadline) and 10.1 times cheaper.

Scope: this machine, Claude Code 2.1.296 on a subscription login, the confirmed first tranche (130 to 200 fixtures a kind, not the 200 and 280 of the plan), the models the aliases resolved to on 2026-10-13, the prices of 2026-10-06, the PoC's requests as they were at the commit of this run. The decisions are [793b349 H2 is disproved as a whole on this machine by its speed clause; relevance is disproved on its guard by the rule, and the unread replies are the reason](./793b_decision-log.md#793b349-h2-is-disproved-as-a-whole-on-this-machine-by-its-speed-clause-relevance-is-disproved-on-its-guard-by-the-rule-and-the-unread-replies-are-the-reason) and the question [793c79](./793c_open-questions.md#793c79-what-follows-from-h2-being-disproved-on-speed-and-for-relevance). Decision (Joe, 2026-10-14): H2 is disproved on speed; every judgement runs on haiku at `low` effort ([793b353](./793b_decision-log.md#793b353-owner-answer-793c79-every-system-judgement-runs-on-haiku-at-the-lowest-accepted-effort)).

## 793d19 Design as currently drafted

These are the author's choices so far. They are not requirements of this spec and are not claimed to be right; each is recorded, with its rationale and the alternatives, in the decision log, and they move to the implementation plan once the hypotheses are settled.

- **The split between the [mod](#793d1b-terms) and the engram commands**, with [judgements](#793d1b-terms) relayed through the mod (the mod orchestrates over the existing commands and no Go runner is built, per [793b256 The PoC orchestrates in JavaScript inside the mod over the existing engram commands](./793b_decision-log.md#793b256-the-poc-orchestrates-in-javascript-inside-the-mod-over-the-existing-engram-commands)): [793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go), [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter).
- **Where the mod's source lives, and how `engram update` installs it:** [793b190 The mod's source lives in agent-instructions/mods and is written in JavaScript](./793b_decision-log.md#793b190-the-mods-source-lives-in-agent-instructionsmods-and-is-written-in-javascript), [793b191 engram update installs the mod as a skills-directory plugin](./793b_decision-log.md#793b191-engram-update-installs-the-mod-as-a-skills-directory-plugin).
- **[Delivery](./793a_product-spec.md#793a1-problem-statement) channels** (prompt [context](./793a_product-spec.md#793a1-problem-statement), tool-result context, subagent-start context): [793b192 Delivery uses prompt context, tool-result context and subagent-start context](./793b_decision-log.md#793b192-delivery-uses-prompt-context-tool-result-context-and-subagent-start-context).
- **[Moments](./793a_product-spec.md#793a1-problem-statement):** the cues and the classification: [793b193 Moments are mechanical cues; the prompt lookup spans recall's four moments](./793b_decision-log.md#793b193-moments-are-mechanical-cues-the-prompt-lookup-spans-recalls-four-moments), [793b239 Author choices in applying 793b238, shown to the owner](./793b_decision-log.md#793b239-author-choices-in-applying-793b238-shown-to-the-owner); [subagents](../../GLOSSARY.md#subagent): [793b194 Subagents get a dispatch delivery and a capture at stop; moments inside a subagent are not detected](./793b_decision-log.md#793b194-subagents-get-a-dispatch-delivery-and-a-capture-at-stop-moments-inside-a-subagent-are-not-detected); tier evidence at routing: [793b218 793c64 is answered (a): tier evidence is delivered when the orchestrator routes through route](./793b_decision-log.md#793b218-793c64-is-answered-a-tier-evidence-is-delivered-when-the-orchestrator-routes-through-route).
- **Presence and the delivery ledger:** [793b195 Presence is read from the live context and a delivery ledger](./793b_decision-log.md#793b195-presence-is-read-from-the-live-context-and-a-delivery-ledger).
- **Judgement as relayed completions, and the tier table:** [793b196 Judgement runs as relayed model completions, not sessions](./793b_decision-log.md#793b196-judgement-runs-as-relayed-model-completions-not-sessions), [793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier); the tier and the effort are haiku at `low` for all four kinds, by the owner ([793b353](./793b_decision-log.md#793b353-owner-answer-793c79-every-system-judgement-runs-on-haiku-at-the-lowest-accepted-effort)); the mod stays a mod ([793b352](./793b_decision-log.md#793b352-owner-answer-793c73-the-systems-claude-code-side-stays-a-mod)).
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
