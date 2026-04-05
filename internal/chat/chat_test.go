package chat_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/chat"
)

func TestMessage_TOMLRoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := chat.Message{
		From:   "planner-17",
		To:     "all",
		Thread: "impl-phase1",
		Type:   "info",
		TS:     time.Date(2026, 4, 5, 7, 0, 0, 0, time.UTC),
		Text:   "hello\nworld",
	}

	var buf bytes.Buffer

	encErr := toml.NewEncoder(&buf).Encode(struct {
		Message []chat.Message `toml:"message"`
	}{Message: []chat.Message{original}})
	g.Expect(encErr).NotTo(HaveOccurred())

	if encErr != nil {
		return
	}

	var parsed struct {
		Message []chat.Message `toml:"message"`
	}

	decErr := toml.Unmarshal(buf.Bytes(), &parsed)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(parsed.Message).To(HaveLen(1))
	g.Expect(parsed.Message[0]).To(Equal(original))
}
