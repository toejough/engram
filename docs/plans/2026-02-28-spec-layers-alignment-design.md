# Specification-Layers Alignment & Audit Design

## Problem

The specification-layers skill completed L1A (UC-1, UC-2, UC-3) through L5 for the engram project. All 58 tests pass across 9 packages. But the resulting code is a library only — no CLI binary, no LLM adapter implementations, no deployed hook scripts. The user cannot exercise the use cases.

The spec tree verified internal logic but never checked that the vertical slice was usable end-to-end. The gap formed because:
- L4 decomposed ARCH items into unit-testable behaviors only; wiring aspects (CLI binary, LLM adapters, hook deployment) described in ARCH-2/8/9/12 got no tests
- The DI pattern masked the gap — all tests pass with mocks, so "green" doesn't mean "runnable"
- The backtrack step checked node status but not end-to-end usability
- The verification type table (unit/integration/linter/llm) exists but the skill never checks that the mix is appropriate

These are hypothesized root causes. The actual fix should start with independent investigation.

## Solution: Three Layers of Defense

### Layer 1: Preventive — Fix the Skill

Update the specification-layers skill to structurally prevent alignment gaps. Targeted minimal edits, not a restructure. Expected areas:

- **L4 derivation completeness**: Strengthen "behavioral composition" to require coverage of ALL aspects of each ARCH item — including integration/wiring, not just pure-logic behaviors.
- **Verification type distribution**: At L4 REFACTOR, check that the mix of unit/integration/linter/llm tests is appropriate for what ARCH describes.
- **Backtrack usability gate**: When a vertical slice completes (L5 done, cursor rises past L1 group), verify the use cases are actually exercisable.
- **DI real-implementation tracking**: When ARCH specifies DI interfaces, explicitly track whether real implementations exist or are deferred to a later group.
- **Periodic vertical restatement**: At session starts, layer transitions, and periodically within L5, restate: which UC group, what the vertical slice is for, what's completed, what's left for end-user usability. Prevents tunnel vision.

### Layer 2: Inline Detection — Self-Checking Traversal

Add checks to the normal traversal that detect gaps even if preventive measures miss something:

- **At REFACTOR (every layer)**: Explicit bidirectional coverage check — "for each parent item, which child items cover it? Are any aspects uncovered?"
- **At backtrack from L5**: Usability gate — "walk the UC group's use cases and verify: can a user exercise them end-to-end?"
- **At session resume**: State.toml drift check — do referenced files exist? Do "complete" nodes actually have corresponding code? Are there code artifacts not reflected in the spec tree?
- **State.toml node status vs. reality**: A node marked `complete` whose ARCH items lack implementations or whose test items lack test files gets flagged.
- **Detection output**: Structured finding (what's misaligned, which spec items affected, suggested remediation) + user choice (fix now or defer).

### Layer 3: Companion Audit Skill

A standalone specification-layers-specific audit skill for use outside normal traversal.

**Four use cases:**
- **UC-A: Safety check** — Project followed the skill, believes it's good. Verify state.toml matches docs, docs match code, coverage complete, no orphaned artifacts.
- **UC-B: Stale catch-up** — Project had non-conformant changes (hotfixes, direct edits). Detect code that doesn't trace to specs, specs whose code drifted, stale state.toml nodes.
- **UC-C: Fresh adoption** — Existing repo that didn't use specification-layers. Reverse-engineer implicit layers, identify gaps, bootstrap state.toml, produce adoption roadmap.
- **UC-D: Version migration** — Project used an older skill version. Compare current expectations against project state, identify new checks that would flag issues.

**Core principle: Bottom-up audit, rebuild upward.** Code is ground truth — it's what's real, it's organized in digestible units, and it's the most expensive to rebuild. The audit starts from code and reconciles upward through tests → ARCH → REQ+DES → UC. Rebuilding upward risks awkward documentation that needs rework later. Rebuilding downward risks specifying equivalent behavior in a way that unnecessarily requires significant implementation rebuild.

**Audit mechanics:**
- Cross-reference matrix: spec item → doc location → code location → test location
- Findings with severity (critical/warning/info) and suggested actions
- Interactive: user confirms findings and chooses remediation path

## Deliverable

A single phased prompt to take to the specification-layers project. Four phases with user checkpoints:

1. **Root-cause analysis**: Read full skill, identify ALL structural weaknesses using engram as evidence. Categorize: omission, ambiguity, missing mechanism. Checkpoint: numbered findings.
2. **Preventive fixes**: Minimal targeted skill edits per finding. Checkpoint: proposed diffs.
3. **Inline detection**: Add self-checking to traversal. Checkpoint: proposed additions.
4. **Companion audit skill**: Design and build. Bottom-up, code-first. Checkpoint: review skill.
