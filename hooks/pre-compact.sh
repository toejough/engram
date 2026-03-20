#!/usr/bin/env bash
# PreCompact hook — no-op (#350).
# Flush runs at end-of-turn via Stop hook. Running it again at
# PreCompact was redundant and wasted API tokens.
exit 0
