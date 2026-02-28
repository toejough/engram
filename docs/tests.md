# Test Specifications (L4)

Behavioral test list for Write Path (L3A: ARCH-1 through ARCH-9) and Read Path (L3B: ARCH-10 through ARCH-12).
BDD format: Given/When/Then triplets per interaction boundary.
Default property-based. Example-based entries justified inline.
Actors explicit in When/Then. "any" means property-generated.

---

## ARCH-1: Memory Storage

Integration tests with real SQLite FTS5 store. No mocks — all operations against actual database.

### T-1: Create populates all metadata fields [integration, ARCH-1]

Given any Memory m with all metadata fields populated
When test calls store.Create with (any ctx, m)
Then store.Create returns nil error

When test calls store.Get with (any ctx, m.ID)
Then store.Get returns (Memory got, nil error)
  And got matches m on: ObservationType, Concepts, Principle, AntiPattern, Rationale, EnrichedContent, Keywords

### T-2: Create rejects empty confidence [integration, ARCH-1]
Example-based: edge case, specific empty-string boundary.

Given a Memory m with Confidence = ""
When test calls store.Create with (any ctx, m)
Then store.Create returns non-nil error

### T-3: FindSimilar returns scored candidates [integration, ARCH-1]
Example-based: FTS5 BM25 ranking requires specific content to verify ordering.

Given two memories in store:
  m1 with content "Always use git add with specific file paths", keywords ["git","staging","add"]
  m2 with content "All file access through injected interfaces", keywords ["dependency-injection","interfaces"]
When test calls store.FindSimilar with (any ctx, "git staging add files", 5)
Then store.FindSimilar returns (results, nil error)
  And results is non-empty
  And results[0].Memory.ID equals m1.ID
  And scores are in descending order

### T-4: FindSimilar respects K limit [integration, ARCH-1]

Given any K in [1,10], more than K memories in store with overlapping content
When test calls store.FindSimilar with (any ctx, any matching query, K)
Then store.FindSimilar returns (results, nil error)
  And len(results) <= K

### T-5: Update preserves CreatedAt [integration, ARCH-1]

Given any Memory m created in store at time T1
When test calls store.Update with (any ctx, m with UpdatedAt = T2 where T2 > T1)
Then store.Update returns nil error

When test calls store.Get with (any ctx, m.ID)
Then store.Get returns (Memory got, nil error)
  And got.CreatedAt equals T1
  And got.UpdatedAt equals T2

### T-6: Enrichment increments count [integration, ARCH-1]

Given any Memory m in store with EnrichmentCount = N
When test calls store.Update with (any ctx, m with EnrichmentCount = N+1)
Then store.Update returns nil error

When test calls store.Get with (any ctx, m.ID)
Then store.Get returns (Memory got, nil error)
  And got.EnrichmentCount equals N+1

### T-7: FindSimilar ranks keyword matches higher [integration, ARCH-1]
Example-based: keyword column weighting verification requires specific content.

Given two memories in store:
  m1 with content "This project uses a build tool", keywords ["targ","build","test"]
  m2 with content "Remember to use targ for building and testing", keywords ["build","tool"]
When test calls store.FindSimilar with (any ctx, "targ", 5)
Then store.FindSimilar returns (results, nil error)
  And len(results) >= 2
  And results[0].Memory.ID equals m1.ID

### T-8: FTS5 index syncs with table [integration, ARCH-1]
Example-based: index synchronization requires controlled keyword sequence.

Given a Memory m in store with keywords ["unique-keyword-xyz"]
When test calls store.FindSimilar with (any ctx, "unique-keyword-xyz", 5)
Then store.FindSimilar returns (non-empty results, nil error)

When test calls store.Update with (any ctx, m with keywords = ["different-keyword-abc"])
Then store.Update returns nil error

When test calls store.FindSimilar with (any ctx, "unique-keyword-xyz", 5)
Then store.FindSimilar returns (empty results, nil error)

When test calls store.FindSimilar with (any ctx, "different-keyword-abc", 5)
Then store.FindSimilar returns (non-empty results, nil error)

### T-9: Concurrent reads don't conflict [integration, ARCH-1]
Example-based: concurrency testing requires controlled parallel execution.

Given a Memory m in store
When 10 goroutines concurrently call store.FindSimilar with (any ctx, any query, 5)
Then all calls return nil error

