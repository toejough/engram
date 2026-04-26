---
level: 2
name: mywebapp-internal
parent: "c1-mywebapp.md"
children: []
last_reviewed_commit: 2153b9be
---

# C2 — MyWebApp (Container)

Test fixture: refines MyWebApp into API, Worker, and Cache containers.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    e1([E1 · Customer])
    e3(E3 · Database)

    subgraph e2 [E2 · MyWebApp]
        e4[E4 · API]
        e5[E5 · Worker]
        e6[E6 · Cache]
    end

    e1 -->|"R1: uses"| e4
    e4 -->|"R2: reads/writes"| e3
    e4 -->|"R3: reads"| e6
    e5 -->|"R4: reads/writes"| e3

    class e1 person
    class e3 external
    class e4,e5,e6 container
    class e2 container

    click e1 href "#e1-customer" "Customer"
    click e2 href "#e2-mywebapp" "MyWebApp"
    click e3 href "#e3-database" "Database"
    click e4 href "#e4-api" "API"
    click e5 href "#e5-worker" "Worker"
    click e6 href "#e6-cache" "Cache"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-customer"></a>E1 | Customer | Person | uses the app | human |
| <a id="e2-mywebapp"></a>E2 | MyWebApp | The system in scope | the system in scope | this fixture |
| <a id="e3-database"></a>E3 | Database | External system | stores app data | Postgres |
| <a id="e4-api"></a>E4 | API | Container | HTTP API serving user requests | this fixture |
| <a id="e5-worker"></a>E5 | Worker | Container | background jobs | this fixture |
| <a id="e6-cache"></a>E6 | Cache | Container | hot-data cache | this fixture |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-customer-api"></a>R1 | Customer | API | uses | HTTPS |
| <a id="r2-api-database"></a>R2 | API | Database | reads/writes | TCP |
| <a id="r3-api-cache"></a>R3 | API | Cache | reads | TCP |
| <a id="r4-worker-database"></a>R4 | Worker | Database | reads/writes | TCP |

## Cross-links

- Parent: [c1-mywebapp.md](c1-mywebapp.md) (refines **E2 · MyWebApp**)
- Refined by:
  - [`c3-api.md`](./c3-api.md) — refines E4 · API
  - [`c3-worker.md`](./c3-worker.md) — refines E5 · Worker
  - [`c3-cache.md`](./c3-cache.md) — refines E6 · Cache
