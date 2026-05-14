package update_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// Property tests --------------------------------------------------------

func TestDetectHarnesses_Property_ResultIsSubset(t *testing.T) {
	t.Parallel()

	// Generate any combination of harness probes; verify result is a subset
	// in stable canonical order.
	for _, combo := range [][]string{
		{},
		{".claude"},
		{".config/opencode"},
		{".claude", ".config/opencode"},
		{".config/opencode", ".claude"}, // order of insertion shouldn't matter
	} {
		fileSystem := newMemFS()
		for _, probe := range combo {
			fileSystem.dirs["/home/joe/"+probe] = true
		}

		got, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
		if err != nil {
			t.Fatalf("detect: %v", err)
		}

		// Order check.
		for i := 1; i < len(got); i++ {
			if got[i-1].Name == update.HarnessOpencode && got[i].Name == update.HarnessClaude {
				t.Fatalf("unexpected order: %v", got)
			}
		}
	}
}

func TestFirstModuleLine_LeadingBlankAndComment(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("\n// comment\nmodule github.com/toejough/engram\n")

	root, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestFirstModuleLine_NotModule(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("go 1.25.0\nmodule github.com/toejough/engram\n")

	_, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}

func TestFirstModuleLine_TrailingComment(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram // generated\n")

	root, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestParseGoListJSON_EmptyDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	cmd := &fakeCmd{
		responses: map[string][]byte{
			"go install github.com/toejough/engram/cmd/engram@latest": []byte("ok"),
			"go list -m -json github.com/toejough/engram@latest":      []byte(`{"Dir":"","Version":"v1"}`),
		},
	}

	updater := &update.Updater{FS: fileSystem, Cmd: cmd, Env: &fakeEnv{home: "/home/joe", cwd: "/tmp"}}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrSkillsSrcMissing)).To(BeTrue())
}

func TestUpdater_Run_DryRun_NoWritesNoCommands(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.DryRun).To(BeTrue())
	g.Expect(skillFileCount(report.Harnesses[0])).To(Equal(1))
	// No real writes occurred.
	g.Expect(fileSystem.written).To(BeEmpty())
	// No commands invoked.
	g.Expect(cmd.calls).To(BeEmpty())
}

func TestUpdater_Run_GoInstallFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true

	cmdErr := errors.New("go install boom")
	cmd := &fakeCmd{err: cmdErr}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, cmdErr)).To(BeTrue())
}

func TestUpdater_Run_GoListBadJSON(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	cmd := &fakeCmd{
		responses: map[string][]byte{
			"go install github.com/toejough/engram/cmd/engram@latest": []byte("ok"),
			"go list -m -json github.com/toejough/engram@latest":      []byte("not json"),
		},
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/tmp"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("parsing go list"))
}

func TestUpdater_Run_Local_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	// One harness present.
	fileSystem.dirs["/home/joe/.claude"] = true
	// Local repo with go.mod, skills, no slash commands.
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("learn skill")
	fileSystem.files["/repo/skills/learn/tests/baseline.md"] = []byte("baseline")
	fileSystem.dirs["/repo/skills/learn/tests"] = true

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo/internal/update"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Source.Mode).To(Equal(update.SourceLocal))
	g.Expect(report.Source.Root).To(Equal("/repo"))
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Name).To(Equal(update.HarnessClaude))
	g.Expect(skillFileCount(report.Harnesses[0])).To(Equal(2))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	// Confirm files written under home, not under repo.
	_, ok := fileSystem.written["/home/joe/.claude/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.claude/skills/learn/tests/baseline.md"]
	g.Expect(ok).To(BeTrue())

	// `go install ./cmd/engram/` should have been invoked.
	g.Expect(cmd.calls).To(HaveLen(1))
	g.Expect(cmd.calls[0]).To(Equal([]string{"go", "install", "./cmd/engram/"}))
}

func TestUpdater_Run_Local_GoInstallRunsFromModuleRoot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo/internal/update"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	// go install must run from the module root, not from the process cwd.
	g.Expect(cmd.calls).To(HaveLen(1))
	g.Expect(cmd.dirs[0]).To(Equal("/repo"))
}

func TestUpdater_Run_Local_OpencodeOnly_CopiesCommands(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.config/opencode"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/recall"] = true
	fileSystem.files["/repo/skills/recall/SKILL.md"] = []byte("recall")
	fileSystem.dirs["/repo/commands"] = true
	fileSystem.files["/repo/commands/recall.md"] = []byte("recall cmd")
	fileSystem.files["/repo/commands/learn.md"] = []byte("learn cmd")
	// Non-md should be ignored.
	fileSystem.files["/repo/commands/README.txt"] = []byte("readme")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	if len(report.Harnesses) == 0 {
		return
	}

	g.Expect(report.Harnesses[0].Name).To(Equal(update.HarnessOpencode))
	g.Expect(skillFileCount(report.Harnesses[0])).To(Equal(1))
	g.Expect(report.Harnesses[0].CommandFiles).To(HaveLen(2))

	_, ok := fileSystem.written["/home/joe/.config/opencode/commands/recall.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.config/opencode/commands/README.txt"]
	g.Expect(ok).To(BeFalse())
}

