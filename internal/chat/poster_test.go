package chat_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/chat"
)

func TestFilePoster_Post_AppendsValidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var written bytes.Buffer

	poster := &chat.FilePoster{
		FilePath: "/fake/chat.toml",
		Lock:     fakeLock,
		AppendFile: func(_ string, data []byte) error {
			written.Write(data)
			return nil
		},
		LineCount: func(_ string) (int, error) { return 42, nil },
	}

	cursor, err := poster.Post(chat.Message{
		From: "executor", To: "all", Thread: "test", Type: "info", Text: "hello",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(cursor).To(Equal(42))

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(written.Bytes(), &parsed)).To(Succeed())
	g.Expect(parsed.Message).To(HaveLen(1))
	g.Expect(parsed.Message[0].From).To(Equal("executor"))
	g.Expect(parsed.Message[0].TS).NotTo(BeZero())
}

func TestFilePoster_Post_GeneratesFreshTS(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	before := time.Now().UTC()

	var written bytes.Buffer

	poster := &chat.FilePoster{
		FilePath:   "/fake/chat.toml",
		Lock:       fakeLock,
		AppendFile: func(_ string, data []byte) error { written.Write(data); return nil },
		LineCount:  func(_ string) (int, error) { return 1, nil },
	}

	_, err := poster.Post(chat.Message{From: "x", To: "all", Thread: "t", Type: "info", Text: "y"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	after := time.Now().UTC()

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	g.Expect(toml.Unmarshal(written.Bytes(), &parsed)).To(Succeed())

	ts := parsed.Message[0].TS
	g.Expect(ts.After(before) || ts.Equal(before)).To(BeTrue())
	g.Expect(ts.Before(after) || ts.Equal(after)).To(BeTrue())
}

func TestFilePoster_Post_LockError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lockErr := errors.New("lock busy")
	poster := &chat.FilePoster{
		FilePath: "/fake/chat.toml",
		Lock:     func(_ string) (func() error, error) { return nil, lockErr },
		AppendFile: func(_ string, _ []byte) error {
			t.Fatal("AppendFile should not be called when lock fails")
			return nil
		},
		LineCount: func(_ string) (int, error) { return 0, nil },
	}

	_, err := poster.Post(chat.Message{From: "x", To: "all", Thread: "t", Type: "info", Text: "y"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, lockErr)).To(BeTrue())
}

func TestFilePoster_Post_TOMLGoldenFixture(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	fixedTS := time.Date(2026, 4, 5, 7, 0, 0, 0, time.UTC)

	var written bytes.Buffer

	poster := &chat.FilePoster{
		FilePath:   "/fake/chat.toml",
		Lock:       fakeLock,
		AppendFile: func(_ string, data []byte) error { written.Write(data); return nil },
		LineCount:  func(_ string) (int, error) { return 1, nil },
		NowFunc:    func() time.Time { return fixedTS },
	}

	_, err := poster.Post(chat.Message{
		From: "lead", To: "all", Thread: "lifecycle", Type: "info", Text: "hello",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	golden := "\n[[message]]\n" +
		"from = \"lead\"\n" +
		"to = \"all\"\n" +
		"thread = \"lifecycle\"\n" +
		"type = \"info\"\n" +
		"ts = 2026-04-05T07:00:00Z\n" +
		"text = \"\"\"\nhello\n\"\"\"\n"
	g.Expect(written.String()).To(Equal(golden))
}

func fakeLock(_ string) (func() error, error) {
	return func() error { return nil }, nil
}
