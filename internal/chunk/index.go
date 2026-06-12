package chunk

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Record is one embedded chunk in a source's .jsonl index file.
type Record struct {
	// Source is the originating file path (transcript or markdown).
	Source string `json:"source"`
	// Anchor locates the chunk within the source (turn-N / heading).
	Anchor string `json:"anchor"`
	// ContentHash is "sha256:<hex>" over Text; ingestion skips chunks whose
	// hash is already indexed, making re-runs idempotent.
	ContentHash string `json:"content_hash"`
	// Text is the chunk content (also what was embedded).
	Text string `json:"text"`
	// Vector is the body embedding of Text.
	Vector []float32 `json:"vector"`
}

// HashText returns the content hash for a chunk text.
func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))

	return "sha256:" + hex.EncodeToString(sum[:])
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

// DecodeRecords parses a JSONL index file. Blank lines are skipped;
// a malformed line is an error (the index is binary-owned, never hand-edited).
func DecodeRecords(data []byte) ([]Record, error) {
	var records []Record

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decoding chunk record: %w", err)
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning chunk index: %w", err)
	}

	return records, nil
}
