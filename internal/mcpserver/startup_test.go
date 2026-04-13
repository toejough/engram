package mcpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"engram/internal/apiclient"
	"engram/internal/mcpserver"
)

func TestEnsureServerRunning_WhenAlreadyRunning_ReturnsNilWithoutStarting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	starter := &fakeStarter{startErr: errors.New("should not be called")}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		data, _ := json.Marshal(apiclient.StatusResponse{Running: true})
		_, _ = writer.Write(data)
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)

	err := mcpserver.EnsureServerRunning(context.Background(), apiClient, server.URL, starter)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestEnsureServerRunning_WhenNotRunning_StartsAndPolls(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		callCount++

		if callCount >= 2 {
			// Second call returns running status (simulating server startup).
			data, _ := json.Marshal(apiclient.StatusResponse{Running: true})
			_, _ = writer.Write(data)

			return
		}

		// First call returns internal error (server not yet up).
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)
	starter := &fakeStarter{startErr: nil}

	err := mcpserver.EnsureServerRunning(context.Background(), apiClient, server.URL, starter)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestEnsureServerRunning_WhenStarterFails_ReturnsError(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		startErr := errors.New("start failed")

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			// Always fail status (simulate server not running).
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(`{}`))
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)
		starter := &fakeStarter{startErr: startErr}

		err := mcpserver.EnsureServerRunning(rt.Context(), apiClient, server.URL, starter)

		g.Expect(err).To(MatchError(ContainSubstring("start failed")))
	})
}

func TestNewOSServerStarter_ReturnsNonNilServerStarter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	starter := mcpserver.NewOSServerStarter()

	g.Expect(starter).NotTo(BeNil())
}

func TestOSStarterStart_WhenBinaryExists_StartsProcess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// t.Context() is cancelled when the test ends, which kills the spawned process.
	err := mcpserver.ExportOSStarterStart(t.Context(), "http://localhost:19999")

	// Engram is on PATH so Start() should succeed (process started, not waited).
	g.Expect(err).NotTo(HaveOccurred())
}

func TestOSStarterStart_WhenBinaryMissing_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Use a binary name that is guaranteed to not exist.
	const nonexistentBinary = "engram-binary-that-does-not-exist-abc123"

	err := mcpserver.ExportOSStarterStartBinary(t.Context(), nonexistentBinary, "http://localhost:9999")

	g.Expect(err).To(HaveOccurred())
}

func TestRunWithDeps_WhenServerAlreadyRunning_TransportClosesCleanly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	apiServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		data, _ := json.Marshal(apiclient.StatusResponse{Running: true})
		_, _ = writer.Write(data)
	}))
	defer apiServer.Close()

	starter := &fakeStarter{startErr: nil}
	transport := &eofTransport{}

	err := mcpserver.ExportRunWithDeps(
		context.Background(),
		apiServer.URL,
		http.DefaultClient,
		starter,
		transport,
	)

	// server.Run returns nil when the session ends cleanly (EOF from transport).
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunWithDeps_WhenServerNotAvailable_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	starter := &fakeStarter{startErr: errors.New("cannot start")}
	transport := &eofTransport{}

	err := mcpserver.ExportRunWithDeps(
		context.Background(),
		"http://localhost:1", // unreachable
		http.DefaultClient,
		starter,
		transport,
	)

	g.Expect(err).To(HaveOccurred())
}

func TestRun_WhenContextAlreadyCancelled_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Run is called

	// With a cancelled context, Status fails and the polling loop exits immediately.
	// Run returns an error regardless of the API address.
	err := mcpserver.Run(ctx)

	g.Expect(err).To(HaveOccurred())
}

// eofConnection is a fake mcp.Connection that returns io.EOF on the first Read.
type eofConnection struct{}

// Close implements mcp.Connection.
func (*eofConnection) Close() error {
	return nil
}

// Read implements mcp.Connection.
func (*eofConnection) Read(_ context.Context) (jsonrpc.Message, error) {
	return nil, io.EOF
}

// SessionID implements mcp.Connection.
func (*eofConnection) SessionID() string {
	return "test-session"
}

// Write implements mcp.Connection.
func (*eofConnection) Write(_ context.Context, _ jsonrpc.Message) error {
	return nil
}

// eofTransport is a fake mcp.Transport whose connection immediately returns io.EOF on Read.
type eofTransport struct{}

// Connect implements mcp.Transport.
func (eofTransport) Connect(_ context.Context) (mcp.Connection, error) {
	return &eofConnection{}, nil
}

// fakeStarter is a mock ServerStarter for testing.
type fakeStarter struct {
	startErr error
}

// Start implements ServerStarter.
func (fs *fakeStarter) Start(_ context.Context, _ string) error {
	return fs.startErr
}
