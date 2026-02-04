# Visual Verification TDD Design

Design for integrating visual/interaction verification into TDD workflow documentation.

**Traces to:** requirements.md

---

## DES-1: Unified Interface Testing Model

All user interfaces (UI, CLI, API) follow the same testing model: structure + behavior + properties. Skills should not treat "visual" as a special case.

### Structure

Add a **Testing User Interfaces** section to tdd-red-producer SKILL.md after the existing "Test Philosophy" section:

```markdown
## Testing User Interfaces

UI, CLI, and API are all user interfaces. Each follows the same testing model:

| Layer | Question | What to Test |
|-------|----------|--------------|
| Structure | Does it exist? | Element presence, argument parsing, endpoint existence |
| Behavior | Does it work? | Interaction -> event -> handler -> state -> output |
| Properties | Does it hold? | Invariants across all screens/commands/endpoints |

**Structure tests** verify existence:
- UI: Element exists with correct properties
- CLI: Command accepts expected arguments
- API: Endpoint exists, request/response shape correct

**Behavior tests** verify the full chain:
- UI: Click -> handler fires -> state changes -> UI updates
- CLI: Command runs -> processing -> output appears
- API: Request -> processing -> response returned

**Property tests** verify invariants:
- UI: "Every screen has X" verified across all screens
- CLI: "All commands support --help" verified for all commands
- API: "All endpoints return valid JSON" verified exhaustively
```

### Rationale

This design unifies REQ-1 through REQ-4 into a single conceptual model rather than scattered guidance. Skills reference this model consistently.

**Traces to:** REQ-1, REQ-2, REQ-3, REQ-4

---

## DES-2: Interface Type Examples

Provide concrete test examples for each interface type in tdd-red-producer SKILL.md.

### Structure

Add subsections under DES-1's section:

```markdown
### UI Testing Examples

**Structure test (element exists):**
```typescript
it('renders add-note button on every screen', () => {
  for (const screen of allScreens) {
    render(<screen.Component />);
    expect(screen.getByRole('button', { name: /add note/i })).toBeTruthy();
  }
});
```

**Behavior test (full chain):**
```typescript
it('add-note button opens note editor', () => {
  const onOpen = vi.fn();
  render(<Screen onNoteEditorOpen={onOpen} />);
  fireEvent.click(screen.getByRole('button', { name: /add note/i }));
  expect(onOpen).toHaveBeenCalled();
  expect(screen.getByRole('dialog', { name: /new note/i })).toBeVisible();
});
```

**Property test (invariant):**
```typescript
it.prop([fc.constantFrom(...allScreens)])('all screens have add-note affordance', (screen) => {
  render(<screen.Component />);
  return screen.queryByRole('button', { name: /add note/i }) !== null;
});
```

### CLI Testing Examples

**Structure test (command parses):**
```go
func TestCommandAcceptsFlags(t *testing.T) {
    cmd := NewRootCmd()
    cmd.SetArgs([]string{"notes", "add", "--title", "Test"})
    Expect(cmd.Execute()).To(Succeed())
}
```

**Behavior test (output produced):**
```go
func TestCommandOutputsJSON(t *testing.T) {
    var buf bytes.Buffer
    cmd := NewRootCmd()
    cmd.SetOut(&buf)
    cmd.SetArgs([]string{"notes", "list", "--json"})
    Expect(cmd.Execute()).To(Succeed())
    var notes []Note
    Expect(json.Unmarshal(buf.Bytes(), &notes)).To(Succeed())
}
```

### API Testing Examples

**Contract test (shape correct):**
```go
func TestEndpointReturnsExpectedShape(t *testing.T) {
    resp := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/notes", nil)
    handler.ServeHTTP(resp, req)
    Expect(resp.Code).To(Equal(200))
    var body map[string]any
    Expect(json.Unmarshal(resp.Body.Bytes(), &body)).To(Succeed())
    Expect(body).To(HaveKey("notes"))
}
```
```

**Traces to:** REQ-2, REQ-3, REQ-9, REQ-10

---

## DES-3: Visual Verification Marker Convention

Tasks requiring visual verification are marked with `[visual]` tag in the task title.

### Format

```markdown
### TASK-5: [visual] Add carousel navigation buttons

**Description:** ...

**Acceptance Criteria:**
- [ ] Left/right buttons render on carousel
- [ ] Buttons navigate between slides
- [ ] Visual appearance matches design spec
```

### Detection Rules

Skills detect the marker by:
1. Check if task title contains `[visual]`
2. Check if any AC mentions "visual", "screenshot", "appearance", or "looks"

### Location

The marker appears in:
- tasks.md task titles
- Task references in yield payloads

### Skill Integration

