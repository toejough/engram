---
level: 4
name: c4-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — c4-skill (Property/Invariant Ledger)

> Component in focus: **E15 · c4 skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/c4/SKILL.md](../../skills/c4/SKILL.md)
> - [../../skills/c4/tests/baseline-output-no-skill.md](../../skills/c4/tests/baseline-output-no-skill.md)
> - [../../skills/c4/tests/baseline-output-with-skill.md](../../skills/c4/tests/baseline-output-with-skill.md)
> - [../../skills/c4/tests/pressure-conflict.md](../../skills/c4/tests/pressure-conflict.md)
> - [../../skills/c4/tests/pressure-propagation.md](../../skills/c4/tests/pressure-propagation.md)
> - [../../skills/c4/tests/pressure-untested-property.md](../../skills/c4/tests/pressure-untested-property.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E15. The c4 skill is the
exception in the Skills container — it does not call the engram CLI binary. Its only L3 edge is
R1 (loaded by Claude Code).

![C4 c4-skill context diagram](svg/c4-c4-skill.svg)

> Diagram source: [svg/c4-c4-skill.mmd](svg/c4-c4-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-c4-skill.mmd -o architecture/c4/svg/c4-c4-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-four-subactions"></a>P1 | Four sub-actions | For all `/c4` invocations, the skill dispatches by intent into exactly four sub-actions: `create`, `update`, `review`, `audit`. | [skills/c4/SKILL.md:17](../../skills/c4/SKILL.md#L17) | **⚠ UNTESTED** | Tests cover create paths only. |
| <a id="p2-ask-on-conflict"></a>P2 | Ask on code/intent conflict | For all detected conflicts between code reality and intent (docs / commit bodies / session memory), the skill instructs the agent to STOP, present both views, and ask the user — never silently pick one. | [skills/c4/SKILL.md:26](../../skills/c4/SKILL.md#L26) | [skills/c4/tests/pressure-conflict.md](../../skills/c4/tests/pressure-conflict.md) | Pressure-tested. |
| <a id="p3-never-invent-tests"></a>P3 | Never invent test pointers | For all L4 properties whose tests do not exist, the skill instructs the agent to mark **⚠ UNTESTED**, never fabricate a test link or omit the property. | [skills/c4/SKILL.md:28](../../skills/c4/SKILL.md#L28) | [skills/c4/tests/pressure-untested-property.md](../../skills/c4/tests/pressure-untested-property.md) | Pressure-tested. |
| <a id="p4-per-file-approval"></a>P4 | Per-file approval for non-target edits | For all edits to files other than the explicit target (i.e., JSON edits to peer/parent specs), the skill instructs the agent to present each as a proposal with `[a]pply`/`[s]kip`/`[d]efer`, never silently edit. Idempotent rebuilds of auto-generated sections do not count as edits. | [skills/c4/SKILL.md:29](../../skills/c4/SKILL.md#L29) | **⚠ UNTESTED** | Rule 3 reconciliation governs the carve-out. |
| <a id="p5-mermaid-convention"></a>P5 | Project mermaid convention | For all generated diagrams, the skill instructs the agent to emit a `classDef` block at the top with `:::person / :::external / :::container / :::component` classes per `references/mermaid-conventions.md`. | [skills/c4/SKILL.md:32](../../skills/c4/SKILL.md#L32) | [skills/c4/tests/baseline-output-with-skill.md](../../skills/c4/tests/baseline-output-with-skill.md) | Behavioral RED/GREEN baseline. |
| <a id="p6-cross-link-in-body"></a>P6 | Cross-links live in body | For all generated files, the skill instructs the agent to name parent and children directly with relative paths in the file body — no central index file. | [skills/c4/SKILL.md:35](../../skills/c4/SKILL.md#L35) | [skills/c4/tests/baseline-output-with-skill.md](../../skills/c4/tests/baseline-output-with-skill.md) | Baseline asserts L1→L2 cross-links. |
| <a id="p7-ids-on-everything"></a>P7 | IDs on every element/edge | For all L1–L3 diagrams, every catalog row gets `E<n>`, every relationships row gets `R<n>`, the same IDs appear in mermaid node and edge labels, every node has a `click NODE href "#anchor"`, and catalog/relationships rows carry HTML anchors. L4 ledgers use `P<n>` instead. | [skills/c4/SKILL.md:37](../../skills/c4/SKILL.md#L37) | **⚠ UNTESTED** | ID-mismatch is a `review`/`audit` finding. |
| <a id="p8-propagation-sweep"></a>P8 | Propagation sweep on every change | For all `create` and `update` runs, the skill instructs the agent to update parent `cross_links.refined_by`, rebuild siblings, walk children for carry-over drift, and run `targ c4-audit` + `targ c4-registry` — capturing intentional gaps as Drift Notes. | [skills/c4/SKILL.md:162](../../skills/c4/SKILL.md#L162) | [skills/c4/tests/pressure-propagation.md](../../skills/c4/tests/pressure-propagation.md) | Pressure-tested. |
| <a id="p9-targ-build-targets"></a>P9 | Use targ build targets | For all L1 and L3 work, the skill instructs the agent to author a JSON spec, run `targ c4-l*-build --noconfirm` to emit markdown, and run `targ c4-audit` to verify zero findings — not to handcraft the markdown. | [skills/c4/SKILL.md:71](../../skills/c4/SKILL.md#L71) | **⚠ UNTESTED** | Mechanical work offloaded; LLM keeps judgment only. |
| <a id="p10-drift-notes-persist"></a>P10 | Drift notes persist | For all deferred propagation proposals or recorded code/intent gaps, the skill instructs the agent to append a `## Drift Notes` entry to the target file; entries persist until a future `update` resolves them. | [skills/c4/SKILL.md:215](../../skills/c4/SKILL.md#L215) | **⚠ UNTESTED** | "Never silently disappear." |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E15 · c4 skill**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
