# Engram roadmap — recall efficiency

Driving recall efficiency for three coupled goals: **experience** (recall feels heavy/slow),
**cost** ($), and **frequency** (let the agent call recall more often and organically). All three
converge on one target — shrink the post-fire recall procedure tax. Levers are ordered by likely
biggest win; **do one at a time**, ship each gated, measure, then take the next.

## Where we are

- Memory's value is **validated and generalizes**: the 4 capability wins (apply-conventions,
  recency-supersession, honor-standard, abduction) hold with zero degradation under a realistic
  200-note crowded vault (2026-06-26). Value is real on idiosyncratic content; cost/speed is the tax.
- **Recall is the tax** (measured): ~150–190s/op, of which **~half (~80–120s) is the agent paging a
  ~200KB `engram query` payload** (~8 reads); the binary itself is only ~3s. Recall is ~20% of op $
  and ~25% of op time. The bottleneck is the **shape and size of what the binary hands the agent**,
  not the computation.

## Standing constraint (non-negotiable)

Every recall/learn skill change ships **gated by the trap regression harness**
(`dev/eval/traps/gate.py`, run before+after) and **measured by the `recall_cost` `$METER`** (cumulative
harness, schema v5). **Never touch the win-nucleus:** Step-3 conventions-as-requirements directive,
Step-2.5B recency-weight, Step-2 matched-note retrieval, the frontmatter `description` field.

## Efficiency levers (ranked; one at a time)

### #1 — Restructure the query output to a compact, one-read payload  ← NEXT
The single highest-leverage lever. Today `engram query` emits a ~200KB blob the agent pages ~8×.
Emit a compact, clusters-first / lazy-content view so the load-bearing content arrives in **one
read**, deferring bulk chunk text to on-demand. **Win:** ~40–80s/op (time + experience), kills the
~$1 per-build-turn re-read premium, and a lighter payload is what makes frequent calls affordable.
**Risk (med):** must not drop the matched-note content the wins depend on — gate hard. Binary change
(query output) + recall-skill change (how it consumes the payload).

### #2 — Async / non-blocking `learn`
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
The smaller, mostly-mechanical trims: **payload-prune-after-Step-3** (drop the raw payload from build
context after the requirements list is synthesized — the ~$1/op $ lever), **cut `recentFillChunks`
200→~25** (non-win-bearing Channel-2 bloat), **dedupe the double ingest sweep**. **Win:** ~$1/op +
smaller payload; low risk.

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
