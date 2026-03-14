package cli

import (
	"io"
	"time"

	"engram/internal/maintain"
	"engram/internal/memory"
	regpkg "engram/internal/registry"
	"engram/internal/retrieve"
	reviewpkg "engram/internal/review"
)

// Exported variables.
var (
	ExportBuildEscalationMemories = buildEscalationMemories
	ExportContentHash             = contentHash
	ExportParseRemindersToml      = parseRemindersToml
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

// ExportNewEvaluateRegistryAdapter creates an evaluateRegistryAdapter for testing.
func ExportNewEvaluateRegistryAdapter(reg regpkg.Registry) interface {
	RecordEvaluation(id, outcome string) error
} {
	return &evaluateRegistryAdapter{reg: reg}
}

// ExportNewLearnRegistryAdapter creates a learnRegistryAdapter for testing.
func ExportNewLearnRegistryAdapter(reg regpkg.Registry) interface {
	RegisterMemory(filePath, title, content string, now time.Time) error
} {
	return &learnRegistryAdapter{reg: reg}
}

// ExportNewNoopTranscriptReader creates a noopTranscriptReader for testing.
func ExportNewNoopTranscriptReader() interface {
	ReadRecent(n int) (string, error)
} {
	return &noopTranscriptReader{}
}

// ExportNewOsClaudeMDStore creates an osClaudeMDStore for testing.
func ExportNewOsClaudeMDStore(path string) interface {
	Read() (string, error)
	Write(content string) error
} {
	return &osClaudeMDStore{path: path}
}

// ExportNewOsMemoryLoader creates an osMemoryLoader for testing.
func ExportNewOsMemoryLoader(dataDir string) *osMemoryLoader {
	return &osMemoryLoader{dataDir: dataDir}
}

// ExportNewOsMemoryRemover creates an osMemoryRemover for testing.
func ExportNewOsMemoryRemover() *osMemoryRemover {
	return &osMemoryRemover{}
}

// ExportNewOsRemindConfigReader creates an osRemindConfigReader for testing.
func ExportNewOsRemindConfigReader(dataDir string) *osRemindConfigReader {
	return &osRemindConfigReader{dataDir: dataDir}
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
