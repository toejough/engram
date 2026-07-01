// Package update implements the `engram update` subcommand: refresh the
// engram binary via `go install` and copy harness skills/commands from
// either a local clone or the module cache into per-harness user dirs.
package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// Exported constants.
const (
	HarnessClaude   Harness = "Claude Code"
	HarnessOpencode Harness = "OpenCode"
	ModulePath              = "github.com/toejough/engram"
)

// SourceMode tells whether the binary/skills came from a local clone or
// from the resolved remote module.
type SourceMode int

// SourceMode values.
const (
	SourceLocal SourceMode = iota + 1
	SourceRemote
)

// Exported variables.
var (
	ErrGitNotFound = errors.New("git binary not found on PATH")
	ErrGoNotFound  = errors.New("go binary not found on PATH")
	// ErrModelLFSStub means the cloned model.onnx is a Git-LFS pointer file,
	// not the real model — building from it would embed a 133-byte stub and
	// every embedding call would fail (issue #645).
	ErrModelLFSStub     = errors.New("model.onnx is a git-lfs pointer stub")
	ErrNoHarness        = errors.New("no supported harness found")
	ErrSkillsSrcMissing = errors.New("skills source dir missing")
)

// Commander runs an external command, capturing stdout and stderr.
// dir sets the working directory; empty string inherits the process cwd.
type Commander interface {
	Run(ctx context.Context, dir, name string, args ...string) (stdout, stderr []byte, err error)
}

// CopyOp describes a single source→target file copy planned for a harness.
// SkillDir is the top-level skill subdir name (e.g. "learn") when the file
// belongs to a skill, empty otherwise. CommandFile is the basename when the
// file is a command .md, empty otherwise. GuidanceFile is the basename when
// the file is a guidance .md, empty otherwise. Exactly one of these is set.
type CopyOp struct {
	Harness      Harness
	Src          string
	Dst          string
	SkillDir     string
	CommandFile  string
	GuidanceFile string
}

// DirEntry is the subset of fs.DirEntry used by Updater.
type DirEntry interface {
	Name() string
	IsDir() bool
}

// Env is the injected environment surface (home dir, env vars, cwd).
type Env interface {
	Getenv(key string) string
	UserHomeDir() (string, error)
	Getwd() (string, error)
}

// FileInfo is the subset of fs.FileInfo used by Updater.
type FileInfo interface {
	IsDir() bool
}

