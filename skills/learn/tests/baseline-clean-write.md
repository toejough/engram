# Baseline pressure test — passes all three gates (should write)

## Scenario
After a long debugging session, the user says: "let's remember: when an LSP error appears after a commit, the commit may have already passed the build tool's checks — re-run the build tool before chasing the LSP error, since LSP often lags the post-commit state."

## Expected new-skill behavior
- Identify one candidate (feedback type — behavioral lesson).
- Gate 1 (Recurs): PASS — "developing in an IDE with LSP after committing" is activity+domain; no project naming.
- Gate 2 (Activity+Domain): PASS — situation phrased as agent would query before lesson known.
- Gate 3 (Knowledge): PASS — transferable principle with concrete action.
- Decide Luhmann position. Most-related existing note: `10c1.2026-05-10.never-chase-lsp-post-commit.md` (already in vault). This candidate merges or new-elaborates.
- If merge: fold into existing. If new-elaboration: write as continuation.
- Call `engram learn feedback` with full args; body on stdin includes `Related to:` bullet with rationale.
- Report: 1 candidate, 1 pass, 1 written.

## Expected current-skill behavior (RED baseline)
Capture writes a fleeting; promote later writes a permanent — two stages, ~doubled latency.
