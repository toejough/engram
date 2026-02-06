# Memory Commands

The `projctl memory` commands manage semantic memory storage and retrieval for cross-session learning.

## Overview

The memory system provides two search paradigms:

| Approach | Command | When to Use |
|----------|---------|-------------|
| **Semantic search** | `memory query` | Finding conceptually related content regardless of exact wording |
| **Pattern search** | `memory grep` | Finding content with specific keywords or phrases |

**Semantic search** uses embeddings to find memories that are conceptually similar to your query, even if they use different words. For example, querying "build performance" might return memories about "CI optimization" or "compilation speed."

**Pattern search** uses literal text matching. Use it when you know the exact terms you're looking for or need to find all occurrences of a specific word.

## Commands

| Command | Purpose |
|---------|---------|
| `projctl memory query` | Semantic search across memories |
| `projctl memory learn` | Store arbitrary insights |
| `projctl memory decide` | Store decisions with context and reasoning |
| `projctl memory extract` | Extract and embed insights from legacy message/result files |
| `projctl memory grep` | Pattern-based search across memories |
| `projctl memory session-end` | Generate end-of-session summary |

---

## memory query

Performs semantic similarity search across all stored memories using embeddings.

### Usage

```bash
projctl memory query "<text>" [-n <limit>]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `<text>` | Text to search for (required) |

### Flags

| Flag | Description |
|------|-------------|
| `-n, --limit` | Maximum number of results (default: 5) |
| `--memory-root` | Memory root directory (default: ~/.claude/memory) |

### Output

```
1. (0.87) Previous project: Gradle cache reduced CI from 8min to 2min
2. (0.82) Learned: Incremental compilation key for Go builds
3. (0.76) Decision: Use build cache for dependency resolution
```

Each result shows:
- Rank number
- Similarity score (0.0 to 1.0, higher is more similar)
- Memory content

### Examples

```bash
# Find memories related to build performance
projctl memory query "build performance patterns"

# Search for testing-related insights with more results
projctl memory query "test organization strategies" -n 10

# Find architecture decisions
projctl memory query "API design choices"
```

---

## memory learn

Stores an arbitrary insight in semantic memory.

### Usage

```bash
projctl memory learn -m "<message>" [-p <project>]
```

### Flags

| Flag | Description |
|------|-------------|
| `-m, --message` | Learning message to store (required) |
| `-p, --project` | Project to tag the learning with |
| `--memory-root` | Memory root directory (default: ~/.claude/memory) |

### Output

```
Learned: GraphQL adds complexity we don't need
```

### Storage

Learnings are stored in `~/.claude/memory/index.md` in a human-readable format:

```
- 2026-02-04 14:30: [myproject] GraphQL adds complexity we don't need
- 2026-02-04 15:45: Table-driven tests work better for validation logic
```

The message is also embedded and stored in `embeddings.db` for semantic search.

### Examples

```bash
# Store a general learning
projctl memory learn -m "Table-driven tests work better for validation logic"

# Store a project-specific learning
projctl memory learn -m "The config loader expects YAML, not JSON" -p myproject

# Store a lesson learned from debugging
projctl memory learn -m "Race conditions often hide in deferred cleanup functions"
```

---

## memory decide

Logs a decision with full context, reasoning, and alternatives considered.

### Usage

```bash
projctl memory decide -c "<context>" --choice "<choice>" -r "<reason>" -p <project> [-a "<alternatives>"]
```

### Flags

| Flag | Description |
|------|-------------|
| `-c, --context` | Decision context (required) |
| `--choice` | The choice made (required) |
| `-r, --reason` | Reason for the decision (required) |
| `-p, --project` | Project name (required) |
| `-a, --alternatives` | Comma-separated alternatives considered |
| `--memory-root` | Memory root directory (default: ~/.claude/memory) |

### Output

```
Decision logged to: /Users/joe/.claude/memory/decisions/2026-02-04-myproject.jsonl
```

### Storage

Decisions are stored as JSON Lines in `~/.claude/memory/decisions/{date}-{project}.jsonl`:

```json
{"timestamp":"2026-02-04T14:30:00Z","context":"Build system choice","choice":"Make with mage helpers","reason":"Simpler than Bazel, team already knows Make","alternatives":["Bazel","Just"]}
```

### Examples

```bash
# Log a technology decision
projctl memory decide \
  -c "Build system choice" \
  --choice "Make with mage helpers" \
  -r "Simpler than Bazel, team already knows Make" \
  -p myproject \
  -a "Bazel, Just"

