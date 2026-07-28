// Package update implements the `engram update` subcommand: refresh the
// engram binary via `go install` and copy harness skills/commands from
// agent-instructions/ in either a local clone or the module cache into
// per-harness user dirs.
package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Exported constants.
const (
	HarnessClaude   Harness = "Claude Code"
	HarnessOpencode Harness = "OpenCode"
	HarnessPi       Harness = "Pi"
	ModulePath              = "github.com/toejough/engram"
)

// DeployMode selects how one harness's harness-visible artifact surfaces
// are materialized from the engram-owned root (D7). DeployModeSymlink links
// the surface path directly into the root — real content lives only in the
// root (D1), never as a harness-visible copy. DeployModeManifest copies real
// files and records every written path in a manifest so sync can delete by
// that record instead of by symlink identity. Per verification-verdicts.md's
// symlink-discovery probes (tasks 1.1-1.3: Claude Code, OpenCode, and Pi
// all discovered a skill, command, and/or guidance file through a symlinked
// path), all three currently-supported harnesses run in DeployModeSymlink —
// see supportedHarnesses. The zero value is invalid; every entry in
// supportedHarnesses must set one explicitly.
type DeployMode int

// DeployMode values.
const (
	DeployModeSymlink DeployMode = iota + 1
	DeployModeManifest
)

// EngramSyncOpKind classifies one planEngramRootSync operation.
type EngramSyncOpKind int

