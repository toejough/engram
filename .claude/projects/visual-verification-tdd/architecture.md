# Visual Verification TDD Architecture

**Traces to:** design.md

---

## ARCH-1: Documentation-Only Project

This project modifies skill documentation files only. No code architecture decisions required.

### Affected Files

| File | Type | Change |
|------|------|--------|
| `~/.claude/skills/tdd-red-producer/SKILL.md` | Skill doc | Add interface testing section |
| `~/.claude/skills/tdd-green-producer/SKILL.md` | Skill doc | Add visual verification section |
| `~/.claude/skills/tdd-qa/SKILL.md` | Skill doc | Add visual evidence requirement |
| `~/.claude/skills/breakdown-producer/SKILL.md` | Skill doc | Add visual task detection |
| `~/.claude/CLAUDE.md` | User config | Expand existing lessons |

### No Code Changes

- No Go code modified
- No test files created (skill docs are verified by TDD doc-testing framework per ISSUE-002)
- No CLI commands added (screenshot capture deferred per DD-3)

**Traces to:** DES-7, DES-9
