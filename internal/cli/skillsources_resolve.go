package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// ResolvedSkillSources is ResolveSkillSources' output (design D1).
type ResolvedSkillSources struct {
	// SkillScanResult holds every candidate in design D2 precedence order
	// (Key still empty: key construction is task 2.1), every root a scanner
	// tried to read with its scanned flag (D5's ScannedRoots are the
	// Scanned ones), and all scanner warnings. Disabled candidates are kept.
	SkillScanResult

	// Project is the project identity; ProjectFound is false outside a git
	// repository, in which case no project source was read.
	Project      ProjectIdentity
	ProjectFound bool
	// PluginManifestsRead, Plugins and PluginConflicts pass the plugin
	// scanner's facts through unchanged (ruling R3): the caller prints each
	// conflict and fails (task 2.3) and treats a conflicted or unscanned
	// plugin's scope as not scanned (task 2.4). They are zero when the
	// Claude Code harness is not detected, so no plugin note is ever
	// removal-eligible then.
	PluginManifestsRead bool
	Plugins             []ClaudePluginStatus
	PluginConflicts     []ClaudePluginConflict
}

// SkillSourceDeps are the capabilities ResolveSkillSources needs: a
// read-only filesystem and a command runner for the git project probe.
type SkillSourceDeps struct {
	FS        SkillSourceFS
	Commander update.Commander
}

// ResolveSkillSources resolves the default skill, command and prompt source
// set (design D1) — the one definition both `engram register-skills` and the
// `engram update` hook use. Harnesses come from update.DetectHarnesses: the
// Claude Code sources are read only when its harness is detected, the Pi
// (and `~/.agents`) sources only when Pi's is. Each harness's paths derive
// from its HarnessSpec (the config dir is the parent of SkillsTargetRel).
//
// Candidates are returned in D2 precedence order: Claude user skills, Claude
// user commands, synced, Pi user, agents user, Pi settings skills, Pi prompt
// templates (default folder, then settings), Pi packages, Claude plugins,
// Claude project, then the Pi project sources in the same order as the
// global Pi ones. Pi's trust decision is computed once (ruling R2) and gates
// every Pi project source; the global and project settings' `!`/`+`/`-`
// overrides mark default-folder candidates Disabled (rulings R8/R10).
//
// Outside a git repository no project source is read. The only error is a
// harness probe that fails for a reason other than not-exist.
func ResolveSkillSources(
	ctx context.Context, home, cwd string, deps SkillSourceDeps,
) (ResolvedSkillSources, error) {
	harnesses, err := update.DetectHarnesses(home, skillSourceProber{fsys: deps.FS})
	if err != nil {
		return ResolvedSkillSources{}, fmt.Errorf("detecting harnesses: %w", err)
	}

	var resolved ResolvedSkillSources

	resolved.Project, resolved.ProjectFound = probeProjectIdentity(ctx, cwd, deps.Commander)

	inputs := skillSourceInputs{
		fsys: deps.FS, home: home, cwd: cwd, project: resolved.Project, projectFound: resolved.ProjectFound,
	}

	var (
		claudeResult claudeSources
		piResult     piSources
	)

	for _, spec := range harnesses {
		configDir := filepath.Join(home, filepath.Dir(spec.SkillsTargetRel))
		skillsRoot := filepath.Join(home, spec.SkillsTargetRel)

		switch spec.Name {
		case update.HarnessClaude:
			claudeResult = inputs.claude(configDir, skillsRoot)
		case update.HarnessPi:
			piResult = inputs.pi(configDir, skillsRoot)
		}
	}

	resolved.SkillScanResult = joinSkillScanResults(
		claudeResult.user, piResult.user, claudeResult.plugins.SkillScanResult,
		claudeResult.project, piResult.project,
	)
	resolved.PluginManifestsRead = claudeResult.plugins.ManifestsRead
	resolved.Plugins = claudeResult.plugins.Plugins
	resolved.PluginConflicts = claudeResult.plugins.PluginConflicts

	return resolved, nil
}

// claudeSources are the Claude Code harness's results, by precedence slot.
type claudeSources struct {
	user    SkillScanResult
	plugins ClaudePluginScanResult
	project SkillScanResult
}

