package mcppoc

import (
	"context"
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Exported types for testing.

// Notifier is the exported form of the notifier interface for testing.
type Notifier = notifier

// ExportNewPOCServer exports newPOCServer for testing.
func ExportNewPOCServer() *mcp.Server {
	return newPOCServer(slog.Default())
}

// ExportNewWriterNotifier creates a writerNotifier writing to the given writer.
// Used to test Send() directly without writing to os.Stdout.
func ExportNewWriterNotifier(writer io.Writer) Notifier {
	return &writerNotifier{writer: writer}
}

// ExportRunNotificationLoop exports runNotificationLoop for testing with zero delays.
func ExportRunNotificationLoop(ctx context.Context, n Notifier) {
	runNotificationLoopWithDelays(ctx, slog.Default(), n, 0, 0)
}

// ExportRunNotificationLoopReal exports runNotificationLoop with real production delays.
// Used to hit the runNotificationLoop function itself (not just its core).
func ExportRunNotificationLoopReal(ctx context.Context, n Notifier) {
	runNotificationLoop(ctx, slog.Default(), n)
}

// ExportRunWithDeps exports runWithDeps for testing with a fake transport.
func ExportRunWithDeps(ctx context.Context, writer io.Writer, transport mcp.Transport) error {
	return runWithDeps(ctx, writer, transport)
}
