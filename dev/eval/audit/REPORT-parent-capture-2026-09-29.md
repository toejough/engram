# Parent-side LESSONS capture check (learn-rate-skill-only, input to task 3.4)

This check was pre-registered in `parent-capture-plan.md` (commit c1d7d32c) before any measurement. It asks: **when a subagent reports a LESSONS item worth capturing, does the parent session get it into the vault?** The W2 re-measure could not see this path (its caveat 3a).

All figures come from `parent_capture.py` over `results-parent-capture-2026-09-29/`. Intervals are percentile bootstrap 95% intervals from `build_scorecard.bootstrap_ci` (n=2000, seed=739), per item.

## 1. Population

| Step | Count |
|---|---|
| Main transcripts enumerated (after the `-private-tmp` and `-dev-eval-` directory rules) | 165 |
| Dropped: started before 2026-09-07 | 22 |
| Excluded: orchestrating session f6dfe139 | 1 |
| Excluded: headless eval arm | 7 |
| Excluded: automated security review | 126 |
| Eligible main sessions | 9 |
| **Population: sessions that received a non-`none` LESSONS block** | **2** |

- **`23a08637`**, the runbook-vs-skill phase-2 orchestration session, 2026-09-09..15. It had 80 receipts with a LESSONS block; 6 were `none`. They yield 130 items.
- **`01e43979`**, phone-llm, 2026-09-24..26. It had 32 receipts with a LESSONS block, none of them `none`. They yield 35 items.
- **Total: 165 items.** Each item is one lesson, deduped within its session.

The vault note set holds 1,792 notes: 953 in HEAD and 839 deleted in history. There were 0 notes written but never committed. 251 notes have an exact write time from the learn-call index (272 entries).

## 2. Result

| Metric | Unit | Value | 95% interval |
|---|---|---|---|
| Worth capturing (judge a) | items | 114/165 = 69.1% | [61.8, 75.8]% |
| **C1: captured or covered, of worth-capturing** | items | **93/114 = 81.6%** | **[74.6, 87.7]%** |
| – covered (a matching note already existed at the return) | items | 40/114 = 35.1% | — |
| – captured (a matching note was written after the return, within the session + 24h) | items | 53/114 = 46.5% | — |
| – lost | items | 21/114 = 18.4% | — |
| C1-new: captured, of worth-capturing items not already covered | items | 53/74 = 71.6% | [60.8, 81.1]% |
| C1-strict: same-day-ambiguous captures counted as lost | items | 81/114 = 71.1% | [62.3, 78.9]% |
| C0: a `learn` invocation after the session's last LESSONS receipt | sessions | 0/2 | — |
| C0 with invocations labelled mid-cycle excluded | sessions | 0/2 | — |
| Post hoc: C1 with matches to route-dispatch/route-evidence notes removed | items | 80/114 = 70.2% | [62.3, 78.1]% |

### Per session

| Session | Worth capturing | C1 | C1-new | Captured / covered / lost |
|---|---|---|---|---|
| 23a08637 | 82/130 = 63.1% | 68/82 = 82.9% [75.6, 91.5] | 33/47 = 70.2% [55.3, 83.0] | 33 / 35 / 14 |
| 01e43979 | 32/35 = 91.4% | 25/32 = 78.1% [62.5, 90.6] | 20/27 = 74.1% [55.6, 88.9] | 20 / 5 / 7 |

### Secondary metrics

- **Worth-capturing items by kind:**
  - corrections: 78;
  - reversals: 17;
  - confirmed approaches: 18;
  - save-requests: 1.
- **Who wrote the 53 captured items' notes:**
  - the parent's own transcript: 38;
  - one of the parent's subagents: 3;
  - unknown (day-resolution write time only): 12.
- **Timing:** 12 captures rest only on notes with `same_day_ambiguous` timing. This is the difference between C1 and C1-strict.
- **Judge errors:** 0. No call exhausted its retries, and no reply needed a parse retry.

### How capture actually happened

C0 is 0/2. Neither parent ran the closing `/learn` that v1's design routes LESSONS lines through:

- `23a08637` invoked `learn` twice (2026-09-10 and 2026-09-12, both labelled mid-cycle), before its last receipt on 2026-09-13.
- `01e43979` invoked `learn` once (mid-cycle, 2026-09-24 14:14), before its first receipt.

