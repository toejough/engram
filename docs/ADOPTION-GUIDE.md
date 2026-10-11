# Documentation Adoption Guide

This governance guide supplements [`docs/AGENTS.md`](./AGENTS.md). Follow it when an existing
repository adopts this documentation framework: its legacy documentation and specifications are
migrated into the framework, one capability and component at a time, and then checked against the
code.

## Purpose

An adoption MUST end with:

- every piece of legacy documentation either ported into the framework's format or deliberately
  deleted, and no legacy directory left behind;
- product specifications per capability and technical specifications per component, held in one
  adoption change until they are promoted;
- every existing test that verifies a migrated scenario renamed with that scenario's ID;
- every migrated requirement traced to the code, with each mismatch resolved as a documentation fix
  or a filed issue.

Adoption describes the system as it is. It MUST NOT add or change behavior. Behavior changes found
along the way become their own issues and changes.

## Roles

- **Owner:** the person adopting the framework. The owner answers open questions and rules on
  mismatches whose cause cannot be determined.
- **Author:** whoever carries out the steps. The author makes and records the routine choices, and
  stops only for questions that block all remaining work.

## Step 1: Quarantine the Legacy Documentation

1. Open an issue for the adoption. Its key roots the adoption change.
2. Move every existing documentation file into `docs/deprecated/`, keeping its relative layout. Use
   the version-control move command so history follows the files.
3. Move back to the top of `docs/` only the files the owner chooses to keep as central documents,
   such as a glossary or a roadmap. List each one among the unnumbered files in
   [`docs/AGENTS.md`](./AGENTS.md).
4. Treat any earlier specification system (for example, a directory of per-feature specs and their
   archived changes) as legacy documentation too, whether or not it moves.
5. Expect references into the moved files to break. They are repaired as each file is ported.

## Step 2: Open the Adoption Change

Create `docs/changes/<issue-key>_documentation-framework-adoption/` with:

- `<key>a_product-spec.md`: the problem statement; capability sections are added by later steps;
- `<key>b_technical-spec.md`: an orientation that says once why option analysis, impact and
  alternative sections are omitted (the change documents an existing system);
