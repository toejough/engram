package cli_test

// Scenario-ID tests of the Controlled Vocabulary product capability, part 2, for `engram update`
// (docs/specs/product/controlled-vocabulary.md): 792a6v Update notices and 792a6w Migrating an old-format vocabulary.
// The other scenarios of part 2 are in controlled_vocabulary_part2_scenarios_test.go.
//
// A scenario starts from a vault written by the real commands over the in-memory vault of the part-1 tests
// (newCV1Vault, learn fact, vocab bootstrap) and puts it, file for file, on a machine of the update tests (iu2Machine:
// Claude Code under a home, a checkout of the engram module, a go toolchain whose install succeeds), where `engram
// update` runs as an operator meets it: the binary the install leaves runs the rest of the update. The vocabulary
// commands of that run work on the machine's own filesystem (cv2EdgeFS), with the topic embedder of the part-1 tests.
// The old-format files of 792w are written in the shape the earlier layout had; nothing of the current layout is.

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792a6v1a_Update_AnOldFormatVaultIsFlagged verifies 792a6v1a (controlled-vocabulary.md:613): over a vault holding
// vocab.index.md, engram update prints the old-format vocab notice and the file is still there.
func Test792a6v1a_Update_AnOldFormatVaultIsFlagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault holding vocab.index.md
	vault := newCV1Vault()
	vault.learnFact("a-note", cv1CookingSituations[0])
	vault.fsys.put(filepath.Join(cv1VaultPath, "vocab.index.md"), []byte(cv2LegacyIndex))

	machine := newCV2Machine(vault)
	before := machine.vaultSnapshot()

	// When the user runs engram update
	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	// Then it prints the old-format vocab notice and the file is still there
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(iu3VocabNotice))
	g.Expect(machine.fileSystem().text(iu3Vault + "/vocab.index.md")).To(Equal(cv2LegacyIndex))
	g.Expect(machine.vaultSnapshot()).To(Equal(before))
	g.Expect(machine.stderr.String()).To(BeEmpty())
}

// Test792a6v1b_Update_AMissingSelfTagIsFlagged verifies 792a6v1b (controlled-vocabulary.md:618): over a vault with a
// definition note tagged only vocab, engram update prints the missing self-tag notice and the note is unchanged.
func Test792a6v1b_Update_AMissingSelfTagIsFlagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a definition note tagged only vocab
	vault := newCV1Vault()
	vault.bootstrap(cv1Cooking)

	definitions := cv2StripSelfTags(g, vault, []string{"cooking"})
	g.Expect(definitions).To(HaveLen(1))

	machine := newCV2Machine(vault)
	before := machine.vaultSnapshot()

	// When the user runs engram update
	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	// Then it prints the missing self-tag notice and the note is unchanged
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(cv2SelfTagNotice))
	g.Expect(machine.vaultSnapshot()).To(Equal(before))
	g.Expect(machine.fileSystem().text(iu3Vault + "/" + definitions[0])).To(Equal(vault.read(definitions[0])))
	g.Expect(machine.stderr.String()).To(BeEmpty())
}

// Test792a6v1c_Update_ACurrentVaultGetsNoVocabNotice verifies 792a6v1c (controlled-vocabulary.md:623): over a vault
// with no vocab.<name>.md file and fully self-tagged definitions, engram update prints neither vocab notice.
func Test792a6v1c_Update_ACurrentVaultGetsNoVocabNotice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with no vocab.<name>.md file and fully self-tagged definitions
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)
	g.Expect(vault.namesMatching(`^vocab\..*\.md$`)).To(BeEmpty())
	g.Expect(vault.namesMatching(`-definition\.md$`)).To(HaveLen(3))

	machine := newCV2Machine(vault)

	// When the user runs engram update
	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	// Then neither vocab notice is printed
	g.Expect(outcome.stdout).To(ContainSubstring("Claude Code"), "the update ran to its report")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("old-format vocab files found"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("missing their vocab/<term> self-tag"))
}

