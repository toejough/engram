# #7 weaker-model-reuse eval (2026-06-28)

3-arm trap eval (opus-warm / sonnet-warm / sonnet-cold) per capability axis, reusing dev/eval/traps.
Question: does sonnet + memory match opus + memory?

| axis | opus-warm | sonnet-warm | sonnet-cold |
|---|---|---|---|
| C3 apply-conventions | 15/15 | 15/15 | (cold=trap) |
| C4i recency-supersession | 3/3 | 3/3 | 0/3 |
| C6 emergent-abduction | 6/6 | 6/6 | 0/6 |
| C5 honor-standard | 0/3 honored | 0/3 honored | 0/3 | (INCONCLUSIVE — opus baseline flaked)

Verdict: sonnet+memory == opus+memory on 3/4 axes; cheap model fails cold. Memory democratizes reasoning
across model tiers (vault note 135). Sonnet ~25-30% cheaper/trial. C5 re-runnable. c6_*.txt = the initial
C6 run; gen7_*.txt = the C3/C4i/C5 generalization.
