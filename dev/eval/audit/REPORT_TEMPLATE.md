# Memory Loop Audit Report Template

**Status: Pre-committed counting units (task 4.1) — NO NUMBERS COMPUTED YET**

This template fixes the counting unit for every rate the audit will report, BEFORE any real numbers exist (task 5.1's full run hasn't happened yet). Deciding the counting unit in advance prevents unconsciously picking a favorable unit after the fact.

Authority: `openspec/changes/memory-loop-audit/design.md` decision D-G(a), requirement "Every audited moment SHALL carry a complete record" in `specs/memory-loop-audit/spec.md`.

---

## 1. The Counting-Unit Principle

**The Rule (from spec.md, scenario "A rate without a fixed counting unit is a violation"):**

> No rate SHALL be computed before its counting unit is fixed in advance. Every reported number SHALL be labeled DERIVED (computed from real records) or ESTIMATE (approximated because exact data wasn't available), and the report SHALL name what's being counted: per-moment, per-dispatch, or per-transcript.

**Rationale (design.md D-G(a)):**

The counting unit is **mainly per-moment** as the default and primary view — one row per individual moment detected in the corpus (a failure, correction, rework, success, or dispatch event).

For a specific exception: **per-dispatch** applies to the finding-side gap — "which memories the orchestrator had but didn't hand over." This is inherently a property of a DISPATCH event, not an individual moment; the rate answers "what fraction of dispatches lacked a relevant note that should have been included." (Design.md D-E specifies this measured from the orchestrator's side, at the moment of dispatch.)

**Per-transcript** is a secondary / supplementary view for stage-level summaries across all windows, used to cross-check for per-repo skew and to contextualize results (e.g., "the writing side showed X loss at the per-moment level; across all transcripts, that's Y distinct transcripts affected").

---

## 2. Scorecard Stages and Their Counting Units

| # | Stage | Scorecard Question | Counting Unit | DERIVED / ESTIMATE |
|---|---|---|---|---|
| **Finding Side** | | | | |
| F1 | Memory existence | Was a relevant memory in the vault at the moment's timestamp? | **per-moment** | DERIVED (git checkout) or ESTIMATE (date filter fallback) |
| F2 | Search ran | Did a search actually run? (quick glance, full recall, injected prompt, or none) | **per-moment** | DERIVED |
| F3 | Search targeted correctly | Did the search target the right thing? (judged separately from whether a search ran) | **per-moment** | DERIVED |
| F4 | Memory surfaced | Was the memory surfaced to the agent? (with rank; did an outdated note out-rank its replacement) | **per-moment** | DERIVED |
| F5 | Memory followed | Was the surfaced memory actually followed? (recording how it deviated: ignored, partial, step skipped, out of order, contradicted) | **per-moment** | DERIVED |
| **Writing Side** | | | | |
| W1 | Worth learning | Was this moment worth learning from? | **per-moment** | DERIVED |
| W2 | Learn step fired | Did a learn/write-memory step actually execute? | **per-moment** | DERIVED |
| W3 | Note written | Did a new or updated note get written to the vault? (was it well-targeted, did it correctly supersede an old note, was it not a duplicate) | **per-moment** | DERIVED |
| W4 | Note strength updated | Was the used note's strength updated? (with mismatches flagged) | **per-moment** | DERIVED |
| **Dispatch Handoff** | | | | |
| D1 | Memories handed over | Which memories the orchestrator had but didn't include in the dispatch prompt? (per design.md D-E: measured from orchestrator side; forks count as "handed over" automatically since they inherit the full parent context) | **per-dispatch** | DERIVED |

---

## 3. Rate Computation Rules

For each stage above, the audit will report a rate: (# of moments / # of total moments) for per-moment stages, or (# of dispatches with the gap / # of total dispatches) for the per-dispatch stage D1.

**Explicit commitment:**

- Every rate in the final report SHALL state its counting unit unambiguously in the `Counting Unit` column above.
- No rate SHALL be reported without a counting-unit label.
- Per-dispatch rates (D1) MUST be labeled as such, never left ambiguous as "% of moments" when the denominator is actually dispatches.
- Per-transcript summaries (secondary view) are used to contextualize results and check for single-repo skew, but SHALL NOT replace the primary per-moment counts in the main results table.

---

## 4. DERIVED vs. ESTIMATE Labeling

Every rate SHALL be labeled with one of:

- **DERIVED**: the count comes directly from real records (e.g., git checkout point-in-time check for vault notes, subprocess-call detection for searches, hand-read moment detection on sampled windows).
- **ESTIMATE**: the count is approximated because exact data wasn't available (e.g., for vault notes where git history doesn't reach back to the moment's timestamp, falling back to creation-date filtering).

The report's "Limits" section (task 6.1) SHALL note the scope of any ESTIMATE labels: which moments fall back to approximate checks, how far back the vault's git history reaches, and what coverage the sample represents versus the full corpus.

---

## 5. Blank Results Table (to be filled in by task 6.1)

**This section is populated by the actual audit run; placeholder values below are placeholders only.**

| Stage | Rate (TBD) | Counting Unit | DERIVED / ESTIMATE | Notes |
|---|---|---|---|---|
| F1: Memory existence | — | per-moment | DERIVED or ESTIMATE | Searched vault as of moment's timestamp |
| F2: Search ran | — | per-moment | DERIVED | Detected from transcripts |
| F3: Search targeted correctly | — | per-moment | DERIVED | Judged separately from F2 |
| F4: Memory surfaced | — | per-moment | DERIVED | Includes rank and out-rank checks |
| F5: Memory followed | — | per-moment | DERIVED | Records deviation modes if not followed |
| W1: Worth learning | — | per-moment | DERIVED | Judged at moment time |
| W2: Learn step fired | — | per-moment | DERIVED | Skill or command invocation detected |
| W3: Note written | — | per-moment | DERIVED | Supersession and duplication flagged |
| W4: Note strength updated | — | per-moment | DERIVED | Mismatches flagged |
| D1: Dispatch handoff gap | — | **per-dispatch** | DERIVED | Orchestrator side; forks auto-handed |

---

## 6. Cross-Reference: Per-Transcript Summary (Secondary View)

To check for single-repo skew and provide context, the report MAY include a per-transcript breakdown:

| Repo | # Transcripts Sampled | Per-Moment Finding-Side Loss (%) | Per-Moment Writing-Side Loss (%) | Dispatch Handoff Gap (# / # dispatches) |
|---|---|---|---|---|
| TBD | — | — | — | — |
| TBD | — | — | — | — |

*(Secondary view only; primary results are per-moment aggregate.)*

---

## 7. #739-Specific Rates (Task 5.2)

For the Phase 2 go/no-go decision (task 4.2 will set thresholds), compute:

- **Writing side:** of the moments marked "worth learning but learn step didn't fire," how many are realistically reachable by a "write lessons in completion report" step? (Count per-moment.)
- **Finding side:** of the dispatches in the D1 gap (memories the orchestrator had but didn't hand over), what fraction is realistically reachable by a "inject memories into dispatch prompt" step? (Count per-dispatch, as an exception to the primary per-moment view.)

These rates will be compared against the pre-agreed thresholds in task 4.2 to decide whether Phase 2 proceeds.

---

## 8. Notes

- **Task 5.1** computes the raw per-moment records and fills in this template's results table.
- **Task 5.2** derives the rates using the counting units fixed here.
- **Task 6.1** writes the full report, filling in the blanks above with real numbers, and adds the "Limits" section documenting the scope of DERIVED vs. ESTIMATE labels.
- **No numbers are computed yet.** This template is purely a commitment to the counting unit, not a result.
- Every number that will appear in the final report is already assigned a counting unit here, so no rate can be computed using an undeclared or post-hoc unit.