// EngramSyncOpKind values.
const (
	EngramSyncCreate EngramSyncOpKind = iota + 1
	EngramSyncDelete
	EngramSyncOverwrite
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
	// ErrCommandNotFound is the Commander contract for "binary not on PATH":
	// implementations translate their platform's not-found error (e.g.
	// exec.ErrNotFound) to this sentinel before returning, keeping this
	// package free of os/exec (#700).
	ErrCommandNotFound = errors.New("command not found")
	// ErrEngramRootNotDir means an engram-owned root path (D1) exists but is
	// a file, not a directory — the sync engine refuses to touch it.
	ErrEngramRootNotDir = errors.New("engram root exists and is not a directory")
	ErrGitNotFound      = errors.New("git binary not found on PATH")
	ErrGoNotFound       = errors.New("go binary not found on PATH")
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

// EngramSyncOp is one planned action against an engram-owned root: create a
// missing intended file, overwrite one whose path already holds content, or
// delete a file present under a managed subtree but absent from the
// intended set (D4). RelPath is relative to the root; AbsPath is root +
// RelPath. Src is the absolute source to read from (Create/Overwrite only).
type EngramSyncOp struct {
	Kind    EngramSyncOpKind
	RelPath string
	AbsPath string
	Src     string
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
	// Mode returns the file mode bits, including the type bits (e.g.
	// fs.ModeSymlink), so callers can distinguish symlink / real file / dir
	// without following the entry (D8).
	Mode() fs.FileMode
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
	// Symlink creates link as a symbolic link to target. Used to
	// materialize the sync engine's engram-owned-root links (D1/D3). Same
	// semantics as os.Symlink: errors if link already exists.
	Symlink(target, link string) error
	// ReadLink returns the destination that the symbolic link at path
	// points to. Used by dangling-link cleanup (D5) to resolve a link's
	// logical target without following it. Same semantics as os.Readlink:
	// errors if path is not a symbolic link.
	ReadLink(path string) (string, error)
	// Lstat returns the FileInfo for path without following a trailing
	// symlink, so callers can distinguish symlink / real file / dir (D8).
	// Same semantics as os.Lstat: errors only on real I/O failure.
	Lstat(path string) (FileInfo, error)
}

// Harness names a supported agent harness. The zero value is invalid.
type Harness string

// HarnessReport summarizes one harness install attempt.
type HarnessReport struct {
	Name      Harness
	ProbeRoot string // home-relative harness root, e.g. ".claude"
	// SkillsRoot and CommandsRoot are pre-resolved ABSOLUTE paths (ready for
	// direct use); the *Rel fields below stay HOME-RELATIVE because the CLI
	// renders them into OS-independent forward-slash "@~/..." import syntax.
	SkillsRoot        string // absolute skills install dir
	CommandsRoot      string // absolute commands install dir (empty if harness has no commands)
	GuidanceTargetRel string // home-relative guidance install dir (empty if harness takes no guidance)
	ImportsFileRel    string // home-relative config file scanned for guidance imports (empty: no import support)
	SkillDirs         []SkillDirCount
	CommandFiles      []string // basenames of .md files copied
	GuidanceFiles     []string // basenames of .md files copied into the guidance dir
	// EngramRoot is the resolved absolute path to this harness's
	// engram-owned root (D1), maintained by the sync engine below.
	EngramRoot string
	// EngramSyncDeleted lists every path (relative to EngramRoot) the sync
	// engine deleted this run (D4).
	EngramSyncDeleted []string
	// EngramUnattributable lists basenames found at the top level of a
	// marker-less engram-owned root that match no intended deploy set
	// basename — reported for manual review, never deleted (D2).
	EngramUnattributable []string
	// EngramDeletionRefused is true when EngramRoot exists without the
	// `.engram-owned` marker: sync-deletion was refused there this run,
	// though creates/overwrites still applied (D2).
	EngramDeletionRefused bool
	// SurfaceUnattributable lists the absolute harness-visible surface paths
	// (skill dirs, command files, guidance files) where a REAL file or
	// directory was found that matches NO intended-set artifact for that
	// surface (D6/task 5.2) — a stray engram cannot prove ownership of,
	// reported for manual review and left completely untouched. Populated
	// only on the first sync for a harness (no marker present at the start
	// of this run): once every intended-set path is a symlink, a stray
	// surviving that conversion is not a new discovery a repeated scan would
	// add value re-detecting. Contrast EngramAdopted: an intended-set real
	// file/dir is adopted, never listed here.
	SurfaceUnattributable []string
	// EngramAdopted lists the absolute harness-visible surface paths (skill
	// dirs, command files, guidance files, and Claude's root-level guidance
	// compat links) where an intended-set REAL file or directory — a
	// pre-sync copy-mode deploy — was replaced by a symlink into the
	// engram-owned root this run (D6/task 5.1). Under DryRun these are the
	// paths that WOULD be adopted; nothing is actually changed.
	EngramAdopted []string
	// DanglingLinksRemoved lists the absolute paths of every harness-surface
	// symlink removed this run because its lexically-resolved target lay
	// inside EngramRoot but no longer existed (D5). Under DryRun these are
	// the links that WOULD be removed; nothing is actually deleted.
	DanglingLinksRemoved []string
	Err                  error
}

// HarnessSpec captures one harness's well-known paths (relative to home).
type HarnessSpec struct {
	Name              Harness
	ProbeRel          string // dir to stat under home (e.g. ".claude")
	SkillsTargetRel   string // skills install dir under home
	CommandsTargetRel string // commands install dir under home (empty: skip commands)
	// GuidanceTargetRel and ImportsFileRel are a coupled pair: both set, or
	// both empty. guidanceImportPrefixes assumes this — a harness with an
	// imports file but no guidance dir would derive a nonsensical "@~//".
	GuidanceTargetRel string // guidance install dir under home (empty: skip guidance)
	ImportsFileRel    string // config file under home scanned for guidance @imports (empty: skip detection)
	// EngramRootRel is the home-relative engram-owned root this harness's
	// sync engine maintains (D1): real skill/command/guidance content lives
	// only here, synced to exactly match the intended deploy set (D4).
	// Harness-visible copies/symlinks materializing FROM it are handled by
	// applySkillLinks/applyCmdLinks/applyGuidanceLinks (symlink mode) or
	// applyForHarnessManifestMode (manifest mode), plus first-sync migration
	// adopting any pre-existing real files at those paths.
	// supportedHarnesses sets this for every currently-supported harness;
	// it must never be left empty (the sync engine trusts it to bound
	// deletion — an empty value would resolve to home itself).
	EngramRootRel string
	// DeployMode selects symlink vs manifest materialization for this
	// harness's surfaces (D7) — see the DeployMode doc comment.
	// supportedHarnesses sets this for every currently-supported harness.
	DeployMode DeployMode
}

// Options controls one Run invocation.
type Options struct {
	DryRun       bool
	WithGuidance bool // deploy agent-instructions/guidance/*.md to the harness guidance dir
}

// Report is the final outcome of Updater.Run, suitable for formatting.
type Report struct {
	DryRun           bool
	WithGuidance     bool   // whether --with-guidance was requested
	Home             string // user home (so the CLI can tildify paths)
	Source           SourceInfo
	GoInstall        string                      // command line invoked (or planned)
	BinaryPath       string                      // resolved install location, e.g. /Users/joe/go/bin/engram
	BinaryVersion    string                      // resolved engram version, empty when unknown
	GuidanceImported bool                        // true when any harness's config file imports ANY engram guidance file
	GuidanceImports  map[Harness]map[string]bool // imported engram-guidance basenames per harness (for per-file hints)
	// VaultHasOldVocabFiles is true when the vault still holds pre-tags
	// vocab.*.md files. Set by the cli package after Run returns (via
	// oldVocabFilesPresent) — Updater.Run itself never touches vault paths;
	// this field is opaque data.
	VaultHasOldVocabFiles bool
	// VaultHasUntaggedVocabDefinitions is true when the vault holds at least
	// one vocab definition note (bare `vocab` tag + "vocab-<term>-definition"
	// slug) missing its vocab/<term> self-tag — a vault minted before the
	// vocab-definition-self-tags change (4f68fada), for which `engram vocab
	// tag-definitions` is the backfill. The family note (slug
	// "vocab-definition") is deliberately bare-vocab-only and never counts.
	// Set by the cli package after Run returns (via
	// vocabDefinitionsMissingSelfTags) — Updater.Run itself never touches
	// vault paths; this field is opaque data.
	VaultHasUntaggedVocabDefinitions bool
	// ChunkIndexHasEmptyFiles is set by the cli package after Run returns
	// (Updater.Run never touches chunk paths; this field is opaque data).
	ChunkIndexHasEmptyFiles bool
	// ChunkIndexHasPrunableDuplicates is set by the cli package after Run
	// returns: true when the chunk-index manifest holds at least one
	// duplicate that `engram prune --duplicates` would actually remove
	// right now (prune's own reconciliation, run in dry-run mode, reports
	// removed > 0). Refusal-only backlogs — every group's canonical
	// missing or not verifiably covering its siblings' records — stay
	// silent (#713): there is no command that resolves them, and
	// prune/ingest explain refusals when run. update deliberately never
	// removes anything on the strength of this field — it only surfaces
	// the notice naming `engram prune --duplicates`, which the user runs
	// explicitly.
	ChunkIndexHasPrunableDuplicates bool

	// VocabRegenRan is true when `engram update --regen-vocab` executed the
	// regen path this run (set by the cli package after Run returns —
	// Updater.Run never touches vault paths; this field is opaque data, same
	// as VaultHasOldVocabFiles above). When true, the CLI report renders the
	// regen summary below INSTEAD of the plain migration notice — a run that
	// just regenerated never repeats the notice it acted on (#712).
	VocabRegenRan bool
	// VocabRegenOldFilesRemoved is the count of old-format vocab.*.md hub
	// files removed (or, under --dry-run, that would be removed).
	VocabRegenOldFilesRemoved int
	// VocabRegenMembersCleaned is the count of member notes whose legacy
	// vocab: frontmatter key / Vocab: body line were stripped (or, under
	// --dry-run, would be stripped).
	VocabRegenMembersCleaned int
	// VocabRegenTermsSeeded is the count of current-format term definition
	// notes harvested from old-format term notes (or, under --dry-run, that
	// would be harvested).
	VocabRegenTermsSeeded int
	// VocabRegenNotesAssigned is the count of vocab-term assignments made by
	// the post-regen re-tag pass. Always 0 under --dry-run, since nothing was
	// actually regenerated to re-tag against.
	VocabRegenNotesAssigned int

	Harnesses []HarnessReport
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

	srcSkills := filepath.Join(source.Root, "agent-instructions", "skills")
	srcCommands := filepath.Join(source.Root, "agent-instructions", "commands")
	srcGuidance := filepath.Join(source.Root, "agent-instructions", "guidance")

	skillOps, planErr := planSkillCopies(srcSkills, home, harnesses, u.FS)
	if planErr != nil {
		return report, planErr
	}

	cmdOps, cmdPlanErr := planCommandCopies(srcCommands, home, harnesses, u.FS)
	if cmdPlanErr != nil {
		return report, cmdPlanErr
	}

	report.GuidanceImports = u.detectGuidanceImports(home, harnesses)
	report.GuidanceImported = len(report.GuidanceImports) > 0

	var guidanceOps []CopyOp

	// Deploy guidance when explicitly requested OR when the user already imports
	// it. This makes --with-guidance a one-time opt-in: once imported, plain
	// `engram update` keeps the guidance current on every run (like skills).
	// guidanceManaged also gates the engram-root sync engine's guidance/
	// subtree (D4): unmanaged means unmanaged, not deleted.
	guidanceManaged := opts.WithGuidance || report.GuidanceImported

	if guidanceManaged {
		var guidancePlanErr error

		guidanceOps, guidancePlanErr = planGuidanceCopies(srcGuidance, home, harnesses, u.FS)
		if guidancePlanErr != nil {
			return report, guidancePlanErr
		}
	}

	report.Harnesses = u.applyOps(harnesses, home, skillOps, cmdOps, guidanceOps, guidanceManaged, opts.DryRun)

	return report, nil
}

// applyCmdLinks materializes one symlink per command file for a
// symlink-mode harness (D3): <CommandsRoot>/<name>.md →
// <EngramRoot>/commands/<name>.md, replacing applyCmdOne's copy. Real
// content lives only under EngramRoot, already synced by
// applyEngramRootSync (called before this in applyForHarnessSymlinkMode). A
// real pre-migration copy at the link path is adopted (task 5.1), never
// merely reported — see materializeOrAdopt.
func (u *Updater) applyCmdLinks(rep *HarnessReport, spec HarnessSpec, cmdOps []CopyOp, dryRun bool) {
	for _, copyOp := range cmdOps {
		if copyOp.Harness != spec.Name {
			continue
		}

		link := filepath.Join(rep.CommandsRoot, copyOp.CommandFile)
		target := filepath.Join(rep.EngramRoot, "commands", copyOp.CommandFile)

		adopted, linkErr := materializeOrAdopt(u.FS, link, target, dryRun)
		if linkErr != nil {
			rep.Err = linkErr

			return
		}

		if adopted {
			rep.EngramAdopted = append(rep.EngramAdopted, link)
		}

		rep.CommandFiles = append(rep.CommandFiles, copyOp.CommandFile)
	}
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

// applyEngramRootOps writes the marker (when writeMarker — a freshly-created
// root, D2) before applying ops, so a mid-plan failure still leaves the root
// correctly claimed for the next run.
func (u *Updater) applyEngramRootOps(root string, writeMarker bool, ops []EngramSyncOp) error {
	if writeMarker {
		mkErr := u.FS.MkdirAll(root, dirPerm)
		if mkErr != nil {
			return fmt.Errorf("mkdir %s: %w", root, mkErr)
		}

		markerErr := u.FS.WriteFile(filepath.Join(root, engramMarkerFile), []byte{}, filePerm)
		if markerErr != nil {
			return fmt.Errorf("write marker %s: %w", root, markerErr)
		}
	}

	return applyEngramSyncOps(u.FS, ops)
}

// applyEngramRootSync maintains spec's engram-owned root (D1): it re-roots
// this run's skill/command/guidance CopyOps under the root's skills/,
// commands/, guidance/ subtrees, resolves ownership from the `.engram-owned`
// marker (D2), plans the sync (D4), plans which managed-subtree directories
// the deletions would leave empty (pruneEmptyDirs — D4 continued: a
// directory holding no intended file is as "present-but-unintended" as a
// file, so it is deleted too, never left as a hollow shell a stale
// harness-visible symlink could still resolve through), and — unless
// dryRun — applies both. Planning always runs (even under dryRun) so the
// report reflects what WOULD happen; applyEngramRootOps/applyPrunedDirs are
// the only steps that write, and pruning is applied AFTER the file-level ops
// so the directories really are empty by then. This runs before
// cleanupDanglingLinks in applyForHarnessSymlinkMode/applyForHarnessManifestMode,
// so a symlink whose target directory this call just emptied-and-removed is
// already gone by the time the dangling-link scan runs, in the SAME update
// invocation (spec: "Removed source artifact disappears on next update").
// Returns isFirstSync (D6): true when root held no marker at the START of
// this run (whether root was absent, or pre-existing and marker-less) — the
// D6 adoption pass runs and the marker is stamped this run (unless dryRun),
// but sync-deletion (and pruning, gated the same way) of managed-subtree
// drift is refused THIS run regardless (D2: "no sync-deletion occurs in
// that directory" when it started unmarked) and only applies on a genuinely
// subsequent sync. Callers use isFirstSync to gate the surface stray-scan
// (task 5.2), which is only worth its ReadDir cost once, before every
// intended-set path becomes a symlink.
func (u *Updater) applyEngramRootSync(
	rep *HarnessReport,
	spec HarnessSpec,
	home string,
	skillOps, cmdOps, guidanceOps []CopyOp,
	guidanceManaged, dryRun bool,
) (isFirstSync bool) {
	root := filepath.Join(home, spec.EngramRootRel)
	rep.EngramRoot = root

	intended := intendedRootFiles(spec, home, skillOps, cmdOps, guidanceOps)

	ownership, ownErr := resolveEngramRootOwnership(root, intended, u.FS)
	if ownErr != nil {
		rep.Err = fmt.Errorf("engram root %s: %w", root, ownErr)

		return false
	}

	rep.EngramDeletionRefused = ownership.deletionRefused
	rep.EngramUnattributable = ownership.unattributable

	ops, planErr := planEngramRootSync(root, intended, guidanceManaged, ownership.allowDelete, u.FS)
	if planErr != nil {
		rep.Err = fmt.Errorf("engram root %s: %w", root, planErr)

		return ownership.writeMarker
	}

	rep.EngramSyncDeleted = deletedRelPaths(ops)

	prunedDirs, pruneErr := pruneEmptyDirs(root, intended, guidanceManaged, ownership.allowDelete, u.FS)
	if pruneErr != nil {
		rep.Err = fmt.Errorf("engram root %s: %w", root, pruneErr)

		return ownership.writeMarker
	}

	rep.EngramSyncDeleted = append(rep.EngramSyncDeleted, prunedDirs...)

	if dryRun {
		return ownership.writeMarker
	}

	applyErr := u.applyEngramRootOps(root, ownership.writeMarker, ops)
	if applyErr != nil {
		rep.Err = applyErr

		return ownership.writeMarker
	}

	pruneApplyErr := applyPrunedDirs(u.FS, root, prunedDirs)
	if pruneApplyErr != nil {
		rep.Err = fmt.Errorf("engram root %s: %w", root, pruneApplyErr)
	}

	return ownership.writeMarker
}

// applyForHarness dispatches to the D7 deploy-mode-specific materialization.
// DeployModeSymlink harnesses get D3/D5 symlink materialization + cleanup;
// DeployModeManifest harnesses get copy-based materialization with manifest
// recording and deletion (#6.1).
func (u *Updater) applyForHarness(
	rep *HarnessReport,
	spec HarnessSpec,
	home string,
	skillOps, cmdOps, guidanceOps []CopyOp,
	guidanceManaged, dryRun bool,
) {
	if spec.DeployMode == DeployModeSymlink {
		u.applyForHarnessSymlinkMode(rep, spec, home, skillOps, cmdOps, guidanceOps, guidanceManaged, dryRun)

		return
	}

	u.applyForHarnessManifestMode(rep, spec, home, skillOps, cmdOps, guidanceOps, guidanceManaged, dryRun)
}

// applyForHarnessManifestMode implements copy-based materialization for
// manifest-mode harnesses: every skill file, command file, and guidance file
// is copied directly into the harness's surface paths, then paths are
// recorded in a manifest and obsolete recorded paths are deleted (#6.1).
func (u *Updater) applyForHarnessManifestMode(
	rep *HarnessReport,
	spec HarnessSpec,
	home string,
	skillOps, cmdOps, guidanceOps []CopyOp,
	guidanceManaged, dryRun bool,
) {
	name := spec.Name

	u.applySkillOps(rep, name, skillOps, dryRun)

	if rep.Err != nil {
		return
	}

	u.applyCmdOps(rep, name, cmdOps, dryRun)

	if rep.Err != nil {
		return
	}

	u.applyGuidanceOps(rep, name, guidanceOps, dryRun)

	if rep.Err != nil {
		return
	}

	// Record written paths in manifest and delete obsolete entries.
	u.applyManifestRecordingAndDeletion(rep, home, spec, name, skillOps, cmdOps, guidanceOps, dryRun)

	if rep.Err != nil {
		return
	}

	u.applyEngramRootSync(rep, spec, home, skillOps, cmdOps, guidanceOps, guidanceManaged, dryRun)
}

// applyForHarnessSymlinkMode implements D3/D5/D6 for a symlink-mode harness:
// the engram-owned root sync (D1/D4) runs FIRST so real content exists
// under the root before any surface symlink is created against it, then
// skill/command surfaces are materialized as symlinks (never copies) —
// adopting any real pre-migration copy found there (task 5.1) — then, on
// the first sync only, every surface dir is scanned for real strays outside
// the intended set (task 5.2), and finally every run scans the harness's
// surface dirs for dangling engram links and removes them (D5). Guidance is
// the D1 exception: Claude Code's GuidanceTargetRel equals its EngramRootRel
// by design, so it gets root-level compat symlinks instead of a separate
// surface dir (task 5.3, applyGuidanceCompatLinks) — this RETIRES the old
// flat-copy path (applyGuidanceOps) for such a harness. Only a harness whose
// GuidanceTargetRel is a genuinely separate surface (Pi today) gets
// guidance-file symlinks via applyGuidanceLinks.
func (u *Updater) applyForHarnessSymlinkMode(
	rep *HarnessReport,
	spec HarnessSpec,
	home string,
	skillOps, cmdOps, guidanceOps []CopyOp,
	guidanceManaged, dryRun bool,
) {
	isFirstSync := u.applyEngramRootSync(rep, spec, home, skillOps, cmdOps, guidanceOps, guidanceManaged, dryRun)

	if rep.Err != nil {
		return
	}

	u.applySkillLinks(rep, spec, skillOps, dryRun)

	if rep.Err != nil {
		return
	}

	u.applyCmdLinks(rep, spec, cmdOps, dryRun)

	if rep.Err != nil {
		return
	}

	if spec.GuidanceTargetRel != "" && spec.GuidanceTargetRel != spec.EngramRootRel {
		u.applyGuidanceLinks(rep, spec, home, guidanceOps, dryRun)
	} else {
		u.applyGuidanceCompatLinks(rep, spec.Name, guidanceOps, dryRun)
	}

	if rep.Err != nil {
		return
	}

	if isFirstSync {
		strayErr := u.reportSurfaceStrays(rep, spec, home, skillOps, cmdOps, guidanceOps)
		if strayErr != nil {
			rep.Err = strayErr

			return
		}
	}

	u.cleanupDanglingLinks(rep, spec, home, dryRun)
}

// applyGuidanceCompatLinks materializes one COMPAT symlink per guidance file
// at the TOP LEVEL of a symlink-mode harness's engram-owned root, for a
// harness whose GuidanceTargetRel equals its EngramRootRel (Claude today —
// D1's guidance caveat): <EngramRoot>/<f>.md -> <EngramRoot>/guidance/<f>.md.
// User-authored @import lines reference the flat path verbatim (D1), so
// this RETIRES the old flat-copy path (applyGuidanceOps) for such a
// harness — canonical content now lives only under guidance/, already
// synced by applyEngramRootSync before this call. A pre-existing REAL flat
// guidance file (today's deployed state) is adopted (task 5.1): its content
// is discarded in favor of the root's already-synced guidance/ copy, since
// the repo is the source of truth (D6, no content comparison needed). The
// compat links are part of the intended set going forward: repointed here
// when wrong, and deleted by cleanupDanglingLinks (D5, extended to scan the
// root itself for this harness shape) once their guidance file leaves the
// intended set.
func (u *Updater) applyGuidanceCompatLinks(rep *HarnessReport, name Harness, guidanceOps []CopyOp, dryRun bool) {
	for _, copyOp := range guidanceOps {
		if copyOp.Harness != name {
			continue
		}

		link := filepath.Join(rep.EngramRoot, copyOp.GuidanceFile)
		target := filepath.Join(rep.EngramRoot, "guidance", copyOp.GuidanceFile)

		adopted, linkErr := materializeOrAdopt(u.FS, link, target, dryRun)
		if linkErr != nil {
			rep.Err = linkErr

			return
		}

		if adopted {
			rep.EngramAdopted = append(rep.EngramAdopted, link)
		}

		rep.GuidanceFiles = append(rep.GuidanceFiles, copyOp.GuidanceFile)
	}
}

// applyGuidanceLinks materializes one symlink per guidance file for a
// symlink-mode harness whose GuidanceTargetRel is a surface distinct from
// EngramRootRel (Pi today — see applyForHarnessSymlinkMode's D1 guard):
// <home>/<GuidanceTargetRel>/<f>.md → <EngramRoot>/guidance/<f>.md. A real
// pre-migration copy at the link path is adopted (task 5.1), never merely
// reported — see materializeOrAdopt.
func (u *Updater) applyGuidanceLinks(
	rep *HarnessReport,
	spec HarnessSpec,
	home string,
	guidanceOps []CopyOp,
	dryRun bool,
) {
	guidanceDir := filepath.Join(home, spec.GuidanceTargetRel)

	for _, copyOp := range guidanceOps {
		if copyOp.Harness != spec.Name {
			continue
		}

		link := filepath.Join(guidanceDir, copyOp.GuidanceFile)
		target := filepath.Join(rep.EngramRoot, "guidance", copyOp.GuidanceFile)

		adopted, linkErr := materializeOrAdopt(u.FS, link, target, dryRun)
		if linkErr != nil {
			rep.Err = linkErr

			return
		}

		if adopted {
			rep.EngramAdopted = append(rep.EngramAdopted, link)
		}

		rep.GuidanceFiles = append(rep.GuidanceFiles, copyOp.GuidanceFile)
	}
}

// applyGuidanceOps flat-copies guidance files directly to their harness
// surface path. Used by manifest-mode harnesses' fallthrough
// (applyForHarnessManifestMode) — symlink-mode harnesses use
// applyGuidanceLinks or applyGuidanceCompatLinks instead (D3/task 5.3).
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

// applyManifestRecordingAndDeletion records written paths in manifest and
// deletes obsolete entries for manifest-mode harnesses (#6.1).
func (u *Updater) applyManifestRecordingAndDeletion(
	rep *HarnessReport,
	home string,
	spec HarnessSpec,
	name Harness,
	skillOps, cmdOps, guidanceOps []CopyOp,
	dryRun bool,
) {
	intendedPaths := collectWrittenPaths(skillOps, cmdOps, guidanceOps, name)
	engramRoot := filepath.Join(home, spec.EngramRootRel)
	manifestPath := filepath.Join(engramRoot, manifestFilename)

	oldPaths, readErr := loadManifestPaths(u.FS, manifestPath)
	if readErr != nil {
		rep.Err = readErr
		return
	}

	deletedRel := calculateDeletedRelPaths(engramRoot, intendedPaths, oldPaths)
	rep.EngramSyncDeleted = append(rep.EngramSyncDeleted, deletedRel...)

	if !dryRun {
		err := applyManifestModeDeletion(u.FS, engramRoot, intendedPaths, dryRun)
		if err != nil {
			rep.Err = err
		}
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
	guidanceManaged, dryRun bool,
) []HarnessReport {
	reports := make([]HarnessReport, 0, len(harnesses))

	for _, spec := range harnesses {
		rep := HarnessReport{
			Name:              spec.Name,
			ProbeRoot:         spec.ProbeRel,
			SkillsRoot:        filepath.Join(home, spec.SkillsTargetRel),
			CommandsRoot:      cmdRootFor(spec, home),
			GuidanceTargetRel: spec.GuidanceTargetRel,
			ImportsFileRel:    spec.ImportsFileRel,
		}

		u.applyForHarness(&rep, spec, home, skillOps, cmdOps, guidanceOps, guidanceManaged, dryRun)
		reports = append(reports, rep)
	}

	return reports
}

// applySkillLinks materializes one symlink per skill directory for a
// symlink-mode harness (D3): <SkillsRoot>/<skill> →
// <EngramRoot>/skills/<skill>, replacing the per-file copy flow
// (applySkillOps/clearSkillDirOnce). Real skill content lives only under
// EngramRoot, already synced by applyEngramRootSync. rep.SkillDirs is still
// populated with per-skill file counts (D3's report-coherence requirement:
// the listing's meaning shifts from "copied" to "linked", but the shape and
// the file count stay meaningful) even though a single symlink op now
// replaces the old per-file copy loop. A real pre-migration copy at the
// link path is adopted (task 5.1), never merely reported — see
// materializeOrAdopt.
func (u *Updater) applySkillLinks(rep *HarnessReport, spec HarnessSpec, skillOps []CopyOp, dryRun bool) {
	skillCounts := map[string]int{}
	skillOrder := make([]string, 0)

	for _, copyOp := range skillOps {
		if copyOp.Harness != spec.Name {
			continue
		}

		if _, seen := skillCounts[copyOp.SkillDir]; !seen {
			skillOrder = append(skillOrder, copyOp.SkillDir)
		}

		skillCounts[copyOp.SkillDir]++
	}

	rep.SkillDirs = collectSkillDirs(skillOrder, skillCounts)

	for _, name := range skillOrder {
		link := filepath.Join(rep.SkillsRoot, name)
		target := filepath.Join(rep.EngramRoot, "skills", name)

		adopted, linkErr := materializeOrAdopt(u.FS, link, target, dryRun)
		if linkErr != nil {
			rep.Err = linkErr

			return
		}

		if adopted {
			rep.EngramAdopted = append(rep.EngramAdopted, link)
		}
	}
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

// cleanupDanglingLinks runs the D5 dangling-link scan over every one of a
// symlink-mode harness's surface dirs: the skills root, the commands root
// (when the harness has one), and either the guidance dir (Pi today, a
// surface distinct from EngramRootRel) or the engram root itself (Claude
// today, GuidanceTargetRel == EngramRootRel — task 5.3's root-level compat
// links live at the root's top level, so scanning the root here is what
// finds and removes a compat link whose guidance file left the intended set;
// the root's real top-level content — the marker, skills/, commands/,
// guidance/ — is never a symlink, so cleanupDanglingLinksInDir's Lstat-mode
// check skips it harmlessly).
func (u *Updater) cleanupDanglingLinks(rep *HarnessReport, spec HarnessSpec, home string, dryRun bool) {
	dirs := make([]string, 0, maxCleanupSurfaceDirs)
	dirs = append(dirs, rep.SkillsRoot)

	if rep.CommandsRoot != "" {
		dirs = append(dirs, rep.CommandsRoot)
	}

	switch {
	case spec.GuidanceTargetRel == "":
		// no guidance surface for this harness (OpenCode) — nothing to add
	case spec.GuidanceTargetRel != spec.EngramRootRel:
		dirs = append(dirs, filepath.Join(home, spec.GuidanceTargetRel)) // Pi: separate guidance dir
	default:
		dirs = append(dirs, rep.EngramRoot) // Claude: compat links live at the root's top level
	}

	for _, dir := range dirs {
		removed, cleanupErr := cleanupDanglingLinksInDir(u.FS, dir, rep.EngramRoot, dryRun)
		if cleanupErr != nil {
			rep.Err = cleanupErr

			return
		}

		rep.DanglingLinksRemoved = append(rep.DanglingLinksRemoved, removed...)
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

// detectGuidanceImports scans each detected harness's config file (the
// spec's ImportsFileRel) for active @import lines pointing at that harness's
// own guidance dir, keyed by harness. Harnesses without an imports file
// (empty ImportsFileRel, e.g. OpenCode) are skipped explicitly, as are
// harnesses whose config file is missing or holds no imports — only
// harnesses with at least one import get an entry.
func (u *Updater) detectGuidanceImports(home string, harnesses []HarnessSpec) map[Harness]map[string]bool {
	imports := map[Harness]map[string]bool{}

	for _, spec := range harnesses {
		if spec.ImportsFileRel == "" {
			continue // harness config cannot @import guidance — detection skipped
		}

		data, readErr := u.FS.ReadFile(filepath.Join(home, spec.ImportsFileRel))
		if readErr != nil {
			continue // missing config file: nothing imported, not an error
		}

		tildePrefix, expandedPrefix := guidanceImportPrefixes(spec, home)

		if found := collectGuidanceImports(data, tildePrefix, expandedPrefix); len(found) > 0 {
			imports[spec.Name] = found
		}
	}

	return imports
}

// reportSurfaceStrays runs the D6/task-5.2 stray scan across every harness
// surface dir this harness materializes into — the skills root, the
// commands root (when present), and the guidance dir ONLY when it is a
// surface distinct from EngramRootRel (Pi today; Claude's compat-link root
// has no separate guidance surface to scan here — a stray flat file there
// is caught by the root-level unattributableRootFiles scan instead, D1's
// guidance caveat). Callers gate this to the first sync only (isFirstSync):
// once every intended-set path is a symlink, repeating this ReadDir/Lstat
// scan every run would not surface anything new.
func (u *Updater) reportSurfaceStrays(
	rep *HarnessReport,
	spec HarnessSpec,
	home string,
	skillOps, cmdOps, guidanceOps []CopyOp,
) error {
	scans := []surfaceScan{{dir: rep.SkillsRoot, intended: intendedSkillDirNames(spec.Name, skillOps)}}

	if rep.CommandsRoot != "" {
		scans = append(scans, surfaceScan{dir: rep.CommandsRoot, intended: intendedCommandBasenames(spec.Name, cmdOps)})
	}

	if spec.GuidanceTargetRel != "" && spec.GuidanceTargetRel != spec.EngramRootRel {
		scans = append(scans, surfaceScan{
			dir:      filepath.Join(home, spec.GuidanceTargetRel),
			intended: intendedGuidanceBasenames(spec.Name, guidanceOps),
		})
	}

	for _, scan := range scans {
		strays, strayErr := surfaceStrays(u.FS, scan.dir, scan.intended)
		if strayErr != nil {
			return strayErr
		}

		rep.SurfaceUnattributable = append(rep.SurfaceUnattributable, strays...)
	}

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
		if errors.Is(cloneErr, ErrCommandNotFound) {
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
	// engramMarkerFile is the ownership marker written at the top level of
	// every engram-owned root (D2): its presence is what allows sync-deletion.
	engramMarkerFile = ".engram-owned"
	// filePerm is the mode used when writing any copied file.
	filePerm fs.FileMode = 0o644
	// lfsPointerPrefix is the first line of every Git-LFS pointer file.
	lfsPointerPrefix = "version https://git-lfs"
	// manifestFilename is the name of the manifest file inside the engram-owned
	// root for manifest-mode harnesses (D7). Every path written to the harness's
	// surface directories is recorded here; on next run, recorded paths whose
	// source artifacts no longer exist are deleted, and the manifest is updated.
	// Manifest mode provides the fallback sync semantics for harnesses whose
	// symlink discovery fails verification.
	manifestFilename = ".engram-manifest.json"
	// maxCleanupSurfaceDirs bounds the surface-dir slice cleanupDanglingLinks
	// builds: skills root, commands root, and one of {guidance dir (Pi), the
	// engram root itself (Claude's root-level compat links, task 5.3} — no
	// currently-supported harness has both a commands root AND a
	// GuidanceTargetRel == EngramRootRel shape, so 3 stays the true bound.
	maxCleanupSurfaceDirs = 3
	maxSupportedHarnesses = 3
	// modelMinBytes: the real MiniLM ONNX is ~90 MB; anything under a
	// megabyte is certainly not it.
	modelMinBytes = 1 << 20
	repoCloneURL  = "https://" + ModulePath + ".git"
)

// engramRootOwnership is the marker-driven verdict for one sync run against
// an engram-owned root (D2/D6): whether this run may sync-delete inside it,
// whether this run must write the marker (root absent, OR root pre-existing
// and marker-less — the D6 adoption pass, task 5.1), and — for a
// pre-existing marker-less root — the top-level basenames that don't match
// any intended-set basename (reported, never deleted; D2/D6's Claude
// flat-guidance interplay excludes basename-matched files from this list).
// A pre-existing marker-less root ALWAYS gets writeMarker=true (adoption
// runs) but allowDelete stays false for THIS run regardless — D2's "no
// sync-deletion occurs in that directory" is evaluated against whether the
// root held the marker at the START of the run; a genuinely subsequent sync
// (marker already present) is what enables normal drift-deletion.
type engramRootOwnership struct {
	allowDelete     bool
	writeMarker     bool
	deletionRefused bool
	unattributable  []string
}

// intendedRootFile pairs an engram-root-relative path with the absolute
// source it should be synced from. Built from planSkillCopies /
// planCommandCopies / planGuidanceCopies output, re-rooted under the
// engram-owned root's skills/, commands/, guidance/ subtrees (D1/D4).
type intendedRootFile struct {
	RelPath string // path relative to the engram-owned root
	Src     string // absolute source path
}

// surfaceScan pairs one harness surface dir with the basename set that is
// NOT a stray there — reportSurfaceStrays' (task 5.2) per-dir input.
type surfaceScan struct {
	dir      string
	intended map[string]bool
}

// applyEngramSyncOp applies a single engram-root sync op: a create/overwrite
// reads Src and writes AbsPath (creating parent dirs as needed); a delete
// RemoveAlls AbsPath.
func applyEngramSyncOp(fileSystem Filesystem, syncOp EngramSyncOp) error {
	if syncOp.Kind == EngramSyncDelete {
		removeErr := fileSystem.RemoveAll(syncOp.AbsPath)
		if removeErr != nil {
			return fmt.Errorf("delete %s: %w", syncOp.AbsPath, removeErr)
		}

		return nil
	}

	data, readErr := fileSystem.ReadFile(syncOp.Src)
	if readErr != nil {
		return fmt.Errorf("read %s: %w", syncOp.Src, readErr)
	}

	mkErr := fileSystem.MkdirAll(filepath.Dir(syncOp.AbsPath), dirPerm)
	if mkErr != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(syncOp.AbsPath), mkErr)
	}

	writeErr := fileSystem.WriteFile(syncOp.AbsPath, data, filePerm)
	if writeErr != nil {
		return fmt.Errorf("write %s: %w", syncOp.AbsPath, writeErr)
	}

	return nil
}

// applyEngramSyncOps executes an engram-root sync op plan in order.
func applyEngramSyncOps(fileSystem Filesystem, ops []EngramSyncOp) error {
	for _, syncOp := range ops {
		opErr := applyEngramSyncOp(fileSystem, syncOp)
		if opErr != nil {
			return opErr
		}
	}

	return nil
}

// applyManifestDeletion deletes recorded paths and updates the manifest.
func applyManifestDeletion(
	fileSystem Filesystem,
	manifestPath string,
	toDelete, updated []string,
) error {
	for _, path := range toDelete {
		err := fileSystem.RemoveAll(path)
		if err != nil {
			return fmt.Errorf("deleting %s: %w", path, err)
		}
	}

	// Update manifest with remaining paths.
	sort.Strings(updated)

	updatedJSON := []byte("[]")

	if len(updated) > 0 {
		data, err := json.Marshal(updated)
		if err != nil {
			return fmt.Errorf("marshaling updated manifest: %w", err)
		}

		updatedJSON = data
	}

	err := fileSystem.WriteFile(manifestPath, updatedJSON, filePerm)
	if err != nil {
		return fmt.Errorf("writing manifest %s: %w", manifestPath, err)
	}

	return nil
}

// applyManifestModeDeletion implements manifest-driven deletion for
// manifest-mode harnesses (D7, #6.1 task): load the prior manifest, delete
// recorded paths whose source artifacts are no longer in the intended set,
// and update the manifest. Unrecorded files are never deleted. In dry-run
// mode, deletion and manifest writes are previewed but not executed.
func applyManifestModeDeletion(
	fileSystem Filesystem,
	engramRoot string,
	intended []string,
	dryRun bool,
) error {
	// Load prior manifest; missing manifest treated as empty (no error).
	manifestPath := filepath.Join(engramRoot, manifestFilename)
	manifestData, readErr := fileSystem.ReadFile(manifestPath)

	var recorded []string

	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("reading manifest %s: %w", manifestPath, readErr)
	}

	if readErr == nil {
		err := json.Unmarshal(manifestData, &recorded)
		if err != nil {
			return fmt.Errorf("unmarshaling manifest: %w", err)
		}
	}

	// Build intent set for O(1) lookup.
	intendedSet := make(map[string]bool)
	for _, path := range intended {
		intendedSet[path] = true
	}

	// Determine which recorded paths to delete: those not in intended set.
	var toDelete []string

	for _, path := range recorded {
		if !intendedSet[path] {
			toDelete = append(toDelete, path)
		}
	}

	// Delete recorded paths whose source is gone (not in dry-run).
	if dryRun {
		return nil
	}

	return applyManifestDeletion(fileSystem, manifestPath, toDelete, intended)
}

// applyPrunedDirs removes every directory pruneEmptyDirs planned (RelPath
// relative to root), each already confirmed empty of surviving content by
// planning. Called AFTER applyEngramSyncOps, so a pruned directory's last
// file is already gone by the time this RemoveAll runs; RemoveAll's
// already-gone-is-fine semantics also make call order harmless if it
// weren't. Never called under dryRun.
func applyPrunedDirs(fileSystem Filesystem, root string, prunedRel []string) error {
	for _, rel := range prunedRel {
		absPath := filepath.Join(root, rel)

		removeErr := fileSystem.RemoveAll(absPath)
		if removeErr != nil {
			return fmt.Errorf("prune %s: %w", absPath, removeErr)
		}
	}

	return nil
}

// calculateDeletedRelPaths determines which old paths are not in intended set
// and converts them to relative paths for reporting.
func calculateDeletedRelPaths(engramRoot string, intended, old []string) []string {
	intendedSet := make(map[string]bool)
	for _, path := range intended {
		intendedSet[path] = true
	}

	var deletedRel []string

	for _, path := range old {
		if !intendedSet[path] {
			rel, err := filepath.Rel(engramRoot, path)
			if err == nil {
				deletedRel = append(deletedRel, rel)
			}
		}
	}

	return deletedRel
}

// classifyGoInstallErr maps a `go install` failure to ErrGoNotFound when the go
// binary is absent from PATH (ErrCommandNotFound from the Commander), otherwise
// wrapping the raw error with the install mode for context.
func classifyGoInstallErr(mode string, runErr error) error {
	if errors.Is(runErr, ErrCommandNotFound) {
		return fmt.Errorf("go install (%s): %w", mode, ErrGoNotFound)
	}

	return fmt.Errorf("go install (%s): %w", mode, runErr)
}

// cleanupDanglingLinkEntry decides and (unless dryRun) enacts the D5 verdict
// for one surface-dir entry: real files/dirs and non-dangling symlinks
// (healthy engram links, foreign symlinks) return false, false; a dangling
// engram link is removed (RemoveAll on a path already confirmed via Lstat to
// be a symlink removes exactly that link) and reported via the bool return.
func cleanupDanglingLinkEntry(
	fileSystem Filesystem,
	entryPath, parentDir, root string,
	dryRun bool,
) (isDanglingEngramLink bool, err error) {
	info, lstatErr := fileSystem.Lstat(entryPath)
	if lstatErr != nil {
		return false, fmt.Errorf("lstat %s: %w", entryPath, lstatErr)
	}

	if info.Mode()&fs.ModeSymlink == 0 {
		return false, nil // real file/dir — never touched by cleanup
	}

	dangling, resolveErr := danglingLinkTarget(fileSystem, entryPath, parentDir, root)
	if resolveErr != nil {
		return false, resolveErr
	}

	if dangling == "" {
		return false, nil // healthy engram link, or a foreign symlink — leave it
	}

	if dryRun {
		return true, nil
	}

	removeErr := fileSystem.RemoveAll(entryPath)
	if removeErr != nil {
		return false, fmt.Errorf("remove dangling link %s: %w", entryPath, removeErr)
	}

	return true, nil
}

// cleanupDanglingLinksInDir is the D5 dangling-link scan for one harness
// surface dir: a ONE-LEVEL-DEEP ReadDir, then for each entry that Lstat
// reports as a symlink, danglingLinkTarget decides whether it is an engram
// link whose target is gone. Matching links are deleted (RemoveAll on a
// path already confirmed to be a symlink removes exactly that link, never
// cascading into whatever it pointed at). Real files/dirs are skipped by
// the Lstat-mode check before any resolution happens; foreign symlinks
// (resolving outside root) and healthy engram links (target present) are
// left alone by danglingLinkTarget. Returns the absolute paths removed (or,
// under dryRun, that WOULD be removed — nothing is deleted under dryRun).
func cleanupDanglingLinksInDir(fileSystem Filesystem, dir, root string, dryRun bool) ([]string, error) {
	if dir == "" {
		return nil, nil
	}

	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		if isNotExist(readErr) {
			return nil, nil
		}

		return nil, fmt.Errorf("read %s: %w", dir, readErr)
	}

	removed := make([]string, 0)

	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())

		isDangling, checkErr := cleanupDanglingLinkEntry(fileSystem, entryPath, dir, root, dryRun)
		if checkErr != nil {
			return nil, checkErr
		}

		if isDangling {
			removed = append(removed, entryPath)
		}
	}

	return removed, nil
}

func cmdRootFor(spec HarnessSpec, home string) string {
	if spec.CommandsTargetRel == "" {
		return ""
	}

	return filepath.Join(home, spec.CommandsTargetRel)
}

// collectGuidanceImports scans one harness config file's content for active
// @import lines matching either derived prefix, returning the set of imported
// guidance basenames. Lines inside fenced code blocks are ignored: a ```
// line toggles fence state, and an unclosed fence suppresses everything
// after it.
func collectGuidanceImports(data []byte, tildePrefix, expandedPrefix string) map[string]bool {
	imported := map[string]bool{}
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

		if base, ok := guidanceImportBase(trimmed, tildePrefix, expandedPrefix); ok {
			imported[base] = true
		}
	}

	return imported
}

