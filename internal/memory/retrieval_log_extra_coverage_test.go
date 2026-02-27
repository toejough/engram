//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestLogRetrieval_MkdirError verifies LogRetrieval returns error when directory cannot be created.
func TestLogRetrieval_MkdirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// /dev/null is a character device, not a directory, so MkdirAll beneath it fails
	entry := memory.RetrievalLogEntry{
		Hook:  "Stop",
		Query: "test query",
	}

	err := memory.LogRetrieval("/dev/null/subpath", entry)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to create memory directory"))
}
