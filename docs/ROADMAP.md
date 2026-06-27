# Engram roadmap — memory value & cost

Forward-looking priorities, ranked by value-for-effort. Anchored to the 2026-06-26 verified benefit
ledger (vault notes 99/100) and the `$METER` + trap-gate prerequisites that landed the same day
(`dev/eval/traps/gate.py`, cumulative harness schema v5). Update as items complete.

## Where we are

- **Memory's only adversarially-verified wins are capability on idiosyncratic / un-derivable
  content** — C3 (apply un-guessable conventions), C4-idio (recency supersession), C5 (honor a
  recency-channel standard), C6 (abduction/synthesis). Cost/speed is net-negative on easy builds.
- Those wins are **measured on n=5 single-note toy vaults** — crowded-vault generalization is
  **untested** (the ledger critic flagged this as the load-bearing uncertainty).
- We now have the guardrails: a **trap regression gate** (catches a capability regression) and a
  **`recall_cost` `$METER`** (makes recall's dollars visible). Baseline smoke = GREEN.

## Ranked next steps

### #1 — Crowded-vault capability eval (highest value) — IN PROGRESS
**Do the 4 verified wins survive a realistic, crowded vault with competing notes, or only in n=5
toys?** Plant the load-bearing note(s) among many semantically-near distractors (varying recency),
For Tier-1 (free) run the **real multi-phrase `engram query`** — the 10-phrase retrieval call `/recall`
makes internally (note 72: validate through the real retrieval, not a single bare query); for Tier-2
run the **real warm harnesses** (which invoke `/recall`). Check both *surfaced* (retrieval precision
under load) and *applied* (the existing pass bar) vs the toy baseline.

**Why first:** it is the load-bearing uncertainty. If the value proposition doesn't hold at realistic
scale, every downstream cost/usage optimization is rearranging deck chairs. Note 72 predicts retrieval
may be robust (the 10-phrase recall already reaches bridges), so a *confirming* (null) result is a
real possibility and is itself valuable — the risk it removes is "we never checked."

### #2 — Clean baselines from the new instruments (cheap, high-info)
Run the trap gate at `--tier full` for the real verified-bar baseline (smoke was n=1), and a clean
`$METER` read across several ops, so "did a change help/hurt?" is answerable on both the capability and
dollar axes. **Why:** cheap confidence before any change; the instruments exist now, so this is low
effort.

### #3 — Payload-prune-after-Step-3 cost lever (small, safe now)
The single dollar lever the ledger endorses (~$1/op: drop the raw recall payload from build context
after Step-3 synthesis, keeping only the requirements list). Gate every attempt behind the trap
harness so a lost win is caught immediately. **Why last:** worth doing only if #1 confirms the value
is real; the ceiling is modest and "lighter prompts for more usage" was largely a dead end (firing is
set by the skill *description*, not the body).

## Explicitly deferred
The broader cost/time/friction trims (Step-2 payload paging restructure, inline candidate_l2 content,
async learn, async Step-2.6 linking) are real but time-axis, not dollars, and lower-leverage than #1.
Revisit after #1 resolves whether capability holds at scale.
