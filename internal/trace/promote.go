package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/toejough/projctl/internal/config"
)

// Promotion represents a single trace promotion that was made.
type Promotion struct {
	File     string `json:"file"`      // relative path to file
	Line     int    `json:"line"`      // line number of the change
	OldTrace string `json:"old_trace"` // original TASK-NNN
	NewTrace string `json:"new_trace"` // new trace target(s)
}

// SkippedPromotion represents a trace that couldn't be promoted.
type SkippedPromotion struct {
	File   string `json:"file"`   // relative path to file
	Line   int    `json:"line"`   // line number
	TaskID string `json:"task_id"` // the TASK-NNN that couldn't be promoted
	Reason string `json:"reason"` // why it was skipped
}

// PromoteResult holds the results of a promote operation.
type PromoteResult struct {
	Promotions []Promotion        `json:"promotions"`
	Skipped    []SkippedPromotion `json:"skipped"`
}

// Promote finds test files with TASK traces and replaces them with permanent IDs.
// It looks up each TASK's "Traces to:" field in tasks.md and replaces the
// "// traces: TASK-NNN" comment with the permanent target ID(s).
// If dryRun is true, reports what would change without modifying files.
func Promote(dir string, dryRun bool) (*PromoteResult, error) {
	// Load task -> traces-to mapping from tasks.md
	taskTraces, err := loadTaskTraces(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load task traces: %w", err)
	}

	result := &PromoteResult{
		Promotions: make([]Promotion, 0),
		Skipped:    make([]SkippedPromotion, 0),
	}

	// Pattern for TASK traces in test files: // traces: TASK-NNN
	taskTracePattern := regexp.MustCompile(`^(\s*//\s*traces:\s*)TASK-(\d{3})\s*$`)

	// Walk directory looking for test files
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories we shouldn't process
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process test files
		if !isTestFile(info.Name()) {
			return nil
		}

		// Read file content
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		relPath, _ := filepath.Rel(dir, path)
		content := string(data)
		lines := strings.Split(content, "\n")
		modified := false

		// Process each line looking for TASK traces
		for i, line := range lines {
			match := taskTracePattern.FindStringSubmatch(line)
			if match == nil {
				continue
			}

			taskID := "TASK-" + match[2]
			prefix := match[1] // preserve leading whitespace and "// traces: "

			// Look up what this task traces to
			info, found := taskTraces[taskID]
			if !found || !info.found {
				result.Skipped = append(result.Skipped, SkippedPromotion{
					File:   relPath,
					Line:   i + 1,
					TaskID: taskID,
					Reason: fmt.Sprintf("task %s not found in tasks.md", taskID),
				})
				continue
			}

			if info.tracesTo == "" {
				result.Skipped = append(result.Skipped, SkippedPromotion{
					File:   relPath,
					Line:   i + 1,
					TaskID: taskID,
					Reason: fmt.Sprintf("task %s has no Traces-to field", taskID),
				})
				continue
			}

			// Replace the line with new trace target
			lines[i] = prefix + info.tracesTo
			modified = true

			result.Promotions = append(result.Promotions, Promotion{
				File:     relPath,
				Line:     i + 1,
				OldTrace: taskID,
				NewTrace: info.tracesTo,
			})
		}

		// Write back if modified (unless dry run)
		if modified && !dryRun {
			newContent := strings.Join(lines, "\n")
			if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
				return fmt.Errorf("failed to write %s: %w", relPath, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// isTestFile returns true if the filename matches test file patterns.
func isTestFile(name string) bool {
	// Go test files
	if strings.HasSuffix(name, "_test.go") {
		return true
	}

	// TypeScript/JavaScript test files
	if strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".spec.ts") {
		return true
	}
	if strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".spec.js") {
		return true
	}
	if strings.HasSuffix(name, ".test.tsx") || strings.HasSuffix(name, ".spec.tsx") {
		return true
	}
	if strings.HasSuffix(name, ".test.jsx") || strings.HasSuffix(name, ".spec.jsx") {
		return true
	}

	return false
}

// loadTaskTraces reads tasks.md and builds a map of TASK-NNN -> task info.
func loadTaskTraces(dir string) (map[string]taskInfo, error) {
	// Get config to find tasks.md path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(dir, homeDir, &realConfigFS{})
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	tasksPath := filepath.Join(dir, cfg.ResolvePath("tasks"))

	data, err := os.ReadFile(tasksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]taskInfo), nil
		}
		return nil, fmt.Errorf("failed to read tasks.md: %w", err)
	}

	return parseTaskTraces(string(data)), nil
}

// taskInfo holds parsed information about a task.
type taskInfo struct {
	found    bool   // task was found in tasks.md
	tracesTo string // the Traces-to value, empty if not present
}

// parseTaskTraces extracts TASK-NNN -> traces-to mappings from tasks.md content.
// Returns a map where presence indicates the task exists, and the value contains trace info.
func parseTaskTraces(content string) map[string]taskInfo {
	result := make(map[string]taskInfo)

	// Pattern for task header: ### TASK-NNN: Title
	taskPattern := regexp.MustCompile(`^###\s+(TASK-\d{3}):\s*`)
	// Pattern for traces-to field: **Traces to:** TARGET-NNN[, TARGET-NNN...]
	tracesToPattern := regexp.MustCompile(`^\*\*Traces to:\*\*\s*(.+)`)

	lines := strings.Split(content, "\n")
	var currentTask string

	for _, line := range lines {
		// Check for new task header
		if match := taskPattern.FindStringSubmatch(line); match != nil {
			// Save previous task if it had no traces-to
			if currentTask != "" {
				if _, exists := result[currentTask]; !exists {
					result[currentTask] = taskInfo{found: true, tracesTo: ""}
				}
			}
			currentTask = match[1]
			// Mark task as found (may update tracesTo later)
			result[currentTask] = taskInfo{found: true, tracesTo: ""}
			continue
		}

		// Check for Traces to: field (must be within a task section)
		if currentTask != "" {
			if match := tracesToPattern.FindStringSubmatch(line); match != nil {
				result[currentTask] = taskInfo{found: true, tracesTo: strings.TrimSpace(match[1])}
				continue
			}

			// Reset current task on section boundary (but not on new task header - handled above)
			if strings.HasPrefix(line, "---") {
				currentTask = ""
			}
		}
	}

	return result
}
