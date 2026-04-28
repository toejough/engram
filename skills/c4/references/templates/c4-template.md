---
level: 4
name: <NAME>
parent: <relative path to c3-*.md>
children: []
last_reviewed_commit: <SHA>
---

# C4 — <NAME> (Property/Invariant Ledger)

> Component in focus: **<S2-N1-M#> · <name>** (refines L3 c<3>-<parent>).
> Source files in scope:
> - [path/to/source.go](../../path/to/source.go)
> - [path/to/source_test.go](../../path/to/source_test.go)

## Context (from L3)

Strict C4 call diagram: focus + sibling components touched + every external the focus
crosses to via DI (carried over from the L3 parent's external set). R-edges only;
labels carry inline property IDs `[P…]`. See
`skills/c4/references/mermaid-conventions.md` for shape conventions.

![C4 <name> context diagram](svg/c4-<name>.svg)

> Diagram source: [svg/c4-<name>.mmd](svg/c4-<name>.mmd). Re-render with `targ c4-render`
> (or `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-<name>.mmd -o architecture/c4/svg/c4-<name>.svg`).
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine.

## Wiring

(Include only if focus has DI deps. Drop the section otherwise.)

Companion view of the call diagram. Edges go `wirer → focus`; each label is the SNM ID
of the wrapped entity (the diagram node the seam ultimately drives behavior against).
Multiple manifest rows that share both wirer and wrapped entity collapse into one edge.

![C4 <name> wiring diagram](svg/c4-<name>-wiring.svg)

> Diagram source: [svg/c4-<name>-wiring.mmd](svg/c4-<name>-wiring.mmd). Re-render with `targ c4-render`.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n1-m#-p1-<slug>"></a>S2-N1-M#-P1 | <name> | For all <quantified inputs>, <guarantee>. | [file:line](../../path#L42) | [test_file:line](../../path_test#L88) OR **⚠ UNTESTED** | <caveats> |

## Dependency Manifest

(Include only if focus has DI deps. Drop the section otherwise.)

Each row is one DI seam the focus consumes. The wrapped entity is the diagram node
(component or external) the seam ultimately drives behavior against; it must also
appear on the call diagram (strict alignment, enforced by `targ c4-l4-build`).
Reciprocal entries live in the wirer's L4 under `## DI Wires`.

| Field | Type | Wired by | Wrapped entity | Properties |
|---|---|---|---|---|
| `<field>` | `<go-type>` | [M# · <wirer>](c3-<parent>.md#m#-<wirer>) ([c4-<wirer>.md](c4-<wirer>.md)) | `<SNM ID>` | <P-list with range notation, e.g. S2-N1-M#-P1–P3> |

## DI Wires

(Include only if focus is a composition root / wires deps into OTHER components. Drop otherwise.)

Each row is one DI seam this component wires into a downstream consumer. Reciprocal
entries live in each consumer's L4 under `## Dependency Manifest`.

| Field | Type | Consumer | Wrapped entity |
|---|---|---|---|
| `<field>` | `<go-type>` | [M# · <consumer>](c3-<parent>.md#m#-<consumer>) ([c4-<consumer>.md](c4-<consumer>.md)) | `<SNM ID>` |

## Cross-links

- Parent: [c3-<parent>.md](c3-<parent>.md) (refines **<S2-N1-M#> · <name>**)

See also:
- `skills/c4/references/property-ledger-format.md` for property statement style + Dependency
  Manifest / DI Wires column schemas.
- `skills/c4/references/mermaid-conventions.md` for the L4 two-diagram convention (call +
  wiring) and the SVG pre-render rule.
