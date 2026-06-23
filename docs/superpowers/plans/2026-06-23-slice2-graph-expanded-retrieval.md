# Slice 2 — Graph-Expanded Retrieval Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development per task (RED→GREEN→REFACTOR). Steps use checkbox (`- [ ]`). Use `targ test` / `targ check-full` — never raw `go test`/`go build`.

**Goal:** At query time, expand the cosine-matched seed set by traversing the vault's wikilink graph 1–2 hops *before* clustering, so bridge notes that cosine retrieval structurally misses (the transitive "Joe wants cake → … → we need sugar" case) enter the result set.

**Architecture:** A binary change in `engram query`'s retrieval stage reusing the existing `internal/vaultgraph.BFSWithCap` (undirected, depth+capacity bounded) and `BuildGraph`. A **pure** function returns the bridge *basenames* reachable from the matched note seeds; the `runQuery` wiring then builds `matchedMember`s for the bridges that have a compatible sidecar (path via the existing `pathOf`, vectors from the sidecar), appends them to the matched set, and clusters the expanded set. Bridges surface in `clusters[].members` tagged `graph_expanded`. GraphRAG *local* search / spreading activation (research §4 Stage 1) — NOT the killed global reduce.

**Tech Stack:** Go; `internal/vaultgraph` (BFSWithCap/BuildGraph/Note); `internal/cli/query.go`; rapid + gomega for tests (the pure function has no I/O to mock — no imptest needed; the wiring is covered by the real-binary integration test); `targ`; Python eval harness for the end-to-end proof.

## Verified code facts (traced 2026-06-23 — do not re-guess these)

- `vaultgraph.Note` = `{Basename string; LuhmannID string; Outgoing []string}`. **No `.Body`, no `.Path`.** `Outgoing = ParseWikilinks(body)`.
- `BuildGraph([]Note) Graph`; `BFSWithCap(Graph, seeds []string, maxDepth, capacity int) BFSResult{Visited map[string]struct{}, …}` — undirected (out ∪ in).
- **Slice-1 edges ARE traversable:** `engram amend --relation` renders a body `Related to:` section with `[[basename]]` wikilinks; `ParseWikilinks` parses them into `Outgoing`. (Verified by writing a real edge and reading the note + `migrateRelationLinks` operating on that section via `wikilinkRE`.) So the BFS sees slice-1's edges. The ask-reviewer's "amend writes frontmatter" claim was wrong (untested).
- `compatibleSidecar` = `{note vaultgraph.Note; sidecar embed.Sidecar}`. `embed.Sidecar` has `SituationVector []float32`, `BodyVector []float32`.
- `matchedMember` = `{basename, notePath string; vector, sitVec, bodyVec []float32; score float32; content string}`.
- `pathOf(basename string) string` builds a note path (used at `loadCompatibleSidecars`); `basenameFromNotePath(notePath) string` reverses it.
- `runQuery(ctx, args, notes []vaultgraph.Note, hits []compatibleSidecar, limit, deps, stdout)` — `notes` and `hits` are in scope. Injection point: between `matchSet := buildMatchedSet(noteUnion)` and `addMatchedChunksToMatchedSet`.
- `queryClusterMember` = `{Path string; Score float32; IsRepresentative bool}` — cluster members render **path + score only** (no content). The recall skill fetches content via `engram show` for members not in `items[]`. So a bridge needs a valid `notePath`, vectors, and score; `content: ""` is acceptable.
- `collectClusterMembers` builds `queryClusterMember` rows from `matchSet.members` — the clean site to surface a `graph_expanded` flag.
- `QueryArgs` flags are pure struct tags (targ auto-wires; no targets.go change). `--limit` uses the `0 → default` pattern.

## Global Constraints

