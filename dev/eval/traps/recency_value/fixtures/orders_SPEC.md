# orders

A small orders command-line tool, invoked as `./orders <cmd> [args]` from this directory. Any
language is fine (a shebang script or a compiled binary) as long as `./orders <cmd>` runs it
directly (an `./orders` executable or wrapper).

## `report <orders.db.json>`

Reads the given orders JSON — an array of `{"id", "customer", "amt"}` records where `amt` is a
dollar amount — and prints the total revenue across all orders, in dollars, on one line to stdout,
in exactly this form:

    total revenue: $<X>

for example `total revenue: $1234.56`. Invoke it exactly as:

    ./orders report orders.db.json
