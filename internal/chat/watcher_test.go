package chat_test

import (
	"context"
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

// unexported variables.
var (
	now = time.Date(2026, 4, 5, 7, 0, 0, 0, time.UTC)
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
