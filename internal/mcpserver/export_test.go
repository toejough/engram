package mcpserver

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"engram/internal/apiclient"
)

// IntentArgs exports intentArgs for use in tests.
type IntentArgs = intentArgs

// LearnArgs exports learnArgs for use in tests.
type LearnArgs = learnArgs

// PostArgs exports postArgs for use in tests.
type PostArgs = postArgs

// Exported functions.

// ExportHandleIntent exports the handleIntent constructor for testing.
func ExportHandleIntent(
	api apiclient.API,
) func(context.Context, *mcp.CallToolRequest, intentArgs) (*mcp.CallToolResult, any, error) {
	return handleIntent(api, noopCapture)
}

// ExportHandleLearn exports the handleLearn constructor for testing.
func ExportHandleLearn(
	api apiclient.API,
) func(context.Context, *mcp.CallToolRequest, learnArgs) (*mcp.CallToolResult, any, error) {
	return handleLearn(api, noopCapture)
}

// ExportHandlePost exports the handlePost constructor for testing.
func ExportHandlePost(
	api apiclient.API,
) func(context.Context, *mcp.CallToolRequest, postArgs) (*mcp.CallToolResult, any, error) {
	return handlePost(api, noopCapture)
}

// ExportHandleStatus exports the handleStatus constructor for testing.
func ExportHandleStatus(
	api apiclient.API,
) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return handleStatus(api)
}

// ExportNewOSServerStarter exports NewOSServerStarter for testing.
func ExportNewOSServerStarter() ServerStarter {
	return NewOSServerStarter()
}

// ExportOSStarterStart exports the osServerStarter.Start method for testing.
func ExportOSStarterStart(ctx context.Context, apiAddr string) error {
	starter := &osServerStarter{}

	return starter.Start(ctx, apiAddr)
}

// ExportOSStarterStartBinary starts a subprocess using the given binary name.
func ExportOSStarterStartBinary(ctx context.Context, binary, apiAddr string) error {
	cmd := osStarterCmd(ctx, binary, apiAddr)

	return cmd.Start()
}

// ExportRunWithDeps exports runWithDeps for testing.
func ExportRunWithDeps(
	ctx context.Context,
	apiAddr string,
	httpClient *http.Client,
	starter ServerStarter,
	transport mcp.Transport,
) error {
	return runWithDeps(ctx, apiAddr, httpClient, starter, transport)
}

// unexported variables.
var (
	noopCapture = NewAgentNameCapture()
)
