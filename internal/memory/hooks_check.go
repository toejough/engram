package memory

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckClaudeMDSizeOpts contains options for checking CLAUDE.md size.
type CheckClaudeMDSizeOpts struct {
	ClaudeMDPath string
	MaxLines     int
}

// CheckClaudeMDSize checks if CLAUDE.md exceeds the maximum line count.
// Returns nil if under threshold, error if over.
func CheckClaudeMDSize(opts CheckClaudeMDSizeOpts) error {
	// Read file
	data, err := os.ReadFile(opts.ClaudeMDPath)
	if err != nil {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	// Count lines
	lineCount := bytes.Count(data, []byte("\n"))

	// If file doesn't end with newline, add 1 to count the last line
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lineCount++
	}

	if lineCount > opts.MaxLines {
		return fmt.Errorf("CLAUDE.md exceeds maximum line count: %d lines (max: %d)", lineCount, opts.MaxLines)
	}

	return nil
}

// CheckSkillContractOpts contains options for checking skill contract validation.
type CheckSkillContractOpts struct {
	SkillsDir string
}

// CheckSkillContract validates SKILL.md files in the skills directory.
// Finds files modified in the last 5 minutes (via os.Stat mtime).
// Validates: YAML frontmatter present, has description field, no TODO/FIXME, token count warning if >2500.
func CheckSkillContract(opts CheckSkillContractOpts) error {
	// Find all SKILL.md files
	var skillFiles []string
	err := filepath.Walk(opts.SkillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "SKILL.md" {
			skillFiles = append(skillFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk skills directory: %w", err)
	}

	if len(skillFiles) == 0 {
		return nil // No skills to check
	}

	// Filter to recently modified files (last 5 minutes)
	var recentFiles []string
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	for _, path := range skillFiles {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().After(fiveMinutesAgo) {
			recentFiles = append(recentFiles, path)
		}
	}

	// If no recent files, check all files (fallback)
	if len(recentFiles) == 0 {
		recentFiles = skillFiles
	}

	// Validate each file
	for _, path := range recentFiles {
		if err := validateSkillFile(path); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(filepath.Dir(path)), err)
		}
	}

	return nil
}

// extractDescriptionFromFrontmatter extracts the description field value from YAML frontmatter.
// Handles both single-line and multi-line (|) YAML format.
func extractDescriptionFromFrontmatter(frontmatter string) string {
	lines := strings.Split(frontmatter, "\n")
	var inDescription bool
	var description strings.Builder

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line starts the description field
		if strings.HasPrefix(trimmed, "description:") {
			inDescription = true
			// Check for single-line description
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if rest != "" && rest != "|" {
				// Single-line description
				return rest
			}
			// Multi-line description (|), continue to next lines
			continue
		}

		// If in description, collect indented lines
		if inDescription {
			// Check if line is indented (part of multi-line value)
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				description.WriteString(strings.TrimSpace(line))
				description.WriteString("\n")
			} else if i > 0 && trimmed != "" {
				// Non-indented line means end of multi-line value
				break
			}
		}
	}

	return strings.TrimSpace(description.String())
}

// validateSkillFile validates a single SKILL.md file.
func validateSkillFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("missing YAML frontmatter (must start with ---)")
	}

	// Extract frontmatter
	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return fmt.Errorf("missing YAML frontmatter closing ---")
	}

	frontmatter := parts[0]
	body := parts[1]

	// Check for description field in frontmatter
	if !strings.Contains(frontmatter, "description:") {
		return fmt.Errorf("missing description field in frontmatter")
	}

	// Extract and validate description value
	description := extractDescriptionFromFrontmatter(frontmatter)
	if len(description) < 100 {
		return fmt.Errorf("description too short: %d chars (minimum: 100)", len(description))
	}

	// Warn if description lacks structured sections
	hasStructure := strings.Contains(description, "Core:") ||
		strings.Contains(description, "Triggers:") ||
		strings.Contains(description, "Domains:")
	if !hasStructure {
		fmt.Fprintf(os.Stderr, "Warning: %s description lacks structured sections (Core:, Triggers:, Domains:)\n",
			filepath.Base(filepath.Dir(path)))
	}

	// Check for TODO/FIXME in entire file
	if strings.Contains(content, "TODO") {
		return fmt.Errorf("contains TODO")
	}
	if strings.Contains(content, "FIXME") {
		return fmt.Errorf("contains FIXME")
	}

	// Estimate token count (rough approximation: ~4 characters per token)
	estimatedTokens := len(content) / 4
	if estimatedTokens > 2500 {
		fmt.Fprintf(os.Stderr, "Warning: %s has ~%d tokens (max recommended: 2500)\n",
			filepath.Base(filepath.Dir(path)), estimatedTokens)
	}

	_ = body // Silence unused variable warning

	return nil
}

// CheckEmbeddingMetaOpts contains options for checking embedding metadata.
type CheckEmbeddingMetaOpts struct {
	MemoryRoot string
	Stdin      io.Reader
}

// hookJSON represents the JSON structure passed to PostToolUse hooks.
type hookJSON struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// CheckEmbeddingMetadata validates that the most recently inserted embedding has complete metadata.
// Reads stdin for hook JSON, checks if command contains "projctl memory learn".
// If not a learn command, exits 0 (fast path).
// If learn command: queries most recent embedding, validates metadata fields.
func CheckEmbeddingMetadata(opts CheckEmbeddingMetaOpts) error {
	// Read and parse hook JSON from stdin
	var hook hookJSON
	decoder := json.NewDecoder(opts.Stdin)
	if err := decoder.Decode(&hook); err != nil {
		// If JSON parse fails, treat as non-learn command (fast path)
		return nil
	}

	// Check if this is a memory learn command
	if !strings.Contains(hook.ToolInput.Command, "projctl memory learn") {
		return nil // Fast path: not a learn command
	}

	// Open database
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Query most recently inserted embedding
	var enrichedContent, observationType, concepts string
	var confidence float64
	err = db.QueryRow(`
		SELECT enriched_content, observation_type, concepts, confidence
		FROM embeddings
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&enrichedContent, &observationType, &concepts, &confidence)

	if err == sql.ErrNoRows {
		// No embeddings yet, treat as pass (avoid false positives on first run)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to query embeddings: %w", err)
	}

	// Validate metadata
	var errors []string

	// enriched_content non-empty OR observation_type non-empty
	if enrichedContent == "" && observationType == "" {
		errors = append(errors, "enriched_content and observation_type are both empty")
	}

	// concepts has at least one entry
	if concepts == "" {
		errors = append(errors, "concepts is empty")
	}

	// confidence in [0.0, 1.0]
	if confidence < 0.0 || confidence > 1.0 {
		errors = append(errors, fmt.Sprintf("confidence out of range: %f (must be 0.0-1.0)", confidence))
	}

	if len(errors) > 0 {
		return fmt.Errorf("embedding metadata validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
