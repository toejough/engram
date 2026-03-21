package cli

import (
	"io"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/retrieve"
	reviewpkg "engram/internal/review"
)

// Exported variables.
var (
	ExportBuildEscalationMemories = buildEscalationMemories
	ExportResolveSkillsDir        = resolveSkillsDir
	ExportTruncateTitle           = truncateTitle
)

type ExportClassifiedMemory = reviewpkg.ClassifiedMemory

type ExportEscalationMemory = maintain.EscalationMemory

type ExportStored = memory.Stored

// --- Factory functions for structs with unexported fields ---

// ExportNewCliConfirmer creates a cliConfirmer for testing.
func ExportNewCliConfirmer(
	stdout io.Writer, stdin io.Reader, autoConfirm bool,
) maintain.Confirmer {
	return &cliConfirmer{stdout: stdout, stdin: stdin, autoConfirm: autoConfirm}
}

// ExportNewOsClaudeMDStore creates an osClaudeMDStore for testing.
func ExportNewOsClaudeMDStore(path string) interface {
	Read() (string, error)
	Write(content string) error
} {
	return &osClaudeMDStore{path: path}
}

// ExportNewOsMemoryRemover creates an osMemoryRemover for testing.
func ExportNewOsMemoryRemover() *osMemoryRemover {
	return &osMemoryRemover{}
}

// ExportNewOsSkillWriter creates an osSkillWriter for testing.
func ExportNewOsSkillWriter(dir string) interface {
	Write(name, content string) (string, error)
} {
	return &osSkillWriter{dir: dir}
}

// ExportNewRetriever creates a retrieve.Retriever for testing.
func ExportNewRetriever() *retrieve.Retriever {
	return retrieve.New()
}

// ExportNewStdinConfirmer creates a stdinConfirmer for testing.
func ExportNewStdinConfirmer(stdout io.Writer, stdin io.Reader) maintain.Confirmer {
	return &stdinConfirmer{stdout: stdout, stdin: stdin}
}
