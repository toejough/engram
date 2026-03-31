package maintain_test

import (
	"errors"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
)

func TestReadProposals_FileNotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readFile := func(_ string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	got, err := maintain.ReadProposals("nonexistent.json", readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(BeEmpty())
}

func TestReadProposals_InvalidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readFile := func(_ string) ([]byte, error) {
		return []byte("not json"), nil
	}

	_, err := maintain.ReadProposals("bad.json", readFile)
	g.Expect(err).To(MatchError(ContainSubstring("unmarshalling proposals")))
}

func TestReadProposals_ReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readFile := func(_ string) ([]byte, error) {
		return nil, errors.New("disk failure")
	}

	_, err := maintain.ReadProposals("bad.json", readFile)
	g.Expect(err).To(MatchError(ContainSubstring("reading proposals")))
}

func TestWriteAndReadProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	proposals := []maintain.Proposal{
		{
			ID:        "p-001",
			Action:    maintain.ActionUpdate,
			Target:    "mem-abc",
			Field:     "situation",
			Value:     "revised situation text",
			Rationale: "original was too vague",
		},
		{
			ID:        "p-002",
			Action:    maintain.ActionMerge,
			Target:    "mem-def",
			Related:   []string{"mem-ghi", "mem-jkl"},
			Rationale: "these three memories overlap significantly",
		},
	}

	var captured []byte

	var capturedPath string

	writeFile := func(path string, data []byte, _ os.FileMode) error {
		capturedPath = path
		captured = data

		return nil
	}

	readFile := func(_ string) ([]byte, error) {
		return captured, nil
	}

	err := maintain.WriteProposals("pending.json", proposals, writeFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedPath).To(Equal("pending.json"))

	got, err := maintain.ReadProposals("pending.json", readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(HaveLen(2))
	g.Expect(got[0].ID).To(Equal("p-001"))
	g.Expect(got[0].Action).To(Equal(maintain.ActionUpdate))
	g.Expect(got[0].Target).To(Equal("mem-abc"))
	g.Expect(got[0].Field).To(Equal("situation"))
	g.Expect(got[0].Value).To(Equal("revised situation text"))
	g.Expect(got[0].Rationale).To(Equal("original was too vague"))
	g.Expect(got[1].ID).To(Equal("p-002"))
	g.Expect(got[1].Action).To(Equal(maintain.ActionMerge))
	g.Expect(got[1].Related).To(Equal([]string{"mem-ghi", "mem-jkl"}))
	g.Expect(got[1].Rationale).To(Equal("these three memories overlap significantly"))
}

func TestWriteProposals_WriteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writeFile := func(_ string, _ []byte, _ os.FileMode) error {
		return errors.New("permission denied")
	}

	proposals := []maintain.Proposal{
		{ID: "p-001", Action: maintain.ActionDelete, Target: "mem-abc", Rationale: "obsolete"},
	}

	err := maintain.WriteProposals("pending.json", proposals, writeFile)
	g.Expect(err).To(MatchError(ContainSubstring("writing proposals")))
}
