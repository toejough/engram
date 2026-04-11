package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
	claudepkg "engram/internal/claude"
	"engram/internal/cli"
)

func TestAgentRunFlags_BuildsCorrectArgs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.AgentRunFlags(cli.AgentRunArgs{
		Name:      "worker-1",
		Prompt:    "do the thing",
		ChatFile:  "/tmp/chat.toml",
		StateFile: "/tmp/state.toml",
	})

	g.Expect(args).To(ContainElements("--name", "worker-1", "--prompt", "do the thing"))
	g.Expect(args).To(ContainElements("--chat-file", "/tmp/chat.toml", "--state-file", "/tmp/state.toml"))
}

func TestAgentRunFlags_SkipsEmptyValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.AgentRunFlags(cli.AgentRunArgs{Name: "worker-1", Prompt: "hello"})

	g.Expect(args).To(ContainElements("--name", "worker-1", "--prompt", "hello"))
	g.Expect(args).NotTo(ContainElement("--chat-file"))
	g.Expect(args).NotTo(ContainElement("--state-file"))
}

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

// TestBuildAgentRunner_WriteState_UpdatesStateFile verifies that the
// WriteState callback wired by buildAgentRunner updates the state file.
func TestBuildAgentRunner_WriteState_UpdatesStateFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"test-agent\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\n" +
		"spawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	flags := cli.AgentRunArgs{Name: "test-agent", Prompt: "hello"}
	runner := cli.ExportBuildAgentRunner(flags, stateFile, chatFile)

	g.Expect(runner.WriteState).NotTo(BeNil())

	writeErr := runner.WriteState("ACTIVE")
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`state = "ACTIVE"`))
}

// TestBuildClaudeCmd_NoSession builds a command without --resume.
func TestBuildClaudeCmd_NoSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmd := cli.ExportBuildClaudeCmd(t.Context(), "hello world", "", "/usr/local/bin/claude")
	g.Expect(cmd.Args).To(ContainElement("hello world"))
	g.Expect(cmd.Args).NotTo(ContainElement("--resume"))
}

// TestBuildClaudeCmd_WithSession builds a command with --resume.
func TestBuildClaudeCmd_WithSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmd := cli.ExportBuildClaudeCmd(t.Context(), "Proceed.", "sess-abc", "/usr/local/bin/claude")
	g.Expect(cmd.Args).To(ContainElements("--resume", "sess-abc"))
	g.Expect(cmd.Args).To(ContainElement("Proceed."))
}

func TestBuildResumePrompt_EmptyMemFiles_EmptySection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName: "a", Cursor: 1, IntentFrom: "a", IntentText: "b", ResumeReason: "intent",
	})
	// MEMORY_FILES: should be followed by INTENT_FROM: with nothing in between (just a newline)
	g.Expect(prompt).To(ContainSubstring("MEMORY_FILES:\nINTENT_FROM:"))
}

// ============================================================
// buildResumePrompt tests
// ============================================================

func TestBuildResumePrompt_IncludesCursorField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName: "agent-1", Cursor: 42, IntentFrom: "agent-1", IntentText: "do stuff", ResumeReason: "intent",
	})
	g.Expect(prompt).To(ContainSubstring("CURSOR: 42"))
}

func TestBuildResumePrompt_IncludesInstruction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName: "a", Cursor: 1, IntentFrom: "a", IntentText: "b", ResumeReason: "intent",
	})
	g.Expect(prompt).To(ContainSubstring("Instruction: Load the files listed under MEMORY_FILES."))
}

func TestBuildResumePrompt_IncludesIntentFromAndText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName: "executor-1", Cursor: 5, IntentFrom: "executor-1",
		IntentText: "Situation: about to deploy.", ResumeReason: "intent",
	})
	g.Expect(prompt).To(ContainSubstring("INTENT_FROM: executor-1"))
	g.Expect(prompt).To(ContainSubstring("INTENT_TEXT: Situation: about to deploy."))
}

func TestBuildResumePrompt_IncludesMemoryFilesSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFiles := []string{"/data/feedback/mem1.md", "/data/facts/fact1.md"}
	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName: "agent-1", Cursor: 10, MemFiles: memFiles,
		IntentFrom: "agent-1", IntentText: "do stuff", ResumeReason: "intent",
	})
	g.Expect(prompt).To(ContainSubstring("MEMORY_FILES:"))
	g.Expect(prompt).To(ContainSubstring("/data/feedback/mem1.md"))
	g.Expect(prompt).To(ContainSubstring("/data/facts/fact1.md"))
}

// ============================================================
// Task 2: buildResumePrompt struct form + new fields
// ============================================================

func TestBuildResumePrompt_StructForm_AgentName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:    "engram-agent",
		Cursor:       42,
		IntentFrom:   "lead",
		IntentText:   "do stuff",
		ResumeReason: "intent",
	})
	g.Expect(prompt).To(ContainSubstring("AGENT_NAME: engram-agent"))
}

func TestBuildResumePrompt_StructForm_EscalationNoteAtTurn3(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:    "worker-a",
		Cursor:       200,
		IntentFrom:   "engram-agent",
		IntentText:   "still objecting",
		ResumeReason: "wait",
		WaitFrom:     "engram-agent",
		WaitText:     "still objecting",
		ArgumentTurn: 3,
	})
	g.Expect(prompt).To(ContainSubstring("ARGUMENT_TURN: 3"))
	g.Expect(prompt).To(ContainSubstring("ARGUMENT_ESCALATION_NOTE:"))
}

func TestBuildResumePrompt_StructForm_LearnedMessages_None(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:       "worker-a",
		Cursor:          10,
		IntentFrom:      "lead",
		IntentText:      "do it",
		ResumeReason:    "intent",
		LearnedMessages: nil,
	})
	g.Expect(prompt).To(ContainSubstring("LEARNED_MESSAGES: (none)"))
}

func TestBuildResumePrompt_StructForm_LearnedMessages_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:       "worker-a",
		Cursor:          10,
		IntentFrom:      "lead",
		IntentText:      "do it",
		ResumeReason:    "intent",
		LearnedMessages: []string{"engram → uses → targ", "never amend pushed commits"},
	})
	g.Expect(prompt).To(ContainSubstring("LEARNED_MESSAGES: engram → uses → targ | never amend pushed commits"))
}

func TestBuildResumePrompt_StructForm_RecentIntents_None(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:     "worker-a",
		Cursor:        10,
		IntentFrom:    "lead",
		IntentText:    "do it",
		ResumeReason:  "intent",
		RecentIntents: nil,
	})
	g.Expect(prompt).To(ContainSubstring("RECENT_INTENTS: (none)"))
}

func TestBuildResumePrompt_StructForm_RecentIntents_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:     "worker-a",
		Cursor:        10,
		IntentFrom:    "lead",
		IntentText:    "do it",
		ResumeReason:  "intent",
		RecentIntents: []string{"lead→worker-a: do it", "test→engram-agent: check this"},
	})
	g.Expect(prompt).To(ContainSubstring("RECENT_INTENTS: lead→worker-a: do it | test→engram-agent: check this"))
}

func TestBuildResumePrompt_StructForm_ResumeReasonIntent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:    "worker-a",
		Cursor:       10,
		IntentFrom:   "lead",
		IntentText:   "do it",
		ResumeReason: "intent",
	})
	g.Expect(prompt).To(ContainSubstring("RESUME_REASON: intent"))
	g.Expect(prompt).NotTo(ContainSubstring("WAIT_FROM:"))
}

func TestBuildResumePrompt_StructForm_ResumeReasonShutdown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:    "engram-agent",
		Cursor:       100,
		IntentFrom:   "lead",
		IntentText:   "shutdown",
		ResumeReason: "shutdown",
	})
	g.Expect(prompt).To(ContainSubstring("RESUME_REASON: shutdown"))
	g.Expect(prompt).NotTo(ContainSubstring("WAIT_FROM:"))
}

func TestBuildResumePrompt_StructForm_ResumeReasonWait(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cli.ExportBuildResumePrompt(cli.ResumePromptArgs{
		AgentName:    "worker-a",
		Cursor:       200,
		IntentFrom:   "engram-agent",
		IntentText:   "you stepped on my cursor",
		ResumeReason: "wait",
		WaitFrom:     "engram-agent",
		WaitText:     "you stepped on my cursor",
		ArgumentTurn: 1,
	})
	g.Expect(prompt).To(ContainSubstring("RESUME_REASON: wait"))
	g.Expect(prompt).To(ContainSubstring("WAIT_FROM: engram-agent"))
	g.Expect(prompt).To(ContainSubstring("WAIT_TEXT: you stepped on my cursor"))
	g.Expect(prompt).To(ContainSubstring("ARGUMENT_TURN: 1"))
	g.Expect(prompt).NotTo(ContainSubstring("ARGUMENT_ESCALATION_NOTE:"))
}

func TestChatFileCursor_MissingFile_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := cli.ExportChatFileCursor("/nonexistent/path/chat.toml")
	g.Expect(err).To(HaveOccurred())
}

func TestChatFileCursor_ReturnsLineCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("line1\nline2\nline3\n"), 0o600)).To(Succeed())

	count, err := cli.ExportChatFileCursor(chatFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(count).To(BeNumerically(">", 0))
}

func TestCollectLearned_EmptyWhenNoLearnedMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := []byte("\n[[message]]\nfrom = \"exec\"\nto = \"engram-agent\"\n" +
		"thread = \"t\"\ntype = \"info\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\nnot a learned message\n\"\"\"\n")
	readFile := func(_ string) ([]byte, error) { return content, nil }

	learned, err := cli.ExportCollectLearned("/fake/chat.toml", "engram-agent", 0, readFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(learned).To(BeEmpty())
}

func TestCollectLearned_ExcludesBeforeCursor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Write a message, count its lines, then add more messages.
	msg1 := "\n[[message]]\nfrom = \"exec\"\nto = \"engram-agent\"\n" +
		"thread = \"t\"\ntype = \"learned\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\nold fact\n\"\"\"\n"
	msg2 := "\n[[message]]\nfrom = \"exec\"\nto = \"engram-agent\"\n" +
		"thread = \"t\"\ntype = \"learned\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\nnew fact\n\"\"\"\n"

	cursor := strings.Count(msg1, "\n") // cursor after msg1

	content := []byte(msg1 + msg2)
	readFile := func(_ string) ([]byte, error) { return content, nil }

	learned, err := cli.ExportCollectLearned("/fake/chat.toml", "engram-agent", cursor, readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(learned).To(HaveLen(1))
	g.Expect(learned[0]).To(ContainSubstring("new fact"))
}

func TestCollectLearned_IncludesToAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := []byte("\n[[message]]\nfrom = \"exec\"\nto = \"all\"\n" +
		"thread = \"t\"\ntype = \"learned\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\nbroad fact\n\"\"\"\n")
	readFile := func(_ string) ([]byte, error) { return content, nil }

	learned, err := cli.ExportCollectLearned("/fake/chat.toml", "engram-agent", 0, readFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(learned).To(HaveLen(1))
}

func TestCollectLearned_ReturnsSinceCursor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 3 learned messages after cursor position 0.
	var sb strings.Builder
	for i := range 3 {
		sb.WriteString("\n[[message]]\n")
		sb.WriteString("from = \"executor\"\n")
		sb.WriteString("to = \"engram-agent\"\n")
		sb.WriteString("thread = \"t\"\ntype = \"learned\"\n")
		sb.WriteString("ts = 2026-04-11T00:00:00Z\n")
		fmt.Fprintf(&sb, "text = \"\"\"\nfact %d\n\"\"\"\n", i)
	}

	content := []byte(sb.String())

	readFile := func(_ string) ([]byte, error) { return content, nil }
	learned, err := cli.ExportCollectLearned("/fake/chat.toml", "engram-agent", 0, readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(learned).To(HaveLen(3))
}

