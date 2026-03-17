# Engram Roadmap

## Current State (2026-03-17)

Evolution plan phases complete through A-3. B-1 in progress (graph + merge packages exist).

| Phase | Status |
|-------|--------|
| A-1: Simplification | ✅ Complete |
| A-2: Foundation | ✅ Complete |
| A-3: High-Impact Fixes | ✅ Complete |
| B-1: Graph + Evolution Core | 🔄 In progress |
| B-2: Integration | ⏳ Blocked (see Cycle 2) |

---

## Cycle 1 — Correctness ✅ Complete

| Issue | What | Status |
|-------|------|--------|
| [#311](https://github.com/toejough/engram/issues/311) | Evaluator persists to `evaluations/` | ✅ Closed (T-108, T-109, commit 1a38b71) |
| [#312](https://github.com/toejough/engram/issues/312) | stop.sh async: per-turn log isolation | ✅ Closed (T-345, ARCH-81, commit 2b952f9) |
| [#317](https://github.com/toejough/engram/issues/317) | Atomic evaluation log write | ✅ Closed (resolved by #311 — temp+rename in persistEvaluationLog) |
| build | reorder-decls + coverage | ✅ Done |

---

## Cycle 2 — Test Hygiene (current)

All independent; can run in parallel.

| Issue | What | Notes |
|-------|------|-------|
| [#314](https://github.com/toejough/engram/issues/314) | Resolve top-N limit (spec says 5, code has 2) | Needs decision first — what is the authoritative limit? |
| [#320](https://github.com/toejough/engram/issues/320) | T-117: runEvaluate end-to-end test | Unblocked by #311 |
| [#319](https://github.com/toejough/engram/issues/319) | T-103: spec contradiction + surfacer integration test | Spec fix + one new test |
| [#313](https://github.com/toejough/engram/issues/313) | T-158: hooks.json structure test | New test in hooks_test.go |
| [#315](https://github.com/toejough/engram/issues/315) | T-120: add stop.sh to evaluate-invocation test | One-line fix |
| [#316](https://github.com/toejough/engram/issues/316) | T-165: frecency formula exact assertion | Replace `> 0` with numeric check |
| [#318](https://github.com/toejough/engram/issues/318) | REQ-22 AC(2): RecordSurfacing field preservation | New round-trip test |
| [#321](https://github.com/toejough/engram/issues/321) | testFS helper in makeTestEvaluator | Premortem P1 — prevents rework in Cycle 3 |

---

## Cycle 2/3 Boundary — Premortem Mitigations

Complete before starting B-2 work. These prevent silent failure at scale.

| Issue | What | Why |
|-------|------|-----|
| [#322](https://github.com/toejough/engram/issues/322) | Binary smoke test | CLI wiring bugs invisible without end-to-end run |
| [#323](https://github.com/toejough/engram/issues/323) | Statement coverage floor | Per-function metric masks 4.2% statement coverage in evaluate |

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