- DI everywhere: the BFS/bridge-selection logic is pure over data structures; member construction (path, vectors) happens at the `runQuery` edge using already-loaded `notes`/`hits`. No new I/O in `internal/` business logic.
- Reuse `vaultgraph.BFSWithCap`/`BuildGraph`/`UndirectedNeighbors` — no new traversal engine.
- Total matched set stays ≤ `matchSetCap` (300) so clustering stays O(n²)-bounded.
- Only surface bridges with a **compatible sidecar** (current-model embedding) — clustering needs a vector.
- Local search only. The across-groups global reduce is killed (research §4) — no reduce task.
- `targ test`/`targ check-full`/`targ build` only. Named constants, wrapped/sentinel errors, `t.Parallel()`, line length < 120 (struct tags that can't break keep `//nolint:lll` as the existing `ChunksDir` tag does). Commit trailer `AI-Used: [claude]`.
- **Honest-caveat (carry into the eval):** payoff depends on **link density** — slice 1 writes only precise means-ends/causal edges, so a bridge surfaces only where a real chain was linked. The proof MUST pre-write the chain edges (via `engram amend`, verified traversable) and report cosine-only vs expanded honestly.

---

### Task 1: Pure bridge-basename BFS

**Files:**
- Create: `internal/cli/query_graph_expand.go`
- Test: `internal/cli/query_graph_expand_test.go`

**Interfaces:**
- Consumes: `vaultgraph.BuildGraph`, `vaultgraph.BFSWithCap`, `vaultgraph.Note`.
- Produces: `graphBridgeBasenames(notes []vaultgraph.Note, seeds []string, hops, capacity int) []string` — sorted bridge basenames = BFS-visited minus seeds. Pure; nil when `hops <= 0` or `seeds` empty.

- [ ] **Step 1: Write the failing test (RED)** — `internal/cli/query_graph_expand_test.go`

```go
package cli //nolint:testpackage // exercises unexported graphBridgeBasenames

import (
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestGraphBridgeBasenames_SurfacesUnmatchedBridge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// Chain A -> B -> C via body wikilinks (Outgoing). Seed = A; one hop must surface B.
	notes := []vaultgraph.Note{
		{Basename: "a-wants-cake", Outgoing: []string{"b-cake-needs-sweetness"}},
		{Basename: "b-cake-needs-sweetness", Outgoing: []string{"c-sugar-provides-sweetness"}},
		{Basename: "c-sugar-provides-sweetness"},
	}
	g.Expect(graphBridgeBasenames(notes, []string{"a-wants-cake"}, 1, 300)).
		To(ConsistOf("b-cake-needs-sweetness"))                 // 1 hop: B only
	g.Expect(graphBridgeBasenames(notes, []string{"a-wants-cake"}, 2, 300)).
		To(ConsistOf("b-cake-needs-sweetness", "c-sugar-provides-sweetness")) // 2 hops: B and C
	g.Expect(graphBridgeBasenames(notes, []string{"a-wants-cake"}, 0, 300)).
		To(BeEmpty())                                            // disabled
}
```

- [ ] **Step 2: Run to verify it fails** — `targ test`; FAIL: `graphBridgeBasenames` undefined.

- [ ] **Step 3: Implement (GREEN)** — `internal/cli/query_graph_expand.go`

```go
package cli

import (
	"sort"

	"github.com/toejough/engram/internal/vaultgraph"
)

// graphBridgeBasenames performs GraphRAG-local-search seed expansion: it
// traverses the vault wikilink graph from the cosine-matched note seeds
// (undirected, hops-bounded, capacity-bounded) and returns the BRIDGE
// basenames — visited nodes that are not themselves seeds. Pure. Returns
// nil when hops <= 0 or seeds is empty.
func graphBridgeBasenames(notes []vaultgraph.Note, seeds []string, hops, capacity int) []string {
	if hops <= 0 || len(seeds) == 0 {
		return nil
	}

	graph := vaultgraph.BuildGraph(notes)
	result := vaultgraph.BFSWithCap(graph, seeds, hops, capacity)

	seedSet := make(map[string]struct{}, len(seeds))
	for _, s := range seeds {
		seedSet[s] = struct{}{}
	}

	bridges := make([]string, 0, len(result.Visited))
	for basename := range result.Visited {
		if _, isSeed := seedSet[basename]; !isSeed {
			bridges = append(bridges, basename)
		}
	}

	sort.Strings(bridges)

	return bridges
}
```

- [ ] **Step 4: Run to verify it passes** — `targ test`; PASS.

- [ ] **Step 5: Add a rapid property test (capacity bound)**

```go
func TestGraphBridgeBasenames_RespectsCapacity(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(2, 30).Draw(rt, "n")
		capacity := rapid.IntRange(1, 10).Draw(rt, "cap")
		outgoing := make([]string, 0, n-1)
		for i := 1; i < n; i++ {
			outgoing = append(outgoing, "note"+strconv.Itoa(i))
		}
		notes := []vaultgraph.Note{{Basename: "note0", Outgoing: outgoing}}
		for i := 1; i < n; i++ {
			notes = append(notes, vaultgraph.Note{Basename: "note" + strconv.Itoa(i)})
		}
		bridges := graphBridgeBasenames(notes, []string{"note0"}, 2, capacity)
		if got := len(bridges) + 1; got > capacity { // visited = seed + bridges
			rt.Fatalf("visited %d exceeds cap %d", got, capacity)
		}
	})
}
```

Run: `targ test`; PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/query_graph_expand.go internal/cli/query_graph_expand_test.go
git commit -m "$(printf 'feat(query): pure graph-bridge BFS over wikilinks\n\nAI-Used: [claude]')"
```

---

### Task 2: Wire expansion into runQuery + `--graph-expand-hops`

**Files:**
- Modify: `internal/cli/query.go` (`QueryArgs`; `matchedMember` gains `graphExpanded bool`; constants; the `runQuery` injection; a `buildBridgeMembers` helper)
- Test: `internal/cli/query_integration_test.go`

**Interfaces:**
- Consumes: `graphBridgeBasenames` (Task 1); `pathOf`; `compatibleSidecar`; `matchedMember`.
- Produces: `QueryArgs.GraphExpandHops int`; bridges appended to `matchSet` before clustering, each `{basename, notePath: pathOf(basename), vector/sitVec/bodyVec from sidecar, score: 0, content: "", graphExpanded: true}`.

- [ ] **Step 1: Write the failing integration test (RED)**

Add to `query_integration_test.go`. Reuse the file's existing synthetic-sidecar planting style (the 30-note block) — factor a local `writeNoteWithSidecar(t, vault, basename, vec, body)` helper if one isn't already present, writing the `.md` (with the body, including any `[[wikilink]]`) and a prestamped `minilm-l6-v2@384` sidecar with `SituationVector=BodyVector=vec`.

```go
func TestEngramQuery_GraphExpand_SurfacesBridge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()
	aligned := unitVec(0)      // aligns with the query phrase
	orthogonal := unitVec(1)   // orthogonal -> cosine(query, bridge) < matchRelevanceFloor
	writeNoteWithSidecar(t, vault, "1.2026-06-23.a-buy-list", aligned, "what to buy [[2.2026-06-23.b-cake-needs-sweetness]]")
	writeNoteWithSidecar(t, vault, "2.2026-06-23.b-cake-needs-sweetness", orthogonal, "cake needs sweetness [[3.2026-06-23.c-sugar-provides-sweetness]]")
	writeNoteWithSidecar(t, vault, "3.2026-06-23.c-sugar-provides-sweetness", orthogonal, "sugar provides sweetness")

	expanded := clusterMemberBasenames(runEngramQueryYAML(t, vault, "what should I buy"))                       // default hops=2
	cosineOnly := clusterMemberBasenames(runEngramQueryYAML(t, vault, "what should I buy", "--graph-expand-hops", "-1"))

	g.Expect(expanded).To(ContainElement("2.2026-06-23.b-cake-needs-sweetness"))      // bridge surfaced by expansion
	g.Expect(cosineOnly).NotTo(ContainElement("2.2026-06-23.b-cake-needs-sweetness")) // cosine alone misses it
}
```

(`runEngramQueryYAML(t, vault, phrase, extraArgs...)` runs the built binary with `--phrase phrase` + extras and unmarshals the YAML. `clusterMemberBasenames(payload)` flattens `clusters[].members[].path` → basenames via `filepath.Base` minus `.md`. `unitVec(i)` returns a 384-dim one-hot. Mirror the existing integration helpers; add the small missing ones in the test file.)

- [ ] **Step 2: Run to verify it fails** — `targ test`; FAIL: bridge absent in both arms (no expansion wired).

- [ ] **Step 3: Add the flag, the constant, and the `graphExpanded` field**

In `QueryArgs` after `Project`:

```go
	GraphExpandHops int `targ:"flag,name=graph-expand-hops,desc=wikilink hops to expand the cosine seed set before clustering; 0=default 2, negative disables"` //nolint:lll // single struct-tag string