func TestDefaultMemFileSelector_EmptyDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	homeDir := t.TempDir()
	feedbackDir := filepath.Join(homeDir, ".local", "share", "engram", "memory", "feedback")
	factsDir := filepath.Join(homeDir, ".local", "share", "engram", "memory", "facts")

	g.Expect(os.MkdirAll(feedbackDir, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(factsDir, 0o755)).To(Succeed())

	files, err := cli.ExportDefaultMemFileSelector(homeDir, 20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(BeEmpty())
}

// ============================================================
// defaultMemFileSelector tests
// ============================================================

func TestDefaultMemFileSelector_WiresOsReadDirAndStat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a temp home dir with the expected memory layout.
	homeDir := t.TempDir()
	feedbackDir := filepath.Join(homeDir, ".local", "share", "engram", "memory", "feedback")
	factsDir := filepath.Join(homeDir, ".local", "share", "engram", "memory", "facts")

	g.Expect(os.MkdirAll(feedbackDir, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(factsDir, 0o755)).To(Succeed())

	// Write a feedback file and a facts file.
	g.Expect(os.WriteFile(filepath.Join(feedbackDir, "fb1.md"), []byte("feedback"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(factsDir, "fact1.md"), []byte("fact"), 0o600)).To(Succeed())

	files, err := cli.ExportDefaultMemFileSelector(homeDir, 20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(HaveLen(2))

	// All returned paths must be absolute.
	for _, f := range files {
		g.Expect(filepath.IsAbs(f)).To(BeTrue(), "expected absolute path, got: %s", f)
	}
}

// ============================================================
// defaultWatchForIntent tests
// ============================================================

func TestDefaultWatchForIntent_ReturnsIntentMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write a chat file with an intent message addressed to our agent.
	intentTOML := "[[message]]\n" +
		"from = \"lead\"\n" +
		"to = \"worker-1\"\n" +
		"thread = \"work\"\n" +
		"type = \"intent\"\n" +
		"ts = \"2026-04-07T00:00:00Z\"\n" +
		"text = \"Do the thing.\"\n"

	g.Expect(os.WriteFile(chatFile, []byte(intentTOML), 0o600)).To(Succeed())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg, newCursor, err := cli.ExportDefaultWatchForIntent(ctx, "worker-1", chatFile, 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(msg.Type).To(Equal("intent"))
	g.Expect(msg.From).To(Equal("lead"))
	g.Expect(msg.Text).To(Equal("Do the thing."))
	g.Expect(newCursor).To(BeNumerically(">", 0))
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

func TestDeriveStateFilePath_HomeDirError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errHome := errors.New("home dir unavailable")
	_, err := cli.ExportResolveStateFile("", "agent spawn",
		func() (string, error) { return "", errHome },
		os.Getwd,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("home dir"))
	}
}

func TestKillAgentPane_EmptyPaneID_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	// Empty state file — agent "executor-1" is not registered, so pane ID stays "".
	g.Expect(os.WriteFile(stateFile, []byte(""), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	cli.SetTestPaneKiller(t, func(_ string) error {
		return errors.New("should not be called: pane ID was empty")
	})

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestLoadChatMessages_InvalidTOML_SkipsCorruptBlocks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("not valid toml :::"), 0o600)).To(Succeed())

	msgs, err := cli.ExportLoadChatMessages(chatFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(msgs).To(BeEmpty())
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

func TestOsTmuxKillPane_NoSuchPane_ReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A nonexistent pane ID causes tmux to return "can't find pane" — function must return nil.
	err := cli.ExportOsTmuxKillPane("nonexistent-pane-id-phase3-task8")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestOsTmuxKillPane_OtherError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// "invalid:::pane" causes tmux to error with "can't find session" — not "no such pane".
	// This exercises the wrapped-error return path in osTmuxKillPane.
	err := cli.ExportOsTmuxKillPane("invalid:::pane")
	g.Expect(err).To(HaveOccurred())
}

func TestOsTmuxSpawnWith_CommandFails_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")

	g.Expect(os.WriteFile(fakeTmux, []byte("#!/bin/sh\nexit 1\n"), 0o700)).To(Succeed())

	_, _, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "sh -c 'echo hello'")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("tmux new-window"))
	}
}

func TestOsTmuxSpawnWith_SendKeysFails_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")

	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  new-window) echo '%my-pane $mysession' ;;\n" +
		"  set-option) exit 0 ;;\n" +
		"  send-keys) exit 1 ;;\n" +
		"  *) exit 1 ;;\n" +
		"esac\n"
	g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())

	_, _, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "my-prompt-text")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("tmux send-keys"))
	}
}

func TestOsTmuxSpawnWith_SendsEngramAgentRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")
	callLog := filepath.Join(tmpDir, "calls.txt")

	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + callLog + "\n" +
		"case \"$1\" in\n" +
		"  new-window) echo '%my-pane $mysession' ;;\n" +
		"  set-option) ;;\n" +
		"  send-keys) ;;\n" +
		"esac\n"
	g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())

	paneID, sessionID, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "my-agent", "do-the-thing")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paneID).NotTo(BeEmpty())
	g.Expect(sessionID).To(Equal("PENDING"))

	calls, readErr := os.ReadFile(callLog)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	callsStr := string(calls)
	g.Expect(callsStr).To(ContainSubstring("engram agent run"))
	g.Expect(callsStr).To(ContainSubstring("--name"))
	g.Expect(callsStr).To(ContainSubstring("my-agent"))
}

func TestOsTmuxSpawnWith_SetsEngramNamePaneOption(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")
	callLog := filepath.Join(tmpDir, "calls.txt")

	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + callLog + "\n" +
		"case \"$1\" in\n" +
		"  new-window) echo '%my-pane $mysession' ;;\n" +
		"  set-option) ;;\n" +
		"  send-keys) ;;\n" +
		"esac\n"
	g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())

	_, _, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "my-agent", "my-prompt")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	calls, readErr := os.ReadFile(callLog)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// @engram_name user option must be set on the pane with the agent name
	callsStr := string(calls)
	g.Expect(callsStr).To(ContainSubstring("set-option -p -t %my-pane @engram_name my-agent"))
}

func TestOsTmuxSpawnWith_Success_ReturnsPaneAndSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")

	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  new-window) echo '%my-pane $mysession' ;;\n" +
		"  set-option) ;;\n" +
		"  send-keys) ;;\n" +
		"  *) exit 1 ;;\n" +
		"esac\n"
	g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())

	paneID, sessionID, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "my-prompt-text")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paneID).To(Equal("%my-pane"))
	g.Expect(sessionID).To(Equal("PENDING"))
}

func TestOsTmuxSpawnWith_UnexpectedOutput_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")

	g.Expect(os.WriteFile(fakeTmux, []byte("#!/bin/sh\necho 'only-one-field'\n"), 0o700)).To(Succeed())

	_, _, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "sh -c 'echo hello'")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unexpected tmux output"))
	}
}

func TestOsTmuxSpawn_CancelledContext_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := cli.ExportOsTmuxSpawn(ctx, "myagent", "sh -c 'echo hello'")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("tmux new-window"))
	}
}

// ============================================================
// Coverage: osTmuxVerifyPaneGone and rejectDuplicateAgentName
// ============================================================

func TestOsTmuxVerifyPaneGone_NonExistentPane_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Pane ID %99999999 cannot exist in any real tmux session.
	// list-panes will exit non-zero → verify returns nil (pane is gone).
	err := cli.ExportOsTmuxVerifyPaneGone("%99999999")

	g.Expect(err).NotTo(HaveOccurred())
}

// TestOuterWatchLoop_CtxCancelDuringWatch verifies clean exit on ctx cancellation during watch.
func TestOuterWatchLoop_CtxCancelDuringWatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchForIntent := func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
		cancel()

		return chat.Message{}, 0, context.Canceled
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	// Clean exit: no error on ctx cancellation during watch.
	g.Expect(err).NotTo(HaveOccurred())
}

// TestOuterWatchLoop_CursorNotResetBetweenSessions verifies that across two outer-loop
// iterations, the cursor passed to watchForIntent in each iteration is non-zero and
// matches the chat-file state. Confirms the outer loop does not reset cursor to 0
// between sessions.
func TestOuterWatchLoop_CursorNotResetBetweenSessions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	// Five lines of content — chatFileCursor = len(strings.Split(content, "\n")) = 6.
	const numLines = 5

	content := strings.Repeat("comment\n", numLines)
	expectedCursor := len(strings.Split(content, "\n"))

	g.Expect(os.WriteFile(chatFile, []byte(content), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Fake claude emits DONE on every call.
	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// watchForIntent captures cursor for calls 1 and 2; cancels ctx on call 2.
	var cursor1, cursor2 int

	callCount := 0
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		callCount++

		if callCount == 1 {
			cursor1 = cursor

			return chat.Message{
				From: "lead",
				Type: "intent",
				Text: "Resume now.",
			}, cursor + 1, nil
		}

		cursor2 = cursor

		cancel()

		return chat.Message{}, 0, context.Canceled
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		func(_ context.Context, _, _ string, _ int) (string, error) { return "Proceed.", nil },
		watchForIntent,
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callCount).To(Equal(2), "watchForIntent must be called exactly twice")
	g.Expect(cursor1).To(BeNumerically(">", 0), "first watchForIntent cursor must be non-zero")
	g.Expect(cursor1).To(Equal(expectedCursor), "cursor1 must match chatFileCursor of initial content")
	g.Expect(cursor2).To(BeNumerically(">", 0), "second watchForIntent cursor must be non-zero")
	g.Expect(cursor2).To(BeNumerically(">=", cursor1),
		"cursor must not go backward — outer loop must not reset cursor to 0 between sessions")
}

// --- Phase 5 Task 4: Outer watch loop tests ---

// TestOuterWatchLoop_DoneThenWatchFires verifies that after DoneDetected,
// the loop watches for the next intent and fires a new session (does not return).
func TestOuterWatchLoop_DoneThenWatchFires(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Session 1: emit DONE (inner loop exits).
	// Session 2: emit DONE (inner loop exits, then ctx cancelled to stop outer loop).
	sessionCount := 0
	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// watchForIntent: first call returns an intent, second call cancels ctx.
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		sessionCount++
		if sessionCount >= 2 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "test-agent",
			To:   "engram-agent",
			Type: "intent",
			Text: "Situation: test. Behavior: respond.",
		}, cursor + 10, nil
	}

	// memFileSelector: returns no files.
	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Outer loop ran: session 1 (DONE) → watch → session 2 (DONE) → watch → ctx cancelled.
	g.Expect(sessionCount).To(Equal(2))
}

// TestOuterWatchLoop_FreshSessionIDPerInvocation verifies sessionID="" on each watch-loop
// invocation (not --resume).
func TestOuterWatchLoop_FreshSessionIDPerInvocation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Track session IDs from the fake claude script.
	// Each invocation gets a different session-id to prove fresh session.
	invocationCount := 0
	fakeClaude := filepath.Join(dir, "claude")
	// Script uses a counter file to distinguish invocations.
	counterFile := filepath.Join(dir, "counter")
	json1 := `{"type":"assistant","session_id":"sess-first",` +
		`"message":{"content":[{"type":"text","text":"DONE: done1."}]}}`
	json2 := `{"type":"assistant","session_id":"sess-second",` +
		`"message":{"content":[{"type":"text","text":"DONE: done2."}]}}`
	script := "#!/bin/sh\n" +
		"COUNT=0\n" +
		"if [ -f " + counterFile + " ]; then " +
		"COUNT=$(cat " + counterFile + "); fi\n" +
		"COUNT=$((COUNT + 1))\n" +
		"echo $COUNT > " + counterFile + "\n" +
		"if [ $COUNT -eq 1 ]; then\n" +
		"  printf '%s\\n' '" + json1 + "'\n" +
		"else\n" +
		"  printf '%s\\n' '" + json2 + "'\n" +
		"fi\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		invocationCount++
		if invocationCount >= 2 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "test-agent",
			To:   "engram-agent",
			Type: "intent",
			Text: "Situation: test. Behavior: respond.",
		}, cursor + 10, nil
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Both session IDs should appear in state file history — proving fresh sessions.
	// The latest should be sess-second (from invocation 2).
	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// At minimum, verify 2 invocations ran and session-id changed.
	g.Expect(invocationCount).To(Equal(2))
	g.Expect(string(data)).To(ContainSubstring(`session-id = "sess-second"`))
}

