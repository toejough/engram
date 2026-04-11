package cli

import (
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
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
	ExportCollectLearned          = collectLearned
	ExportDefaultMemFileSelector  = defaultMemFileSelector
	ExportDefaultWatchForIntent   = defaultWatchForIntent
	ExportDeriveChatFilePath      = deriveChatFilePath
	ExportHasStartingRecord       = hasStartingRecord
	ExportInitWorkerStateRecords  = initWorkerStateRecords
	ExportLoadChatMessages        = func(path string) ([]chat.Message, error) {
		return loadChatMessages(path, os.ReadFile)
	}
	ExportMakeHoldChecker = func(chatFilePath string) func(string) bool {
		hcf := makeHoldChecker(chatFilePath)
		return func(name string) bool { return hcf(name) }
	}
	ExportNewFilePoster            = newFilePoster
	ExportOsAppendFile             = osAppendFile
	ExportOsTmuxKillPane           = osTmuxKillPane
	ExportOsTmuxSpawn              = osTmuxSpawn
	ExportOsTmuxSpawnWith          = osTmuxSpawnWith
	ExportOsTmuxVerifyPaneGone     = osTmuxVerifyPaneGone
	ExportOutputAckResult          = outputAckResult
	ExportParseIntentMarkerTO      = parseIntentMarkerTO
	ExportParseTmuxOutput          = parseTmuxOutput
	ExportReadModifyWriteStateFile = readModifyWriteStateFile
	ExportRecordSurfacing          = recordSurfacing
	ExportReleaseStaleHolds        = releaseStaleHolds
	ExportRenderFactContent        = renderFactContent
	ExportRenderMemoryContent      = renderMemoryContent
	ExportRenderMemoryMeta         = renderMemoryMeta
	ExportResolveChatFile          = resolveChatFile
	ExportResolveStateFile         = resolveStateFile
	ExportRunAgentKill             = runAgentKill
	ExportRunAgentRunWith          = runAgentRunWith
	ExportRunAgentSpawn            = runAgentSpawn
	ExportRunDispatchAssign        = runDispatchAssign
	ExportRunDispatchDispatch      = runDispatchDispatch
	ExportRunDispatchDrain         = runDispatchDrain
	ExportRunDispatchStart         = runDispatchStart
	ExportRunDispatchStatus        = runDispatchStatus
	ExportRunDispatchStop          = runDispatchStop
	ExportSelectMemoryFiles        = selectMemoryFiles
	ExportSelectRecentIntents      = selectRecentIntents
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

// ExportDispatchLoop exposes dispatchLoop for blackbox tests.
func ExportDispatchLoop(
	ctx context.Context,
	workerChans map[string]chan chat.Message,
	stateFilePath, chatFilePath string,
	cursor int,
	silentCh <-chan string,
) error {
	return dispatchLoop(ctx, workerChans, stateFilePath, chatFilePath, cursor, silentCh)
}

// ExportMultiStringFlagString returns the String() result of a multiStringFlag built from vals.
// Empty input returns the nil-receiver result.
func ExportMultiStringFlagString(vals ...string) string {
	if len(vals) == 0 {
		var nilFlag *multiStringFlag

		return nilFlag.String()
	}

	flagVal := new(multiStringFlag)

	for _, v := range vals {
		_ = flagVal.Set(v)
	}

	return flagVal.String()
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

// ExportRouteMessage calls routeMessage with a plain func(string) bool holdChecker
// (because holdCheckerFunc is unexported and tests can't name it).
func ExportRouteMessage(
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	holdChecker func(string) bool,
	stateFilePath, chatFilePath string,
	msg chat.Message,
	cursor int,
) {
	routeMessage(workerChans, deferred, holdCheckerFunc(holdChecker), stateFilePath, chatFilePath, msg, cursor)
}

// ExportRouteMessageWithPoster calls routeMessageWithPoster with a plain func holdChecker.
func ExportRouteMessageWithPoster(
	workerChans map[string]chan chat.Message,
	deferred map[string][]chat.Message,
	holdChecker func(string) bool,
	stateFilePath, chatFilePath string,
	poster *chat.FilePoster,
	msg chat.Message,
	cursor int,
) {
	routeMessageWithPoster(
		workerChans, deferred, holdCheckerFunc(holdChecker),
		stateFilePath, chatFilePath, poster, msg, cursor,
	)
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
		watchForIntent, nil, nil, memFileSelector,
	)
}

// ExportRunConversationLoopWithChannel calls runConversationLoopWith with a channel-based intent source.
// intents is a buffered channel of messages to deliver to the agent; silentCh receives the agent name
// when the session transitions to SILENT. Pass nil for either to use standalone (watch-based) mode.
func ExportRunConversationLoopWithChannel(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	intents <-chan chat.Message,
	silentCh chan<- string,
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	return runConversationLoopWith(
		ctx, runner, name, prompt, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, intents, silentCh, memFileSelector,
	)
}

// ExportRunConversationLoopWithStateHook is like ExportRunConversationLoopWith but wraps
// the runner's WriteState with an observer. The observer is called before every real
// WriteState write; if it returns an error, the write is aborted.
// Use this in tests that need to verify WriteState call patterns across sessions.
func ExportRunConversationLoopWithStateHook(
	ctx context.Context,
	name, prompt, chatFile, stateFile, claudeBinary string,
	stdout io.Writer,
	promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
	writeStateObserver func(state string) error,
) error {
	flags := agentRunFlags{name: name, prompt: prompt, chatFile: chatFile, stateFile: stateFile}
	runner := buildAgentRunner(flags, stateFile, chatFile, stdout)

	if writeStateObserver != nil {
		original := runner.WriteState
		runner.WriteState = func(state string) error {
			if observeErr := writeStateObserver(state); observeErr != nil {
				return observeErr
			}

			if original != nil {
				return original(state)
			}

			return nil
		}
	}

	return runConversationLoopWith(
		ctx, runner, name, prompt, chatFile, stateFile,
		claudeBinary, stdout, promptBuilder,
		watchForIntent, nil, nil, memFileSelector,
	)
}

// ExportRunDispatch is an instrumented version of the runDispatch semaphore logic.
// activeCount receives the concurrent active count each time a worker acquires the semaphore.
// Use this to verify that maxConcurrent is honored.
func ExportRunDispatch(
	ctx context.Context,
	workers []WorkerConfig,
	maxConcurrent int,
	chatFile, stateFile, claudeBinary string,
	activeCount chan<- int32,
) error {
	sem := make(chan struct{}, maxConcurrent)

	intentChans := make(map[string]chan chat.Message, len(workers))
	for _, w := range workers {
		intentChans[w.Name] = make(chan chat.Message, workerChannelCap)
	}

	cursor, _ := chatFileCursor(chatFile, os.ReadFile)
	silentCh := make(chan string, len(workers))

	var active atomic.Int32

	var wg sync.WaitGroup

	for _, w := range workers {
		wg.Add(1)

		go func(cfg WorkerConfig) {
			defer wg.Done()

			sem <- struct{}{}

			cur := active.Add(1)

			select {
			case activeCount <- cur:
			default:
			}

			defer func() {
				active.Add(-1)
				<-sem
			}()

			flags := agentRunFlags{name: cfg.Name, prompt: cfg.Prompt, chatFile: chatFile, stateFile: stateFile}
			runner := buildAgentRunner(flags, stateFile, chatFile, io.Discard)

			_ = runConversationLoopWith(
				ctx, runner, cfg.Name, cfg.Prompt, chatFile, stateFile,
				claudeBinary, io.Discard,
				waitAndBuildPrompt, defaultWatchForIntent,
				intentChans[cfg.Name], silentCh, defaultMemFileSelector,
			)
		}(w)
	}

	loopErr := dispatchLoop(ctx, intentChans, stateFile, chatFile, cursor, silentCh)

	wg.Wait()

	return loopErr
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
	readFile func(string) ([]byte, error),
) (string, error) {
	return watchAndResume(
		ctx, agentName, chatFilePath, stateFilePath, cursor, result, stdout,
		watchForIntent, nil, nil, memFileSelector, readFile,
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
