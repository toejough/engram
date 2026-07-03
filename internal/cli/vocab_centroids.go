package cli

import (
	"encoding/json"
	"path/filepath"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// vocabCentroidsFilename is the derived per-term centroid store in the vault root.
	vocabCentroidsFilename = "vocab.centroids.json"
	// vocabCentroidsSchemaVersion versions the centroids file format.
	vocabCentroidsSchemaVersion = 1
)

// vocabCentroidEntry is one term's derived centroid in vocab.centroids.json.
//
//nolint:tagliatelle // centroids JSON keys follow the sidecar spec contract (snake_case)
type vocabCentroidEntry struct {
	Vector      []float32 `json:"vector"`
	MemberCount int       `json:"member_count"`
}

// vocabCentroidsDoc is the on-disk format of vocab.centroids.json — the
// centroid two-pass assigner's derived per-term vector store (Joe's call
// 2026-07-02, plan §Write-time assign).
//
// Storage design choice: a SEPARATE derived file in the vault root, NOT the
// term note's sidecar. A sidecar must remain a pure function of its note's
// content (content_hash stays honest), and `engram embed apply` rebuilds
// sidecars from note content — a centroid stored there would be silently
// clobbered on the next apply, or force guard code into the embed path. A
// derived file survives `embed apply` untouched with zero guards. The file is
// stamped with the embedding model id; loadAssignmentTermVectors ignores it
// when the id differs from the term sidecars (stale space after a model swap)
// — re-run bootstrap or refit to regenerate.
//
//nolint:tagliatelle // centroids JSON keys follow the sidecar spec contract (snake_case)
type vocabCentroidsDoc struct {
	SchemaVersion    int                           `json:"schema_version"`
	EmbeddingModelID string                        `json:"embedding_model_id"`
	Dims             int                           `json:"dims"`
	Terms            map[string]vocabCentroidEntry `json:"terms"`
	// --- lifecycle fields ---
	RefitPending bool               `json:"refit_pending,omitempty"`
	RefitReason  string             `json:"refit_reason,omitempty"`
	LastRefit    *vocabLastRefitDoc `json:"last_refit,omitempty"`
}

// vocabLastRefitDoc holds the vault state at the time of the last bootstrap or refit.
//
//nolint:tagliatelle // JSON keys follow snake_case sidecar spec contract
type vocabLastRefitDoc struct {
	NoteCount int    `json:"note_count"`
	Date      string `json:"date"` // YYYY-MM-DD
}

// computeTermCentroids derives pass-2 term vectors from the pass-1 assignment:
// each term with members gets the mean of its members' body vectors; a term
// with zero members keeps its description embedding (no centroid to compute)
// and is omitted from the returned entries map.
func computeTermCentroids(
	descTerms []TermWithVector,
	pass1 map[string][]string,
	noteVecs map[string][]float32,
) ([]TermWithVector, map[string]vocabCentroidEntry) {
	memberVecs := make(map[string][][]float32, len(descTerms))

	for note, assigned := range pass1 {
		vec, hasVec := noteVecs[note]
		if !hasVec {
			continue
		}

		for _, term := range assigned {
			memberVecs[term] = append(memberVecs[term], vec)
		}
	}

	pass2 := make([]TermWithVector, len(descTerms))
	entries := make(map[string]vocabCentroidEntry, len(memberVecs))

	for i, term := range descTerms {
		pass2[i] = term

		vecs := uniformDimVectors(memberVecs[term.Term])
		if len(vecs) == 0 {
			continue
		}

		// meanVector (query.go) requires a non-empty, uniformly-dimensioned
		// input — guaranteed by the uniformDimVectors filter above.
		centroid := meanVector(vecs)
		pass2[i].Vector = centroid
		entries[term.Term] = vocabCentroidEntry{Vector: centroid, MemberCount: len(vecs)}
	}

	return pass2, entries
}

// firstTermSidecarMeta returns the embedding model id and dims from the first
// readable vocab term sidecar among names, or zero values when none is readable.
func firstTermSidecarMeta(
	vault string,
	names []string,
	readFile func(path string) ([]byte, error),
) (string, int) {
	for _, name := range names {
		if name == vocabIndexFilename || !isVocabTermFilename(name) {
			continue
		}

		data, readErr := readFile(embed.SidecarPath(filepath.Join(vault, name)))
		if readErr != nil {
			continue
		}

		sidecar, sidecarErr := embed.UnmarshalSidecar(data)
		if sidecarErr != nil {
			continue
		}

		return sidecar.EmbeddingModelID, sidecar.Dims
	}

	return "", 0
}

