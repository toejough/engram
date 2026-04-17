package externalsources

import (
	"path/filepath"
)

// AutoMemorySettingsFunc returns the configured autoMemoryDirectory setting,
// if any, with a found flag. Wired at the edge to a settings.json reader.
type AutoMemorySettingsFunc func() (dir string, found bool)

// DirListerFunc lists files in a directory, returning absolute paths. An
// error or non-existent directory should return (nil, nil) so callers can
// treat both as "no contents". Wired at the edge to os.ReadDir.
type DirListerFunc func(dir string) ([]string, error)

// DiscoverAutoMemory resolves the auto-memory directory in this order:
//
//  1. autoMemoryDirectory setting (if found and the dir has files);
//  2. cwdProjectDir — the per-project default directory the caller computed;
//  3. mainProjectDir — worktree fallback, used only when (2) yields no files
//     and mainProjectDir != "" and != cwdProjectDir.
//
// Returns the *.md files in the resolved directory as ExternalFile entries.
// An empty result is normal — auto memory is opt-in and may not exist yet.
func DiscoverAutoMemory(
	cwdProjectDir, mainProjectDir string,
	settings AutoMemorySettingsFunc,
	dirLister DirListerFunc,
) []ExternalFile {
	if dir, ok := settings(); ok && dir != "" {
		if files := listAutoMemoryDir(dir, dirLister); len(files) > 0 {
			return files
		}
	}

	if files := listAutoMemoryDir(cwdProjectDir, dirLister); len(files) > 0 {
		return files
	}

	if mainProjectDir == "" || mainProjectDir == cwdProjectDir {
		return []ExternalFile{}
	}

	return listAutoMemoryDir(mainProjectDir, dirLister)
}

// listAutoMemoryDir lists *.md files in dir using dirLister and returns them
// as ExternalFile entries with KindAutoMemory.
func listAutoMemoryDir(dir string, dirLister DirListerFunc) []ExternalFile {
	paths, err := dirLister(dir)
	if err != nil || len(paths) == 0 {
		return []ExternalFile{}
	}

	files := make([]ExternalFile, 0, len(paths))

	for _, path := range paths {
		if filepath.Ext(path) == ".md" {
			files = append(files, ExternalFile{Kind: KindAutoMemory, Path: path})
		}
	}

	return files
}