```

Near the query constants:

```go
	// defaultGraphExpandHops is the BFS depth when --graph-expand-hops is
	// unset (0). A negative value disables expansion. Research §4 Stage 1:
	// 1-2 hops surfaces transitive/compositional bridges.
	defaultGraphExpandHops = 2
```

Add `graphExpanded bool` to the `matchedMember` struct (zero value false; existing constructions are unaffected).

- [ ] **Step 4: Add the bridge-member builder + wire it into `runQuery`**

```go
// buildBridgeMembers turns graph-bridge basenames into matchedMembers using
// already-loaded sidecars. Bridges have no query cosine: cluster coordinate is
// the situation axis, score is 0, content is empty (the recall skill fetches it
// via `engram show`). Only bridges with a compatible sidecar are included.
func buildBridgeMembers(bridges []string, hitByBasename map[string]compatibleSidecar) []matchedMember {
	members := make([]matchedMember, 0, len(bridges))
	for _, basename := range bridges {
		hit, ok := hitByBasename[basename]
		if !ok {
			continue
		}
		members = append(members, matchedMember{
			basename:      basename,
			notePath:      pathOf(basename),
			vector:        hit.sidecar.SituationVector,
			sitVec:        hit.sidecar.SituationVector,
			bodyVec:       hit.sidecar.BodyVector,
			score:         0,
			content:       "",
			graphExpanded: true,
		})
	}
	return members
}
```

In `runQuery`, immediately after `matchSet := buildMatchedSet(noteUnion)`:

```go
	hops := args.GraphExpandHops
	if hops == 0 {
		hops = defaultGraphExpandHops
	}
	if hops > 0 && len(matchSet.members) < matchSetCap {
		seeds := make([]string, 0, len(noteUnion))
		for _, c := range noteUnion {
			seeds = append(seeds, c.basename)
		}
		hitByBasename := make(map[string]compatibleSidecar, len(hits))
		for _, h := range hits {
			hitByBasename[h.note.Basename] = h
		}
		// capacity counts seeds in BFS Visited; leave room under matchSetCap.
		capacity := matchSetCap - len(matchSet.members) + len(seeds)
		bridges := graphBridgeBasenames(notes, seeds, hops, capacity)
		matchSet.members = append(matchSet.members, buildBridgeMembers(bridges, hitByBasename)...)
	}
