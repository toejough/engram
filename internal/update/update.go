// Package update implements the `engram update` subcommand: refresh the
// engram binary via `go install` and copy harness skills/commands from
// either a local clone or the module cache into per-harness user dirs.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
	ErrGoNotFound       = errors.New("go binary not found on PATH")
	ErrNoHarness        = errors.New("no supported harness found")
	ErrSkillsSrcMissing = errors.New("skills source dir missing")
)

// Commander runs an external command, capturing stdout and stderr.
type Commander interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// CopyOp describes a single source→target file copy planned for a harness.
type CopyOp struct {
	Harness Harness
	Src     string
	Dst     string
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
}

// Harness names a supported agent harness. The zero value is invalid.
type Harness string

// HarnessReport summarizes one harness install attempt.
type HarnessReport struct {
	Name         Harness
	SkillsRoot   string
	CommandsRoot string
	SkillFiles   int
	CommandFiles int
	Err          error
}

// HarnessSpec captures one harness's well-known paths (relative to home).
type HarnessSpec struct {
	Name              Harness
	ProbeRel          string // dir to stat under home (e.g. ".claude")
	SkillsTargetRel   string // skills install dir under home
	CommandsTargetRel string // commands install dir under home (empty: skip commands)
}

// Options controls one Run invocation.
type Options struct {
	DryRun bool
}

// Report is the final outcome of Updater.Run, suitable for formatting.
type Report struct {
	DryRun    bool
	Source    SourceInfo
	GoInstall string // command line invoked (or planned)
	Harnesses []HarnessReport
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
	report := Report{DryRun: opts.DryRun}

	home, homeErr := u.Env.UserHomeDir()
	if homeErr != nil {
		return report, fmt.Errorf("resolving home: %w", homeErr)
	}

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

	srcSkills := filepath.Join(source.Root, "skills")
	srcCommands := filepath.Join(source.Root, "opencode", "commands")

	skillOps, planErr := planSkillCopies(srcSkills, home, harnesses, u.FS)
	if planErr != nil {
		return report, planErr
	}

	cmdOps, cmdPlanErr := planCommandCopies(srcCommands, home, harnesses, u.FS)
	if cmdPlanErr != nil {
		return report, cmdPlanErr
	}

	report.Harnesses = u.applyOps(harnesses, home, skillOps, cmdOps, opts.DryRun)

	return report, nil
}

func (u *Updater) applyForHarness(
	rep *HarnessReport,
	name Harness,
	skillOps, cmdOps []CopyOp,
	dryRun bool,
) {
	for _, copyOp := range skillOps {
		if copyOp.Harness != name {
			continue
		}

		applyErr := u.applyOne(copyOp, dryRun)
		if applyErr != nil {
			rep.Err = applyErr

			return
		}

		rep.SkillFiles++
	}

	for _, copyOp := range cmdOps {
		if copyOp.Harness != name {
			continue
		}

		applyErr := u.applyOne(copyOp, dryRun)
		if applyErr != nil {
			rep.Err = applyErr

			return
		}

		rep.CommandFiles++
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
// for deciding the exit code (any-success → 0, all-fail → 1).
func (u *Updater) applyOps(
	harnesses []HarnessSpec,
	home string,
	skillOps, cmdOps []CopyOp,
	dryRun bool,
) []HarnessReport {
	reports := make([]HarnessReport, 0, len(harnesses))

	for _, spec := range harnesses {
		rep := HarnessReport{
			Name:         spec.Name,
			SkillsRoot:   filepath.Join(home, spec.SkillsTargetRel),
			CommandsRoot: cmdRootFor(spec, home),
		}

		u.applyForHarness(&rep, spec.Name, skillOps, cmdOps, dryRun)
		reports = append(reports, rep)
	}

	return reports
}

// resolveRemoteModuleRoot calls `go list -m -json` to find the module
// cache directory where the @latest version is unpacked.
func (u *Updater) resolveRemoteModuleRoot(ctx context.Context) (string, string, error) {
	stdout, _, runErr := u.Cmd.Run(ctx, "go", "list", "-m", "-json", ModulePath+"@latest")
	if runErr != nil {
		return "", "", fmt.Errorf("go list module: %w", runErr)
	}

	dir, version, parseErr := parseGoListJSON(stdout)
	if parseErr != nil {
		return "", "", fmt.Errorf("parsing go list output: %w", parseErr)
	}

	_, statErr := u.FS.Stat(dir)
	if statErr != nil {
		return "", "", fmt.Errorf("module cache miss at %s: %w", dir, statErr)
	}

	return dir, version, nil
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
			_, _, runErr := u.Cmd.Run(ctx, "go", "install", "./cmd/engram/")
			if runErr != nil {
				return SourceInfo{}, fmt.Errorf("go install (local): %w", runErr)
			}
		}

		return SourceInfo{Mode: SourceLocal, Root: root}, nil
	}

	if !dryRun {
		_, _, runErr := u.Cmd.Run(ctx, "go", "install", goInstallTarget)
		if runErr != nil {
			return SourceInfo{}, fmt.Errorf("go install (remote): %w", runErr)
		}
	}

	modRoot, version, modErr := u.resolveRemoteModuleRoot(ctx)
	if modErr != nil {
		return SourceInfo{}, modErr
	}

	return SourceInfo{Mode: SourceRemote, Root: modRoot, Version: version}, nil
}

