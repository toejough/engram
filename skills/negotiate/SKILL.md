---
name: negotiate
description: Argue one side of a cross-skill disagreement with evidence
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Negotiate

Argue one position in cross-skill disagreements using evidence.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | conflict ID | position | opposing argument | round (1 or 2) |
| Process | Understand position | Review traceability | Assess opposing | Argue with evidence OR Concede |
| Rules | Evidence-based only | Acknowledge valid points | Max 2-3 key arguments | Reference traceability IDs |

## Argumentation

| Principle | Details |
|-----------|---------|
| Evidence | Every claim references specific artifact + traceability ID |
| Concision | Max 2-3 key arguments per round |
| Fairness | Acknowledge opposing valid points |
| Compromise | Propose middle ground if neither clearly better |

## Concession Criteria

| When to Concede | Reason |
|-----------------|--------|
| Stronger traceability | Opposing has better artifact backing |
| Higher priority violated | Position requires breaking higher constraint |
| Misunderstanding | Position based on artifact misread |
| Compromise available | Both core concerns can be satisfied |

Do NOT concede just to end negotiation - only when evidence supports it.

## Result Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool, outcome=argue|concede|compromise
- Key points with evidence
- Proposed resolution if compromise/concession

## Full Documentation

`projctl skills docs --skillname negotiate` or see SKILL-full.md