| Skill | On `[visual]` Detection |
|-------|-------------------------|
| breakdown-producer | Apply marker to tasks affecting user-visible output |
| tdd-green-producer | Prompt for visual verification before yielding complete |
| tdd-qa | Require visual evidence or explicit waiver |

**Traces to:** REQ-7

---

## DES-4: Breakdown-Producer Visual Task Detection

Breakdown-producer applies `[visual]` marker automatically based on heuristics.

### Heuristics

Add to breakdown-producer SKILL.md under Task Format:

```markdown
## Visual Task Detection

Apply `[visual]` marker to tasks when:

1. **Files created/modified** include:
   - UI components (`.tsx`, `.vue`, `.svelte`)
   - CSS/styling files
   - CLI output formatting code
   - Template/view files

2. **Description mentions**:
   - "display", "show", "render", "appearance"
   - "button", "dialog", "modal", "form"
   - "output format", "table", "color"

3. **Acceptance criteria reference**:
   - Visual properties (size, color, position)
   - User-visible behavior
   - Design spec compliance

### Example

Task affects `components/Button.tsx` and AC says "button displays loading spinner":

```markdown
### TASK-7: [visual] Add loading state to submit button
```
```

**Traces to:** REQ-7, REQ-8

---

## DES-5: TDD-Green-Producer Visual Verification Step

Add visual verification prompt to tdd-green-producer when task has `[visual]` marker.

### Structure

Add after "PRODUCE Phase" section:

```markdown
## Visual Verification

When the task has `[visual]` marker:

1. **After tests pass**, verify visual output:
   - **Web UI**: Take screenshot via Chrome DevTools MCP (`mcp__chrome-devtools__take_screenshot`)
   - **CLI**: Capture terminal output to file
   - **API**: Not applicable (no visual component)

2. **Compare against expectation**:
   - If design spec exists: `projctl screenshot diff --baseline <spec> --current <screenshot>`
   - If no baseline: Manual review of screenshot

3. **Document evidence**:
   - Include screenshot path in yield payload
   - Note any visual issues discovered

### Yield with Visual Evidence

```toml
[yield]
type = "complete"

[payload]
artifact = "components/Button.tsx"
files_modified = ["components/Button.tsx"]
tests_passing = ["TestButtonLoading"]
visual_verified = true
visual_evidence = "screenshots/button-loading-state.png"

[[payload.decisions]]
context = "Visual verification"
choice = "Screenshot captured and reviewed"
reason = "Matches design spec"
```

### No Screenshot Capture Tool

If `projctl screenshot capture` is not available:

1. Use Chrome DevTools MCP for web: `mcp__chrome-devtools__take_screenshot`
2. For CLI: redirect output to file (`cmd > output.txt`)
3. For terminal with ANSI: use `script` command or equivalent

Visual verification is required even without dedicated tooling.
```

**Traces to:** REQ-5, REQ-6, REQ-8

---

## DES-6: TDD-QA Visual Evidence Requirement

TDD-QA requires visual evidence for tasks with `[visual]` marker.

### Structure

Add to tdd-qa SKILL.md under "REVIEW Phase":

```markdown
### Visual Task Validation

For tasks with `[visual]` marker:

1. **Check for visual evidence in producer yield**:
   - `visual_verified = true`
   - `visual_evidence` path provided

2. **If evidence missing**, yield `improvement-request`:
   ```toml
   [yield]
   type = "improvement-request"

   [payload]
   from_agent = "tdd-qa"
   to_agent = "tdd-green"
   issues = ["Visual verification required for [visual] task but no evidence provided"]
   ```

3. **Waiver process**: If visual verification is impractical:
   - Producer must explain why in yield
   - QA escalates to user for approval

### Visual Evidence Checklist

For `[visual]` tasks, add to review:

- [ ] Screenshot or output capture provided
- [ ] Visual matches acceptance criteria
- [ ] No obvious visual defects (blank, corrupted, misaligned)
```

**Traces to:** REQ-5, REQ-8

---

## DES-7: Screenshot Tooling Recommendation

Document MCP-based capture as primary mechanism; defer `projctl screenshot capture` implementation.

### Rationale

1. Chrome DevTools MCP already provides `take_screenshot` - no new code needed
2. CLI output is trivially captured via shell redirection
3. `projctl screenshot diff` already exists for comparison
4. Implementing `projctl screenshot capture` adds maintenance burden for marginal benefit

### Documentation Location

Add to tdd-green-producer SKILL.md (see DES-5) and create brief reference in project README or docs.

### Capture Approaches by Interface Type

| Interface | Capture Method | Tool |
|-----------|----------------|------|
| Web UI | Browser screenshot | `mcp__chrome-devtools__take_screenshot` |
| CLI | Output redirection | `command > output.txt 2>&1` |
| CLI (ANSI) | Script recording | `script -q output.txt command` |
| Desktop app | Manual screenshot | System screenshot tool |

