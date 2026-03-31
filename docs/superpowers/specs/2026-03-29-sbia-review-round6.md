# SBIA Spec Review — Round 6

6 issues from systematic cross-checks (disposition tracing, counter arithmetic, strip mode consumers, extraction flow completeness).

## Issue 1: Disposition names inconsistent between tree and table

**Location:** Decision tree (lines 242, 244) vs Disposition Outcomes table (lines 257, 258)

**Problem:** The decision tree uses `POTENTIAL GENERALIZATION` and `LEGITIMATE SEPARATE MEMORIES` but the outcomes table uses `GENERALIZATION` and `LEGITIMATE SEPARATE`. Sonnet's structured output needs one canonical name per disposition.

**Proposed fix:** Align the outcomes table to match the decision tree (the tree is more precise):

- `GENERALIZATION` → `POTENTIAL GENERALIZATION`
- `LEGITIMATE SEPARATE` → `LEGITIMATE SEPARATE MEMORIES`

Sweep: also check Resolved Questions #3 (line 831) for any disposition name references.

[joe] approved

## Issue 2: `value` field description incomplete for recommend and merge

**Location:** Proposal schema field table, line 465

**Problem:** Says "New text/number, — (for delete)" but the diagnosis mapping shows:

- `recommend` also has null value (line 476)
- `merge` has synthesized content — the entire replacement memory (line 477)

These aren't mentioned in the field description.

**Proposed fix:** Change line 465 from:

> `| value | New value | New text/number, — (for delete) |`

To:

> `| value | New value | New text/number; synthesized content (for merge); null for delete and recommend |`

Sweep: check the JSON example (line 454) — `"related": []` is shown for a non-merge proposal. Verify `value` examples are consistent.

[joe] approved

## Issue 3: Counter simplification says "three" but total is four

**Location:** Line 381

**Problem:** "The current model has five counters (surfaced, followed, contradicted, ignored, irrelevant). SBIA simplifies to three:" — The "five" includes surfaced, but the "three" excludes it. Actual simplification is 5 → 4 (two evaluation counters collapsed into one). Line 391 clarifies surfaced is retained, but the leading sentence is misleading.

**Proposed fix:** Change:

> "The current model has five counters (surfaced, followed, contradicted, ignored, irrelevant). SBIA simplifies to three:"

To:

> "The current model has five counters (surfaced, followed, contradicted, ignored, irrelevant). SBIA simplifies the evaluation counters from four to three:"

Sweep: check Resolved Questions #4 (line 832) for the same count claim.

[joe] I think that just continues to confuse. "The current has 5, the new simplifies from 4 to 3"?! Nevermind the nuance
of "evaluation counters" - it just looks like you're immediately self-contradictory. Just reference all 4 counters we're
keeping. it's simpler.

## Issue 4: Surfacing strip mode not explicitly stated

**Location:** Surfacing pipeline step 1 (line 297-298)

**Problem:** The StripConfig table (lines 190-199) defines "Recall mode" and "SBIA mode" but doesn't list surfacing as a consumer of either. The surfacing pipeline says "(shared with extraction context — same transcript, same budget)" implying SBIA strip mode, but never names it.

**Proposed fix:** Change line 298 from:

> `(shared with extraction context — same transcript, same budget)`

To:

> `(SBIA strip mode — shared with extraction context, same transcript and budget)`

Sweep: check the StripConfig table for whether it should list surfacing as a consumer, or whether the inline note is sufficient.

[joe] approved

## Issue 5: Extraction flow missing disposition handling step

**Location:** After Revised Extraction Flow step 4 (line 214-216)

**Problem:** Step 4 ends with "outputs SBIA fields + per-candidate disposition" but there's no step 5 mapping dispositions to concrete system actions. This matters for non-trivial dispositions:

- **STORE / STORE BOTH / LEGITIMATE SEPARATE MEMORIES**: Write new memory (obvious but unstated)
- **DUPLICATE**: Don't write; log self-diagnosis (documented separately at lines 262-269 but not referenced from extraction flow)
- **POTENTIAL GENERALIZATION**: "Merge into broader situation" — does Sonnet return a broadened version of the existing memory? Does the system update in place?
- **CONTRADICTION / REFINEMENT**: "Surface to user" / "Flag for review" — via what mechanism at UserPromptSubmit time?

**Proposed fix:** Add step 5 after line 216:

```
5. **Handle disposition** per candidate:
   - STORE / STORE BOTH / LEGITIMATE SEPARATE MEMORIES: Write new memory
   - DUPLICATE: Don't write. Run self-diagnosis (see [Self-Diagnosis on DUPLICATE](#self-diagnosis-on-duplicate))
   - POTENTIAL GENERALIZATION: Sonnet returns broadened situation; update existing memory's situation field in place, don't create new
   - CONTRADICTION: Write new memory. Emit warning to user with both memories for resolution at next /memory-triage
   - REFINEMENT: Write new memory. Flag both for user review at next /memory-triage
```

Sweep: verify that the Self-Diagnosis section (lines 262-269) cross-references correctly, and that CONTRADICTION/REFINEMENT handling is consistent with the Disposition Outcomes table.

[joe] approve

## Issue 6: Decision tree doesn't note its scope excludes consolidation

**Location:** After Maintain Decision Tree (line 439)

**Problem:** The decision tree covers per-memory health only. But `engram maintain` also produces consolidation proposals (line 477: "Similar situations → merge") and parameter/prompt proposals (lines 478-479). The presentation order (lines 497-503) includes both at priorities 4 and 5. A reader following the decision tree would think it's the complete set of maintain diagnoses.

**Proposed fix:** Add after line 439:

> This tree covers per-memory health diagnosis. Consolidation (similar situations → merge) and system tuning (parameter/prompt changes via `adapt_sonnet`) are separate analyses run by Sonnet during the same `engram maintain` call. See [How Each Diagnosis Maps to a Proposal](#how-each-diagnosis-maps-to-a-proposal) for the full set of proposal types.

Sweep: check that the diagnosis-to-proposal mapping table header or preamble doesn't claim to be derived solely from the decision tree.

[joe] approve
