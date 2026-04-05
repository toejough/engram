# Phase 1 — Chat I/O: `engram chat post` + `engram chat watch`

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace fragile bash (shlock, heredoc TOML construction, fswatch/tail/grep) with two binary subcommands. `engram chat post` handles all chat file writes atomically. `engram chat watch` blocks on the next matching message using fsnotify (kernel-driven, no polling). Fixes #471.

**Spec:** `docs/superpowers/specs/engram-deterministic-coordination-design.md` §3.1, §4.1, §4.2, §7 (Phase 1)

**Codesign session:** planners 17 (arch), 18 (skill), 19 (agent E2E), 20 (user E2E) — 2026-04-05. All four perspectives converged unanimously before this plan was written.

**Tech Stack:** Go, BurntSushi/toml (already in go.mod), github.com/fsnotify/fsnotify (new dep)

---

## Codesign Decisions

These decisions were argued and resolved during planning. Do NOT revisit without reading the codesign thread.

| Decision | Resolved | Rationale |
|----------|----------|-----------|
| `--type` accepts comma-separated values | Yes | `--type ack` alone misses WAIT responses in ACK-wait subagent; `--type ack,wait` is required for correctness |
| `--timeout` on `engram chat watch` | Yes | Without it, watch blocks forever when a recipient is offline — regression from current skill |
| `engram chat cursor` subcommand | Yes | Removes last pre-intent bash gap (`wc -l < "$CHAT_FILE"`) from skill |
| `AppendFile` injection in FilePoster | Yes | Spec only specifies LockFile injection; actual append also needs injection for DI consistency |
| Subagent return format: JSON in Phase 1 | Yes | PR is atomic — no partial-update window. Pipe-delimited requires dead conversion code deleted in Phase 2. JSON is simpler. |
| Hard cutover (no dual-mode) | Yes | Old bash heredoc writing deleted. Skills call binary. Binary not built = caught by startup check. |
| Startup binary check in skill | Yes | Silent failure (unknown command: chat) is worst failure mode. Fast-fail with clear message is mandatory. |
| TOML format identity test | Yes | tail -f output must be byte-identical to current. Golden fixture test required. |

---

## Package Structure

```
internal/
  chat/         pure domain — no os.* calls, no I/O imports
    chat.go     Message struct, Poster + Watcher interfaces, types
    poster.go   FilePoster implementation
    watcher.go  FileWatcher implementation
    chat_test.go
    poster_test.go
    watcher_test.go
  watch/        fsnotify I/O adapter boundary
    watch.go    Watcher interface + FSNotifyWatcher implementation
    watch_test.go  (integration test — real temp files)
internal/cli/
  cli.go        add runChatPost, runChatWatch, runChatCursor + wiring
  targets.go    add targ.Group("chat", ...) with post/watch/cursor sub-targets
  cli_test.go   add tests for new subcommands
```

---

## Task 1: `internal/chat` — Types + Interfaces

**Files:**
- Create: `internal/chat/chat.go`
- Create: `internal/chat/chat_test.go`

- [ ] **Step 1: Write failing test for Message TOML round-trip**

```go
func TestMessage_TOMLRoundTrip(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    original := chat.Message{
        From:   "planner-17",
        To:     "all",
        Thread: "impl-phase1",
        Type:   "info",
        TS:     time.Date(2026, 4, 5, 7, 0, 0, 0, time.UTC),
        Text:   "hello\nworld",
    }

    var buf bytes.Buffer
    encErr := toml.NewEncoder(&buf).Encode(struct {
        Message []chat.Message `toml:"message"`
    }{Message: []chat.Message{original}})
    g.Expect(encErr).NotTo(HaveOccurred())
    if encErr != nil {
        return
    }

    var parsed struct {
        Message []chat.Message `toml:"message"`
    }
    decErr := toml.Unmarshal(buf.Bytes(), &parsed)
    g.Expect(decErr).NotTo(HaveOccurred())
    if decErr != nil {
        return
    }

    g.Expect(parsed.Message).To(HaveLen(1))
    g.Expect(parsed.Message[0]).To(Equal(original))
}
```

