---
level: 1
name: audit-dirty-orphans
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

    a -->|R1: foo| b
    b -->|R2: bar| a

    class a person
    class b container

    click a href "#e1-foo" "Foo"
    click b href "#e2-bar" "Bar"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-foo"></a>E1 | Foo | Person | resp | sor |
| <a id="e3-baz"></a>E3 | Baz | External system | resp | sor |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-foo-bar"></a>R1 | Foo | Bar | desc | proto |
| <a id="r3-baz-foo"></a>R3 | Baz | Foo | desc | proto |
