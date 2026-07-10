# Plan — #674: route dispatch evidence notes (tags-based), evidence-linked aggregates, count-as-audit

> Executes the ratified 2026-07-10 design (Joe, recorded as the GH #669 closing comment — "no bespoke
> store, no dedicated aggregate subcommand"). Design is settled; this plan is build-only. Scope
> addition ratified mid-draft: the aggregate-drowning gauge (Task 5) + drowning-audit doc lines.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

## Goal

Replace route's free-text dispatch records with: (1) **evidence notes** — ordinary recallable fact
notes tagged `work-kind/<k>`, `tier/<t>`, `outcome/<o>` in frontmatter `tags:`; (2) **aggregate
fact notes** per work-kind (slug `route-evidence-<work-kind>`) holding tier tallies + wikilinks to
every evidence note, amended per dispatch, read via **plain recall**; (3) three **family definition
notes** (bare-tag convention) written as vault data; (4) **`engram count` documented as the audit
surface** (tally recompute + drowning audit) — never on the routing read path. New binary surface:
repeatable `--tag` on `engram learn fact|feedback` only.

## Architecture

- **Write path:** route (skill) → write-memory (skill) → `engram learn fact --tag ...` (binary) →
  vault note with `tags:` frontmatter (+ auto sidecar). Aggregate updates: route → `engram query`
  lookup → `engram amend --object` (match) or `engram learn fact` (no match). No `--tag` on amend
  or qa (YAGNI — aggregate amends change content fields only).
- **Read path:** unchanged — plain recall (`engram query`). Aggregates and evidence notes surface
  as normal memories. Zero query/recall code changes.
- **Audit path:** `engram count --group-by tags --filter tags=...` recomputes tallies from evidence
  tags (`internal/cli/count.go` — `runCountGroupBy`/`attrValues` already handle YAML list attrs and
  list-contains filters; **no count code changes**).
- **Tags round-trip:** adding `Tags` to `factFrontmatterDoc`/`feedbackFrontmatterDoc`
  (`internal/cli/learn.go` lines 130–145 / 162–177) makes `engram amend`'s decode→override→render
  cycle (`internal/cli/amend.go` — `applyTypedAmend`) preserve `tags:` with zero amend-specific
  code; a test locks this.

## Tech stack

Go (pure-Go, DI-at-edges), targ CLI framework (struct-tag flags, `internal/cli/targets.go`),
yaml.v3 frontmatter, gomega asserts + rapid properties + DI struct-of-funcs mocks
(`internal/cli/learn_test.go` conventions), writing-skills TDD for SKILL.md edits, trap gate
`dev/eval/traps/gate.py`.

## Global constraints

1. **Pure-Go / DI at the edges** — no `os.*`/`http.*` in `internal/` business logic
   (CLAUDE.md Design Principles; ADR-0001..0003). Task 1 touches only arg structs, validation,
   frontmatter rendering — all already DI-clean.
2. **`targ test` / `targ check-full` only** — never `go test`/`go vet` directly (CLAUDE.md Code
   Quality). Binary install is `go install ./cmd/engram` (no `targ build` target).
3. **Trap-gate smoke around skill-text changes** (write-memory is touched — repo standing
   constraint, docs/ROADMAP.md "Standing constraint"): run
   `python3 dev/eval/traps/gate.py --tier smoke`
   (exact usage from `dev/eval/traps/gate.py` docstring: `python3 gate.py --tier smoke|full
   [--workers N]`; script resolves its own dir, so repo-root invocation works; exit 0 only on
   GREEN, prints `overall verdict: GREEN`) **before the first skill edit (Task 2) and after the
   last (Task 3, pre-commit)**. Before-run non-GREEN → STOP (pre-existing regression; do not edit
   skills). After-run non-GREEN → revert the skill edits, diagnose, do not commit.
4. **Win-nucleus untouched** (ROADMAP standing constraint): Step-3 conventions directive, Step-2.5B
   recency weight, Step-2 matched-note retrieval, frontmatter `description` fields. This plan edits
   no recall skill text and no query code — the constraint is satisfied by construction; the Task 3
   diff check re-verifies route's own protected text.
5. **Route's cheapest-first doctrine and its invariant are PRESERVED UNTOUCHED** — "no entry starts
   above cheap without recorded evidence" (`skills/route/SKILL.md` lines 14–21, 39–46, 113–126).
   Task 3 has an explicit diff-region check (Gate A will check vault note 182's lesson).
6. **Commits:** conventional commits, trailer `AI-Used: [claude]`, line length ≤ 120, one commit
   per task (steps below give exact messages).
7. **Vault safety:** Task 4 writes the production vault deliberately (definition notes are DATA).
   Task 5 writes ONLY a scratch vault: every command carries `--vault "$GAUGE_ROOT/vault"`
   (flag > env > XDG per `resolveVault`, `internal/cli/learn.go` lines 425–435 — the flag form is
   the strongest override and satisfies the never-touch-production requirement; equivalently
   `ENGRAM_VAULT_PATH` could be exported, but the per-command flag cannot leak between commands).
   Queries in Task 5 also pin `--chunks-dir "$GAUGE_ROOT/chunks"` so the production chunk index is
   never read.

## Two pinned spec corrections (implementation detail, not design)

**Correction 1 — count shorthand.** The design shorthand "numerator `engram count --group-by
work-kind --filter tags=tier/<t> --filter tags=outcome/pass`" predates the tags representation:
under ratified tags-based notes, `work-kind` is a **tag value**, not a frontmatter attribute, and
`--group-by <attr>` groups by frontmatter attribute only (`internal/cli/count.go` —
`runCountGroupBy` line 237, `attrValues` line 78). The literal `--group-by work-kind` would put
every note in the `(work-kind absent)` bucket. The working realization of the same pattern (used
everywhere below):

- **Numerators (per-kind passes at tier t):**
  `engram count --group-by tags --filter tags=tier/<t> --filter tags=outcome/pass` → read the
  `work-kind/<k>	N` rows.
- **Denominators:** same minus the outcome filter → `work-kind/<k>	D` rows.
- **Single-kind spot check:** add `--filter tags=work-kind/<k>`, read the `total:` line.

Two executors cannot disagree: the audit commands in Tasks 3–6 are spelled exactly this way.

**Correction 2 — definition-note identifiers.** The re-scope comment names the family definition
notes `work-kind.definition` / `tier.definition` / `outcome.definition`. Those dot forms are prose
shorthand, not literal slugs: `slugPattern = ^[a-z0-9-]+$` (`internal/cli/learn.go` line 111,
unchanged by this plan) rejects dots. Task 4 realizes them as the kebab-case slugs
`work-kind-definition` / `tier-definition` / `outcome-definition` — the only spelling the binary
accepts.

**Validation-scope note (deliberate, not an oversight):** `validateTags` is grammar-only
(`^[a-z0-9-]+(/[a-z0-9-]+)?$`). Closed-set membership (tier ∈ cheap|mid|deep, outcome ∈ pass|fail)
is convention documented in the family definition notes, NOT binary-enforced — work-kind is an
open set, and the cardinality guard died with #676. Do not add enum validation.

---

## Task 1 — binary: repeatable `--tag` on `engram learn fact|feedback` (TDD)

**Files**
- `/Users/joe/repos/personal/engram/internal/cli/learn.go` (modify)
- `/Users/joe/repos/personal/engram/internal/cli/targets.go` (modify — `CommonLearnArgs`)
- `/Users/joe/repos/personal/engram/internal/cli/learn_test.go` (add tests)
- `/Users/joe/repos/personal/engram/internal/cli/amend_test.go` (add one round-trip test)
- `/Users/joe/repos/personal/engram/internal/cli/export_test.go` (add one shim)

**Interfaces**
- `LearnArgs.Tags []string`; `CommonLearnArgs.Tags []string` (flag `--tag`, repeatable)
- `func validateTags(tags []string) error` (learn.go; sentinel `errTagInvalid`, regex
  `^[a-z0-9-]+(/[a-z0-9-]+)?$`)
- `factFrontmatterDoc.Tags` / `feedbackFrontmatterDoc.Tags` → `yaml:"tags,omitempty"` (string list,
  emitted between `sources:` and `vocab:`)

### Steps

- [ ] **1.0 Setup.** `git status --short` must be empty (else STOP — dirty tree). The plan doc is
  already committed on main (Gate A gates execution). Then:
  ```bash
  git checkout -b 674-route-evidence-notes
  ```

- [ ] **1.1 RED — add the tests first** (they will not compile until 1.2; a compile failure on
  `targ test` is this task's RED). Append to
  `/Users/joe/repos/personal/engram/internal/cli/learn_test.go` (add `"slices"` to its imports):

  ```go
  // TestLearnFact_Tags_WrittenToFrontmatter locks the --tag surface (#674): tags
  // on LearnArgs land in the frontmatter tags: list, preserving order.
  func TestLearnFact_Tags_WrittenToFrontmatter(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	var written []byte

  	args := cli.LearnArgs{
  		Type: "fact", Slug: "route-dispatch-rename", Vault: t.TempDir(), Position: "top",
  		Source: "test", Situation: "routing rename work",
  		Subject: "rename dispatch at cheap (haiku)", Predicate: "resolved as", Object: "pass",
  		Tags: []string{"work-kind/rename", "tier/cheap", "outcome/pass"},
  	}
  	deps := cli.LearnDeps{
  		Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
  		Getenv:        func(string) string { return "" },
  		StatDir:       func(string) error { return nil },
  		InitVault:     func(string) error { return nil },
  		ListIDs:       func(string) ([]string, error) { return nil, nil },
  		ListBasenames: func(string) ([]string, error) { return nil, nil },
  		Lock:          func(string) (func(), error) { return func() {}, nil },
  		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
  	}

  	var buf strings.Builder

  	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
  	g.Expect(err).NotTo(HaveOccurred())

  	if err != nil {
  		return
  	}

  	note := string(written)
  	g.Expect(note).To(ContainSubstring("tags:"))
  	g.Expect(note).To(ContainSubstring("work-kind/rename"))
  	g.Expect(note).To(ContainSubstring("tier/cheap"))
  	g.Expect(note).To(ContainSubstring("outcome/pass"))
  	g.Expect(strings.Index(note, "work-kind/rename")).
  		To(BeNumerically("<", strings.Index(note, "tier/cheap")), "tag order must be preserved")
  	g.Expect(strings.Index(note, "tier/cheap")).
  		To(BeNumerically("<", strings.Index(note, "outcome/pass")), "tag order must be preserved")
  }

  // TestLearnFact_EmptyTags_NoTagsKey: no --tag flags → no tags: key (omitempty).
  func TestLearnFact_EmptyTags_NoTagsKey(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	var written []byte

  	args := cli.LearnArgs{
  		Type: "fact", Slug: "untagged", Vault: t.TempDir(), Position: "top",
  		Source: "test", Situation: "no tags", Subject: "A", Predicate: "has", Object: "B",
  	}
  	deps := cli.LearnDeps{
  		Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
  		Getenv:        func(string) string { return "" },
  		StatDir:       func(string) error { return nil },
  		InitVault:     func(string) error { return nil },
  		ListIDs:       func(string) ([]string, error) { return nil, nil },
  		ListBasenames: func(string) ([]string, error) { return nil, nil },
  		Lock:          func(string) (func(), error) { return func() {}, nil },
  		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
  	}

  	var buf strings.Builder

  	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
  	g.Expect(err).NotTo(HaveOccurred())

  	if err != nil {
  		return
  	}

  	g.Expect(string(written)).NotTo(ContainSubstring("tags:"))
  }

  // TestLearnFeedback_Tags_WrittenToFrontmatter: the feedback form carries tags too.
  func TestLearnFeedback_Tags_WrittenToFrontmatter(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	var written []byte

  	args := cli.LearnArgs{
  		Type: "feedback", Slug: "tagged-feedback", Vault: t.TempDir(), Position: "top",
  		Source: "test", Situation: "routing rename work",
  		Behavior: "b", Impact: "i", Action: "a",
  		Tags: []string{"work-kind/rename", "outcome/fail"},
  	}
  	deps := cli.LearnDeps{
  		Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
  		Getenv:        func(string) string { return "" },
  		StatDir:       func(string) error { return nil },
  		InitVault:     func(string) error { return nil },
  		ListIDs:       func(string) ([]string, error) { return nil, nil },
  		ListBasenames: func(string) ([]string, error) { return nil, nil },
  		Lock:          func(string) (func(), error) { return func() {}, nil },
  		WriteNew:      func(_ string, data []byte) error { written = data; return nil },
  	}

  	var buf strings.Builder

  	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
  	g.Expect(err).NotTo(HaveOccurred())

  	if err != nil {
  		return
  	}

  	g.Expect(string(written)).To(ContainSubstring("tags:"))
  	g.Expect(string(written)).To(ContainSubstring("work-kind/rename"))
  	g.Expect(string(written)).To(ContainSubstring("outcome/fail"))
  }

  // TestLearnFact_InvalidTag_RejectedBeforeWrite: every malformed tag shape is
  // rejected with the sentinel error, before any file write.
  func TestLearnFact_InvalidTag_RejectedBeforeWrite(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	invalidTags := []string{
  		"Work-Kind/Rename", // uppercase
  		"a/b/c",            // two slashes
  		"work kind/rename", // space
  		"/rename",          // empty family
  		"work-kind/",       // empty value
  		"tier=cheap",       // wrong separator
  		"",                 // empty
  	}

  	for _, tag := range invalidTags {
  		var wrote bool

  		args := cli.LearnArgs{
  			Type: "fact", Slug: "bad-tag", Vault: t.TempDir(), Position: "top",
  			Source: "test", Situation: "s", Subject: "a", Predicate: "b", Object: "c",
  			Tags: []string{"work-kind/ok-sibling", tag},
  		}
  		deps := cli.LearnDeps{
  			Now:           func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
  			Getenv:        func(string) string { return "" },
  			StatDir:       func(string) error { return nil },
  			InitVault:     func(string) error { return nil },
  			ListIDs:       func(string) ([]string, error) { return nil, nil },
  			ListBasenames: func(string) ([]string, error) { return nil, nil },
  			Lock:          func(string) (func(), error) { return func() {}, nil },
  			WriteNew:      func(string, []byte) error { wrote = true; return nil },
  		}

  		var buf strings.Builder

  		err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
  		g.Expect(err).To(MatchError(ContainSubstring("tag must be")), "tag %q must be rejected", tag)
  		g.Expect(wrote).To(BeFalse(), "tag %q must be rejected before any write", tag)
  	}
  }

  // TestValidateTags_* exercise the validator directly, split per behavior
  // (repo convention — cf. TestValidateProjectSlug_AcceptsEmpty /
  // _AcceptsKebabCase / _RejectsBadShape, learn_test.go:903-917).
  func TestValidateTags_AcceptsEmpty(t *testing.T) {
  	t.Parallel()
  	g := NewWithT(t)
  	g.Expect(cli.ExportValidateTags(nil)).To(Succeed())
  }

  func TestValidateTags_AcceptsValidShapes(t *testing.T) {
  	t.Parallel()
  	g := NewWithT(t)
  	g.Expect(cli.ExportValidateTags([]string{"work-kind"})).To(Succeed())
  	g.Expect(cli.ExportValidateTags([]string{"work-kind/rename", "tier/cheap", "outcome/pass"})).To(Succeed())
  	g.Expect(cli.ExportValidateTags([]string{"a1-b2/c3-d4"})).To(Succeed())
  }

  func TestValidateTags_RejectsBadShape(t *testing.T) {
  	t.Parallel()
  	g := NewWithT(t)
  	g.Expect(cli.ExportValidateTags([]string{"outcome/pass", "BAD"})).To(HaveOccurred())
  }

  // TestRenderFactFrontmatter_TagsRoundtripFidelity is a property test: any tag
  // list drawn from the valid grammar passes validateTags AND survives the
  // render→parse YAML roundtrip identically (order and values). Mirrors
  // TestRenderFeedbackFrontmatter_RoundtripFidelity.
  func TestRenderFactFrontmatter_TagsRoundtripFidelity(t *testing.T) {
  	t.Parallel()
  	rapid.Check(t, func(rt *rapid.T) {
  		tagGen := rapid.StringMatching(`[a-z0-9-]{1,12}(/[a-z0-9-]{1,12})?`)
  		tags := rapid.SliceOfN(tagGen, 1, 4).Draw(rt, "tags")

  		if err := cli.ExportValidateTags(tags); err != nil {
  			rt.Fatalf("grammar-valid tags rejected: %v", err)
  		}

  		fields := cli.ExportFactFields{
  			Situation: "s", Subject: "a", Predicate: "b", Object: "c",
  			Luhmann: "1", Source: "src", Tags: tags,
  		}
  		when := time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC)
  		got := cli.ExportRenderFactFrontmatter(fields, when)

  		const delim = "---\n"

  		body := strings.TrimPrefix(got, delim)
  		end := strings.Index(body, "\n"+delim)

  		if end < 0 {
  			rt.Fatalf("no closing delimiter in %q", got)
  		}

  		var doc struct {
  			Tags []string `yaml:"tags"`
  		}

  		if err := yaml.Unmarshal([]byte(body[:end+1]), &doc); err != nil {
  			rt.Fatalf("unmarshal %q: %v", body[:end+1], err)
  		}

  		if !slices.Equal(doc.Tags, tags) {
  			rt.Fatalf("tags: got %v want %v\nfull:\n%s", doc.Tags, tags, got)
  		}
  	})
  }
  ```

  And append to `/Users/joe/repos/personal/engram/internal/cli/amend_test.go`:

  ```go
  // TestRunAmend_PreservesTagsFrontmatter (#674): amending a content field of a
  // tagged fact note must not drop tags: — factFrontmatterDoc.Tags carries the
  // list through amend's decode→override→render cycle. No --tag flag exists on
  // amend by design; preservation is the whole contract.
  func TestRunAmend_PreservesTagsFrontmatter(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)

  	const basename = "1aa.2026-07-10.route-dispatch-rename.md"

  	noteContent := []byte(
  		"---\ntype: fact\nsituation: routing rename work\nsubject: rename dispatch\n" +
  			"predicate: resolved as\nobject: pass\nluhmann: \"1aa\"\ncreated: 2026-07-10\n" +
  			"source: test\ntags:\n    - work-kind/rename\n    - tier/cheap\n    - outcome/pass\n" +
  			"---\n\nbody\n",
  	)

  	var written []byte

  	deps := cli.AmendDeps{
  		Scan: func(string) ([]vaultgraph.Note, error) {
  			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
  		},
  		Read:  func(string) ([]byte, error) { return noteContent, nil },
  		Write: func(_ string, data []byte) error { written = data; return nil },
  		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
  			return map[string]bool{}, nil
  		},
  		Now: func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
  	}
  	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Object: "fail per review verdict"}

  	var buf bytes.Buffer

  	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
  	g.Expect(err).NotTo(HaveOccurred())

  	if err != nil {
  		return
  	}

  	g.Expect(string(written)).To(ContainSubstring("tags:"))
  	g.Expect(string(written)).To(ContainSubstring("work-kind/rename"))
  	g.Expect(string(written)).To(ContainSubstring("tier/cheap"))
  	g.Expect(string(written)).To(ContainSubstring("outcome/pass"))
  	g.Expect(string(written)).To(ContainSubstring("object: fail per review verdict"))
  }
  ```

  Add to the alphabetized shim block in
  `/Users/joe/repos/personal/engram/internal/cli/export_test.go` (after `ExportValidateSlug`,
  line ~110):
  ```go
  	ExportValidateTags         = validateTags
  ```

  Run: `targ test; echo "exit=$?"` — **expected: compile FAILURE mentioning `Tags` / `validateTags`
  undefined, `exit=1`. That is RED.** (If it passes, something already implements tags — STOP and
  investigate.)

- [ ] **1.2 GREEN — implement.** All edits in
  `/Users/joe/repos/personal/engram/internal/cli/learn.go` unless noted.

  (a) `LearnArgs` (line 21) — after the `ChunkSources []string` field (line 42):
  ```go
  	// Tags carries repeatable `--tag` categorical tags: a bare family
  	// ([a-z0-9-]+ — by convention marks a family definition note) or a
  	// family/value pair ([a-z0-9-]+/[a-z0-9-]+ — marks a member). Written to
  	// the frontmatter tags: list; validated by validateTags before any write.
  	Tags []string
  ```

  (b) unexported variables block (lines 102–112) — keep alphabetical order; after
  `errSlugInvalid` add, and after `slugPattern` add:
  ```go
  	errTagInvalid = errors.New("tag must be <family> or <family>/<value>, each segment matching [a-z0-9-]+")
  ```
  ```go
  	tagPattern = regexp.MustCompile(`^[a-z0-9-]+(/[a-z0-9-]+)?$`)
  ```

  (c) `factFields` (line 114) — after `ChunkSources []string`:
  ```go
  	Tags         []string
  ```
  Same addition to `feedbackFields` (line 147).

  (d) `factFrontmatterDoc` (line 130) — between `Sources` and `Vocab`:
  ```go
  	Tags       []string          `yaml:"tags,omitempty"`
  ```
  Same addition, same position, in `feedbackFrontmatterDoc` (line 162).

  (e) `assembleLearnContent` (line 229) — in the `feedbackFields{...}` literal (line 246) append
  `Tags: args.Tags,` and in the `factFields{...}` literal (line 259) append `Tags: args.Tags,`.

  (f) `renderFactFrontmatter` (line 374) — add `Tags: f.Tags,` after `Sources: f.ChunkSources,`.
  Same in `renderFeedbackFrontmatter` (line 401).

  (g) `runLearn` (line 441) — after the `issueErr` block (line 452–455):
  ```go
  	tagErr := validateTags(args.Tags)
  	if tagErr != nil {
  		return fmt.Errorf("learn: %w", tagErr)
  	}
  ```

  (h) New function, alphabetically between `validateSlug` and `validateTier`:
  ```go
  // validateTags rejects any --tag entry that is not a bare family or a
  // family/value pair (kebab-case segments). Empty list is allowed: tags are
  // optional categorical metadata. Convention (#674): a bare family tag marks a
  // family definition note; family/value marks a member. Low-cardinality
  // categoricals only — quantities like duration/cost belong in content fields.
  func validateTags(tags []string) error {
  	for _, tag := range tags {
  		if !tagPattern.MatchString(tag) {
  			return fmt.Errorf("%w: got %q", errTagInvalid, tag)
  		}
  	}

  	return nil
  }
  ```

  (i) `runLearnFromFactArgs` (line 474) — add `Tags: a.Tags,` to the `LearnArgs{...}` literal.
  Same in `runLearnFromFeedbackArgs` (line 496).

  (j) `/Users/joe/repos/personal/engram/internal/cli/targets.go` — `CommonLearnArgs` (line 17),
  after the `ChunkSources` field (line 28):
  ```go
  	Tags []string `targ:"flag,name=tag,desc=categorical tag: <family> or <family>/<value> (kebab-case; repeatable)"`
  ```

- [ ] **1.3 Verify.**
  ```bash
  targ test; echo "exit=$?"
  ```
  Expected: `exit=0`, no failures.
  ```bash
  targ check-full; echo "exit=$?"
  ```
  Expected: `exit=0` (nilaway/lint clean — the tests above already follow
  `.claude/rules/go.md`: `if err != nil { return }` after gomega error asserts, `MatchError`
  instead of `err.Error()`, named test values, `t.Parallel()` everywhere).

- [ ] **1.4 End-to-end smoke on a throwaway vault** (uses the real binary — install first):
  ```bash
  go install ./cmd/engram
  TMPV=$(mktemp -d)
  engram learn fact --vault "$TMPV/vault" --slug tag-smoke --position top \
    --source "674 smoke" --situation "smoke-testing the tag flag" \
    --subject "tags flag" --predicate "writes" --object "tags frontmatter" \
    --tag work-kind/smoke --tag tier/cheap --tag outcome/pass
  ```
  Expected: prints exactly one line matching the regex
  `/vault/1\.[0-9]{4}-[0-9]{2}-[0-9]{2}\.tag-smoke\.md$` (the date is today's — do not hard-pin it).
  ```bash
  engram count --vault "$TMPV/vault" --group-by tags
  ```
  Expected output, exactly (count-desc then value-asc — all counts 1, per
  `renderCountGroupBy`, `internal/cli/count.go` line 179):
  ```
  outcome/pass	1
  tier/cheap	1
  work-kind/smoke	1
  total: 1
  ```
  ```bash
  engram learn fact --vault "$TMPV/vault" --slug bad --position top --source s \
    --situation s --subject s --predicate s --object s --tag 'Work-Kind/Rename'; echo "exit=$?"
  ```
  Expected: stderr `learn: tag must be <family> or <family>/<value>, each segment matching
  [a-z0-9-]+: got "Work-Kind/Rename"`, then `exit=1`.
  ```bash
  rm -rf "$TMPV"
  ```

- [ ] **1.5 Commit.**
  ```bash
  git add internal/cli/learn.go internal/cli/targets.go internal/cli/learn_test.go \
    internal/cli/amend_test.go internal/cli/export_test.go
  git commit -m "feat(learn): repeatable --tag writes tags: frontmatter on fact/feedback (#674)

  Bare family or family/value, kebab-case, validated before write; amend
  round-trips tags via the frontmatter doc structs (no --tag on amend — YAGNI).
  Tags are the sole categorical representation per the 2026-07-10 decision on #669.

  AI-Used: [claude]"
  ```

---

## Task 2 — write-memory skill: `tags` in the handoff contract (writing-skills TDD)

**Files**
- `/Users/joe/repos/personal/engram/skills/write-memory/SKILL.md` (modify)

Invoke `superpowers:writing-skills` for this task (CLAUDE.md: mandatory for any SKILL.md edit).

### Steps

- [ ] **2.1 Trap-gate BEFORE baseline** (Global Constraint 3):
  ```bash
  python3 dev/eval/traps/gate.py --tier smoke; echo "exit=$?"
  ```
  Expected: `overall verdict: GREEN`, `exit=0`. Non-GREEN → **STOP the plan** (pre-existing trap
  regression; report, do not edit any skill).

- [ ] **2.2 RED baseline.** Dispatch ONE fresh subagent (no repo context beyond what's given) with:
  (i) the current verbatim content of `skills/write-memory/SKILL.md`, (ii) this fixture handoff,
  (iii) the instruction "You are executing this handoff per the skill. Compose the exact engram
  command you would run. Do NOT execute it — output only the command."

  Fixture handoff (verbatim):
  ```
  kind: fact
  slug: route-dispatch-gauge-red
  source: "write-memory tags RED baseline (#674)"
  situation: "routing gauge-red work"
  subject: "gauge-red dispatch at cheap (haiku)"
  predicate: "resolved as"
  object: "pass per review verdict; duration: 1000 ms; cost: 1000 tok (no rate on hand)"
  tags: [work-kind/gauge-red, tier/cheap, outcome/pass]
  ```
  **Decision procedure:** count occurrences of the literal string `--tag ` in the returned
  command. RED confirmed iff count == 0 (current skill text never mentions tags, so the field is
  dropped/ignored). If count > 0 (the agent improvised), record the transcript verbatim in the
  Execution Log, and still proceed — the contract must be documented either way, but note the
  baseline as "improvised-pass" rather than RED.

- [ ] **2.3 GREEN — the edit.** Four verbatim changes to
  `/Users/joe/repos/personal/engram/skills/write-memory/SKILL.md`. **Uniqueness pre-check:** each
  quoted anchor below must appear exactly once — `grep -cF '<anchor text>'` on the file prints `1`
  per anchor; any other count → STOP (the file drifted; re-derive the anchor before editing).

  (a) In "The handoff contract" list, after the line
  `- optional **chunk-sources** — \`<source#anchor>\` chunk IDs (provenance)`
  insert:
  ```markdown
  - optional **tags** — categorical `<family>` or `<family>/<value>` strings (kebab-case;
    fact/feedback only), e.g. `work-kind/rename`, `tier/cheap`, `outcome/pass`
  ```

  (b) kind=fact compose block (the fenced bash block directly under the line `kind=fact:`) —
  replace its last line
  `  --subject "<the thing>" --predicate "<requires / must use / is>" --object "<the standard or value>"`
  with:
  ```
    --subject "<the thing>" --predicate "<requires / must use / is>" --object "<the standard or value>" \
    [--tag <family>/<value> ...]
  ```

  (c) kind=feedback compose block (the fenced bash block directly under the line
  `kind=feedback:`) — replace its last line
  `  --behavior "<what was done>" --impact "<why it was wrong/costly>" --action "<what to do instead>"`
  with:
  ```
    --behavior "<what was done>" --impact "<why it was wrong/costly>" --action "<what to do instead>" \
    [--tag <family>/<value> ...]
  ```

  (d) In "Append to any kind:", after the bullet
  `- one \`--chunk-source <source#anchor>\` per provided chunk ID` insert:
  ```markdown
  - one `--tag <t>` per provided tag (fact/feedback only — `engram learn qa` and `engram amend`
    take no `--tag`; a qa handoff carrying tags → drop them and say so in your report)
  ```
  And replace the Rules bullet
  `- Never hand-author vocab tags or wikilinks — the binary assigns vocab automatically.`
  with:
  ```markdown
  - Never hand-author vocab tags or wikilinks — the binary assigns vocab automatically. Handed-off
    `--tag` categoricals are NOT vocab: pass them through exactly as provided; never invent tags.
  ```

