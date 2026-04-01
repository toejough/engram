package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// Exported variables.
var (
	// SBIAConverterOverride allows tests to inject a mock converter.
	//
	//nolint:gochecknoglobals // test-overridable dependency
	SBIAConverterOverride SBIAConverter
)

// LegacyMemoryRecord represents the old keyword/principle memory format.
type LegacyMemoryRecord struct {
	Title            string   `toml:"title"`
	Content          string   `toml:"content"`
	ObservationType  string   `toml:"observation_type"`
	Concepts         []string `toml:"concepts"`
	Keywords         []string `toml:"keywords"`
	Principle        string   `toml:"principle"`
	AntiPattern      string   `toml:"anti_pattern"`
	Rationale        string   `toml:"rationale"`
	ProjectSlug      string   `toml:"project_slug,omitempty"`
	Generalizability int      `toml:"generalizability,omitempty"`
	Confidence       string   `toml:"confidence"`
	CreatedAt        string   `toml:"created_at"`
	UpdatedAt        string   `toml:"updated_at"`

	SurfacedCount     int    `toml:"surfaced_count"`
	FollowedCount     int    `toml:"followed_count"`
	ContradictedCount int    `toml:"contradicted_count"`
	IgnoredCount      int    `toml:"ignored_count"`
	IrrelevantCount   int    `toml:"irrelevant_count"`
	LastSurfacedAt    string `toml:"last_surfaced_at"`
}

// MigrationDeps holds injected dependencies for the migration logic.
type MigrationDeps struct {
	ListDir   func(dir string) ([]os.DirEntry, error)
	ReadFile  func(path string) ([]byte, error)
	WriteFile func(path string, data []byte, perm os.FileMode) error
	MkdirAll  func(path string, perm os.FileMode) error
	Rename    func(oldpath, newpath string) error
	Converter SBIAConverter
	Stdout    io.Writer
}

// MigrationResult tracks counts for the summary output.
type MigrationResult struct {
	Converted int
	Archived  int
	Failed    int
}

// SBIAConverter converts a legacy memory record to the new SBIA format.
type SBIAConverter interface {
	Convert(ctx context.Context, legacy LegacyMemoryRecord) (*memory.MemoryRecord, error)
}

// ExecuteMigration runs the SBIA migration with injected dependencies.
func ExecuteMigration(
	ctx context.Context,
	dataDir string,
	deps MigrationDeps,
) error {
	memoriesDir := memory.MemoriesDir(dataDir)
	archiveDir := filepath.Join(dataDir, "archive")

	entries, listErr := deps.ListDir(memoriesDir)
	if listErr != nil {
		return fmt.Errorf("migrate-sbia: listing memories: %w", listErr)
	}

	var result MigrationResult

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(memoriesDir, entry.Name())

		processErr := processLegacyFile(
			ctx, filePath, entry.Name(), archiveDir, deps, &result,
		)
		if processErr != nil {
			_, _ = fmt.Fprintf(
				deps.Stdout,
				"[engram] migrate-sbia: error processing %s: %s\n",
				entry.Name(), processErr,
			)
		}
	}

	_, _ = fmt.Fprintf(
		deps.Stdout,
		"[engram] migrate-sbia: %d converted, %d archived, %d failed\n",
		result.Converted, result.Archived, result.Failed,
	)

	return nil
}

// unexported constants.
const (
	archiveDirPerm                   = 0o755
	maxProjectScopedGeneralizability = 2
	situationPartsCapacity           = 3
)

// defaultSBIAConverter maps legacy fields to SBIA fields without an API call.
// Used as fallback when no API key is available or in tests.
type defaultSBIAConverter struct{}

func (c *defaultSBIAConverter) Convert(
	_ context.Context,
	legacy LegacyMemoryRecord,
) (*memory.MemoryRecord, error) {
	situation := buildSituation(legacy)
	behavior := legacy.AntiPattern
	impact := legacy.Rationale
	action := legacy.Principle

	now := time.Now().UTC().Format(time.RFC3339)

	return &memory.MemoryRecord{
		Situation: situation,
		Behavior:  behavior,
		Impact:    impact,
		Action:    action,
		UpdatedAt: now,
	}, nil
}

