# Slice 2 — Graph-Expanded Retrieval Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development for each task (RED→GREEN→REFACTOR). Steps use checkbox (`- [ ]`) syntax. Use `targ test` / `targ check-full` — never raw `go test`/`go build`.

**Goal:** At query time, expand the cosine-matched seed set by traversing the vault's wikilink graph 1–2 hops *before* clustering, so bridge notes that cosine retrieval structurally misses (the transitive "Joe wants cake → … → we need sugar" case) enter the result set.

**Architecture:** A binary change in `engram query`'s retrieval stage that reuses the existing `internal/vaultgraph.BFSWithCap` (undirected, depth+capacity bounded) and `BuildGraph` primitives. After `buildMatchedSet(noteUnion)` builds the cosine seed set and before `clusterMatchedSet`, BFS-expand from the matched note basenames over the wikilink graph, construct `matchedMember`s for the surfaced bridge notes from their already-loaded sidecars, append them to the matched set (tagged `graph_expanded` for transparency), then cluster the expanded set. No embedding-store change; GraphRAG *local* search / spreading activation (research §4 Stage 1), NOT the killed global reduce.

**Tech Stack:** Go; `internal/vaultgraph` (BFSWithCap/BuildGraph/Note); `internal/cli/query.go` (RunQuery pipeline); imptest + rapid + gomega for tests; `targ` build system; Python eval harness (`dev/eval/traps/`) for the end-to-end value proof.

## Global Constraints

- DI everywhere: no `os.*`/`http.*` in `internal/` business logic; the expansion function is pure over data structures (notes, seeds, sidecars), wired at the `RunQuery` edge.
- Reuse existing primitives — `vaultgraph.BFSWithCap`, `BuildGraph`, `UndirectedNeighbors`. No new traversal engine.
- Bound expansion: total matched set stays ≤ `matchSetCap` (300) so clustering stays O(n²)-bounded.
- Only surface bridges that have a **compatible sidecar** (a current-model embedding) — clustering needs a vector; a bridge without one is skipped.
- Local search only. The across-groups reduce (GraphRAG global) is killed (research §4) — do not build it.
- `targ test`, `targ check-full`, `targ build` only. Line length < 120. Sentinel errors, wrapped errors, named constants per `.claude/rules/go.md`.
- Commit trailer `AI-Used: [claude]`.
- **Honest-caveat requirement (carry into the eval):** the payoff depends on **link density** — slice 1 writes only precise means-ends/causal edges, so a bridge surfaces only where a real chain was linked. The end-to-end harness MUST pre-write the chain edges and report cosine-only vs expanded honestly; do not claim a lift the link density can't deliver.

---

### Task 1: Pure graph-expansion function

**Files:**
- Create: `internal/cli/query_graph_expand.go`
- Test: `internal/cli/query_graph_expand_test.go`

**Interfaces:**
- Consumes: `vaultgraph.BuildGraph([]vaultgraph.Note) Graph`, `vaultgraph.BFSWithCap(Graph, []string, int, int) BFSResult`, the `matchedMember` and `compatibleSidecar` types in `query.go`.
- Produces: `expandSeedsViaGraph(notes []vaultgraph.Note, seeds []string, hitByBasename map[string]compatibleSidecar, hops, capacity int) []matchedMember` — returns the **bridge** members (BFS-visited basenames that are NOT seeds AND have a compatible sidecar), each built from its sidecar with `vector = sitVec`, `score = 0`.

- [ ] **Step 1: Write the failing test (RED)**