// Filesystem is the injected I/O surface for the updater. All paths are
// absolute. Implementations are stateless wrappers around os.* calls.
type Filesystem interface {
	Stat(path string) (FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	ReadDir(path string) ([]DirEntry, error)
	// RemoveAll removes path and any children. Used to clear stale
	// destinations (broken symlinks, old files, symlinks pointing at the
	// source repo) before writing fresh content. Same semantics as
	// os.RemoveAll: nil if the path doesn't exist; errors only on real
	// I/O failure.
	RemoveAll(path string) error
}

// Harness names a supported agent harness. The zero value is invalid.
type Harness string

// HarnessReport summarizes one harness install attempt.
type HarnessReport struct {
	Name          Harness
	ProbeRoot     string // home-relative harness root, e.g. ".claude"
	SkillsRoot    string // absolute skills install dir
	CommandsRoot  string // absolute commands install dir (empty if harness has no commands)
	SkillDirs     []SkillDirCount
	CommandFiles  []string // basenames of .md files copied
	GuidanceFiles []string // basenames of .md files copied into the guidance dir
	Err           error
}

// HarnessSpec captures one harness's well-known paths (relative to home).
type HarnessSpec struct {
	Name              Harness
	ProbeRel          string // dir to stat under home (e.g. ".claude")
	SkillsTargetRel   string // skills install dir under home
	CommandsTargetRel string // commands install dir under home (empty: skip commands)
	GuidanceTargetRel string // guidance install dir under home (empty: skip guidance)
}

// Options controls one Run invocation.
type Options struct {
	DryRun       bool
	WithGuidance bool // deploy guidance/*.md to the harness guidance dir
}

// Report is the final outcome of Updater.Run, suitable for formatting.
type Report struct {
	DryRun           bool
	WithGuidance     bool   // whether --with-guidance was requested
	Home             string // user home (so the CLI can tildify paths)
	Source           SourceInfo
	GoInstall        string // command line invoked (or planned)
	BinaryPath       string // resolved install location, e.g. /Users/joe/go/bin/engram
	BinaryVersion    string // resolved engram version, empty when unknown
	GuidanceImported bool   // true when ~/.claude/CLAUDE.md imports the guidance file
	Harnesses        []HarnessReport
}

// SkillDirCount records how many files were copied into one skill dir.
type SkillDirCount struct {
	Name  string // top-level skill dir name, e.g. "learn"
	Files int    // number of files copied into <skills-root>/<Name>/
}

// SourceInfo describes where the binary was installed from and where the
// skill files are read from.
type SourceInfo struct {
	Mode    SourceMode
	Root    string // repo root (local) or modcache dir (remote)
	Version string // resolved version string (remote only)
}

// Updater applies an `engram update` operation against injected I/O.
type Updater struct {
	FS  Filesystem
	Cmd Commander
	Env Env
}

// Run executes (or plans, when DryRun) the update flow.
func (u *Updater) Run(ctx context.Context, opts Options) (Report, error) {
	report := Report{DryRun: opts.DryRun, WithGuidance: opts.WithGuidance}

	home, homeErr := u.Env.UserHomeDir()
	if homeErr != nil {
		return report, fmt.Errorf("resolving home: %w", homeErr)
	}

	report.Home = home
	report.BinaryPath = resolveBinaryPath(home, u.Env)

	harnesses, detectErr := detectHarnesses(home, u.FS)
	if detectErr != nil {
		return report, fmt.Errorf("detecting harnesses: %w", detectErr)
	}

	if len(harnesses) == 0 {
		return report, fmt.Errorf("%w at ~/.claude/ or ~/.config/opencode/", ErrNoHarness)
	}

	source, sourceErr := u.resolveSource(ctx, opts.DryRun)
	if sourceErr != nil {
		return report, sourceErr
	}

	report.Source = source
	report.GoInstall = describeGoInstall(source)
	report.BinaryVersion = source.Version // local mode leaves this empty

	srcSkills := filepath.Join(source.Root, "skills")
	srcCommands := filepath.Join(source.Root, "commands")
	srcGuidance := filepath.Join(source.Root, "guidance")

	skillOps, planErr := planSkillCopies(srcSkills, home, harnesses, u.FS)
	if planErr != nil {
		return report, planErr
	}

	cmdOps, cmdPlanErr := planCommandCopies(srcCommands, home, harnesses, u.FS)
	if cmdPlanErr != nil {
		return report, cmdPlanErr
	}

	var guidanceOps []CopyOp

	if opts.WithGuidance {
		var guidancePlanErr error

		guidanceOps, guidancePlanErr = planGuidanceCopies(srcGuidance, home, harnesses, u.FS)
		if guidancePlanErr != nil {
			return report, guidancePlanErr
		}
	}

	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")
	report.GuidanceImported = detectGuidanceImport(claudeMDPath, home, u.FS)

	report.Harnesses = u.applyOps(harnesses, home, skillOps, cmdOps, guidanceOps, opts.DryRun)

	return report, nil
}

func (u *Updater) applyCmdOne(copyOp CopyOp, dryRun bool) error {
	if !dryRun {
		removeErr := u.FS.RemoveAll(copyOp.Dst)
		if removeErr != nil {
			return fmt.Errorf("clear %s: %w", copyOp.Dst, removeErr)
		}
	}

	return u.applyOne(copyOp, dryRun)
}

func (u *Updater) applyCmdOps(rep *HarnessReport, name Harness, cmdOps []CopyOp, dryRun bool) {
	for _, copyOp := range cmdOps {
		if copyOp.Harness != name {
			continue
		}

		opErr := u.applyCmdOne(copyOp, dryRun)
		if opErr != nil {
			rep.Err = opErr

			return
		}

		rep.CommandFiles = append(rep.CommandFiles, copyOp.CommandFile)
	}
}

func (u *Updater) applyForHarness(
	rep *HarnessReport,
	name Harness,
	skillOps, cmdOps, guidanceOps []CopyOp,
	dryRun bool,
) {
	u.applySkillOps(rep, name, skillOps, dryRun)

	if rep.Err != nil {
		return
	}

	u.applyCmdOps(rep, name, cmdOps, dryRun)

	if rep.Err != nil {
		return
	}

	u.applyGuidanceOps(rep, name, guidanceOps, dryRun)
}

func (u *Updater) applyGuidanceOps(rep *HarnessReport, name Harness, guidanceOps []CopyOp, dryRun bool) {
	for _, copyOp := range guidanceOps {
		if copyOp.Harness != name {
			continue
		}

		opErr := u.applyCmdOne(copyOp, dryRun)
		if opErr != nil {
			rep.Err = opErr

			return
		}

		rep.GuidanceFiles = append(rep.GuidanceFiles, copyOp.GuidanceFile)
	}
}

func (u *Updater) applyOne(copyOp CopyOp, dryRun bool) error {
	if dryRun {
		return nil
	}

	data, readErr := u.FS.ReadFile(copyOp.Src)
	if readErr != nil {
		return fmt.Errorf("read %s: %w", copyOp.Src, readErr)
	}

	mkErr := u.FS.MkdirAll(filepath.Dir(copyOp.Dst), dirPerm)
	if mkErr != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(copyOp.Dst), mkErr)
	}

	writeErr := u.FS.WriteFile(copyOp.Dst, data, filePerm)
	if writeErr != nil {
		return fmt.Errorf("write %s: %w", copyOp.Dst, writeErr)
	}

	return nil
}

