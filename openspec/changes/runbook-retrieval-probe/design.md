## Context

This is chain-link (b) of #738's four-link runbook-validation chain (fires → **surfaces** → followed → outcome-beats-fact). It is deliberately the cheapest link: no LLM calls, embedder + real `engram query` only, and it gates the expensive applied-efficacy A/B (#737, HARD-GATED on this being green).

Relevant existing state:
- `dev/eval/traps/retrieval_probe.py` already implements this pattern — one combined multi-phrase `engram query` call per "axis," parsing the target's rank out of the payload's top-level `items[]` list (`rank_in_payload`) — but hardcoded to 3 fixed cosine axes (C3/C4i/C6), each with its own baked-in phrase list and target-slug list.
- ADR-0025 (accepted 2026-07-29, BREAKING) narrowed `clusters[].candidate_l2s` to a capped top-5-by-centroid-cosine, within-cluster subset used for L2-synthesis-candidate selection. It is not a general surfacing signal. Both #734's own repro text and `recall-runbook-surfacing/spec.md`'s existing scenario still name `candidate_l2s` as the surfacing check — a stale assumption this change corrects (see `retrieval-probe-signal-fidelity` and the `recall-runbook-surfacing` delta spec).
- `engram serve` is live on this machine (`127.0.0.1:8093`) for the duration of this work. The probe must never touch the real vault or the chunk index directly (the chunk index is not env-var-redirectable — vault notes 516/517 — acceptable for a read-only rank probe given the matched-note floor guarantees matched notes aren't displaced by chunks within the phrase limit).
- A 27-runbook × 2–3-phrasing corpus (69 phrasings total) has already been drafted: fresh agents (no memory of this design conversation) read each runbook file cold and wrote concrete, task-shaped scenarios, then a second fresh agent QA-checked every phrasing against the note's own `situation:`/`done_when:` text for tautology (vault note 288: "handing the agent the answer measures a tautology") and vagueness. 18 of 27 runbooks had at least one phrasing flagged and rewritten. The reviewed draft is pending Joe's pass before being checked in as the final corpus.

## Goals / Non-Goals

**Goals:**
- Measure, per runbook, whether it surfaces (via `items[]` rank) for realistic task-shaped phrasings, against a scratch copy of the real 783-note vault.
- Produce a pre-registered, labeled-table result: aggregate recall@5 and per-runbook rank/miss, checked into `dev/eval/LEDGER.md`.
- Correct the `candidate_l2s`-as-surfacing-signal mistake everywhere it appears in this chain's own artifacts (probe code, spec).
- Produce a `engram resituate` worklist for any runbook that misses.

**Non-Goals:**
- Does not run the applied efficacy eval (#737) — that is a separate, HARD-GATED change that only proceeds once this one is green.
- Does not build the recall-firing cue for the runbook moment (#735) — independent chain link.
- Does not change `engram query`'s production behavior in any way — this is a read-only measurement against a scratch vault copy.
- Does not attempt to finalize the phrasing corpus without Joe's review — the drafted-and-QA'd set is an input to this change, not a foregone conclusion.

## Decisions

**Reuse `retrieval_probe.py`'s parsing plumbing rather than write a new parser.** The rank-extraction logic (`rank_in_payload`, YAML-payload parsing) is already correct and already reads `items[]`. Generalize its data shape from a fixed 3-name axis dict to a per-runbook target list (27 entries: luhmann_id, phrasings, target slug), loaded from data rather than hardcoded per-axis. Alternative considered: a bespoke standalone script — rejected, it would duplicate already-correct, already-tested parsing logic (vault note 200: reuse harness plumbing for one-off smokes).

**Batched multi-phrase-per-runbook first; per-phrase isolated calls only on a miss.** One `engram query --vault <scratch> --phrase P1 --phrase P2 [--phrase P3]` call per runbook (27 calls total) mirrors how `/recall` itself fires — several phrasings of one real task in a single call — and is the faster, cheaper first pass. The alternative (isolating every phrasing into its own call, ~70 calls, answering "does phrasing X alone find it") is strictly more diagnostic but not needed for runbooks that already pass the batched check. Decision: run batched first; escalate to per-phrase isolated calls only for runbooks that miss, to identify which specific phrasing(s) failed.

**`items[]` rank is the surfacing signal, not `candidate_l2s`.** See Context. Encoded as its own capability (`retrieval-probe-signal-fidelity`) rather than left as an implicit property of the script, because the mistake was independently present in two places (the issue's own repro text and the existing spec) — writing it down structurally (with a regression-testable scenario) prevents a third recurrence, e.g. in #737's harness.

**Phrasing corpus drafted by fresh-dispatched agents, not written inline in this conversation.** Chosen over drafting them directly in-session because fresh agents reading the runbook files cold cannot anchor on how this conversation has already been summarizing/paraphrasing each note's `situation:` field — a real tautology risk given how much of this design discussion has already quoted those fields. A second fresh-agent QA pass then checked every phrasing against the actual frontmatter text for verbatim/near-verbatim overlap. This caught real issues (18/27 runbooks flagged) — confirming the two-stage process was necessary, not just cautious. Joe's own review is still the final gate before the corpus is treated as final.

**Scratch-copy vault isolation.** `cp -R` the real vault to a temp directory and pass `--vault` explicitly to every `engram query` call, per this repo's existing `isolation.py` convention and the issue's own explicit warning (`engram serve` is live at `127.0.0.1:8093` right now). Never probe the live vault.

**Pre-registered pass bar.** Aggregate recall@5 (via `items[]` rank) ≥ the matched-note-floor embedder ceiling (~0.83) minus an explicit stated tolerance, stated before the run per house eval convention (never set the bar after seeing the numbers). The exact tolerance value is an open question below.

## Risks / Trade-offs

- [Batching phrasings into one call could dilute or mask a single bad phrasing's signal] → the escalation path (per-phrase isolated calls) exists specifically to recover per-phrasing diagnosis for any runbook that misses the batched check — no diagnostic power is permanently lost, only deferred until it's needed.
- [Phrasing quality is a judgment call, not purely structurally checkable] → mitigated by the two-stage fresh-draft-then-fresh-QA process (already run, 18/27 flagged and fixed) plus a required human review pass before the corpus is finalized; not yet fully closed out.
- [The 783-note vault will keep growing after this snapshot] → this probe is a point-in-time measurement. If the vault changes materially before #737 runs, re-probe first rather than citing a stale result (mirrors the existing "stale eval results don't bind a new architecture" convention).
- [`cp -R` against a live-serving vault is not atomic] → `engram serve` could be mid-write during the copy. Mitigation: check for recent write activity before copying, and note the copy's timestamp in the LEDGER row so a later contamination concern can be checked against it.

## Open Questions

- Exact numeric tolerance below the ~0.83 ceiling to treat as a pass (the issue calls for "an explicitly stated tolerance" but doesn't pin a number) — needs to be set before the run counts as pre-registered.
- Joe's review pass over the drafted 27×2–3 phrasing corpus (sent for review) — outstanding; the corpus checked into `dev/eval/` should reflect his edits, not the raw draft.
- Exact file/module layout for the generalized probe under `dev/eval/` (e.g. new `dev/eval/traps/runbook_probe.py` plus a data file, vs. extending `retrieval_probe.py` in place) — left to tasks/implementation, not architecturally significant either way.
