# Design: Issue Traceability

## Overview

CLI interaction design for ISSUE prefix support in projctl trace commands.

## User Interactions

### DES-001: Trace Add Error Messages
**Links:** REQ-001, REQ-004

When user attempts invalid ISSUE link:

```bash
$ projctl trace add --from ISSUE-1 --to DES-001
error: ISSUE can only link to REQ (got DES-001)
```

When user attempts valid ISSUE link:

```bash
$ projctl trace add --from ISSUE-1 --to REQ-001
Added link: ISSUE-1 → REQ-001
```

### DES-002: Trace Validate Output
**Links:** REQ-002, REQ-003

When ISSUE exists in chain, show full coverage:

```bash
$ projctl trace validate --dir .
{
  "pass": true,
  "coverage": {
    "ISSUE-1": ["REQ-001", "REQ-002"],
    "REQ-001": ["ARCH-001", "ARCH-002"],
    ...
  }
}
```

When no ISSUE exists (REQ is head), output unchanged:

```bash
$ projctl trace validate --dir .
{
  "pass": true,
  "coverage": {
    "REQ-001": ["ARCH-001", "ARCH-002"],
    ...
  }
}
```

### DES-003: Trace Impact Output
**Links:** REQ-005

Forward impact from ISSUE shows downstream chain:

```bash
$ projctl trace impact --id ISSUE-1
Impact analysis for ISSUE-1:
  → REQ-001
    → ARCH-001
      → TASK-001
    → ARCH-002
  → REQ-002
    → ARCH-003
```

Reverse impact from REQ shows upstream ISSUE:

```bash
$ projctl trace impact --id REQ-001 --reverse
Reverse impact for REQ-001:
  ← ISSUE-1
```

### DES-004: Orphan Detection
**Links:** REQ-002

When ISSUE in traceability.toml doesn't exist in issues.md:

```bash
$ projctl trace validate --dir .
{
  "pass": false,
  "orphan_ids": ["ISSUE-3"],
  "message": "ISSUE-3 in traceability but not found in docs/issues.md"
}
```

## Traceability

| ID | Links To |
|----|----------|
| DES-001 | REQ-001, REQ-004 |
| DES-002 | REQ-002, REQ-003 |
| DES-003 | REQ-005 |
| DES-004 | REQ-002 |