- [ ] **2.4 GREEN check + pressure test.** Re-run the 2.2 scenario with the EDITED skill text
  (fresh subagent). Save the subagent's full reply to a file (e.g. `$CLAUDE_JOB_DIR/tmp/wm_green.txt`)
  and decide by grep, not by reading impressionistically. **PASS iff all four greps hold:**
  `grep -oF -- '--tag ' <file> | wc -l` prints exactly `3` (occurrence count — layout-independent,
  works whether the flags share a line or not), and each of
  `grep -cF -- '--tag work-kind/gauge-red' <file>`, `grep -cF -- '--tag tier/cheap' <file>`,
  `grep -cF -- '--tag outcome/pass' <file>` prints exactly `1`. Then the adversarial trial: same
  setup, but a `kind: qa` handoff (any question/answer content) carrying `tags: [work-kind/x]`,
  reply saved to a file. **PASS iff** `grep -cF -- '--tag' <file>` prints `0` for the composed
  `engram learn qa` command AND
  `grep -icE 'tag(s)?.*(drop|omit|ignor|discard|not (supported|applicable|passed))'
  <file>` prints `≥1` (the report acknowledges dropping them). Any FAIL → tighten the edited
  wording, re-run that trial; max 3 iterations, then STOP and report. n=1 per cell (smoke-scale;
  the deterministic string checks leave no judge ambiguity).

