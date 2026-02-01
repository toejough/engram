# Model Routing

This document explains the model routing feature and its limitations.

## Overview

projctl supports model routing to optimize cost by suggesting appropriate models (haiku, sonnet, opus) for different skill complexities.

## How It Works

1. Skills are classified by complexity: simple, medium, or complex
2. Configuration maps complexity levels to models
3. `projctl context write` injects a `[routing]` section with `suggested_model`

## Limitations

### Inline Work Uses Session Model

When Claude performs work inline (in the main conversation), it uses whatever model was selected for the session. The routing suggestion is **advisory only** - the orchestrator cannot force a model change for inline work.

### Task Tool Subagents Can Use Specified Model

When dispatching work via the Task tool, subagents can be launched with a specific model. This is the recommended approach for cost-critical skills.

Example of Task tool with model hint:

```json
{
  "subagent_type": "tdd-red",
  "model": "haiku",
  "prompt": "Write failing tests for the validation function",
  "description": "Write failing tests"
}
```

## Recommendations

1. **Use subagent dispatch for cost-critical skills** - This allows explicit model selection
2. **Configure skill complexity mappings** - Set appropriate complexity levels in `project-config.toml`
3. **Monitor token usage** - Use `projctl log read --model` to analyze cost by model

## Configuration

In `project-config.toml`:

```toml
[routing]
simple = "haiku"
medium = "sonnet"
complex = "opus"

[routing.skill_complexity]
alignment-check = "simple"
tdd-red = "medium"
tdd-green = "medium"
meta-audit = "complex"
```

## Default Skill Mappings

| Skill | Complexity | Rationale |
|-------|------------|-----------|
| alignment-check | simple | Lightweight validation |
| tdd-red | medium | Standard development |
| tdd-green | medium | Standard development |
| tdd-refactor | medium | Standard development |
| commit | medium | Standard development |
| meta-audit | complex | Deep analysis |
| architect-interview | complex | Strategic decisions |
| pm-interview | complex | Requirements discovery |
