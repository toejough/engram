---
level: 2
name: <NAME>
parent: <relative path to c1-*.md>
children: []
last_reviewed_commit: <SHA>
---

# C2 — <NAME> (Container)

![C2 <name> diagram](svg/c2-<name>.svg)

> Diagram source: [svg/c2-<name>.mmd](svg/c2-<name>.mmd). Re-render with `targ c4-render`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine.

`.mmd` source skeleton:

```
%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    %% nodes — embed N<n> in label, e.g. engram[N2 · Engram CLI]
    %% relationships — embed R<n> in edge label
    %% class assignments
```

## Element Catalog

| ID | Name | Type | Responsibility | Source |
|---|---|---|---|---|

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|

## Cross-links

- Parent: <relative path to c1-*.md>
- Refined by: <list relative paths to c3-*.md>
