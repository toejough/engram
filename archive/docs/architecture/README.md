# Architecture

C4 model diagrams for engram, from system context down to code-level entities.

## Diagram Index

| Level | Diagram | Shows |
|-------|---------|-------|
| [C4: Context](c4-context.md) | System context | Engram + external actors (user, Claude Code, filesystem) |
| [C3: Container](c3-container.md) | Containers | API server, MCP server, CLI client, hooks, chat file, memory files |
| [C2: Component](c2-component.md) | Components | Inside each container: handlers, goroutines, parsers, tools |
| [C1: Code](c1-code.md) | Entities (ERD) | Key types and their relationships |
| [Sequences](sequences.md) | Interaction flows | How data flows between boundaries at each level |

Each diagram cross-links to the level above and below it.
