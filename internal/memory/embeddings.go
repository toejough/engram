package memory

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

// embeddingDim is the dimension of the all-MiniLM-L6-v2 embeddings
const embeddingDim = 384

func init() {
	// Auto-register sqlite-vec extension
	sqlite_vec.Auto()
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

// generateMockEmbedding generates a deterministic mock embedding for text.
// This is a stub implementation - real code would use ONNX model.
func generateMockEmbedding(text string) []float32 {
	// Create deterministic embedding based on text content
	embedding := make([]float32, embeddingDim)

	// Use simple hash-based approach for determinism
	words := strings.Fields(strings.ToLower(text))
	for i, word := range words {
		hash := hashString(word)
		for j := 0; j < embeddingDim; j++ {
			// Mix word position and character hash into embedding
			embedding[j] += float32(hash+i*13) / 1000000.0
		}
	}

	// Normalize to unit vector
	var magnitude float32
	for _, v := range embedding {
		magnitude += v * v
	}
	magnitude = float32(1.0 / (math.Sqrt(float64(magnitude)) + 0.0001))
	for i := range embedding {
		embedding[i] *= magnitude
	}

	return embedding
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

// createEmbeddings processes content and creates embeddings.
func createEmbeddings(db *sql.DB, contents []string) (int, error) {
	existing, err := getExistingEmbeddings(db)
	if err != nil {
		return 0, err
	}

	newCount := 0

	for _, content := range contents {
		// Skip if already embedded
		if _, exists := existing[content]; exists {
			continue
		}

		// Generate embedding (mock for now)
		embedding := generateMockEmbedding(content)

		// Insert into vec table using sqlite-vec SerializeFloat32
		vecStmt := `INSERT INTO vec_embeddings(embedding) VALUES (?)`
		embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
		if err != nil {
			return newCount, err
		}
		result, err := db.Exec(vecStmt, embeddingBlob)
		if err != nil {
			return newCount, err
		}

		embeddingID, _ := result.LastInsertId()

		// Insert into metadata table
		metaStmt := `INSERT INTO embeddings(content, source, embedding_id) VALUES (?, ?, ?)`
		if _, err := db.Exec(metaStmt, content, "memory", embeddingID); err != nil {
			return newCount, err
		}

		newCount++
	}

	return newCount, nil
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