- [ ] **2.5 Commit.**
  ```bash
  git add skills/write-memory/SKILL.md
  git commit -m "feat(write-memory): handoff contract gains optional categorical tags (#674)

  fact/feedback compose one --tag per handed-off tag; qa/amend take none;
  tags are passed through exactly, never invented (distinct from vocab).
  writing-skills TDD: RED (tags field dropped) -> GREEN + qa pressure test.

  AI-Used: [claude]"
  ```

---

## Task 3 — route SKILL.md: structured write, aggregate amend-or-create, count audit (writing-skills TDD + pressure test)

**Files**
- `/Users/joe/repos/personal/engram/skills/route/SKILL.md` (modify)
- Reference: `/Users/joe/repos/personal/engram/skills/route/tests/README.md` (baseline
  conventions: author a fresh RED scenario at edit time)

Invoke `superpowers:writing-skills`. **Do-NOT-touch bounds (Global Constraint 5):** lines 14–21
("The rubric is memory…"), the step-2 paragraph lines 39–46 ("Absent evidence, default to the
cheapest…"), and the "Cold-start priors" section lines 113–126 must appear in NO diff hunk.

### Steps

- [ ] **3.1 RED baseline.** Fresh subagent given the CURRENT route SKILL.md + this scenario:
  "You dispatched a subagent for a unit you classified work-kind=doc-edit, at tier cheap (roster:
  cheap=haiku). The review verdict is PASS. The Task-completion usage block reported duration_ms
  42000 and subagent_tokens 45231. Per the skill, record this dispatch. Output every command and
  handoff you would produce (do not execute)."
  **Decision procedure:** RED confirmed iff the output contains 0 occurrences of `write-memory`
  handoff content AND 0 occurrences of `engram query` AND 0 occurrences of `engram amend` /
  `engram learn fact --slug route-evidence-` (the current text only yields the table row). Record
  the transcript.

