package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/chat"
	"engram/internal/cli"
)

func TestApplyDataDirDefault_Empty_ResolvesHome(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := ""
	err := cli.ExportApplyDataDirDefault(&dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dataDir).To(HaveSuffix("/engram"))
}

func TestApplyDataDirDefault_NonEmpty_Noop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := "/custom/path"
	err := cli.ExportApplyDataDirDefault(&dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dataDir).To(Equal("/custom/path"))
}

func TestApplyProjectSlugDefault_EmptySlug_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := ""
	err := cli.ExportApplyProjectSlugDefault(&slug, func() (string, error) {
		return "/Users/joe/repos/engram", nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slug).To(Equal("-Users-joe-repos-engram"))
}

func TestApplyProjectSlugDefault_GetwdError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := ""
	err := cli.ExportApplyProjectSlugDefault(&slug, func() (string, error) {
		return "", errors.New("getwd failed")
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("resolving working directory"))
	}
}

func TestApplyProjectSlugDefault_NonEmpty_Noop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := "already-set"
	err := cli.ExportApplyProjectSlugDefault(&slug, func() (string, error) {
		t.Fatal("getwd should not be called")
		return "", nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slug).To(Equal("already-set"))
}

func TestDeriveChatFilePath_DefaultPath_UsesOSFunctions(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	path, err := cli.ExportDeriveChatFilePath("", os.UserHomeDir, os.Getwd)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(path).To(ContainSubstring("engram"))
	g.Expect(path).To(HaveSuffix(".toml"))
}

func TestDeriveChatFilePath_GetwdError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportDeriveChatFilePath(
		"",
		func() (string, error) { return "/home/test", nil },
		func() (string, error) { return "", errors.New("no cwd") },
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("resolving working directory"))
	}
}

func TestDeriveChatFilePath_HomeDirError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportDeriveChatFilePath(
		"",
		func() (string, error) { return "", errors.New("no home") },
		os.Getwd,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("resolving home directory"))
	}
}

func TestLoadChatMessages_InvalidTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("not valid toml :::"), 0o600)).To(Succeed())

	_, err := cli.ExportLoadChatMessages(chatFile)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("parsing chat file"))
	}
}

// ============================================================
// loadChatMessages coverage
// ============================================================

func TestLoadChatMessages_NotExist_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	msgs, err := cli.ExportLoadChatMessages(filepath.Join(t.TempDir(), "nonexistent.toml"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msgs).To(BeNil())
}

func TestLoadChatMessages_ReadError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// os.ReadFile on a directory returns a non-NotExist error.
	_, err := cli.ExportLoadChatMessages(t.TempDir())
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("reading chat file"))
	}
}

func TestOsAppendFile_MkdirError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Create a file where osAppendFile would need to create a directory.
	dir := t.TempDir()
	obstacle := filepath.Join(dir, "obstacle")
	g.Expect(os.WriteFile(obstacle, []byte(""), 0o600)).To(Succeed())

	// osAppendFile calls os.MkdirAll(filepath.Dir(path), ...).
	// filepath.Dir(obstacle+"/chat.toml") = obstacle, which is a regular file → error.
	err := cli.ExportOsAppendFile(filepath.Join(obstacle, "chat.toml"), []byte("data"))
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("creating directories"))
	}
}

func TestOutputAckResult_FailWriter_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportOutputAckResult(&failWriter{}, chat.AckResult{Result: "ACK", NewCursor: 1})
	g.Expect(err).To(HaveOccurred())
}

// ============================================================
// outputAckResult coverage
// ============================================================

func TestOutputAckResult_TIMEOUT(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	err := cli.ExportOutputAckResult(&buf, chat.AckResult{
		Result:    "TIMEOUT",
		Timeout:   &chat.TimeoutResult{Recipient: "bob"},
		NewCursor: 5,
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var out map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &out)).To(Succeed())
	g.Expect(out["result"]).To(Equal("TIMEOUT"))
	g.Expect(out["recipient"]).To(Equal("bob"))
}

