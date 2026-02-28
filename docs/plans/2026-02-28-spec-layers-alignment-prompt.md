# Specification-Layers: Alignment & Self-Healing

## Context

The specification-layers skill has a structural gap that allows a project to complete a full vertical slice (L1→L5, all tests green) while the resulting code is unusable end-to-end.

**Evidence from a real project (engram):**
A memory system for LLM agents followed the skill through L1A (UC-1: Session Learning, UC-2: Hook-Time Surfacing, UC-3: Self-Correction). The tree descended through L2 (REQ+DES), L3 (ARCH with 12 decisions), L4 (58 BDD test specs), and L5 (all 58 tests green across 9 packages). The cursor backtracked, declaring L1A complete.

Result: 1,379 lines of tested library code that cannot be run. Missing:
- A CLI binary (ARCH-2 designed it, no test or code exists for it)
- Real implementations of LLM-calling interfaces (Enricher, Classifier, Evaluator — tested only via mocks)
- Deployed hook scripts (exist as Go string templates, not as actual files in .claude/hooks/)

The skill's DI architecture standard (ARCH-8) meant every pipeline was tested with mocks. All unit tests pass. But the wiring layer — the part that connects real SQLite, real LLM calls, and real CLI flags — was never specified, tested, or built. The user expected to start using memory writing when the vertical slice completed.

**Hypothesized root causes** (investigate independently — these may be incomplete or wrong):
1. L4 decomposed ARCH items into unit-testable behaviors only. Wiring aspects (binary entrypoint, adapter implementations, deployment artifacts) described in ARCH-2/8/9/12 got no tests.
2. The backtrack step checked node status but never asked "can the user actually exercise these use cases?"
3. The verification type table (unit/integration/linter/llm) exists but the skill never checks that the mix is appropriate.
4. Deep in L5, both the LLM agent and developer lost sight of the vertical slice's purpose — tunnel vision on passing individual tests.

## Your Task

Update the specification-layers skill with three layers of defense against alignment gaps, plus a companion audit skill. Work in four phases, each with a user checkpoint. Do NOT proceed to the next phase until the user approves.

---

## Phase 1: Root-Cause Analysis

Read the full specification-layers skill. Identify every structural point where alignment between specs, code, and usability could break — not just the symptom described above, but any weakness in the traversal, derivation, refactoring, or backtracking steps.

Use the engram case study as a test case: walk through what the skill should have done at each step where the gap formed. Specifically:
- When L4 was derived from L3, what check would have caught that ARCH-2 (CLI binary) had no tests covering the binary itself?
- When L5 completed and the cursor backtracked past L1A, what gate would have caught "all tests pass but you can't run anything"?
- During L5 implementation of 58 tests across multiple sessions, what mechanism would have kept the vertical slice's end-to-end purpose visible?

Categorize each finding:
- **Omission**: The skill doesn't instruct to do X at all
- **Ambiguity**: The skill says something that can be interpreted to skip X
- **Missing mechanism**: The skill has no structural concept for X

**Checkpoint:** Present findings as a numbered list with category, location in the skill, and description. Wait for user review before proceeding.

---

## Phase 2: Preventive Fixes

For each finding from Phase 1, propose a minimal targeted edit to the specification-layers skill. The goal is the smallest change that closes each gap — not a restructure of the skill.

Areas to consider (but let Phase 1 findings drive, not this list):

1. **Derivation completeness at L4**: The current instruction "decompose into behavioral variants using Beck's behavioral composition" guides toward unit-testable behaviors. It should explicitly require coverage of ALL aspects of each ARCH item — including integration/wiring boundaries, not just pure-logic pipelines. If an ARCH item describes a component that must be wired (binary entrypoint, adapter implementation, deployment artifact), that wiring needs test coverage.

2. **Verification type distribution**: At L4 REFACTOR, the skill should check that the mix of verification types (unit/integration/linter/llm) is appropriate for what ARCH describes. A project whose ARCH specifies integration boundaries (CLI binary, hook deployment, LLM adapter contracts) but whose test list is 100% unit tests should be flagged.

3. **Backtrack usability gate**: When a vertical slice completes (L5 done, cursor rises past an L1 group), add a gate: "Walk this UC group's use cases. Can a user exercise them end-to-end with what was built? If not, what's missing?" This catches the "all tests pass but it doesn't run" failure.

4. **DI real-implementation tracking**: When ARCH specifies DI interfaces, the skill should explicitly track: are real implementations part of this group's scope, or deferred to a later group? If deferred, the usability gate knows this slice is intentionally incomplete. If not deferred, missing real implementations are a gap.

