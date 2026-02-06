package memory

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
	ort "github.com/yalue/onnxruntime_go"
)

// embeddingDim is the dimension of the e5-small-v2 embeddings
const embeddingDim = 384

// e5SmallModelURL is the HuggingFace URL for the e5-small-v2 ONNX model
// Note: Using all-MiniLM-L6-v2 as a compatible alternative with same 384 dimensions
const e5SmallModelURL = "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx"

// onnxRuntimeVersion is the version of ONNX Runtime to download
const onnxRuntimeVersion = "1.23.2"

var onnxRuntimeInitialized bool

// Session cache for ISSUE-48: Cache ONNX sessions across test functions
var (
	sessionCache     map[string]*ort.DynamicAdvancedSession
	sessionCacheMu   sync.RWMutex
	sessionInitCount int
	sessionInitMu    sync.Mutex
	sessionOnce      map[string]*sync.Once
	sessionOnceMu    sync.Mutex
	modelDownloadMu  sync.Mutex
)

func init() {
	// Auto-register sqlite-vec extension
	sqlite_vec.Auto()
	// ONNX Runtime will be initialized on first use

	// Initialize session cache
	sessionCache = make(map[string]*ort.DynamicAdvancedSession)
	sessionOnce = make(map[string]*sync.Once)
}

// GetSessionInitCount returns the number of times sessions have been initialized.
// This is a test-only function for verifying caching behavior.
func GetSessionInitCount() int {
	sessionInitMu.Lock()
	defer sessionInitMu.Unlock()
	return sessionInitCount
}

// ClearSessionCache clears the session cache and destroys all cached sessions.
// This is a test-only function for test isolation.
func ClearSessionCache() {
	sessionCacheMu.Lock()
	defer sessionCacheMu.Unlock()

	// Destroy all cached sessions
	for _, session := range sessionCache {
		_ = session.Destroy()
	}

	// Clear the caches
	sessionCache = make(map[string]*ort.DynamicAdvancedSession)

	sessionOnceMu.Lock()
	sessionOnce = make(map[string]*sync.Once)
	sessionOnceMu.Unlock()
}

// initEmbeddingsDB initializes the embeddings database with sqlite-vec.
func initEmbeddingsDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create embeddings table with vec0 virtual table
	createTable := `
		CREATE TABLE IF NOT EXISTS embeddings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			source TEXT NOT NULL,
			embedding_id INTEGER
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS vec_embeddings USING vec0(
			embedding FLOAT[384]
		);
	`

	if _, err := db.Exec(createTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// getExistingEmbeddings returns a map of content -> embedding_id for already embedded content.
func getExistingEmbeddings(db *sql.DB) (map[string]int64, error) {
	rows, err := db.Query("SELECT content, embedding_id FROM embeddings WHERE embedding_id IS NOT NULL")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	existing := make(map[string]int64)
	for rows.Next() {
		var content string
		var embeddingID int64
		if err := rows.Scan(&content, &embeddingID); err != nil {
			return nil, err
		}
		existing[content] = embeddingID
	}

	return existing, rows.Err()
}

// hashString provides a simple hash for word->token mapping.
func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// downloadONNXRuntime downloads and extracts the ONNX Runtime library.
func downloadONNXRuntime(modelDir string) (string, error) {
	// Determine OS-specific library name and download URL
	var libName, archiveURL string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			libName = "libonnxruntime.dylib"
			archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-osx-arm64-%s.tgz", onnxRuntimeVersion, onnxRuntimeVersion)
		} else {
			libName = "libonnxruntime.dylib"
			archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-osx-x86_64-%s.tgz", onnxRuntimeVersion, onnxRuntimeVersion)
		}
	case "linux":
		libName = "libonnxruntime.so"
		archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-linux-x64-%s.tgz", onnxRuntimeVersion, onnxRuntimeVersion)
	case "windows":
		libName = "onnxruntime.dll"
		archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-win-x64-%s.zip", onnxRuntimeVersion, onnxRuntimeVersion)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	libPath := filepath.Join(modelDir, libName)

	// Check if library already exists
	if _, err := os.Stat(libPath); err == nil {
		return libPath, nil
	}

	// Download the archive
	resp, err := http.Get(archiveURL)
	if err != nil {
		return "", fmt.Errorf("failed to download ONNX Runtime: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download ONNX Runtime: HTTP %d", resp.StatusCode)
	}

	// Extract the library from the archive
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("windows ZIP extraction not implemented")
	}

	// Extract from tar.gz
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for the versioned library file (e.g., libonnxruntime.1.23.2.dylib)
		// Skip symlinks and only extract the actual file
		baseName := filepath.Base(header.Name)
		if strings.Contains(header.Name, "/lib/") &&
			strings.HasPrefix(baseName, "libonnxruntime") &&
			strings.HasSuffix(baseName, ".dylib") &&
			header.Typeflag != tar.TypeSymlink &&
			header.Typeflag != tar.TypeDir {

			outFile, err := os.Create(libPath)
			if err != nil {
				return "", fmt.Errorf("failed to create library file: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return "", fmt.Errorf("failed to extract library: %w", err)
			}

			_ = outFile.Close()

			// Make executable on Unix-like systems
			if runtime.GOOS != "windows" {
				_ = os.Chmod(libPath, 0755)
			}

			return libPath, nil
		}
	}

	return "", fmt.Errorf("library not found in archive")
}

