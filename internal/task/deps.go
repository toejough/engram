package task

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DependencyGraph represents task dependencies.
type DependencyGraph struct {
	Tasks     []string            // All task IDs
	Deps      map[string][]string // Task ID -> dependencies
	Status    map[string]string   // Task ID -> status (pending, complete)
	hasCycle  bool
	cyclePath []string
}

// ParseDependencies parses task dependencies from tasks.md.
func ParseDependencies(dir string) (*DependencyGraph, error) {
	tasksPath := filepath.Join(dir, "tasks.md")

	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, err
	}

	graph := &DependencyGraph{
		Deps:   make(map[string][]string),
		Status: make(map[string]string),
	}

	// Find all tasks
	taskPattern := regexp.MustCompile(`(?m)^###\s+(TASK-\d+):`)
	matches := taskPattern.FindAllStringSubmatch(string(content), -1)

	for _, m := range matches {
		taskID := m[1]
		graph.Tasks = append(graph.Tasks, taskID)

		// Extract task content
		taskContent := extractTask(string(content), taskID)

		// Parse dependencies
		deps := parseDependencies(taskContent)
		graph.Deps[taskID] = deps

		// Parse status
		graph.Status[taskID] = parseStatus(taskContent)
	}

	// Check for cycles
	graph.detectCycle()

	return graph, nil
}

// parseDependencies extracts dependencies from task content.
func parseDependencies(content string) []string {
	// Match **Dependencies:** TASK-XXX, TASK-YYY or **Depends on:** ...
	pattern := regexp.MustCompile(`(?m)^\*\*(Dependencies|Depends on):\*\*\s*(.+)$`)
	match := pattern.FindStringSubmatch(content)
	if match == nil {
		return nil
	}

	depsStr := strings.TrimSpace(match[2])
	if depsStr == "None" || depsStr == "none" || depsStr == "" {
		return nil
	}

	// Split by comma and extract task IDs
	taskPattern := regexp.MustCompile(`TASK-\d+`)
	return taskPattern.FindAllString(depsStr, -1)
}

// parseStatus extracts task status from content.
func parseStatus(content string) string {
	pattern := regexp.MustCompile(`(?m)^\*\*Status:\*\*\s*(\w+)`)
	match := pattern.FindStringSubmatch(content)
	if match == nil {
		return "pending" // Default to pending
	}
	return strings.ToLower(match[1])
}

// Parallel returns tasks that can run in parallel (pending with no pending blockers).
func Parallel(dir string) ([]string, error) {
	graph, err := ParseDependencies(dir)
	if err != nil {
		return nil, err
	}

	var parallel []string
	for _, task := range graph.Tasks {
		// Skip non-pending tasks
		if graph.Status[task] != "pending" {
			continue
		}

		// Check if all dependencies are complete
		blocked := false
		for _, dep := range graph.Deps[task] {
			if graph.Status[dep] != "complete" {
				blocked = true
				break
			}
		}

		if !blocked {
			parallel = append(parallel, task)
		}
	}

	return parallel, nil
}

// HasCycle returns true if the dependency graph contains a cycle.
func (g *DependencyGraph) HasCycle() bool {
	return g.hasCycle
}

// CyclePath returns the cycle path if one exists.
func (g *DependencyGraph) CyclePath() []string {
	return g.cyclePath
}

// Roots returns tasks with no dependencies.
func (g *DependencyGraph) Roots() []string {
	var roots []string
	for _, task := range g.Tasks {
		if len(g.Deps[task]) == 0 {
			roots = append(roots, task)
		}
	}
	return roots
}

// detectCycle uses DFS to find cycles.
func (g *DependencyGraph) detectCycle() {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	for _, task := range g.Tasks {
		if visited[task] {
			continue
		}
		if g.dfs(task, visited, inStack, []string{}) {
			return
		}
	}
}

func (g *DependencyGraph) dfs(task string, visited, inStack map[string]bool, path []string) bool {
	visited[task] = true
	inStack[task] = true
	path = append(path, task)

	for _, dep := range g.Deps[task] {
		if inStack[dep] {
			// Found a cycle
			cycleStart := -1
			for i, t := range path {
				if t == dep {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				g.cyclePath = path[cycleStart:]
			} else {
				g.cyclePath = []string{dep}
			}
			g.hasCycle = true
			return true
		}
		if !visited[dep] {
			if g.dfs(dep, visited, inStack, path) {
				return true
			}
		}
	}

	inStack[task] = false
	return false
}
