---
level: 4
name: learn-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — learn-skill (Property/Invariant Ledger)

> Component in focus: **E11 · learn skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/learn/SKILL.md](../../skills/learn/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E11. R-edges cite the P-list each edge backs.

![C4 learn-skill context diagram](svg/c4-learn-skill.svg)

> Diagram source: [svg/c4-learn-skill.mmd](svg/c4-learn-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-learn-skill.mmd -o architecture/c4/svg/c4-learn-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-five-step-flow"></a>P1 | Five-step flow | For all invocations of the learn skill, the body instructs the agent to execute five numbered steps in order: identify learnable moments, apply the quality gate, draft memories, persist, and handle results. | [skills/learn/SKILL.md:12](../../skills/learn/SKILL.md#L12) | **⚠ UNTESTED** | No behavioral test under skills/learn/. |
| <a id="p2-four-candidate-sources"></a>P2 | Four candidate sources | For all session reviews, the skill instructs the agent to consider exactly four candidate sources — user corrections, failed approaches, discovered facts, and recurring patterns — when identifying learnable moments. | [skills/learn/SKILL.md:16](../../skills/learn/SKILL.md#L16) | **⚠ UNTESTED** | Closed list; not a brainstorm prompt. |
| <a id="p3-ordered-quality-gate"></a>P3 | Ordered quality gate | For all candidate memories, the skill instructs the agent to evaluate the three quality gates (Recurs, Actionable, Right home) in order, dropping the candidate on the first failure. | [skills/learn/SKILL.md:24](../../skills/learn/SKILL.md#L24) | **⚠ UNTESTED** | Short-circuit semantics — no gate is skipped, no gate runs after a failure. |
| <a id="p4-recurs-strips-to-activity-domain"></a>P4 | Recurs strips to activity+domain | For all candidates, the skill instructs the agent to reduce the situation to activity+domain and reject any candidate whose stripped form names this project, its internals, phase numbers, issue IDs, commit hashes, dates, or one-time events. | [skills/learn/SKILL.md:26](../../skills/learn/SKILL.md#L26) | **⚠ UNTESTED** | Cross-project plausibility test. |
| <a id="p5-actionable-requires-concrete-action"></a>P5 | Actionable requires concrete action | For all candidates, the skill instructs the agent to reject memories that do not name a concrete action — vague observations, inert facts, and raw debug logs all fail. | [skills/learn/SKILL.md:37](../../skills/learn/SKILL.md#L37) | **⚠ UNTESTED** | Memory must change agent behavior, not merely describe. |
| <a id="p6-right-home-verification"></a>P6 | Right-home verification | For all surviving candidates, the skill instructs the agent to name an alternative home (code, doc, skill, CLAUDE.md, .claude/rules/*.md, or docs/ spec) and verify by running git log over the last 14 days and reading the listed files before deciding whether to persist a memory. | [skills/learn/SKILL.md:41](../../skills/learn/SKILL.md#L41) | **⚠ UNTESTED** | git log command is exact — `git log --since='14 days ago' --name-only`. |
| <a id="p7-surfacing-failure-routing"></a>P7 | Surfacing-failure routing | For all candidates whose alternative home contains the lesson but did not surface in time this session, the skill instructs the agent to persist a new memory rather than defer to the home, because surfacing — not authorship — is the failure to repair. | [skills/learn/SKILL.md:56](../../skills/learn/SKILL.md#L56) | **⚠ UNTESTED** | Reading the home during verification does not count as surfacing. |
| <a id="p8-memory-kind-routing"></a>P8 | Memory-kind routing | For all surviving candidates, the skill instructs the agent to draft corrections and failed approaches as feedback (SBIA shape) and discovered facts and patterns as fact (situation + subject/predicate/object). | [skills/learn/SKILL.md:63](../../skills/learn/SKILL.md#L63) | **⚠ UNTESTED** | Two memory kinds, deterministic mapping from candidate source. |
| <a id="p9-calls-engram-learn"></a>P9 | Calls engram learn | For all persistence steps, the skill instructs the agent to invoke `engram learn feedback` or `engram learn fact` (no other subcommand) with the drafted fields. | [skills/learn/SKILL.md:79](../../skills/learn/SKILL.md#L79) | **⚠ UNTESTED** | Backs L3 R4. |
| <a id="p10-activity-domain-situation-field"></a>P10 | Activity+domain situation field | For all drafted memories, the skill instructs the agent to phrase the `situation` field as activity+domain matching how a /prepare query would be phrased before the lesson is known — not as the diagnosis, symptom, or fix. | [skills/learn/SKILL.md:67](../../skills/learn/SKILL.md#L67) | **⚠ UNTESTED** | Hindsight-baking is the named anti-pattern. |
| <a id="p11-autonomous-default-at-boundaries"></a>P11 | Autonomous default at boundaries | For all task-boundary invocations (not user-initiated /learn), the skill instructs the agent to persist memories autonomously with `--source agent` rather than presenting findings for approval. | [skills/learn/SKILL.md:76](../../skills/learn/SKILL.md#L76) | **⚠ UNTESTED** | Interactive approval is reserved for explicit /learn invocations. |
| <a id="p12-interactive-approval-on-learn"></a>P12 | Interactive approval on /learn | For all user-initiated /learn invocations, the skill instructs the agent to present drafted findings to the user for approval before persisting. | [skills/learn/SKILL.md:77](../../skills/learn/SKILL.md#L77) | **⚠ UNTESTED** | Reciprocal of P11. |
| <a id="p13-duplicate-broadens-situation"></a>P13 | DUPLICATE broadens situation | For all DUPLICATE responses from `engram learn`, the skill instructs the agent to diagnose why /recall or /prepare missed the existing memory and broaden its situation via `engram update --name <name> --situation "..."`, never dismissing the candidate. | [skills/learn/SKILL.md:87](../../skills/learn/SKILL.md#L87) | **⚠ UNTESTED** | Surfacing failure, not authorship failure — the existing memory is widened. |
| <a id="p14-contradiction-mode-split"></a>P14 | CONTRADICTION mode-split | For all CONTRADICTION responses, the skill instructs the agent to ask the user to resolve (update / replace / keep both) when interactive, and to skip the candidate when autonomous. | [skills/learn/SKILL.md:88](../../skills/learn/SKILL.md#L88) | **⚠ UNTESTED** | Autonomous mode never silently overwrites contradictory memories. |
| <a id="p15-created-confirmation-only-when-interactive"></a>P15 | CREATED confirmation only when interactive | For all CREATED responses, the skill instructs the agent to confirm to the user only in interactive mode; autonomous runs treat CREATED as silent success. | [skills/learn/SKILL.md:86](../../skills/learn/SKILL.md#L86) | **⚠ UNTESTED** | Keeps autonomous task-boundary persistence non-chatty. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E11 · learn skill**)
- Siblings:
  - [c4-c4-skill.md](c4-c4-skill.md)
  - [c4-migrate-skill.md](c4-migrate-skill.md)
  - [c4-prepare-skill.md](c4-prepare-skill.md)
  - [c4-recall-skill.md](c4-recall-skill.md)
  - [c4-remember-skill.md](c4-remember-skill.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

