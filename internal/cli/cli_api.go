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
	postCmd        = "post"
)

// unexported errors.
var (
	errFromRequired = errors.New("post: --from is required")
	errToRequired   = errors.New("post: --to is required")
	errTextRequired = errors.New("post: --text is required")
)

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
	case postCmd:
		return runPost(ctx, args, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
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
