# Judged Automatic Memory - Open Questions

A default stands until the owner answers; his answer is recorded in the entry with its date. A question that blocks nothing is an author choice and is not asked: it is a decision in [855b Judged Automatic Memory - Decision Log](./855b_decision-log.md#judged-automatic-memory---decision-log) (`docs/AGENTS.md`).

## 855c1 How long may a judgement or delivery run before it is given up?

- **Context:** the owner dropped the hold on a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) ([855b4 No time bound before a prompt reaches the agent](./855b_decision-log.md#855b4-no-time-bound-before-a-prompt-reaches-the-agent)). A judgement or delivery that never answers would then hold the prompt forever. The host's type definitions give a hook a budget of 10 seconds that does not count a model call or a command, and a model call has no time limit unless one is given ([855b14 A slow step is waited for, a stalled one is not](./855b_decision-log.md#855b14-a-slow-step-is-waited-for-a-stalled-one-is-not)).
- **Options:** (a) Hold until the user interrupts. (b) A liveness bound of 2 minutes, far above any normal judgement and not a latency target, after which the delivery steps aside, delivers nothing and shows a failure line. (c) The host's own ceiling, if it had one.
- **Recommendation:** (b): the host has no ceiling to use, and (a) leaves the user unable to tell a stalled step from a slow one.
- **Blocks:** [855a3c1a A judgement that never answers is stepped aside after 2 minutes](./855a_product-spec.md#855a3c1-uat-scenarios) and the stall clause of [855a3c 793a2d Never in the way](./855a_product-spec.md#855a3c-modified-793a2d-never-in-the-way).
- **Status:** answered (Owner, 2026-10-18): option (b); recorded in [855b14 A slow step is waited for, a stalled one is not](./855b_decision-log.md#855b14-a-slow-step-is-waited-for-a-stalled-one-is-not).

## 855c2 Which tier writes the situation phrases?

- **Context:** the owner's step table has one call for the moment and the phrases; his tiers put the phrases among the content judgements ([855b5 Two tiers of judgement](./855b_decision-log.md#855b5-two-tiers-of-judgement)). The phrases decide what the lookup can find.
- **Options:** (a) The cheap-tier moment call writes them: one call, the table's shape. (b) A mid-tier call writes them after the moment call says the depth: one more call at a recall moment only. (c) The mid tier judges the moment as well.
- **Recommendation:** (b): it follows the tiers the owner named; (a) is one setting away ([855b26 The moment judgement gives the depth, a mid-tier call the phrases](./855b_decision-log.md#855b26-the-moment-judgement-gives-the-depth-a-mid-tier-call-the-phrases)).
- **Blocks:** [855d4a Judging the moment](./855d_technical-spec.md#855d4a-added-judging-the-moment) and [855d4v Phrasing the lookup](./855d_technical-spec.md#855d4v-added-phrasing-the-lookup).
- **Status:** open; (b) stands as the author's default.

## 855c3 Is every tool call judged, and may the chains run at once?

- **Context:** the owner wants recall as often as possible ([855b23 Recall runs at the points the host offers](./855b_decision-log.md#855b23-recall-runs-at-the-points-the-host-offers)); each judged tool call adds a cheap-tier call, and the host may run the events of parallel calls at once.
- **Options:** (a) Judge every call of the main loop, one chain at a time for a [session](../../GLOSSARY.md#session). (b) Judge every call, run the chains at once, and drop a duplicate delivery through a set of delivered IDs the mod holds. (c) Judge only some tools.
- **Recommendation:** (b): no call waits for the chains of others; the time added is measured ([855b30 Every tool call of the main loop is judged, the chains run at once](./855b_decision-log.md#855b30-every-tool-call-of-the-main-loop-is-judged-the-chains-run-at-once)).
- **Blocks:** [855d2c Mid-turn recall flow](./855d_technical-spec.md#855d2c-added-mid-turn-recall-flow).
- **Status:** open; (b) stands as the author's default.

## 855c4 Does the before-tool cue promise the note before the call or before the next action?

- **Context:** a context put in front of a tool call reaches the model with the call's result, at its next request ([855d1a Verified host facts](./855d_technical-spec.md#855d1a-verified-host-facts)). The promise of [855a2f 855a2e1c A note arrives before the agent starts building a new approach](./855a_product-spec.md#855a2f-modified-855a2e1c-a-note-arrives-before-the-agent-starts-building-a-new-approach) said before the edit. The order is measured: the block follows the tool's result and precedes the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s next message ([855b51 A context put at a tool call reaches the model after the tool's result](./855b_decision-log.md#855b51-a-context-put-at-a-tool-call-reaches-the-model-after-the-tools-result)).
- **Options:** (a) Promise the [note](../../GLOSSARY.md#permanent-note) before the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s next action, and measure the timing. (b) Deny the call with the note as the reason, so that the agent reads it before the call: it changes the agent's flow, as every call that meets a note fails once. (c) Keep the old wording.
- **Recommendation:** (a), with (b) rejected ([855b37 The before-tool cue promises the next action](./855b_decision-log.md#855b37-the-before-tool-cue-promises-the-next-action)); (c) the host cannot keep.
- **Blocks:** [855a2f 855a2e1c A note arrives before the agent starts building a new approach](./855a_product-spec.md#855a2f-modified-855a2e1c-a-note-arrives-before-the-agent-starts-building-a-new-approach).
- **Status:** open; (a) stands as the author's default.

## 855c5 Is the 2 minutes a limit for each wait or for the whole flow?

- **Context:** the owner's liveness bound is 2 minutes ([855b14 A slow step is waited for, a stalled one is not](./855b_decision-log.md#855b14-a-slow-step-is-waited-for-a-stalled-one-is-not)); a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s chain has several waits.
- **Options:** (a) One envelope of 2 minutes for the flow, each wait given what is left. (b) 2 minutes for each wait; the flow is stalled when any one wait passes it. (c) Rows for each wait.
- **Recommendation:** (b): a chain of waits that are each only slow is not stalled ([855b27 Each wait has its own 2 minutes](./855b_decision-log.md#855b27-each-wait-has-its-own-2-minutes)); [855a3c1a A judgement that never answers is stepped aside after 2 minutes](./855a_product-spec.md#855a3c1-uat-scenarios) holds as written.
- **Blocks:** [855d2a 793d2a Prompt delivery flow](./855d_technical-spec.md#855d2a-modified-793d2a-prompt-delivery-flow).
- **Status:** open; (b) stands as the author's default.

Step 3, the technical specification ([855d Judged Automatic Memory - Technical Spec](./855d_technical-spec.md#judged-automatic-memory---technical-spec)), raised the four questions above at the fix pass of its review; its other author defaults are [855b24 The three cues of a turn ride three tool events](./855b_decision-log.md#855b24-the-three-cues-of-a-turn-ride-three-tool-events) to [855b40 A ledger of what was delivered, if the transcript lacks it](./855b_decision-log.md#855b40-a-ledger-of-what-was-delivered-if-the-transcript-lacks-it), for the owner to overrule at the walkthrough.

## 855c6 Which date does a note carry?

- **Context:** a candidate line carries the date the [note](../../GLOSSARY.md#permanent-note) was written or last amended, so that the model can tell the newer of two conflicting candidates ([855b32 Candidates are judged one by one, and the session's own transcript is not a candidate](./855b_decision-log.md#855b32-candidates-are-judged-one-by-one-and-the-sessions-own-transcript-is-not-a-candidate), [855d4b Looking up the candidates for the phrases](./855d_technical-spec.md#855d4b-added-looking-up-the-candidates-for-the-phrases)). No field of a note records an amendment: `created` is written once, and the use date is in the sidecar.
- **Options:** (a) The `created` date: an amended note looks as old as it was written, so a revised note can lose to a later note it predates. (b) A stamp that `engram amend` writes when it changes a note's content, read as the date: the promise holds as written, and the amendment component, its specification and the exchange hash change. (c) The date of the file, which the lookup does not return.
- **Recommendation:** (a) for now: the notes a revision makes are rare against the notes written, and (b) is a change of its own to a component outside this change ([855b56 The date of a note is its created date, and the date of a chunk the time the index holds](./855b_decision-log.md#855b56-the-date-of-a-note-is-its-created-date-and-the-date-of-a-chunk-the-time-the-index-holds)).
- **Blocks:** [855d4b1g The request is for the mid tier and holds the verdicts, the window and the dated candidates](./855d_technical-spec.md#855d4b1-scenarios), in its reading of the date of a note.
- **Status:** answered by default (Author, 2026-10-18): (a) for a note, and for a chunk the time the index holds, which is the time of the last row of its [session](../../GLOSSARY.md#session) for a [transcript](../../GLOSSARY.md#transcript); the owner may reopen it ([855b56 The date of a note is its created date, and the date of a chunk the time the index holds](./855b_decision-log.md#855b56-the-date-of-a-note-is-its-created-date-and-the-date-of-a-chunk-the-time-the-index-holds)).

## 855c7 Does the cap on the candidates keep a floor of passages?

- **Context:** the lookup returns [notes](../../GLOSSARY.md#permanent-note) and chunks by score, and the judgement holds 30 candidates at most. Without a floor a [vault](../../GLOSSARY.md#vault) with many notes fills the cap with notes and the passages, the evidence of a deep recall, go ([855b59 The cap keeps a floor of passages, and the budget ends the list at the first that does not fit](./855b_decision-log.md#855b59-the-cap-keeps-a-floor-of-passages-and-the-budget-ends-the-list-at-the-first-that-does-not-fit)).
- **Options:** (a) No floor: the top 30 by score. (b) A floor of 10 of the 30 for passages, when so many matched, in the idea of the lookup's floor of notes. (c) A different split, or a floor for notes as well.
- **Recommendation:** (b): the floor costs nothing when no passage matched and keeps the evidence of a deep recall otherwise.
- **Blocks:** [855d4b1m A cap on the count keeps a floor of passages](./855d_technical-spec.md#855d4b1-scenarios).
- **Status:** answered by default (Author, 2026-10-18): option (b); the owner may reopen it.

## 855c8 Is every tool call judged at the three cues, or only at some?

- **Context:** the owner's North Star says to fire as often as possible ([855b2 North Star](./855b_decision-log.md#855b2-north-star)), and the build judges the call about to run (N, the cue `before-tool`), the call that failed (F) and the batch that resolved (B): a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) of N calls in one batch makes N+B cheap-tier model calls and N+F+B when calls fail, each of about 3,000 characters (793a2c1d, one call: before-tool 3,234, batch-done 3,377, turn-end 2,525, [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) 3,078; characters, not tokens, which group g measures). A context put at a tool call and the context of the batch land at the same model request ([855b51 A context put at a tool call reaches the model after the tool's result](./855b_decision-log.md#855b51-a-context-put-at-a-tool-call-reaches-the-model-after-the-tools-result)), so a judgement per call and a judgement per batch pay twice for one delivery point.
- **Options:** (a) As built: N+F+B. (b) Failure and batch-done only. (c) Batch-done only, with the errors in its window. (d) As built, skipping the read-only tools for before-tool.
- **Recommendation:** (a): the owner's rule is to fire as often as possible, and the cost is measured by group g before it is weighed against the delivery each cue adds.
- **Blocks:** [855d2c Mid-turn recall flow](./855d_technical-spec.md#855d2c-added-mid-turn-recall-flow).
- **Status:** open; (a) stands as the author's default.
