package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"engram/internal/apiclient"
)

// Exported constants.
const (
	DefaultAPIAddr = "http://localhost:7932"
)

// Exported types.

// ServerStarter is the interface for starting the engram API server subprocess.
type ServerStarter interface {
	Start(ctx context.Context, apiAddr string) error
}

// NewOSServerStarter returns a ServerStarter that launches the engram binary as a subprocess.
func NewOSServerStarter() ServerStarter {
	return &osServerStarter{}
}

// Exported functions.

// EnsureServerRunning checks whether the API server is reachable. If not, it
// starts the server via the provided starter and polls until ready (up to startupTimeout).
// Use NewOSServerStarter() for production.
func EnsureServerRunning(
	ctx context.Context,
	apiClient apiclient.API,
	apiAddr string,
	starter ServerStarter,
) error {
	return ensureServerRunning(ctx, apiClient, apiAddr, starter)
}

// Run starts the MCP server over stdio, resolving the API address from
// the ENGRAM_API_ADDR environment variable (falls back to DefaultAPIAddr).
func Run(ctx context.Context) error {
	apiAddr := os.Getenv("ENGRAM_API_ADDR")
	if apiAddr == "" {
		apiAddr = DefaultAPIAddr
	}

	return runWithDeps(ctx, apiAddr, http.DefaultClient, NewOSServerStarter(), &mcp.StdioTransport{})
}

// unexported constants.
const (
	startupPollDelay = 500 * time.Millisecond
	startupTimeout   = 15 * time.Second
)

// unexported variables.
var (
	errServerNotReady = errors.New("engram API server did not become ready within 15 seconds")
)

// unexported types.

// osServerStarter starts the engram server as a real OS subprocess.
type osServerStarter struct{}

// Start implements ServerStarter.
func (s *osServerStarter) Start(ctx context.Context, apiAddr string) error {
	cmd := osStarterCmd(ctx, "engram", apiAddr)

	startErr := cmd.Start()
	if startErr != nil {
		return fmt.Errorf("starting engram server: %w", startErr)
	}

	return nil
}

// unexported functions.

// ensureServerRunning checks whether the API server is reachable. If not, it
// uses the starter to start the server and polls until ready (up to startupTimeout).
func ensureServerRunning(
	ctx context.Context,
	apiClient apiclient.API,
	apiAddr string,
	starter ServerStarter,
) error {
	_, statusErr := apiClient.Status(ctx)
	if statusErr == nil {
		return nil
	}

	startErr := starter.Start(ctx, apiAddr)
	if startErr != nil {
		return startErr
	}

	deadline := time.Now().Add(startupTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for server startup: %w", ctx.Err())
		case <-time.After(startupPollDelay):
		}

		_, pollErr := apiClient.Status(ctx)
		if pollErr == nil {
			return nil
		}
	}

	return errServerNotReady
}

// osStarterCmd builds the exec.Cmd for starting the engram server binary.
// binary is the command name (typically "engram").
func osStarterCmd(ctx context.Context, binary, apiAddr string) *exec.Cmd {
	//nolint:gosec // binary and apiAddr are controlled by the caller.
	cmd := exec.CommandContext(ctx, binary, "server", "up", "--addr", apiAddr)
	cmd.Stdout = os.Stderr // route server output to stderr so it doesn't pollute stdio
	cmd.Stderr = os.Stderr

	return cmd
}

// runWithDeps creates the MCP server with injectable dependencies, ensures the API server
// is running, starts the subscribe loop, and runs the MCP server over the given transport.
func runWithDeps(
	ctx context.Context,
	apiAddr string,
	httpClient apiclient.HTTPDoer,
	starter ServerStarter,
	transport mcp.Transport,
) error {
	apiClient := apiclient.New(apiAddr, httpClient)

	ensureErr := EnsureServerRunning(ctx, apiClient, apiAddr, starter)
	if ensureErr != nil {
		return fmt.Errorf("mcpserver: API server not available: %w", ensureErr)
	}

	logger := slog.Default()
	agentCapture := NewAgentNameCapture()
	server := New(apiClient, agentCapture)

	go RunSubscribeLoop(ctx, apiClient, server, agentCapture, logger)

	runErr := server.Run(ctx, transport)
	if runErr != nil {
		return fmt.Errorf("mcpserver: running: %w", runErr)
	}

	return nil
}
