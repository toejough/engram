# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

engram does not use the `CONTEXT.md`/`docs/adr/` convention these skills default to. It has its
own established documentation system — read these instead.

## Before exploring, read these

- **`docs/README.md`** — the documentation index; one hop to whichever doc answers "where do I
  look/update X?"
- **`docs/architecture/adr.md`** — the ADR authority: one running file of Accepted/Superseded
  decisions (ADR-0001, ADR-0002, ...), not one-file-per-ADR under `docs/adr/`.
- **`docs/architecture/c1-system-context.md` → `c2-containers.md` → `c3-components.md`** — the
  C4 architecture diagrams, in that reading order.
- **`openspec/specs/`** — the PRIMARY behavior record (what's actually shipped), one `spec.md`
  per capability. Read the relevant capability's spec before touching that area.
- **`docs/GLOSSARY.md`** — term definitions.
- **`CLAUDE.md`** at the repo root — design principles, code quality rules, and workflow
  conventions specific to this repo.

If any of these files don't exist, proceed silently — don't flag their absence.

## File structure

Single-context repo:

```
/
├── CLAUDE.md
├── docs/
│   ├── README.md              ← index; start here
│   ├── architecture/
│   │   ├── adr.md             ← the ADR authority (one running file)
│   │   ├── c1-system-context.md
│   │   ├── c2-containers.md
│   │   └── c3-components.md
│   └── GLOSSARY.md
├── openspec/
│   └── specs/                 ← primary behavior record, one dir per capability
└── internal/, cmd/            ← Go source
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis,
a test name), use the term as defined in `docs/GLOSSARY.md`. Don't drift to synonyms the
glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing
language the project doesn't use (reconsider) or there's a real gap worth noting.

## Flag ADR conflicts

If your output contradicts an existing ADR in `docs/architecture/adr.md`, surface it explicitly
rather than silently overriding:

> _Contradicts ADR-0013 (vault flock + atomic-rename write safety), but worth reopening
> because…_

## Flag spec conflicts

If your output contradicts an existing requirement in `openspec/specs/<capability>/spec.md`,
surface it explicitly the same way — a behavior change goes through an OpenSpec change
(`openspec/changes/`, propose → implement → archive) rather than a silent code edit against
already-shipped behavior.
