# Changes since `062f127`

This is a substantial rewrite. Everything listed below changed between commit [`062f127`](../../commit/062f127) (2026-04-04) and commit [`cfd5fb5`](../../commit/cfd5fb5) (2026-04-17).

| Area | Before (at `062f127`) | After (at `cfd5fb5`) |
|------|-----------------------|----------------------|
| Surfacing | BM25 scoring on every `UserPromptSubmit`, hook-driven | Skills load context on demand (`/prepare`, `/recall`) |
| Memory file layout | Flat `~/.local/share/engram/memories/*.toml` | Split: `~/.local/share/engram/memory/feedback/*.toml` and `~/.local/share/engram/memory/facts/*.toml` |
| TOML schema | Flat fields: `title`, `content`, `concepts`, `keywords`, `principle`, `anti_pattern`, `confidence`, outcome counters | `schema_version = 2`, `type` discriminator, `source`, `situation`, `[content]` sub-table |
| Outcome tracking | Per-memory counters (`surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`) | Removed — focus moved to situation-query matching |
| Confidence tiers | A / B / C | Removed — replaced by `source = "human"` or `source = "agent"` |
| Adaptation | `/adapt` skill, effectiveness quadrants, proposals | Removed — simpler model, no self-tuning loop |
| Hooks | `Stop` (async extract), `UserPromptSubmit` (surface) | `SessionStart`, `UserPromptSubmit`, `PostToolUse` — reminders only, no surfacing |
| Recall | Always-on injection via BM25 | Three-phase pipeline: auto-memory ranking, skill frontmatter ranking, CLAUDE.md/rules extraction. Haiku filters for relevance. Triggered by `/recall` or `/prepare`. |