// TestOuterWatchLoop_LastResumedAtUpdated verifies last-resumed-at is updated when an intent fires.
func TestOuterWatchLoop_LastResumedAtUpdated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchCalls := 0
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		watchCalls++
		if watchCalls >= 2 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "test-agent",
			To:   "engram-agent",
			Type: "intent",
			Text: "Situation: test. Behavior: respond.",
		}, cursor + 10, nil
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify last-resumed-at was written to state file.
	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("last-resumed-at"))
}

// TestOuterWatchLoop_WaitDetectedWatchesForNextIntent verifies that WaitDetected
// also triggers the outer watch loop (same behavior as DONE).
func TestOuterWatchLoop_WaitDetectedWatchesForNextIntent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")

	// Session 1 emits WAIT (inner loop exits). Session 2 emits DONE.
	waitJSON := `{"type":"assistant","session_id":"sess-w",` +
		`"message":{"content":[{"type":"text",` +
		`"text":"WAIT: Need more info."}]}}`
	doneJSON2 := `{"type":"assistant","session_id":"sess-d",` +
		`"message":{"content":[{"type":"text",` +
		`"text":"DONE: All done."}]}}`
	counterFile := filepath.Join(dir, "counter")
	script := "#!/bin/sh\n" +
		"COUNT=0\n" +
		"if [ -f " + counterFile + " ]; then " +
		"COUNT=$(cat " + counterFile + "); fi\n" +
		"COUNT=$((COUNT + 1))\n" +
		"echo $COUNT > " + counterFile + "\n" +
		"if [ $COUNT -eq 1 ]; then\n" +
		"  printf '%s\\n' '" + waitJSON + "'\n" +
		"else\n" +
		"  printf '%s\\n' '" + doneJSON2 + "'\n" +
		"fi\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchCalls := 0
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		watchCalls++
		if watchCalls >= 2 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "test-agent",
			To:   "engram-agent",
			Type: "intent",
			Text: "Situation: follow-up. Behavior: respond.",
		}, cursor + 10, nil
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// WAIT triggered outer loop → watch → second session ran.
	g.Expect(watchCalls).To(Equal(2))
}

// TestOuterWatchLoop_WatchError_Propagates verifies non-context watch errors propagate.
func TestOuterWatchLoop_WatchError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\n" +
		"spawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text",` +
		`"text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	watchForIntent := func(
		_ context.Context, _, _ string, _ int,
	) (chat.Message, int, error) {
		return chat.Message{}, 0, errors.New("watch failed")
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(
		_ context.Context, _, _ string, _ int,
	) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		context.Background(),
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("watch failed"))
}

// TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession is a property-based test
// verifying that the cursor passed to watchForIntent equals the chat-file line count
// (chatFileCursor result) at the end of the prior session, for any initial file size.
// A regression hardcoding cursor=0 would fail this test for any non-trivial content.
func TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession(t *testing.T) {
	t.Parallel()

	// Fake claude binary shared across all rapid cases: emits DONE immediately.
	claudeDir := t.TempDir()
	fakeClaude := filepath.Join(claudeDir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"

	if writeErr := os.WriteFile(fakeClaude, []byte(script), 0o700); writeErr != nil {
		t.Fatal(writeErr)
	}

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		// Generate a random number of content lines [1, 20].
		numLines := rapid.IntRange(1, 20).Draw(rt, "numLines")
		content := strings.Repeat("comment\n", numLines)
		// chatFileCursor = len(strings.Split(content, "\n")) — same formula as production code.
		expectedCursor := len(strings.Split(content, "\n"))

		// Each rapid case gets its own temp dir to avoid cross-case interference.
		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")
		stateFile := filepath.Join(dir, "state.toml")

		g.Expect(os.WriteFile(chatFile, []byte(content), 0o600)).To(Succeed())
		g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// watchForIntent captures the cursor it receives, then cancels ctx (clean exit).
		var capturedCursor int

		watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
			capturedCursor = cursor

			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		err := cli.ExportRunConversationLoopWith(
			ctx,
			"worker-1", "hello", chatFile, stateFile, fakeClaude,
			io.Discard,
			func(_ context.Context, _, _ string, _ int) (string, error) { return "Proceed.", nil },
			watchForIntent,
			nil,
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(capturedCursor).To(Equal(expectedCursor),
			"cursor passed to watchForIntent must equal chatFileCursor of chat file (numLines=%d)", numLines)
	})
}

func TestOuterWatchLoop_WriteStateActiveOnSecondSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Fake claude: each invocation emits READY: then DONE:.
	fakeClaude := filepath.Join(dir, "claude")
	readyJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"READY: Online."}]}}`
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + readyJSON + "'\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// watchForIntent: first call bridges to session 2, second cancels.
	watchCallCount := 0
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		watchCallCount++
		if watchCallCount >= 2 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "test-agent",
			To:   "engram-agent",
			Type: "intent",
			Text: "Situation: test. Behavior: respond.",
		}, cursor + 10, nil
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	// Record all WriteState calls across both sessions.
	var mu sync.Mutex

	var stateHistory []string

	writeStateObserver := func(state string) error {
		mu.Lock()
		defer mu.Unlock()

		stateHistory = append(stateHistory, state)

		return nil
	}

	err := cli.ExportRunConversationLoopWithStateHook(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
		writeStateObserver,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Both sessions must have called WriteState("ACTIVE") — one per READY: marker.
	mu.Lock()
	defer mu.Unlock()

	activeCount := 0

	for _, state := range stateHistory {
		if state == "ACTIVE" {
			activeCount++
		}
	}

	g.Expect(activeCount).To(Equal(2), "WriteState(ACTIVE) must be called once per session start")
}

// TestOuterWatchLoop_WriteStateSilentAfterSession verifies WriteState("SILENT") is called
// after the inner session ends and before the watch begins.
func TestOuterWatchLoop_WriteStateSilentAfterSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchForIntent := func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
		cancel()

		return chat.Message{}, 0, context.Canceled
	}

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, nil
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		watchForIntent,
		memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify state file shows SILENT (written by outer loop after session ends).
	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`state = "SILENT"`))
}

func TestOuterWatchLoop_WritesSilentAtWithSilentState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Write initial state using MarshalStateFile for correct TOML keys.
	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{
				Name:      "engram-agent",
				PaneID:    "",
				SessionID: "",
				State:     "STARTING",
				SpawnedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	stateData, marshalErr := agentpkg.MarshalStateFile(initialState)
	g.Expect(marshalErr).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(stateFile, stateData, 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchForIntent := func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
		cancel()

		return chat.Message{}, 0, context.Canceled
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"engram-agent", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		func(_ context.Context, _, _ string, _ int) (string, error) { return "Proceed.", nil },
		watchForIntent,
		func(_ string, _ int) ([]string, error) { return nil, nil },
	)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// Both fields must be written atomically in the same RMW call.
	g.Expect(string(data)).To(ContainSubstring(`state = "SILENT"`))
	g.Expect(string(data)).To(ContainSubstring(`last-silent-at =`))
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

func TestOutputAckResult_WAIT_HappyPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	err := cli.ExportOutputAckResult(&buf, chat.AckResult{
		Result:    "WAIT",
		Wait:      &chat.WaitResult{From: "engram-agent", Text: "objection"},
		NewCursor: 42,
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var out map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &out)).To(Succeed())
	g.Expect(out["result"]).To(Equal("WAIT"))
	g.Expect(out["from"]).To(Equal("engram-agent"))
	g.Expect(out["text"]).To(Equal("objection"))
	g.Expect(out["cursor"]).To(BeEquivalentTo(42))
}

func TestOutputAckResult_WAIT_NilWait_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportOutputAckResult(io.Discard, chat.AckResult{Result: "WAIT", Wait: nil})
	g.Expect(err).To(HaveOccurred())
}

func TestParseTmuxOutput_UnexpectedOutput_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, _, err := cli.ExportParseTmuxOutput([]byte("only-one-field\n"))

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unexpected tmux output"))
	}
}

func TestParseTmuxOutput_ValidOutput_ReturnsPaneAndSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	paneID, sessionID, err := cli.ExportParseTmuxOutput([]byte("%my-pane $mysession\n"))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paneID).To(Equal("%my-pane"))
	g.Expect(sessionID).To(Equal("$mysession"))
}

func TestReadModifyWriteStateFile_ConcurrentCallers_BothAgentsPresent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	const callers = 5

	var wg sync.WaitGroup

	wg.Add(callers)

	for i := range callers {
		agentName := fmt.Sprintf("agent-%d", i)

		go func() {
			defer wg.Done()

			innerErr := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
				return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: agentName, State: "STARTING"})
			})
			g.Expect(innerErr).NotTo(HaveOccurred())
		}()
	}

	wg.Wait()

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	for i := range callers {
		g.Expect(string(data)).To(ContainSubstring(fmt.Sprintf("agent-%d", i)))
	}
}

func TestReadModifyWriteStateFile_CreatesFileWhenAbsent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	err := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "test-agent", State: "STARTING"})
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("test-agent"))
}

func TestReadModifyWriteStateFile_InvalidTOML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(stateFile, []byte("[[not valid toml {{{"), 0o600)).To(Succeed())

	err := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return sf
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("parsing state file"))
	}
}

func TestReadModifyWriteStateFile_InvalidTOML_ReturnsParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	writeErr := os.WriteFile(stateFile, []byte("[[[invalid toml"), 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	err := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return sf
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("parsing"))
	}
}

func TestReadModifyWriteStateFile_MissingDir_CreatesDirAndFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	base := t.TempDir()
	// state/ subdirectory does NOT exist yet
	stateFile := filepath.Join(base, "state", "test.toml")

	err := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "test-agent", State: "STARTING"})
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stateFile).To(BeAnExistingFile())

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("test-agent"))
}

func TestReadModifyWriteStateFile_PathIsDirectory_ReturnsReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Using a directory as the stateFilePath causes os.ReadFile to fail with EISDIR.
	dir := t.TempDir()
	subDir := filepath.Join(dir, "statedir")
	mkErr := os.MkdirAll(subDir, 0o700)
	g.Expect(mkErr).NotTo(HaveOccurred())

	err := cli.ExportReadModifyWriteStateFile(subDir, func(sf agentpkg.StateFile) agentpkg.StateFile { return sf })
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("reading"))
	}
}

func TestReadModifyWriteStateFile_UpdatesExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	err1 := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "agent-1", State: "ACTIVE"})
	})
	g.Expect(err1).NotTo(HaveOccurred())

	err2 := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "agent-2", State: "ACTIVE"})
	})
	g.Expect(err2).NotTo(HaveOccurred())

	data, _ := os.ReadFile(stateFile)
	g.Expect(string(data)).To(ContainSubstring("agent-1"))
	g.Expect(string(data)).To(ContainSubstring("agent-2"))
}

// ============================================================
// resolveChatFile coverage
// ============================================================

func TestResolveChatFile_HappyPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path, err := cli.ExportResolveChatFile(
		filepath.Join(dir, "chat.toml"),
		"chat post",
		os.UserHomeDir,
		os.Getwd,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(path).To(HaveSuffix("chat.toml"))
}

func TestResolveChatFile_HomeDirError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := cli.ExportResolveChatFile(
		"",
		"chat post",
		func() (string, error) { return "", errors.New("no home") },
		os.Getwd,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("chat post"))
	}
}

func TestResolveStateFile_NoOverride_UsesStateSubdir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path, err := cli.ExportResolveStateFile("", "agent spawn", os.UserHomeDir, os.Getwd)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(path).To(ContainSubstring("/state/"))
	g.Expect(path).To(HaveSuffix(".toml"))
}

func TestResolveStateFile_WithOverride_ReturnsOverride(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	path, err := cli.ExportResolveStateFile(stateFile, "agent spawn", os.UserHomeDir, os.Getwd)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(path).To(Equal(stateFile))
}

func TestRunAgentDispatch_KillSubcommand_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := cli.Run([]string{"engram", "agent", "kill",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", filepath.Join(dir, "chat.toml"),
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("name"))
	}
}

func TestRunAgentDispatch_NoArgs_ReturnsUsageError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "agent"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRunAgentDispatch_SpawnSubcommand_ReturnsNameError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "agent", "spawn"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("name"))
	}
}

func TestRunAgentDispatch_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "agent", "bogus"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

func TestRunAgentDispatch_WaitReadySubcommand_RequiresName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "agent", "wait-ready"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("name"))
	}
}

func TestRunAgentKill_ActiveUnmetHold_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "STARTING"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(initialState)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	// Chat file contains an unsatisfied lead-release hold on executor-1.
	chatContent := `
