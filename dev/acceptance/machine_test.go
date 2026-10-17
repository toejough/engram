//go:build network

// Package acceptance_test drives real Claude Code sessions in isolated homes, one test for each scenario of the
// product spec and for the technical scenarios that need a real session (793b9). Every test here starts processes
// that may reach the network (`claude`, `go install`), so the package carries the network build tag and no default
// target runs it; `targ acceptance` does.
//
// Isolation (793f7): every `engram` and `claude` run gets its own HOME, CLAUDE_CONFIG_DIR, GOBIN, ENGRAM_VAULT_PATH,
// ENGRAM_CHUNKS_DIR and ENGRAM_TRANSCRIPT_DIR under a temporary directory, and a working directory outside any
// `.claude`, `.git`, `.hg` or `.jj` ancestor; the owner's own ~/.claude, ~/.local/share/engram, ~/.pi and GOBIN are
// never reached.
package acceptance_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	code := m.Run()

	if sharedBuild.dir != "" {
		_ = os.RemoveAll(sharedBuild.dir)
	}

	os.Exit(code)
}

// unexported constants.
const (
	// the names of the directories and files of the module that building and updating need; everything else of the
	// module (the git data, the .claude directory, the docs) is left out of the copy
	buildInputs      = "go.mod go.sum cmd internal agent-instructions"
	commandTimeout   = 10 * time.Minute
	dirPerm          = 0o750
	engramBinaryName = "engram"
	filePerm         = 0o600
	goBuildTimeout   = 15 * time.Minute
	prewarmPhrase    = "prewarm"
	systemPath       = "/usr/bin:/bin:/usr/sbin:/sbin"
)

// unexported variables.
var (
	// the markers of a directory that holds a repository or a harness's own state: a run inside one would sweep or
	// edit the owner's sources
	ancestorMarkers = []string{".claude", ".git", ".hg", ".jj"}
	errNoModuleRoot = errors.New("no go.mod above the acceptance tests")
	// the caches and paths of the Go toolchain, read from the test process so that a run under an isolated HOME builds
	// from the warm caches
	goCaches    = []string{"GOCACHE", "GOMODCACHE", "GOPATH"}
	sharedBuild shared
)

// machine is a user's computer for one scenario: its own home, its own vault and its own copy of the engram binary.
type machine struct {
	t           *testing.T
	root        string // the temporary directory that holds everything below
	home        string // HOME
	claudeDir   string // CLAUDE_CONFIG_DIR, ~/.claude
	gobin       string // GOBIN, where `engram update` installs the binary and where PATH finds it first
	vault       string // ENGRAM_VAULT_PATH; it does not exist until something creates it
	chunks      string // ENGRAM_CHUNKS_DIR
	transcripts string // ENGRAM_TRANSCRIPT_DIR, the directory Claude Code records sessions in
	gitconfig   string // GIT_CONFIG_GLOBAL, an empty file
	work        string // the working directory of runs that need none of their own
	tmp         string // TMPDIR
}

// assertIsolated fails the test unless a run in dir with env reaches nothing of the owner's.
func (m *machine) assertIsolated(dir string, env []string) {
	m.t.Helper()

	if marker := ancestorMarker(dir); marker != "" {
		m.t.Fatalf("working directory %s has the marker %s above it: a run there reaches the owner's sources", dir, marker)
	}

	values := map[string]string{}

	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		values[name] = value
	}

	owner, err := os.UserHomeDir()
	if err != nil {
		m.t.Fatalf("resolving the owner's home: %v", err)
	}

	for _, name := range []string{
		"HOME", "CLAUDE_CONFIG_DIR", "GOBIN", "ENGRAM_VAULT_PATH", "ENGRAM_CHUNKS_DIR", "ENGRAM_TRANSCRIPT_DIR",
	} {
		value := values[name]

		if value == "" || !strings.HasPrefix(value, m.root) {
			m.t.Fatalf("%s=%q is not inside this machine's directory %s", name, value, m.root)
		}

		for _, forbidden := range []string{".claude", ".pi", filepath.Join(".local", "share", "engram")} {
			if strings.HasPrefix(value, filepath.Join(owner, forbidden)) {
				m.t.Fatalf("%s=%q is inside the owner's %s", name, value, forbidden)
			}
		}
	}
}

