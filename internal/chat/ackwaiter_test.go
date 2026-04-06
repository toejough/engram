package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/chat"
)

// ---------------------------------------------------------------------------
// AckResult JSON round-trip tests (Steps 1–2)
// ---------------------------------------------------------------------------

func TestAckResult_ACK_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := chat.AckResult{Result: "ACK", NewCursor: 1234}
	data, err := json.Marshal(result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var got chat.AckResult
	g.Expect(json.Unmarshal(data, &got)).To(Succeed())
	g.Expect(got.Result).To(Equal("ACK"))
	g.Expect(got.NewCursor).To(Equal(1234))
	g.Expect(got.Wait).To(BeNil())
	g.Expect(got.Timeout).To(BeNil())
}

func TestAckResult_TIMEOUT_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := chat.AckResult{
		Result:    "TIMEOUT",
		NewCursor: 999,
		Timeout:   &chat.TimeoutResult{Recipient: "engram-agent"},
	}
	data, err := json.Marshal(result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`"result":"TIMEOUT"`))
	g.Expect(string(data)).To(ContainSubstring(`"recipient":"engram-agent"`))
}

func TestAckResult_WAIT_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := chat.AckResult{
		Result:    "WAIT",
		NewCursor: 5678,
		Wait:      &chat.WaitResult{From: "engram-agent", Text: "objection text"},
	}
	data, err := json.Marshal(result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`"result":"WAIT"`))
	g.Expect(string(data)).To(ContainSubstring(`"from":"engram-agent"`))
	g.Expect(string(data)).To(ContainSubstring(`"cursor":5678`))
	g.Expect(string(data)).To(ContainSubstring(`"text":"objection text"`))
}

// ---------------------------------------------------------------------------
// FileAckWaiter unit tests (Steps 3–4)
// ---------------------------------------------------------------------------

func TestFileAckWaiter_AllACK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two recipients; both post ack messages after cursor.
	callCount := 0
	ackMessages := []chat.Message{
		{From: "engram-agent", To: "caller", Thread: "t", Type: "ack", TS: now, Text: "ok"},
		{From: "reviewer", To: "caller", Thread: "t", Type: "ack", TS: now, Text: "ok"},
	}

	fakeWatch := watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		callCount++
		idx := callCount - 1

		if idx < len(ackMessages) {
			return ackMessages[idx], cursor + 10, nil
		}

		// Should not be called beyond the number of recipients
		return chat.Message{}, 0, context.Canceled
	})

	// No messages in last 15 min → both offline (implicit ACK available, but real ACKs arrive first)
	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  func() time.Time { return now },
		MaxWait:  5 * time.Second,
	}

	result, err := waiter.AckWait(context.Background(), "caller", 0, []string{"engram-agent", "reviewer"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("ACK"))
	g.Expect(result.Wait).To(BeNil())
	g.Expect(result.Timeout).To(BeNil())
}

func TestFileAckWaiter_CaseInsensitiveRecipientNames(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Recipient registered as "Engram-Agent" but ACK arrives from "engram-agent" (lowercase).
	// With case normalization, this must match and return ACK.
	ackMsg := chat.Message{
		From: "engram-agent", // lowercase — matches "Engram-Agent" after normalization
		To:   "caller", Thread: "t", Type: "ack", TS: now, Text: "ok",
	}

	fakeWatch := watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		return ackMsg, cursor + 10, nil
	})

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  func() time.Time { return now },
		MaxWait:  5 * time.Second,
	}

	// Pass mixed-case recipient name
	result, err := waiter.AckWait(context.Background(), "caller", 0, []string{"Engram-Agent"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("ACK"))
}

func TestFileAckWaiter_CtxCancellation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Cancel ctx while waiting. Expect ctx.Err() returned.
	fakeWatch := watcherFunc(func(ctx context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		<-ctx.Done()

		return chat.Message{}, cursor, ctx.Err()
	})

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  func() time.Time { return now },
		MaxWait:  30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := waiter.AckWait(ctx, "caller", 0, []string{"engram-agent"})
	g.Expect(err).To(MatchError(context.Canceled))
}

