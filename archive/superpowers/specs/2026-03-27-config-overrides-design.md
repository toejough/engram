# Configuration Overrides Design

Closes #397, #402. Makes hardcoded thresholds overridable via policies and config.

## #397: Maintenance Threshold Policy Overrides

Follow the existing surfacing override pattern (`surfacingPolicyToFrecencyOpts`). Active `DimensionMaintenance` policies with `Parameter`/`Value` pairs override thresholds in review and maintain.

### Overridable thresholds

| Parameter | Default | Used in | Effect |
|-----------|---------|---------|--------|
| `effectivenessThreshold` | 50.0 | review.go | Quadrant boundary for "high" effectiveness |
| `flagThreshold` | 40.0 | review.go | Below this → memory flagged for action |
| `minEvaluations` | 5 | review.go | Minimum feedback events for quadrant assignment |
| `stalenessThresholdDays` | 90 | maintain.go | Days before a working memory is considered stale |
| `refineKeywordsIrrelevanceThreshold` | 0.6 | maintain.go | Irrelevance ratio triggering keyword refinement |

### Approach

Add functional options to `review.Classify` and `maintain.Triage`:
- `review.WithEffectivenessThreshold(float64)`
- `review.WithFlagThreshold(float64)`
- `review.WithMinEvaluations(int)`
- `maintain.WithStalenessThresholdDays(int)`
- `maintain.WithRefineKeywordsIrrelevanceThreshold(float64)`

In CLI wiring (`cli.go`), convert active maintenance policies to options:
```
for _, p := range pf.Active(policy.DimensionMaintenance) {
    switch p.Parameter {
    case "effectivenessThreshold": ...
    case "flagThreshold": ...
    ...
    }
}
```

Same pattern as `surfacingPolicyToFrecencyOpts` in surface.go.

## #402: Adaptation Config in policy.toml

Add an `[adaptation]` section to the existing `policy.toml` file. Parsed into `policy.File.Adaptation` field. Values fall back to hardcoded defaults when zero-valued.

### Fields

| TOML key | Type | Default | Maps to |
|----------|------|---------|---------|
| `min_cluster_size` | int | 5 | adapt.Config.MinClusterSize |
| `min_feedback_events` | int | 3 | adapt.Config.MinFeedbackEvents |
| `measurement_window` | int | 10 | adapt.Config.MeasurementWindow |
| `maintenance_min_outcomes` | int | 3 | adapt.Config.MaintenanceMinOutcomes |
| `maintenance_min_success` | float64 | 0.4 | adapt.Config.MaintenanceMinSuccess |
| `min_new_feedback` | int | 5 | adapt.Config.MinNewFeedback |

### Approach

Add `AdaptationConfig` struct to `internal/policy/policy.go`:
```go
type AdaptationConfig struct {
    MinClusterSize         int     `toml:"min_cluster_size,omitempty"`
    MinFeedbackEvents      int     `toml:"min_feedback_events,omitempty"`
    MeasurementWindow      int     `toml:"measurement_window,omitempty"`
    MaintenanceMinOutcomes int     `toml:"maintenance_min_outcomes,omitempty"`
    MaintenanceMinSuccess  float64 `toml:"maintenance_min_success,omitempty"`
    MinNewFeedback         int     `toml:"min_new_feedback,omitempty"`
}
```

Add `Adaptation AdaptationConfig` field to `policy.File`.

In `runAdaptationAnalysis`, read from `adaptPF.Adaptation` with fallback to defaults:
```go
cfg := adaptPF.Adaptation.ToAdaptConfig(defaultConfig)
```

A `ToAdaptConfig` method applies non-zero overrides on top of defaults.

## Integration

Both features are wired in the CLI layer. Policy overrides (#397) are read at runtime per invocation. Config values (#402) are read from the same policy.toml file.