// engram runs the machine's engram with the given arguments, from its working directory.
func (m *machine) engram(args ...string) result {
	m.t.Helper()

	return m.run(m.work, nil, filepath.Join(m.gobin, engramBinaryName), args...)
}

// env is the whole environment of every command run on the machine.
func (m *machine) env(extra ...string) []string {
	env := []string{
		"PATH=" + strings.Join([]string{m.gobin, filepath.Dir(lookPath("go")), systemPath}, ":"),
		"HOME=" + m.home,
		"TMPDIR=" + m.tmp,
		"LANG=en_US.UTF-8",
		"CLAUDE_CONFIG_DIR=" + m.claudeDir,
		"GOBIN=" + m.gobin,
		"GIT_CONFIG_GLOBAL=" + m.gitconfig,
		"ENGRAM_VAULT_PATH=" + m.vault,
		"ENGRAM_CHUNKS_DIR=" + m.chunks,
		"ENGRAM_TRANSCRIPT_DIR=" + m.transcripts,
		"XDG_CACHE_HOME=" + sharedBuild.cache,
		"GOFLAGS=-buildvcs=false",
		"DISABLE_AUTOUPDATER=1",
	}

	return slices.Concat(env, goEnv(), extra)
}

// installEngram puts the binary the machine runs `engram update` with into its GOBIN.
func (m *machine) installEngram(binary string) {
	m.t.Helper()

	err := copyFile(binary, filepath.Join(m.gobin, engramBinaryName))
	if err != nil {
		m.t.Fatalf("installing engram on the machine: %v", err)
	}
}

// run runs one command with the machine's environment and returns what it printed and how it ended.
func (m *machine) run(dir string, extraEnv []string, name string, args ...string) result {
	m.t.Helper()

	env := m.env(extraEnv...)
	m.assertIsolated(dir, env)

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = env

	var stdout, stderr bytes.Buffer

	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

	var exitErr *exec.ExitError

	switch {
	case err == nil:
		return result{stdout: stdout.String(), stderr: stderr.String()}
	case errors.As(err, &exitErr):
		return result{stdout: stdout.String(), stderr: stderr.String(), exit: exitErr.ExitCode()}
	default:
		m.t.Fatalf("running %s: %v", name, err)

		return result{}
	}
}

// update runs `engram update` as the user does: the machine's own engram, from a checkout of this branch, with the
// machine's home. The checkout is the git-less copy of the module, so the working directory is outside any
// repository (793f7).
func (m *machine) update(args ...string) result {
	m.t.Helper()

	return m.run(sharedBuild.source, nil, filepath.Join(m.gobin, engramBinaryName), append([]string{"update"}, args...)...)
}

// result is what a finished command left behind.
type result struct {
	stdout string
	stderr string
	exit   int
}

// shared is built once for all tests of a run: a git-less copy of the module to run `engram update` from, the
// engram binary built from it, and a model cache every machine reads, so the 90 MB model is unpacked once.
type shared struct {
	once   sync.Once
	err    error
	dir    string // holds the three below
	source string
	binary string
	cache  string
}

// build copies the module without its git data, builds the binary from the copy and unpacks the model once.
func (s *shared) build() error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}

	s.dir, err = os.MkdirTemp("", "engram-acceptance-")
	if err != nil {
		return fmt.Errorf("creating the shared directory: %w", err)
	}

	s.source = filepath.Join(s.dir, "source")
	s.binary = filepath.Join(s.dir, "bin", engramBinaryName)
	s.cache = filepath.Join(s.dir, "cache")

	for name := range strings.FieldsSeq(buildInputs) {
		copyErr := copyTree(filepath.Join(root, name), filepath.Join(s.source, name))
		if copyErr != nil {
			return copyErr
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), goBuildTimeout)
	defer cancel()

	env := append(goEnv(), "PATH="+os.Getenv("PATH"), "HOME="+s.dir, "GOBIN="+filepath.Dir(s.binary),
		"GOFLAGS=-buildvcs=false")

	build := exec.CommandContext(ctx, "go", "install", "./cmd/engram")
	build.Dir = s.source
	build.Env = env

	out, buildErr := build.CombinedOutput()
	if buildErr != nil {
		return fmt.Errorf("go install ./cmd/engram: %w\n%s", buildErr, out)
	}

	return s.unpackModel()
}

