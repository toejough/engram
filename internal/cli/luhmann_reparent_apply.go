package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/luhmann"
)

// RunReparentLuhmann implements `engram update --reparent-luhmann`'s
// derive/answer/apply/dry-run flow (design.md, specs/update-reparent-luhmann-batch).
//
// No answersFile: derive-only — computes candidate pairs among top-level
// notes and prints a JSON payload; never writes, regardless of dryRun.
//
// answersFile set: apply phase — validates the echoed fingerprint against the
// vault's current state, computes new Luhmann IDs, and (dryRun==false) calls
// RenameAndRewriteReferences once with the full rename map. dryRun==true
// prints the rename/rewrite preview instead of writing.
//
// dryRun with no answersFile is rejected as a usage error (nothing to preview).
func RunReparentLuhmann(vault, answersFile string, dryRun bool, deps RenameRewriteDeps, stdout io.Writer) error {
	if answersFile == "" {
		if dryRun {
			return errReparentDryRunNeedsAnswers
		}

		return runReparentDerive(vault, deps, stdout)
	}

	return runReparentApply(vault, answersFile, dryRun, deps, stdout)
}

// unexported constants.
const (
	// reparentExcerptMaxLen caps a candidate's body excerpt length in bytes
	// (mirrors vocabNamingSnippetMaxLen's role for vocab naming requests).
	reparentExcerptMaxLen = 200
	// reparentSimilarityFloor is the minimum cosine similarity for a
	// candidate pair to be worth an agent's judgment (design.md Decision 1
	// starting value — tune empirically once run against a real vault).
	reparentSimilarityFloor = float32(0.75)
	// reparentTopK is how many nearest OTHER top-level notes a derive pass
	// considers per note (design.md Decision 1). Kept small to bound the
	// derive payload on a large vault.
	reparentTopK = 3
	// reparentTopLevelDepth is the segment count of a top-level Luhmann ID —
	// derive only considers top-level notes as candidates and targets.
	reparentTopLevelDepth = 1
)

// unexported variables.
var (
	errReparentBadAnswers         = errors.New("update --reparent-luhmann: cannot parse --answers JSON")
	errReparentDryRunNeedsAnswers = errors.New(
		"update --reparent-luhmann --dry-run requires --answers (derive-phase output is already a preview)")
	errReparentStaleAnswers = errors.New(
		"update --reparent-luhmann: the vault changed since the candidates were derived; " +
			"re-run `engram update --reparent-luhmann` to regenerate them")
	errReparentUnknownNote = errors.New("update --reparent-luhmann: --answers references an unknown note")
)

// reparentAnswerEntry is one disposition judgment in an --answers file.
type reparentAnswerEntry struct {
	Note     string `json:"note"`
	Position string `json:"position"`
	Target   string `json:"target"`
}

// reparentAnswersDoc is the parsed shape of an --answers file (design.md
// Decision 2).
type reparentAnswersDoc struct {
	Reparenting []reparentAnswerEntry `json:"reparenting"`
	Fingerprint string                `json:"fingerprint"`
}

// reparentCandidate is one (note, target) pair from the derive phase: note is
// a top-level note ID with an above-floor-similarity neighbor, target is that
// neighbor's ID. Excerpts give the answering agent enough content to judge
// the relationship without a separate note read.
//
//nolint:tagliatelle // payload keys follow the vault's snake_case contract (mirrors vocab naming requests)
type reparentCandidate struct {
	Note          string  `json:"note"`
	NoteExcerpt   string  `json:"note_excerpt"`
	Target        string  `json:"target"`
	TargetExcerpt string  `json:"target_excerpt"`
	Similarity    float32 `json:"similarity"`
}

// reparentDerivePayload is the stdout JSON payload the derive phase emits.
type reparentDerivePayload struct {
	Candidates  []reparentCandidate `json:"candidates"`
	Instruction string              `json:"instruction"`
	Fingerprint string              `json:"fingerprint"`
}