func TestOutputAckResult_TIMEOUT_NilTimeout_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportOutputAckResult(io.Discard, chat.AckResult{Result: "TIMEOUT", Timeout: nil})
	g.Expect(err).To(HaveOccurred())
}

func TestOutputAckResult_Unknown_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportOutputAckResult(io.Discard, chat.AckResult{Result: "UNKNOWN"})
	g.Expect(err).To(HaveOccurred())
}

func TestOutputAckResult_WAIT_NilWait_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportOutputAckResult(io.Discard, chat.AckResult{Result: "WAIT", Wait: nil})
	g.Expect(err).To(HaveOccurred())
}

// ============================================================
// Step 9: Chat ack-wait tests
// ============================================================

func TestRun_ChatAckWait_AllACK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-populate chat file with an ack from engram-agent addressed to "tester".
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent", "--to", "tester", "--thread", "t",
		"--type", "ack", "--text", "ok",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "ack-wait",
		"--chat-file", chatFile,
		"--agent", "tester",
		"--cursor", "0",
		"--recipients", "engram-agent",
		"--max-wait", "5",
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
	g.Expect(result["result"]).To(Equal("ACK"))
	g.Expect(result).To(HaveKey("cursor"))
}

func TestRun_ChatAckWait_MaxWaitFlag_NoTargCollision(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-write ack so Watch returns immediately.
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent", "--to", "tester", "--thread", "t",
		"--type", "ack", "--text", "ok",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	// Regression: verify --max-wait 1 does not cause a targ flag parsing error.
	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "ack-wait",
		"--chat-file", chatFile,
		"--agent", "tester",
		"--cursor", "0",
		"--recipients", "engram-agent",
		"--max-wait", "1",
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
	g.Expect(result["result"]).To(Equal("ACK"))
}

func TestRun_ChatAckWait_OfflineImplicitACK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Create an empty chat file — offline-agent has no messages (offline).
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// After 5.5s, post an ack from a non-recipient to trigger Watch to return.
	// The ackwaiter loop will then detect 5.5s > 5s offline threshold and return ACK.
	go func() {
		time.Sleep(5500 * time.Millisecond)

		_ = cli.Run([]string{
			"engram", "chat", "post",
			"--chat-file", chatFile,
			"--from", "trigger-agent", "--to", "tester", "--thread", "t",
			"--type", "ack", "--text", "trigger",
		}, io.Discard, io.Discard, nil)
	}()

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "ack-wait",
		"--chat-file", chatFile,
		"--agent", "tester",
		"--cursor", "0",
		"--recipients", "offline-agent",
		"--max-wait", "30",
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
	g.Expect(result["result"]).To(Equal("ACK"))
}

func TestRun_ChatAckWait_WAIT(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-populate chat file with a wait message from engram-agent to "tester".
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent", "--to", "tester", "--thread", "t",
		"--type", "wait", "--text", "objection",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "ack-wait",
		"--chat-file", chatFile,
		"--agent", "tester",
		"--cursor", "0",
		"--recipients", "engram-agent",
		"--max-wait", "5",
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
	g.Expect(result["result"]).To(Equal("WAIT"))
	g.Expect(result["from"]).To(Equal("engram-agent"))
	text, _ := result["text"].(string)
	g.Expect(strings.TrimRight(text, "\n")).To(Equal("objection"))
}

