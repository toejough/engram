# Use Cases

## UC-1: Session Learning

Extract learnings from session transcripts at compaction or session end. The `flush` pipeline reads new transcript content (incremental via offset tracking), sends it to the LLM for extraction, classifies each learning into tiers (A/B/C), deduplicates against the existing corpus using keyword overlap (>50%), and writes surviving learnings as TOML files. Triggered by the async `Stop` hook.

## UC-2: Hook-Time Surfacing

Surface relevant memories when the user submits a prompt or when the agent finishes responding. At `UserPromptSubmit`, engram matches the user message against memory keywords using BM25, ranks by quality-weighted score (effectiveness, frequency, tier), applies generalizability and budget constraints, and injects matching memories into the system context. At `Stop`, engram analyzes the agent's recent output for relevant memories and can block the stop to surface them.

## UC-3: Real-Time Correction

Detect correction signals in user messages during `UserPromptSubmit`. Pattern matching identifies phrases like "remember that...", "don't do X", or "actually, you should...". Detected corrections are classified, enriched with structured fields, and written as new memories immediately -- no need to wait for session end.

## UC-6/14: Session Continuity

Maintain context across session boundaries. The `/recall` skill loads context from previous sessions, with optional query-based search. Budget management ensures context injection stays within token limits. The `SessionStart` hook emits a reminder about `/recall` availability.

## UC-15: Automatic Outcome Signal

Track whether surfaced memories were followed, contradicted, or ignored. The `track` package records outcomes by comparing agent behavior against surfaced advice. Outcome counters (followed_count, contradicted_count, ignored_count, irrelevant_count) are stored directly in each memory's TOML file, making effectiveness a first-class property of every memory.

## UC-16: Unified Maintenance

Diagnose memory health using effectiveness quadrants and propose fixes. The `maintain` pipeline runs at `SessionStart` (background) and classifies memories into four quadrants:

- **Working**: High effectiveness, frequently surfaced. No action needed.
- **Leech**: Frequently surfaced but low effectiveness. Candidates for rewriting or escalation.
- **Hidden Gem**: High effectiveness but rarely surfaced. Keywords need broadening.
- **Noise**: Rarely surfaced and low effectiveness. Candidates for removal.

Proposals include rewrite, broaden_keywords, escalate/de-escalate, remove, and refine_keywords. The `/memory-triage` skill presents proposals interactively.

## UC-20/21: Quality Audit and Escalation

Detect duplicate clusters and enforce graduated escalation. The `signal` package identifies near-duplicate memories via keyword overlap and TF-IDF cosine similarity, proposing consolidation. Escalation tracks memories that are repeatedly contradicted -- enforcement level increases when advice is consistently not followed, ensuring important patterns get stronger presentation.

## UC-34: Memory Consolidation

Detect duplicate clusters across the memory corpus and plan merges. The `signal.consolidate` pipeline computes keyword overlap and TF-IDF similarity between all memory pairs, identifies clusters above a configurable similarity threshold, and optionally confirms via LLM. Merge operations transfer evaluation counters from absorbed memories into the surviving memory's `absorbed` records, preserving outcome history.

## UC-27: Global Binary Installation

Create a global symlink at `~/.local/bin/engram` pointing to the built binary. Fire-and-forget during `SessionStart` — enables `engram` to be called from any shell without PATH manipulation.

## UC-28: Automatic Maintenance

Run `engram maintain` during `SessionStart` (background) as the single source of truth for maintenance signals. Parse output, count proposals by quadrant (Noise, Hidden Gem, Leech) and action type (refine_keywords, escalation, consolidate), check `policy.toml` for pending adaptation proposals, and write `pending-maintenance.json` for consumption at next `UserPromptSubmit`.
