> Markdown-only change. Every eval arm is a fresh headless `claude -p` process under design D5's confinement, never a subagent. While a batch runs, the orchestrating session writes nothing to the real vault (vault note 956). Bars are the pre-registered ones in design D5/D6; no task loosens a bar. A failure goes back to Joe.

## 1. Harness (design D5)

- [x] 1.1 Build the confined-arm launcher in `dev/eval/learn-at-return/`, reusing existing harness plumbing where it exists (vault note 200).
  - For each arm, it creates:
    - `ARM=$(mktemp -d /private/tmp/engram-arm.XXXXXX)`;
    - `$ARM/{home,work,tmp,xdg,vault,bin}`;
    - `$ARM/home/.claude/settings.json`, with D5's sandbox and permissions;
    - `$ARM/home/.claude/engram/{recall,delegate,learn,shim}.md`, the full production guidance set, each with its marker line (`learn.md` also gets the arm token). recall, delegate and shim are pinned to 67911117; `learn.md` is the arm's version.
    - `$ARM/home/.claude/CLAUDE.md`, importing the four files with the same `@` lines and order as the real `~/.claude/CLAUDE.md`;
    - all six engram skills, pinned to 67911117;
    - the `unit-worker` fixture agent;
    - `$ARM/bin/engram`.
  - It launches the arm with `env -i` and exactly D5's variable allowlist, with the token read from the keychain per batch and never echoed.
  - It records `run-manifest.json` for each batch: the `claude` version, orchestrator model id, fixture model, commit SHAs of every text installed, and the tokens.
- [x] 1.2 Freeze the fixtures. Write two three-unit task prompts, one for Quillfeather and one for Tarnbrook, and the five unit-1 reports (P1, P2, N1, N2, N3) per D5's table. Before any arm runs, a fresh-context reviewer checks each P line against learn's Step-2 bar ("clears the bar") and each N line against it ("fails the bar"), and confirms each prompt is unambiguous (vault notes 1030/1037).
- [x] 1.3 Write the mechanical scorer over the arm's stream-json transcript. It implements D5's return event, window, fire, fast-path, pass, false-fire, not-scored and degenerate definitions, and the delivery-gate grep over every record type in the arm's session JSONL (vault note 939). The gate requires all four file markers and the arm's own token, and the other arm's token must be absent. Offline tests give each classification at least one hand-built transcript: pass, fired-with-sweep, direct `engram learn`, late fire, false fire, question-stop, degenerate, gate-fail (vault note 988a1).
- [x] 1.4 Write the per-batch guards:
  - **denyWrite list:** generate it from `ls /private/tmp/claude-<uid>` at batch setup.
  - **Write probe:** a probe arm tries to `touch` inside an existing entry. The batch aborts unless the touch fails and the file is absent afterwards.
  - **Isolation check:** save the files under the real vault (outside `.git/`) and the top-level `~/.claude` entries newer than the batch start, before and after the batch, to `results/<batch>/isolation.json`. A real-vault entry newer than the batch start fails the batch.
- [x] 1.5 Smoke: run one P1 GREEN arm and one N1 RED arm, using a provisional GREEN text; the smoke is not scored.
  - Confirm that the `Skill` and `Agent` tool names are right for the installed `claude`.
  - Confirm that all five delivery-gate tokens appear in each arm's transcript (if the imported files' content is not recorded, stop and ask Joe), the fixture agent returns its report verbatim, and the scorer classifies both arms.
  - Re-measure the per-arm cost with the full guidance set loaded; do not reuse any earlier figure. Report it and the projected total for D5's cells to Joe, and get confirmation before task 2.1.

## 2. Guidance RED (design D5, D7 step 1)

- [x] 2.1 Run the RED cells against `learn.md` at `67911117`: P (P1 5 + P2 5) = 10, N1 3, N2 3, N3 3.
  - Replace delivery-gate failures, degenerate runs and question-stops per D5.
  - Save the per-arm classification table to `results/red/scorecard.json`.
  - Report RED P and RED N as a labeled table, with units and counts per cell.

## 3. Guidance edit and GREEN (design D2, D5)

- [x] 3.1 Check D2's pre-written wording against the final `guidance-learn-moments` spec (vault note 1072), then apply enumeration rows 1–2 to `agent-instructions/guidance/learn.md`. Run each row's new-present and old-absent grep. `git diff` must show only those two hunks.
- [x] 3.2 Run the GREEN cells against the edited text: P = 10, N1 5, N2 5, N3 5. The same replacement rules apply.
- [x] 3.3 Apply D5's decision procedure:
  - **Ship gate:** GREEN P ≥ 8/10, N ≤ 1/15, and N1 0/5.
  - **Effect label:** from RED P.
  - **On a failed gate:** one wording revision with new tokens, then a fresh rerun of only the failing GREEN cells. A second failure stops the work and goes to Joe.
  - Record the verdict table. Columns: cell, arm, n scored, passes or false fires, discards, question-stops. State the full-guidance-set noise caveat (design Risks) next to it; no bar is adjusted for it.

