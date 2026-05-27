# Project/Issue Metadata Fields Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional `--project <slug>` and `--issue <id>` write-side flags to `engram learn {fact,feedback,episode}` that render `project:` and `issue:` keys in note frontmatter, add an optional `--project <slug>` read-side filter to `engram query` that restricts emitted items to notes whose frontmatter `project:` matches, and update the `/learn` SKILL guidance plus README + GLOSSARY so project name becomes a queryable metadata field rather than a forbidden situation token.

**Architecture:** Two surface areas. (1) Write path: extend `CommonLearnArgs` with `Project` + `Issue` targ flags; carry through `LearnArgs`, each `*Fields` projection, and each `*FrontmatterDoc`; render with `omitempty` so existing call sites keep working. (2) Read path: add a `Project` field to `QueryArgs`; after items are merged but before rendering, drop items whose loaded content frontmatter doesn't carry a matching `project:` value. Filter at the items stage (per the issue text: "filter that restricts items") — the wikilink graph stays intact so cross-project bridges still bridge during BFS. Slug validation reuses the existing `slugPattern` (`^[a-z0-9-]+$`); `--issue` is validated as a non-empty, no-whitespace token.

**Tech Stack:** Go (pure, no CGO); `targ` for CLI; `go.yaml.in/yaml/v3` for frontmatter; `imptest` + `rapid` + `gomega` for tests; `targ` for build/test/check.

---

## File Structure

**Write path (touched together):**

- `internal/cli/targets.go` — add `Project` + `Issue` flags on `CommonLearnArgs`.
- `internal/cli/learn.go` — extend `LearnArgs`, the three `*Fields` projections, the three `*FrontmatterDoc` structs, `assembleLearnContent`, `runLearnFromFactArgs`, `runLearnFromFeedbackArgs`, `runLearnFromEpisodeArgs`, and `runLearnFromEpisodeArgsWithReader`. Add a `validateProjectSlug` helper that loud-rejects on bad shape, and a `validateIssueID` helper.
- `internal/cli/learn_test.go` — render-frontmatter tests for each note type covering set/unset combinations and YAML key ordering.
- `internal/cli/learn_adapters_test.go` — end-to-end `RunLearnFrom*Args` tests setting `Project` + `Issue` and asserting the on-disk file contains the right keys.

**Read path:**

- `internal/cli/query.go` — add `Project` field to `QueryArgs`; add `applyProjectFilter(items, project)` that scans each resolved item's content for `project: <slug>` in frontmatter and drops non-matches; call it after `aggregatePhraseSummaries` returns, before `renderQueryPayload`.
- `internal/cli/query_test.go` (or a new `query_project_filter_test.go`) — direct unit test on `applyProjectFilter` plus an end-to-end pipeline test asserting filtered items.
- `internal/cli/export_test.go` — export `applyProjectFilter` for direct testing.

**Docs:**

- `skills/learn/SKILL.md` — split the "no project names in situation" rule: situations stay retrieval-shaped (still no project token in `--situation`), project name belongs in `--project` metadata; add `--project` and `--issue` to the example invocations.
- `skills/learn/tests/` — RED behavioral test per writing-skills before the SKILL edit.
- `README.md` — mention `--project`/`--issue` in the relevant command snippet.
- `docs/GLOSSARY.md` — add `project` and `issue` frontmatter field definitions.

**Out of scope** (per issue, explicit):

- Adding `--issue` to `engram query` (write-side only).
- Backfilling existing notes with `project:` fields.
- Cross-project retrieval penalty scoring (Permanent/7c — separate concern).

---

## Task 1: Add `Project` + `Issue` flags to `CommonLearnArgs`

**Files:**
- Modify: `internal/cli/targets.go:16-24`

- [ ] **Step 1: Write the failing test** in `internal/cli/learn_adapters_test.go` (add to the existing file):

```go
func TestLearnFactArgs_AcceptsProjectAndIssueFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "with-project",
			Vault:    vault,
			Position: "top",
			Source:   "test",
			Project:  "engram",
			Issue:    "636",
		},
		Situation: "running tests",
		Subject:   "engram",
		Predicate: "supports",
		Object:    "project metadata",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))

	if len(entries) == 0 {
		return
	}

	body, readErr := os.ReadFile(filepath.Join(vault, "Permanent", entries[0].Name()))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(body)).To(ContainSubstring("project: engram\n"))
	g.Expect(string(body)).To(ContainSubstring("issue: \"636\"\n"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestLearnFactArgs_AcceptsProjectAndIssueFlags ./internal/cli/...`
