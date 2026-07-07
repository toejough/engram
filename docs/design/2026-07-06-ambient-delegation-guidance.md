# Ambient delegation guidance (`delegate.md`)

**Status:** in-flight design. Retire on ship (graduate into FEATURES/CLAUDE.md; delete this file).
**Date:** 2026-07-06

## The gap

The delegate-everything doctrine already exists in two skills:

- **`route`** — *how* to dispatch one unit: tier selection (cheapest-first, evidence-raised),
  exact-handoff spec, per-dispatch evidence recording.
- **`please`** — the full end-to-end *procedure*: seven fixed steps + four adversarial review gates.

Both fire only when their skill triggers (`/route` on a dispatch decision, `/please` on an
end-to-end handoff). Outside those triggers — which is most of the time — nothing keeps the
"orchestrate, don't do object-level work solo" reflex live. The agent defaults to opening the file
and typing.

`recall.md` already solved the identical problem for memory recall: the `recall` skill's own
trigger under-fires at the decision moment, so an always-loaded `@import` guidance doc fires the
reflex at the cues that matter. This design does the same thing for delegation.

## What this is (and is not)

A new **ambient guidance doc**, `guidance/delegate.md`, deployed and `@import`ed exactly like
`recall.md`. It is a **posture**, not a procedure.

| | What it is | When it fires | What it does |
|---|---|---|---|
| **`delegate.md`** (this) | a posture, always loaded | the moment you're about to do object-level work *solo* | biases the default → plan · delegate · review · report; points at route/please |
| **`route`** (skill) | the *how* of one dispatch | about to dispatch a subagent | tier choice, handoff spec, evidence record |
| **`please`** (skill) | the full end-to-end *procedure* | explicit `/please` or "drive X to done" | 7 steps + 4 gates |

**Not a lighter-weight `please`.** `please` is a heavy, opt-in procedure you *enter* for a whole
ask. `delegate.md` runs no steps; it is the always-on nudge that makes you reach for route (or
please) in the first place — the same relationship `recall.md` has to the `recall` skill.

## The doctrine encoded

Four beats: **plan it → hand it to a subagent → review the result → report back.** Review is by a
fresh-context reviewer, not the builder's self-report. Report is route's dispatch-evidence table or
please's gate outcomes.

### The floor: evidence, not a guess

The escape hatch mirrors `route`'s own tier loop, one level up:

- **`route`** decides *which tier* → default cheapest; only recalled evidence raises it; every
  dispatch measured + remembered.
- **`delegate.md`** decides *whether to delegate at all* → default route; only recalled evidence
  that a task-kind is reliably sub-overhead lets you do it inline; every dispatch measured +
  remembered.

> **Default: route it, measure it, remember it.** Do a unit inline *only* when recalled memory
> shows tasks of this kind are reliably below the routing overhead — a measured track record, never
> an in-the-moment "this is quick." No such evidence → route it, even if it feels trivial; the
> measurement you record earns the inline escape next time. Don't guess it's a quick fix — **know** it.

**Cold-start consequence (stated, not hidden):** an empty vault knows no task as sub-overhead, so
everything routes — identical to `route`'s existing "cheapest for everything, even a var rename"
posture. The escape warms up fast as trivial task-kinds prove themselves in a dispatch or two.

