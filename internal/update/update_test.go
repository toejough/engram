package update_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/update"
)

func TestDetectHarnesses_Both_StableOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/home/joe/.config/opencode"] = true

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(HaveLen(2))
	g.Expect(detected[0].Name).To(Equal(update.HarnessClaude))
	g.Expect(detected[1].Name).To(Equal(update.HarnessOpencode))
}

func TestDetectHarnesses_ClaudeOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(HaveLen(1))
	g.Expect(detected[0].Name).To(Equal(update.HarnessClaude))
}

func TestDetectHarnesses_None(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(BeEmpty())
}

func TestPlanSkillCopies_FilesUnderHome_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		homeSeg := rapid.StringMatching(`[a-z]{2,5}`).Draw(rt, "home")
		home := "/h/" + homeSeg

		fileSystem := newMemFS()
		fileSystem.dirs["/src/skills"] = true
		fileSystem.dirs["/src/skills/learn"] = true
		fileSystem.files["/src/skills/learn/SKILL.md"] = []byte("x")
		fileSystem.dirs["/src/skills/recall"] = true
		fileSystem.files["/src/skills/recall/SKILL.md"] = []byte("x")

		// Detect Claude only.
		fileSystem.dirs[home+"/.claude"] = true

		harnesses, err := update.ExportDetectHarnesses(home, fileSystem)
		if err != nil {
			rt.Fatalf("detect: %v", err)
		}

		ops, err := update.ExportPlanSkillCopies("/src/skills", home, harnesses, fileSystem)
		if err != nil {
			rt.Fatalf("plan: %v", err)
		}

		for _, op := range ops {
			if !strings.HasPrefix(op.Dst, home+string(filepath.Separator)) {
				rt.Fatalf("dst %q not under home %q", op.Dst, home)
			}
		}
	})
}

func TestPlanSkillCopies_MissingSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	harnesses := []update.HarnessSpec{
		{Name: update.HarnessClaude, ProbeRel: ".claude", SkillsTargetRel: ".claude/skills"},
	}

	_, err := update.ExportPlanSkillCopies("/nonexistent", "/home/joe", harnesses, fileSystem)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrSkillsSrcMissing)).To(BeTrue())
}

func TestWalkUpForModule_FoundAtStart(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n\ngo 1.25.6\n")

	root, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestWalkUpForModule_FoundInAncestor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")

	root, found, err := update.ExportWalkUpForModule("/repo/internal/update", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestWalkUpForModule_NotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	// no go.mod anywhere

	root, found, err := update.ExportWalkUpForModule("/some/where", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
	g.Expect(root).To(BeEmpty())
}

func TestWalkUpForModule_Property_TerminatesForAnyPath(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate a posix-style absolute path with 0-6 segments.
		depth := rapid.IntRange(0, 6).Draw(rt, "depth")

		segs := rapid.SliceOfN(rapid.StringMatching(`[a-z]{1,4}`), depth, depth).Draw(rt, "segs")

		path := "/"
		for _, seg := range segs {
			path = filepath.Join(path, seg)
		}

		fileSystem := newMemFS() // no go.mod anywhere

		_, found, err := update.ExportWalkUpForModule(path, fileSystem)
		if err != nil {
			rt.Fatalf("unexpected error: %v", err)
		}

		if found {
			rt.Fatalf("expected not-found for empty fs, got found=true")
		}
	})
}

func TestWalkUpForModule_ReadFileErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	wantErr := errors.New("disk failed")
	fileSystem := &errFS{readErr: wantErr}

	_, _, err := update.ExportWalkUpForModule("/anywhere", fileSystem)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, wantErr)).To(BeTrue())
}

