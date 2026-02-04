# Visual Verification TDD Requirements

Requirements for integrating visual/interaction verification into TDD workflow for UI, CLI, and API changes.

**Linked Issues:** ISSUE-007, ISSUE-014

---

## REQ-1: UI/CLI/API Tasks ARE Testable

TDD skills must recognize that visual/interaction changes are first-class testable artifacts, not exceptions to the TDD process.

### Context

Current mental model treats code as testable and "visual stuff" as not really testable. This is incorrect. User-facing changes have:
- **Structure** that can be validated (thing exists with right shape/arguments/inputs)
- **Behavior** that can be tested (interaction produces expected result)
- **Properties** that should hold (invariants expressible as assertions)

### Acceptance Criteria

- [ ] tdd-red-producer SKILL.md documents that UI/CLI/API tasks follow normal TDD, not special treatment
- [ ] Skills do not skip or shortcut TDD for "visual" or "user-facing" tasks
- [ ] AC language for visual tasks is treated as testable specification, not prose description

**Traces to:** ISSUE-007

---

## REQ-2: Structure Testing (Thing Exists With Right Shape)

Tests must validate that UI elements, CLI outputs, and API endpoints exist with the correct structure.

### Context

Structure testing validates the "what exists" question:
- UI: Button/prompt/element exists with correct properties
- CLI: Command exists, accepts expected arguments, outputs expected format
- API: Endpoint exists, accepts expected inputs, returns expected shape

Example: "Every screen has an add-note affordance" decomposes to structural tests checking each screen for the affordance.

### Acceptance Criteria

- [ ] tdd-red-producer documents structure testing for UI (element existence, properties)
- [ ] tdd-red-producer documents structure testing for CLI (command existence, argument parsing)
- [ ] tdd-red-producer documents structure testing for API (endpoint existence, request/response shape)
- [ ] Structure tests fail before implementation, pass after (normal TDD red/green)

**Traces to:** ISSUE-007

---

## REQ-3: Behavior Testing (Interaction Produces Expected Result)

Tests must validate the full behavior chain: interaction triggers event, handler runs, state changes, output updates.

### Context

Behavior testing validates the "what happens when" question:
- UI: Click button -> handler fires -> state changes -> UI updates
- CLI: Run command -> processing occurs -> output appears
- API: Send request -> processing occurs -> response returned

The CLAUDE.md lesson captures this: "Test behavior, not just presence" and "Wire events end-to-end before marking done".

### Acceptance Criteria

- [ ] tdd-red-producer documents behavior testing for UI (full event chain)
- [ ] tdd-red-producer documents behavior testing for CLI (input -> processing -> output)
- [ ] tdd-red-producer documents behavior testing for API (request -> processing -> response)
- [ ] tdd-green-producer requires behavior tests to pass, not just structure tests
- [ ] tdd-qa validates behavior chain is tested, not just element presence

**Traces to:** ISSUE-007

---

## REQ-4: Property-Based Approach for Interaction Testing

Express and verify properties that should hold across interactions, not just example-based tests.

### Context

User gave example: "You should be able to quickly add a note from any screen"

This decomposes into verifiable properties:
1. **Structural property**: Every screen has an add-note affordance
2. **Behavioral property**: Interacting with affordance launches the add-note flow
3. **Completion property**: Walking through flow creates note and returns to origin

Properties can be verified systematically across all cases, not just hand-picked examples.

### Acceptance Criteria

- [ ] tdd-red-producer documents property-based testing approach for interactions
- [ ] Properties expressed for structure (invariant holds across all screens/commands/endpoints)
- [ ] Properties expressed for behavior (invariant holds across all interaction paths)
- [ ] Randomized/exhaustive property exploration encouraged where applicable

**Traces to:** ISSUE-007

---

## REQ-5: Visual Verification Evidence in TDD Cycle

TDD cycle for visual changes must include evidence that the visual output is correct.

### Context

DOM snapshots or structural tests are necessary but not sufficient. CLAUDE.md lesson: "UI testing verifies visual correctness, not just DOM existence."

Visual verification requires:
- Screenshot capture at key states
- Comparison against expected appearance (manual review or automated SSIM diff)
- Evidence preserved for audit trail

### Acceptance Criteria

- [ ] tdd-green-producer prompts for visual verification when task involves UI output
- [ ] tdd-qa requires visual evidence (screenshot or manual confirmation) for UI tasks
- [ ] Visual evidence path documented in yield payload or commit message
- [ ] SSIM-based regression detection available via `projctl screenshot diff`

**Traces to:** ISSUE-007, ISSUE-014

---

## REQ-6: Screenshot Capture Tooling

