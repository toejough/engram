package parser

import "path/filepath"

// FileSystem provides file system operations for discovery.
// This interface allows dependency injection for testing.
type FileSystem interface {
	DirExists(path string) bool
	FileExists(path string) bool
}

// DiscoveryResult contains discovered documentation file paths.
type DiscoveryResult struct {
	Paths        []string // Discovered file paths
	UsedFallback bool     // True if root fallback was used
}

// knownDocFiles lists the documentation files to discover.
var knownDocFiles = []string{
	"requirements.md",
	"design.md",
	"architecture.md",
	"tasks.md",
}

// DiscoverDocs discovers documentation files in the given project root.
// Checks docs/ directory first, falls back to root if docs/ doesn't exist.
func DiscoverDocs(root string, fs FileSystem) DiscoveryResult {
	docsDir := filepath.Join(root, "docs")

	// Try docs/ directory first
	if fs.DirExists(docsDir) {
		paths := discoverFilesIn(docsDir, fs)
		if len(paths) > 0 {
			return DiscoveryResult{
				Paths:        paths,
				UsedFallback: false,
			}
		}
	}

	// Fall back to root
	paths := discoverFilesIn(root, fs)
	if len(paths) > 0 {
		return DiscoveryResult{
			Paths:        paths,
			UsedFallback: true,
		}
	}

	return DiscoveryResult{}
}

// discoverFilesIn finds known documentation files in the given directory.
func discoverFilesIn(dir string, fs FileSystem) []string {
	var paths []string
	for _, name := range knownDocFiles {
		path := filepath.Join(dir, name)
		if fs.FileExists(path) {
			paths = append(paths, path)
		}
	}
	return paths
}

// WalkableFS provides directory traversal for test file discovery.
type WalkableFS interface {
	Walk(root string, fn func(path string, isDir bool) error) error
}

// DiscoverTestFiles finds all *_test.go files in the given root.
// Excludes vendor/ and .git/ directories.
func DiscoverTestFiles(root string, fs WalkableFS) []string {
	return nil
}
