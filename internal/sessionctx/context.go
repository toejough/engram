// Package sessionctx provides transcript processing for LLM agents.
// It reads transcript deltas and strips noisy content for downstream use.
// All I/O is injected via DI interfaces.
package sessionctx

// FileReader reads file contents by path. Wire os.ReadFile in production.
type FileReader interface {
	Read(path string) ([]byte, error)
}
