package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test855d4h1a_Place_TheLessonsAndTheSeenNotesAreInTheRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantNoteOnDisk(t, vault, resetNoteName, zorblaxResetNote)
	plantNoteOnDisk(t, vault, backupNoteName, strings.NewReplacer("reset", "backed up", "1.", "2.").
		Replace(zorblaxResetNote))

	sessions := recordDelivering(t, resetNoteName)

	step := placed(t, vault, sessions, placeInput(twoLessons))

	g.Expect(step.Request.Model).To(Equal("sonnet"))
	g.Expect(step.Request.TimeoutMs).To(Equal(15000), "the row of the placement judgement")
	g.Expect(step.Request.System).To(And(ContainSubstring("continuation"), ContainSubstring("sibling"),
		ContainSubstring("top"), ContainSubstring("updates"), ContainSubstring("narrows"), ContainSubstring("refutes"),
		ContainSubstring(`"supersedes"`)), "the ordered test, the supersedes types and the reply shape")
	g.Expect(step.Request.Prompt).To(And(
		ContainSubstring("<lessons>\n- format-before-commit (feedback; committing a Python file)\n"+
			"- staging-reset (fact; testing on the staging database)\n</lessons>"),
		ContainSubstring("<notes>\n- "+resetNoteName+" (testing against the Zorblax staging database): "),
		Not(ContainSubstring(backupNoteName))), "the lessons and the delivered note only")
	g.Expect(step.Carry.Seen).To(Equal([]string{resetNoteName}), "the carry names the delivered note")
}

func Test855d4h1b_Place_ANoteWrittenAutomaticallyInTheSessionIsASeenNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantNoteOnDisk(t, vault, resetNoteName, strings.Replace(zorblaxResetNote, "source: test",
		"source: written automatically (session "+sessionID+")", 1))
	plantNoteOnDisk(t, vault, backupNoteName, strings.NewReplacer("reset", "backed up", "1.", "2.", "source: test",
		"source: written automatically (session another-session)").Replace(zorblaxResetNote))

	step := placed(t, vault, t.TempDir(), placeInput(twoLessons))

	g.Expect(step.Request.Prompt).To(ContainSubstring("- " + resetNoteName + " ("))
	g.Expect(step.Request.Prompt).NotTo(ContainSubstring(backupNoteName))
	g.Expect(step.Carry.Seen).To(Equal([]string{resetNoteName}))
}

