package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// TestRunServe_CorruptVaultIDWarnsAndStarts: a corrupt or empty ID file (a
// partial write, a git conflict) never stops serve — it warns, naming
// `engram vault-id --regenerate`, and serves; the file is left alone.
func TestRunServe_CorruptVaultIDWarnsAndStarts(t *testing.T) {
	t.Parallel()

	for _, content := range []string{"", "<<<<<<< HEAD\n"} {
		g := NewWithT(t)

		vault := t.TempDir()
		idPath := filepath.Join(vault, ".engram-vault-id")
		g.Expect(os.WriteFile(idPath, []byte(content), 0o600)).To(Succeed())

		var stderr bytes.Buffer

		served := false
		args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: vault, VaultName: "personal", ChunksDir: t.TempDir()}

		g.Expect(cli.RunServe(context.Background(), args, fakeServeDeps(&stderr, 1, &served))).To(Succeed())
		g.Expect(served).To(BeTrue())
		g.Expect(stderr.String()).To(ContainSubstring("engram vault-id --regenerate"))
		g.Expect(readFileString(t, idPath)).To(Equal(content))
	}
}

// TestRunServe_StampsMissingVaultIDAndReusesIt: serve stamps an existing
// vault's missing ID at startup; a later start reuses it unchanged.
func TestRunServe_StampsMissingVaultIDAndReusesIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: vault, VaultName: "personal", ChunksDir: t.TempDir()}

	var stderr bytes.Buffer

	g.Expect(cli.RunServe(context.Background(), args, fakeServeDeps(&stderr, 8, nil))).To(Succeed())
	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(8) + "\n"))
	g.Expect(stderr.String()).To(BeEmpty())

	g.Expect(cli.RunServe(context.Background(), args, fakeServeDeps(&stderr, 99, nil))).To(Succeed())
	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(8) + "\n"))
	g.Expect(stderr.String()).To(BeEmpty())
}

// TestRunServe_WarnsOnLocationMismatchAndStartsAnyway: serve never refuses
// over a failed location check — it warns once and serves.
func TestRunServe_WarnsOnLocationMismatchAndStartsAnyway(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	moved := filepath.Join(t.TempDir(), "moved")
	g.Expect(os.Rename(vault, moved)).To(Succeed())

	var stderr bytes.Buffer

	served := false
	args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: moved, VaultName: "personal", ChunksDir: t.TempDir()}

	g.Expect(cli.RunServe(context.Background(), args, fakeServeDeps(&stderr, 1, &served))).To(Succeed())
	g.Expect(served).To(BeTrue())

	lines := nonEmptyLines(stderr.String())
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(ContainSubstring("engram vault-id --regenerate"))
	g.Expect(lines[0]).To(ContainSubstring("engram vault-id --claim"))
}

// TestTargets_AmendActivateMissingVaultFailOnNotFound: the spec's "never on a
// vault lock error" — amend reports its missing target.
func TestTargets_AmendActivateMissingVaultFailOnNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")

	_, stderr := runWithRand(t, []string{
		"engram", "amend", "--vault", vault, "--target", "nope", "--situation", "s", "--chunks-dir", t.TempDir(),
	}, 1)
	g.Expect(stderr).To(ContainSubstring("not found"))
	g.Expect(strings.ToLower(stderr)).NotTo(ContainSubstring("lock"))
}

