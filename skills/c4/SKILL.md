---
name: c4
description: Use when generating, updating, or reviewing C4 architecture diagrams under architecture/c4/. Triggers on "/c4", "create C4 diagram", "update C4", "add component to L3", "regenerate context diagram", or any request to add/modify architecture diagrams in C4 form.
---

# C4 Diagram Skill

Generate and maintain C4 architecture diagrams under `architecture/c4/`. Diagrams live at
`architecture/c4/c[level]-[name].md`. L1–L3 use mermaid + element catalog + relationships table.
L4 is a property/invariant ledger linking to enforcing code and validating tests.

## Sub-actions

Dispatch by intent. The user invokes `/c4 <sub-action> [args]`.

| Sub-action | What it does |
|---|---|
| `create <level> <name>` | Draft a new diagram file; require explicit user approval before writing. |
| `update <name>` | Modify an existing diagram, then propose updates per affected layer. |
| `review <name>` | Read-only drift report. No edits. |
| `audit` | Sweep `architecture/c4/`, produce a roll-up drift report. No edits. |

## Non-Negotiable Rules

1. **Ask, don't guess, on code/intent conflict.** When code reality and intent (docs / commit
   bodies / session memory) disagree, STOP drafting. Present both views. Ask the user which is
   correct. Record the resolution in the file (drift note) or proceed per their answer.
2. **Never invent test pointers** (L4). If a property has no test, mark it **⚠ UNTESTED**.
3. **Never edit a non-target file without per-file approval.** Propagation is by proposal +
   user approve/skip/defer, not by silent edit.
4. **Every diagram uses the project mermaid convention.** classDef block at top, `:::person /
   :::external / :::container / :::component` classes. See `references/mermaid-conventions.md`.
5. **Cross-link in the file body.** No index file. Each file names its parent and children
   directly with relative paths.
6. **Every diagram element and edge carries a hierarchical ID.** Node IDs follow a level-scoped
   namespace: L1 nodes use `S<n>`, L2 nodes use `N<n>`, L3 nodes use `M<n>`, L4 properties use
   `P<n>`. Cross-doc references use the full hyphen-separated path (`S2-N3-M5-P1`). Relationship
   rows use `R<n>` (runtime calls) — the only edge namespace at any level. The same
   IDs appear inside mermaid node labels and edge labels. Every L1–L3 node also gets a
   `click NODE href "#anchor"` directive linking to the catalog row's anchor. Catalog and
   relationships rows carry HTML anchors (`<a id="s1-…"></a>`, `<a id="m3-…"></a>`, etc.) so
   the links resolve. Mismatches (a node ID with no catalog row, or a catalog row whose ID isn't
   on the diagram) are reported as drift findings by `review` and `audit`. See
   `references/mermaid-conventions.md` for the exact pattern. Lower-level docs consult
   higher-level docs to find their ancestors' path prefix when constructing cross-doc references.
   ID validation is syntactic — `ParseIDPath` enforces the namespace rules; no global registry
   is needed.
7. **L4 has two diagrams: a strict C4 call diagram + a wiring diagram.** No D-edges, no port
   nodes, no `W`/`A` edges anywhere. The convention:
   - **Call diagram.** Strict C4: SNMPR-style nodes and `R<n>` runtime-call edges only.
     Each R-edge label may end with the property IDs the call realizes:
     `R8: strips transcript via TranscriptReader [P3, P4, P9, P10]`. The focus component is
     a node on this diagram; every external the focus crosses to via DI is also a node here
     with at least one R-edge from the focus to it.
   - **Wiring diagram.** A small companion view. Edges go `wirer → focus`; the label is the
     SNM ID of the **wrapped entity** — the diagram entity (component or external) the DI
     seam ultimately drives behavior against. Multiple DI seams that share both a wirer and
     a wrapped entity collapse into one wiring edge. The wiring diagram introduces no nodes
     the call diagram lacks.
   - **Manifest row schema** (consumer-side `## Dependency Manifest`): `field`, `type`,
     `wired_by_id` / `wired_by_name` / `wired_by_l3` / `wired_by_l4`, `wrapped_entity_id`,
     `properties` (P-ID short-form list). No `concrete_adapter` field.
   - **Strict alignment.** Every `wrapped_entity_id` on a manifest row must match an `id` on
     the call diagram. The L4 builder rejects any wiring that violates this.
   - **Wiring edges are derived, not authored.** Group manifest rows by
     `(wired_by_id, wrapped_entity_id)`; emit one wiring edge per group. The build target
     produces both `.mmd` files (`<name>.mmd` and `<name>-wiring.mmd`).

   See `references/mermaid-conventions.md` for shape/edge syntax and
   `references/property-ledger-format.md` for the manifest + DI Wires table schemas.

