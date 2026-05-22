# Plan — Remove all MEMORY.md references

## Why

`MEMORY.md` is an aspirational artifact: bootstrap creates an empty stub, no code
reads or writes it, and SKILL.md prose calls it the vault's "index — names notes"
when nothing maintains it. The diagram authored in the previous turn carried the
same overstatement (R3 claimed engram reads & writes `MEMORY.md`). Remove every
mention so docs match reality.

## Inventory (verified via grep, 19 references)

### Code
1. `internal/cli/vault_init.go:41,74` — bootstrap creates empty MEMORY.md
2. `internal/cli/vault_init_test.go:38,42` — asserts bootstrap creates it
3. `internal/cli/learn_adapters_test.go:238,250` — asserts MEMORY.md exists after bootstrap

### Skills (writing-skills TDD per CLAUDE.md)
4. `skills/learn/SKILL.md:39` — bootstrap list mentions MEMORY.md
5. `skills/recall/SKILL.md:35` — vault layout claims MEMORY.md is "index — names notes"

### User-facing docs
6. `README.md:56` — bootstrap description
7. `docs/GLOSSARY.md:77` — "MEMORY.md" glossary entry
8. `docs/architecture/c1-system-context.md` — R3 edge label and S4 catalog row (3 mentions in the file authored last turn)

### Design / research docs
9. `docs/superpowers/specs/2026-05-12-tiered-memory-research-prompt.md:112` — vault layout example
10. `docs/superpowers/specs/2026-05-14-tiered-memory-design.md` — 5 mentions of MEMORY.md as a human-readable index in the proposed L3 design

## Order of operations

1. **Housekeeping commit**: commit the pending tiered-memory design doc reorg
   from the previous turn (it blocks clean separation; user reviewed the diff
   visually but never asked for commit; lands as its own logical commit before
   MEMORY.md edits go on top).
2. **Skill edits via `superpowers:writing-skills` TDD**:
   - `learn/SKILL.md` — RED: roleplay subagent given current skill text, ask
     "what files exist in a freshly-bootstrapped engram vault?" expecting
     MEMORY.md in the answer. GREEN: same scenario with updated skill, expect no
     MEMORY.md. REFACTOR: near-miss prompt to confirm we didn't over-broaden
     (a query about *the index file* should not invent MEMORY.md).
   - `recall/SKILL.md` — same RED/GREEN/REFACTOR shape with a recall-layout
     prompt.
3. **Code edits, inline TDD**:
   - Remove `{"MEMORY.md", "# Memory Index\n"}` line from
     `internal/cli/vault_init.go`'s bootstrap files slice.
   - Update the comment on line 41 to drop the MEMORY.md mention.
   - Remove the `os.Stat(filepath.Join(vault, "MEMORY.md"))` assertion in
     `learn_adapters_test.go` (and the surrounding comment).
   - Remove the `"/v/MEMORY.md"` entry and matching expectation in
     `vault_init_test.go`.
   - Run `targ check-full`.
4. **Doc edits**:
   - `README.md` — drop MEMORY.md from bootstrap list.
   - `docs/GLOSSARY.md` — remove the MEMORY.md glossary entry entirely.
   - `docs/architecture/c1-system-context.md` — fix R3 label to "reads & writes
     notes and MOCs" and drop MEMORY.md from the S4 catalog source column.
   - `docs/superpowers/specs/2026-05-12-tiered-memory-research-prompt.md` —
     drop MEMORY.md from the vault layout sample.
   - `docs/superpowers/specs/2026-05-14-tiered-memory-design.md` — drop the L3
     root → MEMORY.md bridge mentions; root MOC stands on its own.
5. **Re-grep verification**: `grep -rn MEMORY.md` returns nothing outside the
   vault's historical notes (which are append-only — not touched).
6. **Single MEMORY.md-removal commit + push.**

## TDD specifics

- **Code removal**: the two failing assertions become the RED (since the absence
  of MEMORY.md creation will make them fail); removing both the production line
  and the assertions in one commit keeps the test suite green at every commit
  boundary.
- **Skill edits**: writing-skills with roleplay-subagent RED/GREEN/REFACTOR. Per
  the cluster, RED passes too easily means clause ambiguity — tighten if needed.
- **Diagram edit**: structural (grep verification that "MEMORY.md" no longer
  appears in the file).

## Out of scope

- Historical vault notes that mention MEMORY.md (in
  `~/.local/share/engram/vault/Permanent/` and `MOCs/`) — append-only, not
  touched.
- Migration path for users with existing MEMORY.md files — not needed; the file
  is empty and harmless to leave behind. The bootstrap simply stops creating it.

## Cleanup

Delete this file (`docs/plan-memory-md-removal.md`) in step 6.