// piConfiguredParts splits ScanPiConfiguredSources' candidates into their
// precedence slots.
type piConfiguredParts struct {
	settingsSkills, settingsPrompts, packages                      []SkillCandidate
	projectSettingsSkills, projectSettingsPrompts, projectPackages []SkillCandidate
}

// piSources are the Pi harness's results, by precedence slot.
type piSources struct {
	user    SkillScanResult
	project SkillScanResult
}

// skillSourceInputs carries one resolution's fixed inputs.
type skillSourceInputs struct {
	fsys         SkillSourceFS
	home         string
	cwd          string
	project      ProjectIdentity
	projectFound bool
}

// claude scans the Claude Code harness's sources: user skills, commands and
// synced (D2 1-3), plugins (8), and the project chain (9).
func (in skillSourceInputs) claude(configDir, skillsRoot string) claudeSources {
	var sources claudeSources

	commandsRoot := filepath.Join(configDir, claudeCommandsDirName)

	mergeSkillScanResult(&sources.user, ScanClaudeUserSkills(in.fsys, skillsRoot))
	mergeSkillScanResult(&sources.user, ScanCommandDir(in.fsys, commandsRoot, SkillScopeClaudeCmd))
	mergeSkillScanResult(&sources.user, ScanSyncedSkills(in.fsys, filepath.Join(skillsRoot, syncedDirName)))

	pluginScan := ClaudePluginScan{ClaudeDir: configDir}
	if in.projectFound {
		pluginScan.RepoTopLevel = in.project.TopLevel
	}

	sources.plugins = ScanClaudePlugins(in.fsys, pluginScan)

	if in.projectFound {
		sources.project = ScanClaudeProjectSources(in.fsys, ClaudeProjectScan{
			Cwd:              in.cwd,
			TopLevel:         in.project.TopLevel,
			UserSkillsRoot:   skillsRoot,
			UserCommandsRoot: commandsRoot,
			ScopeID:          in.project.ScopeID(),
		})
	}

	return sources
}

// pi scans the Pi harness's sources: user skills, agents user skills,
// settings skills, prompt templates and packages (D2 4-7), and — for a
// trusted project — the project sources (10), applying each settings file's
// overrides to its default folders.
func (in skillSourceInputs) pi(agentDir, piSkillsRoot string) piSources {
	var sources piSources

	agentsDir := filepath.Join(in.home, agentsDirName)
	projectBase := filepath.Join(in.cwd, piConfigDirName)

	trusted := false

	if in.projectFound {
		var trustWarnings []string

		trusted, trustWarnings = ResolvePiProjectTrust(in.fsys, agentDir, in.cwd)
		sources.user.Warnings = append(sources.user.Warnings, trustWarnings...)
	}

	configured := ScanPiConfiguredSources(in.fsys, PiConfiguredScan{
		Home: in.home, AgentDir: agentDir, Cwd: in.cwd, ScopeID: in.project.ScopeID(), Trusted: trusted,
	})
	parts := splitPiConfigured(configured.Candidates, in.project.ScopeID())

	// ReadPiSettings' warnings were already reported by
	// ScanPiConfiguredSources, which read the same files.
	global, _ := ReadPiSettings(in.fsys, filepath.Join(agentDir, piSettingsFilename))

	piUser := withPiOverrides(ScanPiUserSkills(in.fsys, piSkillsRoot), global.Skills, agentDir)
	agentsUser := withPiOverrides(
		ScanAgentsUserSkills(in.fsys, filepath.Join(agentsDir, piSkillsDirName)), global.Skills, agentsDir,
	)
	prompts := withPiOverrides(
		ScanPiUserPrompts(in.fsys, filepath.Join(agentDir, piPromptsDirName)), global.Prompts, agentDir,
	)

	sources.user = joinSkillScanResults(
		sources.user, piUser, agentsUser, SkillScanResult{Candidates: parts.settingsSkills},
		prompts, SkillScanResult{Candidates: parts.settingsPrompts},
		SkillScanResult{Candidates: parts.packages, Roots: configured.Roots, Warnings: configured.Warnings},
	)

	if !trusted {
		return sources
	}

	project, _ := ReadPiSettings(in.fsys, filepath.Join(projectBase, piSettingsFilename))
	scan := PiProjectScan{
		Cwd:            in.cwd,
		TopLevel:       in.project.TopLevel,
		UserAgentsRoot: filepath.Join(agentsDir, piSkillsDirName),
		ScopeID:        in.project.ScopeID(),
		Trusted:        trusted,
	}

	sources.project = joinSkillScanResults(
		withPiOverrides(ScanPiProjectSkills(in.fsys, scan), project.Skills, projectBase),
		withAgentsChainOverrides(ScanAgentsProjectSkills(in.fsys, scan), project.Skills),
		SkillScanResult{Candidates: parts.projectSettingsSkills},
		withPiOverrides(ScanPiProjectPrompts(in.fsys, scan), project.Prompts, projectBase),
		SkillScanResult{Candidates: parts.projectSettingsPrompts},
		SkillScanResult{Candidates: parts.projectPackages},
	)

	return sources
}

