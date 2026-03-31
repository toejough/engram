package correct

import (
	"fmt"
	"path/filepath"
	"time"

	"engram/internal/memory"
)

// Exported constants.
const (
	DispositionContradiction           = "CONTRADICTION"
	DispositionDuplicate               = "DUPLICATE"
	DispositionImpactUpdate            = "IMPACT_UPDATE"
	DispositionLegitSeparate           = "LEGITIMATE_SEPARATE"
	DispositionPotentialGeneralization = "POTENTIAL_GENERALIZATION"
	DispositionRefinement              = "REFINEMENT"
	DispositionStore                   = "STORE"
	DispositionStoreBoth               = "STORE_BOTH"
)

// DispositionResult describes the outcome of handling an extraction's disposition.
type DispositionResult struct {
	// Action is one of: "stored", "duplicate_skipped", "updated", "contradiction", "refinement".
	Action string
	// Path is the file path of the created or updated memory (empty for duplicate_skipped).
	Path string
	// Reason is a human-readable explanation of the disposition decision.
	Reason string
}

// MemoryModifier reads, mutates, and writes back a MemoryRecord atomically.
type MemoryModifier interface {
	ReadModifyWrite(path string, mutate func(*memory.MemoryRecord)) error
}

// MemoryWriter writes a MemoryRecord to persistent storage.
type MemoryWriter interface {
	Write(record *memory.MemoryRecord, slug, dataDir string) (string, error)
}

// HandleDisposition applies the first matching candidate disposition rule and
// either stores a new memory or mutates an existing one.
func HandleDisposition(
	extraction *ExtractionResult,
	writer MemoryWriter,
	modifier MemoryModifier,
	dataDir, projectSlug string,
) (*DispositionResult, error) {
	for _, candidate := range extraction.Candidates {
		result, done, err := applyCandidate(extraction, writer, modifier, dataDir, projectSlug, candidate)
		if err != nil {
			return nil, err
		}

		if done {
			return result, nil
		}
	}

	// No blocking disposition matched — store the new memory.
	storedPath, err := storeNew(extraction, writer, dataDir, projectSlug)
	if err != nil {
		return nil, err
	}

	return &DispositionResult{Action: "stored", Path: storedPath}, nil
}

// unexported constants.
const (
	memoriesSubdir = "memories"
	triageReminder = "review at next /memory-triage"
)

// applyCandidate evaluates a single candidate disposition.
// Returns (result, true, nil) if the candidate produced a final decision,
// (nil, false, nil) to continue to the next candidate, or (nil, false, err) on error.
func applyCandidate(
	extraction *ExtractionResult,
	writer MemoryWriter,
	modifier MemoryModifier,
	dataDir, projectSlug string,
	candidate CandidateResult,
) (*DispositionResult, bool, error) {
	switch candidate.Disposition {
	case DispositionDuplicate:
		result := &DispositionResult{
			Action: "duplicate_skipped",
			Reason: fmt.Sprintf("duplicate of %s: %s", candidate.Name, candidate.Reason),
		}

		return result, true, nil

	case DispositionImpactUpdate:
		return applyModify(modifier, dataDir, candidate.Name, func(rec *memory.MemoryRecord) {
			rec.Impact = extraction.Impact
		}, "updated", "updating impact for "+candidate.Name)

	case DispositionPotentialGeneralization:
		return applyModify(modifier, dataDir, candidate.Name, func(rec *memory.MemoryRecord) {
			rec.Situation = extraction.Situation
		}, "updated", "updating situation for "+candidate.Name)

	case DispositionContradiction:
		return applyStoreWithTag(extraction, writer, dataDir, projectSlug, "contradiction")

	case DispositionRefinement:
		return applyStoreWithTag(extraction, writer, dataDir, projectSlug, "refinement")

	case DispositionStore, DispositionStoreBoth, DispositionLegitSeparate:
		// Fall through to store.
		return nil, false, nil

	default:
		return nil, false, nil
	}
}

// applyModify runs ReadModifyWrite on the named candidate and returns an "updated" result.
func applyModify(
	modifier MemoryModifier,
	dataDir, candidateName string,
	mutate func(*memory.MemoryRecord),
	action, errContext string,
) (*DispositionResult, bool, error) {
	memPath := memoryPath(dataDir, candidateName)

	err := modifier.ReadModifyWrite(memPath, mutate)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", errContext, err)
	}

	return &DispositionResult{Action: action, Path: memPath}, true, nil
}

// applyStoreWithTag stores a new memory and returns a result with the given action tag
// and a triage reminder.
func applyStoreWithTag(
	extraction *ExtractionResult,
	writer MemoryWriter,
	dataDir, projectSlug, action string,
) (*DispositionResult, bool, error) {
	storedPath, err := storeNew(extraction, writer, dataDir, projectSlug)
	if err != nil {
		return nil, false, err
	}

	return &DispositionResult{
		Action: action,
		Path:   storedPath,
		Reason: triageReminder,
	}, true, nil
}

// memoryPath returns the expected file path for a named memory record.
func memoryPath(dataDir, candidateName string) string {
	return filepath.Join(dataDir, memoriesSubdir, candidateName+".toml")
}

// storeNew creates a MemoryRecord from extraction fields and writes it via writer.
func storeNew(
	extraction *ExtractionResult,
	writer MemoryWriter,
	dataDir, projectSlug string,
) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	record := &memory.MemoryRecord{
		Situation: extraction.Situation,
		Behavior:  extraction.Behavior,
		Impact:    extraction.Impact,
		Action:    extraction.Action,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if extraction.ProjectScoped {
		record.ProjectScoped = true
		record.ProjectSlug = projectSlug
	}

	storedPath, err := writer.Write(record, extraction.FilenameSlug, dataDir)
	if err != nil {
		return "", fmt.Errorf("writing new memory: %w", err)
	}

	return storedPath, nil
}
