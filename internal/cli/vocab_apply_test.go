package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// ── Task 2.2: naming requests + names answers ────────────────────────────────

// TestBuildNamingRequests_ExemplarsAreCentroidNearest verifies each naming
// request carries the cluster's centroid-nearest member notes (title +
// snippet), nearest first, capped at the exemplar limit.
func TestBuildNamingRequests_ExemplarsAreCentroidNearest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Cluster centroid [1,0]; near [1,0.05], mid [0.8,0.6], far [0.1,1].
	// far has no readable note file — its snippet must degrade to empty.
	near := "1aa.2026-01-01.near-note.md"
	mid := "1ab.2026-01-01.mid-note.md"
	far := "1ac.2026-01-01.far-note.md"
	noteBody := func(text string) []byte {
		return []byte("---\ntype: feedback\nsituation: ctx\ncreated: 2026-01-01\n---\n\n" + text + "\n")
	}

	files := map[string][]byte{
		"/vault/" + near: noteBody("Lesson learned: nearest exemplar text."),
		"/vault/" + mid:  noteBody("Lesson learned: middle exemplar text."),
	}
	deps := newCaptureVocabDeps(files, []string{near, mid, far}, map[string][]byte{})

	derivation := cli.ExportVocabDerivation{
		K:          1,
		Silhouette: 0.5,
		Clusters: []cli.ExportDerivedCluster{
			{Members: []string{far, near, mid}, Centroid: []float32{1, 0}},
		},
	}
	noteVecs := map[string][]float32{
		near: {1, 0.05},
		mid:  {0.8, 0.6},
		far:  {0.1, 1},
	}

	requests := cli.ExportBuildNamingRequests(deps, "/vault", derivation, []int{0}, noteVecs)

	g.Expect(requests).To(HaveLen(1))
	g.Expect(requests[0].Cluster).To(Equal(0))
	g.Expect(requests[0].MemberCount).To(Equal(3))
	g.Expect(requests[0].Exemplars).NotTo(BeEmpty())
	g.Expect(requests[0].Exemplars[0].Note).To(Equal(near), "nearest member first")
	g.Expect(requests[0].Exemplars[0].Title).To(Equal("near-note"))
	g.Expect(requests[0].Exemplars[0].Snippet).To(ContainSubstring("nearest exemplar text"))
	g.Expect(requests[0].Exemplars[2].Note).To(Equal(far), "farthest member last")
	g.Expect(requests[0].Exemplars[2].Snippet).To(BeEmpty(), "unreadable note degrades to an empty snippet")
}

// TestNamedClustersToSeedTerms verifies the mint-on-answer conversion carries
// term, description, and the cluster's exemplar snippets into the SeedTerm
// shape consumed by the existing mintDefinitionNote path.
func TestNamedClustersToSeedTerms(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	requests := []cli.ExportVocabNamingRequest{
		{Cluster: 2, MemberCount: 4, Exemplars: []cli.ExportVocabNamingExemplar{
			{Note: "1aa.x.md", Title: "x", Snippet: "snippet one"},
			{Note: "1ab.y.md", Title: "y", Snippet: "snippet two"},
		}},
	}
	names := map[int]cli.ExportVocabClusterName{
		2: {Cluster: 2, Term: "new-cluster-term", Description: "covers new things"},
	}

	seeds := cli.ExportNamedClustersToSeedTerms(requests, names)

	g.Expect(seeds).To(HaveLen(1))
	g.Expect(seeds[0].Term).To(Equal("new-cluster-term"))
	g.Expect(seeds[0].Description).To(Equal("covers new things"))
	g.Expect(seeds[0].Exemplars).To(Equal([]string{"snippet one", "snippet two"}))
}

// TestParseRefitNames_RejectsIncompleteAndUnknown verifies validation: every
// new cluster must be named, unknown cluster ids and empty terms are errors.
func TestParseRefitNames_RejectsIncompleteAndUnknown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, missingErr := cli.ExportParseRefitNames(
		[]byte(`{"names":[{"cluster":1,"term":"x","description":"d"}]}`), []int{1, 3}, "")
	g.Expect(missingErr).To(HaveOccurred(), "cluster 3 unnamed")

	_, unknownErr := cli.ExportParseRefitNames(
		[]byte(`{"names":[{"cluster":9,"term":"x","description":"d"}]}`), []int{1}, "")
	g.Expect(unknownErr).To(HaveOccurred(), "cluster 9 is not a new cluster")

	_, emptyErr := cli.ExportParseRefitNames(
		[]byte(`{"names":[{"cluster":1,"term":"","description":"d"}]}`), []int{1}, "")
	g.Expect(emptyErr).To(HaveOccurred(), "empty term rejected")

	_, badErr := cli.ExportParseRefitNames([]byte(`not json`), []int{1}, "")
	g.Expect(badErr).To(HaveOccurred(), "malformed JSON rejected")

	_, staleErr := cli.ExportParseRefitNames(
		[]byte(`{"fingerprint":"n8-old","names":[{"cluster":1,"term":"x","description":"d"}]}`),
		[]int{1}, "n9-new")
	g.Expect(staleErr).To(MatchError(ContainSubstring("re-run")), "stale fingerprint rejected")
}

