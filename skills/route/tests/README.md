# Route skill — baseline test index

Before editing `skills/route/SKILL.md`, run the `superpowers:writing-skills` TDD cycle —
RED baseline first, GREEN edit, pressure tests (CLAUDE.md mandates it). Same rule as the
recall/learn indexes; the difference is what carries the baseline:

| baseline | locks which behavior | re-run before editing |
|---|---|---|
| *(none — author a fresh RED scenario at edit time)* | The memory-tier-discount rule (a memory-backed unit routes one tier cheaper than its surface-look tier, floored at cheap) is locked by the skill text itself, not by a reusable fixture. | Whole skill |

The rule's original RED/GREEN evidence record (`memory-discount-RED-GREEN.md`, showing the
router over-provisioning 4/6 memory-backed units before the rule) was deleted in the 2026-07
docs restructure — `git log` recovers it; the measured claim lives at
`dev/eval/LEDGER.md#tier-routing-parity`.
