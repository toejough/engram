// Package server implements the engram API server.
package server

import "context"

// SharedWatcher watches a single file and fans out change notifications
// to all registered subscribers via buffered channels (buffer=1).
// If a subscriber's channel is full, the notification coalesces.
type SharedWatcher struct {
	waitForChange WaitFunc
	subscribers   []chan struct{}
}

// NewSharedWatcher creates a SharedWatcher with the given wait function.
func NewSharedWatcher(wait WaitFunc) *SharedWatcher {
	return &SharedWatcher{waitForChange: wait}
}

// Run blocks, watching the file and notifying subscribers on each change.
// Returns when ctx is cancelled or the wait function errors.
func (sw *SharedWatcher) Run(ctx context.Context, path string) error {
	for {
		err := sw.waitForChange(ctx, path)
		if err != nil {
			return err
		}

		for _, ch := range sw.subscribers {
			select {
			case ch <- struct{}{}:
			default: // coalesce — subscriber already has a pending notification
			}
		}
	}
}

// Subscribe registers a new subscriber and returns its notification channel.
// Must be called before Run — not safe for concurrent use.
func (sw *SharedWatcher) Subscribe() <-chan struct{} {
	ch := make(chan struct{}, 1)
	sw.subscribers = append(sw.subscribers, ch)

	return ch
}

// Unsubscribe removes a subscriber so it no longer receives notifications.
// Must be called before Run — not safe for concurrent use.
func (sw *SharedWatcher) Unsubscribe(ch <-chan struct{}) {
	for i, sub := range sw.subscribers {
		if sub == ch {
			sw.subscribers = append(sw.subscribers[:i], sw.subscribers[i+1:]...)

			return
		}
	}
}

// WaitFunc blocks until the watched file changes. Injected for testing.
// In production, wraps fsnotify. Signature matches watch.FSNotifyWatcher.WaitForChange.
type WaitFunc func(ctx context.Context, path string) error
