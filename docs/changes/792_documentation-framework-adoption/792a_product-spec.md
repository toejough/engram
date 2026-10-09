# Documentation Framework Adoption - Product Spec

## 792a1 Problem Statement

Joe and the LLM agents that work on engram can't find engram's current behavior in one
authoritative, traceable place. That behavior is described across legacy documents that predate the
`docs/` framework: `docs/deprecated/` (ADRs, C4 views, invariants, design and research notes),
`openspec/specs/`, and `openspec/changes/archive/`. These documents use incompatible formats and
ID schemes. They mix removed features with shipped ones, and live files still link to paths that no
longer exist. No test is tied to a specification requirement, so nobody can check which behavior is
verified. Each capability and component needs one specification in the framework's format, with
stable issue-rooted IDs. The tests need to be traceable to the scenarios they verify, and no legacy
documentation can be left behind.

Capability sections `792a2` through `792a23` are added by the nodes of
[`792c_implementation-plan.md`](./792c_implementation-plan.md), using the template in its section
`792c3`.