func TestWalkUpForModule_WrongModuleStopsWalk(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/somewhere/go.mod"] = []byte("module example.com/other\n")

	// Should NOT keep walking up past a found go.mod with the wrong module.
	root, found, err := update.ExportWalkUpForModule("/somewhere/sub", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
	g.Expect(root).To(BeEmpty())
}

// errFS returns the same error for every ReadFile call.
type errFS struct {
	readErr error
}

func (*errFS) MkdirAll(_ string, _ fs.FileMode) error {
	return nil
}

func (*errFS) ReadDir(_ string) ([]update.DirEntry, error) {
	return nil, fs.ErrNotExist
}

func (e *errFS) ReadFile(_ string) ([]byte, error) {
	return nil, e.readErr
}

func (*errFS) RemoveAll(_ string) error {
	return nil
}

func (*errFS) Stat(_ string) (update.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func (*errFS) WriteFile(_ string, _ []byte, _ fs.FileMode) error {
	return nil
}

type memEntry struct {
	name  string
	isDir bool
}

func (m *memEntry) IsDir() bool { return m.isDir }

func (m *memEntry) Name() string { return m.name }

// --- in-memory test doubles --------------------------------------------

type memFS struct {
	files   map[string][]byte
	dirs    map[string]bool
	written map[string][]byte
	removed []string
}

func (m *memFS) MkdirAll(path string, _ fs.FileMode) error {
	m.dirs[path] = true
	return nil
}

func (m *memFS) ReadDir(path string) ([]update.DirEntry, error) {
	if !m.dirExists(path) {
		return nil, fs.ErrNotExist
	}

	prefix := dirPrefix(path)
	seen := map[string]bool{}
	out := make([]update.DirEntry, 0)

	for filePath := range m.files {
		addChild(filePath, prefix, false, seen, &out)
	}

	for dirPath := range m.dirs {
		addChild(dirPath, prefix, true, seen, &out)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })

	return out, nil
}

func (m *memFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return data, nil
}

func (m *memFS) RemoveAll(path string) error {
	m.removed = append(m.removed, path)
	delete(m.dirs, path)
	delete(m.files, path)

	prefix := dirPrefix(path)

	for p := range m.files {
		if strings.HasPrefix(p, prefix) {
			delete(m.files, p)
		}
	}

	for p := range m.dirs {
		if strings.HasPrefix(p, prefix) {
			delete(m.dirs, p)
		}
	}

	return nil
}

func (m *memFS) Stat(path string) (update.FileInfo, error) {
	if m.dirs[path] {
		return &memInfo{isDir: true}, nil
	}

	if _, ok := m.files[path]; ok {
		return &memInfo{isDir: false}, nil
	}

	return nil, fs.ErrNotExist
}

func (m *memFS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	m.written[path] = append([]byte(nil), data...)
	m.files[path] = m.written[path]

	return nil
}

func (m *memFS) dirExists(path string) bool {
	if m.dirs[path] {
		return true
	}

	prefix := dirPrefix(path)

	for filePath := range m.files {
		if strings.HasPrefix(filePath, prefix) {
			return true
		}
	}

	for dirPath := range m.dirs {
		if strings.HasPrefix(dirPath, prefix) {
			return true
		}
	}

	return false
}

type memInfo struct{ isDir bool }

func (m *memInfo) IsDir() bool { return m.isDir }

func addChild(
	fullPath, prefix string,
	forceIsDir bool,
	seen map[string]bool,
	out *[]update.DirEntry,
) {
	if !strings.HasPrefix(fullPath, prefix) {
		return
	}

	rest := strings.TrimPrefix(fullPath, prefix)
	if rest == "" {
		return
	}

	name, _, hasSlash := strings.Cut(rest, "/")
	if seen[name] {
		return
	}

	seen[name] = true
	*out = append(*out, &memEntry{name: name, isDir: forceIsDir || hasSlash})
}

func dirPrefix(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}

	return path + "/"
}

func newMemFS() *memFS {
	return &memFS{
		files:   map[string][]byte{},
		dirs:    map[string]bool{},
		written: map[string][]byte{},
	}
}