8. **All C4 diagrams are pre-rendered to SVG via `targ c4-render`.** GitHub's Mermaid
   renderer doesn't support the ELK layout engine that we need for clean R-edge layout.
   The `.mmd` source lives at `architecture/c4/svg/c<level>-<name>.mmd`, the rendered
   SVG at `architecture/c4/svg/c<level>-<name>.svg`. L4 emits two of each pair (call +
   wiring). The markdown file embeds the SVG via `![alt](svg/...)` rather than an inline
   ` ```mermaid ` block. Both source and rendered output are committed. Re-render with
   `targ c4-render` after editing any `.mmd`.

## Workflow: `create <level> <name>`

1. Read `architecture/c4/`. Note what already exists.
2. If `level > 1`, read the parent-level file. The new diagram MUST refine an element of its
   parent.
3. Shallow-scan the repo: `ls`, top-level dirs, `cmd/`, `internal/` package list, `README.md`.
4. For L3/L4, deep-read the specific packages/files in scope.
5. Read intent sources:
   - `CLAUDE.md` (project-level + user-global)
   - `docs/` tree
   - `git log --format=full` (scoped to files in play for L3/L4; recent N=50 repo-wide for L1/L2).
     Commit bodies are first-class evidence of *why*. Quote them when explaining a relationship,
     a deprecation, or a drift note.
   - Recent session memory: `engram recall --query "<topic>"`
6. **If conflict:** stop, present each conflict with both views, ask the user. Record resolutions.
7. Open `references/templates/c<level>-template.md` and fill it in.
8. For L1–L3, follow `references/mermaid-conventions.md`. For L4, follow
   `references/property-ledger-format.md`.
9. Show full draft to user, get approval, write the file at `architecture/c4/c<level>-<name>.md`.
10. After write: scan parent file for cross-link updates. Present those as propagation proposals
    (see Workflow: update). Apply approved ones; skip/defer the rest.

## Workflow: `create 1 <name>` (L1 specifics)

For L1, the workflow uses the `dev/c4-*` targets so mechanical work is offloaded
and only judgment remains in the LLM:

1. Read `architecture/c4/`. Note what already exists.
2. (L1: skip parent — there is no L0.)
3. **Run `targ c4-l1-externals --root . --packages ./...`** — capture JSON listing
   external-system candidates (HTTP calls, filesystem boundaries, subprocess
   invocations, env reads). The LLM picks which ones become diagram externals.
4. (L3/L4 only — skip for L1.)
5. **Run `targ c4-history --since 90d --limit 50`** — capture structured JSON of
   recent commits + bodies for intent context. Also read `CLAUDE.md`, `docs/`,
   and `engram recall --query "<topic>"` for additional intent.
6. **If conflict:** stop, present, ask the user. Record resolutions.
7. **Author `architecture/c4/c1-<name>.json`** filling in the L1Spec schema (see
   the design spec at `docs/superpowers/specs/2026-04-15-c4-l1-targets-design.md`).
8. **Run `targ c4-l1-build --input architecture/c4/c1-<name>.json --noconfirm`**
   to emit the markdown.
9. **Run `targ c4-audit --file architecture/c4/c1-<name>.md`** to verify zero
   findings. If any: revise the JSON and rebuild.
10. Show user the rendered markdown for approval; commit both `.json` and `.md`.

For `update`: edit the JSON, rerun `targ c4-l1-build`, rerun `targ c4-audit`,
then run the **Propagation Discipline** sweep (see below) before presenting
the diff.

## Workflow: `create 3 <name>` (L3 specifics)

L3 follows the same JSON-spec → build → audit pattern as `create 1`, with
two L3-specific additions: each diagram has a `focus` field naming the L2
container being refined, and components carry `code_pointer` paths that the
audit verifies.

1. Read the parent c2 file to find the parent's node IDs (e.g., `N3`).
   The L3's `focus.id`/`focus.name` must match an in-scope element of the parent.
   L3 component IDs use `M<n>` scoped within this diagram (start at `M1`).
   `from_parent` neighbors carry IDs/names from the parent c2 spec.
2. **Author `architecture/c4/c3-<name>.json`** per the L3Spec schema:
   `focus` (id, name, responsibility), `elements` (each with explicit `M<n>` ID;
   `kind: "component"` requires `code_pointer`; `from_parent: true`
   neighbors carry IDs/names from peer specs), and `relationships`
   referencing element names or `focus.name`.
3. **Run `targ c4-l3-build --input architecture/c4/c3-<name>.json --noconfirm`**
   to emit the markdown.
4. **Run `targ c4-audit --file architecture/c4/c3-<name>.md`** to verify
   zero findings.
5. **Run the Propagation Discipline sweep** (see below) — for an L3 create,
   that means updating the parent c2's `cross_links.refined_by` and rebuilding
   existing c3 siblings so their auto-generated cross-links pick up the new file.
6. Show the rendered markdown to the user for approval; commit both `.json`
   and `.md`.

For `update`: edit the JSON, rerun `targ c4-l3-build`, rerun
`targ c4-audit`, then run the Propagation Discipline sweep before
presenting the diff.

## Workflow: `create 4 <name>` (L4 specifics)

L4 follows the same JSON-spec → build → audit pattern with one critical
addition: property identification MUST use the **Two-Tier Extraction
Discipline** (see below). One-pass single-tier authoring tends to
over-decompose, miss test pointers, and let plausible-but-wrong rows
stand. The discipline is non-negotiable.

1. Read the parent c3 file to learn relevant `M<n>` IDs and neighbor names.
   The L4's `focus.id` MUST already exist in the parent c3.
2. **Tier 1 — wide scan.** Dispatch a Haiku-class sub-agent at the
   component's source (Go package files, skill markdown, hook script,
   or JSON manifest as applicable). Instruct it to enumerate **every
   plausible candidate** across the L4 schema:
   - **Property candidates** — testable behaviors, architectural invariants
     likely UNTESTED (DI seams, no-direct-I/O, error-wrapping discipline,
     format contracts), shape/parse/lifecycle invariants.
   - **Call-diagram nodes** — sibling components the focus calls and every
     external system the focus crosses to via DI (filesystem, network, OS,
     Anthropic API, etc.). Each external must end up as a node on the call
     diagram — the L4 builder enforces this via the strict-alignment rule.
   - **Manifest rows** — one per DI seam: `field`, `type`, wirer
     (`wired_by_id`/`name`/`l3`), and `wrapped_entity_id` (which call-diagram
     node the seam ultimately drives behavior against).
   - **R-edge property tags** — for each R-edge, the property IDs the call
     realizes (these become the `properties` field on the edge).

   **Bias toward too many findings.** Tier 1 owns recall, not precision.
3. **Tier 2 — verify and refine.** Take Tier 1's candidate list. For each
   tier of candidate:
   - **Properties:** read source to verify the claim, prune false positives,
     merge near-duplicates, split rows that smuggle two guarantees, locate
     exact `enforced_at` and `tested_at` file:line, mark genuinely untested
     as `tested_at: []`.
   - **Call-diagram nodes:** confirm each external the focus crosses to
     appears on the diagram with at least one R-edge from the focus; confirm
     siblings appear in the L3 parent's catalog (no fabricated nodes).
   - **Manifest rows:** confirm `wrapped_entity_id` matches a node on the
     call diagram (strict alignment); confirm each row's wirer is correct
     by reading the wirer's construction code.
   - **R-edge property tags:** confirm each P-ID in an R-edge's `properties`
     list corresponds to a property the called code path actually realizes.

   **Tier 2 owns correctness.** When in doubt, re-read the source — never
   invent test pointers, never invent externals not actually crossed.
4. **Author `architecture/c4/c4-<name>.json`** per the L4Spec schema in
   `dev/c4_l4.go`: `focus` (id, name, l3_container), `sources`,
   `context_prose`, optional `legend_items`, `diagram` (call-diagram nodes —
   focus + sibling components touched + every external the focus crosses to
   via DI — and `R<n>` edges with optional `properties: ["P3", ...]` lists),
   optional `dependency_manifest` (consumer side: each row carries
   `wrapped_entity_id` matching a `diagram.nodes[].id`) or `di_wires`
   (provider side), `properties`. The wiring diagram is derived from the
   manifest by the build target — never authored directly.
5. **Run `targ c4-l4-build --input architecture/c4/c4-<name>.json --noconfirm`**
   to emit the markdown and the two `.mmd` sources (`<name>.mmd` for the
   call diagram, `<name>-wiring.mmd` for the wiring diagram).
6. **Run `targ c4-render`** to regenerate both SVGs from the new `.mmd` pair.
7. **Run `targ c4-audit --file architecture/c4/c4-<name>.md`** —
   `property_link_unresolved` will catch dead enforced/tested paths.
8. Run the Propagation Discipline sweep — for an L4 create, that means
   updating the parent c3 (no `refined_by` field on c3 today, but
   verify the parent's catalog entry for `focus.id` is current) and
   rebuilding peer L4 siblings so their auto-generated Siblings
   cross-link picks up the new file.
9. Show the rendered markdown to the user for approval; commit the
   `.json`, `.md`, `.mmd`, and `.svg`.

For `update`: edit the JSON, rerun `targ c4-l4-build`, rerun
`targ c4-render`, rerun `targ c4-audit`, then run the Propagation
Discipline sweep before presenting the diff.

## Workflow: `update <name>`

1. Read the target file and its parent + children (per the file's front-matter `parent` and
   `children` fields).
2. Take the user's requested change.
3. Re-ground in code (steps 3–5 of `create`, scoped to affected packages).
4. Resolve any new code/intent conflicts via ask-the-user.
5. Draft the new diagram + catalog state.
6. Classify the change so propagation knows what to do:
   - **Renamed element** → parent's `from_parent` carry-over and every child's `from_parent`
     carry-over need the same rename. Mermaid edges using the old name need rewriting.
   - **Removed element** → parent's `from_parent` reference is orphaned; any child whose
     `focus.id` matches the removed ID is invalidated.
   - **New element** → parent's catalog should add a corresponding entry; an L(N+1) child can
     be scaffolded.
   - **Changed responsibility/relationship** → parent's matching prose may drift; children
     whose `from_parent` element previously had a different responsibility need a re-read.
   - **L3 code-pointer change** → the audit's `code_pointer_unresolved` finding catches dead
     paths automatically on next audit.
7. **Run the Propagation Discipline sweep** (see below) to apply the classified change to every
   affected file.
8. Present, in order: the target-layer diff, then per-affected-layer proposed change. Each
   proposed edit is a unified diff with a one-line reason.
9. For each proposal, the user picks `[a]pply`, `[s]kip`, or `[d]efer`. Apply approved edits.
   Persist deferred ones as drift notes in the target file.

## Workflow: `review <name>`

Steps 1–4 of `update`, read-only. Output a report: drift findings, missing cross-links, broken
code/test pointers, untested L4 properties added since last review, AND **ID-mismatch findings**
(diagram nodes/edges whose IDs don't match catalog/relationships rows, or vice versa). No edits.

## Workflow: `audit`

Loop `review` over every file in `architecture/c4/`. Produce a roll-up report.

## Two-Tier Extraction Discipline

Property identification (and analogous candidate-mining tasks at any
level) MUST be split across two model tiers. One-pass single-tier
authoring is forbidden because the same agent's bias for plausible-
looking output becomes the final output — it over-decomposes to
appear thorough, marks tested things UNTESTED because it didn't trace
the test, and lets conflated claims stand because no separate verifier
ran.

| Tier | Model class | Job | Owns |
|---|---|---|---|
| 1 | Haiku-class | Wide scan: enumerate every plausible candidate from source | Recall (bias toward too many) |
| 2 | Sonnet-class+ | Verify each candidate against source, prune false positives, merge duplicates, split conflated rows, locate exact file:line, fill gaps Tier 1 missed | Precision and correctness of the final artifact |

**Rules:**

- Tier 1 output is **signal**, never the final artifact. Never write
  `c4-<name>.json` directly from Tier 1.
- Tier 2 must read source to verify, not just trust Tier 1's claims.
  When Tier 1 says `tested_at: <file>:<line>`, Tier 2 opens that file
  and confirms the line actually asserts the claim.
- Tier 2 may add candidates Tier 1 missed. Tier 1's enumeration is a
  floor, not a ceiling.
- Tier 2 never invents test pointers (Rule 2 above). When Tier 1
  proposed a `tested_at` that doesn't survive verification, Tier 2
  marks the row UNTESTED — does not delete it, does not fabricate a
  different test.

**Empirical baseline that motivates the rule:** the 19-component L4
regen run (commit history references the regen) showed that single-
tier Sonnet sub-agents over-decomposed small components (`main`: 5
properties where 3 sufficed) and missed individual test pointers
that the original human author had located (`memory` round-trip,
`cli` type-routing, `recall` FormatResult, `main` signal_test seam).
Two-tier dispatch is designed to catch both classes of failure.

**When this rule applies beyond L4:** any task that mines source
artifacts for findings — refactor opportunities, untested invariants,
audit candidates, DI-seam analysis. If the artifact has both "what
COULD be a finding" and "what IS a finding" stages, separate the two
across model tiers.

## Worked example: c4-recall (two-diagram L4)

Concrete walk-through of rule 7 applied to one focus component. The full
spec is at `architecture/c4/c4-recall.json`; rendered output at
`architecture/c4/c4-recall.md` + `svg/c4-recall.svg` + `svg/c4-recall-wiring.svg`.

**Call-diagram nodes** (focus + siblings touched + externals crossed):

- `S2-N3-M3 · recall` (focus)
- `S2-N3-M2 · cli` · `S2-N3-M4 · context` · `S2-N3-M5 · memory`
  · `S2-N3-M6 · externalsources` · `S2-N3-M7 · anthropic` (sibling components)
- `S3 · Claude Code` (external — carried over from the L3 parent;
  required because recall's DI crosses to Claude Code's session
  directory and stderr)

**Call-diagram R-edges** (each ends with the P-IDs the call realizes):

- `R8: recall → context: strips transcript via TranscriptReader [P3, P4]`
- `R9: recall → memory: lists memories via MemoryLister [P11, P12, P13, P15]`
- `R10: recall → anthropic: ranks + extracts + summarizes [P5–P8, P11–P16]`
- `R12: recall → S3: reads sessions, transcripts; writes status [P1, P2, P3, P4, P9, P10, P18]`

**Manifest rows** (one per DI seam — wirer `cli` plugs each seam into recall):

| Field | Type | Wired by | Wrapped entity | Properties |
|---|---|---|---|---|
| `finder` | `Finder` | cli | `S3` | P1, P2, P9 |
| `reader` | `Reader` | cli | `S3` | P3, P4, P9, P10 |
| `summarizer` | `SummarizerI` | cli | `S2-N3-M7` | P5–P8, P11–P16 |
| `memoryLister` | `MemoryLister` | cli | `S2-N3-M5` | P11–P13, P15 |
| `externalFiles` | `[]ExternalFile` | cli | `S2-N3-M6` | P5–P8 |
| `fileCache` | `*FileCache` | cli | `S2-N3-M6` | P5–P7, P17 |
| `statusWriter` | `io.Writer` | cli | `S3` | P18 |

**Wiring edges** (derived from the manifest by grouping
`(wired_by_id, wrapped_entity_id)`; 7 manifest rows collapse to 4 edges):

- `cli → recall` label `S2-N3-M5` (covers `memoryLister`)
- `cli → recall` label `S2-N3-M7` (covers `summarizer`)
- `cli → recall` label `S2-N3-M6` (covers `externalFiles`, `fileCache`)
- `cli → recall` label `S3` (covers `finder`, `reader`, `statusWriter`)

Strict-alignment check: every wrapped entity (`S3`, `S2-N3-M5`, `S2-N3-M6`,
`S2-N3-M7`) is also a node on the call diagram. The L4 builder rejects the
spec if it isn't.

## Propagation Discipline

A C4 diagram set is interconnected: parent ↔ children, siblings ↔ siblings.
Whenever you create a new diagram or rename/remove elements that another
diagram references, you MUST propagate. Skipping propagation creates drift
the registry will surface on the next audit.

**Required sweep after any create or update:**

1. **Update the parent's `cross_links.refined_by`.** Open the parent JSON,
   add or remove the entry for the affected child file, and rerun the
   parent's build target. (L1's children are L2 files; L2's are L3; etc.)
   The build re-emits the parent's "Refined by:" cross-link section from
   that array. **This step is required even when the parent already has
   unrelated drift in `refined_by`** — fix it now or capture as a Drift
   Note. (Note: the front-matter `children` field is currently hard-stamped
   `[]` by the build target and is not the propagation surface — work
   through `cross_links.refined_by` only.)

2. **Rebuild siblings.** For any peer at the same level whose `parent`
   matches the changed file, rerun the build target so its auto-generated
   "Siblings:" cross-link section refreshes. Idempotent rebuild — only the
   auto-generated sections will diff.

3. **Walk children.** For every child of the modified file, check whether
   its `from_parent` carry-overs still match the parent's element names and
   IDs. Rebuild any child whose carry-overs drift.

4. **Sweep.** Run `targ c4-audit` on every modified `.md`. Goal: zero findings. Capture
   intentional gaps as Drift Notes in the relevant file.

### Rule 3 reconciliation

The non-target-edit rule (rule 3 above) forbids silent edits to files you
weren't tasked to change. Propagation needs reconciling with that:

- **Rebuilds that only diff auto-generated sections** (mermaid block,
  catalog, cross-links — anything `c4-l*-build` regenerates) **are
  propagation, not edits**, and don't require per-file approval.
- **JSON edits to non-target files** (changing element names, adding
  carry-overs, etc.) **are edits and DO require per-file approval** —
  present them as proposals with `[a]pply`/`[s]kip`/`[d]efer`.

### Common rationalizations to reject

| Excuse | Reality |
|---|---|
| "That's pre-existing drift, not caused by my change" | If you're touching this set, leaving known drift in place creates audit findings on the next run. Fix it as part of your change, or capture it as a Drift Note. |
| "The change is too small to need propagation" | Every C4 file cross-references peers. Skipping the sweep means `c4-audit` surfaces conflicts later when context is gone. |
| "Rebuilding peers feels like editing files I shouldn't" | Idempotent rebuilds of auto-generated sections are propagation; see Rule 3 reconciliation above. |
| "I'll catch it in the next audit" | The next audit may be in a different session, after the rationale has been lost. Propagate now. |

## Drift Notes

When the user defers a propagation proposal or chooses to record a code/intent gap rather than
resolve it, append to a `## Drift Notes` section at the bottom of the target file:

