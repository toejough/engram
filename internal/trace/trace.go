// Package trace manages the traceability matrix for project artifacts.
package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
var idPattern = regexp.MustCompile(`^(ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+$`)

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
		return fmt.Errorf("invalid source ID: %s (must match ISSUE|REQ|DES|ARCH|TASK-N)", from)
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
// Note: Does NOT scan tests.md - TEST tracing is in source files.
func scanArtifacts(dir string, cfg *config.ProjectConfig) (map[string]bool, error) {
	ids := make(map[string]bool)

	// Get artifact paths from config
	// Note: tests.md is NOT scanned - TEST tracing is in source files
	artifactPaths := []string{
		cfg.ResolvePath("issues"),
		cfg.ResolvePath("requirements"),
		cfg.ResolvePath("design"),
		cfg.ResolvePath("architecture"),
		cfg.ResolvePath("tasks"),
	}

	pattern := regexp.MustCompile(`(ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+`)

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

// RepairResult holds the results of a repair analysis.
type RepairResult struct {
	DanglingRefs []string       `json:"dangling_refs"` // IDs referenced in Traces to: but not defined
	DuplicateIDs []string       `json:"duplicate_ids"` // IDs defined more than once (before fix)
	Renumbered   []RenumberInfo `json:"renumbered"`    // IDs that were renumbered to fix duplicates
	Escalations  []EscalationInfo `json:"escalations"` // Issues that couldn't be auto-fixed
}

// RenumberInfo records an ID renumbering action.
type RenumberInfo struct {
	OldID string `json:"old_id"`
	NewID string `json:"new_id"`
	File  string `json:"file"`
}

// EscalationInfo records an issue that needs manual resolution.
type EscalationInfo struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	File   string `json:"file"`
}

// idLocation tracks where an ID is defined.
type idLocation struct {
	ID   string
	File string // relative path
}

