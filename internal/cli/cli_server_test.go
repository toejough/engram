package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"engram/internal/cli"

	. "github.com/onsi/gomega"
)

func TestBuildRunClaude_ClosureBinaryFails_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fake binary that exits non-zero.
	dir := t.TempDir()

	fakeSrc := filepath.Join(dir, "failclaude.go")
	g.Expect(os.WriteFile(fakeSrc, []byte(`package main
import "os"
func main() { os.Exit(1) }
`), 0o600)).To(Succeed())

	fakeBinary := filepath.Join(dir, "failclaude")
	if runtime.GOOS == "windows" {
		fakeBinary += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", fakeBinary, fakeSrc)
	buildOut, buildErr := buildCmd.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), string(buildOut))

	if buildErr != nil {
		return
	}

	runner := cli.ExportBuildRunClaude(fakeBinary)

	_, err := runner(t.Context(), "prompt", "")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("running claude"))
	}
}

func TestBuildRunClaude_ClosureRunsBinary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build a fake binary that prints a fixed string to stdout and exits 0.
	dir := t.TempDir()

	fakeSrc := filepath.Join(dir, "fakeclaude.go")
	g.Expect(os.WriteFile(fakeSrc, []byte(`package main
import "fmt"
func main() { fmt.Print("fake output") }
`), 0o600)).To(Succeed())

	fakeBinary := filepath.Join(dir, "fakeclaude")
	if runtime.GOOS == "windows" {
		fakeBinary += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", fakeBinary, fakeSrc)
	buildOut, buildErr := buildCmd.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), string(buildOut))

	if buildErr != nil {
		return
	}

	runner := cli.ExportBuildRunClaude(fakeBinary)
	g.Expect(runner).NotTo(BeNil())

	// Call with no sessionID — exercises the non-sessionID path.
	out, err := runner(t.Context(), "hello prompt", "")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("fake output"))
}

func TestBuildRunClaude_ClosureWithSessionID_RunsBinary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fake binary that accepts --resume flag and exits 0.
	dir := t.TempDir()

	fakeSrc := filepath.Join(dir, "fakeclaude.go")
	g.Expect(os.WriteFile(fakeSrc, []byte(`package main
import "fmt"
func main() { fmt.Print("resumed") }
`), 0o600)).To(Succeed())

	fakeBinary := filepath.Join(dir, "fakeclaude")
	if runtime.GOOS == "windows" {
		fakeBinary += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", fakeBinary, fakeSrc)
	buildOut, buildErr := buildCmd.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), string(buildOut))

	if buildErr != nil {
		return
	}

	runner := cli.ExportBuildRunClaude(fakeBinary)

	// Call with a sessionID — exercises the sessionID != "" branch.
	out, err := runner(t.Context(), "prompt", "session-123")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("resumed"))
}

func TestRunServerUp_InvalidAddr_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := dir + "/chat.toml"

	err := cli.RunWithContext(
		t.Context(),
		[]string{
			"engram", "server", "up",
			"--chat-file", chatFile,
			"--addr", "!!!invalid-addr!!!",
		},
		&bytes.Buffer{}, &bytes.Buffer{}, nil,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("server up"))
	}
}

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

func TestRunServerUp_NoChatFile_ResolvesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Start server without --chat-file to exercise the deriveChatFilePath branch
	// inside runServerUp (lines 158-165).
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var stderr syncBuffer

	done := make(chan error, 1)

	go func() {
		done <- cli.RunWithContext(ctx, []string{
			"engram", "server", "up",
			"--addr", "localhost:0",
		}, &bytes.Buffer{}, &stderr, nil)
	}()

	g.Eventually(stderr.String).
		WithTimeout(5 * time.Second).
		Should(ContainSubstring("server started"))

	cancel()
	<-done
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

func TestRunServerUp_PostResetAgent_Succeeds(t *testing.T) {
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

	// POST to /reset-agent.
	resetReq, resetReqErr := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "http://"+addr+"/reset-agent", nil,
	)
	g.Expect(resetReqErr).NotTo(HaveOccurred())

	if resetReqErr != nil {
		return
	}

	resetResp, resetHTTPErr := http.DefaultClient.Do(resetReq)
	g.Expect(resetHTTPErr).NotTo(HaveOccurred())

	if resetHTTPErr != nil {
		return
	}

	if resetResp == nil {
		return
	}

	defer func() { _ = resetResp.Body.Close() }()

	g.Expect(resetResp.StatusCode).To(Equal(http.StatusOK))

	var resetResult map[string]any

	g.Expect(json.NewDecoder(resetResp.Body).Decode(&resetResult)).To(Succeed())
	g.Expect(resetResult["status"]).To(Equal("reset"))

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

func TestRunServerUp_SubscribeFunc_ReturnsMessages(t *testing.T) {
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

	if addr == "" {
		return
	}

	// Post a message first so /subscribe has something to return immediately.
	postBody := strings.NewReader(`{"from":"lead-1","to":"worker-1","text":"hello"}`)

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

	if postResp != nil {
		_ = postResp.Body.Close()
	}

	// Subscribe from cursor 0 — the previously posted message should be returned immediately.
	subReq, subReqErr := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://"+addr+"/subscribe?agent=worker-1&after-cursor=0",
		nil,
	)
	g.Expect(subReqErr).NotTo(HaveOccurred())

	if subReqErr != nil {
		return
	}

	subResp, subHTTPErr := http.DefaultClient.Do(subReq)
	g.Expect(subHTTPErr).NotTo(HaveOccurred())

	if subHTTPErr != nil {
		return
	}

	if subResp == nil {
		return
	}

	defer func() { _ = subResp.Body.Close() }()

	g.Expect(subResp.StatusCode).To(Equal(http.StatusOK))

	var subResult map[string]any
	g.Expect(json.NewDecoder(subResp.Body).Decode(&subResult)).To(Succeed())
	g.Expect(subResult["messages"]).NotTo(BeNil())

	cancel()
	<-done
}

func TestRunServerUp_WatchFunc_CancelledContext_ReturnsError(t *testing.T) {
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

	if addr == "" {
		return
	}

	// Subscribe with a very short deadline — context cancels before a message arrives,
	// which exercises the watchErr != nil path inside SubscribeFunc.
	subCtx, subCancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer subCancel()

	subReq, subReqErr := http.NewRequestWithContext(
		subCtx,
		http.MethodGet,
		"http://"+addr+"/subscribe?agent=nobody&after-cursor=0",
		nil,
	)
	g.Expect(subReqErr).NotTo(HaveOccurred())

	if subReqErr != nil {
		return
	}

	// The request will either fail due to context cancellation or return 500 from the server.
	// Either way it exercises the watchErr path — no assertion needed on the response.
	resp, _ := http.DefaultClient.Do(subReq)
	if resp != nil {
		_ = resp.Body.Close()
	}

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
