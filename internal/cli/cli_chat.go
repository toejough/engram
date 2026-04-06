package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"engram/internal/chat"
)

// watchResult is the JSON-serializable output of engram chat watch.
type watchResult struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Thread string    `json:"thread"`
	Type   string    `json:"type"`
	TS     time.Time `json:"ts"`
	Text   string    `json:"text"`
	Cursor int       `json:"cursor"`
}

// marshalAndWriteWatchResult encodes result as JSON and writes it to stdout.
func marshalAndWriteWatchResult(stdout io.Writer, result watchResult) error {
	encoded, encErr := json.Marshal(result)
	if encErr != nil {
		return fmt.Errorf("chat watch: encoding result: %w", encErr)
	}

	_, err := fmt.Fprintln(stdout, string(encoded))
	if err != nil {
		return fmt.Errorf("chat watch: writing output: %w", err)
	}

	return nil
}

func runChatAckWait(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat ack-wait")

	agent := fs.String("agent", "", "calling agent name")
	cursor := fs.Int("cursor", 0, "line position to start watching from")
	recips := fs.String("recipients", "", "comma-separated recipient names")
	maxWaitS := fs.Int("max-wait", 0, "seconds to wait for online-silent recipients (default 30)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat ack-wait: %w", parseErr)
	}

	if *agent == "" {
		return errAgentRequired
	}

	if *recips == "" {
		return errRecipientsRequired
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat ack-wait", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	waiter := &chat.FileAckWaiter{
		FilePath: chatFilePath,
		Watcher:  newFileWatcher(chatFilePath),
		ReadFile: os.ReadFile,
		NowFunc:  time.Now,
		MaxWait:  time.Duration(*maxWaitS) * time.Second,
	}

	ctx, cancel := signalContext()
	defer cancel()

	result, ackErr := waiter.AckWait(ctx, *agent, *cursor, strings.Split(*recips, ","))
	if ackErr != nil {
		return fmt.Errorf("chat ack-wait: %w", ackErr)
	}

	return outputAckResult(stdout, result)
}

func runChatCursor(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat cursor")

	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat cursor: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat cursor", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	count, countErr := osLineCount(chatFilePath)
	if countErr != nil {
		return fmt.Errorf("chat cursor: %w", countErr)
	}

	_, err := fmt.Fprintln(stdout, count)
	if err != nil {
		return fmt.Errorf("chat cursor: writing output: %w", err)
	}

	return nil
}

// runChatDispatch routes chat subcommands (post|watch|cursor).
func runChatDispatch(subArgs []string, stdout io.Writer) error {
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
	case "ack-wait":
		return runChatAckWait(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: chat %s", errUnknownCommand, subArgs[0])
	}
}

func runChatPost(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat post")

	from := fs.String("from", "", "sender agent name")
	toField := fs.String("to", "", "recipient names or all")
	thread := fs.String("thread", "", "conversation thread name")
	msgType := fs.String("type", "", "message type")
	text := fs.String("text", "", "message content")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat post: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat post", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	poster := newFilePoster(chatFilePath)

	cursor, postErr := poster.Post(chat.Message{
		From:   *from,
		To:     *toField,
		Thread: *thread,
		Type:   *msgType,
		Text:   *text,
	})
	if postErr != nil {
		return fmt.Errorf("chat post: %w", postErr)
	}

	_, err := fmt.Fprintln(stdout, cursor)
	if err != nil {
		return fmt.Errorf("chat post: writing output: %w", err)
	}

	return nil
}

func runChatWatch(args []string, stdout io.Writer) error {
	fs := newFlagSet("chat watch")

	agent := fs.String("agent", "", "agent name to filter messages for")
	cursor := fs.Int("cursor", 0, "line number to start watching from")
	typesStr := fs.String("type", "", "comma-separated message types to filter")
	timeoutSec := fs.Int("max-wait", 0, "seconds before giving up (0=block forever)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("chat watch: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "chat watch", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	var msgTypes []string

	if *typesStr != "" {
		msgTypes = strings.Split(*typesStr, ",")
	}

	ctx, cancel := signalContext()
	defer cancel()

	if *timeoutSec > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutSec)*time.Second)
		defer cancel()
	}

	watcher := newFileWatcher(chatFilePath)

	msg, newCursor, watchErr := watcher.Watch(ctx, *agent, *cursor, msgTypes)
	if watchErr != nil {
		return fmt.Errorf("chat watch: %w", watchErr)
	}

	result := watchResult{
		From:   msg.From,
		To:     msg.To,
		Thread: msg.Thread,
		Type:   msg.Type,
		TS:     msg.TS,
		Text:   msg.Text,
		Cursor: newCursor,
	}

	return marshalAndWriteWatchResult(stdout, result)
}
