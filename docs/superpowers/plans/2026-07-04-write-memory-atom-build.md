# Write-Memory Atom (O-A) Build Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. REQUIRED SUB-SKILL for every `skills/*/SKILL.md` edit: superpowers:writing-skills (Iron Law — the RED/GREEN/REFACTOR structure in Tasks 3–4 and the REFACTOR steps are executed UNDER that skill, not merely shaped like it). Steps use checkbox (`- [ ]`) syntax for tracking.

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

~27 haiku arms + reruns ≈ $1–2 (T1: 9, T2: 6, T2b: 3, T3: 6, canary: 1–3 where 3 is a RETRY budget — a first-attempt pass spends 1 arm). Report actual spend.

## Preflight (run before Task 1; any failure = STOP, report)

```bash
ls dev/eval/atoms/sandbox-texts/write-memory-oa.md dev/eval/atoms/sandbox-texts/recall-oa-new.md \
   dev/eval/atoms/sandbox-texts/learn-oa-new.md docs/design/2026-07-04-atomic-skills-options.md \
   docs/GLOSSARY.md docs/architecture/c1-system-context.md docs/architecture/c2-containers.md
sed -n '175,181p' docs/ROADMAP.md   # expect the atoms charter block
```

## Gate pre-registration (fixed reviewers, fresh context, recall-first — per the please skill)