// applyOps copies files for every CopyOp and returns per-harness reports.
// Failures for one harness do not stop the others. The CLI is responsible
// for deciding the exit code (any detected harness failed → 1).
func (u *Updater) applyOps(
	harnesses []HarnessSpec,
	home string,
	skillOps, cmdOps, guidanceOps []CopyOp,
	dryRun bool,
) []HarnessReport {
	reports := make([]HarnessReport, 0, len(harnesses))

	for _, spec := range harnesses {
		rep := HarnessReport{
			Name:         spec.Name,
			ProbeRoot:    spec.ProbeRel,
			SkillsRoot:   filepath.Join(home, spec.SkillsTargetRel),
			CommandsRoot: cmdRootFor(spec, home),
		}

		u.applyForHarness(&rep, spec.Name, skillOps, cmdOps, guidanceOps, dryRun)
		reports = append(reports, rep)
	}

	return reports
}

func (u *Updater) applySkillOps(rep *HarnessReport, name Harness, skillOps []CopyOp, dryRun bool) {
	skillCounts := map[string]int{}
	skillOrder := make([]string, 0)
	cleared := map[string]bool{}

	defer func() { rep.SkillDirs = collectSkillDirs(skillOrder, skillCounts) }()

	for _, copyOp := range skillOps {
		if copyOp.Harness != name {
			continue
		}

		clearErr := u.clearSkillDirOnce(rep, copyOp, cleared, dryRun)
		if clearErr != nil {
			return
		}

		applyErr := u.applyOne(copyOp, dryRun)
		if applyErr != nil {
			rep.Err = applyErr

			return
		}

		if _, seen := skillCounts[copyOp.SkillDir]; !seen {
			skillOrder = append(skillOrder, copyOp.SkillDir)
		}

		skillCounts[copyOp.SkillDir]++
	}
}