// Repair analyzes artifact files for traceability issues and auto-fixes when possible.
// It renumbers duplicate IDs and creates escalations for dangling references.
func Repair(dir string) (RepairResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return RepairResult{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(dir, homeDir, &realConfigFS{})
	if err != nil {
		return RepairResult{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Scan all artifact files
	definedIDs := make(map[string][]idLocation) // ID -> list of locations
	referencedIDs := make(map[string]bool)      // IDs referenced in Traces to:
	maxIDByPrefix := make(map[string]int)       // prefix -> max number

	// Note: tests.md is NOT scanned - TEST tracing is in source files
	artifactPaths := []string{
		cfg.ResolvePath("issues"),
		cfg.ResolvePath("requirements"),
		cfg.ResolvePath("design"),
		cfg.ResolvePath("architecture"),
		cfg.ResolvePath("tasks"),
	}

	// Also look for feature-specific files (no tests-*.md - TEST tracing is in source)
	docsDir := dir
	featurePatterns := []string{
		filepath.Join(docsDir, "design-*.md"),
		filepath.Join(docsDir, "requirements-*.md"),
		filepath.Join(docsDir, "architecture-*.md"),
	}

	for _, pattern := range featurePatterns {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			for _, match := range matches {
				relPath, _ := filepath.Rel(dir, match)
				artifactPaths = append(artifactPaths, relPath)
			}
		}
	}

	// Patterns for parsing
	idDefPattern := regexp.MustCompile(`^###\s+((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-(\d+)):\s*`)
	tracesToPattern := regexp.MustCompile(`\*\*Traces to:\*\*\s*(.+)`)
	idRefPattern := regexp.MustCompile(`((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+)`)

	for _, relPath := range artifactPaths {
		path := filepath.Join(dir, relPath)

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return RepairResult{}, fmt.Errorf("failed to read %s: %w", relPath, err)
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			// Check for ID definitions
			if match := idDefPattern.FindStringSubmatch(line); match != nil {
				id := match[1]
				definedIDs[id] = append(definedIDs[id], idLocation{ID: id, File: relPath})

				// Track max ID number by prefix
				prefix := strings.Split(id, "-")[0]
				numStr := match[2]
				var num int
				_, _ = fmt.Sscanf(numStr, "%d", &num)
				if num > maxIDByPrefix[prefix] {
					maxIDByPrefix[prefix] = num
				}
			}

			// Check for Traces to: references
			if match := tracesToPattern.FindStringSubmatch(line); match != nil {
				refs := idRefPattern.FindAllString(match[1], -1)
				for _, ref := range refs {
					referencedIDs[ref] = true
				}
			}
		}
	}

	result := RepairResult{}

	// Find dangling references: referenced but not defined
	// ISSUE IDs are exempt because they are defined at repo root, not in project directories
	for ref := range referencedIDs {
		if len(definedIDs[ref]) == 0 && !strings.HasPrefix(ref, "ISSUE-") {
			result.DanglingRefs = append(result.DanglingRefs, ref)
			// Create escalation for dangling ref
			result.Escalations = append(result.Escalations, EscalationInfo{
				ID:     ref,
				Reason: "dangling reference: ID referenced in Traces to: but not defined",
			})
		}
	}

	// Find and fix duplicate IDs
	for id, locations := range definedIDs {
		if len(locations) > 1 {
			result.DuplicateIDs = append(result.DuplicateIDs, id)

			// Keep first occurrence, renumber subsequent ones
			prefix := strings.Split(id, "-")[0]
			for i := 1; i < len(locations); i++ {
				loc := locations[i]

				// Generate new ID
				maxIDByPrefix[prefix]++
				newID := fmt.Sprintf("%s-%d", prefix, maxIDByPrefix[prefix])

				// Update the file
				path := filepath.Join(dir, loc.File)
				content, err := os.ReadFile(path)
				if err != nil {
					continue
				}

				// Replace the ID in the file
				newContent := strings.ReplaceAll(string(content), id, newID)
				if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
					continue
				}

				result.Renumbered = append(result.Renumbered, RenumberInfo{
					OldID: id,
					NewID: newID,
					File:  loc.File,
				})
			}
		}
	}

	return result, nil
}

// ValidateV2ArtifactsResult holds the results of artifact-based validation.
type ValidateV2ArtifactsResult struct {
	Pass        bool     `json:"pass"`
	OrphanIDs   []string `json:"orphan_ids"`   // IDs referenced in Traces to: but not defined
	UnlinkedIDs []string `json:"unlinked_ids"` // IDs defined but nothing traces to them
}

// TestTrace represents a test with traceability information from source file comments.
type TestTrace struct {
	ID       string   // TEST-NNN
	Function string   // TestFunctionName
	File     string   // path/to/file_test.go
	Line     int      // line number
	TracesTo []string // [TASK-NNN, ...]
}

