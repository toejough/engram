# 792g15 learn - Adoption Audit

Audit of [792a15 learn](../792a_product-spec.md#792a15-added-learn) and the [792b](../792b_technical-spec.md) requirements whose `Supports:` lines cite its scenarios. Method: read `agent-instructions/skills/learn/SKILL.md` line by line against each requirement, checked the baselines indexed in `agent-instructions/skills/learn/tests/README.md`, and ran the binary commands the skill calls on a temporary vault and chunk directory (HOME, XDG and ENGRAM_VAULT_PATH pointed at temporary directories). Paths in the Evidence column are relative to `agent-instructions/skills/` for `SKILL.md` files (`write-memory`, `learn`, `route`, `please`, `recall`), to `agent-instructions/guidance/` for `learn.md` and `shim.md`, and to the repository root otherwise.

| Spec ID | Verdict | Evidence |
|---|---|---|
| [792a15a Learn writes only explicit lessons](../792a_product-spec.md#792a15a-learn-writes-only-explicit-lessons) | match | learn/SKILL.md:12-19 (episode workflow gone), :218-221 (`No moments of any kind -> write nothing`), :305-306 red flags. Binary: `engram transcript` is `Unknown command`; `engram learn --help` lists only fact, feedback, qa, runbook |
| [792a15a1a A session with no lesson writes nothing](../792a_product-spec.md#792a15a1-uat-scenarios) | match | same evidence as the requirement |
| [792a15a1b No episode or summary is written](../792a_product-spec.md#792a15a1-uat-scenarios) | match | same evidence as the requirement |
| [792a15b The sweep comes first](../792a_product-spec.md#792a15b-the-sweep-comes-first) | doc fixed | learn/SKILL.md:30-51 matches (sweep first, report, continue on failure, skip when already swept unless closing). The wording `report the one-line tally it prints` did not match the binary: `engram ingest` prints one `ingest <path>: N chunks (R reused, E embedded)` line per re-chunked source (internal/cli/ingest_dedup.go:452) and nothing when the corpus is unchanged (observed: a second `--sweep` run printed no line for the unchanged file); `memory index up to date` is the phrase learn reports, not binary output. Requirement and one scenario reworded; SKILL.md left as is |
| [792a15b1a The sweep runs and is reported](../792a_product-spec.md#792a15b1-uat-scenarios) | doc fixed | reworded to the per-source lines or `memory index up to date` when nothing is printed; observed on a temp chunk dir |
| [792a15b1b A failed sweep does not block the lessons](../792a_product-spec.md#792a15b1-uat-scenarios) | match | learn/SKILL.md:44-45 `surface the error and continue to Step 2` |
| [792a15b1c An earlier sweep is not repeated](../792a_product-spec.md#792a15b1-uat-scenarios) | match | learn/SKILL.md:47-50 skip when a sweep already ran this session, closing learn excepted |
| [792a15b1d The closing learn always sweeps](../792a_product-spec.md#792a15b1-uat-scenarios) | match | learn/SKILL.md:30,47-49 `the closing learn ALWAYS sweeps` |
| [792a15c A correction moment gets a single-note capture](../792a_product-spec.md#792a15c-a-correction-moment-gets-a-single-note-capture) | match | learn/SKILL.md:21-29 (fast path skips Step 1 and Step 1.5, no `engram ingest --auto` mid-task, hands off to write-memory). The block still says `crystallize the ONE confirmed correction`, a wording nit already tracked in #788 item 4 that does not change the behavior specified |
| [792a15c1a A rejected approach is captured at once](../792a_product-spec.md#792a15c1-uat-scenarios) | match | same evidence as the requirement |
| [792a15c1b No mid-task sweep](../792a_product-spec.md#792a15c1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d The vocabulary check and its autonomous refit](../792a_product-spec.md#792a15d-the-vocabulary-check-and-its-autonomous-refit) | match | learn/SKILL.md:53-87. Binary: `engram vocab stats` prints `verdict: OK` or `verdict: REFIT_PENDING (<reason>)` (internal/cli/vocab_commands.go:1118) and `qa round-2 gate: READY (n>=20)` or `accumulating (n/20)` (:1127,1129); `engram vocab refit` prints `vocab refit: no structure found; vocabulary unchanged` (:222) or a JSON payload of `naming_requests` plus `fingerprint` and writes nothing (observed); the answers file shape `{names:[{cluster,term,description}], fingerprint}` and the stale-names error are internal/cli/vocab_apply.go:43,81,36. `Vocab refit applied: <version bump>. Triggered by: <reason>.` is the agent's own report line (the binary prints `vocab refit applied: version -> ...`, vocab_apply.go:136). The ROADMAP row for the round-2 gate exists (docs/ROADMAP.md, GATED band) |
| [792a15d1a A healthy vocabulary needs nothing](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d1b A pending refit is run autonomously](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d1c No structure found is reported as such](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d1d New clusters are named and the refit re-run](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d1e A stale answer restarts the refit](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d1f An applied refit is reported loudly](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15d1g The QA round-2 gate is reported and not run](../792a_product-spec.md#792a15d1-uat-scenarios) | match | same evidence as the requirement |
| [792a15e The four kinds of explicit lesson](../792a_product-spec.md#792a15e-the-four-kinds-of-explicit-lesson) | match | learn/SKILL.md:88-149 gives the four kinds and the handoff fields per kind (feedback with situation/behavior/impact/action; fact for a save-request; reversal as root-cause feedback; confirmed approach as feedback or runbook), source form `session <date>, context: ...` and supersedes details |
| [792a15e1a A correction becomes a feedback note](../792a_product-spec.md#792a15e1-uat-scenarios) | match | same evidence as the requirement |
| [792a15e1b A save-request becomes a fact note at once](../792a_product-spec.md#792a15e1-uat-scenarios) | match | same evidence as the requirement |
| [792a15e1c A reversal becomes a feedback note about the root cause](../792a_product-spec.md#792a15e1-uat-scenarios) | match | same evidence as the requirement |
| [792a15e1d A confirmed approach becomes a feedback note](../792a_product-spec.md#792a15e1-uat-scenarios) | match | same evidence as the requirement |
| [792a15e1e A correction of an existing note carries a supersession](../792a_product-spec.md#792a15e1-uat-scenarios) | match | same evidence as the requirement |
| [792a15f Reversals and repo-doc corrections](../792a_product-spec.md#792a15f-reversals-and-repo-doc-corrections) | match | learn/SKILL.md:137-149 (`Nobody needs to have SAID the correction`; repo-doc CORRECTION does not count; root cause, not the narrative); red flag at :311 |
| [792a15f1a A self-discovered reversal is captured](../792a_product-spec.md#792a15f1-uat-scenarios) | match | same evidence as the requirement |
| [792a15f1b A repo-doc correction does not count](../792a_product-spec.md#792a15f1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for](../792a_product-spec.md#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for) | match | learn/SKILL.md:150-210 (4a explicit praise of a specific behavior, 4b self-validated bet with a session-recorded outcome, three exemplars, runbook by shape, no extra fact for a runbook plus `do that going forward`) |
| [792a15g1a Specific praise is captured](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g1b A bare thanks is not](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g1c A self-validated bet is captured](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g1d A routine success is not](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g1e A procedure becomes a runbook](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g1f A single tweak stays feedback](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15g1g A runbook is not duplicated as a save-request](../792a_product-spec.md#792a15g1-uat-scenarios) | match | same evidence as the requirement |
| [792a15h Each note is placed by an ordered test](../792a_product-spec.md#792a15h-each-note-is-placed-by-an-ordered-test) | match | learn/SKILL.md:106-120 (continuation, then sibling, then top; in-session candidates only; no vault search). Binary placement: `--position continuation --target 1` gave ID `1a` (see the write-memory audit). The baseline test agent-instructions/skills/learn/tests/baseline-clean-write.md still says the Luhmann write path was removed; that skill-test edit needs the writing-skills cycle and the divergence is already logged in the decision log |
| [792a15h1a A sub-point is a continuation](../792a_product-spec.md#792a15h1-uat-scenarios) | match | same evidence as the requirement |
| [792a15h1b The same thought at the same level is a sibling](../792a_product-spec.md#792a15h1-uat-scenarios) | match | same evidence as the requirement |
| [792a15h1c An unrelated note is top-level](../792a_product-spec.md#792a15h1-uat-scenarios) | match | same evidence as the requirement |
| [792a15h1d Continuation is checked before sibling](../792a_product-spec.md#792a15h1-uat-scenarios) | match | same evidence as the requirement |
| [792a15h1e The vault is not searched](../792a_product-spec.md#792a15h1-uat-scenarios) | match | same evidence as the requirement |
| [792a15i A lesson states the general principle](../792a_product-spec.md#792a15i-a-lesson-states-the-general-principle) | match | learn/SKILL.md:213-217 (`State the general principle`, `--situation` as retrieval handle, one note per distinct principle) |
| [792a15i1a The principle, not the instance](../792a_product-spec.md#792a15i1-uat-scenarios) | match | same evidence as the requirement |
| [792a15i1b One note per principle](../792a_product-spec.md#792a15i1-uat-scenarios) | match | same evidence as the requirement |
| [792a15j A write to a parent is reported as done](../792a_product-spec.md#792a15j-a-write-to-a-parent-is-reported-as-done) | match | learn/SKILL.md:223-229. The warning text is the binary's: internal/cli/exchangestate.go:48 `engram: parent unreachable (retry after %s); %d offer(s) queued`. Binary: `ENGRAM_SERVER=... engram vocab stats` fails `ENGRAM_SERVER is no longer supported; set ENGRAM_PARENT=...`. Offers and outbox behavior are parent-sync's |
| [792a15j1a An unreachable parent is not a failure](../792a_product-spec.md#792a15j1-uat-scenarios) | match | same evidence as the requirement |
| [792a15j1b ENGRAM_SERVER is never set](../792a_product-spec.md#792a15j1-uat-scenarios) | match | same evidence as the requirement |
| [792a15k An answered question is captured once](../792a_product-spec.md#792a15k-an-answered-question-is-captured-once) | match | learn/SKILL.md:230-254 (traceability bar, contributors only from wikilinks, no pre-validation, surface a rejection, no duplicate pair, own Step 2 note excluded, prior note still earns a pair). dev/eval/cumulative/runbook_vs_skill/phase2/results/3.4b_write_memory_shim_only_retry2_sonnet5.md exists. Binary: qa with a `[[...]]` contributor wrote the pair |
| [792a15k1a A cited answer gets a pair](../792a_product-spec.md#792a15k1-uat-scenarios) | match | same evidence as the requirement |
| [792a15k1b Generic advice gets none](../792a_product-spec.md#792a15k1-uat-scenarios) | match | same evidence as the requirement |
| [792a15k1c A pair already written is not written again](../792a_product-spec.md#792a15k1-uat-scenarios) | match | same evidence as the requirement |
| [792a15k1d The agent's own note is the capture](../792a_product-spec.md#792a15k1-uat-scenarios) | match | same evidence as the requirement |
| [792a15k1e A prior note still earns a pair](../792a_product-spec.md#792a15k1-uat-scenarios) | match | same evidence as the requirement |
| [792a15k1f A rejected contributor is surfaced](../792a_product-spec.md#792a15k1-uat-scenarios) | match | same evidence as the requirement |
| [792a15l The re-parent batch mode](../792a_product-spec.md#792a15l-the-re-parent-batch-mode) | match | learn/SKILL.md:256-300. Payload fields match internal/cli/luhmann_reparent_apply.go:104-129 (`candidates` with note, note_excerpt, target, target_excerpt, similarity; `instruction`; `fingerprint`) and the answers shape `{reparenting:[{note,position,target}], fingerprint}` (:95-105); `engram update --help` lists `--reparent-luhmann`, `--answers` and `--dry-run` |
| [792a15l1a Every candidate note is answered once](../792a_product-spec.md#792a15l1-uat-scenarios) | match | same evidence as the requirement |
| [792a15l1b A high score does not decide](../792a_product-spec.md#792a15l1-uat-scenarios) | match | same evidence as the requirement |
| [792a15l1c The fingerprint is echoed and the next step is the user's](../792a_product-spec.md#792a15l1-uat-scenarios) | match | same evidence as the requirement |
| [792b18c3 Change detection](../792b_technical-spec.md#792b18c3-change-detection) | match | binary: the unchanged markdown file b.md printed no ingest line on the second sweep (cost nothing); supports the doc fix to the sweep report wording |
| [792b20c7 Statistics](../792b_technical-spec.md#792b20c7-statistics) | match | binary: `engram vocab stats` printed the version, terms/member-notes/untagged counts, `verdict:` and `qa round-2 gate:` lines and wrote nothing |
| [792b20c15 Refit routing](../792b_technical-spec.md#792b20c15-refit-routing) | match | binary: `engram vocab refit` on a temp vault with two untagged notes printed the naming payload and wrote nothing; the no-structure line is internal/cli/vocab_commands.go:222 |
| [792b20c16 Naming requests](../792b_technical-spec.md#792b20c16-naming-requests) | match | binary: the payload carried per cluster `cluster`, `member_count` and `exemplars` with `note`, `title`, `snippet` |
| [792b20c17 Naming answers](../792b_technical-spec.md#792b20c17-naming-answers) | match | internal/cli/vocab_apply.go:459-487 `parseRefitNames` checks fingerprint, unknown/duplicate/empty/incomplete names; stale error text at :36 |
| [792b20c18 Applying a derivation](../792b_technical-spec.md#792b20c18-applying-a-derivation) | match | internal/cli/vocab_apply.go:136 prints the applied line; the learn skill's own report wording is separate and specified by the product spec |
| [792b12c3 Locked write](../792b_technical-spec.md#792b12c3-locked-write) | match | binary: continuation placement allocated `1a` under the vault lock (see the write-memory audit) |
| [792b12c5 Question-and-answer write](../792b_technical-spec.md#792b12c5-question-and-answer-write) | match | binary: qa pair with a verified contributor written, question first |

## Summary

Rows by verdict: doc fixed 2, match 65. Total rows 67 (59 from 792a, 8 from 792b).

Every requirement and scenario of the learn capability was compared with `agent-instructions/skills/learn/SKILL.md` and with the binary commands the skill calls (`ingest`, `vocab stats`, `vocab refit`, `learn`, `update --reparent-luhmann`).

- One doc fix: the sweep report. The skill says to report `the one-line tally it prints`, but the binary prints one line per re-chunked source and nothing when nothing changed. The requirement and scenario now say that. SKILL.md is untouched; its wording is loose but its `or memory index up to date` clause already covers the empty case.
- No code problem found in the capability itself. Known wording nits in the fast-path block are tracked in #788 (item 4) and do not change specified behavior.
- The reversal of the Luhmann write path noted in baseline-clean-write.md is a stale skill test, not a spec problem; fixing it needs the writing-skills cycle and is already in the decision log.
- Not covered here: vault initialization (#795) and the controlled README (#796).
- Stale statements fixed outside the spec: docs/GLOSSARY.md still said `engram learn` has three forms (it has four), that learn's Step 2 hands off only feedback or fact, gave `--supersedes` in the space-separated form, listed a short handoff-contract field set, and linked the write-memory capability by bare ID.