### Comparison

Use existing `projctl screenshot diff --baseline <expected> --current <actual>` for SSIM-based comparison.

**Traces to:** REQ-6

---

## DES-8: CLAUDE.md Update Structure

Expand existing lessons rather than adding new sections.

### Approach

1. **Do not add new subsection** - existing Code & Debugging section has relevant lessons
2. **Expand three existing lessons** to cover all interfaces:
   - "UI testing verifies visual correctness..." - add CLI/API
   - "UI validation is critical..." - generalize to all user interfaces
   - "Test behavior, not just presence" - already general, ensure examples cover CLI/API

### Specific Changes

**Lesson: "UI testing verifies visual correctness..."**

Expand from UI-only to all interfaces:

```markdown
**Interface testing verifies correctness, not just existence**: When doing verification, actually check the output. For UI, look at the screenshot - DOM snapshots showing correct structure is not enough if the screenshot is blank or malformed. For CLI, check the actual output format - parsing success doesn't mean the output is usable. For API, validate response bodies - status 200 with malformed JSON is still wrong. "It runs" is not the same as "it works correctly." For user-facing tasks, acceptance criteria MUST include output verification. Use `projctl screenshot diff` with SSIM for visual regression detection.
```

**Lesson: "UI validation is critical..."**

Generalize:

```markdown
**Interface validation is critical, not optional**: For projects with user-facing output (UI, CLI, API), verification via appropriate tools is a REQUIRED part of the TDD/audit cycle - not something to skip when tools have issues. If verification tools are broken, FIX THEM before proceeding. Passing unit tests are necessary but not sufficient for user-facing components. Structure tests + behavior tests + output verification together constitute complete testing.
```

**Add new lesson after existing:**

```markdown
**Property-based testing for interfaces**: Express and verify properties that should hold across all screens/commands/endpoints, not just hand-picked examples. "Every screen has X" is a property that can be tested exhaustively. "All commands support --help" is a property. "All endpoints return valid JSON" is a property. Use randomized/exhaustive exploration (rapid, fast-check) to verify these invariants.
```

**Traces to:** REQ-11

---

## DES-9: Skill File Modification Summary

Summary of which files change and what content is added.

### tdd-red-producer/SKILL.md

| Location | Content |
|----------|---------|
| After "Test Philosophy" | DES-1: Testing User Interfaces section |
| After DES-1 section | DES-2: Interface type examples |

Estimated addition: ~80 lines

### tdd-green-producer/SKILL.md

| Location | Content |
|----------|---------|
| After "PRODUCE Phase" | DES-5: Visual Verification section |
| In yield examples | Visual evidence fields |

Estimated addition: ~50 lines

### tdd-qa/SKILL.md

| Location | Content |
|----------|---------|
| Under "REVIEW Phase" | DES-6: Visual Task Validation subsection |
| In checklist | Visual evidence items |

Estimated addition: ~30 lines

### breakdown-producer/SKILL.md

| Location | Content |
|----------|---------|
| After "Task Format" | DES-4: Visual Task Detection section |

Estimated addition: ~25 lines

### ~/.claude/CLAUDE.md

| Location | Content |
|----------|---------|
| Code & Debugging section | DES-8: Expanded/new lessons |

Estimated changes: ~15 lines modified, ~5 lines added

**Traces to:** REQ-8

---

## Design Decisions

### DD-1: Single Model vs Separate Sections

**Context:** Should UI/CLI/API each get their own testing sections, or unified model?

**Decision:** Unified model with examples for each type.

**Rationale:** The testing approach is identical (structure + behavior + properties). Separate sections would duplicate concepts and miss the key insight that these are all user interfaces.

### DD-2: Marker Syntax

**Context:** How to mark visual tasks? Options: `[visual]`, `[ui]`, `visual: true` field, tag system.

**Decision:** `[visual]` in task title.

**Rationale:**
- Visible in task list at a glance
- Simple grep/regex detection
- No schema changes to task format
- `[visual]` is clearer than `[ui]` since CLI output is also "visual"

### DD-3: Screenshot Capture Tooling

**Context:** Implement `projctl screenshot capture` or document existing tools?

**Decision:** Document existing tools (Chrome DevTools MCP, shell redirection).

**Rationale:**
- MCP already provides browser screenshots
- Shell handles CLI output
- `projctl screenshot diff` handles comparison
- New command adds maintenance for marginal value
- Can implement later if patterns emerge requiring it

### DD-4: CLAUDE.md Expansion vs New Section

**Context:** Add new "Visual Verification" section or expand existing lessons?

**Decision:** Expand existing lessons in Code & Debugging.

**Rationale:**
- Consolidate, don't fragment (per CLAUDE.md guidance)
- Visual verification is part of testing, not separate category
- Existing lessons already touch on this topic
- Keep flat structure