- [ ] **Step 2: Implement `internal/chat/chat.go`**

```go
// Package chat provides pure domain types and interfaces for the engram chat protocol.
// No os.* calls. All I/O is injected.
package chat

import (
    "context"
    "time"
)

// Message is a single chat protocol message.
type Message struct {
    From   string    `toml:"from"`
    To     string    `toml:"to"`
    Thread string    `toml:"thread"`
    Type   string    `toml:"type"`
    TS     time.Time `toml:"ts"`
    Text   string    `toml:"text"`
}

// Poster appends messages to the chat file atomically.
type Poster interface {
    Post(msg Message) (newCursor int, err error)
}

// Watcher blocks until a matching message arrives after cursor.
// msgTypes filters by message type; empty slice matches all types.
// agent matches messages where the To field contains agent or "all".
type Watcher interface {
    Watch(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error)
}

// LockFile creates an exclusive lock file compatible with bash shlock convention.
// Returns an unlock function to release the lock.
// Implemented via os.OpenFile(O_CREATE|O_EXCL) at the CLI wiring layer.
type LockFile func(name string) (unlock func() error, err error)
```

---

## Task 2: `internal/chat` — FilePoster

**Files:**
- Create: `internal/chat/poster.go`
- Create: `internal/chat/poster_test.go`

- [ ] **Step 3: Write failing unit tests for FilePoster**

```go
func TestFilePoster_Post_AppendsValidTOML(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    var written bytes.Buffer
    poster := &chat.FilePoster{
        FilePath: "/fake/chat.toml",
        Lock:     fakeLock,
        AppendFile: func(_ string, data []byte) error {
            written.Write(data)
            return nil
        },
        LineCount: func(_ string) (int, error) { return 42, nil },
    }

    cursor, err := poster.Post(chat.Message{
        From: "executor", To: "all", Thread: "test", Type: "info", Text: "hello",
    })
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    g.Expect(cursor).To(Equal(42))

    var parsed struct {
        Message []chat.Message `toml:"message"`
    }
    g.Expect(toml.Unmarshal(written.Bytes(), &parsed)).To(Succeed())
    g.Expect(parsed.Message).To(HaveLen(1))
    g.Expect(parsed.Message[0].From).To(Equal("executor"))
    g.Expect(parsed.Message[0].TS).NotTo(BeZero())
}

func TestFilePoster_Post_GeneratesFreshTS(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    before := time.Now().UTC()

    var written bytes.Buffer
    poster := &chat.FilePoster{
        FilePath:   "/fake/chat.toml",
        Lock:       fakeLock,
        AppendFile: func(_ string, data []byte) error { written.Write(data); return nil },
        LineCount:  func(_ string) (int, error) { return 1, nil },
    }

    _, err := poster.Post(chat.Message{From: "x", To: "all", Thread: "t", Type: "info", Text: "y"})
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    after := time.Now().UTC()

    var parsed struct{ Message []chat.Message `toml:"message"` }
    g.Expect(toml.Unmarshal(written.Bytes(), &parsed)).To(Succeed())
    ts := parsed.Message[0].TS
    g.Expect(ts.After(before) || ts.Equal(before)).To(BeTrue())
    g.Expect(ts.Before(after) || ts.Equal(after)).To(BeTrue())
}

func TestFilePoster_Post_TOMLGoldenFixture(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    fixedTS := time.Date(2026, 4, 5, 7, 0, 0, 0, time.UTC)
    var written bytes.Buffer
    poster := &chat.FilePoster{
        FilePath:   "/fake/chat.toml",
        Lock:       fakeLock,
        AppendFile: func(_ string, data []byte) error { written.Write(data); return nil },
        LineCount:  func(_ string) (int, error) { return 1, nil },
        NowFunc:    func() time.Time { return fixedTS }, // injected for deterministic test
    }

    _, err := poster.Post(chat.Message{
        From: "lead", To: "all", Thread: "lifecycle", Type: "info", Text: "hello",
    })
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    golden := "\n[[message]]\n" +
        "from = \"lead\"\n" +
        "to = \"all\"\n" +
        "thread = \"lifecycle\"\n" +
        "type = \"info\"\n" +
        "ts = 2026-04-05T07:00:00Z\n" +
        "text = \"\"\"\nhello\n\"\"\"\n"
    g.Expect(written.String()).To(Equal(golden))
}
```

