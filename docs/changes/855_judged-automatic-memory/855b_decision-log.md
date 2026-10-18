# Judged Automatic Memory - Decision Log

Decisions made while producing #855, for the owner to validate. Decided by: O = owner (Joe, dated); A = author, a derivation or default the owner may override ("A (`855cN`)" is the default of that question); R = reviewer, a review round's own outcome, accepted by the author. Questions for the owner are in [855c Judged Automatic Memory - Open Questions](./855c_open-questions.md#judged-automatic-memory---open-questions).

## 855b1 The follow-on change and its baseline

Open the next change on top of the unmerged branch feat/automatic-memory: branch feat/judged-automatic-memory, from its tip 8efa196a, pushed and gated green on 2026-10-18 (`targ check-full` 12 of 12, `targ test` green, acceptance 48 of 48). That is the baseline; the implementation plan, when it is created, records it. O 2026-10-18.

## 855b2 North Star

The owner's words, unchanged: "If haiku decides this is a reasonable moment for recall, I want sonnet to literally run recall and inject the resulting memory content. Similarly, if it's a reasonable moment for learn, I want learn." [Automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) makes the judgements the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s recall and [learn skills](../../GLOSSARY.md#learn-skill) make, so that what is injected is what an intentional recall would have surfaced and what is written is what an intentional learn would have written; today's delivery is one query on the raw [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) with no judgement, which the owner judges pollutes the agent's context ([855f1 North Star](./855f_algorithm.md#855f1-north-star)). O 2026-10-18.

## 855b3 Steps 1 and 2 only