// unexported constants.
const (
	// dirPerm is the mode used when creating any harness target dir.
	dirPerm fs.FileMode = 0o755
	// filePerm is the mode used when writing any copied file.
	filePerm              fs.FileMode = 0o644
	goInstallTarget                   = ModulePath + "/cmd/engram@latest"
	maxSupportedHarnesses             = 2
)

func cmdRootFor(spec HarnessSpec, home string) string {
	if spec.CommandsTargetRel == "" {
		return ""
	}

	return filepath.Join(home, spec.CommandsTargetRel)
}

func describeGoInstall(source SourceInfo) string {
	if source.Mode == SourceLocal {
		return "go install ./cmd/engram/"
	}

	return "go install " + goInstallTarget
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

// parseGoListJSON extracts Dir and Version from `go list -m -json` output.
// The field names are PascalCase because that's the literal JSON `go list`
// emits — not the Go-conventional camelCase a linter would prefer.
func parseGoListJSON(data []byte) (dir string, version string, err error) {
	var payload struct {
		Dir     string `json:"Dir"`     //nolint:tagliatelle // `go list -m -json` emits PascalCase
		Version string `json:"Version"` //nolint:tagliatelle // `go list -m -json` emits PascalCase
	}

	jsonErr := json.Unmarshal(data, &payload)
	if jsonErr != nil {
		return "", "", fmt.Errorf("unmarshal: %w", jsonErr)
	}

	if payload.Dir == "" {
		return "", "", fmt.Errorf("%w: empty Dir field", ErrSkillsSrcMissing)
	}

	return payload.Dir, payload.Version, nil
}

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
				Harness: spec.Name,
				Src:     filepath.Join(srcCommands, name),
				Dst:     filepath.Join(dstRoot, name),
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
				Harness: spec.Name,
				Src:     filepath.Join(srcSkills, rel),
				Dst:     filepath.Join(dstRoot, rel),
			})
		}
	}

	return ops, nil
}

// supportedHarnesses returns the canonical list in install order.
func supportedHarnesses() []HarnessSpec {
	return []HarnessSpec{
		{
			Name:            HarnessClaude,
			ProbeRel:        ".claude",
			SkillsTargetRel: filepath.Join(".claude", "skills"),
		},
		{
			Name:              HarnessOpencode,
			ProbeRel:          filepath.Join(".config", "opencode"),
			SkillsTargetRel:   filepath.Join(".config", "opencode", "skills"),
			CommandsTargetRel: filepath.Join(".config", "opencode", "commands"),
		},
	}
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
