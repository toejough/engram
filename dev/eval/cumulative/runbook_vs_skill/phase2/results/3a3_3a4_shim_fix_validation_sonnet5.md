# 3a.3 / 3a.4 shim-fix validation (sonnet5), 2026-09-20

## What was tested
The shim change (agent-instructions/guidance/shim.md: a "re-check phrase two" paragraph plus a second example, +559 bytes). RED arm = shim.md from git HEAD (without that paragraph, saved here as `3a3_shim_RED.md`); GREEN arm = the current shim.md. The two files differ ONLY by that paragraph (diff verified). Each trial is a fresh headless `claude -p` (model claude-sonnet-5) in an isolated CLAUDE_CONFIG_DIR (so `~/.claude/CLAUDE.md` is NOT loaded; verified: 0/60 transcripts contain that file's text or recall.md text; the shim text is present in 60/60; the "Re-check phrase two" text is present in 30/30 GREEN and 0/30 RED transcripts), a project CLAUDE.md = the arm's shim + a per-trial marker, no skills installed, a tiny fictional-domain repo per task, a copy of the real vault. The run is killed right after the agent issues its first `engram query`; only that command is scored. Runner: `3a3_phrase_two_red_green_runner.py`; raw data `3a3_phrase_two_red_green_sonnet5.jsonl`.

Scoring rule for the FIRST query's second `--phrase`: GOOD = (a) contains none of that task's ticket-specific names (kelp/shelf/bay/inventory, lighthouse/csv, beacon/scheduler/interval, marmot, quill), (b) none of its class-level ticket nouns (flag, alias, cli, export, report, log, crash, zero, config, loader, parser, validator, module, commands, flags, overhaul), and (c) contains a decision word (deciding, how much, how to, whether, before it is / calling it done, ...). Spot-checked by reading every string below. Caveat: the class-level noun list is strict ("a CLI tool" counts as ticket noun), so GOOD is conservative.

## Stage 1 result: 5 tasks (rename, feature, bugfix, refactor, docs) x 6 trials x 2 arms = 60 trials, 0 discarded

| arm | n | GOOD (process-decision phrase two) | no ticket-specific name | decision-shaped |
|---|---|---|---|---|
| RED | 30 | 0/30 | 24/30 | 0/30 |
| GREEN | 30 | 27/30 | 30/30 | 29/30 |

Per task GOOD (RED vs GREEN): rename 0/6 vs 6/6; feature 0/6 vs 5/6; bugfix 0/6 vs 6/6; refactor 0/6 vs 6/6; docs 0/6 vs 4/6.
Caveat: GREEN bugfix 6/6 are all a verbatim copy of the shim's own second example ("diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done"), so the bugfix cell shows the agent copies the example rather than generalizing.

Gate: PASSED (GREEN clearly better than RED).

### Every first-query phrase two
| arm | task | t | good | phrase 2 |
|---|---|---|---|---|
| GREEN | bugfix | 0 | Y | diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done |
| GREEN | bugfix | 1 | Y | diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done |
| GREEN | bugfix | 2 | Y | diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done |
| GREEN | bugfix | 3 | Y | diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done |
| GREEN | bugfix | 4 | Y | diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done |
| GREEN | bugfix | 5 | Y | diagnosing and fixing a defect in a service and deciding how much verification and review it needs before it is called done |
| GREEN | docs | 0 | Y | reconciling documentation against current behavior of a tool and deciding how to verify it before calling it done |
| GREEN | docs | 1 | Y | rewriting documentation to match current behavior of a tool and deciding how to verify it against the source before calling it done |
| GREEN | docs | 2 | Y | rewriting documentation across a codebase to match current behavior and deciding how to verify it before calling it done |
| GREEN | docs | 3 | Y | rewriting documentation in a repository to match current code behavior and deciding how to verify it before calling it done |
| GREEN | docs | 4 | N | revising documentation for a CLI so it matches the implemented behavior, and deciding how to verify it before calling it done |
| GREEN | docs | 5 | N | rewriting documentation for a CLI to match its current behavior and deciding how to verify it against the source before calling it done |
| GREEN | feature | 0 | N | implementing a scoped feature addition to an existing CLI tool, writing accompanying tests and documentation, and committing the result |
| GREEN | feature | 1 | Y | implementing a new feature in a CLI tool and deciding how to verify, document, and land it as a commit |
| GREEN | feature | 2 | Y | implementing a new feature in a tool end to end and deciding how to verify, document, and land it in a commit |
| GREEN | feature | 3 | Y | implementing a new output option in a CLI tool and deciding how to verify, document, and land it as a commit |
| GREEN | feature | 4 | Y | implementing a feature end to end in a codebase and deciding how to verify, document, and land it as a commit |
| GREEN | feature | 5 | Y | implementing a new feature in a tool with tests and docs and deciding how to verify and land it via commit |
| GREEN | refactor | 0 | Y | restructuring existing code into separate modules while preserving behavior and deciding how to verify it before calling it done |
| GREEN | refactor | 1 | Y | restructuring existing code into separate modules while preserving behavior and deciding how to verify it |
| GREEN | refactor | 2 | Y | restructuring existing code into modules while preserving behavior and deciding how to verify it before calling it done |
| GREEN | refactor | 3 | Y | restructuring existing code into separate modules while preserving behavior and deciding how to verify and review it before calling it done |
| GREEN | refactor | 4 | Y | restructuring existing code into separate modules while preserving behavior and deciding how to verify and land it |
| GREEN | refactor | 5 | Y | restructuring existing code into separate modules while preserving behavior and deciding how to verify it |
| GREEN | rename | 0 | Y | performing a full rename of an identifier across a codebase and verifying no old references remain before calling the work done |
| GREEN | rename | 1 | Y | carrying out a mechanical rename across a repo and deciding how to verify it is complete before calling it done |
| GREEN | rename | 2 | Y | carrying out a repo-wide rename of a user-facing identifier and deciding how much verification it needs before it is called done |
| GREEN | rename | 3 | Y | carrying out a mechanical rename across a codebase and deciding how to verify it is complete before calling it done |
| GREEN | rename | 4 | Y | carrying out a mechanical rename across a repo and deciding how to verify it is complete before calling it done |
| GREEN | rename | 5 | Y | carrying out a mechanical rename across a repo and deciding how much verification it needs before it is called done |
| RED | bugfix | 0 | N | fixing a bug in a scheduler component and updating documentation to describe the new behavior |
| RED | bugfix | 1 | N | fixing a crash bug in a scheduler component and updating its documentation to match the new behavior |
| RED | bugfix | 2 | N | fixing a crash bug in a scheduler component and updating its documentation to describe new behavior |
| RED | bugfix | 3 | N | fixing a crash bug in a scheduler component and updating the accompanying documentation |
| RED | bugfix | 4 | N | fixing a crash bug in a scheduler component and updating the accompanying documentation |
| RED | bugfix | 5 | N | fixing a crash bug in a scheduler and updating documentation to describe the changed behavior |
| RED | docs | 0 | N | updating documentation in a repo to match the current state of a CLI's commands and flags |
| RED | docs | 1 | N | rewriting documentation in a repository to match the current implementation of a command-line tool |
| RED | docs | 2 | N | updating documentation in a repo to match current code behavior after drift |
| RED | docs | 3 | N | rewriting documentation in a repo so it matches the current implementation of a command-line tool |
| RED | docs | 4 | N | updating documentation in a repo to match the current state of a CLI's commands and flags |
| RED | docs | 5 | N | updating documentation in a repo to match the current state of a CLI's commands and flags |
| RED | feature | 0 | N | implementing a new output format option for a reporting tool, including tests, docs, and a commit |
| RED | feature | 1 | N | implementing a new output-format feature in a CLI tool with tests, docs, and committing the result |
| RED | feature | 2 | N | adding a new output option to a CLI report tool in a repo, with tests, docs, and a commit |
| RED | feature | 3 | N | adding a new output-format option to a CLI tool in a repo, including tests, docs, and committing the change |
| RED | feature | 4 | N | adding a new output option to a CLI tool in a repo, including tests, docs, and committing the change |
| RED | feature | 5 | N | implementing a new feature option in a CLI tool, then writing tests and docs and committing the change |
| RED | refactor | 0 | N | restructuring a module into smaller modules in a behavior-preserving refactor |
| RED | refactor | 1 | N | splitting a module into separate modules in a behavior-preserving refactor |
| RED | refactor | 2 | N | splitting a module into separate modules while preserving behavior in a refactor |
| RED | refactor | 3 | N | restructuring a module into separate modules while preserving behavior in a refactor |
| RED | refactor | 4 | N | splitting an existing module into smaller modules in a behavior-preserving refactor |
| RED | refactor | 5 | N | splitting an existing module into smaller modules while preserving behavior in a refactor |
| RED | rename | 0 | N | renaming a CLI flag across a codebase with no backwards-compatible alias |
| RED | rename | 1 | N | renaming a CLI flag everywhere it appears across a repo with no backwards-compatible alias |
| RED | rename | 2 | N | renaming a CLI flag across a repository, updating every reference in code, docs, and tests |
| RED | rename | 3 | N | renaming a CLI flag everywhere it appears across code, docs, and tests in a repo |
| RED | rename | 4 | N | renaming a CLI flag everywhere it appears across a repository with no backwards-compatible alias |
| RED | rename | 5 | N | renaming a CLI flag everywhere it appears across a repo with no backward-compatible alias |

### Extra check with the real please-style prompt wording ("Please take this end-to-end: rename the kelp CLI's `--shelf` flag to `--bay`. It is a hard rename ... Get it landed properly in this repo."), 6 trials per arm (`3a3_phrase_two_red_green_pleaseprompt_sonnet5.jsonl`)

| arm | n | GOOD |
|---|---|---|
| RED | 6 | 0/6 |
| GREEN | 6 | 3/6 |

| arm | task | t | good | phrase 2 |
|---|---|---|---|---|
| GREEN | pleaserename | 0 | N | renaming a CLI flag with a breaking change and deciding how to implement, verify, and land it |
| GREEN | pleaserename | 1 | N | performing a hard rename of a CLI flag with no backward-compatible alias and landing the change in the repository |
| GREEN | pleaserename | 2 | Y | carrying out a mechanical rename across a codebase and deciding how to verify and land the change |
| GREEN | pleaserename | 3 | N | carrying out a rename of a user-facing CLI flag across a repo and deciding how to verify and land it |
| GREEN | pleaserename | 4 | Y | carrying out a breaking rename across a codebase and deciding how to verify and land it |
| GREEN | pleaserename | 5 | Y | carrying out a mechanical rename of a user-facing identifier across a codebase and deciding how to verify and land it |
| RED | pleaserename | 0 | N | renaming a CLI flag end-to-end with no backwards-compatible alias and landing the change in the repo |
| RED | pleaserename | 1 | N | renaming a CLI flag as a breaking change with no backward-compatible alias |
| RED | pleaserename | 2 | N | renaming a user-facing CLI flag across a repo and landing the change |
| RED | pleaserename | 3 | N | renaming a user-facing CLI flag across a codebase and landing the change in the repo |
| RED | pleaserename | 4 | N | renaming a user-facing CLI flag across a repository and landing the change end to end |
| RED | pleaserename | 5 | N | renaming a CLI flag across a repo and landing the change with no backwards-compatible alias |

Reading: the shim fix helps a lot but is not absolute; with the please-worded prompt half of GREEN phrase twos still name the flag ("renaming a CLI flag ... deciding how to ... land it").

## Stage 2 (3a.4): route retrieval regression with the new shim
Command: `probe_phase2.py --task route --arms R --shim-only --model sonnet5 --n 3` (raw `3a4_route_regression_after_shim_fix_sonnet5.jsonl`).

| run | n | route runbook found (via engram query) | followed_all | end_state | $ |
|---|---|---|---|---|---|
| prior route shim-only (before shim change, `change_route-skill-to-runbook_...`) | 3 | 3/3 | 1/3 | 0/3 | 2.04 |
| prior route shim-only (`postfix763_...`) | 3 | 3/3 | 3/3 | 3/3 | 1.43 |
| this run (new shim) | 3 | 3/3 | 3/3 | 3/3 | 1.58 |

Gate: PASSED (route still retrieves 3/3).

## Spend
Stage 1 about $3.5 (token-based estimate at assumed $3/$15 per Mtok because runs were killed after the first query and so have no reported cost: main 60 trials ~2.55, please-prompt 12 trials ~0.82, pilot ~0.16), Stage 2 $1.58 (harness-reported).
