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

// showTOMLRecord mirrors the on-disk TOML format for the show command.
type showTOMLRecord struct {
	Title             string   `toml:"title"`
	Content           string   `toml:"content"`
	Concepts          []string `toml:"concepts"`
	Keywords          []string `toml:"keywords"`
	AntiPattern       string   `toml:"anti_pattern"`
	Principle         string   `toml:"principle"`
	SurfacedCount     int      `toml:"surfaced_count"`
	FollowedCount     int      `toml:"followed_count"`
	ContradictedCount int      `toml:"contradicted_count"`
	IgnoredCount      int      `toml:"ignored_count"`
}

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

// loadMemoryTOML reads and decodes a single memory TOML file.
func loadMemoryTOML(path string) (*memory.Stored, error) {
	var record showTOMLRecord

	_, err := toml.DecodeFile(path, &record)
	if err != nil {
		return nil, fmt.Errorf("decoding TOML: %w", err)
	}

	return &memory.Stored{
		Title:             record.Title,
		Content:           record.Content,
		Concepts:          record.Concepts,
		Keywords:          record.Keywords,
		AntiPattern:       record.AntiPattern,
		Principle:         record.Principle,
		SurfacedCount:     record.SurfacedCount,
		FollowedCount:     record.FollowedCount,
		ContradictedCount: record.ContradictedCount,
		IgnoredCount:      record.IgnoredCount,
		FilePath:          path,
	}, nil
}

// renderMemory writes formatted memory details to w.
// Only fields with non-empty values are printed.
// Effectiveness is printed only when total evaluations > 0.
func renderMemory(writer io.Writer, mem *memory.Stored) {
	if mem.Title != "" {
		_, _ = fmt.Fprintf(writer, "Title: %s\n", mem.Title)
	}

	if mem.Principle != "" {
		_, _ = fmt.Fprintf(writer, "Principle: %s\n", mem.Principle)
	}

	if mem.AntiPattern != "" {
		_, _ = fmt.Fprintf(writer, "Anti-pattern: %s\n", mem.AntiPattern)
	}

	if mem.Content != "" {
		_, _ = fmt.Fprintf(writer, "Content: %s\n", mem.Content)
	}

	if len(mem.Keywords) > 0 {
		_, _ = fmt.Fprintf(writer, "Keywords: %s\n",
			strings.Join(mem.Keywords, ", "))
	}

	total := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
	if total > 0 {
		pct := effectivenessPercent(mem.FollowedCount, total)
		_, _ = fmt.Fprintf(writer,
			"Effectiveness: %d%% (%d followed, %d contradicted, %d ignored)\n",
			pct, mem.FollowedCount, mem.ContradictedCount, mem.IgnoredCount)
	}
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

	parseErr := fs.Parse(flagArgs)
	if parseErr != nil {
		return fmt.Errorf("show: %w", parseErr)
	}

	// Pick up slug from trailing positional args if not found before flags.
	if slug == "" && fs.NArg() > 0 {
		slug = fs.Arg(0)
	}

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
