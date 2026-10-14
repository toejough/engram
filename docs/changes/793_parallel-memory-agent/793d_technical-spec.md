# Parallel Memory Agent - Technical Spec

## 793d1 Orientation

The product scope is the intent of [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec): the [System](./793a_product-spec.md#793a1-problem-statement) remembers for the [working agent](./793a_product-spec.md#793a1-problem-statement), beside the [intentional memory](./793a_product-spec.md#793a1-problem-statement) that stays as it is. This spec does not draft a design for it. It states the owner's two assumptions about the implementation as hypotheses, and records for each the hypothesis, how it was tested, the top-level result and the decision with its rationale ([793b240 The product spec is the owner's intent; the technical spec is his two hypotheses with the scenarios that test them](./793b_decision-log.md#793b240-the-product-spec-is-the-owners-intent-the-technical-spec-is-his-two-hypotheses-with-the-scenarios-that-test-them)). The detailed results are not kept ([793b355 The technical spec keeps hypotheses, method, top-level results and decisions; the detailed results are removed](./793b_decision-log.md#793b355-the-technical-spec-keeps-hypotheses-method-top-level-results-and-decisions-the-detailed-results-are-removed)). Every other implementation choice is an author choice, recorded in the decision log and listed in [793d19 Design as currently drafted](#793d19-design-as-currently-drafted); none is a requirement here, and they move to the implementation plan.

What exists, and the gap. Every [vault](./793a_product-spec.md#793a1-problem-statement) operation the [System](./793a_product-spec.md#793a1-problem-statement) needs exists: [Query Engine](../../specs/technical/query-engine.md), [Note Authoring](../../specs/technical/note-authoring.md), [Note Amendment](../../specs/technical/note-amendment.md), [Transcript Reader](../../specs/technical/transcript-reader.md), `engram ingest --auto` and [Harness Deploy Sync](../../specs/technical/harness-deploy-sync.md). Missing is whatever runs them without the [working agent](./793a_product-spec.md#793a1-problem-statement): observing the [moments](./793a_product-spec.md#793a1-problem-statement), adding text to the agent's [context](./793a_product-spec.md#793a1-problem-statement) without its own tool call, and judging what to deliver and what to save. The two hypotheses concern exactly that.

| Hypothesis | Statement | Result |
| --- | --- | --- |
| [793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks](#793d17-h1-a-claude-code-mod-gives-a-smaller-implementation-surface-and-more-capability-than-settings-hooks) | A Claude Code [mod](#793d1b-terms) gives a smaller implementation [surface](#793d1b-terms) and more capability than [settings hooks](#793d1b-terms). | [793d17g4 Verdict](#793d17g4-verdict): the owner concludes a positive result; the surface clause is not supported, the capability clause is met; the mod is kept. |
| [793d18 H2: the cheapest capable model is adequate for every System judgement, and is faster and cheaper](#793d18-h2-the-cheapest-capable-model-is-adequate-for-every-system-judgement-and-is-faster-and-cheaper) | The cheapest capable model is adequate for every [System](./793a_product-spec.md#793a1-problem-statement) [judgement](#793d1b-terms) and is faster and cheaper than the [session](../../GLOSSARY.md#session)'s model. | [793d18d4 Verdict](#793d18d4-verdict): the owner concludes a mixed result with unexpected advantages; the speed clause is not supported; haiku at `low` is kept. |

How a hypothesis is settled. Every scenario is a runnable experiment or a contract test. The rule of each hypothesis is applied clause by clause and its output is stated per clause; the **conclusion** is the owner's ([793b356 Hypothesis conclusions are the owner's: multi-clause hypotheses are reported per clause](./793b_decision-log.md#793b356-hypothesis-conclusions-are-the-owners-multi-clause-hypotheses-are-reported-per-clause)). A clause is **supported** when its "Confirmed if" conditions hold, **not supported** when a "Disproved if" condition holds, and **inconclusive** when neither, because a result lay inside the [noise](#793d1b-terms).

### 793d1a What the hypotheses rely on in Claude Code mods

"Verified (docs)" is stated by the [Mods reference](https://code.claude.com/docs/en/plugins/mods/reference.md) and its sibling pages; "Verified (declarations)" by [claude-code.d.ts](https://github.com/anthropics/claude-code/blob/main/mods/types/claude-code.d.ts); "Verified (tested)" was observed on Claude Code 2.1.296. The [mods](#793d1b-terms) runtime is early access. Rows 9 (never existed), 15, 17, 18, 19 and 21 (the settings-hook comparison of the removed results) are retired and never reused.

| # | Capability relied on | Status | Settled by |
| --- | --- | --- | --- |
| 1 | A plugin whose `hooks/hooks.json` lists `modules` loads a module exporting `register(on, options)`; handlers are `($, e, next)`; Claude Code 2.1.287 or later | Verified (docs; tested: the debug log says `hooks module <name> loaded ... events: <the handlers>`) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 2 | A folder under `~/.claude/skills` holding `.claude-plugin/plugin.json` loads as a plugin in every [session](../../GLOSSARY.md#session), with no flag, install step or settings edit, | Verified (docs; tested, also through a symlink) | [793d17d1a The mod loads in every session from the install path](#793d17d1-experiments) |
| 3 | `prompt.submit` fires for every origin (composer, task notification, scheduled trigger, peer session) and `next({ ...e, context })` adds text the model reads after the prompt as typed | Verified (tested): all four origins fired and the context reached the model | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 4 | `additionalContext` from `classic.PostToolUse`, `classic.PostToolUseFailure` and `classic.SubagentStart` reaches the model, and appears in the [transcript](../../GLOSSARY.md#transcript) | Verified (tested): a separate `attachment` row of type `hook_additional_context`, in the transcript that received it (a subagent-start block only in the subagent's) | [793d17a1b Added context is attributable in the transcript](#793d17a1-experiments) |
| 5 | `agent.spawn` carries the Agent call's `prompt`, `description` and `fork`; `classic.SubagentStop` carries `agent_transcript_path` and `last_assistant_message` | Verified (tested): the file lacks its final row at the stop, so the file is polled | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 6 | `classic.SessionStart` carries the main agent's `transcript_path` | Verified (tested) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 7 | `session.compact` fires for the main agent and, with `agentId`, for each [subagent](../../GLOSSARY.md#subagent) compaction; module variables survive `/clear`, `/resume` and `/branch` | Verified (docs, declarations; tested: one `session.compact` per `/compact`); module variables across `/clear` stay docs-only | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 8 | `$.session.messages()` returns the newest 4096 messages the next model request carries, main loop only | Verified (tested): it holds typed texts and replies, not a delivered block, so presence is read from the delivery ledger and the transcript file | [793d17a1c The mod can read what the agent's context holds, through a compaction and a resume](#793d17a1-experiments) |
| 10 | Time inside a mods API call (`$.process.run`, `$.model.complete`) does not count against a hook's 10 s budget; `.catch` handlers have 1 s | Verified (docs, declarations; tested: a 25 s call was not cut off) | [793d17c1b Work longer than the hook budget is awaited without the handler being cut off](#793d17c1-experiments) |
| 11 | `$.model.complete({ model, prompt, system, maxTokens, timeoutMs })` returns `{ isAnswered, text, reason, usage }`; `model` is an alias or a full id, billed to the session's credentials | Verified (docs, declarations; tested: `haiku` answered on this login; the debug log names the model, `claude-haiku-5-5`) | [793d17c1a A model call is relayed on a named tier](#793d17c1-experiments) |
| 12 | `$.ui.status(text)` pins one line under the prompt and is not a transcript row | Verified (declarations; tested: in a stream it arrives as a `ui_status` message) | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) |
| 13 | `turn.step` fires once per model request, its result carries the step's `answer`, `toolUses`, `stopReason` and `usage`; a job started there can be awaited from the next `classic.PostToolUse`, whose wait is cut at about 10 s | Verified (tested, [793b320](./793b_decision-log.md#793b320-the-in-turn-recall-cue-is-built-as-793b238-decided)) | [793d17a1a Each surface is probed for each capability the product needs](#793d17a1-experiments) |
| 14 | A settings hook's environment lacks the session's credential; a nested `claude -p` answers on `haiku` with 438 tokens against 23 for `$.model.complete` | Verified (tested, [793b335](./793b_decision-log.md#793b335-the-settings-hook-column-is-the-recorders-counterpart-plus-one-script-installed-by-a-settings-merge)) | [793d17c1a A model call is relayed on a named tier](#793d17c1-experiments) |
| 16 | A settings hook's `systemMessage` is shown to the user and stored in the transcript file as a `hook_system_message` row; the mod's `$.ui.status` line is in no transcript row | Verified (tested, tmux): the model quoted neither | [793d17b1a A message reaches the user and not the agent](#793d17b1-experiments) |
| 20 | A mod timer callback that runs while the handler that set it is open puts its calls in that handler's scope and delays the next event; set as the handler's last action, it does not | Verified (tested, [793b337](./793b_decision-log.md#793b337-defects-of-the-probes-found-by-running-them-and-the-platform-behaviours-they-exposed)): a 20 s job held the next prompt 22.8 s in the first form and 2.7 s in the second | [793d17c1c Background work does not delay the next event](#793d17c1-experiments) |

### 793d1b Terms

The terms **moment**, **delivery**, **lesson**, **working agent**, **context**, **vault** and **System** are as in [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement). This spec adds these:

- **Mod:** a Claude Code plugin whose `hooks/hooks.json` lists a module that Claude Code loads in process and calls with `register(on, options)`.
- **Settings hook:** a command that Claude Code runs on an event, as configured in a settings file, exchanging JSON through its standard input and output.
- **Surface:** the mod or the settings hooks, as a way to build the System's Claude Code side.
- **Hook budget:** the 10 seconds a hook handler's own time may take before Claude Code cuts it off.
- **Stand-in command:** a program that takes the place of the engram program in an experiment, returning a fixed delivery.
- **Probe:** a minimal mod module or settings hook that exercises one capability and writes what it observed to a log outside the transcript.
- **Judgement:** a model decision the System makes about memory. Its **kind** is one of the four the proof of concept has: `classify` (which of recall's four decision moments the agent is about to reach, or none), `relevance` (which candidate notes bear on a moment), `capture` (the lessons in a window of a session, each marked covered by a nearest note or absent) and `curation` (whether a pending offer is covered, near, absent or rejected).
- **Session model:** the model the working session is configured with; for H2 on this machine, the sonnet arm, because the acceptance sessions are pinned to it.
- **Cheapest tier:** the least capable and cheapest model tier available to the session, the `haiku` family, named by an alias and not by a dated id.
- **`judgeTiers`:** the table from a judgement kind to the model its requests name; every kind defaults to the cheapest tier.
- **Fixture set:** for one kind, the inputs drawn from real sessions and the vault, each with a gold label.
- **Gold label:** the correct answer for a fixture, fixed before either tier runs and not taken from the output of either tier.
- **Margin:** the largest shortfall of the cheapest tier against the session model that still counts as parity (0.05).
- **Noise:** the larger of the spread of the session model's own result over repeated runs and the half-width of the 95% paired-bootstrap interval of a difference between two arms.
- **Cue:** an event of a session that the System treats as a moment (a prompt, a route skill result, a subagent start, a step of the agent), or the decision that one has been reached.
- **Block:** the text one delivery adds to an agent's context: the marker `[engram memory]`, a framing text and one line per note.
- **Ledger:** the set of note paths the System has delivered to one agent; the top-level agent's ledger is emptied by a compaction.
- **Window:** the lines of what the user and the agent said, labelled by role, that one capture judgement reads.
- **Nearest notes:** the vault notes closest to a window or an offer by `engram query`, which a judgement compares it with.
- **Burst:** the jobs in the System's queue at one time; the housekeeping runs after the last of them.
- **Housekeeping:** the work that follows a burst: the index, the pass over pending offers and, after a top-level turn, one backlog step.
- **Backlog source:** a session transcript file of any harness that the chunk index's manifest lists and that no capture has covered.
- **Mark:** a record in the backlog mark file that a backlog source, by its hash, or the turns of a session were captured.
- **Cut-off:** the fixed time, the install time minus 7 days, before which a backlog source is history.
- **Status line:** the one line the mod pins under the prompt, which is not a row of the transcript.

## 793d17 H1: a Claude Code mod gives a smaller implementation surface and more capability than settings hooks

Hypothesis: 793d17 The Claude Code side of the [System](./793a_product-spec.md#793a1-problem-statement) is better built as a [mod](#793d1b-terms) than as [settings hooks](#793d1b-terms), because the mod has a smaller implementation [surface](#793d1b-terms) and does more.

Owner's reason: the owner assumes that [mods](#793d1b-terms) offer a smaller [surface](#793d1b-terms) and more functionality, and builds the design on "the novel capabilities mods provide" ([793b189 The System is a Claude Code mod and a Go command, with the logic in Go](./793b_decision-log.md#793b189-the-system-is-a-claude-code-mod-and-a-go-command-with-the-logic-in-go)); for example "with mods, the system can just print stuff to the user" ([793c40 Does the user still see please's audit list?](./793c_open-questions.md#793c40-does-the-user-still-see-pleases-audit-list)). It was an assumption to test, not a settled fact.

How it was tested. A [probe](#793d1b-terms) for each capability the product needs was built twice, as a [mod](#793d1b-terms) and as a [settings hook](#793d1b-terms), and both ran against a [stand-in command](#793d1b-terms) on Claude Code 2.1.296 on this machine only (a macOS terminal, Agent SDK sessions on sonnet signed in with the subscription token, tmux for the on-screen check). Eleven scenarios ran on each surface (22 runs, 55 sessions), each probe logging what it observed outside the [transcript](../../GLOSSARY.md#transcript). The measurement points were the probe logs, the session's debug log, the rows of the transcript file, and a count of the artifacts each surface needs, taken by rule over the probe files as built: files deployed, handlers or hook entries, edits to files the user owns, and glue lines ([793b336 The surfaces are counted by rule, over the probe files as built](./793b_decision-log.md#793b336-the-surfaces-are-counted-by-rule-over-the-probe-files-as-built)). The verdict rule:

- Confirmed if all hold: the mod achieves every capability the hooks achieve and the rest; its [context](./793a_product-spec.md#793a1-problem-statement) is added at every channel and attributable in the transcript, and it shows the user text outside it; it relays a model call on a named tier, awaits work longer than the [hook budget](#793d1b-terms) and runs background work without delaying the next event; it loads in every [session](../../GLOSSARY.md#session) from the install path; a failure inside it leaves the session unaffected and no handler denies, drops or answers an event; and its [surface](#793d1b-terms) total is lower than the hooks' with none of the four counts higher.
- Disproved if any holds: the hooks achieve a capability the mod does not, or neither does; the mod is inactive where the hooks are active; or the hooks reach the same capability set with a lower total or any lower count.

The scenarios follow, one line each.

### 793d17a Capability coverage

#### 793d17a1 Experiments

- 793d17a1a Each surface is probed for each capability the product needs: nine capabilities (prompt origins observed, text added after the prompt and next to a tool result, a subagent's start, dispatch and stop, the main transcript path, a compaction, a step with its tool calls), each as a mod and as a settings hook.
- 793d17a1b Added context is attributable in the transcript: each added block is its own `hook_additional_context` row and the prompt and tool result are unchanged.
- 793d17a1c The mod can read what the agent's context holds, through a compaction and a resume: the delivered blocks are listed after delivery, not after a compaction removed them, and again after a resume.

### 793d17b Output outside the agent's context

#### 793d17b1 Experiments

- 793d17b1a A message reaches the user and not the agent: a line shows on screen under tmux and is neither in the transcript file nor quoted by the model.

### 793d17c Model calls and long work

#### 793d17c1 Experiments

- 793d17c1a A model call is relayed on a named tier: a `haiku` call answers on the session's credentials.
- 793d17c1b Work longer than the hook budget is awaited without the handler being cut off: a 25 s stand-in command.
- 793d17c1c Background work does not delay the next event: a 20 s job, then an immediate next prompt.

### 793d17d Install and load

#### 793d17d1 Experiments

- 793d17d1a The mod loads in every session from the install path: from a link in the skills directory, with no flag, install step or settings edit, in a new, cleared, resumed and branched session.

### 793d17e Failure containment and restraint

#### 793d17e1 Experiments

- 793d17e1a A failure inside the surface leaves the session unaffected: four causes (the command cannot start, its output cannot be parsed, the handler throws, the work passes its deadline); the user is warned outside the transcript.
- 793d17e1b The mod changes nothing it must not: no handler denies, drops or answers an event, and the memory commands' input, permission and result are unchanged.

### 793d17f Surface size

#### 793d17f1 Experiments

- 793d17f1a The same behavior is counted on both surfaces: the four counts, and the capabilities achieved.

### 793d17g Results

Run on 2026-10-13 (`targ acceptance 793d17`). The observations were ephemeral and are not kept ([793b343 The hypothesis-eval artifacts are ephemeral and kept out of the repository](./793b_decision-log.md#793b343-the-hypothesis-eval-artifacts-are-ephemeral-and-kept-out-of-the-repository)).

| Measure | Mod | Settings hooks |
| --- | --- | --- |
| Capabilities achieved (of 9) | 9 | 9 |
| Surface total of the four counts, as built | 107 (8 files, 13 handlers, 0 user-file edits, 86 glue lines) | 94 (2 files, 10 entries, 1 user-file edit, 81 glue lines); 105 packaged as a plugin |
| Restraint and containment | held (four causes contained, no refusal object, 138 contract tests) | contained too; restraint not run (no System hook set exists) |
| What only the mod does | see the verdict below; it also sees a step before its tool runs | |
| What only the hooks do | | run in their own process, work without the early-access mods runtime |

#### 793d17g4 Verdict

Per clause, the measurements:

- Surface: the hooks are smaller, 94 against 107 in total (2 files against 8, 10 entries against 13, 81 glue lines against 86, one user-file edit against none); 105 packaged as a plugin.
- Capability: 9 of 9 on both surfaces. Only the mod: a model call in 23 tokens against 438 for a hook, 0 processes per tour against 27, no credential to provision, a status line off the [transcript](../../GLOSSARY.md#transcript), and the origin of a prompt known.

Rule output: surface clause: not supported. Capability clause: met on the nine; the mod's extras are measured beside them.

**Conclusion (owner, Joe, 2026-10-14):** "The mods do seem more capable in terms of ui, and we discovered that they were more efficient. I'd call that a positive result." H1 is a positive result: the mod is more capable in UI, and the unexpected gain is that it is more efficient ([793b356](./793b_decision-log.md#793b356-hypothesis-conclusions-are-the-owners-multi-clause-hypotheses-are-reported-per-clause)).

Decision (Joe, 2026-10-14): the Claude Code side stays a [mod](#793d1b-terms) ([793c73](./793c_open-questions.md#793c73-does-the-systems-claude-code-side-stay-a-mod-now-that-h1-is-disproved-by-its-own-rule), [793b352](./793b_decision-log.md#793b352-owner-answer-793c73-the-systems-claude-code-side-stays-a-mod)). Rationale: the unexpected advantages are the reason (the in-process call, no credential, the status line off the transcript, the known prompt origin) with the UI capability; the smaller [surface](#793d1b-terms) of the [settings hooks](#793d1b-terms) ([793b338](./793b_decision-log.md#793b338-h1-is-disproved-by-its-own-rule-on-this-machine)) does not outweigh them.

## 793d18 H2: the cheapest capable model is adequate for every System judgement, and is faster and cheaper

Hypothesis: 793d18 Every [judgement](#793d1b-terms) the [System](./793a_product-spec.md#793a1-problem-statement) makes (`classify`, `relevance`, `capture` and `curation`, [793b279 Judgement is one richer prompt per kind, and capture judges extraction and reconciliation in one call](./793b_decision-log.md#793b279-judgement-is-one-richer-prompt-per-kind-and-capture-judges-extraction-and-reconciliation-in-one-call)) can be made by the [cheapest tier](#793d1b-terms) as well as by the [session model](#793d1b-terms), within a [margin](#793d1b-terms), and the cheapest tier is faster and cheaper per call.

Owner's reason: every System task that needs model judgement should go to the least capable tier that can handle that kind of judgement, because the automatic System must be as fast and as cheap as possible ([793b224 Judgement runs on the cheapest capable tier](./793b_decision-log.md#793b224-judgement-runs-on-the-cheapest-capable-tier), [793b238 Delivery has two model judgements on the cheapest tier: moment classification and a relevance filter](./793b_decision-log.md#793b238-delivery-has-two-model-judgements-on-the-cheapest-tier-moment-classification-and-a-relevance-filter)).

How it was tested. A 40-line replay mod relayed the proof of concept's own request for each kind, unchanged except for the model, through `$.model.complete` over prepared [fixture sets](#793d1b-terms) ([793b265 H2 is replayed by a small mod over fixtures, and gold comes from history or the owner](./793b_decision-log.md#793b265-h2-is-replayed-by-a-small-mod-over-fixtures-and-gold-comes-from-history-or-the-owner)), 3 runs per fixture, on this machine only (Claude Code 2.1.296, subscription login, prices of 2026-10-06; [793b268 The PoC is minimal and targeted: environments the owner does not use and everything no scenario needs are cut](./793b_decision-log.md#793b268-the-poc-is-minimal-and-targeted-environments-the-owner-does-not-use-and-everything-no-scenario-needs-are-cut)). The arms were haiku (the cheapest tier), sonnet (the session model and the verdict arm), and, as indicative arms on a stratified half and a fifth of the fixtures, opus and fable. The confirmed fixtures were 200 `classify`, 130 `relevance`, 130 `capture` and 150 `curation`; every [gold label](#793d1b-terms) was drafted by fable (`curation` constructed from real notes) and confirmed by the owner for every fixture ([793b346 The confirmed gold: final fixture counts and the share of drafts the owner changed](./793b_decision-log.md#793b346-the-confirmed-gold-final-fixture-counts-and-the-share-of-drafts-the-owner-changed)). The measurement points: wall time around each call in the mod, cost as the reply's `usage` times the published price, and the reply read by the proof of concept's strict parsers, a reply not read being a miss. The primary measure is agreement with the gold (recall for `relevance` and for `capture`'s lessons); the guard is the rate of a harmful extra (a moment named where the gold is none, a note delivered that the gold excludes, a lesson written where there is none, a destructive curation error). The verdict rule, per kind: parity when the one-sided 95% lower bound of the primary difference from sonnet is at least minus the margin (0.05) and the upper bound of the guard difference is at most the margin; disproved when the primary's upper bound is below minus the margin or the guard's lower bound is above the margin; inconclusive otherwise, as when a gap lies inside the [noise](#793d1b-terms) (a paired bootstrap of 10,000 resamples). H2 is disproved as a whole if the cheapest tier is not faster or not cheaper than sonnet.

### 793d18a Agreement with the gold labels

#### 793d18a1 Experiments

- 793d18a1a The cheapest tier agrees with the gold labels as well as the session model, within the margin: primary and guard per kind, as above.
- 793d18a1b A gap inside the noise is inconclusive, not a tie.
- 793d18a1c Subtle cases are not hidden by easy ones: `capture`'s subtle stratum is scored on its own.

### 793d18b Speed and cost

#### 793d18b1 Experiments

- 793d18b1a The cheapest tier is faster per call: median latency per kind.
- 793d18b1b The cheapest tier is cheaper per fixture: cost per kind.

### 793d18c The tier table

#### 793d18c1 Experiments

- 793d18c1a Every request names the model of its kind: a contract test.
- 793d18c1b Raising one kind changes only that kind: a contract test.

### 793d18d Results

Run on 2026-10-13: 4,956 calls, none failed, about 35 USD at list price. The observations were ephemeral ([793b343](./793b_decision-log.md#793b343-the-hypothesis-eval-artifacts-are-ephemeral-and-kept-out-of-the-repository)).

| Kind | Fixtures | Agreement, haiku / sonnet (strict parsers) | Haiku cheaper (times) | Haiku faster (median call, haiku / sonnet) | Verdict |
| --- | --- | --- | --- | --- | --- |
| `classify` | 200 | 0.612 / 0.450 | 13.8 | no, a tie (1.70 / 1.66 s) | inconclusive: the guard bound is 0.3 point beyond the margin |
| `relevance` | 130 | 0.858 / 0.380 (recall) | 14.9 | yes, inside the noise (2.47 / 2.75 s) | disproved on the guard by the rule, an artifact of unread replies |
| `capture` | 130 | 0.289 / 0.390 (lesson recall) | 11.7 | no (2.47 / 1.31 s) | inconclusive: the recall bound straddles the margin |
| `curation` | 150 | 0.889 / 0.736 | 10.1 | no (2.40 / 1.28 s) | inconclusive: the destructive-error bound straddles the margin |
| all kinds | 610 | | 10.1 to 14.9 | slower in three kinds (2.10 / 1.63 s over all calls) | disproved on speed |

Sonnet's replies were unread by the strict parsers in 27% (`classify`), 63% (`relevance`), 3% (`capture`) and 12% (`curation`) of calls because it writes prose before the JSON, and an unread reply scores as a miss, so the quality gaps are largely an artifact (read leniently, `classify` is 0.612 against 0.608 and `relevance` 0.858 against 0.830). A control at the floor effort (`low`, the lowest this build accepts) left haiku slower than sonnet in `capture` and `curation` by 0.9 and 0.6 s, because haiku thinks at every setting and `$.model.complete` cannot turn that off ([793b351 The H2 replay ran at the PoC's effort (low and medium), not at the floor; a latency control at the floor does not flip the speed clause](./793b_decision-log.md#793b351-the-h2-replay-ran-at-the-pocs-effort-low-and-medium-not-at-the-floor-a-latency-control-at-the-floor-does-not-flip-the-speed-clause)).

#### 793d18d4 Verdict

Per clause, the measurements:

- Cheaper: yes, 10.1 to 14.9 times in every kind.
- Faster: no. Median call, haiku against sonnet: a tie in `classify` (1.70 / 1.66 s) and faster in `relevance` (2.47 / 2.75 s, inside the [noise](#793d1b-terms)); slower in `capture` and `curation`, by 0.6 to 0.9 s at the floor effort. `capture` and `curation` run after the turn, against deadlines no call approached.
- Agreement with the gold, haiku / sonnet: at or above sonnet in three kinds (`classify` 0.612 / 0.450, `relevance` 0.858 / 0.380, `curation` 0.889 / 0.736; strict parsers, so sonnet's figures are depressed by unread replies), inconclusive in `capture` (0.289 / 0.390).
- Structured output: haiku's replies were read in 97 to 100% of calls, sonnet's in 45 to 96% (the effort-floor control). The effort floor is `low`, and haiku still thinks at it.

Rule output: speed clause: not supported ([793b349](./793b_decision-log.md#793b349-h2-is-disproved-as-a-whole-on-this-machine-by-its-speed-clause-relevance-is-disproved-on-its-guard-by-the-rule-and-the-unread-replies-are-the-reason)). Cheaper clause: supported. Quality: no kind is confirmed.

**Conclusion (owner, Joe, 2026-10-14):** "Haiku is much cheaper, though not faster, but clearly better suited for decision style structured output. I'd call that a mixed result with unexpected advantages." H2 is a mixed result with unexpected advantages: much cheaper, not faster, and clearly better suited to decision-style structured output ([793b356](./793b_decision-log.md#793b356-hypothesis-conclusions-are-the-owners-multi-clause-hypotheses-are-reported-per-clause)).

Decision (Joe, 2026-10-14): every [judgement](#793d1b-terms) runs on haiku at `low` effort ([793c79](./793c_open-questions.md#793c79-what-follows-from-h2-being-disproved-on-speed-and-for-relevance), [793b353](./793b_decision-log.md#793b353-owner-answer-793c79-every-system-judgement-runs-on-haiku-at-the-lowest-accepted-effort)). Rationale: the unexpected advantages are the reason, namely 10 to 15 times lower cost and replies the parsers read almost always, with agreement at or above sonnet's in three kinds; the lost speed is a tie or a gain on the two judgements that gate delivery, and 0.6 to 0.9 s on background work.

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

## 793d21 System technical specification from the proof of concept

Canonical targets: the component and flow files named in 793d22 to 793d34, each by its own `Canonical target` line. Sections 793d1 to 793d20 are the change record (hypotheses, method, results, owner conclusions, design pointers, security, traceability) and are not promoted; 793d21 to 793d35 are the part that is promoted ([793b357 The technical specification is derived from the proof of concept as built; 793d21 onward is what is promoted](./793b_decision-log.md#793b357-the-technical-specification-is-derived-from-the-proof-of-concept-as-built-793d21-onward-is-what-is-promoted)).

### 793d21a Orientation

This is the technical specification of the [System](./793a_product-spec.md#793a1-problem-statement) as the proof of concept builds it: the [mod](#793d1b-terms) `agent-instructions/mods/engram-memory` and its deploy in `internal/update` and `internal/cli`. It states only what that code and its tests do; what is not built is marked `Gap` or listed in [793d35c Gaps and what is not built](#793d35c-gaps-and-what-is-not-built). It reuses unchanged the [Query Engine](../../specs/technical/query-engine.md), [Note Authoring](../../specs/technical/note-authoring.md), [Note Amendment](../../specs/technical/note-amendment.md), [Ingestion Source Discovery](../../specs/technical/ingestion-source-discovery.md) (the manifest of `engram ingest --auto`), [Command Line Surface](../../specs/technical/command-line-surface.md) (`engram show`), [Vault Write Lock](../../specs/technical/vault-write-lock.md) (held by learn and amend, which is how the [System](./793a_product-spec.md#793a1-problem-statement) coexists with intentional ingestion) and [Self Installer](../../specs/technical/self-installer.md), and extends [Harness Deploy Sync](../../specs/technical/harness-deploy-sync.md) in [793d31 Harness Deploy Sync (mod additions)](#793d31-harness-deploy-sync-mod-additions).

Reading guide. `Verified by` names contract tests: a file under `agent-instructions/mods/engram-memory/tests/` (run by `claude plugin test`; 23 tests) with the start of a test's name, or a Go test in `internal/auto`, `internal/update` or `internal/cli`. The [mod](#793d1b-terms) is an adapter ([793b368 Within reason the logic moves to Go and the mod becomes a thin adapter](./793b_decision-log.md#793b368-within-reason-the-logic-moves-to-go-and-the-mod-becomes-a-thin-adapter)): `wiring.test.ts` registers it under the test engine with the engine of the Go side scripted and the model, clock and files stubbed, and asserts what it hands over and does with the answers (jobs, relayed requests, [block](#793d1b-terms), [status line](#793d1b-terms), the queue); the Go tests of `internal/auto` run each job through `Run` in a fake world of commands, files, environment and clock, with the model played by a function of the request, and the golden tests compare the pure parts with what the JavaScript they replaced answered for a corpus (`internal/auto/testdata/golden`) ([793b359 Flows carry one interaction requirement per calling owner, component tests drive the mod's event interface, and what is not built is marked](./793b_decision-log.md#793b359-flows-carry-one-interaction-requirement-per-calling-owner-component-tests-drive-the-mods-event-interface-and-what-is-not-built-is-marked)). Each product scenario ID is a `targ acceptance` row of `dev/acceptance/scenarios.js`; flow scenarios cite their rows in comments.

Components, each with one owner ([793b358 The components that qualify and the candidates that fail](./793b_decision-log.md#793b358-the-components-that-qualify-and-the-candidates-that-fail)): [793d22 Memory Mod](#793d22-memory-mod), [793d23 Moment Detection](#793d23-moment-detection), [793d24 Memory Delivery](#793d24-memory-delivery), [793d25 Judgement Relay](#793d25-judgement-relay), [793d26 Secret Redaction](#793d26-secret-redaction), [793d27 Auto Capture](#793d27-auto-capture), [793d28 Backlog Capture](#793d28-backlog-capture), [793d29 Auto Curation](#793d29-auto-curation), [793d30 Activity Report](#793d30-activity-report), [793d31 Harness Deploy Sync (mod additions)](#793d31-harness-deploy-sync-mod-additions). Flows: [793d32 Moment Delivery Flow](#793d32-moment-delivery-flow), [793d33 Turn Memory Work Flow](#793d33-turn-memory-work-flow), [793d34 Mod Install Flow](#793d34-mod-install-flow).

Parts considered as components and folded into an owner: the [delivery](./793a_product-spec.md#793a1-problem-statement) channels (routing, not a contract apart from the Memory Mod), the delivery [ledger](#793d1b-terms) (state of Memory Delivery's decision), failure classification and the queue (neither can be tested apart from the events that feed them), the four [judgement](#793d1b-terms) kinds (they share the Judgement Relay's tier table, fencing and reading), the mark file (state of Backlog Capture), the harness-only seam log (fails necessity) and the `claude --version` check (a boundary used only by Harness Deploy Sync).

## 793d22 Memory Mod

Canonical target: `docs/specs/technical/memory-mod.md` (title `Memory Mod - Technical Spec`).

### 793d22a Orientation

Ownership: repository component (the `engram-memory` Claude Code plugin, a [mod](#793d1b-terms)). Responsibility: observes the events of a Claude Code [session](../../GLOSSARY.md#session), adds the [System](./793a_product-spec.md#793a1-problem-statement)'s delivered text to the agent's [context](./793a_product-spec.md#793a1-problem-statement) through the channel that event provides, runs the System's background work off the event, in order, and contains every failure, changing nothing else. Realization: `hooks/register.js` (the events, the channels and the relay), `Run` and `guard` in `internal/auto/relay.go`, `failureKind` in `internal/auto/failure.go`.

### 793d22b Behavior Requirements

#### 793d22b1 Events and channels

Requirement: 793d22b1 The Memory Mod MUST add a delivered [block](#793d1b-terms) to the `context` of a `prompt.submit` event, and to the `additionalContext` of a top-level route [skill](../../GLOSSARY.md#skill) result and of a [subagent](../../GLOSSARY.md#subagent) start, returning every other field as received. It MUST NOT deny, drop, answer or alter any event, tool call, permission, model choice or prompt text, MUST pass every other tool result and every other event of a subagent on unchanged, and MUST NOT ask the agent to do anything with a block but apply it.

Supports: [793a4c1a Tool calls and permissions unchanged](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1f The agent is neither required nor asked to run a memory command](./793a_product-spec.md#793a4c1-uat-scenarios)

Verified by: `wiring.test.ts` "the ledger the engine hands back goes with the next job…"; `wiring.test.ts` "a request is relayed as it came…"; `wiring.test.ts` "the result of a tool carries the block of the route lookup…"

Rationale: The [mod](#793d1b-terms) adds [context](./793a_product-spec.md#793a1-problem-statement) at every channel, attributable in the [transcript](../../GLOSSARY.md#transcript), and shows the user text outside it ([793d17g4 Verdict](#793d17g4-verdict): the owner's positive result).

##### 793d22b1a Scenarios

```gherkin
Scenario: 793d22b1a1 The prompt is returned as typed with the block beside it
  Given a note matches the prompt
  When the prompt is submitted
  Then the returned text equals the typed text
  And the block is one separate context entry
```

#### 793d22b2 Background queue and housekeeping

Requirement: 793d22b2 After answering a top-level `turn.complete` or a `classic.SubagentStop`, the Memory Mod MUST queue one capture job and start it from a timer, so the event returns before any of the job's work starts, and MUST run queued jobs one at a time in queue order. It MUST queue nothing for the `turn.complete` of a [subagent](../../GLOSSARY.md#subagent). When the last job of a [burst](#793d1b-terms) ends it MUST run the [housekeeping](#793d1b-terms) in this order: `engram ingest --auto`, the pass over pending [offers](../../GLOSSARY.md#offer) and, only when the burst holds a top-level turn, one backlog step. A failing job or step MUST NOT stop the next.

Supports: [793a4j1a The index catches up after the turn](./793a_product-spec.md#793a4j1-uat-scenarios), [793a4j1e Slow housekeeping does not stall the session](./793a_product-spec.md#793a4j1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d22b2; `wiring.test.ts` "a turn is queued before it is answered…"; `wiring.test.ts` "the jobs of a burst run one at a time…"; `wiring.test.ts` "the seam file records a job joining the queue…"

Rationale: The event returns first so that no prompt waits for the [System](./793a_product-spec.md#793a1-problem-statement)'s work, and the order lets a later capture see an earlier [note](./793a_product-spec.md#793a1-problem-statement).

##### 793d22b2a Scenarios

```gherkin
Scenario: 793d22b2a1 The turn is answered before the work, and the index follows the capture
  Given a turn completes
  When the Memory Mod handles turn.complete
  Then the event returns before the capture's first command runs
  And engram ingest --auto runs once the capture has ended
```

#### 793d22b3 Failure containment

Requirement: 793d22b3 The Memory Mod MUST contain any failure of a step it runs (a command that cannot start, passes its deadline or exits non-zero, unparseable output, an unreadable [judgement](#793d1b-terms), a thrown error): the event proceeds as without the [mod](#793d1b-terms), no error reaches the agent and no input is lost. It MUST classify the failure as `unavailable` (the engram program cannot start or the [vault](./793a_product-spec.md#793a1-problem-statement) cannot be read), `timeout` (a deadline passed or a model call was cut) or `error`, and give the Activity Report the step and class. Each command MUST have a deadline, the lookup of a [moment](./793a_product-spec.md#793a1-problem-statement) and its relevance judgement together holding the event at most 8 seconds. A lookup that finds nothing is not a failure.

Supports: [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1e No warning when nothing matched](./793a_product-spec.md#793a4b1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d22b3; `internal/auto/failure_test.go`: Test793d22b3; `internal/auto/golden_test.go`: Test793d22b3; `internal/auto/moment_test.go`: Test793d22b3; `wiring.test.ts` "793d22b3 when the program exits with an error…"; `wiring.test.ts` "a job that fails warns, and the next one still runs…"

##### 793d22b3a Scenarios

```gherkin
Scenario: 793d22b3a1 An unreadable vault leaves the prompt as typed
  Given the vault cannot be read
  When a prompt is submitted
  Then the prompt is returned as typed with no context
  And the Activity Report receives the step and the class unavailable
```

#### 793d22b4 Persisted turn captures

Requirement: 793d22b4 Before a top-level `turn.complete` returns, the Memory Mod MUST have the engine write the turn to the queue file `capture-queue.json` of the chunks directory (`enqueue`), and the capture job it then queues MUST run that entry; a turn the engine could not write MUST be captured from memory, with no warning. The engine MUST write the entry's window redacted, take the entry out of the file only when its capture has ended, and keep it when the capture fails. When a session starts (`startup` or `resume`, not `clear` or `compact`) the mod MUST have the engine hand back the entries a session left (`resume`) and queue a capture for each; the engine MUST count each entry taken up and drop one that has been taken up three times. A `classic.SubagentStop` capture is not queued in the file.

Supports: [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4j1e Slow housekeeping does not stall the session](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/queue_test.go`: Test793d22b2; `hardening.test.ts` "793d22b2 a turn is in the queue file before its hook returns"; `hardening.test.ts` "793d22b2 an entry a session left behind is captured at the next start"; `hardening.test.ts` "793d22b2 a turn the file could not take is captured from memory and nobody is warned"

Rationale: [793b398 The persisted queue is a file the engine owns, and the red tests of 793b362 became tests of the jobs the mod hands it](./793b_decision-log.md#793b398-the-persisted-queue-is-a-file-the-engine-owns-and-the-red-tests-of-793b362-became-tests-of-the-jobs-the-mod-hands-it): the file is written before the event returns so that a session that ends at once loses nothing; it is the engine's because the windows, the redaction and the failure handling are.

##### 793d22b4a Scenarios

```gherkin
Scenario: 793d22b4a1 A session that ends before its last capture finished does not lose the turn
  Given a turn that ended and was written to the queue file
  And the session ended before the capture ran
  When the next session starts
  Then the turn is captured from the file, and the file no longer holds it

Scenario: 793d22b4a2 A capture that fails stays for the next session
  Given an entry in the queue file whose capture fails
  When the capture ends
  Then the entry is still in the file
```

## 793d23 Moment Detection

Canonical target: `docs/specs/technical/moment-detection.md` (title `Moment Detection - Technical Spec`).

### 793d23a Orientation

Ownership: repository component. Responsibility: decides whether an event puts the [working agent](./793a_product-spec.md#793a1-problem-statement) at a [moment](./793a_product-spec.md#793a1-problem-statement) where memory should be delivered, which moment, and what text the lookup is about. Realization: the [cue](#793d1b-terms) wiring of `hooks/register.js`, and `stepJob` in `internal/auto/moment.go`.

### 793d23b Behavior Requirements

#### 793d23b1 Event cues

Requirement: 793d23b1 Moment Detection MUST treat each `prompt.submit` as a [moment](./793a_product-spec.md#793a1-problem-statement) whose text is the prompt, or the `<summary>` of a task notification; a route [skill](../../GLOSSARY.md#skill) result of the top-level agent as a route moment whose text is the skill's arguments, then the last prompt; and the Nth dispatch prompt of `agent.spawn` as the moment of the Nth [subagent](../../GLOSSARY.md#subagent) start. A dispatch prompt that no subagent start has taken MUST be dropped when the turn completes. It MUST NOT [cue](#793d1b-terms) on another skill, another tool result, or a subagent's own steps.

Supports: [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d23b1; `internal/auto/moment_test.go`: Test793d23b1; `wiring.test.ts` "the result of a tool carries the block of the route lookup…"; `wiring.test.ts` "a subagent gets nothing of the main agent…"

Gap: the task-notification summary text and prompts of origins other than the SDK have no test.

Rationale: Please's orientation is a prompt, and the prompt lookup already carries recall's four decision [moments](./793a_product-spec.md#793a1-problem-statement), so it needs no [cue](#793d1b-terms) of its own.

##### 793d23b1a Scenarios

```gherkin
Scenario: 793d23b1a1 A route result is a route moment
  Given the top-level agent's Skill call of route with arguments "write Go tests"
  When its result arrives
  Then the moment is the route moment with the arguments as its text
```

#### 793d23b2 In-turn cue

Requirement: 793d23b2 After a top-level step in which the agent wrote text and called a tool, Moment Detection MUST start in the background, at most one at a time, one classification of the last six texts of the user and the agent and the step's text, redacted, with no tool input or result; the reply names at most one of recall's four [moments](./793a_product-spec.md#793a1-problem-statement), each with a subject. When it names any they MUST be looked up as a step moment; when none, no query runs. The [notes](./793a_product-spec.md#793a1-problem-statement) MUST ride the next tool result of the same turn, which waits for the job up to 7 seconds, or a later one while the job is unfinished, MUST be dropped at the turn's end, and MUST NOT reach a [subagent](../../GLOSSARY.md#subagent)'s tool result. It MUST start no job in a turn whose prompt is `/recall`, `/learn`, `/curate`, `/write-memory` or contains an `engram` query, learn, amend, ingest, show or activate command, nor for a step whose tool is such a [skill](../../GLOSSARY.md#skill) or a Bash command running `engram`.

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios), [793a5k1c An intentional recall by the agent is allowed](./793a_product-spec.md#793a5k1-uat-scenarios), [793a5k1e An intentional engram query by the agent is allowed](./793a_product-spec.md#793a5k1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d23b2; `internal/auto/moment_test.go`: Test793d23b2; `wiring.test.ts` "a step that called a tool starts the job…"; `wiring.test.ts` "the next tool result waits for the job 7 s at most…"

Security: [793d19a Security](#793d19a-security), risk (1): the recent texts are redacted and fenced.

Rationale: A prompt cannot name a [moment](./793a_product-spec.md#793a1-problem-statement) the agent reaches mid-task, such as declaring the work done, so the [cue](#793d1b-terms) reads what the agent says at each step; memory the agent retrieves on purpose is [intentional memory](./793a_product-spec.md#793a1-problem-statement) and gets nothing added.

##### 793d23b2a Scenarios

```gherkin
Scenario: 793d23b2a1 A named moment is looked up and rides a later tool result
  Given a step whose text says the rename is about to be checked
  When the classification names declaring work done
  Then the next tool result carries the note
```

## 793d24 Memory Delivery

Canonical target: `docs/specs/technical/memory-delivery.md` (title `Memory Delivery - Technical Spec`).

### 793d24a Orientation

Ownership: repository component. Responsibility: given a [moment](./793a_product-spec.md#793a1-problem-statement) and its text, finds the [vault](./793a_product-spec.md#793a1-problem-statement) [notes](./793a_product-spec.md#793a1-problem-statement) that bear on it and that the receiving agent does not already hold, composes the [block](#793d1b-terms), and remembers what each agent was given. Realization: `lookup`, `deliver` and the [ledgers](#793d1b-terms) in `internal/auto/moment.go`.

### 793d24b Behavior Requirements

#### 793d24b1 Lookup and selection

Requirement: 793d24b1 For a [moment](./793a_product-spec.md#793a1-problem-statement), Memory Delivery MUST ask the Query Engine for at most 10 items, passing the moment's text (at most 300 characters) as `--text=` and a `--phrase=` beside the moment's phrases (a prompt: recall's four decision moments; a route: "routing a unit of work to a model tier"; a [subagent](../../GLOSSARY.md#subagent): "dispatching a subagent for a task"; a step: the moments named). Free text MUST be passed as `--flag=value`. It MUST consider only [notes](./793a_product-spec.md#793a1-problem-statement) (never chunks) with a situation, a score of at least 0.25 and a direct, cluster-representative or trigger provenance. When none remains after presence it MUST make no [judgement](#793d1b-terms) and deliver nothing; otherwise it MUST make one relevance judgement and deliver exactly the considered notes selected.

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d24b1; `internal/auto/moment_test.go`: Test793d24b1

##### 793d24b1a Scenarios

```gherkin
Scenario: 793d24b1a1 A selected note is delivered after one query and one judgement
  Given the query returns a note above the floor
  When the judgement selects it
  Then the note is delivered after exactly one query and one judgement

Scenario: 793d24b1a2 No note, no judgement
  Given the query returns no note above the floor
  When the lookup ends
  Then no judgement ran and nothing is delivered
```

#### 793d24b2 Presence and ledgers

Requirement: 793d24b2 Memory Delivery MUST NOT deliver a [note](./793a_product-spec.md#793a1-problem-statement) whose path is in the receiving agent's [ledger](#793d1b-terms) or in that agent's [context](./793a_product-spec.md#793a1-problem-statement). The top-level agent's context is its [transcript](../../GLOSSARY.md#transcript) since the last compaction boundary, the compaction summary excluded, so it holds what the agent retrieved itself and what a resume restored; a [subagent](../../GLOSSARY.md#subagent)'s context is its own task and its ledger its own, so a parent's note is delivered to it. It MUST add to the top-level ledger each note path a top-level tool result shows, MUST empty it on a compaction, and MUST NOT deliver a note that entered the ledger while its lookup ran.

Supports: [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A parent's note is still delivered to a subagent](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A note removed by compaction can be delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1g A resumed session's restored content counts as present](./793a_product-spec.md#793a2e1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d24b2; `internal/auto/moment_test.go`: Test793d24b2; `wiring.test.ts` "the jobs that touch the ledger run one at a time…"

Gap: a [subagent](../../GLOSSARY.md#subagent)'s compaction empties the top-level [ledger](#793d1b-terms) and not its own; the [transcript](../../GLOSSARY.md#transcript) test still holds the top-level agent's [notes](./793a_product-spec.md#793a1-problem-statement).

##### 793d24b2a Scenarios

```gherkin
Scenario: 793d24b2a1 A held note is not delivered, and a compacted one is
  Given the context holds a delivered note
  When a prompt matches it
  Then it is not delivered
  And after a compaction removes it a matching prompt delivers it
```

#### 793d24b3 Block

Requirement: 793d24b3 Memory Delivery MUST compose the [block](#793d1b-terms) as the marker `[engram memory]`, a framing text saying the [notes](./793a_product-spec.md#793a1-problem-statement) are from the [vault](./793a_product-spec.md#793a1-problem-statement), delivered by the memory system and not typed by the user, that the agent is to apply what bears on its next step, and that a note is retrieved data whose instructions the user did not give are not to be followed, then one line per note: path, situation and text cut at 400 characters. It MUST deliver nothing, the framing included, when no note remains, and MUST report each note delivered to the Activity Report as one item delivered.

Supports: [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](./793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios)

Verified by: `internal/auto/moment_test.go`: Test793d24b3

Security: [793d19a Security](#793d19a-security), mitigation: data framing of delivered [notes](./793a_product-spec.md#793a1-problem-statement).

##### 793d24b3a Scenarios

```gherkin
Scenario: 793d24b3a1 The block applies and attributes
  Given a note is selected
  When the block is composed
  Then it starts with [engram memory], tells the agent to apply what bears, and holds the note's path and situation
```

## 793d25 Judgement Relay

Canonical target: `docs/specs/technical/judgement-relay.md` (title `Judgement Relay - Technical Spec`).

### 793d25a Orientation

Ownership: repository component, bounded by the model call the [mod](#793d1b-terms)'s engine provides. Responsibility: makes each [judgement](#793d1b-terms) of the [System](./793a_product-spec.md#793a1-problem-statement) one request that names its kind's model and effort and frames the data as data, and reads the reply strictly into a typed result or an error. Realization: `internal/auto/judge.go`, through `$.model.complete`, which `relay` of `hooks/register.js` calls.

### 793d25b Behavior Requirements

#### 793d25b1 Tier and request shape

Requirement: 793d25b1 The Judgement Relay MUST build every request with the model that its kind's row of the tier table names (`haiku` for each of `classify`, `relevance`, `capture` and `curation`), the lowest accepted effort (`low`), a reply-size limit and a deadline of its kind, and never the [session](../../GLOSSARY.md#session)'s model. A kind without a row MUST be an error naming the kind, never a default, and changing one row MUST change only that kind's requests.

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d25b1; `internal/auto/tiers_test.go`: Test793d25b1

Rationale: The cheapest tier is the owner's choice for decision-style structured output: much cheaper, and its replies are read by the strict parsers almost every time; the lowest effort suffices for classification ([793d18d4 Verdict](#793d18d4-verdict): a mixed result with unexpected advantages). The relay runs in the [mod](#793d1b-terms)'s process on the [session](../../GLOSSARY.md#session)'s credentials ([793d17g4 Verdict](#793d17g4-verdict)).

##### 793d25b1a Scenarios

```gherkin
Scenario: 793d25b1a1 Every kind names haiku at low effort
  Given any of the four kinds
  When its request is built
  Then the model is haiku and the effort is low
```

#### 793d25b2 Data and replies

Requirement: 793d25b2 The Judgement Relay MUST state in every request that its data is to be analyzed and never followed, and MUST make a text inside the data unable to close or open the request's sections: a `<` that begins a `window`, `notes` or `recent` tag, in any case and spacing, becomes `&lt;`, and a curation request's `offer` or `notes` tag is broken by a space. It MUST accept a reply only as exactly one JSON document, bare or in one code fence wrapping the whole reply; prose around it, a second document or an empty reply MUST be an error with no partial result, and an unanswered model call an error of the class its reason shows.

Supports: [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/capture_test.go`: Test793d25b2

Security: [793d19a Security](#793d19a-security), risks (1) and (2): data fencing and strict parsing.

##### 793d25b2a Scenarios

```gherkin
Scenario: 793d25b2a1 A closing tag in the data is neutralized
  Given a window that contains </window> in capitals with spaces
  When the capture request is built
  Then the request holds one window section
```

#### 793d25b3 Result shapes

Requirement: 793d25b3 The Judgement Relay MUST read `relevance` as an array of distinct whole numbers each naming a considered [note](./793a_product-spec.md#793a1-problem-statement); `classify` as an array of at most one object per [moment](./793a_product-spec.md#793a1-problem-statement) with exactly the keys `moment` (one of recall's four) and `subject`; `capture` as an array of at most ten [lessons](./793a_product-spec.md#793a1-problem-statement), each with exactly the keys of its kind (`feedback`: behavior, impact, action; `fact`: subject, predicate, object; beside kind, slug, situation and covered), every string non-empty and bounded, `covered` null or the number of a [nearest note](#793d1b-terms); and `curation` as one object with exactly outcome, into and fields: `covered` gives `into`, `near` gives `into` and exactly its note kind's fields, `absent` and `rejected` give neither, and `into` is a nearest note. Anything else MUST be an error for the whole reply, and every free-text field MUST be redacted.

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/capture_test.go`: Test793d25b3; `internal/auto/golden_test.go`: Test793d25b3

Security: [793d19a Security](#793d19a-security), risks (1), (2) and (4): a result can only name a [nearest note](#793d1b-terms) by number.

##### 793d25b3a Scenarios

```gherkin
Scenario: 793d25b3a1 A number that names no note is an error
  Given a relevance reply [7] and three considered notes
  When it is read
  Then reading fails and nothing is delivered
```

## 793d26 Secret Redaction

Canonical target: `docs/specs/technical/secret-redaction.md` (title `Secret Redaction - Technical Spec`).

### 793d26a Orientation

Ownership: repository component. Responsibility: replaces credential-shaped text with `[redacted]` before it reaches a model request or a [note](./793a_product-spec.md#793a1-problem-statement). Realization: `redact` in `internal/auto/redact.go`.

### 793d26b Behavior Requirements

#### 793d26b1 Credential shapes

Requirement: 793d26b1 Secret Redaction MUST replace with `[redacted]` private keys, JWTs, provider keys (`sk`, `pk`, `rk`, GitHub, npm, GitLab, AWS, Google and Slack shapes), `Bearer` and `Basic` credentials, passwords in URLs, an `Authorization` header, a value assigned to, or following `is` after, a name such as token, secret, password, api key, credential or private key or a name ending in key or pass, a mixed run of 32 or more letters and digits, a token wrapped over lines, and a percent-encoded token that decodes to one of these. It MUST NOT replace a 40-character hexadecimal run (a git commit hash). Secrets of other shapes are the residual risk of [793d19a Security](#793d19a-security).

Supports: [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/capture_redact_test.go`: Test793d26b1; `internal/auto/golden_test.go`: Test793d26b1

Security: [793d19a Security](#793d19a-security), risk (3).

##### 793d26b1a Scenarios

```gherkin
Scenario: 793d26b1a1 A provider key is redacted and a commit hash is kept
  Given text holding a key of the sk- shape and a 40-character hexadecimal commit hash
  When it is redacted
  Then the key is replaced by [redacted]
  And the hash is unchanged
```

## 793d27 Auto Capture

Canonical target: `docs/specs/technical/auto-capture.md` (title `Auto Capture - Technical Spec`).

### 793d27a Orientation

Ownership: repository component. Responsibility: turns a conversation [window](#793d1b-terms) into [vault](./793a_product-spec.md#793a1-problem-statement) writes: reads what the user and the agent said, judges the [lessons](./793a_product-spec.md#793a1-problem-statement) against the vault's [nearest notes](#793d1b-terms), and writes each lesson once or reinforces the [note](./793a_product-spec.md#793a1-problem-statement) that already states it. Realization: `capture` in `internal/auto/capture.go`.

### 793d27b Behavior Requirements

#### 793d27b1 Windows

Requirement: 793d27b1 Auto Capture MUST build the [window](#793d1b-terms) of a completed top-level turn as lines labelled by the speaker's role and never by anything in a message's text: from the assistant message before the turn's last typed prompt through the prompt and everything after it, the answer appended when the messages lag. A message MUST be one line (line breaks shown as `⏎`), redacted and then cut at 1,200 characters, the window cut at 8,000. A tool use MUST enter only as `Agent used <tool>`, a task notification only as its summary, and a tool's input or result never. A [subagent](../../GLOSSARY.md#subagent)'s window MUST be its whole [transcript](../../GLOSSARY.md#transcript), read after its stop until the file holds the agent's last message (at most 20 reads, 500 ms apart), the last message appended if it never does.

Supports: [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios)

Verified by: `internal/auto/capture_redact_test.go`: Test793d27b1; `internal/auto/capture_test.go`: Test793d27b1; `internal/auto/golden_test.go`: Test793d27b1

Security: [793d19a Security](#793d19a-security), risks (1) and (3): only speech enters, by role, redacted before it is cut.

##### 793d27b1a Scenarios

```gherkin
Scenario: 793d27b1a1 Only speech enters the window
  Given a turn with a user correction, an agent reply and a tool call whose result holds a file
  When the window is built
  Then it holds the User and Agent lines and "Agent used <tool>" and nothing of the tool's input or result
```

#### 793d27b2 Reconciliation, writes and provenance

Requirement: 793d27b2 Given a [window](#793d1b-terms) and a source line, Auto Capture MUST do nothing when no line is the user's. Otherwise it MUST ask the Query Engine for the window's [nearest notes](#793d1b-terms) (the user's lines first), add the [notes](./793a_product-spec.md#793a1-problem-statement) the [session](../../GLOSSARY.md#session) already wrote, keep at most 8, and make one capture [judgement](#793d1b-terms). A [lesson](./793a_product-spec.md#793a1-problem-statement) that a nearest note covers MUST only reinforce that note (`engram amend --target <note> --activate`); any other MUST be written with `engram learn <kind> --slug <slug> --position top --source=<source> --situation=<situation>` and its kind's fields, every value redacted, counted as one note added and kept as a nearest note for later windows. An unreadable reply writes nothing, and a failing judgement or write MUST NOT break the turn. The source line MUST say the note was captured from the conversation and written automatically, naming the session and turn (and [subagent](../../GLOSSARY.md#subagent)), or, for a window read after the fact, the file and position.

Supports: [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a5k1-uat-scenarios), [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios)

Verified by: `internal/auto/capture_test.go`: Test793d27b2; `internal/auto/golden_test.go`: Test793d27b2

Security: [793d19a Security](#793d19a-security), mitigation: a provenance line on every automatic [note](./793a_product-spec.md#793a1-problem-statement).

##### 793d27b2a Scenarios

```gherkin
Scenario: 793d27b2a1 An absent lesson is written once with its provenance
  Given the window holds the correction "use targ, not go test directly" and no nearest note states it
  When the capture judgement names it
  Then engram learn writes one feedback note with the automatic source line
  And one note is counted as added

Scenario: 793d27b2a2 A covered lesson only reinforces
  Given a nearest note states the lesson
  When the judgement marks it covered
  Then engram amend --activate runs on that note and engram learn does not run
```

## 793d28 Backlog Capture

Canonical target: `docs/specs/technical/backlog-capture.md` (title `Backlog Capture - Technical Spec`).

### 793d28a Orientation

Ownership: repository component. Responsibility: captures the [lessons](./793a_product-spec.md#793a1-problem-statement) of [session](../../GLOSSARY.md#session) [transcripts](../../GLOSSARY.md#transcript) nobody captured, from any [harness](../../GLOSSARY.md#harness), one [backlog source](#793d1b-terms) at a time after a turn, writing only its own [marks](#793d1b-terms). Realization: `backlog` and `markTurns` in `internal/auto/backlog.go`.

### 793d28b Behavior Requirements

#### 793d28b1 Eligible sources and cut-off

Requirement: 793d28b1 After a top-level turn's [housekeeping](#793d1b-terms) has brought the chunk index's manifest up to date, Backlog Capture MUST list the manifest's `.jsonl` entries that are neither a [subagent](../../GLOSSARY.md#subagent)'s [transcript](../../GLOSSARY.md#transcript) nor a duplicate, leaving out the live [session](../../GLOSSARY.md#session)'s file, a source changed in the last 15 minutes, a source whose recorded hash is its present hash, and a session whose turns the [System](./793a_product-spec.md#793a1-problem-statement) captured itself unless the source changed more than 10 seconds after that capture. At its first run it MUST record once the install time and a [cut-off](#793d1b-terms) of that time minus 7 days in the mark file and keep it; no source is eligible before one is recorded, one changed before the cut-off is history and MUST NOT be captured, and one changed after it MUST stay eligible until captured, however long the machine was idle. It MUST take the most recently changed first, look at no more than 20 and capture at most one a turn. An absent manifest is an empty backlog. The manifest and marks are in `ENGRAM_CHUNKS_DIR`, else the data directory's `chunks`.

Supports: [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios)

Verified by: `internal/auto/backlog_test.go`: Test793d28b1; `internal/auto/golden_test.go`: Test793d28b1

Security: [793d19a Security](#793d19a-security), bounds what is read: [sessions](../../GLOSSARY.md#session) that ended after the install time minus 7 days.

##### 793d28b1a Scenarios

```gherkin
Scenario: 793d28b1a1 A quiet uncaptured source is captured and marked
  Given a Pi transcript quiet for an hour, after the cut-off, with no mark
  When a top-level turn's housekeeping ends
  Then the source is captured and marked with its hash

Scenario: 793d28b1a2 History is not mined
  Given a source that changed before the cut-off
  When the backlog runs
  Then it is not captured
```

#### 793d28b2 Reading and windows

Requirement: 793d28b2 Backlog Capture MUST read a source's conversation from the roles of its rows (Claude Code `user` and `assistant` rows, Pi `message` rows of those roles), so that thinking, tool calls, tool results, compaction summaries and text a [harness](../../GLOSSARY.md#harness) injected never enter. It MUST split the conversation into [windows](#793d1b-terms) of at most 8,000 characters, each starting with the last message of the one before, judge up to six a turn from the end of the conversation backward through Auto Capture with a source naming the file and position, read a file over the platform's 4 MiB limit from its last 3,000,000 bytes without the cut first line, and skip for the rest of the process a source it cannot read.

Supports: [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios)

Verified by: `internal/auto/backlog_test.go`: Test793d28b2; `internal/auto/golden_test.go`: Test793d28b2

Security: [793d19a Security](#793d19a-security), risk (1): a planted [session](../../GLOSSARY.md#session) file is read by role and redacted.

##### 793d28b2a Scenarios

```gherkin
Scenario: 793d28b2a1 A Pi transcript yields only speech
  Given a Pi file with thinking, a tool call, a tool result and two messages
  When it is read
  Then only the two messages enter the windows
```

#### 793d28b3 Marks and read-only sources

Requirement: 793d28b3 Backlog Capture MUST mark a source with its hash only when every [window](#793d1b-terms) has been judged without a failure, keep the count of windows judged otherwise (starting again if the source has grown), mark at once a source with nothing to judge, and leave a failed source unmarked for the next turn. It MUST write nothing of its own but the mark file and never a source, so that a source of another [harness](../../GLOSSARY.md#harness) is not altered.

Supports: [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios)

Verified by: `internal/auto/backlog_test.go`: Test793d28b3; `internal/auto/golden_test.go`: Test793d28b3

##### 793d28b3a Scenarios

```gherkin
Scenario: 793d28b3a1 A failed window leaves no mark
  Given a source whose second window fails
  When the turn ends
  Then the source has no mark and the first window's progress is kept
```

## 793d29 Auto Curation

Canonical target: `docs/specs/technical/auto-curation.md` (title `Auto Curation - Technical Spec`).

### 793d29a Orientation

Ownership: repository component. Responsibility: settles the [vault](./793a_product-spec.md#793a1-problem-statement)'s pending [offers](../../GLOSSARY.md#offer) as the curate [skill](../../GLOSSARY.md#skill) does, accepting, folding or discarding each. Realization: `curate`, `curateOne` and `nearest` in `internal/auto/curate.go`.

### 793d29b Behavior Requirements

#### 793d29b1 Offers to settle

Requirement: 793d29b1 Auto Curation MUST find pending [offers](../../GLOSSARY.md#offer) by listing the [vault](./793a_product-spec.md#793a1-problem-statement)'s [notes](./793a_product-spec.md#793a1-problem-statement) whose front matter has `pending: true`, in name order, and settle at most 10 a run; the vault is `ENGRAM_VAULT_PATH`, else the vault under the engram data directory, and one that does not exist yet has none and is no failure. It MUST treat only a file name engram could have written as an offer. It MUST leave to curate, without counting it against the cap, an offer that is a [skill](../../GLOSSARY.md#skill) registration or a [runbook](../../GLOSSARY.md#runbook) and a note no longer pending, and MUST NOT look again in this process at an offer it left or one that failed (warned once).

Supports: [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios), [793a4j1d The agent is not asked to curate](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d29b1; `internal/auto/golden_test.go`: Test793d29b1

##### 793d29b1a Scenarios

```gherkin
Scenario: 793d29b1a1 Pending offers are settled up to the cap, a skill registration is not
  Given twelve pending offers, one a skill registration
  When the pass runs
  Then ten are settled in name order and the registration is not judged or counted
```

#### 793d29b2 Outcomes and guards

Requirement: 793d29b2 For each [offer](../../GLOSSARY.md#offer) Auto Curation MUST read it with `engram show`, take as [nearest notes](#793d1b-terms) the [note](./793a_product-spec.md#793a1-problem-statement) it amends or else the [vault](./793a_product-spec.md#793a1-problem-statement)'s five nearest by `engram query` (which leaves pending offers out), redact both, make one curation [judgement](#793d1b-terms), and run the commands of the outcome, passing the offer's [exchange hash](../../GLOSSARY.md#exchange-hash) when it has one so that an offer changed since is not settled: `covered`: `engram amend --target <note> --activate`, then the offer `--discard --into <note>`; `near`: the offer `--discard --into <note>`, then the note amended with the rewritten fields and a provenance line (`--source=amended automatically from offer <offer>`, [793d29b3 Amend provenance](#793d29b3-amend-provenance)); `absent`: the offer `--clear-pending`; `rejected`: the offer `--discard`, never `--into`. It MUST count a settled offer as one curated and a `near` note as one amended, and an offer whose commands fail as neither. It MUST pass an offer's text only as `--flag=value`, check a note name before showing or amending it, settle nothing on a reply not exactly of the expected shape, and run only these commands, naming only a nearest note by its number.

Supports: [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios), [793a4j1d The agent is not asked to curate](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d29b2; `internal/auto/golden_test.go`: Test793d29b2

Security: [793d19a Security](#793d19a-security), risks (2) and (4): a crafted [offer](../../GLOSSARY.md#offer) cannot reach a [note](./793a_product-spec.md#793a1-problem-statement) it is not judged against; a covered offer is recoverable only from [vault](./793a_product-spec.md#793a1-problem-statement) history.

##### 793d29b2a Scenarios

```gherkin
Scenario: 793d29b2a1 A covered offer reinforces and folds, a near offer folds first
  Given a pending offer a note covers and another that adds a claim to a note
  When the judgements say covered and near
  Then the first note is activated and its offer discarded into it
  And the second offer is discarded into its note before that note is amended

Scenario: 793d29b2a2 A malformed reply settles nothing
  Given a curation reply with an unknown outcome
  When the offer is judged
  Then no amend command runs
```

#### 793d29b3 Amend provenance

Requirement: 793d29b3 `engram amend --source <line>` MUST append the line to the `source:` of the note it amends, after what is there and separated from it by `; `, once (a line the `source:` already holds is not added again), and never replace what was there. It MUST do so for a fact, a feedback and a runbook note, and a `--source` alone MUST NOT be a content change, so nothing is embedded again. Auto Curation MUST pass the line `amended automatically from offer <offer>` on the amend of a `near` outcome, so that a [note](./793a_product-spec.md#793a1-problem-statement) whose claim an automatic judgement rewrote from an [offer](../../GLOSSARY.md#offer) says so. This is a revision of the canonical [note-amendment](../../specs/technical/note-amendment.md) component (a new flag beside `--chunk-source`) and of [note-revision](../../specs/product/note-revision.md), applied at promotion ([793b399 A near fold carries a provenance line, by a new flag of engram amend](./793b_decision-log.md#793b399-a-near-fold-carries-a-provenance-line-by-a-new-flag-of-engram-amend)).

Supports: [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/cli/amend_source_test.go`: Test793d29b3; `internal/auto/curate_test.go`: Test793d29b2

Security: [793d19a Security](#793d19a-security), mitigation: a provenance line on every automatic [note](./793a_product-spec.md#793a1-problem-statement); the note a near fold rewrites was the one write path that had none.

##### 793d29b3a Scenarios

```gherkin
Scenario: 793d29b3a1 A near fold says where the new claim came from
  Given a note whose source is "test" and a near offer folded into it
  When the note is amended with the rewritten fields
  Then its source is "test; amended automatically from offer <the offer>"
  And a second amend with the same line leaves it as it was
```

## 793d30 Activity Report

Canonical target: `docs/specs/technical/activity-report.md` (title `Activity Report - Technical Spec`).

### 793d30a Orientation

Ownership: repository component, bounded by the [mod](#793d1b-terms)'s [status line](#793d1b-terms). Responsibility: tells the user what the [System](./793a_product-spec.md#793a1-problem-statement) did and what it skipped, on one status line outside the agent's [context](./793a_product-spec.md#793a1-problem-statement). Realization: `summaryJob` in `internal/auto/summary.go`, `warn` in `internal/auto/relay.go`, and `report` in `hooks/register.js`.

### 793d30b Behavior Requirements

#### 793d30b1 Counts and summary line

Requirement: 793d30b1 Activity Report MUST keep the counts of items delivered, [notes](./793a_product-spec.md#793a1-problem-statement) added, notes amended and [offers](../../GLOSSARY.md#offer) curated since its last line, across the [session](../../GLOSSARY.md#session) and its [subagents](../../GLOSSARY.md#subagent). Once the work of a top-level turn has finished it MUST show one line `Memory activity: ` and each non-zero count, singular or plural, in that order (`3 items delivered, 1 note added and 2 offers curated`), then reset the counts. A turn with nothing to show MUST show nothing, or clear the line a previous turn left, and a turn that completes during another [burst](#793d1b-terms)'s [housekeeping](#793d1b-terms) MUST be reported by the burst that holds its own work. A reinforced note is neither added nor amended, and a failed step counts nothing.

Supports: [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1b Nothing is shown after a turn without events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a5h1e The counts cover the events since the previous summary](./793a_product-spec.md#793a5h1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d30b1; `internal/auto/golden_test.go`: Test793d30b1; `internal/auto/summary_test.go`: Test793d30b1; `wiring.test.ts` "a quiet turn shows nothing and clears nothing…"

##### 793d30b1a Scenarios

```gherkin
Scenario: 793d30b1a1 The line counts the turn
  Given 3 items delivered, 1 note added, 1 note amended and 2 offers curated since the last line
  When the turn's work has finished
  Then the line is "Memory activity: 3 items delivered, 1 note added, 1 note amended and 2 offers curated"
```

#### 793d30b2 Warnings and surface

Requirement: 793d30b2 When a step reports a failure, Activity Report MUST show at once `Automatic memory was skipped because <the vault is unavailable|the System timed out|of an error> (<step>)`; the line after the turn MUST carry each warning of the turn once, then the summary, joined by `. `, and the next turn MUST start clean. It MUST show a line only on the [mod](#793d1b-terms)'s [status line](#793d1b-terms), which is not a row of the [transcript](../../GLOSSARY.md#transcript), MUST NOT put it in any hook result or [context](./793a_product-spec.md#793a1-problem-statement), and MUST build it only from constants and counts, never from the text of a [note](./793a_product-spec.md#793a1-problem-statement), an [offer](../../GLOSSARY.md#offer) or a [lesson](./793a_product-spec.md#793a1-problem-statement).

Supports: [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios), [793a4b1c No error reaches the agent and the user is warned](./793a_product-spec.md#793a4b1-uat-scenarios), [793a5h1c The summary is not in the agent's context](./793a_product-spec.md#793a5h1-uat-scenarios), [793a4c1c The only effect on the working agent's context is what is delivered](./793a_product-spec.md#793a4c1-uat-scenarios)

Verified by: `internal/auto/summary_test.go`: Test793d30b2; `wiring.test.ts` "the effects of a round are applied…"

Security: [793d19a Security](#793d19a-security), mitigation: the user-visible activity line, which cannot carry injected text.

Rationale: The [status line](#793d1b-terms) is a [mod](#793d1b-terms) capability that sits off the [transcript](../../GLOSSARY.md#transcript), so the user sees the [System](./793a_product-spec.md#793a1-problem-statement)'s work without the agent reading it ([793d17g4 Verdict](#793d17g4-verdict)).

##### 793d30b2a Scenarios

```gherkin
Scenario: 793d30b2a1 A timeout is shown with its step and kept out of the transcript
  Given a step of the prompt lookup passes its deadline
  When it is reported
  Then the status line is "Automatic memory was skipped because the System timed out (the prompt lookup)"
  And no hook result holds it
```

## 793d31 Harness Deploy Sync (mod additions)

### 793d31a Orientation

Canonical target: [harness-deploy-sync](../../specs/technical/harness-deploy-sync.md), as Added requirements; they also Modify its 792b29a Orientation, whose deploy set (skills, and guidance while it is managed) gains the mods of `agent-instructions/mods`. Realization: `planModCopies`, `modNotice` and `HarnessSpec.TakesMods` in `internal/update/update.go`, and `writeSkillRows` in `internal/cli/update.go`. The dependency boundary added is the `claude --version` command.

### 793d31b Behavior Requirements

#### 793d31b1 Mod deploy set

Requirement: 793d31b1 For every [harness](../../GLOSSARY.md#harness) that takes [mods](#793d1b-terms) (Claude Code only), Harness Deploy Sync MUST plan one copy of every file under `agent-instructions/mods/`, keeping its path under the harness's [skills](../../GLOSSARY.md#skill) directory and naming the mod's top-level directory as its unit, so that a folder holding `.claude-plugin/plugin.json` loads as a plugin in every Claude Code [session](../../GLOSSARY.md#session). It MUST plan nothing, without an error, when the source has no mods directory, MUST fail with the cause when the listing fails, and MUST plan no mod for Pi nor create a Pi directory.

Supports: [793a5j1a Update installs the System for the harness it updates](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1d One update serves both harnesses differently](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5c1a Unsupported harness](./793a_product-spec.md#793a5c1-uat-scenarios), [793a5c1b A Pi session on a machine whose Claude Code has the System](./793a_product-spec.md#793a5c1-uat-scenarios)

Verified by: `internal/update/mods_test.go`: Test793e8e1a, 1b, 1c, 1d and 1f (Pi absent and present)

##### 793d31b1a Scenarios

```gherkin
Scenario: 793d31b1a1 Only Claude Code receives the mod
  Given a source with the engram-memory mod and both harnesses present
  When the copies are planned
  Then every file is planned under the Claude Code skills directory and none for Pi
```

#### 793d31b2 Mods are synchronized like skills

Requirement: 793d31b2 Harness Deploy Sync MUST deploy a [mod](#793d1b-terms) into the engram-owned root and link it from the [harness](../../GLOSSARY.md#harness)'s [skills](../../GLOSSARY.md#skill) directory, and keep it synchronized as it keeps a skill: a second run changes nothing, a dry run writes nothing and still reports the mod, a mod removed from the source is removed with its link, and no skill link the update deploys is removed or altered.

Supports: [793a5j1a Update installs the System for the harness it updates](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1c A preview installs none of the System](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1l Running update twice changes nothing](./793a_product-spec.md#793a5j1-uat-scenarios)

Verified by: `internal/update/mods_test.go`: Test793e8e1e, 1g, 1h, 1i

Rationale: One state to reason about: deploying the [mod](#793d1b-terms) turns the [System](./793a_product-spec.md#793a1-problem-statement) on, and removing it from the source turns it off, with everything [intentional memory](./793a_product-spec.md#793a1-problem-statement) relies on left in place.

##### 793d31b2a Scenarios

```gherkin
Scenario: 793d31b2a1 The mod is deployed beside the skills
  Given a source with the mod and the learn skill
  When the update runs
  Then the mod is linked from the skills directory and the learn skill's link is unchanged
```

#### 793d31b3 Version check and report rows

Requirement: 793d31b3 When the source holds a [mod](#793d1b-terms), Harness Deploy Sync MUST read `claude --version` once per run, a dry run included, and give the [harness](../../GLOSSARY.md#harness) an empty mod notice when Claude Code is 2.1.287 or later, otherwise a sentence naming the reason: the version is older, the version cannot be read, or `claude` is not on the path. A notice MUST NOT stop the deploy. With no mod in the source it MUST NOT run `claude`. It MUST list each deployed mod in the harness section beside the [skills](../../GLOSSARY.md#skill) as `agent-instructions/mods/<name>/ → <destination>  (<n> files)`, and print a non-empty notice as `mods: <notice>` under them.

Supports: [793a5j1a Update installs the System for the harness it updates](./793a_product-spec.md#793a5j1-uat-scenarios)

Verified by: `internal/update/mods_test.go`: Test793e8e2a, 2b, 2c; `internal/cli/update_mods_test.go`: Test793e8e3a, 3b

Rationale: The [System](./793a_product-spec.md#793a1-problem-statement) is inactive on a Claude Code that cannot load [mods](#793d1b-terms), so the report says so rather than leaving the user to find out.

##### 793d31b3a Scenarios

```gherkin
Scenario: 793d31b3a1 An older Claude Code is reported and the deploy goes on
  Given claude --version prints 2.1.286
  When the update runs
  Then the mod notice names 2.1.286 and 2.1.287 and the deploy completes
```

## 793d32 Moment Delivery Flow

Canonical target: `docs/specs/technical/moment-delivery-flow.md` (title `Moment Delivery Flow - Technical Spec`).

A [moment](./793a_product-spec.md#793a1-problem-statement) event of a Claude Code [session](../../GLOSSARY.md#session) (a prompt, a route result, a [subagent](../../GLOSSARY.md#subagent) start, or an agent step) is turned into delivered [vault](./793a_product-spec.md#793a1-problem-statement) [notes](./793a_product-spec.md#793a1-problem-statement) added to the receiving agent's [context](./793a_product-spec.md#793a1-problem-statement), or into nothing. Each interaction requirement has its owning caller; each product scenario ID below is a `targ acceptance` row.

### 793d32a Sequence

```mermaid
sequenceDiagram
    participant Eng as Claude Code
    participant Mod as Memory Mod
    participant Det as Moment Detection
    participant Del as Memory Delivery
    participant JR as Judgement Relay
    participant Red as Secret Redaction
    participant QE as Query Engine
    participant Act as Activity Report

    Eng->>Mod: 793d22b1, 793d32b1 prompt, route result, subagent start or step
    Mod->>Det: 793d23b1, 793d32b1 what does this event cue
    alt a prompt, a route result or a subagent start
        Det-->>Mod: 793d23b1 the moment and its text
        Mod->>Del: 793d24b1, 793d32b1 look up the moment for this agent
    else a step with text and a tool call
        Det->>Det: 793d23b2 no job when the agent runs memory itself
        Det->>Red: 793d26b1, 793d32b2 redact the recent texts
        Det->>JR: 793d25b1, 793d32b2 classification request
        JR-->>Det: 793d25b2, 793d25b3 the moments named
        Det->>Del: 793d24b1, 793d32b2 look up the named moments
        Note over Det,Del: the notes ride a later tool result of the same turn
    else nothing is cued
        Det-->>Mod: 793d23b1 none
    end
    Del->>QE: 792b24c5, 793d32b3 query with the moment's phrases
    QE-->>Del: 792b24c5 items
    Del->>Del: 793d24b2 drop the notes the agent holds
    alt a considered note remains
        Del->>JR: 793d25b1, 793d25b2, 793d32b3 relevance request
        JR-->>Del: 793d25b2, 793d25b3 the selection
        Del-->>Mod: 793d24b3 the block
        Mod->>Act: 793d30b1 items delivered
        Mod-->>Eng: 793d22b1, 793d32b1 the event with the block added
    else none remains
        Del-->>Mod: 793d24b1 nothing
        Mod-->>Eng: 793d22b1, 793d32b1 the event as received
    end
    alt any step fails
        Mod->>Act: 793d30b2, 793d22b3, 793d32b1 warning with step and class
        Mod-->>Eng: 793d22b3, 793d32b1 the event as received
    end
```

### 793d32b Interaction Requirements

#### 793d32b1 Moment routing

Requirement: 793d32b1 On an event the Memory Mod MUST first ask Moment Detection what it [cues](#793d1b-terms), and run no lookup for an event that cues nothing. For a cued [moment](./793a_product-spec.md#793a1-problem-statement) it MUST hand the moment, its text and the agent it is for (the top-level agent, or the starting [subagent](../../GLOSSARY.md#subagent)) to Memory Delivery, return the event with the [block](#793d1b-terms) added by the event's channel (`context` for a prompt, `additionalContext` for a route result or a subagent start) and report the delivered count to the Activity Report. When no [note](./793a_product-spec.md#793a1-problem-statement) is delivered it MUST return the event as received and report nothing. When any step of the lookup fails it MUST return the event as received and report the step and class.

Owner: [Memory Mod](#793d22-memory-mod)

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1g A subagent receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1h A prompt from any source receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1i Please's orientation receives relevant notes](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1j Tier evidence is delivered before a dispatch is routed](./793a_product-spec.md#793a2n1-uat-scenarios), [793a4b1a Vault unavailable](./793a_product-spec.md#793a4b1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d23b1, Test793d24b1; `internal/auto/moment_test.go`: Test793d23b1, Test793d24b1; `wiring.test.ts` "the result of a tool carries the block of the route lookup…"

##### 793d32b1a Scenarios

```gherkin
Scenario: 793d32b1a1 A prompt delivers a note through the context
  Given a note matches the prompt
  When the prompt event is routed
  Then the returned event holds the block in context and the Activity Report receives 1 item delivered
  # acceptance rows: 793a2n1h, 793a2n1i, 793a4c1b

Scenario Outline: 793d32b1a2 A route result or a subagent start delivers through additionalContext
  Given a note matches the <moment>
  When the <event> is routed
  Then the returned result holds the block in additionalContext for <agent>
  # acceptance rows: 793a2n1j, 793a2n1g, 793a2e1d
  Examples:
    | moment | event | agent |
    | route moment | route skill result | the top-level agent |
    | dispatch prompt | subagent start | that subagent |

Scenario: 793d32b1a3 Nothing cued, or nothing selected, delivers nothing
  Given a tool result that cues nothing, or a lookup that selects nothing
  When the event is routed
  Then no block is added and the Activity Report receives no count
  # acceptance rows: 793a2n1e

Scenario: 793d32b1a4 A failed lookup returns the event as received
  Given the query fails
  When a prompt is routed
  Then the prompt is returned as received and the Activity Report receives the step and class
  # acceptance rows: 793a4b1a
```

#### 793d32b2 Step cue sequence

Requirement: 793d32b2 After a top-level step with text and a tool call, Moment Detection MUST redact the recent texts before asking the Judgement Relay for one classification, ask Memory Delivery for a lookup only when the classification names a [moment](./793a_product-spec.md#793a1-problem-statement), and make the [notes](./793a_product-spec.md#793a1-problem-statement) available only to a tool result of the same turn. A classification that fails MUST deliver nothing and let the turn go on.

Owner: [Moment Detection](#793d23-moment-detection)

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d23b2; `internal/auto/moment_test.go`: Test793d23b2; `wiring.test.ts` "a step that called a tool starts the job…"; `wiring.test.ts` "no job is started for a step of a subagent…"

##### 793d32b2a Scenarios

```gherkin
Scenario: 793d32b2a1 A step at the point of declaring done delivers at the next tool result
  Given a step whose text says the rename is about to be checked
  When the classification names declaring work done
  Then the next tool result of the turn carries the note
  # acceptance rows: 793a2n1a

Scenario: 793d32b2a2 A failed classification delivers nothing
  Given the classification is unanswered
  When the step ends
  Then nothing is delivered and the turn continues
  # acceptance rows: 793a4b1b
```

#### 793d32b3 Lookup sequence

Requirement: 793d32b3 Memory Delivery MUST query before it judges, drop the [notes](./793a_product-spec.md#793a1-problem-statement) the receiving agent holds before it judges, judge only when a considered note remains, and return to the Memory Mod the [block](#793d1b-terms) of the selected notes, or nothing; a failed query or [judgement](#793d1b-terms) MUST be returned to the caller as a failure and MUST deliver nothing.

Owner: [Memory Delivery](#793d24-memory-delivery)

Supports: [793a2n1a Recall happens without the agent running it](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2n1e No result, nothing delivered](./793a_product-spec.md#793a2n1-uat-scenarios), [793a2e1a Note delivered earlier is not delivered again](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not delivered](./793a_product-spec.md#793a2e1-uat-scenarios)

Verified by: `internal/auto/golden_test.go`: Test793d24b1, Test793d24b2; `internal/auto/moment_test.go`: Test793d24b1, Test793d24b2

##### 793d32b3a Scenarios

```gherkin
Scenario: 793d32b3a1 A held note is dropped before the judgement
  Given every considered note is already in the agent's context
  When the lookup runs
  Then no judgement runs and nothing is returned
  # acceptance rows: 793a2e1a, 793a2e1c

Scenario: 793d32b3a2 A considered note is judged and returned
  Given one considered note the agent does not hold
  When the judgement selects it
  Then its block is returned
  # acceptance rows: 793a2n1a
```

## 793d33 Turn Memory Work Flow

Canonical target: `docs/specs/technical/turn-memory-work-flow.md` (title `Turn Memory Work Flow - Technical Spec`).

A completed top-level turn, or a stopped [subagent](../../GLOSSARY.md#subagent), is followed by the [System](./793a_product-spec.md#793a1-problem-statement)'s background memory work: capture, the index, curation, one [backlog source](#793d1b-terms) and the user's summary line. The event is answered before the work starts. Each product scenario ID below is a `targ acceptance` row.

### 793d33a Sequence

```mermaid
sequenceDiagram
    participant Eng as Claude Code
    participant Mod as Memory Mod
    participant Cap as Auto Capture
    participant Red as Secret Redaction
    participant JR as Judgement Relay
    participant QE as Query Engine
    participant NA as Note Authoring
    participant NM as Note Amendment
    participant Ix as Chunk Index
    participant CLS as Command Line Surface
    participant Cur as Auto Curation
    participant BL as Backlog Capture
    participant Act as Activity Report

    Eng->>Mod: 793d22b2, 793d33b1 turn complete or subagent stop
    Mod-->>Eng: 793d22b2, 793d33b1 the event answered before any work
    Mod->>Cap: 793d27b1, 793d33b1 capture job with the turn or subagent window
    Cap->>Red: 793d26b1, 793d33b2 redact the window
    Cap->>QE: 792b24c5, 793d33b2 nearest notes of the window
    QE-->>Cap: 792b24c5 items
    Cap->>JR: 793d25b1, 793d25b2, 793d33b2 capture request
    JR-->>Cap: 793d25b2, 793d25b3 the lessons
    alt a lesson is covered by a nearest note
        Cap->>NM: 792b14c9, 793d27b2, 793d33b2 amend with activate
    else a lesson is absent
        Cap->>NA: 792b12c3, 793d27b2, 793d33b2 learn with the source line
        Cap->>Act: 793d30b1 note added
    end
    Mod->>Ix: 792b17c3, 793d33b1 ingest the new content
    Mod->>Cur: 793d29b1, 793d33b1 pass over the pending offers
    Cur->>CLS: 792b2c7, 793d33b3 show the offer
    Cur->>QE: 792b24c5, 793d33b3 nearest notes of the offer
    Cur->>Red: 793d26b1, 793d33b3 redact the offer and the notes
    Cur->>JR: 793d25b1, 793d25b2, 793d33b3 curation request
    JR-->>Cur: 793d25b2, 793d25b3 the outcome
    Cur->>NM: 792b14c14, 793d29b2, 793d33b3 settle the offer
    Cur->>Act: 793d30b1 offers curated and notes amended
    opt the burst holds a top-level turn
        Mod->>BL: 793d28b1, 793d33b1 one backlog step
        BL->>BL: 793d28b2, 793d28b3 read the source and its marks
        BL->>Cap: 793d27b2, 793d33b4 capture one window of the source
    end
    Mod->>Act: 793d30b1, 793d33b1 the burst's work has finished
    Act-->>Eng: 793d30b2 the status line
    alt a step fails
        Mod->>Act: 793d30b2, 793d22b3, 793d33b1 warning with step and class
    end
```

### 793d33b Interaction Requirements

#### 793d33b1 Burst sequencing

Requirement: 793d33b1 The Memory Mod MUST answer the event before queueing any work, run each queued capture in order, and after the last job of the [burst](#793d1b-terms) run, in this order, the index, the [offer](../../GLOSSARY.md#offer) pass and, if the burst holds a top-level turn, one backlog step, continuing with the next step when one fails. When the burst's work has finished it MUST have the Activity Report show the line once, for a burst that holds a top-level turn. A failure MUST be reported to the Activity Report with its step and class.

Owner: [Memory Mod](#793d22-memory-mod)

Supports: [793a4j1a The index catches up after the turn](./793a_product-spec.md#793a4j1-uat-scenarios), [793a4j1e Slow housekeeping does not stall the session](./793a_product-spec.md#793a4j1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5h1a The summary shows the counts after a turn with events](./793a_product-spec.md#793a5h1-uat-scenarios), [793a4b1b Evaluation crashes](./793a_product-spec.md#793a4b1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d22b2; `wiring.test.ts` "a turn is queued before it is answered…"; `wiring.test.ts` "the jobs of a burst run one at a time…"

##### 793d33b1a Scenarios

```gherkin
Scenario: 793d33b1a1 The index catches up after the turn
  Given a completed turn
  When its burst ends
  Then engram ingest --auto runs after the capture and before the offer pass
  # acceptance rows: 793a4j1a

Scenario: 793d33b1a2 Slow work does not hold the next prompt
  Given the System's work takes 20 seconds
  When the next prompt is submitted at once
  Then it is processed without waiting for the work
  # acceptance rows: 793a4j1e

Scenario: 793d33b1a3 The summary follows the work
  Given a turn whose work delivers, adds and curates
  When the work has finished
  Then the Activity Report shows one line
  # acceptance rows: 793a5h1a

Scenario: 793d33b1a4 A failing step does not stop the next
  Given the index step fails
  When the burst ends
  Then the offer pass still runs and a warning is reported for the index
  # acceptance rows: 793a4b1b
```

#### 793d33b2 Capture sequence

Requirement: 793d33b2 Auto Capture MUST redact before it asks the Query Engine for the [nearest notes](#793d1b-terms) and the Judgement Relay for the [judgement](#793d1b-terms), and write only after the judgement is read: reinforce through Note Amendment a [lesson](./793a_product-spec.md#793a1-problem-statement) a nearest note covers, write through Note Authoring one that none states, and report each [note](./793a_product-spec.md#793a1-problem-statement) written to the Activity Report. A judgement or write that fails MUST stop the capture of that [window](#793d1b-terms), write nothing further from it, and be reported.

Owner: [Auto Capture](#793d27-auto-capture)

Supports: [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1g A lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1h Two subagents contain the same lesson](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5k1d An intentional learn by the agent is allowed and is not undone](./793a_product-spec.md#793a5k1-uat-scenarios)

Verified by: `internal/auto/capture_test.go`: Test793d27b2; `internal/auto/golden_test.go`: Test793d27b2

##### 793d33b2a Scenarios

```gherkin
Scenario: 793d33b2a1 A correction is captured as one note
  Given a turn or a stopped subagent whose window holds "no, use targ, not go test directly"
  When the capture ends
  Then the vault holds one note stating the rule
  # acceptance rows: 793a3a1a, 793a3a1g

Scenario: 793d33b2a2 A lesson already recorded is not written again
  Given a note already states the lesson
  When a later window holds it
  Then the note is reinforced and no new note is written
  # acceptance rows: 793a3a1j

Scenario: 793d33b2a3 Two subagents with one lesson make one note
  Given two subagents stop together with the same lesson
  When both captures have run in order
  Then the vault holds one note
  # acceptance rows: 793a3a1h

Scenario: 793d33b2a4 A failed judgement writes nothing
  Given the capture judgement is unreadable
  When the window is captured
  Then nothing is written and the failure is reported
  # acceptance rows: 793a4b1b
```

#### 793d33b3 Curation sequence

Requirement: 793d33b3 For each pending [offer](../../GLOSSARY.md#offer) Auto Curation MUST show the offer, fetch its [nearest notes](#793d1b-terms), redact both, judge, and only then run the commands of the outcome through Note Amendment; an offer that cannot be read, judged or settled MUST be reported once and MUST NOT stop the offers after it. It MUST report each settled offer, and each [note](./793a_product-spec.md#793a1-problem-statement) amended by a `near` outcome, to the Activity Report.

Owner: [Auto Curation](#793d29-auto-curation)

Supports: [793a4j1c Offers are resolved by the System](./793a_product-spec.md#793a4j1-uat-scenarios), [793a4j1d The agent is not asked to curate](./793a_product-spec.md#793a4j1-uat-scenarios)

Verified by: `internal/auto/curate_test.go`: Test793d29b2; `internal/auto/golden_test.go`: Test793d29b2

##### 793d33b3a Scenarios

```gherkin
Scenario: 793d33b3a1 An offer a note covers is folded and one no note covers is accepted
  Given a pending offer O1 that a note covers and a pending offer O2 that none covers
  When the pass runs
  Then O1 is folded into the note and O2 is accepted, neither pending
  And the agent is asked nothing
  # acceptance rows: 793a4j1c, 793a4j1d

Scenario: 793d33b3a2 A failed offer does not stop the rest
  Given the first of two offers cannot be read
  When the pass runs
  Then the second is settled and one warning is reported
  # acceptance rows: 793a4b1b
```

#### 793d33b4 Backlog sequence

Requirement: 793d33b4 After the index has run, Backlog Capture MUST pick an eligible source from the manifest, read its conversation from the file, hand each [window](#793d1b-terms) to Auto Capture with a source naming the file, and mark the source only when every window has been captured without a failure, so that it is tried again at a later turn after a failure.

Owner: [Backlog Capture](#793d28-backlog-capture)

Supports: [793a3a1k A lesson in a session that ended before it was saved is captured later](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1j A lesson that is already recorded is not written again](./793a_product-spec.md#793a3a1-uat-scenarios), [793a5c1c A captured Pi session's transcript is left unchanged](./793a_product-spec.md#793a5c1-uat-scenarios)

Verified by: `internal/auto/backlog_test.go`: Test793d28b1, Test793d28b3; `internal/auto/golden_test.go`: Test793d28b1, Test793d28b3

##### 793d33b4a Scenarios

```gherkin
Scenario: 793d33b4a1 An ended session is captured at a later turn
  Given an ended Pi session whose lesson was not saved
  When a later turn's burst reaches its backlog step
  Then the vault holds a note stating the lesson and the Pi transcript is unchanged
  # acceptance rows: 793a3a1k, 793a5c1c

Scenario: 793d33b4a2 A failure leaves the source for the next turn
  Given a window's capture fails
  When the turn ends
  Then the source is unmarked
  # acceptance rows: 793a3a1k
```

## 793d34 Mod Install Flow

Canonical target: `docs/specs/technical/mod-install-flow.md` (title `Mod Install Flow - Technical Spec`).

`engram update` deploys the engram-memory mod for Claude Code beside the [skills](../../GLOSSARY.md#skill), checks that the installed Claude Code can load it, and reports both. Each product scenario ID below is a `targ acceptance` row.

### 793d34a Sequence

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant SI as Self Installer
    participant HDS as Harness Deploy Sync
    participant CC as Claude Code program

    Dev->>SI: 793d34b1 engram update, with or without the dry-run flag
    SI->>HDS: 793d31b1, 793d31b2, 793d34b1 deploy the mods with the skills
    alt the source holds a mod
        HDS->>CC: 793d31b3, 793d34b1 claude --version
        CC-->>HDS: 793d31b3 the version, or no answer
        alt Claude Code 2.1.287 or later
            HDS-->>SI: 793d31b3 no notice
        else older, unreadable or not on the path
            HDS-->>SI: 793d31b3 a notice and the deploy goes on
        end
    end
    HDS-->>SI: 793d31b1, 793d31b2, 793d31b3 harness sections with mod rows
    SI-->>Dev: 793d34b1 the report
```

### 793d34b Interaction Requirements

#### 793d34b1 Update with the mod

Requirement: 793d34b1 On `engram update` the Self Installer MUST run the harness deploy, with the dry-run option passed on unchanged so that a dry run installs and writes no [mod](#793d1b-terms), and MUST return the deploy's [harness](../../GLOSSARY.md#harness) sections to the developer as received, a mod notice included and never turned into a failure. A second update MUST leave the install tree as the first left it, and a harness the update does not deploy to MUST get no mod.

Owner: [Self Installer](../../specs/technical/self-installer.md)

Supports: [793a5j1a Update installs the System for the harness it updates](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1b An existing install gains the System and keeps what it had](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1c A preview installs none of the System](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1d One update serves both harnesses differently](./793a_product-spec.md#793a5j1-uat-scenarios), [793a5j1l Running update twice changes nothing](./793a_product-spec.md#793a5j1-uat-scenarios)

Verified by: `internal/update/mods_test.go`: Test793e8e1e, 1f, 1g, 1h, 2a

##### 793d34b1a Scenarios

```gherkin
Scenario: 793d34b1a1 Update deploys the mod for Claude Code only
  Given Claude Code is installed and Pi is not
  When the user runs engram update
  Then the mod is linked in the Claude Code skills directory and the install tree has no Pi configuration
  # acceptance rows: 793a5j1a, 793a5j1b, 793a5j1d (Not run: the Pi agent's behavior)

Scenario: 793d34b1a2 A preview installs none of it
  Given the mod is not installed
  When the user runs engram update --dry-run
  Then nothing is written and the report lists the mod
  # acceptance rows: 793a5j1c

Scenario: 793d34b1a3 Running update twice changes nothing
  Given a completed update
  When the user runs engram update again
  Then the install tree is unchanged
  # acceptance rows: 793a5j1l

Scenario: 793d34b1a4 An older Claude Code is reported, not fatal
  Given claude --version prints a version before 2.1.287
  When the user runs engram update
  Then the report carries a mods notice and the deploy completes
  # acceptance rows: none (Go test only)
```

## 793d35 Traceability and gaps

### 793d35a Check

All 43 scenarios of the product spec are supported by at least one requirement below, and all 32 requirements (24 behavior requirements of 10 components, 8 interaction requirements of 3 flows) name at least one contract test. Scenario-level support is on each requirement's `Supports` line; the table gives the union per product requirement.

### 793d35b Product requirements to technical requirements

| Product requirement | Scenarios supported | Supporting requirements |
| --- | --- | --- |
| [793a2n Relevant memory reaches the agent at the moments](./793a_product-spec.md#793a2n-relevant-memory-reaches-the-agent-at-the-moments) | 6 | [793d23b1 Event cues](#793d23b1-event-cues), [793d23b2 In-turn cue](#793d23b2-in-turn-cue), [793d24b1 Lookup and selection](#793d24b1-lookup-and-selection), [793d24b3 Block](#793d24b3-block), [793d25b1 Tier and request shape](#793d25b1-tier-and-request-shape), [793d25b3 Result shapes](#793d25b3-result-shapes), [793d32b1 Moment routing](#793d32b1-moment-routing), [793d32b2 Step cue sequence](#793d32b2-step-cue-sequence), [793d32b3 Lookup sequence](#793d32b3-lookup-sequence) |
| [793a2e Nothing already in context is delivered again](./793a_product-spec.md#793a2e-nothing-already-in-context-is-delivered-again) | 5 | [793d23b2 In-turn cue](#793d23b2-in-turn-cue), [793d24b2 Presence and ledgers](#793d24b2-presence-and-ledgers), [793d32b3 Lookup sequence](#793d32b3-lookup-sequence) |
| [793a3a Lessons are captured without the agent](./793a_product-spec.md#793a3a-lessons-are-captured-without-the-agent) | 5 | [793d22b2 Background queue and housekeeping](#793d22b2-background-queue-and-housekeeping), [793d25b1 Tier and request shape](#793d25b1-tier-and-request-shape), [793d25b2 Data and replies](#793d25b2-data-and-replies), [793d25b3 Result shapes](#793d25b3-result-shapes), [793d26b1 Credential shapes](#793d26b1-credential-shapes), [793d27b1 Windows](#793d27b1-windows), [793d27b2 Reconciliation, writes and provenance](#793d27b2-reconciliation-writes-and-provenance), [793d28b1 Eligible sources and cut-off](#793d28b1-eligible-sources-and-cut-off), [793d28b2 Reading and windows](#793d28b2-reading-and-windows), [793d28b3 Marks and read-only sources](#793d28b3-marks-and-read-only-sources), [793d33b1 Burst sequencing](#793d33b1-burst-sequencing), [793d33b2 Capture sequence](#793d33b2-capture-sequence), [793d33b4 Backlog sequence](#793d33b4-backlog-sequence) |
| [793a4j Indexing and curation happen after the turn](./793a_product-spec.md#793a4j-indexing-and-curation-happen-after-the-turn) | 4 | [793d22b2 Background queue and housekeeping](#793d22b2-background-queue-and-housekeeping), [793d25b1 Tier and request shape](#793d25b1-tier-and-request-shape), [793d25b2 Data and replies](#793d25b2-data-and-replies), [793d25b3 Result shapes](#793d25b3-result-shapes), [793d26b1 Credential shapes](#793d26b1-credential-shapes), [793d29b1 Offers to settle](#793d29b1-offers-to-settle), [793d29b2 Outcomes and guards](#793d29b2-outcomes-and-guards), [793d33b1 Burst sequencing](#793d33b1-burst-sequencing), [793d33b3 Curation sequence](#793d33b3-curation-sequence) |
| [793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only) | 4 | [793d22b1 Events and channels](#793d22b1-events-and-channels), [793d24b3 Block](#793d24b3-block), [793d30b2 Warnings and surface](#793d30b2-warnings-and-surface) |
| [793a4b The session is never broken or stalled](./793a_product-spec.md#793a4b-the-session-is-never-broken-or-stalled) | 4 | [793d22b3 Failure containment](#793d22b3-failure-containment), [793d30b2 Warnings and surface](#793d30b2-warnings-and-surface), [793d32b1 Moment routing](#793d32b1-moment-routing), [793d33b1 Burst sequencing](#793d33b1-burst-sequencing) |
| [793a5j Installed by engram update and active in Claude Code](./793a_product-spec.md#793a5j-installed-by-engram-update-and-active-in-claude-code) | 5 | [793d31b1 Mod deploy set](#793d31b1-mod-deploy-set), [793d31b2 Mods are synchronized like skills](#793d31b2-mods-are-synchronized-like-skills), [793d31b3 Version check and report rows](#793d31b3-version-check-and-report-rows), [793d34b1 Update with the mod](#793d34b1-update-with-the-mod) |
| [793a5h The user sees a summary of the memory activity](./793a_product-spec.md#793a5h-the-user-sees-a-summary-of-the-memory-activity) | 4 | [793d30b1 Counts and summary line](#793d30b1-counts-and-summary-line), [793d30b2 Warnings and surface](#793d30b2-warnings-and-surface), [793d33b1 Burst sequencing](#793d33b1-burst-sequencing) |
| [793a5k Intentional memory is never blocked or changed](./793a_product-spec.md#793a5k-intentional-memory-is-never-blocked-or-changed) | 3 | [793d23b2 In-turn cue](#793d23b2-in-turn-cue), [793d27b2 Reconciliation, writes and provenance](#793d27b2-reconciliation-writes-and-provenance), [793d33b2 Capture sequence](#793d33b2-capture-sequence) |
| [793a5c Other harnesses are unchanged](./793a_product-spec.md#793a5c-other-harnesses-are-unchanged) | 3 | [793d28b1 Eligible sources and cut-off](#793d28b1-eligible-sources-and-cut-off), [793d28b2 Reading and windows](#793d28b2-reading-and-windows), [793d28b3 Marks and read-only sources](#793d28b3-marks-and-read-only-sources), [793d31b1 Mod deploy set](#793d31b1-mod-deploy-set), [793d33b4 Backlog sequence](#793d33b4-backlog-sequence) |

### 793d35c Gaps and what is not built

- `Not run`: [793a5j1d One update serves both harnesses differently](./793a_product-spec.md#793a5j1-uat-scenarios) and [793a5c1b A Pi session on a machine whose Claude Code has the System](./793a_product-spec.md#793a5c1-uat-scenarios) have no headless Pi, so what a Pi agent does (it runs recall before acting, its own ingest and learn run unchanged, nothing is delivered into its [transcript](../../GLOSSARY.md#transcript)) is unverified. What is verified is that no [mod](#793d1b-terms) is deployed for Pi ([793d31b1 Mod deploy set](#793d31b1-mod-deploy-set)) and that Backlog Capture writes only its mark file ([793d28b3 Marks and read-only sources](#793d28b3-marks-and-read-only-sources)).
- `Red` once: the third example of [793a3a1a A lesson is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) in the last full acceptance run (the [judgement](#793d1b-terms) worded the [note](./793a_product-spec.md#793a1-problem-statement) without the corrected cause; 15 of 15 in five reruns). A judgement-quality gap, not a missing requirement.
- Built and untested, so not required: the task-notification summary and non-SDK prompt origins of [793d23b1 Event cues](#793d23b1-event-cues); the age limit of an unfinished step job ([793d23b2 In-turn cue](#793d23b2-in-turn-cue)).
- Built differently from the product intent: a [subagent](../../GLOSSARY.md#subagent)'s compaction empties the top-level [ledger](#793d1b-terms) and not its own ([793d24b2 Presence and ledgers](#793d24b2-presence-and-ledgers)).
- Built, no longer a gap: persistence of queued work ([793b203 Background work is one in-memory queue, and a final turn can be lost on exit](./793b_decision-log.md#793b203-background-work-is-one-in-memory-queue-and-a-final-turn-can-be-lost-on-exit)); the persisted queue is built by [793e8p Hardening](./793e_implementation-plan.md#793e8p-hardening) ([793b398 The persisted queue is a file the engine owns, and the red tests of 793b362 became tests of the jobs the mod hands it](./793b_decision-log.md#793b398-the-persisted-queue-is-a-file-the-engine-owns-and-the-red-tests-of-793b362-became-tests-of-the-jobs-the-mod-hands-it)).
- Duplicated: `internal/auto` reads transcript rows by role itself ([793d28b2 Reading and windows](#793d28b2-reading-and-windows)) and not through the [Transcript Reader](../../specs/technical/transcript-reader.md), so the two can drift.
- Harness-only, so a MAY with no requirement: the seam file named by `ENGRAM_AUTO_SEAM_LOG`, which lets a test wait for the queue (`wiring.test.ts` covers it).
- The experiment scenarios of 793d17 and 793d18 are not contract tests of these requirements, except [793d18c1a Every request names the model of its kind](#793d18c1-experiments) and [793d18c1b Raising one kind changes only that kind](#793d18c1-experiments), which are the tier tests of [793d25b1 Tier and request shape](#793d25b1-tier-and-request-shape).

### 793d35d Security cross-reference

Risks of [793d19a Security](#793d19a-security): (1) injection into a delivered [note](./793a_product-spec.md#793a1-problem-statement): [793d23b2 In-turn cue](#793d23b2-in-turn-cue), [793d25b2 Data and replies](#793d25b2-data-and-replies), [793d25b3 Result shapes](#793d25b3-result-shapes), [793d27b1 Windows](#793d27b1-windows), [793d28b2 Reading and windows](#793d28b2-reading-and-windows); (2) a crafted [offer](../../GLOSSARY.md#offer): [793d25b2 Data and replies](#793d25b2-data-and-replies), [793d25b3 Result shapes](#793d25b3-result-shapes), [793d29b2 Outcomes and guards](#793d29b2-outcomes-and-guards); (3) secrets: [793d26b1 Credential shapes](#793d26b1-credential-shapes), [793d27b1 Windows](#793d27b1-windows); (4) a wrongly judged `covered`: [793d29b2 Outcomes and guards](#793d29b2-outcomes-and-guards).

Mitigations: provenance lines [793d27b2 Reconciliation, writes and provenance](#793d27b2-reconciliation-writes-and-provenance); aliases on folded [offers](../../GLOSSARY.md#offer) [793d29b2 Outcomes and guards](#793d29b2-outcomes-and-guards); data framing [793d24b3 Block](#793d24b3-block); the activity line [793d30b2 Warnings and surface](#793d30b2-warnings-and-surface).
