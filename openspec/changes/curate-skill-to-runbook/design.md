## Context

`please-skill-to-runbook` (archived 2026-09-21) is the template: validated fixture, conversion-fidelity
report, retrieval check with phrases real agents generate (vault note 1039), three-part retirement gate
(vault note 1031a). `runbook-lexical-triggers` (shipped) adds `triggers:` on runbooks and `engram query
--text`; `runbook-trigger-whole-word` (commit b8e955a8) makes matching whole-word, so the bare word `curate`
no longer fires inside "accurate" or "curated".

`curate/SKILL.md` today: 83 lines, 5,724 bytes; sections: intro (what a pending offer is, host-local only),
Step 1 (scan files for `pending: true` because `engram query` excludes offers by design), Step 2
(covered/near/absent table with the exact `engram amend` action per outcome, judge content not similarity,
never hand off to `write-memory`), Step 3 (why covered and near both discard), Step 4 (`engram check`,
verify), a 7-row red-flags table. Its `description` lists four surfacing signals plus the explicit ask.

The four surfacing signals map to code: `engram query` payload `pending_offers: true` (a bare boolean,
`internal/cli/query.go`), `engram update` notice (`pendingOfferUpdateNotice`), write-path log nudge
(`pendingOfferWriteNudge`, after learn/amend/resituate), and "engram serve has just accepted a served
write" (an agent-side observation, not a printed line).

## Goals / Non-Goals

**Goals:** one runbook that reproduces the skill's behavior under the shim only; the automatic paths keep
working after retirement; the spec matches the shipped behavior.