```markdown
## Drift Notes

- **YYYY-MM-DD** — <one-line description>. Reason: <why deferred>.
```

Drift notes never silently disappear. They persist until a future `update` resolves them.

## References (load on demand)

- `references/c4-principles.md` — the 4 abstractions, 4 levels, common pitfalls.
- `references/mermaid-conventions.md` — classDef block + shape conventions + GitHub quirks.
- `references/property-ledger-format.md` — L4 row format + untested-property discipline.
- `references/templates/c<1-4>-template.md` — per-level starter scaffolds.

## Verification

This skill was verified with:

1. **Behavioral RED/GREEN test** — `tests/baseline-output-no-skill.md` vs.
   `tests/baseline-output-with-skill.md`. The skill-loaded run must produce a file at
   `architecture/c4/c1-<name>.md` with mermaid classDef block, element catalog, relationships
   table, and explicit cross-links to L2.
2. **Pressure test 1 — code/intent conflict** (`tests/pressure-conflict.md`): given docs that
   disagree with code, the skill must surface the conflict and ask, not silently pick one.
3. **Pressure test 2 — propagation** (`tests/pressure-propagation.md`): renaming a container at
   L2 must trigger proposed updates to the L1 parent and every L3 child file referencing it.
4. **Pressure test 3 — L4 untested property** (`tests/pressure-untested-property.md`): given a
   real invariant with no test, the skill must mark it **⚠ UNTESTED**, not omit it and not
   fabricate a test link.
