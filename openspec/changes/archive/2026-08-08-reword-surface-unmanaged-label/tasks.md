## 1. Red: update test expectations to the new wording

- [x] 1.1 In `internal/cli/update_test.go`, change the expected substring at ~line 641 and ~line 779 from `"stale artifact (not deleted): ..."` to `"unmanaged (left alone): ..."`; run `targ test` and confirm these tests now fail against current code (RED).
- [x] 1.2 Grep `internal/update/update_test.go` for any assertions referencing `SurfaceUnattributable` or "stray"/"stale" wording and update them to the new field name `SurfaceUnmanaged`, confirming they also fail (RED).

## 2. Green: rename field and reword rendered output

- [x] 2.1 In `internal/update/update.go`, rename `HarnessReport.SurfaceUnattributable` to `SurfaceUnmanaged` and rewrite its doc comment to describe it as an unmanaged surface entry (no ownership signal either way), dropping "stray"/"cannot prove ownership" framing.
- [x] 2.2 Rename the local `strays` variable near the field's population site (~line 1234) to `unmanaged` and update the assignment (`rep.SurfaceUnmanaged = append(rep.SurfaceUnmanaged, unmanaged...)`).
- [x] 2.3 In `internal/cli/update.go` (~line 568-570), change the rendered line from `"    stale artifact (not deleted): %s\n"` to `"    unmanaged (left alone): %s\n"`, and update the loop variable's source to `harness.SurfaceUnmanaged`.
- [x] 2.4 Run `targ test` and confirm all tests from step 1 now pass (GREEN).

## 3. Spec sync and full verification

- [x] 3.1 Apply the `update-deploy-sync` delta spec (rename "Unattributable stray is reported, not deleted" scenario to "Unmanaged surface entry is reported, not deleted" with reworded THEN clause) into `openspec/specs/update-deploy-sync/spec.md` at archive time.
- [x] 3.2 Run `targ check-full` and confirm no other references to `SurfaceUnattributable` or the old rendered string remain (`grep -rn "SurfaceUnattributable\|stale artifact" .`).
