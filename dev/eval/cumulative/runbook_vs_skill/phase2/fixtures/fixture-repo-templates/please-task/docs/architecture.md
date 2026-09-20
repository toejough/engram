# Architecture

`tally` is a single module. The flow:

```mermaid
flowchart LR
    F[input file] --> C[count_words]
    C --> R[render]
    R -->|stdout| S[terminal]
    R -->|--out| O[report file]
```

`build_parser` owns every flag; `main` wires parsing, counting, and output. When `--out` is
given the report is written to that path, otherwise it goes to stdout.