```go
package cli //nolint:testpackage // exercises unexported expandSeedsViaGraph

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestExpandSeedsViaGraph_SurfacesUnmatchedBridge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Chain: query matches A and C; B is the bridge (linked A->B->C) that
	// did NOT match a phrase. One hop from A must surface B.
	notes := []vaultgraph.Note{
		{Basename: "a-wants-cake", Body: []byte("Joe wants cake [[b-cake-needs-sweetness]]")},
		{Basename: "b-cake-needs-sweetness", Body: []byte("cake needs sweetness [[c-sugar-provides-sweetness]]")},
		{Basename: "c-sugar-provides-sweetness", Body: []byte("sugar provides sweetness")},
	}
	hitByBasename := map[string]compatibleSidecar{
		"a-wants-cake":               {note: notes[0], sidecar: embed.Sidecar{SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0}}},
		"b-cake-needs-sweetness":     {note: notes[1], sidecar: embed.Sidecar{SituationVector: []float32{0, 1}, BodyVector: []float32{0, 1}}},
		"c-sugar-provides-sweetness": {note: notes[2], sidecar: embed.Sidecar{SituationVector: []float32{0, 1}, BodyVector: []float32{0, 1}}},
	}

	bridges := expandSeedsViaGraph(notes, []string{"a-wants-cake"}, hitByBasename, 1, 300)

	names := make([]string, 0, len(bridges))
	for _, b := range bridges {
		names = append(names, b.basename)
	}
	g.Expect(names).To(ConsistOf("b-cake-needs-sweetness"))            // bridge surfaced
	g.Expect(bridges[0].sitVec).To(Equal([]float32{0, 1}))            // carries its sidecar vector
	g.Expect(bridges[0].vector).To(Equal([]float32{0, 1}))           // cluster coord = situation axis
}

func TestExpandSeedsViaGraph_SkipsSeedsAndSidecarlessNotes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	notes := []vaultgraph.Note{
		{Basename: "seed", Body: []byte("[[bridge-no-vec]] [[bridge-vec]]")},
		{Basename: "bridge-no-vec", Body: []byte("no sidecar")},
		{Basename: "bridge-vec", Body: []byte("has sidecar")},
	}
	hitByBasename := map[string]compatibleSidecar{
		"seed":       {note: notes[0], sidecar: embed.Sidecar{SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0}}},
		"bridge-vec": {note: notes[2], sidecar: embed.Sidecar{SituationVector: []float32{0, 1}, BodyVector: []float32{0, 1}}},
	}
	bridges := expandSeedsViaGraph(notes, []string{"seed"}, hitByBasename, 1, 300)
	names := make([]string, 0, len(bridges))
	for _, b := range bridges {
		names = append(names, b.basename)
	}
	g.Expect(names).To(ConsistOf("bridge-vec"))   // seed excluded; sidecar-less bridge skipped
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `targ test` (or narrow to the package). Expected: FAIL — `expandSeedsViaGraph` undefined.

- [ ] **Step 3: Implement (GREEN)**

```go
package cli

import "github.com/toejough/engram/internal/vaultgraph"

