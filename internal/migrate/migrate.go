// Package migrate converts legacy memory files to the v2 format.
package migrate

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Deps holds injected I/O dependencies for the migration.
type Deps struct {
	ReadDir    func(string) ([]os.DirEntry, error)
	Stat       func(string) (os.FileInfo, error)
	MkdirAll   func(string, os.FileMode) error
	Rename     func(string, string) error
	ReadFile   func(string) ([]byte, error)
	CreateTemp func(string, string) (*os.File, error)
	Remove     func(string) error
	Stdout     io.Writer
	Stderr     io.Writer
}

// DefaultDeps returns Deps wired to real filesystem operations.
func DefaultDeps() Deps {
	return Deps{
		ReadDir:    os.ReadDir,
		Stat:       os.Stat,
		MkdirAll:   os.MkdirAll,
		Rename:     os.Rename,
		ReadFile:   os.ReadFile,
		CreateTemp: os.CreateTemp,
		Remove:     os.Remove,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
}

// Run migrates all .toml files from dataDir/memories/ to dataDir/memory/feedback/.
// It creates the facts directory, backs up the source, and is idempotent.
func Run(dataDir string, deps Deps) error {
	srcDir := filepath.Join(dataDir, "memories")
	dstDir := filepath.Join(dataDir, "memory", "feedback")
	factsDir := filepath.Join(dataDir, "memory", "facts")
	backupDir := filepath.Join(dataDir, "memories.v1-backup")

	entries, readErr := deps.ReadDir(srcDir)
	if readErr != nil {
		return fmt.Errorf("reading %s: %w", srcDir, readErr)
	}

	mkErr := deps.MkdirAll(dstDir, dirPerm)
	if mkErr != nil {
		return fmt.Errorf("creating %s: %w", dstDir, mkErr)
	}

	mkFactsErr := deps.MkdirAll(factsDir, dirPerm)
	if mkFactsErr != nil {
		return fmt.Errorf("creating %s: %w", factsDir, mkFactsErr)
	}

	migrated := 0
	skipped := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		dstPath := filepath.Join(dstDir, entry.Name())

		_, statErr := deps.Stat(dstPath)
		if statErr == nil {
			_, _ = fmt.Fprintf(deps.Stdout, "SKIP (exists): %s\n", entry.Name())

			skipped++

			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())

		migrateErr := migrateFile(srcPath, dstPath, deps)
		if migrateErr != nil {
			return fmt.Errorf("migrating %s: %w", entry.Name(), migrateErr)
		}

		_, _ = fmt.Fprintf(deps.Stdout, "OK: %s\n", entry.Name())

		migrated++
	}

	renameErr := deps.Rename(srcDir, backupDir)
	if renameErr != nil {
		_, _ = fmt.Fprintf(deps.Stderr, "warning: could not rename %s to %s: %v\n",
			srcDir, backupDir, renameErr)
	}

	_, _ = fmt.Fprintf(deps.Stdout, "\nMigrated: %d, Skipped: %d\n", migrated, skipped)

	return nil
}

// RunCLI parses flags from args and runs the migration.
// Returns a non-zero exit code on failure.
func RunCLI(args []string) int {
	flags := flag.NewFlagSet("migrate-v2", flag.ContinueOnError)
	dataDir := flags.String("data-dir",
		filepath.Join(os.Getenv("HOME"), ".claude", "engram", "data"),
		"path to data directory")

	parseErr := flags.Parse(args)
	if parseErr != nil {
		return 1
	}

	runErr := Run(*dataDir, DefaultDeps())
	if runErr != nil {
		return 1
	}

	return 0
}

// unexported constants.
const (
	dirPerm = 0o750
)

// contentSection nests feedback fields under [content] in the new format.
type contentSection struct {
	Behavior string `toml:"behavior,omitempty"`
	Impact   string `toml:"impact,omitempty"`
	Action   string `toml:"action,omitempty"`
}

