# Engram Roadmap

## Current State (2026-03-18)

Evolution plan phases complete through B-2. Between Cycle 3 (complete) and Cycle 4 (not started).

| Phase | Status |
|-------|--------|
| A-1: Simplification | ✅ Complete |
| A-2: Foundation | ✅ Complete |
| A-3: High-Impact Fixes | ✅ Complete |
| B-1: Graph + Evolution Core | ✅ Complete |
| B-2: Integration | ✅ Complete (Cycle 3) |

---

## Cycle 1 — Correctness ✅ Complete

| Issue | What | Status |
|-------|------|--------|
| [#311](https://github.com/toejough/engram/issues/311) | Evaluator persists to `evaluations/` | ✅ Closed (T-108, T-109, commit 1a38b71) |
| [#312](https://github.com/toejough/engram/issues/312) | stop.sh async: per-turn log isolation | ✅ Closed (T-345, ARCH-81, commit 2b952f9) |
| [#317](https://github.com/toejough/engram/issues/317) | Atomic evaluation log write | ✅ Closed (resolved by #311 — temp+rename in persistEvaluationLog) |
| build | reorder-decls + coverage | ✅ Done |

---

## Cycle 2 — Test Hygiene ✅ Complete

| Issue | What | Status |
|-------|------|--------|
| [#314](https://github.com/toejough/engram/issues/314) | Resolve top-N limit (spec says 5, code has 2) | ✅ Closed (top-N canonically 2 per REQ-P4e-4; specs + test updated) |
| [#320](https://github.com/toejough/engram/issues/320) | T-117: runEvaluate end-to-end test | ✅ Closed (commit 8f7da4d) |
| [#319](https://github.com/toejough/engram/issues/319) | T-103: spec contradiction + T-346 surfacer test | ✅ Closed (spec fixed + T-346 test added) |
| [#313](https://github.com/toejough/engram/issues/313) | T-158: hooks.json structure test | ✅ Closed (commit 4dfef9a) |
| [#315](https://github.com/toejough/engram/issues/315) | T-120: add stop.sh to evaluate-invocation test | ✅ Closed (commit 8f7da4d) |
| [#316](https://github.com/toejough/engram/issues/316) | T-165: frecency formula exact assertion | ✅ Closed (commit 7991206) |
| [#318](https://github.com/toejough/engram/issues/318) | REQ-22 AC(2): RecordSurfacing field preservation | ✅ Closed (commit 6d27d84) |
| [#321](https://github.com/toejough/engram/issues/321) | testFS helper in makeTestEvaluator | ✅ Closed (commit 8f08613) |

---

## Cycle 2/3 Boundary — Premortem Mitigations ✅ Complete

Complete before starting B-2 work. These prevent silent failure at scale.

| Issue | What | Why |
|-------|------|-----|
| [#322](https://github.com/toejough/engram/issues/322) | Binary smoke test | CLI wiring bugs invisible without end-to-end run |
| [#323](https://github.com/toejough/engram/issues/323) | Statement coverage floor | Per-function metric masks 4.2% statement coverage in evaluate |
| [#324](https://github.com/toejough/engram/issues/324) | Track traced#38 | `traced verify` blocked on upstream fix; skip step 1 of boundary protocol until it ships |
| [#326](https://github.com/toejough/engram/issues/326) | Evaluation schema versioning | Add schema_version before format diverges across cycles |
| [#327](https://github.com/toejough/engram/issues/327) | Fix concurrent evaluation write collision | Unaddressed P2 from prior premortem; worsens as turn rate increases |

---

## Cycle Boundary Protocol

At every cycle end, before declaring complete:

1. ~~Run `traced stamp` and commit~~ — blocked on toejough/traced#38; skip until upstream ships (#324)
2. Mark T-items for completed tests as `status = "implemented"` in tests.toml
3. Retire superseded ARCH items (move to archive section with reason)
4. Record spec item delta in retro: items added vs. items retired
5. Check for orphaned worktrees: `git worktree list`

---

## Cycle 3 — B-2 + Backlog Features ✅ Complete

Resume the evolution plan, now that effectiveness data is real.

**Gates (must ship alongside or before cluster merge code):**
- [#328](https://github.com/toejough/engram/issues/328) — Real-filesystem integration test for cluster merge (survivor correctness) ✅ Closed (T-355)
- [#329](https://github.com/toejough/engram/issues/329) — Pre-merge backup step before any cluster delete ✅ Closed (T-347–354)

| Work | What | Status |
|------|------|--------|
| P4-full | Cluster dedup + cross-source suppression + transcript suppression | ✅ Done (#331 T-360, #332 T-359) |
| P5-full | Re-compute links after merge | ✅ Done (#333 T-358) |
| LLM principle synthesis | Synthesize merged principle from cluster on merge | ✅ Done (#330 T-356–357) |
| [#305](https://github.com/toejough/engram/issues/305) | UC-34: TF-IDF as secondary duplicate detection signal | ⏳ Deferred (see Cycle 4) |
| [#309](https://github.com/toejough/engram/issues/309) | Unify memory management across clear/compact/restart | ⏳ Deferred (see Cycle 4) |

---

## Cycle 3/4 Boundary — Premortem Mitigations

Complete before starting Cycle 4 feature work.

| Issue | What | Why |
|-------|------|-----|
| [#334](https://github.com/toejough/engram/issues/334) | RecomputeMergeLinks ignores MergedConceptSet — concept links stale after merge | Correctness gap in shipped B-2 feature; caught by code review (retro action item) |
| [#335](https://github.com/toejough/engram/issues/335) | `--dry-run` flag on all memory-mutating CLI commands | Gate: no Cycle 4 memory-mutating feature ships without dry-run visibility |
| [#336](https://github.com/toejough/engram/issues/336) | REQ: TF-IDF as tie-breaker only, not cluster initiator | Gate: blocks #305 spec derivation until constraint is in requirements.toml |

---

## Cycle 4 — Dedup Sophistication + Memory Management

| Issue | What | Status |
|-------|------|--------|
| [#305](https://github.com/toejough/engram/issues/305) | UC-34: TF-IDF as secondary duplicate detection signal | ✅ Done (ARCH-82, T-363–369) |
| [#309](https://github.com/toejough/engram/issues/309) | Unify memory management across clear/compact/restart | ✅ Done (flush command, audit deleted, hooks unified) |
| [#214](https://github.com/toejough/engram/issues/214) | Detect memories redundant with skill/plugin behavior | ✅ Descoped — deleted non-memory registration code |

---

## Backlog (unscheduled)

| Issue | What | Blocker |
|-------|------|---------|
| [#324](https://github.com/toejough/engram/issues/324) | `traced verify` broken | Upstream traced#38 |
