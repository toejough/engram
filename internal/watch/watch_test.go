package watch_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/watch"
)

func TestFSNotifyWatcher_WaitForChange_ReturnsErrorOnBadPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	watcher := &watch.FSNotifyWatcher{}
	ctx := context.Background()

	err := watcher.WaitForChange(ctx, "/nonexistent/path/that/does/not/exist.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("watching file")))
}

func TestFSNotifyWatcher_WaitForChange_ReturnsOnCtxCancel(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(path, []byte("initial"), 0o600)).To(Succeed())

	watcher := &watch.FSNotifyWatcher{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	errCh := make(chan error, 1)

	go func() {
		errCh <- watcher.WaitForChange(ctx, path)
	}()

	time.Sleep(10 * time.Millisecond) // let watcher register
	cancel()

	err := <-errCh
	g.Expect(err).To(MatchError(ContainSubstring("context canceled")))
}

// Integration test — uses real temp files and fsnotify.
func TestFSNotifyWatcher_WaitForChange_ReturnsOnFileWrite(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(path, []byte("initial"), 0o600)).To(Succeed())

	watcher := &watch.FSNotifyWatcher{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		errCh <- watcher.WaitForChange(ctx, path)
	}()

	time.Sleep(10 * time.Millisecond) // let watcher register
	g.Expect(os.WriteFile(path, []byte("changed"), 0o600)).To(Succeed())

	err := <-errCh
	g.Expect(err).NotTo(HaveOccurred())
}
