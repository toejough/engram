# Architecture Documentation - ISSUE-92

**Issue:** ISSUE-92 - Per-phase QA in TDD loop
**Status:** Architecture Complete
**Date:** 2026-02-06

---

## ISSUE-92 Architecture Summary

| Decision | Choice |
|----------|--------|
| TDD sub-phase QA | tdd-red-qa, tdd-green-qa, tdd-refactor-qa phases |
| Commit pattern | commit-producer → commit-qa for each TDD phase |
| Transition enforcement | State machine prevents skipping QA |
| projctl step next | Returns QA actions between producer and commit |
| tdd-qa scope | Meta-check only (did the right steps happen?) |
| Commit staging rules | Phase-specific file scope (red=tests, green=tests+impl, refactor=impl) |
| Secret detection | Pre-commit validation for .env, credentials, API keys |
| Commit message format | Conventional commits with AI-Used trailer |

**Traceability Matrix:**

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-034 | ISSUE-92 |
| ARCH-035 | ISSUE-92 |
| ARCH-036 | ISSUE-92 |
| ARCH-037 | ISSUE-92 |
| ARCH-038 | ISSUE-92 |
| ARCH-039 | ISSUE-92 |
| ARCH-040 | ISSUE-92 |
| ARCH-041 | ISSUE-92 |

---

## Architecture Decisions

### ARCH-034: TDD Sub-Phase QA Phases

Add dedicated QA phases after each TDD sub-phase: `tdd-red-qa`, `tdd-green-qa`, `tdd-refactor-qa`.

**Purpose:** Validate correctness immediately after each TDD phase before proceeding to commit. Catches issues early when context is fresh.

**Responsibilities:**

| Phase | Validates |
|-------|-----------|
| tdd-red-qa | Test fails for right reason, test is well-formed, no false positives |
| tdd-green-qa | Test passes, implementation meets requirements, no over-engineering |
| tdd-refactor-qa | Code improved without behavior change, tests still pass, no regression |

**QA actions:**
- `approval: ready to commit` - QA passed, proceed to commit-phase
- `improvement-request: <specific issue>` - QA found problem, return to producer
- `error: <critical issue>` - Fundamental problem, escalate

**Transition flow:**
```
tdd-red → tdd-red-qa → commit-red → commit-red-qa →
tdd-green → tdd-green-qa → commit-green → commit-green-qa →
tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa →
task-audit
```

**Rationale:** Per-phase QA catches mistakes immediately. Waiting until final tdd-qa means lost context and harder debugging.

**Alternatives considered:**
- Single tdd-qa at end: Too late, context lost, harder to pinpoint which phase failed
- No QA automation: Relies on human discipline, inconsistent

**Traces to:** ISSUE-92

---

### ARCH-035: Commit-Phase Pair Loops

Add commit-phase pair for each TDD phase: `commit-{red,green,refactor}` and `commit-{red,green,refactor}-qa`.

**Pattern:** Every commit goes through validation before finalization.

```
commit-producer (commit-red) → commit-qa (commit-red-qa) → next phase
```

**Commit producer responsibilities:**
- Stage correct files for the phase
- Generate conventional commit message
- Create the commit
- Report files staged and message used

**Commit QA responsibilities:**
- Verify correct files staged (phase-specific scope)
- Check for secrets in staged files
- Validate commit message format
- Confirm commit succeeded

**QA actions on success:**
- `approval: commit finalized` - Proceed to next TDD phase or task-audit

**QA actions on failure:**
- `improvement-request: <fix>` - Return to commit-producer with correction
- `error: <critical issue>` - Commit failed, escalate

**Rationale:** Automated commit validation prevents common mistakes (wrong files, secrets, malformed messages) from reaching version control.

**Alternatives considered:**
- Manual commit review: Too slow, error-prone
- Pre-commit hooks only: Not orchestrated, inconsistent with QA pattern
- Single commit at end: Loses granular history, harder to bisect

**Traces to:** ISSUE-92

---

### ARCH-036: Transition Enforcement

State machine programmatically prevents skipping QA phases.

