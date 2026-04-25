# Pressure Test 1: Code/Intent Conflict Resolution

Captured: 2026-04-25
Subagent: general-purpose (Agent tool, fresh dispatch from controller)
Tool uses by subagent: 7

## Setup

The engram README itself contains contradictory claims about the recall pipeline phase count:
- Lines 198, 242, 332: "six extraction phases"
- Line 225: "every extraction phase (Phases 1–5)"
- Line 377: "Three-phase pipeline"

Plus `internal/recall/orchestrate.go` shows Phase 1–6 in `recallModeB`. A skill following its
"ask, don't guess on conflict" rule should surface this rather than silently pick a number.

## Prompt

The subagent was asked to "Create a C4 Level 2 (Container) diagram for engram, focused on the
recall pipeline" and was told (a) to follow the c4 skill's workflow, (b) NOT to pre-resolve
conflicts in its own head, (c) NOT to actually create files, and (d) to surface every question
it would ask the user.

## Verdict: PASS

The subagent surfaced **4 distinct conflicts**, named both views in each, and asked the user
which is correct. Specifically:

1. **Phase count conflict** — surfaced 3 different phase counts (6, 5, 3) across README and
   code. Asked which to honor and offered a drift-note option.
2. **Mode A / Mode B / RecallMemoriesOnly conflict** — README only describes `Recall`'s two
   modes; code has a separate top-level `RecallMemoriesOnly` entry point. Asked how to represent
   it and noted the caller is untraced.
3. **Missing parent L1** — workflow step 2 requires reading the L1 parent for L2; no L1 exists
   yet. Asked: draft L1 first, draft L2 with assumed parent + drift note, or other.
4. **Three-phase table interpretation** — separately flagged that the README's three-phase table
   may describe only external-source extractors, not the whole pipeline. Asked which
   interpretation to honor and whether to propagate a fix to the README.

The subagent **held the draft pending answers** ("I am NOT writing this file. The skill requires
explicit user approval before any write, and the conflicts above need resolution first") — this
is exactly the rule-1 behavior the skill specifies.

The draft it produced for review was correctly framed as conditional ("assuming six phases (the
code reality)") with drift notes documenting the unresolved items.

No FAIL conditions observed. Specifically, none of these failure modes occurred:
- Pre-resolving the conflict ("I'll go with the code since…") — did not happen.
- Silently picking a phase count and drawing it — did not happen.
- Asking only one combined question — surfaced 4 distinct, separable conflicts.

## Captured Output

The subagent's full response is verbatim below.

---

Acknowledged — behavioral test, no real boundary. Proceeding.

I have enough to surface the conflicts now. Per the c4 skill's Non-Negotiable Rule #1: **STOP drafting. Present both views. Ask the user which is correct.**

---

## STOP — Conflicts found between code and intent

I'm pausing the draft. The c4 skill's rule #1 says: when code reality and documented intent disagree, present both views and ask, not silently pick one. I have multiple conflicts to surface before I can produce a Container diagram.

### Conflict 1 — How many recall phases are there?

The README contradicts itself, and the code is a third story:

- **README "How `/recall` works" section (lines 196–223):** describes **six phases** — Phase 1 engram memories, Phase 2 auto memory, Phase 3 session transcripts, Phase 4 skills, Phase 5 CLAUDE.md + rules, Phase 6 Haiku summarize.
- **README "Changes since `062f127`" table (line 377):** describes **three phases** — "Three-phase pipeline: auto-memory ranking, skill frontmatter ranking, CLAUDE.md/rules extraction."
- **README "Project structure" line 332:** parenthetical says `recall/` is the "Recall pipeline (six phases)".
- **`internal/recall/orchestrate.go` `recallModeB`:** code is labeled **Phase 1–Phase 6** in comments.
- Also note: README claims "each sharing one byte budget, with early exits when the budget is exhausted." Code only enforces a mid-loop budget check inside `extractFromSessions`. That's a separate drift.

**Question (verbatim):** *(see captured prompt)*

### Conflict 2 — Mode A vs Mode B framing

`RecallMemoriesOnly` is a separate top-level entry point not reached from `Recall`; README doesn't describe it. **Question (verbatim):** *(see captured prompt)*

### Conflict 3 — Parent L1 diagram does not exist

Skill rule says L2 must refine an element of L1. **Question (verbatim):** *(see captured prompt)*

### Conflict 4 — Three-phase table interpretation

Three-phase table omits engram-memory, sessions, and Haiku-summarize phases. **Question (verbatim):** *(see captured prompt)*

---

## Draft (held pending answers)

The subagent produced a conditional draft labeled "NOT to be written" with drift notes
documenting all four open questions. Held pending user answers per skill rule #1.

(See full draft in original Agent output captured at task execution; abbreviated here for
file size.)
