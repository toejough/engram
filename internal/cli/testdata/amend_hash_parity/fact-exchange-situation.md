---
type: fact
tier: L2
situation: a new situation
subject: the widget
predicate: uses
object: a gear
luhmann: "1aa"
created: "2026-01-01"
source: test
repo: github.com/acme/widgets
user: bob@example.com
vault: personal
pending: true
tags:
    - vocab/old-term
supersedes:
    - note: 9.2026-01-01.deleted-def.md
      type: narrows
      claim: old claim
xid: 7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e
parent:
    vault: 9a1e2b3c4d5e6f708192a3b4c5d6e7f8
    links:
        - note: 1100.2026-09-27.x
          via: offered
          hash: xh1:abababababababababababababababababababababababababababababababab
        - note: 812.2026-08-01.y
          via: covered
          hash: xh1:cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd
    author:
        repo: github.com/acme/widgets
        user: alice@example.com
        vault: team
aliases:
    - 1101.2026-09-27.z
    - 12.2026-06-01.old-name
offer:
    origin: 0123456789abcdef0123456789abcdef:fedcba9876543210fedcba9876543210
    key: xh1:efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef
    for: 1100.2026-09-27.x
    path:
        - 0123456789abcdef0123456789abcdef
        - 9a1e2b3c4d5e6f708192a3b4c5d6e7f8
---

Information learned: when in a new situation, the widget uses a gear.

Supersedes: [[9.2026-01-01.deleted-def]] — narrows: old claim
