package parser

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

// DiscoverDocs discovers documentation files in the given project root.
// Checks docs/ directory first, falls back to root if docs/ doesn't exist.
func DiscoverDocs(root string, fs FileSystem) DiscoveryResult {
	return DiscoveryResult{}
}