// collectPrunableDirs recursively determines, for the subtree rooted at dir
// (rel is dir's path relative to root — the RelPath convention pruneEmptyDirs'
// callers and EngramSyncOp share), which directories hold no path that
// survives this sync. protected is the full set of paths (files AND every
// one of their ancestor directories) the sync intends to keep or create —
// see protectedIntendedPaths; a directory not in protected, and with no
// descendant that is, has nothing left in it after this sync applies.
// Reports only the OUTERMOST such directory per pruned (a nested
// content-free directory is not reported separately — removing its parent
// removes it too). Returns whether dir itself holds surviving content, so a
// caller one level up can fold that into its own verdict.
func collectPrunableDirs(
	dir, rel string,
	protected map[string]bool,
	fileSystem Filesystem,
	pruned *[]string,
) (hasContent bool, err error) {
	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		if isNotExist(readErr) {
			return false, nil
		}

		return false, fmt.Errorf("read %s: %w", dir, readErr)
	}

	for _, entry := range entries {
		childRel := filepath.Join(rel, entry.Name())
		childAbs := filepath.Join(dir, entry.Name())

		if !entry.IsDir() {
			if protected[childRel] {
				hasContent = true
			}

			continue
		}

		childHasContent, subErr := collectPrunableDirs(childAbs, childRel, protected, fileSystem, pruned)
		if subErr != nil {
			return false, subErr
		}

		if childHasContent || protected[childRel] {
			hasContent = true

			continue
		}

		*pruned = append(*pruned, childRel)
	}

	return hasContent, nil
}

