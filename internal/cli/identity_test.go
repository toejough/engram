package cli_test

import (
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// TestApplyIdentityBackfill covers the update.Report-copying wrapper around
// backfillIdentity: a non-dry-run backfill sets IdentityBackfillRan/
// NotesStamped and re-checks VaultHasNotesMissingIdentity (false once
// everything is stamped, so a follow-up plain `engram update` prints no
// notice); a dry run leaves VaultHasNotesMissingIdentity untouched (it
// never got a chance to write, so the caller's own pre-existing scan value
// stands).
func TestApplyIdentityBackfill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const missingIdentityNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\n---\n\nbody\n"

	fileSystem := newU1FS()
	fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(missingIdentityNote)

	// deps reads/writes through the SAME map notesMissingIdentityFields'
	// re-check scans (fileSystem.files) — a write must be visible to the
	// re-check, or the assertion below would pass for the wrong reason
	// (two independently-populated but never-reconciled maps).
	deps := cli.IdentityDeps{
		Lock:       func(string) (func(), error) { return func() {}, nil },
		ListMD:     func(string) ([]string, error) { return []string{"1.2026-01-01.a.md"}, nil },
		ReadFile:   func(path string) ([]byte, error) { return fileSystem.files[path], nil },
		WriteFile:  func(path string, data []byte) error { fileSystem.files[path] = data; return nil },
		DetectRepo: func(context.Context) string { return "git@github.com:example/vault.git" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}

	report := &update.Report{}

	err := cli.ExportApplyIdentityBackfill(t.Context(), "/vault", deps, false, fileSystem, report)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.IdentityBackfillRan).To(BeTrue())
	g.Expect(report.IdentityBackfillNotesStamped).To(Equal(1))
	g.Expect(report.VaultHasNotesMissingIdentity).To(BeFalse(), "re-checked after a real (non-dry-run) backfill")

	// Dry run: report.VaultHasNotesMissingIdentity is left as whatever it
	// already was — applyIdentityBackfill only re-checks on a real run.
	dryFiles := map[string][]byte{"/vault/1.2026-01-01.a.md": []byte(missingIdentityNote)}
	dryDeps := deps
	dryDeps.ReadFile = func(path string) ([]byte, error) { return dryFiles[path], nil }
	dryDeps.WriteFile = func(string, []byte) error {
		t.Fatal("dry run must not write")

		return nil
	}
	dryReport := &update.Report{VaultHasNotesMissingIdentity: true}

	dryErr := cli.ExportApplyIdentityBackfill(t.Context(), "/vault", dryDeps, true, fileSystem, dryReport)
	g.Expect(dryErr).NotTo(HaveOccurred())
	g.Expect(dryReport.IdentityBackfillNotesStamped).To(Equal(1), "dry run still reports the would-stamp count")
	g.Expect(dryReport.VaultHasNotesMissingIdentity).To(BeTrue(), "left untouched under --dry-run")
}

