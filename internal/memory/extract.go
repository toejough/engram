package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// Extract parses a yield or result TOML file, extracts decisions and learnings,
// generates embeddings via ONNX, and stores them in SQLite-vec.
//
// It automatically detects whether the file is a yield or result file based on content.
// For yield files, it extracts summary, findings, and learnings from the payload.
// For result files, it extracts decisions from the decisions array.
//
// The source field is set to "yield:{filename}" or "result:{filename}" for traceability.
func (opts ExtractOpts) Extract() (*ExtractResult, error) {
	// Use injected ReadFile or default to os.ReadFile
	readFile := opts.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}

	// Read the file
	data, err := readFile(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", opts.FilePath, err)
	}

	// Determine file type and parse accordingly
	var items []ExtractedItem
	var fileType string

	// Try parsing as yield file first
	yieldFile, yieldErr := ParseYieldFile(data)
	if yieldErr == nil {
		fileType = "yield"
		items = extractFromYield(yieldFile)
	} else {
		// Try parsing as result file
		resultFile, resultErr := ParseResultFile(data)
		if resultErr == nil {
			fileType = "result"
			items = extractFromResult(resultFile)
		} else {
			// Both failed, return the first error with context
			return nil, fmt.Errorf("failed to parse file %s: %w", opts.FilePath, yieldErr)
		}
	}

	// Set source field for all items
	filename := filepath.Base(opts.FilePath)
	source := fmt.Sprintf("%s:%s", fileType, filename)
	for i := range items {
		items[i].Source = source
	}

	// Write to database if there are items to store
	if len(items) > 0 {
		if opts.WriteDB != nil {
			// Use injected WriteDB for testing
			if err := opts.WriteDB(items); err != nil {
				return nil, fmt.Errorf("failed to write to database: %w", err)
			}
		} else {
			// Use real embedding infrastructure
			if err := storeExtractedItems(opts, items); err != nil {
				return nil, fmt.Errorf("failed to store extracted items: %w", err)
			}
		}
	}

	return &ExtractResult{
		Status:         "success",
		FilePath:       opts.FilePath,
		ItemsExtracted: len(items),
		Items:          items,
	}, nil
}

// extractFromYield extracts items from a yield file's payload.
// It looks for summary, findings, and learnings fields.
func extractFromYield(yf *YieldFile) []ExtractedItem {
	var items []ExtractedItem

	// Build context string from the context section
	contextStr := buildContextString(yf.Context)

	// Extract summary
	if summary, ok := yf.Payload["summary"].(string); ok && summary != "" {
		items = append(items, ExtractedItem{
			Type:    "summary",
			Context: contextStr,
			Content: summary,
		})
	}

	// Extract findings (array of strings)
	if findings, ok := yf.Payload["findings"].([]interface{}); ok {
		for _, f := range findings {
			if finding, ok := f.(string); ok && finding != "" {
				items = append(items, ExtractedItem{
					Type:    "finding",
					Context: contextStr,
					Content: finding,
				})
			}
		}
	}

	// Extract learnings (array of strings)
	if learnings, ok := yf.Payload["learnings"].([]interface{}); ok {
		for _, l := range learnings {
			if learning, ok := l.(string); ok && learning != "" {
				items = append(items, ExtractedItem{
					Type:    "learning",
					Context: contextStr,
					Content: learning,
				})
			}
		}
	}

	return items
}

// extractFromResult extracts decisions from a result file.
func extractFromResult(rf *ResultFile) []ExtractedItem {
	var items []ExtractedItem

	for _, decision := range rf.Decisions {
		// Build content from decision fields
		var contentParts []string
		contentParts = append(contentParts, fmt.Sprintf("Choice: %s", decision.Choice))
		contentParts = append(contentParts, fmt.Sprintf("Reason: %s", decision.Reason))
		if len(decision.Alternatives) > 0 {
			contentParts = append(contentParts, fmt.Sprintf("Alternatives: %s", strings.Join(decision.Alternatives, ", ")))
		}

		items = append(items, ExtractedItem{
			Type:    "decision",
			Context: decision.Context,
			Content: strings.Join(contentParts, "; "),
		})
	}

	return items
}

// buildContextString creates a readable context string from the context section.
func buildContextString(ctx ContextSection) string {
	var parts []string
	if ctx.Phase != "" {
		parts = append(parts, ctx.Phase)
	}
	if ctx.Subphase != "" {
		parts = append(parts, ctx.Subphase)
	}
	if ctx.Task != "" {
		parts = append(parts, ctx.Task)
	}
	return strings.Join(parts, "/")
}

// storeExtractedItems stores extracted items in the SQLite-vec database using embeddings.
func storeExtractedItems(opts ExtractOpts, items []ExtractedItem) error {
	// Ensure memory directory exists
	if err := os.MkdirAll(opts.MemoryRoot, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	// Determine model directory
	modelDir := opts.ModelDir
	if modelDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		modelDir = filepath.Join(homeDir, ".claude", "models")
	}

	// Ensure model directory exists
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Initialize ONNX Runtime
	if err := initializeONNXRuntime(modelDir); err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Model path
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")

	// Download model if needed
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Initialize embeddings database
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize embeddings database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Store each item with embedding
	for _, item := range items {
		// Combine type, context, and content for embedding
		textForEmbedding := fmt.Sprintf("%s: %s - %s", item.Type, item.Context, item.Content)

		// Generate embedding
		embedding, _, _, err := generateEmbeddingONNX(textForEmbedding, modelPath)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}

		// Store in database with source field
		if err := storeItemWithEmbeddingDB(db, item, embedding); err != nil {
			return fmt.Errorf("failed to store item: %w", err)
		}
	}

	return nil
}

// storeItemWithEmbeddingDB stores a single item with its embedding in the database.
func storeItemWithEmbeddingDB(db *sql.DB, item ExtractedItem, embedding []float32) error {
	// Insert into vec table using sqlite-vec SerializeFloat32
	vecStmt := `INSERT INTO vec_embeddings(embedding) VALUES (?)`
	embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return err
	}
	result, err := db.Exec(vecStmt, embeddingBlob)
	if err != nil {
		return err
	}

	embeddingID, _ := result.LastInsertId()

	// Insert into metadata table with source field
	metaStmt := `INSERT INTO embeddings(content, source, embedding_id) VALUES (?, ?, ?)`
	content := fmt.Sprintf("[%s] %s: %s", item.Type, item.Context, item.Content)
	if _, err := db.Exec(metaStmt, content, item.Source, embeddingID); err != nil {
		return err
	}

	return nil
}