- [ ] **3.2 GREEN — the edits.** Five verbatim changes to
  `/Users/joe/repos/personal/engram/skills/route/SKILL.md`. **Uniqueness pre-check (grep is
  line-based — multi-line quotes cannot be grepped whole; use these pinned SINGLE-LINE anchors,
  each measured `grep -cF` = 1 against the file at plan-write time):**
  - E1 → `Y — needs mid"). Recalled evidence sets the starting tier.`
  - E2 → `After each dispatch resolves, record one line of evidence.`
  - E3 → `other names without changing this table.`
  - E4 → `3. The record lands in your session transcript`
  - E5 → `| You skipped recording the dispatch outcome`

  Run `grep -cF '<anchor>' skills/route/SKILL.md` for each before editing; any count ≠ 1 → STOP
  and re-derive. (Line numbers cited below are locators as of f2fedb8b; the verbatim quoted text
  is the binding anchor.)

  **(E1) Read side** — replace (current lines 36–38):
  ```markdown
  1. **Recall first (you, the orchestrator).** Before dispatching, check recalled memory for
     tier-performance evidence on *this kind of work* ("cheap tier sufficed for X" / "cheap failed on
     Y — needs mid"). Recalled evidence sets the starting tier.
  ```
  with:
  ```markdown
  1. **Recall first (you, the orchestrator).** Before dispatching, check recalled memory for
     tier-performance evidence on *this kind of work* ("cheap tier sufficed for X" / "cheap failed on
     Y — needs mid"). Recalled evidence sets the starting tier. Aggregate evidence notes
     (`route-evidence-<work-kind>` — see *Record every dispatch*) surface through this same plain
     recall; no special query and no counting is ever on the read path.
  ```

  **(E2) Record section intro** — in current line 73, replace the sentence
  `After each dispatch resolves, record one line of evidence.` with
  `After each dispatch resolves, record one evidence row for the mini-report AND make the
  structured vault write (below).`

  **(E3) The structured write + count audit** — insert immediately after the "Current harness"
  paragraph (which ends `...re-exposes them under other names without changing this table.`,
  current line 95), before `## The loop that improves the rubric`:

  ````markdown
  ### The structured write (one evidence note + one aggregate update per dispatch)

  The table row is the user-facing mini-report; the durable evidence is a vault write. After each
  dispatch resolves (review verdict in hand), do BOTH:

  **(a) Evidence note — hand off to write-memory.** kind=fact, tags carrying the three categoricals
  (low-cardinality only — duration/cost stay in the object prose with explicit units, never tags):

  - slug: `route-dispatch-<work-kind>`
  - tags: `work-kind/<k>`, `tier/<cheap|mid|deep>`, `outcome/<pass|fail>`
  - situation: "routing <work-kind> work"
  - subject: "<work-kind> dispatch at <tier> (<model @ dispatch>)"
  - predicate: "resolved as"
  - object: "<pass|fail> per review verdict; why: <recalled evidence|memory-discount|cold-start
    default>; escalation: <none|passed at TIER>; duration: <duration_ms> ms; cost:
    <subagent_tokens> tok (<rate note, or 'no rate on hand'>)"
  - source: "route dispatch record, <project>, <date>"

  Work-kind values are kebab-case, an open set — reuse a prior kind before minting a new one
  ([[work-kind-definition]] documents the family; tier and outcome are closed sets, see
  [[tier-definition]] and [[outcome-definition]]).

  **(b) Aggregate update — amend or create `route-evidence-<work-kind>`.** Look it up:

  ```bash
  engram query --lazy-chunks --phrase "route evidence <work-kind> tier tally"
  ```

  Deterministic check: for each returned `path:` (items or any cluster's candidate_l2s), take the
  basename (text after the last `/`), strip `.md`, and split on `.` — the final segment is the
  slug. A match iff that slug EQUALS `route-evidence-<work-kind>` exactly. Prefix/fuzzy matches do
  not count.

  - **Match** → recompute the tally from the aggregate's current object text (already in the query
    payload — notes render full content under --lazy-chunks) plus this dispatch, then:

    ```bash
    engram amend --target <matched basename, no .md> \
      --object "<tier tallies, e.g. cheap 14/16, mid 2/2> as of <date> — evidence: [[<existing
      evidence wikilinks, kept>]], [[<new evidence-note basename>]]"
    ```

  - **No match** → create it (NO tags — aggregates are prose summaries, not evidence rows):

    ```bash
    engram learn fact --slug route-evidence-<work-kind> --position top \
      --source "route dispatch record, <project>, <date>" \
      --situation "routing <work-kind> work: which tier the evidence supports" \
      --subject "route evidence for <work-kind>" \
      --predicate "tallies" \
      --object "<tier> 1/1 as of <date> — evidence: [[<evidence-note basename>]]"
    ```

  The wikilink list inside the object text is the aggregate's evidence trail — every evidence note
  it summarizes, append-only.

  ### Count as audit (never on the read path)

  Aggregate tallies are LLM-maintained and WILL drift. `engram count` recomputes ground truth from
  the evidence notes' tags — use it to verify/repair an aggregate, never to route (routing reads
  are plain recall). Note `--group-by work-kind` would NOT work: work-kind is a tag value, not a
  frontmatter attribute.

  ```bash
  # numerators: passes per work-kind at tier <t> — read the work-kind/<k> rows
  engram count --group-by tags --filter tags=tier/<t> --filter tags=outcome/pass
  # denominators: all dispatches per work-kind at tier <t> — read the work-kind/<k> rows
  engram count --group-by tags --filter tags=tier/<t>
  # single-kind spot check: read the "total:" line
  engram count --group-by tags --filter tags=work-kind/<k> --filter tags=tier/<t>
  ```

  Per kind: numerator P over denominator D → "<t> P/D". Run this when a tally is doubted and at
  periodic consolidation (the same moment the consolidation paragraph below names); if count
  disagrees with an aggregate, amend the aggregate to the recomputed numbers.

  **Drowning audit (same trigger moments):** run the (b) lookup query for a work-kind with many
  evidence notes and confirm the aggregate still surfaces (its `path:` in items or any cluster's
  candidate_l2s). If it does not, that is the pre-registered drowning case — report it rather than
  patching ad hoc; the two candidate remedies are named in `docs/architecture/adr.md` (ADR-0019).
  ````

  **(E4) The loop, step 3** — replace (current lines 101–102 first sentence):
  ```markdown
  3. The record lands in your session transcript, which **`/learn`**'s sweep auto-ingests as
     recallable memory — so even uncrystallized, a tier outcome is retrievable next time. When a
  ```
  with:
  ```markdown
  3. The evidence note and amended aggregate are recallable immediately (and the mini-report row in
     your transcript still auto-ingests via **`/learn`**'s sweep). When a
  ```

  **(E5) Red-flags table** — insert a new row directly after the row
  `| You skipped recording the dispatch outcome | The record IS the rubric; no record → no evidence → the rubric never improves. |`:
  ```markdown
  | You produced the table row but skipped the evidence note or aggregate update | Do the structured write: write-memory handoff (tags work-kind/tier/outcome), then amend-or-create `route-evidence-<work-kind>`. |
  ```

