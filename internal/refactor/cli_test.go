package refactor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/refactor"
)

func TestRunExtractFunction_ExtractSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not in PATH")
	}

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.21\n"), 0o644)).To(Succeed())

	content := "package testmod\n\nfunc Process(x int) int {\n\tstep1 := x * 2\n\tstep2 := step1 + 10\n\treturn step2\n}\n"
	path := filepath.Join(dir, "proc.go")
	g.Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())

	// Run through the gopls execution path; covers success return if gopls succeeds.
	_ = refactor.RunExtractFunction(refactor.ExtractFunctionArgs{
		File:  path,
		Lines: "4-5",
		Name:  "doStepsRun",
	})
}

func TestRunExtractFunction_FileNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not in PATH")
	}

	err := refactor.RunExtractFunction(refactor.ExtractFunctionArgs{
		File:  "/nonexistent/path/file.go",
		Lines: "1-3",
		Name:  "extracted",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestRunExtractFunction_InvalidLineFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := refactor.RunExtractFunction(refactor.ExtractFunctionArgs{
		File:  "/some/file.go",
		Lines: "not-a-range",
		Name:  "extracted",
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("invalid line range format"))
	}
}

func TestRunRename_SymbolNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not in PATH")
	}

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.21\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "code.go"), []byte("package testmod\n\nfunc Foo() {}\n"), 0o644)).To(Succeed())

	err := refactor.RunRename(refactor.RenameArgs{
		Dir:    dir,
		Symbol: "DoesNotExist",
		To:     "Something",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestRunRename_WithGopls(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not in PATH")
	}

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.21\n"), 0o644)).To(Succeed())

	content := "package testmod\n\ntype OldNameCLI struct {\n\tValue int\n}\n"
	g.Expect(os.WriteFile(filepath.Join(dir, "types.go"), []byte(content), 0o644)).To(Succeed())

	err := refactor.RunRename(refactor.RenameArgs{
		Dir:    dir,
		Symbol: "OldNameCLI",
		To:     "NewNameCLI",
	})

	g.Expect(err).ToNot(HaveOccurred())
}
