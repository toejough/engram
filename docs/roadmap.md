# Engram Roadmap

## Current State (2026-03-17)

Evolution plan phases complete through A-3. B-1 in progress (graph + merge packages exist).

| Phase | Status |
|-------|--------|
| A-1: Simplification | ✅ Complete |
| A-2: Foundation | ✅ Complete |
| A-3: High-Impact Fixes | ✅ Complete |
| B-1: Graph + Evolution Core | 🔄 In progress |
| B-2: Integration | ⏳ Blocked (see Cycle 1) |

---

## Cycle 1 — Correctness (current)

**Do not start B-2 until this cycle is complete.** P4-full's effectiveness gating and
P5-full's link recompute both depend on evaluations/ having real data. Running B-2
against empty evaluations/ produces unmeasurable behavior.

| Issue | What | Why |
|-------|------|-----|
| [#311](https://github.com/toejough/engram/issues/311) | Evaluator never persists to `evaluations/` | Entire effectiveness pipeline is silent no-op without this. Unblocks #317, #320. |
| [#312](https://github.com/toejough/engram/issues/312) | stop.sh `"async": true` vs ARCH-42 ordered phases | Data loss risk on long sessions. Needs product decision: sync or async? |
| build | Fix `reorder-decls` (1 file) + coverage below 80% | Keep build green. |

---

## Cycle 2 — Test Hygiene

All independent; can run in parallel after Cycle 1.

| Issue | What | Notes |
|-------|------|-------|
| [#314](https://github.com/toejough/engram/issues/314) | Resolve top-N limit (spec says 5, code has 2) | Needs decision first — what is the authoritative limit? |
| [#317](https://github.com/toejough/engram/issues/317) | REQ-28 AC(4): atomic evaluation log write | Unblocked by #311 |
| [#320](https://github.com/toejough/engram/issues/320) | T-117: runEvaluate end-to-end test | Unblocked by #311 |
| [#319](https://github.com/toejough/engram/issues/319) | T-103: spec contradiction + surfacer integration test | Spec fix + one new test |
| [#313](https://github.com/toejough/engram/issues/313) | T-158: hooks.json structure test | New test in hooks_test.go |
| [#315](https://github.com/toejough/engram/issues/315) | T-120: add stop.sh to evaluate-invocation test | One-line fix |
| [#316](https://github.com/toejough/engram/issues/316) | T-165: frecency formula exact assertion | Replace `> 0` with numeric check |
| [#318](https://github.com/toejough/engram/issues/318) | REQ-22 AC(2): RecordSurfacing field preservation | New round-trip test |

---

## Cycle 3 — B-2 + Backlog Features

Resume the evolution plan, now that effectiveness data is real.

| Work | What | Maps to |
|------|------|---------|
| P4-full | Cluster dedup + cross-source suppression + transcript suppression | Evolution plan B-2 |
| P5-full | Re-compute links after merge | Evolution plan B-2 |
| [#305](https://github.com/toejough/engram/issues/305) | UC-34: TF-IDF as secondary duplicate detection signal | Natural fit post-B-2 dedup work |
| [#309](https://github.com/toejough/engram/issues/309) | Unify memory management across clear/compact/restart | Needs new UC before speccing |

---

## Backlog (unscheduled)

| Issue | What | Blocker |
|-------|------|---------|
| [#214](https://github.com/toejough/engram/issues/214) | Detect memories redundant with skill/plugin behavior | Maps to P4-full; schedule after Cycle 3 |
