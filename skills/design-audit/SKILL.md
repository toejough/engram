---
name: design-audit
description: Validate actual visual output against intended designs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Design Audit Skill

Validate actual visual output against intended visuals (design files, .pen mockups).

## Purpose

Compare what the implementation actually looks like against what it was designed to look like. This means real screenshots and visual comparison -- not just reading markdown descriptions.

**"It renders" is not the same as "it looks right." And "it looks reasonable" is not the same as "it matches the design."**

## Domain Ownership

This skill audits within the **user interaction solution space** (same domain as `/design-interview`).

**Validates:**
- Does actual UI match intended design?
- Are interactions working as designed?
- Is terminology, feedback, and flow correct?

**Does NOT validate:**
- Requirements fulfillment → `/pm-audit`
- Architecture adherence → `/architect-audit`

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Path to design.md (with DES- IDs and .pen file references)
- Project directory
- How to start/access the application
- Viewport sizes to test
- Traceability references

## Audit Philosophy

**Compare against the ACTUAL DESIGNS, not just prose specs.** Open the .pen design files and screenshot each screen as your reference. Then screenshot the running app at the same viewport. Compare them.

**Never audit from text descriptions alone.** Prose specs describe intent; the .pen files show the exact layout, spacing, and structure the implementation should match.

**Check structure before style.** Verify the layout skeleton matches first (panels, columns, sections), then check visual details (colors, typography, spacing). A pixel-perfect color match is worthless if an entire panel is missing.

## Audit Steps

1. **Load design source files** - Read design.md to find .pen file paths and screen node IDs. Open .pen files.

2. **Validate design references -- STOP if any fail:**
   - For EVERY screen in the node ID table, get a screenshot of the design
   - If any screenshot fails, STOP and report as blocked
   - If a screen has no design, report as DESIGN GAP
   - If a viewport has no design, report as VIEWPORT GAP
   - **Never silently skip a screen or viewport**

3. **Start dev server** - Ensure app is running

4. **For each screen that HAS a design, do a SIDE-BY-SIDE comparison:**

   a. **Reference screenshot** - From step 2
   b. **Implementation screenshot** - Navigate to the screen, screenshot at SAME viewport size

   c. **Structural comparison (do this FIRST):**
   - Same number of layout panels/columns?
   - All designed sections present?
   - Same element hierarchy?
   - No extra elements not in design?
   - No missing elements that are in design?

   d. **Redundancy and clutter check:**
   - No duplicate interactive elements
   - Elements appropriate for the viewport
   - No orphaned text or icons without context

   e. **Edge and boundary check:**
   - No elements cut off at container edges
   - No elements butting against edges without padding
   - No overflow causing horizontal scroll
   - Content fills containers appropriately

   f. **Visual detail check:**
   - Layout integrity (no overlapping elements)
   - Proper spacing between components
   - Correct visual hierarchy
   - Colors match design tokens
   - Typography matches spec
   - Icon alignment
   - Responsive behavior

5. **Check interactions** (MANDATORY for any interactive element):

   **Every clickable element must be clicked. Every input must receive input. Every form must be submitted.**

   a. **Inventory all interactive elements:**
   - Buttons, links, nav items
   - Form inputs, selects, checkboxes
   - Carousels, tabs, accordions
   - Any element with a pointer cursor

   b. **For EACH interactive element, verify the FULL behavior chain:**
   - Element is visible and reachable
   - Click/interaction triggers expected response
   - State change occurs (UI updates, navigation happens, data changes)
   - If no visible response, this is a DEFECT

   c. **Document each interaction:**
   - What you clicked/typed
   - What happened (or didn't happen)
   - Expected vs. actual behavior

   **"Button renders" is not "button works." A button that does nothing on click is a blocking defect, even if it looks correct.**

   d. **Verify hover/active states**
   e. **Check navigation state updates**

6. **Use screenshot diff tooling if available:**
   - `projctl screenshot diff` for SSIM-based comparison
   - Report similarity scores and spatial clusters of differences
   - The tool measures, this skill judges significance

## Visual Bug Categories

| Category | What to Look For |
|----------|------------------|
| **Overlap** | Elements colliding, content on top of other content |
| **Spacing** | Missing margins, cramped layouts, inconsistent gaps |
| **Alignment** | Off-center content, ragged edges, uneven columns |
| **Overflow** | Text clipping, horizontal scroll, content outside containers |
| **Sizing** | Elements too large/small, incorrect aspect ratios |
| **Color** | Wrong colors, poor contrast, missing states |
| **Typography** | Wrong fonts, sizes, weights, line heights |
| **Responsiveness** | Layout breaks at breakpoints, wrong nav showing |

## Findings Classification

| Classification | Meaning | Action |
|----------------|---------|--------|
| **DEFECT** | Implementation wrong, design spec is correct | Fix implementation |
| **SPEC_GAP** | Design spec missed something implementation handles well | Propose spec addition |
| **SPEC_REVISION** | Design spec was impractical, implementation found better way | Propose spec change |
| **CROSS_SKILL** | Finding affects requirements or architecture domain | Flag for resolution |

## Works for All Project Types

This skill is tool-agnostic. It specifies **what** to validate, not which tool to use:
- **GUI apps**: Use Pencil MCP for designs, Chrome DevTools or browser for screenshots
- **TUI apps**: Use terminal captures
- **CLI apps**: Compare expected vs actual terminal output

Use whatever tools are available to capture actual visual output. The principle is the same: compare actual against intended.

## Structured Result

```
Status: success | failure | blocked
Summary: Audited N screens across M viewports. X pass, Y fail.
Design gaps: [screens with no design reference]
Viewport gaps: [viewports with no design]
Findings:
  defects:
    - id: DES-NNN
      screen: <name>
      viewport: <size>
      category: <overlap|spacing|alignment|etc>
      description: <what's wrong>
      severity: blocking | warning | info
      evidence: <screenshot references, diff scores>
  proposals:
    - id: DES-NNN
      current_spec: <what design shows>
      proposed_change: <what to change>
      rationale: <why>
  cross_skill:
    - id: DES-NNN
      conflicts_with: <REQ-NNN or ARCH-NNN>
      issue: <description>
Traceability: [DES IDs audited]
Screens audited: X/Y
Recommendation: PASS | FIX_REQUIRED | PROPOSALS_PENDING
```
