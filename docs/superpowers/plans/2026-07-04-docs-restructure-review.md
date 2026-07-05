# Docs Restructure Review Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a gate-passed suggestions report (`docs/design/2026-07-04-docs-restructure-suggestions.md`) giving Joe concrete, evidence-anchored recommendations to make the repo's documentation live-only, SRP-shaped, diagram-supported, and verifiably correct — review only; no restructuring executed this cycle.

**Architecture:** Hybrid orchestration. Mechanical ground truth (file inventory + inbound-reference graph) is built inline first, then a Workflow fans out six per-angle review agents (each recalls first, per `route`) over that shared ground truth, with adversarial verification of correctness findings. The orchestrator synthesizes the verified outputs into one report with per-file dispositions and decision points.

**Tech Stack:** Claude Code Workflow tool (fan-out), engram recall (per-agent), git/grep (ground truth), mermaid (diagram proposals only — no diagrams built this cycle).

**Paths:** `$SCRATCH` below = `/private/tmp/claude-501/-Users-joe-repos-personal-engram/8399f498-0652-46e7-b4a3-792b7a4cc933/scratchpad` (this session's scratchpad). Repo root = `/Users/joe/repos/personal/engram`.

## Global Constraints

Vault-note numbers below (e.g. "note 60") are provenance for Joe — the rule text here is complete; no executor needs to read the vault to comply.

- **Review only.** No doc is deleted, moved, merged, or rewritten this cycle. The report proposes; Joe disposes.
- **The ask (verbatim coverage contract):** (a) docs live-only — historical committed then deleted; (b) SRP — one glossary, one roadmap, one features doc, one ADR/standards doc, etc.; (c) sane folder + README explaining the breakdown; one obvious place to go; (d) mermaid diagrams (C4, sequence, flow) for key features; (e) remaining docs obviously correct vs existing **code** AND **skills** (future-work docs exempt from "describes current code"); (f) complete and concise; (g) thorough multi-angle review → concrete suggestions.
- **Element (a) is SETTLED, not a decision point.** Historical docs are recommended for extract-then-delete — including `DESIGN-HISTORY.md`. The ask's SRP list has no history-doc slot; git is the history register. If the review finds a specific historical doc whose non-obligation content seems worth keeping, the report flags it explicitly as *"this contradicts your stated instruction — here is why an exception may be warranted"*, never as a neutral keep-vs-delete menu (notes 87/61: a prior session's heuristic does not soften this ask's explicit instruction).
- **Every proposed edit in the report pins a verbatim anchor** — quoted target text + `file:line` (note 170). Relational anchors are findings-in-waiting.
- **Every measured/derived number carries an evidence pointer + vintage** (notes 156, 162); every correctness finding is verified-against-code or explicitly labeled an unverified hypothesis.
- **Deletion recommendations follow extract-then-delete:** a doc with live obligations (pre-registered gates, parked triggers, inbound references from ROADMAP/skills/vault notes) gets its obligations extracted to a named target before its delete recommendation is valid (notes 62, 64).
- **Cleanup posture: lean aggressive** (note 60) — recommend deleting historical docs decisively once obligations are extracted; git history + chunk ingest are the safety net.
- **Scope:** all tracked `*.md` plus `docs/images/`. `dev/eval/**` fixtures/sandbox-texts/vault-seeds are classified wholesale as FIXTURE (test data, not docs); `dev/eval` results/ledger docs get individual dispositions. Vault notes and `~/.claude` files are **reference sources only, never review subjects**: agents read them to corroborate repo-doc claims; repo-vs-deployed drift is NOT a report finding (at most a one-line out-of-band note to Joe).
- **Decision points, not silent calls:** dev/eval ledger placement, baseline-test-doc placement, features-doc charter, and *where* extracted historical content lands (e.g. adr.md vs ROADMAP) are presented as options with a recommendation each. (Whether historical docs are deleted is settled — see element (a) above.)
- **Recall mode for fan-out agents:** every dispatched agent's first action is `/recall` (route rule); the **glance** mode is this plan's own call — the angle agents are read-side evidence gatherers, the weighty dispositions are made by the orchestrator and Joe, and crystallization happens at the cycle's closing `/learn` — not a route-skill mandate.

---

### Task 1: Freeze ground truth — inventory + inbound-reference graph

**Files:**
- Create: `$SCRATCH/inventory.md` (per-file: path, last-commit date, lines)
- Create: `$SCRATCH/refs.md` (inbound references, grouped by target doc)

**Interfaces:**
- Produces: `inventory.md` — one line per tracked `*.md`: `<last-commit-date>  <lines>  <path>`. `refs.md` — one `## <target doc path>` section per referenced doc; under it one line per inbound ref: `[repo|vault|deployed] <referencing-file>:<line>: <quoted match>`.

- [ ] **Step 1: Write the inventory**

```bash
SCRATCH=/private/tmp/claude-501/-Users-joe-repos-personal-engram/8399f498-0652-46e7-b4a3-792b7a4cc933/scratchpad
cd /Users/joe/repos/personal/engram
git ls-files '*.md' | while read f; do d=$(git log -1 --format=%as -- "$f"); l=$(wc -l < "$f" | tr -d ' '); echo "$d  $(printf %5s $l)  $f"; done | sort -r > $SCRATCH/inventory.md
```

- [ ] **Step 2: Build the inbound-reference graph**

Grep each corpus for citations of repo doc paths — including bare `README`, `CLAUDE.md`, `questions.md`, and `dev/eval` doc mentions, not only `docs/...` paths:

```bash
SCRATCH=/private/tmp/claude-501/-Users-joe-repos-personal-engram/8399f498-0652-46e7-b4a3-792b7a4cc933/scratchpad
cd /Users/joe/repos/personal/engram
grep -rn --include='*.md' --include='*.go' --include='*.py' -E '(docs/(design|research|architecture|superpowers|GLOSSARY|ROADMAP|DESIGN-HISTORY|triage|validation-harness|images)[^ )`"]*|README(\.md)?|CLAUDE\.md|questions\.md|dev/eval/[a-zA-Z0-9_/.-]+\.md)' . | grep -v '^\.git' > $SCRATCH/refs-repo.txt
grep -rn -E '(docs/[a-zA-Z0-9_/.-]+\.md|engram/docs/|README\.md|CLAUDE\.md|dev/eval/[a-zA-Z0-9_/.-]+\.md)' ~/.local/share/engram/vault/*.md > $SCRATCH/refs-vault.txt
grep -rn -E '(docs/[a-zA-Z0-9_/.-]+\.md|README\.md|CLAUDE\.md|dev/eval/[a-zA-Z0-9_/.-]+\.md)' ~/.claude/projects/-Users-joe-repos-personal-engram/memory/*.md ~/.claude/skills/recall/SKILL.md ~/.claude/skills/learn/SKILL.md ~/.claude/skills/write-memory/SKILL.md ~/.claude/skills/please/SKILL.md ~/.claude/skills/route/SKILL.md ~/.claude/engram/recall.md > $SCRATCH/refs-deployed.txt
```

Merge into `$SCRATCH/refs.md` **grouped by target doc** (one `##` section per target), each line prefixed with its corpus tag (`[repo]` / `[vault]` / `[deployed]`). Self-references (a doc citing itself) are dropped at merge.

- [ ] **Step 3: Verify the graph reproduces 4 known references (RED-analogue probe — must pass before the graph is trusted)**

Expected hits, all four required:
1. `docs/ROADMAP.md` → `docs/design/2026-07-03-qa-memory-proposals.md`
2. vault note `68.…aggregation-is-not-emergent-synthesis.md` → `docs/research/2026-06-22-emergent-synthesis-case.md`
3. `skills/learn/SKILL.md` → `docs/design/2026-07-03-qa-memory-proposals.md`
4. `docs/ROADMAP.md` → a bare `CLAUDE.md` mention (the "global CLAUDE.md guidance" Track-A text) — exercises the non-`docs/` pattern branch

If any is missing, the grep patterns are too narrow — widen and re-run before proceeding.

- [ ] **Step 4: Commit nothing** (scratchpad artifacts are temp; deleted at step 6 of the /please cycle).

---

### Task 2: Multi-angle review fan-out (Workflow)

**Files:**
- Create: `$SCRATCH/angle-outputs/A1..A6.md` (structured agent outputs)
- Create: `$SCRATCH/verified-findings.md` (post-verification correctness findings)

**Interfaces:**
- Consumes: `inventory.md`, `refs.md` (Task 1).
- Produces: per-angle structured outputs (schemas below) + a verified-findings list where every A2/A3/A5-staleness finding carries `verdict: CONFIRMED|REFUTED|UNVERIFIED-HYPOTHESIS`.

Six angles, one fresh agent each (Workflow `agent()` calls; every prompt opens with: *"First run `/recall glance` with phrases drawn from your angle. Then…"* — recall-first per route; glance per the Global Constraints rationale). Each receives: the ask verbatim, the Global Constraints block above, the inventory, the refs map, its angle charge, and its output schema.

**Shared finding-table schema (A2, A3, and A5-staleness findings)** — markdown table with exactly this header:

```
| doc file:line | verbatim quote | what the source-of-truth shows (file:line) | severity | proposed fix text |
|---|---|---|---|---|
```

Severity vocabulary: `misleads-design` (an agent/reader acting on it would build against dead paths) / `minor-drift` (wrong but low consequence). Example row: `| README.md:158 | targ build — build the engram binary | no build target: dev/targs.go registers test/check/lint only | misleads-design | replace line with: go install ./cmd/engram (there is no targ build target) |`

- [ ] **Step 1: Launch the Workflow with these six angle charges**

**A1 — Liveness/disposition (model: sonnet).** For EVERY inventory file assign exactly one: `LIVE` / `HIST-DEAD` (historical, no live obligations) / `HIST-OBLIGATED` (historical BUT carries live obligations — list each obligation verbatim + its proposed extraction target) / `FIXTURE` (test data) / `SOURCE` (deployable skill/command/guidance source). Evidence per non-obvious call: last-commit date + inbound refs from `refs.md` + content signals. Content signals are judged by READING the doc (word-match misses subtle cases — note 112); indicative markers include `CORRECTION` sections, `SHIPPED`/`PARKED`/`DEFERRED` status lines, "pre-registered" gates or bars, "trigger"/"fires if" clauses, and "Round-N gate" language — treat these as hints to read around, not as the detection method. Output schema: markdown table `| path | disposition | evidence | obligations → extraction target (HIST-OBLIGATED only) |`. FIXTURE groups may be one row per directory subtree with a file count (e.g. `dev/eval/testdata/** | FIXTURE | test data | — (61 files)`).

**A2 — Correctness vs code (model: sonnet, effort high).** Over the LIVE set only (README.md, CLAUDE.md, docs/GLOSSARY.md, docs/ROADMAP.md "Where we are"/constraint claims, docs/architecture/{adr,c1-system-context,c2-containers,c3-components}.md, .claude/rules/go.md, .claude/skills/engram-go-conventions.md): verify every checkable claim against the code. "Verify" = read the call chain from `cmd/engram/main.go` through `internal/cli/*` (flag definitions, subcommand registration, defaults, hardcoded paths, package lists); running read-only commands (`engram --help`, `targ` with no args) is allowed, nothing that writes. For each `⚠ KNOWN` defect annotation: read the cited code and judge whether the defect is still true. Known seed finding to confirm and extend, not just echo: README.md:158 documents `targ build`, which does not exist. Output: the shared finding-table schema.

**A3 — Correctness vs skills (model: sonnet, effort high).** Verify every skill-behavior claim in the LIVE set (README skills table, GLOSSARY skill/mode/step entries, c1-system-context flows for recall/learn/please/update, ROADMAP claims about what skills do) against the REPO sources: `skills/{recall,learn,write-memory,please,route}/SKILL.md`, `commands/*.md`, `guidance/recall.md`. Deployed copies under `~/.claude` may be read ONLY to corroborate a repo-doc claim; do not emit repo-vs-deployed drift findings (scope rule above) — if drift is noticed, record at most one out-of-band line at the end of the output, outside the finding table. Output: the shared finding-table schema.

**A4 — SRP/overlap + target tree (model: sonnet, effort high).** Map responsibilities → current locations (decisions: adr.md + ROADMAP + GLOSSARY D5′ + design docs; features/behavior: README + ROADMAP Shipped + GLOSSARY; status/results: ROADMAP + EXPERIMENT-LOG + scattered results docs; history: DESIGN-HISTORY + plans + design docs). Identify every responsibility living in ≥2 places. Propose the target doc set honoring element (a) as settled (no history doc; DESIGN-HISTORY content is extracted-then-deleted) and including: one glossary, one roadmap (future-only), one features doc, one ADR/standards doc, docs/README.md index explaining the breakdown. Output schema, four parts:
1. **Overlap table:** `| responsibility | current docs (comma-separated) | recommended single home |`
2. **Target tree:** nested markdown outline, max 2 levels, one `[charter: one line]` per doc.
3. **Migration map:** `| current doc | current section | target doc | target section |`
4. **Decision points:** bulleted, each exactly `Option A: … | Option B: … | Recommendation: … because …` — restricted to the decision points the Global Constraints allow.

**A5 — Diagram audit + proposals (model: sonnet).** (1) Verify existing mermaid (c1/c2/c3 + c1 Key-flows sequence diagrams) currency against code+skills — stale nodes/edges/annotations go in the shared finding-table schema (quote the diagram line verbatim). (2) Propose diagrams for key features lacking them — candidates to evaluate, minimum: recall query pipeline (channels/floor/clustering/nomination), learn capture kinds incl. reversals, please 7-step + gates, vocab lifecycle (refit triggers), QA capture path, ingest/chunking, update/deploy flow. Per proposal: diagram type (C4 level / sequence / flowchart), target doc, and a 5–15 bullet node outline (plain bullets naming the nodes/steps and key edges — not mermaid syntax). (3) Note the c4-skill path mismatch (deployed c4 skill targets `architecture/c4/`; repo diagrams live in `docs/architecture/`) and propose the reconciliation as a decision option pair.

**A6 — Completeness/concision (model: sonnet).** Gaps: for each shipped behavior (vocab lifecycle, QA round-1, glance/deep dial, tag nomination, supersession ride-along, concurrency/flock model, non-persistent-workspace sweep config — and any others found in ROADMAP ✅ entries), check it appears in (a) code, (b) a live doc (docs/ or README), (c) a repo skill/guidance source where applicable; shipped-in-code but absent-from-live-docs = a gap, with a proposed home. Bloat: live-doc passages explaining removed mechanisms or duplicating GLOSSARY — verbatim anchor + cut proposal. Plus a disposition for each `docs/triage.md` open item (4, 9, 11, 13, 14, 15): `| item | recommended resolution | where the fix lands |`.

- [ ] **Step 2: Adversarial verification stage (same Workflow, pipelined)**

Every A2/A3/A5-staleness finding → one fresh verifier agent (model: sonnet) prompted to REFUTE it against the code/skills (`Read the cited file:line yourself; try to prove the finding wrong; verdict CONFIRMED/REFUTED + one-line reason`). REFUTED findings are dropped (logged in `verified-findings.md`). A1's `HIST-OBLIGATED` rows (the costly-if-wrong class) each get a verifier that (1) greps the doc for each claimed obligation's quoted text verbatim — an unfindable quote demotes that obligation to UNVERIFIED-HYPOTHESIS; and (2) independently re-reads the doc hunting for obligations the row MISSED (pre-registered gates/bars, "fires if" triggers, inbound refs in `refs.md`) — missed obligations are added, not silently fixed. A1 `HIST-DEAD` rows: sample 5 via `sort -R | head -5` over the HIST-DEAD table rows; for each, re-run the Task-1 Step-2 repo grep restricted to that doc's basename and re-check `refs.md` — any surfaced inbound ref flips the row to HIST-OBLIGATED for re-review.

- [ ] **Step 3: Acceptance check on fan-out output (pre-registered)**

All must hold before Task 3:
1. Every inventory file has exactly one A1 disposition (FIXTURE subtree rows count their files; per-row files + subtree counts must sum to the inventory line count).
2. Zero A2/A3/A5-staleness findings lacking a verdict.
3. Every HIST-OBLIGATED row names extraction target(s) for every obligation.
4. A4 target tree covers ask elements (a)–(g).
If any fails, re-dispatch the gap (not the whole angle).

---

### Task 3: Synthesize the suggestions report

**Files:**
- Create: `docs/design/2026-07-04-docs-restructure-suggestions.md`

**Interfaces:**
- Consumes: all Task-2 outputs.
- Produces: the report, structured exactly as: **1. Summary + target doc tree** (with charters, incl. docs/README.md index design) · **2. Per-file disposition table** (every tracked .md; disposition + one-line rationale + extraction targets) · **3. Extraction list** (obligation → verbatim source quote → target doc/section) · **4. Correctness fixes** (verified findings only; verbatim anchor + proposed replacement text; unverified hypotheses in a separate clearly-labeled subsection) · **5. Mermaid diagram proposals** (type, target doc, node outline; c4-skill reconciliation) · **6. Reference-scrub checklist** (every inbound ref to a delete-recommended doc, incl. vault-note citations that will dangle) · **7. Decision points for Joe** (each: options + recommendation + what changes downstream; any exception-to-element-(a) argument appears here, explicitly labeled as contradicting the stated instruction) · **8. Suggested execution order** for the follow-up cycle.

- [ ] **Step 1: Write the report** from Task-2 verified outputs only — numbers copied from agent evidence pointers, never from memory. Section headers map to the ask elements they cover; the mapping table goes at the report's end.

- [ ] **Step 2: Run the pre-registered report acceptance checks**

1. Ask-coverage: every element (a)–(g) has ≥1 section addressing it — the end-of-report mapping table proves it.
2. Disposition completeness: `git ls-files '*.md' | wc -l` = (rows without a file count) + (sum of file counts on rows carrying one).
3. Anchor rule: every proposed edit quotes its target verbatim with file:line.
4. Provenance rule: every number/claim has an evidence pointer; anything unverified is labeled.
If a check fails, fix the report before gating.

- [ ] **Step 3: Gate B (design-fit, sonnet)** — fresh reviewer, three charges: (1) coherence — does the report read as one artifact, no angle-silo seams; (2) each decision point presents ≥2 named options + an explicit recommendation, and no settled ask element is reopened as an option; (3) DRY — no claim duplicated across sections.

---

### Task 4: Gates and commit

**Files:**
- Modify: none beyond the report (Gate C may produce edits to it).

- [ ] **Step 1: Gate C** — two fresh reviewers over the report: relevance (haiku) — does each suggestion actually need to exist; is any OTHER doc now stale because of this report (review-only, so expected answer: no — verify); clarity/cohesion (haiku).
- [ ] **Step 2: Resolve all findings** (fix or rebut to ACK; escalate after ~2 rounds).
- [ ] **Step 3: Commit** the report + this plan via the commit skill; message passes Gate D (clarity/standards, haiku). Expected shape: `docs(review): docs-restructure suggestions — dispositions, SRP tree, diagram proposals`.
- [ ] **Step 4: Present the report to Joe** — lead with the target tree + headline dispositions table (labeled columns), decision points explicit.

## Self-Review (run at plan-write time)

- Spec coverage: (a) liveness → A1 + report §2/§3/§6 (settled, not a decision point); (b) SRP → A4 + §1; (c) folder/README → A4 charter incl. docs/README.md + §1; (d) diagrams → A5 + §5; (e) correctness code+skills → A2 + A3 + §4; (f) complete/concise → A6 + §4/§5; (g) multi-angle + concrete suggestions → the whole pipeline + §7/§8. ✓
- Placeholder scan: `$SCRATCH` defined under **Paths**; agent charges carry full output schemas with header rows and worked examples; no TBDs. ✓
- Consistency: disposition vocabulary (LIVE/HIST-DEAD/HIST-OBLIGATED/FIXTURE/SOURCE) and the shared finding-table schema are defined once and referenced by name everywhere. ✓

## Gate A resolution log (2026-07-04)

- **ask-alignment F1 (DESIGN-HISTORY relitigated element (a)):** FIXED — element (a) declared settled in Global Constraints; DESIGN-HISTORY removed from decision points; A4 charge and report §7 updated; exception arguments must be explicitly labeled as contradicting the instruction.
- **ask-alignment F2 (A3 deployed-drift scope leak):** FIXED — deployed copies are corroboration-only; drift findings barred from the report; at most one out-of-band line.
- **ask-alignment F3 (refs grep drops bare README/CLAUDE.md):** FIXED — patterns widened in all three Step-2 commands; probe 4 added exercising the non-`docs/` branch.
- **code-alignment F1 (glance misattributed to route):** FIXED — recall-first attributed to route; glance owned as this plan's call with rationale (Global Constraints, last bullet).
- **clarity F1–F13:** FIXED — `$SCRATCH` defined; sampling method pinned (`sort -R | head -5`); HIST-OBLIGATED verification de-circularized (verbatim-grep per obligation + independent miss-hunt); FIXTURE arithmetic made checkable; refs.md grouping unified (by target doc, corpus-tagged lines); content signals reframed as read-and-judge with marker examples; A2 "verify" defined; A6 gap procedure made 3-step checkable; shared finding-table schema with header + example; A4 output formats pinned; diagram outlines defined as plain bullets; Task-3 Step-1 tightened; Gate B charges concretized. REBUTTED (partial): vault-note citations stay as provenance-only pointers — rule text is complete inline (now stated explicitly at the top of Global Constraints).
