# Query timing instrument (--timings) Specification

## Purpose

`engram query --timings` emits per-stage wall-clock timing blocks (scan / embed / cluster / nominate / render) for the query in-flight phase, computed through the injected clock (internal/cli/query.go `phaseTimer`) and gated so the default recall payload remains byte-identical. The `nominate` stage key is retained from the tag-nomination era (ADR-0011) but now measures explore-half sampling (centroid-proximity softmax allocation, `internal/cli/query_explore.go`) plus supersession ride-along assembly — see ADR-0025. It is a measurement-only diagnostic (#691) for decomposing query speed to identify dominant stages before optimizing. Why: #691 (no dedicated ADR). Validation: dev/eval/LEDGER.md#query-inflight-split.

## Requirements

### Requirement: Timings block measures per-stage wall-clock durations

When `--timings` is set, the query SHALL emit a `timings:` YAML block on the payload with separate millisecond measurements for scan, embed, cluster, nominate (explore-half sampling + ride-along assembly), and render stages.

#### Scenario: Timings block structure on normal query
- **WHEN** running `engram query --timings` with valid phrases
- **THEN** the output payload includes a `timings:` block with five keys: `scan_ms`, `embed_ms`, `cluster_ms`, `nominate_ms`, `render_ms`

#### Scenario: Timings identifies dominant cost
- **WHEN** running `engram query --timings` on a real-scale vault with 41,367 chunk-index files (364 MB)
- **THEN** the `scan` stage dominates at ~67% of binary wall (4,787 ms median [4,748–4,965 ms], vs embed 2,009 ms, cluster 212 ms, nominate 60 ms, render 0 ms)

### Requirement: Timings measurement is transparent to recall consumers

The `--timings` flag SHALL NOT alter the payload structure for clients that ignore it; the timings block is additive and the default behavior is byte-identical.

#### Scenario: Timings omitted by default
- **WHEN** running `engram query` without `--timings`
- **THEN** no `timings:` block appears in the payload