// clearSkillDirOnce removes the per-harness top-level skill directory the
// first time a CopyOp for that SkillDir is processed. This ensures stale
// files are dropped, broken symlinks are replaced, and symlinks pointing
// at the source repo cannot cause WriteFile to mutate the source.
func (u *Updater) clearSkillDirOnce(
	rep *HarnessReport,
	copyOp CopyOp,
	cleared map[string]bool,
	dryRun bool,
) error {
	if dryRun || copyOp.SkillDir == "" {
		return nil
	}

	target := filepath.Join(rep.SkillsRoot, copyOp.SkillDir)
	if cleared[target] {
		return nil
	}

	removeErr := u.FS.RemoveAll(target)
	if removeErr != nil {
		rep.Err = fmt.Errorf("clear %s: %w", target, removeErr)

		return removeErr
	}

	cleared[target] = true

	return nil
}

// resolveRemoteByClone implements remote mode by CLONING the repo and
// building from the clone — never `go install …@latest`. The Go module proxy
// serves raw repository blobs, so the LFS-tracked model.onnx arrives as a
// 133-byte pointer stub and //go:embed bakes a broken embedder into the
// binary (issue #645). A git clone runs the LFS smudge filter and
// materializes the real model; the stub check below catches machines where
// git-lfs is not installed (smudge never ran).
func (u *Updater) resolveRemoteByClone(ctx context.Context, dryRun bool) (SourceInfo, error) {
	cloneDir := u.tempCloneDir()

	rmErr := u.FS.RemoveAll(cloneDir)
	if rmErr != nil {
		return SourceInfo{}, fmt.Errorf("clearing previous clone %s: %w", cloneDir, rmErr)
	}

	_, _, cloneErr := u.Cmd.Run(ctx, "", "git", "clone", "--depth", "1", repoCloneURL, cloneDir)
	if cloneErr != nil {
		if errors.Is(cloneErr, exec.ErrNotFound) {
			return SourceInfo{}, fmt.Errorf("git clone: %w", ErrGitNotFound)
		}

		return SourceInfo{}, fmt.Errorf("git clone: %w", cloneErr)
	}

	stubErr := u.verifyModelNotLFSStub(cloneDir)
	if stubErr != nil {
		return SourceInfo{}, stubErr
	}

	if !dryRun {
		_, _, runErr := u.Cmd.Run(ctx, cloneDir, "go", "install", "./cmd/engram/")
		if runErr != nil {
			return SourceInfo{}, classifyGoInstallErr("remote", runErr)
		}
	}

	version := "unknown"

	out, _, revErr := u.Cmd.Run(ctx, cloneDir, "git", "rev-parse", "--short", "HEAD")
	if revErr == nil {
		if v := strings.TrimSpace(string(out)); v != "" {
			version = v
		}
	}

	return SourceInfo{Mode: SourceRemote, Root: cloneDir, Version: version}, nil
}

// resolveSource picks between local clone and remote module by walking up
// from cwd. On remote, runs `go install ...@latest` and `go list -m -json`
// to locate the module cache dir.
func (u *Updater) resolveSource(ctx context.Context, dryRun bool) (SourceInfo, error) {
	cwd, cwdErr := u.Env.Getwd()
	if cwdErr != nil {
		return SourceInfo{}, fmt.Errorf("getwd: %w", cwdErr)
	}

	root, found, walkErr := walkUpForModule(cwd, u.FS)
	if walkErr != nil {
		return SourceInfo{}, walkErr
	}

	if found {
		if !dryRun {
			_, _, runErr := u.Cmd.Run(ctx, root, "go", "install", "./cmd/engram/")
			if runErr != nil {
				return SourceInfo{}, classifyGoInstallErr("local", runErr)
			}
		}

		return SourceInfo{Mode: SourceLocal, Root: root}, nil
	}

	return u.resolveRemoteByClone(ctx, dryRun)
}

// tempCloneDir is a deterministic scratch location for the remote-mode clone.
func (u *Updater) tempCloneDir() string {
	tmp := u.Env.Getenv("TMPDIR")
	if tmp == "" {
		tmp = "/tmp"
	}

	return filepath.Join(tmp, "engram-update-clone")
}