// TestParseRefitNames_RoundTripProperty verifies that for any set of cluster
// ids with generated non-empty names, marshaling an answer doc and parsing it
// back yields exactly those names (parse accepts everything emit produces).
func TestParseRefitNames_RoundTripProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		clusterIDs := rapid.SliceOfNDistinct(rapid.IntRange(0, 30), 1, 8, rapid.ID).Draw(rt, "clusters")

		entries := make([]map[string]any, 0, len(clusterIDs))
		for i, clusterID := range clusterIDs {
			entries = append(entries, map[string]any{
				"cluster":     clusterID,
				"term":        fmt.Sprintf("term-%d", i),
				"description": fmt.Sprintf("description %d", i),
			})
		}

		data, marshalErr := json.Marshal(map[string]any{"names": entries})
		g.Expect(marshalErr).NotTo(HaveOccurred())

		parsed, parseErr := cli.ExportParseRefitNames(data, clusterIDs, "")
		g.Expect(parseErr).NotTo(HaveOccurred())
		g.Expect(parsed).To(HaveLen(len(clusterIDs)))

		for i, clusterID := range clusterIDs {
			g.Expect(parsed[clusterID].Term).To(Equal(fmt.Sprintf("term-%d", i)))
		}
	})
}

// TestParseRefitNames_ValidCoversAllClusters verifies a complete names answer
// parses into a per-cluster map.
func TestParseRefitNames_ValidCoversAllClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	data := []byte(`{"names":[
		{"cluster":1,"term":"alpha-things","description":"alpha stuff"},
		{"cluster":3,"term":"beta-things","description":"beta stuff"}
	]}`)

	names, err := cli.ExportParseRefitNames(data, []int{1, 3}, "")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(HaveLen(2))
	g.Expect(names[1].Term).To(Equal("alpha-things"))
	g.Expect(names[3].Description).To(Equal("beta stuff"))
}

// TestRemoveNoteReferences_FrontmatterlessContent verifies content without
// frontmatter still gets its body references scrubbed, and reference-free
// frontmatterless content reports unchanged.
func TestRemoveNoteReferences_FrontmatterlessContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deleted := map[string]bool{"5.2026-07-02.gone.md": true}

	scrubbed, changed := cli.ExportRemoveNoteReferences("see [[5.2026-07-02.gone]] here\n", deleted)
	g.Expect(changed).To(BeTrue())
	g.Expect(scrubbed).To(Equal("see 5.2026-07-02.gone here\n"), "inline link unlinked to plain text")

	same, unchanged := cli.ExportRemoveNoteReferences("nothing linked here\n", deleted)
	g.Expect(unchanged).To(BeFalse())
	g.Expect(same).To(Equal("nothing linked here\n"))
}

// TestRetireVocabTerms_DeletesDefinitionNoteAndSidecar verifies a retired
// term's definition note and its .vec.json sidecar are deleted outright (no
// demoted note left behind), while member notes lose the vocab/<term> tag and
// keep unrelated tags.
func TestRetireVocabTerms_DeletesDefinitionNoteAndSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := retirementFixtureFiles()
	written := map[string][]byte{}

	var deleted []string

	deps := newCaptureVocabDeps(files, names, written)
	deps.DeleteFile = func(path string) error { deleted = append(deleted, path); return nil }

	cli.ExportRetireVocabTerms(deps, "/vault", []string{retiredTermName})

	g.Expect(deleted).To(ContainElement(retiredDefPath), "definition note deleted")
	g.Expect(deleted).To(ContainElement(embed.SidecarPath(retiredDefPath)), "sidecar deleted")
	g.Expect(written).NotTo(HaveKey(retiredDefPath), "no demoted note written in place")

	updatedMember := string(written[retireMemberPath])
	g.Expect(updatedMember).NotTo(ContainSubstring("vocab/old-term"), "member tag stripped")
	g.Expect(updatedMember).To(ContainSubstring("keep-me"), "unrelated tags preserved")
}

// TestRetireVocabTerms_ListError_Warns verifies a vault-listing failure aborts
// retirement with a warning and no writes or deletes.
func TestRetireVocabTerms_ListError_Warns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}

	var (
		warnings []string
		deleted  []string
	)

	deps := newCaptureVocabDeps(map[string][]byte{}, nil, written)
	deps.ListMD = func(string) ([]string, error) { return nil, errors.New("list boom") }
	deps.DeleteFile = func(path string) error { deleted = append(deleted, path); return nil }
	deps.LogWarning = func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	cli.ExportRetireVocabTerms(deps, "/vault", []string{retiredTermName})

	g.Expect(warnings).NotTo(BeEmpty())
	g.Expect(written).To(BeEmpty())
	g.Expect(deleted).To(BeEmpty())
}

