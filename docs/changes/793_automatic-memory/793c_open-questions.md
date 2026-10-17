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

- **Context:** [793a4 Not in the minimal set](./793a_product-spec.md#793a4-not-in-the-minimal-set) leaves out [subagents](../../GLOSSARY.md#subagent) and the chunk-index lookup. In the earlier attempt the owner
  answered the opposite on 2026-10-11 for subagents and [prompt](./793a_product-spec.md#793a1a-terms) sources: capture inside subagent [sessions](../../GLOSSARY.md#session), and
  memory for every prompt regardless of source. A lookup that included the chunk index took about 12 seconds
  on his [vault](../../GLOSSARY.md#vault) there, longer than the 8 seconds of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Options:** (a) The user's own sessions and [notes](../../GLOSSARY.md#permanent-note) only. (b) Add subagents. (c) Add the chunk-index
  lookup.
- **Recommendation:** (a) for the minimal set. This reverses his earlier answers, so it needs his confirmation.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), and [793a4 Not in the minimal set](./793a_product-spec.md#793a4-not-in-the-minimal-set).
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
  [vault](../../GLOSSARY.md#vault) may be under version control. The earlier attempt held a MUST NOT on it (the owner, 2026-10-11) that
  its final set no longer carries.
- **Options:** (a) No rule in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec). (b) Add "a note MUST NOT contain a secret" to [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Recommendation:** (b). The default is (a) only because [docs/AGENTS.md](../../AGENTS.md#general-documentation-rules)
  says to document confirmed scope and the restart did not confirm it; the owner can add it with one sentence.
- **Blocks:** nothing now; the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) if (b) is chosen.
- **Status:** open. Default (a) taken: [793b24 Secrets in written notes are not addressed](./793b_decision-log.md#793b24-secrets-in-written-notes-are-not-addressed).

## 793c9 Does a turn with nothing to report get a line?

- **Context:** [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows a line only on a [turn](./793a_product-spec.md#793a1a-terms) on which something was delivered, written or
  skipped.
- **Options:** (a) No line. (b) A line on every turn, so the user can tell [automatic memory](./793a_product-spec.md#793a1a-terms) is on.
- **Recommendation:** (a), as in the earlier attempt, which omitted zero counts.
- **Blocks:** the scenarios of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did), and any scenario that needs a turn with nothing to report.
- **Status:** open. Default (a) taken: [793b25 A turn with nothing to report shows no line](./793b_decision-log.md#793b25-a-turn-with-nothing-to-report-shows-no-line).

## 793c10 Does "already has" survive a compaction?

- **Context:** [793a3b Nothing the agent already has is delivered again](./793a_product-spec.md#793a3b-nothing-the-agent-already-has-is-delivered-again) does not deliver a [note](../../GLOSSARY.md#permanent-note) the [agent](./793a_product-spec.md#793a1a-terms) already has in the [session](../../GLOSSARY.md#session). When the
  session's context is compacted, the agent may no longer hold the note's text.
- **Options:** (a) Per session: a note delivered once is not delivered again, whatever the agent still holds.
  (b) Deliver it again after a compaction.
- **Recommendation:** (a) for the minimal set: (b) needs knowledge of what the agent still holds.
- **Blocks:** the scenarios of [793a3b Nothing the agent already has is delivered again](./793a_product-spec.md#793a3b-nothing-the-agent-already-has-is-delivered-again).
- **Status:** open. Default (a) taken: [793b27 "Already has" is per session](./793b_decision-log.md#793b27-already-has-is-per-session).
