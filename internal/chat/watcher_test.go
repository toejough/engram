package chat_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/chat"
)

func TestFileWatcher_Watch_AllInToField(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// "all" in To field matches any agent name.
	msgs := []chat.Message{
		{From: "a", To: "all", Thread: "t", Type: "info", TS: now, Text: "broadcast"},
	}

	allContent := buildChatTOML(msgs)

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return allContent, nil },
	}

	msg, _, err := watcher.Watch(context.Background(), "anyagent", 0, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("broadcast"))
}

func TestFileWatcher_Watch_CommaSepratedToField(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// To field has comma-separated recipients.
	msgs := []chat.Message{
		{From: "a", To: "engram-agent, myagent", Thread: "t", Type: "info", TS: now, Text: "targeted"},
	}

	allContent := buildChatTOML(msgs)

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return allContent, nil },
	}

	msg, _, err := watcher.Watch(context.Background(), "myagent", 0, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("targeted"))
}

func TestFileWatcher_Watch_CorruptSuffixBlocksThenCancels(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Suffix itself is corrupt — findMessage logs and returns false, Watch blocks on WaitForChange.
	// Context cancelled so Watch exits with Canceled.
	corruptSuffix := []byte(`
[[message]]
from = "x"
to = "myagent"
thread = "t"
type = "info"
ts = 2026-04-05T07:00:00Z
text = "bad \[ escape in suffix"
`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &blockingWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return corruptSuffix, nil },
	}

	_, _, err := watcher.Watch(ctx, "myagent", 0, nil)
	g.Expect(err).To(MatchError(context.Canceled))
}

func TestFileWatcher_Watch_CtxCancellation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// No messages match — watcher will block, context cancels it.
	msgs := []chat.Message{
		{From: "a", To: "other", Thread: "t", Type: "info", TS: now, Text: "not for me"},
	}

	allContent := buildChatTOML(msgs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &blockingWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return allContent, nil },
	}

	_, _, err := watcher.Watch(ctx, "myagent", 0, nil)
	g.Expect(err).To(MatchError(context.Canceled))
}

func TestFileWatcher_Watch_CursorPastEnd(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// File has a few lines but cursor is past the end.
	// suffixAtLine returns nil → ParseMessages(nil) → empty, no match.
	// Context cancelled immediately so Watch exits instead of looping.
	data := buildChatTOML([]chat.Message{
		{From: "a", To: "other", Thread: "t", Type: "info", TS: now, Text: "x"},
	})

	cursorPastEnd := countLines(data) + 100

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &blockingWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return data, nil },
	}

	_, _, err := watcher.Watch(ctx, "myagent", cursorPastEnd, nil)
	g.Expect(err).To(MatchError(context.Canceled))
}

func TestFileWatcher_Watch_EmptyAgentMatchesAll(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// agent="" is a wildcard — matches messages addressed to any recipient.
	msgs := []chat.Message{
		{From: "sender", To: "lead", Thread: "t", Type: "intent", TS: now, Text: "to lead"},
	}

	content := buildChatTOML(msgs)

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return content, nil },
	}

	// empty agent string should match "lead" (or any other recipient)
	msg, _, err := watcher.Watch(context.Background(), "", 0, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("to lead"))
}

func TestFileWatcher_Watch_FiltersByType(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Two messages for "myagent": first is "info", second is "ack".
	// Filter: ["ack"] — should return second message, skipping info.
	msgs := []chat.Message{
		{From: "a", To: "myagent", Thread: "t", Type: "info", TS: now, Text: "first info"},
		{From: "b", To: "myagent", Thread: "t", Type: "ack", TS: now, Text: "first ack"},
	}

	allContent := buildChatTOML(msgs)

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return allContent, nil },
	}

	msg, _, err := watcher.Watch(context.Background(), "myagent", 0, []string{"ack"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("first ack"))
	g.Expect(msg.Type).To(Equal("ack"))
}

// TestFileWatcher_Watch_InvalidTOMLBeforeCursor verifies that Watch finds
// a valid message after the cursor even when historical data before the cursor
// contains invalid TOML (e.g. a backslash-bracket escape in a basic string).
func TestFileWatcher_Watch_InvalidTOMLBeforeCursor(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Corrupt "historical" section — contains a TOML basic-string escape that
	// is invalid: \[ is not a recognized escape sequence.
	corruptHistory := []byte(`
[[message]]
from = "old-agent"
to = "all"
thread = "t"
type = "info"
ts = 2026-04-05T07:00:00Z
text = "bad escape \[ causes full-file parse to fail"
`)

	cursor := countLines(corruptHistory)

	// Valid message that arrives AFTER the cursor.
	validSuffix := buildChatTOML([]chat.Message{
		{From: "b", To: "myagent", Thread: "t", Type: "ack", TS: now, Text: "valid after cursor"},
	})

	data := make([]byte, 0, len(corruptHistory)+len(validSuffix))
	data = append(data, corruptHistory...)
	data = append(data, validSuffix...)

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return data, nil },
	}

	msg, _, err := watcher.Watch(context.Background(), "myagent", cursor, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("valid after cursor"))
}

func TestFileWatcher_Watch_NoMatchLoops(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// On first read, no match. On second read (after WaitForChange), a match appears.
	callCount := 0
	msgs1 := buildChatTOML([]chat.Message{
		{From: "a", To: "other", Thread: "t", Type: "info", TS: now, Text: "no match"},
	})
	msgs2 := buildChatTOML([]chat.Message{
		{From: "a", To: "other", Thread: "t", Type: "info", TS: now, Text: "no match"},
		{From: "b", To: "myagent", Thread: "t", Type: "info", TS: now, Text: "match"},
	})

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile: func(_ string) ([]byte, error) {
			callCount++

			if callCount == 1 {
				return msgs1, nil
			}

			return msgs2, nil
		},
	}

	msg, _, err := watcher.Watch(context.Background(), "myagent", 0, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("match"))
	g.Expect(callCount).To(BeNumerically(">=", 2))
}