// downloadModel downloads the e5-small-v2 model from HuggingFace if not already present.
// Thread-safe: uses mutex to prevent concurrent downloads.
func downloadModel(modelPath string) error {
	// Fast path: check without lock
	if _, err := os.Stat(modelPath); err == nil {
		return nil // Model already exists
	}

	// Acquire lock for download
	modelDownloadMu.Lock()
	defer modelDownloadMu.Unlock()

	// Check again after acquiring lock (another goroutine may have downloaded)
	if _, err := os.Stat(modelPath); err == nil {
		return nil // Model already exists
	}

	// Create model directory
	modelDir := filepath.Dir(modelPath)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Download the model
	resp, err := http.Get(e5SmallModelURL)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tempPath := modelPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Copy model data
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to save model: %w", err)
	}

	// Close the file before renaming (required on Windows)
	_ = out.Close()

	// Rename temp file to final path
	if err := os.Rename(tempPath, modelPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to finalize model: %w", err)
	}

	return nil
}

// initializeONNXRuntime initializes the ONNX Runtime environment with the downloaded library.
func initializeONNXRuntime(modelDir string) error {
	if onnxRuntimeInitialized {
		return nil
	}

	// Download ONNX Runtime library
	libPath, err := downloadONNXRuntime(modelDir)
	if err != nil {
		return fmt.Errorf("failed to download ONNX Runtime: %w", err)
	}

	// Set the library path
	ort.SetSharedLibraryPath(libPath)

	// Initialize the environment
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	onnxRuntimeInitialized = true
	return nil
}

// getOrCreateSession retrieves a cached session or creates a new one.
// Returns the session and whether it was newly created in THIS call.
func getOrCreateSession(modelPath string) (*ort.DynamicAdvancedSession, bool, error) {
	// Try read lock first for fast path (session already exists)
	sessionCacheMu.RLock()
	if session, exists := sessionCache[modelPath]; exists {
		sessionCacheMu.RUnlock()
		return session, false, nil
	}
	sessionCacheMu.RUnlock()

	// Get or create the sync.Once for this model path
	sessionOnceMu.Lock()
	once, exists := sessionOnce[modelPath]
	if !exists {
		once = &sync.Once{}
		sessionOnce[modelPath] = once
	}
	sessionOnceMu.Unlock()

	// Track if we actually created the session in THIS goroutine
	var session *ort.DynamicAdvancedSession
	var err error
	createdInThisCall := false

	once.Do(func() {
		session, err = ort.NewDynamicAdvancedSession(modelPath,
			[]string{"input_ids", "attention_mask", "token_type_ids"},
			[]string{"last_hidden_state"},
			nil)
		if err != nil {
			return
		}

		// Store in cache
		sessionCacheMu.Lock()
		sessionCache[modelPath] = session
		sessionCacheMu.Unlock()

		// Increment init count
		sessionInitMu.Lock()
		sessionInitCount++
		sessionInitMu.Unlock()

		createdInThisCall = true
	})

	if err != nil {
		return nil, false, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	// If we didn't create it in this call, retrieve from cache
	if !createdInThisCall {
		sessionCacheMu.RLock()
		session = sessionCache[modelPath]
		sessionCacheMu.RUnlock()
	}

	return session, createdInThisCall, nil
}

// generateEmbeddingONNX generates an embedding using the e5-small-v2 ONNX model.
// Returns the embedding, whether a new session was created, and whether a session was reused.
func generateEmbeddingONNX(text string, modelPath string) ([]float32, bool, bool, error) {
	// Simple tokenization (this is a placeholder - real implementation would use proper tokenizer)
	// For now, we'll use a basic word-based approach
	words := strings.Fields(strings.ToLower(text))

	// Create a simple input representation (this is simplified)
	// Real e5-small uses BERT tokenization, but for testing we'll use a simpler approach
	inputSize := 128 // Max sequence length
	inputIDs := make([]int64, inputSize)
	attentionMask := make([]int64, inputSize)
	tokenTypeIDs := make([]int64, inputSize) // All zeros for single-sequence input

	// Fill input IDs with word hashes (simplified tokenization)
	for i, word := range words {
		if i >= inputSize {
			break
		}
		inputIDs[i] = int64(hashString(word) % 30000) // Vocab size approximation
		attentionMask[i] = 1                          // Mark as valid token
		tokenTypeIDs[i] = 0                           // All zeros for single sequence
	}

	// Get or create cached session
	session, sessionCreated, err := getOrCreateSession(modelPath)
	if err != nil {
		return nil, false, false, err
	}
	// Don't destroy the session - it's cached for reuse

	// Create input tensors
	inputShape := ort.NewShape(1, int64(inputSize))
	inputIDsTensor, err := ort.NewTensor(inputShape, inputIDs)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer func() { _ = inputIDsTensor.Destroy() }()

	attentionMaskTensor, err := ort.NewTensor(inputShape, attentionMask)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer func() { _ = attentionMaskTensor.Destroy() }()

	tokenTypeIDsTensor, err := ort.NewTensor(inputShape, tokenTypeIDs)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}
	defer func() { _ = tokenTypeIDsTensor.Destroy() }()

	// Create output tensor
	outputShape := ort.NewShape(1, int64(inputSize), int64(embeddingDim))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer func() { _ = outputTensor.Destroy() }()

	// Run inference
	inputs := []ort.Value{inputIDsTensor, attentionMaskTensor, tokenTypeIDsTensor}
	outputs := []ort.Value{outputTensor}
	if err := session.Run(inputs, outputs); err != nil {
		return nil, false, false, fmt.Errorf("failed to run inference: %w", err)
	}

	// Extract embeddings from output
	// The output is [batch_size, sequence_length, hidden_size]
	// We need to pool it to get [batch_size, hidden_size]
	outputData := outputTensor.GetData()

	// Mean pooling over sequence dimension
	embedding := make([]float32, embeddingDim)

	// Simple mean pooling
	validTokens := 0
	for i := 0; i < inputSize; i++ {
		if attentionMask[i] != 0 {
			validTokens++
			for j := 0; j < embeddingDim; j++ {
				idx := i*embeddingDim + j
				if idx < len(outputData) {
					embedding[j] += outputData[idx]
				}
			}
		}
	}

	// Average the pooled values
	if validTokens > 0 {
		for j := 0; j < embeddingDim; j++ {
			embedding[j] /= float32(validTokens)
		}
	}

	// Normalize to unit vector
	var magnitude float32
	for _, v := range embedding {
		magnitude += v * v
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))
	if magnitude > 0 {
		for i := range embedding {
			embedding[i] /= magnitude
		}
	}

	sessionReused := !sessionCreated

	return embedding, sessionCreated, sessionReused, nil
}