// expandSeedsViaGraph performs GraphRAG-local-search seed expansion: it
// traverses the vault wikilink graph from the cosine-matched note seeds
// (undirected, hops-bounded, capacity-bounded) and returns matchedMembers
// for the surfaced BRIDGE notes — visited basenames that are not themselves
// seeds and that carry a compatible sidecar (clustering needs a vector).
//
// Bridges have no query cosine: their cluster coordinate is the situation
// axis (consistent with how notes embed) and their score is 0. Returns nil
// when hops <= 0, seeds is empty, or nothing new is reachable.
func expandSeedsViaGraph(
	notes []vaultgraph.Note,
	seeds []string,
	hitByBasename map[string]compatibleSidecar,
	hops, capacity int,
) []matchedMember {
	if hops <= 0 || len(seeds) == 0 {
		return nil
	}

	graph := vaultgraph.BuildGraph(notes)
	result := vaultgraph.BFSWithCap(graph, seeds, hops, capacity)

	seedSet := make(map[string]struct{}, len(seeds))
	for _, s := range seeds {
		seedSet[s] = struct{}{}
	}

	bridges := make([]matchedMember, 0, len(result.Visited))
	for basename := range result.Visited {
		if _, isSeed := seedSet[basename]; isSeed {
			continue
		}

		hit, ok := hitByBasename[basename]
		if !ok {
			continue // no compatible sidecar -> cannot cluster -> skip
		}

		bridges = append(bridges, matchedMember{
			basename: basename,
			notePath: hit.note.Path,
			vector:   hit.sidecar.SituationVector,
			sitVec:   hit.sidecar.SituationVector,
			bodyVec:  hit.sidecar.BodyVector,
			score:    0,
			content:  string(hit.note.Body),
		})
	}

	return bridges
}
```

(During execution, verify the exact field names: `vaultgraph.Note.Path`/`.Body`/`.Basename`, `embed.Sidecar.SituationVector`/`.BodyVector`, and the `matchedMember` fields — read the structs; adjust if a name differs. Use the stripped-content helper `stripWikilinks` on `content` if the rendered items elsewhere strip pointers — match the existing note-member construction.)

- [ ] **Step 4: Run to verify it passes**

Run: `targ test`. Expected: PASS (both tests).

- [ ] **Step 5: Add a rapid property test (determinism + bound)**

```go
func TestExpandSeedsViaGraph_RespectsCapacity(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(2, 30).Draw(rt, "n")
		cap := rapid.IntRange(1, 10).Draw(rt, "cap")
		notes := make([]vaultgraph.Note, n)
		hits := map[string]compatibleSidecar{}
		body := ""
		for i := 1; i < n; i++ {
			body += "[[note" + strconv.Itoa(i) + "]] "
		}
		notes[0] = vaultgraph.Note{Basename: "note0", Body: []byte(body)}
		hits["note0"] = compatibleSidecar{note: notes[0], sidecar: embed.Sidecar{SituationVector: []float32{1}, BodyVector: []float32{1}}}
		for i := 1; i < n; i++ {
			bn := "note" + strconv.Itoa(i)
			notes[i] = vaultgraph.Note{Basename: bn, Body: []byte("leaf")}
			hits[bn] = compatibleSidecar{note: notes[i], sidecar: embed.Sidecar{SituationVector: []float32{0}, BodyVector: []float32{0}}}
		}
		bridges := expandSeedsViaGraph(notes, []string{"note0"}, hits, 2, cap)
		// visited (seed + bridges) never exceeds capacity
		if got := len(bridges) + 1; got > cap {
			rt.Fatalf("visited %d exceeds cap %d", got, cap)
		}
	})
}
```

Run: `targ test`. Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/query_graph_expand.go internal/cli/query_graph_expand_test.go
git commit -m "feat(query): pure graph-expansion seed function (BFS over wikilinks)

AI-Used: [claude]"
```

---

### Task 2: Wire expansion into RunQuery + the `--graph-expand-hops` flag

**Files:**
- Modify: `internal/cli/query.go` (the `QueryArgs` struct ~line; the `runQuery` body between `buildMatchedSet` and `clusterMatchedSet`; add constants)
- Modify: `internal/cli/targets.go` (if the flag needs wiring there — verify)
- Test: `internal/cli/query_integration_test.go` (add a transitive-bridge case)

**Interfaces:**
- Consumes: `expandSeedsViaGraph` (Task 1).
- Produces: `QueryArgs.GraphExpandHops int`; bridges appended to `matchSet` before clustering.

- [ ] **Step 1: Write the failing integration test (RED)**

Add to `query_integration_test.go` a test that builds the real binary, plants a 3-note transitive vault with synthetic sidecars where the bridge note's vector is **orthogonal to the query** (so cosine misses it) but it is wikilink-reachable from a matched note. Assert: with default hops the bridge appears in some `clusters[].members`; with `--graph-expand-hops -1` it does NOT.

