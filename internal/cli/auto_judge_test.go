package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test793d4g1a_RunAutoJudge_PrintsTheRequestForTheTurnTheRecordEndsWith(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	project := filepath.Join(sessions, "-home-joe-zorblax")
	g.Expect(os.MkdirAll(project, 0o750)).To(Succeed())

	record := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"How do I build the Zorblax project?"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Try make all."}]}}`,
		`{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}`,
	}, "\n") + "\n"
	g.Expect(os.WriteFile(filepath.Join(project, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    strings.NewReader(`{"origin":"composer","session":"` + sessionID + `","answer":"Understood."}`),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var request struct {
		Model  string `json:"model"`
		Effort string `json:"effort"`
		Prompt string `json:"prompt"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &request)).To(Succeed(), "the output is one JSON object: %q", out.String())
	g.Expect(request.Model).To(Equal("haiku"))
	g.Expect(request.Effort).To(Equal("low"))
	g.Expect(request.Prompt).To(ContainSubstring("User: How do I build the Zorblax project?\nAgent: Try make all.\n" +
		"--- the turn ---\nUser: No - build with make zorblax-build.\nAgent: Understood."))
	g.Expect(out.String()).To(HavePrefix(`{"model":`), "the mod knows a request by its first characters")
}

func Test793d4g1c_RunAutoJudge_ASessionWithNoRecordIsJudgedNothing(t *testing.T) {
	t.Parallel()

	for name, session := range map[string]string{"a session no record holds": sessionID, "no session": ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out bytes.Buffer

			err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: t.TempDir()},
				cli.AutoJudgeDeps{
					Stdin:    strings.NewReader(`{"origin":"composer","session":"` + session + `","answer":"Understood."}`),
					Query:    queryOver(newQueryDeps(newInMemoryFS())),
					ReadDir:  os.ReadDir,
					ReadFile: os.ReadFile,
				}, &out)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out.String()).To(BeEmpty())
		})
	}
}

func Test793d4g1d_RunAutoJudge_AnInputThatIsNotAJSONObjectFailsTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: t.TempDir()},
		cli.AutoJudgeDeps{
			Stdin: strings.NewReader("not a JSON object"), Query: queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir: os.ReadDir, ReadFile: os.ReadFile,
		}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("auto judge: reading the turn")))
	g.Expect(out.String()).To(BeEmpty())
}

func Test793d4g1h_RunAutoJudge_TheNearestNotesAreInTheRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: vault, SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    strings.NewReader(`{"origin":"composer","session":"` + sessionID + `","answer":"On Sundays."}`),
			Query:    queryOver(newQueryDeps(memFS)),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var request struct {
		Prompt string `json:"prompt"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &request)).To(Succeed())
	g.Expect(request.Prompt).To(ContainSubstring("\n<notes>\n- 1.2026-10-17.zorblax-staging-reset.md " +
		"(testing against the Zorblax staging database): " +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC.\n</notes>"))
}

func Test793d4g1i_RunAutoJudge_AFailedLookupFailsTheCommandAndPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	query := newQueryDeps(newInMemoryFS())
	query.Embedder = errorEmbedder{}

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    strings.NewReader(`{"origin":"composer","session":"` + sessionID + `","answer":"On Sundays."}`),
			Query:    queryOver(query),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("embedder down")))
	g.Expect(out.String()).To(BeEmpty())
}
