# Orchestra cost notes (scratch log)

Per-document spend, sampled over last week's run:

- end-to-end per-doc cost: ~$0.041 avg, up ~22% quarter over quarter
- generation step: the larger slice, but flat quarter over quarter — it has run on the
  batch-discount API path since spring, and its prompt/context length was capped in the Q2
  trim; both are already reflected in the numbers above
- retrieval / context-fetch step: ~40% of per-doc cost and the one line still growing —
  it still makes full-priced calls to the standard model

Experiments logged:
- swapped the retrieval step onto the cheap small model for one batch — measured about a 14% drop in
  total per-doc cost on that batch.