// TestRetireVocabTerms_NoSupersessionRecorded verifies retirement records no
// supersession anywhere: vocab definition notes have no supersession story —
// the family note gains no supersedes entry and no Supersedes body line.
func TestRetireVocabTerms_NoSupersessionRecorded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := retirementFixtureFiles()
	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	cli.ExportRetireVocabTerms(deps, "/vault", []string{retiredTermName})

	if familyRaw, rewritten := written[retireFamilyPath]; rewritten {
		family := string(familyRaw)
		g.Expect(family).NotTo(ContainSubstring("supersedes:"), "no frontmatter supersedes list")
		g.Expect(family).NotTo(ContainSubstring("Supersedes: [["), "no body supersedes line")
	}
}

// TestRetireVocabTerms_ScrubDropsEmptySupersedesKey verifies the frontmatter
// supersedes: key itself is dropped when the scrub removes its last entry.
func TestRetireVocabTerms_ScrubDropsEmptySupersedesKey(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := retirementFixtureFiles()
	referrer := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: x\npredicate: covers\nobject: o\n" +
		"luhmann: \"7\"\ncreated: \"2026-01-01\"\nsource: test\nsupersedes:\n" +
		"    - note: " + retiredDefName + "\n      type: updates\n      claim: old term retired\n" +
		"tags:\n    - keep-me\n---\n\nInformation learned: x covers o.\n\n" +
		"Supersedes: [[" + retiredDefName + "]] — updates: old term retired\n"

	const referrerName = "7.2026-01-01.referrer.md"

	files["/vault/"+referrerName] = []byte(referrer)
	names = append(names, referrerName)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	cli.ExportRetireVocabTerms(deps, "/vault", []string{retiredTermName})

	updatedReferrer := string(written["/vault/"+referrerName])
	g.Expect(updatedReferrer).NotTo(BeEmpty(), "referrer must be rewritten")
	g.Expect(updatedReferrer).NotTo(ContainSubstring("supersedes:"), "empty supersedes key dropped")
	g.Expect(updatedReferrer).NotTo(ContainSubstring("Supersedes: [["), "body line removed")
	g.Expect(updatedReferrer).To(ContainSubstring("tags:"), "rest of frontmatter intact")
}

// TestRetireVocabTerms_ScrubsReferencesVaultWide verifies every reference to a
// deleted definition note is removed: a frontmatter supersedes entry naming
// its basename, the matching Supersedes body line (with .md inside the
// wikilink), and an inline wikilink (without .md) elsewhere — while notes
// without references stay untouched (not rewritten at all) and a scrubbed
// note's content hash is unchanged when only hash-excluded lines were removed.
func TestRetireVocabTerms_ScrubsReferencesVaultWide(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := retirementFixtureFiles()
	referrer := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: x\npredicate: covers\nobject: o\n" +
		"luhmann: \"7\"\ncreated: \"2026-01-01\"\nsource: test\nsupersedes:\n" +
		"    - note: " + retiredDefName + "\n      type: updates\n      claim: old term retired\n" +
		"    - note: 8.2026-01-01.unrelated.md\n      type: narrows\n      claim: keep this one\n" +
		"tags:\n    - keep-me\n---\n\nInformation learned: x covers o.\n\n" +
		"Supersedes: [[" + retiredDefName + "]] — updates: old term retired\n" +
		"Supersedes: [[8.2026-01-01.unrelated]] — narrows: keep this one\n"
	inlineLinker := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1ab\"\ncreated: 2026-01-01\nsource: test\ntags:\n    - keep-me\n---\n\n" +
		"Lesson learned: see [[" + strings.TrimSuffix(retiredDefName, ".md") + "]] for the old term.\n"
	untouched := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1ac\"\ncreated: 2026-01-01\nsource: test\ntags:\n    - keep-me\n---\n\n" +
		"Lesson learned: nothing to see here.\n"

	const (
		referrerName  = "7.2026-01-01.referrer.md"
		inlineName    = "1ab.2026-01-01.inline-linker.md"
		untouchedName = "1ac.2026-01-01.untouched.md"
	)

	files["/vault/"+referrerName] = []byte(referrer)
	files["/vault/"+inlineName] = []byte(inlineLinker)
	files["/vault/"+untouchedName] = []byte(untouched)
	names = append(names, referrerName, inlineName, untouchedName)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	cli.ExportRetireVocabTerms(deps, "/vault", []string{retiredTermName})

	updatedReferrer := string(written["/vault/"+referrerName])
	g.Expect(updatedReferrer).NotTo(BeEmpty(), "referrer must be rewritten")
	g.Expect(updatedReferrer).NotTo(ContainSubstring(retiredDefName), "frontmatter entry + body line removed")
	g.Expect(updatedReferrer).To(ContainSubstring("note: 8.2026-01-01.unrelated.md"), "other entries kept")
	g.Expect(updatedReferrer).To(ContainSubstring("Supersedes: [[8.2026-01-01.unrelated]]"), "other body lines kept")
	g.Expect(embed.ContentHash([]byte(updatedReferrer))).To(Equal(embed.ContentHash(files["/vault/"+referrerName])),
		"removing hash-excluded supersedes machinery must not change the content hash")

	updatedInline := string(written["/vault/"+inlineName])
	g.Expect(updatedInline).NotTo(BeEmpty(), "inline linker must be rewritten")
	g.Expect(updatedInline).NotTo(ContainSubstring("[["+strings.TrimSuffix(retiredDefName, ".md")+"]]"),
		"inline wikilink removed")
	g.Expect(updatedInline).To(ContainSubstring(strings.TrimSuffix(retiredDefName, ".md")),
		"unlinked plain text preserved")

	g.Expect(written).NotTo(HaveKey("/vault/"+untouchedName), "reference-free notes stay byte-identical")
}

