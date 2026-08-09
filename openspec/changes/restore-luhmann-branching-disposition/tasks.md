## 1. Baseline (RED)

- [x] 1.1 Using `superpowers:writing-skills`, capture a baseline scenario (real or scripted
      capture sequence) showing that today's `learn` skill always writes `--position top`, even
      when a note clearly develops a sub-point of an earlier in-session note — confirm this is
      the failure the change fixes.

## 2. write-memory handoff contract

- [x] 2.1 Update `agent-instructions/skills/write-memory/SKILL.md` to accept `position`
      (`top`/`continuation`/`sibling`) and `target` fields in its handoff, and compose
      `--position <position> [--target <target>]` into the `engram learn <kind>` command
      accordingly (target omitted only when position is `top`).
- [x] 2.2 Verify existing write-memory examples/tests (if any) still pass with `position=top` as
      the default when the calling skill omits the field, preserving current behavior for callers
      that don't yet send disposition. (No automated tests exist for skill prose; verified by
      inspection — the compose section states `--position` defaults to `top` and `--target` is
      omitted whenever the handoff doesn't include one, matching pre-change behavior.)

## 3. Learn skill disposition step (GREEN)

- [x] 3.1 Add the disposition step to `agent-instructions/skills/learn/SKILL.md` Step 2: before
      each write-memory handoff, apply the ordered test from design.md Decision 2 (continuation →
      sibling → top) against notes written or recalled earlier in the current session, and
      include the resulting `position`/`target` in the write-memory handoff for kinds 1-4.
- [x] 3.2 Cite the grounding (Luhmann's Zettelkasten system / Ahrens, *How to Take Smart Notes*)
      inline in the skill so the rule's provenance is traceable, per the issue's requirement that
      this not be reinvented ad hoc.
- [x] 3.3 Re-run the Step 1.1 baseline scenario and confirm continuation/sibling positions are now
      selected and passed through to `engram learn`, producing branched IDs instead of flat
      top-level ones. (GREEN subagent run: composed `--position continuation --target 42` for a
      note that develops one specific sub-point of note 42.)

## 4. Verification

- [x] 4.1 Run the `superpowers:writing-skills` pressure tests against the updated `learn` and
      `write-memory` skills. (This is a technique skill, not a discipline-enforcing one under
      adversarial pressure — tested per "Technique Skills" guidance with a RED baseline scenario
      and a matching GREEN application scenario; both ran via fresh-context subagents, see 3.3.)
- [x] 4.2 Run `targ check-full` to confirm no Go-side regressions (none expected — no `internal/`
      changes in this proposal).
- [x] 4.3 Confirm ADR-0025 (tag retrieval) and ADR-0012 (supersession) behavior is unchanged by
      spot-checking that `--supersedes` and tag flags on `engram learn` are unaffected by the new
      disposition fields. (Inspected write-memory/SKILL.md: `--supersedes` and `--tag` composition
      is untouched by this change; the new position/target fields are additive.)

## 5. Close-out

- [x] 5.1 Update `openspec/specs/` (sync) to add the new `learn-branching-disposition` capability
      once implementation lands, per `openspec-sync-specs`.
- [x] 5.2 Close GitHub issue #701, referencing this change.
