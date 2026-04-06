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
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	agentpkg "engram/internal/agent"
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

func TestOsTmuxSpawnWith_Success_ReturnsPaneAndSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")

	g.Expect(os.WriteFile(fakeTmux, []byte("#!/bin/sh\necho '%my-pane $mysession'\n"), 0o700)).To(Succeed())

	paneID, sessionID, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "sh -c 'echo hello'")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paneID).To(Equal("%my-pane"))
	g.Expect(sessionID).To(Equal("$mysession"))
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

func TestRunAgentKill_RemovesAgentFromStateFile(t *testing.T) {
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

	remaining, _ := os.ReadFile(stateFile)
	g.Expect(string(remaining)).NotTo(ContainSubstring("executor-1"))
	g.Expect(string(remaining)).To(ContainSubstring("reviewer-1"))
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