// TestRetireVocabTerms_WriteErrors_Warn verifies delete and scrub-write
// failures are warned about, not fatal.
func TestRetireVocabTerms_WriteErrors_Warn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := retirementFixtureFiles()

	var warnings []string

	deps := newCaptureVocabDeps(files, names, map[string][]byte{})
	deps.DeleteFile = func(string) error { return errors.New("delete boom") }
	deps.WriteFile = func(string, []byte) error { return errors.New("write boom") }
	deps.LogWarning = func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	cli.ExportRetireVocabTerms(deps, "/vault", []string{retiredTermName})

	g.Expect(warnings).NotTo(BeEmpty(), "delete/write failures must be surfaced as warnings")
}

// TestRunVocabPropose_StampCarriesSidecarVector verifies that when the minted
// definition note is embedded, the proposed term's centroid entry carries the
// sidecar's body vector.
func TestRunVocabPropose_StampCarriesSidecarVector(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(map[string][]byte{}, nil, written)
	deps.Embedder = stubEmbedder{modelID: "test", dims: 4}

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "scattered-concept", Description: "a scattered concept"}

	var buf strings.Builder

	g.Expect(cli.RunVocabPropose(t.Context(), args, deps, &buf)).To(Succeed())

	doc := requireWrittenCentroidsDoc(g, written)
	g.Expect(doc.Terms["scattered-concept"].Origin).To(Equal(cli.ExportVocabOriginProposed))
	g.Expect(doc.Terms["scattered-concept"].Vector).NotTo(BeEmpty(),
		"the embedded definition's body vector must be carried into the centroid entry")
}

// TestRunVocabPropose_StampPreservesExistingCentroidEntries verifies stamping
// merges into an existing centroids doc without clobbering other terms or
// lifecycle fields.
func TestRunVocabPropose_StampPreservesExistingCentroidEntries(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existingDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		Terms: map[string]cli.ExportVocabCentroidEntry{
			"alpha": {Vector: []float32{1, 0}, MemberCount: 3, Origin: cli.ExportVocabOriginDerived},
		},
		LastRefit: &cli.ExportVocabLastRefitDoc{NoteCount: 4, Date: "2026-07-01"},
	}
	docJSON, marshalErr := json.Marshal(existingDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	files := map[string][]byte{"/vault/vocab.centroids.json": docJSON}
	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, nil, written)

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "scattered-concept", Description: "a scattered concept"}

	var buf strings.Builder

	g.Expect(cli.RunVocabPropose(t.Context(), args, deps, &buf)).To(Succeed())

	doc := requireWrittenCentroidsDoc(g, written)
	g.Expect(doc.Terms["alpha"].Origin).To(Equal(cli.ExportVocabOriginDerived), "existing entries preserved")
	g.Expect(doc.Terms["alpha"].MemberCount).To(Equal(3))
	g.Expect(doc.Terms["scattered-concept"].Origin).To(Equal(cli.ExportVocabOriginProposed))
	g.Expect(doc.LastRefit).NotTo(BeNil(), "lifecycle fields preserved")
}

// TestRunVocabPropose_StampSurvivesMintFailure verifies the origin stamp is
// still written (vector-less) when the definition note itself failed to mint.
func TestRunVocabPropose_StampSurvivesMintFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(map[string][]byte{}, nil, written)
	deps.WriteFile = func(path string, data []byte) error {
		if strings.HasSuffix(path, ".md") {
			return errors.New("mint boom")
		}

		written[path] = data

		return nil
	}

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "scattered-concept", Description: "a scattered concept"}

	var buf strings.Builder

	g.Expect(cli.RunVocabPropose(t.Context(), args, deps, &buf)).To(Succeed())

	doc := requireWrittenCentroidsDoc(g, written)
	g.Expect(doc.Terms["scattered-concept"].Origin).To(Equal(cli.ExportVocabOriginProposed))
	g.Expect(doc.Terms["scattered-concept"].Vector).To(BeEmpty())
}

// ── Task 2.4: propose stamps origin: proposed ────────────────────────────────

// TestRunVocabPropose_StampsProposedOriginInCentroids verifies that `vocab
// propose` writes an origin: proposed entry for the minted term into
// vocab.centroids.json, creating the file when absent.
func TestRunVocabPropose_StampsProposedOriginInCentroids(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(map[string][]byte{}, nil, written)

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "scattered-concept", Description: "a scattered concept"}

	var buf strings.Builder

	g.Expect(cli.RunVocabPropose(t.Context(), args, deps, &buf)).To(Succeed())

	doc := requireWrittenCentroidsDoc(g, written)
	g.Expect(doc.Terms).To(HaveKey("scattered-concept"))
	g.Expect(doc.Terms["scattered-concept"].Origin).To(Equal(cli.ExportVocabOriginProposed))
}

