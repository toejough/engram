# Roadmap / Issue-List True-Up Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring `docs/ROADMAP.md` and the GitHub issue list into agreement, and make the roadmap's own keep-current rule name the reconciliation it currently omits.

**Architecture:** Three units, all documentation and issue state. No code changes.

**Tech Stack:** `gh` CLI, markdown.

## Global Constraints

- **Reconcile on SECTION MEMBERSHIP, not mention** (vault note 564). A clean set-diff hides the real defect: an issue can be mentioned and still misplaced. The two properties to assert are (a) every OPEN issue appears in at least one actionable band, (b) no CLOSED issue OWNS a row in an actionable band.
- **Read every flagged hit before acting.** Cross-references live inside row prose (`#674 already shipped the evidence notes`) and dominate raw counts — a naive scan flagged 10 closed issues in actionable bands and all 10 were citations.
- **An annotation must REPAIR the row, not contradict it** (vault note 455). Rewrite so every column reads consistently; never bolt a correction onto a label that still asserts the old claim.
- **Correcting a document means replacing the wrong text, not appending beside it** (vault note 477).
- **Verify a disposition against the running binary or the decision record**, not the issue body, which can predate its own outcome (vault note 419).
- **Write status language in the voice the evidence supports at the moment of writing** (vault note 545).
- **A second Claude session is live in this repo.** Re-check `git status` and pull immediately before committing; do not assume the working tree is still exclusively ours.
- **Commit trailer is `AI-Used: [claude]`.**

## Doc-surface enumeration grep (Step 3, non-waivable)

The invariant this change touches is "#696 / pi-session ingest shipped status". Grep run over the repo for `#696`, `pi session`, `pi-session`, `piSessions`, `.pi ancestor`, `pi harness`, and separately for the annotation phrase family `issue still open` / `— reconcile`. Per-file disposition:

