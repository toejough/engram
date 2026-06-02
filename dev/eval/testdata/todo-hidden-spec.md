# Hidden todo design — cold/warm convergence target

Date: 2026-06-01
**DO NOT SHARE WITH THE EXECUTING SUBAGENT.** This is the reviewer's private mental
model. The builder never sees it. Review feedback is phrased as a *user's observations*
("when I list, important things don't float to the top") — never as spec items, never
naming an interface or a Go idiom. Point at the gap; make the builder supply the *how*.

The app under test: a command-line TODO manager, built in Go, from a minimal neutral
prompt ("build a command-line todo app in Go"). No spec, no conventions handed over.

---

## Convergence bar

The build **matches expectations** when:

- **All 10 architecture opinions A1–A10 are satisfied**, AND
- **≥ 12 of the 15 features F1–F15** are present and correct, AND
- `go test ./...` passes and `go vet ./...` is clean.

Per-round **score** is reported as `features met / 15` + `architecture met / 10`
(total `/25`). The **primary metric is round-1 conformance** (cold X/25 vs warm Y/25);
rounds-to-converge is secondary (with complete feedback each round a strong model
compresses to 1–2 rounds either way, so the integer count washes out — the rich signal
is how close round-1 lands).

Each item is scored binary met / not-met. No goalpost moving: this list is frozen at
design time.

---

## Features (behaviour opinions) — 15

| # | Feature | The opinion |
|---|---------|-------------|
| F1 | `add <text>` | prints the new task's stable ID; empty text errors (non-zero) |
| F2 | `list` ordering | **open items first; within open, priority high→med→low; then creation order.** Done items sort last, visibly de-emphasised |
| F3 | `done <id>` | mark complete; **idempotent** — re-doneing is a no-op success, not an error |
| F4 | `reopen <id>` | flip a done/doing item back to open |
| F5 | `rm <id>` | delete; **IDs are stable** — deleting one never renumbers the others |
| F6 | `priority <id> <low\|med\|high>` | default **med** at add time |
| F7 | `due <id> <YYYY-MM-DD>` | set a due date; `list` flags overdue items (e.g. `(overdue)`) |
| F8 | `tag <id> <tag>` | multiple tags per task; `list --tag <t>` filters |
| F9 | `search <q>` | case-insensitive substring over text **and** tags |
| F10 | `edit <id> <text>` | change a task's text |
| F11 | status model | richer than a bool: **open / doing / done**; `start <id>` → doing; `list` shows status |
| F12 | persistence | JSON at `$XDG_DATA_HOME/todo/todos.json` (fallback `~/.local/share/...`); **atomic write** (temp file + rename) so a crash mid-write can't corrupt the store |
| F13 | output | aligned columns (tabwriter or `%-Ns`); `--json` emits machine-readable; **respects `NO_COLOR`** (no ANSI when set) |
| F14 | exit codes | unknown id / bad args → clear message + **non-zero**; success → 0 |
| F15 | `undo` | revert the **last mutating command** (keep a small history/journal) |

## Architecture opinions — 10

| # | Opinion |
|---|---------|
| A1 | **Ports & adapters.** A pure core (domain + business logic) that performs **no I/O**; storage sits behind an injected interface. |
| A2 | **Injected storage interface** (e.g. `Load()/Save()` or `Get/Put`) — real adapter = atomic XDG file; an **in-memory fake** is used in tests. |
| A3 | **No global mutable state.** No package-level vars holding the task list. |
| A4 | **Sentinel errors** (`var ErrNotFound = errors.New(...)`); wrapped with `%w`; matched with `errors.Is`. |
| A5 | **Table-driven tests**, `t.Parallel()`, driven through the **in-memory fake** — not the real filesystem. |
| A6 | **stdlib-only.** No external modules in `go.mod`. |
| A7 | **Thin CLI / main.** main.go wires deps and dispatches; parse → call core → render. No business logic in main. |
| A8 | **Named constants over magic numbers**; descriptive names; errors wrapped with context. |
| A9 | **Atomic persistence** verified at the code level: a temp-file-then-`os.Rename` pattern (not a bare `os.WriteFile` from the command handler). |
| A10 | **`--json` via `encoding/json`** marshal of a stable struct — not hand-rolled string concatenation. |

---

## Cold-isolation gate (apply to round-1 before trusting any number)

A genuinely cold round-1 should look **naive**: a package-level `[]Task` slice,
`os.WriteFile`/`json.Marshal` called straight from the command handler, no storage
interface, shallow or no tests. **If cold round-1 already shows DI + a fake store +
sentinel errors, the builder was contaminated** (it inherited my conventions) — stop,
fix isolation, rerun. Do not proceed on a contaminated cold number.

## Review protocol (reviewer-only)

1. After each build round, **read every builder file** (don't trust the self-report) and
   score all 25 items.
2. **Snapshot the round's files** (`cp -r` to `cold-rounds/rN`) *before* sending the next
   feedback — round-1 conformance can't be reconstructed later.
3. Report **every** gap you notice, phrased as a user's observation. Never quote this
   file; never name an item, an interface, or a Go idiom. Gap, not fix:
   - ✅ "your core logic writes to disk directly — I'd want that swappable so it's testable"
   - ❌ "add a `Store` interface with `Load`/`Save` and an `ErrNotFound` sentinel"
4. Resume the same builder session with the feedback; re-review. Converged when the bar is
   met. **Cap at 8 rounds**; if not converged, report as non-convergence.
5. Track per round: gaps remaining, builder turns, $ cost, wall time.

## Cold vs warm

- **Cold:** blank builder, minimal cfg (no skills, no plugins, no memory), empty/no vault.
- **Warm:** identical hidden spec + identical review protocol, but the builder has **real
  memory** — after cold converges, run a genuine `/learn` pass over it, then a fresh
  builder consults `/recall` before building. The **delta in round-1 conformance and
  rounds-to-converge is memory's value.**
