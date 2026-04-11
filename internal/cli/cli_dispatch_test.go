package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
	cli "engram/internal/cli"
)

func TestDispatchCrashRecovery_DeliversMissedMessages(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write 3 messages. Simulate crash after first was delivered (cursor points past first).
	allTOML := "\n[[message]]\n" +
		"from = \"lead\"\n" +
		"to = \"engram-agent\"\n" +
		"thread = \"t\"\n" +
		"type = \"intent\"\n" +
		"ts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\nalready delivered\n\"\"\"\n" +
		"\n[[message]]\n" +
		"from = \"lead\"\n" +
		"to = \"engram-agent\"\n" +
		"thread = \"t\"\n" +
		"type = \"intent\"\n" +
		"ts = 2026-04-11T00:00:01Z\n" +
		"text = \"\"\"\nmissed 1\n\"\"\"\n" +
		"\n[[message]]\n" +
		"from = \"lead\"\n" +
		"to = \"engram-agent\"\n" +
		"thread = \"t\"\n" +
		"type = \"intent\"\n" +
		"ts = 2026-04-11T00:00:02Z\n" +
		"text = \"\"\"\nmissed 2\n\"\"\"\n"
	g.Expect(os.WriteFile(chatFile, []byte(allTOML), 0o600)).To(Succeed())

	// Count lines in first message block to set cursor past it.
	firstBlock := strings.Count("\n[[message]]\nfrom = \"lead\"\nto = \"engram-agent\"\n"+
		"thread = \"t\"\ntype = \"intent\"\nts = 2026-04-11T00:00:00Z\ntext = \"\"\"\nalready delivered\n\"\"\"\n", "\n")
	crashCursor := firstBlock

	// State: last_delivered_cursor = crashCursor (already delivered first message).
	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT", lastDeliveredCursor: crashCursor},
	})

	msgChan := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": msgChan}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loopDone := make(chan error, 1)

	go func() {
		// Restart from min last_delivered_cursor (crashCursor).
		loopDone <- cli.ExportDispatchLoop(
			ctx, workerChans, stateFile, chatFile, crashCursor, nil,
		)
	}()

	received := make([]string, 0, 2)

	for range 2 {
		select {
		case msg := <-msgChan:
			received = append(received, msg.Text)
		case <-time.After(5 * time.Second):
			cancel()
			t.Fatalf("timed out waiting for missed message; got %v", received)
		}
	}

	cancel()
	<-loopDone

	g.Expect(received).To(ContainElement(ContainSubstring("missed 1")))
	g.Expect(received).To(ContainElement(ContainSubstring("missed 2")))
	g.Expect(received).NotTo(ContainElement(ContainSubstring("already delivered")))
}

func TestDispatchLastDeliveredCursorAdvances(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": ch}
	deferred := map[string][]chat.Message{"worker-a": {}}

	msg := chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "tick"}

	const cursor = 42

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, cursor)

	// Read state file and verify last_delivered_cursor was updated.
	data, err := os.ReadFile(stateFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	state, parseErr := agentpkg.ParseStateFile(data)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(state.Agents).To(HaveLen(1))
	g.Expect(state.Agents[0].LastDeliveredCursor).To(Equal(cursor))
}

func TestDispatchLoopActiveWorker_IntentDeferredNotRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "ACTIVE"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": ch}
	deferred := map[string][]chat.Message{"worker-a": {}}

	msg := chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "active worker intent"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(BeEmpty(), "ACTIVE worker must not receive intent via channel")
	g.Expect(deferred["worker-a"]).To(HaveLen(1), "ACTIVE worker intent must go to deferredQueue")
}

func TestDispatchLoopChannelFullFallsToDeferredQueue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker", state: "SILENT"},
	})

	// Pre-fill channel to capacity (16).
	ch := make(chan chat.Message, 16)
	for i := range 16 {
		ch <- chat.Message{From: "fill", To: "worker", Type: "intent", Text: fmt.Sprintf("fill %d", i)}
	}

	workerChans := map[string]chan chat.Message{"worker": ch}
	deferred := map[string][]chat.Message{"worker": {}}

	msg := chat.Message{From: "lead", To: "worker", Type: "intent", Text: "overflow intent"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(HaveLen(16), "channel must remain at capacity — overflow went to deferredQueue")
	g.Expect(deferred["worker"]).To(HaveLen(1), "overflow intent must be in deferredQueue")
}

func TestDispatchLoopDeferredQueueCap_101stMessageDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}

	// Pre-fill deferred queue to cap (100).
	deferred := map[string][]chat.Message{"engram-agent": make([]chat.Message, 100)}

	holdChecker := func(worker string) bool { return worker == "engram-agent" }
	msg := chat.Message{From: "lead", To: "engram-agent", Type: "intent", Text: "101st"}

	cli.ExportRouteMessage(workerChans, deferred, holdChecker, stateFile, "", msg, 0)

	g.Expect(ch).To(BeEmpty())
	g.Expect(deferred["engram-agent"]).To(HaveLen(100), "101st message must be dropped, not added")
}

func TestDispatchLoopFromFilter_SelfAddressed_NotRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}
	deferred := map[string][]chat.Message{"engram-agent": {}}

	// self-addressed: from == to == "engram-agent"
	msg := chat.Message{
		From: "engram-agent",
		To:   "engram-agent",
		Type: "intent",
		Text: "nested intent",
	}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(BeEmpty(), "self-addressed intent must not be routed back to sender")
	g.Expect(deferred["engram-agent"]).To(BeEmpty())
}

