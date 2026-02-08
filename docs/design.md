# ISSUE-53: Universal QA Skill Design

Design decisions for replacing 13 phase-specific QA skills with one universal QA skill using team messaging.

---

### DES-001: Contract YAML Format

Simple flat YAML structure optimized for LLM parsing (Sonnet producing, Haiku validating).

```yaml
contract:
  outputs:
    - path: "docs/design.md"
      id_format: "DES-N"

  traces_to:
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every entry has a DES-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every DES-N traces to at least one REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "No orphan ID references"
      severity: warning
```

**Design rationale:**
- Flat structure (no nested categories) for straightforward parsing
- `severity: error` means QA fails; `severity: warning` means QA passes with note
- `outputs` specifies what files the producer creates and their ID format
- `traces_to` specifies what upstream artifacts must be referenced
- `checks` is an ordered list of validation criteria

**Traces to:** REQ-006

---

### DES-002: Contract Section Placement

Contracts live in producer SKILL.md files as a fenced YAML code block under a `## Contract` heading.

```markdown
## Contract

```yaml
contract:
  outputs:
    - path: "docs/requirements.md"
      id_format: "REQ-N"
  traces_to:
    - "issue description"
  checks:
    - id: "CHECK-001"
      description: "Every requirement has REQ-N ID"
      severity: error
```
```

**Design rationale:**
- Single source of truth in producer's own documentation
- QA skill receives SKILL.md path, extracts contract section
- No separate contract files to maintain

**Traces to:** REQ-006, REQ-007

---

### DES-003: QA Output Format

Full checklist display for every QA run, regardless of pass/fail status.

**Pass example:**
```
QA Results: PASSED

[x] CHECK-001: Every entry has a DES-N identifier
[x] CHECK-002: Every DES-N traces to at least one REQ-N
[x] CHECK-003: No orphan ID references
```

**Fail example:**
```
QA Results: FAILED

[x] CHECK-001: Every entry has a DES-N identifier
[ ] CHECK-002: Every DES-N traces to at least one REQ-N
    - DES-003 has no traces
    - DES-007 has no traces
[x] CHECK-003: No orphan ID references
```

