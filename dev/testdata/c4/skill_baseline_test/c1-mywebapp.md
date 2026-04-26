---
level: 1
name: mywebapp
parent: null
children: []
last_reviewed_commit: 2153b9be
---

# C1 — MyWebApp (System Context)

Test fixture for skill TDD baseline: a small webapp.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    e1([E1 · Customer])
    e2[E2 · MyWebApp]
    e3(E3 · Database)

    e1 -->|"R1: uses"| e2
    e2 -->|"R2: reads/writes"| e3

    class e1 person
    class e3 external
    class e2 container

    click e1 href "#e1-customer" "Customer"
    click e2 href "#e2-mywebapp" "MyWebApp"
    click e3 href "#e3-database" "Database"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-customer"></a>E1 | Customer | Person | uses the app | human |
| <a id="e2-mywebapp"></a>E2 | MyWebApp | The system in scope | the system in scope | this fixture |
| <a id="e3-database"></a>E3 | Database | External system | stores app data | Postgres |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-customer-mywebapp"></a>R1 | Customer | MyWebApp | uses | HTTPS |
| <a id="r2-mywebapp-database"></a>R2 | MyWebApp | Database | reads/writes | TCP |

## Cross-links

- Parent: none (L1 is the root).
- Refined by: *(none yet)*