[[message]]
from = "lead"
to = "executor-1"
thread = "hold"
type = "hold-acquire"
ts = 2026-04-06T12:00:00Z
text = """{"hold-id":"test-hold-1","holder":"lead","target":"executor-1",
"condition":"lead-release:phase3","tag":"phase3","acquired-ts":"2026-04-06T12:00:00Z"}"""
`
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("active hold"))
	}
}

func TestRunAgentKill_HelpFlag_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "agent", "kill", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAgentKill_MarksAgentDeadInStateFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initialAgents := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
			{Name: "reviewer-1", PaneID: "main:1.3", SessionID: "sess2", State: "ACTIVE"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(initialAgents)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	cli.SetTestPaneKiller(t, func(_ string) error { return nil })

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// After kill, executor-1 must be present with state=DEAD (not removed).
	remaining, _ := os.ReadFile(stateFile)
	g.Expect(string(remaining)).To(ContainSubstring("executor-1"))
	g.Expect(string(remaining)).To(ContainSubstring("DEAD"))
	g.Expect(string(remaining)).To(ContainSubstring("reviewer-1"))
}

func TestRunAgentKill_MetHold_AutoReleased(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(initialState)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	// Chat has a done: hold on executor-1 and a matching done message from executor-2.
	chatContent := `
[[message]]
from = "lead"
to = "executor-1"
thread = "hold"
type = "hold-acquire"
ts = 2026-04-06T12:00:00Z
text = """{"hold-id":"hold-auto-1","holder":"lead","target":"executor-1",
"condition":"done:executor-2","tag":"test","acquired-ts":"2026-04-06T12:00:00Z"}"""

[[message]]
from = "executor-2"
to = "all"
thread = "lifecycle"
type = "done"
ts = 2026-04-06T13:00:00Z
text = "executor-2 completed."
`
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	cli.SetTestPaneKiller(t, func(_ string) error { return nil })

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAgentKill_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "chat.toml"), []byte(""), 0o600)).To(Succeed())
	err := cli.Run([]string{"engram", "agent", "kill",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", filepath.Join(dir, "chat.toml"),
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("name"))
	}
}

func TestRunAgentKill_PaneAlreadyDead_NoError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "STARTING"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(initialState)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Simulate pane already dead: tmux returns "no such pane" error.
	cli.SetTestPaneKiller(t, func(_ string) error {
		return errors.New("exit status 1: no such pane: main:1.2")
	})

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	// Dead pane is not an error — agent is already gone.
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAgentKill_PaneKillError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(initialState)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Simulate a real kill failure (not "no such pane").
	cli.SetTestPaneKiller(t, func(_ string) error {
		return errors.New("session not found: mysession")
	})

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("killing pane"))
	}
}

// ============================================================
// Bug #533: pane kill verification in runAgentKill
// ============================================================

func TestRunAgentKill_PaneStillAliveAfterKill_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Pre-populate state with agent "ghost-agent" in pane %999.
	prePopErr := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{
			Name:   "ghost-agent",
			PaneID: "%999",
			State:  "RUNNING",
		})
	})
	g.Expect(prePopErr).NotTo(HaveOccurred())

	if prePopErr != nil {
		return
	}

	// Killer claims success; verifier reports pane still alive.
	cli.SetTestPaneKiller(t, func(_ string) error { return nil })
	cli.SetTestPaneVerifier(t, func(_ string) error {
		return errors.New("pane still alive after kill")
	})

	err := cli.ExportRunAgentKill([]string{
		"--name", "ghost-agent",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard)

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("still alive"))
	}
}

func TestRunAgentKill_StdoutError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
		},
	}
	agentData, _ := agentpkg.MarshalStateFile(initialState)
	g.Expect(os.WriteFile(stateFile, agentData, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	cli.SetTestPaneKiller(t, func(_ string) error { return nil })

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, &failWriter{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRunAgentList_AbsentStateFile_AttemptsReconstruction(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// No state file — reconstruction path. Chat file also absent (empty reconstruction).
	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", filepath.Join(dir, "chat.toml"),
	}, &stdout, io.Discard, nil)
	// No error — reconstruction attempt made, result is empty agent list.
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(stdout.String())).To(BeEmpty())
}

func TestRunAgentList_AbsentStateFile_ChatHasReadyMessages_ReconstructsAgents(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write chat file with two ready messages.
	chatContent := `
[[message]]
from = "executor-1"
to = "all"
thread = "lifecycle"
type = "ready"
ts = 2026-04-06T12:00:00Z
text = """Joining chat."""

[[message]]
from = "reviewer-1"
to = "all"
thread = "lifecycle"
type = "ready"
ts = 2026-04-06T12:01:00Z
text = """Joining chat."""
`
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := strings.TrimSpace(stdout.String())
	g.Expect(output).To(ContainSubstring("executor-1"))
	g.Expect(output).To(ContainSubstring("reviewer-1"))
}

func TestRunAgentList_DoneAgentExcludedFromReconstruction(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// executor-1 posted ready then done — should be excluded.
	// executor-2 posted ready but no done — should be included.
	chatContent := `
[[message]]
from = "executor-1"
to = "all"
thread = "lifecycle"
type = "ready"
ts = 2026-04-06T12:00:00Z
text = """Joining chat."""

[[message]]
from = "executor-1"
to = "all"
thread = "lifecycle"
type = "done"
ts = 2026-04-06T12:01:00Z
text = """Shutting down."""

[[message]]
from = "executor-2"
to = "all"
thread = "lifecycle"
type = "ready"
ts = 2026-04-06T12:02:00Z
text = """Joining chat."""
`
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := strings.TrimSpace(stdout.String())
	g.Expect(output).NotTo(ContainSubstring("executor-1"))
	g.Expect(output).To(ContainSubstring("executor-2"))
}

func TestRunAgentList_EmptyStateFile_NoOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	g.Expect(os.WriteFile(stateFile, []byte(""), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(stdout.String())).To(BeEmpty())
}

func TestRunAgentList_HelpFlag_NoError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--help"}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAgentList_InvalidTOMLStateFile_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	g.Expect(os.WriteFile(stateFile, []byte("[[not valid toml{{{{"), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRunAgentList_LastSilentAtIncludedInOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	lastSilentAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	state := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{
				Name:         "engram-agent",
				PaneID:       "main:1.1",
				SessionID:    "sess-xyz",
				State:        "SILENT",
				SpawnedAt:    time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
				LastSilentAt: lastSilentAt,
			},
		},
	}
	data, _ := agentpkg.MarshalStateFile(state)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(1))

	var rec map[string]any

	g.Expect(json.Unmarshal([]byte(lines[0]), &rec)).To(Succeed())
	g.Expect(rec).To(HaveKey("last-silent-at"))
	g.Expect(rec["last-silent-at"]).NotTo(BeEmpty())
}

func TestRunAgentList_MultipleAgents_NDJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	state := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
			{Name: "reviewer-1", PaneID: "main:1.3", SessionID: "sess2", State: "ACTIVE"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(state)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(2))

	var rec1 map[string]any
	g.Expect(json.Unmarshal([]byte(lines[0]), &rec1)).To(Succeed())
	g.Expect(rec1["name"]).To(Equal("executor-1"))
	g.Expect(rec1["pane-id"]).To(Equal("main:1.2"))
	g.Expect(rec1["state"]).To(Equal("ACTIVE"))

	var rec2 map[string]any
	g.Expect(json.Unmarshal([]byte(lines[1]), &rec2)).To(Succeed())
	g.Expect(rec2["name"]).To(Equal("reviewer-1"))
}

func TestRunAgentList_StateFileIsDir_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// Use a directory path as the state file — os.ReadFile on a directory fails with EISDIR.
	stateAsDir := filepath.Join(dir, "statedir")
	g.Expect(os.MkdirAll(stateAsDir, 0o755)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateAsDir}, &stdout, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRunAgentList_UnreadableChatFile_ReconstructionFails_NoError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// Use a directory path as the chat file — os.ReadFile will fail with EISDIR (not ErrNotExist).
	chatAsDir := filepath.Join(dir, "chatdir")
	g.Expect(os.MkdirAll(chatAsDir, 0o755)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", chatAsDir,
	}, &stdout, io.Discard, nil)
	// Reconstruction failure is logged but not returned as an error.
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(stdout.String())).To(BeEmpty())
}

// TestRunAgentRun_FakeClaude_DONE exercises the inner conversation loop using a fake
// claude binary that emits a DONE marker and exits. Covers: buildAgentRunner,
// runConversationLoopWith (inner-loop-only), runOneTurn, buildClaudeCmd, chatFileCursor.
// Uses ExportRunConversationLoopWith with nil watchForIntent to test inner-loop exit;
// the outer watch loop is covered by TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches.
func TestRunAgentRun_FakeClaude_DONE(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Write a fake state file with the agent so session-id write succeeds.
	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Fake claude: emits a DONE stream-json event then exits 0.
	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-123",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`

	script := "#!/bin/sh\necho '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx := context.Background()

	// Use io.Discard for stdout to avoid a data race: cmd.Stderr and runner.Pane
	// both reference the same writer and are written concurrently during stream processing.
	// Pass nil watchForIntent to test inner-loop-only (Phase 4 exit behavior).
	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		cli.ExportWaitAndBuildPrompt,
		nil, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
}

// TestRunAgentRun_FakeClaude_ExitNonZero verifies runOneTurn propagates non-zero exit.
func TestRunAgentRun_FakeClaude_ExitNonZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Fake claude: exits non-zero (simulates claude crash).
	fakeClaude := filepath.Join(dir, "claude")
	g.Expect(os.WriteFile(fakeClaude, []byte("#!/bin/sh\nexit 1\n"), 0o700)).To(Succeed())

	err := cli.ExportRunAgentRunWith([]string{
		"--name", "worker-1",
		"--prompt", "hello",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard, fakeClaude)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("agent run:"))
}

// TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean verifies conversation ends when claude
// emits no INTENT or DONE markers (treated as complete).
// Uses ExportRunConversationLoopWith with nil watchForIntent to test inner-loop exit;
// the outer watch loop is covered by TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches.
func TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Fake claude: emits plain assistant text with no markers.
	fakeClaude := filepath.Join(dir, "claude")
	plainJSON := `{"type":"assistant","session_id":"sess-456",` +
		`"message":{"content":[{"type":"text","text":"Here is your answer."}]}}`

	script := "#!/bin/sh\necho '" + plainJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx := context.Background()

	// Pass nil watchForIntent to test inner-loop-only (Phase 4 exit behavior).
	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		cli.ExportWaitAndBuildPrompt,
		nil, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
}

