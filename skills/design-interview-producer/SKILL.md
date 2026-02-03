---
name: design-interview-producer
description: Gathers design decisions via user interview and produces design.md with DES-N IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: design
variant: interview
---

# Design Interview Producer

Gather design decisions through structured user interview and produce design.md with traced DES-N IDs.

**Pattern:** GATHER → SYNTHESIZE → PRODUCE (see [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md))

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Domain | User interaction space - workflows, layouts, interactions |
| Pattern | GATHER → SYNTHESIZE → PRODUCE |
| Input | requirements.md (REQ-N IDs), user responses |
| Output | design.md with DES-N IDs |
| Yields | `need-user-input`, `need-context`, `complete` |

## Workflow

### 1. GATHER Phase

Collect design information via interview:

1. Read context from `[inputs]` section
2. Check for `[query_results]` (resuming after need-context)
3. If requirements not available:
   - Yield `need-context` requesting requirements.md
4. Interview user about design preferences:
   - Yield `need-user-input` with design questions
   - Questions cover: visual style, layout preferences, component patterns, accessibility needs
5. Proceed to SYNTHESIZE when sufficient information gathered

**Yield `need-user-input` for:**
- Visual style preferences (colors, typography, spacing)
- Layout approach (responsive, fixed, adaptive)
- Component library preferences
- Accessibility requirements
- Brand guidelines or constraints

### 2. SYNTHESIZE Phase

Process gathered design information:

1. Map requirements to design elements
2. Identify design patterns needed
3. Resolve conflicts between requirements
4. If blocked, yield `blocked` with details
5. Structure findings for design.md

### 3. PRODUCE Phase

Create the design.md artifact:

1. Generate DES-N IDs for each design element
2. Include `**Traces to:**` links to REQ-N IDs
3. Write to configured path from context
4. Yield `complete` with artifact details

## DES Entry Format

```markdown
### DES-001: Login Screen Layout

Two-column layout with branding on left, form on right.
- Logo and tagline in left column
- Email/password form in right column
- "Forgot password" link below form

**Traces to:** REQ-001, REQ-002
```

## Yield Examples

### Interview Question

```toml
[yield]
type = "need-user-input"
timestamp = 2026-02-02T10:30:00Z

[payload]
question = "What visual style are you aiming for? (e.g., minimal, corporate, playful)"
context = "Establishing design direction before creating screen layouts"

[context]
phase = "design"
subphase = "GATHER"
awaiting = "user-response"
```

### Complete

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:30:00Z

[payload]
artifact = "docs/design.md"
ids_created = ["DES-001", "DES-002", "DES-003"]
files_modified = ["docs/design.md"]

[[payload.decisions]]
context = "Layout approach"
choice = "Mobile-first responsive"
reason = "User indicated mobile users are primary audience"
alternatives = ["Desktop-first", "Adaptive layouts"]

[context]
phase = "design"
subphase = "complete"
```

## Rules

| Rule | Action |
|------|--------|
| Missing requirements | Yield `need-context` to request requirements.md |
| Ambiguous preference | Yield `need-user-input` with clarifying question |
| Every DES-N | Must trace to at least one REQ-N |
| No implementation | Focus on WHAT the user sees, not HOW it's built |