// skillSourceProber adapts a SkillSourceFS to update.HarnessProber: Stat
// follows symlinks (ResolveSkillPath), so a symlinked ~/.claude still
// counts as the harness being present.
type skillSourceProber struct {
	fsys SkillSourceFS
}

func (p skillSourceProber) Stat(path string) (update.FileInfo, error) {
	_, info, err := resolveSkillPathInfo(p.fsys, path)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// appendUniqueWarnings appends each warning not already in warnings.
func appendUniqueWarnings(warnings, more []string) []string {
	for _, warning := range more {
		if !slices.Contains(warnings, warning) {
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

// joinSkillScanResults concatenates results in order.
func joinSkillScanResults(results ...SkillScanResult) SkillScanResult {
	var joined SkillScanResult

	for _, result := range results {
		mergeSkillScanResult(&joined, result)
	}

	return joined
}

// splitPiConfigured sorts ScanPiConfiguredSources' candidates into their
// precedence slots: global settings skills (D2 6), global settings prompts
// (6b), global packages (7), and the same three for the project (10).
func splitPiConfigured(candidates []SkillCandidate, projectScope string) piConfiguredParts {
	var parts piConfiguredParts

	for _, candidate := range candidates {
		isProject := candidate.ScopeID == projectScope
		isPackage := strings.HasPrefix(candidate.SourceSegment, SkillScopePiPkgPrefix)
		isPrompt := candidate.Kind == SkillSourceKindPrompt

		var slot *[]SkillCandidate

		switch {
		case isPackage && isProject:
			slot = &parts.projectPackages
		case isPackage:
			slot = &parts.packages
		case isPrompt && isProject:
			slot = &parts.projectSettingsPrompts
		case isPrompt:
			slot = &parts.settingsPrompts
		case isProject:
			slot = &parts.projectSettingsSkills
		default:
			slot = &parts.settingsSkills
		}

		*slot = append(*slot, candidate)
	}

	return parts
}

// withAgentsChainOverrides applies the project settings' overrides to a
// project `.agents/skills` chain, each level against its own `<dir>/.agents`
// base (Pi matches each ancestor level separately). An override Pi cannot
// evaluate (`**`) is warned about once per level.
func withAgentsChainOverrides(result SkillScanResult, patterns []string) SkillScanResult {
	baseByRoot := make(map[string]string, len(result.Roots))
	for _, root := range result.Roots {
		baseByRoot[root.Resolved] = filepath.Dir(root.Path)
	}

	out := result
	out.Candidates = make([]SkillCandidate, 0, len(result.Candidates))

	for _, candidate := range result.Candidates {
		overridden, warnings := ApplyPiSettingsOverrides(
			[]SkillCandidate{candidate}, patterns, baseByRoot[candidate.ReadRoot],
		)
		out.Candidates = append(out.Candidates, overridden...)
		out.Warnings = appendUniqueWarnings(out.Warnings, warnings)
	}

	return out
}

// withPiOverrides applies one settings file's overrides to a default-folder
// result matched against baseDir.
func withPiOverrides(result SkillScanResult, patterns []string, baseDir string) SkillScanResult {
	if len(result.Candidates) == 0 {
		return result
	}

	candidates, warnings := ApplyPiSettingsOverrides(result.Candidates, patterns, baseDir)
	result.Candidates = candidates
	result.Warnings = append(result.Warnings, warnings...)

	return result
}
