# Task 3.3 Calibration

Calibrates the semantic auditor (`audit_moments.py`, built in tasks 3.1–3.2) against independently
hand-labeled ground truth, **before** the real corpus run (task 5.1). Real `claude -p` detector runs
were made (haiku detection + sonnet low-yield re-check) over 12 windows spanning 4 repos and both
main-agent and subagent transcript types. Every finding below comes from **separate, blind hand-reading**
of the actual transcript files against the detector's output — not from the tool re-deriving its own
results, ensuring independent verification.

## 1. The 12 calibration windows

Per the task brief, these 12 windows were pre-selected to span 4 repos and both main-agent and subagent
transcripts:

| # | Repo | Type | File (relative to ~/.claude/projects/ or subagents/) |
|---|------|------|-------|
| 1 | engram | main (whole file, 145 lines, under 500KB) | -Users-joe-repos-personal-engram/e5ccd18d-c5d3-4593-9c41-61394832e987.jsonl |
| 2 | targ | main, window 0 (lines 1-69 of 678, file needs windowing) | -Users-joe-repos-personal-targ/3d697973-ee50-42a8-9657-f57181ca991a.jsonl |
| 3 | llmcpp | main (whole file, 71 lines) | -Users-joe-repos-personal-llmcpp/ccf6fbe1-4134-4f76-83f4-19c6835540a6.jsonl |
| 4 | engram | subagent (fork) | .../a19cc352.../subagents/agent-a5e3cc9e58c463d77.jsonl |
| 5 | engram | subagent (fresh) | .../32505058.../subagents/agent-a7c233939e3d57189.jsonl |
| 6 | engram | subagent (fresh) | .../32505058.../subagents/agent-a62dcb4d0c1554476.jsonl |
| 7 | targ | subagent (fresh, Gate A plan reviewer) | .../d1c5ca03.../subagents/agent-ac7ec7f863667110d.jsonl |
| 8 | targ | subagent (fresh, Gate D commit reviewer) | .../af8b6c87.../subagents/agent-a55d9fee44953f18d.jsonl |
| 9 | targ | subagent (fresh, code reviewer) | .../d1c5ca03.../subagents/agent-a6faba7b1e62b8aad.jsonl |
| 10 | phone-llm | subagent (fresh, Gate D clarity reviewer) | .../57207951.../subagents/agent-a4a11f70fe9cb1e3d.jsonl |
| 11 | phone-llm | subagent (fresh, Gate D commit reviewer) | .../57207951.../subagents/agent-ab6024d33132cf23c.jsonl |
| 12 | phone-llm | subagent (fresh, Gate D commit-message reviewer) | .../c74b4a61.../subagents/agent-a0e08744c1ff39ff4.jsonl |

