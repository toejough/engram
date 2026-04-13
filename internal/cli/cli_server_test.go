package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"engram/internal/cli"

	. "github.com/onsi/gomega"
)

func TestRunServerUp_InvalidFlag_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.RunWithContext(
		t.Context(),
		[]string{"engram", "server", "up", "--unknown-flag"},
		&bytes.Buffer{}, &bytes.Buffer{}, nil,
	)
	g.Expect(err).To(HaveOccurred())
}

func TestRunServerUp_PostMessageWritesToChatFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := dir + "/chat.toml"

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var stderr syncBuffer

	done := make(chan error, 1)

	go func() {
		done <- cli.RunWithContext(ctx, []string{
			"engram", "server", "up",
			"--chat-file", chatFile,
			"--addr", "localhost:0",
		}, &bytes.Buffer{}, &stderr, nil)
	}()

	g.Eventually(stderr.String).
		WithTimeout(5 * time.Second).
		Should(ContainSubstring("server started"))

	addr := extractServerAddr(stderr.String())
	g.Expect(addr).NotTo(BeEmpty())

	// POST a message.
	postBody := strings.NewReader(`{"from":"lead-1","to":"engram-agent","text":"hello"}`)

	postReq, postReqErr := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "http://"+addr+"/message", postBody,
	)
	g.Expect(postReqErr).NotTo(HaveOccurred())

	if postReqErr != nil {
		return
	}

	postResp, postHTTPErr := http.DefaultClient.Do(postReq)
	g.Expect(postHTTPErr).NotTo(HaveOccurred())

	if postHTTPErr != nil {
		return
	}

	if postResp == nil {
		return
	}

	defer func() { _ = postResp.Body.Close() }()

	g.Expect(postResp.StatusCode).To(Equal(http.StatusOK))

	var postResult map[string]any
	g.Expect(json.NewDecoder(postResp.Body).Decode(&postResult)).To(Succeed())
	g.Expect(postResult["cursor"]).NotTo(BeZero())

	cancel()
	<-done
}

func TestRunServerUp_StartsAndRespondsToStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := dir + "/chat.toml"

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var stdout syncBuffer

	var stderr syncBuffer

	done := make(chan error, 1)

	go func() {
		done <- cli.RunWithContext(ctx, []string{
			"engram", "server", "up",
			"--chat-file", chatFile,
			"--addr", "localhost:0",
		}, &stdout, &stderr, nil)
	}()

	// Wait for server to start (check stderr for "server started").
	g.Eventually(stderr.String).
		WithTimeout(5 * time.Second).
		Should(ContainSubstring("server started"))

	// Extract the address from the slog JSON output.
	addr := extractServerAddr(stderr.String())
	g.Expect(addr).NotTo(BeEmpty())

	// Verify /status responds.
	req, reqErr := http.NewRequestWithContext(
		t.Context(), http.MethodGet, "http://"+addr+"/status", nil,
	)
	g.Expect(reqErr).NotTo(HaveOccurred())

	if reqErr != nil {
		return
	}

	resp, httpErr := http.DefaultClient.Do(req)
	g.Expect(httpErr).NotTo(HaveOccurred())

	if httpErr != nil {
		return
	}

	if resp == nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var body map[string]any
	g.Expect(json.NewDecoder(resp.Body).Decode(&body)).To(Succeed())
	g.Expect(body["running"]).To(BeTrue())

	cancel()
	<-done
}

func TestRunServerUp_WithLogFile_Starts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := dir + "/chat.toml"
	logFile := dir + "/server.log"

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var stderr syncBuffer

	done := make(chan error, 1)

	go func() {
		done <- cli.RunWithContext(ctx, []string{
			"engram", "server", "up",
			"--chat-file", chatFile,
			"--addr", "localhost:0",
			"--log-file", logFile,
		}, &bytes.Buffer{}, &stderr, nil)
	}()

	g.Eventually(stderr.String).
		WithTimeout(5 * time.Second).
		Should(ContainSubstring("server started"))

	cancel()
	<-done
}

func TestRunServer_NoSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.RunWithContext(
		t.Context(),
		[]string{"engram", "server"},
		&bytes.Buffer{}, &bytes.Buffer{}, nil,
	)
	g.Expect(err).To(MatchError(ContainSubstring("subcommand required")))
}

func TestRunServer_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.RunWithContext(
		t.Context(),
		[]string{"engram", "server", "down"},
		&bytes.Buffer{}, &bytes.Buffer{}, nil,
	)
	g.Expect(err).To(MatchError(ContainSubstring("down")))
}

func TestRun_ServerUp_BadFlag_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Call Run (not RunWithContext) to exercise runServerWithSignal.
	// Flag parsing fails fast before any server starts.
	err := cli.Run(
		[]string{"engram", "server", "up", "--unknown-flag"},
		&bytes.Buffer{}, &bytes.Buffer{}, nil,
	)
	g.Expect(err).To(HaveOccurred())
}

// unexported types.

// syncBuffer is a goroutine-safe bytes.Buffer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// unexported functions.

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

// extractServerAddr parses the slog JSON log output for the "addr" field
// in the "server started" log line.
func extractServerAddr(logOutput string) string {
	for line := range strings.SplitSeq(logOutput, "\n") {
		if !strings.Contains(line, "server started") {
			continue
		}

		var entry map[string]any

		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if addr, ok := entry["addr"].(string); ok && addr != "" {
			return addr
		}
	}

	return ""
}