Provide mechanism for capturing visual output for verification.

### Context

ISSUE-014 notes that `projctl screenshot capture` doesn't exist, only `projctl screenshot diff`. Options:
1. Implement `projctl screenshot capture --url <url> --output <path>`
2. Document Chrome DevTools MCP as the capture mechanism
3. Document manual screenshot workflow

### Acceptance Criteria

- [ ] Either `projctl screenshot capture` implemented OR documented alternative exists
- [ ] Capture mechanism works for web UI (browser-based)
- [ ] Capture mechanism documented for CLI output (terminal snapshot)
- [ ] Integration with TDD workflow documented (when/how to capture)

**Traces to:** ISSUE-014

---

## REQ-7: Task Marking for Visual Verification

Tasks requiring visual verification should be identifiable.

### Context

ISSUE-007 AC includes: "Add `ui` flag or marker to tasks requiring visual verification"

This enables:
- tdd-green-producer to prompt for visual check when marker present
- tdd-qa to fail if UI task lacks visual evidence
- Breakdown-producer to flag visual tasks during task creation

### Acceptance Criteria

- [ ] Tasks can be marked as requiring visual verification (tag, flag, or convention)
- [ ] breakdown-producer applies marker to tasks affecting user-visible output
- [ ] tdd-green-producer detects marker and prompts for visual verification
- [ ] tdd-qa detects marker and requires visual evidence before approval

**Traces to:** ISSUE-007

---

## REQ-8: Skill Updates for Visual Verification

Update relevant skills to incorporate visual verification requirements.

### Context

ISSUE-007 AC specifies: "Document visual verification requirements in TDD skill docs"

Skills requiring updates:
- tdd-red-producer: Include visual acceptance criteria for UI changes
- tdd-green-producer: Verify visual output matches design
- tdd-qa: Require visual verification evidence for UI tasks
- breakdown-producer: Flag tasks requiring visual verification

### Acceptance Criteria

- [ ] tdd-red-producer SKILL.md updated with visual verification guidance
- [ ] tdd-green-producer SKILL.md updated with visual verification step
- [ ] tdd-qa SKILL.md updated to require visual evidence for marked tasks
- [ ] breakdown-producer SKILL.md updated to flag visual tasks
- [ ] CLAUDE.md lesson "UI validation is critical" reinforced in skill docs

**Traces to:** ISSUE-007

---

## REQ-9: CLI Output Testing

Tests for CLI changes must validate both structure (command exists) and output (produces expected result).

### Context

CLI is a user interface like any other. Testing should verify:
- Structure: Command parses, accepts expected flags, rejects invalid input
- Behavior: Command execution produces expected output format
- Visual: Output is readable/usable (not just technically correct)

For CLI, "visual verification" might mean capturing terminal output or using ASCII/ANSI-aware comparison.

### Acceptance Criteria

- [ ] tdd-red-producer documents CLI output testing (structure + behavior)
- [ ] CLI output capture mechanism documented (stdout/stderr capture)
- [ ] ANSI-aware comparison available for colored CLI output
- [ ] Example CLI test patterns included in skill docs

**Traces to:** ISSUE-007

---

## REQ-10: API Interaction Testing

Tests for API changes must validate both contract (endpoint shape) and behavior (request/response flow).

### Context

API is a programmatic interface but still has user (developer) experience. Testing should verify:
- Contract: Endpoint exists, accepts expected inputs, returns expected shape
- Behavior: Processing produces correct results
- Error handling: Invalid inputs produce appropriate errors

### Acceptance Criteria

- [ ] tdd-red-producer documents API contract testing
- [ ] tdd-red-producer documents API behavior testing
- [ ] Example API test patterns included in skill docs
- [ ] Integration with existing test tooling (http client mocks, etc.)

**Traces to:** ISSUE-007

---

## REQ-11: CLAUDE.md Standard Practice Update

Update CLAUDE.md to codify visual verification as standard TDD practice for all user-facing changes.

### Context

ISSUE-007 AC explicitly states: "CLAUDE.md lesson updated to make this standard practice"

This is distinct from updating skill docs (REQ-8). CLAUDE.md contains persistent lessons that guide all work, making visual verification a baseline expectation rather than a skill-specific detail.

### Acceptance Criteria

- [ ] CLAUDE.md lesson "UI validation is critical" expanded to cover all user interfaces (UI, CLI, API)
- [ ] Visual verification documented as non-optional for user-facing changes
- [ ] Lesson reinforces structure + behavior testing, not just "look at the screenshot"
- [ ] Property-based interaction testing added as standard practice

**Traces to:** ISSUE-007
