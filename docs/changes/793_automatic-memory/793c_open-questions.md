# Automatic Memory - Open Questions

A default stands until the owner answers; his answer is recorded in the entry with its date. A question that blocks nothing is an author choice and is not asked: its entry is withdrawn to its decision in [793b Automatic Memory - Decision Log](./793b_decision-log.md#automatic-memory---decision-log) (`docs/AGENTS.md`).

## 793c1 How long may a prompt be held?

- **Context:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) promises that a [prompt](./793a_product-spec.md#793a1a-terms) is never held longer than a short bound; a promise with no number cannot be verified.
- **Options:** (a) 8 seconds. (b) A shorter bound, such as 3 seconds. (c) A longer one. (d) The owner's earlier rule: no longer than the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup on the same [vault](../../GLOSSARY.md#vault) and input (`old 793c7` item 4, `old 793c11` (b), 2026-10-11; 10.6 to 11.0 seconds on his vault).
- **Recommendation:** (a). The earlier attempt's author set it and the owner did not raise it when offered a longer hold (`old 793b317`, `old 793c87`, 2026-10-14), and a lookup of [notes](../../GLOSSARY.md#permanent-note) alone took about 1.4 seconds there; (d) would allow more than 10 seconds on his vault.
- **Blocks:** the hold scenarios of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), worked in [793e6a On after update, and memory arrives with the prompt](./793e_implementation-plan.md#793e6a-on-after-update-and-memory-arrives-with-the-prompt).
- **Status:** open; default (a) taken in [793b17 A prompt is held for at most 8 seconds](./793b_decision-log.md#793b17-a-prompt-is-held-for-at-most-8-seconds).

## 793c2 How soon after a turn must automatic memory have finished writing?

- **Context:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) say "within 2 minutes" because "shortly" cannot be verified. 246 live [notes](../../GLOSSARY.md#permanent-note) took 0.9 to 6.0 seconds ([793b70 The capture reports its own failures, and nothing times it as a whole](./793b_decision-log.md#793b70-the-capture-reports-its-own-failures-and-nothing-times-it-as-a-whole)); the stages together are 115 seconds ([793b85 Each stage of the capture has a bound of its own](./793b_decision-log.md#793b85-each-stage-of-the-capture-has-a-bound-of-its-own)).
- **Options:** (a) 2 minutes. (b) 1 minute. (c) 5 minutes. (d) 30 seconds. (e) 10 seconds.
- **Recommendation:** (a), until a measured capture replaces it; (d) and (e) would break at the first slow call, and nothing enforces them.
- **Blocks:** the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) and [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later).
- **Status:** open; default (a) taken in [793b18 Writing finishes within 2 minutes of a turn](./793b_decision-log.md#793b18-writing-finishes-within-2-minutes-of-a-turn).

## 793c3 Is the volume of delivered notes limited?

- **Context:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers the [notes](../../GLOSSARY.md#permanent-note) that bear on a [prompt](./793a_product-spec.md#793a1a-terms) and states no number; without a limit, a [vault](../../GLOSSARY.md#vault) with many related notes could flood the [agent](./793a_product-spec.md#793a1a-terms)'s context.
- **Options:** (a) No number in the product spec; the technical spec limits the notes to the number the agent's own lookup returns. (b) A maximum stated in [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt).
- **Recommendation:** (a). The owner answered on 2026-10-11 that volume equals what the agent's own lookup returns, with no per-[turn](./793a_product-spec.md#793a1a-terms) or per-[session](../../GLOSSARY.md#session) bounds (`old 793c7` items 1 and 2).
- **Blocks:** the scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that involve many related notes.
- **Status:** open; default (a) taken in [793b19 Volume of delivered notes](./793b_decision-log.md#793b19-volume-of-delivered-notes).

## 793c5 Are lessons written straight into the vault or held for review?

- **Context:** issue #793's open decision 4; [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) says the [note](../../GLOSSARY.md#permanent-note) must be in the [vault](../../GLOSSARY.md#vault) within 2 minutes.
- **Options:** (a) Straight in, marked automatic. (b) As [pending offers](../../GLOSSARY.md#offer) reviewed with the curate [skill](../../GLOSSARY.md#skill).
- **Recommendation:** (a), the owner's answer on 2026-10-11 (`old 793c4`).
- **Blocks:** what "in the vault" means in the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Status:** open; default (a) taken in [793b21 Lessons are written straight into the vault](./793b_decision-log.md#793b21-lessons-are-written-straight-into-the-vault).

## 793c6 Is the scope top-level sessions and notes only?

- **Context:** [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions) leaves out [subagents](../../GLOSSARY.md#subagent) and the chunk-index lookup, against the owner's answers of 2026-10-11 for subagents and [prompt](./793a_product-spec.md#793a1a-terms) sources (`old 793c6`, `old 793c63`, `old 793c5`; asked together in [793c13 Which earlier owner answers may the minimal set reverse?](#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse)). A lookup with the chunk index took about 12 seconds on his [vault](../../GLOSSARY.md#vault), over the 8 of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Options:** (a) The user's own [sessions](../../GLOSSARY.md#session) and [notes](../../GLOSSARY.md#permanent-note) only. (b) Add subagents. (c) Add the chunk-index lookup.
- **Recommendation:** (a) for the minimal set; it reverses his earlier answers, so it needs his confirmation.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), and [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions).
- **Status:** open; default (a) taken in [793b22 The scope is top-level sessions and notes only](./793b_decision-log.md#793b22-the-scope-is-top-level-sessions-and-notes-only).

## 793c8 What about secrets in written notes?

- **Context:** automatic [notes](../../GLOSSARY.md#permanent-note) quote what the user typed, including anything typed in a correction, and the [vault](../../GLOSSARY.md#vault) may be under version control; the earlier attempt's final product spec no longer carried the owner's MUST NOT on it (`old 793c7` item 10, 2026-10-11).
- **Options:** (a) No rule in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec). (b) Add "a note MUST NOT contain a secret" to [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Recommendation:** (b).
- **Blocks:** the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
- **Status:** answered (Joe, 2026-10-11 in the earlier attempt, `old 793c7` item 10; the redaction approach and its residual risk ratified 2026-10-13, `old 793b324`): (b), a note MUST NOT contain a credential, token, key or password that appeared in the [turn](./793a_product-spec.md#793a1a-terms). Applied in [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793b24 A note holds no secret](./793b_decision-log.md#793b24-a-note-holds-no-secret).

## 793c9 Does a turn with nothing to report show anything?

- **Context:** [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows something only on a [turn](./793a_product-spec.md#793a1a-terms) on which something was delivered, written or could not be done.
- **Options:** (a) Nothing is shown. (b) Something is shown on every turn, so the user can tell [automatic memory](./793a_product-spec.md#793a1a-terms) is on.
- **Recommendation:** (a), as in the earlier attempt, which omitted zero counts.
- **Blocks:** the scenarios of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did), and any scenario that needs a turn with nothing to report.
- **Status:** open; default (a) taken in [793b25 What is shown after a turn, and when](./793b_decision-log.md#793b25-what-is-shown-after-a-turn-and-when).

## 793c10 Does "already has" survive a compaction?

- **Context:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does not deliver a [note](../../GLOSSARY.md#permanent-note) the [agent](./793a_product-spec.md#793a1a-terms) already has in the [session](../../GLOSSARY.md#session); after a compaction it may no longer hold the text. The owner answered the other way (`old 793c7` item 5, 2026-10-11; row 11 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)).
- **Options:** (a) Per session: a note delivered once is not delivered again, whatever the agent still holds. (b) Deliver it again after a compaction.
- **Recommendation:** (a) for the minimal set: (b) needs knowledge of what the agent still holds.
- **Blocks:** the scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that test what the agent already has.
- **Status:** open; default (a) taken in [793b27 "Already has" is per session](./793b_decision-log.md#793b27-already-has-is-per-session).

## 793c12 Is memory delivered at mid-task moments?

- **Context:** issue #793 asks for recall "at human [prompts](./793a_product-spec.md#793a1a-terms), tool results and [subagent](../../GLOSSARY.md#subagent) dispatch"; [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers at the prompt only. The main [agent](./793a_product-spec.md#793a1a-terms) also recalled mid-task by its own judgement ([792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments)); the owner approved delivery at those moments in the earlier attempt (`old 793b187`, `old 793b240`), and the minimal set leaves them out ([793b8 Exclusions](./793b_decision-log.md#793b8-exclusions)).
- **Options:** (a) The prompt only. (b) The prompt and tool results. (c) The prompt, tool results and the four decision moments.
- **Recommendation:** (a) for the minimal set: the mid-task re-queries were moments the main agent chose by judgement, and finding them from outside needs a model judgement or a look at every tool result; (b) or (c) can be a later change.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) if (b) or (c) is chosen.
- **Status:** open; default (a) taken in [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions).

## 793c13 Which earlier owner answers may the minimal set reverse?

- **Context:** the minimal set leaves out or changes things the owner answered in the earlier attempt (the 14 rows of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)). Rows 1, 2 and 5 are also [793c6 Is the scope top-level sessions and notes only?](#793c6-is-the-scope-top-level-sessions-and-notes-only), row 4 [793c12 Is memory delivered at mid-task moments?](#793c12-is-memory-delivered-at-mid-task-moments), row 11 [793c10 Does "already has" survive a compaction?](#793c10-does-already-has-survive-a-compaction), row 12 [793c1 How long may a prompt be held?](#793c1-how-long-may-a-prompt-be-held); an answer here answers those.
- **Options:** (a) Accept every reversal for the minimal set; any row can come back as a separate change. (b) Keep named rows in the set (the owner names them by number).
- **Recommendation:** (a): the owner asked for the minimal set, and each row needs a measurement, a second model judgement or machinery the set leaves out.
- **Blocks:** the scope of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt), [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) for the rows kept.
- **Status:** open; defaults taken in [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses).

## 793c14 Is a turn that is followed by the end of the session covered?

- **Context:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) promise their work within 2 minutes after a [turn](./793a_product-spec.md#793a1a-terms); a user who ends the [session](../../GLOSSARY.md#session) right after leaves no time. The earlier attempt recorded such a turn as lost (`old 793b203`), kept work for the next session (`old 793c65`), then picked up leftover sessions (`old 793c67`); the minimal set has none of those (row 8 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)).
- **Options:** (a) Not covered: the promise holds while the session stays open for the 2 minutes. (b) Covered: the work finishes after the session ends, or the next session finishes it.
- **Recommendation:** (a) for the minimal set: (b) needs work that outlives the session. The cost is that a lesson in the last turn before an exit can be lost.
- **Blocks:** the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) for a session that ends within 2 minutes after a turn.
- **Status:** open; default (a) taken in [793b37 A turn followed by the end of the session is not covered](./793b_decision-log.md#793b37-a-turn-followed-by-the-end-of-the-session-is-not-covered).

## 793c15 Should the activity be shown in the agent's reply, so that it reaches the claude.ai mobile and web view?

- **Context:** the owner follows [sessions](../../GLOSSARY.md#session) from the claude.ai mobile app and sees none of what [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows there; the only route the earlier attempt found was the [agent](./793a_product-spec.md#793a1a-terms)'s own reply (`old 793c93`).
- **Options:** (a) Leave it in the session only. (b) Ask the agent to end its reply with one fixed activity line.
- **Recommendation:** (a).
- **Blocks:** nothing.
- **Status:** parked (Joe, 2026-10-16, earlier attempt, `old 793c93`): "Nah, leave it for now." Revisit after [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) has landed.

## 793c16 Are the ten scenarios of the first group the minimum?

Withdrawn: author choice, [793b40 The scenarios of the first group](./793b_decision-log.md#793b40-the-scenarios-of-the-first-group).

## 793c17 Is the automatic behavior inside a session one mod deployed with the skills?

- **Context:** automatic behavior inside a [session](../../GLOSSARY.md#session) needs a plugin of function hooks, a mod, that [`engram update`](../../GLOSSARY.md#engram-update) can deploy without editing a file the user owns.
- **Options:** (a) One mod, `engram-memory`, in `agent-instructions/mods/`, deployed with the [skills](../../GLOSSARY.md#skill), running a new command for each [prompt](./793a_product-spec.md#793a1a-terms). (b) A `UserPromptSubmit` hook in the user's `settings.json`.
- **Recommendation:** (a): (b) makes `engram update` edit a file the user owns on every run.
- **Blocks:** [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update).
- **Status:** open; default (a) taken in [793b41 The mod, its place and its command](./793b_decision-log.md#793b41-the-mod-its-place-and-its-command).

## 793c18 What does a delivered block say?

- **Context:** the block puts [notes](../../GLOSSARY.md#permanent-note) in the [agent](./793a_product-spec.md#793a1a-terms)'s context, and the earlier attempt's guard keeps a note from acting as an instruction.
- **Options:** (a) A framing line that says the lines below are notes delivered by the memory system, and one line per note: its file name, situation and body. (b) Whole notes. (c) A count and no text.
- **Recommendation:** (a): (b) grows with the [vault](../../GLOSSARY.md#vault), and (c) makes the agent look each note up.
- **Blocks:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt).
- **Status:** open; default (a) taken in [793b42 What the block says](./793b_decision-log.md#793b42-what-the-block-says).

## 793c19 Is the body of a note delivered whole, and on one line?

- **Context:** the body of a delivered [note](../../GLOSSARY.md#permanent-note) is not cut and is written on one line, so a note of several steps, a runbook, loses its line structure; a non-blocking question from the review of the first group.
- **Options:** (a) Whole, and on one line. (b) Cut to a length, as the earlier attempt cut at 400 characters. (c) Whole, with its lines kept, which needs a way to keep a line from imitating the framing line or another note ([793b43 A note cannot reshape the block](./793b_decision-log.md#793b43-a-note-cannot-reshape-the-block)).
- **Recommendation:** (a) for the minimal set; (c) if the loss of structure matters to the owner.
- **Blocks:** the line form of a delivered note, [793d4c Framing the notes of a block as data](./793d_technical-spec.md#793d4c-framing-the-notes-of-a-block-as-data).
- **Status:** open; default (a) taken in [793b44 Only the notes that match are delivered, whole](./793b_decision-log.md#793b44-only-the-notes-that-match-are-delivered-whole).

## 793c20 What does a session hold, so that a note is not delivered twice?

- **Context:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does not deliver a [note](../../GLOSSARY.md#permanent-note) the [agent](./793a_product-spec.md#793a1a-terms) already has in the [session](../../GLOSSARY.md#session); something has to say what it has.
- **Options:** (a) The file names on the note lines of the blocks delivered, and the command and result of each `engram query` and `engram show` the agent ran, read from the session's record. (b) Every tool result, the first version. (c) A ledger kept by the mod.
- **Recommendation:** (a): (b) counted an `ls` listing and missed an `engram show`, and (c) is lost with the process.
- **Blocks:** the scenarios of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) that test what the agent already has.
- **Status:** open; default (a) taken in [793b45 What a session holds](./793b_decision-log.md#793b45-what-a-session-holds).

## 793c21 How long may the mod's command run, short of the hold?

- **Context:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) holds a [prompt](./793a_product-spec.md#793a1a-terms) for 8 seconds at most, and the mod needs time to start the command and run the prompt's hooks.
- **Options:** (a) 7 seconds for the command, one second left. (b) 8 seconds for the command. (c) A limit inside the command.
- **Recommendation:** (a): the platform's own limit does not count the call, and a command that hangs cannot end itself.
- **Blocks:** the hold scenarios of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Status:** open; default (a) taken in [793b46 The command is ended at 7 seconds](./793b_decision-log.md#793b46-the-command-is-ended-at-7-seconds).

## 793c22 Does a dry run report the mod as a mod?

- **Context:** a preview exists to tell the user what an update would [turn](./793a_product-spec.md#793a1a-terms) on, and the report listed the mod under the source of a [skill](../../GLOSSARY.md#skill).
- **Options:** (a) The mod is listed in the Claude Code section with `agent-instructions/mods/<name>/` as its source. (b) As before, under the skills. (c) A section of the report for mods.
- **Recommendation:** (a): (b) names a directory the mod does not come from, and (c) changes the report for every reader.
- **Blocks:** [793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios).
- **Status:** open; default (a) taken in [793b47 A dry run reports the mod as a mod](./793b_decision-log.md#793b47-a-dry-run-reports-the-mod-as-a-mod).

## 793c23 Does the conformance check read this change's specs, and fail when they are gone?

Withdrawn: author choice, [793b48 The conformance check reads this change's specs and fails when they are gone](./793b_decision-log.md#793b48-the-conformance-check-reads-this-changes-specs-and-fails-when-they-are-gone).

## 793c24 How many acceptance tests run at once?

Withdrawn: author choice, [793b49 The acceptance run is capped at four tests at once](./793b_decision-log.md#793b49-the-acceptance-run-is-capped-at-four-tests-at-once).

## 793c25 Does the mod use what a failed or unrecognised command printed?

Withdrawn: author choice, [793b50 The mod uses what the command printed only when it succeeded and printed a block](./793b_decision-log.md#793b50-the-mod-uses-what-the-command-printed-only-when-it-succeeded-and-printed-a-block).

## 793c26 Does the delivery read the records where Claude Code writes them, and does the sweep follow?

Withdrawn: author choice, [793b51 The delivery finds the records where Claude Code writes them](./793b_decision-log.md#793b51-the-delivery-finds-the-records-where-claude-code-writes-them).

## 793c27 Does the delivery's lookup need imptest evidence of its arguments, or a waiver?

Withdrawn: author choice, [793b52 The delivery runs its lookup through an injected function](./793b_decision-log.md#793b52-the-delivery-runs-its-lookup-through-an-injected-function).

## 793c28 Does an acceptance test skip or fail when Claude Code or a login is missing?

Withdrawn: author choice, [793b53 A missing login or Claude Code fails an acceptance test](./793b_decision-log.md#793b53-a-missing-login-or-claude-code-fails-an-acceptance-test).

## 793c29 Does the scenario of a note the agent looked up itself hold a second note?

Withdrawn: author choice, [793b54 Two acceptance scenarios prove more than that nothing arrived](./793b_decision-log.md#793b54-two-acceptance-scenarios-prove-more-than-that-nothing-arrived).

## 793c30 Does a gate parse the mod's JavaScript, and what if node is missing?

Withdrawn: author choice, [793b55 The mod's JavaScript is parsed by a test of the dev integration suite](./793b_decision-log.md#793b55-the-mods-javascript-is-parsed-by-a-test-of-the-dev-integration-suite).

## 793c31 Do the 7 seconds bound the whole hook or the command?

Withdrawn: author choice, [793b56 The 7 seconds bound the whole hook, not the command alone](./793b_decision-log.md#793b56-the-7-seconds-bound-the-whole-hook-not-the-command-alone).

## 793c32 Is the stall between the spawn of the command and its first instruction a risk to the product?

- **Context:** under many parallel acceptance tests the command stalled between its spawn and its first instruction (`prompt.submit settled in 7060.0ms`, no output from the child), so on a loaded machine a [prompt](./793a_product-spec.md#793a1a-terms) is held about 7 seconds with nothing delivered or shown; the cause is not established. With a [vault](../../GLOSSARY.md#vault) of 10 [notes](../../GLOSSARY.md#permanent-note), 5 runs after a warm-up: one delivery takes about 0.19 seconds, what one `engram query` takes, and a record of 300 KB adds about 2 milliseconds.
- **Options:** (a) Record it as an open risk, measure the cost of one delivery so that the groups that add work to the path know what they add to, and optimise nothing. (b) Find and remove the stall in this change. (c) Accept the risk without recording it.
- **Recommendation:** (a).
- **Blocks:** the sizing of the work that [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) and the later groups add to the per-prompt path.
- **Status:** open; default (a) taken in [793b49 The acceptance run is capped at four tests at once](./793b_decision-log.md#793b49-the-acceptance-run-is-capped-at-four-tests-at-once).

## 793c33 Does the promise of automatic memory name a minimum version of Claude Code?

- **Context:** [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) and the README promise [automatic memory](./793a_product-spec.md#793a1a-terms) for every [session](../../GLOSSARY.md#session) after [`engram update`](../../GLOSSARY.md#engram-update), unconditionally, while only a Claude Code that loads mods delivers anything (2.1.302 verified, 2.1.287 the first the earlier attempt found); nothing checks the version (row 13 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)). Raised by the review of the first group.
- **Options:** (a) The promise stays unconditional; a session that cannot load the mod steps aside. (b) The README and the report of `engram update` name the minimum version. (c) `engram update` checks the version and says when it is older.
- **Recommendation:** (a) for this change, as row 13 decided; (b) is cheap if the owner wants it.
- **Blocks:** the wording of the promise in [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) and the README if (b) or (c) is chosen.
- **Status:** open; default (a) taken in [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses).

## 793c34 Does the delivery merge the parent vault, as `engram query` does?

- **Context:** `engram query` merges the [notes](../../GLOSSARY.md#permanent-note) of the parent [vault](../../GLOSSARY.md#vault) (`ENGRAM_PARENT`), and the delivery of the first group looks up the local vault only; [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers the notes `engram query` returns, so a vault that has a parent is delivered less than it asks for.
- **Options:** (a) The local vault only for now, and a named node, [793e6f The delivery merges the parent vault](./793e_implementation-plan.md#793e6f-the-delivery-merges-the-parent-vault), builds the merge later. (b) Build it in the first group.
- **Recommendation:** (a): the review asked that it be recorded and not built.
- **Blocks:** the node [793e6f The delivery merges the parent vault](./793e_implementation-plan.md#793e6f-the-delivery-merges-the-parent-vault).
- **Status:** open; default (a) taken in [793b57 The delivery looks up the local vault only, until a later node](./793b_decision-log.md#793b57-the-delivery-looks-up-the-local-vault-only-until-a-later-node).

## 793c35 Does the mod tell a prompt the user sent from a prompt of another origin?

Withdrawn: author choice, [793b58 The first group delivers for every origin of a prompt](./793b_decision-log.md#793b58-the-first-group-delivers-for-every-origin-of-a-prompt).

## 793c36 Are the restated sentences of 792a2e kept as they are?

Withdrawn: author choice, [793b60 The restated sentences of 792a2e are kept as they are](./793b_decision-log.md#793b60-the-restated-sentences-of-792a2e-are-kept-as-they-are).

## 793c37 Are the fourteen scenarios of the second group the minimum?

Withdrawn: author choice, [793b62 The scenarios of the second group](./793b_decision-log.md#793b62-the-scenarios-of-the-second-group).

## 793c38 Does automatic memory start sessions of its own?

- **Context:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) forbids delivering to or writing from the [sessions](../../GLOSSARY.md#session) [automatic memory](./793a_product-spec.md#793a1a-terms) starts; the plan placed the guard in [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent), the first group that works after a [turn](./793a_product-spec.md#793a1a-terms).
- **Options:** (a) None: the judgement is one call of the mod's model API, which raises no [prompt](./793a_product-spec.md#793a1a-terms) and no turn, so the clause holds by construction and its scenario observes that no other session exists. (b) A headless session of its own for the judgement, with a marker in its environment that makes the mod step aside in it.
- **Recommendation:** (a): (b) starts a second process that loads the user's whole configuration for each turn.
- **Blocks:** the scenario of the clause, [793a2d1b Automatic memory's work starts no session of its own](./793a_product-spec.md#793a2d1-uat-scenarios).
- **Status:** open; default (a) taken in [793b63 A turn is judged by one model call, and no session is started](./793b_decision-log.md#793b63-a-turn-is-judged-by-one-model-call-and-no-session-is-started).

## 793c39 Which origins count as the prompts the user sends?

- **Context:** the first group delivered for every origin ([793b58 The first group delivers for every origin of a prompt](./793b_decision-log.md#793b58-the-first-group-delivers-for-every-origin-of-a-prompt)); this group adds the filter. A headless [session](../../GLOSSARY.md#session)'s [prompt](./793a_product-spec.md#793a1a-terms) has the origin `sdk`, as does a prompt another program sends with the SDK.
- **Options:** (a) `composer`, `bridge`, `slack-ping` and `sdk` (the first default, [793b64 The prompts the user sends](./793b_decision-log.md#793b64-the-prompts-the-user-sends)). (b) `composer`, `bridge` and `slack-ping`, so a headless session gets nothing. (c) As (a) for the delivery, and for the judgement `sdk` only when the user switches it on with `ENGRAM_AUTO_CAPTURE_SDK`.
- **Recommendation:** (c), amended from (a): (b) would leave out the user's own `claude -p` runs, and the tests drive one; a headless run under the real home otherwise writes fiction into the real [vault](../../GLOSSARY.md#vault).
- **Blocks:** the origin filter, [793d4m Telling the prompts the user sends](./793d_technical-spec.md#793d4m-telling-the-prompts-the-user-sends).
- **Status:** open; default (c) taken in [793b84 A headless prompt is judged only when the user asks](./793b_decision-log.md#793b84-a-headless-prompt-is-judged-only-when-the-user-asks).

## 793c40 Is the window of the judgement the right one?

Withdrawn: author choice, [793b65 What the judgement reads, and what it is asked](./793b_decision-log.md#793b65-what-the-judgement-reads-and-what-it-is-asked).

## 793c41 Is the risk of a secret of another shape acceptable?

- **Context:** [793b69 Credentials are redacted by shape, and the rest is a known risk](./793b_decision-log.md#793b69-credentials-are-redacted-by-shape-and-the-rest-is-a-known-risk) redacts the shapes it lists; a password in a sentence of more than six words or without `is` or `:`, a token split over two lines, and the earlier attempt's wrapped and percent-encoded forms are not, so such a secret reaches the model and may reach a [note](../../GLOSSARY.md#permanent-note).
- **Options:** (a) The shapes as built, the rest a known risk. (b) Port the earlier attempt's patterns. (c) Do not write a note from a [turn](./793a_product-spec.md#793a1a-terms) in which the redaction found a credential.
- **Recommendation:** (a), with (c) as the next step if the owner finds the risk too large: (b) adds patterns without making the risk go away, and (c) loses real lessons.
- **Blocks:** the scenario [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios) if (c) is chosen.
- **Status:** open; default (a) taken in [793b69 Credentials are redacted by shape, and the rest is a known risk](./793b_decision-log.md#793b69-credentials-are-redacted-by-shape-and-the-rest-is-a-known-risk).

## 793c42 Should the bound of 2 minutes be tightened?

Withdrawn: author choice, [793b70 The capture reports its own failures, and nothing times it as a whole](./793b_decision-log.md#793b70-the-capture-reports-its-own-failures-and-nothing-times-it-as-a-whole) and [793b85 Each stage of the capture has a bound of its own](./793b_decision-log.md#793b85-each-stage-of-the-capture-has-a-bound-of-its-own); the bound is asked in [793c2 How soon after a turn must automatic memory have finished writing?](#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing).

## 793c43 Should a missing vault end the capture before the model is asked?

Withdrawn: author choice, [793b82 A judgement without a vault or a record fails, and the model is not asked](./793b_decision-log.md#793b82-a-judgement-without-a-vault-or-a-record-fails-and-the-model-is-not-asked), which amends [793b71 The commands of the group auto take no first-use step](./793b_decision-log.md#793b71-the-commands-of-the-group-auto-take-no-first-use-step).

## 793c44 Is the window between a note's file and its embedding a risk?

- **Context:** for the 0.3 seconds the learn path takes to embed a new [note](../../GLOSSARY.md#permanent-note), a lookup in a [vault](../../GLOSSARY.md#vault) whose only note it is fails (`vault has notes but no current-model embeddings`), and so does a delivery ([793b73 The tests of an absence wait for the end of the write, and for a settling time](./793b_decision-log.md#793b73-the-tests-of-an-absence-wait-for-the-end-of-the-write-and-for-a-settling-time)).
- **Options:** (a) A risk to the first [prompt](./793a_product-spec.md#793a1a-terms) after the first note of an empty vault, which steps aside as any failed delivery does. (b) Write the sidecar before the note, in the learn path. (c) Let the lookup skip a note with no embedding.
- **Recommendation:** (a): a prompt typed 0.3 seconds after a note appeared is rare and costs one prompt without its notes, and (b) and (c) change the lookup of the [agent](./793a_product-spec.md#793a1a-terms)'s own use of memory, which [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working) keeps unchanged.
- **Blocks:** [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working) if (b) or (c) is chosen.
- **Status:** open; default (a) taken in [793b73 The tests of an absence wait for the end of the write, and for a settling time](./793b_decision-log.md#793b73-the-tests-of-an-absence-wait-for-the-end-of-the-write-and-for-a-settling-time).

## 793c45 Is the reading of a prompt in the record the rule of the transcript-reader?

Withdrawn: author choice, [793b74 A row the harness wrote is no prompt](./793b_decision-log.md#793b74-a-row-the-harness-wrote-is-no-prompt).

## 793c46 Is 2 seconds for a record of 1 MB the right bound for the judgement?

Withdrawn: author choice, [793b75 The record is read once, and the lines of the window are the only ones redacted](./793b_decision-log.md#793b75-the-record-is-read-once-and-the-lines-of-the-window-are-the-only-ones-redacted).

## 793c47 Is the turn the one that began with the text the mod saw?

Withdrawn: author choice, [793b76 A turn is found by the text it began with](./793b_decision-log.md#793b76-a-turn-is-found-by-the-text-it-began-with).

## 793c48 Is writing a < as &lt; the right defence of the window and the notes?

Withdrawn: author choice, [793b77 The instructions are the system of the request, and a < is written &lt;](./793b_decision-log.md#793b77-the-instructions-are-the-system-of-the-request-and-a--is-written-lt).

## 793c49 Should the promise that a note holds no credential stay absolute?

- **Context:** [793b78 The promise names the forms of credential that are redacted](./793b_decision-log.md#793b78-the-promise-names-the-forms-of-credential-that-are-redacted) narrows the promise of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) to the recognised forms: an absolute promise cannot be proven without the model's cooperation (the live scenario passes with or without the redaction; only the redaction's properties fail for a form it misses).
- **Options:** (a) The promise names the forms, as built; the rest stays the risk of [793c41 Is the risk of a secret of another shape acceptable?](#793c41-is-the-risk-of-a-secret-of-another-shape-acceptable). (b) The promise stays absolute, and no [note](../../GLOSSARY.md#permanent-note) is written from a [turn](./793a_product-spec.md#793a1a-terms) in which the redaction found a credential. (c) The promise stays absolute, and the model checks each note for a secret.
- **Recommendation:** (a) for this change, with (b) as the next step if the owner finds the risk too large: (c) makes the promise depend on the model, and (b) lowers the risk without making the promise provable.
- **Blocks:** the promise of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and its scenario [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios).
- **Status:** open; default (a) taken in [793b78 The promise names the forms of credential that are redacted](./793b_decision-log.md#793b78-the-promise-names-the-forms-of-credential-that-are-redacted).

## 793c50 Is a lesson of a cut reply worth writing?

Withdrawn: author choice, [793b79 A field of a lesson is one line, and a reply that is cut keeps its whole lessons](./793b_decision-log.md#793b79-a-field-of-a-lesson-is-one-line-and-a-reply-that-is-cut-keeps-its-whole-lessons).

## 793c51 Should the delivered line mark a note written automatically?

Withdrawn: author choice, [793b80 The delivered line shows the mark of a note written automatically](./793b_decision-log.md#793b80-the-delivered-line-shows-the-mark-of-a-note-written-automatically).

## 793c52 Is the redaction of a text written as one line the right order?

Withdrawn: author choice, [793b81 A text is written as one line before it is redacted, and cut last](./793b_decision-log.md#793b81-a-text-is-written-as-one-line-before-it-is-redacted-and-cut-last).

## 793c53 Should a judgement without a record fail, and be recorded as a failure?

Withdrawn: author choice, [793b82 A judgement without a vault or a record fails, and the model is not asked](./793b_decision-log.md#793b82-a-judgement-without-a-vault-or-a-record-fails-and-the-model-is-not-asked).

## 793c54 Should a failure that the writing recorded still fail the command?

Withdrawn: author choice, [793b83 A failure that is recorded does not fail the command](./793b_decision-log.md#793b83-a-failure-that-is-recorded-does-not-fail-the-command).

## 793c55 Is 9 passes in 10 runs the bar for a scenario that the judgement of a model decides?

Withdrawn: author choice, [793b86 The acceptance run takes a count](./793b_decision-log.md#793b86-the-acceptance-run-takes-a-count).

## 793c56 Is the bar of the learn skill the bar for a note written automatically?

- **Context:** [793b87 The instructions carry the bar of the learn skill](./793b_decision-log.md#793b87-the-instructions-carry-the-bar-of-the-learn-skill) holds the judgement to the learn [skill](../../GLOSSARY.md#skill)'s bar: a lesson in the form in which it applies again, exact names kept, none from a bare thanks or a routine success. A stricter bar writes fewer [notes](../../GLOSSARY.md#permanent-note) and misses some a user would have kept.
- **Options:** (a) The bar of the learn skill, with the exact names kept. (b) A lower bar: every confirmation is a lesson. (c) The skill's bar to the letter, the principle without the instance.
- **Recommendation:** (a): a note that a thanks wrote is noise in every later lookup, and a note without its names cannot be found or checked.
- **Blocks:** [793a2c1h A thanks that names no approach produces no note](./793a_product-spec.md#793a2c1-uat-scenarios) and [793a3c1a A lesson an earlier note states is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), whose runs depend on the bar.
- **Status:** open; default (a) taken in [793b87 The instructions carry the bar of the learn skill](./793b_decision-log.md#793b87-the-instructions-carry-the-bar-of-the-learn-skill).

## 793c57 Should a reversal of the agent's own claim be written as a fact or as feedback?

- **Context:** the learn [skill](../../GLOSSARY.md#skill) writes a reversal of the [agent](./793a_product-spec.md#793a1a-terms)'s own claim as feedback (behavior, impact, action), the judgement as a fact, the corrected conclusion ([793b65 What the judgement reads, and what it is asked](./793b_decision-log.md#793b65-what-the-judgement-reads-and-what-it-is-asked)); a fact is found by the situation it is about, a feedback by the failure it guards against.
- **Options:** (a) A fact, the corrected conclusion. (b) Feedback, as the skill writes it. (c) Both, a fact and a feedback for the same reversal.
- **Recommendation:** (a): the window of one [turn](./793a_product-spec.md#793a1a-terms) holds the conclusion and what overturned it, and not the cost or the guard that a feedback needs; [793a2c1d The agent's own correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios) asks for a [note](../../GLOSSARY.md#permanent-note) that states what the file says.
- **Blocks:** [793a2c1d The agent's own correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios).
- **Status:** open; default (a) taken in [793b65 What the judgement reads, and what it is asked](./793b_decision-log.md#793b65-what-the-judgement-reads-and-what-it-is-asked).

## 793c58 Is the probe of the judgement the right way to change its instructions?

Withdrawn: author choice, [793b88 The judgement is measured by a probe of its own](./793b_decision-log.md#793b88-the-judgement-is-measured-by-a-probe-of-its-own).

## 793c59 Should an automatic note be offered to the parent vault while the duplicate check ignores it?

- **Context:** [793b89 Each automatic note is offered to the parent vault, and the duplicate check stays local](./793b_decision-log.md#793b89-each-automatic-note-is-offered-to-the-parent-vault-and-the-duplicate-check-stays-local): each [note](../../GLOSSARY.md#permanent-note) written automatically is offered to the parent [vault](../../GLOSSARY.md#vault) while the duplicate check looks in the local vault only, so a lesson the parent holds is offered to it again.
- **Options:** (a) As built: offer upward, check locally, and let the curation of the parent fold the duplicates. (b) Merge the parent vault into the check, in the node that merges it for the delivery. (c) Do not offer automatic notes upward.
- **Recommendation:** (a) until [793e6f The delivery merges the parent vault](./793e_implementation-plan.md#793e6f-the-delivery-merges-the-parent-vault) is built, then (b).
- **Blocks:** the nearest-notes lookup of [793d4g Judging a turn](./793d_technical-spec.md#793d4g-judging-a-turn), with [793e6f The delivery merges the parent vault](./793e_implementation-plan.md#793e6f-the-delivery-merges-the-parent-vault).
- **Status:** open; default (a) taken in [793b89 Each automatic note is offered to the parent vault, and the duplicate check stays local](./793b_decision-log.md#793b89-each-automatic-note-is-offered-to-the-parent-vault-and-the-duplicate-check-stays-local).

## 793c60 Should a request to remember be written when the agent says that it did not save it?

- **Context:** [793b90 A request to remember is a lesson whatever the agent says about saving it](./793b_decision-log.md#793b90-a-request-to-remember-is-a-lesson-whatever-the-agent-says-about-saving-it): a user's request to remember is a lesson whatever the [agent](./793a_product-spec.md#793a1a-terms) answers; an agent may rightly not save it (it keeps no secrets, or was told not to), and the [note](../../GLOSSARY.md#permanent-note) is written anyway, a secret as [redacted].
- **Options:** (a) Write it: the user asked for it, and the note holds no secret. (b) Leave it to the agent's word. (c) Write it, and say in the note that the agent refused.
- **Recommendation:** (a): the user's request is the confirmation, and a refusal of the agent is about its own memory.
- **Blocks:** the scenario [793a2c1i A request to remember is written when the agent says it did not save it](./793a_product-spec.md#793a2c1-uat-scenarios).
- **Status:** open; default (a) taken in [793b90 A request to remember is a lesson whatever the agent says about saving it](./793b_decision-log.md#793b90-a-request-to-remember-is-a-lesson-whatever-the-agent-says-about-saving-it).

## 793c61 Are references the IDs of nodes, or links with their titles?

Withdrawn: answered by the owner's instruction to follow the guides, [793b91 The documents follow the guides and are held to the minimum sufficient](./793b_decision-log.md#793b91-the-documents-follow-the-guides-and-are-held-to-the-minimum-sufficient).

## 793c62 Does 793d5 restate the product scenarios?

Withdrawn: answered by the owner's instruction to follow the guides, [793b91 The documents follow the guides and are held to the minimum sufficient](./793b_decision-log.md#793b91-the-documents-follow-the-guides-and-are-held-to-the-minimum-sufficient).

## 793c63 Is an open question one line, or context, options and recommendation?

Withdrawn: answered by the owner's instruction to follow the guides, [793b91 The documents follow the guides and are held to the minimum sufficient](./793b_decision-log.md#793b91-the-documents-follow-the-guides-and-are-held-to-the-minimum-sufficient).

## 793c64 Is the order of the groups a table or a graph?

- **Context:** the plan guide has the dependency graph carry the status of each node and be the source of sequencing; the first version of [793e4 Depth-first order](./793e_implementation-plan.md#793e4-depth-first-order) held a 90-line chain graph of the order.
- **Options:** (a) The table of [793e4 Depth-first order](./793e_implementation-plan.md#793e4-depth-first-order), with [793e9 Dependency graph](./793e_implementation-plan.md#793e9-dependency-graph) the one dependency graph. (b) A chain graph in [793e4 Depth-first order](./793e_implementation-plan.md#793e4-depth-first-order) as well.
- **Recommendation:** (a): the table gives the order the groups are worked in, one at a time on one branch, and the guide asks for one graph.
- **Blocks:** the guide's rules for the graph at promotion.
- **Status:** open; default (a) taken in [793b96 The order of the groups is a table](./793b_decision-log.md#793b96-the-order-of-the-groups-is-a-table).

## 793c65 Do the catalog rows keep their three commit columns?

- **Context:** the catalog gives each row three commit columns, which the reduction rule would cut as hashes.
- **Options:** (a) Keep the columns. (b) Drop them and leave the commits to git history.
- **Recommendation:** (a): they are the record of step 5 ([793f2 The algorithm](./793f_algorithm.md#793f2-the-algorithm)).
- **Blocks:** the gate of [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates) that every `Done` row has its three commits.
- **Status:** open; default (a) taken in [793b97 The commit columns stay](./793b_decision-log.md#793b97-the-commit-columns-stay).

## 793c66 Does a requirement with two owning components stay whole?

- **Context:** [793d4j Redacting credentials](./793d_technical-spec.md#793d4j-redacting-credentials) is owned by the `turn-judgement` and the `lesson-writing`, and [793d4m Telling the prompts the user sends](./793d_technical-spec.md#793d4m-telling-the-prompts-the-user-sends) by the `prompt-delivery` and the `turn-judgement`; the technical guide has one requirement for each owning component.
- **Options:** (a) Each stays one requirement. (b) One requirement for each owner.
- **Recommendation:** (a): the shapes of a credential and the origins of a [prompt](./793a_product-spec.md#793a1a-terms) are stated once, and two copies would drift.
- **Blocks:** the canonical component files at promotion.
- **Status:** open; default (a) taken in [793b98 Two requirements name two owners](./793b_decision-log.md#793b98-two-requirements-name-two-owners).

## 793c67 Do the default-suite tests that cross the real filesystem keep their tag?

- **Context:** seven rows of [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan) are in the default suite and run over a real temporary directory, which the technical guide reserves for the `integration` tag ([793b99 Default-suite tests that cross the filesystem](./793b_decision-log.md#793b99-default-suite-tests-that-cross-the-filesystem)).
- **Options:** (a) Deferred, not changed in this change. (b) Tag them `integration`. (c) Inject the filesystem so that they are pure.
- **Recommendation:** (a): they pass in `targ test`, which a tag would take them out of.
- **Blocks:** the guide's rule that the default suite holds only pure tests, at promotion.
- **Status:** open; default (a) taken in [793b99 Default-suite tests that cross the filesystem](./793b_decision-log.md#793b99-default-suite-tests-that-cross-the-filesystem).
