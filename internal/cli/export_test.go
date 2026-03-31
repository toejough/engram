package cli

import (
	"context"
	"io"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/recall"
	"engram/internal/retrieve"
)

// Exported variables.
var (
	ExportBuildRecallSurfacer     = buildRecallSurfacer
	ExportFindTranscriptForMemory = findTranscriptForMemory
	ExportRecordSurfacing         = recordSurfacing
	ExportResolveSkillsDir        = resolveSkillsDir
	ExportTruncateTitle           = truncateTitle
)

type ExportStored = memory.Stored

// --- Factory functions for structs with unexported fields ---

// ExportNewCliConfirmer creates a cliConfirmer for testing.
func ExportNewCliConfirmer(
	stdout io.Writer, stdin io.Reader, autoConfirm bool,
) maintain.Confirmer {
	return &cliConfirmer{stdout: stdout, stdin: stdin, autoConfirm: autoConfirm}
}

// ExportNewHaikuCallerAdapter creates a haikuCallerAdapter for testing.
func ExportNewHaikuCallerAdapter(
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) recall.HaikuCaller {
	return &haikuCallerAdapter{caller: caller}
}

// ExportNewOsClaudeMDStore creates an osClaudeMDStore for testing.
func ExportNewOsClaudeMDStore(path string) interface {
	Read() (string, error)
	Write(content string) error
} {
	return &osClaudeMDStore{path: path}
}

// ExportNewOsDirLister creates an osDirLister for testing.
func ExportNewOsDirLister() recall.DirLister {
	return &osDirLister{}
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
func ExportNewStdinConfirmer(stdout io.Writer, stdin io.Reader) interface {
	Confirm(preview string) (bool, error)
} {
	return &stdinConfirmer{stdout: stdout, stdin: stdin}
}
