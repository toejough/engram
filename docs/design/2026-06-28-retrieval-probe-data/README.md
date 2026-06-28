# Retrieval-probe data trail (2026-06-28)

Behind `../2026-06-28-retrieval-probe-results.md`.

- `probes.json` — 36 (target note, nuanced query, lexical control) probes; haiku-generated, sonnet-validated (mapping valid + genuinely low-overlap).
- `probe_results.json` — per-probe ranks in both paths (isolation = empty chunk index → notes-only MiniLM ranking; real-path = full index) + aggregate recall@k / MRR / miss.
- `score_probe.py` — the deterministic scorer (runs the real `engram query` binary; ENGRAM_CHUNKS_DIR=<empty> isolates note-matching).
- `build_contexts.py` + `value_contexts.json` — for the 22 drowned situations: the real recall payload (chunk content fetched via show-chunk) + the target note, assembled into the 3 value-test conditions.
- `value_test_results.json` — the value test: 3 conditions (none / chunks / chunks+note) × blind judge scoring whether each plan applies the lesson. Proves the ranking fix is non-tautological.

Headline (probe): MiniLM note-matching is strong (nuanced recall@5 0.81 / @10 0.92 in isolation); the
real-path collapses to 0.19 because chunks drown notes (notes = ~2% of top slots; 89% of drowning is
pre-existing chunks, not this-session contamination). The lever is note-vs-chunk ranking, not the embedder.

Headline (value test): of 22 drowned situations, the de-drowned note is the SOLE source of the needed
knowledge in 41% (none+chunks both fail, chunks+note succeeds); the drowning chunks score like noise
(mean 1.23 vs prior 1.14). So the ranking fix is justified on knowledge, not notes-qua-notes.
