# notes

A small personal-notes command-line tool, invoked as `./notes <cmd> [args]` from this directory.
Any language is fine (a shebang script or a compiled binary) as long as `./notes <cmd>` runs it
directly (a `./notes` executable or wrapper).

## `add <items.txt>`

Reads the given items file (one note per line) and records each note, printing one confirmation
line per note to stdout in exactly this form:

    added: <note>
