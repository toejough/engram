package memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ScanForRefinements scans memory tiers for content refinement opportunities.
// Returns proposals for:
// - Rewriting flagged entries (embeddings tier)
// - Adding rationale to entries with principle but no rationale (embeddings tier)
// - Adding rationale to imperative CLAUDE.md entries
// - Extracting examples from CLAUDE.md entries with code blocks
//
// If extractor is nil, returns empty (no proposals generated).
func ScanForRefinements(db *sql.DB, claudeMDPath string, extractor LLMExtractor) ([]MaintenanceProposal, error) {
	// If no extractor, skip refinement scanning
	if extractor == nil {
		return []MaintenanceProposal{}, nil
	}

	var proposals []MaintenanceProposal

	// Scan embeddings tier for flagged_for_rewrite=1 entries
	rewriteProposals, err := scanFlaggedForRewrite(db, extractor)
	if err != nil {
		return nil, fmt.Errorf("failed to scan flagged entries: %w", err)
	}

	proposals = append(proposals, rewriteProposals...)

	// Scan embeddings tier for entries with principle but no rationale
	rationaleProposals, err := scanMissingRationale(db, extractor)
	if err != nil {
		return nil, fmt.Errorf("failed to scan missing rationale: %w", err)
	}

	proposals = append(proposals, rationaleProposals...)

	// Scan CLAUDE.md for imperative entries without explanation
	claudeMDProposals, err := scanClaudeMDForRationale(claudeMDPath, extractor)
	if err != nil {
		return nil, fmt.Errorf("failed to scan CLAUDE.md: %w", err)
	}

	proposals = append(proposals, claudeMDProposals...)

	// Scan CLAUDE.md for entries with code blocks
	codeBlockProposals, err := scanClaudeMDCodeBlocks(claudeMDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan CLAUDE.md code blocks: %w", err)
	}

	proposals = append(proposals, codeBlockProposals...)

	return proposals, nil
}

// WriteFile is a helper for tests to write files (thin wrapper around os.WriteFile).
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// extractPrinciple extracts the principle part from an entry with examples.
func extractPrinciple(entry string) string {
	// Try to extract principle before ". Example:", ". E.g.", etc.
	markers := []string{". Example:", ". E.g.", ". For example,", ". Usage:"}
	for _, marker := range markers {
		if before, _, ok := strings.Cut(entry, marker); ok {
			return strings.TrimSpace(before)
		}
	}

	// If no marker, try to remove code blocks
	if strings.Contains(entry, "`") {
		// Remove everything in backticks
		result := entry
		for strings.Contains(result, "`") {
			start := strings.Index(result, "`")

			end := strings.Index(result[start+1:], "`")
			if end == -1 {
				break
			}

			end += start + 1
			result = result[:start] + result[end+1:]
		}

		return strings.TrimSpace(result)
	}

	// Otherwise return as-is
	return entry
}

// hasCodeBlockOrPath checks if entry contains code blocks or file paths.
func hasCodeBlockOrPath(entry string) bool {
	// Check for backticks (code blocks)
	if strings.Contains(entry, "`") {
		return true
	}

	// Check for file paths (contains / or .go, .py, .ts, etc.)
	if strings.Contains(entry, "/") {
		return true
	}

	extensions := []string{".go", ".py", ".ts", ".js", ".yaml", ".json", ".md"}
	for _, ext := range extensions {
		if strings.Contains(entry, ext) {
			return true
		}
	}

	return false
}

// isImperativeWithoutRationale checks if entry is imperative without explanation.
func isImperativeWithoutRationale(entry string) bool {
	// Imperative entries typically start with verbs: "Always", "Never", "Use", "Avoid", etc.
	// And lack explanation markers: "because", "to", "-", ":", etc.
	lower := strings.ToLower(entry)

	// Check for imperative starters
	imperativeStarters := []string{
		"always ", "never ", "use ", "avoid ", "prefer ", "ensure ",
		"check ", "verify ", "validate ", "do not ", "don't ",
	}
	hasImperative := false

	for _, starter := range imperativeStarters {
		if strings.HasPrefix(lower, starter) {
			hasImperative = true
			break
		}
	}

	if !hasImperative {
		return false
	}

	// Check for explanation markers
	explanationMarkers := []string{
		" because ", " to ", " - ", " since ", " as ",
		" for ", " when ", " so that ",
	}
	for _, marker := range explanationMarkers {
		if strings.Contains(lower, marker) {
			return false // Has explanation, not a candidate
		}
	}

	return true // Imperative without explanation
}