**Legal transitions:**

```go
transitions := map[Phase][]Phase{
    "tdd-red":            {"tdd-red-qa"},
    "tdd-red-qa":         {"commit-red"},
    "commit-red":         {"commit-red-qa"},
    "commit-red-qa":      {"tdd-green"},
    "tdd-green":          {"tdd-green-qa"},
    "tdd-green-qa":       {"commit-green"},
    "commit-green":       {"commit-green-qa"},
    "commit-green-qa":    {"tdd-refactor"},
    "tdd-refactor":       {"tdd-refactor-qa"},
    "tdd-refactor-qa":    {"commit-refactor"},
    "commit-refactor":    {"commit-refactor-qa"},
    "commit-refactor-qa": {"task-audit"},
}
```

**Improvement request transitions:** When QA returns `improvement-request`, transition back to the producer phase:

```go
improvementTransitions := map[Phase]Phase{
    "tdd-red-qa":         "tdd-red",
    "tdd-green-qa":       "tdd-green",
    "tdd-refactor-qa":    "tdd-refactor",
    "commit-red-qa":      "commit-red",
    "commit-green-qa":    "commit-green",
    "commit-refactor-qa": "commit-refactor",
}
```

**Validation:** `internal/state/tdd_qa_phases_test.go` tests all legal and illegal transitions.

**Rationale:** Programmatic enforcement prevents accidental skips. Human discipline is insufficient.

**Alternatives considered:**
- Documentation only: Easy to skip, no enforcement
- Tooling checks: Not integrated with orchestrator state
- CI validation: Too late, after commit already made

**Traces to:** ISSUE-92

---

### ARCH-037: projctl step next Integration

Extend `projctl step next` command to return QA action expectations for QA phases.

**Behavior:**

| Current Phase | `projctl step next` returns |
|---------------|---------------------------|
| tdd-red-qa | QA action for red phase validation |
| tdd-green-qa | QA action for green phase validation |
| tdd-refactor-qa | QA action for refactor phase validation |
| commit-red-qa | QA action for commit validation |
| commit-green-qa | QA action for commit validation |
| commit-refactor-qa | QA action for commit validation |

**Output format:**
```
Phase: tdd-red-qa
Action: Validate test fails for correct reason
Expected: approval | improvement-request | error
```

**Rationale:** Consistent with existing QA pattern (pm-qa, design-qa, arch-qa). Enables automation and prevents skipping.

**Alternatives considered:**
- Hardcoded in orchestrator: Less flexible, harder to test
- Manual phase tracking: Error-prone, inconsistent

**Traces to:** ISSUE-92

---

### ARCH-038: TDD-QA Scope Reduction

Reduce final `tdd-qa` phase to meta-check only: verify the TDD cycle completed correctly (red → green → refactor → commits).

**Old tdd-qa scope:**
- Verify tests pass ✓
- Verify implementation correct ✓
- Verify refactor improved code ✓
- Verify commits made ✓
- Verify code quality ✓

**New tdd-qa scope:**
- Verify tdd-red-qa approved ✓
- Verify commit-red-qa approved ✓
- Verify tdd-green-qa approved ✓
- Verify commit-green-qa approved ✓
- Verify tdd-refactor-qa approved ✓
- Verify commit-refactor-qa approved ✓

**QA action:**
- `approval: TDD cycle complete` - All sub-phases validated
- `error: missing validation` - One or more sub-phase QA not approved

**Rationale:** Per-phase QA already validated correctness. Final QA becomes orchestration check, not duplicate validation.

**Alternatives considered:**
- Keep full validation: Duplicates work already done in sub-phase QA
- Remove tdd-qa entirely: Lose cycle-level integrity check
- Merge into task-audit: Different concerns (TDD cycle vs task completion)

**Traces to:** ISSUE-92

---

### ARCH-039: Commit Producer Skill Requirements

Commit producer skills (`commit-red`, `commit-green`, `commit-refactor`) stage phase-specific files and create conventional commits.

**Phase-specific staging rules:**