func collectSkillDirs(order []string, counts map[string]int) []SkillDirCount {
	out := make([]SkillDirCount, 0, len(order))

	for _, name := range order {
		out = append(out, SkillDirCount{Name: name, Files: counts[name]})
	}

	return out
}

// collectWrittenPaths extracts all Dst (destination) paths from CopyOps
// for a given harness, used to track written files in manifest mode.
func collectWrittenPaths(skillOps, cmdOps, guidanceOps []CopyOp, name Harness) []string {
	out := make([]string, 0)

	for _, op := range skillOps {
		if op.Harness == name {
			out = append(out, op.Dst)
		}
	}

	for _, op := range cmdOps {
		if op.Harness == name {
			out = append(out, op.Dst)
		}
	}

	for _, op := range guidanceOps {
		if op.Harness == name {
			out = append(out, op.Dst)
		}
	}

	sort.Strings(out)

	return out
}

// createSymlink materializes a fresh symlink at an absent surface path
// (materializeSymlink's "absent" branch): MkdirAll the parent, then
// Symlink. A no-op under dryRun.
func createSymlink(fileSystem Filesystem, link, target string, dryRun bool) error {
	if dryRun {
		return nil
	}

	mkErr := fileSystem.MkdirAll(filepath.Dir(link), dirPerm)
	if mkErr != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(link), mkErr)
	}

	symlinkErr := fileSystem.Symlink(target, link)
	if symlinkErr != nil {
		return fmt.Errorf("symlink %s -> %s: %w", link, target, symlinkErr)
	}

	return nil
}