```go
func TestEngramQuery_GraphExpand_SurfacesBridge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()
	// note A: vector aligns with the query phrase; links to bridge B.
	// note B (bridge): vector orthogonal to the query; links to C.
	// Plant sidecars so cosine(query, B) < matchRelevanceFloor.
	writeNoteWithSidecar(t, vault, "a-buy-list", alignedVec, "what to buy [[b-cake-needs-sweetness]]")
	writeNoteWithSidecar(t, vault, "b-cake-needs-sweetness", orthogonalVec, "cake needs sweetness [[c-sugar-provides-sweetness]]")
	writeNoteWithSidecar(t, vault, "c-sugar-provides-sweetness", orthogonalVec, "sugar provides sweetness")

	expanded := runEngramQueryYAML(t, vault, []string{"--phrase", "what should I buy"})           // default hops
	cosineOnly := runEngramQueryYAML(t, vault, []string{"--phrase", "what should I buy", "--graph-expand-hops", "-1"})

	g.Expect(clusterMemberBasenames(expanded)).To(ContainElement("b-cake-needs-sweetness"))     // bridge surfaced
	g.Expect(clusterMemberBasenames(cosineOnly)).NotTo(ContainElement("b-cake-needs-sweetness")) // cosine misses it
}
```

(Reuse the existing synthetic-sidecar planting helpers in the integration test file; if none is factored out, add a small `writeNoteWithSidecar` helper mirroring the existing 30-note planting block. `clusterMemberBasenames` parses the YAML payload's `clusters[].members[].path` to basenames.)

- [ ] **Step 2: Run to verify it fails**

Run: `targ test`. Expected: FAIL — the bridge is absent in both arms (no expansion wired yet), so the `expanded` assertion fails.

- [ ] **Step 3: Add the flag + constant + default**

In `QueryArgs` (after `Project`):

```go
	GraphExpandHops int `targ:"flag,name=graph-expand-hops,desc=wikilink hops to expand the cosine seed set before clustering (default 2; negative disables)"` //nolint:lll
```

Add constants near the other query constants:

```go
	// defaultGraphExpandHops is the BFS depth used when --graph-expand-hops
	// is unset (0). Negative disables expansion (cosine-only). Research §4
	// Stage 1: 1-2 hops surfaces transitive/compositional bridges.
	defaultGraphExpandHops = 2
```

- [ ] **Step 4: Wire expansion into `runQuery`**

Between `matchSet := buildMatchedSet(noteUnion)` and `addMatchedChunksToMatchedSet`, insert:

```go
	hops := args.GraphExpandHops
	if hops == 0 {
		hops = defaultGraphExpandHops
	}
	if hops > 0 {
		hitByBasename := make(map[string]compatibleSidecar, len(hits))
		for _, h := range hits {
			hitByBasename[h.note.Basename] = h
		}
		seeds := make([]string, 0, len(noteUnion))
		for _, c := range noteUnion {
			seeds = append(seeds, c.basename)
		}
		room := matchSetCap - len(matchSet.members)
		if room > 0 {
			bridges := expandSeedsViaGraph(notes, seeds, hitByBasename, hops, room+len(seeds))
			matchSet.members = append(matchSet.members, bridges...)
		}
	}
```

(`hits` and `notes` are already in scope in `runQuery` — verify; `runQuery`'s signature includes `notes []vaultgraph.Note, hits []compatibleSidecar`. The capacity passed to BFS counts seeds in its visited set, hence `room+len(seeds)`.)

- [ ] **Step 5: Run to verify it passes**

Run: `targ test`. Expected: PASS — bridge present with default hops, absent with `-1`.

- [ ] **Step 6: REFACTOR + gate B**

Extract the seed/hit-map/append block into a helper `appendGraphBridges(matchSet *matchedSet, notes []vaultgraph.Note, hits []compatibleSidecar, noteUnion []scoredCandidate, hops int)` if `runQuery` grows unwieldy, keeping `runQuery` linear. Run `targ check-full` (all lints + coverage). Then gate B (design-fit) on the diff.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/query.go internal/cli/query_integration_test.go
git commit -m "feat(query): graph-expanded retrieval — traverse wikilinks before clustering

AI-Used: [claude]"
```

---

### Task 3: Tag bridges `graph_expanded` in the payload (transparency)

**Files:**
- Modify: `internal/cli/query.go` (provenance constant + tag bridge items via `appendUniqueProvenance`)
- Test: `internal/cli/query_integration_test.go` (assert the bridge item carries `provenances: [graph_expanded]`)

**Interfaces:**
- Consumes: the bridge members from Task 2; the existing `appendUniqueProvenance(item, role)` and `Provenances` payload field.
- Produces: `provenanceGraphExpanded = "graph_expanded"`; bridge items[] carry it so the recall skill can tell a note was surfaced by traversal, not similarity.

- [ ] **Step 1: Write the failing test (RED)**

```go
func TestEngramQuery_GraphExpand_TagsBridgeProvenance(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// ... same transitive vault as Task 2 ...
	payload := runEngramQueryYAML(t, vault, []string{"--phrase", "what should I buy"})
	bridge := findItemByBasename(payload, "b-cake-needs-sweetness")
	g.Expect(bridge.Provenances).To(ContainElement("graph_expanded"))
}
```

- [ ] **Step 2: Run to verify it fails** — `targ test`; FAIL (no such provenance).

- [ ] **Step 3: Implement** — add `provenanceGraphExpanded = "graph_expanded"` constant; carry a marker from `expandSeedsViaGraph` (e.g. a `graphExpanded bool` field on `matchedMember`, default false, true for bridges) and, where members become `resolvedItem`s, call `appendUniqueProvenance(item, provenanceGraphExpanded)` for graph-expanded members. (Verify the member→resolvedItem mapping site; reuse `appendUniqueProvenance`.)

- [ ] **Step 4: Run to verify it passes** — `targ test`; PASS.

- [ ] **Step 5: `targ check-full` + commit**

```bash
git add internal/cli/query.go internal/cli/query_integration_test.go
git commit -m "feat(query): tag graph-expanded bridges in the payload provenance

AI-Used: [claude]"
```

---

### Task 4: End-to-end value proof (transitive bridge-surfaced rate)

**Files:**
- Modify: `dev/eval/traps/cake_fixtures.py` (the `transitive` fixture: write the chain notes WITH `[[wikilink]]` edges so the graph exists)
- Create: `dev/eval/traps/graphexpand.py` (deterministic query-level proof: cosine-only vs expanded bridge-surfaced rate)

**Interfaces:**
- Consumes: the built binary (`engram query --graph-expand-hops`), the transitive fixture.
- Produces: a table — for a query whose cosine misses the bridge, the bridge-surfaced rate at hops=-1 (cosine-only) vs default (expanded).

- [ ] **Step 1: Make the transitive fixture write real edges**

Update `cake_fixtures.build("transitive", …)` so the three notes carry the chain wikilinks in their bodies (or amend them post-`learn`): `joe-wants-cake` → `[[…cake-needs-sweetness]]` → `[[…sugar-provides-sweetness]]`. (The slice-1 fixture wrote notes without edges; slice 2 needs the edges present to traverse. Use `engram amend --target <A> --relation "<B>|causal: cake"` to write them exactly as slice-1 recall would, so the proof uses real persisted edges.)

- [ ] **Step 2: Write `graphexpand.py`**

```python
"""Slice-2 proof: does graph expansion surface the transitive bridge cosine misses?
Builds the transitive fixture (chain edges present), runs `engram query` twice —
cosine-only (--graph-expand-hops -1) and expanded (default) — and reports whether the
bridge note `sugar-provides-sweetness` appears in clusters[].members.

Usage: python3 graphexpand.py [--n 3]
"""
import json, os, subprocess, sys, glob
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cake_fixtures

QUERY = ["--phrase", "what ingredient should I buy for the recipe"]   # cosine-distant from "provides sweetness"
BRIDGE = "sugar-provides-sweetness"

def _members(vault, extra):
    env = dict(os.environ); env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = os.path.join(vault, "_chunks")
    out = subprocess.run(["engram", "query", *QUERY, *extra], env=env, capture_output=True, text=True).stdout
    return BRIDGE in out   # bridge basename present anywhere in the clustered payload

def main():
    vault = "/tmp/graphexpand/vault"
    cake_fixtures.build("transitive", vault)   # writes chain notes + edges
    cosine = _members(vault, ["--graph-expand-hops", "-1"])
    expand = _members(vault, [])               # default hops=2
    print(f"bridge '{BRIDGE}' surfaced:  cosine-only={cosine}  graph-expanded={expand}")
    assert not cosine and expand, "expected bridge MISSED by cosine, SURFACED by expansion"
    print("PASS: graph expansion surfaces the transitive bridge cosine missed")

if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Run the proof**

Run: `cd dev/eval/traps && targ build && python3 graphexpand.py`
Expected: `cosine-only=False  graph-expanded=True` → PASS. (If the bridge is cosine-close enough to surface even cosine-only, tighten the QUERY phrase / fixture wording so the bridge is genuinely embedding-distant — the RED of this proof is "cosine alone misses it". If link density is the blocker, report it per the honest-caveat constraint.)

- [ ] **Step 4: Optional warm /recall confirmation (isolated agent)**

If time/budget allows, run one warm `/recall` over the transitive fixture (reuse `cake.py`'s warm harness pattern) and confirm the agent's payload now contains the bridge note — the value realized end-to-end. Not a gate; a confirmation.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/traps/cake_fixtures.py dev/eval/traps/graphexpand.py
git commit -m "test(query): end-to-end proof graph expansion surfaces the transitive bridge

AI-Used: [claude]"
```

---

### Task 5: Document + close

**Files:**
- Modify: `docs/design/2026-06-23-cross-cluster-linking.md` (§1b roadmap row 2 → built; note the BFS reuse)
- Modify: `docs/research/2026-06-22-emergent-synthesis-case.md` (§4 Stage 1 → built, with the success-criterion result)
- Modify: `docs/architecture/c1-system-context.md` (recall sequence: add the graph-expansion step before clustering)
- Modify: `CLAUDE.md` (the `internal/cli` / query description, if it characterizes retrieval) and `dev/eval/cumulative/EXPERIMENT-LOG.md` (slice-2 entry)

- [ ] **Step 1: Update each doc** to reflect graph-expanded retrieval as built, with the measured bridge-surfaced result (cosine-only vs expanded) and the link-density caveat. Gate C over every touched doc.

- [ ] **Step 2: Final commit** (gate D over prose):

```bash
git add -A
git commit -m "docs: mark slice-2 graph-expanded retrieval built; record results

AI-Used: [claude]"
```

## Self-Review

**1. Spec coverage:** research §4 Stage 1 (expand seed by traversing vaultgraph 1–2 hops before clustering) → Tasks 1–2; success criterion (surface bridge, lift over cosine-only, reference-based not LLM-judge) → Task 4; design §1b row 2 (traverse slice-1 edges, transitive case) → Tasks 2/4; local-not-global (killed reduce) → honored (no reduce task); link-density caveat → Global Constraints + Task 4 Step 3. Covered.

**2. Placeholder scan:** every code step has real Go/Python; every run step has exact command + expected output. Field-name verification is flagged as an execution check, not a placeholder (the structs exist; names confirmed for `compatibleSidecar{note,sidecar}`, `scoredCandidate{basename,...}`, `matchedMember{basename,notePath,vector,sitVec,bodyVec,score,content}`).

**3. Type consistency:** `expandSeedsViaGraph(notes, seeds, hitByBasename, hops, capacity) []matchedMember` consistent across Tasks 1–2; `QueryArgs.GraphExpandHops`, `defaultGraphExpandHops`, `provenanceGraphExpanded` consistent; `BFSWithCap`/`BuildGraph` signatures match `internal/vaultgraph`.

**Known accepted risk:** the end-to-end proof depends on (a) the bridge being genuinely cosine-distant from the query and (b) the chain edges existing — both controlled by the fixture. If real-vault link density is low, the value is bounded (honest-caveat constraint); the deterministic query-level test still proves the *mechanism* works where edges exist.
