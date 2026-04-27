---
level: 4
name: prepare-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 035a717d
---

# C4 — prepare-skill (Property/Invariant Ledger)

> Component in focus: **S2-N1-M1 · prepare skill**.
> Source files in scope:
> - [skills/prepare/SKILL.md](skills/prepare/SKILL.md)

## Context (from L3)

E10 prepare skill is a markdown skill body Claude Code loads on /prepare or auto-trigger before new work begins. It contains no executable code: the body is a three-step instruction sequence the agent follows in its own context. Step 1 tells the agent to analyze the current conversation for what work is about to happen. Step 2 tells the agent to issue 2–3 targeted `engram recall --query "<topic>"` invocations described by task (not by fear). Step 3 tells the agent to summarize results back to the user. The skill's only outbound dependency is R2 (prepare skill → engram CLI binary, E9): the body's example bash lines instruct the agent to spawn `engram recall` subprocesses. R1 (Claude Code → Skills) is the inbound load edge from E3. The skill has no DI surface and no Go code — its "contract" is its prose, and every property below is enforced by the SKILL.md text itself.

![C4 prepare-skill context diagram](svg/c4-prepare-skill.svg)

> Diagram source: [svg/c4-prepare-skill.mmd](svg/c4-prepare-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-prepare-skill.mmd -o architecture/c4/svg/c4-prepare-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

**Legend:**
- **Focus** — yellow (E10 prepare skill).
- **Component** — light blue (sibling skills not shown; E9 engram CLI binary as call target).
- **External** — grey (E3 Claude Code, the loader).
- **R-edges** — solid; no DI back-edges (the skill is pure markdown — no Go DI surface).

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n1-m1-p1-skill-name-matches-directory"></a>S2-N1-M1-P1 | Skill name matches directory | For all loads of this skill by Claude Code, the YAML front-matter `name` field equals `prepare`, matching the parent directory name `skills/prepare/` and the slash-command `/prepare`. | [skills/prepare/SKILL.md:2](../../skills/prepare/SKILL.md#L2) | **⚠ UNTESTED** | Architectural: Claude Code routes /prepare to the skill whose front-matter name is `prepare`. Mismatch breaks the slash-command binding. |
| <a id="s2-n1-m1-p2-trigger-description-names-work-start-situations"></a>S2-N1-M1-P2 | Trigger description names work-start situations | For all readers of the front-matter `description`, the text enumerates concrete work-start situations (starting new work, switching tasks, beginning a feature, changing direction, tackling an issue) so Claude Code's auto-trigger heuristic fires before — not after — implementation effort. | [skills/prepare/SKILL.md:3](../../skills/prepare/SKILL.md#L3) | **⚠ UNTESTED** | Architectural: skill triggering is description-driven; vague triggers fail to fire at the intended boundaries. |
| <a id="s2-n1-m1-p3-three-step-flow-shape"></a>S2-N1-M1-P3 | Three-step flow shape | For all invocations of /prepare, the body presents exactly three flow steps in order: (1) analyze the situation, (2) make targeted recall queries, (3) present briefing to user. | [skills/prepare/SKILL.md:14](../../skills/prepare/SKILL.md#L14), [:22](../../skills/prepare/SKILL.md#L22), [:41](../../skills/prepare/SKILL.md#L41) | **⚠ UNTESTED** | Order matters: analysis must precede query construction, and queries must precede the briefing. |
| <a id="s2-n1-m1-p4-bounded-query-count"></a>S2-N1-M1-P4 | Bounded query count | For all invocations following the skill, Step 2 instructs the agent to issue between 2 and 3 targeted `engram recall` queries — not zero, not a flood. | [skills/prepare/SKILL.md:23](../../skills/prepare/SKILL.md#L23) | **⚠ UNTESTED** | Bounds prevent both no-context starts and context-budget exhaustion before work begins. |
| <a id="s2-n1-m1-p5-recall-invocation-form"></a>S2-N1-M1-P5 | Recall invocation form | For all example query lines in Step 2, the bash command shape is `engram recall --query "<topic>"` (single subcommand `recall` with the `--query` flag), matching the engram CLI binary's documented recall surface. | [skills/prepare/SKILL.md:26](../../skills/prepare/SKILL.md#L26), [:27](../../skills/prepare/SKILL.md#L27) | **⚠ UNTESTED** | Drift here would teach the agent to call a non-existent flag or subcommand. Mirrors R2 in c3-skills.md. |
| <a id="s2-n1-m1-p6-query-by-task-discipline"></a>S2-N1-M1-P6 | Query-by-task discipline | For all guidance about query phrasing, the skill instructs the agent to query by the task being undertaken ("what are you trying to do") rather than by anticipated failure modes ("what might go wrong"). | [skills/prepare/SKILL.md:30](../../skills/prepare/SKILL.md#L30), [:32](../../skills/prepare/SKILL.md#L32) | **⚠ UNTESTED** | Memory situations are stored task-shaped; fear-shaped queries miss because the cosine match space disagrees. |
| <a id="s2-n1-m1-p7-positive-and-negative-examples-present"></a>S2-N1-M1-P7 | Positive and negative examples present | For all readers, the skill provides both positive query examples (e.g., "implementing Claude Code hooks", "writing Go tests in [domain]", "git push workflow") and at least one DON'T example demonstrating the failure mode ("common mistakes when writing hooks"). | [skills/prepare/SKILL.md:35](../../skills/prepare/SKILL.md#L35), [:38](../../skills/prepare/SKILL.md#L38) | **⚠ UNTESTED** | Pairing concrete dos with a contrasting don't materially raises adherence vs. abstract guidance alone. |
| <a id="s2-n1-m1-p8-briefing-surfaces-context-to-user"></a>S2-N1-M1-P8 | Briefing surfaces context to user | For all invocations, Step 3 instructs the agent to summarize the relevant recalled context and memories back to the user before proceeding with the work itself. | [skills/prepare/SKILL.md:42](../../skills/prepare/SKILL.md#L42) | **⚠ UNTESTED** | Without an explicit user-facing briefing the user has no chance to correct trajectory before implementation begins. |
| <a id="s2-n1-m1-p9-no-direct-i-o-in-skill-body"></a>S2-N1-M1-P9 | No direct I/O in skill body | For all behaviors prescribed by this skill, no step instructs the agent to read or write filesystem state, network endpoints, or memory storage directly — every external effect routes through the engram CLI binary (E9) via R2. | [skills/prepare/SKILL.md:25](../../skills/prepare/SKILL.md#L25) | **⚠ UNTESTED** | Architectural: matches the project DI principle (skills are behavior, the binary owns I/O). Any future drift here would create a second I/O surface to keep in sync. |
| <a id="s2-n1-m1-p10-read-only-with-respect-to-memory-store"></a>S2-N1-M1-P10 | Read-only with respect to memory store | For all invocations following the skill, the only engram subcommand invoked is `recall`; the skill never instructs the agent to call `engram learn`, `engram update`, or any mutating subcommand. | [skills/prepare/SKILL.md:26](../../skills/prepare/SKILL.md#L26), [:27](../../skills/prepare/SKILL.md#L27) | **⚠ UNTESTED** | Mutation belongs to remember/learn/migrate skills (E11/E13/E14). Prepare is a read-only context-loading boundary. |
| <a id="s2-n1-m1-p11-front-matter-parses-as-yaml"></a>S2-N1-M1-P11 | Front-matter parses as YAML | For all loads of skills/prepare/SKILL.md by Claude Code, the leading `---`-delimited block parses as YAML and exposes both `name` and `description` keys. | [skills/prepare/SKILL.md:1](../../skills/prepare/SKILL.md#L1), [:8](../../skills/prepare/SKILL.md#L8) | **⚠ UNTESTED** | Architectural: Claude Code's skill loader requires valid YAML front-matter. Malformed YAML silently disables the skill. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **S2-N1-M1 · prepare skill**)
- Siblings:
  - [c4-c4-skill.md](c4-c4-skill.md)
  - [c4-learn-skill.md](c4-learn-skill.md)
  - [c4-migrate-skill.md](c4-migrate-skill.md)
  - [c4-recall-skill.md](c4-recall-skill.md)
  - [c4-remember-skill.md](c4-remember-skill.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

