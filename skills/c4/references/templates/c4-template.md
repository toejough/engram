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
crosses to (carried over from the L3 parent's external set). R-edges only; labels
carry inline property IDs `[P…]`. See `skills/c4/references/mermaid-conventions.md`
for shape conventions.

![C4 <name> context diagram](svg/c4-<name>.svg)

> Diagram source: [svg/c4-<name>.mmd](svg/c4-<name>.mmd). Re-render with `targ c4-render`
> (or `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-<name>.mmd -o architecture/c4/svg/c4-<name>.svg`).
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n1-m#-p1-<slug>"></a>S2-N1-M#-P1 | <name> | For all <quantified inputs>, <guarantee>. | [file:line](../../path#L42) | [test_file:line](../../path_test#L88) OR **⚠ UNTESTED** | <caveats> |

## Cross-links

- Parent: [c3-<parent>.md](c3-<parent>.md) (refines **<S2-N1-M#> · <name>**)

See also:
- `skills/c4/references/property-ledger-format.md` for property statement style and
  the untested-property discipline.
- `skills/c4/references/mermaid-conventions.md` for the L4 call diagram convention and
  the SVG pre-render rule.
