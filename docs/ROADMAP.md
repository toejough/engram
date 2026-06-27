# Engram roadmap — memory value & cost

Forward-looking priorities, ranked by value-for-effort. Anchored to the 2026-06-26 verified benefit
ledger (vault notes 99/100) and the `$METER` + trap-gate prerequisites that landed the same day
(`dev/eval/traps/gate.py`, cumulative harness schema v5). Update as items complete.

## Where we are

- **Memory's only adversarially-verified wins are capability on idiosyncratic / un-derivable
  content** — C3 (apply un-guessable conventions), C4-idio (recency supersession), C5 (honor a
  recency-channel standard), C6 (abduction/synthesis). Cost/speed is net-negative on easy builds.
- Those wins **generalize to a realistic crowded vault** (tested 2026-06-26: all 4 hold with zero
  degradation under a 200-note real-vault crowd). Remaining bound: *same-domain competing* notes untested.
- We now have the guardrails: a **trap regression gate** (catches a capability regression) and a
  **`recall_cost` `$METER`** (makes recall's dollars visible). Baseline smoke = GREEN.

## Ranked next steps

### #1 — Crowded-vault capability eval (highest value) — DONE 2026-06-26
**RESULT: the wins generalize — all 4 hold with zero degradation under a 200-note real-vault crowd**
(C3 25/25, C4i 5/5, C5 5/5, C6 8/8; retrieval rank flat 0→400). Idiosyncratic notes are distinctive,
so a realistic (off-topic) crowd ranks strictly below them and doesn't degrade application. Bound:
*same-domain competing* notes remain untested. See `dev/eval/traps/RESULTS.md` + `README.md`.

**Approach (for the record):** crowd = variants of the real vault (+ links), swept 0→400; Tier-1 ran
the real multi-phrase `engram query` (free retrieval probe), Tier-2 the real warm harnesses (applied
check) vs a paired toy baseline. This was *the* load-bearing uncertainty — note 72 predicted retrieval
would be robust, and it was.

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
