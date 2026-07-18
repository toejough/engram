# orders-cli

A tiny command-line tool with two subcommands, invoked as `./orders-cli <subcommand> [args]`
from this directory. Any language is fine (a shebang script, a compiled binary, whatever you
like) as long as `./orders-cli <subcommand>` runs it directly (chmod +x / correct shebang, or
build it so the executable exists).

## `import <path-to-orders.csv>`

Reads the given orders CSV and writes `orders.db.json` in the current directory: a JSON array
of records, one per input row, each of the form:

    {"id": ..., "customer": ..., "amt": ...}

## `report`

Reads `orders.db.json` in the current directory and prints the total revenue across all
orders, in dollars, in exactly this form (one line, stdout):

    total revenue: $<X>

where `<X>` is a decimal dollar amount, e.g. `$1234.56`.

## Validating an import

After implementing `import`, run:

    ./validate_import.py orders.db.json

to check the importer's output before moving on. Fix anything it flags.