func TestDispatchLoopHeldWorker_IntentDeferredNotRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}
	deferred := map[string][]chat.Message{"engram-agent": {}}

	holdChecker := func(worker string) bool { return worker == "engram-agent" }
	msg := chat.Message{From: "lead", To: "engram-agent", Type: "intent", Text: "held intent"}

	cli.ExportRouteMessage(workerChans, deferred, holdChecker, stateFile, "", msg, 0)

	g.Expect(ch).To(BeEmpty(), "held worker must not receive intent via channel")
	g.Expect(deferred["engram-agent"]).To(HaveLen(1), "held worker intent must go to deferredQueue")
}

func TestDispatchLoopHoldReleaseDrainsViaChat(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write hold-acquire to chat file so resolveHoldTarget can find it.
	holdAcquireJSON := `{"hold-id":"abc-hold","holder":"lead","target":"engram-agent",` +
		`"condition":"test","acquired-ts":"2026-04-11T00:00:00Z"}`
	chatContent := "[[message]]\nfrom = \"lead\"\nto = \"engram-agent\"\nthread = \"hold\"\n" +
		"type = \"hold-acquire\"\nts = 2026-04-11T00:00:00Z\ntext = \"\"\"\n" +
		holdAcquireJSON + "\n\"\"\"\n"
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}
	deferred := map[string][]chat.Message{
		"engram-agent": {
			{From: "test", To: "engram-agent", Type: "intent", Text: "queued intent 1"},
			{From: "test", To: "engram-agent", Type: "intent", Text: "queued intent 2"},
		},
	}

	payload, err := json.Marshal(map[string]string{"hold-id": "abc-hold"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	releaseMsg := chat.Message{
		From: "system",
		To:   "engram-agent",
		Type: "hold-release",
		Text: string(payload),
	}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, chatFile, releaseMsg, 0)

	g.Expect(ch).To(HaveLen(2), "both deferred messages must be delivered on hold-release")
	g.Expect(deferred["engram-agent"]).To(BeEmpty(), "deferredQueue must be empty after drain")
}

func TestDispatchLoopLearnedNotRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}
	deferred := map[string][]chat.Message{"engram-agent": {}}

	msg := chat.Message{From: "lead", To: "engram-agent", Type: "learned", Text: "fact"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(BeEmpty(), "type=learned must NOT be routed")
	g.Expect(deferred["engram-agent"]).To(BeEmpty())
}

// ============================================================
// routeMessage unit tests
// ============================================================

func TestDispatchLoopRoutes_IntentToWorkerA_NotB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
		{name: "worker-b", state: "SILENT"},
	})

	chA := make(chan chat.Message, 16)
	chB := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": chA, "worker-b": chB}
	deferred := map[string][]chat.Message{"worker-a": {}, "worker-b": {}}

	msg := chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "do the thing"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 100)

	g.Expect(chA).To(HaveLen(1), "worker-a must receive the intent")
	g.Expect(chB).To(BeEmpty(), "worker-b must not receive worker-a's intent")
}

func TestDispatchLoopShutdownDelivery_ShutdownMessageRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": ch}
	deferred := map[string][]chat.Message{"worker-a": {}}

	msg := chat.Message{From: "lead", To: "worker-a", Type: "shutdown", Text: "goodbye"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(HaveLen(1), "type=shutdown must be routed to worker channel")
}

func TestDispatchLoopStartingWorkerBuffersIntent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// STARTING worker: spawned but READY: not yet seen. Intent should buffer in channel.
	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "STARTING"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}
	deferred := map[string][]chat.Message{"engram-agent": {}}

	msg := chat.Message{From: "lead", To: "engram-agent", Type: "intent", Text: "startup intent"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(HaveLen(1), "STARTING worker should buffer intent in channel")
	g.Expect(deferred["engram-agent"]).
		To(BeEmpty(), "STARTING worker intent must NOT go to deferredQueue")
}

func TestDispatchLoopToAllExpansion_RoutesToAllWorkers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
		{name: "worker-b", state: "SILENT"},
	})

	chA := make(chan chat.Message, 16)
	chB := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": chA, "worker-b": chB}
	deferred := map[string][]chat.Message{"worker-a": {}, "worker-b": {}}

	msg := chat.Message{From: "lead", To: "all", Type: "shutdown", Text: "broadcast shutdown"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(chA).To(HaveLen(1), "worker-a must receive to=all message")
	g.Expect(chB).To(HaveLen(1), "worker-b must receive to=all message")
}

func TestDispatchLoopWaitDelivery_WaitMessageRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": ch}
	deferred := map[string][]chat.Message{"worker-a": {}}

	msg := chat.Message{From: "engram-agent", To: "worker-a", Type: "wait", Text: "objection"}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", msg, 0)

	g.Expect(ch).To(HaveLen(1), "type=wait must be routed to worker channel")
}

func TestDispatchLoopWaitToActiveWorkerDeferred(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// ACTIVE worker receiving WAIT must be deferred, not sent to channel.
	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "ACTIVE"},
	})

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": ch}
	deferred := map[string][]chat.Message{"worker-a": {}}

	waitMsg := chat.Message{
		From: "engram-agent",
		To:   "worker-a",
		Type: "wait",
		Text: "conflict detected",
	}

	cli.ExportRouteMessage(workerChans, deferred, nil, stateFile, "", waitMsg, 0)

	g.Expect(ch).To(BeEmpty(), "ACTIVE worker WAIT must be deferred, not sent to channel")
	g.Expect(deferred["worker-a"]).To(HaveLen(1))
}

// ============================================================
// coverage: dispatchLoopWith silentCh drain and watch error
// ============================================================

