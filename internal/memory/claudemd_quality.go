package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CLAUDEMDProposal represents a proposed change to CLAUDE.md.
type CLAUDEMDProposal struct {
	Action         string          // "add", "update", "remove"
	Section        string          // Target section type
	Content        string          // Recommended content
	SourceMemoryID int64           // Memory that triggered this
	Reason         string          // Why this change is proposed
	QualityChecks  map[string]bool // Pass/fail for each gate
	Recommendation Recommendation
}

// CLAUDEMDScore holds the quality scores for a CLAUDE.md file.
type CLAUDEMDScore struct {
	ContextPrecision float64
	Faithfulness     float64
	Currency         float64
	Conciseness      float64
	Coverage         float64
	OverallGrade     string
	OverallScore     float64
	Issues           []string
}

// CLAUDEMDSection represents a parsed section from a CLAUDE.md file.
type CLAUDEMDSection struct {
	Name      string
	Type      string // "commands", "architecture", "gotchas", "code_style", "testing", "other"
	Content   string
	LineCount int
}

// EnforceClaudeMDBudget proposes removals when CLAUDE.md is over the 100-line budget.
func EnforceClaudeMDBudget(claudeMDPath string, db *sql.DB, fs FileSystem) ([]CLAUDEMDProposal, error) {
	const defaultBudget = 100

	data, err := fs.ReadFile(claudeMDPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") {
			return nil, nil
		}

		return nil, fmt.Errorf("EnforceClaudeMDBudget: read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) <= defaultBudget {
		return nil, nil
	}

	overBy := len(lines) - defaultBudget

	rows, err := db.Query(`
		SELECT id, content, effectiveness FROM embeddings
		WHERE promoted = 1
		ORDER BY effectiveness ASC`)
	if err != nil {
		return nil, fmt.Errorf("EnforceClaudeMDBudget: %w", err)
	}

	defer func() { _ = rows.Close() }()

	type entry struct {
		id            int64
		content       string
		effectiveness float64
	}

	var entries []entry

	for rows.Next() {
		var e entry

		err := rows.Scan(&e.id, &e.content, &e.effectiveness)
		if err != nil {
			continue
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("EnforceClaudeMDBudget: scan: %w", err)
	}

	var proposals []CLAUDEMDProposal

	remaining := len(lines)

	for _, e := range entries {
		if remaining <= defaultBudget {
			break
		}

		entryLines := strings.Count(e.content, "\n") + 1
		cat, recText := demotionRecommendation(e.content, e.effectiveness)
		proposals = append(proposals, CLAUDEMDProposal{
			Action:         "remove",
			Content:        e.content,
			SourceMemoryID: e.id,
			Reason:         fmt.Sprintf("Low effectiveness (%.2f), over budget by %d lines", e.effectiveness, overBy),
			QualityChecks:  map[string]bool{"budget_enforcement": true},
			Recommendation: Recommendation{Category: cat, Text: recText},
		})
		remaining -= entryLines
	}

	return proposals, nil
}

// ParseClaudeMDSections parses a CLAUDE.md content string into typed sections.
func ParseClaudeMDSections(content string) ([]CLAUDEMDSection, error) {
	var sections []CLAUDEMDSection
	if content == "" {
		return sections, nil
	}

	lines := strings.Split(content, "\n")

	var (
		currentName  string
		currentLines []string
	)

	flush := func() {
		if currentName == "" {
			return
		}

		body := strings.Join(currentLines, "\n")
		sections = append(sections, CLAUDEMDSection{
			Name:      currentName,
			Type:      classifySectionType(body),
			Content:   body,
			LineCount: countNonEmptyLines(currentLines),
		})
		currentLines = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			flush()

			currentName = strings.TrimSpace(strings.TrimLeft(line, "#"))
		} else if currentName != "" {
			currentLines = append(currentLines, line)
		}
	}

	flush()

	return sections, nil
}

