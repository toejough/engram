# Write-Memory Atom (O-A) Build Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship O-A as scoped in `docs/design/2026-07-04-atomic-skills-options.md` — a `write-memory` atom skill holding the mechanical vault-write procedures, invoked by name from recall and learn, behavior preserved.

**Architecture:** Three text artifacts, all already smoke-validated at `dev/eval/atoms/sandbox-texts/` (commit 58ad18a4): the atom (`write-memory-oa.md`), and marker-stripped copies of `recall-oa-new.md` / `learn-oa-new.md` replacing `skills/recall/SKILL.md` / `skills/learn/SKILL.md`. Verified 2026-07-04: the smoke's "old" texts differ from today's production texts ONLY by sandbox-marker lines, so the validated new texts are directly shippable. One content delta beyond the validated texts: the atom gains an explicit `--chunk-source` paragraph (the validated atom text omitted the flag; recall's absent case passes chunk IDs — a gap the smokes never exercised because no smoke scenario hit the Absent branch).

**Tech Stack:** Markdown skill texts; headless `claude -p` eval arms (claude-haiku-4-5, same tier as the smokes for comparability); `engram update` (SourceLocal) for deploy; bash harness under `dev/eval/atoms-build/`.

## Global Constraints

- **Behavior preservation is the bar** (Joe, verbatim: "refactors like this must preserve functionality").
- **Judgment-free atom scope is load-bearing**: the atom contains ONLY mechanical flag procedures; coverage verdicts, D2 decisions, and when-to-fire stay in parents (options doc bound 3).
- **Non-triggering description**: the atom's description names invocation-by-parent only; ship-gate 1 (Task 5) verifies nothing fires it autonomously.
- **Arms discipline** (notes 160/151 + standing): headless `claude -p`, NEVER Task-tool subagents; `ENGRAM_VAULT_PATH` → throwaway vault per arm; NO `--dangerously-skip-permissions` (a writable real repo is reachable) — scope with `--allowedTools`; fixture text contains no `@import`; `git status --short` on the real repo after every arm batch (contamination check).
- **Pre-registered branches are verdicts**: interpreted verbatim, never post-hoc. A RED that fails to fail is a premise finding (note 70) — STOP and report, don't engineer the fixture.
- Commit trailer `AI-Used: [claude]`; no push without Joe's word.
- Skill edits land via this plan's TDD tasks — no `skills/` edit outside them (writing-skills Iron Law; the smoke battery at 58ad18a4 + this plan's arms are the failing-test-first record).

## Cost envelope

~24 haiku arms + reruns ≈ $1–2 (T1: 9, T2: 6, T3: 6, canary: 1–3). Report actual spend.

---

### Task 1: Arm harness + skill-discovery canary

**Files:**
- Create: `dev/eval/atoms-build/run-arm.sh`
- Create: `dev/eval/atoms-build/results-2026-07-04.md` (append per task)

**Interfaces:**
- Produces: `run-arm.sh <arm-id> <project-dir> <prompt-file>` → writes `dev/eval/atoms-build/raw/<arm-id>.out` (final text) and `<arm-id>.jsonl` (transcript); exit code recorded.

The deployed-context tests (Tasks 3–5) need real Skill-tool discovery of project-level `.claude/skills/` in headless mode — an assumption the inline-fixture smokes never tested. Validate it before anything is scored.

- [ ] **Step 1: Write the runner**

```bash
#!/usr/bin/env bash
# run-arm.sh <arm-id> <project-dir> <prompt-file>
set -euo pipefail
ARM_ID="$1"; PROJ="$2"; PROMPT_FILE="$3"
RAW="$(cd "$(dirname "$0")" && pwd)/raw"; mkdir -p "$RAW"
export ENGRAM_VAULT_PATH="/tmp/oa-build-vault-${ARM_ID}"
mkdir -p "$ENGRAM_VAULT_PATH"
cd "$PROJ"
claude -p "$(cat "$PROMPT_FILE")" \
  --model claude-haiku-4-5 \
  --allowedTools "Bash(engram *) Read Skill" \
  > "$RAW/${ARM_ID}.out" 2>"$RAW/${ARM_ID}.err" || echo "EXIT:$?" >> "$RAW/${ARM_ID}.out"
```