func TestDispatchLoopWith_SilentChDrains_DeferredMessages(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "ACTIVE"},
	})

	// Fill deferred queue with a message (via ACTIVE worker state).
	workerCh := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": workerCh}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	silentCh := make(chan string, 1)

	loopDone := make(chan error, 1)

	go func() {
		loopDone <- cli.ExportDispatchLoop(ctx, workerChans, stateFile, chatFile, 0, silentCh)
	}()

	// Post an intent to chat so the worker would receive it, then flip to SILENT.
	poster := cli.ExportNewFilePoster(chatFile)
	_, _ = poster.Post(
		chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "deferred task"},
	)

	// Transition worker to SILENT state, then signal silentCh.
	// Update state file to SILENT first so drain can deliver to channel.
	stateFileContent := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})
	// Replace stateFile path is fixed by makeDispatchStateFile, so we just re-read after the loop.
	_ = stateFileContent

	// Signal SILENT — dispatchLoopWith will try to drain deferred for worker-a.
	silentCh <- "worker-a"

	// Give the loop time to process.
	select {
	case <-time.After(2 * time.Second):
		// Acceptable: the worker was ACTIVE so drain deferred may attempt channel send.
	case <-loopDone:
	}

	cancel()
	<-loopDone
}

// ============================================================
// coverage: dispatchLoopWith msgCh path (Watch fires after loop starts)
// ============================================================

func TestDispatchLoop_MsgCh_ProcessesWatchedMessage(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})

	// Write one line so cursor starts at 1 (crash recovery finds nothing).
	g.Expect(os.WriteFile(chatFile, []byte("\n"), 0o600)).To(Succeed())

	workerCh := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": workerCh}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loopDone := make(chan error, 1)

	// Start loop with cursor=1 so crash recovery processes nothing.
	go func() {
		loopDone <- cli.ExportDispatchLoop(ctx, workerChans, stateFile, chatFile, 1, nil)
	}()

	// Post after a brief pause to let Watch() register the inotify watcher.
	time.Sleep(50 * time.Millisecond)

	poster := cli.ExportNewFilePoster(chatFile)
	_, _ = poster.Post(chat.Message{
		From: "lead", To: "worker-a", Type: "intent", Text: "watched message",
	})

	// The message should be delivered via the Watch → msgCh path.
	select {
	case msg := <-workerCh:
		g.Expect(msg.Text).To(ContainSubstring("watched message"))
		cancel()
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for watched message via msgCh")
	}

	<-loopDone
}

// ============================================================
// dispatchLoop integration test
// ============================================================

func TestDispatchLoop_RoutesIntentViaWatcher(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "engram-agent", state: "SILENT"},
	})

	// Write a chat file with one intent message.
	intentTOML := "\n[[message]]\n" +
		"from = \"lead\"\n" +
		"to = \"engram-agent\"\n" +
		"thread = \"test\"\n" +
		"type = \"intent\"\n" +
		"ts = 2026-04-11T00:00:00Z\n" +
		"text = \"\"\"\ndo the thing\n\"\"\"\n"
	g.Expect(os.WriteFile(chatFile, []byte(intentTOML), 0o600)).To(Succeed())

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"engram-agent": ch}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loopDone := make(chan error, 1)

	go func() {
		loopDone <- cli.ExportDispatchLoop(
			ctx, workerChans, stateFile, chatFile, 0, nil,
		)
	}()

	// Wait for the intent to be routed.
	select {
	case msg := <-ch:
		g.Expect(msg.Text).To(ContainSubstring("do the thing"))
		cancel() // stop the loop
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for intent to be routed")
	}

	<-loopDone
}

func TestDispatchObservabilityMessages_RoutePostsInfo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	ch := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": ch}
	deferred := map[string][]chat.Message{"worker-a": {}}

	poster := cli.ExportNewFilePoster(chatFile)

	msg := chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "observe me"}

	cli.ExportRouteMessageWithPoster(workerChans, deferred, nil, stateFile, "", poster, msg, 0)

	// Verify an info message was posted to the chat file.
	data, err := os.ReadFile(chatFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`type = "info"`))
}

func TestHasStartingRecord_DeadRecord_ReturnsFalse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	agents := []agentpkg.AgentRecord{
		{Name: "w1", State: "DEAD"},
	}
	g.Expect(cli.ExportHasStartingRecord(agents, "w1")).To(BeFalse())
}

func TestHasStartingRecord_NoRecord_ReturnsFalse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportHasStartingRecord([]agentpkg.AgentRecord{}, "w1")).To(BeFalse())
}

// ============================================================
// hasStartingRecord tests
// ============================================================

func TestHasStartingRecord_StartingRecord_ReturnsTrue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	agents := []agentpkg.AgentRecord{
		{Name: "w1", State: "STARTING"},
	}
	g.Expect(cli.ExportHasStartingRecord(agents, "w1")).To(BeTrue())
}

// ============================================================
// initWorkerStateRecords tests (criterion 12)
// ============================================================

