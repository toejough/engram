package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// unexported variables.
var (
	errShowMissingDataDir = errors.New("show: --data-dir required")
	errShowMissingSlug    = errors.New("show: slug argument required")
)

// effectivenessPercent computes followed/total as a rounded integer percentage.
func effectivenessPercent(followed, total int) int {
	const percentMultiplier = 100

	return (followed * percentMultiplier) / total
}

// extractSlug separates a positional slug argument from flag arguments.
// It scans args for the first non-flag token that is not a flag-value
// (i.e., it skips values that follow known flag prefixes like "--data-dir").
// Returns the slug (may be empty) and remaining args for flag parsing.
func extractSlug(args []string) (string, []string) {
	remaining := make([]string, 0, len(args))
	skipNext := false

	for idx, arg := range args {
		if skipNext {
			skipNext = false

			remaining = append(remaining, arg)

			continue
		}

		if strings.HasPrefix(arg, "-") {
			remaining = append(remaining, arg)

			// If this flag uses "--key value" form (not "--key=value"),
			// the next arg is its value — skip it.
			if !strings.Contains(arg, "=") && idx+1 < len(args) {
				skipNext = true
			}

			continue
		}

		// First non-flag, non-value token is the slug.
		return arg, append(remaining, args[idx+1:]...)
	}

	return "", remaining
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

// renderMemory writes formatted memory details to w.
// Only fields with non-empty/non-zero values are printed.
// Effectiveness is printed only when total evaluations > 0.
func renderMemory(writer io.Writer, mem *memory.MemoryRecord) {
	renderMemoryContent(writer, mem)
	renderMemoryMeta(writer, mem)
}

// renderMemoryContent writes the content fields of a memory record to w.
func renderMemoryContent(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Title != "" {
		_, _ = fmt.Fprintf(writer, "Title: %s\n", mem.Title)
	}

	if mem.ObservationType != "" {
		_, _ = fmt.Fprintf(writer, "Type: %s\n", mem.ObservationType)
	}

	if mem.Principle != "" {
		_, _ = fmt.Fprintf(writer, "Principle: %s\n", mem.Principle)
	}

	if mem.AntiPattern != "" {
		_, _ = fmt.Fprintf(writer, "Anti-pattern: %s\n", mem.AntiPattern)
	}

	if mem.Rationale != "" {
		_, _ = fmt.Fprintf(writer, "Rationale: %s\n", mem.Rationale)
	}

	if mem.Content != "" {
		_, _ = fmt.Fprintf(writer, "Content: %s\n", mem.Content)
	}

	if len(mem.Keywords) > 0 {
		_, _ = fmt.Fprintf(writer, "Keywords: %s\n",
			strings.Join(mem.Keywords, ", "))
	}
}

// renderMemoryMeta writes the metadata and tracking fields of a memory record to w.
func renderMemoryMeta(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Confidence != "" {
		_, _ = fmt.Fprintf(writer, "Confidence: %s\n", mem.Confidence)
	}

	if mem.CreatedAt != "" {
		_, _ = fmt.Fprintf(writer, "Created: %s\n", mem.CreatedAt)
	}

	if mem.LastSurfacedAt != "" {
		_, _ = fmt.Fprintf(writer, "Last surfaced: %s\n", mem.LastSurfacedAt)
	}

	total := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
	if total > 0 {
		pct := effectivenessPercent(mem.FollowedCount, total)
		_, _ = fmt.Fprintf(writer,
			"Effectiveness: %d%% (%d followed, %d contradicted, %d ignored)\n",
			pct, mem.FollowedCount, mem.ContradictedCount, mem.IgnoredCount)
	}

	if mem.IrrelevantCount > 0 {
		totalFeedback := mem.FollowedCount + mem.ContradictedCount +
			mem.IgnoredCount + mem.IrrelevantCount
		relevantFeedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
		pct := effectivenessPercent(relevantFeedback, totalFeedback)
		_, _ = fmt.Fprintf(writer,
			"Relevance: %d%% (%d relevant, %d irrelevant of %d feedback)\n",
			pct, relevantFeedback, mem.IrrelevantCount, totalFeedback)
	}
}

// resolveSlug picks the slug from positional arg, --name flag, or trailing arg (in priority order).
func resolveSlug(positional, nameFlag string, fs *flag.FlagSet) string {
	if positional != "" {
		return positional
	}

	if nameFlag != "" {
		return nameFlag
	}

	if fs.NArg() > 0 {
		return fs.Arg(0)
	}

	return ""
}

// runShow implements the show subcommand: displays full details of a memory.
// Supports slug before or after flags (e.g., "show my-mem --data-dir /path").
func runShow(args []string, stdout io.Writer) error {
	// Separate the positional slug from flag args so flag.Parse works
	// regardless of argument order.
	slug, flagArgs := extractSlug(args)

	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	nameFlag := fs.String("name", "", "memory slug (alternative to positional arg)")

	parseErr := fs.Parse(flagArgs)
	if parseErr != nil {
		return fmt.Errorf("show: %w", parseErr)
	}

	slug = resolveSlug(slug, *nameFlag, fs)

	if *dataDir == "" {
		return errShowMissingDataDir
	}

	if slug == "" {
		return errShowMissingSlug
	}

	memPath := filepath.Join(*dataDir, "memories", slug+".toml")

	mem, err := loadMemoryTOML(memPath)
	if err != nil {
		return fmt.Errorf("show: loading %s: %w", slug, err)
	}

	renderMemory(stdout, mem)

	return nil
}
