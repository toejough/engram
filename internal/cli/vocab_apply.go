package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// vocabNamingExemplarCount is how many centroid-nearest member notes a
	// naming request carries per new cluster (design risk mitigation: naming
	// quality depends on good exemplars, so nearest — not random — members).
	vocabNamingExemplarCount = 3
	// vocabNamingSnippetMaxLen caps an exemplar snippet's length in bytes.
	vocabNamingSnippetMaxLen = 200
	vocabRefitDeriveSeed     = uint64(1)
)

// unexported variables.
var (
	errVocabRefitBadNames            = errors.New("vocab refit: cannot parse --names JSON")
	errVocabRefitNameEmpty           = errors.New("vocab refit: --names entry needs non-empty term and description")
	errVocabRefitNamesDuplicate      = errors.New("vocab refit: --names names the same cluster twice")
	errVocabRefitNamesIncomplete     = errors.New("vocab refit: --names must name every new cluster")
	errVocabRefitNamesUnknownCluster = errors.New("vocab refit: --names references an unknown cluster")
	errVocabRefitStaleNames          = errors.New(
		"vocab refit: the vault changed since the naming requests were emitted; " +
			"re-run `engram vocab refit` to regenerate them")
)

// vocabClusterName is one entry in the --names answer: the cluster index from
// the naming request plus the chosen kebab-case term and its description.
type vocabClusterName struct {
	Cluster     int    `json:"cluster"`
	Term        string `json:"term"`
	Description string `json:"description"`
}

