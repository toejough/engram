## 1. Recall skill wording fix (writing-skills TDD — required, no exceptions per vault note 26)

- [x] 1.1 Baseline (RED): Confirmed via vault note 794 (2026-08-26), which analyzed the exact
      preserved failing-trial transcript from `gate-C5-6l_mjvl1/warm-cfg/projects/.../warm-3-*/*.jsonl`.
      Agent made ZERO `engram show-chunk` calls on zero-note cluster; synthesis text "0 notes;
      crystallized 0 lessons" treated empty candidates as license to skip chunk reads entirely.
- [x] 1.2 Invoked `superpowers:writing-skills` and edited `agent-instructions/skills/recall/SKILL.md`
      Step 2.5A: added visually distinct callout for zero-note clusters (line 174) that explicitly
      mandates `engram show-chunk` calls even when `candidate_l2s` is empty. Answered Open
      Question 2 via pressure-tested wording.
- [x] 1.3 GREEN: Ran `dev/eval/traps/seed_c5.py && dev/eval/traps/c5.py --arms warm --n 5`
      against the new wording on 2026-08-27. Result: 5/5 trials passed
      (valid=5/5, C5a surfaced=5/5, C5b honored=5/5). The zero-note cluster failure mode
      (zero `engram show-chunk` calls with empty candidates) confirmed NOT reproduced.
- [x] 1.4 Pressure-tested the new wording: verified no wasteful over-correction (mandate logically
      gates to "if chunk members exist"; zero-member clusters skip reading). Identified and closed
      rationalization loopholes (explicit "does NOT skip", "MUST still call"). Per vault note 198,
      acknowledged ~95% ceiling on prose-only skill-text fixes.
- [x] 1.5 Synced via `engram update --with-guidance` — 7 recall skill files deployed to ~/.claude/skills/recall/
      and ~/.pi/agent/skills/recall/. Verified deployed SKILL.md contains new wording at line 174
      of ~/.claude/skills/recall/SKILL.md (grep: "For zero-note clusters").

## 2. Re-measure and decide the gate threshold

- [x] 2.1 Ran `dev/eval/traps/c5.py --arms warm --n 15` post-fix measurement (2026-08-27):
      15/15 honored (100%). Combined with independent n=5 arm (5/5 honored, writing-skills
      GREEN check) = 20/20 honored (100%) across two independent runs, substantial improvement
      from pre-fix 4/5 (80%). LEDGER entry: `733-c5-threshold-decision`.
- [x] 2.2 Used post-fix rate (100%) to decide design.md Open Question 1: KEEP
      `dev/eval/traps/gate_verdict.py`'s exact-bar C5 verdict (exact-bar criterion
      reliably cleared by fix). Rationale in LEDGER entry `733-c5-threshold-decision`:
      the repo's same-tree-recheck discipline (vault note 793) distinguishes real regressions
      from variance; the fix restores original 100% baseline; threshold selection would be
      arbitrary; exact-bar is more conservative and predictable.
- [x] 2.3 Gate criterion does NOT change — `dev/eval/traps/gate_verdict.py` (`_norm_c5` / 
      `axis_verdict` line 11) and `test_gate.py` (lines 15–22) remain exactly as written
      (exact-bar: `passed == len(valid)`). No update needed.
- [x] 2.4 Updated `docs/GLOSSARY.md`'s lazy-chunks entry (lines 372–381),
      `docs/architecture/c2-containers.md:126`, and `docs/architecture/c1-system-context.md:165`
      to add zero-note-cluster mandatory-read case explicitly. Kept the general "0/13 realistic
      recalls" fetch-frequency statistic (describes general chunk-fetch rarity), added explicit
      exception: zero-note clusters are MANDATORY read case regardless of general rarity, with
      post-fix C5 compliance evidence (20/20 honored, 2026-08-27).

## 3. Close out

- [x] 3.1 Sync specs (`/opsx:sync` or `openspec` equivalent) to merge the ADDED requirement into
      `openspec/specs/recall-payload-cuts/spec.md`. Merged 2026-08-27: added requirement
      "Agent reads chunk content before judging cluster relevance, even in zero-note clusters"
      with 3 scenarios; `openspec validate` passes.
- [x] 3.2 Updated `dev/eval/traps/README.md` and `dev/eval/LEDGER.md` to reflect fix shipped.
      Removed interim C5b exception callout from README.md Verdict section (lines 31-39, previously
      "Exception, interim (#733)..."); updated LEDGER.md row `733-c5b-honoring-rate` verdict column
      to reflect fix shipped 2026-08-27, added forward-reference to `733-c5-threshold-decision` row
      documenting post-fix 100% rate (20/20 honored); updated figure narrative to document SKILL.md
      edit shipped and deployment via `engram update --with-guidance`.
- [ ] 3.3 Close GitHub issue #733, referencing the merged spec, the SKILL.md commit, and the
      final measured rate.
- [ ] 3.4 Archive this change (`openspec archive recall-chunk-only-cluster-read-gate`).