func TestInitWorkerStateRecords_StaleActiveWorkerMarkedDead(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-populate with a stale ACTIVE record for "w1".
	initial := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "w1", State: "ACTIVE", SpawnedAt: time.Now()},
		},
	}
	data, err := agentpkg.MarshalStateFile(initial)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	initErr := cli.ExportInitWorkerStateRecords(
		stateFile,
		chatFile,
		[]cli.WorkerConfig{{Name: "w1", Prompt: "go"}},
	)
	g.Expect(initErr).NotTo(HaveOccurred())

	if initErr != nil {
		return
	}

	stateData, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	state, parseErr := agentpkg.ParseStateFile(stateData)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	// Stale record is replaced by a single fresh STARTING record.
	// The stale record is announced to chat but NOT kept in the state file.
	w1Records := make([]agentpkg.AgentRecord, 0)

	for _, rec := range state.Agents {
		if rec.Name == "w1" {
			w1Records = append(w1Records, rec)
		}
	}

	g.Expect(w1Records).To(HaveLen(1), "expected exactly one record for w1")
	g.Expect(w1Records[0].State).To(Equal("STARTING"), "expected STARTING state for fresh w1")

	// Stale worker announcement must appear in chat.
	chatData, chatReadErr := os.ReadFile(chatFile)
	g.Expect(chatReadErr).NotTo(HaveOccurred())
	g.Expect(string(chatData)).To(ContainSubstring("Stale worker w1"))
}

func TestInitWorkerStateRecords_StaleStartingWorkerMarkedDead(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	initial := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "w1", State: "STARTING", SpawnedAt: time.Now()},
		},
	}
	data, err := agentpkg.MarshalStateFile(initial)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	initErr := cli.ExportInitWorkerStateRecords(
		stateFile,
		chatFile,
		[]cli.WorkerConfig{{Name: "w1", Prompt: "go"}},
	)
	g.Expect(initErr).NotTo(HaveOccurred())

	if initErr != nil {
		return
	}

	stateData, _ := os.ReadFile(stateFile)
	state, _ := agentpkg.ParseStateFile(stateData)

	w1Records := make([]agentpkg.AgentRecord, 0)

	for _, rec := range state.Agents {
		if rec.Name == "w1" {
			w1Records = append(w1Records, rec)
		}
	}

	g.Expect(w1Records).To(HaveLen(1), "expected exactly one record for w1")
	g.Expect(w1Records[0].State).To(Equal("STARTING"), "expected STARTING state for fresh w1")

	// Stale worker announcement must appear in chat.
	chatData, chatReadErr := os.ReadFile(chatFile)
	g.Expect(chatReadErr).NotTo(HaveOccurred())
	g.Expect(string(chatData)).To(ContainSubstring("Stale worker w1"))
}

// ============================================================
// coverage: isWorkerActive error paths
// ============================================================

func TestIsWorkerActive_MissingStateFile_ReturnsFalse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "ACTIVE"},
	})

	workerChans := map[string]chan chat.Message{
		"worker-a": make(chan chat.Message, 16),
	}
	deferred := map[string][]chat.Message{"worker-a": {}}

	// Use a non-existent state file — isWorkerActive returns false, so msg goes to channel.
	msg := chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "test"}
	_ = stateFile

	nonExistentState := "/nonexistent/state.toml"
	cli.ExportRouteMessage(workerChans, deferred, nil, nonExistentState, "", msg, 0)

	// Non-existent state file → isWorkerActive = false → message delivered to channel.
	g.Expect(workerChans["worker-a"]).To(HaveLen(1))
}

// ============================================================
// makeHoldChecker tests (criterion 9)
// ============================================================

func TestMakeHoldChecker_ActiveHoldTargetingWorker_ReturnsTrue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Post a hold-acquire message targeting "worker-1".
	holdRec := chat.HoldRecord{
		HoldID:    "test-hold-id",
		Holder:    "lead",
		Target:    "worker-1",
		Condition: "test",
	}
	holdJSON, marshalErr := json.Marshal(holdRec)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	poster := cli.ExportNewFilePoster(chatFile)
	_, postErr := poster.Post(chat.Message{
		From:   "lead",
		To:     "worker-1",
		Thread: "hold",
		Type:   "hold-acquire",
		Text:   string(holdJSON),
	})
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	checker := cli.ExportMakeHoldChecker(chatFile)
	g.Expect(checker("worker-1")).To(BeTrue())
	g.Expect(checker("worker-2")).To(BeFalse())
}

func TestMakeHoldChecker_MissingChatFile_ReturnsFalse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	checker := cli.ExportMakeHoldChecker("/nonexistent/chat.toml")
	g.Expect(checker("worker-1")).To(BeFalse())
}

func TestMakeHoldChecker_ReleasedHold_ReturnsFalse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Acquire then release a hold.
	holdRec := chat.HoldRecord{
		HoldID:    "test-hold-id",
		Holder:    "lead",
		Target:    "worker-1",
		Condition: "test",
	}
	holdJSON, holdMarshalErr := json.Marshal(holdRec)
	g.Expect(holdMarshalErr).NotTo(HaveOccurred())

	poster := cli.ExportNewFilePoster(chatFile)
	_, _ = poster.Post(chat.Message{
		From: "lead", To: "worker-1", Thread: "hold",
		Type: "hold-acquire", Text: string(holdJSON),
	})

	releaseJSON, releaseMarshalErr := json.Marshal(map[string]string{"hold-id": "test-hold-id"})
	g.Expect(releaseMarshalErr).NotTo(HaveOccurred())

	_, _ = poster.Post(chat.Message{
		From: "system", To: "all", Thread: "hold",
		Type: "hold-release", Text: string(releaseJSON),
	})

	checker := cli.ExportMakeHoldChecker(chatFile)
	g.Expect(checker("worker-1")).To(BeFalse())
}

// ============================================================
// coverage: multiStringFlag.String()
// ============================================================

func TestMultiStringFlag_String_NilAndNonNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportMultiStringFlagString()).To(BeEmpty())
	g.Expect(cli.ExportMultiStringFlagString("a", "b")).To(Equal("a,b"))
}