// TestRunVocabRefit_AppliesDerivationWithNames verifies the full apply path:
// matched term kept, new cluster minted from the names answer, unmatched
// derived term retired, proposed term preserved, centroids re-derived with
// origin + derivation metadata, members re-tagged, and version bumped.
func TestRunVocabRefit_AppliesDerivationWithNames(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := refitFixture(g)
	refitNamesFileFromRequests(t, g, files, names)

	written := map[string][]byte{}
	lockCalls := 0
	deps := newCaptureVocabDeps(files, names, written)
	deps.Lock = func(string) (func(), error) { lockCalls++; return func() {}, nil }

	var buf strings.Builder

	args := cli.VocabRefitArgs{Vault: "/vault", NamesFile: "/names.json"}
	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed())
	g.Expect(lockCalls).To(BeNumerically(">=", 1), "refit must hold the vault lock")

	// New term minted with the definition-note tag convention.
	var mintedNorth string

	for path, data := range written {
		if strings.Contains(path, "vocab-north-term-definition.md") {
			mintedNorth = string(data)
		}
	}

	g.Expect(mintedNorth).NotTo(BeEmpty(), "north-term definition note must be minted")
	g.Expect(mintedNorth).To(ContainSubstring("- vocab\n"), "bare vocab marker")
	g.Expect(mintedNorth).To(ContainSubstring("- vocab/north-term"), "self-tag")

	// Retired term's definition note deleted outright; proposed term untouched.
	g.Expect(written).NotTo(HaveKey("/vault/211.2026-01-01.vocab-stale-term-definition.md"),
		"stale-term definition must be deleted, not rewritten")
	g.Expect(written).NotTo(HaveKey("/vault/212.2026-01-01.vocab-prop-term-definition.md"),
		"proposed term's definition note must not be touched")

	// Centroids: derived origins, proposed carried, retired gone, metadata set.
	doc := requireWrittenCentroidsDoc(g, written)
	g.Expect(doc.Terms).To(HaveKey("east-term"))
	g.Expect(doc.Terms["east-term"].Origin).To(Equal(cli.ExportVocabOriginDerived),
		"pre-existing term defaults to derived on first derivation")
	g.Expect(doc.Terms["east-term"].MemberCount).To(Equal(4))
	g.Expect(doc.Terms).To(HaveKey("north-term"))
	g.Expect(doc.Terms["north-term"].Origin).To(Equal(cli.ExportVocabOriginDerived))
	g.Expect(doc.Terms).To(HaveKey("prop-term"))
	g.Expect(doc.Terms["prop-term"].Origin).To(Equal(cli.ExportVocabOriginProposed))
	g.Expect(doc.Terms).NotTo(HaveKey("stale-term"))
	g.Expect(doc.Derivation).NotTo(BeNil())

	if doc.Derivation == nil {
		return
	}

	g.Expect(doc.Derivation.K).To(Equal(2))
	g.Expect(doc.LastRefit).NotTo(BeNil())
	g.Expect(doc.RefitPending).To(BeFalse())

	// Members re-tagged against the derived centroids.
	eastNote := string(written["/vault/1ae0.2026-01-01.e-note-0.md"])
	g.Expect(eastNote).To(ContainSubstring("vocab/east-term"))

	// Major version bump persisted on the family note.
	family := string(written["/vault/9.2026-01-01.vocab-definition.md"])
	g.Expect(family).To(ContainSubstring(`vocab_version: "7.0"`))
}

// TestRunVocabRefit_BadNamesFile_Errors verifies an unreadable or invalid
// --names answer fails loud without writing.
func TestRunVocabRefit_BadNamesFile_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := refitFixture(g)
	files["/names.json"] = []byte(`{"names":[]}`) // incomplete: north cluster unnamed

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	var buf strings.Builder

	args := cli.VocabRefitArgs{Vault: "/vault", NamesFile: "/names.json"}
	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).NotTo(Succeed())
	g.Expect(written).To(BeEmpty(), "a rejected names answer must not write")
}

// TestRunVocabRefit_DryRun_PrintsDiffWithoutWrites verifies --dry-run prints
// the derivation diff (K, silhouette, matched/new/retired) and writes nothing.
func TestRunVocabRefit_DryRun_PrintsDiffWithoutWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := refitFixture(g)
	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	var buf strings.Builder

	refitErr := cli.RunVocabRefit(t.Context(), cli.VocabRefitArgs{Vault: "/vault", DryRun: true}, deps, &buf)
	g.Expect(refitErr).NotTo(HaveOccurred())

	out := buf.String()
	g.Expect(out).To(ContainSubstring("dry-run"))
	g.Expect(out).To(ContainSubstring("K=2"))
	g.Expect(out).To(ContainSubstring("silhouette="))
	g.Expect(out).To(ContainSubstring("east-term"), "matched term listed")
	g.Expect(out).To(ContainSubstring("stale-term"), "retired term listed")
	g.Expect(written).To(BeEmpty(), "dry-run must not write")
}

