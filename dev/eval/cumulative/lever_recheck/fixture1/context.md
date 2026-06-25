# Orchestra cost notes (scratch log)

Per-document spend, sampled over last week's run:

- end-to-end per-doc cost: ~$0.041 avg
- retrieval / context-fetch step: a minority slice of the per-doc cost
- generation step: the majority of the per-doc cost (it makes the long model calls)

Experiments logged:
- swapped the retrieval step onto the cheap small model for one batch — measured about a 14% drop in
  total per-doc cost on that batch.
- batching the generation calls: not yet tried.
- trimming the prompt context length: not yet tried.
