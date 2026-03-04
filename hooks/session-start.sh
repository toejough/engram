#!/usr/bin/env bash
set -euo pipefail

cd "${CLAUDE_PLUGIN_ROOT}"
go build -o bin/engram ./cmd/engram/ 2>/dev/null || echo "[engram] Warning: build failed. Is Go installed?" >&2