- [ ] **3.3 Do-NOT-touch verification.**
  ```bash
  git diff -U0 skills/route/SKILL.md
  ```
  **Decision procedure (content-based, not line-based):** every hunk's changed lines must lie
  within one of the five edit regions, each identified by its pinned single-line anchor from
  3.2's pre-check list. Then run BOTH checks over the five protected single-line fragments —
  `starts at the cheapest` · `Absent evidence, default to the cheapest` · `Cold-start priors` ·
  `no entry starts above` · `cheap without recorded evidence` (the last two jointly cover the
  doctrine sentence, which wraps across two file lines and cannot be grepped whole; each fragment
  measured `grep -cF` = 1 at plan-write time):
  1. In-file: `grep -cF '<fragment>' skills/route/SKILL.md` prints exactly `1` for each.
  2. In-diff: `git diff -U0 skills/route/SKILL.md | grep -cF '<fragment>'` prints exactly `0` for
     each (`-U0` is load-bearing — default context windows pull adjacent protected lines into
     hunks and false-fail this check).
  Any violation → revert that hunk before proceeding. Zero tolerance.

- [ ] **3.4 GREEN check + pressure test.** Fresh subagent, EDITED skill text, the 3.1 scenario
  verbatim. Save the reply to a file and decide by grep. **PASS iff ALL FOUR hold:** (1)
  `grep -ci 'write-memory' <file>` ≥1 AND `grep -cE 'kind[:=][[:space:]]*fact' <file>` ≥1 (the
  handoff names kind=fact, either separator spelling), AND each of
  `grep -cF 'work-kind/doc-edit' <file>`, `grep -cF 'tier/cheap' <file>`,
  `grep -cF 'outcome/pass' <file>` ≥1; (2) `grep -cF 'engram query --lazy-chunks --phrase "route
  evidence doc-edit tier tally"' <file>` ≥1; (3) `grep -cF 'engram amend' <file>` ≥1 AND
  `grep -cF 'route-evidence-doc-edit' <file>` ≥1 (both branches stated, slug exact); (4)
  `grep -c '|' <file>` ≥2 (the mini-report table row still produced). Second pressure trial
  (no-match branch): same scenario plus "the query returned no note whose slug segment equals
  route-evidence-doc-edit", reply saved to a file — **PASS iff** `grep -cF 'engram learn fact'
  <file>` ≥1 AND `grep -cF -- '--slug route-evidence-doc-edit' <file>` ≥1 AND the create command's
  own text carries no tag flags, scoped by a pinned window (the create template is ≤10 lines):
  `grep -B2 -A12 -- '--slug route-evidence-doc-edit' <file> | grep -cF -- '--tag'` prints `0`.
  (The window, not the whole reply, is the scope — the same reply legitimately contains the
  evidence-note handoff, whose tags ride as handoff fields, not `--tag` flags.)
  Any FAIL → tighten wording, re-run the failed trial;
  max 3 iterations, then STOP and report. n=1 per cell.

- [ ] **3.5 Trap-gate AFTER** (Global Constraint 3):
  ```bash
  python3 dev/eval/traps/gate.py --tier smoke; echo "exit=$?"
  ```
  Expected: `overall verdict: GREEN`, `exit=0`. Non-GREEN → revert Task 2+3 skill edits
  (`git checkout -- skills/`), diagnose, STOP.

- [ ] **3.6 Commit + deploy.**
  ```bash
  git add skills/route/SKILL.md
  git commit -m "feat(route): structured dispatch evidence — tagged notes + aggregate amend + count audit (#674)

  Each dispatch: write-memory handoff (fact, tags work-kind/tier/outcome) and
  amend-or-create route-evidence-<work-kind> (tallies + evidence wikilinks).
  Reads stay plain recall; count documented as tally/drowning audit only.
  Cheapest-first doctrine untouched (diff-region verified). writing-skills TDD
  RED->GREEN + two pressure trials; trap gate smoke GREEN before/after.

  AI-Used: [claude]"
  engram update
  ```
  Expected `engram update` output: lists the refreshed skills including `route` and `write-memory`
  for each detected harness.

---

## Task 4 — family definition notes (vault DATA writes + count verification)

No repo files change here except the plan's Execution Log. These three commands write the
**production vault** (default resolution — no `--vault` flag) via the Task-1 binary
(`go install` already run in step 1.4). The `--issue` flag is pre-existing
(`internal/cli/targets.go:25`), not added by Task 1.

### Steps

- [ ] **4.1 Write the three definition notes** (full literal commands — run exactly; each prints
  one line, the absolute note path, ending `.work-kind-definition.md` / `.tier-definition.md` /
  `.outcome-definition.md` respectively — record all three paths):

  ```bash
  engram learn fact --slug work-kind-definition --position top --issue 674 \
    --source "route evidence build (#674), ratified design 2026-07-10 (GH #669 closing comment)" \
    --tag work-kind \
    --situation "classifying route dispatch evidence by work-kind, or auditing work-kind tags" \
    --subject "the work-kind tag family" \
    --predicate "classifies" \
    --object "route dispatch evidence notes by the unit's shape and concept. Values are an open set, kebab-case (work-kind/<value>, e.g. work-kind/single-file-refactor) — reuse an existing value before minting a new one. Convention: the bare tag work-kind marks this definition note; nested work-kind/<value> marks a member. Counting pattern (audit only, never the routing read path): engram count --group-by tags --filter tags=tier/<t> --filter tags=outcome/pass — the work-kind/<k> rows are per-kind pass counts; drop the outcome filter for denominators"
  ```

  ```bash
  engram learn fact --slug tier-definition --position top --issue 674 \
    --source "route evidence build (#674), ratified design 2026-07-10 (GH #669 closing comment)" \
    --tag tier \
    --situation "tagging route dispatch evidence by tier, or auditing tier tags" \
    --subject "the tier tag family" \
    --predicate "records" \
    --object "which model tier a route dispatch ran at. Closed set: tier/cheap, tier/mid, tier/deep (route's roster maps tiers to concrete models at dispatch time; the model name is provenance in the evidence note's subject, never a tag). Convention: the bare tag tier marks this definition note; nested tier/<value> marks a member. Counting pattern (audit only): restrict engram count with --filter tags=tier/<value>"
  ```

  ```bash
  engram learn fact --slug outcome-definition --position top --issue 674 \
    --source "route evidence build (#674), ratified design 2026-07-10 (GH #669 closing comment)" \
    --tag outcome \
    --situation "tagging route dispatch evidence by outcome, or auditing outcome tags" \
    --subject "the outcome tag family" \
    --predicate "records" \
    --object "the review/gate verdict of a route dispatch. Closed set: outcome/pass, outcome/fail — always the reviewer's verdict, never the subagent's self-report. Convention: the bare tag outcome marks this definition note; nested outcome/<value> marks a member. Counting pattern (audit only): --filter tags=outcome/pass as numerator against the unfiltered denominator in engram count"
  ```

- [ ] **4.2 Verify via count** (bare-family filters match ONLY definition notes — evidence notes
  carry `family/value`, never the bare family; list-contains is exact string match,
  `internal/cli/count.go` `matchesAllFilters` line 110):
  ```bash
  engram count --group-by tags --filter tags=work-kind
  engram count --group-by tags --filter tags=tier
  engram count --group-by tags --filter tags=outcome
  ```
  **PASS condition, each command:** output is exactly two lines — `<family>	1` then `total: 1`.
  Anything else (0 rows, count > 1, extra rows) → a write failed or an unexpected bare-tagged note
  exists: STOP this task, inspect with `engram show <basename>`, fix, re-verify.

- [ ] **4.2b Obsidian hand-verification (AC item; user-verifiable, non-blocking for the executor).**
  The AC requires the count numbers be hand-verifiable in Obsidian's tag pane. The executor cannot
  drive a GUI; the step is: record in the Execution Log the exact check for Joe — "open the vault
  (`~/.local/share/engram/vault`) in Obsidian → tag pane → expand `work-kind` / `tier` / `outcome`:
  each bare family tag shows exactly 1 note (the definition note); after evidence notes accrue,
  each `family/value` tag's pane count must equal the corresponding `engram count --group-by tags
  --filter tags=<family>/<value>` total." Caution for the reader: Obsidian's tag pane aggregates
  nested-child counts into the parent row, so once `family/value` evidence accrues the bare-family
  pane number reads 1+N while `engram count --filter tags=<family>` (exact match) stays 1 — the
  audit figures are the `family/value` rows, never the parent row. Log status: DOCUMENTED-FOR-JOE
  (plus the expected values as of this run). Joe's confirmation is welcome but does not block
  execution.

- [ ] **4.3 Recallability check (non-blocking, pre-registered as WARN-only — retrieval ranking on
  the live vault is not this task's contract):**
  ```bash
  engram query --lazy-chunks --phrase "work-kind tag family definition" | \
    grep -c 'work-kind-definition\.md'
  ```
  Expected: `1` or more → note PASS in the Execution Log. `0` → note WARN in the Execution Log and
  continue (the drowning question is Task 5's job; do not tune anything here).

- [ ] **4.4 Commit** (Execution Log only):
  Append to the Execution Log below: the three note paths, the three count outputs, the 4.3
  verdict. Then:
  ```bash
  git add docs/superpowers/plans/2026-07-10-674-route-evidence-notes.md
  git commit -m "docs(plan): #674 execution log — family definition notes written, count-verified

  AI-Used: [claude]"
  ```

---

## Task 5 — aggregate-drowning gauge (scratch vault; pre-registered STOP)

Ratified scope addition (2026-07-10): operationalize the drowning concern as a gauge, NOT a
mechanism — no ride-along/edge work (standing rule: a new edge type must first demonstrate
retrieval value — ADR-0011, ROADMAP standing constraint, vault note 73).

### Steps

- [ ] **5.1 Seed the scratch vault** (every command pins `--vault`; production vault and chunk
  index are never touched — Global Constraint 7):
  ```bash
  GAUGE_ROOT=$(mktemp -d)
  mkdir -p "$GAUGE_ROOT/chunks"
  TODAY=$(date +%F)
  BASENAMES=()
  for i in $(seq 1 20); do
    NOTE_PATH=$(engram learn fact --vault "$GAUGE_ROOT/vault" \
      --slug "route-dispatch-gauge-test-$i" --position top \
      --source "drowning gauge fixture $i (#674)" \
      --tag work-kind/gauge-test --tag tier/cheap --tag outcome/pass \
      --situation "routing gauge-test work" \
      --subject "gauge-test dispatch $i at cheap (haiku)" \
      --predicate "resolved as" \
      --object "pass per review verdict; why: cold-start default; escalation: none; duration: ${i}000 ms; cost: ${i}000 tok (no rate on hand)")
    BASENAMES+=("$(basename "$NOTE_PATH" .md)")
  done
  echo "seeded ${#BASENAMES[@]} evidence notes"
  ```
  Expected: `seeded 20 evidence notes` (each learn printed a path under `$GAUGE_ROOT/vault`).
  ```bash
  LINKS=$(printf '[[%s]], ' "${BASENAMES[@]}"); LINKS=${LINKS%, }
  engram learn fact --vault "$GAUGE_ROOT/vault" --slug route-evidence-gauge-test --position top \
    --source "drowning gauge aggregate fixture (#674)" \
    --situation "routing gauge-test work: which tier the evidence supports" \
    --subject "route evidence for gauge-test" \
    --predicate "tallies" \
    --object "cheap 20/20 as of $TODAY — evidence: $LINKS"
  ```
  Expected: one path ending `.route-evidence-gauge-test.md`.

- [ ] **5.2 Count-audit parity on the fixtures** (verifies the documented audit pattern
  end-to-end):
  ```bash
  engram count --vault "$GAUGE_ROOT/vault" --group-by tags \
    --filter tags=tier/cheap --filter tags=outcome/pass
  ```
  **PASS condition:** output contains the row `work-kind/gauge-test	20` and the line `total: 20`
  (plus rows `outcome/pass	20`, `tier/cheap	20`). Any other numbers → STOP, diagnose seeding.

- [ ] **5.3 THE GAUGE — the route read query against the seeded vault:**
  ```bash
  engram query --vault "$GAUGE_ROOT/vault" --chunks-dir "$GAUGE_ROOT/chunks" --lazy-chunks \
    --phrase "route evidence gauge-test tier tally" > "$GAUGE_ROOT/query-out.yaml"
  grep -E 'path: .*\.route-evidence-gauge-test\.md$' "$GAUGE_ROOT/query-out.yaml" \
    && echo GAUGE-PASS || echo GAUGE-FAIL
  ```
  **Executable decision procedure:** the grep covers both surfaces — top-level `items:` and every
  cluster's `candidate_l2s:` (both render `path:` fields — `internal/cli/query.go` lines 288/314).
  - Prints `GAUGE-PASS` (grep matched ≥1 line) → the aggregate surfaces over 20 sibling evidence
    notes. Clean up and continue:
    ```bash
    rm -rf "$GAUGE_ROOT"
    ```
  - Prints `GAUGE-FAIL` → **pre-registered STOP.** Do NOT delete `$GAUGE_ROOT` (keep
    `query-out.yaml` as the measured case). Record FAIL + the file path in the Execution Log,
    commit the log (message below), and STOP the plan before Task 6 — surface to Joe with the two
    pre-named remedies: (a) a "summarizes" ride-along edge (supersession-shaped insertion of the
    aggregate when its evidence surfaces), (b) demoting evidence notes to the chunk-population
    ranking tier. The choice is made with the measured case in hand, per the standing new-edge
    rule. No silent continuation, no ad-hoc patch.

- [ ] **5.4 Commit** (Execution Log: gauge verdict + 5.2 output):
  ```bash
  git add docs/superpowers/plans/2026-07-10-674-route-evidence-notes.md
  git commit -m "docs(plan): #674 execution log — drowning gauge verdict recorded

  AI-Used: [claude]"
  ```

---

## Task 6 — docs: GLOSSARY, FEATURES, ROADMAP, ADR

(Only reached on GAUGE-PASS; the shipped wording below asserts the gauge result.)

**Files**
- `/Users/joe/repos/personal/engram/docs/GLOSSARY.md`
- `/Users/joe/repos/personal/engram/docs/FEATURES.md`
- `/Users/joe/repos/personal/engram/docs/ROADMAP.md`
- `/Users/joe/repos/personal/engram/docs/architecture/adr.md`
- `/Users/joe/repos/personal/engram/README.md` (CLI reference — 6.2c)

### Steps

- [ ] **6.1 GLOSSARY — three edits.**

  (a) New `--tag` entry: insert immediately BEFORE the heading line `### \`--source\``
  (currently line 395; the `--supersedes` entry ends just above it):
  ```markdown
  ### `--tag`
  Repeatable categorical-tag flag on `engram learn fact` / `engram learn feedback` (not qa, not
  amend — amend nonetheless round-trips an existing `tags:` list unchanged). Each value must match
  `[a-z0-9-]+` (bare family) or `[a-z0-9-]+/[a-z0-9-]+` (family/value); anything else is rejected
  before any write. Writes the frontmatter `tags:` string list — the sole categorical
  representation (ADR-0019). Distinct from the binary-assigned `vocab:` channel.

  ```

  (b) New section: insert immediately BEFORE the lines (currently 487–489):
  ```
  ---

  ## Transcript
  ```
  the following:
  ```markdown
  ---

  ## Route evidence

  ### evidence note (route)
  An ordinary fact note recording one route dispatch, written via write-memory when the dispatch
  resolves. Carries frontmatter `tags: [work-kind/<k>, tier/<t>, outcome/<o>]` — the three
  low-cardinality categoricals; duration/cost live in the object prose with explicit units, never
  in tags. Fully recallable (no query exclusion, no new note type) — the structured replacement for
  route's old free-text transcript record. Slug convention: `route-dispatch-<work-kind>`.

  ### aggregate note (route)
  One fact note per work-kind, slug `route-evidence-<work-kind>`, whose object text states the
  current tier tallies ("cheap 14/16, mid 2/2 as of <date>") and wikilinks every evidence note it
  summarizes (append-only trail). Amended (`engram amend --object`) as each dispatch lands; created
  untagged. Route READS it via plain recall — it surfaces as a normal memory; `engram count` over
  the evidence notes' tags is the audit that verifies/repairs its tallies (ADR-0019).

  ### family definition note / bare-tag convention
  A fact note documenting a tag family — its meaning, allowed values, and counting pattern —
  carrying the BARE family tag (e.g. `tags: [work-kind]`). Convention: a bare family tag marks the
  family's definition note; a nested `family/value` tag marks a member. Three ship with #674:
  `work-kind` (open kebab-case set), `tier` (cheap|mid|deep), `outcome` (pass|fail) — slugs
  `work-kind-definition`, `tier-definition`, `outcome-definition`. Vault data, not repo files.
  ```

  (c) `engram count` entry: replace its final sentence `See ADR-0018.` (currently line 581) with:
  ```markdown
  See ADR-0018. Since #674 it is also the audit surface for route's dispatch-evidence tallies —
  `--group-by tags --filter tags=...` recomputes ground truth from evidence-note tags (see
  "aggregate note (route)" and ADR-0019); audit only, never the routing read path.
  ```

