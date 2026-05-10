package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestParseLuhmannID_AlternatingSegments(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := cli.ExportParseLuhmannID("1a3b")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{"1", "a", "3", "b"}))
}

func TestParseLuhmannID_MultiCharSegments(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := cli.ExportParseLuhmannID("12ab3")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{"12", "ab", "3"}))
}

func TestParseLuhmannID_RejectsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportParseLuhmannID("")
	g.Expect(err).To(HaveOccurred())
}

func TestParseLuhmannID_RejectsLeadingLetter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportParseLuhmannID("a1")
	g.Expect(err).To(HaveOccurred())
}

func TestParseLuhmannID_TopLevelDigit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := cli.ExportParseLuhmannID("1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{"1"}))
}
