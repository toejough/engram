# Recall skill — baseline test index

These are reusable RED/GREEN scenario inputs for `superpowers:writing-skills` TDD — re-run the
relevant baseline before editing `agent-instructions/skills/recall/SKILL.md` (CLAUDE.md mandates it).

| baseline | locks which behavior | re-run before editing |
|---|---|---|
| `baseline-bootstrap-create.md` | Across a 4-cluster payload, empty/absent `candidate_l2s` (clusters 0/1) issues CREATE writes rather than being skipped for "no candidates to compare against"; non-empty covered candidates (clusters 2/3) get `amend --activate` / re-synthesis instead of a duplicate write. | Step 2.5 A/C |
| `baseline-empty-vault-skip-l2.md` | Pressure-tests the same empty-vault invariant against four adversarial rationalizations (colleague argument, score/cosine conflation, chunk-only-payload misread, full-flow omission) plus three write-discipline pressures (hurry, "merge into one note", "it's noise") — the absent→CREATE rule and one-write-per-cluster rule hold under pushback. | Step 2.5 A/C |
| `baseline-judgement-and-synthesis.md` | The end-to-end procedure shape: Step 0's three labeled blocks (ask/situation/plan) print before any retrieval call, Step 0.5 sweeps, exactly one `engram query` call carries all phrases, Step 2.5 runs inline (no dispatch), and Step 3's closing synthesis opens with the surfaced-item count and walks the Step 0 plan action-by-action. | Step 0, Step 0.5, Step 2, Step 2.5, Step 3 |
| `baseline-multi-query.md` | The single unified query invocation itself: all 5–15 Step 1 phrases passed as repeatable `--phrase` flags in ONE `engram query` call — never one call per phrase, never truncated/collapsed, never `--vault`/`--chunks-dir`/legacy `engram recall` — with Step 2.5 handled inline and one write per cluster. | Step 1, Step 2, Step 2.5 |
| `baseline-recency-conflict.md` | Step 2.5 B's recency tiebreaker: when two notes conflict on a convention, the newer `created` date wins; a candidate matching only the superseded stance is judged "near" (not "covered"), triggering a content re-synthesis via `engram amend` rather than a plain `--activate`. | Step 2.5 B |

The dated `*-RED-results.md` / `*-GREEN-results.md` files are NOT indexed here — they delete this
cycle (docs-restructure). Run records live in git history; `baseline-bootstrap-create.md`'s one
cited snippet is inlined in the scenario file itself so it stays self-contained.
