# curate task: why this ask (GitHub #759, convert the `curate` skill to a runbook)

**The ask** (`task-prompt.txt`): "A few notes submitted to our beekeeping memory vault (the engram vault you're
already set up with) are sitting there as pending offers that nobody has reviewed yet. Please go through them and
curate them."

This is the **explicit-ask variant**. A second, **system-signal variant** is deliberately deferred: the agent does
routine work, engram itself prints the pending-offers notice mid-turn (query payload `pending_offers: true`, update
notice, or the write-path nudge), and the question is whether the agent curates unprompted. That tests the skill's
*trigger*, which only matters once the explicit-ask row shows the procedure itself is followable.

## The fixture (vault-only)

`vault-template/` is a small fictional beekeeping vault, built by `build_vault_template.sh` with the real
`engram learn` (real sidecars, real embeddings), then three notes marked `pending: true` (only a served write sets
that; no CLI flag does). The harness copies it to the trial's `ENGRAM_VAULT_PATH` (new `vault_template` task.json
key). There is no real-vault copy, so nothing engram/please/curate-related can leak into the bare arm (note 996).
The repo is a two-file stub; it exists only as the agent's cwd.

| # | Note | Role | Expected end state (curate/SKILL.md, verified against real `engram amend`) |
| --- | --- | --- | --- |
| 1, 4, 5, 6 | inspection interval, queen colour, smoker fuel, site-B flood line | untouched decoys | byte-identical |
| 2 | oxalic vapor only when broodless | near target | keeps original claim, now also says repeat 3x at 5-day intervals |
| 3 | pull frames only at 80% capped | covered target | claim intact; reinforced (`--activate`: sidecar `last_used` set) |
| 7 | offer: "spun uncapped frames ... confirm ~4/5 capped" | COVERED by 3 (reworded, nothing omitted) | discarded (md and sidecar gone) |
| 8 | offer: single oxalic dose misses late brood, repeat 3x / 5 days | NEAR 2 (same topic, one substantive extra claim) | folded into note 2 via `--object`, then discarded |
| 9 | offer: swarm-trap placement | ABSENT (no note on bait hives) | `--clear-pending`, file kept, content untouched |

Vault-wide: no pending left, `engram check` passes, exactly notes {1..6, 9} remain (so no `engram learn`, no
hand-written notes, no write-memory handoff).

## Why it is representative

| curate requirement | How the task forces it |
| --- | --- |
| Step 1: query cannot list offers; scan files for `pending: true` | The offers are only findable by scanning the vault dir. |
| Step 2: judge by content, three outcomes | One offer of each outcome; covered and near overlap the same topic areas as existing notes, so a similarity score alone does not separate covered from near. |
| Covered/near end in `--discard`, only absent uses `--clear-pending` | The natural shortcut (clear everything as "new information") is the bare agent's failure mode and is caught by the file-set and content checks. |
| Never `engram learn` / write-memory | Note-id set check. |
| Step 4 verify | `engram check` is scored in steps.json and re-run in done_when. |

## What is measured

- `steps.json` (12 steps): scan for pending, read each of the three offers, `engram query` for candidates,
  reinforce 3, discard 7, fold into 2, discard 8, clear 9, `engram check`, confirm-not-pending. Actions are not
  chained to the query step (an agent may judge from the files it read); discard-after-reinforce and
  discard-after-fold keep the skill's own ordering.
- `done_when_checks.sh`: repo-observable end state above. Reads the vault from `$ENGRAM_VAULT_PATH`, or
  `<trial>/vault-final` for `--rescore` (the harness snapshots the final vault there; it is otherwise deleted).
- Validation (note 1017a): the ideal state is produced by running the skill's steps literally with the real
  `engram amend` and must PASS; 15 single-defect mutants (untouched seed, covered cleared-not-discarded, covered
  left pending, covered not reinforced, near cleared / not discarded / claim lost / original overwritten, absent
  discarded / left pending / rewritten, all-three-cleared, extra note via `engram learn`, corrupt sidecar so
  `engram check` fails, unrelated note edited) must each FAIL. `engram activate --note` is accepted as an
  equivalent reinforcement.

## Known limits

- The spec (`vault-offer-curation`) says the marker is cleared "in every case"; the skill says covered/near offers
  are discarded. The skill's behavior is what is scored.
- `--activate` on the covered target is skill-prescribed but not something a bare agent has reason to do, so
  end_state also requires it; a bare agent that reasons perfectly but skips activation fails end_state.
- Prompt wording contains "curate", the skill's own noun; that is the explicit-ask variant by design.
