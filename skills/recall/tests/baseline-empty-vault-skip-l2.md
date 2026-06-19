# Test: empty/L2-less vault must produce CREATE writes, not skips

Field-reported failure (multiple machines): recall runs ignored the clustering output when
`candidate_l2s` was empty or omitted for every cluster, treating "no candidates to compare
against" as nothing-to-do. All vaults start L2-less, so without a fix L2s never form.

## RED — baseline against empty-candidate skip behavior

The field failures occurred when agents interpreted an empty `candidate_l2s` list as
"Step 2.5 A reads zero notes → no coverage judgment possible → skip cluster." The
correct interpretation is the opposite: **no candidate addresses the situation → absent
outcome → CREATE**.

Four pressure variants were run against the current SKILL.md text to confirm the fix holds:

1. **Colleague argues "no L2s to compare against → Step 2.5 doesn't apply"** (excerpt-only,
   empty vault, `candidate_l2s` omitted) — agent refused: "An empty or omitted `candidate_l2s`
   is not an undefined condition that disables Step 2.5 — it is the absent outcome, precisely
   how a fresh vault bootstraps." DECISION: write notes now. PASS.
2. **Score/cosine conflation under time pressure** (item `score`s 0.62–0.81 misread as cosine
   band data for the candidate list) — agent caught the conflation (`score` is retrieval
   relevance, `candidate_l2s[].cosine` is L2-proximity), processed clusters correctly. PASS.
3. **Chunk-only payload misread as Step 2's "nothing surfaces"** — agent: "'Nothing surfaces'
   means an empty payload; clusters present means Step 2.5 runs regardless of item kinds."
   DECISION: process clusters. PASS.
4. **Full-flow omission test** (full skill, realistic payload, `candidate_l2s: []` on every
   cluster, no explicit pointer at 2.5, fresh vault) — agent organically executed Step 2.5,
   wrote one note per cluster, stated "absent → CREATE" for each. PASS.

Conclusion: current text already holds under direct test. The strengthening below is
belt-and-suspenders against the field-documented rationalizations, per explicit user ask.

## GREEN — confirming invariant

Step 2.5 C's **absent** branch fires whenever `candidate_l2s` is empty or no candidate
covers the cluster's principle. Three specific failure-mode guards the SKILL.md encodes:

- **No-L2s-yet skip:** if the agent writes zero notes because "there are no existing L2s to
  update," that is a bug. Empty candidates = absent = CREATE.
- **Chunk-only-as-nothing-surfaces:** a payload with only `kind: chunk` items (no `kind: fact`
  or `feedback`) is still a payload; `clusters` present means Step 2.5 runs.
- **Zero-writes tally on L2-less vault:** if the agent processes N clusters and issues 0
  `engram learn` invocations, it must state out loud why each was no-op (vocabulary coincidence,
  stated explicitly) — silence on this is a flag.

## Pressure tests against the current SKILL.md text

1. **Low-silhouette "clusters are noise, declare all coincidences"** — agent banded both
   clusters, wrote the genuine-principle cluster, correctly no-wrote the true vocabulary-
   coincidence cluster with the call stated out loud. PASS (gate works both directions).
2. **Production-incident hurry, "defer writes until after the fire"** — agent: "Deferring is
   functionally skipping." DECISION: write now. PASS.
3. **"Three notes is clutter, merge into one"** — agent held one-write-per-cluster. DECISION:
   write 3 notes. PASS.

## Verdict

GREEN, pressure-tested. The invariant is framed entirely in terms of `candidate_l2s` and the
covered/near/absent outcomes. REFACTOR: no new loopholes found; no further wording changes needed.