- [ ] **6.2 FEATURES — two edits.**

  (a) Count entry: after the sentence ending `...without frontmatter-listing them).`
  (currently line 166), append to the same paragraph:
  ```markdown
  With #674, count is also the audit surface for route's dispatch evidence:
  `--group-by tags --filter tags=tier/<t> [--filter tags=outcome/pass]` recomputes true
  tier×work-kind tallies from evidence-note tags to verify/repair the LLM-maintained aggregate
  notes — never on the routing read path (plain recall reads the aggregates).
  ```

  (b) New capability entry: insert immediately BEFORE the heading
  `## Validated goals (mission rollup — not a capability)` (currently line 174):
  ```markdown
  ## Route dispatch evidence + aggregates (tags-based)

  Every route dispatch is recorded as an ordinary recallable fact note tagged with three
  categoricals (`work-kind/<k>`, `tier/<t>`, `outcome/<o>` in frontmatter `tags:`, written by the
  repeatable `engram learn --tag` flag), and each work-kind keeps one aggregate fact note
  (`route-evidence-<work-kind>`) whose object text holds the running tier tallies plus wikilinks to
  every evidence note. Route reads evidence by plain recall — aggregates surface as normal
  memories; `engram count` recomputes tallies from tags as the drift audit. Family definition notes
  (bare-tag convention) document the three tag families in the vault itself.

  why: `docs/architecture/adr.md` — ADR-0019 (the 2026-07-10 decision on #669); issue #674
  validation: `internal/cli/learn_test.go` (`TestLearnFact_Tags_WrittenToFrontmatter`,
  `TestLearnFact_InvalidTag_RejectedBeforeWrite`, `TestRenderFactFrontmatter_TagsRoundtripFidelity`)
  + `internal/cli/amend_test.go` (`TestRunAmend_PreservesTagsFrontmatter`); scratch-vault drowning
  gauge PASS at 20 sibling evidence notes + count recompute parity (2026-07-10 — this plan's
  execution log, retired to git history at cycle close)

  ```

