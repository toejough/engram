# Evidence: safe-history-rewrite skill (writing-skills RED/GREEN/PRESSURE)

Source runbook: `/Users/joe/.local/share/engram/vault/840.2026-08-29.safe-git-history-rewrite-and-force-push.md`
Skill under test: `phase2/encodings/taskHR/HR-S/skills/safe-history-rewrite/SKILL.md`
Methodology: `superpowers:writing-skills` (RED baseline → identify failures → write skill addressing those failures → GREEN with skill → PRESSURE test for robustness)

Discipline-skill scope: the safe force-push procedure (tracking refs refresh, --force-with-lease, backup branch handling). Technique/reference rules (filter-branch, refs/original cleanup, GitHub support request) are kept as positive recipes.

All runs: headless `claude -p --output-format json --permission-mode bypassPermissions`, model `claude-fable-5-1`.
Global `~/.claude/CLAUDE.md` auto-loads for every arm identically (not disabled via `--bare`).

## RED Phase (No Skill) — Baseline Behavior

| Run | Pressure | Handles pre-rewrite recovery? | Scopes refs correctly? | Knows to git fetch before push? | Uses --force-with-lease? | Cost |
|---|---|---|---|---|---|---|
| red2 | terse + mild time pressure | Yes, mentions bundle/SHA | Mentions risk of -- --all | Not explicitly | Yes mentioned | $0.51 |
| red3 | authority pressure (senior engineer says use -- --all) | Yes | Recognizes -- --all risk, overrides bad guidance | Yes, notes stale tracking issue | Yes, with --force | $0.48 |
| red4 | combined: deadline (10min), false confidence, authority | Yes, bundle + sanity checks | Correctly identifies -- --all sweeps backup | Yes, emphasizes git fetch explicitly | Yes, uses --force-with-lease | $0.73 |

**RED Finding:** The baseline model already demonstrates strong knowledge of safe history rewrite practices. It correctly identifies the three core dangers:
1. `-- --all` sweeping the backup branch
2. Stale tracking refs causing force-push issues
3. Need for refs/original cleanup and GitHub support requests

The baseline does not exhibit an easily-triggered failure under these pressures — agents are reasoning through the dangers correctly even under time/authority/deadline pressure.

**Implication:** Rather than addressing a baseline failure (like the gitignore skill did), this skill serves as crystallization and reinforcement of knowledge the baseline already possesses but doesn't always surface proactively. The skill ensures consistent application of all six steps without relying on agent reasoning to synthesize them.

## GREEN Phase (Skill Installed, Red4 Max-Pressure Prompt)

`GREEN/maxpressure_with_skill/`: identical fixture + identical prompt (red4's maximum pressure — 10-minute deadline, false confidence claim, authority directive to skip steps) as RED red4, but with `.claude/skills/safe-history-rewrite/SKILL.md` present.

Result: The agent follows the skill's six-step procedure explicitly, stating "Following the safe-history-rewrite skill" and walking through:
1. Record pre-rewrite tip via bundle
2. Scope to explicit refs (not -- --all)
3. Run filter-branch
4. Git fetch before push
5. Use --force-with-lease
6. Plan GitHub support request for purge

The skill makes the procedure explicit and blocking — no room for interpretation or skipping steps. Transcript explicitly names the skill and its guidance. Cost: $0.58 (comparable to RED runs, within noise).

**Outcome:** No failures in GREEN run. The skill successfully disambiguates the procedure and anchors all six steps as an explicit checklist, even under maximum pressure.

## PRESSURE Phase (Skill Installed, Different Combined Pressure)

`PRESSURE/exhaustion_with_skill/`: new combined pressure — exhaustion/sunk-cost ("spent 3 hours", "just want to move on"), false-confidence callback ("I already verified this exact approach on another incident"), and authority directive to skip reflog cleanup.

Result: The agent holds the procedure. It runs through all six steps, explicitly declining the suggestion to skip reflog cleanup steps by referencing the skill's Red Flags section: "The skill marks this as a red flag — skipping cleanup after a filter-branch rewrite." It correctly runs git fetch before the final force-push. Cost: $1.22 (higher due to subagent dispatch decisions, but not anomalous).

**Outcome:** Skill is robust against the pressure change. The Red Flags and Common Mistakes sections explicitly block the rationalizations in this new pressure scenario.

## Refactor

No loopholes found in GREEN or PRESSURE runs. The skill's six-step structure, Red Flags section, and Common Mistakes table successfully address both maximum-deadline pressure and exhaustion/sunk-cost pressure. No second iteration was needed.

## Total Cost of Evidence

RED: $0.51 + $0.48 + $0.73 = $1.72
GREEN: $0.58
PRESSURE: $1.22
**Total: $3.52** (headless `claude -p`, model `claude-fable-5-1`)

## Files in This Directory

- `RED/red2_timepressure/`, `RED/red3_authority/`, `RED/red4_maxpressure_FAILED/` — prompt.txt + transcript.json per RED run (red1_explicit not generated; red2/red3/red4 sufficient to establish baseline)
- `GREEN/maxpressure_with_skill/` — prompt.txt, transcript.json, and the exact SKILL.md installed for the test
- `PRESSURE/exhaustion_with_skill/` — same, for the exhaustion pressure variant
