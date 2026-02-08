---
name: qa
description: Universal QA skill that validates any producer against its SKILL.md contract
context: inherit
model: haiku
skills: ownership-rules
user-invocable: true
role: qa
---

# Universal QA Skill

Validates any producer's output against the contract defined in its SKILL.md file.

**Contract Standard:** See [CONTRACT.md](../shared/CONTRACT.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Pattern | LOAD → VALIDATE → RETURN |
| Input | Producer SKILL.md path, artifact paths, iteration count |
| Output | Verdict: approved or improvement-request |
| Verdicts | `approved`, `improvement-request`, `escalate-phase`, `escalate-user`, `error` |

---

## Workflow

### 1. LOAD Phase

Load and parse all inputs needed for validation.

**Team mode:** Context arrives in spawn prompt from team lead. It includes:
- Producer SKILL.md path
- Artifact paths to validate
- Iteration number and max iterations

**Then for both modes:**

1. **Query memory for known failures:**
   ```bash
   projctl memory query "known failures in <artifact-type> validation"
   ```
   If memory is unavailable, proceed gracefully without blocking
   - Use to verify producer addressed known pitfalls (verification backstop)
   - Replace `<artifact-type>` with the artifact being validated (e.g., "requirements", "design", "architecture")

2. **Extract contract from producer SKILL.md:**
   - Search for `## Contract` section
   - Extract YAML code block immediately following the heading
   - Parse YAML to get `outputs`, `traces_to`, `checks`

3. **Handle missing contract (fallback to prose):**
   - If no `## Contract` section found, scan entire SKILL.md
   - Extract implicit checks from prose patterns:
     - Checklists (`- [ ]` items)
     - "Must" statements
     - Validation tables
   - Log warning: "No contract section found, using prose extraction"

4. **Read artifacts:**
   - Load each file from artifact paths
   - Handle missing artifacts → report as improvement-request

5. **Handle unreadable producer SKILL.md:**
   - If cannot read producer SKILL.md → report error (cannot proceed)

---

### 2. VALIDATE Phase

Execute each check from the contract against the artifacts:

1. **For each check in `contract.checks`:**
   - Evaluate check against artifact content
   - Record result: `passed: true/false`
   - If failed, capture specific details (which IDs, what's wrong)

2. **Common check patterns:**

   | Check Type | How to Validate |
   |------------|-----------------|
   | "Every entry has X-N ID" | Scan for `### X-NNN:` pattern |
   | "Traces to upstream" | Look for `**Traces to:**` fields |
   | "No orphan references" | Cross-reference all ID mentions |
   | "Content describes X" | Pattern match for prohibited terms |

3. **Classify failures by severity:**
   - `error` failures → QA fails
   - `warning` failures → QA passes with notes

4. **Compile results:**
   ```
   results = [
     { id: "CHECK-001", description: "...", passed: true },
     { id: "CHECK-002", description: "...", passed: false, details: ["DES-003 missing trace"] }
   ]
   ```

---

### 3. RETURN Phase

Report result based on validation findings.

#### Decision Tree

```
Has error-severity failures?
├─ YES → Can producer fix them?
│        ├─ YES → report `improvement-request`
│        └─ NO → Is issue in upstream artifact?
│                ├─ YES → report `escalate-phase`
│                └─ NO → report `escalate-user`
└─ NO → report `approved` (with warnings if any)
```

#### Iteration Tracking

Check iteration count before reporting `improvement-request`:

```
if iteration >= max_iterations (3):
    report `escalate-user` with:
        reason = "Max iterations reached"
        context = "Issues remain after 3 attempts"
```

#### Memory Persistence

When returning `improvement-request` or `escalate-phase` verdicts with **error-severity** findings, persist to memory:

```bash
projctl memory learn -m "QA failure in <artifact-type>: <check-id> - <description>" -p <issue-id>
```

- Only persist error-severity findings (not warnings or approvals)
- Replace `<artifact-type>` with artifact type (e.g., "requirements", "design")
- Replace `<check-id>` with the failing check ID (e.g., "CHECK-002")
- Replace `<description>` with brief failure description
- Replace `<issue-id>` with current project/issue ID for traceability

---

## Output Format

Full checklist display:

**Pass example:**
```
QA Results: PASSED

[x] CHECK-001: Every entry has DES-N identifier
[x] CHECK-002: Every DES-N traces to at least one REQ-N
[x] CHECK-003: No orphan ID references (warning: 1 unused ID found)
```

**Fail example:**
```
QA Results: FAILED

[x] CHECK-001: Every entry has DES-N identifier
[ ] CHECK-002: Every DES-N traces to at least one REQ-N
    - DES-003 has no traces
    - DES-007 has no traces
[x] CHECK-003: No orphan ID references
```

---

## Reporting Results

### Team Mode (preferred)

Send verdict to team lead via `SendMessage`:

**Approved:**
```
Verdict: approved

Reviewed: docs/design.md
Iteration: 1/3

[x] CHECK-001: Every entry has DES-N ID
[x] CHECK-002: Traces to REQ-N
[x] CHECK-003: No orphan references (note: 1 unused ID)
```

**Improvement request:**
```
Verdict: improvement-request

Reviewed: docs/design.md
Iteration: 2/3

Issues:
- CHECK-002: DES-003 has no traces
- CHECK-002: DES-007 has no traces
```

**Escalate to prior phase:**
```
Verdict: escalate-phase
From: design → To: pm

DES-005 describes error recovery but no REQ addresses error handling.
Proposed: add REQ for error recovery.
```

**Escalate to user (max iterations):**
```
Verdict: escalate-user

Issues remain after 3 QA iterations. Unresolved:
- CHECK-002: DES-003 has no traces
```

---

## Error Handling Summary

| Condition | Yield Type | Rationale |
|-----------|------------|-----------|
| Malformed producer yield | `improvement-request` | Producer can fix their yield |
| Missing artifact files | `improvement-request` | Producer can create missing files |
| No contract section | Continue with fallback | Graceful degradation |
| Unreadable producer SKILL.md | `error` | Cannot validate without contract |
| Check failures (error severity) | `improvement-request` | Producer can fix issues |
| Upstream artifact problem | `escalate-phase` | Not producer's fault |
| Unresolvable conflict | `escalate-user` | Need human decision |
| Max iterations reached | `escalate-user` | Prevent infinite loops |

---

## Contract Extraction Algorithm

Per ARCH-021, extract contract from producer SKILL.md:

```
1. Read producer SKILL.md content
2. Search for "## Contract" heading (case-insensitive)
3. If found:
   a. Find next fenced code block (```yaml ... ```)
   b. Parse YAML content
   c. Validate against CONTRACT.md schema
   d. Return contract object
4. If not found (fallback per ARCH-024):
   a. Scan for checklist patterns: "- [ ]" lines
   b. Scan for "must" statements in prose
   c. Convert to implicit checks with severity: warning
   d. Log warning about missing contract
   e. Return implicit contract
```

---

## Commit-QA Validation Contract

<!-- Traces: ARCH-040 -->

When validating commit-producer phases (`commit-red-qa`, `commit-green-qa`, `commit-refactor-qa`), apply these checks:

### Phase-Specific Checks

| Check ID | Description | Severity |
|----------|-------------|----------|
| CHECK-COMMIT-001 | Files staged match phase scope | error |
| CHECK-COMMIT-002 | No secrets in staged files | error |
| CHECK-COMMIT-003 | Commit message follows conventional format | error |
| CHECK-COMMIT-004 | Commit message describes change accurately | warning |
| CHECK-COMMIT-005 | No blanket lint suppressions added | error |
| CHECK-COMMIT-006 | Commit created successfully | error |

### Phase Scope Validation

| Phase | Expected Files |
|-------|----------------|
| commit-red-qa | Only test files (no implementation) |
| commit-green-qa | Test files + implementation (no refactoring-only changes) |
| commit-refactor-qa | Implementation files (behavior unchanged) |

### Validation Steps

1. **Read commit details:**
   ```bash
   git log -1 --pretty=format:"%H%n%s%n%b" HEAD
   git show --stat --name-only HEAD
   ```

2. **Validate staged files:**
   - Extract file list from `git show --name-only HEAD`
   - Check each file against phase scope rules
   - Flag violations as CHECK-COMMIT-001 failures

3. **Check for secrets:**
   - Scan file paths for: `.env`, `.env.*`, `credentials.json`, `secrets.yaml`
   - Scan file content for: `API_KEY=`, `SECRET=`, `PASSWORD=`, `-----BEGIN PRIVATE KEY-----`
   - Flag violations as CHECK-COMMIT-002 failures

4. **Validate commit message format:**
   - Pattern: `^(feat|fix|test|refactor|docs|chore)(\([^)]+\))?: .+$`
   - Check for `AI-Used: [claude]` trailer (NOT `Co-Authored-By`)
   - Flag violations as CHECK-COMMIT-003 failures

5. **Validate message accuracy:**
   - Compare commit description to actual changes
   - Check type matches phase (test for red, feat for green, refactor for refactor)
   - Flag mismatches as CHECK-COMMIT-004 warnings

6. **Check for blanket suppressions:**
   - Scan for patterns: `// nolint`, `/* eslint-disable */`, `[[linters.exclusions.rules]]`
   - Flag violations as CHECK-COMMIT-005 failures

7. **Verify commit exists:**
   - Check `git log -1` succeeds
   - Verify commit hash returned
   - Flag failures as CHECK-COMMIT-006 failures

### QA Responses on Failure

| Failure Type | QA Response |
|--------------|-------------|
| Wrong files staged | `improvement-request: unstage <files>, re-stage correct scope` |
| Secrets detected | `improvement-request: remove commit, unstage <files>, add to .gitignore` |
| Bad commit message | `improvement-request: amend commit message to: <suggestion>` |
| Commit doesn't exist | `error: commit creation failed: <details>` |
| Max iterations reached | `escalate-user: commit issues unresolved after 3 attempts` |

### Example Output

**Pass:**
```
QA Results: PASSED

[x] CHECK-COMMIT-001: Files staged match phase scope (commit-red: 2 test files)
[x] CHECK-COMMIT-002: No secrets in staged files
[x] CHECK-COMMIT-003: Commit message follows conventional format
[x] CHECK-COMMIT-004: Commit message describes change accurately
[x] CHECK-COMMIT-005: No blanket lint suppressions added
[x] CHECK-COMMIT-006: Commit created successfully (abc1234)
```

**Fail:**
```
QA Results: FAILED

[x] CHECK-COMMIT-001: Files staged match phase scope (commit-red)
[ ] CHECK-COMMIT-002: No secrets in staged files
    - .env file contains API_KEY
    - credentials.json staged
[x] CHECK-COMMIT-003: Commit message follows conventional format
[ ] CHECK-COMMIT-005: No blanket lint suppressions added
    - internal/foo/bar.go:10 has // nolint:errcheck
[x] CHECK-COMMIT-006: Commit created successfully

Verdict: improvement-request
- Remove .env and credentials.json from staging
- Add to .gitignore
- Remove nolint comment at internal/foo/bar.go:10
```

---

## Related Documents

- **CONTRACT.md**: Contract format specification
- **DES-003**: QA output format (full checklist)
- **DES-004**: QA context input schema
- **DES-005**: QA yield types
- **DES-006**: Malformed yield handling
- **DES-007**: Missing artifact handling
- **DES-009**: Unreadable SKILL.md handling
- **ARCH-021**: Contract extraction algorithm
- **ARCH-024**: Prose fallback behavior
- **ARCH-028**: Iteration tracking (max 3)
- **ARCH-040**: Commit-QA validation contract (ISSUE-92)
