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
//
// Imports are deduplicated by path across CLAUDE.md ancestors: when two
// ancestors transitively import the same file, it appears once in the result.
func Discover(deps DiscoverDeps) []ExternalFile {
	claudeMdFiles := DiscoverClaudeMd(deps.CWD, deps.Home, deps.GOOS, deps.StatFn)

	files := make([]ExternalFile, 0, len(claudeMdFiles)*importCapacityFactor)
	files = append(files, claudeMdFiles...)

	visited := make(map[string]bool, len(claudeMdFiles))
	for _, file := range claudeMdFiles {
		visited[file.Path] = true
	}

	for _, base := range claudeMdFiles {
		for _, imported := range ExpandImports(base.Path, deps.Reader) {
			if visited[imported.Path] {
				continue
			}

			visited[imported.Path] = true
			files = append(files, imported)
		}
	}

	files = append(files, DiscoverRules(deps.CWD, deps.Home, deps.MdWalker, deps.Reader, deps.MatchAny)...)
	files = append(files, DiscoverAutoMemory(
		deps.CWDProjectDir, deps.MainProjectDir, deps.Settings, deps.DirLister,
	)...)
	files = append(files, DiscoverSkills(deps.CWD, deps.Home, deps.SkillFinder)...)

	return files
}

// unexported constants.
const (
	// importCapacityFactor is the heuristic multiplier on len(claudeMdFiles)
	// used to pre-size the discovered-files slice — most projects produce
	// imports roughly equal in count to ancestor CLAUDE.md files, so 2× is
	// a reasonable initial capacity that avoids the first reallocation.
	importCapacityFactor = 2
)
