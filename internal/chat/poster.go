package chat

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// FilePoster appends messages to the chat file atomically with locking.
// All I/O is injected — no os.* calls in this package.
type FilePoster struct {
	FilePath   string
	Lock       LockFile
	AppendFile func(path string, data []byte) error
	LineCount  func(path string) (int, error)
	NowFunc    func() time.Time // injectable for tests; defaults to time.Now().UTC()
}

// Post appends msg to the chat file and returns the new cursor (line count).
func (p *FilePoster) Post(msg Message) (int, error) {
	msg.TS = p.now()

	data := formatMessage(msg)

	unlock, lockErr := p.Lock(p.FilePath + ".lock")
	if lockErr != nil {
		return 0, fmt.Errorf("acquiring lock: %w", lockErr)
	}

	defer unlock() //nolint:errcheck

	appendErr := p.AppendFile(p.FilePath, data)
	if appendErr != nil {
		return 0, fmt.Errorf("appending to chat file: %w", appendErr)
	}

	cursor, countErr := p.LineCount(p.FilePath)
	if countErr != nil {
		return 0, fmt.Errorf("counting lines: %w", countErr)
	}

	return cursor, nil
}

func (p *FilePoster) now() time.Time {
	if p.NowFunc != nil {
		return p.NowFunc()
	}

	return time.Now().UTC()
}

// formatMessage encodes a Message as TOML with exact field order:
// from, to, thread, type, ts, text.
// Uses fmt.Fprintf instead of toml.Encoder to guarantee field order.
// Text field is always triple-quoted multiline per the chat protocol spec.
func formatMessage(msg Message) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "\n[[message]]\n")
	fmt.Fprintf(&buf, "from = %q\n", msg.From)
	fmt.Fprintf(&buf, "to = %q\n", msg.To)
	fmt.Fprintf(&buf, "thread = %q\n", msg.Thread)
	fmt.Fprintf(&buf, "type = %q\n", msg.Type)
	fmt.Fprintf(&buf, "ts = %s\n", msg.TS.UTC().Format(time.RFC3339Nano))
	// Escape any triple-quote sequence that would prematurely terminate the TOML multi-line string.
	escapedText := strings.ReplaceAll(msg.Text, `"""`, `""\"`)
	fmt.Fprintf(&buf, "text = \"\"\"\n%s\n\"\"\"\n", escapedText)

	return buf.Bytes()
}