- [ ] **6.2c README — CLI reference gains `--tag`.** In `README.md` (lines 78–79), replace the two
  learn signature lines:
  ```
  engram learn feedback --slug ... --source ... --situation ... --behavior ... --impact ... --action ... [--project <slug>] [--issue <id>]
  engram learn fact     --slug ... --source ... --situation ... --subject ... --predicate ... --object ... [--project <slug>] [--issue <id>]
  ```
  with:
  ```
  engram learn feedback --slug ... --source ... --situation ... --behavior ... --impact ... --action ... [--tag <family>[/<value>] ...] [--project <slug>] [--issue <id>]
  engram learn fact     --slug ... --source ... --situation ... --subject ... --predicate ... --object ... [--tag <family>[/<value>] ...] [--project <slug>] [--issue <id>]
  ```

- [ ] **6.3 ROADMAP — three edits.**

  (a) Insert a new paragraph immediately AFTER the paragraph beginning
  `**Also recently shipped:** **\`engram count\`** (ADR-0018)...` (currently line 26):
  ```markdown
  **Also shipped 2026-07-10:** **#674** — route dispatch evidence, tags-based: each dispatch lands
  as an ordinary recallable evidence note (`tags: [work-kind/<k>, tier/<t>, outcome/<o>]` via the
  new repeatable `engram learn --tag` flag) plus an amended per-work-kind aggregate fact note
  (`route-evidence-<work-kind>`, tier tallies + evidence wikilinks) surfaced by **plain recall**;
  `engram count` is the tally/drowning **audit** surface only (ADR-0019). The scratch-vault
  drowning gauge passed at 20 sibling evidence notes; if drowning is ever measured on the real
  vault, the pre-registered remedies are (a) a "summarizes" ride-along edge or (b) demoting
  evidence notes to the chunk-population ranking tier — chosen with the measured case in hand, per
  the standing new-edge rule (ADR-0019 records both).
  ```

  (b) DELETE the #674 bullet from "Actionable now" — the verbatim lines (currently 31–35):
  ```
  - **#674** (M) — route evidence, re-scoped by the **2026-07-10 decision (Joe, recorded on #669)**: dispatch
    evidence notes (ordinary recallable fact notes, `tags: [work-kind/<k>, tier/<t>, outcome/<o>]`) +
    evidence-linked aggregate fact notes per work-kind, amended per dispatch and surfaced by **plain recall**;
    `engram count` repositioned as the audit/recompute surface (never on the recall path). #669 (bespoke store)
    closed subsumed; #676 (attr-node dual-write) closed moot — tags are the single categorical representation.
  ```

  (c) In the "Parked (revisit on trigger)" line (currently line 46), replace the fragment
  `#670 rubric-refit (needs accrued dispatch evidence from #674; #669 closed subsumed)` with
  `#670 rubric-refit (needs accrued evidence — #674 shipped the evidence/aggregate notes
  2026-07-10; #669 closed subsumed)`.

