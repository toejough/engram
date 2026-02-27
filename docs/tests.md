# Test Specifications

Test specifications for the Write Path (L3A: ARCH-1 through ARCH-9). Each test traces through ARCH to L2 items (REQ/DES).

Three test types per the specification-layers model:
- **Property-based (P):** Verify REQ invariants through ARCH
- **Example-based (E):** Verify DES scenarios through ARCH
- **Integration (I):** Verify component boundaries from ARCH

---

## ARCH-1: Memory Storage (SQLite + FTS5)

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-1 | P | CreatePopulatesAllMetadataFields | REQ-2 |
| T-2 | P | ConfidenceTierIsRequired | REQ-3 |
| T-3 | P | FindSimilarReturnsScoredCandidates | REQ-5 |
| T-4 | P | FindSimilarRespectsK | REQ-5 |
| T-5 | P | UpdatePreservesCreatedAt | REQ-2 |
| T-6 | E | EnrichmentIncrementsCount | DES-6 |
| T-7 | E | FindSimilarRanksKeywordMatchesHigher | DES-6 |
| T-8 | I | FTS5IndexSyncsWithTable | ARCH-1 |
| T-9 | I | ConcurrentReads | ARCH-1 |

## ARCH-3: Extraction Pipeline

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-10 | P | EmptyTranscriptProducesNoMemories | REQ-1 |
| T-11 | P | QualityGateRejectsVagueContent | REQ-2 |
| T-12 | P | EveryMemoryHasConfidenceTier | REQ-3 |
| T-13 | P | DedupSkipsMidSessionCorrections | REQ-18 |
| T-14 | P | ReconciliationEnrichesOnOverlap | REQ-5 |
| T-15 | P | ReconciliationCreatesOnNoOverlap | REQ-5 |
| T-16 | E | RealSessionScenario_ThreeLearningsOneDedup | DES-6 |
| T-17 | E | AllRejected_AuditLogRecordsReasons | DES-6 |
| T-18 | I | PipelineEndToEnd | ARCH-3 |

## ARCH-4: Correction Detection

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-19 | P | NoMatchReturnsEmpty | REQ-13 |
| T-20 | P | MatchTriggersReconciliation | REQ-13, REQ-14 |
| T-21 | P | AllFifteenPatternsMatch | REQ-13 |
| T-22 | P | MatchRecordsToSessionLog | REQ-18 |
| T-23 | E | EnrichedExistingMemory_SystemReminder | DES-3 |
| T-24 | E | CreatedNewMemory_SystemReminder | DES-3 |
| T-25 | E | FalsePositive_CapturedAnyway | DES-5 |
| T-26 | I | EndToEnd | ARCH-4 |

## ARCH-5: Reconciler (shared)

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-27 | P | NoExistingMemories_Creates | REQ-5, REQ-14 |
| T-28 | P | OverlapGateSaysYes_Enriches | REQ-5, REQ-14 |
| T-29 | P | OverlapGateSaysNo_Creates | REQ-5, REQ-14 |
| T-30 | P | RespectsKBudget | REQ-5 |
| T-31 | E | EnrichAddsKeywordsAndContext | DES-3 |

## ARCH-6: Catch-Up Processor

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-32 | P | NoMissedCorrections_NoNewMemories | REQ-15 |
| T-33 | P | MissedCorrectionReconciled | REQ-15 |
| T-34 | P | NewPatternAddedAsCandidate | REQ-15 |
| T-35 | E | CorpusGrowth_Scenario | DES-8 |

## ARCH-7: Audit Log

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-36 | P | EntryHasTimestampOperationAction | REQ-22 |
| T-37 | P | AppendOnly | REQ-22 |
| T-38 | E | FormatMatchesDES7 | DES-7 |

## ARCH-8: DI Wiring

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-39 | I | Wiring_ExtractorHasAllDependencies | ARCH-8 |
| T-40 | I | Wiring_CorrectionDetectorHasAllDependencies | ARCH-8 |
| T-41 | I | Wiring_CatchupProcessorHasAllDependencies | ARCH-8 |

## ARCH-9: Hook Scripts

| ID | Type | Test | Traces to |
|----|------|------|-----------|
| T-42 | I | HookScript_StopInvokesExtractAndCatchup | ARCH-9 |
| T-43 | I | HookScript_UserPromptSubmitInvokesCorrect | ARCH-9 |

---

## Summary

- Property-based: 18 tests (T-1..T-5, T-10..T-15, T-19..T-22, T-27..T-30, T-32..T-34, T-36..T-37)
- Example-based: 11 tests (T-6..T-7, T-16..T-17, T-23..T-25, T-31, T-35, T-38)
- Integration: 8 tests (T-8..T-9, T-18, T-26, T-39..T-43)
- Total: 37 tests

## Bidirectional Traceability

Every ARCH decision has at least one test. ARCH-2 (binary command structure) is verified through ARCH-9 integration tests and end-to-end tests in ARCH-3/4.

Every L2A REQ item is verified through at least one test:
- REQ-1 → T-10
- REQ-2 → T-1, T-5, T-11
- REQ-3 → T-2, T-12
- REQ-5 → T-3, T-4, T-14, T-15, T-27..T-30
- REQ-6 → verified by build (Go binary compiles without CGO)
- REQ-13 → T-19..T-21
- REQ-14 → T-20, T-27..T-29
- REQ-15 → T-32..T-34
- REQ-18 → T-13, T-22
- REQ-22 → T-36..T-37

Every L2A DES item is verified through at least one test:
- DES-3 → T-23, T-24, T-31
- DES-5 → T-25
- DES-6 → T-6, T-7, T-16, T-17
- DES-7 → T-38
- DES-8 → T-35
