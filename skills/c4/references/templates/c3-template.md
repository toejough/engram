---
level: 3
name: <NAME>
parent: <relative path to c2-*.md>
children: []
last_reviewed_commit: <SHA>
---

# C3 — <NAME> (Component)

> Container in focus: <name of the L2 container being expanded>

```mermaid
flowchart LR
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    subgraph focus [<container name>]
        %% components inside the focus container
    end
    class focus container
    %% neighboring containers/people/externals as context
    %% relationships
```

## Element Catalog

| Name | Type | Responsibility | Code Pointer |
|---|---|---|---|
| <component> | Component | <one sentence> | [path/to/pkg](../../path/to/pkg) |

## Relationships

| From | To | Description | Protocol/Medium |
|---|---|---|---|

## Cross-links

- Parent: <relative path to c2-*.md>
- Refined by: <list relative paths to c4-*.md, or "(none)">
