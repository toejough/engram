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
        %% components inside the focus container — embed E<n> in label
    end
    class focus container
    %% neighboring containers/people/externals as context
    %% relationships — embed R<n> in edge label
    %% click directives — one per node, jumping to its catalog row anchor
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e1-PLACEHOLDER"></a>E1 | <component> | Component | <one sentence> | [path/to/pkg](../../path/to/pkg) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|

## Cross-links

- Parent: <relative path to c2-*.md>
- Refined by: <list relative paths to c4-*.md, or "(none)">
