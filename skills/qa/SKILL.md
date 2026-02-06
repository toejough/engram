---
name: qa
description: Universal QA skill that validates any producer against its SKILL.md contract
context: fork
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

1. **Extract contract from producer SKILL.md:**
   - Search for `## Contract` section
   - Extract YAML code block immediately following the heading
   - Parse YAML to get `outputs`, `traces_to`, `checks`

2. **Handle missing contract (fallback to prose):**
   - If no `## Contract` section found, scan entire SKILL.md
   - Extract implicit checks from prose patterns:
     - Checklists (`- [ ]` items)
     - "Must" statements
     - Validation tables
   - Log warning: "No contract section found, using prose extraction"

3. **Read artifacts:**
   - Load each file from artifact paths
   - Handle missing artifacts → report as improvement-request

4. **Handle unreadable producer SKILL.md:**
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
