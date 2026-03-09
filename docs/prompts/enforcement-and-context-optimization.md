# Prompt: Instruction Adherence & Context Optimization Strategy

## What engram is

Engram is a Claude Code plugin (Go binary + hooks + skills) that gives LLM agents self-correcting memory. It:

1. **Detects and stores memories** — When users correct the agent or teach it something, engram extracts structured memories as TOML files with confidence tiers (A/B/C), keywords, and principles.
2. **Surfaces memories at hook time** — SessionStart, UserPromptSubmit, and PreToolUse hooks retrieve relevant memories and inject them as system-reminder context.
3. **Evaluates effectiveness** — After sessions, an LLM evaluates whether surfaced memories were followed, contradicted, or ignored. This builds per-memory effectiveness scores.
4. **Reviews and maintains** — `engram review` classifies memories into a 2x2 matrix (Working/Leech/Hidden Gem/Noise). `engram maintain` generates proposals: staleness checks, LLM-powered rewrites for leeches, keyword broadening for hidden gems, removal for noise.

Currently all enforcement is **advisory** — memories are surfaced as system-reminder messages. Nothing blocks actions. The agent sees the memory, but may not follow it.

## The broader problem

Engram memories are one layer in a stack of instructions the agent is supposed to follow:

- **CLAUDE.md** — Project-level and global instructions, always loaded
- **Rules** — Instructions scoped globally or to file patterns (e.g., `*.go`), loaded into context when working with matching files
- **Skills** — Context-specific instructions loaded by similarity match when invoked
- **MEMORY.md** — Claude Code's built-in auto-memory system
- **Engram memories** — Structured TOML files surfaced at hook time
- **Hook outputs** — System messages injected by SessionStart, UserPromptSubmit, PreToolUse, etc.

The problem isn't specific to engram memories — it's holistic: **reduce the incidence and severity of violations of ANY instructions the LLM is supposed to follow, and reduce the incidence and severity of forgetting important context in general.**

Engram currently only tracks its own memories' effectiveness. But the user experiences failures across the entire instruction stack: a CLAUDE.md rule ignored, a skill step skipped, a memory contradicted, important context lost after compaction. These all have the same root causes and the same solution space.

### Why this is hard

1. **Context is finite.** Every surfaced memory, hook output, skill, and CLAUDE.md instruction competes for the model's attention window. Adding more instructions has diminishing returns and eventually actively hurts performance by crowding out other important context.

2. **Advisory enforcement has a ceiling.** Some instructions are consistently ignored not because they're bad, but because the model is focused on the immediate task and loses track of meta-requirements. A simple reminder at the right moment is often all it takes — but the current system doesn't know when "the right moment" is.

3. **Complexity breeds non-compliance.** Long, complex skills (like the `traced` specification skill, which is hundreds of lines) contain important instructions that get diluted by sheer volume. The model "knows" it should pressure-test skill updates but doesn't because that instruction is buried in a sea of other instructions.

4. **The enforcement paradox.** Adding more enforcement mechanisms (more hooks, more reminders, more context) to solve non-compliance can itself cause non-compliance by consuming the context budget that was enabling compliance with other instructions.

## What I want to figure out

For instructions that aren't being followed — whether they come from CLAUDE.md, skills, engram memories, or any other source — I want a systematic framework that evaluates six dimensions:

### Dimension 1: LLM hook enforcement
Can the instruction be enforced with an LLM-based hook? For example, a PreToolUse hook that checks whether the proposed action violates the instruction and blocks or warns. This adds some context cost (the hook output) but less than having the full instruction in conversation.

### Dimension 2: Deterministic automation
Can the instruction be fully or partially automated without any LLM involvement? An instruction that says "always run linting before committing" could become a pre-commit hook — zero context cost, 100% reliable. Every instruction that becomes deterministic code frees context AND is perfectly enforced.

