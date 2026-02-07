package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/memory"
)

type memoryExtractArgs struct {
	Result     string `targ:"flag,short=r,desc=Path to result.toml file"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	ModelDir   string `targ:"flag,desc=Model directory (defaults to ~/.claude/models)"`
}

// ExtractOutput is the TOML output structure for orchestrator consumption.
type ExtractOutput struct {
	Status         string            `toml:"status"`
	FilePath       string            `toml:"file_path"`
	ItemsExtracted int               `toml:"items_extracted"`
	Breakdown      map[string]int    `toml:"breakdown"`
	StorageLocation string           `toml:"storage_location"`
	Error          string            `toml:"error,omitempty"`
}

func memoryExtract(args memoryExtractArgs) error {
	// Validate that result file is provided
	if args.Result == "" {
		fmt.Fprintln(os.Stderr, "Error: --result must be provided")
		os.Exit(1)
	}

	filePath := args.Result

	// Set up memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Set up model directory
	modelDir := args.ModelDir
	if modelDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		modelDir = filepath.Join(home, ".claude", "models")
	}

	// Create extract options
	opts := memory.ExtractOpts{
		FilePath:   filePath,
		MemoryRoot: memoryRoot,
		ModelDir:   modelDir,
	}

	// Execute extraction
	result, err := opts.Extract()
	if err != nil {
		// Output error in TOML format
		output := ExtractOutput{
			Status:   "error",
			FilePath: filePath,
			Error:    err.Error(),
		}
		outputTOML(output)

		// Also print human-readable error
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Calculate breakdown by type
	breakdown := make(map[string]int)
	for _, item := range result.Items {
		breakdown[item.Type]++
	}

	// Build storage location
	storageLocation := filepath.Join(memoryRoot, "embeddings.db")

	// Output TOML for orchestrator
	output := ExtractOutput{
		Status:         result.Status,
		FilePath:       result.FilePath,
		ItemsExtracted: result.ItemsExtracted,
		Breakdown:      breakdown,
		StorageLocation: storageLocation,
	}
	outputTOML(output)

	// Print human-readable terminal output
	printTerminalOutput(result, breakdown, storageLocation)

	return nil
}

func outputTOML(output ExtractOutput) {
	var buf bytes.Buffer
	buf.WriteString("[result]\n")
	encoder := toml.NewEncoder(&buf)
	_ = encoder.Encode(output)
	fmt.Print(buf.String())
}

func printTerminalOutput(result *memory.ExtractResult, breakdown map[string]int, storageLocation string) {
	// Success message with item count (to stderr so stdout is pure TOML)
	filename := filepath.Base(result.FilePath)
	fmt.Fprintf(os.Stderr, "\n\u2713 Extracted %d items from %s\n", result.ItemsExtracted, filename)

	// Item breakdown
	if len(breakdown) > 0 {
		fmt.Fprintln(os.Stderr, "Breakdown:")
		for itemType, count := range breakdown {
			fmt.Fprintf(os.Stderr, "  - %d %ss\n", count, itemType)
		}
	}

	// Storage location
	fmt.Fprintf(os.Stderr, "\nStored in semantic memory (%s)\n", storageLocation)
}
