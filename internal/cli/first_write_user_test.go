package cli_test

// First writes omit an undetectable user (#789 follow-up, design D6; spec
// vault-note-identity "User field auto-detected at note creation"): when
// user detection comes back empty, learn and pull-down write the note with
// no user: key and print one warning — never user: "".

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// TestActivate_PullDownOmitsEmptyUser: a pull-down whose user detection is
// empty writes the copy without user: and warns once.
func TestActivate_PullDownOmitsEmptyUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := pulledFact("7.2026-09-01.pulled", "c", "")

	env := newWiringEnv(t)
	env.wrap = withoutUserDetection
	env.parent.addNote(note)

	_, stderr := env.run("activate", "--note", note.basename+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	raw := readFileString(t, filepath.Join(env.vault, files[0]))
	frontmatter, _, _ := strings.Cut(strings.TrimPrefix(raw, "---\n"), "\n---\n")
	g.Expect(frontmatter).NotTo(MatchRegexp(`(?m)^user:`), "no top-level user: key")
	g.Expect(strings.Count(stderr, "user detection resolved empty")).To(Equal(1))
}

// TestLearn_OmitsEmptyUser: a learn whose user detection is empty writes
// the note without user: and warns once.
func TestLearn_OmitsEmptyUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		written  []byte
		warnings []string
	)

	args := cli.LearnArgs{
		Type: "fact", Slug: "no-user", Vault: t.TempDir(), Position: "top", Source: "test",
		Situation: "no user detected", Subject: "A", Predicate: "has", Object: "B", VaultName: "personal",
	}
	deps := learnDepsForUserTest(&written)
	deps.LogWarning = func(format string, _ ...any) { warnings = append(warnings, format) }

	err := cli.ExportRunLearn(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(written)).NotTo(ContainSubstring("user:"))
	g.Expect(string(written)).To(ContainSubstring("vault: personal"))
	g.Expect(warnings).To(HaveLen(1))
	g.Expect(strings.Join(warnings, "")).To(ContainSubstring("user detection resolved empty"))
}

// TestLearn_StampsDetectedUser: a detected user is still stamped, with no
// warning.
func TestLearn_StampsDetectedUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		written  []byte
		warnings int
	)

	args := cli.LearnArgs{
		Type: "fact", Slug: "user", Vault: t.TempDir(), Position: "top", Source: "test",
		Situation: "user detected", Subject: "A", Predicate: "has", Object: "B", VaultName: "personal",
	}
	deps := learnDepsForUserTest(&written)
	deps.DetectUser = func(context.Context) string { return "bob@example.com" }
	deps.LogWarning = func(string, ...any) { warnings++ }

	err := cli.ExportRunLearn(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(written)).To(ContainSubstring("user: bob@example.com\n"))
	g.Expect(warnings).To(BeZero())
}

// unexported variables.
var (
	errNoUser = errors.New("no user")
)

// noUserCommander fails `git config user.email` and passes every other
// command through, so user detection resolves empty.
type noUserCommander struct {
	inner update.Commander
}

func (c noUserCommander) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	if name == "git" && len(args) == 2 && args[0] == "config" && args[1] == "user.email" {
		return nil, nil, errNoUser
	}

	return c.inner.Run(ctx, dir, name, args...)
}

func learnDepsForUserTest(written *[]byte) cli.LearnDeps {
	return cli.LearnDeps{
		DetectRepo:    func(context.Context) string { return "" },
		DetectUser:    func(context.Context) string { return "" },
		Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(_ string, data []byte) error { *written = data; return nil },
	}
}

// withoutUserDetection makes both user detection sources fail.
func withoutUserDetection(deps *cli.Deps) {
	deps.Commander = noUserCommander{inner: deps.Commander}
	deps.Username = func() (string, error) { return "", errNoUser }
}