// createEmbeddings processes content and creates embeddings using ONNX model.
// Returns the number of new embeddings created, whether a session was created, and whether it was reused.
func createEmbeddings(db *sql.DB, contents []string, modelPath string) (int, bool, bool, error) {
	existing, err := getExistingEmbeddings(db)
	if err != nil {
		return 0, false, false, err
	}

	newCount := 0
	var sessionCreated, sessionReused bool

	for _, content := range contents {
		// Skip if already embedded
		if _, exists := existing[content]; exists {
			continue
		}

		// Generate embedding using ONNX model
		embedding, created, reused, err := generateEmbeddingONNX(content, modelPath)
		if err != nil {
			return newCount, sessionCreated, sessionReused, fmt.Errorf("failed to generate embedding: %w", err)
		}

		// Track session status from first call
		if newCount == 0 {
			sessionCreated = created
			sessionReused = reused
		}

		// Insert into vec table using sqlite-vec SerializeFloat32
		vecStmt := `INSERT INTO vec_embeddings(embedding) VALUES (?)`
		embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
		if err != nil {
			return newCount, sessionCreated, sessionReused, err
		}
		result, err := db.Exec(vecStmt, embeddingBlob)
		if err != nil {
			return newCount, sessionCreated, sessionReused, err
		}

		embeddingID, _ := result.LastInsertId()

		// Insert into metadata table
		metaStmt := `INSERT INTO embeddings(content, source, embedding_id) VALUES (?, ?, ?)`
		if _, err := db.Exec(metaStmt, content, "memory", embeddingID); err != nil {
			return newCount, sessionCreated, sessionReused, err
		}

		newCount++
	}

	return newCount, sessionCreated, sessionReused, nil
}

// searchSimilar finds the most similar embeddings using cosine similarity.
func searchSimilar(db *sql.DB, queryEmbedding []float32, limit int) ([]QueryResult, error) {
	// Use sqlite-vec's distance function for similarity search
	query := `
		SELECT e.content, e.source,
		       (1 - vec_distance_cosine(v.embedding, ?)) as score
		FROM vec_embeddings v
		JOIN embeddings e ON e.embedding_id = v.rowid
		ORDER BY score DESC
		LIMIT ?
	`

	queryBlob, err := sqlite_vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, queryBlob, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []QueryResult
	for rows.Next() {
		var r QueryResult
		if err := rows.Scan(&r.Content, &r.Source, &r.Score); err != nil {
			return nil, err
		}
		// Clamp score to [0, 1] due to floating point precision
		if r.Score > 1.0 {
			r.Score = 1.0
		}
		if r.Score < 0.0 {
			r.Score = 0.0
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

