---
level: 1
name: <NAME>
parent: null
children: []
last_reviewed_commit: <SHA>
---

# C1 — <NAME> (System Context)

![C1 <name> diagram](svg/c1-<name>.svg)

> Diagram source: [svg/c1-<name>.mmd](svg/c1-<name>.mmd). Re-render with `targ c4-render`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine.

`.mmd` source skeleton (place at `architecture/c4/svg/c1-<name>.mmd`):

```
%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    %% nodes — embed S<n> in label, e.g. user([S1 · Joe])
    %% relationships — embed R<n> in edge label, e.g. user -->|R1: invokes ...| cc
    %% class assignments
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="s1-PLACEHOLDER"></a>S1 | <element> | Person / External system / The system | <one sentence> | <where this lives in reality> |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-PLACEHOLDER"></a>R1 | <from> | <to> | <one sentence> | <protocol> |

## Cross-links

- Refined by: <list relative paths to L2 files, or "(none yet)">
