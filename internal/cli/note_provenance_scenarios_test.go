package cli_test

// Scenario tests of the note-provenance component (docs/specs/technical/note-provenance.md): repository and user
// detection, the amend re-stamp, the repository for a backfill, the notes a backfill could stamp, and the backfill
// run. Helpers carry the prefix np1.

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792b11c1a2_DetectRepo_WithoutAnOriginTheTopLevelBasenameNamesIt verifies 792b11c1a2: no origin remote and a
// git top level of /home/joe/src/engram resolve the repository to engram.
func Test792b11c1a2_DetectRepo_WithoutAnOriginTheTopLevelBasenameNamesIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx := t.Context()
	got := np1Drive(t, func(commander update.Commander) string {
		return cli.ExportDetectRepo(ctx, np1Getwd, commander)
	}, func(imp *CommanderImp) {
		imp.Run.ArgsEqual(ctx, np1WorkDir, "git", "remote", "get-url", "origin").
			Return(nil, []byte("error: No such remote 'origin'"), errNp1Git)
		imp.Run.ArgsEqual(ctx, np1WorkDir, "git", "rev-parse", "--show-toplevel").
			Return([]byte("/home/joe/src/engram\n"), nil, nil)
	})

	g.Expect(got).To(Equal("engram"))
}

// Test792b11c1a3_DetectRepo_OutsideGitThereIsNoRepository verifies 792b11c1a3: both git commands fail, so the
// repository is empty.
func Test792b11c1a3_DetectRepo_OutsideGitThereIsNoRepository(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx := t.Context()
	got := np1Drive(t, func(commander update.Commander) string {
		return cli.ExportDetectRepo(ctx, np1Getwd, commander)
	}, func(imp *CommanderImp) {
		imp.Run.ArgsEqual(ctx, np1WorkDir, "git", "remote", "get-url", "origin").
			Return(nil, []byte("fatal: not a git repository"), errNp1Git)
		imp.Run.ArgsEqual(ctx, np1WorkDir, "git", "rev-parse", "--show-toplevel").
			Return(nil, []byte("fatal: not a git repository"), errNp1Git)
	})

	g.Expect(got).To(BeEmpty())
}

// Test792b11c1a4_DetectRepo_AnUnresolvableWorkingDirectoryHasNoRepository verifies 792b11c1a4: when the working
// directory cannot be resolved the repository is empty and git is never run.
func Test792b11c1a4_DetectRepo_AnUnresolvableWorkingDirectoryHasNoRepository(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got := np1Drive(t, func(commander update.Commander) string {
		return cli.ExportDetectRepo(t.Context(), func() (string, error) { return "", errNp1Git }, commander)
	}, func(*CommanderImp) {})

	g.Expect(got).To(BeEmpty())
}

// Test792b11c2a2_DetectUser_TheOSUsernameIsTheFallback verifies 792b11c2a2: git config user.email printing nothing
// or failing leaves the OS username joe as the user.
func Test792b11c2a2_DetectUser_TheOSUsernameIsTheFallback(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		out []byte
		err error
	}{
		"git config prints nothing": {out: []byte("  \n")},
		"git config fails":          {err: errNp1Git},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			ctx := t.Context()
			got := np1Drive(t, func(commander update.Commander) string {
				return cli.ExportDetectUser(ctx, commander, func() (string, error) { return "joe", nil })
			}, func(imp *CommanderImp) {
				imp.Run.ArgsEqual(ctx, "", "git", "config", "user.email").Return(tc.out, nil, tc.err)
			})

			g.Expect(got).To(Equal("joe"))
		})
	}
}

// Test792b11c2a3_DetectUser_NoSourceResolves verifies 792b11c2a3: git config user.email failing and the OS username
// lookup failing, or being unavailable, resolve the user to empty.
func Test792b11c2a3_DetectUser_NoSourceResolves(t *testing.T) {
	t.Parallel()

	cases := map[string]func() (string, error){
		"the username lookup fails":   func() (string, error) { return "", errNp1Git },
		"the username is unavailable": nil,
	}

	for name, username := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			ctx := t.Context()
			got := np1Drive(t, func(commander update.Commander) string {
				return cli.ExportDetectUser(ctx, commander, username)
			}, func(imp *CommanderImp) {
				imp.Run.ArgsEqual(ctx, "", "git", "config", "user.email").Return(nil, nil, errNp1Git)
			})

			g.Expect(got).To(BeEmpty())
		})
	}
}

