# Trace Commands

The `projctl trace` commands manage traceability validation and repair for project artifacts.

## Commands

| Command | Purpose |
|---------|---------|
| `projctl trace validate` | Validate traceability chain completeness |
| `projctl trace repair` | Auto-fix duplicates, escalate dangling references |
| `projctl trace show` | Visualize the traceability graph |
| `projctl trace promote` | Promote TASK traces to upstream IDs |

---

## trace repair

Analyzes artifact files for traceability issues and auto-fixes when possible. Issues that cannot be auto-fixed are escalated for manual resolution.

### Usage

```bash
projctl trace repair -d <project-dir> [-j]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |
| `-j, --json` | Output as JSON |

### Behavior

The repair command handles two types of issues differently:

| Issue Type | Behavior | Result |
|------------|----------|--------|
| Duplicate IDs | Auto-fixed | Renumbered to next available ID |
| Dangling references | Escalated | Requires manual resolution |

### Auto-Fix: Duplicate IDs

When the same ID is defined in multiple locations, the repair command automatically renumbers subsequent occurrences to the next available ID in that prefix's sequence.

**Example: Duplicate DES-001 across files**

Before repair:
```
docs/design.md:
  ### DES-001: Main Design
  ### DES-002: Other Design

docs/design-feature.md:
  ### DES-001: Feature Design    <-- duplicate
```

After repair:
```
docs/design.md:
  ### DES-001: Main Design       <-- unchanged (first occurrence)
  ### DES-002: Other Design

docs/design-feature.md:
  ### DES-003: Feature Design    <-- renumbered to next available
```

**Key behaviors:**
- First occurrence of a duplicate ID is preserved
- Subsequent occurrences are renumbered to the next available ID
- All references to the old ID within the renamed file are updated
- The command tracks the maximum ID number per prefix (REQ, DES, ARCH, TASK) to ensure uniqueness

**Duplicate IDs never create escalations.** They are always auto-fixed.

### Escalation: Dangling References

When an artifact references an ID in its `**Traces to:**` field that doesn't exist, the repair command creates an escalation. These cannot be auto-fixed because they require human judgment about whether to:
- Create the missing artifact
- Fix the typo in the reference
- Remove the invalid reference

**Example: Reference to non-existent REQ-999**

```markdown
### DES-001: Orphan Design

Design that references non-existent requirement.

**Traces to:** REQ-999    <-- REQ-999 not defined anywhere
```

This creates an escalation entry:
```json
{
  "id": "REQ-999",
  "reason": "dangling reference: ID referenced in Traces to: but not defined"
}
```

### Terminal Output

**When no issues are found:**
```
No traceability issues found
```

**When issues are found:**
```
Traceability Issues Found
=========================

Dangling references (2):
  - REQ-999 (referenced but not defined)
  - REQ-888 (referenced but not defined)

Duplicate IDs (1):
  - DES-001 (defined multiple times)
```

Exit code is 1 when issues are found (after auto-fixes are applied), 0 otherwise.

### JSON Output

With the `-j` flag, output is machine-readable:

```json
{
  "dangling_refs": ["REQ-999"],
  "duplicate_ids": ["DES-001"],
  "renumbered": [
    {
      "old_id": "DES-001",
      "new_id": "DES-003",
      "file": "design-feature.md"
    }
  ],
  "escalations": [
    {
      "id": "REQ-999",
      "reason": "dangling reference: ID referenced in Traces to: but not defined"
    }
  ]
}
```

### Escalation Entry Format

Each escalation contains:

| Field | Description |
|-------|-------------|
| `id` | The problematic ID |
| `reason` | Why this needs manual resolution |
| `file` | Source file where the issue was found (when available) |

### Idempotency

The repair command is idempotent: running it multiple times produces the same result.

**First run:**
- Detects duplicate DES-001
- Renumbers to DES-003
- Reports the fix

**Second run:**
- No duplicates found (already fixed)
- No dangling references (if none existed)
- Reports "No traceability issues found"

Files are only modified when actual fixes are needed. Running repair on a clean codebase is a no-op.

### Artifact Files Scanned

The repair command scans these artifact files:
- `issues.md` (or configured path)
- `requirements.md` (or configured path)
- `design.md` (or configured path)
- `architecture.md` (or configured path)
- `tasks.md` (or configured path)
- Feature-specific files matching `design-*.md`, `requirements-*.md`, `architecture-*.md`

Test files are NOT scanned by repair - TEST tracing is handled via source file comments.

### ID Definition Pattern

IDs are recognized when they appear as markdown headers:
```
### PREFIX-NNN: Title
```

Where PREFIX is one of: ISSUE, REQ, DES, ARCH, TASK, TEST

### Reference Pattern

References are recognized in `**Traces to:**` fields:
```
**Traces to:** REQ-001, DES-002
```

---

## trace validate

Validates the traceability chain is complete. Unlike repair, this is read-only.

### Usage

```bash
projctl trace validate -d <project-dir> [-p <phase>] [-j]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |
| `-p, --phase` | Workflow phase for phase-aware validation |
| `-j, --json` | Output as JSON |

### Phase-Aware Validation

Different workflow phases have different requirements:

| Phase | Allowed Unlinked |
|-------|------------------|
| `design-complete` | DES (ARCH doesn't exist yet) |
| `architect-complete` | ARCH (TASK doesn't exist yet) |
| `breakdown-complete` | TASK (TEST doesn't exist yet) |
| `task-complete` or later | None (full chain required) |

---

## trace show

Visualizes the traceability graph.

### Usage

```bash
projctl trace show -d <project-dir> [-f <format>]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |
| `-f, --format` | Output format: `ascii` (default) or `json` |

### ASCII Output

```
REQ-001
├── DES-001
│   └── ARCH-001
│       └── TASK-001
└── DES-002 [UNLINKED]
REQ-002 [ORPHAN]
```

Markers:
- `[ORPHAN]`: Referenced but not defined
- `[UNLINKED]`: Defined but nothing traces to it

---

## Reference

See [orchestration-system.md Section 13.3](../orchestration-system.md) for the complete Layer 0 foundation specification.
