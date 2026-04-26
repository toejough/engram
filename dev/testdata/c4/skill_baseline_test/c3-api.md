---
level: 3
name: api
parent: "c2-mywebapp-internal.md"
children: []
last_reviewed_commit: 2153b9be
---

# C3 — API (Component)

Test fixture: refines API into a router and handler.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    e1([E1 · Customer])

    subgraph e4 [E4 · API]
        e10[E10 · router]
        e11[E11 · handler]
    end

    e1 -->|"R1: calls"| e10
    e10 -->|"R2: dispatches"| e11

    class e1 person
    class e10,e11 component
    class e4 container

    click e4 href "#e4-api" "API"
    click e1 href "#e1-customer" "Customer"
    click e10 href "#e10-router" "router"
    click e11 href "#e11-handler" "handler"
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e4-api"></a>E4 | API | Container in focus | Container in focus — refined from c2-mywebapp-internal.md. | — |
| <a id="e1-customer"></a>E1 | Customer | Person | uses the app | — |
| <a id="e10-router"></a>E10 | router | Component | dispatches incoming HTTP requests | [./router.go](./router.go) |
| <a id="e11-handler"></a>E11 | handler | Component | endpoint logic | [./handler.go](./handler.go) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-customer-router"></a>R1 | Customer | router | calls | HTTPS |
| <a id="r2-router-handler"></a>R2 | router | handler | dispatches | Go function call |

## Cross-links

- Parent: [c2-mywebapp-internal.md](c2-mywebapp-internal.md) (refines **E4 · API**)
- Siblings:
  - [c3-cache.md](c3-cache.md)
  - [c3-worker.md](c3-worker.md)
- Refined by: *(none yet)*
