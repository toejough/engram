package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	"engram/internal/cli"
)

func TestRun_CatchupBadSession(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{
		"engram", "catchup",
		"--session", "/nonexistent/session.txt",
		"--data-dir", t.TempDir(),
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("read session"))
	}
}

func TestRun_CatchupMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{"engram", "catchup"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--session"))
	}
}

func TestRun_CatchupNoLLM(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")
	sessionFile := filepath.Join(t.TempDir(), "session.txt")
	err := os.WriteFile(sessionFile, []byte("test transcript"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = cli.Run([]string{
		"engram", "catchup",
		"--session", sessionFile,
		"--data-dir", dataDir,
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("LLM client not yet implemented"))
	}
}

func TestRun_CorrectMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{"engram", "correct"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--message"))
	}
}

func TestRun_CorrectNoMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "perfectly normal message",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_CorrectTwiceTriggersGate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	// First correction creates a memory.
	err := cli.Run([]string{
		"engram", "correct",
		"--message", "no, always use bun instead of npm for packages",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Second similar correction triggers overlap gate (noOpGate returns false, so creates new).
	err = cli.Run([]string{
		"engram", "correct",
		"--message", "no, remember to use bun for all package operations",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_CorrectWithMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "no, use bun instead of npm for this project",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_ExtractBadSession(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{
		"engram", "extract",
		"--session", "/nonexistent/session.txt",
		"--data-dir", t.TempDir(),
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("read session"))
	}
}

func TestRun_ExtractMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{"engram", "extract"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--session"))
	}
}

func TestRun_ExtractNoLLM(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")
	sessionFile := filepath.Join(t.TempDir(), "session.txt")
	err := os.WriteFile(sessionFile, []byte("test transcript"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = cli.Run([]string{
		"engram", "extract",
		"--session", sessionFile,
		"--data-dir", dataDir,
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("LLM client not yet implemented"))
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{"engram"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_SurfaceAfterCorrection(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	// First, insert a memory via correction.
	err := cli.Run([]string{
		"engram", "correct",
		"--message", "no, always use bun instead of npm",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Then surface it — the keyword "bun" should match.
	err = cli.Run([]string{
		"engram", "surface",
		"--hook", "user-prompt",
		"--message", "bun",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_SurfaceBadAuditLog(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	// Pre-create the audit.log path as a directory so OpenFile fails.
	err := os.MkdirAll(filepath.Join(dataDir, "audit.log"), 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = cli.Run([]string{
		"engram", "surface",
		"--hook", "session-start",
		"--project-dir", "test",
		"--data-dir", dataDir,
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("audit log"))
	}
}

func TestRun_SurfaceBadDataDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// /dev/null is a file, not a directory — MkdirAll will fail.
	err := cli.Run([]string{
		"engram", "surface",
		"--hook", "session-start",
		"--project-dir", "test",
		"--data-dir", "/dev/null/impossible",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("create data dir"))
	}
}

func TestRun_SurfaceEmptyStore(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	err := cli.Run([]string{
		"engram", "surface",
		"--hook", "session-start",
		"--project-dir", "test project",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Verify data dir was created with expected files.
	_, statErr := os.Stat(filepath.Join(dataDir, "engram.db"))
	g.Expect(statErr).NotTo(HaveOccurred())
}

func TestRun_SurfaceMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{"engram", "surface"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--hook"))
	}
}

func TestRun_SurfaceMissingQuery(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{
		"engram", "surface",
		"--hook", "session-start",
		"--data-dir", "/tmp/unused",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--query"))
	}
}

func TestRun_SurfacePreToolUse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	err := cli.Run([]string{
		"engram", "surface",
		"--hook", "pre-tool-use",
		"--tool-input", "some tool input",
		"--data-dir", dataDir,
	})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := cli.Run([]string{"engram", "bogus"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}