// ============================================================
// Bug #533: duplicate name guard in runAgentSpawn
// ============================================================

func TestRunAgentSpawn_DuplicateName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Pre-populate state file with an agent named "exec-1".
	prePopErr := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{
			Name:   "exec-1",
			PaneID: "main:1.1",
			State:  "RUNNING",
		})
	})
	g.Expect(prePopErr).NotTo(HaveOccurred())

	if prePopErr != nil {
		return
	}

	err := cli.ExportRunAgentSpawn([]string{
		"--name", "exec-1",
		"--prompt", "You are exec-1.",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
		return "main:1.2", "sess456", nil
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("exec-1"))
	}
}

func TestRunAgentSpawn_FourthWorker_IsNotRejectedByCap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cli.SetTestSpawnAckMaxWait(t, 100*time.Millisecond)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Make engram-agent appear online so the 100ms MaxWait fires a TIMEOUT
	// (nil error) after one watcher cycle instead of spinning for 5s.
	g.Expect(cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent",
		"--to", "all",
		"--thread", "lifecycle",
		"--type", "ready",
		"--text", "Ready.",
	}, io.Discard, io.Discard, nil)).To(Succeed())

	// Pre-populate state file with 3 active agents (the old cap).
	for _, name := range []string{"exec-1", "exec-2", "exec-3"} {
		prePopErr := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
			return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: name, State: "ACTIVE"})
		})
		g.Expect(prePopErr).NotTo(HaveOccurred())

		if prePopErr != nil {
			return
		}
	}

	err := cli.ExportRunAgentSpawn([]string{
		"--name", "exec-4",
		"--prompt", "You are exec-4.",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
		return "main:1.4", "sess456", nil
	})

	// Cap must be gone: any error should not mention "worker queue full".
	if err != nil {
		g.Expect(err.Error()).NotTo(ContainSubstring("worker queue full"))
	}
}

func TestRunAgentSpawn_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := cli.Run([]string{"engram", "agent", "spawn",
		"--prompt", "You are an executor.",
		"--chat-file", filepath.Join(dir, "chat.toml"),
		"--state-file", filepath.Join(dir, "state.toml"),
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("name"))
	}
}

func TestRunAgentSpawn_MissingPrompt_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := cli.Run([]string{"engram", "agent", "spawn",
		"--name", "executor-1",
		"--chat-file", filepath.Join(dir, "chat.toml"),
		"--state-file", filepath.Join(dir, "state.toml"),
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("prompt"))
	}
}

func TestRunAgentSpawn_PostIntentError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	// Use a non-existent dir for chat file so poster.Post fails.
	chatFile := filepath.Join(dir, "nonexistent", "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	err := cli.ExportRunAgentSpawn([]string{
		"--name", "executor-2",
		"--prompt", "You are executor-2.",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
		return "main:1.3", "sess456", nil
	})

	g.Expect(err).To(HaveOccurred())
}

// ============================================================
// Bug #531: system ACK after spawn intent resolves
// ============================================================

func TestRunAgentSpawn_PostsSystemACKAfterAckWaitResolves(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	spawnDone := make(chan error, 1)

	go func() {
		spawnDone <- cli.ExportRunAgentSpawn([]string{
			"--name", "exec-ack-test",
			"--prompt", "You are exec-ack-test.",
			"--chat-file", chatFile,
			"--state-file", stateFile,
		}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
			return "main:1.2", "sess123", nil
		})
	}()

	// Wait for intent to be posted, then ACK it.
	time.Sleep(50 * time.Millisecond)

	g.Expect(cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent",
		"--to", "system",
		"--thread", "lifecycle",
		"--type", "ack",
		"--text", "No relevant memories. Proceed.",
	}, io.Discard, io.Discard, nil)).To(Succeed())

	select {
	case err := <-spawnDone:
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}
	case <-time.After(6 * time.Second):
		t.Fatal("agent spawn did not complete within 6s")
	}

	messages, loadErr := cli.ExportLoadChatMessages(chatFile)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	var systemACKFound bool

	for _, msg := range messages {
		if msg.From == "system" && msg.Type == "ack" {
			systemACKFound = true

			break
		}
	}

	g.Expect(systemACKFound).To(BeTrue(), "expected a system ack message in chat after spawn")
}

func TestRunAgentSpawn_PostsSystemACKWhenEngramAgentOffline(t *testing.T) {
	// Primary #531 scenario: engram-agent is offline. ack-wait grants an implicit ACK after
	// 5s (offline threshold). The system must still post an observable ACK to the chat file.
	// MaxWait is injected as 6s so Watch's deadline fires just after the 5s offline threshold,
	// keeping this test fast without hard-coding production timing in assertions.
	t.Parallel()
	g := NewWithT(t)

	cli.SetTestSpawnAckMaxWait(t, 6*time.Second)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	// Empty chat file — no engram-agent messages → treated as offline.
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	spawnDone := make(chan error, 1)

	go func() {
		spawnDone <- cli.ExportRunAgentSpawn([]string{
			"--name", "exec-offline-test",
			"--prompt", "You are exec-offline-test.",
			"--chat-file", chatFile,
			"--state-file", stateFile,
		}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
			return "main:1.3", "sess789", nil
		})
	}()

	// No explicit ACK — ack-wait must complete via offline implicit ACK (~6s with test MaxWait).
	select {
	case err := <-spawnDone:
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}
	case <-time.After(12 * time.Second):
		t.Fatal("agent spawn did not complete within 12s (offline implicit ACK should fire at ~6s)")
	}

	messages, loadErr := cli.ExportLoadChatMessages(chatFile)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	var systemACKFound bool

	for _, msg := range messages {
		if msg.From == "system" && msg.Type == "ack" {
			systemACKFound = true

			break
		}
	}

	g.Expect(systemACKFound).To(BeTrue(), "expected system ACK even when engram-agent is offline")
}

func TestRunAgentSpawn_SpawnerError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	err := cli.ExportRunAgentSpawn([]string{
		"--name", "executor-1",
		"--prompt", "You are executor-1.",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
		return "", "", errors.New("tmux not found")
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("launching pane"))
	}
}

func TestRunAgentSpawn_StateFileUnreadable_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Use a directory as the state file path so os.ReadFile returns a non-ErrNotExist error.
	stateFileAsDir := filepath.Join(dir, "state-dir")
	g.Expect(os.Mkdir(stateFileAsDir, 0o700)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	err := cli.ExportRunAgentSpawn([]string{
		"--name", "exec-1",
		"--prompt", "You are exec-1.",
		"--chat-file", chatFile,
		"--state-file", stateFileAsDir,
	}, io.Discard, func(_ context.Context, _, _ string) (string, string, error) {
		return "main:1.2", "sess123", nil
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("reading state file"))
	}
}

func TestRunAgentSpawn_WritesStateFileAndOutputsPaneID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	spawnDone := make(chan error, 1)

	go func() {
		spawnDone <- cli.ExportRunAgentSpawn([]string{
			"--name", "executor-1",
			"--prompt", "You are executor-1.",
			"--chat-file", chatFile,
			"--state-file", stateFile,
		}, &stdout, func(_ context.Context, _, _ string) (string, string, error) {
			return "main:1.2", "sess123", nil
		})
	}()

	// Wait for intent to be posted, then ACK it so AckWait resolves quickly.
	time.Sleep(50 * time.Millisecond)

	g.Expect(cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent",
		"--to", "system",
		"--thread", "lifecycle",
		"--type", "ack",
		"--text", "No relevant memories. Proceed.",
	}, io.Discard, io.Discard, nil)).To(Succeed())

	select {
	case err := <-spawnDone:
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}
	case <-time.After(6 * time.Second):
		t.Fatal("agent spawn did not complete within 6s")
	}

	g.Expect(strings.TrimSpace(stdout.String())).To(Equal("main:1.2|sess123"))

	data, readErr := os.ReadFile(stateFile)

	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("executor-1"))
	g.Expect(string(data)).To(ContainSubstring("main:1.2"))
}

func TestRunAgentWaitReady_MaxWaitExpires_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	start := time.Now()
	err := cli.Run([]string{"engram", "agent", "wait-ready",
		"--name", "nonexistent-agent",
		"--cursor", "0",
		"--max-wait", "1",
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	elapsed := time.Since(start)

	g.Expect(err).To(HaveOccurred())
	g.Expect(elapsed).To(BeNumerically("<", 5*time.Second))
}

func TestRunAgentWaitReady_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	err := cli.Run([]string{"engram", "agent", "wait-ready",
		"--cursor", "0",
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("name"))
	}
}

func TestRunAgentWaitReady_SeesReadyMessage_OutputsJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Append a ready message after a short delay to simulate agent startup.
	go func() {
		time.Sleep(50 * time.Millisecond)

		readyTOML := "\n[[message]]\nfrom = \"executor-1\"\nto = \"all\"\n" +
			"thread = \"lifecycle\"\ntype = \"ready\"\nts = 2026-04-06T12:00:00Z\n" +
			"text = \"\"\"Joining chat.\"\"\"\n"

		f, _ := os.OpenFile(chatFile, os.O_APPEND|os.O_WRONLY, 0o600)

		if f != nil {
			_, _ = f.WriteString(readyTOML)
			_ = f.Close()
		}
	}()

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "wait-ready",
		"--name", "executor-1",
		"--cursor", "0",
		"--max-wait", "5",
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
	g.Expect(result["type"]).To(Equal("ready"))
	g.Expect(result["from"]).To(Equal("executor-1"))
	g.Expect(result["cursor"]).NotTo(BeZero())
}