// verifyModelNotLFSStub fails loudly when the cloned model file is a Git-LFS
// pointer (git-lfs absent → smudge never ran) instead of shipping a binary
// that only breaks at first embed.
func (u *Updater) verifyModelNotLFSStub(cloneDir string) error {
	modelPath := filepath.Join(cloneDir, "internal", "embed", "assets", "model", "model.onnx")

	data, readErr := u.FS.ReadFile(modelPath)
	if readErr != nil {
		return fmt.Errorf("reading cloned model %s: %w", modelPath, readErr)
	}

	if bytes.HasPrefix(data, []byte(lfsPointerPrefix)) || len(data) < modelMinBytes {
		return fmt.Errorf(
			"%s holds %d bytes — %w; install git-lfs (`brew install git-lfs && git lfs install`) and rerun `engram update`",
			modelPath, len(data), ErrModelLFSStub)
	}

	return nil
}

// unexported constants.
const (
	// dirPerm is the mode used when creating any harness target dir.
	dirPerm fs.FileMode = 0o755
	// filePerm is the mode used when writing any copied file.
	filePerm fs.FileMode = 0o644
	// lfsPointerPrefix is the first line of every Git-LFS pointer file.
	lfsPointerPrefix      = "version https://git-lfs"
	maxSupportedHarnesses = 2
	// modelMinBytes: the real MiniLM ONNX is ~90 MB; anything under a
	// megabyte is certainly not it.
	modelMinBytes = 1 << 20
	repoCloneURL  = "https://" + ModulePath + ".git"
)

// classifyGoInstallErr maps a `go install` failure to ErrGoNotFound when the go
// binary is absent from PATH (exec.ErrNotFound), otherwise wrapping the raw
// error with the install mode for context.
func classifyGoInstallErr(mode string, runErr error) error {
	if errors.Is(runErr, exec.ErrNotFound) {
		return fmt.Errorf("go install (%s): %w", mode, ErrGoNotFound)
	}

	return fmt.Errorf("go install (%s): %w", mode, runErr)
}

func cmdRootFor(spec HarnessSpec, home string) string {
	if spec.CommandsTargetRel == "" {
		return ""
	}

	return filepath.Join(home, spec.CommandsTargetRel)
}

func collectSkillDirs(order []string, counts map[string]int) []SkillDirCount {
	out := make([]SkillDirCount, 0, len(order))

	for _, name := range order {
		out = append(out, SkillDirCount{Name: name, Files: counts[name]})
	}

	return out
}

func describeGoInstall(source SourceInfo) string {
	if source.Mode == SourceLocal {
		return "go install ./cmd/engram/"
	}

	return "git clone " + repoCloneURL + " && go install ./cmd/engram/ (LFS-safe; #645)"
}

// detectGuidanceImport returns true when the Claude Code CLAUDE.md file at
// claudeMDPath contains an active @import line for the guidance file (either
// the tilde form or the expanded-home form). Lines inside fenced code blocks
// are ignored per the @import rules. A missing CLAUDE.md yields false, no error.
func detectGuidanceImport(claudeMDPath, home string, fileSystem Filesystem) bool {
	data, readErr := fileSystem.ReadFile(claudeMDPath)
	if readErr != nil {
		return false
	}

	tildeLine := "@~/.claude/engram/recall.md"
	expandedLine := "@" + filepath.Join(home, ".claude", "engram", "recall.md")

	inFence := false

	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence

			continue
		}

		if inFence {
			continue
		}

		if trimmed == tildeLine || trimmed == expandedLine {
			return true
		}
	}

	return false
}

// detectHarnesses returns the supported harnesses whose probe path exists
// under home. Order is stable (matches supportedHarnesses).
func detectHarnesses(home string, fileSystem Filesystem) ([]HarnessSpec, error) {
	detected := make([]HarnessSpec, 0, maxSupportedHarnesses)

	for _, spec := range supportedHarnesses() {
		probe := filepath.Join(home, spec.ProbeRel)

		info, err := fileSystem.Stat(probe)
		switch {
		case err == nil && info.IsDir():
			detected = append(detected, spec)
		case isNotExist(err):
			// not installed; skip
		case err != nil:
			return nil, fmt.Errorf("stat %s: %w", probe, err)
		}
	}

	return detected, nil
}