func TestFileAckWaiter_MultiRecipient_BothMustACK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two recipients: "reviewer" ACKs immediately, "engram-agent" is offline → implicit ACK after 5s.
	// Both offline (no messages in last 15 min) but reviewer ACKs explicitly.

	ackMsg := chat.Message{From: "reviewer", To: "caller", Thread: "t", Type: "ack", TS: now, Text: "ok"}

	watchCallCount := 0
	fakeWatch := watcherFunc(func(ctx context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		watchCallCount++

		if watchCallCount == 1 {
			// First call: return reviewer ACK immediately
			return ackMsg, cursor + 10, nil
		}

		// Subsequent: block until ctx done (engram-agent never responds)
		<-ctx.Done()

		return chat.Message{}, cursor, ctx.Err()
	})

	// No messages → both offline
	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	nowCallCount := 0
	baseTime := now
	fakeNow := func() time.Time {
		nowCallCount++

		if nowCallCount <= 2 {
			return baseTime
		}

		return baseTime.Add(6 * time.Second) // engram-agent offline 5s elapsed
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  fakeNow,
		MaxWait:  30 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := waiter.AckWait(ctx, "caller", 0, []string{"engram-agent", "reviewer"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("ACK"))
}

func TestFileAckWaiter_OfflineImplicitACKAfter5s(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Recipient has no messages in last 15 min (offline).
	// NowFunc advances time so that 5s elapsed offline timer fires before Watch is called.
	fakeWatch := watcherFunc(func(ctx context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		// Block until context cancelled (simulates no message arriving)
		<-ctx.Done()

		return chat.Message{}, cursor, ctx.Err()
	})

	// No messages at all → recipient is offline
	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	// Time advances past 5s on second call (first call builds states; second is the loop check)
	nowCallCount := 0
	baseTime := now
	fakeNow := func() time.Time {
		nowCallCount++

		if nowCallCount <= 1 {
			return baseTime
		}

		// Simulate 6s elapsed — fires offline implicit ACK before Watch is called
		return baseTime.Add(6 * time.Second)
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  fakeNow,
		MaxWait:  30 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := waiter.AckWait(ctx, "caller", 0, []string{"engram-agent"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("ACK"))
}

func TestFileAckWaiter_OnlineSilentTIMEOUT(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Recipient posted a message within last 15 min (online).
	// Watch never returns matching message before max-wait expires.

	// Build a file with a recent message from engram-agent (5 min ago = online)
	recentMsg := chat.Message{
		From:   "engram-agent",
		To:     "all",
		Thread: "heartbeat",
		Type:   "info",
		TS:     now.Add(-5 * time.Minute),
		Text:   "alive",
	}

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML([]chat.Message{recentMsg}), nil
	}

	// Watch blocks until context cancels
	fakeWatch := watcherFunc(func(ctx context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		<-ctx.Done()

		return chat.Message{}, cursor, ctx.Err()
	})

	// Time advances past MaxWait on second call (first builds states; second is loop check)
	nowCallCount := 0
	baseTime := now
	fakeNow := func() time.Time {
		nowCallCount++

		if nowCallCount <= 1 {
			return baseTime
		}

		return baseTime.Add(35 * time.Second) // past 30s MaxWait
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  fakeNow,
		MaxWait:  30 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := waiter.AckWait(ctx, "caller", 0, []string{"engram-agent"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("TIMEOUT"))
	g.Expect(result.Timeout).NotTo(BeNil())

	if result.Timeout == nil {
		return
	}

	g.Expect(result.Timeout.Recipient).To(Equal("engram-agent"))
}

func TestFileAckWaiter_ReadFileError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ioErr := errors.New("disk I/O error")

	waiter := &chat.FileAckWaiter{
		FilePath: "/unreachable/chat.toml",
		Watcher: watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
			return chat.Message{}, cursor, errors.New("should not be called")
		}),
		ReadFile: func(_ string) ([]byte, error) {
			return nil, ioErr
		},
		NowFunc: time.Now,
		MaxWait: 30 * time.Second,
	}

	_, err := waiter.AckWait(context.Background(), "caller", 0, []string{"executor-1"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("disk I/O error")))
}

func TestFileAckWaiter_ReadFileNotExist_TreatedAsOffline(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// ReadFile returns ErrNotExist — recipient has no messages, so treated as offline.
	// NowFunc jumps 6s ahead on second call to trigger offline implicit ACK without blocking.
	startTime := time.Now()
	callCount := 0

	waiter := &chat.FileAckWaiter{
		FilePath: "/nonexistent/chat.toml",
		Watcher: watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
			return chat.Message{}, cursor, errors.New("should not be called")
		}),
		ReadFile: func(_ string) ([]byte, error) {
			return nil, fs.ErrNotExist
		},
		NowFunc: func() time.Time {
			callCount++
			if callCount == 1 {
				return startTime
			}

			return startTime.Add(6 * time.Second)
		},
		MaxWait: 30 * time.Second,
	}

	result, err := waiter.AckWait(context.Background(), "caller", 0, []string{"executor-1"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("ACK"))
}