// TestBackfillIdentity covers the backfill mechanism end to end: a note
// missing identity with a project: field gets repo: from project: (not
// fresh detection), a note missing identity with no project: gets repo:
// from fresh detection, an already-stamped note is left untouched, and a
// second run over the same vault stamps nothing further (idempotent).
func TestBackfillIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const withProject = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\nproject: my-project\n---\n\n" +
		"Information learned: when in s, a b c.\n\n"

	const withoutProject = "---\ntype: feedback\nsituation: s2\nbehavior: b\nimpact: i\naction: act\n" +
		"luhmann: \"2\"\ncreated: 2026-01-02\nsource: agent\n---\n\n" +
		"Lesson learned: when s2, act.\n\n"

	const alreadyStamped = "---\ntype: fact\nsituation: s3\nsubject: x\npredicate: y\nobject: z\n" +
		"luhmann: \"3\"\ncreated: 2026-01-03\nsource: agent\nuser: existing@example.com\nvault: work\n---\n\n" +
		"Information learned: when in s3, x y z.\n\n"

	files := map[string][]byte{
		"/vault/1.2026-01-01.with-project.md":    []byte(withProject),
		"/vault/2.2026-01-02.without-project.md": []byte(withoutProject),
		"/vault/3.2026-01-03.already-stamped.md": []byte(alreadyStamped),
	}

	writeCalls := 0

	deps := cli.IdentityDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{
				"1.2026-01-01.with-project.md",
				"2.2026-01-02.without-project.md",
				"3.2026-01-03.already-stamped.md",
			}, nil
		},
		ReadFile: func(path string) ([]byte, error) { return files[path], nil },
		WriteFile: func(path string, data []byte) error {
			writeCalls++
			files[path] = data

			return nil
		},
		DetectRepo: func(context.Context) string { return "git@github.com:example/vault.git" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}

	stamped, err := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).
		To(Equal(2), "the two notes missing identity get stamped; the already-stamped one doesn't")
	g.Expect(writeCalls).To(Equal(2))

	g.Expect(string(files["/vault/1.2026-01-01.with-project.md"])).
		To(ContainSubstring("repo: my-project"),
			"repo: prefers the note's own project: field")
	g.Expect(string(files["/vault/1.2026-01-01.with-project.md"])).
		To(ContainSubstring("user: agent@example.com"))
	g.Expect(string(files["/vault/1.2026-01-01.with-project.md"])).
		To(ContainSubstring("vault: personal"))

	g.Expect(string(files["/vault/2.2026-01-02.without-project.md"])).
		To(ContainSubstring("repo: git@github.com:example/vault.git"), "no project: falls back to fresh detection")

	g.Expect(string(files["/vault/3.2026-01-03.already-stamped.md"])).To(Equal(alreadyStamped),
		"an already-stamped note must be left byte-for-byte untouched")

	// Idempotency: a second run finds nothing newly missing.
	secondStamped, secondErr := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
	g.Expect(secondErr).NotTo(HaveOccurred())
	g.Expect(secondStamped).To(Equal(0))
	g.Expect(writeCalls).To(Equal(2), "no additional writes on the idempotent second run")
}

// TestBackfillIdentity_SkipsUnrelatedContentAndPropagatesReadErrors covers
// backfillOneNote's other branches: a note with no parseable frontmatter
// and a non-fact/feedback note (e.g. a vocab definition) both self-silence
// (stamped=false, err=nil) rather than failing the whole run; a genuine
// read failure on ANY note propagates as a real error and aborts the run
// (backfillIdentity must never silently skip an I/O failure the way it
// silently skips unrelated content).
func TestBackfillIdentity_SkipsUnrelatedContentAndPropagatesReadErrors(t *testing.T) {
	t.Parallel()

	const vocabNote = "---\ntype: term\nterm: recall\ndescription: recall the vault\n---\n\nbody\n"

	t.Run("skips non-frontmatter and non-fact/feedback content", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		files := map[string][]byte{
			"/vault/garbage.md":      []byte("not frontmatter at all"),
			"/vault/vocab.recall.md": []byte(vocabNote),
		}

		deps := cli.IdentityDeps{
			Lock:       func(string) (func(), error) { return func() {}, nil },
			ListMD:     func(string) ([]string, error) { return []string{"garbage.md", "vocab.recall.md"}, nil },
			ReadFile:   func(path string) ([]byte, error) { return files[path], nil },
			WriteFile:  func(string, []byte) error { t.Fatal("must not write"); return nil },
			DetectRepo: func(context.Context) string { return "repo" },
			DetectUser: func(context.Context) string { return "user" },
			Getenv:     func(string) string { return "" },
		}

		stamped, err := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stamped).To(Equal(0))
	})

	t.Run("a read failure propagates and aborts the run", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		errRead := errors.New("disk gone")
		deps := cli.IdentityDeps{
			Lock:       func(string) (func(), error) { return func() {}, nil },
			ListMD:     func(string) ([]string, error) { return []string{"1.a.md"}, nil },
			ReadFile:   func(string) ([]byte, error) { return nil, errRead },
			WriteFile:  func(string, []byte) error { t.Fatal("must not write"); return nil },
			DetectRepo: func(context.Context) string { return "repo" },
			DetectUser: func(context.Context) string { return "user" },
			Getenv:     func(string) string { return "" },
		}

		_, err := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
		g.Expect(err).To(HaveOccurred())
	})
}