```

- [ ] **Step 5: Run to verify it passes** — `targ test`; PASS (bridge present at default hops, absent at `-1`).

- [ ] **Step 6: REFACTOR + gate B** — if `runQuery` grows unwieldy, extract the block above into `appendGraphBridges(matchSet *matchedSet, notes []vaultgraph.Note, hits []compatibleSidecar, noteUnion []scoredCandidate, hops int)`. Run `targ check-full`. Gate B (design-fit) on the diff.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/query.go internal/cli/query_integration_test.go
git commit -m "$(printf 'feat(query): graph-expanded retrieval before clustering\n\nAI-Used: [claude]')"
```

---

### Task 3: Surface `graph_expanded` on cluster members (transparency)

**Files:**
- Modify: `internal/cli/query.go` (`queryClusterMember` gains a field; `collectClusterMembers` copies it)
- Test: `internal/cli/query_integration_test.go`

**Interfaces:**
- Consumes: `matchedMember.graphExpanded` (Task 2).
- Produces: `queryClusterMember.GraphExpanded bool` (`yaml:"graph_expanded,omitempty"`) so the recall skill can tell a member was surfaced by traversal, not similarity.

- [ ] **Step 1: Write the failing test (RED)**

```go
func TestEngramQuery_GraphExpand_TagsBridgeMember(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// ... same transitive vault as Task 2 ...
	payload := runEngramQueryYAML(t, vault, "what should I buy")
	row := findClusterMember(payload, "2.2026-06-23.b-cake-needs-sweetness")
	g.Expect(row.GraphExpanded).To(BeTrue())
}
```

