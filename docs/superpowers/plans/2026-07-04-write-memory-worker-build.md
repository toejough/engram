# Write-Memory WORKER Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. REQUIRED SUB-SKILL for every `skills/*/SKILL.md` edit: superpowers:writing-skills (Iron Law). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Ship write-memory as a WORKER skill — parents (recall, learn) judge and hand off; write-memory composes, executes, verifies, and reports the vault write — replacing the failed reference-card atom design.

**Supersession record (the please skill's dissent rule):** the Gate-A-approved reference-card plan (`2026-07-04-write-memory-atom-build.md`, executed through commit 3c697e34) measured its own core mechanism dead: **0/27 headless arms (haiku ×2 wordings, sonnet) dereferenced the atom mid-procedure**, while the whole-skill-as-next-action pattern fired 27/27 in the same transcripts. Joe's correction, verbatim: *"It sounds to me like you just drew the boundaries between skills at the wrong points."* Settled redraw: a skill-share is a WORKER invoked as the parent's next whole action (the please→learn pattern), never a mid-procedure reference fetch. Considered build-time expansion; Joe chose the worker redraw.

**Architecture:** One new skill `skills/write-memory/SKILL.md` (the worker, full text pinned in Task W2). Parent edits FROM TODAY'S PRODUCTION TEXTS (not the dead candidates): learn Step 2 items 1–2 + Step 2.5, recall 2.5C Absent row + Step 4 — each write site becomes a handoff. recall's Covered/Near amend rows are UNTOUCHED (amend is single-site — no duplication, stays self-contained per YAGNI). Judgment (which case, what content, wikilink extraction, certainty, supersedes DECISION) stays parent-side; mechanics + execution + CLI-error retry + reporting live worker-side.

**Tech Stack / carried infrastructure:** the committed atoms-build harness (run-arm.sh with stream-json + jq + seed-dir, run-arm-sonnet.sh, vault-seed fixture, canary result) is reused as-is; new arms and prompts land under `dev/eval/atoms-build/worker/`.

## Global Constraints

- **Behavior preservation is the bar** (Joe, standing): the write that lands in the vault must be the same write today's texts produce.
- **Boundary rule (settled, this cycle):** worker = executes handed-off work; NO coverage/D2/when-to-fire judgment inside it. Parents = all judgment; NO flag mechanics in parents' learn-write sites (amend rows exempt — single-site).
- **Instrument rule (measured, this cycle):** arm prompts must NOT presuppose self-composition ("state the command you ran" is FORBIDDEN in worker-arm prompts — it biased T1R against handoff). Deliverable framing: "complete the step(s), following the <skill> available in this session." Scoring is TRANSCRIPT-based (jsonl tool events + throwaway-vault contents), not output-prose-based.
- **Fixture rule (measured, this cycle):** the absent-case fixture must be judgment-UNAMBIGUOUS (the T1 candidate was bistable NEAR/ABSENT 8/4 — use an obviously unrelated candidate note).
- Arms discipline unchanged (headless claude -p, ENGRAM_VAULT_PATH throwaway, no bypassPermissions, --allowedTools "Bash(engram *) Read Skill", git status contamination check per batch, prompts verbatim from this plan's appendix).
- Pre-registered branches verbatim; any STOP → report to Joe (no auto-fallback — Joe chose pure worker over the fallback variant).
- Commit trailer `AI-Used: [claude]`; no push without Joe's word.

## Cost envelope

~30 arms (W1: 6, W2: 6, W3: 6 haiku + 3 sonnet, non-fire: 6, canary reuse: 0, retry margin ~3) ≈ $1.5–2.5. Report actual.

## Gate pre-registration

- **Gate B** — design-fit (sonnet, fresh) over `git diff skills/` in Task W6 BEFORE deploy; ACK blocks deploy.
- **Gate C** — relevance + clarity/cohesion (haiku ×2, fresh) over every doc touched, end of Task W7.
- **Gate D** — clarity/standards (haiku, fresh) over the step-6 commit prose.

---

### Task W1: Worker-round harness additions

**Files:**
- Create: `dev/eval/atoms-build/worker/prompts/{w1,w2,w3,w-generic,w-adjacent}.txt` (from this plan's appendix, fenced content only)
- Create: `dev/eval/atoms-build/worker/results-2026-07-04.md`

- [ ] **Step 1:** write the five prompt files from the appendix, verbatim.
- [ ] **Step 2:** `git status --short` (expect only worker/ additions); commit: `test(atoms-build): worker-round prompts + results scaffold` (+ trailer).

### Task W2: Author the worker skill text (candidate)

**Files:**
- Create: `dev/eval/atoms-build/worker/candidate/write-memory.md` (full text below, verbatim)

- [ ] **Step 1:** write the file with EXACTLY this content:

```markdown
---
name: write-memory
description: >
  Executes a vault write handed off by another skill (recall, learn): composes the engram
  command from the provided fields, runs it, verifies the result, and reports the written
  note path. Requires a handoff — do not fire on your own judgment that something is worth
  remembering.
---

# Write Memory — execute a handed-off vault write

You were invoked by a parent skill that already made the judgment (what to write and why).
Your job is the write itself: compose, execute, verify, report. Do not re-litigate the
parent's judgment; do not decide WHETHER to write.

## The handoff contract

The parent provides:

- **kind** — `fact`, `feedback`, or `qa`
- **content fields** — by kind, per the blocks below
- **source** — human-readable provenance string
- optional **chunk-sources** — `<source#anchor>` chunk IDs (provenance)
- optional **supersedes** — `<basename>|<type>|<claim>` (types: `updates|narrows|refutes`),
  when the parent determined this write corrects a surfaced note

If a required field is missing, ask for it from the in-session parent context — do not invent
content on the parent's behalf.

## Compose

kind=feedback:

```bash
engram learn feedback --slug <kebab-slug> --position top \
  --source "<source>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --behavior "<what was done>" --impact "<why it was wrong/costly>" --action "<what to do instead>"
```

kind=fact:

```bash
engram learn fact --slug <kebab-slug> --position top \
  --source "<source>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --subject "<the thing>" --predicate "<requires / must use / is>" --object "<the standard or value>"
```

kind=qa:

```bash
engram learn qa \
  --slug "<kebab summary of the question>" \
  --question "<verbatim question>" \
  --answer "<the answer body, copied — no re-derive>" \
  --contributors "<full-basename>" \
  --certainty "<high|medium|low>" \
  --source "<source>"
```

Append to any kind:

- one `--chunk-source <source#anchor>` per provided chunk ID
- `--supersedes "<basename>|<type>|<claim>"` if provided (repeatable)
- for qa: one `--contributors <full-basename>` per basename the parent provided

Rules:

- Never mix fact flags (`--subject/--predicate/--object`) with feedback flags
  (`--behavior/--impact/--action`) in one command.
- Never hand-author vocab tags or wikilinks — the binary assigns vocab automatically.

## Execute, verify, report

Run the command. On success the CLI prints the written note path(s).

- CLI error → read it, fix exactly the named problem (missing/typo'd flag, bad value), retry.
  Max 2 retries.
- Success → report the printed note path(s) to the parent flow in one line.
- Still failing after retries → report the exact command and the CLI error verbatim. Never
  silently skip a handed-off write.
```

- [ ] **Step 2:** commit: `test(atoms-build): worker candidate — write-memory as executor with handoff contract` (+ trailer).

### Task W3: Parent candidate texts (from PRODUCTION)

**Files:**
- Create: `dev/eval/atoms-build/worker/candidate/recall.md` (from `skills/recall/SKILL.md`)
- Create: `dev/eval/atoms-build/worker/candidate/learn.md` (from `skills/learn/SKILL.md`)

- [ ] **Step 1:** copy production texts, then apply EXACTLY these edits (each anchor verified unique before editing; assert-style, no chained commit):

**learn.md — Step 2 item 1 (Corrections):** replace the fenced `engram learn feedback` block (and its intro clause "Write feedback:") with:

```markdown
   **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=feedback,
   slug, source ("session <date>, context: <one-line what-was-happening>"), situation
   (retrieval-shaped), behavior, impact, action; plus supersedes details if this correction
   corrects an existing vault note. write-memory composes, executes, and reports the note path.
```

**learn.md — Step 2 item 2 (Explicit save-requests):** replace the fenced `engram learn fact` block (and "Write a fact:") with the same pattern, kind=fact, fields subject/predicate/object.

**learn.md — Rules block:** the two mechanical lines (`--supersedes` syntax line; vocab-tags-automatic line) are REMOVED (they live in the worker); the judgment rules (general principle, situation-as-retrieval-handle, one-note-per-principle, save-request-immediately, no-moments-write-nothing) stay, with the supersedes line rephrased judgment-side: "If the new lesson CORRECTS, narrows, or refutes an existing vault note, include the superseded note's basename, type, and claim in the handoff."

**learn.md — Step 2.5 (QA capture):** replace the fenced `engram learn qa` block with:

```markdown
For each uncaptured substantive Q&A from this session, **invoke the write-memory skill** with
this handoff — kind=qa, slug, verbatim question, answer body (copy; no re-derive), contributor
basenames, certainty, source ("ad-hoc capture, learn session <date>").
```

The contributor-extraction judgment lines (wikilinks-only, never free-listed, no pre-validation, report rejections, D2 bar, no-duplicate gate) STAY in learn — they are parent judgment.

**recall.md — 2.5C Absent row:** action cell becomes:

```
Invoke the **write-memory** skill with this handoff — kind=fact or feedback (pick per the
cluster's principle), situation + content fields, `--source "<descriptive>"`, the cluster's
chunk-source IDs, plus supersedes details if the new note corrects a surfaced note. write-memory
composes, executes, and reports the note path.
```

**recall.md — Covered/Near rows: UNTOUCHED** (self-contained amend, single-site).

**recall.md — Step 4:** the judgment bullets (conclusion-is-the-note, certainty-by-inference-mode, mark-derived-in-source, supersedes-decision, do-not-rot gate) STAY; the sentence "Write ONE synthesis note for the conclusion with `engram learn fact|feedback`:" becomes "Hand ONE synthesis note per conclusion to the **write-memory** skill (kind=fact or feedback, per the conclusion's shape):". The fenced `engram learn qa` block is replaced with:

```markdown
**After the synthesis note: if the synthesis body contains ≥1 `[[full-basename]]` wikilink,
ALSO invoke the write-memory skill** with kind=qa — verbatim question, the synthesis conclusion
as the answer, certainty matching the synthesis note's label, contributors = the wikilink
basenames, source "recall Step 4, session <date>".
```

The wikilink-extraction rules and the skip-if-no-wikilinks D2 bar STAY in recall.

**recall.md — Red-flags table:** the row "You're about to write `--relation` or hand-author wikilinks..." stays; ADD one row: `| You composed an engram learn command yourself at a write site | Write sites hand off to write-memory — parents judge, the worker writes |`.

- [ ] **Step 2 (verify):** `diff skills/learn/SKILL.md dev/eval/atoms-build/worker/candidate/learn.md` and same for recall — every hunk must map to an edit above; any unmapped hunk = fix before commit. Record both diffs in the results file.
- [ ] **Step 3:** commit: `test(atoms-build): worker-round parent candidates — write sites hand off, judgment stays` (+ trailer).

### Task W4: Handoff validation battery (transcript-scored)

Fixture dirs per arm (same cp pattern as the atom rounds): old arms = production recall+learn; new arms = candidate recall+learn+write-memory. Vault seed (`fixtures/vault-seed/`) for W2 arms only. Arm IDs `w{1,2,3}-{old,new}-{1,2,3}` + `ws3-new-{1,2,3}` (sonnet W3).

**Scoring (all scenarios, transcript-based):**
- *handoff-fired* (new arms only): jsonl shows a `Skill` tool_use with skill=write-memory.
- *write-correct*: jsonl Bash events show a well-formed `engram learn <kind>` (right kind; no fact/feedback flag mixing; required content flags present; chunk-sources/contributors when the scenario provides them) AND the throwaway vault contains the written note file(s) afterward.
- Old-arm baseline scores *write-correct* only.

| Scenario | Fixture | write-correct expectation |
|---|---|---|
| W1 learn Step-2 correction (terminal handoff) | the lint-suppression correction (appendix w1) | old ≥2/3 (arms execute; sonnet-round precedent), new ≥ old |
| W2 recall Step-4 qa (terminal, seeded vault) | the checkpoint-synthesis fixture (appendix w2) | old n=3 baseline, new ≥ old |
| W3 recall 2.5C loop (the risky site): TWO clusters — cluster A absent with an UNAMBIGUOUS off-domain candidate ("commit message format" note vs a parallel-test principle), cluster B covered (candidate states the principle verbatim → amend --activate, parent-side) | appendix w3 | old n=3, new ≥ old; new arms additionally: cluster-B amend composed by the PARENT (no write-memory handoff for amends) |
| W3-sonnet | same, claude-sonnet-4-6, n=3 new only | 3/3 expected, ≥2/3 passes |

**Pre-registered branches:**
- *handoff-fired* ≥2/3 per scenario (new arms) — the design's load-bearing measure. ANY scenario <2/3 → **STOP, report to Joe** (no auto-fallback).
- *write-correct*: new ≥ old per scenario; new < old anywhere → STOP, report.
- W3 cluster-B: an arm handing the AMEND to write-memory is a boundary violation — record it; ≥2 such arms → the parent text's seam line needs tightening (one refactor round allowed, then re-run W3).

- [ ] **Step 1:** build dirs, run W1 (6 arms), score from transcripts, contamination check, append table.
- [ ] **Step 2:** run W2 (6, seeded), score, check, append.
- [ ] **Step 3:** run W3 (6 haiku) + W3-sonnet (3), score, check, append.
- [ ] **Step 4:** commit: `test(atoms-build): worker handoff battery Wn results` (+ trailer).

### Task W5: Non-fire ship-gate (re-run with the worker description)

Same design as the atom round's T3 (all four skills deployed in fixture dirs; prompts w-generic + w-adjacent × 3 arms each; classification rule: PARENT-INSTRUCTED iff the transcript shows recall/learn loaded before the invocation and its text directs it; learn-mediated on w-adjacent is the designed path). **PASS = 0 AUTONOMOUS write-memory invocations across 6 arms; any = ship-gate FAIL, no deploy.** Note: the worker's description is intentionally more inviting than the atom's ("Executes a vault write...") — this re-run is NOT optional.

- [ ] **Step 1:** run 6 arms, classify each invocation with quoted context, append table, contamination check.
- [ ] **Step 2:** commit: `test(atoms-build): worker non-fire negative results` (+ trailer).

### Task W6: Scope audit + production apply + deploy (blocked on W4/W5 PASS + Gate B)

- [ ] **Step 1 (scope audit, both directions):**
  - Worker: `grep -inE "covered|near|absent|verdict|judge|whether|worth remembering" dev/eval/atoms-build/worker/candidate/write-memory.md` — FAIL-classify any hit where the WORKER's reader must evaluate a judgment; expected PASS hits only: the description's "do not fire on your own judgment" prohibition, the "already made the judgment / do not re-litigate" scope sentences.
  - Parents: `grep -n "engram learn " dev/eval/atoms-build/worker/candidate/{recall,learn}.md` — expected hits ONLY inside recall's Covered/Near amend rows' context and prose references; NO composable `engram learn` command blocks at write sites. Record both audits.
- [ ] **Step 2:** copy the three candidates to `skills/write-memory/SKILL.md`, `skills/recall/SKILL.md`, `skills/learn/SKILL.md`.
- [ ] **Step 3 (Gate B):** design-fit reviewer (sonnet, fresh) over `git diff skills/` — DRY/SRP/YAGNI + the settled boundary rule. ACK blocks deploy.
- [ ] **Step 4 (deploy):** from the REPO ROOT (`engram update` picks SourceLocal by walking up from cwd; from /tmp it silently deploys the remote module's texts): `engram update`, then `diff -q` all three skills repo↔`~/.claude/skills/` — 3/3 identical.
- [ ] **Step 5 (sanity):** with `ENGRAM_VAULT_PATH=/tmp/oa-worker-sanity`, hand-fill and run the worker's feedback block and qa block; `ls` the vault — feedback note + `.q.md`/`.a.md` pair exist.
- [ ] **Step 6:** commit: `feat(skills): write-memory worker — recall/learn write sites hand off to an executing skill` with body citing W1–W3/non-fire/audit results + the 0/27 reference-card record (+ trailer).

### Task W7: Documentation; Gate C

Scope note: GLOSSARY entries are options-doc ship-gate 3; the rest is note-64 maintenance, bounded to write-step annotations.

- Modify `docs/GLOSSARY.md`: **write-memory (worker skill)** — executes vault writes handed off by recall/learn; parents judge, the worker composes/executes/verifies/reports. **handoff contract** — the field set a parent passes (kind, content fields, source, chunk-sources, supersedes). **non-triggering description** — as previously scoped, now "requires a handoff" phrasing.
- Modify `docs/ROADMAP.md:175–181` charter: status line — write-memory shipped as a WORKER at the write seams (2026-07-04); the reference-card atom variant was built, measured dead at runtime (0/27 mid-procedure dereference), and superseded by Joe's boundary redraw; read-memory deliberately not extracted.
- Modify `CLAUDE.md`: Directory Structure skills line + Key Files list add write-memory; reconcile the intro's "Two skills — recall and learn" phrasing with the worker's existence.
- Update `docs/architecture/c1-system-context.md` (flow notes ~97–102, learn-qa mention) and `docs/architecture/c2-containers.md` (sequence arrows ~110/~136): write sites now hand off to write-memory, which shells `engram learn ...`. Bounded to write-step annotations.
- Update `docs/design/2026-07-04-atomic-skills-options.md`: append a dated postscript — O-A's reference-card form was eliminated post-pick by the 0/27 runtime-dereference measurement; shipped form is the worker redraw (Joe, 2026-07-04).
- [ ] Gate C (relevance + clarity/cohesion) to ACK; commit docs (+ trailer).

## Verification summary

| Gate | Measure (unit) | PASS (pre-registered) | STOP (pre-registered) |
|---|---|---|---|
| W1/W2/W3 handoff-fired | write-memory Skill invocations (arms of n=3 per scenario, new) | ≥2/3 each | any scenario <2/3 → STOP, report (no auto-fallback) |
| W1/W2/W3 write-correct | correct executed write + note file (of n=3 per condition) | new ≥ old each | new < old anywhere → STOP |
| W3 boundary | amends handed to worker (count, new arms) | 0–1 recorded | ≥2 → one tightening round, re-run W3 |
| W3-sonnet | handoff + write-correct (of 3) | ≥2/3 | — |
| Non-fire | AUTONOMOUS worker invocations (count, 6 arms) | exactly 0 | ≥1 → no deploy |
| Scope audit | FAIL-classified hits, both directions (count) | 0 | unremovable hit → STOP |
| Deploy | repo↔deployed identical (of 3 files) | 3/3 | — |
| Sanity | real CLI writes → note files (of 2) | 2/2 | — |

Honest bounds: n=3/cell haiku + n=3 sonnet on the risky site — direction, not magnitude; interactive-session handoff behavior remains observational (this session's own sub-skill record), not arm-measured; live watch continues via ingested transcripts.

## Pinned arm prompts (verbatim)

**w1 (learn correction — neutral deliverable):**

```
You are running the learn skill. Step 1 done: engram ingest --auto swept 3 chunks. Step 1.5: verdict OK, qa round-2 gate accumulating (2/20). This session contained ONE user correction: 'don't suppress lint warnings — fix the underlying issue' (context: you had proposed adding a nolint directive to silence a warning). Complete the remaining steps now, following the learn skill available in this session.
```

**w2 (recall Step 4 qa — neutral deliverable, seeded vault):**

```
You just finished a deep recall Step 4. The user's question that prompted this recall was: 'Why did our eval runs lose data when the orchestrator died?' You wrote this synthesis note via the write-memory handoff:
---
type: fact
situation: designing long-running eval harnesses
subject: eval trial results
predicate: must checkpoint per trial
object: JSONL append after every trial so orchestrator death loses at most one trial
---

Information learned: per-trial JSONL checkpointing bounds data loss; see [[159.2026-07-02.eval-runs-checkpoint-per-trial]].

The synthesis body contains the wikilink [[159.2026-07-02.eval-runs-checkpoint-per-trial]].
Complete ALL remaining recall actions now, in order, following the recall skill available in this session.
```

(Old-arm variant: the line "You wrote this synthesis note via the write-memory handoff:" reads "You wrote this synthesis note via engram learn fact:" — matching each condition's own text; everything else identical.)

**w3 (recall 2.5C two-cluster loop — neutral deliverable):**

```
Execute recall Step 2.5C now for BOTH clusters below, in order, following the recall skill available in this session.

Cluster 1:
- candidate_l2s: one note, "31.2026-06-01.commit-messages-imperative-mood.md" — claim: "Commit message subjects use imperative mood." 
- Cluster chunk members evidence this principle: parallel Go table tests must not share the fixture map — each subtest gets its own copy; sharing caused flaky failures twice.
- Chunk source IDs: sess-a.jsonl#turn-3 and sess-b.jsonl#turn-7

Cluster 2:
- candidate_l2s: one note, "47.2026-06-10.go-test-fixtures-per-subtest.md" — claim: "Parallel Go table tests must not share the fixture map; each subtest gets its own copy — sharing caused flaky failures in two sessions."
- Cluster chunk members evidence the same principle with no additional claims.
- Chunk source IDs: sess-c.jsonl#turn-9
```

**w-generic (non-fire):**

```
Refactor this function for readability. Reply with the refactored code only.

func P(m map[string]int, k []string) int {
    t := 0
    for i := 0; i < len(k); i++ {
        v, ok := m[k[i]]
        if ok == true {
            t = t + v
        } else {
            t = t + 0
        }
    }
    return t
}
```

**w-adjacent (non-fire):**

```
I just learned that our CI requires make lint before push. Make sure we don't lose that.
```

## Decisions log

- Joe 2026-07-04: reference-card atom boundaries were wrong ("drew the boundaries between skills at the wrong points"); redraw as WORKER at the write seams. Chosen over build-time expansion and over park.
- No auto-fallback: a handoff-fired STOP goes back to Joe.
- Amend rows stay in recall (single-site, YAGNI); W3 additionally polices that boundary.
- The W3 fixture seeds cluster-2 candidate content that a covered-judging arm amends; cluster-1's candidate is deliberately off-domain (fixes the T1 bistability).
