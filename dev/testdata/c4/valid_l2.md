---
level: 2
name: foo-system
parent: "c1-foo-system.md"
children: []
last_reviewed_commit: df51bc93
---

# C2 — Foo (Container)

Refines S2 Foo into a worker that processes input and a loader that hydrates state from disk.

![C2 foo-system container diagram](svg/c2-foo-system.svg)

> Diagram source: [svg/c2-foo-system.mmd](svg/c2-foo-system.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c2-foo-system.mmd -o architecture/c4/svg/c2-foo-system.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R-edges between the same node pair.

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
