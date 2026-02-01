---
name: meta-audit
description: Analyze correction patterns and propose skill/CLAUDE.md improvements
context: fork
model: opus
skills: ownership-rules
user-invocable: true
---

# Meta Audit

Analyze patterns in corrections to propose permanent improvements.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | corrections.jsonl | CLAUDE.md | skill files |
| Process | Read corrections | Identify patterns (2+ occurrences) | Draft CLAUDE.md additions | Draft skill edits | Validate |
| Rules | Only 2+ occurrences | Be specific | Fit existing taxonomy | Test against history |

## Pattern Criteria

| Requirement | Details |
|-------------|---------|
| Frequency | 2+ occurrences to qualify |
| Specificity | Exact text, location, rationale - not vague |
| Structure | Use existing CLAUDE.md sections, don't invent new ones |
| Consolidation | Check if belongs in existing rule first |

## What NOT to Propose

| Anti-Pattern | Why |
|--------------|-----|
| Single occurrence | Could be a fluke |
| Vague rules | "Be careful" doesn't prevent anything |
| Contradict existing | Must fit current CLAUDE.md rules |
| Tool changes | Only change prompts/rules, not tools |

## Validation

For each proposal: Would it have prevented ALL N occurrences?

## Result Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- Patterns found with frequency
- CLAUDE.md additions (exact text)
- Skill edits (exact changes)

## Full Documentation

`projctl skills docs --skillname meta-audit` or see SKILL-full.md
