package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DiscoverOpts controls session discovery behavior.
type DiscoverOpts struct {
	ProjectsDir string // Directory to search (e.g., ~/.claude/projects/)
	Days        int    // Filter: only sessions modified in last N days (0 = no filter)
	Last        int    // Filter: return only last N sessions (0 = no filter)
	MinSize     int64  // Filter: minimum file size in bytes (0 = no filter)
}

// DiscoveredSession represents a session transcript file found on disk.
type DiscoveredSession struct {
	SessionID string
	Project   string
	Path      string
	ModTime   time.Time
	Size      int64
}

// DiscoverSessions walks ProjectsDir for *.jsonl session files, applies filters,
// and returns them sorted by most-recent-first.
func DiscoverSessions(opts DiscoverOpts) ([]DiscoveredSession, error) {
	var sessions []DiscoveredSession

	err := filepath.WalkDir(opts.ProjectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip subagents directories
		if d.IsDir() && d.Name() == "subagents" {
			return filepath.SkipDir
		}

		// Only process .jsonl files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Apply MinSize filter
		if opts.MinSize > 0 && info.Size() < opts.MinSize {
			return nil
		}

		// Apply Days filter
		if opts.Days > 0 {
			cutoff := time.Now().Add(-time.Duration(opts.Days) * 24 * time.Hour)
			if info.ModTime().Before(cutoff) {
				return nil
			}
		}

		// Derive project name from directory structure
		// Path structure: ProjectsDir/{encoded-project-path}/session.jsonl
		relPath, err := filepath.Rel(opts.ProjectsDir, path)
		if err != nil {
			return err
		}

		// Extract the encoded directory name (first path component)
		dirName := strings.Split(relPath, string(filepath.Separator))[0]

		// Reverse the Claude project directory encoding
		projectPath := ReverseClaudeProjectDir(dirName)

		// Derive project name from the reversed path
		projectName := DeriveProjectName(projectPath)

		sessions = append(sessions, DiscoveredSession{
			SessionID: strings.TrimSuffix(d.Name(), ".jsonl"),
			Project:   projectName,
			Path:      path,
			ModTime:   info.ModTime(),
			Size:      info.Size(),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by most-recent-first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})

	// Apply Last filter
	if opts.Last > 0 && len(sessions) > opts.Last {
		sessions = sessions[:opts.Last]
	}

	return sessions, nil
}

// ReverseClaudeProjectDir converts a dash-separated encoded directory name
// back to a filesystem path. For example:
// "-Users-joe-repos-personal-projctl" -> "/Users/joe/repos/personal/projctl"
func ReverseClaudeProjectDir(dirName string) string {
	if dirName == "" {
		return ""
	}

	// Remove leading dash and split by dashes
	trimmed := strings.TrimPrefix(dirName, "-")
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, "-")

	return "/" + filepath.Join(parts...)
}
