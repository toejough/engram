package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanClaudeMD scans CLAUDE.md for maintenance opportunities and returns proposals.
// It detects:
// - Redundant entries (similarity > threshold) → consolidate
// - Overly broad entries (token count > threshold, multiple topics) → split
// - Too-specific entries (domain-specific, not universal) → demote to skill
// - Stale entries → prune
func ScanClaudeMD(fs FileSystem, claudeMDPath string, similarityThreshold float64) ([]MaintenanceProposal, error) {
	content, err := fs.ReadFile(claudeMDPath)
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

	// Parse learning entries
	type learningEntry struct {
		line    string
		content string
	}
	var entries []learningEntry
	for _, line := range promoted {
		trimmed := strings.TrimSpace(line)
		entry := strings.TrimPrefix(trimmed, "- ")
		// Strip timestamp prefix if present (e.g., "2026-02-08 21:40: ")
		entry = stripTimestampPrefix(entry)
		if entry != "" {
			entries = append(entries, learningEntry{line: line, content: entry})
		}
	}

	if len(entries) == 0 {
		return nil, nil
	}

	var proposals []MaintenanceProposal

	// Initialize ONNX for similarity checks
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}
	if err := initializeONNXRuntime(modelDir); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Generate embeddings for all entries
	type embeddedEntry struct {
		learningEntry
		embedding []float32
	}
	var embeddedEntries []embeddedEntry
	for _, e := range entries {
		emb, _, _, err := generateEmbeddingONNX("passage: "+e.content, modelPath)
		if err != nil {
			continue
		}
		embeddedEntries = append(embeddedEntries, embeddedEntry{learningEntry: e, embedding: emb})
	}

	// 1. Detect redundant entries (similarity > threshold)
	seen := make(map[int]bool)
	for i := 0; i < len(embeddedEntries); i++ {
		if seen[i] {
			continue
		}
		for j := i + 1; j < len(embeddedEntries); j++ {
			if seen[j] {
				continue
			}
			sim := cosineSimilarity(embeddedEntries[i].embedding, embeddedEntries[j].embedding)
			if sim > similarityThreshold {
				// Consolidate: merge entries
				merged := mergeEntries(embeddedEntries[i].content, embeddedEntries[j].content)
				proposal := MaintenanceProposal{
					Tier:    "claude-md",
					Action:  "consolidate",
					Target:  embeddedEntries[i].content + "|" + embeddedEntries[j].content,
					Reason:  fmt.Sprintf("redundant entries with similarity %.2f", sim),
					Preview: merged,
				}
				proposals = append(proposals, proposal)
				seen[j] = true
			}
		}
	}

	// 2. Detect overly broad entries (token count > 100, likely multiple topics)
	for _, e := range entries {
		tokenCount := len(strings.Fields(e.content))
		if tokenCount > 100 {
			// Propose split
			splitParts := splitLongEntry(e.content)
			proposal := MaintenanceProposal{
				Tier:    "claude-md",
				Action:  "split",
				Target:  e.content,
				Reason:  fmt.Sprintf("entry covers multiple topics (%d tokens)", tokenCount),
				Preview: strings.Join(splitParts, "|"),
			}
			proposals = append(proposals, proposal)
		}
	}

	// 3. Detect too-specific entries (domain-specific, not universal)
	for _, e := range entries {
		isNarrow, reason := isNarrowByKeywords(e.content)
		if isNarrow {
			proposal := MaintenanceProposal{
				Tier:   "claude-md",
				Action: "demote",
				Target: e.content,
				Reason: reason,
			}
			proposals = append(proposals, proposal)
		}
	}

	return proposals, nil
}

