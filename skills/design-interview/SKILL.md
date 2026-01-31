---
name: design-interview
description: Visual and interaction design interview producing design specs with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Design Interview Skill

Transform product specifications into visual designs using Pencil MCP, through a structured interview process.

## Purpose

Take a product spec and produce a coherent visual design that:
- Matches the spec's requirements and constraints
- Reflects user's visual preferences
- Prioritizes simplicity and usability
- Uses Pencil MCP for all design work

Every design element gets a `DES-NNN` traceability ID linked to upstream REQ IDs.

## Domain Ownership

This skill owns the **user interaction solution space**: how users accomplish their goals through the interface.

**Owns:**
- User workflows and journeys
- Screen layouts and information hierarchy
- Interaction patterns and affordances
- Error states, feedback, and guidance
- Terminology and microcopy
- Transitions and navigation

**Does NOT own (defer to other phases):**
- What problem we're solving → PM
- Whether a feature should exist → PM
- Technology choices and constraints → Architecture
- Data models and API design → Architecture
- Performance and scalability decisions → Architecture

Note: "Design" here means interaction design, not visual styling. The focus is on how users accomplish tasks, regardless of colors or typography.

When users raise out-of-scope concerns, acknowledge them, note for the appropriate phase, and redirect: "That sounds like a PM/Architecture concern. I'll note it for that phase. For now, let's focus on user interactions."

## Inputs

- Product specification file (requirements.md with REQ- IDs)
- User's visual/UX preferences (discovered through interview)

## Outputs

- design.md with DES- IDs and REQ- traceability references
- Design system (colors, typography, spacing, components)
- Key screens as .pen files
- Component library for reuse

## Phases

### 1. UNDERSTAND - Read the Spec

**Goal:** Internalize requirements before designing anything.

**Steps:**
1. Read the spec file completely
2. Identify core user flows (what screens are needed?)
3. Note constraints (platform, simplicity requirements, etc.)
4. List P0 features that must appear in initial designs
5. Identify UI patterns implied by requirements

**Extract from spec:**
- Success criteria (REQ IDs) -> what must the UI enable?
- User stories (REQ IDs) -> what flows are needed?
- Acceptance criteria (REQ IDs) -> what interactions are required?
- Constraints -> what limits the design?
- Approaches to avoid -> what NOT to do?

**Summarize understanding before proceeding.**

### 2. PREFERENCES - Interview for Visual Direction

**Goal:** Understand user's aesthetic preferences without overwhelming with options.

**Core questions:**

1. **Density:** Do you prefer airy/spacious layouts or compact/information-dense?

2. **Style:** Which feels more like you?
   - Minimal and clean (lots of whitespace, subtle)
   - Warm and friendly (rounded, soft colors)
   - Bold and confident (strong colors, clear hierarchy)
   - Utilitarian and fast (dense, keyboard-first)

3. **Reference apps:** What apps do you enjoy using? What do you like about their design?

4. **Anti-references:** Any apps whose design you dislike? What bothers you?

5. **Color:** Any color preferences or constraints? (Brand colors, accessibility needs)

6. **Platform conventions:** How native should it feel? (iOS-like, web-like, custom)

**Listen for:**
- Speed vs. aesthetics tradeoffs
- Information density preferences
- Touch vs. keyboard expectations
- Emotional tone (calm, energetic, serious, playful)

**Do NOT show designs until preferences are understood.**

### 3. SYSTEM - Establish Design Tokens

**Goal:** Create the foundational design system before any screens.

**Use Pencil MCP to define:**

**Colors (assign DES IDs):**
- Primary (main actions, active states)
- Secondary (supporting actions)
- Background (app background)
- Surface (cards, elevated elements)
- Foreground (text)
- Muted (secondary text, borders)
- Semantic (success, warning, error)
- Dark mode variants

**Typography (assign DES IDs):**
- Font families (display, body, mono if needed)
- Size scale (xs, sm, base, lg, xl, 2xl, 3xl)
- Weight scale (normal, medium, semibold, bold)
- Line heights

**Spacing:**
- Base unit (4px or 8px typically)
- Scale (1, 2, 3, 4, 6, 8, 12, 16...)

**Radii:**
- None, sm, md, lg, xl, full/pill

**Shadows:**
- Subtle, medium, pronounced

**Create in Pencil:**
- Variables/tokens document
- Apply to a simple test component to validate

**Confirm design tokens with user before proceeding to screens.**

### 4. COMPONENTS - Build the Primitives

**Goal:** Create reusable components before full screens. Assign DES IDs to each component.

**Identify from spec which components are needed.**

**Common components:**
- Button (primary, secondary, ghost, danger)
- Input (text, with icon, error state)
- Card (container for content)
- List item (for collections)
- Navigation (tab bar, header)
- Modal/sheet (overlays)

**For each component:**
1. Assign a DES-NNN ID
2. Create in Pencil as reusable component
3. Show user for feedback
4. Iterate until approved
5. Document states (default, hover, pressed, disabled)

