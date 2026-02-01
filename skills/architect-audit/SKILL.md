---
name: architect-audit
description: Validate implementation against architecture specification
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Architect Audit

Validate implementation follows documented architecture patterns.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | architecture.md | project dir |
| Process | Load ARCH IDs | Check layers | Verify DI | Compare data models | Check file structure |
| Domain | Implementation solution space | Does NOT validate: requirements or design |

## Critical Rule

**"Code exists" ≠ "Architecture is followed"**

## Audit Categories

| Category | What to Verify |
|----------|----------------|
| Layer Boundaries | No imports crossing boundaries incorrectly |
| Dependency Direction | Dependencies flow inward (domain has no deps) |
| DI Pattern | Interfaces defined, implementations injected |
| Data Models | Fields, types, relationships match spec |
| Service Interfaces | Contract satisfied, wired at composition root |
| File Structure | Matches documented structure, naming conventions |

## Verification Steps

| Check | Action |
|-------|--------|
| Layers | Trace imports, verify inward flow |
| Models | Compare fields/types against spec |
| Services | Verify interface + implementation + DI wiring |
| Structure | Walk tree, compare to documented structure |

## Output Format

`result.toml`: `[status]`, violations by ARCH ID, `[[decisions]]`, `[[learnings]]`

## Full Documentation

`projctl skills docs --skillname architect-audit` or see SKILL-full.md
