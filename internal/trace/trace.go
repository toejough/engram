// Package trace manages the traceability matrix for project artifacts.
package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/config"
)

// TraceFile is the filename for the traceability matrix.
const TraceFile = "traceability.toml"

// realConfigFS implements config.ConfigFS using the real file system.
type realConfigFS struct{}

func (r *realConfigFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *realConfigFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// idPattern matches a traceability ID.
var idPattern = regexp.MustCompile(`^(ISSUE|REQ|DES|ARCH|TASK)-\d{3}$`)

// Link represents a single traceability link.
type Link struct {
	From string   `toml:"from"`
	To   []string `toml:"to"`
}

// Matrix is the complete traceability matrix.
type Matrix struct {
	Links []Link `toml:"links"`
}

// ValidID returns true if the ID matches the expected pattern.
func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

// Add adds traceability links from one ID to one or more target IDs.
// Creates the file if it doesn't exist. Rejects duplicate links.
func Add(dir, from string, to []string) error {
	if !ValidID(from) {
		return fmt.Errorf("invalid source ID: %s (must match ISSUE|REQ|DES|ARCH|TASK-NNN)", from)
	}

	for _, t := range to {
		if !ValidID(t) {
			return fmt.Errorf("invalid target ID: %s (must match ISSUE|REQ|DES|ARCH|TASK-NNN)", t)
		}
	}

	// ISSUE can only link to REQ
	if strings.HasPrefix(from, "ISSUE-") {
		for _, t := range to {
			if !strings.HasPrefix(t, "REQ-") {
				return fmt.Errorf("ISSUE can only link to REQ (got %s)", t)
			}
		}
	}

	m, err := load(dir)
	if err != nil {
		return err
	}

	// Find or create the link entry for this source
	var link *Link

	for i := range m.Links {
		if m.Links[i].From == from {
			link = &m.Links[i]

			break
		}
	}

	if link == nil {
		m.Links = append(m.Links, Link{From: from, To: nil})
		link = &m.Links[len(m.Links)-1]
	}

	// Add targets, rejecting duplicates
	existing := make(map[string]bool, len(link.To))
	for _, t := range link.To {
		existing[t] = true
	}

	for _, t := range to {
		if existing[t] {
			return fmt.Errorf("duplicate link: %s → %s already exists", from, t)
		}

		link.To = append(link.To, t)
		existing[t] = true
	}

	return save(dir, m)
}

// ValidateResult holds the results of a traceability validation.
type ValidateResult struct {
	Pass           bool     `json:"pass"`
	OrphanIDs      []string `json:"orphan_ids"`
	UnlinkedIDs    []string `json:"unlinked_ids"`
	MissingCoverage []MissingLink `json:"missing_coverage"`
}

// MissingLink represents a required but missing traceability link.
type MissingLink struct {
	ID          string `json:"id"`
	MissingType string `json:"missing_type"`
}

// Validate checks traceability coverage and consistency.
// It reads artifact files to discover IDs and compares against the matrix.
func Validate(dir string) (ValidateResult, error) {
	// Load config to get artifact paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ValidateResult{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(dir, homeDir, &realConfigFS{})
	if err != nil {
		return ValidateResult{}, fmt.Errorf("failed to load config: %w", err)
	}

	m, err := load(dir)
	if err != nil {
		return ValidateResult{}, err
	}

	// Collect all IDs referenced in the matrix
	matrixIDs := make(map[string]bool)

	for _, link := range m.Links {
		matrixIDs[link.From] = true

		for _, t := range link.To {
			matrixIDs[t] = true
		}
	}

	// Scan artifact files for embedded IDs
	artifactIDs, err := scanArtifacts(dir, cfg)
	if err != nil {
		return ValidateResult{}, err
	}

	result := ValidateResult{Pass: true}

	// Orphans: in matrix but not in any artifact
	for id := range matrixIDs {
		if !artifactIDs[id] {
			result.OrphanIDs = append(result.OrphanIDs, id)
			result.Pass = false
		}
	}

	// Unlinked: in artifact but not in matrix
	for id := range artifactIDs {
		if !matrixIDs[id] {
			result.UnlinkedIDs = append(result.UnlinkedIDs, id)
			result.Pass = false
		}
	}

	// Check coverage rules
	// Build downstream map: from → [to]
	downstream := make(map[string][]string)
	for _, link := range m.Links {
		downstream[link.From] = append(downstream[link.From], link.To...)
	}

	for id := range artifactIDs {
		targets := downstream[id]

		switch {
		case strings.HasPrefix(id, "REQ-"):
			// REQ must link to DES or ARCH (design is mandatory)
			if !hasPrefix(targets, "DES-") && !hasPrefix(targets, "ARCH-") {
				result.MissingCoverage = append(result.MissingCoverage, MissingLink{
					ID:          id,
					MissingType: "DES or ARCH",
				})
				result.Pass = false
			}
		case strings.HasPrefix(id, "DES-"):
			if !hasPrefix(targets, "ARCH-") {
				result.MissingCoverage = append(result.MissingCoverage, MissingLink{
					ID:          id,
					MissingType: "ARCH",
				})
				result.Pass = false
			}
		case strings.HasPrefix(id, "ARCH-"):
			if !hasPrefix(targets, "TASK-") {
				result.MissingCoverage = append(result.MissingCoverage, MissingLink{
					ID:          id,
					MissingType: "TASK",
				})
				result.Pass = false
			}
		}
	}

	return result, nil
}

// ImpactResult holds forward or backward impact analysis results.
type ImpactResult struct {
	SourceID    string   `json:"source_id"`
	AffectedIDs []string `json:"affected_ids"`
	Reverse     bool     `json:"reverse"`
}

// Impact performs forward or backward impact analysis.
// Forward: given REQ-003, returns all DES, ARCH, TASK that trace from it.
// Backward (reverse): given TASK-005, returns all upstream IDs.
func Impact(dir, id string, reverse bool) (ImpactResult, error) {
	if !ValidID(id) {
		return ImpactResult{}, fmt.Errorf("invalid ID: %s", id)
	}

	m, err := load(dir)
	if err != nil {
		return ImpactResult{}, err
	}

	var graph map[string][]string
	if reverse {
		graph = buildReverseGraph(m)
	} else {
		graph = buildForwardGraph(m)
	}

	visited := make(map[string]bool)
	var result []string

	walk(graph, id, visited, &result)

	return ImpactResult{
		SourceID:    id,
		AffectedIDs: result,
		Reverse:     reverse,
	}, nil
}

func walk(graph map[string][]string, id string, visited map[string]bool, result *[]string) {
	for _, next := range graph[id] {
		if visited[next] {
			continue
		}

		visited[next] = true
		*result = append(*result, next)

		walk(graph, next, visited, result)
	}
}

func buildForwardGraph(m Matrix) map[string][]string {
	g := make(map[string][]string)

	for _, link := range m.Links {
		g[link.From] = append(g[link.From], link.To...)
	}

	return g
}

func buildReverseGraph(m Matrix) map[string][]string {
	g := make(map[string][]string)

	for _, link := range m.Links {
		for _, t := range link.To {
			g[t] = append(g[t], link.From)
		}
	}

	return g
}

func hasPrefix(targets []string, prefix string) bool {
	for _, t := range targets {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}

	return false
}

// scanArtifacts scans known artifact files for embedded traceability IDs.
// Uses config to resolve artifact paths (typically docs/ subdirectory).
func scanArtifacts(dir string, cfg *config.ProjectConfig) (map[string]bool, error) {
	ids := make(map[string]bool)

	// Get artifact paths from config
	artifactPaths := []string{
		cfg.ResolvePath("issues"),
		cfg.ResolvePath("requirements"),
		cfg.ResolvePath("design"),
		cfg.ResolvePath("architecture"),
		cfg.ResolvePath("tasks"),
	}

	pattern := regexp.MustCompile(`(ISSUE|REQ|DES|ARCH|TASK)-\d{3}`)

	for _, relPath := range artifactPaths {
		path := filepath.Join(dir, relPath)

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("failed to read %s: %w", relPath, err)
		}

		matches := pattern.FindAllString(string(data), -1)
		for _, m := range matches {
			ids[m] = true
		}
	}

	return ids, nil
}

func load(dir string) (Matrix, error) {
	path := filepath.Join(dir, TraceFile)

	var m Matrix

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Matrix{}, nil
	}

	if _, err := toml.DecodeFile(path, &m); err != nil {
		return Matrix{}, fmt.Errorf("failed to read traceability file: %w", err)
	}

	return m, nil
}

func save(dir string, m Matrix) error {
	path := filepath.Join(dir, TraceFile)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create traceability file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := toml.NewEncoder(f).Encode(m); err != nil {
		return fmt.Errorf("failed to encode traceability matrix: %w", err)
	}

	return nil
}
