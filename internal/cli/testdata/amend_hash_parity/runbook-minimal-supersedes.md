---
type: runbook
situation: ctx
done_when: old
red_flags:
    - a flag
luhmann: "1aa"
created: "2026-01-01"
source: test
repo: github.com/acme/widgets
user: bob@example.com
vault: personal
supersedes:
    - note: 5.2026-01-01.older
      type: narrows
      claim: an older claim
---

1. old step

Supersedes: [[5.2026-01-01.older]] — narrows: an older claim