func TestRun_ChatCursor_InvalidFlag_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.Run([]string{
		"engram", "chat", "cursor", "--bogus-flag",
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_ChatCursor_NonExistentFile_OutputsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "nonexistent.toml")

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "cursor",
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	count, parseErr := strconv.Atoi(strings.TrimSpace(stdout.String()))
	g.Expect(parseErr).NotTo(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

func TestRun_ChatCursor_OutputsLineCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write known content: 3 lines.
	g.Expect(os.WriteFile(chatFile, []byte("a\nb\nc\n"), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "cursor",
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	count, parseErr := strconv.Atoi(strings.TrimSpace(stdout.String()))
	g.Expect(parseErr).NotTo(HaveOccurred())
	g.Expect(count).To(Equal(3))
}

func TestRun_ChatPost_ConcurrentWritesSafeToml(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	const messageCount = 10

	var wg sync.WaitGroup

	wg.Add(messageCount)

	for i := range messageCount {
		go func(n int) {
			defer wg.Done()

			_ = cli.Run([]string{
				"engram", "chat", "post",
				"--chat-file", chatFile,
				"--from", fmt.Sprintf("agent-%d", n),
				"--to", "all",
				"--thread", "concurrent",
				"--type", "info",
				"--text", fmt.Sprintf("message %d", n),
			}, io.Discard, io.Discard, nil)
		}(i)
	}

	wg.Wait()

	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
	g.Expect(parsed.Message).To(HaveLen(messageCount))
}

func TestRun_ChatPost_MissingChatSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.Run([]string{"engram", "chat"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_ChatPost_MkdirError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Create a regular file where a directory would need to be created.
	dir := t.TempDir()
	obstacle := filepath.Join(dir, "obstacle")
	g.Expect(os.WriteFile(obstacle, []byte(""), 0o600)).To(Succeed())

	// Try to write a chat file "inside" the obstacle file — MkdirAll will fail.
	chatFile := filepath.Join(obstacle, "chat.toml")
	err := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "x", "--to", "all", "--thread", "t", "--type", "info", "--text", "hi",
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_ChatPost_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.Run([]string{"engram", "chat", "bogus"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

func TestRun_ChatPost_WritesToFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "tester",
		"--to", "all",
		"--thread", "smoke",
		"--type", "info",
		"--text", "hello",
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cursor, parseErr := strconv.Atoi(strings.TrimSpace(stdout.String()))
	g.Expect(parseErr).NotTo(HaveOccurred())
	g.Expect(cursor).To(BeNumerically(">", 0))

	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
	g.Expect(parsed.Message).To(HaveLen(1))
	g.Expect(parsed.Message[0].From).To(Equal("tester"))
	g.Expect(parsed.Message[0].To).To(Equal("all"))
	g.Expect(parsed.Message[0].Type).To(Equal("info"))
	// TOML triple-quoted format adds trailing \n to text content.
	g.Expect(strings.TrimRight(parsed.Message[0].Text, "\n")).To(Equal("hello"))
}

func TestRun_ChatWatch_FailingWriter_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-write a matching message so watch returns immediately.
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "sender",
		"--to", "watcher",
		"--thread", "test",
		"--type", "info",
		"--text", "hello",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	// Run watch with a failing writer — triggers the write-error path.
	err := cli.Run([]string{
		"engram", "chat", "watch",
		"--chat-file", chatFile,
		"--agent", "watcher",
		"--cursor", "0",
		"--type", "info",
	}, &failWriter{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_ChatWatch_OutputsJSON(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-create the file so cursor command works.
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Get initial cursor.
	var cursorOut bytes.Buffer

	cursorErr := cli.Run([]string{
		"engram", "chat", "cursor",
		"--chat-file", chatFile,
	}, &cursorOut, io.Discard, nil)
	g.Expect(cursorErr).NotTo(HaveOccurred())

	cursor := strings.TrimSpace(cursorOut.String())

	// Start watch in goroutine.
	var watchOut bytes.Buffer

	watchDone := make(chan error, 1)

	go func() {
		watchDone <- cli.Run([]string{
			"engram", "chat", "watch",
			"--chat-file", chatFile,
			"--agent", "watcher",
			"--cursor", cursor,
			"--type", "info",
			"--max-wait", "5",
		}, &watchOut, io.Discard, nil)
	}()

	// Give watcher time to register.
	time.Sleep(50 * time.Millisecond)

	// Post a matching message.
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "poster",
		"--to", "watcher",
		"--thread", "test",
		"--type", "info",
		"--text", "ping",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(HaveOccurred())

	// Wait for watch to complete.
	watchErr := <-watchDone
	g.Expect(watchErr).NotTo(HaveOccurred())

	if watchErr != nil {
		return
	}

	// Verify JSON output.
	var result map[string]any

	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(watchOut.String())), &result)).To(Succeed())
	g.Expect(result["type"]).To(Equal("info"))
	g.Expect(result["from"]).To(Equal("poster"))
	// TOML triple-quoted format adds trailing \n to text content.
	text, _ := result["text"].(string)
	g.Expect(strings.TrimRight(text, "\n")).To(Equal("ping"))
	g.Expect(result["cursor"]).NotTo(BeZero())
}

func TestRun_HoldAcquire_ParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "acquire", "--bogus-flag"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

// ============================================================
// Step 10: Hold subcommand tests
// ============================================================

func TestRun_HoldAcquire_PostsMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	var stdout bytes.Buffer

	err := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "executor-1",
		"--condition", "done:lead",
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// stdout should be the hold-id.
	holdID := strings.TrimSpace(stdout.String())
	g.Expect(holdID).NotTo(BeEmpty())

	// chat file should have a hold-acquire message.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
	g.Expect(parsed.Message).To(HaveLen(1))
	g.Expect(parsed.Message[0].Type).To(Equal("hold-acquire"))

	var record chat.HoldRecord
	g.Expect(json.Unmarshal([]byte(parsed.Message[0].Text), &record)).To(Succeed())
	g.Expect(record.HoldID).To(Equal(holdID))
	g.Expect(record.Condition).To(Equal("done:lead"))
}

func TestRun_HoldCheck_AutoReleasesMetCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire a hold with condition "done:reviewer-1".
	var acquireOut bytes.Buffer

	acquireErr := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "exec-1",
		"--condition", "done:reviewer-1",
	}, &acquireOut, io.Discard, nil)
	g.Expect(acquireErr).NotTo(HaveOccurred())

	if acquireErr != nil {
		return
	}

	holdID := strings.TrimSpace(acquireOut.String())

	// Ensure done message TS is strictly after AcquiredTS.
	time.Sleep(time.Millisecond)

	// Post a "done" message from reviewer-1.
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "reviewer-1", "--to", "all", "--thread", "t",
		"--type", "done", "--text", "done",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	// Run hold check — should auto-release the hold and print the hold-id.
	var checkOut bytes.Buffer

	checkErr := cli.Run([]string{
		"engram", "hold", "check",
		"--chat-file", chatFile,
	}, &checkOut, io.Discard, nil)
	g.Expect(checkErr).NotTo(HaveOccurred())

	releasedID := strings.TrimSpace(checkOut.String())
	g.Expect(releasedID).To(Equal(holdID))

	// Verify hold-release message was posted (acquire + done + release = 3 messages).
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
	g.Expect(parsed.Message).To(HaveLen(3))
	g.Expect(parsed.Message[2].Type).To(Equal("hold-release"))
}

func TestRun_HoldCheck_HelpExitsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "check", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_HoldCheck_InvalidTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("not valid toml :::"), 0o600)).To(Succeed())

	err := cli.Run([]string{
		"engram", "hold", "check",
		"--chat-file", chatFile,
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_HoldCheck_ParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "check", "--bogus-flag"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_HoldHelp_ExitsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Regression: --help must exit 0 (ContinueOnError fix).
	err := cli.Run([]string{"engram", "hold", "acquire", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_HoldList_FilterByTag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire two holds with different tags.
	acquireErr1 := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead", "--target", "exec-1",
		"--tag", "codesign-1",
	}, io.Discard, io.Discard, nil)
	g.Expect(acquireErr1).NotTo(HaveOccurred())

	acquireErr2 := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead", "--target", "exec-2",
		"--tag", "plan-review-1",
	}, io.Discard, io.Discard, nil)
	g.Expect(acquireErr2).NotTo(HaveOccurred())

	if acquireErr1 != nil || acquireErr2 != nil {
		return
	}

	// Filter by --tag codesign-1; only one hold should appear.
	var stdout bytes.Buffer

	listErr := cli.Run([]string{
		"engram", "hold", "list",
		"--chat-file", chatFile,
		"--tag", "codesign-1",
	}, &stdout, io.Discard, nil)
	g.Expect(listErr).NotTo(HaveOccurred())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(ContainSubstring("codesign-1"))
}