// TestTargets_EveryVaultCommandCreatesMissingVault pins design D2's
// first-use creation through dispatch: every vault-resolving command on a
// missing path creates the starters, the vault ID and the self-ignoring
// .engram/, printing exactly one creation line; a second run creates and
// prints nothing. amend/activate/resituate/show fail only on their own
// merits (not found), never on a lock error.
func TestTargets_EveryVaultCommandCreatesMissingVault(t *testing.T) {
	t.Parallel()

	commands := []struct {
		name string
		args func(vault, scratch string) []string
	}{
		{"query", func(v, s string) []string { return []string{"query", "--phrase", "x", "--vault", v, "--chunks-dir", s} }},
		{"show", func(v, _ string) []string { return []string{"show", "nope", "--vault", v} }},
		{"amend", func(v, s string) []string {
			return []string{"amend", "--vault", v, "--target", "nope", "--situation", "s", "--chunks-dir", s}
		}},
		{"activate", func(v, _ string) []string { return []string{"activate", "--vault", v, "--note", "nope.md"} }},
		{"resituate", func(v, _ string) []string {
			return []string{"resituate", "--vault", v, "--note", "nope", "--situation", "s"}
		}},
		{"count", func(v, _ string) []string { return []string{"count", "--vault", v, "--group-by", "type"} }},
		{"check", func(v, _ string) []string { return []string{"check", "--vault", v} }},
		{"register-skills", func(v, s string) []string {
			return []string{"register-skills", "--vault", v, "--skills-dir", s, "--dry-run"}
		}},
		{"embed-status", func(v, _ string) []string { return []string{"embed", "status", "--vault", v} }},
		{"vocab-stats", func(v, _ string) []string { return []string{"vocab", "stats", "--vault", v} }},
		{"vault-id", func(v, _ string) []string { return []string{"vault-id", "--vault", v} }},
		{"ingest", func(v, s string) []string {
			return []string{"ingest", "--vault", v, "--chunks-dir", s, "--sweep", s}
		}},
		{"learn-fact", func(v, _ string) []string {
			return []string{
				"learn", "fact", "--vault", v, "--situation", "s", "--source", "test",
				"--subject", "a", "--predicate", "b", "--object", "c",
			}
		}},
	}

	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := filepath.Join(t.TempDir(), "vault")
			args := append([]string{"engram"}, command.args(vault, t.TempDir())...)

			_, firstErr := runWithRand(t, args, 3)
			g.Expect(strings.Count(firstErr, createdVaultLine)).To(Equal(1), firstErr)
			g.Expect(firstErr).To(ContainSubstring(createdVaultLine + vault + "\n"))
			g.Expect(strings.ToLower(firstErr)).NotTo(ContainSubstring("lock"))
			g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(3) + "\n"))
			g.Expect(readFileString(t, filepath.Join(vault, ".engram", ".gitignore"))).To(Equal("*\n"))
			g.Expect(filepath.Join(vault, ".obsidian", "app.json")).To(BeARegularFile())

			_, secondErr := runWithRand(t, args, 90)
			g.Expect(secondErr).NotTo(ContainSubstring(createdVaultLine))
			g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(3) + "\n"))
		})
	}
}

// TestTargets_QueryMissingVaultReturnsEmptyResult: the spec's "A read
// command creates the vault" — the query then returns an empty result
// without error.
func TestTargets_QueryMissingVaultReturnsEmptyResult(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")

	stdout, stderr := runWithRand(t, []string{
		"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir(),
	}, 1)
	g.Expect(stderr).To(Equal(createdVaultLine + vault + "\n"))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(BeEmpty())
}

// TestTargets_QueryWithParent_StampsIDAndWarnsOnLocationMismatch: the first
// parent contact stamps a missing vault ID (no warning, the record is
// fresh); a vault whose location record is missing (a clone) still runs the
// merged query but prints the one remedies warning.
func TestTargets_QueryWithParent_StampsIDAndWarnsOnLocationMismatch(t *testing.T) {
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	args := []string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()}
	unreachable := func(d *cli.Deps) {
		d.RandRead = seqRand(4)
		d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
		d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
			return cli.FetchResponse{}, errors.New("connection refused")
		}
	}

	_, stderr := executeCapturingBoth(t, args, unreachable)
	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(4) + "\n"))
	g.Expect(stderr).NotTo(ContainSubstring("vault-id"))

	g.Expect(os.Remove(filepath.Join(vault, ".engram", "home.json"))).To(Succeed())

	stdout, stderr := executeCapturingBoth(t, args, unreachable)
	g.Expect(strings.Count(stderr, "engram vault-id --claim")).To(Equal(1))
	g.Expect(stderr).To(ContainSubstring("engram vault-id --regenerate"))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))
}

