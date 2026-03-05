package transcript_test

import (
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/transcript"
)

// TestReadRecent_EmptyPath returns empty string.
func TestReadRecent_EmptyPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := transcript.New(fakeReadFile(nil, nil))

	result, err := reader.ReadRecent("", 2000)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

// TestReadRecent_SmallFile returns entire content.
func TestReadRecent_SmallFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := "short transcript"
	reader := transcript.New(fakeReadFile([]byte(content), nil))

	result, err := reader.ReadRecent("/some/path.txt", 2000)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal(content))
}

// T-5: ReadRecent reads recent transcript portion (~2000 tokens)
func TestT5_ReadRecentReadsTail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Create content with ~5000 chars (approx tokens)
	content := strings.Repeat("word ", 1000) // 5000 chars
	reader := transcript.New(fakeReadFile([]byte(content), nil))

	const maxTokens = 2000

	result, err := reader.ReadRecent("/some/transcript.txt", maxTokens)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Result should be approximately maxTokens characters (tail of file)
	g.Expect(len(result)).To(BeNumerically("<=", maxTokens+100))
	g.Expect(result).ToNot(BeEmpty())
}

// T-6: ReadRecent with missing file returns empty string (non-fatal)
func TestT6_ReadRecentMissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readErr := errors.New("open /nonexistent: no such file or directory")
	reader := transcript.New(fakeReadFile(nil, readErr))

	result, err := reader.ReadRecent("/nonexistent/path/transcript.txt", 2000)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

// fakeReadFile returns a FileReader that returns the given content and error.
func fakeReadFile(
	content []byte,
	err error,
) transcript.FileReader {
	return func(_ string) ([]byte, error) {
		return content, err
	}
}