5. **Periodic vertical restatement**: At session starts, layer transitions, and periodically during L5 (e.g., every N tests or every session), restate: which UC group we're in, what the vertical slice is building toward, what's completed, and what's still needed for the end user to actually use the result. This prevents tunnel vision when deep in implementation across multiple sessions.

After proposing fixes, re-walk the engram scenario through the updated skill as a regression test: would the updated skill have caught the gap? At which step? How?

**Checkpoint:** Present proposed skill edits (as diffs or before/after sections). Wait for user review before applying.

---

## Phase 3: Inline Detection

Add self-checking to the normal traversal so that even if preventive measures miss something, the skill detects gaps during execution:

1. **At REFACTOR (every layer)**: Add an explicit bidirectional coverage check. For each parent item, list which child items cover it and identify uncovered aspects. This is already implied by "bidirectional satisfaction" in the skill but should be made systematic — an enumerated check, not a vibes assessment.

2. **At backtrack from L5**: Add the usability gate from Phase 2 as an active check: "Before marking this L1 group complete, verify: can a user exercise the use cases end-to-end? List what they would do and what system components are involved. Flag any component that exists only as a tested interface with no real implementation, or as a template with no deployed artifact."

3. **At session resume**: When reading state.toml to resume, verify state matches reality:
   - Do files referenced in context_files exist?
   - For "complete" nodes: do corresponding code artifacts exist? Do tests still pass?
   - Are there code artifacts (packages, files, functions) not reflected in any spec item?

4. **State.toml drift check**: A node marked `complete` whose ARCH items lack corresponding implementations, or whose test items lack corresponding test files, gets flagged with a structured finding.

5. **Detection output format**: When a gap is detected, present it as:
   - What's misaligned (specific spec items, code locations, state.toml entries)
   - Severity (critical: can't use the feature; warning: spec/code drift; info: cosmetic)
   - Suggested remediation path
   - User choice: fix now, defer (with tracking), or dismiss

**Checkpoint:** Present proposed inline detection additions. Wait for user review before applying.

---

## Phase 4: Companion Audit Skill

Design and build a standalone companion skill (`specification-layers-audit` or similar) for use outside the normal traversal. This skill is specification-layers-specific — it knows about the diamond topology, state.toml format, L1-L5, dirty/unsatisfiable flags, and the full traversal model.

### Use Cases

**UC-A: Safety check** — Project followed specification-layers, believes it's in good shape. Audit verifies: state.toml matches docs, docs match code, coverage is complete across all layers, no orphaned artifacts. Output: pass/fail report with specific findings.

**UC-B: Stale catch-up** — Project used specification-layers but had non-conformant changes (hotfixes, direct code edits without spec updates, feature work outside the skill). Audit detects: code that doesn't trace to any spec item, spec items whose code has drifted, state.toml nodes with stale status. Output: remediation plan (which nodes to dirty, which specs to update, which state.toml entries to revise).

**UC-C: Fresh adoption** — Existing repo that didn't use specification-layers. Audit reverse-engineers what layers exist implicitly (README/docs as UC/REQ, architecture decision records as ARCH, test files as L4/L5, code as IMPL). Identifies gaps and misalignment. Output: proposed state.toml bootstrap, gap list, adoption roadmap.

**UC-D: Version migration** — Project used an older version of the skill. Audit compares current skill expectations against project state: new checks that would now flag issues, structural requirements that changed, new concepts (like the alignment checks from Phases 2-3) that the project doesn't yet satisfy. Output: migration plan with specific updates needed.

### Core Principle: Bottom-Up Audit, Rebuild Upward

Code is ground truth. It's what's real, it's organized in digestible units (packages, modules, files), and it's the most expensive layer to rebuild. The audit always starts from code and reconciles upward:

```
Code (ground truth) → Tests → ARCH → REQ+DES → UC → state.toml
```

Rebuilding upward risks documenting in awkward ways that need rework later — but that's cheaper than the alternative. Rebuilding downward risks specifying equivalent end-to-end behavior in a way that unnecessarily requires significant implementation rebuild.

When the audit finds a mismatch between code and specs, the default recommendation is to update specs to match code (rebuild up), not to rewrite code to match specs (rebuild down). The user can override this default, but the skill should present the upward path first.

### Audit Mechanics

- Build a cross-reference matrix: spec item ID → doc file location → code location → test location
- Use grep/glob to trace identifiers (UC-1, REQ-3, ARCH-2, T-15) through all project files
- Identify: uncovered spec items, untraced code, status mismatches, orphaned artifacts
- Present findings with severity (critical/warning/info) and suggested actions
- Work interactively: user confirms findings and chooses remediation path at each step

**Checkpoint:** Present the companion audit skill design and content. Wait for user review.
