---
level: 4
name: main
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — main (Property/Invariant Ledger)

> Component in focus: **E20 · main.go** (refines L3 c3-engram-cli-binary).
> Source files in scope:
> - [../../cmd/engram/main.go](../../cmd/engram/main.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): Claude Code execs the
binary as a subprocess (R1) and `main` forwards the cli targets to `targ.Main` (R2).
`main.go` itself contains no business logic — all real I/O concrete adapters are wired by
`cli.SetupSignalHandling`, which is the entry point's only call.

![C4 main context diagram](svg/c4-main.svg)

> Diagram source: [svg/c4-main.mmd](svg/c4-main.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-main.mmd -o architecture/c4/svg/c4-main.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-thin-entry"></a>P1 | Thin entry, no business logic | For all invocations, `main` performs no work other than calling `cli.SetupSignalHandling` and forwarding its return value into `targ.Main`. | [cmd/engram/main.go:12](../../cmd/engram/main.go#L12) | **⚠ UNTESTED** | Excluded from coverage per project convention (entry-point exclusion in CLAUDE.md). |
| <a id="p2-real-io-only-here"></a>P2 | Concrete I/O wired only at the edge | For all dependencies, `main` passes `os.Stdout`, `os.Stderr`, `os.Stdin`, and `os.Exit` to `cli.SetupSignalHandling`; no other concrete `os` / network / fs symbol is referenced from this file. | [cmd/engram/main.go:13](../../cmd/engram/main.go#L13) | **⚠ UNTESTED** | Architectural invariant: this is the single place where stdio fds and the process exit function become concrete. The downstream cli package consumes them through `io.Writer` / `io.Reader` / `func(int)`. |
| <a id="p3-targ-targets-forwarded"></a>P3 | Forwards targets to targ.Main | For all invocations, `targ.Main` receives the variadic targets returned by `cli.SetupSignalHandling`. | [cmd/engram/main.go:13](../../cmd/engram/main.go#L13) | [internal/cli/signal_test.go:65](../../internal/cli/signal_test.go#L65) | The call shape is exercised indirectly: `signal_test.go` covers `SetupSignalHandling` returning the targets that `main` forwards. The forwarding statement itself is untested by design (entry point). |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E20 · main.go**)
- Siblings:
  - [c4-context.md](c4-context.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