// reparentNote is one top-level note's identity, filename, and embedding
// vector, as loaded from the vault for candidate derivation and fingerprinting.
type reparentNote struct {
	ID       string
	Filename string
	Vector   []float32
}

// buildReparentRenameMap computes the old-basename→new-basename rename map
// from answered entries, processed in ascending original-ID order
// (design.md Decision 3): position=top is a no-op; continuation/sibling
// compute a new ID via nextLuhmannID against the vault's current ID set,
// which grows as earlier entries in this same run are assigned.
func buildReparentRenameMap(
	vault string,
	entries []reparentAnswerEntry,
	deps RenameRewriteDeps,
) (map[string]string, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return nil, fmt.Errorf("update --reparent-luhmann: listing %s: %w", vault, listErr)
	}

	idToFilename := make(map[string]string, len(names))
	existingIDs := make([]string, 0, len(names))

	for _, name := range names {
		id, ok := extractLuhmannFromFilename(name)
		if !ok {
			continue
		}

		idToFilename[id] = name
		existingIDs = append(existingIDs, id)
	}

	sorted := make([]reparentAnswerEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		return reparentIDLess(sorted[i].Note, sorted[j].Note)
	})

	renameMap := make(map[string]string, len(sorted))

	for _, entry := range sorted {
		if entry.Position == positionTop {
			continue
		}

		oldFilename, ok := idToFilename[entry.Note]
		if !ok {
			return nil, fmt.Errorf("%w: %q", errReparentUnknownNote, entry.Note)
		}

		newID, idErr := nextLuhmannID(existingIDs, entry.Target, entry.Position)
		if idErr != nil {
			return nil, fmt.Errorf("update --reparent-luhmann: computing new id for %q: %w", entry.Note, idErr)
		}

		existingIDs = append(existingIDs, newID)

		date := reparentDateFromFilename(oldFilename)
		slug := slugFromNoteFilename(oldFilename)
		oldBasename := strings.TrimSuffix(oldFilename, mdExt)
		newBasename := newID + "." + date + "." + slug

		renameMap[oldBasename] = newBasename
	}

	return renameMap, nil
}

// deriveReparentCandidates computes, for each note, its top-K nearest OTHER
// notes by cosine similarity, keeping only pairs above reparentSimilarityFloor
// (design.md Decision 1).
func deriveReparentCandidates(vault string, notes []reparentNote, deps RenameRewriteDeps) []reparentCandidate {
	candidates := make([]reparentCandidate, 0, min(reparentTopK, len(notes))*len(notes))

	for _, note := range notes {
		type scored struct {
			target reparentNote
			sim    float32
		}

		neighbors := make([]scored, 0, len(notes)-1)

		for _, other := range notes {
			if other.ID == note.ID {
				continue
			}

			sim := embed.Cosine(note.Vector, other.Vector)
			if sim >= reparentSimilarityFloor {
				neighbors = append(neighbors, scored{target: other, sim: sim})
			}
		}

		sort.SliceStable(neighbors, func(i, j int) bool { return neighbors[i].sim > neighbors[j].sim })

		count := min(reparentTopK, len(neighbors))
		for _, n := range neighbors[:count] {
			candidates = append(candidates, reparentCandidate{
				Note:          note.ID,
				NoteExcerpt:   reparentExcerpt(vault, note.Filename, deps.ReadFile),
				Target:        n.target.ID,
				TargetExcerpt: reparentExcerpt(vault, n.target.Filename, deps.ReadFile),
				Similarity:    n.sim,
			})
		}
	}

	return candidates
}