**Non-Goals:** changing curation behavior; `recall`/`learn`/`write-memory` (#760); the openspec skills
(#761); editing the shim in this change; running paid evals in the conversion stage.

## Decisions

**D1. One top runbook, `curate`.** 5,724 bytes fits a single retrieved item (please needed a set because
it is ~20x larger), so no sub-runbooks. `red_flags` derive from the 7-row table (at most 1200 bytes; the
rows are short and specific, verify with the real `redflags_truncation.go` budget). Wikilinks: the runbook
does not ask the agent to cite vault notes, so no wikilink syntax note is needed and the body carries no
`[[...]]` links (nothing to link: `write-memory` and `recall` are named as things NOT to hand off to or as
the source of the judgment, in prose).

**D2. `situation` is process-shaped, not test-fitted.** Draft: "reviewing pending offers in a vault and
deciding, for each, whether existing notes already cover it, nearly cover it, or do not". It names no
beekeeping, no offer number and no example from the eval (please's lesson: a situation carrying the eval
task's own example is test-fitting). The exact final text is chosen in the fixture task and recorded in
the fidelity report and retrieval check.

**D3. `triggers`: `curate`, `/curate`, `pending offers`, `pending offer`.** Whole-word matching is in the
repo (b8e955a8); the installed binary must be verified to include it before relying on it (the deployed
binary at `~/go/bin/engram` predates the commit; the check builds a scratch binary). Over-fire review:
`curate` fires on "curate a playlist"/"curate the docs" (a trigger hit is a candidate only, the agent
judges it against the situation; documented in the glossary as the accepted trade for a deliberate
invocation word); `curated`/`accurate` do not fire. `/curate` is redundant with `curate` under whole-word
matching (the slash is a boundary) but is kept for parity with please's `/please` + `please` pair and to
survive a future matcher change; it costs nothing. `pending offers` is specific (engram vocabulary).
`pending offer` (singular) is separate because whole-word matching rejects the `s` continuation. Rejected:
`offers`, `review`, `triage` (common words), `awaiting curation` (already reachable via `curate`).

**D4. Automatic firing rides on the notice text.** A skill was offered by its `description`; a runbook is
reached by `engram query`. So the two notices carry the trigger query: e.g. `engram query --text "curate
pending offers" --phrase "reviewing pending offers in a vault and judging each against existing notes"`.
`--text` contains the trigger words (`curate`, `pending offers`), so the runbook surfaces with `trigger`
provenance and ranks first; the `--phrase` is the semantic fallback and is process-shaped. The query
payload's `pending_offers: true` flag gets a companion hint string (D9). The served-write-accepted signal is not
a printed line and is dropped as a firing signal (it only ever mattered to an agent that operated the
server, and the two notices already cover the next `update` or write). "curation skill" in help text and
comments becomes "curation runbook" (help text for `--discard` and `--clear-pending`) so a reader is not
sent looking for a skill after retirement.

**D5. Spec reconcile, verified against code and history.** Findings:
- `engram amend --discard` deletes the target note and its sidecar (`discardNote`, amend.go);
  `--clear-pending` sets `pending: false` and keeps the file. Their doc comments call discard the
  "covered" outcome.
- The main spec's curation requirement says a covered offer is discarded, a near offer is folded into the
  existing note, an absent offer is accepted, "and in every case the pending-offer marker SHALL be cleared
  once curated". A discarded note has no marker, so "every case" cannot apply to covered. The near
  scenario says the existing note is amended "and the pending offer's marker is cleared", which, read
  literally, KEEPS the offer as a second live note. The archived `serve-vault-api` design diagram says
  covered: "discard, clear marker"; near: "engram amend existing note, clear marker"; absent: "engram
  learn as normal note, clear marker" (its "learn" for absent is also superseded: the skill's absent action
  is `--clear-pending`, never a new learn).
- The skill (authored later, task 9.1 of that change) states the opposite for near, with a written reason
  (Step 3: leaving the offer live after folding it elsewhere creates a second live redundant note, exactly
  the duplication curation exists to prevent), and the eval's end states and the skill row's 3/3 verify it.
Decision: the delta spec follows the skill (covered: discard; near: fold into the existing note, then
discard the offer; absent: clear the marker; the marker is cleared on the offer only when it is kept). This
is the behavior actually shipped, tested and evaluated, and the literal spec reading contradicts itself
for covered. **Confirmed by Joe 2026-09-21** (near offer is discarded after its claim is folded into the
existing note); the earlier flag and one-line-revert note are moot.

**D9. The query payload carries a hint next to `pending_offers`.** Added 2026-09-21 on Joe's answer to the
open question about the flag. The shim runs `engram query` on every request, so the payload is the most
reliable place to reach an agent; the flag was a bare boolean that cannot prompt anything. New payload field
`pending_offers_hint` (string, omitted when there are no pending offers, so payloads without offers are
byte-identical to before) carries the same instruction the update notice and write nudge print, held in one
shared constant (`pendingOfferCurateInstruction`, `internal/cli/offer.go`) so the three cues cannot drift.
The hint is derived from the flag at the two payload-construction sites (`runQuery` and
`mergeQueryPayloads`), never decoded from the wire, so a merged or served payload cannot carry a stale copy;
the served path is byte-identical to local because the handler captures `RunQuery` stdout.
- Cue coverage: of the automatic cues, the update notice and the write nudge (D4) and now the payload hint
  are all wired. The "serve just accepted a write" cue (dropped in D4) is partly covered: `serveLearn` and
  `serveAmend` run `RunLearn`/`RunAmend` with deps whose `LogWarning` is the server process's stderr and
  whose `ListMD`/`ReadSidecar` are set, so the write nudge does fire, but on the SERVER host's stderr, not in
  the served client's response. It reaches an operator watching the server, and any agent on the host that
  next runs `engram query` sees the hint. **Open item:** a served client never sees the nudge; not changed
  here.
- Alternatives rejected: (a) keep the bare boolean: cannot prompt, and D8's mid-turn gap stays; (b) embed the
  hint in every item: multiplies the bytes by the item count and belongs to no item; (c) a top-level
  `notices:` list: a general mechanism with one user, and a shape change every consumer must learn.
- Blast radius (verified by grep): `queryPayload` (`internal/cli/query.go`) gets one field; the two constructors
  (`runQuery`, `mergeQueryPayloads`) set it; `serve_client.go` decodes the parent payload into `queryPayload`,
  so a parent's hint is decoded harmlessly and the merged hint is recomputed from the OR-ed flag. Tests
  touched: `query_test.go`, `merged_query_internal_test.go`, `serve_test.go`, `offer_test.go`, `update_test.go`.
  Docs: `docs/GLOSSARY.md` and `docs/architecture/c3-components.md` (K6 payload shape) do not list
  `pending_offers` today, so no doc edit is required. Specs: `openspec/specs/vault-offer-curation/spec.md`
  (this change's delta modifies it); `vault-serve-api` mentions pending offers only for the write path.
- Follow-ups (not edited here, per the change's scope): `agent-instructions/skills/curate/SKILL.md` (lines 5
  and 28 describe the flag as a bare boolean; the skill is retired in 4.5, so no edit needed if it goes),
  `agent-instructions/guidance/shim.md` (no mention of `pending_offers`; option B in D8 would touch it),
  and `recall`'s SKILL.md (no mention found).

**D9 update (2026-09-21).** The hint text now instructs, not just points: "curating pending offers is expected
vault upkeep, not an extra: after you finish the user's request, curate them without asking — run `<command>`
and follow the curate runbook it returns". Still one shared constant; the notice and nudge prefixes dropped
their own "to curate them," so the sentence is not said twice. Reason: `curate-signal` (1.3, 3.3) showed all 9
S/N/R agents wrote the requested note and declined to curate ("you didn't ask for that", then an offer).

**D10. `pending_offers` and its hint are the first payload keys after `version`.** The shim-only R arm's first
query payload was 94-112 KB; the Bash tool showed a preview and saved the rest to a file, and 0/3 agents saw
the hint at the payload tail. Struct field order in `queryPayload` fixes YAML key order, so the two fields
were moved ahead of `phrases` and `items` (still `omitempty`, so payloads without offers are byte-identical).
Merged and served paths build the same struct, so they inherit the order. Alternatives rejected: (a) shrink the
payload so the tail fits the preview: the size is items' content, a separate lever with its own recall trade-offs
(see the payload-cut findings); (b) a separate `notices:` list: a new shape for every consumer, with the same
placement question. Test: a >100 KB payload has the hint in its first 1500 bytes.

**D11. Agents curate offers themselves on the signal (Joe, 2026-09-21).** Joe chose auto-curation knowing all 9
baseline agents declined and the skill row was 0/3, so the goal is reliable curation, not parity with a 0.
Risk: curate's `--discard` deletes an offer file; the vault is a git store so it is recoverable, and the
mitigation is the curate steps' judge-first rule (judge each offer against existing notes before any action).
Fallback if the stronger text still fails in `curate-signal`: revisit (option B in D8, or ask-first wording).

**D6. Validation plan (no paid runs in the conversion stage).** (1) Retrieval check with phrases real
agents generate (harvested from the kept baseline transcripts, plus `--text` forms), with over-fire
probes; no LLM spend. (2) Shim-only R arm n=3 on the explicit-ask task: bar is within one trial of the
skill row; the skill row is `end_state` 3/3, so the bar is `end_state` >= 2/3, and the runbook must
surface 3/3 by `trigger` provenance. (3) A second eval task `curate-signal`: the agent does routine work
in a vault with pending offers and the notice appears mid-turn (an `engram update` report or a learn/amend
nudge); arms S (skill installed), N (bare), R (runbook, shim only); measured: does the agent follow the
notice's instruction, surface the runbook and curate? This tests option A (firing through the notice
text). Cost is estimated and confirmed before any paid run.

**D7. Retirement gate.** `curate/SKILL.md` is deleted only when (a) the D6 bars are met, (b) retrieval is
verified against the production vault after promotion (`engram learn runbook`, never a hand copy), and (c)
`shim.md` is confirmed imported in the real `~/.claude/CLAUDE.md` (vault note 1031a); plus a
doc-surface enumeration (every live reference to the curate skill, the notice texts, and the three
surfacing signals) reviewed independently by a fresh-context reviewer, per the please change's
enumeration precedent.

**D8. Firing for the mid-turn case is via the notice text, not the shim.** The shim's four re-entry
cues (endorse an approach, declare done, unexplained failure, new approach) are not keyed on
tool-output notices, so it does not cover "engram printed a pending-offers line". Option A (this change):
the notice says what to run. Option B (only if A proves unreliable in `curate-signal`): a generic fifth
shim re-entry cue "when engram output tells you to run `engram query`, do". Not built here.

## Risks / Trade-offs

- [The agent ignores the notice] -> `curate-signal` measures it; fall back to option B.
- [`curate` fires on unrelated uses of the word] -> a trigger hit is a candidate only; the situation
  text discriminates; accepted.
- [Spec reconcile guesses Joe's intent] -> resolved: D5 confirmed by Joe 2026-09-21.
- [Retirement strands curate if the shim is not imported] -> gate (c).
