# Memory Effectiveness Plan

How engram evolves from "capture and surface" to "self-correcting memory."

## Current State

UC-1 (learn), UC-2 (surface), UC-3 (correct) are complete. Engram captures
memories in real-time (UserPromptSubmit fast-path + LLM), extracts from
sessions (PreCompact/SessionEnd), and surfaces via keyword matching with
tier-aware anti-pattern filtering.

**What's missing:** No feedback loop. We don't know if surfaced memories
help, hurt, or get ignored.

## Goals

1. **Learn ASAP** — capture memories the moment they appear
2. **Use ASAP** — surface memories when they're relevant
3. **Situational judgment** — agent can override/ignore when context demands
4. **Improve effectiveness** — know which memories help, fix those that don't

## Relationship with Claude Auto-Memories

Claude Code has a built-in auto-memory system (MEMORY.md + topic files). Key
differences from engram:

| Dimension | Auto-Memory | Engram |
|-----------|-------------|--------|
| Write speed | Immediate (model writes in-turn) | Next turn (hook) or end-of-session |
| Retrieval | Always loaded, first 200 lines | Keyword-matched per prompt |
| Quality gate | None (model decides) | LLM classifier with A/B/C tiers |
| Effectiveness | No tracking | Core design goal |
| Scale | 200-line limit | Unlimited, relevance-filtered |

**Strategy: Complementary layers.** Auto-memories handle project context
(architecture notes, file locations, discovered patterns). Engram handles
behavioral corrections (always/never/remember, anti-patterns, principles
requiring precision surfacing and effectiveness tracking). Don't fight
auto-memories — they serve a different purpose.

## Phase 1: Surfacing Instrumentation — issue #46

Add observability to the existing system without changing behavior.

**Per-memory tracking fields:**
- `surfaced_count` — incremented each time the memory is surfaced
- `last_surfaced` — timestamp of most recent surfacing event
- `surfacing_contexts` — recent context types (session-start, prompt, tool)

**Implementation:** Update TOML files in-place during surface events. Read
counts during learn/review phases. No new subcommands — just data collection.

## Phase 2: Automatic Outcome Signal — issue #47

At PreCompact and SessionEnd, automatically assess whether surfaced memories
were followed, contradicted, or ignored during the session.

**Mechanism:**
- PreCompact/SessionEnd hooks already read the transcript
- After running `engram learn`, run a new `engram evaluate` pass
- LLM reviews transcript against memories that were surfaced this session
- Writes outcome to each memory's TOML: `{followed, contradicted, ignored}`
- Accumulates over sessions into effectiveness scores

**Visibility:**
- SessionEnd summary shows which memories were surfaced and their outcomes
- `engram review` command shows effectiveness stats per memory
- Surfacing events include "(surfaced N times, followed M%)" annotations

**Key design:** This is automatic — the user sees outcomes without asking.
The agent sees effectiveness context when memories surface.

## Phase 3: Diagnosis and Action — issues #39-#43

With surfacing counts and outcome signals, populate the effectiveness matrix:

|  | Often Surfaced | Rarely Surfaced |
|--|----------------|-----------------|
| **High Follow-Through** | **Working** — maintain (issue #40) | **Hidden Gem** — broaden triggers (issue #42) |
| **Low Follow-Through** | **Leech** — diagnose and fix (issue #41) | **Noise** — prune (issue #43) |

Foundation: issue #39 (UC-6) provides the tracking framework that
issues #40-#43 consume.

**Diagnosis actions (all user-confirmed):**
- **Leech:** Rewrite content, adjust tier, expand keywords, or escalate to hook
- **Hidden gem:** Broaden keywords/concepts, add alias terms
- **Noise:** Propose removal with evidence (surfacing count, follow rate, age)
- **Working:** Check for staleness (referenced code changed, practices shifted)

## Phase 4: Evolution — issues #37, #38, #44

Once the base memory system is effective, graduate high-performing memories
to more efficient delivery mechanisms:

- **UC-4 (issue #37):** Promote frequently-surfaced memories to skills
  (loaded by context similarity, not keyword matching per prompt)
- **UC-5 (issue #38):** Promote universally-useful principles to CLAUDE.md
  (always loaded, no surfacing cost)
- **UC-12 (issue #44):** Escalate repeatedly-ignored advisory memories to
  blocking hooks (deterministic enforcement)

## Phase 5: Session Continuity — issue #45

Independent of the effectiveness pipeline. Capture task-specific context
(constraints discovered, patterns found, what's been tried) that survives
session boundaries. Different from general memories — this is continuity
for multi-session work.

## What We're Ignoring (For Now)

- **Deterministic tool evolution** — only needed for truly mechanical patterns
  where there's zero situational judgment. Not a priority until we have
  effectiveness data showing which patterns are truly deterministic.
- **Skill generation (UC-4)** — premature optimization. At ~20 memories,
  keyword matching per prompt is fine. Skills matter at 100+ memories.
- **CLAUDE.md management (UC-5)** — manual promotion works at current scale.
  Automate when cross-project effectiveness data exists.

## Bug Fix: Stale Binary — issue #48

The installed binary at `~/.claude/engram/bin/engram` was built before the
`--format json` flag was added. The SessionStart hook uses `--format json`,
so it silently produced no output — memories were not loading at session
start. The session-start hook's rebuild check (`find -newer`) didn't trigger
because file mtimes matched after checkout.

**Fix:** Make the build check more robust — compare binary version hash
against source, not just file modification times.