func archiveFile(
	filePath, fileName, archiveDir string,
	deps MigrationDeps,
	result *MigrationResult,
) error {
	mkdirErr := deps.MkdirAll(archiveDir, archiveDirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("creating archive dir: %w", mkdirErr)
	}

	destPath := filepath.Join(archiveDir, fileName)

	renameErr := deps.Rename(filePath, destPath)
	if renameErr != nil {
		return fmt.Errorf("archiving %s: %w", fileName, renameErr)
	}

	result.Archived++

	return nil
}

func buildSituation(legacy LegacyMemoryRecord) string {
	parts := make([]string, 0, situationPartsCapacity)

	if legacy.Title != "" {
		parts = append(parts, legacy.Title)
	}

	if legacy.Content != "" {
		parts = append(parts, legacy.Content)
	}

	if len(legacy.Keywords) > 0 {
		parts = append(parts, "Keywords: "+strings.Join(legacy.Keywords, ", "))
	}

	return strings.Join(parts, ". ")
}

func convertTierA(
	ctx context.Context,
	filePath, fileName, archiveDir string,
	legacy LegacyMemoryRecord,
	deps MigrationDeps,
	result *MigrationResult,
) error {
	converted, convertErr := deps.Converter.Convert(ctx, legacy)
	if convertErr != nil {
		_, _ = fmt.Fprintf(
			deps.Stdout,
			"[engram] migrate-sbia: conversion failed for %s, archiving: %s\n",
			fileName, convertErr,
		)

		archiveErr := archiveFile(filePath, fileName, archiveDir, deps, result)
		if archiveErr != nil {
			result.Failed++

			return fmt.Errorf("archiving failed conversion %s: %w", fileName, archiveErr)
		}

		result.Failed++

		return nil
	}

	// Apply counter mapping.
	converted.SurfacedCount = legacy.SurfacedCount
	converted.FollowedCount = legacy.FollowedCount
	converted.NotFollowedCount = legacy.ContradictedCount + legacy.IgnoredCount
	converted.IrrelevantCount = legacy.IrrelevantCount

	// Apply scope mapping.
	converted.ProjectSlug = legacy.ProjectSlug
	converted.ProjectScoped = legacy.Generalizability <= maxProjectScopedGeneralizability

	// Preserve timestamps.
	converted.CreatedAt = legacy.CreatedAt
	converted.UpdatedAt = legacy.UpdatedAt

	writeErr := writeMemoryRecord(filePath, converted, deps)
	if writeErr != nil {
		result.Failed++

		return fmt.Errorf("writing converted %s: %w", fileName, writeErr)
	}

	result.Converted++

	return nil
}

func newDefaultSBIAConverter() *defaultSBIAConverter {
	return &defaultSBIAConverter{}
}

func processLegacyFile(
	ctx context.Context,
	filePath, fileName, archiveDir string,
	deps MigrationDeps,
	result *MigrationResult,
) error {
	data, readErr := deps.ReadFile(filePath)
	if readErr != nil {
		return fmt.Errorf("reading %s: %w", fileName, readErr)
	}

	var legacy LegacyMemoryRecord

	_, decodeErr := toml.Decode(string(data), &legacy)
	if decodeErr != nil {
		return fmt.Errorf("decoding %s: %w", fileName, decodeErr)
	}

	confidence := strings.ToUpper(legacy.Confidence)

	if confidence != "A" {
		return archiveFile(filePath, fileName, archiveDir, deps, result)
	}

	return convertTierA(ctx, filePath, fileName, archiveDir, legacy, deps, result)
}

func runMigrateSBIA(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("migrate-sbia", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("migrate-sbia: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("migrate-sbia: %w", defaultErr)
	}

	converter := SBIAConverterOverride
	if converter == nil {
		converter = newDefaultSBIAConverter()
	}

	deps := MigrationDeps{
		ListDir:   os.ReadDir,
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		MkdirAll:  os.MkdirAll,
		Rename:    os.Rename,
		Converter: converter,
		Stdout:    stdout,
	}

	ctx, cancel := signalContext()
	defer cancel()

	return ExecuteMigration(ctx, *dataDir, deps)
}

func writeMemoryRecord(
	filePath string,
	record *memory.MemoryRecord,
	deps MigrationDeps,
) error {
	var buf strings.Builder

	encErr := toml.NewEncoder(&buf).Encode(record)
	if encErr != nil {
		return fmt.Errorf("encoding TOML: %w", encErr)
	}

	writeErr := deps.WriteFile(filePath, []byte(buf.String()), filePerms)
	if writeErr != nil {
		return fmt.Errorf("writing file: %w", writeErr)
	}

	return nil
}
