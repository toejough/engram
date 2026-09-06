# Real Post-Hoc Detector Spot-Check (2026-09-02)

`CALIBRATION.md` (task 3.3) pre-registered a promise this file fulfills: "the true miss rate will
be better estimated after task 5.1's larger hand-checked spot-sample" (line 63), and proposed a
tolerance for it (§6): "a hand-checked spot-check during or after the full run to show no more than
roughly 15-20% of real moments missed... If task 5.1's measured spot-check lands ≤10% missed, that's
a pass with confidence to spare." An adversarial review of `REPORT-2026-09-02.md` (2026-09-02) found
this promised check had never actually been run — only the original n=6 pre-registration calibration
existed. This file is that real, run-for-real spot-check, produced in response to that finding. **A
second adversarial round then reviewed this file itself and found real problems in its first draft**
(an undersold false-positive pattern, a wrong "7 repos" coverage claim, a mischaracterized window,
and unreproducible table rows) — this is the corrected version; see "What changed" at the end.

## Method

Same blind-independent-then-compare method as `CALIBRATION.md`'s original process, applied
post-hoc to the real production output instead of pre-run calibration windows:

1. Selected 15 windows stratified across all 4 dispatch types (main/fork/fresh/workflow) and **6 of
   the corpus's 7 repos**, drawn from `results/moments.jsonl`'s real source transcripts (`full_run`/
   `refill` stages — distinct from `CALIBRATION.md`'s original 12 windows, and distinct from the
   `pilot` stage, which was already separately sanity-checked per task 4.3). **The 7th repo, `targ`,
   was structurally impossible to include**: all 8 of targ's moments are `source_stage=='pilot'`
   (zero `full_run`/`refill` moments exist for targ) — it was never overlooked, but it also was
   never independently re-verified by *this specific* check; its coverage rests entirely on task
   4.3's separate pilot-stage hand-verification, not on this file.
2. For each window, independently read the raw transcript text and listed every real moment
   (failure/correction/rework/success/dispatch, per design.md D-D) found — **before** looking at
   what `moments.jsonl` already captured for that window, to avoid anchoring bias.
3. Compared the independent list against `moments.jsonl`'s actual records for the same
   transcript_path + location range. A moment in the independent list with no matching record in
   `moments.jsonl` counts as a miss.

## Result

**2 misses out of 22 independently-found real moments = 9.09% miss rate.**

Falls in `CALIBRATION.md` §6's own tightest proposed band: ≤10% = "a pass with confidence to
spare" — comfortably inside the 15–20% upper tolerance and well clear of its ~25%
"real problem" line.

| # | Repo | Dispatch type | Stage | Transcript | Independent | Existing | Misses |
|---|---|---|---|---|---|---|---|
| 1 | dotfiles | main | full_run | `6eecc514-5a9e-469d-8fa9-7fed249fb9b8.jsonl` | 3 | 4 | 0 |
| 2 | engram | main | full_run | `b1ffad9e-5318-418a-8719-442cdffe06c1.jsonl` | 0 | 3 | 0 |
| 3 | `-Users-joe` | main | full_run | `1df64b43-aa52-402f-a852-0534776b788e.jsonl` | 1 | 3 | 0 |
| 4 | local-llm-optimization | main | refill | `ecc84443-5a36-45f9-9d0b-3553550a585c.jsonl` | 0 | 1 | 0 |
| 5 | dotfiles | fresh | full_run | `.../subagents/agent-a89cbbb396928d46a.jsonl` | 1 | 3 | 0 |
| 6 | llmcpp | fresh | refill | `.../subagents/agent-aa31ba62ae2e98519.jsonl` | 0 | 4 | 0 |
| 7 | engram | fresh | refill | `.../subagents/agent-a898d11986e0f5882.jsonl` | 1 | 3 | 0 |
| 8 | phone-llm | fresh | refill | `.../subagents/agent-afdea369b5bac50d9.jsonl` | 1 | 1 | **1** |
| 9 | engram | fork | refill | `.../subagents/agent-a6493eb381d2089cc.jsonl` | 6 | 7 | 0 |
| 10 | phone-llm | fork | refill | `.../subagents/agent-a81b80335e1cdd38e.jsonl` | 2 | 7 | 0 |
| 11 | phone-llm | fork | refill | `.../subagents/agent-a8ce598d27fa736b9.jsonl` | 3 | 5 | **1** |
| 12 | llmcpp | workflow | refill | `.../workflows/wf_b8a87d24-24d/agent-a255a361f26705438.jsonl` | 0 | 5 | 0 |
| 13 | engram | workflow | refill | `.../workflows/wf_98d221e2-e5a/agent-aa1fe140de6bdf75f.jsonl` | 1 | 1 | 0 |
| 14 | phone-llm | workflow | refill | `.../workflows/wf_a714fbc5-61c/agent-a8a8155f08868de5d.jsonl` | 1 | 1 | 0 |
| 15 | engram | workflow | refill | `.../workflows/wf_3a74510a-d99/agent-a96aadd41edf00c3d.jsonl` | 2 | 5 | 0 |
| **Total** | | | | | **22** | **53** | **2** |

(Truncated paths root at `/Users/joe/.claude/projects/-Users-joe-repos-personal-<repo>/<session-dir>/`
— combine with the Repo column to reconstruct the full path; rows 1-4 are top-level session
transcripts, not subagent files.)

**The 2 real misses:**