// currentManagedSubtreeFiles enumerates every file currently under root's
// managed subtrees (skills/, commands/, and guidance/ only when
// guidanceManaged), keyed by path relative to root. A missing subtree
// contributes nothing — that is not an error.
func currentManagedSubtreeFiles(root string, guidanceManaged bool, fileSystem Filesystem) (map[string]bool, error) {
	subtrees := []string{"skills", "commands"}
	if guidanceManaged {
		subtrees = append(subtrees, "guidance")
	}

	out := map[string]bool{}

	for _, subtree := range subtrees {
		files, listErr := listSubtreeFiles(filepath.Join(root, subtree), fileSystem)
		if listErr != nil {
			return nil, listErr
		}

		for _, rel := range files {
			out[filepath.Join(subtree, rel)] = true
		}
	}

	return out, nil
}

// danglingLinkTarget reads link's literal (unresolved) target, resolves it
// LEXICALLY against parentDir (D5) — never via filepath.EvalSymlinks, per
// vault note 475: resolution is for reaching bytes, never for naming them,
// and on macOS EvalSymlinks silently rewrites /var to /private/var, which
// would corrupt the prefix match below even for a tree with no user
// symlinks at all — and prefix-matches the resolved target against root.
// Returns the resolved target path when the link is a dangling engram link
// (root-matched, target missing); returns "" when the link is healthy
// (target present) or foreign (resolves outside root).
func danglingLinkTarget(fileSystem Filesystem, link, parentDir, root string) (string, error) {
	rawTarget, readErr := fileSystem.ReadLink(link)
	if readErr != nil {
		return "", fmt.Errorf("readlink %s: %w", link, readErr)
	}

	resolved := lexicallyResolveSymlinkTarget(rawTarget, parentDir)

	if !pathWithinRoot(resolved, root) {
		return "", nil // foreign symlink — never touched
	}

	_, statErr := fileSystem.Stat(resolved)

	switch {
	case statErr == nil:
		return "", nil // target present — healthy engram link
	case isNotExist(statErr):
		return resolved, nil
	default:
		return "", fmt.Errorf("stat %s: %w", resolved, statErr)
	}
}

