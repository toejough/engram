package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test793d4h1a_RunAutoWrite_AFeedbackLessonBecomesAFeedbackNoteMarkedAsWrittenAutomatically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal"},
		cli.AutoWriteDeps{
			Stdin: strings.NewReader(`{"session":"` + sessionID + `","reply":"[{\"kind\":\"feedback\",` +
				`\"slug\":\"format-before-commit\",\"situation\":\"committing a Python file\",\"covered\":null,` +
				`\"behavior\":\"left trailing whitespace\",\"impact\":\"the lint check failed\",` +
				`\"action\":\"run the formatter before committing\"}]"}`),
			Learn:       learnInto(written),
			Now:         func() time.Time { return recordedAt },
			WriteRecord: (&recordedFiles{}).write,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(written).To(HaveLen(1))

	for path, note := range written {
		g.Expect(path).To(HaveSuffix(".format-before-commit.md"))
		g.Expect(note).To(ContainSubstring("type: feedback"))
		g.Expect(note).To(ContainSubstring("situation: committing a Python file"))
		g.Expect(note).To(ContainSubstring("left trailing whitespace"))
		g.Expect(note).To(ContainSubstring("the lint check failed"))
		g.Expect(note).To(ContainSubstring("run the formatter before committing"))
		g.Expect(note).To(ContainSubstring("source: written automatically (session " + sessionID + ")"))
		g.Expect(out.String()).To(ContainSubstring(path))
	}
}

func Test793d4h1b_RunAutoWrite_AFactLessonBecomesAFactNoteMarkedAsWrittenAutomatically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal"},
		cli.AutoWriteDeps{
			Stdin: strings.NewReader(`{"session":"` + sessionID + `","reply":"[{\"kind\":\"fact\",` +
				`\"slug\":\"staging-reset\",\"situation\":\"testing on the staging database\",\"covered\":null,` +
				`\"subject\":\"the staging database\",\"predicate\":\"is reset every\",\"object\":\"Sunday night\"}]"}`),
			Learn:       learnInto(written),
			Now:         func() time.Time { return recordedAt },
			WriteRecord: (&recordedFiles{}).write,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(written).To(HaveLen(1))

	for path, note := range written {
		g.Expect(path).To(HaveSuffix(".staging-reset.md"))
		g.Expect(note).To(ContainSubstring("type: fact"))
		g.Expect(note).To(ContainSubstring("situation: testing on the staging database"))
		g.Expect(note).To(ContainSubstring("the staging database"))
		g.Expect(note).To(ContainSubstring("Sunday night"))
		g.Expect(note).To(ContainSubstring("source: written automatically (session " + sessionID + ")"))
	}
}

func Test793d4h1c_RunAutoWrite_AReplyThatWritesNothingCallsNoLearn(t *testing.T) {
	t.Parallel()

	for name, reply := range map[string]string{
		"not JSON":      `"I think there is one lesson here."`,
		"an empty list": `"[]"`,
		"a lesson that is not whole": `"[{\"kind\":\"fact\",\"slug\":\"\",\"situation\":\"s\",\"subject\":\"x\",` +
			`\"predicate\":\"y\",\"object\":\"z\"}]"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			written := map[string]string{}

			var out bytes.Buffer

			err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault"},
				cli.AutoWriteDeps{
					Stdin:       strings.NewReader(`{"session":"` + sessionID + `","reply":` + reply + `}`),
					Learn:       learnInto(written),
					Now:         func() time.Time { return recordedAt },
					WriteRecord: (&recordedFiles{}).write,
				}, &out)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(written).To(BeEmpty())
		})
	}
}

func Test793d4k1a_RunAutoWrite_ALessonThatIsWrittenIsRecordedWithItsNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}
	records := &recordedFiles{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
		cli.AutoWriteDeps{
			Stdin:       replyInput(feedbackLessonReply),
			Learn:       learnInto(written),
			Now:         func() time.Time { return recordedAt },
			WriteRecord: records.write,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(records.all()).To(HaveLen(1))

	for path, record := range records.all() {
		g.Expect(path).To(HavePrefix("/data/auto/" + sessionID + "/20261017T120000.000000123Z-"))
		g.Expect(path).To(HaveSuffix("-written.json"))
		g.Expect(record).To(HaveKeyWithValue("kind", "written"))
		g.Expect(record).To(HaveKeyWithValue("session", sessionID))
		g.Expect(record).To(HaveKeyWithValue("turn", turnID))
		g.Expect(record).To(HaveKeyWithValue("note", "1.2026-10-17.format-before-commit.md"))
		g.Expect(record).To(HaveKeyWithValue("slug", "format-before-commit"))
	}
}

func Test793d4k1b_RunAutoWrite_AReplyWithNoLessonIsRecordedAsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := &recordedFiles{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
		cli.AutoWriteDeps{
			Stdin:       replyInput("[]"),
			Learn:       learnInto(map[string]string{}),
			Now:         func() time.Time { return recordedAt },
			WriteRecord: records.write,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(records.kinds()).To(Equal([]string{"nothing"}))

	for _, record := range records.all() {
		g.Expect(record).To(HaveKeyWithValue("session", sessionID))
		g.Expect(record).To(HaveKeyWithValue("turn", turnID))
	}
}

func Test793d4k1c_RunAutoWrite_ALessonLeftOutAndAReplyThatIsNotAListAreRecordedWithTheReason(t *testing.T) {
	t.Parallel()

	t.Run("a lesson with no slug beside a lesson that is whole", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		written := map[string]string{}
		records := &recordedFiles{}

		reply := `[{"kind":"fact","slug":"","situation":"s","subject":"x","predicate":"y","object":"z"},` +
			`{"kind":"fact","slug":"staging-reset","situation":"testing on the staging database",` +
			`"subject":"the staging database","predicate":"is reset every","object":"Sunday night"}]`

		var out bytes.Buffer

		err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
			cli.AutoWriteDeps{
				Stdin:       replyInput(reply),
				Learn:       learnInto(written),
				Now:         func() time.Time { return recordedAt },
				WriteRecord: records.write,
			}, &out)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(written).To(HaveLen(1))

		kinds := records.kinds()
		slices.Sort(kinds)
		g.Expect(kinds).To(Equal([]string{"skipped", "written"}))

		for _, record := range records.all() {
			if record["kind"] == "skipped" {
				g.Expect(record["reason"]).To(ContainSubstring("slug"))
			}
		}
	})

	t.Run("a reply that is not a list", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		records := &recordedFiles{}

		var out bytes.Buffer

		err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
			cli.AutoWriteDeps{
				Stdin:       replyInput("no lessons, sorry"),
				Learn:       learnInto(map[string]string{}),
				Now:         func() time.Time { return recordedAt },
				WriteRecord: records.write,
			}, &out)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(records.kinds()).To(Equal([]string{"skipped"}))

		for _, record := range records.all() {
			g.Expect(record["reason"]).To(ContainSubstring("not a list of lessons"))
		}
	})
}

func Test793d4k1d_RunAutoWrite_AFailureIsRecorded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}
	records := &recordedFiles{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
		cli.AutoWriteDeps{
			Stdin: strings.NewReader(`{"session":"` + sessionID + `","turn":"` + turnID +
				`","failure":"the judgement exited with 1"}`),
			Learn:       learnInto(written),
			Now:         func() time.Time { return recordedAt },
			WriteRecord: records.write,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(written).To(BeEmpty())
	g.Expect(records.kinds()).To(Equal([]string{"failed"}))

	for _, record := range records.all() {
		g.Expect(record).To(HaveKeyWithValue("reason", "the judgement exited with 1"))
	}
}

func Test793d4k1e_RunAutoWrite_ARecordThatCannotBeWrittenDoesNotFailTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
		cli.AutoWriteDeps{
			Stdin: replyInput(feedbackLessonReply),
			Learn: learnInto(written),
			Now:   func() time.Time { return recordedAt },
			WriteRecord: func(string, []byte) error {
				return errors.New("the disk is full")
			},
		}, &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(written).To(HaveLen(1))
}

