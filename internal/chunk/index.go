package chunk

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Record is one embedded chunk in a source's .jsonl index file.
type Record struct {
	// Source is the originating file path (transcript or markdown).
	Source string `json:"source"`
	// Anchor locates the chunk within the source (turn-N / heading).
	Anchor string `json:"anchor"`
	// ContentHash is "sha256:<hex>" over Text; ingestion skips chunks whose
	// hash is already indexed, making re-runs idempotent.
	ContentHash string `json:"content_hash"` //nolint:tagliatelle // index schema uses snake_case like .vec.json
	// Text is the chunk content (also what was embedded).
	Text string `json:"text"`
	// Vector is the body embedding of Text.
	Vector []float32 `json:"vector"`
	// IngestedAt is the wall-clock time this chunk was first written to the
	// index. Zero for legacy records (pre-D5); backfilled on first merge.
	IngestedAt time.Time `json:"ingested_at,omitzero"` //nolint:tagliatelle // index schema uses snake_case like .vec.json
}

// DecodeRecords parses a JSONL index file. Blank lines are skipped;
// a malformed line is an error (the index is binary-owned, never hand-edited).
func DecodeRecords(data []byte) ([]Record, error) {
	var records []Record

	const (
		scanInitialBuf = 64 * 1024
		scanMaxBuf     = 4 * 1024 * 1024
	)

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, scanInitialBuf), scanMaxBuf)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record Record

		err := json.Unmarshal([]byte(line), &record)
		if err != nil {
			return nil, fmt.Errorf("decoding chunk record: %w", err)
		}

		records = append(records, record)
	}

	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("scanning chunk index: %w", err)
	}

	return records, nil
}

// EncodeRecords renders records as JSONL (one compact JSON object per line).
func EncodeRecords(records []Record) ([]byte, error) {
	var buf strings.Builder

	for _, r := range records {
		line, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("encoding chunk record: %w", err)
		}

		buf.Write(line)
		buf.WriteString("\n")
	}

	return []byte(buf.String()), nil
}

// HashText returns the content hash for a chunk text.
func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))

	return "sha256:" + hex.EncodeToString(sum[:])
}
