# Roadmap / Issue-List True-Up Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring `docs/ROADMAP.md` and the GitHub issue list into agreement, and confirm the standing "always keep these current" instruction is durably captured where it governs behavior — vault note 562, not a second copy in the repo.

**Architecture:** Two units plus a baseline capture. Issue state and one documentation row. No code changes.

**Tech Stack:** `gh` CLI, markdown, python3 for the reconciliation check.

## Global Constraints

- **Reconcile on SECTION MEMBERSHIP, not mention** (vault note 564). A clean set-diff hides the real defect: an issue can be mentioned and still misplaced. Assert two properties: every OPEN issue appears in at least one actionable band, and no CLOSED issue OWNS a row in an actionable band.
- **Read every flagged hit before acting.** Cross-references live inside row prose (`#674 already shipped the evidence notes`) and dominate raw counts — a naive scan flagged 10 closed issues in actionable bands and all 10 were citations.
- **An annotation must REPAIR the row, not contradict it** (vault note 455), and correcting a document means replacing the wrong text, not appending beside it (vault note 477).
- **Verify a disposition against the running binary or the decision record**, not the issue body, which can predate its own outcome (vault note 419).
- **A second Claude session is live in this repo.** Re-check `git status` and fetch immediately before committing.
- **Commit trailer is `AI-Used: [claude]`.**

## Doc-surface enumeration grep (Step 3, non-waivable)

Invariant touched: "#696 / pi-session ingest shipped status".

Terms actually run, using the real identifiers and phrasings present in the tree (an earlier draft listed `piSessions` and `.pi ancestor`, which match nothing anywhere — the real forms are the capitalized struct field, the function name, and "ancestor `.pi` dir"): `#696`, case-insensitive `pi[ _-]session`, exact-case `PiSessions`, `piSessionSources`, `pi harness`, plus the annotation phrase family `issue still open` and `— reconcile`.

