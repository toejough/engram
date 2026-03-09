package cli

import (
	"io"
	"time"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/promote"
	regpkg "engram/internal/registry"
	"engram/internal/retrieve"
	reviewpkg "engram/internal/review"
)

// Exported variables.
var (
	ExportBuildEscalationMemories = buildEscalationMemories
	ExportBuildExtractor          = buildExtractor
	ExportContentHash             = contentHashForRegistry
	ExportLoadMemoryContent       = loadMemoryContent
	ExportLoadSkillContent        = loadSkillContent
	ExportParseRemindersToml      = parseRemindersToml
	ExportResolveSkillsDir        = resolveSkillsDir
	ExportTruncateTitle           = truncateTitle
)

type ExportClassifiedMemory = reviewpkg.ClassifiedMemory

type ExportEscalationMemory = maintain.EscalationMemory

type ExportMemoryContent = promote.MemoryContent

type ExportSkillContent = promote.SkillContent

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

// ExportNewOsCreationLogReader creates an osCreationLogReader for testing.
func ExportNewOsCreationLogReader(dataDir string) *osCreationLogReader {
	return &osCreationLogReader{dataDir: dataDir}
}

// ExportNewOsEvaluationsReader creates an osEvaluationsReader for testing.
func ExportNewOsEvaluationsReader(dataDir string) *osEvaluationsReader {
	return &osEvaluationsReader{dataDir: dataDir}
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

// ExportNewOsSurfacingLogReader creates an osSurfacingLogReader for testing.
func ExportNewOsSurfacingLogReader(dataDir string) *osSurfacingLogReader {
	return &osSurfacingLogReader{dataDir: dataDir}
}

// ExportNewRetriever creates a retrieve.Retriever for testing.
func ExportNewRetriever() *retrieve.Retriever {
	return retrieve.New()
}

// ExportNewStdinConfirmer creates a stdinConfirmer for testing.
func ExportNewStdinConfirmer(stdout io.Writer, stdin io.Reader) maintain.Confirmer {
	return &stdinConfirmer{stdout: stdout, stdin: stdin}
}

// ExportNewTemplateClaudeMDGenerator creates a templateClaudeMDGenerator for testing.
func ExportNewTemplateClaudeMDGenerator() *templateClaudeMDGenerator {
	return &templateClaudeMDGenerator{}
}

// ExportNewTemplateGenerator creates a templateGenerator for testing.
func ExportNewTemplateGenerator() *templateGenerator {
	return &templateGenerator{}
}
