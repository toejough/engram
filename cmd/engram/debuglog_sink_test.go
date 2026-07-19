package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestOpenDebugSink_AppendsAcrossOpens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "debug.log")

	sink := openDebugSink(path)
	g.Expect(sink).NotTo(gomega.BeNil())

	if sink == nil {
		return
	}

	_, err := sink.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Re-open the same path: append mode must preserve the first line —
	// the tail -F contract debuglog documents.
	second := openDebugSink(path)
	g.Expect(second).NotTo(gomega.BeNil())

	if second == nil {
		return
	}

	_, err = second.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	contents, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(contents)).To(gomega.Equal("line one\nline two\n"))
}

func TestOpenDebugSink_EmptyPathReturnsNil(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	g.Expect(openDebugSink("")).To(gomega.BeNil())
}

func TestOpenDebugSink_UnopenablePathReturnsNil(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Parent is a regular file, so opening a child path fails -> nil sink
	// (the CLI must run without debug logging rather than fail).
	dir := t.TempDir()
	blocked := filepath.Join(dir, "isfile")
	g.Expect(os.WriteFile(blocked, []byte("x"), testFilePerm)).To(gomega.Succeed())

	g.Expect(openDebugSink(filepath.Join(blocked, "debug.log"))).To(gomega.BeNil())
}
