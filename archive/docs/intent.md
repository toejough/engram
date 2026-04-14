# Engram: Intent & Rationale

What engram is, why it exists, and what must survive any rewrite.

## Core Hypothesis

**The value of a memory is not how often it surfaces, but whether it improves outcomes when surfaced.**

LLM agents make the same mistakes repeatedly. Users correct them, but corrections are lost at session end. Instruction files (CLAUDE.md, rules, skills) accumulate guidance, but nobody knows if it's working. An instruction surfaced 1000 times but never followed is a leech, not a success. An instruction that only matches narrow keywords goes unseen when needed.

Engram's hypothesis: if you **measure impact** (did the agent follow this memory when it was surfaced?), **diagnose failures** (why isn't it being followed?), and **surface at the right moment** (situational matching, not just keywords), you can make agents get to the right answer faster, cheaper, and with less repetition from the human.

## What Engram Does

Engram is a self-correcting memory system for LLM agents. It:

1. **Records** what the agent learned (corrections, facts, patterns) as structured memories
2. **Surfaces** relevant memories at the right moment, before the agent acts
3. **Measures** whether surfaced memories actually improved outcomes
4. **Diagnoses** why memories aren't working and proposes fixes
5. **Pushes** memories to agents asynchronously (they don't have to ask)

## Memory Formats

### Feedback Memories: SBIA

Feedback memories capture behavioral corrections in **Situation / Behavior / Impact / Action** format.

**Why SBIA?** The choice is grounded in cognitive psychology research:

- **Encoding Specificity Principle** (Tulving & Thomson, 1973): Memory retrieval is most effective when the retrieval cue matches the encoding context. By anchoring corrections to the *situation* where they apply (not just topic keywords), engram uses situational context as the retrieval cue.

- **Case-Based Reasoning** (Kolodner 1993, Aamodt & Plaza 1994): AI systems that retrieve past solutions by matching *problem situations* consistently outperform those matching on abstract rules. SBIA enables case-based retrieval.

- **SBI/SBIA Model** (Center for Creative Leadership): One of the most validated feedback frameworks in organizational psychology. Power comes from anchoring feedback to *observable behavior in a specific situation*, making it actionable rather than abstract.

- **Situated Cognition** (Brown, Collins, Duguid 1989): Knowledge abstracted from its situation of use is harder to apply. SBIA preserves situational grounding.

**The gap it fills:** Prior systems stored *what to do* (principle) and *what not to do* (anti-pattern), but not *when* the correction applies. Keywords alone fail because they're topic-level matches, not scenario descriptors. A memory about "temp file cleanup" should surface when writing Go file operations, not when discussing temperature.

**Example:**

```toml
situation = "writing Go code that creates temporary files or performs file operations"
[content]
behavior = "using predictable temp file names, not cleaning up on failure paths"
impact = "security vulnerabilities, resource leaks, untestable code"
action = "use os.CreateTemp, add defer cleanup on failure paths, inject I/O for testability"
```

### Fact Memories: SPO

Fact memories capture propositional knowledge in **Subject / Predicate / Object** triples.

**Why SPO?** Facts are informational context, not behavioral corrections. They need a different structure because they answer "what is true?" not "what should I do differently?" SPO triples are the simplest structured representation for declarative knowledge — they're what knowledge graphs use.

**Example:**

```toml
situation = "referencing engram chat file paths"
[content]
subject = "engram chat files"
predicate = "stored-in"
object = "~/.local/share/engram/chat/<project-slug>.toml"
```

## Impact Measurement

Every memory tracks four outcome counters:

| Counter | Meaning |
|---------|---------|
| `surfaced_count` | How many times this memory was shown to an agent |
| `followed_count` | Agent acted consistently with the memory |
| `not_followed_count` | Agent contradicted or ignored the memory |
| `irrelevant_count` | Memory was surfaced but didn't apply to the situation |

**Effectiveness** = followed / (followed + not_followed + irrelevant)

These counters enable the **effectiveness quadrant model**:

| | High effectiveness | Low effectiveness |
|--|---|---|
| **Often surfaced** | **Working** -- keep | **Leech** -- diagnose & rewrite |
| **Rarely surfaced** | **Hidden Gem** -- broaden retrieval | **Noise** -- remove |

### Why Diagnose, Not Delete

A memory surfaced often but not followed is a *leech* — but leeches are *fixable*, not disposable. Four possible fixes:

1. **Content quality issue** -- the memory is poorly written. Rewrite it.
2. **Wrong scope** -- the memory applies narrowly but surfaces broadly. Narrow the situation.
3. **Enforcement gap** -- the memory describes something that should be deterministic. Convert to a hook.
4. **Retrieval mismatch** -- the memory surfaces in wrong contexts. Refine the situation field.

This is the core of "content quality > mechanical sophistication" — don't build better tier-movement machinery, fix the content.

## Confidence Tiers

Memories have confidence based on their source:

- **Tier A** (1.0): Explicit user instruction ("always", "never", "remember this"). Highest surfacing priority.
- **Tier B** (0.7): Observed correction from user behavior or external source. Medium priority.
- **Tier C** (0.4): Contextual fact or pattern inference. Lower priority.

Confidence affects ranking but not presence — even low-confidence memories can surface if they're the best BM25 match.

## Async Push: Why It Matters

In prior iterations, agents had to actively poll for memories. This failed because:

1. Agents forgot to poll
2. Polling added latency to every interaction
3. Agents misunderstood the polling model and abandoned it

The server-driven model with async push via MCP channels solves this: memories arrive in the agent's context *between turns* without the agent doing anything. For time-critical surfacing (before the agent acts), the agent explicitly calls `engram_intent` — but routine memories flow automatically.

## Key Capabilities

### What Must Survive Rewrites

These are the properties of the system, not implementation details:

1. **Structured memory format** — SBIA for feedback, SPO for facts, with a `situation` field on both
2. **Impact measurement** — surfaced/followed/not_followed/irrelevant counters per memory
3. **Effectiveness quadrant diagnosis** — working/leech/hidden-gem/noise classification with actionable fixes
4. **Situational retrieval** — memories matched by situation context, not just keywords
5. **Async push** — memories surfaced to the agent without the agent requesting them
6. **Synchronous intent** — agent can explicitly ask "what should I know before doing X?"
7. **Structured learning** — agent can report what it learned in the memory format
8. **Observable chat log** — all inter-agent communication in a persistent, readable log file
9. **Server-mediated routing** — deterministic code handles routing, not unreliable LLM agents
10. **Self-correction** — the system improves its own memories based on measured outcomes

### What Can Change

These are implementation choices that could be replaced:

- Go as the implementation language
- TOML as the serialization format
- BM25 as the retrieval algorithm (could add embeddings)
- File-per-memory storage (could use a database)
- HTTP as the API transport (could use gRPC, Unix sockets)
- MCP as the push mechanism (could use any async protocol)
- Claude Code as the host (could integrate with other agent frameworks)

## Lessons from Predecessor Systems

These lessons (from `archive/lessons.md`) shaped engram's design:

**What worked:**
- Hybrid search from complementary signals (BM25 exact + semantic enrichment)
- Structured extraction at write-time (LLM classifies into SBIA/SPO at creation, not retrieval)
- Confidence tiers based on source visibility
- Deterministic hashing for change detection
- End-of-session evaluation at natural boundaries

**What failed (and why engram rejects them):**
- **Count-based promotion** (5 retrievals = promote) — frequency is not quality. Engram measures impact.
- **Mechanical synthesis** (auto-generate keywords from frequency) — produces noise. Engram uses situational context.
- **Append-only instruction files** — grow past budget. Engram manages lifecycle via quadrants.
- **Over-engineered tier movement** (12+ thresholds) — fix the content, not the machinery.
- **No impact tracking** — retrieved-often was tracked, improved-outcomes-when-retrieved was not. Engram's core purpose.
- **Nil dependencies at runtime** — interfaces wired but never connected. Engram uses DI with smoke tests.

## Design Principles

These principles guide implementation decisions:

1. **Content quality > mechanical sophistication** — fix the memory, not the retrieval machinery
2. **Measure impact, not frequency** — surfaced_count is vanity; followed_count is value
3. **Diagnose, don't discard** — leeches are fixable, noise is removable, but know which is which
4. **DI everywhere** — all I/O injected, testable without integration tests
5. **Server-side intelligence, thin clients** — deterministic routing in the server, not unreliable agents
6. **Observable by default** — chat log, debug log, structured JSON logging. If you can't see it, you can't fix it.
7. **Retire immediately** — dead code and deprecated interfaces are liabilities
8. **Plugin form factor** — skills for behavior, binary for computation, hooks for determinism
