package maintain_test

import (
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// TestTOMLHistoryRecorder_ReadRecord_Stubbed verifies the stubbed ReadRecord.
func TestTOMLHistoryRecorder_ReadRecord_Stubbed(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	recorder := maintain.NewTOMLHistoryRecorder()
	record, err := recorder.ReadRecord("/any/path.toml")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(record).NotTo(gomega.BeNil())
}