**Build smallest components first, compose into larger ones.**

### 5. FLOWS - Map the Screens

**Goal:** Identify what screens are needed and how they connect.

**From the spec, identify:**
- Entry point (what does user see first?)
- Core loop (the main repeated action)
- Key screens for P0 features
- Navigation model (tabs, stack, drawer?)

**Create a simple flow diagram or list with DES IDs for each screen.**

**Confirm flow with user before designing screens.**

### 6. SCREENS - Design Key Views

**Goal:** Create the actual screen designs in Pencil. Each screen gets a DES ID.

**Prioritize:**
1. Core loop screens (where user spends most time)
2. Entry/capture screens (first impression, frequent use)
3. Navigation/chrome (how to move between screens)
4. Secondary screens (settings, detail views)

**For each screen:**
1. Assign DES-NNN ID
2. Start with mobile-first (if PWA for phones)
3. Use established components
4. Apply design tokens
5. Show user, gather feedback
6. Iterate until approved

**Design principles to follow:**
- Show what matters - hide what doesn't
- One primary action per screen
- Progressive disclosure (simple first, details on demand)
- Consistency with established components

**Use Pencil tools:**
- `get_editor_state` - check current file
- `batch_get` - read existing components
- `batch_design` - create/update designs
- `get_screenshot` - validate visually
- `get_style_guide` - for inspiration if stuck

### 7. REVIEW - Validate and Refine

**Goal:** Ensure designs meet spec requirements.

**Checklist against spec:**
- [ ] All P0 features have a home in the UI
- [ ] Core user stories are achievable
- [ ] Acceptance criteria are supported
- [ ] Constraints are respected
- [ ] "Approaches to avoid" are avoided
- [ ] Simplicity principle maintained

**Review with user:**
- Walk through each screen
- Trace key user flows
- Ask: "Does this feel right?"
- Note concerns and iterate

**Finalize:**
- Ensure all components are properly named
- Document any design decisions
- Export/organize .pen files

## Traceability

### DES ID Assignment

- Assign sequential `DES-NNN` IDs starting from `DES-001`
- Every design token set, component, screen, and flow gets a DES ID
- Each DES ID must reference at least one upstream REQ ID

### design.md Structure

```markdown
# Design Specification

## Design System

### Colors (DES-001)
Traces to: REQ-NNN (accessibility), REQ-NNN (brand)
...

### Typography (DES-002)
Traces to: REQ-NNN
...

## Components

### DES-003: Button Component
Traces to: REQ-NNN (interaction model)
...

## Screens

### DES-010: Home Screen
Traces to: REQ-NNN (user story), REQ-NNN (success criterion)
...

## Node ID Reference

| Screen | .pen File | Node ID | Viewport |
|--------|-----------|---------|----------|
| Home | designs/app.pen | abc123 | 390x844 |
...
```

## Pencil MCP Usage

**Starting a design session:**
```
1. get_editor_state(include_schema: true) - understand current state
2. get_guidelines(topic: "design-system") - get design rules
3. get_style_guide_tags() - see available styles
4. get_style_guide(tags: [...]) - get inspiration
```

**Creating designs:**
```
1. batch_get(patterns: [{reusable: true}]) - see existing components
2. batch_design(operations: "...") - create/update
3. get_screenshot(nodeId: "...") - validate visually
```

**Iteration loop:**
```
1. Make changes with batch_design
2. Screenshot to validate
3. Show user, get feedback
4. Repeat until approved
```

## Interview Rules

1. **One question at a time** - Don't overwhelm with design options
2. **Show, don't tell** - Use screenshots and references over descriptions
3. **Iterate quickly** - Small changes, frequent feedback
4. **Respect constraints** - Don't propose what the spec prohibits
5. **Simplicity first** - Add complexity only when user asks

## Error Handling

**User can't articulate preferences:**
- Show 2-3 contrasting options
- Ask "Which feels more right?"
- Use reference apps as anchors

**Design doesn't match spec:**
- Revisit spec requirements
- Ask: "Should the spec change, or the design?"
- Document any spec amendments as findings

**Pencil errors:**
- Check file path is correct
- Verify node IDs exist
- Read error messages for guidance

**Scope creep:**
- "That sounds like a P2 feature. Should we note it and focus on P0 first?"

## Structured Result

When the design is complete, produce a result summary for the orchestrator:

```
Status: success
Summary: Conducted design interview. Produced design.md and .pen files with N design elements.
Files created: design.md, designs/*.pen
Traceability: DES-001 through DES-NNN assigned, all linked to REQ IDs
Findings: (any spec gaps or cross-skill issues discovered)
Context for next phase: (design system summary, key decisions, component inventory)
```

## Output Expectations

The design should be:
- **Spec-compliant** - All P0 requirements represented
- **Consistent** - Uses design system throughout
- **Simple** - No unnecessary complexity
- **Usable** - Core flows are obvious
- **Documented** - Components named and organized
- **Traceable** - Every element has a DES-NNN ID linked to REQ IDs