// TestTargets_QueryWithParent_UnstampableIDOnlyWarns: a vault whose ID file
// is corrupt cannot be stamped at parent contact; the query warns and still
// runs rather than failing.
func TestTargets_QueryWithParent_UnstampableIDOnlyWarns(t *testing.T) {
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "3.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: y\n---\n\nanother local body\n", []float32{1, 0, 0, 0})
	g.Expect(os.WriteFile(filepath.Join(vault, ".engram-vault-id"), []byte("junk\n"), 0o600)).To(Succeed())

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.RandRead = seqRand(4)
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{}, errors.New("connection refused")
			}
		})
	g.Expect(stderr).To(ContainSubstring("could not stamp the vault ID"))
	g.Expect(stderr).To(ContainSubstring("engram vault-id --regenerate"))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))
}

// TestTargets_ServeCreatesMissingVault: serve is a vault-resolving command
// like the rest — a missing vault is created (one line) before it binds.
func TestTargets_ServeCreatesMissingVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")

	var stderr bytes.Buffer

	served := false
	deps := fakeServeDeps(&stderr, 2, &served)

	err := targExecute([]string{
		"engram", "serve", "--addr", "127.0.0.1:0", "--vault", vault, "--chunks-dir", t.TempDir(),
	}, deps)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(served).To(BeTrue())
	g.Expect(stderr.String()).To(Equal(createdVaultLine + vault + "\n"))
	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(2) + "\n"))
}

// TestTargets_StarterReadmeHiddenUserReadmeVisible: the untouched starter
// README (identified by content) is invisible to check and embed status, so
// a fresh vault is clean; a user's own README.md stays a visible note.
func TestTargets_StarterReadmeHiddenUserReadmeVisible(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)

	checkOut, checkErr := runWithRand(t, []string{"engram", "check", "--vault", vault}, 1)
	g.Expect(checkErr).To(BeEmpty())
	g.Expect(checkOut).NotTo(ContainSubstring("missing a sidecar"))

	statusOut, _ := runWithRand(t, []string{"engram", "embed", "status", "--vault", vault}, 1)
	g.Expect(statusOut).To(MatchRegexp(`total: +0\n`))
	g.Expect(statusOut).To(MatchRegexp(`without: +0\n`))

	g.Expect(os.WriteFile(filepath.Join(vault, "README.md"), []byte("# my notes\n"), 0o600)).To(Succeed())

	userOut, _ := runWithRand(t, []string{"engram", "embed", "status", "--vault", vault}, 1)
	g.Expect(userOut).To(MatchRegexp(`total: +1\n`))

	userCheck, _ := runWithRand(t, []string{"engram", "check", "--vault", vault}, 1)
	g.Expect(userCheck).To(ContainSubstring("1 note(s) missing a sidecar"))
}

