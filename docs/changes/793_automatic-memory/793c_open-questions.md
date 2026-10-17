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
- **Blocks:** the scenarios of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) and [793a3d Failures never break a prompt and always show](./793a_product-spec.md#793a3d-failures-never-break-a-prompt-and-always-show).
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
- **Recommendation:** (a). The owner's answer on 2026-10-11 in the earlier attempt was that volume equals what
  the agent's own lookup returns, with no per-[turn](./793a_product-spec.md#793a1a-terms) or per-[session](../../GLOSSARY.md#session) bounds.
- **Blocks:** scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that involve many related notes.
- **Status:** open. Default (a) taken: [793b19 Volume of delivered notes](./793b_decision-log.md#793b19-volume-of-delivered-notes).

## 793c4 Are there modes or an off switch?

- **Context:** issue #793 expected observe and active modes and a kill switch. The earlier attempt removed
  modes on the owner's answer of 2026-10-11: active whenever installed, and disabling or uninstalling stops it.
- **Options:** (a) None. (b) An off switch. (c) An observe mode first.
- **Recommendation:** (a) for the minimal set; (b) and (c) can be separate changes.
- **Blocks:** nothing in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec).
- **Status:** open. Default (a) taken: [793b20 No modes and no off switch](./793b_decision-log.md#793b20-no-modes-and-no-off-switch).

## 793c5 Are lessons written straight into the vault or held for review?

- **Context:** issue #793's open decision 4. [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) says the [note](../../GLOSSARY.md#permanent-note) must be in the [vault](../../GLOSSARY.md#vault) within 2
  minutes.
- **Options:** (a) Straight in, marked automatic. (b) As [pending offers](../../GLOSSARY.md#offer) reviewed with the curate [skill](../../GLOSSARY.md#skill).
- **Recommendation:** (a), the owner's answer on 2026-10-11 in the earlier attempt.
- **Blocks:** what "in the vault" means in the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Status:** open. Default (a) taken: [793b21 Lessons are written straight into the vault](./793b_decision-log.md#793b21-lessons-are-written-straight-into-the-vault).

## 793c6 Is the scope top-level sessions and notes only?

- **Context:** [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions) leaves out [subagents](../../GLOSSARY.md#subagent) and the chunk-index lookup. In the earlier attempt the owner
  answered the opposite on 2026-10-11 for subagents and [prompt](./793a_product-spec.md#793a1a-terms) sources: capture inside subagent [sessions](../../GLOSSARY.md#session), and
  memory for every prompt regardless of source (`old 793c6` and `old 793c63`, `old 793c5`, 2026-10-11). The
  full list of reversed answers is asked in [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse). A lookup that included the chunk index took about 12 seconds
  on his [vault](../../GLOSSARY.md#vault) there, longer than the 8 seconds of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Options:** (a) The user's own sessions and [notes](../../GLOSSARY.md#permanent-note) only. (b) Add subagents. (c) Add the chunk-index
  lookup.
- **Recommendation:** (a) for the minimal set. This reverses his earlier answers, so it needs his confirmation.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), and [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions).
- **Status:** open. Default (a) taken: [793b22 The scope is top-level sessions and notes only](./793b_decision-log.md#793b22-the-scope-is-top-level-sessions-and-notes-only).

## 793c7 Are there capture rate or precision targets?

- **Context:** issue #793's open decision 1. In the earlier attempt the owner answered on 2026-10-11 that the
  capture rate must exceed the 5.6% baseline and that precision must be recorded.
- **Options:** (a) None in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec); the scenarios are the proof. (b) The earlier targets.
- **Recommendation:** (a): measuring a rate needs machinery the minimal set leaves out.
- **Blocks:** nothing in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec).
- **Status:** open. Default (a) taken: [793b23 No capture rate or precision targets](./793b_decision-log.md#793b23-no-capture-rate-or-precision-targets).

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

## 793c9 Does a turn with nothing to report get a line?

- **Context:** [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows a line only on a [turn](./793a_product-spec.md#793a1a-terms) on which something was delivered, written or
  skipped.
- **Options:** (a) No line. (b) A line on every turn, so the user can tell [automatic memory](./793a_product-spec.md#793a1a-terms) is on.
- **Recommendation:** (a), as in the earlier attempt, which omitted zero counts.
- **Blocks:** the scenarios of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did), and any scenario that needs a turn with nothing to report.
- **Status:** open. Default (a) taken: [793b25 A turn with nothing to report shows no line](./793b_decision-log.md#793b25-a-turn-with-nothing-to-report-shows-no-line).

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
| 2 | [Prompt](./793a_product-spec.md#793a1a-terms) sources | memory for every prompt regardless of source (`old 793c5`, 2026-10-11) | prompts the user sends |
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