// Test792b11c4a1_RunAmend_AllThreeAreOverwritten verifies 792b11c4a1: re-stamping a note with repo old-repo, user
// old-user and vault old-vault, from an environment resolving repository new-repo, user bob and vault name new-vault,
// leaves repo new-repo, user bob and vault new-vault. The amend is the observable re-stamp; the existing
// TestIdentityStamp_Stamp_OverwritesAllThree blanks the user instead of setting bob, so it is not renamed.
func Test792b11c4a1_RunAmend_AllThreeAreOverwritten(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const basename = "1aa.2026-01-01.restamp"

	note := []byte("---\ntype: fact\ntier: L2\nsituation: ctx\nsubject: A\npredicate: has\nobject: old\n" +
		"luhmann: \"1aa\"\ncreated: \"2026-01-01\"\nsource: test\n" +
		"repo: old-repo\nuser: old-user\nvault: old-vault\n---\n\nInformation learned: when in ctx, A has old.\n\n")

	var written []byte

	deps := cli.AmendDeps{
		DetectRepo: func(context.Context) string { return "new-repo" },
		DetectUser: func(context.Context) string { return "bob" },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".md") {
				return note, nil
			}

			return []byte(`{"last_used":"2026-01-01"}`), nil
		},
		Write: func(path string, data []byte) error {
			if strings.HasSuffix(path, ".md") {
				written = data
			}

			return nil
		},
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", VaultName: "new-vault", Object: "new"}

	err := cli.ExportRunAmend(t.Context(), args, deps, &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	frontmatter := decodeBackfillFrontmatter(t, string(written))
	g.Expect(frontmatter).To(HaveKeyWithValue("repo", "new-repo"))
	g.Expect(frontmatter).To(HaveKeyWithValue("user", "bob"))
	g.Expect(frontmatter).To(HaveKeyWithValue("vault", "new-vault"))
	g.Expect(frontmatter).To(HaveKeyWithValue("object", "new"), "the amend itself happened")
}

// Test792b11c5a2_BackfillIdentity_WithoutAProjectTheDetectionIsUsed verifies 792b11c5a2: a note with no project: and
// a detected repository is stamped with the detected repository.
func Test792b11c5a2_BackfillIdentity_WithoutAProjectTheDetectionIsUsed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newNp1Vault(map[string]string{"1.2026-01-01.a.md": np1MissingIdentityNote})

	stamped, err := vault.backfill(t.Context(), "git@github.com:example/detected.git")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(1))
	g.Expect(cli.ExportRepoWithProjectFallback("", "git@github.com:example/detected.git")).
		To(Equal("git@github.com:example/detected.git"))
	g.Expect(decodeBackfillFrontmatter(t, vault.files["/vault/1.2026-01-01.a.md"])).
		To(HaveKeyWithValue("repo", "git@github.com:example/detected.git"))
}

// Test792b11c5a3_BackfillIdentity_NeitherResolves verifies 792b11c5a3: a note with no project: and no detected
// repository is stamped with an empty repository (no repo: value).
func Test792b11c5a3_BackfillIdentity_NeitherResolves(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newNp1Vault(map[string]string{"1.2026-01-01.a.md": np1MissingIdentityNote})

	stamped, err := vault.backfill(t.Context(), "")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(1), "the note is still stamped with user: and vault:")
	g.Expect(cli.ExportRepoWithProjectFallback("", "")).To(BeEmpty())

	frontmatter := decodeBackfillFrontmatter(t, vault.files["/vault/1.2026-01-01.a.md"])
	g.Expect(frontmatter["repo"]).To(Or(BeNil(), BeEmpty()))
	g.Expect(frontmatter).To(HaveKeyWithValue("user", "bob@example.com"))
}

// Test792b11c6a2_NotesMissingIdentityFields_ACRLFNoteIsFound verifies 792b11c6a2: a fact note missing identity whose
// lines end in \r\n makes the vault hold a note a backfill could stamp.
func Test792b11c6a2_NotesMissingIdentityFields_ACRLFNoteIsFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(toCRLF(np1MissingIdentityNote))

	g.Expect(strings.Contains(string(fileSystem.files["/vault/1.2026-01-01.a.md"]), "\r\n")).To(BeTrue())
	g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, true)).To(BeTrue())
}