// TestRunVocabRefit_EmitsNamingRequestsWithoutWrites verifies a refit with
// unmatched clusters and no --names emits structured naming requests (with
// exemplars) and defers all writes.
func TestRunVocabRefit_EmitsNamingRequestsWithoutWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := refitFixture(g)
	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), cli.VocabRefitArgs{Vault: "/vault"}, deps, &buf)).To(Succeed())

	var requestsDoc struct {
		NamingRequests []cli.ExportVocabNamingRequest `json:"naming_requests"` //nolint:tagliatelle // contract key
		Instruction    string                         `json:"instruction"`
		Fingerprint    string                         `json:"fingerprint"`
	}

	g.Expect(json.Unmarshal([]byte(buf.String()), &requestsDoc)).To(Succeed())
	g.Expect(requestsDoc.NamingRequests).To(HaveLen(1))
	g.Expect(requestsDoc.NamingRequests[0].Exemplars).NotTo(BeEmpty(), "exemplars carried")
	g.Expect(requestsDoc.NamingRequests[0].Exemplars[0].Note).To(ContainSubstring("n-note"),
		"exemplars come from the unmatched (north) cluster's members")
	g.Expect(requestsDoc.Instruction).NotTo(BeEmpty())
	g.Expect(requestsDoc.Fingerprint).NotTo(BeEmpty(),
		"the payload must carry the derivation-input fingerprint for the --names run to echo")
	g.Expect(written).To(BeEmpty(), "naming emission must not write")
}

// TestRunVocabRefit_NoStructure_NoWrites verifies a vault too small for
// clustering leaves the vocabulary untouched.
func TestRunVocabRefit_NoStructure_NoWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	sidecar := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: "test", Dims: 2,
		BodyVector: []float32{1, 0}, SituationVector: make([]float32, 2),
	})
	files := map[string][]byte{
		"/vault/1aa.2026-01-01.only.md": []byte("---\ntype: feedback\nsituation: ctx\ncreated: 2026-01-01\n" +
			"---\n\nLesson learned: only note.\n"),
		"/vault/1aa.2026-01-01.only.vec.json": sidecar,
	}
	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, []string{"1aa.2026-01-01.only.md"}, written)

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), cli.VocabRefitArgs{Vault: "/vault"}, deps, &buf)).To(Succeed())
	g.Expect(buf.String()).To(ContainSubstring("no structure"))
	g.Expect(written).To(BeEmpty())
}

// TestRunVocabRefit_VaultChangedBetweenRuns_FailsFingerprint verifies the
// two-run naming protocol fails loudly when the vault drifted between the
// naming-request emission and the --names apply run: cluster indices may no
// longer mean what the agent named, so the answer must be rejected (with a
// re-run instruction) and nothing written.
func TestRunVocabRefit_VaultChangedBetweenRuns_FailsFingerprint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := refitFixture(g)
	refitNamesFileFromRequests(t, g, files, names)

	// Vault drifts after the emit run: a concurrent learn adds a member note.
	driftNote := "1az0.2026-01-02.drift-note.md"
	files["/vault/"+driftNote] = []byte("---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\n" +
		"action: a\nluhmann: \"1az0\"\ncreated: 2026-01-02\nsource: test\n---\n\nLesson learned: drift.\n")
	files["/vault/1az0.2026-01-02.drift-note.vec.json"] = embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: "test", Dims: 2,
		BodyVector: []float32{0.7, 0.7}, SituationVector: make([]float32, 2),
	})

	names = append(names, driftNote)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	var buf strings.Builder

	args := cli.VocabRefitArgs{Vault: "/vault", NamesFile: "/names.json"}
	refitErr := cli.RunVocabRefit(t.Context(), args, deps, &buf)
	g.Expect(refitErr).To(MatchError(ContainSubstring("re-run")),
		"a drifted vault must reject the stale names answer and tell the user to re-emit")
	g.Expect(written).To(BeEmpty(), "a rejected stale answer must not write")
}

// TestWriteCentroidsFile_OriginAndDerivationAreAdditive verifies the new
// fields are omitted when unset (old-binary-readable additive schema), and
// that a pre-change document (no origin, no derivation) still parses.
func TestWriteCentroidsFile_OriginAndDerivationAreAdditive(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(map[string][]byte{}, nil, written)

	entries := map[string]cli.ExportVocabCentroidEntry{
		"legacy": {Vector: []float32{1, 0}, MemberCount: 1},
	}

	cli.ExportWriteCentroidsFile(deps, "/vault", entries, nil, nil)

	raw := written["/vault/vocab.centroids.json"]
	g.Expect(string(raw)).NotTo(ContainSubstring(`"origin"`), "unset origin must be omitted")
	g.Expect(string(raw)).NotTo(ContainSubstring(`"derivation"`), "unset derivation must be omitted")

	// A pre-change doc (no origin/derivation keys) still parses cleanly.
	oldDoc := []byte(`{"schema_version":1,"embedding_model_id":"test","dims":2,` +
		`"terms":{"alpha":{"vector":[1,0],"member_count":3}}}`)

	files := map[string][]byte{"/vault/vocab.centroids.json": oldDoc}

	doc, ok := cli.ExportReadCentroidsDoc("/vault", func(path string) ([]byte, error) {
		if data, present := files[path]; present {
			return data, nil
		}

		return nil, &testNotFoundError{path: path}
	})
	g.Expect(ok).To(BeTrue())
	g.Expect(doc.Terms["alpha"].Origin).To(BeEmpty(), "absent origin decodes to empty (defaulted later)")
	g.Expect(doc.Derivation).To(BeNil())
}

