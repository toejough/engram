// Package watch provides a file change notification abstraction.
// FSNotifyWatcher is the I/O adapter boundary for fsnotify.
package watch

import "context"

// Watcher blocks until a file changes or context is cancelled.
type Watcher interface {
	WaitForChange(ctx context.Context, path string) error
}
