# QA vs Producer Gap Analysis

This document compares each QA skill's checklist/validation criteria against the corresponding producer SKILL.md files to identify gaps where QA checks for things that producers do not document they will produce.

---

## pm-qa vs pm-interview-producer, pm-infer-producer

### Covered Checks
- Each requirement has a unique REQ-N identifier (both producers document REQ-N format)
- Requirements trace to source (producers include `**Traces to:**` links)
- Priority/scope indicated (interview-producer documents Priority P0/P1/P2)

### Gaps (in QA but not producer)
- **Acceptance criteria are specific and measurable**: QA checks for testable AC, but producers don't explicitly document that AC must be measurable
- **No ambiguous language ("should", "may", "might")**: QA checks for this, not in producer contracts
- **No conflicting requirements**: QA checks for conflicts, producers mention "Resolve conflicts" in SYNTHESIZE but don't make it explicit in output format
- **Edge cases identified where applicable**: QA checks this, producers don't document edge case sections
- **Dependencies between requirements documented**: QA checks this, not in producer output format

### Decision Required
- **Measurable AC**: ADD to producer contract - this is core to testability
- **No ambiguous language**: ADD to producer contract - include in PRODUCE phase rules
- **Conflict detection**: ADD to producer contract - document in SYNTHESIZE that conflicts must be resolved
- **Edge cases**: ADD to producer contract - add to requirements format
- **Dependencies**: ADD to producer contract - add optional Dependencies field to REQ format

---

## design-qa vs design-interview-producer, design-infer-producer

### Covered Checks
- All entries use DES-NNN format (both producers document DES-N format)
- Every DES-N traces to at least one REQ-N (producers include `**Traces to:**`)
- Content describes visual/interaction, not implementation (interview-producer has "No implementation" rule)

### Gaps (in QA but not producer)
- **All user-facing REQ-N have corresponding DES-N (Coverage)**: QA checks coverage, producers don't document a coverage validation step
- **No conflicting design decisions (Consistency)**: QA checks for conflicts, not explicit in producer contracts
- **All screens/flows addressed (Completeness)**: QA checks completeness, producers don't have explicit completeness criteria

### Decision Required
- **Coverage check**: ADD to producer contract - require producers to verify all user-facing REQs are addressed
- **Consistency check**: ADD to producer contract - add to SYNTHESIZE phase as explicit step
- **Completeness**: ADD to producer contract - document as exit criterion for PRODUCE phase

---

## arch-qa vs arch-interview-producer, arch-infer-producer

### Covered Checks
- ID follows ARCH-N format (both producers document this)
- Traces to REQ-N and/or DES-N IDs (both producers include `**Traces to:**`)
- Rationale provided and justified (interview-producer documents rationale in ARCH format)
- Alternatives considered documented (interview-producer documents alternatives in ARCH format)

