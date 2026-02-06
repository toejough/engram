# Summary: orchestrator-skill-contract

## Accomplishment

Added **ARCH-018: Orchestrator-Skill Contract** to `docs/architecture.md`, formally documenting the bidirectional communication protocol between the `/project` orchestrator and skills.

## Deliverables

| Artifact | Description |
|----------|-------------|
| `docs/architecture.md` | Added ARCH-018 section (~155 lines) |

## ARCH-018 Contents

1. **Context Input Format** - TOML structure for orchestrator → skill communication
2. **Modes** - interview, infer, update
3. **Yield Output Format** - TOML structure for skill → orchestrator communication
4. **Producer Yield Types** - 7 types with orchestrator actions
5. **QA Yield Types** - 4 types with orchestrator actions
6. **Resumption Protocol** - How orchestrator handles each of 11 yield types
7. **Query Result Injection** - How need-context results are provided

## Issue Resolved

- **ISSUE-24**: Create ARCH-N for explicit orchestrator-skill contract ✅

## Commit

- `a7169d6` - docs(arch): add ARCH-018 orchestrator-skill contract
