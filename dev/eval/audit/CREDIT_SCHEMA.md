# Outcome-Record Schema (CREDIT_SCHEMA)

Task 5.3 of the `memory-loop-audit` OpenSpec change. This documents the format of
`dev/eval/audit/results/outcome-records.jsonl`, produced by
`dev/eval/audit/generate_outcome_records.py` from the finalized
`dev/eval/audit/results/moments.jsonl` (150 audited moments) and
`dev/eval/audit/results/transcript-events.jsonl` (568 mechanical-extractor records).

**This file is a data contract for a follow-on change.** Nothing in the
`memory-loop-audit` change reads or acts on these records — see design.md's D-F
("output only, nothing reads it yet... Nothing in this change consumes these
records — they're a contract that the follow-on change ... will be built
against"). The records are draft input for a later capability that has not been
designed yet.

## Schema

Verbatim from `openspec/changes/memory-loop-audit/specs/memory-loop-audit/spec.md`:

> The audit SHALL emit draft outcome records —
> `{moment_id, note_ref, situation_text, outcome ∈ {applied-helped, applied-hurt,
> surfaced-ignored, injected-unused, absent-needed}, ts, evidence{transcript,
> anchor}}` — as plain data files. The audit SHALL NOT write, amend, activate,
> supersede, or remove any vault note: acting on these records requires a human
> judgment call (ADR-0028 D3) and belongs to a separate, later capability.

Six top-level fields, one JSON object per line (JSONL):

| Field            | Type              | Meaning                                                                 |
|------------------|-------------------|--------------------------------------------------------------------------|
| `moment_id`      | string            | The source moment's own `moment_id` from `moments.jsonl`.               |
| `note_ref`       | string \| null    | Vault note path this record is about, or `null` when no note applies/resolves. |
| `situation_text` | string            | Human-readable context for the record (see derivation below).           |
| `outcome`        | enum (5 values)   | One of `applied-helped`, `applied-hurt`, `surfaced-ignored`, `injected-unused`, `absent-needed`. |
| `ts`             | string            | The source moment's own `timestamp` (ISO 8601).                         |
| `evidence`       | object            | `{transcript, anchor}`, copied verbatim from the moment's own `evidence`. |

The outcome enum's five values, exactly as specified:

- **applied-helped** — the agent had the note surfaced and followed it, and it helped.
- **applied-hurt** — the agent had the note surfaced and followed it, and it contradicted/hurt the outcome.
- **surfaced-ignored** — a discrete recall search surfaced the note but the agent didn't follow it (ignored / skipped a step / did it out of order / partially).
- **injected-unused** — the note reached the agent via inherited fork context (not a discrete search) but was still not followed.
- **absent-needed** — at a failure/rework moment, either no relevant memory existed in the vault, or relevant memory existed but was never surfaced to the agent.

## Derivation rules

Two independent rules run over each of the 150 moments; a single moment can
produce records from both rules, from neither, or from one.

### Rule 1 — applied-helped / applied-hurt / surfaced-ignored / injected-unused

Applies only when `moment_type ∈ {success, failure, rework}` **and**
`finding_side.followed` is not null. (`dispatch` and `correction` moments never
produce Rule 1 records regardless of `followed`.)

1. **Resolve candidate note paths** for the moment:
   - `search_ran == "injected"` (a forked subagent that inherited its parent's
     context rather than running a discrete search): resolve via
     `_resolve_fork_parent_context` (imported unchanged from `audit_moments.py`,
     not reimplemented) and take the paths from its `recalled_notes`.
   - `search_ran ∈ {"quick_glance", "full_recall"}` (a discrete recall search ran):
     look up the moment's `transcript_path` in `transcript-events.jsonl` by exact
     string match, then take the **last** `recall_call` in that transcript whose
     `location` is `<=` the moment's own `location`, and use the paths from its
     `matched_items`.
   - `search_ran == "none"`: Rule 1 does not apply at all — no record is emitted
     from this rule for this moment, regardless of what `followed` says.
   - Exact-duplicate paths within one moment's candidate list are deduped before
     emission.
2. **If resolution was attempted (injected / quick_glance / full_recall) but
   yields zero candidate paths**, emit exactly one record with `note_ref: null`
   and `situation_text: "note path could not be resolved (source data missing or
   resolution failed)"` — the moment is never silently dropped.
3. **If resolution yields one or more candidate paths**, emit one record **per
   distinct path**, all sharing the same `outcome`, `moment_id`, `situation_text`
   (the moment's own `description`), `ts`, and `evidence` — only `note_ref`
   differs across the group.
4. **Outcome tag** (same for every record emitted for this moment under Rule 1):
   - `followed == "yes"` → `applied-helped`
   - `followed == "contradicted"` → `applied-hurt`
   - `followed ∈ {"ignored", "skipped_step", "out_of_order", "partial"}` →
     `surfaced-ignored` if `search_ran` was `quick_glance`/`full_recall`,
     `injected-unused` if `search_ran` was `injected`.

### Rule 2 — absent-needed

Applies only when `moment_type ∈ {failure, rework}`.

- `finding_side.memory_existed == false` → emit one record, `note_ref: null`,
  `situation_text: "no relevant memory existed in the vault at this moment (<EXISTENCE_DERIVED_OR_ESTIMATE_UPPER>)"`.
- `finding_side.memory_existed == true` **and** `finding_side.surfaced == false`
  → emit one record, `note_ref: null`,
  `situation_text: "relevant memory existed in the vault but was not surfaced to the agent (search_ran=<SEARCH_RAN>, <EXISTENCE_DERIVED_OR_ESTIMATE_UPPER>)"`.
- Otherwise (`memory_existed` is null, or `memory_existed == true` and
  `surfaced == true`) → no record.

### Shared fields

- `moment_id` — the moment's own `moment_id`.
- `ts` — the moment's own `timestamp`, copied verbatim.
- `evidence` — `{transcript, anchor}`, copied verbatim from the moment's own
  `evidence` sub-object.
- `situation_text` for Rule 1 records is the moment's own `description` field
  verbatim (unless the record is the "could not be resolved" fallback above).

## Real-run counts (2026-09-02)

Generated from the finalized 150-moment / 568-transcript-event corpus:

| Outcome            | Count |
|---------------------|------:|
| `applied-helped`    |   140 |
| `applied-hurt`       |     0 |
| `surfaced-ignored`   |     1 |
| `injected-unused`    |     0 |
| `absent-needed`      |    27 |
| **Total**            | **168** |

The large `applied-helped` count relative to the number of underlying moments
(16 distinct moments) is expected and is the direct, documented consequence of
the multi-note fan-out in Rule 1 step 3 below — see the first known limitation.
`applied-hurt` and `injected-unused` are both zero in this real corpus: no
moment in the audited sample carries `finding_side.followed == "contradicted"`,
and none of the `search_ran == "injected"` moments with a non-null `followed`
happen to fall into the ignored-like bucket.

## Known limitations

1. **Multiple co-surfaced notes share one verdict.** The audit judges
   "followed" once per *moment*, not once per note. When a moment's recall
   surfaced several notes together (a single `recall_call` can match many
   items, or a fork can inherit several `recalled_notes`), Rule 1 step 3 emits
   one outcome record per distinct note path, but all of them carry the *same*
   outcome tag — there is no way, from the source judgment, to tell which of
   the several surfaced notes specifically drove (or failed to drive) the
   moment's outcome. A moment with 20 co-surfaced notes and a "yes" verdict
   produces 20 `applied-helped` records, one per note, all asserting the note
   helped equally. This is a real artifact of the real corpus, and not a rare
   one: two moments tie for the largest fan-out in the 2026-09-02 run (20 notes
   each, from the same transcript), and 8 of the 16 distinct `applied-helped`
   moments (50%) have fan-out ≥10, together accounting for 130 of the 140
   `applied-helped` records (92.9%).
2. **"Last recall_call at or before this location" is a heuristic, not an
   exact window boundary.** For `search_ran ∈ {quick_glance, full_recall}`
   moments, `moments.jsonl` does not store the exact span of transcript lines
   the judge considered when writing `followed` — only the moment's own
   anchor `location`. The script approximates "the recall search this moment's
   judgment refers to" as the most recent `recall_call` at or before that
   `location` in the same transcript. This is a reasonable proxy (recall
   calls are chronologically ordered ascending by `location` in
   `transcript-events.jsonl`, confirmed for all 568 records) but is not a
   verified exact match to whatever search the human/LLM judge actually had
   in view when scoring `followed`.
3. **Dispatch-type moments are excluded from this file entirely.** Rule 1
   requires `moment_type ∈ {success, failure, rework}` and Rule 2 requires
   `moment_type ∈ {failure, rework}`; `moment_type == "dispatch"` (28 of the
   150 moments) never matches either rule and produces zero outcome records
   here, by design. The dispatch-side handoff gap — whether the orchestrator
   handed a subagent the memory it had — is already computed separately as
   `results/scorecard-handoff-gap.json`'s D1 metric. Re-deriving it here would
   double-count the same underlying evidence under a different schema and
   risk the two disagreeing; outcome-records.jsonl deliberately leaves that
   ground to the scorecard.
4. **`correction`-type moments are also excluded from this file entirely**,
   for the same reason as dispatch moments above: neither rule's `moment_type`
   set includes `correction`, so none of the 6 correction moments in the
   corpus produce a record here (this is a narrower, less consequential
   instance of the same "moment_type gate" as #3, called out separately since
   it isn't already covered by a parallel scorecard computation the way
   dispatch is — corrections simply fall outside both rules' scope per the
   task 5.3 spec as written).

## Reproducing this file

```
cd dev/eval/audit
python3 generate_outcome_records.py
```

Reads `results/moments.jsonl` and `results/transcript-events.jsonl`, writes
`results/outcome-records.jsonl`. Deterministic — no LLM calls, no network
calls, no randomness. The only vault interaction is read-only, inherited from
`_resolve_fork_parent_context` (opens vault note files under
`~/.local/share/engram/vault` to read text back for `search_ran == "injected"`
resolution); the script never writes, amends, activates, supersedes, or
removes any vault note.