// loadTopLevelReparentNotes returns every top-level note in vault that has an
// embedding sidecar, sorted by filename for determinism (mirrors
// deriveVaultVocab's sorted-name-order convention).
func loadTopLevelReparentNotes(vault string, deps RenameRewriteDeps) ([]reparentNote, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return nil, fmt.Errorf("update --reparent-luhmann: listing %s: %w", vault, listErr)
	}

	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	notes := make([]reparentNote, 0, len(sorted))

	for _, name := range sorted {
		id, _, ok := idAndDateFromNoteFilename(name)
		if !ok {
			continue
		}

		segments, parseErr := luhmann.ParseID(id)
		if parseErr != nil || len(segments) != reparentTopLevelDepth {
			continue
		}

		vector, vectorOK := reparentSidecarVector(vault, name, deps.ReadFile)
		if !vectorOK {
			continue
		}

		notes = append(notes, reparentNote{ID: id, Filename: name, Vector: vector})
	}

	return notes, nil
}

// printReparentPreview prints the old→new basename map and, for each, the
// list of other notes whose references would be rewritten (design.md
// Decision 5) — reusing rewriteNoteReferences (the exact function
// RenameAndRewriteReferences applies) so the preview can never drift from
// what apply actually does.
func printReparentPreview(stdout io.Writer, vault string, deps RenameRewriteDeps, renameMap map[string]string) {
	if len(renameMap) == 0 {
		_, _ = fmt.Fprintln(stdout, "update --reparent-luhmann (dry-run): no renames")

		return
	}

	_, _ = fmt.Fprintln(stdout, "update --reparent-luhmann (dry-run): would rename")

	oldBasenames := make([]string, 0, len(renameMap))
	for oldBasename := range renameMap {
		oldBasenames = append(oldBasenames, oldBasename)
	}

	sort.Strings(oldBasenames)

	for _, oldBasename := range oldBasenames {
		_, _ = fmt.Fprintf(stdout, "  %s -> %s\n", oldBasename, renameMap[oldBasename])
	}

	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return
	}

	for _, name := range names {
		basename := strings.TrimSuffix(name, mdExt)
		if _, renaming := renameMap[basename]; renaming {
			continue
		}

		raw, readErr := deps.ReadFile(vault + "/" + name)
		if readErr != nil {
			continue
		}

		_, changed := rewriteNoteReferences(string(raw), renameMap)
		if changed {
			_, _ = fmt.Fprintf(stdout, "  references rewritten in: %s\n", name)
		}
	}
}

// reparentDateFromFilename extracts the <date> segment from a note filename
// of the form "<id>.<date>.<slug>.md".
func reparentDateFromFilename(name string) string {
	_, date, _ := idAndDateFromNoteFilename(name)

	return date
}

// reparentExcerpt reads a note's body and returns its leading text, trimmed
// and capped at reparentExcerptMaxLen. Empty when the note is unreadable.
func reparentExcerpt(vault, name string, readFile func(string) ([]byte, error)) string {
	raw, readErr := readFile(vault + "/" + name)
	if readErr != nil {
		return ""
	}

	body := strings.TrimSpace(string(embed.ExtractBody(raw)))
	if len(body) > reparentExcerptMaxLen {
		body = body[:reparentExcerptMaxLen]
	}

	return body
}

// reparentFingerprint hashes the derivation inputs — reuses
// computeDerivationFingerprint's SHA-256-over-sorted-names-and-vectors
// approach (vocab refit's fingerprint mechanism) so an --answers file is
// rejected the same way a stale vocab refit --names is.
func reparentFingerprint(notes []reparentNote) string {
	inputs := make([]noteVector, 0, len(notes))
	for _, note := range notes {
		inputs = append(inputs, noteVector{Name: note.Filename, Vector: note.Vector})
	}

	return computeDerivationFingerprint(inputs)
}

// reparentIDLess orders two top-level Luhmann IDs ascending numerically
// (design.md Decision 3: apply processes answered notes oldest-ID-first).
// Non-numeric IDs (should not occur among derive's top-level candidates) sort
// after numeric ones, by string comparison, defensively.
func reparentIDLess(a, b string) bool {
	an, aErr := strconv.Atoi(a)
	bn, bErr := strconv.Atoi(b)

	if aErr == nil && bErr == nil {
		return an < bn
	}

	return a < b
}