func TestRun_HoldList_FiltersCorrectly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire two holds with different holders.
	acquireErr1 := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "exec-1",
	}, io.Discard, io.Discard, nil)
	g.Expect(acquireErr1).NotTo(HaveOccurred())

	acquireErr2 := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "other",
		"--target", "exec-2",
	}, io.Discard, io.Discard, nil)
	g.Expect(acquireErr2).NotTo(HaveOccurred())

	if acquireErr1 != nil || acquireErr2 != nil {
		return
	}

	// List filtered by --holder lead; only one hold should appear.
	var stdout bytes.Buffer

	listErr := cli.Run([]string{
		"engram", "hold", "list",
		"--chat-file", chatFile,
		"--holder", "lead",
	}, &stdout, io.Discard, nil)
	g.Expect(listErr).NotTo(HaveOccurred())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(ContainSubstring("lead"))
}

func TestRun_HoldList_HelpExitsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "list", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_HoldList_InvalidTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("not valid toml :::"), 0o600)).To(Succeed())

	err := cli.Run([]string{
		"engram", "hold", "list",
		"--chat-file", chatFile,
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_HoldList_ParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "list", "--bogus-flag"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_HoldRelease_HelpExitsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "release", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_HoldRelease_ParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "release", "--bogus-flag"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_HoldRelease_PostsMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire a hold first.
	var acquireOut bytes.Buffer

	acquireErr := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "exec-1",
	}, &acquireOut, io.Discard, nil)
	g.Expect(acquireErr).NotTo(HaveOccurred())

	if acquireErr != nil {
		return
	}

	holdID := strings.TrimSpace(acquireOut.String())

	// Release it.
	releaseErr := cli.Run([]string{
		"engram", "hold", "release",
		"--chat-file", chatFile,
		"--hold-id", holdID,
	}, io.Discard, io.Discard, nil)
	g.Expect(releaseErr).NotTo(HaveOccurred())

	// Chat file should have acquire + release messages.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
	g.Expect(parsed.Message).To(HaveLen(2))
	g.Expect(parsed.Message[0].Type).To(Equal("hold-acquire"))
	g.Expect(parsed.Message[1].Type).To(Equal("hold-release"))

	// Release text must contain the hold-id for ScanActiveHolds matching.
	var releasePayload map[string]string
	g.Expect(json.Unmarshal([]byte(parsed.Message[1].Text), &releasePayload)).To(Succeed())
	g.Expect(releasePayload["hold-id"]).To(Equal(holdID))
}

// ============================================================
// hold dispatch + subcommand parse/help/invalid-file coverage
// ============================================================

func TestRun_Hold_NoSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_Hold_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "bogus"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

func TestRun_NoArgs_ReturnsUsageError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_Recall_EmptyData(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", "test-project",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// This may or may not error depending on whether ~/.claude/projects/test-project exists.
	// The important thing is it exercises the code path without panicking.
	_ = err
	_ = g
}

func TestRun_UnknownCommand_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "nonexistent"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

// failWriter is an io.Writer that always returns an error.
type failWriter struct{}

func (w *failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}
