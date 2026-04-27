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

Scoped slice of [c3-<parent>.md](c3-<parent>.md): only the L3 elements and edges that touch
the focus, plus DI back-edges to wirers (dotted). The DI back-edge convention is per
`skills/c4/references/mermaid-conventions.md`.

![C4 <name> context diagram](svg/c4-<name>.svg)

> Diagram source: [svg/c4-<name>.mmd](svg/c4-<name>.mmd). Re-render with `targ c4-render`
> (or `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-<name>.mmd -o architecture/c4/svg/c4-<name>.svg`).
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n1-m#-p1-<slug>"></a>S2-N1-M#-P1 | <name> | For all <quantified inputs>, <guarantee>. | [file:line](../../path#L42) | [test_file:line](../../path_test#L88) OR **⚠ UNTESTED** | <caveats> |

## Dependency Manifest

(Include only if focus has DI deps. Drop the section otherwise.)

Each row is one injected dependency the focus receives. Manifest expands the L3 D-edge into
per-dep wiring rows. Reciprocal entries live in the wirer's L4 under `## DI Wires` — the two
sections must stay in sync.

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `<field>` | `<go-type>` | [M# · <wirer>](c3-<parent>.md#m#-<wirer>) ([c4-<wirer>.md](c4-<wirer>.md)) | <concrete adapter> | <P-list with range notation, e.g. S2-N1-M#-P1–P3> |

## DI Wires

(Include only if focus is a composition root / wires deps into OTHER components. Drop otherwise.)

Each row is one adapter the focus wires for a downstream consumer. Reciprocal entries live in
each consumer's L4 under `## Dependency Manifest`.

| Wired adapter | Concrete value | Consumer | Consumer field |
|---|---|---|---|
| <adapter> | <concrete value at file:line> | [M# · <consumer>](c3-<parent>.md#m#-<consumer>) ([c4-<consumer>.md](c4-<consumer>.md)) | `<field>` |

## Cross-links

- Parent: [c3-<parent>.md](c3-<parent>.md) (refines **<S2-N1-M#> · <name>**)

See also:
- `skills/c4/references/property-ledger-format.md` for property statement style + Dependency
  Manifest / DI Wires conventions.
- `skills/c4/references/mermaid-conventions.md` for the D[n] dotted back-edge convention and
  the SVG pre-render rule.