// scanTestFiles scans Go test files for TEST-NNN comments with traces.
// Pattern: // TEST-NNN: description
//
//	// traces: TARGET-NNN[, TARGET-NNN...]
//	func TestFunctionName(t *testing.T) {
func scanTestFiles(dir string) (map[string]TestTrace, error) {
	result := make(map[string]TestTrace)

	// Pattern for TEST-NNN comment followed by traces comment
	// We look for: // TEST-NNN: ...
	//              // traces: ...
	testIDPattern := regexp.MustCompile(`^//\s*(TEST-\d+):\s*(.*)`)
	tracesPattern := regexp.MustCompile(`^//\s*traces:\s*(.+)`)
	funcPattern := regexp.MustCompile(`^func\s+(Test\w+)\s*\(`)
	idRefPattern := regexp.MustCompile(`((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+)`)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip vendor directory
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Only process *_test.go files
		if info.IsDir() || !strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// Skip trace_test.go which contains test fixtures with TEST-NNN patterns
		if info.Name() == "trace_test.go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		relPath, _ := filepath.Rel(dir, path)
		lines := strings.Split(string(data), "\n")

		var currentTestID string
		var currentDesc string
		var currentTraces []string
		var testIDLine int

		for lineNum, line := range lines {
			// Check for TEST-NNN comment
			if match := testIDPattern.FindStringSubmatch(line); match != nil {
				currentTestID = match[1]
				currentDesc = match[2]
				testIDLine = lineNum + 1
				currentTraces = nil
				continue
			}

			// Check for traces comment (must follow TEST-NNN)
			if currentTestID != "" && currentTraces == nil {
				if match := tracesPattern.FindStringSubmatch(line); match != nil {
					refs := idRefPattern.FindAllString(match[1], -1)
					currentTraces = refs
					continue
				}
			}

			// Check for function declaration
			if currentTestID != "" {
				if match := funcPattern.FindStringSubmatch(line); match != nil {
					result[currentTestID] = TestTrace{
						ID:       currentTestID,
						Function: match[1],
						File:     relPath,
						Line:     testIDLine,
						TracesTo: currentTraces,
					}
					// Reset for next test
					currentTestID = ""
					currentDesc = ""
					currentTraces = nil
				}
			}

			// Reset if we hit a blank line or other content
			if strings.TrimSpace(line) == "" || (!strings.HasPrefix(strings.TrimSpace(line), "//") && !strings.HasPrefix(strings.TrimSpace(line), "func")) {
				if currentTestID != "" && currentTraces == nil {
					// TEST-NNN without traces - still record it
					result[currentTestID] = TestTrace{
						ID:       currentTestID,
						Function: "",
						File:     relPath,
						Line:     testIDLine,
						TracesTo: nil,
					}
				}
				currentTestID = ""
				currentDesc = ""
				currentTraces = nil
			}
		}

		// Handle case where file ends with a TEST comment
		if currentTestID != "" {
			result[currentTestID] = TestTrace{
				ID:       currentTestID,
				Function: "",
				File:     relPath,
				Line:     testIDLine,
				TracesTo: currentTraces,
			}
		}

		// Suppress unused variable warning
		_ = currentDesc

		return nil
	})

	return result, err
}

// phaseAllowsUnlinked returns which ID prefixes are allowed to be unlinked at a given phase.
// The workflow creates artifacts progressively:
//   - At design-complete: DES exists but ARCH doesn't trace to it yet
//   - At architect-complete: ARCH exists but TASK doesn't trace to it yet
//   - At breakdown-complete: TASK exists but TEST doesn't trace to it yet
//   - At tdd_commit and later: full chain required
func phaseAllowsUnlinked(phase string) map[string]bool {
	allowed := make(map[string]bool)

	switch phase {
	case "pm_produce", "pm_qa", "pm_decide", "pm_commit":
		// Only REQ exists, nothing traces to it - REQ is always allowed as root
		// No special exemptions needed
	case "design_produce", "design_qa", "design_decide", "design_commit":
		// DES exists, but ARCH doesn't exist yet to trace to it
		allowed["DES-"] = true
	case "arch_produce", "arch_qa", "arch_decide", "arch_commit":
		// ARCH exists, but TASK doesn't exist yet to trace to it
		// DES should have ARCH tracing to it now
		allowed["ARCH-"] = true
	case "breakdown_produce", "breakdown_qa", "breakdown_decide", "breakdown_commit":
		// TASK exists, but TEST doesn't exist yet to trace to it
		// ARCH should have TASK tracing to it now
		allowed["TASK-"] = true
	case "":
		// No phase specified = strictest validation (default behavior)
	default:
		// All other phases (tdd_*, documentation_*, etc.)
		// require full chain - no exemptions
	}

	return allowed
}

