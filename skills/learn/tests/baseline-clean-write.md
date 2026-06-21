# Baseline pressure test — explicit save-request: sweep then write one note

## Scenario

After a long debugging session, the user says: "let's remember: when an LSP error appears after a commit, the commit may have already passed the build tool's checks — re-run the build tool before chasing the LSP error, since LSP often lags the post-commit state."

## Expected current-skill behavior (PASS)

- **Step 1 — sweep first:** `engram ingest --auto` runs before anything else. Report the one-line tally.
- **Step 2 — explicit save-request detected:** the user said "let's remember" — this is an explicit save-request. Write exactly one note:

```bash
engram learn feedback --slug lsp-lags-post-commit-rerun-build \
  --position top \
  --source "session <date>, context: debugging LSP errors after commit" \
  --situation "after a commit, LSP reports errors but the build tool passed" \
  --behavior "chasing the LSP error directly" \
  --impact "wastes time on phantom errors that the commit already fixed" \
  --action "re-run the build tool first; if it passes, the LSP error is stale lag"
```

- **One note, one write, done.** No Luhmann-continuation logic. No gate checks. No position arithmetic. `--position top` is the default for new notes.
- Report: "1 explicit save-request crystallized."

## Failure modes that must FAIL this test

- Skipping Step 1 (the sweep must come first).
- Not writing the note ("this is just a fact, not a correction" — irrelevant; user said remember).
- Writing more than one note for this single principle.
- Running Gate 1 / Gate 2 / Gate 3 explicit checks (three-gate logic is removed).
- Using `--target` / `--position continuation` / Luhmann-continuation logic (removed).
- Running `engram transcript` anything.
- Writing an episode or session summary.

## Expected RED baseline (pre-v2)

The pre-v2 fixture expected: identify one candidate, run three gates (all PASS), decide a Luhmann continuation under `10c1`, call `engram learn feedback --target 10c1 --position continuation`. That write mechanism and gate vocab are removed. The current skill writes with `--position top` and no gate checks.