// Test792a6w1a_UpdateRegenVocab_AnOldFormatVocabularyIsMigrated verifies 792a6w1a (controlled-vocabulary.md:649): over
// a vault holding vocab.cooking.md, vocab.index.md and a note with a Vocab: line, engram update --regen-vocab leaves a
// vocab-cooking-definition note, no old file, no Vocab: line in the note, and reports that it removed 2 old-format
// file(s) and cleaned 1 member note(s).
func Test792a6w1a_UpdateRegenVocab_AnOldFormatVocabularyIsMigrated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault holding vocab.cooking.md, vocab.index.md and a note with a Vocab: line
	machine := newCV2Machine(newCV2LegacyVault(g))

	// When the user runs engram update --regen-vocab
	outcome := machine.update(t, cli.UpdateArgs{RegenVocab: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	// Then the report says it removed 2 old-format file(s) and cleaned 1 member note(s)
	g.Expect(outcome.stdout).To(MatchRegexp(
		`(?m)^update --regen-vocab: removed 2 old-format file\(s\), cleaned 1 member note\(s\), ` +
			`seeded 1 term\(s\), reassigned [0-9]+ note\(s\)$`))

	// And a vocab-cooking-definition note exists, with the description and the exemplar of the old term note
	definitions := machine.names(`\.vocab-cooking-definition\.md$`)
	g.Expect(definitions).To(HaveLen(1))

	if len(definitions) != 1 {
		return
	}

	definition := machine.fileSystem().text(iu3Vault + "/" + definitions[0])
	g.Expect(definition).To(ContainSubstring(cv1Cooking.Description))
	g.Expect(definition).To(ContainSubstring("- " + cv1Exemplar))

	// And both old files are gone
	g.Expect(machine.fileSystem().kindOf(iu3Vault + "/vocab.cooking.md")).To(BeEmpty())
	g.Expect(machine.fileSystem().kindOf(iu3Vault + "/vocab.index.md")).To(BeEmpty())

	// And the note has no Vocab: line and no vocab: key
	member := machine.fileSystem().text(iu3Vault + "/" + machine.onlyName(g, `\.legacy-member\.md$`))
	g.Expect(member).NotTo(MatchRegexp(`(?m)^Vocab:`))
	g.Expect(member).NotTo(MatchRegexp(`(?m)^vocab:`))
	g.Expect(member).To(ContainSubstring("Information learned: when in tasting a recipe"))

	// And the old-format notice is not printed any more
	g.Expect(outcome.stdout).NotTo(ContainSubstring("old-format vocab files found"))
	g.Expect(machine.stderr.String()).To(BeEmpty())
}

// Test792a6w1b_UpdateRegenVocab_ADryRunChangesNothing verifies 792a6w1b (controlled-vocabulary.md:654): over the same
// vault, engram update --regen-vocab --dry-run prints "[dry-run] update --regen-vocab: would remove 2 old-format
// file(s), ..." and every file is unchanged.
func Test792a6w1b_UpdateRegenVocab_ADryRunChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given the same vault
	machine := newCV2Machine(newCV2LegacyVault(g))
	before := machine.fileSystem().snapshot("")

	// When the user runs engram update --regen-vocab --dry-run
	outcome := machine.update(t, cli.UpdateArgs{DryRun: true, RegenVocab: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	// Then it prints the preview, and every file is unchanged
	g.Expect(outcome.stdout).To(MatchRegexp(
		`(?m)^\[dry-run\] update --regen-vocab: would remove 2 old-format file\(s\), cleaned 1 member note\(s\), ` +
			`seeded 1 term\(s\), reassigned 0 note\(s\)$`))
	g.Expect(machine.fileSystem().snapshot("")).To(Equal(before))
	g.Expect(machine.stderr.String()).To(BeEmpty())
}

// Test792a6w1c_UpdateRegenVocab_NothingToMigrate verifies 792a6w1c (controlled-vocabulary.md:659): over a vault with no
// old-format vocabulary, engram update --regen-vocab prints "update --regen-vocab: nothing to regenerate".
func Test792a6w1c_UpdateRegenVocab_NothingToMigrate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with no old-format vocabulary
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)

	machine := newCV2Machine(vault)
	before := machine.vaultSnapshot()

	// When the user runs engram update --regen-vocab
	outcome := machine.update(t, cli.UpdateArgs{RegenVocab: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	// Then it prints that there is nothing to regenerate, and the vault is as it was
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement("update --regen-vocab: nothing to regenerate"))
	g.Expect(machine.vaultSnapshot()).To(Equal(before))
	g.Expect(machine.stderr.String()).To(BeEmpty())
}

// unexported constants.
const (
	// cv2LegacyIndex is the vocab.index.md the earlier layout generated: the list of the terms' notes.
	cv2LegacyIndex = "---\ntype: vocab-index\n---\n\n# Vocabulary\n\n- [[vocab.cooking]]\n"
	// cv2LegacyTerm is the vocab.cooking.md the earlier layout held for the term cooking.
	cv2LegacyTerm = "---\ntype: fact\nterm: cooking\ndescription: recipes and kitchens\n---\n\n" +
		"Exemplars:\n- Roast a chicken with fresh herbs\n"
	// cv2SelfTagNotice is the line the update prints when a definition note lacks its vocab/<term> self-tag.
	cv2SelfTagNotice = "vocab definition notes missing their vocab/<term> self-tag found — " +
		"run `engram vocab tag-definitions` to backfill them"
)

// unexported variables.
var (
	_                 cli.EdgeFS      = (*cv2EdgeFS)(nil)
	_                 fs.DirEntry     = cv2DirEntry{}
	_                 fs.FileInfo     = cv2FileInfo{}
	_                 update.FileInfo = cv2FileInfo{}
	errCV2Unsupported                 = errors.New("cv2EdgeFS: operation not supported")
)

// cv2DirEntry is one entry of a directory the machine lists.
type cv2DirEntry struct {
	entry update.DirEntry
}

func (e cv2DirEntry) Info() (fs.FileInfo, error) { return nil, errCV2Unsupported }

func (e cv2DirEntry) IsDir() bool { return e.entry.IsDir() }

func (e cv2DirEntry) Name() string { return e.entry.Name() }

func (e cv2DirEntry) Type() fs.FileMode {
	if e.entry.IsDir() {
		return fs.ModeDir
	}

	return 0
}

// cv2EdgeFS is the cli.EdgeFS view of the in-memory filesystem of an update machine (iu2FS), so that the vocabulary
// commands and the update work on the same files, links and directories.
type cv2EdgeFS struct {
	fileSystem *iu2FS
}

func (e *cv2EdgeFS) Lstat(path string) (fs.FileInfo, error) {
	info, err := e.fileSystem.Lstat(path)
	if err != nil {
		return nil, err
	}

	return cv2FileInfo{name: filepath.Base(path), info: info}, nil
}

func (e *cv2EdgeFS) MkdirAll(path string, perm fs.FileMode) error {
	return e.fileSystem.MkdirAll(path, perm)
}

func (e *cv2EdgeFS) MkdirTemp(string, string) (string, error) { return "", errCV2Unsupported }

func (e *cv2EdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := e.fileSystem.ReadDir(path)
	if err != nil {
		return nil, err
	}

	listed := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		listed = append(listed, cv2DirEntry{entry: entry})
	}

	return listed, nil
}

