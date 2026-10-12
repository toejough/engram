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
| 1 | A plugin whose `hooks/hooks.json` lists `modules` loads a module exporting `register(on, options)`; handlers are `($, e, next)`; Claude Code 2.1.287 or later | Verified (docs) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 2 | A folder under `~/.claude/skills` holding `.claude-plugin/plugin.json` loads as a plugin in every [session](../../GLOSSARY.md#session), with no flag, install step or settings edit; its hooks module loads, also through a symlink into the engram root | Plugin load: Verified (docs); hooks module and symlink: Verified (tested) Also tested (SP2): the SDK `plugins` option loads it; a refused module is found in the debug log, not on `stderr`. | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 3 | `prompt.submit` fires for every origin (composer, task notification, scheduled trigger, peer [session](../../GLOSSARY.md#session)) and `next({ ...e, context })` adds text the model reads after the prompt as typed | Verified (tested, SP3): `prompt.submit` fired for a typed prompt and for `/recall ...` (origin `sdk` under the driver) and the context reached the model | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 4 | `additionalContext` from `classic.PostToolUse`, `classic.PostToolUseFailure` and `classic.SubagentStart` reaches the model; [context](./793a_product-spec.md#793a1-problem-statement) added by a `tool.call` hook appears in the [transcript](../../GLOSSARY.md#transcript) as an `attachment` row; whether `prompt.submit` context is stored as a transcript row | Verified (tested, SP4): context is a separate `attachment` row, `attachment.type` `hook_additional_context`, in the transcript that received it; `prompt.submit` and `classic.PostToolUse` context in the main transcript, `classic.SubagentStart` context only in the subagent's | [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments) |
| 5 | `agent.spawn` carries the Agent call's `prompt`, `description` and `fork`; `classic.SubagentStop` carries `agent_transcript_path` and `last_assistant_message` | Verified (tested, SP5): the fields are as declared; the subagent transcript file is not complete at the stop (its final assistant row is written after), so `last_assistant_message` is used and the file polled | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 6 | `classic.SessionStart` carries the main agent's `transcript_path` | Verified (tested, SP6): `classic.SessionStart` `transcript_path` is the session's transcript file | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 7 | `session.compact` fires for the main agent and, with `agentId`, for each [subagent](../../GLOSSARY.md#subagent) or fork compaction; `turn.complete` also fires for an interrupted turn; module variables survive `/clear`, `/resume` and `/branch` | Verified (docs, declarations) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 8 | `$.session.messages()` returns the newest 4096 messages the next model request carries, main loop only; after a compaction or a resume, and whether a [context](./793a_product-spec.md#793a1-problem-statement) block added through row 3 or 4 appears in it | Shape and limit: Verified (declarations). Tested (SP7): it holds typed texts and replies, not a delivered block; a `/compact` in the stream does not shorten it; a resume restores it. Presence is read from the ledger and the transcript file instead | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments) |
| 10 | Time inside a [mods](#793d1b-terms) API call (`$.process.run`, `$.model.complete`) does not count against a hook's 10 s budget; `.catch` handlers have 1 s | Verified (docs, declarations; tested, SP12: a 25 s `$.process.run` in a `prompt.submit` hook was not cut off) | [793d17c1b Work longer than the hook budget is awaited without the handler being cut off](#793d17c1-experiments) |
| 11 | `$.model.complete({ model, prompt, system, maxTokens, timeoutMs })` returns `{ isAnswered, text, reason, usage }`; `model` is an alias (`haiku`, `sonnet`, `opus`) or a full id, billed to the [session](../../GLOSSARY.md#session)'s credentials. That the `haiku` alias resolves on this machine's subscription login | Verified (docs, declarations; tested, SP1: `haiku` answered on this login) | [793d17c1a A model call is relayed on a named tier](#793d17c1-experiments) |
| 12 | `$.ui.status(text)` pins one line under the prompt; it is not a [transcript](../../GLOSSARY.md#transcript) row; `claude -p` draws no status line | Verified (declarations). Tested (SP9): in a stream `$.ui.status` arrives as a `system` message of subtype `ui_status`, and is not a transcript row | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) |

### 793d1b Terms

The terms **moment**, **delivery**, **lesson**, **working agent**, **context**, **vault** and **System** are as in [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement). This spec adds these:

- **Mod:** a Claude Code plugin whose `hooks/hooks.json` lists a module that Claude Code loads in process and calls with `register(on, options)`.
- **Settings hook:** a command that Claude Code runs on an event, as configured in a settings file, exchanging JSON through its standard input and output.
- **Surface:** the mod or the settings hooks, as a way to build the System's Claude Code side.
- **Hook budget:** the 10 seconds a hook handler's own time may take before Claude Code cuts it off.
- **Stand-in command:** a program that takes the place of the engram program in an experiment: it returns a fixed delivery after a set delay, so that a surface is measured apart from the real memory logic.
- **Probe:** a minimal mod module or settings hook that exercises one capability and writes what it observed to a log outside the transcript.
- **Judgement:** a model decision the System makes about memory. Its **kind** is one of: `classify` (from the recent texts, which of the four decision moments of recall the agent is about to reach, or none), `relevance` (from a moment and candidate notes, which bear on the moment), `extraction` (from a window of a session, which lessons it contains), `reconcile` (whether a candidate lesson is already covered by the nearest notes or absent) and `curation` (whether a pending offer is covered, near, absent or rejected).
- **Session model:** the model the working session is configured with.
- **Cheapest tier:** the least capable and cheapest model tier available to the session, the `haiku` family, named by an alias and not by a dated id.
- **`judgeTiers`:** the table from a judgement kind to the model its requests name; every kind defaults to the cheapest tier.
- **Fixture set:** for one kind, the inputs drawn from real sessions and the vault, each with a gold label.
- **Gold label:** the correct answer for a fixture, fixed before either tier runs and not taken from the output of either tier.
- **Margin:** the largest shortfall of the cheapest tier against the session model that still counts as parity.
- **Noise:** the spread of the session model's own result over repeated runs of the same fixtures.

## 793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks

Hypothesis: 793d17 The Claude Code side of the [System](./793a_product-spec.md#793a1-problem-statement) is better built as a [mod](#793d1b-terms) than as [settings hooks](#793d1b-terms), because the mod has a smaller implementation [surface](#793d1b-terms) and does more.

Owner's reason: the owner assumes that [mods](#793d1b-terms) offer a smaller [surface](#793d1b-terms) and more functionality, and builds the design on "the novel capabilities mods provide" ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go)); for example "with mods, the system can just print stuff to the user" ([793c40 Does the user still see please's audit list?](./793c_open-questions.md#793c40-does-the-user-still-see-pleases-audit-list)). It is an assumption to test, not a settled fact.

Scope: this machine only, a macOS terminal session and `claude -p`, signed in with a subscription, on the installed Claude Code version ([793b268 The PoC is minimal and targeted: environments the owner does not use and everything no scenario needs are cut](./793b_decision-log.md#793b268-the-poc-is-minimal-and-targeted-environments-the-owner-does-not-use-and-everything-no-scenario-needs-are-cut)). The verdict says nothing about other environments.

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

### 793d17a Capability coverage

Settles rows 3 to 8 of [793d1a What the hypotheses rely on in Claude Code mods](#793d1a-what-the-hypotheses-rely-on-in-claude-code-mods). Run: live.

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
    | a subagent's stop is observed with its transcript path, and the file read at the stop holds the subagent's last message                 |
    | the main agent's transcript path is known at the session's start                                                                         |
    | a compaction of the main agent is observed                                                                                               |

Scenario: 793d17a1b Added context is attributable in the transcript
  Given a mod that adds one block through each channel: after a prompt, next to a tool result and at a subagent's start
  When the session runs and its transcript file is read
  Then each block appears as its own row, separate from the prompt as typed and from the tool result, and names the channel it came through
  And the prompt text and the tool result are exactly as they were without the mod

Scenario: 793d17a1c The mod can read what the agent's context holds, through a compaction and a resume
  Given a mod that delivered block X to the main agent, and a probe that lists the delivered blocks found in the messages the mod can read
  When the session is compacted so that X no longer reaches the model, X is delivered again, and the session is then ended and resumed
  Then the probe lists X after its delivery, does not list X after the compaction removed it, and lists X again after the resume restored it
```

### 793d17b Output outside the agent's context

Settles row 12. Run: live.

#### 793d17b1 Experiments

```gherkin
Scenario: 793d17b1a A message reaches the user and not the agent
  Given a mod that shows the user one warning line, and a settings hook that returns the equivalent user-facing message
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
  Then the mod's call returns an answered reply with its text and usage, on the session's credentials, and the usage names a model of the haiku family
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
  And the job's result appears after about 20 seconds and, in the mod, reaches the agent at its next tool result
  And the report records how a settings hook's detached result would reach the agent
```

### 793d17d Install and load

Settles rows 1 and 2. Run: live.

#### 793d17d1 Experiments

```gherkin
Scenario: 793d17d1a The mod loads in every session from the install path
  Given the mod directory in the engram-owned root, a link to it in the skills directory, and no flag, install step or settings edit
  When a new session starts, is cleared, is resumed and is branched
  Then the mod's probe log shows its register ran and a handler fired in each
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
  And the user sees a warning outside the transcript saying what was skipped and why
  And the report counts the code each surface needed to contain this failure

  Examples:
    | cause                                                |
    | the engram command cannot start                      |
    | the engram command's output cannot be parsed         |
    | the handler or script throws                         |
    | the work passes its deadline                         |

Scenario: 793d17e1b The mod changes nothing it must not
  Given the loaded mod and an agent that runs engram query and the learn skill in a turn
  When claude plugin validate lists its handlers and Claude Code fires the tool events of those commands
  Then no handler denies, drops, or answers an event in place of continuing it
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

## 793d18 H2: the cheapest capable model is adequate for every System judgement, and is faster and cheaper

Hypothesis: 793d18 Every [judgement](#793d1b-terms) the [System](./793a_product-spec.md#793a1-problem-statement) makes (`classify`, `relevance`, `extraction`, `reconcile`, `curation`) can be made by the [cheapest tier](#793d1b-terms) as well as by the [session model](#793d1b-terms), within a [margin](#793d1b-terms), and the cheapest tier is faster and cheaper per call.

Scope: this machine only, `claude -p` on a subscription login, priced at the published price of each model ([793b268 The PoC is minimal and targeted: environments the owner does not use and everything no scenario needs are cut](./793b_decision-log.md#793b268-the-poc-is-minimal-and-targeted-environments-the-owner-does-not-use-and-everything-no-scenario-needs-are-cut)).

Owner's reason: every [System](./793a_product-spec.md#793a1-problem-statement) task that needs model [judgement](#793d1b-terms) should go to the least capable tier that can handle that kind of judgement, not the [session](../../GLOSSARY.md#session)'s model, because the automatic System must be as fast and as cheap as possible ([793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier), which records the same for the [delivery](./793a_product-spec.md#793a1-problem-statement) side in [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)). Prior evidence, not a result for this System: the engram evaluation found a haiku-family tier at parity with a top tier on subtle lesson-sifting, with the framing of the request mattering more than the tier; a kind is raised only on measured evidence.

Each kind is measured on its own [fixture set](#793d1b-terms) against its [gold labels](#793d1b-terms):

| Kind | Fixture input | [Gold label](#793d1b-terms) | Source of the gold | Primary measure | Guard measure |
| --- | --- | --- | --- | --- | --- |
| `classify` | the recent texts at a judged cue | which of the four [moments](./793a_product-spec.md#793a1-problem-statement) the agent reached next, or none | the next action in the original [transcript](../../GLOSSARY.md#transcript), confirmed by the owner for a sample | agreement with the gold [moment](./793a_product-spec.md#793a1-problem-statement) | rate of naming a [moment](./793a_product-spec.md#793a1-problem-statement) where the gold is none |
| `relevance` | a [moment](./793a_product-spec.md#793a1-problem-statement) and its candidate [notes](./793a_product-spec.md#793a1-problem-statement) | which candidates bear on it | the owner's labels | recall of the gold-relevant [notes](./793a_product-spec.md#793a1-problem-statement) | rate of delivering a [note](./793a_product-spec.md#793a1-problem-statement) the gold excludes |
| `extraction` | a window of a [session](../../GLOSSARY.md#session) | the [lessons](./793a_product-spec.md#793a1-problem-statement) it contains | the [lessons](./793a_product-spec.md#793a1-problem-statement) the original [session](../../GLOSSARY.md#session)'s learn recorded, plus the owner's additions | [lesson](./793a_product-spec.md#793a1-problem-statement) recall (one minus the miss rate) | rate of candidates that are not [lessons](./793a_product-spec.md#793a1-problem-statement) |
| `reconcile` | one candidate [lesson](./793a_product-spec.md#793a1-problem-statement) and its nearest [notes](./793a_product-spec.md#793a1-problem-statement) | covered or absent | the outcomes already decided in the [vault](./793a_product-spec.md#793a1-problem-statement)'s history | accuracy | rate of "absent" where the gold is covered (a duplicate) |
| `curation` | one pending offer and its candidates | covered, near, absent or rejected | the outcomes past curation recorded | agreement | rate of destructive errors (a fold or discard the gold does not make) |

Constants, initial values the owner can change: [margin](#793d1b-terms) 0.05, 3 runs per fixture per tier, at least 40 fixtures per kind of which at least 10 have the gold "none" or "absent", at most 200 fixtures.

Confirmed if all hold, for each kind:

- Parity: the [cheapest tier](#793d1b-terms)'s primary measure is at least the [session model](#793d1b-terms)'s minus the [margin](#793d1b-terms), and its guard measure is no worse than the session model's plus the margin, on the whole set and on its subtle stratum ([793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin](#793d18a1-experiments), [793d18a1c Subtle cases are not hidden by easy ones](#793d18a1-experiments)).
- Speed and cost: the cheapest tier's median latency per call and its cost per fixture are lower than the session model's ([793d18b1a The cheapest tier is faster per call](#793d18b1-experiments), [793d18b1b The cheapest tier is cheaper per fixture](#793d18b1-experiments)).
- Table: the kind can be raised alone, on evidence, and its request always names its own model ([793d18c1a Every request names the model of its kind](#793d18c1-experiments), [793d18c1b Raising one kind changes only that kind](#793d18c1-experiments)).

Disproved for a kind if the [cheapest tier](#793d1b-terms) falls short of parity by more than the [margin](#793d1b-terms) on either measure; that kind is then raised in [`judgeTiers`](#793d1b-terms) and measured again, and H2 holds for the other kinds. Disproved as a whole if the cheapest tier is not faster or not cheaper than the [session model](#793d1b-terms) on the fixtures. Inconclusive for a kind if the gap lies inside the [noise](#793d1b-terms) ([793d18a1b A gap inside the noise is inconclusive, not a tie](#793d18a1-experiments)).

### 793d18a Agreement with the gold labels

Run: replay of the [fixture sets](#793d1b-terms) through both tiers.

#### 793d18a1 Experiments

```gherkin
Scenario Outline: 793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin
  Given the fixture set of <kind> and the same request text for both tiers
  When each tier judges every fixture 3 times
  Then the cheapest tier's mean <primary> is at least the session model's mean minus 0.05
  And the cheapest tier's mean <guard> is no worse than the session model's plus 0.05

  Examples:
    | kind       | primary                              | guard                                                    |
    | classify   | agreement with the gold moment       | rate of naming a moment where the gold is none           |
    | relevance  | recall of the gold-relevant notes    | rate of delivering a note the gold excludes              |
    | extraction | lesson recall                        | rate of candidates that are not lessons                  |
    | reconcile  | accuracy                             | rate of "absent" where the gold is covered               |
    | curation   | agreement                            | rate of destructive errors                               |

Scenario: 793d18a1b A gap inside the noise is inconclusive, not a tie
  Given a kind whose cheapest-tier result differs from the session model's by less than the spread of the session model's own result over its 3 runs
  When the result is recorded
  Then the kind is marked inconclusive and not confirmed
  And fixtures are added, up to 200, until the gap exceeds the spread or the cap is reached

Scenario: 793d18a1c Subtle cases are not hidden by easy ones
  Given the extraction fixtures split into a stratum with an explicit correction in the user's words and a subtle stratum where the lesson is implied by what the user did
  When both tiers are scored
  Then each stratum is reported on its own
  And H2 is confirmed for extraction only if the subtle stratum also meets the parity of 793d18a1a
```

### 793d18b Speed and cost

Run: the same replay, timing and pricing each call.

#### 793d18b1 Experiments

```gherkin
Scenario Outline: 793d18b1a The cheapest tier is faster per call
  Given the fixture set of <kind> and the same request text for both tiers
  When each tier judges every fixture 3 times and each call's latency is recorded
  Then the cheapest tier's median latency is below the session model's

  Examples:
    | kind       |
    | classify   |
    | relevance  |
    | extraction |
    | reconcile  |
    | curation   |

Scenario: 793d18b1b The cheapest tier is cheaper per fixture
  Given the usage reported for each call and the published price of each model
  When the cost per fixture is computed for each kind and each tier
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
  Then the requests of that kind name the new model, the requests of the other four kinds are unchanged, and the raised kind is measured again with 793d18a1a
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
- **Capture:** the window, redaction and the [System](./793a_product-spec.md#793a1-problem-statement)'s own blocks [793b199 The capture window, redaction and the System's own blocks](./793b_decision-log.md#793b199-the-capture-window-redaction-and-the-systems-own-blocks); reconciliation [793b200 Reconciliation is covered or absent; dispatch outcomes are deduplicated mechanically](./793b_decision-log.md#793b200-reconciliation-is-covered-or-absent-dispatch-outcomes-are-deduplicated-mechanically); curation [793b201 Curation follows curate's table, including the skill-runbook case](./793b_decision-log.md#793b201-curation-follows-curates-table-including-the-skill-runbook-case); the index [793b202 The chunk index is brought up to date with engram ingest --auto after the turn's other work](./793b_decision-log.md#793b202-the-chunk-index-is-brought-up-to-date-with-engram-ingest---auto-after-the-turns-other-work).
- **Background work, the activity line and failures:** [793b203 Background work is one in-memory queue, and a final turn can be lost on exit](./793b_decision-log.md#793b203-background-work-is-one-in-memory-queue-and-a-final-turn-can-be-lost-on-exit), [793b204 The memory-activity line is the mod's status line](./793b_decision-log.md#793b204-the-memory-activity-line-is-the-mods-status-line), [793b205 Failures are classified by the mod and the runner, with named deadlines](./793b_decision-log.md#793b205-failures-are-classified-by-the-mod-and-the-runner-with-named-deadlines).
- **Backlog capture, the watermark and coexistence with intentional ingestion:** [793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](./793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact), [793b232 Backlog capture is a watermark per transcript source and a scan after each turn's own capture, and the $.store queue is dropped](./793b_decision-log.md#793b232-backlog-capture-is-a-watermark-per-transcript-source-and-a-scan-after-each-turns-own-capture-and-the-store-queue-is-dropped), [793b233 Choices made in applying 793b232, shown to the owner](./793b_decision-log.md#793b233-choices-made-in-applying-793b232-shown-to-the-owner), [793b234 Intentional ingestion in other harnesses continues; the System coexists with it through the write lock](./793b_decision-log.md#793b234-intentional-ingestion-in-other-harnesses-continues-the-system-coexists-with-it-through-the-write-lock).
- **Validating the mod through the existing integration gate:** [793b236 The mod is validated and tested through the existing integration gate, not a new check target](./793b_decision-log.md#793b236-the-mod-is-validated-and-tested-through-the-existing-integration-gate-not-a-new-check-target).
- **The initial values** of deadlines, windows and limits: [793b239 Author choices in applying 793b238, shown to the owner](./793b_decision-log.md#793b239-author-choices-in-applying-793b238-shown-to-the-owner), [793b233 Choices made in applying 793b232, shown to the owner](./793b_decision-log.md#793b233-choices-made-in-applying-793b232-shown-to-the-owner).

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
