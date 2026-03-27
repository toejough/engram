# Consolidation Confidence Threshold Filter

**Issue:** #403
**Date:** 2026-03-27

## Problem

`Consolidator.Plan()` returns all clusters with >= 2 members regardless of TF-IDF confidence score. Clusters with confidence as low as 0.08 get surfaced in triage, wasting user attention. In a recent triage session, 5 of 7 consolidation candidates had confidence < 0.25.

## Design

### Config field

Add `ConsolidationMinConfidence float64` to `policy.AdaptationConfig` with TOML key `consolidation_min_confidence`. Add matching field to `adapt.Config`. Wire through `adaptationConfigToAdaptConfig` using the existing zero-means-default pattern.

Default: **0.8**, set in `defaultAdaptConfig()`.

### Domain filter

Add `minConfidence float64` field to `signal.Consolidator`. Add `WithMinConfidence(float64)` functional option.

In `Plan()`, after computing confidence for a cluster, skip the cluster if:
- `confidence >= 0` (scorer is available) AND `confidence < c.minConfidence`

Clusters where confidence is -1 (no scorer configured) pass through unfiltered, preserving existing behavior when no TF-IDF scorer is wired.

### CLI wiring

In `RunMaintain()`, pass `signal.WithMinConfidence(analysisConfig.ConsolidationMinConfidence)` when constructing the consolidator. Same for `runAdaptationAnalysis()` if it also constructs a consolidator.

## Scope

- `internal/policy/policy.go` — add field to `AdaptationConfig`
- `internal/adapt/analyze.go` — add field to `Config`
- `internal/cli/cli.go` — wire override in `adaptationConfigToAdaptConfig`, pass to consolidator
- `internal/signal/consolidate.go` — add field, option, filter in `Plan()`
- Tests for each layer
