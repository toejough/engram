# #603 — DI notation brainstorm: rendered sketches

Real `recall` L4 data (7 DI seams: Finder, Reader, SummarizerI, MemoryLister, externalFiles, fileCache, statusWriter — all wired by `cli`).

---

## Baseline (current convention)

```mermaid
flowchart LR
  cli[S2-N3-M2 · cli]
  recall[/S2-N3-M3 · recall/]
  ctx[S2-N3-M4 · context]
  mem[S2-N3-M5 · memory]
  ext[S2-N3-M6 · externalsources]
  ant[S2-N3-M7 · anthropic]
  cli -->|"R3: constructs + invokes Recall"| recall
  recall -->|"R8: strips transcript"| ctx
  recall -->|"R9: lists memories"| mem
  recall -->|"R10: ranks/extracts/summarizes"| ant
  recall -->|"R11: reads CLAUDE.md/rules/auto-memory/skills"| ext
  recall -.->|"D5: DI back-edge: Finder, Reader, SummarizerI,<br/>MemoryLister, externalFiles, statusWriter, dataDir"| cli
```

D5 points the wrong direction per C4's "arrow = initiates" rule, and the label is a comma-soup that doesn't tell you which adapter goes with which port.

---

## Approach D — annotated call edges

```mermaid
flowchart LR
  cli[S2-N3-M2 · cli]
  recall[/S2-N3-M3 · recall/]
  ctx[S2-N3-M4 · context]
  mem[S2-N3-M5 · memory]
  ext[S2-N3-M6 · externalsources]
  ant[S2-N3-M7 · anthropic]
  cli -->|"R3: constructs + invokes Recall"| recall
  recall -->|"R8: via Reader port<br/>(wired by cli: NewTranscriptReader)"| ctx
  recall -->|"R9: via MemoryLister port<br/>(wired by cli: memory.NewLister)"| mem
  recall -->|"R10: via SummarizerI port<br/>(wired by cli: NewSummarizer over Haiku)"| ant
  recall -->|"R11: via FileCache port<br/>(wired by cli: NewFileCache + Discover)"| ext
```

**Wins:** no back-edges, all arrows point the right way.
**Loses:** Finder and statusWriter (and dataDir) have no outbound call edge to ride on, so they vanish entirely. That's nearly half of recall's wiring made invisible.

---

## Approach C — separate wiring view (two diagrams per L4)

### View 1: call flow

```mermaid
flowchart LR
  cli[S2-N3-M2 · cli]
  recall[/S2-N3-M3 · recall/]
  ctx[S2-N3-M4 · context]
  mem[S2-N3-M5 · memory]
  ext[S2-N3-M6 · externalsources]
  ant[S2-N3-M7 · anthropic]
  cli -->|"R3: constructs + invokes Recall"| recall
  recall -->|"R8: strips transcript"| ctx
  recall -->|"R9: lists memories"| mem
  recall -->|"R10: ranks/extracts/summarizes"| ant
  recall -->|"R11: reads files"| ext
```

### View 2: wiring graph

```mermaid
flowchart LR
  cli[S2-N3-M2 · cli<br/>composition root]
  recall[/S2-N3-M3 · recall/]
  cli -->|"W1: Finder = recall.NewSessionFinder over DirLister"| recall
  cli -->|"W2: Reader = recall.NewTranscriptReader over context.FileReader"| recall
  cli -->|"W3: SummarizerI = recall.NewSummarizer over anthropic.CallerFunc"| recall
  cli -->|"W4: MemoryLister = memory.NewLister"| recall
  cli -->|"W5: externalFiles = externalsources.Discover(...)"| recall
  cli -->|"W6: fileCache = externalsources.NewFileCache(os.ReadFile)"| recall
  cli -->|"W7: statusWriter = os.Stderr or nil"| recall
```

All seven seams visible; arrows correctly point in the wiring-call direction (cli → recall, since cli initiates the construction). New `W<n>` namespace keeps it from competing with R/D.

---

## Approach C-hex — ports & adapters view

```mermaid
flowchart LR
  subgraph recall_hex ["recall · ports"]
    direction TB
    P1((Finder))
    P2((Reader))
    P3((SummarizerI))
    P4((MemoryLister))
    P5((externalFiles))
    P6((FileCache))
    P7((statusWriter))
  end
  cli[S2-N3-M2 · cli<br/>wirer]
  ctx[S2-N3-M4 · context]
  mem[S2-N3-M5 · memory]
  ext[S2-N3-M6 · externalsources]
  ant[S2-N3-M7 · anthropic]
  cli -->|"plugs in NewSessionFinder over os.ReadDir"| P1
  cli -->|"plugs in NewTranscriptReader"| P2
  cli -->|"plugs in NewSummarizer"| P3
  cli -->|"plugs in memory.NewLister"| P4
  cli -->|"plugs in Discover(...) result"| P5
  cli -->|"plugs in NewFileCache(os.ReadFile)"| P6
  cli -->|"plugs in os.Stderr"| P7
  P2 -.->|"wraps"| ctx
  P3 -.->|"wraps Haiku caller"| ant
  P4 -.->|"wraps"| mem
  P5 -.->|"wraps"| ext
  P6 -.->|"wraps"| ext
```

What it adds over plain C:

- **Ports are first-class diagram nodes** owned by the focus component. The interface name *is* the node identity, not a label on an edge.
- **Adapter-to-driven-system relationship is visible** via dotted "wraps" edges (e.g., the Reader port wraps `context`; the SummarizerI port wraps `anthropic`).
- **The wirer's role becomes obvious by position**: `cli` only ever connects to ports, never to driven systems directly.
- **Generalises to fakes/mocks**: swap the cli-side adapter with a test adapter; port stays, hexagon stays.

Cost: more nodes per diagram (one port node per DI seam), and we'd want a node style/class for ports so they read as distinct from components.

---

## Read

**C-hex** is the strongest answer to "how do we indicate who SETS the dependency cleanly?" It treats ports as named nodes, makes the wirer's role visual, and exposes the adapter-wraps-driven-system axis we've been missing.

**D** is too lossy — components with non-call DI seams (Finder, statusWriter, dataDir on recall) lose visibility entirely.

**Plain C** is a fine middle ground if C-hex feels too heavy.