---

## ARCH-3: Extraction Pipeline

Unit tests. Mocks: Enricher, Classifier, MemoryStore, OverlapGate.
Target function: ExtractRun.

### T-10: Empty transcript produces no memories [unit, ARCH-3]

Given any transcript t
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given empty learnings, nil error
When enricher.Enrich responds with (empty, nil)
Then ExtractRun returns (any string, nil error)
  And ExtractRun never calls classifier, store, or gate

### T-11: Quality gate rejects content below token threshold [unit, ARCH-3]

Given any transcript t, any RawLearning with content shorter than 10 tokens
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [shortLearning], nil error
When enricher.Enrich responds with ([shortLearning], nil)
Then ExtractRun returns (string containing "rejected", any error)
  And ExtractRun never calls classifier, store, or gate

### T-12: Every created memory has a confidence tier [unit, ARCH-3]

Given any transcript t, any two RawLearnings l1 l2, any tier in {A, B, C}
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [l1, l2], nil error
When enricher.Enrich responds with ([l1, l2], nil)
Then for each learning, ExtractRun calls classifier.Classify with (any ctx, learning, t)

Given tier, nil error
When classifier.Classify responds with (tier, nil)
Then ExtractRun calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then ExtractRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then continues to next learning (same sequence repeats for l2)

ExtractRun returns (string containing "confidence=", nil error)

### T-13: Dedup skips mid-session corrections [unit, ARCH-3]

Given any transcript t, any two RawLearnings: overlapping and fresh
  sessionOverlaps marks overlapping.Content as already captured
