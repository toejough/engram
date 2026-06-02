# Contacts build — scoring rubric (layer experiment)

Score each item 0/1 per build. Two buckets, scored separately.

The **memories are todo-derived**, so only Bucket A (conventions) can transfer.
Bucket B (contacts features) is a **control** — it measures baseline build
competence, which should be roughly constant across arms; if it isn't, the
arms differ in agent effort, not memory.

## Bucket A — Architecture & cross-cutting conventions (TRANSFER TARGET) — /10

1. **DI**: `Store`, `Clock`, and an output `io.Writer` injected into an `App`
   struct via a constructor; real impls wired only in `main()`.
2. **Pure core / thin shell**: ~5-file split (types, store, core, display,
   main); core business logic makes zero direct `os.*` / `time.Now()` calls.
3. **Table-driven unit tests** using an in-memory store + fake clock +
   `bytes.Buffer` — no real filesystem or wall-clock in unit tests.
4. **Sentinel errors**: package-level `var Err… = errors.New(...)`, returned
   wrapped with `%w`; callers use `errors.Is`.
5. **No global mutable state**; all state lives in the constructed `App`.
6. **Stdlib-only**, single binary, hand-rolled subcommand dispatch
   (`map[string]cmdFunc` or switch), no third-party CLI/color deps.
7. **Atomic XDG persistence**: store at `$XDG_DATA_HOME/<app>/<file>.json`
   (fallback `~/.local/share/...`), atomic write (temp file + rename).
8. **Output**: default aligned human table; `--json` machine output available.
9. **Color** for emphasis, auto-disabled when not a TTY or `NO_COLOR` is set.
10. **Conventional command-set shape**: add / list / show|view / edit / rm /
    search present as subcommands.

## Bucket B — Contacts-specific features (CONTROL, not memory signal) — /7

1. Contact fields: name + email + phone (+ optional tags/notes/id).
2. `add` creates and persists a contact.
3. `list` shows contacts in a sensible order.
4. `show`/`view <id>` displays one contact.
5. `edit <id>` updates fields.
6. `rm <id>` deletes a contact.
7. `search <q>` matches across fields (name/email/phone).

## Headline
- **Architecture score /10** per arm (mean across trials) is the transfer signal.
- Report Bucket B /7 alongside as the control.
- Also record: did recall fire + what it surfaced; turns; cost.

(Derived from the todo 18-item rubric in
`docs/superpowers/specs/2026-05-30-cold-warm-todo-test.md` and the L2/L3 notes
in `dev/eval/testdata/layer-vaults/{l2,l3}/`.)
