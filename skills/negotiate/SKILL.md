---
name: negotiate
description: Argue one side of a cross-skill disagreement with evidence
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Negotiate Skill

Argue one position in a cross-skill disagreement using evidence and traceability references.

## Purpose

When audit skills disagree (e.g., PM audit says a requirement isn't met but architect audit says the architecture makes it impractical), the orchestrator invokes this skill to argue each side. The skill receives a position and produces a reasoned argument or concession.

This skill is invoked multiple times per conflict -- once for each side, for up to 2 rounds of negotiation.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Conflict ID (CONF-NNN)
- Which side to argue (e.g., "pm" or "architect")
- The position to argue (what this side claims)
- The opposing argument (what the other side said, if round > 1)
- Traceability references (REQ/DES/ARCH/TASK IDs involved)
- Round number (1 or 2)
- Relevant artifact excerpts

## Process

1. **Understand the position** - Read the assigned position and supporting evidence
2. **Review traceability** - Check which upstream artifacts support this position
3. **Assess the opposing argument** (if round > 1) - Find weaknesses or acknowledge strengths
4. **Produce reasoned argument** - Support position with evidence from artifacts, OR
5. **Concede if warranted** - If the opposing argument is stronger, concede with justification

## Argumentation Rules

1. **Evidence-based only** - Every claim must reference a specific artifact and traceability ID
2. **No ad hominem** - Argue positions, not skills
3. **Acknowledge valid points** - If the other side has a strong point, say so
4. **Propose compromises** - If neither position is clearly better, suggest a middle ground
5. **Be concise** - Maximum 2-3 key arguments per round
6. **Reference traceability** - Use REQ/DES/ARCH/TASK IDs in arguments

## Concession Criteria

Concede when:
- The opposing argument has stronger traceability backing
- The position requires violating a higher-priority constraint
- A compromise would satisfy both sides' core concerns
- The position was based on a misunderstanding of the artifact

Do NOT concede just to end the negotiation. Only concede when the evidence supports it.

## Structured Result

```
Status: success
Summary: [Argued position | Conceded] for CONF-NNN round N.
Outcome: argue | concede | compromise
Argument:
  position: <what this side claims>
  key_points:
    - point: <argument>
      evidence: <artifact reference, traceability ID>
    - point: <argument>
      evidence: <artifact reference>
  acknowledged_opposing_points:
    - <valid point from other side>
  weaknesses_in_opposing:
    - <weakness found>
      evidence: <why>
Proposed resolution: <if compromise or concession, what specifically to do>
Traceability: [IDs referenced in arguments]
```
