// Package audit provides structured append-only audit logging.
package audit

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Entry represents a single audit log record with timestamp, operation, and fields.
type Entry struct {
	Timestamp time.Time
	Operation string
	Action    string
	Fields    map[string]string
}

// Logger writes structured audit entries to an io.Writer.
type Logger struct {
	w io.Writer
}

// NewLogger creates a Logger that writes audit entries to w.
func NewLogger(w io.Writer) *Logger {
	return &Logger{w: w}
}

// Log writes a structured audit entry to the underlying writer.
func (l *Logger) Log(e Entry) error {
	var sb strings.Builder

	sb.WriteString(e.Timestamp.Format(time.RFC3339))
	sb.WriteByte(' ')
	sb.WriteString(e.Operation)
	sb.WriteByte(' ')
	sb.WriteString(e.Action)

	for k, v := range e.Fields {
		sb.WriteByte(' ')
		fmt.Fprintf(&sb, "%s=%q", k, v)
	}

	sb.WriteByte('\n')

	_, err := io.WriteString(l.w, sb.String())
	if err != nil {
		return fmt.Errorf("audit: write: %w", err)
	}

	return nil
}
