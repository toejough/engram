## Why

The 27 `runbook` notes (Luhmann 820–846) migrated into the production vault under #730 have never been tested for retrieval under realistic crowding — chain link (b) of #738's four-link runbook-validation chain (fires → **surfaces** → followed → outcome-beats-fact). The only existing surfacing test, `internal/cli/query_runbook_test.go`, is a 2-note toy vault proving structural symmetry (no exclusion, no kind-specific boost); it passed with zero production-code changes, i.e. it confirmed an already-true property, not that these 27 notes actually rank for the real 783-note vault when a matching task is phrased the way a future task would phrase it. ADR-0026 explicitly records that #719's planned validation gate was cut at ship time. #734 is deliberately the cheapest, first-run check in the chain: no LLM calls, embedder + real `engram query` only — it gates the expensive applied-efficacy A/B (#737, HARD-GATED on this being green) per the vault's own migrated doctrine (`822.2026-08-29.tier1-gate-before-tier2-stressor-eval`): run the free probe before the costly one.

Grounding this probe's design surfaced a second, independent finding: both #734's own repro text and `recall-runbook-surfacing/spec.md`'s existing scenario name `candidate_l2s` as the "did it surface" signal. That's stale. ADR-0025 (accepted 2026-07-29, BREAKING) narrowed `candidate_l2s` to a capped top-5-by-centroid-cosine, within-cluster subset used for L2-synthesis-candidate selection — not a general "did this note match" signal. A note can match the query, sit correctly in the top-level `items[]` matched set, and still be absent from `candidate_l2s` simply because its cluster has more than 5 members closer to the centroid. With 27 topically-related runbooks likely clustering together, that's not an edge case. Checking `candidate_l2s` would silently misreport healthy runbooks as retrieval failures. `items[]` — the flat, ranked, deduped list actually rendered to the recalling agent — is the correct signal, and it's what `dev/eval/traps/retrieval_probe.py` already parses today.

## What Changes

- New probe script under `dev/eval/`, generalizing `dev/eval/traps/retrieval_probe.py`'s reusable plumbing (`rank_in_payload`, YAML-payload parsing) from its current fixed 3-axis shape (C3/C4i/C6) to 27 dynamically-derived runbook targets.
- For each of the 27 runbooks: one batched `engram query --vault <scratch-copy> --phrase P1 --phrase P2 [--phrase P3]` call — 2–3 task-shaped phrasings per runbook in a single call, mirroring how `/recall` itself fires (several phrasings of one real task in one call). Faster and cheaper (27 calls) than isolating every phrasing into its own call; chosen deliberately over the more diagnostic per-phrase-isolated variant as the first pass.
- **Escalation path** (built only if the batched pass shows a miss): per-phrase isolated calls for any runbook that misses on the batched check, to diagnose which specific phrasing(s) failed.
- Phrasing corpus: 27 runbooks × 2–3 non-tautological, task-shaped phrasings — already drafted and QA-checked (fresh-agent draft, then a second fresh-agent tautology/vagueness pass against each note's own `situation:`/`done_when:` text per vault note 288's named trap; 18 of 27 runbooks had at least one phrasing caught and rewritten). Checked into `dev/eval/` as reusable data, pending Joe's review pass over the draft.
- Pre-registered pass bar, stated before running: aggregate recall@5 (via `items[]` rank) ≥ the matched-note-floor embedder ceiling (~0.83) minus an explicit stated tolerance. Any runbook missing on all its phrasings is flagged with its best rank and a diagnosis (situation wording vs. genuine embedder miss).
- A `dev/eval/LEDGER.md` row recording the result, following the existing row format.
- A list of any runbooks needing `engram resituate`, if the pass bar isn't met.
- Fix `openspec/specs/recall-runbook-surfacing/spec.md`'s stale scenario, which currently asserts a runbook "appears in `candidate_l2s` ... like any other retrieval-competitive note" — a spec-accuracy correction independent of what the probe run itself finds, since as written the spec describes the wrong mechanism post-ADR-0025.

## Capabilities

### New Capabilities
- `retrieval-probe-signal-fidelity`: the requirement that any harness measuring whether a note "surfaced" in an `engram query` result reads the payload's top-level `items[]` rank — never `clusters[].candidate_l2s`, which ADR-0025 narrowed to a capped top-5-per-cluster L2-synthesis-candidate subset, not a general surfacing signal. Mirrors the precedent set by `eval-warm-config-fidelity`: a silent harness-correctness bug, root-caused, fixed structurally, and written down so it can't recur in this or future probes (e.g. #737).

### Modified Capabilities
- `recall-runbook-surfacing`: correct the "Runbook note inclusion in query results" scenario to name `items[]` (the actual matched-set/surfacing signal) instead of `candidate_l2s` (a narrower, capped, within-cluster L2-synthesis-candidate list unrelated to general surfacing since ADR-0025).

## Impact

- New file(s) under `dev/eval/` (probe script + phrasing data), reusing `dev/eval/traps/retrieval_probe.py`'s parsing plumbing rather than duplicating it.
- `dev/eval/LEDGER.md` — new dated row.
- `openspec/specs/recall-runbook-surfacing/spec.md` — scenario correction (`candidate_l2s` → `items[]`).
- Read-only against a **scratch copy** of the vault only — `engram serve` is live at `127.0.0.1:8093` on this machine; the probe never touches the real vault or the (env-var-non-redirectable) chunk index directly.
- Gates #737 (runbook-vs-original efficacy A/B) per #738's chain ordering: green here unblocks it; a miss means `engram resituate` the affected notes and re-probe before #737 proceeds.
- Closes/updates #734; informs #738's tracking table.