// TestTargets_VaultID covers `engram vault-id [--regenerate | --claim]`: no
// flag prints the ID and the location check; --regenerate mints a new ID;
// --claim fixes a moved vault's record; both flags together are refused.
func TestTargets_VaultID(t *testing.T) {
	t.Parallel()

	t.Run("no-flag-prints-id-and-check", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := createdVault(t)
		stdout, stderr := runWithRand(t, []string{"engram", "vault-id", "--vault", vault}, 1)
		g.Expect(stderr).To(BeEmpty())
		g.Expect(stdout).To(ContainSubstring("vault_id: " + seqID(1)))
		g.Expect(stdout).To(ContainSubstring("location: ok"))
	})

	t.Run("no-flag-reports-mismatch-with-remedies", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := createdVault(t)
		moved := filepath.Join(t.TempDir(), "moved")
		g.Expect(os.Rename(vault, moved)).To(Succeed())

		stdout, _ := runWithRand(t, []string{"engram", "vault-id", "--vault", moved}, 1)
		g.Expect(stdout).To(ContainSubstring("location: mismatch"))
		g.Expect(stdout).To(ContainSubstring("engram vault-id --claim"))
		g.Expect(stdout).To(ContainSubstring("engram vault-id --regenerate"))
	})

	t.Run("no-flag-reads-without-writing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := t.TempDir()
		before := treeSnapshot(t, vault)

		stdout, _ := runWithRand(t, []string{"engram", "vault-id", "--vault", vault}, 1)
		g.Expect(stdout).To(ContainSubstring("location: no vault id"))
		g.Expect(treeSnapshot(t, vault)).To(Equal(before))
	})

	t.Run("regenerate", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := createdVault(t)
		stdout, stderr := runWithRand(t, []string{"engram", "vault-id", "--vault", vault, "--regenerate"}, 40)
		g.Expect(stderr).To(BeEmpty())
		g.Expect(stdout).To(ContainSubstring("vault_id: " + seqID(40)))
		g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(40) + "\n"))
	})

	t.Run("claim", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := createdVault(t)
		moved := filepath.Join(t.TempDir(), "moved")
		g.Expect(os.Rename(vault, moved)).To(Succeed())

		_, stderr := runWithRand(t, []string{"engram", "vault-id", "--vault", moved, "--claim"}, 40)
		g.Expect(stderr).To(BeEmpty())

		stdout, _ := runWithRand(t, []string{"engram", "vault-id", "--vault", moved}, 40)
		g.Expect(stdout).To(ContainSubstring("vault_id: " + seqID(1)))
		g.Expect(stdout).To(ContainSubstring("location: ok"))
	})

	t.Run("errors-surface", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		noID := t.TempDir()
		_, claimErr := runWithRand(t, []string{"engram", "vault-id", "--vault", noID, "--claim"}, 1)
		g.Expect(claimErr).To(ContainSubstring("no vault ID"))

		corrupt := t.TempDir()
		g.Expect(os.WriteFile(filepath.Join(corrupt, ".engram-vault-id"), []byte("junk\n"), 0o600)).To(Succeed())
		_, printErr := runWithRand(t, []string{"engram", "vault-id", "--vault", corrupt}, 1)
		g.Expect(printErr).To(ContainSubstring("invalid vault id"))

		underFile := filepath.Join(t.TempDir(), "file")
		g.Expect(os.WriteFile(underFile, []byte("x"), 0o600)).To(Succeed())
		_, regenErr := runWithRand(t, []string{"engram", "vault-id", "--vault", underFile, "--regenerate"}, 1)
		g.Expect(regenErr).NotTo(BeEmpty())
	})

	t.Run("both-flags-refused", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := createdVault(t)
		_, stderr := runWithRand(t, []string{"engram", "vault-id", "--vault", vault, "--regenerate", "--claim"}, 40)
		g.Expect(stderr).To(ContainSubstring("--regenerate"))
		g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(1) + "\n"))
	})
}

