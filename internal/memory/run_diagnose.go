package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunDiagnose diagnoses memory issues (leeches).
func RunDiagnose(args DiagnoseArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	db, err := InitDBForTest(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	var memoryIDs []int64
	if args.ID != 0 {
		memoryIDs = []int64{args.ID}
	} else {
		leeches, err := GetLeeches(db)
		if err != nil {
			return fmt.Errorf("failed to get leeches: %w", err)
		}

		if len(leeches) == 0 {
			fmt.Println("No leech memories found.")
			return nil
		}

		for _, l := range leeches {
			memoryIDs = append(memoryIDs, l.MemoryID)
		}

		fmt.Printf("Found %d leech memory/memories to diagnose.\n\n", len(memoryIDs))
	}

	var llm LLMExtractor
	if !args.NoLLM {
		llm = NewLLMExtractor()
	}

	var (
		recommendations []*Recommendation
		output          strings.Builder
	)

	for _, id := range memoryIDs {
		diag, err := DiagnoseLeech(db, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: diagnose memory %d: %v\n", id, err)
			continue
		}

		if diag.DiagnosisType == "content_quality" && llm != nil {
			preview, previewErr := PreviewLeechRewrite(db, *diag, llm)
			if previewErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: preview rewrite for memory %d: %v\n", id, previewErr)
			} else {
				diag.SuggestedContent = preview
			}
		}

		output.WriteString(FormatLeechDiagnosis(diag))
		output.WriteString("\n")

		if diag.Recommendation != nil {
			recommendations = append(recommendations, diag.Recommendation)
		}
	}

	fmt.Print(output.String())

	if len(recommendations) > 0 && !args.NoSave {
		filename := fmt.Sprintf("memory-recommendations-%s.md", time.Now().Format("2006-01-02"))
		filePath := filepath.Join(memoryRoot, filename)

		if err := saveLeechRecommendations(filePath, recommendations); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save recommendations: %v\n", err)
		} else {
			fmt.Printf("Recommendations saved to: %s\n", filePath)
		}
	}

	return nil
}

func saveLeechRecommendations(path string, recommendations []*Recommendation) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Memory Recommendations\n\nGenerated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	for i, rec := range recommendations {
		fmt.Fprintf(&sb, "## Recommendation %d: %s\n\n", i+1, rec.Category)

		if rec.Description != "" {
			sb.WriteString("**Action**: " + rec.Description + "\n\n")
		}

		if rec.Evidence != "" {
			sb.WriteString("**Evidence**: " + rec.Evidence + "\n\n")
		}

		if rec.Text != "" {
			sb.WriteString(rec.Text + "\n\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
