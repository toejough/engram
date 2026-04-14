# Data Model

## Memory TOML Files

Each memory is a single TOML file in `~/.claude/engram/data/memories/`. The filename is a slugified summary. The canonical struct is `memory.MemoryRecord`.

### Content Fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Short descriptive title |
| `content` | string | Full memory content |
| `observation_type` | string | Category of observation |
| `concepts` | string[] | Semantic concept tags |
| `keywords` | string[] | Keywords for BM25 retrieval |
| `principle` | string | Actionable principle (required for tier A) |
| `anti_pattern` | string | What to avoid (required for tier A, optional for B) |
| `rationale` | string | Why this matters |
| `confidence` | string | Tier: "A" (high), "B" (medium), "C" (low) |
| `generalizability` | int | 0 = project-specific, higher = more general |
| `project_slug` | string | Originating project identifier |

### Timestamps

| Field | Type | Description |
|-------|------|-------------|
| `created_at` | string | ISO 8601 creation time |
| `updated_at` | string | ISO 8601 last modification time |
| `last_surfaced_at` | string | ISO 8601 time last shown to agent |

### Tracking Counters

| Field | Type | Description |
|-------|------|-------------|
| `surfaced_count` | int | Times this memory was surfaced |
| `followed_count` | int | Times the advice was followed |
| `contradicted_count` | int | Times the advice was contradicted |
| `ignored_count` | int | Times the memory was surfaced but ignored |
| `irrelevant_count` | int | Times marked irrelevant to context |
| `irrelevant_queries` | string[] | Queries where memory was irrelevant |

### Provenance

| Field | Type | Description |
|-------|------|-------------|
| `source_type` | string | How the memory was created |
| `source_path` | string | Original source file/transcript |
| `content_hash` | string | Hash for dedup and change detection |

### Relationships

| Field | Type | Description |
|-------|------|-------------|
| `absorbed` | AbsorbedRecord[] | Memories merged into this one |

Each `AbsorbedRecord` contains: `from` (original path), `surfaced_count`, `evaluations` (followed/contradicted/ignored counts), `content_hash`, `merged_at`.

### Maintenance History

Each `MaintenanceAction` records: `action`, `applied_at`, `effectiveness_before`, `surfaced_count_before`, `feedback_count_before`, `effectiveness_after`, `surfaced_count_after`, `measured`.

## Supporting Data Files

All stored in `~/.claude/engram/data/`.

### `creation-log.jsonl`

Append-only log of memory creation events. Each line is a JSON object recording what was created, when, and from what source.

### `policy.toml`

Adaptive policy directives. Contains:
- `[[policies]]` array with lifecycle (proposed/approved/active/rejected/retired), dimension (extraction/surfacing/maintenance), directive text, evidence, and effectiveness tracking.
- `[approval_streak]` tracking consecutive approvals per dimension.
- `[adaptation]` configuration overrides for analysis thresholds.

## Effectiveness Computation

```
effectiveness = followed / (followed + contradicted + ignored + irrelevant) * 100
```

Returns null/0 if total evaluations < minimum threshold. The `frecency` package uses a default of 0.5 (50%) for memories with no evaluations.

## Quality-Weighted Scoring

The `frecency.Scorer` computes a combined score:

```
combined = relevance * genFactor * (1 + quality)
quality  = wEff * effectiveness + wFreq * frequency + wTier * tierBoost
```

Where:
- `effectiveness` = followed / total evaluations (0.5 default if no data)
- `frequency` = log(1 + surfaced_count) / log(1 + max_surfaced)
- `tierBoost` = 1.2 for tier A, 0.2 for tier B, 0 for tier C

Default weights: effectiveness 0.3, frequency 1.0, tier 0.3.