- [ ] **Step 4: Implement `internal/chat/poster.go`**

```go
// FilePoster appends messages to the chat file atomically with locking.
// All I/O is injected — no os.* calls in this package.
type FilePoster struct {
    FilePath   string
    Lock       LockFile
    AppendFile func(path string, data []byte) error
    LineCount  func(path string) (int, error)
    NowFunc    func() time.Time // injectable for tests; defaults to time.Now().UTC()
}

func (p *FilePoster) Post(msg Message) (int, error) {
    msg.TS = p.now()

    var buf bytes.Buffer
    // format: leading newline, [[message]] header, then TOML fields
    // field order: from, to, thread, type, ts, text (matches spec)
    // ...encode into buf...

    unlock, lockErr := p.Lock(p.FilePath + ".lock")
    if lockErr != nil {
        return 0, fmt.Errorf("acquiring lock: %w", lockErr)
    }
    defer unlock() //nolint:errcheck

    if appendErr := p.AppendFile(p.FilePath, buf.Bytes()); appendErr != nil {
        return 0, fmt.Errorf("appending to chat file: %w", appendErr)
    }

    cursor, countErr := p.LineCount(p.FilePath)
    if countErr != nil {
        return 0, fmt.Errorf("counting lines: %w", countErr)
    }

    return cursor, nil
}

func (p *FilePoster) now() time.Time {
    if p.NowFunc != nil {
        return p.NowFunc()
    }
    return time.Now().UTC()
}
```

**CRITICAL:** TOML encoding must produce the exact field order `from, to, thread, type, ts, text` with a triple-quoted multiline `text` field. The golden fixture test enforces this. Do NOT use `toml.NewEncoder` with struct reflection — field order is not guaranteed. Write the TOML manually using `fmt.Fprintf` to match the exact format the tail pane currently shows.

---

## Task 3: `internal/chat` — FileWatcher

**Files:**
- Create: `internal/chat/watcher.go`
- Create: `internal/chat/watcher_test.go`

- [ ] **Step 5: Write failing unit tests for FileWatcher**

```go
func TestFileWatcher_Watch_ReturnsFirstMatchingMessage(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    // Synthetic chat file with 3 messages. Cursor at end of first message.
    // Only the 3rd message matches agent="myagent".
    content := buildChatTOML([]chat.Message{
        {From: "a", To: "other", Thread: "t", Type: "info", TS: now, Text: "x"},
        {From: "b", To: "myagent", Thread: "t", Type: "info", TS: now, Text: "first match"},
        {From: "c", To: "myagent", Thread: "t", Type: "ack",  TS: now, Text: "ack match"},
    })

    // cursor = line count after first message
    cursorAfterFirst := countLines(buildChatTOML([]chat.Message{content[0]}))

    watcher := &chat.FileWatcher{
        FilePath:  "/fake/chat.toml",
        FSWatcher: &fakeWatcher{},                      // returns immediately
        ReadFile:  func(_ string) ([]byte, error) { return []byte(content), nil },
    }

    msg, newCursor, err := watcher.Watch(context.Background(), "myagent", cursorAfterFirst, nil)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    g.Expect(msg.Text).To(Equal("first match"))
    g.Expect(msg.Type).To(Equal("info"))
    g.Expect(newCursor).To(BeNumerically(">", cursorAfterFirst))
}

func TestFileWatcher_Watch_FiltersByType(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    // Two messages for "myagent": first is "info", second is "ack".
    // Filter: ["ack"] — should return second message, skipping first.
    // ...test implementation...
}

func TestFileWatcher_Watch_AllInToField(t *testing.T) {
    t.Parallel()
    // "all" in To field matches any agent name.
    // ...test implementation...
}

func TestFileWatcher_Watch_CtxCancellation(t *testing.T) {
    t.Parallel()
    // Cancelling ctx while watch is blocked should return ctx.Err().
    // ...test implementation...
}
```

