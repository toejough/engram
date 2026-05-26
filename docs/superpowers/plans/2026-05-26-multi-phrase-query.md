# Multi-Phrase Query Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `engram query` to accept multiple `--phrase` flags and aggregate results server-side (dedup items, per-phrase clusters, union hubs), then update `/recall` SKILL.md to use a single invocation.

**Architecture:** Add a `Phrases []string` flag to `QueryArgs` (backward-compat: existing positional `Query string` still works as a single phrase). Extract the current single-phrase pipeline into `runSinglePhraseQuery`; loop over all phrases in `RunQuery` and aggregate with `aggregatePhraseSummaries`. Payload gains a `phrases []string` top-level field (replacing `query string`), a `phrase string` field on each cluster, and `phrases_queried: N` in budget.

**Tech Stack:** Go, `targ` (build/test), `go.yaml.in/yaml/v3` (YAML encoding), `gomega` (test assertions).

---

## Files

- **Modify:** `internal/cli/query.go` — all pipeline and type changes
- **Modify:** `internal/cli/query_test.go` — update `QueryArgs{Query:…}` usages and add multi-phrase tests
- **Modify:** `internal/cli/query_subgraph_test.go` — update `queryParsed` struct (`.Query` → `.Phrases`)
- **Modify:** `internal/cli/query_pipeline_test.go` — update `QueryArgs{Query:…}` usages
- **Modify:** `internal/cli/query_property_test.go` — update `QueryArgs{Query:…}` usages
- **Modify:** `internal/cli/query_integration_test.go` — update positional-arg invocation if needed
- **Modify:** `skills/recall/SKILL.md` — replace step 3 agent-side union prose with single-invocation shape; use `superpowers:writing-skills` skill

---

## Task 1: Add `Phrases []string` flag; resolve effective phrases in `RunQuery`

**Files:**
- Modify: `internal/cli/query.go` (lines ~22-91)
- Modify: `internal/cli/query_test.go`

- [ ] **Step 1: Write failing tests**

In `internal/cli/query_test.go`, add:

```go
func TestQuery_PhrasesFlag_AcceptsMultiplePhrases(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    vault := t.TempDir()
    memFS := newInMemoryFS()
    plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
        "---\ntype: fact\n---\nbody\n")

    var out bytes.Buffer

    err := cli.RunQuery(context.Background(),
        cli.QueryArgs{Phrases: []string{"body", "fact"}, VaultPath: vault},
        newQueryDeps(memFS), &out)

    g.Expect(err).NotTo(HaveOccurred())

    var parsed struct {
        Phrases []string `yaml:"phrases"`
    }
    g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
    g.Expect(parsed.Phrases).To(ConsistOf("body", "fact"))
}

func TestQuery_EmptyPhrasesAndEmptyQuery_ReturnsError(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    vault := t.TempDir()
    memFS := newInMemoryFS()
    var out bytes.Buffer

    err := cli.RunQuery(context.Background(),
        cli.QueryArgs{Phrases: []string{}, VaultPath: vault},
        newQueryDeps(memFS), &out)

    g.Expect(err).To(MatchError(ContainSubstring("empty query")))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test 2>&1 | grep -A3 "PhrasesFlag\|EmptyPhrasesAndEmpty"
```

Expected: FAIL — `cli.QueryArgs` has no `Phrases` field.

- [ ] **Step 3: Add `Phrases` to `QueryArgs` and resolve effective phrases**

In `internal/cli/query.go`, update `QueryArgs`:

```go
// QueryArgs holds parsed flags for `engram query`.
type QueryArgs struct {
    Query     string   `targ:"positional,name=query,desc=natural-language query string"`
    Phrases   []string `targ:"flag,name=phrase,desc=query phrase (repeatable; use instead of positional for multi-phrase)"`
    VaultPath string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
    Limit     int      `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
}
```

At the top of `RunQuery`, replace:

```go
if args.Query == "" {
    return errQueryEmptyString
}
```

with:

```go
phrases := args.Phrases
if len(phrases) == 0 {
    if args.Query == "" {
        return errQueryEmptyString
    }
    phrases = []string{args.Query}
}
```

Do not change anything else yet — the rest of `RunQuery` still uses `args.Query` for the single phrase. Pass `phrases[0]` wherever `args.Query` was used for now (temporary, replaced fully in Task 2).

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test 2>&1 | grep -E "PASS|FAIL|PhrasesFlag|EmptyPhrasesAndEmpty"
```