This stage follows steps 1 and 2 of [793f2 The algorithm](../793_automatic-memory/793f_algorithm.md#793f2-the-algorithm): the product requirements, the minimal scenarios, the decisions and the questions, and the modifications they force on 793's documents. No technical document, plan, code or test. O 2026-10-18.

## 855b4 No time bound before a prompt reaches the agent

The 8-second hold is dropped: nothing bounds what runs before a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) reaches the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms); "time is an optimization for later". It answers [793c1 How long may a prompt be held?](../793_automatic-memory/793c_open-questions.md#793c1-how-long-may-a-prompt-be-held) and overrides [793b17 A prompt is held for at most 8 seconds](../793_automatic-memory/793b_decision-log.md#793b17-a-prompt-is-held-for-at-most-8-seconds), and with it the 7 seconds of [793b46 The command is ended at 7 seconds](../793_automatic-memory/793b_decision-log.md#793b46-the-command-is-ended-at-7-seconds), [793b56 The 7 seconds bound the whole hook, not the command alone](../793_automatic-memory/793b_decision-log.md#793b56-the-7-seconds-bound-the-whole-hook-not-the-command-alone) and [793c21 How long may the mod's command run, short of the hold?](../793_automatic-memory/793c_open-questions.md#793c21-how-long-may-the-mods-command-run-short-of-the-hold). The promise changes as [855a3c 793a2d Never in the way](./855a_product-spec.md#855a3c-modified-793a2d-never-in-the-way) says. O 2026-10-18.

## 855b5 Two tiers of judgement

The moment judgement (is this [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) a recall moment, is this [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) a learn moment) is a cheap-tier model call; the content judgements (situation phrases, candidate selection, lesson crystallization with placement and supersession) are a mid-tier call. The model names are the current roster for a tier and not part of a promise, so [855a](./855a_product-spec.md#judged-automatic-memory---product-spec) names none. O 2026-10-18.

## 855b6 The automatic channel follows the skills

[Automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) follows the algorithms of the [recall skill](../../GLOSSARY.md#recall-skill) and the [learn skill](../../GLOSSARY.md#learn-skill); the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s own skills and guidance stay unchanged, and the agent-side recall gate #665 stays closed. O 2026-10-18.

## 855b7 The product spec holds promises only

[855a](./855a_product-spec.md#judged-automatic-memory---product-spec) is the owner's intent as promises a user can verify from a [session](../../GLOSSARY.md#session): what arrives, what does not, what is written, what the user sees. The mechanism and the step split belong to the technical specification. Documents are held to the minimum sufficient, within the guides. O 2026-10-18.

## 855b8 The starting step table

The owner saw and accepted this table as the starting point for the technical specification; it is not in [855a](./855a_product-spec.md#judged-automatic-memory---product-spec). Recall: moment and situation phrases (mod, one call); the query with the phrases (engine); judge candidates covered, near or absent and resolve by recency (mod, one call); activate the matched, fetch the chosen bodies, drop what the [session](../../GLOSSARY.md#session) holds (engine); inject (mod). Learn after the [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms): the turn window (engine); judge the lessons of the four kinds (mod); placement and supersession against the nearest [notes](../../GLOSSARY.md#permanent-note) (mod, one call); redact, drop duplicates, write (engine). O 2026-10-18.

## 855b9 The change layers on 793 by Modified operations

793 is unmerged, so its promises are not yet canonical. This change still states each promise it replaces as a Modified operation with the target's complete replacement text ([855a3 Automatic memory of 793](./855a_product-spec.md#855a3-modified-automatic-memory-of-793)), so that promotion applies 793 and then 855 in order. 793's own documents change only where an owner's answer is recorded ([793c1 How long may a prompt be held?](../793_automatic-memory/793c_open-questions.md#793c1-how-long-may-a-prompt-be-held), [793c21 How long may the mod's command run, short of the hold?](../793_automatic-memory/793c_open-questions.md#793c21-how-long-may-the-mods-command-run-short-of-the-hold), [793c71 How is the work after a turn kept from holding the next prompt?](../793_automatic-memory/793c_open-questions.md#793c71-how-is-the-work-after-a-turn-kept-from-holding-the-next-prompt)). A.

## 855b10 A recall moment is the recall skill's own rule

A recall moment is a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) for which the [recall skill](../../GLOSSARY.md#recall-skill)'s rule says to run it (any request that might entail more than a single tool call or more than quick, shallow thinking), and nothing narrower or broader ([855b6 The automatic channel follows the skills](#855b6-the-automatic-channel-follows-the-skills)). The 793 scenarios that send a one-line question about when the staging database is reset are taken to meet it, since the answer lies in the user's environment and not in the prompt; if the first run of the judgement refuses them, their prompts are reworded by a Modified operation. A.

## 855b11 A note applies when the recall skill draws on it

What arrives is what the [recall skill](../../GLOSSARY.md#recall-skill), run for the [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) on the same [vault](../../GLOSSARY.md#vault), draws on: the [notes](../../GLOSSARY.md#permanent-note) it judges covered or near, the newer of two that conflict. Chunks, recent activity and explore-sampled notes are not delivered ([793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) already delivers notes only). The user checks it by running the skill ([855a3b1d The notes that arrive are the notes the recall skill draws on](./855a_product-spec.md#855a3b1-uat-scenarios)). A.

## 855b12 Delivery is the read side of recall

The automatic channel runs the read side of the [recall skill](../../GLOSSARY.md#recall-skill), as its `glance` mode does: no [note](../../GLOSSARY.md#permanent-note) is crystallized or amended and no synthesis is persisted ([855a2a Delivering writes nothing](./855a_product-spec.md#855a2a-delivering-writes-nothing)); the writing half is the learn channel. This follows the owner's step table, which has no write step on the recall side. A.

## 855b13 A prompt that is not a recall moment leaves no line

[793a2e The user can see what it did](../793_automatic-memory/793a_product-spec.md#793a2e-the-user-can-see-what-it-did) shows what was delivered, written, left out and failed. A [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) judged not a recall moment is none of these, so it leaves no line; the user sees that nothing arrived. A.

## 855b14 A slow step is not a failure

With no time bound, only a step that fails makes [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) step aside; a step that is slow is waited for ([855a3c 793a2d Never in the way](./855a_product-spec.md#855a3c-modified-793a2d-never-in-the-way)). A step that never answers holds the [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) until the user interrupts it; that follows the owner's rule, and he may want a bound back. A.

## 855b15 Lessons stay of the two kinds 793 writes

The [learn skill](../../GLOSSARY.md#learn-skill) also writes a runbook [note](../../GLOSSARY.md#permanent-note) for a confirmed multi-step procedure. 793 writes facts and feedback only, the owner's table has no runbook step, and this change keeps the two kinds. A.

## 855b16 Placement is the learn skill's ordered test

A written [note](../../GLOSSARY.md#permanent-note) is placed by [792a15h Each note is placed by an ordered test](../../specs/product/learn.md#792a15h-each-note-is-placed-by-an-ordered-test), over the notes written, delivered or looked up earlier in the same [session](../../GLOSSARY.md#session), the [skill](../../GLOSSARY.md#skill)'s "written or recalled". The owner's table says "against the nearest notes", which can read as the whole [vault](../../GLOSSARY.md#vault); the skill forbids that search. A (`855c2`).

## 855b17 Supersession is the learn skill's

A lesson that corrects, narrows or refutes a [note](../../GLOSSARY.md#permanent-note) makes the new note name it with its type, as the [learn skill](../../GLOSSARY.md#learn-skill)'s handoff carries it; the derived inverse, the older note's marking, is the binary's ([`--supersedes`](../../GLOSSARY.md#--supersedes)). A.

## 855b18 The 2-minute promises stay

[793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) keep their 2 minutes after a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), and the scenarios of [855a](./855a_product-spec.md#judged-automatic-memory---product-spec) use them. The owner's rule drops the hold only; a capture of three model calls may not fit ([793c2 How soon after a turn must automatic memory have finished writing?](../793_automatic-memory/793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)). A (`855c1`).

## 855b19 The 793 scenarios that stay, and those that change

Every scenario of 793a stays except four, which name the old behavior and are replaced by Modified operations: 793a2d1a and 793a2d1c (the 8 seconds), 793a3a1a and 793a2e1g ([notes](../../GLOSSARY.md#permanent-note) that "bear on" a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms)). 793a2b1a to 793a2b1d stay under the modified 793a2b. A.

## 855b20 The 793 technical nodes the decision forces

The technical nodes that derive from the 8 seconds are replaced when the technical specification is written, in whole, because the judged flow replaces [793d2a Prompt delivery flow](../793_automatic-memory/793d_technical-spec.md#793d2a-prompt-delivery-flow): the requirement 793d2a (the 7-second bound and its failure report), its scenarios 793d2a1d (a slow command is ended), 793d2a1g (a [session](../../GLOSSARY.md#session) ID not given in 7 seconds) and 793d2a1j (a delivery that does not finish within its bound), the mappings [793d5e Mapping of 793a2d1a](../793_automatic-memory/793d_technical-spec.md#793d5e-mapping-of-793a2d1a) and [793d5zc Mapping of 793a2d1c](../793_automatic-memory/793d_technical-spec.md#793d5zc-mapping-of-793a2d1c), and the operational and risk lines of [793d6 Recommended impact](../793_automatic-memory/793d_technical-spec.md#793d6-recommended-impact). No replacement text is written now: it is step 3. A.

## 855b21 The mod API is verified before a step relies on it

Whether context can be added only at the [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s event, and what the model call accepts, are read in the host's type definitions before any step of the technical specification relies on them; the owner's table is a starting point and not a verified mechanism. O 2026-10-18.

## 855b22 The change number is the next free one

797 is the next free number under `docs/changes/` (793, 795 and 796 are taken; the specs already carry 794 IDs), taken as the brief says; the guide asks for an issue key as the root ([855c3 What is the root key of this change?](./855c_open-questions.md#855c3-what-is-the-root-key-of-this-change)). A (`855c3`).