- `<key>c_implementation-plan.md`: the plan that tracks the migration (Step 4);
- `<key>d_decision-log.md`: every decision made, each with its rationale, the alternatives
  considered, and who decided it (see [Decisions and Open Questions](./AGENTS.md#decisions-and-open-questions));
- `<key>e_open-questions.md`: only the questions that need the owner, each with options, a
  recommendation, a provisional default, and what it blocks; answers are recorded in place.

Working inventories MAY live in a further artifact directory inside the change. They are deleted
with the change.

## Step 3: Inventory Capabilities and Components

Build the inventory from four independent sources and reconcile them:

1. **Legacy documentation:** what each file, section, decision record and legacy spec describes.
2. **Code and tests:** the user-facing surface (commands, flags, shipped instructions), the
   candidate components, and which test files exercise each one.
3. **Version-control history and issues:** what was added, renamed, removed or parked. Removed and
   parked features MUST NOT be documented as current.
4. **Project memory or notes,** where the repository keeps them: standing decisions, the owner's own
   vocabulary, and known stale claims.

From these, name:

- **Capabilities:** things a user of the system does or gets, in the owner's vocabulary.
- **Components:** units that pass every qualification criterion in
  [`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md). Record each candidate that fails,
  with the reason.

Then write a **source map**: every legacy file, section, decision record and legacy spec maps to
exactly one destination. A destination is a capability, a component, a central document, the
repository's agent or contributor guidance, a separate in-flight change, or deletion with a reason.

## Step 4: Plan the Migration

Write the implementation plan with:

- one node per capability (its product-spec port), followed by one node per component that the
  capability is first to need (its technical-spec port and test renames);
- capabilities ordered so foundational ones come first; a component shared by several capabilities
  is ported once and later capabilities only add mappings to it;
- the exact section template every node uses, and the ID rules;
- for each node, the legacy sources it ports and deletes, the shipped artifacts it describes, and
  any caution recorded during the inventory;
- closing nodes, in this order: a residual legacy sweep; relocation of in-flight designs; removal of
  the earlier specification system and its tooling; removal of `docs/deprecated/` with a final
  stale-reference sweep; and, where tests must be reclassified (for example by build tag), a
  deferred classification node.

Nodes that edit the same files run one after another. Each node's statuses change in the commit
that completes it.

## Step 5: Port Each Capability

For each capability node:

1. Read its legacy sources and the shipped artifacts they describe.
2. Add the capability section to the product spec as an **Added** operation. State current behavior
   as the shipped system shows it. Do not judge correctness yet; record any divergence you notice in
   the decision log.
3. Place each piece of legacy rationale in a `Rationale:` paragraph under the requirement it
   justifies. Full history may stay here; it is trimmed at promotion, as
   [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) "Rationale at
   Promotion" requires.
4. Remove from the legacy sources only the content you ported. Content listed for a later node stays
   until that node. Delete a legacy file once nothing in it remains unported.
5. Repoint every live reference to removed content at the new section.
6. Confirm the capability's component list. Add or remove plan nodes and record the change if the
   inventory was wrong.
7. Add mapping rows and test renames to components already ported, where this capability's
   scenarios are verified by their tests.

A legacy item that is both product behavior and a component contract stays in its legacy file until
the component node that reads it, which then deletes it.

## Step 6: Port Each Component

For each component node:

1. Read the node's legacy sources, the current code, and its tests.
2. Add the component section to the technical spec: orientation, behavior requirements with scenarios,
   and the product scenarios it supports. Port sequence diagrams from any architecture views into
   flow specs, one per flow, with the flow's interaction requirements.
3. Remove ported legacy content and repoint references, as in Step 5.
4. Rename each existing test that verifies a scenario so its name begins with the scenario ID, as
   [`docs/IMPLEMENTATION-PLAN-GUIDE.md`](./IMPLEMENTATION-PLAN-GUIDE.md) requires. Update every live
   reference to the old name.
5. Under the node's catalog section, record a traceability table (scenario, old and new test name,
   classification), the scenarios with no test (coverage gaps), and the tests that verify no
   scenario. Do not write new tests during adoption.
6. Run the full quality checks before committing the node.

Rename and reference rewrites MUST NOT change dated historical records: archived evaluation output,
logs and transcripts, completed-change archives, the plan's own lists of old test names, the
decision log, and working inventories. Match exemptions by substring, not path prefix, and dry-run
every bulk rewrite.

## Step 7: Close Out the Migration

1. **Residual sweep:** port what is left (central-document content, contributor guidance,
   still-useful research conclusions) and delete the rest. Version control is the fallback; keep no
   restore copies.
2. **In-flight designs:** each draft for unbuilt work becomes its own change, rooted at its own
   issue. File the issue if none exists.
3. **Earlier specification system:** mine its archive for rationale the capability nodes did not
   port, then delete it, and remove its tooling if the owner agrees.
4. **Deprecated directory:** remove `docs/deprecated/`, then search the whole repository for every
   legacy path and identifier and repoint each live hit.
5. **References:** every reference to addressable content follows
   [`docs/AGENTS.md`](./AGENTS.md) "References to Addressable Content". A scripted pass can link bare
   IDs. A reviewed pass then finds content referred to only by name.

## Step 8: Audit Each Capability Against the Code

Run one audit per capability, recording each in a file inside the adoption change. Cover every
requirement and scenario in the capability's section, plus every technical requirement that
supports them.

1. Find the implementing code and tests. Check behavior against a built binary or a real run where
   that is cheap, using temporary state only.
2. For each mismatch, compare when and why the ported text was written with when and why the code
   last changed. Recover deleted legacy sources from version-control history.
   - **Documentation problem:** the legacy text was stale and the code reflects a later, deliberate
     decision. Fix the spec text, keeping IDs stable. Also fix the same stale claim in the README,
     glossary and code comments.
   - **Code problem:** the text states the intended behavior and the code was never updated,
     regressed, or has a defect. Keep the spec stating the intent and file an issue with the
     observation, a reproduction, the expected behavior, and the commits compared. Reference an
     existing issue instead of filing a duplicate.
   - **Unclear:** record the evidence and a best judgment. The owner rules on each one. A ruling
     that the behavior is wrong becomes an issue; a ruling that the requirement is wrong becomes a
     spec edit or a removal.
3. Leave matching requirements unedited.
4. Give each row a verdict (`match`, `doc fixed`, `code bug #<n>`, `unclear`, or covered by a named
   pending change) and its evidence.

Shipped instruction text (for example, agent skills) counts as code here. A defect in it is filed as
an issue; it is not edited during the audit.

## Behavior Changes Raised During Adoption

When the owner wants behavior to change, open one issue and one change per independently shippable
change, even when several were decided in one discussion. The new change carries **Modified** or
**Removed** operations against the adoption change's IDs. In the adoption spec, replace any review
note with a link to the change that alters that requirement.

## Completion Checklist

Verify:

- `docs/deprecated/` and the earlier specification system are gone.
- Every legacy source in the source map was ported or deleted for a recorded reason.
- Every capability and component section follows the plan's template.
- Every live reference resolves, and none points at a moved or deleted path.
- Every renamed test begins with the scenario ID it verifies, and every component node lists its
  coverage gaps and no-scenario tests.
- Every capability has an audit, every mismatch is resolved as a documentation fix or an issue, and
  every unclear item has the owner's ruling.
- Every in-flight design and every owner-requested behavior change has its own issue and change.
- The decision log and open questions are ready for the owner.
- The full quality checks pass.
- Promotion to `docs/specs/` follows the
  [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) once the owner has
  validated the decisions.
- The shared checklist in [`docs/AGENTS.md`](./AGENTS.md) passes.
