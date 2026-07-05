# Docs Restructure Review Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a gate-passed suggestions report (`docs/design/2026-07-04-docs-restructure-suggestions.md`) giving Joe concrete, evidence-anchored recommendations to make the repo's documentation live-only, SRP-shaped, diagram-supported, and verifiably correct — review only; no restructuring executed this cycle.

**Architecture:** Hybrid orchestration. Mechanical ground truth (file inventory + inbound-reference graph) is built inline first, then a Workflow fans out six per-angle review agents (each recalls first, per `route`) over that shared ground truth, with adversarial verification of correctness findings. The orchestrator synthesizes the verified outputs into one report with per-file dispositions and decision points.

**Tech Stack:** Claude Code Workflow tool (fan-out), engram recall (per-agent), git/grep (ground truth), mermaid (diagram proposals only — no diagrams built this cycle).

## Global Constraints

- **Review only.** No doc is deleted, moved, merged, or rewritten this cycle. The report proposes; Joe disposes.
- **The ask (verbatim coverage contract, note 155):** (a) docs live-only — historical committed then deleted; (b) SRP — one glossary, one roadmap, one features doc, one ADR/standards doc, etc.; (c) sane folder + README explaining the breakdown; one obvious place to go; (d) mermaid diagrams (C4, sequence, flow) for key features; (e) remaining docs obviously correct vs existing **code** AND **skills** (future-work docs exempt from "describes current code"); (f) complete and concise; (g) thorough multi-angle review → concrete suggestions.
- **Every proposed edit in the report pins a verbatim anchor** — quoted target text + `file:line` (note 170). Relational anchors are findings-in-waiting.
- **Every measured/derived number carries an evidence pointer + vintage** (notes 156, 162); every correctness finding is verified-against-code or explicitly labeled an unverified hypothesis.
- **Deletion recommendations follow extract-then-delete:** a doc with live obligations (pre-registered gates, parked triggers, inbound references from ROADMAP/skills/vault notes) gets its obligations extracted to a named target before its delete recommendation is valid (notes 62, 64).
- **Cleanup posture: lean aggressive** (note 60) — consolidate still-relevant history into ONE concise record, recommend deleting the rest; git history + chunk ingest are the safety net.
- **Scope:** all tracked `*.md` plus `docs/images/`. `dev/eval/**` fixtures/sandbox-texts/vault-seeds are classified wholesale as FIXTURE (test data, not docs); `dev/eval` results/ledger docs get individual dispositions. Vault notes and `~/.claude` files are reference sources, not review subjects.
- **Decision points, not silent calls:** DESIGN-HISTORY keep-vs-extract, dev/eval ledger placement, baseline-test-doc placement, and features-doc charter are presented as options with a recommendation each (deliver the full set, don't prune).

---

### Task 1: Freeze ground truth — inventory + inbound-reference graph

**Files:**
- Create: `<scratchpad>/inventory.md` (per-file: path, last-commit date, lines)
- Create: `<scratchpad>/refs.md` (per doc: inbound references from repo, skills, deployed skills/guidance, vault notes, auto-memory)

**Interfaces:**
- Produces: `inventory.md` — one line per tracked `*.md`: `<last-commit-date>  <lines>  <path>`. `refs.md` — sections per referencing corpus, each line `<referencing-file>: <quoted match> -> <doc path>`.

- [ ] **Step 1: Write the inventory**

```bash
cd /Users/joe/repos/personal/engram
git ls-files '*.md' | while read f; do d=$(git log -1 --format=%as -- "$f"); l=$(wc -l < "$f" | tr -d ' '); echo "$d  $(printf %5s $l)  $f"; done | sort -r > <scratchpad>/inventory.md
```

- [ ] **Step 2: Build the inbound-reference graph**

Grep each corpus for citations of repo doc paths (`docs/`, `README`, `questions.md`, `dev/eval/*.md`):

```bash
cd /Users/joe/repos/personal/engram
grep -rn --include='*.md' --include='*.go' --include='*.py' -E 'docs/(design|research|architecture|superpowers|GLOSSARY|ROADMAP|DESIGN-HISTORY|triage|validation-harness|images)[^ )`"]*' . | grep -v '^\.git' > <scratchpad>/refs-repo.txt
grep -rn -E 'docs/[a-zA-Z0-9_/.-]+\.md|engram/docs/' ~/.local/share/engram/vault/*.md > <scratchpad>/refs-vault.txt
grep -rn -E 'docs/[a-zA-Z0-9_/.-]+\.md' ~/.claude/projects/-Users-joe-repos-personal-engram/memory/*.md ~/.claude/skills/{recall,learn,write-memory,please,route}/SKILL.md ~/.claude/engram/recall.md > <scratchpad>/refs-deployed.txt
```

Merge into `refs.md` grouped by target doc.

- [ ] **Step 3: Verify the graph reproduces 3 known references (RED-analogue probe — must pass before the graph is trusted)**

Expected hits, all three required:
1. `docs/ROADMAP.md` → `docs/design/2026-07-03-qa-memory-proposals.md`
2. vault note `68.…emergent-synthesis….md` (or its successors) → `docs/research/2026-06-22-emergent-synthesis-case.md`
3. `skills/learn/SKILL.md` → `docs/design/2026-07-03-qa-memory-proposals.md`

If any is missing, the grep patterns are too narrow — widen and re-run before proceeding.

- [ ] **Step 4: Commit nothing** (scratchpad artifacts are temp; deleted at step 6 of the /please cycle).

---

### Task 2: Multi-angle review fan-out (Workflow)

**Files:**
- Create: `<scratchpad>/angle-outputs/A1..A6.md` (structured agent outputs)
- Create: `<scratchpad>/verified-findings.md` (post-verification correctness findings)

**Interfaces:**
- Consumes: `inventory.md`, `refs.md` (Task 1).
- Produces: per-angle structured outputs (schemas below) + a verified-findings list where every A2/A3 finding carries `verdict: CONFIRMED|REFUTED|UNVERIFIED-HYPOTHESIS`.

Six angles, one fresh agent each (Workflow `agent()` calls; every prompt opens with: *"First run `/recall glance` with phrases drawn from your angle. Then…"* — route rule). Each receives: the ask verbatim, the Global Constraints block above, the inventory, the refs map, its angle charge, and its output schema.

- [ ] **Step 1: Launch the Workflow with these six angle charges**

**A1 — Liveness/disposition (model: sonnet).** For EVERY inventory file assign exactly one: `LIVE` / `HIST-DEAD` (historical, no live obligations) / `HIST-OBLIGATED` (historical BUT carries live obligations — list each obligation verbatim + its proposed extraction target) / `FIXTURE` (test data) / `SOURCE` (deployable skill/command/guidance source). Evidence per non-obvious call: last-commit date + inbound refs from `refs.md` + content signals (CORRECTION markers, "SHIPPED", pre-registered gates, parked triggers). Output schema: markdown table `path | disposition | evidence | obligations→target (if HIST-OBLIGATED)`.

**A2 — Correctness vs code (model: sonnet, effort high).** Over the LIVE set only (README.md, CLAUDE.md, docs/GLOSSARY.md, docs/ROADMAP.md "Where we are"/constraint claims, docs/architecture/{adr,c1,c2,c3}.md, .claude/rules/go.md, .claude/skills/engram-go-conventions.md): verify every checkable claim against the code, traced from real entry points (`cmd/engram/main.go` → `internal/cli`) — flags, subcommands, defaults, paths, package lists, pipeline descriptions, defect annotations (⚠ KNOWN lines — is each still true?). Known seed finding to confirm and extend, not just echo: README.md:158 documents `targ build`, which does not exist. Output schema: per finding — `doc file:line | verbatim quote | what the code actually shows (file:line) | severity (misleads-design / minor-drift) | proposed fix text`.

**A3 — Correctness vs skills (model: sonnet, effort high).** Verify every skill-behavior claim in the LIVE set (README skills table, GLOSSARY skill/mode/step entries, c1-system-context flows for recall/learn/please/update, ROADMAP claims about what skills do) against `skills/{recall,learn,write-memory,please,route}/SKILL.md`, `commands/*.md`, `guidance/recall.md` — the REPO sources, cross-checked against deployed `~/.claude/skills/*` for drift between repo source and deployed copies. Same output schema as A2.

**A4 — SRP/overlap + target tree (model: sonnet, effort high).** Map responsibilities → current locations (decisions: adr.md + ROADMAP + GLOSSARY D5′ + design docs; features/behavior: README + ROADMAP Shipped + GLOSSARY; status/results: ROADMAP + EXPERIMENT-LOG + scattered results docs; history: DESIGN-HISTORY + plans + design docs). Identify every responsibility living in ≥2 places. Propose the target doc set with a one-line charter per doc (must include: one glossary, one roadmap [future-only], one features doc, one ADR/standards doc, docs/README.md index explaining the breakdown; may include: one history doc — present keep-vs-extract as a decision point) and the migration mapping (current doc → target section). Output schema: overlap table + target tree + charters + migration map + decision points.

**A5 — Diagram audit + proposals (model: sonnet).** (1) Verify existing mermaid (c1/c2/c3 + c1 Key-flows sequence diagrams) currency against code+skills — flag stale nodes/edges/annotations with the verbatim diagram line. (2) Propose diagrams for key features lacking them — candidates to evaluate, minimum: recall query pipeline (channels/floor/clustering/nomination), learn capture kinds incl. reversals, please 7-step + gates, vocab lifecycle (refit triggers), QA capture path, ingest/chunking, update/deploy flow. Per proposal: diagram type (C4 level / sequence / flowchart), target doc, and the ~5-15 node outline (not full mermaid). (3) Note the c4-skill path mismatch (skill targets `architecture/c4/`; repo uses `docs/architecture/`) and propose the reconciliation. Output schema: staleness findings (A2 schema) + proposal list.

**A6 — Completeness/concision (model: sonnet).** Gaps: shipped behavior with no live-doc home (check: vocab lifecycle, QA round-1, glance/deep dial, tag nomination, supersession ride-along, concurrency/flock model, non-persistent-workspace sweep config). Bloat: live docs explaining removed mechanisms or duplicating GLOSSARY. Disposition for each `docs/triage.md` open item (4, 9, 11, 13, 14, 15). Output schema: gap list (what, evidence it shipped, proposed home) + bloat list (verbatim anchor + cut proposal) + triage-item table.

- [ ] **Step 2: Adversarial verification stage (same Workflow, pipelined)**

Every A2/A3/A5-staleness finding → one fresh verifier agent (model: sonnet) prompted to REFUTE it against the code/skills (`Read the cited file:line yourself; try to prove the finding wrong; verdict CONFIRMED/REFUTED + one-line reason`). REFUTED findings are dropped (logged). A1's `HIST-OBLIGATED` rows (the costly-if-wrong class) each get a verifier confirming every claimed obligation exists at the quoted location AND no obligation was missed (checks the doc against refs.md). A1 `HIST-DEAD` rows are sampled: 5 random rows re-checked for missed inbound refs.

- [ ] **Step 3: Acceptance check on fan-out output (pre-registered)**

All must hold before Task 3:
1. Every inventory file has exactly one A1 disposition.
2. Zero A2/A3 findings lacking a verdict.
3. Every HIST-OBLIGATED row names extraction target(s) for every obligation.
4. A4 target tree covers all seven ask elements (a)–(g).
If any fails, re-dispatch the gap (not the whole angle).

---

### Task 3: Synthesize the suggestions report

**Files:**
- Create: `docs/design/2026-07-04-docs-restructure-suggestions.md`

**Interfaces:**
- Consumes: all Task-2 outputs.
- Produces: the report, structured exactly as: **1. Summary + target doc tree** (with charters, incl. docs/README.md index design) · **2. Per-file disposition table** (every tracked .md; disposition + one-line rationale + extraction targets) · **3. Extraction list** (obligation → verbatim source quote → target doc/section) · **4. Correctness fixes** (verified findings only; verbatim anchor + proposed replacement text; unverified hypotheses in a separate clearly-labeled subsection) · **5. Mermaid diagram proposals** (type, target doc, node outline; c4-skill reconciliation) · **6. Reference-scrub checklist** (every inbound ref to a delete-recommended doc, incl. vault-note citations that will dangle) · **7. Decision points for Joe** (each: options + recommendation + what changes downstream) · **8. Suggested execution order** for the follow-up cycle.

- [ ] **Step 1: Write the report** (orchestrator, from verified outputs only — numbers copied from agent evidence pointers, never from memory; each section header maps to ask elements it covers).

- [ ] **Step 2: Run the pre-registered report acceptance checks**

1. Ask-coverage: every element (a)–(g) has ≥1 section addressing it — list the mapping at the report's end.
2. Disposition completeness: `git ls-files '*.md' | wc -l` equals the disposition-table row count (± FIXTURE wholesale groups, which list their file-count).
3. Anchor rule: every proposed edit quotes its target verbatim with file:line.
4. Provenance rule: every number/claim has an evidence pointer; anything unverified is labeled.
If a check fails, fix the report before gating.

- [ ] **Step 3: Gate B (design-fit, sonnet)** — fresh reviewer: does the report read as one coherent artifact (DRY, no angle-silo seams, decision points genuinely open rather than pre-decided)?

---

### Task 4: Gates and commit

**Files:**
- Modify: none beyond the report (Gate C may produce edits to it).

- [ ] **Step 1: Gate C** — two fresh reviewers over the report: relevance (haiku) — does each suggestion actually need to exist; is any OTHER doc now stale because of this report (it's review-only, so expected answer: no, but verify); clarity/cohesion (haiku).
- [ ] **Step 2: Resolve all findings** (fix or rebut to ACK; escalate after ~2 rounds).
- [ ] **Step 3: Commit** the report + this plan via the commit skill; message passes Gate D (clarity/standards, haiku). Expected shape: `docs(review): docs-restructure suggestions — dispositions, SRP tree, diagram proposals`.
- [ ] **Step 4: Present the report to Joe** — lead with the target tree + headline dispositions table (labeled columns), decision points explicit.

## Self-Review (run at plan-write time)

- Spec coverage: (a) liveness → A1 + report §2/§3/§6; (b) SRP → A4 + §1; (c) folder/README → A4 charter incl. docs/README.md + §1; (d) diagrams → A5 + §5; (e) correctness code+skills → A2 + A3 + §4; (f) complete/concise → A6 + §4/§5; (g) multi-angle + concrete suggestions → the whole pipeline + §7/§8. ✓
- Placeholder scan: agent charges carry full output schemas; no TBDs. ✓
- Consistency: disposition vocabulary (LIVE/HIST-DEAD/HIST-OBLIGATED/FIXTURE/SOURCE) used identically in A1, Task-2 verification, and report §2. ✓
