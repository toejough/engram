package server_test

import (
	"context"
	"testing"
	"time"

	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestSharedWatcher_AllSubscribersNotifiedOnChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notify := make(chan struct{}, 1)
	watcher := server.NewSharedWatcher(func(_ context.Context, _ string) error {
		<-notify
		return nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch1 := watcher.Subscribe()
	ch2 := watcher.Subscribe()

	go watcher.Run(ctx, "/fake/path") //nolint:errcheck

	// Trigger a file change notification.
	notify <- struct{}{}

	// Both subscribers should be notified.
	g.Eventually(func() bool {
		select {
		case <-ch1:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())

	g.Eventually(func() bool {
		select {
		case <-ch2:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())
}

func TestSharedWatcher_CoalescesWhenSubscriberBusy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notify := make(chan struct{}, 1)
	watcher := server.NewSharedWatcher(func(_ context.Context, _ string) error {
		<-notify
		return nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch1 := watcher.Subscribe()

	go watcher.Run(ctx, "/fake/path") //nolint:errcheck

	// Send two notifications without draining the subscriber channel.
	// The second should coalesce (not block the watcher).
	notify <- struct{}{}

	// Wait for first notification to arrive.
	g.Eventually(func() int {
		return len(ch1)
	}).WithTimeout(time.Second).Should(Equal(1))

	// Send another notification while ch1 is still full.
	notify <- struct{}{}

	// Give the watcher time to process -- it should not block.
	// Drain and verify we only get one buffered notification.
	<-ch1

	g.Eventually(func() int {
		return len(ch1)
	}).WithTimeout(time.Second).Should(Equal(1))
}

func TestSharedWatcher_RunStopsOnContextCancel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	watcher := server.NewSharedWatcher(func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)

	go func() { done <- watcher.Run(ctx, "/fake") }()

	cancel()

	g.Eventually(done).WithTimeout(time.Second).Should(Receive())
}

func TestSharedWatcher_UnsubscribeRemovesSubscriber(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notify := make(chan struct{}, 1)
	watcher := server.NewSharedWatcher(func(_ context.Context, _ string) error {
		<-notify
		return nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch1 := watcher.Subscribe()
	ch2 := watcher.Subscribe()
	watcher.Unsubscribe(ch1)

	go watcher.Run(ctx, "/fake/path") //nolint:errcheck

	notify <- struct{}{}

	// ch2 should be notified.
	g.Eventually(func() bool {
		select {
		case <-ch2:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())

	// ch1 should NOT be notified (unsubscribed).
	g.Consistently(func() bool {
		select {
		case <-ch1:
			return true
		default:
			return false
		}
	}).WithTimeout(100 * time.Millisecond).Should(BeFalse())
}

func TestSharedWatcher_UnsubscribeUnknownChannelIsNoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notify := make(chan struct{}, 1)
	watcher := server.NewSharedWatcher(func(_ context.Context, _ string) error {
		<-notify
		return nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch1 := watcher.Subscribe()

	// Unsubscribe a channel that was never subscribed -- should be a no-op.
	unknown := make(chan struct{}, 1)
	watcher.Unsubscribe(unknown)

	go watcher.Run(ctx, "/fake/path") //nolint:errcheck

	notify <- struct{}{}

	// The real subscriber should still be notified.
	g.Eventually(func() bool {
		select {
		case <-ch1:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())
}
