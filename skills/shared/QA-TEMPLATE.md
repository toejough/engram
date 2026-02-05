# QA Skill Template

**DEPRECATED:** Phase-specific QA skills have been replaced by the universal `qa` skill.

See:
- `skills/qa/SKILL.md` - The universal QA skill
- `skills/shared/CONTRACT.md` - Contract format for producer validation

---

## Migration Notes

The previous pattern of creating `<phase>-qa` skills (e.g., `pm-qa`, `design-qa`) has been replaced by a single universal QA skill that validates any producer against its SKILL.md contract.

### Old Pattern (Deprecated)

```yaml
---
name: <phase>-qa
description: <Brief description of what this QA validates>
role: qa
phase: <pm | design | arch | ...>
---
```

Each phase had a dedicated QA skill with hardcoded validation logic.

### New Pattern

1. **Producers** define contracts in their SKILL.md:
   ```yaml
   ## Contract

   ```yaml
   contract:
     outputs:
       - path: "docs/requirements.md"
         id_format: "REQ-N"
     traces_to:
       - "issue description"
     checks:
       - id: "CHECK-001"
         description: "Every requirement has REQ-N ID"
         severity: error
   ```
   ```

2. **Universal QA** (`skills/qa/SKILL.md`) extracts the contract and validates:
   - Reads producer's SKILL.md
   - Extracts `## Contract` section
   - Runs each check against artifacts
   - Yields appropriate result

### Benefits

- Single QA skill to maintain
- Validation logic defined by producers (who know their outputs best)
- Consistent contract format across all phases
- Easier to add new producers (just add Contract section)

---

## Yield Protocol

The yield types remain unchanged:

| Type | When to Use |
|------|-------------|
| `approved` | Work passes all checks |
| `improvement-request` | Issues producer can fix |
| `escalate-phase` | Upstream phase has problems |
| `escalate-user` | Cannot resolve without user input |
| `error` | Cannot proceed (e.g., unreadable SKILL.md) |

See [YIELD.md](./YIELD.md) for full protocol specification.

---

## References

- **skills/qa/SKILL.md**: Universal QA skill implementation
- **skills/shared/CONTRACT.md**: Contract format specification
- **ARCH-019**: Universal QA skill architecture decision
- **DES-001**: Contract YAML format design decision
- **DES-002**: Contract section placement design decision