// ProposeClaudeMDChange evaluates a memory for CLAUDE.md promotion.
// Returns nil proposal (no error) when any quality gate fails.
// Returns error only on DB failures or when the memory is not found.
func ProposeClaudeMDChange(db *sql.DB, memoryID int64, _ FileSystem, llm LLMExtractor) (*CLAUDEMDProposal, error) {
	ctx := context.Background()

	var content, quadrant, projectsRetrieved string

	err := db.QueryRow(`SELECT content, quadrant, projects_retrieved FROM embeddings WHERE id = ?`,
		memoryID).Scan(&content, &quadrant, &projectsRetrieved)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("ProposeClaudeMDChange: memory %d not found", memoryID)
	}

	if err != nil {
		return nil, fmt.Errorf("ProposeClaudeMDChange: %w", err)
	}

	checks := make(map[string]bool)

	// Gate 1: Working quadrant
	checks["working_knowledge"] = quadrant == "working"
	if !checks["working_knowledge"] {
		return nil, nil
	}

	// Gate 2: Universal -- surfaced across 3+ distinct projects
	projects := parseProjectsList(projectsRetrieved)

	checks["universal"] = len(projects) >= 3
	if !checks["universal"] {
		return nil, nil
	}

	// Gate 3: Actionable (Haiku check -- skip gracefully if LLM unavailable)
	checks["actionable"] = true

	if llm != nil {
		if caller, ok := llm.(APIMessageCaller); ok {
			if actionable, aErr := checkActionabilityLLM(ctx, caller, content); aErr == nil {
				checks["actionable"] = actionable
			}
			// On LLM error: keep default pass
		}
	}

	if !checks["actionable"] {
		return nil, nil
	}

	// Gate 4: Non-redundant -- no existing promoted memory has high word overlap
	redundant, rErr := checkRedundancyDB(db, memoryID, content)
	if rErr != nil {
		return nil, fmt.Errorf("ProposeClaudeMDChange: redundancy check: %w", rErr)
	}

	checks["non_redundant"] = !redundant
	if !checks["non_redundant"] {
		return nil, nil
	}

	// Gate 5: Right tier (Haiku check -- skip gracefully if LLM unavailable)
	checks["right_tier"] = true

	if llm != nil {
		if caller, ok := llm.(APIMessageCaller); ok {
			if isClaudeMD, tErr := checkTierFitLLM(ctx, caller, content); tErr == nil {
				checks["right_tier"] = isClaudeMD
			}
		}
	}

	if !checks["right_tier"] {
		return nil, nil
	}

	// All gates passed: classify target section
	sectionType := classifySectionType(content)

	if llm != nil {
		if caller, ok := llm.(APIMessageCaller); ok {
			if t, sErr := classifySectionLLM(ctx, caller, content); sErr == nil && t != "" {
				sectionType = t
			}
		}
	}

	rec := Recommendation{
		Category: "claude-md-promotion",
		Text: fmt.Sprintf("Add this entry to the %s section of CLAUDE.md: %s. Evidence: working knowledge across %d projects.",
			sectionType, content, len(projects)),
	}

	return &CLAUDEMDProposal{
		Action:         "add",
		Section:        sectionType,
		Content:        content,
		SourceMemoryID: memoryID,
		Reason: fmt.Sprintf("Memory qualifies for CLAUDE.md: working quadrant, universal across %d projects",
			len(projects)),
		QualityChecks:  checks,
		Recommendation: rec,
	}, nil
}