| File | Disposition | Reason |
|---|---|---|
| `docs/ROADMAP.md:200` | **rewrite** | The defect itself — Type cell reads `Shipped (issue still OPEN — reconcile)` while its own Note cell asserts the work shipped. Self-contradicting row (note 455). Target substring is unique in the file. |
| `README.md:12` | keep | Describes ingest coverage (Claude JSONL + Pi session files; OpenCode SQLite gap → #644). Accurate; no issue-state claim. |
| `docs/architecture/c1-system-context.md:49,59` | keep | S5 and R4 describe the Pi JSONL read path as implemented. No issue-state claim. |
| `docs/architecture/c2-containers.md:53,63` | keep | `--pi-sessions` documented as a live flag and a live override. Accurate. |
| `docs/architecture/adr.md:568` | keep | Source-precedence list includes `--pi-sessions`. Accurate. |
| `docs/superpowers/plans/2026-07-27-708-eval-vault-isolation-plan.md:1083` | keep | Historical plan doc. Repo convention: plan bodies are the historical record and are not rewritten. |
| `docs/superpowers/plans/2026-07-25-chunk-index-dedup-and-prune-fixes.md:146,155,166` | keep | References to `piSessionSources`, code-internal; unrelated to issue state. |
| `docs/GLOSSARY.md:133`, `docs/FEATURES.md:193,224` | keep | Mention Pi JSONL and the `.pi`-ancestor precedence/sweep; neither asserts issue state. Added after the docs-alignment reviewer independently scanned them; line numbers added after the code-alignment reviewer noted this was the table's only uncited row. |
| `dev/eval/LEDGER.md` | N/A | Zero hits on every invariant term. No measured claim exists to correct. |
| `agent-instructions/skills/recall/SKILL.md:220`, `dev/eval/atoms*/**` (5 sandboxed copies) | N/A | The `reconcile` hits are the recall skill's anti-displacement paragraph — an unrelated word-sense collision. |

---

### Task 1: Capture the baseline, then close #696

**Files:** none. Issue state only.

**Interfaces:**
- Consumes: nothing.
- Produces: #696 CLOSED, and the recorded "before" evidence Task 2 compares against.

**Ordering note (Gate A, code-alignment finding 1).** The reconciliation check reads live issue state, so it MUST run before `gh issue close`. Run it after the close and `open-but-only-in-provenance` prints `[]` regardless of whether the roadmap was ever fixed — the evidence trail would silently stop distinguishing fixed from broken. Baseline first, always.

- [ ] **Step 1: Capture the baseline BEFORE closing anything**

```bash
cd /Users/joe/repos/personal/engram && python3 - <<'PY'
import re, subprocess, json
src = open("docs/ROADMAP.md").read().split("\n")
SECTIONS = ["### NOW","### NEXT","### LATER","### GATED","### DEFERRED",
            "## Standing constraint","## Parked backlog","## Provenance","## How we prioritize"]
cur="(preamble)"; where={}
for line in src:
    for s in SECTIONS:
        if line.startswith(s): cur=s
    # 3-or-more digits, not followed by another digit: matches #696 and a future #1024,
    # while ignoring the "Change #1"/"Change #2" ordinals that appear in row prose.
    for n in set(re.findall(r"#(\d{3,})(?!\d)", line)): where.setdefault(n,set()).add(cur)
state={str(r["number"]):r["state"] for r in json.loads(subprocess.run(
    ["gh","issue","list","--limit","200","--state","all","--json","number,state"],
    capture_output=True,text=True).stdout)}
ACT={"### NOW","### NEXT","### LATER","### GATED","### DEFERRED","## Parked backlog"}
bad=[n for n in where if state.get(n)=="OPEN" and not (where[n]&ACT)]
# Check the TYPE CELL, not the whole line. A repaired row may legitimately QUOTE its old
# wrong text as a disclosure (the convention row #687 uses), and a substring scan would then
# fire on the fix itself — the quoted-prose false-positive class of vault note 463.
# Table shape: | item | type | note | bar |  ->  split("|")[2] is the type cell.
contradictions=[l for l in src
                if len(l.split("|"))>=3 and "still OPEN" in l.split("|")[2]]
print("open-but-only-in-provenance:", sorted(bad))
print("type-cell contradictions:", len(contradictions))
print("PASS" if not bad and not contradictions else "FAIL (expected at baseline)")
PY
```

Real output, executed against the live repo while #696 was still open:

```
open-but-only-in-provenance: ['696']
self-contradicting rows: 1
FAIL (expected at baseline)
```

- [ ] **Step 2: Re-verify the shipped evidence**

```bash
engram ingest --help | grep -- --pi-sessions
grep -n 'PiSessions' internal/cli/ingest.go | head -2
grep -n 'func piSessionSources' internal/cli/ingest.go
```

Expected: the flag line; the struct-tag binding at `:24`; the function at `:522` (its doc comment begins at `:518`).

- [ ] **Step 3: Close, with the evidence verbatim as the comment**

```bash
gh issue close 696 --comment "Shipped 2026-07-24; closing after verifying against the running binary rather than the issue text.

- \`engram ingest --help\` lists \`--pi-sessions   PI session transcript directory (JSONL; repeatable)\`.
- Flag bound at \`internal/cli/ingest.go:24\`; \`piSessionSources\` implemented at \`internal/cli/ingest.go:522\` (doc comment from :518).
- \`engram ingest --auto\` swept a real ancestor \`.pi\` source (\`~/.pi/agent/run-history.jsonl\`) during this check.

This issue's own Files-to-Modify list — \`internal/cli/ingest.go\` plus a \`--pi-session-dir\` or auto-detect flag — is satisfied by the shipped \`--pi-sessions\` flag and the \`.pi\`-ancestor auto-sweep. Recorded in docs/ROADMAP.md's Provenance table."
```

- [ ] **Step 4: Verify**

```bash
gh issue view 696 --json state --template '{{.state}}'
```

Expected: `CLOSED`.

---

### Task 2: Repair the ROADMAP #696 row

**Files:**
- Modify: `docs/ROADMAP.md:200`

**Interfaces:**
- Consumes: Task 1's closure — the new text asserts the issue is closed, so it must be.
- Produces: nothing downstream.

- [ ] **Step 1: Replace the Type cell**

Find (unique in the file, `grep -c` = 1):

```
| Shipped (issue still OPEN — reconcile) |
```

Replace with:

```
| Shipped |
```

- [ ] **Step 2: Replace the Note cell's first clause**

Find:

```
**#696** (Pi session-file ingest) — shipped 2026-07-24:
```

Replace with:

```
**#696** (Pi session-file ingest) — shipped 2026-07-24, issue closed 2026-07-27 after re-verifying against the running binary (`--pi-sessions` in `engram ingest --help`; `piSessionSources` at `internal/cli/ingest.go:522`; `ingest --auto` sweeping a real ancestor `.pi` source):
```

Replace, do not append (notes 455, 477). Every other cell in the row is unchanged.

- [ ] **Step 3: Re-run the Task 1 Step 1 check**

Expected now:

```
open-but-only-in-provenance: []
self-contradicting rows: 0
PASS
```

Both lines must flip. `open-but-only-in-provenance` going empty is necessary but NOT sufficient on its own — it also empties merely by closing the issue, which is why the baseline was captured first and why `self-contradicting rows` is the line that proves the row was actually repaired.

---

### Task 3: Commit

- [ ] **Step 1: Re-check for the concurrent session**

```bash
git status --short && git fetch origin && git log --oneline HEAD..origin/main
```

Expected: only our own edits; if `origin/main` has moved, pull before committing.

- [ ] **Step 2: Commit** (Gate D passes over the message first)

```bash
git add docs/ROADMAP.md docs/superpowers/plans/2026-07-27-roadmap-issue-trueup.md
git commit -m "docs(roadmap): reconcile against the issue list; close #696

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage.** The ask has two halves.

*"True up both the roadmap and the issue list"* → Task 1 mutates the issue tracker (closing #696), Task 2 mutates the roadmap. Both halves are touched, proportionate to the verified defect surface: one issue. The baseline output pasted into Task 1 Step 1 is the evidence for "one", not a prediction — it was executed against the live repo.

*"Remember that I always want them kept up to date"* → vault note 562, written in step 1 of this cycle, which governs agent behavior on every future cycle. Its content, quoted so this claim is checkable without leaving the document:

> **situation:** finishing a unit of work in one of Joe's repos — closing an issue, shipping a change, filing new work — or being asked what to work on next
> **subject:** docs/ROADMAP.md and the repo's GitHub issue list
> **predicate:** are kept current as standing practice, never only when asked
> **object:** …a closed issue leaves the NOW band; shipped work moves to Provenance with its evidence; a newly filed issue is scored and placed in the same cycle it is filed; an issue whose work has demonstrably shipped gets closed rather than left open. Drift between the two artifacts IS the defect…

A repo-side copy of this rule was drafted and cut at Gate A: it repairs no existing defect, and a rule with two unlinked homes drifts.

**Placeholder scan.** None. Every command is verbatim, including the `gh issue close` comment body.

**Type consistency.** The reconciliation script appears once (Task 1 Step 1) and is re-run by reference in Task 2 Step 3, so `where`, `state`, and `ACT` cannot diverge between the before and after readings.

**Scope check.** Two mutations, one file and one issue. The one item that exceeded the ask was cut rather than argued.
