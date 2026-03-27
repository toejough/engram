package maintain

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// HistoryRecorderOption configures a TOMLHistoryRecorder.
type HistoryRecorderOption func(*TOMLHistoryRecorder)

// TOMLHistoryRecorder reads and appends maintenance history to memory TOML files.
type TOMLHistoryRecorder struct {
	readFile  func(name string) ([]byte, error)
	writeFile func(name string, data []byte, perm os.FileMode) error
}

// NewTOMLHistoryRecorder creates a TOMLHistoryRecorder with real filesystem operations.
func NewTOMLHistoryRecorder(opts ...HistoryRecorderOption) *TOMLHistoryRecorder {
	recorder := &TOMLHistoryRecorder{
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
	}

	for _, opt := range opts {
		opt(recorder)
	}

	return recorder
}

// AppendAction reads the memory TOML, appends a MaintenanceAction, and writes it back.
func (r *TOMLHistoryRecorder) AppendAction(path string, action memory.MaintenanceAction) error {
	data, err := r.readFile(path)
	if err != nil {
		return fmt.Errorf("reading record for history: %w", err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return fmt.Errorf("decoding record for history: %w", decodeErr)
	}

	record.MaintenanceHistory = append(record.MaintenanceHistory, action)

	var buf bytes.Buffer

	encodeErr := toml.NewEncoder(&buf).Encode(record)
	if encodeErr != nil {
		return fmt.Errorf("encoding record with history: %w", encodeErr)
	}

	const filePerm = 0o644

	writeErr := r.writeFile(path, buf.Bytes(), filePerm)
	if writeErr != nil {
		return fmt.Errorf("writing record with history: %w", writeErr)
	}

	return nil
}

// ReadRecord reads a memory TOML file and returns the MemoryRecord.
func (r *TOMLHistoryRecorder) ReadRecord(path string) (*memory.MemoryRecord, error) {
	data, err := r.readFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading record: %w", err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return nil, fmt.Errorf("decoding record: %w", decodeErr)
	}

	return &record, nil
}

// WithHistoryReadFile overrides the file reading function.
func WithHistoryReadFile(fn func(name string) ([]byte, error)) HistoryRecorderOption {
	return func(r *TOMLHistoryRecorder) { r.readFile = fn }
}

// WithHistoryWriteFile overrides the file writing function.
func WithHistoryWriteFile(fn func(name string, data []byte, perm os.FileMode) error) HistoryRecorderOption {
	return func(r *TOMLHistoryRecorder) { r.writeFile = fn }
}
