---
level: 3
name: worker
parent: "c2-mywebapp-internal.md"
children: []
last_reviewed_commit: 2153b9be
---

# C3 — Worker (Component)

Test fixture: refines Worker into a queue reader and email sender.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000


    subgraph e5 [E5 · Worker]
        e12[E12 · queue]
        e13[E13 · sender]
    end

    e12 -->|"R1: hands off jobs"| e13

    class e12,e13 component
    class e5 container

    click e5 href "#e5-worker" "Worker"
    click e12 href "#e12-queue" "queue"
    click e13 href "#e13-sender" "sender"
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e5-worker"></a>E5 | Worker | Container in focus | Container in focus — refined from c2-mywebapp-internal.md. | — |
| <a id="e12-queue"></a>E12 | queue | Component | reads jobs off the queue | [./queue.go](./queue.go) |
| <a id="e13-sender"></a>E13 | sender | Component | sends email | [./sender.go](./sender.go) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-queue-sender"></a>R1 | queue | sender | hands off jobs | Go function call |

## Cross-links

- Parent: [c2-mywebapp-internal.md](c2-mywebapp-internal.md) (refines **E5 · Worker**)
- Siblings:
  - [c3-api.md](c3-api.md)
  - [c3-cache.md](c3-cache.md)
- Refined by: *(none yet)*
