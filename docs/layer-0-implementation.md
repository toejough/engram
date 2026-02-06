# Layer 0 Implementation Summary

High-level summary of Layer 0 (Foundation) commands for the projctl orchestration system.

---

## Overview

Layer 0 provides the foundational infrastructure for the orchestration system without agent spawning. These commands enable state management, context serialization, ID generation, traceability validation, and semantic memory.

---

## Command Inventory

### State Commands

Manage project state and workflow progression.

| Command | Status | Description |
|---------|--------|-------------|
| `projctl state get` | EXISTING | Get current project state |
| `projctl state transition` | EXISTING | Transition to a new state |
| `projctl state next` | EXISTING | Determine next action based on current state |
| `projctl state init` | EXISTING | Initialize project state file |
| `projctl state set` | EXISTING | Update state fields without transitioning |
| `projctl state retry` | EXISTING | Re-attempt the last failed transition |
| `projctl state recovery` | EXISTING | Show recovery options after failure |
| `projctl state complete` | EXISTING | Mark a task as complete |
| `projctl state pair set` | EXISTING | Set pair loop state for a phase or task |
| `projctl state pair clear` | EXISTING | Clear pair loop state |
| `projctl state yield set` | EXISTING | Set pending yield state |
| `projctl state yield clear` | EXISTING | Clear pending yield |

### Context Commands

Manage skill dispatch context files and result collection.

| Command | Status | Description |
|---------|--------|-------------|
| `projctl context write` | ENHANCED | Write context file with `yield_path` for parallel execution |
| `projctl context read` | EXISTING | Read context or result file |
| `projctl context write-parallel` | EXISTING | Create context files for multiple tasks |
| `projctl context check` | EXISTING | Check context budget usage |

The `context write` command now includes an `output.yield_path` field that provides a unique path for each skill invocation, enabling parallel execution without file conflicts. See [context.md](commands/context.md) for details.

### ID Commands

Generate sequential IDs for traceability artifacts.

| Command | Status | Description |
|---------|--------|-------------|
| `projctl id next` | EXISTING | Get next sequential ID (REQ, DES, ARCH, TASK) |

### Trace Commands

Validate and repair traceability chains across artifacts.

| Command | Status | Description |
|---------|--------|-------------|
| `projctl trace validate` | EXISTING | Validate traceability chain completeness |
| `projctl trace repair` | EXISTING, DOCUMENTED | Auto-fix duplicates, escalate dangling references |
| `projctl trace show` | EXISTING | Visualize the traceability graph |
| `projctl trace promote` | EXISTING | Promote TASK traces to upstream IDs |

See [trace.md](commands/trace.md) for comprehensive documentation.

### Territory Commands

Map and display codebase structure.

| Command | Status | Description |
|---------|--------|-------------|
| `projctl territory map` | EXISTING | Generate compressed territory map |
| `projctl territory show` | EXISTING | Show current cached territory map |

### Memory Commands

Semantic memory system with local embedding generation.

| Command | Status | Description |
|---------|--------|-------------|
| `projctl memory query` | NEW | Semantic search using ONNX embeddings |
| `projctl memory learn` | NEW | Store arbitrary insights with embeddings |
| `projctl memory grep` | EXISTING | Structural search (no ONNX, just grep) |
| `projctl memory extract` | NEW | Extract decisions/learnings from yield/result files |
| `projctl memory session-end` | EXISTING | Generate compressed session summary |
| `projctl memory decide` | EXISTING | Log a decision with reasoning and alternatives |

---

## Architecture Summary

### Embedding Engine

Local semantic search using ONNX runtime - no external API calls required.

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Runtime | ONNX | Cross-platform, no Python dependency |
| Model | all-MiniLM-L6-v2 | 384-dimension embeddings, ~90MB |
| Storage | SQLite-vec | Single file, no server, vector search built-in |

The embedding system auto-downloads required components on first use:
- ONNX Runtime library (~20MB, platform-specific)
- Sentence transformer model (~90MB from HuggingFace)

### Storage Layout

```
~/.projctl/memory/
├── index.md              # Human-readable learnings (grep-able)
├── embeddings.db         # SQLite-vec for semantic search
├── sessions/
│   └── <project>-<date>.md
└── decisions/
    └── <project>.jsonl
```

---

## Key Patterns

### Yield Path Generation

The `context write` command generates unique yield paths for each skill invocation:

```
.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
```

Components:
- **projectUUID**: Stable project identifier
- **fileUUID**: Unique per-invocation identifier

This pattern ensures uniqueness even when:
- Same task is invoked multiple times (retries)
- Multiple tasks run in parallel with same timestamp
- Multiple invocations occur within the same second

### Auto-Download

ONNX runtime and model files are downloaded automatically on first use:
1. Check if files exist in `~/.projctl/models/`
2. If missing, download from official sources (GitHub releases, HuggingFace)
3. Extract and cache for future use

### Parallel Safety via UUID

Parallel execution is supported through:
1. **Unique yield paths**: Each invocation writes to a distinct file
2. **UUID components**: Both project and file UUIDs ensure no collisions
3. **Atomic writes**: Temp file + rename pattern for safe file creation

---

## Documentation

| Topic | Document |
|-------|----------|
| Context commands | [commands/context.md](commands/context.md) |
| Trace commands | [commands/trace.md](commands/trace.md) |
| Memory commands | [commands/memory.md](commands/memory.md) |
