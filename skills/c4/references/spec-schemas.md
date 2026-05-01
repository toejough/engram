# C4 Spec Schemas

The JSON spec authored at each level is the source of truth. Its schema is defined
canonically as a Go struct in the `dev/` package; the `c4-l*-build` target decodes the
JSON strictly (`DisallowUnknownFields`) into that struct and validates it.

**Always author against this canonical schema.** Do not copy from an existing
`architecture/c4/c<level>-*.json` as a template — that propagates whatever quirks,
stale fields, or instance-specific values are present in the sibling, and any drift
between siblings becomes drift in the new spec.

## Where each level's schema lives

| Level | Source file | Top-level struct | Notes |
|---|---|---|---|
| 1 | `dev/c4.go` | `L1Spec` | Element types: `L1Element` (with `Source` for the where), `L1Relationship`, `L1CrossLinks`, `L1DriftNote`, `L1RefinedBy`. |
| 2 | `dev/c4_l2.go` | `L2Spec` | Element type: `L2Element` (adds `InScope bool` — exactly one element must be in-scope). |
| 3 | `dev/c4_l3.go` | `L3Spec` | Adds `Focus L3Focus` naming the parent L2 element being refined. Element type: `L3Element` (each `kind: "component"` requires `Source`). |
| 4 | `dev/c4_l4.go` | `L4Spec` | Largest schema. Includes `Focus`, `Diagram` (call-diagram nodes + R-edges), `Properties` (each with `EnforcedAt`/`TestedAt` `[]L4CodeLink`), `DependencyManifest` (rows with `wired_by_*` + `wrapped_entity_id`), `DIWires` (provider side). |

## When to read these

- **Authoring any spec** — fresh or otherwise. The struct shows every field, its JSON
  tag, whether it's `omitempty`, and the expected type. Author by enumerating the
  required fields from the struct.
- **Debugging a validator error.** `c4-l*-build` rejects malformed input with a
  message naming the offending field. The struct disambiguates the expected type/values.
- **Adding a new field.** Schema changes start in the Go struct, propagate to the
  build target's emit/validate functions, then to existing specs.

## When NOT to read these

- **Reverse-engineering the build target's behavior.** The struct is the schema; the
  build target's internals (validation order, error formatting, emit pipeline) are out
  of scope. Treat the build target as a black box that reads JSON in, writes
  `.md`/`.mmd` out, and validates strictly.

## Authoring procedure

1. Open the level's Go struct (table above).
2. Enumerate required fields. For each, note the JSON tag, type, and any constraints
   (enum values, regex, embedded struct shape).
3. Author the JSON top-down, field by field, against that enumeration. Don't speculate
   field names from another spec; speculate field names from the struct definition.
4. Run `targ c4-l<level>-build --input <spec>.json --noconfirm`.
5. Read the validator's error message (if any), fix, re-run.

## Version field

Every spec carries `"schema_version": "1"` at the top. The build target checks this and
rejects unknown values. If the schema changes incompatibly, this version bumps.

## Idempotent rebuilds

The build target reads the JSON, emits the rendered `.md` and `.mmd`, and (in `--check`
mode) compares against on-disk output. Re-running the build with no JSON change produces
no diff. The audit (`targ c4-audit`) uses byte-compare against a fresh emit to detect
staleness.