// validPhases lists all valid phase names for validation.
// Generated from the flat state machine defined in workflows.toml.
var validPhases = map[string]bool{
	"":     true, // empty = strictest
	"init": true,
	// PM phases
	"pm_produce": true, "pm_qa": true, "pm_decide": true, "pm_commit": true,
	// Design phases
	"design_produce": true, "design_qa": true, "design_decide": true, "design_commit": true,
	// Architecture phases
	"arch_produce": true, "arch_qa": true, "arch_decide": true, "arch_commit": true,
	// Breakdown phases
	"breakdown_produce": true, "breakdown_qa": true, "breakdown_decide": true, "breakdown_commit": true,
	// Item execution
	"item_select": true, "item_fork": true, "worktree_create": true,
	// TDD red
	"tdd_red_produce": true, "tdd_red_qa": true, "tdd_red_decide": true,
	// TDD green
	"tdd_green_produce": true, "tdd_green_qa": true, "tdd_green_decide": true,
	// TDD refactor
	"tdd_refactor_produce": true, "tdd_refactor_qa": true, "tdd_refactor_decide": true,
	// TDD commit and item lifecycle
	"tdd_commit": true, "item_escalated": true, "item_parked": true,
	"merge_acquire": true, "rebase": true, "merge": true,
	"worktree_cleanup": true, "item_join": true, "item_assess": true, "items_done": true,
	// Documentation phases
	"documentation_produce": true, "documentation_qa": true, "documentation_decide": true, "documentation_commit": true,
	// Alignment phases
	"alignment_produce": true, "alignment_qa": true, "alignment_decide": true, "alignment_commit": true,
	// Retro phases
	"retro_produce": true, "retro_qa": true, "retro_decide": true, "retro_commit": true,
	// Summary phases
	"summary_produce": true, "summary_qa": true, "summary_decide": true, "summary_commit": true,
	// Ending phases
	"issue_update": true, "next_steps": true, "complete": true,
	// Align workflow phases
	"align_infer_reqs_produce": true, "align_infer_design_produce": true,
	"align_infer_arch_produce": true, "align_infer_tests_produce": true,
	"align_crosscut_qa": true, "align_crosscut_decide": true, "align_artifact_commit": true,
	// Error handling
	"phase_blocked": true,
}

