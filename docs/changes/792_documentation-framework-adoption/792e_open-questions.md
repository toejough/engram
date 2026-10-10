# Documentation Framework Adoption - Open Questions

These questions need Joe's input. Every question has a provisional default so that work not blocked
by it can continue. Choices that need only validation, not input, are in
[`792d_decision-log.md`](./792d_decision-log.md).

## 792e1 Is eval and audit tooling a product capability?

- **Context:** five OpenSpec specs describe dev tooling, not shipped CLI behavior:
  `eval-warm-config-fidelity`, `eval-trial-vault-leak-detection`,
  `eval-phase2-fixture-scoring-fidelity`, `retrieval-probe-signal-fidelity`, and `memory-loop-audit`.
  ADR-0028 D5 calls the audit "product". Vault note 293 says not to spec dev tooling as product.
- **Options:**
  1. Product capability `memory-effectiveness-evaluation`, with components `eval-trial-harness` and
     `memory-loop-auditor`.
  2. Technical components only, under a dev-tooling capability that stays out of `docs/specs/product/`.
  3. Out of scope for `docs/specs/`. The knowledge goes to `dev/eval/` docs and the LEDGER, and the
     specs are deleted.
- **Recommendation:** 1. Joe is the user of these tools, their contracts are tested, and option 3
  would lose stated requirements.
- **Blocks:** nodes 792c7v1–792c7v3, which are last in the plan. Default: option 1.

## 792e2 Which issue key roots the parallel memory agent change?

- **Context:** `docs/deprecated/design/parallel-memory-agent-requirements.md` is a draft
  requirements document for an unbuilt feature. `gh issue list` shows no matching issue, and this
  migration must not file one.
- **Options:**
  1. Joe files or names an issue; the design moves to `docs/changes/<key>_parallel-memory-agent/`.
  2. Root it under #792 for now. This conflicts with 792d14.
  3. Delete it and rely on git history.
- **Recommendation:** 1.
- **Blocks:** node 792c8b, and through it 792c8d and 792c8e. Default: none. The node waits.
- **Answer:** option 1. Joe filed #793, and the design is rooted there (792d131).

## 792e3 Should the OpenSpec tooling be removed?

- **Context:** after migration, `openspec/config.yaml`, `.claude/skills/openspec-*` and
  `.claude/commands/opsx/*` have no specs to act on. The config still describes `openspec/specs/` as
  the primary behavior record.
- **Options:**
  1. Remove all of it in node 792c8c.
  2. Keep it for other uses.
- **Recommendation:** 1. Git is the fallback, and keeping deprecated machinery contradicts Joe's
  standing rule.
- **Blocks:** the tooling item of node 792c8c. Default: option 1.
- **Applied default:** option 1 was applied in node 792c8c (792d129).

## 792e4 Do rationale paragraphs survive promotion?

- **Context:** 792d7 puts a `Rationale:` paragraph under each requirement. The lifecycle guide allows
  only "the concise context needed to understand that behavior" in canonical specs, and keeps
  implementation decisions out of them.
- **Options:**
  1. Promote rationale as-is, treating it as concise context.
  2. Trim it to one sentence at promotion.
  3. Drop it at promotion and rely on git and the issue.
- **Recommendation:** 2.
- **Blocks:** nothing until promotion. Default: write the paragraphs now.

## 792e5 Where does `vault-graph.png` live?

- **Context:** the root `README.md` embeds `docs/images/vault-graph.png`, which now lives under
  `docs/deprecated/images/`. `docs/AGENTS.md` defines no images directory.
- **Options:**
  1. `docs/images/vault-graph.png`, which restores the original link.
  2. A repository-root `assets/` directory.
  3. Drop the image from the README.
- **Recommendation:** 1.
- **Blocks:** one item in node 792c8d. Default: option 1.

## 792e6 When are the specs promoted?

- **Context:** the plan excludes promotion (792d44). Until promotion, references point at the change
  files (792d34).
- **Options:**
  1. Promote once Joe has validated 792d and answered 792e.
  2. Promote capability by capability during later work.
- **Recommendation:** 1, as a single integration.
- **Blocks:** nothing in this plan.

## 792e7 What should `TARG_BASELINE_PATTERN` become?

- **Context:** `dev/targs.go` defaults the pattern to `TestT[0-9]+_`. Renames remove that prefix from
  the eight tests that match it, and targ's use of the pattern is opaque to agents.
- **Options:**
  1. Match the new scenario-ID prefix.
  2. Remove the default.
  3. Joe states the pattern's purpose and its target value.
- **Recommendation:** 3, then apply it in node 792c8e.
- **Blocks:** nothing. Default: 792d38 (update the pattern at the first `TestT` rename).

## 792e8 Should the Go eval harness behind `targ eval` be specified?

- **Context:** `dev/eval/*.go` (arms `nothing`, `skills-only` and `current-state`, three scenarios, behavior checks, aggregation and
  a JSONL results writer) is the Milestone 1 harness that `targ eval` runs. No legacy source states its behavior, and
  ADR-0028 D7 retired the build-value direction that the cumulative harness served. 792a23 and 792b38 specify the Python
  tooling that the four eval specs cover and leave this harness out.
- **Options:**
  1. Leave it unspecified and keep its tests as no-scenario tests.
  2. Specify it as another requirement set of eval-trial-harness, with scenarios derived from its tests.
  3. Retire it.
- **Recommendation:** 1, until Joe says whether `targ eval` is still used.
- **Blocks:** nothing. Default: option 1.
