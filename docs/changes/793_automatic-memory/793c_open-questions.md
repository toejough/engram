# Automatic Memory - Open Questions

Each question would need Joe's input before [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) is final. Joe was unavailable, so each has a default,
recorded in [793b Automatic Memory - Decision Log](./793b_decision-log.md#automatic-memory---decision-log) as "Decided by: Author (default; owner may override)", and the default stands until
he answers. When he answers, the entry records the answer and the date and stays in place
([Decisions and Open Questions](../../AGENTS.md#decisions-and-open-questions)).

## 793c1 How long may a prompt be held?

- **Context:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) promises that [automatic memory](./793a_product-spec.md#793a1a-terms) never holds a [prompt](./793a_product-spec.md#793a1a-terms) longer than a short
  bound, and a promise with no number cannot be verified. It says 8 seconds.
- **Options:** (a) 8 seconds. (b) A shorter bound, such as 3 seconds. (c) A longer one. (d) The owner's rule from
  the earlier attempt: no longer than the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup on the same
  [vault](../../GLOSSARY.md#vault) and input (`old 793c7` item 4 and `old 793c11` (b), 2026-10-11), measured at
  10.6 to 11.0 seconds on his vault (`old 793c2`).
- **Recommendation:** (a). The earlier attempt's author set it and the owner did not raise it when offered a
  longer hold (`old 793b317`, `old 793c87`, 2026-10-14), and a lookup of [notes](../../GLOSSARY.md#permanent-note)
  alone took about 1.4 seconds there. (d) is his own rule, but it moves with the vault and on his would allow a
  hold of more than 10 seconds.
- **Blocks:** the scenarios of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) that test the hold, which [793e6a On after update, and memory arrives with the prompt](./793e_implementation-plan.md#793e6a-on-after-update-and-memory-arrives-with-the-prompt) works.
- **Status:** open. Default (a) taken: [793b17 A prompt is held for at most 8 seconds](./793b_decision-log.md#793b17-a-prompt-is-held-for-at-most-8-seconds).

## 793c2 How soon after a turn must automatic memory have finished writing?

- **Context:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) say "within 2 minutes" because "shortly" cannot be verified. The earlier
  attempt set no bound for the normal path, and this attempt has not measured a capture yet.
- **Options:** (a) 2 minutes. (b) 1 minute. (c) 5 minutes.
- **Recommendation:** (a), until a measured capture replaces it when [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) is built.
- **Blocks:** the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) and [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later).
- **Status:** open. Default (a) taken: [793b18 Writing finishes within 2 minutes of a turn](./793b_decision-log.md#793b18-writing-finishes-within-2-minutes-of-a-turn).

## 793c3 Is the volume of delivered notes limited?

- **Context:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers the [notes](../../GLOSSARY.md#permanent-note) that bear on a [prompt](./793a_product-spec.md#793a1a-terms) and states no number. Without a limit, a
  [vault](../../GLOSSARY.md#vault) with many related notes could flood the [agent](./793a_product-spec.md#793a1a-terms)'s context.
- **Options:** (a) No number in the product spec; the technical spec limits the number of notes to the number the
  agent's own explicit lookup returns. (b) A maximum stated in [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt).
- **Recommendation:** (a). The owner's answer on 2026-10-11 in the earlier attempt (`old 793c7` items 1 and 2) was
  that volume equals what the agent's own lookup returns, with no per-[turn](./793a_product-spec.md#793a1a-terms) or per-[session](../../GLOSSARY.md#session) bounds.
- **Blocks:** scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that involve many related notes.
- **Status:** open. Default (a) taken: [793b19 Volume of delivered notes](./793b_decision-log.md#793b19-volume-of-delivered-notes).

## 793c5 Are lessons written straight into the vault or held for review?

- **Context:** issue #793's open decision 4. [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) says the [note](../../GLOSSARY.md#permanent-note) must be in the [vault](../../GLOSSARY.md#vault) within 2
  minutes.
- **Options:** (a) Straight in, marked automatic. (b) As [pending offers](../../GLOSSARY.md#offer) reviewed with the curate [skill](../../GLOSSARY.md#skill).
- **Recommendation:** (a), the owner's answer on 2026-10-11 in the earlier attempt (`old 793c4`).
- **Blocks:** what "in the vault" means in the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Status:** open. Default (a) taken: [793b21 Lessons are written straight into the vault](./793b_decision-log.md#793b21-lessons-are-written-straight-into-the-vault).

## 793c6 Is the scope top-level sessions and notes only?

- **Context:** [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions) leaves out [subagents](../../GLOSSARY.md#subagent) and the chunk-index lookup. In the earlier attempt the owner
  answered the opposite on 2026-10-11 for subagents and [prompt](./793a_product-spec.md#793a1a-terms) sources: capture inside subagent [sessions](../../GLOSSARY.md#session), and
  memory for every prompt regardless of source (`old 793c6` and `old 793c63` for subagents, `old 793c5` for
  prompt sources, 2026-10-11). The full list of reversed answers is asked in [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse). A lookup that included the chunk index took about 12 seconds
  on his [vault](../../GLOSSARY.md#vault) there, longer than the 8 seconds of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Options:** (a) The user's own sessions and [notes](../../GLOSSARY.md#permanent-note) only. (b) Add subagents. (c) Add the chunk-index
  lookup.
- **Recommendation:** (a) for the minimal set. This reverses his earlier answers, so it needs his confirmation.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), and [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions).
- **Status:** open. Default (a) taken: [793b22 The scope is top-level sessions and notes only](./793b_decision-log.md#793b22-the-scope-is-top-level-sessions-and-notes-only).

## 793c8 What about secrets in written notes?

- **Context:** automatic [notes](../../GLOSSARY.md#permanent-note) quote what the user typed, including anything typed in a correction, and the
  [vault](../../GLOSSARY.md#vault) may be under version control. The earlier attempt's final product spec no
  longer carried the MUST NOT on it that the owner had set (`old 793c7` item 10, 2026-10-11).
- **Options:** (a) No rule in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec). (b) Add "a note MUST NOT contain a secret" to [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Recommendation:** (b).
- **Blocks:** the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Status:** answered (Joe, 2026-10-11, in the earlier attempt: `old 793c7` item 10; the redaction approach and
  its residual risk ratified on 2026-10-13, `old 793b324`).
- **Answer:** (b): a note MUST NOT contain a credential, token, key or password that appeared in the [turn](./793a_product-spec.md#793a1a-terms).
- **Applied in:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), and [793b24 A note holds no secret](./793b_decision-log.md#793b24-a-note-holds-no-secret).

## 793c9 Does a turn with nothing to report show anything?

- **Context:** [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows something only on a [turn](./793a_product-spec.md#793a1a-terms) on which something was delivered, written or could not be done.
- **Options:** (a) Nothing is shown. (b) Something is shown on every turn, so the user can tell [automatic memory](./793a_product-spec.md#793a1a-terms) is on.
- **Recommendation:** (a), as in the earlier attempt, which omitted zero counts.
- **Blocks:** the scenarios of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did), and any scenario that needs a turn with nothing to report.
- **Status:** open. Default (a) taken: [793b25 What is shown after a turn, and when](./793b_decision-log.md#793b25-what-is-shown-after-a-turn-and-when).

## 793c10 Does "already has" survive a compaction?

- **Context:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does not deliver a [note](../../GLOSSARY.md#permanent-note) the [agent](./793a_product-spec.md#793a1a-terms) already has in the [session](../../GLOSSARY.md#session). When the
  session's context is compacted, the agent may no longer hold the note's text. The owner's earlier answer was the
  other way: never re-inject a note already in the receiving agent's context, and a truncated note is not present
  (`old 793c7` item 5, 2026-10-11), so default (a) reverses it (row 11 of [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse)).
- **Options:** (a) Per session: a note delivered once is not delivered again, whatever the agent still holds.
  (b) Deliver it again after a compaction.
- **Recommendation:** (a) for the minimal set: (b) needs knowledge of what the agent still holds.
- **Blocks:** the scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that test what the agent already has.
- **Status:** open. Default (a) taken: [793b27 "Already has" is per session](./793b_decision-log.md#793b27-already-has-is-per-session).

## 793c12 Is memory delivered at mid-task moments?

- **Context:** issue #793 asks for automatic recall "at human [prompts](./793a_product-spec.md#793a1a-terms), tool results and
  [subagent](../../GLOSSARY.md#subagent) dispatch".
  [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers at the [prompt](./793a_product-spec.md#793a1a-terms) only. The main [agent](./793a_product-spec.md#793a1a-terms)'s recall also fired mid-task, by its own judgement:
  before it endorsed or ranked an approach, before it declared work done, after a failure it could not explain,
  and before it started building a new approach
  ([792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments)).
  In the earlier attempt the owner approved delivery at those moments (`old 793b187`, 2026-10-11, and
  `old 793b240`, 2026-10-12). The minimal set leaves them out ([793b8 Exclusions](./793b_decision-log.md#793b8-exclusions)).
- **Options:** (a) The prompt only. (b) The prompt and tool results. (c) The prompt, tool results and the four
  decision moments.
- **Recommendation:** (a) for the minimal set. The main agent's mid-task re-queries were moments it chose by
  judgement, and finding them from outside needs a model judgement or a look at every tool result, which the
  set leaves out. (b) or (c) can be a later change.
- **Blocks:** nothing in the minimal set; the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) if (b) or (c) is chosen.
- **Status:** open. Default (a) taken: [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions).

## 793c13 Which earlier owner answers may the minimal set reverse?

- **Context:** the minimal set leaves out, or does differently, things the owner answered on in the earlier
  attempt. The table lists every one with its earlier entry and date; the default for each row is the
  author's ([793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)). Rows 1, 2 and 5 are also the question of [793c6 Is the scope top-level sessions and notes only?](./793c_open-questions.md#793c6-is-the-scope-top-level-sessions-and-notes-only), row 4 of [793c12 Is memory delivered at mid-task moments?](./793c_open-questions.md#793c12-is-memory-delivered-at-mid-task-moments), row 11 of [793c10 Does "already has" survive a compaction?](./793c_open-questions.md#793c10-does-already-has-survive-a-compaction) and row 12
  of [793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held); answering here answers those too.

| # | What | Earlier owner answer | This change |
| --- | --- | --- | --- |
| 1 | [Subagents](../../GLOSSARY.md#subagent) | capture inside subagent [sessions](../../GLOSSARY.md#session): yes (`old 793c6`, 2026-10-11); delivery at a [subagent](../../GLOSSARY.md#subagent)'s start and capture at its stop (`old 793c63` (a), 2026-10-11) | the user's own sessions only |
| 2 | [Prompt](./793a_product-spec.md#793a1a-terms) sources | memory for every prompt regardless of source (`old 793c5`, 2026-10-11) | prompts the user sends; until [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) the mod delivers for every origin ([793c35 Does the mod tell a prompt the user sent from a prompt of another origin?](#793c35-does-the-mod-tell-a-prompt-the-user-sent-from-a-prompt-of-another-origin)) |
| 3 | Route results | tier evidence delivered before the orchestrator routes a unit (`old 793c39` (b) and `old 793c64` (a), 2026-10-11) | none |
| 4 | Mid-task moments | delivery at the moments where recall should fire (`old 793b187`, 2026-10-11, and `old 793b240`, 2026-10-12; `old 793c49` (b) and `old 793c55` (a), 2026-10-11) | the prompt only |
| 5 | Chunk stubs | conversation fragments delivered beside the [notes](../../GLOSSARY.md#permanent-note) as stubs (`old 793c15` (b), 2026-10-11), kept by a background pass when the full lookup proved too slow (`old 793c87` (a), 2026-10-14) | none delivered; the [agent](./793a_product-spec.md#793a1a-terms) finds fragments when it asks ([793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)) |
| 6 | Relevance filter and moment classification | two model judgements before delivery (`old 793c69` (a), 2026-10-12), kept on haiku at the lowest effort (`old 793c79` (a), 2026-10-14) | none; the agent judges what it is given |
| 7 | Curation after the [turn](./793a_product-spec.md#793a1a-terms) | indexing and curation after each turn (`old 793b240`, 2026-10-12); curation run by the System (`old 793c43` (a), 2026-10-11) | indexing is back as [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory); curation of [pending offers](../../GLOSSARY.md#offer) is out |
| 8 | Backlog capture | leftover sessions of any [harness](../../GLOSSARY.md#harness) picked up by the next session (`old 793c67`, 2026-10-12; `old 793c65` (a), 2026-10-11; cut-off `old 793c75`, 2026-10-13) | none; a turn followed by the end of the session is [793c14 Is a turn that is followed by the end of the session covered?](./793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered) |
| 9 | Reinforcing a note | reinforcement is a MUST (`old 793c14` (b), 2026-10-11); the intent of 2026-10-12 (`old 793b240`) no longer names it | a lesson the [vault](../../GLOSSARY.md#vault) already states is not written again, and nothing more |
| 10 | Stopping it | turned off by uninstalling, with uninstall extended to remove it (`old 793c18` (a) and `old 793c22` (b), 2026-10-11) | no requirement covers how ([793b20 No modes and no off switch](./793b_decision-log.md#793b20-no-modes-and-no-off-switch)) |
| 11 | "Already has" | a note already in the receiving agent's context is never re-injected; a truncated note is not present (`old 793c7` item 5, 2026-10-11) | per session, even after a compaction |
| 12 | The prompt hold | no longer than the agent's own lookup on the same vault and input (`old 793c7` item 4 and `old 793c11` (b), 2026-10-11) | 8 seconds |
| 13 | Sessions covered | the promise covers the sessions in which the earlier attempt's mod runs, and [`engram update`](../../GLOSSARY.md#engram-update) reports when mods cannot load (`old 793c66` (a), 2026-10-11) | every Claude Code session the user starts after the update ([793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)); where it cannot run, [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) has it step aside, and no check is required |
| 14 | Activity for the agent | the activity lines may reach the agent (`old 793c90` (b), 2026-10-15), and the previous turn's sentences ride the next prompt (`old 793c92` (b), 2026-10-16) | the user sees the activity in the session ([793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did)); nothing carries it to the agent |

- **Options:** (a) Accept every reversal for the minimal set; any row can come back as a separate change.
  (b) Keep named rows in the minimal set; the owner names them by number.
- **Recommendation:** (a). The owner asked for the minimal set, each row needs a measurement, a second model
  judgement or machinery the set leaves out, and a separate change can restore any of them.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt), [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) for the rows kept.
- **Status:** open. Defaults taken: [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses).

## 793c14 Is a turn that is followed by the end of the session covered?

- **Context:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) promises a [note](../../GLOSSARY.md#permanent-note) within 2 minutes after a [turn](./793a_product-spec.md#793a1a-terms) ends, and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) makes the conversation
  findable within the same 2 minutes. A user who ends the [session](../../GLOSSARY.md#session) right after a turn leaves [automatic memory](./793a_product-spec.md#793a1a-terms) no time to
  finish. The earlier attempt recorded such a final turn as lost (`old 793b203`), then kept unfinished work for
  the next session (`old 793c65` (a), 2026-10-11), then picked up leftover sessions of any [harness](../../GLOSSARY.md#harness)
  (`old 793c67`, 2026-10-12). The minimal set has none of those (row 8 of [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse)).
- **Options:** (a) Not covered: the promise holds while the session stays open for the 2 minutes. (b) Covered:
  the work finishes after the session ends, or the next session finishes it.
- **Recommendation:** (a) for the minimal set: (b) needs work that outlives the session, which the set leaves
  out. The cost is that a lesson in the last turn before an exit can be lost.
- **Blocks:** the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) for a session that ends within 2 minutes after a turn.
- **Status:** open. Default (a) taken: [793b37 A turn followed by the end of the session is not covered](./793b_decision-log.md#793b37-a-turn-followed-by-the-end-of-the-session-is-not-covered).

## 793c15 Should the activity be shown in the agent's reply, so that it reaches the claude.ai mobile and web view?

- **Context:** the owner follows [sessions](../../GLOSSARY.md#session) from the claude.ai mobile app and sees none of what [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows there;
  the only route the earlier attempt found was the [agent](./793a_product-spec.md#793a1a-terms)'s own reply (`old 793c93`).
- **Options:** (a) Leave it in the session only. (b) Ask the agent to end its reply
  with one fixed activity line.
- **Recommendation:** (a).
- **Blocks:** nothing.
- **Status:** parked (Joe, 2026-10-16, in the earlier attempt: `old 793c93`): "Nah, leave it for now." Revisit
  after the behavior of [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) has landed.

## 793c16 Are the ten scenarios of the first group the minimum?

- **Context:** step 2 for the first group wrote ten scenarios, one per mandatory branch of [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update), [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and the hold of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Options:** (a) The ten as written. (b) Fewer: the dry run and Pi folded into the first scenario of the update.
- **Recommendation:** (a): the outcomes differ and each needs its own report.
- **Blocks:** nothing; the scenarios are written and worked.
- **Status:** open. Default (a) taken: [793b40 The scenarios of the first group](./793b_decision-log.md#793b40-the-scenarios-of-the-first-group).
## 793c17 Is the automatic behavior inside a session one mod deployed with the skills?

- **Context:** automatic behavior inside a [session](../../GLOSSARY.md#session) needs a plugin of function hooks, a mod, that [`engram update`](../../GLOSSARY.md#engram-update) can deploy without editing a file the user owns.
- **Options:** (a) One mod, `engram-memory`, kept in `agent-instructions/mods/`, deployed to the [skills](../../GLOSSARY.md#skill) directory with the skills, running a new command for each [prompt](./793a_product-spec.md#793a1a-terms). (b) A `UserPromptSubmit` command hook in the user's `settings.json`.
- **Recommendation:** (a): (b) makes `engram update` edit a file the user owns on every run.
- **Blocks:** [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update).
- **Status:** open. Default (a) taken: [793b41 The mod, its place and its command](./793b_decision-log.md#793b41-the-mod-its-place-and-its-command).
## 793c18 What does a delivered block say?

- **Context:** the block puts [notes](../../GLOSSARY.md#permanent-note) in the [agent](./793a_product-spec.md#793a1a-terms)'s context, and the earlier attempt's guard keeps a note from acting as an instruction.
- **Options:** (a) A framing line that says the lines below are notes delivered by the memory system, and one line for each note: its file name, its situation and its body. (b) Whole notes. (c) A count and no text.
- **Recommendation:** (a): (b) grows with the [vault](../../GLOSSARY.md#vault), and (c) makes the agent look each note up.
- **Blocks:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt).
- **Status:** open. Default (a) taken: [793b42 What the block says](./793b_decision-log.md#793b42-what-the-block-says).
## 793c19 Is the body of a note delivered whole, and on one line?

- **Context:** the body of a delivered [note](../../GLOSSARY.md#permanent-note) is not cut and is written on one line, so a note of several steps, a runbook, loses the line structure that its steps had. A non-blocking question from the review of the first group.
- **Options:** (a) Whole, and on one line. (b) Cut to a length, as the earlier attempt cut at 400 characters. (c) Whole, with its lines kept, which needs a way to keep a line of a body from imitating the framing line or another note ([793b43 A note cannot reshape the block](./793b_decision-log.md#793b43-a-note-cannot-reshape-the-block)).
- **Recommendation:** (a) for the minimal set; (c) if the loss of structure matters to the owner.
- **Blocks:** nothing in the minimal set.
- **Status:** open. Default (a) taken: [793b44 Only the notes that match are delivered, whole](./793b_decision-log.md#793b44-only-the-notes-that-match-are-delivered-whole).
## 793c20 What does a session hold, so that a note is not delivered twice?

- **Context:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does not deliver a [note](../../GLOSSARY.md#permanent-note) the [agent](./793a_product-spec.md#793a1a-terms) already has in the [session](../../GLOSSARY.md#session); something has to say what it has.
- **Options:** (a) The file names on the note lines of the blocks delivered, and the command and the result of each `engram query` and `engram show` the agent ran, read from the session's record. (b) Every tool result, the first version. (c) A ledger kept by the mod.
- **Recommendation:** (a): (b) counted an `ls` listing and missed an `engram show`, and (c) is lost with the process.
- **Blocks:** the scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that test what the agent already has.
- **Status:** open. Default (a) taken: [793b45 What a session holds](./793b_decision-log.md#793b45-what-a-session-holds).
## 793c21 How long may the mod's command run, short of the hold?

- **Context:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) holds a [prompt](./793a_product-spec.md#793a1a-terms) for 8 seconds at most, and the mod needs time to start the command and run the prompt's hooks.
- **Options:** (a) 7 seconds for the command, one second left. (b) 8 seconds for the command. (c) A limit inside the command.
- **Recommendation:** (a): the platform's own limit does not count the call, and a command that hangs cannot end itself.
- **Blocks:** the scenarios of the hold.
- **Status:** open. Default (a) taken: [793b46 The command is ended at 7 seconds](./793b_decision-log.md#793b46-the-command-is-ended-at-7-seconds).
## 793c22 Does a dry run report the mod as a mod?

- **Context:** a preview exists to tell the user what an update would [turn](./793a_product-spec.md#793a1a-terms) on, and the report listed the mod under the source of a [skill](../../GLOSSARY.md#skill).
- **Options:** (a) The mod is listed in the section of Claude Code with `agent-instructions/mods/<name>/` as its source. (b) As before, under the skills. (c) A section of the report for mods.
- **Recommendation:** (a): (b) names a directory the mod does not come from, and (c) changes the report for every reader.
- **Blocks:** [793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios).
- **Status:** open. Default (a) taken: [793b47 A dry run reports the mod as a mod](./793b_decision-log.md#793b47-a-dry-run-reports-the-mod-as-a-mod).
## 793c23 Does the conformance check read this change's specs, and fail when they are gone?

- **Context:** the owner's rule asks a check that every scenario ID of the specs has a test, and the specs are deleted at promotion.
- **Options:** (a) A test of the dev integration suite, which fails, naming the cause, when the specs are not there. (b) A target of its own. (c) A skip when the specs are missing.
- **Recommendation:** (a): (b) could be forgotten and (c) passes with nothing checked.
- **Blocks:** nothing; the check is written.
- **Status:** open. Default (a) taken: [793b48 The conformance check reads this change's specs and fails when they are gone](./793b_decision-log.md#793b48-the-conformance-check-reads-this-changes-specs-and-fails-when-they-are-gone).
## 793c24 How many acceptance tests run at once?

- **Context:** the tests start real processes, and two kinds are bounded by time; ten at once made two of them fail in each of two runs.
- **Options:** (a) At most four at once. (b) The number of cores. (c) One at a time.
- **Recommendation:** (a): it left a margin below six, where nothing failed, and costs about 40 seconds for the full set.
- **Blocks:** nothing; the cap is set.
- **Status:** open. Default (a) taken: [793b49 The acceptance run is capped at four tests at once](./793b_decision-log.md#793b49-the-acceptance-run-is-capped-at-four-tests-at-once).

## 793c25 Does the mod use what a failed or unrecognised command printed?

- **Context:** [793d2a Prompt delivery flow](./793d_technical-spec.md#793d2a-prompt-delivery-flow) has the mod put the
  output of `engram auto deliver` beside the [prompt](./793a_product-spec.md#793a1a-terms). A command that fails, such as an older `engram` that does not know
  `auto` (it prints its usage and exits 1), prints text that is not a block.
- **Options:** (a) The output is used only when the command exited with 0 and the output begins with `[engram memory]`.
  (b) Any non-empty output is used, as the first version did.
- **Recommendation:** (a): (b) puts an error message or a usage text in the [agent](./793a_product-spec.md#793a1a-terms)'s context as if it were a [note](../../GLOSSARY.md#permanent-note).
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b50 The mod uses what the command printed only when it succeeded and printed a block](./793b_decision-log.md#793b50-the-mod-uses-what-the-command-printed-only-when-it-succeeded-and-printed-a-block).

## 793c26 Does the delivery read the records where Claude Code writes them, and does the sweep follow?

- **Context:** Claude Code writes the record of a [session](../../GLOSSARY.md#session) under `$CLAUDE_CONFIG_DIR/projects` when `CLAUDE_CONFIG_DIR`
  is set. [`engram update`](../../GLOSSARY.md#engram-update) installs the mod under `<home>/.claude`, so the delivery matters for a user whose configuration
  directory links to it. [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) reads `ENGRAM_TRANSCRIPT_DIR`, else `<home>/.claude/projects`
  ([792b17c6 Session-log directory](../../specs/technical/ingestion-source-discovery.md#792b17c6-session-log-directory)),
  which ignores `CLAUDE_CONFIG_DIR`.
- **Options:** (a) The delivery reads `ENGRAM_TRANSCRIPT_DIR`, else `$CLAUDE_CONFIG_DIR/projects`, else
  `<home>/.claude/projects`; the sweep is unchanged. (b) Both use that order, which changes a canonical requirement of
  `engram ingest`. (c) Neither: the delivery reads `<home>/.claude/projects` only, as the first version did.
- **Recommendation:** (a) for this change. (b) is a separate change to the sweep.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b51 The delivery finds the records where Claude Code writes them](./793b_decision-log.md#793b51-the-delivery-finds-the-records-where-claude-code-writes-them).

## 793c27 Does the delivery's lookup need imptest evidence of its arguments, or a waiver?

- **Context:** [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates) asks `imptest` evidence of
  the call and arguments of every injected collaboration. The delivery's lookup was the query dependencies, called
  directly, so its arguments showed only through the real engine.
- **Options:** (a) The delivery takes the lookup as a function, and `imptest` shows the arguments. (b) A waiver of the
  gate for the lookup, with the arguments tested through the real engine.
- **Recommendation:** (a): a delivery that left the message out passed every test.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b52 The delivery runs its lookup through an injected function](./793b_decision-log.md#793b52-the-delivery-runs-its-lookup-through-an-injected-function).

## 793c28 Does an acceptance test skip or fail when Claude Code or a login is missing?

- **Context:** the tests that drive a real [session](../../GLOSSARY.md#session) need `claude` and a login. With neither, they skipped, and
  `targ acceptance` exited 0 having run nothing.
- **Options:** (a) They fail, naming the cause. (b) They skip, and the target fails on any skip.
- **Recommendation:** (a): the tests run only when somebody asks for them.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b53 A missing login or Claude Code fails an acceptance test](./793b_decision-log.md#793b53-a-missing-login-or-claude-code-fails-an-acceptance-test).

## 793c29 Does the scenario of a note the agent looked up itself hold a second note?

- **Context:** [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios)
  ended with "nothing from the [vault](../../GLOSSARY.md#vault)", which is also the outcome when the mod never loads.
- **Options:** (a) The vault gains a second [note](../../GLOSSARY.md#permanent-note) that bears on the [prompt](./793a_product-spec.md#793a1a-terms) after the [agent](./793a_product-spec.md#793a1a-terms)'s lookup, and the prompt arrives
  with it and not with the note the agent looked up. (b) The scenario stays as it was.
- **Recommendation:** (a): it tells a delivery that leaves out the held note from one that delivers nothing.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b54 Two acceptance scenarios prove more than that nothing arrived](./793b_decision-log.md#793b54-two-acceptance-scenarios-prove-more-than-that-nothing-arrived).

## 793c30 Does a gate parse the mod's JavaScript, and what if node is missing?

- **Context:** no default gate parsed `register.js`. `targ check-full` runs Go and Markdown checks only.
- **Options:** (a) A test of the dev integration suite parses each file under `agent-instructions/mods/` with node, and
  fails when node is not found. (b) The same, skipped without node. (c) No gate.
- **Recommendation:** (a): a check that cannot run must not pass.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b55 The mod's JavaScript is parsed by a test of the dev integration suite](./793b_decision-log.md#793b55-the-mods-javascript-is-parsed-by-a-test-of-the-dev-integration-suite).

## 793c31 Do the 7 seconds bound the whole hook or the command?

- **Context:** [793d2a Prompt delivery flow](./793d_technical-spec.md#793d2a-prompt-delivery-flow) ended the command at 7
  seconds, but the call for the [session](../../GLOSSARY.md#session)'s ID ran before it, with no bound.
- **Options:** (a) One timer of 7 seconds for everything the mod does for a [prompt](./793a_product-spec.md#793a1a-terms). (b) The command alone.
- **Recommendation:** (a): the 8 seconds of the product promise hold for any reason.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b56 The 7 seconds bound the whole hook, not the command alone](./793b_decision-log.md#793b56-the-7-seconds-bound-the-whole-hook-not-the-command-alone).

## 793c32 Is the stall between the spawn of the command and its first instruction a risk to the product?

- **Context:** when the acceptance tests ran many at once, a timed test failed because the stand-in `engram` that the mod
  started did nothing for the 7 seconds ([793b49 The acceptance run is capped at four tests at once](./793b_decision-log.md#793b49-the-acceptance-run-is-capped-at-four-tests-at-once)
  took it for the load of the other tests). The review of the first group read the failing runs again: Claude Code's
  debug log showed `$.process.run` logging the spawn at once and then `prompt.submit settled in 7060.0ms`, with no output
  from the child, so the stall is between the spawn and the child's first instruction. The effect for a user on a loaded
  machine is that the [prompt](./793a_product-spec.md#793a1a-terms) is held about 7 seconds and nothing is delivered or shown. The cause is not established.
  The cost of one delivery on an idle machine, so that the groups that add work to the path know what they add to,
  measured on the machine this was built on (10 cores, a load average of 2.2 at the start), in an isolated home with a [vault](../../GLOSSARY.md#vault)
  of 10 [notes](../../GLOSSARY.md#permanent-note), 5 runs of each after one run to warm the model: `engram auto deliver` for one prompt took 0.189 to 0.201
  seconds (mean 0.193) with no record of the [session](../../GLOSSARY.md#session) and 0.189 to 0.207 (mean 0.195) with a record of about 300 KB, and
  `engram query` for the same text took 0.188 to 0.197 (mean 0.191). The delivery costs what a query costs, a process
  start, the model and the lookup; reading the record adds about 2 milliseconds.
- **Options:** (a) Record it as an open risk, measure the cost of one delivery on an idle machine so that the groups that
  add work to the path know what they add to, and optimise nothing. (b) Find and remove the stall in this change. (c) Accept
  the risk without recording it.
- **Recommendation:** (a).
- **Blocks:** the sizing of the work that [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) and the later groups add to the per-prompt path.
- **Status:** open. Default (a) taken: [793b49 The acceptance run is capped at four tests at once](./793b_decision-log.md#793b49-the-acceptance-run-is-capped-at-four-tests-at-once).

## 793c33 Does the promise of automatic memory name a minimum version of Claude Code?

- **Context:** [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) and the README
  promise [automatic memory](./793a_product-spec.md#793a1a-terms) for every [session](../../GLOSSARY.md#session) the user starts after [`engram update`](../../GLOSSARY.md#engram-update), without condition, while only a Claude
  Code that loads mods delivers anything (2.1.302 is the build this was verified with, and the earlier attempt found
  2.1.287 the first). Nothing checks the version, so a session of an older release delivers nothing and says nothing
  ([793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses), row 13).
  A non-blocking question from the review of the first group.
- **Options:** (a) The promise stays unconditional; a session that cannot load the mod steps aside. (b) The README and the
  report of `engram update` name the minimum version. (c) `engram update` checks the version and says when it is older.
- **Recommendation:** (a) for this change, as row 13 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses) decided; (b) is cheap if the owner wants it.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses).

## 793c34 Does the delivery merge the parent vault, as `engram query` does?

- **Context:** `engram query` merges the [notes](../../GLOSSARY.md#permanent-note) of the parent [vault](../../GLOSSARY.md#vault) (`ENGRAM_PARENT`) into its
  result ([parent-sync](../../specs/product/parent-sync.md)), and the delivery of the first group looks up the local vault
  only. [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers the
  notes `engram query` returns, so a vault that has a parent is delivered less than it asks for.
- **Options:** (a) Local vault only for now, and a named node, [793e6f The delivery merges the parent vault](./793e_implementation-plan.md#793e6f-the-delivery-merges-the-parent-vault), builds the merge later. (b) Build it in the first group.
- **Recommendation:** (a): the review asked that it be recorded and not built.
- **Blocks:** nothing in the first group.
- **Status:** open. Default (a) taken: [793b57 The delivery looks up the local vault only, until a later node](./793b_decision-log.md#793b57-the-delivery-looks-up-the-local-vault-only-until-a-later-node).

## 793c35 Does the mod tell a prompt the user sent from a prompt of another origin?

- **Context:** the mod delivers for every `prompt.submit` of a [session](../../GLOSSARY.md#session), while [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses) says
  [prompts](./793a_product-spec.md#793a1a-terms) the user sends. Until the first group that starts sessions of its own, there is no prompt of another origin to
  tell apart.
- **Options:** (a) Every origin until [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent), which takes the clause about its own sessions and the filter. (b) Filter now.
- **Recommendation:** (a).
- **Blocks:** nothing in the first group.
- **Status:** open. Default (a) taken: [793b58 The first group delivers for every origin of a prompt](./793b_decision-log.md#793b58-the-first-group-delivers-for-every-origin-of-a-prompt).

## 793c36 Are the restated sentences of 792a2e kept as they are?

- **Context:** [793a5a 792a2e First-use vault creation](./793a_product-spec.md#793a5a-modified-792a2e-first-use-vault-creation)
  restates the canonical requirement with one clause added, and its sentence about the files of a new [vault](../../GLOSSARY.md#vault) names file
  names, which a product specification keeps out.
- **Options:** (a) Keep the sentence verbatim: it is the canonical requirement's text. (b) Reword it.
- **Recommendation:** (a): a reworded restatement changes the canonical requirement under a Modified operation.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b60 The restated sentences of 792a2e are kept as they are](./793b_decision-log.md#793b60-the-restated-sentences-of-792a2e-are-kept-as-they-are).

## 793c37 Are the fourteen scenarios of the second group the minimum?

- **Context:** step 2 for the second group wrote fourteen scenarios, one for each clause of
  [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), of the three interactions the group takes and of the two clauses of
  [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) and [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) that it proves.
- **Options:** (a) The fourteen as written. (b) Fewer: the two scenarios of [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again) folded into one, or the
  three kinds of lesson that come from the [agent](./793a_product-spec.md#793a1a-terms)'s own words folded into one.
- **Recommendation:** (a): the clause lists the kinds, and "whoever wrote that [note](../../GLOSSARY.md#permanent-note)" names two writers a test can tell apart.
- **Blocks:** nothing; the scenarios are written and worked.
- **Status:** open. Default (a) taken: [793b62 The scenarios of the second group](./793b_decision-log.md#793b62-the-scenarios-of-the-second-group).

## 793c38 Does automatic memory start sessions of its own?

- **Context:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) forbids delivering to or writing from the [sessions](../../GLOSSARY.md#session) that [automatic memory](./793a_product-spec.md#793a1a-terms) starts itself, and the plan
  placed the guard against them in [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent), the first group that does work after a [turn](./793a_product-spec.md#793a1a-terms).
- **Options:** (a) None: the judgement is one call of the mod's model API, which raises no [prompt](./793a_product-spec.md#793a1a-terms) and no turn, so the clause holds by
  construction and its scenario observes that no other session exists. (b) A headless session of its own for the
  judgement, with a marker in its environment that makes the mod step aside in it.
- **Recommendation:** (a): (b) starts a second process that loads the user's whole configuration for each turn.
- **Blocks:** the scenario of the clause, [793a2d1b Automatic memory's work starts no session of its own](./793a_product-spec.md#793a2d1-uat-scenarios).
- **Status:** open. Default (a) taken: [793b63 A turn is judged by one model call, and no session is started](./793b_decision-log.md#793b63-a-turn-is-judged-by-one-model-call-and-no-session-is-started).

## 793c39 Which origins count as the prompts the user sends?

- **Context:** the mod delivered for every origin of a [prompt](./793a_product-spec.md#793a1a-terms) in the first group ([793c35 Does the mod tell a prompt the user sent from a prompt of another origin?](./793c_open-questions.md#793c35-does-the-mod-tell-a-prompt-the-user-sent-from-a-prompt-of-another-origin)), and this group
  adds the filter. A headless [session](../../GLOSSARY.md#session)'s prompt has the origin `sdk`, the same as a prompt another program sends with the SDK.
- **Options:** (a) `composer`, `bridge`, `slack-ping` and `sdk`. (b) `composer`, `bridge` and `slack-ping`, so a headless session gets nothing.
- **Recommendation:** (a): (b) would leave out the user's own `claude -p` runs, and the tests drive one.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b64 The prompts the user sends](./793b_decision-log.md#793b64-the-prompts-the-user-sends).

## 793c40 Is the window of the judgement the right one?

- **Context:** [793b65 What the judgement reads, and what it is asked](./793b_decision-log.md#793b65-what-the-judgement-reads-and-what-it-is-asked) gives the model the last 6 lines before the [turn](./793a_product-spec.md#793a1a-terms) and the
  turn, without the inputs and results of tools, cut to 1,200 characters a line and 8,000 a window.
- **Options:** (a) As built. (b) The whole turn with the results of tools. (c) A smaller window, the turn alone.
- **Recommendation:** (a): a lesson comes from words and outcomes, and the results of tools are where the credentials
  and the bulk are. The probes and the live runs of this change used (a).
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b65 What the judgement reads, and what it is asked](./793b_decision-log.md#793b65-what-the-judgement-reads-and-what-it-is-asked).

## 793c41 Is the risk of a secret of another shape acceptable?

- **Context:** [793b69 Credentials are redacted by shape, and the rest is a known risk](./793b_decision-log.md#793b69-credentials-are-redacted-by-shape-and-the-rest-is-a-known-risk) redacts the shapes it lists. A password written in a sentence of more than
  six words or without `is` or `:` (a sentence of up to six words is redacted since
  [793b78 The promise names the forms of credential that are redacted](./793b_decision-log.md#793b78-the-promise-names-the-forms-of-credential-that-are-redacted)), a
  token split over two lines, and the wrapped and percent-encoded forms that the earlier attempt covered are not
  redacted, so such a secret in a [turn](./793a_product-spec.md#793a1a-terms) reaches the model and may reach a [note](../../GLOSSARY.md#permanent-note).
- **Options:** (a) The shapes as built, the rest a known risk. (b) Port the earlier attempt's patterns. (c) Do not
  write a note from a turn in which the redaction found a credential.
- **Recommendation:** (a) for this change, with (c) as the next step if the owner finds the risk too large: (b) adds
  patterns without making the risk go away, and (c) loses real lessons.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b69 Credentials are redacted by shape, and the rest is a known risk](./793b_decision-log.md#793b69-credentials-are-redacted-by-shape-and-the-rest-is-a-known-risk).

## 793c42 Should the bound of 2 minutes be tightened?

- **Context:** [793b70 The capture reports its own failures, and nothing times it as a whole](./793b_decision-log.md#793b70-the-capture-reports-its-own-failures-and-nothing-times-it-as-a-whole) measured 246 [notes](../../GLOSSARY.md#permanent-note) of the live runs, written 0.9 to 6.0 seconds after the [turn](./793a_product-spec.md#793a1a-terms)
  ended, and enforces no bound over the chain.
- **Options:** (a) Keep the 2 minutes of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent). (b) Say 30 seconds. (c) Say 10 seconds.
- **Recommendation:** (a): the number is the promise under load, when the model's call may take its 60 seconds and each
  command its 30, which together are the 2 minutes; (b) and (c) would be broken by the first slow call and nothing
  enforces them.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b70 The capture reports its own failures, and nothing times it as a whole](./793b_decision-log.md#793b70-the-capture-reports-its-own-failures-and-nothing-times-it-as-a-whole).

## 793c43 Should a missing vault end the capture before the model is asked?

- **Context:** [793b71 The commands of the group auto take no first-use step](./793b_decision-log.md#793b71-the-commands-of-the-group-auto-take-no-first-use-step) lets the capture run when the [vault](../../GLOSSARY.md#vault) was removed: each [turn](./793a_product-spec.md#793a1a-terms) costs a
  call of the model, and records two failures that name the lock of a vault that is not there.
- **Options:** (a) As built. (b) The judgement prints nothing when the vault is missing, so there is no call. (c) The
  writing records a skip with the reason that the vault does not exist, and the judgement still runs.
- **Recommendation:** (a) until the display of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows the user that message; then (c).
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b71 The commands of the group auto take no first-use step](./793b_decision-log.md#793b71-the-commands-of-the-group-auto-take-no-first-use-step).

## 793c44 Is the window between a note's file and its embedding a risk?

- **Context:** during the 0.3 seconds that the learn path takes to embed a [note](../../GLOSSARY.md#permanent-note) after it creates the file, a lookup in a
  [vault](../../GLOSSARY.md#vault) whose only note it is fails (`vault has notes but no current-model embeddings`), and so does a delivery; a
  vault with other notes is not known to fail. [793b73 The tests of an absence wait for the end of the write, and for a settling time](./793b_decision-log.md#793b73-the-tests-of-an-absence-wait-for-the-end-of-the-write-and-for-a-settling-time) found it.
- **Options:** (a) A risk to the first [prompt](./793a_product-spec.md#793a1a-terms) after the first note of an empty vault, which steps aside as any failed
  delivery does. (b) Write the sidecar before the note, in the learn path. (c) Let the lookup skip a note with no
  embedding.
- **Recommendation:** (a): a prompt the user types 0.3 seconds after a note appeared is rare, the cost is one prompt
  without its notes, and (b) and (c) change the lookup of the [agent](./793a_product-spec.md#793a1a-terms)'s own use of memory, which
  [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working) keeps unchanged.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b73 The tests of an absence wait for the end of the write, and for a settling time](./793b_decision-log.md#793b73-the-tests-of-an-absence-wait-for-the-end-of-the-write-and-for-a-settling-time).

## 793c45 Is the reading of a prompt in the record the rule of the transcript-reader?

- **Context:** [793b74 A row the harness wrote is no prompt](./793b_decision-log.md#793b74-a-row-the-harness-wrote-is-no-prompt) makes the judgement read a [prompt](./793a_product-spec.md#793a1a-terms) as the [transcript](../../GLOSSARY.md#transcript)-reader does, and adds the flag of meta rows and the
  origin of the row, which the record of Claude Code 2.1.302 stamps.
- **Options:** (a) The rule of 792b15c3, the flag and the origin. (b) The origin and the flag alone. (c) A list of marks of
  the judgement's own.
- **Recommendation:** (a): (b) leaves the command rows, and (c) copies a rule that has an owner.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b74 A row the harness wrote is no prompt](./793b_decision-log.md#793b74-a-row-the-harness-wrote-is-no-prompt).

## 793c46 Is 2 seconds for a record of 1 MB the right bound for the judgement?

- **Context:** [793b75 The record is read once, and the lines of the window are the only ones redacted](./793b_decision-log.md#793b75-the-record-is-read-once-and-the-lines-of-the-window-are-the-only-ones-redacted) measured 2.2 seconds for each reading of a record of 2 MB before the change, and the
  capture has a bound of 2 minutes ([793b70 The capture reports its own failures, and nothing times it as a whole](./793b_decision-log.md#793b70-the-capture-reports-its-own-failures-and-nothing-times-it-as-a-whole)).
- **Options:** (a) 2 seconds for 1 MB. (b) A tighter bound. (c) No bound in the spec.
- **Recommendation:** (a): the bound is a regression guard with room for a loaded machine, and the command now spends most
  of that time parsing.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b75 The record is read once, and the lines of the window are the only ones redacted](./793b_decision-log.md#793b75-the-record-is-read-once-and-the-lines-of-the-window-are-the-only-ones-redacted).

## 793c47 Is the turn the one that began with the text the mod saw?

- **Context:** [793b76 A turn is found by the text it began with](./793b_decision-log.md#793b76-a-turn-is-found-by-the-text-it-began-with) judges the [turn](./793a_product-spec.md#793a1a-terms) that began with the text its start carried, and not the last [prompt](./793a_product-spec.md#793a1a-terms) of the record.
  A prompt that a user types ahead is no longer the turn of the one before it; a slash command, whose text is not the
  text of its row, falls back to the last prompt.
- **Options:** (a) The text the turn started with, the last prompt as the fallback. (b) The last prompt of the record, as
  before. (c) The prompt ID of the record, if the event gives it later.
- **Recommendation:** (a): (b) loses the correction of a turn when the user types ahead, which is the common way to work.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b76 A turn is found by the text it began with](./793b_decision-log.md#793b76-a-turn-is-found-by-the-text-it-began-with).

## 793c48 Is writing a < as &lt; the right defence of the window and the notes?

- **Context:** [793b77 The instructions are the system of the request, and a < is written &lt;](./793b_decision-log.md#793b77-the-instructions-are-the-system-of-the-request-and-a--is-written-lt) separates the instructions from the record and escapes the character that opens a
  tag in the lines of the window and the [notes](../../GLOSSARY.md#permanent-note). A lesson that quotes code may come back with `&lt;` in a field if the model copies it so.
- **Options:** (a) The escape, and the instructions say what it stands for. (b) A delimiter with a random value in each
  request. (c) Neither.
- **Recommendation:** (a): it is deterministic and testable, and the instructions ask for `<` where it is meant.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b77 The instructions are the system of the request, and a < is written &lt;](./793b_decision-log.md#793b77-the-instructions-are-the-system-of-the-request-and-a--is-written-lt).

## 793c49 Should the promise that a note holds no credential stay absolute?

- **Context:** [793b78 The promise names the forms of credential that are redacted](./793b_decision-log.md#793b78-the-promise-names-the-forms-of-credential-that-are-redacted) narrows the promise of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) to the forms the redaction
  recognises. An absolute promise that no [note](../../GLOSSARY.md#permanent-note) holds a credential of the [turn](./793a_product-spec.md#793a1a-terms) cannot be proven without the model's cooperation: the live scenario passes whether or not
  the redaction runs, and only the properties of the redaction fail for a form it misses.
- **Options:** (a) The promise names the forms, as built; the rest stays the risk of
  [793c41 Is the risk of a secret of another shape acceptable?](./793c_open-questions.md#793c41-is-the-risk-of-a-secret-of-another-shape-acceptable). (b) The promise stays absolute, and no note is written from a turn in
  which the redaction found a credential. (c) The promise stays absolute, and the model checks each note for a secret
  before it is written.
- **Recommendation:** (a) for this change, with (b) as the next step if the owner finds the risk too large: (c) makes the
  promise depend on the model, and (b) lowers the risk without making the promise provable.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b78 The promise names the forms of credential that are redacted](./793b_decision-log.md#793b78-the-promise-names-the-forms-of-credential-that-are-redacted).

## 793c50 Is a lesson of a cut reply worth writing?

- **Context:** [793b79 A field of a lesson is one line, and a reply that is cut keeps its whole lessons](./793b_decision-log.md#793b79-a-field-of-a-lesson-is-one-line-and-a-reply-that-is-cut-keeps-its-whole-lessons) keeps the whole lessons of a reply that the cap of the model's reply cut. A lesson
  before the cut was judged as the others were, and the lesson the cut ended is lost.
- **Options:** (a) Keep the whole lessons, and record the cut. (b) Write nothing from a reply that was cut, as before.
  (c) Raise the cap of the reply.
- **Recommendation:** (a): a lesson is whole or it is not, whatever follows it; (c) does not remove the cut.
- **Blocks:** nothing.
- **Status:** open. Default (a) taken: [793b79 A field of a lesson is one line, and a reply that is cut keeps its whole lessons](./793b_decision-log.md#793b79-a-field-of-a-lesson-is-one-line-and-a-reply-that-is-cut-keeps-its-whole-lessons).