func TestFileAckWaiter_WAITReturnedImmediately(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// First response is a WAIT from engram-agent.
	waitMsg := chat.Message{
		From: "engram-agent", To: "caller", Thread: "t",
		Type: "wait", TS: now, Text: "I have a relevant memory",
	}

	fakeWatch := watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		return waitMsg, cursor + 10, nil
	})

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  func() time.Time { return now },
		MaxWait:  5 * time.Second,
	}

	result, err := waiter.AckWait(context.Background(), "caller", 0, []string{"engram-agent"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("WAIT"))
	g.Expect(result.Wait).NotTo(BeNil())

	if result.Wait == nil {
		return
	}

	g.Expect(result.Wait.From).To(Equal("engram-agent"))
	g.Expect(result.Wait.Text).To(Equal("I have a relevant memory"))
}

func TestFileAckWaiter_WatchIOError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Watch returns a non-context, non-DeadlineExceeded error (e.g., I/O failure).
	// AckWait must propagate it rather than silently looping.
	ioErr := errors.New("permission denied reading chat file")

	fakeWatch := watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		return chat.Message{}, cursor, ioErr
	})

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  func() time.Time { return now },
		MaxWait:  30 * time.Second,
	}

	_, err := waiter.AckWait(context.Background(), "caller", 0, []string{"engram-agent"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("permission denied")))
}

func TestFileAckWaiter_WatchInternalTimeoutThenACK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Simulates: Watch returns an internal timeout (DeadlineExceeded) on first call
	// while parent ctx is still alive. On second call Watch returns an ACK.
	// Expected: AckWait continues the loop and returns ACK.
	ackMsg := chat.Message{From: "engram-agent", To: "caller", Thread: "t", Type: "ack", TS: now, Text: "ok"}

	watchCallCount := 0
	fakeWatch := watcherFunc(func(_ context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		watchCallCount++

		if watchCallCount == 1 {
			// Simulate Watch's own internal timeout (not parent ctx cancellation)
			return chat.Message{}, cursor, context.DeadlineExceeded
		}

		return ackMsg, cursor + 10, nil
	})

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML(nil), nil
	}

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  func() time.Time { return now },
		MaxWait:  30 * time.Second,
	}

	result, err := waiter.AckWait(context.Background(), "caller", 0, []string{"engram-agent"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("ACK"))
}

// watcherFunc adapts a raw function to the chat.Watcher interface for tests.
type watcherFunc func(ctx context.Context, agent string, cursor int, msgTypes []string) (chat.Message, int, error)

func (f watcherFunc) Watch(
	ctx context.Context, agent string, cursor int, msgTypes []string,
) (chat.Message, int, error) {
	return f(ctx, agent, cursor, msgTypes)
}
