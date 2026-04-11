package cli

import (
	"context"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"engram/internal/chat"
	claudepkg "engram/internal/claude"
	"engram/internal/recall"
	"engram/internal/surface"
)

// Exported variables.
var (
	ExportApplyDataDirDefault     = applyDataDirDefault
	ExportApplyProjectSlugDefault = applyProjectSlugDefault
	ExportBuildClaudeCmd          = buildClaudeCmd
	ExportBuildRecallSurfacer     = buildRecallSurfacer
	ExportBuildResumePrompt       = buildResumePrompt
	ExportChatFileCursor          = func(path string) (int, error) { return chatFileCursor(path, os.ReadFile) }
	ExportDefaultMemFileSelector  = defaultMemFileSelector
	ExportDefaultWatchForIntent   = defaultWatchForIntent
	ExportDeriveChatFilePath      = deriveChatFilePath
	ExportLoadChatMessages        = func(path string) ([]chat.Message, error) {
		return loadChatMessages(path, os.ReadFile)
	}
	ExportOsAppendFile             = osAppendFile
	ExportOsTmuxKillPane           = osTmuxKillPane
	ExportOsTmuxSpawn              = osTmuxSpawn
	ExportOsTmuxSpawnWith          = osTmuxSpawnWith
	ExportOsTmuxVerifyPaneGone     = osTmuxVerifyPaneGone
	ExportOutputAckResult          = outputAckResult
	ExportParseTmuxOutput          = parseTmuxOutput
	ExportReadModifyWriteStateFile = readModifyWriteStateFile
	ExportRecordSurfacing          = recordSurfacing
	ExportRenderFactContent        = renderFactContent
	ExportRenderMemoryContent      = renderMemoryContent
	ExportRenderMemoryMeta         = renderMemoryMeta
	ExportResolveChatFile          = resolveChatFile
	ExportResolveStateFile         = resolveStateFile
	ExportRunAgentKill             = runAgentKill
	ExportRunAgentRunWith          = runAgentRunWith
	ExportRunAgentSpawn            = runAgentSpawn
	ExportSelectMemoryFiles        = selectMemoryFiles
	ExportWaitAndBuildPrompt       = waitAndBuildPrompt
	ExportWriteKilledLine          = writeKilledLine
)

// --- Factory functions for structs with unexported fields ---

// ExportBuildAgentRunner wraps buildAgentRunner for test access.
func ExportBuildAgentRunner(
	args AgentRunArgs, stateFilePath, chatFilePath string,
) claudepkg.Runner {
	flags := agentRunFlags{
		name:      args.Name,
		prompt:    args.Prompt,
		chatFile:  args.ChatFile,
		stateFile: args.StateFile,
	}

	return buildAgentRunner(flags, stateFilePath, chatFilePath, io.Discard)
}

// ExportNewHaikuCallerAdapter creates a haikuCallerAdapter for testing.
func ExportNewHaikuCallerAdapter(
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) recall.HaikuCaller {
	return &haikuCallerAdapter{caller: caller}
}

// ExportNewOsDirLister creates an osDirLister for testing.
func ExportNewOsDirLister() recall.DirLister {
	return &osDirLister{}
}

// ExportNewOsFileReader creates an osFileReader for testing.
func ExportNewOsFileReader() interface {
	Read(path string) ([]byte, error)
} {
	return &osFileReader{}
}

// ExportNewSurfaceRunnerAdapter creates a surfaceRunnerAdapter for testing.
func ExportNewSurfaceRunnerAdapter(surfacer *surface.Surfacer) SurfaceRunner {
	return &surfaceRunnerAdapter{surfacer: surfacer}
}

// ExportRunConversationLoopWith calls runConversationLoopWith with an injectable prompt builder.
// name, prompt, chatFile, stateFile identify the agent; claudeBinary is the fake binary in tests.
func ExportRunConversationLoopWith(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, name, prompt, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, memFileSelector,
	)
}

// ExportWaitAndBuildPromptWith calls waitAndBuildPromptWith with an injectable ackWaiter.
// cursor must be captured before claude -p starts (HARD RULE compliance).
func ExportWaitAndBuildPromptWith(
	ctx context.Context,
	agentName string,
	cursor int,
	waiter interface {
		AckWait(ctx context.Context, callerAgent string, cursor int, recipients []string) (chat.AckResult, error)
	},
) (string, error) {
	return waitAndBuildPromptWith(ctx, agentName, cursor, waiter)
}

// ExportWatchAndResume calls watchAndResume with all injectable dependencies for testing.
func ExportWatchAndResume(
	ctx context.Context,
	agentName, chatFilePath, stateFilePath string,
	cursor int,
	result claudepkg.StreamResult,
	stdout io.Writer,
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) (string, error) {
	return watchAndResume(
		ctx, agentName, chatFilePath, stateFilePath, cursor, result, stdout, watchForIntent, memFileSelector,
	)
}

// SetTestPaneKiller installs a test-only pane killer and serializes parallel tests
// that override the same global. The caller must not defer a nil reset — cleanup is
// handled automatically via t.Cleanup.
func SetTestPaneKiller(tb testing.TB, f func(paneID string) error) {
	tb.Helper()
	testPaneKillerMu.Lock()

	testPaneKiller = f

	tb.Cleanup(func() {
		testPaneKiller = nil
		testPaneKillerMu.Unlock()
	})
}

// SetTestPaneVerifier installs a test-only pane verifier and serializes parallel tests
// that override the same global. The caller must not defer a nil reset — cleanup is
// handled automatically via t.Cleanup.
func SetTestPaneVerifier(tb testing.TB, f func(paneID string) error) {
	tb.Helper()
	testPaneVerifierMu.Lock()

	testPaneVerifier = f

	tb.Cleanup(func() {
		testPaneVerifier = nil
		testPaneVerifierMu.Unlock()
	})
}

// SetTestSpawnAckMaxWait installs a test-only ack-wait timeout and serializes parallel tests
// that override the same global. The caller must not defer a nil reset — cleanup is
// handled automatically via t.Cleanup.
func SetTestSpawnAckMaxWait(tb testing.TB, d time.Duration) {
	tb.Helper()
	testSpawnAckMaxWaitMu.Lock()

	testSpawnAckMaxWait = d

	tb.Cleanup(func() {
		testSpawnAckMaxWait = 0
		testSpawnAckMaxWaitMu.Unlock()
	})
}

// unexported variables.
var (
	testPaneKillerMu      sync.Mutex
	testPaneVerifierMu    sync.Mutex
	testSpawnAckMaxWaitMu sync.Mutex
)
