---
level: 4
name: learn-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6e3db8fb
---

# C4 — learn-skill (Property/Invariant Ledger)

> Component in focus: **S2-N1-M2 · learn skill**.
> Source files in scope:
> - [../../skills/learn/SKILL.md](../../skills/learn/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E11. R-edges cite the P-list each edge backs.

![C4 learn-skill context diagram](svg/c4-learn-skill.svg)

> Diagram source: [svg/c4-learn-skill.mmd](svg/c4-learn-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-learn-skill.mmd -o architecture/c4/svg/c4-learn-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R-edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n1-m2-p1-user-invoked-trigger"></a>S2-N1-M2-P1 | User-invoked trigger | For all explicit user invocations ("/learn", "remember this", "save that for later", "write up what we just did"), the skill instructs the agent to determine input grain from context — single observation when a moment is flagged, session-batch sweep at the end of a chunk of work. | [skills/learn/SKILL.md:26](../../skills/learn/SKILL.md#L26) | **⚠ UNTESTED** | One of two trigger modes; mode does not gate write behavior. |
| <a id="s2-n1-m2-p2-autonomous-task-boundary-trigger"></a>S2-N1-M2-P2 | Autonomous task-boundary trigger | For all discrete task completions (feature shipped, bug fixed, plan step closed, direction changed), the skill instructs the agent to self-fire and sweep the just-completed work using the same gate sequence and write discipline, with no user prompt before write. | [skills/learn/SKILL.md:27](../../skills/learn/SKILL.md#L27) | **⚠ UNTESTED** | Autonomous writes are not gated by approval; the gates are the gate. |
| <a id="s2-n1-m2-p3-microtask-suppression"></a>S2-N1-M2-P3 | Micro-task auto-fire suppression | For all autonomous trigger evaluations, the skill instructs the agent not to auto-fire on micro-tasks (one-line edits, single-file moves, trivial renames, typo fixes); the threshold is "a chunk of work that could plausibly produce lessons," and when unsure, do not fire. | [skills/learn/SKILL.md:29](../../skills/learn/SKILL.md#L29) | **⚠ UNTESTED** | Default-off bias for ambiguous chunks. |
| <a id="s2-n1-m2-p4-three-gates-ordered-fail-any-drops"></a>S2-N1-M2-P4 | Three gates ordered, fail-any drops | For all candidates, the skill instructs the agent to run gates in order — Recurs → Activity-and-Domain → Knowledge — and drop the candidate on the first failure with no retries or escape hatches (except the single Gate 2 reframe). | [skills/learn/SKILL.md:33](../../skills/learn/SKILL.md#L33) | **⚠ UNTESTED** | Short-circuit semantics. |
| <a id="s2-n1-m2-p5-recurs-strips-to-activity-domain"></a>S2-N1-M2-P5 | Recurs strips to activity+domain | For all candidates, the skill instructs the agent to reduce the situation to activity+domain and fail the candidate at Recurs if it names this project, its internals or architecture, phase numbers, issue IDs, commit hashes, dates, or one-time events. | [skills/learn/SKILL.md:37](../../skills/learn/SKILL.md#L37) | **⚠ UNTESTED** | Cross-project plausibility test — an agent on an unrelated project should plausibly hit the same situation. |
| <a id="s2-n1-m2-p6-activity-and-domain-pre-lesson-framing"></a>S2-N1-M2-P6 | Activity-and-domain pre-lesson framing | For all candidates, the skill instructs the agent to phrase the `situation` field as what an agent would be embarking on, framed as it would be queried before the lesson is known — no hindsight, no diagnosis-as-situation. | [skills/learn/SKILL.md:47](../../skills/learn/SKILL.md#L47) | **⚠ UNTESTED** | Bad/Good table on lines 49–53 illustrates the anti-pattern. |
| <a id="s2-n1-m2-p7-one-reframe-rerun"></a>S2-N1-M2-P7 | One reframe, re-run all gates | For all candidates that fail Gate 2, the skill instructs the agent that it may reframe the situation exactly once and re-run all three gates; if still failing, drop. | [skills/learn/SKILL.md:55](../../skills/learn/SKILL.md#L55) | **⚠ UNTESTED** | Re-run scope is all three gates, not just Gate 2. |
| <a id="s2-n1-m2-p8-knowledge-bar"></a>S2-N1-M2-P8 | Knowledge bar (information vs knowledge) | For all candidates, the skill instructs the agent to pass Gate 3 only if the candidate can be restated as a transferable principle with applicability beyond the originating event — mere descriptions of what happened are information and fail. | [skills/learn/SKILL.md:59](../../skills/learn/SKILL.md#L59) | **⚠ UNTESTED** | Bar drawn from zettelkasten.de; no word counts, no graduation rates. |
| <a id="s2-n1-m2-p9-four-dispositions"></a>S2-N1-M2-P9 | Four dispositions per survivor | For all surviving candidates, the skill instructs the agent to choose one of four dispositions — new permanent, merge into an existing permanent, split into multiple permanents, or new-elaboration as a continuation — preferring new-elaboration over merge when the candidate adds claims the existing permanent doesn't make. | [skills/learn/SKILL.md:80](../../skills/learn/SKILL.md#L80) | **⚠ UNTESTED** | Editing a dated permanent erases the time-shape of the thinking. |
| <a id="s2-n1-m2-p10-luhmann-position-assignment"></a>S2-N1-M2-P10 | Luhmann position assignment, binary computes ID | For all writes, the skill instructs the agent to find the most-related existing note, choose the relation (`top`, `continuation`, or `sibling`), and pass `--target` and `--relation` to the binary so the binary computes the actual ID under a vault lock — the agent does not compute the ID itself. | [skills/learn/SKILL.md:89](../../skills/learn/SKILL.md#L89) | **⚠ UNTESTED** | Three relation values; lock owned by the binary. |
| <a id="s2-n1-m2-p11-llm-voice-body-related-to-rationale"></a>S2-N1-M2-P11 | LLM-voice body with per-link rationale | For all drafted bodies, the skill instructs the agent to write in its own LLM voice (rephrasing verbatim user quotes) and to include `Related to:` bullets where every bullet carries a per-link rationale — no bare wikilinks. | [skills/learn/SKILL.md:111](../../skills/learn/SKILL.md#L111) | **⚠ UNTESTED** | Quality bar reinforced at line 164 ("Per-link rationale"). |
| <a id="s2-n1-m2-p12-moc-on-judgement"></a>S2-N1-M2-P12 | MOC on judgement, no count threshold | For all MOC-creation decisions, the skill instructs the agent to create or update a MOC when a real framing paragraph emerges across notes — judgement, not a cluster count — and to put the framing prose in the body without auto-listing constituents. | [skills/learn/SKILL.md:127](../../skills/learn/SKILL.md#L127) | **⚠ UNTESTED** | Backlinks already enumerate constituents. |
| <a id="s2-n1-m2-p13-calls-engram-promote"></a>S2-N1-M2-P13 | Calls engram promote | For all persistence steps, the skill instructs the agent to invoke `engram promote feedback`, `engram promote fact`, or `engram promote moc` (never direct filesystem writes) with the drafted fields and the body on stdin. | [skills/learn/SKILL.md:102](../../skills/learn/SKILL.md#L102) | **⚠ UNTESTED** | Backs L3 R4. Subcommand set is exactly {feedback, fact, moc}. |
| <a id="s2-n1-m2-p14-parallel-tool-use-block"></a>S2-N1-M2-P14 | Single parallel tool-use block per pass | For all `engram promote` invocations in a single /learn pass, the skill instructs the agent to issue them in one parallel tool-use block — serial writes across tool turns are explicitly forbidden. | [skills/learn/SKILL.md:147](../../skills/learn/SKILL.md#L147) | **⚠ UNTESTED** | "Hard rule" — collapses N tool-roundtrip latencies into one. |
| <a id="s2-n1-m2-p15-contradictions-as-related-to-bullets"></a>S2-N1-M2-P15 | Contradictions as Related-to bullets | For all new permanents that contradict an existing permanent, the skill instructs the agent to write the new permanent with a `Related to:` bullet whose rationale names the discrepancy, surface the contradiction in the final report, and refrain from smoothing. | [skills/learn/SKILL.md:143](../../skills/learn/SKILL.md#L143) | **⚠ UNTESTED** | No silent overwrite; no merge that hides the disagreement. |
| <a id="s2-n1-m2-p16-final-report"></a>S2-N1-M2-P16 | Final report per pass | For all completed passes, the skill instructs the agent to report candidates considered, gates passed/failed (with gate name and one-line reason), permanents written (with Luhmann IDs), MOCs written or updated, and contradictions surfaced. | [skills/learn/SKILL.md:149](../../skills/learn/SKILL.md#L149) | **⚠ UNTESTED** | Same report shape for user-invoked and autonomous passes. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **S2-N1-M2 · learn skill**)
- Siblings:
  - [c4-c4-skill.md](c4-c4-skill.md)
  - [c4-recall-skill.md](c4-recall-skill.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
