//go:build targ

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/toejough/targ"
)

func init() {
	targ.Register(targ.Targ(c4Registry).Name("c4-registry").
		Description("Walk architecture/c4/c*.json and emit a unified ID/name projection " +
			"with cross-spec conflict findings."))
}

// C4RegistryArgs configures the c4-registry target.
type C4RegistryArgs struct {
	Dir string `targ:"flag,name=dir,desc=Directory to scan (default architecture/c4)"`
}

// RegistryAppearance is one (file, id, name) tuple from a scanned spec.
type RegistryAppearance struct {
	File string `json:"file"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// RegistryConflict reports a cross-spec inconsistency.
type RegistryConflict struct {
	Kind        string   `json:"kind"`
	ID          string   `json:"id,omitempty"`
	NamePattern string   `json:"name_pattern,omitempty"`
	Detail      string   `json:"detail"`
	Evidence    []string `json:"evidence"`
}

// RegistryElement groups every appearance of a single E-ID across files.
type RegistryElement struct {
	ID        string               `json:"id"`
	Names     []string             `json:"names"`
	AppearsIn []RegistryAppearance `json:"appears_in"`
}

// RegistryNameGroup groups appearances by a shared name token. Informational —
// any token of length >= minRegistryTokenLen that appears under >= 2 distinct
// IDs becomes a group. A subset of these will also surface as `name_id_split`
// conflicts (those passing the Jaccard threshold).
type RegistryNameGroup struct {
	NamePattern string               `json:"name_pattern"`
	IDs         []string             `json:"ids"`
	AppearsIn   []RegistryAppearance `json:"appears_in"`
}

// RegistryView is the unified projection emitted by `c4-registry`.
type RegistryView struct {
	SchemaVersion string              `json:"schema_version"`
	ScannedDir    string              `json:"scanned_dir"`
	ScannedFiles  []string            `json:"scanned_files"`
	Elements      []RegistryElement   `json:"elements"`
	NamesToIDs    []RegistryNameGroup `json:"names_to_ids"`
	Conflicts     []RegistryConflict  `json:"conflicts"`
}

// unexported constants.
const (
	minRegistryTokenLen = 5
	// nameIDSplitJaccard is the Jaccard token-set similarity threshold above
	// which two IDs' names are deemed to refer to the same element.
	nameIDSplitJaccard = 0.5
	// nameIDSplitMinShared is the minimum number of shared tokens required
	// before name_id_split fires. Single-token overlap (e.g., a peer set of
	// "X skill", "Y skill", "Z skill") is too common to flag — natural for
	// architecturally peer elements. Two or more shared tokens (e.g.,
	// "Anthropic API Service" vs "Anthropic Service") is a genuine
	// same-thing-under-different-id signal.
	nameIDSplitMinShared = 2
)

// unexported variables.
var (
	registryFileRe  = regexp.MustCompile(`^c[1-4]-[a-z0-9-]+\.json$`)
	registryTokenRe = regexp.MustCompile(`[a-z0-9]+`)
)

// registryRecord is one normalized (file, id, name) row produced by parsing a
// single spec file. The intermediate representation is shared across L1/L2/L3.
type registryRecord struct {
	File string
	ID   string
	Name string
}

func c4Registry(ctx context.Context, args C4RegistryArgs) error {
	dir := args.Dir
	if dir == "" {
		dir = "architecture/c4"
	}
	files, records, err := scanRegistryDir(ctx, dir)
	if err != nil {
		return err
	}
	view := deriveRegistry(dir, files, records)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(view); err != nil {
		return fmt.Errorf("encode registry: %w", err)
	}
	return nil
}

// deriveRegistry computes the cross-file projection from a flat list of records.
// Pure: no I/O. Tested directly with hand-built record lists.
func deriveRegistry(scannedDir string, files []string, records []registryRecord) RegistryView {
	conflicts := []RegistryConflict{}
	conflicts = append(conflicts, detectIDCollisionsWithinFile(records)...)

	elements := groupByID(records)
	conflicts = append(conflicts, detectIDNameDrift(elements)...)

	nameGroups := groupBySharedToken(records)
	conflicts = append(conflicts, detectNameIDSplits(records, nameGroups)...)

	sortConflicts(conflicts)

	return RegistryView{
		SchemaVersion: "1",
		ScannedDir:    scannedDir,
		ScannedFiles:  files,
		Elements:      elements,
		NamesToIDs:    nameGroups,
		Conflicts:     conflicts,
	}
}

// detectIDCollisionsWithinFile flags any spec file that declares the same E-ID
// for two distinct elements (e.g., two records share file+id but have either
// different names or are simply duplicated).
func detectIDCollisionsWithinFile(records []registryRecord) []RegistryConflict {
	seen := map[string]map[string]bool{}
	conflicts := []RegistryConflict{}
	for _, rec := range records {
		if seen[rec.File] == nil {
			seen[rec.File] = map[string]bool{}
		}
		if seen[rec.File][rec.ID] {
			conflicts = append(conflicts, RegistryConflict{
				Kind:   "id_collision_within_file",
				ID:     rec.ID,
				Detail: fmt.Sprintf("%s declares %s more than once", rec.File, rec.ID),
				Evidence: []string{
					fmt.Sprintf("%s: %s = %q (duplicate)", rec.File, rec.ID, rec.Name),
				},
			})
			continue
		}
		seen[rec.File][rec.ID] = true
	}
	return conflicts
}

// detectIDNameDrift flags any RegistryElement whose appearances disagree on
// the element name.
func detectIDNameDrift(elements []RegistryElement) []RegistryConflict {
	conflicts := []RegistryConflict{}
	for _, element := range elements {
		if len(element.Names) <= 1 {
			continue
		}
		evidence := make([]string, 0, len(element.AppearsIn))
		for _, appearance := range element.AppearsIn {
			evidence = append(evidence, fmt.Sprintf("%s: %q", appearance.File, appearance.Name))
		}
		conflicts = append(conflicts, RegistryConflict{
			Kind:     "id_name_drift",
			ID:       element.ID,
			Detail:   fmt.Sprintf("%s has different names across files", element.ID),
			Evidence: evidence,
		})
	}
	return conflicts
}

// detectNameIDSplits returns name_id_split conflicts for any pair of distinct
// E-IDs whose token-set Jaccard similarity meets nameIDSplitJaccard.
func detectNameIDSplits(records []registryRecord, groups []RegistryNameGroup) []RegistryConflict {
	tokensByID := tokenSetsByID(records)
	idsWithSharedTokens := map[string]bool{}
	for _, group := range groups {
		for _, id := range group.IDs {
			idsWithSharedTokens[id] = true
		}
	}
	candidateIDs := sortedKeys(idsWithSharedTokens)

	conflicts := []RegistryConflict{}
	for indexA, idA := range candidateIDs {
		for _, idB := range candidateIDs[indexA+1:] {
			tokensA := tokensByID[idA]
			tokensB := tokensByID[idB]
			if len(tokensA) == 0 || len(tokensB) == 0 {
				continue
			}
			shared := intersectStringSets(tokensA, tokensB)
			if len(shared) < nameIDSplitMinShared {
				continue
			}
			jaccard := jaccardSimilarity(tokensA, tokensB)
			if jaccard < nameIDSplitJaccard {
				continue
			}
			pattern := strings.Join(sortedKeys(shared), " ")
			evidence := []string{}
			for _, rec := range records {
				if rec.ID == idA || rec.ID == idB {
					evidence = append(evidence, fmt.Sprintf("%s %q in %s", rec.ID, rec.Name, rec.File))
				}
			}
			conflicts = append(conflicts, RegistryConflict{
				Kind:        "name_id_split",
				NamePattern: pattern,
				Detail: fmt.Sprintf(
					"%s and %s share enough name tokens (Jaccard %.2f) to look like the same element",
					idA, idB, jaccard),
				Evidence: evidence,
			})
		}
	}
	return conflicts
}

// groupByID returns one RegistryElement per distinct E-ID, sorted by ID.
func groupByID(records []registryRecord) []RegistryElement {
	byID := map[string][]registryRecord{}
	for _, rec := range records {
		byID[rec.ID] = append(byID[rec.ID], rec)
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	elements := make([]RegistryElement, 0, len(ids))
	for _, id := range ids {
		recs := byID[id]
		nameSet := map[string]bool{}
		appearances := make([]RegistryAppearance, 0, len(recs))
		for _, rec := range recs {
			nameSet[rec.Name] = true
			appearances = append(appearances, RegistryAppearance{File: rec.File, Name: rec.Name})
		}
		elements = append(elements, RegistryElement{
			ID:        id,
			Names:     sortedKeys(nameSet),
			AppearsIn: appearances,
		})
	}
	return elements
}

// groupBySharedToken returns one RegistryNameGroup per name token (length >=
// minRegistryTokenLen) that appears under two or more distinct E-IDs.
// Informational only — name_id_split conflicts are a heuristic-filtered subset.
func groupBySharedToken(records []registryRecord) []RegistryNameGroup {
	tokenIDs := map[string]map[string]bool{}
	tokenAppearances := map[string][]RegistryAppearance{}
	for _, rec := range records {
		for _, token := range tokenizeRegistryName(rec.Name) {
			if tokenIDs[token] == nil {
				tokenIDs[token] = map[string]bool{}
			}
			if tokenIDs[token][rec.ID] {
				continue
			}
			tokenIDs[token][rec.ID] = true
			tokenAppearances[token] = append(tokenAppearances[token],
				RegistryAppearance{File: rec.File, ID: rec.ID, Name: rec.Name})
		}
	}
	tokens := make([]string, 0, len(tokenIDs))
	for token := range tokenIDs {
		if len(tokenIDs[token]) >= 2 {
			tokens = append(tokens, token)
		}
	}
	sort.Strings(tokens)
	groups := make([]RegistryNameGroup, 0, len(tokens))
	for _, token := range tokens {
		groups = append(groups, RegistryNameGroup{
			NamePattern: token,
			IDs:         sortedKeys(tokenIDs[token]),
			AppearsIn:   tokenAppearances[token],
		})
	}
	return groups
}

func intersectStringSets(left, right map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key := range left {
		if right[key] {
			out[key] = true
		}
	}
	return out
}

func jaccardSimilarity(left, right map[string]bool) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	intersection := 0
	for key := range left {
		if right[key] {
			intersection++
		}
	}
	union := len(left) + len(right) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func normalizeL1(filename string, raw []byte) ([]registryRecord, error) {
	var spec L1Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("parse L1 spec: %w", err)
	}
	ids, err := assignElementIDs(spec.Elements)
	if err != nil {
		return nil, fmt.Errorf("assign L1 element ids: %w", err)
	}
	records := make([]registryRecord, 0, len(ids))
	for _, item := range ids {
		records = append(records, registryRecord{
			File: filename, ID: item.ID, Name: item.Element.Name,
		})
	}
	return records, nil
}

func normalizeL2(filename string, raw []byte) ([]registryRecord, error) {
	var spec L2Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("parse L2 spec: %w", err)
	}
	ids, err := validateL2ElementIDs(spec.Elements)
	if err != nil {
		return nil, fmt.Errorf("validate L2 element ids: %w", err)
	}
	records := make([]registryRecord, 0, len(ids))
	for _, item := range ids {
		records = append(records, registryRecord{
			File: filename, ID: item.ID, Name: item.Element.Name,
		})
	}
	return records, nil
}

// normalizeL3 emits one record per L3 element plus one for the focus carry-
// over. Both contribute to cross-spec ID/name tracking the same way as L1/L2.
func normalizeL3(filename string, raw []byte) ([]registryRecord, error) {
	var spec L3Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("parse L3 spec: %w", err)
	}
	records := make([]registryRecord, 0, len(spec.Elements)+1)
	records = append(records, registryRecord{
		File: filename, ID: spec.Focus.ID, Name: spec.Focus.Name,
	})
	for _, element := range spec.Elements {
		records = append(records, registryRecord{
			File: filename, ID: element.ID, Name: element.Name,
		})
	}
	return records, nil
}

// normalizeSpec parses raw as the level-appropriate spec type and emits one
// registryRecord per element using the same ID-assignment logic the level's
// build target uses.
func normalizeSpec(filename string, raw []byte) ([]registryRecord, error) {
	var meta struct {
		Level int `json:"level"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("parse meta: %w", err)
	}
	switch meta.Level {
	case 1:
		return normalizeL1(filename, raw)
	case 2:
		return normalizeL2(filename, raw)
	case 3:
		return normalizeL3(filename, raw)
	default:
		return nil, fmt.Errorf("unsupported level %d", meta.Level)
	}
}