**Breakdown:** 4 repos — engram 4 (#1, #4, #5, #6), targ 4 (#2, #7, #8, #9), llmcpp 1 (#3), phone-llm 3
(#10, #11, #12). 3 main-agent windows (1 whole-file, 1 windowed slice, 1 whole-file), 9 subagent windows.

## 2. Measured miss rate

Ground truth was built by independent human reading of every one of the 12 windows' raw transcript
content (without running the tool under test) and hand-labeling every real failure/correction/rework/
success/dispatch moment. This hand-reading was done in 5 separate blind-labeling passes by independent
reviewers, each blind to what the automated detector found. The hand-labeled results were then compared
against the detector's candidate list.

| Window | Repo/type | Hand-labeled real moments | Detector candidates (haiku + sonnet re-check) | Missed |
|---|---|---|---|---|
| 1 | engram main | 0 | 3 success | 0 |
| 2 | targ main | 1 failure (line ~22-23) | 1 failure@21, 1 correction@26, 3 success (33, 54, 67) | 0 |
| 3 | llmcpp main | 0 | 1 success@34, 2 rework (50, 62) | 0 |
| 4 | engram sub (fork) | 1 dispatch@2 | 1 dispatch@2, 1 success@5 | 0 |
| 5 | engram sub (fresh) | 0 | 1 success@5 (re-check fired, no change) | 0 |
| 6 | engram sub (fresh) | 0 | 1 failure@5 (re-check fired, no change) | 0 |
| 7 | targ sub (fresh) | 1 correction@7 | 0 from haiku alone; sonnet re-check found 1 failure@7 | 0 |
| 8 | targ sub (fresh) | 1 success (lines 1-15) | 3 success (11, 13, 15) — location-15 matches GT | 0 |
| 9 | targ sub (fresh) | 0 | 1 success@13 (re-check fired, added false-positive dispatch@1) | 0 |
| 10 | phone-llm sub (fresh) | 2 moments: success + correction@line 5 | 1 success@5 (re-check fired, no change) | 0 |
| 11 | phone-llm sub (fresh) | 0 | 2 success (11, 12) — both false positives (re-check fired, no change) | 0 |
| 12 | phone-llm sub (fresh) | 0 | 1 success@14 (re-check fired, no change) — explicitly examined and found unattributable | 0 |
| **Total** | | **6** | **25 across all windows** | **0** |

**Overall measured miss rate: 0/6 = 0%.**

**Critical caveat on this result:** n=6 is a small sample. 0% on n=6 is a strong initial signal that the
detector is working well, but a somewhat higher real-world miss rate on the full corpus is entirely
plausible. Do not treat 0% as a tight statistical bound. This finding is useful primarily as validation
that the detector's basic mechanism isn't fundamentally broken; the true miss rate will be better
estimated after task 5.1's larger hand-checked spot-sample.

## 3. Did the low-yield sonnet re-check help?

The re-check fired on 7 of the 12 windows (haiku yield `< 2`, the current threshold):
windows 5, 6, 7, 9, 10, 11, 12.

**Re-check decisiveness:**

| Window | Haiku found | Sonnet added | Decisive (caught new real moment)? |
|---|---|---|---|
| 5 | 1 | no change | No |
| 6 | 1 | no change | No |
| **7** | **0** | **1 candidate** | **Yes** — haiku alone found zero; sonnet caught the real moment |
| 9 | 1 | 1 additional candidate | No (false positive added, real moment already caught by haiku) |
| 10 | 1 | no change | No |
| 11 | 1 | 1 additional candidate | No (duplicate of same real moment, both false positives anyway) |
| 12 | 1 | no change (sonnet undershoot) | No |

**Finding:** the re-check was decisive exactly once (window 7), where haiku alone found zero candidates
on a window that genuinely contained a real correction moment. In the other 6 re-check windows, the
sonnet pass either changed nothing or added over-flagging with no corresponding ground-truth support.
This validates the mechanism's intended behavior (cover sparse windows) at modest cost (extra flagging
noise elsewhere, no regressions).

## 4. Two bugs found-and-fixed during hand-reading

**These are RESOLVED — not open issues. Listed here for completeness, since they affect interpretation
of later task results.**

1. **`extract.py`'s Claude Code parser did not detect bare-Bash `engram query` calls.** Subagent
   transcripts (particularly Gate A/D reviewer agents) often run `engram query --lazy-chunks --phrase ...`
   via raw Bash tool-use rather than wrapping it in the `recall` Skill. The extractor's `recall_calls`
   detection was scoped to Skill-mediated invocations only (per spec: "the Skill call plus result").
   This meant those bare-Bash calls were invisible to the point-in-time check's `search_ran` field.
   **Fix applied:** expanded `extract.py` to detect bare-Bash `engram query` calls, with a safety guard
   restricting the new detection to subagent transcripts only (never main sessions, to avoid
   misattributing unrelated skills' own query calls).

2. **Point-in-time check returned indistinguishable results for "check failed" vs. "nothing existed."**
   A failed `engram query` subprocess call in the existence check was recorded as `memory_existed: False`,
   the same value used when a genuine empty result came back. This silently conflated two distinct cases.
   **Fix applied:** point-in-time check now returns `memory_existed: None` plus an explicit
   `existence_check_error` field when the check itself errors (e.g., a subprocess failure), distinct from
   `False` (nothing found) or `True` (match found).

## 5. Other hand-read findings — documented open limitations for task 6.1

**Not yet fixed; flagged here as known precision gaps for the full-run report:**

1. **Hallucinated "dispatch" categorization:** the detector once labeled a plain CLI tool-use
   (`agent runs 'gh issue view'`) as a DISPATCH event, conflating external tool-use with subagent
   launch. This is a category-definition looseness in the detection prompt, not a location-drift bug.
   Worth a one-line prompt clarification before task 5.1 if dispatch-type counts matter to reporting.

2. **Main-session-specific location drift:** detector candidates in targ main transcripts sometimes landed
   on Claude Code UI bookkeeping lines (`ai-title`, `last-prompt`, `system` markers) rather than the
   actual content line 3-6 lines earlier. This pattern did not appear in subagent transcripts checked.
   Specific to main-session structure; subagent windows are unaffected.

3. **No explicit downstream filtering on `worth_learning_from`.** Design.md's "over-flag then discard"
   philosophy currently relies entirely on a downstream consumer filtering on the `worth_learning_from`
   field; if task 5.2's rate computations don't explicitly filter it, over-flagged noise will inflate raw
   candidate counts. The raw coverage (whether real moments are caught) is sound; the candidate *count*
   may be inflated.

4. **`search_phrases` not persisted.** The point-in-time check's own query phrases are not stored in the
   final moment record, limiting later auditability of why an existence judgment came out a particular way.

5. **Conflation of conceptually distinct moments.** Window 10 had a single turn where an agent used a
   recalled memory (success) AND received a corrective finding as a result (correction) — two logically
   distinct GT concepts on the same moment. The detector emitted one conflated candidate; the hand-labeler
   counted two separate moments. The location-first matching rule treats this as a non-miss, but it's a
   real structural gap for task 4.1's counting-unit design.

6. **Correction-type detection is underspecified.** Of the 6 hand-labeled moments, 2 were correction-type,
   but the detector never once emitted a candidate literally labeled "correction." Both correction moments
   were caught via cross-type matches (falling under "failure" or "success" candidates at the same
   location). This suggests the detection prompt's correction definition may need sharpening before task
   5.1 if correction-type breakouts matter to reporting.

## 6. Proposed miss-rate tolerance for full-run approval gate

**This is a proposal for Joe's explicit approval, not a decision.** (Note: task 4.2's scope changed
after this was written — 2026-09-01, Joe declined pre-committing numeric thresholds for the
Phase 2 go/no-go decision specifically, a different topic from this detector-quality tolerance.
That approval never happened as a discrete task; see the real spot-check result reported in
REPORT-2026-09-02.md §8h, which supersedes this section's un-actioned proposal with a real
post-hoc measurement.)

Measured performance on this calibration: **0/6 moments missed (0%), but on a small sample (n=6).**

**Proposed commitment for task 5.1's full run:** *given the small sample, propose allowing a hand-checked
spot-check during or after the full run to show no more than roughly 15-20% of real moments missed.* This
is deliberately hedged above the measured 0%, reflecting:

- The small calibration sample (n=6) doesn't provide tight statistical power to predict the full corpus.
- A measured 0% suggests the detector is fundamentally sound, but real-world miss rates typically run
  higher on larger, noisier datasets.
- 15-20% represents a meaningful but not-catastrophic false-negative rate, giving headroom for the
  detector to encounter patterns outside this calibration's diversity.
- If task 5.1's measured spot-check lands ≤10% missed, that's a pass with confidence to spare; if it
  exceeds ~25%, that signals a real problem worth re-tuning before shipping.

**Joe should weigh in on whether 15-20% is the right target**, whether the proposal should be per-type
(e.g., "no more than 30% missed on failure-type, 40% on correction-type") given the downstream
limitations identified above, and whether the small calibration sample justifies the hedge at all.

## 7. Proposed `LOW_YIELD_THRESHOLD` value

**Current placeholder:** `LOW_YIELD_THRESHOLD = 2` (`audit_moments.py:27`).

**Proposed value: keep it at 2.**

Data supporting this:

- At threshold 2, the re-check fired exactly on windows with haiku yield 0–1 (windows 5, 6, 7, 9, 10,
  11, 12).
- The re-check was decisive in 1 of those 7 cases (window 7, yield 0). In that case it recovered a real
  missed moment, validating the re-check's core purpose.
- Windows with yield ≥ 2 that were NOT rechecked (window 4 at yield 2, window 8 at yield 3) had zero
  real misses — no evidence that raising the threshold to 3+ would catch additional real moments; it
  would only add sonnet-tier cost to windows that were already complete.

**Conclusion:** threshold 2 is justified by the data. Raising it would add cost with no demonstrated
benefit; lowering it would cost more without evidence of additional catches.

## 8. Real LLM cost spent on this calibration

| Run | Cost |
|---|---|
| Miss-rate comparison (12 windows, detect-only, haiku + sonnet re-check on sparse windows) | $3.24 |
| Full-pipeline hand-read (3 windows, detect + judge + point-in-time check with real endpoint tests) | $6.10 |
| **Total** | **$9.34** |

Individual subprocess call costs and timings are logged in this session's scratchpad.
