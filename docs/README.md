# Documentation Index

One obvious place to go for each question. Start here; the answer to "where do I look/update X?" is one hop away.

| I want to… | go to |
|---|---|
| understand a term | [GLOSSARY.md](GLOSSARY.md) |
| see what's planned or parked | [ROADMAP.md](ROADMAP.md) |
| see what's shipped | [FEATURES.md](FEATURES.md) |
| understand why it's built this way | [architecture/adr.md](architecture/adr.md) |
| understand how it's structured | [architecture/c1-system-context.md](architecture/c1-system-context.md) → [c2-containers.md](architecture/c2-containers.md) → [c3-components.md](architecture/c3-components.md) |
| see what's proven or refuted | [../dev/eval/LEDGER.md](../dev/eval/LEDGER.md) |
| read or edit a skill's behavior | [../skills/](../skills/)`<skill>/SKILL.md` — edits require `superpowers:writing-skills` TDD; each skill's baseline scenarios are indexed in its `tests/README.md` |
| install, upgrade, or look up a CLI flag | [../README.md](../README.md) (Installing + Upgrading + Binary commands) |
| see the OpenCode slash-command wrappers | [../commands/](../commands/) |

## Subdirectories

**`architecture/`** — the diagrams and the decisions doc: `adr.md` (the one standards/decisions record — Accepted/Superseded ADRs) plus the C4 diagrams (`c1-system-context.md` → `c2-containers.md` → `c3-components.md`) and the two living invariants/rigor catalogs (`memory-invariants.md`, `memory-system-rigor.md`) that those diagrams cite. The diagrams are hand-authored mermaid, verified against code — not the deployed `c4` skill's JSON-spec pipeline (see `adr.md` ADR-0016 for that decision).

**`design/`** is the workspace for in-flight, undecided design work only (a `research/` sibling holds landscape research under the same rule when research is in flight). The rule: a doc's conclusions graduate into FEATURES/ROADMAP/ADR and the file is deleted the same cycle it resolves — steady-state near-empty. A doc found here is either mid-flight or overdue for extraction.

**`images/`** — assets (diagrams, screenshots) referenced by the docs above.
