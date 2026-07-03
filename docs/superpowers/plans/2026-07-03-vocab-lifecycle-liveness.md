# Vocab lifecycle liveness — investigation plan

> **For agentic workers:** investigate-and-propose ask. The deliverable is a PROPOSALS doc Joe
> reviews — NO build this round. Measurements are free/cheap; no production changes.

**Ask (Joe, 2026-07-03, condensed):** think end-to-end about keeping the vocabulary live and
effective. Learn isn't the only note writer — if vocab checks must follow every write, recall needs
them too. Joe's lean: learn is the sensible check site (to avoid re-inflating recall's cost), but
the actual impact is unknown. Investigate a few options balancing liveness/effectiveness vs time/$,
and present proposals.

**Measured facts this plan stands on (2026-07-03, all free):**
- `engram vocab stats`: 22 ms, 236 tokens of output — detection cost is negligible ANYWHERE; the
  real placement cost is PROCEDURE INFLATION (skill length / steps), per the recall-economics
  notes (140/141/144: under-firing is the risk; every added recall step erodes glance's 2.23×).
- **Recall is the MAJORITY note-writer:** 73/141 notes (46 via Step 2.5 coverage writes + 27 via
  Step 4 synthesis) vs learn's 55 — so "detection at the write site" means covering recall, and
  any learn-only detection misses half the writes unless detection PERSISTS between sites.
- **The documented +30%-growth refit trigger is mis-tuned:** historical replay fires 8 refits in
  3 weeks (~$16 + tag churn + gate runs). Percentage triggers on a small base run hot; triggers
  need recalibration (absolute growth + minimum interval + utility signals), not just placement.
- Vault growth: 10/35/70/26 notes per week over the last 4 weeks; untagged today 3.6%.

**Key reframe to test (stated at orientation):** detection and action DECOUPLE. If the binary
evaluates thresholds during any vocab-touching write (learn + amend — which recall's writes flow
through) and persists a pending verdict, the write SITE stops mattering: a threshold tripped by a
recall-written note waits, flagged, until the action site next runs. Then Joe's lean (act in learn)
costs nothing in coverage.

## Options to model (detection site × action site × trigger set)

- **O1 — Learn-anchored stateless:** learn's sweep runs `vocab stats` (now emitting a binary-computed
  verdict line); a skill conditional keyed to the observable verdict acts (refit flow / propose).
  Recall untouched. Staleness bound: triggers are slow-moving; learn runs at every /please bracket.
- **O2 — Binary-persistent flag:** the binary evaluates thresholds on EVERY vocab-touching write
  (covers recall's 52% automatically), persists `refit_pending` + reason in index metadata; learn's
  sweep surfaces + acts on it. Optional: the query payload carries a 1-line pending flag (≈5 tokens,
  no instruction change) purely as visibility.
- **O3 — Scheduled autonomous maintenance:** a weekly headless agent (cron/scheduled job) runs
  stats → refit-if-tripped → regression gates → commits nothing (vault-only). Zero skill inflation
  anywhere; full autonomy (Joe's stated interest); harness-scheduling dependency. Not
  relocation-not-reduction (note 108): the check is ~free; the $2 action fires only when tripped.
- **O4 — Recall-side checks:** modeled honestly for completeness, expected REJECTED on the
  economics (procedure inflation × fire frequency) given O2 covers recall's writes without recall
  doing anything.
- **Cross-cutting — trigger recalibration (applies to every option):** replace +30%-growth with
  {absolute growth ≥ N notes since last refit (propose N≈40), minimum interval, untagged-rate
  window with binary-tracked write outcomes, hub threshold unchanged}. Replay each candidate set
  against vault history (free) and report fire counts.

## Cost model per option (the deliverable's comparison table)

Per option: added tokens/steps per learn fire · per recall fire · staleness window · autonomy
(fires without a human/workflow?) · agent-reliability class (observable predicate vs judgment —
note 145) · implementation cost (binary/skill/infra) · $ per month at measured growth (check cost ×
fire frequency + action cost × replayed trip rate).

## Steps

1. Replay recalibrated trigger candidates against vault history (free python).
2. Fill the cost model with measured numbers; stress the common cases end-to-end (a heavy /please
   day with 10 recall writes; an idle week; a new-domain influx driving untagged up).
3. Draft 3–4 proposals with an explicit recommendation + honest bounds; include what each option
   does NOT catch.
4. Proposals doc → `docs/design/2026-07-03-vocab-lifecycle-proposals.md`; Gate C; commit (Gate D);
   PRESENT to Joe and stop.

## Constraints
- No production changes; measurements read-only. Labeled tables with units. Every claim dated.
- The recommendation must respect: observable-predicate reliability (145), recall economics
  (140/141/144), relocation≠reduction (108), fire-unit pinning (109).
