package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildSelfArgs holds parsed flags for the build-self subcommand.
type BuildSelfArgs struct {
	IfStale    bool   `targ:"flag,name=if-stale,desc=only build if sources are newer than the binary"`
	PluginRoot string `targ:"flag,name=plugin-root,env=ENGRAM_PLUGIN_ROOT,desc=path to plugin source root"`
	BinPath    string `targ:"flag,name=bin-path,desc=path to install the engram binary (default $HOME/.local/bin/engram)"`
}

// buildAndInstall builds via the supplied builder function then renames the
// temp output into the final binary path.
func buildAndInstall(
	ctx context.Context,
	build func(ctx context.Context, pluginRoot, tmpPath string, w io.Writer) error,
	pluginRoot, binPath string,
	stdout io.Writer,
) error {
	tmpPath := binPath + ".tmp"

	buildErr := build(ctx, pluginRoot, tmpPath, stdout)
	if buildErr != nil {
		_, _ = fmt.Fprintf(stdout, "build failed: %v\n", buildErr)
		return fmt.Errorf("build: %w", buildErr)
	}

	renameErr := os.Rename(tmpPath, binPath)
	if renameErr != nil {
		_, _ = fmt.Fprintf(stdout, "install failed: %v\n", renameErr)
		return fmt.Errorf("rename: %w", renameErr)
	}

	return nil
}

// goBuildRunner runs `go build -o tmpPath ./cmd/engram/` in pluginRoot, with output to w.
func goBuildRunner(ctx context.Context, pluginRoot, tmpPath string, w io.Writer) error {
	//nolint:gosec // hardcoded "go" command; tmpPath derives from operator-supplied flag.
	cmd := exec.CommandContext(ctx, "go", "build", "-o", tmpPath, "./cmd/engram/")
	cmd.Dir = pluginRoot
	cmd.Stdout = w
	cmd.Stderr = w

	runErr := cmd.Run()
	if runErr != nil {
		return fmt.Errorf("go build: %w", runErr)
	}

	return nil
}

// isStale reports whether any .go file under pluginRoot is newer than binPath
// (or the binary is missing).
func isStale(pluginRoot, binPath string) (bool, error) {
	binInfo, err := os.Stat(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return false, fmt.Errorf("stat binary: %w", err)
	}

	binMtime := binInfo.ModTime()
	stale := false

	walkErr := filepath.WalkDir(pluginRoot, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr //nolint:wrapcheck // walk callback
		}

		if info.ModTime().After(binMtime) {
			stale = true
			return filepath.SkipAll
		}

		return nil
	})
	if walkErr != nil {
		return false, fmt.Errorf("walk: %w", walkErr)
	}

	return stale, nil
}

// resolveBinPath returns the configured binary path or the standard default
// under the user's home directory.
func resolveBinPath(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home: %w", err)
	}

	return filepath.Join(home, ".local", "bin", "engram"), nil
}

func runBuildSelf(ctx context.Context, args BuildSelfArgs, stdout io.Writer) error {
	binPath, resolveErr := resolveBinPath(args.BinPath)
	if resolveErr != nil {
		return fmt.Errorf("build-self: %w", resolveErr)
	}

	if args.IfStale {
		stale, err := isStale(args.PluginRoot, binPath)
		if err != nil {
			return fmt.Errorf("build-self: checking staleness: %w", err)
		}

		if !stale {
			return nil
		}
	}

	buildErr := buildAndInstall(ctx, goBuildRunner, args.PluginRoot, binPath, stdout)
	if buildErr != nil {
		return fmt.Errorf("build-self: %w", buildErr)
	}

	return nil
}
