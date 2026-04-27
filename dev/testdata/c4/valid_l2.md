---
level: 2
name: foo-system
parent: "c1-foo-system.md"
children: []
last_reviewed_commit: df51bc93
---

# C2 — Foo (Container)

Refines S2 Foo into a worker that processes input and a loader that hydrates state from disk.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    s1([S1 · Operator<br/>runs Foo])
    s3(S3 · BarAPI<br/>remote service)
    s4(S4 · Disk<br/>local filesystem)

    subgraph s2 [S2 · Foo]
        s2-n1[S2-N1 · Worker]
        s2-n2[S2-N2 · Loader]
    end

    s1 -->|"R1: submits work items"| s2-n1
    s2-n2 -->|"R2: reads cached state"| s4
    s2-n1 -->|"R3: fetches input"| s3
    s2-n1 -->|"R4: asks for hydrated state"| s2-n2

    class s1 person
    class s3,s4 external
    class s2-n1,s2-n2 container
    class s2 container

    click s1 href "#s1-operator" "Operator"
    click s2 href "#s2-foo" "Foo"
    click s3 href "#s3-barapi" "BarAPI"
    click s4 href "#s4-disk" "Disk"
    click s2-n1 href "#s2-n1-worker" "Worker"
    click s2-n2 href "#s2-n2-loader" "Loader"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="s1-operator"></a>S1 | Operator | Person | Operator who triggers Foo runs | Human |
| <a id="s2-foo"></a>S2 | Foo | The system in scope | The system being refined at this level | This repo |
| <a id="s3-barapi"></a>S3 | BarAPI | External system | Upstream data source | bar.example.com |
| <a id="s4-disk"></a>S4 | Disk | External system | On-disk state | OS filesystem |
| <a id="s2-n1-worker"></a>S2-N1 | Worker | The system in scope | Processes incoming work items | internal/worker |
| <a id="s2-n2-loader"></a>S2-N2 | Loader | The system in scope | Hydrates state from disk on startup | internal/loader |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-s1-s2-n1"></a>R1 | S1 | S2-N1 | submits work items | CLI |
| <a id="r2-s2-n2-s4"></a>R2 | S2-N2 | S4 | reads cached state | filesystem |
| <a id="r3-s2-n1-s3"></a>R3 | S2-N1 | S3 | fetches input | HTTPS |
| <a id="r4-s2-n1-s2-n2"></a>R4 | S2-N1 | S2-N2 | asks for hydrated state | function call |

## Cross-links

- Parent: [c1-foo-system.md](c1-foo-system.md) (refines **S2 · Foo**)
- Refined by: *(none yet)*
