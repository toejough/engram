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

    a[S1 · Foo]
    b[S2 · Bar]

    a -->|R1: foo| b
    b -->|R2: bar| a

    class a person
    class b container

    click a href "#s1-foo" "Foo"
    click b href "#s2-bar" "Bar"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="s1-foo"></a>S1 | Foo | Person | resp | sor |
| <a id="s3-baz"></a>S3 | Baz | External system | resp | sor |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-foo-bar"></a>R1 | Foo | Bar | desc | proto |
| <a id="r3-baz-foo"></a>R3 | Baz | Foo | desc | proto |