func Test855d4h1c_Place_ASessionThatHasSeenNoNoteAndAReplyWithNoWholeLessonPrintNothing(t *testing.T) {
	t.Parallel()

	vault := t.TempDir()
	plantNoteOnDisk(t, vault, resetNoteName, zorblaxResetNote)

	t.Run("a session whose transcript names no note", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out bytes.Buffer

		err := cli.RunAutoPlace(context.Background(), cli.AutoPlaceArgs{VaultPath: vault, SessionsDir: recordOfAPrompt(t)},
			cli.AutoPlaceDeps{Stdin: placeInput(twoLessons), ReadDir: os.ReadDir, ReadFile: os.ReadFile}, &out)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out.String()).To(BeEmpty())
	})

	for name, reply := range map[string]string{
		"an empty list": `[]`,
		"a lesson that lacks a field": `[{"kind":"fact","slug":"staging-reset","situation":"s","subject":"x",` +
			`"predicate":"y"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			reads := 0

			var out bytes.Buffer

			err := cli.RunAutoPlace(context.Background(),
				cli.AutoPlaceArgs{VaultPath: vault, SessionsDir: recordDelivering(t, resetNoteName)},
				cli.AutoPlaceDeps{
					Stdin: placeInput(reply),
					ReadDir: func(path string) ([]fs.DirEntry, error) {
						reads++

						return os.ReadDir(path)
					},
					ReadFile: func(path string) ([]byte, error) {
						reads++

						return os.ReadFile(path)
					},
				}, &out)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out.String()).To(BeEmpty())
			g.Expect(reads).To(BeZero(), "nothing was read: there is no lesson to place")
		})
	}
}

func Test855d4h1e_Place_AReplyThatIsNotAListFailsTheCommand(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		stdin string
		cause string
	}{
		"a reply that is not JSON": {
			stdin: `{"session":"` + sessionID + `","turn":"a-turn","reply":"I think there is one lesson here.","carry":{}}`,
			cause: "the reply",
		},
		"an input that is not a JSON object": {stdin: "not a JSON object", cause: "reading the lessons"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out bytes.Buffer

			err := cli.RunAutoPlace(context.Background(),
				cli.AutoPlaceArgs{VaultPath: t.TempDir(), SessionsDir: t.TempDir()},
				cli.AutoPlaceDeps{Stdin: strings.NewReader(test.stdin), ReadDir: os.ReadDir, ReadFile: os.ReadFile}, &out)

			g.Expect(err).To(MatchError(And(ContainSubstring("auto place"), ContainSubstring(test.cause))))
			g.Expect(out.String()).To(BeEmpty())
		})
	}
}

func Test858b2a1a_Place_ALessonTheVaultAlreadyStatesIsNotPlaced(t *testing.T) {
	t.Parallel()

	vault := t.TempDir()
	plantNoteOnDisk(t, vault, resetNoteName, zorblaxResetNote)

	sessions := recordDelivering(t, resetNoteName)
	covered := func(reply string, count int) string {
		return strings.Replace(reply, `"covered":null`, `"covered":"`+resetNoteName+`"`, count)
	}

	t.Run("one of two lessons", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		step := placed(t, vault, sessions, placeInput(covered(twoLessons, 1)))

		g.Expect(step.Request.Prompt).To(ContainSubstring("<lessons>\n- staging-reset (fact; testing on the staging "+
			"database)\n</lessons>"), "the lesson the note covers is not asked about")
		g.Expect(step.Request.Prompt).NotTo(ContainSubstring("format-before-commit"))
	})

	t.Run("every lesson", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out bytes.Buffer

		err := cli.RunAutoPlace(context.Background(), cli.AutoPlaceArgs{VaultPath: vault, SessionsDir: sessions},
			cli.AutoPlaceDeps{Stdin: placeInput(covered(twoLessons, 2)), ReadDir: os.ReadDir, ReadFile: os.ReadFile}, &out)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out.String()).To(BeEmpty(), "there is no lesson to place")
	})
}

// unexported constants.
const (
	twoLessons = `[{"kind":"feedback","slug":"format-before-commit","situation":"committing a Python file",` +
		`"covered":null,"behavior":"left trailing whitespace","impact":"the lint check failed",` +
		`"action":"run the formatter before committing"},{"kind":"fact","slug":"staging-reset",` +
		`"situation":"testing on the staging database","covered":null,"subject":"the staging database",` +
		`"predicate":"is reset every","object":"Sunday night"}]`
)

// placedStep is the request the placement prints, with the fields of it that the tests read.
type placedStep struct {
	Request struct {
		Model     string `json:"model"`
		System    string `json:"system"`
		Prompt    string `json:"prompt"`
		TimeoutMs int    `json:"timeoutMs"`
	} `json:"request"`
	Carry struct {
		Seen []string `json:"seen"`
	} `json:"carry"`
}

// placeInput is the JSON object the memory mod hands to the placement: the session, the turn, the reply of the lessons
// judgement and its carry.
func placeInput(reply string) *strings.Reader {
	encoded, err := json.Marshal(map[string]any{
		"session": sessionID, "turn": "a-turn", "reply": reply, "carry": map[string]any{},
	})
	if err != nil {
		panic(err)
	}

	return strings.NewReader(string(encoded))
}

// placed runs the placement over a vault and a directory of sessions, and returns the step it printed.
func placed(t *testing.T, vault, sessions string, stdin *strings.Reader) placedStep {
	t.Helper()

	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoPlace(context.Background(), cli.AutoPlaceArgs{VaultPath: vault, SessionsDir: sessions},
		cli.AutoPlaceDeps{Stdin: stdin, ReadDir: os.ReadDir, ReadFile: os.ReadFile}, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(HavePrefix(`{"request":`), "the mod knows a request by its first characters")

	var step placedStep

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed(), "the output is one JSON object: %q", out.String())

	return step
}

// recordDelivering writes the record of a session in which a note arrived with a prompt, and returns the directory of
// the records.
func recordDelivering(tb testing.TB, note string) string {
	tb.Helper()

	dir := tb.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}` + "\n" +
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit","content":` +
		`["[engram memory] The lines below are notes.\n- ` + note + ` (testing against the Zorblax staging database): ` +
		`the database is reset"]}}` + "\n"

	err := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte(record), 0o600)
	if err != nil {
		tb.Fatalf("writing the record: %v", err)
	}

	return dir
}
