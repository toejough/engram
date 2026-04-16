package externalsources

import (
	"path/filepath"
	"regexp"
	"strings"
)

// ExpandImports walks @path imports starting from the given file, returning
// every distinct file reached (excluding the starting file itself). Cycles
// are detected and broken; depth is capped at maxImportHops.
func ExpandImports(startPath string, reader ReaderFunc) []ExternalFile {
	visited := map[string]bool{startPath: true}
	pattern := importRegexp()

	discovered := make([]ExternalFile, 0)
	queue := []importNode{{path: startPath, depth: 0}}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.depth >= maxImportHops {
			continue
		}

		body, err := reader(node.path)
		if err != nil || body == nil {
			continue
		}

		newDiscovered, newQueue := enqueueImports(node, body, pattern, visited)
		discovered = append(discovered, newDiscovered...)
		queue = append(queue, newQueue...)
	}

	return discovered
}

// unexported constants.
const (
	maxImportHops = 5
)

// importNode represents one item in the BFS queue.
type importNode struct {
	path  string
	depth int
}

// enqueueImports parses @path references from body and returns the new files
// to record plus the new BFS nodes to enqueue. Mutates visited so subsequent
// references to the same path are skipped.
func enqueueImports(
	node importNode, body []byte, pattern *regexp.Regexp, visited map[string]bool,
) ([]ExternalFile, []importNode) {
	matches := pattern.FindAllStringSubmatch(string(body), -1)

	discovered := make([]ExternalFile, 0, len(matches))
	queued := make([]importNode, 0, len(matches))

	for _, match := range matches {
		rawPath := match[1]
		absPath := resolveImportPath(node.path, rawPath)

		if visited[absPath] {
			continue
		}

		visited[absPath] = true

		discovered = append(discovered, ExternalFile{Kind: KindClaudeMd, Path: absPath})
		queued = append(queued, importNode{path: absPath, depth: node.depth + 1})
	}

	return discovered, queued
}

// importRegexp compiles the @path pattern. The expression matches:
//   - start-of-line, whitespace, or "(" before the "@" so we don't pick up
//     email addresses
//   - path segments without whitespace, "@", or parentheses
func importRegexp() *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[\s(])@([^\s@()]+)`)
}

// resolveImportPath returns the absolute path for a @import target relative
// to the file that contains the import. Absolute paths are returned as-is.
// "~" paths are returned as-is (to be expanded by callers if needed).
func resolveImportPath(containingFile, importTarget string) string {
	if strings.HasPrefix(importTarget, "/") || strings.HasPrefix(importTarget, "~") {
		return importTarget
	}

	return filepath.Join(filepath.Dir(containingFile), importTarget)
}