// TestWriteCentroidsFile_PersistsOriginAndDerivationMetadata verifies that a
// derivation-produced centroids write records each term's origin, the
// derivation metadata (K, silhouette, date), and the last_refit baseline.
func TestWriteCentroidsFile_PersistsOriginAndDerivationMetadata(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(map[string][]byte{}, nil, written)

	entries := map[string]cli.ExportVocabCentroidEntry{
		"alpha": {Vector: []float32{1, 0}, MemberCount: 3, Origin: cli.ExportVocabOriginDerived},
		"beta":  {Vector: []float32{0, 1}, MemberCount: 2, Origin: cli.ExportVocabOriginProposed},
	}
	lastRefit := &cli.ExportVocabLastRefitDoc{NoteCount: 5, Date: "2026-07-28"}
	derivation := &cli.ExportVocabDerivationMeta{K: 2, Silhouette: 0.42, Date: "2026-07-28"}

	cli.ExportWriteCentroidsFile(deps, "/vault", entries, lastRefit, derivation)

	raw, ok := written["/vault/vocab.centroids.json"]
	g.Expect(ok).To(BeTrue(), "centroids file must be written")

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(raw, &doc)).To(Succeed())
	g.Expect(doc.Terms["alpha"].Origin).To(Equal(cli.ExportVocabOriginDerived))
	g.Expect(doc.Terms["beta"].Origin).To(Equal(cli.ExportVocabOriginProposed))
	g.Expect(doc.Derivation).NotTo(BeNil())

	if doc.Derivation == nil {
		return
	}

	g.Expect(doc.Derivation.K).To(Equal(2))
	g.Expect(doc.Derivation.Silhouette).To(BeNumerically("~", 0.42, 1e-9))
	g.Expect(doc.Derivation.Date).To(Equal("2026-07-28"))
	g.Expect(doc.LastRefit).NotTo(BeNil())
	g.Expect(doc.RefitPending).To(BeFalse(), "a fresh derivation clears refit_pending")
}

// unexported constants.
const (
	retireFamilyName = "9.2026-01-01.vocab-definition.md"
	retireFamilyPath = "/vault/" + retireFamilyName
	retireMemberName = "1aa.2026-01-01.member.md"
	retireMemberPath = "/vault/" + retireMemberName
	retiredDefName   = "5.2026-07-02.vocab-old-term-definition.md"
	retiredDefPath   = "/vault/" + retiredDefName
	retiredTermName  = "old-term"
)

// unexported variables.
var (
	applyTestNow = time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
)

// newCaptureVocabDeps builds VocabDeps over an in-memory file map, capturing
// writes into written. names is the ListMD listing.
func newCaptureVocabDeps(files map[string][]byte, names []string, written map[string][]byte) cli.VocabDeps {
	return cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := written[path]; ok {
				return data, nil
			}

			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		DeleteFile:   func(string) error { return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return applyTestNow },
	}
}

// ── Task 2.5: RunVocabRefit derivation flow ──────────────────────────────────

