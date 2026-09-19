## 1. Pre-flight

- [x] 1.1 Confirm no route runbook already exists in the production vault (`engram query --lazy-chunks --phrase "routing a subagent dispatch and deciding its tier"` and a basename check for `route-dispatch-tier-selection`) — determines `engram learn runbook` vs `engram amend` in 2.1
- [x] 1.2 Read `Route-R`'s current fixture body in full (`dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskRoute/Route-R/vault/1.2026-09-12.route-dispatch-tier-selection.md`) and confirm it still matches the 3 fixes documented in design.md's Context (red_flags field populated, 4 write-memory references wikilinked, `--tags`-vs-`--tag` wording clarified) — re-derive if drifted since 2026-09-12

## 2. Promote the runbook

- [x] 2.1 Promote `Route-R`'s content into the production vault via `engram learn runbook` (rebuild whole-note, per vault note 958 — never a hand-edit or file copy), including situation, steps/body, done_when, and the existing `red_flags` entries verbatim — production note `1036.2026-09-18.route-dispatch-tier-selection`
- [x] 2.2 Add the one new `red_flags` entry (design.md D2): "about to write an aggregate-evidence field as plain prose instead of the `[[note-basename]]` wikilink bracket syntax the template above already shows — stop and use the bracketed form"
- [x] 2.3 Re-embed (`engram embed apply` on the note or `--stale`); confirm `engram embed status` clean

## 3. Verify retrieval

- [x] 3.1 Run representative route-shaped `engram query` phrases (mirroring the archived eval's task 3.3 retrieval-check pattern, e.g. "routing a subagent dispatch and deciding its tier") and confirm the promoted runbook surfaces top-ranked as `kind: runbook`

## 4. Re-run the D8 verification

- [x] 4.1 Run `dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py --task route --model sonnet5 --n 3 --arms R --shim-only --keep` (or the current equivalent invocation) against the amended runbook — run twice: first attempt (pre-#763-fix) found 2/2 scoreable, followed_all 1/2, end_state 0/2; **final rerun (post-#763-fix, `results/postfix763_sonnet5_shimonly_route.jsonl`) found 3/3, followed_all 3/3, end_state 3/3**
- [x] 4.2 Record `followed_all` and `end_state` against the skill row's 2/3 reference; state plainly whether the D8 bar (within one trial of 2/3) is met — report the actual numbers even if it is not met — **MET, and exceeded**: end_state 3/3 vs skill row's 2/3, $1.43 total spend on the final rerun
- [x] 4.3 If not met: read the kept trial transcripts to determine whether the `red_flags` entry changed the failure mode at all (e.g., agent noticed but still wrote plain text vs. never engaged with the red flag) — record the finding; do not attempt a second fix in this task without a new design decision — **first attempt's finding**: the red_flags entry was never delivered to the agent's context at all (truncated past Claude Code's ~2KB Bash-output boundary), traced to and fixed as #763 (`runbook-redflags-truncation-safety`); the final rerun above confirms the fix resolved it

## 5. Retire the skill (only if 4.2 met the bar AND shim.md is confirmed activated)

- [x] 5.0 Confirm `agent-instructions/guidance/shim.md` is actually activated in the real target CLAUDE.md(s) (an `@~/.claude/engram/shim.md`-style import present in `~/.claude/CLAUDE.md`, and the `~/.pi/agent/` equivalent if that harness is in scope) — not merely deployed to `~/.claude/engram/guidance/shim.md`. A D8 pass alone does not satisfy this: the eval only proves the runbook works inside its own shim-only fixture config (per design.md D4 and vault note `1031a`). Do NOT proceed to 5.1 unless this is confirmed true. — **confirmed true** after `openspec/changes/activate-shim-guidance/`: `@~/.claude/engram/shim.md` and `@~/.pi/agent/guidance/shim.md` are now live imports
- [x] 5.1 Remove `agent-instructions/skills/route/SKILL.md` (or replace with a stub, per the Open Question resolution in design.md — record which was chosen) — **clean removal chosen** (design.md's stated default). Evidence: skill discovery is directory-presence-based only (`~/.claude/skills/route` was a symlink into a materialized copy synced from this repo) — no error/crash mode found for an absent skill, matching every skill that was simply never installed; no reason surfaced to keep a stub. Removed `agent-instructions/skills/route/{SKILL.md,price-table.md,tests/}`.
- [x] 5.2 Update any deployment path that packages `agent-instructions/skills/route/` (e.g. `engram update --with-guidance` manifest/sync list) to stop shipping the retired directory — **no code change needed**: skill deployment (`internal/cli/update.go`'s `SkillDirs`, `internal/update/update.go`'s `EngramSyncOp`/`planEngramRootSync`) already discovers `agent-instructions/skills/*` generically and treats deploys as syncs (removals propagate, per vault note 442). Verified live: `engram update` printed `engram root: deleted skills/route/...` and `cleanup: removed dangling link ~/.claude/skills/route` (and the Pi equivalent) with zero code edits.
- [x] 5.3 Confirm `openspec validate --strict` still passes and no other doc/skill references `agent-instructions/skills/route/SKILL.md` as a live path (grep the repo) — `openspec validate --all --strict`: 44/44 clean. Fixed two stale live-path claims (`README.md`, `docs/architecture/adr.md`); historical/eval-artifact references left untouched. Flagged, not fixed (separate pre-existing change, out of scope): `openspec/changes/learn-rate-skill-only/` also references route/SKILL.md as live.

## 6. Close out

- [x] 6.1 Comment on GitHub issue #757 with the outcome: promoted (yes/no), D8 bar met (yes/no) with the actual numbers, SKILL.md retired (yes/no) — done, updated after the D8 loop-back: promoted yes, D8 met yes (3/3), SKILL.md retired yes
- [x] 6.2 N/A — D8 WAS met after the #762/#763 loop-back, so this condition never applied. Originally (5.* skipped): file a follow-up issue capturing the transcript finding from 4.3, since #757 itself should close as "promoted, defect investigated, D8 still open" rather than staying open indefinitely on unstated scope
- [x] 6.3 `openspec archive route-skill-to-runbook` once 6.1 (and 6.2 if applicable) are done — done