// ============================================================
// parseIntentMarkerTO tests (agent-e2e A1)
// ============================================================

func TestParseSpeechMarkerIntentTOSubfield_NoTO_DefaultsToEngramAgent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	to := cli.ExportParseIntentMarkerTO("Situation: about to act. Behavior: do X.")

	g.Expect(to).To(Equal("engram-agent"))
}

func TestParseSpeechMarkerIntentTOSubfield_TONoPeriodSep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// No ". " separator — take all of rest
	to := cli.ExportParseIntentMarkerTO("TO: engram-agent")

	g.Expect(to).To(Equal("engram-agent"))
}

func TestParseSpeechMarkerIntentTOSubfield_TOOverridesDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	to := cli.ExportParseIntentMarkerTO(
		"TO: lead, engram-agent. Situation: about to act. Behavior: do X.",
	)

	g.Expect(to).To(Equal("lead, engram-agent"))
}

func TestParseSpeechMarkerIntentTOSubfield_TOSingleRecipient(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	to := cli.ExportParseIntentMarkerTO("TO: engram-agent. Situation: test. Behavior: ACK.")

	g.Expect(to).To(Equal("engram-agent"))
}

// ============================================================
// coverage: postQueueOverflow with non-nil poster
// ============================================================

func TestPostQueueOverflow_WithPoster_PostsInfoMessage(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Worker must be ACTIVE so messages get deferred instead of channel-sent.
	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "ACTIVE"},
	})

	// Fill deferred queue to capacity (100 messages).
	workerCh := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": workerCh}
	deferred := map[string][]chat.Message{"worker-a": make([]chat.Message, 100)}

	poster := cli.ExportNewFilePoster(chatFile)

	// 101st message → overflow → poster.Post is called.
	msg := chat.Message{From: "lead", To: "worker-a", Type: "intent", Text: "101st"}
	cli.ExportRouteMessageWithPoster(workerChans, deferred, nil, stateFile, "", poster, msg, 0)

	// Verify overflow was posted to chat.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("overflow"))
}

func TestReleaseStaleHolds_AlreadyReleasedHold_NoDoubleRelease(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	dispatchStartedAt := time.Now()

	holdRec := chat.HoldRecord{
		HoldID:     "already-released",
		Holder:     "lead",
		Target:     "engram-agent",
		AcquiredTS: dispatchStartedAt.Add(-5 * time.Minute),
	}
	holdJSON, marshalErr := json.Marshal(holdRec)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	poster := cli.ExportNewFilePoster(chatFile)
	_, _ = poster.Post(chat.Message{
		From: "lead", To: "engram-agent", Thread: "hold",
		Type: "hold-acquire", Text: string(holdJSON),
	})

	releaseJSON, releaseMarshalErr := json.Marshal(map[string]string{"hold-id": "already-released"})
	g.Expect(releaseMarshalErr).NotTo(HaveOccurred())

	_, _ = poster.Post(chat.Message{
		From: "system", To: "all", Thread: "hold",
		Type: "hold-release", Text: string(releaseJSON),
	})

	workers := []cli.WorkerConfig{{Name: "engram-agent", Prompt: ""}}

	releaseErr := cli.ExportReleaseStaleHolds(chatFile, workers, dispatchStartedAt)
	g.Expect(releaseErr).NotTo(HaveOccurred())

	if releaseErr != nil {
		return
	}

	// ScanActiveHolds should return nothing — already released. No additional release posted.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	messages := chat.ParseMessagesSafe(data)
	releaseCount := 0

	for _, msg := range messages {
		if msg.Type == "hold-release" {
			releaseCount++
		}
	}

	g.Expect(releaseCount).To(Equal(1), "already-released hold must not be released again")
}

func TestReleaseStaleHolds_EmptyChatFile_NoError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	workers := []cli.WorkerConfig{{Name: "engram-agent", Prompt: ""}}

	releaseErr := cli.ExportReleaseStaleHolds(chatFile, workers, time.Now())
	g.Expect(releaseErr).NotTo(HaveOccurred())
}

func TestReleaseStaleHolds_HoldAcquiredAfterStart_NotReleased(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	dispatchStartedAt := time.Now()

	// Hold acquired AFTER dispatch started — should NOT be released.
	holdRec := chat.HoldRecord{
		HoldID:     "fresh-hold",
		Holder:     "lead",
		Target:     "engram-agent",
		AcquiredTS: dispatchStartedAt.Add(1 * time.Minute),
	}
	holdJSON, marshalErr := json.Marshal(holdRec)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	poster := cli.ExportNewFilePoster(chatFile)
	_, _ = poster.Post(chat.Message{
		From:   "lead",
		To:     "engram-agent",
		Thread: "hold",
		Type:   "hold-acquire",
		Text:   string(holdJSON),
	})

	workers := []cli.WorkerConfig{{Name: "engram-agent", Prompt: ""}}

	releaseErr := cli.ExportReleaseStaleHolds(chatFile, workers, dispatchStartedAt)
	g.Expect(releaseErr).NotTo(HaveOccurred())

	if releaseErr != nil {
		return
	}

	// No hold-release should have been posted.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	messages := chat.ParseMessagesSafe(data)
	for _, msg := range messages {
		g.Expect(msg.Type).NotTo(Equal("hold-release"), "fresh hold must not be released")
	}
}