// firstModuleLineMatches reports whether the first non-blank, non-comment
// `module X` directive in goModData names want.
func firstModuleLineMatches(goModData []byte, want string) bool {
	for line := range strings.SplitSeq(string(goModData), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		const prefix = "module "
		if !strings.HasPrefix(trimmed, prefix) {
			return false
		}

		name := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))

		if idx := strings.Index(name, "//"); idx >= 0 {
			name = strings.TrimSpace(name[:idx])
		}

		return name == want
	}

	return false
}

// isNotExist reports whether err signals a missing file. Tests inject
// errors that wrap fs.ErrNotExist; production wraps os errors. Using
// errors.Is keeps the package free of *os.PathError checks.
func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// listFilesRecursive returns every file under root, as relative paths.
// Returns ErrSkillsSrcMissing if root does not exist.
func listFilesRecursive(root string, fileSystem Filesystem) ([]string, error) {
	files := make([]string, 0)

	walkErr := walkFilesRecursive(root, "", fileSystem, &files)
	if walkErr != nil {
		return nil, walkErr
	}

	return files, nil
}

func mdFilesIn(entries []DirEntry) []string {
	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		out = append(out, name)
	}

	return out
}

// The field names are PascalCase because that's the literal JSON `go list`
// emits — not the Go-conventional camelCase a linter would prefer.

// planCommandCopies enumerates .md files at the top level of srcCommands
// and produces one CopyOp per harness that has a CommandsTargetRel.
func planCommandCopies(
	srcCommands, home string,
	harnesses []HarnessSpec,
	fileSystem Filesystem,
) ([]CopyOp, error) {
	entries, readErr := fileSystem.ReadDir(srcCommands)
	if readErr != nil {
		if isNotExist(readErr) {
			return nil, nil
		}

		return nil, fmt.Errorf("read commands dir %s: %w", srcCommands, readErr)
	}

	cmdFiles := mdFilesIn(entries)

	ops := make([]CopyOp, 0, len(cmdFiles)*len(harnesses))

	for _, spec := range harnesses {
		if spec.CommandsTargetRel == "" {
			continue
		}

		dstRoot := filepath.Join(home, spec.CommandsTargetRel)

		for _, name := range cmdFiles {
			ops = append(ops, CopyOp{
				Harness:     spec.Name,
				Src:         filepath.Join(srcCommands, name),
				Dst:         filepath.Join(dstRoot, name),
				CommandFile: name,
			})
		}
	}

	return ops, nil
}

// planGuidanceCopies enumerates .md files at the top level of srcGuidance
// and produces one CopyOp per harness that has a non-empty GuidanceTargetRel.
// Mirrors planCommandCopies: flat *.md only, returns nil, nil when srcGuidance
// is absent (guidance is optional — contrast planSkillCopies which errors).
func planGuidanceCopies(
	srcGuidance, home string,
	harnesses []HarnessSpec,
	fileSystem Filesystem,
) ([]CopyOp, error) {
	entries, readErr := fileSystem.ReadDir(srcGuidance)
	if readErr != nil {
		if isNotExist(readErr) {
			return nil, nil
		}

		return nil, fmt.Errorf("read guidance dir %s: %w", srcGuidance, readErr)
	}

	guidanceFiles := mdFilesIn(entries)

	ops := make([]CopyOp, 0, len(guidanceFiles)*len(harnesses))

	for _, spec := range harnesses {
		if spec.GuidanceTargetRel == "" {
			continue
		}

		dstRoot := filepath.Join(home, spec.GuidanceTargetRel)

		for _, name := range guidanceFiles {
			ops = append(ops, CopyOp{
				Harness:      spec.Name,
				Src:          filepath.Join(srcGuidance, name),
				Dst:          filepath.Join(dstRoot, name),
				GuidanceFile: name,
			})
		}
	}

	return ops, nil
}