// unpackModel runs one lookup against an empty vault so the bundled model is unpacked into the shared cache before
// the machines that read it run side by side.
func (s *shared) unpackModel() error {
	vault := filepath.Join(s.dir, "prewarm-vault")

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	prewarm := exec.CommandContext(ctx, s.binary, "query", "--phrase", prewarmPhrase, "--vault", vault, "--limit", "1")
	prewarm.Dir = s.dir
	prewarm.Env = []string{"HOME=" + s.dir, "XDG_CACHE_HOME=" + s.cache, "PATH=" + systemPath}

	out, err := prewarm.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unpacking the model: %w\n%s", err, out)
	}

	return nil
}

// ancestorMarker returns the first marker found at or above dir, or "".
func ancestorMarker(dir string) string {
	for current := dir; ; current = filepath.Dir(current) {
		for _, marker := range ancestorMarkers {
			candidate := filepath.Join(current, marker)

			_, err := os.Lstat(candidate)
			if err == nil {
				return candidate
			}
		}

		if filepath.Dir(current) == current {
			return ""
		}
	}
}

// copyFile copies one file, keeping its mode.
func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}

	defer func() { _ = input.Close() }()

	info, err := input.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	output, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}

	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()

	return errors.Join(copyErr, closeErr)
}

// copyTree copies the regular files of src into dst, linking them where the file system allows, which keeps the
// 90 MB model from being copied for each run.
func copyTree(src, dst string) error {
	walkErr := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return fmt.Errorf("relative path of %s: %w", path, relErr)
		}

		target := filepath.Join(dst, rel)

		if entry.IsDir() {
			return os.MkdirAll(target, dirPerm)
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		mkdirErr := os.MkdirAll(filepath.Dir(target), dirPerm)
		if mkdirErr != nil {
			return fmt.Errorf("creating the directory of %s: %w", target, mkdirErr)
		}

		if os.Link(path, target) == nil {
			return nil
		}

		return copyFile(path, target)
	})
	if walkErr != nil {
		return fmt.Errorf("copying %s: %w", src, walkErr)
	}

	return nil
}

// goEnv returns the Go toolchain's own caches as the test process sees them, so a run under an isolated HOME builds
// from the warm caches and not from nothing.
func goEnv() []string {
	env := make([]string, 0, len(goCaches))

	for _, name := range goCaches {
		out, err := exec.CommandContext(context.Background(), "go", "env", name).Output()
		if err == nil {
			env = append(env, name+"="+strings.TrimSpace(string(out)))
		}
	}

	return env
}

func lookPath(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return "/usr/local/go/bin/" + name
	}

	return path
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
}

// moduleRoot finds the directory of the module that these tests belong to.
func moduleRoot() (string, error) {
	_, file, _, _ := runtime.Caller(0)

	for dir := filepath.Dir(file); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		_, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil {
			return dir, nil
		}
	}

	return "", errNoModuleRoot
}

// newMachine makes a computer with a home and nothing in it: no Claude Code, no vault, no engram on its PATH until the
// scenario gives it one.
func newMachine(t *testing.T) *machine {
	t.Helper()

	build := prepareShared(t)

	resolved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolving the temporary directory: %v", err)
	}

	machine := &machine{t: t, root: resolved}
	machine.home = filepath.Join(resolved, "home")
	machine.claudeDir = filepath.Join(machine.home, ".claude")
	machine.gobin = filepath.Join(resolved, "gobin")
	machine.vault = filepath.Join(resolved, "vault")
	machine.chunks = filepath.Join(resolved, "chunks")
	machine.transcripts = filepath.Join(machine.claudeDir, "projects")
	machine.gitconfig = filepath.Join(resolved, "gitconfig")
	machine.work = filepath.Join(resolved, "work")
	machine.tmp = filepath.Join(resolved, "tmp")

	for _, dir := range []string{machine.home, machine.gobin, machine.work, machine.tmp} {
		mkdirAll(t, dir)
	}

	writeFile(t, machine.gitconfig, "")
	machine.installEngram(build.binary)

	return machine
}

// prepareShared builds, once, what every machine needs, and fails the test that finds it unbuilt.
func prepareShared(t *testing.T) *shared {
	t.Helper()

	sharedBuild.once.Do(func() { sharedBuild.err = sharedBuild.build() })

	if sharedBuild.err != nil {
		t.Fatalf("preparing the shared build: %v", sharedBuild.err)
	}

	return &sharedBuild
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), filePerm)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