# Log an architecture decision
projctl memory decide \
  -c "Database selection for user data" \
  --choice "PostgreSQL" \
  -r "Strong consistency requirements, team expertise" \
  -p userservice \
  -a "MongoDB, DynamoDB, CockroachDB"

# Log a design decision
projctl memory decide \
  -c "API versioning strategy" \
  --choice "URL path versioning" \
  -r "Explicit, easy to route, industry standard" \
  -p api-gateway \
  -a "Header versioning, Query parameter"
```

---

## memory extract

Extracts decisions and learnings from legacy orchestration message/result files and stores them in semantic memory.

### Usage

```bash
projctl memory extract --result <path>
projctl memory extract --message <path>
```

### Flags

| Flag | Description |
|------|-------------|
| `-r, --result` | Path to result.toml file (mutually exclusive with --message) |
| `-m, --message` | Path to legacy message.toml file (mutually exclusive with --result) |
| `--memory-root` | Memory root directory (default: ~/.claude/memory) |
| `--model-dir` | Model directory (default: ~/.claude/models) |

### Output

The command outputs TOML to stdout for orchestrator consumption:

```toml
[result]
status = "success"
file_path = "/path/to/pm-result.toml"
items_extracted = 5
storage_location = "/Users/joe/.claude/memory/embeddings.db"

[result.breakdown]
decision = 3
learning = 2
```

Terminal output (to stderr):

```
Extracted 5 items from pm-result.toml
Breakdown:
  - 3 decisions
  - 2 learnings

Stored in semantic memory (/Users/joe/.claude/memory/embeddings.db)
```

### Extracted Item Types

From **legacy message files**, the command extracts:
- `summary` - Overview from payload.summary
- `finding` - Items from payload.findings array
- `learning` - Items from payload.learnings array

From **result files**, the command extracts:
- `decision` - Each entry from the decisions array

### Examples

```bash
# Extract from a result file after PM phase completes
projctl memory extract --result .claude/context/pm-result.toml

# Extract from a legacy message file
projctl memory extract --message .claude/context/design-message.toml
```

---

## memory grep

Pattern-based search across memory files. Unlike `memory query`, this performs literal text matching.

### Usage

```bash
projctl memory grep <pattern> [-p <project>] [-d]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `<pattern>` | Pattern to search for (required) |

### Flags

| Flag | Description |
|------|-------------|
| `-p, --project` | Limit search to specific project |
| `-d, --include-decisions` | Also search decisions files |
| `--memory-root` | Memory root directory (default: ~/.claude/memory) |

### Output

```
/Users/joe/.claude/memory/index.md:5: - 2026-02-04 14:30: [myproject] caching reduces build time
/Users/joe/.claude/memory/sessions/myproject-2026-02-04.md:12: Implemented caching layer for API responses
```

Each match shows: `file:line: content`

### Examples

```bash
# Search all memories for "caching"
projctl memory grep "caching"

# Search only project-specific memories
projctl memory grep "authentication" -p userservice

# Include decision files in search
projctl memory grep "PostgreSQL" -d

# Search for error-related learnings
projctl memory grep "error handling"
```

---

## memory session-end

Generates a compressed summary of the current session and stores it in semantic memory.

### Usage

```bash
projctl memory session-end -p <project>
```

### Flags

| Flag | Description |
|------|-------------|
| `-p, --project` | Project name (required) |
| `--memory-root` | Memory root directory (default: ~/.claude/memory) |

### Output

```
Session summary saved to: /Users/joe/.claude/memory/sessions/2026-02-04-myproject.md
```

