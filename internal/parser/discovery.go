package parser

import (
	"errors"
	"path/filepath"
	"strings"
)

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

// DiscoveryConfig holds configuration for documentation discovery.
type DiscoveryConfig struct {
	DocsDir      string // Directory for documentation files
	Requirements string // Requirements file name
	Design       string // Design file name
	Architecture string // Architecture file name
	Tasks        string // Tasks file name
	Issues       string // Issues file name
	Glossary     string // Glossary file name
}

// DiscoverDocsWithConfig discovers documentation files using configuration.
// Checks the configured docs directory first, falls back to root.
func DiscoverDocsWithConfig(root string, cfg *DiscoveryConfig, fs FileSystem) DiscoveryResult {
	docsDir := filepath.Join(root, cfg.DocsDir)

	// Build list of files to discover
	files := collectConfiguredFiles(cfg)

	// Try docs directory first
	if fs.DirExists(docsDir) {
		paths := discoverConfiguredFiles(docsDir, files, fs)
		if len(paths) > 0 {
			return DiscoveryResult{
				Paths:        paths,
				UsedFallback: false,
			}
		}
	}

	// Fall back to root
	paths := discoverConfiguredFiles(root, files, fs)
	if len(paths) > 0 {
		return DiscoveryResult{
			Paths:        paths,
			UsedFallback: true,
		}
	}

	return DiscoveryResult{}
}

// collectConfiguredFiles builds a list of non-empty file names from config.
func collectConfiguredFiles(cfg *DiscoveryConfig) []string {
	var files []string
	if cfg.Requirements != "" {
		files = append(files, cfg.Requirements)
	}
	if cfg.Design != "" {
		files = append(files, cfg.Design)
	}
	if cfg.Architecture != "" {
		files = append(files, cfg.Architecture)
	}
	if cfg.Tasks != "" {
		files = append(files, cfg.Tasks)
	}
	if cfg.Issues != "" {
		files = append(files, cfg.Issues)
	}
	if cfg.Glossary != "" {
		files = append(files, cfg.Glossary)
	}
	return files
}

// discoverConfiguredFiles finds specified files in a directory.
func discoverConfiguredFiles(dir string, files []string, fs FileSystem) []string {
	var paths []string
	for _, name := range files {
		path := filepath.Join(dir, name)
		if fs.FileExists(path) {
			paths = append(paths, path)
		}
	}
	return paths
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

// errSkipDir signals to skip this directory
var errSkipDir = errors.New("skip")

// DiscoverTestFiles finds all *_test.go files in the given root.
// Excludes vendor/ and .git/ directories.
func DiscoverTestFiles(root string, fs WalkableFS) []string {
	var paths []string

	_ = fs.Walk(root, func(path string, isDir bool) error {
		// Skip excluded directories
		if isDir {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" {
				return errSkipDir
			}
			return nil
		}

		// Check for test file pattern
		if strings.HasSuffix(path, "_test.go") {
			// Make sure it's not in an excluded directory
			if !strings.Contains(path, "/vendor/") && !strings.Contains(path, "/.git/") {
				paths = append(paths, path)
			}
		}

		return nil
	})

	return paths
}
