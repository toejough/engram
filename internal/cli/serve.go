package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"engram/internal/memory"
	"engram/internal/retrieve"
	"engram/internal/server"
	"engram/internal/tomlwriter"
)

// Exported variables.
var (
	// BrowserOpener opens a URL in the user's default browser.
	// Override in tests to prevent actually opening a browser.
	BrowserOpener = func(url string) { //nolint:gochecknoglobals // test-overridable
		var cmd string

		switch runtime.GOOS {
		case "darwin":
			cmd = "open"
		default:
			cmd = "xdg-open"
		}

		_ = exec.Command(cmd, url).Start() //nolint:gosec // url is constructed internally
	}
)

// unexported constants.
const (
	serverReadHeaderTimeout = 10 * time.Second
	serverShutdownTimeout   = 5 * time.Second
)

// osFileOps implements server.FileOps using the real filesystem.
type osFileOps struct{}

func (osFileOps) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

func (osFileOps) Rename(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldpath, newpath, err)
	}

	return nil
}

func (osFileOps) Stat(path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

// runServe starts the engram HTTP API server.
//
//nolint:funlen // CLI wiring: flag parsing + server setup + graceful shutdown
func runServe(args []string, stdout io.Writer) error {
	flagSet := flag.NewFlagSet("serve", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	port := flagSet.String("port", server.DefaultPort, "HTTP server port")
	dataDir := flagSet.String("data-dir", "", "path to data directory")

	parseErr := flagSet.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("serve: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("serve: %w", defaultErr)
	}

	retriever := retrieve.New()
	modifier := memory.NewModifier(
		memory.WithModifierWriter(tomlwriter.New()),
	)
	srv := server.NewServer(retriever, *dataDir,
		server.WithModifier(modifier),
		server.WithFileOps(osFileOps{}),
	)

	addr := server.ListenAddr(*port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	ctx, cancel := signalContext()
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		listenErr := httpServer.ListenAndServe()
		if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			errCh <- listenErr
		}

		close(errCh)
	}()

	url := "http://localhost:" + *port
	_, _ = fmt.Fprintf(stdout, "engram server listening on %s\n", url)

	BrowserOpener(url)

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer shutdownCancel()

		shutdownErr := httpServer.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			return fmt.Errorf("serve: shutdown: %w", shutdownErr)
		}

		_, _ = fmt.Fprintln(stdout, "engram server stopped")
	}

	return nil
}