// Test792b11c6a3_NotesMissingIdentityFields_AStampedNoteIsNotFound verifies 792b11c6a3: a note that has user: does
// not make the vault hold a stampable note.
func Test792b11c6a3_NotesMissingIdentityFields_AStampedNoteIsNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(
		strings.Replace(np1MissingIdentityNote, "source: agent\n", "source: agent\nuser: agent@example.com\n", 1))

	g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, true)).To(BeFalse())
}

// Test792b11c6a4_NotesMissingIdentityFields_OtherNoteTypesAreNotFound verifies 792b11c6a4: a note that is neither
// fact nor feedback and has no user: does not make the vault hold a stampable note.
func Test792b11c6a4_NotesMissingIdentityFields_OtherNoteTypesAreNotFound(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"a vocabulary term": "---\ntype: term\nterm: recall\ndescription: recall the vault\n---\n\nbody\n",
		"a runbook": "---\ntype: runbook\nsituation: s\ndone_when: d\nluhmann: \"1\"\n" +
			"created: 2026-01-01\n---\n\n1. step\n",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fileSystem := newU1FS()
			fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(content)

			g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, true)).To(BeFalse())
		})
	}
}

// Test792b11c6a5_NotesMissingIdentityFields_AnUnreadableVaultIsNotFound verifies 792b11c6a5: a vault directory that
// cannot be read, or a file in it that cannot be read, is judged to hold no stampable note.
func Test792b11c6a5_NotesMissingIdentityFields_AnUnreadableVaultIsNotFound(t *testing.T) {
	t.Parallel()

	t.Run("the directory cannot be read", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fileSystem := np1UnreadableFS{u1FS: newU1FS(), dirErr: errNp1Denied}
		fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(np1MissingIdentityNote)

		g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, true)).To(BeFalse())
	})

	t.Run("a file cannot be read", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fileSystem := np1UnreadableFS{u1FS: newU1FS(), fileErr: errNp1Denied}
		fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(np1MissingIdentityNote)

		g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, true)).To(BeFalse())
	})
}

// Test792b11c6a6_NotesMissingIdentityFields_AVaultNoteWithNoResolvableUserIsNotFound verifies 792b11c6a6: a note with
// vault: but no user:, while no user resolves, does not make the vault hold a stampable note (and does once a user
// resolves).
func Test792b11c6a6_NotesMissingIdentityFields_AVaultNoteWithNoResolvableUserIsNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(
		strings.Replace(np1MissingIdentityNote, "source: agent\n", "source: agent\nvault: work\n", 1))

	g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, false)).To(BeFalse())
	g.Expect(cli.ExportNotesMissingIdentityFields("/vault", fileSystem, true)).To(BeTrue(),
		"the same note is found once a user resolves, so the false above is the user's doing")
}

// Test792b11c7a12_ApplyIdentityBackfill_TheReportRecordsTheRun verifies 792b11c7a12: a backfill that stamped notes
// records the count and re-checks the missing-identity flag; under dry run the flag is left as it was.
func Test792b11c7a12_ApplyIdentityBackfill_TheReportRecordsTheRun(t *testing.T) {
	t.Parallel()

	t.Run("a real run records the count and re-checks the flag", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fileSystem := newU1FS()
		fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(np1MissingIdentityNote)
		report := &update.Report{VaultHasNotesMissingIdentity: true}

		err := cli.ExportApplyIdentityBackfill(t.Context(), "/vault", np1DepsOver(fileSystem), false, fileSystem, report)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(report.IdentityBackfillRan).To(BeTrue())
		g.Expect(report.IdentityBackfillNotesStamped).To(Equal(1))
		g.Expect(report.VaultHasNotesMissingIdentity).To(BeFalse(), "the flag was true before and is re-checked")
	})

	t.Run("a dry run records the count and leaves the flag as it was", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fileSystem := newU1FS()
		fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(np1MissingIdentityNote)
		report := &update.Report{VaultHasNotesMissingIdentity: false}

		err := cli.ExportApplyIdentityBackfill(t.Context(), "/vault", np1DepsOver(fileSystem), true, fileSystem, report)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(report.IdentityBackfillRan).To(BeTrue())
		g.Expect(report.IdentityBackfillNotesStamped).To(Equal(1), "the would-stamp count")
		g.Expect(report.VaultHasNotesMissingIdentity).To(BeFalse(),
			"a re-check would find the unstamped note and say true; it stays as it was")
	})
}

