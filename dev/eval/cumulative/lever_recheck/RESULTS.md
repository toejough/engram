# C7 lever-recheck — run evidence (2026-06-24)

Two live runs of the **real `/recall` skill** (fresh opus agents, isolated to `fixture1`) during the
build of this harness. They ground the findings in `README.md`. Both are reproducible with the
committed fixture + stub.

## Run 1 — RED-reproduction probe (no stub; real vault)

Setup: fresh opus agent, `ENGRAM_VAULT_PATH=fixture1/vault_with_closed`, isolated empty chunks dir,
`task.txt`. The agent ran its `/recall` skill, then recommended.

Result: **RECONCILED.** Recall surfaced note 8 (`cheap-retrieval-model-rolled-back`) at cosine ≈ 0.95.
The agent excluded the cheaper-retrieval lever and stated, verbatim, in its `PRIOR-ATTEMPTS-CONSIDERED`:

> recall surfaced note 8 … the "swap retrieval onto the cheap small model" experiment … was already
> tried and ROLLED BACK … This removed the retrieval-model swap from consideration entirely.

Conclusion: when the disproving note **surfaces**, opus reconciles correctly — the miss is not a
synthesis failure with a salient note.

## Run 2 — validation with `stub_engram` on PATH

Setup: same, but with `stub_engram` on PATH (buries note 8 unless a query is lever-keyed) and its query
log enabled. The skill's single upfront recall emitted these 10 phrases (captured by the stub log):

```
Orchestra document-processing pipeline cost per document
cut per-document cost of retrieval and generation pipeline
swap generation model to cheaper smaller tier
retrieval step runs on every document cost driver
use a cheaper retrieval model to save cost          <-- lever-keyed
LLM model tier selection cost vs quality
prior experiment cheap retrieval model rolled back  <-- queries the lever's HISTORY
trim retrieved context fed into generation
cost optimization that regressed output quality
retrieval generation two-stage RAG cost optimization
```
→ `lever_keyed: true, returned_buried: true` (the stub surfaced note 8 because the recall was
lever-keyed), and the agent again **RECONCILED**.

Conclusion (the load-bearing finding): the current recall skill's 10-angle phrasing — its
candidate-solution, prior-work, and failure-mode angles — **proactively queries the lever AND its prior
outcome** when the lever is conceivable at recall time. So the skill already defends against the miss in
that case; the real miss requires the lever to be conceived *strictly after* the single recall.

## Retrieval sweep (deterministic)

`engram query` against `vault_with_closed/` ranks note 8 **#1 for every framing tried** (cost,
architecture, quality, latency, reliability) — confirming a small vault cannot bury the note by scale.
