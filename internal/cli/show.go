package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// unexported variables.
var (
	errShowMissingSlug = errors.New("show: slug argument required")
)

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// loadMemoryTOML reads and decodes a single memory TOML file into MemoryRecord.
func loadMemoryTOML(path string) (*memory.MemoryRecord, error) {
	var record memory.MemoryRecord

	_, err := toml.DecodeFile(path, &record)
	if err != nil {
		return nil, fmt.Errorf("decoding TOML: %w", err)
	}

	return &record, nil
}

// renderFactContent writes fact-specific fields to w.
func renderFactContent(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Situation != "" {
		_, _ = fmt.Fprintf(writer, "Situation: %s\n", mem.Situation)
	}

	if mem.Content.Subject != "" {
		_, _ = fmt.Fprintf(writer, "Subject: %s\n", mem.Content.Subject)
	}

	if mem.Content.Predicate != "" {
		_, _ = fmt.Fprintf(writer, "Predicate: %s\n", mem.Content.Predicate)
	}

	if mem.Content.Object != "" {
		_, _ = fmt.Fprintf(writer, "Object: %s\n", mem.Content.Object)
	}
}

// renderFeedbackContent writes feedback-specific SBIA fields to w.
func renderFeedbackContent(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Situation != "" {
		_, _ = fmt.Fprintf(writer, "Situation: %s\n", mem.Situation)
	}

	if mem.Content.Behavior != "" {
		_, _ = fmt.Fprintf(writer, "Behavior: %s\n", mem.Content.Behavior)
	}

	if mem.Content.Impact != "" {
		_, _ = fmt.Fprintf(writer, "Impact: %s\n", mem.Content.Impact)
	}

	if mem.Content.Action != "" {
		_, _ = fmt.Fprintf(writer, "Action: %s\n", mem.Content.Action)
	}
}

// renderMemory writes formatted memory details to w.
// Only fields with non-empty/non-zero values are printed.
func renderMemory(writer io.Writer, mem *memory.MemoryRecord) {
	renderMemoryContent(writer, mem)
}

// renderMemoryContent writes the content fields of a memory record to w.
// Facts show Subject/Predicate/Object; feedback shows Situation/Behavior/Impact/Action.
func renderMemoryContent(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Type != "" {
		_, _ = fmt.Fprintf(writer, "Type: %s\n", mem.Type)
	}

	if mem.Type == typeFact {
		renderFactContent(writer, mem)
	} else {
		renderFeedbackContent(writer, mem)
	}

	if mem.Source != "" {
		_, _ = fmt.Fprintf(writer, "Source: %s\n", mem.Source)
	}

	if mem.CreatedAt != "" {
		_, _ = fmt.Fprintf(writer, "Created: %s\n", mem.CreatedAt)
	}

	if mem.UpdatedAt != "" {
		_, _ = fmt.Fprintf(writer, "Updated: %s\n", mem.UpdatedAt)
	}
}

// runShow implements the show subcommand: displays full details of a memory.
func runShow(ctx context.Context, args ShowArgs, stdout io.Writer) error {
	_ = ctx

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("show: %w", defaultErr)
	}

	if args.Name == "" {
		return errShowMissingSlug
	}

	memPath := memory.ResolveMemoryPath(dataDir, args.Name, fileExists)

	mem, err := loadMemoryTOML(memPath)
	if err != nil {
		return fmt.Errorf("show: loading %s: %w", args.Name, err)
	}

	renderMemory(stdout, mem)

	return nil
}