// ValidateV2Artifacts validates traceability by scanning artifact files directly.
// Unlike Validate, this doesn't use the traceability.toml matrix.
// - Orphan: ID in **Traces to:** field but not defined as header (### ID: Title)
// - Unlinked: ID defined but not connected to chain:
//   - DES, ARCH, TASK: nothing traces TO them
//   - TEST: doesn't trace TO anything (no **Traces to:** field or // traces: comment)
//
// The optional phase parameter enables phase-aware validation:
//   - At design-complete: DES allowed to be unlinked (ARCH doesn't exist yet)
//   - At architect-complete: ARCH allowed to be unlinked (TASK doesn't exist yet)
//   - At breakdown_commit: TASK allowed to be unlinked (TEST doesn't exist yet)
//   - Without phase or at later phases: full chain required (strictest)
func ValidateV2Artifacts(dir string, phase ...string) (ValidateV2ArtifactsResult, error) {
	// Validate and extract phase parameter
	currentPhase := ""
	if len(phase) > 0 {
		currentPhase = phase[0]
	}
	if !validPhases[currentPhase] {
		return ValidateV2ArtifactsResult{}, fmt.Errorf("invalid phase: %q", currentPhase)
	}

	allowedUnlinked := phaseAllowsUnlinked(currentPhase)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ValidateV2ArtifactsResult{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(dir, homeDir, &realConfigFS{})
	if err != nil {
		return ValidateV2ArtifactsResult{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Collect defined IDs and referenced IDs from artifacts
	definedIDs := make(map[string]bool)    // ID → true if defined as header
	referencedIDs := make(map[string]bool) // ID → true if referenced in Traces to:
	hasTracesTo := make(map[string]bool)   // ID → true if has a Traces to: field

	// Note: tests.md is NOT scanned - TEST tracing is in source files via comments
	artifactPaths := []string{
		cfg.ResolvePath("issues"),
		cfg.ResolvePath("requirements"),
		cfg.ResolvePath("design"),
		cfg.ResolvePath("architecture"),
		cfg.ResolvePath("tasks"),
	}

	// Also look for feature-specific files (no tests-*.md - TEST tracing is in source)
	docsDir := dir
	featurePatterns := []string{
		filepath.Join(docsDir, "design-*.md"),
		filepath.Join(docsDir, "requirements-*.md"),
		filepath.Join(docsDir, "architecture-*.md"),
	}

	for _, pattern := range featurePatterns {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			for _, match := range matches {
				relPath, _ := filepath.Rel(dir, match)
				artifactPaths = append(artifactPaths, relPath)
			}
		}
	}

	// Patterns for parsing
	idDefPattern := regexp.MustCompile(`^###\s+((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+):\s*`)
	tracesToPattern := regexp.MustCompile(`\*\*Traces to:\*\*\s*(.+)`)
	idRefPattern := regexp.MustCompile(`((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+)`)

	for _, relPath := range artifactPaths {
		path := filepath.Join(dir, relPath)

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ValidateV2ArtifactsResult{}, fmt.Errorf("failed to read %s: %w", relPath, err)
		}

		lines := strings.Split(string(data), "\n")
		var currentID string
		for _, line := range lines {
			// Check for ID definitions
			if match := idDefPattern.FindStringSubmatch(line); match != nil {
				currentID = match[1]
				definedIDs[currentID] = true
			}

			// Check for Traces to: references
			if match := tracesToPattern.FindStringSubmatch(line); match != nil {
				refs := idRefPattern.FindAllString(match[1], -1)
				for _, ref := range refs {
					referencedIDs[ref] = true
				}
				// Mark that this ID has a Traces to: field
				if currentID != "" {
					hasTracesTo[currentID] = true
				}
			}
		}
	}

	// Scan test source files for TEST-NNN comments
	testTraces, err := scanTestFiles(dir)
	if err != nil {
		return ValidateV2ArtifactsResult{}, fmt.Errorf("failed to scan test files: %w", err)
	}

	// Integrate test file results
	for id, trace := range testTraces {
		definedIDs[id] = true
		if len(trace.TracesTo) > 0 {
			hasTracesTo[id] = true
			// Also add the trace targets to referencedIDs
			for _, target := range trace.TracesTo {
				referencedIDs[target] = true
			}
		}
	}

	result := ValidateV2ArtifactsResult{Pass: true}

	// Orphan: referenced in Traces to: but not defined
	// ISSUE IDs are exempt from orphan checks because issues are always defined
	// at the repo-level docs/issues.md, not in project subdirectories. When
	// validation runs from a project subdirectory, ISSUE-NNN references are
	// cross-boundary references to the repo root and cannot be resolved locally.
	for ref := range referencedIDs {
		if !definedIDs[ref] && !strings.HasPrefix(ref, "ISSUE-") {
			result.OrphanIDs = append(result.OrphanIDs, ref)
			result.Pass = false
		}
	}

	// Unlinked: defined but not connected to the chain
	// - DES, ARCH, TASK: need something tracing TO them (unless phase allows it)
	// - TEST: needs to trace TO something (must have Traces to: field)
	// - ISSUE, REQ: can be roots (exempt)
	for id := range definedIDs {
		if strings.HasPrefix(id, "TEST-") {
			// TEST is a leaf node - nothing traces TO it, but it must trace TO something
			if !hasTracesTo[id] {
				result.UnlinkedIDs = append(result.UnlinkedIDs, id)
				result.Pass = false
			}
		} else if !strings.HasPrefix(id, "ISSUE-") && !strings.HasPrefix(id, "REQ-") {
			// DES, ARCH, TASK need something tracing TO them
			if !referencedIDs[id] {
				// Check if this ID type is allowed to be unlinked at the current phase
				isAllowed := false
				for prefix := range allowedUnlinked {
					if strings.HasPrefix(id, prefix) {
						isAllowed = true
						break
					}
				}
				if !isAllowed {
					result.UnlinkedIDs = append(result.UnlinkedIDs, id)
					result.Pass = false
				}
			}
		}
	}

	return result, nil
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

// ShowNode represents a node in the trace graph for JSON output.
type ShowNode struct {
	ID       string `json:"id"`
	Orphan   bool   `json:"orphan,omitempty"`
	Unlinked bool   `json:"unlinked,omitempty"`
}

// ShowEdge represents an edge in the trace graph for JSON output.
type ShowEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ShowGraph represents the complete trace graph for JSON output.
type ShowGraph struct {
	Nodes []ShowNode `json:"nodes"`
	Edges []ShowEdge `json:"edges"`
}

// Show returns a visualization of the traceability graph.
// Format can be "ascii" for an ASCII tree or "json" for a JSON graph.
func Show(dir, format string) (string, error) {
	if format != "ascii" && format != "json" {
		return "", fmt.Errorf("invalid format: %s (must be 'ascii' or 'json')", format)
	}

	// Use ValidateV2Artifacts to get orphan/unlinked status
	result, err := ValidateV2Artifacts(dir)
	if err != nil {
		return "", err
	}

	// Build graph from artifact files
	graph, err := buildShowGraph(dir, result)
	if err != nil {
		return "", err
	}

	if format == "json" {
		return formatJSON(graph)
	}

	return formatASCII(graph, result)
}

// buildShowGraph constructs the graph from artifact files.
func buildShowGraph(dir string, validation ValidateV2ArtifactsResult) (ShowGraph, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ShowGraph{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(dir, homeDir, &realConfigFS{})
	if err != nil {
		return ShowGraph{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Collect all defined IDs and edges
	definedIDs := make(map[string]bool)
	edges := make([]ShowEdge, 0)

	artifactPaths := []string{
		cfg.ResolvePath("issues"),
		cfg.ResolvePath("requirements"),
		cfg.ResolvePath("design"),
		cfg.ResolvePath("architecture"),
		cfg.ResolvePath("tasks"),
	}

	// Also look for feature-specific files
	docsDir := dir
	featurePatterns := []string{
		filepath.Join(docsDir, "design-*.md"),
		filepath.Join(docsDir, "requirements-*.md"),
		filepath.Join(docsDir, "architecture-*.md"),
	}

	for _, pattern := range featurePatterns {
		matches, globErr := filepath.Glob(pattern)
		if globErr == nil {
			for _, match := range matches {
				relPath, _ := filepath.Rel(dir, match)
				artifactPaths = append(artifactPaths, relPath)
			}
		}
	}

	// Patterns for parsing
	idDefPattern := regexp.MustCompile(`^###\s+((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+):\s*`)
	tracesToPattern := regexp.MustCompile(`\*\*Traces to:\*\*\s*(.+)`)
	idRefPattern := regexp.MustCompile(`((?:ISSUE|REQ|DES|ARCH|TASK|TEST)-\d+)`)

	for _, relPath := range artifactPaths {
		path := filepath.Join(dir, relPath)

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ShowGraph{}, fmt.Errorf("failed to read %s: %w", relPath, err)
		}

		lines := strings.Split(string(data), "\n")
		var currentID string
		for _, line := range lines {
			// Check for ID definitions
			if match := idDefPattern.FindStringSubmatch(line); match != nil {
				currentID = match[1]
				definedIDs[currentID] = true
			}

			// Check for Traces to: references
			if match := tracesToPattern.FindStringSubmatch(line); match != nil {
				refs := idRefPattern.FindAllString(match[1], -1)
				for _, ref := range refs {
					if currentID != "" {
						edges = append(edges, ShowEdge{From: currentID, To: ref})
					}
				}
			}
		}
	}

	// Also scan test files for TEST traces
	testTraces, err := scanTestFiles(dir)
	if err != nil {
		return ShowGraph{}, fmt.Errorf("failed to scan test files: %w", err)
	}

	for id, testTrace := range testTraces {
		definedIDs[id] = true
		for _, target := range testTrace.TracesTo {
			edges = append(edges, ShowEdge{From: id, To: target})
		}
	}

	// Build orphan and unlinked sets for quick lookup
	orphanSet := make(map[string]bool)
	for _, id := range validation.OrphanIDs {
		orphanSet[id] = true
	}

	unlinkedSet := make(map[string]bool)
	for _, id := range validation.UnlinkedIDs {
		unlinkedSet[id] = true
	}

	// Create node list
	var nodes []ShowNode
	allIDs := make(map[string]bool)

	// Add defined IDs
	for id := range definedIDs {
		allIDs[id] = true
	}

	// Add orphan IDs (referenced but not defined)
	for _, id := range validation.OrphanIDs {
		allIDs[id] = true
	}

	// Sort IDs for deterministic output
	sortedIDs := make([]string, 0, len(allIDs))
	for id := range allIDs {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	for _, id := range sortedIDs {
		nodes = append(nodes, ShowNode{
			ID:       id,
			Orphan:   orphanSet[id],
			Unlinked: unlinkedSet[id],
		})
	}

	return ShowGraph{Nodes: nodes, Edges: edges}, nil
}

func formatJSON(graph ShowGraph) (string, error) {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

func formatASCII(graph ShowGraph, validation ValidateV2ArtifactsResult) (string, error) {
	// Build adjacency list: parent -> children (reverse of edges direction)
	// edges go from child to parent (e.g., DES-001 -> REQ-001)
	// for tree display, we want parent -> children
	children := make(map[string][]string)
	hasParent := make(map[string]bool)

	for _, edge := range graph.Edges {
		// edge.From traces to edge.To, meaning edge.To is the parent
		children[edge.To] = append(children[edge.To], edge.From)
		hasParent[edge.From] = true
	}

	// Build sets for markers
	orphanSet := make(map[string]bool)
	for _, id := range validation.OrphanIDs {
		orphanSet[id] = true
	}

	unlinkedSet := make(map[string]bool)
	for _, id := range validation.UnlinkedIDs {
		unlinkedSet[id] = true
	}

	// Find root nodes (IDs that have no parent, or are orphans referenced but not defined)
	var roots []string
	for _, node := range graph.Nodes {
		if !hasParent[node.ID] {
			roots = append(roots, node.ID)
		}
	}
	sort.Strings(roots)

	// Handle case with no nodes
	if len(graph.Nodes) == 0 {
		return "(empty trace graph)\n", nil
	}

	var sb strings.Builder
	visited := make(map[string]bool)

	// Print each root and its descendants
	for i, root := range roots {
		printTree(&sb, root, "", i == len(roots)-1, children, orphanSet, unlinkedSet, visited)
	}

	return sb.String(), nil
}

func printTree(sb *strings.Builder, id, prefix string, isLast bool, children map[string][]string, orphanSet, unlinkedSet map[string]bool, visited map[string]bool) {
	// Prevent infinite loops from cycles
	if visited[id] {
		return
	}
	visited[id] = true

	// Determine connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	// Build line with markers
	line := prefix + connector + id
	if orphanSet[id] {
		line += " [ORPHAN]"
	}
	if unlinkedSet[id] {
		line += " [UNLINKED]"
	}
	sb.WriteString(line + "\n")

	// Get and sort children
	childIDs := children[id]
	sort.Strings(childIDs)

	// Calculate new prefix for children
	newPrefix := prefix
	if prefix != "" {
		if isLast {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
	}

	// Print children
	for i, child := range childIDs {
		isChildLast := i == len(childIDs)-1
		printTree(sb, child, newPrefix, isChildLast, children, orphanSet, unlinkedSet, visited)
	}
}