### Generated Summary

The command generates a markdown summary (max 2000 characters):

```markdown
# Session Summary

**Project:** myproject
**Date:** 2026-02-04

## Decisions

- **Make with mage helpers**: Simpler than Bazel, team already knows...
- **PostgreSQL**: Strong consistency requirements, team expertise
- **URL path versioning**: Explicit, easy to route, industry standard

... and 2 more decisions
```

The summary is embedded and stored for semantic search.

### Example

```bash
# Generate end-of-session summary
projctl memory session-end -p myproject
```

---

## Architecture

### Embedding Model

| Property | Value |
|----------|-------|
| Model | e5-small-v2 (all-MiniLM-L6-v2 compatible) |
| Dimensions | 384 |
| Size | ~90MB |
| Runtime | ONNX Runtime |

The system uses a local ONNX model for embedding generation - no API calls are made. This ensures:
- Fast inference (~50ms per embedding)
- No network dependency
- No API costs
- Privacy (data never leaves your machine)

### Storage

| Component | Location |
|-----------|----------|
| Embeddings database | `~/.claude/memory/embeddings.db` |
| Human-readable index | `~/.claude/memory/index.md` |
| Session summaries | `~/.claude/memory/sessions/` |
| Decision logs | `~/.claude/memory/decisions/` |

The embeddings database uses **SQLite-vec**, a SQLite extension for vector similarity search. This provides:
- Single-file storage (no server required)
- Built-in vector similarity search
- ACID compliance
- Cross-platform compatibility

### Supported Platforms

| Platform | Status |
|----------|--------|
| macOS (arm64) | Supported |
| macOS (x86_64) | Supported |
| Linux (x86_64) | Supported |
| Windows | Future work |

### First-Use Behavior

On first use of `memory query` or `memory extract`, the system automatically downloads required components:

1. **ONNX Runtime library** (~15MB compressed)
   - Downloaded from GitHub releases
   - Platform-specific binary selected automatically
   - Extracted to `~/.claude/models/`

2. **e5-small embedding model** (~90MB)
   - Downloaded from HuggingFace
   - Stored as `~/.claude/models/e5-small-v2.onnx`

**Expected first-run output:**

```
Downloading ONNX Runtime for darwin/arm64...
  [====================================] 100% (15.2 MB)
Downloading e5-small-v2 model...
  [====================================] 100% (89.7 MB)
Model ready.

1. (0.87) Previous project: Gradle cache reduced CI from 8min to 2min
```

Subsequent runs use the cached components with no download delay.

---

## Examples

### Real-World Usage Scenarios

**Scenario 1: Starting a new task**

Before beginning work, query memory for relevant past learnings:

```bash
# Find relevant past decisions about similar work
projctl memory query "user authentication implementation"

# Check for specific patterns
projctl memory grep "JWT" -d
```

**Scenario 2: Recording a learning during development**

When you discover something important:

```bash
projctl memory learn -m "OAuth refresh tokens need 2x the access token lifetime" -p auth-service
```

**Scenario 3: Documenting a decision**

After making an important architectural choice:

```bash
projctl memory decide \
  -c "Token storage strategy" \
  --choice "HTTP-only cookies" \
  -r "More secure than localStorage, prevents XSS access to tokens" \
  -p auth-service \
  -a "localStorage, sessionStorage, in-memory"
```

**Scenario 4: End of session wrap-up**

Before ending a coding session:

```bash
# Generate session summary
projctl memory session-end -p auth-service

# Extract learnings from completed phase
projctl memory extract --result .claude/context/impl-result.toml
```

**Scenario 5: Orchestrator context injection**

The orchestrator automatically queries memory before spawning agents:

```bash
# Orchestrator runs this internally
projctl memory query "build performance" -n 3
```

And injects results into agent context:

```toml
[memory]
relevant = [
    "Previous project: Gradle cache reduced CI from 8min to 2min",
    "Learned: Incremental compilation key for Go builds",
]
query = "build performance"
```