// loadAssignmentTermVectors returns the vectors write-time assignment uses:
// the stored member centroid where vocab.centroids.json has one (same
// embedding space only), else the term sidecar (description) embedding.
// A missing, malformed, or model-mismatched centroids file degrades cleanly
// to description embeddings — never an error.
func loadAssignmentTermVectors(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(path string) ([]byte, error),
) ([]TermWithVector, error) {
	terms, err := loadTermVectors(vault, listMD, readFile)
	if err != nil || len(terms) == 0 {
		return terms, err
	}

	doc, docOK := readCentroidsDoc(vault, readFile)
	if !docOK {
		return terms, nil // no usable centroids — description embeddings
	}

	if doc.EmbeddingModelID != "" {
		names, _ := listMD(vault)

		modelID, _ := firstTermSidecarMeta(vault, names, readFile)
		if modelID != "" && modelID != doc.EmbeddingModelID {
			return terms, nil // stale embedding space (model swap) — regenerate via bootstrap/refit
		}
	}

	for i, term := range terms {
		if entry, hasCentroid := doc.Terms[term.Term]; hasCentroid && len(entry.Vector) > 0 {
			terms[i].Vector = entry.Vector
		}
	}

	return terms, nil
}

// loadMemberNoteVectors returns basename→body-vector for every non-vocab note
// with a readable sidecar. Notes without sidecars are skipped, mirroring
// assignVocabToNote's skip semantics.
func loadMemberNoteVectors(deps VocabDeps, vault string) map[string][]float32 {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return nil
	}

	result := make(map[string][]float32, len(names))

	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		sidecarData, readErr := deps.ReadFile(embed.SidecarPath(filepath.Join(vault, name)))
		if readErr != nil {
			continue
		}

		sidecar, sidecarErr := embed.UnmarshalSidecar(sidecarData)
		if sidecarErr != nil || len(sidecar.BodyVector) == 0 {
			continue
		}

		result[name] = sidecar.BodyVector
	}

	return result
}

// readCentroidsDoc loads and parses vocab.centroids.json. ok=false when the
// file is missing or malformed — callers degrade to description embeddings.
func readCentroidsDoc(vault string, readFile func(string) ([]byte, error)) (vocabCentroidsDoc, bool) {
	var doc vocabCentroidsDoc

	data, readErr := readFile(filepath.Join(vault, vocabCentroidsFilename))
	if readErr != nil {
		return doc, false
	}

	unmarshalErr := json.Unmarshal(data, &doc)
	if unmarshalErr != nil {
		return doc, false
	}

	return doc, true
}

// retagAllNotesTwoPass runs the centroid two-pass over all member notes:
// pass 1 assigns in-memory against descTerms (description+exemplar embeddings),
// pass 2 re-assigns every note against the member centroids and writes both
// channels. The derived centroids are persisted to vocab.centroids.json.
// Returns pass-2 member counts (for the index).
func retagAllNotesTwoPass(
	deps VocabDeps,
	vault string,
	descTerms []TermWithVector,
	floor float32,
) map[string]int {
	noteVecs := loadMemberNoteVectors(deps, vault)

	pass1 := make(map[string][]string, len(noteVecs))
	for name, vec := range noteVecs {
		pass1[name] = AssignVocabTerms(vec, descTerms, floor)
	}

	centroidTerms, entries := computeTermCentroids(descTerms, pass1, noteVecs)

	memberCounts, assignErr := assignTermsToAllNotes(deps, vault, centroidTerms, floor)
	if assignErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab: pass-2 assignment: %v", assignErr)
	}

	writeCentroidsFile(deps, vault, entries)

	return memberCounts
}

// uniformDimVectors returns the vectors whose length matches the first
// vector's, satisfying meanVector's uniform-dimension contract. Sidecar
// vectors from one embedding model always match; this guards corrupt input.
func uniformDimVectors(vecs [][]float32) [][]float32 {
	if len(vecs) == 0 {
		return nil
	}

	dims := len(vecs[0])
	uniform := make([][]float32, 0, len(vecs))

	for _, vec := range vecs {
		if len(vec) == dims {
			uniform = append(uniform, vec)
		}
	}

	return uniform
}

// writeCentroidsFile persists the derived centroids to vocab.centroids.json,
// stamped with the term sidecars' model id and dims. See vocabCentroidsDoc
// for the storage rationale (separate derived file, survives embed apply).
func writeCentroidsFile(deps VocabDeps, vault string, entries map[string]vocabCentroidEntry) {
	names, _ := deps.ListMD(vault)
	modelID, dims := firstTermSidecarMeta(vault, names, deps.ReadFile)

	doc := vocabCentroidsDoc{
		SchemaVersion:    vocabCentroidsSchemaVersion,
		EmbeddingModelID: modelID,
		Dims:             dims,
		Terms:            entries,
	}

	data, _ := json.Marshal(doc) //nolint:errchkjson // finite floats + string keys never fail to encode

	writeErr := deps.WriteFile(filepath.Join(vault, vocabCentroidsFilename), data)
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab: writing %s: %v", vocabCentroidsFilename, writeErr)
	}
}
