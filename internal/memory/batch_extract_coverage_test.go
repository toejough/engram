package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestFormatBytes_Bytes verifies byte format for small values.
func TestFormatBytes_Bytes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatBytes(512)

	g.Expect(result).To(Equal("512B"))
}

// TestFormatBytes_Kilobytes verifies KB format.
func TestFormatBytes_Kilobytes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatBytes(2048)

	g.Expect(result).To(Equal("2.0KB"))
}

// TestFormatBytes_Megabytes verifies MB format.
func TestFormatBytes_Megabytes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatBytes(2 * 1024 * 1024)

	g.Expect(result).To(Equal("2.0MB"))
}

// TestFormatBytes_Zero verifies zero bytes format.
func TestFormatBytes_Zero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatBytes(0)

	g.Expect(result).To(Equal("0B"))
}

// TestParsePrinciplesJSON_CompleteJSON verifies clean JSON array parsing.
func TestParsePrinciplesJSON_CompleteJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := `[{"principle":"Always use TDD","evidence":"Tests first","category":"testing"}]`

	principles, err := parsePrinciplesJSON(raw)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(principles).ToNot(BeNil())
	g.Expect(principles).To(HaveLen(1))

	if len(principles) == 0 {
		t.Fatal("principles must be non-empty")
	}

	g.Expect(principles[0].Principle).To(Equal("Always use TDD"))
	g.Expect(principles[0].Category).To(Equal("testing"))
}

// TestParsePrinciplesJSON_EmptyArray verifies empty array parsing.
func TestParsePrinciplesJSON_EmptyArray(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	principles, err := parsePrinciplesJSON("[]")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(principles).To(BeEmpty())
}

// TestParsePrinciplesJSON_FullyUnrecoverable verifies error on no braces.
func TestParsePrinciplesJSON_FullyUnrecoverable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// No valid JSON objects at all
	raw := "[just garbage with no braces"

	_, err := parsePrinciplesJSON(raw)

	g.Expect(err).To(HaveOccurred())
}

// TestParsePrinciplesJSON_MultipleItems verifies multiple principles parsed.
func TestParsePrinciplesJSON_MultipleItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := `[
		{"principle":"Always use TDD","evidence":"test first","category":"testing"},
		{"principle":"Never push untested","evidence":"bugs escaped","category":"testing"}
	]`

	principles, err := parsePrinciplesJSON(raw)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(principles).To(HaveLen(2))
}

// TestParsePrinciplesJSON_TruncatedRecoverable verifies partial JSON recovery.
func TestParsePrinciplesJSON_TruncatedRecoverable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two complete objects but no closing ] — truncated at end
	raw := `[{"principle":"Use TDD","evidence":"tests first","category":"testing"},{"principle":"Never skip","evidence":"bad outcomes","category":"testing"}`

	principles, err := parsePrinciplesJSON(raw)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(principles).ToNot(BeEmpty())
}

// TestSortEvents_AlreadySorted verifies no reordering when events are already sorted.
func TestSortEvents_AlreadySorted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	events := []HaikuEvent{
		{ChunkIndex: 0, LineRange: "1-5"},
		{ChunkIndex: 0, LineRange: "6-10"},
		{ChunkIndex: 1, LineRange: "11-20"},
	}

	sortEvents(events)

	g.Expect(events[0].ChunkIndex).To(Equal(0))
	g.Expect(events[0].LineRange).To(Equal("1-5"))
	g.Expect(events[1].ChunkIndex).To(Equal(0))
	g.Expect(events[1].LineRange).To(Equal("6-10"))
	g.Expect(events[2].ChunkIndex).To(Equal(1))
}

// TestSortEvents_EmptySlice verifies sortEvents handles empty input.
func TestSortEvents_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	events := []HaikuEvent{}

	sortEvents(events)

	g.Expect(events).To(BeEmpty())
}

// TestSortEvents_ReverseOrder verifies events are sorted ascending by chunk index.
func TestSortEvents_ReverseOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	events := []HaikuEvent{
		{ChunkIndex: 2, LineRange: "50-60"},
		{ChunkIndex: 1, LineRange: "20-30"},
		{ChunkIndex: 0, LineRange: "1-10"},
	}

	sortEvents(events)

	g.Expect(events[0].ChunkIndex).To(Equal(0))
	g.Expect(events[1].ChunkIndex).To(Equal(1))
	g.Expect(events[2].ChunkIndex).To(Equal(2))
}

// TestSortEvents_SameChunkSortedByLineRange verifies stable sort within a chunk.
func TestSortEvents_SameChunkSortedByLineRange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	events := []HaikuEvent{
		{ChunkIndex: 0, LineRange: "20-30"},
		{ChunkIndex: 0, LineRange: "1-10"},
	}

	sortEvents(events)

	g.Expect(events[0].LineRange).To(Equal("1-10"))
	g.Expect(events[1].LineRange).To(Equal("20-30"))
}

// TestSortEvents_SingleElement verifies sortEvents handles single-element slice.
func TestSortEvents_SingleElement(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	events := []HaikuEvent{
		{ChunkIndex: 0, LineRange: "1-5"},
	}

	sortEvents(events)

	g.Expect(events).To(HaveLen(1))
}