- **Gate B** — fires in Task 7 Step 2, BEFORE deploy (Step 3). One design-fit reviewer (sonnet) over `git diff skills/`. Pass criterion: reviewer ACK with every finding fixed or rebutted. Deploy is blocked until ACK.
- **Gate C** — fires at end of Task 8 over every doc file touched. Two reviewers (haiku): relevance, clarity/cohesion. Same ACK bar.
- **Gate D** — fires before each commit batch's final commit per the please skill (clarity/standards, haiku) — commit messages only.

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
# run-arm.sh <arm-id> <project-dir> <prompt-file> [seed-dir]
# Emits raw/<arm-id>.jsonl (full event stream incl. Skill/tool invocations — T3's
# classification source) and raw/<arm-id>.out (final result text — checkpoint scoring source).
set -euo pipefail
ARM_ID="$1"; PROJ="$2"; PROMPT_FILE="$3"; SEED_DIR="${4:-}"
RAW="$(cd "$(dirname "$0")" && pwd)/raw"; mkdir -p "$RAW"
export ENGRAM_VAULT_PATH="/tmp/oa-build-vault-${ARM_ID}"
mkdir -p "$ENGRAM_VAULT_PATH"
[ -n "$SEED_DIR" ] && cp "$SEED_DIR"/* "$ENGRAM_VAULT_PATH/"
cd "$PROJ"
claude -p "$(cat "$PROMPT_FILE")" \
  --model claude-haiku-4-5 \
  --allowedTools "Bash(engram *) Read Skill" \
  --output-format stream-json --verbose \
  > "$RAW/${ARM_ID}.jsonl" 2>"$RAW/${ARM_ID}.err" || echo "EXIT:$?" >> "$RAW/${ARM_ID}.err"
jq -r 'select(.type=="result") | .result' "$RAW/${ARM_ID}.jsonl" > "$RAW/${ARM_ID}.out" || true
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

Run: `dev/eval/atoms-build/run-arm.sh canary-1 /tmp/oa-build-canary /tmp/oa-build-canary/prompt.txt && grep -i "write-memory" dev/eval/atoms-build/raw/canary-1.out && test -s dev/eval/atoms-build/raw/canary-1.jsonl && echo JSONL-OK`

Expected: `write-memory` appears in the skill list AND `JSONL-OK` prints (the stream-json event capture works — T3's classification depends on it; if the CLI rejects the `--output-format stream-json --verbose` combination, that is a harness bug to fix at the canary, same INVALID rule). **If discovery is absent after 3 canary attempts: the deployed-context harness mechanism is INVALID — STOP, report the discovery failure; do not substitute inline fixtures for Tasks 3–5** (T3's non-fire test is meaningless without real description-based discovery).

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

In `dev/eval/atoms-build/candidate/write-memory.md`, insert immediately BEFORE the `## QA pair capture` heading (unique anchor — the supersedes sentence appears in BOTH the Feedback and Fact sections and must not be used as the anchor):

```markdown
### Chunk-source provenance (recall's absent case)

When the invoking skill provides chunk source IDs, pass one flag per chunk on the
`engram learn fact|feedback` call:

```
--chunk-source <source#anchor>
```

Repeatable; provenance only. Pass one flag per chunk ID the invoking skill provided; if the
invoking skill provided none, omit the flag.
```

The wording is deliberately context-free — it conditions on the INPUT provided (chunk IDs or
none), never on which caller is invoking (a caller-identity conditional inside the atom would
be a judgment-seam fracture, note 78; Gate-A ask-alignment finding 2).

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

- [ ] **Step 0 (setup): create the per-arm project dirs and prompt file**

Copy each prompt from this plan's "Pinned arm prompts" appendix verbatim into its file: `dev/eval/atoms-build/prompts/{t1,t2,t2b,t3-generic,t3-adjacent}.txt` (the fenced block content only, no fences).

```bash
mkdir -p dev/eval/atoms-build/prompts
for cond in baseline red green; do for i in 1 2 3; do
  P="/tmp/oa-t1-${cond}-${i}"
  mkdir -p "$P/.claude/skills/recall" "$P/.claude/skills/learn"
  if [ "$cond" = baseline ]; then
    cp skills/recall/SKILL.md "$P/.claude/skills/recall/SKILL.md"
    cp skills/learn/SKILL.md  "$P/.claude/skills/learn/SKILL.md"
  else
    cp dev/eval/atoms-build/candidate/recall.md "$P/.claude/skills/recall/SKILL.md"
    cp dev/eval/atoms-build/candidate/learn.md  "$P/.claude/skills/learn/SKILL.md"
  fi
  if [ "$cond" = green ]; then
    mkdir -p "$P/.claude/skills/write-memory"
    cp dev/eval/atoms-build/candidate/write-memory.md "$P/.claude/skills/write-memory/SKILL.md"
  fi
  printf 'T1 fixture. No other content.\n' > "$P/CLAUDE.md"
done; done
```

Arm IDs: `t1-baseline-{1,2,3}`, `t1-red-{1,2,3}`, `t1-green-{1,2,3}` — e.g. `dev/eval/atoms-build/run-arm.sh t1-baseline-1 /tmp/oa-t1-baseline-1 dev/eval/atoms-build/prompts/t1.txt`.

- [ ] **Step 1 (baseline): project dirs with CURRENT production texts as `.claude/skills/{recall,learn}` (no atom). Run 3 arms.**

Expected: 3/3 checkpoint (current text carries the flags inline; smokes ran 3/3 on adjacent scenarios).

- [ ] **Step 2 (RED): project dirs with candidate recall+learn, NO write-memory dir. Run 3 arms.**

Expected: ≥1 arm fails the checkpoint (invocation target missing → confabulated or missing flags; the O-B smoke measured exactly this mode at 0/3). **If 3/3 pass anyway, the pointer is behaviorally inert for this scenario — note-70 premise finding: STOP, report; the atom may be unnecessary for this branch.**

- [ ] **Step 3 (GREEN): add candidate write-memory to the same project spec. Run 3 arms.**

Expected: GREEN ≥ baseline (3/3), correct `--chunk-source` flags, no invented flags.

- [ ] **Step 3b (REFACTOR, under writing-skills): re-read the atom + parent texts against the GREEN transcripts — any loophole an arm exploited or narrowly avoided (rationalization phrasing, skipped delegation, flag drift) gets tightened, then the affected condition re-runs n=3.** No transcript-evidenced loophole → record "no refactor needed" in results and move on.

- [ ] **Step 4: contamination check; append per-arm table (old→RED→GREEN, of 3, with disqualifiers) to results; commit.**

```bash
git status --short   # expect clean except results file
git add dev/eval/atoms-build/ && git commit -m "test(atoms-build): T1 absent-case baseline/RED/GREEN results

AI-Used: [claude]"
```

### Task 4: T2 — deployed-context equivalence on QA capture (real Skill-tool dereference)

The smokes validated this scenario 3/3→3/3 with INLINE text; this re-run validates the real skill-discovery + invocation mechanism the production deploy uses.

**Scenario (pinned, from smoke S2):** arm is at recall Step 4 having written a synthesis note whose body cites `[[159.2026-07-02.eval-runs-checkpoint-per-trial]]`; prompt instructs executing the Step-4 QA capture per the skills available. Checkpoint: `engram learn qa` with `--contributors 159.2026-07-02.eval-runs-checkpoint-per-trial` (wikilink-derived, plural flag), `--question`, `--answer`, `--certainty`, `--source`; no invented flags.

- [ ] **Step 0 (setup):** project dirs `/tmp/oa-t2-old-{1,2,3}` (production recall+learn as project skills) and `/tmp/oa-t2-new-{1,2,3}` (candidate recall+learn+write-memory), built with the same cp pattern as T1 Step 0; prompt file `dev/eval/atoms-build/prompts/t2.txt` from the appendix (identical prompt both conditions). Arm IDs `t2-old-{1,2,3}`, `t2-new-{1,2,3}`.

  **Vault seed (smoke parity — Gate-A code-alignment finding D):** the fixture wikilink's target must exist in each arm's throwaway vault, else `engram learn qa --contributors` errors at validation (the smoke arms ran seeded; an unseeded error could alter arm behavior). Create the seed once:

```bash
mkdir -p dev/eval/atoms-build/fixtures/vault-seed
cat > dev/eval/atoms-build/fixtures/vault-seed/159.2026-07-02.eval-runs-checkpoint-per-trial.md <<'EOF'
---
type: fact
situation: designing long-running eval harnesses
subject: eval trial results
predicate: must checkpoint per trial
object: JSONL append after every trial so orchestrator death loses at most one trial
luhmann: "159"
created: "2026-07-02"
source: fixture seed (short-basename copy for the T2/T2b instrument; the live vault's real note has the longer -and-survive-orchestrator basename)
---

Information learned: per-trial JSONL checkpointing bounds data loss.
EOF
```

  T2 and T2b arms pass it as the runner's 4th arg: `run-arm.sh t2-old-1 /tmp/oa-t2-old-1 dev/eval/atoms-build/prompts/t2.txt dev/eval/atoms-build/fixtures/vault-seed`. T1/T3 arms run unseeded (their scenarios reference no vault note).
- [ ] **Step 1: old arms (n=3): production texts as project skills, no atom.** Expected 3/3.
- [ ] **Step 2: new arms (n=3): candidate recall+learn+write-memory as project skills.** Expected ≥ old; arm invokes write-memory (transcript shows the Skill invocation) and emits correct flags.
- [ ] **Step 2b (T2b — learn parent, deployed): 3 arms, project dirs `/tmp/oa-t2b-new-{1,2,3}` with candidate recall+learn+write-memory, seeded with the same vault-seed dir; prompt file `prompts/t2b.txt` (appendix — the smoke S3 learn-correction scenario, with the added final clause "following the learn skill available in this session").** Checkpoint (from smoke S3, pre-registered): one `engram learn feedback` with `--behavior/--impact/--action` populated and a retrieval-shaped `--situation`; disqualifiers: `learn fact` for a correction, zero writes, >1 note. PASS branch: 3/3 ≥ the committed smoke-S3 old baseline (3/3 at 58ad18a4). Rationale: T1/T2 exercise recall's edited paths; this pins the LEARN parent's deployed-path flag correctness by measurement instead of the S3-inline + T2-mechanism composition inference (Gate-A ask-alignment round-2 residual).
- [ ] **Step 3: contamination check; append table (incl. T2b); commit** (message: `test(atoms-build): T2/T2b deployed-context equivalence`, trailer as standard).

### Task 5: T3 — ship-gate 1: non-fire negative test (deployed sandbox)

**Scenario (pinned):** project dirs with ALL FOUR skills deployed (candidate recall/learn/write-memory + production please untouched). Two prompts × 3 arms:
- P-generic: "Refactor this function for readability" + a 20-line Go snippet inline. (No memory context at all.)
- P-adjacent: "I just learned that our CI requires make lint before push. Make sure we don't lose that." (Vault-adjacent — legitimate learn-skill or direct-engram territory; the atom must NOT fire autonomously.)

**Pre-registered verdict:** count Skill-tool invocations of `write-memory` NOT preceded in the same transcript by a recall/learn skill body instructing it. **PASS = 0 across all 6 arms. Any autonomous fire = ship-gate FAIL → do NOT deploy (Task 7 blocked); report to Joe with the transcript.** (A learn-skill-mediated invocation on P-adjacent is NOT a failure — that's the designed path.)

- [ ] **Step 0 (setup):** project dirs `/tmp/oa-t3-{generic,adjacent}-{1,2,3}` each containing ALL FOUR skills as project skills, same cp pattern as T1 Step 0: candidate recall/learn/write-memory from `dev/eval/atoms-build/candidate/`, please from `skills/please/SKILL.md` (production, untouched). Prompt files `prompts/t3-generic.txt`, `prompts/t3-adjacent.txt` from the appendix. Arm IDs `t3-generic-{1,2,3}`, `t3-adjacent-{1,2,3}`.
- [ ] **Step 1: run 6 arms; grep each transcript JSONL for write-memory Skill invocations; classify each.**

**Classification rule (operational):** an invocation is PARENT-INSTRUCTED iff the same transcript shows, BEFORE the invocation, the recall or learn skill's body loaded (its Skill invocation or its text quoted) AND that body's text is what directs the write-memory call. Otherwise it is AUTONOMOUS. Quote the preceding transcript lines for every invocation either way. A learn-skill-mediated invocation on P-adjacent is PARENT-INSTRUCTED (the designed path); write-memory invoked with NO parent skill active is AUTONOMOUS = ship-gate FAIL.
- [ ] **Step 2: contamination check; append the 6-arm table (prompt, fired?, classification, quote); commit** (`test(atoms-build): T3 non-fire negative results`).

### Task 6: Ship-gate 2 — atom scope-pin audit

The options doc's gate 2 verbatim requires the audit "against the smoke-validated sandbox text" — so the PRIMARY check is a diff, the vocabulary scan is supplementary (Gate-A ask-alignment finding 1).

- [ ] **Step 1 (primary): diff against the validated text**

Run: `diff dev/eval/atoms/sandbox-texts/write-memory-oa.md dev/eval/atoms-build/candidate/write-memory.md`

Expected hunks, EXACTLY two: (1) the deleted `SANDBOX-MARKER-OA-WM` line; (2) the added `### Chunk-source provenance` block as pinned in Task 2 Step 2. Any third hunk = FAIL → remove it or STOP if removal would change validated content.

- [ ] **Step 2 (supplementary): conditional/judgment scan**

Run: `grep -inE "covered|near|absent|verdict|judge|decide|gate|D2 bar|omit|only when|only if|the (recall|learn|please) skill" dev/eval/atoms-build/candidate/write-memory.md`

**Classification rule (operational):** a hit FAILS iff it states a condition the ATOM's reader must evaluate about coverage, firing, or caller identity (an if/whether/when clause deciding WHETHER or WHICH-CASE to write, or naming a specific caller skill as the condition). A hit PASSES iff it is (a) a case LABEL naming which parent branch called in ("absent case" as a name), (b) an explicit delegation to the caller ("caller's responsibility to check", "stays in the invoking skill"), or (c) an input-conditioned mechanical rule ("if the invoking skill provided none, omit the flag" — conditions on provided input, not on caller identity or coverage). Expected PASS-classified hits: the scope sentence, the "absent case" label, the D2-bar delegation line, the chunk-source input rule. Record every hit + its classification.

- [ ] **Step 3: record the audit (diff output + grep output + disposition per hit) in results; commit** (`test(atoms-build): scope-pin audit — diff-primary + conditional scan`).

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

Run from the repo root — `engram update` picks SourceLocal by walking up from cwd to the repo's go.mod; from /tmp it silently deploys the remote module's texts instead (update.go:459–478).

```bash
engram update
for s in recall learn write-memory; do
  diff -q "skills/$s/SKILL.md" "$HOME/.claude/skills/$s/SKILL.md" || echo "DEPLOY MISMATCH: $s"
done
```

Expected: three matches, no mismatch lines. (`engram update` SourceLocal copies repo `skills/` top-level dirs recursively — verified in `internal/update/update.go`; new dirs deploy automatically.)

- [ ] **Step 4: real-binary sanity pass (sandboxed)** — with `ENGRAM_VAULT_PATH=/tmp/oa-build-sanity`, hand-fill each placeholder (`<kebab-slug>`, `<...>`) in the deployed atom's `engram learn feedback` block and its `engram learn qa` block with any concrete values (content immaterial — the check is flag validity), run both commands, then `ls /tmp/oa-build-sanity/` — expect the feedback note file and the qa `.q.md`/`.a.md` pair to exist. A flag-rejection error = FAIL (the atom text names a flag the binary rejects).

- [ ] **Step 5: commit production texts** (`feat(skills): write-memory atom — recall/learn write procedures single-sourced (O-A)`; body cites T1/T2/T3/audit results + smoke provenance 58ad18a4; trailer standard).

### Task 8: Documentation (please step 5; Gate C follows)

Scope note (Gate-A ask-alignment finding 4): the GLOSSARY entries are the options doc's explicit ship-gate 3; the ROADMAP/CLAUDE.md/c1/c2 items are BEYOND the options-doc scope, added per the note-64 maintenance obligation (every doc naming the old write path updates in the same effort). The c1/c2 edits are bounded to the write-step annotations only — not a re-architectural pass.

**Files:**
- Modify: `docs/GLOSSARY.md` — add entries: **atom** (a skill holding one mechanical procedure shared by parent skills, invoked by name; judgment stays in parents) and **non-triggering description** (a skill description naming only parent-instructed invocation so it never competes for autonomous firing; measured smoke evidence only — no official guidance exists either way).
- Modify: `docs/ROADMAP.md:175–181` — append one status line to the atoms charter: round 1 (write-memory) shipped this date; read-memory deliberately NOT extracted (recall's sequential cohesion, options doc bound 4); pointer to the options doc.
- Modify: `CLAUDE.md` — skills line in Directory Structure ("Source for the recall, learn, please, and route skills" → include write-memory), and the Key Files skill list.
- Check-and-update-if-stale: `docs/architecture/c1-system-context.md` AND `docs/architecture/c2-containers.md` — both name the write commands inline (c1 flow notes lines ~97–100 + the `engram learn qa` mention ~170; c2 L2 sequence-diagram arrows lines ~110 and ~136 shell `engram learn fact|feedback` directly). Update both to route the write through the write-memory atom invocation (note 64: the doc scrub is part of the change — every doc naming the old path, same effort).

- [ ] Steps: apply edits; run Gate C (relevance + clarity/cohesion, haiku, fresh reviewers); commit (`docs: write-memory atom — glossary, roadmap charter status, structure`).

---

## Verification summary (what "done" means)

| Gate | Measure (unit) | PASS branch (pre-registered) | STOP branch (pre-registered) |
|---|---|---|---|
| Canary | arms listing write-memory (arms passing, retry budget 3) | ≥1 lists it | 0/3 list it → harness INVALID; no inline-fixture substitute |
| T1 | arms hitting absent-case checkpoint (of n=3 per condition) | GREEN ≥ baseline AND RED < 3/3 | RED = 3/3 → note-70 premise finding, STOP + report |
| T2 | arms hitting qa-capture checkpoint (of n=3 per condition) | new ≥ old | — |
| T2b | arms hitting learn-correction checkpoint (of n=3, new condition) | 3/3 (≥ committed smoke-S3 old baseline 3/3) | — |
| T3 | AUTONOMOUS write-memory invocations (count across 6 arms) | exactly 0 | ≥1 → ship-gate FAIL, Task 7 blocked, report with transcript |
| Scope audit | diff hunks vs validated text (count) + FAIL-classified grep hits (count) | exactly 2 hunks AND 0 FAIL hits | 3rd hunk or FAIL hit whose removal changes validated content → STOP |
| Deploy | files diff-identical repo↔deployed (of 3) | 3/3 | — |
| Sanity | real CLI writes producing a vault note file (of 2 commands) | 2/2 note files exist in the sandbox vault | — |

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

**T2 (identical prompt for old and new arms — the corrected qa-round-1 instrument, with one added final sentence directing skill use).** The wikilink basename below is fixture-internal (verbatim from the qa-round-1 instrument); it deliberately does not match the live vault note, whose basename is longer (`...-and-survive-orchestrator`) — arms never touch the live vault (throwaway `ENGRAM_VAULT_PATH`), and checkpoint scoring matches the flag value against THIS fixture token, not the vault.

```
You just finished a deep recall Step 4. The user's question that prompted this recall was: 'Why did our eval runs lose data when the orchestrator died?' You wrote this synthesis note via engram learn fact:
---
type: fact
situation: designing long-running eval harnesses
subject: eval trial results
predicate: must checkpoint per trial
object: JSONL append after every trial so orchestrator death loses at most one trial
---

Information learned: per-trial JSONL checkpointing bounds data loss; see [[159.2026-07-02.eval-runs-checkpoint-per-trial]].

The synthesis body contains the wikilink [[159.2026-07-02.eval-runs-checkpoint-per-trial]].
Describe ALL remaining actions you take to finish the recall, in order, following the recall skill available in this session.
```

**T2b (learn parent, deployed — the smoke S3 scenario + skill-use clause):**

```
You are running the learn skill. Step 1 done: engram ingest --auto swept 3 chunks. Step 1.5: verdict OK, qa round-2 gate accumulating (1/20). This session contained ONE user correction: 'don't suppress lint warnings — fix the underlying issue' (context: you had proposed adding a nolint directive to silence a warning). List the EXACT commands you run for the remaining steps, following the learn skill available in this session.
```

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