This resolves the apparent contradiction between `route` ("no inline escape — easy work is
delegated, not skipped") and the global CLAUDE.md Quick Fix tier ("known single edit → just do
it"): the escape is **overhead-based and evidence-gated**, never difficulty-based. "It feels easy"
is never a reason to work solo.

## Deliverables

### 1. `guidance/delegate.md`

recall.md sibling: same `engram-owned` header contract (deployed by `engram update`, edited via
writing-skills TDD), ~25 lines. Deployed to `~/.claude/engram/delegate.md`, activated by
`@~/.claude/engram/delegate.md`. Shape: thesis (you're an orchestrator; four beats; under-delegating
costs more than routing overhead) → decision cues (before writing code/prose; before a multi-step
change; when a unit is too big for one pass) → review+report → the evidence floor → pointer to
route/please → closing "catches the solo-by-habit gap" line.

### 2. First-class in `engram update`

Generalize the recall.md-hardcoded guidance-import path so any guidance file is a peer:

- **Detection** (`detectGuidanceImport`, `internal/update/update.go`): return *which* engram
  guidance basenames are imported in `~/.claude/CLAUDE.md` — not a single recall.md bool. Reuse the
  existing fence-aware scan; recognize both tilde and expanded-home forms.
- **`report.GuidanceImported`** becomes "any guidance file imported" (so importing `delegate.md`
  alone triggers the auto-refresh-all — the first-class part), plus per-file import status on the
  report for messaging.
- **CLI hint** (`writeGuidanceHints`, `internal/cli/update.go`): iterate the deployed guidance
  files instead of hardcoding recall.md — per file, "refreshed" (imported) or "deployed — add
  `@~/.claude/engram/<file>` to activate it."
- **Tests**: mirror `internal/update/update_test.go` and `internal/cli/update_test.go` — mixed
  imported/not state, `delegate.md`-only import triggering refresh, hint output naming each file.
- `planGuidanceCopies` needs **no change** — it already copies every `.md` via `mdFilesIn`.

### 3. Activation, docs, validation

- **Activation**: run `engram update` to deploy `delegate.md`, then add
  `@~/.claude/engram/delegate.md` to `~/.claude/CLAUDE.md` (the one manual step — it's global,
  outside the repo).
- **Docs**: update the repo `CLAUDE.md` `guidance/` description ("recall-firing guidance" → recall
  + delegation) and any `docs/` reference to the guidance mechanism.
- **Validation**: guidance authoring runs under **writing-skills TDD**; per the established engram
  pattern the behavioral RED/GREEN is a **headless `claude -p`** eval (fictional domains, project
  CLAUDE.md the only variable) — agent defaults to solo work *without* the doc, to
  plan→delegate→review *with* it.

## Out of scope (YAGNI)

- No changes to `route`/`please` skill bodies — `delegate.md` points at them; it does not restate
  or supersede them.
- No counterfactual "would-inline-have-been-cheaper" meter — the evidence floor uses recalled
  dispatch track record, same as route's tier evidence, not a new measurement apparatus.
- No new deploy plumbing beyond generalizing the import-detection/messaging — `planGuidanceCopies`
  already handles multi-file copy.

## Execution note

The work itself follows the doctrine being encoded: plan → delegate the units to subagents →
gated review → report. Fitting.

---

## Implementation plan

> **For agentic workers:** implement task-by-task via `superpowers:subagent-driven-development`.
> Tasks 1 and 2 touch disjoint files and may run in parallel; Task 3 depends on both. Each task
> ends with an independently reviewable, committed deliverable.

**Tech stack:** pure Go (no CGO; DI everywhere), `targ` for test/lint/check, gomega + imptest +
rapid for tests, markdown guidance docs, headless `claude -p` for behavioral guidance validation.

### Global constraints (every task)

- `targ test` / `targ check-full` only — never `go test` / `go vet` directly. Use `targ check-full`
  to surface all errors at once.
- Go: named constants (no magic numbers), descriptive names, wrap errors
  `fmt.Errorf("context: %w", err)`, `t.Parallel()` on every test + subtest (no shared mutable
  state), lines < 120 chars, gomega nil-guards per `.claude/rules/go.md` (after
  `g.Expect(err).NotTo(HaveOccurred())` add `if err != nil { return }` before using values).
- Guidance docs carry the `engram-owned` header comment and are edited via `superpowers:writing-skills` TDD.
- One commit per task: conventional-commit subject + `AI-Used: [claude]` trailer.

---

### Task 1: `guidance/delegate.md` — the ambient doctrine

**Files:**
- Create: `guidance/delegate.md`
- Validation: headless `claude -p` RED/GREEN rep-pair; result recorded in `dev/eval/LEDGER.md`
  (guidance validation is LEDGER-recorded — there is no committed per-guidance test file, matching
  `recall.md`).

**Why headless, not a subagent:** a subagent inherits this session's context, so a "no-guidance"
control would still recall the doctrine verbatim (false RED). Each arm must be a fresh `claude -p`
process with the guidance `@import` the only variable, over a **fictional** domain so no engram
memory leaks in.

- [ ] **Step 1: Invoke `superpowers:writing-skills`.** Mandatory for guidance edits (repo CLAUDE.md
  + the doc's own header contract). It owns the RED → GREEN → REFACTOR loop below.

- [ ] **Step 2: RED baseline (headless, guidance absent).** Two fictional tasks, minimal project
  `CLAUDE.md`, **no** `@~/.claude/engram/delegate.md` import. Run on the **production model**
  (`claude-opus-4-8[1m]` or current equivalent — matching recall.md's validation, LEDGER row 41).
  Minimal `CLAUDE.md` = the single line `# scratch` with no imports, so the guidance `@import` is
  the only variable between arms. Prompts (verbatim):

  - **Large unit:** "Add a `--preview` flag to the `zib` CLI that lists which `.glorp` files the
    `transmute` command would rewrite, without rewriting them — touching the glorp parser, the
    transmuter, and the CLI wiring, with tests. Go ahead."
  - **Small-looking unit** (tests the overhead-vs-difficulty boundary — Gate-A finding): "Rename the
    `frob` field to `frobnicate` across the two files `widget.zib` and `gadget.zib`. Go ahead."

  Run 5 reps per prompt: `claude -p "<prompt>" --append-system-prompt "<minimal CLAUDE.md>"` (fresh
  cwd, no delegate import). **Scoring rubric (per rep):** score **delegate** iff the agent's first
  substantive act is one of — (a) writing a plan before touching code, (b) invoking `/route` or
  `/please`, or (c) explicitly naming/dispatching a subagent; score **solo** if it opens a file and
  begins editing/writing code without any of (a)–(c). Expected RED: ~0/5 delegate on **both**
  prompts (solo default — the small-looking unit especially).

- [ ] **Step 3: Write `guidance/delegate.md`.** Exact content:

  ```markdown
  <!-- engram-owned: delegation-firing guidance. Deployed by 'engram update --with-guidance' to ~/.claude/engram/delegate.md; activate via '@~/.claude/engram/delegate.md' in CLAUDE.md. Edit via writing-skills TDD. -->

  ## Delegate object-level work — plan it, route it, review it, report it

  You are an orchestrator. The default shape for **any** unit of work is **plan it → hand it to a
  subagent → review what returns (fresh context, never the builder's own "done") → report the
  outcome** (route's evidence table, please's gate verdicts). You don't write the code, run the
  build, or judge the artifact yourself — doing it solo out of habit costs more than the routing
  overhead it saves.

  Fire at the moments you'd otherwise just-do-it-yourself:

  - **Before you open a file to write code or prose** — draft the plan and route the unit, don't start typing.
  - **Before a multi-step change** — decompose into units, hand each to a subagent (`route` sets the tier, handoff, and evidence).
  - **When a unit is too big for one focused pass** — decompose first, then dispatch the pieces.

  **The floor is evidence, not a guess.** Default to route-measure-remember; go inline **only** when
  recalled memory shows this kind of task runs reliably below the routing overhead — a measured track
  record. "It's easy" and "the overhead would exceed the work" are the same forbidden forecast; only
  the record earns the escape. No record → route it, even if it feels trivial. Don't guess it's a
  quick fix — **know** it.

  For one dispatch's *how* use `route`; for a full end-to-end ask, `/please`. This fires even when
  neither has triggered — it catches the solo-by-habit gap.
  ```

- [ ] **Step 4: GREEN (headless, guidance present).** Same 5 reps **per prompt**, now with
  `@~/.claude/engram/delegate.md` imported (place the file, add the import line to the arm's
  CLAUDE.md), same model and rubric. Expected GREEN: ≥4/5 delegate on the large unit **and** the
  small-looking unit — the small unit is the real test that the floor routes despite the "it's just
  a rename" temptation. If either is < 4/5, REFACTOR the wording (writing-skills) and re-run — do
  not ship a doc that doesn't flip the reflex at the small-edit boundary.

- [ ] **Step 5: Record + commit.** Append a row to `dev/eval/LEDGER.md` in the existing table format
  (`claim | verdict | figure | superseded-by | raw data`), e.g.:

  `| **delegate.md flips the delegation reflex** | proven | RED 0/5 → GREEN ≥4/5 on both large + small-edit prompts (headless claude -p, fresh process, fictional domain, n=5/arm, opus-4.8[1m]) (2026-07-07) | — | scores in commit body |`

  Commit: `feat(guidance): delegation-firing guidance doc (delegate.md)`.

---

### Task 2: first-class multi-guidance in `engram update`

Generalize the recall.md-hardcoded import path so every guidance file is a peer. `planGuidanceCopies`
already copies all `.md` — no change there.

**Files:**
- Modify: `internal/update/update.go` — replace `detectGuidanceImport` with `detectGuidanceImports`
  + `guidanceImportBase` helper; add `Report.GuidanceImports`; rewire `Run`.
- Modify: `internal/cli/update.go` — rewrite `writeGuidanceHints` to iterate deployed files; add
  `claudeGuidanceFiles` helper.
- Test: `internal/update/update_test.go`, `internal/cli/update_test.go`.

**Interfaces produced:**
- `detectGuidanceImports(claudeMDPath, home string, fs Filesystem) map[string]bool` — set of
  imported engram-guidance basenames.
- `Report.GuidanceImports map[string]bool` — same set on the report; `Report.GuidanceImported bool`
  stays, redefined as `len(GuidanceImports) > 0` (any file imported).

**Test doubles (existing — reuse, do NOT redefine):** `newMemFS()` / `memFS` (fields `.files`,
`.dirs`, `.written`) are in `internal/update/update_test.go`; `fakeCmd` / `fakeEnv` are in
`internal/update/runner_test.go`. Every test below reuses these.

- [ ] **Step 1: RED — detection returns a per-file set.** Add to `internal/update/update_test.go`:

  ```go
  func TestGuidanceImportDetection_PerFileSet(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	const home = "/home/joe"

  	fileSystem := newMemFS()
  	fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte(
  		"# joe\n\n@~/.claude/engram/recall.md\n@~/.claude/engram/delegate.md\n",
  	)
  	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
  	fileSystem.dirs["/repo/skills"] = true
  	fileSystem.dirs[home+"/.claude"] = true

  	updater := &update.Updater{
  		FS:  fileSystem,
  		Cmd: &fakeCmd{},
  		Env: &fakeEnv{home: home, cwd: "/repo"},
  	}

  	report, err := updater.Run(context.Background(), update.Options{})
  	g.Expect(err).NotTo(HaveOccurred())

  	if err != nil {
  		return
  	}

  	g.Expect(report.GuidanceImported).To(BeTrue())
  	g.Expect(report.GuidanceImports).To(HaveKeyWithValue("recall.md", true))
  	g.Expect(report.GuidanceImports).To(HaveKeyWithValue("delegate.md", true))
  }
  ```

- [ ] **Step 2: RED — `delegate.md`-only import triggers refresh-all.** Add:

  ```go
  func TestRun_PlainUpdate_DelegateOnlyImport_RefreshesAll(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	const home = "/home/joe"

  	fileSystem := newMemFS()
  	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
  	fileSystem.dirs["/repo/skills"] = true
  	fileSystem.dirs[home+"/.claude"] = true
  	fileSystem.files["/repo/guidance/recall.md"] = []byte("recall guidance")
  	fileSystem.files["/repo/guidance/delegate.md"] = []byte("delegate guidance")
  	// Only delegate.md is imported — recall.md is not.
  	fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte("# joe\n\n@~/.claude/engram/delegate.md\n")

  	updater := &update.Updater{
  		FS:  fileSystem,
  		Cmd: &fakeCmd{},
  		Env: &fakeEnv{home: home, cwd: "/repo"},
  	}

  	report, err := updater.Run(context.Background(), update.Options{})
  	g.Expect(err).NotTo(HaveOccurred())

  	if err != nil {
  		return
  	}

  	g.Expect(report.GuidanceImported).To(BeTrue())
  	g.Expect(report.Harnesses[0].GuidanceFiles).To(ConsistOf("recall.md", "delegate.md"))
  	g.Expect(fileSystem.written[home+"/.claude/engram/delegate.md"]).NotTo(BeNil())
  	g.Expect(fileSystem.written[home+"/.claude/engram/recall.md"]).NotTo(BeNil())
  }
  ```

- [ ] **Step 2b: RED — `guidanceImportBase` exclusion branches.** Add three rows to the EXISTING
  `TestGuidanceImportDetection` table (each asserts `GuidanceImported` stays **false**, covering the
  new reject branches so `targ check-full` coverage passes): `content: "@~/.claude/engram/sub/recall.md\n"`
  (nested path), `content: "@~/.claude/engram/recall.txt\n"` (non-`.md`), `content: "@~/.claude/engram/\n"`
  (bare prefix, no basename) — all `wantBool: false`.

- [ ] **Step 3: Run RED, verify failure.** `targ test` → the new per-file / refresh tests FAIL
  (`GuidanceImports` undefined; refresh not triggered by delegate-only import). The three exclusion
  rows pass only once Step 4 adds `guidanceImportBase`.

- [ ] **Step 4: GREEN — generalize detection in `internal/update/update.go`.** Add the field to
  `Report` (next to `GuidanceImported`):

  ```go
  	GuidanceImported bool            // true when ~/.claude/CLAUDE.md imports ANY engram guidance file
  	GuidanceImports  map[string]bool // set of imported engram-guidance basenames (for per-file hints)
  ```

  Rewire `Run` (replace the single `report.GuidanceImported = detectGuidanceImport(...)` line):

  ```go
  	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")
  	report.GuidanceImports = detectGuidanceImports(claudeMDPath, home, u.FS)
  	report.GuidanceImported = len(report.GuidanceImports) > 0
  ```

  Replace `detectGuidanceImport` with:

  ```go
  // detectGuidanceImports scans the Claude Code CLAUDE.md at claudeMDPath for
  // active @import lines pointing at engram guidance files under
  // ~/.claude/engram/, returning the set of imported guidance basenames. Both
  // the tilde form (@~/.claude/engram/foo.md) and the expanded-home form
  // (@<home>/.claude/engram/foo.md) are recognized. Lines inside fenced code
  // blocks are ignored. A missing CLAUDE.md yields an empty set, no error.
  func detectGuidanceImports(claudeMDPath, home string, fileSystem Filesystem) map[string]bool {
  	imported := map[string]bool{}

  	data, readErr := fileSystem.ReadFile(claudeMDPath)
  	if readErr != nil {
  		return imported
  	}

  	tildePrefix := "@~/.claude/engram/"
  	expandedPrefix := "@" + filepath.Join(home, ".claude", "engram") + string(filepath.Separator)

  	inFence := false

  	for line := range strings.SplitSeq(string(data), "\n") {
  		trimmed := strings.TrimSpace(line)
  		if strings.HasPrefix(trimmed, "```") {
  			inFence = !inFence

  			continue
  		}

  		if inFence {
  			continue
  		}

  		if base, ok := guidanceImportBase(trimmed, tildePrefix, expandedPrefix); ok {
  			imported[base] = true
  		}
  	}

  	return imported
  }

  // guidanceImportBase returns the guidance basename imported by an exact
  // @import line (either prefix form) and whether the line is such an import.
  // The remainder after the prefix must be a single .md basename — no nested
  // path segment, no trailing content.
  func guidanceImportBase(trimmed, tildePrefix, expandedPrefix string) (string, bool) {
  	for _, prefix := range []string{tildePrefix, expandedPrefix} {
  		rest, ok := strings.CutPrefix(trimmed, prefix)
  		if !ok {
  			continue
  		}

  		if rest == "" || strings.Contains(rest, "/") || !strings.HasSuffix(rest, ".md") {
  			continue
  		}

  		return rest, true
  	}

  	return "", false
  }
  ```

- [ ] **Step 5: GREEN — per-file hints in `internal/cli/update.go`.** Replace `writeGuidanceHints`:

  ```go
  func writeGuidanceHints(buffer *bytes.Buffer, report update.Report) {
  	deployed := claudeGuidanceFiles(report)

  	if len(deployed) > 0 {
  		for _, name := range deployed {
  			if report.GuidanceImports[name] {
  				fmt.Fprintf(buffer, "guidance refreshed: ~/.claude/engram/%s\n", name)

  				continue
  			}

  			fmt.Fprintf(buffer,
  				"guidance deployed to ~/.claude/engram/%s — add '@~/.claude/engram/%s'"+
  					" to ~/.claude/CLAUDE.md to activate it (Claude Code will ask you to"+
  					" approve the import once)\n", name, name,
  			)
  		}

  		return
  	}

  	if !report.WithGuidance && !report.GuidanceImported {
  		fmt.Fprintf(buffer,
  			"engram ships recall-firing guidance; run 'engram update --with-guidance' to deploy it\n",
  		)
  	}
  }

  // claudeGuidanceFiles returns the guidance basenames deployed to Claude Code
  // this run (empty if none / harness absent).
  func claudeGuidanceFiles(report update.Report) []string {
  	for _, harness := range report.Harnesses {
  		if harness.Name == update.HarnessClaude {
  			return harness.GuidanceFiles
  		}
  	}

  	return nil
  }
  ```

- [ ] **Step 6: Update the CLI hint test contract (assert per-file behavior — Gate-A finding: the
  original mixed row asserted nothing delegate-specific and passed even with the delegate hint
  deleted).** In `internal/cli/update_test.go` `TestWriteUpdateReport_GuidanceActivationHint`,
  replace the fixed `wantActivation` / `wantPlainHint` booleans with generic `wantContains []string`
  / `wantNotContains []string` columns, and assert each generically
  (`for _, s := range tc.wantContains { g.Expect(out).To(ContainSubstring(s)) }`, likewise `NotTo`
  for `wantNotContains`). Set `GuidanceImports: tc.guidanceImports` on the constructed `report`. Rows:

  - "deployed-not-imported": `guidanceImports: nil`, `wantContains: ["@~/.claude/engram/recall.md"]`.
  - "deployed-and-imported": `guidanceImports: {"recall.md": true}`,
    `wantContains: ["guidance refreshed: ~/.claude/engram/recall.md"]`,
    `wantNotContains: ["add '@~/.claude/engram/recall.md'"]`.
  - "plain-update-not-imported": `wantContains: ["engram ships recall-firing guidance"]`.
  - "plain-update-already-imported": `wantNotContains: ["engram ships", "activate it"]`.
  - **new "mixed-recall-imported-delegate-not":** `guidanceFiles: ["recall.md", "delegate.md"]`,
    `guidanceImports: {"recall.md": true}`, `guidanceImported: true`, `withGuidance: false`,
    `wantContains: ["guidance refreshed: ~/.claude/engram/recall.md", "@~/.claude/engram/delegate.md"]`,
    `wantNotContains: ["add '@~/.claude/engram/recall.md'"]`.

  **Adversarial check (Gate-A required):** after GREEN, temporarily delete the delegate.md activation
  branch in `writeGuidanceHints` and re-run — the mixed row MUST now fail. Restore, confirm green.

- [ ] **Step 7: Run `targ check-full`.** All tests + lint + coverage green. Fix any output-snapshot
  drift in neighboring `TestWriteUpdateReport_*` cases caused by the new per-file wording.

- [ ] **Step 8: Commit.** `feat(update): first-class multi-file guidance import handling`.

---

### Task 3: doc scrub + activation

**Files:**
- Modify: repo `CLAUDE.md` — the `guidance/` structure line.
- Modify (sweep, update where the singular framing is now inaccurate): `docs/GLOSSARY.md`,
  `docs/architecture/c1-system-context.md`, `docs/architecture/c2-containers.md`,
  `docs/architecture/c3-components.md`, `docs/ROADMAP.md`, `README.md`.
- Activation + Quick-Fix reconciliation (outside the repo — Joe's global `~/.claude/CLAUDE.md`; not
  a repo commit; **confirm with Joe first** — it's his personal doctrine).

- [ ] **Step 1: Find every guidance reference.** Run
  `grep -rn "recall-firing\|engram/recall.md\|with-guidance\|guidance" CLAUDE.md docs/ README.md`.
  Read each hit; adding `delegate.md` makes any "the guidance is recall.md" / "recall-firing
  guidance" framing incomplete. Known hit (Gate-A): `docs/ROADMAP.md:150` ("the shipped CLAUDE.md
  guidance already covers these moments") → qualify as "the shipped CLAUDE.md **recall-firing**
  guidance (recall.md)".

- [ ] **Step 2: Update repo `CLAUDE.md`.** Change the `guidance/` line from "Source for the
  deployable recall-firing guidance (activated via CLAUDE.md `@import`)" to name both docs, e.g.
  "Source for the deployable ambient guidance docs — recall-firing (`recall.md`) and
  delegation-firing (`delegate.md`) — activated via CLAUDE.md `@import`."

- [ ] **Step 3: Update docs where inaccurate (Gate-A: objective criterion).** **Generalize** any doc
  that presents the guidance mechanism as a *feature/transport* (CLAUDE.md, README, the C1–C3
  broad-stroke descriptions, the GLOSSARY term, the ROADMAP line) to name the mechanism —
  "guidance docs: recall.md, delegate.md". **Leave** docs that discuss recall's *behavior* or
  *validation results* in depth. When uncertain, update the broad-stroke description, not the deep
  dive.

- [ ] **Step 4: Commit.** `docs: name delegation guidance across CLAUDE.md, C4, glossary, README`.

- [ ] **Step 5: Activate + reconcile the Quick-Fix tier (Gate-A finding B — CONFIRM WITH JOE).**
  After merge: `go install ./cmd/engram/` → `engram update` (deploys `delegate.md`; the CLI hint now
  names it) → add `@~/.claude/engram/delegate.md` to `~/.claude/CLAUDE.md` and approve the import.
  Then reconcile the contradiction the design section names: Joe's global `~/.claude/CLAUDE.md`
  "Producing Artifacts" table still says **Quick Fix | exact files/lines known, single commit | just
  do it** — a difficulty/scope guess, exactly what `delegate.md`'s floor forbids. Propose amending
  that row to gate the inline escape on recalled sub-overhead evidence (e.g. "exact files known
  **and** recalled evidence this task-kind is reliably sub-overhead — see delegate.md"), so the two
  always-loaded instructions stop disagreeing. This edits Joe's personal doctrine file — present the
  exact change and get his nod; do not rewrite it unilaterally. Verify: `engram update` reports
  `guidance refreshed: ~/.claude/engram/delegate.md`.

---

### Plan self-review

- **Spec coverage:** deliverable 1 → Task 1; deliverable 2 → Task 2; deliverable 3 (activation +
  docs + validation) → Task 3 + Task 1's headless validation. Covered.
- **Type consistency:** `detectGuidanceImports`, `guidanceImportBase`, `Report.GuidanceImports`,
  `claudeGuidanceFiles` used consistently across Task 2 steps.
- **Placeholders:** none — full doc body, full Go code, exact test bodies, exact commands.
- **Out-of-scope held:** no route/please skill-body edits; no counterfactual meter; no
  `planGuidanceCopies` change.
- **Gate A (4 angles) resolved 2026-07-07:** delegate.md now carries the overhead-vs-difficulty
  reframe and is compressed to recall.md altitude; the mixed CLI test asserts delegate-specific
  behavior + an adversarial regression check (code-alignment applied the diff in a live worktree —
  compiles, tests pass, golangci-lint + nilaway clean); `guidanceImportBase` branch coverage,
  ROADMAP scrub, the RED/GREEN rubric + LEDGER format + production model, a small-edit eval prompt,
  and the Quick-Fix reconciliation (Joe's call) all added.
