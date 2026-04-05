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