// deletedRelPaths extracts the RelPath of every delete op, in plan order,
// for HarnessReport.EngramSyncDeleted.
func deletedRelPaths(ops []EngramSyncOp) []string {
	out := make([]string, 0, len(ops))

	for _, syncOp := range ops {
		if syncOp.Kind == EngramSyncDelete {
			out = append(out, syncOp.RelPath)
		}
	}

	return out
}

func describeGoInstall(source SourceInfo) string {
	if source.Mode == SourceLocal {
		return "go install ./cmd/engram/"
	}

	return "git clone " + repoCloneURL + " && go install ./cmd/engram/ (LFS-safe; #645)"
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

// guidanceImportBase returns the guidance basename imported by an exact
// @import line (either prefix form) and whether the line is such an import.
// The remainder after the prefix must be a single .md basename — no nested
// path segment, no trailing content.
func guidanceImportBase(trimmed, tildePrefix, expandedPrefix string) (string, bool) {
	for _, prefix := range []string{tildePrefix, expandedPrefix} {
		rest, ok := strings.CutPrefix(trimmed, prefix)
		if !ok {
			continue
		}

		if rest == "" || strings.Contains(rest, "/") || !strings.HasSuffix(rest, ".md") {
			continue
		}

		return rest, true
	}

	return "", false
}

// guidanceImportPrefixes derives the two recognized @import prefixes for one
// harness from its spec, so detection can never drift from the deploy target
// (this branch's original bug). The tilde form is built with literal forward
// slashes — import syntax is OS-independent — while the expanded-home form
// matches a rendered filesystem path under home.
func guidanceImportPrefixes(spec HarnessSpec, home string) (tildePrefix, expandedPrefix string) {
	tildePrefix = "@~/" + filepath.ToSlash(spec.GuidanceTargetRel) + "/"
	expandedPrefix = "@" + filepath.Join(home, spec.GuidanceTargetRel) + string(filepath.Separator)

	return tildePrefix, expandedPrefix
}

// intendedCommandBasenames collects the CommandFile basenames one harness's
// planned command CopyOps intend to deploy — reportSurfaceStrays' (task 5.2)
// "not a stray" set for a harness's commands root.
func intendedCommandBasenames(name Harness, cmdOps []CopyOp) map[string]bool {
	out := map[string]bool{}

	for _, copyOp := range cmdOps {
		if copyOp.Harness == name {
			out[copyOp.CommandFile] = true
		}
	}

	return out
}

// intendedGuidanceBasenames collects the GuidanceFile basenames one
// harness's planned guidance CopyOps intend to deploy — reportSurfaceStrays'
// (task 5.2) "not a stray" set for a harness's separate guidance dir (Pi).
func intendedGuidanceBasenames(name Harness, guidanceOps []CopyOp) map[string]bool {
	out := map[string]bool{}

	for _, copyOp := range guidanceOps {
		if copyOp.Harness == name {
			out[copyOp.GuidanceFile] = true
		}
	}

	return out
}

// intendedRootFiles re-roots one harness's planned skill/command/guidance
// CopyOps under its engram-owned root's skills/, commands/, guidance/
// subtrees (D1/D4): the sync engine's "intended set" input. Ops belonging to
// other harnesses are skipped. guidanceOps is already empty when guidance is
// not part of the intended set this run (Run's opt-in gate), so no separate
// guidance filter is needed here.
func intendedRootFiles(spec HarnessSpec, home string, skillOps, cmdOps, guidanceOps []CopyOp) []intendedRootFile {
	out := make([]intendedRootFile, 0, len(skillOps)+len(cmdOps)+len(guidanceOps))

	skillsPrefix := filepath.Join(home, spec.SkillsTargetRel) + string(filepath.Separator)

	for _, copyOp := range skillOps {
		if copyOp.Harness != spec.Name {
			continue
		}

		rel := strings.TrimPrefix(copyOp.Dst, skillsPrefix)
		out = append(out, intendedRootFile{RelPath: filepath.Join("skills", rel), Src: copyOp.Src})
	}

	for _, copyOp := range cmdOps {
		if copyOp.Harness != spec.Name {
			continue
		}

		out = append(out, intendedRootFile{RelPath: filepath.Join("commands", copyOp.CommandFile), Src: copyOp.Src})
	}

	for _, copyOp := range guidanceOps {
		if copyOp.Harness != spec.Name {
			continue
		}

		out = append(out, intendedRootFile{RelPath: filepath.Join("guidance", copyOp.GuidanceFile), Src: copyOp.Src})
	}

	return out
}

// intendedSkillDirNames collects the top-level skill-dir names one harness's
// planned skill CopyOps intend to deploy — reportSurfaceStrays' (task 5.2)
// "not a stray" set for a harness's skills root.
func intendedSkillDirNames(name Harness, skillOps []CopyOp) map[string]bool {
	out := map[string]bool{}

	for _, copyOp := range skillOps {
		if copyOp.Harness == name {
			out[copyOp.SkillDir] = true
		}
	}

	return out
}

// isNotExist reports whether err signals a missing file. Tests inject
// errors that wrap fs.ErrNotExist; production wraps os errors. Using
// errors.Is keeps the package free of *os.PathError checks.
func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// lexicallyResolveSymlinkTarget resolves a symlink's literal target string
// against the link's parent directory using pure string operations only
// (filepath.IsAbs, filepath.Join, filepath.Clean) — never
// filepath.EvalSymlinks, per vault note 475 and design.md D5: resolution is
// for reaching bytes, never for naming them. An absolute target is used
// as-is (Cleaned); a relative target is joined against parentDir first.
func lexicallyResolveSymlinkTarget(rawTarget, parentDir string) string {
	if filepath.IsAbs(rawTarget) {
		return filepath.Clean(rawTarget)
	}

	return filepath.Clean(filepath.Join(parentDir, rawTarget))
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

// listSubtreeFiles returns every file's path relative to dir, or an empty
// slice when dir does not exist — a not-yet-synced managed subtree
// (skills/, commands/, guidance/) under an engram-owned root is not an
// error, unlike listFilesRecursive's source-dir contract.
func listSubtreeFiles(dir string, fileSystem Filesystem) ([]string, error) {
	files := make([]string, 0)

	walkErr := walkSubtreeFiles(dir, "", fileSystem, &files)
	if walkErr != nil {
		return nil, walkErr
	}

	return files, nil
}

// loadManifestPaths loads the recorded paths from the manifest file.
// Missing manifest is treated as empty (no error).
func loadManifestPaths(fileSystem Filesystem, manifestPath string) ([]string, error) {
	manifestData, readErr := fileSystem.ReadFile(manifestPath)

	// Missing manifest is treated as empty.
	if readErr != nil {
		if !errors.Is(readErr, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading manifest %s: %w", manifestPath, readErr)
		}

		return []string{}, nil
	}

	var recorded []string

	err := json.Unmarshal(manifestData, &recorded)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling manifest: %w", err)
	}

	return recorded, nil
}

// materializeOrAdopt wraps materializeSymlink with the D6/task-5.1 adoption
// rule: a REAL file or directory already occupying an INTENDED-SET surface
// path (a pre-sync copy-mode deploy) is replaced with a symlink into the
// already-synced engram root, instead of being left blocked. RemoveAll is
// permitted HERE ONLY because every call site passes an intended-set member
// (a link path drawn from a planned CopyOp) — the corresponding content
// under root was already written by applyEngramRootSync before any caller
// of this function runs this same harness pass (D6: "the repo is the
// source of truth, so no content comparison is needed"). A real path
// OUTSIDE the intended set must never reach this function — see
// surfaceStrays for that case (task 5.2). Returns adopted=true when a real
// path was (or, under dryRun, would be) converted; dryRun previews without
// touching disk, matching materializeSymlink's own dryRun contract.
func materializeOrAdopt(fileSystem Filesystem, link, target string, dryRun bool) (adopted bool, err error) {
	blocked, linkErr := materializeSymlink(fileSystem, link, target, dryRun)
	if linkErr != nil {
		return false, linkErr
	}

	if !blocked {
		return false, nil
	}

	if dryRun {
		return true, nil
	}

	removeErr := fileSystem.RemoveAll(link)
	if removeErr != nil {
		return false, fmt.Errorf("adopt %s: remove real copy: %w", link, removeErr)
	}

	symlinkErr := fileSystem.Symlink(target, link)
	if symlinkErr != nil {
		return false, fmt.Errorf("adopt %s: symlink: %w", link, symlinkErr)
	}

	return true, nil
}

// materializeSymlink applies the D3 materialization rule at one
// harness-visible surface path: absent → create it (MkdirAll the parent,
// then Symlink); already a symlink → verify its target via ReadLink and
// repoint (RemoveAll the link, then Symlink again) only when it differs
// from target — a correctly-pointed link is left alone; a REAL file or
// directory at link → left COMPLETELY untouched (never RemoveAll'd) and
// reported back via the blockedByReal return so the caller can decide.
// materializeOrAdopt wraps this to convert an intended-set block into an
// adoption (task 5.1); a caller scanning for non-intended-set strays (task
// 5.2, surfaceStrays) never calls this function at all. Under dryRun, the
// create/repoint branches report their intended action's target state but
// perform no write.
func materializeSymlink(fileSystem Filesystem, link, target string, dryRun bool) (blockedByReal bool, err error) {
	info, lstatErr := fileSystem.Lstat(link)

	switch {
	case isNotExist(lstatErr):
		return false, createSymlink(fileSystem, link, target, dryRun)
	case lstatErr != nil:
		return false, fmt.Errorf("lstat %s: %w", link, lstatErr)
	case info.Mode()&fs.ModeSymlink != 0:
		return false, repointSymlinkIfWrong(fileSystem, link, target, dryRun)
	default:
		return true, nil
	}
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

// pathWithinRoot reports whether path is root itself or lies inside it,
// comparing the Cleaned strings only (no filesystem access, no symlink
// resolution) — the lexical-identity check D5 requires.
func pathWithinRoot(path, root string) bool {
	cleanRoot := filepath.Clean(root)
	if path == cleanRoot {
		return true
	}

	return strings.HasPrefix(path, cleanRoot+string(filepath.Separator))
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
				Harness:     spec.Name,
				Src:         filepath.Join(srcCommands, name),
				Dst:         filepath.Join(dstRoot, name),
				CommandFile: name,
			})
		}
	}

	return ops, nil
}

