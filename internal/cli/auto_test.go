package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
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

// zorblaxResetNote is the note `engram learn fact` writes for the staging database's reset schedule.
const zorblaxResetNote = "---\ntype: fact\ntier: L2\nsituation: testing against the Zorblax staging database\n" +
	"subject: the Zorblax staging database\npredicate: is reset every\nobject: Sunday at 03:40 UTC\n" +
	"luhmann: \"1\"\ncreated: \"2026-10-17\"\nsource: test\nuser: u\nvault: personal\n---\n\n" +
	"Information learned: when in testing against the Zorblax staging database, " +
	"the Zorblax staging database is reset every Sunday at 03:40 UTC.\n"
