# Route price table — $/token rates by model

Maintained data, not code. The route mini-report renderer looks up a dispatch's model here to
convert `subagent_tokens` into a dollar estimate. **Cheap/mid/deep are cold-start tier labels**
(see SKILL.md `## Cold-start priors`); a model resolves to whichever tier the roster assigns it
at dispatch time — this table prices models, not tiers, since the same model can serve different
tiers on different rosters.

Rates are $ per 1,000,000 tokens (Anthropic first-party Claude API list rates). Cache write is
priced relative to input at two TTLs; cache read is a fixed ~0.1x input across every model listed
(no per-model override is documented). When a dispatch's model is not a key in this table, the
mini-report MUST fall back to unit-labeled raw tokens — never compute or guess a $ figure.

| model | input $/M | output $/M | cache write 5m $/M | cache write 1h $/M | cache read $/M |
| --- | --- | --- | --- | --- | --- |
| claude-haiku-4-5 | 1.00 | 5.00 | 1.25 | 2.00 | 0.10 |
| claude-sonnet-5 | 3.00 (2.00 intro thru 2026-08-31) | 15.00 (10.00 intro) | 3.75 | 6.00 | 0.30 |
| claude-sonnet-4-6 | 3.00 | 15.00 | 3.75 | 6.00 | 0.30 |
| claude-opus-5 | 5.00 | 25.00 | 6.25 | 10.00 | 0.50 |
| claude-opus-4-8 | 5.00 | 25.00 | 6.25 | 10.00 | 0.50 |
| claude-fable-5 | 10.00 | 50.00 | 12.50 | 20.00 | 1.00 |
| claude-mythos-5 | 10.00 | 50.00 | 12.50 | 20.00 | 1.00 |

Source: `claude-api` skill's cached model/pricing table (`shared/live-sources.md` → Pricing URL is
the live source of truth; re-sync this table from there, or from a fresh `claude-api` skill
lookup, whenever a rate looks stale — do not hand-edit a number without checking the source).

## How the mini-report uses this table

1. Look up the dispatch's model (the "model (roster @ dispatch)" field, e.g. `cheap (haiku)` →
   resolve to the concrete model ID `claude-haiku-4-5`) as an exact-match row key.
2. **Match found:** compute a blended $ estimate from `subagent_tokens`. If the harness's usage
   block splits input/output/cache tokens, price each split against its own column and sum; if it
   only reports a single blended token count, estimate using the input rate as the conservative
   floor and label it clearly as a blended estimate (e.g. `45,231 tok (~$0.68 @ opus, blended
   input rate)`).
3. **No match found** (model not in this table, or the harness doesn't expose a model string at
   all — see SKILL.md's non-Claude-Code harness handling): show the raw unit-labeled token count.
   Never fabricate a $ figure for an unmapped model.
4. A model on a non-Anthropic platform (Bedrock, Vertex, Foundry) is priced differently by that
   platform — treat it as "no match" here even if the bare model ID matches a row, and note the
   platform in the mini-report (`n/a — Bedrock pricing not in this table`).
