package result_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/result"
)

func TestRealFS_ReadFile_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	fs := result.RealFS{}
	_, err := fs.ReadFile(filepath.Join(dir, "nonexistent.toml"))
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_ReadFile_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	err := os.WriteFile(path, []byte("[status]\nsuccess = true\n"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	fs := result.RealFS{}
	data, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())
}

func TestRunCollect_AllSucceeded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	contextDir := filepath.Join(dir, result.ContextDir)
	err := os.MkdirAll(contextDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	resultContent := "[status]\nsuccess = true\n[outputs]\nfiles_modified = []\n"
	err = os.WriteFile(filepath.Join(contextDir, "TASK-001-tdd-red.result.toml"), []byte(resultContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = result.RunCollect(result.CollectArgs{
		Dir:   dir,
		Tasks: "TASK-001",
		Skill: "tdd-red",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCollect_DefaultSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	contextDir := filepath.Join(dir, result.ContextDir)
	err := os.MkdirAll(contextDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	resultContent := "[status]\nsuccess = true\n[outputs]\nfiles_modified = []\n"
	err = os.WriteFile(filepath.Join(contextDir, "TASK-001-tdd-red.result.toml"), []byte(resultContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = result.RunCollect(result.CollectArgs{
		Dir:   dir,
		Tasks: "TASK-001",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCollect_JSONFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	contextDir := filepath.Join(dir, result.ContextDir)
	err := os.MkdirAll(contextDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	resultContent := "[status]\nsuccess = true\n[outputs]\nfiles_modified = []\n"
	err = os.WriteFile(filepath.Join(contextDir, "TASK-001-tdd-red.result.toml"), []byte(resultContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = result.RunCollect(result.CollectArgs{
		Dir:    dir,
		Tasks:  "TASK-001",
		Skill:  "tdd-red",
		Format: "json",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunValidateCore_FileRequired(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := result.RunValidateCore(result.ValidateArgs{File: ""}, func(_ int) {})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("required"))
	}
}

func TestRunValidateCore_JSONParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	err := os.WriteFile(path, []byte("[invalid toml\n"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	var exitCode int

	err = result.RunValidateCore(result.ValidateArgs{File: path, Format: "json"}, func(code int) { exitCode = code })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunValidateCore_JSONReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	var exitCode int

	err := result.RunValidateCore(result.ValidateArgs{
		File:   filepath.Join(dir, "nonexistent.toml"),
		Format: "json",
	}, func(code int) { exitCode = code })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunValidateCore_JSONSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	content := "[status]\nsuccess = true\n[outputs]\nfiles_modified = []\n"
	path := filepath.Join(dir, "result.toml")
	err := os.WriteFile(path, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = result.RunValidateCore(result.ValidateArgs{File: path, Format: "json"}, func(_ int) {})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunValidateCore_TextParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	err := os.WriteFile(path, []byte("[invalid toml\n"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	var exitCode int

	err = result.RunValidateCore(result.ValidateArgs{File: path}, func(code int) { exitCode = code })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunValidate_NonExistentFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := result.RunValidate(result.ValidateArgs{
		File: filepath.Join(dir, "nonexistent.toml"),
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to read file"))
	}
}

func TestRunValidate_ValidFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	content := "[status]\nsuccess = true\n[outputs]\nfiles_modified = []\n"
	path := filepath.Join(dir, "result.toml")
	err := os.WriteFile(path, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = result.RunValidate(result.ValidateArgs{File: path})
	g.Expect(err).ToNot(HaveOccurred())
}