Expected: both new tests PASS; existing tests still PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/query.go internal/cli/query_test.go
git commit -m "$(cat <<'EOF'
feat(query): add --phrase repeatable flag; resolve effective phrases

Phrases []string accepts multiple --phrase flags. When empty, falls back
to the existing positional Query string for backward compat. Empty both
→ errQueryEmptyString as before.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: Extract `runSinglePhraseQuery`; loop + aggregate in `RunQuery`

**Files:**
- Modify: `internal/cli/query.go`
- Modify: `internal/cli/query_test.go`

- [ ] **Step 1: Write failing tests for multi-phrase aggregation**

In `internal/cli/query_test.go`, add:

```go
func TestQuery_MultiPhrase_DeduplicatesItemsByPath(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    vault := t.TempDir()
    memFS := newInMemoryFS()
    plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
        "---\ntype: fact\n---\nbody of note one\n")

    var out bytes.Buffer

    // Two identical phrases — same note matched by both
    err := cli.RunQuery(context.Background(),
        cli.QueryArgs{Phrases: []string{"body of note one", "body of note one"}, VaultPath: vault},
        newQueryDeps(memFS), &out)

    g.Expect(err).NotTo(HaveOccurred())

    var parsed struct {
        Items []struct {
            Path string `yaml:"path"`
        } `yaml:"items"`
    }
    g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

    paths := make([]string, 0, len(parsed.Items))
    for _, item := range parsed.Items {
        paths = append(paths, item.Path)
    }

    // Each path must appear at most once
    seen := map[string]int{}
    for _, p := range paths {
        seen[p]++
    }
    for path, count := range seen {
        g.Expect(count).To(Equal(1), "path %s appeared %d times", path, count)
    }
}

func TestQuery_MultiPhrase_MaxScoreAcrossPhrases(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    vault := t.TempDir()
    memFS := newInMemoryFS()
    // One note that scores differently for two phrases (embedder is cosine;
    // fake embedder returns the query string's hash-based vector so we can
    // control relative scores by using the note body as one phrase)
    plantNoteWithSidecar(t, memFS, vault, "Permanent/1.body.md",
        "---\ntype: fact\n---\nbody\n")

    var outSingle bytes.Buffer
    _ = cli.RunQuery(context.Background(),
        cli.QueryArgs{Query: "body", VaultPath: vault},
        newQueryDeps(memFS), &outSingle)

    var parsedSingle struct {
        Items []struct {
            Path  string  `yaml:"path"`
            Score float32 `yaml:"score"`
        } `yaml:"items"`
    }
    _ = yaml.Unmarshal(outSingle.Bytes(), &parsedSingle)
    if len(parsedSingle.Items) == 0 {
        t.Skip("no items returned; skip score comparison")
    }
    singleScore := parsedSingle.Items[0].Score

    var outMulti bytes.Buffer
    _ = cli.RunQuery(context.Background(),
        cli.QueryArgs{Phrases: []string{"body", "xyzzy"}, VaultPath: vault},
        newQueryDeps(memFS), &outMulti)

    var parsedMulti struct {
        Items []struct {
            Path  string  `yaml:"path"`
            Score float32 `yaml:"score"`
        } `yaml:"items"`
    }
    _ = yaml.Unmarshal(outMulti.Bytes(), &parsedMulti)
    if len(parsedMulti.Items) == 0 {
        t.Skip("no items in multi result; skip score comparison")
    }

    // Score in multi-phrase result must be >= single-phrase score (max, not avg)
    g.Expect(parsedMulti.Items[0].Score).To(BeNumerically(">=", singleScore))
}

func TestQuery_MultiPhrase_ClustersTaggedWithPhrase(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    vault := t.TempDir()
    memFS := newInMemoryFS()
    // Plant enough notes to trigger clustering (≥6 per phrase)
    for i := range 12 {
        plantNoteWithSidecar(t, memFS, vault,
            fmt.Sprintf("Permanent/%d.note.md", i+1),
            fmt.Sprintf("---\ntype: fact\n---\nbody %d\n", i))
    }

    var out bytes.Buffer
    err := cli.RunQuery(context.Background(),
        cli.QueryArgs{Phrases: []string{"body", "fact"}, VaultPath: vault},
        newQueryDeps(memFS), &out)

    g.Expect(err).NotTo(HaveOccurred())

    var parsed struct {
        Clusters []struct {
            Phrase string `yaml:"phrase"`
            ID     int    `yaml:"id"`
        } `yaml:"clusters"`
    }
    g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

    for _, cluster := range parsed.Clusters {
        g.Expect(cluster.Phrase).NotTo(BeEmpty(), "cluster id=%d has no phrase label", cluster.ID)
    }
}

func TestQuery_MultiPhrase_BudgetHasPhrasesQueried(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    vault := t.TempDir()
    memFS := newInMemoryFS()
    plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
        "---\ntype: fact\n---\nbody\n")

    var out bytes.Buffer
    err := cli.RunQuery(context.Background(),
        cli.QueryArgs{Phrases: []string{"body", "fact", "something"}, VaultPath: vault},
        newQueryDeps(memFS), &out)

    g.Expect(err).NotTo(HaveOccurred())

    var parsed struct {
        Budget struct {
            PhrasesQueried int `yaml:"phrases_queried"`
        } `yaml:"budget"`
    }
    g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
    g.Expect(parsed.Budget.PhrasesQueried).To(Equal(3))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test 2>&1 | grep -E "FAIL|MultiPhrase"
```

