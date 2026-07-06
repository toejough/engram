# Route skill — baseline test index

Before editing `skills/route/SKILL.md`, run the `superpowers:writing-skills` TDD cycle —
RED baseline first, GREEN edit, pressure tests (CLAUDE.md mandates it). Same rule as the
recall/learn indexes; the difference is what carries the baseline:

| baseline | locks which behavior | re-run before editing |
|---|---|---|
| *(none — author a fresh RED scenario at edit time)* | The behaviors below, locked by the skill text itself (not a reusable fixture). | Whole skill |

## Behaviors locked by the skill text

- **Cheapest-first default** — every unit starts at the cheapest tier on a cold start, including work that *feels* hard (no "looks-hard"/"genuinely-hard" exception).
- **Spec-first escalation** — first fail rewrites the spec + retries the same tier; second fail escalates one tier.
- **Dispatch record** — every dispatch records work-kind/tier/model/why/outcome, with OUTCOME sourced from a review verdict, never the subagent's self-report.
- **Usage capture** — on Claude Code, the record's duration/cost come from the subagent's Task-completion `<usage>` block (`duration_ms`, `subagent_tokens`), unit-labeled, not `n/a`.
- **Evidence loop** — records auto-ingest as recallable memory and crystallize via `/learn`.
- **Memory-tier-discount** — a memory-backed unit routes one tier cheaper, floored at cheap.

RED/GREEN evidence records are **transient** (like the deleted `memory-discount-RED-GREEN.md`):
`git log` recovers them. The 2026-07-06 evidence-based-rubric cycle's record
(`evidence-rubric-RED-GREEN.md`) showed the old table over-provisioning 5/6 units → 0/6 after the
rewrite; it deletes at cycle close. The measured memory-discount claim lives at
`dev/eval/LEDGER.md#tier-routing-parity`.
