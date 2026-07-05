"""Corpus for the question-anchored distillation delivery eval (design:
docs/design/2026-07-01-question-anchored-distillation.md, deleted 2026-07; git log).

Each entry is a PAIR: an INVESTIGATION (a symptom-framed question + the evidence a real recall would
surface) and a FUTURE question in a *sibling* subsystem where the SAME idiosyncratic rule applies.

Two validity constraints every pair must honor (the eval is worthless without them):

  1. IDIOSYNCRATIC → headroom. The rule uses an invented platform + a specific un-guessable mechanism,
     so a cold agent CANNOT apply it without the note. If cold opus already applies it, there is no
     headroom and no lever can move delivery (measured by the Stage-2 none-floor probe).

  2. DIVERGENT topic vs question. The `evidence` members all name the FIRST subsystem (narrow topic),
     but the `question` names the SYMPTOM/intent. So a faithful topic-anchored note (arm A) pitches at
     the first subsystem, while a question-anchored note (arm B) pitches at the transferable rule.
     `future_q` is a DIFFERENT subsystem/operation on the same platform — transfer succeeds only if the
     note generalized past the first subsystem. If A already generalizes (opus is a strong distiller),
     A ≈ B and the lever is null: that is an honest finding the eval is built to catch.

`E` is the required principle-application the blind judge checks — the reasoning a HIT answer must show,
tracking the PATTERN, never a note name or vocabulary (bias guard, note on scorer-vocabulary bias).
"""