// scanRegistryDir walks dir, parses each `c[1-4]-*.json` per its declared
// `level`, and returns the list of accepted filenames plus the normalized
// (file, id, name) records. Malformed or unrecognized-level files are logged to
// stderr and skipped.
func scanRegistryDir(ctx context.Context, dir string) ([]string, []registryRecord, error) {
	_ = ctx
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	files := []string{}
	records := []registryRecord{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !registryFileRe.MatchString(name) {
			continue
		}
		path := filepath.Join(dir, name)
		raw, readErr := os.ReadFile(path) //nolint:gosec // path is under user-supplied dir, not user input
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "registry: skip %s: read: %v\n", name, readErr)
			continue
		}
		recs, recErr := normalizeSpec(name, raw)
		if recErr != nil {
			fmt.Fprintf(os.Stderr, "registry: skip %s: %v\n", name, recErr)
			continue
		}
		files = append(files, name)
		records = append(records, recs...)
	}
	sort.Strings(files)
	return files, records, nil
}

func sortConflicts(conflicts []RegistryConflict) {
	sort.SliceStable(conflicts, func(i, j int) bool {
		if conflicts[i].Kind != conflicts[j].Kind {
			return conflicts[i].Kind < conflicts[j].Kind
		}
		if conflicts[i].ID != conflicts[j].ID {
			return conflicts[i].ID < conflicts[j].ID
		}
		return conflicts[i].NamePattern < conflicts[j].NamePattern
	})
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// tokenSetsByID returns one token set per E-ID, taking the union over all
// names that ID has across all files.
func tokenSetsByID(records []registryRecord) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, rec := range records {
		if out[rec.ID] == nil {
			out[rec.ID] = map[string]bool{}
		}
		for _, token := range tokenizeRegistryName(rec.Name) {
			out[rec.ID][token] = true
		}
	}
	return out
}

// tokenizeRegistryName lowercases name, splits on non-[a-z0-9] runs, and drops
// tokens shorter than minRegistryTokenLen.
func tokenizeRegistryName(name string) []string {
	lower := strings.ToLower(name)
	matches := registryTokenRe.FindAllString(lower, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= minRegistryTokenLen {
			out = append(out, match)
		}
	}
	return out
}
