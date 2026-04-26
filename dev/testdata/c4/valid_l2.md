---
level: 2
name: foo-system
parent: "c1-foo-system.md"
children: []
last_reviewed_commit: df51bc93
---

# C2 — Foo (Container)

Refines E2 Foo into a worker that processes input and a loader that hydrates state from disk.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    e1([E1 · Operator<br/>runs Foo])
    e3(E3 · BarAPI<br/>remote service)
    e4(E4 · Disk<br/>local filesystem)

    subgraph e2 [E2 · Foo]
        e5[E5 · Worker]
        e6[E6 · Loader]
    end

    e1 -->|"R1: submits work items"| e5
    e6 -->|"R2: reads cached state"| e4
    e5 -->|"R3: fetches input"| e3
    e5 -->|"R4: asks for hydrated state"| e6

    class e1 person
    class e3,e4 external
    class e5,e6 container
    class e2 container

    click e1 href "#e1-operator" "Operator"
    click e2 href "#e2-foo" "Foo"
    click e3 href "#e3-barapi" "BarAPI"
    click e4 href "#e4-disk" "Disk"
    click e5 href "#e5-worker" "Worker"
    click e6 href "#e6-loader" "Loader"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-operator"></a>E1 | Operator | Person | Operator who triggers Foo runs | Human |
| <a id="e2-foo"></a>E2 | Foo | The system in scope | The system being refined at this level | This repo |
| <a id="e3-barapi"></a>E3 | BarAPI | External system | Upstream data source | bar.example.com |
| <a id="e4-disk"></a>E4 | Disk | External system | On-disk state | OS filesystem |
| <a id="e5-worker"></a>E5 | Worker | Container | Processes incoming work items | internal/worker |
| <a id="e6-loader"></a>E6 | Loader | Container | Hydrates state from disk on startup | internal/loader |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-operator-worker"></a>R1 | Operator | Worker | submits work items | CLI |
| <a id="r2-loader-disk"></a>R2 | Loader | Disk | reads cached state | filesystem |
| <a id="r3-worker-barapi"></a>R3 | Worker | BarAPI | fetches input | HTTPS |
| <a id="r4-worker-loader"></a>R4 | Worker | Loader | asks for hydrated state | function call |

## Cross-links

- Parent: [c1-foo-system.md](c1-foo-system.md) (refines **E2 · Foo**)
- Refined by: *(none yet)*
