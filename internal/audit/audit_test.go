package audit_test

// Tests for ARCH-7: Audit Logging (pure implementation, no I/O mocks).
// Won't compile yet — RED phase.

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"engram/internal/audit"
	"github.com/onsi/gomega"
)

var testTime = time.Date(2026, 2, 27, 16, 30, 0, 0, time.UTC)

// T-36: Every audit entry contains a timestamp, operation, and action field.
func TestAuditLog_EntryHasTimestampOperationAction(t *testing.T) {
	g := gomega.NewWithT(t)
	var buf bytes.Buffer
	log := audit.NewLogger(&buf)

	err := log.Log(audit.Entry{
		Timestamp: testTime,
		Operation: "extract",
		Action:    "created",
		Fields:    map[string]string{"memory_id": "m_7f3a"},
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	line := buf.String()
	g.Expect(line).To(gomega.ContainSubstring("2026-02-27T16:30:00Z"))
	g.Expect(line).To(gomega.ContainSubstring("extract"))
	g.Expect(line).To(gomega.ContainSubstring("created"))
}

// T-37: Writing an entry does not modify any prior entries in the log file.
func TestAuditLog_AppendOnly(t *testing.T) {
	g := gomega.NewWithT(t)
	var buf bytes.Buffer
	log := audit.NewLogger(&buf)

	_ = log.Log(audit.Entry{
		Timestamp: testTime,
		Operation: "extract",
		Action:    "created",
		Fields:    map[string]string{"memory_id": "m_0001"},
	})
	first := buf.String()

	_ = log.Log(audit.Entry{
		Timestamp: testTime.Add(time.Second),
		Operation: "correct",
		Action:    "enriched",
		Fields:    map[string]string{"memory_id": "m_0002"},
	})
	combined := buf.String()

	g.Expect(combined).To(gomega.HavePrefix(first))

	lines := strings.Split(strings.TrimSpace(combined), "\n")
	g.Expect(lines).To(gomega.HaveLen(2))
}

// T-38: Output line matches DES-7 key-value format.
func TestAuditLog_FormatMatchesDES7(t *testing.T) {
	g := gomega.NewWithT(t)
	var buf bytes.Buffer
	log := audit.NewLogger(&buf)

	_ = log.Log(audit.Entry{
		Timestamp: testTime,
		Operation: "extract",
		Action:    "created",
		Fields: map[string]string{
			"memory_id":  "m_7f3a",
			"confidence": "B",
		},
	})

	line := strings.TrimSpace(buf.String())

	// Must start with RFC3339 timestamp
	g.Expect(line).To(gomega.HavePrefix("2026-02-27T16:30:00Z"))

	// Must contain operation and action after timestamp
	parts := strings.SplitN(line, " ", 4)
	g.Expect(parts).To(gomega.HaveLen(4))
	g.Expect(parts[1]).To(gomega.Equal("extract"))
	g.Expect(parts[2]).To(gomega.Equal("created"))

	// Must contain key=value fields
	g.Expect(line).To(gomega.ContainSubstring("memory_id=m_7f3a"))
	g.Expect(line).To(gomega.ContainSubstring("confidence=B"))

	// Values with spaces must be quoted
	buf.Reset()
	_ = log.Log(audit.Entry{
		Timestamp: testTime,
		Operation: "extract",
		Action:    "rejected",
		Fields:    map[string]string{"content": "Always check things carefully"},
	})
	line2 := buf.String()
	g.Expect(line2).To(gomega.ContainSubstring(`content="Always check things carefully"`))
}