## 4. learn SKILL.md (writing-skills TDD, design D3, D6)

- [x] 4.1 Use `superpowers:writing-skills`. RED: run cells L1 and L2 (n = 5 each) against learn SKILL.md at `67911117`, with the GREEN `learn.md` in the arm. Record the result against D6's RED expectations. If a RED cell already meets its GREEN bar, record it as premise-falsified per D6, tell Joe, and continue with GREEN as a non-regression check.
- [x] 4.2 Apply enumeration rows 3–4 and run each row's greps.
- [x] 4.3 GREEN: run L1 and L2 (n = 5 each). Bars: L1 5/5 write exactly one note, with no `engram ingest` and no `vocab stats`; L2 5/5 sweep, write no duplicate, and judge the other two.
- [x] 4.4 Pressure: run L3 (n = 5). Bar: 5/5 write only the 2 uncaptured worth-keeping lines and still sweep. On a miss, refactor the wording and rerun L1–L3.

## 5. please and route SKILL.md (writing-skills TDD, design D4, D6)

- [x] 5.1 please: RED Q1 (n = 5) against please SKILL.md at `67911117`.
  - Apply enumeration rows 8–9 and run the greps.
  - GREEN Q1 (n = 5). Bar: ≥ 4/5 fire on the fast path before unit 2. (The original "and the list entry is marked captured" half was dropped by Joe on 2026-10-01 — ruling T14; the closing learn dedupes by vault coverage, per L2.)
  - Pressure: Q1 with "keep moving, we're behind". Bar: ≥ 4/5.
- [x] 5.2 route: RED R1 (n = 5) against route SKILL.md at `67911117`.
  - Apply enumeration row 11 and run the greps.
  - GREEN R1 (n = 5). Bar: ≥ 4/5 fire on the fast path before the next dispatch.
  - `git diff` of route SKILL.md must show only the one bullet's wording.
- [x] 5.3 A fresh-context reviewer reads the final diffs of learn.md and the three SKILL.md files against design D2–D4 and enumeration rows 1–12. Every finding is fixed or rebutted until the reviewer ACKs.

## 6. Docs and records

- [x] 6.1 Append enumeration row 15: the `dev/eval/LEDGER.md` row `learn-at-return-guidance`. It records the task 3.3 verdict table, the effect label, discards and question-stops, the cost from `cost-log.jsonl`, and the results path.
- [x] 6.2 For every *no change* row in `enumeration.md`, re-read its cited lines and confirm that the reason still holds against the edited texts.

## 7. Deploy and verify

- [ ] 7.1 Run `engram update --with-guidance`, then `engram update`, from a directory outside the repo. Confirm that the deployed `learn.md` is byte-identical to `agent-instructions/guidance/learn.md` (`cmp`), at every canonical path the update output names, including at least `~/.claude/engram/guidance/learn.md` and `~/.pi/agent/engram/guidance/learn.md`. Confirm that the compat symlinks (`~/.claude/engram/learn.md`, and the Pi surface path) resolve to those files.
- [ ] 7.2 Confirm that the deployed `learn`, `please` and `route` SKILL.md files, at the paths `engram update` names, are byte-identical to their sources (`cmp`). Leave the skill-mirror refresh offers for Joe; do not accept or decline them.

## 8. Close-out (design D8, D9)

- [x] 8.1 Write `dev/eval/audit/escalation-decision-2026-09-29.md`. It quotes Joe's 2026-09-29 instruction, cites both reports' headline numbers (W2 4.5% vs 5.6%, caveat 3a; C0 0/2; C1 ≤ 81.6%, 70.2% without route records), and records the decision: none of D-F's layers escalates, the next lever is this change, and the revisit condition is unchanged. It links this change and task 3.3's verdict.
- [ ] 8.2 Enumeration row 25: tick `learn-rate-skill-only` task 3.4, with a pointer to the 8.1 file and this change. Enumeration rows 13–14: update the ROADMAP rows. Run each row's greps.
- [ ] 8.3 Run the requirement-header collision sweep (vault note 744) across all active changes and any archived-but-unsynced change, and read every sibling's tasks.md in full (vault note 757). `learn-rate-skill-only` must have no unticked task left.
- [x] 8.4 Run `openspec archive learn-rate-skill-only`. Confirm that `openspec/specs/learn/spec.md` contains `### Requirement: Closing learn curates collected LESSONS lines`, and that `openspec/specs/please/spec.md` contains `### Requirement: Orchestrator collects LESSONS lines from all dispatched work` and `### Requirement: Please passes collected LESSONS lines to closing learn`, each byte-for-byte.
- [ ] 8.5 Run `openspec validate learn-at-subagent-return --strict` and `openspec validate --all --strict`. Both must be clean.
- [ ] 8.6 Run `openspec archive learn-at-subagent-return`. Confirm that the MODIFIED requirements were applied to `openspec/specs/{learn,please}` and that `openspec/specs/guidance-learn-moments/spec.md` exists.
