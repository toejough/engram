// Package watch provides a file change notification abstraction.
// FSNotifyWatcher is the I/O adapter boundary for fsnotify.
package watch

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// FSNotifyWatcher uses fsnotify (kqueue on macOS, inotify on Linux). No CGO.
type FSNotifyWatcher struct{}

// WaitForChange blocks until the file at path is modified or ctx is cancelled.
func (w *FSNotifyWatcher) WaitForChange(ctx context.Context, path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	defer watcher.Close() //nolint:errcheck

	err = watcher.Add(path)
	if err != nil {
		return fmt.Errorf("watching file: %w", err)
	}

	select {
	case <-watcher.Events:
		return nil
	case watchErr := <-watcher.Errors:
		return fmt.Errorf("fsnotify error: %w", watchErr)
	case <-ctx.Done():
		return fmt.Errorf("watch cancelled: %w", ctx.Err())
	}
}

// Watcher blocks until a file changes or context is cancelled.
type Watcher interface {
	WaitForChange(ctx context.Context, path string) error
}