- [ ] **Step 6: Implement `internal/chat/watcher.go`**

Cursor-to-message-index strategy (spec §4.2):
1. `ReadFile(filePath)` → raw bytes
2. Count `[[message]]` occurrences up to line `cursor` in raw bytes → `startIdx`
3. `toml.Unmarshal(rawBytes)` → `[]Message`
4. Scan from `startIdx`; find first where `(To == agent || strings.Contains(To, agent) || To == "all") && typeMatches(msg.Type, msgTypes)`
5. Return message + new cursor = total line count

```go
// FileWatcher watches a chat file for messages matching agent and type filter.
// All I/O is injected.
type FileWatcher struct {
    FilePath  string
    FSWatcher watch.Watcher
    ReadFile  func(path string) ([]byte, error)
}

func (w *FileWatcher) Watch(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error) {
    for {
        data, readErr := w.ReadFile(w.FilePath)
        if readErr != nil {
            return Message{}, 0, fmt.Errorf("reading chat file: %w", readErr)
        }

        msg, newCursor, found := findMessage(data, agent, cursor, msgTypes)
        if found {
            return msg, newCursor, nil
        }

        if err := w.FSWatcher.WaitForChange(ctx, w.FilePath); err != nil {
            return Message{}, 0, err
        }
    }
}
```

---

## Task 4: `internal/watch` — FSNotifyWatcher

**Files:**
- Create: `internal/watch/watch.go`
- Create: `internal/watch/watch_test.go`

- [ ] **Step 7: Write failing integration test for FSNotifyWatcher**

```go
// Integration test — uses real temp files and fsnotify.
func TestFSNotifyWatcher_WaitForChange_ReturnsOnFileWrite(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    dir := t.TempDir()
    path := filepath.Join(dir, "chat.toml")
    g.Expect(os.WriteFile(path, []byte("initial"), 0o600)).To(Succeed())

    watcher := &watch.FSNotifyWatcher{}
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    errCh := make(chan error, 1)
    go func() {
        errCh <- watcher.WaitForChange(ctx, path)
    }()

    time.Sleep(10 * time.Millisecond) // let watcher register
    g.Expect(os.WriteFile(path, []byte("changed"), 0o600)).To(Succeed())

    err := <-errCh
    g.Expect(err).NotTo(HaveOccurred())
}

func TestFSNotifyWatcher_WaitForChange_ReturnsOnCtxCancel(t *testing.T) {
    t.Parallel()
    // Start watcher, cancel ctx, assert error is ctx.Err().
    // ...
}
```

- [ ] **Step 8: Implement `internal/watch/watch.go` + add fsnotify dependency**