// ScoreClaudeMD evaluates the quality of a CLAUDE.md file.
func ScoreClaudeMD(claudeMDPath string, db *sql.DB, fs FileSystem, llm LLMExtractor) (*CLAUDEMDScore, error) {
	score := &CLAUDEMDScore{}

	data, err := fs.ReadFile(claudeMDPath)
	if err != nil {
		score.Issues = append(score.Issues, "file not found: "+claudeMDPath)
		score.OverallGrade = gradeFromScore(0)

		return score, nil
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	ctx := context.Background()

	// Context Precision (20%): heuristic for actionable entries
	score.ContextPrecision = scoreContextPrecision(ctx, content, llm)

	// Faithfulness (25%): avg effectiveness from promoted memories
	score.Faithfulness, err = scoreFaithfulness(db)
	if err != nil {
		score.Issues = append(score.Issues, "faithfulness: "+err.Error())
	}

	// Currency (20%): verify commands exist on PATH
	score.Currency = scoreCurrency(content)

	// Conciseness (15%): line count vs budget
	const defaultBudget = 100

	score.Conciseness = scoreConciseness(lines, defaultBudget)

	// Coverage (20%): % of working-universal memories represented
	score.Coverage, err = scoreCoverage(db)
	if err != nil {
		score.Issues = append(score.Issues, "coverage: "+err.Error())
	}

	score.OverallScore = score.ContextPrecision*0.20 +
		score.Faithfulness*0.25 +
		score.Currency*0.20 +
		score.Conciseness*0.15 +
		score.Coverage*0.20

	score.OverallGrade = gradeFromScore(score.OverallScore)

	return score, nil
}

// LLM helpers -- use APIMessageCaller type assertion for direct API access.

func checkActionabilityLLM(ctx context.Context, caller APIMessageCaller, content string) (bool, error) {
	prompt := "Can Claude follow this as a direct instruction? Answer ONLY with valid JSON: {\"actionable\": true} or {\"actionable\": false}\n\nInstruction: " + content

	resp, err := caller.CallAPIWithMessages(ctx, APIMessageParams{
		Messages:  []APIMessage{{Role: "user", Content: prompt}},
		MaxTokens: 50,
	})
	if err != nil {
		return false, err
	}

	var result struct {
		Actionable bool `json:"actionable"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("checkActionabilityLLM: parse: %w", err)
	}

	return result.Actionable, nil
}

// checkRedundancyDB checks whether any promoted memory has high text overlap with the candidate.
func checkRedundancyDB(db *sql.DB, excludeID int64, content string) (bool, error) {
	rows, err := db.Query(`SELECT content FROM embeddings WHERE promoted = 1 AND id != ?`, excludeID)
	if err != nil {
		return false, err
	}

	defer func() { _ = rows.Close() }()

	candidate := wordSet(content)

	for rows.Next() {
		var existing string

		err := rows.Scan(&existing)
		if err != nil {
			continue
		}

		if wordOverlap(candidate, wordSet(existing)) > 0.7 {
			return true, nil
		}
	}

	return false, rows.Err()
}

func checkTierFitLLM(ctx context.Context, caller APIMessageCaller, content string) (bool, error) {
	prompt := "Is this guidance best placed in CLAUDE.md (universal developer rules), or would a hook (deterministic enforcement) or skill (context-specific reference) be more appropriate? Answer ONLY with valid JSON: {\"tier\": \"claude-md\"} or {\"tier\": \"hook\"} or {\"tier\": \"skill\"}\n\nContent: " + content

	resp, err := caller.CallAPIWithMessages(ctx, APIMessageParams{
		Messages:  []APIMessage{{Role: "user", Content: prompt}},
		MaxTokens: 50,
	})
	if err != nil {
		return false, err
	}

	var result struct {
		Tier string `json:"tier"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("checkTierFitLLM: parse: %w", err)
	}

	return result.Tier == "claude-md", nil
}

func classifySectionLLM(ctx context.Context, caller APIMessageCaller, content string) (string, error) {
	prompt := "Classify this content as one of: commands, architecture, gotchas, code_style, testing, other. Answer ONLY with valid JSON: {\"section_type\": \"<type>\"}\n\nContent: " + content

	resp, err := caller.CallAPIWithMessages(ctx, APIMessageParams{
		Messages:  []APIMessage{{Role: "user", Content: prompt}},
		MaxTokens: 50,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		SectionType string `json:"section_type"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("classifySectionLLM: parse: %w", err)
	}

	return result.SectionType, nil
}

// classifySectionType determines section type from content patterns.
func classifySectionType(content string) string {
	// Tables with command/task columns -> "commands"
	if strings.Contains(content, "|") &&
		(strings.Contains(content, "Command") || strings.Contains(content, "command") ||
			strings.Contains(content, "Task") || strings.Contains(content, "task")) {
		return "commands"
	}
	// Directory tree diagrams -> "architecture"
	if strings.Contains(content, "\u251c\u2500\u2500") || strings.Contains(content, "\u2514\u2500\u2500") {
		return "architecture"
	}
	// Bullets with **NEVER**/**ALWAYS** -> "gotchas"
	if strings.Contains(content, "**NEVER**") || strings.Contains(content, "**ALWAYS**") {
		return "gotchas"
	}
	// Test-related commands/patterns -> "testing"
	if strings.Contains(content, "go test") || strings.Contains(content, "targ test") ||
		(strings.Contains(content, "TDD") && strings.Contains(content, "test")) {
		return "testing"
	}
	// Short content (<= 5 non-empty body lines) -> "code_style"
	nonEmpty := countNonEmptyLines(strings.Split(content, "\n"))
	if nonEmpty > 0 && nonEmpty <= 5 {
		return "code_style"
	}

	return "other"
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func countFillerLines(lines []string) int {
	patterns := []string{"TODO", "FIXME", "NOTE:", "see also", "refer to"}
	count := 0

	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, p := range patterns {
			if strings.Contains(lower, strings.ToLower(p)) {
				count++
				break
			}
		}
	}

	return count
}

func countNonEmptyLines(lines []string) int {
	count := 0

	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			count++
		}
	}

	return count
}

// demotionRecommendation determines the appropriate demotion category and text.
func demotionRecommendation(content string, effectiveness float64) (string, string) {
	upper := strings.ToUpper(content)
	if strings.Contains(upper, "NEVER") || strings.Contains(upper, "ALWAYS") {
		return "claude-md-demotion-to-hook",
			fmt.Sprintf("Remove this entry from CLAUDE.md and create a deterministic hook that enforces the rule. Evidence: low effectiveness (%.2f), enforcement pattern detected.", effectiveness)
	}

	if strings.Contains(content, "internal/") || strings.Contains(content, ".go") ||
		strings.Contains(content, "go test") || strings.Contains(content, "go build") {
		return "claude-md-demotion-to-skill",
			fmt.Sprintf("Remove this entry from CLAUDE.md and convert to a skill covering the specific topic. Evidence: narrow/domain-specific content, low effectiveness (%.2f).", effectiveness)
	}

	return "claude-md-demotion-to-memory",
		fmt.Sprintf("Remove this entry from CLAUDE.md. Evidence: low effectiveness (%.2f), insufficient universal value.", effectiveness)
}

// extractCommands finds backtick-quoted command tokens from markdown content.
func extractCommands(content string) []string {
	var cmds []string

	seen := make(map[string]bool)

	for line := range strings.SplitSeq(content, "\n") {
		if !strings.Contains(line, "`") {
			continue
		}

		parts := strings.Split(line, "`")
		for i := 1; i < len(parts); i += 2 {
			fields := strings.Fields(parts[i])
			if len(fields) == 0 {
				continue
			}

			cmd := fields[0]
			if len(cmd) >= 2 && len(cmd) <= 20 && isAlphanumericDash(cmd) && !seen[cmd] {
				seen[cmd] = true
				cmds = append(cmds, cmd)
			}
		}
	}

	return cmds
}

// gradeFromScore converts a weighted composite score to a letter grade.
func gradeFromScore(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func isAlphanumericDash(s string) bool {
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '-' && c != '_' {
			return false
		}
	}

	return true
}

// scoreConciseness scores based on line count versus the 100-line budget.
func scoreConciseness(lines []string, budget int) float64 {
	lineCount := len(lines)
	if lineCount == 0 {
		return 100.0
	}

	if lineCount > budget {
		excess := float64(lineCount-budget) / float64(budget)

		score := 100.0 - excess*50.0
		if score < 0 {
			score = 0
		}

		return score
	}

	fillerCount := countFillerLines(lines)
	fillerRatio := float64(fillerCount) / float64(lineCount)

	return 100.0 - fillerRatio*20.0
}

// scoreContextPrecision evaluates how many lines look like actionable guidance (heuristic).
func scoreContextPrecision(_ context.Context, content string, _ LLMExtractor) float64 {
	lines := strings.Split(content, "\n")
	total := 0
	actionable := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") ||
			strings.HasPrefix(line, "---") {
			continue
		}

		total++

		if len(line) > 0 {
			c := rune(line[0])
			if (c >= 'A' && c <= 'Z') || c == '-' || c == '*' || c == '`' {
				actionable++
			}
		}
	}

	if total == 0 {
		return 50.0
	}

	return float64(actionable) / float64(total) * 100.0
}