// TestVaultIDUncommitted pins `engram update`'s notify-only detector: true
// only when the ID file exists and git reports it untracked/uncommitted.
func TestVaultIDUncommitted(t *testing.T) {
	t.Parallel()

	const (
		repoArgs   = "rev-parse --is-inside-work-tree"
		lsArgs     = "ls-files --error-unmatch -- .engram-vault-id"
		statusArgs = "status --porcelain -- .engram-vault-id"
	)

	inRepo := scriptedGitReply{out: "true\n"}
	tracked := scriptedGitReply{out: ".engram-vault-id\n"}

	table := []struct {
		name   string
		hasID  bool
		script scriptedGit
		want   bool
	}{
		{"untracked", true, scriptedGit{repoArgs: inRepo, statusArgs: {out: "?? .engram-vault-id\n"}}, true},
		{"ignored", true, scriptedGit{repoArgs: inRepo, statusArgs: {out: ""}}, true},
		{"staged-not-committed", true, scriptedGit{
			repoArgs: inRepo, lsArgs: tracked, statusArgs: {out: "A  .engram-vault-id\n"},
		}, true},
		{"committed", true, scriptedGit{repoArgs: inRepo, lsArgs: tracked, statusArgs: {out: ""}}, false},
		{"status-fails", true, scriptedGit{repoArgs: inRepo, lsArgs: tracked}, false},
		{"not-a-git-repo", true, scriptedGit{}, false},
		{"no-id-file", false, scriptedGit{repoArgs: inRepo}, false},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fileSystem := newU1FS()
			fileSystem.dirs["/vault"] = true

			if tc.hasID {
				fileSystem.files["/vault/.engram-vault-id"] = []byte(seqID(1) + "\n")
			}

			got := cli.ExportVaultIDUncommitted(context.Background(), "/vault", fileSystem, tc.script)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestVaultIDUncommitted_RealGitIgnoredAndCommitted drives the detector
// against real git: an ignored ID file is still flagged (git status alone
// hides it), and a committed one is not.
func TestVaultIDUncommitted_RealGitIgnoredAndCommitted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	commander := newTestDeps(io.Discard, io.Discard).Commander
	fileSystem := cli.ExportUpdateFSFromEdge(realFSForTest())
	gitIn := func(args ...string) {
		_, stderr, err := commander.Run(t.Context(), vault, "git", args...)
		g.Expect(err).NotTo(HaveOccurred(), string(stderr))
	}

	gitIn("init", "-q")
	g.Expect(os.WriteFile(filepath.Join(vault, ".gitignore"), []byte(".engram-vault-id\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, ".engram-vault-id"), []byte(seqID(1)+"\n"), 0o600)).To(Succeed())

	g.Expect(cli.ExportVaultIDUncommitted(t.Context(), vault, fileSystem, commander)).To(BeTrue())

	gitIn("add", "-f", ".engram-vault-id")
	gitIn("-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "vault: add vault id")

	g.Expect(cli.ExportVaultIDUncommitted(t.Context(), vault, fileSystem, commander)).To(BeFalse())
}

// TestWriteUpdateReport_VaultIDUncommittedNotice: the report asks for the
// one-time vault commit only while the flag is set.
func TestWriteUpdateReport_VaultIDUncommittedNotice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var flagged bytes.Buffer

	g.Expect(cli.ExportWriteUpdateReport(&flagged, update.Report{VaultIDUncommitted: true})).To(Succeed())
	g.Expect(flagged.String()).To(ContainSubstring(".engram-vault-id"))
	g.Expect(flagged.String()).To(ContainSubstring("commit"))

	var clean bytes.Buffer

	g.Expect(cli.ExportWriteUpdateReport(&clean, update.Report{})).To(Succeed())
	g.Expect(clean.String()).NotTo(ContainSubstring(".engram-vault-id"))
}

// unexported constants.
const (
	createdVaultLine = "engram: created new vault at "
)

// fakeServeDeps returns test deps whose HTTP server never binds: the mux and
// route registration are no-ops, and ListenAndServe records that it ran.
func fakeServeDeps(stderr io.Writer, start byte, served *bool) cli.Deps {
	deps := exchangeDeps(stderr, start)
	deps.NewServeMux = func() cli.RawServeMux { return "fake-mux" }
	deps.RegisterRoute = func(cli.RawServeMux, string, string, cli.ServeHandler) {}
	deps.ListenAndServe = func(context.Context, cli.RawServeMux, string) error {
		if served != nil {
			*served = true
		}

		return nil
	}

	return deps
}

// runWithRand runs args through Targets with deterministic RandRead
// seqRand(start) and a fixed-vector embedder, returning stdout and stderr.
func runWithRand(t *testing.T, args []string, start byte) (string, string) {
	t.Helper()

	return executeCapturingBoth(t, args, func(d *cli.Deps) {
		d.RandRead = seqRand(start)
		d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
	})
}

// targExecute runs args through Targets(deps), returning targ's own error.
func targExecute(args []string, deps cli.Deps) error {
	_, err := targ.Execute(args, cli.Targets(deps)...)

	return err
}
