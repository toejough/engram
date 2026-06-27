# Engram roadmap — recall efficiency

Driving recall efficiency for three coupled goals: **experience** (recall feels heavy/slow),
**cost** ($), and **frequency** (let the agent call recall more often and organically). All three
converge on one target — shrink the post-fire recall procedure tax. Levers are ordered by likely
biggest win; **do one at a time**, ship each gated, measure, then take the next.

## Where we are

- Memory's value is **validated and generalizes**: the 4 capability wins (apply-conventions,
  recency-supersession, honor-standard, abduction) hold with zero degradation under a realistic
  200-note crowded vault (2026-06-26). Value is real on idiosyncratic content; cost/speed is the tax.
- **Recall is the tax** (measured): ~150–190s/op, of which **~half (~80–120s at the original ~200 KB) is the agent paging the
  `engram query` payload** (~8 reads; trimmed to ~97 KB by #1+#5, so paging cost scales down); the binary itself is only ~3s. Recall is ~20% of op $
  and ~25% of op time. The bottleneck is the **shape and size of what the binary hands the agent**,
  not the computation.

## Standing constraint (non-negotiable)

Every recall/learn skill change ships **gated by the trap regression harness**
(`dev/eval/traps/gate.py`, run before+after) and **measured by the `recall_cost` `$METER`** (cumulative
harness, schema v5). **Never touch the win-nucleus:** Step-3 conventions-as-requirements directive,
Step-2.5B recency-weight, Step-2 matched-note retrieval, the frontmatter `description` field.

## Efficiency levers (ranked; one at a time)

### #1 — ✅ Compact, lazy-content payload — DONE 2026-06-27
Shipped `--lazy-chunks` (recall's default invocation): matched + recency **chunk** items render
path/source-only; **notes (fact/feedback) keep full content inline** — the win-nucleus is untouched.
The agent fetches a chunk's text on demand via the new `engram show-chunk <source#anchor>`. Measured:
query payload **−33.7%** (146→97 KB, ~36.5K→24.2K tokens) on the 10-phrase baseline; trap gate
**GREEN** (matched notes/clusters untouched); `targ check-full` clean (8/8).
**Net-economics validation (the on-demand-vs-dump risk Joe raised):** across 13 realistic uninstructed
recalls the agent fetched **0** chunks (notes are load-bearing — note 72), so there is no iterative-fetch
tax to trade for; and in 2/2 sole-source fixtures it reached for `show-chunk` on its own and surfaced the
exact fact — no evidence drop. Selection is reliable both ways (sparing when notes suffice, on-target when
a chunk is the only source). Explicit clusters-block reorder assessed marginal (notes already lead by
score) and skipped. Stacks on #5: cumulative payload ~230→97 KB (**~−58%**).

### #2 — Async / non-blocking `learn`  ← NEXT
The closing `/learn` (~61s) runs on the critical path before the result is delivered. Detach it so it
ingests + crystallizes in the background. **Win:** ~61s off perceived latency; low risk (same notes
written, just later — guard a same-session re-recall against the async write).

### #3 — Two-speed recall (the frequency keystone)
Today recall is one heavy 287-line procedure invoked in full every time. Split into a **fast/cheap
"quick recall"** (the binary query + the compact #1 view, minimal procedure) the agent can fire
liberally and organically, reserving the **full** synthesis/linking machinery for moments that
warrant it. **Win:** directly enables "called more often" via affordability, not exhortation.
Composes #1; more architectural — brainstorm the split before building.

### #4 — Inline `candidate_l2` content; kill the blocking round-trips
Step 2.5 currently makes serial blocking `engram show` calls per candidate. Ship that content inline
in the query payload. **Win:** ~15–40s/op, ~3–8 fewer blocking round-trips; low risk (same bytes,
delivered earlier).

### #5 — Cost-cleanup bundle
The smaller, mostly-mechanical trims:
- ✅ **Recent-fill cut — DONE 2026-06-27** (`--recent-fill`, default 200→25). Measured: query payload
  **−28%** (230→165 KB, 426→252 items, recent 205→25), trap gate **GREEN** (matched set untouched, no
  capability regression), `targ check-full` clean. The recency channel is now configurable.
- **payload-prune-after-Step-3** (drop the raw payload from build context after the requirements list
  is synthesized — the ~$1/op $ lever) — open.
- **dedupe the double ingest sweep** — open.

> **Note:** the recent-fill cut was the *safe biggest single* payload reducer, done first. It does
> NOT close **#1** (the matched-set clusters-first/lazy-content restructure) — that remains the next
> structural win (~40-80s) once we decide the −28% slice isn't enough.

## Dead ends (measured — do not revisit)
Payload-size cap *for dollars* (payload is cheap cache_read); whole-op or split **haiku** (−14%, broke
the build half, rolled back); cutting the 10 query phrases (breadth surfaces the un-guessable notes);
lightening the skill *body* to increase firing (firing is set by the `description`, not the body).

## Done
- **Crowded-vault capability eval** (2026-06-26) — the 4 wins generalize to a realistic crowded vault
  (zero degradation @ 200 notes). Bound: *same-domain competing* notes still untested. See
  `dev/eval/traps/{RESULTS.md, README.md}`.
- **Instruments** (2026-06-26) — the `recall_cost` `$METER` (schema v5) + the C3/C4i/C5/C6 trap
  regression gate. These make every lever above safe (regression-caught) and measurable.