// scoreCoverage calculates what percentage of working-universal memories are promoted.
func scoreCoverage(db *sql.DB) (float64, error) {
	var total int

	err := db.QueryRow(`
		SELECT COUNT(*) FROM embeddings
		WHERE quadrant = 'working'
		AND length(projects_retrieved) - length(replace(projects_retrieved, ',', '')) >= 2`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("scoreCoverage: count working: %w", err)
	}

	if total == 0 {
		return 50.0, nil // neutral when no working memories
	}

	var promoted int

	err = db.QueryRow(`
		SELECT COUNT(*) FROM embeddings
		WHERE quadrant = 'working' AND promoted = 1
		AND length(projects_retrieved) - length(replace(projects_retrieved, ',', '')) >= 2`).Scan(&promoted)
	if err != nil {
		return 0, fmt.Errorf("scoreCoverage: count promoted: %w", err)
	}

	return float64(promoted) / float64(total) * 100.0, nil
}

// scoreCurrency checks whether backtick-quoted commands in the file exist on PATH.
func scoreCurrency(content string) float64 {
	commands := extractCommands(content)
	if len(commands) == 0 {
		return 75.0 // neutral when nothing to verify
	}

	valid := 0

	for _, cmd := range commands {
		if commandExists(cmd) {
			valid++
		}
	}

	return float64(valid) / float64(len(commands)) * 100.0
}

// scoreFaithfulness returns the average effectiveness score of promoted memories (0-100).
func scoreFaithfulness(db *sql.DB) (float64, error) {
	var avg sql.NullFloat64

	err := db.QueryRow(`
		SELECT AVG(effectiveness) FROM embeddings
		WHERE promoted = 1 AND effectiveness > 0`).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("scoreFaithfulness: %w", err)
	}

	if !avg.Valid {
		return 50.0, nil // no data = neutral
	}

	v := avg.Float64 * 100.0
	if v > 100 {
		v = 100
	}

	return v, nil
}

// wordOverlap returns the Jaccard similarity between two word sets.
func wordOverlap(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}

	intersection := 0

	for w := range a {
		if b[w] {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// wordSet converts text to a set of lowercase words (3+ chars, stripped of punctuation).
func wordSet(text string) map[string]bool {
	words := make(map[string]bool)

	for w := range strings.FieldsSeq(strings.ToLower(text)) {
		w = strings.Trim(w, ".,!?;:'\"()-[]`")
		if len(w) >= 3 {
			words[w] = true
		}
	}

	return words
}
