package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"engram/internal/apiclient"
)

// unexported constants.
const (
	defaultAPIAddr = "http://localhost:7932"
	intentCmd      = "intent"
	postCmd        = "post"
)

// unexported variables.
var (
	errFromRequired       = errors.New("post: --from is required")
	errIntentFromRequired = errors.New("intent: --from is required")
	errIntentToRequired   = errors.New("intent: --to is required")
	errTextRequired       = errors.New("post: --text is required")
	errToRequired         = errors.New("post: --to is required")
)

// doIntent posts an intent message and waits for the engram agent's response.
// Pure function — no I/O construction. Accepts API interface.
func doIntent(
	ctx context.Context,
	api apiclient.API,
	from, toAgent, situation, plannedAction string,
	stdout io.Writer,
) error {
	text := "situation: " + situation + "\nplanned-action: " + plannedAction

	postResp, postErr := api.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: toAgent, Text: text,
	})
	if postErr != nil {
		return fmt.Errorf("intent: posting: %w", postErr)
	}

	waitResp, waitErr := api.WaitForResponse(ctx, apiclient.WaitRequest{
		From: toAgent, To: from, AfterCursor: postResp.Cursor,
	})
	if waitErr != nil {
		return fmt.Errorf("intent: waiting: %w", waitErr)
	}

	_, printErr := fmt.Fprintln(stdout, waitResp.Text)
	if printErr != nil {
		return fmt.Errorf("intent: writing response: %w", printErr)
	}

	return nil
}

// doPost posts a message via the API and prints the cursor.
// Pure function — no I/O construction. Accepts API interface.
func doPost(
	ctx context.Context,
	api apiclient.API,
	from, to, text string,
	stdout io.Writer,
) error {
	resp, err := api.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: to, Text: text,
	})
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)
	if printErr != nil {
		return fmt.Errorf("post: writing cursor: %w", printErr)
	}

	return nil
}

// runAPIDispatch dispatches API subcommands.
func runAPIDispatch(ctx context.Context, cmd string, args []string, stdout io.Writer) error {
	switch cmd {
	case intentCmd:
		return runIntent(ctx, args, stdout)
	case postCmd:
		return runPost(ctx, args, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// runIntent is the thin wiring layer: parses flags, constructs real client, calls doIntent.
func runIntent(ctx context.Context, args []string, stdout io.Writer) error {
	fs := newFlagSet(intentCmd)

	var from, toAgent, situation, plannedAction, addr string

	fs.StringVar(&from, "from", "", "sender agent name")
	fs.StringVar(&toAgent, "to", "", "recipient agent name")
	fs.StringVar(&situation, "situation", "", "current situation description")
	fs.StringVar(&plannedAction, "planned-action", "", "planned action description")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("intent: %w", parseErr)
	}

	if from == "" {
		return errIntentFromRequired
	}

	if toAgent == "" {
		return errIntentToRequired
	}

	client := apiclient.New(addr, http.DefaultClient)

	return doIntent(ctx, client, from, toAgent, situation, plannedAction, stdout)
}

// runPost is the thin wiring layer: parses flags, constructs real client, calls doPost.
func runPost(ctx context.Context, args []string, stdout io.Writer) error {
	fs := newFlagSet(postCmd)

	var from, toAgent, text, addr string

	fs.StringVar(&from, "from", "", "sender agent name")
	fs.StringVar(&toAgent, "to", "", "recipient agent name")
	fs.StringVar(&text, "text", "", "message content")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("post: %w", parseErr)
	}

	if from == "" {
		return errFromRequired
	}

	if toAgent == "" {
		return errToRequired
	}

	if text == "" {
		return errTextRequired
	}

	client := apiclient.New(addr, http.DefaultClient)

	return doPost(ctx, client, from, toAgent, text, stdout)
}
