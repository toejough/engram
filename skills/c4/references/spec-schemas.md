# C4 Spec Schemas

The JSON spec authored at each level is the source of truth. Its schema is defined
canonically as a Go struct in the `dev/` package; the `c4-l*-build` target decodes the
JSON strictly (`DisallowUnknownFields`) into that struct and validates it.

## Where each level's schema lives

| Level | Source file | Top-level struct | Notes |
|---|---|---|---|
| 1 | `dev/c4.go` | `L1Spec` | Element types: `L1Element` (with `Source` for the where), `L1Relationship`, `L1CrossLinks`, `L1DriftNote`, `L1RefinedBy`. |
| 2 | `dev/c4_l2.go` | `L2Spec` | Element type: `L2Element` (adds `InScope bool` flag — exactly one element must be in-scope). |
| 3 | `dev/c4_l3.go` | `L3Spec` | Adds `Focus L3Focus` field naming the parent L2 element being refined. Element type: `L3Element` (each `kind: "component"` requires `Source`). |
| 4 | `dev/c4_l4.go` | `L4Spec` | Largest schema. Includes `Focus`, `Diagram` (call-diagram nodes + R-edges), `Properties` (each with `EnforcedAt`/`TestedAt` `[]L4CodeLink`), `DependencyManifest` (rows with `wired_by_*` + `wrapped_entity_id`), `DIWires` (provider side). |

## When to read these

- **Authoring a fresh spec.** When no working sibling exists at the same level (e.g.,
  starting a brand-new c1 or a c2 in a project with no other c2 yet). The struct
  shows every field, its JSON tag, whether it's `omitempty`, and the type.
- **Debugging a validator error.** When `c4-l*-build` rejects a spec, the error names
  the offending field. Reading the struct shows the expected type/values.
- **Adding a new field.** Schema changes start in the Go struct, then propagate to
  the build target's emit/validate functions, then to existing specs.

## When NOT to read these

- **Editing an existing spec.** Copy shape from a working sibling
  (`architecture/c4/c<level>-*.json`); let the build target validate. You don't need to
  read the struct to add or remove an element.
- **Reverse-engineering targ's behavior.** The struct is the schema; the build target's
  internals (validation order, error formatting, emit pipeline) are out of scope.

## Authoring shortcut

The fastest path to a valid spec:

1. Copy a working sibling at the same level: `cp architecture/c4/c2-engram-plugin.json
   architecture/c4/c2-<your-name>.json`.
2. Edit names, IDs, elements, relationships.
3. Run `targ c4-l<level>-build --input architecture/c4/c<level>-<your-name>.json --noconfirm`.
4. Read the validator's error message (if any), fix, re-run.

If no working sibling exists at this level (bootstrapping), open the Go struct file
listed above for the canonical field list, then author from scratch using a sibling
level's structure as a rough mental model.

## Version field

Every spec carries `"schema_version": "1"` at the top. The build target checks this and
rejects unknown values. If the schema changes incompatibly, this version bumps.

## Idempotent rebuilds

The build target reads the JSON, emits the rendered `.md` and `.mmd`, and (in `--check`
mode) compares against on-disk output. Re-running the build with no JSON change produces
no diff. The audit (`targ c4-audit`) uses byte-compare against a fresh emit to detect
staleness.