Expected: all four new tests FAIL — no multi-phrase aggregation yet.

- [ ] **Step 3: Extract `runSinglePhraseQuery` from `RunQuery`**

In `internal/cli/query.go`, add a new unexported function. Move the embedding + ranking + subgraph + cluster + hub + merge logic out of `RunQuery` into:

```go
// runSinglePhraseQuery runs the full per-phrase pipeline for one phrase
// and returns a queryPipelineSummary. notes and hits are already loaded
// (shared across all phrases in a multi-phrase run).
func runSinglePhraseQuery(
    ctx context.Context,
    phrase string,
    notes []vaultgraph.Note,
    hits []compatibleSidecar,
    vault string,
    limit int,
    deps QueryDeps,
) (queryPipelineSummary, error) {
    queryVec, qErr := deps.Embedder.Embed(ctx, phrase)
    if qErr != nil {
        return queryPipelineSummary{}, fmt.Errorf("query: embed: %w", qErr)
    }

    directHits := rankCandidates(hits, vault, deps.Read, queryVec)
    if len(directHits) > limit {
        directHits = directHits[:limit]
    }

    subgraph := expandSubgraph(notes, hits, directHits, vault, deps.Read, queryVec)
    clusters := clusterSubgraph(subgraph, phrase)
    hubs := identifyHubs(subgraph)
    resolved := mergeProvenances(directHits, subgraph, clusters, hubs)

    return queryPipelineSummary{
        directHits:     directHits,
        subgraph:       subgraph,
        clusters:       clusters,
        hubs:           hubs,
        resolvedItems:  resolved,
        totalNotes:     len(notes),
        withEmbeddings: len(hits),
        limit:          limit,
    }, nil
}
```

- [ ] **Step 4: Add `phrasedCluster` type and `aggregatePhraseSummaries` function**

In `internal/cli/query.go`, add after the existing type block:

```go
// phrasedCluster pairs a cluster report with the phrase that produced it,
// so the payload can tag each cluster with its originating query phrase.
type phrasedCluster struct {
    phrase   string
    report   clusterReport
    subgraph expandedSubgraph
}

// aggregatedSummary holds the merged result of running RunQuery across
// multiple phrases.
type aggregatedSummary struct {
    phrases        []string
    resolvedItems  []resolvedItem
    phraseClusters []phrasedCluster
    totalNotes     int
    withEmbeddings int
    limit          int
    subgraphSize   int
    subgraphCapped bool
    hopsTraversed  int
}
```

Then add the aggregation function:

```go
// aggregatePhraseSummaries merges per-phrase pipeline results into a single
// aggregatedSummary per the issue-639 spec:
//   - items: dedup by path, max score across phrases, union provenances,
//     max in_degree; cluster_id cleared (clusters are per-phrase).
//   - clusters: retained per-phrase, tagged with their phrase.
//   - budget: subgraphSize is sum, hopsTraversed is max, capped is OR.
func aggregatePhraseSummaries(phrases []string, summaries []queryPipelineSummary, limit int) aggregatedSummary {
    byPath := make(map[string]*resolvedItem, len(summaries)*limit)

    for _, s := range summaries {
        for i := range s.resolvedItems {
            src := &s.resolvedItems[i]
            existing, ok := byPath[src.notePath]
            if !ok {
                c := *src
                c.clusterID = nil // cluster IDs are per-phrase; drop from merged items
                byPath[src.notePath] = &c
                continue
            }
            if src.score > existing.score {
                existing.score = src.score
                existing.content = src.content
            }
            for _, p := range src.provenances {
                appendUniqueProvenance(existing, p)
            }
            if src.inDegree != nil {
                if existing.inDegree == nil || *src.inDegree > *existing.inDegree {
                    v := *src.inDegree
                    existing.inDegree = &v
                }
            }
        }
    }

    paths := make([]string, 0, len(byPath))
    for path := range byPath {
        paths = append(paths, path)
    }
    sort.Strings(paths)

    items := make([]resolvedItem, 0, len(byPath))
    for _, path := range paths {
        items = append(items, *byPath[path])
    }
    sort.SliceStable(items, func(i, j int) bool {
        return resolvedItemLess(items[i], items[j])
    })

    phraseClusters := make([]phrasedCluster, 0, len(summaries))
    for i, s := range summaries {
        phraseClusters = append(phraseClusters, phrasedCluster{
            phrase:   phrases[i],
            report:   s.clusters,
            subgraph: s.subgraph,
        })
    }

    totalSubgraph := 0
    capped := false
    maxHops := 0
    for _, s := range summaries {
        totalSubgraph += len(s.subgraph.members)
        if s.subgraph.capped {
            capped = true
        }
        if s.subgraph.hopsTraversed > maxHops {
            maxHops = s.subgraph.hopsTraversed
        }
    }

    first := summaries[0]

    return aggregatedSummary{
        phrases:        phrases,
        resolvedItems:  items,
        phraseClusters: phraseClusters,
        totalNotes:     first.totalNotes,
        withEmbeddings: first.withEmbeddings,
        limit:          limit,
        subgraphSize:   totalSubgraph,
        subgraphCapped: capped,
        hopsTraversed:  maxHops,
    }
}
```

- [ ] **Step 5: Update `RunQuery` to use the new helpers**

Replace the body of `RunQuery` (after the phrases resolution from Task 1) with:

```go
limit := args.Limit
if limit == 0 {
    limit = defaultQueryLimit
}

notes, scanErr := deps.Scan(args.VaultPath)
if scanErr != nil {
    return fmt.Errorf("query: scan: %w", scanErr)
}

modelID := deps.Embedder.ModelID()
hits := loadCompatibleSidecars(notes, args.VaultPath, deps.Read, modelID)

if len(notes) > 0 && len(hits) == 0 {
    return errQueryNoEmbeddings
}

summaries := make([]queryPipelineSummary, 0, len(phrases))
for _, phrase := range phrases {
    summary, err := runSinglePhraseQuery(ctx, phrase, notes, hits, args.VaultPath, limit, deps)
    if err != nil {
        return err
    }
    summaries = append(summaries, summary)
}

merged := aggregatePhraseSummaries(phrases, summaries, limit)

return renderQueryPayload(stdout, merged)
```

- [ ] **Step 6: Run tests**

```bash
targ test 2>&1 | tail -20
```

Expected: compile errors from `renderQueryPayload` signature mismatch — fix in next task.

---

## Task 3: Update payload types and rendering for multi-phrase shape

**Files:**
- Modify: `internal/cli/query.go`
- Modify: `internal/cli/query_subgraph_test.go` (update `queryParsed`)
- Modify: `internal/cli/query_pipeline_test.go` (update `QueryArgs{Query:…}` → `QueryArgs{Phrases:…}` or keep Query, also update `queryParsed.Query` references)

- [ ] **Step 1: Update payload types**

In `internal/cli/query.go`, change:

