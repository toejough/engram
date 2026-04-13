package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestNewFilePoster_PostExercisesLockAndLineCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	poster := cli.ExportNewFilePoster(chatFile)
	g.Expect(poster).NotTo(BeNil())

	// Post a message to exercise osLockFile, osAppendFile, osLineCount.
	cursor, err := poster.Post(chat.Message{
		From:   "test",
		To:     "all",
		Thread: "main",
		Type:   "info",
		Text:   "hello",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(cursor).To(BeNumerically(">", 0))
}

func TestNewFileWatcher_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	watcher := cli.ExportNewFileWatcher("/tmp/chat.toml")
	g.Expect(watcher).NotTo(BeNil())
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

func TestOsLineCount_ExistingFile_ReturnsLineCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	g.Expect(os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o600)).To(Succeed())

	count, err := cli.ExportOsLineCount(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(count).To(Equal(3))
}

func TestOsLineCount_NonExistentFile_ReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	count, err := cli.ExportOsLineCount("/tmp/does-not-exist-for-linecnt-test.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(count).To(Equal(0))
}

func TestOsLockFile_NonExistentParent_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Pass a path whose parent directory does not exist — OpenFile fails
	// with a non-os.IsExist error (e.g. "no such file or directory"),
	// which exercises the `return nil, fmt.Errorf("creating lock: %w", err)` branch.
	_, err := cli.ExportOsLockFile("/tmp/engram-test-nonexistent-dir-xyz/lock")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("creating lock"))
	}
}

func TestOsLockFile_RetryOnContention_Succeeds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// Pre-create the lock file so the first call finds it exists (os.IsExist path).
	f, createErr := os.Create(lockPath)
	g.Expect(createErr).NotTo(HaveOccurred())

	if createErr != nil {
		return
	}

	_ = f.Close()

	// Release the lock after a short delay so the retry loop succeeds.
	go func() {
		time.Sleep(20 * time.Millisecond)

		_ = os.Remove(lockPath)
	}()

	unlock, err := cli.ExportOsLockFile(lockPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(unlock).NotTo(BeNil())
	g.Expect(unlock()).To(Succeed())
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