**Design rationale:**
- Always show full checklist so user sees what was validated
- Failed checks include specific details (which IDs, what's wrong)
- Warnings show as passed with note: `[x] CHECK-003: ... (warning: found 1 unused ID)`

**Traces to:** REQ-005

---

### DES-004: QA Context Input

Team lead provides QA teammate with context via spawn prompt:

```
Invoke the /qa skill to validate the producer's output.

Producer SKILL.md: skills/design-interview-producer/SKILL.md
Artifact paths: docs/design.md
Iteration: 1

Context:
The design-interview-producer completed and reported:
- Created DES-001 through DES-012
- Modified: docs/design.md
```

**Design rationale:**
- QA reads producer SKILL.md to extract contract
- QA reads artifact files directly using Read tool
- QA validates artifacts against contract checks

**Traces to:** REQ-005, REQ-010

---

### DES-005: QA Message Types

QA teammate sends one of four message patterns to team lead based on validation results:

| Condition | Message Pattern |
|-----------|-----------------|
| All checks pass | `approved` |
| Check failures that producer can fix | `improvement-request: <issues>` |
| Problem in upstream phase artifact | `escalate-phase: <reason>` |
| Cannot resolve without user | `escalate-user: <reason>` |

**approved message:**
```
approved

Reviewed artifact: docs/design.md
Checklist:
[x] CHECK-001: Every entry has DES-N ID
[x] CHECK-002: Traces to REQ-N
```

**improvement-request message:**
```
improvement-request: missing traces

Issues to fix:
- CHECK-002: DES-003 has no traces
- CHECK-002: DES-007 has no traces
```

**Traces to:** REQ-005

---

### DES-006: Error Handling - Invalid Producer Output

When producer's completion message is missing expected information:

- QA sends `improvement-request: incomplete producer output`
- Message includes what information is missing
- Team lead spawns new producer with feedback

**Example message:**
```
improvement-request: incomplete producer output

Missing information:
- No artifact paths provided in completion message
- No IDs reported for created items
```

**Traces to:** REQ-005

---

### DES-007: Error Handling - Missing Artifacts

When artifact files don't exist:

- QA sends `improvement-request: missing artifacts`
- Message includes missing file paths
- Team lead spawns new producer to create the files

**Example message:**
```
improvement-request: missing artifacts

Missing files:
- docs/design.md (file not found)
```

**Traces to:** REQ-005

---

### DES-008: Error Handling - Missing Contract

When producer SKILL.md has no Contract section:

- QA falls back to reading entire SKILL.md
- QA extracts implicit requirements from prose (best effort)
- QA logs warning that producer should add contract section
- Validation continues with extracted requirements

**Fallback behavior:**
1. Search SKILL.md for structured patterns (checklists, tables, "must" statements)
2. Convert found patterns to implicit checks
3. Validate artifact against implicit checks
4. Include warning in output: "Warning: No contract section found, using prose extraction"

**Traces to:** REQ-005, REQ-011

---

### DES-009: Error Handling - Unreadable SKILL.md

When producer SKILL.md cannot be read (file not found, permissions):

- QA sends `error: cannot read producer SKILL.md`
- Cannot validate without contract
- Team lead must resolve before QA can proceed

**Example message:**
```
error: cannot read producer SKILL.md

Details: File not found: skills/design-interview-producer/SKILL.md
This is not recoverable - team lead must fix the path.
```

**Traces to:** REQ-005

---

### DES-010: Escalation to Upstream Phase

When QA discovers problem in upstream artifact (not current producer's fault):

- QA sends `escalate-phase: <reason>`
- Includes proposed changes for upstream phase
- Team lead routes back to correct phase

**Example: Design QA finds missing requirement**
```
escalate-phase: gap in upstream requirements

From phase: design
To phase: pm
Reason: gap

Issue:
Design references capability not in requirements.
DES-005 describes error recovery but no REQ addresses error handling.

Proposed change:
Add REQ-012: Error Recovery
"System must provide clear error messages when validation fails"
```

**Traces to:** REQ-005

---

### DES-011: Escalation to User

When QA cannot resolve conflict or ambiguity:

- QA sends `escalate-user: <reason>`
- Presents question with options
- Team lead prompts user, sends answer back to QA

**Example: Conflicting requirements**
```
escalate-user: conflicting traces

Reason: Conflicting traces
Context: DES-003 traces to both REQ-002 and REQ-005, which contradict each other.

Question: Which requirement takes priority?
Options:
1. REQ-002 (offline-first)
2. REQ-005 (real-time sync)
3. Both with user toggle
```

**Traces to:** REQ-005

---

### DES-012: Iteration Limits

QA tracks producer-QA iterations to prevent infinite loops:

- Maximum 3 iterations per producer-QA pair
- After max iterations with issues remaining: send `escalate-user: max iterations reached`
- Iteration count tracked in team lead's PairState

**Example message on max iterations:**
```
escalate-user: max iterations reached

Iteration 3 of 3 reached with remaining issues:
- CHECK-002: DES-003 still has no traces
- CHECK-005: Missing design rationale

User decision needed: accept as-is, extend iterations, or modify requirements?
```

**Traces to:** REQ-005

---

### DES-013: Single QA Skill Invocation

User invokes QA with producer name:

```
/qa design-interview-producer
```

Team lead resolves this to:
1. Find producer SKILL.md at `skills/design-interview-producer/SKILL.md`
2. Find producer's most recent completion message for artifact paths
3. Spawn QA teammate with producer SKILL.md path and artifact paths

**Traces to:** REQ-005, REQ-010

---

## ISSUE-56: Inferred Specification Warning Design

Design decisions for how producers flag inferred specifications via AskUserQuestion and how the team lead presents them for user approval.

---

### DES-014: Inferred Message Format

Inferred specifications use AskUserQuestion with an `inferred = true` flag in the options. This distinguishes inferred items from regular interview questions.

**AskUserQuestion structure:**

The producer teammate uses AskUserQuestion to present inferred items:

```
AskUserQuestion with multiSelect: true
Question: "The following specifications were inferred. Accept or reject each:"
Options:
  1. REQ-X: Input validation for empty strings
     (Reasoning: Edge case - empty input could cause downstream errors, Source: best-practice)
  2. REQ-Y: Rate limiting on API calls
     (Reasoning: Implicit need - without rate limiting, external API costs could spike, Source: edge-case)
```

**Metadata fields:**
- `inferred = true`: Signals this is inference confirmation
- Each option includes: specification text, reasoning, source category
- `source` values: `best-practice`, `edge-case`, `implicit-need`, `professional-judgment`

**Traces to:** REQ-012

---

### DES-015: Team Lead Relay of Inferred Items

When a producer teammate sends inferred items via AskUserQuestion, the team lead may relay the question to the user or handle it directly based on context.

**User presentation format (via team lead relay):**
```
The producer inferred the following specifications that were not
explicitly requested. Please accept or reject each:

1. REQ-X: Input validation for empty strings
   Reasoning: Edge case - empty input could cause downstream errors
   Source: best-practice

2. REQ-Y: Rate limiting on API calls
   Reasoning: Implicit need - without rate limiting, external API costs could spike
   Source: edge-case

Select which items to accept (e.g., "1, 2" for both, "1" for first only):
```

**User response handling:**
- Selections captured via AskUserQuestion multiSelect
- Teammate receives user decisions and proceeds with only accepted + explicit items

**Traces to:** REQ-014

---

### DES-016: Producer Inference Detection Workflow

During the SYNTHESIZE phase, producers separate gathered information into two categories before producing artifacts:

1. **Explicit**: Directly traceable to user input, issue description, or gathered context
2. **Inferred**: Added by the producer based on professional judgment

**Workflow:**
1. Producer teammate completes GATHER phase (interview or context analysis)
2. During SYNTHESIZE, producer classifies each specification as explicit or inferred
3. If any inferred items exist, producer uses AskUserQuestion with `inferred = true` BEFORE producing the artifact
4. User responds with accepted items (via team lead relay or directly)
5. Producer receives user decisions
6. Producer produces artifact with only explicit + accepted items

**Traces to:** REQ-013, REQ-015

---

## Summary

| Decision | Choice |
|----------|--------|
| Contract format | Flat YAML, no versions |
| Contract location | `## Contract` section in producer SKILL.md |
| QA output | Full checklist always |
| Missing artifacts | `improvement-request: missing artifacts <list>` message |
| Missing contract | Prose fallback with warning |
| Unreadable SKILL.md | `error: cannot read producer SKILL.md` message |
| Upstream issues | `escalate-phase: <reason>` message with proposed changes |
| Unresolvable | `escalate-user: <reason>` message with options |
| Max iterations | 3, then escalate to user |
| Inferred spec format | AskUserQuestion with `inferred = true` metadata |
| Inferred presentation | Numbered list with reasoning, multiSelect for accept/reject |
| Inference detection | SYNTHESIZE phase classifies explicit vs inferred before producing |

---

## ISSUE-152: Semantic Memory Integration Design

Design decisions for user-facing aspects of semantic memory integration into the orchestration workflow. Architecture decisions (BERT tokenization, ONNX models, database schema) are in docs/architecture.md (ARCH-052 through ARCH-063).

---

### DES-017: Memory Query Visibility in Producer Output

Producers surface memory query results to users during interview/gather phases for transparency.

**User experience:**
When a producer queries memory, results appear in the conversation:

```
Gathering context for design decisions...

Memory query: "design patterns for authentication"
Found 3 relevant learnings:
- Success: Two-factor auth improved security posture (project: auth-v2)
- Challenge: SMS codes had delivery delays (project: mobile-login)
- Recommendation: Use TOTP over SMS for reliability (project: secure-app)
```

**Design rationale:**
- User sees what past learnings influence current decisions
- Transparency builds trust in memory-informed recommendations
- User can correct if retrieved context is irrelevant

**Traces to:** REQ-008, ARCH-055

---

### DES-018: Memory Query Failure Graceful Degradation

When memory queries fail (service unavailable, timeout), producers continue without blocking with visible notification.

**User experience:**
```
Gathering context for requirements...

Memory query: "prior requirements for payment processing"
⚠️ Memory service unavailable - continuing without historical context

Proceeding with interview...
```

**Design rationale:**
- Memory is enhancement, not dependency
- Failed queries don't block project progress
- User awareness prevents confusion about missing context

**Traces to:** REQ-008, ARCH-055

---

### DES-019: Session-End Summary Visibility

Session-end capture happens automatically but surfaces a summary for user awareness.

**User experience:**
At project completion, orchestrator shows:

```
Capturing session learnings...

Session summary created:
- 5 decisions indexed (technology choices, API design)
- 3 key challenges logged (integration complexity, test flakiness)
- 2 successes recorded (zero QA failures, ahead of schedule)

Summary saved to: ~/.claude/memory/sessions/2026-02-08-issue-152.md
```

**Design rationale:**
- User sees what gets remembered from this session
- Summary file path provided for manual review if needed
- Automatic capture without user action required

**Traces to:** REQ-007, ARCH-054

---

### DES-020: Promotion Candidate Notification

Retro-producer presents memory promotion candidates as recommendations, not automatic promotions.

**User experience:**
In retrospective report:

```
## Memory Promotion Candidates

The following memories have been retrieved 3+ times across 2+ projects
and may warrant promotion to CLAUDE.md for permanent knowledge:

1. "Always inject side-effectful dependencies for testability"
   - Retrieved: 5 times across 3 projects
   - Last used: 2026-02-05

2. "Use property-based testing (rapid) for tokenization/parsing logic"
   - Retrieved: 4 times across 2 projects
   - Last used: 2026-02-08

Would you like to promote any of these to CLAUDE.md?
```

**Design rationale:**
- Human judgment required for tier-1 knowledge promotion
- Statistics help user assess value
- Opt-in, not automatic (prevents noise in CLAUDE.md)

**Traces to:** REQ-013, ARCH-060

---

### DES-021: External Knowledge Source Attribution

When producers capture external best practices, source attribution is visible in memory content.

**User experience:**
Memory query result:

```
Memory query: "architecture patterns for API rate limiting"
Found 2 relevant learnings:

1. [Internal] Token bucket algorithm scales better than fixed window
   - Project: api-gateway, 2026-01-15

2. [External] Redis-backed rate limiting with sliding window (source: nginx.com/blog/rate-limiting)
   - Captured: 2026-02-01, Confidence: 0.7
```

**Design rationale:**
- User distinguishes internal experience from external research
- Source attribution enables follow-up (link to original article)
- Confidence score visible for transparency

**Traces to:** REQ-014, ARCH-061

---

### DES-022: Memory Conflict Warning Display

When learning new information that conflicts with existing memories, user sees warning with comparison.

**User experience:**
During `projctl memory learn`:

```
Learning stored: "Use pattern B for authentication"

⚠️  Potential conflict detected:
Existing memory (similarity 0.91): "Use pattern A for authentication"
  - Project: auth-v2, 2026-01-10

Both entries kept. Review ~/.claude/memory/ to resolve conflict.
```

**Design rationale:**
- User aware of contradictory guidance
- Both preserved (may be context-dependent)
- Manual resolution prevents automatic overwrites

**Traces to:** REQ-016, ARCH-062

---

### DES-023: Orchestrator Memory Read Timing

Orchestrator memory queries happen at startup before any phase begins, with results included in initial context message.

**User experience:**
When `/project` starts:

```
Initializing project for ISSUE-152...

Querying past learnings...
- "lessons from past projects": 4 results
- "common challenges in scoped projects": 2 results

Starting workflow with historical context.

[Orchestrator proceeds to first phase]
```

**Design rationale:**
- User sees memory retrieval happen once upfront
- No repeated memory queries each phase (efficiency)
- Context available to all spawned producers from start

**Traces to:** REQ-012, ARCH-059

---

### DES-024: QA Failure Pattern Feedback Loop Visibility

When QA finds known failure patterns that producer missed, this is surfaced as an escalated finding.

**User experience:**
QA message to team lead:

```
improvement-request: missed known failure pattern

Issues:
- CHECK-002: DES-003 has no traces (KNOWN PATTERN)
  Memory shows this failed QA in 2 prior projects:
  * project-alpha (2026-01-20): "design entries without traces"
  * dashboard-v2 (2026-01-28): "missing REQ traces in DES-005"

Producer GATHER should have surfaced and avoided this pattern.
```

**Design rationale:**
- Escalated when known patterns are missed (not just generic failures)
- Shows memory query happened but producer didn't act on it
- Helps refine producer memory usage over time

**Traces to:** REQ-010, ARCH-056

---

### DES-025: Memory Decay and Pruning User Notification

Decay and pruning operations surface summary statistics but don't require user interaction.

**User experience:**
At end of project:

```
Running memory hygiene...

Decay applied:
- 127 entries aged (confidence reduced 10%)
- 89 entries maintained (recently retrieved)

Prune applied:
- 3 entries removed (confidence < 0.1)
- 213 entries remaining

Memory database cleaned.
```

**Design rationale:**
- User aware memory quality is actively maintained
- Statistics show system is working (not silent)
- No action required (automatic maintenance)

**Traces to:** REQ-015, ARCH-062

---

### DES-026: Context Explorer Auto-Memory Enrichment Transparency

When context-explorer automatically adds memory queries, this is visible in query execution log.

**User experience:**
Context-explorer output:

```
Executing queries:
✓ File search: "authentication" (3 matches)
✓ Semantic search: "auth patterns" (2 matches)
✓ Memory query: "authentication" (auto-enriched, 4 matches)

Aggregating results...
```

**Design rationale:**
- User sees memory was consulted automatically
- "(auto-enriched)" tag distinguishes from explicit requests
- Transparency into context sources used

**Traces to:** REQ-008, ARCH-063

---

### DES-027: Producer Memory Read Phase Placement

All producer memory queries happen in GATHER phase before user interview/inference, preventing double-work.

**User workflow:**
1. Producer spawned by orchestrator
2. Producer reads memory for domain context (user sees query results)
3. Producer presents interview questions OR begins inference (informed by memory)
4. User responds / producer produces artifact

**Design rationale:**
- Memory informs questions (prevents asking about known decisions)
- User doesn't see redundant "researching" after answering questions
- Consistent placement across all 18 LLM-driven skills

**Traces to:** REQ-008, ARCH-055

---

### DES-028: Universal Yield Capture Invisibility

Decision extraction from producer yield TOMLs happens automatically without user-visible operations.

**User experience:**
User sees:
```
Producer completed: design-interview-producer
Spawning QA teammate...
```

User does NOT see:
```
Extracting decisions from result.toml...
Indexing 3 decisions to memory...
```

**Design rationale:**
- Yield capture is implementation detail
- No user action required or possible
- Reduces notification noise
- Failures logged but not surfaced (best-effort)

**Traces to:** REQ-009, ARCH-058

---

### DES-029: Retro Learning Persistence Visibility

Retro-producer shows which learnings are being captured to memory during produce phase.

**User experience:**
Retro output:

```
Producing retrospective.md...

Persisting learnings to semantic memory:
✓ Success: Zero QA failures across 12 tasks
✓ Challenge: Context compaction caused mid-session rework
✓ Recommendation: Add schema migration tests for DB changes

Retrospective complete.
```

**Design rationale:**
- User sees high-value learnings being indexed
- Transparency into what future projects will retrieve
- Confirmation that retro insights aren't siloed in one document

**Traces to:** REQ-011, ARCH-057

---

## ISSUE-152 Design Summary

| Decision | Choice |
|----------|--------|
| Memory query visibility | Show results during GATHER with source attribution |
| Query failure | Graceful degradation with warning, continue without blocking |
| Session-end | Automatic capture with visible summary statistics |
| Promotion | Human-approved based on retrieval stats in retro |
| External knowledge | Source attribution and confidence score visible |
| Conflicts | Warning with comparison, both entries kept |
| Orchestrator timing | Startup query before phases, results in initial context |
| QA failure loop | Escalate when known patterns missed |
| Decay/prune | Automatic with summary statistics, no user action |
| Auto-enrichment | Visible "(auto-enriched)" tag in query log |
| Producer read phase | GATHER phase before interview/inference |
| Yield capture | Invisible to user (automatic, best-effort) |
| Retro persistence | Visible confirmation of indexed learnings |

