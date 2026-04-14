# Scoring Formulas

Exact formulas used in engram's scoring pipeline, matched to implementation.

## Effectiveness Scoring

Source: `internal/effectiveness/aggregate.go`

```
effectiveness = (followed / (followed + contradicted + ignored + irrelevant)) * 100
```

Returns 0 when total evaluations is 0.

The review system (`internal/review/review.go`) classifies memories with fewer than `minEvaluations` (default 5) total feedback events as `InsufficientData`, excluding them from quadrant assignment.

## Quality-Weighted Ranking

Source: `internal/frecency/frecency.go`

The surfacing score combines BM25 relevance with quality:

```
score = relevance * genFactor * (1.0 + quality)
```

Where quality is:

```
quality = (wEff * effectiveness) + (wFreq * frequency) + (wTier * tierBoost)
```

Components:

- **effectiveness**: `followed / (followed + contradicted + ignored + irrelevant)`, returns `defaultEffectiveness` (0.5) when total is 0
- **frequency**: `log(1 + surfacedCount) / log(1 + maxSurfaced)`, returns 0 when `maxSurfaced <= 0`
- **tierBoost**: A = 1.2, B = 0.2, C/other = 0

Default weights:

| Weight | Default | Purpose |
|--------|---------|---------|
| wEff | 0.3 | Effectiveness contribution |
| wFreq | 1.0 | Frequency contribution |
| wTier | 0.3 | Confidence tier contribution |

All weights and tier boosts are overridable via `policy.toml` surfacing policies.

## Generalizability Factor

Source: `internal/surface/genfactor.go`

Applied only for cross-project surfacing. Same project, empty project slug, or matching slugs return 1.0.

| Generalizability | Factor |
|-----------------|--------|
| 0 (unset) | 0.25 |
| 1 (this-project) | 0.0 |
| 2 (narrow) | 0.0 |
| 3 (moderate) | 0.25 |
| 4 (similar) | 0.7 |
| 5 (universal) | 1.0 |

Out-of-range values fall back to 0.25 (the unset default).

## Irrelevance Penalty

Source: `internal/surface/surface.go`

Applied to BM25 score before quality weighting:

```
irrelevancePenalty = K / (K + irrelevantCount)
```

Where K = `irrelevancePenaltyHalfLife` = 5. At 0 irrelevant: 1.0, at 5: 0.5, at 10: 0.33.

The penalized BM25 score feeds into `relevance` in the combined score formula above.

## Surfacing Budget

Source: `internal/surface/budget.go`

Token estimation:

```
tokens = len(text) / 4
```

Per-hook budget caps:

| Hook | Budget (tokens) |
|------|----------------|
| SessionStart | 600 |
| UserPromptSubmit | 250 |
| Stop | 500 |

Memories are added in rank order until the next memory would exceed the remaining budget. Budget of 0 means unlimited.

Additional limits from `internal/surface/surface.go`:

- **coldStartBudget = 2**: max unproven memories per invocation (memories with no surfacing history)
- **promptLimit = 2**: max results before budget enforcement
