# Matched-note floor Specification

## Purpose

Recall's clustering reserves per-phrase note slots so a crystallized lesson competing against a flood of raw transcript chunks still surfaces — a relevant note is not drowned out by noisier transcript fragments. The `capWithNoteFloor` function (internal/cli/query.go) guarantees up to 5 relevance-qualified notes (non-chunk, baseScore ≥ 0.25) survive per phrase, evicting lower-scoring chunks to make room. Why: docs/architecture/adr.md ADR-0004. Validation: dev/eval/LEDGER.md#matched-note-floor.

## Requirements

### Requirement: Per-phrase note floor reservation

The query SHALL reserve up to 5 relevance-qualified (baseScore ≥ matchRelevanceFloor = 0.25) note slots per phrase before enforcing the per-phrase matched-set limit (matchPhraseLimit = 30), even if evicting chunks below the cut.

#### Scenario: Note survives cap above relevance floor
- **WHEN** a phrase matches 30+ items, with a note ranked 6–30 at baseScore ≥ 0.25 and lower-scoring chunks ranked 1–5
- **THEN** the note is promoted into the final per-phrase set and the lowest-scoring chunk is evicted to make room

#### Scenario: Below-floor note does not get promoted
- **WHEN** a phrase matches a note below matchRelevanceFloor (baseScore < 0.25)
- **THEN** the note is not promoted by the floor reservation and drops out if ranked below the cap

### Requirement: Crystallized notes preserved against chunk drowning

The unified ranking across all matched items SHALL prevent a high-scoring chunk flood from fully evicting crystallized notes that clear the relevance floor, preserving their visibility in the clustered result.

#### Scenario: Note surfaces in crowded vault with mixed chunk/note matches
- **WHEN** clustering unions all per-phrase results across 10 phrases with a real-scale vault (400+ notes + chunk index)
- **THEN** crystallized notes ranked top-5 per phrase are not lost from the unified clustering; C3 trap gate measurably GREEN (note recall@5 0.22→0.83 isolation ceiling)
