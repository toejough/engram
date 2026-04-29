# Two-Tier Extraction Discipline

The c4 skill's Rule 7. Every "scan source for findings" task at every level MUST split
across two model tiers. Single-pass authoring is forbidden because the same agent's bias
for plausible-looking output becomes the final output — it over-decomposes to appear
thorough, marks tested things UNTESTED because it didn't trace the test, and lets
conflated claims stand because no separate verifier ran.

## The two tiers

| Tier | Model class | Job | Owns |
|---|---|---|---|
| 1 | Haiku-class sub-agent | Wide scan: enumerate every plausible candidate from source | Recall (bias toward too many) |
| 2 | Sonnet-class+ orchestrator | Verify each candidate against source, prune false positives, merge near-duplicates, split conflated rows, locate exact file:line, fill gaps Tier 1 missed | Precision and correctness of the final artifact |

## Per-level enumeration

What Tier 1 enumerates depends on the level. Tier 2's job is always: read source, verify,
prune, locate, fill gaps.

### L1 — external-system identification

**Tier 1 enumerates:** every plausible external boundary the system crosses — HTTP/API
calls, filesystem reads/writes, subprocess invocations, environment-variable reads, OS
signal handling, IPC. Use `targ c4-l1-externals` JSON output as the seed; instruct Tier 1
to expand beyond it (the target is conservative).

**Tier 2 verifies:** which crossings rise to "external system" in the diagram (vs.
implementation detail), names + responsibilities, system-of-record column.

### L2 — container identification

**Tier 1 enumerates:** every distinct deployable/loadable artifact within the in-scope L1
element — binaries, libraries loaded by other systems, hooks, skills, on-disk stores,
config files that act as runtime contracts.

**Tier 2 verifies:** which are first-class containers vs. internal sub-modules of one
container; in-scope flag (exactly one `in_scope: true` element); `from_parent` carry-overs
match the L1 parent.

### L3 — component identification + code_pointer location

**Tier 1 enumerates:** every Go package, every notable file, every cohesive cluster of
functions inside the focus container's source surface. Bias toward many.

**Tier 2 verifies:** which clusters merit a separate component vs. collapse together;
exact `code_pointer` paths; `from_parent` neighbors; `kind: "component"` requires a
`code_pointer` that resolves on disk (the audit catches dead paths via
`code_pointer_unresolved`).

### L4 — properties, manifest, R-edge tags

Tier 1 enumerates **four candidate types**:

1. **Property candidates** — testable behaviors, architectural invariants likely UNTESTED
   (DI seams, no-direct-I/O, error-wrapping discipline, format contracts), shape/parse/lifecycle
   invariants.
2. **Call-diagram nodes** — sibling components the focus calls and every external system
   the focus crosses to via DI (filesystem, network, OS, Anthropic API, Claude Code, etc.).
   Each external must end up as a node with at least one R-edge from the focus — the L4
   builder enforces this via strict alignment.
3. **Manifest rows** — one per DI seam: `field`, `type`, wirer (`wired_by_id`/`name`/`l3`),
   and `wrapped_entity_id` (which call-diagram node the seam ultimately drives behavior
   against).
4. **R-edge property tags** — for each R-edge, the property IDs the call realizes (these
   become the `properties` field on the edge).

**Tier 2 verifies each:**

- **Properties** — read source to verify the claim, prune false positives, merge near-
  duplicates, split rows that smuggle two guarantees, locate exact `enforced_at` and
  `tested_at` file:line, mark genuinely untested as `tested_at: []`.
- **Call-diagram nodes** — confirm each external the focus crosses to appears with at
  least one R-edge from the focus; confirm siblings appear in the L3 parent's catalog
  (no fabricated nodes).
- **Manifest rows** — confirm `wrapped_entity_id` matches a node on the call diagram
  (strict alignment); confirm each row's wirer is correct by reading the wirer's
  construction code.
- **R-edge property tags** — confirm each P-ID in an R-edge's `properties` list
  corresponds to a property the called code path actually realizes.

## Iron rules

- **Tier 1 output is signal, never the final artifact.** Never write `c<level>-<name>.json`
  directly from Tier 1.
- **Tier 2 must read source to verify, not just trust Tier 1's claims.** When Tier 1 says
  `tested_at: <file>:<line>`, Tier 2 opens that file and confirms the line actually
  asserts the claim.
- **Tier 2 may add candidates Tier 1 missed.** Tier 1's enumeration is a floor, not a
  ceiling.
- **Tier 2 never invents pointers** (Rule 2 of the skill). When Tier 1 proposed a
  `tested_at` or `code_pointer` that doesn't survive verification, Tier 2 marks the row
  UNTESTED / removes the bad pointer — does not delete the row, does not fabricate a
  different pointer.
- **Tier 1 sub-agent dispatch must specify the model.** Use the Agent tool with the
  Haiku-class model. The orchestrator (you) is the Sonnet-class+ verifier.

## Empirical baseline

The 19-component L4 regen run showed that single-tier Sonnet sub-agents over-decomposed
small components (`main`: 5 properties where 3 sufficed) and missed individual test
pointers the original human author had located (`memory` round-trip, `cli` type-routing,
`recall` FormatResult, `main` signal_test seam). Two-tier dispatch is designed to catch
both classes of failure. The same failure mode applies symmetrically to L3 component
mining and L1/L2 candidate identification — Sonnet single-pass over-decomposes and
fabricates plausible-but-absent items at every level.

## Tier 1 dispatch template

```
You are Tier 1 of the C4 Two-Tier Extraction Discipline. Your job is RECALL, not precision.
Bias toward enumerating too many candidates — Tier 2 will prune.

Target: <component / container / system surface>
Source: <files/packages/paths>

Enumerate every plausible <property | component | container | external | manifest row | ...>.
For each candidate include: name, the source file:line you spotted it from, a one-line
description, and any guess at related fields (code_pointer, tested_at, wrapped_entity_id).

Do NOT prune. Do NOT verify pointers — speculate them with file:line and let Tier 2
verify. If you're unsure whether something belongs, INCLUDE IT and flag it as uncertain.

Report as a JSON array of candidate objects.
```