- [ ] **Step 2: Build the canary project**

```bash
mkdir -p /tmp/oa-build-canary/.claude/skills/write-memory
# temporary canary copy: the validated atom text, marker stripped
sed '/SANDBOX-MARKER-OA-WM/d' \
  dev/eval/atoms/sandbox-texts/write-memory-oa.md \
  > /tmp/oa-build-canary/.claude/skills/write-memory/SKILL.md
printf 'Canary fixture. No other content.\n' > /tmp/oa-build-canary/CLAUDE.md
printf 'List the names of all skills available to you in this session. Names only.\n' \
  > /tmp/oa-build-canary/prompt.txt
```

- [ ] **Step 3: Run the canary and verify discovery**

Run: `dev/eval/atoms-build/run-arm.sh canary-1 /tmp/oa-build-canary /tmp/oa-build-canary/prompt.txt && grep -i "write-memory" dev/eval/atoms-build/raw/canary-1.out`

Expected: `write-memory` appears in the skill list. **If absent after 3 canary attempts: the deployed-context harness mechanism is INVALID — STOP, report the discovery failure; do not substitute inline fixtures for Tasks 3–5** (T3's non-fire test is meaningless without real description-based discovery).

- [ ] **Step 4: Contamination check + commit harness**

```bash
git status --short   # expect: only dev/eval/atoms-build/ additions
git add dev/eval/atoms-build/ && git commit -m "test(atoms-build): arm runner + skill-discovery canary

AI-Used: [claude]"
```

### Task 2: Author the three candidate texts (not yet production)

**Files:**
- Create: `dev/eval/atoms-build/candidate/write-memory.md`
- Create: `dev/eval/atoms-build/candidate/recall.md`
- Create: `dev/eval/atoms-build/candidate/learn.md`

**Interfaces:**
- Produces: the exact texts Tasks 3–6 deploy; Task 7 copies them to `skills/` byte-for-byte.

- [ ] **Step 1: Generate marker-stripped copies of the validated texts**

```bash
mkdir -p dev/eval/atoms-build/candidate
sed '/SANDBOX-MARKER-OA-WM/d'    dev/eval/atoms/sandbox-texts/write-memory-oa.md > dev/eval/atoms-build/candidate/write-memory.md
sed '/SANDBOX-MARKER-OA-RECALL/d' dev/eval/atoms/sandbox-texts/recall-oa-new.md   > dev/eval/atoms-build/candidate/recall.md
sed '/SANDBOX-MARKER-OA-LEARN/d'  dev/eval/atoms/sandbox-texts/learn-oa-new.md    > dev/eval/atoms-build/candidate/learn.md
```

- [ ] **Step 2: Apply the one content delta — the atom's `--chunk-source` paragraph**

In `dev/eval/atoms-build/candidate/write-memory.md`, insert after the fact block's supersedes paragraph (after the line "The binary maintains the inverse automatically." in the **Fact** section):

```markdown
### Chunk-source provenance (recall's absent case)

When the invoking skill provides chunk source IDs, pass one flag per chunk on the
`engram learn fact|feedback` call:

```
--chunk-source <source#anchor>
```

Repeatable; provenance only. The learn skill's own Step 2 path provides no chunk sources — omit the flag there.
```

- [ ] **Step 3: Verify the candidates differ from production exactly as pre-registered**

Run: `diff skills/recall/SKILL.md dev/eval/atoms-build/candidate/recall.md`
Expected: exactly the three hunks recorded in the options-cycle diff — 2.5C table rows (atom invocation for Absent, "consult write-memory" for supersedes syntax in Covered/Near), Step-4 synthesis rewrite, Step-4 QA block replaced by atom invocation. No marker lines, no other hunks.

Run: `diff skills/learn/SKILL.md dev/eval/atoms-build/candidate/learn.md`
Expected: exactly — Step 2 items 1–2 become atom invocations, the Step-2 supersedes/vocab rules lines removed (they live in the atom), Step 2.5 qa block becomes atom invocation. No other hunks.

- [ ] **Step 4: Commit candidates**

```bash
git add dev/eval/atoms-build/candidate/ && git commit -m "test(atoms-build): candidate texts — validated smoke texts, markers stripped, atom +--chunk-source

AI-Used: [claude]"
```

### Task 3: T1 — recall Absent case (the unsmoked branch): baseline → RED → GREEN

**Files:**
- Create: `dev/eval/atoms-build/fixtures/t1-*` (project dirs under /tmp, prompt + seed files in repo)
- Modify: `dev/eval/atoms-build/results-2026-07-04.md`

**Scenario (pinned):** the arm is mid-recall Step 2.5C with a cluster whose candidates are all off-topic (Absent), two chunk sources `sess-a.jsonl#turn-3`, `sess-b.jsonl#turn-7`, and cluster evidence for the principle "parallel Go table tests must not share the fixture map" phrased in the prompt. Checkpoint (all required): the arm's final output contains an executed-or-stated `engram learn feedback` (or `fact`) command with `--position top`, BOTH `--chunk-source` values, `--situation`, and the behavior/impact/action (or s/p/o) content flags; no invented flags.

**Arms:** n=3 per condition, claude-haiku-4-5, throwaway vault + project dir per arm, prompt instructs: "Execute recall Step 2.5C for this cluster using the skills available in this session" + the cluster data inline. Score old vs new on the same checkpoint (smoke scoring model: new ≥ old, no new-only disqualifier).

- [ ] **Step 1 (baseline): project dirs with CURRENT production texts as `.claude/skills/{recall,learn}` (no atom). Run 3 arms.**

Expected: 3/3 checkpoint (current text carries the flags inline; smokes ran 3/3 on adjacent scenarios).

- [ ] **Step 2 (RED): project dirs with candidate recall+learn, NO write-memory dir. Run 3 arms.**

Expected: ≥1 arm fails the checkpoint (invocation target missing → confabulated or missing flags; the O-B smoke measured exactly this mode at 0/3). **If 3/3 pass anyway, the pointer is behaviorally inert for this scenario — note-70 premise finding: STOP, report; the atom may be unnecessary for this branch.**

- [ ] **Step 3 (GREEN): add candidate write-memory to the same project spec. Run 3 arms.**

Expected: GREEN ≥ baseline (3/3), correct `--chunk-source` flags, no invented flags.

- [ ] **Step 4: contamination check; append per-arm table (old→RED→GREEN, of 3, with disqualifiers) to results; commit.**

```bash
git status --short   # expect clean except results file
git add dev/eval/atoms-build/ && git commit -m "test(atoms-build): T1 absent-case baseline/RED/GREEN results

AI-Used: [claude]"
```

### Task 4: T2 — deployed-context equivalence on QA capture (real Skill-tool dereference)

The smokes validated this scenario 3/3→3/3 with INLINE text; this re-run validates the real skill-discovery + invocation mechanism the production deploy uses.

**Scenario (pinned, from smoke S2):** arm is at recall Step 4 having written a synthesis note whose body cites `[[159.2026-07-02.eval-runs-checkpoint-per-trial]]`; prompt instructs executing the Step-4 QA capture per the skills available. Checkpoint: `engram learn qa` with `--contributors 159.2026-07-02.eval-runs-checkpoint-per-trial` (wikilink-derived, plural flag), `--question`, `--answer`, `--certainty`, `--source`; no invented flags.

- [ ] **Step 1: old arms (n=3): production texts as project skills, no atom.** Expected 3/3.
- [ ] **Step 2: new arms (n=3): candidate recall+learn+write-memory as project skills.** Expected ≥ old; arm invokes write-memory (transcript shows the Skill invocation) and emits correct flags.
- [ ] **Step 3: contamination check; append table; commit** (message: `test(atoms-build): T2 deployed-context qa-capture equivalence`, trailer as standard).

### Task 5: T3 — ship-gate 1: non-fire negative test (deployed sandbox)

**Scenario (pinned):** project dirs with ALL FOUR skills deployed (candidate recall/learn/write-memory + production please untouched). Two prompts × 3 arms:
- P-generic: "Refactor this function for readability" + a 20-line Go snippet inline. (No memory context at all.)
- P-adjacent: "I just learned that our CI requires make lint before push. Make sure we don't lose that." (Vault-adjacent — legitimate learn-skill or direct-engram territory; the atom must NOT fire autonomously.)

**Pre-registered verdict:** count Skill-tool invocations of `write-memory` NOT preceded in the same transcript by a recall/learn skill body instructing it. **PASS = 0 across all 6 arms. Any autonomous fire = ship-gate FAIL → do NOT deploy (Task 7 blocked); report to Joe with the transcript.** (A learn-skill-mediated invocation on P-adjacent is NOT a failure — that's the designed path.)

- [ ] **Step 1: run 6 arms; grep each transcript JSONL for write-memory Skill invocations; classify each as parent-instructed vs autonomous (quote the preceding context).**
- [ ] **Step 2: contamination check; append the 6-arm table (prompt, fired?, classification, quote); commit** (`test(atoms-build): T3 non-fire negative results`).

### Task 6: Ship-gate 2 — atom scope-pin audit

- [ ] **Step 1: mechanical scan**

Run: `grep -inE "covered|near|absent|verdict|judge|decide|gate|D2 bar" dev/eval/atoms-build/candidate/write-memory.md`

Expected surviving mentions ONLY: the "Judgment about WHEN to write … stays in the invoking skill" scope sentence, the "absent case" **label** naming which parent branch calls in (a name, not a criterion), and the D2-bar line that explicitly assigns the check to the caller ("caller's responsibility to check"). Any coverage criterion, verdict rule, or when-to-fire condition INSIDE the atom = FAIL → remove it or STOP if removal changes validated content.

- [ ] **Step 2: record the audit (grep output + disposition per hit) in results; commit** (`test(atoms-build): scope-pin audit`).

### Task 7: Apply to production + deploy (blocked on Tasks 3–6 all PASS + Gate B)

- [ ] **Step 1: copy candidates to production paths**

```bash
mkdir -p skills/write-memory
cp dev/eval/atoms-build/candidate/write-memory.md skills/write-memory/SKILL.md
cp dev/eval/atoms-build/candidate/recall.md       skills/recall/SKILL.md
cp dev/eval/atoms-build/candidate/learn.md        skills/learn/SKILL.md
```

- [ ] **Step 2: Gate B (design-fit, sonnet, fresh reviewer) over the full skills/ diff** — DRY (the 3-copy block now single-sourced), SRP (judgment in parents, mechanics in atom), YAGNI (no speculative atoms; amend/ingest untouched as pre-registered), reads-as-written-from-the-start.

- [ ] **Step 3: deploy + verify**

```bash
engram update
for s in recall learn write-memory; do
  diff -q "skills/$s/SKILL.md" "$HOME/.claude/skills/$s/SKILL.md" || echo "DEPLOY MISMATCH: $s"
done
```

Expected: three matches, no mismatch lines. (`engram update` SourceLocal copies repo `skills/` top-level dirs recursively — verified in `internal/update/update.go`; new dirs deploy automatically.)

- [ ] **Step 4: real-binary sanity pass (sandboxed)** — with `ENGRAM_VAULT_PATH=/tmp/oa-build-sanity`, run one real `engram learn feedback` and one `engram learn qa` with full flag sets copied from the deployed atom text; expect both to write notes (the atom's flag blocks are CLI-valid against the installed binary).

- [ ] **Step 5: commit production texts** (`feat(skills): write-memory atom — recall/learn write procedures single-sourced (O-A)`; body cites T1/T2/T3/audit results + smoke provenance 58ad18a4; trailer standard).

### Task 8: Documentation (please step 5; Gate C follows)

**Files:**
- Modify: `docs/GLOSSARY.md` — add entries: **atom** (a skill holding one mechanical procedure shared by parent skills, invoked by name; judgment stays in parents) and **non-triggering description** (a skill description naming only parent-instructed invocation so it never competes for autonomous firing; measured smoke evidence only — no official guidance exists either way).
- Modify: `docs/ROADMAP.md:175–181` — append one status line to the atoms charter: round 1 (write-memory) shipped this date; read-memory deliberately NOT extracted (recall's sequential cohesion, options doc bound 4); pointer to the options doc.
- Modify: `CLAUDE.md` — skills line in Directory Structure ("Source for the recall, learn, please, and route skills" → include write-memory), and the Key Files skill list.
- Check-and-update-if-stale: `docs/architecture/c1-system-context.md` (its learn/recall sequence notes now route writes through the atom — update the flow notes if they name the write commands inline).

- [ ] Steps: apply edits; run Gate C (relevance + clarity/cohesion, haiku, fresh reviewers); commit (`docs: write-memory atom — glossary, roadmap charter status, structure`).

---

## Verification summary (what "done" means)

| Gate | Measure | PASS branch (pre-registered) |
|---|---|---|
| Canary | write-memory listed by headless arm | listed; else harness INVALID, STOP |
| T1 | absent-case checkpoint, of 3: baseline / RED / GREEN | GREEN ≥ baseline AND RED shows ≥1 failure; RED 3/3-pass = note-70 STOP |
| T2 | qa-capture checkpoint, of 3: old / new (real skills) | new ≥ old |
| T3 | autonomous write-memory fires, of 6 arms | exactly 0; any fire = no deploy |
| Scope audit | judgment tokens in atom | only the 3 permitted mentions |
| Deploy | repo vs deployed diff, 3 files | byte-identical |
| Sanity | 2 real CLI writes from atom flag text | both succeed |

Honest bounds carried forward: haiku n=3 per cell (direction, not magnitude); T3's 6 arms bound obvious over-fire only — the 147× history was measured at far larger n, so the live-usage watch (does write-memory ever appear uninvited in real sessions) remains a standing check, cheap because transcripts are ingested.

## Pinned arm prompts (verbatim)

**T1 (all three conditions use this prompt):**

```
You are executing the recall skill's Step 2.5C (coverage judgment) using the skills available
in this session. Cluster data:
- candidate_l2s: one note, "12.2026-05-01.prefer-table-tests.md" — claim: "Go tests should use
  table-driven style for repeated cases." (Off-topic for the cluster principle below.)
- Cluster chunk members evidence this principle: parallel Go table tests must not share the
  fixture map — each subtest gets its own copy; sharing caused flaky failures twice.
- Chunk source IDs for provenance: sess-a.jsonl#turn-3 and sess-b.jsonl#turn-7
Judge coverage and take the action the skill directs for your judgment. Then state, verbatim,
the full command you ran (or would run).
```

**T2:** the Scenario-2 prompt verbatim from `docs/superpowers/plans/2026-07-04-atomic-skills-exploration.md` (§ pinned smoke scenarios, Scenario 2), with ONE substitution: the sentence directing the arm to the inline skill text is replaced by "using the skills available in this session". No other changes.

**T3 P-generic:**

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

**T3 P-adjacent:**

```
I just learned that our CI requires make lint before push. Make sure we don't lose that.
```

## Decisions log

- Joe 2026-07-04: "try out OA as scoped" — O-A picked against the options doc; ship-gates blessed by the pick.
- Amend blocks stay in recall (single-site, no duplication — YAGNI); ingest duplication stays (intentional, different roles); please/route untouched (measured anti-amnesia value, note 100).
- Atom gains `--chunk-source` beyond the validated text (gap: no smoke scenario exercised Absent); T1 validates the addition.
