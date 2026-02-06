---
name: context-explorer
description: Gathers context from multiple query sources, returns aggregated results
context: fork
model: sonnet
skills: ownership-rules
user-invocable: false
role: standalone
---

# Context Explorer

Standalone skill that executes queries from need-context yields and returns aggregated context to the requesting producer.

---

## Purpose

When a producer yields `need-context` with a list of queries, the orchestrator dispatches this skill to gather the requested information. Results are returned via `complete` yield with aggregated context.

---

## Input

Receives queries from a producer's need-context yield:

```toml
[[payload.queries]]
type = "file"
path = "docs/requirements.md"

[[payload.queries]]
type = "memory"
query = "caching patterns"

[[payload.queries]]
type = "territory"
scope = "tests"

[[payload.queries]]
type = "web"
url = "https://example.com/docs"
prompt = "Extract the API format"

[[payload.queries]]
type = "semantic"
question = "How does authentication work in this codebase?"
```

---

## Query Types

| Type | Parameters | Tool | Description |
|------|------------|------|-------------|
| `file` | `path` | Read tool | Read file contents directly |
| `memory` | `query` | projctl memory | Semantic memory search via ONNX embeddings |
| `territory` | `scope` | projctl territory | Codebase structure map |
| `web` | `url`, `prompt` | WebFetch tool | Fetch and interpret URL content |
| `semantic` | `question` | Task tool (explore) | LLM-based code exploration |

---

## Workflow

### 1. Parse Queries

Read query list from input context. Validate each query has required fields for its type.

### 2. Execute Queries (Parallel)

Use Task tool to parallelize independent queries:

```markdown
For queries that can run in parallel:
- file queries: Batch read via parallel Read tool calls
- memory queries: Execute projctl memory query
- territory queries: Execute projctl territory map
- web queries: Batch fetch via parallel WebFetch tool calls
- semantic queries: Spawn explore agents via Task tool
```

**Parallelization Strategy:**
- Group independent queries of the same type
- Execute groups in parallel where possible
- Collect results as they complete

### 3. Aggregate Results

Combine all query results into unified structure:

```toml
[[results]]
query_index = 0
type = "file"
path = "docs/requirements.md"
success = true
content = "... file contents ..."

[[results]]
query_index = 1
type = "memory"
query = "caching patterns"
success = true
matches = [
    { file = "internal/cache/lru.go", relevance = 0.89, snippet = "..." },
    { file = "internal/cache/ttl.go", relevance = 0.76, snippet = "..." }
]

[[results]]
query_index = 2
type = "semantic"
question = "How does authentication work?"
success = true
answer = "Authentication uses JWT tokens stored in..."
files_referenced = ["internal/auth/jwt.go", "internal/auth/middleware.go"]
```

### 4. Yield Complete

Return aggregated results to orchestrator:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:45:00Z

[payload]
query_count = 5
success_count = 5
failure_count = 0

[[payload.results]]
query_index = 0
type = "file"
path = "docs/requirements.md"
success = true
content = "..."

[[payload.results]]
query_index = 1
type = "memory"
query = "caching patterns"
success = true
matches = [...]

# ... additional results

[context]
role = "context-explorer"
```

---

## Error Handling

Individual query failures do not block other queries:

```toml
[[payload.results]]
query_index = 3
type = "web"
url = "https://example.com/broken"
success = false
error = "HTTP 404: Not Found"
```

Only yield `error` if all queries fail or critical infrastructure is unavailable:

```toml
[yield]
type = "error"
timestamp = 2026-02-02T10:50:00Z

[payload]
error = "All queries failed"
details = "Network unavailable"
recoverable = true
retry_count = 1

[context]
role = "context-explorer"
```

---

## Query Implementation Details

### file

```markdown
1. Use Read tool with absolute path
2. If file not found, return success=false with error
3. Truncate very large files (>50KB) with note
```

### memory

```markdown
1. Execute: projctl memory query "<query>"
2. Parse JSON output for matches
3. Include file paths, relevance scores, snippets
```

### territory

```markdown
1. Execute: projctl territory map --scope <scope>
2. Parse structure output
3. Include directory tree, key files
```

### web

```markdown
1. Use WebFetch tool with url and prompt
2. Return interpreted content per prompt
3. Handle redirects, timeouts gracefully
```

### semantic

```markdown
1. Use Task tool to spawn exploration agent
2. Agent reads code, answers question
3. Return answer with files referenced
```

---

## Boundaries

| In Scope | Out of Scope |
|----------|--------------|
| Executing query types | Deciding what to query |
| Aggregating results | Interpreting results for domain |
| Parallel execution | Caching results |
| Error reporting per query | Retry logic (orchestrator handles) |

---

## Traces

**Traces to:** ARCH-7, REQ-10
