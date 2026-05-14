# Baseline RED results — current skills' behavior

Five fresh general-purpose subagents were each loaded with the full text of all four current skills (`capturing-fleeting-notes`, `promoting-to-permanent-notes`, `learn` in-repo, `remember` in-repo) and handed one scenario. Each was asked to describe — not execute — what they would do.

**Meta-finding.** The in-repo `/remember` and `/learn` skills already apply Recurs / Actionable / Right-Home gates. The two-stage capture→promote rhythm is unique to the global vault skills, but for the simulated "user asks /remember" scenarios, the subagents reached for the in-repo `/remember` rather than the global capturing skill. So RED is less stark than the plan predicted: the new skill mainly consolidates four skills into one, switches the backend from the SBIA store (`engram learn`) to the vault (`engram learn`), and removes the interactive-approval step.

The new skill should still be verified per scenario in Task 4 against its own expected behavior, but the RED→GREEN behavioral *delta* is consolidation, not gate-introduction.

---

## baseline-project-specific

**Subagent reached for:** in-repo `/remember` skill.

**Behavior:** Classified as feedback. Recurs gate: PASS (subagent generalized "cyclomatic complexity refactoring" away from the project naming). Right-Home gate: NO — flagged `.claude/rules/go.md` as the right home and would ask the user before saving. Would NOT silently write; would ask user to override the right-home gate.

**Subcommand that would be invoked on approval:**
```
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source human
```

**Would write:** Not without user approval (right-home conflict gating).

---

## baseline-hindsight-framing

**Subagent reached for:** in-repo `/remember` skill.

**Behavior:** Recognized the hindsight problem and explicitly reframed "when fixing context cancellation" → "When writing concurrent Go code with context" in its draft. Recurs / Actionable / Right-Home all pass. Would present drafted fields to user for approval, then write.

**Subcommand that would be invoked on approval:**
```
engram learn feedback --situation "When writing concurrent Go code with context" ... --source human
```

**Would write:** 1 note (after approval), with reframed situation.

---

## baseline-information-not-knowledge

**Subagent reached for:** in-repo `/remember` skill.

**Behavior:** Recurs gate PASS (generalized "targ" → "using a build tool"). Actionable gate FAIL: "prints warnings in yellow" doesn't change agent behavior. Suggested `.claude/rules/` as alternative home if worth capturing at all. Would NOT write.

**Would write:** 0 notes — fails Actionable gate.

---

## baseline-clean-write

**Subagent reached for:** in-repo `/remember` skill.

**Behavior:** Classified as feedback. All three gates pass. Reframed situation to "Debugging tool integration errors after committing code". Would present for approval, then write via `engram learn feedback --source human`.

**Subcommand that would be invoked on approval:**
```
engram learn feedback --situation "Debugging tool integration errors after committing code" --behavior "Re-run the build tool before investigating LSP errors" --impact "..." --action "..." --source human
```

**Would write:** 1 note (after approval) to SBIA backend, not the vault.

---

## baseline-autonomous-trigger

**Subagent reached for:** in-repo `/learn` skill (the only one with an autonomous post-task-boundary trigger).

**Behavior:** Identified all three candidates. Applied Recurs gate to each:
- Candidate 1 (Luhmann IDs): FAIL Recurs (project-specific flags).
- Candidate 2 (build tool stderr quirk): FAIL Recurs (project-specific).
- Candidate 3 (typo): FAIL Recurs (one-time event).

All three dropped. **Would NOT write any notes autonomously.** No `engram learn` calls.

**Would autonomously act:** NO — all three fail Recurs.

---

## Comparison to plan's expected RED baseline

| Scenario | Plan predicted RED | Actual RED |
|---|---|---|
| project-specific | "current skills write; neither rejects" | Recurs passes after generalization; right-home gate would ask user — would NOT silently write |
| hindsight | "skills write situation as-given, hindsight baked in" | Subagent already reframes to activity+domain |
| info-not-knowledge | "captures as fleeting fact and likely promotes" | Rejects at Actionable gate; would NOT write |
| clean-write | "two stages, ~doubled latency" | Single-stage via `/remember` → `engram learn` (SBIA, not vault) |
| autonomous | "no autonomous trigger exists" | `/learn` skill has autonomous trigger; would drop all three at Recurs |

## Implications for the new skill

The new unified `/learn` skill's value is therefore:

1. **One skill instead of four** — removes the "which one applies?" decision.
2. **Vault backend** — writes to the agent-memory zettelkasten via `engram learn`, replacing SBIA writes via `engram learn`. This affects how recall surfaces things and how the memory graph grows.
3. **Autonomous decide-and-write** — no interactive approval step. The agent decides and writes; the human curates by directing the work, not by approving each save.
4. **Single gate vocabulary** — Recurs, Activity+Domain (with mandatory reframe-once), Knowledge — applied at write time instead of being scattered across capture-vs-promote and skill-vs-skill.

The RED→GREEN delta is consolidation. Task 4 verifies the new skill produces the expected per-scenario behavior; the delta vs RED is informative but not the success criterion.
