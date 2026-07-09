# Deskmate search notes (scratch log)

Search quality metrics, sampled over the last month:

- precision@3 (top-3 results relevant to the incoming ticket): down from ~0.71 to ~0.58 overall
- deflection rate: down from ~54% to ~41%
- diagnostic sample: for the subset of incoming tickets whose wording closely echoes the
  wording of an earlier ticket that had originally prompted a KB article's creation,
  precision@3 on that subset is ~0.95
- for incoming tickets phrased differently from any article's originating ticket, precision@3
  on that subset is ~0.34 — and this subset is the majority of current ticket volume
- the KB's search index currently matches on each article's own title/body content
- expanding the synonym list to cover more paraphrase variants: not yet tried
- re-weighting recency (surfacing recently-edited articles higher): not yet tried