```go
// queryBudget — add PhrasesQueried field
type queryBudget struct {
    PhrasesQueried       int  `yaml:"phrases_queried"`  // NEW — always emitted
    TotalNotes           int  `yaml:"total_notes"`
    WithEmbeddings       int  `yaml:"with_embeddings"`
    SubgraphSize         int  `yaml:"subgraph_size"`
    SubgraphSizeCapped   bool `yaml:"subgraph_size_capped"`
    HopsTraversed        int  `yaml:"hops_traversed"`
    ClustersFound        int  `yaml:"clusters_found"`
    HubsReturned         int  `yaml:"hubs_returned"`
    DirectHitsReturned   int  `yaml:"direct_hits_returned"`
    ItemsWithFullContent int  `yaml:"items_with_full_content"`
    Limit                int  `yaml:"limit"`
}

// queryCluster — add Phrase field
type queryCluster struct {
    ID         int                  `yaml:"id"`
    Phrase     string               `yaml:"phrase"`  // NEW
    Size       int                  `yaml:"size"`
    Silhouette float64              `yaml:"silhouette"`
    Members    []queryClusterMember `yaml:"members"`
}

// queryPayload — replace Query string with Phrases []string
type queryPayload struct {
    Version  int            `yaml:"version"`
    Phrases  []string       `yaml:"phrases"`  // CHANGED from query string
    Items    []queryItem    `yaml:"items"`
    Clusters []queryCluster `yaml:"clusters"`
    Budget   queryBudget    `yaml:"budget"`
}
```

- [ ] **Step 2: Update `renderClusters` to accept `[]phrasedCluster`**

Replace the existing `renderClusters` signature:

```go
// renderClusters converts per-phrase cluster reports into the YAML wire shape.
// Each cluster is tagged with the phrase that produced it.
func renderClusters(phraseClusters []phrasedCluster) []queryCluster {
    var out []queryCluster

    for _, pc := range phraseClusters {
        if pc.report.autoK.K == 0 {
            continue
        }
        for clusterID := range pc.report.autoK.K {
            members := collectClusterMembers(pc.subgraph, pc.report, clusterID)
            out = append(out, queryCluster{
                ID:         clusterID,
                Phrase:     pc.phrase,
                Size:       len(members),
                Silhouette: pc.report.silhouettesByID[clusterID],
                Members:    members,
            })
        }
    }

    if out == nil {
        return []queryCluster{}
    }

    return out
}
```

- [ ] **Step 3: Update `renderQueryPayload` to accept `aggregatedSummary`**

Replace `renderQueryPayload(stdout io.Writer, query string, summary queryPipelineSummary)` with:

```go
// renderQueryPayload encodes the resolved YAML payload for the multi-phrase
// pipeline output.
func renderQueryPayload(stdout io.Writer, merged aggregatedSummary) error {
    items := renderItems(merged.resolvedItems)
    clusters := renderClusters(merged.phraseClusters)
    contentful := countItemsWithContent(items)

    directCount := 0
    hubCount := 0
    totalClusters := 0
    for _, cluster := range clusters {
        _ = cluster
        totalClusters++
    }
    for _, item := range items {
        for _, p := range item.Provenances {
            if p == provenanceDirect {
                directCount++
                break
            }
        }
        if item.InDegree != nil {
            hubCount++
        }
    }

    payload := queryPayload{
        Version:  1,
        Phrases:  merged.phrases,
        Items:    items,
        Clusters: clusters,
        Budget: queryBudget{
            PhrasesQueried:       len(merged.phrases),
            TotalNotes:           merged.totalNotes,
            WithEmbeddings:       merged.withEmbeddings,
            SubgraphSize:         merged.subgraphSize,
            SubgraphSizeCapped:   merged.subgraphCapped,
            HopsTraversed:        merged.hopsTraversed,
            ClustersFound:        totalClusters,
            HubsReturned:         hubCount,
            DirectHitsReturned:   directCount,
            ItemsWithFullContent: contentful,
            Limit:                merged.limit,
        },
    }

    const yamlIndent = 2

    encoder := yaml.NewEncoder(stdout)
    encoder.SetIndent(yamlIndent)

    err := encoder.Encode(payload)
    if err != nil {
        return fmt.Errorf("query: encode: %w", err)
    }

    closeErr := encoder.Close()
    if closeErr != nil {
        return fmt.Errorf("query: close encoder: %w", closeErr)
    }

    return nil
}
```