func TestUpdater_Run_ModuleCacheMiss(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	// No /go/pkg/mod/... directory exists.

	cmd := &fakeCmd{
		responses: map[string][]byte{
			"go install github.com/toejough/engram/cmd/engram@latest": []byte("ok"),
			"go list -m -json github.com/toejough/engram@latest":      []byte(`{"Dir":"/nope","Version":"v1"}`),
		},
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/tmp"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("module cache miss"))
}

func TestUpdater_Run_NoCwdReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe"}, // empty cwd
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("getwd"))
}

func TestUpdater_Run_NoHarness(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater := &update.Updater{
		FS:  newMemFS(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/anywhere"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrNoHarness)).To(BeTrue())
}

func TestUpdater_Run_NoHomeReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater := &update.Updater{
		FS:  newMemFS(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{}, // empty home
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("resolving home"))
}

func TestUpdater_Run_PartialFailure_AllFail(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/joe/.claude"] = true
	base.dirs["/repo"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/skills"] = true
	base.dirs["/repo/skills/learn"] = true
	base.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	fileSystem := &failWriteFS{memFS: base, failOn: ".claude/skills"}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
}

func TestUpdater_Run_Remote_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/home/joe/.config/opencode"] = true

	// No go.mod under cwd → remote mode.
	modCache := "/go/pkg/mod/github.com/toejough/engram@v0.2.0"
	fileSystem.dirs[modCache] = true
	fileSystem.dirs[modCache+"/skills"] = true
	fileSystem.dirs[modCache+"/skills/learn"] = true
	fileSystem.files[modCache+"/skills/learn/SKILL.md"] = []byte("learn from cache")
	fileSystem.dirs[modCache+"/commands"] = true
	fileSystem.files[modCache+"/commands/learn.md"] = []byte("learn cmd")

	goListOut := []byte(`{"Dir":"` + modCache + `","Version":"v0.2.0"}`)
	cmd := &fakeCmd{
		responses: map[string][]byte{
			"go install github.com/toejough/engram/cmd/engram@latest": []byte("ok"),
			"go list -m -json github.com/toejough/engram@latest":      goListOut,
		},
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/tmp"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Source.Mode).To(Equal(update.SourceRemote))
	g.Expect(report.Source.Root).To(Equal(modCache))
	g.Expect(report.Source.Version).To(Equal("v0.2.0"))
	g.Expect(report.Harnesses).To(HaveLen(2))

	// Both harnesses got the skill.
	_, ok := fileSystem.written["/home/joe/.claude/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.config/opencode/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())

	// OpenCode also got the command file; Claude did not (no commands target).
	_, ok = fileSystem.written["/home/joe/.config/opencode/commands/learn.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.claude/commands/learn.md"]
	g.Expect(ok).To(BeFalse())
}

func TestUpdater_Run_SkillsSrcMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	// No /repo/skills present.

	cmd := &fakeCmd{}
	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrSkillsSrcMissing)).To(BeTrue())
}

// failWriteFS wraps memFS but returns an error from WriteFile.
type failWriteFS struct {
	*memFS

	failOn string
}

func (f *failWriteFS) WriteFile(path string, data []byte, mode fs.FileMode) error {
	if strings.Contains(path, f.failOn) {
		return errors.New("write boom")
	}

	return f.memFS.WriteFile(path, data, mode)
}

type fakeCmd struct {
	responses map[string][]byte
	err       error
	calls     [][]string
	dirs      []string
}

func (f *fakeCmd) Run(_ context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)
	f.dirs = append(f.dirs, dir)

	if f.err != nil {
		return nil, nil, f.err
	}

	var keyBuilder strings.Builder

	keyBuilder.WriteString(name)

	for _, a := range args {
		keyBuilder.WriteString(" ")
		keyBuilder.WriteString(a)
	}

	key := keyBuilder.String()

	if resp, ok := f.responses[key]; ok {
		return resp, nil, nil
	}

	return nil, nil, nil
}

// --- fakes ---------------------------------------------------------------

type fakeEnv struct {
	home string
	cwd  string
}

func (f *fakeEnv) Getenv(_ string) string {
	return ""
}

func (f *fakeEnv) Getwd() (string, error) {
	if f.cwd == "" {
		return "", fs.ErrNotExist
	}

	return f.cwd, nil
}

func (f *fakeEnv) UserHomeDir() (string, error) {
	if f.home == "" {
		return "", fs.ErrNotExist
	}

	return f.home, nil
}

func skillFileCount(h update.HarnessReport) int {
	total := 0

	for _, d := range h.SkillDirs {
		total += d.Files
	}

	return total
}
