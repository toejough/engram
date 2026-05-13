---
description: Refresh the engram binary and harness skills/commands
---

Run `engram update` to reinstall the binary via `go install` and copy the
current skills/commands into each detected harness's user dir.

- Add `--dry-run` to see planned changes without writing anything.
- Source is auto-detected: a local clone if `cwd` is inside one, otherwise
  the latest published module (`go install …@latest`).
