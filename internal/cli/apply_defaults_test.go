package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestApplyDataDirDefault_KeepsNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := "/custom/path"
	err := cli.ExportApplyDataDirDefault(&dir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dir).To(Equal("/custom/path"))
}

func TestApplyDataDirDefault_SetsDefaultWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := ""
	err := cli.ExportApplyDataDirDefault(&dir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dir).NotTo(BeEmpty())
}
