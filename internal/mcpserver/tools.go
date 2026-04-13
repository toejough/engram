// Package mcpserver sets up the MCP server and its tool handlers.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"engram/internal/apiclient"
)

// unexported constants.
const (
	engramAgentName = "engram-agent"
)

// unexported variables.
var (
	errLearnInvalidType = errors.New("engram_learn: --type must be 'feedback' or 'fact'")
)

// intentArgs are the parameters for the engram_intent tool.
type intentArgs struct {
	From          string `json:"from"          jsonschema:"sender agent name"`
	To            string `json:"to"            jsonschema:"recipient agent name"`
	Situation     string `json:"situation"     jsonschema:"description of the current situation"`
	PlannedAction string `json:"plannedAction" jsonschema:"description of the intended action"`
}

// learnArgs are the parameters for the engram_learn tool.
type learnArgs struct {
	From      string `json:"from"      jsonschema:"sender agent name"`
	Type      string `json:"type"      jsonschema:"learn type: feedback or fact"`
	Situation string `json:"situation" jsonschema:"current situation"`
	Behavior  string `json:"behavior"  jsonschema:"observed behavior (feedback only)"`
	Impact    string `json:"impact"    jsonschema:"impact description (feedback only)"`
	Action    string `json:"action"    jsonschema:"corrective action (feedback only)"`
	Subject   string `json:"subject"   jsonschema:"fact subject (fact only)"`
	Predicate string `json:"predicate" jsonschema:"fact predicate (fact only)"`
	Object    string `json:"object"    jsonschema:"fact object (fact only)"`
}

// postArgs are the parameters for the engram_post tool.
type postArgs struct {
	From string `json:"from" jsonschema:"sender agent name"`
	To   string `json:"to"   jsonschema:"recipient agent name"`
	Text string `json:"text" jsonschema:"message content"`
}

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

	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "", fmt.Errorf("engram_learn: marshalling text: %w", marshalErr)
	}

	return string(data), nil
}

// handleIntent implements the engram_intent tool: posts the intent then waits for
// surfaced memories from the engram-agent.
func handleIntent(
	apiClient apiclient.API,
	agentCapture *AgentNameCapture,
) func(context.Context, *mcp.CallToolRequest, intentArgs) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context, _ *mcp.CallToolRequest, args intentArgs,
	) (*mcp.CallToolResult, any, error) {
		agentCapture.Set(args.From)

		text := "situation: " + args.Situation + "\nplanned-action: " + args.PlannedAction

		postResp, postErr := apiClient.PostMessage(ctx, apiclient.PostMessageRequest{
			From: args.From,
			To:   args.To,
			Text: text,
		})
		if postErr != nil {
			return nil, nil, fmt.Errorf("engram_intent: posting: %w", postErr)
		}

		waitResp, waitErr := apiClient.WaitForResponse(ctx, apiclient.WaitRequest{
			From:        args.To,
			To:          args.From,
			AfterCursor: postResp.Cursor,
		})
		if waitErr != nil {
			return nil, nil, fmt.Errorf("engram_intent: waiting: %w", waitErr)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: waitResp.Text},
			},
		}, nil, nil
	}
}

// handleLearn implements the engram_learn tool: validates and posts a learn message.
func handleLearn(
	apiClient apiclient.API,
	agentCapture *AgentNameCapture,
) func(context.Context, *mcp.CallToolRequest, learnArgs) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context, _ *mcp.CallToolRequest, args learnArgs,
	) (*mcp.CallToolResult, any, error) {
		agentCapture.Set(args.From)

		text, buildErr := buildLearnText(
			args.Type, args.Situation, args.Behavior, args.Impact, args.Action,
			args.Subject, args.Predicate, args.Object,
		)
		if buildErr != nil {
			// Return as a tool error (not a protocol error). The SDK sets IsError: true.
			return nil, nil, buildErr
		}

		resp, postErr := apiClient.PostMessage(ctx, apiclient.PostMessageRequest{
			From: args.From,
			To:   engramAgentName,
			Text: text,
		})
		if postErr != nil {
			return nil, nil, fmt.Errorf("engram_learn: posting: %w", postErr)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("cursor: %d", resp.Cursor)},
			},
		}, nil, nil
	}
}

// handlePost implements the engram_post tool: posts a message and returns the cursor.
func handlePost(
	apiClient apiclient.API,
	agentCapture *AgentNameCapture,
) func(context.Context, *mcp.CallToolRequest, postArgs) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context, _ *mcp.CallToolRequest, args postArgs,
	) (*mcp.CallToolResult, any, error) {
		agentCapture.Set(args.From)

		resp, err := apiClient.PostMessage(ctx, apiclient.PostMessageRequest{
			From: args.From,
			To:   args.To,
			Text: args.Text,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("engram_post: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("cursor: %d", resp.Cursor)},
			},
		}, nil, nil
	}
}

// handleStatus implements the engram_status tool: returns server status as JSON.
func handleStatus(
	apiClient apiclient.API,
) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context, _ *mcp.CallToolRequest, _ struct{},
	) (*mcp.CallToolResult, any, error) {
		resp, err := apiClient.Status(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("engram_status: %w", err)
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return nil, nil, fmt.Errorf("engram_status: encoding: %w", marshalErr)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(data)},
			},
		}, nil, nil
	}
}