When test calls ExtractRun with (enricher, classifier, store, gate, sessionOverlaps, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [overlapping, fresh], nil error
When enricher.Enrich responds with ([overlapping, fresh], nil)
Then ExtractRun skips overlapping, calls classifier.Classify with (any ctx, fresh, t)

Given "B", nil error
When classifier.Classify responds with ("B", nil)
Then ExtractRun calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then ExtractRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then ExtractRun returns (string containing "skipped", nil error)

### T-14: Reconciliation enriches on overlap [unit, ARCH-3]

Given any transcript t, any RawLearning, any existing Memory
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [learning], nil error
When enricher.Enrich responds with ([learning], nil)
Then ExtractRun calls classifier.Classify with (any ctx, learning, t)

Given "B", nil error
When classifier.Classify responds with ("B", nil)
Then ExtractRun calls store.FindSimilar with (any ctx, any string, K > 0)

Given [ScoredMemory{existing, 0.9}], nil error
When store.FindSimilar responds with ([ScoredMemory{existing, 0.9}], nil)
Then ExtractRun calls gate.Check with (any ctx, any Learning, existing)

Given true, "overlapping", nil error
When gate.Check responds with (true, "overlapping", nil)
Then ExtractRun calls store.Update with (any ctx, non-nil Memory)

Given nil error
When store.Update responds with nil
Then ExtractRun returns (string containing "enriched", nil error)

### T-15: Reconciliation creates on no overlap [unit, ARCH-3]

Given any transcript t, any RawLearning
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [learning], nil error
When enricher.Enrich responds with ([learning], nil)
Then ExtractRun calls classifier.Classify with (any ctx, learning, t)

Given "C", nil error
When classifier.Classify responds with ("C", nil)
Then ExtractRun calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then ExtractRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then ExtractRun returns (string containing "created", nil error)

### T-16: Real session scenario — 4 learnings, 1 dedup, 3 reconciled [unit, ARCH-3]

Given any transcript t, any four RawLearnings l1 l2 l3 l4
  sessionOverlaps marks l2.Content as already captured
When test calls ExtractRun with (enricher, classifier, store, gate, sessionOverlaps, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [l1, l2, l3, l4], nil error
When enricher.Enrich responds with ([l1, l2, l3, l4], nil)
Then for each of l1, l3, l4 (l2 skipped by dedup):
  ExtractRun calls classifier.Classify → responds with ("B", nil)
  ExtractRun calls store.FindSimilar → responds with (nil, nil)
  ExtractRun calls store.Create → responds with nil

ExtractRun returns (string containing "skipped" and "created", nil error)

### T-17: All rejected by quality gate — audit records reasons [unit, ARCH-3]

Given any transcript t, any two RawLearnings l1 l2 each with content shorter than 10 tokens
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [l1, l2], nil error
When enricher.Enrich responds with ([l1, l2], nil)
Then ExtractRun returns (string containing "rejected", any error)
  And ExtractRun never calls classifier, store, or gate

### T-18: Pipeline end-to-end [unit, ARCH-3]

Given any transcript t, any RawLearning
When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given [learning], nil error
When enricher.Enrich responds with ([learning], nil)
Then ExtractRun calls classifier.Classify with (any ctx, learning, t)

Given "A", nil error
When classifier.Classify responds with ("A", nil)
Then ExtractRun calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then ExtractRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then ExtractRun returns (non-empty string, nil error)

---

## ARCH-4: Correction Detection

Unit tests. Mocks: MemoryStore, OverlapGate.
Pure deps wired internally: PatternCorpus, Reconciler, SessionRecorder, AuditLogger.
Target function: DetectCorrection.

### T-19: No match returns empty [unit, ARCH-4]

Given any message, nil patterns (empty corpus)
When test calls DetectCorrection with (store, gate, nil, any ctx, message)
Then DetectCorrection returns ("", empty recordings, any string, nil error)

### T-20: Match triggers reconciliation [unit, ARCH-4]
Example-based: specific message needed to trigger known pattern.

Given patterns including `^no,`
When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, use specific files not git add -A")
Then DetectCorrection calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then DetectCorrection calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then DetectCorrection returns (non-empty reminder, 1 recording, non-empty audit, nil error)

### T-21: All 15 initial patterns match expected input [unit, ARCH-4a]
Example-based: each pattern tested against its specific matching string.

Given each pattern from the initial corpus and its expected matching string:
  `^no,` matches "no, use specific files"
  `^wait` matches "wait, that's wrong"
  `^hold on` matches "hold on, let me check"
  `\bwrong\b` matches "that's wrong"
  `\bdon't\s+\w+` matches "don't use that"
  `\bstop\s+\w+ing` matches "stop deleting files"
  `\btry again` matches "try again with the right path"
  `\bgo back` matches "go back to the previous version"
  `\bthat's not` matches "that's not what I meant"
  `^actually,` matches "actually, use bun instead"
  `\bremember\s+(that|to)` matches "remember to run tests"
  `\bstart over` matches "start over from scratch"
  `\bpre-?existing` matches "that's a pre-existing issue"
  `\byou're still` matches "you're still making that mistake"
  `\bincorrect` matches "that's incorrect"
When test calls corpus.Match with the input string
Then corpus.Match returns a non-nil match

### T-22: Correction recorded to session log [unit, ARCH-4]
Example-based: specific pattern-matching message.

Given patterns including `^no,`
When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, that's not right")
Then DetectCorrection calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then DetectCorrection calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then DetectCorrection returns (any string, recordings with len 1, any string, nil error)

### T-23: Enriched existing memory — system reminder says "Enriched:" [unit, ARCH-4]
Example-based: specific message and existing memory to trigger enrichment path.

Given patterns including `^no,`, existing Memory with ID "m_0001" and Title "Use git add specific files"
When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, don't use git add -A")
Then DetectCorrection calls store.FindSimilar with (any ctx, any string, K > 0)

Given [ScoredMemory{existing, 0.9}], nil error
When store.FindSimilar responds with ([ScoredMemory{existing, 0.9}], nil)
Then DetectCorrection calls gate.Check with (any ctx, any Learning, existing)

Given true, "overlapping", nil error
When gate.Check responds with (true, "overlapping", nil)
Then DetectCorrection calls store.Update with (any ctx, non-nil Memory)

Given nil error
When store.Update responds with nil
Then DetectCorrection returns (reminder, any, any, any)
  And reminder contains "Enriched:"
  And reminder contains "Use git add specific files"
  And reminder contains "Correction captured"

### T-24: Created new memory — system reminder says "Created:" [unit, ARCH-4]
Example-based: specific pattern-matching message.

Given patterns including `^wait`
When test calls DetectCorrection with (store, gate, patterns, any ctx, "wait, this project uses bun not npm")
Then DetectCorrection calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then DetectCorrection calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then DetectCorrection returns (reminder containing "Created:", any, any, nil error)

### T-25: False positive captured anyway [unit, ARCH-4]
Example-based: "remember to run tests" triggers `\bremember\s+(that|to)` — a false positive, but captured without confirmation per DES-5.

Given patterns including `\bremember\s+(that|to)`
When test calls DetectCorrection with (store, gate, patterns, any ctx, "remember to run tests before committing")
Then DetectCorrection calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then DetectCorrection calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then DetectCorrection returns (non-empty reminder, any, any, nil error)

### T-26: End-to-end correction detection [unit, ARCH-4]
Example-based: specific correction message.

Given patterns including `^no,`
When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, use targ test not go test")
Then DetectCorrection calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then DetectCorrection calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then DetectCorrection returns (non-empty reminder, 1 recording, non-empty audit, nil error)

---

## ARCH-5: Reconciler

Unit tests. Mocks: MemoryStore, OverlapGate.
Target function: ReconcileRun.

### T-27: Empty store creates new memory [unit, ARCH-5]

Given any Learning l, any K in [1,10]
When test calls ReconcileRun with (store, gate, K, any ctx, l)
Then ReconcileRun calls store.FindSimilar with (any ctx, any string, equal to K)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then ReconcileRun returns (ReconcileResult{Action: "created"}, nil error)

### T-28: Overlap gate says yes — enriches best candidate [unit, ARCH-5]

Given any Learning l, any existing Memory
When test calls ReconcileRun with (store, gate, 3, any ctx, l)
Then ReconcileRun calls store.FindSimilar with (any ctx, any string, 3)

Given [ScoredMemory{existing, 0.9}], nil error
When store.FindSimilar responds with ([ScoredMemory{existing, 0.9}], nil)
Then ReconcileRun calls gate.Check with (any ctx, l, existing)

Given true, "overlapping content", nil error
When gate.Check responds with (true, "overlapping content", nil)
Then ReconcileRun calls store.Update with (any ctx, non-nil Memory)

Given nil error
When store.Update responds with nil
Then ReconcileRun returns (ReconcileResult{Action: "enriched"}, nil error)

### T-29: Overlap gate says no for all — creates new [unit, ARCH-5]

Given any Learning l, any two existing Memories m1 m2
When test calls ReconcileRun with (store, gate, 3, any ctx, l)
Then ReconcileRun calls store.FindSimilar with (any ctx, any string, 3)

Given [ScoredMemory{m1, 0.8}, ScoredMemory{m2, 0.5}], nil error
When store.FindSimilar responds with candidates
Then ReconcileRun calls gate.Check with (any ctx, l, m1)

Given false, "different topic", nil error
When gate.Check responds with (false, "different topic", nil)
Then ReconcileRun calls gate.Check with (any ctx, l, m2)

Given false, "different topic", nil error
When gate.Check responds with (false, "different topic", nil)
Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then ReconcileRun returns (ReconcileResult{Action: "created"}, nil error)

### T-30: Respects K budget [unit, ARCH-5]

Given any Learning l, any K in [1,3], exactly K candidates
When test calls ReconcileRun with (store, gate, K, any ctx, l)
Then ReconcileRun calls store.FindSimilar with (any ctx, any string, K)

Given K candidates, nil error
When store.FindSimilar responds with (K candidates, nil)
Then ReconcileRun calls gate.Check exactly K times, once per candidate

Given false for each
When gate.Check responds with (false, "no overlap", nil) for each
Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then ReconcileRun returns (ReconcileResult{Action: "created"}, nil error)

### T-31: Enrich adds keywords and increments count [unit, ARCH-5]

Given any Learning l with keywords newKW, any existing Memory with keywords existingKW and EnrichmentCount N
When test calls ReconcileRun with (store, gate, 3, any ctx, l)
Then ReconcileRun calls store.FindSimilar with (any ctx, any string, 3)

Given [ScoredMemory{existing, 0.9}], nil error
When store.FindSimilar responds with candidates
Then ReconcileRun calls gate.Check with (any ctx, l, existing)

Given true, "overlapping", nil error
When gate.Check responds with (true, "overlapping", nil)
Then ReconcileRun calls store.Update with (any ctx, updated Memory)

Verify updated Memory:
  Keywords contains all of existingKW AND all of newKW
  EnrichmentCount equals N+1

Given nil error
When store.Update responds with nil
Then ReconcileRun returns (ReconcileResult{Action: "enriched"}, nil error)

---

## ARCH-6: Catch-Up Processor

Unit tests. Mocks: CatchupEvaluator, MemoryStore, OverlapGate.
Pure deps wired internally: Reconciler, PatternCorpus, SessionLog, AuditLogger.
Target function: CatchupRun.

### T-32: No missed corrections — no new memories [unit, ARCH-6]

Given any transcript t, empty capturedEvents
When test calls CatchupRun with (evaluator, store, gate, capturedEvents, any ctx, t)
Then CatchupRun calls evaluator.FindMissed with (any ctx, equal to t, equal to capturedEvents)

Given nil, nil error
When evaluator.FindMissed responds with (nil, nil)
Then CatchupRun returns (empty candidates, any string, nil error)

### T-33: Missed correction reconciled [unit, ARCH-6]

Given any transcript t
When test calls CatchupRun with (evaluator, store, gate, nil, any ctx, t)
Then CatchupRun calls evaluator.FindMissed with (any ctx, equal to t, any)

Given one MissedCorrection, nil error
When evaluator.FindMissed responds with ([{Content, Context, Phrase}], nil)
Then CatchupRun calls store.FindSimilar with (any ctx, any string, K > 0)

Given nil, nil error
When store.FindSimilar responds with (nil, nil)
Then CatchupRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil
Then CatchupRun returns (non-empty candidates, non-empty audit, nil error)

### T-34: New pattern added as candidate [unit, ARCH-6]

Given any transcript t, any phrase
When test calls CatchupRun with (evaluator, store, gate, nil, any ctx, t)
Then CatchupRun calls evaluator.FindMissed with (any ctx, equal to t, any)

Given [{Content: "missed correction", Context: "context", Phrase: phrase}], nil error
When evaluator.FindMissed responds
Then CatchupRun calls store.FindSimilar → responds with (nil, nil)
Then CatchupRun calls store.Create → responds with nil

CatchupRun returns (candidates, audit, nil error)
  And len(candidates) equals 1
  And candidates[0].Regex equals phrase

### T-35: Full scenario — missed correction, memory, candidate, audit [unit, ARCH-6]
Example-based: specific scenario with pre-captured events and known missed correction.

Given any transcript t, capturedEvents = [{MemoryID: "m_other", Pattern: `^no,`, Message: "no, use bun"}]
When test calls CatchupRun with (evaluator, store, gate, capturedEvents, any ctx, t)
Then CatchupRun calls evaluator.FindMissed with (any ctx, equal to t, equal to capturedEvents)

Given [{Content: "you didn't shut them down", Context: "orphaned teammates", Phrase: `\byou didn't\b`}], nil error
When evaluator.FindMissed responds
Then CatchupRun calls store.FindSimilar → responds with (nil, nil)
Then CatchupRun calls store.Create → responds with nil

CatchupRun returns (candidates, auditOutput, nil error)
  And len(candidates) equals 1
  And candidates[0].Regex equals `\byou didn't\b`
  And auditOutput is non-empty

---

## ARCH-7: Audit Log

Unit tests. No mocks — pure implementation writing to bytes.Buffer.

### T-36: Entry has timestamp, operation, and action [unit, ARCH-7]
Example-based: verifies specific field presence in output format.

Given an audit.Logger writing to a bytes.Buffer
When test calls log.Log with Entry{Timestamp: 2026-02-27T16:30:00Z, Operation: "extract", Action: "created", Fields: {"memory_id": "m_7f3a"}}
Then log.Log returns nil error
  And buffer contains "2026-02-27T16:30:00Z"
  And buffer contains "extract"
  And buffer contains "created"

### T-37: Append only — new entries don't modify prior entries [unit, ARCH-7]
Example-based: verifies sequential write behavior.

Given an audit.Logger writing to a bytes.Buffer
When test calls log.Log with first entry
Then buffer contains first entry text

When test calls log.Log with second entry
Then buffer starts with first entry text (prefix preserved)
  And buffer contains exactly 2 lines

### T-38: Format matches DES-7 key-value specification [unit, ARCH-7]
Example-based: verifies exact output format against DES-7 spec.

Given an audit.Logger writing to a bytes.Buffer
When test calls log.Log with Entry{Timestamp: 2026-02-27T16:30:00Z, Operation: "extract", Action: "created", Fields: {"memory_id": "m_7f3a", "confidence": "B"}}
Then output line format is: <RFC3339> <operation> <action> <key=value pairs>
  And line starts with "2026-02-27T16:30:00Z"
  And parts[1] equals "extract"
  And parts[2] equals "created"
  And line contains "memory_id=m_7f3a"
  And line contains "confidence=B"

When test calls log.Log with Fields: {"content": "Always check things carefully"}
Then values containing spaces are quoted: content="Always check things carefully"

---

## ARCH-8: DI Wiring

Integration tests. No mocks — verifies constructors reject nil dependencies.

### T-39: Extractor constructor requires all dependencies [integration, ARCH-8]
Example-based: specific nil-check behavior.

Given an empty ExtractorConfig (all fields nil/zero)
When test calls extract.NewExtractor with (ExtractorConfig{})
Then NewExtractor returns non-nil error
  And error message contains each dependency name: "Enricher", "Gate", "Classifier", "Reconciler", "Session", "Audit"

### T-40: CorrectionDetector constructor requires all dependencies [integration, ARCH-8]
Example-based: specific nil-check behavior.

Given an empty DetectorConfig (all fields nil/zero)
When test calls correct.NewDetector with (DetectorConfig{})
Then NewDetector returns non-nil error
  And error message contains each dependency name: "Corpus", "Recon", "Session", "Audit"

### T-41: CatchupProcessor constructor requires all dependencies [integration, ARCH-8]
Example-based: specific nil-check behavior.

Given an empty ProcessorConfig (all fields nil/zero)
When test calls catchup.NewProcessor with (ProcessorConfig{})
Then NewProcessor returns non-nil error
  And error message contains each dependency name: "Evaluator", "Reconciler", "Corpus", "Session", "Audit"

---

## ARCH-9: Hook Scripts

Unit tests. No mocks — pure string content verification.

### T-42: Stop hook invokes extract and catchup [unit, ARCH-9]
Example-based: verifies specific script content.

When test calls hooks.StopScript()
Then returned string contains "engram extract"
  And contains "engram catchup"
  And contains "CLAUDE_SESSION_TRANSCRIPT"
  And contains "set -euo pipefail"

### T-43: UserPromptSubmit hook invokes correct [unit, ARCH-9]
Example-based: verifies specific script content.

When test calls hooks.UserPromptSubmitScript()
Then returned string contains "engram correct"
  And contains "CLAUDE_USER_MESSAGE"
  And contains "set -euo pipefail"

---

---

## ARCH-10: Frecency Ranking (Store.Surface)

Integration tests with real SQLite FTS5 store. Tests the `Surface` method which uses FTS5 MATCH + frecency ORDER BY.

### T-44: Surface returns frecency-ranked results [integration, ARCH-10]
Example-based: specific timestamps and impact scores needed to verify ordering.

Given three memories in store with overlapping content matching query:
  m1: created 1 day ago, impact_score = 0.9 (high frecency: both recent and high impact)
  m2: created 30 days ago, impact_score = 0.9 (high impact but old — harmonic mean penalizes)
  m3: created 1 day ago, impact_score = 0.1 (recent but low impact — harmonic mean penalizes)
When test calls store.Surface with (any ctx, matching query, 5)
Then store.Surface returns (results, nil error)
  And results are non-empty
  And results[0].Memory.ID equals m1.ID
  And results are in descending frecency order

### T-45: Cold start ranking equals recency [integration, ARCH-10]
Example-based: all default impact, ordering by creation time.

Given three memories in store all with default impact_score = 0.5:
  m1: created 1 day ago
  m2: created 7 days ago
  m3: created 30 days ago
  All with overlapping content matching query
When test calls store.Surface with (any ctx, matching query, 5)
Then store.Surface returns (results, nil error)
  And results ordered: m1, m2, m3 (most recent first)

### T-46: Confidence tiebreaker when frecency tied [integration, ARCH-10]
Example-based: identical timestamps and impact, different confidence.

Given two memories in store with identical created_at, updated_at, impact_score:
  m1: confidence = "A"
  m2: confidence = "B"
  Both with overlapping content matching query
When test calls store.Surface with (any ctx, matching query, 5)
Then store.Surface returns (results, nil error)
  And results[0].Memory.ID equals m1.ID (A > B tiebreaker)

### T-47: Surface respects K limit [integration, ARCH-10]

Given any K in [1,5], more than K memories in store with overlapping content
When test calls store.Surface with (any ctx, any matching query, K)
Then store.Surface returns (results, nil error)
  And len(results) <= K

### T-48: IncrementSurfacing updates metadata [integration, ARCH-10/ARCH-11]
Example-based: specific count increment and timestamp update.

Given a Memory m in store with SurfacingCount = 0 and LastSurfacedAt = ""
When test calls store.IncrementSurfacing with (any ctx, [m.ID])
Then store.IncrementSurfacing returns nil error

When test calls store.Get with (any ctx, m.ID)
Then store.Get returns (Memory got, nil error)
  And got.SurfacingCount equals 1
  And got.LastSurfacedAt is non-empty

---

## ARCH-11: Surfacing Pipeline

### Formatter (pure function, no mocks)

Impact label mapping: surfacing_count == 0 → "new"; impact_score >= 0.75 → "high"; >= 0.25 → "medium"; < 0.25 → "low".

### T-49: Full format with numbered list [unit, ARCH-11]
Example-based: verifies DES-1 multi-memory format.

Given two ScoredMemory items:
  sm1: Title "Use targ build system", Content "Build commands: targ test, targ lint", Confidence "A", ImpactScore 0.9, SurfacingCount 5
  sm2: Title "DI pattern in internal/", Content "All I/O through injected interfaces", Confidence "B", ImpactScore 0.5, SurfacingCount 0
When test calls FormatSurfacing with ([sm1, sm2], "session-start")
Then result contains "<system-reminder source=\"engram\">"
  And result contains "[engram] 2 memories for this context:"
  And result contains "1. Use targ build system (A, high)"
  And result contains "Build commands:"
  And result contains "2. DI pattern in internal/ (B, new)"
  And result contains "</system-reminder>"

### T-50: Compact format for pre-tool-use [unit, ARCH-11]
Example-based: verifies DES-1 compact single-line variant.

Given one ScoredMemory: Title "Use targ test not go test", Confidence "A", ImpactScore 0.9, SurfacingCount 3
When test calls FormatSurfacing with ([sm], "pre-tool-use")
Then result contains "<system-reminder source=\"engram\">"
  And result contains "[engram] Use targ test not go test (A, high)"
  And result does NOT contain "1."
  And result does NOT contain "memories for this context"
  And result contains "</system-reminder>"

### T-51: Empty memories returns empty string [unit, ARCH-11]

Given empty memories
When test calls FormatSurfacing with ([], any hookType)
Then result equals ""

### T-52: Singular wording for one memory [unit, ARCH-11]
Example-based: DES-1 specifies "1 memory" singular, not "1 memories".

Given one ScoredMemory
When test calls FormatSurfacing with ([sm], "user-prompt")
Then result contains "[engram] 1 memory for this context:"
  And result does NOT contain "memories"

### Pipeline orchestration (mocked store, formatter, audit)

### T-53: Empty result returns empty string [unit, ARCH-11]

Given any query, any hook type, any budget
When test calls SurfaceRun with (store, formatter, audit, hookType, query, budget, ctx)
Then SurfaceRun calls store.Surface with (any ctx, query, budget)

Given empty results, nil error
When store.Surface responds with ([], nil)
Then SurfaceRun returns ("", nil error)
  And SurfaceRun never calls formatter, audit, or store.IncrementSurfacing

### T-54: Surfacing pipeline end-to-end [unit, ARCH-11]

Given any query, any hookType, any budget
When test calls SurfaceRun with (store, formatter, audit, hookType, query, budget, ctx)
Then SurfaceRun calls store.Surface with (any ctx, query, budget)

Given [sm1], nil error
When store.Surface responds with ([sm1], nil)
Then SurfaceRun calls formatter.FormatSurfacing with ([sm1], hookType)

Given formatted string
When formatter.FormatSurfacing responds with formatted string
Then SurfaceRun calls store.IncrementSurfacing with (any ctx, [sm1.Memory.ID])

Given nil error
When store.IncrementSurfacing responds with nil
Then SurfaceRun calls audit.Log with (Entry where Operation="surface", Action="returned")

Given nil error
When audit.Log responds with nil
Then SurfaceRun returns (formatted string, nil error)

---

## ARCH-12: Read-Path Hook Scripts

Unit tests. No mocks — pure string content verification.

### T-55: SessionStart hook invokes surface with session-start [unit, ARCH-12]
Example-based: verifies specific script content.

When test calls hooks.SessionStartScript()
Then returned string contains "surface"
  And contains "--hook session-start"
  And contains "CLAUDE_PROJECT_DIR"
  And contains "set -euo pipefail"
  And contains "bin/engram"

### T-56: UserPromptSubmit hook invokes both correct and surface [unit, ARCH-12]
Example-based: verifies DES-4 ordering — correction before surfacing.

When test calls hooks.UserPromptSubmitScript()
Then returned string contains "correct"
  And contains "surface"
  And contains "--hook user-prompt"
  And contains "CLAUDE_USER_MESSAGE"
  And index of "correct" < index of "surface" (correction first per DES-4)

### T-57: PreToolUse hook invokes surface with pre-tool-use [unit, ARCH-12]
Example-based: verifies specific script content.

When test calls hooks.PreToolUseScript()
Then returned string contains "surface"
  And contains "--hook pre-tool-use"
  And contains "CLAUDE_TOOL_INPUT"
  And contains "set -euo pipefail"
  And contains "bin/engram"

---

## ARCH-8/ARCH-11: DI Wiring (Read Path)

### T-58: SurfacePipeline constructor requires all dependencies [integration, ARCH-8/ARCH-11]
Example-based: specific nil-check behavior.

Given an empty SurfaceConfig (all fields nil/zero)
When test calls surface.NewPipeline with (SurfaceConfig{})
Then NewPipeline returns non-nil error
  And error message contains each dependency name: "Store", "Formatter", "Audit"

---

## Summary

- Property-based: 23 tests (T-1, T-4, T-5, T-6, T-10..T-19, T-27..T-34, T-47)
- Example-based: 35 tests (T-2, T-3, T-7..T-9, T-20..T-26, T-35..T-46, T-48..T-58)
- Unit: 41 tests (T-10..T-38, T-42..T-43, T-49..T-57)
- Integration: 17 tests (T-1..T-9, T-39..T-41, T-44..T-48, T-58)
- Total: 58 tests

## Bidirectional Traceability

Every ARCH decision has at least one test:
- ARCH-1 → T-1..T-9
- ARCH-2 → verified through ARCH-9/ARCH-12 tests and end-to-end tests in ARCH-3/4/11
- ARCH-3 → T-10..T-18
- ARCH-4 → T-19..T-26
- ARCH-4a (corpus) → T-21
- ARCH-5 → T-27..T-31
- ARCH-6 → T-32..T-35
- ARCH-7 → T-36..T-38
- ARCH-8 → T-39..T-41, T-58
- ARCH-9 → T-42..T-43
- ARCH-10 → T-44..T-48
- ARCH-11 → T-48..T-54, T-58
- ARCH-12 → T-55..T-57

Every L2A REQ item verified:
- REQ-1 → T-10, T-18, T-42
- REQ-2 → T-1, T-5, T-11
- REQ-3 → T-2, T-12
- REQ-5 → T-3, T-4, T-14, T-15, T-27..T-30
- REQ-6 → verified by build (pure Go, no CGO)
- REQ-13 → T-19..T-21
- REQ-14 → T-20, T-27..T-29
- REQ-15 → T-32..T-34
- REQ-18 → T-13, T-22
- REQ-22 → T-36..T-38

Every L2B REQ/DES item verified:
- REQ-4 → T-44..T-46
- REQ-7 → T-54, T-55
- REQ-8 → T-54, T-56
- REQ-9 → T-54, T-57
- REQ-10 → T-44 (no LLM in Surface query — pure SQL), T-53, T-54
- REQ-12 → T-49, T-50, T-52
- DES-1 → T-49, T-50, T-51, T-52
- DES-2 → T-55, T-56, T-57

Every L2A DES item verified:
- DES-3 → T-23, T-24, T-31
- DES-5 → T-25
- DES-6 → T-6, T-7, T-16, T-17
- DES-7 → T-38
- DES-8 → T-35
