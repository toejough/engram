# endorse_cue — headless ambient-cue-firing probe

Measures whether `guidance/recall.md` fires `/recall` at the moment an agent is about to
**endorse, rank, or recommend a proposal** — see `dev/eval/LEDGER.md#endorse-cue-red-baseline-null`
for the RED-baseline finding this harness produced (the originating plan doc was scratch and
has been deleted; its finding now lives in that LEDGER row).

## Design

Described-first-action, no-tools (recovered pattern, note 139/251 —
`git show c0b00b1c:docs/design/2026-06-29-recall-moments-revalidation-data/run_revalidation.sh`).
Each trial is a fresh `claude -p`, given one fictional fixture, constrained to respond with
exactly two lines and call no tools:

```
MARKER: <token>
ACTION: <one sentence>
```

- **`fired`** = the `ACTION` line names a recall (`/recall`, `engram query`, or a
  recall-verb + prior-work referent). `None` first action counts as not-fired.
- **`marker_seen`** = the run echoed the unique per-run marker that was embedded in the
  inlined guidance. `marker_seen=False` trials are **discarded**, never scored — an
  unloaded-guidance run is a validity failure, not a data point (verify-treatment-delivery).

The recall.md variant is **inlined verbatim into the trial's project `CLAUDE.md`**, never
`@import`ed — a fresh temp project's first external `@import` triggers Claude Code's
one-time approval dialog, which a headless run with no TTY silently fails, loading nothing
in *either* arm (a false null indistinguishable from a real one). Inlining plus the marker
closes that hole.

## Fixtures

`fixtures/*.txt` — 3 fictional endorse/rank/recommend scenarios (`endorse_global_mutex`,
`rank_polling_replace_streams`, `recommend_sync_auditlog`), each keyed to one of the cue's
concrete positive trigger phrases ("highest-leverage", "NOW #1", "want me to pick it up?")
and to a proposal whose mechanism is a classic anti-pattern in its fictional codebase
(global-mutex serialization, hand-rolled polling reinventing Streams consumer groups, a
synchronous write in a hot request path) — fictional so the model's training priors can't
substitute for an actual recall action.

## Usage

```
python3 probe.py --recall-md <path-to-recall.md-variant> --n 5 --out results.jsonl \
    [--model opus] [--fixtures fixtures/] [--workers 4]
```

- `--recall-md` — RED = `../../../guidance/recall.md` (current); GREEN = the edited variant
  (Task 3). This is the ONLY variable between arms.
- `--model` — defaults to `opus`, the target model this guidance runs on; do not swap in a
  weaker model as a stand-in.
- Output JSONL: one record per trial —
  `{fixture, idx, marker_seen, fired, first_action, raw_result, cost, num_turns, sid}`.
  `fired` is `null` for discarded (marker_seen=false) trials.

Scoring the fire-rate: `fired_n / len([r for r in results if r["marker_seen"]])` — never
divide by the raw trial count; report discards separately (a validity issue, not a 0).

## Scope note

This directory implements Task 1 of the plan only: the probe + fixtures, smoke-tested for
instrument validity against the *current* `guidance/recall.md`. The pre-registered
RED/GREEN/pressure runs (Task 2/3) and the ship decision (Task 4) are separate,
orchestrator-gated steps — this script does not run them itself.

## Status (2026-07-15)

The harness is **KEPT** as reusable ambient-cue-firing infra — it works end-to-end and is
cheap to re-run against a new `guidance/recall.md` variant. But it runs a **clean single-shot
context** in which the inlined guidance is maximally salient (no competing turns, no other
in-flight work), so it **over-fires** relative to real sessions and **cannot reproduce**
context-emergent under-load firing failures — the actual #677 failure mode (recall not firing
when it's buried mid-task, not the opening move). A future #677 validation harness needs to
reproduce multi-turn load, not a fresh single-prompt process, before it can speak to that
failure mode.
