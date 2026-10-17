package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

func Test793d4a1a_RunAutoDeliver_PrintsTheNotesThatBearOnThePromptAsOneBlock(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault},
		cli.AutoDeliverDeps{
			Stdin: strings.NewReader(`{"text":"When is the Zorblax staging database reset?"}`),
			Query: newQueryDeps(memFS),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")

	g.Expect(lines[0]).To(HavePrefix("[engram memory]"))
	g.Expect(lines).To(ContainElement("- 1.2026-10-17.zorblax-staging-reset.md " +
		"(testing against the Zorblax staging database): " +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC."))
}

func Test793d4a1b_RunAutoDeliver_AFailedLookupFailsTheCommandAndPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	query := newQueryDeps(newInMemoryFS())
	query.Embedder = errorEmbedder{}

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir()},
		cli.AutoDeliverDeps{Stdin: strings.NewReader(`{"text":"anything"}`), Query: query}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("embedder down")))
	g.Expect(out.String()).To(BeEmpty())
}

// TestTargets_AutoDeliver reaches the command the way the mod does, as `engram auto deliver` with the prompt on
// standard input, so the wiring of the group, the flag and the dependencies is run.
func TestTargets_AutoDeliverPrintsTheNotesThatBearOnThePromptOnStandardInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantNoteOnDisk(t, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{"engram", "auto", "deliver", "--vault", vault}, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Stdin = strings.NewReader(`{"text":"When is the Zorblax staging database reset?"}`)
		d.Stdout = &stdout
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(ContainSubstring("- 1.2026-10-17.zorblax-staging-reset.md (testing against"))
}

// unexported constants.
const (
	zorblaxResetNote = "---\ntype: fact\ntier: L2\nsituation: testing against the Zorblax staging database\n" +
		"subject: the Zorblax staging database\npredicate: is reset every\nobject: Sunday at 03:40 UTC\n" +
		"luhmann: \"1\"\ncreated: \"2026-10-17\"\nsource: test\nuser: u\nvault: personal\n---\n\n" +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC.\n"
)

// plantNoteOnDisk writes a note and its embedding sidecar into a vault directory.
func plantNoteOnDisk(t *testing.T, vault, name, body string) {
	t.Helper()

	sidecar, err := embed.BuildSidecar(context.Background(), stubEmbedder{modelID: "m@4", dims: 4}, []byte(body))
	if err != nil {
		t.Fatalf("building the sidecar: %v", err)
	}

	for path, content := range map[string][]byte{
		filepath.Join(vault, name):                    []byte(body),
		filepath.Join(vault, embed.SidecarPath(name)): embed.MarshalSidecar(sidecar),
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
}
