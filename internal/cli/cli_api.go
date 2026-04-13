package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"engram/internal/apiclient"
)

// unexported constants.
const (
	defaultAPIAddr  = "http://localhost:7932"
	engramAgentName = "engram-agent"
	intentCmd       = "intent"
	learnCmd        = "learn"
	postCmd         = "post"
	statusCmd       = "status"
	subscribeCmd    = "subscribe"
)

// unexported variables.
var (
	errFromRequired           = errors.New("post: --from is required")
	errIntentFromRequired     = errors.New("intent: --from is required")
	errIntentToRequired       = errors.New("intent: --to is required")
	errLearnFromRequired      = errors.New("learn: --from is required")
	errLearnInvalidType       = errors.New("learn: --type must be 'feedback' or 'fact'")
	errSubscribeAgentRequired = errors.New("subscribe: --agent is required")
	errTextRequired           = errors.New("post: --text is required")
	errToRequired             = errors.New("post: --to is required")
)

// buildLearnText constructs JSON text for a learn message.
// learnType must be "feedback" or "fact"; returns errLearnInvalidType otherwise.
func buildLearnText(
	learnType, situation, behavior, impact, action, subject, predicate, object string,
) (string, error) {
	var payload map[string]string

	switch learnType {
	case "feedback":
		payload = map[string]string{
			"type":      learnType,
			"situation": situation,
			"behavior":  behavior,
			"impact":    impact,
			"action":    action,
		}
	case "fact":
		payload = map[string]string{
			"type":      learnType,
			"situation": situation,
			"subject":   subject,
			"predicate": predicate,
			"object":    object,
		}
	default:
		return "", fmt.Errorf("%w, got %q", errLearnInvalidType, learnType)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("learn: marshalling text: %w", err)
	}

	return string(data), nil
}

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

// doLearn builds a learn message and posts it to the engram agent.
// Pure function -- no I/O construction. Accepts API interface.
func doLearn(
	ctx context.Context,
	api apiclient.API,
	from, learnType, situation, behavior, impact, action, subject, predicate, object string,
	stdout io.Writer,
) error {
	text, buildErr := buildLearnText(
		learnType, situation, behavior, impact, action, subject, predicate, object,
	)
	if buildErr != nil {
		return buildErr
	}

	resp, postErr := api.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: engramAgentName, Text: text,
	})
	if postErr != nil {
		return fmt.Errorf("learn: posting: %w", postErr)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)
	if printErr != nil {
		return fmt.Errorf("learn: writing cursor: %w", printErr)
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

// doStatus queries the API for server status and prints it as indented JSON.
// Pure function — no I/O construction. Accepts API interface.
func doStatus(
	ctx context.Context,
	api apiclient.API,
	stdout io.Writer,
) error {
	resp, err := api.Status(ctx)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	data, marshalErr := json.MarshalIndent(resp, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("status: encoding: %w", marshalErr)
	}

	_, printErr := fmt.Fprintln(stdout, string(data))
	if printErr != nil {
		return fmt.Errorf("status: writing: %w", printErr)
	}

	return nil
}

// doSubscribe polls the API for new messages in a loop until the context
// is cancelled or an error occurs. Pure function — no I/O construction.
func doSubscribe(
	ctx context.Context,
	api apiclient.API,
	agent string,
	afterCursor int,
	stdout io.Writer,
) error {
	cursor := afterCursor

	for {
		resp, err := api.Subscribe(ctx, apiclient.SubscribeRequest{
			Agent:       agent,
			AfterCursor: cursor,
		})
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}

		for _, msg := range resp.Messages {
			_, printErr := fmt.Fprintf(
				stdout, "[%s -> %s] %s\n", msg.From, msg.To, msg.Text,
			)
			if printErr != nil {
				return fmt.Errorf("subscribe: writing: %w", printErr)
			}
		}

		cursor = resp.Cursor
	}
}

// runAPIDispatch dispatches API subcommands.
func runAPIDispatch(ctx context.Context, cmd string, args []string, stdout io.Writer) error {
	switch cmd {
	case intentCmd:
		return runIntent(ctx, args, stdout)
	case learnCmd:
		return runLearn(ctx, args, stdout)
	case postCmd:
		return runPost(ctx, args, stdout)
	case statusCmd:
		return runStatus(ctx, args, stdout)
	case subscribeCmd:
		return runSubscribe(ctx, args, stdout)
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

// runLearn is the thin wiring layer: parses flags, constructs real client, calls doLearn.
func runLearn(ctx context.Context, args []string, stdout io.Writer) error {
	fs := newFlagSet(learnCmd)

	var (
		from, learnType, situation, behavior, impact, action string
		subject, predicate, object, addr                     string
	)

	fs.StringVar(&from, "from", "", "sender agent name")
	fs.StringVar(&learnType, "type", "", "learn type: feedback or fact")
	fs.StringVar(&situation, "situation", "", "situation description")
	fs.StringVar(&behavior, "behavior", "", "observed behavior (feedback)")
	fs.StringVar(&impact, "impact", "", "impact description (feedback)")
	fs.StringVar(&action, "action", "", "corrective action (feedback)")
	fs.StringVar(&subject, "subject", "", "fact subject")
	fs.StringVar(&predicate, "predicate", "", "fact predicate")
	fs.StringVar(&object, "object", "", "fact object")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("learn: %w", parseErr)
	}

	if from == "" {
		return errLearnFromRequired
	}

	client := apiclient.New(addr, http.DefaultClient)

	return doLearn(
		ctx, client, from, learnType, situation, behavior, impact,
		action, subject, predicate, object, stdout,
	)
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

// runStatus is the thin wiring layer: parses flags, constructs real client, calls doStatus.
func runStatus(ctx context.Context, args []string, stdout io.Writer) error {
	fs := newFlagSet(statusCmd)

	var addr string

	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("status: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)

	return doStatus(ctx, client, stdout)
}

// runSubscribe is the thin wiring layer: parses flags, constructs real client, calls doSubscribe.
func runSubscribe(ctx context.Context, args []string, stdout io.Writer) error {
	fs := newFlagSet(subscribeCmd)

	var (
		agent       string
		afterCursor int
		addr        string
	)

	fs.StringVar(&agent, "agent", "", "agent name to subscribe as")
	fs.IntVar(&afterCursor, "after-cursor", 0, "cursor position to start from")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("subscribe: %w", parseErr)
	}

	if agent == "" {
		return errSubscribeAgentRequired
	}

	client := apiclient.New(addr, http.DefaultClient)

	return doSubscribe(ctx, client, agent, afterCursor, stdout)
}