// The field names are PascalCase because that's the literal JSON `go list`
// emits — not the Go-conventional camelCase a linter would prefer.

// planEngramRootSync is the sync engine's pure diff step (D4): it compares
// the intended set against root's current managed-subtree contents
// (skills/, commands/, and guidance/ only when guidanceManaged — D4's
// opt-in gate) and returns an ordered op plan. Deletes are omitted entirely
// when allowDelete is false (D2's marker-less-root refusal). Every op stays
// inside root; nothing outside it is ever read or touched.
func planEngramRootSync(
	root string,
	intended []intendedRootFile,
	guidanceManaged, allowDelete bool,
	fileSystem Filesystem,
) ([]EngramSyncOp, error) {
	current, currentErr := currentManagedSubtreeFiles(root, guidanceManaged, fileSystem)
	if currentErr != nil {
		return nil, currentErr
	}

	ops := make([]EngramSyncOp, 0, len(intended))
	intendedRel := make(map[string]bool, len(intended))

	for _, item := range intended {
		intendedRel[item.RelPath] = true

		kind := EngramSyncCreate
		if current[item.RelPath] {
			kind = EngramSyncOverwrite
		}

		ops = append(ops, EngramSyncOp{
			Kind:    kind,
			RelPath: item.RelPath,
			AbsPath: filepath.Join(root, item.RelPath),
			Src:     item.Src,
		})
	}

	if allowDelete {
		for rel := range current {
			if intendedRel[rel] {
				continue
			}

			ops = append(ops, EngramSyncOp{Kind: EngramSyncDelete, RelPath: rel, AbsPath: filepath.Join(root, rel)})
		}
	}

	sortEngramSyncOps(ops)

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

// protectedIntendedPaths returns every intended file's RelPath plus every
// one of its ancestor directories (down to, but not including, the managed
// subtree it lives under), all keyed relative to root — collectPrunableDirs'
// "this must survive" set. A path not in this set, once the sync's own
// deletions apply, holds nothing this run intends to keep: an ancestor
// entry here protects a directory even when the file that will occupy it
// (an EngramSyncCreate target) does not exist on disk yet at plan time.
func protectedIntendedPaths(intended []intendedRootFile) map[string]bool {
	protected := make(map[string]bool, len(intended))

	for _, item := range intended {
		protected[item.RelPath] = true

		for dir := filepath.Dir(item.RelPath); dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
			if protected[dir] {
				break
			}

			protected[dir] = true
		}
	}

	return protected
}

