package memory

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// MemoryStoreSemanticMatcher implements SemanticMatcher using the local memory store.
type MemoryStoreSemanticMatcher struct {
	MemoryRoot     string
	ModelDir       string
	HTTPClient     *http.Client // HTTP client for downloads (default: http.DefaultClient)
	VectorEmbedder Embedder     // Injected embedder; bypasses ONNX init/download when set
}

// NewMemoryStoreSemanticMatcher creates a new MemoryStoreSemanticMatcher.
func NewMemoryStoreSemanticMatcher(memoryRoot string) *MemoryStoreSemanticMatcher {
	return &MemoryStoreSemanticMatcher{MemoryRoot: memoryRoot}
}

// FindSimilarMemories queries the memory store for semantically similar memories.
func (m *MemoryStoreSemanticMatcher) FindSimilarMemories(text string, threshold float64, limit int) ([]string, error) {
	results, err := Query(QueryOpts{
		Text:       text,
		Limit:      limit,
		MemoryRoot: m.MemoryRoot,
		ModelDir:   m.ModelDir,
	})
	if err != nil {
		return nil, err
	}

	if results == nil || len(results.Results) == 0 {
		return nil, nil
	}

	var matches []string

	for _, r := range results.Results {
		if r.Score >= threshold {
			matches = append(matches, r.Content)
		}
	}

	if len(matches) > limit {
		matches = matches[:limit]
	}

	if len(matches) == 0 {
		return nil, nil
	}

	return matches, nil
}

// FindSimilarMemoriesBatch queries multiple texts in a single batch, sharing DB and model setup.
// Returns a slice parallel to texts where each entry is the matching memories (or nil).
func (m *MemoryStoreSemanticMatcher) FindSimilarMemoriesBatch(texts []string, threshold float64, limit int) ([][]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var embedder Embedder

	if m.VectorEmbedder != nil {
		// Injected embedder path — no ONNX init or model downloads.
		embedder = m.VectorEmbedder
	} else {
		// ONNX runtime path.
		client := m.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}

		// Determine model directory (done once)
		modelDir := m.ModelDir
		if modelDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			modelDir = filepath.Join(homeDir, ".claude", "models")
		}

		if err := os.MkdirAll(modelDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create model directory: %w", err)
		}

		// Initialize ONNX Runtime (once)
		if err := initializeONNXRuntimeWithClient(modelDir, client); err != nil {
			return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
		}

		modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			if err := downloadModel(modelPath, client); err != nil {
				return nil, fmt.Errorf("failed to download model: %w", err)
			}
		}

		embedder = &onnxEmbedder{modelPath: modelPath}
	}

	// Open DB (once)
	dbPath := filepath.Join(m.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embeddings database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Model checks only when using ONNX.
	if m.VectorEmbedder == nil {
		client := m.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}

		modelDir := m.ModelDir
		if modelDir == "" {
			if homeDir, err := os.UserHomeDir(); err == nil {
				modelDir = filepath.Join(homeDir, ".claude", "models")
			}
		}

		modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")

		needsDownload, err := ensureCorrectModel(db, modelPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check model validity: %w", err)
		}

		if needsDownload {
			if err := downloadModel(modelPath, client); err != nil {
				return nil, fmt.Errorf("failed to download model: %w", err)
			}
		}

		if err := migrateModelVersion(db, embedder); err != nil {
			return nil, fmt.Errorf("failed to migrate model version: %w", err)
		}
	}

	// Process each text: embed + search (DB stays open)
	results := make([][]string, len(texts))
	for i, text := range texts {
		matches, err := querySingleWithDB(db, text, embedder, threshold, limit)
		if err != nil {
			continue // skip failures, leave nil
		}

		results[i] = matches
	}

	return results, nil
}

// querySingleWithDB runs a single embedding+search using an already-open DB.
func querySingleWithDB(db *sql.DB, text string, embed Embedder, threshold float64, limit int) ([]string, error) {
	queryEmbedding, err := embed.Embed("query: " + text)
	if err != nil {
		return nil, err
	}

	searchResults, err := hybridSearch(db, queryEmbedding, text, limit, 60)
	if err != nil {
		return nil, err
	}

	var matches []string

	for _, r := range searchResults {
		if r.Score >= threshold {
			matches = append(matches, r.Content)
		}
	}

	if len(matches) > limit {
		matches = matches[:limit]
	}

	if len(matches) == 0 {
		return nil, nil
	}

	return matches, nil
}
