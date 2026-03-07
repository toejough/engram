package learn

// Offset holds the byte offset and session ID for incremental learning.
type Offset struct {
	Offset    int64  `json:"offset"`
	SessionID string `json:"session_id"` //nolint:tagliatelle // JSON convention
}

// OffsetStore reads and writes learn offset state.
type OffsetStore interface {
	Read(path string) (Offset, error)
	Write(path string, offset Offset) error
}