// vocabNamingExemplar is one centroid-nearest member note in a naming
// request: its basename, filename-derived title, and a body snippet.
type vocabNamingExemplar struct {
	Note    string `json:"note"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// vocabNamingRequest asks the agent-side LLM to name one unmatched derived
// cluster, identified by its cluster index within the current derivation.
//
//nolint:tagliatelle // naming-request JSON keys follow the vault's snake_case contract
type vocabNamingRequest struct {
	Cluster     int                   `json:"cluster"`
	MemberCount int                   `json:"member_count"`
	Exemplars   []vocabNamingExemplar `json:"exemplars"`
}

// vocabNamingRequestsDoc is the stdout payload emitted when a derivation
// produced clusters that need naming.
//
//nolint:tagliatelle // naming-request JSON keys follow the vault's snake_case contract
type vocabNamingRequestsDoc struct {
	NamingRequests []vocabNamingRequest `json:"naming_requests"`
	Instruction    string               `json:"instruction"`
	// Fingerprint hashes the derivation inputs; the --names answer must echo
	// it so a vault that drifted between the two runs is rejected loudly.
	Fingerprint string `json:"fingerprint"`
}

// vocabRefitNamesDoc is the parsed shape of the --names answer JSON. The
// echoed fingerprint proves the answer names THIS vault state's clusters.
type vocabRefitNamesDoc struct {
	Names       []vocabClusterName `json:"names"`
	Fingerprint string             `json:"fingerprint"`
}

// vocabRefitState carries one derivation's inputs and outputs through the
// refit phases (diff printing, naming emission, apply). fingerprint hashes
// the derivation inputs — the two-run naming protocol is only sound while
// the vault is unchanged between the emit run and the --names run.
type vocabRefitState struct {
	noteVecs    map[string][]float32
	existing    []existingVocabTerm
	derivation  vocabDerivation
	match       vocabMatchResult
	fingerprint string
}

// applyVocabDerivation executes the derivation against the vault: major
// version bump, retirement of unmatched derived terms, minting of named new
// clusters, a full re-tag pass against the derived centroids, and the
// centroids-file write (origins + derivation metadata + last_refit;
// refit_pending cleared).
func applyVocabDerivation(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	state vocabRefitState,
	requests []vocabNamingRequest,
	names map[int]vocabClusterName,
	stdout io.Writer,
) {
	when := deps.Now()
	newVersion := bumpAndPersistVocabVersion(deps, vault, bumpMajorVersion, "vocab refit")

	retireVocabTerms(deps, vault, state.match.RetiredTerms)

	vaultNames, _ := deps.ListMD(vault)
	applyRefitNewTerms(ctx, deps, vault, &vaultNames, namedClustersToSeedTerms(requests, names), newVersion, when)

	entries, termVectors := derivedCentroidState(state, names)

	_, assignErr := assignTermsToAllNotes(deps, vault, termVectors, DefaultVocabFloor)
	if assignErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: re-tag pass: %v", assignErr)
	}

	refreshedNames, _ := deps.ListMD(vault)
	writeCentroidsFile(deps, vault, entries,
		buildLastRefitDoc(vault, refreshedNames, deps.ReadFile, when),
		&vocabDerivationMeta{
			K:          state.derivation.K,
			Silhouette: state.derivation.Silhouette,
			Date:       when.Format(dateFormat),
		})

	_, _ = fmt.Fprintf(stdout, "vocab refit applied: version → %s (K=%d, matched=%d, new=%d, retired=%d)\n",
		newVersion, state.derivation.K, len(state.match.Matched),
		len(state.match.NewClusters), len(state.match.RetiredTerms))
}

// buildNamingRequests builds one naming request per unmatched cluster: the
// cluster's member count plus its centroid-nearest members as exemplars
// (title from the filename slug, snippet from the note body).
func buildNamingRequests(
	deps VocabDeps,
	vault string,
	derivation vocabDerivation,
	newClusters []int,
	noteVecs map[string][]float32,
) []vocabNamingRequest {
	requests := make([]vocabNamingRequest, 0, len(newClusters))

	for _, clusterIdx := range newClusters {
		derived := derivation.Clusters[clusterIdx]

		requests = append(requests, vocabNamingRequest{
			Cluster:     clusterIdx,
			MemberCount: len(derived.Members),
			Exemplars:   clusterExemplars(deps, vault, derived, noteVecs),
		})
	}

	return requests
}

// clusterExemplars returns the centroid-nearest members of a cluster as
// naming exemplars, nearest first, capped at vocabNamingExemplarCount.
func clusterExemplars(
	deps VocabDeps,
	vault string,
	derived derivedCluster,
	noteVecs map[string][]float32,
) []vocabNamingExemplar {
	members := make([]string, len(derived.Members))
	copy(members, derived.Members)

	sort.SliceStable(members, func(i, j int) bool {
		return embed.Cosine(derived.Centroid, noteVecs[members[i]]) >
			embed.Cosine(derived.Centroid, noteVecs[members[j]])
	})

	count := min(vocabNamingExemplarCount, len(members))
	exemplars := make([]vocabNamingExemplar, 0, count)

	for _, name := range members[:count] {
		title := slugFromNoteFilename(name)
		if title == "" {
			title = name
		}

		exemplars = append(exemplars, vocabNamingExemplar{
			Note:    name,
			Title:   title,
			Snippet: noteSnippet(deps, vault, name),
		})
	}

	return exemplars
}

// collectSupersedesEntryBlocks groups the indented list-item lines following
// a supersedes: key into per-entry line blocks (each starting at a "- note:"
// line), returning the blocks and the index of the first line past the list.
func collectSupersedesEntryBlocks(lines []string, start int) ([][]string, int) {
	entries := make([][]string, 0)
	i := start

	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" || !strings.HasPrefix(line, " ") {
			break
		}

		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "- "):
			entries = append(entries, []string{line})
		case len(entries) > 0:
			entries[len(entries)-1] = append(entries[len(entries)-1], line)
		default:
			return entries, i
		}

		i++
	}

	return entries, i
}

// computeDerivationFingerprint hashes the derivation inputs (note count plus
// a SHA-256 over the sorted note names and their vector bytes). The naming
// emission stamps it into the payload; the --names run recomputes it and
// rejects a stale answer — cluster indices from a drifted vault may no
// longer mean what the agent named.
func computeDerivationFingerprint(notes []noteVector) string {
	hash := sha256.New()

	for _, note := range notes {
		_, _ = io.WriteString(hash, note.Name)
		_, _ = io.WriteString(hash, "\n")

		for _, value := range note.Vector {
			_, _ = fmt.Fprintf(hash, "%08x", math.Float32bits(value))
		}

		_, _ = io.WriteString(hash, "\n")
	}

	return fmt.Sprintf("n%d-%x", len(notes), hash.Sum(nil))
}

// deleteDefinitionNote removes a retired definition note and its embedding
// sidecar. Failures are warned about, not fatal — retirement proceeds so the
// re-tag pass still converges the vocabulary.
func deleteDefinitionNote(deps VocabDeps, notePath string) {
	deleteErr := deps.DeleteFile(notePath)
	if deleteErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: deleting %s: %v", notePath, deleteErr)
	}

	sidecarErr := deps.DeleteFile(embed.SidecarPath(notePath))
	if sidecarErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: deleting sidecar for %s: %v", notePath, sidecarErr)
	}
}

// deriveVaultVocab loads the derivation inputs (non-definition note vectors
// in sorted-name order for determinism; existing terms with origins) and runs
// clustering + name matching.
func deriveVaultVocab(deps VocabDeps, vault string) (vocabRefitState, error) {
	noteVecs := loadMemberNoteVectors(deps, vault)

	sortedNames := make([]string, 0, len(noteVecs))
	for name := range noteVecs {
		sortedNames = append(sortedNames, name)
	}

	sort.Strings(sortedNames)

	notes := make([]noteVector, 0, len(sortedNames))
	for _, name := range sortedNames {
		notes = append(notes, noteVector{Name: name, Vector: noteVecs[name]})
	}

	existing, loadErr := loadExistingTermsWithOrigins(deps, vault)
	if loadErr != nil {
		return vocabRefitState{}, loadErr
	}

	derivation, deriveErr := deriveVocabClusters(notes, previousDerivedK(existing), vocabRefitDeriveSeed)
	if deriveErr != nil {
		return vocabRefitState{}, fmt.Errorf("vocab refit: %w", deriveErr)
	}

	centroids := make([][]float32, len(derivation.Clusters))
	for i, derived := range derivation.Clusters {
		centroids[i] = derived.Centroid
	}

	return vocabRefitState{
		noteVecs:    noteVecs,
		existing:    existing,
		derivation:  derivation,
		match:       matchClustersToTerms(centroids, existing, vocabNameMatchThreshold),
		fingerprint: computeDerivationFingerprint(notes),
	}, nil
}

// derivedCentroidState builds the post-derivation term set: matched terms and
// named new clusters carry their cluster's mean centroid and member count
// with origin: derived; unmatched proposed terms are carried forward
// unchanged (origin: proposed — the provenance shield). Returns the centroid
// entries for the file write and the term vectors for the re-tag pass.
func derivedCentroidState(
	state vocabRefitState,
	names map[int]vocabClusterName,
) (map[string]vocabCentroidEntry, []TermWithVector) {
	entries := make(map[string]vocabCentroidEntry, state.derivation.K)
	termVectors := make([]TermWithVector, 0, state.derivation.K)

	addTerm := func(term string, clusterIdx int) {
		derived := state.derivation.Clusters[clusterIdx]
		entries[term] = vocabCentroidEntry{
			Vector:      derived.Centroid,
			MemberCount: len(derived.Members),
			Origin:      vocabOriginDerived,
		}
		termVectors = append(termVectors, TermWithVector{Term: term, Vector: derived.Centroid})
	}

	for _, matched := range state.match.Matched {
		addTerm(matched.Term, matched.ClusterIndex)
	}

	for _, clusterIdx := range state.match.NewClusters {
		if name, ok := names[clusterIdx]; ok {
			addTerm(name.Term, clusterIdx)
		}
	}

	matchedTerms := make(map[string]bool, len(state.match.Matched))
	for _, matched := range state.match.Matched {
		matchedTerms[matched.Term] = true
	}

	for _, term := range state.existing {
		if term.Origin != vocabOriginProposed || matchedTerms[term.Name] {
			continue
		}

		entries[term.Name] = vocabCentroidEntry{Vector: term.Vector, Origin: vocabOriginProposed}
		termVectors = append(termVectors, TermWithVector{Term: term.Name, Vector: term.Vector})
	}

	return entries, termVectors
}

// emitNamingRequests prints the naming-request payload as indented JSON,
// stamped with the derivation-input fingerprint the answer must echo.
func emitNamingRequests(stdout io.Writer, requests []vocabNamingRequest, fingerprint string) error {
	doc := vocabNamingRequestsDoc{
		NamingRequests: requests,
		Instruction: "Name each cluster from its exemplars: choose a kebab-case term and a " +
			"one-line description, then re-run `engram vocab refit --names <file>` where the file " +
			`is JSON {"names":[{"cluster":N,"term":"...","description":"..."}],"fingerprint":"..."} ` +
			"covering every cluster and echoing this payload's fingerprint verbatim.",
		Fingerprint: fingerprint,
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	encErr := enc.Encode(doc)
	if encErr != nil {
		return fmt.Errorf("vocab refit: encoding naming requests: %w", encErr)
	}

	return nil
}

// loadExistingTermsWithOrigins returns the existing vocabulary as name-match
// inputs: assignment vectors (stored centroid where present, else description
// embedding) plus each term's origin from vocab.centroids.json — defaulting
// to derived when absent (Task 2.4: pre-provenance and bootstrap terms are
// derived-origin on first derivation).
func loadExistingTermsWithOrigins(deps VocabDeps, vault string) ([]existingVocabTerm, error) {
	terms, loadErr := loadAssignmentTermVectors(vault, deps.ListMD, deps.ReadFile)
	if loadErr != nil {
		return nil, fmt.Errorf("vocab refit: loading term vectors: %w", loadErr)
	}

	doc, _ := readCentroidsDoc(vault, deps.ReadFile) // zero-value on missing file

	existing := make([]existingVocabTerm, 0, len(terms))

	for _, term := range terms {
		origin := doc.Terms[term.Term].Origin
		if origin == "" {
			origin = vocabOriginDerived
		}

		existing = append(existing, existingVocabTerm{Name: term.Term, Vector: term.Vector, Origin: origin})
	}

	return existing, nil
}

// namedClustersToSeedTerms converts naming answers into the SeedTerm shape
// the existing mintDefinitionNote path consumes; each new term's definition
// note carries the cluster's exemplar snippets (the body IS the embedding
// text, so exemplars pull the term vector toward its members).
func namedClustersToSeedTerms(
	requests []vocabNamingRequest,
	names map[int]vocabClusterName,
) []SeedTerm {
	seeds := make([]SeedTerm, 0, len(requests))

	for _, request := range requests {
		name, ok := names[request.Cluster]
		if !ok {
			continue
		}

		exemplars := make([]string, 0, len(request.Exemplars))
		for _, exemplar := range request.Exemplars {
			exemplars = append(exemplars, exemplar.Snippet)
		}

		seeds = append(seeds, SeedTerm{Term: name.Term, Description: name.Description, Exemplars: exemplars})
	}

	return seeds
}

// noteSnippet reads a note's body and returns its leading text, trimmed and
// capped at vocabNamingSnippetMaxLen. Empty when the note is unreadable.
func noteSnippet(deps VocabDeps, vault, name string) string {
	raw, readErr := deps.ReadFile(filepath.Join(vault, name))
	if readErr != nil {
		return ""
	}

	body := strings.TrimSpace(string(embed.ExtractBody(raw)))
	if len(body) > vocabNamingSnippetMaxLen {
		body = body[:vocabNamingSnippetMaxLen]
	}

	return body
}

// parseRefitNames parses and validates a --names answer: valid JSON, an
// echoed fingerprint matching wantFingerprint (the freshly recomputed
// derivation-input hash — a mismatch means the vault drifted since the
// naming requests were emitted), every entry naming a known new cluster
// exactly once with a non-empty term and description, and every new cluster
// covered.
func parseRefitNames(data []byte, newClusters []int, wantFingerprint string) (map[int]vocabClusterName, error) {
	var doc vocabRefitNamesDoc

	unmarshalErr := json.Unmarshal(data, &doc)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: %w", errVocabRefitBadNames, unmarshalErr)
	}

	if doc.Fingerprint != wantFingerprint {
		return nil, fmt.Errorf("%w: answer fingerprint %q, current vault %q",
			errVocabRefitStaleNames, doc.Fingerprint, wantFingerprint)
	}

	valid := make(map[int]bool, len(newClusters))
	for _, clusterIdx := range newClusters {
		valid[clusterIdx] = true
	}

	names := make(map[int]vocabClusterName, len(doc.Names))

	for _, entry := range doc.Names {
		entryErr := validateRefitNameEntry(entry, valid, names)
		if entryErr != nil {
			return nil, entryErr
		}

		names[entry.Cluster] = entry
	}

	for _, clusterIdx := range newClusters {
		if _, named := names[clusterIdx]; !named {
			return nil, fmt.Errorf("%w: cluster %d unnamed", errVocabRefitNamesIncomplete, clusterIdx)
		}
	}

	return names, nil
}

// previousDerivedK counts the derived-origin terms — the previous
// derivation's K, fed to the silhouette hysteresis.
func previousDerivedK(existing []existingVocabTerm) int {
	count := 0

	for _, term := range existing {
		if term.Origin != vocabOriginProposed {
			count++
		}
	}

	return count
}

// printDerivationDiff prints the --dry-run report: selected K, silhouette
// score, and the matched / new / retired term sets. No writes.
func printDerivationDiff(stdout io.Writer, state vocabRefitState) {
	_, _ = fmt.Fprintf(stdout, "vocab refit (dry-run): K=%d silhouette=%.3f\n",
		state.derivation.K, state.derivation.Silhouette)

	for _, matched := range state.match.Matched {
		_, _ = fmt.Fprintf(stdout, "matched: %s (cluster %d, similarity %.3f)\n",
			matched.Term, matched.ClusterIndex, matched.Similarity)
	}

	for _, clusterIdx := range state.match.NewClusters {
		_, _ = fmt.Fprintf(stdout, "new: cluster %d (%d members, needs naming)\n",
			clusterIdx, len(state.derivation.Clusters[clusterIdx].Members))
	}

	for _, term := range state.match.RetiredTerms {
		_, _ = fmt.Fprintf(stdout, "retired: %s\n", term)
	}
}

// proposedTermSidecarVector loads the body vector from term's definition-note
// sidecar, or nil when the note or sidecar is absent/unreadable (embedding is
// optional at propose time).
func proposedTermSidecarVector(deps VocabDeps, vault string, names []string, term string) []float32 {
	notePath, found := findDefinitionNotePathForTerm(vault, names, term, deps.ReadFile)
	if !found {
		return nil
	}

	sidecarData, readErr := deps.ReadFile(embed.SidecarPath(notePath))
	if readErr != nil {
		return nil
	}

	sidecar, sidecarErr := embed.UnmarshalSidecar(sidecarData)
	if sidecarErr != nil {
		return nil
	}

	return sidecar.BodyVector
}

// removeNoteReferences strips every reference to the deleted filenames from
// one note's content, returning the updated content and whether it changed.
func removeNoteReferences(content string, deleted map[string]bool) (string, bool) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		scrubbed, changed := scrubBodyReferences(content, deleted)

		return scrubbed, changed
	}

	newFrontmatter, frontChanged := scrubSupersedesFrontmatter(frontmatter, deleted)

	newBody, bodyChanged := scrubBodyReferences(body, deleted)
	if !frontChanged && !bodyChanged {
		return content, false
	}

	return fmStart + newFrontmatter + fmEnd + newBody, true
}

// retireVocabTerms retires derivation-unmatched derived-origin terms
// (dormant terms re-attract members and defeat convergence, so retirement is
// active). Vocab definition notes have no supersession story — they are term
// identity, not preserved facts — so for each retired term it:
//
//  1. deletes the term's definition note and its .vec.json sidecar outright
//     (no demoted note, no supersession record);
//  2. scrubs every remaining reference to the deleted note vault-wide:
//     frontmatter supersedes: entries, "Supersedes: [[...]]" body lines, and
//     inline wikilinks naming the deleted basename;
//  3. strips vocab/<term> from member notes (clearRemovedTermsFromMembers) —
//     the post-derivation re-tag pass also replaces the vocab namespace, this
//     covers notes without sidecars.
//
// origin: proposed terms never reach this function — matchClustersToTerms
// excludes them from RetiredTerms (the provenance shield).
func retireVocabTerms(deps VocabDeps, vault string, retired []string) {
	if len(retired) == 0 {
		return
	}

	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab refit: listing vault for retirement: %v", listErr)
		}

		return
	}

	deleted := make([]string, 0, len(retired))

	for _, term := range retired {
		notePath, found := findDefinitionNotePathForTerm(vault, names, term, deps.ReadFile)
		if !found {
			continue
		}

		deleteDefinitionNote(deps, notePath)
		deleted = append(deleted, filepath.Base(notePath))
	}

	scrubDeletedNoteReferences(deps, vault, names, deleted)

	clearErr := clearRemovedTermsFromMembers(deps, vault, retired)
	if clearErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: stripping retired terms from members: %v", clearErr)
	}
}

// scrubBodyReferences removes body references to the deleted filenames:
// "Supersedes: [[...]]" lines linking a deleted note are dropped whole
// (machine-written, content-hash-excluded), and inline wikilinks — with or
// without the .md extension — are unlinked to their plain-text basename.
func scrubBodyReferences(body string, deleted map[string]bool) (string, bool) {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	changed := false

	for _, line := range lines {
		if strings.HasPrefix(line, supersedesBodyMarker) && supersedesLineTargetsDeleted(line, deleted) {
			changed = true

			continue
		}

		unlinked := unlinkDeletedWikilinks(line, deleted)
		if unlinked != line {
			changed = true
		}

		kept = append(kept, unlinked)
	}

	return strings.Join(kept, "\n"), changed
}

// scrubDeletedNoteReferences removes every remaining reference to the deleted
// definition notes from the rest of the vault: frontmatter supersedes:
// entries naming a deleted filename, "Supersedes: [[...]]" body lines linking
// one, and inline wikilinks (with or without the .md extension), which are
// unlinked to plain text. Notes without references are left byte-identical.
func scrubDeletedNoteReferences(deps VocabDeps, vault string, names, deleted []string) {
	if len(deleted) == 0 {
		return
	}

	deletedSet := make(map[string]bool, len(deleted))
	for _, filename := range deleted {
		deletedSet[filename] = true
	}

	for _, name := range names {
		if deletedSet[name] {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		updated, changed := removeNoteReferences(string(raw), deletedSet)
		if !changed {
			continue
		}

		writeErr := deps.WriteFile(notePath, []byte(updated))
		if writeErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: scrubbing references in %s: %v", notePath, writeErr)
		}
	}
}

// scrubSupersedesFrontmatter removes supersedes: list entries whose note:
// value names a deleted filename, dropping the supersedes: key itself when
// its last entry goes. All other frontmatter lines pass through verbatim.
func scrubSupersedesFrontmatter(frontmatter string, deleted map[string]bool) (string, bool) {
	lines := strings.Split(frontmatter, "\n")
	kept := make([]string, 0, len(lines))
	changed := false

	for i := 0; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " ") != "supersedes:" {
			kept = append(kept, lines[i])

			continue
		}

		entries, next := collectSupersedesEntryBlocks(lines, i+1)
		keptEntries := make([][]string, 0, len(entries))

		for _, entry := range entries {
			if deleted[supersedesEntryNoteValue(entry)] {
				changed = true

				continue
			}

			keptEntries = append(keptEntries, entry)
		}

		if len(keptEntries) > 0 {
			kept = append(kept, lines[i])
			for _, entry := range keptEntries {
				kept = append(kept, entry...)
			}
		} else {
			changed = true
		}

		i = next - 1
	}

	return strings.Join(kept, "\n"), changed
}

// stampProposedTermOrigin upserts an origin: proposed centroid entry for term
// into vocab.centroids.json (Task 2.4 — provenance is new plumbing: nothing
// distinguished proposed terms before, so propose must write the marker that
// derivation-driven retirement later reads). An existing entry keeps its
// vector and member count; a fresh entry carries the definition sidecar's
// body vector (empty when not embedded yet — regenerated by the next refit).
// A missing centroids file is created; all other doc fields are preserved.
// names is the vault listing including the just-minted definition note.
func stampProposedTermOrigin(deps VocabDeps, vault string, names []string, term string) {
	doc, ok := readCentroidsDoc(vault, deps.ReadFile)
	if !ok {
		modelID, dims := firstTermSidecarMeta(vault, names, deps.ReadFile)
		doc = vocabCentroidsDoc{
			SchemaVersion:    vocabCentroidsSchemaVersion,
			EmbeddingModelID: modelID,
			Dims:             dims,
		}
	}

	if doc.Terms == nil {
		doc.Terms = make(map[string]vocabCentroidEntry, 1)
	}

	entry := doc.Terms[term]
	entry.Origin = vocabOriginProposed

	if len(entry.Vector) == 0 {
		entry.Vector = proposedTermSidecarVector(deps, vault, names, term)
	}

	doc.Terms[term] = entry

	writeErr := writeCentroidsDocRaw(vault, doc, deps.WriteFile)
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab propose: stamping origin for %s: %v", term, writeErr)
	}
}

// supersedesEntryNoteValue extracts the note: filename from one supersedes
// entry block, or "" when the block has no note: line.
func supersedesEntryNoteValue(entry []string) string {
	for _, line := range entry {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if value, found := strings.CutPrefix(trimmed, "note:"); found {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

// supersedesLineTargetsDeleted reports whether a Supersedes: body line's
// wikilink targets one of the deleted filenames (either link form).
func supersedesLineTargetsDeleted(line string, deleted map[string]bool) bool {
	for filename := range deleted {
		basename := strings.TrimSuffix(filename, ".md")
		if strings.Contains(line, "[["+filename+"]]") || strings.Contains(line, "[["+basename+"]]") {
			return true
		}
	}

	return false
}

// unlinkDeletedWikilinks replaces inline wikilinks to deleted notes with
// their plain-text basename, removing the graph edge but keeping the prose.
func unlinkDeletedWikilinks(line string, deleted map[string]bool) string {
	for filename := range deleted {
		basename := strings.TrimSuffix(filename, ".md")
		line = strings.ReplaceAll(line, "[["+filename+"]]", basename)
		line = strings.ReplaceAll(line, "[["+basename+"]]", basename)
	}

	return line
}

// validateRefitNameEntry rejects a --names entry naming an unknown cluster,
// a cluster already named, or carrying an empty term/description.
func validateRefitNameEntry(
	entry vocabClusterName,
	valid map[int]bool,
	names map[int]vocabClusterName,
) error {
	if !valid[entry.Cluster] {
		return fmt.Errorf("%w: cluster %d", errVocabRefitNamesUnknownCluster, entry.Cluster)
	}

	if _, dup := names[entry.Cluster]; dup {
		return fmt.Errorf("%w: cluster %d", errVocabRefitNamesDuplicate, entry.Cluster)
	}

	if strings.TrimSpace(entry.Term) == "" || strings.TrimSpace(entry.Description) == "" {
		return fmt.Errorf("%w: cluster %d", errVocabRefitNameEmpty, entry.Cluster)
	}

	return nil
}