PAIRS = {
    "vorl-slot": {
        "principle": "Vorl shared structures have no internal lock; claimSlot() before any concurrent write or entries interleave.",
        "question": "Why do writes to the Vorl dispatch queue sometimes interleave with another worker's and corrupt the batch?",
        "evidence": [
            "Incident VZ-88: two workers enqueued to the Vorl dispatch queue at the same time and the entries interleaved, corrupting the batch.",
            "Fix for VZ-88: call claimSlot() before any enqueue on the Vorl dispatch queue — it serializes the write.",
            "The Vorl dispatch queue has no internal lock; claimSlot() is the only thing preventing concurrent-write interleave.",
        ],
        "future_q": ("I'm wiring up a new Vorl telemetry buffer that several collectors append to at "
                     "the same time. Anything Vorl-specific I should handle before I ship it?"),
        "E": ("Warns that on Vorl any shared structure written by concurrent producers has no internal "
              "lock, so you must call claimSlot() BEFORE the append or entries silently interleave/"
              "corrupt — and applies that to the telemetry buffer's concurrent collectors rather than "
              "treating it as a fresh problem."),
    },
    "quill-freeze": {
        "principle": "Quill: `quill freeze` the replica cursor before ANY schema change (not just drops), else replicas desync.",
        "question": "Why does my Quill read-replica desync whenever I drop a column on the primary?",
        "evidence": [
            "Quill replicas desynced after a column drop until we started running `quill freeze` first.",
            "`quill freeze` snapshots the replica cursor so a schema change doesn't race the replication stream; run it before dropping a column, then thaw.",
            "The Quill runbook rule: freeze before a column drop — a schema change that races replication corrupts the replica.",
        ],
        "future_q": ("I need to RENAME a column on our Quill primary next week. Replicas are downstream. "
                     "What's the safe procedure?"),
        "E": ("Says to `quill freeze` before the rename — any schema change that races the replication "
              "stream, not only drops — then thaw, generalizing the freeze-before-schema-change rule "
              "from column-drop to column-rename rather than treating rename as unaffected."),
    },
    "zephyr-shard": {
        "principle": "Zephyr cache is shared across tenants and does no isolation; the tenant-shard key prefix is the only boundary.",
        "question": "Why am I occasionally getting another tenant's data back from the Zephyr cache?",
        "evidence": [
            "Cross-tenant leak in the Zephyr cache traced to data keys missing the tenant-shard prefix.",
            "Every Zephyr cache key for tenant data must be prefixed with the tenant-shard id; the cache is shared across tenants.",
            "The Zephyr cache does no tenant isolation itself — the shard prefix in the key is the only boundary.",
        ],
        "future_q": ("I'm adding a new Zephyr rate-limiter that stores per-user request counters in the "
                     "same Zephyr cache. How should I key the counters?"),
        "E": ("Requires the tenant-shard prefix on the rate-limiter's counter keys because the Zephyr "
              "cache is shared and does no isolation (the prefix is the only boundary), applying the "
              "key-prefix rule to the counters — not only the original data-cache case."),
    },
    "grix-idempotency": {
        "principle": "Grix has at-least-once delivery; every mutating call needs a client idempotency-key or it can double-apply.",
        "question": "Why does retrying a failed Grix payment sometimes double-charge the customer?",
        "evidence": [
            "Double-charge bug: Grix payment calls retried without an idempotency token get processed twice.",
            "Grix mutating calls must carry a client-generated idempotency-key header; Grix dedups on it.",
            "Grix has at-least-once delivery, so any mutating call can run twice unless it is idempotency-keyed.",
        ],
        "future_q": ("I'm adding a Grix call that issues account credits (refunds). Our job runner "
                     "retries it on failure. What do I need to get right?"),
        "E": ("Requires a client-generated idempotency-key on the refund/credit call because Grix is "
              "at-least-once and any mutating call can double-apply, applying the idempotency-token rule "
              "to credits — not only to payments."),
    },
    "morp-normalize": {
        "principle": "Morp timestamps are planet-local with no offset; call morp.normalize() before ANY comparison/sort.",
        "question": "Why do my Morp schedule comparisons fire at the wrong time?",
        "evidence": [
            "Morp schedule bug: comparing two Morp timestamps directly misfired because Morp stores times as planet-local, not UTC.",
            "Always call morp.normalize() before comparing Morp timestamps — it converts planet-local into a common frame.",
            "Morp timestamps carry no offset; the raw planet-local value is meaningless until normalized.",
        ],
        "future_q": ("I'm building a Morp feature that sorts events into a timeline and dedups any two "
                     "within 5 minutes of each other. Any Morp time gotcha?"),
        "E": ("Calls morp.normalize() before sorting/diffing the Morp timestamps because they are "
              "planet-local with no offset (meaningless raw), applying the normalize-before-compare rule "
              "to sorting/dedup — not only the original schedule comparison."),
    },
    "fenn-seal": {
        "principle": "Fenn doesn't track generated files in its dep graph; `fenn seal` after every codegen is the only staleness guard.",
        "question": "Why do stale generated files sometimes ship out of our Fenn build?",
        "evidence": [
            "Fenn shipped stale generated code because `fenn seal` wasn't run after the codegen step.",
            "`fenn seal` re-hashes generated artifacts so the build fails if they're out of date; run it after every codegen step.",
            "Fenn does not track generated files in its dependency graph — seal is the only staleness check.",
        ],
        "future_q": ("I'm adding a new Fenn step that generates API client stubs from a schema. How do I "
                     "make sure a stale stub can never ship?"),
        "E": ("Runs `fenn seal` after the stub codegen because Fenn doesn't track generated files and "
              "seal is the only staleness guard, applying the seal-after-codegen rule to the new stub "
              "generator — not treating it as untracked-but-safe."),
    },
    "dax-refresh": {
        "principle": "Dax service tokens expire at 6h with no auto-renew; long jobs must call dax.refresh() mid-run.",
        "question": "Why do my long-running Dax export jobs 401 near the end even though they start fine?",
        "evidence": [
            "Long Dax export jobs 401'd near completion — Dax service tokens expire after 6 hours.",
            "Call dax.refresh() periodically inside any job that runs longer than 6h; the initial token won't last.",
            "Dax does not auto-renew tokens; a process holding one token for hours will hit expiry mid-run.",
        ],
        "future_q": ("I'm writing a Dax backfill that streams for maybe 10 hours. It authenticates once "
                     "at startup. Will that hold up?"),
        "E": ("Warns the single startup token won't hold because Dax tokens expire at 6h with no "
              "auto-renew, and requires periodic dax.refresh() inside the backfill loop, applying the "
              "mid-job-refresh rule to the backfill — not assuming the startup auth suffices."),
    },
    "plim-micros": {
        "principle": "Plim money is int64 micros to avoid float rounding; never do float arithmetic on it.",
        "question": "Why do our Plim ledger totals drift by fractions of a cent?",
        "evidence": [
            "Plim totals drifted because someone converted amounts to float for arithmetic; Plim stores money as integer 'micros'.",
            "Never convert Plim micros to float for math — sum in integer micros and convert only for display.",
            "Plim amounts are int64 micros precisely to avoid float rounding; any float op reintroduces the error.",
        ],
        "future_q": ("I'm adding a Plim feature that splits a bill evenly across N people and shows each "
                     "person's share. How should I compute the shares?"),
        "E": ("Computes the split in integer micros (integer divide plus distribute the remainder), "
              "never floating the division, because Plim uses int micros precisely to avoid rounding "
              "drift — applying the integer-money rule to the bill-split."),
    },
    "wynn-reopen": {
        "principle": "Wynn readers pin a segment at open time; call wynn.reopen() to see any write made after open.",
        "question": "Why don't my Wynn searches see documents right after a bulk import?",
        "evidence": [
            "After a Wynn bulk import, queries kept returning the old results until we called wynn.reopen().",
            "wynn.reopen() swaps the searcher to the newest segment; bulk writes aren't visible until you do.",
            "Wynn readers pin a segment at open time; without a reopen they never see post-open writes.",
        ],
        "future_q": ("I'm adding a Wynn feature where a user edits a doc and immediately searches for it. "
                     "They report the edit doesn't show up. Why, and what fixes it?"),
        "E": ("Explains Wynn readers pin a segment at open time so the edit isn't visible until "
              "wynn.reopen() swaps to the new segment, and requires reopen after the edit — applying the "
              "reopen-after-write rule to the single-doc edit, not only bulk import."),
    },
    "korl-seq": {
        "principle": "Korl sorts by stamp; independent per-producer counters aren't globally monotonic — all producers draw from korl.seq().",
        "question": "Why do events in our Korl log sometimes sort out of order across producers?",
        "evidence": [
            "Korl events sorted wrong when two producers each stamped their own local sequence numbers.",
            "All Korl producers must draw stamps from the central korl.seq() allocator; independent local counters collide.",
            "Korl sorts purely by stamp; per-producer counters are not globally monotonic across producers.",
        ],
        "future_q": ("I'm adding a SECOND writer to our Korl audit stream (currently single-writer). It "
                     "will assign its own incrementing ids. Any problem with that?"),
        "E": ("Warns that a second writer with its own local counter breaks Korl's global ordering "
              "(Korl sorts by stamp; independent counters aren't monotonic across producers) and "
              "requires both writers draw from the central korl.seq() allocator, applying the "
              "central-allocator rule to the new audit writer."),
    },
}