func TestFileWatcher_Watch_ReadFileError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return nil, errFakeRead },
	}

	_, _, err := watcher.Watch(context.Background(), "myagent", 0, nil)
	g.Expect(err).To(MatchError(ContainSubstring("fake read error")))
}

func TestFileWatcher_Watch_ReturnsFirstMatchingMessage(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	msgs := []chat.Message{
		{From: "a", To: "other", Thread: "t", Type: "info", TS: now, Text: "x"},
		{From: "b", To: "myagent", Thread: "t", Type: "info", TS: now, Text: "first match"},
		{From: "c", To: "myagent", Thread: "t", Type: "ack", TS: now, Text: "ack match"},
	}

	allContent := buildChatTOML(msgs)

	// cursor = line count after first message only
	cursorAfterFirst := countLines(buildChatTOML(msgs[:1]))

	watcher := &chat.FileWatcher{
		FilePath:  "/fake/chat.toml",
		FSWatcher: &fakeWatcher{},
		ReadFile:  func(_ string) ([]byte, error) { return allContent, nil },
	}

	msg, newCursor, err := watcher.Watch(context.Background(), "myagent", cursorAfterFirst, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Text).To(Equal("first match"))
	g.Expect(msg.Type).To(Equal("info"))
	g.Expect(newCursor).To(BeNumerically(">", cursorAfterFirst))
}

func TestParseMessagesSafe_CleanFile_AllMessages(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	data := []byte(`
[[message]]
from = "lead"
to = "all"
thread = "test"
type = "info"
ts = 2026-04-06T12:00:00Z
text = """hello"""

[[message]]
from = "executor"
to = "lead"
thread = "test"
type = "done"
ts = 2026-04-06T12:01:00Z
text = """done"""
`)
	msgs := chat.ParseMessagesSafe(data)
	g.Expect(msgs).To(HaveLen(2))

	if len(msgs) < 2 {
		return
	}

	g.Expect(msgs[0].From).To(Equal("lead"))
	g.Expect(msgs[1].From).To(Equal("executor"))
}

func TestParseMessagesSafe_EmptyData_ReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(chat.ParseMessagesSafe(nil)).To(BeEmpty())
	g.Expect(chat.ParseMessagesSafe([]byte(""))).To(BeEmpty())
}

func TestParseMessagesSafe_OneCorruptBlock_OthersReturned(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Embed a null byte in the middle block to cause TOML parse failure.
	data := []byte(
		"[[message]]\nfrom = \"lead\"\nto = \"all\"\nthread = \"t\"\ntype = \"info\"\n" +
			"ts = 2026-04-06T12:00:00Z\ntext = \"\"\"good\"\"\"\n\n" +
			"[[message]]\nfrom = \"corrupt\"\nto = \"all\"\nthread = \"t\"\ntype = \"info\"\n" +
			"ts = 2026-04-06T12:01:00Z\ntext = \"\"\"bad\x00bytes\"\"\"\n\n" +
			"[[message]]\nfrom = \"executor\"\nto = \"lead\"\nthread = \"t\"\ntype = \"done\"\n" +
			"ts = 2026-04-06T12:02:00Z\ntext = \"\"\"also good\"\"\"\n",
	)

	msgs := chat.ParseMessagesSafe(data)
	froms := make([]string, 0, len(msgs))

	for _, m := range msgs {
		froms = append(froms, m.From)
	}

	g.Expect(froms).To(ContainElement("lead"))
	g.Expect(froms).To(ContainElement("executor"))
	g.Expect(froms).NotTo(ContainElement("corrupt"))
}

func TestParseMessages_Empty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	msgs, err := chat.ParseMessages(nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msgs).To(BeEmpty())
}

func TestParseMessages_InvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	_, err := chat.ParseMessages([]byte(`not valid toml ][`))
	g.Expect(err).To(HaveOccurred())
}

// unexported variables.
var (
	errFakeRead = errors.New("fake read error")
	now         = time.Date(2026, 4, 5, 7, 0, 0, 0, time.UTC)
)

// blockingWatcher blocks until its context is cancelled.
type blockingWatcher struct{}

func (b *blockingWatcher) WaitForChange(ctx context.Context, _ string) error {
	<-ctx.Done()

	return ctx.Err()
}

// fakeWatcher implements watch.Watcher and returns immediately.
type fakeWatcher struct {
	err error
}

func (f *fakeWatcher) WaitForChange(_ context.Context, _ string) error {
	return f.err
}

// buildChatTOML creates raw TOML bytes for a slice of messages.
func buildChatTOML(msgs []chat.Message) []byte {
	var sb strings.Builder

	for _, m := range msgs {
		sb.WriteString("\n[[message]]\n")
		sb.WriteString("from = \"" + m.From + "\"\n")
		sb.WriteString("to = \"" + m.To + "\"\n")
		sb.WriteString("thread = \"" + m.Thread + "\"\n")
		sb.WriteString("type = \"" + m.Type + "\"\n")
		sb.WriteString("ts = " + m.TS.UTC().Format("2006-01-02T15:04:05Z") + "\n")
		sb.WriteString("text = \"\"\"\n" + m.Text + "\"\"\"\n")
	}

	return []byte(sb.String())
}

// countLines returns the number of newline characters in b.
func countLines(b []byte) int {
	return strings.Count(string(b), "\n")
}
