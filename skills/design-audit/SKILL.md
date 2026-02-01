---
name: design-audit
description: Validate actual visual output against intended designs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Design Audit

Compare actual UI against design files via real screenshots.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | design.md | .pen files | project dir | viewports |
| Process | Load designs | Screenshot designs | Start app | Screenshot app | Side-by-side compare |
| Domain | User interaction space | Does NOT validate: requirements or architecture |

## Critical Rules

**"It renders" ≠ "It looks right"**
**"It looks reasonable" ≠ "It matches the design"**

Compare against ACTUAL DESIGNS (.pen files), not prose specs.
Check structure before style.

## Comparison Steps

| Step | Action |
|------|--------|
| 1. Reference | Screenshot each screen from .pen at target viewport |
| 2. Implementation | Screenshot running app at SAME viewport |
| 3. Structure first | Same panels/columns? All sections present? Same hierarchy? |
| 4. Details second | Colors, typography, spacing only after structure matches |

## Failure Modes

| Issue | Classification |
|-------|---------------|
| Missing design | DESIGN GAP - report and stop |
| Missing viewport | VIEWPORT GAP - report |
| Structure mismatch | CRITICAL - panel/section wrong |
| Style mismatch | MAJOR/MINOR - visual details differ |

## Output Format

`result.toml`: `[status]`, findings by DES ID with screenshots, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname design-audit` or see SKILL-full.md
