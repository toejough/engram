# Adaptive Policy System

**Issue:** #387 (expanded scope)
**Date:** 2026-03-27
**Status:** Approved

## Problem

Engram collects comprehensive feedback (followed, contradicted, ignored, irrelevant) but barely uses it. The extraction prompt is static. Surfacing weights are hardcoded. Maintenance action outcomes are unmeasured. Engram is a feedforward system pretending to be feedback-driven.

Meanwhile, Claude Code shipped auto memories and "dreaming" with zero feedback loops — write-and-hope with no quality measurement. Engram's moat is impact measurement, but the loop isn't closed.

## Solution

A policy file (`policy.toml`) stores learned adaptation policies across three dimensions: extraction, surfacing, and maintenance. Each policy has a directive, rationale, evidence, and effectiveness tracking. Users approve proposals (rationale-first), with per-dimension auto-apply as trust builds.

## Design

### Policy File Format

`policy.toml` lives in the engram data directory alongside config.

```toml
[[policies]]
id = "pol-001"
dimension = "extraction"          # extraction | surfacing | maintenance
directive = "De-prioritize tool-specific mechanical patterns"
rationale = "12 of 15 memories about build-tool commands marked irrelevant across 8 sessions"
evidence = { irrelevant_rate = 0.80, sample_size = 15, sessions_observed = 8 }
status = "proposed"               # proposed | approved | rejected | active | retired
created_at = "2026-03-27T10:00:00Z"
approved_at = ""
effectiveness = { before = 0.0, after = 0.0, measured_sessions = 0 }
```

### Policy Lifecycle

1. **Detect** — feedback analysis identifies a pattern (at extract time)
2. **Propose** — write to `policy.toml` with status `proposed`
3. **Surface** — show in triage output alongside quadrant suggestions
4. **Approve/Reject** — user decides via `/adapt` skill (or auto-applies if dimension has `auto = true`)
5. **Apply** — active policies modify extraction prompt, surfacing weights, or maintenance thresholds
6. **Measure** — track before/after effectiveness over measurement window (default 5 sessions)
7. **Retire** — if measured effectiveness shows no improvement, propose retirement

### Feedback Pattern Analysis

Runs at extract time (piggybacks on `engram extract`). Reads all memories and feedback counters, detects patterns across four categories:

**Content patterns** — cluster memories by keyword/concept overlap. For each cluster, compute aggregate effectiveness and irrelevance rates. Detect "memories about X are consistently irrelevant" or "memories about Y are consistently followed." Produces extraction policies.

**Structural patterns** — correlate tier assignment with actual effectiveness, generalizability score with follow rate, keyword count/breadth with irrelevance rate. Produces extraction or surfacing policies.

**Surfacing patterns** — compare follow rates across different effectiveness/frequency/tier weight regimes. Assess irrelevance penalty half-life calibration. Produces surfacing policies.

**Maintenance outcome patterns** — for memories that were rewritten/refined/broadened, compare effectiveness before vs. after. Track action success rates. Produces maintenance policies.

**Implementation:** Analysis done by Haiku receiving aggregated feedback statistics (not raw memory content). Minimal token cost.

**Minimum data thresholds:** No proposals until sufficient data exists. Configurable minimums (e.g., content cluster needs 5+ memories with 3+ feedback events each).

### Extraction Adaptation

The extraction prompt becomes two-part:

1. **Static base** — existing quality gates, tier definitions, JSON schema. Invariant.
2. **Dynamic policy sections** — injected from active extraction policies. Short paragraphs with directive and condensed rationale.

Example injected guidance:

```
## Learned Extraction Guidance

Based on feedback from this user's memory corpus:

- DE-PRIORITIZE tool-specific mechanical patterns (e.g., "always run targ check",
  "use --flag for X"). These are consistently marked irrelevant (80% irrelevance
  rate across 15 memories). Prefer capturing the *why* behind tool choices.

- PRIORITIZE design rationale and architectural decision context. These have 90%
  follow rate across 12 memories.

- Tier B assignments are underperforming Tier A in this corpus (38% vs 62%
  effectiveness). Be more selective with Tier B — only extract when the correction
  reveals a generalizable principle.
```

**What extraction policies can control:** content categories to prioritize/de-prioritize, tier assignment calibration, keyword quality guidance, generalizability threshold adjustments.

**What they cannot change:** JSON output schema, tier A/B/C definitions, hard rejection rules (common knowledge, ephemeral context).

**Measurement:** Each policy records corpus-wide extraction quality at activation. After N sessions (default 5), compare: did memories extracted under this policy have better follow/irrelevance rates? If not, propose retirement.

### Surfacing Adaptation

Active surfacing policies override hardcoded defaults:

| Parameter | Default | What it controls |
|-----------|---------|-----------------|
| `wEff` | 0.3 | Effectiveness weight in quality score |
| `wFreq` | 1.0 | Frequency weight in quality score |
| `wTier` | 0.3 | Tier boost weight in quality score |
| `tierABoost` | 1.2 | Tier A multiplier |
| `tierBBoost` | 0.2 | Tier B multiplier |
| `irrelevancePenaltyHalfLife` | 5 | Half-life of irrelevance decay |
| `coldStartBudget` | 2 | Max unproven memories per invocation |

Surfacing code reads active policies at startup. Each numeric parameter: policy override > hardcoded default.

**Measurement:** Track corpus-wide follow rate and irrelevance ratio before/after policy activation. After measurement window, compare and propose retirement if no improvement.

### Maintenance Adaptation

Active maintenance policies adjust classification thresholds and action selection:

| Parameter | Default | What it controls |
|-----------|---------|-----------------|
| `effectivenessThreshold` | 50% | High/low effectiveness boundary |
| `flagThreshold` | 40% | Flagged-for-action boundary |
| `minEvaluations` | 5 | Min feedback events before classifying |
| `stalenessThresholdDays` | 90 | When to propose staleness review |
| `refineKeywordsIrrelevanceThreshold` | 60% | Irrelevance ratio to trigger refinement |
| `refineKeywordsMinFeedback` | 5 | Min feedback before refining |

**Maintenance action outcome tracking** — new field on memory TOML records:

```toml
[maintenance_history]
[[maintenance_history.actions]]
action = "rewrite"
applied_at = "2026-03-20T10:00:00Z"
effectiveness_before = 0.25
surfaced_count_before = 12
effectiveness_after = 0.0          # filled after measurement window
surfaced_count_after = 0           # filled after measurement window
measured = false
```

After measurement window, the analysis engine computes action success rates and generates maintenance policies (e.g., "rewrite actions improve effectiveness 60% of the time but keyword refinement only helps 30% — prioritize rewrites").

### User Interface

**Primary interaction through skills**, not raw CLI.

**Triage output (session start)** — adaptation proposals appear alongside quadrant suggestions:

```
## Adaptation Proposals (2 pending)
  1. pol-003 [extraction] — De-prioritize tool-specific mechanical patterns
     Rationale: 80% irrelevance rate across 15 memories
  2. pol-004 [surfacing] — Increase effectiveness weight from 0.3 to 0.5
     Rationale: High-effectiveness memories followed 3x more often

Run /adapt to review proposals or adjust adaptation settings.
```

**The `/adapt` skill** handles: reviewing pending proposals (approve/reject), viewing active policy status, toggling auto mode per dimension, manually adjusting parameter overrides, retiring policies.

**Auto-promotion via approval streaks:**

Track consecutive approvals per dimension:

```toml
[adaptation.approval_streak]
extraction = 2
surfacing = 4
maintenance = 0
```

When a dimension hits 3+ consecutive approvals, offer auto-promotion in triage output — one dimension at a time, highest streak first:

```
You've approved 4 surfacing proposals in a row. Want to make surfacing
adaptation automatic? Run /adapt to toggle auto-apply.
```

Rejecting a proposal resets the streak for that dimension to 0.

### Configuration

In existing config TOML:

```toml
[adaptation]
enabled = true                    # master switch
measurement_window = 5            # sessions before measuring policy effectiveness
min_cluster_size = 5              # minimum memories in a content cluster for analysis
min_feedback_events = 3           # minimum feedback per memory before including

# Per-dimension auto mode
extraction_auto = false
surfacing_auto = false
maintenance_auto = false
```

### Prerequisites

**Bug fix:** Include `IrrelevantCount` in effectiveness denominator. Current formula: `followed / (followed + contradicted + ignored)`. Correct formula: `followed / (followed + contradicted + ignored + irrelevant)`. Without this, effectiveness data is unreliable and every adaptation built on it will be miscalibrated. First commit.

### Strategic Positioning

| Capability | Claude Code | Engram (after this) |
|-----------|-------------|-------------------|
| Memory creation | LLM judgment, no scoring | Impact-measured, tier-classified |
| Quality feedback | None | Followed/contradicted/ignored/irrelevant |
| Adaptation | None | Closed-loop: feedback → analysis → proposals → policy → measurement |
| Transparency | No audit trail | Rationale-first proposals, effectiveness tracking, policy history |
| User control | Delete or keep | Per-dimension auto/manual, streak-based trust gradient |
| Self-correction | Manual editing only | Automatic retirement of policies that don't improve outcomes |

Engram's differentiator: it learns whether it's getting better and adjusts when it isn't. Claude Code can't answer "are my memories helping?" Engram can.