| Phase | Files to stage |
|-------|----------------|
| commit-red | Test files only (`*_test.go`, `*.test.ts`, etc.) |
| commit-green | Test files + implementation files (new functions/types) |
| commit-refactor | Implementation files only (no test changes) |

**Secret detection patterns:**
- `.env` files
- `credentials.json`, `*.pem`, `*.key`
- Strings matching `password`, `secret`, `api_key`, `token` in new/modified lines
- High-entropy strings (potential keys)

**Commit message format:**
```
<type>(<scope>): <description>

<optional body>

AI-Used: [claude]
```

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

**Rationale:** Phase-specific staging prevents mixing concerns. Secret detection prevents credential leaks. Conventional commits enable automation and changelogs.

**Alternatives considered:**
- Single commit at end: Loses granular history
- Manual staging: Error-prone, inconsistent
- No secret detection: Risk of leaked credentials

**Traces to:** ISSUE-92

---

### ARCH-040: Commit-QA Validation Contract

Commit-QA phases validate 6 aspects of commit correctness:

**Validation checks:**

1. **Correct files staged:**
   - Red: Only test files
   - Green: Tests + implementation
   - Refactor: Only implementation

2. **No secrets in staged files:**
   - Check against secret patterns from ARCH-039
   - Report specific files and matched patterns

3. **Valid commit message format:**
   - Conventional commit structure
   - Non-empty description
   - AI-Used trailer present

4. **Commit succeeded:**
   - `git log -1` shows new commit
   - Commit hash returned

5. **Correct scope:**
   - Red: Test file count > 0
   - Green: Test + impl count > 0
   - Refactor: Impl count > 0, test count = 0

6. **No extraneous files:**
   - No unrelated files staged
   - No merge conflict markers
   - No temporary files

**QA actions on failure:**

| Failure Type | QA Response |
|--------------|-------------|
| Wrong files staged | `improvement-request: unstage <files>, stage <correct-files>` |
| Secrets detected | `improvement-request: remove <files> from staging, add to .gitignore` |
| Bad commit message | `improvement-request: amend commit message to: <suggestion>` |
| Commit failed | `error: commit creation failed: <details>` |

**Rationale:** Automated commit validation catches common mistakes before they reach CI. Secret detection prevents credential leaks.

**Alternatives considered:**
- Pre-commit hooks only: Not enforced in orchestrator, inconsistent
- Manual review: Too slow, error-prone

**Traces to:** ISSUE-92

---

### ARCH-041: State Machine Changes Summary

Summary of all state machine modifications for ISSUE-92.

**New phases added (10 total):**
- `tdd-red-qa`
- `tdd-green-qa`
- `tdd-refactor-qa`
- `commit-red`
- `commit-red-qa`
- `commit-green`
- `commit-green-qa`
- `commit-refactor`
- `commit-refactor-qa`
- (tdd-qa scope changed, not new)

**Transition updates:**

```
OLD: tdd-red → commit-red → tdd-green → commit-green → tdd-refactor → commit-refactor → tdd-qa

NEW: tdd-red → tdd-red-qa → commit-red → commit-red-qa →
     tdd-green → tdd-green-qa → commit-green → commit-green-qa →
     tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa →
     task-audit
```

**Files to modify:**
- `internal/state/transitions.go` - Add new phases to legal targets
- `internal/step/registry.go` - Add commit-phase entries (if using registry for commits)
- `internal/state/state.go` - No changes needed (generic transition logic)

**Tests:**
- `internal/state/tdd_qa_phases_test.go` - Already exists with full test coverage

**Rationale:** Clean separation of QA and commit responsibilities. Each phase has a single concern.

**Traces to:** ISSUE-92

---

## Context Sources

**Primary sources:**
1. Issue description (`docs/issues.md` ISSUE-92)
2. Test file (`internal/state/tdd_qa_phases_test.go`)
3. Phase registry (`internal/step/registry.go`)
4. Existing architecture (ARCH-019 through ARCH-030)

**Coverage metrics:**
- Total key questions: 10
- Questions answered from context: 9
- Coverage percentage: 90%
- Gap classification: Small
- Interview questions needed: 0
