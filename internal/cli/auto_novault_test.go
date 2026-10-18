package cli_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test855d4f1a_Targets_TheMomentPhrasesRecallAndPlacementCommandsCreateNoVault(t *testing.T) {
	t.Parallel()

	// the placement command, auto place, joins the commands of this table in 855e6d4
	phrasesInput := `{"session":"` + sessionID + `","cue":"prompt","reply":"{\"depth\":\"glance\"}",` +
		`"carry":{"cue":"prompt","window":"User: Prepare a test run."}}`

	for _, command := range []struct {
		name  string
		input io.Reader
	}{
		{name: "moment", input: momentInput("composer")},
		{name: "phrases", input: strings.NewReader(phrasesInput)},
		{name: "recall", input: recallInput("glance", "staging resets")},
	} {
		t.Run(command.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := filepath.Join(t.TempDir(), "no-such-vault")
			sessions, home := t.TempDir(), t.TempDir()

			args := []string{"engram", "auto", command.name}

			switch command.name {
			case "recall":
				args = append(args, "--vault", vault, "--chunks-dir", t.TempDir())
			case "moment":
				args = append(args, "--sessions", sessions)
			}

			var stdout bytes.Buffer

			stderr := executeForTestWithDeps(t, args, func(d *cli.Deps) {
				d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
				d.Getenv = func(key string) string {
					if key == "ENGRAM_VAULT_PATH" {
						return vault
					}

					return ""
				}
				d.UserHomeDir = func() (string, error) { return home, nil }
				d.Stdin = command.input
				d.Stdout = &stdout
			})

			_, statErr := os.Stat(vault)

			g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "no vault was created at "+vault)
			g.Expect(stderr).NotTo(ContainSubstring("created new vault"))
		})
	}
}
