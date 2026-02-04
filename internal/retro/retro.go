// Package retro provides functionality for parsing retrospective files.
package retro

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Recommendation represents an extracted recommendation from a retrospective.
type Recommendation struct {
	ID        string
	Title     string
	Priority  string // "High", "Medium", "Low"
	Action    string
	Rationale string
}

// OpenQuestion represents an extracted open question from a retrospective.
type OpenQuestion struct {
	ID              string
	Title           string
	Context         string
	Options         []string
	DecisionNeeded  string
}

// ExtractRecommendations parses retro.md or retrospective.md for process improvement recommendations.
func ExtractRecommendations(dir string) ([]Recommendation, error) {
	content, err := readRetroFile(dir)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, nil
	}

	return parseRecommendations(content), nil
}

// readRetroFile reads retro.md or retrospective.md, whichever exists.
func readRetroFile(dir string) (string, error) {
	names := []string{"retro.md", "retrospective.md"}
	for _, name := range names {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", nil // Neither file exists
}

// ExtractOpenQuestions parses retro.md or retrospective.md for open questions needing decisions.
func ExtractOpenQuestions(dir string) ([]OpenQuestion, error) {
	content, err := readRetroFile(dir)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, nil
	}

	return parseOpenQuestions(content), nil
}

// FilterByPriority filters recommendations to include items at or above the given priority.
func FilterByPriority(items []Recommendation, minPriority string) []Recommendation {
	priorityOrder := map[string]int{
		"High":   3,
		"Medium": 2,
		"Low":    1,
	}

	threshold := priorityOrder[minPriority]
	if threshold == 0 {
		threshold = 1 // Default to including all
	}

	var filtered []Recommendation
	for _, item := range items {
		if priorityOrder[item.Priority] >= threshold {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func parseRecommendations(content string) []Recommendation {
	var recommendations []Recommendation
	var currentPriority string
	var currentRec *Recommendation
	var currentField string

	scanner := bufio.NewScanner(strings.NewReader(content))
	inRecsSection := false
	inPrioritySection := false

	// Patterns
	recsHeaderRe := regexp.MustCompile(`^## Process Improvement Recommendations`)
	priorityRe := regexp.MustCompile(`^### (High|Medium|Low) Priority`)
	recHeaderRe := regexp.MustCompile(`^#### (R\d+): (.+)$`)
	fieldRe := regexp.MustCompile(`^\*\*(\w+):\*\*\s*(.*)$`)
	nextSectionRe := regexp.MustCompile(`^## `)

	for scanner.Scan() {
		line := scanner.Text()

		// Check for recommendations section start
		if recsHeaderRe.MatchString(line) {
			inRecsSection = true
			continue
		}

		// Check for next section (exit recommendations)
		if inRecsSection && nextSectionRe.MatchString(line) && !priorityRe.MatchString(line) {
			// Save current recommendation
			if currentRec != nil {
				recommendations = append(recommendations, *currentRec)
				currentRec = nil
			}
			break
		}

		if !inRecsSection {
			continue
		}

		// Check for priority subsection
		if matches := priorityRe.FindStringSubmatch(line); matches != nil {
			// Save previous recommendation
			if currentRec != nil {
				recommendations = append(recommendations, *currentRec)
				currentRec = nil
			}
			currentPriority = matches[1]
			inPrioritySection = true
			continue
		}

		if !inPrioritySection {
			continue
		}

		// Check for recommendation header
		if matches := recHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous recommendation
			if currentRec != nil {
				recommendations = append(recommendations, *currentRec)
			}
			currentRec = &Recommendation{
				ID:       matches[1],
				Title:    matches[2],
				Priority: currentPriority,
			}
			currentField = ""
			continue
		}

		if currentRec == nil {
			continue
		}

		// Check for field
		if matches := fieldRe.FindStringSubmatch(line); matches != nil {
			currentField = strings.ToLower(matches[1])
			value := matches[2]
			switch currentField {
			case "action":
				currentRec.Action = value
			case "rationale":
				currentRec.Rationale = value
			}
			continue
		}

		// Continue accumulating field content
		if currentField != "" && !strings.HasPrefix(line, "---") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				switch currentField {
				case "action":
					currentRec.Action += " " + trimmed
				case "rationale":
					currentRec.Rationale += " " + trimmed
				}
			}
		}
	}

	// Save final recommendation
	if currentRec != nil {
		recommendations = append(recommendations, *currentRec)
	}

	return recommendations
}

func parseOpenQuestions(content string) []OpenQuestion {
	var questions []OpenQuestion
	var currentQ *OpenQuestion
	var currentField string

	scanner := bufio.NewScanner(strings.NewReader(content))
	inQuestionsSection := false

	// Patterns
	questionsHeaderRe := regexp.MustCompile(`^## Open Questions`)
	questionRe := regexp.MustCompile(`^### (Q\d+): (.+)$`)
	fieldRe := regexp.MustCompile(`^\*\*(\w+[^*]*):\*\*\s*(.*)$`)
	nextSectionRe := regexp.MustCompile(`^## `)

	for scanner.Scan() {
		line := scanner.Text()

		// Check for questions section start
		if questionsHeaderRe.MatchString(line) {
			inQuestionsSection = true
			continue
		}

		// Check for next section
		if inQuestionsSection && nextSectionRe.MatchString(line) {
			if currentQ != nil {
				questions = append(questions, *currentQ)
				currentQ = nil
			}
			break
		}

		if !inQuestionsSection {
			continue
		}

		// Check for question header
		if matches := questionRe.FindStringSubmatch(line); matches != nil {
			if currentQ != nil {
				questions = append(questions, *currentQ)
			}
			currentQ = &OpenQuestion{
				ID:    matches[1],
				Title: matches[2],
			}
			currentField = ""
			continue
		}

		if currentQ == nil {
			continue
		}

		// Check for field
		if matches := fieldRe.FindStringSubmatch(line); matches != nil {
			currentField = strings.ToLower(matches[1])
			value := matches[2]
			switch currentField {
			case "context":
				currentQ.Context = value
			case "decision needed before":
				currentQ.DecisionNeeded = value
			}
			continue
		}

		// Accumulate context content
		if currentField == "context" && !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "**") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				currentQ.Context += " " + trimmed
			}
		}
	}

	// Save final question
	if currentQ != nil {
		questions = append(questions, *currentQ)
	}

	return questions
}
