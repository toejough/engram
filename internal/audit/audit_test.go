package audit_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/audit"
)

func TestT36_EntryHasTimestampOperationAndAction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given an audit.Logger writing to a bytes.Buffer
	var buf bytes.Buffer

	log := audit.NewLogger(&buf)

	// When log.Log called with timestamp, operation, action, and fields
	timestamp := time.Date(2026, 2, 27, 16, 30, 0, 0, time.UTC)
	err := log.Log(audit.Entry{
		Timestamp: timestamp,
		Operation: "extract",
		Action:    "created",
		Fields:    map[string]string{"memory_id": "m_7f3a"},
	})

	// Then nil error, buffer contains timestamp, operation, and action
	g.Expect(err).NotTo(HaveOccurred())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("2026-02-27T16:30:00Z"))
	g.Expect(output).To(ContainSubstring("extract"))
	g.Expect(output).To(ContainSubstring("created"))
}

func TestT37_AppendOnlyNewEntriesDontModifyPrior(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given an audit.Logger writing to a bytes.Buffer
	var buf bytes.Buffer

	log := audit.NewLogger(&buf)
	timestamp := time.Date(2026, 2, 27, 16, 30, 0, 0, time.UTC)

	// When log.Log called with first entry
	err := log.Log(audit.Entry{
		Timestamp: timestamp,
		Operation: "extract",
		Action:    "created",
		Fields:    map[string]string{"memory_id": "m_7f3a"},
	})
	// Then buffer contains first entry text
	g.Expect(err).NotTo(HaveOccurred())

	first := buf.String()

	// When log.Log called with second entry
	err = log.Log(audit.Entry{
		Timestamp: timestamp.Add(time.Minute),
		Operation: "correct",
		Action:    "enriched",
		Fields:    map[string]string{"memory_id": "m_aa11"},
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Then buffer starts with first entry text (prefix preserved), contains exactly 2 lines
	output := buf.String()
	g.Expect(output).To(HavePrefix(first))
	lines := strings.Split(strings.TrimSpace(output), "\n")
	g.Expect(lines).To(HaveLen(2))
}

func TestT38_FormatMatchesDES7KeyValueSpec(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given an audit.Logger writing to a bytes.Buffer
	var buf bytes.Buffer

	log := audit.NewLogger(&buf)
	timestamp := time.Date(2026, 2, 27, 16, 30, 0, 0, time.UTC)

	// When log.Log called with timestamp, operation, action, and key-value fields
	err := log.Log(audit.Entry{
		Timestamp: timestamp,
		Operation: "extract",
		Action:    "created",
		Fields:    map[string]string{"memory_id": "m_7f3a", "confidence": "B"},
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Then format is: <RFC3339> <operation> <action> <key=value pairs>
	line := strings.TrimSpace(buf.String())
	parts := strings.Fields(line)
	g.Expect(parts[0]).To(Equal("2026-02-27T16:30:00Z"))
	g.Expect(parts[1]).To(Equal("extract"))
	g.Expect(parts[2]).To(Equal("created"))
	g.Expect(line).To(ContainSubstring(`memory_id="m_7f3a"`))
	g.Expect(line).To(ContainSubstring(`confidence="B"`))

	// When log.Log called with fields containing spaces
	buf.Reset()

	err = log.Log(audit.Entry{
		Timestamp: timestamp,
		Operation: "extract",
		Action:    "created",
		Fields:    map[string]string{"content": "Always check things carefully"},
	})
	g.Expect(err).NotTo(HaveOccurred())
	// Then values containing spaces are quoted
	g.Expect(buf.String()).To(ContainSubstring(`content="Always check things carefully"`))
}