// scanClaudeMDCodeBlocks scans CLAUDE.md for entries mixing rule + code blocks/paths.
func scanClaudeMDCodeBlocks(claudeMDPath string) ([]MaintenanceProposal, error) {
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	if len(content) == 0 {
		return nil, nil
	}

	sections := ParseCLAUDEMD(string(content))

	promoted, ok := sections["Promoted Learnings"]
	if !ok || len(promoted) == 0 {
		return nil, nil
	}

	var proposals []MaintenanceProposal

	for _, line := range promoted {
		trimmed := strings.TrimSpace(line)
		entry := strings.TrimPrefix(trimmed, "- ")
		// Strip timestamp prefix
		entry = stripTimestampPrefix(entry)

		if entry == "" {
			continue
		}

		// Check if entry contains code blocks (backticks) or file paths
		if hasCodeBlockOrPath(entry) {
			// Extract the principle part (before ". Example:" or similar)
			principleOnly := extractPrinciple(entry)

			proposals = append(proposals, MaintenanceProposal{
				Tier:    "claude-md",
				Action:  "extract-examples",
				Target:  entry,
				Reason:  "contains code block or specific path - extract to keep principle clean",
				Preview: principleOnly,
			})
		}
	}

	return proposals, nil
}

// scanClaudeMDForRationale scans CLAUDE.md for imperative entries without explanation.
func scanClaudeMDForRationale(claudeMDPath string, extractor LLMExtractor) ([]MaintenanceProposal, error) {
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	if len(content) == 0 {
		return nil, nil
	}

	sections := ParseCLAUDEMD(string(content))

	promoted, ok := sections["Promoted Learnings"]
	if !ok || len(promoted) == 0 {
		return nil, nil
	}

	var proposals []MaintenanceProposal

	ctx := context.Background()

	for _, line := range promoted {
		trimmed := strings.TrimSpace(line)
		entry := strings.TrimPrefix(trimmed, "- ")
		// Strip timestamp prefix
		entry = stripTimestampPrefix(entry)

		if entry == "" {
			continue
		}

		// Check if entry is imperative without explanation (lacks "because", "to", "-", etc.)
		if isImperativeWithoutRationale(entry) {
			// Generate enriched version with rationale
			enriched, err := extractor.AddRationale(ctx, entry)
			if err != nil {
				continue // Skip on LLM error
			}

			proposals = append(proposals, MaintenanceProposal{
				Tier:    "claude-md",
				Action:  "add-rationale",
				Target:  entry,
				Reason:  "imperative rule without explanation",
				Preview: enriched,
			})
		}
	}

	return proposals, nil
}

// scanFlaggedForRewrite scans embeddings tier for entries flagged for rewriting.
func scanFlaggedForRewrite(db *sql.DB, extractor LLMExtractor) ([]MaintenanceProposal, error) {
	rows, err := db.Query(`
		SELECT id, content
		FROM embeddings
		WHERE flagged_for_rewrite = 1
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	var proposals []MaintenanceProposal

	ctx := context.Background()

	for rows.Next() {
		var (
			id      int64
			content string
		)

		if err := rows.Scan(&id, &content); err != nil {
			continue
		}

		// Generate refined version via LLM
		refined, err := extractor.Rewrite(ctx, content)
		if err != nil {
			continue // Skip on LLM error
		}

		proposals = append(proposals, MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "rewrite",
			Target:  strconv.FormatInt(id, 10),
			Reason:  "flagged for clarity/specificity improvements",
			Preview: refined,
		})
	}

	return proposals, nil
}

// scanMissingRationale scans embeddings tier for entries with principle but no rationale.
func scanMissingRationale(db *sql.DB, extractor LLMExtractor) ([]MaintenanceProposal, error) {
	rows, err := db.Query(`
		SELECT id, content, principle
		FROM embeddings
		WHERE principle != ''
		  AND (rationale IS NULL OR rationale = '')
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	var proposals []MaintenanceProposal

	ctx := context.Background()

	for rows.Next() {
		var (
			id                 int64
			content, principle string
		)

		if err := rows.Scan(&id, &content, &principle); err != nil {
			continue
		}

		// Generate enriched version with rationale
		enriched, err := extractor.AddRationale(ctx, principle)
		if err != nil {
			continue // Skip on LLM error
		}

		proposals = append(proposals, MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "add-rationale",
			Target:  strconv.FormatInt(id, 10),
			Reason:  "principle without explanation of why",
			Preview: enriched,
		})
	}

	return proposals, nil
}