### Dimension 3: Context pruning
Can we identify what's currently filling context and proactively prune unnecessary content? This includes: trimming verbose hook outputs, compressing skill instructions, removing redundant CLAUDE.md entries, reducing memory surfacing to only high-risk moments. Can we also prevent unnecessary context from entering in the future (e.g., hooks that output less, skills that are shorter)?

### Dimension 4: Context automation
Can we automate what's currently consuming context via deterministic tooling? For example, if a skill has a 200-line specification but 150 lines are rote procedural steps that could be handled by a CLI tool, the skill could shrink to 50 lines of judgment-requiring instructions plus a "run `tool X` for the mechanical parts" directive.

### Dimension 5: Effectiveness ranking
When multiple approaches exist for a given instruction, which is likely to be most effective? The ranking should consider: reliability (deterministic > LLM hook > advisory), context cost (zero > low > high), maintenance burden (code requires updates, hooks need testing), and user experience (transparent automation > noisy blocking > silent advice).

### Dimension 6: Proactive non-compliance detection
Can we notice when memories, skills, and CLAUDE.md instructions aren't being followed — before the user has to remind us? For example:
- A PostToolUse hook that checks "did the agent just do something a memory says not to do?"
- A Stop hook that reviews the session for missed requirements
- A UserPromptSubmit hook that notices patterns of non-compliance in the conversation so far

**Concrete example:** The `traced` skill clearly states that skill updates must be pressure-tested. But the model frequently skips this step. A simple user reminder ("did you pressure test?") is enough to trigger compliance. This suggests the problem isn't knowledge but salience — the instruction exists but gets lost in the skill's complexity. Possible solutions: (a) a PostToolUse hook that fires after skill file writes and injects a reminder, (b) deterministic tooling that runs pressure tests automatically, (c) simplifying the skill so the instruction stands out more, (d) all of the above.

## What I want as output

1. **A decision framework** — Given an instruction that's being violated (from any source: CLAUDE.md, skill, engram memory, or convention), how do I systematically evaluate which of the six dimensions apply, and in what priority order? This should be a repeatable process, not just a one-time analysis.

2. **An architecture sketch** — How would engram's current architecture need to evolve to support these capabilities? What new components, hooks, or CLI commands are needed? How do they interact with the existing effectiveness pipeline?

3. **Candidate UCs** — Concrete use case definitions (in the style of engram's existing UC-1 through UC-16) for the most impactful capabilities. Each UC should have a clear scope, dependencies on existing UCs, and a rough sense of complexity.

4. **A context budget model** — How do we reason about the total context cost of engram's enforcement? What's the current budget (approximate), what's the theoretical maximum before it hurts performance, and how do the proposed changes affect the budget?

5. **Quick wins** — Which of these dimensions could be addressed with minimal new code, by tweaking existing hooks, skills, or CLAUDE.md instructions? Not everything needs a UC.

6. **Revision of #44** — Issue #44 (Hook Enforcement Lifecycle) was written before this broader analysis. Given the six dimensions above, should #44 be revised, split, or absorbed into a larger effort? What's the right scope?

## Constraints

- Engram is a Claude Code plugin — it can use hooks (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop, PreCompact), skills, rules (global or per-file-type, always injected into context for matching files), CLAUDE.md management, and a Go binary for computation.
- **Rules** are another enforcement surface worth investigating. They can be scoped globally or to specific file patterns (e.g., `*.go`, `*.md`), and are always loaded into context when working with matching files. This makes them a middle ground between CLAUDE.md (always loaded) and skills (loaded on invocation) — targeted context injection without hook overhead.
- Pure Go, no CGO. External LLM calls via Anthropic API when needed.
- DI everywhere in internal packages. I/O at the edges only.
- The goal is to make the agent more effective with LESS context, not more. Any solution that significantly increases context consumption needs to justify itself by freeing more context elsewhere.
- User confirmation required before any automated changes to memories, hooks, skills, CLAUDE.md, or rules.