(`findClusterMember` scans `clusters[].members` for the basename and returns the row; add the `GraphExpanded bool` field to the test's payload-decode struct.)

- [ ] **Step 2: Run to verify it fails** — `targ test`; FAIL (field absent/false).

- [ ] **Step 3: Implement** — add `GraphExpanded bool yaml:"graph_expanded,omitempty"` to `queryClusterMember`; in `collectClusterMembers`, set `GraphExpanded: member.graphExpanded` in the row literal.

- [ ] **Step 4: Run to verify it passes** — `targ test`; PASS.

- [ ] **Step 5: `targ check-full` + commit**

```bash
git add internal/cli/query.go internal/cli/query_integration_test.go
git commit -m "$(printf 'feat(query): tag graph-expanded bridges on cluster members\n\nAI-Used: [claude]')"
```

---

### Task 4: End-to-end value proof (transitive bridge-surfaced rate)

**Files:**
- Modify: `dev/eval/traps/cake_fixtures.py` (the `transitive` fixture writes the chain EDGES via `engram amend --relation`, verified traversable)
- Create: `dev/eval/traps/graphexpand.py`

**Interfaces:**
- Consumes: the built binary (`engram query --graph-expand-hops`), the transitive fixture.
- Produces: a table — for a query whose cosine misses the bridge, the bridge present in `clusters[].members` at hops=-1 (cosine-only) vs default (expanded).

- [ ] **Step 1: Make the transitive fixture write real edges**

After `build("transitive", …)` writes the three notes, persist the chain with `engram amend` (renders body `Related to:` wikilinks → `Outgoing` → traversable, verified):

```python
def _amend(vault, target, rel_basename, typed):
    env = dict(os.environ); env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(["engram", "amend", "--target", target,
                    "--relation", f"{rel_basename}|{typed}"], env=env, check=True, capture_output=True, text=True)
# in build() for kind == "transitive", after _learn() of the 3 notes:
#   joe-wants-cake --(causal: cake)--> cake-needs-sweetness --(means-ends: sweetness)--> sugar-provides-sweetness
```

(Resolve targets by Luhmann id or full basename — `engram amend --target 1 --relation "2.<date>.cake-needs-sweetness|causal: cake"`. Confirm the written note shows a `Related to:` `[[…]]` line.)

- [ ] **Step 2: Write `graphexpand.py`** — parse the YAML payload's `clusters[].members[].path` (NOT a raw substring), check whether the bridge is a cluster member:

```python
"""Slice-2 proof: does graph expansion surface the transitive bridge cosine misses?
Builds the transitive fixture (chain edges present), runs `engram query` cosine-only
(--graph-expand-hops -1) vs expanded (default), and checks clusters[].members[].path.

Usage: python3 graphexpand.py
"""
import os, subprocess, sys
import yaml  # PyYAML; if unavailable, parse the `path:` lines under `members:` manually
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cake_fixtures

QUERY = ["--phrase", "what ingredient should I buy for the recipe"]  # cosine-distant from the bridge
BRIDGE = "sugar-provides-sweetness"

def _bridge_in_members(vault, extra):
    env = dict(os.environ); env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = os.path.join(vault, "_chunks")
    out = subprocess.run(["engram", "query", *QUERY, *extra], env=env, capture_output=True, text=True).stdout
    doc = yaml.safe_load(out) or {}
    for cl in doc.get("clusters", []):
        for m in cl.get("members", []):
            if BRIDGE in os.path.basename(m.get("path", "")):
                return True
    return False

def main():
    vault = "/tmp/graphexpand/vault"
    cake_fixtures.build("transitive", vault)
    cosine = _bridge_in_members(vault, ["--graph-expand-hops", "-1"])
    expand = _bridge_in_members(vault, [])
    print(f"bridge '{BRIDGE}' in clusters[].members:  cosine-only={cosine}  graph-expanded={expand}")
    assert not cosine and expand, "expected bridge MISSED by cosine, SURFACED by expansion"
    print("PASS: graph expansion surfaces the transitive bridge cosine missed")

if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Run the proof**

Run: `targ build && cd dev/eval/traps && python3 graphexpand.py`
Expected: `cosine-only=False  graph-expanded=True` → PASS. If the bridge surfaces even cosine-only, the bridge is not actually embedding-distant from the QUERY — adjust the QUERY wording so cosine genuinely misses it (the RED of this proof). If expansion still fails to surface it, confirm the fixture's `Related to:` edges exist (link-density caveat) before concluding.

- [ ] **Step 4: Optional warm /recall confirmation (isolated agent)** — if budget allows, run one warm `/recall` over the transitive fixture (reuse `cake.py`'s warm harness) and confirm the agent's payload contains the bridge and it reasons over it. Confirmation, not a gate.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/traps/cake_fixtures.py dev/eval/traps/graphexpand.py
git commit -m "$(printf 'test(query): e2e proof graph expansion surfaces the transitive bridge\n\nAI-Used: [claude]')"
```

---

### Task 5: Document + close

**Files & exact edits:**
- `docs/research/2026-06-22-emergent-synthesis-case.md` §4 Stage 1 → mark BUILT, record the success-criterion result (bridge surfaced cosine-only vs expanded; reference-based).
- `docs/design/2026-06-23-cross-cluster-linking.md` §1b roadmap row 2 → status `built`; note the `BFSWithCap` reuse and that slice-1 `Related to:` edges are traversable.
- `docs/architecture/c1-system-context.md` — in the **recall sequence diagram**, insert a `Note over E:` between the cosine-match step ("top-30 per phrase, unioned") and the clustering step ("one AutoK cluster"): *"Graph expansion (local search): BFS 1–2 hops over wikilinks from the matched note seeds; append bridge notes with compatible sidecars (tagged graph_expanded); then cluster."*
- `CLAUDE.md` — in the `internal/cli` / query description, add: *"`query.go` graph-expands the cosine-matched note seeds via 1–2 wikilink hops (`vaultgraph.BFSWithCap`) before clustering, surfacing bridge notes for transitive/compositional synthesis."*
- `dev/eval/cumulative/EXPERIMENT-LOG.md` — append a 2026-06-23 slice-2 entry: bridge-surfaced rate (cosine-only vs graph-expanded) + the link-density caveat.

- [ ] **Step 1: Apply each edit above.** Gate C over every touched doc.
- [ ] **Step 2: Final commit (gate D over prose):**

```bash
git add -A
git commit -m "$(printf 'docs: mark slice-2 graph-expanded retrieval built; record results\n\nAI-Used: [claude]')"
```

## Self-Review

**1. Spec coverage:** research §4 Stage 1 (expand seed by BFS 1–2 hops before cluster) → Tasks 1–2; success criterion (surface bridge, reference-based, not LLM-judge) → Task 4 (parses `clusters[].members[].path`); design §1b row 2 (traverse slice-1 edges, transitive) → Tasks 2/4 (edges verified body-traversable); local-not-global → honored; link-density caveat → Constraints + Task 4 Step 3; provenance transparency → Task 3. Covered.

**2. Placeholder scan:** every code step has real Go/Python against **verified** struct fields (see "Verified code facts"); every run step has exact command + expected output. No deferred "verify field names" placeholder remains.

**3. Type consistency:** `graphBridgeBasenames(notes, seeds, hops, capacity) []string` (Task 1) → consumed in Task 2; `buildBridgeMembers(bridges, hitByBasename) []matchedMember`; `matchedMember.graphExpanded` (Task 2) → `queryClusterMember.GraphExpanded` (Task 3); `QueryArgs.GraphExpandHops` + `defaultGraphExpandHops`; `pathOf`/`matchSetCap` reused as defined.

**Known accepted risk:** the proof depends on the bridge being genuinely cosine-distant from the query and the chain edges existing — both controlled by the fixture. Low real-vault link density bounds the value honestly (caveat constraint); the deterministic integration test still proves the mechanism where edges exist.