### Gaps (in QA but not producer)
- **No conflicts with requirements or design**: QA checks for upstream conflicts, not explicit in producer contracts
- **All technical implications from requirements addressed (Completeness)**: QA checks this, producers don't have explicit completeness criteria
- **All technology decisions from design covered**: QA checks design coverage, producers don't document this
- **No orphan references (mentions IDs that don't exist)**: QA checks for invalid references, not in producer contracts

### Decision Required
- **Conflict detection**: ADD to producer contract - add to SYNTHESIZE phase
- **Completeness criteria**: ADD to producer contract - document what "complete" means for architecture
- **Design coverage**: ADD to producer contract - ensure all design tech implications are addressed
- **Reference validation**: ADD to producer contract - verify all referenced IDs exist before PRODUCE

---

## breakdown-qa vs breakdown-producer

### Covered Checks
- All tasks have unique TASK-N IDs (producer documents TASK-N format)
- Sequential numbering (no gaps) (producer documents sequential IDs)
- Dependencies reference valid TASK-N IDs (producer documents explicit dependencies)
- No prose dependencies (producer has "Explicit TASK-N references only" rule)
- Each task has `**Traces to:**` field (producer documents traces)
- `projctl trace validate` passes (QA documents this, but producer doesn't mention running it)

### Gaps (in QA but not producer)
- **All ARCH-N IDs have at least one implementing TASK**: QA checks architecture coverage, producer doesn't document this validation
- **No orphan tasks (all trace to ARCH/DES/REQ)**: QA checks this, producer doesn't explicitly require validation
- **Appropriate granularity (not too large/small)**: QA checks this, producer doesn't define granularity criteria
- **Each task has testable criteria**: QA checks criteria are measurable, producer doesn't explicitly require this
- **Criteria are measurable, not vague**: Same as above

### Decision Required
- **Architecture coverage**: ADD to producer contract - require verification all ARCH-N are covered
- **Orphan validation**: ADD to producer contract - add to PRODUCE phase as validation step
- **Granularity guidance**: ADD to producer contract - provide sizing heuristics (producer has "Sizing Priority" but not validation)
- **Testable criteria**: ADD to producer contract - document that AC must be measurable

---

## tdd-qa vs tdd-producer

### Covered Checks
- Tests written before implementation (tdd-producer orchestrates RED -> GREEN -> REFACTOR)
- All tests pass after implementation (tdd-producer documents this)
- Refactoring preserved behavior (tdd-producer documents this)

### Gaps (in QA but not producer)
- **Tests failed initially for correct reasons**: tdd-qa checks failure reasons, tdd-producer doesn't document verifying failure reasons
- **Minimal implementation (GREEN minimal)**: tdd-qa checks this, tdd-producer defers to tdd-green but doesn't document validation
- **Linter issues addressed during REFACTOR**: tdd-qa checks this, tdd-producer doesn't explicitly document linter as exit criterion
- **Each AC has corresponding test(s)**: tdd-qa checks AC coverage, tdd-producer doesn't document this validation
- **All AC items are `[x]` (none remain `[ ]`)**: tdd-qa checks for unchecked AC, tdd-producer doesn't document AC completion tracking
- **No deferral language in producer artifacts**: tdd-qa checks for deferral, tdd-producer doesn't document this rule
- **Visual evidence for `[visual]` tasks**: tdd-qa requires visual verification, tdd-producer doesn't document this (defers to tdd-green)

### Decision Required
- **Failure reason verification**: ADD to tdd-producer contract - document that RED phase must verify correct failure reasons
- **Linter as exit criterion**: ADD to tdd-producer contract - document linter clean as REFACTOR exit criterion
- **AC coverage tracking**: ADD to tdd-producer contract - track AC to test mapping
- **AC completion tracking**: ADD to tdd-producer contract - verify all AC are checked before complete
- **No deferral rule**: ADD to tdd-producer contract - document prohibition on silent deferrals
- **Visual evidence**: Already in tdd-green-producer - no gap, tdd-producer correctly defers

---

## tdd-red-qa vs tdd-red-producer, tdd-red-infer-producer

### Covered Checks
- Each acceptance criterion has at least one corresponding test (producers document AC-to-test mapping)
- Tests fail for correct reasons (producers document "Tests must fail" and failure verification)
- Tests describe expected behavior clearly (producers document behavior testing)
- Property tests used for invariants and edge cases (producer documents property-based testing)
- Tests are blackbox (test public API, not internals) (producer documents blackbox testing)

### Gaps (in QA but not producer)
- **No compilation/import errors**: QA explicitly checks tests don't fail due to syntax errors, producers assume this but don't document
- **No implementation code beyond minimal stubs**: QA checks for premature implementation, producers don't explicitly prohibit this

### Decision Required
- **Compilation error check**: ADD to producer contract - document that tests must compile cleanly before submitting
- **No implementation rule**: ADD to producer contract - explicit prohibition on implementation code in RED phase

---

## tdd-green-qa vs tdd-green-producer

### Covered Checks
- All new tests from red phase pass (producer documents "All targeted tests must pass")
- All existing tests still pass (producer documents regression checking)
- Implementation is minimal (producer has "MINIMAL code only" rule)
- Implementation follows architecture patterns (producer has "Follow arch patterns" rule)
- No new tests added (producer has "NO new tests" rule)
- Build succeeds with no errors (implied by "All tests must pass")
- Visual verification for `[visual]` tasks (producer documents visual verification section)

### Gaps (in QA but not producer)
None identified - tdd-green-producer is well-aligned with tdd-green-qa.

### Decision Required
No changes needed.

---

## tdd-refactor-qa vs tdd-refactor-producer

### Covered Checks
- All tests still pass after refactoring (producer documents "Tests must stay green")
- Linter issues reduced or eliminated (producer documents linter priority and fixing)
- No new linter issues introduced (implicit in linter fixing approach)
- Behavior unchanged (producer documents "No behavior changes")
- No blanket lint suppressions added (producer has "No blanket lint overrides" rule)

### Gaps (in QA but not producer)
- **Code readability improved**: QA checks readability, producer mentions clarity but doesn't make it explicit criterion

### Decision Required
- **Readability criterion**: ADD to producer contract - add explicit readability improvement as goal (not blocker, but documented goal)

---

## doc-qa vs doc-producer

### Covered Checks
- All public APIs documented (producer documents API docs generation)
- Installation and quick start present (producer documents README sections)
- Traces to REQ-N, DES-N, ARCH-N (producer documents traceability comments)

### Gaps (in QA but not producer)
- **Code examples compile/run (Accuracy)**: QA checks example validity, producer doesn't document example validation
- **API signatures match implementation**: QA checks signature accuracy, producer doesn't document validation step
- **Version numbers current**: QA checks versions, producer doesn't mention version tracking
- **No orphan traces (referencing non-existent IDs)**: QA checks for invalid traces, producer documents re-pointing test traces but doesn't validate
- **User guides cover key workflows**: QA checks user guide coverage, producer lists "User guides: Tutorials, recipes, FAQs" but no coverage criteria

### Decision Required
- **Example validation**: ADD to producer contract - document that examples must be verified
- **Signature accuracy**: ADD to producer contract - document validation against implementation
- **Version tracking**: ADD to producer contract - include version verification step
- **Trace validation**: Already mentions `projctl trace validate` - ensure producer runs it
- **User guide coverage**: ADD to producer contract - define what "key workflows" must be covered

---

## context-qa vs context-explorer

### Covered Checks
- All requested queries have results (context-explorer documents query execution and result aggregation)
- Individual query failures do not block other queries (context-explorer documents error handling)
- Results include success/failure status (context-explorer documents result structure)

### Gaps (in QA but not producer)
- **Results are relevant to the requesting producer's task**: QA checks relevance, context-explorer doesn't document relevance validation
- **No contradictions detected between multiple sources**: QA checks for contradictions, context-explorer doesn't detect contradictions
- **No stale or outdated information**: QA checks staleness, context-explorer doesn't document freshness checking
- **File contents exist (not "file not found" errors)**: Context-explorer documents error handling but QA wants explicit validation
- **Semantic queries answered the actual question asked**: QA checks semantic answer quality, context-explorer doesn't validate answer quality
- **Memory results are from relevant projects/sessions**: QA checks memory relevance, context-explorer doesn't filter by project

### Decision Required
- **Relevance validation**: DROP (reason: context-explorer is a data fetcher, relevance is context-qa's job)
- **Contradiction detection**: DROP (reason: context-explorer is a data fetcher, contradiction detection is context-qa's job)
- **Staleness detection**: DROP (reason: context-explorer is a data fetcher, staleness checking is context-qa's job)
- **Error reporting**: Already covered - context-explorer reports failures per query
- **Answer quality**: DROP (reason: semantic query quality is inherently variable, QA's job to assess)
- **Memory relevance**: DROP (reason: memory search returns what matches, filtering is QA's job)

Note: context-qa and context-explorer have appropriate separation of concerns. Explorer fetches data, QA validates quality. No changes needed.

---

## alignment-qa vs alignment-producer

### Covered Checks
- All expected artifact files were analyzed (producer documents artifact gathering)
- ID extraction used correct patterns (producer documents ID extraction)
- Orphan IDs detection (producer documents orphan detection)
- Unlinked IDs detection (producer documents unlinked detection)
- Chain direction is correct (producer documents chain direction)
- Suggested fixes are actionable and specific (producer documents issue reporting)

### Gaps (in QA but not producer)
- **Orphan IDs are truly undefined (not just in a different file)**: QA validates producer's orphan detection accuracy, producer doesn't document cross-file validation
- **Coverage metrics are mathematically accurate**: QA validates metrics, producer documents metrics but doesn't document validation
- **Domain boundary violations are correctly identified**: QA checks domain validation, producer mentions it but doesn't detail criteria

### Decision Required
- **Cross-file validation**: ADD to producer contract - document that all artifact files must be checked before flagging orphans
- **Metric validation**: DROP (reason: QA's job is to verify producer's math, not producer's job to self-validate)
- **Domain boundary criteria**: ADD to producer contract - expand domain boundary validation section with specific criteria

---

## retro-qa vs retro-producer

### Covered Checks
- Project summary is accurate (producer documents project summary section)
- What went well section has specific examples (producer documents successes with examples)
- What could improve section identifies real challenges (producer documents challenges section)
- Recommendations are actionable (producer requires "Actionable: Specific change to implement")
- Recommendations are prioritized (producer documents High/Medium/Low priority)
- Recommendations include rationale (producer documents rationale field)
- Open questions section (producer documents open questions)

### Gaps (in QA but not producer)
- **Metrics/data support observations where available**: QA checks for data support, producer mentions "Include metrics where available" but not as requirement
- **No critical successes or challenges omitted**: QA checks completeness, producer doesn't define what "complete" means

### Decision Required
- **Metrics support**: ADD to producer contract - make metrics inclusion more explicit, not just "where available"
- **Completeness criteria**: ADD to producer contract - define what successes/challenges must be included (e.g., all phases reviewed)

---

## summary-qa vs summary-producer

### Covered Checks
- Executive overview present (producer documents executive overview)
- All major decisions documented (producer documents key decisions extraction)
- Outcomes and deliverables (producer documents outcomes section)
- Lessons learned section included (producer documents lessons learned)
- Traces to REQ-N, DES-N, ARCH-N, TASK-N (producer documents traceability)

### Gaps (in QA but not producer)
- **Decision descriptions match actual choices made (Accuracy)**: QA validates accuracy, producer doesn't document verification step
- **Outcomes reflect actual implementation results**: QA validates outcome accuracy, producer doesn't document verification
- **Metrics are verifiable (coverage %, performance numbers)**: QA checks metric verifiability, producer doesn't document metric validation
- **Timeline and milestones are accurate**: QA checks timeline accuracy, producer doesn't mention timeline validation
- **No contradictions with upstream artifacts**: QA checks for contradictions, producer doesn't document consistency check
- **Key trade-offs explained**: QA checks for trade-off documentation, producer doesn't explicitly require this
- **Known limitations documented**: QA checks for limitations, producer lists under "Outcomes" but not explicit

### Decision Required
- **Accuracy verification**: ADD to producer contract - document that decisions/outcomes must be verified against artifacts
- **Metric validation**: ADD to producer contract - require evidence for claimed metrics
- **Timeline accuracy**: ADD to producer contract - include timeline verification step
- **Contradiction check**: ADD to producer contract - add consistency validation with upstream artifacts
- **Trade-offs**: ADD to producer contract - explicitly require trade-off documentation
- **Known limitations**: Already in producer under Outcomes - just make more explicit

---

## Summary of Required Changes

### High Priority (Core Contract Gaps)

| QA Skill | Gap | Action |
|----------|-----|--------|
| pm-qa | Measurable AC, No ambiguous language | Add to pm-*-producer contracts |
| design-qa | Coverage validation | Add to design-*-producer contracts |
| arch-qa | Completeness criteria | Add to arch-*-producer contracts |
| breakdown-qa | Architecture coverage, Testable criteria | Add to breakdown-producer contract |
| tdd-qa | AC completion tracking, No deferral rule | Add to tdd-producer contract |
| tdd-red-qa | No implementation in RED | Add to tdd-red-producer contract |
| doc-qa | Example validation, Signature accuracy | Add to doc-producer contract |
| summary-qa | Accuracy verification | Add to summary-producer contract |

### Medium Priority (Quality Improvements)

| QA Skill | Gap | Action |
|----------|-----|--------|
| pm-qa | Edge cases, Dependencies | Add optional fields to REQ format |
| arch-qa | Reference validation | Add validation step to producers |
| breakdown-qa | Granularity guidance | Add sizing validation criteria |
| tdd-refactor-qa | Readability criterion | Document as explicit goal |
| alignment-qa | Domain boundary criteria | Expand criteria documentation |
| retro-qa | Completeness criteria | Define what must be included |

### No Changes Needed

| QA Skill | Reason |
|----------|--------|
| tdd-green-qa | Well-aligned with producer |
| context-qa | Appropriate separation of concerns |
