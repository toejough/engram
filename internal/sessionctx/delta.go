package sessionctx

import (
	"strings"
)

// DeltaReader reads new lines from a transcript file since a given byte offset.
type DeltaReader struct {
	reader FileReader
}

// NewDeltaReader creates a DeltaReader with the given file reader.
func NewDeltaReader(reader FileReader) *DeltaReader {
	return &DeltaReader{reader: reader}
}

// Read returns lines added to the file since the given byte offset.
// If the file is shorter than offset (e.g. log rotation), it resets to 0.
// Returns the new lines and the updated byte offset.
func (d *DeltaReader) Read(path string, offset int64) ([]string, int64, error) {
	content, err := d.reader.Read(path)
	if err != nil {
		return nil, 0, err
	}

	fileLen := int64(len(content))

	if fileLen == 0 {
		return nil, 0, nil
	}

	// Reset if file is shorter than stored offset (rotation).
	if offset > fileLen {
		offset = 0
	}

	tail := string(content[offset:])
	if tail == "" {
		return nil, fileLen, nil
	}

	raw := strings.Split(tail, "\n")

	lines := make([]string, 0, len(raw))

	for _, line := range raw {
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines, fileLen, nil
}