// unexported constants.
const (
	feedbackLessonReply = `[{"kind":"feedback","slug":"format-before-commit","situation":"committing a Python file",` +
		`"covered":null,"behavior":"left trailing whitespace","impact":"the lint check failed",` +
		`"action":"run the formatter before committing"}]`
	turnID = "9b0a6a8e-5b0e-4f0e-8a3a-1f3c7d1e2b44"
)

// unexported variables.
var (
	recordedAt = time.Date(2026, 10, 17, 12, 0, 0, 123, time.UTC)
)

// recordedFiles is the place of the records, in memory: the files written to it, by path.
type recordedFiles struct {
	mutex sync.Mutex
	files map[string][]byte
}

// all returns the records written, by path, as the objects their files hold.
func (r *recordedFiles) all() map[string]map[string]any {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	records := map[string]map[string]any{}

	for path, data := range r.files {
		var record map[string]any

		if json.Unmarshal(data, &record) == nil {
			records[path] = record
		}
	}

	return records
}

// kinds returns the kinds of the records written, in the order of their paths.
func (r *recordedFiles) kinds() []string {
	records := r.all()
	paths := make([]string, 0, len(records))

	for path := range records {
		paths = append(paths, path)
	}

	slices.Sort(paths)

	kinds := make([]string, 0, len(paths))

	for _, path := range paths {
		kind, _ := records[path]["kind"].(string)
		kinds = append(kinds, kind)
	}

	return kinds
}

// write is the write of one record.
func (r *recordedFiles) write(path string, data []byte) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.files == nil {
		r.files = map[string][]byte{}
	}

	r.files[path] = data

	return nil
}

// replyInput is the input of the writing for a reply of the model.
func replyInput(reply string) *strings.Reader {
	return strings.NewReader(`{"session":"` + sessionID + `","turn":"` + turnID + `","reply":` + quoted(reply) + `}`)
}

// quoted is a string as JSON writes it.
func quoted(text string) string {
	encoded, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// learnInto is the learn path over an in-memory vault: it writes each note into the map, by its path, and returns the
// path as `engram learn` prints it.
func learnInto(written map[string]string) cli.LearnRun {
	return func(ctx context.Context, args cli.LearnArgs) (string, error) {
		deps := cli.LearnDeps{
			DetectRepo:    func(context.Context) string { return "" },
			DetectUser:    func(context.Context) string { return "" },
			Now:           func() time.Time { return time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC) },
			Getenv:        func(string) string { return "" },
			ListIDs:       func(string) ([]string, error) { return nil, nil },
			ListBasenames: func(string) ([]string, error) { return nil, nil },
			Lock:          func(string) (func(), error) { return func() {}, nil },
			WriteNew: func(path string, data []byte) error {
				written[path] = string(data)

				return nil
			},
		}

		var printed bytes.Buffer

		err := cli.RunLearn(ctx, args, deps, &printed)

		return strings.TrimSpace(printed.String()), err
	}
}