```go
// Package watch provides a file change notification abstraction.
// FSNotifyWatcher is the I/O adapter boundary for fsnotify.
package watch

import (
    "context"

    "github.com/fsnotify/fsnotify"
)

// Watcher blocks until a file changes.
type Watcher interface {
    WaitForChange(ctx context.Context, path string) error
}

// FSNotifyWatcher uses fsnotify (kqueue on macOS, inotify on Linux). No CGO.
type FSNotifyWatcher struct{}

func (w *FSNotifyWatcher) WaitForChange(ctx context.Context, path string) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return fmt.Errorf("creating fsnotify watcher: %w", err)
    }
    defer watcher.Close()

    if err := watcher.Add(path); err != nil {
        return fmt.Errorf("watching file: %w", err)
    }

    select {
    case <-watcher.Events:
        return nil
    case err := <-watcher.Errors:
        return fmt.Errorf("fsnotify error: %w", err)
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

Run: `go get github.com/fsnotify/fsnotify` then `targ build` to verify compilation.

---

## Task 5: CLI Wiring — `runChatPost` + `runChatWatch` + `runChatCursor`

**Files:**
- Modify: `internal/cli/cli.go` — add runChatPost, runChatWatch, runChatCursor, chat case in Run()
- Modify: `internal/cli/targets.go` — add targ.Group("chat", ...)
- Modify: `internal/cli/cli_test.go` — add tests for new subcommands

- [ ] **Step 9: Write failing tests for CLI chat subcommands**

```go
func TestRun_ChatPost_WritesToFile(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

    dir := t.TempDir()
    chatFile := filepath.Join(dir, "chat.toml")

    var stdout bytes.Buffer
    err := cli.Run([]string{
        "engram", "chat", "post",
        "--chat-file", chatFile,     // test-only flag; CLI derives path in prod
        "--from", "tester",
        "--to", "all",
        "--thread", "smoke",
        "--type", "info",
        "--text", "hello",
    }, &stdout, io.Discard, nil)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }

    // stdout should be the new cursor (integer)
    cursor, parseErr := strconv.Atoi(strings.TrimSpace(stdout.String()))
    g.Expect(parseErr).NotTo(HaveOccurred())
    g.Expect(cursor).To(BeNumerically(">", 0))

    // chat file should have valid TOML with our message
    data, readErr := os.ReadFile(chatFile)
    g.Expect(readErr).NotTo(HaveOccurred())
    if readErr != nil { return }

    var parsed struct{ Message []chat.Message `toml:"message"` }
    g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
    g.Expect(parsed.Message).To(HaveLen(1))
    g.Expect(parsed.Message[0].From).To(Equal("tester"))
}

func TestRun_ChatWatch_OutputsJSON(t *testing.T) {
    t.Parallel()
    // Start watch in goroutine, post a matching message, verify JSON output.
    // ...
}

func TestRun_ChatCursor_OutputsLineCount(t *testing.T) {
    t.Parallel()
    // Create file with known content, call chat cursor, verify line count.
    // ...
}
```

**Note:** `--chat-file` is a test-only escape hatch to bypass path derivation. In production, path is derived from os.UserHomeDir + os.Getwd + ProjectSlugFromPath + DataDirFromHome.

- [ ] **Step 10: Implement CLI wiring in `cli.go`**

```go
// in Run(), add:
case "chat":
    if len(subArgs) < 1 {
        return fmt.Errorf("%w: chat requires a subcommand (post|watch|cursor)", errUsage)
    }
    switch subArgs[0] {
    case "post":
        return runChatPost(subArgs[1:], stdout)
    case "watch":
        return runChatWatch(subArgs[1:], stdout)
    case "cursor":
        return runChatCursor(subArgs[1:], stdout)
    default:
        return fmt.Errorf("%w: chat %s", errUnknownCommand, subArgs[0])
    }
