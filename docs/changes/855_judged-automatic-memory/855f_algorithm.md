# Judged Automatic Memory - Algorithm

How change #855 is produced; change context, deleted at promotion. The owner's words are quoted exactly (Joe, 2026-10-18, [855b2 North Star](./855b_decision-log.md#855b2-north-star)).

## 855f1 North Star

> If haiku decides this is a reasonable moment for recall, I want sonnet to literally run recall and inject the resulting memory content. Similarly, if it's a reasonable moment for learn, I want learn.

Everything in this change serves it. It follows [793a Automatic Memory - Product Spec](../793_automatic-memory/793a_product-spec.md#automatic-memory---product-spec), which delivers every match of a query on the raw [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms).

## 855f2 The algorithm

The five steps, depth first and the delegation rule are those of [793f2 The algorithm](../793_automatic-memory/793f_algorithm.md#793f2-the-algorithm), [793f3 Depth first](../793_automatic-memory/793f_algorithm.md#793f3-depth-first) and [793f4 Delegation](../793_automatic-memory/793f_algorithm.md#793f4-delegation); the departures of [793f6 Departures from the guides](../793_automatic-memory/793f_algorithm.md#793f6-departures-from-the-guides) and the test isolation of [793f7 Test isolation](../793_automatic-memory/793f_algorithm.md#793f7-test-isolation) apply unchanged. Files: [855a](./855a_product-spec.md#judged-automatic-memory---product-spec) holds steps 1 and 2; the technical specification (steps 3 and 4) and the implementation plan (step 5) are written later.

## 855f3 What differs

- The change layers on the unmerged change 793: its product promises that this change replaces are Modified operations of 855a, as step 2 of [793f5 How this change applies it](../793_automatic-memory/793f_algorithm.md#793f5-how-this-change-applies-it) says ([855b9 The change layers on 793 by Modified operations](./855b_decision-log.md#855b9-the-change-layers-on-793-by-modified-operations)).
- Steps 1 and 2 are done first for the whole change ([855b3 Steps 1 and 2 only](./855b_decision-log.md#855b3-steps-1-and-2-only)); no technical document exists yet.
- Before any step of the technical specification relies on the mod's model call or on adding context, it is read in the host's type definitions ([855b21 The mod API is verified before a step relies on it](./855b_decision-log.md#855b21-the-mod-api-is-verified-before-a-step-relies-on-it)).
