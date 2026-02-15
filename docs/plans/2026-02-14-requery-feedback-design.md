# ISSUE-232: Remove Re-Query Feedback Detection

**Date:** 2026-02-14
**Status:** Approved
**Approach:** A — Remove entirely

## Problem

The re-query similarity detection in `Query()` (`memory.go:502-534`) records "wrong" feedback for all previous query results whenever a new query has >0.85 cosine similarity to the last query. Hook-driven queries (SessionStart, UserPromptSubmit, PreToolUse) naturally produce similar queries across invocations, causing bulk "wrong" feedback insertion on every hook fire. Observed: 98,272 bogus "wrong" feedback rows across 171 embeddings in days of normal use.

Cascading effects: all embedding confidences driven to 0.0, all flagged_for_review, hundreds of false-positive "surface" proposals during optimize, unbounded feedback table growth.

## Decision

Remove re-query detection entirely. The feature predates the self-reinforcing learning design and its use case (detecting bad retrieval results) is covered by the designed retrieval-relevance scoring signal (correction-after-injection detection).

## Changes

1. **Delete re-query detection block** in `Query()` (`memory.go:502-534`) — similarity comparison and `RecordFeedback` loop
2. **Delete `SaveLastQueryResults` call** (`memory.go:536-540`)
3. **Delete `SaveLastQueryResults()` and `LoadLastQueryResults()` functions**
4. **Delete `LastQueryCache` struct**
5. **Delete `last_query.json`** at runtime during optimize (stale artifact cleanup)
6. **Reset `flagged_for_review`** on all embeddings — flags were set by bogus feedback
7. **Remove related tests** that exercise re-query detection / SaveLastQueryResults / LoadLastQueryResults

## What stays

- `RecordFeedback()` — needed for future explicit feedback mechanisms
- `feedback` table schema — will be used by designed retrieval-relevance scoring
- All other feedback-related code (GetFeedbackStats, ListFlaggedForReview, PropagateEmbeddingFeedbackToSkills)

## Relationship

- Supersedes the implicit re-ask detection from ISSUE-214
- Aligns with self-reinforcing learning design (2026-02-14) MEASURE phase
- Future retrieval-relevance scoring will provide a proper signal for "were results useful?"