Expected: FAIL — compile error on `cli.CommonLearnArgs{...Project, Issue...}` (unknown fields).

- [ ] **Step 3: Add the fields** in `internal/cli/targets.go`:

```go
// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Slug      string   `targ:"flag,name=slug,desc=kebab-case tag for the filename"`
	Vault     string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Target    string   `targ:"flag,name=target,desc=Luhmann ID this note relates to (empty for top-level)"`
	Position  string   `targ:"flag,name=position,desc=top|continuation|sibling"`
	Source    string   `targ:"flag,name=source,required,desc=provenance string for the source field (required)"`
	Relations []string `targ:"flag,name=relation,desc=related note as <wikilink-target>|<rationale> (repeatable)"`
	Project   string   `targ:"flag,name=project,desc=kebab-case project slug for cross-project filtering (optional)"`
	Issue     string   `targ:"flag,name=issue,desc=originating issue ID (optional)"`
}
```

- [ ] **Step 4: Run the test — it now fails further along** (LearnArgs / render path doesn't carry the fields yet). Expected: PASS only after Tasks 2-4 land. Defer commit.

Note: this task does not commit on its own — it is the test-and-flag-introduction step; Tasks 2-5 below complete the wire-through.

---

## Task 2: Thread `Project` + `Issue` through `LearnArgs` and the three `*Fields` projections

**Files:**
- Modify: `internal/cli/learn.go:23-50` (LearnArgs)
- Modify: `internal/cli/learn.go:109-118` (episodeFields)
- Modify: `internal/cli/learn.go:147-154` (factFields)
- Modify: `internal/cli/learn.go:169-176` (feedbackFields)
- Modify: `internal/cli/learn.go:206-234` (assembleLearnContent)
- Modify: `internal/cli/learn.go:278-309` (buildEpisodeFields)

- [ ] **Step 1: Extend `LearnArgs`** — add two trailing fields:

```go
type LearnArgs struct {
	Type     string
	Slug     string
	Vault    string
	Target   string
	Position string
	Source   string
	Project  string
	Issue    string

	Relations []string
	// ... rest unchanged
}
```

- [ ] **Step 2: Extend each `*Fields` projection** — add `Project string` and `Issue string` at the end of `feedbackFields`, `factFields`, `episodeFields`. Example for `factFields`:

```go
type factFields struct {
	Situation string
	Subject   string
	Predicate string
	Object    string
	Luhmann   string
	Source    string
	Project   string
	Issue     string
}
```

Mirror for `feedbackFields` and `episodeFields`.

- [ ] **Step 3: Thread through `assembleLearnContent`** at lines 211-222:

```go
case typeFeedback:
	f := feedbackFields{
		Situation: args.Situation, Behavior: args.Behavior, Impact: args.Impact,
		Action: args.Action, Luhmann: luhmann, Source: args.Source,
		Project: args.Project, Issue: args.Issue,
	}
	return renderFeedbackFrontmatter(f, when) + renderFeedbackBody(f, related), nil
case typeFact:
	f := factFields{
		Situation: args.Situation, Subject: args.Subject, Predicate: args.Predicate,
		Object: args.Object, Luhmann: luhmann, Source: args.Source,
		Project: args.Project, Issue: args.Issue,
	}
	return renderFactFrontmatter(f, when) + renderFactBody(f, related), nil
```

- [ ] **Step 4: Thread through `buildEpisodeFields`** — extend the returned struct:

```go
return episodeFields{
	Situation:         args.Situation,
	BoundaryRationale: args.BoundaryRationale,
	TranscriptText:    args.TranscriptText,
	Sessions:          sessions,
	TranscriptStart:   start,
	TranscriptEnd:     end,
	Luhmann:           luhmann,
	Source:            args.Source,
	Project:           args.Project,
	Issue:             args.Issue,
}, nil
```

- [ ] **Step 5: Verify it still compiles**

Run: `targ build`
Expected: build succeeds (rendering doesn't emit Project/Issue yet; tests from Task 1 still fail at the assertion, not at compile).

- [ ] **Step 6: Defer commit** — render layer comes next.

---

## Task 3: Render `project:` and `issue:` in frontmatter (`omitempty`)

**Files:**
- Modify: `internal/cli/learn.go:124-145` (episodeFrontmatterDoc + sub-docs)
- Modify: `internal/cli/learn.go:156-167` (factFrontmatterDoc)
- Modify: `internal/cli/learn.go:178-188` (feedbackFrontmatterDoc)
- Modify: `internal/cli/learn.go:474-490` (renderEpisodeFrontmatter)
- Modify: `internal/cli/learn.go:501-512` (renderFactFrontmatter)
- Modify: `internal/cli/learn.go:523-534` (renderFeedbackFrontmatter)

- [ ] **Step 1: Write the failing render test** at the end of `internal/cli/learn_test.go`:

```go
func TestRenderFactFrontmatter_EmitsProjectAndIssueBelowSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "subj", Predicate: "pred", Object: "obj",
		Luhmann: "1", Source: "src",
		Project: "engram", Issue: "636",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	g.Expect(got).To(ContainSubstring("source: src\n"))
	g.Expect(got).To(ContainSubstring("project: engram\n"))
	g.Expect(got).To(ContainSubstring("issue: \"636\"\n"))
	// Source must appear before project; project before issue.
	srcIdx := strings.Index(got, "source:")
	projIdx := strings.Index(got, "project:")
	issueIdx := strings.Index(got, "issue:")
	g.Expect(srcIdx).To(BeNumerically("<", projIdx))
	g.Expect(projIdx).To(BeNumerically("<", issueIdx))
}

func TestRenderFactFrontmatter_OmitsProjectAndIssueWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "subj", Predicate: "pred", Object: "obj",
		Luhmann: "1", Source: "src",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	g.Expect(got).NotTo(ContainSubstring("project:"))
	g.Expect(got).NotTo(ContainSubstring("issue:"))
}
```

Mirror these for feedback (using `ExportFeedbackFields` + `ExportRenderFeedbackFrontmatter`) and episode (will need a new exported render hook — see Task 3a).

- [ ] **Step 2: Run tests — verify they fail**

Run: `targ test -- -run 'TestRenderFactFrontmatter_(EmitsProjectAndIssueBelowSource|OmitsProjectAndIssueWhenEmpty)' ./internal/cli/...`
Expected: FAIL — substring `project:` not found in rendered output.

- [ ] **Step 3: Extend `factFrontmatterDoc`**:

```go
type factFrontmatterDoc struct {
	Type      string       `yaml:"type"`
	Situation string       `yaml:"situation"`
	Subject   string       `yaml:"subject"`
	Predicate string       `yaml:"predicate"`
	Object    string       `yaml:"object"`
	Luhmann   quotedString `yaml:"luhmann"`
	Created   string       `yaml:"created"`
	Source    string       `yaml:"source"`
	Project   string       `yaml:"project,omitempty"`
	Issue     quotedString `yaml:"issue,omitempty"`
}
```

`Issue` uses `quotedString` so numeric IDs like `636` render quoted (consistent with `luhmann` and avoiding YAML's numeric coercion on read-back). For `omitempty` to fire on `quotedString`, define `IsZero()` on it:

```go
// IsZero returns true when the underlying value is empty, so yaml.v3
// honors `omitempty` on quotedString fields.
func (q quotedString) IsZero() bool { return string(q) == "" }
```

Add this method once in `learn.go`.

- [ ] **Step 4: Mirror for `feedbackFrontmatterDoc`** — append the same two trailing fields with the same tags.

- [ ] **Step 5: Extend `episodeFrontmatterDoc`** — same trailing fields.

- [ ] **Step 6: Update `renderFactFrontmatter`** at lines 501-512:

```go
func renderFactFrontmatter(f factFields, when time.Time) string {
	return marshalFrontmatter(factFrontmatterDoc{
		Type:      "fact",
		Situation: f.Situation,
		Subject:   f.Subject,
		Predicate: f.Predicate,
		Object:    f.Object,
		Luhmann:   quotedString(f.Luhmann),
		Created:   when.Format(dateFormat),
		Source:    f.Source,
		Project:   f.Project,
		Issue:     quotedString(f.Issue),
	})
}
```

Mirror for `renderFeedbackFrontmatter` and `renderEpisodeFrontmatter`.

- [ ] **Step 7: Run all render tests**

Run: `targ test -- -run 'TestRenderFactFrontmatter|TestRenderFeedbackFrontmatter|TestRenderEpisodeFrontmatter' ./internal/cli/...`
Expected: PASS, including the existing roundtrip property test (the new optional fields don't break empty-value roundtrip because `omitempty` skips them on the empty input).

- [ ] **Step 8: Commit Tasks 1-3 together** (the wire-through is one logical change):

```bash
git add internal/cli/targets.go internal/cli/learn.go internal/cli/learn_test.go internal/cli/learn_adapters_test.go
git commit -m "feat(learn): add --project and --issue frontmatter metadata fields

Wire optional Project and Issue flags through CommonLearnArgs into the
three learn subcommands (fact/feedback/episode). Render project: and
issue: keys below source in frontmatter via yaml omitempty. Issue uses
quotedString so numeric IDs survive YAML round-trip without numeric
coercion.

Closes part of #636.

AI-Used: [claude]"
```

---

## Task 3a: Export episode render hook for tests

**Files:**
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1:** Add an export alias near line 31:

```go
ExportRenderEpisodeFrontmatter = renderEpisodeFrontmatter
```

and the type alias:

```go
type ExportEpisodeFields = episodeFields
```

- [ ] **Step 2: Add episode-side render tests** mirroring the fact tests in Task 3 Step 1 (`TestRenderEpisodeFrontmatter_EmitsProjectAndIssueBelowSource` and `_OmitsProjectAndIssueWhenEmpty`). Episode frontmatter is more complex (nested provenance) — assert the same substring + ordering invariants.

- [ ] **Step 3: Run, commit** with Task 3.

---

## Task 4: Validate slug shape, loud-reject invalid `--project` / `--issue`

**Files:**
- Modify: `internal/cli/learn.go` (add `validateProjectSlug`, `validateIssueID`)
- Modify: `internal/cli/learn.go:651-679` (runLearn — call validators)
- Modify: `internal/cli/learn_test.go` (failing tests)
- Modify: `internal/cli/export_test.go` (export validators)

- [ ] **Step 1: Write failing tests**:

```go
func TestRunLearn_RejectsInvalidProjectSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug: "ok-slug", Vault: vault, Position: "top", Source: "src",
			Project: "Engram!", // capitals + punctuation — invalid kebab-case
		},
		Situation: "s", Subject: "x", Predicate: "y", Object: "z",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)
	g.Expect(err).To(MatchError(ContainSubstring("project")))
	g.Expect(err).To(MatchError(ContainSubstring("Engram!")))
}

func TestRunLearn_RejectsIssueWithWhitespace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug: "ok-slug", Vault: vault, Position: "top", Source: "src",
			Issue: "636 with spaces",
		},
		Situation: "s", Subject: "x", Predicate: "y", Object: "z",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)
	g.Expect(err).To(MatchError(ContainSubstring("issue")))
}
```

- [ ] **Step 2: Run — verify FAIL** (no validation yet → notes get written).

- [ ] **Step 3: Add validators** in `learn.go` after `validateSlug`:

```go
var (
	errProjectSlugInvalid = errors.New("project slug must match [a-z0-9-]+")
	errIssueIDInvalid     = errors.New("issue must be non-empty with no whitespace")
)

// validateProjectSlug returns an error if slug is non-empty and does not
// match the kebab-case shape. An empty slug is allowed (project is optional).
func validateProjectSlug(slug string) error {
	if slug == "" {
		return nil
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: got %q", errProjectSlugInvalid, slug)
	}
	return nil
}

// validateIssueID rejects whitespace or empty-but-only-whitespace IDs.
// An empty issue is allowed (issue is optional).
func validateIssueID(id string) error {
	if id == "" {
		return nil
	}
	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("%w: got %q", errIssueIDInvalid, id)
	}
	return nil
}
```

- [ ] **Step 4: Wire into `runLearn`** at the top, just after `validateSlug`:

```go
func runLearn(ctx context.Context, args LearnArgs, deps LearnDeps, stdout io.Writer) error {
	if slugErr := validateSlug(args.Slug); slugErr != nil {
		return fmt.Errorf("learn: %w", slugErr)
	}
	if projErr := validateProjectSlug(args.Project); projErr != nil {
		return fmt.Errorf("learn: %w", projErr)
	}
	if issueErr := validateIssueID(args.Issue); issueErr != nil {
		return fmt.Errorf("learn: %w", issueErr)
	}
	// ... rest unchanged
}
```

- [ ] **Step 5: Export validators** in `export_test.go`:

```go
ExportValidateProjectSlug = validateProjectSlug
ExportValidateIssueID     = validateIssueID
```

- [ ] **Step 6: Run all learn tests**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 7: Commit**:

```bash
git add internal/cli/learn.go internal/cli/learn_test.go internal/cli/export_test.go
git commit -m "feat(learn): loud-reject invalid --project / --issue values

Reject project slugs that don't match [a-z0-9-]+ and issue IDs that
contain whitespace. Empty values pass through (both flags are optional).

AI-Used: [claude]"
```

---

## Task 5: Thread `Project` + `Issue` through `runLearnFrom*Args` adapters

**Files:**
- Modify: `internal/cli/learn.go:681-727` (episode)
- Modify: `internal/cli/learn.go:729-745` (fact)
- Modify: `internal/cli/learn.go:747-763` (feedback)

- [ ] **Step 1:** In `runLearnFromFactArgs`, extend the `LearnArgs` literal:

```go
return runLearn(ctx, LearnArgs{
	Type:      typeFact,
	Slug:      a.Slug,
	Vault:     a.Vault,
	Target:    a.Target,
	Position:  a.Position,
	Source:    a.Source,
	Project:   a.Project,
	Issue:     a.Issue,
	Relations: a.Relations,
	Situation: a.Situation,
	Subject:   a.Subject,
	Predicate: a.Predicate,
	Object:    a.Object,
}, deps, stdout)
```

Mirror in `runLearnFromFeedbackArgs` and in `runLearnFromEpisodeArgsWithReader`'s `LearnArgs{...}` literal.

- [ ] **Step 2: Run the Task 1 acceptance test** — it should now PASS end-to-end:

Run: `targ test -- -run TestLearnFactArgs_AcceptsProjectAndIssueFlags ./internal/cli/...`
Expected: PASS.

- [ ] **Step 3: Commit**:

```bash
git add internal/cli/learn.go
git commit -m "feat(learn): forward Project/Issue from adapter args into LearnArgs

Closes the write-side wire-through for #636 — the three learn
subcommand adapters now carry --project and --issue values into the
shared runLearn path. End-to-end test passes.

AI-Used: [claude]"
```

---

## Task 6: Add `--project` filter to `engram query`

**Files:**
- Modify: `internal/cli/query.go:22-28` (QueryArgs)
- Modify: `internal/cli/query.go:42-83` (RunQuery — apply filter)
- Modify: `internal/cli/query.go` (add `applyProjectFilter` helper)
- Modify: `internal/cli/export_test.go` (export helper)
- Modify: `internal/cli/query_test.go` (or new `query_project_filter_test.go`)

- [ ] **Step 1: Write the failing helper unit test**:

```go
func TestApplyProjectFilter_DropsNonMatching(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		{NotePath: "Permanent/a.md", Content: "---\ntype: fact\nproject: engram\n---\nbody"},
		{NotePath: "Permanent/b.md", Content: "---\ntype: fact\nproject: other\n---\nbody"},
		{NotePath: "Permanent/c.md", Content: "---\ntype: fact\n---\nbody"}, // no project field
	}

	filtered := cli.ExportApplyProjectFilter(items, "engram")
	g.Expect(filtered).To(HaveLen(1))
	g.Expect(filtered[0].NotePath).To(Equal("Permanent/a.md"))
}

func TestApplyProjectFilter_NoFilterReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		{NotePath: "Permanent/a.md", Content: "---\ntype: fact\nproject: engram\n---\nbody"},
		{NotePath: "Permanent/b.md", Content: "---\ntype: fact\n---\nbody"},
	}

	filtered := cli.ExportApplyProjectFilter(items, "")
	g.Expect(filtered).To(HaveLen(2))
}
```

- [ ] **Step 2: Run — verify FAIL** (helper does not exist).

Run: `targ test -- -run TestApplyProjectFilter ./internal/cli/...`
Expected: compile error.

- [ ] **Step 3: Add `--project` to QueryArgs** in `query.go:22-28`:

```go
type QueryArgs struct {
	Query     string   `targ:"positional,name=query,desc=natural-language query string"`
	Phrases   []string `targ:"flag,name=phrase,desc=query phrase (repeatable; use instead of positional for multi-phrase)"`
	VaultPath string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
	Limit     int      `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
	Project   string   `targ:"flag,name=project,desc=restrict items to notes with matching project: frontmatter field (optional)"`
}
```

- [ ] **Step 4: Add the filter helper** in `query.go`:

```go
// projectLineRE matches a `project: <slug>` line in YAML frontmatter,
// permitting optional surrounding whitespace and an optional trailing
// comment. Anchored to start-of-line (multiline mode) so we don't match
// "project:" inside the note body.
var projectLineRE = regexp.MustCompile(`(?m)^project:\s*([a-z0-9-]+)\s*$`)

// applyProjectFilter returns items whose frontmatter declares the given
// project. An empty project string is a no-op (returns items unchanged).
// Items with no content (e.g., hubs with elided body) are dropped when
// a non-empty project is specified — we can't verify a match without the
// frontmatter.
func applyProjectFilter(items []resolvedItem, project string) []resolvedItem {
	if project == "" {
		return items
	}
	out := make([]resolvedItem, 0, len(items))
	for _, item := range items {
		if itemMatchesProject(item, project) {
			out = append(out, item)
		}
	}
	return out
}

func itemMatchesProject(item resolvedItem, project string) bool {
	if item.content == "" {
		return false
	}
	// Restrict the scan to the frontmatter block so body text can't
	// false-match. Frontmatter is delimited by leading "---\n" and a
	// terminating "\n---\n".
	const delim = "---\n"
	body := strings.TrimPrefix(item.content, delim)
	end := strings.Index(body, "\n"+delim)
	if end < 0 {
		return false
	}
	front := body[:end+1]
	match := projectLineRE.FindStringSubmatch(front)
	return len(match) == 2 && match[1] == project
}
```

- [ ] **Step 5: Wire into RunQuery** — call after aggregation, before render. Replace the tail of `RunQuery`:

```go
merged := aggregatePhraseSummaries(phrases, summaries, limit)
merged.resolvedItems = applyProjectFilter(merged.resolvedItems, args.Project)

return renderQueryPayload(stdout, merged)
```

- [ ] **Step 6: Export for test access** in `export_test.go`:

```go
ExportApplyProjectFilter = applyProjectFilter

type ExportResolvedItem = resolvedItem
```

Also add accessor exports (Go's field visibility — `resolvedItem` fields are unexported). Add accessor methods or change the test to construct via the exported alias plus a helper:

```go
// ExportNewResolvedItem builds a resolvedItem for tests; lets the cli_test
// package set unexported fields without granting field-by-field access.
func ExportNewResolvedItem(notePath, content string) ExportResolvedItem {
	return ExportResolvedItem{notePath: notePath, content: content}
}
```

Adjust the Step 1 test to call `cli.ExportNewResolvedItem(...)` and to read result paths via a small `ExportResolvedItemPath(item) string` accessor.

- [ ] **Step 7: Run all query tests**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 8: Commit**:

```bash
git add internal/cli/query.go internal/cli/export_test.go internal/cli/query_test.go
git commit -m "feat(query): add --project filter that drops items without matching frontmatter

Filter runs after items merge, before payload render. Empty --project is
a no-op. Items with elided content are dropped when --project is set —
we cannot verify a match without frontmatter. Wikilink graph is
unaffected (cross-project bridges still bridge during BFS).

Closes part of #636.

AI-Used: [claude]"
```

---

## Task 7: Pipeline integration test for `--project` filter

**Files:**
- Modify: `internal/cli/query_integration_test.go` (or `query_test.go` — match the existing convention)

- [ ] **Step 1: Write the end-to-end test** using the existing fixture/embedder pattern in `query_test.go`. Build a fixture vault with two facts: one with `project: engram`, one with `project: opencode`, both with sidecars compatible with the test embedder. Call `RunQuery` with `Project: "engram"` and assert the rendered payload's `items` list contains only the engram note path.

```go
func TestRunQuery_ProjectFilterRestrictsItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Setup pattern follows existing query_test.go fixtures —
	// see TestRunQuery_* tests above for the deps construction pattern.
	vault := t.TempDir()
	// ... write two .md + .vec.json pairs with project: engram and project: opencode ...

	var buf bytes.Buffer
	deps := cli.QueryDeps{ /* ... existing pattern ... */ }
	err := cli.RunQuery(context.Background(), cli.QueryArgs{
		Query:     "anything",
		VaultPath: vault,
		Limit:     20,
		Project:   "engram",
	}, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(buf.String()).To(ContainSubstring("Permanent/engram-note.md"))
	g.Expect(buf.String()).NotTo(ContainSubstring("Permanent/opencode-note.md"))
}
```

(Engineer: open `internal/cli/query_test.go` and copy the closest existing `TestRunQuery_*` test as a starting template — match its embedder + vault-fixture construction exactly.)

- [ ] **Step 2: Run — verify PASS**

Run: `targ test -- -run TestRunQuery_ProjectFilterRestrictsItems ./internal/cli/...`
Expected: PASS.

- [ ] **Step 3: Commit**:

```bash
git add internal/cli/query_test.go
git commit -m "test(query): cover --project filter end-to-end through RunQuery

AI-Used: [claude]"
```

---

## Task 8: Full check-full before doc work

- [ ] **Step 1: Run the full check**

Run: `targ check-full`
Expected: all green. If lint fires, address every reported issue in one commit before moving on — don't play whack-a-mole.

- [ ] **Step 2:** If anything failed, fix in one pass, commit, re-run.

---

## Task 9: Update `/learn` SKILL.md via writing-skills

**Files:**
- Modify: `skills/learn/SKILL.md`
- Test: `skills/learn/tests/`

Per `engram/CLAUDE.md`: SKILL.md edits MUST use `superpowers:writing-skills` with TDD discipline (RED baseline behavior test → GREEN edit → REFACTOR + pressure tests).

- [ ] **Step 1: Invoke `superpowers:writing-skills`** with the directive: "Update `skills/learn/SKILL.md` to relax the no-project-names rule. Project name belongs in `--project` metadata; situation phrasing still stays retrieval-shaped (still no project token in `--situation`). Touch every place the old rule appears."

- [ ] **Step 2: Specific edits** (the writing-skills skill will drive these via TDD):

  1. Insert into the **What to write** or workflow section, near the situation-shaping guidance: a paragraph stating that project name belongs in `--project` metadata, not in `--situation`, and that `--issue` records the originating issue ID.
  2. Update the example `engram learn fact`, `engram learn feedback`, and `engram learn episode` invocations (lines ~173-225) to show `--project engram` and `--issue 636` placement.
  3. Touch the four rule-statement sites:
     - Line ~120: "drop. Either the lesson is too project-specific..." — restate as: project-specific lessons are still candidates, but they must use `--project` to mark project boundedness; only drop if the lesson is genuinely too narrow to be useful even within the project.
     - Line ~126: "Situation names this project, this file, this issue, or today's date" — restate: situation still must not name the project (retrieval shape); use `--project` for that.
     - Line ~133: "no project names, no hindsight, must be a principle" — restate: no project names in `--situation`; project names go in `--project`.
     - Lines ~276, ~291: "Project-specific knowledge doesn't belong in the vault" → soften to: "Project-specific knowledge belongs in the vault when tagged via `--project`; situation phrasing still stays retrieval-shaped."

- [ ] **Step 3: Run writing-skills pressure tests** as that skill prescribes.

- [ ] **Step 4: Commit**:

```bash
git add skills/learn/SKILL.md skills/learn/tests/
git commit -m "docs(learn): split no-project-names rule across situation vs metadata

Project name belongs in --project metadata; situation phrasing still
stays retrieval-shaped (still no project token in --situation). Updated
example invocations and the four rule-statement sites accordingly.

Closes part of #636.

AI-Used: [claude]"
```

---

## Task 10: Update README + GLOSSARY

**Files:**
- Modify: `README.md`
- Modify: `docs/GLOSSARY.md`

- [ ] **Step 1: Update README** — find the section that lists `engram learn` and `engram query` flags (likely a code block or short flag table). Add `--project <slug>` and `--issue <id>` to the learn flag list, and `--project <slug>` to the query flag list. Mention in one sentence that `--project` is the metadata field used by `engram query --project` to filter cross-project queries.

- [ ] **Step 2: Update `docs/GLOSSARY.md`** — add two entries (alphabetical placement under whatever convention the file uses):

```markdown
### project (frontmatter field)

Optional kebab-case slug naming the project a note belongs to. Set on
write via `engram learn {fact,feedback,episode} --project <slug>`.
Queryable via `engram query --project <slug>` to restrict results to
notes from a single project. Absent on notes that capture universal
principles.

### issue (frontmatter field)

Optional identifier for the originating GitHub/Jira/etc. issue. Set on
write via `engram learn {fact,feedback,episode} --issue <id>`. Free-form
non-whitespace string (e.g., `636`, `GH-636`, `PROJ-1234`). Recorded for
provenance; no read-side filter.
```

- [ ] **Step 3: Sanity-check**

Run: `grep -n "project\|issue" README.md docs/GLOSSARY.md | head -20`
Expected: both new mentions present.

- [ ] **Step 4: Commit**:

```bash
git add README.md docs/GLOSSARY.md
git commit -m "docs: document new project/issue frontmatter fields and query filter

AI-Used: [claude]"
```

---

## Task 11: Final verification

- [ ] **Step 1: Run the full check** one more time:

Run: `targ check-full`
Expected: all green.

- [ ] **Step 2: Exercise the binary end-to-end** to verify wiring (per Joe's "passing tests ≠ usable system" rule):

```bash
# Build
targ build

# Use a throwaway vault so we don't pollute the real one
export ENGRAM_VAULT_PATH=$(mktemp -d)/v

# Write a fact with project + issue
./engram learn fact \
  --slug verify-636 \
  --position top \
  --source "session log engram, 2026-05-26, context: #636 verification" \
  --project engram \
  --issue 636 \
  --situation "When verifying the engram project metadata feature" \
  --subject "engram learn" \
  --predicate "now supports" \
  --object "--project and --issue flags"

# Inspect the on-disk file
cat "$ENGRAM_VAULT_PATH/Permanent/"*.md | head -20
# Expect: project: engram  and  issue: "636"  lines below source:

# Query with the filter
./engram query --project engram "metadata" 2>&1 | head -40
# Expect: items[] contains the verify-636 note; phrases shows the query
```

- [ ] **Step 3: Close issue**

Run: `gh issue close 636 -c "Implemented. --project and --issue flags added to engram learn fact|feedback|episode; --project filter added to engram query; /learn SKILL guidance, README, and GLOSSARY updated."`

- [ ] **Step 4: Delete the plan doc** (per `please` step 6 — temporary planning artifact):

```bash
git rm docs/superpowers/plans/2026-05-26-project-issue-metadata-fields.md
git commit -m "chore: remove completed plan doc for #636

AI-Used: [claude]"
```

---

## Self-Review

**1. Spec coverage** — issue text checked section by section:
- ✓ `--project <slug>` + `--issue <id>` flags on learn fact|feedback|episode (Tasks 1-5)
- ✓ Slug pattern kebab-case (Task 4, reuses `slugPattern`)
- ✓ Frontmatter `project:` (optionally `issue:`) below `source` (Task 3)
- ✓ `/learn` SKILL guidance updated (Task 9)
- ✓ `engram query --project <slug>` filter (Task 6)
- ✓ README + GLOSSARY surfaced (Task 10)
- ✓ Out of scope: no `--issue` on query (per advisor + plain reading of issue spec)

**2. Placeholder scan** — none found.

**3. Type consistency** — `Project string` / `Issue string` used everywhere on the Go side; frontmatter keys `project`/`issue` lowercase; `Issue` is `quotedString` end-to-end with `IsZero()` defined.

One known caveat noted in Task 6 Step 6: `resolvedItem` fields are unexported, so the cli_test package needs an `ExportNewResolvedItem` constructor + a tiny accessor for `notePath`. Step 6 already covers this.