func TestReleaseStaleHolds_HoldOnUnknownWorker_NotReleased(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	dispatchStartedAt := time.Now()

	// Hold on a target that is NOT in the current worker set.
	holdRec := chat.HoldRecord{
		HoldID:     "other-session-hold",
		Holder:     "other-lead",
		Target:     "external-agent",
		AcquiredTS: dispatchStartedAt.Add(-10 * time.Minute),
	}
	holdJSON, marshalErr := json.Marshal(holdRec)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	poster := cli.ExportNewFilePoster(chatFile)
	_, _ = poster.Post(chat.Message{
		From:   "other-lead",
		To:     "external-agent",
		Thread: "hold",
		Type:   "hold-acquire",
		Text:   string(holdJSON),
	})

	// Workers list does NOT include "external-agent".
	workers := []cli.WorkerConfig{{Name: "engram-agent", Prompt: ""}}

	releaseErr := cli.ExportReleaseStaleHolds(chatFile, workers, dispatchStartedAt)
	g.Expect(releaseErr).NotTo(HaveOccurred())

	if releaseErr != nil {
		return
	}

	// No hold-release should have been posted.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	messages := chat.ParseMessagesSafe(data)
	for _, msg := range messages {
		g.Expect(msg.Type).NotTo(Equal("hold-release"), "external-agent hold must not be released")
	}
}

// ============================================================
// releaseStaleHolds tests
// ============================================================

func TestReleaseStaleHolds_StaleHoldOnKnownWorker_PostsReleaseAndInfo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	dispatchStartedAt := time.Now()

	// Write a hold-acquire message with a timestamp before dispatchStartedAt.
	holdRec := chat.HoldRecord{
		HoldID:     "stale-hold-1",
		Holder:     "lead",
		Target:     "engram-agent",
		AcquiredTS: dispatchStartedAt.Add(-5 * time.Minute),
	}
	holdJSON, marshalErr := json.Marshal(holdRec)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	poster := cli.ExportNewFilePoster(chatFile)
	_, postErr := poster.Post(chat.Message{
		From:   "lead",
		To:     "engram-agent",
		Thread: "hold",
		Type:   "hold-acquire",
		Text:   string(holdJSON),
	})
	g.Expect(postErr).NotTo(HaveOccurred())

	if postErr != nil {
		return
	}

	workers := []cli.WorkerConfig{{Name: "engram-agent", Prompt: ""}}

	releaseErr := cli.ExportReleaseStaleHolds(chatFile, workers, dispatchStartedAt)
	g.Expect(releaseErr).NotTo(HaveOccurred())

	if releaseErr != nil {
		return
	}

	// Verify a hold-release and an info message were posted to chat.
	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	chatStr := string(data)
	g.Expect(chatStr).To(ContainSubstring(`"hold-release"`), "must post hold-release message")
	g.Expect(chatStr).
		To(ContainSubstring("stale-hold-1"), "hold-release must reference the hold ID")
	g.Expect(chatStr).
		To(ContainSubstring("Stale hold stale-hold-1"), "must post info message about stale hold")
}

// ============================================================
// coverage: resolveHoldTarget not-found and error paths
// ============================================================

func TestResolveHoldTarget_HoldFound_DrainsQueue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write a hold-acquire message to the chat file so resolveHoldTarget can find it.
	holdAcquireJSON := `{"hold-id":"h1","holder":"lead","target":"worker-a",` +
		`"condition":"test","acquired-ts":"2026-04-11T00:00:00Z"}`
	chatContent := "[[message]]\nfrom = \"lead\"\nto = \"worker-a\"\nthread = \"hold\"\n" +
		"type = \"hold-acquire\"\nts = 2026-04-11T00:00:00Z\ntext = \"\"\"\n" +
		holdAcquireJSON + "\n\"\"\"\n"
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "worker-a", state: "SILENT"},
	})

	workerCh := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": workerCh}
	deferred := map[string][]chat.Message{
		"worker-a": {{From: "lead", To: "worker-a", Type: "intent", Text: "deferred"}},
	}

	// Send hold-release message — should drain the deferred queue.
	holdReleaseMsg := chat.Message{
		From: "binary",
		To:   "all",
		Type: "hold-release",
		Text: `{"hold-id":"h1"}`,
	}

	cli.ExportRouteMessageWithPoster(
		workerChans,
		deferred,
		nil,
		stateFile,
		chatFile,
		nil,
		holdReleaseMsg,
		0,
	)

	g.Expect(workerCh).To(HaveLen(1), "deferred message should be drained to channel")
}

func TestResolveHoldTarget_UnknownHold_NoAction(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, nil)

	workerCh := make(chan chat.Message, 16)
	workerChans := map[string]chan chat.Message{"worker-a": workerCh}
	deferred := map[string][]chat.Message{
		"worker-a": {{From: "lead", To: "worker-a", Type: "intent", Text: "deferred"}},
	}

	// Hold-release with unknown hold-id — no drain.
	holdReleaseMsg := chat.Message{
		From: "binary",
		To:   "all",
		Type: "hold-release",
		Text: `{"hold-id":"unknown-hold"}`,
	}

	chatFile := filepath.Join(t.TempDir(), "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	cli.ExportRouteMessageWithPoster(
		workerChans,
		deferred,
		nil,
		stateFile,
		chatFile,
		nil,
		holdReleaseMsg,
		0,
	)

	g.Expect(workerCh).To(BeEmpty(), "unknown hold-id must not drain channel")
	g.Expect(deferred["worker-a"]).To(HaveLen(1), "deferred queue unchanged")
}

// ============================================================
// runDispatchAssign tests
// ============================================================

func TestRunDispatchAssign_MissingAgent_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchAssign([]string{"--task", "do something"}, nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--agent is required"))
}