- [ ] **6.4 ADR — new ADR-0019.** Insert between ADR-0018's final paragraph (ends
  `...locks the divergence case.`, currently line 454) and the heading
  `## Decisions deliberately NOT made into ADRs` (currently line 456):

  ```markdown
  ---

  ## ADR-0019 — Tags are the sole categorical representation; recall reads, count audits

  **Status:** Accepted (2026-07-10 — Joe's decision recorded on #669; shipped via #674)

  **Context.** Route's dispatch records were free transcript text — recallable as fuzzy chunks but
  not aggregable (ADR-0017's deferred ledger, #669). ADR-0018 shipped `engram count` as a general
  aggregation surface. The overlap needed one representation for low-cardinality categoricals
  (work-kind, tier, outcome) that recall, counting, and Obsidian can all read without a bespoke
  store.

  **Decision.** Frontmatter `tags:` — a plain YAML string list written by the repeatable
  `engram learn --tag <family>[/<value>]` flag (fact/feedback only; not qa, not amend, though amend
  round-trips an existing list) — is the **sole** categorical mechanism: no attr nodes, no
  categorical wikilinks, no bespoke tables (#676 closed moot; #669 closed subsumed). Three note
  roles ride on it: **evidence notes** (one per route dispatch, tagged `work-kind/<k>`, `tier/<t>`,
  `outcome/<o>`; ordinary recallable facts — no query exclusion); **aggregate notes** (one per
  work-kind, slug `route-evidence-<work-kind>`, object text = tier tallies + wikilinks to every
  summarized evidence note; amended per dispatch; untagged); **family definition notes** (bare
  family tag = definition, nested `family/value` = member; tier: cheap|mid|deep, outcome:
  pass|fail, work-kind: open kebab-case set). Route's read path is **plain recall** — aggregates
  surface as normal memories. `engram count --group-by tags --filter tags=...` is the **audit**
  surface: it recomputes true tallies from evidence tags to verify/repair the LLM-maintained
  aggregates, and is never on the read path. (`--group-by work-kind` does not apply — work-kind is
  a tag value, not a frontmatter attribute.)

  **Consequences.** LLM-maintained tallies WILL drift; count makes them falsifiable (audit commands
  live in `skills/route/SKILL.md`). Evidence notes stay in recall — excluding them would regress on
  the already-recallable free-text records they replace. The aggregate-drowning risk (many
  near-identical evidence notes outranking their aggregate on the read query) is gauged, not
  pre-engineered: a scratch-vault gauge (20 sibling evidence notes + 1 aggregate; PASS = the
  aggregate's path appears in the read query's items or candidate_l2s) passed 2026-07-10, and the
  same check is documented in the route skill as a standing drowning audit. **Pre-registered
  follow-up** if drowning is ever measured on the real vault: (a) a "summarizes" ride-along edge
  (supersession-shaped insertion of the aggregate when its evidence surfaces) or (b) demoting
  evidence notes to the chunk-population ranking tier — choose with the measured case in hand, per
  the standing rule that a new edge type must first demonstrate retrieval value (ADR-0011; the
  ROADMAP standing constraint; vault note 73). Vocab's hub-note channel migrates to this tags
  convention under #678.
  ```

- [ ] **6.5 Verify + commit.**
  ```bash
  targ check-full; echo "exit=$?"
  ```
  Expected `exit=0` (docs-only diff since Task 3; guards against accidental code edits).
  ```bash
  grep -n "ADR-0019" docs/architecture/adr.md docs/GLOSSARY.md docs/FEATURES.md docs/ROADMAP.md | wc -l
  ```
  Expected: ≥ 5 (all four docs cross-reference the new ADR).
  ```bash
  git add docs/GLOSSARY.md docs/FEATURES.md docs/ROADMAP.md docs/architecture/adr.md README.md
  git commit -m "docs: route evidence notes, tags convention, count-as-audit — ADR-0019 (#674)

  GLOSSARY: --tag entry + Route evidence section (evidence/aggregate/definition
  notes, bare-tag convention). FEATURES: count audit note + new capability
  entry. ROADMAP: #674 shipped wording, #670 dependency now 'accrued evidence'.
  README: learn signatures gain --tag. ADR-0019 records
  tags-as-sole-categorical + count-as-audit + the pre-registered drowning
  remedies (references ADR-0011/0017/0018).

  AI-Used: [claude]"
  ```

---

## Execution log (filled at execution time)

- Task 2.1 trap gate BEFORE: _verdict, date_
- Task 2.2 RED baseline: _--tag count in composed command (expect 0)_
- Task 2.4 GREEN + qa pressure: _verdicts_
- Task 3.1 RED baseline: _verdict_
- Task 3.4 GREEN + no-match pressure: _verdicts_
- Task 3.5 trap gate AFTER: _verdict_
- Task 4.1 definition-note paths:
  - /Users/joe/.local/share/engram/vault/204.2026-07-10.work-kind-definition.md
  - /Users/joe/.local/share/engram/vault/205.2026-07-10.tier-definition.md
  - /Users/joe/.local/share/engram/vault/206.2026-07-10.outcome-definition.md
- Task 4.2 count outputs:
  - `engram count --group-by tags --filter tags=work-kind`: work-kind	1 / total: 1
  - `engram count --group-by tags --filter tags=tier`: tier	1 / total: 1
  - `engram count --group-by tags --filter tags=outcome`: outcome	1 / total: 1
- Task 4.2b Obsidian hand-check: DOCUMENTED-FOR-JOE. As of this run (2026-07-10): open vault (~/.local/share/engram/vault) in Obsidian → tag pane → expand work-kind / tier / outcome: each bare family tag shows exactly 1 note (the definition note written above). After evidence notes accrue, each family/value tag's pane count must equal the corresponding `engram count --group-by tags --filter tags=<family>/<value>` total. Caution: Obsidian tag pane aggregates nested counts into parent row (bare-family pane will read 1+N once family/value evidence exists), audit figures are family/value rows only.
- Task 4.3 recallability: PASS (grep returned 3)
- Task 5.2 count parity: outcome/pass	20 / tier/cheap	20 / work-kind/gauge-test	20 / total: 20
- Task 5.3 GAUGE: GAUGE-PASS (aggregate path 21.2026-07-10.route-evidence-gauge-test.md surfaced in query items + candidate_l2s; cleanup completed)