// ApplyClaudeMDProposal applies a maintenance proposal to CLAUDE.md.
// Supported actions:
// - prune: remove entry from Promoted Learnings
// - consolidate: merge similar entries into one
// - split: break long entry into multiple entries
// - demote: remove entry from CLAUDE.md (caller should create skill)
func ApplyClaudeMDProposal(fs FileSystem, claudeMDPath string, proposal MaintenanceProposal) error {
	switch proposal.Action {
	case "prune":
		return RemoveFromClaudeMD(fs, claudeMDPath, []string{proposal.Target})

	case "consolidate":
		// Target format: "entry1|entry2"
		parts := strings.Split(proposal.Target, "|")
		if len(parts) != 2 {
			return fmt.Errorf("consolidate target must be 'entry1|entry2', got: %s", proposal.Target)
		}
		// Remove both entries
		if err := RemoveFromClaudeMD(fs, claudeMDPath, parts); err != nil {
			return err
		}
		// Add merged entry
		return appendToClaudeMDWithFS(fs, claudeMDPath, []string{proposal.Preview})

	case "split":
		// Preview format: "part1|part2|part3"
		parts := strings.Split(proposal.Preview, "|")
		if len(parts) < 2 {
			return fmt.Errorf("split preview must contain at least 2 parts, got: %s", proposal.Preview)
		}
		// Remove original entry
		if err := RemoveFromClaudeMD(fs, claudeMDPath, []string{proposal.Target}); err != nil {
			return err
		}
		// Add split parts
		return appendToClaudeMDWithFS(fs, claudeMDPath, parts)

	case "demote":
		// Just remove from CLAUDE.md (caller handles skill creation)
		return RemoveFromClaudeMD(fs, claudeMDPath, []string{proposal.Target})

	default:
		return fmt.Errorf("unknown action: %s", proposal.Action)
	}
}

// stripTimestampPrefix removes the timestamp prefix from a learning entry.
// Format: "2026-02-08 21:40: content" -> "content"
func stripTimestampPrefix(entry string) string {
	// Check for timestamp prefix: "YYYY-MM-DD HH:MM: "
	if len(entry) < 19 {
		return entry
	}
	// Quick check for pattern
	if entry[4] == '-' && entry[7] == '-' && entry[10] == ' ' && entry[13] == ':' && entry[16] == ':' {
		// Check if followed by space
		if len(entry) > 19 && entry[19] == ' ' {
			return strings.TrimSpace(entry[20:])
		}
	}
	return entry
}

// mergeEntries merges two similar entries into a consolidated version.
func mergeEntries(entry1, entry2 string) string {
	// Simple heuristic: use the longer entry as base, incorporate unique words from shorter
	// For now, just pick the longer one (can enhance with LLM later)
	if len(entry1) > len(entry2) {
		return entry1
	}
	return entry2
}

// splitLongEntry splits a long entry into multiple parts based on sentence boundaries.
func splitLongEntry(entry string) []string {
	// Split on sentence boundaries (. followed by space or end)
	sentences := strings.Split(entry, ". ")

	// If we have multiple sentences, return each as a separate part
	if len(sentences) > 1 {
		var parts []string
		for _, s := range sentences {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				// Add period back if it was removed
				if !strings.HasSuffix(trimmed, ".") && !strings.HasSuffix(trimmed, "?") && !strings.HasSuffix(trimmed, "!") {
					trimmed += "."
				}
				parts = append(parts, trimmed)
			}
		}
		return parts
	}

	// If no sentence boundaries, split on conjunction words
	conjunctions := []string{". ", "; ", ", and ", ", but ", ": "}
	for _, conj := range conjunctions {
		if strings.Contains(entry, conj) {
			parts := strings.Split(entry, conj)
			var cleaned []string
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					cleaned = append(cleaned, trimmed)
				}
			}
			if len(cleaned) > 1 {
				return cleaned
			}
		}
	}

	// Fallback: return original as single part
	return []string{entry}
}

// appendToClaudeMDWithFS is a variant of appendToClaudeMD that uses FileSystem interface.
func appendToClaudeMDWithFS(fs FileSystem, claudeMDPath string, learnings []string) error {
	// Build the new learning lines
	var newLines strings.Builder
	for _, learning := range learnings {
		newLines.WriteString("- " + learning + "\n")
	}

	// Read existing content (empty if file doesn't exist)
	existing, err := fs.ReadFile(claudeMDPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	content := string(existing)
	const sectionHeader = "## Promoted Learnings"

	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		// No Promoted Learnings section - add it
		if content != "" && !strings.HasSuffix(content, "\n\n") {
			content += "\n\n"
		}
		content += sectionHeader + "\n\n" + newLines.String()
	} else {
		// Find next section or end of file
		afterHeader := idx + len(sectionHeader)
		nextSection := strings.Index(content[afterHeader:], "\n## ")

		var insertPos int
		if nextSection == -1 {
			// No next section - append at end
			insertPos = len(content)
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
		} else {
			// Insert before next section
			insertPos = afterHeader + nextSection
		}

		content = content[:insertPos] + newLines.String() + content[insertPos:]
	}

	return fs.WriteFile(claudeMDPath, []byte(content), 0644)
}