- [ ] **Step 4: Update `queryParsed` in test files**

In `internal/cli/query_subgraph_test.go`, update:

```go
type queryParsed struct {
    Version int      `yaml:"version"`
    Phrases []string `yaml:"phrases"`  // was: Query string
    Items   []struct {
        Path        string   `yaml:"path"`
        Kind        string   `yaml:"kind"`
        Score       float32  `yaml:"score"`
        Provenances []string `yaml:"provenances"`
        ClusterID   *int     `yaml:"cluster_id,omitempty"`
        InDegree    *int     `yaml:"in_degree,omitempty"`
        Content     string   `yaml:"content"`
    } `yaml:"items"`
    Clusters []struct {
        ID         int     `yaml:"id"`
        Phrase     string  `yaml:"phrase"`  // NEW
        Size       int     `yaml:"size"`
        Silhouette float64 `yaml:"silhouette"`
        Members    []struct {
            Path             string  `yaml:"path"`
            Score            float32 `yaml:"score"`
            IsRepresentative bool    `yaml:"is_representative"`
        } `yaml:"members"`
    } `yaml:"clusters"`
    Budget struct {
        PhrasesQueried       int  `yaml:"phrases_queried"`  // NEW
        TotalNotes           int  `yaml:"total_notes"`
        WithEmbeddings       int  `yaml:"with_embeddings"`
        SubgraphSize         int  `yaml:"subgraph_size"`
        SubgraphSizeCapped   bool `yaml:"subgraph_size_capped"`
        HopsTraversed        int  `yaml:"hops_traversed"`
        ClustersFound        int  `yaml:"clusters_found"`
        HubsReturned         int  `yaml:"hubs_returned"`
        DirectHitsReturned   int  `yaml:"direct_hits_returned"`
        ItemsWithFullContent int  `yaml:"items_with_full_content"`
        Limit                int  `yaml:"limit"`
    } `yaml:"budget"`
}
```

Also, in any test that previously checked `parsed.Query`, update to check `parsed.Phrases`. Search all test files for `.Query` references and update accordingly. The existing `QueryArgs{Query: "..."}` usages in tests do NOT need to change — the positional `Query` field still works as before.

- [ ] **Step 5: Run full check**

```bash
targ check-full 2>&1 | tail -30
```

Expected: all tests pass, no lint errors. Fix any compilation failures before proceeding.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/query.go internal/cli/query_test.go internal/cli/query_subgraph_test.go internal/cli/query_pipeline_test.go internal/cli/query_property_test.go internal/cli/query_integration_test.go
git commit -m "$(cat <<'EOF'
feat(query): multi-phrase aggregation; per-phrase clusters; phrases_queried budget

RunQuery now accepts Phrases []string (repeatable --phrase flag) and
loops over each phrase, aggregating server-side:
- items: dedup by path, max score, union provenances, max in_degree
- clusters: per-phrase, tagged with originating phrase
- budget: phrases_queried: N

Payload changes: top-level phrases[] replaces query string;
queryCluster gains phrase field; queryBudget gains phrases_queried.
Single-phrase backward compat preserved via Query positional arg.

AI-Used: [claude]
EOF
)"
```

---

## Task 4: Update `/recall` SKILL.md — single invocation with all phrases

**Files:**
- Modify: `skills/recall/SKILL.md`

**REQUIRED:** Use the `superpowers:writing-skills` skill for this task. Do not edit `skills/recall/SKILL.md` directly without running the full RED→GREEN→REFACTOR TDD cycle that the skill enforces.

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

```
/skill superpowers:writing-skills
```

Follow the skill's TDD discipline:
1. **RED (baseline behavior test):** Run a pressure test that verifies the *current* step-3 behavior (multiple `engram query` calls, agent-side union) before making changes.
2. **Update the skill (GREEN):** Replace step 3 multi-call prose with single-invocation prose (see below).
3. **Verify behavioral change (REFACTOR/pressure test):** Re-run the pressure test to confirm the new single-invocation behavior is correctly described and triggered.

The new step 3 content to put in:

```markdown
### Step 3 — Run one `engram query` with all Step 1 phrases

Pass every Step 1 phrase as a separate `--phrase` flag in a single invocation:

```bash
engram query \
  --phrase "<Step 1 phrase 1>" \
  --phrase "<Step 1 phrase 2>" \
  --phrase "<Step 1 phrase 3>"
  # ... one --phrase per Step 1 phrase
```

The binary returns one merged payload:
- **`items[]`** — dedup by path; max score across phrases; union provenances.
- **`clusters[]`** — per-phrase; each cluster carries a `phrase` field naming which phrase's subgraph it came from. Do not merge clusters across phrases.
- **`budget.phrases_queried`** — how many phrases were submitted.

`--limit N` caps direct hits per phrase before subgraph expansion. Default 20.
```

Also update the red-flag table entry (the one that previously said "You collapsed Step 1's phrases into one query before invoking the binary") to match the new single-invocation shape.

Also update any failure mode rows that referenced multiple `engram query` calls.

- [ ] **Step 2: Commit the SKILL.md update**

```bash
git add skills/recall/SKILL.md
git commit -m "$(cat <<'EOF'
feat(recall): step 3 uses single multi-phrase engram query invocation

Replaces N parallel engram query calls + agent-side union with one
call that passes all Step 1 phrases as --phrase flags. Binary now
owns the mechanical dedup/max-score/union-provenances aggregation.

AI-Used: [claude]
EOF
)"
```

---

## Task 5: File vault note on the binary/skill aggregation boundary

Per issue 639: "new note should be filed when this lands explaining the boundary."

- [ ] **Step 1: Write the vault note**

```bash
engram learn fact \
  --slug "binary-owns-mechanical-aggregation-skill-owns-policy" \
  --position continuation \
  --target 218 \
  --source "session log engram, 2026-05-26 UTC, context: implementing issue-639 multi-phrase engram query" \
  --situation "When the binary performs N homogeneous operations (N phrase embeddings, N subgraph expansions) and the results need combining with procedure-heavy, policy-light logic" \
  --subject "Mechanical aggregation (dedup by path, max score, union provenances)" \
  --predicate "belongs in the binary, not the consuming SKILL, when" \
  --object "the merge is procedure-heavy and policy-light — deterministic rules with no workflow judgment (which to prefer, how to weight, which conflicts to escalate). The SKILL retains ownership of policy choices (synthesis gate, plan-walk, dispatch decisions). This refines the general principle in 218, which prefers SKILL-side merging except in this case." \
  --relation "218.2026-05-26.combine-multi-primitive-calls-in-skill-not-binary|parent principle being refined: 218 says prefer SKILL-side merge; this note names the exception condition (procedure-heavy, policy-light merge)" 2>&1
```

- [ ] **Step 2: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
docs(vault): note on binary vs skill aggregation boundary (refines 218)

AI-Used: [claude]
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- ✅ `engram query --phrase` repeatable flag (Task 1)
- ✅ Backward-compat single-phrase positional still works (Task 1 — `Query string` kept)
- ✅ Items dedup by path, max score, union provenances (Task 2 `aggregatePhraseSummaries`)
- ✅ Per-phrase clusters tagged with phrase (Tasks 2+3, `phrasedCluster`)
- ✅ Union-deduped hubs via `in_degree` on merged items (Task 2 aggregation — max in_degree)
- ✅ `budget.phrases_queried: N` (Task 3 `queryBudget`)
- ✅ `/recall` SKILL.md step 3 updated to single invocation (Task 4)
- ✅ Tests: aggregation correctness, single-phrase backward compat (Tasks 1+2+3)
- ✅ Vault note on boundary refinement (Task 5, per issue-639 requirement)

**Gap:** Issue 639 says "per-phrase `direct_hits_returned` may be reported as a list or as totals." The plan uses post-merge count (items with `direct` provenance in merged result) — this is simpler and sufficient. If a list is needed later, it can be a follow-up.

**Placeholder scan:** No TBD, no "similar to Task N", all code blocks are complete.

**Type consistency:**
- `phrasedCluster` defined in Task 2 and used in Task 3's `renderClusters` — consistent.
- `aggregatedSummary` defined in Task 2 and used as `renderQueryPayload` parameter — consistent.
- `queryBudget.PhrasesQueried`, `queryCluster.Phrase`, `queryPayload.Phrases` — defined in Task 3 and tested in Task 2's new tests — consistent (tests run after implementation).
