---
level: 4
name: main
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 658e2ee3
---

# C4 — main (Property/Invariant Ledger)

> Component in focus: **S2-N3-M1 · main.go**.
> Source files in scope:
> - [cmd/engram/main.go](cmd/engram/main.go)

## Context (from L3)

main.go is the process entry point for the engram CLI binary. It contains no business logic: it wires real-OS I/O streams (`os.Stdout`, `os.Stderr`, `os.Stdin`) and `os.Exit` into `cli.SetupSignalHandling`, then forwards the resulting target list into `targ.Main`. By project convention it is excluded from coverage — only re-exports and thin wrappers, no testable logic. All command dispatch, signal-handler installation, and DI wiring is delegated to the `cli` package; this file's only architectural job is to be the boundary at which concrete OS handles enter the program.

![C4 main context diagram](svg/c4-main.svg)

> Diagram source: [svg/c4-main.mmd](svg/c4-main.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-main.mmd -o architecture/c4/svg/c4-main.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R-edges between the same node pair.

**Legend:**
- **focus** (yellow): the file in scope for this ledger.
- **component** (light blue): peer Go components inside the binary.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n3-m1-p1-single-entry-point"></a>S2-N3-M1-P1 | Single entry point | For all process invocations of the engram binary, control enters at `main.main` and exits only when `targ.Main` returns or a forced-exit signal handler calls `os.Exit`. | [cmd/engram/main.go:12](../../cmd/engram/main.go#L12) | **⚠ UNTESTED** | main is excluded from coverage by project convention; entry behavior is exercised by every CLI integration run. |
| <a id="s2-n3-m1-p2-concrete-i-o-wired-at-the-edge"></a>S2-N3-M1-P2 | Concrete I/O wired at the edge | For all process invocations, the standard I/O handles supplied to the CLI are exactly `os.Stdout`, `os.Stderr`, and `os.Stdin` — no other code path injects real-OS streams into the binary. | [cmd/engram/main.go:13](../../cmd/engram/main.go#L13) | **⚠ UNTESTED** | Honors the project's DI-at-the-edge rule: `internal/` packages never call `os.*` directly; the only `os.Stdout/Stderr/Stdin` reference is here. |
| <a id="s2-n3-m1-p3-real-os-exit-is-the-terminator"></a>S2-N3-M1-P3 | Real os.Exit is the terminator | For all process invocations, the exit-function passed into the signal-handling setup is `os.Exit` — second-signal force-exit terminates the real process, not a stubbed one. | [cmd/engram/main.go:13](../../cmd/engram/main.go#L13) | **⚠ UNTESTED** | Tests cover `SetupSignalHandling` with a stub `exitFn` (signal_test.go); the binding of the real `os.Exit` here is intentionally untested at the unit layer. |
| <a id="s2-n3-m1-p4-signal-handlers-installed-before-dispatch"></a>S2-N3-M1-P4 | Signal handlers installed before dispatch | For all process invocations, `cli.SetupSignalHandling` is invoked before `targ.Main`, so SIGINT/SIGTERM handlers and the force-exit goroutine are active for the entire lifetime of subcommand execution. | [cmd/engram/main.go:13](../../cmd/engram/main.go#L13) | **⚠ UNTESTED** | Ordering is a single-expression invariant: `SetupSignalHandling(...)` evaluates and returns before `targ.Main` receives its arguments. |
| <a id="s2-n3-m1-p5-no-business-logic-in-entry-point"></a>S2-N3-M1-P5 | No business logic in entry point | For all changes to `cmd/engram/main.go`, the file contains only the entry-point shell: package declaration, imports, and a `main` function that delegates to `cli` and `targ` — no command logic, parsing, or I/O is performed here. | [cmd/engram/main.go:12](../../cmd/engram/main.go#L12) | **⚠ UNTESTED** | Enforced by convention and the L3 catalog row for E20 ("No business logic; excluded from coverage per project convention."). Drift would show up as new statements inside `main`. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **S2-N3-M1 · main.go**)
- Siblings:
  - [c4-anthropic.md](c4-anthropic.md)
  - [c4-cli.md](c4-cli.md)
  - [c4-context.md](c4-context.md)
  - [c4-externalsources.md](c4-externalsources.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-recall.md](c4-recall.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)
  - [c4-tomlwriter.md](c4-tomlwriter.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