// reparentSidecarVector loads a note's body vector from its .vec.json
// sidecar. Returns ok=false when the sidecar is missing or unparseable —
// derive treats such notes as having no signal, not an error.
func reparentSidecarVector(
	vault, name string,
	readFile func(string) ([]byte, error),
) ([]float32, bool) {
	notePath := vault + "/" + name

	sidecarData, readErr := readFile(embed.SidecarPath(notePath))
	if readErr != nil {
		return nil, false
	}

	sidecar, unmarshalErr := embed.UnmarshalSidecar(sidecarData)
	if unmarshalErr != nil || len(sidecar.BodyVector) == 0 {
		return nil, false
	}

	return sidecar.BodyVector, true
}

// runReparentApply validates the --answers file's fingerprint, computes the
// full rename map (design.md Decision 3), and either previews it (dryRun) or
// applies it via a single RenameAndRewriteReferences call (design.md
// Decision 4).
func runReparentApply(vault, answersFile string, dryRun bool, deps RenameRewriteDeps, stdout io.Writer) error {
	notes, listErr := loadTopLevelReparentNotes(vault, deps)
	if listErr != nil {
		return listErr
	}

	answersData, readErr := deps.ReadFile(answersFile)
	if readErr != nil {
		return fmt.Errorf("update --reparent-luhmann: reading --answers: %w", readErr)
	}

	var doc reparentAnswersDoc

	unmarshalErr := json.Unmarshal(answersData, &doc)
	if unmarshalErr != nil {
		return fmt.Errorf("%w: %w", errReparentBadAnswers, unmarshalErr)
	}

	wantFingerprint := reparentFingerprint(notes)
	if doc.Fingerprint != wantFingerprint {
		return fmt.Errorf("%w: answer fingerprint %q, current vault %q",
			errReparentStaleAnswers, doc.Fingerprint, wantFingerprint)
	}

	renameMap, buildErr := buildReparentRenameMap(vault, doc.Reparenting, deps)
	if buildErr != nil {
		return buildErr
	}

	if dryRun {
		printReparentPreview(stdout, vault, deps, renameMap)

		return nil
	}

	applyErr := RenameAndRewriteReferences(deps, vault, renameMap)
	if applyErr != nil {
		return fmt.Errorf("update --reparent-luhmann: applying renames: %w", applyErr)
	}

	_, _ = fmt.Fprintf(stdout, "update --reparent-luhmann: renamed %d note(s)\n", len(renameMap))

	return nil
}

// runReparentDerive computes candidate pairs and prints the derive-phase
// payload. Never writes any file.
func runReparentDerive(vault string, deps RenameRewriteDeps, stdout io.Writer) error {
	notes, listErr := loadTopLevelReparentNotes(vault, deps)
	if listErr != nil {
		return listErr
	}

	candidates := deriveReparentCandidates(vault, notes, deps)
	if len(candidates) == 0 {
		_, _ = fmt.Fprintln(stdout, "update --reparent-luhmann: no candidates found")

		return nil
	}

	payload := reparentDerivePayload{
		Candidates: candidates,
		Instruction: "For each candidate note, judge whether it is a continuation (a deeper elaboration of " +
			"target), a sibling (a related but distinct point next to target), or unrelated (top — no change). " +
			`Write an answers file: {"reparenting":[{"note":"<id>","position":"top|continuation|sibling",` +
			`"target":"<id>"}],"fingerprint":"<echoed>"}, one entry per candidate note, then re-run ` +
			"`engram update --reparent-luhmann --answers <file>` echoing this payload's fingerprint verbatim.",
		Fingerprint: reparentFingerprint(notes),
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	encErr := enc.Encode(payload)
	if encErr != nil {
		return fmt.Errorf("update --reparent-luhmann: encoding candidates: %w", encErr)
	}

	return nil
}