1. **Window 8** (phone-llm, `agent-afdea369b5bac50d9.jsonl`): a successful `openspec-sync-specs`
   completion (~loc 13-18) — the agent judged the delta spec already matched the target, applied two
   precise Edit calls with zero errors, and confirmed "Specs Synced" with a correct before/after
   summary. No moment record exists near this location; only the initial loc-4 dispatch was
   captured. The haiku pass yielded only 1 candidate here, which *per `LOW_YIELD_THRESHOLD=2`
   should have triggered* the sonnet low-yield re-check — whether that re-check actually ran and
   still missed it, or didn't fire at all, wasn't independently confirmed; this is an inference
   from the threshold rule, not an observed fact.
2. **Window 11** (phone-llm fork, `agent-a8ce598d27fa736b9.jsonl`): a token-limit failure at loc 21
   (`Read offset=2763 limit=832` → 33,049 tokens exceeds the 25,000 cap) that was never written as
   its own moment. This transcript has 3 read-attempt failures total across the corpus: `#7` (a
   256KB size-limit failure) and `#12` (a 51,430-token failure) are both captured in
   `moments.jsonl`; this loc-21 attempt is the transcript's *third* read attempt overall and its
   *second* token-limit-specific failure (matching `#12`'s failure mode) — PROPOSALS.md's Group D
   independently calls `#12` "the second failed attempt" using a different count (attempts within
   the pair `#7`+`#12` only); both counts are correct, they're just counting different things. The
   detector's own loc-23 rework record explicitly references "second token limit failure," proving
   the *pattern* was recognized downstream — but no discrete failure moment was ever written at
   loc 21 itself, only its downstream recovery (loc 23 rework, loc 24 success).

Both misses are a *duplicate/downstream* moment sitting next to one the detector did catch, not a
total blind spot on a novel failure mode.

**The gap between "Existing" and "Independent" is larger than 2 misses, and most of it is NOT a
miss — disclosed fully here, not just gestured at.** Existing (detector) counts sum to 53 across
these 15 windows; independent counts sum to 22 — a 2.4x gap. Only 2 of those 31 "extra" detector
records are misses in the strict sense used above (a real moment the detector's *own* independent
re-derivation didn't produce); the rest is the detector flagging more candidates than an independent
blind read confirmed as real, per design's own over-flag-then-filter philosophy (a detector meant to
over-flag and let the LLM discard non-moments is *expected* to produce more raw candidates than a
strict independent count). **4 of the 15 windows had ZERO independently-confirmed real moments
despite the detector reporting 3, 1, 4, and 5 candidates respectively (windows 2, 4, 6, 12 — 13
detector-flagged candidates total with no independent corroboration in this sample):**

- **Window 2** (engram, `b1ffad9e...`): independently re-read directly — the 3 detector-flagged
  locations (34, 42, 53) are a stated plan, a real context-gathering Bash call, and a real file
  Write — plausible genuine moments by design.md D-D's own definition. The independent-count=0 here
  looks more like the blind read missing real content than the detector over-flagging.
- **Window 4** (local-llm-optimization, `ecc84443...`): the detector's single flagged moment
  (`#30`, a `dispatch`-type recall-glance invocation) is confirmed real on direct re-read — the
  independent-count=0 here is also more consistent with the blind read under-counting than a
  genuine false positive. Because this file's method only scores misses in the "independent found
  it, detector didn't" direction, an independent-reviewer undercount like this silently *lowers*
  the reported miss rate's denominator rather than flagging as either a miss or a false positive —
  a real methodological soft spot, noted here rather than hidden.
- **Window 6** (llmcpp, `aa31ba62...`): the detector's 4 candidates include a genuine, non-trivial
  `failure`-type moment (`#25`: stale release-notes data used without being flagged as stale) and a
  `rework`, not just the "routine retrieval" characterization an earlier draft of this file gave —
  corrected here. Whether the independent blind read genuinely found none of these real is not
  separately confirmed.
- **Window 12** (llmcpp, `a255a361...`, workflow): 5 detector candidates, 0 independently confirmed;
  not separately re-verified beyond the original blind read.

**Net honest read:** at minimum, windows 2 and 4 look like independent-reviewer undercounts, not
detector false positives — meaning the true miss rate could be *higher* than 9.09% if those windows'
detector-flagged moments are real and simply weren't independently re-confirmed. Windows 6 and 12
are more ambiguous. This file does not resolve that ambiguity; it discloses it rather than rounding
it into a clean "false positives, not scored" dismissal the way an earlier draft did. The speculative
link to `REPORT-2026-09-02.md` §8a's F1/F3 "suspiciously perfect" concern has been removed from that
report in response to this finding (see REPORT §8a, which now cites a simpler, code-verified
explanation instead).

## Provenance

Run via a real Workflow dispatch (`memory-loop-audit-real-spotcheck`), one sonnet-tier agent, real
`claude -p` reasoning over the raw transcript windows (no stubs). Full transcript paths are in the
table above (combine the truncated path with the Repo column); independently re-derivable by
re-running the same selection criteria against `results/moments.jsonl`.

## What changed (round 2 fixes)

A second adversarial review round independently re-checked this file's first draft and found: (1)
the false-positive framing named only 2 of 4 zero-independent-corroboration windows, understating
the scale — fixed above with the full disclosure; (2) the "7 repos" coverage claim was wrong (6
repos actually covered; targ is structurally all-`pilot`-stage) — fixed in Method step 1; (3) window
6 was mischaracterized as routine when it contains a real failure+rework — fixed; (4) the table's
(repo, dispatch_type, stage) rows were ambiguous against multiple real transcripts for about half
the sample — fixed by adding the Transcript column; (5) window 4's likely independent-reviewer
undercount of a confirmed-real dispatch moment was undisclosed — now disclosed. The headline 9.09%
miss-rate figure itself was independently re-verified against the raw transcript files in round 2
and did not change.
