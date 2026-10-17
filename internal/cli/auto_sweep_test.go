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
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/internal/cli"
)

//go:generate impgen cli.IngestRun --dependency --import-path github.com/toejough/engram/internal/cli

func Test793d4e1d_Targets_AutoSweepCreatesNoVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "no-such-vault")
	sessions, chunks, home := t.TempDir(), t.TempDir(), t.TempDir()
	plantSessionRecord(t, sessions, "a-project", `{"type":"user","message":{"role":"user","content":"hello"}}`)

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "auto", "sweep", "--sessions", sessions, "--chunks-dir", chunks},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(name string) string {
				if name == "ENGRAM_VAULT_PATH" {
					return vault
				}

				return ""
			}
			d.UserHomeDir = func() (string, error) { return home, nil }
			d.Stdin = strings.NewReader(`{"origin":"composer","session":"` + sessionID + `"}`)
			d.Stdout = &stdout
		})

	_, statErr := os.Stat(vault)

	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "no vault was created at "+vault)
	g.Expect(stdout.String()).To(BeEmpty())
	g.Expect(stderr).NotTo(ContainSubstring("created new vault"))
}

func Test793d4n1a_RunAutoSweep_TheRecordOfTheSessionIsIndexedAsTheOneExplicitSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	record := plantSessionRecord(t, sessions, "-home-joe-zorblax",
		`{"type":"user","message":{"role":"user","content":"hello"}}`)

	ingest, ingestCalls := MockIngestRun(t)
	done := make(chan error, 1)

	go func() {
		done <- cli.RunAutoSweep(context.Background(), cli.AutoSweepArgs{SessionsDir: sessions, ChunksDir: "/a/chunks"},
			cli.AutoSweepDeps{
				Stdin:   strings.NewReader(`{"origin":"composer","session":"` + sessionID + `"}`),
				Ingest:  ingest,
				ReadDir: os.ReadDir,
				Exists:  fileExists,
			})
	}()

	// the expectation runs apart, so that a call that does not match it, which this version of imptest leaves waiting,
	// cannot hold the test: the test waits for the outcome with a time limit
	go func() {
		ingestCalls.ArgsShould(match.BeAny, Equal(cli.IngestArgs{
			Transcripts: []string{record}, ChunksDir: "/a/chunks",
		}), match.BeAny).Return(nil)
	}()

	select {
	case err := <-done:
		g.Expect(err).NotTo(HaveOccurred())
	case <-time.After(mockTimeout):
		t.Fatal("the sweep did not finish: the index was not run for the record's path alone, as an explicit transcript")
	}
}

func Test793d4n1b_RunAutoSweep_ASessionWithNoRecordFailsTheCommand(t *testing.T) {
	t.Parallel()

	sessions := t.TempDir()
	plantSessionRecord(t, sessions, "a-project", `{"type":"user","message":{"role":"user","content":"hello"}}`)

	for name, session := range map[string]string{
		"a session that no record holds": "1f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11",
		"no session at all":              "",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			err := cli.RunAutoSweep(context.Background(), cli.AutoSweepArgs{SessionsDir: sessions},
				cli.AutoSweepDeps{
					Stdin:   strings.NewReader(`{"origin":"composer","session":"` + session + `"}`),
					Ingest:  indexNothing(t),
					ReadDir: os.ReadDir,
					Exists:  fileExists,
				})

			g.Expect(err).To(MatchError(ContainSubstring("no record of session " + session + " under " + sessions)))
		})
	}
}

func Test793d4n1c_RunAutoSweep_APromptOfAnOriginThatIsNotSweptIsSweptNothing(t *testing.T) {
	t.Parallel()

	sessions := t.TempDir()
	plantSessionRecord(t, sessions, "a-project", `{"type":"user","message":{"role":"user","content":"hello"}}`)

	for name, origin := range map[string]string{
		"a notification of a task":   "task-notification",
		"a peer":                     "peer",
		"no origin":                  "",
		"a headless prompt, no flag": "sdk",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			err := cli.RunAutoSweep(context.Background(), cli.AutoSweepArgs{SessionsDir: sessions},
				cli.AutoSweepDeps{
					Stdin:   strings.NewReader(`{"origin":"` + origin + `","session":"` + sessionID + `"}`),
					Ingest:  indexNothing(t),
					ReadDir: os.ReadDir,
					Exists:  fileExists,
				})

			g.Expect(err).NotTo(HaveOccurred())
		})
	}

	t.Run("a headless prompt, with the flag", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		indexed := 0

		err := cli.RunAutoSweep(context.Background(), cli.AutoSweepArgs{SessionsDir: sessions, CaptureSDK: true},
			cli.AutoSweepDeps{
				Stdin: strings.NewReader(`{"origin":"sdk","session":"` + sessionID + `"}`),
				Ingest: func(context.Context, cli.IngestArgs, io.Writer) error {
					indexed++

					return nil
				},
				ReadDir: os.ReadDir,
				Exists:  fileExists,
			})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(indexed).To(Equal(1), "the record was indexed")
	})
}

func Test793d4n1d_RunAutoSweep_AFailedIndexAndAnInputThatIsNotAJSONObjectFailTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	plantSessionRecord(t, sessions, "a-project", `{"type":"user","message":{"role":"user","content":"hello"}}`)

	errDisk := errors.New("the disk is full")

	failed := cli.RunAutoSweep(context.Background(), cli.AutoSweepArgs{SessionsDir: sessions},
		cli.AutoSweepDeps{
			Stdin: strings.NewReader(`{"origin":"composer","session":"` + sessionID + `"}`),
			Ingest: func(context.Context, cli.IngestArgs, io.Writer) error {
				return errDisk
			},
			ReadDir: os.ReadDir,
			Exists:  fileExists,
		})

	g.Expect(failed).To(MatchError(errDisk))
	g.Expect(failed).To(MatchError(ContainSubstring("auto sweep: indexing")))

	notJSON := cli.RunAutoSweep(context.Background(), cli.AutoSweepArgs{SessionsDir: sessions},
		cli.AutoSweepDeps{
			Stdin: strings.NewReader("not a JSON object"), Ingest: indexNothing(t), ReadDir: os.ReadDir, Exists: fileExists,
		})

	g.Expect(notJSON).To(MatchError(ContainSubstring("auto sweep: reading the turn")))
}

// fileExists reports whether a file is at the path.
func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// indexNothing is an index that fails the test when it is run.
func indexNothing(t *testing.T) cli.IngestRun {
	t.Helper()

	return func(context.Context, cli.IngestArgs, io.Writer) error {
		t.Error("the sweep indexed a record it should not have")

		return nil
	}
}

// plantSessionRecord writes the record of the session of the tests in a directory of the project, and returns its path.
func plantSessionRecord(t *testing.T, sessions, project, rows string) string {
	t.Helper()

	dir := filepath.Join(sessions, project)

	mkdirErr := os.MkdirAll(dir, 0o750)
	if mkdirErr != nil {
		t.Fatalf("creating %s: %v", dir, mkdirErr)
	}

	path := filepath.Join(dir, sessionID+".jsonl")

	writeErr := os.WriteFile(path, []byte(rows+"\n"), 0o600)
	if writeErr != nil {
		t.Fatalf("writing the record: %v", writeErr)
	}

	return path
}
