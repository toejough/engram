# Compressed Skill Template

Skills should use this format when full documentation exceeds 500 tokens.
Keep compressed version under 500 tokens (~2000 chars). Store full docs in SKILL-full.md.

---

## Template Structure

```markdown
---
name: skill-name
description: Brief description
context: fork
model: sonnet
---

# Skill Name

One-sentence purpose.

## Quick Reference

Purpose | Key rules in brief, pipe-separated
--------|------------------------------------
Input | Context TOML via $ARGUMENTS | task ID | acceptance criteria
Process | Step1 | Step2 | Step3 (max 4-5 steps)
Rules | MUST do X | NEVER do Y | ALWAYS Z (3-5 rules max)
Output | result.toml | summary | files_modified list

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Tests pass unexpectedly | Feature exists or tests are wrong |
| Build fails | Check imports/types; minimal stubs if needed |
| Missing criteria | Map each acceptance criterion to test |

## Output Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- `[outputs]` files_modified=[]
- `[[decisions]]` context, choice, reason
- `[[learnings]]` content

## Full Documentation

For complete details: `projctl skill docs SKILL-NAME` or see SKILL-full.md
```

---

## Design Principles

1. **Pipe-delimited rules** - Quick scanning without prose
2. **3-5 items per section** - Cognitive limit for working memory
3. **Failure hints** - Most common issues first
4. **Link to full docs** - Details available on demand

## Token Budget

| Section | Max Tokens |
|---------|------------|
| Header/frontmatter | ~50 |
| Quick Reference table | ~150 |
| Failure Hints | ~100 |
| Output Format | ~100 |
| Full Docs link | ~50 |
| **Total** | **~450** |

## Conversion Checklist

When compressing existing SKILL.md:

- [ ] Extract 3-5 core rules as pipe-delimited entries
- [ ] Identify top 3 failure modes from experience
- [ ] Keep only essential output format fields
- [ ] Move detailed examples to SKILL-full.md
- [ ] Verify under 500 tokens (use `wc -w` / 1.3 as estimate)