// TestDetectRepo covers the three cases from design.md: origin remote
// configured, no origin remote (dirname fallback), and not inside a git
// repo at all (Getwd failing stands in for "no git" too, since detectRepo
// treats every failure path the same way — resolve to "").
func TestDetectRepo(t *testing.T) {
	t.Parallel()

	table := []struct {
		name      string
		getwd     func() (string, error)
		commander fakeGitCommander
		want      string
	}{
		{
			name:      "origin remote configured",
			getwd:     func() (string, error) { return "/repo", nil },
			commander: fakeGitCommander{remoteOut: "git@github.com:example/repo.git\n"},
			want:      "git@github.com:example/repo.git",
		},
		{
			name:  "no origin remote falls back to toplevel basename",
			getwd: func() (string, error) { return "/repo", nil },
			commander: fakeGitCommander{
				remoteErr:   errFakeGitFailed,
				revParseOut: "/home/joe/src/engram\n",
			},
			want: "engram",
		},
		{
			name:  "not inside a git repo resolves to empty",
			getwd: func() (string, error) { return "/tmp/scratch", nil },
			commander: fakeGitCommander{
				remoteErr:   errFakeGitFailed,
				revParseErr: errFakeGitFailed,
			},
			want: "",
		},
		{
			name:      "getwd failure resolves to empty",
			getwd:     func() (string, error) { return "", errFakeGitFailed },
			commander: fakeGitCommander{remoteOut: "should never be reached"},
			want:      "",
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := cli.ExportDetectRepo(t.Context(), tc.getwd, tc.commander)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestDetectUser covers user.email present and user.email absent (machine-
// username fallback), plus both signals failing.
func TestDetectUser(t *testing.T) {
	t.Parallel()

	table := []struct {
		name      string
		commander fakeGitCommander
		username  func() (string, error)
		want      string
	}{
		{
			name:      "git config user.email present",
			commander: fakeGitCommander{configOut: "agent@example.com\n"},
			username:  func() (string, error) { return "should-not-be-used", nil },
			want:      "agent@example.com",
		},
		{
			name:      "git config user.email absent falls back to OS username",
			commander: fakeGitCommander{configOut: ""},
			username:  func() (string, error) { return "joe", nil },
			want:      "joe",
		},
		{
			name:      "git config fails, username fallback used",
			commander: fakeGitCommander{configErr: errFakeGitFailed},
			username:  func() (string, error) { return "joe", nil },
			want:      "joe",
		},
		{
			name:      "both git config and username fail resolves to empty",
			commander: fakeGitCommander{configErr: errFakeGitFailed},
			username:  func() (string, error) { return "", errFakeGitFailed },
			want:      "",
		},
		{
			name:      "nil username func with no git config resolves to empty",
			commander: fakeGitCommander{configOut: ""},
			username:  nil,
			want:      "",
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := cli.ExportDetectUser(t.Context(), tc.commander, tc.username)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestNewIdentityDeps covers the production Deps->IdentityDeps composition
// end to end: a real temp vault, a note missing identity, backfillIdentity
// driven through newIdentityDeps(deps) rather than a hand-built IdentityDeps
// (TestBackfillIdentity's fixture) — proves the wiring itself works, not
// just the backfill algorithm.
func TestNewIdentityDeps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, "1.2026-01-01.a.md")
	g.Expect(os.WriteFile(notePath, []byte(
		"---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n"+
			"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\n---\n\nbody\n",
	), 0o600)).To(Succeed())

	deps := newTestDeps(io.Discard, io.Discard)
	identityDeps := cli.ExportNewIdentityDeps(deps)

	stamped, err := cli.ExportBackfillIdentity(t.Context(), vault, identityDeps, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(1))

	raw, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).To(ContainSubstring("vault: personal"))
}

// TestNotesMissingIdentityFields covers the notesMissingIdentityFields
// detector: a fact/feedback note missing both user:/vault: flags the vault;
// one that already has them does not; a non-fact/feedback note (e.g. a
// vocab definition note) is never flagged even without them; a missing
// vault directory self-silences to false.
func TestNotesMissingIdentityFields(t *testing.T) {
	t.Parallel()

	const missingIdentityNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\n---\n\nbody\n"

	const stampedNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\nuser: agent@example.com\nvault: personal\n---\n\nbody\n"

	const vocabNote = "---\ntype: term\nterm: recall\ndescription: recall the vault\n---\n\nbody\n"

	table := []struct {
		name  string
		files map[string][]byte
		want  bool
	}{
		{
			name:  "note missing identity flags the vault",
			files: map[string][]byte{"/vault/1.2026-01-01.a.md": []byte(missingIdentityNote)},
			want:  true,
		},
		{
			name:  "already-stamped note does not flag",
			files: map[string][]byte{"/vault/1.2026-01-01.a.md": []byte(stampedNote)},
			want:  false,
		},
		{
			name:  "non-fact/feedback note is never flagged",
			files: map[string][]byte{"/vault/vocab.recall.md": []byte(vocabNote)},
			want:  false,
		},
		{
			name:  "missing vault dir self-silences",
			files: map[string][]byte{},
			want:  false,
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fileSystem := newU1FS()
			maps.Copy(fileSystem.files, tc.files)

			got := cli.ExportNotesMissingIdentityFields("/vault", fileSystem)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestRepoWithProjectFallback covers the backfill-only repo fallback: a
// non-empty project: field wins over a freshly detected repo; an empty
// project: field falls back to the detected value.
func TestRepoWithProjectFallback(t *testing.T) {
	t.Parallel()

	table := []struct {
		name      string
		project   string
		freshRepo string
		want      string
	}{
		{
			name:    "project present is preferred",
			project: "my-project", freshRepo: "git@github.com:x/y.git",
			want: "my-project",
		},
		{
			name:    "project absent falls back to fresh repo",
			project: "", freshRepo: "git@github.com:x/y.git",
			want: "git@github.com:x/y.git",
		},
		{name: "both empty resolves to empty", project: "", freshRepo: "", want: ""},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := cli.ExportRepoWithProjectFallback(tc.project, tc.freshRepo)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestResolveVaultName covers the flag -> env -> default("personal") order,
// mirroring resolveVault's flag/env/default shape for the vault path.
func TestResolveVaultName(t *testing.T) {
	t.Parallel()

	table := []struct {
		name string
		flag string
		env  string
		want string
	}{
		{name: "flag set wins", flag: "work", env: "personal-env", want: "work"},
		{name: "env used when flag empty", flag: "", env: "team", want: "team"},
		{name: "defaults to personal when neither set", flag: "", env: "", want: "personal"},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			getenv := func(key string) string {
				if key == "ENGRAM_VAULT_NAME" {
					return tc.env
				}

				return ""
			}

			got := cli.ExportResolveVaultName(tc.flag, getenv)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// unexported variables.
var (
	errFakeGitFailed = errors.New("git: fake failure")
)

// fakeGitCommander is a scriptable update.Commander fake for testing
// detectRepo/detectUser: it dispatches on the git subcommand (args[0]) and
// returns canned output/error per command, so a single fake can drive every
// origin-present / origin-absent / not-a-git-repo / user.email-present /
// user.email-absent table case.
type fakeGitCommander struct {
	remoteOut   string
	remoteErr   error
	revParseOut string
	revParseErr error
	configOut   string
	configErr   error
}

func (f fakeGitCommander) Run(
	_ context.Context,
	_, _ string,
	args ...string,
) ([]byte, []byte, error) {
	if len(args) == 0 {
		return nil, nil, nil
	}

	switch args[0] {
	case "remote":
		return []byte(f.remoteOut), nil, f.remoteErr
	case "rev-parse":
		return []byte(f.revParseOut), nil, f.revParseErr
	case "config":
		return []byte(f.configOut), nil, f.configErr
	default:
		return nil, nil, nil
	}
}
