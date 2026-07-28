# Baseline pressure test — Step 1.5: derivational refit, not plan authoring

## Scenario

During the Step 1.5 liveness check, `engram vocab stats` prints
`verdict: REFIT_PENDING (growth: 47 notes since last refit, 21 days)`.

## Expected current-skill behavior (PASS)

- Run `engram vocab refit` autonomously — no deferral to the user, no YAML plan authoring.
- If the command prints a `naming_requests` JSON payload and exits without writing: name each
  new cluster from its exemplars, write an answer file `{"names": [{"cluster": N, "term":
  "kebab-case-term", "description": "one line"}], "fingerprint": "<echoed verbatim>"}` covering
  every new cluster exactly once, then re-run `engram vocab refit --names /path/answer.json`.
- If no new clusters: the single run applies directly.
- If the run prints `vocab refit: no structure found; vocabulary unchanged` (derivation K==0,
  vocab_commands.go:220-224): nothing was written, no version bump — report that plainly
  ("Vocab refit ran: no structure found, vocabulary unchanged.") and continue to Step 2.
- On a stale-names error ("the vault changed since the naming requests were emitted"): re-run
  `engram vocab refit` from scratch to regenerate the requests — never resubmit the old answer.
- **Report loudly:** "Vocab refit applied: <version bump>. Triggered by: <reason>."

## Failure modes that must FAIL this test

- Running `engram vocab refit --emit-request` (flag removed; the binary errors:
  `flag provided but not defined: --emit-request`).
- Authoring a YAML refit plan or running `engram vocab refit --plan ...` (removed).
- Proposing merges/splits/removals yourself — derivation is the binary's job; the agent only
  names new clusters.
- Deferring the refit to the user.
- Editing the fingerprint, or naming a subset/superset of the new clusters.
- Retrying `--names` with the same answer file after a stale-names error.
- On the no-structure output: claiming "Vocab refit applied" or any version bump (nothing was
  written), or silently dropping the trigger without reporting the outcome.

## Expected RED baseline (pre-derivational skill)

The old Step 1.5 instructed: `engram vocab refit --emit-request` → derive a YAML plan
(merges/splits/removals for orphans < 2 members and hubs > 25%) → `engram vocab refit --plan
/tmp/vocab-refit-plan.yaml`. Against the derivational binary (commits 76e240a6/b306afd5/7b52bf3a)
step 1 fails immediately with `flag provided but not defined: --emit-request`; an agent
following the old text either errors out or hallucinates a plan flow that no longer exists.
Verified 2026-07-28 against a fresh `go build ./cmd/engram` binary.
