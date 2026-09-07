# Re-measure plan: learn-rate-skill-only (v1)

Pre-registered per design D-E of `openspec/changes/learn-rate-skill-only/design.md`, before any
natural usage of the v1 skill edits has occurred. This commits the instrument, the timeline, and
the metric in writing ahead of running the re-measure, per ADR-0028 D-G (mechanical commitment
prevents p-hacking).

## Ship date

v1 shipped 2026-09-07: the three skill edits (`route`, `please`, `learn`) landed via
`superpowers:writing-skills` TDD (RED/GREEN pressure tests, tasks 1.1–1.3) and were deployed via
`engram update` to `~/.claude/skills/{route,please,learn}/` and
`~/.pi/agent/skills/{route,please,learn}/` (tasks 2.1–2.2), both verified byte-identical to
source at deploy time.

## Re-measure date

**Target: 2026-09-28** (3 weeks after ship). Acceptable window: 2026-09-21 through 2026-09-28
(2–3 weeks, per design D-E's calibration to Joe's real usage cadence rather than an assumed
timescale). Run no earlier than 2026-09-21 — less than 2 weeks of natural usage is unlikely to
accumulate enough worth-learning moments for a meaningful per-moment rate.

## Instrument

`dev/eval/audit/run_audit.py`, **instrument v2** (bootstrap CIs) — the same version used for the
baseline run. The re-measure MUST declare its instrument version explicitly in its output so the
two runs are comparable; if the instrument has changed by re-measure time, that is itself a
finding to note, not something to paper over.

## Baseline (from `dev/eval/audit/REPORT-2026-09-02.md`, gate-reviewed 2026-09-02..06)

| Metric | Baseline value | Counting unit |
| --- | --- | --- |
| **W2 (primary)** — % of worth-learning moments where learn fired | **5.6%** (4/71), CI95 [1.4, 11.3] | per-moment |
| Lost lessons (secondary) | 67/150 total; **40/67 (59.7%) judged fixable** | per-moment |
| Fixable-loss role breakdown | workflow 14, fresh 11, fork 8, main 7 (82.5% in dispatched work) | per-moment (recomputed from `results/moments.jsonl`, see `openspec/changes/learn-rate-skill-only/design.md` Alternatives Considered) |

## Go/no-go

No pre-set green/yellow/red threshold (Joe's 2026-09-01 override — judgment call at gate time,
per design D-E). If W2 moves below Joe's bar in the re-measure, escalate per design D-F (watcher/
metacognition layer, mechanical validator, hooks/harness push — all currently parked). If W2 hits
or exceeds Joe's bar, park the escalations further.

## Cost

~$35 (comparable to the original audit's LLM spend), per design D-E.

## Next steps (tasks 3.2–3.4)

Blocked on the elapsed-time window above and on Joe's own machine/vault (the re-measure needs
real natural usage and the real vault/transcript history — not reproducible in this sandbox,
which has no local vault). When the window opens:
- 3.2: run the re-measure, capture raw results under `dev/eval/audit/results/re-measure-<date>/`.
- 3.3: compute W2 + secondary metrics, compare to baseline, document in
  `dev/eval/audit/REPORT-re-measure-<date>.md`.
- 3.4: joint review with Joe of the comparison; decide escalation per design D-F; record in
  `escalation-decision-<date>.md`.