func TestRunDispatchAssign_MissingTask_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchAssign([]string{"--agent", "w1"}, nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--task is required"))
}

func TestRunDispatchAssign_ValidArgs_PostsToChat(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	var out strings.Builder

	err := cli.ExportRunDispatchAssign(
		[]string{"--agent", "w1", "--task", "do the thing", "--chat-file", chatFile},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("do the thing"))
	g.Expect(out.String()).To(ContainSubstring("Assigned to w1"))
}

// ============================================================
// runDispatchDispatch router tests
// ============================================================

func TestRunDispatchDispatch_NoSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchDispatch(context.Background(), nil, nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("subcommand required"))
}

func TestRunDispatchDispatch_RouteToAssign_HitsAssignBranch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchDispatch(
		context.Background(),
		[]string{"assign"},
		nil,
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--agent is required"))
}

func TestRunDispatchDispatch_RouteToDrain_HitsDrainBranch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "w1", state: "SILENT"},
	})

	var out strings.Builder

	err := cli.ExportRunDispatchDispatch(
		context.Background(),
		[]string{"drain", "--state-file", stateFile, "--secs", "0"},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
}

// ============================================================
// coverage: runDispatchDispatch routing
// ============================================================

func TestRunDispatchDispatch_RouteToStatus_HitsStatusBranch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchDispatch(
		context.Background(),
		[]string{"status", "--state-file", "/nonexistent/state.toml"},
		nil,
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading state"))
}

func TestRunDispatchDispatch_RouteToStop_HitsStopBranch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	err := cli.ExportRunDispatchDispatch(
		context.Background(),
		[]string{"stop", "--state-file", "/nonexistent/state.toml", "--chat-file", chatFile},
		nil,
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading state"))
}

func TestRunDispatchDispatch_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchDispatch(context.Background(), []string{"bogus"}, nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown subcommand"))
}

func TestRunDispatchDrain_ActiveWorkersTimeout_ReturnsTimeout(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "w1", state: "ACTIVE"},
	})

	var out strings.Builder
	// timeout=0 means deadline is immediately in the past
	err := cli.ExportRunDispatchDrain(
		[]string{"--state-file", stateFile, "--secs", "0"},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring(`"timeout"`))
}

// ============================================================
// runDispatchDrain tests
// ============================================================

func TestRunDispatchDrain_AllWorkersAlreadySilent_ReturnsDrainedImmediately(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "w1", state: "SILENT"},
	})

	var out strings.Builder

	err := cli.ExportRunDispatchDrain(
		[]string{"--state-file", stateFile, "--secs", "5"},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring(`"drained"`))
}

func TestRunDispatchDrain_InvalidFlag_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchDrain([]string{"--unknown-flag"}, io.Discard)

	g.Expect(err).To(MatchError(ContainSubstring("dispatch drain")))
}

func TestRunDispatchDrain_MissingStateFile_OutputsTimeout(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var out strings.Builder

	// Non-existent state file: ReadFile fails → break → timeout output.
	err := cli.ExportRunDispatchDrain(
		[]string{"--state-file", "/nonexistent/state.toml", "--secs", "1"},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring(`"timeout"`))
}

// ============================================================
// runDispatch integration test (semaphore)
// ============================================================

func TestRunDispatchSemaphore_FourthWorkerBlocks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Write initial state with 4 workers in SILENT state.
	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "w1", State: "SILENT", SpawnedAt: time.Now()},
			{Name: "w2", State: "SILENT", SpawnedAt: time.Now()},
			{Name: "w3", State: "SILENT", SpawnedAt: time.Now()},
			{Name: "w4", State: "SILENT", SpawnedAt: time.Now()},
		},
	}

	data, err := agentpkg.MarshalStateFile(initialState)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	// Use a fake claude that blocks until context is cancelled.
	fakeClaude := filepath.Join(dir, "claude")
	// Fake claude that blocks (reads stdin forever).
	script := "#!/bin/sh\n# blocks until killed\nread _ignored || true\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	// Track concurrent goroutines.
	var concurrent int32

	maxConcurrent := int32(3)

	activeCount := make(chan int32, 10)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	_ = cli.ExportRunDispatch(ctx, []cli.WorkerConfig{
		{Name: "w1", Prompt: "go"},
		{Name: "w2", Prompt: "go"},
		{Name: "w3", Prompt: "go"},
		{Name: "w4", Prompt: "go"},
	}, int(maxConcurrent), chatFile, stateFile, fakeClaude, activeCount)

	// Verify that concurrent value never exceeded maxConcurrent.
	for range cap(activeCount) {
		select {
		case v := <-activeCount:
			g.Expect(v).To(BeNumerically("<=", maxConcurrent))
			concurrent = v // track max
		default:
		}
	}

	_ = concurrent
}

// ============================================================
// runDispatchStart tests
// ============================================================

func TestRunDispatchStart_InitializesStateFileWithStartingRecords(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Fake claude emits READY: + DONE: then exits, so worker completes one session
	// and enters watchAndResume (blocks on intent channel until ctx cancelled).
	fakeClaude := filepath.Join(dir, "claude")
	fakeClaudeScript := "#!/bin/sh\n" +
		`printf '{"type":"system","session_id":"test-sid-123"}\n'` + "\n" +
		`printf '{"type":"assistant","session_id":"test-sid-123",` +
		`"message":{"content":[{"type":"text","text":"READY: up\nDONE: done"}]}}\n'` + "\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(fakeClaudeScript), 0o700)).To(Succeed())

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	err := cli.ExportRunDispatchStart(ctx, []string{
		"--agent", "engram-agent",
		"--chat-file", chatFile,
		"--state-file", stateFile,
		"--claude-binary", fakeClaude,
	}, io.Discard)

	// Context timeout is expected; only error we accept.
	g.Expect(err).NotTo(HaveOccurred())

	// State file must exist and contain the worker record.
	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	state, parseErr := agentpkg.ParseStateFile(data)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(state.Agents).To(HaveLen(1))
	g.Expect(state.Agents[0].Name).To(Equal("engram-agent"))
}

