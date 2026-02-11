// Package corrections provides structured JSONL logging for tracking corrections in the learning loop.
package corrections

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// FileSystem provides file system operations for corrections management.
type FileSystem interface {
	AppendFile(path string, data []byte) error
	ReadFile(path string) ([]byte, error)
	FileExists(path string) bool
	MkdirAll(path string) error
}

// RealFS implements FileSystem using the real file system.
type RealFS struct{}

// AppendFile appends data to a file, creating it if needed.
func (RealFS) AppendFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(data)
	return err
}

// ReadFile reads a file.
func (RealFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// FileExists checks if a file exists.
func (RealFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MkdirAll creates a directory and all necessary parents.
func (RealFS) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// Entry represents a single correction log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Context   string `json:"context"`
	SessionID string `json:"session_id,omitempty"`
}

// LogOpts holds optional fields for a correction entry.
type LogOpts struct {
	SessionID string
}

// Log appends a correction entry to the project-specific corrections.jsonl file.
func Log(dir string, message string, context string, opts LogOpts, now func() time.Time, fs FileSystem) error {
	path := filepath.Join(dir, "corrections.jsonl")
	return writeEntry(path, message, context, opts, now, fs)
}

// LogGlobal appends a correction entry to the global ~/.claude/corrections.jsonl file.
func LogGlobal(message string, context string, opts LogOpts, homeDir string, now func() time.Time, fs FileSystem) error {
	claudeDir := filepath.Join(homeDir, ".claude")
	if err := fs.MkdirAll(claudeDir); err != nil {
		return err
	}
	path := filepath.Join(claudeDir, "corrections.jsonl")
	return writeEntry(path, message, context, opts, now, fs)
}

// Read reads correction entries from the project-specific corrections.jsonl file.
func Read(dir string, fs FileSystem) ([]Entry, error) {
	path := filepath.Join(dir, "corrections.jsonl")
	return readEntries(path, fs)
}

// ReadGlobal reads correction entries from the global ~/.claude/corrections.jsonl file.
func ReadGlobal(homeDir string, fs FileSystem) ([]Entry, error) {
	path := filepath.Join(homeDir, ".claude", "corrections.jsonl")
	return readEntries(path, fs)
}

func writeEntry(path string, message string, context string, opts LogOpts, now func() time.Time, fs FileSystem) error {
	if now == nil {
		now = time.Now
	}
	entry := Entry{
		Timestamp: now().UTC().Format(time.RFC3339),
		Message:   message,
		Context:   context,
		SessionID: opts.SessionID,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Append data with newline
	line := append(data, '\n')
	return fs.AppendFile(path, line)
}

func readEntries(path string, fs FileSystem) ([]Entry, error) {
	if !fs.FileExists(path) {
		return []Entry{}, nil
	}

	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// Pattern represents a recurring correction pattern detected by analysis.
type Pattern struct {
	Message  string   // Representative message for the pattern
	Count    int      // Number of occurrences
	Proposal string   // Proposed CLAUDE.md addition
	Examples []Entry  // Sample entries that match this pattern
}

// AnalyzeOpts holds options for analyzing correction patterns.
type AnalyzeOpts struct {
	MinOccurrences int // Minimum occurrences to report a pattern (default: 2)
}

// Analyze detects patterns in corrections using fuzzy matching.
// Returns patterns sorted by count (descending).
func Analyze(dir string, opts AnalyzeOpts, fs FileSystem) ([]Pattern, error) {
	// Set default MinOccurrences to 2
	minOccurrences := opts.MinOccurrences
	if minOccurrences == 0 {
		minOccurrences = 2
	}

	// Read all corrections
	entries, err := Read(dir, fs)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []Pattern{}, nil
	}

	// Group entries by keywords
	groups := groupByKeywords(entries)

	// Build patterns from groups
	var patterns []Pattern
	for message, groupEntries := range groups {
		if len(groupEntries) >= minOccurrences {
			patterns = append(patterns, Pattern{
				Message:  message,
				Count:    len(groupEntries),
				Proposal: makeProposal(message),
				Examples: groupEntries,
			})
		}
	}

	// Sort by count descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Count > patterns[i].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns, nil
}

// groupByKeywords groups entries by extracted keywords, using fuzzy matching
func groupByKeywords(entries []Entry) map[string][]Entry {
	groups := make(map[string][]Entry)

	for _, entry := range entries {
		keywords := extractKeywords(entry.Message)

		// Find existing group with matching keywords
		matched := false
		for groupMsg, groupEntries := range groups {
			groupKeywords := extractKeywords(groupMsg)
			if keywordsMatch(keywords, groupKeywords) {
				groups[groupMsg] = append(groupEntries, entry)
				matched = true
				break
			}
		}

		// Create new group if no match
		if !matched {
			groups[entry.Message] = append(groups[entry.Message], entry)
		}
	}

	return groups
}

// extractKeywords extracts significant words from a message, filtering stop words
func extractKeywords(message string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "up": true, "about": true,
		"into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "under": true, "again": true,
		"further": true, "then": true, "once": true, "here": true, "there": true,
		"when": true, "where": true, "why": true, "how": true, "all": true, "both": true,
		"each": true, "few": true, "more": true, "most": true, "other": true,
		"some": true, "such": true, "only": true, "own": true, "same": true,
		"so": true, "than": true, "too": true, "very": true, "can": true, "will": true,
		"just": true, "should": true, "now": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "doing": true,
		"that": true, "this": true, "these": true, "those": true, "if": true,
		"using": true, "use": true, "used": true,
	}

	// Split on non-word characters and filter
	var keywords []string
	word := ""
	for _, ch := range message + " " {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			word += string(ch)
		} else if word != "" {
			lower := toLower(word)
			if !stopWords[lower] && len(lower) > 1 {
				keywords = append(keywords, lower)
			}
			word = ""
		}
	}

	return keywords
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := ""
	for _, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			result += string(ch - 'A' + 'a')
		} else {
			result += string(ch)
		}
	}
	return result
}