| File | Disposition | Reason |
|---|---|---|
| `docs/ROADMAP.md:200` | **rewrite** | The defect itself — the Type column reads `Shipped (issue still OPEN — reconcile)` while its own Note column asserts the work shipped. Self-contradicting row (note 455). |
| `README.md:12` | keep | Describes ingest coverage (Claude JSONL + Pi session files; OpenCode SQLite gap → #644). Accurate, and makes no claim about any issue's state. |
| `docs/architecture/c1-system-context.md:49,59` | keep | S5 and R4 describe the Pi JSONL read path as implemented. Accurate; no issue-state claim. |
| `docs/architecture/c2-containers.md:53,63` | keep | `--pi-sessions` documented as a live flag and a live override. Accurate. |
| `docs/architecture/adr.md:568` | keep | Source-precedence list includes `--pi-sessions`. Accurate. |
| `docs/superpowers/plans/2026-07-27-708-eval-vault-isolation-plan.md:1083` | keep | Historical plan doc recording the task as it stood when written. Repo convention is that plan bodies are the historical record and are not rewritten (the #687 plan states this explicitly). |
| `docs/superpowers/plans/2026-07-25-chunk-index-dedup-and-prune-fixes.md:146,155,166` | keep | References to the `piSessionSources` function, code-internal; unrelated to issue state. |
| `dev/eval/LEDGER.md` | N/A | Zero mentions of #696 — no measured claim exists to correct. |
| `agent-instructions/skills/recall/SKILL.md:220`, `dev/eval/atoms*/**` | N/A | The `reconcile` hits are the recall skill's anti-displacement paragraph — an unrelated word-sense collision, not this invariant. |

---

### Task 1: Close #696

**Files:** none. Issue state only.

**Interfaces:**
- Consumes: the verification evidence below.
- Produces: #696 CLOSED, which Task 2's row rewrite depends on being true.

Evidence that it is genuinely shipped, gathered against the running binary rather than the issue text:

- `engram ingest --help` lists `--pi-sessions   PI session transcript directory (JSONL; repeatable)`.
- `piSessionSources` is implemented at `internal/cli/ingest.go:518`, with the flag bound at `:24`.
- `engram ingest --auto` swept a real `.pi` source during this check (1 pi source ingested).
- The issue's own "Files to Modify" list (`internal/cli/ingest.go`, plus a `--pi-session-dir` or auto-detect flag) is satisfied by that code.

- [ ] **Step 1: Re-verify immediately before closing**

```bash
engram ingest --help | grep -- --pi-sessions
grep -n 'PiSessions' internal/cli/ingest.go | head -3
```

Expected: the flag line, and the struct-tag binding at line 24.

- [ ] **Step 2: Close with the evidence as the comment**

```bash
gh issue close 696 --comment "<evidence block: flag in --help, piSessionSources at ingest.go:518, a real .pi source swept by ingest --auto>"
```

- [ ] **Step 3: Verify**

```bash
gh issue view 696 --json state --template '{{.state}}'
```

Expected: `CLOSED`.

---

### Task 2: Repair the ROADMAP #696 row

**Files:**
- Modify: `docs/ROADMAP.md:200`

**Interfaces:**
- Consumes: Task 1's closure (the row states the issue is closed, so it must be).
- Produces: nothing downstream.

- [ ] **Step 1: Write the failing check**

The analogue of a test for a doc row is an assertion over the file. Add to a scratch check (not committed — this is a one-shot verification, not a permanent guard):

```bash
python3 - <<'PY'
import re, subprocess, json
src = open("docs/ROADMAP.md").read().split("\n")
SECTIONS = ["### NOW","### NEXT","### LATER","### GATED","### DEFERRED",
            "## Standing constraint","## Parked backlog","## Provenance","## How we prioritize"]
cur="(preamble)"; where={}
for line in src:
    for s in SECTIONS:
        if line.startswith(s): cur=s
    for n in set(re.findall(r"#(\d{3})", line)): where.setdefault(n,set()).add(cur)
state={str(r["number"]):r["state"] for r in json.loads(subprocess.run(
    ["gh","issue","list","--limit","200","--state","all","--json","number,state"],
    capture_output=True,text=True).stdout)}
ACT={"### NOW","### NEXT","### LATER","### GATED","### DEFERRED","## Parked backlog"}
bad=[n for n in where if state.get(n)=="OPEN" and not (where[n]&ACT)]
contradictions=[l for l in src if "still OPEN" in l or "— reconcile" in l]
print("open-but-only-in-provenance:", sorted(bad))
print("self-contradicting rows:", len(contradictions))
assert not bad and not contradictions, "FAIL"
print("PASS")
PY
```

Expected before the fix: `open-but-only-in-provenance: ['696']`, `self-contradicting rows: 1`, assertion FAILS.

- [ ] **Step 2: Rewrite the row**

Replace the Type cell `Shipped (issue still OPEN — reconcile)` with `Shipped`, and extend the Note cell's first sentence to record the closure date and its evidence, so the label and the note agree. Do NOT append a dated annotation beside the old text — replace it (notes 455, 477).

- [ ] **Step 3: Re-run the check**

Expected: `open-but-only-in-provenance: []`, `self-contradicting rows: 0`, `PASS`.

---

### Task 3: Name the reconciliation in the roadmap's keep-current rule

**Files:**
- Modify: `docs/ROADMAP.md`, the `**Keep-current rule**` block (currently two numbered items).

**Interfaces:**
- Consumes: nothing.
- Produces: nothing.

**Scope note for Gate A's ask-alignment reviewer:** this is the one item that goes beyond a literal true-up. The ask's second half — "I always want them kept up to date" — is captured in the vault as note 562, which governs the agent's behavior. This task puts the same requirement in the artifact a future reader actually opens. The roadmap's keep-current rule today covers filing and re-scoring but is silent on the issue tracker, which is precisely the gap that let #696 sit contradicted. **If the reviewer judges this scope creep, cut it — note 562 already carries the behavioral half.**

- [ ] **Step 1: Read the current rule**

```bash
sed -n '/Keep-current rule/,/^> Shorthand/p' docs/ROADMAP.md
```

- [ ] **Step 2: Add item 3**

Add one numbered item to the existing list, in the file's own voice:

> 3. **On every cycle close:** reconcile this file against the issue tracker in both directions — a closed issue leaves its actionable band, shipped work gains a Provenance row with its evidence, and an issue whose work has demonstrably landed gets closed rather than left open. Check SECTION MEMBERSHIP, not mention: an issue can be named here and still be in the wrong band.

- [ ] **Step 3: Verify the block still reads as one list**

```bash
sed -n '/Keep-current rule/,/^> Shorthand/p' docs/ROADMAP.md
```

Expected: three numbered items, consistent voice, no orphaned parenthetical.

---

### Task 4: Commit

- [ ] **Step 1: Re-check for the concurrent session**

```bash
git status --short && git fetch origin && git log --oneline HEAD..origin/main
```

Expected: clean tree apart from our own edits; if `origin/main` has moved, pull before committing.

- [ ] **Step 2: Commit** (Gate D passes over the message first)

```bash
git add docs/ROADMAP.md docs/superpowers/plans/2026-07-27-roadmap-issue-trueup.md
git commit -m "docs(roadmap): reconcile against the issue list; close #696

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage.** The ask has two halves. "True up both" → Tasks 1 and 2 (the single genuine defect) plus the verification in Task 2 Step 1 that proves nothing else is out of agreement. "Remember I always want them kept up to date" → vault note 562, written in step 1 of this cycle, plus Task 3's repo-side statement of the same rule.

**Placeholder scan.** None. Task 1's comment body is described rather than quoted verbatim because it is assembled from the four evidence lines listed above it; every other step carries its exact command.

**Type consistency.** The section-membership check in Task 2 Step 1 and Step 3 is the same script; `ACT` and `where` mean the same thing in both. Task 3's added item is numbered 3 because the existing list ends at 2 — verified in Step 1 before writing.

**Scope check.** Task 3 is flagged in its own body as the one item beyond a literal reading of the ask, with an explicit instruction to cut it if Gate A disagrees.