// legacyRecord represents the old memory format with top-level behavior/impact/action.
type legacyRecord struct {
	SchemaVersion     int     `toml:"schema_version,omitempty"`
	Situation         string  `toml:"situation"`
	Behavior          string  `toml:"behavior"`
	Impact            string  `toml:"impact"`
	Action            string  `toml:"action"`
	ProjectScoped     bool    `toml:"project_scoped"`
	ProjectSlug       string  `toml:"project_slug,omitempty"`
	CreatedAt         string  `toml:"created_at"`
	UpdatedAt         string  `toml:"updated_at"`
	SurfacedCount     int     `toml:"surfaced_count"`
	FollowedCount     int     `toml:"followed_count"`
	NotFollowedCount  int     `toml:"not_followed_count"`
	IrrelevantCount   int     `toml:"irrelevant_count"`
	MissedCount       int     `toml:"missed_count"`
	InitialConfidence float64 `toml:"initial_confidence,omitempty"`
}

// v2Record represents the v2 memory format.
type v2Record struct {
	SchemaVersion     int            `toml:"schema_version"`
	Type              string         `toml:"type"`
	Situation         string         `toml:"situation"`
	Source            string         `toml:"source,omitempty"`
	Core              bool           `toml:"core,omitempty"`
	ProjectScoped     bool           `toml:"project_scoped"`
	ProjectSlug       string         `toml:"project_slug,omitempty"`
	Content           contentSection `toml:"content"`
	CreatedAt         string         `toml:"created_at"`
	UpdatedAt         string         `toml:"updated_at"`
	SurfacedCount     int            `toml:"surfaced_count"`
	FollowedCount     int            `toml:"followed_count"`
	NotFollowedCount  int            `toml:"not_followed_count"`
	IrrelevantCount   int            `toml:"irrelevant_count"`
	MissedCount       int            `toml:"missed_count"`
	InitialConfidence float64        `toml:"initial_confidence,omitempty"`
}

func atomicWrite(dst string, out v2Record, deps Deps) error {
	dir := filepath.Dir(dst)

	tmpFile, createErr := deps.CreateTemp(dir, ".tmp-migrate-*")
	if createErr != nil {
		return fmt.Errorf("creating temp file: %w", createErr)
	}

	tmpPath := tmpFile.Name()

	encErr := toml.NewEncoder(tmpFile).Encode(out)

	closeErr := tmpFile.Close()

	if encErr != nil {
		_ = deps.Remove(tmpPath)

		return fmt.Errorf("encoding TOML: %w", encErr)
	}

	if closeErr != nil {
		_ = deps.Remove(tmpPath)

		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	// Validate by re-reading the temp file.
	verifyData, verifyReadErr := deps.ReadFile(tmpPath)
	if verifyReadErr != nil {
		_ = deps.Remove(tmpPath)

		return fmt.Errorf("reading temp for validation: %w", verifyReadErr)
	}

	var verify v2Record

	_, verifyDecErr := toml.Decode(string(verifyData), &verify)
	if verifyDecErr != nil {
		_ = deps.Remove(tmpPath)

		return fmt.Errorf("validation re-read: %w", verifyDecErr)
	}

	renameErr := deps.Rename(tmpPath, dst)
	if renameErr != nil {
		_ = deps.Remove(tmpPath)

		return fmt.Errorf("renaming temp to destination: %w", renameErr)
	}

	return nil
}

func migrateFile(src, dst string, deps Deps) error {
	data, readErr := deps.ReadFile(src)
	if readErr != nil {
		return fmt.Errorf("reading %s: %w", src, readErr)
	}

	var legacy legacyRecord

	_, decErr := toml.Decode(string(data), &legacy)
	if decErr != nil {
		return fmt.Errorf("decoding %s: %w", src, decErr)
	}

	out := v2Record{
		SchemaVersion: 1,
		Type:          "feedback",
		Situation:     legacy.Situation,
		ProjectScoped: legacy.ProjectScoped,
		ProjectSlug:   legacy.ProjectSlug,
		Content: contentSection{
			Behavior: legacy.Behavior,
			Impact:   legacy.Impact,
			Action:   legacy.Action,
		},
		CreatedAt:         legacy.CreatedAt,
		UpdatedAt:         legacy.UpdatedAt,
		SurfacedCount:     legacy.SurfacedCount,
		FollowedCount:     legacy.FollowedCount,
		NotFollowedCount:  legacy.NotFollowedCount,
		IrrelevantCount:   legacy.IrrelevantCount,
		MissedCount:       legacy.MissedCount,
		InitialConfidence: legacy.InitialConfidence,
	}

	return atomicWrite(dst, out, deps)
}