// keywordsMatch checks if two keyword sets have significant overlap
func keywordsMatch(keywords1, keywords2 []string) bool {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return false
	}

	// Count common keywords and fuzzy matches (word stems)
	common := 0
	for _, k1 := range keywords1 {
		for _, k2 := range keywords2 {
			if k1 == k2 {
				common++
				break
			} else if wordsAreSimilar(k1, k2) {
				common++
				break
			}
		}
	}

	// Require at least 2 keywords in common for any match
	if common < 2 {
		return false
	}

	// Calculate overlap percentage for both sets
	overlap1 := float64(common) / float64(len(keywords1))
	overlap2 := float64(common) / float64(len(keywords2))

	// Use adaptive matching rules:
	// 1. If at least one set has 4+ keywords and smaller set is 60%+ matched with 2+ common, match
	//    (handles "amend pushed commits" with extra context words)
	// 2. For small keyword sets (both <=3), require both to be 67%+ matched
	//    (prevents "unique correction one" vs "unique correction two" from matching)
	// 3. For larger sets, require both to be 60%+ matched

	// Find the overlap for the smaller set
	smallerSetOverlap := overlap1
	if len(keywords2) < len(keywords1) {
		smallerSetOverlap = overlap2
	}

	// Special rule: if at least one set is large (>=4) and smaller set is well matched
	if common >= 2 && (len(keywords1) >= 4 || len(keywords2) >= 4) && smallerSetOverlap >= 0.6 {
		return true
	}

	// For smaller keyword sets (both <=3), require stricter matching on both
	threshold := 0.67
	if len(keywords1) >= 4 || len(keywords2) >= 4 {
		threshold = 0.6
	}

	return overlap1 >= threshold && overlap2 >= threshold
}

// wordsAreSimilar checks if two words are variations of each other (e.g., amend/amending)
func wordsAreSimilar(w1, w2 string) bool {
	// Check if one is a prefix of the other (minimum 4 chars)
	if len(w1) >= 4 && len(w2) >= 4 {
		minLen := len(w1)
		if len(w2) < minLen {
			minLen = len(w2)
		}

		// Compare first 4-5 characters
		compareLen := 4
		if minLen > 5 {
			compareLen = 5
		}
		if minLen < compareLen {
			compareLen = minLen
		}

		return w1[:compareLen] == w2[:compareLen]
	}
	return false
}

// makeProposal generates a markdown-formatted proposal for CLAUDE.md
func makeProposal(message string) string {
	return "**" + message + "**"
}

// AnalyzeGlobal detects patterns in global corrections (~/.claude/corrections.jsonl).
func AnalyzeGlobal(homeDir string, opts AnalyzeOpts, fs FileSystem) ([]Pattern, error) {
	// Set default MinOccurrences to 2
	minOccurrences := opts.MinOccurrences
	if minOccurrences == 0 {
		minOccurrences = 2
	}

	// Read all corrections
	entries, err := ReadGlobal(homeDir, fs)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []Pattern{}, nil
	}

	// Group entries by keywords
	groups := groupByKeywords(entries)

	// Build patterns from groups
	var patterns []Pattern
	for message, groupEntries := range groups {
		if len(groupEntries) >= minOccurrences {
			patterns = append(patterns, Pattern{
				Message:  message,
				Count:    len(groupEntries),
				Proposal: makeProposal(message),
				Examples: groupEntries,
			})
		}
	}

	// Sort by count descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Count > patterns[i].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns, nil
}