func (e *cv2EdgeFS) ReadFile(path string) ([]byte, error) { return e.fileSystem.ReadFile(path) }

func (e *cv2EdgeFS) Readlink(path string) (string, error) { return e.fileSystem.ReadLink(path) }

func (e *cv2EdgeFS) Remove(path string) error {
	if e.fileSystem.kindOf(path) == "" {
		return &fs.PathError{Op: "remove", Path: path, Err: fs.ErrNotExist}
	}

	return e.fileSystem.RemoveAll(path)
}

func (e *cv2EdgeFS) RemoveAll(path string) error { return e.fileSystem.RemoveAll(path) }

func (e *cv2EdgeFS) Rename(string, string) error { return errCV2Unsupported }

func (e *cv2EdgeFS) Stat(path string) (fs.FileInfo, error) {
	info, err := e.fileSystem.Stat(path)
	if err != nil {
		return nil, err
	}

	return cv2FileInfo{name: filepath.Base(path), info: info}, nil
}

func (e *cv2EdgeFS) Symlink(target, link string) error { return e.fileSystem.Symlink(target, link) }

func (e *cv2EdgeFS) WalkDir(string, fs.WalkDirFunc) error { return errCV2Unsupported }

func (e *cv2EdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return e.fileSystem.WriteFile(path, data, perm)
}

func (e *cv2EdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return e.fileSystem.WriteFile(path, data, perm)
}

func (e *cv2EdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	if e.fileSystem.kindOf(path) != "" {
		return &fs.PathError{Op: "create", Path: path, Err: fs.ErrExist}
	}

	return e.fileSystem.WriteFile(path, data, perm)
}

// cv2FileInfo is what the machine answers for Stat and Lstat.
type cv2FileInfo struct {
	name string
	info update.FileInfo
}

func (i cv2FileInfo) IsDir() bool { return i.info.IsDir() }

func (i cv2FileInfo) ModTime() time.Time { return time.Time{} }

func (i cv2FileInfo) Mode() fs.FileMode {
	if i.info.IsDir() {
		return i.info.Mode() | fs.ModeDir
	}

	return i.info.Mode()
}

func (i cv2FileInfo) Name() string { return i.name }

func (i cv2FileInfo) Size() int64 { return 0 }

func (i cv2FileInfo) Sys() any { return nil }

// cv2Machine is an update machine (iu2Machine) whose vault holds the files of an in-memory vault, and the vocabulary
// commands the update runs work on.
type cv2Machine struct {
	machine  *iu2Machine
	edge     *cv2EdgeFS
	embedder *cv1TopicEmbedder
	stderr   bytes.Buffer
}

// fileSystem is the filesystem of the machine.
func (m *cv2Machine) fileSystem() *iu2FS { return m.machine.fileSystem }

