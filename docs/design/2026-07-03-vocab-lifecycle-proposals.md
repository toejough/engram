# Vocab lifecycle liveness — proposals

**Status:** proposals for Joe's decision — NO build this round.
**Ask (Joe, 2026-07-03):** keep the vocabulary live and effective, balanced against time/$ —
learn feels like the sensible check site (don't re-inflate recall), but the actual impact is
unknown; investigate options and present proposals.
**Evidence:** `dev/eval/vocab/trigger_replay.py` + `trigger_replay_results.md` (commit c57d08e1).
"Replayed" = each candidate trigger retroactively applied, day by day, to the vault's real
141-note write history (2026-06-12 → 2026-07-03, 22 days) — real fire counts, not simulation.
Every number below is labeled **measured / estimated / projected / replayed**.

**Decisions needed (detail at the end):** (1) option pick — O2 recommended, O1 fallback;
(2) trigger numbers — 40 notes/14 days recommended, 30/7 for weekly freshness; (3) riders —
immediate: resituate fix + one metered refit; deferred: doc updates + re-replay at 30+ days.

## TL;DR — the finding that reframes the question

**Check placement is cost-irrelevant. Trigger calibration is the entire cost lever.**

- The liveness CHECK costs ~nothing anywhere: ~237 tokens (~$0.0004) per fire in an agent
  context (measured output size, estimated pricing); in-process it is a token-free Go
  comparison, <$0.0001 per fire (code-verified: no LLM call). The placement difference between
  learn / recall / binary / cron is **~$0.01/month** (estimated).
- The ACTION (refit + gates) costs **~$4–5 per fire** (refit ~$1–2 estimated from token
  arithmetic — never metered; trap smoke $3.09 measured,
  `docs/design/2026-07-03-vocab-notes-build-results.md` slice-3 gates table).
- **The headline delta:** the shipped trigger set fires a refit **every ~3 days** at observed
  growth (replayed: 6 fires/22d, median gap 3d, no interval floor) → **~$33–41/month**. The
  recommended calibration (growth ≥40 notes AND ≥14d) fires **~1.4×/month** → **~$5–7/month**
  (replayed rate, estimated $) — an ~80% cost cut from trigger tuning alone, independent of
  where the check lives.

So Joe's instinct "don't inflate recall" is right — but on procedure-complexity grounds
(notes 140/141: recall's step count is the protected asset), not $ grounds. And the real money
question is "how often does a refit fire," which is decided by the trigger set, not the check site.

## Options, side by side

| Criterion | O0 recalibrate-only | O1 learn-anchored | **O2 binary flag (rec.)** | O3 scheduled | O4 recall-side |
|---|---|---|---|---|---|
| Detection runs at | never (human asks) | each learn run | **every hooked write** | cron (weekly) | each recall run |
| Detection staleness (events) | unbounded | 0–30+ writes (heavy day: 10+) | **0** | up to a week of writes | ~0–5 writes |
| Surfacing latency | manual | closing learn | **next query (≈5-token payload flag) + learn verdict** | next cron tick | same recall |
| Added tokens/fire | 0 | ~237/learn run (meas. output size) | ~0 + 5/query when pending (meas. field size) | 0 in-agent | ~237/recall run (meas. output size) |
| $/month, active growth (est., at 40/14 triggers) | $0 (no action fires) | ~$5–7 | **~$5–7** | ~$5–7 + infra | ~$5–7 |
| $/month, idle week | $0 | ~$0.01 | $0 | $0 | ~$0.01 |
| Binary work (est. LOC) | 0 | ~50–100 (verdict line) | **~100–150** (verdict + hooks + flag) | ~50–100 + external cron | ~50–100 (reuses verdict) |
| Skill edits (each = writing-skills TDD) | 0 | learn | **learn** | 0 | recall (deep + glance paths) |
| Covers deep-recall-only sessions (no learn run) | no | no — waits for next learn | **yes (detect at write; surface next query)** | yes (eventually) | yes |
| Covers resituate writes | no (hole, all options — see Riders) | no | no | no | no |
| Reliability class (note 145) | human memory | observable predicate | **observable predicate** | observable predicate | observable predicate |
| Autonomy tension | — | none | none | ROADMAP:99 hooks-rejection adjacency — needs Joe's explicit sign-off | none |
| **Rating** | PARK (absorbed) | CONTENDER (fallback) | **CONTENDER (recommended)** | PARK | PARK |

### O2 — binary-persistent flag (recommended)

Threshold evaluation in-process at the two existing assignment call sites
(`applyVocabAssignmentAfterLearn` learn.go:199, `applyVocabAssignmentAfterAmend` amend.go:285 —
these already see every learn AND recall-origin write). When a trigger trips, persist
`refit_pending` + reason in `vocab.centroids.json` (machine-maintained, survives `embed apply`,
versioned). Surfacing is dual: a ≈5-token `refit_pending` field in every query payload (verified
against queryPayload — visible to any agent and to Joe), and a net-new verdict line in `vocab
stats` that learn's sweep reads; the learn skill gains one observable-predicate conditional
("verdict says REFIT_PENDING → run the refit flow"). Flag clears when the action runs.
**Skip the windowed ring-buffer variant** (the heaviest sub-piece) — the vault-wide untagged
rate is nearly free in-process and catches the same influx scenario (analytical; axis
unreplayable).
**Does NOT catch:** resituate writes (hook bypass — rider below); drift with zero writes
(impossible — drift requires writes).

### O1 — learn-anchored stateless (fallback)

Learn's sweep runs `vocab stats` (+ net-new verdict line, ~50–100 LOC); the learn skill
conditional acts on it. Zero write-path changes. Staleness window modeled honestly: on a heavy
/please day (replayed peaks: 22 notes on Jun-17, 26 on Jun-28), 10+ recall-origin writes
accumulate for ~hours before the closing learn — immaterial for slow-moving aggregate triggers.
The real gap is **deep-recall-only sessions**: writes land, detection waits for whenever the
next learn runs (could be days).
**Does NOT catch:** anything between learn runs; resituate; sessions that never learn.

### O0 — recalibrate only (parked, absorbed)

Fix the trigger numbers, document a cadence, change no code. It answers "is the trigger the
whole problem?" — replay says the shipped set over-fires 6×/22d, so recalibration is necessary —
but detection stays manual: nothing ever fires unless someone remembers to run `vocab stats`.
That's the exact status-quo problem that prompted this ask. **The recalibration itself is folded
into every other option.**

### O3 — scheduled autonomous (parked)

A weekly cron/launchd job runs stats → refit-if-tripped → gates. Autonomy buys nothing here:
drift requires writes, and writes come from agent workflows that already pass through hooks
(O2) or end in learn (O1). Costs: external scheduling infra (none exists in-repo), and it
re-opens an autonomy-adjacent decision (ROADMAP:99 hooks rejection — distinguishable as
background GC, but Joe's sign-off was a gate criterion). Revisit if vault writes ever decouple
from agent workflows.

### O4 — recall-side checks (parked, by modeling — not pre-judged)

Modeled with the same numbers as every option: cost-equivalent (~$5–7/month at 40/14 triggers;
check overhead ~$0.01/month), detection gain over O1 is hours-scale on heavy days. It is
**dominated by O2**, which gets strictly better staleness (0 events) with zero recall-skill
surface — O4 instead adds a step to the procedure notes 140/141 protect and requires
writing-skills TDD across BOTH recall paths (deep + glance). The rating followed the modeling.

## Trigger calibration — the actual money lever (replayed)

Event series: 141 notes over 22 days, bursty (peaks 22 and 26 notes/day; weekly note counts
35, 42, 61, 3 by rolling 7-day window from the first note Jun-12 — the final window is the
partial day Jul-03 — superseding the plan's pre-replay 10/37/75/19, which used 7-day windows
anchored at Jun-9 rather than the first-note date; both sum to 141 notes).

| Trigger candidate | Fires in 22d (replayed) | Median gap (days) | Est. $/month at observed pace | Note |
|---|---|---|---|---|
| Shipped (c): +30% growth, no floor | 6 | 3.0 | ~$33–41 | hot at any threshold — % triggers on a small base |
| +30% growth + 7d floor | 3 | 7.0 | ~$16–20 | same fire dates as absolute 30/7 |
| Absolute ≥30 notes + 7d | 3 | 7.0 | ~$16–20 | weekly refits during active work |
| **Absolute ≥40 notes + 14d (recommended)** | **1** | — | **~$5–7** | ~biweekly-monthly at observed pace |
| Absolute ≥50 notes + 14d | 1 | — | ~$5–7 | indistinguishable from 40/14 in 22d of history |
| Any × 30d interval | 0 | — | $0 | **no data — a 30-day cycle cannot complete in 22d of history; not evidence for or against (rider 4)** |

Conjunct co-occurrence (note 161): all 7d/14d cells genuinely co-fire (growth-ready +
interval-ready overlap; e.g., 40/14: 1 joint fire, 8 growth-ready-but-interval-blocked days,
0 interval-ready-but-growth-blocked). The 30d cells never co-fire — structurally, not
calibration-wise. Keep alongside: vault-wide untagged >8% (replaces shipped (a); catches
new-domain influx — analytical, axis unreplayable since tags were backfilled at migration) and
hub >25% (kept; no growth dependence).

## Common cases (stress, from the plan)

| Case | O1 | O2 | O4 |
|---|---|---|---|
| Heavy /please day (replayed peaks 22–26 notes) | detect at closing learn (~hours stale) | detect at the crossing write; surfaced on every query from then on (zero-latency within session) | detect at next recall (~hours) |
| Idle week (analytical — no idle week exists in the 22d history; nearest sample: Jul-03 partial day = 3 notes) | ~$0 | $0 | ~$0 |
| New-domain influx (analytical: 6 notes/day × 7d) | day-7 learn | day-5 write sets flag (growth); day-1–2 via untagged >8% | day-5 recall |

## Riders (apply regardless of option — executed on Joe's go)

**Immediate — ride along with the chosen build:**

1. **Fix the resituate hole:** `resituate` re-embeds a note's sidecar but never refreshes vocab
   (code-verified: zero assignment calls in resituate.go/supersedes.go). Small binary change:
   call the assignment path after resituate's rewrite.
2. **Meter one real refit** before trusting the fire-rate math — the ~$1–2/refit figure is token
   arithmetic, never measured. One metered run replaces the estimate.

**Deferred — after the build lands / after time passes:**

3. **Doc updates when recalibration lands:** ROADMAP Track-A integration line + the build-results
   "What remains" section (documents the shipped trigger set verbatim — stale the day the
   numbers change).
4. **Re-replay after 30+ days of history** — the 30d-interval column and any seasonal pattern
   are unanswerable today.

## Recommendation

**O2-lite + recalibrated triggers** — in-process checks at the two existing hook sites with
{absolute growth ≥40 AND ≥14d, vault-wide untagged >8%, hub >25%}, `refit_pending` persisted in
`vocab.centroids.json`, ≈5-token payload flag, learn-skill conditional as the actor. No ring
buffer, no scheduler, no recall changes.

O2 and O1 are cost-identical (~$5–7/month at 40/14, estimated); O2 wins on three non-$ grounds:

1. **0-event detection staleness** vs O1's hours-long window on heavy /please days (replayed
   peaks: 22–26 notes/day) — the flag is set on the crossing write, and every subsequent query
   that session surfaces it.
2. **Coverage of deep-recall-only sessions** — recall-origin amend/learn writes in a session
   that never runs the learn skill; O1's detection waits days for the next learn run.
3. **Surfacing without acting-agent dependence** — the ≈5-token payload flag reaches any agent
   (and Joe) on the next query, whether or not a learn skill ever fires.

**O1 is the fallback** if zero write-path changes is the priority: same $, same triggers, same
verdict line — with an hours-to-days staleness window that slow-moving aggregate triggers
mostly forgive.

Decision needed from Joe: (1) option pick — O2 recommended, O1 fallback; (2) trigger numbers —
40/14 recommended, 30/7 if you want weekly freshness during active stretches; (3) riders —
whether immediate riders 1–2 (resituate fix, metered refit) ride along with the build.

## Metered refit result (2026-07-03)

per-refit cost: $0.0857 (measured 2026-07-03, dedicated headless derivation; binary phases ~$0)
wall-clock: ~9s (phases A–C; Phase B derivation 9s, binary phases A+C <1s each)
