# Judged Automatic Memory - Open Questions

A default stands until the owner answers; his answer is recorded in the entry with its date. A question that blocks nothing is an author choice and is not asked: it is a decision in [855b Judged Automatic Memory - Decision Log](./855b_decision-log.md#judged-automatic-memory---decision-log) (`docs/AGENTS.md`).

## 855c1 Does the 2-minute bound after a turn hold for a judged capture?

- **Context:** [793c2 How soon after a turn must automatic memory have finished writing?](../793_automatic-memory/793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing) set 2 minutes for a capture of one judgement call and a write (115 seconds of bounds). The judged capture makes a cheap-tier call for the moment, a mid-tier call for the lessons and one for placement and supersession, each of unknown duration. The owner's "time is an optimization for later" was about the hold before a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) ([855b4 No time bound before a prompt reaches the agent](./855b_decision-log.md#855b4-no-time-bound-before-a-prompt-reaches-the-agent)).
- **Options:** (a) Keep 2 minutes. (b) A longer bound, such as 5 minutes. (c) No bound, as for the hold.
- **Recommendation:** (a), until the three calls are measured; a promise without a number cannot be verified, and (c) would leave the user unable to tell a slow write from a lost one.
- **Blocks:** the timing of the scenarios of [855a2b Lessons are written as the learn skill writes them](./855a_product-spec.md#855a2b-lessons-are-written-as-the-learn-skill-writes-them), [855a2c Lessons are placed as the learn skill places them](./855a_product-spec.md#855a2c-lessons-are-placed-as-the-learn-skill-places-them) and [855a2d Lessons that correct a note say so](./855a_product-spec.md#855a2d-lessons-that-correct-a-note-say-so), and of [793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory).
- **Status:** open; default (a) taken in [855b18 The 2-minute promises stay](./855b_decision-log.md#855b18-the-2-minute-promises-stay).

## 855c2 Which notes may a written note be placed under?

- **Context:** the owner's step table places a lesson "against the nearest [notes](../../GLOSSARY.md#permanent-note)"; the [learn skill](../../GLOSSARY.md#learn-skill) places it only against notes written or recalled earlier in the [session](../../GLOSSARY.md#session) and forbids a search of the rest of the [vault](../../GLOSSARY.md#vault) ([792a15h Each note is placed by an ordered test](../../specs/product/learn.md#792a15h-each-note-is-placed-by-an-ordered-test)). The North Star says automatic learn is learn.
- **Options:** (a) The skill's rule: notes written, delivered or looked up earlier in the same session. (b) The nearest notes of the whole vault.
- **Recommendation:** (a): it is the skill's rule, and (b) would place a note under one the session never saw, which is the search the skill forbids.
- **Blocks:** [855a2c Lessons are placed as the learn skill places them](./855a_product-spec.md#855a2c-lessons-are-placed-as-the-learn-skill-places-them) and its scenarios.
- **Status:** open; default (a) taken in [855b16 Placement is the learn skill's ordered test](./855b_decision-log.md#855b16-placement-is-the-learn-skills-ordered-test).

## 855c3 What is the root key of this change?

- **Context:** the lifecycle guide roots every change at its external issue key ([Issue-Rooted Change IDs](../../AGENTS.md#issue-rooted-change-ids)). This change has no issue; 797 is the next free number under `docs/changes/`, and GitHub issue #797 is an unrelated bug. A departure from the guide is the owner's to decide.
- **Options:** (a) Keep 797 and file the issue later. (b) File an issue for this change now and rename the directory and every ID to its number. (c) Keep 797 as a departure from the guide.
- **Recommendation:** (b): IDs are immutable once cited, and renaming costs least before anything outside this directory cites them.
- **Blocks:** every ID of this change.
- **Status:** open; default (a) taken in [855b22 The change number is the next free one](./855b_decision-log.md#855b22-the-change-number-is-the-next-free-one).
