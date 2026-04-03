package cli_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRun_Serve_GracefulShutdown(t *testing.T) { //nolint:paralleltest // mutates globals + sends signal
	g := NewWithT(t)

	original := cli.BrowserOpener
	urlCh := make(chan string, 1)
	cli.BrowserOpener = func(url string) { urlCh <- url }

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	port := freePort(t)
	stdout := &syncWriter{}
	done := make(chan error, 1)

	go func() {
		done <- cli.Run(
			[]string{"engram", "serve", "--data-dir", dataDir, "--port", port},
			stdout, stdout,
			strings.NewReader(""),
		)
	}()

	// Wait for BrowserOpener to be called.
	select {
	case <-urlCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for browser opener")
	}

	cli.BrowserOpener = original

	waitForServer(t, port)

	// Send SIGINT to trigger graceful shutdown.
	g.Expect(syscall.Kill(syscall.Getpid(), syscall.SIGINT)).To(Succeed())

	// Wait for server to stop.
	select {
	case err := <-done:
		g.Expect(err).NotTo(HaveOccurred())
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within timeout")
	}

	g.Expect(stdout.String()).To(ContainSubstring("engram server stopped"))
}

func TestRun_Serve_ParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stdout := &syncWriter{}

	err := cli.Run(
		[]string{"engram", "serve", "--unknown-flag"},
		stdout, stdout,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("serve"))
	}
}

func TestRun_Serve_PortInUse(t *testing.T) { //nolint:paralleltest // mutates package-level BrowserOpener
	g := NewWithT(t)

	original := cli.BrowserOpener
	cli.BrowserOpener = func(_ string) {}

	t.Cleanup(func() { cli.BrowserOpener = original })

	// Bind to a port so the server can't use it.
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	g.Expect(listenErr).NotTo(HaveOccurred())

	if listenErr != nil {
		return
	}

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port) //nolint:forcetypeassert // always *net.TCPAddr

	defer func() { _ = listener.Close() }()

	dataDir := t.TempDir()
	stdout := &syncWriter{}
	done := make(chan error, 1)

	go func() {
		done <- cli.Run(
			[]string{"engram", "serve", "--data-dir", dataDir, "--port", port},
			stdout, stdout,
			strings.NewReader(""),
		)
	}()

	select {
	case err := <-done:
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("serve"))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not return error within timeout")
	}
}

func TestRun_Serve_StartsAndResponds(t *testing.T) { //nolint:paralleltest // mutates package-level BrowserOpener
	g := NewWithT(t)

	original := cli.BrowserOpener

	urlCh := make(chan string, 1)
	cli.BrowserOpener = func(url string) { urlCh <- url }

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	port := freePort(t)
	stdout := &syncWriter{}

	go func() {
		_ = cli.Run(
			[]string{"engram", "serve", "--data-dir", dataDir, "--port", port},
			stdout, stdout,
			strings.NewReader(""),
		)
	}()

	// Wait for BrowserOpener to be called.
	var openedURL string
	select {
	case openedURL = <-urlCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for browser opener")
	}

	cli.BrowserOpener = original

	waitForServer(t, port)

	expectedURL := "http://localhost:" + port

	// Verify startup message.
	g.Eventually(stdout.String, time.Second, 10*time.Millisecond).Should(ContainSubstring(
		"engram server listening on " + expectedURL,
	))

	// Verify browser was told to open.
	g.Expect(openedURL).To(Equal(expectedURL))

	// Verify server responds on 127.0.0.1.
	req, reqErr := http.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"http://127.0.0.1:"+port+"/api/memories", nil,
	)
	g.Expect(reqErr).NotTo(HaveOccurred())

	if reqErr != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

// syncWriter wraps an io.Writer with a mutex for concurrent access.
type syncWriter struct {
	mu  sync.Mutex
	buf []byte
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return string(w.buf)
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)

	return len(p), nil
}

// freePort returns an available TCP port as a string.
func freePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // always *net.TCPAddr
	_ = listener.Close()

	return strconv.Itoa(port)
}

// waitForServer polls until the server is accepting connections.
func waitForServer(t *testing.T, port string) {
	t.Helper()

	const (
		maxAttempts = 50
		pollDelay   = 20 * time.Millisecond
	)

	addr := "127.0.0.1:" + port

	for range maxAttempts {
		conn, err := net.DialTimeout("tcp", addr, pollDelay)
		if err == nil {
			_ = conn.Close()

			return
		}

		time.Sleep(pollDelay)
	}

	t.Fatalf("server did not start on port %s within timeout", port)
}
