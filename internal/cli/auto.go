package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/update"
)

// unexported constants.
const (
	autoDirPerm  fs.FileMode = 0o755
	autoFilePerm fs.FileMode = 0o644
)

// unexported variables.
var (
	errAutoCut        = errors.New("still running after")
	errAutoNotStarted = errors.New("failed to start")
)

// autoClock is the wall clock of a job.
type autoClock struct{ now func() time.Time }

// Now is the time.
func (c autoClock) Now() time.Time { return c.now() }

// autoEnv reads the environment of a job.
type autoEnv struct{ getenv func(string) string }

// Get reads a variable.
func (e autoEnv) Get(key string) string { return e.getenv(key) }

// autoFiles is the file system of a job.
type autoFiles struct{ edge EdgeFS }

// Exists says whether the path is there.
func (f autoFiles) Exists(path string) bool {
	_, err := f.edge.Stat(path)

	return err == nil
}

// Read reads a file.
func (f autoFiles) Read(path string) (string, error) {
	data, err := f.edge.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading: %w", err)
	}

	return string(data), nil
}

// Walk lists the regular files below root.
func (f autoFiles) Walk(root string) ([]string, error) {
	var files []string

	err := f.edge.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr == nil && entry.Type().IsRegular() {
			files = append(files, path)
		}

		return walkErr
	})
	if err != nil {
		return nil, fmt.Errorf("walking: %w", err)
	}

	return files, nil
}

// Write writes a file, and the directory it is in.
func (f autoFiles) Write(path, text string) error {
	err := f.edge.MkdirAll(filepath.Dir(path), autoDirPerm)
	if err == nil {
		err = f.edge.WriteFileAtomic(path, []byte(text), autoFilePerm)
	}

	if err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return nil
}

// autoRunner runs the commands of a job.
type autoRunner func(argv []string, deadline time.Duration) (auto.Output, error)

// Run runs a command within its deadline.
func (r autoRunner) Run(argv []string, deadline time.Duration) (auto.Output, error) {
	return r(argv, deadline)
}

// autoTargets returns `engram auto run`.
func autoTargets(deps Deps, errHandler func(error)) any {
	return targ.Group("auto",
		targ.Targ(func(ctx context.Context) {
			errHandler(runAuto(ctx, deps))
		}).Name("run").Description(
			"Advance one job of the memory mod: one JSON job on standard input, one JSON response on standard output"),
	)
}

// runAuto advances one job of the memory mod (793b368, 793e8m): the job is the JSON on standard input, the response the
// JSON on standard output.
func runAuto(ctx context.Context, deps Deps) error {
	input, err := io.ReadAll(deps.Stdin)
	if err != nil {
		return fmt.Errorf("auto run: reading the job: %w", err)
	}

	output, err := auto.Run(auto.Deps{
		Runner: autoRunner(func(argv []string, deadline time.Duration) (auto.Output, error) {
			return runAutoCommand(ctx, deps.Commander, argv, deadline)
		}),
		Files: autoFiles{edge: deps.FS}, Env: autoEnv{getenv: deps.Getenv}, Clock: autoClock{now: deps.Now},
	}, input)
	if err != nil {
		return fmt.Errorf("auto run: %w", err)
	}

	_, err = deps.Stdout.Write(output)
	if err != nil {
		return fmt.Errorf("auto run: writing the response: %w", err)
	}

	return nil
}

// runAutoCommand runs a command through the injected Commander within its deadline. What it reports is the text the
// engine classifies a failure by: the standard error of a command that failed, "failed to start" for one that could
// not, "still running after" for one that was cut.
func runAutoCommand(
	ctx context.Context, cmd update.Commander, argv []string, deadline time.Duration,
) (auto.Output, error) {
	limited, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	stdout, stderr, err := cmd.Run(limited, "", argv[0], argv[1:]...)
	out := auto.Output{Stdout: string(stdout), Stderr: string(stderr)}

	switch {
	case err == nil:
		return out, nil
	case limited.Err() != nil:
		return out, fmt.Errorf("%w %dms", errAutoCut, deadline.Milliseconds())
	case errors.Is(err, update.ErrCommandNotFound):
		return out, fmt.Errorf("%w: %s is not on the path", errAutoNotStarted, argv[0])
	}

	for errors.Unwrap(err) != nil {
		err = errors.Unwrap(err)
	}

	return out, err
}
