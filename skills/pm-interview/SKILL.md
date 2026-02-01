---
name: pm-interview
description: Structured problem discovery interview producing requirements with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# PM Interview

Interview to discover problems and produce requirements.md with REQ-NNN IDs.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Domain | Problem space - what problem, for whom, why it matters |
| Phases | PROBLEM → CURRENT STATE → FUTURE STATE → SUCCESS CRITERIA → EDGE CASES |
| Output | requirements.md with REQ-NNN IDs |
| Does NOT own | UI/UX (Design) | Technology choices (Architecture) |

## Interview Phases

| Phase | Goal | Key Questions |
|-------|------|---------------|
| PROBLEM | Identify the pain | What's broken? Who's affected? Impact? |
| CURRENT STATE | Map the present | How does it work today? Pain points? |
| FUTURE STATE | Define success | What should happen instead? |
| SUCCESS CRITERIA | Make measurable | How will we know it's working? |
| EDGE CASES | Handle exceptions | What could go wrong? |

## Requirements Format

Each REQ-NNN has:
- User story: "As a [persona], I want [capability], so that [benefit]"
- Acceptance criteria (checkboxes)
- Priority (P0/P1/P2)

## Rules

| Rule | Action |
|------|--------|
| Out of scope | Note for Design/Architecture, redirect to problem |
| Before proceeding | Must articulate problem in one sentence |
| Implementation | NEVER discuss - that's Architecture |

## Output Format

`result.toml`: `[status]`, requirements.md path, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname pm-interview` or see SKILL-full.md
