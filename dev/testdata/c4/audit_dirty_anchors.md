---
level: 1
name: audit-dirty-anchors
parent: null
children: []
last_reviewed_commit: HEAD
---

# C1 — Doesn't Matter

```mermaid
flowchart LR
    classDef person      fill:#000
    classDef external    fill:#000
    classDef container   fill:#000

    a[E1 · Foo]
    b[E2 · Bar]
    c[E3 · Baz]

    a -->|R1: foo to bar| b
    b -->|R2: bar to baz| c

    class a person
    class b container
    class c external

    click a href "#nowhere" "Foo"
    click b href "#e2-bar" "Bar"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-foo"></a>E1 | Foo | Person | resp | sor |
| <a id="e2-bar"></a>E2 | Bar | Container | resp | sor |
| E3 | Baz | External system | resp | sor |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-foo-bar"></a>R1 | Foo | Bar | desc | proto |
| <a id="r2-bar-baz"></a>R2 | Bar | Baz | desc | proto |