// names lists the vault's files matching the pattern.
func (m *cv2Machine) names(pattern string) []string {
	entries, err := m.machine.fileSystem.ReadDir(iu3Vault)
	if err != nil {
		return nil
	}

	matcher := regexp.MustCompile(pattern)
	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if matcher.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}

	return names
}

// onlyName is the one vault file matching the pattern; it fails the scenario when there is not exactly one.
func (m *cv2Machine) onlyName(g *WithT, pattern string) string {
	names := m.names(pattern)
	g.Expect(names).To(HaveLen(1), "files matching %s", pattern)

	if len(names) != 1 {
		return ""
	}

	return names[0]
}

// update runs `engram update` once as the machine's operator does: the binary the install leaves runs the rest of the
// update with the environment it was handed and the flags that reach it, and its status is the one the parent exits
// with. A dry run installs nothing, so the parent does it all.
func (m *cv2Machine) update(t *testing.T, args cli.UpdateArgs) iu2Outcome {
	t.Helper()

	var stdout bytes.Buffer

	exits := []int{}
	childErrs := []error{}

	m.machine.spawn.run = func(_ string, _ []string, env []string) (int, error) {
		vars := map[string]string{}

		for _, entry := range env {
			key, value, _ := strings.Cut(entry, "=")
			vars[key] = value
		}

		childEnv := iu1Env{home: m.machine.env.home, cwd: m.machine.env.cwd, vars: vars}
		childDeps := cli.ExportNewUpdateDepsWithSpawnAndVocab(
			m.machine.fileSystem, m.machine.cmd, childEnv, &iu1Spawner{}, func(int) {}, m.vocabDeps(),
		)

		childArgs := cli.UpdateArgs{WithGuidance: args.WithGuidance, RegenVocab: args.RegenVocab}

		childErr := cli.ExportRunUpdate(t.Context(), childArgs, childDeps, &stdout)
		if childErr != nil {
			childErrs = append(childErrs, childErr)

			return 1, nil
		}

		return 0, nil
	}

	deps := cli.ExportNewUpdateDepsWithSpawnAndVocab(
		m.machine.fileSystem, m.machine.cmd, m.machine.env, m.machine.spawn, func(code int) { exits = append(exits, code) },
		m.vocabDeps(),
	)

	err := cli.ExportRunUpdate(t.Context(), args, deps, &stdout)

	return iu2Outcome{err: err, stdout: stdout.String(), exits: exits, childErrs: childErrs}
}

// vaultSnapshot lists every node of the vault with its content.
func (m *cv2Machine) vaultSnapshot() []string { return m.machine.fileSystem.snapshot(iu3Vault) }

// vocabDeps are the vocabulary dependencies of an update run: the machine's filesystem and the topic embedder.
func (m *cv2Machine) vocabDeps() cli.VocabDeps {
	return cli.ExportNewVocabDeps(cli.Deps{
		FS:     m.edge,
		Lock:   fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }},
		Embed:  m.embedder,
		Stderr: &m.stderr,
		Now:    func() time.Time { return cv1Clock },
	})
}

// newCV2LegacyVault is a vault of the earlier layout: a note carrying a vocab: key and a Vocab: line, the term note
// vocab.cooking.md and the generated vocab.index.md.
func newCV2LegacyVault(g *WithT) *cv1Vault {
	vault := newCV1Vault()
	vault.learnFact("legacy-member", cv1CookingSituations[0])

	name := vault.requireNote(g, `\.legacy-member\.md$`)
	content := vault.read(name)

	legacy := strings.Replace(content, "vault: personal\n", "vault: personal\nvocab: [cooking]\n", 1)
	legacy = strings.Replace(legacy, "\n---\n\n", "\n---\n\nVocab: cooking\n", 1)
	g.Expect(legacy).To(ContainSubstring("\nvocab: [cooking]\n"))
	g.Expect(legacy).To(ContainSubstring("\nVocab: cooking\n"))

	vault.fsys.put(filepath.Join(cv1VaultPath, name), []byte(legacy))
	vault.fsys.put(filepath.Join(cv1VaultPath, "vocab.cooking.md"), []byte(cv2LegacyTerm))
	vault.fsys.put(filepath.Join(cv1VaultPath, "vocab.index.md"), []byte(cv2LegacyIndex))

	return vault
}

// newCV2Machine is an update machine whose vault holds the files of the in-memory vault.
func newCV2Machine(vault *cv1Vault) *cv2Machine {
	machine := iu3NewMachine()

	for path, content := range vault.fsys.snapshot() {
		machine.fileSystem.seedFile(iu3Vault+strings.TrimPrefix(path, cv1VaultPath), content)
	}

	return &cv2Machine{
		machine:  machine,
		edge:     &cv2EdgeFS{fileSystem: machine.fileSystem},
		embedder: &cv1TopicEmbedder{},
	}
}