// refitFixture builds a two-blob vault: 4 "east" notes near [1,0] and 4
// "north" notes near [0,1], with three existing terms — east-term (matches
// the east cluster), stale-term (matches nothing → retired), prop-term
// (origin: proposed, matches nothing → survives) — plus the family note.
func refitFixture(g Gomega) (map[string][]byte, []string) {
	sidecar := func(vec []float32) []byte {
		return embed.MarshalSidecar(embed.Sidecar{
			SchemaVersion: 1, EmbeddingModelID: "test", Dims: 2,
			BodyVector: vec, SituationVector: make([]float32, 2),
		})
	}
	memberNote := func(id string) string {
		return "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
			"luhmann: \"" + id + "\"\ncreated: 2026-01-01\nsource: test\n---\n\nLesson learned: when ctx, a.\n"
	}
	definitionNote := func(id, term string) string {
		return "---\ntype: fact\ntier: L2\nsituation: s\nsubject: the " + term + " vocab term\n" +
			"predicate: covers\nobject: desc\nluhmann: \"" + id + "\"\ncreated: \"2026-01-01\"\nsource: test\n" +
			"tags:\n    - vocab\n    - vocab/" + term + "\n---\n\nInformation learned: the " + term +
			" vocab term covers desc.\n"
	}
	familyNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: the vocab tag family\n" +
		"predicate: covers\nobject: the tags convention\nluhmann: \"9\"\ncreated: \"2026-01-01\"\nsource: test\n" +
		"vocab_version: \"6.0\"\ntags:\n    - vocab\n---\n\nInformation learned: the vocab tag family covers stuff.\n"

	files := map[string][]byte{}
	names := []string{}
	blob := func(prefix string, base []float32) {
		for i := range 4 {
			name := fmt.Sprintf("1a%s%d.2026-01-01.%s-note-%d.md", prefix, i, prefix, i)
			vec := []float32{base[0] + float32(i)*0.02, base[1] + float32(3-i)*0.02}
			files["/vault/"+name] = []byte(memberNote(fmt.Sprintf("1a%s%d", prefix, i)))
			files["/vault/"+strings.TrimSuffix(name, ".md")+".vec.json"] = sidecar(vec)
			names = append(names, name)
		}
	}
	blob("e", []float32{1, 0})
	blob("n", []float32{0, 1})

	defs := map[string][]float32{
		"east-term":  {1, 0},
		"stale-term": {-1, 0},
		"prop-term":  {0, -1},
	}
	defIDs := map[string]string{"east-term": "210", "stale-term": "211", "prop-term": "212"}

	for term, vec := range defs {
		name := defIDs[term] + ".2026-01-01.vocab-" + term + "-definition.md"
		files["/vault/"+name] = []byte(definitionNote(defIDs[term], term))
		files["/vault/"+strings.TrimSuffix(name, ".md")+".vec.json"] = sidecar(vec)

		names = append(names, name)
	}

	files["/vault/9.2026-01-01.vocab-definition.md"] = []byte(familyNote)

	names = append(names, "9.2026-01-01.vocab-definition.md")

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1, EmbeddingModelID: "test", Dims: 2,
		Terms: map[string]cli.ExportVocabCentroidEntry{
			"east-term":  {Vector: []float32{1, 0}, MemberCount: 4},
			"stale-term": {Vector: []float32{-1, 0}, MemberCount: 1},
			"prop-term":  {Vector: []float32{0, -1}, MemberCount: 1, Origin: cli.ExportVocabOriginProposed},
		},
		LastRefit: &cli.ExportVocabLastRefitDoc{NoteCount: 8, Date: "2026-07-01"},
	}
	docJSON, marshalErr := json.Marshal(centroidsDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	files["/vault/vocab.centroids.json"] = docJSON

	return files, names
}

// refitNamesFileFromRequests runs the naming-emission phase and builds a
// --names answer file naming every requested cluster "north-term".
func refitNamesFileFromRequests(t *testing.T, g Gomega, files map[string][]byte, names []string) {
	t.Helper()

	written := map[string][]byte{}
	deps := newCaptureVocabDeps(files, names, written)

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), cli.VocabRefitArgs{Vault: "/vault"}, deps, &buf)).To(Succeed())

	var requestsDoc struct {
		NamingRequests []cli.ExportVocabNamingRequest `json:"naming_requests"` //nolint:tagliatelle // contract key
		Fingerprint    string                         `json:"fingerprint"`
	}

	g.Expect(json.Unmarshal([]byte(buf.String()), &requestsDoc)).To(Succeed())
	g.Expect(requestsDoc.NamingRequests).To(HaveLen(1), "the north blob is one unmatched cluster")

	answer := map[string]any{
		"fingerprint": requestsDoc.Fingerprint,
		"names": []map[string]any{{
			"cluster":     requestsDoc.NamingRequests[0].Cluster,
			"term":        "north-term",
			"description": "covers north things",
		}},
	}
	answerJSON, marshalErr := json.Marshal(answer)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	files["/names.json"] = answerJSON
}

// requireWrittenCentroidsDoc unmarshals the captured centroids write.
func requireWrittenCentroidsDoc(g Gomega, written map[string][]byte) cli.ExportVocabCentroidsDoc {
	raw, ok := written["/vault/vocab.centroids.json"]
	g.Expect(ok).To(BeTrue(), "vocab.centroids.json must be written")

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(raw, &doc)).To(Succeed())

	return doc
}

// retirementFixtureFiles builds a vault with one retired-term definition
// note, the vocab-definition family note, and one member note tagged with the
// retired term plus an unrelated tag.
func retirementFixtureFiles() (map[string][]byte, []string) {
	definitionNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: the old-term vocab term\n" +
		"predicate: covers\nobject: desc\nluhmann: \"5\"\ncreated: \"2026-07-02\"\nsource: test\n" +
		"tags:\n    - vocab\n    - vocab/old-term\n---\n\nInformation learned: the old-term vocab term covers desc.\n"
	familyNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: the vocab tag family\n" +
		"predicate: covers\nobject: the tags convention\nluhmann: \"9\"\ncreated: \"2026-01-01\"\nsource: test\n" +
		"vocab_version: \"6.0\"\ntags:\n    - vocab\n---\n\nInformation learned: the vocab tag family covers stuff.\n"
	memberNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\ntags:\n    - keep-me\n    - vocab/old-term\n---\n\n" +
		"Lesson learned: when ctx, a.\n"

	files := map[string][]byte{
		retiredDefPath:   []byte(definitionNote),
		retireFamilyPath: []byte(familyNote),
		retireMemberPath: []byte(memberNote),
	}

	return files, []string{retiredDefName, retireFamilyName, retireMemberName}
}