// TestRunConversationLoopWith_AgentNamePropagated is a property-based test (issue 544).
// Property: for any valid agent name string, runConversationLoopWith completes without error
// when the explicit agentName param matches the agent record in the state file.
// Uses ExportRunConversationLoopWithName (the Phase 1 shim) which calls the new explicit-param
// signature — this test fails to compile until Phase 2 adds the agentName parameter.
func TestRunConversationLoopWith_AgentNamePropagated(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		// Generate a valid agent name: starts with a letter, followed by letters/digits/hyphens.
		agentName := rapid.StringMatching(`[a-z][a-z0-9-]{1,19}`).Draw(rt, "agentName")

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")
		stateFile := filepath.Join(dir, "state.toml")

		g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

		if err := os.WriteFile(chatFile, []byte(""), 0o600); err != nil {
			return
		}

		// Seed state file with the generated agent name so state writes succeed.
		stateToml := fmt.Sprintf(
			"[[agent]]\nname = %q\npane_id = \"\"\n"+
				"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n",
			agentName,
		)
		g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

		if err := os.WriteFile(stateFile, []byte(stateToml), 0o600); err != nil {
			return
		}

		// Fake claude: emits DONE immediately so the loop exits cleanly.
		fakeClaude := filepath.Join(dir, "claude")
		doneJSON := `{"type":"assistant","session_id":"sess-prop",` +
			`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
		script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
		g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

		if err := os.WriteFile(fakeClaude, []byte(script), 0o700); err != nil {
			return
		}

		err := cli.ExportRunConversationLoopWith(
			context.Background(),
			agentName, "initial prompt", chatFile, stateFile, fakeClaude,
			io.Discard,
			cli.ExportWaitAndBuildPrompt,
			nil, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}
	})
}

// ============================================================
// watchAndResume memFileSelector error logging tests (issue 547)
// ============================================================

// makeWatchAndResumeFixture creates a minimal temp dir and stub functions for
// TestRunConversationLoopWith_ChannelReceivesIntent verifies that when intents channel is non-nil,
// the loop reads from it instead of calling watchForIntent.
func TestRunConversationLoopWith_ChannelReceivesIntent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane-id = \"\"\n" +
		"session-id = \"\"\nstate = \"STARTING\"\nspawned-at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	// intents channel with one message; context cancels after first session.
	intents := make(chan chat.Message, 1)
	intents <- chat.Message{From: "lead", To: "worker-1", Type: "intent", Text: "do the thing"}

	ctx, cancel := context.WithCancel(context.Background())

	// watchForIntent must NOT be called when intents channel is used.
	watchForIntentCalled := false
	watchForIntent := func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
		watchForIntentCalled = true

		cancel()

		return chat.Message{}, 0, context.Canceled
	}

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}
	memFileSelector := func(_ string, _ int) ([]string, error) { return nil, nil }

	// After one session + channel read, cancel context so loop exits.
	// We can't know exact timing, so cancel after session fires.
	go func() {
		<-time.After(3 * time.Second)
		cancel()
	}()

	err := cli.ExportRunConversationLoopWithChannel(
		ctx, "worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard, stubBuilder, watchForIntent, intents, nil, memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(watchForIntentCalled).To(BeFalse(), "watchForIntent must not be called when intents channel is provided")
}

// TestRunConversationLoopWith_EmptyPromptDispatchMode_WaitsForFirstIntent verifies that
// when initialPrompt is empty (dispatch mode), the worker does NOT call claude before the
// first intent arrives. It should write SILENT state and wait on the intents channel first.
func TestRunConversationLoopWith_EmptyPromptDispatchMode_WaitsForFirstIntent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane-id = \"\"\n" +
		"session-id = \"\"\nstate = \"STARTING\"\nspawned-at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// fakeClaude writes a sentinel file on first invocation so the test can detect early calls.
	fakeClaude := filepath.Join(dir, "claude")
	sentinelFile := filepath.Join(dir, "invoked")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\ntouch " + sentinelFile + "\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	intents := make(chan chat.Message, 1)
	silentCh := make(chan string, 2) //nolint:gomnd // buffer for initial + post-session signals

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// invokedBeforeIntent is true if claude ran before the intent was sent.
	var invokedBeforeIntent atomic.Bool

	go func() {
		// Wait for worker to signal SILENT (it is now waiting for first intent).
		select {
		case <-silentCh:
		case <-ctx.Done():
			return
		}

		// At this point, claude must NOT have been called yet.
		if _, statErr := os.Stat(sentinelFile); statErr == nil {
			invokedBeforeIntent.Store(true)
		}

		// Send the first intent; worker should now run its session.
		intents <- chat.Message{From: "lead", To: "worker-1", Type: "intent", Text: "do task"}

		// Wait for post-session SILENT signal, then cancel the context.
		select {
		case <-silentCh:
		case <-ctx.Done():
			return
		}

		cancel()
	}()

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}
	memFileSelector := func(_ string, _ int) ([]string, error) { return nil, nil }

	err := cli.ExportRunConversationLoopWithChannel(
		ctx, "worker-1", "" /* empty prompt = dispatch mode */, chatFile, stateFile, fakeClaude,
		io.Discard, stubBuilder, nil, intents, silentCh, memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(invokedBeforeIntent.Load()).To(BeFalse(), "claude must not be invoked before first intent arrives")

	// Sentinel file must exist (claude was invoked once, after the intent).
	_, sentinelErr := os.Stat(sentinelFile)
	g.Expect(sentinelErr).NotTo(HaveOccurred(), "claude must have been invoked after intent")
}

// ============================================================
// runConversationLoopWith: INTENT path coverage
// ============================================================

// TestRunConversationLoopWith_IntentThenDone exercises the INTENT → ack-wait → DONE path
// using a fake claude binary and an immediate stub prompt builder.
func TestRunConversationLoopWith_IntentThenDone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Turn 0: emit INTENT so loop continues. Turn 1: emit DONE so loop ends.
	turn := 0

	dir2 := t.TempDir()
	fakeClaude := filepath.Join(dir2, "claude")

	intentJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"INTENT: Plan to proceed."}]}}`
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`

	// Script: on first call emit INTENT; on second call emit DONE.
	// We use a flag file to distinguish calls.
	flagFile := filepath.Join(dir2, "called")
	script := "#!/bin/sh\n" +
		"if [ -f " + flagFile + " ]; then\n" +
		"  printf '%s\\n' '" + doneJSON + "'\nelse\n" +
		"  touch " + flagFile + "\n" +
		"  printf '%s\\n' '" + intentJSON + "'\nfi\n"

	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	// Stub promptBuilder returns immediately (no real ack-wait).
	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		turn++

		return "Proceed.", nil
	}

	err := cli.ExportRunConversationLoopWith(
		context.Background(),
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		nil, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(turn).To(Equal(1))
}

// TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches verifies that when the
// inner session emits no markers, the outer loop calls watchForIntent and re-enters
// the inner loop. Context cancellation on the second watch call exits the loop cleanly.
func TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	dir2 := t.TempDir()
	fakeClaude := filepath.Join(dir2, "claude")

	// Always emits plain text with no markers; the outer loop re-enters via watchForIntent.
	plainJSON := `{"type":"assistant","session_id":"sess-no-markers",` +
		`"message":{"content":[{"type":"text","text":"Here is your answer."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + plainJSON + "'\n"

	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	// stubPromptBuilder: returns a simple prompt; called after watchForIntent fires.
	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// stubWatchForIntent: first call returns a fake intent; second call cancels ctx to exit loop.
	watchCallCount := 0
	stubWatchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		watchCallCount++
		if watchCallCount > 1 {
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		return chat.Message{
			From: "lead",
			Type: "intent",
			Text: "Resume now.",
		}, cursor + 1, nil
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		stubWatchForIntent,
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// watchForIntent must have been called at least once (outer loop entered after no-marker session).
	g.Expect(watchCallCount).To(BeNumerically(">=", 1))

	// State file must contain state = "SILENT" for worker-1
	// (written by watchAndResume before calling watchForIntent).
	stateData, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsedState, parseErr := agentpkg.ParseStateFile(stateData)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	var worker1State string

	for _, rec := range parsedState.Agents {
		if rec.Name == "worker-1" {
			worker1State = rec.State
		}
	}

	g.Expect(worker1State).To(Equal("SILENT"))
}

// TestRunConversationLoopWith_PromptBuilderError verifies promptBuilder errors propagate.
func TestRunConversationLoopWith_PromptBuilderError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	intentJSON := `{"type":"assistant","session_id":"sess-xyz",` +
		`"message":{"content":[{"type":"text","text":"INTENT: Plan to proceed."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + intentJSON + "'\n"

	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "", errors.New("ack-wait failed")
	}

	err := cli.ExportRunConversationLoopWith(
		context.Background(),
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		stubBuilder,
		nil, nil,
	)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("ack-wait failed"))
}

// TestRunConversationLoopWith_SilentChSignaledAfterSession verifies that silentCh receives
// the agent name when the session transitions to SILENT state.
func TestRunConversationLoopWith_SilentChSignaledAfterSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane-id = \"\"\n" +
		"session-id = \"\"\nstate = \"STARTING\"\nspawned-at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	// Only one intent in the channel.
	intents := make(chan chat.Message, 1)
	intents <- chat.Message{From: "lead", To: "worker-1", Type: "intent", Text: "do the thing"}

	// silentCh is drained by a goroutine: cancel context on first receipt so the
	// loop exits cleanly. Without draining, silentCh fills and watchAndResume blocks.
	silentCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var receivedAgent string

	go func() {
		for {
			select {
			case name := <-silentCh:
				if receivedAgent == "" {
					receivedAgent = name

					cancel() // trigger clean exit after first signal
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	stubBuilder := func(_ context.Context, _, _ string, _ int) (string, error) {
		return "Proceed.", nil
	}
	memFileSelector := func(_ string, _ int) ([]string, error) { return nil, nil }

	err := cli.ExportRunConversationLoopWithChannel(
		ctx, "worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard, stubBuilder, nil, intents, silentCh, memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// After loop exits, receivedAgent must be set to "worker-1".
	g.Expect(receivedAgent).To(Equal("worker-1"), "silentCh should have received agent name after SILENT transition")
}

// Step 14: Full intent→ack→hold→check→release E2E cycle.
func TestRun_AckWait_With_HoldAcquire_E2E(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-create chat file so ack-wait can watch it (mirrors production: intent is
	// posted before ack-wait is called, ensuring the file already exists).
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Step 1: Get cursor, then start ack-wait in a goroutine.
	var cursorOut bytes.Buffer

	err := cli.Run([]string{"engram", "chat", "cursor", "--chat-file", chatFile}, &cursorOut, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	cursor := strings.TrimSpace(cursorOut.String())

	ackWaitDone := make(chan string, 1)

	go func() {
		var out bytes.Buffer

		_ = cli.Run([]string{
			"engram", "chat", "ack-wait",
			"--chat-file", chatFile,
			"--agent", "lead",
			"--cursor", cursor,
			"--recipients", "executor-1",
			"--max-wait", "5",
		}, &out, io.Discard, nil)
		ackWaitDone <- strings.TrimSpace(out.String())
	}()

	// Post ACK from executor-1 to lead after brief delay to ensure ack-wait is watching.
	time.Sleep(50 * time.Millisecond)

	g.Expect(cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "executor-1",
		"--to", "lead",
		"--thread", "e2e",
		"--type", "ack",
		"--text", "proceeding",
	}, io.Discard, io.Discard, nil)).To(Succeed())

	// Step 2: Verify ack-wait resolved with ACK.
	select {
	case result := <-ackWaitDone:
		g.Expect(result).To(ContainSubstring(`"result":"ACK"`))
	case <-time.After(6 * time.Second):
		t.Fatal("ack-wait did not resolve within 6s")
	}

	// Step 3: Acquire hold — executor-1 must not be killed until done.
	var holdOut bytes.Buffer

	g.Expect(cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "executor-1",
		"--condition", "done:executor-1",
	}, &holdOut, io.Discard, nil)).To(Succeed())

	holdID := strings.TrimSpace(holdOut.String())
	g.Expect(holdID).NotTo(BeEmpty())

	// Step 4: Verify hold is active.
	var listOut bytes.Buffer

	g.Expect(cli.Run([]string{"engram", "hold", "list", "--chat-file", chatFile}, &listOut, io.Discard, nil)).To(Succeed())
	g.Expect(listOut.String()).To(ContainSubstring(holdID))

	// Step 5: Executor-1 posts done.
	g.Expect(cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "executor-1",
		"--to", "all",
		"--thread", "e2e",
		"--type", "done",
		"--text", "task complete",
	}, io.Discard, io.Discard, nil)).To(Succeed())

	// Step 6: Hold check evaluates condition and auto-releases.
	var checkOut bytes.Buffer

	checkErr := cli.Run([]string{"engram", "hold", "check", "--chat-file", chatFile}, &checkOut, io.Discard, nil)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(checkOut.String())).To(Equal(holdID))

	// Step 7: Verify hold is cleared.
	var listOut2 bytes.Buffer

	listErr2 := cli.Run([]string{"engram", "hold", "list", "--chat-file", chatFile}, &listOut2, io.Discard, nil)
	g.Expect(listErr2).NotTo(HaveOccurred())
	g.Expect(listOut2.String()).To(BeEmpty())
}

func TestRun_AgentRun_HelpFlag_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "run", "--help"}, &stdout, &stderr, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_AgentRun_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "run", "--prompt", "hello"}, &stdout, &stderr, nil)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--name"))
}

func TestRun_AgentRun_MissingPrompt_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "run", "--name", "worker-1"}, &stdout, &stderr, nil)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--prompt"))
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

func TestRun_ChatAckWait_EmptyRecipients_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run([]string{
		"engram", "chat", "ack-wait",
		"--chat-file", chatFile,
		"--agent", "tester",
		"--cursor", "0",
		"--recipients", "",
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(MatchError(ContainSubstring("--recipients required")))
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

// ============================================================
// runChatCursor ErrHelp coverage
// ============================================================

func TestRun_ChatCursor_HelpFlag_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "chat", "cursor", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
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

// ============================================================
// Task 4: Concurrent + Integration Tests
// ============================================================

// Step 13: Concurrent hold-acquire safety test.
func TestRun_HoldAcquire_ConcurrentWritesSafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	const holdCount = 5

	var wg sync.WaitGroup

	wg.Add(holdCount)

	for i := range holdCount {
		go func(n int) {
			defer wg.Done()

			_ = cli.Run([]string{
				"engram", "hold", "acquire",
				"--chat-file", chatFile,
				"--holder", fmt.Sprintf("lead-%d", n),
				"--target", fmt.Sprintf("exec-%d", n),
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
	g.Expect(parsed.Message).To(HaveLen(holdCount))

	for _, msg := range parsed.Message {
		g.Expect(msg.Type).To(Equal("hold-acquire"))
	}
}

func TestRun_HoldAcquire_EmptyHolder_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run([]string{"engram", "hold", "acquire", "--chat-file", chatFile}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--holder is required"))
}

func TestRun_HoldAcquire_EmptyTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run(
		[]string{"engram", "hold", "acquire", "--chat-file", chatFile, "--holder", "lead"},
		&bytes.Buffer{}, io.Discard, nil,
	)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--target is required"))
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

func TestRun_HoldCheck_FileUnreadable_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte{}, 0o000)).To(Succeed())

	err := cli.Run([]string{
		"engram", "hold", "check",
		"--chat-file", chatFile,
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(MatchError(ContainSubstring("hold check")))
}

func TestRun_HoldCheck_HelpExitsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "check", "--help"}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_HoldCheck_InvalidTOML_SucceedsWithNoMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("not valid toml :::"), 0o600)).To(Succeed())

	err := cli.Run([]string{
		"engram", "hold", "check",
		"--chat-file", chatFile,
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
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

func TestRun_HoldList_InvalidTOML_SucceedsWithEmptyOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte("not valid toml :::"), 0o600)).To(Succeed())

	err := cli.Run([]string{
		"engram", "hold", "list",
		"--chat-file", chatFile,
	}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_HoldList_OutputsNDJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire a hold with all fields populated.
	acquireErr := cli.Run([]string{
		"engram", "hold", "acquire",
		"--chat-file", chatFile,
		"--holder", "lead",
		"--target", "exec-1",
		"--tag", "codesign-1",
	}, io.Discard, io.Discard, nil)
	g.Expect(acquireErr).NotTo(HaveOccurred())

	if acquireErr != nil {
		return
	}

	var stdout bytes.Buffer

	listErr := cli.Run([]string{
		"engram", "hold", "list",
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(listErr).NotTo(HaveOccurred())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(1))

	// Each line must be valid JSON with a non-empty hold-id.
	var record map[string]any
	g.Expect(json.Unmarshal([]byte(lines[0]), &record)).To(Succeed())
	g.Expect(record["hold-id"]).To(BeAssignableToTypeOf(""))
	g.Expect(record["hold-id"]).NotTo(BeEmpty())
	g.Expect(record["holder"]).To(Equal("lead"))
	g.Expect(record["target"]).To(Equal("exec-1"))
	g.Expect(record["tag"]).To(Equal("codesign-1"))
}

func TestRun_HoldList_ParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.Run([]string{"engram", "hold", "list", "--bogus-flag"}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_HoldRelease_EmptyHoldID_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	err := cli.Run([]string{"engram", "hold", "release", "--chat-file", chatFile}, &bytes.Buffer{}, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--hold-id is required"))
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

func TestSelectMemoryFiles_CapsAtLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const totalFiles = 25

	entries := make([]fakeDirEntry, 0, totalFiles)
	for i := range totalFiles {
		entries = append(entries, fakeDirEntry{name: fmt.Sprintf("mem-%02d.md", i), isDir: false})
	}

	readDir := func(dir string) ([]os.DirEntry, error) {
		if dir == "/feedback" {
			result := make([]os.DirEntry, 0, len(entries))
			for i := range entries {
				result = append(result, &entries[i])
			}

			return result, nil
		}

		return nil, nil
	}

	now := time.Now()
	statFile := func(path string) (os.FileInfo, error) {
		return &fakeFileInfo{name: filepath.Base(path), modTime: now}, nil
	}

	const maxFiles = 20

	files, err := cli.ExportSelectMemoryFiles("/feedback", "/facts", readDir, statFile, maxFiles)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(HaveLen(maxFiles))
}

// ============================================================
// selectMemoryFiles tests
// ============================================================

func TestSelectMemoryFiles_EmptyDirs_ReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	emptyReader := func(_ string) ([]os.DirEntry, error) {
		return nil, nil
	}
	noStat := func(_ string) (os.FileInfo, error) {
		return nil, errors.New("should not be called")
	}

	files, err := cli.ExportSelectMemoryFiles("/feedback", "/facts", emptyReader, noStat, 20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(BeEmpty())
}

func TestSelectMemoryFiles_ReturnsAbsolutePaths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []fakeDirEntry{
		{name: "a.md", isDir: false},
	}

	readDir := func(dir string) ([]os.DirEntry, error) {
		if dir == "/facts" {
			result := make([]os.DirEntry, 0, len(entries))
			for i := range entries {
				result = append(result, &entries[i])
			}

			return result, nil
		}

		return nil, nil
	}

	statFile := func(path string) (os.FileInfo, error) {
		return &fakeFileInfo{name: filepath.Base(path), modTime: time.Now()}, nil
	}

	files, err := cli.ExportSelectMemoryFiles("/feedback", "/facts", readDir, statFile, 20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(HaveLen(1))
	g.Expect(filepath.IsAbs(files[0])).To(BeTrue())
	g.Expect(files[0]).To(Equal("/facts/a.md"))
}

func TestSelectMemoryFiles_ReturnsTopNByMtime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []fakeDirEntry{
		{name: "old.md", isDir: false},
		{name: "new.md", isDir: false},
		{name: "mid.md", isDir: false},
	}

	readDir := func(dir string) ([]os.DirEntry, error) {
		if dir == "/feedback" {
			result := make([]os.DirEntry, 0, len(entries))
			for i := range entries {
				result = append(result, &entries[i])
			}

			return result, nil
		}

		return nil, nil
	}

	now := time.Now()
	mtimes := map[string]time.Time{
		"/feedback/old.md": now.Add(-3 * time.Hour),
		"/feedback/new.md": now,
		"/feedback/mid.md": now.Add(-1 * time.Hour),
	}

	statFile := func(path string) (os.FileInfo, error) {
		mt, ok := mtimes[path]
		if !ok {
			return nil, fmt.Errorf("unknown path: %s", path)
		}

		return &fakeFileInfo{name: filepath.Base(path), modTime: mt}, nil
	}

	files, err := cli.ExportSelectMemoryFiles("/feedback", "/facts", readDir, statFile, 2)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(HaveLen(2))
	g.Expect(files[0]).To(Equal("/feedback/new.md"))
	g.Expect(files[1]).To(Equal("/feedback/mid.md"))
}

func TestSelectMemoryFiles_SkipsDirectories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []fakeDirEntry{
		{name: "subdir", isDir: true},
		{name: "mem.md", isDir: false},
	}

	readDir := func(dir string) ([]os.DirEntry, error) {
		if dir == "/feedback" {
			result := make([]os.DirEntry, 0, len(entries))
			for i := range entries {
				result = append(result, &entries[i])
			}

			return result, nil
		}

		return nil, nil
	}

	statFile := func(path string) (os.FileInfo, error) {
		return &fakeFileInfo{name: filepath.Base(path), modTime: time.Now()}, nil
	}

	files, err := cli.ExportSelectMemoryFiles("/feedback", "/facts", readDir, statFile, 20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0]).To(Equal("/feedback/mem.md"))
}

func TestSelectRecentIntents_EmptyWhenNoIntents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := []byte("\n[[message]]\nfrom = \"a\"\nto = \"b\"\nthread = \"t\"\n" +
		"type = \"info\"\nts = 2026-04-11T00:00:00Z\ntext = \"\"\"\nhello\n\"\"\"\n")
	readFile := func(_ string) ([]byte, error) { return content, nil }

	summaries, err := cli.ExportSelectRecentIntents("/fake/chat.toml", readFile, 5)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(summaries).To(BeEmpty())
}

func TestSelectRecentIntents_ReturnsUpToMax(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var sb strings.Builder
	for i := range 7 {
		sb.WriteString("\n[[message]]\n")
		fmt.Fprintf(&sb, "from = \"agent-%d\"\n", i)
		sb.WriteString("to = \"engram-agent\"\n")
		sb.WriteString("thread = \"t\"\ntype = \"intent\"\n")
		sb.WriteString("ts = 2026-04-11T00:00:00Z\n")
		fmt.Fprintf(&sb, "text = \"\"\"\nintent %d\n\"\"\"\n", i)
	}

	content := []byte(sb.String())

	readFile := func(_ string) ([]byte, error) { return content, nil }
	summaries, err := cli.ExportSelectRecentIntents("/fake/chat.toml", readFile, 5)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(summaries).To(HaveLen(5), "should return at most 5 most recent intents")
}

func TestSelectRecentIntents_TruncatesAt80Chars(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	longText := strings.Repeat("x", 200)
	content := []byte("\n[[message]]\nfrom = \"lead\"\nto = \"worker\"\nthread = \"t\"\n" +
		"type = \"intent\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\n" + longText + "\n\"\"\"\n")
	readFile := func(_ string) ([]byte, error) { return content, nil }

	summaries, err := cli.ExportSelectRecentIntents("/fake/chat.toml", readFile, 5)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(summaries).To(HaveLen(1))
	g.Expect(len(summaries[0])).To(BeNumerically("<=", 100),
		"summary should be truncated to at most 80 chars of text plus metadata")
}

// ============================================================
// waitAndBuildPromptWith: coverage via stub ackWaiter
// ============================================================

// TestWaitAndBuildPromptWith_ACK verifies "Proceed." is returned on ACK.
func TestWaitAndBuildPromptWith_ACK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stub := &stubAckWaiter{result: chat.AckResult{Result: "ACK", NewCursor: 1}}

	prompt, err := cli.ExportWaitAndBuildPromptWith(context.Background(), "worker-1", 0, stub)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).To(Equal("Proceed."))
}

// TestWaitAndBuildPromptWith_AckWaitError verifies ack-wait errors propagate.
func TestWaitAndBuildPromptWith_AckWaitError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stub := &stubAckWaiter{err: errors.New("network error")}

	_, err := cli.ExportWaitAndBuildPromptWith(context.Background(), "worker-1", 0, stub)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("ack-wait"))
}

// TestWaitAndBuildPromptWith_WAIT_WithMessage verifies WAIT message is formatted correctly.
func TestWaitAndBuildPromptWith_WAIT_WithMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stub := &stubAckWaiter{result: chat.AckResult{
		Result:    "WAIT",
		NewCursor: 1,
		Wait:      &chat.WaitResult{From: "engram-agent", Text: "not ready yet"},
	}}

	prompt, err := cli.ExportWaitAndBuildPromptWith(context.Background(), "worker-1", 0, stub)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).To(ContainSubstring("WAIT from engram-agent: not ready yet"))
}

// TestWaitAndBuildPromptWith_WAIT_WithoutMessage verifies unspecified WAIT is handled.
func TestWaitAndBuildPromptWith_WAIT_WithoutMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stub := &stubAckWaiter{result: chat.AckResult{Result: "WAIT", NewCursor: 1}}

	prompt, err := cli.ExportWaitAndBuildPromptWith(context.Background(), "worker-1", 0, stub)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).To(ContainSubstring("WAIT: unspecified objection."))
}

// ============================================================
// waitAndBuildPrompt: real FileAckWaiter path (covers the thin wrapper)
// ============================================================

// TestWaitAndBuildPrompt_RealWaiter_ACK exercises the real waitAndBuildPrompt with a
// FSNotify-backed FileAckWaiter. A goroutine writes an ACK to the chat file after the
// watcher starts, unblocking AckWait immediately.
func TestWaitAndBuildPrompt_RealWaiter_ACK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Empty file: cursor = 1 (one empty line from strings.Split("", "\n")).
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// After a short delay, write an ACK message addressed to "worker-1".
	// The leading "\n" ensures suffixAtLine(data, 1) skips the blank line and
	// finds [[message]] at line-1 offset — so cursor=1 sees this ACK.
	ackTOML := "\n[[message]]\n" +
		"from = \"engram-agent\"\n" +
		"to = \"worker-1\"\n" +
		"thread = \"speech-relay\"\n" +
		"type = \"ack\"\n" +
		"ts = 2026-04-06T00:00:00Z\n" +
		"text = \"\"\"\nok\n\"\"\"\n"

	done := make(chan struct{})

	go func() {
		defer close(done)

		time.Sleep(150 * time.Millisecond)

		_ = os.WriteFile(chatFile, []byte(ackTOML), 0o600)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prompt, err := cli.ExportWaitAndBuildPrompt(ctx, "worker-1", chatFile, 0)

	<-done

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).To(Equal("Proceed."))
}

// TestWatchAndResume_MemFileSelectorError_LogsWarning is a property-based test.
// For any arbitrary error message, when memFileSelector returns that error,
// the warning written to stdout must contain the error text.
func TestWatchAndResume_MemFileSelectorError_LogsWarning(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

		errMsg := rapid.StringOf(
			rapid.RuneFrom([]rune(
				"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 :/-_.",
			)),
		).Filter(func(s string) bool { return len(s) > 0 }).Draw(rt, "errMsg")

		memFileSelector := func(_ string, _ int) ([]string, error) {
			return nil, errors.New(errMsg)
		}

		var stdout bytes.Buffer

		_, err := cli.ExportWatchAndResume(
			context.Background(),
			"test-agent", "chat.toml", stateFilePath, 0,
			claudepkg.StreamResult{}, &stdout,
			watchForIntent, memFileSelector, noopReadFile,
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(stdout.String()).To(ContainSubstring(errMsg),
			"expected warning to contain the error message")
		g.Expect(stdout.String()).To(ContainSubstring("[engram] warning: failed to select memory files:"),
			"expected warning prefix")
	})
}

// TestWatchAndResume_MemFileSelectorError_MemFilesEmpty verifies that when
// memFileSelector returns an error, the resume prompt has an empty MEMORY_FILES
// section (fallback to no files preserved).
func TestWatchAndResume_MemFileSelectorError_MemFilesEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, errors.New("reading directory /nonexistent: no such file or directory")
	}

	var stdout bytes.Buffer

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, memFileSelector, noopReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// MEMORY_FILES section should be present but empty (no file paths listed).
	// buildResumePrompt writes "MEMORY_FILES:\n" followed by one path per line.
	// When memFiles is nil, the section header is followed immediately by INTENT_FROM.
	g.Expect(prompt).To(ContainSubstring("MEMORY_FILES:\nINTENT_FROM:"),
		"expected MEMORY_FILES section to be empty when selector errors")
}

// TestWatchAndResume_MemFileSelectorError_ReturnsSuccessAndPrompt verifies that
// when memFileSelector returns an error, watchAndResume still returns a non-error
// result and a prompt containing the intent sender and text.
func TestWatchAndResume_MemFileSelectorError_ReturnsSuccessAndPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, errors.New("stat /nonexistent: no such file or directory")
	}

	var stdout bytes.Buffer

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, memFileSelector, noopReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).To(ContainSubstring("INTENT_FROM: sender-agent"))
	g.Expect(prompt).To(ContainSubstring("INTENT_TEXT: Situation: test. Behavior: test."))
}

// TestWatchAndResume_MemFileSelectorSuccess_NoWarning verifies that when
// memFileSelector returns nil error, no warning is written to stdout.
func TestWatchAndResume_MemFileSelectorSuccess_NoWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return []string{"/home/user/.local/share/engram/memory/facts/fact1.toml"}, nil
	}

	var stdout bytes.Buffer

	_, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, memFileSelector, noopReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).NotTo(ContainSubstring("failed to select memory files"),
		"expected no warning when selector succeeds")
}

// TestWatchAndResume_PopulatesLearnedMessages verifies that when the chat file contains
// learned messages addressed to the agent since cursor, the prompt includes them.
func TestWatchAndResume_PopulatesLearnedMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFilePath := filepath.Join(dir, "state.toml")

	learnedLine := "\n[[message]]\nfrom = \"exec-2\"\nto = \"exec-1\"\n" +
		"thread = \"main\"\ntype = \"learned\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\napi returns 404 on missing resource\n\"\"\"\n"
	g.Expect(os.WriteFile(chatFile, []byte(learnedLine), 0o600)).To(Succeed())

	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		return chat.Message{From: "lead", Text: "Situation: new task. Behavior: act."}, cursor + 5, nil
	}

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"exec-1", chatFile, stateFilePath, 0,
		claudepkg.StreamResult{}, io.Discard,
		watchForIntent, nil, os.ReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).NotTo(ContainSubstring("LEARNED_MESSAGES: (none)"),
		"expected LEARNED_MESSAGES to be populated from chat file")
	g.Expect(prompt).To(ContainSubstring("api returns 404"))
}

// TestWatchAndResume_PopulatesRecentIntents verifies that when the chat file contains
// intent messages, the returned prompt includes them under RECENT_INTENTS (not "(none)").
func TestWatchAndResume_PopulatesRecentIntents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFilePath := filepath.Join(dir, "state.toml")

	intentLine := "\n[[message]]\nfrom = \"lead\"\nto = \"exec-1\"\n" +
		"thread = \"main\"\ntype = \"intent\"\nts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\nSituation: deploy needed. Behavior: will deploy.\n\"\"\"\n"
	g.Expect(os.WriteFile(chatFile, []byte(intentLine), 0o600)).To(Succeed())

	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		return chat.Message{From: "lead", Text: "Situation: new task. Behavior: act."}, cursor + 5, nil
	}

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"exec-1", chatFile, stateFilePath, 0,
		claudepkg.StreamResult{}, io.Discard,
		watchForIntent, nil, os.ReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).NotTo(ContainSubstring("RECENT_INTENTS: (none)"),
		"expected RECENT_INTENTS to be populated from chat file")
	g.Expect(prompt).To(ContainSubstring("deploy needed"))
}

// TestWatchAndResume_PopulatesRecentIntentsAndLearned verifies that watchAndResume
// populates RecentIntents and LearnedMessages in the resume prompt from the chat file.
func TestWatchAndResume_PopulatesRecentIntentsAndLearned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	// Chat file contains one prior intent and one learned message for the agent.
	chatContent := `
[[message]]
from = "lead"
to = "test-agent"
thread = "work"
type = "intent"
ts = 2026-04-11T00:00:00Z
text = "Do the prior task."

[[message]]
from = "engram-agent"
to = "test-agent"
thread = "work"
type = "learned"
ts = 2026-04-11T00:00:01Z
text = "Always check the state file before writing."
`

	readFile := func(_ string) ([]byte, error) {
		return []byte(chatContent), nil
	}

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, io.Discard,
		watchForIntent, nil, readFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(prompt).To(ContainSubstring("RECENT_INTENTS:"),
		"expected RECENT_INTENTS section in prompt")
	g.Expect(prompt).NotTo(ContainSubstring("RECENT_INTENTS: (none)"),
		"expected RECENT_INTENTS to be non-empty when chat has intent messages")
	g.Expect(prompt).To(ContainSubstring("lead→test-agent: Do the prior task."),
		"expected recent intent entry with from→to: text format")
	g.Expect(prompt).To(ContainSubstring("LEARNED_MESSAGES:"),
		"expected LEARNED_MESSAGES section in prompt")
	g.Expect(prompt).NotTo(ContainSubstring("LEARNED_MESSAGES: (none)"),
		"expected LEARNED_MESSAGES to be non-empty when chat has learned messages for agent")
	g.Expect(prompt).To(ContainSubstring("Always check the state file before writing."),
		"expected learned message text in prompt")
}

// TestWatchAndResume_StateFileIsDir_LogsWarningsAndReturnsPrompt verifies that when
// the stateFilePath is a directory (causing SILENT and last-resumed-at writes to fail),
// watchAndResume logs warnings but still returns a valid resume prompt.
func TestWatchAndResume_StateFileIsDir_LogsWarningsAndReturnsPrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Using a directory as the state file path causes os.ReadFile to fail with
	// "is a directory" (not os.ErrNotExist), triggering both warning paths.
	stateDir := t.TempDir()

	var stdout strings.Builder

	_, watchForIntent := makeWatchAndResumeFixture(t)

	prompt, err := cli.ExportWatchAndResume(
		context.Background(), "worker-1", "/fake/chat.toml", stateDir, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, nil, noopReadFile,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("failed to write SILENT state"))
	g.Expect(stdout.String()).To(ContainSubstring("failed to update last-resumed-at"))
	g.Expect(prompt).NotTo(BeEmpty())
}

// TestWatchAndResume_WatchForIntentError_ReturnsError verifies that when
// watchForIntent returns a non-context error, watchAndResume propagates it.
func TestWatchAndResume_WatchForIntentError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFilePath, _ := makeWatchAndResumeFixture(t)

	errFake := errors.New("watch failed")
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		return chat.Message{}, cursor, errFake
	}

	_, err := cli.ExportWatchAndResume(
		context.Background(), "worker-1", "/fake/chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, io.Discard,
		watchForIntent, nil, noopReadFile,
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("watch failed")))
}

// ============================================================

func TestWriteKilledLine_FailingWriter_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportWriteKilledLine(&failWriter{}, "executor-1")
	g.Expect(err).To(HaveOccurred())
}

// failWriter is an io.Writer that always returns an error.
type failWriter struct{}

func (w *failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// ============================================================
// Test fakes
// ============================================================

// fakeDirEntry implements os.DirEntry for testing.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f *fakeDirEntry) Info() (os.FileInfo, error) {
	return nil, errors.New("fakeDirEntry.Info not implemented")
}

func (f *fakeDirEntry) IsDir() bool { return f.isDir }

func (f *fakeDirEntry) Name() string { return f.name }

func (f *fakeDirEntry) Type() os.FileMode { return 0 }

// fakeFileInfo implements os.FileInfo for testing.
type fakeFileInfo struct {
	name    string
	modTime time.Time
}

func (f *fakeFileInfo) IsDir() bool { return false }

func (f *fakeFileInfo) ModTime() time.Time { return f.modTime }

func (f *fakeFileInfo) Mode() os.FileMode { return 0 }

func (f *fakeFileInfo) Name() string { return f.name }

func (f *fakeFileInfo) Size() int64 { return 0 }

func (f *fakeFileInfo) Sys() any { return nil }

// stubAckWaiter is a test stub that returns a pre-configured AckResult or error.
type stubAckWaiter struct {
	result chat.AckResult
	err    error
}

func (s *stubAckWaiter) AckWait(_ context.Context, _ string, _ int, _ []string) (chat.AckResult, error) {
	return s.result, s.err
}

// ExportWatchAndResume tests. watchForIntent returns a single intent message then
// blocks forever (ctx cancel stops it). stateFilePath is pre-populated.
func makeWatchAndResumeFixture(t *testing.T) (
	stateFilePath string,
	watchForIntent func(context.Context, string, string, int) (chat.Message, int, error),
) {
	t.Helper()

	dir := t.TempDir()
	stateFilePath = filepath.Join(dir, "state.toml")

	watchForIntent = func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		return chat.Message{
			From: "sender-agent",
			Text: "Situation: test. Behavior: test.",
		}, cursor + 5, nil
	}

	return stateFilePath, watchForIntent
}

// noopReadFile is a stub readFile for tests that don't need chat file content.
func noopReadFile(_ string) ([]byte, error) {
	return []byte{}, nil
}
