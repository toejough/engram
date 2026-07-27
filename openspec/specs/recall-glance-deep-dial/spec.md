# Glance/deep recall dial Specification

## Purpose

Recall runs at two depths: a cheap, read-only rung (glance) for firing often at everyday decision points, and a full rung (deep) that also crystallizes new lessons into the vault. The glance mode is a read-only pass that applies memory (Steps 0–3.5 and Step 2.7 activation) but skips vault-write steps (Step 2.5C coverage amend/learn, Step 4 synthesis persist), running with ~3 query phrases instead of 10. The glance mode escalates to deep automatically when the decision requires applying a convention that was recently corrected or superseded (recency-channel standards, C5 criteria) — retrieving that a newer convention exists is not enough; the agent must act on it. Implementation and behavior defined in agent-instructions/skills/recall/SKILL.md (Modes section). Why: docs/architecture/adr.md ADR-0004. Validation: dev/eval/LEDGER.md#glance-delivers-c3-c4i-c6; dev/eval/LEDGER.md#glance-fails-c5-delivery; dev/eval/LEDGER.md#glance-cost-realvault.

## Requirements

### Requirement: Glance mode is read-only vault-knowledge pass

Glance SHALL execute Steps 0–3.5 and Step 2.7 (activation) with ~3 query phrases, skipping the vault-write side (Step 2.5C coverage amend/learn, Step 4 synthesis-persist). Glance applies memory to the current decision but does not grow the vault's knowledge.

#### Scenario: Glance skips vault writes
- **WHEN** recall is invoked with `glance` mode
- **THEN** the agent reads chunk and note candidates, applies them to the decision, activates used notes, but does NOT create or amend vault notes via learn/amend

#### Scenario: Glance uses ~3 phrases
- **WHEN** recall is invoked with `glance` mode
- **THEN** Step 1 generates ~3 query phrases (not the 10 used in deep mode), reducing embedding and clustering compute cost

### Requirement: Glance escalates to deep for recency-channel standards

When the decision turns on honoring a standard that surfaced in the recent-activity channel (Channel 2) — a "use X going forward" or "the new convention is Y" item — glance SHALL escalate to deep mode (full write side + 10 phrases) to ensure the new standard is elevated to a requirement.

#### Scenario: C5 escalation on recent convention
- **WHEN** glance retrieves a recent-channel (provenanceRecent) item that states a new convention, and the decision requires applying it
- **THEN** the agent escalates to `deep` mode; deep honors recency-channel standards (4/5 measured, vs glance 0/5)

#### Scenario: C3/C4i/C6 do not trigger escalation
- **WHEN** glance surfaces C3 (apply-conventions), C4i (recency-supersession within matched set), or C6 (abduction/synthesis) — where glance is validated as deep-equivalent
- **THEN** no escalation occurs; glance applies the memory and completes

### Requirement: Glance cost advantage over deep

Glance SHALL run at approximately 2× speed and 46% cost reduction per fire compared to deep, measured on a real-scale vault (2026-06-29).

#### Scenario: Glance per-fire performance
- **WHEN** running glance on a real-scale vault with 400+ notes
- **THEN** per-fire duration and cost are measured at 2.23× faster / 46% cheaper than deep