```

```go
// ChatPostArgs / ChatWatchArgs / ChatCursorArgs in targets.go:
type ChatPostArgs struct {
    From     string `targ:"flag,name=from,desc=sender agent name"`
    To       string `targ:"flag,name=to,desc=comma-separated recipient names or all"`
    Thread   string `targ:"flag,name=thread,desc=conversation thread name"`
    MsgType  string `targ:"flag,name=type,desc=message type (intent|ack|wait|info|done|learned|ready|shutdown|escalate)"`
    Text     string `targ:"flag,name=text,desc=message content"`
    ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

type ChatWatchArgs struct {
    Agent    string `targ:"flag,name=agent,desc=agent name to filter messages for"`
    Cursor   int    `targ:"flag,name=cursor,desc=line number to start watching from"`
    Types    string `targ:"flag,name=type,desc=comma-separated message types to filter (empty=all)"`
    Timeout  int    `targ:"flag,name=timeout,desc=seconds before giving up (0=block forever)"`
    ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}

type ChatCursorArgs struct {
    ChatFile string `targ:"flag,name=chat-file,desc=override chat file path (testing only)"`
}
```

```go
// in Targets(), add alongside existing targets:
targ.Group("chat",
    targ.Targ(func(a ChatPostArgs) {
        args := append([]string{"engram", "chat", "post"}, ChatPostFlags(a)...)
        RunSafe(args, stdout, stderr, stdin)
    }).Name("post").Description("Post a message to the engram chat file"),
    targ.Targ(func(a ChatWatchArgs) {
        args := append([]string{"engram", "chat", "watch"}, ChatWatchFlags(a)...)
        RunSafe(args, stdout, stderr, stdin)
    }).Name("watch").Description("Block until a matching message arrives in the chat file"),
    targ.Targ(func(a ChatCursorArgs) {
        args := append([]string{"engram", "chat", "cursor"}, ChatCursorFlags(a)...)
        RunSafe(args, stdout, stderr, stdin)
    }).Name("cursor").Description("Output the current chat file line count (cursor position)"),
)
```

DI wiring in `runChatPost` / `runChatWatch`:
```go
func runChatPost(args []string, stdout io.Writer) error {
    // ... flag parsing ...

    chatFilePath, err := deriveChatFilePath(chatFileOverride)
    if err != nil { return err }

    lockFile := func(name string) (func() error, error) {
        f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec
        if err != nil { return nil, err }
        return func() error { f.Close(); return os.Remove(name) }, nil
    }

    poster := &chat.FilePoster{
        FilePath:   chatFilePath,
        Lock:       lockFile,
        AppendFile: func(path string, data []byte) error {
            f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec
            if err != nil { return err }
            defer f.Close()
            _, err = f.Write(data)
            return err
        },
        LineCount: func(path string) (int, error) {
            data, err := os.ReadFile(path) //nolint:gosec
            if err != nil { return 0, err }
            return bytes.Count(data, []byte("\n")), nil
        },
    }

    cursor, err := poster.Post(chat.Message{
        From: from, To: to, Thread: thread, Type: msgType, Text: text,
    })
    if err != nil { return fmt.Errorf("chat post: %w", err) }

    _, err = fmt.Fprintln(stdout, cursor)
    return err
}
```

---

## Task 6: Concurrent Write Safety Test

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 11: Write concurrent write safety test**

```go
func TestRun_ChatPost_ConcurrentWritesSafeToml(t *testing.T) {
    t.Parallel()

    g := NewGomegaWithT(t)

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
    if readErr != nil { return }

    var parsed struct{ Message []chat.Message `toml:"message"` }
    g.Expect(toml.Unmarshal(data, &parsed)).To(Succeed())
    g.Expect(parsed.Message).To(HaveLen(messageCount))
}
```

---

## Task 7: Skill Rewrites

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`
- Modify: `skills/engram-tmux-lead/SKILL.md`

**IMPORTANT:** Use `superpowers:writing-skills` skill for ALL skill edits. This enforces the TDD baseline→update→verify cycle.

- [ ] **Step 12: Rewrite use-engram-chat-as per skill plan**

**DELETED** (remove entirely):
- "Writing Messages" bash block (~25 lines: shlock while-loop, cat heredoc, lockfile removal, mkdir, all timestamp freshness BAD/GOOD examples)
- Timestamp Freshness section header + content (~20 lines)
- Heartbeat bash heredoc (~10 lines) → replace with `engram chat post --type info --thread heartbeat ...`
- User Input Parroting TOML heredoc (~10 lines) → replace with `engram chat post --type info ...`

**REWRITTEN:**
- Writing Messages section → single example: `CURSOR=$(engram chat post --from <name> --to <names> --thread <t> --type <t> --text "<content>")`
- Background Monitor Pattern (~30 lines → ~15 lines): replace fswatch/tail/grep with `engram chat watch --agent AGENT_NAME --cursor CURSOR`; subagent returns JSON pass-through
- ACK-wait subagent template (~35 lines → ~12 lines): `RESULT=$(engram chat watch --agent AGENT_NAME --cursor CURSOR --type ack,wait --timeout 30)`

**ADDED:**
- Startup binary check (before Agent Lifecycle step 1):
  ```bash
  if ! engram chat post --help >/dev/null 2>&1; then
    echo "ERROR: engram binary missing 'chat' subcommand. Run: targ build"; exit 1
  fi
  ```
- Cursor tracking note: "Initial cursor = the integer returned by your ready post. No separate wc -l needed."
- `engram chat cursor` note for pre-intent cursor: `CURSOR=$(engram chat cursor)`
- JSON parsing example for subagent results:
  ```bash
  TYPE=$(echo "$RESULT" | jq -r '.type')
  FROM=$(echo "$RESULT" | jq -r '.from')
  CURSOR=$(echo "$RESULT" | jq -r '.cursor')
  TEXT=$(echo "$RESULT" | jq -r '.text')
  # Background Agent subagents can parse JSON natively without jq.
  ```
- Phase-transition note in Background Monitor Pattern: "This subagent survives Phase 1. Phase 2 (`engram chat ack-wait`) will eliminate it entirely."

**UNCHANGED:**
- Online/offline detection bash (deleted in Phase 2)
- Compaction recovery section (updated in Phase 4)
- Intent Protocol flow text
- Agent Lifecycle step structure (mechanism improves, steps same)
- Chat File Location section (bash derivation retained as observability reference, no longer mandatory for write path)

- [ ] **Step 13: Rewrite engram-tmux-lead shlock/heredoc posting sites**

Replace each `shlock + cat >> $CHAT_FILE << EOF ... EOF + rm -f $CHAT_FILE.lock` block (~5 call sites) with:
```bash
engram chat post --from lead --to <names> --thread <thread> --type <type> --text "<content>"
```

---

## Task 8: targ.go Additions (if needed)

- [ ] **Step 14: Add targ check target for new packages**

If `targets.go` (build system) needs updating to include `internal/chat` and `internal/watch` in test coverage, add them. Run `targ check-full` to verify all linters pass on new code.

---

## Verification Protocol

These are the explicit steps to run before declaring Phase 1 done. Run them in order.

**Step 1 — Binary smoke test:**
```bash
targ build
engram chat post --from test --to all --thread smoke --type info --text "phase1 smoke test"
tail -5 ~/.local/share/engram/chat/$(pwd | tr '/' '-').toml
```
Verify: `[[message]]` block with correct fields, fresh `ts` (within 1 second of now), no formatting artifacts.

**Step 2 — Watch smoke test (two terminals):**
```bash
# Terminal 1:
CURSOR=$(engram chat cursor)
engram chat watch --agent test --cursor $CURSOR
# Terminal 2:
engram chat post --from x --to test --thread smoke --type info --text "ping"
```
Verify: Terminal 1 prints JSON and exits immediately (under 500ms, not after timeout).

**Step 3 — Timestamp freshness:**
Run a session. Check that `ts` values in `tail -f` output are within 1 second of actual post time. (Tests the stale-timestamp bug fix.)

**Step 4 — Concurrent write safety:**
`targ test` — the concurrent write test catches any locking regression. Verify TOML file is valid after 10 concurrent writes.

---

## Done Criteria

- [ ] `targ check-full` passes (all linters, all tests, coverage thresholds)
- [ ] `engram chat post` appends valid TOML with correct field order
- [ ] `engram chat watch` exits with JSON when matching message arrives
- [ ] `engram chat watch --timeout 5` exits with error after 5s with no matching message
- [ ] `engram chat watch --type ack,wait` matches both ack and wait, skips info/intent
- [ ] `engram chat cursor` outputs current line count
- [ ] Concurrent write safety test passes (10 agents, valid TOML after)
- [ ] TOML golden fixture test passes (byte-identical field order)
- [ ] Startup binary check present in use-engram-chat-as skill
- [ ] All shlock/heredoc bash removed from use-engram-chat-as and engram-tmux-lead
- [ ] Skill still E2E-functional: agent lifecycle, intent protocol, ACK-wait subagent all work
- [ ] Phase-transition note present in Background Monitor Pattern section
