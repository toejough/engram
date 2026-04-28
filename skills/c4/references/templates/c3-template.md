---
level: 3
name: <NAME>
parent: <relative path to c2-*.md>
children: []
last_reviewed_commit: <SHA>
---

# C3 — <NAME> (Component)

> Container in focus: <name of the L2 container being expanded>

![C3 <name> diagram](svg/c3-<name>.svg)

> Diagram source: [svg/c3-<name>.mmd](svg/c3-<name>.mmd). Re-render with `targ c4-render`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine.

`.mmd` source skeleton:

```
%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%
flowchart LR
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    subgraph focus [<container name>]
        %% components inside the focus container — embed M<n> in label, e.g. cli[M1 · cli dispatcher]
    end
    class focus container
    %% neighboring containers/people/externals as context
    %% relationships — embed R<n> in edge label (solid arrow for runtime calls)
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="m1-PLACEHOLDER"></a>M1 | <component> | Component | <one sentence> | [path/to/pkg](../../path/to/pkg) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-PLACEHOLDER"></a>R1 | <from> | <to> | <one sentence> | <protocol> |

## Cross-links

- Parent: <relative path to c2-*.md>
- Refined by: <list relative paths to c4-*.md, or "(none)">
