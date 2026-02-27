package memory

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ExtractOutput is the TOML output structure for orchestrator consumption.
type ExtractOutput struct {
	Status          string         `toml:"status"`
	FilePath        string         `toml:"file_path"`
	ItemsExtracted  int            `toml:"items_extracted"`
	Breakdown       map[string]int `toml:"breakdown"`
	StorageLocation string         `toml:"storage_location"`
	Error           string         `toml:"error,omitempty"`
}

// RunExtract extracts learnings from a result file.
func RunExtract(args ExtractArgs, homeDir string) error {
	if args.Result == "" {
		return errors.New("--result must be provided")
	}

	filePath := args.Result

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	modelDir := args.ModelDir
	if modelDir == "" {
		modelDir = filepath.Join(homeDir, ".claude", "models")
	}

	opts := ExtractOpts{
		FilePath:   filePath,
		MemoryRoot: memoryRoot,
		ModelDir:   modelDir,
	}

	result, err := opts.Extract()
	if err != nil {
		output := ExtractOutput{
			Status:   "error",
			FilePath: filePath,
			Error:    err.Error(),
		}
		printExtractOutputTOML(output)

		return fmt.Errorf("extract failed: %w", err)
	}

	breakdown := make(map[string]int)
	for _, item := range result.Items {
		breakdown[item.Type]++
	}

	storageLocation := filepath.Join(memoryRoot, "embeddings.db")

	output := ExtractOutput{
		Status:          result.Status,
		FilePath:        result.FilePath,
		ItemsExtracted:  result.ItemsExtracted,
		Breakdown:       breakdown,
		StorageLocation: storageLocation,
	}
	printExtractOutputTOML(output)
	printExtractTerminalOutput(result, breakdown, storageLocation)

	return nil
}

func printExtractOutputTOML(output ExtractOutput) {
	var buf bytes.Buffer
	buf.WriteString("[result]\n")
	encoder := toml.NewEncoder(&buf)
	_ = encoder.Encode(output)

	fmt.Print(buf.String())
}

func printExtractTerminalOutput(result *ExtractResult, breakdown map[string]int, storageLocation string) {
	filename := filepath.Base(result.FilePath)
	fmt.Fprintf(os.Stderr, "\n\u2713 Extracted %d items from %s\n", result.ItemsExtracted, filename)

	if len(breakdown) > 0 {
		fmt.Fprintln(os.Stderr, "Breakdown:")

		for itemType, count := range breakdown {
			fmt.Fprintf(os.Stderr, "  - %d %ss\n", count, itemType)
		}
	}

	fmt.Fprintf(os.Stderr, "\nStored in semantic memory (%s)\n", storageLocation)
}
