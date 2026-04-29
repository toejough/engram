# Worked Example: c4-recall (two-diagram L4)

Concrete walk-through of the L4 two-diagram convention applied to one focus component.
The full spec lives at `architecture/c4/c4-recall.json`; rendered output at
`architecture/c4/c4-recall.md` + `svg/c4-recall.svg` + `svg/c4-recall-wiring.svg`.

## Call-diagram nodes

Focus + siblings touched + externals crossed:

- `S2-N3-M3 · recall` (focus)
- `S2-N3-M2 · cli` · `S2-N3-M4 · context` · `S2-N3-M5 · memory`
  · `S2-N3-M6 · externalsources` · `S2-N3-M7 · anthropic` (sibling components)
- `S3 · Claude Code` (external — carried over from the L3 parent; required because
  recall's DI crosses to Claude Code's session directory and stderr)

## Call-diagram R-edges

Each ends with the P-IDs the call realizes:

- `R8: recall → context: strips transcript via TranscriptReader [P3, P4]`
- `R9: recall → memory: lists memories via MemoryLister [P11, P12, P13, P15]`
- `R10: recall → anthropic: ranks + extracts + summarizes [P5–P8, P11–P16]`
- `R12: recall → S3: reads sessions, transcripts; writes status [P1, P2, P3, P4, P9, P10, P18]`

## Manifest rows

One per DI seam — wirer `cli` plugs each seam into recall:

| Field | Type | Wired by | Wrapped entity | Properties |
|---|---|---|---|---|
| `finder` | `Finder` | cli | `S3` | P1, P2, P9 |
| `reader` | `Reader` | cli | `S3` | P3, P4, P9, P10 |
| `summarizer` | `SummarizerI` | cli | `S2-N3-M7` | P5–P8, P11–P16 |
| `memoryLister` | `MemoryLister` | cli | `S2-N3-M5` | P11–P13, P15 |
| `externalFiles` | `[]ExternalFile` | cli | `S2-N3-M6` | P5–P8 |
| `fileCache` | `*FileCache` | cli | `S2-N3-M6` | P5–P7, P17 |
| `statusWriter` | `io.Writer` | cli | `S3` | P18 |

## Wiring edges

Derived from the manifest by grouping `(wired_by_id, wrapped_entity_id)`; 7 manifest
rows collapse to 4 edges:

- `cli → recall` label `S2-N3-M5` (covers `memoryLister`)
- `cli → recall` label `S2-N3-M7` (covers `summarizer`)
- `cli → recall` label `S2-N3-M6` (covers `externalFiles`, `fileCache`)
- `cli → recall` label `S3` (covers `finder`, `reader`, `statusWriter`)

## Strict-alignment check

Every wrapped entity (`S3`, `S2-N3-M5`, `S2-N3-M6`, `S2-N3-M7`) is also a node on the
call diagram. The L4 builder rejects the spec if it isn't.