The capture that did happen came from direct `engram learn` calls in the parent transcripts, made in the middle of each session:

- **`23a08637`** wrote 45 notes this way. 32 of them are route-dispatch or route-evidence records, and 13 are principle notes.
- **`01e43979`** wrote 9 principle notes in one batch at 2026-09-25 00:21 UTC, plus 1 other note. That batch was before its last receipt.

So the parent-side path is reaching the vault, but not through the mechanism v1 specified.

## 3. Judge validity (spot-check, post hoc, one reader, unblinded)

I read 10 randomly drawn captured items (seed 7) and 8 covered items (seed 3) next to the notes the judge matched them to.

| Group | Match holds on reading | Main failure |
|---|---|---|
| captured | 4/10 | Items matched to route-dispatch records, or to notes on an adjacent topic. For example, `23a08637-005` (use `engram embed status` for sidecar freshness) was matched to route-dispatch note 945. |
| covered | 6/8 | Generic or route-record matches. For example, `23a08637-013` was matched to note 743, "write the claim from the artifact". |

Judge (b) is lenient. A large candidate list (up to 102 notes per batch) invites topic-level matches that the prompt forbids. Dropping route-record matches alone takes C1 from 81.6% to 70.2%, and the spot-check suggests true capture precision is lower still. **C1 as judged is an upper bound.** A hand-corrected rate would need a blinded second judgement, which the plan does not include.

## 4. Deviations from the plan

1. **Coverage requires the note to exist at the return.** A deleted note counts as pre-return coverage only if its deleting commit is after `t_ret`. Without this guard, 800+ notes purged in June would have counted as coverage. This is an implementation clarification of "already covered by an existing note".
2. **Exact write times.** A learn-call index hit is used as the note's write time only when its local date equals the note's `created` date. Otherwise it is an amend of an older note, and the note falls back to day-resolution timing.
3. **Post hoc additions.** The route-record sensitivity row and the spot-check were not pre-registered. Both are labelled post hoc.
4. **Scouting before the plan.** The plan's disclosure section records the scouting counts, taken before the plan was written.

## 5. Validity caveats

- **Tiny, clustered population.** There are 2 sessions, and one of them contributes 79% of the items; that session was itself eval-building work in engram. The per-item intervals ignore clustering. C0 at n=2 is descriptive only.
- **Judge (b) over-matches** (section 3). It inflates `captured` and `covered`.
- **Items are undercounted.** Single-line comma-separated LESSONS lines were not split (plan limitation), and blocks with bullets were split.
- **Some captures are the subagent's own work.** Several covered items are lessons the subagent had already written itself before returning (for example `23a08637-047`, note 985). That is subagent self-capture, not the parent path.
- **Pi transcripts were not scanned** for learn calls. The only effect is that some write times fall back to day resolution.

## 6. Cost

- **Projected** by the dry run (an upper bound assuming every item is worth capturing, with a 25% margin): $11.54. That is within the $25 gate.
- **Actual:** $7.99 across 42 sonnet calls: 17 for judge (a) and 25 for judge (b). The figure is the sum of `total_cost_usd` in `cost-log.jsonl`.

## 7. For task 3.4 (no pass bar pre-registered)

- By the judge, 82% of worth-capturing LESSONS items end up in the vault in some form (70% after removing route-record matches, and likely lower on a stricter reading).
- The designed closing-learn step ran in 0 of 2 sessions.
- Capture is carried by the parents' ad-hoc direct writes and by subagents writing notes themselves.
- Whether that meets the bar, and whether a stricter re-judge is worth about $5–8, is Joe's call.

## 8. Artifacts

- `parent_capture.py` and `test_parent_capture.py`: 40 tests, no API calls.
- `results-parent-capture-2026-09-29/`:
  - `dryrun.json`;
  - `items.jsonl`, with candidates;
  - `items-judged.jsonl`;
  - `judge-a.jsonl` and `judge-b.jsonl`, the raw replies;
  - `cost-log.jsonl`;
  - `scorecard.json`;
  - `run-manifest.json`.
- `dev/eval/LEDGER.md`: row `learn-rate-parent-capture`.

Command: `python3 dev/eval/audit/parent_capture.py --output-dir dev/eval/audit/results-parent-capture-2026-09-29 --run`.