// pruneEmptyDirs plans the D4-continued half of the sync: every directory
// within root's managed subtrees (skills/, commands/, and guidance/ only
// when guidanceManaged) that would hold no intended file once this sync's
// deletions apply. A directory a sync deletion leaves with zero content is
// as "present-but-unintended" as a lone file — left standing, it keeps a
// harness-visible symlink's target technically alive (Stat succeeds on the
// empty dir), so cleanupDanglingLinks never fires for it (the bug this
// closes: a removed skill's dir survived deletion of its last file, leaving
// the surface symlink pointing at a hollow shell). Gated on allowDelete the
// same way EngramSyncDelete ops are (D2): a deletion-refused run prunes
// nothing. Pure planning — reads the filesystem, writes nothing; callers
// apply via applyPrunedDirs. Returns paths relative to root, sorted, each
// the OUTERMOST directory left empty (removing it removes any
// now-content-free descendants with it — no redundant nested entries).
func pruneEmptyDirs(
	root string,
	intended []intendedRootFile,
	guidanceManaged, allowDelete bool,
	fileSystem Filesystem,
) ([]string, error) {
	if !allowDelete {
		return nil, nil
	}

	protected := protectedIntendedPaths(intended)

	subtrees := []string{"skills", "commands"}
	if guidanceManaged {
		subtrees = append(subtrees, "guidance")
	}

	pruned := make([]string, 0)

	for _, subtree := range subtrees {
		_, walkErr := collectPrunableDirs(filepath.Join(root, subtree), subtree, protected, fileSystem, &pruned)
		if walkErr != nil {
			return nil, walkErr
		}
	}

	sort.Strings(pruned)

	return pruned, nil
}

// repointSymlinkIfWrong reads an existing symlink's literal target and
// leaves it alone when it already equals target (the healthy, common case —
// this comparison is literal-string, matching how this package always
// writes engram-owned targets as absolute paths); otherwise it removes the
// stale link and creates a fresh one. RemoveAll here operates on a path
// materializeSymlink has already confirmed (via Lstat) is a symlink, so it
// removes exactly that link — never anything the link points at. A no-op
// under dryRun once the mismatch is detected.
func repointSymlinkIfWrong(fileSystem Filesystem, link, target string, dryRun bool) error {
	current, readErr := fileSystem.ReadLink(link)
	if readErr != nil {
		return fmt.Errorf("readlink %s: %w", link, readErr)
	}

	if current == target {
		return nil // healthy — already points at the right place
	}

	if dryRun {
		return nil
	}

	removeErr := fileSystem.RemoveAll(link)
	if removeErr != nil {
		return fmt.Errorf("remove stale link %s: %w", link, removeErr)
	}

	symlinkErr := fileSystem.Symlink(target, link)
	if symlinkErr != nil {
		return fmt.Errorf("symlink %s -> %s: %w", link, target, symlinkErr)
	}

	return nil
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

// resolveEngramRootOwnership decides how this run may treat root: absent →
// create root + marker, full sync; present with the `.engram-owned` marker →
// full sync; present without it → the D6 adoption pass (task 5.1): compute
// and report the top-level basenames that don't match the intended set
// (D2, "Ownership marker bounds sync deletion"), then write the marker —
// but sync-deletion of managed-subtree drift stays refused for THIS run
// regardless (D2's invariant is evaluated against root's state at the START
// of the run); a genuinely subsequent sync, with the marker already
// present, is what enables normal drift-deletion.
func resolveEngramRootOwnership(
	root string,
	intended []intendedRootFile,
	fileSystem Filesystem,
) (engramRootOwnership, error) {
	info, statErr := fileSystem.Stat(root)

	switch {
	case isNotExist(statErr):
		return engramRootOwnership{allowDelete: true, writeMarker: true}, nil
	case statErr != nil:
		return engramRootOwnership{}, fmt.Errorf("stat %s: %w", root, statErr)
	case !info.IsDir():
		return engramRootOwnership{}, fmt.Errorf("%s: %w", root, ErrEngramRootNotDir)
	}

	markerPath := filepath.Join(root, engramMarkerFile)

	_, markerErr := fileSystem.Stat(markerPath)

	switch {
	case markerErr == nil:
		return engramRootOwnership{allowDelete: true}, nil
	case !isNotExist(markerErr):
		return engramRootOwnership{}, fmt.Errorf("stat %s: %w", markerPath, markerErr)
	}

	unattributable, strayErr := unattributableRootFiles(root, intended, fileSystem)
	if strayErr != nil {
		return engramRootOwnership{}, strayErr
	}

	return engramRootOwnership{deletionRefused: true, writeMarker: true, unattributable: unattributable}, nil
}

// sortEngramSyncOps orders ops by RelPath for deterministic planning output.
// RelPath is unique per op (a path is never both a create/overwrite target
// and a delete target in the same plan), so this alone gives a total order.
func sortEngramSyncOps(ops []EngramSyncOp) {
	sort.Slice(ops, func(i, j int) bool { return ops[i].RelPath < ops[j].RelPath })
}

// supportedHarnesses returns the canonical list in install order.
func supportedHarnesses() []HarnessSpec {
	return []HarnessSpec{
		{
			Name:              HarnessClaude,
			ProbeRel:          ".claude",
			SkillsTargetRel:   filepath.Join(".claude", "skills"),
			GuidanceTargetRel: filepath.Join(".claude", "engram"),
			ImportsFileRel:    filepath.Join(".claude", "CLAUDE.md"),
			EngramRootRel:     filepath.Join(".claude", "engram"),
			// Verified symlink-capable (skill discovered through a
			// symlinked skill dir) — verification-verdicts.md.
			DeployMode: DeployModeSymlink,
		},
		{
			Name:              HarnessOpencode,
			ProbeRel:          filepath.Join(".config", "opencode"),
			SkillsTargetRel:   filepath.Join(".config", "opencode", "skills"),
			CommandsTargetRel: filepath.Join(".config", "opencode", "commands"),
			// OpenCode @import support unverified — GuidanceTargetRel and
			// ImportsFileRel empty until confirmed (guidance + detection skipped).
			EngramRootRel: filepath.Join(".config", "opencode", "engram"),
			// Verified symlink-capable (skill discovered, command executed
			// through symlinked paths) — verification-verdicts.md.
			DeployMode: DeployModeSymlink,
		},
		{
			Name:              HarnessPi,
			ProbeRel:          ".pi",
			SkillsTargetRel:   filepath.Join(".pi", "agent", "skills"),
			GuidanceTargetRel: filepath.Join(".pi", "agent", "guidance"),
			ImportsFileRel:    filepath.Join(".pi", "agent", "AGENTS.md"),
			EngramRootRel:     filepath.Join(".pi", "agent", "engram"),
			// Verified symlink-capable (skill discovered, guidance readable
			// through symlinked paths) — verification-verdicts.md.
			DeployMode: DeployModeSymlink,
		},
	}
}

// surfaceStrays scans dir ONE level deep (D6 item 3 / task 5.2) for REAL
// (non-symlink, via Lstat) entries whose name matches no basename in
// intended — engram cannot prove ownership of what it has no record of
// writing, so these are reported for manual review and NEVER touched.
// Symlinks are skipped entirely: a healthy, foreign, or dangling symlink is
// D5's concern (cleanupDanglingLinksInDir), not this stray scan's — and an
// intended-set path that has already been converted to a symlink is by
// definition not a stray. A missing dir contributes no strays (a
// not-yet-materialized surface is not an error).
func surfaceStrays(fileSystem Filesystem, dir string, intended map[string]bool) ([]string, error) {
	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		if isNotExist(readErr) {
			return nil, nil
		}

		return nil, fmt.Errorf("read %s: %w", dir, readErr)
	}

	strays := make([]string, 0, len(entries))

	for _, entry := range entries {
		if intended[entry.Name()] {
			continue
		}

		entryPath := filepath.Join(dir, entry.Name())

		info, lstatErr := fileSystem.Lstat(entryPath)
		if lstatErr != nil {
			return nil, fmt.Errorf("lstat %s: %w", entryPath, lstatErr)
		}

		if info.Mode()&fs.ModeSymlink != 0 {
			continue // a symlink here is D5's concern, not a stray
		}

		strays = append(strays, entryPath)
	}

	sort.Strings(strays)

	return strays, nil
}

// topLevelDir returns the first path segment of rel (skill-dir name).
func topLevelDir(rel string) string {
	first, _, _ := strings.Cut(rel, string(filepath.Separator))

	return first
}

// unattributableRootFiles scans root's TOP-LEVEL entries only (the managed
// subtrees skills/, commands/, guidance/ are always legitimate structure,
// never reported) for files whose basename matches no intended-set
// basename. A basename match (e.g. a pre-migration flat Claude guidance
// file sharing a name with an intended guidance file) is "adoptable": it is
// not reported and does not block anything except its own deletion, which
// never happens here since deletion is refused root-wide in this branch.
func unattributableRootFiles(root string, intended []intendedRootFile, fileSystem Filesystem) ([]string, error) {
	entries, readErr := fileSystem.ReadDir(root)
	if readErr != nil {
		if isNotExist(readErr) {
			return nil, nil
		}

		return nil, fmt.Errorf("read %s: %w", root, readErr)
	}

	wantBasenames := make(map[string]bool, len(intended))
	for _, item := range intended {
		wantBasenames[filepath.Base(item.RelPath)] = true
	}

	strays := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == engramMarkerFile || wantBasenames[entry.Name()] {
			continue
		}

		strays = append(strays, entry.Name())
	}

	sort.Strings(strays)

	return strays, nil
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

// walkSubtreeFiles is walkFilesRecursive's counterpart for a managed
// subtree: a missing dir yields no files rather than ErrSkillsSrcMissing,
// since an as-yet-unsynced skills/commands/guidance subtree under an
// engram-owned root is expected, not an error.
func walkSubtreeFiles(dir, rel string, fileSystem Filesystem, files *[]string) error {
	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		if isNotExist(readErr) {
			return nil
		}

		return fmt.Errorf("read %s: %w", dir, readErr)
	}

	for _, entry := range entries {
		childRel := filepath.Join(rel, entry.Name())
		childAbs := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			subErr := walkSubtreeFiles(childAbs, childRel, fileSystem, files)
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