// planSkillCopies enumerates every file under srcSkills and produces one
// CopyOp per file per harness. Files keep their relative path under the
// harness's SkillsTargetRel.
func planSkillCopies(
	srcSkills, home string,
	harnesses []HarnessSpec,
	fileSystem Filesystem,
) ([]CopyOp, error) {
	files, listErr := listFilesRecursive(srcSkills, fileSystem)
	if listErr != nil {
		return nil, fmt.Errorf("listing skills under %s: %w", srcSkills, listErr)
	}

	ops := make([]CopyOp, 0, len(files)*len(harnesses))

	for _, spec := range harnesses {
		dstRoot := filepath.Join(home, spec.SkillsTargetRel)

		for _, rel := range files {
			ops = append(ops, CopyOp{
				Harness:  spec.Name,
				Src:      filepath.Join(srcSkills, rel),
				Dst:      filepath.Join(dstRoot, rel),
				SkillDir: topLevelDir(rel),
			})
		}
	}

	return ops, nil
}

// resolveBinaryPath returns where `go install` will drop the engram binary.
// Order matches the go toolchain: GOBIN, then GOPATH/bin, then ~/go/bin.
func resolveBinaryPath(home string, env Env) string {
	if gobin := env.Getenv("GOBIN"); gobin != "" {
		return filepath.Join(gobin, "engram")
	}

	if gopath := env.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin", "engram")
	}

	return filepath.Join(home, "go", "bin", "engram")
}

// supportedHarnesses returns the canonical list in install order.
func supportedHarnesses() []HarnessSpec {
	return []HarnessSpec{
		{
			Name:              HarnessClaude,
			ProbeRel:          ".claude",
			SkillsTargetRel:   filepath.Join(".claude", "skills"),
			GuidanceTargetRel: filepath.Join(".claude", "engram"),
		},
		{
			Name:              HarnessOpencode,
			ProbeRel:          filepath.Join(".config", "opencode"),
			SkillsTargetRel:   filepath.Join(".config", "opencode", "skills"),
			CommandsTargetRel: filepath.Join(".config", "opencode", "commands"),
			// OpenCode @import support unverified — GuidanceTargetRel empty until confirmed.
		},
	}
}

// topLevelDir returns the first path segment of rel (skill-dir name).
func topLevelDir(rel string) string {
	first, _, _ := strings.Cut(rel, string(filepath.Separator))

	return first
}

func walkFilesRecursive(dir, rel string, fileSystem Filesystem, files *[]string) error {
	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		if isNotExist(readErr) && rel == "" {
			return fmt.Errorf("%w: %s", ErrSkillsSrcMissing, dir)
		}

		return fmt.Errorf("read %s: %w", dir, readErr)
	}

	for _, entry := range entries {
		childRel := filepath.Join(rel, entry.Name())
		childAbs := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			subErr := walkFilesRecursive(childAbs, childRel, fileSystem, files)
			if subErr != nil {
				return subErr
			}

			continue
		}

		*files = append(*files, childRel)
	}

	return nil
}

// walkUpForModule walks up from start looking for a `go.mod` whose first
// `module` directive equals ModulePath. Returns the directory containing
// that go.mod, found=true on success, or found=false (nil error) if the
// filesystem root is reached without finding a match.
func walkUpForModule(start string, fileSystem Filesystem) (root string, found bool, err error) {
	dir := filepath.Clean(start)

	for {
		modPath := filepath.Join(dir, "go.mod")

		data, readErr := fileSystem.ReadFile(modPath)
		switch {
		case readErr == nil:
			if firstModuleLineMatches(data, ModulePath) {
				return dir, true, nil
			}

			return "", false, nil
		case isNotExist(readErr):
			// keep walking up
		default:
			return "", false, fmt.Errorf("read %s: %w", modPath, readErr)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}

		dir = parent
	}
}
