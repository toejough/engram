package externalsources

// DiscoverDeps bundles the function-typed dependencies Discover needs.
// Wiring this as a struct makes the public API stable as new optional
// dependencies are added.
type DiscoverDeps struct {
	CWD            string
	Home           string
	GOOS           string
	CWDProjectDir  string // pre-slugified project dir for auto memory (caller computes via ProjectSlug)
	MainProjectDir string // pre-slugified main repo dir for worktree fallback (empty if not in worktree)
	StatFn         StatFunc
	Reader         ReaderFunc
	MdWalker       MdWalker
	MatchAny       GlobMatcher
	Settings       AutoMemorySettingsFunc
	DirLister      DirListerFunc
	SkillFinder    SkillFinder
}

// Discover runs each per-source discovery and concatenates the results.
// Within the result, ordering is: CLAUDE.md (ancestors → user → managed) →
// imports → rules → auto memory → skills.
//
// The returned slice is the input to the recall pipeline phases; ordering
// here does NOT determine phase priority — that is set in
// internal/recall/orchestrate.go.
func Discover(deps DiscoverDeps) []ExternalFile {
	files := DiscoverClaudeMd(deps.CWD, deps.Home, deps.GOOS, deps.StatFn)

	for _, base := range files {
		files = append(files, ExpandImports(base.Path, deps.Reader)...)
	}

	files = append(files, DiscoverRules(deps.CWD, deps.Home, deps.MdWalker, deps.Reader, deps.MatchAny)...)
	files = append(files, DiscoverAutoMemory(
		deps.CWDProjectDir, deps.MainProjectDir, deps.Settings, deps.DirLister,
	)...)
	files = append(files, DiscoverSkills(deps.CWD, deps.Home, deps.SkillFinder)...)

	return files
}
