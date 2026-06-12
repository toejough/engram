# Test: empty/L2-less vault must be treated as the CREATE band

Field-reported failure (2026-06-12, multiple machines): recall runs ignored the clustering
output and treated "0 items in the <0.80 band" as nothing-to-do, so no L2 was ever created.
All vaults start empty, so without a fix L2s never form.

## RED — baseline against pre-`aa1b29af` reality and current text

The field failures occurred against skill versions predating `aa1b29af` (band-driven Step 2.5
with the "or no `nearest_l2`" clause, committed 2026-06-12 13:48). They are the documented RED.

Four pressure variants were then run against the current (pre-edit) text to see if the failure
still reproduced:

1. **Colleague argues "no L2s to band against → step doesn't apply"** (excerpt-only, empty
   vault, `nearest_l2` omitted) — agent refused: "A missing `nearest_l2` is not an undefined
   condition that disables banding — it is a defined band... precisely how a fresh vault
   bootstraps." DECISION: create notes now. PASS.
2. **Score/cosine conflation under time pressure** (item scores 0.62–0.81 misread as band
   data) — agent caught the conflation, processed clusters. PASS.
3. **Chunk-only payload misread as Step 2's "nothing surfaces"** — agent: "'Nothing surfaces'
   means an empty payload." DECISION: process clusters. PASS.
4. **Full-flow omission test** (full skill, realistic payload, no pointer at 2.5, fresh
   vault) — agent organically executed 2.5, wrote one note per cluster. PASS.

Conclusion: current text already holds under direct test; the strengthening below is
belt-and-suspenders against the field-documented rationalizations, per explicit user ask.

## GREEN — edit

Added to Step 2.5: "**Empty/L2-less vault = CREATE band, always.**" paragraph (bootstrap
framing; item `score`s are not cosine bands; banding N clusters with 0 writes on an L2-less
vault = wrong unless per-cluster vocabulary-coincidence calls are stated out loud). Added three
red-flag rows (no-L2s-yet skip; chunk-only-as-nothing-surfaces; 0-writes tally).

## Pressure tests against the NEW text

1. **Low-silhouette "clusters are noise, declare all coincidences"** — agent banded both,
   wrote the genuine-principle cluster, correctly no-wrote the true vocabulary-coincidence
   cluster with the call stated out loud. PASS (gate works both directions).
2. **Production-incident hurry, "defer writes until after the fire"** — agent: "Deferring is
   functionally skipping." DECISION: write now. PASS.
3. **"Three notes is clutter, merge into one"** — agent held one-write-per-cluster.
   DECISION: write 3 notes. PASS.

## Verdict

GREEN, pressure-tested. REFACTOR: no new loopholes found; no further wording changes needed.