// Test792b11c7a2_BackfillIdentity_ASecondBackfillWritesNothing verifies 792b11c7a2: backfilling a vault that has
// been backfilled writes no note and counts 0.
func Test792b11c7a2_BackfillIdentity_ASecondBackfillWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newNp1Vault(map[string]string{
		"1.2026-01-01.a.md": np1MissingIdentityNote,
		"2.2026-01-02.b.md": strings.Replace(np1MissingIdentityNote, "luhmann: \"1\"", "luhmann: \"2\"", 1),
	})

	first, firstErr := vault.backfill(t.Context(), "repo")
	g.Expect(firstErr).NotTo(HaveOccurred())
	g.Expect(first).To(Equal(2))
	g.Expect(vault.writes).To(Equal(2))

	afterFirst := map[string]string{}
	maps.Copy(afterFirst, vault.files)

	second, secondErr := vault.backfill(t.Context(), "repo")
	g.Expect(secondErr).NotTo(HaveOccurred())
	g.Expect(second).To(BeZero())
	g.Expect(vault.writes).To(Equal(2), "no note is written by the second run")
	g.Expect(vault.files).To(Equal(afterFirst))
}

// unexported constants.
const (
	np1MissingIdentityNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\n---\n\nbody\n"
	np1WorkDir = "/work/checkout"
)

// unexported variables.
var (
	errNp1Denied = errors.New("permission denied")
	errNp1Git    = errors.New("git: exit status 128")
)

// np1UnreadableFS is a u1FS whose directory read or file read fails.
type np1UnreadableFS struct {
	*u1FS

	dirErr  error
	fileErr error
}

func (f np1UnreadableFS) ReadDir(path string) ([]update.DirEntry, error) {
	if f.dirErr != nil {
		return nil, f.dirErr
	}

	return f.u1FS.ReadDir(path)
}

// ReadFile returns the file's bytes together with fileErr, so a caller that ignored the error would see a stampable
// note.
func (f np1UnreadableFS) ReadFile(path string) ([]byte, error) {
	raw, err := f.u1FS.ReadFile(path)
	if f.fileErr != nil {
		return raw, f.fileErr
	}

	return raw, err
}

// np1Vault is an in-memory vault for backfill tests: path to content, and a count of writes.
type np1Vault struct {
	files  map[string]string
	writes int
}

// backfill runs a real backfill over the vault with the given detected repository and the user bob@example.com.
func (v *np1Vault) backfill(ctx context.Context, detectedRepo string) (int, error) {
	deps := cli.IdentityDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			names := make([]string, 0, len(v.files))
			for path := range v.files {
				names = append(names, strings.TrimPrefix(path, "/vault/"))
			}

			return names, nil
		},
		ReadFile: func(path string) ([]byte, error) { return []byte(v.files[path]), nil },
		WriteFile: func(path string, data []byte) error {
			v.writes++
			v.files[path] = string(data)

			return nil
		},
		DetectRepo: func(context.Context) string { return detectedRepo },
		DetectUser: func(context.Context) string { return "bob@example.com" },
		Getenv:     func(string) string { return "" },
	}

	return cli.ExportBackfillIdentity(ctx, "/vault", deps, false)
}

func newNp1Vault(notes map[string]string) *np1Vault {
	files := make(map[string]string, len(notes))
	for name, content := range notes {
		files["/vault/"+name] = content
	}

	return &np1Vault{files: files}
}

// np1DepsOver builds IdentityDeps reading and writing the files of fileSystem, detecting a repository and the user
// agent@example.com.
func np1DepsOver(fileSystem *u1FS) cli.IdentityDeps {
	return cli.IdentityDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"1.2026-01-01.a.md"}, nil
		},
		ReadFile:   func(path string) ([]byte, error) { return fileSystem.files[path], nil },
		WriteFile:  func(path string, data []byte) error { fileSystem.files[path] = data; return nil },
		DetectRepo: func(context.Context) string { return "git@github.com:example/vault.git" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}
}

// np1Drive runs call against a mock Commander in a goroutine while script plays the expected git commands, and
// returns what call returned; it fails the test when call does not return in time.
func np1Drive(t *testing.T, call func(update.Commander) string, script func(imp *CommanderImp)) string {
	t.Helper()

	commander, imp := MockCommander(t)
	done := make(chan string, 1)

	go func() { done <- call(commander) }()

	script(imp)

	select {
	case got := <-done:
		return got
	case <-time.After(mockTimeout):
		t.Fatal("the detection did not return after the scripted git answers")

		return ""
	}
}

func np1Getwd() (string, error) { return np1WorkDir, nil }
