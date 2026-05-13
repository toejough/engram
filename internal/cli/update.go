package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	DryRun bool `targ:"flag,name=dry-run,desc=print planned actions without executing them"`
}

// unexported variables.
var (
	_                     update.Filesystem = (*osUpdateFS)(nil)
	errAllHarnessesFailed                   = errors.New("update: all detected harnesses failed")
)

type osCommander struct{}

func (*osCommander) Run(
	ctx context.Context, name string, args ...string,
) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // name/args from internal callers
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

type osDirEntry struct{ entry fs.DirEntry }

func (o *osDirEntry) IsDir() bool { return o.entry.IsDir() }

func (o *osDirEntry) Name() string { return o.entry.Name() }

type osFileInfo struct{ info fs.FileInfo }

func (o *osFileInfo) IsDir() bool { return o.info.IsDir() }

type osUpdateEnv struct{}

func (*osUpdateEnv) Getenv(key string) string {
	return os.Getenv(key)
}

func (*osUpdateEnv) Getwd() (string, error) {
	return os.Getwd() //nolint:wrapcheck // adapter; caller wraps with context
}

func (*osUpdateEnv) UserHomeDir() (string, error) {
	return os.UserHomeDir() //nolint:wrapcheck // adapter; caller wraps with context
}

// --- production adapters --------------------------------------------------

type osUpdateFS struct{}

func (*osUpdateFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

func (*osUpdateFS) ReadDir(path string) ([]update.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
		return nil, err
	}

	out := make([]update.DirEntry, 0, len(entries))

	for _, entry := range entries {
		out = append(out, &osDirEntry{entry: entry})
	}

	return out, nil
}

func (*osUpdateFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path supplied by walker
	if err != nil {
		//nolint:wrapcheck // adapter; caller adds context
		return nil, err
	}

	return data, nil
}

func (*osUpdateFS) Stat(path string) (update.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		//nolint:wrapcheck // thin I/O adapter; caller distinguishes fs.ErrNotExist via errors.Is
		return nil, err
	}

	return &osFileInfo{info: info}, nil
}

func (*osUpdateFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func anyHarnessSucceeded(report update.Report) bool {
	return slices.ContainsFunc(report.Harnesses, harnessOK)
}

func describeSource(source update.SourceInfo) string {
	switch source.Mode {
	case update.SourceLocal:
		return "local clone at " + source.Root
	case update.SourceRemote:
		return "remote module " + update.ModulePath + " " + source.Version
	default:
		return "unknown"
	}
}

// finishUpdate is the pure-decision tail of runUpdate, broken out for tests.
func finishUpdate(stdout io.Writer, report update.Report, runErr error) error {
	if runErr != nil {
		return fmt.Errorf("update: %w", runErr)
	}

	writeErr := writeUpdateReport(stdout, report)
	if writeErr != nil {
		return fmt.Errorf("update: writing report: %w", writeErr)
	}

	if !anyHarnessSucceeded(report) {
		return errAllHarnessesFailed
	}

	return nil
}

func harnessOK(harness update.HarnessReport) bool { return harness.Err == nil }

// runUpdate wires production adapters and invokes Updater.Run.
func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error {
	updater := &update.Updater{
		FS:  &osUpdateFS{},
		Cmd: &osCommander{},
		Env: &osUpdateEnv{},
	}

	report, runErr := updater.Run(ctx, update.Options{DryRun: args.DryRun})

	return finishUpdate(stdout, report, runErr)
}

func writeUpdateReport(out io.Writer, report update.Report) error {
	var buffer bytes.Buffer

	prefix := ""
	if report.DryRun {
		prefix = "[dry-run] "
	}

	fmt.Fprintf(&buffer, "%sengram update\n", prefix)
	fmt.Fprintf(&buffer, "  source: %s\n", describeSource(report.Source))
	fmt.Fprintf(&buffer, "  binary: %s\n", report.GoInstall)

	successes := make([]string, 0, len(report.Harnesses))

	for _, harness := range report.Harnesses {
		fmt.Fprintf(&buffer, "  %s (%s):\n", harness.Name, harness.SkillsRoot)

		if harness.Err != nil {
			fmt.Fprintf(&buffer, "    error: %v\n", harness.Err)

			continue
		}

		fmt.Fprintf(&buffer, "    skills: %d file(s)\n", harness.SkillFiles)

		if harness.CommandsRoot != "" {
			fmt.Fprintf(&buffer, "    commands: %d file(s)\n", harness.CommandFiles)
		}

		successes = append(successes, string(harness.Name))
	}

	if len(successes) > 0 {
		fmt.Fprintf(&buffer, "%sinstalled: %s\n", prefix, strings.Join(successes, ", "))
	}

	_, err := out.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