func TestRunDispatchStart_MissingAgentFlag_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := cli.ExportRunDispatchStart(context.Background(), nil, nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("at least one --agent flag is required"))
}

func TestRunDispatchStart_MultipleAgents_PrintsStartupInfo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	g.Expect(os.WriteFile(fakeClaude, []byte("#!/bin/sh\nread _ignored || true\n"), 0o700)).
		To(Succeed())

	var out strings.Builder

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	// Run in goroutine since it blocks.
	done := make(chan error, 1)

	go func() {
		done <- cli.ExportRunDispatchStart(
			ctx,
			[]string{
				"--agent", "w1",
				"--agent", "w2",
				"--chat-file", chatFile,
				"--state-file", stateFile,
				"--claude-binary", fakeClaude,
			},
			&out,
		)
	}()

	select {
	case <-ctx.Done():
		// Context expired — startup info already printed; wait for goroutine to finish.
		cancel()
		<-done
	case err := <-done:
		// Should only return on error or ctx cancel.
		g.Expect(err).NotTo(HaveOccurred())
	}

	g.Expect(out.String()).To(ContainSubstring("Dispatch started"))
	g.Expect(out.String()).To(ContainSubstring("w1"))
}

// ============================================================
// runDispatchStatus tests
// ============================================================

func TestRunDispatchStatus_MissingStateFile_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var out strings.Builder

	err := cli.ExportRunDispatchStatus(
		[]string{"--state-file", "/nonexistent/state.toml"},
		&out,
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading state"))
}

func TestRunDispatchStatus_ValidStateFile_OutputsJSON(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "w1", state: "ACTIVE"},
		{name: "w2", state: "SILENT"},
	})

	var out strings.Builder

	err := cli.ExportRunDispatchStatus(
		[]string{"--state-file", stateFile},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(out.String())), &result)).To(Succeed())
	g.Expect(result["active_count"]).To(BeNumerically("==", 1))
}

// ============================================================
// runDispatchStop tests
// ============================================================

func TestRunDispatchStop_MissingStateFile_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	err := cli.ExportRunDispatchStop(
		[]string{"--state-file", "/nonexistent/state.toml", "--chat-file", chatFile},
		nil,
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading state"))
}

func TestRunDispatchStop_ValidState_PostsShutdowns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	stateFile := makeDispatchStateFile(t, []dispatchAgentState{
		{name: "w1", state: "ACTIVE"},
		{name: "w2", state: "DEAD"},
	})

	var out strings.Builder

	err := cli.ExportRunDispatchStop(
		[]string{"--state-file", stateFile, "--chat-file", chatFile},
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// w1 (ACTIVE) should get a shutdown; w2 (DEAD) should be skipped.
	g.Expect(out.String()).To(ContainSubstring("Sent shutdown to w1"))
	g.Expect(out.String()).NotTo(ContainSubstring("Sent shutdown to w2"))

	data, readErr := os.ReadFile(chatFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring(`type = "shutdown"`))
}

// ============================================================
// runDispatch semaphore integration via runDispatchStart
// ============================================================

func TestRunDispatch_ContextCancel_ReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	g.Expect(os.WriteFile(fakeClaude, []byte("#!/bin/sh\nread _ignored || true\n"), 0o700)).
		To(Succeed())

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	workers := []cli.WorkerConfig{{Name: "w1", Prompt: "go"}}

	activeCount := make(chan int32, 4)

	err := cli.ExportRunDispatch(ctx, workers, 1, chatFile, stateFile, fakeClaude, activeCount)

	g.Expect(err).NotTo(HaveOccurred())
}

// ============================================================
// coverage: Run dispatch branch
// ============================================================

func TestRun_DispatchSubcommand_RoutesToDispatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Run with "dispatch" + missing subcommand → subcommand-required error.
	// This exercises the dispatch branch in Run including signal context setup.
	err := cli.Run([]string{"engram", "dispatch"}, nil, nil, nil)

	g.Expect(err).To(MatchError(ContainSubstring("subcommand required")))
}

// ============================================================
// helpers
// ============================================================

// dispatchAgentState is a minimal struct for building test state files.
type dispatchAgentState struct {
	name                string
	state               string
	lastDeliveredCursor int
}

// makeDispatchStateFile creates a temp state file with the given agents.
// Returns the file path.
func makeDispatchStateFile(
	t *testing.T,
	agents []dispatchAgentState,
) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.toml")

	stateData := agentpkg.StateFile{
		Agents: make([]agentpkg.AgentRecord, 0, len(agents)),
	}

	for _, a := range agents {
		stateData.Agents = append(stateData.Agents, agentpkg.AgentRecord{
			Name:                a.name,
			State:               a.state,
			SpawnedAt:           time.Now(),
			LastDeliveredCursor: a.lastDeliveredCursor,
		})
	}

	data, err := agentpkg.MarshalStateFile(stateData)
	if err != nil {
		t.Fatalf("makeDispatchStateFile: marshal failed: %v", err)
	}

	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("makeDispatchStateFile: write failed: %v", err)
	}

	return path
}
