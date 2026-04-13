package mcpserver

import (
	"context"
	"log/slog"
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

// ExportMCPNotificationSender returns the production mcpNotificationSender as a NotificationSender.
// Used in tests to exercise the real SendLog implementation with a live ServerSession.
func ExportMCPNotificationSender() NotificationSender {
	return mcpNotificationSender{}
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
// Used in tests to verify behavior when a specific binary is absent.
func ExportOSStarterStartBinary(ctx context.Context, binary, apiAddr string) error {
	cmd := osStarterCmd(ctx, binary, apiAddr)

	return cmd.Start()
}

// ExportRunSubscribeLoop exports runSubscribeLoop for testing.
func ExportRunSubscribeLoop(
	ctx context.Context,
	apiClient apiclient.API,
	sessions SessionProvider,
	sender NotificationSender,
	agentName string,
) {
	runSubscribeLoop(ctx, apiClient, sessions, sender, agentName, slog.Default())
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
